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

	let call = 0;
	globalThis.fetch = mock(async () => ++call === 1
		? new Response(JSON.stringify({ status: "ok", requestId: "csrf", data: { csrfToken: "csrf-261" } }), { status: 200 })
		: new Response(JSON.stringify({ status: "ok" }), { status: 200 })) as typeof fetch;
	await expect(deletePrivateCustomItem(itemId)).rejects.toBeInstanceOf(AccountDataClientError);
});
