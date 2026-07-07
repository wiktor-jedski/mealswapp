import { expect, test, type Page, type Route } from "@playwright/test";
import type {
  AuthSessionEnvelope,
  EntitlementStatusEnvelope,
  ProfileEnvelope,
  SearchResponseEnvelope
} from "../src/lib/api/generated";

// Implements DESIGN-001 SidebarComponent browser verification for authenticated Search and Subscription navigation.
// Implements DESIGN-018 AuthenticatedActionGuard verification for protected Subscription navigation.
// Verifies IT-ARCH-018-002, ARCH-018, DESIGN-018, SW-REQ-044, SW-REQ-058, and SW-REQ-061.

function authSessionEnvelope(): AuthSessionEnvelope {
  return {
    status: "ok",
    requestId: "subscription-nav-auth-session",
    data: {
      userId: "user-subscription-nav-1",
      role: "user",
      hasVerifiedLoginMethod: true,
      accessExpiresAt: "2026-07-05T13:00:00Z",
      refreshExpiresAt: "2026-07-12T13:00:00Z"
    }
  };
}

function profileEnvelope(): ProfileEnvelope {
  return {
    status: "ok",
    requestId: "subscription-nav-profile",
    data: {
      userId: "user-subscription-nav-1",
      displayName: "Navigation User",
      unitSystem: "metric",
      themePreference: "system",
      requiresUnitRecalculation: false
    }
  };
}

function entitlementEnvelope(): EntitlementStatusEnvelope {
  return {
    status: "ok",
    requestId: "subscription-nav-entitlement",
    data: {
      userId: "user-subscription-nav-1",
      tier: "trial",
      status: "active",
      allowedModes: ["catalog", "substitution"],
      searchLimitPer24h: 25,
      usageUsed: 0,
      usageRemaining: 25,
      usageWindowStartedAt: "2026-07-05T00:00:00Z",
      trialExpiresAt: "2026-07-12T00:00:00Z",
      billingRecoveryState: "none"
    }
  };
}

function searchEnvelope(): SearchResponseEnvelope {
  return {
    status: "ok",
    requestId: "subscription-nav-search",
    data: {
      items: [
        {
          id: "food-apple",
          name: "Apple",
          physicalState: "solid",
          imageUrl: null,
          classifications: [{ id: "fruit", name: "Fruit", kind: "food_category" }],
          primaryFoodCategory: { id: "fruit", name: "Fruit", kind: "food_category" },
          macros: { protein: 0.3, carbohydrates: 14, fat: 0.2 },
          macroBasis: "100g",
          calories: 52
        }
      ],
      totalCount: 1,
      page: 1,
      similarityScores: [1],
      similarityMetadata: [],
      warnings: []
    }
  };
}

async function fulfillJson(route: Route, status: number, body: unknown): Promise<void> {
  await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

async function stubAuthenticatedApp(page: Page): Promise<void> {
  await page.route(/\/api\/v1\/profile$/, (route) => fulfillJson(route, 200, profileEnvelope()));
  await page.route(/\/api\/v1\/auth\/refresh$/, (route) => fulfillJson(route, 200, authSessionEnvelope()));
  await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => fulfillJson(route, 200, entitlementEnvelope()));
  await page.route(/\/api\/v1\/search-history$/, (route) =>
    fulfillJson(route, 200, { status: "ok", requestId: "subscription-nav-history", data: { history: [] } })
  );
  await page.route(/\/api\/v1\/saved-items\?kind=favorite$/, (route) =>
    fulfillJson(route, 200, { status: "ok", requestId: "subscription-nav-favorites", data: { items: [] } })
  );
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) =>
    fulfillJson(route, 200, { status: "ok", requestId: "subscription-nav-autocomplete", data: { items: [] } })
  );
  await page.route(/\/api\/v1\/search$/, (route) => fulfillJson(route, 200, searchEnvelope()));
}

async function openMobileSidebarIfNeeded(page: Page): Promise<void> {
  const mobileToggle = page.getByLabel("Open activity sidebar");
  if (await mobileToggle.isVisible()) {
    await mobileToggle.click();
  }
}

async function closeMobileSidebarIfNeeded(page: Page): Promise<void> {
  const mobileClose = page.getByLabel("Close activity sidebar");
  if (await mobileClose.isVisible()) {
    await mobileClose.click();
  }
}

async function chooseAccountNavigation(page: Page, label: "Search" | "Subscription"): Promise<void> {
  await openMobileSidebarIfNeeded(page);
  await page.getByRole("navigation", { name: "Account navigation" }).getByRole("button", { name: label }).click();
}

async function focusAccountNavigationFromUnits(page: Page, label: "Search" | "Subscription"): Promise<void> {
  await openMobileSidebarIfNeeded(page);
  await page.locator("#sidebar-unit-system").focus();
  await page.keyboard.press("Tab");
  await expect(page.locator("[data-sidebar-sign-out]")).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(page.getByRole("navigation", { name: "Account navigation" }).getByRole("button", { name: "Search" })).toBeFocused();
  if (label === "Subscription") {
    await page.keyboard.press("Tab");
    await expect(page.getByRole("navigation", { name: "Account navigation" }).getByRole("button", { name: "Subscription" })).toBeFocused();
  }
}

// Implements DESIGN-016 ComponentStyles handheld viewport integrity checks.
async function expectNoMobileHorizontalScrollOrClippedAccountControls(page: Page): Promise<void> {
  const scrollMetrics = await page.evaluate(() => ({
    clientWidth: document.documentElement.clientWidth,
    scrollWidth: document.documentElement.scrollWidth
  }));
  expect(scrollMetrics.scrollWidth).toBeLessThanOrEqual(scrollMetrics.clientWidth);

  const navigation = page.getByRole("navigation", { name: "Account navigation" });
  const searchBox = await navigation.getByRole("button", { name: "Search" }).boundingBox();
  const subscriptionBox = await navigation.getByRole("button", { name: "Subscription" }).boundingBox();
  expect(searchBox).not.toBeNull();
  expect(subscriptionBox).not.toBeNull();
  expect(searchBox!.x).toBeGreaterThanOrEqual(0);
  expect(subscriptionBox!.x).toBeGreaterThanOrEqual(0);
  expect(searchBox!.x + searchBox!.width).toBeLessThanOrEqual(scrollMetrics.clientWidth);
  expect(subscriptionBox!.x + subscriptionBox!.width).toBeLessThanOrEqual(scrollMetrics.clientWidth);
  expect(searchBox!.y + searchBox!.height).toBeLessThanOrEqual(subscriptionBox!.y);
}

test("authenticated sidebar links separate Subscription from Search and preserve search state", async ({ page }) => {
  await stubAuthenticatedApp(page);

  await page.goto("/");
  await expect(page.locator("[data-sidebar-sign-out]")).toHaveCount(1);
  await expect(page.getByText("Signed in")).toHaveCount(0);
  await openMobileSidebarIfNeeded(page);
  await expect(page.getByRole("navigation", { name: "Account navigation" }).getByRole("button", { name: "Search" })).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Account navigation" }).getByRole("button", { name: "Subscription" })).toBeVisible();
  await expectNoMobileHorizontalScrollOrClippedAccountControls(page);
  await closeMobileSidebarIfNeeded(page);

  await page.getByLabel("Food search").fill("apple");
  await page.getByLabel("Food search").press("Enter");
  await expect(page.locator("[data-result-card]")).toHaveCount(1);

  await chooseAccountNavigation(page, "Subscription");
  await expect(page.locator("[data-subscription-view]")).toBeVisible();
  await expect(page.locator("[data-subscription-view] [data-subscription-billing]")).toBeVisible();
  await expect(page.locator("[data-subscription-view] #autocomplete-input")).toHaveCount(0);
  await expect(page.locator("[data-subscription-view] [data-results-grid]")).toHaveCount(0);

  await chooseAccountNavigation(page, "Search");
  await expect(page.locator("[data-subscription-view]")).toHaveCount(0);
  await expect(page.getByLabel("Food search")).toHaveValue("apple");
  await expect(page.locator("[data-result-card]")).toHaveCount(1);
});

test("keyboard focus reaches account links and Enter activation preserves search state", async ({ page }) => {
  await stubAuthenticatedApp(page);

  await page.goto("/");
  await expect(page.locator("[data-sidebar-sign-out]")).toHaveCount(1);
  await expect(page.getByText("Signed in")).toHaveCount(0);
  await page.getByLabel("Food search").fill("apple");
  await page.getByLabel("Food search").press("Enter");
  await expect(page.locator("[data-result-card]")).toHaveCount(1);

  await focusAccountNavigationFromUnits(page, "Subscription");
  await page.keyboard.press("Enter");
  await expect(page.locator("[data-subscription-view]")).toBeVisible();
  await expect(page.locator("[data-subscription-view] [data-subscription-billing]")).toBeVisible();

  await focusAccountNavigationFromUnits(page, "Search");
  await page.keyboard.press("Enter");
  await expect(page.locator("[data-subscription-view]")).toHaveCount(0);
  await expect(page.getByLabel("Food search")).toHaveValue("apple");
  await expect(page.locator("[data-result-card]")).toHaveCount(1);
});

test("mobile sidebar navigation remains usable and closes after Subscription selection", async ({ page }) => {
  await stubAuthenticatedApp(page);
  await page.setViewportSize({ width: 390, height: 844 });

  await page.goto("/");
  await expect(page.getByLabel("Open activity sidebar")).toBeVisible();
  await page.getByLabel("Open activity sidebar").click();
  await expectNoMobileHorizontalScrollOrClippedAccountControls(page);
  await page.getByRole("navigation", { name: "Account navigation" }).getByRole("button", { name: "Subscription" }).click();

  await expect(page.locator("[data-subscription-view]")).toBeVisible();
  const subscriptionMetrics = await page.evaluate(() => ({
    clientWidth: document.documentElement.clientWidth,
    scrollWidth: document.documentElement.scrollWidth
  }));
  expect(subscriptionMetrics.scrollWidth).toBeLessThanOrEqual(subscriptionMetrics.clientWidth);
  await expect(page.locator("[data-sidebar-content]")).toBeHidden();
  await expect(page.getByLabel("Open activity sidebar")).toBeVisible();
});
