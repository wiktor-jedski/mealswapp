import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { expect, test, type APIResponse, type Page, type TestInfo } from "@playwright/test";
import { fixture, openSidebarForControl, recordAcceptance, responseRequestId, signIn, screenshot } from "./task281-acceptance-helpers";

// Implements DESIGN-009 AdminController manual catalog/classification acceptance.
const enabled = process.env.MEALSWAPP_TASK283_REAL_E2E === "1" && process.env.MEALSWAPP_REAL_STACK_MANAGED === "1";
const task294Enabled = process.env.MEALSWAPP_TASK294_REAL_E2E === "1";
const ROOTS = {
	"019": "ROOT-T283-FILTER-CROSS-INSTANCE",
	"032": "ROOT-T283-METRIC-NORMALIZATION",
	"033": "ROOT-T283-DISCOVERY-PARTITION",
	"056": "ROOT-T283-MANUAL-CATALOG",
	"057": "ROOT-T283-CLASSIFICATION-LIFECYCLE",
	"090": "ROOT-T283-MICRONUTRIENT-VALIDATION"
} as const;
const BLOCKED_CRITERIA = [
	"P08-SWR033-STEP-01", "P08-SWR033-STEP-02", "P08-SWR033-STEP-03", "P08-SWR033-STEP-04",
] as const;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
test.skip(!enabled, "Run scripts/run-task283-acceptance.py for isolated real-stack evidence.");
test.beforeAll(async ({}, info) => {
	if (info.project.name === "real-stack-mobile-chromium") {
		info.setTimeout(70_000);
		await new Promise((resolve) => setTimeout(resolve, 61_000));
	}
});

interface MacroProfile {
	protein: number;
	carbohydrates: number;
	fat: number;
}

interface AdminItem {
	id: string;
	name: string;
	physicalState: "solid" | "liquid";
	densityGramsPerMilliliter?: number;
	densitySourceKind?: string;
	macrosPer100: MacroProfile;
	micros: Record<string, number>;
	allergenKeys: string[];
	foodCategories: Array<{ id: string; name: string; kind: string }>;
	culinaryRoles: Array<{ id: string; name: string; kind: string }>;
}

interface SearchItem {
	id: string;
	name: string;
	physicalState: "solid" | "liquid";
	macroBasis: "100g" | "100ml";
	macros: MacroProfile;
	classifications: Array<{ id: string; name: string; kind: string }>;
}

interface AdminSearchItem {
	itemId: string;
	name: string;
	physicalState: "solid" | "liquid";
	macrosPer100: MacroProfile;
}

interface SourceSummary {
	macros: MacroProfile;
	calories: number;
	totalGrams: number;
	totalMilliliters: number;
}

interface OperationEvidence {
	schema: "mealswapp.task283-operation.v1";
	criterionIds: string[];
	kind: "item" | "classification" | "rejected_item" | "rejected_classification" | "audit_rollback";
	entityId?: string;
	privateItemId?: string;
	name: string;
	idempotencyKey?: string;
	requestIds: string[];
	expected: Record<string, unknown>;
	generationSnapshots?: Record<string, string>;
	observed?: Record<string, string | number | boolean | null>;
}

function rootFor(criterionId: string): string {
	return ROOTS[criterionId.slice(7, 10) as keyof typeof ROOTS];
}

async function requireManaged(): Promise<void> {
	if (!enabled) test.skip(true, "Task 283 requires the managed real stack");
}

async function admin(page: Page, _info: TestInfo): Promise<void> {
	const stateDirectory = fixture("MEALSWAPP_TASK283_AUTH_STATE_DIR");
	const statePath = join(stateDirectory, "shared.json");
	if (existsSync(statePath)) {
		const state = JSON.parse(readFileSync(statePath, "utf8")) as {
			cookies?: Parameters<ReturnType<Page["context"]>["addCookies"]>[0];
		};
		await page.context().addCookies(state.cookies ?? []);
	}
	await page.goto("/");
	const mobileSidebar = page.getByLabel("Open activity sidebar");
	if (await mobileSidebar.isVisible()) await mobileSidebar.click();
	if (await page.getByRole("button", { name: "Sign in", exact: true }).isVisible()) {
		await signIn(page);
		mkdirSync(stateDirectory, { recursive: true, mode: 0o700 });
		await page.context().storageState({ path: statePath });
	}
	const administration = page.getByRole("button", { name: "Administration" });
	await openSidebarForControl(page, administration);
	await administration.click();
	await expect(page.locator("[data-admin-data-management]")).toBeVisible();
	await page.context().storageState({ path: statePath });
}

function secondAPI(): string {
	const value = fixture("MEALSWAPP_TASK283_SECOND_API_URL");
	const url = new URL(value);
	if (url.protocol !== "http:" || url.hostname !== "127.0.0.1" || !url.port || url.pathname !== "/" || url.search || url.hash) {
		throw new Error("Task 283 second API must be an uncredentialed loopback origin");
	}
	return value.replace(/\/$/, "");
}

async function csrf(page: Page, baseURL = ""): Promise<string> {
	const response = await page.request.get(`${baseURL}/api/v1/auth/csrf-token`);
	expect(response.status()).toBe(200);
	const body = await response.json() as { data?: { csrfToken?: string } };
	expect(body.data?.csrfToken).toBeTruthy();
	return body.data!.csrfToken!;
}

async function item(response: APIResponse): Promise<AdminItem> {
	const raw = await response.text();
	let body: { data?: AdminItem };
	try {
		body = JSON.parse(raw) as { data?: AdminItem };
	} catch (error) {
		throw new Error(`Task 283 item response is not JSON: ${raw.slice(0, 500)}`, { cause: error });
	}
	expect(body.data?.id).toMatch(UUID);
	return body.data!;
}

async function classification(response: APIResponse): Promise<{ id: string; name: string; parentId?: string }> {
	const body = await response.json() as { data?: { classification?: { id?: string; name?: string; parentId?: string } } };
	expect(body.data?.classification?.id).toMatch(UUID);
	return body.data!.classification as { id: string; name: string; parentId?: string };
}

function solid(name: string, macros: MacroProfile = { protein: 10, carbohydrates: 20, fat: 3 }, extras: Record<string, unknown> = {}): Record<string, unknown> {
	return { name, physicalState: "solid", macrosPer100: macros, micros: {}, foodCategoryIds: [], culinaryRoleIds: [], allergenKeys: [], ...extras };
}

function liquid(name: string): Record<string, unknown> {
	return {
		name,
		physicalState: "liquid",
		densityGramsPerMilliliter: 0.92,
		densitySourceKind: "manual",
		macrosPer100: { protein: 2.5, carbohydrates: 6.25, fat: 1.75 },
		micros: {},
		foodCategoryIds: [],
		culinaryRoleIds: [],
		allergenKeys: []
	};
}

async function createItem(page: Page, token: string, body: Record<string, unknown>, key = crypto.randomUUID(), baseURL = ""): Promise<{ response: APIResponse; value: AdminItem; key: string }> {
	const response = await page.request.post(`${baseURL}/api/v1/admin/items`, {
		headers: { "X-CSRF-Token": token, "Idempotency-Key": key },
		data: body
	});
	expect(response.status()).toBe(201);
	return { response, value: await item(response), key };
}

async function search(page: Page, data: Record<string, unknown>, baseURL = ""): Promise<{ response: APIResponse; items: SearchItem[]; sourceSummary?: SourceSummary }> {
	const response = await page.request.post(`${baseURL}/api/v1/search`, { data });
	expect(response.status()).toBe(200);
	const body = await response.json() as { data?: { items?: SearchItem[]; sourceSummary?: SourceSummary } };
	expect(Array.isArray(body.data?.items)).toBeTruthy();
	return { response, items: body.data!.items!, sourceSummary: body.data?.sourceSummary };
}

async function adminSearch(page: Page, name: string, baseURL = ""): Promise<{ response: APIResponse; items: AdminSearchItem[] }> {
	const response = await page.request.get(`${baseURL}/api/v1/admin/items?query=${encodeURIComponent(name)}&page=1&pageSize=50`);
	expect(response.status()).toBe(200);
	const body = await response.json() as { data?: { items?: AdminSearchItem[] } };
	expect(Array.isArray(body.data?.items)).toBeTruthy();
	return { response, items: body.data!.items! };
}

function generation(): string {
	const container = fixture("MEALSWAPP_TASK283_REDIS_CONTAINER");
	if (!/^mealswapp-e2e-[0-9a-f]{24}$/.test(container)) throw new Error("Task 283 Redis container identity is invalid");
	return execFileSync("docker", ["exec", container, "redis-cli", "GET", "classification:cache-generation:v1"], { encoding: "utf8" }).trim();
}

function generationSnapshot(): { id: string; value: string } {
	const id = crypto.randomUUID();
	const directory = fixture("MEALSWAPP_TASK283_REDIS_OBSERVATION_REQUEST_DIR");
	const request = join(directory, `${id}.request.json`);
	const response = join(directory, `${id}.response.json`);
	writeFileSync(request, `${JSON.stringify({ schema: "mealswapp.task283-redis-observation-request.v1", id })}\n`, { encoding: "utf8", mode: 0o600 });
	const deadline = Date.now() + 10_000;
	while (!existsSync(response) && Date.now() < deadline) {
		Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, 10);
	}
	if (!existsSync(response)) throw new Error("Task 283 Redis observation timed out");
	const observed = JSON.parse(readFileSync(response, "utf8")) as { schema?: string; id?: string; value?: string; error?: string };
	if (observed.schema !== "mealswapp.task283-redis-observation.v1" || observed.id !== id || !observed.value?.match(/^\d+$/)) {
		throw new Error(`Task 283 independent Redis observation mismatch: ${observed.error ?? "value changed"}`);
	}
	return { id, value: observed.value };
}

function runAuditFailureSQL(statement: "install" | "drop"): void {
	const databaseURL = fixture("MEALSWAPP_DATABASE_URL");
	const url = new URL(databaseURL);
	if (!/^mealswapp_e2e_[0-9a-f]{24}_test$/.test(url.pathname.slice(1)) || !["127.0.0.1", "localhost"].includes(url.hostname)) {
		throw new Error("Task 283 audit injection target is not the run-owned database");
	}
	const sql = statement === "install"
		? "CREATE FUNCTION task283_reject_manual_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action = 'manual_create' THEN RAISE EXCEPTION 'task 283 forced audit failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER task283_reject_manual_audit BEFORE INSERT ON admin_audit_entries FOR EACH ROW EXECUTE FUNCTION task283_reject_manual_audit();"
		: "DROP TRIGGER IF EXISTS task283_reject_manual_audit ON admin_audit_entries; DROP FUNCTION IF EXISTS task283_reject_manual_audit();";
	execFileSync("psql", [databaseURL, "-X", "-v", "ON_ERROR_STOP=1", "-c", sql], { stdio: "pipe" });
}

async function evidence(info: TestInfo, slug: string, operation: Omit<OperationEvidence, "schema">): Promise<string> {
	const relative = `operations/${info.project.name}-${slug}.json`;
	const root = fixture("PHASE08_ACCEPTANCE_RESULT_DIR");
	mkdirSync(join(root, "operations"), { recursive: true });
	writeFileSync(join(root, relative), `${JSON.stringify({ schema: "mealswapp.task283-operation.v1", ...operation }, null, 2)}\n`, { encoding: "utf8", mode: 0o600 });
	return relative;
}

async function record(
	info: TestInfo,
	slug: string,
	criteria: string[],
	operation: Omit<OperationEvidence, "schema" | "criterionIds">,
	summaries: string[] = [],
	rootCauseId?: string,
	requiredProjects?: string[]
): Promise<void> {
	const path = await evidence(info, slug, { criterionIds: criteria, ...operation });
	await recordAcceptance(info, criteria, operation.requestIds, [{ type: "backend", path }], summaries, rootCauseId, requiredProjects);
}

// Exercises the real Vite proxy seam, not a Playwright route stub: the first
// production create response is malformed after the backend has committed it.
test("Task 294 production transport corruption recovers exactly once", async ({ page }, info) => {
	if (!task294Enabled) test.skip(true, "Task 294 real-stack seam is disabled");
	if (info.project.name === "real-stack-mobile-chromium") test.skip(true, "The one-shot transport seam is consumed by the desktop acceptance run");
	await requireManaged();
	await admin(page, info);
	const name = `Task 294 transport ${info.project.name}`;
	const beforeGeneration = generationSnapshot();
	const requests: Array<{ key: string; body: Record<string, unknown>; csrf: string }> = [];
	page.on("request", async (request) => {
		if (request.method() === "POST" && request.url().endsWith("/api/v1/admin/items")) {
			const headers = await request.allHeaders();
			const key = headers["idempotency-key"];
			const csrf = headers["x-csrf-token"];
			if (key && csrf) requests.push({ key, csrf, body: request.postDataJSON() as Record<string, unknown> });
		}
	});
	await page.getByLabel("Name", { exact: true }).first().fill(name);
	await page.getByLabel("Protein per 100").fill("18");
	await page.getByLabel("Carbohydrates per 100").fill("3");
	await page.getByLabel("Fat per 100").fill("9");
	await page.getByRole("button", { name: "Create item" }).click();
	await expect(page.locator("[data-admin-item-recovery]")).toBeFocused();
	await expect(page.getByRole("button", { name: "Create item" })).toBeDisabled();
	await expect.poll(() => requests.length).toBe(1);
	await page.getByRole("button", { name: "Check authoritative items" }).click();
	await expect(page.getByText("Create recovered from authoritative state.")).toBeVisible();
	expect(requests).toHaveLength(1);
	const picker = await page.request.get(`/api/v1/admin/items?query=${encodeURIComponent(name)}&page=1&pageSize=20`);
	expect(picker.status()).toBe(200);
	const pickerBody = await picker.json() as { data?: { total?: number; items?: Array<{ itemId?: string }> } };
	expect(pickerBody.data?.total).toBe(1);
	expect(pickerBody.data?.items).toHaveLength(1);
	const recoveredID = pickerBody.data?.items?.[0]?.itemId;
	expect(recoveredID).toMatch(UUID);
	const authoritative = await page.request.get(`/api/v1/admin/items/${recoveredID}`);
	expect(authoritative.status()).toBe(200);
	const authoritativeBody = await authoritative.json() as { data?: { id?: string; name?: string } };
	expect(authoritativeBody.data).toMatchObject({ id: recoveredID, name });
	const replay = await page.request.post("/api/v1/admin/items", {
		headers: { "X-CSRF-Token": requests[0]!.csrf, "Idempotency-Key": requests[0]!.key },
		data: requests[0]!.body
	});
	expect(replay.status()).toBe(201);
	const replayBody = await replay.json() as { requestId?: string; data?: { id?: string } };
	expect(replayBody.data?.id).toBe(recoveredID);
	expect(replayBody.requestId).toMatch(UUID);
	const afterGeneration = generationSnapshot();
	writeFileSync(join(fixture("PHASE08_ACCEPTANCE_RESULT_DIR"), "task294-transport-proof.json"), `${JSON.stringify({
		schema: "mealswapp.task294-transport-proof.v1", name, entityId: recoveredID,
		idempotencyKey: requests[0]!.key, requestIds: [replayBody.requestId],
		generationSnapshots: { generationBefore: beforeGeneration.id, generationAfter: afterGeneration.id },
		expected: { foodCount: 1, auditCount: 1, idempotencyCount: 1, generationBefore: beforeGeneration.value, generationAfter: afterGeneration.value, generationDelta: 1 }
	}, null, 2)}\n`, { encoding: "utf8", mode: 0o600 });
});

test("solid and liquid creation persists ownerless canonical state and density provenance", async ({ page }, info) => {
	await requireManaged();
	await admin(page, info);
	const token = await csrf(page);
	const solidCreate = await createItem(page, token, solid(`Task 283 solid ${info.project.name}`));
	const liquidCreate = await createItem(page, token, liquid(`Task 283 liquid ${info.project.name}`));
	expect(solidCreate.value).toMatchObject({ name: `Task 283 solid ${info.project.name}`, physicalState: "solid", macrosPer100: { protein: 10, carbohydrates: 20, fat: 3 } });
	expect(liquidCreate.value).toMatchObject({
		name: `Task 283 liquid ${info.project.name}`,
		physicalState: "liquid",
		densityGramsPerMilliliter: 0.92,
		densitySourceKind: "manual",
		macrosPer100: { protein: 2.5, carbohydrates: 6.25, fat: 1.75 }
	});
	const secondRead = await page.request.get(`${secondAPI()}/api/v1/admin/items/${liquidCreate.value.id}`);
	expect(secondRead.status()).toBe(200);
	expect(await item(secondRead)).toEqual(liquidCreate.value);
	const sharedSearchQuery = "Task 283";
	const privatePartitionID = fixture("MEALSWAPP_TASK283_PRIVATE_ITEM_ID");
	const createdPicker = await adminSearch(page, sharedSearchQuery, secondAPI());
	expect(createdPicker.items).toContainEqual(expect.objectContaining({ itemId: solidCreate.value.id, name: solidCreate.value.name, macrosPer100: solidCreate.value.macrosPer100 }));
	expect(createdPicker.items.map(({ itemId }) => itemId)).not.toContain(privatePartitionID);
	const createdCatalog = await search(page, { query: sharedSearchQuery, mode: "catalog", page: 1, filters: [] });
	expect(createdCatalog.items.map(({ id }) => id)).toContain(solidCreate.value.id);
	expect(createdCatalog.items.map(({ id }) => id)).not.toContain(privatePartitionID);
	const createdSubstitution = await search(page, {
		query: sharedSearchQuery, mode: "substitution", page: 1, filters: [],
		substitutionInputs: [{ foodObjectId: solidCreate.value.id, foodObjectType: "food_item", quantity: 100, unit: "g" }]
	}, secondAPI());
	expect(createdSubstitution.items.map(({ id }) => id)).toContain(liquidCreate.value.id);
	expect(createdSubstitution.items.map(({ id }) => id)).not.toContain(privatePartitionID);
	const criteria = ["P08-SWR056-STEP-01", "P08-SWR056-STEP-02", "P08-SWR056-ACCEPT-01", "P08-SWR033-STEP-05"];
	await record(info, "solid-create", criteria, {
		kind: "item", entityId: solidCreate.value.id, name: solidCreate.value.name, idempotencyKey: solidCreate.key,
		requestIds: [await responseRequestId(solidCreate.response), await responseRequestId(createdPicker.response), await responseRequestId(createdCatalog.response)],
		expected: { active: true, ownerless: true, auditActions: { manual_create: 1 }, idempotencyCount: 1, physicalState: "solid", metricBasis: "100g", macros: solidCreate.value.macrosPer100 }
	}, ["mutation_count=1", "audit_count=1", "owner_state=global", "metric_basis=100g"]);
	await record(info, "liquid-create", criteria, {
		kind: "item", entityId: liquidCreate.value.id, name: liquidCreate.value.name, idempotencyKey: liquidCreate.key,
		requestIds: [await responseRequestId(liquidCreate.response), await responseRequestId(secondRead), await responseRequestId(createdSubstitution.response)],
		expected: { active: true, ownerless: true, auditActions: { manual_create: 1 }, idempotencyCount: 1, physicalState: "liquid", metricBasis: "100ml", density: 0.92, densitySourceKind: "manual", macros: liquidCreate.value.macrosPer100 }
	}, ["mutation_count=1", "audit_count=1", "owner_state=global", "metric_basis=100ml"]);
});

// Implements DESIGN-009 AdminController and DESIGN-001 SearchView mobile global-item discovery surfaces for Task 303.
test("mobile global-item discovery refreshes the picker, Catalog, and Substitution projections", async ({ page }, info) => {
	if (info.project.name !== "real-stack-mobile-chromium") test.skip(true, "Task 303 evidence runs on the required mobile project");
	await requireManaged();
	await admin(page, info);
	const sourceName = `Task 283 Task 303 mobile source ${info.project.name}`;
	const targetName = `Task 283 Task 303 mobile target ${info.project.name}`;
	const privateItemID = fixture("MEALSWAPP_TASK283_PRIVATE_ITEM_ID");

	const warmAutocomplete = await page.request.get(`${secondAPI()}/api/v1/search/autocomplete?query=${encodeURIComponent(targetName)}`);
	const warmCatalog = await search(page, { query: targetName, mode: "catalog", page: 1, filters: [] }, secondAPI());
	const warmPicker = await adminSearch(page, targetName, secondAPI());
	expect(warmAutocomplete.status()).toBe(200);
	expect(warmCatalog.items).toHaveLength(0);
	expect(warmPicker.items).toHaveLength(0);

	const token = await csrf(page);
	const sourceCreate = await createItem(page, token, solid(sourceName, { protein: 18, carbohydrates: 9, fat: 4 }));
	const warmAutocompleteAfterSource = await page.request.get(`${secondAPI()}/api/v1/search/autocomplete?query=${encodeURIComponent(targetName)}`);
	const warmCatalogAfterSource = await search(page, { query: targetName, mode: "catalog", page: 1, filters: [] }, secondAPI());
	const warmPickerAfterSource = await adminSearch(page, targetName, secondAPI());
	expect(warmAutocompleteAfterSource.status()).toBe(200);
	expect(warmCatalogAfterSource.items).toHaveLength(0);
	expect(warmPickerAfterSource.items).toHaveLength(0);

	const targetToken = await csrf(page);
	const targetCreate = await createItem(page, targetToken, solid(targetName));
	const createdAutocomplete = await page.request.get(`/api/v1/search/autocomplete?query=${encodeURIComponent(targetName)}`);
	const createdAutocompleteBody = await createdAutocomplete.json() as { data?: { items?: Array<{ itemId: string; label: string }> } };
	const createdCatalog = await search(page, { query: targetName, mode: "catalog", page: 1, filters: [] });
	const createdSubstitution = await search(page, {
		query: targetName, mode: "substitution", page: 1, filters: [],
		substitutionInputs: [{ foodObjectId: sourceCreate.value.id, foodObjectType: "food_item", quantity: 100, unit: "g" }]
	}, secondAPI());
	const createdPicker = await adminSearch(page, targetName, secondAPI());
	expect(createdAutocomplete.status()).toBe(200);
	expect(createdAutocompleteBody.data?.items).toContainEqual(expect.objectContaining({ itemId: targetCreate.value.id, label: targetName }));
	expect(createdCatalog.items.map(({ id }) => id)).toContain(targetCreate.value.id);
	expect(createdCatalog.items.map(({ id }) => id)).not.toContain(privateItemID);
	expect(createdSubstitution.items.map(({ id }) => id)).toContain(targetCreate.value.id);
	expect(createdSubstitution.items.map(({ id }) => id)).not.toContain(privateItemID);
	expect(createdPicker.items).toContainEqual(expect.objectContaining({ itemId: targetCreate.value.id, name: targetName }));
	expect(createdPicker.items.map(({ itemId }) => itemId)).not.toContain(privateItemID);

	const pickerForm = page.getByRole("form", { name: "Search global items" });
	await pickerForm.getByLabel("Item name").fill(targetName);
	await pickerForm.getByRole("button", { name: "Search", exact: true }).click();
	const pickerResult = page.locator("[data-admin-item-search-result]").filter({ hasText: targetName });
	await expect(pickerResult).toBeVisible();
	await pickerResult.getByRole("button", { name: `Edit ${targetName}`, exact: true }).click();
	await expect(page.getByLabel("Name", { exact: true }).first()).toHaveValue(targetName);

	await page.goto("/");
	const catalogInput = page.getByLabel("Food search");
	await catalogInput.fill(targetName);
	const catalogOption = page.getByRole("option", { name: targetName, exact: true });
	await expect(catalogOption).toBeVisible();
	await catalogOption.click();
	await expect(page.locator(`[data-result-id="${targetCreate.value.id}"]`)).toBeVisible();
	await expect(page.locator(`[data-result-id="${privateItemID}"]`)).toHaveCount(0);

	await page.goto("/?mode=substitution");
	const substitutionInput = page.getByLabel("Food search");
	await substitutionInput.fill(sourceName);
	const sourceOption = page.getByRole("option", { name: sourceName, exact: true });
	await expect(sourceOption).toBeVisible();
	await sourceOption.click();
	await expect(page.locator(`[data-food-object-id="${sourceCreate.value.id}"]`)).toBeVisible();
	await expect(page.locator("[data-substitution-search]")).toBeEnabled();
	await substitutionInput.fill(targetName);
	await page.locator("[data-substitution-search]").click();
	await expect(page.locator(`[data-result-id="${targetCreate.value.id}"]`)).toBeVisible();
	await expect(page.locator(`[data-result-id="${privateItemID}"]`)).toHaveCount(0);

	const deletion = await page.request.delete(`${secondAPI()}/api/v1/admin/items/${targetCreate.value.id}`, { headers: { "X-CSRF-Token": await csrf(page, secondAPI()) } });
	const deletedAutocomplete = await page.request.get(`${secondAPI()}/api/v1/search/autocomplete?query=${encodeURIComponent(targetName)}`);
	const deletedAutocompleteBody = await deletedAutocomplete.json() as { data?: { items?: Array<{ itemId: string }> } };
	const deletedPicker = await adminSearch(page, targetName, secondAPI());
	const deletedCatalog = await search(page, { query: targetName, mode: "catalog", page: 1, filters: [] }, secondAPI());
	const deletedSubstitution = await search(page, {
		query: targetName, mode: "substitution", page: 1, filters: [],
		substitutionInputs: [{ foodObjectId: sourceCreate.value.id, foodObjectType: "food_item", quantity: 100, unit: "g" }]
	}, secondAPI());
	expect(deletion.status()).toBe(204);
	expect((deletedAutocompleteBody.data?.items ?? []).map(({ itemId }) => itemId)).not.toContain(targetCreate.value.id);
	expect(deletedPicker.items.map(({ itemId }) => itemId)).not.toContain(targetCreate.value.id);
	expect(deletedCatalog.items.map(({ id }) => id)).not.toContain(targetCreate.value.id);
	expect(deletedSubstitution.items.map(({ id }) => id)).not.toContain(targetCreate.value.id);

	await page.goto("/");
	await page.getByLabel("Food search").fill(targetName);
	await expect(page.getByRole("option", { name: targetName, exact: true })).toHaveCount(0);
	const criteria = ["P08-SWR056-STEP-01", "P08-SWR056-STEP-02", "P08-SWR056-STEP-03", "P08-SWR056-STEP-04", "P08-SWR056-STEP-05", "P08-SWR056-STEP-06", "P08-SWR056-ACCEPT-01", "P08-SWR056-ACCEPT-05", "P08-SWR056-ACCEPT-06", "P08-SWR033-STEP-05"];
	await record(info, "mobile-global-discovery", criteria, {
		kind: "item", entityId: targetCreate.value.id, privateItemId: privateItemID, name: targetName, idempotencyKey: targetCreate.key,
		requestIds: [await responseRequestId(sourceCreate.response), await responseRequestId(targetCreate.response), await responseRequestId(warmCatalog.response), await responseRequestId(warmCatalogAfterSource.response), await responseRequestId(createdAutocomplete), await responseRequestId(createdCatalog.response), await responseRequestId(createdSubstitution.response), await responseRequestId(createdPicker.response), await responseRequestId(deletedAutocomplete), await responseRequestId(deletedPicker.response), await responseRequestId(deletedCatalog.response), await responseRequestId(deletedSubstitution.response)],
		expected: {
			active: false, deleted: true, ownerless: true,
			auditActions: { manual_create: 1, manual_delete: 1 }, idempotencyCount: 1, name: targetName,
			partition: { globalCount: 1, privateCount: 1, globalOwnerless: true, privateOwned: true },
			mobilePickerContainsGlobal: true, mobileCatalogContainsGlobal: true, mobileSubstitutionContainsGlobal: true,
			catalogExcludesPrivate: true, substitutionExcludesPrivate: true, deletedAutocompleteContainsGlobal: false,
			deletedPickerContainsGlobal: false, deletedCatalogContainsGlobal: false, deletedSubstitutionContainsGlobal: false,
			mobileDeletedAutocompleteContainsGlobal: false
		},
		observed: {
			mobilePickerContainsGlobal: true,
			mobileCatalogContainsGlobal: (createdCatalog.items.map(({ id }) => id)).includes(targetCreate.value.id),
			mobileSubstitutionContainsGlobal: (createdSubstitution.items.map(({ id }) => id)).includes(targetCreate.value.id),
			catalogExcludesPrivate: !createdCatalog.items.map(({ id }) => id).includes(privateItemID),
			substitutionExcludesPrivate: !createdSubstitution.items.map(({ id }) => id).includes(privateItemID),
			deletedAutocompleteContainsGlobal: (deletedAutocompleteBody.data?.items ?? []).map(({ itemId }) => itemId).includes(targetCreate.value.id),
			deletedPickerContainsGlobal: deletedPicker.items.map(({ itemId }) => itemId).includes(targetCreate.value.id),
			deletedCatalogContainsGlobal: deletedCatalog.items.map(({ id }) => id).includes(targetCreate.value.id),
			deletedSubstitutionContainsGlobal: deletedSubstitution.items.map(({ id }) => id).includes(targetCreate.value.id),
			mobileDeletedAutocompleteContainsGlobal: await page.getByRole("option", { name: targetName, exact: true }).count() > 0
		}
	}, [], undefined, ["real-stack-mobile-chromium"]);
});

test("invalid density, nutrition, classification, and image inputs roll back independently", async ({ page }, info) => {
	await requireManaged();
	await admin(page, info);
	const token = await csrf(page);
	const invalidBodies = [
		liquid(`Task 283 invalid density ${info.project.name}`),
		solid(`Task 283 invalid nutrition ${info.project.name}`, { protein: -1, carbohydrates: 0, fat: 0 }),
		solid(`Task 283 invalid classification ${info.project.name}`, undefined, { foodCategoryIds: ["00000000-0000-0000-0000-000000000000"] }),
		solid(`Task 283 invalid image ${info.project.name}`, undefined, { imageUrl: "://invalid" })
	];
	delete invalidBodies[0].densityGramsPerMilliliter;
	delete invalidBodies[0].densitySourceKind;
	for (const [index, body] of invalidBodies.entries()) {
		const key = crypto.randomUUID();
		const before = generationSnapshot();
		const response = await page.request.post(`${index % 2 === 0 ? secondAPI() : ""}/api/v1/admin/items`, {
			headers: { "X-CSRF-Token": await csrf(page, index % 2 === 0 ? secondAPI() : ""), "Idempotency-Key": key },
			data: body
		});
		expect(response.status()).toBe(400);
		const after = generationSnapshot();
		expect(after.value).toBe(before.value);
		const criteria = index === 0 ? ["P08-SWR056-ACCEPT-02"] : ["P08-SWR056-ACCEPT-03"];
		await record(info, `invalid-${index}`, criteria, {
			kind: "rejected_item", name: String(body.name), idempotencyKey: key,
			requestIds: [await responseRequestId(response)],
			expected: { rowCount: 0, auditCount: 0, idempotencyCount: 0, generationBefore: before.value, generationAfter: after.value, generationDelta: 0 },
			generationSnapshots: { generationBefore: before.id, generationAfter: after.id }
		}, ["http_status=400", "rollback_state=complete", "cache_generation=unchanged"]);
	}
});

test("create replay returns one stable identity, one row, one audit, and one durable claim", async ({ page }, info) => {
	await requireManaged();
	await admin(page, info);
	const token = await csrf(page);
	const key = crypto.randomUUID();
	const body = solid(`Task 283 replay ${info.project.name}`);
	const first = await createItem(page, token, body, key);
	const replayResponse = await page.request.post("/api/v1/admin/items", { headers: { "X-CSRF-Token": token, "Idempotency-Key": key }, data: body });
	expect(replayResponse.status()).toBe(201);
	const replay = await item(replayResponse);
	expect(replay).toEqual(first.value);
	await record(info, "idempotent-replay", ["P08-SWR056-ACCEPT-04"], {
		kind: "item", entityId: first.value.id, name: first.value.name, idempotencyKey: key,
		requestIds: [await responseRequestId(first.response), await responseRequestId(replayResponse)],
		expected: { active: true, rowCount: 1, auditActions: { manual_create: 1 }, idempotencyCount: 1 }
	}, ["mutation_count=1", "audit_count=1", "row_count=1"]);
});

test("catalog include/exclude filters use stable IDs and survive a cross-instance rename", async ({ page }, info) => {
	await requireManaged();
	await admin(page, info);
	const primaryToken = await csrf(page);
	const categoryResponse = await page.request.post("/api/v1/admin/classifications/food_category", { headers: { "X-CSRF-Token": primaryToken }, data: { name: `Task 283 filter ${info.project.name}` } });
	expect(categoryResponse.status()).toBe(201);
	const category = await classification(categoryResponse);
	const matching = await createItem(page, primaryToken, solid(`Task 283 filter match ${info.project.name}`, undefined, { foodCategoryIds: [category.id] }));
	const other = await createItem(page, primaryToken, solid(`Task 283 filter other ${info.project.name}`));
	const secondaryToken = await csrf(page, secondAPI());
	const query = `Task 283 filter`;
	const includeBody = { query, mode: "catalog", page: 1, filters: [{ filterId: category.id, kind: "food_category", include: true }] };
	expect(JSON.stringify(includeBody)).toContain(category.id);
	expect(JSON.stringify(includeBody)).not.toContain(category.name);
	const included = await search(page, includeBody, secondAPI());
	expect(included.items.map((value) => value.id)).toContain(matching.value.id);
	expect(included.items.map((value) => value.id)).not.toContain(other.value.id);
	expect(included.items.every((value) => value.classifications.some((entry) => entry.id === category.id))).toBeTruthy();
	const excluded = await search(page, { ...includeBody, filters: [{ ...includeBody.filters[0], include: false }] });
	expect(excluded.items.map((value) => value.id)).not.toContain(matching.value.id);
	expect(excluded.items.map((value) => value.id)).toContain(other.value.id);
	const renamed = `${category.name} renamed`;
	const renameGenerationBefore = generationSnapshot();
	const renameResponse = await page.request.put(`${secondAPI()}/api/v1/admin/classifications/${category.id}`, { headers: { "X-CSRF-Token": secondaryToken }, data: { name: renamed } });
	expect(renameResponse.status()).toBe(200);
	const renameGenerationAfter = generationSnapshot();
	expect(Number(renameGenerationAfter.value) - Number(renameGenerationBefore.value)).toBe(1);
	expect(await classification(renameResponse)).toMatchObject({ id: category.id, name: renamed });
	const afterRename = await search(page, includeBody);
	expect(afterRename.items.map((value) => value.id)).toContain(matching.value.id);
	expect(afterRename.items.find((value) => value.id === matching.value.id)?.classifications).toContainEqual({ id: category.id, name: renamed, kind: "food_category" });
	const options = await page.request.get(`${secondAPI()}/api/v1/search/filter-options?mode=substitution`);
	expect(options.status()).toBe(200);
	const optionsText = JSON.stringify(await options.json());
	expect(optionsText).toContain(category.id);
	expect(optionsText).toContain(renamed);
	expect(optionsText).not.toContain(`"${category.name}"`);
	const criteria = ["P08-SWR019-STEP-01", "P08-SWR019-STEP-02", "P08-SWR019-STEP-03", "P08-SWR019-STEP-04", "P08-SWR057-ACCEPT-03", "P08-SWR057-ACCEPT-05"];
	await record(info, "stable-id-filter", criteria, {
		kind: "classification", entityId: category.id, name: renamed,
		requestIds: [await responseRequestId(categoryResponse), await responseRequestId(renameResponse), await responseRequestId(included.response), await responseRequestId(excluded.response), await responseRequestId(afterRename.response), await responseRequestId(options)],
		expected: { active: true, kind: "food_category", auditActions: { "classification.create": 1, "classification.update": 1 }, parentId: null, generationBefore: renameGenerationBefore.value, generationAfter: renameGenerationAfter.value, generationDelta: 1 },
		generationSnapshots: { generationBefore: renameGenerationBefore.id, generationAfter: renameGenerationAfter.id }
	}, ["mutation_count=2", "audit_count=2", "cache_generation=advanced"]);
});

test("updates are authoritative in Catalog and Substitution Search", async ({ page }, info) => {
	await requireManaged();
	await admin(page, info);
	const token = await csrf(page);
	const source = await createItem(page, token, solid(`Task 283 update source ${info.project.name}`, { protein: 18, carbohydrates: 9, fat: 4 }));
	const target = await createItem(page, token, solid(`Task 283 update only ${info.project.name}`, { protein: 11, carbohydrates: 12, fat: 2 }));
	const updatedName = `${target.value.name} canonical`;
	const updatedMacros = { protein: 21.25, carbohydrates: 12.5, fat: 2.75 };
	const update = await page.request.put(`${secondAPI()}/api/v1/admin/items/${target.value.id}`, {
		headers: { "X-CSRF-Token": await csrf(page, secondAPI()) },
		data: solid(updatedName, updatedMacros)
	});
	expect(update.status()).toBe(200);
	expect(await item(update)).toMatchObject({ id: target.value.id, name: updatedName, macrosPer100: updatedMacros });
	const picker = await adminSearch(page, updatedName);
	expect(picker.items).toContainEqual(expect.objectContaining({ itemId: target.value.id, name: updatedName, macrosPer100: updatedMacros }));
	const catalog = await search(page, { query: updatedName, mode: "catalog", page: 1, filters: [] });
	expect(catalog.items).toContainEqual(expect.objectContaining({ id: target.value.id, name: updatedName, macros: updatedMacros, macroBasis: "100g" }));
	const substitution = await search(page, {
		query: updatedName, mode: "substitution", page: 1, filters: [],
		substitutionInputs: [{ foodObjectId: source.value.id, foodObjectType: "food_item", quantity: 100, unit: "g" }]
	}, secondAPI());
	expect(substitution.items).toContainEqual(expect.objectContaining({ id: target.value.id, name: updatedName, macros: updatedMacros }));
	const autocomplete = await page.request.get(`/api/v1/search/autocomplete?query=${encodeURIComponent(updatedName)}`);
	const autocompleteBody = await autocomplete.json() as { data?: { items?: Array<{ itemId: string; label: string }> } };
	expect(autocompleteBody.data?.items).toContainEqual(expect.objectContaining({ itemId: target.value.id, label: updatedName }));
	const criteria = ["P08-SWR056-STEP-03", "P08-SWR056-STEP-04", "P08-SWR056-ACCEPT-05"];
	await record(info, "update-search", criteria, {
		kind: "item", entityId: target.value.id, name: updatedName, idempotencyKey: target.key,
		requestIds: [await responseRequestId(target.response), await responseRequestId(update), await responseRequestId(picker.response), await responseRequestId(catalog.response), await responseRequestId(substitution.response), await responseRequestId(autocomplete)],
		expected: { active: true, auditActions: { manual_create: 1, manual_update: 1 }, idempotencyCount: 1, name: updatedName, macros: updatedMacros }
	}, ["mutation_count=2", "audit_count=2", "rollback_state=committed"]);
});

test("deletion removes the stable identity from active reads and every search projection", async ({ page }, info) => {
	await requireManaged();
	await admin(page, info);
	const token = await csrf(page);
	const source = await createItem(page, token, solid(`Task 283 source ${info.project.name}`, { protein: 18, carbohydrates: 9, fat: 4 }));
	const target = await createItem(page, token, solid(`Task 283 update ${info.project.name}`, { protein: 11, carbohydrates: 12, fat: 2 }));
	const updatedName = `${target.value.name} canonical`;
	const updatedMacros = { protein: 21.25, carbohydrates: 12.5, fat: 2.75 };
	const update = await page.request.put(`${secondAPI()}/api/v1/admin/items/${target.value.id}`, {
		headers: { "X-CSRF-Token": await csrf(page, secondAPI()) },
		data: solid(updatedName, updatedMacros)
	});
	expect(update.status()).toBe(200);
	expect(await item(update)).toMatchObject({ id: target.value.id, name: updatedName, macrosPer100: updatedMacros });
	const catalog = await search(page, { query: updatedName, mode: "catalog", page: 1, filters: [] });
	expect(catalog.items).toContainEqual(expect.objectContaining({ id: target.value.id, name: updatedName, macros: updatedMacros, macroBasis: "100g" }));
	const substitution = await search(page, {
		query: updatedName,
		mode: "substitution",
		page: 1,
		filters: [],
		substitutionInputs: [{ foodObjectId: source.value.id, foodObjectType: "food_item", quantity: 100, unit: "g" }]
	}, secondAPI());
	expect(substitution.items).toContainEqual(expect.objectContaining({ id: target.value.id, name: updatedName, macros: updatedMacros }));
	const autocomplete = await page.request.get(`/api/v1/search/autocomplete?query=${encodeURIComponent(updatedName)}`);
	expect(autocomplete.status()).toBe(200);
	const autocompleteBody = await autocomplete.json() as { data?: { items?: Array<{ itemId: string; label: string }> } };
	expect(autocompleteBody.data?.items).toContainEqual(expect.objectContaining({ itemId: target.value.id, label: updatedName }));
	const activePicker = await adminSearch(page, updatedName);
	expect(activePicker.items.map(({ itemId }) => itemId)).toContain(target.value.id);
	const deletion = await page.request.delete(`${secondAPI()}/api/v1/admin/items/${target.value.id}`, { headers: { "X-CSRF-Token": await csrf(page, secondAPI()) } });
	expect(deletion.status()).toBe(204);
	const deletedRead = await page.request.get(`/api/v1/admin/items/${target.value.id}`);
	expect(deletedRead.status()).toBe(404);
	const deletedPicker = await adminSearch(page, updatedName, secondAPI());
	expect(deletedPicker.items.map(({ itemId }) => itemId)).not.toContain(target.value.id);
	const deletedCatalog = await search(page, { query: updatedName, mode: "catalog", page: 1, filters: [] }, secondAPI());
	expect(deletedCatalog.items.map((value) => value.id)).not.toContain(target.value.id);
	const deletedSubstitution = await search(page, {
		query: updatedName,
		mode: "substitution",
		page: 1,
		filters: [],
		substitutionInputs: [{ foodObjectId: source.value.id, foodObjectType: "food_item", quantity: 100, unit: "g" }]
	});
	expect(deletedSubstitution.items.map((value) => value.id)).not.toContain(target.value.id);
	const deletedAutocomplete = await page.request.get(`${secondAPI()}/api/v1/search/autocomplete?query=${encodeURIComponent(updatedName)}`);
	const deletedAutocompleteBody = await deletedAutocomplete.json() as { data?: { items?: Array<{ itemId: string }> } };
	const criteria = ["P08-SWR056-STEP-05", "P08-SWR056-STEP-06", "P08-SWR056-ACCEPT-06"];
	await record(info, "update-delete-search", criteria, {
		kind: "item", entityId: target.value.id, name: updatedName, idempotencyKey: target.key,
		requestIds: [await responseRequestId(target.response), await responseRequestId(update), await responseRequestId(activePicker.response), await responseRequestId(catalog.response), await responseRequestId(substitution.response), await responseRequestId(autocomplete), await responseRequestId(deletedRead), await responseRequestId(deletedPicker.response), await responseRequestId(deletedCatalog.response), await responseRequestId(deletedSubstitution.response), await responseRequestId(deletedAutocomplete)],
		expected: { active: false, deleted: true, auditActions: { manual_create: 1, manual_update: 1, manual_delete: 1 }, idempotencyCount: 1, name: updatedName, macros: updatedMacros, autocompleteContainsDeleted: false },
		observed: { autocompleteContainsDeleted: (deletedAutocompleteBody.data?.items?.map((value) => value.itemId) ?? []).includes(target.value.id) }
	}, ["mutation_count=3", "audit_count=3", "rollback_state=committed"]);
});

test("audit persistence failure through API-2 rolls back item, claim, audit, and generation", async ({ page }, info) => {
	await requireManaged();
	await admin(page, info);
	const name = `Task 283 audit rollback ${info.project.name}`;
	const key = crypto.randomUUID();
	const before = generationSnapshot();
	runAuditFailureSQL("install");
	let response: APIResponse;
	try {
		response = await page.request.post(`${secondAPI()}/api/v1/admin/items`, {
			headers: { "X-CSRF-Token": await csrf(page, secondAPI()), "Idempotency-Key": key },
			data: solid(name)
		});
	} finally {
		runAuditFailureSQL("drop");
	}
	expect(response!.status()).toBe(503);
	const after = generationSnapshot();
	expect(after.value).toBe(before.value);
	await record(info, "audit-rollback", ["P08-SWR056-ACCEPT-07"], {
		kind: "audit_rollback", name, idempotencyKey: key,
		requestIds: [await responseRequestId(response!)],
		expected: { rowCount: 0, auditCount: 0, idempotencyCount: 0, generationBefore: before.value, generationAfter: after.value, generationDelta: 0 },
		generationSnapshots: { generationBefore: before.id, generationAfter: after.id }
	}, ["http_status=503", "rollback_state=complete", "cache_generation=unchanged"]);
});

test("classification lifecycle proves duplicate, cycle, in-use conflict, detach, and deletion", async ({ page }, info) => {
	await requireManaged();
	await admin(page, info);
	let token = await csrf(page);
	const create = async (kind: "food_category" | "culinary_role", name: string, parentId?: string) => {
		const response = await page.request.post(`/api/v1/admin/classifications/${kind}`, { headers: { "X-CSRF-Token": token }, data: { name, ...(parentId ? { parentId } : {}) } });
		expect(response.status()).toBe(201);
		return { response, value: await classification(response) };
	};
	const category = await create("food_category", `Task 283 lifecycle ${info.project.name}`);
	const child = await create("food_category", `Task 283 lifecycle child ${info.project.name}`, category.value.id);
	const role = await create("culinary_role", `Task 283 role ${info.project.name}`);
	const duplicateBefore = generationSnapshot();
	const duplicate = await page.request.post(`${secondAPI()}/api/v1/admin/classifications/food_category`, { headers: { "X-CSRF-Token": await csrf(page, secondAPI()) }, data: { name: category.value.name } });
	expect(duplicate.status()).toBe(409);
	const duplicateAfter = generationSnapshot();
	expect(duplicateAfter.value).toBe(duplicateBefore.value);
	const cycle = await page.request.put(`${secondAPI()}/api/v1/admin/classifications/${category.value.id}`, { headers: { "X-CSRF-Token": await csrf(page, secondAPI()) }, data: { name: category.value.name, parentId: child.value.id } });
	expect(cycle.status()).toBe(409);
	const cycleAfter = generationSnapshot();
	expect(cycleAfter.value).toBe(duplicateAfter.value);
	token = await csrf(page);
	const attached = await createItem(page, token, solid(`Task 283 classified ${info.project.name}`, undefined, { foodCategoryIds: [category.value.id], culinaryRoleIds: [role.value.id] }));
	expect(attached.value.foodCategories.map((value) => value.id)).toContain(category.value.id);
	expect(attached.value.culinaryRoles.map((value) => value.id)).toContain(role.value.id);
	const inUseDelete = await page.request.delete(`${secondAPI()}/api/v1/admin/classifications/${category.value.id}`, { headers: { "X-CSRF-Token": await csrf(page, secondAPI()) } });
	expect(inUseDelete.status()).toBe(409);
	token = await csrf(page);
	const detached = await page.request.put(`/api/v1/admin/items/${attached.value.id}`, { headers: { "X-CSRF-Token": token }, data: solid(attached.value.name) });
	expect(detached.status()).toBe(200);
	expect((await item(detached)).foodCategories).toEqual([]);
	const deleteChild = await page.request.delete(`/api/v1/admin/classifications/${child.value.id}`, { headers: { "X-CSRF-Token": token } });
	expect(deleteChild.status()).toBe(204);
	const deleteCategory = await page.request.delete(`${secondAPI()}/api/v1/admin/classifications/${category.value.id}`, { headers: { "X-CSRF-Token": await csrf(page, secondAPI()) } });
	expect(deleteCategory.status()).toBe(204);
	const options = await page.request.get("/api/v1/search/filter-options?mode=substitution");
	expect(options.status()).toBe(200);
	expect(JSON.stringify(await options.json())).not.toContain(category.value.id);
	const criteria = [
		"P08-SWR057-STEP-01", "P08-SWR057-STEP-02", "P08-SWR057-STEP-03", "P08-SWR057-STEP-04",
		"P08-SWR057-STEP-05", "P08-SWR057-STEP-06", "P08-SWR057-STEP-07", "P08-SWR057-ACCEPT-01",
		"P08-SWR057-ACCEPT-02", "P08-SWR057-ACCEPT-04", "P08-SWR057-ACCEPT-06"
	];
	await record(info, "classification-lifecycle", criteria, {
		kind: "classification", entityId: category.value.id, name: category.value.name,
		requestIds: [await responseRequestId(category.response), await responseRequestId(duplicate), await responseRequestId(cycle), await responseRequestId(inUseDelete), await responseRequestId(detached), await responseRequestId(options)],
		expected: { active: false, deleted: true, kind: "food_category", auditActions: { "classification.create": 1, "classification.delete": 1 }, failedGenerationBefore: duplicateBefore.value, failedGenerationAfter: cycleAfter.value, failedGenerationDelta: 0 },
		generationSnapshots: { failedGenerationBefore: duplicateBefore.id, failedGenerationAfter: cycleAfter.id }
	}, ["mutation_count=2", "audit_count=2", "cache_generation=unchanged"]);
});

test("API-2 observes commits while an API-1 stale absolute write is rejected", async ({ page }, info) => {
	await requireManaged();
	await admin(page, info);
	const primaryToken = await csrf(page);
	const created = await createItem(page, primaryToken, solid(`Task 283 stale ${info.project.name}`, { protein: 4, carbohydrates: 5, fat: 6 }));
	const staleRead = await page.request.get(`/api/v1/admin/items/${created.value.id}`);
	expect(staleRead.status()).toBe(200);
	const staleProjection = await staleRead.json();
	const freshUpdate = await page.request.put(`${secondAPI()}/api/v1/admin/items/${created.value.id}`, {
		headers: { "X-CSRF-Token": await csrf(page, secondAPI()) },
		data: solid(`${created.value.name} fresh`, { protein: 7, carbohydrates: 8, fat: 9 })
	});
	expect(freshUpdate.status()).toBe(200);
	const refreshedPrimaryToken = await csrf(page);
	const staleUpdate = await page.request.put(`/api/v1/admin/items/${created.value.id}`, {
		headers: { "X-CSRF-Token": refreshedPrimaryToken, "If-Match": staleProjection.data.updatedAt },
		data: solid(`${created.value.name} stale`, created.value.macrosPer100)
	});
	await record(info, "cross-instance-stale-write", ["P08-SWR057-STEP-08", "P08-SWR057-ACCEPT-05"], {
		kind: "item", entityId: created.value.id, name: created.value.name, idempotencyKey: created.key,
		requestIds: [await responseRequestId(created.response), await responseRequestId(staleRead), await responseRequestId(freshUpdate), await responseRequestId(staleUpdate)],
		expected: { active: true, staleWriteStatus: 409, auditActions: { manual_create: 1, manual_update: 1 }, idempotencyCount: 1 },
		observed: { staleWriteStatus: staleUpdate.status() }
	}, ["http_status=" + staleUpdate.status(), "request_correlation=cross_instance"], ROOTS["057"]);
});

test("metric and imperial quantities preserve solid and liquid backend calculations once", async ({ page }, info) => {
	await requireManaged();
	await admin(page, info);
	const token = await csrf(page);
	const solidItem = await createItem(page, token, solid(`Task 283 units solid ${info.project.name}`, { protein: 10, carbohydrates: 5, fat: 2 }));
	const liquidItem = await createItem(page, token, liquid(`Task 283 units liquid ${info.project.name}`));
	const source = await createItem(page, token, solid(`Task 283 units source ${info.project.name}`, { protein: 9, carbohydrates: 4, fat: 2 }));
	const substitution = async (foodObjectId: string, quantity: number, unit: string) => search(page, {
		query: "Task 283 units",
		mode: "substitution",
		page: 1,
		filters: [],
		substitutionInputs: [{ foodObjectId, foodObjectType: "food_item", quantity, unit }]
	});
	const grams = await substitution(solidItem.value.id, 28.349523125, "g");
	const ounces = await substitution(solidItem.value.id, 1, "oz");
	expect(ounces.sourceSummary?.macros).toEqual(grams.sourceSummary?.macros);
	expect(ounces.sourceSummary?.totalGrams).toBeCloseTo(grams.sourceSummary!.totalGrams, 4);
	const milliliters = await substitution(liquidItem.value.id, 29.5735295625, "ml");
	const fluidOunces = await substitution(liquidItem.value.id, 1, "fl_oz");
	expect(fluidOunces.sourceSummary?.macros).toEqual(milliliters.sourceSummary?.macros);
	expect(fluidOunces.sourceSummary?.totalMilliliters).toBeCloseTo(milliliters.sourceSummary!.totalMilliliters, 4);
	await record(info, "metric-imperial", ["P08-SWR032-STEP-01", "P08-SWR032-STEP-02", "P08-SWR032-STEP-03"], {
		kind: "item", entityId: source.value.id, name: source.value.name, idempotencyKey: source.key,
		requestIds: [await responseRequestId(grams.response), await responseRequestId(ounces.response), await responseRequestId(milliliters.response), await responseRequestId(fluidOunces.response)],
		expected: { active: true, auditActions: { manual_create: 1 }, idempotencyCount: 1 }
	}, ["metric_basis=canonical", "request_correlation=metric_imperial"]);
	const searchNavigation = page.locator("[data-sidebar-nav-search]");
	if (!(await searchNavigation.isVisible())) await page.getByRole("button", { name: "Open activity sidebar" }).click();
	await searchNavigation.click();
	const units = page.locator("#sidebar-unit-system");
	const setUnits = async (value: "metric" | "imperial") => {
		await units.selectOption(value, { force: true });
		const close = page.getByRole("button", { name: "Close activity sidebar" });
		if (await close.isVisible()) await close.click();
	};
	await setUnits("metric");
	await page.getByLabel("Food search").fill(solidItem.value.name);
	await page.getByLabel("Food search").press("Enter");
	const solidCard = page.locator(`[data-result-card][data-result-id="${solidItem.value.id}"]`);
	await expect(solidCard).toBeVisible();
	await expect(solidCard.locator("[data-result-macro-basis]")).toHaveText("values per 100 g");
	await setUnits("imperial");
	await expect(solidCard.locator("[data-result-macro-basis]")).toHaveText("values per 3.5 oz");
	await setUnits("metric");
	await page.getByLabel("Food search").fill(liquidItem.value.name);
	await page.getByLabel("Food search").press("Enter");
	const liquidCard = page.locator(`[data-result-card][data-result-id="${liquidItem.value.id}"]`);
	await expect(liquidCard).toBeVisible();
	await expect(liquidCard.locator("[data-result-macro-basis]")).toHaveText("values per 100 ml");
	await setUnits("imperial");
	await expect(liquidCard.locator("[data-result-macro-basis]")).toHaveText("values per 3.4 fl oz");
});

test("canonical vocabulary, allergen, and classification values persist with exact rollback evidence", async ({ page }, info) => {
	await requireManaged();
	await admin(page, info);
	let token = await csrf(page);
	const createClassification = async (kind: "food_category" | "culinary_role", name: string) => {
		const before = generationSnapshot();
		const response = await page.request.post(`/api/v1/admin/classifications/${kind}`, {
			headers: { "X-CSRF-Token": token }, data: { name }
		});
		expect(response.status()).toBe(201);
		const value = await classification(response);
		const after = generationSnapshot();
		expect(Number(after.value) - Number(before.value)).toBe(1);
		return { response, value, before, after };
	};
	const category = await createClassification("food_category", `Task 283 Task 304 category ${info.project.name}`);
	const role = await createClassification("culinary_role", `Task 283 Task 304 role ${info.project.name}`);
	await record(info, "task304-category", ["P08-SWR057-STEP-01"], {
		kind: "classification", entityId: category.value.id, name: category.value.name,
		requestIds: [await responseRequestId(category.response)],
		expected: { active: true, deleted: false, kind: "food_category", auditActions: { "classification.create": 1 }, generationBefore: category.before.value, generationAfter: category.after.value, generationDelta: 1 },
		generationSnapshots: { generationBefore: category.before.id, generationAfter: category.after.id }
	}, ["mutation_count=1", "audit_count=1", "cache_generation=incremented"]);
	await record(info, "task304-role", ["P08-SWR057-STEP-02"], {
		kind: "classification", entityId: role.value.id, name: role.value.name,
		requestIds: [await responseRequestId(role.response)],
		expected: { active: true, deleted: false, kind: "culinary_role", auditActions: { "classification.create": 1 }, generationBefore: role.before.value, generationAfter: role.after.value, generationDelta: 1 },
		generationSnapshots: { generationBefore: role.before.id, generationAfter: role.after.id }
	}, ["mutation_count=1", "audit_count=1", "cache_generation=incremented"]);

	const vocabularyKey = `Task304Micro${info.project.name.replace(/[^A-Za-z0-9]/g, "")}`;
	const vocabularyBefore = generationSnapshot();
	const vocabularyCreate = await page.request.post("/api/v1/admin/micronutrients", {
		headers: { "X-CSRF-Token": token }, data: { key: vocabularyKey, displayName: "Task 304 Micro", unit: "mg" }
	});
	expect(vocabularyCreate.status()).toBe(201);
	const deactivate = await page.request.post(`${secondAPI()}/api/v1/admin/micronutrients/${vocabularyKey}/deactivate`, {
		headers: { "X-CSRF-Token": await csrf(page, secondAPI()) }
	});
	expect(deactivate.status()).toBe(200);
	const inactive = await deactivate.json() as { data?: { micronutrient?: { active?: boolean } } };
	expect(inactive.data?.micronutrient?.active).toBe(false);
	const reactivate = await page.request.post(`${secondAPI()}/api/v1/admin/micronutrients/${vocabularyKey}/reactivate`, {
		headers: { "X-CSRF-Token": await csrf(page, secondAPI()) }
	});
	expect(reactivate.status()).toBe(200);
	const active = await reactivate.json() as { data?: { micronutrient?: { active?: boolean } } };
	expect(active.data?.micronutrient?.active).toBe(true);
	expect(generationSnapshot().value).toBe(vocabularyBefore.value);
	const inactiveVocabularyKey = `Task304Inactive${info.project.name.replace(/[^A-Za-z0-9]/g, "")}`;
	token = await csrf(page);
	const inactiveVocabularyCreate = await page.request.post("/api/v1/admin/micronutrients", {
		headers: { "X-CSRF-Token": token }, data: { key: inactiveVocabularyKey, displayName: "Task 304 Inactive", unit: "mg" }
	});
	expect(inactiveVocabularyCreate.status()).toBe(201);
	const inactiveVocabularyDeactivate = await page.request.post(`${secondAPI()}/api/v1/admin/micronutrients/${inactiveVocabularyKey}/deactivate`, {
		headers: { "X-CSRF-Token": await csrf(page, secondAPI()) }
	});
	expect(inactiveVocabularyDeactivate.status()).toBe(200);
	const inactiveVocabulary = await inactiveVocabularyDeactivate.json() as { data?: { micronutrient?: { active?: boolean } } };
	expect(inactiveVocabulary.data?.micronutrient?.active).toBe(false);
	expect(generationSnapshot().value).toBe(vocabularyBefore.value);

	const validBefore = generationSnapshot();
	token = await csrf(page);
	const valid = await createItem(page, token, solid(`Task 283 Task 304 canonical ${info.project.name}`, undefined, {
		micros: { Sodium: 125.5, [vocabularyKey]: 7.25 },
		foodCategoryIds: [category.value.id], culinaryRoleIds: [role.value.id], allergenKeys: ["peanut", "dairy"]
	}));
	const validAfter = generationSnapshot();
	expect(Number(validAfter.value) - Number(validBefore.value)).toBe(1);
	expect(valid.value.micros).toEqual({ Sodium: 125.5, [vocabularyKey]: 7.25 });
	expect(valid.value.allergenKeys).toEqual(["dairy", "peanut"]);
	expect(valid.value.foodCategories.map(({ id }) => id)).toEqual([category.value.id]);
	expect(valid.value.culinaryRoles.map(({ id }) => id)).toEqual([role.value.id]);
	const crossInstanceRead = await page.request.get(`${secondAPI()}/api/v1/admin/items/${valid.value.id}`);
	expect(crossInstanceRead.status()).toBe(200);
	expect(await item(crossInstanceRead)).toEqual(valid.value);
	await record(info, "task304-canonical-values", ["P08-SWR090-STEP-01", "P08-SWR090-ACCEPT-01"], {
		kind: "item", entityId: valid.value.id, name: valid.value.name, idempotencyKey: valid.key,
		requestIds: [await responseRequestId(category.response), await responseRequestId(role.response), await responseRequestId(vocabularyCreate), await responseRequestId(deactivate), await responseRequestId(reactivate), await responseRequestId(valid.response)],
		expected: {
			active: true, ownerless: true, auditActions: { manual_create: 1 },
			requestAuditActions: { "classification.create": 2, "micronutrient.create": 1, "micronutrient.deactivate": 1, "micronutrient.reactivate": 1, manual_create: 1 },
			idempotencyCount: 1, micros: { Sodium: 125.5, [vocabularyKey]: 7.25 },
			foodCategoryIds: [category.value.id], culinaryRoleIds: [role.value.id], allergenKeys: ["dairy", "peanut"],
			generationBefore: validBefore.value, generationAfter: validAfter.value, generationDelta: 1
		},
		generationSnapshots: { generationBefore: validBefore.id, generationAfter: validAfter.id }
	}, ["mutation_count=1", "audit_count=1", "row_count=1", "owner_state=global", "cache_generation=incremented"]);

	const invalidInputs = [
		["alias", { Na: 1 }, "P08-SWR090-STEP-02"],
		["unknown", { unknown_key: 1 }, "P08-SWR090-STEP-03"],
		["inactive", { [inactiveVocabularyKey]: 1 }, "P08-SWR090-STEP-04"],
		["allergen", { Sodium: 1 }, "P08-SWR056-ACCEPT-03", { allergenKeys: ["not_in_vocabulary"] }],
		["classification", { Sodium: 1 }, "P08-SWR056-ACCEPT-03", { foodCategoryIds: [role.value.id] }]
	] as const;
	for (const [slug, micros, criterion, extra = {}] of invalidInputs) {
		const name = `Task 283 Task 304 invalid ${slug} ${info.project.name}`;
		const key = crypto.randomUUID();
		const before = generationSnapshot();
		const response = await page.request.post(`${secondAPI()}/api/v1/admin/items`, {
			headers: { "X-CSRF-Token": await csrf(page, secondAPI()), "Idempotency-Key": key },
			data: solid(name, undefined, { micros, ...extra })
		});
		expect(response.status()).toBe(400);
		const after = generationSnapshot();
		expect(after.value).toBe(before.value);
		await record(info, `task304-invalid-${slug}`, [criterion, ...(criterion.startsWith("P08-SWR090") ? ["P08-SWR090-ACCEPT-01"] : [])], {
			kind: "rejected_item", name, idempotencyKey: key, requestIds: [await responseRequestId(response)],
			expected: { rowCount: 0, auditCount: 0, requestAuditActions: {}, idempotencyCount: 0, generationBefore: before.value, generationAfter: after.value, generationDelta: 0 },
			generationSnapshots: { generationBefore: before.id, generationAfter: after.id }
		}, ["http_status=400", "rollback_state=complete", "cache_generation=unchanged"]);
	}
});

test("admin surface remains responsive, keyboard reachable, and reportable", async ({ page }, info) => {
	await requireManaged();
	await admin(page, info);
	await page.getByRole("form", { name: "Classification form" }).getByLabel("Name").first().focus();
	expect(await page.evaluate(() => document.activeElement?.tagName)).toBe("INPUT");
	await page.setViewportSize({ width: 390, height: 844 });
	await page.emulateMedia({ colorScheme: "dark", reducedMotion: "reduce" });
	const shot = await screenshot(page, info, "manual-catalog-responsive", "[data-admin-data-management]");
	await recordAcceptance(info, [], [], [{ type: "playwright", path: shot }], []);
});

for (const criterionId of BLOCKED_CRITERIA) {
	test(`capability blocker: ${criterionId}`, async ({}, info) => {
		await recordAcceptance(info, [criterionId], [], [], [], rootFor(criterionId));
		test.skip(true, `Task 283 capability is unavailable: ${criterionId}`);
	});
}
