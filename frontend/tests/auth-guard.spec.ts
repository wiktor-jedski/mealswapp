import { expect, test, type Page, type Route } from "@playwright/test";
import type {
	AuthSessionEnvelope,
	CSRFTokenEnvelope,
	EntitlementStatusEnvelope,
	ProfileEnvelope,
	SearchResponseEnvelope
} from "../src/lib/api/generated";

// Implements DESIGN-018 AuthenticatedActionGuard browser verification for protected checkout and entitlement actions.
// Verifies IT-ARCH-018-003, IT-ARCH-018-006, IT-ARCH-018-007, ARCH-018, DESIGN-018, SW-REQ-044, SW-REQ-058, SW-REQ-061, and SW-REQ-066.

function csrfEnvelope(): CSRFTokenEnvelope {
	return {
		status: "ok",
		requestId: "auth-guard-csrf",
		data: { csrfToken: "csrf-auth-guard" }
	};
}

function authSessionEnvelope(hasVerifiedLoginMethod = true): AuthSessionEnvelope {
	return {
		status: "ok",
		requestId: "auth-guard-session",
		data: {
			userId: "user-auth-guard-1",
			role: "user",
			hasVerifiedLoginMethod,
			accessExpiresAt: "2026-07-05T13:00:00Z",
			refreshExpiresAt: "2026-07-12T13:00:00Z"
		}
	};
}

function profileEnvelope(): ProfileEnvelope {
	return {
		status: "ok",
		requestId: "auth-guard-profile",
		data: {
			userId: "user-auth-guard-1",
			displayName: "Auth Guard User",
			unitSystem: "metric",
			themePreference: "system",
			requiresUnitRecalculation: false
		}
	};
}

function entitlementEnvelope(): EntitlementStatusEnvelope {
	return {
		status: "ok",
		requestId: "auth-guard-entitlement",
		data: {
			userId: "user-auth-guard-1",
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

function searchEnvelope(): SearchResponseEnvelope {
	return {
		status: "ok",
		requestId: "auth-guard-search",
		data: {
			items: [
				{
					id: "food-apple",
					name: "Apple",
					physicalState: "solid",
					imageUrl: null,
					classifications: [{ id: "fruit", name: "Fruit", kind: "food_category" }],
					primaryFoodCategory: { id: "fruit", name: "Fruit", kind: "food_category" },
					macros: { protein: 0.3, carbohydrates: 14, fat: 0.2 },
					macroBasis: "100g",
					calories: 52
				}
			],
			totalCount: 1,
			page: 1,
			similarityScores: [1],
			similarityMetadata: [],
			warnings: []
		}
	};
}

async function fulfillJson(route: Route, status: number, body: unknown): Promise<void> {
	await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
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
			requestId: "auth-guard-profile-anonymous",
			error: {
				category: "auth",
				code: "anonymous_session",
				message: "Please sign in.",
				retryable: false
			}
		})
	);
}

async function stubExpiredProfile(page: Page): Promise<void> {
	await page.route(/\/api\/v1\/profile$/, (route) =>
		fulfillJson(route, 401, {
			status: "error",
			requestId: "auth-guard-profile-expired",
			error: {
				category: "auth",
				code: "session_expired",
				message: "Session expired.",
				retryable: false
			}
		})
	);
}

async function stubVerifiedProfile(page: Page, hasVerifiedLoginMethod = true): Promise<void> {
	await page.route(/\/api\/v1\/profile$/, (route) => fulfillJson(route, 200, profileEnvelope()));
	await page.route(/\/api\/v1\/auth\/refresh$/, (route) =>
		fulfillJson(route, 200, authSessionEnvelope(hasVerifiedLoginMethod))
	);
}

async function stubAuthMutations(page: Page): Promise<void> {
	await page.route(/\/api\/v1\/auth\/csrf-token$/, (route) => fulfillJson(route, 200, csrfEnvelope()));
	await page.route(/\/api\/v1\/auth\/login$/, (route) => fulfillJson(route, 200, authSessionEnvelope()));
	await page.route(/\/api\/v1\/auth\/register$/, (route) => fulfillJson(route, 201, authSessionEnvelope()));
	await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => fulfillJson(route, 200, entitlementEnvelope()));
	await page.route(/\/api\/v1\/search-history$/, (route) =>
		fulfillJson(route, 200, { status: "ok", requestId: "auth-guard-history", data: { history: [] } })
	);
	await page.route(/\/api\/v1\/saved-items\?kind=favorite$/, (route) =>
		fulfillJson(route, 200, { status: "ok", requestId: "auth-guard-favorites", data: { items: [] } })
	);
}

async function stubCatalogSearch(page: Page): Promise<void> {
	await page.route(/\/api\/v1\/search$/, (route) => fulfillJson(route, 200, searchEnvelope()));
	await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) =>
		fulfillJson(route, 200, { status: "ok", requestId: "auth-guard-autocomplete", data: { items: [] } })
	);
}

async function stubCheckout(page: Page): Promise<{ attempts: () => number }> {
	let checkoutAttempts = 0;
	await page.route(/\/api\/v1\/billing\/checkout$/, async (route) => {
		checkoutAttempts += 1;
		await fulfillJson(route, 200, {
			status: "ok",
			requestId: "auth-guard-checkout",
			data: {
				checkoutSessionId: "cs_auth_guard",
				checkoutUrl: "http://localhost:4173/stripe-hosted/auth-guard",
				plan: route.request().postDataJSON().plan,
				priceId: "price_auth_guard",
				amountCents: 900
			}
		});
	});
	return { attempts: () => checkoutAttempts };
}

async function trackEntitlementRefresh(page: Page): Promise<{ attempts: () => number }> {
	let entitlementAttempts = 0;
	await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => {
		entitlementAttempts += 1;
		return fulfillJson(route, 200, entitlementEnvelope());
	});
	return { attempts: () => entitlementAttempts };
}

async function trackSidebarProtectedEndpoints(page: Page): Promise<{ attempts: () => number }> {
	let sidebarAttempts = 0;
	await page.route(/\/api\/v1\/search-history$/, (route) => {
		sidebarAttempts += 1;
		return fulfillJson(route, 200, { status: "ok", requestId: "auth-guard-history", data: { history: [] } });
	});
	await page.route(/\/api\/v1\/saved-items\?kind=favorite$/, (route) => {
		sidebarAttempts += 1;
		return fulfillJson(route, 200, { status: "ok", requestId: "auth-guard-favorites", data: { items: [] } });
	});
	return { attempts: () => sidebarAttempts };
}

async function fillRegistration(page: Page): Promise<void> {
	await page.getByLabel("Email").fill("person@example.com");
	await page.getByLabel("Password", { exact: true }).fill("correct-horse-1");
	await page.getByLabel("Confirm password").fill("correct-horse-1");
	await page.getByLabel("I accept the current Privacy Policy and Terms of Service.").check();
}

test("unknown sessions do not automatically call protected entitlement refresh", async ({ page }) => {
	const entitlement = await trackEntitlementRefresh(page);
	await stubCatalogSearch(page);
	await page.route(/\/api\/v1\/profile$/, async (route) => {
		await new Promise((resolve) => setTimeout(resolve, 1_000));
		await fulfillJson(route, 401, {
			status: "error",
			requestId: "auth-guard-profile-delayed",
			error: {
				category: "auth",
				code: "anonymous_session",
				message: "Please sign in.",
				retryable: false
			}
		});
	});

	await page.goto("/");
	await page.waitForTimeout(250);

	expect(entitlement.attempts()).toBe(0);
});

test("unknown sessions do not load protected sidebar activity", async ({ page }) => {
	const sidebar = await trackSidebarProtectedEndpoints(page);
	await stubCatalogSearch(page);
	await page.route(/\/api\/v1\/profile$/, async (route) => {
		await new Promise((resolve) => setTimeout(resolve, 1_000));
		await fulfillJson(route, 401, {
			status: "error",
			requestId: "auth-guard-sidebar-profile-delayed",
			error: {
				category: "auth",
				code: "anonymous_session",
				message: "Please sign in.",
				retryable: false
			}
		});
	});

	await page.goto("/");
	await page.waitForTimeout(250);

	expect(sidebar.attempts()).toBe(0);
});

test("anonymous sessions do not automatically call protected entitlement refresh", async ({ page }) => {
	const entitlement = await trackEntitlementRefresh(page);
	await stubAnonymousProfile(page);
	await stubCatalogSearch(page);

	await page.goto("/");
	await page.waitForResponse(/\/api\/v1\/profile$/);
	await page.waitForTimeout(250);

	expect(entitlement.attempts()).toBe(0);
});

test("anonymous sessions do not load protected sidebar activity", async ({ page }) => {
	const sidebar = await trackSidebarProtectedEndpoints(page);
	await stubAnonymousProfile(page);
	await stubCatalogSearch(page);

	await page.goto("/");
	await page.waitForResponse(/\/api\/v1\/profile$/);
	await page.waitForTimeout(250);

	expect(sidebar.attempts()).toBe(0);
});

test("authenticated but unverified sessions do not load protected sidebar activity", async ({ page }) => {
	const sidebar = await trackSidebarProtectedEndpoints(page);
	await stubVerifiedProfile(page, false);
	await stubCatalogSearch(page);
	await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => fulfillJson(route, 200, entitlementEnvelope()));

	await page.goto("/");
	await page.waitForResponse(/\/api\/v1\/auth\/refresh$/);
	await page.waitForTimeout(250);

	expect(sidebar.attempts()).toBe(0);
});

test("verified authenticated sessions load protected sidebar activity", async ({ page }) => {
	const sidebar = await trackSidebarProtectedEndpoints(page);
	await stubVerifiedProfile(page);
	await stubCatalogSearch(page);
	await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => fulfillJson(route, 200, entitlementEnvelope()));

	await page.goto("/");
	await page.waitForResponse(/\/api\/v1\/auth\/refresh$/);
	await expect.poll(() => sidebar.attempts()).toBe(2);
});

test("anonymous Catalog Search stays usable while Subscription navigation is guarded", async ({ page }) => {
	await stubAnonymousProfile(page);
	await stubCatalogSearch(page);
	await page.route(/\/api\/v1\/billing\/entitlement$/, (route) =>
		fulfillJson(route, 401, {
			status: "error",
			requestId: "auth-guard-entitlement-anonymous",
			error: {
				category: "auth",
				code: "anonymous_session",
				message: "Please sign in.",
				retryable: false
			}
		})
	);
	const checkout = await stubCheckout(page);

	await page.goto("/");
	await page.waitForResponse(/\/api\/v1\/profile$/);
	await page.getByLabel("Food search").fill("apple");
	await page.getByLabel("Food search").press("Enter");
	await expect(page.locator("[data-result-card]")).toHaveCount(1);

	await expect(page.getByRole("navigation", { name: "Account navigation" })).toHaveCount(0);
	await expect(page.getByRole("button", { name: "Subscription" })).toHaveCount(0);
	await page.goto("/subscription");
	await expect(page.locator("[data-auth-guidance]")).toContainText("open subscription");
	await expect(page.locator("[data-subscription-view]")).toHaveCount(0);

	expect(checkout.attempts()).toBe(0);
	await expect(
		page.locator(
			'input[name*="card" i], input[name*="pan" i], input[name*="cvc" i], input[name*="cvv" i], input[autocomplete="cc-number"], input[autocomplete="cc-csc"]'
		)
	).toHaveCount(0);
});

test("expired sessions clear frontend-safe auth state before guarded Subscription navigation", async ({ page }) => {
	await page.addInitScript(() => {
		sessionStorage.setItem(
			"mealswapp.auth-session",
			JSON.stringify({ status: "authenticated", userId: "stale-user", displayName: "Stale User" })
		);
	});
	await stubExpiredProfile(page);
	await stubCatalogSearch(page);
	await page.route(/\/api\/v1\/billing\/entitlement$/, (route) =>
		fulfillJson(route, 401, {
			status: "error",
			requestId: "auth-guard-entitlement-expired",
			error: {
				category: "auth",
				code: "session_expired",
				message: "Session expired.",
				retryable: false
			}
		})
	);
	await stubCheckout(page);

	await page.goto("/");
	await page.waitForResponse(/\/api\/v1\/profile$/);
	await page.goto("/subscription");

	await expect(page.locator("[data-auth-guidance]")).toContainText("open subscription");
	await expect
		.poll(() => page.evaluate(() => sessionStorage.getItem("mealswapp.auth-session")))
		.toBeNull();
});

test("successful registration retries queued Subscription navigation with the cookie-backed session", async ({ page }) => {
	await stubAnonymousProfile(page);
	await stubCatalogSearch(page);
	await stubAuthMutations(page);
	const checkout = await stubCheckout(page);

	await page.goto("/subscription");
	await page.waitForResponse(/\/api\/v1\/profile$/);
	await page.getByRole("button", { name: "Create account" }).click();
	await fillRegistration(page);
	await page.getByRole("button", { name: "Create account" }).last().click();

	await expect(page.locator("[data-subscription-view]")).toBeVisible();
	await expect(page).not.toHaveURL(/stripe-hosted/);
	expect(checkout.attempts()).toBe(0);
});

test("canceling auth clears queued Subscription navigation", async ({ page }) => {
	await stubAnonymousProfile(page);
	await stubCatalogSearch(page);
	await stubAuthMutations(page);
	const checkout = await stubCheckout(page);

	await page.goto("/subscription");
	await page.waitForResponse(/\/api\/v1\/profile$/);
	await page.mouse.click(5, 5);
	await clickSidebarSignIn(page);
	await page.getByLabel("Email").fill("user@example.com");
	await page.getByLabel("Password").fill("correct-password");
	await page.getByRole("form").getByRole("button", { name: "Sign in" }).click();

	await expect(page).not.toHaveURL(/stripe-hosted/);
	await expect(page.locator("[data-subscription-view]")).toHaveCount(0);
	expect(checkout.attempts()).toBe(0);
});
