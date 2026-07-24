import { execFileSync } from "node:child_process";
import { expect, test, type Page } from "@playwright/test";

// Verifies IT-ARCH-009-004 and IT-ARCH-009-005 through the real generated-client,
// Administration Panel, API, PostgreSQL, and dynamic-filter browser flow.
// Implements DESIGN-008 DataExporter/ProfileController and DESIGN-009 UserAdminPanel/TagManager.

const runRealFlow = process.env.MEALSWAPP_TASK261_REAL_E2E === "1";
test.skip(!runRealFlow, "Run scripts/verify-task-261-ui.sh for the real task-261 browser flow.");

test("Admin Panel generated client deletes exported private data and publishes a dynamic filter", async ({ page }) => {
	test.setTimeout(60_000);
	const nonce = `${Date.now()}-${Math.random().toString(36).slice(2)}`;
	const email = `task-261-${nonce}@example.test`;
	const password = "StrongerPassword1!";
	const classification = `Task 261 dynamic ${nonce}`;
	const privateItem = `Task 261 private ${nonce}`;
	const globalItem = `Task 261 global ${nonce}`;

	const registerResponse = await register(page, email, password);
	const registerBody = await registerResponse.json() as { data?: { userId?: string } };
	const userId = registerBody.data?.userId;
	if (!userId || !/^[0-9a-f-]{36}$/i.test(userId)) throw new Error("registration did not return a safe user ID");
	promoteToAdmin(userId);
	await verifyEmailFixture(page);

	// Registration keeps the unverified-login modal open; re-authentication obtains fresh admin claims.
	await page.getByRole("group", { name: "Authentication mode" }).getByRole("button", { name: "Sign in" }).click();
	await page.locator("[data-login-view]").getByLabel("Email").fill(email);
	await page.locator("[data-login-view]").getByLabel("Password").fill(password);
	await page.locator("[data-login-view]").getByRole("button", { name: "Sign in" }).click();
	await expect(page.getByRole("button", { name: "Administration" })).toBeVisible();

	const itemId = await createPrivateItemFixture(page, privateItem);
	await page.getByRole("button", { name: "Administration" }).click();
	const privateData = page.locator("[data-admin-private-data]");
	await expect(privateData.getByText(privateItem)).toBeVisible();
	await privateData.getByRole("button", { name: "Delete private item" }).click();
	const deletionResponse = page.waitForResponse((response) => response.url().endsWith(`/api/v1/custom-items/${itemId}`) && response.request().method() === "DELETE");
	await privateData.getByRole("button", { name: "Confirm private item deletion" }).click();
	expect((await deletionResponse).status()).toBe(204);
	await expect(privateData.getByText("Private item deleted and authoritative export refreshed.")).toBeVisible();
	await expect(privateData.locator("[data-admin-private-data-empty]")).toBeVisible();
	await expect(privateData).not.toContainText(privateItem);

	const classificationForm = page.getByRole("form", { name: "Classification form" });
	await classificationForm.getByLabel("Name").fill(classification);
	const classificationResponse = page.waitForResponse((response) => response.url().endsWith("/api/v1/admin/classifications/food_category") && response.request().method() === "POST");
	await classificationForm.getByRole("button", { name: "Create", exact: true }).click();
	expect((await classificationResponse).status()).toBe(201);
	const itemForm = page.getByRole("form", { name: "Manual global item form" });
	await itemForm.getByLabel("Name").fill(globalItem);
	await itemForm.getByLabel("Protein per 100").fill("18");
	await itemForm.getByLabel("Carbohydrates per 100").fill("4");
	await itemForm.getByLabel("Fat per 100").fill("8");
	const itemResponse = page.waitForResponse((response) => response.url().endsWith("/api/v1/admin/items") && response.request().method() === "POST");
	await itemForm.getByRole("button", { name: "Create item" }).click();
	expect((await itemResponse).status()).toBe(201);

	await page.goto("/?mode=substitution");
	await page.getByLabel("Food search").fill(globalItem);
	await page.getByRole("listbox", { name: "Autocomplete suggestions" }).getByRole("option", { name: globalItem }).click();
	await page.locator("#substitution-include-filter").fill(classification);
	await page.waitForTimeout(250);
	await page.evaluate(() => window.dispatchEvent(new Event("focus")));
	await page.locator("#substitution-include-filter").fill(classification);
	await expect(page.locator("[data-substitution-include-options]")).toContainText(classification);
});

async function register(page: Page, email: string, password: string) {
	await page.goto("/");
	await page.getByRole("button", { name: "Sign in", exact: true }).click();
	await page.getByRole("group", { name: "Authentication mode" }).getByRole("button", { name: "Create account" }).click();
	const form = page.locator("[data-register-view]");
	await form.getByLabel("Email").fill(email);
	await form.getByLabel("Password", { exact: true }).fill(password);
	await form.getByLabel("Confirm password").fill(password);
	await form.getByLabel(/I accept the current Privacy Policy and Terms of Service/i).check();
	const response = page.waitForResponse((candidate) => candidate.url().endsWith("/api/v1/auth/register"));
	await form.getByRole("button", { name: "Create account" }).click();
	const registered = await response;
	expect(registered.status()).toBe(201);
	await expect(page.getByText("Registration complete. Your browser session is authenticated.")).toBeVisible();
	return registered;
}

function promoteToAdmin(userId: string): void {
	const sql = `UPDATE users SET role='admin' WHERE id='${userId}'::uuid`;
	execFileSync("docker", ["compose", "exec", "-T", "postgres", "psql", "-U", "mealswapp", "-d", "mealswapp", "-v", "ON_ERROR_STOP=1", "-c", sql], { cwd: "..", stdio: "pipe" });
}

async function verifyEmailFixture(page: Page): Promise<void> {
	await page.evaluate(async () => {
		const csrfResponse = await fetch("/api/v1/auth/csrf-token", { credentials: "include", headers: { Accept: "application/json" } });
		const csrf = await csrfResponse.json() as { data?: { csrfToken?: string } };
		const response = await fetch("/api/v1/auth/verify-email", { method: "POST", credentials: "include", headers: { Accept: "application/json", "X-CSRF-Token": csrf.data?.csrfToken ?? "" } });
		if (response.status !== 200) throw new Error(`verification fixture failed with ${response.status}`);
	});
}

async function createPrivateItemFixture(page: Page, name: string): Promise<string> {
	return page.evaluate(async (itemName) => {
		const csrfResponse = await fetch("/api/v1/auth/csrf-token", { credentials: "include", headers: { Accept: "application/json" } });
		const csrf = await csrfResponse.json() as { data?: { csrfToken?: string } };
		const response = await fetch("/api/v1/custom-items", {
			method: "POST", credentials: "include",
			headers: { Accept: "application/json", "Content-Type": "application/json", "X-CSRF-Token": csrf.data?.csrfToken ?? "", "Idempotency-Key": crypto.randomUUID() },
			body: JSON.stringify({ name: itemName, physicalState: "solid", prepTimeMinutes: 0, macrosPer100: { protein: 20, carbohydrates: 8, fat: 11 }, micros: {}, foodCategoryIds: [], culinaryRoleIds: [] })
		});
		const body = await response.json() as { data?: { id?: string } };
		if (response.status !== 201 || !body.data?.id) throw new Error(`private-item fixture failed with ${response.status}`);
		return body.data.id;
	}, name);
}
