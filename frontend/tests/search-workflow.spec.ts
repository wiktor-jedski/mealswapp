import { expect, test, type Page, type Route } from "@playwright/test";
import type {
  AutocompleteEnvelope,
  FoodObjectEnvelope,
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

/** Sets the resolved document theme through the binary sidebar switch only when needed. */
async function setResolvedTheme(page: Page, target: "light" | "dark"): Promise<void> {
  const current = await page.locator("html").getAttribute("data-theme");
  if (current !== target) {
    const toggle = page.getByLabel("Theme preference");
    const openedSidebar = !(await toggle.isVisible());
    if (openedSidebar) {
      await page.getByLabel("Open activity sidebar").click();
    }
    await toggle.click();
    if (openedSidebar) {
      await page.getByLabel("Close activity sidebar").click();
    }
  }
}

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

const hydratedFoodObjectEnvelope: FoodObjectEnvelope = {
  status: "ok",
  requestId: "food-object-workflow-0001",
  data: {
    id: "food-apple",
    name: "Apple",
    physicalState: "solid",
    imageUrl: null,
    classifications: [{ id: "cat-fruit", name: "Fruit", kind: "food_category" }],
    primaryFoodCategory: { id: "cat-fruit", name: "Fruit", kind: "food_category" },
    macros: { protein: 1, carbohydrates: 14, fat: 0.2 },
    macroBasis: "100g",
    calories: 62
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

const entitlementEnvelope = {
  status: "ok",
  requestId: "entitlements-workflow-0001",
  data: {
    tier: "paid",
    allowedModes: ["catalog", "substitution", "substitution:multi", "daily_diet_alternative"],
    searchLimitPer24h: 100,
    usageRemaining: 100
  }
};

/** Fulfills a route with JSON. */
async function fulfillJson(route: Route, status: number, body: unknown): Promise<void> {
  await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

test.beforeEach(async ({ page }) => {
  await page.route(/\/api\/v1\/entitlements$/, (route) => fulfillJson(route, 200, entitlementEnvelope));
});

/** Stubs the core search + autocomplete endpoints so the shell renders without a backend. */
async function stubCoreApi(page: Page, search: SearchResponseEnvelope = searchEnvelope(12, 1)): Promise<void> {
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
  await page.route(/\/api\/v1\/food-objects\/[^/]+$/, (route) => fulfillJson(route, 200, hydratedFoodObjectEnvelope));
  await page.route(/\/api\/v1\/search$/, (route) => fulfillJson(route, 200, search));
}

// Verifies IT-ARCH-001-001.
// Verifies ARCH-001.
// Traces SW-REQ-001, SW-REQ-010, SW-REQ-011.
// Implements DESIGN-001 SearchView initial empty-results suppression verification.
test("initial Catalog view hides search results until the user enters a query", async ({ page }) => {
  await stubCoreApi(page, searchEnvelope(5, 1));
  await page.goto("/");

  await expect(page.locator("[data-results-grid]")).toHaveCount(0);
  await page.getByLabel("Food search").fill("apple");
  await page.getByLabel("Food search").press("Enter");
  await expect(page.locator("[data-results-grid]")).toBeVisible();
});

// Verifies IT-ARCH-001-001.
// Verifies ARCH-001.
// Traces SW-REQ-001, SW-REQ-010, SW-REQ-011, SW-REQ-012.
// Implements DESIGN-001 SearchView initial Catalog search verification.
test("initial Catalog search renders ranked results after typing a query", async ({ page }) => {
  await stubCoreApi(page, searchEnvelope(5, 1));
  await page.goto("/");

  await page.getByLabel("Food search").fill("apple");
  await page.getByLabel("Food search").press("Enter");
  await expect(page.locator("[data-result-card]")).toHaveCount(5);
  await expect(page.locator("[data-result-name]").first()).toContainText("Food");
});

// Verifies IT-ARCH-001-002.
// Verifies ARCH-001.
// Traces SW-REQ-002, SW-REQ-008, SW-REQ-009.
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

// Verifies IT-ARCH-001-001.
// Verifies ARCH-001.
// Traces SW-REQ-001, SW-REQ-010, SW-REQ-011.
// Implements DESIGN-001 SearchView committed server-side search verification.
test("typing a query waits for Enter before sending the final search text", async ({ page }) => {
  let searchRequestCount = 0;
  let lastQuery = "";
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
  await page.route(/\/api\/v1\/search$/, async (route) => {
    searchRequestCount += 1;
    const body = (await route.request().postDataJSON()) as SearchRequest;
    lastQuery = body.query;
    await fulfillJson(route, 200, searchEnvelope(1, body.page));
  });
  await page.goto("/");

  await page.getByLabel("Food search").pressSequentially("apple", { delay: 25 });
  expect(searchRequestCount).toBe(0);
  await page.getByLabel("Food search").press("Enter");
  await expect(page.locator("[data-result-card]")).toHaveCount(1);
  expect(searchRequestCount).toBe(1);
  expect(lastQuery).toBe("apple");
});

// Implements DESIGN-001 SearchView artifact-free loading verification.
test("typing a zero-result query does not flash skeleton result rows", async ({ page }) => {
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
  await page.route(/\/api\/v1\/search$/, async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 300));
    await fulfillJson(route, 200, searchEnvelope(0, 1));
  });
  await page.goto("/");

  await page.getByLabel("Food search").pressSequentially("zzzz", { delay: 25 });
  await page.waitForTimeout(225);
  await expect(page.locator("[data-results-grid]")).toHaveCount(0);
  await page.getByLabel("Food search").press("Enter");
  await expect(page.locator("[data-results-skeletons]")).toHaveCount(0);
  await expect(page.locator("[data-result-skeleton]")).toHaveCount(0);
  await expect(page.locator("[data-results-empty]")).toHaveText("No results found.");
});

// Verifies IT-ARCH-001-003.
// Verifies ARCH-001.
// Traces SW-REQ-005, SW-REQ-007, SW-REQ-011, SW-REQ-018, SW-REQ-025.
// Implements DESIGN-001 SearchView Substitution Input search verification.
test("Substitution Input search sends inputs and renders ranked results", async ({ page }) => {
  let seenRequestBody: SearchRequest | null = null;
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
  await page.route(/\/api\/v1\/food-objects\/[^/]+$/, (route) => fulfillJson(route, 200, hydratedFoodObjectEnvelope));
  await page.route(/\/api\/v1\/search$/, async (route) => {
    seenRequestBody = (await route.request().postDataJSON()) as SearchRequest;
    await fulfillJson(route, 200, searchEnvelope(3, 1));
  });
  await page.goto("/");

  await page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Substitution" }).click();
  await expect(page.locator("[data-results-grid]")).toHaveCount(0);
  await expect(page.locator("#substitution-food-object-id")).toHaveCount(0);

  await page.getByLabel("Food search").fill("apple");
  await page.getByRole("listbox", { name: "Autocomplete suggestions" }).getByRole("option", { name: "Apple", exact: true }).click();
  await expect(page.locator("[data-substitution-card]")).toHaveCount(1);
  await expect(page.locator("[data-substitution-macros]")).toContainText("Protein");
  await expect(page.locator("[data-substitution-calories]")).toContainText("62 kcal");
  await expect(page.locator("[data-substitution-macro-basis]")).toHaveText("values per 100 g");
  await expect(page.locator("[data-substitution-categories]")).toContainText("Fruit");
  await expect(page.locator("[data-substitution-controls]")).toBeVisible();
  await expect(page.locator("[data-food-object-id='food-apple']")).toHaveText("Apple");
  await expect(page.locator("[data-result-card]")).toHaveCount(0);

  await page.locator("#qty-food-apple").fill("150");
  await page.getByRole("button", { name: "Find substitutions" }).click();
  await expect(page.locator("[data-result-card]")).toHaveCount(3);
  await expect(page.locator("[data-result-similarity]").first()).toBeVisible();
  expect(seenRequestBody).not.toBeNull();
  expect(seenRequestBody!.mode).toBe("substitution");
  expect(seenRequestBody!.query).toBe("");
  expect(seenRequestBody!.substitutionInputs?.[0]?.foodObjectId).toBe("food-apple");
  expect(seenRequestBody!.substitutionInputs?.[0]?.quantity).toBe(150);
});

// Verifies IT-ARCH-001-003.
// Verifies ARCH-001.
// Traces SW-REQ-005, SW-REQ-007, SW-REQ-011, SW-REQ-025.
// Implements DESIGN-001 SearchView Catalog-to-Substitution selected item verification.
test("Catalog results can add full item data to the substitution input list", async ({ page }) => {
  const seenRequests: SearchRequest[] = [];
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
  await page.route(/\/api\/v1\/search$/, async (route) => {
    const body = (await route.request().postDataJSON()) as SearchRequest;
    seenRequests.push(body);
    await fulfillJson(route, 200, searchEnvelope(3, body.page));
  });
  await page.goto("/");

  await page.getByLabel("Food search").fill("apple");
  await page.getByLabel("Food search").press("Enter");
  await expect(page.locator("[data-result-card]")).toHaveCount(3);
  await page.locator("[data-result-add-substitution]").first().click();

  await page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Substitution" }).click();
  await expect(page.locator("[data-results-grid]")).toHaveCount(0);
  await expect(page.locator("[data-substitution-card]")).toHaveCount(1);
  await expect(page.locator("[data-food-object-id='food-1']")).toHaveText("Food 1");
  await expect(page.locator("[data-substitution-macros]")).toContainText("Protein");
  await expect(page.locator("[data-substitution-calories]")).toContainText("60 kcal");
  await expect(page.locator("[data-substitution-macro-basis]")).toHaveText("values per 100 ml");
  await expect(page.locator("[data-substitution-categories]")).toContainText("Fruit");
  await expect(page.locator("#unit-food-1")).toHaveValue("ml");

  await page.getByRole("button", { name: "Find substitutions" }).click();
  await expect(page.locator("[data-result-card]")).toHaveCount(3);
  const substitutionRequest = seenRequests.find((request) => request.mode === "substitution");
  expect(substitutionRequest?.substitutionInputs?.[0]).toEqual({
    foodObjectId: "food-1",
    quantity: 100,
    unit: "ml"
  });
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
  await page.getByLabel("Food search").press("Enter");

  await expect(page.locator("[data-rejection-message]")).toContainText("No daily diet alternative");
});

// Verifies IT-ARCH-001-001.
// Verifies ARCH-001.
// Traces SW-REQ-010, SW-REQ-011.
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

  await page.getByLabel("Food search").fill("apple");
  await page.getByLabel("Food search").press("Enter");
  await expect(page.locator("[data-results-page]")).toHaveText("Page 1 of 2");
  await page.locator("[data-results-next]").click();
  await expect(page.locator("[data-results-page]")).toHaveText("Page 2 of 2");
  await expect.poll(() => lastPage).toBe(2);
});

// Verifies IT-ARCH-001-004.
// Verifies ARCH-001.
// Traces SW-REQ-003, SW-REQ-088.
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
  await page.getByLabel("Food search").press("Enter");
  await expect(page.locator("[data-result-card]")).toHaveCount(4);
  expect(appleRequestCount).toBe(1);

  // Clear and re-enter the same query; TanStack + local cache should serve without a new apple fetch.
  await page.getByLabel("Food search").fill("");
  await page.getByLabel("Food search").fill("apple");
  await page.getByLabel("Food search").press("Enter");
  await expect(page.locator("[data-result-card]")).toHaveCount(4);
  expect(appleRequestCount).toBe(1);
});

// Verifies IT-ARCH-001-005.
// Verifies ARCH-001.
// Traces SW-REQ-013, SW-REQ-048.
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

// Verifies IT-ARCH-001-005.
// Verifies ARCH-001.
// Traces SW-REQ-013.
// Implements DESIGN-001 SidebarComponent account-level unit preference verification.
test("sidebar unit preference changes between metric and imperial", async ({ page }) => {
  await stubCoreApi(page);
  await page.goto("/");

  const openedSidebar = !(await page.locator("#sidebar-unit-system").isVisible());
  if (openedSidebar) {
    await page.getByRole("button", { name: "Open activity sidebar" }).click();
  }

  const units = page.locator("#sidebar-unit-system");
  await expect(units).toHaveValue("metric");
  await units.selectOption("imperial");
  await expect(units).toHaveValue("imperial");
});

// Verifies IT-ARCH-001-004.
// Verifies ARCH-001.
// Traces SW-REQ-003, SW-REQ-087, SW-REQ-088.
// Implements DESIGN-001 OfflineBanner offline cached-result display verification.
test("going offline shows the OfflineBanner while cached results remain visible", async ({ page }) => {
  await stubCoreApi(page, searchEnvelope(6, 1));
  await page.goto("/");

  await page.getByLabel("Food search").fill("apple");
  await page.getByLabel("Food search").press("Enter");
  await expect(page.locator("[data-result-card]")).toHaveCount(6);

  await page.context().setOffline(true);
  await expect(page.locator("[data-offline-banner]")).toBeVisible();
  await expect(page.locator("[data-result-card]")).toHaveCount(6);

  await page.context().setOffline(false);
});

// Verifies IT-ARCH-001-004.
// Verifies ARCH-001.
// Traces SW-REQ-077, SW-REQ-087.
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
  await page.getByLabel("Food search").press("Enter");
  await expect(page.locator("[data-results-error]")).toBeVisible({ timeout: 15_000 });

  // Retry with a fast stub and a fresh query (a new query key forces a refetch).
  await page.unroute(/\/api\/v1\/search$/);
  await page.route(/\/api\/v1\/search$/, (route) => fulfillJson(route, 200, searchEnvelope(3, 1)));
  await page.getByLabel("Food search").fill("banana");
  await page.getByLabel("Food search").press("Enter");
  await expect(page.locator("[data-result-card]")).toHaveCount(3);
});

// Implements DESIGN-001 SearchView empty-state verification.
test("a zero-result search shows the empty state", async ({ page }) => {
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope));
  await page.route(/\/api\/v1\/search$/, (route) => fulfillJson(route, 200, searchEnvelope(0, 1)));
  await page.goto("/");

  await page.getByLabel("Food search").fill("zzz");
  await page.getByLabel("Food search").press("Enter");
  await expect(page.locator("[data-results-empty]")).toHaveText("No results found.");
});

// Verifies IT-ARCH-001-005.
// Verifies ARCH-001.
// Traces SW-REQ-015.
// Implements DESIGN-016 ThemeProvider explicit selection restoration across reload verification.
test("explicit theme selection restores across a reload", async ({ page }) => {
  await stubCoreApi(page);
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/");

  await setResolvedTheme(page, "dark");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");

  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.getByLabel("Theme preference")).toHaveAttribute("aria-pressed", "true");
});
