import { derived, get, writable } from "svelte/store";
import type {
	SearchFilter,
	SearchMode,
	SearchRequest,
	SubstitutionInput,
	FoodObject
} from "../api/generated";
import { selectedDailyDietId } from "./selected-daily-diet";

// Implements DESIGN-001 SearchView typed search state, mode transitions, and SearchRequest construction.

/** Shared state carried by every SearchView mode. */
interface CommonSearchState {
	query: string;
	submittedQuery: string;
	searchSubmitted: boolean;
	filters: SearchFilter[];
	page: number;
	loading: boolean;
	error: string | null;
}

/** Catalog mode has no substitution or daily-diet-specific state. */
export type CatalogSearchState = CommonSearchState & {
	mode: "catalog";
};

/** Substitution mode owns all selected-input display and request state. */
export type SubstitutionSearchState = CommonSearchState & {
	mode: "substitution";
	substitutionInputs: SubstitutionInput[];
	substitutionInputLabels: Record<string, string>;
	substitutionInputItems: Record<string, FoodObject>;
};

/** Minimal saved-diet identity used by the mode model before the full collection workflow lands. */
export interface DailyDietCollectionViewModel {
	id: string;
	name: string;
}

/** Daily Diet mode owns the saved-diet collection list; it has no substitution or alternative id. */
export type DailyDietSearchState = CommonSearchState & {
	mode: "daily_diet";
	dailyDietCollections: DailyDietCollectionViewModel[];
};

/** Daily Diet Alternative mode consumes the shared authoritative saved-diet selection. */
export type DailyDietAlternativeSearchState = CommonSearchState & {
	mode: "daily_diet_alternative";
};

/**
 * Typed SPA search state backing the SearchView.
 *
 * @remarks Implements DESIGN-001 SearchView SearchState as a discriminated union so mode-only
 * fields cannot be represented in incompatible Catalog, Substitution, or Daily Diet states.
 */
export type SearchState =
	| CatalogSearchState
	| SubstitutionSearchState
	| DailyDietSearchState
	| DailyDietAlternativeSearchState;

/**
 * Default Catalog-mode search state.
 *
 * @remarks Implements DESIGN-001 SearchView startup initialization (mode = "catalog").
 */
export function createInitialSearchState(): CatalogSearchState {
	return {
		mode: "catalog",
		query: "",
		submittedQuery: "",
		searchSubmitted: false,
		filters: [],
		page: 1,
		loading: false,
		error: null
	};
}

/**
 * Svelte writable store holding the current SearchView state.
 *
 * @remarks Implements DESIGN-001 SearchView Svelte store initialization.
 */
export const searchStore = writable<SearchState>(createInitialSearchState());

/** Typed projection for components that render only while Substitution mode is active. */
export const substitutionState = derived(searchStore, (state): SubstitutionSearchState | null =>
	state.mode === "substitution" ? state : null
);

/** Typed projection for components that render only while Daily Diet Alternative is active. */
export const dailyDietAlternativeState = derived(
	searchStore,
	(state): DailyDietAlternativeSearchState | null =>
		state.mode === "daily_diet_alternative" ? state : null
);

/** Narrows a SearchState to the mode that owns Substitution Inputs. */
export function isSubstitutionState(state: SearchState): state is SubstitutionSearchState {
	return state.mode === "substitution";
}

/** Narrows a SearchState to the mode that owns the Daily Diet identifier. */
export function isDailyDietAlternativeState(state: SearchState): state is DailyDietAlternativeSearchState {
	return state.mode === "daily_diet_alternative";
}

/**
 * Switches the active search mode, clearing state that is incompatible with the new mode and resetting pagination.
 *
 * @remarks Implements DESIGN-001 SearchView setSearchMode.
 */
export function setMode(mode: SearchMode): void {
	searchStore.update((state): SearchState => {
		if (state.mode === mode) {
			return {
				...state,
				page: 1,
				submittedQuery: "",
				searchSubmitted: false
			};
		}

		const common: CommonSearchState = {
			query: state.query,
			submittedQuery: "",
			searchSubmitted: false,
			filters: state.filters,
			page: 1,
			loading: state.loading,
			error: state.error
		};

		switch (mode) {
			case "catalog":
				return { ...common, mode };
			case "substitution":
				const pending = state.mode === "catalog" ? pendingCatalogSubstitution : [];
				pendingCatalogSubstitution = [];
				return {
					...common,
					mode,
					substitutionInputs: pending.map(({ input }) => input),
					substitutionInputLabels: Object.fromEntries(pending.map(({ input, label, item }) => [input.foodObjectId, item?.name ?? label ?? input.foodObjectId])),
					substitutionInputItems: Object.fromEntries(pending.filter(({ item }) => item).map(({ input, item }) => [input.foodObjectId, item as FoodObject]))
				};
			case "daily_diet":
				return { ...common, mode, dailyDietCollections: [] };
			case "daily_diet_alternative":
				return { ...common, mode };
		}
	});
}

let pendingCatalogSubstitution: Array<{ input: SubstitutionInput; label?: string; item?: FoodObject }> = [];

/**
 * Updates the free-text query and resets pagination so new results start at page one.
 *
 * @remarks Implements DESIGN-001 SearchView query input handling.
 */
export function setQuery(query: string): void {
	searchStore.update((state) => ({
		...state,
		query,
		page: 1
	}));
}

/**
 * Commits the current or provided free-text query for server-side result loading.
 *
 * @remarks Implements DESIGN-001 SearchView committed Catalog search execution.
 */
export function submitSearch(query?: string): void {
	searchStore.update((state) => ({
		...state,
		query: query ?? state.query,
		submittedQuery: query ?? state.query,
		searchSubmitted: (query ?? state.query).trim().length > 0,
		page: 1
	}));
}

/**
 * Commits the current Substitution Input list for server-side substitution result loading.
 *
 * @remarks Implements DESIGN-001 SearchView explicit two-step Substitution Search execution.
 */
export function requestSubstitutionSearch(): void {
	searchStore.update((state) => {
		if (state.mode !== "substitution") {
			return state;
		}
		return {
			...state,
			submittedQuery: "",
			searchSubmitted: state.substitutionInputs.length > 0,
			page: 1
		};
	});
}

/**
 * Replaces the active filter set and resets pagination.
 *
 * @remarks Implements DESIGN-001 SearchView updateFilters.
 */
export function setFilters(filters: SearchFilter[]): void {
	searchStore.update((state) => ({
		...state,
		filters,
		page: 1
	}));
}

/**
 * Adds a filter by id, replacing any existing filter with the same id, and resets pagination.
 *
 * @remarks Implements DESIGN-001 SearchView updateFilters.
 */
export function addFilter(filter: SearchFilter): void {
	searchStore.update((state) => ({
		...state,
		filters: mergeFilter(state.filters, filter),
		page: 1
	}));
}

/**
 * Removes a filter by id and resets pagination.
 *
 * @remarks Implements DESIGN-001 SearchView updateFilters.
 */
export function removeFilter(filterId: string): void {
	searchStore.update((state) => ({
		...state,
		filters: state.filters.filter((existing) => existing.filterId !== filterId),
		page: 1
	}));
}

/**
 * Updates the active page index without touching other search state.
 *
 * @remarks Implements DESIGN-001 SearchView pagination handling.
 */
export function setPage(page: number): void {
	searchStore.update((state) => ({
		...state,
		page: Math.max(1, Math.trunc(page))
	}));
}

/**
 * Adds a substitution input, replacing any input with the same food object id, and resets pagination.
 *
 * @remarks Implements DESIGN-001 SearchView Substitution Input composition.
 */
export function addSubstitutionInput(input: SubstitutionInput, label?: string, item?: FoodObject): void {
	searchStore.update((state) => {
		if (state.mode !== "substitution") {
			if (state.mode === "catalog") {
				pendingCatalogSubstitution = [
					...pendingCatalogSubstitution.filter(({ input: existing }) => existing.foodObjectId !== input.foodObjectId),
					{ input, label, item }
				];
			}
			return state;
		}
		return {
			...state,
			substitutionInputs: mergeSubstitutionInput(state.substitutionInputs, input),
			substitutionInputLabels: {
				...state.substitutionInputLabels,
				[input.foodObjectId]: item?.name ?? label ?? state.substitutionInputLabels[input.foodObjectId] ?? input.foodObjectId
			},
			substitutionInputItems: {
				...state.substitutionInputItems,
				...(item ? { [input.foodObjectId]: item } : {})
			},
			searchSubmitted: false,
			page: 1
		};
	});
}

/**
 * Stores full FoodObject display data for an existing Substitution Input without changing list order.
 *
 * @remarks Implements DESIGN-001 SearchView selected Substitution Input hydration.
 */
export function setSubstitutionInputItem(item: FoodObject): void {
	searchStore.update((state) => {
		if (state.mode !== "substitution") {
			return state;
		}
		if (!state.substitutionInputs.some((input) => input.foodObjectId === item.id)) {
			return state;
		}
		return {
			...state,
			substitutionInputLabels: {
				...state.substitutionInputLabels,
				[item.id]: item.name
			},
			substitutionInputItems: {
				...state.substitutionInputItems,
				[item.id]: item
			}
		};
	});
}

/**
 * Removes a substitution input by food object id and resets pagination.
 *
 * @remarks Implements DESIGN-001 SearchView Substitution Input composition.
 */
export function removeSubstitutionInput(foodObjectId: string): void {
	searchStore.update((state) => {
		if (state.mode !== "substitution") {
			return state;
		}
		const substitutionInputs = state.substitutionInputs.filter(
			(existing) => existing.foodObjectId !== foodObjectId
		);
		return {
			...state,
			substitutionInputs,
			filters: substitutionInputs.length === 0 ? [] : state.filters,
			substitutionInputLabels: omitKey(state.substitutionInputLabels, foodObjectId),
			substitutionInputItems: omitKey(state.substitutionInputItems, foodObjectId),
			searchSubmitted: false,
			page: 1
		};
	});
}

/**
 * Patches a substitution input by food object id and resets pagination.
 *
 * @remarks Implements DESIGN-001 SearchView Substitution Input composition.
 */
export function updateSubstitutionInput(
	foodObjectId: string,
	patch: Partial<Pick<SubstitutionInput, "quantity" | "unit">>
): void {
	searchStore.update((state) => {
		if (state.mode !== "substitution") {
			return state;
		}
		return {
			...state,
			substitutionInputs: state.substitutionInputs.map((existing) =>
				existing.foodObjectId === foodObjectId
					? { ...existing, ...patch }
					: existing
			),
			searchSubmitted: false,
			page: 1
		};
	});
}

/**
 * Sets the active Daily Diet Alternative id and resets pagination.
 *
 * @remarks Implements DESIGN-001 SearchView Daily Diet Alternative selection.
 */
export function setDailyDietId(dailyDietId: string | undefined): void {
	selectedDailyDietId.set(dailyDietId ?? null);
	searchStore.update((state) =>
		state.mode === "daily_diet_alternative" ? { ...state, page: 1 } : state
	);
}

/**
 * Sets the in-flight loading flag used by the SearchView orchestration.
 *
 * @remarks Implements DESIGN-001 SearchView loading state.
 */
export function setLoading(loading: boolean): void {
	searchStore.update((state) => ({
		...state,
		loading
	}));
}

/**
 * Sets or clears the user-facing search error message.
 *
 * @remarks Implements DESIGN-001 SearchView error state.
 */
export function setError(error: string | null): void {
	searchStore.update((state) => ({
		...state,
		error
	}));
}

/**
 * Resets the search store to the default Catalog-mode state.
 *
 * @remarks Implements DESIGN-001 SearchView startup initialization.
 */
export function resetSearch(): void {
	searchStore.set(createInitialSearchState());
}

/**
 * Builds a generated-contract SearchRequest from the current SearchView state without duplicating API types.
 *
 * @remarks Implements DESIGN-001 SearchView buildSearchRequest.
 */
export function buildSearchRequest(state: SearchState, selectedId: string | null = get(selectedDailyDietId)): SearchRequest {
	const request: SearchRequest = {
		query: state.query,
		mode: state.mode,
		page: state.page
	};

	if (state.filters.length > 0) {
		request.filters = state.filters;
	}

	if (state.mode === "substitution") {
		request.substitutionInputs = state.substitutionInputs;
	} else if (state.mode === "daily_diet_alternative" && selectedId !== null) {
		request.dailyDietId = selectedId;
	}

	return request;
}

/**
 * Produces a deterministic cache/query key from mode, query, filters, page, and Substitution Input ids and quantities.
 *
 * @remarks Implements DESIGN-001 SearchView query-key derivation (step 6).
 */
export function searchRequestKey(state: SearchState, selectedId: string | null = get(selectedDailyDietId)): string {
	const inputs = state.mode === "substitution" ? state.substitutionInputs : [];
	const dailyDietId = state.mode === "daily_diet_alternative" ? selectedId : null;
	const normalized = {
		mode: state.mode,
		query: state.query.trim(),
		filters: [...state.filters]
			.sort(compareFilter)
			.map((filter) => ({
				id: filter.filterId,
				kind: filter.kind,
				include: filter.include
			})),
		page: state.page,
		inputs: [...inputs]
			.sort(compareSubstitutionInput)
			.map((input) => ({
				id: input.foodObjectId,
				quantity: input.quantity,
				unit: input.unit
			})),
		dailyDietId: dailyDietId ?? ""
	};

	return JSON.stringify(normalized);
}

function mergeFilter(filters: SearchFilter[], filter: SearchFilter): SearchFilter[] {
	const existing = filters.find((item) => item.filterId === filter.filterId);
	if (existing === undefined) {
		return [...filters, filter];
	}
	return filters.map((item) => (item.filterId === filter.filterId ? filter : item));
}

function mergeSubstitutionInput(
	inputs: SubstitutionInput[],
	input: SubstitutionInput
): SubstitutionInput[] {
	const existing = inputs.find((item) => item.foodObjectId === input.foodObjectId);
	if (existing === undefined) {
		return [...inputs, input];
	}
	return inputs.map((item) => (item.foodObjectId === input.foodObjectId ? input : item));
}

function omitKey<T>(record: Record<string, T>, key: string): Record<string, T> {
	const { [key]: _removed, ...rest } = record;
	return rest;
}

function compareFilter(a: SearchFilter, b: SearchFilter): number {
	if (a.filterId < b.filterId) {
		return -1;
	}
	if (a.filterId > b.filterId) {
		return 1;
	}
	return 0;
}

function compareSubstitutionInput(a: SubstitutionInput, b: SubstitutionInput): number {
	if (a.foodObjectId < b.foodObjectId) {
		return -1;
	}
	if (a.foodObjectId > b.foodObjectId) {
		return 1;
	}
	return 0;
}
