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
	OptimizationFailureCode,
	OptimizationJobData
} from "../api/generated";

// Implements DESIGN-017 RetryManager policy, timing, controller ownership, and identity lifecycle.
// Implements DESIGN-004 JobStatusTracker bounded polling and safe retry semantics.

/** User-visible lifecycle phase for one saved-diet optimization workflow. */
export type OptimizationPhase = "idle" | "submitting" | "queued" | "processing" | "completed" | "failed" | "expired";

/** Retry action exposed to the optimization workflow. */
export type OptimizationRetryMode = "none" | "reuse" | "new_submission";

/** Exact internal action selected by the DESIGN-017 retry policy. */
export type OptimizationRetryAction = "none" | "replay_submission" | "poll_job" | "new_submission";

/** Operation boundary where an optimization outcome occurred. */
export type OptimizationFailureStage = "submission" | "poll" | "terminal";

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
	return { selectedDietId, phase: "idle", jobId: null, job: null, alternatives: [], failure: null, retryMode: "none" };
}

/** Shared writable optimization state consumed by the saved-diet workflow UI. */
export const optimizationStore = writable<OptimizationState>(createInitialOptimizationState());

/** Injectable dependencies and polling bounds for an optimization controller. */
export interface OptimizationControllerOptions {
	/** API boundary; defaults to the production optimization client. */
	api?: OptimizationApi;
	/** Writable state boundary; defaults to the shared optimization store. */
	store?: Writable<OptimizationState>;
	/** Non-empty finite delay schedule bounded to one minute per delay. */
	pollDelaysMs?: readonly number[];
	/** Positive integer poll limit bounded to 1,000 and ten minutes total. */
	maxPolls?: number;
	/** Abort-aware delay function, injectable for deterministic tests. */
	sleep?: (delayMs: number, signal: AbortSignal) => Promise<void>;
	/** Creates caller-owned idempotency keys for new submissions. */
	createIdempotencyKey?: () => string;
}

/** Controls submission, polling, retry, remount recovery, identity scope, and cancellation. */
export interface OptimizationController {
	store: Writable<OptimizationState>;
	/** Changes authenticated ownership and clears every previous user's optimization artifact. */
	setIdentity(identityId: string | null): void;
	/** Changes selected-diet scope and clears the previous diet's optimization artifacts. */
	setDiet(selectedDietId: string | null): void;
	/** Resumes an interrupted submission or acknowledged job after a controller remount. */
	resume(): Promise<void>;
	/** Submits the current request with a fresh in-memory idempotency key. */
	submit(request: DietOptimizationRequest): Promise<void>;
	/** Applies policy using current form input, rotating intent when that input changed. */
	retry(currentRequest?: DietOptimizationRequest): Promise<void>;
	/** Relinquishes ownership while preserving enough memory-only intent for a safe remount. */
	dispose(): void;
}

interface PendingSubmission {
	request: DietOptimizationRequest;
	idempotencyKey: string | null;
}

interface SharedOptimizationRuntime {
	identityId: string | null;
	selectedDietId: string | null;
	pending: PendingSubmission | null;
	retryAction: OptimizationRetryAction;
	owner: symbol | null;
	controllers: Map<symbol, OptimizationControllerRegistration>;
	operation: number;
	activeAbort: AbortController | null;
}

interface OptimizationControllerRegistration {
	clearScope(): void;
	resume(): Promise<void>;
}

const DEFAULT_POLL_DELAYS_MS = [500, 1000, 2000, 4000, 8000, 10000] as const;
const DEFAULT_MAX_POLLS = 60;
const MAX_POLL_DELAY_MS = 60_000;
const MAX_POLLS = 1_000;
const MAX_TOTAL_POLL_WAIT_MS = 600_000;
const runtimes = new WeakMap<Writable<OptimizationState>, SharedOptimizationRuntime>();

/** Clears protected optimization state and private retry material at an authenticated identity boundary. */
export function clearOptimizationIdentity(store: Writable<OptimizationState> = optimizationStore): void {
	const runtime = runtimeFor(store);
	runtime.operation += 1;
	runtime.activeAbort?.abort();
	runtime.activeAbort = null;
	runtime.identityId = null;
	runtime.selectedDietId = null;
	runtime.pending = null;
	runtime.retryAction = "none";
	for (const controller of runtime.controllers.values()) controller.clearScope();
	store.set(createInitialOptimizationState());
}

/**
 * Maps every accepted optimization failure code and stage to one explicit action.
 * Submission transport failures are ambiguous and replay the exact key; polling
 * transport failures keep polling the acknowledged job; terminal solver/worker
 * failures and missing results require a fresh submission. Caller input changes
 * are handled separately by {@link OptimizationController.retry}.
 */
export function optimizationRetryAction(
	stage: OptimizationFailureStage,
	code: string,
	retryable: boolean
): OptimizationRetryAction {
	if (code === "unauthorized" || code === "session_expired" || code === "entitlement_denied") return "none";
	if (stage === "terminal") {
		if (code === "solver_timeout" || code === "solver_infeasible" || code === "worker_crash" || code === "cancelled") return "new_submission";
		return "none";
	}
	if (code === "result_expired" || code === "optimization_not_found" || code === "not_found") return "new_submission";
	if (!retryable) return "none";
	return stage === "submission" ? "replay_submission" : "poll_job";
}

/** Creates a controller coordinated with every other controller using the same store. */
export function createOptimizationController(options: OptimizationControllerOptions = {}): OptimizationController {
	const api = options.api ?? optimizationApi;
	const store = options.store ?? optimizationStore;
	const pollDelaysMs = [...(options.pollDelaysMs ?? DEFAULT_POLL_DELAYS_MS)];
	const maxPolls = options.maxPolls ?? DEFAULT_MAX_POLLS;
	validatePollingConfiguration(pollDelaysMs, maxPolls);
	const sleep = options.sleep ?? waitForOptimizationPoll;
	const createKey = options.createIdempotencyKey ?? generateOptimizationIdempotencyKey;
	const owner = Symbol("optimization-controller");
	const runtime = runtimeFor(store);
	let desiredIdentityId: string | null | undefined;
	let desiredDietId: string | null | undefined;

	function claim(): boolean {
		if (runtime.owner === null) runtime.owner = owner;
		return runtime.owner === owner;
	}

	function setIdentity(identityId: string | null): void {
		desiredIdentityId = identityId;
		if (!claim() || identityId === runtime.identityId) return;
		resetScope(identityId, null);
	}

	function setDiet(selectedDietId: string | null): void {
		desiredDietId = selectedDietId;
		if (!claim() || selectedDietId === runtime.selectedDietId) return;
		resetScope(runtime.identityId, selectedDietId);
	}

	function resetScope(identityId: string | null, selectedDietId: string | null): void {
		cancelOperation();
		runtime.identityId = identityId;
		runtime.selectedDietId = selectedDietId;
		runtime.pending = null;
		runtime.retryAction = "none";
		store.set(createInitialOptimizationState(selectedDietId));
	}

	async function resume(): Promise<void> {
		if (!claim()) return;
		if (desiredIdentityId !== undefined && desiredIdentityId !== runtime.identityId) resetScope(desiredIdentityId, null);
		if (desiredDietId !== undefined && desiredDietId !== runtime.selectedDietId) resetScope(runtime.identityId, desiredDietId);
		return resumeOwned();
	}

	async function resumeOwned(): Promise<void> {
		const state = get(store);
		if ((state.phase === "submitting" || runtime.retryAction === "replay_submission") && runtime.pending?.idempotencyKey) return runSubmission(runtime.pending);
		if ((state.phase === "queued" || state.phase === "processing") && state.jobId) return pollExistingJob(state.jobId);
	}

	async function submit(request: DietOptimizationRequest): Promise<void> {
		if (!claim()) return;
		const state = get(store);
		if (state.phase === "submitting" || state.phase === "queued" || state.phase === "processing") return;
		if (request.dailyDietId !== runtime.selectedDietId) resetScope(runtime.identityId, request.dailyDietId);
		cancelOperation();
		runtime.pending = null;
		runtime.retryAction = "none";
		store.set(createInitialOptimizationState(runtime.selectedDietId));
		let idempotencyKey: string;
		try {
			idempotencyKey = createKey();
		} catch (error) {
			const failure = displayError(error);
			store.update((current) => ({ ...current, phase: "failed", failure, retryMode: "none" }));
			return;
		}
		const submission = snapshotSubmission(request, idempotencyKey);
		runtime.pending = submission;
		return runSubmission(submission);
	}

	async function retry(currentRequest?: DietOptimizationRequest): Promise<void> {
		if (!claim() || runtime.retryAction === "none" || !runtime.pending) return;
		const request = currentRequest ? snapshotRequest(currentRequest) : runtime.pending.request;
		if (!sameRequest(request, runtime.pending.request)) return submitFresh(request);
		if (runtime.retryAction === "replay_submission" && runtime.pending.idempotencyKey) return runSubmission(runtime.pending);
		if (runtime.retryAction === "poll_job" && get(store).jobId) return pollExistingJob(get(store).jobId!);
		if (runtime.retryAction === "new_submission") return submitFresh(request);
	}

	async function submitFresh(request: DietOptimizationRequest): Promise<void> {
		return submit(request);
	}

	async function runSubmission(submission: PendingSubmission): Promise<void> {
		if (!submission.idempotencyKey) return;
		const token = beginOperation("submitting");
		try {
			const acknowledgement = await api.submitOptimization(submission.request, {
				idempotencyKey: submission.idempotencyKey,
				signal: runtime.activeAbort?.signal
			});
			if (!isCurrent(token)) return;
			if (runtime.pending === submission) runtime.pending = { ...submission, idempotencyKey: null };
			runtime.retryAction = "none";
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
			setOperationFailure(error, get(store).jobId ? "poll" : "submission");
		}
	}

	async function pollExistingJob(jobId: string): Promise<void> {
		const token = beginOperation("queued");
		try {
			await pollJob(jobId, token);
		} catch (error) {
			if (!isCurrent(token) || isAbortError(error)) return;
			setOperationFailure(error, "poll");
		}
	}

	async function pollJob(jobId: string, token: number): Promise<void> {
		for (let attempt = 0; attempt < maxPolls; attempt += 1) {
			await sleep(pollDelaysMs[Math.min(attempt, pollDelaysMs.length - 1)]!, runtime.activeAbort!.signal);
			if (!isCurrent(token)) return;
			const job = await api.getOptimizationJob(jobId, runtime.activeAbort!.signal);
			if (!isCurrent(token)) return;
			if (job.status === "queued" || job.status === "processing") {
				store.update((state) => ({ ...state, phase: job.status, job, jobId, failure: null, retryMode: "none" }));
				continue;
			}
			if (job.status === "completed") {
				runtime.retryAction = "new_submission";
				store.update((state) => ({ ...state, phase: "completed", job, jobId, alternatives: [...job.alternatives].slice(0, 3), failure: null, retryMode: "new_submission" }));
				return;
			}
			if (job.status === "failed") {
				const failure = displayFailure(job.failure.code, job.failure.message, job.failure.code !== "solver_infeasible" && job.failure.code !== "failed_validation");
				runtime.retryAction = optimizationRetryAction("terminal", failure.code, failure.retryable);
				store.update((state) => ({
					...state,
					phase: "failed",
					job,
					jobId,
					alternatives: job.alternatives ? [...job.alternatives].slice(0, 3) : [],
					failure,
					retryMode: retryMode(runtime.retryAction)
				}));
				return;
			}
			const failure = displayFailure("cancelled", "This optimization was cancelled. Please try again.");
			runtime.retryAction = optimizationRetryAction("terminal", failure.code, failure.retryable);
			store.update((state) => ({ ...state, phase: "failed", job, jobId, failure, retryMode: retryMode(runtime.retryAction) }));
			return;
		}
		throw new OptimizationClientError(
			{ category: "timeout", code: "optimization_poll_timeout", message: "Optimization is taking longer than expected. Please try again.", retryable: true },
			0
		);
	}

	function beginOperation(phase: OptimizationPhase): number {
		cancelOperation();
		runtime.activeAbort = new AbortController();
		store.update((state) => ({
			...state,
			phase,
			jobId: phase === "submitting" ? null : state.jobId,
			job: phase === "submitting" ? null : state.job,
			alternatives: phase === "submitting" ? [] : state.alternatives,
			failure: null,
			retryMode: "none"
		}));
		return runtime.operation;
	}

	function cancelOperation(): void {
		runtime.operation += 1;
		runtime.activeAbort?.abort();
		runtime.activeAbort = null;
	}

	function isCurrent(token: number): boolean {
		return runtime.owner === owner && token === runtime.operation && runtime.activeAbort?.signal.aborted === false;
	}

	function setOperationFailure(error: unknown, stage: "submission" | "poll"): void {
		const failure = displayError(error);
		runtime.retryAction = optimizationRetryAction(stage, failure.code, failure.retryable);
		store.update((state) => ({
			...state,
			phase: failure.code === "result_expired" ? "expired" : "failed",
			failure,
			retryMode: retryMode(runtime.retryAction)
		}));
	}

	function dispose(): void {
		runtime.controllers.delete(owner);
		if (runtime.owner !== owner) return;
		cancelOperation();
		const state = get(store);
		if (state.phase === "submitting") {
			runtime.retryAction = runtime.pending?.idempotencyKey ? "replay_submission" : "none";
			store.update((current) => ({
				...current,
				phase: "failed",
				failure: displayFailure("optimization_interrupted", "Optimization was interrupted. Please try again."),
				retryMode: retryMode(runtime.retryAction)
			}));
		}
		runtime.owner = null;
		const successor = runtime.controllers.values().next().value as OptimizationControllerRegistration | undefined;
		if (successor) void successor.resume();
	}

	runtime.controllers.set(owner, {
		clearScope: () => {
			desiredIdentityId = null;
			desiredDietId = null;
		},
		resume
	});

	return { store, setIdentity, setDiet, resume, submit, retry, dispose };
}

function runtimeFor(store: Writable<OptimizationState>): SharedOptimizationRuntime {
	let runtime = runtimes.get(store);
	if (!runtime) {
		const state = get(store);
		runtime = {
			identityId: null,
			selectedDietId: state.selectedDietId,
			pending: null,
			retryAction: "none",
			owner: null,
			controllers: new Map(),
			operation: 0,
			activeAbort: null
		};
		runtimes.set(store, runtime);
	}
	return runtime;
}

function snapshotRequest(request: DietOptimizationRequest): DietOptimizationRequest {
	return { ...request, excludedMealIds: [...request.excludedMealIds] };
}

function snapshotSubmission(request: DietOptimizationRequest, idempotencyKey: string): PendingSubmission {
	return { request: snapshotRequest(request), idempotencyKey };
}

function sameRequest(left: DietOptimizationRequest, right: DietOptimizationRequest): boolean {
	return left.dailyDietId === right.dailyDietId &&
		left.tolerancePercent === right.tolerancePercent &&
		left.excludedMealIds.length === right.excludedMealIds.length &&
		left.excludedMealIds.every((id, index) => id === right.excludedMealIds[index]);
}

function retryMode(action: OptimizationRetryAction): OptimizationRetryMode {
	if (action === "none") return "none";
	return action === "new_submission" ? "new_submission" : "reuse";
}

function validatePollingConfiguration(pollDelaysMs: readonly number[], maxPolls: number): void {
	if (pollDelaysMs.length === 0) throw new RangeError("Optimization polling requires at least one delay");
	if (!Number.isInteger(maxPolls) || maxPolls < 1 || maxPolls > MAX_POLLS) throw new RangeError("Optimization maxPolls is outside the supported range");
	if (pollDelaysMs.some((delay) => !Number.isFinite(delay) || delay < 0 || delay > MAX_POLL_DELAY_MS)) throw new RangeError("Optimization poll delays must be finite and bounded");
	let total = 0;
	for (let attempt = 0; attempt < maxPolls; attempt += 1) total += pollDelaysMs[Math.min(attempt, pollDelaysMs.length - 1)]!;
	if (!Number.isSafeInteger(total) || total > MAX_TOTAL_POLL_WAIT_MS) throw new RangeError("Optimization polling window is excessive");
}

function displayFailure(code: string, message: string, retryable = true): OptimizationDisplayError {
	const terminalMessages: Record<OptimizationFailureCode, string> = {
		failed_validation: "The optimization request could not be validated. Please review the saved diet and try again.",
		solver_timeout: "Optimization took too long. You can safely try again.",
		solver_infeasible: "No meal combination matched these macro targets. Try a wider tolerance.",
		worker_crash: "Optimization could not be completed. Please try again."
	};
	const operationMessages: Record<string, string> = {
		queue_unavailable: "The optimization queue is temporarily unavailable. Please try again.",
		result_expired: "This optimization result has expired. Submit again for a fresh result.",
		optimization_not_found: "This optimization is no longer available. Please submit again."
	};
	return { code, message: terminalMessages[code as OptimizationFailureCode] ?? operationMessages[code] ?? message, retryable };
}

function displayError(error: unknown): OptimizationDisplayError {
	if (error instanceof OptimizationClientError) return displayFailure(error.appError.code, error.appError.message, error.appError.retryable);
	return { code: "optimization_request_failed", message: "Optimization could not be completed. Please try again.", retryable: true };
}

function isAbortError(error: unknown): boolean {
	return error instanceof DOMException && error.name === "AbortError";
}

/** Abortable polling delay that removes its listener on every settlement path. */
export function waitForOptimizationPoll(delayMs: number, signal: AbortSignal): Promise<void> {
	return new Promise((resolve, reject) => {
		if (signal.aborted) {
			reject(signal.reason ?? new DOMException("Aborted", "AbortError"));
			return;
		}
		let settled = false;
		const finish = (error?: unknown) => {
			if (settled) return;
			settled = true;
			clearTimeout(timer);
			signal.removeEventListener("abort", onAbort);
			error === undefined ? resolve() : reject(error);
		};
		const timer = setTimeout(() => finish(), delayMs);
		const onAbort = () => finish(signal.reason ?? new DOMException("Aborted", "AbortError"));
		signal.addEventListener("abort", onAbort, { once: true });
	});
}
