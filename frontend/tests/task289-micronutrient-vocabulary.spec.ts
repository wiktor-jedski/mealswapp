import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page, type Route } from "@playwright/test";

// Implements DESIGN-005 MicronutrientVocabulary browser lifecycle and accessibility verification.

const ok = (data: unknown) => ({ status: "ok", requestId: "task-289", data });
type Entry = { key: string; displayName: string; unit: "g" | "mg" | "mcg"; active: boolean };

async function json(route: Route, status: number, body?: unknown): Promise<void> {
	await route.fulfill(body === undefined ? { status } : { status, contentType: "application/json", body: JSON.stringify(body) });
}

async function stubShell(page: Page): Promise<void> {
	await page.route("**/api/v1/**", async (route) => {
		const path = new URL(route.request().url()).pathname;
		if (path === "/api/v1/profile") return json(route, 200, ok({ userId: "admin-289", displayName: "Vocabulary Admin", unitSystem: "metric", themePreference: "system", requiresUnitRecalculation: false }));
		if (path === "/api/v1/auth/refresh") return json(route, 200, ok({ userId: "admin-289", role: "admin", hasVerifiedLoginMethod: true, accessExpiresAt: "2026-07-29T20:00:00Z", refreshExpiresAt: "2026-08-05T20:00:00Z" }));
		if (path === "/api/v1/billing/entitlement") return json(route, 200, ok({ userId: "admin-289", tier: "paid", status: "active", allowedModes: ["catalog", "substitution", "daily_diet", "daily_diet_alternative"], searchLimitPer24h: null, usageUsed: 0, usageRemaining: null, usageWindowStartedAt: "2026-07-29T00:00:00Z", trialExpiresAt: null, billingRecoveryState: "none" }));
		if (path === "/api/v1/auth/csrf-token") return json(route, 200, ok({ csrfToken: "csrf-task-289" }));
		if (path === "/api/v1/search-history") return json(route, 200, ok({ history: [] }));
		if (path === "/api/v1/saved-items") return json(route, 200, ok({ items: [] }));
		if (path === "/api/v1/account/export") return json(route, 200, ok({ format: "json", generatedAt: "2026-07-29T20:00:00Z", profile: {}, savedItems: [], searchHistory: [], dailyDiets: [], customItems: [] }));
		if (path === "/api/v1/admin/classifications") return json(route, 200, ok({ classifications: [] }));
		return json(route, 404, { status: "error", requestId: "task-289", error: { category: "validation", code: "not_found", message: "Not found", retryable: false } });
	});
}

test("administrator adds, edits, deactivates, and reactivates canonical entries with authoritative refresh", async ({ page }) => {
	let entries: Entry[] = [
		{ key: "Sodium", displayName: "Sodium", unit: "mg", active: true },
		{ key: "VitaminD", displayName: "Vitamin D", unit: "mcg", active: false }
	];
	const mutationHeaders: string[] = [];
	await page.emulateMedia({ reducedMotion: "reduce" });
	await stubShell(page);
	await page.route(/\/api\/v1\/admin\/micronutrients(\/.*)?$/, async (route) => {
		const request = route.request();
		const path = new URL(request.url()).pathname;
		if (request.method() === "GET") return json(route, 200, ok({ micronutrients: entries }));
		mutationHeaders.push(request.headers()["x-csrf-token"] ?? "");
		if (path === "/api/v1/admin/micronutrients") {
			const body = request.postDataJSON() as Omit<Entry, "active">;
			const entry = { ...body, active: true };
			entries = [...entries, entry];
			return json(route, 201, ok({ micronutrient: entry }));
		}
		const [, key, action] = /\/micronutrients\/([^/]+)\/([^/]+)$/.exec(path) ?? [];
		const current = entries.find((entry) => entry.key === key)!;
		const body = request.postDataJSON() as { displayName?: string; unit?: Entry["unit"] } | null;
		const updated: Entry = action === "display-name" ? { ...current, displayName: body!.displayName! }
			: action === "unit" ? { ...current, unit: body!.unit! }
				: { ...current, active: action === "reactivate" };
		entries = entries.map((entry) => entry.key === key ? updated : entry);
		return json(route, 200, ok({ micronutrient: updated }));
	});

	await page.goto("/admin");
	const vocabulary = page.locator("[data-admin-micronutrients]");
	await expect(vocabulary).toBeVisible();
	const addForm = vocabulary.getByRole("form", { name: "Add micronutrient" });
	await addForm.getByLabel("Canonical key").fill("VitaminK");
	await addForm.getByLabel("Display name").fill("Vitamin K");
	await addForm.getByLabel("Unit").selectOption("mcg");
	const createdResponse = page.waitForResponse((response) => response.url().endsWith("/api/v1/admin/micronutrients") && response.request().method() === "POST");
	await addForm.getByRole("button", { name: "Add micronutrient" }).click();
	expect((await createdResponse).status()).toBe(201);
	const vitaminK = vocabulary.locator('[data-micronutrient-key="VitaminK"]');
	await expect(vitaminK).toContainText("Active");
	await vitaminK.getByLabel("Display name").fill("Vitamin K1");
	await vitaminK.getByRole("button", { name: "Save name" }).click();
	await vitaminK.getByLabel("Unit").selectOption("mg");
	await vitaminK.getByRole("button", { name: "Deactivate" }).click();
	await expect(page.getByRole("alertdialog")).toBeVisible();
	await page.keyboard.press("Escape");
	await expect(vitaminK).toContainText("Active");
	await vitaminK.getByRole("button", { name: "Deactivate" }).click();
	await page.getByRole("button", { name: "Confirm deactivation" }).click();
	await expect(vitaminK).toContainText("Inactive");
	await vitaminK.getByRole("button", { name: "Reactivate" }).click();
	await expect(vitaminK).toContainText("Active");
	expect(mutationHeaders).toEqual(["csrf-task-289", "csrf-task-289", "csrf-task-289", "csrf-task-289", "csrf-task-289"]);
	expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
	const results = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
	expect(results.violations.filter(({ impact }) => impact === "serious" || impact === "critical")).toEqual([]);
	expect(await vocabulary.innerText()).not.toMatch(/owner|private food|email|user id/i);
});

test("administrator recovers from audit and refresh failures with keyboard controls on mobile dark theme", async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await page.emulateMedia({ colorScheme: "dark", reducedMotion: "reduce" });
	let entries: Entry[] = [{ key: "Sodium", displayName: "Sodium", unit: "mg", active: true }];
	let failMutation = true;
	let failRefresh = false;
	await stubShell(page);
	await page.route(/\/api\/v1\/admin\/micronutrients(\/.*)?$/, async (route) => {
		const request = route.request();
		if (request.method() === "GET") {
			if (failRefresh) return json(route, 503, { status: "error", requestId: "task-289", error: { category: "dependency", code: "dependency_unavailable", message: "Vocabulary refresh unavailable", retryable: true } });
			return json(route, 200, ok({ micronutrients: entries }));
		}
		if (failMutation) {
			failMutation = false;
			return json(route, 503, { status: "error", requestId: "task-289", error: { category: "dependency", code: "audit_write_failed", message: "The vocabulary action could not be recorded.", retryable: true } });
		}
		const [, key] = /\/micronutrients\/([^/]+)\/([^/]+)$/.exec(new URL(request.url()).pathname) ?? [];
		const current = entries.find((entry) => entry.key === key)!;
		const updated = { ...current, displayName: (request.postDataJSON() as { displayName: string }).displayName };
		entries = entries.map((entry) => entry.key === key ? updated : entry);
		return json(route, 200, ok({ micronutrient: updated }));
	});
	await page.goto("/admin");
	const vocabulary = page.locator("[data-admin-micronutrients]");
	const sodium = vocabulary.locator('[data-micronutrient-key="Sodium"]');
	const name = sodium.getByLabel("Display name");
	await name.fill("Sodium audit");
	await name.press("Enter");
	await expect(vocabulary.getByRole("alert")).toContainText("No change was shown as successful");
	failRefresh = true;
	await name.fill("Sodium recovered");
	await name.press("Enter");
	await expect(vocabulary.getByRole("status")).toContainText("successful result");
	await expect(name).toHaveValue("Sodium recovered");
	failRefresh = true;
	await vocabulary.getByRole("button", { name: "Refresh" }).click();
	await expect(vocabulary.getByRole("alert")).toContainText("No change was shown as successful");
	failRefresh = false;
	await vocabulary.getByRole("button", { name: "Refresh" }).click();
	await expect(sodium.getByLabel("Display name")).toHaveValue("Sodium recovered");
	expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
});
