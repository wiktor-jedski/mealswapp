import { expect, test, type Page, type Route } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import type {
  AuthSessionEnvelope,
  CSRFTokenEnvelope,
  EntitlementStatusEnvelope,
  ProfileEnvelope,
  SearchResponseEnvelope
} from "../src/lib/api/generated";

// Implements DESIGN-018 AuthView/AuthSessionStore browser workflow coverage.
// Implements DESIGN-001 SearchView and SidebarComponent composition coverage.
// Verifies ARCH-018 and ARCH-001 browser-session integration behavior.
// Verifies IT-ARCH-018-001, IT-ARCH-018-002, ARCH-018, DESIGN-018, SW-REQ-058, SW-REQ-060, SW-REQ-061, SW-REQ-064, and SW-REQ-070.

function csrfEnvelope(): CSRFTokenEnvelope {
  return {
    status: "ok",
    requestId: "auth-session-csrf",
    data: { csrfToken: "csrf-auth-session" }
  };
}

function authSessionEnvelope(): AuthSessionEnvelope {
  return {
    status: "ok",
    requestId: "auth-session",
    data: {
      userId: "user-auth-session-1",
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
    requestId: "auth-session-profile",
    data: {
      userId: "user-auth-session-1",
      displayName: "Session User",
      unitSystem: "metric",
      themePreference: "system",
      requiresUnitRecalculation: false
    }
  };
}

function entitlementEnvelope(): EntitlementStatusEnvelope {
  return {
    status: "ok",
    requestId: "auth-session-entitlement",
    data: {
      userId: "user-auth-session-1",
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
    requestId: "auth-session-search",
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

async function stubSessionWorkflow(page: Page): Promise<{
  checkoutAttempts: () => number;
  entitlementAttempts: () => number;
  logoutAttempts: () => number;
}> {
  let authenticated = false;
  let checkoutAttempts = 0;
  let entitlementAttempts = 0;
  let logoutAttempts = 0;

  await page.route(/\/api\/v1\/profile$/, (route) => {
    if (!authenticated) {
      return fulfillJson(route, 401, {
        status: "error",
        requestId: "auth-session-profile-anonymous",
        error: {
          category: "auth",
          code: "anonymous_session",
          message: "Please sign in.",
          retryable: false
        }
      });
    }
    return fulfillJson(route, 200, profileEnvelope());
  });
  await page.route(/\/api\/v1\/auth\/refresh$/, (route) => fulfillJson(route, 200, authSessionEnvelope()));
  await page.route(/\/api\/v1\/auth\/csrf-token$/, (route) => fulfillJson(route, 200, csrfEnvelope()));
  await page.route(/\/api\/v1\/auth\/register$/, (route) => {
    authenticated = true;
    return fulfillJson(route, 201, authSessionEnvelope());
  });
  await page.route(/\/api\/v1\/auth\/login$/, (route) => {
    authenticated = true;
    return fulfillJson(route, 200, authSessionEnvelope());
  });
  await page.route(/\/api\/v1\/auth\/logout$/, (route) => {
    authenticated = false;
    logoutAttempts += 1;
    return fulfillJson(route, 200, { status: "ok", requestId: "auth-session-logout" });
  });
  await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => {
    entitlementAttempts += 1;
    if (!authenticated) {
      return fulfillJson(route, 401, {
        status: "error",
        requestId: "auth-session-entitlement-anonymous",
        error: {
          category: "auth",
          code: "anonymous_session",
          message: "Sign in to view your billing status.",
          retryable: false
        }
      });
    }
    return fulfillJson(route, 200, entitlementEnvelope());
  });
  await page.route(/\/api\/v1\/billing\/checkout$/, (route) => {
    checkoutAttempts += 1;
    return fulfillJson(route, 200, {
      status: "ok",
      requestId: "auth-session-checkout",
      data: {
        checkoutSessionId: "cs_auth_session",
        checkoutUrl: "http://localhost:4173/stripe-hosted/auth-session",
        plan: route.request().postDataJSON().plan,
        priceId: "price_auth_session",
        amountCents: 900
      }
    });
  });
  await page.route(/\/api\/v1\/search-history$/, (route) =>
    fulfillJson(route, 200, { status: "ok", requestId: "auth-session-history", data: { history: [] } })
  );
  await page.route(/\/api\/v1\/saved-items\?kind=favorite$/, (route) =>
    fulfillJson(route, 200, { status: "ok", requestId: "auth-session-favorites", data: { items: [] } })
  );
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) =>
    fulfillJson(route, 200, { status: "ok", requestId: "auth-session-autocomplete", data: { items: [] } })
  );
  await page.route(/\/api\/v1\/search$/, (route) => fulfillJson(route, 200, searchEnvelope()));

  return {
    checkoutAttempts: () => checkoutAttempts,
    entitlementAttempts: () => entitlementAttempts,
    logoutAttempts: () => logoutAttempts
  };
}

async function fillRegistration(page: Page): Promise<void> {
  await page.getByLabel("Email").fill("person@example.com");
  await page.getByLabel("Password", { exact: true }).fill("correct-horse-1");
  await page.getByLabel("Confirm password").fill("correct-horse-1");
  await page.getByLabel("I accept the current Privacy Policy and Terms of Service.").check();
}

async function assertNoSeriousAxeViolations(page: Page, selector: string): Promise<void> {
  const results = await new AxeBuilder({ page })
    .include(selector)
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
    .analyze();
  const seriousOrCritical = results.violations.filter(
    (violation) => violation.impact === "critical" || violation.impact === "serious"
  );
  expect(seriousOrCritical, seriousOrCritical.map((violation) => `${violation.id}: ${violation.description}`).join("\n")).toEqual([]);
}

test("registration, login, logout, anonymous search fallback, sidebar navigation, keyboard flow, and axe checks work together", async ({ page }) => {
  const workflow = await stubSessionWorkflow(page);

  await page.goto("/subscription");
  await page.waitForResponse(/\/api\/v1\/profile$/);
  await expect(page.locator("[data-auth-guidance]")).toContainText("open subscription");
  await expect(page.locator("[data-subscription-view]")).toHaveCount(0);
  expect(workflow.checkoutAttempts()).toBe(0);
  await assertNoSeriousAxeViolations(page, "[data-auth-surface]");

  await page.getByRole("button", { name: "Create account" }).first().click();
  await fillRegistration(page);
  await page.getByRole("button", { name: "Create account" }).last().click();
  await expect(page.locator("[data-subscription-view]")).toBeVisible();
  await expect(page.locator("[data-subscription-view] [data-subscription-billing]")).toBeVisible();
  await expect(page.locator("[data-sidebar-sign-out]")).toHaveCount(1);
  await expect(page.getByText("Signed in")).toHaveCount(0);
  expect(workflow.entitlementAttempts()).toBeGreaterThan(0);

  await openMobileSidebarIfNeeded(page);
  await page.locator("[data-sidebar-sign-out]").click();
  await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();
  await expect(page.locator("[data-subscription-view]")).toHaveCount(0);
  expect(workflow.logoutAttempts()).toBe(1);

  await page.getByLabel("Food search").fill("apple");
  await page.getByLabel("Food search").press("Enter");
  await expect(page.locator("[data-result-card]")).toHaveCount(1);

  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByRole("dialog", { name: "Sign in" })).toBeVisible();
  await assertNoSeriousAxeViolations(page, "[data-auth-surface]");
  await page.getByLabel("Email").fill("person@example.com");
  await page.getByLabel("Password").fill("correct-horse-1");
  await page.getByRole("form").getByRole("button", { name: "Sign in" }).click();
  await expect(page.locator("[data-sidebar-sign-out]")).toHaveCount(1);
  await expect(page.getByText("Signed in")).toHaveCount(0);

  await openMobileSidebarIfNeeded(page);
  await expect(page.getByRole("navigation", { name: "Account navigation" }).getByRole("button", { name: "Search" })).toBeVisible();
  await page.locator("#sidebar-unit-system").focus();
  await page.keyboard.press("Tab");
  await expect(page.locator("[data-sidebar-sign-out]")).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(page.getByRole("navigation", { name: "Account navigation" }).getByRole("button", { name: "Search" })).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(page.getByRole("navigation", { name: "Account navigation" }).getByRole("button", { name: "Subscription" })).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.locator("[data-subscription-view]")).toBeVisible();

  await openMobileSidebarIfNeeded(page);
  await page.getByRole("navigation", { name: "Account navigation" }).getByRole("button", { name: "Search" }).click();
  await closeMobileSidebarIfNeeded(page);
  await expect(page.getByLabel("Food search")).toHaveValue("apple");
  await expect(page.locator("[data-result-card]")).toHaveCount(1);
});
