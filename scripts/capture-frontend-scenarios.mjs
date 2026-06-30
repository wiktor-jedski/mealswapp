#!/usr/bin/env node

// Implements DESIGN-001 SearchView scenario screenshot capture for frontend UAT reports.

import { chromium } from "../frontend/node_modules/playwright/index.mjs";
import { mkdir } from "node:fs/promises";
import { join } from "node:path";

const [, , baseUrl, artifactDir, screenshotStem] = process.argv;

if (!baseUrl || !artifactDir || !screenshotStem) {
  console.error("Usage: capture-frontend-scenarios.mjs <base-url> <artifact-dir> <screenshot-stem>");
  process.exit(2);
}

const viewports = [
  { key: "desktop", label: "Desktop", width: 1280, height: 900 },
  { key: "mobile", label: "Mobile", width: 390, height: 844 }
];

const scenarios = [
  {
    key: "catalog-autocomplete-mil",
    label: "Catalog Autocomplete - mil",
    run: async (page) => {
      await page.goto(baseUrl);
      await page.getByLabel("Food search").fill("mil");
      await page.getByRole("listbox", { name: "Autocomplete suggestions" }).waitFor({ state: "visible" });
      await page.getByRole("option", { name: "Cow Milk" }).waitFor({ state: "visible" });
    }
  },
  {
    key: "catalog-cow-milk",
    label: "Catalog Search - Cow Milk",
    run: async (page) => {
      await page.goto(baseUrl);
      await page.getByLabel("Food search").fill("Cow Milk");
      await page.getByLabel("Food search").press("Enter");
      await page.locator("[data-result-card]").first().waitFor({ state: "visible" });
    }
  },
  {
    key: "substitution-empty",
    label: "Substitution View",
    run: async (page) => {
      await page.goto(baseUrl);
      await page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Substitution" }).click();
      await page.getByLabel("Substitution inputs").waitFor({ state: "visible" });
    }
  },
  {
    key: "substitution-apple-oat-milk",
    label: "Substitution Search - Apple + Oat Milk",
    run: async (page) => {
      await page.goto(baseUrl);
      await page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Substitution" }).click();
      await addSubstitutionInput(page, "Apple");
      await addSubstitutionInput(page, "Oat Milk");
      await page.getByRole("button", { name: "Find substitutions" }).click();
      await page.locator("[data-result-card]").first().waitFor({ state: "visible" });
      await page.locator("[data-result-similarity]").first().waitFor({ state: "visible" });
    }
  }
];

await mkdir(artifactDir, { recursive: true });

const browser = await chromium.launch({ headless: true });
try {
  for (const viewport of viewports) {
    for (const scenario of scenarios) {
      const page = await newScenarioPage(browser, viewport);
      try {
        await scenario.run(page);
        await page.screenshot({
          path: join(artifactDir, `${screenshotStem}-${scenario.key}-${viewport.key}.png`),
          fullPage: true
        });
        console.log(`Captured ${scenario.label} (${viewport.label})`);
      } finally {
        await page.close();
      }
    }
  }
} finally {
  await browser.close();
}

async function addSubstitutionInput(page, label) {
  await page.getByLabel("Food search").fill(label);
  await page.getByRole("listbox", { name: "Autocomplete suggestions" }).waitFor({ state: "visible" });
  await page.getByRole("option", { name: label, exact: true }).click();
  await page.locator(`[data-food-object-id="${foodObjectId(label)}"]`).waitFor({ state: "visible" });
}

async function newScenarioPage(browser, viewport) {
  const context = await browser.newContext({
    viewport: { width: viewport.width, height: viewport.height },
    deviceScaleFactor: 1
  });
  const page = await context.newPage();
  await installRoutes(page);
  page.on("console", (message) => {
    if (message.type() === "error") {
      console.error(`browser console error: ${message.text()}`);
    }
  });
  page.on("pageerror", (error) => {
    console.error(`browser page error: ${error.message}`);
  });
  const close = page.close.bind(page);
  page.close = async (...args) => {
    await close(...args);
    await context.close();
  };
  return page;
}

async function installRoutes(page) {
  await page.route(/\/api\/v1\/profile$/, (route) => fulfillJson(route, 200, {
    status: "ok",
    requestId: "profile-screenshot-0001",
    data: {
      userId: "user-screenshot",
      displayName: "Screenshot User",
      unitSystem: "metric",
      themePreference: "system",
      requiresUnitRecalculation: false
    }
  }));
  await page.route(/\/api\/v1\/search-history$/, (route) => fulfillJson(route, 200, {
    status: "ok",
    requestId: "history-screenshot-0001",
    data: { history: [] }
  }));
  await page.route(/\/api\/v1\/saved-items(\?.*)?$/, (route) => fulfillJson(route, 200, {
    status: "ok",
    requestId: "favorites-screenshot-0001",
    data: { items: [] }
  }));
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => {
    const url = new URL(route.request().url());
    const query = url.searchParams.get("query")?.toLowerCase() ?? "";
    return fulfillJson(route, 200, autocompleteEnvelope(query));
  });
  await page.route(/\/api\/v1\/food-objects\/[^/]+$/, (route) => {
    const id = route.request().url().split("/").pop() ?? "";
    return fulfillJson(route, 200, {
      status: "ok",
      requestId: `food-object-${id}`,
      data: foodObjectById(id)
    });
  });
  await page.route(/\/api\/v1\/search$/, async (route) => {
    const body = await route.request().postDataJSON();
    const response = body.mode === "substitution"
      ? substitutionSearchEnvelope()
      : catalogSearchEnvelope(body.query);
    return fulfillJson(route, 200, response);
  });
}

function autocompleteEnvelope(query) {
  const catalogItems = [
    autocompleteItem("food-cow-milk", "Cow Milk", query === "cow milk", 1),
    autocompleteItem("food-oat-milk", "Oat Milk", query === "oat milk", 2),
    autocompleteItem("food-almond-milk", "Almond Milk", false, 3)
  ];
  const appleItems = [
    autocompleteItem("food-apple", "Apple", query === "apple", 1),
    autocompleteItem("food-applesauce", "Applesauce", false, 2)
  ];
  const items = query.includes("app")
    ? appleItems
    : query.includes("oat")
      ? [autocompleteItem("food-oat-milk", "Oat Milk", query === "oat milk", 1), autocompleteItem("food-cow-milk", "Cow Milk", false, 2)]
      : catalogItems;
  return {
    status: "ok",
    requestId: "autocomplete-screenshot-0001",
    data: { items }
  };
}

function autocompleteItem(itemId, label, exactMatch, rank) {
  return {
    itemId,
    label,
    exactMatch,
    levenshteinDistance: exactMatch ? 0 : rank,
    length: label.length,
    rank
  };
}

function catalogSearchEnvelope(query) {
  const items = query.toLowerCase().includes("cow milk")
    ? [
      foodObject("food-cow-milk", "Cow Milk", "liquid", "Dairy", { protein: 3.2, carbohydrates: 4.8, fat: 3.3 }, "100ml", 61),
      foodObject("food-skim-milk", "Skim Milk", "liquid", "Dairy", { protein: 3.4, carbohydrates: 5, fat: 0.1 }, "100ml", 34),
      foodObject("food-oat-milk", "Oat Milk", "liquid", "Plant milk", { protein: 1, carbohydrates: 6.7, fat: 1.5 }, "100ml", 43)
    ]
    : [
      foodObject("food-cow-milk", "Cow Milk", "liquid", "Dairy", { protein: 3.2, carbohydrates: 4.8, fat: 3.3 }, "100ml", 61)
    ];
  return searchEnvelope("catalog-search-screenshot-0001", items, []);
}

function substitutionSearchEnvelope() {
  const items = [
    foodObject("food-almond-milk", "Almond Milk", "liquid", "Plant milk", { protein: 0.6, carbohydrates: 0.3, fat: 1.2 }, "100ml", 15),
    foodObject("food-soy-milk", "Soy Milk", "liquid", "Plant milk", { protein: 3.3, carbohydrates: 2.7, fat: 1.8 }, "100ml", 45),
    foodObject("food-pear", "Pear", "solid", "Fruit", { protein: 0.4, carbohydrates: 15, fat: 0.1 }, "100g", 57)
  ];
  const similarityMetadata = items.map((item, index) => ({
    itemId: item.id,
    score: [0.93, 0.86, 0.72][index],
    tier: ["excellent", "good", "fair"][index],
    imageUrl: "",
    matchingQuantity: [180, 150, 120][index]
  }));
  return searchEnvelope("substitution-search-screenshot-0001", items, similarityMetadata, {
    totalGrams: 100,
    totalMilliliters: 100,
    macros: { protein: 1.2, carbohydrates: 19.8, fat: 1.7 },
    calories: 105
  });
}

function searchEnvelope(requestId, items, similarityMetadata, sourceSummary = undefined) {
  return {
    status: "ok",
    requestId,
    data: {
      items,
      totalCount: items.length,
      page: 1,
      similarityScores: similarityMetadata.map((item) => item.score),
      similarityMetadata,
      sourceSummary,
      warnings: [],
      cache: {
        status: "miss",
        namespace: "search",
        schemaVersion: "search-response-v1",
        ttlSeconds: 300
      }
    }
  };
}

function foodObjectById(id) {
  if (id === "food-apple") {
    return foodObject(id, "Apple", "solid", "Fruit", { protein: 0.3, carbohydrates: 14, fat: 0.2 }, "100g", 52);
  }
  if (id === "food-oat-milk") {
    return foodObject(id, "Oat Milk", "liquid", "Plant milk", { protein: 1, carbohydrates: 6.7, fat: 1.5 }, "100ml", 43);
  }
  if (id === "food-cow-milk") {
    return foodObject(id, "Cow Milk", "liquid", "Dairy", { protein: 3.2, carbohydrates: 4.8, fat: 3.3 }, "100ml", 61);
  }
  return foodObject(id, "Food Object", "solid", "Pantry", { protein: 1, carbohydrates: 1, fat: 1 }, "100g", 20);
}

function foodObjectId(label) {
  return `food-${label.toLowerCase().replaceAll(" ", "-")}`;
}

function foodObject(id, name, physicalState, category, macros, macroBasis, calories) {
  const categoryId = `cat-${category.toLowerCase().replaceAll(" ", "-")}`;
  const categorySummary = { id: categoryId, name: category, kind: "food_category" };
  return {
    id,
    name,
    physicalState,
    imageUrl: null,
    classifications: [categorySummary],
    primaryFoodCategory: categorySummary,
    macros,
    macroBasis,
    calories
  };
}

async function fulfillJson(route, status, body) {
  await route.fulfill({
    status,
    contentType: "application/json",
    body: JSON.stringify(body)
  });
}
