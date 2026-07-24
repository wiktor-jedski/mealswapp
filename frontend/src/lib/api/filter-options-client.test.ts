import { afterEach, expect, mock, test } from "bun:test";
import { fetchSubstitutionFilterOptions, FilterOptionsClientError } from "./filter-options-client";

// Implements DESIGN-001 SearchView strict generated dynamic filter-option client verification.

const originalFetch = globalThis.fetch;
afterEach(() => { globalThis.fetch = originalFetch; });

const envelope = (options: unknown[]) => ({ status: "ok", requestId: "task-257", data: { mode: "substitution", options } });
const option = { filterId: "category-id", kind: "food_category", label: "Świeże owoce", includeAllowed: true, excludeAllowed: true, excludes: [] };

async function expectRejected(payload: unknown): Promise<void> {
	globalThis.fetch = mock(async () => new Response(JSON.stringify(payload), { status: 200 })) as typeof fetch;
	const error = await fetchSubstitutionFilterOptions().catch((caught: unknown) => caught);
	expect(error).toBeInstanceOf(FilterOptionsClientError);
	expect(String(error)).not.toContain("database secret");
}

test("loads exact backend labels and policy with cookies from the substitution route", async () => {
	const fetchMock = mock(async (_input: string | URL | Request, init?: RequestInit) => {
		expect(init).toMatchObject({ method: "GET", credentials: "include", headers: { Accept: "application/json" } });
		return new Response(JSON.stringify(envelope([option])), { status: 200 });
	});
	globalThis.fetch = fetchMock as typeof fetch;

	expect(await fetchSubstitutionFilterOptions()).toEqual([option]);
	expect(String(fetchMock.mock.calls[0]?.[0])).toBe("/api/v1/search/filter-options?mode=substitution");
});

test("rejects unsupported food_object_type values in options and nested references", async () => {
	await expectRejected(envelope([{ ...option, kind: "food_object_type" }]));
	await expectRejected(envelope([{ ...option, excludes: [{ filterId: "legacy", kind: "food_object_type" }] }]));
});

test("rejects missing, blank, oversized, and invalid-enum fields", async () => {
	const oversized = "x".repeat(201);
	for (const payload of [
		{ status: "pending", requestId: "task-257", data: { mode: "substitution", options: [] } },
		{ status: "ok", requestId: "task-257", data: { mode: "catalog", options: [] } },
		envelope([{ kind: "food_category", label: "Label", includeAllowed: true, excludeAllowed: true, excludes: [] }]),
		envelope([{ ...option, filterId: "" }]),
		envelope([{ ...option, filterId: oversized }]),
		envelope([{ ...option, label: "" }]),
		envelope([{ ...option, label: oversized }]),
		envelope([{ ...option, labelKey: "" }]),
		envelope([{ ...option, labelKey: oversized }]),
		envelope([{ ...option, includeAllowed: "yes" }]),
		envelope([{ ...option, excludeAllowed: null }]),
		envelope([{ ...option, kind: "frontend_policy" }]),
		envelope([{ ...option, excludes: [{ filterId: "", kind: "allergen" }] }]),
		envelope([{ ...option, excludes: [{ filterId: "allergen", kind: "frontend_policy" }] }]),
		envelope([{ ...option, excludes: null }])
	]) await expectRejected(payload);
});

test("rejects out-of-bounds arrays and unknown fields at every nested level", async () => {
	for (const payload of [
		{ ...envelope([]), sortOrder: 1 },
		{ status: "ok", requestId: "task-257", data: { mode: "substitution", options: [], order: "label" } },
		envelope([{ ...option, sortOrder: 1 }]),
		envelope([{ ...option, excludes: [{ filterId: "milk", kind: "allergen", label: "Milk" }] }]),
		envelope(Array.from({ length: 1001 }, () => option)),
		envelope([{ ...option, excludes: Array.from({ length: 21 }, () => ({ filterId: "milk", kind: "allergen" })) }]),
		{ status: "ok", requestId: "task-257", data: null },
		envelope([null])
	]) await expectRejected(payload);
});

test("rejects unavailable, invalid JSON, and declared or streamed oversized bodies safely", async () => {
	for (const response of [
		new Response(JSON.stringify({ status: "error", error: { message: "database secret" } }), { status: 503 }),
		new Response(JSON.stringify(envelope([option])), { status: 201 }),
		new Response("not-json", { status: 200 }),
		new Response("{}", { status: 200, headers: { "Content-Length": String(32 * 1024 * 1024 + 1) } })
	]) {
		globalThis.fetch = mock(async () => response) as typeof fetch;
		await expect(fetchSubstitutionFilterOptions()).rejects.toBeInstanceOf(FilterOptionsClientError);
	}

	let chunks = 0;
	const body = new ReadableStream<Uint8Array>({
		pull(controller) {
			controller.enqueue(new Uint8Array(1024 * 1024));
			if (++chunks === 33) controller.close();
		}
	});
	globalThis.fetch = mock(async () => new Response(body, { status: 200 })) as typeof fetch;
	await expect(fetchSubstitutionFilterOptions()).rejects.toBeInstanceOf(FilterOptionsClientError);
});

test("preserves aborts during fetch and while reading the response body", async () => {
	const fetchController = new AbortController();
	globalThis.fetch = mock(async (_input, init) => {
		fetchController.abort();
		throw init?.signal?.reason ?? new DOMException("Aborted", "AbortError");
	}) as typeof fetch;
	await expect(fetchSubstitutionFilterOptions(fetchController.signal)).rejects.toHaveProperty("name", "AbortError");

	const bodyController = new AbortController();
	const body = new ReadableStream<Uint8Array>({
		pull(controller) {
			bodyController.abort();
			controller.error(bodyController.signal.reason);
		}
	});
	globalThis.fetch = mock(async () => new Response(body, { status: 200 })) as typeof fetch;
	await expect(fetchSubstitutionFilterOptions(bodyController.signal)).rejects.toHaveProperty("name", "AbortError");
});
