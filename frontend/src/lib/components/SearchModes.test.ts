import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-001 SearchView mode controls static-source verification.
//
// No DOM library (jsdom/happy-dom) is installed and Bun's isolated install-cache layout
// breaks `svelte/server` rendering, so the component is not rendered in a Bun unit test.
// These tests assert the Svelte source declares the three mode controls bound to `setMode`,
// with `aria-pressed` reflecting `$searchStore.mode`, visible labels, focus states, and a
// traceability comment. `vite build` compiles the component (it is wired into SearchShell),
// validating the Svelte source at build time.

const source = readFileSync(join(import.meta.dir, "SearchModes.svelte"), "utf8");

function countOccurrences(haystack: string, needle: string): number {
	return haystack.split(needle).length - 1;
}

// Implements DESIGN-001 SearchView mode option set verification.
test("declares Catalog, Substitution, and Daily Diet Alternative mode options", () => {
	expect(countOccurrences(source, 'id: "search-mode-')).toBe(3);
	expect(source).toContain('id: "search-mode-catalog"');
	expect(source).toContain('id: "search-mode-substitution"');
	expect(source).toContain('id: "search-mode-daily-diet"');
	expect(source).toContain('value: "catalog"');
	expect(source).toContain('value: "substitution"');
	expect(source).toContain('value: "daily_diet_alternative"');
	expect(source).toContain('label: "Catalog"');
	expect(source).toContain('label: "Substitution"');
	expect(source).toContain('label: "Daily Diet Alternative"');
});

// Implements DESIGN-001 SearchView mode selection binding verification.
test("mode buttons call setMode and reflect active state via aria-pressed and $searchStore.mode", () => {
	expect(source).toContain("setMode");
	expect(source).toContain("$searchStore.mode");
	expect(source).toContain("aria-pressed");
	expect(source).toContain("on:click={() => setMode(option.value)}");
});

// Implements DESIGN-001 SearchView mode controls landmark and traceability verification.
test("uses a labelled nav landmark and cites the DESIGN source", () => {
	expect(source).toContain('aria-label="Search modes"');
	expect(source).toContain("<!-- Implements DESIGN-001 SearchView mode controls");
});

// Implements DESIGN-001 SearchView mode controls keyboard focus verification.
test("the mode button template declares a visible Tailwind focus state rendered for each option", () => {
	expect(source).toContain("focus:ring-2");
	expect(source).toContain("focus:outline-none");
});
