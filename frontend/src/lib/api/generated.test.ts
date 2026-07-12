import { expect, test } from "bun:test";

import {
	AUTH_CSRF_TOKEN_ENDPOINT,
	AUTH_LOGIN_ENDPOINT,
	AUTH_LOGOUT_ENDPOINT,
	AUTH_REFRESH_ENDPOINT,
	AUTH_REGISTER_ENDPOINT,
	BILLING_CHECKOUT_ENDPOINT,
	BILLING_ENTITLEMENT_ENDPOINT,
	BILLING_PORTAL_ENDPOINT,
	DAILY_DIETS_ENDPOINT,
	DISCLAIMER_ENDPOINT,
	OPTIMIZATION_JOBS_ENDPOINT,
	PROFILE_ENDPOINT,
	buildBillingPortalCreateRequestInit,
	buildCsrfTokenRequestInit,
	buildDisclaimerRequestInit,
	buildDisclaimerUrl,
	buildCheckoutCreateRequestInit,
	buildDailyDietCreateRequestInit,
	buildEntitlementStatusRequestInit,
	buildLoginRequestInit,
	buildLogoutRequestInit,
	buildOAuthStartUrl,
	buildOptimizationJobRequestInit,
	buildOptimizationJobUrl,
	buildOptimizationSubmissionRequestInit,
	buildProfileRequestInit,
	buildRefreshSessionRequestInit,
	buildRegisterRequestInit,
	type BillingErrorEnvelope,
	type BillingPortalSessionEnvelope,
	type CheckoutCreateRequest,
	type CheckoutSessionEnvelope,
	type CSRFTokenEnvelope,
	type AuthSessionEnvelope,
	type DisclaimerEnvelope,
	type EntitlementStatusEnvelope,
	type DailyDietCreateRequest,
	type DietOptimizationRequest,
	type OptimizationJobAcknowledgementEnvelope,
	type OptimizationJobData,
	type OptimizationAlternative
} from "./generated";

// Implements DESIGN-018 AuthApiClient generated contract verification.
test("generated auth contracts are importable for the frontend auth surface", () => {
	const csrf: CSRFTokenEnvelope = {
		status: "ok",
		requestId: "req-csrf",
		data: {
			csrfToken: "csrf-token"
		}
	};
	const session: AuthSessionEnvelope = {
		status: "ok",
		requestId: "req-session",
		data: {
			userId: "00000000-0000-0000-0000-000000000001",
			role: "user",
			hasVerifiedLoginMethod: true,
			accessExpiresAt: "2026-07-05T10:00:00Z",
			refreshExpiresAt: "2026-07-12T10:00:00Z"
		}
	};
	const disclaimer: DisclaimerEnvelope = {
		status: "ok",
		requestId: "req-disclaimer",
		data: {
			location: "login",
			version: "2026-07",
			markdown: "Medical disclaimer.",
			fallback: false
		}
	};

	expect(AUTH_CSRF_TOKEN_ENDPOINT).toBe("/api/v1/auth/csrf-token");
	expect(AUTH_REGISTER_ENDPOINT).toBe("/api/v1/auth/register");
	expect(AUTH_LOGIN_ENDPOINT).toBe("/api/v1/auth/login");
	expect(AUTH_LOGOUT_ENDPOINT).toBe("/api/v1/auth/logout");
	expect(AUTH_REFRESH_ENDPOINT).toBe("/api/v1/auth/refresh");
	expect(PROFILE_ENDPOINT).toBe("/api/v1/profile");
	expect(DISCLAIMER_ENDPOINT).toBe("/api/v1/disclaimers");
	expect(csrf.data.csrfToken).toBe("csrf-token");
	expect(session.data.hasVerifiedLoginMethod).toBe(true);
	expect(disclaimer.data.location).toBe("login");
	expect(buildOAuthStartUrl("google")).toBe("/api/v1/auth/oauth/google/start");
	expect(buildOAuthStartUrl("google", "/subscription?plan=annual")).toBe(
		"/api/v1/auth/oauth/google/start?return_to=%2Fsubscription%3Fplan%3Dannual"
	);
	expect(buildOAuthStartUrl("google", "https://evil.test")).toBe("/api/v1/auth/oauth/google/start");
	expect(buildDisclaimerUrl("account")).toBe("/api/v1/disclaimers?location=account");
});

// Implements DESIGN-018 AuthApiClient generated request helper verification.
test("generated auth helpers build credentialed request init objects", () => {
	const csrfInit = buildCsrfTokenRequestInit();
	const registerInit = buildRegisterRequestInit(
		{
			email: "user@example.com",
			password: "correct horse battery staple",
			privacyPolicyVersion: "privacy-2026-07",
			termsVersion: "terms-2026-07"
		},
		{ csrfToken: "csrf-token" }
	);
	const loginInit = buildLoginRequestInit(
		{
			email: "user@example.com",
			password: "correct horse battery staple"
		},
		{ csrfToken: "csrf-token" }
	);
	const logoutInit = buildLogoutRequestInit({ csrfToken: "csrf-token" });
	const refreshInit = buildRefreshSessionRequestInit();
	const profileInit = buildProfileRequestInit();
	const disclaimerInit = buildDisclaimerRequestInit();

	expect(csrfInit.method).toBe("GET");
	expect(csrfInit.credentials).toBe("include");
	expect(registerInit.method).toBe("POST");
	expect(registerInit.credentials).toBe("include");
	expect(registerInit.headers["X-CSRF-Token"]).toBe("csrf-token");
	expect(JSON.parse(registerInit.body)).toEqual({
		email: "user@example.com",
		password: "correct horse battery staple",
		privacyPolicyVersion: "privacy-2026-07",
		termsVersion: "terms-2026-07"
	});
	expect(loginInit.headers["X-CSRF-Token"]).toBe("csrf-token");
	expect(JSON.parse(loginInit.body)).toEqual({
		email: "user@example.com",
		password: "correct horse battery staple"
	});
	expect(logoutInit.method).toBe("POST");
	expect(logoutInit.credentials).toBe("include");
	expect(logoutInit.headers["X-CSRF-Token"]).toBe("csrf-token");
	expect(refreshInit.method).toBe("POST");
	expect(refreshInit.credentials).toBe("include");
	expect(profileInit.method).toBe("GET");
	expect(profileInit.credentials).toBe("include");
	expect(disclaimerInit.headers.Accept).toBe("application/json");
});

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
	const portal: BillingPortalSessionEnvelope = {
		status: "ok",
		requestId: "req-portal",
		data: {
			portalUrl: "https://billing.stripe.com/p/session"
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
	expect(BILLING_PORTAL_ENDPOINT).toBe("/api/v1/billing/portal");
	expect(entitlement.data.usageRemaining).toBeNull();
	expect(checkout.data.plan).toBe("monthly");
	expect(portal.data.portalUrl).toContain("billing.stripe.com");
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

// Implements DESIGN-007 SubscriptionController billing portal helper verification.
test("generated billing portal helper sends credentialed CSRF request init", () => {
	const request = { returnUrl: "https://app.example/subscription" };
	const init = buildBillingPortalCreateRequestInit(request, { csrfToken: "csrf-token" });

	expect(init.method).toBe("POST");
	expect(init.credentials).toBe("include");
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

// Implements DESIGN-004 JobStatusTracker and DESIGN-008 SavedDataRepository generated contract verification.
test("generated daily-diet and optimization contracts enforce protected request shapes", () => {
	const dietRequest: DailyDietCreateRequest = {
		name: "Training day",
		entries: [{ mealId: "00000000-0000-0000-0000-000000000001", quantity: 100, unit: "g", position: 0 }]
	};
	const optimizationRequest: DietOptimizationRequest = {
		dailyDietId: "00000000-0000-0000-0000-000000000002",
		targetMacros: { protein: 120, carbohydrates: 180, fat: 60 },
		tolerancePercent: 10,
		 excludedMealIds: []
	};
	const alternative: OptimizationAlternative = {
		meals: [{ mealId: "00000000-0000-0000-0000-000000000004", quantity: 100, unit: "g", position: 0 }],
		macros: { protein: 120, carbohydrates: 180, fat: 60, calories: 1740 },
		similarityScore: 0.9
	};
	const completedJob: OptimizationJobData = {
		jobId: "00000000-0000-0000-0000-000000000005",
		dailyDietId: "00000000-0000-0000-0000-000000000002",
		status: "completed",
		pollUrl: "/api/v1/optimization/jobs/00000000-0000-0000-0000-000000000005",
		createdAt: "2026-07-11T00:00:00Z",
		startedAt: "2026-07-11T00:00:01Z",
		finishedAt: "2026-07-11T00:00:02Z",
		alternatives: [alternative]
	};
	const failedJob: OptimizationJobData = {
		jobId: "00000000-0000-0000-0000-000000000006",
		dailyDietId: "00000000-0000-0000-0000-000000000002",
		status: "failed",
		pollUrl: "/api/v1/optimization/jobs/00000000-0000-0000-0000-000000000006",
		createdAt: "2026-07-11T00:00:00Z",
		failure: { code: "solver_timeout", message: "The optimization took too long." }
	};
	const queuedJob: OptimizationJobData = {
		jobId: "00000000-0000-0000-0000-000000000007",
		dailyDietId: "00000000-0000-0000-0000-000000000002",
		status: "queued",
		pollUrl: "/api/v1/optimization/jobs/00000000-0000-0000-0000-000000000007",
		createdAt: "2026-07-11T00:00:00Z"
	};
	const processingJob: OptimizationJobData = {
		...queuedJob,
		status: "processing",
		startedAt: "2026-07-11T00:00:01Z"
	};
	const cancelledJob: OptimizationJobData = {
		...queuedJob,
		status: "cancelled",
		finishedAt: "2026-07-11T00:00:01Z"
	};
	// @ts-expect-error Queued jobs cannot expose alternatives.
	const invalidQueuedJob: OptimizationJobData = { ...queuedJob, alternatives: [] };
	// @ts-expect-error Completed jobs require alternatives.
	const invalidCompletedJob: OptimizationJobData = { ...completedJob, alternatives: [] };
	// @ts-expect-error Completed jobs cannot expose a failure.
	const invalidCompletedJobWithFailure: OptimizationJobData = {
		...completedJob,
		failure: { code: "worker_crash", message: "Optimization failed." }
	};
	// @ts-expect-error Failed jobs require a safe failure.
	const invalidFailedJob: OptimizationJobData = {
		jobId: failedJob.jobId,
		dailyDietId: failedJob.dailyDietId,
		status: "failed",
		pollUrl: failedJob.pollUrl,
		createdAt: failedJob.createdAt
	};
	const acknowledgement: OptimizationJobAcknowledgementEnvelope = {
		status: "accepted",
		requestId: "req-optimization",
		data: { jobId: "00000000-0000-0000-0000-000000000003", status: "queued", pollUrl: "/api/v1/optimization/jobs/00000000-0000-0000-0000-000000000003" }
	};
	const dietInit = buildDailyDietCreateRequestInit(dietRequest, "diet-key-123", { csrfToken: "csrf-token" });
	const submitInit = buildOptimizationSubmissionRequestInit(optimizationRequest, "optimization-key-123", { csrfToken: "csrf-token" });
	const pollInit = buildOptimizationJobRequestInit();

	expect(DAILY_DIETS_ENDPOINT).toBe("/api/v1/daily-diets");
	expect(OPTIMIZATION_JOBS_ENDPOINT).toBe("/api/v1/optimization/jobs");
	expect(buildOptimizationJobUrl("job/1")).toBe("/api/v1/optimization/jobs/job%2F1");
	expect(dietInit.headers["Idempotency-Key"]).toBe("diet-key-123");
	expect(submitInit.headers["X-CSRF-Token"]).toBe("csrf-token");
	expect(pollInit.credentials).toBe("include");
	expect(acknowledgement.data?.status).toBe("queued");
	expect(completedJob.status).toBe("completed");
	expect(completedJob.alternatives).toHaveLength(1);
	expect(failedJob.failure.code).toBe("solver_timeout");
	expect(queuedJob.status).toBe("queued");
	expect(processingJob.status).toBe("processing");
	expect(cancelledJob.status).toBe("cancelled");
	expect(invalidQueuedJob.status).toBe("queued");
	expect(invalidCompletedJob.status).toBe("completed");
	expect(invalidCompletedJobWithFailure.status).toBe("completed");
	expect(invalidFailedJob.status).toBe("failed");
});
