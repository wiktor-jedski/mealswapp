import { fetchCsrfToken } from "./auth-client";
import type {
	AdminClassification,
	AdminClassificationRequest,
	AdminItem,
	AdminItemRequest,
	AdminUser,
	AdminUserPageData,
	AppError,
	IdempotencyKey
} from "./generated";

// Implements DESIGN-009 AdminController generated-contract client for ItemCurator, TagManager, and UserAdminPanel.

export type ClassificationKind = AdminClassification["kind"];

export interface AdminMutationOptions {
	csrfToken?: string;
	signal?: AbortSignal;
}

export class AdminClientError extends Error {
	constructor(readonly status: number, readonly appError: AppError) {
		super(appError.message);
		this.name = "AdminClientError";
	}
}

const MAX_REQUEST_BYTES = 64 * 1024;
const MAX_SUCCESS_BYTES = 256 * 1024;
const MAX_ERROR_BYTES = 16 * 1024;
const MAX_NUTRITION_VALUE = 99_999_999.9999;
const SAFE_ERROR_CODES = new Set(["classification_in_use", "audit_write_failed", "conflict", "not_found", "validation_failed", "unauthorized", "forbidden", "dependency_unavailable", "internal_error", "rate_limited"]);

/** Reads one active ownerless global item. */
export async function getAdminItem(itemId: string, signal?: AbortSignal): Promise<AdminItem> {
	return decodeItem(await request(`/api/v1/admin/items/${encodeURIComponent(itemId)}`, { method: "GET", signal }), 200);
}

/** Creates one global item with a caller-owned retry-stable idempotency key. */
export async function createAdminItem(requestBody: AdminItemRequest, idempotencyKey: IdempotencyKey, options: AdminMutationOptions = {}): Promise<AdminItem> {
	return decodeItem(await mutation("/api/v1/admin/items", "POST", requestBody, options, { "Idempotency-Key": idempotencyKey }), 201);
}

/** Replaces one global item and returns only the server projection. */
export async function replaceAdminItem(itemId: string, requestBody: AdminItemRequest, options: AdminMutationOptions = {}): Promise<AdminItem> {
	return decodeItem(await mutation(`/api/v1/admin/items/${encodeURIComponent(itemId)}`, "PUT", requestBody, options), 200);
}

/** Soft-deletes one global item only after an empty 204 response. */
export async function deleteAdminItem(itemId: string, options: AdminMutationOptions = {}): Promise<void> {
	await emptyMutation(`/api/v1/admin/items/${encodeURIComponent(itemId)}`, "DELETE", options);
}

/** Lists one deterministic global classification hierarchy. */
export async function listAdminClassifications(kind: ClassificationKind, signal?: AbortSignal): Promise<AdminClassification[]> {
	const response = await request(`/api/v1/admin/classifications?kind=${kind}`, { method: "GET", signal });
	const data = decodeData(await json(response, 200), response.status);
	if (!exact(data, ["classifications"]) || !Array.isArray(data.classifications) || data.classifications.length > 1000) throw malformed(response.status);
	return data.classifications.map((value) => decodeClassification(value, response.status));
}

/** Creates one Food Category or Culinary Role. */
export async function createAdminClassification(kind: ClassificationKind, body: AdminClassificationRequest, options: AdminMutationOptions = {}): Promise<AdminClassification> {
	return decodeClassificationEnvelope(await mutation(`/api/v1/admin/classifications/${kind}`, "POST", body, options), 201);
}

/** Renames or reparents one global classification. */
export async function replaceAdminClassification(id: string, body: AdminClassificationRequest, options: AdminMutationOptions = {}): Promise<AdminClassification> {
	return decodeClassificationEnvelope(await mutation(`/api/v1/admin/classifications/${encodeURIComponent(id)}`, "PUT", body, options), 200);
}

/** Deletes one unused global classification after an empty 204 response. */
export async function deleteAdminClassification(id: string, options: AdminMutationOptions = {}): Promise<void> {
	await emptyMutation(`/api/v1/admin/classifications/${encodeURIComponent(id)}`, "DELETE", options);
}

/** Performs an exact or bounded privacy-minimized user lookup. */
export async function lookupAdminUsers(query: { userId?: string; email?: string; cursor?: string; limit?: number }, signal?: AbortSignal): Promise<AdminUserPageData> {
	const parameters = new URLSearchParams();
	if (query.userId) parameters.set("userId", query.userId);
	if (query.email) parameters.set("email", query.email);
	if (query.cursor) parameters.set("cursor", query.cursor);
	if (query.limit !== undefined) parameters.set("limit", String(query.limit));
	const response = await request(`/api/v1/admin/users?${parameters}`, { method: "GET", signal });
	const data = decodeData(await json(response, 200), response.status);
	if (!exact(data, ["users"], ["nextCursor"]) || !Array.isArray(data.users) || data.users.length > 25) throw malformed(response.status);
	if (data.nextCursor !== undefined && !uuid(data.nextCursor)) throw malformed(response.status);
	return { users: data.users.map((value) => decodeUser(value, response.status)), ...(data.nextCursor ? { nextCursor: data.nextCursor } : {}) };
}

/** Requests one scoped legal deletion retry; callers must subsequently refresh user state. */
export async function retryAdminDeletion(userId: string, requestId: string, options: AdminMutationOptions = {}): Promise<void> {
	const response = await mutation(`/api/v1/admin/users/${encodeURIComponent(userId)}/deletion-requests/${encodeURIComponent(requestId)}/retry`, "POST", undefined, options);
	const data = decodeData(await json(response, 200), response.status);
	if (!exact(data, ["requestId", "status"]) || data.requestId !== requestId || data.status !== "pending") throw malformed(response.status);
}

export interface AdminApi {
	getItem: typeof getAdminItem;
	createItem: typeof createAdminItem;
	replaceItem: typeof replaceAdminItem;
	deleteItem: typeof deleteAdminItem;
	listClassifications: typeof listAdminClassifications;
	createClassification: typeof createAdminClassification;
	replaceClassification: typeof replaceAdminClassification;
	deleteClassification: typeof deleteAdminClassification;
	lookupUsers: typeof lookupAdminUsers;
	retryDeletion: typeof retryAdminDeletion;
}

export const adminApi: AdminApi = {
	getItem: getAdminItem,
	createItem: createAdminItem,
	replaceItem: replaceAdminItem,
	deleteItem: deleteAdminItem,
	listClassifications: listAdminClassifications,
	createClassification: createAdminClassification,
	replaceClassification: replaceAdminClassification,
	deleteClassification: deleteAdminClassification,
	lookupUsers: lookupAdminUsers,
	retryDeletion: retryAdminDeletion
};

async function mutation(url: string, method: "POST" | "PUT", body: unknown, options: AdminMutationOptions, headers: Record<string, string> = {}): Promise<Response> {
	let payload: string | undefined;
	try { payload = body === undefined ? undefined : JSON.stringify(body); } catch { throw invalidRequest(); }
	if (payload !== undefined && new TextEncoder().encode(payload).byteLength > MAX_REQUEST_BYTES) throw invalidRequest();
	const csrfToken = options.csrfToken ?? (await fetchCsrfToken(options.signal)).csrfToken;
	return request(url, {
		method,
		credentials: "include",
		headers: { Accept: "application/json", "Content-Type": "application/json", "X-CSRF-Token": csrfToken, ...headers },
		...(payload === undefined ? {} : { body: payload }),
		signal: options.signal
	});
}

async function emptyMutation(url: string, method: "DELETE", options: AdminMutationOptions): Promise<void> {
	const csrfToken = options.csrfToken ?? (await fetchCsrfToken(options.signal)).csrfToken;
	const response = await request(url, { method, credentials: "include", headers: { Accept: "application/json", "X-CSRF-Token": csrfToken }, signal: options.signal });
	if (response.status !== 204) { await cancelResponseBody(response); throw malformed(response.status); }
	if ((await readBoundedText(response, 0)) !== "") throw malformed(response.status);
}

async function request(url: string, init: RequestInit): Promise<Response> {
	let response: Response;
	try {
		response = await fetch(url, { credentials: "include", headers: { Accept: "application/json", ...init.headers }, ...init });
	} catch (error) {
		if (isAbort(error)) throw error;
		throw new AdminClientError(0, { category: "network", code: "network_error", message: "The administration service could not be reached. Try again.", retryable: true });
	}
	if (!response.ok) throw await responseError(response);
	return response;
}

async function json(response: Response, expectedStatus: number): Promise<unknown> {
	if (response.status !== expectedStatus) { await cancelResponseBody(response); throw malformed(response.status); }
	try { return JSON.parse(await readBoundedText(response, MAX_SUCCESS_BYTES)) as unknown; } catch (error) { if (error instanceof AdminClientError) throw error; throw malformed(response.status); }
}

function decodeData(value: unknown, status: number): unknown {
	if (!exact(value, ["status", "requestId", "data"]) || value.status !== "ok" || !boundedString(value.requestId, 1, 128, false)) throw malformed(status);
	return value.data;
}

function decodeItem(responsePromise: Promise<Response> | Response, expectedStatus: number): Promise<AdminItem> {
	return Promise.resolve(responsePromise).then(async (response) => {
		const value = decodeData(await json(response, expectedStatus), response.status);
		const optional = ["averageUnitWeightGrams", "averageServingVolumeMilliliters", "densityGramsPerMilliliter", "densitySourceProvider", "densitySourceFoodId", "densitySourceKind", "foodCategoryIds", "culinaryRoleIds", "imageUrl"];
		if (!exact(value, ["id", "name", "physicalState", "prepTimeMinutes", "macrosPer100", "micros", "foodCategories", "culinaryRoles"], optional) || !uuid(value.id) || !boundedString(value.name, 1, 200) || (value.physicalState !== "solid" && value.physicalState !== "liquid") || !nonnegativeInteger(value.prepTimeMinutes) || value.prepTimeMinutes > MAX_NUTRITION_VALUE || !macroProfile(value.macrosPer100) || !micronutrients(value.micros) || !Array.isArray(value.foodCategories) || value.foodCategories.length > 100 || !Array.isArray(value.culinaryRoles) || value.culinaryRoles.length > 100) throw malformed(response.status);
		if (!optionalPositive(value.averageUnitWeightGrams) || !optionalPositive(value.averageServingVolumeMilliliters) || !optionalPositive(value.densityGramsPerMilliliter) || !optionalBoundedString(value.densitySourceProvider, 200) || !optionalBoundedString(value.densitySourceFoodId, 200) || (value.densitySourceKind !== undefined && !["imported", "manual", "estimated"].includes(String(value.densitySourceKind))) || !optionalBoundedString(value.imageUrl, 2048) || (value.imageUrl !== undefined && !safeUriReference(value.imageUrl))) throw malformed(response.status);
		if (!optionalUuidCollection(value.foodCategoryIds) || !optionalUuidCollection(value.culinaryRoleIds)) throw malformed(response.status);
		if (value.physicalState === "solid" && (value.macrosPer100.protein as number) + (value.macrosPer100.carbohydrates as number) + (value.macrosPer100.fat as number) > 100) throw malformed(response.status);
		if (value.physicalState === "solid" && [value.averageServingVolumeMilliliters, value.densityGramsPerMilliliter, value.densitySourceProvider, value.densitySourceFoodId, value.densitySourceKind].some((field) => field !== undefined)) throw malformed(response.status);
		if (value.physicalState === "liquid" && (value.densityGramsPerMilliliter === undefined || value.densitySourceKind === undefined)) throw malformed(response.status);
		if (value.densitySourceKind === "imported" && (!value.densitySourceFoodId || !["usda", "openfoodfacts"].includes(String(value.densitySourceProvider)))) throw malformed(response.status);
		value.foodCategories.forEach((classification) => decodeClassificationSummary(classification, "food_category", response.status));
		value.culinaryRoles.forEach((classification) => decodeClassificationSummary(classification, "culinary_role", response.status));
		return value as unknown as AdminItem;
	});
}

async function decodeClassificationEnvelope(responsePromise: Promise<Response> | Response, expectedStatus: number): Promise<AdminClassification> {
	const response = await responsePromise;
	const data = decodeData(await json(response, expectedStatus), response.status);
	if (!exact(data, ["classification"])) throw malformed(response.status);
	return decodeClassification(data.classification, response.status);
}

function decodeClassification(value: unknown, status: number): AdminClassification {
	if (!exact(value, ["id", "name", "kind"], ["parentId"]) || !uuid(value.id) || !boundedString(value.name, 1, 120) || (value.kind !== "food_category" && value.kind !== "culinary_role") || (value.parentId !== undefined && !uuid(value.parentId))) throw malformed(status);
	return value as unknown as AdminClassification;
}

function decodeUser(value: unknown, status: number): AdminUser {
	if (!exact(value, ["id", "email", "emailVerified", "createdAt"], ["deletion"]) || !uuid(value.id) || !email(value.email) || typeof value.emailVerified !== "boolean" || !dateTime(value.createdAt)) throw malformed(status);
	if (value.deletion !== undefined) {
		const deletion = value.deletion;
		if (!exact(deletion, ["requestId", "status", "retryCount", "requestedAt"], ["failureCategory"]) || !uuid(deletion.requestId) || !["pending", "processing", "completed", "failed"].includes(String(deletion.status)) || !nonnegativeInteger(deletion.retryCount) || (deletion.failureCategory !== undefined && !["transient", "permanent", "unknown"].includes(String(deletion.failureCategory))) || !dateTime(deletion.requestedAt)) throw malformed(status);
	}
	return value as unknown as AdminUser;
}

async function responseError(response: Response): Promise<AdminClientError> {
	let code = "admin_request_failed";
	try {
		const value = JSON.parse(await readBoundedText(response, MAX_ERROR_BYTES)) as unknown;
		if (record(value) && record(value.error) && typeof value.error.code === "string" && SAFE_ERROR_CODES.has(value.error.code)) code = value.error.code;
	} catch { /* Status and approved code provide the safe fallback. */ }
	const status = safeErrorStatus(response.status);
	const conflict = status === 409;
	return new AdminClientError(status, {
		category: conflict ? "validation" : status >= 500 ? "server" : "unknown",
		code,
		message: conflict ? "The record changed or conflicts with authoritative data. It has been refreshed." : "The administration action did not complete. No change was shown as successful.",
		retryable: conflict || status >= 500
	});
}

function malformed(status: number): AdminClientError {
	return new AdminClientError(Number.isInteger(status) && status >= 100 && status <= 599 ? status : 0, { category: "server", code: "malformed_admin_response", message: "The administration service returned an invalid response. No change was shown as successful.", retryable: true });
}

function invalidRequest(): AdminClientError {
	return new AdminClientError(0, { category: "validation", code: "invalid_admin_request", message: "The administration request is too large or cannot be encoded.", retryable: false });
}

async function cancelResponseBody(response: Response): Promise<void> {
	try { await response.body?.cancel(); } catch { /* Preserve the safe status error if transport cleanup fails. */ }
}

async function readBoundedText(response: Response, maximumBytes: number): Promise<string> {
	const declared = Number(response.headers.get("Content-Length"));
	if (Number.isFinite(declared) && declared > maximumBytes) {
		try { await response.body?.cancel(); } catch { /* Preserve the bounded-response error if transport cleanup fails. */ }
		throw malformed(response.status);
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
			if (size > maximumBytes) throw malformed(response.status);
			result += decoder.decode(value, { stream: true });
		}
		return result + decoder.decode();
	} catch (error) {
		await reader.cancel().catch(() => undefined);
		throw error;
	} finally { reader.releaseLock(); }
}

function decodeClassificationSummary(value: unknown, kind: ClassificationKind, status: number): void {
	if (!exact(value, ["id", "name", "kind"]) || !uuid(value.id) || !boundedString(value.name, 1, 200) || value.kind !== kind) throw malformed(status);
}

function macroProfile(value: unknown): value is Record<"protein" | "carbohydrates" | "fat", number> {
	return exact(value, ["protein", "carbohydrates", "fat"]) && [value.protein, value.carbohydrates, value.fat].every((field) => finiteBetween(field, 0, MAX_NUTRITION_VALUE));
}

function micronutrients(value: unknown): value is Record<string, number> {
	return record(value) && Object.keys(value).length <= 200 && Object.entries(value).every(([key, field]) => key.length >= 1 && key.length <= 120 && !key.includes("\0") && finiteBetween(field, 0, MAX_NUTRITION_VALUE));
}

function optionalUuidCollection(value: unknown): boolean {
	return value === undefined || Array.isArray(value) && value.length <= 100 && new Set(value).size === value.length && value.every(uuid);
}

function optionalPositive(value: unknown): boolean { return value === undefined || finiteBetween(value, Number.MIN_VALUE, MAX_NUTRITION_VALUE); }
function optionalBoundedString(value: unknown, maximum: number): boolean { return value === undefined || boundedString(value, 0, maximum, false); }
function finiteBetween(value: unknown, minimum: number, maximum: number): value is number { return typeof value === "number" && Number.isFinite(value) && value >= minimum && value <= maximum; }
function nonnegativeInteger(value: unknown): value is number { return Number.isSafeInteger(value) && (value as number) >= 0; }
function boundedString(value: unknown, minimum: number, maximum: number, trim = true): value is string { return typeof value === "string" && value.length >= minimum && value.length <= maximum && !value.includes("\0") && (!trim || value.trim() === value); }
function email(value: unknown): value is string { return boundedString(value, 3, 320) && /^[^\s@]+@[^\s@]+$/.test(value); }
function dateTime(value: unknown): value is string {
	if (!boundedString(value, 20, 40, false)) return false;
	const match = /^(\d{4})-(\d{2})-(\d{2})T(?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d(?:\.\d+)?(?:Z|[+-](?:[01]\d|2[0-3]):[0-5]\d)$/.exec(value);
	if (!match) return false;
	const [year, month, day] = match.slice(1, 4).map(Number) as [number, number, number];
	const daysInMonth = [31, year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0) ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
	return month >= 1 && month <= 12 && day >= 1 && day <= daysInMonth[month - 1]!;
}
function safeUriReference(value: unknown): value is string {
	if (!boundedString(value, 1, 2048, false)) return false;
	try { const parsed = new URL(value, "https://mealswapp.invalid"); return parsed.protocol === "http:" || parsed.protocol === "https:"; } catch { return false; }
}
function safeErrorStatus(status: number): number { return Number.isInteger(status) && status >= 400 && status <= 599 ? status : 0; }
function isAbort(error: unknown): boolean { return error instanceof DOMException && error.name === "AbortError" || error instanceof Error && error.name === "AbortError"; }

function record(value: unknown): value is Record<string, unknown> { return typeof value === "object" && value !== null && !Array.isArray(value); }
function exact(value: unknown, required: string[], optional: string[] = []): value is Record<string, unknown> {
	if (!record(value) || required.some((key) => !(key in value))) return false;
	const allowed = new Set([...required, ...optional]);
	return Object.keys(value).every((key) => allowed.has(key));
}
function uuid(value: unknown): value is string { return typeof value === "string" && /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(value); }
