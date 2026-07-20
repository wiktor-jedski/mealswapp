#!/usr/bin/env node

// Implements DESIGN-001 SearchView and DESIGN-018 AuthView scenario screenshot capture for frontend UAT reports.

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

const TASK_233_USER_ID = "00000000-0000-0000-0000-000000000233";
const TASK_233_DIET_ID = "00000000-0000-0000-0000-000000000235";
const TASK_233_JOB_ID = "00000000-0000-0000-0000-000000000237";
const TASK_233_MEAL_ID = "00000000-0000-0000-0000-000000000238";
const TASK_233_SECOND_MEAL_ID = "00000000-0000-0000-0000-000000000239";
const TASK_233_ENTRY_ID = "00000000-0000-0000-0000-000000000240";
const TASK_233_SECOND_ENTRY_ID = "00000000-0000-0000-0000-000000000241";

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
  },
  {
    key: "auth-login",
    label: "Auth Login View",
    authState: "anonymous",
    run: async (page) => {
      await page.goto(baseUrl);
      await clickSignIn(page);
      await page.getByRole("dialog", { name: "Sign in" }).waitFor({ state: "visible" });
      await page.getByLabel("Email").fill("screenshot@example.com");
      await page.getByLabel("Password").fill("correct horse battery staple");
    }
  },
  {
    key: "auth-register",
    label: "Auth Registration View",
    authState: "anonymous",
    run: async (page) => {
      await page.goto(baseUrl);
      await clickSignIn(page);
      await page.getByRole("group", { name: "Authentication mode" }).getByRole("button", { name: "Create account" }).click();
      await page.locator("[data-register-view]").waitFor({ state: "visible" });
      await page.getByLabel("Email").fill("screenshot@example.com");
      await page.getByLabel("Password", { exact: true }).fill("CorrectHorseBatteryStaple1!");
      await page.getByLabel("Confirm password").fill("CorrectHorseBatteryStaple1!");
      await page.getByRole("checkbox", { name: /I accept the current Privacy Policy and Terms of Service\./ }).check();
    }
  },
  {
    key: "authenticated-subscription",
    label: "Authenticated Subscription View",
    authState: "authenticated",
    run: async (page) => {
      await page.goto(baseUrl);
      await page.locator("[data-sidebar-sign-out]").waitFor({ state: "attached" });
      await openMobileSidebarIfNeeded(page);
      await page.getByRole("navigation", { name: "Account navigation" }).getByRole("button", { name: "Subscription" }).click();
      await page.locator("[data-subscription-view] [data-subscription-billing]").waitFor({ state: "visible" });
    }
  },
  {
    key: "task-233-daily-diet-light",
    label: "Task 233 Daily Diet - Light",
    authState: "authenticated",
    run: async (page) => {
      await page.goto(baseUrl);
      await setTheme(page, "light");
      await page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Daily Diet", exact: true }).click();
      await page.locator(`[data-saved-daily-diet="${TASK_233_DIET_ID}"]`).waitFor({ state: "visible" });
      await assertTask233SafeState(page);
    }
  },
  {
    key: "task-233-optimization-dark",
    label: "Task 233 Optimization - Dark",
    authState: "authenticated",
    run: async (page) => {
      await page.goto(baseUrl);
      await setTheme(page, "dark");
      await page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Daily Diet Alternative", exact: true }).click();
      await page.getByRole("radio", { name: "Use Task 233 training day as Daily Diet Alternative input" }).click();
      await page.getByRole("button", { name: "Generate alternatives" }).click();
      await page.locator("[data-optimization-alternative]").waitFor({ state: "visible" });
      await assertTask233SafeState(page);
    }
  }
];

assertTask233FixtureSafe();
await mkdir(artifactDir, { recursive: true });

const browser = await chromium.launch({ headless: true });
try {
  for (const viewport of viewports) {
    for (const scenario of scenarios) {
      const page = await newScenarioPage(browser, viewport, scenario.authState);
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

async function newScenarioPage(browser, viewport, authState = "authenticated") {
  const context = await browser.newContext({
    viewport: { width: viewport.width, height: viewport.height },
    deviceScaleFactor: 1
  });
  const page = await context.newPage();
  page.on("console", (message) => {
    if (message.type() === "error") {
      console.error(`browser console error: ${message.text()}`);
    }
  });
  page.on("pageerror", (error) => {
    console.error(`browser page error: ${error.message}`);
  });
  await installRoutes(page, authState);
  const close = page.close.bind(page);
  page.close = async (...args) => {
    await close(...args);
    await context.close();
  };
  return page;
}

async function installRoutes(page, authState) {
  await page.route(/\/api\/v1\/profile$/, (route) => {
    if (authState !== "authenticated") {
      return fulfillJson(route, 401, authErrorEnvelope("profile-screenshot-anonymous", "invalid_credentials", "Not signed in."));
    }
    return fulfillJson(route, 200, {
      status: "ok",
      requestId: "profile-screenshot-0001",
      data: {
        userId: TASK_233_USER_ID,
        displayName: "Screenshot User",
        unitSystem: "metric",
        themePreference: "system",
        requiresUnitRecalculation: false
      }
    });
  });
  await page.route(/\/api\/v1\/auth\/refresh$/, (route) => {
    if (authState !== "authenticated") {
      return fulfillJson(route, 401, authErrorEnvelope("refresh-screenshot-anonymous", "invalid_credentials", "Not signed in."));
    }
    return fulfillJson(route, 200, authSessionEnvelope());
  });
  await page.route(/\/api\/v1\/auth\/csrf-token$/, (route) => fulfillJson(route, 200, {
    status: "ok",
    requestId: "csrf-screenshot-0001",
    data: { csrfToken: "csrf-screenshot" }
  }));
  await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => fulfillJson(route, 200, entitlementEnvelope()));
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
  await page.route(/\/api\/v1\/daily-diets$/, (route) => fulfillJson(route, 200, {
    status: "ok",
    requestId: "task-233-screenshot-diets",
    data: { diets: [task233Diet()] }
  }));
  await page.route(/\/api\/v1\/optimization\/jobs$/, (route) => fulfillJson(route, 202, {
    status: "accepted",
    requestId: "task-233-screenshot-accepted",
    data: { jobId: TASK_233_JOB_ID, status: "queued", pollUrl: `/api/v1/optimization/jobs/${TASK_233_JOB_ID}` }
  }));
  await page.route(/\/api\/v1\/optimization\/jobs\/[0-9a-f-]+$/, (route) => fulfillJson(route, 200, task233CompletedJob()));
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

async function openMobileSidebarIfNeeded(page) {
  const mobileToggle = page.getByLabel("Open activity sidebar");
  if (await mobileToggle.isVisible()) {
    await mobileToggle.click();
  }
}

async function clickSignIn(page) {
  await openMobileSidebarIfNeeded(page);
  await page.getByRole("button", { name: "Sign in" }).click();
}

async function setTheme(page, theme) {
  await openMobileSidebarIfNeeded(page);
  const toggle = page.getByLabel("Theme preference");
  const dark = await toggle.getAttribute("aria-pressed") === "true";
  if ((theme === "dark") !== dark) await toggle.click();
  if (await page.locator("html").getAttribute("data-theme") !== theme) {
    throw new Error(`Task 233 theme did not resolve to ${theme}`);
  }
  const close = page.getByLabel("Close activity sidebar");
  if (await close.isVisible()) await close.click();
}

async function assertTask233SafeState(page) {
  assertTask233FixtureSafe();
  const text = await page.locator("body").innerText();
  if (/redis:\/\/|postgres:\/\/|<script>|backendDiagnostic|secret\.internal/i.test(text)) {
    throw new Error("Task 233 screenshot contains unsafe backend state");
  }
  if (await page.locator("[data-optimization-progress], [data-daily-diet-save-error], [data-optimization-error]").count() !== 0) {
    throw new Error("Task 233 screenshot contains stale loading or error state");
  }
  const dietSummary = page.locator(`[data-saved-daily-diet="${TASK_233_DIET_ID}"], [data-daily-diet-choice="${TASK_233_DIET_ID}"]`);
  if (await dietSummary.count() !== 1 || !/^2 meals\b/m.test(await dietSummary.innerText())) {
    throw new Error("Task 233 screenshot does not show the required two-meal Daily Diet");
  }
}

function assertTask233FixtureSafe() {
  const session = authSessionEnvelope().data;
  const entitlement = entitlementEnvelope().data;
  const diet = task233Diet();
  const now = Date.now();
  const sessionExpiries = [session.accessExpiresAt, session.refreshExpiresAt].map(Date.parse);
  const trialExpiry = entitlement.trialExpiresAt === null ? null : Date.parse(entitlement.trialExpiresAt);
  if (session.userId !== TASK_233_USER_ID || entitlement.userId !== TASK_233_USER_ID) {
    throw new Error("Task 233 fixture identity is inconsistent");
  }
  if (sessionExpiries.some((expiry) => !Number.isFinite(expiry) || expiry <= now)) {
    throw new Error("Task 233 fixture session is expired or invalid");
  }
  if (entitlement.status !== "active" || (entitlement.tier === "trial" && (trialExpiry === null || !Number.isFinite(trialExpiry) || trialExpiry <= now))) {
    throw new Error("Task 233 fixture entitlement is expired or invalid");
  }
  if (diet.entries.length !== 2 || new Set(diet.entries.map((entry) => entry.mealId)).size !== 2) {
    throw new Error("Task 233 fixture must contain exactly two distinct meals");
  }
}

function authSessionEnvelope() {
  return {
    status: "ok",
    requestId: "auth-session-screenshot-0001",
    data: {
      userId: TASK_233_USER_ID,
      role: "user",
      hasVerifiedLoginMethod: true,
      accessExpiresAt: "2027-07-18T13:00:00Z",
      refreshExpiresAt: "2027-07-25T13:00:00Z"
    }
  };
}

function entitlementEnvelope() {
  return {
    status: "ok",
    requestId: "entitlement-screenshot-0001",
    data: {
      userId: TASK_233_USER_ID,
      tier: "trial",
      status: "active",
      allowedModes: ["catalog", "substitution", "daily_diet", "daily_diet_alternative"],
      searchLimitPer24h: 25,
      usageUsed: 4,
      usageRemaining: 21,
      usageWindowStartedAt: "2026-07-05T00:00:00Z",
      trialExpiresAt: "2027-07-25T00:00:00Z",
      billingRecoveryState: "none"
    }
  };
}

function task233Diet() {
  return {
    id: TASK_233_DIET_ID,
    name: "Task 233 training day",
    entries: [
      { id: TASK_233_ENTRY_ID, mealId: TASK_233_MEAL_ID, quantity: 150, unit: "g", position: 0 },
      { id: TASK_233_SECOND_ENTRY_ID, mealId: TASK_233_SECOND_MEAL_ID, quantity: 100, unit: "g", position: 1 }
    ],
    aggregateMacros: { protein: 45, carbohydrates: 90, fat: 12, calories: 648 },
    createdAt: "2026-07-18T00:00:00Z",
    updatedAt: "2026-07-18T00:00:01Z"
  };
}

function task233CompletedJob() {
  return {
    status: "ok",
    requestId: "task-233-screenshot-completed",
    data: {
      jobId: TASK_233_JOB_ID,
      dailyDietId: TASK_233_DIET_ID,
      status: "completed",
      pollUrl: `/api/v1/optimization/jobs/${TASK_233_JOB_ID}`,
      createdAt: "2026-07-18T00:00:00Z",
      startedAt: "2026-07-18T00:00:01Z",
      finishedAt: "2026-07-18T00:00:02Z",
      alternatives: [{
        meals: [
          { mealId: TASK_233_MEAL_ID, quantity: 120, unit: "g", position: 0 },
          { mealId: TASK_233_SECOND_MEAL_ID, quantity: 100, unit: "g", position: 1 }
        ],
        macros: { protein: 45, carbohydrates: 90, fat: 12, calories: 648 },
        similarityScore: 0.91
      }]
    }
  };
}

function authErrorEnvelope(requestId, code, message) {
  return {
    status: "error",
    requestId,
    error: {
      category: "auth",
      code,
      message,
      retryable: false
    }
  };
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
