import type { AppError } from '../api/types';
import { classifyClientError, defaultRetryPolicy, nextRetryDelay, shouldRetry, type RetryPolicy } from './errorHandling';

export interface RetryManagerState {
  status: 'idle' | 'waiting' | 'running' | 'cancelled' | 'failed' | 'succeeded';
  attempt: number;
  nextDelayMs?: number;
  error?: AppError;
}

export interface RetryManagerOptions {
  policy?: RetryPolicy;
  setTimeoutFn?: (callback: () => void, delay: number) => unknown;
  clearTimeoutFn?: (timer: unknown) => void;
}

export class RetryCancelledError extends Error {
  constructor() {
    super('Retry cancelled');
    this.name = 'RetryCancelledError';
  }
}

export function createRetryManager(options: RetryManagerOptions = {}) {
  const policy = options.policy ?? defaultRetryPolicy;
  const setTimer = options.setTimeoutFn ?? ((callback: () => void, delay: number) => setTimeout(callback, delay));
  const clearTimer = options.clearTimeoutFn ?? ((timer: unknown) => clearTimeout(timer as ReturnType<typeof setTimeout>));
  const listeners = new Set<(state: RetryManagerState) => void>();
  let timer: unknown;
  let runId = 0;
  let state: RetryManagerState = { status: 'idle', attempt: 0 };

  function subscribe(listener: (state: RetryManagerState) => void): () => void {
    listeners.add(listener);
    listener(state);
    return () => listeners.delete(listener);
  }

  function emit(next: Partial<RetryManagerState>): RetryManagerState {
    state = { ...state, ...next };
    for (const listener of listeners) {
      listener(state);
    }
    return state;
  }

  async function run<T>(operation: () => Promise<T>): Promise<T> {
    const currentRun = ++runId;
    clearPendingTimer();
    emit({ status: 'running', attempt: 0, nextDelayMs: undefined, error: undefined });

    for (let attempt = 0; attempt < policy.maxAttempts; attempt += 1) {
      if (currentRun !== runId) {
        throw new RetryCancelledError();
      }
      try {
        const value = await operation();
        emit({ status: 'succeeded', attempt, nextDelayMs: undefined, error: undefined });
        return value;
      } catch (error) {
        const appError = classifyClientError(error);
        if (!shouldRetry(appError, policy, attempt + 1)) {
          emit({ status: 'failed', attempt: attempt + 1, nextDelayMs: undefined, error: appError });
          throw error;
        }
        const delay = nextRetryDelay(policy, attempt + 1);
        emit({ status: 'waiting', attempt: attempt + 1, nextDelayMs: delay, error: appError });
        await wait(delay, currentRun);
        emit({ status: 'running', attempt: attempt + 1, nextDelayMs: undefined, error: appError });
      }
    }

    emit({ status: 'failed', attempt: policy.maxAttempts, nextDelayMs: undefined });
    throw new Error('Retry attempts exhausted');
  }

  function cancel(): RetryManagerState {
    runId += 1;
    clearPendingTimer();
    return emit({ status: 'cancelled', nextDelayMs: undefined });
  }

  function wait(delay: number, currentRun: number): Promise<void> {
    return new Promise((resolve, reject) => {
      timer = setTimer(() => {
        timer = undefined;
        if (currentRun !== runId) {
          reject(new RetryCancelledError());
          return;
        }
        resolve();
      }, delay);
    });
  }

  function clearPendingTimer(): void {
    if (timer) {
      clearTimer(timer);
      timer = undefined;
    }
  }

  function getState(): RetryManagerState {
    return state;
  }

  return { subscribe, getState, run, cancel };
}
