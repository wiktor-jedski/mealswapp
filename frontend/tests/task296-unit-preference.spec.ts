import { expect, test, type Page, type Route } from "@playwright/test";
import type {
	AuthSessionEnvelope,
	EntitlementStatusEnvelope,
	ProfileEnvelope
} from "../src/lib/api/generated";

// Implements DESIGN-001 SettingsPanel authoritative unit preference browser verification.
// Verifies Task 296 anonymous persistence, authenticated hydration, confirmed updates, failure recovery, and mobile/desktop accessibility.

const userId = "00000000-0000-4000-8000-000000000296";

function profile(unitSystem: "metric" | "imperial"): ProfileEnvelope {
	return {
		status: "ok",
		requestId: `task296-profile-${unitSystem}`,
		data: {
			userId,
			displayName: "Unit User",
			unitSystem,
			themePreference: "system",
			requiresUnitRecalculation: true
		}
	};
}

async function json(route: Route, status: number, body: unknown): Promise<void> {
	await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

async function openSidebar(page: Page): Promise<void> {
	const open = page.getByLabel("Open activity sidebar");
	if (await open.isVisible()) {
		await open.click();
	}
}

async function stubAnonymous(page: Page): Promise<void> {
	await page.route(/\/api\/v1\/profile$/, (route) =>
		json(route, 401, {
			status: "error",
			requestId: "task296-anonymous",
			error: { category: "auth", code: "anonymous_session", message: "Sign in.", retryable: false }
		})
	);
	await page.route(/\/api\/v1\/billing\/entitlement$/, (route) =>
		json(route, 401, {
			status: "error",
			requestId: "task296-entitlement-anonymous",
			error: { category: "auth", code: "anonymous_session", message: "Sign in.", retryable: false }
		})
	);
}

async function stubAuthenticatedShell(page: Page): Promise<void> {
	const session: AuthSessionEnvelope = {
		status: "ok",
		requestId: "task296-session",
		data: {
			userId,
			role: "user",
			hasVerifiedLoginMethod: true,
			accessExpiresAt: "2026-07-29T12:00:00Z",
			refreshExpiresAt: "2026-08-05T12:00:00Z"
		}
	};
	const entitlement: EntitlementStatusEnvelope = {
		status: "ok",
		requestId: "task296-entitlement",
		data: {
			userId,
			tier: "trial",
			status: "active",
			allowedModes: ["catalog", "substitution"],
			searchLimitPer24h: 25,
			usageUsed: 0,
			usageRemaining: 25,
			usageWindowStartedAt: "2026-07-29T00:00:00Z",
			trialExpiresAt: "2026-08-05T00:00:00Z",
			billingRecoveryState: "none"
		}
	};
	await page.route(/\/api\/v1\/auth\/refresh$/, (route) => json(route, 200, session));
	await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => json(route, 200, entitlement));
	await page.route(/\/api\/v1\/search-history$/, (route) =>
		json(route, 200, { status: "ok", requestId: "task296-history", data: { history: [] } })
	);
	await page.route(/\/api\/v1\/saved-items\?kind=favorite$/, (route) =>
		json(route, 200, { status: "ok", requestId: "task296-favorites", data: { items: [] } })
	);
	await page.route(/\/api\/v1\/auth\/csrf-token$/, (route) =>
		json(route, 200, { status: "ok", requestId: "task296-csrf", data: { csrfToken: "csrf-task296" } })
	);
}

test("anonymous device preference survives reload independently of profile state", async ({ page }) => {
	await stubAnonymous(page);
	await page.goto("/");
	await openSidebar(page);

	const units = page.locator("#sidebar-unit-system");
	await expect(units).toHaveValue("metric");
	await units.selectOption("imperial");
	await expect(units).toHaveValue("imperial");

	await page.reload();
	await openSidebar(page);
	await expect(page.locator("#sidebar-unit-system")).toHaveValue("imperial");
});

test("authenticated UI hydrates from profile and changes only after confirmation", async ({ page }) => {
	await stubAuthenticatedShell(page);
	let confirmSave!: () => void;
	const saveConfirmation = new Promise<void>((resolve) => {
		confirmSave = resolve;
	});
	await page.route(/\/api\/v1\/profile$/, async (route) => {
		if (route.request().method() === "PUT") {
			expect(route.request().headers()["x-csrf-token"]).toBe("csrf-task296");
			expect((await route.request().postDataJSON()).unitSystem).toBe("metric");
			await saveConfirmation;
			await json(route, 200, profile("metric"));
			return;
		}
		await json(route, 200, profile("imperial"));
	});

	await page.goto("/");
	await openSidebar(page);
	const units = page.locator("#sidebar-unit-system");
	await expect(units).toHaveValue("imperial");

	await units.selectOption("metric");
	await expect(units).toHaveValue("imperial");
	await expect(page.getByText("Saving units…")).toBeVisible();
	confirmSave();
	await expect(units).toHaveValue("metric");
	await expect(page.getByText("Saving units…")).toBeHidden();
});

test("failed authenticated save preserves the confirmed value and exposes keyboard recovery", async ({ page }) => {
	await stubAuthenticatedShell(page);
	let putCalls = 0;
	await page.route(/\/api\/v1\/profile$/, async (route) => {
		if (route.request().method() === "PUT") {
			putCalls += 1;
			if (putCalls === 1) {
				await json(route, 503, {
					status: "error",
					requestId: "task296-save-failed",
					error: { category: "dependency", code: "profile_unavailable", message: "Unavailable.", retryable: true }
				});
				return;
			}
			await json(route, 200, profile("metric"));
			return;
		}
		await json(route, 200, profile("imperial"));
	});

	await page.goto("/");
	await openSidebar(page);
	const units = page.locator("#sidebar-unit-system");
	await units.selectOption("metric");
	await expect(units).toHaveValue("imperial");
	const alert = page.getByRole("alert");
	await expect(alert).toContainText("Couldn't save your unit preference.");

	const retry = page.getByRole("button", { name: "Retry" });
	await retry.focus();
	await expect(retry).toBeFocused();
	await retry.press("Enter");
	await expect(units).toHaveValue("metric");
	expect(putCalls).toBe(2);
});

test("sign-out restores anonymous units and the next account cannot inherit prior units", async ({ page }) => {
	await page.addInitScript(() => {
		window.localStorage.setItem("mealswapp.preferences", JSON.stringify({ unitSystem: "metric" }));
	});
	let currentUser: "account-a" | "account-b" | null = "account-a";
	await stubAuthenticatedShell(page);
	await page.route(/\/api\/v1\/profile$/, (route) => {
		if (currentUser === null) {
			return json(route, 401, {
				status: "error",
				requestId: "task296-signed-out",
				error: { category: "auth", code: "anonymous_session", message: "Sign in.", retryable: false }
			});
		}
		const envelope = profile(currentUser === "account-a" ? "imperial" : "metric");
		envelope.data.userId = currentUser === "account-a" ? userId : "00000000-0000-4000-8000-000000000297";
		return json(route, 200, envelope);
	});
	await page.route(/\/api\/v1\/auth\/logout$/, async (route) => {
		currentUser = null;
		await route.fulfill({ status: 204 });
	});
	await page.route(/\/api\/v1\/auth\/login$/, async (route) => {
		currentUser = "account-b";
		const envelope: AuthSessionEnvelope = {
			status: "ok",
			requestId: "task296-account-b-session",
			data: {
				userId: "00000000-0000-4000-8000-000000000297",
				role: "user",
				hasVerifiedLoginMethod: true,
				accessExpiresAt: "2026-07-29T12:00:00Z",
				refreshExpiresAt: "2026-08-05T12:00:00Z"
			}
		};
		await json(route, 200, envelope);
	});

	await page.goto("/");
	await openSidebar(page);
	const units = page.locator("#sidebar-unit-system");
	await expect(units).toHaveValue("imperial");

	await page.getByRole("button", { name: "Sign out" }).click();
	await expect(units).toHaveValue("metric");
	await page.getByRole("button", { name: "Sign in", exact: true }).click();
	await page.getByLabel("Email").fill("account-b@example.test");
	await page.getByLabel("Password").fill("not-a-real-password");
	await page.locator("[data-login-view]").getByRole("button", { name: "Sign in" }).click();
	await expect(units).toHaveValue("metric");
});
