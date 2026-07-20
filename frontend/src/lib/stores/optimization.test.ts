import { expect, test } from "bun:test";
import { get, writable } from "svelte/store";

import {
	OptimizationClientError,
	getOptimizationJob,
	submitOptimization,
	type OptimizationApi
} from "../api/optimization-client";
import type { DietOptimizationRequest, OptimizationJobData } from "../api/generated";
import {
	clearOptimizationIdentity,
	createInitialOptimizationState,
	createOptimizationController,
	optimizationRetryAction,
	waitForOptimizationPoll,
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
		alternatives: [{ meals: [{ mealId: "meal-1", name: "Chicken Breast", quantity: 100, unit: "g", position: 0 }], macros: { protein: 40, carbohydrates: 80, fat: 20, calories: 640 }, similarityScore: 0.8 }]
	};
}

function active(jobId: string, status: "queued" | "processing"): OptimizationJobData {
	const common = {
		jobId,
		dailyDietId: request.dailyDietId,
		pollUrl: `/api/v1/optimization/jobs/${jobId}`,
		createdAt: "2026-07-11T00:00:00Z"
	};
	return status === "queued" ? { ...common, status } : { ...common, status, startedAt: "2026-07-11T00:00:01Z" };
}

function terminalFailure(code: "failed_validation" | "solver_timeout" | "solver_infeasible" | "worker_crash", jobId = "job-1"): OptimizationJobData {
	return {
		jobId,
		dailyDietId: request.dailyDietId,
		status: "failed",
		pollUrl: `/api/v1/optimization/jobs/${jobId}`,
		createdAt: "2026-07-11T00:00:00Z",
		startedAt: null,
		finishedAt: "2026-07-11T00:00:02Z",
		alternatives: [],
		failure: { code, message: code }
	};
}

function cancelled(jobId = "job-1"): OptimizationJobData {
	return {
		jobId,
		dailyDietId: request.dailyDietId,
		status: "cancelled",
		pollUrl: `/api/v1/optimization/jobs/${jobId}`,
		createdAt: "2026-07-11T00:00:00Z",
		finishedAt: "2026-07-11T00:00:02Z"
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

test("suppresses concurrent submits and rotates the key for deliberate intent after ambiguity", async () => {
	let release!: () => void;
	let submits = 0;
	const seenKeys: string[] = [];
	const first = new Promise<void>((resolve) => { release = resolve; });
	const api: OptimizationApi = {
		submitOptimization: async (_request, options) => {
			submits += 1;
			seenKeys.push(options.idempotencyKey);
			if (submits === 1) {
				await first;
				throw new OptimizationClientError({ category: "network", code: "optimization_network_error", message: "lost response", retryable: true }, 0);
			}
			return ack();
		},
		getOptimizationJob: async () => completed()
	};
	const testController = controller(api);
	const running = testController.controller.submit(request);
	await Promise.resolve();
	await testController.controller.submit(request);
	expect(submits).toBe(1);
	release();
	await running;

	await testController.controller.submit(request);
	expect(seenKeys).toEqual(["key-1", "key-2"]);
	expect(get(testController.store).phase).toBe("completed");
});

test("secure-random failure exposes a safe state without submitting", async () => {
	let submits = 0;
	const store = writable(createInitialOptimizationState("diet-1"));
	const testController = createOptimizationController({
		store,
		api: { submitOptimization: async () => { submits += 1; return ack(); }, getOptimizationJob: async () => completed() },
		createIdempotencyKey: () => { throw new OptimizationClientError({ category: "security", code: "secure_random_unavailable", message: "A secure optimization request could not be created. Please try again.", retryable: true }, 0); }
	});

	await testController.submit(request);

	expect(submits).toBe(0);
	expect(get(store)).toMatchObject({ phase: "failed", retryMode: "none", failure: { code: "secure_random_unavailable" } });
});

test("secure-random failure for direct fresh intent or policy retry clears a completed result", async () => {
	for (const action of ["submit", "retry"] as const) {
		let keyCalls = 0;
		const store = writable(createInitialOptimizationState("diet-1"));
		const testController = createOptimizationController({
			store,
			api: { submitOptimization: async () => ack(), getOptimizationJob: async () => completed() },
			pollDelaysMs: [0],
			sleep: async () => undefined,
			createIdempotencyKey: () => {
				keyCalls += 1;
				if (keyCalls === 1) return "key-1";
				throw new OptimizationClientError({ category: "security", code: "secure_random_unavailable", message: "A secure optimization request could not be created. Please try again.", retryable: true }, 0);
			}
		});

		await testController.submit(request);
		expect(get(store)).toMatchObject({ phase: "completed", jobId: "job-1" });
		await testController[action](request);

		expect(get(store)).toEqual({
			...createInitialOptimizationState("diet-1"),
			phase: "failed",
			failure: {
				code: "secure_random_unavailable",
				message: "A secure optimization request could not be created. Please try again.",
				retryable: true
			}
		});
		testController.dispose();
	}
});

test("authenticated teardown aborts pre-acknowledgement work and destroys its private retry key", async () => {
	const store = writable(createInitialOptimizationState("diet-1"));
	let submits = 0;
	let submitSignal: AbortSignal | undefined;
	const testController = createOptimizationController({
		store,
		api: {
			submitOptimization: async (_value, options) => {
				submits += 1;
				submitSignal = options.signal;
				return new Promise((_, reject) => options.signal?.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")), { once: true }));
			},
			getOptimizationJob: async () => completed()
		},
		createIdempotencyKey: () => "private-key"
	});
	testController.setIdentity("user-1");
	testController.setDiet("diet-1");
	const running = testController.submit(request);
	while (!submitSignal) await Promise.resolve();

	clearOptimizationIdentity(store);
	await running;
	await testController.retry(request);
	await testController.resume();

	expect(submitSignal.aborted).toBe(true);
	expect(submits).toBe(1);
	expect(get(store)).toEqual(createInitialOptimizationState());
	testController.dispose();
});

test("submission keys remain memory-only and clear with diet scope or disposal", async () => {
	let storageAccesses = 0;
	const originalWindow = globalThis.window;
	Object.defineProperty(globalThis, "window", {
		configurable: true,
		value: {
			get localStorage() { storageAccesses += 1; throw new Error("must not access localStorage"); },
			get sessionStorage() { storageAccesses += 1; throw new Error("must not access sessionStorage"); }
		}
	});
	try {
		const api: OptimizationApi = { submitOptimization: async () => ack(), getOptimizationJob: async () => completed() };
		const first = controller(api);
		await first.controller.submit(request);
		first.controller.setDiet("diet-2");
		first.controller.dispose();
		expect(storageAccesses).toBe(0);
		expect(get(first.store)).toEqual(createInitialOptimizationState("diet-2"));
	} finally {
		Object.defineProperty(globalThis, "window", { configurable: true, value: originalWindow });
	}
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

// Verifies IT-ARCH-004-008, ARCH-004, DESIGN-004 JobStatusTracker,
// DESIGN-017 RetryManager, and SW-REQ-006/SW-REQ-043/SW-REQ-080.
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

// Verifies IT-ARCH-004-006, ARCH-004, DESIGN-004 JobStatusTracker,
// DESIGN-017 ErrorMessageMapper, and SW-REQ-021/SW-REQ-080.
test("strict client rejection keeps malformed polling payloads out of store and rendering state", async () => {
	const originalFetch = globalThis.fetch;
	const responses = [
		new Response(JSON.stringify({
			status: "accepted",
			requestId: "submit-safe",
			data: {
				jobId: "00000000-0000-0000-0000-000000000002",
				status: "queued",
				pollUrl: "/api/v1/optimization/jobs/00000000-0000-0000-0000-000000000002"
			}
		}), { status: 202 }),
		new Response(JSON.stringify({
			status: "ok",
			requestId: "poll-safe",
			data: {
				jobId: "00000000-0000-0000-0000-000000000002",
				dailyDietId: "00000000-0000-0000-0000-000000000001",
				status: "completed",
				pollUrl: "/api/v1/optimization/jobs/00000000-0000-0000-0000-000000000002",
				createdAt: "2026-07-11T00:00:00Z",
				startedAt: "2026-07-11T00:00:01Z",
				finishedAt: "2026-07-11T00:00:02Z",
				alternatives: [{ meals: [], macros: { protein: 0, carbohydrates: 0, fat: 0, calories: 0 }, similarityScore: 0 }]
			}
		}), { status: 200 })
	];
	globalThis.fetch = async () => responses.shift() ?? new Response(null, { status: 500 });
	const store = writable(createInitialOptimizationState("00000000-0000-0000-0000-000000000001"));
	const strictApi: OptimizationApi = {
		submitOptimization: (value, options) => submitOptimization(value, { ...options, csrfToken: "csrf" }),
		getOptimizationJob
	};
	const testController = createOptimizationController({
		api: strictApi,
		store,
		pollDelaysMs: [0],
		sleep: async () => undefined,
		createIdempotencyKey: () => "optimization-00000000-0000-4000-8000-000000000004"
	});
	try {
		await testController.submit({
			dailyDietId: "00000000-0000-0000-0000-000000000001",
			tolerancePercent: 10,
			excludedMealIds: []
		});
		expect(get(store)).toMatchObject({
			phase: "failed",
			jobId: "00000000-0000-0000-0000-000000000002",
			job: null,
			alternatives: [],
			failure: { code: "malformed_optimization_response" }
		});
	} finally {
		testController.dispose();
		globalThis.fetch = originalFetch;
	}
});

test("defines an exhaustive stage-aware retry policy for ambiguous, polling, terminal, auth, and entitlement outcomes", () => {
	expect(optimizationRetryAction("submission", "optimization_network_error", true)).toBe("replay_submission");
	expect(optimizationRetryAction("submission", "queue_unavailable", true)).toBe("replay_submission");
	expect(optimizationRetryAction("poll", "optimization_network_error", true)).toBe("poll_job");
	expect(optimizationRetryAction("poll", "optimization_poll_timeout", true)).toBe("poll_job");
	expect(optimizationRetryAction("poll", "result_expired", true)).toBe("new_submission");
	expect(optimizationRetryAction("poll", "optimization_not_found", true)).toBe("new_submission");
	expect(optimizationRetryAction("terminal", "solver_timeout", true)).toBe("new_submission");
	expect(optimizationRetryAction("terminal", "solver_infeasible", false)).toBe("new_submission");
	expect(optimizationRetryAction("terminal", "worker_crash", true)).toBe("new_submission");
	expect(optimizationRetryAction("terminal", "cancelled", true)).toBe("new_submission");
	expect(optimizationRetryAction("terminal", "failed_validation", false)).toBe("none");
	expect(optimizationRetryAction("submission", "session_expired", false)).toBe("none");
	expect(optimizationRetryAction("submission", "entitlement_denied", false)).toBe("none");
});

test("poll ambiguity after acknowledgement retries only the exact acknowledged job", async () => {
	let submits = 0;
	const polled: string[] = [];
	const api: OptimizationApi = {
		submitOptimization: async () => { submits += 1; return ack("acknowledged-job"); },
		getOptimizationJob: async (jobId) => {
			polled.push(jobId);
			if (polled.length === 1) throw new OptimizationClientError({ category: "network", code: "optimization_network_error", message: "offline", retryable: true }, 0);
			return completed(jobId);
		}
	};
	const testController = controller(api);

	await testController.controller.submit(request);
	expect(get(testController.store)).toMatchObject({ jobId: "acknowledged-job", retryMode: "reuse" });
	await testController.controller.retry(request);

	expect(submits).toBe(1);
	expect(polled).toEqual(["acknowledged-job", "acknowledged-job"]);
});

test("a repeated polling outage remains retryable when the repoll itself fails", async () => {
	let polls = 0;
	const outage = new OptimizationClientError({ category: "network", code: "optimization_network_error", message: "offline", retryable: true }, 0);
	const testController = controller({
		submitOptimization: async () => ack(),
		getOptimizationJob: async () => { polls += 1; throw outage; }
	});
	await testController.controller.submit(request);
	await testController.controller.retry(request);
	expect(polls).toBe(2);
	expect(get(testController.store)).toMatchObject({ phase: "failed", retryMode: "reuse", failure: { code: "optimization_network_error" } });
});

test("an unknown submission exception becomes a bounded generic replayable failure", async () => {
	const testController = controller({
		submitOptimization: async () => { throw new Error("private detail"); },
		getOptimizationJob: async () => completed()
	});
	await testController.controller.submit(request);
	expect(get(testController.store)).toMatchObject({
		phase: "failed",
		retryMode: "reuse",
		failure: { code: "optimization_request_failed", message: "Optimization could not be completed. Please try again." }
	});
});

test("bounded polling times out after the exact configured polls with a stable capped delay schedule", async () => {
	const delays: number[] = [];
	let polls = 0;
	const store = writable(createInitialOptimizationState("diet-1"));
	const testController = createOptimizationController({
		store,
		api: {
			submitOptimization: async () => ack(),
			getOptimizationJob: async () => { polls += 1; return active("job-1", polls === 1 ? "queued" : "processing"); }
		},
		pollDelaysMs: [5, 10],
		maxPolls: 4,
		sleep: async (delay) => { delays.push(delay); },
		createIdempotencyKey: () => "key-1"
	});

	await testController.submit(request);

	expect(polls).toBe(4);
	expect(delays).toEqual([5, 10, 10, 10]);
	expect(get(store)).toMatchObject({ phase: "failed", jobId: "job-1", retryMode: "reuse", failure: { code: "optimization_poll_timeout" } });
});

test("queue outage before acknowledgement replays the key while queue outage after acknowledgement only repolls", async () => {
	let submitCalls = 0;
	let pollCalls = 0;
	const seenKeys: string[] = [];
	const api: OptimizationApi = {
		submitOptimization: async (_value, options) => {
			submitCalls += 1;
			seenKeys.push(options.idempotencyKey);
			if (submitCalls === 1) throw new OptimizationClientError({ category: "dependency", code: "queue_unavailable", message: "queue", retryable: true }, 503);
			return ack();
		},
		getOptimizationJob: async () => {
			pollCalls += 1;
			if (pollCalls === 1) throw new OptimizationClientError({ category: "dependency", code: "queue_unavailable", message: "queue", retryable: true }, 503);
			return completed();
		}
	};
	const testController = controller(api);

	await testController.controller.submit(request);
	await testController.controller.retry(request);
	expect(get(testController.store)).toMatchObject({ jobId: "job-1", retryMode: "reuse" });
	await testController.controller.retry(request);

	expect(seenKeys).toEqual(["key-1", "key-1"]);
	expect(submitCalls).toBe(2);
	expect(pollCalls).toBe(2);
});

test("expiry and not-found polling outcomes always rotate to a fresh job and key", async () => {
	for (const code of ["result_expired", "optimization_not_found"] as const) {
		let submits = 0;
		const seenKeys: string[] = [];
		const api: OptimizationApi = {
			submitOptimization: async (_value, options) => { submits += 1; seenKeys.push(options.idempotencyKey); return ack(`job-${submits}`); },
			getOptimizationJob: async (jobId) => {
				if (jobId === "job-1") throw new OptimizationClientError({ category: "validation", code, message: code, retryable: true }, code === "result_expired" ? 410 : 404);
				return completed(jobId);
			}
		};
		const testController = controller(api);
		await testController.controller.submit(request);
		await testController.controller.retry(request);
		expect(seenKeys).toEqual(["key-1", "key-2"]);
		expect(get(testController.store)).toMatchObject({ phase: "completed", jobId: "job-2" });
	}
});

test("validation, authentication, and entitlement failures hide retry while preserving safe state", async () => {
	for (const failure of [
		{ category: "validation", code: "optimization_invalid_request", status: 400 },
		{ category: "auth", code: "session_expired", status: 401 },
		{ category: "entitlement", code: "entitlement_denied", status: 403 }
	] as const) {
		let submits = 0;
		const api: OptimizationApi = {
			submitOptimization: async () => {
				submits += 1;
				throw new OptimizationClientError({ ...failure, message: "safe", retryable: false }, failure.status);
			},
			getOptimizationJob: async () => completed()
		};
		const testController = controller(api);
		await testController.controller.submit(request);
		await testController.controller.retry(request);
		expect(submits).toBe(1);
		expect(get(testController.store)).toMatchObject({ phase: "failed", retryMode: "none", failure: { code: failure.code } });
	}
});

test("terminal validation hides retry while infeasible, timeout, cancellation, and worker failure rotate", async () => {
	for (const outcome of ["failed_validation", "solver_infeasible", "solver_timeout", "cancelled", "worker_crash"] as const) {
		let submits = 0;
		const keys: string[] = [];
		const api: OptimizationApi = {
			submitOptimization: async (_value, options) => { submits += 1; keys.push(options.idempotencyKey); return ack(`job-${submits}`); },
			getOptimizationJob: async (jobId) => submits === 1
				? outcome === "cancelled" ? cancelled(jobId) : terminalFailure(outcome, jobId)
				: completed(jobId)
		};
		const testController = controller(api);
		await testController.controller.submit(request);
		const expectedMode = outcome === "failed_validation" ? "none" : "new_submission";
		expect(get(testController.store).retryMode).toBe(expectedMode);
		await testController.controller.retry(request);
		expect(keys).toEqual(outcome === "failed_validation" ? ["key-1"] : ["key-1", "key-2"]);
	}
});

test("edited tolerance or exclusions convert replay and repoll actions into fresh current-input submissions", async () => {
	for (const acknowledged of [false, true]) {
		let submits = 0;
		const submitted: Array<{ request: DietOptimizationRequest; key: string }> = [];
		const api: OptimizationApi = {
			submitOptimization: async (value, options) => {
				submits += 1;
				submitted.push({ request: value, key: options.idempotencyKey });
				if (!acknowledged && submits === 1) throw new OptimizationClientError({ category: "network", code: "optimization_network_error", message: "offline", retryable: true }, 0);
				return ack(`job-${submits}`);
			},
			getOptimizationJob: async (jobId) => {
				if (acknowledged && submits === 1) throw new OptimizationClientError({ category: "network", code: "optimization_network_error", message: "offline", retryable: true }, 0);
				return completed(jobId);
			}
		};
		const testController = controller(api);
		const edited = { ...request, tolerancePercent: 25, excludedMealIds: ["meal-2"] };
		await testController.controller.submit(request);
		await testController.controller.retry(edited);
		expect(submitted.map(({ key }) => key)).toEqual(["key-1", "key-2"]);
		expect(submitted[1]?.request).toEqual(edited);
	}
});

test("dispose and remount during submission replays the exact memory-only request and key", async () => {
	const store = writable(createInitialOptimizationState("diet-1"));
	const keys: string[] = [];
	let firstSignal: AbortSignal | undefined;
	const api: OptimizationApi = {
		submitOptimization: async (_value, options) => {
			keys.push(options.idempotencyKey);
			if (keys.length === 1) {
				firstSignal = options.signal;
				return new Promise((_, reject) => options.signal?.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")), { once: true }));
			}
			return ack();
		},
		getOptimizationJob: async () => completed()
	};
	const first = createOptimizationController({ api, store, pollDelaysMs: [0], sleep: async () => undefined, createIdempotencyKey: () => "key-1" });
	const running = first.submit(request);
	await Promise.resolve();
	first.dispose();
	await running;
	expect(firstSignal?.aborted).toBe(true);
	expect(get(store)).toMatchObject({ phase: "failed", retryMode: "reuse" });

	const remount = createOptimizationController({ api, store, pollDelaysMs: [0], sleep: async () => undefined, createIdempotencyKey: () => "unused" });
	await remount.resume();
	expect(keys).toEqual(["key-1", "key-1"]);
	expect(get(store).phase).toBe("completed");
});

test("dispose and remount during queued or processing resumes the same job without resubmission", async () => {
	for (const status of ["queued", "processing"] as const) {
		const store = writable(createInitialOptimizationState("diet-1"));
		let submits = 0;
		let polls = 0;
		let release!: (job: OptimizationJobData) => void;
		const api: OptimizationApi = {
			submitOptimization: async () => { submits += 1; return ack(`${status}-job`); },
			getOptimizationJob: async (jobId) => {
				polls += 1;
				if (polls === 1) return active(jobId, status);
				if (polls === 2) return new Promise((resolve) => { release = resolve; });
				return completed(jobId);
			}
		};
		const first = createOptimizationController({ api, store, pollDelaysMs: [0], sleep: async () => undefined, createIdempotencyKey: () => "key-1" });
		const running = first.submit(request);
		while (polls < 2) await Promise.resolve();
		first.dispose();
		release(completed(`${status}-job`));
		await running;
		expect(get(store).phase).toBe(status);

		const remount = createOptimizationController({ api, store, pollDelaysMs: [0], sleep: async () => undefined });
		await remount.resume();
		expect(submits).toBe(1);
		expect(get(store)).toMatchObject({ phase: "completed", jobId: `${status}-job` });
	}
});

// Verifies IT-ARCH-004-006, ARCH-004, DESIGN-001 SearchView,
// DESIGN-017 RetryManager, and SW-REQ-006/SW-REQ-043/SW-REQ-080.
test("logout and account switch abort and clear job, results, errors, keys, polls, and retry intent", async () => {
	const store = writable(createInitialOptimizationState());
	let submits = 0;
	let pollSignal: AbortSignal | undefined;
	const keys = ["key-user-1", "key-user-2"];
	const testController = createOptimizationController({
		store,
		api: {
			submitOptimization: async () => { submits += 1; return ack(`job-${submits}`); },
			getOptimizationJob: async (_jobId, signal) => {
				pollSignal = signal;
				return new Promise((_, reject) => signal?.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")), { once: true }));
			}
		},
		pollDelaysMs: [0],
		sleep: async () => undefined,
		createIdempotencyKey: () => keys.shift()!
	});
	testController.setIdentity("user-1");
	testController.setDiet("diet-1");
	const running = testController.submit(request);
	while (!pollSignal) await Promise.resolve();
	clearOptimizationIdentity(store);
	await running;
	expect(pollSignal.aborted).toBe(true);
	expect(get(store)).toEqual(createInitialOptimizationState(null));
	await testController.retry(request);
	expect(submits).toBe(1);

	testController.setIdentity("user-2");
	testController.setDiet("diet-1");
	expect(get(store)).toEqual(createInitialOptimizationState("diet-1"));
	store.set({
		...createInitialOptimizationState("diet-1"),
		phase: "completed",
		jobId: "old-job",
		job: completed("old-job"),
		alternatives: completed("old-job").alternatives,
		failure: { code: "old-error", message: "old error", retryable: true },
		retryMode: "new_submission"
	});
	testController.setIdentity("user-3");
	expect(get(store)).toEqual(createInitialOptimizationState(null));
});

test("two controllers sharing one store cannot race or overwrite the active owner", async () => {
	const store = writable(createInitialOptimizationState("diet-1"));
	let submits = 0;
	let release!: () => void;
	const blocked = new Promise<void>((resolve) => { release = resolve; });
	const api: OptimizationApi = {
		submitOptimization: async () => { submits += 1; await blocked; return ack(); },
		getOptimizationJob: async () => completed()
	};
	const first = createOptimizationController({ api, store, pollDelaysMs: [0], sleep: async () => undefined, createIdempotencyKey: () => "key-1" });
	const second = createOptimizationController({ api, store, pollDelaysMs: [0], sleep: async () => undefined, createIdempotencyKey: () => "key-2" });
	first.setIdentity("user-1");
	first.setDiet("diet-1");
	const running = first.submit(request);
	await Promise.resolve();
	await second.submit(request);
	expect(submits).toBe(1);
	release();
	await running;
	expect(get(store)).toMatchObject({ phase: "completed", jobId: "job-1" });
});

test("a mounted successor receives ownership and resumes the acknowledged job after owner disposal", async () => {
	const store = writable(createInitialOptimizationState("diet-1"));
	let polls = 0;
	let releaseFirstPoll!: (job: OptimizationJobData) => void;
	const api: OptimizationApi = {
		submitOptimization: async () => ack("handoff-job"),
		getOptimizationJob: async (jobId) => {
			polls += 1;
			if (polls === 1) return new Promise((resolve) => { releaseFirstPoll = resolve; });
			return completed(jobId);
		}
	};
	const first = createOptimizationController({ api, store, pollDelaysMs: [0], sleep: async () => undefined, createIdempotencyKey: () => "key-1" });
	const successor = createOptimizationController({ api, store, pollDelaysMs: [0], sleep: async () => undefined, createIdempotencyKey: () => "key-2" });
	first.setIdentity("user-1");
	first.setDiet("diet-1");
	const running = first.submit(request);
	while (polls < 1) await Promise.resolve();
	successor.setIdentity("user-1");
	successor.setDiet("diet-1");
	await successor.resume();

	first.dispose();
	releaseFirstPoll(active("handoff-job", "queued"));
	await running;
	for (let attempt = 0; attempt < 10 && get(store).phase !== "completed"; attempt += 1) await Promise.resolve();

	expect(polls).toBe(2);
	expect(get(store)).toMatchObject({ phase: "completed", jobId: "handoff-job" });
	successor.dispose();
});

test("rejects empty, invalid, non-finite, and excessive polling configuration before creating a controller", () => {
	const invalid: OptimizationControllerOptionsForTest[] = [
		{ pollDelaysMs: [] },
		{ pollDelaysMs: [-1] },
		{ pollDelaysMs: [Number.NaN] },
		{ pollDelaysMs: [Number.POSITIVE_INFINITY] },
		{ pollDelaysMs: [60_001] },
		{ maxPolls: 0 },
		{ maxPolls: 1.5 },
		{ maxPolls: 1_001 },
		{ pollDelaysMs: [60_000], maxPolls: 11 }
	];
	for (const options of invalid) expect(() => createOptimizationController(options)).toThrow(RangeError);
});

type OptimizationControllerOptionsForTest = Parameters<typeof createOptimizationController>[0];

test("abortable poll delay removes listeners and settles exactly once on timer and abort paths", async () => {
	const originalSetTimeout = globalThis.setTimeout;
	const originalClearTimeout = globalThis.clearTimeout;
	let timerCallback: (() => void) | undefined;
	let clears = 0;
	globalThis.setTimeout = ((callback: TimerHandler) => { timerCallback = callback as () => void; return 1; }) as typeof setTimeout;
	globalThis.clearTimeout = (() => { clears += 1; }) as typeof clearTimeout;
	try {
		for (const settleBy of ["timer", "abort"] as const) {
			const listeners = new Set<EventListenerOrEventListenerObject>();
			let removes = 0;
			const signal = {
				aborted: false,
				reason: new DOMException("Aborted", "AbortError"),
				addEventListener: (_type: string, listener: EventListenerOrEventListenerObject) => listeners.add(listener),
				removeEventListener: (_type: string, listener: EventListenerOrEventListenerObject) => { removes += 1; listeners.delete(listener); }
			} as unknown as AbortSignal;
			const delay = waitForOptimizationPoll(10, signal);
			expect(listeners.size).toBe(1);
			if (settleBy === "timer") timerCallback?.();
			else for (const listener of [...listeners]) typeof listener === "function" ? listener(new Event("abort")) : listener.handleEvent(new Event("abort"));
			if (settleBy === "timer") await expect(delay).resolves.toBeUndefined();
			else await expect(delay).rejects.toMatchObject({ name: "AbortError" });
			timerCallback?.();
			expect(removes).toBe(1);
			expect(listeners.size).toBe(0);
		}
		expect(clears).toBe(2);
	} finally {
		globalThis.setTimeout = originalSetTimeout;
		globalThis.clearTimeout = originalClearTimeout;
	}
});

test("abortable poll delay rejects an already-aborted signal without installing a listener", async () => {
	let listeners = 0;
	const signal = {
		aborted: true,
		reason: undefined,
		addEventListener: () => { listeners += 1; },
		removeEventListener: () => undefined
	} as unknown as AbortSignal;
	await expect(waitForOptimizationPoll(10, signal)).rejects.toMatchObject({ name: "AbortError" });
	expect(listeners).toBe(0);
});
