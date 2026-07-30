import { fetchCsrfToken } from "./auth-client";
import {
	buildAccountExportRequestInit,
	buildAccountExportUrl,
	buildCustomItemDeleteRequestInit,
	buildCustomItemUrl
} from "./generated";
import type { CustomItem, ExportBundle } from "./generated";

// Implements DESIGN-008 DataExporter and ProfileController generated-contract client.

const MAX_EXPORT_BYTES = 1024 * 1024;

/** Safe failure exposed by authenticated Account Export operations. */
export class AccountDataClientError extends Error {
	readonly affectedDiets?: Array<{ id: string; name: string }>;
	constructor(message = "Account data could not be refreshed. Try again.", affectedDiets?: Array<{ id: string; name: string }>) {
		super(message);
		this.name = "AccountDataClientError";
		this.affectedDiets = affectedDiets;
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
	value.savedDiets.forEach(assertSavedDiet);
	value.customItems.forEach(assertCustomItemSummary);
	return value as unknown as ExportBundle;
}

/** Deletes one private item through the generated owner-scoped URL and CSRF request builder. */
export async function deletePrivateCustomItem(itemId: string, signal?: AbortSignal): Promise<void> {
	if (!uuid(itemId)) throw new AccountDataClientError("The private item identifier is invalid.");
	const { csrfToken } = await fetchCsrfToken(signal);
	const response = await request(buildCustomItemUrl(itemId), buildCustomItemDeleteRequestInit(csrfToken, { signal }));
	if (response.status !== 204) {
		let affectedDiets: Array<{ id: string; name: string }> | undefined;
		try {
			const value = JSON.parse(await readBoundedText(response, MAX_EXPORT_BYTES)) as Record<string, unknown>;
			const data = value.error && typeof value.error === "object" ? (value.error as Record<string, unknown>).data : undefined;
			const diets = data && typeof data === "object" ? (data as Record<string, unknown>).affectedDiets : undefined;
			if (Array.isArray(diets)) affectedDiets = diets.filter((diet): diet is { id: string; name: string } => isRecord(diet) && typeof diet.id === "string" && typeof diet.name === "string");
		} catch { /* use the bounded generic error */ }
		throw new AccountDataClientError(affectedDiets?.length ? "Remove this item from the listed saved diets before permanent deletion." : "The private item could not be deleted. Try again.", affectedDiets);
	}
	if ((await readBoundedText(response, 0)) !== "") throw new AccountDataClientError("The private item could not be deleted. Try again.");
}

/** Injectable Account Export and private-item mutation operations. */
export interface AccountDataApi {
	loadExport: typeof loadAccountExport;
	deleteCustomItem: typeof deletePrivateCustomItem;
}

/** Account Export operations exposed to the Administration Panel. */
export const accountDataApi: AccountDataApi = { loadExport: loadAccountExport, deleteCustomItem: deletePrivateCustomItem };

function assertCustomItemSummary(value: unknown): asserts value is CustomItem {
	if (!isRecord(value) || !uuid(value.id) || typeof value.name !== "string" || value.name.trim() === "" || value.name.length > 200 || "ownerId" in value) throw new AccountDataClientError();
}

function assertSavedDiet(value: unknown): void {
	if (!isRecord(value) || !uuid(value.id) || typeof value.name !== "string" || value.name.trim() === "" || !Array.isArray(value.entries) || "userId" in value || "UserID" in value) throw new AccountDataClientError();
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
