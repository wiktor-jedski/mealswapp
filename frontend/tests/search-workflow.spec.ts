import { expect, test, type Page, type Route } from "@playwright/test";
import type {
  AutocompleteEnvelope,
  ProfileEnvelope,
  SavedItemsEnvelope,
  SearchHistoryEnvelope,
  SearchRequest,
  SearchResponse,
  SearchResponseEnvelope,
  SearchRejectionEnvelope
} from "../src/lib/api/generated";

// Implements DESIGN-001 SearchView Phase 05 integration coverage with controlled API responses.
//
// Task 151 composes the production search shell: sidebar, mode controls, autocomplete search
// bar, mode-specific controls, filters, settings, results (TanStack Query over generated
// envelopes), cache, offline status, and theming. These tests exercise that composition with
// route interception and no handwritten contract fixtures drifting from generated types.

/** Builds a deterministic food object so result rendering and pagination can be verified. */
function foodObject(i: number): SearchResponse["items"][number] {
  return {
    id: `food-${i}`,
    name: `Food ${i}`,
    physicalState: i % 2 === 0 ? "solid" : "liquid",
    imageUrl: i % 3 === 0 ? null : `https://example.test/food-${i}.png`,
    classifications: [{ id: `cat-${i % 4}`, name: "Fruit", kind: "food_category" }],
    primaryFoodCategory: { id: `cat-${i % 4}`, name: "Fruit", kind: "food_category" },
    macros: { protein: i, carbohydrates: i * 2, fat: i / 2 },
    macroBasis: i % 2 === 0 ? "100g" : "100ml",
    calories: 50 + i * 10
  };
}

/** Builds a search response envelope with `count` items on the requested page. */
function searchEnvelope(count: number, page = 1): SearchResponseEnvelope {
  const items = Array.from({ length: count }, (_, i) => foodObject(i + 1));
  return {
    status: "ok",
    requestId: `search-${count}-${page}`,
    data: {
      items,
      totalCount: count,
      page,
      similarityScores: items.map((_, i) => 1 - i * 0.05),
      similarityMetadata: items.map((item, i) => ({
        itemId: item.id,
        score: 1 - i * 0.05,
        tier: (["excellent", "good", "fair", "poor"] as const)[i % 4],
        imageUrl: "",
        matchingQuantity: 100
      })),
      warnings: []
    }
  };
}

const autocompleteEnvelope: AutocompleteEnvelope = {
  status: "ok",
  requestId: "autocomplete-workflow-0001",
  data: {
    items: [
      { itemId: "food-apple", label: "Apple", exactMatch: true, levenshteinDistance: 0, length: 5, rank: 1 },
      { itemId: "food-applesauce", label: "Applesauce", exactMatch: false, levenshteinDistance: 2, length: 10, rank: 2 }
    ]
  }
};

const profileEnvelope: ProfileEnvelope = {
  status: "ok",
  requestId: "profile-workflow-0001",
  data: {
    userId: "user-1",
    displayName: "Test User",
    unitSystem: "metric",
    themePreference: "system",
    requiresUnitRecalculation: false
  }
};

const historyEnvelope: SearchHistoryEnvelope = {
  status: "ok",
  requestId: "history-workflow-0001",
  data: {
    history: [
      { id: "hist-1", query: "apple", mode: "catalog", filtersHash: "hash-1" },
      { id: "hist-2", query: "oats", mode: "substitution", filtersHash: "hash-2" }
    ]
  }
};

const favoritesEnvelope: SavedItemsEnvelope = {
  status: "ok",
  requestId: "favorites-workflow-0001",
  data: { items: [{ id: "fav-1", itemId: "food-apple", kind: "favorite" }] }
};

/** Fulfills a route with JSON. */
async function fulfillJson(route: Route, status: number, body: unknown): Promise<void> {
  await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

/** Stubs the core search + autocomplete endpoints so the shell renders without a backend. */
async function stubCoreApi(page: Page, search: SearchResponseEnvelope = searchEnvelope(12, 1)): Promise<void> {
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
  await page.route(/\/api\/v1\/search$/, (route) => fulfillJson(route, 200, search));
}

// Implements DESIGN-001 SearchView initial Catalog search verification.
test("initial Catalog search renders ranked results after typing a query", async ({ page }) => {
  await stubCoreApi(page, searchEnvelope(5, 1));
  await page.goto("/");

  await page.getByLabel("Food search").fill("apple");
  await expect(page.locator("[data-result-card]")).toHaveCount(5);
  await expect(page.locator("[data-result-name]").first()).toContainText("Food");
});

// Implements DESIGN-001 SearchView debounced autocomplete verification.
test("autocomplete shows ranked suggestions after the 150ms debounce", async ({ page }) => {
  await stubCoreApi(page);
  await page.goto("/");

  await page.getByLabel("Food search").fill("app");
  const listbox = page.getByRole("listbox", { name: "Autocomplete suggestions" });
  await expect(listbox).toBeVisible();
  await expect(listbox.getByRole("option").nth(0)).toHaveText("Apple");
  await expect(listbox.getByRole("option").nth(1)).toHaveText("Applesauce");
});

// Implements DESIGN-001 SearchView Substitution Input search verification.
test("Substitution Input search sends inputs and renders ranked results", async ({ page }) => {
  let seenRequestBody: SearchRequest | null = null;
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
  await page.route(/\/api\/v1\/search$/, async (route) => {
    seenRequestBody = (await route.request().postDataJSON()) as SearchRequest;
    await fulfillJson(route, 200, searchEnvelope(3, 1));
  });
  await page.goto("/");

  await page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Substitution" }).click();
  await page.locator("#substitution-food-object-id").fill("food-apple");
  await page.locator("#substitution-quantity").fill("150");
  await page.getByRole("button", { name: "Add", exact: true }).click();

  await page.getByLabel("Food search").fill("apple");
  await expect(page.locator("[data-result-card]")).toHaveCount(3);
  expect(seenRequestBody).not.toBeNull();
  expect(seenRequestBody!.mode).toBe("substitution");
  expect(seenRequestBody!.substitutionInputs?.[0]?.foodObjectId).toBe("food-apple");
  expect(seenRequestBody!.substitutionInputs?.[0]?.quantity).toBe(150);
});

// Implements DESIGN-001 SearchView Daily Diet Alternative 422 rejection verification.
test("Daily Diet Alternative search shows the structured 422 rejection", async ({ page }) => {
  const rejectionEnvelope: SearchRejectionEnvelope = {
    status: "error",
    requestId: "rejection-workflow-0001",
    data: { rejection: { code: "no_alternative_found", message: "No daily diet alternative available.", field: "dailyDietId" } },
    error: { category: "validation", code: "no_alternative_found", message: "No daily diet alternative available.", retryable: false }
  };
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
  await page.route(/\/api\/v1\/search$/, (route) => fulfillJson(route, 422, rejectionEnvelope));
  await page.goto("/");

  await page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Daily Diet Alternative" }).click();
  await page.locator("#daily-diet-id").fill("00000000-0000-0000-0000-000000000000");
  await page.getByLabel("Food search").fill("apple");

  await expect(page.locator("[data-rejection-message]")).toContainText("No daily diet alternative");
});

// Implements DESIGN-001 SearchView filter composer verification.
test("applies a filter and includes it in the search request", async ({ page }) => {
  let seenRequestBody: SearchRequest | null = null;
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
  await page.route(/\/api\/v1\/search$/, async (route) => {
    seenRequestBody = (await route.request().postDataJSON()) as SearchRequest;
    await fulfillJson(route, 200, searchEnvelope(2, 1));
  });
  await page.goto("/");

  await page.locator("#filter-id").fill("cat-fruit");
  await page.getByRole("button", { name: "Add filter", exact: true }).click();
  await page.getByLabel("Food search").fill("apple");

  await expect(page.locator("[data-result-card]")).toHaveCount(2);
  await expect(page.locator("[data-active-filters] [data-filter-id='cat-fruit']")).toBeVisible();
  expect(seenRequestBody).not.toBeNull();
  expect(seenRequestBody!.filters?.[0]?.filterId).toBe("cat-fruit");
  expect(seenRequestBody!.filters?.[0]?.kind).toBe("food_category");
});

// Implements DESIGN-001 SearchView pagination verification.
test("pagination loads page 2 and reflects the page in the request", async ({ page }) => {
  let lastPage: number | null = null;
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
  await page.route(/\/api\/v1\/search$/, async (route) => {
    const body = (await route.request().postDataJSON()) as SearchRequest;
    lastPage = body.page;
    await fulfillJson(route, 200, searchEnvelope(12, body.page));
  });
  await page.goto("/");

  await expect(page.locator("[data-results-page]")).toHaveText("Page 1 of 2");
  await page.locator("[data-results-next]").click();
  await expect(page.locator("[data-results-page]")).toHaveText("Page 2 of 2");
  expect(lastPage).toBe(2);
});

// Implements DESIGN-001 SearchView local cache reuse verification.
test("repeating the same search reuses the cache without a second network call", async ({ page }) => {
  let appleRequestCount = 0;
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
  await page.route(/\/api\/v1\/search$/, async (route) => {
    const body = (await route.request().postDataJSON()) as SearchRequest;
    if (body.query === "apple") appleRequestCount += 1;
    await fulfillJson(route, 200, searchEnvelope(4, 1));
  });
  await page.goto("/");

  await page.getByLabel("Food search").fill("apple");
  await expect(page.locator("[data-result-card]")).toHaveCount(4);
  expect(appleRequestCount).toBe(1);

  // Clear and re-enter the same query; TanStack + local cache should serve without a new apple fetch.
  await page.getByLabel("Food search").fill("");
  await page.getByLabel("Food search").fill("apple");
  await expect(page.locator("[data-result-card]")).toHaveCount(4);
  expect(appleRequestCount).toBe(1);
});

// Implements DESIGN-001 SidebarComponent authenticated history and favorites verification.
test("authenticated sidebar loads search history and favorites", async ({ page }) => {
  await page.route(/\/api\/v1\/profile$/, (route) => fulfillJson(route, 200, profileEnvelope));
  await page.route(/\/api\/v1\/search-history$/, (route) => fulfillJson(route, 200, historyEnvelope));
  await page.route(/\/api\/v1\/saved-items\?kind=favorite$/, (route) => fulfillJson(route, 200, favoritesEnvelope));
  await stubCoreApi(page);
  await page.goto("/");

  // On mobile the sidebar starts collapsed; open it via the activity toggle when present.
  if ((page.viewportSize()?.width ?? 1280) < 640) {
    await page.getByRole("button", { name: "Open activity sidebar" }).click();
  }

  await expect(page.locator("[data-sidebar-history-entry='hist-1']")).toBeVisible();
  await expect(page.locator("[data-sidebar-favorite='food-apple']")).toBeVisible();
});

// Implements DESIGN-001 OfflineBanner offline cached-result display verification.
test("going offline shows the OfflineBanner while cached results remain visible", async ({ page }) => {
  await stubCoreApi(page, searchEnvelope(6, 1));
  await page.goto("/");

  await page.getByLabel("Food search").fill("apple");
  await expect(page.locator("[data-result-card]")).toHaveCount(6);

  await page.context().setOffline(true);
  await expect(page.locator("[data-offline-banner]")).toBeVisible();
  await expect(page.locator("[data-result-card]")).toHaveCount(6);

  await page.context().setOffline(false);
});

// Implements DESIGN-001 SearchView 10-second timeout handling verification.
test("a slow search surfaces an error and a retry succeeds", async ({ page }) => {
  test.setTimeout(30_000);
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
  await page.route(/\/api\/v1\/search$/, async (route) => {
    // Exceeds the search-client 10-second timeout budget so the request is aborted.
    await new Promise((resolve) => setTimeout(resolve, 11_000));
    await fulfillJson(route, 200, searchEnvelope(2, 1));
  });
  await page.goto("/");

  await page.getByLabel("Food search").fill("apple");
  await expect(page.locator("[data-results-error]")).toBeVisible({ timeout: 15_000 });

  // Retry with a fast stub and a fresh query (a new query key forces a refetch).
  await page.unroute(/\/api\/v1\/search$/);
  await page.route(/\/api\/v1\/search$/, (route) => fulfillJson(route, 200, searchEnvelope(3, 1)));
  await page.getByLabel("Food search").fill("banana");
  await expect(page.locator("[data-result-card]")).toHaveCount(3);
});

// Implements DESIGN-001 SearchView empty-state verification.
test("a zero-result search shows the empty state", async ({ page }) => {
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
  await page.route(/\/api\/v1\/search$/, (route) => fulfillJson(route, 200, searchEnvelope(0, 1)));
  await page.goto("/");

  await page.getByLabel("Food search").fill("zzz");
  await expect(page.locator("[data-results-empty]")).toHaveText("No results found.");
});

// Implements DESIGN-016 ThemeProvider explicit selection restoration across reload verification.
test("explicit theme selection restores across a reload", async ({ page }) => {
  await stubCoreApi(page);
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/");

  await page.getByLabel("Theme preference").selectOption("dark");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");

  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.getByLabel("Theme preference")).toHaveValue("dark");
});
