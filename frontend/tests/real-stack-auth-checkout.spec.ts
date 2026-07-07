import { expect, test, type Page } from "@playwright/test";

// Implements DESIGN-018 AuthenticatedActionGuard real frontend auth/session UAT coverage.
// Verifies ARCH-018 and DESIGN-007 checkout handoff using the real local backend and HttpOnly cookies.

const runRealStack = process.env.MEALSWAPP_REAL_STACK_E2E === "1";

test.skip(!runRealStack, "Set MEALSWAPP_REAL_STACK_E2E=1 and run the local backend stack to execute real-stack UAT.");

test("real registration creates HttpOnly cookies before authenticated checkout", async ({ page, context }) => {
	const email = `uat-${Date.now()}-${Math.random().toString(36).slice(2)}@example.test`;
	const password = "CorrectHorseBatteryStaple1!";

	await page.goto("/auth/register");
	await expect(page.locator("[data-register-view]")).toBeVisible();
	await page.getByLabel("Email").fill(email);
	await page.getByLabel("Password", { exact: true }).fill(password);
	await page.getByLabel("Confirm password").fill(password);
	await page.getByLabel(/I accept the current Privacy Policy and Terms of Service/i).check();

	const registerResponsePromise = page.waitForResponse((response) =>
		response.url().includes("/api/v1/auth/register")
	);
	await page.getByRole("button", { name: "Create account" }).click();
	const registerResponse = await registerResponsePromise;
	expect(registerResponse.status()).toBe(201);
	await expect(page.getByText("Registration complete. Your browser session is authenticated.")).toBeVisible();

	const authCookies = (await context.cookies()).filter((cookie) =>
		cookie.name === "mealswapp_access" || cookie.name === "mealswapp_refresh"
	);
	expect(authCookies.map((cookie) => cookie.name).sort()).toEqual(["mealswapp_access", "mealswapp_refresh"]);
	expect(authCookies.every((cookie) => cookie.httpOnly)).toBe(true);
	expect(JSON.stringify(await browserStorageSnapshot(page))).not.toMatch(/mealswapp_access|mealswapp_refresh|CorrectHorseBatteryStaple|csrf/i);

	const verification = await verifyEmailAndRefreshSession(page);
	expect(verification.csrfStatus).toBe(200);
	expect(verification.verifyStatus).toBe(200);
	expect(verification.refreshStatus).toBe(200);
	expect(verification.verified).toBe(true);
	expect(verification.refreshedVerified).toBe(true);

	await page.goto("/subscription");
	await expect(page.locator("[data-subscription-view] [data-subscription-billing]")).toBeVisible();
	await expect(page.getByText(/free · active|trial · active|paid · active/i)).toBeVisible();

	const checkoutResponsePromise = page.waitForResponse((response) =>
		response.url().includes("/api/v1/billing/checkout")
	);
	const checkoutRequestPromise = page.waitForRequest((request) =>
		request.url().includes("/api/v1/billing/checkout")
	);
	await page.getByRole("button", { name: "Choose Monthly" }).click();
	const checkoutRequest = await checkoutRequestPromise;
	const checkoutResponse = await checkoutResponsePromise;

	expect(checkoutRequest.method()).toBe("POST");
	expect(checkoutRequest.headers()["idempotency-key"]).toMatch(/^checkout-/);
	expect(checkoutRequest.headers()["x-csrf-token"]).toBeTruthy();
	expect(checkoutRequest.postDataJSON()).toMatchObject({ plan: "monthly" });
	expect(checkoutResponse.status(), "checkout must not fail before backend auth/CSRF handling").not.toBe(401);
	expect(checkoutResponse.status(), "checkout must not fail before backend auth/CSRF handling").not.toBe(403);

	if (checkoutResponse.status() === 200) {
		await expect(page).toHaveURL(/checkout\.stripe\.com|\/billing\/success|\/billing\/cancel/, { timeout: 15_000 });
		return;
	}

	expect(checkoutResponse.status(), "real checkout must reach billing auth/CSRF and fail only at Stripe dependency").toBe(503);
	const body = (await checkoutResponse.json()) as { error?: { code?: string } };
	expect(body.error?.code).toBe("stripe_unavailable");
	await expect(page.getByText("Stripe is temporarily unavailable.")).toBeVisible();
});

async function browserStorageSnapshot(page: Page): Promise<Record<string, Record<string, string>>> {
	return page.evaluate(() => {
		const storage = (store: Storage): Record<string, string> => {
			const values: Record<string, string> = {};
			for (let index = 0; index < store.length; index += 1) {
				const key = store.key(index);
				if (key) {
					values[key] = store.getItem(key) ?? "";
				}
			}
			return values;
		};
		return {
			localStorage: storage(window.localStorage),
			sessionStorage: storage(window.sessionStorage)
		};
	});
}

interface VerificationResult {
	csrfStatus: number;
	verifyStatus: number;
	refreshStatus: number;
	verified?: boolean;
	refreshedVerified?: boolean;
}

async function verifyEmailAndRefreshSession(page: Page): Promise<VerificationResult> {
	return page.evaluate(async () => {
		const csrfResponse = await fetch("/api/v1/auth/csrf-token", {
			credentials: "include",
			headers: { Accept: "application/json" }
		});
		const csrfEnvelope = (await csrfResponse.json()) as { data?: { csrfToken?: string } };
		const csrfToken = csrfEnvelope.data?.csrfToken ?? "";
		const verifyResponse = await fetch("/api/v1/auth/verify-email", {
			method: "POST",
			credentials: "include",
			headers: {
				Accept: "application/json",
				"X-CSRF-Token": csrfToken
			}
		});
		const verifyEnvelope = (await verifyResponse.json()) as { data?: { hasVerifiedLoginMethod?: boolean } };
		const refreshResponse = await fetch("/api/v1/auth/refresh", {
			method: "POST",
			credentials: "include",
			headers: { Accept: "application/json" }
		});
		const refreshEnvelope = (await refreshResponse.json()) as { data?: { hasVerifiedLoginMethod?: boolean } };
		return {
			csrfStatus: csrfResponse.status,
			verifyStatus: verifyResponse.status,
			refreshStatus: refreshResponse.status,
			verified: verifyEnvelope.data?.hasVerifiedLoginMethod,
			refreshedVerified: refreshEnvelope.data?.hasVerifiedLoginMethod
		};
	});
}
