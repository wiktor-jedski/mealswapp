import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-001 SearchView Substitution Input controls static-source verification.
//
// Mirrors the static component test approach: no DOM library is installed, so the component is
// not rendered in Bun. These tests assert the Svelte source declares canonical units, explicit
// two-step substitution search, card-style selected item labels, removal and update bindings,
// labelled controls, and a traceability comment. `vite build` compiles the component via SearchShell.

const source = readFileSync(join(import.meta.dir, "SubstitutionInputs.svelte"), "utf8");

function countOccurrences(haystack: string, needle: string): number {
	return haystack.split(needle).length - 1;
}

// Implements DESIGN-001 SearchView unit preference propagation verification.
test("limits substitution row units through the active sidebar unit preference", () => {
	expect(source).toContain("preferencesStore");
	expect(source).toContain("unitOptionsForBasis(item?.macroBasis ?? \"100g\", unitSystem)");
	expect(source).toContain("rowUnitOptions(selectedItem, $preferencesStore.unitSystem)");
	expect(source).toContain("synchronizeInputUnits");
	expect(source).toContain("displayUnitForBasis(item.macroBasis, unitSystem)");
	expect(source).toContain("convertQuantity(input.quantity, input.unit, targetUnit)");
});

// Implements DESIGN-001 SearchView explicit two-step Substitution Search verification.
test("declares explicit Find substitutions action without a raw food object id input", () => {
	expect(source).toContain("requestSubstitutionSearch");
	expect(source).toContain("data-substitution-search");
	expect(source).toContain("Find substitutions");
	expect(source).toContain("disabled={$searchStore.substitutionInputs.length === 0 || isBlocked || isLoading}");
	expect(source).not.toContain('id="substitution-food-object-id"');
	expect(source).not.toContain("Food object id");
	expect(source).not.toContain("addInput");
});

// Implements DESIGN-001 SearchView substitution filter picker verification.
test("declares include and exclude substitution filter comboboxes with removable chips", () => {
	expect(source).toContain('aria-label="Substitution filters"');
	expect(source).toContain('data-substitution-filters');
	expect(source).toContain("{#if $searchStore.substitutionInputs.length > 0}");
	expect(source).toContain('id="substitution-include-filter"');
	expect(source).toContain('id="substitution-exclude-filter"');
	expect(source).toContain("Must include");
	expect(source).toContain("Must exclude");
	expect(source).toContain("role=\"combobox\"");
	expect(source).toContain("data-substitution-include-options");
	expect(source).toContain("data-substitution-exclude-options");
	expect(source).toContain("data-substitution-include-chips");
	expect(source).toContain("data-substitution-exclude-chips");
	expect(source).toContain("addSubstitutionFilter");
	expect(source).toContain("removeSubstitutionFilter");
	expect(source).toContain("setFilters");
});

// Implements DESIGN-001 SearchView substitution filter option verification.
test("maps user-facing substitution filters to backend filter kinds", () => {
	expect(source).toContain('filterId: "dairy"');
	expect(source).toContain('kind: "allergen"');
	expect(source).toContain('filterId: "dairy_free"');
	expect(source).toContain('kind: "dietary_preset"');
	expect(source).toContain('filterId: "solid"');
	expect(source).toContain('kind: "physical_state"');
	expect(source).toContain('classification.kind === "food_category"');
	expect(source).toContain('classification.kind === "food_category" ? "Food category" : "Culinary role"');
});

// Implements DESIGN-001 SearchView Substitution Input removal and update verification.
test("removal calls removeSubstitutionInput and row edits call updateSubstitutionInput", () => {
	expect(source).toContain("removeSubstitutionInput(input.foodObjectId)");
	expect(source).toContain("updateSubstitutionInput");
	expect(source).toContain("oninput={(event) => onRowQuantityInput(input.foodObjectId, event)}");
	expect(source).toContain("onchange={(event) => onRowUnitChange(input.foodObjectId, event)}");
});

// Implements DESIGN-001 SearchView human-facing substitution input label verification.
test("renders selected items as cards by human-facing label instead of raw id when available", () => {
	expect(source).toContain("inputLabel(input.foodObjectId)");
	expect(source).toContain("inputInitial(input.foodObjectId)");
	expect(source).toContain("inputItem(input.foodObjectId)");
	expect(source).toContain("substitutionInputLabels");
	expect(source).toContain("substitutionInputItems");
	expect(source).toContain("data-food-object-id={input.foodObjectId}");
	expect(source).toContain("data-substitution-card");
	expect(source).toContain("data-substitution-placeholder");
	expect(source).toContain("data-substitution-controls");
	expect(source).toContain("sm:grid-cols-[96px_1fr_auto]");
});

// Implements DESIGN-001 SearchView rich Catalog-selected substitution card verification.
test("renders full macro, calorie, category, and image data for catalog-added substitution items", () => {
	expect(source).toContain("data-substitution-macros");
	expect(source).toContain("data-substitution-calories");
	expect(source).toContain("data-substitution-macro-basis");
	expect(source).toContain("data-substitution-categories");
	expect(source).toContain("data-substitution-image-wrapper");
	expect(source).toContain("data-substitution-image");
	expect(source).toContain("selectedItem.macros.protein");
	expect(source).toContain("selectedItem.calories");
	expect(source).toContain("macroBasisDisplayLabel(item.macroBasis, $preferencesStore.unitSystem)");
	expect(source).toContain("grid h-24 content-between");
});

// Implements DESIGN-001 SearchView Substitution Input labelled controls and traceability verification.
test("section landmark labels all controls and cites the DESIGN source", () => {
	expect(source).toContain('aria-label="Substitution inputs"');
	expect(source).toContain("<!-- Implements DESIGN-001 SearchView Substitution Input controls");
	expect(source).toContain('data-substitution-empty');
	expect(source).toContain('id={`qty-${input.foodObjectId}`}');
	expect(source).toContain('id={`unit-${input.foodObjectId}`}');
	expect(source).toContain("Quantity");
	expect(source).toContain("Unit");
	expect(source).toContain("aria-label={`Remove ${inputLabel(input.foodObjectId)} from substitutions`}");
	expect(source).toContain("<span class=\"-translate-y-px leading-none\" aria-hidden=\"true\">−</span>");
	expect(source).toContain("rounded-full");
	expect(source).toContain("absolute bottom-4 right-4");
});

// Implements DESIGN-001 SearchView Substitution Input focus states verification.
test("draft and row controls declare visible Tailwind focus states", () => {
	expect(countOccurrences(source, "focus:ring-2")).toBeGreaterThanOrEqual(4);
	expect(countOccurrences(source, "focus:outline-none")).toBeGreaterThanOrEqual(4);
});

// Implements DESIGN-001 SearchView right-aligned compact Substitution Input controls verification.
test("quantity, unit, and remove controls render as a compact right-side column", () => {
	expect(source).toContain("justify-self-start sm:justify-self-end");
	expect(source).toContain("grid-cols-[10.5ch_7ch]");
	expect(source).toContain("w-[7ch]");
	expect(source).toContain("h-8 w-[10.5ch]");
	expect(source).toContain("gap-0.5");
	expect(source).toContain("pr-12");
	expect(source).toContain("border-[var(--color-accent)]");
	expect(source).toContain("bg-[var(--color-accent)]");
	expect(source).not.toContain("hover:bg-[var(--color-primary)]");
});

// Implements DESIGN-001 SearchView single-input vs multi-input limit verification.
test("conditionally renders entitlement feedback for multi-input limits", () => {
	expect(source).toContain('entitlement !== undefined && !entitlement.allowedModes.includes("substitution:multi")');
	expect(source).toContain("data-entitlement-feedback");
});
