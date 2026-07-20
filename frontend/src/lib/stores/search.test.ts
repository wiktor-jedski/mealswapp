import { afterEach, expect, test } from "bun:test";
import { get } from "svelte/store";
import type { FoodObject, SearchRequest } from "../api/generated";
import {
	addFilter,
	addSubstitutionInput,
	buildSearchRequest,
	createInitialSearchState,
	removeFilter,
	removeSubstitutionInput,
	requestSubstitutionSearch,
	resetSearch,
	searchRequestKey,
	searchStore,
	setDailyDietId,
	setError,
	setFilters,
	setLoading,
	setMode,
	setPage,
	setQuery,
	setSubstitutionInputItem,
	submitSearch,
	updateSubstitutionInput
} from "./search";
import type {
	DailyDietAlternativeSearchState,
	DailyDietSearchState,
	SearchState,
	SubstitutionSearchState
} from "./search";
import { selectedDailyDietId } from "./selected-daily-diet";

afterEach(() => {
	resetSearch();
	selectedDailyDietId.set(null);
});

function foodObject(id = "food-1", name = "Apple"): FoodObject {
	return {
		id,
		name,
		physicalState: "solid",
		imageUrl: null,
		classifications: [{ id: "cat-fruit", name: "Fruit", kind: "food_category" }],
		primaryFoodCategory: { id: "cat-fruit", name: "Fruit", kind: "food_category" },
		macros: { protein: 1, carbohydrates: 14, fat: 0 },
		macroBasis: "100g",
		calories: 52
	};
}

function currentSubstitutionState(): SubstitutionSearchState {
	const state = get(searchStore);
	if (state.mode !== "substitution") {
		throw new Error(`Expected substitution mode, got ${state.mode}`);
	}
	return state;
}

function currentDailyDietAlternativeState(): DailyDietAlternativeSearchState {
	const state = get(searchStore);
	if (state.mode !== "daily_diet_alternative") {
		throw new Error(`Expected daily diet alternative mode, got ${state.mode}`);
	}
	return state;
}

function currentDailyDietState(): DailyDietSearchState {
	const state = get(searchStore);
	if (state.mode !== "daily_diet") {
		throw new Error(`Expected daily diet mode, got ${state.mode}`);
	}
	return state;
}

// Implements DESIGN-001 SearchView initial mode verification.
test("createInitialSearchState defaults to catalog mode", () => {
	const state = createInitialSearchState();
	expect(state.mode).toBe("catalog");
});

// Implements DESIGN-001 SearchView initial store value verification.
test("searchStore starts in catalog mode with empty query and page 1", () => {
	const state = get(searchStore);
	expect(state.mode).toBe("catalog");
	expect(state.query).toBe("");
	expect(state.submittedQuery).toBe("");
	expect(state.searchSubmitted).toBe(false);
	expect(state.page).toBe(1);
	expect(state.filters).toEqual([]);
	expect(state).not.toHaveProperty("substitutionInputs");
	expect(state).not.toHaveProperty("substitutionInputLabels");
	expect(state).not.toHaveProperty("substitutionInputItems");
	expect(state).not.toHaveProperty("dailyDietId");
	expect(state.loading).toBe(false);
	expect(state.error).toBeNull();
});

// Implements DESIGN-001 SearchView mode transition verification.
test("setMode clears substitution inputs when leaving substitution mode and resets page", () => {
	setMode("substitution");
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" });
	setPage(3);

	setMode("catalog");

	const state = get(searchStore);
	expect(state.mode).toBe("catalog");
	expect(state).not.toHaveProperty("substitutionInputs");
	expect(state).not.toHaveProperty("substitutionInputLabels");
	expect(state).not.toHaveProperty("substitutionInputItems");
	expect(state.submittedQuery).toBe("");
	expect(state.searchSubmitted).toBe(false);
	expect(state.page).toBe(1);
});

// Implements DESIGN-001 SearchView daily-diet mode transition verification.
test("setMode preserves authoritative selection when leaving daily_diet_alternative and resets page", () => {
	setMode("daily_diet_alternative");
	setDailyDietId("diet-42");
	setPage(4);

	setMode("catalog");

	const state = get(searchStore);
	expect(state.mode).toBe("catalog");
	expect(state).not.toHaveProperty("dailyDietId");
	expect(get(selectedDailyDietId)).toBe("diet-42");
	expect(state.page).toBe(1);
});

// Implements DESIGN-001 SearchView mode-specific state shape verification.
test("setMode creates only the valid Daily Diet and Daily Diet Alternative fields", () => {
	setMode("daily_diet");
	const dailyDiet = currentDailyDietState();
	expect(dailyDiet.dailyDietCollections).toEqual([]);
	expect(dailyDiet).not.toHaveProperty("substitutionInputs");
	expect(dailyDiet).not.toHaveProperty("dailyDietId");

	setMode("daily_diet_alternative");
	const alternative = currentDailyDietAlternativeState();
	expect(alternative).not.toHaveProperty("dailyDietId");
	expect(alternative).not.toHaveProperty("substitutionInputs");
	expect(alternative).not.toHaveProperty("dailyDietCollections");
});

// Implements DESIGN-001 SearchView same-mode setMode verification.
test("setMode keeps compatible state when reselecting the same mode", () => {
	setMode("substitution");
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" });

	setMode("substitution");

	const state = get(searchStore);
	const substitutionState = currentSubstitutionState();
	expect(substitutionState.substitutionInputs).toHaveLength(1);
	expect(substitutionState.substitutionInputItems["food-1"]).toBeUndefined();
	expect(state.page).toBe(1);
});

// Implements DESIGN-001 SearchView query change pagination reset verification.
test("setQuery resets page to 1", () => {
	setPage(5);
	setQuery("apple");
	expect(get(searchStore).page).toBe(1);
	expect(get(searchStore).query).toBe("apple");
	expect(get(searchStore).submittedQuery).toBe("");
	expect(get(searchStore).searchSubmitted).toBe(false);
});

// Implements DESIGN-001 SearchView committed query verification.
test("submitSearch commits the current or provided query and resets page", () => {
	setQuery("apple");
	setPage(3);
	submitSearch();
	expect(get(searchStore).submittedQuery).toBe("apple");
	expect(get(searchStore).searchSubmitted).toBe(true);
	expect(get(searchStore).page).toBe(1);

	submitSearch("yogurt");
	expect(get(searchStore).query).toBe("yogurt");
	expect(get(searchStore).submittedQuery).toBe("yogurt");
	expect(get(searchStore).searchSubmitted).toBe(true);
});

// Implements DESIGN-001 SearchView explicit two-step Substitution Search verification.
test("requestSubstitutionSearch submits only when substitution inputs exist", () => {
	setMode("substitution");
	requestSubstitutionSearch();
	expect(get(searchStore).searchSubmitted).toBe(false);

	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" }, "Apple");
	requestSubstitutionSearch();
	expect(get(searchStore).submittedQuery).toBe("");
	expect(get(searchStore).searchSubmitted).toBe(true);
});

// Implements DESIGN-001 SearchView filter change pagination reset verification.
test("setFilters resets page to 1", () => {
	setPage(4);
	setFilters([{ filterId: "cat-fruit", kind: "food_category", include: true }]);
	expect(get(searchStore).page).toBe(1);
	expect(get(searchStore).filters).toHaveLength(1);
});

// Implements DESIGN-001 SearchView filter add/remove pagination reset verification.
test("addFilter and removeFilter reset page to 1", () => {
	setPage(7);
	addFilter({ filterId: "cat-fruit", kind: "food_category", include: true });
	expect(get(searchStore).page).toBe(1);

	setPage(7);
	removeFilter("cat-fruit");
	expect(get(searchStore).page).toBe(1);
	expect(get(searchStore).filters).toEqual([]);
});

// Implements DESIGN-001 SearchView filter dedup verification.
test("addFilter replaces existing filters with the same id", () => {
	addFilter({ filterId: "cat-fruit", kind: "food_category", include: true });
	addFilter({ filterId: "cat-fruit", kind: "food_category", include: false });

	const filters = get(searchStore).filters;
	expect(filters).toHaveLength(1);
	expect(filters[0]?.include).toBe(false);
});

// Implements DESIGN-001 SearchView substitution input pagination reset verification.
test("substitution input add, update, and remove reset page to 1", () => {
	setMode("substitution");
	setPage(6);

	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" });
	expect(get(searchStore).page).toBe(1);

	requestSubstitutionSearch();
	expect(get(searchStore).searchSubmitted).toBe(true);

	setPage(6);
	updateSubstitutionInput("food-1", { quantity: 200 });
	expect(get(searchStore).page).toBe(1);
	expect(get(searchStore).searchSubmitted).toBe(false);
	expect(currentSubstitutionState().substitutionInputs[0]?.quantity).toBe(200);

	requestSubstitutionSearch();
	expect(get(searchStore).searchSubmitted).toBe(true);

	setPage(6);
	removeSubstitutionInput("food-1");
	expect(get(searchStore).page).toBe(1);
	expect(get(searchStore).searchSubmitted).toBe(false);
	expect(currentSubstitutionState().substitutionInputs).toEqual([]);
});

// Implements DESIGN-001 SearchView substitution input dedup verification.
test("addSubstitutionInput replaces existing inputs with the same food object id", () => {
	setMode("substitution");
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" });
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 200, unit: "ml" }, "Apple");

	const inputs = currentSubstitutionState().substitutionInputs;
	expect(inputs).toHaveLength(1);
	expect(inputs[0]).toEqual({ foodObjectId: "food-1", quantity: 200, unit: "ml" });
	expect(currentSubstitutionState().substitutionInputLabels["food-1"]).toBe("Apple");
});

// Implements DESIGN-001 SearchView Catalog-to-Substitution selected item display data verification.
test("addSubstitutionInput can preserve full FoodObject display data for catalog-added items", () => {
	setMode("substitution");
	const item = foodObject();
	addSubstitutionInput({ foodObjectId: item.id, quantity: 100, unit: "g" }, item.name, item);

	const state = currentSubstitutionState();
	expect(state.substitutionInputLabels[item.id]).toBe("Apple");
	expect(state.substitutionInputItems[item.id]).toEqual(item);

	removeSubstitutionInput(item.id);
	expect(currentSubstitutionState().substitutionInputItems[item.id]).toBeUndefined();
});

// Implements DESIGN-001 SearchView substitution filter cleanup verification.
test("removeSubstitutionInput clears filters when the input list becomes empty", () => {
	setMode("substitution");
	const item = foodObject();
	addSubstitutionInput({ foodObjectId: item.id, quantity: 100, unit: "g" }, item.name, item);
	setFilters([{ filterId: "cat-fruit", kind: "food_category", include: true }]);

	removeSubstitutionInput(item.id);

	const state = get(searchStore);
	expect(currentSubstitutionState().substitutionInputs).toEqual([]);
	expect(state.filters).toEqual([]);
	expect(state.searchSubmitted).toBe(false);
});

// Implements DESIGN-001 SearchView selected Substitution Input hydration verification.
test("setSubstitutionInputItem hydrates display data without reordering the input list", () => {
	setMode("substitution");
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" }, "Apple");
	addSubstitutionInput({ foodObjectId: "food-2", quantity: 50, unit: "g" }, "Pear");
	const hydrated = foodObject("food-1", "Apple Hydrated");

	setSubstitutionInputItem(hydrated);

	const state = currentSubstitutionState();
	expect(state.substitutionInputs.map((input) => input.foodObjectId)).toEqual(["food-1", "food-2"]);
	expect(state.substitutionInputLabels["food-1"]).toBe("Apple Hydrated");
	expect(state.substitutionInputItems["food-1"]).toEqual(hydrated);
});

// Implements DESIGN-001 SearchView selected Substitution Input hydration race verification.
test("setSubstitutionInputItem ignores late hydration after an input is removed", () => {
	setMode("substitution");
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" }, "Apple");
	removeSubstitutionInput("food-1");

	setSubstitutionInputItem(foodObject("food-1", "Apple Hydrated"));

	const state = currentSubstitutionState();
	expect(state.substitutionInputs).toEqual([]);
	expect(state.substitutionInputLabels["food-1"]).toBeUndefined();
	expect(state.substitutionInputItems["food-1"]).toBeUndefined();
});

// Implements DESIGN-001 SearchView daily diet id pagination reset verification.
test("setDailyDietId resets page to 1", () => {
	setMode("daily_diet_alternative");
	setPage(5);
	setDailyDietId("diet-9");
	expect(get(searchStore).page).toBe(1);
	expect(get(selectedDailyDietId)).toBe("diet-9");
});

// Implements DESIGN-001 SearchView page index verification.
test("setPage updates the page index without resetting other state", () => {
	setQuery("apple");
	setPage(5);
	expect(get(searchStore).page).toBe(5);
	expect(get(searchStore).query).toBe("apple");
});

// Implements DESIGN-001 SearchView loading and error flag verification.
test("setLoading and setError update search state flags", () => {
	setLoading(true);
	setError("Something went wrong");

	const inFlight = get(searchStore);
	expect(inFlight.loading).toBe(true);
	expect(inFlight.error).toBe("Something went wrong");

	setLoading(false);
	setError(null);

	const cleared = get(searchStore);
	expect(cleared.loading).toBe(false);
	expect(cleared.error).toBeNull();
});

// Implements DESIGN-001 SearchView reset verification.
test("resetSearch restores the default catalog state", () => {
	setMode("substitution");
	setQuery("flour");
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" });
	setPage(4);

	resetSearch();

	const state = get(searchStore);
	expect(state.mode).toBe("catalog");
	expect(state.query).toBe("");
	expect(state.submittedQuery).toBe("");
	expect(state.searchSubmitted).toBe(false);
	expect(state).not.toHaveProperty("substitutionInputs");
	expect(state).not.toHaveProperty("substitutionInputLabels");
	expect(state).not.toHaveProperty("substitutionInputItems");
	expect(state.page).toBe(1);
});

// Implements DESIGN-001 SearchView catalog request construction verification.
test("buildSearchRequest omits substitutionInputs and dailyDietId for catalog mode", () => {
	setQuery("rice");
	addFilter({ filterId: "cat-grain", kind: "food_category", include: true });
	setPage(2);

	const request = buildSearchRequest(get(searchStore));

	const expected: SearchRequest = {
		query: "rice",
		mode: "catalog",
		filters: [{ filterId: "cat-grain", kind: "food_category", include: true }],
		page: 2
	};
	expect(request).toEqual(expected);
	expect(request).not.toHaveProperty("substitutionInputs");
	expect(request).not.toHaveProperty("dailyDietId");
});

// Implements DESIGN-001 SearchView substitution request construction verification.
test("buildSearchRequest includes substitutionInputs in substitution mode", () => {
	setMode("substitution");
	setQuery("flour");
	addSubstitutionInput({ foodObjectId: "food-7", quantity: 50, unit: "g" });

	const request = buildSearchRequest(get(searchStore));

	expect(request.mode).toBe("substitution");
	expect(request.substitutionInputs).toEqual([
		{ foodObjectId: "food-7", quantity: 50, unit: "g" }
	]);
	expect(request).not.toHaveProperty("dailyDietId");
});

// Implements DESIGN-001 SearchView daily-diet request construction verification.
test("buildSearchRequest includes dailyDietId in daily_diet_alternative mode", () => {
	setMode("daily_diet_alternative");
	setDailyDietId("diet-100");

	const request = buildSearchRequest(get(searchStore));

	expect(request.mode).toBe("daily_diet_alternative");
	expect(request.dailyDietId).toBe("diet-100");
	expect(request).not.toHaveProperty("substitutionInputs");
});

// Implements DESIGN-001 SearchView request key content verification.
test("searchRequestKey includes mode, query, filters, page, and input quantities", () => {
	setMode("substitution");
	setQuery("flour");
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" });
	addFilter({ filterId: "cat-grain", kind: "food_category", include: true });
	setPage(3);

	const key = searchRequestKey(get(searchStore));

	expect(key).toContain('"mode":"substitution"');
	expect(key).toContain('"query":"flour"');
	expect(key).toContain('"page":3');
	expect(key).toContain('"id":"cat-grain"');
	expect(key).toContain('"id":"food-1"');
	expect(key).toContain('"quantity":100');
});

// Implements DESIGN-001 SearchView request key determinism verification.
test("searchRequestKey is deterministic and normalizes filter and input order", () => {
	setMode("substitution");
	setQuery("flour");
	setPage(2);
	addFilter({ filterId: "cat-b", kind: "food_category", include: true });
	addFilter({ filterId: "cat-a", kind: "food_category", include: true });
	addSubstitutionInput({ foodObjectId: "food-2", quantity: 200, unit: "ml" });
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" });
	const keyA = searchRequestKey(get(searchStore));

	resetSearch();
	setMode("substitution");
	setQuery("flour");
	setPage(2);
	addFilter({ filterId: "cat-a", kind: "food_category", include: true });
	addFilter({ filterId: "cat-b", kind: "food_category", include: true });
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" });
	addSubstitutionInput({ foodObjectId: "food-2", quantity: 200, unit: "ml" });
	const keyB = searchRequestKey(get(searchStore));

	expect(keyA).toBe(keyB);
});

// Implements DESIGN-001 SearchView request key input quantity sensitivity verification.
test("searchRequestKey changes when a substitution input quantity changes", () => {
	setMode("substitution");
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" });
	const beforeKey = searchRequestKey(get(searchStore));

	updateSubstitutionInput("food-1", { quantity: 250 });
	const afterKey = searchRequestKey(get(searchStore));

	expect(afterKey).not.toBe(beforeKey);
});

// Implements DESIGN-001 SearchView substitution input no-match update verification.
test("updateSubstitutionInput leaves other inputs unchanged when the id does not match", () => {
	setMode("substitution");
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" });
	setPage(6);

	updateSubstitutionInput("food-missing", { quantity: 200 });

	const state = get(searchStore);
	expect(state.page).toBe(1);
	expect(state.substitutionInputs).toHaveLength(1);
	expect(state.substitutionInputs[0]).toEqual({ foodObjectId: "food-1", quantity: 100, unit: "g" });
});

// Implements DESIGN-001 SearchView request key equal-id comparator verification.
test("searchRequestKey is stable for duplicate filter and substitution input ids", () => {
	const withDuplicates: SearchState = {
		mode: "substitution",
		query: "",
		submittedQuery: "",
		searchSubmitted: false,
		filters: [
			{ filterId: "dup", kind: "food_category", include: true },
			{ filterId: "dup", kind: "food_category", include: true }
		],
		substitutionInputs: [
			{ foodObjectId: "dup-food", quantity: 100, unit: "g" },
			{ foodObjectId: "dup-food", quantity: 200, unit: "ml" }
		],
		page: 1,
		substitutionInputLabels: {},
		substitutionInputItems: {},
		loading: false,
		error: null
	};

	const key = searchRequestKey(withDuplicates);
	expect(typeof key).toBe("string");
	expect(key).toContain('"id":"dup"');
	expect(key).toContain('"id":"dup-food"');

	// The same duplicate-id state produces the same key on a second call.
	expect(searchRequestKey(withDuplicates)).toBe(key);
});

// Implements DESIGN-001 SearchView request key mode sensitivity verification.
test("searchRequestKey changes when the mode changes", () => {
	setQuery("flour");
	const catalogKey = searchRequestKey(get(searchStore));

	setMode("substitution");
	const substitutionKey = searchRequestKey(get(searchStore));

	expect(catalogKey).not.toBe(substitutionKey);
});
