import { mkdir } from "node:fs/promises";
import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Locator, type Page, type Route } from "@playwright/test";

// Implements DESIGN-009 UserAdminPanel Phase 08.01 functional, visual, and accessibility regression gate.

const privateItemId = "00000000-0000-4000-8000-000000000272";
const screenshotDirectory = "/tmp/mealswapp-task-272";
const ok = (data: unknown) => ({ status: "ok", requestId: "task-272", data });

async function json(route: Route, status: number, body?: unknown): Promise<void> {
	await route.fulfill(body === undefined ? { status } : { status, contentType: "application/json", body: JSON.stringify(body) });
}

async function stubAdmin(page: Page): Promise<void> {
	await page.route("**/api/v1/**", async (route) => {
		const url = new URL(route.request().url());
		if (url.pathname === "/api/v1/profile") return json(route, 200, ok({ userId: "admin-272", displayName: "Regression Admin", unitSystem: "metric", themePreference: "system", requiresUnitRecalculation: false }));
		if (url.pathname === "/api/v1/auth/refresh") return json(route, 200, ok({ userId: "admin-272", role: "admin", hasVerifiedLoginMethod: true, accessExpiresAt: "2026-07-24T13:00:00Z", refreshExpiresAt: "2026-07-31T13:00:00Z" }));
		if (url.pathname === "/api/v1/billing/entitlement") return json(route, 200, ok({ userId: "admin-272", tier: "paid", status: "active", allowedModes: ["catalog", "substitution", "daily_diet", "daily_diet_alternative"], searchLimitPer24h: null, usageUsed: 0, usageRemaining: null, usageWindowStartedAt: "2026-07-24T00:00:00Z", trialExpiresAt: null, billingRecoveryState: "none" }));
		if (url.pathname === "/api/v1/auth/csrf-token") return json(route, 200, ok({ csrfToken: "csrf-task-272" }));
		if (url.pathname === "/api/v1/search-history") return json(route, 200, ok({ history: [] }));
		if (url.pathname === "/api/v1/saved-items") return json(route, 200, ok({ items: [] }));
		if (url.pathname === "/api/v1/search/autocomplete") return json(route, 200, ok({ items: [] }));
		if (url.pathname === "/api/v1/account/export") return json(route, 200, { user: {}, consent: [], savedItems: [], history: [], customItems: [{ id: privateItemId, name: "Current private regression item" }] });
		if (url.pathname === "/api/v1/admin/classifications") return json(route, 200, ok({ classifications: [] }));
		if (url.pathname === "/api/v1/admin/external-search") return json(route, 200, ok({ candidates: [], warnings: [], page: 1 }));
		return json(route, 404, { status: "error", requestId: "task-272-unhandled", error: { category: "validation", code: "not_found", message: "Not found", retryable: false } });
	});
}

async function tabTo(page: Page, target: Locator): Promise<void> {
	for (let attempt = 0; attempt < 80; attempt++) {
		if (await target.evaluate((element) => element === document.activeElement)) return;
		await page.keyboard.press("Tab");
	}
	throw new Error("Keyboard focus did not reach the requested administration control.");
}

async function setTheme(page: Page, theme: "light" | "dark"): Promise<void> {
	if ((await page.locator("html").getAttribute("data-theme")) === theme) return;
	const openSidebar = page.getByLabel("Open activity sidebar");
	let opened = false;
	if (await openSidebar.isVisible()) {
		await openSidebar.focus();
		await page.keyboard.press("Enter");
		opened = true;
	}
	const toggle = page.getByLabel("Theme preference");
	await toggle.focus();
	await page.keyboard.press("Enter");
	await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
	if (opened) {
		const closeSidebar = page.getByLabel("Close activity sidebar");
		await closeSidebar.focus();
		await page.keyboard.press("Enter");
	}
}

// Verifies IT-ARCH-009-010, IT-ARCH-009-011, and IT-ARCH-012-004,
// ARCH-009, ARCH-012, DESIGN-009 UserAdminPanel/ExternalSearchProxy,
// DESIGN-012 RateLimitHandler, and SW-REQ-054/SW-REQ-055.
test("administration regressions remain keyboard-safe, responsive, accessible, themed, and motion-reduced", async ({ page }, testInfo) => {
	await page.emulateMedia({ reducedMotion: "reduce" });
	await stubAdmin(page);
	await page.goto("/admin");

	const panel = page.locator("[data-administration-panel]");
	const search = panel.getByLabel("External food search");
	await expect(panel).toBeVisible();
	await expect(panel.getByText("Current private regression item")).toBeVisible();

	await page.locator("body").focus();
	await tabTo(page, search);
	await expect(search).toBeFocused();
	await page.keyboard.type("keyboard apples");
	await page.keyboard.press("Enter");
	await expect(panel.getByText("No external candidates matched this search.")).toBeVisible();

	const renderedStyles = await panel.evaluate((element) => {
		const heading = element.querySelector("h1");
		const control = element.querySelector("input");
		const button = element.querySelector("button");
		if (!heading || !control || !button) throw new Error("Administration style fixtures are missing.");
		const headingStyle = getComputedStyle(heading);
		const controlStyle = getComputedStyle(control);
		const buttonStyle = getComputedStyle(button);
		return {
			headingWeight: headingStyle.fontWeight,
			controlBackground: controlStyle.backgroundColor,
			controlBorderWidth: controlStyle.borderTopWidth,
			buttonTransitionProperty: buttonStyle.transitionProperty,
			buttonTransitionDuration: buttonStyle.transitionDuration
		};
	});
	expect(renderedStyles.headingWeight).toBe("700");
	expect(renderedStyles.controlBackground).not.toBe("rgba(0, 0, 0, 0)");
	expect(renderedStyles.controlBorderWidth).toBe("1px");
	expect(renderedStyles.buttonTransitionProperty).toBe("none");
	expect(renderedStyles.buttonTransitionDuration).toBe("0s");

	await mkdir(screenshotDirectory, { recursive: true });
	for (const theme of ["light", "dark"] as const) {
		await setTheme(page, theme);
		const axe = await new AxeBuilder({ page }).include("[data-administration-panel]").withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
		expect(axe.violations.filter(({ impact }) => impact === "serious" || impact === "critical")).toEqual([]);
		expect(await panel.evaluate((element) => {
			const controls = [...element.querySelectorAll<HTMLElement>("button, input, select, textarea")].filter((control) => control.getClientRects().length > 0);
			return element.scrollWidth <= element.clientWidth + 1 && controls.every((control) => {
				const rectangle = control.getBoundingClientRect();
				return rectangle.left >= -1 && rectangle.right <= window.innerWidth + 1 && control.scrollWidth <= control.clientWidth + 1;
			});
		})).toBe(true);
		await page.screenshot({ path: `${screenshotDirectory}/task-272-${testInfo.project.name}-${theme}.png`, fullPage: true, animations: "disabled" });
	}
});
