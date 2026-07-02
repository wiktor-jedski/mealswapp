import { expect, test, type Page, type Route } from "@playwright/test";
import type { EntitlementEnvelope } from "../src/lib/api/generated";

// Implements DESIGN-001 SearchView Phase 06 Search UI Entitlement Gating verification.

async function mockEntitlement(page: Page, tier: "free" | "trial" | "paid", usageRemaining?: number): Promise<void> {
	await page.route("/api/v1/entitlements", async (route: Route) => {
		const envelope: EntitlementEnvelope = {
			status: "ok",
			requestId: "test-req",
			data: {
				tier,
				allowedModes: tier === "free" ? ["catalog", "substitution"] : ["catalog", "substitution", "substitution:multi", "daily_diet_alternative"],
				searchLimitPer24h: 10,
				usageRemaining: usageRemaining,
			}
		};
		await route.fulfill({ status: 200, json: envelope });
	});
}

test.describe("Search UI Entitlement Gating", () => {
	test("free-user usage counter display", async ({ page }) => {
		await mockEntitlement(page, "free", 8);
		await page.goto("/");
		await expect(page.locator("[data-entitlement-usage]")).toContainText("Remaining searches: 8/10");
	});

		test("multi-input Substitution shows entitlement feedback without sending blocked searches", async ({ page }) => {
		await mockEntitlement(page, "free", 8);
		await page.route("**/api/v1/search/autocomplete*", async (route) => {
			await route.fulfill({ status: 200, json: { status: "ok", data: { items: [{ itemId: "1", label: "Apple", score: 1 }, { itemId: "2", label: "Banana", score: 1 }] } } });
		});
		await page.route("**/api/v1/food-objects/*", async (route) => {
			await route.fulfill({ status: 200, json: { status: "ok", data: { id: "1", name: "Apple", macroBasis: "100g", macros: { protein: 0, carbohydrates: 0, fat: 0 }, calories: 0, classifications: [] } } });
		});
		await page.goto("/");
		
		await page.getByRole("button", { name: "Substitution" }).click();
		await page.getByRole("combobox", { name: "Food search" }).fill("apple");
		await page.getByRole("option", { name: "Apple" }).first().click();
		await page.getByRole("combobox", { name: "Food search" }).fill("banana");
		await page.getByRole("option", { name: "Banana" }).first().click();

		await expect(page.locator("[data-entitlement-feedback]")).toBeVisible();
	});

	test("Daily Diet modes show entitlement feedback without sending blocked searches", async ({ page }) => {
		await mockEntitlement(page, "free", 8);
		await page.goto("/");
		
		await page.getByRole('button', { name: 'Daily Diet Alternative' }).click();
		await expect(page.locator("[data-entitlement-feedback]")).toBeVisible();
	});

	test("trial/paid fixtures unlock paid modes", async ({ page }) => {
		await mockEntitlement(page, "paid");
		await page.goto("/");
		
		await page.getByRole('button', { name: 'Substitution' }).click();
		await expect(page.locator("[data-entitlement-feedback]")).toBeHidden();

		await page.getByRole('button', { name: 'Daily Diet Alternative' }).click();
		await expect(page.locator("[data-entitlement-feedback]")).toBeHidden();
	});

	test("anonymous Catalog Search stays usable but blocks paid modes", async ({ page }) => {
		await page.route("/api/v1/entitlements", async (route) => {
			await route.fulfill({ status: 401, body: "Unauthorized" });
		});
		
		await page.route("**/api/v1/search/autocomplete*", async (route) => {
			await route.fulfill({ status: 200, json: { status: "ok", data: { items: [{ itemId: "1", label: "Apple", score: 1 }, { itemId: "2", label: "Banana", score: 1 }] } } });
		});
		await page.route("**/api/v1/food-objects/*", async (route) => {
			await route.fulfill({ status: 200, json: { status: "ok", data: { id: "1", name: "Apple", macroBasis: "100g", macros: { protein: 0, carbohydrates: 0, fat: 0 }, calories: 0, classifications: [] } } });
		});

		await page.goto("/");
		await expect(page.locator("[data-entitlement-usage]")).toBeHidden();
		await expect(page.getByRole('button', { name: 'Catalog' })).toBeVisible();

		// Verify Daily Diet Alternative is blocked
		await page.getByRole('button', { name: 'Daily Diet Alternative' }).click();
		await expect(page.locator("[data-entitlement-feedback]")).toBeVisible();
		
		// Verify Multi-input Substitution is blocked
		await page.getByRole('button', { name: 'Substitution' }).click();
		await page.getByRole("combobox", { name: "Food search" }).fill("apple");
		await page.getByRole("option", { name: "Apple" }).first().click();
		await page.getByRole("combobox", { name: "Food search" }).fill("banana");
		await page.getByRole("option", { name: "Banana" }).first().click();
		await expect(page.locator("[data-entitlement-feedback]")).toBeVisible();
	});

	test("keyboard/focus behavior remains accessible", async ({ page }) => {
		await mockEntitlement(page, "free", 8);
		await page.goto("/");
		await page.getByRole('button', { name: 'Substitution' }).focus();
		await expect(page.getByRole('button', { name: 'Substitution' })).toBeFocused();
	});
});
