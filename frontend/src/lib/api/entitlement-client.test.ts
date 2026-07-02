import { afterEach, expect, test } from "bun:test";
import { QueryClient, QueryObserver } from "@tanstack/query-core";
import { get } from "svelte/store";

import type {
	AppError,
	CheckoutCreateRequest,
	CheckoutSessionEnvelope,
	EntitlementStatusEnvelope
} from "./generated";
import {
	EntitlementClientError,
	buildCheckoutMutationOptions,
	buildEntitlementQueryOptions,
	createCheckoutSession,
	fetchEntitlementStatus,
	generateCheckoutIdempotencyKey
} from "./entitlement-client";
import {
	allowedSearchModesStore,
	entitlementErrorStore,
	entitlementStatusStore,
	resetEntitlementState,
	setEntitlementError,
	setEntitlementStatus,
	usageRemainingStore
} from "../stores/entitlement";

// Implements DESIGN-001 SearchView frontend entitlement client and TanStack Query state verification.
// Implements DESIGN-017 ErrorMessageMapper billing entitlement error mapping verification.

type MockResponseProvider = (init: RequestInit) => Response | Promise<Response>;

class FetchMock {
	calls: Array<{ url: string; init: RequestInit }> = [];
	private providers: MockResponseProvider[] = [];
	private index = 0;

	enqueueResponse(response: Response): void {
		this.providers.push(() => response);
	}

	enqueueProvider(provider: MockResponseProvider): void {
		this.providers.push(provider);
	}

	reset(): void {
		this.calls = [];
		this.providers = [];
		this.index = 0;
	}

	fetch = (input: string | URL | Request, init?: RequestInit): Promise<Response> => {
		const url = typeof input === "string" ? input : input.toString();
		const requestInit = init ?? {};
		this.calls.push({ url, init: requestInit });
		const provider = this.providers[this.index++];
		if (!provider) {
			throw new Error(`FetchMock: no response queued for ${url}`);
		}
		return Promise.resolve(provider(requestInit));
	};
}

const originalFetch = globalThis.fetch;
const originalCrypto = globalThis.crypto;
const fetchMock = new FetchMock();

afterEach(() => {
	globalThis.fetch = originalFetch;
	Object.defineProperty(globalThis, "crypto", {
		configurable: true,
		value: originalCrypto
	});
	fetchMock.reset();
	resetEntitlementState();
});

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { "Content-Type": "application/json" }
	});
}

function entitlementEnvelope(): EntitlementStatusEnvelope {
	return {
		status: "ok",
		requestId: "req-entitlement",
		data: {
			userId: "user-1",
			tier: "trial",
			status: "active",
			allowedModes: ["catalog", "substitution"],
			searchLimitPer24h: 25,
			usageUsed: 5,
			usageRemaining: 20,
			usageWindowStartedAt: "2026-07-02T00:00:00Z",
			trialExpiresAt: "2026-07-09T00:00:00Z",
			billingRecoveryState: "none"
		}
	};
}

function checkoutRequest(): CheckoutCreateRequest {
	return {
		plan: "monthly",
		successUrl: "https://app.example/success",
		cancelUrl: "https://app.example/cancel"
	};
}

function checkoutEnvelope(): CheckoutSessionEnvelope {
	return {
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
}

function billingError(status: number, code: string, category: AppError["category"], retryable: boolean): unknown {
	return {
		status: "error",
		requestId: `req-${status}`,
		error: {
			category,
			code,
			message: "Billing request failed.",
			retryable
		}
	};
}

function lastCall(): { url: string; init: RequestInit } {
	const call = fetchMock.calls[fetchMock.calls.length - 1];
	if (!call) {
		throw new Error("FetchMock: no fetch was recorded");
	}
	return call;
}

function setCryptoUuid(uuid: string): void {
	Object.defineProperty(globalThis, "crypto", {
		configurable: true,
		value: { randomUUID: () => uuid }
	});
}

async function tick(): Promise<void> {
	await new Promise<void>((resolve) => setTimeout(resolve, 0));
}

// Implements DESIGN-001 SearchView credentialed entitlement fetch verification.
test("fetchEntitlementStatus fetches generated entitlement endpoint with credentials", async () => {
	globalThis.fetch = fetchMock.fetch as typeof fetch;
	fetchMock.enqueueResponse(jsonResponse(200, entitlementEnvelope()));

	const status = await fetchEntitlementStatus(new AbortController().signal);

	expect(status.allowedModes).toEqual(["catalog", "substitution"]);
	expect(status.usageRemaining).toBe(20);
	const call = lastCall();
	expect(call.url).toBe("/api/v1/billing/entitlement");
	expect(call.init.method).toBe("GET");
	expect(call.init.credentials).toBe("include");
	expect((call.init.headers as { Accept: string }).Accept).toBe("application/json");
});

// Implements DESIGN-017 ErrorMessageMapper 401 anonymous entitlement handling verification.
test("fetchEntitlementStatus maps 401 anonymous users without retry recovery", async () => {
	globalThis.fetch = fetchMock.fetch as typeof fetch;
	fetchMock.enqueueResponse(jsonResponse(401, billingError(401, "anonymous_session", "auth", false)));

	try {
		await fetchEntitlementStatus();
		throw new Error("expected fetchEntitlementStatus to throw");
	} catch (error) {
		const clientError = error as EntitlementClientError;
		expect(clientError).toBeInstanceOf(EntitlementClientError);
		expect(clientError.status).toBe(401);
		expect(clientError.recoverable).toBe(false);
		expect(clientError.appError.category).toBe("auth");
		expect(clientError.appError.requestId).toBe("req-401");
	}
});

// Implements DESIGN-017 ErrorMessageMapper 402/409/503 billing mapping verification.
test("billing client maps recoverable 402, 409, and 503 statuses", async () => {
	globalThis.fetch = fetchMock.fetch as typeof fetch;
	fetchMock.enqueueResponse(jsonResponse(402, billingError(402, "billing_payment_required", "entitlement", false)));
	fetchMock.enqueueResponse(jsonResponse(409, billingError(409, "checkout_idempotency_conflict", "entitlement", false)));
	fetchMock.enqueueResponse(jsonResponse(503, billingError(503, "entitlement_unavailable", "dependency", true)));

	for (const expected of [
		{ status: 402, code: "billing_payment_required", category: "entitlement" },
		{ status: 409, code: "checkout_idempotency_conflict", category: "entitlement" },
		{ status: 503, code: "entitlement_unavailable", category: "dependency" }
	] as const) {
		try {
			if (expected.status === 409) {
				await createCheckoutSession(checkoutRequest(), { idempotencyKey: "checkout-fixed" });
			} else {
				await fetchEntitlementStatus();
			}
			throw new Error("expected billing request to throw");
		} catch (error) {
			const clientError = error as EntitlementClientError;
			expect(clientError.status).toBe(expected.status);
			expect(clientError.recoverable).toBe(true);
			expect(clientError.appError.code).toBe(expected.code);
			expect(clientError.appError.category).toBe(expected.category);
		}
	}
});

// Implements DESIGN-001 SearchView stable entitlement query-key verification.
test("buildEntitlementQueryOptions returns a stable current-user query key", () => {
	const optionsA = buildEntitlementQueryOptions();
	const optionsB = buildEntitlementQueryOptions();

	expect(optionsA.queryKey).toEqual(["billing-entitlement"]);
	expect(optionsA.queryKey).toEqual(optionsB.queryKey);
	expect(optionsA.retry).toBe(false);
});

// Implements DESIGN-001 SearchView TanStack Query entitlement state verification.
test("QueryClient stores entitlement status under the stable key", async () => {
	globalThis.fetch = fetchMock.fetch as typeof fetch;
	fetchMock.enqueueResponse(jsonResponse(200, entitlementEnvelope()));
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
	const options = buildEntitlementQueryOptions();
	const observer = new QueryObserver(queryClient, options);
	observer.subscribe(() => {});

	await queryClient.fetchQuery(options);
	await tick();

	const result = observer.getCurrentResult();
	expect(result.data?.tier).toBe("trial");
	expect(queryClient.getQueryData(options.queryKey)?.usageRemaining).toBe(20);
	queryClient.clear();
});

// Implements DESIGN-001 SearchView checkout idempotency-key generation verification.
test("generateCheckoutIdempotencyKey prefixes browser UUIDs for checkout requests", () => {
	setCryptoUuid("00000000-0000-4000-8000-000000000001");

	expect(generateCheckoutIdempotencyKey()).toBe("checkout-00000000-0000-4000-8000-000000000001");
});

// Implements DESIGN-001 SearchView checkout creation credential and idempotency verification.
test("createCheckoutSession sends generated checkout request with idempotency key", async () => {
	globalThis.fetch = fetchMock.fetch as typeof fetch;
	fetchMock.enqueueResponse(jsonResponse(200, checkoutEnvelope()));

	const checkout = await createCheckoutSession(checkoutRequest(), {
		idempotencyKey: "checkout-fixed",
		csrfToken: "csrf-token"
	});

	expect(checkout.checkoutSessionId).toBe("cs_test_123");
	const call = lastCall();
	expect(call.url).toBe("/api/v1/billing/checkout");
	expect(call.init.method).toBe("POST");
	expect(call.init.credentials).toBe("include");
	expect((call.init.headers as Record<string, string>)["Idempotency-Key"]).toBe("checkout-fixed");
	expect((call.init.headers as Record<string, string>)["X-CSRF-Token"]).toBe("csrf-token");
});

// Implements DESIGN-001 SearchView checkout retry behavior verification.
test("checkout mutation retries exactly one recoverable 503 and never retries 409 conflicts", () => {
	const options = buildCheckoutMutationOptions();
	const retry = options.retry as (failureCount: number, error: EntitlementClientError) => boolean;
	const unavailable = new EntitlementClientError(
		{ category: "dependency", code: "stripe_unavailable", message: "Try later.", retryable: true },
		503,
		true
	);
	const conflict = new EntitlementClientError(
		{ category: "entitlement", code: "checkout_idempotency_conflict", message: "Conflict.", retryable: false },
		409,
		true
	);

	expect(retry(0, unavailable)).toBe(true);
	expect(retry(1, unavailable)).toBe(false);
	expect(retry(0, conflict)).toBe(false);
});

// Implements DESIGN-001 SearchView checkout retry idempotency verification.
test("checkout mutation pins one generated idempotency key across retry invocations", async () => {
	globalThis.fetch = fetchMock.fetch as typeof fetch;
	setCryptoUuid("00000000-0000-4000-8000-000000000002");
	fetchMock.enqueueResponse(jsonResponse(503, billingError(503, "stripe_unavailable", "dependency", true)));
	fetchMock.enqueueResponse(jsonResponse(200, checkoutEnvelope()));
	const options = buildCheckoutMutationOptions();
	const variables = { request: checkoutRequest() };
	const mutationFn = options.mutationFn as (value: typeof variables) => Promise<unknown>;

	await expect(mutationFn(variables)).rejects.toBeInstanceOf(EntitlementClientError);
	await mutationFn(variables);

	expect(fetchMock.calls).toHaveLength(2);
	expect((fetchMock.calls[0]?.init.headers as Record<string, string>)["Idempotency-Key"]).toBe(
		"checkout-00000000-0000-4000-8000-000000000002"
	);
	expect((fetchMock.calls[1]?.init.headers as Record<string, string>)["Idempotency-Key"]).toBe(
		"checkout-00000000-0000-4000-8000-000000000002"
	);
});

// Implements DESIGN-001 SearchView entitlement store verification.
test("entitlement stores expose allowed modes, usage remaining, and recoverable errors", () => {
	const status = entitlementEnvelope().data;
	const error: AppError = {
		category: "entitlement",
		code: "billing_payment_required",
		message: "Update billing.",
		retryable: false
	};

	setEntitlementStatus(status);
	setEntitlementError(error);

	expect(get(entitlementStatusStore)).toBe(status);
	expect(get(allowedSearchModesStore)).toEqual(["catalog", "substitution"]);
	expect(get(usageRemainingStore)).toBe(20);
	expect(get(entitlementErrorStore)).toEqual(error);
});

// Implements DESIGN-001 SearchView generated-type-only billing DTO drift guard.
test("entitlement client imports generated billing DTOs instead of declaring duplicates", async () => {
	const source = await Bun.file(new URL("./entitlement-client.ts", import.meta.url)).text();

	for (const dto of [
		"CheckoutCreateRequest",
		"CheckoutSessionData",
		"EntitlementStatusData",
		"EntitlementStatusEnvelope"
	]) {
		expect(source).not.toMatch(new RegExp(`export\\s+interface\\s+${dto}\\b`));
		expect(source).not.toMatch(new RegExp(`export\\s+type\\s+${dto}\\b`));
	}
	expect(source).toContain('from "./generated"');
});
