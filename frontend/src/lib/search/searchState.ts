import type { ApiClient } from '../api/client';
import type { AppError, MacroToggles, SearchMode, SearchRequest, SearchResponse, TagFilter } from '../api/types';
import type { LocalStorageManager } from '../storage/localStorageManager';

export type SearchStatus = 'idle' | 'debouncing' | 'loading' | 'success' | 'empty' | 'error';

export interface SearchState {
  query: string;
  mode: SearchMode;
  filters: TagFilter[];
  page: number;
  selectedIndex: number;
  enabledMacros: MacroToggles;
  isOnline: boolean;
  isLoading: boolean;
  status: SearchStatus;
  response?: SearchResponse;
  error?: AppError;
}

export interface SearchControllerOptions {
  api: Pick<ApiClient, 'search'>;
  localStorageManager?: Pick<LocalStorageManager, 'readQuery' | 'writeQuery'>;
  retryManager?: { run: <T>(operation: () => Promise<T>) => Promise<T>; cancel?: () => unknown };
  debounceMs?: number;
  setTimeoutFn?: (callback: () => void, delay: number) => unknown;
  clearTimeoutFn?: (timer: unknown) => void;
}

export function createDefaultSearchState(): SearchState {
  return {
    query: '',
    mode: 'single',
    filters: [],
    page: 1,
    selectedIndex: -1,
    enabledMacros: { protein: true, carbs: true, fat: true },
    isOnline: true,
    isLoading: false,
    status: 'idle'
  };
}

export function buildSearchRequest(state: SearchState): SearchRequest {
  return {
    query: state.query.trim(),
    mode: state.mode,
    page: state.page,
    filters: state.filters,
    enabledMacros: state.enabledMacros
  };
}

export function createSearchController(options: SearchControllerOptions) {
  const debounceMs = options.debounceMs ?? 150;
  const setTimer = options.setTimeoutFn ?? ((callback: () => void, delay: number) => setTimeout(callback, delay));
  const clearTimer = options.clearTimeoutFn ?? ((value: unknown) => clearTimeout(value as ReturnType<typeof setTimeout>));
  let timer: unknown;
  let state = createDefaultSearchState();
  const listeners = new Set<(state: SearchState) => void>();

  function subscribe(listener: (state: SearchState) => void): () => void {
    listeners.add(listener);
    listener(state);
    return () => listeners.delete(listener);
  }

  function emit(next: Partial<SearchState>): SearchState {
    state = { ...state, ...next };
    for (const listener of listeners) {
      listener(state);
    }
    return state;
  }

  function setQuery(query: string): void {
    if (timer) {
      clearTimer(timer);
    }
    emit({ query, status: 'debouncing', error: undefined });
    timer = setTimer(() => {
      void execute();
    }, debounceMs);
  }

  function setMode(mode: SearchMode): void {
    emit({ mode, page: 1, selectedIndex: -1 });
  }

  function setMacro(key: keyof MacroToggles, enabled: boolean): void {
    emit({ enabledMacros: { ...state.enabledMacros, [key]: enabled }, page: 1 });
  }

  function setFilters(filters: TagFilter[]): void {
    emit({ filters, page: 1 });
  }

  function setPage(page: number): void {
    emit({ page: Math.max(1, page) });
    void execute();
  }

  function setOnline(isOnline: boolean): void {
    emit({ isOnline });
  }

  async function execute(): Promise<SearchState> {
    options.retryManager?.cancel?.();
    const request = buildSearchRequest(state);
    if (!state.isOnline) {
      const cached = options.localStorageManager?.readQuery(request);
      if (cached) {
        return emit({
          response: cached.response,
          isLoading: false,
          status: cached.response.items.length === 0 ? 'empty' : 'success',
          error: undefined
        });
      }
      return emit({ isLoading: false, status: 'error', error: { category: 'network', code: 'offline', message: 'Offline', retryable: true } });
    }
    if (request.query === '' && request.mode !== 'diet') {
      return emit({ status: 'idle', isLoading: false, response: undefined, error: undefined });
    }

    emit({ isLoading: true, status: 'loading', error: undefined });
    try {
      const response = await (options.retryManager?.run(() => options.api.search(request)) ?? options.api.search(request));
      options.localStorageManager?.writeQuery(request, response);
      return emit({
        response,
        isLoading: false,
        status: response.items.length === 0 ? 'empty' : 'success'
      });
    } catch (error) {
      return emit({
        isLoading: false,
        status: 'error',
        error: normalizeAppError(error)
      });
    }
  }

  function getState(): SearchState {
    return state;
  }

  return { subscribe, getState, setQuery, setMode, setMacro, setFilters, setPage, setOnline, execute };
}

function normalizeAppError(error: unknown): AppError {
  if (error && typeof error === 'object' && 'code' in error && 'category' in error) {
    return error as AppError;
  }
  return {
    category: 'unknown',
    code: 'unknown_error',
    message: 'Something went wrong',
    retryable: false,
    cause: error
  };
}
