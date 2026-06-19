import { expect, test, type Page, type Route } from "@playwright/test";
import type { AutocompleteEnvelope, SearchResponseEnvelope } from "../src/lib/api/generated";

// Implements DESIGN-001 AutocompleteDropdown browser interaction.
//
// Task 151 wires AutocompleteDropdown into SearchShell as the search bar (`query` from
// searchStore, `onQueryInput` to setQuery, `onSelect` to add a substitution input or set the
// query). These flows exercise the real running app against controlled autocomplete responses.

const autocompleteEnvelope: AutocompleteEnvelope = {
	status: "ok",
	requestId: "autocomplete-workflow-0001",
	data: {
		items: [
			{ itemId: "food-apple", label: "Apple", exactMatch: true, levenshteinDistance: 0, length: 5, rank: 1 },
			{ itemId: "food-applesauce", label: "Applesauce", exactMatch: false, levenshteinDistance: 2, length: 10, rank: 2 },
			{ itemId: "food-snapple", label: "Snapple", exactMatch: false, levenshteinDistance: 3, length: 7, rank: 3 }
		]
	}
};

const emptySearch: SearchResponseEnvelope = {
	status: "ok",
	requestId: "autocomplete-search-empty",
	data: { items: [], totalCount: 0, page: 1, similarityScores: [], similarityMetadata: [], warnings: [] }
};

async function fulfillJson(route: Route, status: number, body: unknown): Promise<void> {
	await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

/** Stubs autocomplete and search endpoints so the wired shell renders without backend noise. */
async function stubApi(page: Page): Promise<void> {
	await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
	await page.route(/\/api\/v1\/search$/, (route) => fulfillJson(route, 200, emptySearch));
}

/** Scoped combobox locator for the autocomplete search bar (the theme select is also a combobox). */
function searchCombobox(page: Page) {
	return page.getByRole("combobox", { name: "Food search" });
}

/** Scoped option locator within the autocomplete listbox (the theme select also exposes options). */
function autocompleteOptions(page: Page) {
	return page.getByRole("listbox", { name: "Autocomplete suggestions" }).getByRole("option");
}

// Implements DESIGN-001 AutocompleteDropdown ranked display after debounce.
test("types in the search bar and verifies ranked suggestions appear after the 150ms debounce", async ({ page }) => {
	await stubApi(page);
	await page.goto("/");

	await searchCombobox(page).fill("app");

	// Suggestions appear in server rank order: Apple, Applesauce, Snapple.
	const listbox = page.getByRole("listbox", { name: "Autocomplete suggestions" });
	await expect(listbox).toBeVisible();
	await expect(autocompleteOptions(page).nth(0)).toHaveText("Apple");
	await expect(autocompleteOptions(page).nth(1)).toHaveText("Applesauce");
	await expect(autocompleteOptions(page).nth(2)).toHaveText("Snapple");
});

// Implements DESIGN-001 AutocompleteDropdown Tab/Shift+Tab focus movement.
//
// The dropdown pre-activates the first option (activeIndex 0) when results arrive, so the first
// Tab moves focus to the second option; Shift+Tab moves backward. See the follow-up note in the
// completion report about the pre-activation behavior for the Task 152 a11y gate.
test("Tab moves focus forward through options and Shift+Tab moves it backward", async ({ page }) => {
	await stubApi(page);
	await page.goto("/");
	await searchCombobox(page).fill("app");
	await page.getByRole("listbox", { name: "Autocomplete suggestions" }).waitFor();

	const input = searchCombobox(page);
	await input.focus();
	await page.keyboard.press("Tab");
	await expect(autocompleteOptions(page).nth(1)).toBeFocused();
	await page.keyboard.press("Tab");
	await expect(autocompleteOptions(page).nth(2)).toBeFocused();
	await page.keyboard.press("Shift+Tab");
	await expect(autocompleteOptions(page).nth(1)).toBeFocused();
});

// Implements DESIGN-001 AutocompleteDropdown Enter selects the active option.
test("Enter selects the active option and invokes the onSelect handler", async ({ page }) => {
	await stubApi(page);
	await page.goto("/");
	await searchCombobox(page).fill("app");
	await page.getByRole("listbox", { name: "Autocomplete suggestions" }).waitFor();

	const input = searchCombobox(page);
	await input.focus();
	// The first option is pre-active; Enter selects it without Tabbing.
	await page.keyboard.press("Enter");

	// onSelect sets the query to the selected suggestion label in Catalog mode.
	await expect(input).toHaveValue("Apple");
});

// Implements DESIGN-001 AutocompleteDropdown Escape dismissal.
test("Escape dismisses the dropdown and returns focus to the combobox", async ({ page }) => {
	await stubApi(page);
	await page.goto("/");
	await searchCombobox(page).fill("app");
	await page.getByRole("listbox", { name: "Autocomplete suggestions" }).waitFor();

	const input = searchCombobox(page);
	await input.focus();
	await page.keyboard.press("Escape");

	await expect(page.getByRole("listbox", { name: "Autocomplete suggestions" })).toBeHidden();
	await expect(input).toBeFocused();
});

// Implements DESIGN-001 AutocompleteDropdown ARIA combobox/listbox state.
test("combobox exposes aria-expanded, aria-controls, and aria-selected on the active option", async ({ page }) => {
	await stubApi(page);
	await page.goto("/");
	await searchCombobox(page).fill("app");
	await page.getByRole("listbox", { name: "Autocomplete suggestions" }).waitFor();

	const input = searchCombobox(page);
	await expect(input).toHaveAttribute("aria-expanded", "true");
	const listboxId = await input.getAttribute("aria-controls");
	expect(listboxId).toBeTruthy();
	await expect(page.locator(`#${listboxId}`)).toHaveAttribute("role", "listbox");

	// The first option is pre-active (aria-selected true) before any Tab.
	await expect(autocompleteOptions(page).nth(0)).toHaveAttribute("aria-selected", "true");
});
