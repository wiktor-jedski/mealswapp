import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-001 SearchView Substitution Input controls static-source verification.
//
// Mirrors the SettingsPanel test approach: no DOM library is installed, so the component is
// not rendered in Bun. These tests assert the Svelte source declares canonical units, the
// add path (Enter + Add button) calling `addSubstitutionInput`, deterministic duplicate
// rejection, positive quantity validation, removal and update bindings, labelled controls,
// and a traceability comment. `vite build` compiles the component via SearchShell.

const source = readFileSync(join(import.meta.dir, "SubstitutionInputs.svelte"), "utf8");

function countOccurrences(haystack: string, needle: string): number {
	return haystack.split(needle).length - 1;
}

// Implements DESIGN-001 SearchView canonical SubstitutionUnit set verification.
test("declares all four canonical SubstitutionUnit options g, ml, oz, and fl_oz", () => {
	expect(source).toContain('value: "g"');
	expect(source).toContain('value: "ml"');
	expect(source).toContain('value: "oz"');
	expect(source).toContain('value: "fl_oz"');
});

// Implements DESIGN-001 SearchView Substitution Input accumulation verification.
test("Enter on the foodObjectId input and Add button both add one Substitution Input via addSubstitutionInput", () => {
	expect(source).toContain("addSubstitutionInput");
	expect(source).toContain("addInput");
	expect(source).toContain("on:keydown");
	expect(source).toContain('event.key === "Enter"');
	expect(source).toContain("on:click={addInput}");
});

// Implements DESIGN-001 SearchView deterministic duplicate handling verification.
test("duplicate foodObjectId is rejected with a message before reaching the store", () => {
	expect(source).toContain("some((existing) => existing.foodObjectId === trimmedId)");
	expect(source).toContain("Duplicate");
});

// Implements DESIGN-001 SearchView positive quantity validation verification.
test("positive finite quantity validation guards the add path", () => {
	expect(source).toContain("Quantity must be a positive number");
	expect(source).toContain("draftQuantity <= 0");
	expect(source).toContain("Number.isFinite(draftQuantity)");
});

// Implements DESIGN-001 SearchView Substitution Input removal and update verification.
test("removal calls removeSubstitutionInput and row edits call updateSubstitutionInput", () => {
	expect(source).toContain("removeSubstitutionInput(input.foodObjectId)");
	expect(source).toContain("updateSubstitutionInput");
	expect(source).toContain("on:input={(event) => onRowQuantityInput(input.foodObjectId, event)}");
	expect(source).toContain("on:change={(event) => onRowUnitChange(input.foodObjectId, event)}");
});

// Implements DESIGN-001 SearchView Substitution Input labelled controls and traceability verification.
test("section landmark labels all controls and cites the DESIGN source", () => {
	expect(source).toContain('aria-label="Substitution inputs"');
	expect(source).toContain("<!-- Implements DESIGN-001 SearchView Substitution Input controls");
	expect(source).toContain('id="substitution-food-object-id"');
	expect(source).toContain('id="substitution-quantity"');
	expect(source).toContain('id="substitution-unit"');
});

// Implements DESIGN-001 SearchView Substitution Input focus states verification.
test("draft and row controls declare visible Tailwind focus states", () => {
	expect(countOccurrences(source, "focus:ring-2")).toBeGreaterThanOrEqual(4);
	expect(countOccurrences(source, "focus:outline-none")).toBeGreaterThanOrEqual(4);
});
