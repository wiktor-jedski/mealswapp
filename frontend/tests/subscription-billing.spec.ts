import { expect, test, type Page, type Route } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import type {
  AuthSessionEnvelope,
  CheckoutPlan,
  CheckoutSessionEnvelope,
  CSRFTokenEnvelope,
  EntitlementState,
  EntitlementStatusEnvelope,
  BillingPortalSessionEnvelope,
  ProfileEnvelope,
  SubscriptionTier
} from "../src/lib/api/generated";

// Implements DESIGN-007 SubscriptionController browser tests for hosted checkout and billing recovery UI.
// Implements DESIGN-018 AuthenticatedActionGuard billing workflow coverage composed with DESIGN-001 Subscription navigation.
// Verifies ARCH-018 and ARCH-001 browser workflow integration for task 185.

function entitlementEnvelope(
  overrides: Partial<EntitlementStatusEnvelope["data"]> = {}
): EntitlementStatusEnvelope {
  const status = overrides.status ?? "active";
  return {
    status: "ok",
    requestId: `entitlement-${status}`,
    data: {
      userId: "user-billing-1",
      tier: "free",
      status,
      allowedModes: ["catalog"],
      searchLimitPer24h: 3,
      usageUsed: 0,
      usageRemaining: 3,
      usageWindowStartedAt: "2026-07-02T08:00:00Z",
      trialExpiresAt: null,
      billingRecoveryState: "none",
      ...overrides
    }
  };
}

function checkoutEnvelope(plan: CheckoutPlan): CheckoutSessionEnvelope {
  return {
    status: "ok",
    requestId: `checkout-${plan}`,
    data: {
      checkoutSessionId: `cs_test_${plan}`,
      checkoutUrl: `http://localhost:4173/stripe-hosted/${plan}`,
      plan,
      priceId: `price_${plan}`,
      amountCents: plan === "monthly" ? 900 : 9000
    }
  };
}

function portalEnvelope(): BillingPortalSessionEnvelope {
  return {
    status: "ok",
    requestId: "billing-portal",
    data: {
      portalUrl: "http://localhost:4173/stripe-portal/session"
    }
  };
}

function authSessionEnvelope(): AuthSessionEnvelope {
  return {
    status: "ok",
    requestId: "billing-auth-session",
    data: {
      userId: "user-billing-1",
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
    requestId: "billing-profile",
    data: {
      userId: "user-billing-1",
      displayName: "Billing User",
      unitSystem: "metric",
      themePreference: "system",
      requiresUnitRecalculation: false
    }
  };
}

function csrfEnvelope(): CSRFTokenEnvelope {
  return {
    status: "ok",
    requestId: "billing-csrf",
    data: { csrfToken: "csrf-billing-checkout" }
  };
}

async function fulfillJson(route: Route, status: number, body: unknown): Promise<void> {
  await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

async function stubVerifiedAuth(page: Page): Promise<void> {
  await page.route(/\/api\/v1\/auth\/csrf-token$/, (route) => fulfillJson(route, 200, csrfEnvelope()));
  await page.route(/\/api\/v1\/profile$/, (route) => fulfillJson(route, 200, profileEnvelope()));
  await page.route(/\/api\/v1\/auth\/refresh$/, (route) => fulfillJson(route, 200, authSessionEnvelope()));
  await page.route(/\/api\/v1\/search-history$/, (route) =>
    fulfillJson(route, 200, { status: "ok", requestId: "billing-history", data: { history: [] } })
  );
  await page.route(/\/api\/v1\/saved-items\?kind=favorite$/, (route) =>
    fulfillJson(route, 200, { status: "ok", requestId: "billing-favorites", data: { items: [] } })
  );
}

async function waitForVerifiedAuth(page: Page): Promise<void> {
  await expect(page.locator("[data-sidebar-sign-out]")).toHaveCount(1);
  await expect(page.getByText("Signed in")).toHaveCount(0);
}

async function openSubscriptionView(page: Page): Promise<void> {
  const mobileToggle = page.getByLabel("Open activity sidebar");
  if (await mobileToggle.isVisible()) {
    await mobileToggle.click();
  }
  await page.getByRole("navigation", { name: "Account navigation" }).getByRole("button", { name: "Subscription" }).click();
  await expect(page.locator("[data-subscription-view]")).toBeVisible();
}

async function stubEntitlement(
  page: Page,
  envelope: EntitlementStatusEnvelope = entitlementEnvelope()
): Promise<{ requests: string[] }> {
  const requests: string[] = [];
  await page.route(/\/api\/v1\/billing\/entitlement$/, async (route) => {
    requests.push(route.request().url());
    await fulfillJson(route, 200, envelope);
  });
  return { requests };
}

async function stubCheckoutSuccess(page: Page): Promise<{ payloads: unknown[]; csrfHeaders: Array<string | null> }> {
  const payloads: unknown[] = [];
  const csrfHeaders: Array<string | null> = [];
  await page.route(/\/api\/v1\/billing\/checkout$/, async (route) => {
    const body = route.request().postDataJSON() as { plan: CheckoutPlan };
    payloads.push(body);
    csrfHeaders.push(route.request().headers()["x-csrf-token"] ?? null);
    await fulfillJson(route, 200, checkoutEnvelope(body.plan));
  });
  return { payloads, csrfHeaders };
}

async function stubPortalSuccess(page: Page): Promise<{ payloads: unknown[]; csrfHeaders: Array<string | null> }> {
  const payloads: unknown[] = [];
  const csrfHeaders: Array<string | null> = [];
  await page.route(/\/api\/v1\/billing\/portal$/, async (route) => {
    payloads.push(route.request().postDataJSON());
    csrfHeaders.push(route.request().headers()["x-csrf-token"] ?? null);
    await fulfillJson(route, 200, portalEnvelope());
  });
  return { payloads, csrfHeaders };
}

// Verifies IT-ARCH-007-006.
// Verifies ARCH-007.
// Verifies ARCH-001.
// Traces SW-REQ-044, SW-REQ-050, and SW-REQ-052.
// Verifies task 170 checkout contract creation and server-returned redirect behavior.
test("monthly and annual buttons create generated checkout contracts and follow server redirect URLs", async ({ page }) => {
  await stubVerifiedAuth(page);
  await stubEntitlement(page);
  const checkout = await stubCheckoutSuccess(page);

  await page.goto("/");
  await waitForVerifiedAuth(page);
  await openSubscriptionView(page);
  await page.getByRole("button", { name: "Choose Monthly" }).click();
  await expect(page).toHaveURL("http://localhost:4173/stripe-hosted/monthly");

  await page.goto("/");
  await waitForVerifiedAuth(page);
  await openSubscriptionView(page);
  await page.getByRole("button", { name: "Choose Annual" }).click();
  await expect(page).toHaveURL("http://localhost:4173/stripe-hosted/annual");

  expect(checkout.payloads).toHaveLength(2);
  expect(checkout.payloads).toEqual([
    {
      plan: "monthly",
      successUrl: "http://localhost:4173/billing/success?plan=monthly",
      cancelUrl: "http://localhost:4173/billing/cancel?plan=monthly"
    },
    {
      plan: "annual",
      successUrl: "http://localhost:4173/billing/success?plan=annual",
      cancelUrl: "http://localhost:4173/billing/cancel?plan=annual"
    }
  ]);
  expect(checkout.csrfHeaders).toEqual(["csrf-billing-checkout", "csrf-billing-checkout"]);
  expect(JSON.stringify(checkout.payloads)).not.toMatch(/pan|card|cvc|cvv|securityCode/i);
});

// Verifies task 170 billing entitlement reads stay bounded while the Subscription view is idle.
test("subscription view does not poll entitlement continuously while idle", async ({ page }) => {
  await stubVerifiedAuth(page);
  const entitlement = await stubEntitlement(page);

  await page.goto("/subscription");
  await waitForVerifiedAuth(page);
  await expect(page.locator("[data-subscription-view]")).toBeVisible();
  await expect(page.getByText("free · active")).toBeVisible();
  const requestsAfterRender = entitlement.requests.length;
  expect(requestsAfterRender).toBeLessThanOrEqual(2);
  await page.waitForTimeout(250);

  expect(entitlement.requests.length).toBe(requestsAfterRender);
});

// Verifies task 170 paid users use hosted billing portal instead of duplicate checkout.
test("paid active subscription shows portal management instead of duplicate checkout", async ({ page }) => {
  await stubVerifiedAuth(page);
  await stubEntitlement(
    page,
    entitlementEnvelope({
      tier: "paid",
      status: "active",
      allowedModes: ["catalog", "substitution", "daily_diet_alternative"],
      searchLimitPer24h: 0,
      usageRemaining: null
    })
  );
  const portal = await stubPortalSuccess(page);

  await page.goto("/subscription");
  await waitForVerifiedAuth(page);
  await expect(page.getByText("paid · active")).toBeVisible();
  await expect(page.getByRole("button", { name: "Choose Monthly" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Choose Annual" })).toHaveCount(0);
  await page.getByRole("button", { name: "Manage or cancel subscription" }).click();
  await expect(page).toHaveURL("http://localhost:4173/stripe-portal/session");
  expect(portal.payloads).toEqual([{ returnUrl: "http://localhost:4173/subscription" }]);
  expect(portal.csrfHeaders).toEqual(["csrf-billing-checkout"]);
});

// Verifies task 170 loading and retry state behavior for checkout creation.
test("checkout loading and shared error states are visible", async ({ page }) => {
  await stubVerifiedAuth(page);
  await stubEntitlement(page);
  let checkoutAttempts = 0;
  await page.route(/\/api\/v1\/billing\/checkout$/, async (route) => {
    checkoutAttempts += 1;
    if (checkoutAttempts <= 2) {
      await fulfillJson(route, 503, {
        status: "error",
        requestId: `stripe-down-${checkoutAttempts}`,
        error: {
          category: "dependency",
          code: "stripe_unavailable",
          message: "Stripe is temporarily unavailable.",
          retryable: true
        }
      });
      return;
    }
    await fulfillJson(route, 200, checkoutEnvelope("monthly"));
  });

  await page.goto("/");
  await waitForVerifiedAuth(page);
  await openSubscriptionView(page);
  await page.getByRole("button", { name: "Choose Monthly" }).click();
  await expect(page.getByRole("button", { name: "Creating checkout..." })).toBeVisible();
  await expect(page.getByText("Stripe is temporarily unavailable.")).toBeVisible();
  await expect(page.getByText("Stripe is temporarily unavailable.")).toHaveAttribute("role", "alert");
  await expect(page.getByRole("button", { name: "Retry checkout" })).toHaveCount(0);
  await page.getByRole("button", { name: "Choose Monthly" }).click();
  await expect(page).toHaveURL("http://localhost:4173/stripe-hosted/monthly");
  expect(checkoutAttempts).toBe(3);
});

// Verifies task 170 success and cancellation return routes refresh entitlement state.
test("success and cancel return routes refresh entitlement state", async ({ page }) => {
  await stubVerifiedAuth(page);
  const entitlement = await stubEntitlement(page, entitlementEnvelope({ tier: "paid" }));

  await page.goto("/billing/success?plan=monthly");
  await expect(page.getByText("Checkout completed. Billing access is active.")).toBeVisible();
  await expect(page.getByText("paid · active")).toBeVisible();

  await page.goto("/billing/cancel?plan=annual");
  await expect(page.getByText("Checkout was cancelled. Your current entitlement is unchanged.")).toBeVisible();
  await expect(page.getByText("paid · active")).toBeVisible();
  expect(entitlement.requests.length).toBeGreaterThanOrEqual(2);
});

// Verifies IT-ARCH-007-006.
// Verifies ARCH-007.
// Verifies ARCH-001.
// Traces SW-REQ-042, SW-REQ-052, and SW-REQ-053.
// Verifies task 170 past_due and cancelled billing recovery states.
for (const scenario of [
  { status: "past_due" as EntitlementState, tier: "paid" as SubscriptionTier, label: "Update billing" },
  { status: "cancelled" as EntitlementState, tier: "paid" as SubscriptionTier, label: "Restart billing" }
]) {
  test(`${scenario.status} state shows billing recovery action`, async ({ page }) => {
    await stubVerifiedAuth(page);
    await stubEntitlement(
      page,
      entitlementEnvelope({
        tier: scenario.tier,
        status: scenario.status,
        usageRemaining: null,
        billingRecoveryState: scenario.status === "cancelled" ? "cancelled" : "action_required"
      })
    );
    await stubCheckoutSuccess(page);

    await page.goto("/");
    await waitForVerifiedAuth(page);
    await openSubscriptionView(page);
    await expect(page.locator("[data-billing-recovery]")).toBeVisible();
    await expect(page.getByRole("button", { name: scenario.label })).toBeVisible();
  });
}

// Verifies IT-ARCH-007-006.
// Verifies ARCH-007.
// Verifies ARCH-001.
// Traces SW-REQ-044 and SW-REQ-052.
// Verifies task 170 accessibility and no raw payment-card capture in application UI.
test("subscription billing has no serious axe violations and no PAN or CVC fields", async ({ page }) => {
  await stubVerifiedAuth(page);
  await stubEntitlement(page);
  await page.goto("/");
  await waitForVerifiedAuth(page);
  await openSubscriptionView(page);

  const results = await new AxeBuilder({ page }).include("[data-subscription-billing]").analyze();
  const serious = results.violations.filter(
    (violation) => violation.impact === "critical" || violation.impact === "serious"
  );
  expect(serious, serious.map((violation) => `${violation.id}: ${violation.description}`).join("\n")).toEqual([]);

  await expect(
    page.locator(
      'input[name*="card" i], input[name*="pan" i], input[name*="cvc" i], input[name*="cvv" i], input[autocomplete="cc-number"], input[autocomplete="cc-csc"]'
    )
  ).toHaveCount(0);
});
