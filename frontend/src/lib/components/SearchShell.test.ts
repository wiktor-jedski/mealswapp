import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-001 SearchView shell composition verification.
//
// Static-source assertions verify the Task 151 composed shell: SidebarComponent, mode controls,
// autocomplete search bar, mode-specific controls, filter composer, settings, results, and
// offline banner in the documented visual order, plus traceability. `vite build` compiles the
// full shell, validating the composed Svelte source at build time.

const source = readFileSync(join(import.meta.dir, "SearchShell.svelte"), "utf8");

function indexOf(fragment: string): number {
	return source.indexOf(fragment);
}

// Implements DESIGN-001 SearchView composed component presence verification.
test("composes sidebar, mode controls, autocomplete, mode-specific controls, settings, results, and offline banner", () => {
	expect(source).toContain("<SidebarComponent />");
	expect(source).toContain("<SearchModes />");
	expect(source).toContain("<AutocompleteDropdown");
	expect(source).toContain("<SubstitutionInputs />");
	expect(source).toContain("<DailyDietControls");
	expect(source).toContain("<SettingsPanel />");
	expect(source).toContain("<SearchResults");
	expect(source).toContain("<OfflineBanner />");
});

// Implements DESIGN-001 SearchView documented visual order verification.
test("visual order: modes → autocomplete → mode controls → settings → results → offline banner", () => {
	const modesPos = indexOf("<SearchModes />");
	const searchPos = indexOf("<AutocompleteDropdown");
	const settingsPos = indexOf("<SettingsPanel />");
	const resultsPos = indexOf("<SearchResults");
	const offlinePos = indexOf("<OfflineBanner />");
	expect(modesPos).toBeGreaterThan(-1);
	expect(searchPos).toBeGreaterThan(modesPos);
	expect(settingsPos).toBeGreaterThan(searchPos);
	expect(resultsPos).toBeGreaterThan(settingsPos);
	expect(offlinePos).toBeGreaterThan(resultsPos);
});

// Implements DESIGN-001 SearchView search bar bound to setQuery via autocomplete verification.
test("autocomplete search bar is bound to setQuery and has no disabled attribute", () => {
	expect(source).toContain("setQuery");
	expect(source).not.toContain("disabled");
});

// Implements DESIGN-001 SearchView mode-specific controls composition verification.
test("mode-specific controls render conditionally based on searchStore.mode", () => {
	expect(source).toContain('$searchStore.mode === "substitution"');
	expect(source).toContain('$searchStore.mode === "daily_diet_alternative"');
});

// Implements DESIGN-001 SearchView Daily Diet rejection wiring verification.
test("DailyDietControls receives the rejection lifted from SearchResults", () => {
	expect(source).toContain("{rejection}");
	expect(source).toContain("onRejection");
});

// Implements DESIGN-001 SearchView shell traceability verification.
test("shell cites the DESIGN source", () => {
	expect(source).toContain("<!-- Implements DESIGN-001 SearchView");
});
