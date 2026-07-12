import { expect, test } from "bun:test";
import { get, writable } from "svelte/store";

import { OptimizationClientError, type OptimizationApi } from "../api/optimization-client";
import type { DietOptimizationRequest, OptimizationJobData } from "../api/generated";
import {
	createInitialOptimizationState,
	createOptimizationController,
	type OptimizationState
} from "./optimization";

// Implements DESIGN-001 SearchView OptimizationWorkflow controller verification.
// Implements DESIGN-004 JobStatusTracker bounded polling, idempotency replay, terminal stop, and cancellation.

const request: DietOptimizationRequest = {
	dailyDietId: "diet-1",
	tolerancePercent: 10,
	excludedMealIds: []
};

function ack(jobId = "job-1") {
	return { jobId, status: "queued" as const, pollUrl: `/api/v1/optimization/jobs/${jobId}` };
}

function completed(jobId = "job-1"): OptimizationJobData {
	return {
		jobId,
		dailyDietId: request.dailyDietId,
		status: "completed",
		pollUrl: `/api/v1/optimization/jobs/${jobId}`,
		createdAt: "2026-07-11T00:00:00Z",
		startedAt: "2026-07-11T00:00:01Z",
		finishedAt: "2026-07-11T00:00:02Z",
		alternatives: [{ meals: [{ mealId: "meal-1", quantity: 100, unit: "g", position: 0 }], macros: { protein: 40, carbohydrates: 80, fat: 20, calories: 640 }, similarityScore: 0.8 }]
	};
}

function failed(jobId = "job-1", code: "solver_timeout" | "solver_infeasible" = "solver_timeout"): OptimizationJobData {
	return {
		jobId,
		dailyDietId: request.dailyDietId,
		status: "failed",
		pollUrl: `/api/v1/optimization/jobs/${jobId}`,
		createdAt: "2026-07-11T00:00:00Z",
		startedAt: null,
		finishedAt: "2026-07-11T00:00:02Z",
		alternatives: [],
		failure: { code, message: code === "solver_timeout" ? "solver timeout" : "infeasible" }
	};
}

function controller(api: OptimizationApi, keys = ["key-1", "key-2"]): { controller: ReturnType<typeof createOptimizationController>; store: ReturnType<typeof writable<OptimizationState>>; keysUsed: string[] } {
	const store = writable(createInitialOptimizationState("diet-1"));
	const keysUsed: string[] = [];
	let keyIndex = 0;
	return {
		store,
		keysUsed,
		controller: createOptimizationController({
			api,
			store,
			pollDelaysMs: [0],
			sleep: async () => undefined,
			createIdempotencyKey: () => {
				const key = keys[keyIndex++] ?? `key-${keyIndex}`;
				keysUsed.push(key);
				return key;
			}
		})
	};
}

test("submits once, polls with bounded delay, stops on completion, and exposes one to three validated alternatives", async () => {
	let submitCalls = 0;
	let pollCalls = 0;
	const api: OptimizationApi = {
		submitOptimization: async (_request, options) => {
			submitCalls += 1;
			expect(options.idempotencyKey).toBe("key-1");
			return ack();
		},
		getOptimizationJob: async () => {
			pollCalls += 1;
			return completed();
		}
	};
	const testController = controller(api);

	await testController.controller.submit(request);

	expect(submitCalls).toBe(1);
	expect(pollCalls).toBe(1);
	expect(get(testController.store)).toMatchObject({ phase: "completed", jobId: "job-1" });
	expect(get(testController.store).alternatives).toHaveLength(1);
});

test("retries an ambiguous submission with the exact same idempotency key and does not allocate another job", async () => {
	let submitCalls = 0;
	const seenKeys: string[] = [];
	const api: OptimizationApi = {
		submitOptimization: async (_request, options) => {
			submitCalls += 1;
			seenKeys.push(String(options.idempotencyKey));
			if (submitCalls === 1) {
				throw new OptimizationClientError({ category: "network", code: "optimization_network_error", message: "lost response", retryable: true }, 0);
			}
			return ack();
		},
		getOptimizationJob: async () => completed()
	};
	const testController = controller(api);

	await testController.controller.submit(request);
	expect(get(testController.store)).toMatchObject({ phase: "failed", retryMode: "reuse", failure: { retryable: true } });
	await testController.controller.retry();

	expect(submitCalls).toBe(2);
	expect(seenKeys).toEqual(["key-1", "key-1"]);
	expect(testController.keysUsed).toEqual(["key-1"]);
	expect(get(testController.store).phase).toBe("completed");
});

test("terminal infeasible and timeout states use safe messages and an explicit retry gets a fresh key", async () => {
	let submitCalls = 0;
	const api: OptimizationApi = {
		submitOptimization: async () => {
			submitCalls += 1;
			return ack(`job-${submitCalls}`);
		},
		getOptimizationJob: async (jobId) => submitCalls === 1 ? failed(jobId, "solver_infeasible") : completed(jobId)
	};
	const testController = controller(api);

	await testController.controller.submit(request);
	expect(get(testController.store)).toMatchObject({ phase: "failed", retryMode: "new_submission", failure: { code: "solver_infeasible", message: "No meal combination matched these macro targets. Try a wider tolerance." } });
	await testController.controller.retry();

	expect(testController.keysUsed).toEqual(["key-1", "key-2"]);
	expect(get(testController.store).phase).toBe("completed");
});

test("projects queue-unavailable and solver-timeout messages without exposing infrastructure details", async () => {
	let submitCalls = 0;
	const api: OptimizationApi = {
		submitOptimization: async () => {
			submitCalls += 1;
			if (submitCalls === 1) {
				throw new OptimizationClientError({ category: "dependency", code: "queue_unavailable", message: "redis://internal", retryable: true }, 503);
			}
			return ack();
		},
		getOptimizationJob: async () => failed("job-1", "solver_timeout")
	};
	const testController = controller(api);

	await testController.controller.submit(request);
	expect(get(testController.store)).toMatchObject({ phase: "failed", failure: { code: "queue_unavailable", message: "The optimization queue is temporarily unavailable. Please try again." } });
	await testController.controller.retry();
	expect(get(testController.store)).toMatchObject({ phase: "failed", failure: { code: "solver_timeout", message: "Optimization took too long. You can safely try again." } });
	expect(get(testController.store).failure?.message).not.toContain("redis://");
});

test("expired results retry as a fresh submission instead of polling the expired job again", async () => {
	let submitCalls = 0;
	let pollCalls = 0;
	const api: OptimizationApi = {
		submitOptimization: async () => {
			submitCalls += 1;
			return ack(`job-${submitCalls}`);
		},
		getOptimizationJob: async () => {
			pollCalls += 1;
			throw new OptimizationClientError({ category: "validation", code: "result_expired", message: "expired", retryable: true }, 410);
		}
	};
	const testController = controller(api);

	await testController.controller.submit(request);
	expect(get(testController.store)).toMatchObject({ phase: "expired", retryMode: "new_submission" });
	await testController.controller.retry();

	expect(submitCalls).toBe(2);
	expect(pollCalls).toBe(2);
	// The second request starts a fresh bounded poll and remains expired until its own result is available.
	expect(get(testController.store).phase).toBe("expired");
});

test("stops polling after a terminal state", async () => {
	let pollCalls = 0;
	const api: OptimizationApi = {
		submitOptimization: async () => ack(),
		getOptimizationJob: async () => {
			pollCalls += 1;
			return completed();
		}
	};
	const testController = controller(api);

	await testController.controller.submit(request);

	expect(pollCalls).toBe(1);
	expect(get(testController.store).phase).toBe("completed");

	const terminalApi: OptimizationApi = { submitOptimization: async () => ack(), getOptimizationJob: async () => completed() };
	const terminal = controller(terminalApi);
	await terminal.controller.submit(request);
	expect(get(terminal.store).phase).toBe("completed");
	terminal.controller.dispose();
	await terminal.controller.retry();
	expect(get(terminal.store).phase).toBe("completed");
});

test("unmount during an active poll aborts the request and ignores its late result", async () => {
	let pollCalls = 0;
	let pollSignal: AbortSignal | undefined;
	let releasePoll!: (job: OptimizationJobData) => void;
	const pendingPoll = new Promise<OptimizationJobData>((resolve) => { releasePoll = resolve; });
	const api: OptimizationApi = {
		submitOptimization: async () => ack(),
		getOptimizationJob: async (_jobId, signal) => {
			pollCalls += 1;
			pollSignal = signal;
			return pendingPoll;
		}
	};
	const testController = controller(api);
	const running = testController.controller.submit(request);
	await new Promise((resolve) => setTimeout(resolve, 0));
	const beforeLateResult = get(testController.store);
	testController.controller.dispose();
	expect(pollSignal?.aborted).toBe(true);
	releasePoll(completed());
	await running;

	expect(pollCalls).toBe(1);
	expect(get(testController.store)).toEqual(beforeLateResult);
	expect(get(testController.store).alternatives).toEqual([]);
});
