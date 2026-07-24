import { expect, test, type Page, type Route } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import type { AuthSessionEnvelope, EntitlementStatusEnvelope, ProfileEnvelope } from "../src/lib/api/generated";

// Implements DESIGN-009 UserAdminPanel browser verification for task 254.

type Role = "user" | "admin";

function sessionEnvelope(userId: string, role: Role): AuthSessionEnvelope {
	return {
		status: "ok",
		requestId: `task-254-session-${userId}`,
		data: {
			userId,
			role,
			hasVerifiedLoginMethod: true,
			accessExpiresAt: "2026-07-21T13:00:00Z",
			refreshExpiresAt: "2026-07-28T13:00:00Z"
		}
	};
}

function profileEnvelope(userId: string): ProfileEnvelope {
	return {
		status: "ok",
		requestId: `task-254-profile-${userId}`,
		data: {
			userId,
			displayName: userId,
			unitSystem: "metric",
			themePreference: "system",
			requiresUnitRecalculation: false
		}
	};
}

function entitlementEnvelope(userId: string): EntitlementStatusEnvelope {
	return {
		status: "ok",
		requestId: `task-254-entitlement-${userId}`,
		data: {
			userId,
			tier: "paid",
			status: "active",
			allowedModes: ["catalog", "substitution", "daily_diet", "daily_diet_alternative"],
			searchLimitPer24h: null,
			usageUsed: 0,
			usageRemaining: null,
			usageWindowStartedAt: "2026-07-21T00:00:00Z",
			trialExpiresAt: null,
			billingRecoveryState: "none"
		}
	};
}

function errorEnvelope(code: string, message: string) {
	return { status: "error", requestId: `task-254-${code}`, error: { category: "auth", code, message, retryable: false } };
}

async function fulfillJson(route: Route, status: number, body: unknown): Promise<void> {
	await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

async function stubCommon(page: Page): Promise<void> {
	await page.route(/\/api\/v1\/search-history$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "task-254-history", data: { history: [] } }));
	await page.route(/\/api\/v1\/saved-items\?kind=favorite$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "task-254-favorites", data: { items: [] } }));
	await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "task-254-autocomplete", data: { items: [] } }));
}

async function stubSession(page: Page, role: Role, userId = `${role}-254`): Promise<void> {
	await stubCommon(page);
	await page.route(/\/api\/v1\/profile$/, (route) => fulfillJson(route, 200, profileEnvelope(userId)));
	await page.route(/\/api\/v1\/auth\/refresh$/, (route) => fulfillJson(route, 200, sessionEnvelope(userId, role)));
	await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => fulfillJson(route, 200, entitlementEnvelope(userId)));
}

async function openSidebar(page: Page): Promise<void> {
	const open = page.getByLabel("Open activity sidebar");
	if (await open.isVisible()) await open.click();
}

// Verifies IT-ARCH-009-001, ARCH-009, DESIGN-009 UserAdminPanel, and SW-REQ-054.
test("verified admins reach the keyboard-operable responsive panel in light and dark themes", async ({ page }, testInfo) => {
	await stubSession(page, "admin");
	await page.emulateMedia({ reducedMotion: "reduce" });
	await page.goto("/");
	await openSidebar(page);

	const administration = page.locator("[data-sidebar-nav-administration]");
	await expect(administration).toBeVisible();
	await administration.focus();
	await expect(administration).toBeFocused();
	await page.keyboard.press("Enter");

	await expect(page).toHaveURL(/\/admin$/);
	await expect(page.getByRole("heading", { name: "Administration Panel", level: 1 })).toBeVisible();
	await expect(page.locator("[data-admin-responsive-grid]")).toBeVisible();
	await expect(page.locator("[data-admin-server-auth-notice]")).toContainText("server authorizes every administration request");

	const columns = await page.locator("[data-admin-responsive-grid]").evaluate((element) => getComputedStyle(element).gridTemplateColumns.split(" ").length);
	expect(columns).toBe(testInfo.project.name === "mobile-chromium" ? 1 : 3);

	for (const theme of ["light", "dark"] as const) {
		await openSidebar(page);
		if ((await page.locator("html").getAttribute("data-theme")) !== theme) await page.getByLabel("Theme preference").click();
		await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
		const axe = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
		expect(axe.violations.filter((violation) => violation.impact === "serious" || violation.impact === "critical")).toEqual([]);
	}
});

// Verifies IT-ARCH-009-001, ARCH-009, DESIGN-009 UserAdminPanel, and SW-REQ-054.
test("anonymous and standard users see no administration control and direct routes fail closed", async ({ page }) => {
	await stubCommon(page);
	await page.route(/\/api\/v1\/profile$/, (route) => fulfillJson(route, 401, errorEnvelope("anonymous_session", "Please sign in.")));
	await page.goto("/admin");
	await expect(page).toHaveURL(/\/$/);
	await expect(page.locator("[data-admin-access-denied]")).toBeVisible();
	await expect(page.locator("[data-sidebar-nav-administration]")).toHaveCount(0);
	await expect(page.locator("[data-administration-panel]")).toHaveCount(0);

	await page.unroute(/\/api\/v1\/profile$/);
	await page.route(/\/api\/v1\/profile$/, (route) => fulfillJson(route, 200, profileEnvelope("user-254")));
	await page.route(/\/api\/v1\/auth\/refresh$/, (route) => fulfillJson(route, 200, sessionEnvelope("user-254", "user")));
	await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => fulfillJson(route, 200, entitlementEnvelope("user-254")));
	await page.goto("/admin");
	await expect(page).toHaveURL(/\/$/);
	await expect(page.locator("[data-sidebar-nav-administration]")).toHaveCount(0);
	await expect(page.locator("[data-administration-panel]")).toHaveCount(0);
});

test("a malformed admin-shaped session exposes no administration control", async ({ page }) => {
	await stubCommon(page);
	await page.route(/\/api\/v1\/profile$/, (route) => fulfillJson(route, 200, profileEnvelope("admin-malformed-254")));
	await page.route(/\/api\/v1\/auth\/refresh$/, (route) => fulfillJson(route, 200, {
		status: "ok",
		requestId: "task-254-malformed-session",
		data: { role: "admin", hasVerifiedLoginMethod: true }
	}));
	await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => fulfillJson(route, 200, entitlementEnvelope("admin-malformed-254")));
	await page.goto("/");
	await openSidebar(page);
	await expect(page.locator("[data-sidebar-nav-administration]")).toHaveCount(0);
});

test("logout and account replacement reset administration while preserving search state", async ({ page }) => {
	let current: { userId: string; role: Role } | null = { userId: "admin-254", role: "admin" };
	await stubCommon(page);
	await page.route(/\/api\/v1\/profile$/, (route) => current ? fulfillJson(route, 200, profileEnvelope(current.userId)) : fulfillJson(route, 401, errorEnvelope("anonymous_session", "Please sign in.")));
	await page.route(/\/api\/v1\/auth\/refresh$/, (route) => current ? fulfillJson(route, 200, sessionEnvelope(current.userId, current.role)) : fulfillJson(route, 401, errorEnvelope("anonymous_session", "Please sign in.")));
	await page.route(/\/api\/v1\/auth\/csrf-token$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "task-254-csrf", data: { csrfToken: "csrf-task-254" } }));
	await page.route(/\/api\/v1\/auth\/logout$/, (route) => { current = null; return fulfillJson(route, 200, { status: "ok", requestId: "task-254-logout" }); });
	await page.route(/\/api\/v1\/auth\/login$/, (route) => { current = { userId: "user-254", role: "user" }; return fulfillJson(route, 200, sessionEnvelope(current.userId, current.role)); });
	await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => current ? fulfillJson(route, 200, entitlementEnvelope(current.userId)) : fulfillJson(route, 401, errorEnvelope("anonymous_session", "Please sign in.")));

	await page.goto("/");
	await page.getByLabel("Food search").fill("preserved apples");
	await openSidebar(page);
	await page.locator("[data-sidebar-nav-administration]").click();
	await expect(page.locator("[data-administration-panel]")).toBeVisible();
	await openSidebar(page);
	await page.locator("[data-sidebar-sign-out]").click();
	await expect(page).toHaveURL(/\/$/);
	await expect(page.getByLabel("Food search")).toHaveValue("preserved apples");
	await expect(page.locator("[data-sidebar-nav-administration]")).toHaveCount(0);

	await openSidebar(page);
	await page.locator("[data-sidebar-sign-in]").click();
	await page.getByLabel("Email").fill("user@example.test");
	await page.getByLabel("Password").fill("not-a-real-password");
	await page.getByRole("button", { name: "Sign in", exact: true }).last().click();
	await expect(page.locator("[data-auth-surface]")).toHaveCount(0);
	await expect(page.getByLabel("Food search")).toHaveValue("preserved apples");
	await expect(page.locator("[data-sidebar-nav-administration]")).toHaveCount(0);
	await page.goto("/admin");
	await expect(page).toHaveURL(/\/$/);
});

test("direct admin routes expose feature-local loading and safe error boundaries", async ({ page }) => {
	await stubCommon(page);
	let releaseProfile!: () => void;
	const profileReleased = new Promise<void>((resolve) => { releaseProfile = resolve; });
	await page.route(/\/api\/v1\/profile$/, async (route) => {
		await profileReleased;
		await fulfillJson(route, 500, errorEnvelope("session_probe_failed", "Safe failure."));
	});

	await page.goto("/admin");
	await expect(page.locator("[data-admin-loading]")).toBeVisible();
	releaseProfile();
	await expect(page.locator("[data-admin-error]")).toBeVisible();
	await expect(page.locator("[data-sidebar-nav-administration]")).toHaveCount(0);
});
