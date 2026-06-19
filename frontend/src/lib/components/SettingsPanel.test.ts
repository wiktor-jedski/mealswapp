import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-001 SettingsPanel accessible controls verification.
//
// Bun's isolated install-cache layout breaks transitive resolution for
// `svelte/server`/`svelte/compiler` (esrap, esm-env/browser), and no DOM library
// (jsdom/happy-dom) is installed, so the component cannot be rendered in a Bun
// unit test. Instead these tests assert the component declares native focusable
// controls with visible labels and Tailwind `focus:` states, driven by typed data
// arrays. `vite build` compiles the component (it is wired into SearchShell),
// validating the Svelte source, and at runtime each `{#each}` iteration renders a
// labelled, focus-styled native control for every macro and unit entry.

const source = readFileSync(
	join(import.meta.dir, "SettingsPanel.svelte"),
	"utf8"
);

function countOccurrences(haystack: string, needle: string): number {
	return haystack.split(needle).length - 1;
}

// Implements DESIGN-001 SettingsPanel macro toggle controls verification.
test("declares three macro toggle entries and a native checkbox bound to toggleMacro", () => {
	expect(countOccurrences(source, 'id: "macro-')).toBe(3);
	expect(source).toContain('id: "macro-protein"');
	expect(source).toContain('id: "macro-carbohydrates"');
	expect(source).toContain('id: "macro-fat"');
	expect(source).toContain('type="checkbox"');
	expect(source).toContain("toggleMacro(");
	expect(source).toContain("$searchStore.enabledMacros");
	expect(source).toContain("on:change={() => toggleMacro(macro.key)}");
});

// Implements DESIGN-001 SettingsPanel visible macro labels verification.
test("macro checkbox is associated with a visible label via for/id", () => {
	expect(source).toContain("id={macro.id}");
	expect(source).toContain("for={macro.id}");
	expect(source).toContain("{macro.label}");
	expect(source).toContain('label: "Protein"');
	expect(source).toContain('label: "Carbohydrates"');
	expect(source).toContain('label: "Fat"');
});

// Implements DESIGN-001 SettingsPanel unit preference controls verification.
test("declares two unit-system entries and a native radio bound to setUnitSystem", () => {
	expect(countOccurrences(source, 'id: "unit-')).toBe(2);
	expect(source).toContain('id: "unit-metric"');
	expect(source).toContain('id: "unit-imperial"');
	expect(source).toContain('type="radio"');
	expect(source).toContain('name="unit-system"');
	expect(source).toContain("setUnitSystem(");
	expect(source).toContain("$preferencesStore.unitSystem");
	expect(source).toContain("on:change={() => setUnitSystem(unit.value)}");
});

// Implements DESIGN-001 SettingsPanel visible unit labels verification.
test("unit radio is associated with a visible label via for/id", () => {
	expect(source).toContain("id={unit.id}");
	expect(source).toContain("for={unit.id}");
	expect(source).toContain("{unit.label}");
	expect(source).toContain('label: "Metric"');
	expect(source).toContain('label: "Imperial"');
});

// Implements DESIGN-001 SettingsPanel keyboard focus state verification.
test("each rendered input block declares a visible Tailwind focus state", () => {
	expect(countOccurrences(source, "focus:ring-2")).toBe(2);
	expect(countOccurrences(source, "focus:outline-none")).toBe(2);
});

// Implements DESIGN-001 SettingsPanel section landmark and traceability verification.
test("uses a labelled section landmark and cites the DESIGN source", () => {
	expect(source).toContain('aria-label="Search settings"');
	expect(source).toContain("<!-- Implements DESIGN-001 SettingsPanel");
});
