import { afterEach, expect, test } from "bun:test";

import type { DailyDiet, DailyDietCreateRequest } from "./generated";
import {
	DailyDietClientError,
	createDailyDiet,
	deleteDailyDiet,
	generateDailyDietIdempotencyKey,
	getDailyDiet,
	listDailyDiets,
	replaceDailyDiet
} from "./daily-diet-client";

// Implements DESIGN-001 SearchView exact Daily Diet response and retry-key client verification.
// Implements DESIGN-017 ErrorMessageMapper cross-user-safe Daily Diet error verification.

type FetchProvider = (init: RequestInit) => Response | Promise<Response>;

class FetchMock {
	calls: Array<{ url: string; init: RequestInit }> = [];
	private providers: FetchProvider[] = [];
	private index = 0;

	enqueue(response: Response): void {
		this.providers.push(() => response);
	}

	fetch = (input: string | URL | Request, init?: RequestInit): Promise<Response> => {
		const url = typeof input === "string" ? input : input.toString();
		this.calls.push({ url, init: init ?? {} });
		const provider = this.providers[this.index++];
		if (!provider) throw new Error(`No response queued for ${url}`);
		return Promise.resolve(provider(init ?? {}));
	};

	reset(): void {
		this.calls = [];
		this.providers = [];
		this.index = 0;
	}
}

const originalFetch = globalThis.fetch;
const fetchMock = new FetchMock();
const dietId = "00000000-0000-0000-0000-000000000001";
const entryId = "00000000-0000-0000-0000-000000000002";
const mealId = "00000000-0000-0000-0000-000000000003";
const createRequest: DailyDietCreateRequest = {
	name: "Training day",
	entries: [{ mealId, quantity: 100, unit: "g", position: 0 }]
};

afterEach(() => {
	globalThis.fetch = originalFetch;
	fetchMock.reset();
});

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

function diet(overrides: Partial<DailyDiet> = {}): DailyDiet {
	return {
		id: dietId,
		name: "Training day",
		entries: [{ id: entryId, mealId, quantity: 100, unit: "g", position: 0 }],
		aggregateMacros: { protein: 20, carbohydrates: 30, fat: 10, calories: 290 },
		createdAt: "2026-07-11T00:00:00Z",
		updatedAt: "2026-07-11T00:00:00.123+02:00",
		...overrides
	};
}

function itemEnvelope(data: unknown = diet(), requestId = "req-diet"): unknown {
	return { status: "ok", requestId, data };
}

function collectionEnvelope(diets: unknown[], requestId = "req-list"): unknown {
	return { status: "ok", requestId, data: { diets } };
}

test("decodes exact empty, list, item, create, replace, and empty-delete responses at their documented statuses", async () => {
	globalThis.fetch = fetchMock.fetch;
	const first = diet();
	const replaced = diet({ name: "Updated day" });
	fetchMock.enqueue(jsonResponse(200, collectionEnvelope([])));
	fetchMock.enqueue(jsonResponse(200, collectionEnvelope([first])));
	fetchMock.enqueue(jsonResponse(200, itemEnvelope(first)));
	fetchMock.enqueue(jsonResponse(201, itemEnvelope(first, "req-create")));
	fetchMock.enqueue(jsonResponse(200, itemEnvelope(replaced, "req-replace")));
	fetchMock.enqueue(new Response(null, { status: 204 }));

	expect(await listDailyDiets()).toEqual([]);
	expect(await listDailyDiets()).toEqual([first]);
	expect(await getDailyDiet("diet/1")).toEqual(first);
	expect(await createDailyDiet(createRequest, { csrfToken: "csrf-create", idempotencyKey: "daily-diet-key" })).toEqual(first);
	expect(await replaceDailyDiet(dietId, { ...createRequest, name: "Updated day" }, { csrfToken: "csrf-replace" })).toEqual(replaced);
	await deleteDailyDiet(dietId, { csrfToken: "csrf-delete" });

	expect(fetchMock.calls[0]).toMatchObject({ url: "/api/v1/daily-diets", init: { method: "GET", credentials: "include" } });
	expect(fetchMock.calls[2]?.url).toBe("/api/v1/daily-diets/diet%2F1");
	expect(fetchMock.calls[3]?.init.headers).toMatchObject({ "Idempotency-Key": "daily-diet-key", "X-CSRF-Token": "csrf-create" });
	expect(fetchMock.calls[4]?.init.method).toBe("PUT");
	expect(fetchMock.calls[5]?.init.method).toBe("DELETE");
});

test("rejects every unexpected successful endpoint status", async () => {
	globalThis.fetch = fetchMock.fetch;
	fetchMock.enqueue(jsonResponse(201, collectionEnvelope([])));
	fetchMock.enqueue(jsonResponse(201, itemEnvelope()));
	fetchMock.enqueue(jsonResponse(200, itemEnvelope()));
	fetchMock.enqueue(jsonResponse(201, itemEnvelope()));
	fetchMock.enqueue(new Response("", { status: 200 }));

	await expect(listDailyDiets()).rejects.toMatchObject({ status: 201, appError: { code: "malformed_daily_diet_response" } });
	await expect(getDailyDiet(dietId)).rejects.toMatchObject({ status: 201, appError: { code: "malformed_daily_diet_response" } });
	await expect(createDailyDiet(createRequest, { csrfToken: "csrf", idempotencyKey: "daily-diet-key" })).rejects.toMatchObject({ status: 200 });
	await expect(replaceDailyDiet(dietId, createRequest, { csrfToken: "csrf" })).rejects.toMatchObject({ status: 201 });
	await expect(deleteDailyDiet(dietId, { csrfToken: "csrf" })).rejects.toMatchObject({ status: 200 });
});

test("rejects malformed envelopes, wrong request IDs, nulls, and additional fields", async () => {
	globalThis.fetch = fetchMock.fetch;
	const malformed = [
		{ status: "accepted", requestId: "req", data: diet() },
		{ status: "ok", requestId: "has space", data: diet() },
		{ status: "ok", requestId: "x".repeat(121), data: diet() },
		{ status: "ok", requestId: "req", data: null },
		{ status: "ok", requestId: "req", data: diet(), error: null },
		{ status: "ok", requestId: "req", data: { ...diet(), debug: "postgres://secret" } },
		{ status: "ok", requestId: "req", data: { ...diet(), aggregateMacros: { ...diet().aggregateMacros, sodium: 1 } } }
	];
	for (const body of malformed) fetchMock.enqueue(jsonResponse(200, body));
	for (const _ of malformed) await expect(getDailyDiet(dietId)).rejects.toMatchObject({ appError: { code: "malformed_daily_diet_response" } });
});

test("rejects hostile collection and wrong-typed nested payloads before they reach callers", async () => {
	globalThis.fetch = fetchMock.fetch;
	const hostile = [
		{ status: "ok", requestId: "req", data: { diets: [], extra: true } },
		{ status: "ok", requestId: "req", data: { diets: [null] } },
		itemEnvelope({ ...diet(), name: 7 }),
		itemEnvelope({ ...diet(), entries: [null] }),
		itemEnvelope({ ...diet(), entries: [{ ...diet().entries[0]!, quantity: "100" }] }),
		itemEnvelope({ ...diet(), aggregateMacros: null }),
		itemEnvelope({ ...diet(), aggregateMacros: { ...diet().aggregateMacros, fat: "10" } })
	];
	for (const body of hostile) fetchMock.enqueue(jsonResponse(200, body));

	await expect(listDailyDiets()).rejects.toMatchObject({ appError: { code: "malformed_daily_diet_response" } });
	await expect(listDailyDiets()).rejects.toMatchObject({ appError: { code: "malformed_daily_diet_response" } });
	for (let index = 2; index < hostile.length; index += 1) {
		await expect(getDailyDiet(dietId)).rejects.toMatchObject({ appError: { code: "malformed_daily_diet_response" } });
	}
});

test("rejects malformed UUIDs and dates", async () => {
	globalThis.fetch = fetchMock.fetch;
	for (const data of [
		diet({ id: "diet-1" }),
		diet({ entries: [{ ...diet().entries[0]!, mealId: "MEAL" }] }),
		diet({ createdAt: "yesterday" }),
		diet({ updatedAt: "2026-02-30T00:00:00Z" })
	]) fetchMock.enqueue(jsonResponse(200, itemEnvelope(data)));

	for (let index = 0; index < 4; index += 1) await expect(getDailyDiet(dietId)).rejects.toBeInstanceOf(DailyDietClientError);
});

test("rejects unsupported units and invalid quantities, positions, and macros", async () => {
	globalThis.fetch = fetchMock.fetch;
	const invalid = [
		diet({ entries: [{ ...diet().entries[0]!, unit: "kg" as "g" }] }),
		diet({ entries: [{ ...diet().entries[0]!, quantity: 0 }] }),
		diet({ entries: [{ ...diet().entries[0]!, quantity: 1.0001 }] }),
		diet({ entries: [{ ...diet().entries[0]!, position: 100 }] }),
		diet({ aggregateMacros: { ...diet().aggregateMacros, protein: -1 } }),
		diet({ aggregateMacros: { ...diet().aggregateMacros, calories: Number.POSITIVE_INFINITY } })
	];
	for (const data of invalid) fetchMock.enqueue(jsonResponse(200, itemEnvelope(data)));
	for (const _ of invalid) await expect(getDailyDiet(dietId)).rejects.toMatchObject({ appError: { code: "malformed_daily_diet_response" } });
});

test("rejects empty and oversized nested collections, names, and response documents", async () => {
	globalThis.fetch = fetchMock.fetch;
	const entry = diet().entries[0]!;
	fetchMock.enqueue(jsonResponse(200, itemEnvelope(diet({ entries: [] }))));
	fetchMock.enqueue(jsonResponse(200, itemEnvelope(diet({ entries: Array.from({ length: 101 }, () => entry) }))));
	fetchMock.enqueue(jsonResponse(200, itemEnvelope(diet({ name: "x".repeat(121) }))));
	fetchMock.enqueue(new Response(`{"status":"ok","requestId":"req","data":"${"x".repeat(5 * 1024 * 1024)}"}`, { status: 200 }));

	for (let index = 0; index < 4; index += 1) await expect(getDailyDiet(dietId)).rejects.toMatchObject({ appError: { code: "malformed_daily_diet_response" } });
});

test("cancels a chunked response as soon as its bounded body limit is exceeded", async () => {
	globalThis.fetch = fetchMock.fetch;
	const chunk = new Uint8Array(2 * 1024 * 1024);
	let pulls = 0;
	let cancelled = false;
	const body = new ReadableStream<Uint8Array>({
		pull(controller) {
			pulls += 1;
			if (pulls <= 3) controller.enqueue(chunk);
			else controller.close();
		},
		cancel() {
			cancelled = true;
		}
	}, { highWaterMark: 0 });
	fetchMock.enqueue(new Response(body, { status: 200 }));

	await expect(getDailyDiet(dietId)).rejects.toMatchObject({ appError: { code: "malformed_daily_diet_response" } });
	expect(pulls).toBe(3);
	expect(cancelled).toBe(true);
});

test("requires a caller-owned create key before CSRF or network I/O", async () => {
	globalThis.fetch = fetchMock.fetch;
	const untypedCreate = createDailyDiet as unknown as (request: DailyDietCreateRequest, options?: object) => Promise<unknown>;

	await expect(untypedCreate(createRequest)).rejects.toMatchObject({ status: 0, appError: { code: "daily_diet_idempotency_key_required" } });
	await expect(untypedCreate(createRequest, { idempotencyKey: "short" })).rejects.toMatchObject({ status: 0 });
	await expect(untypedCreate(createRequest, { idempotencyKey: "unsafe\nkey" })).rejects.toMatchObject({ status: 0 });
	expect(fetchMock.calls).toHaveLength(0);
	if (false) {
		// @ts-expect-error DESIGN-001 requires operation-level idempotency ownership.
		void createDailyDiet(createRequest);
	}
});

test("secure random unavailability and provider failure fail closed without a weak fallback", () => {
	const originalCrypto = globalThis.crypto;
	try {
		Object.defineProperty(globalThis, "crypto", { configurable: true, value: undefined });
		expect(() => generateDailyDietIdempotencyKey()).toThrow(DailyDietClientError);
		expect(() => generateDailyDietIdempotencyKey()).toThrow("secure Daily Diet request");
		Object.defineProperty(globalThis, "crypto", {
			configurable: true,
			value: { randomUUID: () => { throw new Error("provider failed"); } }
		});
		expect(() => generateDailyDietIdempotencyKey()).toThrow(DailyDietClientError);
		expect(() => generateDailyDietIdempotencyKey()).toThrow("secure Daily Diet request");
	} finally {
		Object.defineProperty(globalThis, "crypto", { configurable: true, value: originalCrypto });
	}
});

test("malformed secure-random provider results fail closed before network I/O", () => {
	const originalCrypto = globalThis.crypto;
	globalThis.fetch = fetchMock.fetch;
	try {
		Object.defineProperty(globalThis, "crypto", {
			configurable: true,
			value: { randomUUID: () => "123e4567-e89b-42d3-a456-426614174000" }
		});
		expect(generateDailyDietIdempotencyKey()).toBe("daily-diet-123e4567-e89b-42d3-a456-426614174000");
		for (const value of [null, undefined, "fixed", {}, "00000000-0000-0000-0000-000000000000", "AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA"]) {
			Object.defineProperty(globalThis, "crypto", {
				configurable: true,
				value: { randomUUID: () => value }
			});
			expect(() => generateDailyDietIdempotencyKey()).toThrow(DailyDietClientError);
			expect(() => generateDailyDietIdempotencyKey()).toThrow("secure Daily Diet request");
		}
		expect(fetchMock.calls).toHaveLength(0);
	} finally {
		Object.defineProperty(globalThis, "crypto", { configurable: true, value: originalCrypto });
	}
});

test("acquires CSRF in memory and maps ownership and shared errors through one safe path", async () => {
	globalThis.fetch = fetchMock.fetch;
	fetchMock.enqueue(jsonResponse(200, { status: "ok", requestId: "req-csrf", data: { csrfToken: "csrf-memory" } }));
	fetchMock.enqueue(jsonResponse(201, itemEnvelope()));
	fetchMock.enqueue(jsonResponse(403, { status: "error", requestId: "req-forbidden", error: { category: "security", code: "cross_user_access", message: "postgres://secret", retryable: false } }));
	fetchMock.enqueue(jsonResponse(404, { status: "error", requestId: "req-missing", error: { message: "different user diet" } }));
	fetchMock.enqueue(jsonResponse(500, { status: "error", requestId: "x".repeat(121), error: { category: "server", code: "internal_error", message: "password=secret", retryable: true } }));

	await createDailyDiet(createRequest, { idempotencyKey: "daily-diet-key" });
	expect(fetchMock.calls[0]?.url).toBe("/api/v1/auth/csrf-token");
	expect(fetchMock.calls[1]?.init.headers).toMatchObject({ "X-CSRF-Token": "csrf-memory" });

	const forbidden = await clientError(() => getDailyDiet(dietId));
	const missing = await clientError(() => getDailyDiet(dietId));
	expect(forbidden.appError.message).toBe("Saved daily diet is unavailable.");
	expect(missing.appError.message).toBe(forbidden.appError.message);
	const server = await clientError(() => listDailyDiets());
	expect(server.appError).toMatchObject({ code: "internal_error", message: "Saved daily diets are temporarily unavailable. Please try again." });
	expect(server.appError.requestId).toBeUndefined();
});

async function clientError(action: () => Promise<unknown>): Promise<DailyDietClientError> {
	try {
		await action();
	} catch (error) {
		if (error instanceof DailyDietClientError) return error;
	}
	throw new Error("Expected DailyDietClientError");
}
