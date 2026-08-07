import { mkdirSync } from "node:fs";
import { join } from "node:path";
import { expect, type APIResponse, type Page, type TestInfo } from "@playwright/test";

// Implements DESIGN-009 AdminController isolated SW-REQ-054 acceptance support.

/** Minimal safe API envelope projection used by authorization assertions. */
export interface SafeEnvelope {
	requestId?: string;
	data?: unknown;
	error?: { code?: string };
}

/** Returns one mandatory isolated fixture value without placing it in evidence. */
export function fixture(name: string): string {
	const value = process.env[name];
	if (!value) throw new Error(`isolated fixture ${name} is required`);
	return value;
}

/** Parses a response and proves its correlation identifier is server-generated. */
export async function safeEnvelope(response: APIResponse): Promise<SafeEnvelope> {
	const body = await response.json() as SafeEnvelope;
	expect(body.requestId).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i);
	expect(body.requestId).not.toBe("spoofed-request-id");
	return body;
}

/** Returns only the server-issued correlation ID for acceptance evidence. */
export async function responseRequestId(response: APIResponse): Promise<string> {
	return (await safeEnvelope(response)).requestId!;
}

/** Signs in through the production authentication UI. */
export async function signIn(page: Page): Promise<void> {
	await openSidebarForControl(page, page.getByRole("button", { name: "Sign in", exact: true }));
	await page.getByRole("button", { name: "Sign in", exact: true }).click();
	await page.locator("[data-login-view]").getByLabel("Email").fill(fixture("MEALSWAPP_E2E_EMAIL"));
	await page.locator("[data-login-view]").getByLabel("Password").fill(fixture("MEALSWAPP_E2E_PASSWORD"));
	await page.locator("[data-login-view]").getByRole("button", { name: "Sign in" }).click();
	await expect(page.locator("[data-login-view]")).toHaveCount(0);
}

/** Opens the responsive sidebar when a navigation control is not currently visible. */
export async function openSidebarForControl(
	page: Page,
	control: ReturnType<Page["locator"]>
): Promise<void> {
	if (await control.isVisible()) return;
	const trigger = page.getByLabel("Open activity sidebar");
	if (await trigger.isVisible()) await trigger.click();
	await expect(control).toBeVisible();
}

/** Writes a screenshot below the acceptance evidence root and returns its safe relative path. */
export async function screenshot(
	page: Page,
	testInfo: TestInfo,
	name: string,
	selector: string
): Promise<string> {
	const relative = `screenshots/${testInfo.project.name}-${name}.png`;
	const root = fixture("PHASE08_ACCEPTANCE_RESULT_DIR");
	mkdirSync(join(root, "screenshots"), { recursive: true });
	await page.locator(selector).screenshot({ path: join(root, relative) });
	return relative;
}

/** Attaches only Task 280 allow-listed acceptance metadata to a Playwright result. */
export async function recordAcceptance(
	testInfo: TestInfo,
	criterionIds: string[],
	requestIds: string[],
	evidence: Array<{ type: "playwright" | "backend"; path: string }>,
	backendEvidence: string[],
	rootCauseId?: string,
	requiredProjects?: string[]
): Promise<void> {
	await testInfo.attach("phase08-acceptance", {
		body: JSON.stringify({ criterionIds, requestIds, evidence, backendEvidence, rootCauseId, requiredProjects }),
		contentType: "application/json"
	});
}

/** Returns the per-project temp storage-state path used across bootstrap. */
export function staleStatePath(testInfo: TestInfo, purpose: "claim" | "reauth" = "claim"): string {
	return join(fixture("MEALSWAPP_E2E_STALE_STATE_DIR"), `${testInfo.project.name}-${purpose}.json`);
}
