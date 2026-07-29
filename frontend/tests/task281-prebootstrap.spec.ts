import { expect, test } from "@playwright/test";
import {
	fixture,
	recordAcceptance,
	safeEnvelope,
	screenshot,
	signIn,
	staleStatePath
} from "./task281-acceptance-helpers";

// Implements DESIGN-009 AdminController anonymous and ordinary-user SW-REQ-054 acceptance.

const runRealFlow =
	process.env.MEALSWAPP_TASK281_REAL_E2E === "1" &&
	process.env.MEALSWAPP_REAL_STACK_MANAGED === "1" &&
	Boolean(process.env.MEALSWAPP_TASK281_CAPABILITY_FILE) &&
	Boolean(process.env.MEALSWAPP_TASK281_CAPABILITY_NONCE);
test.skip(!runRealFlow, "Run scripts/run-task281-acceptance.py for isolated SW-REQ-054 evidence.");

test("anonymous direct navigation and documented administration APIs fail closed [P08-SWR054-STEP-01] [P08-SWR054-ACCEPT-01]", async ({ page, request }, testInfo) => {
	const read = await request.get("/api/v1/admin/classifications?kind=food_category", {
		headers: { "X-Role": "admin", "X-User-ID": "spoofed", "X-Request-ID": "spoofed-request-id" }
	});
	const readBody = await safeEnvelope(read);
	expect(read.status()).toBe(401);
	expect(readBody.data).toBeUndefined();

	const mutation = await request.post("/api/v1/admin/classifications/food_category", {
		headers: { "X-Role": "admin", "X-User-ID": "spoofed", "X-Request-ID": "spoofed-request-id" },
		data: { name: "must-not-be-created", parentId: null }
	});
	const mutationBody = await safeEnvelope(mutation);
	expect(mutation.status()).toBe(401);
	expect(mutationBody.data).toBeUndefined();

	await page.goto("/admin");
	await expect(page).toHaveURL(/\/$/);
	await expect(page.locator("[data-admin-access-denied]")).toBeVisible();
	await expect(page.locator("[data-sidebar-nav-administration]")).toHaveCount(0);
	await expect(page.locator("[data-administration-panel]")).toHaveCount(0);
	const image = await screenshot(page, testInfo, "anonymous-denied", "[data-admin-access-denied]");
	await recordAcceptance(
		testInfo,
		["P08-SWR054-STEP-01", "P08-SWR054-ACCEPT-01"],
		[readBody.requestId!, mutationBody.requestId!],
		[{ type: "playwright", path: image }],
		["http_status=401", "owner_state=restricted", "request_correlation=server_derived"]
	);
});

test("ordinary user navigation, reads, mutations, and spoofed identity remain forbidden [P08-SWR054-STEP-02] [P08-SWR054-STEP-03] [P08-SWR054-ACCEPT-02] [P08-SWR054-ACCEPT-04] [P08-SWR054-ACCEPT-05]", async ({ page, request }, testInfo) => {
	const login = await request.post("/api/v1/auth/login", {
		data: { email: fixture("MEALSWAPP_E2E_EMAIL"), password: fixture("MEALSWAPP_E2E_PASSWORD") }
	});
	expect(login.status()).toBe(200);
	const loginBody = await safeEnvelope(login);
	const csrfResponse = await request.get("/api/v1/auth/csrf-token");
	const csrfBody = await safeEnvelope(csrfResponse) as SafeEnvelopeWithCsrf;
	expect(csrfResponse.status()).toBe(200);

	const read = await request.get("/api/v1/admin/classifications?kind=food_category", {
		headers: { "X-Role": "admin", "X-User-ID": "spoofed", "X-Request-ID": "spoofed-request-id" }
	});
	const readBody = await safeEnvelope(read);
	expect(read.status()).toBe(403);
	expect(readBody.data).toBeUndefined();
	const mutation = await request.post("/api/v1/admin/classifications/food_category", {
		headers: {
			"X-CSRF-Token": csrfBody.data.csrfToken,
			"X-Role": "admin",
			"X-User-ID": "spoofed",
			"X-Request-ID": "spoofed-request-id"
		},
		data: { name: "must-not-be-created", parentId: null, role: "admin", userId: "spoofed" }
	});
	const mutationBody = await safeEnvelope(mutation);
	expect(mutation.status()).toBe(403);
	expect(mutationBody.data).toBeUndefined();
	await request.storageState({ path: staleStatePath(testInfo, "reauth") });

	await page.goto("/");
	await signIn(page);
	await expect(page.locator("[data-sidebar-nav-administration]")).toHaveCount(0);
	await page.goto("/admin");
	await expect(page).toHaveURL(/\/$/);
	await expect(page.locator("[data-admin-access-denied]")).toBeVisible();
	await expect(page.locator("[data-administration-panel]")).toHaveCount(0);
	await page.context().storageState({ path: staleStatePath(testInfo, "claim") });
	const image = await screenshot(page, testInfo, "ordinary-denied", "[data-admin-access-denied]");
	await recordAcceptance(
		testInfo,
		[
			"P08-SWR054-STEP-02",
			"P08-SWR054-STEP-03",
			"P08-SWR054-ACCEPT-02",
			"P08-SWR054-ACCEPT-04",
			"P08-SWR054-ACCEPT-05"
		],
		[loginBody.requestId!, csrfBody.requestId!, readBody.requestId!, mutationBody.requestId!],
		[{ type: "playwright", path: image }],
		["http_status=403", "mutation_count=0", "owner_state=restricted", "request_correlation=server_derived"]
	);
});

interface SafeEnvelopeWithCsrf {
	requestId: string;
	data: { csrfToken: string };
}
