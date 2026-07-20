import { describe, expect, test } from "bun:test";

import { convertQuantity, defaultDisplayQuantity, displayUnitForBasis, formatCalories } from "./units";

// Implements DESIGN-001 SearchView Daily Diet default quantity verification.
describe("defaultDisplayQuantity", () => {
	test.each([
		["100g", "metric", 100],
		["100ml", "metric", 100],
		["100g", "imperial", 3.53],
		["100ml", "imperial", 3.38]
	] as const)("uses the 100-unit nutrition basis for %s in %s mode", (macroBasis, unitSystem, expected) => {
		expect(defaultDisplayQuantity(macroBasis, unitSystem)).toBe(expected);
	});

	test.each([
		["100g", "g"],
		["100ml", "ml"]
	] as const)("keeps the rounded imperial %s default within 0.1 base units", (macroBasis, baseUnit) => {
		const displayUnit = displayUnitForBasis(macroBasis, "imperial");
		const displayQuantity = defaultDisplayQuantity(macroBasis, "imperial");
		expect(Math.abs(convertQuantity(displayQuantity, displayUnit, baseUnit) - 100)).toBeLessThanOrEqual(0.1);
	});
});

// Implements DESIGN-001 SearchView whole-kilocalorie display verification.
test("formatCalories rounds floating-point energy values to the nearest whole kcal", () => {
	expect(formatCalories(98.30000000000001)).toBe("98");
	expect(formatCalories(98.5)).toBe("99");
});
