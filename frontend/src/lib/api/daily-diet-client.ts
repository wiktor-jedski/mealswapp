import { fetchCsrfToken } from "./auth-client";
import {
	DAILY_DIETS_ENDPOINT,
	buildDailyDietCreateRequestInit,
	buildDailyDietDeleteRequestInit,
	buildDailyDietGetRequestInit,
	buildDailyDietReplaceRequestInit,
	buildDailyDietUrl,
	type AppError,
	type DailyDiet,
	type DailyDietCollectionEnvelope,
	type DailyDietCreateRequest,
	type DailyDietEnvelope,
	type DailyDietReplaceRequest,
	type Envelope,
	type IdempotencyKey
} from "./generated";

// Implements DESIGN-001 SearchView authenticated Daily Diet collection client over generated contracts.
// Implements DESIGN-017 ErrorMessageMapper safe cross-user error projection.

/** Options shared by authenticated Daily Diet mutations. */
export interface DailyDietMutationOptions {
	csrfToken?: string;
	idempotencyKey?: IdempotencyKey;
	signal?: AbortSignal;
}

/** Error exposed by Daily Diet operations after raw network/API failures are safely mapped. */
export class DailyDietClientError extends Error {
	readonly appError: AppError;
	readonly status: number;

	constructor(appError: AppError, status: number) {
		super(appError.message);
		this.name = "DailyDietClientError";
		this.appError = appError;
		this.status = status;
	}
}

/** Lists the current cookie-authenticated user's saved Daily Diet collections. */
export async function listDailyDiets(signal?: AbortSignal): Promise<DailyDiet[]> {
	const response = await request(DAILY_DIETS_ENDPOINT, buildDailyDietGetRequestInit({ signal }));
	const envelope = await requireData<DailyDietCollectionEnvelope>(response, "Saved daily diets are temporarily unavailable. Please try again.");
	if (!isObject(envelope.data) || !Array.isArray(envelope.data.diets)) {
		throw malformedResponse(response.status, envelope.requestId);
	}
	return envelope.data.diets;
}

/** Reads one current-user-owned saved Daily Diet by id. */
export async function getDailyDiet(dietId: string, signal?: AbortSignal): Promise<DailyDiet> {
	const response = await request(buildDailyDietUrl(dietId), buildDailyDietGetRequestInit({ signal }));
	const envelope = await requireData<DailyDietEnvelope>(response, "Saved daily diet is temporarily unavailable. Please try again.");
	if (!isObject(envelope.data)) {
		throw malformedResponse(response.status, envelope.requestId);
	}
	return envelope.data;
}

/** Creates a user-owned Daily Diet with generated DTOs, CSRF, and one stable idempotency key. */
export async function createDailyDiet(
	request: DailyDietCreateRequest,
	options: DailyDietMutationOptions = {}
): Promise<DailyDiet> {
	const csrfToken = await resolveCsrfToken(options);
	const idempotencyKey = options.idempotencyKey ?? generateDailyDietIdempotencyKey();
	const response = await requestJson(
		DAILY_DIETS_ENDPOINT,
		buildDailyDietCreateRequestInit(request, idempotencyKey, { csrfToken, signal: options.signal })
	);
	const envelope = await requireData<DailyDietEnvelope>(response, "Saved daily diet could not be created. Please try again.");
	if (!isObject(envelope.data)) {
		throw malformedResponse(response.status, envelope.requestId);
	}
	return envelope.data;
}

/** Replaces one user-owned Daily Diet; server totals remain authoritative. */
export async function replaceDailyDiet(
	dietId: string,
	request: DailyDietReplaceRequest,
	options: Pick<DailyDietMutationOptions, "csrfToken" | "signal"> = {}
): Promise<DailyDiet> {
	const csrfToken = await resolveCsrfToken(options);
	const response = await requestJson(
		buildDailyDietUrl(dietId),
		buildDailyDietReplaceRequestInit(request, { csrfToken, signal: options.signal })
	);
	const envelope = await requireData<DailyDietEnvelope>(response, "Saved daily diet could not be updated. Please try again.");
	if (!isObject(envelope.data)) {
		throw malformedResponse(response.status, envelope.requestId);
	}
	return envelope.data;
}

/** Deletes one user-owned Daily Diet after CSRF validation. */
export async function deleteDailyDiet(dietId: string, options: Pick<DailyDietMutationOptions, "csrfToken" | "signal"> = {}): Promise<void> {
	const csrfToken = await resolveCsrfToken(options);
	const response = await requestJson(
		buildDailyDietUrl(dietId),
		buildDailyDietDeleteRequestInit({ csrfToken, signal: options.signal })
	);
	if (response.status !== 204 && response.status !== 200) {
		throw new DailyDietClientError(safeErrorForStatus(response.status), response.status);
	}
}

/** Alias kept explicit for callers that use fetch terminology for collection reads. */
export const fetchDailyDiets = listDailyDiets;

/** Alias kept explicit for callers that use fetch terminology for item reads. */
export const fetchDailyDiet = getDailyDiet;

/** API dependency shape accepted by the Svelte controller for deterministic unit tests. */
export interface DailyDietApi {
	listDailyDiets: typeof listDailyDiets;
	createDailyDiet: typeof createDailyDiet;
	replaceDailyDiet: typeof replaceDailyDiet;
	deleteDailyDiet: typeof deleteDailyDiet;
}

/** Default API implementation used by the production Daily Diet controller. */
export const dailyDietApi: DailyDietApi = {
	listDailyDiets,
	createDailyDiet,
	replaceDailyDiet,
	deleteDailyDiet
};

/** Generates a retry-stable key for one create mutation without persisting session data. */
export function generateDailyDietIdempotencyKey(): IdempotencyKey {
	const cryptoValue = globalThis.crypto;
	if (cryptoValue && typeof cryptoValue.randomUUID === "function") {
		return `daily-diet-${cryptoValue.randomUUID()}`;
	}
	return `daily-diet-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
}

async function request(url: string, init: RequestInit): Promise<Response> {
	try {
		return await requestJson(url, init);
	} catch (error) {
		if (error instanceof DailyDietClientError) {
			throw error;
		}
		throw networkError(error);
	}
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

async function resolveCsrfToken(options: Pick<DailyDietMutationOptions, "csrfToken" | "signal">): Promise<string> {
	if (options.csrfToken) {
		return options.csrfToken;
	}
	try {
		return (await fetchCsrfToken(options.signal)).csrfToken;
	} catch (error) {
		if (error instanceof DailyDietClientError) {
			throw error;
		}
		const source = error as { appError?: AppError; status?: number };
		throw new DailyDietClientError(
			safeErrorFromSource(source.appError, source.status ?? 503),
			source.status ?? 503
		);
	}
}

async function responseError(response: Response): Promise<DailyDietClientError> {
	let envelope: Envelope | null = null;
	try {
		const body: unknown = await response.json();
		if (isObject(body)) {
			envelope = body as unknown as Envelope;
		}
	} catch {
		// The status-derived message is the safe fallback for an empty or malformed error body.
	}
	const appError = safeErrorFromSource(envelope?.error ?? undefined, response.status);
	if (!appError.requestId && typeof envelope?.requestId === "string" && envelope.requestId.length > 0) {
		appError.requestId = envelope.requestId;
	}
	return new DailyDietClientError(appError, response.status);
}

async function requireData<TEnvelope extends Envelope>(response: Response, fallback: string): Promise<TEnvelope & { data: NonNullable<TEnvelope["data"]> }> {
	return readJson(response).then((body) => {
		if (!isObject(body) || body.data === undefined || body.data === null) {
			throw malformedResponse(response.status, isObject(body) && typeof body.requestId === "string" ? body.requestId : undefined);
		}
		return body as unknown as TEnvelope & { data: NonNullable<TEnvelope["data"]> };
	}).catch((error) => {
		if (error instanceof DailyDietClientError) {
			throw error;
		}
		throw new DailyDietClientError(
			{ category: "server", code: "malformed_envelope", message: fallback, retryable: true },
			response.status
		);
	});
}

async function readJson(response: Response): Promise<unknown> {
	try {
		return await response.json();
	} catch {
		throw malformedResponse(response.status);
	}
}

function malformedResponse(status: number, requestId?: string): DailyDietClientError {
	const appError: AppError = {
		category: "server",
		code: "malformed_envelope",
		message: "Saved daily diets are temporarily unavailable. Please try again.",
		retryable: true
	};
	if (requestId) appError.requestId = requestId;
	return new DailyDietClientError(appError, status);
}

function networkError(error: unknown): DailyDietClientError {
	if (error instanceof DailyDietClientError) {
		return error;
	}
	const appError: AppError = {
		category: "network",
		code: "daily_diet_network_error",
		message: "Network is unavailable. Please check your connection and try again.",
		retryable: true
	};
	if (error instanceof DOMException && error.name === "AbortError") {
		appError.category = "timeout";
		appError.code = "daily_diet_request_aborted";
		appError.message = "The saved daily-diet request was cancelled. Please try again.";
	}
	return new DailyDietClientError(appError, 0);
}

function safeErrorFromSource(source: AppError | undefined, status: number): AppError {
	const fallback = safeErrorForStatus(status);
	if (status === 403 || status === 404) {
		return fallback;
	}
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
	if (status === 401) {
		return { category: "auth", code: "session_expired", message: "Your session expired. Please sign in and try again.", retryable: false };
	}
	if (status === 403 || status === 404) {
		return { category: "security", code: "daily_diet_unavailable", message: "Saved daily diet is unavailable.", retryable: false };
	}
	if (status === 400 || status === 409 || status === 422) {
		return { category: "validation", code: "daily_diet_invalid_request", message: "Saved daily diet request could not be processed. Please review it and try again.", retryable: false };
	}
	if (status === 429) {
		return { category: "server", code: "daily_diet_rate_limited", message: "Too many saved daily-diet requests. Please wait and try again.", retryable: true };
	}
	if (status === 503) {
		return { category: "dependency", code: "daily_diet_unavailable", message: "Saved daily diets are temporarily unavailable. Please try again shortly.", retryable: true };
	}
	return { category: "unknown", code: "daily_diet_request_failed", message: "Something went wrong. Please try again.", retryable: false };
}

function isObject(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null && !Array.isArray(value);
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
