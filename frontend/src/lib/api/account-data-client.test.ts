import { afterEach, expect, mock, test } from "bun:test";
import { AccountDataClientError, deletePrivateCustomItem, loadAccountExport } from "./account-data-client";

// Implements DESIGN-008 DataExporter/ProfileController generated-client verification.

const originalFetch = globalThis.fetch;
afterEach(() => { globalThis.fetch = originalFetch; });

const itemId = "00000000-0000-4000-8000-000000000261";
const exportBundle = { user: {}, consent: [], savedItems: [], history: [], customItems: [{ id: itemId, name: "Private tofu" }] };

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
	expect(calls[0]).toMatchObject(["/api/v1/account/export?format=json", { method: "GET", credentials: "include" }]);
	expect(calls[2]).toMatchObject([`/api/v1/custom-items/${itemId}`, { method: "DELETE", credentials: "include", headers: { "X-CSRF-Token": "csrf-261" } }]);
});

test("rejects ownership leakage, malformed identifiers, oversized exports, and non-empty deletes", async () => {
	globalThis.fetch = mock(async () => new Response(JSON.stringify({ ...exportBundle, customItems: [{ ...exportBundle.customItems[0], ownerId: itemId }] }), { status: 200 })) as typeof fetch;
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
