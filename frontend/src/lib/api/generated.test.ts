import { expect, test } from "bun:test";

import {
	BILLING_CHECKOUT_ENDPOINT,
	BILLING_ENTITLEMENT_ENDPOINT,
	buildCheckoutCreateRequestInit,
	buildEntitlementStatusRequestInit,
	type BillingErrorEnvelope,
	type CheckoutCreateRequest,
	type CheckoutSessionEnvelope,
	type EntitlementStatusEnvelope
} from "./generated";

// Implements DESIGN-017 ErrorMessageMapper generated billing contract verification.
test("generated billing contracts are importable for frontend entitlement gates", () => {
	const entitlement: EntitlementStatusEnvelope = {
		status: "ok",
		requestId: "req-entitlement",
		data: {
			userId: "00000000-0000-0000-0000-000000000001",
			tier: "trial",
			status: "active",
			allowedModes: ["catalog", "substitution", "daily_diet_alternative"],
			searchLimitPer24h: 0,
			usageUsed: 2,
			usageRemaining: null,
			usageWindowStartedAt: null,
			trialExpiresAt: "2026-07-09T00:00:00Z",
			billingRecoveryState: "none"
		}
	};
	const checkout: CheckoutSessionEnvelope = {
		status: "ok",
		requestId: "req-checkout",
		data: {
			checkoutSessionId: "cs_test_123",
			checkoutUrl: "https://checkout.stripe.com/c/test",
			plan: "monthly",
			priceId: "price_monthly",
			amountCents: 1200
		}
	};
	const billingError: BillingErrorEnvelope = {
		status: "error",
		requestId: "req-error",
		error: {
			category: "entitlement",
			code: "billing_payment_required",
			message: "Update billing to continue.",
			retryable: false
		}
	};

	expect(BILLING_ENTITLEMENT_ENDPOINT).toBe("/api/v1/billing/entitlement");
	expect(BILLING_CHECKOUT_ENDPOINT).toBe("/api/v1/billing/checkout");
	expect(entitlement.data.usageRemaining).toBeNull();
	expect(checkout.data.plan).toBe("monthly");
	expect(billingError.error.code).toBe("billing_payment_required");
});

// Implements DESIGN-007 SubscriptionController checkout idempotency helper verification.
test("generated checkout helper sends idempotency-aware request init", () => {
	const request: CheckoutCreateRequest = {
		plan: "annual",
		successUrl: "https://app.example/success",
		cancelUrl: "https://app.example/cancel"
	};

	const init = buildCheckoutCreateRequestInit(request, "checkout-key-123", { csrfToken: "csrf-token" });

	expect(init.method).toBe("POST");
	expect(init.credentials).toBe("include");
	expect(init.headers["Idempotency-Key"]).toBe("checkout-key-123");
	expect(init.headers["X-CSRF-Token"]).toBe("csrf-token");
	expect(init.body).toBe(JSON.stringify(request));
});

// Implements DESIGN-007 SubscriptionController entitlement request helper verification.
test("generated entitlement helper reads status with credentialed JSON headers", () => {
	const init = buildEntitlementStatusRequestInit();

	expect(init.method).toBe("GET");
	expect(init.credentials).toBe("include");
	expect(init.headers.Accept).toBe("application/json");
});
