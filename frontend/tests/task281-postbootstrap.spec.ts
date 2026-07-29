import { readFileSync } from "node:fs";
import AxeBuilder from "@axe-core/playwright";
import { expect, request as apiRequest, test } from "@playwright/test";
import {
	fixture,
	openSidebarForControl,
	recordAcceptance,
	safeEnvelope,
	screenshot,
	signIn,
	staleStatePath
} from "./task281-acceptance-helpers";

// Implements DESIGN-009 AdminController malformed, stale, and administrator SW-REQ-054 acceptance.

const runRealFlow =
	process.env.MEALSWAPP_TASK281_REAL_E2E === "1" &&
	process.env.MEALSWAPP_REAL_STACK_MANAGED === "1" &&
	Boolean(process.env.MEALSWAPP_TASK281_CAPABILITY_FILE) &&
	Boolean(process.env.MEALSWAPP_TASK281_CAPABILITY_NONCE);
test.skip(!runRealFlow, "Run scripts/run-task281-acceptance.py for isolated SW-REQ-054 evidence.");

test("malformed administration sessions fail closed [P08-SWR054-ACCEPT-05]", async ({}, testInfo) => {
	const malformed = await apiRequest.newContext({
		baseURL: fixture("MEALSWAPP_REAL_STACK_BASE_URL"),
		extraHTTPHeaders: {
			Cookie: "mealswapp_access=malformed; mealswapp_refresh=malformed",
			"X-Role": "admin",
			"X-User-ID": "spoofed",
			"X-Request-ID": "spoofed-request-id"
		}
	});
	const malformedResponse = await malformed.get("/api/v1/admin/classifications?kind=food_category");
	const malformedBody = await safeEnvelope(malformedResponse);
	expect(malformedResponse.status()).toBe(401);
	expect(malformedBody.data).toBeUndefined();
	await malformed.dispose();
	await recordAcceptance(
		testInfo,
		["P08-SWR054-ACCEPT-05"],
		[malformedBody.requestId!],
		[],
		["http_status=401", "mutation_count=0", "owner_state=restricted", "request_correlation=server_derived"]
	);
});

test("bootstrap does not promote stale browser claims before reauthentication [P08-SWR054-STEP-04] [P08-SWR054-ACCEPT-04]", async ({ page }, testInfo) => {
	const state = JSON.parse(readFileSync(staleStatePath(testInfo, "claim"), "utf8")) as {
		cookies: Array<{ name: string; value: string; domain: string; path: string; expires: number; httpOnly: boolean; secure: boolean; sameSite: "Strict" | "Lax" | "None" }>;
	};
	await page.context().addCookies(state.cookies);
	const staleSession = await apiRequest.newContext({
		baseURL: fixture("MEALSWAPP_REAL_STACK_BASE_URL"),
		storageState: staleStatePath(testInfo, "claim")
	});
	const staleRead = await staleSession.get("/api/v1/admin/classifications?kind=food_category");
	const staleReadBody = await safeEnvelope(staleRead);
	expect(staleRead.status()).toBe(403);
	await staleSession.dispose();
	await page.goto("/admin");
	await recordAcceptance(
		testInfo,
		["P08-SWR054-STEP-04", "P08-SWR054-ACCEPT-04"],
		[staleReadBody.requestId!],
		[],
		["http_status=403", "owner_state=stale_claim_promoted", "request_correlation=server_derived"],
		"ROOT-T281-STALE-REFRESH"
	);
	await expect(page).toHaveURL(/\/$/);
	await expect(page.locator("[data-admin-access-denied]")).toBeVisible();
	await expect(page.locator("[data-administration-panel]")).toHaveCount(0);
});

test("verified administrator documented APIs succeed while spoofed identity and undocumented routes are denied [P08-SWR054-STEP-03] [P08-SWR054-ACCEPT-03] [P08-SWR054-ACCEPT-05]", async ({ request }, testInfo) => {
	const login = await request.post("/api/v1/auth/login", {
		headers: { "X-Request-ID": "spoofed-request-id" },
		data: { email: fixture("MEALSWAPP_E2E_EMAIL"), password: fixture("MEALSWAPP_E2E_PASSWORD") }
	});
	const loginBody = await safeEnvelope(login);
	expect(login.status()).toBe(200);
	const read = await request.get("/api/v1/admin/classifications?kind=food_category", {
		headers: { "X-Role": "user", "X-User-ID": "spoofed", "X-Request-ID": "spoofed-request-id" }
	});
	const readBody = await safeEnvelope(read);
	expect(read.status()).toBe(200);
	const csrf = await request.get("/api/v1/auth/csrf-token");
	const csrfBody = await safeEnvelope(csrf) as SafeEnvelopeWithCsrf;
	const spoofedMutation = await request.post("/api/v1/admin/classifications/food_category", {
		headers: {
			"X-CSRF-Token": csrfBody.data.csrfToken,
			"X-Role": "admin",
			"X-User-ID": "spoofed",
			"X-Request-ID": "spoofed-request-id"
		},
		data: { name: "must-not-be-created", parentId: null, role: "admin", userId: "spoofed" }
	});
	const spoofedBody = await safeEnvelope(spoofedMutation);
	expect(spoofedMutation.status()).toBe(400);
	const undocumentedRead = await request.get("/api/v1/admin/undocumented");
	const undocumentedReadBody = await safeEnvelope(undocumentedRead);
	expect(undocumentedRead.status()).toBe(404);
	const undocumentedMutation = await request.post("/api/v1/admin/undocumented", {
		data: { role: "admin", userId: "spoofed" }
	});
	const undocumentedMutationBody = await safeEnvelope(undocumentedMutation);
	expect(undocumentedMutation.status()).toBe(404);
	await recordAcceptance(
		testInfo,
		["P08-SWR054-STEP-03", "P08-SWR054-ACCEPT-03", "P08-SWR054-ACCEPT-05"],
		[
			loginBody.requestId!,
			readBody.requestId!,
			csrfBody.requestId,
			spoofedBody.requestId!,
			undocumentedReadBody.requestId!,
			undocumentedMutationBody.requestId!
		],
		[],
		["http_status=200", "mutation_count=0", "owner_state=administrator", "request_correlation=server_derived"]
	);
});

test("fresh sign-in drives generated-client mutation in the accessible responsive administrator panel [P08-SWR054-STEP-04] [P08-SWR054-ACCEPT-03]", async ({ page }, testInfo) => {
	const reauth = await apiRequest.newContext({
		baseURL: fixture("MEALSWAPP_REAL_STACK_BASE_URL"),
		storageState: staleStatePath(testInfo, "reauth")
	});
	const csrf = await reauth.get("/api/v1/auth/csrf-token");
	const csrfBody = await safeEnvelope(csrf) as SafeEnvelopeWithCsrf;
	const logout = await reauth.post("/api/v1/auth/logout", {
		headers: { "X-CSRF-Token": csrfBody.data.csrfToken }
	});
	expect(logout.status()).toBe(204);
	const logoutRequestId = logout.headers()["x-request-id"];
	expect(logoutRequestId).toMatch(/^[0-9a-f-]{36}$/i);
	await reauth.dispose();
	await page.goto("/");
	await signIn(page);
	const administration = page.locator("[data-sidebar-nav-administration]");
	await openSidebarForControl(page, administration);
	await administration.focus();
	await page.keyboard.press("Enter");
	const grid = page.locator("[data-admin-responsive-grid]");
	await expect(grid).toBeVisible();
	const form = page.getByRole("form", { name: "Classification form" });
	await form.getByLabel("Name").fill(`Acceptance ${testInfo.project.name}`);
	const create = page.waitForResponse((response) =>
		response.url().endsWith("/api/v1/admin/classifications/food_category") &&
		response.request().method() === "POST"
	);
	await form.getByRole("button", { name: "Create", exact: true }).click();
	const created = await create;
	const createdBody = await created.json() as { requestId?: string };
	expect(created.status()).toBe(201);
	expect(createdBody.requestId).toMatch(/^[0-9a-f-]{36}$/i);
	const columns = await grid.evaluate((element) => getComputedStyle(element).gridTemplateColumns.split(" ").length);
	expect(columns).toBe(testInfo.project.name.includes("mobile") ? 1 : 3);
	for (const theme of ["light", "dark"] as const) {
		const openSidebar = page.getByLabel("Open activity sidebar");
		if (await openSidebar.isVisible()) await openSidebar.click();
		if ((await page.locator("html").getAttribute("data-theme")) !== theme) {
			await page.getByLabel("Theme preference").click();
		}
		await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
		const axe = await new AxeBuilder({ page })
			.withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
			.analyze();
		expect(axe.violations.filter((violation) =>
			violation.impact === "serious" || violation.impact === "critical"
		)).toEqual([]);
	}
	const image = await screenshot(page, testInfo, "admin-panel", "[data-administration-panel]");
	await recordAcceptance(
		testInfo,
		["P08-SWR054-STEP-04", "P08-SWR054-ACCEPT-03"],
		[csrfBody.requestId, logoutRequestId, createdBody.requestId!],
		[{ type: "playwright", path: image }],
		["http_status=200", "owner_state=administrator", "request_correlation=server_derived"]
	);
});

interface SafeEnvelopeWithCsrf {
	requestId: string;
	data: { csrfToken: string };
}
