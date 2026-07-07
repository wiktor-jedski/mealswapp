import { afterEach, expect, test } from "bun:test";

import type {
	AppError,
	AuthSessionEnvelope,
	CSRFTokenEnvelope,
	DisclaimerEnvelope,
	EntitlementStatusEnvelope,
	LoginRequest,
	ProfileEnvelope,
	RegisterRequest
} from "./generated";
import {
	AuthClientError,
	fetchCsrfToken,
	getOAuthStartUrl,
	loadDisclaimer,
	loginWithEmail,
	logoutCurrentSession,
	probeProfileSession,
	refreshAuthSession,
	refreshAuthStateAfterOAuthReturn,
	refreshEntitlementAfterAuth,
	registerWithEmail
} from "./auth-client";

// Implements DESIGN-018 AuthApiClient frontend wrapper verification.
// Implements DESIGN-017 ErrorMessageMapper auth status mapping verification.
// Verifies IT-ARCH-018-001, IT-ARCH-018-004, IT-ARCH-018-005, IT-ARCH-018-007, ARCH-018, DESIGN-018, SW-REQ-046, SW-REQ-058, SW-REQ-061, SW-REQ-064, SW-REQ-065, and SW-REQ-070.

type MockResponseProvider = (init: RequestInit) => Response | Promise<Response>;

class FetchMock {
	calls: Array<{ url: string; init: RequestInit }> = [];
	private providers: MockResponseProvider[] = [];
	private index = 0;

	enqueueResponse(response: Response): void {
		this.providers.push(() => response);
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
const fetchMock = new FetchMock();

afterEach(() => {
	globalThis.fetch = originalFetch;
	fetchMock.reset();
});

function jsonResponse(status: number, body: unknown, headers: Record<string, string> = {}): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { "Content-Type": "application/json", ...headers }
	});
}

function emptyJsonResponse(status: number): Response {
	return new Response("{}", {
		status,
		headers: { "Content-Type": "application/json" }
	});
}

function authSessionEnvelope(): AuthSessionEnvelope {
	return {
		status: "ok",
		requestId: "req-session",
		data: {
			userId: "user-1",
			role: "user",
			hasVerifiedLoginMethod: true,
			accessExpiresAt: "2026-07-05T10:00:00Z",
			refreshExpiresAt: "2026-07-12T10:00:00Z"
		}
	};
}

function profileEnvelope(): ProfileEnvelope {
	return {
		status: "ok",
		requestId: "req-profile",
		data: {
			userId: "user-1",
			displayName: "User One",
			unitSystem: "metric",
			themePreference: "system",
			requiresUnitRecalculation: false
		}
	};
}

function disclaimerEnvelope(): DisclaimerEnvelope {
	return {
		status: "ok",
		requestId: "req-disclaimer",
		data: {
			location: "login",
			version: "2026-07",
			markdown: "Medical disclaimer.",
			fallback: false
		}
	};
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
			usageUsed: 2,
			usageRemaining: 23,
			usageWindowStartedAt: "2026-07-05T00:00:00Z",
			trialExpiresAt: "2026-07-12T00:00:00Z",
			billingRecoveryState: "none"
		}
	};
}

function authError(status: number, code: string, category: AppError["category"], retryable: boolean): unknown {
	return {
		status: "error",
		requestId: `req-${status}`,
		error: {
			category,
			code,
			message: "Auth request failed.",
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

function bodyJson(init: RequestInit): Record<string, unknown> {
	if (typeof init.body !== "string") {
		throw new Error("FetchMock: request body was not a JSON string");
	}
	return JSON.parse(init.body) as Record<string, unknown>;
}

test("fetchCsrfToken requests the generated CSRF endpoint with credentials and decodes request context", async () => {
	globalThis.fetch = fetchMock.fetch as typeof fetch;
	const envelope: CSRFTokenEnvelope = {
		status: "ok",
		requestId: "req-csrf",
		data: { csrfToken: "csrf-token" }
	};
	fetchMock.enqueueResponse(jsonResponse(200, envelope));

	const context = await fetchCsrfToken(new AbortController().signal);

	expect(context).toEqual({ csrfToken: "csrf-token", requestId: "req-csrf" });
	const call = lastCall();
	expect(call.url).toBe("/api/v1/auth/csrf-token");
	expect(call.init.method).toBe("GET");
	expect(call.init.credentials).toBe("include");
	expect((call.init.headers as Record<string, string>).Accept).toBe("application/json");
});

test("registerWithEmail uses generated DTOs, CSRF header, credentialed POST, and clears caller password", async () => {
	globalThis.fetch = fetchMock.fetch as typeof fetch;
	fetchMock.enqueueResponse(jsonResponse(200, authSessionEnvelope()));
	const request: RegisterRequest = {
		email: "user@example.com",
		password: "correct horse battery staple",
		privacyPolicyVersion: "privacy-2026-07",
		termsVersion: "terms-2026-07"
	};

	const session = await registerWithEmail(request, { csrfToken: "csrf-token" });

	expect(session.userId).toBe("user-1");
	expect(request.password).toBe("");
	const call = lastCall();
	expect(call.url).toBe("/api/v1/auth/register");
	expect(call.init.method).toBe("POST");
	expect(call.init.credentials).toBe("include");
	expect((call.init.headers as Record<string, string>)["X-CSRF-Token"]).toBe("csrf-token");
	expect(bodyJson(call.init)).toEqual({
		email: "user@example.com",
		password: "correct horse battery staple",
		privacyPolicyVersion: "privacy-2026-07",
		termsVersion: "terms-2026-07"
	});
});

test("loginWithEmail uses generated LoginRequest, maps session envelope, and clears caller password", async () => {
	globalThis.fetch = fetchMock.fetch as typeof fetch;
	fetchMock.enqueueResponse(jsonResponse(200, authSessionEnvelope()));
	const request: LoginRequest = {
		email: "user@example.com",
		password: "raw-password"
	};

	const session = await loginWithEmail(request, { csrfToken: "csrf-token" });

	expect(session.hasVerifiedLoginMethod).toBe(true);
	expect(request.password).toBe("");
	const call = lastCall();
	expect(call.url).toBe("/api/v1/auth/login");
	expect(call.init.method).toBe("POST");
	expect(call.init.credentials).toBe("include");
	expect(bodyJson(call.init)).toEqual({ email: "user@example.com", password: "raw-password" });
});

test("auth mutation helpers clear passwords after failed submissions too", async () => {
	globalThis.fetch = fetchMock.fetch as typeof fetch;
	fetchMock.enqueueResponse(jsonResponse(401, authError(401, "invalid_credentials", "auth", false)));
	const request: LoginRequest = { email: "user@example.com", password: "bad-password" };

	await expect(loginWithEmail(request, { csrfToken: "csrf-token" })).rejects.toBeInstanceOf(AuthClientError);

	expect(request.password).toBe("");
});

test("logout, refresh, profile, disclaimer, and entitlement wrappers use generated endpoints and decoding", async () => {
	globalThis.fetch = fetchMock.fetch as typeof fetch;
	fetchMock.enqueueResponse(emptyJsonResponse(204));
	fetchMock.enqueueResponse(jsonResponse(200, authSessionEnvelope()));
	fetchMock.enqueueResponse(jsonResponse(200, profileEnvelope()));
	fetchMock.enqueueResponse(jsonResponse(200, disclaimerEnvelope()));
	fetchMock.enqueueResponse(jsonResponse(200, entitlementEnvelope()));

	await logoutCurrentSession({ csrfToken: "csrf-token" });
	const session = await refreshAuthSession();
	const profile = await probeProfileSession();
	const disclaimer = await loadDisclaimer("login");
	const entitlement = await refreshEntitlementAfterAuth();

	expect(session.userId).toBe("user-1");
	expect(profile.displayName).toBe("User One");
	expect(disclaimer.version).toBe("2026-07");
	expect(entitlement.usageRemaining).toBe(23);
	expect(fetchMock.calls.map((call) => call.url)).toEqual([
		"/api/v1/auth/logout",
		"/api/v1/auth/refresh",
		"/api/v1/profile",
		"/api/v1/disclaimers?location=login",
		"/api/v1/billing/entitlement"
	]);
	expect(fetchMock.calls[0]?.init.method).toBe("POST");
	expect((fetchMock.calls[0]?.init.headers as Record<string, string>)["X-CSRF-Token"]).toBe("csrf-token");
	for (const call of fetchMock.calls) {
		expect(call.init.credentials).toBe("include");
	}
});

test("getOAuthStartUrl returns generated provider start URLs without provider secrets", () => {
	expect(getOAuthStartUrl("google")).toBe("/api/v1/auth/oauth/google/start");
	expect(getOAuthStartUrl("apple")).toBe("/api/v1/auth/oauth/apple/start");
	expect(getOAuthStartUrl("google", "/subscription?plan=annual")).toBe(
		"/api/v1/auth/oauth/google/start?return_to=%2Fsubscription%3Fplan%3Dannual"
	);
	expect(getOAuthStartUrl("google")).not.toContain("client_secret");
});

test("refreshAuthStateAfterOAuthReturn coordinates session then entitlement refresh", async () => {
	globalThis.fetch = fetchMock.fetch as typeof fetch;
	fetchMock.enqueueResponse(jsonResponse(200, authSessionEnvelope()));
	fetchMock.enqueueResponse(jsonResponse(200, entitlementEnvelope()));

	const result = await refreshAuthStateAfterOAuthReturn();

	expect(result.session.userId).toBe("user-1");
	expect(result.entitlement.tier).toBe("trial");
	expect(fetchMock.calls.map((call) => call.url)).toEqual([
		"/api/v1/auth/refresh",
		"/api/v1/billing/entitlement"
	]);
});

test("session and profile decoding strip unexpected token strings from JavaScript-visible results", async () => {
	globalThis.fetch = fetchMock.fetch as typeof fetch;
	const sessionEnvelopeWithTokens = {
		...authSessionEnvelope(),
		data: {
			...authSessionEnvelope().data,
			accessToken: "access-token-secret",
			refreshToken: "refresh-token-secret"
		}
	};
	const profileEnvelopeWithTokens = {
		...profileEnvelope(),
		data: {
			...profileEnvelope().data,
			sessionToken: "session-token-secret"
		}
	};
	fetchMock.enqueueResponse(jsonResponse(200, sessionEnvelopeWithTokens));
	fetchMock.enqueueResponse(jsonResponse(200, profileEnvelopeWithTokens));

	const session = await refreshAuthSession();
	const profile = await probeProfileSession();

	expect("accessToken" in session).toBe(false);
	expect("refreshToken" in session).toBe(false);
	expect("sessionToken" in profile).toBe(false);
	expect(JSON.stringify({ session, profile })).not.toContain("token-secret");
});

test("auth client maps 400, 401, 403, 409, 429, and 503 envelopes to safe AppError values", async () => {
	globalThis.fetch = fetchMock.fetch as typeof fetch;
	const cases = [
		{ status: 400, code: "validation_failed", category: "validation", retryable: false },
		{ status: 401, code: "invalid_credentials", category: "auth", retryable: false },
		{ status: 403, code: "csrf_invalid", category: "security", retryable: false },
		{ status: 409, code: "duplicate_email", category: "validation", retryable: false },
		{ status: 429, code: "auth_rate_limited", category: "timeout", retryable: true },
		{ status: 503, code: "auth_unavailable", category: "dependency", retryable: true }
	] as const;
	for (const item of cases) {
		fetchMock.enqueueResponse(
			jsonResponse(
				item.status,
				authError(item.status, item.code, item.category, item.retryable),
				item.status === 429 ? { "Retry-After": "60" } : {}
			)
		);
	}

	for (const item of cases) {
		try {
			await refreshAuthSession();
			throw new Error("expected refreshAuthSession to throw");
		} catch (error) {
			const clientError = error as AuthClientError;
			expect(clientError).toBeInstanceOf(AuthClientError);
			expect(clientError.status).toBe(item.status);
			expect(clientError.appError.code).toBe(item.code);
			expect(clientError.appError.category).toBe(item.category);
			expect(clientError.appError.retryable).toBe(item.retryable);
			expect(clientError.appError.requestId).toBe(`req-${item.status}`);
			if (item.status === 429) {
				expect(clientError.retryAfterSeconds).toBe(60);
			}
		}
	}
});

test("auth client normalizes Retry-After seconds and HTTP-date values before exposing retry metadata", async () => {
	globalThis.fetch = fetchMock.fetch as typeof fetch;
	const originalDateNow = Date.now;
	const fixedNow = Date.parse("2026-07-05T12:00:00Z");
	Date.now = () => fixedNow;
	const cases = [
		{ header: "90 seconds", expected: undefined },
		{ header: "-5", expected: undefined },
		{ header: "999999", expected: 3600 },
		{ header: new Date(fixedNow + 90_000).toUTCString(), expected: 90 }
	] as const;
	for (const item of cases) {
		fetchMock.enqueueResponse(
			jsonResponse(429, authError(429, "auth_rate_limited", "timeout", true), { "Retry-After": item.header })
		);
	}

	try {
		for (const item of cases) {
			try {
				await refreshAuthSession();
				throw new Error("expected refreshAuthSession to throw");
			} catch (error) {
				const clientError = error as AuthClientError;
				expect(clientError).toBeInstanceOf(AuthClientError);
				expect(clientError.retryAfterSeconds).toBe(item.expected);
			}
		}
	} finally {
		Date.now = originalDateNow;
	}
});
