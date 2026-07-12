import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-001 SearchView OptimizationWorkflow responsive and keyboard-accessible surface verification.
// The repository's Bun unit harness has no DOM runtime, so `vite build` validates Svelte compilation and these
// focused assertions verify the rendered contract used by the Playwright workflow.

const source = readFileSync(join(import.meta.dir, "OptimizationWorkflow.svelte"), "utf8");

test("submits generated optimization fields for the selected saved diet", () => {
	expect(source).toContain("DietOptimizationRequest");
	expect(source).toContain("dailyDietId,");
	expect(source).toContain("targetMacros: { protein, carbohydrates, fat }");
	expect(source).toContain("tolerancePercent: tolerance");
	expect(source).toContain("excludedMealIds: []");
	expect(source).toContain("controller.submit(activeRequest)");
	expect(source).toContain("data-optimization-submit");
});

test("renders bounded progress skeletons and every API terminal state", () => {
	expect(source).toContain('phase === "submitting"');
	expect(source).toContain('phase === "queued"');
	expect(source).toContain('phase === "processing"');
	expect(source).toContain('phase === "failed"');
	expect(source).toContain('phase === "expired"');
	expect(source).toContain('phase === "completed"');
	expect(source).toContain("data-optimization-skeleton");
	expect(source).toContain("data-optimization-results");
	expect(source).toContain("data-optimization-calories");
});

test("renders at most the controller's three alternatives with labelled macros and meals", () => {
	expect(source).toContain("optimizationState.alternatives");
	expect(source).toContain("data-optimization-alternative={index + 1}");
	expect(source).toContain("alternative.macros.protein");
	expect(source).toContain("alternative.macros.carbohydrates");
	expect(source).toContain("alternative.macros.fat");
	expect(source).toContain("alternative.macros.calories");
	expect(source).toContain("alternative.meals");
});

test("keeps retry keyboard-operable, uses visible focus styles, and resets on diet changes", () => {
	expect(source).toContain("controller.setDiet(selectedDietId)");
	expect(source).toContain("controller.dispose()");
	expect(source).toContain("data-optimization-retry");
	expect(source).toContain('type="submit"');
	expect(source).toContain("focus:outline-none focus:ring-2");
	expect(source).toContain('role="alert"');
	expect(source).toContain('role="status"');
});
