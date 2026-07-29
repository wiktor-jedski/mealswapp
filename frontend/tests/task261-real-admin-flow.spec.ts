import { expect, test, type Page } from "@playwright/test";

// Verifies IT-ARCH-009-004 and IT-ARCH-009-005 through the real generated-client,
// Administration Panel, API, PostgreSQL, and dynamic-filter browser flow.
// Implements DESIGN-008 DataExporter/ProfileController and DESIGN-009 UserAdminPanel/TagManager.

const runRealFlow = process.env.MEALSWAPP_TASK261_REAL_E2E === "1";
test.skip(!runRealFlow, "Run scripts/verify-task-261-ui.sh for the real task-261 browser flow.");

test("Admin Panel generated client deletes exported private data and publishes a dynamic filter", async ({ page }) => {
	test.setTimeout(60_000);
	const email = requiredFixture("MEALSWAPP_E2E_EMAIL");
	const password = requiredFixture("MEALSWAPP_E2E_PASSWORD");
	const userId = requiredFixture("MEALSWAPP_E2E_USER_ID");
	const classification = requiredFixture("MEALSWAPP_E2E_CLASSIFICATION");
	const privateItem = requiredFixture("MEALSWAPP_E2E_PRIVATE_ITEM");
	const globalItem = requiredFixture("MEALSWAPP_E2E_GLOBAL_ITEM");
	expect(userId).toMatch(/^[0-9a-f-]{36}$/i);

	await page.goto("/");
	await page.getByRole("button", { name: "Sign in", exact: true }).click();
	await page.locator("[data-login-view]").getByLabel("Email").fill(email);
	await page.locator("[data-login-view]").getByLabel("Password").fill(password);
	await page.locator("[data-login-view]").getByRole("button", { name: "Sign in" }).click();
	await expect(page.getByRole("button", { name: "Administration" })).toBeVisible();
	if (process.env.MEALSWAPP_E2E_INJECT_ASSERTION_FAILURE === "1") {
		expect("injected Task 279 failure").toBe("successful assertion");
	}

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
	const createdClassification = await classificationResponse;
	expect(createdClassification.status()).toBe(201);
	expect(await responseID(createdClassification)).toMatch(/^[0-9a-f-]{36}$/i);
	const itemForm = page.getByRole("form", { name: "Manual global item form" });
	await itemForm.getByLabel("Name").fill(globalItem);
	await itemForm.getByLabel("Protein per 100").fill("18");
	await itemForm.getByLabel("Carbohydrates per 100").fill("4");
	await itemForm.getByLabel("Fat per 100").fill("8");
	const itemResponse = page.waitForResponse((response) => response.url().endsWith("/api/v1/admin/items") && response.request().method() === "POST");
	await itemForm.getByRole("button", { name: "Create item" }).click();
	const createdGlobalItem = await itemResponse;
	expect(createdGlobalItem.status()).toBe(201);
	expect(await responseID(createdGlobalItem)).toMatch(/^[0-9a-f-]{36}$/i);

	await page.goto("/?mode=substitution");
	await page.getByLabel("Food search").fill(globalItem);
	await page.getByRole("listbox", { name: "Autocomplete suggestions" }).getByRole("option", { name: globalItem }).click();
	await page.locator("#substitution-include-filter").fill(classification);
	await page.waitForTimeout(250);
	await page.evaluate(() => window.dispatchEvent(new Event("focus")));
	await page.locator("#substitution-include-filter").fill(classification);
	await expect(page.locator("[data-substitution-include-options]")).toContainText(classification);
});

function requiredFixture(name: string): string {
	const value = process.env[name];
	if (!value) throw new Error(`isolated fixture ${name} is required`);
	return value;
}

async function responseID(response: { json(): Promise<unknown> }): Promise<string> {
	const body = await response.json() as { data?: { id?: string; classification?: { id?: string } } };
	return body.data?.id ?? body.data?.classification?.id ?? "";
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
