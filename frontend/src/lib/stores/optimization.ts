import { get, writable, type Writable } from "svelte/store";

import {
	generateOptimizationIdempotencyKey,
	optimizationApi,
	OptimizationClientError,
	type OptimizationApi
} from "../api/optimization-client";
import type {
	DietOptimizationRequest,
	OptimizationAlternative,
	OptimizationJobData
} from "../api/generated";

// Implements DESIGN-001 SearchView OptimizationWorkflow state and lifecycle.
// Implements DESIGN-004 JobStatusTracker bounded polling and safe retry semantics.

/** User-visible lifecycle phase for one saved-diet optimization workflow. */
export type OptimizationPhase = "idle" | "submitting" | "queued" | "processing" | "completed" | "failed" | "expired";

/** Retry action permitted after the latest submission or polling outcome. */
export type OptimizationRetryMode = "none" | "reuse" | "new_submission";

/** Safe optimization failure projected for display and retry decisions. */
export interface OptimizationDisplayError {
	/** Stable application error code. */
	code: string;
	/** User-safe message without backend or solver diagnostics. */
	message: string;
	/** Whether the current operation may be attempted again. */
	retryable: boolean;
}

/** Complete client state for the selected saved diet's optimization workflow. */
export interface OptimizationState {
	/** Saved-diet identifier that owns this state, or null when none is selected. */
	selectedDietId: string | null;
	/** Current workflow lifecycle phase. */
	phase: OptimizationPhase;
	/** Server-created job identifier after an accepted submission. */
	jobId: string | null;
	/** Latest contract-validated server job snapshot. */
	job: OptimizationJobData | null;
	/** At most three validated alternatives retained for display. */
	alternatives: OptimizationAlternative[];
	/** Current user-safe terminal or transport failure. */
	failure: OptimizationDisplayError | null;
	/** Retry action available for the current failure or terminal result. */
	retryMode: OptimizationRetryMode;
}

/** Creates empty workflow state scoped to an optional selected saved diet. */
export function createInitialOptimizationState(selectedDietId: string | null = null): OptimizationState {
	return {
		selectedDietId,
		phase: "idle",
		jobId: null,
		job: null,
		alternatives: [],
		failure: null,
		retryMode: "none"
	};
}

/** Shared writable optimization state consumed by the saved-diet workflow UI. */
export const optimizationStore = writable<OptimizationState>(createInitialOptimizationState());

/** Injectable dependencies and polling bounds for an optimization controller. */
export interface OptimizationControllerOptions {
	/** API boundary; defaults to the production optimization client. */
	api?: OptimizationApi;
	/** Writable state boundary; defaults to the shared optimization store. */
	store?: Writable<OptimizationState>;
	/** Bounded polling delay schedule in milliseconds. */
	pollDelaysMs?: readonly number[];
	/** Maximum status polls before returning a retryable timeout. */
	maxPolls?: number;
	/** Abort-aware delay function, injectable for deterministic tests. */
	sleep?: (delayMs: number, signal: AbortSignal) => Promise<void>;
	/** Creates caller-owned idempotency keys for new submissions. */
	createIdempotencyKey?: () => string;
}

/** Controls submission, polling, retry, and cancellation for one selected diet. */
export interface OptimizationController {
	/** Writable state updated throughout the workflow lifecycle. */
	store: Writable<OptimizationState>;
	/** Changes identity scope and clears any prior job, result, and pending retry. */
	setDiet(selectedDietId: string | null): void;
	/** Submits a new request with a fresh in-memory idempotency key and polls it. */
	submit(request: DietOptimizationRequest): Promise<void>;
	/** Performs the retry action allowed by the current state. */
	retry(): Promise<void>;
	/** Cancels active polling and releases controller-local pending state. */
	dispose(): void;
}

interface PendingSubmission {
	request: DietOptimizationRequest;
	idempotencyKey: string;
}

const DEFAULT_POLL_DELAYS_MS = [500, 1000, 2000, 4000, 8000, 10000] as const;
const DEFAULT_MAX_POLLS = 60;

/**
 * Creates one in-memory optimization controller whose pending idempotency key
 * is never persisted. Call `dispose` when its UI owner is destroyed.
 */
export function createOptimizationController(options: OptimizationControllerOptions = {}): OptimizationController {
	const api = options.api ?? optimizationApi;
	const store = options.store ?? optimizationStore;
	const pollDelaysMs = options.pollDelaysMs ?? DEFAULT_POLL_DELAYS_MS;
	const maxPolls = options.maxPolls ?? DEFAULT_MAX_POLLS;
	const sleep = options.sleep ?? wait;
	const createKey = options.createIdempotencyKey ?? generateOptimizationIdempotencyKey;
	let selectedDietId: string | null = get(store).selectedDietId;
	let pending: PendingSubmission | null = null;
	let operation = 0;
	let activeAbort: AbortController | null = null;

	function setDiet(nextDietId: string | null): void {
		if (nextDietId === selectedDietId) return;
		selectedDietId = nextDietId;
		operation += 1;
		activeAbort?.abort();
		activeAbort = null;
		pending = null;
		store.set(createInitialOptimizationState(nextDietId));
	}

	async function submit(request: DietOptimizationRequest): Promise<void> {
		if (get(store).phase === "submitting" || get(store).phase === "queued" || get(store).phase === "processing") return;
		if (request.dailyDietId !== selectedDietId) setDiet(request.dailyDietId);
		const submission: PendingSubmission = {
			request: {
				...request,
				excludedMealIds: [...request.excludedMealIds]
			},
			idempotencyKey: createKey()
		};
		pending = submission;
		return runSubmission(submission);
	}

	async function retry(): Promise<void> {
		if (!pending) return;
		const state = get(store);
		if (state.retryMode === "reuse" && state.jobId) {
			return pollExistingJob(state.jobId);
		}
		if (state.retryMode === "reuse") {
			return runSubmission(pending);
		}
		if (state.retryMode === "new_submission") {
			pending = { ...pending, idempotencyKey: createKey() };
			return runSubmission(pending);
		}
	}

	async function runSubmission(submission: PendingSubmission): Promise<void> {
		const token = beginOperation("submitting");
		try {
			const acknowledgement = await api.submitOptimization(submission.request, {
				idempotencyKey: submission.idempotencyKey,
				signal: activeAbort?.signal
			});
			if (!isCurrent(token)) return;
			store.update((state) => ({
				...state,
				phase: "queued",
				jobId: acknowledgement.jobId,
				job: null,
				alternatives: [],
				failure: null,
				retryMode: "none"
			}));
			await pollJob(acknowledgement.jobId, token);
		} catch (error) {
			if (!isCurrent(token) || isAbortError(error)) return;
			setSubmissionFailure(error);
		}
	}

	async function pollExistingJob(jobId: string): Promise<void> {
		const token = beginOperation("queued");
		try {
			await pollJob(jobId, token);
		} catch (error) {
			if (!isCurrent(token) || isAbortError(error)) return;
			setSubmissionFailure(error);
		}
	}

	async function pollJob(jobId: string, token: number): Promise<void> {
		for (let attempt = 0; attempt < maxPolls; attempt += 1) {
			await sleep(pollDelaysMs[Math.min(attempt, pollDelaysMs.length - 1)] ?? 10000, activeAbort?.signal ?? new AbortController().signal);
			if (!isCurrent(token)) return;
			const job = await api.getOptimizationJob(jobId, activeAbort?.signal);
			if (!isCurrent(token)) return;
			if (job.status === "queued") {
				store.update((state) => ({ ...state, phase: "queued", job, jobId, failure: null }));
				continue;
			}
			if (job.status === "processing") {
				store.update((state) => ({ ...state, phase: "processing", job, jobId, failure: null }));
				continue;
			}
			if (job.status === "completed") {
				store.update((state) => ({ ...state, phase: "completed", job, jobId, alternatives: [...job.alternatives].slice(0, 3), failure: null, retryMode: "new_submission" }));
				return;
			}
			if (job.status === "failed") {
				store.update((state) => ({
					...state,
					phase: "failed",
					job,
					jobId,
					alternatives: job.alternatives ? [...job.alternatives].slice(0, 3) : [],
					failure: displayFailure(job.failure.code, job.failure.message),
					retryMode: "new_submission"
				}));
				return;
			}
			store.update((state) => ({
				...state,
				phase: "failed",
				job,
				jobId,
				failure: displayFailure("cancelled", "This optimization was cancelled. Please try again."),
				retryMode: "new_submission"
			}));
			return;
		}
		throw new OptimizationClientError(
			{ category: "timeout", code: "optimization_poll_timeout", message: "Optimization is taking longer than expected. Please try again.", retryable: true },
			0
		);
	}

	function beginOperation(phase: OptimizationPhase): number {
		operation += 1;
		activeAbort?.abort();
		activeAbort = new AbortController();
		store.update((state) => ({
			...state,
			phase,
			jobId: phase === "submitting" ? null : state.jobId,
			job: phase === "submitting" ? null : state.job,
			alternatives: phase === "submitting" ? [] : state.alternatives,
			failure: null,
			retryMode: "none"
		}));
		return operation;
	}

	function isCurrent(token: number): boolean {
		return token === operation && !activeAbort?.signal.aborted;
	}

	function setSubmissionFailure(error: unknown): void {
		const failure = displayError(error);
		const reusable = failure.retryable && failure.code !== "result_expired" && failure.code !== "optimization_not_found";
		store.update((state) => ({ ...state, phase: failure.code === "result_expired" ? "expired" : "failed", failure, retryMode: reusable ? "reuse" : "new_submission" }));
	}

	function dispose(): void {
		operation += 1;
		activeAbort?.abort();
		activeAbort = null;
		pending = null;
	}

	return { store, setDiet, submit, retry, dispose };
}

function displayFailure(code: string, message: string, retryable = code !== "solver_infeasible"): OptimizationDisplayError {
	const friendlyMessages: Record<string, string> = {
		solver_timeout: "Optimization took too long. You can safely try again.",
		solver_infeasible: "No meal combination matched these macro targets. Try a wider tolerance.",
		queue_unavailable: "The optimization queue is temporarily unavailable. Please try again.",
		worker_crash: "Optimization could not be completed. Please try again.",
		result_expired: "This optimization result has expired. Submit again for a fresh result."
	};
	return { code, message: friendlyMessages[code] ?? message, retryable };
}

function displayError(error: unknown): OptimizationDisplayError {
	if (error instanceof OptimizationClientError) {
		return displayFailure(error.appError.code, error.appError.message, error.appError.retryable);
	}
	return { code: "optimization_request_failed", message: "Optimization could not be completed. Please try again.", retryable: true };
}

function isAbortError(error: unknown): boolean {
	return error instanceof DOMException && error.name === "AbortError";
}

function wait(delayMs: number, signal: AbortSignal): Promise<void> {
	return new Promise((resolve, reject) => {
		if (signal.aborted) {
			reject(signal.reason ?? new DOMException("Aborted", "AbortError"));
			return;
		}
		const timer = setTimeout(resolve, delayMs);
		const onAbort = () => {
			clearTimeout(timer);
			reject(signal.reason ?? new DOMException("Aborted", "AbortError"));
		};
		signal.addEventListener("abort", onAbort, { once: true });
	});
}
