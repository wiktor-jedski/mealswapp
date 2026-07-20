import { fetchCsrfToken } from "./auth-client";
import { mapErrorMessage } from "./error-message-mapper";
import {
	DAILY_DIETS_ENDPOINT,
	buildDailyDietCreateRequestInit,
	buildDailyDietDeleteRequestInit,
	buildDailyDietGetRequestInit,
	buildDailyDietListRequestInit,
	buildDailyDietReplaceRequestInit,
	buildDailyDietUrl,
	type AppError,
	type CanonicalQuantityUnit,
	type DailyDiet,
	type DailyDietCreateRequest,
	type DailyDietReplaceRequest,
	type IdempotencyKey
} from "./generated";

// Implements DESIGN-001 SearchView authenticated Daily Diet collection client over generated contracts.
// Implements DESIGN-017 ErrorMessageMapper safe cross-user error projection.

/** Options shared by authenticated Daily Diet mutations. */
export interface DailyDietMutationOptions {
	csrfToken?: string;
	signal?: AbortSignal;
}

/** Caller-owned create options keep one key stable for one user operation. */
export interface DailyDietCreateOptions extends DailyDietMutationOptions {
	idempotencyKey: IdempotencyKey;
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
	const response = await requestJson(DAILY_DIETS_ENDPOINT, buildDailyDietListRequestInit({ signal }));
	return decodeCollection(await readSuccess(response, 200), response.status);
}

/** Reads one current-user-owned saved Daily Diet by id. */
export async function getDailyDiet(dietId: string, signal?: AbortSignal): Promise<DailyDiet> {
	const response = await requestJson(buildDailyDietUrl(dietId), buildDailyDietGetRequestInit({ signal }));
	return decodeItem(await readSuccess(response, 200), response.status);
}

/** Creates a user-owned Daily Diet with a caller-owned retry-stable idempotency key. */
export async function createDailyDiet(request: DailyDietCreateRequest, options: DailyDietCreateOptions): Promise<DailyDiet> {
	if (!validIdempotencyKey(options?.idempotencyKey)) throw idempotencyKeyRequired();
	const csrfToken = await resolveCsrfToken(options);
	const response = await requestJson(
		DAILY_DIETS_ENDPOINT,
		buildDailyDietCreateRequestInit(request, options.idempotencyKey, { csrfToken, signal: options.signal })
	);
	return decodeItem(await readSuccess(response, 201), response.status);
}

/** Replaces one user-owned Daily Diet; server totals remain authoritative. */
export async function replaceDailyDiet(
	dietId: string,
	request: DailyDietReplaceRequest,
	options: DailyDietMutationOptions = {}
): Promise<DailyDiet> {
	const csrfToken = await resolveCsrfToken(options);
	const response = await requestJson(
		buildDailyDietUrl(dietId),
		buildDailyDietReplaceRequestInit(request, { csrfToken, signal: options.signal })
	);
	return decodeItem(await readSuccess(response, 200), response.status);
}

/** Deletes one user-owned Daily Diet only on the documented empty 204 response. */
export async function deleteDailyDiet(dietId: string, options: DailyDietMutationOptions = {}): Promise<void> {
	const csrfToken = await resolveCsrfToken(options);
	const response = await requestJson(
		buildDailyDietUrl(dietId),
		buildDailyDietDeleteRequestInit({ csrfToken, signal: options.signal })
	);
	if (response.status !== 204) throw malformedResponse(response.status);
	try {
		if ((await boundedText(response)) !== "") throw malformedResponse(response.status);
	} catch (error) {
		if (error instanceof DailyDietClientError) throw error;
		throw malformedResponse(response.status);
	}
}

/** API dependency shape accepted by the Svelte controller for deterministic unit tests. */
export interface DailyDietApi {
	listDailyDiets: typeof listDailyDiets;
	createDailyDiet: typeof createDailyDiet;
	replaceDailyDiet: typeof replaceDailyDiet;
	deleteDailyDiet: typeof deleteDailyDiet;
}

/** Default API implementation used by the production Daily Diet controller. */
export const dailyDietApi: DailyDietApi = { listDailyDiets, createDailyDiet, replaceDailyDiet, deleteDailyDiet };

/** Generates a collision-resistant memory-only key for one intentional create operation. */
export function generateDailyDietIdempotencyKey(): IdempotencyKey {
	const cryptoValue = globalThis.crypto;
	if (!cryptoValue || typeof cryptoValue.randomUUID !== "function") {
		throw secureRandomUnavailable();
	}
	try {
		const randomUuid = cryptoValue.randomUUID();
		if (!randomUuidV4(randomUuid)) throw secureRandomUnavailable();
		const key = `daily-diet-${randomUuid}`;
		if (!validIdempotencyKey(key)) throw secureRandomUnavailable();
		return key;
	} catch (error) {
		if (error instanceof DailyDietClientError) throw error;
		throw secureRandomUnavailable();
	}
}

const MAX_RESPONSE_BYTES = 5 * 1024 * 1024;

async function requestJson(url: string, init: RequestInit): Promise<Response> {
	let response: Response;
	try {
		response = await fetch(url, init);
	} catch (error) {
		throw networkError(error);
	}
	if (!response.ok) throw await responseError(response);
	return response;
}

async function resolveCsrfToken(options: DailyDietMutationOptions): Promise<string> {
	if (options.csrfToken) return options.csrfToken;
	try {
		return (await fetchCsrfToken(options.signal)).csrfToken;
	} catch (error) {
		if (error instanceof DailyDietClientError) throw error;
		const source = error as { appError?: AppError; status?: number };
		throw new DailyDietClientError(
			mapErrorMessage("daily_diet", source.status ?? 503, { error: source.appError }),
			source.status ?? 503
		);
	}
}

async function readSuccess(response: Response, expectedStatus: number): Promise<unknown> {
	if (response.status !== expectedStatus) throw malformedResponse(response.status);
	try {
		const text = await boundedText(response);
		return JSON.parse(text) as unknown;
	} catch (error) {
		if (error instanceof DailyDietClientError) throw error;
		throw malformedResponse(response.status);
	}
}

async function boundedText(response: Response): Promise<string> {
	const contentLength = response.headers.get("Content-Length");
	const declaredLength = contentLength === null ? null : Number(contentLength);
	if (declaredLength !== null && Number.isFinite(declaredLength) && declaredLength > MAX_RESPONSE_BYTES) {
		throw malformedResponse(response.status);
	}
	if (!response.body) return "";

	const reader = response.body.getReader();
	const chunks: Uint8Array[] = [];
	let byteLength = 0;
	try {
		while (true) {
			const { done, value } = await reader.read();
			if (done) break;
			byteLength += value.byteLength;
			if (byteLength > MAX_RESPONSE_BYTES) {
				void reader.cancel().catch(() => undefined);
				throw malformedResponse(response.status);
			}
			chunks.push(value);
		}
	} finally {
		reader.releaseLock();
	}

	const bytes = new Uint8Array(byteLength);
	let offset = 0;
	for (const chunk of chunks) {
		bytes.set(chunk, offset);
		offset += chunk.byteLength;
	}
	return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
}

async function responseError(response: Response): Promise<DailyDietClientError> {
	let envelope: unknown;
	try {
		envelope = JSON.parse(await boundedText(response)) as unknown;
	} catch {
		// The status-derived message is the safe fallback for an empty, malformed, or oversized error body.
	}
	return new DailyDietClientError(mapErrorMessage("daily_diet", response.status, envelope), response.status);
}

function decodeCollection(value: unknown, status: number): DailyDiet[] {
	const { requestId, data } = decodeEnvelope(value, status);
	if (!exactObject(data, ["diets"]) || !Array.isArray(data.diets)) throw malformedResponse(status, requestId);
	return data.diets.map((diet) => decodeDiet(diet, status, requestId));
}

function decodeItem(value: unknown, status: number): DailyDiet {
	const { requestId, data } = decodeEnvelope(value, status);
	return decodeDiet(data, status, requestId);
}

function decodeEnvelope(value: unknown, status: number): { requestId: string; data: unknown } {
	if (!exactObject(value, ["status", "requestId", "data"]) || value.status !== "ok" || !safeRequestId(value.requestId)) {
		throw malformedResponse(status);
	}
	return { requestId: value.requestId, data: value.data };
}

function decodeDiet(value: unknown, status: number, requestId: string): DailyDiet {
	if (
		!exactObject(value, ["id", "name", "entries", "aggregateMacros", "createdAt", "updatedAt"]) ||
		!uuid(value.id) || !boundedString(value.name, 1, 120) ||
		!Array.isArray(value.entries) || value.entries.length < 1 || value.entries.length > 100 ||
		!dateTime(value.createdAt) || !dateTime(value.updatedAt)
	) throw malformedResponse(status, requestId);

	const entries = value.entries.map((entry) => {
		if (
			!exactObject(entry, ["id", "mealId", "quantity", "unit", "position"]) ||
			!uuid(entry.id) || !uuid(entry.mealId) || !boundedQuantity(entry.quantity) ||
			!canonicalUnit(entry.unit) || !boundedPosition(entry.position)
		) throw malformedResponse(status, requestId);
		return { id: entry.id, mealId: entry.mealId, quantity: entry.quantity, unit: entry.unit, position: entry.position };
	});

	const macros = value.aggregateMacros;
	if (
		!exactObject(macros, ["protein", "carbohydrates", "fat", "calories"]) ||
		!boundedMacro(macros.protein) || !boundedMacro(macros.carbohydrates) ||
		!boundedMacro(macros.fat) || !boundedMacro(macros.calories)
	) throw malformedResponse(status, requestId);

	return {
		id: value.id,
		name: value.name,
		entries,
		aggregateMacros: { protein: macros.protein, carbohydrates: macros.carbohydrates, fat: macros.fat, calories: macros.calories },
		createdAt: value.createdAt,
		updatedAt: value.updatedAt
	};
}

function malformedResponse(status: number, requestId?: string): DailyDietClientError {
	const appError: AppError = {
		category: "server",
		code: "malformed_daily_diet_response",
		message: "Saved daily diets returned an invalid response. Please try again.",
		retryable: true
	};
	if (requestId) appError.requestId = requestId;
	return new DailyDietClientError(appError, status);
}

function idempotencyKeyRequired(): DailyDietClientError {
	return new DailyDietClientError(
		{ category: "security", code: "daily_diet_idempotency_key_required", message: "A secure Daily Diet request could not be created. Please try again.", retryable: false },
		0
	);
}

function secureRandomUnavailable(): DailyDietClientError {
	return new DailyDietClientError(
		{ category: "security", code: "secure_random_unavailable", message: "A secure Daily Diet request could not be created. Please try again.", retryable: true },
		0
	);
}

function networkError(error: unknown): DailyDietClientError {
	if (error instanceof DailyDietClientError) return error;
	if (error instanceof DOMException && error.name === "AbortError") {
		return new DailyDietClientError(
			{ category: "timeout", code: "daily_diet_request_aborted", message: "The saved daily-diet request was cancelled. Please try again.", retryable: true },
			0
		);
	}
	return new DailyDietClientError(
		{ category: "network", code: "daily_diet_network_error", message: "Network is unavailable. Please check your connection and try again.", retryable: true },
		0
	);
}

function exactObject(value: unknown, keys: readonly string[]): value is Record<string, unknown> {
	if (typeof value !== "object" || value === null || Array.isArray(value)) return false;
	const actual = Object.keys(value).sort();
	const expected = [...keys].sort();
	return actual.length === expected.length && actual.every((key, index) => key === expected[index]);
}

function safeRequestId(value: unknown): value is string {
	return typeof value === "string" && /^[A-Za-z0-9._:-]{1,120}$/.test(value);
}

function validIdempotencyKey(value: unknown): value is IdempotencyKey {
	return typeof value === "string" && /^[\x21-\x7E]{8,255}$/.test(value);
}

function uuid(value: unknown): value is string {
	return typeof value === "string" && /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(value);
}

function randomUuidV4(value: unknown): value is `${string}-${string}-${string}-${string}-${string}` {
	return typeof value === "string" && /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(value);
}

function boundedString(value: unknown, minimum: number, maximum: number): value is string {
	return typeof value === "string" && value.length >= minimum && value.length <= maximum;
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
