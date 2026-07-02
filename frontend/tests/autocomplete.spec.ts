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

async function stubApiWithResults(page: Page): Promise<void> {
	await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
	await page.route(/\/api\/v1\/search$/, (route) =>
		fulfillJson(route, 200, {
			status: "ok",
			requestId: "autocomplete-search-results",
			data: {
				items: [
					{
						id: "food-apple",
						name: "Apple",
						physicalState: "solid",
						imageUrl: null,
						classifications: [{ id: "cat-fruit", name: "Fruit", kind: "food_category" }],
						primaryFoodCategory: { id: "cat-fruit", name: "Fruit", kind: "food_category" },
						macros: { protein: 0, carbohydrates: 14, fat: 0 },
						macroBasis: "100g",
						calories: 52
					}
				],
				totalCount: 1,
				page: 1,
				similarityScores: [],
				similarityMetadata: [],
				warnings: []
			}
		} satisfies SearchResponseEnvelope)
	);
}

/** Scoped combobox locator for the autocomplete search bar (the theme select is also a combobox). */
function searchCombobox(page: Page) {
	return page.getByRole("combobox", { name: "Food search" });
}

/** Scoped option locator within the autocomplete listbox (the theme select also exposes options). */
function autocompleteOptions(page: Page) {
	return page.getByRole("listbox", { name: "Autocomplete suggestions" }).getByRole("option");
}

async function resultGridDocumentTop(page: Page): Promise<number> {
	return page.locator("[data-results-grid]").evaluate((element) => {
		const rect = element.getBoundingClientRect();
		return rect.top + window.scrollY;
	});
}

// Verifies IT-ARCH-001-002.
// Verifies ARCH-001.
// Traces SW-REQ-002, SW-REQ-008, SW-REQ-009.
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

// Implements DESIGN-001 SearchView initial and mode-change search focus verification.
test("focuses the search bar on initial load and after mode changes", async ({ page }) => {
	await stubApi(page);
	await page.goto("/");

	const input = searchCombobox(page);
	await expect(input).toBeFocused();
	await expect(page.locator("[data-search-mode-description]")).toHaveText("Find foods, meals, or ingredients by name.");

	await page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Substitution" }).click();
	await expect(input).toBeFocused();
	await expect(input).toHaveAttribute("placeholder", "Search a food to add as a substitution target…");
	await expect(page.locator("[data-search-mode-description]")).toHaveText("Find alternatives for a food using quantity and unit context.");

	await page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Daily Diet", exact: true }).click();
	await expect(input).toBeFocused();
	await expect(input).toHaveAttribute("placeholder", "Search saved daily diets…");
	await expect(page.locator("[data-search-mode-description]")).toHaveText("Search across saved daily diets.");

	await page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Daily Diet Alternative" }).click();
	await expect(input).toBeFocused();
	await expect(input).toHaveAttribute("placeholder", "Search within a saved daily diet or paste its ID…");
	await expect(page.locator("[data-search-mode-description]")).toHaveText("Search for replacements within a saved daily diet.");

	await page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Catalog" }).click();
	await expect(input).toBeFocused();
	await expect(input).toHaveAttribute("placeholder", "Search foods, meals, or ingredients…");
	await expect(page.locator("[data-search-mode-description]")).toHaveText("Find foods, meals, or ingredients by name.");
});

// Implements DESIGN-001 SidebarComponent duplicate search-mode navigation removal verification.
test("keeps search-mode buttons only in the main view, not in the sidebar", async ({ page }) => {
	await stubApi(page);
	await page.goto("/");

	await expect(page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Catalog" })).toBeVisible();
	await expect(page.locator("[data-sidebar-modes]")).toHaveCount(0);
	await expect(page.getByRole("navigation", { name: "Search mode navigation" })).toHaveCount(0);
});

// Verifies IT-ARCH-001-002.
// Verifies ARCH-001.
// Traces SW-REQ-008.
// Implements DESIGN-001 AutocompleteDropdown floating overlay layout verification.
test("opening autocomplete suggestions does not push results down", async ({ page }) => {
	await stubApiWithResults(page);
	await page.goto("/");

	await searchCombobox(page).fill("apple");
	await searchCombobox(page).press("Enter");
	await expect(page.locator("[data-results-grid]")).toBeVisible();
	const before = await resultGridDocumentTop(page);

	await searchCombobox(page).fill("app");
	await expect(page.getByRole("listbox", { name: "Autocomplete suggestions" })).toBeVisible();
	const after = await resultGridDocumentTop(page);

	expect(after).toBeLessThanOrEqual(before);
});

// Verifies IT-ARCH-001-002.
// Verifies ARCH-001.
// Traces SW-REQ-009, SW-REQ-086.
// Implements DESIGN-001 AutocompleteDropdown Tab/Shift+Tab focus movement.
//
test("Tab moves focus forward through options and Shift+Tab moves it backward", async ({ page }) => {
	await stubApi(page);
	await page.goto("/");
	await searchCombobox(page).fill("app");
	await page.getByRole("listbox", { name: "Autocomplete suggestions" }).waitFor();

	const input = searchCombobox(page);
	await input.focus();
	await page.keyboard.press("Tab");
	await expect(autocompleteOptions(page).nth(0)).toBeFocused();
	await page.keyboard.press("Tab");
	await expect(autocompleteOptions(page).nth(1)).toBeFocused();
	await page.keyboard.press("Shift+Tab");
	await expect(autocompleteOptions(page).nth(0)).toBeFocused();
});

// Verifies IT-ARCH-001-002.
// Verifies ARCH-001.
// Traces SW-REQ-009, SW-REQ-086.
// Implements DESIGN-001 AutocompleteDropdown ArrowUp/ArrowDown option movement.
test("ArrowDown moves the active suggestion instead of moving the text caret", async ({ page }) => {
	await stubApi(page);
	await page.goto("/");
	await searchCombobox(page).fill("app");
	await page.getByRole("listbox", { name: "Autocomplete suggestions" }).waitFor();

	const input = searchCombobox(page);
	await input.focus();
	await page.keyboard.press("ArrowDown");

	await expect(input).toBeFocused();
	await expect(autocompleteOptions(page).nth(0)).toHaveAttribute("aria-selected", "true");
	await expect(autocompleteOptions(page).nth(1)).toHaveAttribute("aria-selected", "false");

	await page.keyboard.press("Enter");
	await expect(input).toHaveValue("Apple");
	await expect(page.getByRole("listbox", { name: "Autocomplete suggestions" })).toBeHidden();
});

// Implements DESIGN-001 AutocompleteDropdown Enter submits the typed query without requiring suggestion selection.
test("Enter submits the typed query without selecting the top suggestion", async ({ page }) => {
	await stubApi(page);
	await page.goto("/");
	await searchCombobox(page).fill("app");
	await page.getByRole("listbox", { name: "Autocomplete suggestions" }).waitFor();

	const input = searchCombobox(page);
	await input.focus();
	await page.keyboard.press("Enter");

	await expect(input).toHaveValue("app");
	await expect(page.getByRole("listbox", { name: "Autocomplete suggestions" })).toBeHidden();
});

// Verifies IT-ARCH-001-002.
// Verifies ARCH-001.
// Traces SW-REQ-009, SW-REQ-086.
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

// Verifies IT-ARCH-001-002.
// Verifies ARCH-001.
// Traces SW-REQ-009, SW-REQ-085, SW-REQ-086.
// Implements DESIGN-001 AutocompleteDropdown ARIA combobox/listbox state.
test("combobox exposes aria-expanded, aria-controls, and inactive suggestions before navigation", async ({ page }) => {
	await stubApi(page);
	await page.goto("/");
	await searchCombobox(page).fill("app");
	await page.getByRole("listbox", { name: "Autocomplete suggestions" }).waitFor();

	const input = searchCombobox(page);
	await expect(input).toHaveAttribute("aria-expanded", "true");
	const listboxId = await input.getAttribute("aria-controls");
	expect(listboxId).toBeTruthy();
	await expect(page.locator(`#${listboxId}`)).toHaveAttribute("role", "listbox");

	await expect(input).not.toHaveAttribute("aria-activedescendant", /.+/);
	await expect(autocompleteOptions(page).nth(0)).toHaveAttribute("aria-selected", "false");
	await expect(autocompleteOptions(page).nth(1)).toHaveAttribute("aria-selected", "false");
});
