import { fetchCsrfToken } from "./auth-client";
import { mapErrorMessage } from "./error-message-mapper";
import {
	OPTIMIZATION_JOBS_ENDPOINT,
	buildOptimizationJobRequestInit,
	buildOptimizationJobUrl,
	buildOptimizationSubmissionRequestInit,
	type AppError,
	type CanonicalQuantityUnit,
	type CompletedOptimizationAlternativeList,
	type DietOptimizationRequest,
	type IdempotencyKey,
	type OptimizationAlternative,
	type OptimizationFailureCode,
	type OptimizationJobAcknowledgementData,
	type OptimizationJobData,
	type OptimizationJobFailed
} from "./generated";

// Implements DESIGN-001 SearchView OptimizationWorkflow over the generated DESIGN-004 contract.
// Implements DESIGN-017 ErrorMessageMapper safe optimization error projection.

export interface OptimizationSubmissionOptions {
	csrfToken?: string;
	idempotencyKey: IdempotencyKey;
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
	options: OptimizationSubmissionOptions
): Promise<OptimizationJobAcknowledgementData> {
	if (!validIdempotencyKey(options?.idempotencyKey)) {
		throw new OptimizationClientError(
			{ category: "security", code: "optimization_idempotency_key_required", message: "A secure optimization request could not be created. Please try again.", retryable: false },
			0
		);
	}
	const csrfToken = await resolveCsrfToken(options);
	const response = await requestJson(
		OPTIMIZATION_JOBS_ENDPOINT,
		buildOptimizationSubmissionRequestInit(request, options.idempotencyKey, { csrfToken, signal: options.signal })
	);
	if (response.status !== 202) throw malformedResponse(response.status);
	return decodeAcknowledgement(await readJson(response), response.status);
}

/** Polls one user-scoped optimization job through the generated credentialed GET contract. */
export async function getOptimizationJob(jobId: string, signal?: AbortSignal): Promise<OptimizationJobData> {
	const response = await requestJson(buildOptimizationJobUrl(jobId), buildOptimizationJobRequestInit({ signal }));
	if (response.status !== 200) throw malformedResponse(response.status);
	return decodeJobEnvelope(await readJson(response), response.status);
}

export interface OptimizationApi {
	submitOptimization: typeof submitOptimization;
	getOptimizationJob: typeof getOptimizationJob;
}

export const optimizationApi: OptimizationApi = { submitOptimization, getOptimizationJob };

const TERMINAL_FAILURE_MESSAGES: Record<OptimizationFailureCode, string> = {
	failed_validation: "The optimization request could not be validated.",
	solver_timeout: "Optimization took too long. Please try again.",
	solver_infeasible: "No meal combination matches the requested targets.",
	worker_crash: "Optimization could not be completed. Please try again."
};

/** Generates an in-memory key for one intentional optimization submission. */
export function generateOptimizationIdempotencyKey(): IdempotencyKey {
	const cryptoValue = globalThis.crypto;
	if (!cryptoValue || typeof cryptoValue.randomUUID !== "function") {
		throw secureRandomUnavailable();
	}
	let value: unknown;
	try {
		value = cryptoValue.randomUUID();
	} catch {
		throw secureRandomUnavailable();
	}
	if (!uuidV4(value)) throw secureRandomUnavailable();
	return `optimization-${value}`;
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

async function resolveCsrfToken(options: Pick<OptimizationSubmissionOptions, "csrfToken" | "signal">): Promise<string> {
	if (options.csrfToken) return options.csrfToken;
	try {
		return (await fetchCsrfToken(options.signal)).csrfToken;
	} catch (error) {
		if (error instanceof OptimizationClientError) throw error;
		const source = error as { appError?: AppError; status?: number };
		throw new OptimizationClientError(
			mapErrorMessage("optimization", source.status ?? 503, { error: source.appError }),
			source.status ?? 503
		);
	}
}

async function readJson(response: Response): Promise<unknown> {
	try {
		return await response.json();
	} catch (error) {
		if (error instanceof OptimizationClientError) throw error;
		throw malformedResponse(response.status);
	}
}

async function responseError(response: Response): Promise<OptimizationClientError> {
	let envelope: unknown;
	try {
		envelope = await response.json();
	} catch {
		// Status-derived text is the safe fallback for empty or malformed error bodies.
	}
	return new OptimizationClientError(mapErrorMessage("optimization", response.status, envelope), response.status);
}

function decodeAcknowledgement(value: unknown, status: number): OptimizationJobAcknowledgementData {
	const { requestId, data } = decodeEnvelope(value, status, "accepted");
	if (!exactObject(data, ["jobId", "status", "pollUrl"]) || !uuid(data.jobId) || data.status !== "queued" || !canonicalPollUrl(data.pollUrl, data.jobId)) {
		throw malformedResponse(status, requestId);
	}
	return { jobId: data.jobId, status: "queued", pollUrl: data.pollUrl };
}

function decodeJobEnvelope(value: unknown, status: number): OptimizationJobData {
	const { requestId, data: job } = decodeEnvelope(value, status, "ok");
	if (!isObject(job) || typeof job.status !== "string") throw malformedResponse(status, requestId);
	const common = decodeJobCommon(job, status, requestId);
	switch (job.status) {
		case "queued":
			assertKeys(job, ["jobId", "dailyDietId", "status", "pollUrl", "createdAt"], status, requestId);
			return { ...common, status: "queued" };
		case "processing":
			assertKeys(job, ["jobId", "dailyDietId", "status", "pollUrl", "createdAt", "startedAt"], status, requestId);
			if (!dateTime(job.startedAt)) throw malformedResponse(status, requestId);
			return { ...common, status: "processing", startedAt: job.startedAt };
		case "completed": {
			assertKeys(job, ["jobId", "dailyDietId", "status", "pollUrl", "createdAt", "startedAt", "finishedAt", "alternatives"], status, requestId);
			if (!dateTime(job.startedAt) || !dateTime(job.finishedAt)) throw malformedResponse(status, requestId);
			const alternatives = decodeAlternatives(job.alternatives, status, requestId, 1);
			return { ...common, status: "completed", startedAt: job.startedAt, finishedAt: job.finishedAt, alternatives: alternatives as CompletedOptimizationAlternativeList };
		}
		case "failed":
			return decodeFailedJob(job, common, status, requestId);
		case "cancelled":
			assertKeys(job, ["jobId", "dailyDietId", "status", "pollUrl", "createdAt", "finishedAt"], status, requestId);
			if (!dateTime(job.finishedAt)) throw malformedResponse(status, requestId);
			return { ...common, status: "cancelled", finishedAt: job.finishedAt };
		default:
			throw malformedResponse(status, requestId);
	}
}

function decodeJobCommon(job: Record<string, unknown>, status: number, requestId: string) {
	if (!uuid(job.jobId) || !uuid(job.dailyDietId) || !canonicalPollUrl(job.pollUrl, job.jobId) || !dateTime(job.createdAt)) {
		throw malformedResponse(status, requestId);
	}
	return { jobId: job.jobId, dailyDietId: job.dailyDietId, pollUrl: job.pollUrl, createdAt: job.createdAt };
}

function decodeFailedJob(job: Record<string, unknown>, common: ReturnType<typeof decodeJobCommon>, status: number, requestId: string): OptimizationJobFailed {
	const optional = ["startedAt", "finishedAt", "alternatives"].filter((key) => key in job);
	assertKeys(job, ["jobId", "dailyDietId", "status", "pollUrl", "createdAt", "failure", ...optional], status, requestId);
	if (("startedAt" in job && job.startedAt !== null && !dateTime(job.startedAt)) || ("finishedAt" in job && job.finishedAt !== null && !dateTime(job.finishedAt))) {
		throw malformedResponse(status, requestId);
	}
	if (!exactObject(job.failure, ["code", "message"]) || !isOptimizationFailureCode(job.failure.code) || job.failure.message !== TERMINAL_FAILURE_MESSAGES[job.failure.code]) {
		throw malformedResponse(status, requestId);
	}
	const result: OptimizationJobFailed = { ...common, status: "failed", failure: { code: job.failure.code, message: job.failure.message } };
	if ("startedAt" in job) result.startedAt = job.startedAt as string | null;
	if ("finishedAt" in job) result.finishedAt = job.finishedAt as string | null;
	if ("alternatives" in job) result.alternatives = decodeAlternatives(job.alternatives, status, requestId) as OptimizationJobFailed["alternatives"];
	return result;
}

function decodeAlternatives(value: unknown, status: number, requestId: string, minimum = 0): OptimizationAlternative[] {
	if (!Array.isArray(value) || value.length < minimum || value.length > 3) throw malformedResponse(status, requestId);
	return value.map((raw) => decodeAlternative(raw, status, requestId));
}

function decodeAlternative(raw: unknown, status: number, requestId: string): OptimizationAlternative {
	if (!exactObject(raw, ["meals", "macros", "similarityScore"]) || !Array.isArray(raw.meals) || raw.meals.length < 1 || raw.meals.length > 100 || !exactObject(raw.macros, ["protein", "carbohydrates", "fat", "calories"]) || !validSimilarityScore(raw.similarityScore)) {
		throw malformedResponse(status, requestId);
	}
	const macros = raw.macros;
	if (!boundedMacro(macros.protein) || !boundedMacro(macros.carbohydrates) || !boundedMacro(macros.fat) || !boundedMacro(macros.calories)) throw malformedResponse(status, requestId);
	const meals = raw.meals.map((meal) => {
		if (!exactObject(meal, ["mealId", "quantity", "unit", "position"]) || !uuid(meal.mealId) || !boundedQuantity(meal.quantity) || !canonicalUnit(meal.unit) || !boundedPosition(meal.position)) {
			throw malformedResponse(status, requestId);
		}
		return { mealId: meal.mealId, quantity: meal.quantity, unit: meal.unit, position: meal.position };
	});
	return { meals, macros: { protein: macros.protein, carbohydrates: macros.carbohydrates, fat: macros.fat, calories: macros.calories }, similarityScore: raw.similarityScore };
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

function secureRandomUnavailable(): OptimizationClientError {
	return new OptimizationClientError(
		{ category: "security", code: "secure_random_unavailable", message: "A secure optimization request could not be created. Please try again.", retryable: true },
		0
	);
}

function isObject(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null && !Array.isArray(value);
}

function exactObject(value: unknown, keys: readonly string[]): value is Record<string, unknown> {
	if (!isObject(value)) return false;
	const actual = Object.keys(value).sort();
	const expected = [...keys].sort();
	return actual.length === expected.length && actual.every((key, index) => key === expected[index]);
}

function assertKeys(value: Record<string, unknown>, keys: readonly string[], status: number, requestId: string): void {
	if (!exactObject(value, keys)) throw malformedResponse(status, requestId);
}

function decodeEnvelope(value: unknown, status: number, expectedStatus: "accepted" | "ok"): { requestId: string; data: unknown } {
	if (!exactObject(value, ["status", "requestId", "data"]) || value.status !== expectedStatus || !safeRequestId(value.requestId)) {
		throw malformedResponse(status);
	}
	return { requestId: value.requestId, data: value.data };
}

function safeRequestId(value: unknown): value is string {
	return typeof value === "string" && /^[A-Za-z0-9._:-]{1,120}$/.test(value);
}

function validIdempotencyKey(value: unknown): value is IdempotencyKey {
	return typeof value === "string" && /^optimization-[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(value);
}

function uuid(value: unknown): value is string {
	return typeof value === "string" && /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(value);
}

function uuidV4(value: unknown): value is `${string}-${string}-${string}-${string}-${string}` {
	return typeof value === "string" && /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(value);
}

function canonicalPollUrl(value: unknown, jobId: string): value is string {
	return typeof value === "string" && value.length <= 128 && value === `${OPTIMIZATION_JOBS_ENDPOINT}/${jobId}`;
}

function dateTime(value: unknown): value is string {
	if (typeof value !== "string") return false;
	const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,9})?(?:Z|[+-](\d{2}):(\d{2}))$/.exec(value);
	if (!match) return false;
	const [year, month, day, hour, minute, second] = match.slice(1, 7).map(Number);
	const calendar = new Date(Date.UTC(year!, month! - 1, day!));
	const offsetValid = !match[7] || (Number(match[7]) <= 23 && Number(match[8]) <= 59);
	return calendar.getUTCFullYear() === year && calendar.getUTCMonth() === month! - 1 && calendar.getUTCDate() === day && hour! <= 23 && minute! <= 59 && second! <= 59 && offsetValid && Number.isFinite(Date.parse(value));
}

function finiteNumber(value: unknown): value is number {
	return typeof value === "number" && Number.isFinite(value);
}

function boundedMacro(value: unknown): value is number {
	return finiteNumber(value) && value >= 0 && value <= 1_000_000_000;
}

function boundedQuantity(value: unknown): value is number {
	return finiteNumber(value) && value > 0 && value <= 1_000_000 && multipleOf(value, 0.001);
}

function boundedPosition(value: unknown): value is number {
	return typeof value === "number" && Number.isInteger(value) && value >= 0 && value <= 99;
}

function canonicalUnit(value: unknown): value is CanonicalQuantityUnit {
	return value === "g" || value === "ml" || value === "oz" || value === "fl_oz";
}

function multipleOf(value: number, step: number): boolean {
	return Math.abs(value / step - Math.round(value / step)) <= 1e-9;
}

function validSimilarityScore(value: unknown): value is number {
	return finiteNumber(value) && value >= 0 && value <= 1 && multipleOf(value, 0.0001);
}

function isOptimizationFailureCode(value: unknown): value is OptimizationFailureCode {
	return value === "failed_validation" || value === "solver_timeout" || value === "solver_infeasible" || value === "worker_crash";
}
