import {
	AUTH_CSRF_TOKEN_ENDPOINT,
	buildCsrfTokenRequestInit
} from "./generated";
import type {
	AdminClassification,
	AdminClassificationCollectionEnvelope,
	AppError,
	CSRFTokenData,
	CSRFTokenEnvelope,
	CuratedImportEnvelope,
	CuratedImportRequest,
	CuratedImportResult,
	ErrorCategory,
	ErrorEnvelope,
	ExternalSearchData,
	ExternalSearchEnvelope
} from "./generated";

// Implements DESIGN-009 ExternalSearchProxy and DataImporter generated-contract client boundary.

const EXTERNAL_SEARCH_ENDPOINT = "/api/v1/admin/external-search";
const IMPORT_ENDPOINT = "/api/v1/admin/imports";
const CLASSIFICATIONS_ENDPOINT = "/api/v1/admin/classifications";
const MAX_SUCCESS_BYTES = 256 * 1024;
const MAX_ERROR_BYTES = 16 * 1024;
const SUCCESS_STATUS = { search: 200, classifications: 200, import: 201, csrf: 200 } as const;

type ExternalAdminOperation = keyof typeof SUCCESS_STATUS;
type StrictCsrfTokenEnvelope = CSRFTokenEnvelope & { status: "ok"; data: CSRFTokenData };

/** External provider selector accepted by the generated administration contract. */
export type ExternalProvider = "usda" | "openfoodfacts" | "all";

/** Safe failure exposed by external administration calls without provider or server diagnostics. */
export class ExternalAdminClientError extends Error {
	readonly appError: AppError;
	readonly status: number;
	readonly retryAfterSeconds?: number;

	constructor(appError: AppError, status: number, retryAfterSeconds?: number) {
		super(appError.message);
		this.name = "ExternalAdminClientError";
		this.appError = appError;
		this.status = status;
		this.retryAfterSeconds = retryAfterSeconds;
	}
}

/** Searches one external provider or the combined provider projection with bounded pagination. */
export async function searchExternalFoods(
	query: string,
	provider: ExternalProvider,
	page: number,
	signal?: AbortSignal
): Promise<ExternalSearchData> {
	const parameters = new URLSearchParams({ query: query.trim(), provider, page: String(page) });
	const response = await safeFetch(`${EXTERNAL_SEARCH_ENDPOINT}?${parameters}`, {
		method: "GET",
		credentials: "include",
		headers: { Accept: "application/json" },
		signal
	});
	const envelope = await decodeResponse<ExternalSearchEnvelope>(response, "search");
	if (!isExternalSearchData(envelope.data)) throw malformedResponse("search", response.status);
	return envelope.data;
}

/** Loads one backend-owned classification hierarchy for curation selection. */
export async function loadAdminClassifications(
	kind: "food_category" | "culinary_role",
	signal?: AbortSignal
): Promise<AdminClassification[]> {
	const response = await safeFetch(`${CLASSIFICATIONS_ENDPOINT}?kind=${kind}`, {
		method: "GET",
		credentials: "include",
		headers: { Accept: "application/json" },
		signal
	});
	const envelope = await decodeResponse<AdminClassificationCollectionEnvelope>(response, "classifications");
	if (!isClassificationData(envelope.data)) throw malformedResponse("classifications", response.status);
	return envelope.data.classifications;
}

/** Imports an edited curation draft with a caller-owned key that remains stable across retries. */
export async function importCuratedItem(
	request: CuratedImportRequest,
	idempotencyKey: string,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): Promise<CuratedImportResult> {
	const csrfToken = options.csrfToken ?? await fetchImportCsrfToken(options.signal);
	const response = await safeFetch(IMPORT_ENDPOINT, {
		method: "POST",
		credentials: "include",
		headers: {
			Accept: "application/json",
			"Content-Type": "application/json",
			"X-CSRF-Token": csrfToken,
			"Idempotency-Key": idempotencyKey
		},
		body: JSON.stringify(request),
		signal: options.signal
	});
	const envelope = await decodeResponse<CuratedImportEnvelope>(response, "import");
	if (!isImportResult(envelope.data)) throw malformedResponse("import", response.status);
	return envelope.data;
}

/** Creates one opaque browser-generated key for a curation attempt. */
export function createImportIdempotencyKey(): string {
	return crypto.randomUUID();
}

async function safeFetch(input: string, init: RequestInit): Promise<Response> {
	try {
		return await fetch(input, init);
	} catch (error) {
		const signalReason = init.signal?.aborted ? init.signal.reason : undefined;
		const timedOut = isTimeout(signalReason) || !init.signal?.aborted && isTimeout(error);
		if (timedOut) throw new ExternalAdminClientError(
			{
				category: "timeout",
				code: "external_request_timeout",
				message: "The request timed out. Try again.",
				retryable: true
			},
			0
		);
		if (init.signal?.aborted) throw signalReason ?? error;
		if (isAbort(error)) throw error;
		throw new ExternalAdminClientError(
			{
				category: "network",
				code: "external_request_ambiguous",
				message: "The result could not be confirmed. Retry the same request safely.",
				retryable: true
			},
			0
		);
	}
}

async function decodeResponse<T extends { status: "ok"; data: unknown }>(
	response: Response,
	operation: ExternalAdminOperation
): Promise<T> {
	if (response.ok && response.status !== SUCCESS_STATUS[operation]) {
		try { await response.body?.cancel(); } catch { /* The response is rejected even if stream cancellation fails. */ }
		throw malformedResponse(operation, response.status);
	}
	let body: unknown = null;
	try {
		body = JSON.parse(await readBoundedText(response, response.ok ? MAX_SUCCESS_BYTES : MAX_ERROR_BYTES)) as unknown;
	} catch (error) {
		if (isAbort(error)) throw error;
		if (response.ok) throw malformedResponse(operation, response.status);
	}
	if (!response.ok) throw safeResponseError(response, body, operation);
	if (!isRecord(body) || body.status !== "ok" || !safeRequestId(body.requestId) || !isRecord(body.data)) throw malformedResponse(operation, response.status);
	return body as T;
}

async function fetchImportCsrfToken(signal?: AbortSignal): Promise<string> {
	const response = await safeFetch(AUTH_CSRF_TOKEN_ENDPOINT, buildCsrfTokenRequestInit({ signal }));
	const envelope = await decodeResponse<StrictCsrfTokenEnvelope>(response, "csrf");
	if (!isCsrfTokenData(envelope.data)) throw malformedResponse("csrf", response.status);
	return envelope.data.csrfToken;
}

function safeResponseError(response: Response, body: unknown, operation: string): ExternalAdminClientError {
	const status = response.status;
	const source = isErrorEnvelope(body) ? body.error : undefined;
	const code = safeCodeForStatus(status, operation, typeof source?.code === "string" ? source.code : undefined);
	const category = categoryForStatus(status);
	const appError: AppError = {
		category,
		code,
		message: safeMessageForStatus(status, operation, code),
		retryable: source?.retryable === true || [429, 500, 503, 504].includes(status)
	};
	if (isErrorEnvelope(body) && safeRequestId(body.requestId)) appError.requestId = body.requestId;
	return new ExternalAdminClientError(appError, status, parseRetryAfter(response.headers.get("Retry-After")));
}

function malformedResponse(operation: string, status: number): ExternalAdminClientError {
	return new ExternalAdminClientError(
		{
			category: "server",
			code: `malformed_${operation}_response`,
			message: "The service returned an unexpected response. Try again.",
			retryable: true
		},
		status
	);
}

function safeMessageForStatus(status: number, operation: string, code = ""): string {
	if (status === 409 && code === "name_conflict_confirmation_required") return "A matching local item already exists. Confirm the name match before continuing.";
	if (status === 409 && code === "provider_identity_conflict") return "This provider item was already imported with different curated data. Refresh the external result before trying again.";
	if (status === 409 && code === "idempotency_key_conflict") return "This import attempt was already used for different data. Start a fresh import attempt before retrying.";
	if (status === 409) return "This item conflicts with existing data. Review the draft or refresh the external result.";
	if (status === 429) return "External providers are rate limited. Wait before trying again.";
	if (status === 504) return "The external provider timed out. Try again.";
	if (status === 503) return "External food data is temporarily unavailable. Try again later.";
	if (status === 401 || status === 403) return "Administration access is unavailable for this session.";
	if (status === 400 || status === 422) return operation === "import" ? "Review the curation fields and try again." : "Review the search and try again.";
	return "The administration service is temporarily unavailable. Try again.";
}

function safeCodeForStatus(status: number, operation: string, sourceCode?: string): string {
	if (status === 409 && operation === "import" && ["name_conflict_confirmation_required", "provider_identity_conflict", "idempotency_key_conflict"].includes(sourceCode ?? "")) return sourceCode!;
	if (status === 409) return "import_conflict";
	if (status === 429) return "provider_rate_limited";
	if (status === 504) return "provider_timeout";
	if (status === 503) return "provider_unavailable";
	return `${operation}_failed`;
}

function categoryForStatus(status: number): ErrorCategory {
	if (status === 429) return "rate_limit";
	if (status === 504) return "timeout";
	if (status === 503) return "dependency";
	if (status === 401 || status === 403) return "auth";
	if (status === 400 || status === 422 || status === 409) return "validation";
	return "server";
}

function parseRetryAfter(value: string | null): number | undefined {
	if (value === null || !/^\d+$/.test(value)) return undefined;
	const seconds = Number(value);
	return Number.isSafeInteger(seconds) && seconds > 0 ? Math.min(seconds, 3600) : undefined;
}

async function readBoundedText(response: Response, maximumBytes: number): Promise<string> {
	const declared = Number(response.headers.get("Content-Length"));
	if (Number.isFinite(declared) && declared > maximumBytes) {
		try { await response.body?.cancel(); } catch { /* The response remains rejected. */ }
		throw new Error("response body too large");
	}
	if (!response.body) return "";
	const reader = response.body.getReader();
	const decoder = new TextDecoder("utf-8", { fatal: true });
	let size = 0;
	let result = "";
	try {
		while (true) {
			const { done, value } = await reader.read();
			if (done) break;
			size += value.byteLength;
			if (size > maximumBytes) throw new Error("response body too large");
			result += decoder.decode(value, { stream: true });
		}
		return result + decoder.decode();
	} catch (error) {
		try { await reader.cancel(); } catch { /* Preserve the original decode or size failure. */ }
		throw error;
	} finally { reader.releaseLock(); }
}

function safeRequestId(value: unknown): value is string {
	return typeof value === "string" && /^[A-Za-z0-9][A-Za-z0-9._:-]{0,119}$/.test(value);
}

function isCsrfTokenData(value: unknown): value is CSRFTokenData {
	return exact(value, ["csrfToken"]) && boundedString(value.csrfToken, 1, 4096);
}

function isErrorEnvelope(value: unknown): value is ErrorEnvelope {
	return isRecord(value) && value.status === "error" && typeof value.requestId === "string" && isRecord(value.error);
}

function isExternalSearchData(value: unknown): value is ExternalSearchData {
	return exact(value, ["candidates", "warnings", "page"])
		&& Array.isArray(value.candidates) && value.candidates.length <= 40 && value.candidates.every(isExternalCandidate)
		&& Array.isArray(value.warnings) && value.warnings.length <= 4 && value.warnings.every(isExternalDataWarning)
		&& positiveInteger(value.page) && value.page <= 10_000;
}

function isImportResult(value: unknown): value is CuratedImportResult {
	return exact(value, ["importId", "foodItemId", "name", "physicalState", "merged", "replayed"])
		&& uuid(value.importId) && uuid(value.foodItemId) && boundedString(value.name, 1, 200)
		&& (value.physicalState === "solid" || value.physicalState === "liquid")
		&& typeof value.merged === "boolean" && typeof value.replayed === "boolean";
}

function isClassificationData(value: unknown): value is { classifications: AdminClassification[] } {
	return exact(value, ["classifications"])
		&& Array.isArray(value.classifications) && value.classifications.length <= 1000
		&& value.classifications.every((classification) => exact(classification, ["id", "name", "kind"], ["parentId"])
			&& uuid(classification.id) && boundedString(classification.name, 1, 120)
			&& (classification.kind === "food_category" || classification.kind === "culinary_role")
			&& (classification.parentId === undefined || uuid(classification.parentId)));
}

function isExternalCandidate(value: unknown): boolean {
	if (!exact(value, ["provider", "externalId", "name", "physicalState", "macrosPer100", "micronutrients", "warnings"], ["imageUrl"])) return false;
	if ((value.provider !== "usda" && value.provider !== "openfoodfacts") || !boundedString(value.externalId, 1, 200) || !boundedString(value.name, 1, 1000)) return false;
	if (value.physicalState !== "solid" && value.physicalState !== "liquid") return false;
	if (!isMacroProfile(value.macrosPer100) || !isNumericMap(value.micronutrients, 512)) return false;
	if (!Array.isArray(value.warnings) || value.warnings.length > 8 || new Set(value.warnings).size !== value.warnings.length || !value.warnings.every(isCandidateWarning)) return false;
	return value.imageUrl === undefined || boundedString(value.imageUrl, 1, 2048) && isUri(value.imageUrl);
}

function isExternalDataWarning(value: unknown): boolean {
	if (!exact(value, ["provider", "code", "message"])) return false;
	return ["usda", "openfoodfacts", "external"].includes(String(value.provider))
		&& isProviderWarningCode(value.code) && value.message === value.code;
}

function isMacroProfile(value: unknown): boolean {
	return exact(value, ["protein", "carbohydrates", "fat"])
		&& [value.protein, value.carbohydrates, value.fat].every(nonnegativeFiniteNumber);
}

function isNumericMap(value: unknown, maximumProperties: number): boolean {
	if (!isRecord(value)) return false;
	const entries = Object.entries(value);
	return entries.length <= maximumProperties && entries.every(([key, number]) => boundedString(key, 1, 120) && !key.includes("\0") && nonnegativeFiniteNumber(number));
}

function isCandidateWarning(value: unknown): boolean {
	return typeof value === "string" && ["missing_image", "missing_macros", "missing_micronutrients", "missing_liquid_density", "uncertain_unit_conversion", "suspicious_liquid_macros"].includes(value);
}

function isProviderWarningCode(value: unknown): boolean {
	return typeof value === "string" && ["provider_rate_limited", "provider_unavailable", "timeout", "retry_exhausted", "invalid_external_payload"].includes(value);
}

function exact(value: unknown, required: string[], optional: string[] = []): value is Record<string, unknown> {
	if (!isRecord(value)) return false;
	const keys = Object.keys(value);
	return required.every((key) => key in value) && keys.every((key) => required.includes(key) || optional.includes(key));
}

function boundedString(value: unknown, minimum: number, maximum: number): value is string {
	return typeof value === "string" && value.length >= minimum && value.length <= maximum;
}

function nonnegativeFiniteNumber(value: unknown): value is number {
	return typeof value === "number" && Number.isFinite(value) && value >= 0;
}

function positiveInteger(value: unknown): value is number {
	return typeof value === "number" && Number.isInteger(value) && value >= 1;
}

function uuid(value: unknown): value is string {
	return typeof value === "string" && /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(value);
}

function isUri(value: string): boolean {
	try {
		new URL(value);
		return true;
	} catch {
		return false;
	}
}

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null;
}

function isAbort(error: unknown): error is DOMException {
	return isDOMExceptionNamed(error, "AbortError");
}

function isTimeout(error: unknown): error is DOMException {
	return isDOMExceptionNamed(error, "TimeoutError");
}

function isDOMExceptionNamed(error: unknown, name: string): error is DOMException {
	if (error instanceof DOMException) return error.name === name;
	try {
		return Object.getOwnPropertyDescriptor(DOMException.prototype, "name")?.get?.call(error) === name;
	} catch {
		return false;
	}
}
