import { expect, test, type Page, type Route } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

// Implements DESIGN-009 ItemCurator, TagManager, and UserAdminPanel task-256 browser verification.

const itemId = "00000000-0000-4000-8000-000000000101";
const categoryId = "00000000-0000-4000-8000-000000000102";
const categoryParentId = "00000000-0000-4000-8000-000000000110";
const conflictId = "00000000-0000-4000-8000-000000000103";
const roleId = "00000000-0000-4000-8000-000000000104";
const userId = "00000000-0000-4000-8000-000000000105";
const deletionId = "00000000-0000-4000-8000-000000000106";
const secondItemId = "00000000-0000-4000-8000-000000000108";
const ok = (data: unknown) => ({ status: "ok", requestId: "task-256", data });
const failure = (status: number, code: string) => ({ status: "error", requestId: "task-256-error", error: { category: status === 409 ? "validation" : "server", code, message: "Safe failure", retryable: true } });

interface State {
	item?: Record<string, unknown>;
	categories: Array<{ id: string; name: string; kind: "food_category"; parentId?: string }>;
	roles: Array<{ id: string; name: string; kind: "culinary_role" }>;
	user: Record<string, unknown>;
	conflictNextRetry?: boolean;
	lastItemPut?: Record<string, unknown>;
	lastClassificationPut?: Record<string, unknown>;
	deletedItemIds: string[];
	classificationReads: number;
	authoritativeNameAfterPut?: string;
	userLookupDelays?: Record<string, number>;
	itemReadDelays?: Record<string, number>;
	classificationMutationDelays?: Record<string, number>;
}

async function json(route: Route, status: number, body?: unknown): Promise<void> {
	await route.fulfill(body === undefined ? { status } : { status, contentType: "application/json", body: JSON.stringify(body) });
}

async function stubApp(page: Page): Promise<State> {
	const state: State = {
		categories: [{ id: categoryParentId, name: "Food", kind: "food_category" }, { id: categoryId, name: "Produce", kind: "food_category", parentId: categoryParentId }, { id: conflictId, name: "In use", kind: "food_category" }],
		roles: [{ id: roleId, name: "Base", kind: "culinary_role" }],
		user: { id: userId, email: "minimal@example.test", emailVerified: true, createdAt: "2026-07-21T00:00:00Z", deletion: { requestId: deletionId, status: "failed", failureCategory: "unknown", retryCount: 1, requestedAt: "2026-07-20T00:00:00Z" } },
		deletedItemIds: [], classificationReads: 0
	};
	const session = ok({ userId: "admin-256", role: "admin", hasVerifiedLoginMethod: true, accessExpiresAt: "2026-07-21T22:00:00Z", refreshExpiresAt: "2026-07-28T22:00:00Z" });
	await page.route(/\/api\/v1\/(profile|auth\/refresh|billing\/entitlement|search-history|saved-items|search\/autocomplete|auth\/csrf-token)(\?.*)?$/, async (route) => {
		const url = route.request().url();
		if (url.includes("/profile")) return json(route, 200, ok({ userId: "admin-256", displayName: "Admin", unitSystem: "metric", themePreference: "system", requiresUnitRecalculation: false }));
		if (url.includes("/auth/refresh")) return json(route, 200, session);
		if (url.includes("/billing/entitlement")) return json(route, 200, ok({ userId: "admin-256", tier: "paid", status: "active", allowedModes: ["catalog", "substitution", "daily_diet", "daily_diet_alternative"], searchLimitPer24h: null, usageUsed: 0, usageRemaining: null, usageWindowStartedAt: "2026-07-21T00:00:00Z", trialExpiresAt: null, billingRecoveryState: "none" }));
		if (url.includes("csrf-token")) return json(route, 200, ok({ csrfToken: "csrf-task-256" }));
		if (url.includes("search-history")) return json(route, 200, ok({ history: [] }));
		if (url.includes("saved-items")) return json(route, 200, ok({ items: [] }));
		return json(route, 200, ok({ items: [] }));
	});
	await page.route(/\/api\/v1\/admin\//, async (route) => {
		const request = route.request(); const url = new URL(request.url()); const method = request.method(); const path = url.pathname;
		if (path === "/api/v1/admin/classifications" && method === "GET") { state.classificationReads++; return json(route, 200, ok({ classifications: url.searchParams.get("kind") === "food_category" ? state.categories : state.roles })); }
		if (path === "/api/v1/admin/classifications/food_category" && method === "POST") { const body = request.postDataJSON(); const delay = state.classificationMutationDelays?.[body.name] ?? 0; if (delay) await new Promise((resolve) => setTimeout(resolve, delay)); const classification = { id: body.name === "Slow category" ? "00000000-0000-4000-8000-000000000109" : "00000000-0000-4000-8000-000000000107", name: body.name, kind: "food_category" as const }; state.categories.push(classification); return json(route, 201, ok({ classification })); }
		if (path === `/api/v1/admin/classifications/${categoryId}` && method === "PUT") { const body = request.postDataJSON() as Record<string, unknown>; state.lastClassificationPut = body; state.categories[1] = { ...state.categories[1]!, name: String(body.name), ...(typeof body.parentId === "string" ? { parentId: body.parentId } : { parentId: undefined }) }; return json(route, 200, ok({ classification: state.categories[1] })); }
		if (path === `/api/v1/admin/classifications/${conflictId}` && method === "DELETE") return json(route, 409, failure(409, "classification_in_use"));
		if (path.startsWith("/api/v1/admin/classifications/") && method === "DELETE") { const id = path.split("/").at(-1); state.categories = state.categories.filter((value) => value.id !== id); state.roles = state.roles.filter((value) => value.id !== id); return json(route, 204); }
		if (path === "/api/v1/admin/items" && method === "POST") { const body = request.postDataJSON(); state.item = { ...body, id: itemId, prepTimeMinutes: 0, foodCategories: [], culinaryRoles: [] }; return json(route, 201, ok(state.item)); }
		if (path === `/api/v1/admin/items/${itemId}` && method === "GET") { const delay = state.itemReadDelays?.[itemId] ?? 0; if (delay) await new Promise((resolve) => setTimeout(resolve, delay)); return state.item ? json(route, 200, ok(state.item)) : json(route, 404, failure(404, "not_found")); }
		if (path === `/api/v1/admin/items/${secondItemId}` && method === "GET") return json(route, 200, ok({ ...state.item, id: secondItemId, name: "Second item" }));
		if (path === `/api/v1/admin/items/${itemId}` && method === "PUT") { const body = request.postDataJSON() as Record<string, unknown>; state.lastItemPut = body; if (body.name === "Audit fail") return json(route, 500, failure(500, "audit_write_failed")); const mutationProjection = { ...state.item, ...body }; state.item = { ...mutationProjection, ...(state.authoritativeNameAfterPut ? { name: state.authoritativeNameAfterPut } : {}) }; return json(route, 200, ok(mutationProjection)); }
		if (path.startsWith("/api/v1/admin/items/") && method === "DELETE") { const id = path.split("/").at(-1)!; state.deletedItemIds.push(id); if (id === itemId) state.item = undefined; return json(route, 204); }
		if (path === "/api/v1/admin/users" && method === "GET") { const query = url.searchParams.get("email") ?? url.searchParams.get("userId") ?? ""; const delay = state.userLookupDelays?.[query] ?? 0; if (delay) await new Promise((resolve) => setTimeout(resolve, delay)); return json(route, 200, ok({ users: [{ ...state.user, email: query.includes("@") ? query : state.user.email }] })); }
		if (path === `/api/v1/admin/users/${userId}/deletion-requests/${deletionId}/retry` && method === "POST") {
			state.user = { ...state.user, deletion: { requestId: deletionId, status: "pending", retryCount: 0, requestedAt: "2026-07-20T00:00:00Z" } };
			if (state.conflictNextRetry) { state.conflictNextRetry = false; return json(route, 409, failure(409, "conflict")); }
			return json(route, 200, ok({ requestId: deletionId, status: "pending" }));
		}
		return json(route, 404, failure(404, "not_found"));
	});
	return state;
}

async function openAdmin(page: Page): Promise<void> { await page.goto("/admin"); await expect(page.locator("[data-admin-data-management]")).toBeVisible(); }

async function assertConfirmationContainment(page: Page): Promise<void> {
	const confirm = page.getByRole("button", { name: "Confirm" });
	const cancel = page.getByRole("button", { name: "Cancel" });
	const desktopOutside = page.locator("[data-sidebar-nav-search]");
	const mobileOutside = page.locator("[data-sidebar-mobile-toggle]");
	const outside = await desktopOutside.isVisible() ? desktopOutside : mobileOutside;

	await expect(confirm).toBeFocused();
	await page.keyboard.press("Tab");
	await expect(cancel).toBeFocused();
	await page.keyboard.press("Tab");
	await expect(confirm).toBeFocused();
	await page.keyboard.press("Shift+Tab");
	await expect(cancel).toBeFocused();
	await page.keyboard.press("Shift+Tab");
	await expect(confirm).toBeFocused();

	await outside.focus();
	await expect(confirm).toBeFocused();
	const box = await outside.boundingBox();
	expect(box).not.toBeNull();
	await page.mouse.click(box!.x + box!.width / 2, box!.y + box!.height / 2);
	await expect(page.locator("[data-admin-confirmation]")).toBeVisible();
	await expect(confirm).toBeFocused();
	await expect(page.locator("[data-admin-data-management]")).toBeVisible();
	await expect(page.locator("[data-sidebar]")).toHaveAttribute("data-mobile-open", "false");
}

async function cancelWithKeyboard(page: Page): Promise<void> {
	await assertConfirmationContainment(page);
	await page.keyboard.press("Tab");
	await expect(page.getByRole("button", { name: "Cancel" })).toBeFocused();
	await page.keyboard.press("Enter");
	await expect(page.locator("[data-admin-confirmation]")).toHaveCount(0);
	expect(await page.evaluate(() => document.activeElement?.tagName)).not.toBe("BODY");
}

test("keyboard cancellation restores focus for every destructive confirmation and uses a safe fallback", async ({ page }) => {
	const state = await stubApp(page);
	state.item = { id: itemId, name: "Focus item", physicalState: "solid", prepTimeMinutes: 0, macrosPer100: { protein: 1, carbohydrates: 2, fat: 3 }, micros: {}, foodCategories: [], culinaryRoles: [] };
	await openAdmin(page);
	await page.getByLabel("Item ID").fill(itemId);
	await page.getByRole("button", { name: "Load" }).click();

	const itemDelete = page.getByRole("button", { name: "Delete item" });
	await itemDelete.focus();
	await itemDelete.press("Enter");
	await cancelWithKeyboard(page);
	await expect(itemDelete).toBeFocused();

	const classificationDelete = page.getByRole("listitem").filter({ hasText: "Produce" }).getByRole("button", { name: "Delete" });
	await classificationDelete.focus();
	await classificationDelete.press("Enter");
	await cancelWithKeyboard(page);
	await expect(classificationDelete).toBeFocused();
	await classificationDelete.press("Enter");
	await classificationDelete.evaluate((button) => button.remove());
	await cancelWithKeyboard(page);
	expect(await page.evaluate(() => document.activeElement?.closest('section[aria-labelledby="classifications-title"]') !== null)).toBe(true);

	await page.getByLabel("Email or user ID").fill("minimal@example.test");
	await page.getByRole("button", { name: "Look up" }).click();
	const retry = page.getByRole("button", { name: "Retry legal deletion" });
	await retry.focus();
	await retry.press("Enter");
	await cancelWithKeyboard(page);
	await expect(retry).toBeFocused();

	await retry.press("Enter");
	await retry.evaluate((button: HTMLButtonElement) => { button.disabled = true; });
	await cancelWithKeyboard(page);
	await expect(page.locator("[data-admin-user]")).toContainText("minimal@example.test");
	expect(await page.evaluate(() => document.activeElement?.closest('section[aria-labelledby="user-admin-title"]') !== null)).toBe(true);
});

// Verifies IT-ARCH-009-006, ARCH-009, DESIGN-009 ItemCurator, and SW-REQ-056.
test("manual global item CRUD validates, confirms, refreshes, and never shows audit false success", async ({ page }) => {
	await stubApp(page); await openAdmin(page);
	await page.getByLabel("Physical state").selectOption("liquid");
	await page.getByLabel("Name", { exact: true }).first().fill("Broth");
	await page.getByLabel("Protein per 100").fill("2"); await page.getByLabel("Carbohydrates per 100").fill("3"); await page.getByLabel("Fat per 100").fill("1");
	await page.getByRole("button", { name: "Create item" }).click();
	await expect(page.locator("[data-admin-item-error]")).toContainText("positive density");
	await page.getByLabel("Density (g/ml)").fill("1.01"); await page.getByRole("button", { name: "Create item" }).click();
	await expect(page.getByText("Item created and refreshed.")).toBeVisible();
	await page.getByLabel("Name", { exact: true }).first().fill("Stock"); await page.getByRole("button", { name: "Save item" }).click();
	await expect(page.getByText("Item saved and refreshed.")).toBeVisible();
	await page.getByLabel("Name", { exact: true }).first().fill("Audit fail"); await page.getByRole("button", { name: "Save item" }).click();
	await expect(page.locator("[data-admin-item-error]")).toContainText("did not complete"); await expect(page.getByLabel("Name", { exact: true }).first()).toHaveValue("Stock");
	await page.getByRole("button", { name: "Delete item" }).click(); await expect(page.getByRole("button", { name: "Confirm" })).toBeFocused(); await page.getByRole("button", { name: "Cancel" }).click();
	await expect(page.getByRole("button", { name: "Save item" })).toBeVisible(); await page.getByRole("button", { name: "Delete item" }).click(); await page.getByRole("button", { name: "Confirm" }).click();
	await expect(page.getByText("Item deleted after server confirmation.")).toBeVisible();
});

// Verifies IT-ARCH-009-005 and IT-ARCH-009-007, ARCH-009,
// DESIGN-009 TagManager/UserAdminPanel, and SW-REQ-054/SW-REQ-057/SW-REQ-073.
test("classification conflicts and legal deletion retries preserve authoritative state", async ({ page }) => {
	const state = await stubApp(page); await openAdmin(page);
	await page.getByLabel("Name", { exact: true }).last().fill("Vegetable"); await page.getByRole("button", { name: "Create", exact: true }).click(); await expect(page.getByRole("listitem").filter({ hasText: "Vegetable" })).toBeVisible();
	const produce = page.getByRole("listitem").filter({ hasText: "Produce" }); await produce.getByRole("button", { name: "Rename" }).click(); await page.getByLabel("Name", { exact: true }).last().fill("Fresh produce"); await page.getByRole("button", { name: "Save rename" }).click(); await expect(page.getByRole("listitem").filter({ hasText: "Fresh produce" })).toBeVisible();
	expect(state.lastClassificationPut).toEqual({ name: "Fresh produce", parentId: categoryParentId });
	expect(state.categories.find(({ id }) => id === categoryId)?.parentId).toBe(categoryParentId);
	const inUse = page.getByRole("listitem").filter({ hasText: "In use" }); await inUse.getByRole("button", { name: "Delete" }).click(); await page.getByRole("button", { name: "Confirm" }).click(); await expect(page.locator("[data-admin-classification-error]")).toContainText("authoritative data"); await expect(page.getByRole("listitem").filter({ hasText: "In use" })).toBeVisible();
	await page.getByLabel("Email or user ID").fill("minimal@example.test"); await page.getByRole("button", { name: "Look up" }).click();
	await expect(page.locator("[data-admin-user]")).toContainText("unknown · retries 1"); state.conflictNextRetry = true; await page.getByRole("button", { name: "Retry legal deletion" }).click(); await page.getByRole("button", { name: "Confirm" }).click();
	await expect(page.locator("[data-admin-user-error]")).toContainText("authoritative data"); await expect(page.locator("[data-admin-user]")).toContainText("pending"); await expect(page.getByText("Deletion retry accepted and authoritative state refreshed.")).toHaveCount(0);
	state.user = { ...state.user, deletion: { requestId: deletionId, status: "failed", failureCategory: "permanent", retryCount: 0, requestedAt: "2026-07-20T00:00:00Z" } }; await page.getByRole("button", { name: "Look up" }).click(); await page.getByRole("button", { name: "Retry legal deletion" }).click(); await page.getByRole("button", { name: "Confirm" }).click();
	await expect(page.getByText("Deletion retry accepted and authoritative state refreshed.")).toBeVisible(); await expect(page.locator("[data-admin-user]")).toContainText("pending"); await expect(page.getByRole("button", { name: "Retry legal deletion" })).toHaveCount(0);
});

test("unused classification delete reloads the authoritative hierarchy", async ({ page }) => {
	const state = await stubApp(page); await openAdmin(page);
	const readsBeforeDelete = state.classificationReads;
	const produce = page.getByRole("listitem").filter({ hasText: "Produce" });
	await produce.getByRole("button", { name: "Delete" }).click(); await page.getByRole("button", { name: "Confirm" }).click();
	await expect(page.getByText("Classification deleted and refreshed.")).toBeVisible();
	await expect(page.getByRole("listitem").filter({ hasText: "Produce" })).toHaveCount(0);
	expect(state.classificationReads).toBeGreaterThanOrEqual(readsBeforeDelete + 2);
});

test("item replacement preserves all fields and renders the differing authoritative follow-up", async ({ page }) => {
	const state = await stubApp(page);
	state.item = {
		id: itemId, name: "Imported milk", physicalState: "liquid", prepTimeMinutes: 12, averageUnitWeightGrams: 250, averageServingVolumeMilliliters: 240,
		densityGramsPerMilliliter: 1.2, densitySourceProvider: "usda", densitySourceFoodId: "171265", densitySourceKind: "imported",
		macrosPer100: { protein: 10, carbohydrates: 110, fat: 5 }, micros: { sodium: 42 }, foodCategoryIds: [categoryId], culinaryRoleIds: [roleId],
		foodCategories: [{ id: categoryId, name: "Produce", kind: "food_category" }], culinaryRoles: [{ id: roleId, name: "Base", kind: "culinary_role" }], imageUrl: "https://images.example.test/milk.png"
	};
	state.authoritativeNameAfterPut = "Authoritative milk";
	await openAdmin(page); await page.getByLabel("Item ID").fill(itemId); await page.getByRole("button", { name: "Load" }).click();
	await expect(page.getByLabel("Image URL")).toHaveValue("https://images.example.test/milk.png");
	await page.getByLabel("Name", { exact: true }).first().fill("Submitted milk"); await page.getByRole("button", { name: "Save item" }).click();
	await expect(page.getByLabel("Name", { exact: true }).first()).toHaveValue("Authoritative milk");
	await expect(page.getByText("Item saved and refreshed.")).toBeVisible();
	expect(state.lastItemPut).toMatchObject({
		name: "Submitted milk", prepTimeMinutes: 12, averageUnitWeightGrams: 250, averageServingVolumeMilliliters: 240,
		densityGramsPerMilliliter: 1.2, densitySourceProvider: "usda", densitySourceFoodId: "171265", densitySourceKind: "imported",
		macrosPer100: { protein: 10, carbohydrates: 110, fat: 5 }, micros: { sodium: 42 }, foodCategoryIds: [categoryId], culinaryRoleIds: [roleId], imageUrl: "https://images.example.test/milk.png"
	});
});

test("confirmation target cannot race mutable item state", async ({ page }) => {
	const state = await stubApp(page);
	state.item = { id: itemId, name: "First item", physicalState: "solid", prepTimeMinutes: 0, macrosPer100: { protein: 1, carbohydrates: 2, fat: 3 }, micros: {}, foodCategories: [], culinaryRoles: [] };
	await openAdmin(page); await page.getByLabel("Item ID").fill(itemId); await page.getByRole("button", { name: "Load" }).click(); await page.getByRole("button", { name: "Delete item" }).click();
	await expect(page.locator("[data-admin-background]")).toHaveAttribute("inert", "");
	await page.locator("[data-admin-confirmation]").evaluate((dialog: HTMLDialogElement) => dialog.close());
	await page.locator("[data-admin-background]").evaluate((element) => element.removeAttribute("inert"));
	await page.getByRole("button", { name: "New item" }).click();
	await page.locator("[data-admin-confirmation]").evaluate((dialog: HTMLDialogElement) => dialog.showModal());
	await page.getByRole("button", { name: "Confirm" }).click();
	await expect(page.locator("[data-admin-item-error]")).toContainText("no longer current");
	expect(state.deletedItemIds).toEqual([]);
});

test("older item reads and user lookups cannot overwrite newer state", async ({ page }) => {
	const state = await stubApp(page);
	state.item = { id: itemId, name: "Slow item", physicalState: "solid", prepTimeMinutes: 0, macrosPer100: { protein: 1, carbohydrates: 2, fat: 3 }, micros: {}, foodCategories: [], culinaryRoles: [] };
	state.itemReadDelays = { [itemId]: 200 };
	state.userLookupDelays = { "slow@example.test": 200, "latest@example.test": 5 };
	await openAdmin(page);
	await page.getByLabel("Item ID").fill(itemId); await page.locator('form[aria-label="Load global item"]').evaluate((form: HTMLFormElement) => form.requestSubmit());
	await page.getByLabel("Item ID").fill(secondItemId); await page.locator('form[aria-label="Load global item"]').evaluate((form: HTMLFormElement) => form.requestSubmit());
	await expect(page.getByLabel("Name", { exact: true }).first()).toHaveValue("Second item"); await page.waitForTimeout(250); await expect(page.getByLabel("Name", { exact: true }).first()).toHaveValue("Second item");
	await page.getByLabel("Email or user ID").fill("slow@example.test"); await page.locator('form[aria-label="User lookup"]').evaluate((form: HTMLFormElement) => form.requestSubmit());
	await page.getByLabel("Email or user ID").fill("latest@example.test"); await page.locator('form[aria-label="User lookup"]').evaluate((form: HTMLFormElement) => form.requestSubmit());
	await expect(page.locator("[data-admin-user]")).toContainText("latest@example.test"); await page.waitForTimeout(250); await expect(page.locator("[data-admin-user]")).toContainText("latest@example.test");
});

test("older classification mutations and refreshes cannot overwrite the latest projection", async ({ page }) => {
	const state = await stubApp(page); state.classificationMutationDelays = { "Slow category": 200, "Latest category": 5 }; await openAdmin(page);
	const name = page.getByLabel("Name", { exact: true }).last(); const form = page.locator('form[aria-label="Classification form"]');
	await name.fill("Slow category"); await form.evaluate((element: HTMLFormElement) => element.requestSubmit());
	await name.fill("Latest category"); await form.evaluate((element: HTMLFormElement) => element.requestSubmit());
	await expect(page.getByRole("listitem").filter({ hasText: "Latest category" })).toBeVisible(); await page.waitForTimeout(250);
	await expect(page.getByRole("listitem").filter({ hasText: "Latest category" })).toBeVisible();
	await expect(page.getByRole("listitem").filter({ hasText: "Slow category" })).toHaveCount(0);
});

test("admin views remain keyboard-accessible, responsive, and theme-safe", async ({ page }, testInfo) => {
	await stubApp(page); await openAdmin(page); await page.emulateMedia({ reducedMotion: "reduce" });
	const columns = await page.locator("[data-admin-classification-grid]").evaluate((element) => getComputedStyle(element).gridTemplateColumns.split(" ").length);
	expect(columns).toBe(testInfo.project.name === "mobile-chromium" ? 1 : 2);
	for (const theme of ["light", "dark"] as const) {
		if ((await page.locator("html").getAttribute("data-theme")) !== theme) { const toggle = page.getByLabel("Theme preference"); if (!(await toggle.isVisible())) await page.getByLabel("Open activity sidebar").click(); await toggle.click(); }
		await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
		const axe = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
		expect(axe.violations.filter(({ impact }) => impact === "serious" || impact === "critical")).toEqual([]);
	}
});
