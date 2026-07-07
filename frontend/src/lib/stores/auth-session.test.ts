import { afterEach, expect, test } from "bun:test";
import { get } from "svelte/store";

import { AuthClientError } from "../api/auth-client";
import type { AppError, AuthSessionData, EntitlementStatusData, ProfileData } from "../api/generated";
import { entitlementErrorStore, entitlementStatusStore, resetEntitlementState } from "./entitlement";
import { resetSearch, searchStore, setQuery, submitSearch } from "./search";
import {
	authSessionStore,
	clearAuthSession,
	createInitialAuthSession,
	initAuthSessionStore,
	loginWithEmail,
	logoutCurrentSession,
	probeAuthSession,
	refreshAuthSessionAfterOAuthReturn,
	registerWithEmail,
	resetAuthSessionStore,
	setAuthSession,
	setAuthSessionDependencies
} from "./auth-session";

// Implements DESIGN-018 AuthSessionStore frontend-safe state transition verification.
// Verifies IT-ARCH-018-001, IT-ARCH-018-004, IT-ARCH-018-006, IT-ARCH-018-007, ARCH-018, DESIGN-018, SW-REQ-058, SW-REQ-061, SW-REQ-064, SW-REQ-066, and SW-REQ-070.

const originalWindow = globalThis.window;
const fixedNow = "2026-07-05T12:00:00.000Z";

afterEach(() => {
	if (originalWindow === undefined) {
		delete (globalThis as { window?: unknown }).window;
	} else {
		Object.defineProperty(globalThis, "window", {
			configurable: true,
			value: originalWindow
		});
	}
	resetAuthSessionStore();
	resetEntitlementState();
	resetSearch();
});

test("createInitialAuthSession starts with unknown state", () => {
	expect(createInitialAuthSession()).toEqual({ status: "unknown" });
	expect(get(authSessionStore)).toEqual({ status: "unknown" });
});

test("initAuthSessionStore does not trust stored authenticated projections before server verification", () => {
	const storage = new MapStorage();
	storage.setItem(
		"mealswapp.auth-session",
		JSON.stringify({ status: "authenticated", userId: "stored-user", hasVerifiedLoginMethod: true })
	);
	setWindowGlobals(storage);

	initAuthSessionStore();

	expect(get(authSessionStore)).toEqual({ status: "unknown" });
});

test("probeAuthSession stores only frontend-safe projection fields when authenticated", async () => {
	const storage = new MapStorage();
	setWindowGlobals(storage);
	setAuthSessionDependencies({
		now: () => fixedNow,
		probeProfileSession: async () =>
			({
				userId: "user-1",
				displayName: "User One",
				unitSystem: "metric",
				themePreference: "system",
				requiresUnitRecalculation: false,
				accessToken: "must-not-leak",
				password: "must-not-leak"
			}) as ProfileData,
		refreshAuthSession: async () =>
			({
				...sessionData("user-1"),
				accessToken: "must-not-leak",
				password: "must-not-leak"
			}) as AuthSessionData
	});

	const session = await probeAuthSession();

	expect(session).toEqual({
		status: "authenticated",
		userId: "user-1",
		displayName: "User One",
		role: "user",
		hasVerifiedLoginMethod: true,
		lastCheckedAt: fixedNow
	});
	expect(JSON.stringify(session)).not.toContain("must-not-leak");
	expect(storage.getItem("mealswapp.auth-session")).toBe(JSON.stringify(session));
});

test("probeAuthSession maps 401 to anonymous and session-expired semantics to expired", async () => {
	setAuthSessionDependencies({
		now: () => fixedNow,
		probeProfileSession: async () => {
			throw clientError(401, "invalid_credentials");
		}
	});
	await expect(probeAuthSession()).resolves.toMatchObject({ status: "anonymous" });

	setAuthSessionDependencies({
		now: () => fixedNow,
		probeProfileSession: async () => {
			throw clientError(401, "session_expired");
		}
	});
	await expect(probeAuthSession()).resolves.toMatchObject({ status: "expired" });
});

test("probeAuthSession maps lockout and unexpected failures to locked and error states", async () => {
	setAuthSessionDependencies({
		now: () => fixedNow,
		probeProfileSession: async () => {
			throw clientError(429, "auth_rate_limited");
		}
	});
	await expect(probeAuthSession()).resolves.toMatchObject({ status: "locked" });

	setAuthSessionDependencies({
		now: () => fixedNow,
		probeProfileSession: async () => {
			throw new Error("network down");
		}
	});
	await expect(probeAuthSession()).resolves.toMatchObject({ status: "error" });
});

test("probeAuthSession preserves an authenticated projection on transient probe failures", async () => {
	setAuthSession({
		status: "authenticated",
		userId: "user-1",
		role: "user",
		hasVerifiedLoginMethod: true,
		lastCheckedAt: "2026-07-05T11:00:00.000Z"
	});
	setAuthSessionDependencies({
		now: () => fixedNow,
		probeProfileSession: async () => {
			throw new Error("network down");
		}
	});

	await expect(probeAuthSession()).resolves.toEqual({
		status: "authenticated",
		userId: "user-1",
		role: "user",
		hasVerifiedLoginMethod: true,
		lastCheckedAt: fixedNow
	});
	expect(get(authSessionStore).status).toBe("authenticated");
});

test("probeAuthSession still clears authenticated projection on server auth failures", async () => {
	setAuthSession({ status: "authenticated", userId: "user-1", role: "user", hasVerifiedLoginMethod: true });
	setAuthSessionDependencies({
		now: () => fixedNow,
		probeProfileSession: async () => {
			throw clientError(401, "session_expired");
		}
	});

	await expect(probeAuthSession()).resolves.toMatchObject({ status: "expired" });
	expect(get(authSessionStore).status).toBe("expired");
});

test("loginWithEmail and registerWithEmail use CSRF, store authenticated state, and refresh entitlements", async () => {
	const csrfTokens: string[] = [];
	const entitlement = entitlementData();
	setAuthSessionDependencies({
		now: () => fixedNow,
		fetchCsrfToken: async () => ({ csrfToken: "csrf-token" }),
		loginWithEmail: async (request, options) => {
			csrfTokens.push(options.csrfToken ?? "");
			expect(request.password).toBe("raw-password");
			return sessionData("login-user");
		},
		registerWithEmail: async (_request, options) => {
			csrfTokens.push(options.csrfToken ?? "");
			return sessionData("registered-user");
		},
		refreshEntitlementAfterAuth: async () => entitlement
	});

	await expect(loginWithEmail({ email: "user@example.com", password: "raw-password" })).resolves.toMatchObject({
		status: "authenticated",
		userId: "login-user",
		role: "user",
		hasVerifiedLoginMethod: true
	});
	await expect(
		registerWithEmail({
			email: "new@example.com",
			password: "new-password",
			privacyPolicyVersion: "privacy-2026-07",
			termsVersion: "terms-2026-07"
		})
	).resolves.toMatchObject({ status: "authenticated", userId: "registered-user" });

	expect(csrfTokens).toEqual(["csrf-token", "csrf-token"]);
	expect(get(entitlementStatusStore)).toEqual(entitlement);
});

test("logout clears authenticated state while preserving anonymous Catalog Search state", async () => {
	setQuery("apple");
	submitSearch();
	setAuthSession({ status: "authenticated", userId: "user-1", role: "user" });
	setAuthSessionDependencies({
		now: () => fixedNow,
		fetchCsrfToken: async () => ({ csrfToken: "logout-csrf" }),
		logoutCurrentSession: async ({ csrfToken }) => {
			expect(csrfToken).toBe("logout-csrf");
		}
	});

	await logoutCurrentSession();

	expect(get(authSessionStore)).toEqual({ status: "anonymous", lastCheckedAt: fixedNow });
	expect(get(searchStore).query).toBe("apple");
	expect(get(searchStore).submittedQuery).toBe("apple");
	expect(get(searchStore).mode).toBe("catalog");
});

test("OAuth-return refresh ignores URL parameters and trusts only server session refresh", async () => {
	let refreshCalls = 0;
	setAuthSessionDependencies({
		now: () => fixedNow,
		refreshAuthSession: async () => {
			refreshCalls += 1;
			throw clientError(401, "session_expired");
		}
	});

	await expect(refreshAuthSessionAfterOAuthReturn("https://app.test/auth/callback?success=true")).rejects.toThrow();

	expect(refreshCalls).toBe(1);
	expect(get(authSessionStore).status).toBe("expired");
});

test("OAuth-return refresh stores authenticated projection and coordinates entitlement refresh", async () => {
	const entitlement = entitlementData();
	setAuthSessionDependencies({
		now: () => fixedNow,
		refreshAuthSession: async () => sessionData("oauth-user"),
		refreshEntitlementAfterAuth: async () => entitlement
	});

	await expect(refreshAuthSessionAfterOAuthReturn("https://app.test/auth/callback?code=abc")).resolves.toMatchObject({
		status: "authenticated",
		userId: "oauth-user"
	});
	expect(get(entitlementStatusStore)).toEqual(entitlement);
});

test("storage failures keep cookie-based auth usable without persisting tokens or passwords", async () => {
	const throwingStorage = {
		getItem(): string | null {
			throw new Error("denied");
		},
		setItem(): void {
			throw new Error("denied");
		},
		removeItem(): void {
			throw new Error("denied");
		}
	};
	setWindowGlobals(throwingStorage);
	setAuthSessionDependencies({
		now: () => fixedNow,
		refreshAuthSession: async () =>
			({
				...sessionData("cookie-user"),
				accessToken: "must-not-leak",
				refreshToken: "must-not-leak",
				password: "must-not-leak"
			}) as AuthSessionData,
		refreshEntitlementAfterAuth: async () => entitlementData()
	});

	expect(() => initAuthSessionStore()).not.toThrow();
	const projection = await refreshAuthSessionAfterOAuthReturn();

	expect(projection).toEqual({
		status: "authenticated",
		userId: "cookie-user",
		role: "user",
		hasVerifiedLoginMethod: true,
		lastCheckedAt: fixedNow
	});
	expect(JSON.stringify(projection)).not.toContain("must-not-leak");
});

test("entitlement refresh errors are captured without failing authenticated session transition", async () => {
	const entitlementError = appError("entitlement_unavailable", "dependency");
	setAuthSessionDependencies({
		now: () => fixedNow,
		fetchCsrfToken: async () => ({ csrfToken: "csrf-token" }),
		loginWithEmail: async () => sessionData("user-1"),
		refreshEntitlementAfterAuth: async () => {
			throw new AuthClientError(entitlementError, 503);
		}
	});

	await expect(loginWithEmail({ email: "user@example.com", password: "password" })).resolves.toMatchObject({
		status: "authenticated",
		userId: "user-1"
	});
	expect(get(entitlementErrorStore)).toEqual(entitlementError);
});

test("clearAuthSession removes user fields for anonymous and expired transitions", () => {
	setAuthSession({
		status: "authenticated",
		userId: "user-1",
		displayName: "User One",
		role: "admin",
		hasVerifiedLoginMethod: true
	});

	expect(clearAuthSession("anonymous")).toEqual({ status: "anonymous", lastCheckedAt: expect.any(String) });
	expect(clearAuthSession("expired")).toEqual({ status: "expired", lastCheckedAt: expect.any(String) });
});

function sessionData(userId: string): AuthSessionData {
	return {
		userId,
		role: "user",
		hasVerifiedLoginMethod: true,
		accessExpiresAt: "2026-07-05T13:00:00Z",
		refreshExpiresAt: "2026-07-12T13:00:00Z"
	};
}

function entitlementData(): EntitlementStatusData {
	return {
		userId: "user-1",
		tier: "trial",
		status: "active",
		allowedModes: ["catalog", "substitution"],
		searchLimitPer24h: 25,
		usageUsed: 1,
		usageRemaining: 24,
		usageWindowStartedAt: "2026-07-05T00:00:00Z",
		trialExpiresAt: "2026-07-12T00:00:00Z",
		billingRecoveryState: "none"
	};
}

function clientError(status: number, code: string): AuthClientError {
	return new AuthClientError(appError(code, status === 503 ? "dependency" : "auth"), status);
}

function appError(code: string, category: AppError["category"]): AppError {
	return {
		category,
		code,
		message: "Safe auth error.",
		retryable: category === "dependency"
	};
}

class MapStorage {
	private data = new Map<string, string>();
	getItem(key: string): string | null {
		return this.data.has(key) ? (this.data.get(key) as string) : null;
	}
	setItem(key: string, value: string): void {
		this.data.set(key, value);
	}
	removeItem(key: string): void {
		this.data.delete(key);
	}
}

function setWindowGlobals(storage: {
	getItem(key: string): string | null;
	setItem(key: string, value: string): void;
	removeItem(key: string): void;
}): void {
	Object.defineProperty(globalThis, "window", {
		configurable: true,
		value: { sessionStorage: storage }
	});
}
