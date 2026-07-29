import { writeFileSync } from "node:fs";
import AxeBuilder from "@axe-core/playwright";
import { expect, test, type APIResponse, type Page } from "@playwright/test";
import {
	AUTH_LOGIN_ENDPOINT,
	buildLoginRequestInit,
	type CuratedImportRequest,
	type ExternalSearchEnvelope
} from "../src/lib/api/generated";

// Implements DESIGN-014 LogAggregator deployed SW-REQ-084 action production.
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const enabled = process.env.MEALSWAPP_TASK285_DEPLOYMENT_ACK === "deployed-test-with-centralized-log-sink";
const startedAt = new Date().toISOString();
const events: ActionEvent[] = [];
test.skip(!enabled, "Run scripts/run-task285-acceptance.py against an acknowledged deployed sink.");

interface ActionEvent {
	category: string;
	requestId: string;
	action: string;
	resource: string;
	outcome: string;
}

interface Envelope<T = unknown> {
	requestId?: string;
	data?: T;
	error?: { code?: string };
}

function fixture(name: string): string {
	const value = process.env[name];
	if (!value) throw new Error(`Task 285 deployed fixture ${name} is required`);
	return value;
}

async function envelope<T = unknown>(response: APIResponse): Promise<Envelope<T>> {
	const body = await response.json() as Envelope<T>;
	expect(body.requestId).toMatch(UUID);
	return body;
}

function record(category: string, requestId: string, action: string, resource: string, outcome: string): void {
	if (events.some((item) => item.category === category || item.requestId === requestId)) {
		throw new Error("Task 285 action correlation is not unique");
	}
	events.push({ category, requestId, action, resource, outcome });
}

async function csrf(page: Page): Promise<string> {
	const response = await page.request.get("/api/v1/auth/csrf-token");
	expect(response.status()).toBe(200);
	const body = await envelope<{ csrfToken?: string }>(response);
	expect(body.data?.csrfToken).toBeTruthy();
	return body.data!.csrfToken!;
}

async function login(page: Page, password: string): Promise<APIResponse> {
	const init = buildLoginRequestInit({
		email: fixture("MEALSWAPP_TASK285_ADMIN_EMAIL"),
		password
	});
	return page.request.fetch(AUTH_LOGIN_ENDPOINT, {
		method: init.method,
		headers: init.headers as Record<string, string>,
		data: JSON.parse(String(init.body))
	});
}

function manualItem(name: string, extras: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		name,
		physicalState: "solid",
		macrosPer100: { protein: 8, carbohydrates: 14, fat: 3 },
		micros: {},
		foodCategoryIds: [],
		culinaryRoleIds: [],
		allergenKeys: [],
		...extras
	};
}

test.beforeAll(() => {
	fixture("MEALSWAPP_TASK285_ACTION_FILE");
	fixture("MEALSWAPP_TASK285_ACTION_PROVENANCE");
	fixture("MEALSWAPP_TASK285_MARKER");
	fixture("MEALSWAPP_TASK285_ADMIN_EMAIL");
	fixture("MEALSWAPP_TASK285_ADMIN_PASSWORD");
	fixture("MEALSWAPP_TASK285_EXTERNAL_QUERY");
	fixture("MEALSWAPP_TASK285_DEPENDENCY_QUERY");
	fixture("MEALSWAPP_TASK285_USER_LOOKUP");
});

test.afterAll(() => {
	if (events.length !== 11) throw new Error("Task 285 did not produce every required action");
	writeFileSync(
		fixture("MEALSWAPP_TASK285_ACTION_FILE"),
		`${JSON.stringify({
			schema: "mealswapp.task285-actions.v1",
			provenance: fixture("MEALSWAPP_TASK285_ACTION_PROVENANCE"),
			startedAt,
			finishedAt: new Date().toISOString(),
			events: [...events].sort((left, right) => left.category.localeCompare(right.category))
		}, null, 2)}\n`,
		{ encoding: "utf8", mode: 0o600 }
	);
});

test("successful and failed generated-client authentication reach the production admin UI accessibly", async ({ page }) => {
	const failed = await login(page, `${fixture("MEALSWAPP_TASK285_ADMIN_PASSWORD")}-wrong`);
	expect(failed.status()).toBe(401);
	record("auth_failure", (await envelope(failed)).requestId!, "auth_login", "session", "failed");

	const success = await login(page, fixture("MEALSWAPP_TASK285_ADMIN_PASSWORD"));
	expect(success.status()).toBe(200);
	record("auth_success", (await envelope(success)).requestId!, "auth_login", "session", "succeeded");

	await page.goto("/admin");
	await expect(page.locator("[data-admin-data-management]")).toBeVisible();
	await page.keyboard.press("Tab");
	await expect(page.locator(":focus")).toBeVisible();
	await page.emulateMedia({ colorScheme: "dark", reducedMotion: "reduce" });
	const accessibility = await new AxeBuilder({ page }).include("[data-admin-data-management]").analyze();
	expect(accessibility.violations).toEqual([]);
});

test("manual success, validation failure, and audit failure are distinct and cleaned up", async ({ page }) => {
	await login(page, fixture("MEALSWAPP_TASK285_ADMIN_PASSWORD"));
	const token = await csrf(page);
	const marker = fixture("MEALSWAPP_TASK285_MARKER");
	let createdId = "";
	try {
		const created = await page.request.post("/api/v1/admin/items", {
			headers: { "X-CSRF-Token": token, "Idempotency-Key": crypto.randomUUID() },
			data: manualItem(`${marker}-manual`)
		});
		expect(created.status()).toBe(201);
		const createdBody = await envelope<{ id?: string }>(created);
		expect(createdBody.data?.id).toMatch(UUID);
		createdId = createdBody.data!.id!;
		record("manual_success", createdBody.requestId!, "manual_create", "global_item", "succeeded");

		const failed = await page.request.post("/api/v1/admin/items", {
			headers: { "X-CSRF-Token": token, "Idempotency-Key": crypto.randomUUID() },
			data: manualItem(`${marker}-manual-failed`, { macrosPer100: { protein: -1, carbohydrates: 0, fat: 0 } })
		});
		expect(failed.status()).toBe(400);
		record("manual_failure", (await envelope(failed)).requestId!, "manual_create", "global_item", "validation_failed");

		const validation = await page.request.post("/api/v1/admin/items", {
			headers: { "X-CSRF-Token": token, "Idempotency-Key": crypto.randomUUID() },
			data: manualItem(`${marker}-validation`, { foodCategoryIds: ["00000000-0000-0000-0000-000000000000"] })
		});
		expect(validation.status()).toBe(400);
		record("validation", (await envelope(validation)).requestId!, "manual_create", "global_item", "validation_failed");

		const audit = await page.request.post("/api/v1/admin/items", {
			headers: {
				"X-CSRF-Token": token,
				"Idempotency-Key": crypto.randomUUID(),
				"X-Mealswapp-Acceptance-Scenario": "audit-failure"
			},
			data: manualItem(`${marker}-audit-failure`)
		});
		expect(audit.status()).toBe(500);
		const auditBody = await envelope(audit);
		expect(auditBody.error?.code).toBe("audit_write_failed");
		record("audit_failure", auditBody.requestId!, "manual_create", "global_item", "audit_failed");
	} finally {
		if (createdId) {
			const cleanup = await page.request.delete(`/api/v1/admin/items/${createdId}`, {
				headers: { "X-CSRF-Token": token }
			});
			expect(cleanup.status()).toBe(204);
		}
	}
});

test("classification mutation and restricted user administration use production boundaries", async ({ page }) => {
	await login(page, fixture("MEALSWAPP_TASK285_ADMIN_PASSWORD"));
	const token = await csrf(page);
	let classificationId = "";
	try {
		const classification = await page.request.post("/api/v1/admin/classifications/food_category", {
			headers: { "X-CSRF-Token": token },
			data: { name: `${fixture("MEALSWAPP_TASK285_MARKER")}-classification` }
		});
		expect(classification.status()).toBe(201);
		const classificationBody = await envelope<{ classification?: { id?: string } }>(classification);
		expect(classificationBody.data?.classification?.id).toMatch(UUID);
		classificationId = classificationBody.data!.classification!.id!;
		record("classification", classificationBody.requestId!, "classification_create", "classification", "succeeded");

		const lookup = await page.request.get(
			`/api/v1/admin/users?email=${encodeURIComponent(fixture("MEALSWAPP_TASK285_USER_LOOKUP"))}`
		);
		expect(lookup.status()).toBe(200);
		record("user_administration", (await envelope(lookup)).requestId!, "user_lookup", "user_administration", "succeeded");
	} finally {
		if (classificationId) {
			const cleanup = await page.request.delete(
				`/api/v1/admin/classifications/${classificationId}`,
				{ headers: { "X-CSRF-Token": token } }
			);
			expect(cleanup.status()).toBe(204);
		}
	}
});

test("external search, curated import, and dependency failure are independently correlated", async ({ page }) => {
	await login(page, fixture("MEALSWAPP_TASK285_ADMIN_PASSWORD"));
	const token = await csrf(page);
	const search = await page.request.get(
		`/api/v1/admin/external-search?query=${encodeURIComponent(fixture("MEALSWAPP_TASK285_EXTERNAL_QUERY"))}&provider=all&page=1`
	);
	expect(search.status()).toBe(200);
	const searchBody = await envelope<ExternalSearchEnvelope["data"]>(search);
	const candidate = searchBody.data?.candidates?.find((item) => item.physicalState === "solid");
	expect(candidate).toBeTruthy();
	record("external_search", searchBody.requestId!, "external_search", "external_catalog", "succeeded");

	const importBody: CuratedImportRequest = {
		sourceProvider: candidate!.provider,
		externalId: candidate!.externalId,
		name: `${fixture("MEALSWAPP_TASK285_MARKER")}-import`,
		physicalState: "solid",
		macrosPer100: candidate!.macrosPer100,
		micros: candidate!.micronutrients,
		foodCategoryIds: [],
		culinaryRoleIds: []
	};
	let importedItemId = "";
	try {
		const imported = await page.request.post("/api/v1/admin/imports", {
			headers: { "X-CSRF-Token": token, "Idempotency-Key": crypto.randomUUID() },
			data: importBody
		});
		expect(imported.status()).toBe(201);
		const importedBody = await envelope<{ foodItemId?: string }>(imported);
		expect(importedBody.data?.foodItemId).toMatch(UUID);
		importedItemId = importedBody.data!.foodItemId!;
		record("external_import", importedBody.requestId!, "curated_import", "global_item", "succeeded");

		const dependency = await page.request.get(
			`/api/v1/admin/external-search?query=${encodeURIComponent(fixture("MEALSWAPP_TASK285_DEPENDENCY_QUERY"))}&provider=all&page=1`,
			{ headers: { "X-Mealswapp-Acceptance-Scenario": "dependency-failure" } }
		);
		expect(dependency.status()).toBe(503);
		record("dependency", (await envelope(dependency)).requestId!, "external_search", "external_catalog", "dependency_failed");
	} finally {
		if (importedItemId) {
			const cleanup = await page.request.delete(`/api/v1/admin/items/${importedItemId}`, {
				headers: { "X-CSRF-Token": token }
			});
			expect(cleanup.status()).toBe(204);
		}
	}
});
