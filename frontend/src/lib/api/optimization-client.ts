import { fetchCsrfToken } from "./auth-client";
import {
	OPTIMIZATION_JOBS_ENDPOINT,
	buildOptimizationJobRequestInit,
	buildOptimizationJobUrl,
	buildOptimizationSubmissionRequestInit,
	type AppError,
	type DietOptimizationRequest,
	type Envelope,
	type IdempotencyKey,
	type OptimizationAlternative,
	type OptimizationJobAcknowledgementEnvelope,
	type OptimizationJobAcknowledgementData,
	type OptimizationJobData,
	type OptimizationJobStatusEnvelope
} from "./generated";

// Implements DESIGN-001 SearchView OptimizationWorkflow over the generated DESIGN-004 contract.
// Implements DESIGN-017 ErrorMessageMapper safe optimization error projection.

export interface OptimizationRequestOptions {
	csrfToken?: string;
	idempotencyKey?: IdempotencyKey;
	signal?: AbortSignal;
}

export class OptimizationClientError extends Error {
	readonly appError: AppError;
	readonly status: number;

	constructor(appError: AppError, status: number) {
		super(appError.message);
		this.name = "OptimizationClientError";
		this.appError = appError;
		this.status = status;
	}
}

/** Submits one server-owned saved diet and returns only the accepted job acknowledgement. */
export async function submitOptimization(
	request: DietOptimizationRequest,
	options: OptimizationRequestOptions = {}
): Promise<OptimizationJobAcknowledgementData> {
	const csrfToken = await resolveCsrfToken(options);
	const idempotencyKey = options.idempotencyKey ?? generateOptimizationIdempotencyKey();
	const response = await requestJson(
		OPTIMIZATION_JOBS_ENDPOINT,
		buildOptimizationSubmissionRequestInit(request, idempotencyKey, { csrfToken, signal: options.signal })
	);
	const envelope = await readEnvelope<OptimizationJobAcknowledgementEnvelope>(response);
	if (!isObject(envelope.data) || envelope.status !== "accepted" || envelope.data.status !== "queued") {
		throw malformedResponse(response.status, envelope.requestId);
	}
	return envelope.data;
}

/** Polls one user-scoped optimization job through the generated credentialed GET contract. */
export async function getOptimizationJob(jobId: string, signal?: AbortSignal): Promise<OptimizationJobData> {
	const response = await requestJson(buildOptimizationJobUrl(jobId), buildOptimizationJobRequestInit({ signal }));
	const envelope = await readEnvelope<OptimizationJobStatusEnvelope>(response);
	if (!isObject(envelope.data)) {
		throw malformedResponse(response.status, envelope.requestId);
	}
	return normalizeJob(envelope.data, response.status, envelope.requestId);
}

export interface OptimizationApi {
	submitOptimization: typeof submitOptimization;
	getOptimizationJob: typeof getOptimizationJob;
}

export const optimizationApi: OptimizationApi = { submitOptimization, getOptimizationJob };

/** Generates an in-memory key for one intentional optimization submission. */
export function generateOptimizationIdempotencyKey(): IdempotencyKey {
	const cryptoValue = globalThis.crypto;
	if (cryptoValue && typeof cryptoValue.randomUUID === "function") {
		return `optimization-${cryptoValue.randomUUID()}`;
	}
	return `optimization-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
}

async function requestJson(url: string, init: RequestInit): Promise<Response> {
	let response: Response;
	try {
		response = await fetch(url, init);
	} catch (error) {
		throw networkError(error);
	}
	if (!response.ok) {
		throw await responseError(response);
	}
	return response;
}

async function resolveCsrfToken(options: Pick<OptimizationRequestOptions, "csrfToken" | "signal">): Promise<string> {
	if (options.csrfToken) return options.csrfToken;
	try {
		return (await fetchCsrfToken(options.signal)).csrfToken;
	} catch (error) {
		if (error instanceof OptimizationClientError) throw error;
		const source = error as { appError?: AppError; status?: number };
		throw new OptimizationClientError(
			safeErrorFromSource(source.appError, source.status ?? 503),
			source.status ?? 503
		);
	}
}

async function readEnvelope<TEnvelope extends Envelope>(response: Response): Promise<TEnvelope> {
	try {
		const body: unknown = await response.json();
		if (!isObject(body) || typeof body.requestId !== "string") {
			throw malformedResponse(response.status);
		}
		return body as TEnvelope;
	} catch (error) {
		if (error instanceof OptimizationClientError) throw error;
		throw malformedResponse(response.status);
	}
}

async function responseError(response: Response): Promise<OptimizationClientError> {
	let envelope: Envelope | null = null;
	try {
		const body: unknown = await response.json();
		if (isObject(body)) envelope = body as unknown as Envelope;
	} catch {
		// Status-derived text is the safe fallback for empty or malformed error bodies.
	}
	const appError = safeErrorFromSource(envelope?.error ?? undefined, response.status);
	if (!appError.requestId && typeof envelope?.requestId === "string" && envelope.requestId.length > 0) {
		appError.requestId = envelope.requestId;
	}
	return new OptimizationClientError(appError, response.status);
}

function normalizeJob(job: OptimizationJobData, status: number, requestId: string): OptimizationJobData {
	if (job.status === "completed") {
		const alternatives = normalizeAlternatives(job.alternatives, status, requestId);
		if (alternatives.length < 1) throw malformedResponse(status, requestId);
		return { ...job, alternatives: alternatives as typeof job.alternatives };
	}
	if (job.status === "failed" && job.alternatives) {
		return { ...job, alternatives: normalizeAlternatives(job.alternatives, status, requestId) as typeof job.alternatives };
	}
	if (job.status !== "queued" && job.status !== "processing" && job.status !== "failed" && job.status !== "cancelled") {
		throw malformedResponse(status, requestId);
	}
	return job;
}

function normalizeAlternatives(value: unknown, status: number, requestId: string): OptimizationAlternative[] {
	if (!Array.isArray(value) || value.length > 3) throw malformedResponse(status, requestId);
	return value.map((raw) => {
		if (!isObject(raw) || !Array.isArray(raw.meals) || !isObject(raw.macros)) {
			throw malformedResponse(status, requestId);
		}
		const macros = raw.macros;
		const calories = finiteNumber(macros.calories) ? macros.calories : finiteNumber(raw.calories) ? raw.calories : null;
		if (
			calories === null ||
			!finiteNumber(macros.protein) ||
			!finiteNumber(macros.carbohydrates) ||
			!finiteNumber(macros.fat) ||
			!finiteNumber(raw.similarityScore)
		) {
			throw malformedResponse(status, requestId);
		}
		return {
			meals: raw.meals as OptimizationAlternative["meals"],
			macros: {
				protein: macros.protein,
				carbohydrates: macros.carbohydrates,
				fat: macros.fat,
				calories
			},
			similarityScore: raw.similarityScore
		};
	});
}

function malformedResponse(status: number, requestId?: string): OptimizationClientError {
	const appError: AppError = {
		category: "server",
		code: "malformed_optimization_response",
		message: "Optimization returned an invalid response. Please try again.",
		retryable: true
	};
	if (requestId) appError.requestId = requestId;
	return new OptimizationClientError(appError, status);
}

function networkError(error: unknown): OptimizationClientError {
	if (error instanceof OptimizationClientError) return error;
	if (error instanceof DOMException && error.name === "AbortError") {
		return new OptimizationClientError(
			{ category: "timeout", code: "optimization_request_aborted", message: "The optimization request was cancelled. Please try again.", retryable: true },
			0
		);
	}
	return new OptimizationClientError(
		{ category: "network", code: "optimization_network_error", message: "Optimization is unavailable. Check your connection and try again.", retryable: true },
		0
	);
}

function safeErrorFromSource(source: AppError | undefined, status: number): AppError {
	const fallback = safeErrorForStatus(status);
	const appError: AppError = {
		category: isErrorCategory(source?.category) ? source.category : fallback.category,
		code: isSafeCode(source?.code) ? source.code : fallback.code,
		message: isSafeMessage(source?.message) ? source.message : fallback.message,
		retryable: source?.retryable ?? fallback.retryable
	};
	if (isSafeRequestId(source?.requestId)) appError.requestId = source.requestId;
	return appError;
}

function safeErrorForStatus(status: number): AppError {
	if (status === 401) return { category: "auth", code: "session_expired", message: "Your session expired. Please sign in and try again.", retryable: false };
	if (status === 403) return { category: "entitlement", code: "entitlement_denied", message: "An active trial or paid subscription is required for optimization.", retryable: false };
	if (status === 410) return { category: "validation", code: "result_expired", message: "This optimization result has expired. Submit again for a fresh result.", retryable: true };
	if (status === 503) return { category: "dependency", code: "queue_unavailable", message: "The optimization queue is temporarily unavailable. Please try again.", retryable: true };
	if (status === 422) return { category: "validation", code: "solver_infeasible", message: "No meal combination matched these macro targets. Try a wider tolerance.", retryable: false };
	if (status === 404) return { category: "validation", code: "optimization_not_found", message: "This optimization is no longer available. Please submit again.", retryable: true };
	if (status === 400 || status === 409) return { category: "validation", code: "optimization_invalid_request", message: "Optimization request could not be processed. Please review it and try again.", retryable: false };
	return { category: "unknown", code: "optimization_request_failed", message: "Optimization could not be completed. Please try again.", retryable: true };
}

function isObject(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null && !Array.isArray(value);
}

function finiteNumber(value: unknown): value is number {
	return typeof value === "number" && Number.isFinite(value);
}

function isErrorCategory(value: unknown): value is AppError["category"] {
	return value === "validation" || value === "auth" || value === "entitlement" || value === "security" || value === "network" || value === "timeout" || value === "server" || value === "dependency" || value === "unknown";
}

function isSafeMessage(value: unknown): value is string {
	return typeof value === "string" && value.length > 0 && value.length <= 240 && !value.includes("\n") && !value.includes(" at ") && !/https?:\/\//i.test(value);
}

function isSafeCode(value: unknown): value is string {
	return typeof value === "string" && /^[a-z][a-z0-9_]{0,79}$/.test(value);
}

function isSafeRequestId(value: unknown): value is string {
	return typeof value === "string" && value.length > 0 && value.length <= 120 && !/[\s\n]/.test(value);
}
