import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { expect, test, type APIResponse, type BrowserContext, type Page, type TestInfo } from "@playwright/test";
import {
	ACCOUNT_ENDPOINT,
	CUSTOM_ITEMS_ENDPOINT,
	DAILY_DIETS_ENDPOINT,
	PROFILE_ENDPOINT,
	buildAccountDeletionRequestInit,
	buildAccountExportRequestInit,
	buildAccountExportUrl,
	buildCustomItemDeleteRequestInit,
	buildCustomItemMutationRequestInit,
	buildCustomItemUrl,
	buildDailyDietCreateRequestInit
} from "../src/lib/api/generated";
import type { CustomItemRequest, DailyDietCreateRequest, ExportBundle } from "../src/lib/api/generated";
import { fixture, openSidebarForControl, recordAcceptance, responseRequestId, screenshot } from "./task281-acceptance-helpers";

// Implements DESIGN-008 AccountDeleter, DataExporter, and owner-scoped ProfileController acceptance.
const enabled = process.env.MEALSWAPP_TASK284_REAL_E2E === "1" && process.env.MEALSWAPP_REAL_STACK_MANAGED === "1";
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const EXPORT_OWNER_ROOT = "ROOT-T284-EXPORT-OWNER-PROJECTION";

test.skip(!enabled, "Run scripts/run-task284-acceptance.py for isolated real-stack evidence.");

interface FixtureState {
	aItem: string;
	aDeletedItem: string;
	bItem: string;
	globalItem: string;
	diet: string;
	deletionRequest?: string;
	beforeSnapshot?: Snapshot;
}

interface Snapshot {
	schema: "mealswapp.task284-snapshot.v1";
	phase: "before" | "after";
	a: Record<string, number | boolean | string | null>;
	b: Record<string, number | boolean | string | null>;
	global: Record<string, number | boolean | string | null>;
	cache: { a: number; b: number };
}

type CSVRow = [section: string, field: string, value: string];

const state: FixtureState = { aItem: "", aDeletedItem: "", bItem: "", globalItem: "", diet: "" };

function statePath(): string {
	return join(fixture("MEALSWAPP_TASK284_PROOF_REQUEST_DIR"), "scenario-state.json");
}

function saveState(): void {
	writeFileSync(statePath(), `${JSON.stringify(state)}\n`, { mode: 0o600 });
}

function loadState(): void {
	Object.assign(state, JSON.parse(readFileSync(statePath(), "utf8")) as FixtureState);
}

function privateItem(name: string): CustomItemRequest {
	return {
		name,
		physicalState: "solid",
		prepTimeMinutes: 0,
		macrosPer100: { protein: 12, carbohydrates: 32, fat: 8 },
		micros: {},
		foodCategoryIds: [],
		culinaryRoleIds: []
	};
}

async function signInAs(page: Page, email: string, password: string): Promise<void> {
	await page.goto("/");
	await openSidebarForControl(page, page.getByRole("button", { name: "Sign in", exact: true }));
	await page.getByRole("button", { name: "Sign in", exact: true }).click();
	await page.locator("[data-login-view]").getByLabel("Email").fill(email);
	await page.locator("[data-login-view]").getByLabel("Password").fill(password);
	await page.locator("[data-login-view]").getByRole("button", { name: "Sign in" }).click();
	await expect(page.locator("[data-login-view]")).toHaveCount(0);
}

async function csrf(page: Page): Promise<string> {
	const response = await page.request.get("/api/v1/auth/csrf-token");
	expect(response.status()).toBe(200);
	const body = await response.json() as { data?: { csrfToken?: string } };
	expect(body.data?.csrfToken).toBeTruthy();
	return body.data!.csrfToken!;
}

async function createCustom(page: Page, token: string, name: string): Promise<{ id: string; response: APIResponse }> {
	const generated = buildCustomItemMutationRequestInit(
		"POST",
		privateItem(name),
		token,
		{ idempotencyKey: crypto.randomUUID() }
	);
	const response = await page.request.fetch(CUSTOM_ITEMS_ENDPOINT, {
		method: generated.method,
		headers: generated.headers,
		data: generated.body
	});
	const text = await response.text();
	expect(response.status(), text).toBe(201);
	const body = JSON.parse(text) as { data?: { id?: string } };
	expect(body.data?.id).toMatch(UUID);
	return { id: body.data!.id!, response };
}

async function createGlobal(page: Page, token: string, name = "Task 284 global survivor"): Promise<{ id: string; response: APIResponse }> {
	const response = await page.request.post("/api/v1/admin/items", {
		headers: { "X-CSRF-Token": token, "Idempotency-Key": crypto.randomUUID() },
		data: { ...privateItem(name), allergenKeys: [] }
	});
	expect(response.status()).toBe(201);
	const body = await response.json() as { data?: { id?: string } };
	expect(body.data?.id).toMatch(UUID);
	return { id: body.data!.id!, response };
}

async function exportJSON(page: Page): Promise<ExportBundle> {
	const response = await page.request.fetch(buildAccountExportUrl("json"), buildAccountExportRequestInit("json"));
	expect(response.status()).toBe(200);
	return await response.json() as ExportBundle;
}

function parseCSV(source: string): CSVRow[] {
	const rows: string[][] = [];
	let row: string[] = [];
	let field = "";
	let quoted = false;
	for (let index = 0; index < source.length; index += 1) {
		const character = source[index];
		if (quoted) {
			if (character === '"' && source[index + 1] === '"') {
				field += '"';
				index += 1;
			} else if (character === '"') {
				quoted = false;
			} else {
				field += character;
			}
		} else if (character === '"' && field === "") {
			quoted = true;
		} else if (character === ",") {
			row.push(field);
			field = "";
		} else if (character === "\n") {
			row.push(field);
			rows.push(row);
			row = [];
			field = "";
		} else if (character !== "\r") {
			field += character;
		}
	}
	if (quoted) throw new Error("CSV ended inside a quoted field");
	if (field !== "" || row.length > 0) {
		row.push(field);
		rows.push(row);
	}
	if (rows.some((value) => value.length !== 3)) throw new Error("CSV row does not contain exactly three fields");
	return rows as CSVRow[];
}

function assertExportCSV(rows: CSVRow[], expected: {
	customItems: string[];
	diet?: string;
	dietSource?: string;
	forbidden: string[];
}): void {
	expect(rows[0]).toEqual(["section", "field", "value"]);
	const body = rows.slice(1);
	const allowedSections = new Set(["user", "savedItems", "savedDiets", "savedDietEntries", "history", "consent", "customItems"]);
	expect(body.every(([section]) => allowedSections.has(section))).toBe(true);
	expect(body.filter(([section]) => section === "user").map(([, field]) => field)).toEqual([
		"userId", "email", "displayName", "unitSystem", "themePreference"
	]);
	const customRows = body.filter(([section]) => section === "customItems");
	expect(customRows.map(([, field]) => field).sort()).toEqual([...expected.customItems].sort());
	for (const [, , value] of customRows) {
		const customProjection = JSON.parse(value) as unknown;
		expect([...collectKeys(customProjection)].some((key) => /^(owner|ownerId)$/i.test(key))).toBe(false);
	}
	if (expected.diet) {
		expect(body.filter(([section]) => section === "savedDiets")).toEqual([
			["savedDiets", "Task 284 portable diet", expected.diet]
		]);
		expect(body.filter(([section]) => section === "savedDietEntries")).toEqual([
			["savedDietEntries", "food_item", expected.dietSource]
		]);
	} else {
		expect(body.filter(([section]) => section === "savedDiets" || section === "savedDietEntries")).toEqual([]);
	}
	for (const forbidden of expected.forbidden) {
		expect(body.some((cells) => cells.some((cell) => cell.includes(forbidden)))).toBe(false);
	}
}

function collectKeys(value: unknown, keys: Set<string> = new Set()): Set<string> {
	if (!value || typeof value !== "object") return keys;
	for (const [key, child] of Object.entries(value)) {
		keys.add(key);
		if (child && typeof child === "object") collectKeys(child, keys);
	}
	return keys;
}

async function snapshot(phase: "before" | "after"): Promise<Snapshot> {
	const directory = fixture("MEALSWAPP_TASK284_PROOF_REQUEST_DIR");
	const id = crypto.randomUUID();
	const request = join(directory, `${id}.request.json`);
	const response = join(directory, `${id}.response.json`);
	writeFileSync(request, `${JSON.stringify({
		schema: "mealswapp.task284-snapshot-request.v2",
		id,
		capabilityNonce: fixture("MEALSWAPP_TASK284_CAPABILITY_NONCE"),
		phase,
		userA: fixture("MEALSWAPP_TASK284_USER_A_ID"),
		userB: fixture("MEALSWAPP_TASK284_USER_B_ID"),
		itemA: state.aItem,
		deletedItemA: state.aDeletedItem,
		itemB: state.bItem,
		globalItem: state.globalItem,
		deletionRequest: state.deletionRequest ?? null
	})}\n`, { mode: 0o600 });
	const deadline = Date.now() + 15_000;
	while (!existsSync(response) && Date.now() < deadline) {
		await new Promise((resolve) => setTimeout(resolve, 50));
	}
	if (!existsSync(response)) throw new Error("Task 284 read-only snapshot timed out");
	const value = JSON.parse(readFileSync(response, "utf8")) as Snapshot & { detail?: string };
	if (value.schema !== "mealswapp.task284-snapshot.v1") throw new Error(value.detail ?? "Task 284 snapshot failed");
	expect(value.schema).toBe("mealswapp.task284-snapshot.v1");
	expect(value.phase).toBe(phase);
	return value;
}

async function record(
	info: TestInfo,
	criteria: string[],
	requests: APIResponse[],
	evidence: Array<{ type: "playwright" | "backend"; path: string }>,
	summaries: string[]
): Promise<void> {
	await recordAcceptance(
		info,
		criteria,
		await Promise.all(requests.map(responseRequestId)),
		evidence,
		summaries
	);
}

test("users A and B retain indistinguishable owner isolation and owner-only JSON/CSV portability", async ({ browser, page }, info) => {
	await signInAs(page, fixture("MEALSWAPP_TASK284_USER_A_EMAIL"), fixture("MEALSWAPP_TASK284_USER_A_PASSWORD"));
	const aToken = await csrf(page);
	const a = await createCustom(page, aToken, "Task 284 owner A retained");
	const selected = await createCustom(page, aToken, "Task 284 owner A selected deletion");
	const bContext: BrowserContext = await browser.newContext();
	const bPage = await bContext.newPage();
	await signInAs(bPage, fixture("MEALSWAPP_TASK284_USER_B_EMAIL"), fixture("MEALSWAPP_TASK284_USER_B_PASSWORD"));
	const bToken = await csrf(bPage);
	const b = await createCustom(bPage, bToken, "Task 284 owner B survivor");
	const global = await createGlobal(bPage, bToken);
	const dietSource = await createGlobal(bPage, bToken, "Task 284 global diet source");
	state.aItem = a.id;
	state.aDeletedItem = selected.id;
	state.globalItem = global.id;
	state.bItem = b.id;

	const dietRequest: DailyDietCreateRequest = {
		name: "Task 284 portable diet",
		entries: [{ foodObjectId: dietSource.id, foodObjectType: "food_item", quantity: 100, unit: "g", position: 0 }]
	};
	const generatedDiet = buildDailyDietCreateRequestInit(
		dietRequest,
		crypto.randomUUID(),
		{ csrfToken: aToken }
	);
	const dietResponse = await page.request.fetch(DAILY_DIETS_ENDPOINT, {
		method: generatedDiet.method,
		headers: generatedDiet.headers,
		data: generatedDiet.body
	});
	expect(dietResponse.status()).toBe(201);
	const dietBody = await dietResponse.json() as { data?: { id?: string } };
	expect(dietBody.data?.id).toMatch(UUID);
	state.diet = dietBody.data!.id!;

	const historyResponse = await page.request.post("/api/v1/search", {
		data: { query: "Task 284 owner A retained", mode: "catalog", filters: [], page: 1 }
	});
	expect(historyResponse.status()).toBe(200);

	const crossRead = await bPage.request.get(buildCustomItemUrl(a.id));
	const generatedCrossUpdate = buildCustomItemMutationRequestInit(
		"PUT",
		privateItem("Task 284 forbidden replacement"),
		bToken
	);
	const crossUpdate = await bPage.request.fetch(buildCustomItemUrl(a.id), {
		method: generatedCrossUpdate.method,
		headers: generatedCrossUpdate.headers,
		data: generatedCrossUpdate.body
	});
	const crossDelete = await bPage.request.fetch(buildCustomItemUrl(a.id), buildCustomItemDeleteRequestInit(bToken));
	const guessed = await bPage.request.get(buildCustomItemUrl(crypto.randomUUID()));
	for (const response of [crossRead, crossUpdate, crossDelete, guessed]) expect(response.status()).toBe(404);
	const denialBodies = await Promise.all([crossRead, crossUpdate, crossDelete, guessed].map(async (response) => {
		const body = await response.json() as { error?: { category?: string; code?: string; message?: string; retryable?: boolean } };
		return {
			category: body.error?.category,
			code: body.error?.code,
			message: body.error?.message,
			retryable: body.error?.retryable
		};
	}));
	expect(new Set(denialBodies.map((body) => JSON.stringify(body))).size).toBe(1);

	const aJSON = await exportJSON(page);
	const bJSON = await exportJSON(bPage);
	expect(aJSON.customItems.map((item) => (item as { ID?: string; id?: string }).ID ?? (item as { id?: string }).id).sort()).toEqual([a.id, selected.id].sort());
	expect(aJSON.savedDiets).toHaveLength(1);
	expect(aJSON.savedDiets[0]).toMatchObject({
		id: state.diet,
		name: "Task 284 portable diet",
		entries: [{ foodObjectId: dietSource.id, foodObjectType: "food_item", quantity: 100, unit: "g", position: 0 }]
	});
	expect(collectKeys(aJSON.savedDiets).has("UserID")).toBe(false);
	expect(bJSON.savedDiets).toEqual([]);
	expect(JSON.stringify(aJSON)).not.toContain(b.id);
	expect(JSON.stringify(aJSON)).not.toContain(global.id);
	expect(JSON.stringify(bJSON)).toContain(b.id);
	expect(JSON.stringify(bJSON)).not.toContain(a.id);
	expect(JSON.stringify(bJSON)).not.toContain(global.id);
	const aCSVResponse = await page.request.fetch(buildAccountExportUrl("csv"), buildAccountExportRequestInit("csv"));
	const bCSVResponse = await bPage.request.fetch(buildAccountExportUrl("csv"), buildAccountExportRequestInit("csv"));
	expect(aCSVResponse.status()).toBe(200);
	expect(bCSVResponse.status()).toBe(200);
	const aCSV = parseCSV(await aCSVResponse.text());
	const bCSV = parseCSV(await bCSVResponse.text());
	assertExportCSV(aCSV, {
		customItems: [a.id, selected.id],
		diet: state.diet,
		dietSource: dietSource.id,
		forbidden: [b.id, global.id]
	});
	assertExportCSV(bCSV, { customItems: [b.id], forbidden: [a.id, selected.id, global.id] });

	const selectedDelete = await page.request.fetch(buildCustomItemUrl(selected.id), buildCustomItemDeleteRequestInit(aToken));
	expect(selectedDelete.status()).toBe(204);
	expect(JSON.stringify(await exportJSON(page))).not.toContain(selected.id);

	await bPage.goto("/");
	const administration = bPage.getByRole("button", { name: "Administration" });
	await openSidebarForControl(bPage, administration);
	await administration.focus();
	await bPage.keyboard.press("Enter");
	await expect(bPage.locator("[data-admin-private-data]")).toBeVisible();
	await expect(bPage.locator("[data-admin-private-data] h2")).toHaveAccessibleName("Current admin private data");
	const shot = await screenshot(bPage, info, "private-data-heading", "[data-admin-private-data] h2");

	state.beforeSnapshot = await snapshot("before");
	expect(state.beforeSnapshot.a.customItems).toBe(1);
	expect(state.beforeSnapshot.a.customItemsRetained).toBe(2);
	expect(state.beforeSnapshot.b.customItems).toBe(1);
	expect(state.beforeSnapshot.a.savedDiets).toBeGreaterThanOrEqual(1);
	expect(state.beforeSnapshot.a.savedDietEntries).toBe(1);
	expect(state.beforeSnapshot.a.history).toBeGreaterThanOrEqual(1);
	saveState();

	await record(
		info,
		[
			"P08-SWR043-STEP-01", "P08-SWR043-STEP-02", "P08-SWR043-STEP-03",
			"P08-SWR043-STEP-04",
			"P08-SWR072-STEP-01", "P08-SWR072-STEP-02", "P08-SWR072-STEP-03",
			"P08-SWR072-STEP-05", "P08-SWR072-STEP-06"
		],
		[a.response, b.response, global.response, dietSource.response, dietResponse, historyResponse, crossRead, crossUpdate, crossDelete, guessed],
		[{ type: "playwright", path: shot }, { type: "backend", path: "backend/task284-proof.json" }],
		["owner_state=isolated", "export_record_count=owner_only", "http_status=404", "request_correlation=verified"]
	);
	await bContext.close();
});

test("P08-SWR043-ACCEPT-01 P08-SWR072-STEP-04 export projections omit nested owner identity", async ({ page }, info) => {
	loadState();
	await signInAs(page, fixture("MEALSWAPP_TASK284_USER_A_EMAIL"), fixture("MEALSWAPP_TASK284_USER_A_PASSWORD"));
	const exported = await exportJSON(page);
	const csvResponse = await page.request.fetch(buildAccountExportUrl("csv"), buildAccountExportRequestInit("csv"));
	expect(csvResponse.status()).toBe(200);
	const csvRows = parseCSV(await csvResponse.text());
	const csvProjectionKeys = new Set<string>();
	for (const [section, , value] of csvRows.slice(1)) {
		if (section === "customItems") collectKeys(JSON.parse(value) as unknown, csvProjectionKeys);
	}
	await recordAcceptance(
		info,
		["P08-SWR043-ACCEPT-01", "P08-SWR072-STEP-04"],
		[],
		[{ type: "backend", path: "backend/task284-proof.json" }],
		["owner_state=projection_leak"],
		EXPORT_OWNER_ROOT
	);
	expect(
		[...collectKeys(exported), ...csvProjectionKeys].some((key) => /^(owner|ownerId|UserID)$/i.test(key)),
		"owner identity must not leak from nested export projections"
	).toBeFalsy();
});

test("real deletion worker locks writes, erases A, denies stale access, and preserves B/global state", async ({ page }, info) => {
	loadState();
	await signInAs(page, fixture("MEALSWAPP_TASK284_USER_A_EMAIL"), fixture("MEALSWAPP_TASK284_USER_A_PASSWORD"));
	const staleState = await page.context().storageState();
	const token = await csrf(page);
	const deletion = await page.request.fetch(ACCOUNT_ENDPOINT, buildAccountDeletionRequestInit(token));
	expect(deletion.status()).toBe(200);
	const deletionBody = await deletion.json() as { data?: { requestId?: string; status?: string } };
	expect(deletionBody.data?.requestId).toMatch(UUID);
	expect(deletionBody.data?.status).toBe("pending");
	state.deletionRequest = deletionBody.data!.requestId!;
	const pending = await snapshot("after");
	expect(pending.a.deletionStatus).toMatch(/^(pending|processing)$/);
	const repeatedDeletion = await page.request.fetch(ACCOUNT_ENDPOINT, buildAccountDeletionRequestInit(token));
	expect(repeatedDeletion.status()).toBe(401);

	const staleContext = await page.context().browser()!.newContext({ storageState: staleState });
	const stalePage = await staleContext.newPage();
	const pendingTokenResponse = await stalePage.request.get("/api/v1/auth/csrf-token");
	const pendingTokenBody = await pendingTokenResponse.json() as { data?: { csrfToken?: string } };
	const generatedPendingWrite = buildCustomItemMutationRequestInit(
		"POST",
		privateItem("Task 284 pending write"),
		pendingTokenBody.data?.csrfToken ?? token,
		{ idempotencyKey: crypto.randomUUID() }
	);
	const pendingWrite = await stalePage.request.fetch(CUSTOM_ITEMS_ENDPOINT, {
		method: generatedPendingWrite.method,
		headers: generatedPendingWrite.headers,
		data: generatedPendingWrite.body
	});
	expect(pendingWrite.status()).toBe(401);

	let after: Snapshot | undefined;
	const deadline = Date.now() + 50_000;
	while (Date.now() < deadline) {
		after = await snapshot("after");
		if (after.a.deletionStatus === "completed") break;
		await new Promise((resolve) => setTimeout(resolve, 500));
	}
	expect(after?.a.deletionStatus).toBe("completed");
	expect(after?.a.userExists).toBe(false);
	expect(after?.a.customItems).toBe(0);
	expect(after?.a.customItemsRetained).toBe(0);
	expect(after?.a.history).toBe(0);
	expect(after?.a.savedItems).toBe(0);
	expect(after?.a.savedDiets).toBe(0);
	expect(after?.a.sessions).toBe(0);
	expect(after?.a.oauthIdentities).toBe(0);
	expect(after?.a.passwordResetTokens).toBe(0);
	expect(after?.a.profiles).toBe(0);
	expect(after?.a.consent).toBe(0);
	expect(after?.a.entitlements).toBe(0);
	expect(after?.a.usageWindows).toBe(0);
	expect(after?.a.mutationIdempotency).toBe(0);
	expect(after?.a.savedDietEntries).toBe(0);
	expect(after?.a.customItemClassifications).toBe(0);
	expect(after?.a.receiptPresent).toBe(true);
	expect(after?.a.receiptPseudonymous).toBe(true);
	expect(after?.cache.a).toBe(0);
	expect(after?.b).toEqual(state.beforeSnapshot!.b);
	expect(after?.global).toEqual(state.beforeSnapshot!.global);
	expect(after?.cache.b).toBe(state.beforeSnapshot!.cache.b);

	for (const response of [
		await stalePage.request.get(PROFILE_ENDPOINT),
		await stalePage.request.fetch(buildAccountExportUrl("json"), buildAccountExportRequestInit("json")),
		await stalePage.request.get(buildCustomItemUrl(state.aItem))
	]) expect(response.status()).toBe(401);

	await page.goto("/");
	await openSidebarForControl(page, page.getByRole("button", { name: "Sign in", exact: true }));
	await page.getByRole("button", { name: "Sign in", exact: true }).click();
	await page.locator("[data-login-view]").getByLabel("Email").fill(fixture("MEALSWAPP_TASK284_USER_A_EMAIL"));
	await page.locator("[data-login-view]").getByLabel("Password").fill(fixture("MEALSWAPP_TASK284_USER_A_PASSWORD"));
	await page.locator("[data-login-view]").getByRole("button", { name: "Sign in" }).click();
	await expect(page.locator("[data-login-view] [role=alert]")).toBeVisible();

	const bContext = await page.context().browser()!.newContext();
	const bPage = await bContext.newPage();
	await signInAs(bPage, fixture("MEALSWAPP_TASK284_USER_B_EMAIL"), fixture("MEALSWAPP_TASK284_USER_B_PASSWORD"));
	expect((await bPage.request.get(buildCustomItemUrl(state.bItem))).status()).toBe(200);
	expect((await bPage.request.get(`/api/v1/food-objects/${state.globalItem}`)).status()).toBe(200);

	await record(
		info,
		[
			"P08-SWR073-STEP-01", "P08-SWR073-STEP-02", "P08-SWR073-STEP-03",
			"P08-SWR073-STEP-04", "P08-SWR073-STEP-05", "P08-SWR073-STEP-06"
		],
		[deletion, repeatedDeletion, pendingWrite],
		[{ type: "backend", path: "backend/task284-proof.json" }],
		["worker_state=completed", "owner_state=erased", "cache_generation=owner_purged", "rollback_state=survivors_unchanged"]
	);
	await bContext.close();
	await staleContext.close();
});
