import { afterEach, expect, test } from "bun:test";

import type { DailyDiet, DailyDietCollectionEnvelope, DailyDietEnvelope } from "./generated";
import {
	DailyDietClientError,
	createDailyDiet,
	deleteDailyDiet,
	fetchDailyDiets,
	getDailyDiet,
	replaceDailyDiet
} from "./daily-diet-client";

// Implements DESIGN-001 SearchView Daily Diet generated-contract client verification.
// Implements DESIGN-017 ErrorMessageMapper cross-user-safe Daily Diet error verification.

type FetchProvider = (init: RequestInit) => Response | Promise<Response>;

class FetchMock {
	calls: Array<{ url: string; init: RequestInit }> = [];
	private providers: FetchProvider[] = [];
	private index = 0;

	enqueue(response: Response): void {
		this.providers.push(() => response);
	}

	reset(): void {
		this.calls = [];
		this.providers = [];
		this.index = 0;
	}

	fetch = (input: string | URL | Request, init?: RequestInit): Promise<Response> => {
		const url = typeof input === "string" ? input : input.toString();
		this.calls.push({ url, init: init ?? {} });
		const provider = this.providers[this.index++];
		if (!provider) throw new Error(`No response queued for ${url}`);
		return Promise.resolve(provider(init ?? {}));
	};
}

const originalFetch = globalThis.fetch;
const fetchMock = new FetchMock();

afterEach(() => {
	globalThis.fetch = originalFetch;
	fetchMock.reset();
});

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

function diet(id = "diet-1", name = "Training day"): DailyDiet {
	return {
		id,
		name,
		entries: [{ id: "entry-1", mealId: "meal-1", quantity: 100, unit: "g", position: 0 }],
		aggregateMacros: { protein: 20, carbohydrates: 30, fat: 10, calories: 290 },
		createdAt: "2026-07-11T00:00:00Z",
		updatedAt: "2026-07-11T00:00:00Z"
	};
}

function dietEnvelope(data: DailyDiet): DailyDietEnvelope {
	return { status: "ok", requestId: "req-diet", data };
}

function collectionEnvelope(diets: DailyDiet[]): DailyDietCollectionEnvelope {
	return { status: "ok", requestId: "req-list", data: { diets } };
}

const request = { name: "Training day", entries: [{ mealId: "meal-1", quantity: 100, unit: "g" as const, position: 0 }] };

test("list and get use credentialed generated read requests and decode DTO envelopes", async () => {
	globalThis.fetch = fetchMock.fetch;
	const first = diet();
	fetchMock.enqueue(jsonResponse(200, collectionEnvelope([first])));
	fetchMock.enqueue(jsonResponse(200, dietEnvelope(first)));

	expect(await fetchDailyDiets()).toEqual([first]);
	expect(await getDailyDiet("diet/1")).toEqual(first);
	expect(fetchMock.calls[0]?.url).toBe("/api/v1/daily-diets");
	expect(fetchMock.calls[0]?.init.credentials).toBe("include");
	expect(fetchMock.calls[0]?.init.headers).toEqual({ Accept: "application/json" });
	expect(fetchMock.calls[1]?.url).toBe("/api/v1/daily-diets/diet%2F1");
});

test("create sends a generated DTO with CSRF and one idempotency key", async () => {
	globalThis.fetch = fetchMock.fetch;
	const created = diet("diet-created");
	fetchMock.enqueue(jsonResponse(201, dietEnvelope(created)));

	await createDailyDiet(request, { csrfToken: "csrf-token", idempotencyKey: "diet-key" });

	const call = fetchMock.calls[0];
	expect(call?.url).toBe("/api/v1/daily-diets");
	expect(call?.init.credentials).toBe("include");
	expect(call?.init.method).toBe("POST");
	expect(call?.init.headers).toEqual({
		Accept: "application/json",
		"Content-Type": "application/json",
		"Idempotency-Key": "diet-key",
		"X-CSRF-Token": "csrf-token"
	});
	expect(JSON.parse(String(call?.init.body))).toEqual(request);
});

test("replace and delete use CSRF-protected generated mutation requests", async () => {
	globalThis.fetch = fetchMock.fetch;
	const replaced = diet("diet-1", "Updated day");
	fetchMock.enqueue(jsonResponse(200, dietEnvelope(replaced)));
	fetchMock.enqueue(new Response(null, { status: 204 }));

	await replaceDailyDiet("diet-1", { ...request, name: "Updated day" }, { csrfToken: "csrf-put" });
	await deleteDailyDiet("diet-1", { csrfToken: "csrf-delete" });

	expect(fetchMock.calls[0]?.init.method).toBe("PUT");
	expect(fetchMock.calls[0]?.init.headers).toEqual({
		Accept: "application/json",
		"Content-Type": "application/json",
		"X-CSRF-Token": "csrf-put"
	});
	expect(fetchMock.calls[1]?.init.method).toBe("DELETE");
	expect(fetchMock.calls[1]?.init.headers).toEqual({ Accept: "application/json", "X-CSRF-Token": "csrf-delete" });
});

test("mutations acquire CSRF in memory and never persist session data", async () => {
	globalThis.fetch = fetchMock.fetch;
	fetchMock.enqueue(jsonResponse(200, { status: "ok", requestId: "req-csrf", data: { csrfToken: "csrf-memory" } }));
	fetchMock.enqueue(jsonResponse(201, dietEnvelope(diet("diet-created"))));

	await createDailyDiet(request, { idempotencyKey: "diet-key" });

	expect(fetchMock.calls[0]?.url).toBe("/api/v1/auth/csrf-token");
	expect(fetchMock.calls[1]?.init.headers).toMatchObject({ "X-CSRF-Token": "csrf-memory" });
});

test("403 and 404 project the same safe cross-user-unavailable error", async () => {
	globalThis.fetch = fetchMock.fetch;
	fetchMock.enqueue(jsonResponse(403, {
		status: "error",
		requestId: "req-forbidden",
		error: { category: "security", code: "cross_user_access", message: "postgres stack at https://internal", retryable: false }
	}));
	fetchMock.enqueue(jsonResponse(404, { status: "error", requestId: "req-missing", error: { message: "different user diet" } }));

	const first = await getClientError("diet-other");
	const second = await getClientError("diet-missing");
	expect(first.appError.message).toBe("Saved daily diet is unavailable.");
	expect(second.appError.message).toBe(first.appError.message);
	expect(first.appError.message).not.toContain("postgres");
});

async function getClientError(dietId: string): Promise<DailyDietClientError> {
	try {
		await getDailyDiet(dietId);
	} catch (error) {
		if (error instanceof DailyDietClientError) return error;
	}
	throw new Error("Expected DailyDietClientError");
}
