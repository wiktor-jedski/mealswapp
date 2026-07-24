import { expect, test } from "bun:test";
import type { FilterOption, FoodObject } from "./api/generated";
import { substitutionFilterOptions } from "./substitution-filter-options";

// Implements DESIGN-001 SearchView dynamic substitution filter merge verification.

const backendOptions: FilterOption[] = [
	{ filterId: "solid", kind: "physical_state", label: "Stałe", labelKey: "filter.physical_state.solid", includeAllowed: true, excludeAllowed: true, excludes: [] },
	{ filterId: "fruit-id", kind: "food_category", label: "Owoc — świeży", includeAllowed: true, excludeAllowed: true, excludes: [] },
	{ filterId: "vegan", kind: "dietary_preset", label: "Wegańskie", includeAllowed: false, excludeAllowed: true, excludes: [{ filterId: "egg", kind: "allergen" }] }
];
const selectedItem = {
	id: "apple", objectType: "food_item", name: "Apple", physicalState: "solid", imageUrl: null,
	classifications: [
		{ id: "fruit-id", name: "Stale selected label", kind: "food_category" },
		{ id: "snack-id", name: "Snack", kind: "culinary_role" }
	],
	primaryFoodCategory: null, macros: { protein: 0, carbohydrates: 10, fat: 0 }, macroBasis: "100g", calories: 40
} satisfies FoodObject;

test("preserves backend order and localized labels while merging selected classifications once", () => {
	const options = substitutionFilterOptions(backendOptions, [selectedItem, selectedItem], true);
	expect(options.map(({ filterId, label }) => [filterId, label])).toEqual([
		["solid", "Stałe"], ["fruit-id", "Owoc — świeży"], ["snack-id", "Snack"]
	]);
});

test("honors backend operation permissions and uses IDs independently of labels", () => {
	const include = substitutionFilterOptions(backendOptions, [], true);
	const exclude = substitutionFilterOptions(backendOptions, [], false);
	expect(include.map((option) => option.filterId)).toEqual(["solid", "fruit-id"]);
	expect(exclude.map((option) => option.filterId)).toEqual(["solid", "fruit-id", "vegan"]);
	expect(exclude[2]).toMatchObject({ filterId: "vegan", label: "Wegańskie", include: false });
});

test("empty backend data invents no policy but retains selected-item classifications", () => {
	expect(substitutionFilterOptions([], [], false)).toEqual([]);
	expect(substitutionFilterOptions([], [selectedItem], false).map((option) => option.filterId)).toEqual(["fruit-id", "snack-id"]);
});
