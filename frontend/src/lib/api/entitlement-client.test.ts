import { describe, it, expect, beforeEach, afterEach, mock, spyOn } from "bun:test";
import {
	fetchEntitlement,
	createCheckoutSession,
	buildEntitlementQueryOptions,
	EntitlementClientError,
	ENTITLEMENT_TIMEOUT_MS
} from "./entitlement-client";
import { createIdempotencyHeader } from "./generated";

// Implements DESIGN-001 SearchView TanStack Query entitlement client tests

describe("entitlement-client", () => {
	let originalFetch: typeof globalThis.fetch;

	beforeEach(() => {
		originalFetch = globalThis.fetch;
	});

	afterEach(() => {
		globalThis.fetch = originalFetch;
	});

	describe("fetchEntitlement", () => {
		it("fetches entitlement successfully with credentials", async () => {
			const mockData = {
				tier: "paid",
				status: "active",
				allowedModes: ["catalog", "substitution"],
				searchLimitPer24h: 100,
				usageRemaining: 90
			};
			globalThis.fetch = mock().mockResolvedValue({
				ok: true,
				status: 200,
				json: async () => ({
					status: "ok",
					requestId: "req-123",
					data: mockData
				})
			});

			const result = await fetchEntitlement();
			expect(result).toEqual(mockData as any);
			
			const fetchMock = globalThis.fetch as ReturnType<typeof mock>;
			expect(fetchMock).toHaveBeenCalledTimes(1);
			const args = fetchMock.mock.calls[0];
			expect(args[0]).toBe("/api/v1/entitlements");
			expect(args[1].credentials).toBe("include");
		});

		it("maps 401 anonymous requests to auth AppError", async () => {
			globalThis.fetch = mock().mockResolvedValue({
				ok: false,
				status: 401,
				json: async () => ({
					status: "error",
					requestId: "req-401",
					error: {
						category: "auth",
						code: "unauthorized",
						message: "Missing session",
						retryable: false
					}
				})
			});

			try {
				await fetchEntitlement();
				expect.unreachable("Should have thrown");
			} catch (error) {
				expect(error).toBeInstanceOf(EntitlementClientError);
				const clientError = error as EntitlementClientError;
				expect(clientError.status).toBe(401);
				expect(clientError.appError.category).toBe("auth");
				expect(clientError.appError.retryable).toBe(false);
			}
		});

		it("maps 503 errors correctly with retryable true", async () => {
			globalThis.fetch = mock().mockResolvedValue({
				ok: false,
				status: 503,
				json: async () => ({
					status: "error",
					requestId: "req-503"
				})
			});

			try {
				await fetchEntitlement();
				expect.unreachable("Should have thrown");
			} catch (error) {
				const clientError = error as EntitlementClientError;
				expect(clientError.status).toBe(503);
				expect(clientError.appError.category).toBe("dependency");
				expect(clientError.appError.retryable).toBe(true);
			}
		});

		it("maps 402 payment required to entitlement AppError", async () => {
			globalThis.fetch = mock().mockResolvedValue({
				ok: false,
				status: 402,
				json: async () => ({
					status: "error",
					requestId: "req-402"
				})
			});

			try {
				await fetchEntitlement();
				expect.unreachable("Should have thrown");
			} catch (error) {
				const clientError = error as EntitlementClientError;
				expect(clientError.status).toBe(402);
				expect(clientError.appError.category).toBe("entitlement");
				expect(clientError.appError.retryable).toBe(false);
			}
		});
	});

	describe("createCheckoutSession", () => {
		it("creates checkout session with idempotency key and credentials", async () => {
			const request = {
				priceId: "price_123",
				successUrl: "https://example.com/success",
				cancelUrl: "https://example.com/cancel"
			};
			const mockData = {
				sessionId: "cs_123",
				checkoutUrl: "https://checkout.stripe.com/c/pay/cs_123"
			};
			
			globalThis.fetch = mock().mockResolvedValue({
				ok: true,
				status: 200,
				json: async () => ({
					status: "ok",
					requestId: "req-123",
					data: mockData
				})
			});

			const result = await createCheckoutSession(request);
			expect(result).toEqual(mockData as any);
			
			const fetchMock = globalThis.fetch as ReturnType<typeof mock>;
			const args = fetchMock.mock.calls[0];
			expect(args[0]).toBe("/api/v1/billing/checkout");
			expect(args[1].method).toBe("POST");
			expect(args[1].credentials).toBe("include");
			expect(args[1].headers).toHaveProperty("Idempotency-Key");
			expect(args[1].body).toBe(JSON.stringify(request));
		});

		it("maps 409 conflict errors correctly", async () => {
			globalThis.fetch = mock().mockResolvedValue({
				ok: false,
				status: 409,
				json: async () => ({
					status: "error",
					requestId: "req-409"
				})
			});

			try {
				await createCheckoutSession({ priceId: "p", successUrl: "s", cancelUrl: "c" });
				expect.unreachable();
			} catch (error) {
				const clientError = error as EntitlementClientError;
				expect(clientError.status).toBe(409);
				expect(clientError.appError.category).toBe("server");
			}
		});
	});

	describe("buildEntitlementQueryOptions", () => {
		it("returns stable query keys and exact retry behavior", () => {
			const options = buildEntitlementQueryOptions();
			expect(options.queryKey).toEqual(["entitlement"]);
			
			const retryFn = options.retry as (failureCount: number, error: any) => boolean;
			expect(typeof retryFn).toBe("function");

			// 401 should not be retryable
			const authError = new EntitlementClientError({
				category: "auth",
				code: "unauthorized",
				message: "missing",
				retryable: false
			}, 401);
			expect(retryFn(1, authError)).toBe(false);

			// 503 should be retryable up to 3 times
			const dependencyError = new EntitlementClientError({
				category: "dependency",
				code: "dependency_unavailable",
				message: "down",
				retryable: true
			}, 503);
			expect(retryFn(1, dependencyError)).toBe(true);
			expect(retryFn(3, dependencyError)).toBe(false);
		});
	});
});
