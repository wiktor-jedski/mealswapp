import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-005 MicronutrientVocabulary administration UI source verification.

const source = readFileSync(join(import.meta.dir, "AdminMicronutrientVocabulary.svelte"), "utf8");

test("vocabulary UI exposes every lifecycle operation without hard delete", () => {
	for (const label of ["Add micronutrient", "Save name", "Deactivate", "Reactivate", "Refresh"]) expect(source).toContain(label);
	expect(source).toContain("api.updateMicronutrientUnit");
	expect(source).not.toContain("deleteMicronutrient");
});

test("vocabulary UI confirms deactivation and refreshes authoritative state after mutations and errors", () => {
	expect(source).toContain('role="alertdialog"');
	expect(source).toContain('aria-modal="true"');
	expect(source).toContain("Confirm deactivation");
	expect(source.match(/api\.listMicronutrients/g)?.length).toBeGreaterThanOrEqual(3);
	expect(source).toContain('role="alert"');
	expect(source).toContain('role="status"');
	expect(source).toContain("this projection contains no private food-item data");
});
