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
	expect(source).toContain("tolerancePercent: tolerance");
	expect(source).toContain("excludedMealIds: []");
	expect(source).not.toContain("targetMacros:");
	expect(source).toContain("controller.submit(activeRequest)");
	expect(source).toContain("data-optimization-submit");
});

test("renders saved-diet macro targets as read-only server-derived values", () => {
	expect(source).toContain("Target Macros");
	expect(source).not.toContain("Optimize this Daily Diet");
	expect(source).not.toContain("Server-derived target macros");
	expect(source).toContain("selectedDiet.aggregateMacros.protein");
	expect(source).toContain("selectedDiet.aggregateMacros.carbohydrates");
	expect(source).toContain("selectedDiet.aggregateMacros.fat");
	expect(source).toContain("data-optimization-target-protein");
	expect(source).not.toContain('id="optimization-protein"');
	expect(source).not.toContain('id="optimization-carbohydrates"');
	expect(source).not.toContain('id="optimization-fat"');
	expect(source).toContain("grid-cols-2 gap-3");
	expect(source).toContain("sm:grid-cols-3");
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
	expect(source).toContain("meal.name");
	expect(source).not.toContain("meal.mealId}");
	expect(source).toContain("Math.round(alternative.similarityScore * 100)");
	expect(source).toContain("Similarity");
});

test("saves each result card as a numbered Daily Diet through the shared mutation flow", () => {
	expect(source).toContain("saveAlternativeDiet(");
	expect(source).toContain("selectedDiet.name");
	expect(source).toContain("index + 1");
	expect(source).toContain("createDailyDiet");
	expect(source).toContain("data-optimization-save={index + 1}");
	expect(source).toContain('savingAlternativeIndex === index ? "Saving…" : "Save"');
	expect(source).toContain("Saved as {alternativeSaveNames[index]}.");
	expect(source).toContain('class="justify-self-start rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)]');
});

test("uses the shared dark numeric input style and omits the redundant results label", () => {
	expect(source).toContain("border-[var(--color-border)] bg-transparent");
	expect(source).not.toContain("bg-white");
	expect(source).not.toContain("Validated alternatives");
});

test("keeps retry keyboard-operable, uses visible focus styles, and resets on diet changes", () => {
	expect(source).toContain("controller.setIdentity(identityId)");
	expect(source).toContain("controller.setDiet(selectedDietId)");
	expect(source).toContain("controller.resume()");
	expect(source).toContain("controller.dispose()");
	expect(source).toContain("controller.retry(activeRequest ?? undefined)");
	expect(source).toContain("data-optimization-retry");
	expect(source).toContain('type="submit"');
	expect(source).toContain("focus:outline-none focus:ring-2");
	expect(source).toContain('role="alert"');
	expect(source).toContain('role="status"');
});

test("shows retry only for policy-approved non-completed outcomes and sends current form input", () => {
	expect(source).toContain('optimizationState.retryMode !== "none" && optimizationState.phase !== "completed"');
	expect(source).toContain("controller.retry(activeRequest ?? undefined)");
	expect(source).toContain("data-optimization-retry");
	expect(source).toContain('optimizationState.phase === "completed"');
	expect(source).toContain("data-optimization-new");
});
