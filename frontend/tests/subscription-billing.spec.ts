import { expect, test, type Page, type Route } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import type {
  CheckoutPlan,
  CheckoutSessionEnvelope,
  EntitlementState,
  EntitlementStatusEnvelope,
  SubscriptionTier
} from "../src/lib/api/generated";

// Implements DESIGN-007 SubscriptionController browser tests for hosted checkout and billing recovery UI.

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

async function fulfillJson(route: Route, status: number, body: unknown): Promise<void> {
  await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
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

async function stubCheckoutSuccess(page: Page): Promise<{ payloads: unknown[] }> {
  const payloads: unknown[] = [];
  await page.route(/\/api\/v1\/billing\/checkout$/, async (route) => {
    const body = route.request().postDataJSON() as { plan: CheckoutPlan };
    payloads.push(body);
    await fulfillJson(route, 200, checkoutEnvelope(body.plan));
  });
  return { payloads };
}

// Verifies IT-ARCH-007-006.
// Verifies ARCH-007.
// Verifies ARCH-001.
// Traces SW-REQ-044, SW-REQ-050, and SW-REQ-052.
// Verifies task 170 checkout contract creation and server-returned redirect behavior.
test("monthly and annual buttons create generated checkout contracts and follow server redirect URLs", async ({ page }) => {
  await stubEntitlement(page);
  const checkout = await stubCheckoutSuccess(page);

  await page.goto("/");
  await page.getByRole("button", { name: "Choose Monthly" }).click();
  await expect(page).toHaveURL("http://localhost:4173/stripe-hosted/monthly");

  await page.goto("/");
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
  expect(JSON.stringify(checkout.payloads)).not.toMatch(/pan|card|cvc|cvv|securityCode/i);
});

// Verifies task 170 loading and retry state behavior for checkout creation.
test("checkout loading and retry states are visible", async ({ page }) => {
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
  await page.getByRole("button", { name: "Choose Monthly" }).click();
  await expect(page.getByRole("button", { name: "Creating checkout..." })).toBeVisible();
  await expect(page.getByText("Stripe is temporarily unavailable.")).toBeVisible();
  await page.getByRole("button", { name: "Retry checkout" }).click();
  await expect(page).toHaveURL("http://localhost:4173/stripe-hosted/monthly");
  expect(checkoutAttempts).toBe(3);
});

// Verifies task 170 success and cancellation return routes refresh entitlement state.
test("success and cancel return routes refresh entitlement state", async ({ page }) => {
  const entitlement = await stubEntitlement(page, entitlementEnvelope({ tier: "paid" }));

  await page.goto("/billing/success?plan=monthly");
  await expect(page.getByText("Checkout completed. Billing access is refreshing.")).toBeVisible();
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
  await stubEntitlement(page);
  await page.goto("/");

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
