import { expect, test, type Page, type Route } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import type {
  AuthSessionEnvelope,
  AutocompleteEnvelope,
  CSRFTokenEnvelope,
  DailyDiet,
  DailyDietCollectionEnvelope,
  DailyDietEnvelope,
  EntitlementStatusEnvelope,
  FoodObjectEnvelope,
  ProfileEnvelope,
  SearchResponseEnvelope
} from "../src/lib/api/generated";

// Implements DESIGN-001 SearchView authenticated Daily Diet Collection UI browser workflow.
// Implements DESIGN-008 SavedDataRepository server-derived collection and macro projection coverage.

function fulfillJson(route: Route, status: number, body: unknown): Promise<void> {
  return route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

function profileEnvelope(): ProfileEnvelope {
  return {
    status: "ok",
    requestId: "daily-diet-profile",
    data: {
      userId: "daily-diet-user",
      displayName: "Daily Diet User",
      unitSystem: "metric",
      themePreference: "system",
      requiresUnitRecalculation: false
    }
  };
}

function authSessionEnvelope(): AuthSessionEnvelope {
  return {
    status: "ok",
    requestId: "daily-diet-session",
    data: {
      userId: "daily-diet-user",
      role: "user",
      hasVerifiedLoginMethod: true,
      accessExpiresAt: "2026-07-05T13:00:00Z",
      refreshExpiresAt: "2026-07-12T13:00:00Z"
    }
  };
}

function entitlementEnvelope(tier: "free" | "paid" = "paid"): EntitlementStatusEnvelope {
  return {
    status: "ok",
    requestId: "daily-diet-entitlement",
    data: {
      userId: "daily-diet-user",
      tier,
      status: "active",
      allowedModes: tier === "paid" ? ["catalog", "substitution", "daily_diet", "daily_diet_alternative"] : ["catalog"],
      searchLimitPer24h: 10,
      usageUsed: 0,
      usageRemaining: tier === "free" ? 10 : null,
      usageWindowStartedAt: "2026-07-05T00:00:00Z",
      trialExpiresAt: null,
      billingRecoveryState: "none"
    }
  };
}

function meal(id: "meal-apple" | "meal-oats"): FoodObjectEnvelope {
  const apple = id === "meal-apple";
  return {
    status: "ok",
    requestId: `daily-diet-${id}`,
    data: {
      id,
      name: apple ? "Apple" : "Oats",
      physicalState: "solid",
      imageUrl: null,
      classifications: [{ id: "breakfast", name: "Breakfast", kind: "food_category" }],
      primaryFoodCategory: { id: "breakfast", name: "Breakfast", kind: "food_category" },
      macros: apple ? { protein: 1, carbohydrates: 14, fat: 0.2 } : { protein: 13, carbohydrates: 68, fat: 7 },
      macroBasis: "100g",
      calories: apple ? 52 : 389
    }
  };
}

function autocompleteEnvelope(): AutocompleteEnvelope {
  return {
    status: "ok",
    requestId: "daily-diet-autocomplete",
    data: {
      items: [
        { itemId: "meal-apple", label: "Apple", exactMatch: true, levenshteinDistance: 0, length: 5, rank: 1 },
        { itemId: "meal-oats", label: "Oats", exactMatch: true, levenshteinDistance: 0, length: 4, rank: 1 }
      ]
    }
  };
}

function emptyDailyDiets(): DailyDietCollectionEnvelope {
  return { status: "ok", requestId: "daily-diet-list-empty", data: { diets: [] } };
}

function savedDailyDiet(): DailyDiet {
  return {
    id: "diet-1",
    name: "Saved breakfast",
    entries: [{ id: "entry-1", mealId: "meal-apple", quantity: 100, unit: "g", position: 0 }],
    aggregateMacros: { protein: 1, carbohydrates: 14, fat: 0.2, calories: 52 },
    createdAt: "2026-07-11T00:00:00Z",
    updatedAt: "2026-07-11T00:00:00Z"
  };
}

function searchEnvelope(): SearchResponseEnvelope {
  return {
    status: "ok",
    requestId: "daily-diet-search",
    data: { items: [], totalCount: 0, page: 1, similarityScores: [], similarityMetadata: [], warnings: [] }
  };
}

async function stubAuthenticatedDailyDiet(
  page: Page,
  tier: "free" | "paid" = "paid",
  listBehavior?: (route: Route) => Promise<void>
): Promise<{ createBodies: () => Array<Record<string, unknown>> }> {
  const createBodies: Array<Record<string, unknown>> = [];
  let savedDiet: DailyDiet | null = null;
  await page.route(/\/api\/v1\/profile$/, (route) => fulfillJson(route, 200, profileEnvelope()));
  await page.route(/\/api\/v1\/auth\/refresh$/, (route) => fulfillJson(route, 200, authSessionEnvelope()));
  await page.route(/\/api\/v1\/auth\/csrf-token$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "daily-diet-csrf", data: { csrfToken: "csrf-daily-diet" } } satisfies CSRFTokenEnvelope));
  await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => fulfillJson(route, 200, entitlementEnvelope(tier)));
  await page.route(/\/api\/v1\/search-history$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "daily-diet-history", data: { history: [] } }));
  await page.route(/\/api\/v1\/saved-items\?kind=favorite$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "daily-diet-favorites", data: { items: [] } }));
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope()));
  await page.route(/\/api\/v1\/food-objects\/meal-apple$/, (route) => fulfillJson(route, 200, meal("meal-apple")));
  await page.route(/\/api\/v1\/food-objects\/meal-oats$/, (route) => fulfillJson(route, 200, meal("meal-oats")));
  await page.route(/\/api\/v1\/search$/, (route) => fulfillJson(route, 200, searchEnvelope()));
  await page.route(/\/api\/v1\/daily-diets$/, async (route) => {
    if (route.request().method() === "POST") {
      const body = route.request().postDataJSON() as Record<string, unknown>;
      createBodies.push(body);
      const entries = (body.entries as Array<Record<string, unknown>>).map((entry, index) => ({
        id: `entry-${index + 1}`,
        ...entry
      }));
      const saved: DailyDiet = {
        id: "diet-1",
        name: String(body.name),
        entries: entries as DailyDiet["entries"],
        aggregateMacros: { protein: 31, carbohydrates: 82, fat: 7.2, calories: 500 },
        createdAt: "2026-07-11T00:00:00Z",
        updatedAt: "2026-07-11T00:00:00Z"
      };
      savedDiet = saved;
      return fulfillJson(route, 201, { status: "ok", requestId: "daily-diet-created", data: saved } satisfies DailyDietEnvelope);
    }
    if (listBehavior) return listBehavior(route);
    return fulfillJson(route, 200, savedDiet ? { status: "ok", requestId: "daily-diet-list", data: { diets: [savedDiet] } } satisfies DailyDietCollectionEnvelope : emptyDailyDiets());
  });
  await page.route(/\/api\/v1\/daily-diets\/diet-1$/, (route) => fulfillJson(route, 200, emptyDailyDiets()));
  return { createBodies: () => createBodies };
}

async function selectMeal(page: Page, query: string, label: string, keyboard = false): Promise<void> {
  await page.getByLabel("Food search").fill(query);
  await expect(page.getByRole("listbox", { name: "Autocomplete suggestions" })).toBeVisible();
  if (keyboard) {
    await page.getByLabel("Food search").press("Enter");
  } else {
    await page.getByRole("option", { name: label }).click();
  }
  await expect(page.locator(`[data-daily-diet-meal]`).filter({ hasText: label })).toBeVisible();
}

test("authenticated user builds, edits, saves, and selects a two-meal Daily Diet", async ({ page }) => {
  const api = await stubAuthenticatedDailyDiet(page);
  await page.goto("/");
  await page.getByRole("button", { name: "Daily Diet", exact: true }).click();
  await expect(page.locator("[data-daily-diet-empty]")).toBeVisible();

  await selectMeal(page, "apple", "Apple", true);
  await selectMeal(page, "oats", "Oats");
  await page.getByLabel("Quantity for Apple").fill("150");
  await page.getByRole("button", { name: "Move Oats up" }).click();
  await expect(page.locator("[data-daily-diet-meals] li").first()).toContainText("Oats");
  await page.getByRole("button", { name: "Remove Oats" }).click();
  await selectMeal(page, "oats", "Oats");

  await page.getByLabel("Collection name").fill("Training day");
  await page.getByRole("button", { name: "Save Daily Diet" }).click();
  await expect(page.locator("[data-daily-diet-server-total]")).toHaveText("Totals confirmed by the server.");
  await expect(page.locator("[data-macro-protein]")).toHaveText("31g");
  await expect(page.locator("[data-macro-carbs]")).toHaveText("82g");
  expect(api.createBodies()[0]).toMatchObject({
    name: "Training day",
    entries: [
      { mealId: "meal-apple", quantity: 150, position: 0 },
      { mealId: "meal-oats", quantity: 100, position: 1 }
    ]
  });

  const collectionAxe = await new AxeBuilder({ page }).include("[data-daily-diet-collection]").analyze();
  expect(collectionAxe.violations.filter((violation) => violation.impact === "serious" || violation.impact === "critical")).toEqual([]);

  await page.getByRole("button", { name: "Daily Diet Alternative", exact: true }).click();
  await expect(page.getByRole("radio", { name: "Use Training day as Daily Diet Alternative input" })).toBeVisible();
  await page.getByRole("radio", { name: "Use Training day as Daily Diet Alternative input" }).click();
  await expect(page.locator("[data-daily-diet-alternative-selected]")).toBeVisible();

  const axe = await new AxeBuilder({ page }).include("[data-daily-diet-alternative-controls]").analyze();
  expect(axe.violations.filter((violation) => violation.impact === "serious" || violation.impact === "critical")).toEqual([]);
});

test("logout clears the authenticated user's unsaved Daily Diet draft", async ({ page }) => {
  await stubAuthenticatedDailyDiet(page);
  await page.route(/\/api\/v1\/auth\/logout$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "daily-diet-logout" }));
  await page.goto("/");
  await page.getByRole("button", { name: "Daily Diet", exact: true }).click();
  await selectMeal(page, "apple", "Apple");
  await selectMeal(page, "oats", "Oats");
  await expect(page.locator("[data-daily-diet-meal]")).toHaveCount(2);

  const mobileToggle = page.getByLabel("Open activity sidebar");
  if (await mobileToggle.isVisible()) {
    await mobileToggle.click();
  }
  await page.getByRole("button", { name: "Sign out" }).click();

  await expect(page.locator("[data-daily-diet-meal]")).toHaveCount(0);
  await expect(page.locator("[data-daily-diet-auth-guidance]")).toContainText("Sign in to build and save a Daily Diet.");
});

test("anonymous Daily Diet view gives sign-in guidance without loading protected collections", async ({ page }) => {
  let dailyDietRequests = 0;
  await page.route(/\/api\/v1\/profile$/, (route) => fulfillJson(route, 401, { status: "error", requestId: "daily-diet-anonymous", error: { category: "auth", code: "anonymous_session", message: "Please sign in.", retryable: false } }));
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, autocompleteEnvelope()));
  await page.route(/\/api\/v1\/daily-diets/, (route) => { dailyDietRequests += 1; return fulfillJson(route, 401, { status: "error", requestId: "unexpected-daily-diet-call", error: { category: "auth", code: "anonymous_session", message: "Please sign in.", retryable: false } }); });
  await page.goto("/");
  await page.getByRole("button", { name: "Daily Diet", exact: true }).click();
  await expect(page.locator("[data-daily-diet-auth-guidance]")).toBeVisible();
  expect(dailyDietRequests).toBe(0);
});

test("authenticated free user sees entitlement guidance and cannot save", async ({ page }) => {
  await stubAuthenticatedDailyDiet(page, "free");
  await page.goto("/");
  await page.getByRole("button", { name: "Daily Diet", exact: true }).click();
  await expect(page.locator("[data-daily-diet-entitlement]")).toContainText(/not included|available on trial and paid plans/);
  await expect(page.getByRole("button", { name: "Save Daily Diet" })).toBeDisabled();
});

test("shows the real loading state while collections load, then resolves to empty", async ({ page }) => {
  let releaseList!: () => void;
  const listPending = new Promise<void>((resolve) => { releaseList = resolve; });
  await stubAuthenticatedDailyDiet(page, "paid", async (route) => {
    await listPending;
    await fulfillJson(route, 200, emptyDailyDiets());
  });

  await page.goto("/");
  await page.getByRole("button", { name: "Daily Diet", exact: true }).click();
  await expect(page.locator("[data-saved-daily-diets-loading]")).toBeVisible();
  releaseList();
  await expect(page.locator("[data-saved-daily-diets-empty]")).toBeVisible();
});

test("recovers from a real collection-list error through the retry action", async ({ page }) => {
  let listAttempts = 0;
  await stubAuthenticatedDailyDiet(page, "paid", async (route) => {
    listAttempts += 1;
    if (listAttempts === 1) {
      await fulfillJson(route, 503, {
        status: "error",
        requestId: "daily-diet-list-failure",
        error: {
          category: "dependency",
          code: "daily_diet_unavailable",
          message: "Saved daily diets are temporarily unavailable. Please try again shortly.",
          retryable: true
        }
      });
      return;
    }
    await fulfillJson(route, 200, { status: "ok", requestId: "daily-diet-list-recovered", data: { diets: [savedDailyDiet()] } } satisfies DailyDietCollectionEnvelope);
  });

  await page.goto("/");
  await page.getByRole("button", { name: "Daily Diet", exact: true }).click();
  await expect(page.locator("[data-saved-daily-diets-error]")).toContainText("temporarily unavailable");
  await page.getByRole("button", { name: "Try again" }).click();
  await expect(page.locator("[data-saved-daily-diet=diet-1]")).toContainText("Saved breakfast");
  expect(listAttempts).toBe(2);
});

test("keyboard focus moves from mode to search and into the collection editor", async ({ page }) => {
  await stubAuthenticatedDailyDiet(page);
  await page.goto("/");

  const dailyDietMode = page.getByRole("button", { name: "Daily Diet", exact: true });
  await dailyDietMode.focus();
  await dailyDietMode.press("Enter");
  await expect(page.getByLabel("Food search")).toBeFocused();

  await page.keyboard.type("apple");
  await expect(page.getByRole("listbox", { name: "Autocomplete suggestions" })).toBeVisible();
  await page.getByLabel("Food search").press("Enter");
  await expect(page.getByLabel("Food search")).toBeFocused();

  await page.keyboard.press("Tab");
  await expect(page.getByLabel("Collection name")).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(page.getByLabel("Quantity for Apple")).toBeFocused();
  await page.keyboard.press("ControlOrMeta+A");
  await page.keyboard.type("125");
  await expect(page.getByLabel("Quantity for Apple")).toHaveValue("125");
});
