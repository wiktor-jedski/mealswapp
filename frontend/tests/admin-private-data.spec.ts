import { expect, test, type Page, type Route } from "@playwright/test";

// Implements DESIGN-008 DataExporter authoritative Account Export refresh verification in DESIGN-009 UserAdminPanel.

const firstItemId = "00000000-0000-4000-8000-000000000267";
const secondItemId = "00000000-0000-4000-8000-000000000268";
const ok = (data: unknown) => ({ status: "ok", requestId: "task-267", data });
const item = (id: string, name: string) => ({ id, name, physicalState: "solid", prepTimeMinutes: 0, macrosPer100: { protein: 1, carbohydrates: 1, fat: 1 }, micros: {}, foodCategories: [], culinaryRoles: [] });
const bundle = (customItems: unknown[]) => ({ user: { userId: firstItemId, email: "admin@example.test", role: "admin", displayName: "Admin", unitSystem: "metric", themePreference: "system" }, consent: [], savedItems: [], savedDiets: [], history: [], customItems });

async function json(route: Route, status: number, body?: unknown): Promise<void> {
	await route.fulfill(body === undefined ? { status } : { status, contentType: "application/json", body: JSON.stringify(body) });
}

async function stubAdminShell(page: Page, exportHandler: (route: Route) => Promise<void>, deletedIds: string[] = [], deleteHandler?: (route: Route) => Promise<void>): Promise<void> {
	await page.route("**/api/v1/**", async (route) => {
		const request = route.request();
		const url = new URL(request.url());
		if (url.pathname === "/api/v1/account/export") return exportHandler(route);
		if (url.pathname.startsWith("/api/v1/custom-items/") && request.method() === "DELETE") {
			if (deleteHandler) return deleteHandler(route);
			deletedIds.push(url.pathname.split("/").at(-1)!);
			return json(route, 204);
		}
		if (url.pathname === "/api/v1/profile") return json(route, 200, ok({ userId: "admin-267", displayName: "Admin", unitSystem: "metric", themePreference: "system", requiresUnitRecalculation: false }));
		if (url.pathname === "/api/v1/auth/refresh") return json(route, 200, ok({ userId: "admin-267", role: "admin", hasVerifiedLoginMethod: true, accessExpiresAt: "2026-07-24T13:00:00Z", refreshExpiresAt: "2026-07-31T13:00:00Z" }));
		if (url.pathname === "/api/v1/billing/entitlement") return json(route, 200, ok({ userId: "admin-267", tier: "paid", status: "active", allowedModes: ["catalog", "substitution", "daily_diet", "daily_diet_alternative"], searchLimitPer24h: null, usageUsed: 0, usageRemaining: null, usageWindowStartedAt: "2026-07-24T00:00:00Z", trialExpiresAt: null, billingRecoveryState: "none" }));
		if (url.pathname === "/api/v1/auth/csrf-token") return json(route, 200, ok({ csrfToken: "csrf-task-267" }));
		if (url.pathname === "/api/v1/search-history") return json(route, 200, ok({ history: [] }));
		if (url.pathname === "/api/v1/saved-items") return json(route, 200, ok({ items: [] }));
		if (url.pathname === "/api/v1/search/autocomplete") return json(route, 200, ok({ items: [] }));
		if (url.pathname === "/api/v1/admin/classifications") return json(route, 200, ok({ classifications: [] }));
		return json(route, 404, { status: "error", requestId: "task-267-unhandled", error: { category: "validation", code: "not_found", message: "Not found", retryable: false } });
	});
}

async function openPrivateData(page: Page) {
	await page.goto("/admin");
	const privateData = page.locator("[data-admin-private-data]");
	await expect(privateData).toBeVisible();
	return privateData;
}

// Verifies IT-ARCH-009-011, ARCH-009, DESIGN-008 DataExporter,
// DESIGN-009 UserAdminPanel, and SW-REQ-054/SW-REQ-072.
test("failed refresh clears loaded private objects and controls until an owner-free retry succeeds", async ({ page }) => {
	let attempt = 0;
	let releaseFailure!: () => void;
	const failureReleased = new Promise<void>((resolve) => { releaseFailure = resolve; });
	await stubAdminShell(page, async (route) => {
		attempt++;
		if (attempt === 1) return json(route, 200, bundle([item(firstItemId, "Previously loaded private item")]));
		if (attempt === 2) {
			await failureReleased;
			return json(route, 200, bundle([{ ...item(secondItemId, "Foreign private secret"), ownerId: "00000000-0000-4000-8000-000000000999" }]));
		}
		return json(route, 200, bundle([item(secondItemId, "Current private item")]));
	});

	const privateData = await openPrivateData(page);
	await expect(privateData.getByText("Previously loaded private item")).toBeVisible();
	await privateData.getByRole("button", { name: "Delete private item" }).click();
	await expect(privateData.getByRole("button", { name: "Permanently delete private item" })).toBeVisible();
	await privateData.getByRole("button", { name: "Refresh export" }).click();
	await expect(privateData.getByText("Loading authoritative account export…")).toBeVisible();
	await expect(privateData.getByText("Previously loaded private item")).toHaveCount(0);
	await expect(privateData.getByRole("button", { name: "Delete private item" })).toHaveCount(0);
	await expect(privateData.getByRole("button", { name: "Permanently delete private item" })).toHaveCount(0);

	releaseFailure();
	await expect(privateData.getByRole("alert")).toContainText("Account data could not be refreshed");
	await expect(privateData).not.toContainText("Foreign private secret");
	await expect(privateData).not.toContainText("ownerId");
	await expect(privateData.getByRole("button", { name: "Delete private item" })).toHaveCount(0);

	await privateData.getByRole("button", { name: "Refresh export" }).click();
	await expect(privateData.getByText("Current private item")).toBeVisible();
	await expect(privateData.getByRole("button", { name: "Delete private item" })).toHaveCount(1);
	await expect(privateData.getByRole("alert")).toHaveCount(0);
	await expect(privateData).not.toContainText("Previously loaded private item");
});

// Verifies IT-ARCH-009-011, ARCH-009, DESIGN-008 DataExporter,
// DESIGN-009 UserAdminPanel, and SW-REQ-054/SW-REQ-072.
test("accepted deletion reports verification-required failure and claims success only after a later complete cycle", async ({ page }) => {
	let attempt = 0;
	let releaseFailedRefresh!: () => void;
	const failedRefreshReleased = new Promise<void>((resolve) => { releaseFailedRefresh = resolve; });
	let releasePostSuccessFailure!: () => void;
	const postSuccessFailureReleased = new Promise<void>((resolve) => { releasePostSuccessFailure = resolve; });
	const deletedIds: string[] = [];
	await stubAdminShell(page, async (route) => {
		attempt++;
		if (attempt === 1) return json(route, 200, bundle([item(firstItemId, "Delete then verify"), item(secondItemId, "Delete successfully")]));
		if (attempt === 2) {
			await failedRefreshReleased;
			return json(route, 503, { status: "error" });
		}
		if (attempt === 3) return json(route, 200, bundle([item(secondItemId, "Delete successfully")]));
		if (attempt === 4) return json(route, 200, bundle([]));
		await postSuccessFailureReleased;
		return json(route, 503, { status: "error" });
	}, deletedIds);

	const privateData = await openPrivateData(page);
	await privateData.getByRole("listitem").filter({ hasText: "Delete then verify" }).getByRole("button", { name: "Delete private item" }).click();
	await privateData.getByRole("button", { name: "Permanently delete private item" }).click();
	await expect(privateData.getByText("Loading authoritative account export…")).toBeVisible();
	await expect(privateData.getByRole("button", { name: "Delete private item" })).toHaveCount(0);
	await expect(privateData).not.toContainText("Private item deleted and authoritative export refreshed.");

	releaseFailedRefresh();
	await expect(privateData.getByRole("alert")).toContainText("The private item was deleted, but current account data could not be verified. Refresh the export before continuing.");
	await expect(privateData).not.toContainText("Private item deleted and authoritative export refreshed.");
	await expect(privateData.getByRole("button", { name: "Delete private item" })).toHaveCount(0);
	expect(deletedIds).toEqual([firstItemId]);

	await privateData.getByRole("button", { name: "Refresh export" }).click();
	await expect(privateData.getByText("Delete successfully")).toBeVisible();
	await expect(privateData.getByRole("alert")).toHaveCount(0);
	await expect(privateData).not.toContainText("Private item deleted and authoritative export refreshed.");

	await privateData.getByRole("button", { name: "Delete private item" }).click();
	await privateData.getByRole("button", { name: "Permanently delete private item" }).click();
	await expect(privateData.getByText("Private item deleted and authoritative export refreshed.")).toBeVisible();
	await expect(privateData.locator("[data-admin-private-data-empty]")).toBeVisible();
	expect(deletedIds).toEqual([firstItemId, secondItemId]);

	await privateData.getByRole("button", { name: "Refresh export" }).click();
	await expect(privateData.getByText("Loading authoritative account export…")).toBeVisible();
	await expect(privateData).not.toContainText("Private item deleted and authoritative export refreshed.");
	releasePostSuccessFailure();
	await expect(privateData.getByRole("alert")).toContainText("Account data could not be refreshed");
	await expect(privateData).not.toContainText("Private item deleted and authoritative export refreshed.");
});

// Implements DESIGN-008 AccountDeleter permanent-deletion safety and saved-diet conflict recovery.
test("requires irreversible confirmation and retains actionable saved-diet references after conflict", async ({ page }) => {
	const dietId = "00000000-0000-4000-8000-000000000269";
	await stubAdminShell(
		page,
		(route) => json(route, 200, bundle([item(firstItemId, "Referenced private item")])),
		[],
		(route) => json(route, 409, {
			status: "error",
			requestId: "task-298-conflict",
			error: {
				category: "conflict",
				code: "custom_item_in_use",
				message: "private item is still referenced",
				retryable: false,
				data: { affectedDiets: [{ id: dietId, name: "Weeknight private meals" }] }
			}
		})
	);

	const privateData = await openPrivateData(page);
	await privateData.getByRole("button", { name: "Delete private item" }).click();
	const dialog = privateData.getByRole("alertdialog");
	await expect(dialog).toContainText("permanently and irreversibly");
	await expect(dialog).toContainText("There is no Trash or Restore workflow");
	await expect(dialog.getByRole("button", { name: "Permanently delete private item" })).toBeVisible();
	await dialog.getByRole("button", { name: "Cancel" }).click();
	await expect(privateData.getByText("Referenced private item")).toBeVisible();

	await privateData.getByRole("button", { name: "Delete private item" }).click();
	await privateData.getByRole("button", { name: "Permanently delete private item" }).click();
	const alert = privateData.getByRole("alert");
	await expect(alert).toContainText("Remove this item from the listed saved diets before permanent deletion.");
	await expect(alert).toContainText("Open Daily Diets, remove this item from each listed diet, save the diet, then retry permanent deletion.");
	await expect(alert.getByRole("list", { name: "Saved diets blocking permanent deletion" })).toContainText("Weeknight private meals");
	await expect(privateData.getByText("Referenced private item")).toHaveCount(0);
});
