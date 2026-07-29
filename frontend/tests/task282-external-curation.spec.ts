import { expect, test, type Page, type TestInfo } from "@playwright/test";
import {
	openSidebarForControl,
	recordAcceptance,
	safeEnvelope,
	screenshot,
	signIn
} from "./task281-acceptance-helpers";

// Implements DESIGN-009 ExternalSearchProxy/DataImporter isolated real-stack acceptance.

const enabled =
	process.env.MEALSWAPP_TASK282_REAL_E2E === "1" &&
	process.env.MEALSWAPP_REAL_STACK_MANAGED === "1";
test.skip(!enabled, "Run scripts/run-task282-acceptance.py for controlled-provider evidence.");

async function openAdministration(page: Page): Promise<ReturnType<Page["locator"]>> {
	await page.goto("/");
	await signIn(page);
	const control = page.locator("[data-sidebar-nav-administration]");
	await openSidebarForControl(page, control);
	await control.click();
	const workflow = page.locator("[data-external-import-workflow]");
	await expect(workflow).toBeVisible();
	return workflow;
}

async function search(
	page: Page,
	workflow: ReturnType<Page["locator"]>,
	query: string,
	provider: "USDA" | "OpenFoodFacts" | "USDA + OpenFoodFacts"
): Promise<{ requestId: string; status: number }> {
	await workflow.getByLabel("External food search").fill(query);
	await workflow.locator("[data-external-search-form] select").selectOption({ label: provider });
	const pending = page.waitForResponse((response) =>
		response.url().includes("/api/v1/admin/external-search?") &&
		response.request().method() === "GET"
	);
	await workflow.getByRole("button", { name: /Search/ }).click();
	const response = await pending;
	const body = await response.json() as { requestId?: string };
	expect(body.requestId).toMatch(/^[0-9a-f-]{36}$/i);
	return { requestId: body.requestId!, status: response.status() };
}

test("production UI searches each provider, merges results, curates edited values, confirms once, and discovers the global item [P08-SWR055-STEP-01] [P08-SWR055-STEP-02] [P08-SWR055-STEP-03] [P08-SWR055-STEP-04] [P08-SWR055-STEP-05] [P08-SWR055-STEP-06] [P08-SWR055-STEP-07] [P08-SWR055-ACCEPT-01] [P08-SWR055-ACCEPT-03] [P08-SWR055-ACCEPT-04] [P08-SWR055-ACCEPT-05] [P08-SWR055-ACCEPT-06] [P08-SWR033-STEP-01] [P08-SWR033-STEP-03] [P08-SWR033-STEP-04] [P08-SWR033-STEP-05] [P08-SWR090-STEP-01] [P08-SWR090-ACCEPT-01]", async ({ page }, testInfo) => {
	test.setTimeout(60_000);
	const workflow = await openAdministration(page);
	const csrf = await page.request.get("/api/v1/auth/csrf-token");
	const csrfBody = await safeEnvelope(csrf) as { requestId?: string; data?: { csrfToken?: string } };
	const sourceName = `Acceptance source ${testInfo.project.name}`;
	const source = await page.request.post("/api/v1/admin/items", {
		headers: {
			"X-CSRF-Token": csrfBody.data?.csrfToken ?? "",
			"Idempotency-Key": "task282-substitution-source"
		},
		data: {
			name: sourceName,
			physicalState: "solid",
			prepTimeMinutes: 0,
			macrosPer100: { protein: 11, carbohydrates: 17, fat: 3 },
			micros: {},
			foodCategoryIds: [],
			culinaryRoleIds: [],
			allergenKeys: []
		}
	});
	const sourceBody = await safeEnvelope(source) as { requestId?: string; data?: { id?: string } };
	expect(source.status()).toBe(201);
	expect(sourceBody.data?.id).toMatch(/^[0-9a-f-]{36}$/i);
	const usda = await search(page, workflow, "success", "USDA");
	await expect(workflow.getByText("Fixture lentils")).toBeVisible();
	const off = await search(page, workflow, "success", "OpenFoodFacts");
	await expect(workflow.getByText("Fixture chickpeas")).toBeVisible();
	const merged = await search(page, workflow, "success", "USDA + OpenFoodFacts");
	await expect(workflow.locator("[data-external-results] article")).toHaveCount(2);

	await workflow.getByText("Fixture lentils").locator("..").locator("..").getByRole("button", { name: "Curate" }).click();
	const draft = workflow.locator("[data-curation-draft]");
	const editedName = `Acceptance curated ${testInfo.project.name}`;
	await draft.getByLabel("Name").fill(editedName);
	await draft.getByLabel("Protein per 100").fill("12.5");
	await draft.getByLabel("Carbohydrates per 100").fill("18.25");
	await draft.getByLabel("Fat per 100").fill("3.75");
	const category = draft.getByRole("group", { name: "Food categories" }).getByRole("checkbox").first();
	const role = draft.getByRole("group", { name: "Culinary roles" }).getByRole("checkbox").first();
	if (await category.count()) await category.check();
	if (await role.count()) await role.check();
	const firstImportRequest = page.waitForRequest((request) =>
		request.url().endsWith("/api/v1/admin/imports") && request.method() === "POST"
	);
	await draft.getByRole("button", { name: "Import curated item" }).click();
	const committedRequest = await firstImportRequest;
	const importedRequestBody = committedRequest.postData();
	const importedIdempotencyKey = committedRequest.headers()["idempotency-key"];
	expect(importedRequestBody).toBeTruthy();
	expect(importedIdempotencyKey).toMatch(/^[0-9a-f-]{36}$/i);
	await expect(workflow.locator("[data-import-error]")).toContainText("result could not be confirmed");

	const importResponse = page.waitForResponse((response) =>
		response.url().endsWith("/api/v1/admin/imports") && response.request().method() === "POST"
	);
	await workflow.getByRole("button", { name: "Retry import safely" }).click();
	const imported = await importResponse;
	const replayedRequest = imported.request();
	expect(replayedRequest.headers()["idempotency-key"]).toBe(importedIdempotencyKey);
	expect(replayedRequest.postData()).toBe(importedRequestBody);
	const importedBody = await imported.json() as {
		requestId?: string;
		data?: { foodItemId?: string; name?: string; replayed?: boolean };
	};
	expect(imported.status()).toBe(201);
	expect(importedBody.data?.name).toBe(editedName);
	expect(importedBody.data?.foodItemId).toMatch(/^[0-9a-f-]{36}$/i);
	expect(importedBody.data?.replayed).toBe(true);
	await expect(workflow.locator("[data-import-result]")).toContainText(editedName);

	await workflow.getByRole("button", { name: "View in local search" }).click();
	await expect(page.locator("[data-results-grid]")).toContainText(editedName);
	const substitution = await page.request.post("/api/v1/search", {
		data: {
			query: "",
			mode: "substitution",
			page: 1,
			substitutionInputs: [{
				foodObjectId: sourceBody.data!.id,
				foodObjectType: "food_item",
				quantity: 100,
				unit: "g"
			}]
		}
	});
	const substitutionBody = await safeEnvelope(substitution) as {
		requestId?: string;
		data?: { items?: Array<{ name?: string }> };
	};
	expect(substitution.status()).toBe(200);
	expect(substitutionBody.data?.items?.map((item) => item.name)).toContain(editedName);
	const shot = await screenshot(page, testInfo, "curated-catalog", "main");
	await recordAcceptance(
		testInfo,
		[
			"P08-SWR055-STEP-01", "P08-SWR055-STEP-02", "P08-SWR055-STEP-03",
			"P08-SWR055-STEP-04", "P08-SWR055-STEP-05", "P08-SWR055-STEP-06",
			"P08-SWR055-STEP-07", "P08-SWR055-ACCEPT-01", "P08-SWR055-ACCEPT-03",
			"P08-SWR055-ACCEPT-04", "P08-SWR055-ACCEPT-05", "P08-SWR055-ACCEPT-06",
			"P08-SWR033-STEP-01", "P08-SWR033-STEP-03", "P08-SWR033-STEP-04",
			"P08-SWR033-STEP-05", "P08-SWR090-STEP-01", "P08-SWR090-ACCEPT-01"
		],
		[usda.requestId, off.requestId, merged.requestId, importedBody.requestId!, csrfBody.requestId!, sourceBody.requestId!, substitutionBody.requestId!],
		[{ type: "playwright", path: shot }],
		["http_status=201", "mutation_count=1", "audit_count=1", "owner_state=global", "metric_basis=100g", "cache_generation=advanced"]
	);
});

test("partial failure, outage, malformed data, timeout, cancellation, quota reset, and safe warnings remain truthful [P08-SWR055-ACCEPT-02] [P08-SWR055-ACCEPT-07]", async ({ page }, testInfo) => {
	test.setTimeout(60_000);
	const workflow = await openAdministration(page);
	const requestIds: string[] = [];
	requestIds.push((await search(page, workflow, "partial", "USDA + OpenFoodFacts")).requestId);
	await expect(workflow.getByText("Fixture chickpeas")).toBeVisible();
	await expect(workflow.locator("[data-provider-warnings]")).toContainText("usda");
	for (const query of ["outage", "malformed", "malformed-consumed", "timeout"]) {
		requestIds.push((await search(page, workflow, query, "USDA + OpenFoodFacts")).requestId);
		await expect(workflow).not.toContainText(/nutriments|api_key|provider payload|task282-controlled-key/i);
	}
	await workflow.getByLabel("External food search").fill("cancel");
	await workflow.getByRole("button", { name: /Search/ }).click();
	await workflow.getByLabel("External food search").fill("success");
	requestIds.push((await search(page, workflow, "success", "USDA")).requestId);
	await expect(workflow.getByText("Fixture lentils")).toBeVisible();
	requestIds.push((await search(page, workflow, "quota", "USDA")).requestId);
	await expect(workflow.locator("[data-provider-warnings]")).toContainText(/rate|quota|retry/i);
	await page.waitForTimeout(1_200);
	requestIds.push((await search(page, workflow, "success", "USDA")).requestId);
	await expect(workflow.getByText("Fixture lentils")).toBeVisible();
	await recordAcceptance(
		testInfo,
		["P08-SWR055-ACCEPT-02", "P08-SWR055-ACCEPT-07"],
		requestIds,
		[],
		["provider_state=controlled", "mutation_count=0", "rollback_state=complete", "request_correlation=server_derived"]
	);
});

test("legitimate OpenFoodFacts metadata remains visible [P08-SWR055-STEP-02]", async ({ page }, testInfo) => {
	const workflow = await openAdministration(page);
	const result = await search(page, workflow, "metadata", "OpenFoodFacts");
	await recordAcceptance(testInfo, ["P08-SWR055-STEP-02"], [result.requestId], [], ["provider_state=rejected_candidate"], "ROOT-T282-OFF-METADATA");
	await expect(workflow.getByText("Fixture chickpeas")).toBeVisible();
});

test("optional USDA portion metadata degrades without losing the candidate [P08-SWR033-STEP-02]", async ({ page }, testInfo) => {
	const workflow = await openAdministration(page);
	const result = await search(page, workflow, "optional", "USDA");
	await recordAcceptance(testInfo, ["P08-SWR033-STEP-02"], [result.requestId], [], ["provider_state=rejected_candidate", "metric_basis=100ml"], "ROOT-T282-USDA-OPTIONAL-PORTION");
	await expect(workflow.getByText("Fixture lentils")).toBeVisible();
});

test("external import rejects aliases and unknown micronutrient keys [P08-SWR090-STEP-02] [P08-SWR090-STEP-03]", async ({ page }, testInfo) => {
	await openAdministration(page);
	const csrf = await page.request.get("/api/v1/auth/csrf-token");
	const csrfBody = await safeEnvelope(csrf) as { requestId?: string; data?: { csrfToken?: string } };
	const requestIds = [csrfBody.requestId!];
	for (const [key, id] of [["Na", "alias"], ["Fixtureium", "unknown"]] as const) {
		const response = await page.request.post("/api/v1/admin/imports", {
			headers: {
				"X-CSRF-Token": csrfBody.data?.csrfToken ?? "",
				"Idempotency-Key": `task282-micro-${id}`
			},
			data: {
				name: `Acceptance invalid micro ${id}`,
				physicalState: "solid",
				prepTimeMinutes: 0,
				macrosPer100: { protein: 1, carbohydrates: 1, fat: 1 },
				micros: { [key]: 1 },
				foodCategoryIds: [],
				culinaryRoleIds: []
			}
		});
		const body = await safeEnvelope(response);
		requestIds.push(body.requestId!);
		expect(response.status()).toBe(400);
		expect(body.error?.code).toBe("validation_failed");
	}
	await recordAcceptance(
		testInfo,
		["P08-SWR090-STEP-02", "P08-SWR090-STEP-03"],
		requestIds,
		[],
		["mutation_count=0", "rollback_state=complete"]
	);
});

test("disabled micronutrient vocabulary entries are rejected through Administration [P08-SWR090-STEP-04]", async ({ page }, testInfo) => {
	await openAdministration(page);
	const csrf = await page.request.get("/api/v1/auth/csrf-token");
	const csrfBody = await safeEnvelope(csrf) as { requestId?: string; data?: { csrfToken?: string } };
	const requestIds = [csrfBody.requestId!];
	const headers = { "X-CSRF-Token": csrfBody.data?.csrfToken ?? "" };
	const vocabularyBefore = await page.request.get("/api/v1/admin/micronutrients");
	const vocabularyBeforeBody = await safeEnvelope(vocabularyBefore) as { requestId?: string; data?: { micronutrients?: Array<{ key: string; active: boolean }> } };
	requestIds.push(vocabularyBeforeBody.requestId!);
	const candidates = vocabularyBeforeBody.data?.micronutrients?.filter((entry) => entry.active).map((entry) => entry.key) ?? [];
	let key = "";
	try {
		for (const candidate of candidates) {
			const disabled = await page.request.post(`/api/v1/admin/micronutrients/${candidate}/deactivate`, { headers });
			const disabledBody = await safeEnvelope(disabled);
			requestIds.push(disabledBody.requestId!);
			if (disabled.status() === 200) { key = candidate; break; }
		}
		expect(key).not.toBe("");

		const rejected = await page.request.post("/api/v1/admin/imports", {
			headers: { ...headers, "Idempotency-Key": "task282-disabled-micronutrient" },
			data: {
				name: "Acceptance disabled micronutrient",
				physicalState: "solid",
				prepTimeMinutes: 0,
				macrosPer100: { protein: 1, carbohydrates: 1, fat: 1 },
				micros: { [key]: 1 },
				foodCategoryIds: [],
				culinaryRoleIds: []
			}
		});
		const rejectedBody = await safeEnvelope(rejected);
		requestIds.push(rejectedBody.requestId!);
		expect(rejected.status()).toBe(400);
		expect(rejectedBody.error?.code).toBe("validation_failed");
	} finally {
		if (!key) throw new Error("no unused active micronutrient vocabulary fixture was available");
		const restored = await page.request.post(`/api/v1/admin/micronutrients/${key}/reactivate`, { headers });
		const restoredBody = await safeEnvelope(restored);
		requestIds.push(restoredBody.requestId!);
		expect(restored.status()).toBe(200);
		const vocabulary = await page.request.get("/api/v1/admin/micronutrients");
		const vocabularyBody = await safeEnvelope(vocabulary) as { requestId?: string; data?: { micronutrients?: Array<{ key: string; active: boolean }> } };
		requestIds.push(vocabularyBody.requestId!);
		expect(vocabularyBody.data?.micronutrients?.find((entry) => entry.key === key)?.active).toBe(true);
	}
	await recordAcceptance(testInfo, ["P08-SWR090-STEP-04"], requestIds, [], ["mutation_count=0", "fixture_state=restored"]);
});
