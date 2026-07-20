import { afterEach, expect, test } from "bun:test";

import type {
	DietOptimizationRequest,
	OptimizationJobAcknowledgementEnvelope,
	OptimizationJobStatusEnvelope
} from "./generated";
import { OptimizationClientError, generateOptimizationIdempotencyKey, getOptimizationJob, submitOptimization } from "./optimization-client";

// Implements DESIGN-001 SearchView OptimizationWorkflow generated request/response verification.
// Implements DESIGN-004 JobStatusTracker idempotent submission and user-safe polling errors.

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

afterEach(() => {
	globalThis.fetch = originalFetch;
	fetchMock.reset();
});

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

const request: DietOptimizationRequest = {
	dailyDietId: "00000000-0000-0000-0000-000000000001",
	tolerancePercent: 10,
	excludedMealIds: []
};
const mealId = "00000000-0000-0000-0000-000000000003";
const submissionKey = "optimization-00000000-0000-4000-8000-000000000004";

const acknowledgement: OptimizationJobAcknowledgementEnvelope = {
	status: "accepted",
	requestId: "optimization-submit",
	data: { jobId: "00000000-0000-0000-0000-000000000002", status: "queued", pollUrl: "/api/v1/optimization/jobs/00000000-0000-0000-0000-000000000002" }
};

test("submit uses generated DTO JSON, cookies, CSRF, and exactly one caller-owned idempotency key", async () => {
	globalThis.fetch = fetchMock.fetch;
	fetchMock.enqueue(jsonResponse(202, acknowledgement));

	await submitOptimization(request, { csrfToken: "csrf-token", idempotencyKey: submissionKey });

	const call = fetchMock.calls[0];
	expect(call?.url).toBe("/api/v1/optimization/jobs");
	expect(call?.init.method).toBe("POST");
	expect(call?.init.credentials).toBe("include");
	expect(call?.init.headers).toEqual({
		Accept: "application/json",
		"Content-Type": "application/json",
		"Idempotency-Key": submissionKey,
		"X-CSRF-Token": "csrf-token"
	});
	expect(JSON.parse(String(call?.init.body))).toEqual(request);
});

test("polling decodes completed alternatives through the generated nutrition projection", async () => {
	globalThis.fetch = fetchMock.fetch;
	const response: OptimizationJobStatusEnvelope = {
		status: "ok",
		requestId: "optimization-complete",
		data: {
			jobId: acknowledgement.data!.jobId,
			dailyDietId: request.dailyDietId,
			status: "completed",
			pollUrl: acknowledgement.data!.pollUrl,
			createdAt: "2026-07-11T00:00:00Z",
			startedAt: "2026-07-11T00:00:01Z",
			finishedAt: "2026-07-11T00:00:02Z",
			alternatives: [{ meals: [{ mealId, quantity: 100, unit: "g", position: 0 }], macros: { protein: 40, carbohydrates: 80, fat: 20, calories: 640 }, similarityScore: 0.8 }]
		}
	};
	fetchMock.enqueue(jsonResponse(200, response));

	const job = await getOptimizationJob(acknowledgement.data!.jobId);

	expect(fetchMock.calls[0]?.url).toBe(`/api/v1/optimization/jobs/${acknowledgement.data!.jobId}`);
	expect(fetchMock.calls[0]?.init.headers).toEqual({ Accept: "application/json" });
	expect(job.status).toBe("completed");
	if (job.status === "completed") expect(job.alternatives[0]?.macros.calories).toBe(640);
});

test("polling decodes failed jobs with contract-shaped partial alternatives", async () => {
	globalThis.fetch = fetchMock.fetch;
	fetchMock.enqueue(jsonResponse(200, {
		status: "ok",
		requestId: "optimization-partial",
		data: {
			jobId: acknowledgement.data!.jobId,
			dailyDietId: request.dailyDietId,
			status: "failed",
			pollUrl: acknowledgement.data!.pollUrl,
			createdAt: "2026-07-11T00:00:00Z",
			alternatives: [{ meals: [{ mealId, quantity: 100, unit: "g", position: 0 }], macros: { protein: 40, carbohydrates: 80, fat: 20, calories: 640 }, similarityScore: 0.8 }],
			failure: { code: "solver_timeout", message: "Optimization took too long. Please try again." }
		}
	}));

	const job = await getOptimizationJob(acknowledgement.data!.jobId);

	expect(job.status).toBe("failed");
	if (job.status === "failed") expect(job.alternatives?.[0]?.macros.calories).toBe(640);
});

test("polling rejects the legacy top-level calorie placement", async () => {
	globalThis.fetch = fetchMock.fetch;
	fetchMock.enqueue(jsonResponse(200, {
		status: "ok",
		requestId: "optimization-legacy",
		data: {
			jobId: acknowledgement.data!.jobId,
			dailyDietId: request.dailyDietId,
			status: "completed",
			pollUrl: acknowledgement.data!.pollUrl,
			createdAt: "2026-07-11T00:00:00Z",
			startedAt: "2026-07-11T00:00:01Z",
			finishedAt: "2026-07-11T00:00:02Z",
			alternatives: [{ meals: [{ mealId: "meal-1", quantity: 100, unit: "g", position: 0 }], macros: { protein: 40, carbohydrates: 80, fat: 20 }, calories: 640, similarityScore: 0.8 }]
		}
	}));

	await expect(getOptimizationJob(acknowledgement.data!.jobId)).rejects.toBeInstanceOf(OptimizationClientError);
});

test("polling maps expired and queue responses to safe retryable errors", async () => {
	globalThis.fetch = fetchMock.fetch;
	fetchMock.enqueue(jsonResponse(410, { status: "error", requestId: "expired", error: { category: "validation", code: "result_expired", message: "optimization result has expired", retryable: true } }));
	fetchMock.enqueue(jsonResponse(503, { status: "error", requestId: "queue", error: { category: "dependency", code: "queue_unavailable", message: "optimization queue is unavailable", retryable: true } }));

	await expect(getOptimizationJob("00000000-0000-0000-0000-000000000002")).rejects.toMatchObject({ status: 410, appError: { code: "result_expired", retryable: true } });
	await expect(getOptimizationJob("00000000-0000-0000-0000-000000000002")).rejects.toMatchObject({ status: 503, appError: { code: "queue_unavailable", retryable: true } });
});

test("submission maps audited 429, 500, 503, and 504 envelopes to bounded errors", async () => {
	globalThis.fetch = fetchMock.fetch;
	for (const [status, category, code] of [
		[429, "rate_limit", "optimization_rate_limited"],
		[500, "server", "internal_error"],
		[503, "dependency", "queue_unavailable"],
		[504, "timeout", "request_timeout"]
	] as const) {
		fetchMock.enqueue(jsonResponse(status, {
			status: "error",
			requestId: `req-${status}`,
			error: { category, code, message: `safe ${status} response`, retryable: true }
		}));
	}

	for (const [status, category, code] of [
		[429, "rate_limit", "optimization_rate_limited"],
		[500, "server", "internal_error"],
		[503, "dependency", "queue_unavailable"],
		[504, "timeout", "request_timeout"]
	] as const) {
		await expect(submitOptimization(request, { csrfToken: "csrf", idempotencyKey: submissionKey })).rejects.toMatchObject({
			status,
			appError: { category, code, requestId: `req-${status}`, retryable: true }
		});
	}
});

test("malformed successful job responses fail closed", async () => {
	globalThis.fetch = fetchMock.fetch;
	fetchMock.enqueue(jsonResponse(200, { status: "ok", requestId: "malformed", data: { status: "completed", alternatives: [] } }));

	await expect(getOptimizationJob("00000000-0000-0000-0000-000000000002")).rejects.toBeInstanceOf(OptimizationClientError);
});

test("accepts only exact 202 acknowledgement and exact 200 job variants", async () => {
	globalThis.fetch = fetchMock.fetch;
	const common = {
		jobId: acknowledgement.data!.jobId,
		dailyDietId: request.dailyDietId,
		pollUrl: acknowledgement.data!.pollUrl,
		createdAt: "2026-07-11T00:00:00Z"
	};
	const alternative = { meals: [{ mealId, quantity: 100, unit: "g", position: 0 }], macros: { protein: 40, carbohydrates: 80, fat: 20, calories: 640 }, similarityScore: 0.8 };
	const jobs = [
		{ ...common, status: "queued" },
		{ ...common, status: "processing", startedAt: "2026-07-11T00:00:01Z" },
		{ ...common, status: "completed", startedAt: "2026-07-11T00:00:01Z", finishedAt: "2026-07-11T00:00:02Z", alternatives: [alternative] },
		{ ...common, status: "failed", startedAt: null, finishedAt: "2026-07-11T00:00:02Z", alternatives: [], failure: { code: "failed_validation", message: "The optimization request could not be validated." } },
		{ ...common, status: "cancelled", finishedAt: "2026-07-11T00:00:02Z" }
	];
	for (const [index, job] of jobs.entries()) fetchMock.enqueue(jsonResponse(200, { status: "ok", requestId: `variant-${index}`, data: job }));

	for (const expected of ["queued", "processing", "completed", "failed", "cancelled"]) {
		expect((await getOptimizationJob(common.jobId)).status).toBe(expected);
	}

	fetchMock.enqueue(jsonResponse(200, acknowledgement));
	await expect(submitOptimization(request, { csrfToken: "csrf", idempotencyKey: submissionKey })).rejects.toMatchObject({ appError: { code: "malformed_optimization_response" } });
	fetchMock.enqueue(jsonResponse(201, acknowledgement));
	await expect(submitOptimization(request, { csrfToken: "csrf", idempotencyKey: submissionKey })).rejects.toMatchObject({ status: 201, appError: { code: "malformed_optimization_response" } });
	fetchMock.enqueue(jsonResponse(202, acknowledgement));
	await expect(getOptimizationJob(common.jobId)).rejects.toMatchObject({ status: 202, appError: { code: "malformed_optimization_response" } });
});

test("rejects malformed envelopes, identity fields, variant fields, poll URLs, dates, and nested alternatives", async () => {
	globalThis.fetch = fetchMock.fetch;
	const common = {
		jobId: acknowledgement.data!.jobId,
		dailyDietId: request.dailyDietId,
		status: "completed",
		pollUrl: acknowledgement.data!.pollUrl,
		createdAt: "2026-07-11T00:00:00Z",
		startedAt: "2026-07-11T00:00:01Z",
		finishedAt: "2026-07-11T00:00:02Z",
		alternatives: [{ meals: [{ mealId, quantity: 100, unit: "g", position: 0 }], macros: { protein: 40, carbohydrates: 80, fat: 20, calories: 640 }, similarityScore: 0.8 }]
	};
	const malformed: unknown[] = [
		{ status: "ok", requestId: "has space", data: common },
		{ status: "ok", requestId: "x".repeat(121), data: common },
		{ status: "ok", requestId: "extra-envelope", data: common, extra: true },
		{ status: "ok", requestId: "bad-job", data: { ...common, jobId: "not-a-uuid" } },
		{ status: "ok", requestId: "bad-diet", data: { ...common, dailyDietId: null } },
		{ status: "ok", requestId: "foreign-url", data: { ...common, pollUrl: "https://evil.example/jobs/1" } },
		{ status: "ok", requestId: "wrong-url-id", data: { ...common, pollUrl: `${common.pollUrl}0` } },
		{ status: "ok", requestId: "bad-date", data: { ...common, finishedAt: "yesterday" } },
		{ status: "ok", requestId: "cross-variant", data: { ...common, status: "queued" } },
		{ status: "ok", requestId: "extra-job", data: { ...common, debug: "secret" } },
		{ status: "ok", requestId: "empty-alts", data: { ...common, alternatives: [] } },
		{ status: "ok", requestId: "too-many-alts", data: { ...common, alternatives: Array(4).fill(common.alternatives[0]) } },
		{ status: "ok", requestId: "empty-meals", data: { ...common, alternatives: [{ ...common.alternatives[0], meals: [] }] } },
		{ status: "ok", requestId: "bad-meal", data: { ...common, alternatives: [{ ...common.alternatives[0], meals: [{ mealId, quantity: 0, unit: "kg", position: 100 }] }] } },
		{ status: "ok", requestId: "bad-macros", data: { ...common, alternatives: [{ ...common.alternatives[0], macros: { protein: -1, carbohydrates: 80, fat: 20, calories: 640 } }] } },
		{ status: "ok", requestId: "extra-alt", data: { ...common, alternatives: [{ ...common.alternatives[0], debug: true }] } }
	];
	for (const body of malformed) fetchMock.enqueue(jsonResponse(200, body));
	for (const _body of malformed) await expect(getOptimizationJob(common.jobId)).rejects.toMatchObject({ appError: { code: "malformed_optimization_response" } });
});

test("secure random unavailability fails closed without weak fallback", () => {
	const originalCrypto = globalThis.crypto;
	Object.defineProperty(globalThis, "crypto", { configurable: true, value: undefined });
	try {
		expect(() => generateOptimizationIdempotencyKey()).toThrow(OptimizationClientError);
		expect(() => generateOptimizationIdempotencyKey()).toThrow("secure optimization request");
	} finally {
		Object.defineProperty(globalThis, "crypto", { configurable: true, value: originalCrypto });
	}
});

test("secure random generation accepts only canonical UUIDv4 provider output", () => {
	const originalCrypto = globalThis.crypto;
	const values: unknown[] = [
		undefined,
		null,
		"00000000-0000-0000-0000-000000000000",
		"00000000-0000-1000-8000-000000000000",
		"00000000-0000-4000-7000-000000000000",
		"00000000-0000-4000-8000-00000000000G",
		"abcdefab-cdef-4abc-8def-abcdefabcdef".toUpperCase(),
		{}
	];
	try {
		for (const value of values) {
			Object.defineProperty(globalThis, "crypto", { configurable: true, value: { randomUUID: () => value } });
			expect(() => generateOptimizationIdempotencyKey()).toThrow(OptimizationClientError);
		}
		Object.defineProperty(globalThis, "crypto", { configurable: true, value: { randomUUID: () => { throw new Error("provider failed"); } } });
		expect(() => generateOptimizationIdempotencyKey()).toThrow(OptimizationClientError);
		Object.defineProperty(globalThis, "crypto", { configurable: true, value: { randomUUID: () => "00000000-0000-4000-a000-000000000000" } });
		expect(generateOptimizationIdempotencyKey()).toBe("optimization-00000000-0000-4000-a000-000000000000");
	} finally {
		Object.defineProperty(globalThis, "crypto", { configurable: true, value: originalCrypto });
	}
});

test("submission requires a caller-owned key at compile time", () => {
	if (false) {
		// @ts-expect-error DESIGN-004 requires the user-operation boundary to own the key.
		void submitOptimization(request);
	}
	expect(submitOptimization.length).toBe(2);
});

test("submission fails before I/O when runtime JavaScript omits the caller-owned key", async () => {
	globalThis.fetch = fetchMock.fetch;
	const untypedSubmit = submitOptimization as unknown as (value: DietOptimizationRequest) => Promise<unknown>;

	await expect(untypedSubmit(request)).rejects.toMatchObject({ status: 0, appError: { code: "optimization_idempotency_key_required", retryable: false } });
	expect(fetchMock.calls).toHaveLength(0);
});

test("submission rejects weak or malformed caller keys before any I/O", async () => {
	globalThis.fetch = fetchMock.fetch;
	for (const idempotencyKey of ["abcdefgh", "optimization-key", "optimization-00000000-0000-1000-8000-000000000000", "optimization-00000000-0000-4000-7000-000000000000"]) {
		await expect(submitOptimization(request, { csrfToken: "csrf", idempotencyKey })).rejects.toMatchObject({
			status: 0,
			appError: { code: "optimization_idempotency_key_required", retryable: false }
		});
	}
	expect(fetchMock.calls).toHaveLength(0);
});

test("acknowledgement decoding rejects every malformed exact-contract boundary", async () => {
	globalThis.fetch = fetchMock.fetch;
	const valid = acknowledgement;
	const malformed = [
		null,
		[],
		{ ...valid, status: "ok" },
		{ ...valid, requestId: "" },
		{ ...valid, requestId: "unsafe request" },
		{ ...valid, extra: true },
		{ ...valid, data: null },
		{ ...valid, data: { ...valid.data!, jobId: "not-a-uuid" } },
		{ ...valid, data: { ...valid.data!, status: "processing" } },
		{ ...valid, data: { ...valid.data!, pollUrl: "//evil.example/jobs/1" } },
		{ ...valid, data: { ...valid.data!, pollUrl: `${valid.data!.pollUrl}/extra` } },
		{ ...valid, data: { ...valid.data!, debug: true } },
		{ ...valid, data: { jobId: valid.data!.jobId, status: "queued" } }
	];
	for (const body of malformed) fetchMock.enqueue(jsonResponse(202, body));
	for (const _body of malformed) {
		await expect(submitOptimization(request, { csrfToken: "csrf", idempotencyKey: submissionKey })).rejects.toMatchObject({
			status: 202,
			appError: { code: "malformed_optimization_response" }
		});
	}
});

test("job decoding table rejects missing, cross-variant, unsafe, and out-of-range values", async () => {
	globalThis.fetch = fetchMock.fetch;
	const common = {
		jobId: acknowledgement.data!.jobId,
		dailyDietId: request.dailyDietId,
		status: "completed",
		pollUrl: acknowledgement.data!.pollUrl,
		createdAt: "2026-07-11T00:00:00Z",
		startedAt: "2026-07-11T00:00:01Z",
		finishedAt: "2026-07-11T00:00:02Z",
		alternatives: [{ meals: [{ mealId, quantity: 100, unit: "g", position: 0 }], macros: { protein: 40, carbohydrates: 80, fat: 20, calories: 640 }, similarityScore: 0.8 }]
	};
	const alternative = common.alternatives[0]!;
	const malformed = [
		{ ...common, createdAt: "2026-02-30T00:00:00Z" },
		{ ...common, startedAt: null },
		{ ...common, finishedAt: "2026-07-11 00:00:02Z" },
		{ ...common, alternatives: [alternative, alternative, alternative, alternative] },
		{ ...common, alternatives: [{ ...alternative, similarityScore: -0.0001 }] },
		{ ...common, alternatives: [{ ...alternative, similarityScore: 1.0001 }] },
		{ ...common, alternatives: [{ ...alternative, similarityScore: null }] },
		{ ...common, alternatives: [{ ...alternative, meals: Array(101).fill(alternative.meals[0]) }] },
		{ ...common, alternatives: [{ ...alternative, meals: [{ ...alternative.meals[0], mealId: "bad" }] }] },
		{ ...common, alternatives: [{ ...alternative, meals: [{ ...alternative.meals[0], quantity: 0.0001 }] }] },
		{ ...common, alternatives: [{ ...alternative, meals: [{ ...alternative.meals[0], quantity: 1_000_001 }] }] },
		{ ...common, alternatives: [{ ...alternative, meals: [{ ...alternative.meals[0], quantity: null }] }] },
		{ ...common, alternatives: [{ ...alternative, meals: [{ ...alternative.meals[0], unit: "serving" }] }] },
		{ ...common, alternatives: [{ ...alternative, meals: [{ ...alternative.meals[0], position: 0.5 }] }] },
		{ ...common, alternatives: [{ ...alternative, meals: [{ ...alternative.meals[0], extra: true }] }] },
		{ ...common, alternatives: [{ ...alternative, macros: { ...alternative.macros, protein: 1_000_000_001 } }] },
		{ ...common, alternatives: [{ ...alternative, macros: { ...alternative.macros, calories: null } }] },
		{ ...common, alternatives: [{ ...alternative, macros: { ...alternative.macros, extra: 1 } }] },
		{ ...common, status: "failed", failure: { code: "solver_timeout", message: "redis://internal" } },
		{ ...common, status: "failed", alternatives: [], failure: { code: "solver_timeout", message: "Optimization took too long. Please try again.", debug: true } },
		{ ...common, status: "cancelled", alternatives: undefined }
	];
	for (const data of malformed) fetchMock.enqueue(jsonResponse(200, { status: "ok", requestId: "table-case", data }));
	for (const _data of malformed) {
		await expect(getOptimizationJob(common.jobId)).rejects.toMatchObject({ appError: { code: "malformed_optimization_response" } });
	}
});

test("polling rejects unknown terminal codes and out-of-contract similarity scores", async () => {
	globalThis.fetch = fetchMock.fetch;
	const common = {
		jobId: acknowledgement.data!.jobId,
		dailyDietId: request.dailyDietId,
		pollUrl: acknowledgement.data!.pollUrl,
		createdAt: "2026-07-11T00:00:00Z",
		finishedAt: "2026-07-11T00:00:02Z"
	};
	fetchMock.enqueue(jsonResponse(200, {
		status: "ok", requestId: "unknown-code",
		data: { ...common, status: "failed", failure: { code: "database_secret", message: "postgres://internal" } }
	}));
	fetchMock.enqueue(jsonResponse(200, {
		status: "ok", requestId: "empty-code",
		data: { ...common, status: "failed", failure: { code: "", message: "" } }
	}));
	fetchMock.enqueue(jsonResponse(200, {
		status: "ok", requestId: "bad-score",
		data: {
			...common, status: "completed", startedAt: "2026-07-11T00:00:01Z",
			alternatives: [{ meals: [{ mealId, quantity: 100, unit: "g", position: 0 }], macros: { protein: 40, carbohydrates: 80, fat: 20, calories: 640 }, similarityScore: 0.333333 }]
		}
	}));

	await expect(getOptimizationJob(common.jobId)).rejects.toMatchObject({ appError: { code: "malformed_optimization_response" } });
	await expect(getOptimizationJob(common.jobId)).rejects.toMatchObject({ appError: { code: "malformed_optimization_response" } });
	await expect(getOptimizationJob(common.jobId)).rejects.toMatchObject({ appError: { code: "malformed_optimization_response" } });
});
