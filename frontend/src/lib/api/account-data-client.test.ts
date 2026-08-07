import { afterEach, expect, mock, test } from "bun:test";
import { AccountDataClientError, deletePrivateCustomItem, loadAccountExport } from "./account-data-client";

// Implements DESIGN-008 DataExporter/ProfileController generated-client verification.

const originalFetch = globalThis.fetch;
afterEach(() => { globalThis.fetch = originalFetch; });

const itemId = "00000000-0000-4000-8000-000000000261";
const objectId = "00000000-0000-4000-8000-000000000262";
const createdAt = "2026-07-28T00:00:00Z";
const exportBundle = {
	user: { userId: itemId, email: "owner@example.test", role: "user", displayName: "Owner", unitSystem: "metric", themePreference: "system" },
	consent: [{ privacyPolicyVersion: "privacy-v1", termsVersion: "terms-v1" }],
	savedItems: [{ id: objectId, itemId, kind: "favorite", createdAt }],
	savedDiets: [{ id: itemId, name: "Portable diet", entries: [{ id: objectId, foodObjectId: itemId, foodObjectType: "food_item", quantity: 100, unit: "g", position: 0 }], createdAt, updatedAt: createdAt }],
	history: [{ id: objectId, query: "tofu", mode: "catalog", filtersHash: "hash", createdAt }],
	customItems: [{ id: itemId, name: "Private tofu", physicalState: "solid", prepTimeMinutes: 0, macrosPer100: { protein: 12, carbohydrates: 3, fat: 6 }, micros: { sodium: 4 }, foodCategories: [{ id: objectId, name: "Legumes", kind: "food_category" }], culinaryRoles: [] }]
};

test("loads the raw generated export and deletes its owner-free custom item with CSRF", async () => {
	const calls: Array<[string, RequestInit | undefined]> = [];
	globalThis.fetch = mock(async (input: string | URL | Request, init?: RequestInit) => {
		calls.push([String(input), init]);
		if (String(input).includes("csrf-token")) return new Response(JSON.stringify({ status: "ok", requestId: "csrf", data: { csrfToken: "csrf-261" } }), { status: 200 });
		if (init?.method === "DELETE") return new Response(null, { status: 204 });
		return new Response(JSON.stringify(exportBundle), { status: 200 });
	}) as typeof fetch;

	expect(await loadAccountExport()).toEqual(exportBundle);
	await deletePrivateCustomItem(itemId);
	expect(calls[0]).toMatchObject(["/api/v1/account/export?format=json", { method: "GET", credentials: "include", headers: { Accept: "application/json" } }]);
	expect(calls[2]).toMatchObject([`/api/v1/custom-items/${itemId}`, { method: "DELETE", credentials: "include", headers: { "X-CSRF-Token": "csrf-261" } }]);
});

test("rejects ownership leakage, malformed identifiers, oversized exports, and non-empty deletes", async () => {
	const customDietExport = {
		...exportBundle,
		savedDiets: [{ ...exportBundle.savedDiets[0], entries: [{ ...exportBundle.savedDiets[0].entries[0], foodObjectType: "custom_food_item" }] }]
	};
	globalThis.fetch = mock(async () => new Response(JSON.stringify(customDietExport), { status: 200 })) as typeof fetch;
	await expect(loadAccountExport()).resolves.toEqual(customDietExport);

	for (const leaking of [
		{ ...exportBundle, customItems: [{ ...exportBundle.customItems[0], ownerId: itemId }] },
		{ ...exportBundle, savedItems: [{ ...exportBundle.savedItems[0], UserID: itemId }] },
		{ ...exportBundle, savedDiets: [{ ...exportBundle.savedDiets[0], entries: [{ ...exportBundle.savedDiets[0].entries[0], owner_id: itemId }] }] },
		{ ...exportBundle, customItems: [{ ...exportBundle.customItems[0], macrosPer100: { ...exportBundle.customItems[0].macrosPer100, owner: itemId } }] }
	]) {
		globalThis.fetch = mock(async () => new Response(JSON.stringify(leaking), { status: 200 })) as typeof fetch;
		await expect(loadAccountExport()).rejects.toBeInstanceOf(AccountDataClientError);
	}
	for (const [key, valid] of [["", false], ["x".repeat(121), false], ["😀".repeat(120), true], ["😀".repeat(121), false]] as const) {
		const invalidMicronutrientKey = { ...exportBundle, customItems: [{ ...exportBundle.customItems[0], micros: { [key]: 4 } }] };
		globalThis.fetch = mock(async () => new Response(JSON.stringify(invalidMicronutrientKey), { status: 200 })) as typeof fetch;
		if (valid) await expect(loadAccountExport()).resolves.toEqual(invalidMicronutrientKey);
		else await expect(loadAccountExport()).rejects.toBeInstanceOf(AccountDataClientError);
	}
	globalThis.fetch = mock(async () => new Response(JSON.stringify({ ...exportBundle, savedDiets: undefined }), { status: 200 })) as typeof fetch;
	await expect(loadAccountExport()).rejects.toBeInstanceOf(AccountDataClientError);
	await expect(deletePrivateCustomItem("not-a-uuid")).rejects.toBeInstanceOf(AccountDataClientError);

	globalThis.fetch = mock(async () => new Response("{}", { status: 200, headers: { "Content-Length": String(1024 * 1024 + 1) } })) as typeof fetch;
	await expect(loadAccountExport()).rejects.toBeInstanceOf(AccountDataClientError);
	globalThis.fetch = mock(async () => { throw new Error("network failure"); }) as typeof fetch;
	await expect(loadAccountExport()).rejects.toBeInstanceOf(AccountDataClientError);

	let call = 0;
	globalThis.fetch = mock(async () => ++call === 1
		? new Response(JSON.stringify({ status: "ok", requestId: "csrf", data: { csrfToken: "csrf-261" } }), { status: 200 })
		: new Response(JSON.stringify({
			status: "error", requestId: "task-298", error: {
				category: "validation", code: "custom_item_in_use", message: "in use", retryable: false,
				data: { affectedDiets: [{ id: objectId, name: "Portable diet" }] }
			}
		}), { status: 409 })) as typeof fetch;
	await expect(deletePrivateCustomItem(itemId)).rejects.toMatchObject({
		message: "Remove this item from the listed saved diets before permanent deletion.",
		affectedDiets: [{ id: objectId, name: "Portable diet" }]
	});
});

// Implements DESIGN-008 AccountDeleter adversarial 409 conflict decoding verification.
test("rejects malformed, oversized, and unsafe custom-item deletion conflict responses", async () => {
	const conflict = (): Record<string, unknown> => ({
		status: "error",
		requestId: "task-298",
		error: {
			category: "validation",
			code: "custom_item_in_use",
			message: "in use",
			retryable: false,
			data: { affectedDiets: [{ id: objectId, name: "Portable diet" }] }
		}
	});
	const edited = (edit: (body: Record<string, unknown>) => void): Record<string, unknown> => {
		const body = conflict();
		edit(body);
		return body;
	};
	const cases: Array<[string, number, unknown]> = [
		["wrong status", 404, conflict()],
		["invalid JSON", 409, "{"],
		["missing top-level field", 409, edited((body) => { delete body.status; })],
		["extra top-level field", 409, edited((body) => { body.extra = true; })],
		["missing error field", 409, edited((body) => { delete body.error; })],
		["extra error field", 409, edited((body) => { (body.error as Record<string, unknown>).requestId = "leak"; })],
		["wrong category", 409, edited((body) => { (body.error as Record<string, unknown>).category = "conflict"; })],
		["wrong code", 409, edited((body) => { (body.error as Record<string, unknown>).code = "not_found"; })],
		["retryable conflict", 409, edited((body) => { (body.error as Record<string, unknown>).retryable = true; })],
		["missing data", 409, edited((body) => { delete (body.error as Record<string, unknown>).data; })],
		["extra data field", 409, edited((body) => { (body.error as Record<string, unknown>).data = { affectedDiets: [{ id: objectId, name: "Portable diet" }], ownerId: itemId }; })],
		["malformed diet id", 409, edited((body) => { ((body.error as Record<string, unknown>).data as Record<string, unknown>).affectedDiets = [{ id: "not-a-uuid", name: "Portable diet" }]; })],
		["missing diet name", 409, edited((body) => { ((body.error as Record<string, unknown>).data as Record<string, unknown>).affectedDiets = [{ id: objectId }]; })],
		["extra diet field", 409, edited((body) => { ((body.error as Record<string, unknown>).data as Record<string, unknown>).affectedDiets = [{ id: objectId, name: "Portable diet", ownerId: itemId }]; })],
		["oversized diet name", 409, edited((body) => { ((body.error as Record<string, unknown>).data as Record<string, unknown>).affectedDiets = [{ id: objectId, name: "x".repeat(201) }]; })],
		["unsafe diet name", 409, edited((body) => { ((body.error as Record<string, unknown>).data as Record<string, unknown>).affectedDiets = [{ id: objectId, name: "Portable\u0000diet" }]; })],
		["too many diets", 409, edited((body) => { ((body.error as Record<string, unknown>).data as Record<string, unknown>).affectedDiets = Array.from({ length: 26 }, () => ({ id: objectId, name: "Portable diet" })); })],
		["unsafe request id", 409, edited((body) => { body.requestId = "task-298\nleak"; })]
	];

	for (const [label, status, body] of cases) {
		let call = 0;
		globalThis.fetch = mock(async () => ++call === 1
			? new Response(JSON.stringify({ status: "ok", requestId: "csrf", data: { csrfToken: "csrf-298" } }), { status: 200 })
			: new Response(typeof body === "string" ? body : JSON.stringify(body), { status })) as typeof fetch;
		const error = await deletePrivateCustomItem(itemId).catch((cause: unknown) => cause);
		expect(error, label).toBeInstanceOf(AccountDataClientError);
		expect(error, label).toMatchObject({ message: "The private item could not be deleted. Try again." });
		expect((error as AccountDataClientError).affectedDiets, label).toBeUndefined();
	}
});
