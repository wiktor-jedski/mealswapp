import { expect, test, type Page } from "@playwright/test";

// Implements DESIGN-001 PreferenceManager real-stack persistence and authority acceptance.

const runRealStack = process.env.MEALSWAPP_REAL_STACK_E2E === "1";

test.skip(!runRealStack, "Set MEALSWAPP_REAL_STACK_E2E=1 and run the isolated local stack.");

test("authenticated profile preference persists and remains server-authoritative", async ({ page }) => {
	const email = `task296-${Date.now()}-${Math.random().toString(36).slice(2)}@example.test`;
	const password = "CorrectHorseBatteryStaple1!";

	await page.goto("/");
	await page.getByRole("button", { name: "Sign in", exact: true }).click();
	await page.getByRole("group", { name: "Authentication mode" }).getByRole("button", { name: "Create account" }).click();
	await page.getByLabel("Email").fill(email);
	await page.getByLabel("Password", { exact: true }).fill(password);
	await page.getByLabel("Confirm password").fill(password);
	await page.getByLabel(/I accept the current Privacy Policy and Terms of Service/i).check();
	await page.locator("[data-register-view]").getByRole("button", { name: "Create account" }).click();
	await expect(page.getByText("Registration complete. Your browser session is authenticated.")).toBeVisible();

	const result = await page.evaluate(async () => {
		const csrf = await fetch("/api/v1/auth/csrf-token", { credentials: "include" });
		const csrfBody = (await csrf.json()) as { data?: { csrfToken?: string } };
		const token = csrfBody.data?.csrfToken ?? "";
		const initial = await fetch("/api/v1/profile", { credentials: "include" });
		const initialBody = (await initial.json()) as { data?: { unitSystem?: string; displayName?: string; themePreference?: string } };
		const update = await fetch("/api/v1/profile", {
			method: "PUT",
			credentials: "include",
			headers: { "Content-Type": "application/json", "X-CSRF-Token": token },
			body: JSON.stringify({
				displayName: initialBody.data?.displayName ?? "",
				unitSystem: "imperial",
				themePreference: initialBody.data?.themePreference ?? "system"
			})
		});
		const updateBody = (await update.json()) as { data?: { unitSystem?: string } };
		const reload = await fetch("/api/v1/profile", { credentials: "include" });
		const reloadBody = (await reload.json()) as { data?: { unitSystem?: string } };
		return {
			initialStatus: initial.status,
			updateStatus: update.status,
			reloadStatus: reload.status,
			initialUnit: initialBody.data?.unitSystem,
			confirmedUnit: updateBody.data?.unitSystem,
			reloadedUnit: reloadBody.data?.unitSystem
		};
	});

	expect(result.initialStatus).toBe(200);
	expect(result.updateStatus).toBe(200);
	expect(result.reloadStatus).toBe(200);
	expect(result.initialUnit).toBe("metric");
	expect(result.confirmedUnit).toBe("imperial");
	expect(result.reloadedUnit).toBe("imperial");
});
