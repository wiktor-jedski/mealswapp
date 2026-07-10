import { expect, test, type Page, type Route } from "@playwright/test";
import type {
  AuthSessionEnvelope,
  CSRFTokenEnvelope,
  EntitlementStatusEnvelope,
  ProfileEnvelope
} from "../src/lib/api/generated";

// Implements DESIGN-018 LoginView browser verification for credential feedback, lockout metadata, duplicate-submit prevention, and protected-action handoff.
// Verifies IT-ARCH-018-001, IT-ARCH-018-003, IT-ARCH-018-005, ARCH-018, DESIGN-018, SW-REQ-044, SW-REQ-058, SW-REQ-061, SW-REQ-062, SW-REQ-063, and SW-REQ-065.

function csrfEnvelope(): CSRFTokenEnvelope {
  return {
    status: "ok",
    requestId: "csrf-login",
    data: { csrfToken: "csrf-token" }
  };
}

function authSessionEnvelope(): AuthSessionEnvelope {
  return {
    status: "ok",
    requestId: "login-session",
    data: {
      userId: "user-login-1",
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
    requestId: "profile-login",
    data: {
      userId: "user-login-1",
      displayName: "Login User",
      unitSystem: "metric",
      themePreference: "system",
      requiresUnitRecalculation: false
    }
  };
}

function entitlementEnvelope(): EntitlementStatusEnvelope {
  return {
    status: "ok",
    requestId: "entitlement-login",
    data: {
      userId: "user-login-1",
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

async function fulfillJson(route: Route, status: number, body: unknown, headers: Record<string, string> = {}): Promise<void> {
  await route.fulfill({ status, contentType: "application/json", headers, body: JSON.stringify(body) });
}

async function clickSidebarSignIn(page: Page): Promise<void> {
  const mobileToggle = page.getByLabel("Open activity sidebar");
  if (await mobileToggle.isVisible()) {
    await mobileToggle.click();
  }
  await page.locator("[data-sidebar-sign-in]").click();
}

async function stubAnonymousProfile(page: Page): Promise<void> {
  await page.route(/\/api\/v1\/profile$/, (route) =>
    fulfillJson(route, 401, {
      status: "error",
      requestId: "profile-anonymous",
      error: {
        category: "auth",
        code: "invalid_credentials",
        message: "Not signed in.",
        retryable: false
      }
    })
  );
}

async function stubCommonAuth(page: Page): Promise<void> {
  await page.route(/\/api\/v1\/auth\/csrf-token$/, (route) => fulfillJson(route, 200, csrfEnvelope()));
  await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => fulfillJson(route, 200, entitlementEnvelope()));
}

test("login form validates focus order, generic invalid credentials, lockout retry timing, duplicate submissions, and password clearing", async ({ page }) => {
  await stubAnonymousProfile(page);
  await stubCommonAuth(page);
  let loginAttempts = 0;
  await page.route(/\/api\/v1\/auth\/login$/, async (route) => {
    loginAttempts += 1;
    if (loginAttempts === 1) {
      await fulfillJson(route, 401, {
        status: "error",
        requestId: "bad-login",
        error: {
          category: "auth",
          code: "invalid_credentials",
          message: "Server text must not enumerate accounts.",
          retryable: false
        }
      });
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, 150));
    await fulfillJson(
      route,
      429,
      {
        status: "error",
        requestId: "locked-login",
        error: {
          category: "auth",
          code: "auth_rate_limited",
          message: "Too many sign-in attempts.",
          retryable: true
        }
      },
      { "Retry-After": "90" }
    );
  });

  await page.goto("/");
  await page.waitForResponse(/\/api\/v1\/profile$/);
  await clickSidebarSignIn(page);

  await expect(page.getByLabel("Email")).toBeFocused();
  await page.getByLabel("Email").fill("missing@example.com");
  await page.keyboard.press("Tab");
  await expect(page.getByLabel("Password")).toBeFocused();
  await page.getByLabel("Password").fill("bad-password");
  await page.keyboard.press("Tab");
  await expect(page.getByRole("form").getByRole("button", { name: "Sign in" })).toBeFocused();

  await page.getByRole("form").getByRole("button", { name: "Sign in" }).click();

  await expect(page.getByText("Email or password is incorrect.")).toBeVisible();
  await expect(page.getByText(/No account|email exists|account exists/i)).toHaveCount(0);
  await expect(page.getByLabel("Password")).toHaveValue("");

  await page.getByLabel("Password").fill("bad-password");
  await page.getByRole("form").getByRole("button", { name: "Sign in" }).click();
  await page.getByRole("button", { name: "Signing in..." }).click({ force: true });
  await expect(page.getByText("Too many sign-in attempts.")).toBeVisible();
  await expect(page.getByText("Try again in 90 seconds.")).toBeVisible();
  await expect(page.getByLabel("Password")).toHaveValue("");
  expect(loginAttempts).toBe(2);
});

test("login retry timing suppresses malformed values, clamps huge values, and supports HTTP-date metadata", async ({ page }) => {
  const fixedNow = Date.parse("2026-07-05T12:00:00Z");
  await page.addInitScript((now) => {
    Date.now = () => now;
  }, fixedNow);
  await stubAnonymousProfile(page);
  await stubCommonAuth(page);
  const retryHeaders = ["soon", "-5", "999999", new Date(fixedNow + 90_000).toUTCString()];
  let loginAttempts = 0;
  await page.route(/\/api\/v1\/auth\/login$/, async (route) => {
    const retryAfter = retryHeaders[loginAttempts] ?? "90";
    loginAttempts += 1;
    await fulfillJson(
      route,
      429,
      {
        status: "error",
        requestId: `retry-login-${loginAttempts}`,
        error: {
          category: "auth",
          code: "auth_rate_limited",
          message: "Too many sign-in attempts.",
          retryable: true
        }
      },
      { "Retry-After": retryAfter }
    );
  });

  await page.goto("/");
  await page.waitForResponse(/\/api\/v1\/profile$/);
  await clickSidebarSignIn(page);
  await page.getByLabel("Email").fill("locked@example.com");

  await page.getByLabel("Password").fill("bad-password");
  await page.getByRole("form").getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByText("Too many sign-in attempts.")).toBeVisible();
  await expect(page.getByText(/Try again in/)).toHaveCount(0);

  await page.getByLabel("Password").fill("bad-password");
  await page.getByRole("form").getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByText("Too many sign-in attempts.")).toBeVisible();
  await expect(page.getByText(/Try again in/)).toHaveCount(0);

  await page.getByLabel("Password").fill("bad-password");
  await page.getByRole("form").getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByText("Try again in 3600 seconds.")).toBeVisible();

  await page.getByLabel("Password").fill("bad-password");
  await page.getByRole("form").getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByText("Try again in 90 seconds.")).toBeVisible();
  expect(loginAttempts).toBe(4);
});

test("successful sidebar login creates a session and preserves search state after closing", async ({ page }) => {
  let authenticated = false;
  await page.route(/\/api\/v1\/profile$/, (route) => {
    if (!authenticated) {
      return fulfillJson(route, 401, {
        status: "error",
        requestId: "profile-anonymous",
        error: {
          category: "auth",
          code: "invalid_credentials",
          message: "Not signed in.",
          retryable: false
        }
      });
    }
    return fulfillJson(route, 200, profileEnvelope());
  });
  await stubCommonAuth(page);
  await page.route(/\/api\/v1\/auth\/login$/, (route) => {
    authenticated = true;
    return fulfillJson(route, 200, authSessionEnvelope());
  });

  await page.goto("/");
  await page.waitForResponse(/\/api\/v1\/profile$/);
  await page.locator("#autocomplete-input").fill("apple");

  await clickSidebarSignIn(page);
  await expect(page.getByRole("dialog", { name: "Sign in" })).toBeVisible();
  await page.mouse.click(5, 5);
  await expect(page.getByRole("dialog", { name: "Sign in" })).toHaveCount(0);
  await expect(page.locator("#autocomplete-input")).toHaveValue("apple");

  await clickSidebarSignIn(page);
  await page.getByLabel("Email").fill("user@example.com");
  await page.getByLabel("Password").fill("correct-password");
  await page.getByRole("form").getByRole("button", { name: "Sign in" }).click();
  await expect(page.locator("[data-sidebar-sign-out]")).toHaveCount(1);
  await expect(page.getByText("Signed in")).toHaveCount(0);
  await expect(page.locator("#autocomplete-input")).toHaveValue("apple");
});
