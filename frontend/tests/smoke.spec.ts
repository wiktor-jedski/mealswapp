import { expect, test, type Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import type {
  AutocompleteEnvelope,
  SearchResponseEnvelope
} from "../src/lib/api/generated";

// Implements DESIGN-001 SearchView browser smoke harness against controlled API responses.

const searchResponse: SearchResponseEnvelope = {
  status: "ok",
  requestId: "smoke-search-0001",
  data: {
    items: [
      {
        id: "food-oat-1",
        name: "Rolled Oats",
        physicalState: "solid",
        imageUrl: null
      }
    ],
    totalCount: 1,
    page: 1,
    similarityScores: [0.96],
    similarityMetadata: [
      {
        itemId: "food-oat-1",
        score: 0.96,
        tier: "excellent",
        imageUrl: "",
        matchingQuantity: 100
      }
    ],
    warnings: []
  }
};

const autocompleteResponse: AutocompleteEnvelope = {
  status: "ok",
  requestId: "smoke-autocomplete-0001",
  data: {
    items: [
      {
        itemId: "food-oat-1",
        label: "Rolled Oats",
        exactMatch: false,
        levenshteinDistance: 0,
        length: 11,
        rank: 1
      }
    ]
  }
};

// Implements DESIGN-001 SearchView controlled API response interception.
async function stubApi(page: Page): Promise<void> {
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(autocompleteResponse)
    });
  });
  await page.route(/\/api\/v1\/search$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(searchResponse)
    });
  });
}

// Implements DESIGN-001 SearchView shell rendering smoke check.
test("search shell renders against controlled API responses", async ({ page }) => {
  await stubApi(page);
  await page.goto("/");

  await expect(page.getByLabel("Food search")).toBeVisible();
  await expect(
    page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Catalog" })
  ).toBeVisible();
});

// Implements DESIGN-001 SearchView accessibility smoke check via @axe-core/playwright.
test("axe smoke check reports no serious or critical violations", async ({ page }) => {
  await stubApi(page);
  await page.goto("/");

  const results = await new AxeBuilder({ page }).analyze();
  const serious = results.violations.filter(
    (violation) => violation.impact === "critical" || violation.impact === "serious"
  );
  const summary = serious
    .map((violation) => `${violation.id}: ${violation.description}`)
    .join("\n");
  expect(serious, summary).toEqual([]);
});
