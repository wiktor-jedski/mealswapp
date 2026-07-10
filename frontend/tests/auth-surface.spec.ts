import { expect, test, type Page, type Route } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

// Implements DESIGN-018 AuthView modal and OAuth entry verification plus DESIGN-015 Terms disclaimer placement.
// Verifies IT-ARCH-018-004, ARCH-018, DESIGN-018, SW-REQ-046, SW-REQ-058, SW-REQ-061, SW-REQ-071, and SW-REQ-074.

async function fulfillJson(route: Route, status: number, body: unknown): Promise<void> {
  await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

async function stubAnonymousProfileProbe(page: Page): Promise<void> {
  await page.route(/\/api\/v1\/profile$/, (route) =>
    fulfillJson(route, 401, {
      status: "error",
      requestId: "auth-surface-profile-anonymous",
      error: {
        category: "auth",
        code: "anonymous_session",
        message: "Please sign in.",
        retryable: false
      }
    })
  );
}

async function openAuthModal(page: Page): Promise<void> {
  const mobileToggle = page.getByLabel("Open activity sidebar");
  if (await mobileToggle.isVisible()) {
    await mobileToggle.click();
  }
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
}

test("SearchShell modal is the sole auth surface and exposes Google sign-in", async ({ page }) => {
  await stubAnonymousProfileProbe(page);
  await page.goto("/");
  await openAuthModal(page);

  await expect(page.locator("[data-auth-surface]")).toBeVisible();
  await expect(page.getByRole("dialog", { name: "Sign in" })).toBeVisible();
  await expect(page.locator("[data-oauth-provider='google']")).toBeVisible();
  await expect(page.locator("[data-oauth-provider='apple']")).toHaveCount(0);
  await expect(page.locator("[data-auth-disclaimer]")).toHaveCount(0);

  const accessibilityScanResults = await new AxeBuilder({ page })
    .include("[data-auth-surface]")
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
    .analyze();
  const seriousOrCritical = accessibilityScanResults.violations.filter((violation) =>
    violation.impact === "serious" || violation.impact === "critical"
  );
  expect(seriousOrCritical).toEqual([]);
});

test("Terms of Service contains medical information outside the auth surface", async ({ page }) => {
  await stubAnonymousProfileProbe(page);
  await page.goto("/terms");

  await expect(page.locator("[data-medical-disclaimer]")).toContainText("does not provide medical advice");
  await expect(page.locator("[data-auth-surface]")).toHaveCount(0);
});

test("OAuth callback keeps the SearchShell mounted and refreshes the modal session", async ({ page }) => {
  await stubAnonymousProfileProbe(page);
  await page.route(/\/api\/v1\/auth\/refresh$/, (route) =>
    fulfillJson(route, 200, {
      status: "ok",
      requestId: "oauth-callback-refresh",
      data: {
        userId: "oauth-user",
        role: "user",
        hasVerifiedLoginMethod: true,
        accessExpiresAt: "2026-07-10T12:00:00Z",
        refreshExpiresAt: "2026-07-17T12:00:00Z"
      }
    })
  );
  await page.route(/\/api\/v1\/billing\/entitlement$/, (route) =>
    fulfillJson(route, 200, {
      status: "ok",
      requestId: "oauth-callback-entitlement",
      data: {
        userId: "oauth-user",
        tier: "free",
        status: "active",
        allowedModes: ["catalog", "substitution"],
        searchLimitPer24h: 3,
        usageUsed: 0,
        usageRemaining: 3,
        usageWindowStartedAt: "2026-07-10T00:00:00Z",
        billingRecoveryState: "none"
      }
    })
  );

  await page.goto("/auth/callback?success=true");

  await expect(page.getByRole("dialog", { name: "Sign in" })).toBeVisible();
  await expect(page.locator("[data-oauth-message]")).toHaveText("Sign-in session refreshed.");
  await expect(page.locator("main").first()).toBeVisible();
});
