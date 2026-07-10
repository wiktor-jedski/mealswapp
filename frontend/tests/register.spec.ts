import { expect, test, type Page, type Route } from "@playwright/test";

import type { AuthSessionEnvelope, EntitlementStatusEnvelope } from "../src/lib/api/generated";

// Implements DESIGN-018 RegisterView browser tests for consent-gated email/password registration.
// Verifies IT-ARCH-018-001, IT-ARCH-018-005, IT-ARCH-018-007, ARCH-018, DESIGN-018, SW-REQ-058, SW-REQ-060, SW-REQ-071, and SW-REQ-074.

function authSessionEnvelope(hasVerifiedLoginMethod = true): AuthSessionEnvelope {
	return {
		status: "ok",
		requestId: "register-session",
		data: {
			userId: "user-register-1",
			role: "user",
			hasVerifiedLoginMethod,
			accessExpiresAt: "2026-07-05T10:00:00Z",
			refreshExpiresAt: "2026-07-12T10:00:00Z"
		}
	};
}

function entitlementEnvelope(): EntitlementStatusEnvelope {
	return {
		status: "ok",
		requestId: "register-entitlement",
		data: {
			userId: "user-register-1",
			tier: "trial",
			status: "active",
			allowedModes: ["catalog"],
			searchLimitPer24h: 25,
			usageUsed: 0,
			usageRemaining: 25,
			usageWindowStartedAt: "2026-07-05T00:00:00Z",
			trialExpiresAt: "2026-07-12T00:00:00Z",
			billingRecoveryState: "none"
		}
	};
}

async function fulfillJson(route: Route, status: number, body: unknown): Promise<void> {
	await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

async function stubRegisterApi(
	page: Page,
	options: { registerStatus?: "success" | "duplicate" | "stale" | "unverified" } = {}
): Promise<{ payloads: unknown[] }> {
	const payloads: unknown[] = [];
	await page.route(/\/api\/v1\/profile$/, async (route) => {
		await fulfillJson(route, 401, {
			status: "error",
			requestId: "register-profile-anonymous",
			error: {
				category: "auth",
				code: "session_invalid",
				message: "Session required.",
				retryable: false
			}
		});
	});
	await page.route(/\/api\/v1\/auth\/csrf-token$/, async (route) => {
		await fulfillJson(route, 200, {
			status: "ok",
			requestId: "csrf-register",
			data: { csrfToken: "csrf-register-token" }
		});
	});
	await page.route(/\/api\/v1\/auth\/register$/, async (route) => {
		payloads.push(route.request().postDataJSON());
		const status = options.registerStatus ?? "success";
		if (status === "duplicate") {
			await fulfillJson(route, 409, {
				status: "error",
				requestId: "duplicate-register",
				error: {
					category: "validation",
					code: "duplicate_email",
					message: "Email already registered.",
					retryable: false
				}
			});
			return;
		}
		if (status === "stale") {
			await fulfillJson(route, 409, {
				status: "error",
				requestId: "stale-consent-register",
				error: {
					category: "validation",
					code: "consent_stale",
					message: "Legal terms changed.",
					retryable: false
				}
			});
			return;
		}
		await fulfillJson(route, 201, authSessionEnvelope(status !== "unverified"));
	});
	await page.route(/\/api\/v1\/billing\/entitlement$/, async (route) => {
		await fulfillJson(route, 200, entitlementEnvelope());
	});
	return { payloads };
}

async function fillValidRegistration(page: Page): Promise<void> {
	await page.getByLabel("Email").fill("person@example.com");
	await page.getByLabel("Password", { exact: true }).fill("correct-horse-1");
	await page.getByLabel("Confirm password").fill("correct-horse-1");
	await page.getByLabel("I accept the current Privacy Policy and Terms of Service.").check();
}

async function openRegisterModal(page: Page): Promise<void> {
	await page.goto("/");
	const mobileToggle = page.getByLabel("Open activity sidebar");
	if (await mobileToggle.isVisible()) {
		await mobileToggle.click();
	}
	await page.getByRole("button", { name: "Sign in", exact: true }).click();
	await page.getByRole("group", { name: "Authentication mode" }).getByRole("button", { name: "Create account" }).click();
	await expect(page.locator("[data-register-view]")).toBeVisible();
}

async function browserStorageSnapshot(page: Page): Promise<string> {
	return page.evaluate(() => {
		const entries = {
			localStorage: Array.from({ length: localStorage.length }, (_, index) => {
				const key = localStorage.key(index) ?? "";
				return [key, localStorage.getItem(key)];
			}),
			sessionStorage: Array.from({ length: sessionStorage.length }, (_, index) => {
				const key = sessionStorage.key(index) ?? "";
				return [key, sessionStorage.getItem(key)];
			})
		};
		return JSON.stringify(entries);
	});
}

test("registration cannot submit until current consent versions are checked", async ({ page }) => {
	const api = await stubRegisterApi(page);
	await openRegisterModal(page);

	await page.getByLabel("Email").fill("person@example.com");
	await page.getByLabel("Password", { exact: true }).fill("correct-horse-1");
	await page.getByLabel("Confirm password").fill("correct-horse-1");

	await expect(page.locator("[data-register-view]").getByRole("button", { name: "Create account" })).toBeDisabled();
	await page.getByLabel("I accept the current Privacy Policy and Terms of Service.").check();
	await expect(page.locator("[data-register-view]").getByRole("button", { name: "Create account" })).toBeEnabled();
	expect(api.payloads).toHaveLength(0);
});

test("combined consent checkbox accepts both legal versions and legal links target placeholder views", async ({ page }) => {
	const api = await stubRegisterApi(page);
	await openRegisterModal(page);

	await page.getByLabel("Email").fill("person@example.com");
	await page.getByLabel("Password", { exact: true }).fill("correct-horse-1");
	await page.getByLabel("Confirm password").fill("correct-horse-1");
	await expect(page.locator("[data-register-view]").getByRole("button", { name: "Create account" })).toBeDisabled();

	await expect(page.getByRole("link", { name: "Privacy Policy" })).toHaveAttribute("href", "/privacy");
	await expect(page.getByRole("link", { name: "Terms of Service" })).toHaveAttribute("href", "/terms");
	await page.getByLabel("I accept the current Privacy Policy and Terms of Service.").check();

	await expect(page.getByLabel("I accept the current Privacy Policy and Terms of Service.")).toBeChecked();
	await expect(page.locator("[data-register-view]").getByRole("button", { name: "Create account" })).toBeEnabled();
	expect(api.payloads).toHaveLength(0);
});

test("legal placeholder routes render Privacy Policy and Terms of Service views", async ({ page }) => {
	await stubRegisterApi(page);

	await page.goto("/privacy");
	await expect(page.locator("[data-privacy-view]")).toBeVisible();
	await expect(page.getByRole("heading", { name: "Privacy Policy" })).toBeVisible();
	await expect(page.getByText("Privacy Policy placeholder text.")).toBeVisible();

	await page.goto("/terms");
	await expect(page.locator("[data-terms-view]")).toBeVisible();
	await expect(page.getByRole("heading", { name: "Terms of Service" })).toBeVisible();
	await expect(page.getByText("Terms of Service placeholder text.")).toBeVisible();
	await expect(page.locator("[data-medical-disclaimer]")).toContainText("does not provide medical advice");
});

test("password mismatch and policy failures are shown safely", async ({ page }) => {
	const api = await stubRegisterApi(page);
	await openRegisterModal(page);

	await page.getByLabel("Email").fill("person@example.com");
	await page.getByLabel("Password", { exact: true }).fill("short");
	await page.getByLabel("Confirm password").fill("different");
	await page.getByLabel("I accept the current Privacy Policy and Terms of Service.").check();
	await page.locator("[data-register-view]").getByRole("button", { name: "Create account" }).click();

	await expect(page.getByRole("alert")).toHaveCount(0);
	await expect(page.locator("#register-password-error")).toHaveText("Use at least 12 characters.");
	await expect(page.locator("#register-confirm-error")).toHaveText("Passwords do not match.");
	await expect(page.getByText("short")).toHaveCount(0);
	await expect(page.getByText("different")).toHaveCount(0);
	expect(api.payloads).toHaveLength(0);
});

test("duplicate email offers login mode without storing PII or passwords", async ({ page }) => {
	const api = await stubRegisterApi(page, { registerStatus: "duplicate" });
	await openRegisterModal(page);
	await fillValidRegistration(page);
	await page.locator("[data-register-view]").getByRole("button", { name: "Create account" }).click();

	await expect(page.getByText("An account already exists for this email.")).toBeVisible();
	await expect(page.getByRole("button", { name: "Log in instead" })).toBeVisible();
	expect(api.payloads).toHaveLength(1);
	expect(await browserStorageSnapshot(page)).not.toMatch(/person@example.com|correct-horse-1/);
});

test("stale consent clears acceptance and requires re-acceptance", async ({ page }) => {
	await stubRegisterApi(page, { registerStatus: "stale" });
	await openRegisterModal(page);
	await fillValidRegistration(page);
	await page.locator("[data-register-view]").getByRole("button", { name: "Create account" }).click();

	await expect(page.getByText("Privacy Policy or Terms changed.")).toBeVisible();
	await expect(page.getByLabel("I accept the current Privacy Policy and Terms of Service.")).not.toBeChecked();
	await expect(page.locator("[data-register-view]").getByRole("button", { name: "Create account" })).toBeDisabled();
});

test("successful registration creates an authenticated session projection", async ({ page }) => {
	await stubRegisterApi(page);
	await openRegisterModal(page);
	await fillValidRegistration(page);
	await page.locator("[data-register-view]").getByRole("button", { name: "Create account" }).click();

	await expect(page.getByRole("dialog")).toHaveCount(0);
	await expect(page.locator("[data-sidebar-sign-out]")).toBeAttached();
	expect(await browserStorageSnapshot(page)).not.toMatch(/correct-horse-1|csrf-register-token/);
});

test("unverified login method restrictions are displayed from server state", async ({ page }) => {
	await stubRegisterApi(page, { registerStatus: "unverified" });
	await openRegisterModal(page);
	await fillValidRegistration(page);
	await page.locator("[data-register-view]").getByRole("button", { name: "Create account" }).click();

	await expect(
		page.getByText("Verify your email before using features that require a verified login method.")
	).toBeVisible();
});
