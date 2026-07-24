import { afterEach, expect, mock, test } from "bun:test";
import {
	ExternalAdminClientError,
	createImportIdempotencyKey,
	importCuratedItem,
	loadAdminClassifications,
	searchExternalFoods
} from "./external-admin-client";
import type { CuratedImportRequest } from "./generated";

// Implements DESIGN-009 ExternalSearchProxy and DataImporter client contract verification.

const originalFetch = globalThis.fetch;
afterEach(() => { globalThis.fetch = originalFetch; });

function response(status: number, body: unknown, headers?: Record<string, string>): Response {
	return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json", ...headers } });
}

function abortingBodyResponse(status: number, abortError: DOMException): { response: Response; cancel: ReturnType<typeof mock>; releaseLock: ReturnType<typeof mock> } {
	const cancel = mock(async () => undefined);
	const releaseLock = mock(() => undefined);
	return {
		response: {
			status,
			ok: status >= 200 && status < 300,
			headers: new Headers({ "Content-Type": "application/json" }),
			body: { getReader: () => ({ read: async () => { throw abortError; }, cancel, releaseLock }) }
		} as unknown as Response,
		cancel,
		releaseLock
	};
}

function cancellableResponse(status: number, headers: Record<string, string> = {}): { response: Response; cancel: ReturnType<typeof mock> } {
	const cancel = mock(async () => undefined);
	return {
		response: {
			status,
			ok: status >= 200 && status < 300,
			headers: new Headers(headers),
			body: { cancel }
		} as unknown as Response,
		cancel
	};
}

const draft: CuratedImportRequest = {
	sourceProvider: "usda",
	externalId: "100",
	name: "Apple",
	physicalState: "solid",
	macrosPer100: { protein: 1, carbohydrates: 20, fat: 0 },
	micros: {},
	foodCategoryIds: ["fruit"],
	culinaryRoleIds: []
};

test("searches USDA, OpenFoodFacts, and combined providers with bounded page parameters", async () => {
	for (const provider of ["usda", "openfoodfacts", "all"] as const) {
		const fetchMock = mock(async (input: string | URL | Request) => {
			expect(String(input)).toBe(`/api/v1/admin/external-search?query=green+apple&provider=${provider}&page=2`);
			return response(200, { status: "ok", requestId: "search", data: { candidates: [], warnings: [], page: 2 } });
		});
		globalThis.fetch = fetchMock as typeof fetch;
		expect((await searchExternalFoods(" green apple ", provider, 2)).page).toBe(2);
		expect(fetchMock).toHaveBeenCalledTimes(1);
	}
});

test("loads generated classification collections", async () => {
	globalThis.fetch = mock(async (input: string | URL | Request) => response(200, {
		status: "ok",
		requestId: "classes",
		data: { classifications: [{ id: "10000000-0000-4000-8000-000000000001", name: "Fruit", kind: "food_category" }] }
	})) as typeof fetch;
	expect((await loadAdminClassifications("food_category"))[0]?.name).toBe("Fruit");
});

test("imports with cookies, CSRF, generated DTO JSON, and the unchanged caller key", async () => {
	const calls: Array<{ input: string; init?: RequestInit }> = [];
	globalThis.fetch = mock(async (input: string | URL | Request, init?: RequestInit) => {
		calls.push({ input: String(input), init });
		return response(201, { status: "ok", requestId: "import", data: { importId: "10000000-0000-4000-8000-000000000002", foodItemId: "10000000-0000-4000-8000-000000000003", name: "Apple", physicalState: "solid", merged: false, replayed: false } });
	}) as typeof fetch;
	await importCuratedItem(draft, "stable-key", { csrfToken: "csrf" });
	expect(calls[0]?.input).toBe("/api/v1/admin/imports");
	expect(calls[0]?.init?.credentials).toBe("include");
	expect(calls[0]?.init?.headers).toMatchObject({ "X-CSRF-Token": "csrf", "Idempotency-Key": "stable-key" });
	expect(JSON.parse(String(calls[0]?.init?.body))).toEqual(draft);
});

test("uses a strict bounded CSRF preflight when the caller supplies no token", async () => {
	const calls: Array<{ input: string; init?: RequestInit }> = [];
	globalThis.fetch = mock(async (input: string | URL | Request, init?: RequestInit) => {
		calls.push({ input: String(input), init });
		if (calls.length === 1) return response(200, { status: "ok", requestId: "csrf-request", data: { csrfToken: "csrf" } });
		return response(201, { status: "ok", requestId: "import", data: { importId: "10000000-0000-4000-8000-000000000002", foodItemId: "10000000-0000-4000-8000-000000000003", name: "Apple", physicalState: "solid", merged: false, replayed: false } });
	}) as typeof fetch;

	await importCuratedItem(draft, "stable-key");
	expect(calls.map(({ input }) => input)).toEqual(["/api/v1/auth/csrf-token", "/api/v1/admin/imports"]);
	expect(calls[0]?.init).toMatchObject({ method: "GET", credentials: "include" });
	expect(calls[1]?.init?.headers).toMatchObject({ "X-CSRF-Token": "csrf", "Idempotency-Key": "stable-key" });
});

test("rejects undocumented successful statuses for every operation", async () => {
	for (const [status, body, invoke, code] of [
		[201, { status: "ok", requestId: "wrong-search-status", data: { candidates: [], warnings: [], page: 1 } }, () => searchExternalFoods("apple", "all", 1), "malformed_search_response"],
		[201, { status: "ok", requestId: "wrong-classifications-status", data: { classifications: [] } }, () => loadAdminClassifications("food_category"), "malformed_classifications_response"],
		[200, { status: "ok", requestId: "wrong-import-status", data: { importId: "10000000-0000-4000-8000-000000000002", foodItemId: "10000000-0000-4000-8000-000000000003", name: "Apple", physicalState: "solid", merged: false, replayed: false } }, () => importCuratedItem(draft, "stable", { csrfToken: "csrf" }), "malformed_import_response"]
	] as const) {
		globalThis.fetch = mock(async () => response(status, body)) as typeof fetch;
		await expect(invoke()).rejects.toMatchObject({ appError: { code } });
	}
});

test("rejects oversized success bodies and bounds oversized error bodies", async () => {
	globalThis.fetch = mock(async () => response(200, {
		status: "ok",
		requestId: "x".repeat(300 * 1024),
		data: { candidates: [], warnings: [], page: 1 }
	})) as typeof fetch;
	await expect(searchExternalFoods("apple", "all", 1)).rejects.toMatchObject({ appError: { code: "malformed_search_response" } });

	globalThis.fetch = mock(async () => response(503, {
		status: "error",
		requestId: "x".repeat(20 * 1024),
		error: { category: "dependency", code: "provider_unavailable", message: "unsafe", retryable: true }
	})) as typeof fetch;
	try {
		await searchExternalFoods("apple", "all", 1);
		throw new Error("expected failure");
	} catch (error) {
		expect(error).toMatchObject({ status: 503, appError: { code: "provider_unavailable" } });
		expect((error as ExternalAdminClientError).appError.requestId).toBeUndefined();
	}
});

test("exposes only bounded printable request IDs", async () => {
	for (const requestId of ["x".repeat(121), "request id", "request\nsecret", "request\u0000secret"]) {
		globalThis.fetch = mock(async () => response(503, {
			status: "error",
			requestId,
			error: { category: "dependency", code: "provider_unavailable", message: "unsafe", retryable: true }
		})) as typeof fetch;
		try {
			await searchExternalFoods("apple", "all", 1);
			throw new Error("expected failure");
		} catch (error) {
			expect((error as ExternalAdminClientError).appError.requestId).toBeUndefined();
		}
	}
	globalThis.fetch = mock(async () => response(200, {
		status: "ok",
		requestId: "request id",
		data: { candidates: [], warnings: [], page: 1 }
	})) as typeof fetch;
	await expect(searchExternalFoods("apple", "all", 1)).rejects.toMatchObject({ appError: { code: "malformed_search_response" } });

	const maximumRequestId = `r${"x".repeat(119)}`;
	globalThis.fetch = mock(async () => response(503, {
		status: "error",
		requestId: maximumRequestId,
		error: { category: "dependency", code: "provider_unavailable", message: "safe", retryable: true }
	})) as typeof fetch;
	await expect(searchExternalFoods("apple", "all", 1)).rejects.toMatchObject({ appError: { requestId: maximumRequestId } });
});

test("classifies rate, timeout, unavailable, conflict, and malformed failures with safe messages", async () => {
	for (const [status, expected] of [[429, "rate limited"], [504, "timed out"], [503, "temporarily unavailable"], [409, "conflicts with existing data"]] as const) {
		globalThis.fetch = mock(async () => response(status, { status: "error", requestId: "safe-id", error: { category: "unknown", code: "raw-secret", message: "RAW PROVIDER DIAGNOSTIC", retryable: true } }, { "Retry-After": "12" })) as typeof fetch;
		try {
			if (status === 409) await importCuratedItem(draft, "stable", { csrfToken: "csrf" });
			else await searchExternalFoods("apple", "all", 1);
			throw new Error("expected failure");
		} catch (error) {
			expect(error).toBeInstanceOf(ExternalAdminClientError);
			expect((error as Error).message).toContain(expected);
			expect((error as Error).message).not.toContain("RAW");
		}
	}
});

test("preserves only allowlisted import conflict kinds", async () => {
	for (const [sourceCode, expectedCode, expectedMessage] of [
		["name_conflict_confirmation_required", "name_conflict_confirmation_required", "matching local item"],
		["provider_identity_conflict", "provider_identity_conflict", "provider item"],
		["idempotency_key_conflict", "idempotency_key_conflict", "import attempt"],
		["raw-secret", "import_conflict", "conflicts with existing data"]
	] as const) {
		globalThis.fetch = mock(async () => response(409, {
			status: "error",
			requestId: "safe-id",
			error: { category: "validation", code: sourceCode, message: "RAW SQL diagnostics", retryable: false }
		})) as typeof fetch;
		await expect(importCuratedItem(draft, "stable", { csrfToken: "csrf" })).rejects.toMatchObject({
			status: 409,
			appError: { code: expectedCode, message: expect.stringContaining(expectedMessage) }
		});
	}
});

test("rejects malformed nested search candidates and provider warnings", async () => {
	const validCandidate = {
		provider: "usda",
		externalId: "100",
		name: "Apple",
		physicalState: "solid",
		macrosPer100: { protein: 1, carbohydrates: 20, fat: 0 },
		micronutrients: {},
		warnings: []
	};
	for (const data of [
		{ candidates: [{ ...validCandidate, warnings: null }], warnings: [], page: 1 },
		{ candidates: [{ ...validCandidate, warnings: ["provider-secret-warning"] }], warnings: [], page: 1 },
		{ candidates: [{ ...validCandidate, macrosPer100: { protein: 1, carbohydrates: Number.NaN, fat: 0 } }], warnings: [], page: 1 },
		{ candidates: [{ ...validCandidate, micronutrients: { vitamin_c: -1 } }], warnings: [], page: 1 },
		{ candidates: [validCandidate], warnings: [{ provider: "usda", code: "provider-secret", message: "provider-secret" }], page: 1 },
		{ candidates: [validCandidate], warnings: [{ provider: "usda", code: "timeout", message: "provider-secret" }], page: 1 }
	]) {
		globalThis.fetch = mock(async () => response(200, { status: "ok", requestId: "search", data })) as typeof fetch;
		await expect(searchExternalFoods("apple", "all", 1)).rejects.toMatchObject({ appError: { code: "malformed_search_response" } });
	}
});

test("rejects malformed nested classifications and import decisions", async () => {
	globalThis.fetch = mock(async () => response(200, {
		status: "ok",
		requestId: "classes",
		data: { classifications: [{ id: "not-a-uuid", name: "Fruit", kind: "food_category" }] }
	})) as typeof fetch;
	await expect(loadAdminClassifications("food_category")).rejects.toMatchObject({ appError: { code: "malformed_classifications_response" } });

	globalThis.fetch = mock(async () => response(201, {
		status: "ok",
		requestId: "import",
		data: { importId: "not-a-uuid", foodItemId: "also-bad", name: "Apple", physicalState: "solid", merged: "yes", replayed: false }
	})) as typeof fetch;
	await expect(importCuratedItem(draft, "stable", { csrfToken: "csrf" })).rejects.toMatchObject({ appError: { code: "malformed_import_response" } });
});

test("maps ambiguous transport failures to a retry-safe message without diagnostics", async () => {
	globalThis.fetch = mock(async () => { throw new Error("socket provider-secret.example"); }) as typeof fetch;
	await expect(importCuratedItem(draft, "same-key", { csrfToken: "csrf" })).rejects.toMatchObject({
		status: 0,
		appError: { code: "external_request_ambiguous", retryable: true }
	});
});

test("preserves caller fetch cancellation and maps only a genuine timeout", async () => {
	const abortError = new DOMException("caller canceled", "AbortError");
	globalThis.fetch = mock(async () => { throw abortError; }) as typeof fetch;
	try {
		await searchExternalFoods("apple", "all", 1, new AbortController().signal);
		throw new Error("expected cancellation");
	} catch (error) {
		expect(error).toBe(abortError);
	}

	globalThis.fetch = mock(async () => { throw new DOMException("deadline", "TimeoutError"); }) as typeof fetch;
	await expect(searchExternalFoods("apple", "all", 1)).rejects.toMatchObject({
		status: 0,
		appError: { category: "timeout", code: "external_request_timeout" }
	});

	const timeoutSignal = AbortSignal.timeout(1);
	globalThis.fetch = mock(async (_input, init) => new Promise<Response>((_resolve, reject) => {
		if (init?.signal?.aborted) reject(init.signal.reason);
		else init?.signal?.addEventListener("abort", () => reject(init.signal?.reason), { once: true });
	})) as typeof fetch;
	await expect(searchExternalFoods("apple", "all", 1, timeoutSignal)).rejects.toMatchObject({
		status: 0,
		appError: { category: "timeout", code: "external_request_timeout" }
	});

	for (const reason of [abortError, { cancellation: "navigation" }, { name: "TimeoutError", cancellation: "custom" }]) {
		const cancellationController = new AbortController();
		cancellationController.abort(reason);
		globalThis.fetch = mock(async (_input, init) => { throw init?.signal?.reason; }) as typeof fetch;
		try {
			await searchExternalFoods("apple", "all", 1, cancellationController.signal);
			throw new Error("expected signal cancellation");
		} catch (error) {
			expect(error).toBe(reason);
		}
	}
});

test("maps a timed-out signal when fetch rejects with a generic AbortError", async () => {
	const timeoutSignal = AbortSignal.timeout(1);
	globalThis.fetch = mock(async (_input, init) => {
		await new Promise<void>((resolve) => {
			if (init?.signal?.aborted) resolve();
			else init?.signal?.addEventListener("abort", () => resolve(), { once: true });
		});
		throw new DOMException("aborted by fetch", "AbortError");
	}) as typeof fetch;

	await expect(searchExternalFoods("apple", "all", 1, timeoutSignal)).rejects.toMatchObject({
		status: 0,
		appError: { category: "timeout", code: "external_request_timeout" }
	});
});

test("preserves success and error body cancellation while cleaning up readers", async () => {
	for (const status of [200, 503]) {
		const abortError = new DOMException(`body canceled at ${status}`, "AbortError");
		const bodyResponse = abortingBodyResponse(status, abortError);
		globalThis.fetch = mock(async () => bodyResponse.response) as typeof fetch;
		try {
			await searchExternalFoods("apple", "all", 1);
			throw new Error("expected body cancellation");
		} catch (error) {
			expect(error).toBe(abortError);
		}
		expect(bodyResponse.cancel).toHaveBeenCalledTimes(1);
		expect(bodyResponse.releaseLock).toHaveBeenCalledTimes(1);
	}
});

test("rejects bounded-status and hostile-ID CSRF preflight responses before import", async () => {
	const oversized = cancellableResponse(200, { "Content-Length": String(256 * 1024 + 1) });
	globalThis.fetch = mock(async () => oversized.response) as typeof fetch;
	await expect(importCuratedItem(draft, "oversized-csrf")).rejects.toMatchObject({ appError: { code: "malformed_csrf_response" } });
	expect(oversized.cancel).toHaveBeenCalledTimes(1);

	const wrongStatus = cancellableResponse(201);
	globalThis.fetch = mock(async () => wrongStatus.response) as typeof fetch;
	await expect(importCuratedItem(draft, "wrong-status-csrf")).rejects.toMatchObject({ appError: { code: "malformed_csrf_response" } });
	expect(wrongStatus.cancel).toHaveBeenCalledTimes(1);

	const fetchMock = mock(async () => response(200, {
		status: "ok",
		requestId: "hostile\nrequest-id",
		data: { csrfToken: "csrf" }
	}));
	globalThis.fetch = fetchMock as typeof fetch;
	await expect(importCuratedItem(draft, "hostile-id-csrf")).rejects.toMatchObject({ appError: { code: "malformed_csrf_response" } });
	expect(fetchMock).toHaveBeenCalledTimes(1);
});

test("creates a canonical browser idempotency key", () => {
	expect(createImportIdempotencyKey()).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i);
});

test("rejects an invalid candidate image URI", async () => {
	globalThis.fetch = mock(async () => response(200, {
		status: "ok",
		requestId: "search",
		data: {
			candidates: [{ provider: "usda", externalId: "100", name: "Apple", physicalState: "solid", macrosPer100: { protein: 1, carbohydrates: 20, fat: 0 }, micronutrients: {}, imageUrl: "not a URI", warnings: [] }],
			warnings: [],
			page: 1
		}
	})) as typeof fetch;
	await expect(searchExternalFoods("apple", "usda", 1)).rejects.toMatchObject({ appError: { code: "malformed_search_response" } });
});
