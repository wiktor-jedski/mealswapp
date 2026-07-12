import { afterEach, expect, test } from "bun:test";

import type {
	DietOptimizationRequest,
	OptimizationJobAcknowledgementEnvelope,
	OptimizationJobStatusEnvelope
} from "./generated";
import { OptimizationClientError, getOptimizationJob, submitOptimization } from "./optimization-client";

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

const acknowledgement: OptimizationJobAcknowledgementEnvelope = {
	status: "accepted",
	requestId: "optimization-submit",
	data: { jobId: "00000000-0000-0000-0000-000000000002", status: "queued", pollUrl: "/api/v1/optimization/jobs/00000000-0000-0000-0000-000000000002" }
};

test("submit uses generated DTO JSON, cookies, CSRF, and exactly one caller-owned idempotency key", async () => {
	globalThis.fetch = fetchMock.fetch;
	fetchMock.enqueue(jsonResponse(202, acknowledgement));

	await submitOptimization(request, { csrfToken: "csrf-token", idempotencyKey: "optimization-key-1" });

	const call = fetchMock.calls[0];
	expect(call?.url).toBe("/api/v1/optimization/jobs");
	expect(call?.init.method).toBe("POST");
	expect(call?.init.credentials).toBe("include");
	expect(call?.init.headers).toEqual({
		Accept: "application/json",
		"Content-Type": "application/json",
		"Idempotency-Key": "optimization-key-1",
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
			alternatives: [{ meals: [{ mealId: "meal-1", quantity: 100, unit: "g", position: 0 }], macros: { protein: 40, carbohydrates: 80, fat: 20, calories: 640 }, similarityScore: 0.8 }]
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
			alternatives: [{ meals: [{ mealId: "meal-1", quantity: 100, unit: "g", position: 0 }], macros: { protein: 40, carbohydrates: 80, fat: 20, calories: 640 }, similarityScore: 0.8 }],
			failure: { code: "solver_timeout", message: "Optimization timed out." }
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

test("malformed successful job responses fail closed", async () => {
	globalThis.fetch = fetchMock.fetch;
	fetchMock.enqueue(jsonResponse(200, { status: "ok", requestId: "malformed", data: { status: "completed", alternatives: [] } }));

	await expect(getOptimizationJob("00000000-0000-0000-0000-000000000002")).rejects.toBeInstanceOf(OptimizationClientError);
});
