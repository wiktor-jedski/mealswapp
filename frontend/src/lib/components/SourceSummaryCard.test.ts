import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-001 ResultsGrid source summary static-source verification.

const source = readFileSync(join(import.meta.dir, "SourceSummaryCard.svelte"), "utf8");

// Implements DESIGN-001 ResultsGrid substitution source summary verification.
test("renders Your Meal source totals with mass and volume kept separate", () => {
	expect(source).toContain("export let sourceSummary");
	expect(source).toContain("Your Meal");
	expect(source).toContain("data-source-summary-card");
	expect(source).toContain("data-source-summary-amount");
	expect(source).toContain("sourceSummary.totalGrams");
	expect(source).toContain("sourceSummary.totalMilliliters");
	expect(source).not.toContain("Mass and volume are shown separately.");
});

// Implements DESIGN-001 ResultsGrid source macro total verification.
test("renders backend-scaled source macros and calories", () => {
	expect(source).toContain("sourceSummary.macros.protein");
	expect(source).toContain("sourceSummary.macros.carbohydrates");
	expect(source).toContain("sourceSummary.macros.fat");
	expect(source).toContain("sourceSummary.calories");
	expect(source).toContain("data-source-summary-macros");
	expect(source).toContain("data-source-summary-calories");
});

// Implements DESIGN-001 SettingsPanel unit preference propagation verification.
test("converts displayed summary mass and volume using the sidebar unit preference", () => {
	expect(source).toContain("preferencesStore");
	expect(source).toContain('unitSystem === "imperial" ? "oz" : "g"');
	expect(source).toContain('unitSystem === "imperial" ? "fl_oz" : "ml"');
	expect(source).toContain('convertQuantity(sourceSummary.totalGrams, "g", massUnit)');
	expect(source).toContain('convertQuantity(sourceSummary.totalMilliliters, "ml", volumeUnit)');
	expect(source).toContain("unitLabel(massUnit)");
	expect(source).toContain("unitLabel(volumeUnit)");
});
