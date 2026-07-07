import { expect, test, type Page, type Route } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

// Implements DESIGN-018 AuthView browser verification for disclaimer and OAuth entry surfaces.
// Verifies IT-ARCH-018-004, IT-ARCH-018-005, ARCH-018, DESIGN-018, SW-REQ-046, SW-REQ-058, SW-REQ-061, SW-REQ-071, and SW-REQ-074.

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

test("auth surface loads login disclaimer and exposes generated OAuth provider actions", async ({ page }) => {
  await stubAnonymousProfileProbe(page);
  await page.route(/\/api\/v1\/disclaimers\?location=login$/, (route) =>
    fulfillJson(route, 200, {
      status: "ok",
      requestId: "auth-surface-disclaimer",
      data: {
        location: "login",
        version: "2026-07",
        markdown: "Generated login medical disclaimer.",
        effectiveAt: "2026-07-05T00:00:00.000Z"
      }
    })
  );

  await page.goto("/auth");

  await expect(page.locator("[data-auth-surface]")).toBeVisible();
  await expect(page.locator("[data-auth-disclaimer]")).toContainText("Generated login medical disclaimer.");
  await expect(page.locator("[data-oauth-provider='google']")).toBeVisible();
  await expect(page.locator("[data-oauth-provider='apple']")).toHaveCount(0);

  const accessibilityScanResults = await new AxeBuilder({ page })
    .include("[data-auth-surface]")
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
    .analyze();
  const seriousOrCritical = accessibilityScanResults.violations.filter((violation) =>
    violation.impact === "serious" || violation.impact === "critical"
  );
  expect(seriousOrCritical).toEqual([]);
});

test("auth surface renders bundled fallback when disclaimer API is unavailable", async ({ page }) => {
  await stubAnonymousProfileProbe(page);
  await page.route(/\/api\/v1\/disclaimers\?location=login$/, (route) =>
    fulfillJson(route, 503, {
      status: "error",
      requestId: "auth-surface-disclaimer-unavailable",
      error: {
        category: "dependency",
        code: "disclaimer_unavailable",
        message: "Disclaimer unavailable.",
        retryable: true
      }
    })
  );

  await page.goto("/auth");

  await expect(page.locator("[data-disclaimer-fallback]")).toBeVisible();
  await expect(page.locator("[data-auth-disclaimer]")).toContainText("not medical advice");
});
