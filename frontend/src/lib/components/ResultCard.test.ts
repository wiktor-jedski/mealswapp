import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-001 ResultsGrid result card structure verification.
//
// Bun's isolated install-cache layout breaks transitive resolution for
// `svelte/server`/`svelte/compiler`, and no DOM library (jsdom/happy-dom) is
// installed, so the card cannot be rendered in a Bun unit test. Instead these
// tests assert the component binds FoodObject fields (image, name, Food
// Category classifications, macros with basis, calories), SimilarityMetadata
// score/tier, category-based image placeholder selection, broken-image
// on:error fallback, and traceability via static-source assertions.
// `vite build` compiles the component once Task 151 wires ResultsGrid into
// SearchShell, validating the Svelte source at build time.

const source = readFileSync(join(import.meta.dir, "ResultCard.svelte"), "utf8");

// Implements DESIGN-001 ResultsGrid required FoodObject field bindings verification.
test("binds FoodObject imageUrl, name, classifications, macros, macroBasis, and calories", () => {
	expect(source).toContain("item.imageUrl");
	expect(source).toContain("{item.name}");
	expect(source).toContain("item.classifications");
	expect(source).toContain("item.macroBasis");
	expect(source).toContain("item.calories");
	expect(source).toContain("item.macros.protein");
	expect(source).toContain("item.macros.carbohydrates");
	expect(source).toContain("item.macros.fat");
});

// Implements DESIGN-001 SearchView Catalog-to-Substitution action verification.
test("can expose an optional add-to-substitutions action for Catalog result cards", () => {
	expect(source).toContain("export let onAddToSubstitution");
	expect(source).toContain("data-result-add-substitution");
	expect(source).toContain("absolute bottom-4 right-4");
	expect(source).toContain("pr-12");
	expect(source).toContain("aria-label={`Add ${item.name} to substitutions`}");
	expect(source).toContain("<span class=\"-translate-y-px leading-none\" aria-hidden=\"true\">+</span>");
	expect(source).toContain("onAddToSubstitution?.(item)");
	expect(source).not.toContain("hover:bg-[var(--color-accent)]");
});

// Implements DESIGN-001 ResultsGrid Food Category classifications filtering verification.
test("renders Food Category classifications filtered by kind === food_category", () => {
	expect(source).toContain("foodCategories");
	expect(source).toContain('classification.kind === "food_category"');
	expect(source).toContain("{#each foodCategories as category (category.id)}");
	expect(source).toContain("{category.name}");
	expect(source).toContain("flex flex-wrap");
});

// Implements DESIGN-001 ResultsGrid macro basis label verification.
test("macro basis follows the sidebar unit preference", () => {
	expect(source).toContain("preferencesStore");
	expect(source).toContain("macroBasisDisplayLabel(item.macroBasis, $preferencesStore.unitSystem)");
	expect(source).toContain("macroBasisLabel");
	expect(source).toContain("data-result-macro-basis");
});

// Implements DESIGN-001 ResultsGrid calories display verification.
test("calories are rendered as a nutrition row", () => {
	expect(source).toContain("data-result-calories");
	expect(source).toContain("<dt");
	expect(source).toContain("Calories");
	expect(source).toContain("{formatDisplayQuantity(displayCalories)} kcal");
});

// Implements DESIGN-001 ResultsGrid standardized card layout verification.
test("places name above media, nutrition beside the icon, and tags below", () => {
	const namePos = source.indexOf("data-result-name");
	const imagePos = source.indexOf("data-result-image-wrapper");
	const macrosPos = source.indexOf("data-result-macros");
	const categoriesPos = source.indexOf("data-result-categories");

	expect(namePos).toBeGreaterThan(-1);
	expect(imagePos).toBeGreaterThan(namePos);
	expect(macrosPos).toBeGreaterThan(imagePos);
	expect(categoriesPos).toBeGreaterThan(macrosPos);
	expect(source).toContain("sm:grid-cols-[96px_1fr]");
	expect(source).toContain("grid h-24 content-between");
	expect(source).toContain("grid-cols-[5rem_auto]");
	expect(source).not.toContain("justify-between");
});

// Implements DESIGN-001 ResultsGrid similarity score and tier display verification.
test("binds SimilarityMetadata score and tier and imports the generated types", () => {
	expect(source).toContain("SimilarityMetadata");
	expect(source).toContain("SimilarityTier");
	expect(source).toContain("similarity?.score");
	expect(source).toContain("similarity?.tier");
	expect(source).toContain("data-result-similarity-score");
	expect(source).toContain("data-result-similarity-tier");
});

// Implements DESIGN-001 ResultsGrid backend-calculated replacement quantity verification.
test("renders backend matching quantity with physical-state-aware units", () => {
	expect(source).toContain("similarity.matchingQuantity");
	expect(source).toContain("matchingQuantityLabel");
	expect(source).toContain("displayUnitForBasis(item.macroBasis, $preferencesStore.unitSystem)");
	expect(source).toContain("convertQuantity(similarity.matchingQuantity");
	expect(source).toContain("unitLabel(matchingQuantityDisplayUnit)");
	expect(source).toContain("macroScale = similarity ? similarity.matchingQuantity / 100 : 1");
	expect(source).toContain("macroContextLabel = matchingQuantityLabel ? `for about ${matchingQuantityLabel}` : macroBasisLabel");
	expect(source).toContain("formatDisplayQuantity(displayMacros.protein)");
});

// Implements DESIGN-001 ResultsGrid similarity tier badge styling verification.
test("tier badge maps each SimilarityTier to a labelled style", () => {
	expect(source).toContain("excellent");
	expect(source).toContain("good");
	expect(source).toContain("fair");
	expect(source).toContain("poor");
	expect(source).toContain("Record<SimilarityTier");
});

// Implements DESIGN-001 ResultsGrid category-based image placeholder selection verification.
test("placeholder selects primaryFoodCategory then the first Food Category classification", () => {
	expect(source).toContain("item.primaryFoodCategory");
	expect(source).toContain("foodCategories[0]");
	expect(source).toContain("placeholderCategory");
	expect(source).toContain("placeholderInitial");
});

// Implements DESIGN-001 ResultsGrid broken-image fallback verification.
test("on:error handler toggles imageFailed and renders a category placeholder element", () => {
	expect(source).toContain("on:error={onImageError}");
	expect(source).toContain("onImageError");
	expect(source).toContain("imageFailed");
	expect(source).toContain("showImage");
	expect(source).toContain("data-result-image");
	expect(source).toContain("data-result-placeholder");
});

// Implements DESIGN-001 ResultsGrid image retry on item swap verification.
test("resets the broken-image flag when the item image URL changes", () => {
	expect(source).toContain("resetBrokenImage(item.imageUrl)");
	expect(source).toContain("imageFailed = false");
});

// Implements DESIGN-001 ResultsGrid result card traceability verification.
test("cites the DESIGN-001 ResultsGrid source", () => {
	expect(source).toContain("<!-- Implements DESIGN-001 ResultsGrid -->");
});
