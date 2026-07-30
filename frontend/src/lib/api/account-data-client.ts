import { fetchCsrfToken } from "./auth-client";
import {
	buildAccountExportRequestInit,
	buildAccountExportUrl,
	buildCustomItemDeleteRequestInit,
	buildCustomItemUrl
} from "./generated";
import type { ExportBundle, ExportCustomItem } from "./generated";

// Implements DESIGN-008 DataExporter and ProfileController generated-contract client.

const MAX_EXPORT_BYTES = 1024 * 1024;

/** Safe failure exposed by authenticated Account Export operations. */
export class AccountDataClientError extends Error {
	constructor(message = "Account data could not be refreshed. Try again.") {
		super(message);
		this.name = "AccountDataClientError";
	}
}

/** Loads the authenticated JSON export and validates the private-item projection used by the panel. */
export async function loadAccountExport(signal?: AbortSignal): Promise<ExportBundle> {
	const response = await request(buildAccountExportUrl("json"), buildAccountExportRequestInit("json", { signal }));
	if (response.status !== 200) throw new AccountDataClientError();
	const body = await readBoundedText(response, MAX_EXPORT_BYTES);
	let value: unknown;
	try { value = JSON.parse(body) as unknown; } catch { throw new AccountDataClientError(); }
	if (!isRecord(value) || !isRecord(value.user) || !Array.isArray(value.consent) || !Array.isArray(value.savedItems) || !Array.isArray(value.savedDiets) || !Array.isArray(value.history) || !Array.isArray(value.customItems)) throw new AccountDataClientError();
	assertNoNestedOwnership(value);
	assertExactKeys(value, ["user", "consent", "savedItems", "savedDiets", "history", "customItems"]);
	assertUser(value.user);
	value.consent.forEach(assertConsent);
	value.savedItems.forEach(assertSavedItem);
	value.savedDiets.forEach(assertSavedDiet);
	value.history.forEach(assertSearchHistory);
	value.customItems.forEach(assertCustomItem);
	return value as unknown as ExportBundle;
}

/** Deletes one private item through the generated owner-scoped URL and CSRF request builder. */
export async function deletePrivateCustomItem(itemId: string, signal?: AbortSignal): Promise<void> {
	if (!uuid(itemId)) throw new AccountDataClientError("The private item identifier is invalid.");
	const { csrfToken } = await fetchCsrfToken(signal);
	const response = await request(buildCustomItemUrl(itemId), buildCustomItemDeleteRequestInit(csrfToken, { signal }));
	if (response.status !== 204 || (await readBoundedText(response, 0)) !== "") throw new AccountDataClientError("The private item could not be deleted. Try again.");
}

/** Injectable Account Export and private-item mutation operations. */
export interface AccountDataApi {
	loadExport: typeof loadAccountExport;
	deleteCustomItem: typeof deletePrivateCustomItem;
}

/** Account Export operations exposed to the Administration Panel. */
export const accountDataApi: AccountDataApi = { loadExport: loadAccountExport, deleteCustomItem: deletePrivateCustomItem };

function assertUser(value: Record<string, unknown>): void {
	assertExactKeys(value, ["userId", "email", "role", "displayName", "unitSystem", "themePreference"]);
	if (!uuid(value.userId) || !nonempty(value.email) || (value.role !== "user" && value.role !== "admin") || typeof value.displayName !== "string" || (value.unitSystem !== "metric" && value.unitSystem !== "imperial") || !["system", "light", "dark"].includes(String(value.themePreference))) throw new AccountDataClientError();
}

function assertConsent(value: unknown): void {
	if (!isRecord(value)) throw new AccountDataClientError();
	assertExactKeys(value, ["privacyPolicyVersion", "termsVersion"]);
	if (!nonempty(value.privacyPolicyVersion) || !nonempty(value.termsVersion)) throw new AccountDataClientError();
}

function assertSavedItem(value: unknown): void {
	if (!isRecord(value)) throw new AccountDataClientError();
	assertExactKeys(value, ["id", "itemId", "kind", "createdAt"]);
	if (!uuid(value.id) || !uuid(value.itemId) || !["favorite", "saved_meal", "saved_diet"].includes(String(value.kind)) || !timestamp(value.createdAt)) throw new AccountDataClientError();
}

function assertSavedDiet(value: unknown): void {
	if (!isRecord(value)) throw new AccountDataClientError();
	assertExactKeys(value, ["id", "name", "entries", "createdAt", "updatedAt"]);
	if (!uuid(value.id) || !nonempty(value.name) || value.name.length > 120 || !Array.isArray(value.entries) || !timestamp(value.createdAt) || !timestamp(value.updatedAt)) throw new AccountDataClientError();
	value.entries.forEach(assertSavedDietEntry);
}

function assertSavedDietEntry(value: unknown): void {
	if (!isRecord(value)) throw new AccountDataClientError();
	assertExactKeys(value, ["id", "foodObjectId", "foodObjectType", "quantity", "unit", "position"]);
	if (!uuid(value.id) || !uuid(value.foodObjectId) || !["food_item", "meal"].includes(String(value.foodObjectType)) || !positive(value.quantity) || !["g", "ml", "oz", "fl_oz"].includes(String(value.unit)) || !Number.isInteger(value.position) || Number(value.position) < 0 || Number(value.position) > 99) throw new AccountDataClientError();
}

function assertSearchHistory(value: unknown): void {
	if (!isRecord(value)) throw new AccountDataClientError();
	assertExactKeys(value, ["id", "query", "mode", "filtersHash", "createdAt"]);
	if (!uuid(value.id) || typeof value.query !== "string" || !nonempty(value.mode) || typeof value.filtersHash !== "string" || !timestamp(value.createdAt)) throw new AccountDataClientError();
}

function assertCustomItem(value: unknown): asserts value is ExportCustomItem {
	if (!isRecord(value)) throw new AccountDataClientError();
	assertExactKeys(value, ["id", "name", "physicalState", "prepTimeMinutes", "macrosPer100", "micros", "foodCategories", "culinaryRoles"], ["averageUnitWeightGrams", "averageServingVolumeMilliliters", "densityGramsPerMilliliter", "densitySourceProvider", "densitySourceFoodId", "densitySourceKind", "imageUrl"]);
	if (!uuid(value.id) || !nonempty(value.name) || value.name.length > 200 || !["solid", "liquid"].includes(String(value.physicalState)) || !Number.isInteger(value.prepTimeMinutes) || Number(value.prepTimeMinutes) < 0 || !isRecord(value.macrosPer100) || !isRecord(value.micros) || !Array.isArray(value.foodCategories) || !Array.isArray(value.culinaryRoles)) throw new AccountDataClientError();
	assertExactKeys(value.macrosPer100, ["protein", "carbohydrates", "fat"]);
	if (!nonnegative(value.macrosPer100.protein) || !nonnegative(value.macrosPer100.carbohydrates) || !nonnegative(value.macrosPer100.fat) || Object.entries(value.micros).some(([key, item]) => Array.from(key).length < 1 || Array.from(key).length > 120 || !nonnegative(item))) throw new AccountDataClientError();
	value.foodCategories.forEach(assertClassification);
	value.culinaryRoles.forEach(assertClassification);
	for (const field of ["averageUnitWeightGrams", "averageServingVolumeMilliliters", "densityGramsPerMilliliter"]) if (field in value && !positive(value[field])) throw new AccountDataClientError();
	for (const field of ["densitySourceProvider", "densitySourceFoodId", "imageUrl"]) if (field in value && typeof value[field] !== "string") throw new AccountDataClientError();
	if ("densitySourceKind" in value && !["imported", "manual", "estimated"].includes(String(value.densitySourceKind))) throw new AccountDataClientError();
}

function assertClassification(value: unknown): void {
	if (!isRecord(value)) throw new AccountDataClientError();
	assertExactKeys(value, ["id", "name", "kind"]);
	if (!uuid(value.id) || !nonempty(value.name) || !["food_category", "culinary_role"].includes(String(value.kind))) throw new AccountDataClientError();
}

function assertNoNestedOwnership(value: unknown, path: string[] = []): void {
	if (Array.isArray(value)) {
		value.forEach((item, index) => assertNoNestedOwnership(item, [...path, String(index)]));
		return;
	}
	if (!isRecord(value)) return;
	for (const [key, child] of Object.entries(value)) {
		const normalized = key.replace(/[_-]/g, "").toLowerCase();
		const topLevelIdentity = path.length === 1 && path[0] === "user" && key === "userId";
		if (!topLevelIdentity && (normalized === "userid" || normalized === "ownerid" || normalized === "owner")) throw new AccountDataClientError();
		assertNoNestedOwnership(child, [...path, key]);
	}
}

function assertExactKeys(value: Record<string, unknown>, required: string[], optional: string[] = []): void {
	const keys = Object.keys(value);
	if (required.some((key) => !(key in value)) || keys.some((key) => !required.includes(key) && !optional.includes(key))) throw new AccountDataClientError();
}

async function request(input: string, init: RequestInit): Promise<Response> {
	try { return await fetch(input, init); }
	catch (error) { if (init.signal?.aborted) throw error; throw new AccountDataClientError(); }
}

async function readBoundedText(response: Response, maximum: number): Promise<string> {
	const declared = response.headers.get("content-length");
	if (declared !== null && /^\d+$/.test(declared) && Number(declared) > maximum) throw new AccountDataClientError();
	const reader = response.body?.getReader();
	if (!reader) return "";
	const chunks: Uint8Array[] = [];
	let size = 0;
	try {
		while (true) {
			const { value, done } = await reader.read();
			if (done) break;
			size += value.byteLength;
			if (size > maximum) throw new AccountDataClientError();
			chunks.push(value);
		}
	} catch (error) { try { await reader.cancel(); } catch { /* The bounded rejection is authoritative. */ } throw error; }
	const bytes = new Uint8Array(size);
	let offset = 0;
	for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.byteLength; }
	return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
}

function isRecord(value: unknown): value is Record<string, unknown> { return typeof value === "object" && value !== null && !Array.isArray(value); }
function uuid(value: unknown): value is string { return typeof value === "string" && /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(value); }
function nonempty(value: unknown): value is string { return typeof value === "string" && value.trim() !== ""; }
function timestamp(value: unknown): value is string { return typeof value === "string" && Number.isFinite(Date.parse(value)); }
function nonnegative(value: unknown): value is number { return typeof value === "number" && Number.isFinite(value) && value >= 0; }
function positive(value: unknown): value is number { return nonnegative(value) && value > 0; }
