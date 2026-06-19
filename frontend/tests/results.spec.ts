import { expect, test, type Page } from "@playwright/test";
import type { SearchResponseEnvelope } from "../src/lib/api/generated";

// Implements DESIGN-001 ResultsGrid browser interaction.
//
// Task 151 wires ResultsGrid into SearchShell via SearchResults, which drives the grid from a
// TanStack Query result over the generated search envelope (results, similarity metadata,
// scores, loading, error, totalCount, page, onPageChange). These flows exercise the real
// running app against controlled search responses.

const categoryNames = ["Fruit", "Vegetable", "Dairy", "Meat"] as const;
const tiers = ["excellent", "good", "fair", "poor"] as const;

/** Builds a deterministic 12-item search response so the 10-item page cap and 2-page pagination can be verified once wired. */
function buildSearchEnvelope(): SearchResponseEnvelope {
  const items = Array.from({ length: 12 }, (_, i) => ({
    id: `food-${i + 1}`,
    name: `Food ${i + 1}`,
    physicalState: (i % 2 === 0 ? "solid" : "liquid") as "solid" | "liquid",
    imageUrl: i % 3 === 0 ? null : `https://example.test/food-${i + 1}.png`,
    classifications: [
      { id: `cat-${i % 4}`, name: categoryNames[i % 4], kind: "food_category" as const }
    ],
    primaryFoodCategory: {
      id: `cat-${i % 4}`,
      name: categoryNames[i % 4],
      kind: "food_category" as const
    },
    macros: { protein: i, carbohydrates: i * 2, fat: i / 2 },
    macroBasis: (i % 2 === 0 ? "100g" : "100ml") as "100g" | "100ml",
    calories: 50 + i * 10
  }));
  return {
    status: "ok",
    requestId: "results-scaffold-0001",
    data: {
      items,
      totalCount: 12,
      page: 1,
      similarityScores: items.map((_, i) => 1 - i * 0.05),
      similarityMetadata: items.map((item, i) => ({
        itemId: item.id,
        score: 1 - i * 0.05,
        tier: tiers[i % 4],
        imageUrl: "",
        matchingQuantity: 100
      })),
      warnings: []
    }
  };
}

/** Task 151 will reuse this stub once ResultsGrid is wired into the shell. */
async function stubSearch(page: Page): Promise<void> {
  await page.route(/\/api\/v1\/search$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(buildSearchEnvelope())
    });
  });
}

// Implements DESIGN-001 ResultsGrid 10-item page cap and required card data (scaffold for Task 151).
test("renders at most 10 result cards per page with name, macros, and similarity", async ({ page }) => {
  await stubSearch(page);
  await page.goto("/");

  const cards = page.locator("[data-result-card]");
  await expect(cards).toHaveCount(10);
  await expect(page.locator("[data-result-name]").first()).toBeVisible();
  await expect(page.locator("[data-result-macros]").first()).toBeVisible();
  await expect(page.locator("[data-result-similarity]").first()).toBeVisible();
});

// Implements DESIGN-001 ResultsGrid pagination forward/backward with disabled boundaries (scaffold for Task 151).
test("paginates forward and backward with disabled boundaries", async ({ page }) => {
  await stubSearch(page);
  await page.goto("/");

  await expect(page.locator("[data-results-prev]")).toBeDisabled();
  await expect(page.locator("[data-results-next]")).toBeEnabled();
  await page.locator("[data-results-next]").click();
  await expect(page.locator("[data-results-page]")).toHaveText("Page 2 of 2");
  await expect(page.locator("[data-results-next]")).toBeDisabled();
  await expect(page.locator("[data-results-prev]")).toBeEnabled();
  await page.locator("[data-results-prev]").click();
  await expect(page.locator("[data-results-page]")).toHaveText("Page 1 of 2");
});

// Implements DESIGN-001 ResultsGrid category placeholder for items without an image (scaffold for Task 151).
test("shows a category placeholder for items without an image", async ({ page }) => {
  await stubSearch(page);
  await page.goto("/");
  await expect(page.locator("[data-result-placeholder]").first()).toBeVisible();
});

// Implements DESIGN-001 ResultsGrid zero-result empty state (scaffold for Task 151).
test("shows the empty state when the search returns zero results", async ({ page }) => {
  await page.route(/\/api\/v1\/search$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        status: "ok",
        requestId: "results-empty-0001",
        data: {
          items: [],
          totalCount: 0,
          page: 1,
          similarityScores: [],
          similarityMetadata: [],
          warnings: []
        }
      })
    });
  });
  await page.goto("/");
  await expect(page.locator("[data-results-empty]")).toHaveText("No results found.");
});
