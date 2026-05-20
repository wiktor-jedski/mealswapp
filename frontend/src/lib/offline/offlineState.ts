export type OfflineStatus = 'online' | 'offline' | 'reconnecting';

export interface OfflineState {
  status: OfflineStatus;
  isOnline: boolean;
  queuedRetries: number;
  blockedMutations: string[];
  lastChangedAt?: string;
  message?: string;
}

export interface ConnectivityTarget {
  onLine?: boolean;
  addEventListener?: (type: 'online' | 'offline', listener: () => void) => void;
  removeEventListener?: (type: 'online' | 'offline', listener: () => void) => void;
}

export interface OfflineControllerOptions {
  target?: ConnectivityTarget;
  now?: () => Date;
}

export function createDefaultOfflineState(isOnline = true, now = new Date()): OfflineState {
  return {
    status: isOnline ? 'online' : 'offline',
    isOnline,
    queuedRetries: 0,
    blockedMutations: [],
    lastChangedAt: now.toISOString(),
    message: isOnline ? undefined : offlineMessage(0)
  };
}

export function createOfflineController(options: OfflineControllerOptions = {}) {
  const target = options.target ?? defaultConnectivityTarget();
  const now = options.now ?? (() => new Date());
  let retry: (() => void | Promise<void>) | undefined;
  let disposed = false;
  let state = createDefaultOfflineState(target?.onLine ?? true, now());
  const listeners = new Set<(state: OfflineState) => void>();

  function subscribe(listener: (state: OfflineState) => void): () => void {
    listeners.add(listener);
    listener(state);
    return () => listeners.delete(listener);
  }

  function emit(next: Partial<OfflineState>): OfflineState {
    state = { ...state, ...next, lastChangedAt: now().toISOString() };
    for (const listener of listeners) {
      listener(state);
    }
    return state;
  }

  function queueRetry(callback: () => void | Promise<void>): OfflineState {
    retry = callback;
    if (state.isOnline) {
      return state;
    }
    return emit({
      queuedRetries: 1,
      message: offlineMessage(1)
    });
  }

  function blockMutation(label: string): OfflineState {
    if (state.isOnline) {
      return state;
    }
    const blockedMutations = state.blockedMutations.includes(label)
      ? state.blockedMutations
      : [...state.blockedMutations, label];
    return emit({
      blockedMutations,
      message: `${label} needs a connection. Your current search state is preserved.`
    });
  }

  function setOnline(isOnline: boolean): OfflineState {
    if (!isOnline) {
      return emit({
        status: 'offline',
        isOnline: false,
        message: offlineMessage(state.queuedRetries)
      });
    }

    const queued = retry;
    retry = undefined;
    emit({
      status: queued ? 'reconnecting' : 'online',
      isOnline: true,
      blockedMutations: [],
      queuedRetries: 0,
      message: queued ? 'Connection restored. Retrying now.' : undefined
    });
    if (queued) {
      void Promise.resolve(queued()).finally(() => {
        if (!disposed && state.isOnline) {
          emit({ status: 'online', message: undefined });
        }
      });
    }
    return state;
  }

  function attach(): () => void {
    const handleOnline = () => setOnline(true);
    const handleOffline = () => setOnline(false);
    target?.addEventListener?.('online', handleOnline);
    target?.addEventListener?.('offline', handleOffline);
    return () => {
      disposed = true;
      target?.removeEventListener?.('online', handleOnline);
      target?.removeEventListener?.('offline', handleOffline);
    };
  }

  function getState(): OfflineState {
    return state;
  }

  return { subscribe, getState, attach, setOnline, queueRetry, blockMutation };
}

function defaultConnectivityTarget(): ConnectivityTarget | undefined {
  if (typeof globalThis.window !== 'undefined') {
    return globalThis.window;
  }
  return undefined;
}

function offlineMessage(queuedRetries: number): string {
  if (queuedRetries > 0) {
    return 'You are offline. Cached results stay visible and the current search will retry when the connection returns.';
  }
  return 'You are offline. Cached results stay visible while the connection is unavailable.';
}
