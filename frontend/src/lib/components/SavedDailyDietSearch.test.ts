import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";

const source = readFileSync(new URL("./SavedDailyDietSearch.svelte", import.meta.url), "utf8");

// Implements DESIGN-001 AutocompleteDropdown saved Daily Diet lookup verification.
test("saved Daily Diet lookup filters by name and highlights the first match", () => {
	expect(source).toContain('placeholder="Search saved Daily Diets by name…"');
	expect(source).toContain("diet.name.toLocaleLowerCase().includes(needle)");
	expect(source).toContain("let activeIndex = $state(0)");
	expect(source).toContain('event.key === "Enter"');
	expect(source).toContain("selectDiet(matches[activeIndex] ?? matches[0])");
	expect(source).toContain('role="combobox"');
	expect(source).toContain('role="listbox"');
});
