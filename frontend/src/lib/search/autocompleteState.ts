import type { ApiClient } from '../api/client';
import type { AppError, RankedAutocomplete } from '../api/types';

export interface AutocompleteState {
  query: string;
  options: RankedAutocomplete[];
  selectedIndex: number;
  isOpen: boolean;
  isLoading: boolean;
  error?: AppError;
}

export interface AutocompleteControllerOptions {
  api: Pick<ApiClient, 'autocomplete'>;
  limit?: number;
  debounceMs?: number;
  setTimeoutFn?: (callback: () => void, delay: number) => unknown;
  clearTimeoutFn?: (timer: unknown) => void;
  onSelect?: (option: RankedAutocomplete) => void;
}

export function createAutocompleteState(): AutocompleteState {
  return { query: '', options: [], selectedIndex: -1, isOpen: false, isLoading: false };
}

export function createAutocompleteController(options: AutocompleteControllerOptions) {
  const debounceMs = options.debounceMs ?? 150;
  const limit = options.limit ?? 10;
  const setTimer = options.setTimeoutFn ?? ((callback: () => void, delay: number) => setTimeout(callback, delay));
  const clearTimer = options.clearTimeoutFn ?? ((value: unknown) => clearTimeout(value as ReturnType<typeof setTimeout>));
  let timer: unknown;
  let state = createAutocompleteState();
  const listeners = new Set<(state: AutocompleteState) => void>();

  function subscribe(listener: (state: AutocompleteState) => void): () => void {
    listeners.add(listener);
    listener(state);
    return () => listeners.delete(listener);
  }

  function emit(next: Partial<AutocompleteState>): AutocompleteState {
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
    emit({ query, error: undefined });
    if (query.trim() === '') {
      emit({ options: [], selectedIndex: -1, isOpen: false, isLoading: false });
      return;
    }
    timer = setTimer(() => {
      void fetchOptions();
    }, debounceMs);
  }

  async function fetchOptions(): Promise<AutocompleteState> {
    const query = state.query.trim();
    if (query === '') {
      return emit({ options: [], selectedIndex: -1, isOpen: false, isLoading: false });
    }
    emit({ isLoading: true, isOpen: true, error: undefined });
    try {
      const options = await optionsApi(query);
      return emit({ options, selectedIndex: options.length > 0 ? 0 : -1, isOpen: options.length > 0, isLoading: false });
    } catch (error) {
      return emit({ options: [], selectedIndex: -1, isOpen: true, isLoading: false, error: normalizeAppError(error) });
    }
  }

  function handleKey(key: string, shiftKey = false): boolean {
    if (!state.isOpen && key !== 'Escape') {
      return false;
    }
    if (key === 'ArrowDown' || (key === 'Tab' && !shiftKey)) {
      moveSelection(1);
      return true;
    }
    if (key === 'ArrowUp' || (key === 'Tab' && shiftKey)) {
      moveSelection(-1);
      return true;
    }
    if (key === 'Enter') {
      const option = state.options[state.selectedIndex];
      if (option) {
        options.onSelect?.(option);
        emit({ isOpen: false, query: option.label });
        return true;
      }
    }
    if (key === 'Escape') {
      emit({ isOpen: false, selectedIndex: -1 });
      return true;
    }
    return false;
  }

  function blur(): void {
    emit({ isOpen: false, selectedIndex: -1 });
  }

  function hover(index: number): void {
    if (index >= 0 && index < state.options.length) {
      emit({ selectedIndex: index });
    }
  }

  function select(index: number): void {
    const option = state.options[index];
    if (!option) {
      return;
    }
    options.onSelect?.(option);
    emit({ isOpen: false, selectedIndex: index, query: option.label });
  }

  function getState(): AutocompleteState {
    return state;
  }

  async function optionsApi(query: string) {
    return options.api.autocomplete(query, limit);
  }

  function moveSelection(delta: number): void {
    if (state.options.length === 0) {
      return;
    }
    const next = (state.selectedIndex + delta + state.options.length) % state.options.length;
    emit({ selectedIndex: next });
  }

  return { subscribe, getState, setQuery, fetchOptions, handleKey, blur, hover, select };
}

export function highlightParts(label: string, query: string): Array<{ text: string; highlighted: boolean }> {
  const normalizedLabel = label.toLowerCase();
  const normalizedQuery = query.trim().toLowerCase();
  const start = normalizedQuery === '' ? -1 : normalizedLabel.indexOf(normalizedQuery);
  if (start < 0) {
    return [{ text: label, highlighted: false }];
  }
  return [
    { text: label.slice(0, start), highlighted: false },
    { text: label.slice(start, start + normalizedQuery.length), highlighted: true },
    { text: label.slice(start + normalizedQuery.length), highlighted: false }
  ].filter((part) => part.text.length > 0);
}

function normalizeAppError(error: unknown): AppError {
  if (error && typeof error === 'object' && 'code' in error && 'category' in error) {
    return error as AppError;
  }
  return { category: 'unknown', code: 'autocomplete_error', message: 'Autocomplete unavailable', retryable: true, cause: error };
}
