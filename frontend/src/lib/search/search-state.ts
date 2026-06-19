import { writable, type Writable } from "svelte/store";
import type { AppError, SearchFilter, SearchMode, SearchRequest, SubstitutionInput } from "../api/generated";

// Implements DESIGN-001 SearchView typed client state.
export interface SearchState {
  mode: SearchMode;
  query: string;
  page: number;
  filters: SearchFilter[];
  substitutionInputs: SubstitutionInput[];
  dailyDietId?: string;
  enabledMacros: { protein: boolean; carbohydrate: boolean; fat: boolean };
  loading: boolean;
  error: AppError | null;
}

// Implements DESIGN-001 SearchView safe initial state.
export function initialSearchState(): SearchState {
  return {
    mode: "catalog",
    query: "",
    page: 1,
    filters: [],
    substitutionInputs: [],
    enabledMacros: { protein: true, carbohydrate: true, fat: true },
    loading: false,
    error: null
  };
}

// Implements DESIGN-001 SearchView mode transition behavior.
export function stateForMode(state: SearchState, mode: SearchMode): SearchState {
  if (state.mode === mode) return state;
  return {
    ...state,
    mode,
    page: 1,
    substitutionInputs: mode === "substitution" ? state.substitutionInputs : [],
    dailyDietId: mode === "daily_diet_alternative" ? state.dailyDietId : undefined,
    loading: false,
    error: null
  };
}

// Implements DESIGN-001 SearchView deterministic Substitution Input accumulation.
export function addSubstitutionInput(state: SearchState, input: SubstitutionInput): SearchState {
  const existing = state.substitutionInputs.findIndex((candidate) => candidate.foodObjectId === input.foodObjectId);
  const substitutionInputs = [...state.substitutionInputs];
  if (existing >= 0) substitutionInputs[existing] = input;
  else substitutionInputs.push(input);
  return { ...state, substitutionInputs, page: 1 };
}

// Implements DESIGN-001 SearchView deterministic Substitution Input removal.
export function removeSubstitutionInput(state: SearchState, foodObjectId: string): SearchState {
  return { ...state, substitutionInputs: state.substitutionInputs.filter((input) => input.foodObjectId !== foodObjectId), page: 1 };
}

// Implements DESIGN-001 SearchView deterministic filter accumulation and removal.
export function addSearchFilter(state: SearchState, filter: SearchFilter): SearchState {
  const filters = state.filters.filter((value) => !(value.kind === filter.kind && value.filterId === filter.filterId));
  return { ...state, filters: [...filters, filter], page: 1 };
}

export function removeSearchFilter(state: SearchState, filter: SearchFilter): SearchState {
  return { ...state, filters: state.filters.filter((value) => !(value.kind === filter.kind && value.filterId === filter.filterId)), page: 1 };
}

// Implements DESIGN-001 SearchView generated-contract request construction.
export function buildSearchRequest(state: SearchState): SearchRequest {
  const request: SearchRequest = {
    query: state.query.trim(),
    mode: state.mode,
    filters: [...state.filters],
    page: state.page
  };
  if (state.mode === "substitution") request.substitutionInputs = [...state.substitutionInputs];
  if (state.mode === "daily_diet_alternative" && state.dailyDietId) request.dailyDietId = state.dailyDietId;
  return request;
}

// Implements DESIGN-001 SearchView stable TanStack/local-cache request identity.
export function searchRequestKey(request: SearchRequest): string {
  const filters = [...(request.filters ?? [])].sort((a, b) =>
    `${a.kind}:${a.filterId}:${a.include}`.localeCompare(`${b.kind}:${b.filterId}:${b.include}`)
  );
  const substitutionInputs = [...(request.substitutionInputs ?? [])].sort((a, b) =>
    `${a.foodObjectId}:${a.unit}:${a.quantity}`.localeCompare(`${b.foodObjectId}:${b.unit}:${b.quantity}`)
  );
  return JSON.stringify({
    mode: request.mode,
    query: request.query.trim().toLocaleLowerCase(),
    filters,
    page: request.page,
    substitutionInputs,
    dailyDietId: request.dailyDietId ?? null
  });
}

// Implements DESIGN-001 SearchView store facade.
export interface SearchStateStore extends Writable<SearchState> {
  setMode(mode: SearchMode): void;
}

export function createSearchStateStore(seed: SearchState = initialSearchState()): SearchStateStore {
  const store = writable(seed) as SearchStateStore;
  store.setMode = (mode) => store.update((state) => stateForMode(state, mode));
  return store;
}
