import { test as base, type Page } from "@playwright/test";
import type { SavedItemsEnvelope, SearchHistoryEnvelope, SearchRequest, SearchResponse } from "../src/lib/api/generated";

type ControlledAPIFixtures = { controlledPage: Page; submittedSearchRequests: SearchRequest[] };

// Implements DESIGN-001 SearchView deterministic browser fixture.
export const test = base.extend<ControlledAPIFixtures>({
  submittedSearchRequests: async ({}, use) => { await use([]); },
  controlledPage: async ({ page, submittedSearchRequests }, use) => {
    let retryAttempts = 0;
    await page.route("**/api/v1/**", async (route) => {
      const autocomplete = route.request().url().includes("/search/autocomplete");
      if (route.request().url().includes("/search-history")) {
        const history = { status: "ok", requestId: "history", data: { history: [{ id: "history-1", query: "banana", mode: "catalog", filtersHash: "none" }] } } satisfies SearchHistoryEnvelope;
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(history) }); return;
      }
      if (route.request().url().includes("/saved-items")) {
        const favorites = { status: "ok", requestId: "favorites", data: { items: [{ id: "saved-1", itemId: "food-1", kind: "favorite" }] } } satisfies SavedItemsEnvelope;
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(favorites) }); return;
      }
      let data: Record<string, unknown> = autocomplete ? { items: [
        { itemId: "food-2", label: "Apple sauce", exactMatch: false, levenshteinDistance: 5, length: 11, rank: 1 },
        { itemId: "food-1", label: "Apple", exactMatch: true, levenshteinDistance: 0, length: 5, rank: 2 }
      ] } : {};
      if (!autocomplete && route.request().method() === "POST") {
        const body = route.request().postDataJSON() as SearchRequest;
        submittedSearchRequests.push(body);
        const mode = body.mode;
        if (mode === "daily_diet_alternative") {
          await route.fulfill({ status: 422, contentType: "application/json", body: JSON.stringify({ status: "error", requestId: "daily", data: { rejection: { code: "daily_diet_phase_07_required", message: "Daily Diet Alternative requires Phase 07 data", field: "dailyDietId" } }, error: { category: "validation", code: "daily_diet_unavailable", message: "Daily diet data unavailable until Phase 07", retryable: false } }) }); return;
        }
        if (body.query === "retry" && retryAttempts++ === 0) {
          await route.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ status: "error", requestId: "retry", error: { category: "dependency", code: "unavailable", message: "Try again", retryable: true } }) }); return;
        }
        if (body.query === "error") {
          await route.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ status: "error", requestId: "browser-error", error: { category: "dependency", code: "unavailable", message: "Search unavailable", retryable: true, requestId: "browser-error" } }) });
          return;
        }
        if (body.query === "slow" || (body.query === "delayed-page" && body.page === 2)) await new Promise((resolve) => setTimeout(resolve, 250));
        const count = body.page === 1 ? 10 : 1;
        const items = body.query === "empty" ? [] : Array.from({ length: count }, (_, index) => ({
          id: `00000000-0000-0000-0000-${String((body.page - 1) * 10 + index + 1).padStart(12, "0")}`,
          name: body.query === "delayed-page" ? `Page ${body.page} apple ${index + 1}` : `Apple ${index + 1}`,
          physicalState: "solid",
          imageUrl: index === 0 ? "/broken-image.jpg" : null,
          classifications: [{ id: "00000000-0000-0000-0000-000000000100", name: "Fruit", kind: "food_category" }],
          primaryFoodCategory: { id: "00000000-0000-0000-0000-000000000100", name: "Fruit", kind: "food_category" },
          macros: { protein: 1, carbohydrate: 20, fat: 2, basis: "100g" }, calories: 102
        }));
        data = { items, totalCount: body.query === "empty" ? 0 : 11, page: body.page, similarityScores: items.map(() => 0.9), similarityMetadata: items.length ? [{ itemId: items[0].id, score: 0.9, tier: "excellent", imageUrl: "/assets/similarity/excellent.svg", matchingQuantity: 100 }] : [], warnings: [] } satisfies SearchResponse;
      }
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ status: "ok", requestId: "browser-fixture", data }) });
    });
    await use(page);
  }
});

export { expect } from "@playwright/test";
