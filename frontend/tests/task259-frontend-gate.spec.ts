import { mkdir } from "node:fs/promises";
import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page, type Route } from "@playwright/test";

// Implements DESIGN-009 UserAdminPanel functional, end-to-end, and accessibility gate for task 259.

const categoryId = "00000000-0000-4000-8000-000000000259";
const customItemId = "00000000-0000-4000-8000-000000000260";
const screenshotDirectory = "/tmp/mealswapp-task-259";
const ok = (data: unknown) => ({ status: "ok", requestId: "task-259", data });

async function json(route: Route, status: number, body?: unknown): Promise<void> {
	await route.fulfill(body === undefined ? { status } : { status, contentType: "application/json", body: JSON.stringify(body) });
}

async function stubShell(page: Page): Promise<void> {
	await page.route("**/api/v1/**", async (route) => {
		const url = new URL(route.request().url());
		if (url.pathname === "/api/v1/profile") return json(route, 200, ok({ userId: "admin-259", displayName: "Gate Admin", unitSystem: "metric", themePreference: "system", requiresUnitRecalculation: false }));
		if (url.pathname === "/api/v1/auth/refresh") return json(route, 200, ok({ userId: "admin-259", role: "admin", hasVerifiedLoginMethod: true, accessExpiresAt: "2026-07-22T13:00:00Z", refreshExpiresAt: "2026-07-29T13:00:00Z" }));
		if (url.pathname === "/api/v1/billing/entitlement") return json(route, 200, ok({ userId: "admin-259", tier: "paid", status: "active", allowedModes: ["catalog", "substitution", "daily_diet", "daily_diet_alternative"], searchLimitPer24h: null, usageUsed: 0, usageRemaining: null, usageWindowStartedAt: "2026-07-22T00:00:00Z", trialExpiresAt: null, billingRecoveryState: "none" }));
		if (url.pathname === "/api/v1/auth/csrf-token") return json(route, 200, ok({ csrfToken: "csrf-task-259" }));
		if (url.pathname === "/api/v1/search-history") return json(route, 200, ok({ history: [] }));
		if (url.pathname === "/api/v1/saved-items") return json(route, 200, ok({ items: [] }));
		if (url.pathname === "/api/v1/search/autocomplete") return json(route, 200, ok({ items: [{ itemId: "food-259", objectType: "food_item", label: "Tempeh", exactMatch: true, levenshteinDistance: 0, length: 6, rank: 1 }] }));
		if (url.pathname === "/api/v1/food-objects/food-259") return json(route, 200, ok({ id: "food-259", objectType: "food_item", name: "Tempeh", physicalState: "solid", imageUrl: null, classifications: [], primaryFoodCategory: null, macros: { protein: 20, carbohydrates: 8, fat: 11 }, macroBasis: "100g", calories: 211 }));
		return json(route, 404, { status: "error", requestId: "task-259-unhandled", error: { category: "validation", code: "not_found", message: "Not found", retryable: false } });
	});
}

// Verifies IT-ARCH-009-005, ARCH-009, DESIGN-009 TagManager, and SW-REQ-057.
test("classification administration refreshes substitution filters and remains accessible in every viewport and theme", async ({ page }, testInfo) => {
	let categories = [{ id: categoryId, name: "Fermented", kind: "food_category" as const }];
	await page.emulateMedia({ reducedMotion: "reduce" });
	await stubShell(page);
	await page.route(/\/api\/v1\/admin\/classifications(\/food_category)?(\?.*)?$/, async (route) => {
		const url = new URL(route.request().url());
		if (route.request().method() === "POST") {
			const body = route.request().postDataJSON() as { name: string };
			categories = [{ id: categoryId, name: body.name, kind: "food_category" }];
			return json(route, 201, ok({ classification: categories[0] }));
		}
		return json(route, 200, ok({ classifications: url.searchParams.get("kind") === "food_category" ? categories : [] }));
	});
	await page.route(/\/api\/v1\/admin\/external-search(\?.*)?$/, (route) => json(route, 200, ok({ provider: "all", page: 1, candidates: [], warnings: [] })));
	await page.route(/\/api\/v1\/search\/filter-options(\?.*)?$/, (route) => json(route, 200, ok({ mode: "substitution", options: categories.map(({ id, name }) => ({ filterId: id, kind: "food_category", label: name, includeAllowed: true, excludeAllowed: true, excludes: [] })) })));

	await page.goto("/admin");
	await expect(page.locator("[data-admin-data-management]")).toBeVisible();
	await page.getByLabel("Name", { exact: true }).last().fill("Cultured foods");
	await page.getByRole("button", { name: "Create", exact: true }).click();
	await expect(page.getByRole("listitem").filter({ hasText: "Cultured foods" })).toBeVisible();

	for (const theme of ["light", "dark"] as const) {
		if ((await page.locator("html").getAttribute("data-theme")) !== theme) {
			const toggle = page.getByLabel("Theme preference");
			if (!(await toggle.isVisible())) await page.getByLabel("Open activity sidebar").click();
			await toggle.click();
		}
		await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
		const results = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
		expect(results.violations.filter(({ impact }) => impact === "serious" || impact === "critical")).toEqual([]);
		expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
		expect(await page.locator("body").innerText()).not.toMatch(/raw provider|stack trace|audit_write_failed/i);
		await mkdir(screenshotDirectory, { recursive: true });
		await page.screenshot({ path: `${screenshotDirectory}/task-259-${testInfo.project.name}-${theme}.png`, fullPage: true, animations: "disabled" });
	}

	await page.goto("/?mode=substitution");
	await page.getByLabel("Food search").fill("tempeh");
	await page.getByRole("listbox", { name: "Autocomplete suggestions" }).getByRole("option", { name: "Tempeh" }).click();
	await page.locator("#substitution-include-filter").focus();
	await expect(page.locator("[data-substitution-include-options]")).toContainText("Cultured foods");
});

// Supporting direct-transport regression for DESIGN-008 AccountDeleter and
// SW-REQ-043/SW-REQ-072/SW-REQ-073; SWE.5 UI evidence is task261-real-admin-flow.spec.ts.
test("browser export reflects private custom-item deletion without leaking ownership", async ({ page }) => {
	let customItemPresent = true;
	let deleteHeaders: Record<string, string> = {};
	await stubShell(page);
	await page.route(/\/api\/v1\/profile\/export\?format=json$/, (route) => json(route, 200, ok({ format: "json", generatedAt: "2026-07-22T12:00:00Z", profile: {}, savedItems: [], searchHistory: [], dailyDiets: [], customItems: customItemPresent ? [{ id: customItemId, name: "Private tempeh", physicalState: "solid", prepTimeMinutes: 0, macrosPer100: { protein: 20, carbohydrates: 8, fat: 11 }, micros: {}, foodCategories: [], culinaryRoles: [] }] : [] })));
	await page.route(`**/api/v1/custom-items/${customItemId}`, async (route) => {
		deleteHeaders = route.request().headers();
		customItemPresent = false;
		await json(route, 204);
	});
	await page.goto("/");

	const result = await page.evaluate(async (itemId) => {
		const before = await fetch("/api/v1/profile/export?format=json", { credentials: "include" }).then((response) => response.json());
		const csrf = await fetch("/api/v1/auth/csrf-token", { credentials: "include" }).then((response) => response.json());
		const deletion = await fetch(`/api/v1/custom-items/${itemId}`, { method: "DELETE", credentials: "include", headers: { "X-CSRF-Token": csrf.data.csrfToken } });
		const after = await fetch("/api/v1/profile/export?format=json", { credentials: "include" }).then((response) => response.json());
		return { before: before.data.customItems, deletionStatus: deletion.status, after: after.data.customItems };
	}, customItemId);

	expect(result.before).toHaveLength(1);
	expect(result.before[0]).toMatchObject({ id: customItemId, name: "Private tempeh" });
	expect(result.before[0]).not.toHaveProperty("ownerId");
	expect(result.deletionStatus).toBe(204);
	expect(result.after).toEqual([]);
	expect(deleteHeaders["x-csrf-token"]).toBe("csrf-task-259");
});
