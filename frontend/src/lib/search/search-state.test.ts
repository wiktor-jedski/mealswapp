import { describe, expect, test } from "bun:test";
import { get } from "svelte/store";
import type { SearchRequest } from "../api/generated";
import { addSearchFilter, addSubstitutionInput, buildSearchRequest, createSearchStateStore, initialSearchState, removeSearchFilter, removeSubstitutionInput, searchRequestKey, stateForMode } from "./search-state";

// Implements DESIGN-001 SearchView state and request-builder verification.
describe("search state", () => {
  test("starts in Catalog with every macro enabled", () => {
    const state = initialSearchState();
    expect(state.mode).toBe("catalog");
    expect(state.page).toBe(1);
    expect(state.enabledMacros).toEqual({ protein: true, carbohydrate: true, fat: true });
    expect(state.loading).toBe(false);
    expect(state.error).toBeNull();
  });

  test("mode changes reset pagination and incompatible state", () => {
    const seeded = {
      ...initialSearchState(),
      mode: "substitution" as const,
      page: 4,
      substitutionInputs: [{ foodObjectId: "food-1", quantity: 25, unit: "g" as const }],
      dailyDietId: "diet-1",
      loading: true
    };
    const daily = stateForMode(seeded, "daily_diet_alternative");
    expect(daily.page).toBe(1);
    expect(daily.substitutionInputs).toEqual([]);
    expect(daily.dailyDietId).toBe("diet-1");
    expect(daily.loading).toBe(false);
    expect(stateForMode(daily, "catalog").dailyDietId).toBeUndefined();
  });

  test("store exposes typed mode transitions", () => {
    const store = createSearchStateStore({ ...initialSearchState(), page: 3 });
    store.setMode("substitution");
    expect(get(store).mode).toBe("substitution");
    expect(get(store).page).toBe(1);
  });

  test("builds generated SearchRequest variants without incompatible fields", () => {
    const request: SearchRequest = buildSearchRequest({
      ...initialSearchState(),
      mode: "substitution",
      query: " apple ",
      page: 2,
      substitutionInputs: [{ foodObjectId: "food-1", quantity: 150, unit: "g" }]
    });
    expect(request).toEqual({
      query: "apple",
      mode: "substitution",
      filters: [],
      page: 2,
      substitutionInputs: [{ foodObjectId: "food-1", quantity: 150, unit: "g" }]
    });
  });

  test("request keys normalize query and include filters, page, and input quantities", () => {
    const first: SearchRequest = {
      query: " Apple ", mode: "substitution", page: 2,
      filters: [{ filterId: "fruit", kind: "food_category", include: true }],
      substitutionInputs: [{ foodObjectId: "food-1", quantity: 100, unit: "g" }]
    };
    const equivalent = { ...first, query: "apple" };
    expect(searchRequestKey(first)).toBe(searchRequestKey(equivalent));
    expect(searchRequestKey(first)).not.toBe(searchRequestKey({ ...equivalent, page: 3 }));
    expect(searchRequestKey(first)).not.toBe(searchRequestKey({ ...equivalent, substitutionInputs: [{ ...first.substitutionInputs![0], quantity: 200 }] }));
    const reordered = { ...first, filters: [
      { filterId: "role", kind: "culinary_role" as const, include: true },
      first.filters![0]
    ] };
    expect(searchRequestKey(reordered)).toBe(searchRequestKey({ ...reordered, filters: [...reordered.filters].reverse() }));
    const reorderedInputs = { ...first, substitutionInputs: [first.substitutionInputs![0], { foodObjectId: "food-2", quantity: 10, unit: "oz" as const }] };
    expect(searchRequestKey(reorderedInputs)).toBe(searchRequestKey({ ...reorderedInputs, substitutionInputs: [...reorderedInputs.substitutionInputs].reverse() }));
  });

  test("accumulates, replaces, and removes quantity-bearing substitution inputs", () => {
    const initial = { ...initialSearchState(), mode: "substitution" as const };
    const added = addSubstitutionInput(initial, { foodObjectId: "food-1", quantity: 100, unit: "g" });
    const replaced = addSubstitutionInput(added, { foodObjectId: "food-1", quantity: 2, unit: "oz" });
    expect(replaced.substitutionInputs).toEqual([{ foodObjectId: "food-1", quantity: 2, unit: "oz" }]);
    expect(buildSearchRequest(replaced).substitutionInputs).toEqual(replaced.substitutionInputs);
    expect(removeSubstitutionInput(replaced, "food-1").substitutionInputs).toEqual([]);
  });

  test("accumulates and replaces typed filters", () => {
    const included = addSearchFilter(initialSearchState(), { filterId: "fruit", kind: "food_category", include: true });
    const excluded = addSearchFilter(included, { filterId: "fruit", kind: "food_category", include: false });
    expect(excluded.filters).toEqual([{ filterId: "fruit", kind: "food_category", include: false }]);
    expect(removeSearchFilter(excluded, excluded.filters[0]).filters).toEqual([]);
  });

  test("creates a store with default state when no seed is supplied", () => expect(get(createSearchStateStore()).mode).toBe("catalog"));
});
