import { expect, test, type Page, type Route } from "@playwright/test";
import type { FilterOption, SearchRequest } from "../src/lib/api/generated";

// Implements DESIGN-001 SearchView dynamic substitution filter browser verification.

const selectedItem = {
	id: "food-apple",
	objectType: "food_item" as const,
	name: "Apple",
	physicalState: "solid" as const,
	imageUrl: null,
	classifications: [
		{ id: "fruit-id", name: "Selected Fruit", kind: "food_category" as const },
		{ id: "snack-id", name: "Snack", kind: "culinary_role" as const }
	],
	primaryFoodCategory: { id: "fruit-id", name: "Selected Fruit", kind: "food_category" as const },
	macros: { protein: 1, carbohydrates: 14, fat: 0.2 },
	macroBasis: "100g" as const,
	calories: 62
};

function option(filterId: string, kind: FilterOption["kind"], label: string, includeAllowed = true, excludeAllowed = true): FilterOption {
	return { filterId, kind, label, includeAllowed, excludeAllowed, excludes: [] };
}

async function fulfill(route: Route, status: number, body: unknown): Promise<void> {
	await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

async function stubApplication(page: Page, filterResponse: (route: Route, requestNumber: number) => Promise<void>, requests: SearchRequest[]): Promise<void> {
	let filterRequest = 0;
	await page.route("**/api/v1/**", async (route) => {
		const url = new URL(route.request().url());
		if (url.pathname === "/api/v1/search/filter-options") return filterResponse(route, ++filterRequest);
		if (url.pathname === "/api/v1/search/autocomplete") return fulfill(route, 200, { status: "ok", requestId: "autocomplete", data: { items: [{ itemId: "food-apple", objectType: "food_item", label: "Apple", exactMatch: true, levenshteinDistance: 0, length: 5, rank: 1 }] } });
		if (url.pathname === "/api/v1/food-objects/food-apple") return fulfill(route, 200, { status: "ok", requestId: "food", data: selectedItem });
		if (url.pathname === "/api/v1/search" && route.request().method() === "POST") {
			requests.push(await route.request().postDataJSON() as SearchRequest);
			return fulfill(route, 200, { status: "ok", requestId: "search", data: { items: [], totalCount: 0, page: 1, similarityScores: [], similarityMetadata: [], warnings: [] } });
		}
		if (url.pathname === "/api/v1/auth/refresh") return fulfill(route, 200, { status: "ok", requestId: "auth", data: { userId: "user", role: "user", hasVerifiedLoginMethod: true, accessExpiresAt: "2026-07-22T00:00:00Z", refreshExpiresAt: "2026-07-29T00:00:00Z" } });
		if (url.pathname === "/api/v1/profile") return fulfill(route, 200, { status: "ok", requestId: "profile", data: { userId: "user", displayName: "User", unitSystem: "metric", themePreference: "system", requiresUnitRecalculation: false } });
		if (url.pathname === "/api/v1/billing/entitlement") return fulfill(route, 200, { status: "ok", requestId: "entitlement", data: { userId: "user", tier: "paid", status: "active", allowedModes: ["catalog", "substitution", "daily_diet", "daily_diet_alternative"], searchLimitPer24h: null, usageUsed: 0, usageRemaining: null, usageWindowStartedAt: "2026-07-21T00:00:00Z", trialExpiresAt: null, billingRecoveryState: "none" } });
		if (url.pathname === "/api/v1/search-history") return fulfill(route, 200, { status: "ok", requestId: "history", data: { history: [] } });
		if (url.pathname === "/api/v1/saved-items") return fulfill(route, 200, { status: "ok", requestId: "saved", data: { items: [] } });
		return fulfill(route, 404, { status: "error", requestId: "unhandled", error: { category: "validation", code: "not_found", message: "Not found", retryable: false } });
	});
}

async function addSelectedItem(page: Page): Promise<void> {
	await page.goto("/?mode=substitution");
	await page.getByLabel("Food search").fill("apple");
	await page.getByRole("listbox", { name: "Autocomplete suggestions" }).getByRole("option", { name: "Apple" }).click();
	await expect(page.locator("[data-substitution-card]")).toHaveCount(1);
}

test("renders backend order and labels, merges selected classifications, sends IDs, and ignores stale refreshes", async ({ page }) => {
	const searchRequests: SearchRequest[] = [];
	const inventories = [
		[option("solid-id", "physical_state", "Stałe"), option("fruit-id", "food_category", "Owoc serwera"), option("vegan-id", "dietary_preset", "Wegańskie", false, true)],
		[option("fruit-id", "food_category", "Stara nazwa")],
		[option("fruit-id", "food_category", "Nowa nazwa")]
	];
	await stubApplication(page, async (route, requestNumber) => {
		if (requestNumber === 2) await new Promise((resolve) => setTimeout(resolve, 250));
		await fulfill(route, 200, { status: "ok", requestId: `filters-${requestNumber}`, data: { mode: "substitution", options: inventories[Math.min(requestNumber - 1, 2)] } });
	}, searchRequests);
	await addSelectedItem(page);

	const include = page.locator("#substitution-include-filter");
	await include.focus();
	await expect(page.locator("[data-substitution-include-options] [role=option]")).toHaveText(["Stałe Physical State", "Owoc serwera Food Category", "Snack Culinary Role"]);
	const solidOption = page.locator("[data-substitution-include-options] [role=option]").first();
	await solidOption.focus();
	await solidOption.press("Enter");
	await expect(page.getByRole("button", { name: "Remove include filter Stałe" })).toBeVisible();
	await page.getByRole("button", { name: "Find substitutions" }).click();
	await expect.poll(() => searchRequests.length).toBe(1);
	expect(searchRequests[0]?.filters).toContainEqual({ filterId: "solid-id", kind: "physical_state", include: true });

	await page.evaluate(() => window.dispatchEvent(new Event("focus")));
	await page.evaluate(() => window.dispatchEvent(new Event("focus")));
	await page.waitForTimeout(350);
	await include.focus();
	await expect(page.locator("[data-substitution-include-options]")).toContainText("Nowa nazwa");
	await expect(page.locator("[data-substitution-include-options]")).not.toContainText("Stara nazwa");
});

test("unavailable and empty inventories remain recoverable without invented policy", async ({ page }) => {
	const requests: SearchRequest[] = [];
	await stubApplication(page, async (route, requestNumber) => {
		if (requestNumber === 1) return fulfill(route, 503, { status: "error", requestId: "filters-error", error: { category: "dependency", code: "unavailable", message: "internal detail", retryable: true } });
		return fulfill(route, 200, { status: "ok", requestId: "filters-empty", data: { mode: "substitution", options: [] } });
	}, requests);
	await addSelectedItem(page);

	await expect(page.locator("[data-filter-options-error]")).toContainText("temporarily unavailable");
	await page.getByRole("button", { name: "Retry filter options" }).click();
	await expect(page.locator("[data-filter-options-empty]")).toBeVisible();
	await page.locator("#substitution-exclude-filter").focus();
	const options = page.locator("[data-substitution-exclude-options] [role=option]");
	await expect(options).toHaveCount(2);
	await expect(options).toHaveText(["Selected Fruit Food Category", "Snack Culinary Role"]);
	await expect(page.locator("[data-substitution-filters]")).not.toContainText(/Dairy-free|Gluten-free|Vegan|Vegetarian/);
});

test("schema-invalid inventories fail closed while selected classifications remain safely usable", async ({ page }) => {
	const requests: SearchRequest[] = [];
	const malformed = [
		[{ ...option("legacy", "food_category", "Legacy"), kind: "food_object_type" }],
		[{ ...option("ordered", "food_category", "Ordered"), sortOrder: 1 }],
		[{ ...option("oversized", "food_category", "x".repeat(201)) }]
	];
	await stubApplication(page, async (route, requestNumber) => fulfill(route, 200, {
		status: "ok",
		requestId: `filters-${requestNumber}`,
		data: { mode: "substitution", options: malformed[requestNumber - 1] ?? [] }
	}), requests);
	await addSelectedItem(page);

	for (let attempt = 0; attempt < malformed.length; attempt++) {
		await expect(page.locator("[data-filter-options-error]")).toContainText("temporarily unavailable");
		await page.locator("#substitution-exclude-filter").focus();
		await expect(page.locator("[data-substitution-exclude-options] [role=option]")).toHaveText(["Selected Fruit Food Category", "Snack Culinary Role"]);
		await page.getByRole("button", { name: "Retry filter options" }).click();
	}

	await expect(page.locator("[data-filter-options-empty]")).toBeVisible();
	await expect(page.locator("[data-substitution-filters]")).not.toContainText(/Legacy|Ordered|x{20}/);
});
