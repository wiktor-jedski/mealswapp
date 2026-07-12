import { afterEach, expect, test } from "bun:test";
import { get } from "svelte/store";
import type { SearchRequest, SubstitutionUnit } from "../api/generated";
import {
	addSubstitutionInput,
	buildSearchRequest,
	resetSearch,
	searchStore,
	setDailyDietId,
	setMode
} from "../stores/search";

// Implements DESIGN-001 SearchView Substitution Input canonical unit and quantity round-trip verification.
//
// Exercises the search store (read-only dependency from Task 140) to prove that quantity-bearing
// Substitution Inputs with every canonical unit reach `SearchRequest.substitutionInputs` via
// `buildSearchRequest`, satisfying the Task 144 verification criteria at the contract boundary.

afterEach(() => {
	resetSearch();
});

// Implements DESIGN-001 SearchView canonical unit round-trip verification.
test("each canonical SubstitutionUnit reaches SearchRequest.substitutionInputs via buildSearchRequest", () => {
	setMode("substitution");
	const units: SubstitutionUnit[] = ["g", "ml", "oz", "fl_oz"];
	for (const unit of units) {
		addSubstitutionInput({ foodObjectId: `food-${unit}`, quantity: 100, unit });
	}

	const request = buildSearchRequest(get(searchStore));

	const expected: SearchRequest = {
		query: "",
		mode: "substitution",
		page: 1,
		substitutionInputs: [
			{ foodObjectId: "food-g", quantity: 100, unit: "g" },
			{ foodObjectId: "food-ml", quantity: 100, unit: "ml" },
			{ foodObjectId: "food-oz", quantity: 100, unit: "oz" },
			{ foodObjectId: "food-fl_oz", quantity: 100, unit: "fl_oz" }
		]
	};
	expect(request).toEqual(expected);
	expect(request.substitutionInputs?.map((input) => input.unit)).toEqual(units);
	expect(request).not.toHaveProperty("dailyDietId");
});

// Implements DESIGN-001 SearchView quantity sensitivity at the request boundary verification.
test("Substitution Input quantities are preserved on SearchRequest.substitutionInputs", () => {
	setMode("substitution");
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 250, unit: "g" });

	const request = buildSearchRequest(get(searchStore));

	expect(request.substitutionInputs).toEqual([
		{ foodObjectId: "food-1", quantity: 250, unit: "g" }
	]);
});

// Implements DESIGN-001 SearchView Daily Diet Alternative request shape verification.
test("daily_diet_alternative mode exposes dailyDietId on SearchRequest without substitutionInputs", () => {
	setMode("daily_diet_alternative");
	const dietId = "11111111-2222-3333-4444-555555555555";
	setDailyDietId(dietId);

	const request = buildSearchRequest(get(searchStore));

	expect(request.mode).toBe("daily_diet_alternative");
	expect(request.dailyDietId).toBe(dietId);
	expect(request).not.toHaveProperty("substitutionInputs");
});
