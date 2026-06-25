import type { FoodObject, SubstitutionUnit } from "./api/generated";
import type { UnitSystem } from "./stores/preferences";

// Implements DESIGN-001 SearchView unit-system presentation helpers for metric/imperial frontend display.

const GRAMS_PER_OUNCE = 28.349523125;
const MILLILITERS_PER_FLUID_OUNCE = 29.5735295625;

export type MacroBasis = FoodObject["macroBasis"];

export interface UnitOption {
	value: SubstitutionUnit;
	label: string;
}

/**
 * Returns the unit matching a food object's physical basis and the active sidebar unit system.
 *
 * @remarks Implements DESIGN-001 SettingsPanel unit preference propagation to SearchView controls.
 */
export function displayUnitForBasis(macroBasis: MacroBasis, unitSystem: UnitSystem): SubstitutionUnit {
	if (macroBasis === "100ml") {
		return unitSystem === "imperial" ? "fl_oz" : "ml";
	}
	return unitSystem === "imperial" ? "oz" : "g";
}

/**
 * User-facing label for generated substitution units.
 *
 * @remarks Implements DESIGN-001 SearchView human-readable unit labels.
 */
export function unitLabel(unit: SubstitutionUnit): string {
	return unit === "fl_oz" ? "fl oz" : unit;
}

/**
 * Single compatible substitution-unit option for an item under the active unit preference.
 *
 * @remarks Implements DESIGN-001 SearchView physical-state-aware unit control options.
 */
export function unitOptionsForBasis(macroBasis: MacroBasis, unitSystem: UnitSystem): UnitOption[] {
	const unit = displayUnitForBasis(macroBasis, unitSystem);
	return [{ value: unit, label: unitLabel(unit) }];
}

/**
 * Converts a quantity between compatible metric and imperial mass/volume units.
 *
 * @remarks Implements DESIGN-001 SearchView frontend unit preference recalculation.
 */
export function convertQuantity(quantity: number, fromUnit: SubstitutionUnit, toUnit: SubstitutionUnit): number {
	if (fromUnit === toUnit) {
		return quantity;
	}
	if (fromUnit === "g" && toUnit === "oz") {
		return quantity / GRAMS_PER_OUNCE;
	}
	if (fromUnit === "oz" && toUnit === "g") {
		return quantity * GRAMS_PER_OUNCE;
	}
	if (fromUnit === "ml" && toUnit === "fl_oz") {
		return quantity / MILLILITERS_PER_FLUID_OUNCE;
	}
	if (fromUnit === "fl_oz" && toUnit === "ml") {
		return quantity * MILLILITERS_PER_FLUID_OUNCE;
	}
	return quantity;
}

/**
 * Rounds editable quantities without turning small imperial amounts into zero.
 *
 * @remarks Implements DESIGN-001 SearchView quantity display normalization.
 */
export function normalizeDisplayQuantity(quantity: number): number {
	return Number(quantity.toFixed(quantity >= 10 ? 1 : 2));
}

/**
 * Formats read-only matching quantities for substitution result cards.
 *
 * @remarks Implements DESIGN-001 ResultsGrid backend-calculated matching quantity display.
 */
export function formatDisplayQuantity(quantity: number): string {
	const normalized = normalizeDisplayQuantity(quantity);
	return Number.isInteger(normalized) ? `${normalized}` : `${normalized}`;
}

/**
 * Human-readable nutrition basis label following the active unit preference.
 *
 * @remarks Implements DESIGN-001 ResultsGrid macro basis display.
 */
export function macroBasisDisplayLabel(macroBasis: MacroBasis, unitSystem: UnitSystem): string {
	if (macroBasis === "100ml") {
		return unitSystem === "imperial" ? "values per 3.4 fl oz" : "values per 100 ml";
	}
	return unitSystem === "imperial" ? "values per 3.5 oz" : "values per 100 g";
}
