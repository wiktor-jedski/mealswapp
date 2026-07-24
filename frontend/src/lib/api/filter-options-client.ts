import type { FilterOption, FilterOptionsEnvelope } from "./generated";

// Implements DESIGN-001 SearchView dynamic substitution filter inventory client.

const FILTER_OPTIONS_ENDPOINT = "/api/v1/search/filter-options?mode=substitution";
const MAX_RESPONSE_BYTES = 32 * 1024 * 1024;
const MAX_OPTIONS = 1000;
const MAX_EXCLUDES = 20;
const MAX_TEXT_LENGTH = 200;

/** Safe public failure for the recoverable substitution-filter inventory boundary. */
export class FilterOptionsClientError extends Error {
	constructor(readonly status: number) {
		super("Filter options are temporarily unavailable.");
		this.name = "FilterOptionsClientError";
	}
}

/** Loads the backend-owned substitution filter inventory without caching frontend policy. */
export async function fetchSubstitutionFilterOptions(signal?: AbortSignal): Promise<FilterOption[]> {
	let response: Response;
	try {
		response = await fetch(FILTER_OPTIONS_ENDPOINT, {
			method: "GET",
			credentials: "include",
			headers: { Accept: "application/json" },
			signal
		});
	} catch (error) {
		if (signal?.aborted) throw error;
		throw new FilterOptionsClientError(0);
	}

	if (response.status !== 200) throw new FilterOptionsClientError(response.status);

	let payload: unknown;
	try {
		payload = await readBoundedJson(response, signal);
	} catch (error) {
		if (signal?.aborted) throw signal.reason ?? error;
		if (error instanceof DOMException && error.name === "AbortError") throw error;
		throw new FilterOptionsClientError(response.status);
	}
	if (!isFilterOptionsEnvelope(payload)) throw new FilterOptionsClientError(response.status);
	return payload.data.options;
}

function isFilterOptionsEnvelope(value: unknown): value is FilterOptionsEnvelope {
	if (!isRecord(value) || !hasOnlyKeys(value, ["status", "requestId", "data"]) || value.status !== "ok" || typeof value.requestId !== "string" || !isRecord(value.data)) return false;
	return hasOnlyKeys(value.data, ["mode", "options"])
		&& value.data.mode === "substitution"
		&& Array.isArray(value.data.options)
		&& value.data.options.length <= MAX_OPTIONS
		&& value.data.options.every(isFilterOption);
}

function isFilterOption(value: unknown): value is FilterOption {
	return isRecord(value)
		&& hasOnlyKeys(value, ["filterId", "kind", "label", "labelKey", "includeAllowed", "excludeAllowed", "excludes"])
		&& isBoundedText(value.filterId)
		&& isSearchFilterKind(value.kind)
		&& isBoundedText(value.label)
		&& (value.labelKey === undefined || isBoundedText(value.labelKey))
		&& typeof value.includeAllowed === "boolean"
		&& typeof value.excludeAllowed === "boolean"
		&& Array.isArray(value.excludes)
		&& value.excludes.length <= MAX_EXCLUDES
		&& value.excludes.every(isFilterOptionReference);
}

function isFilterOptionReference(value: unknown): boolean {
	return isRecord(value)
		&& hasOnlyKeys(value, ["filterId", "kind"])
		&& isBoundedText(value.filterId)
		&& isSearchFilterKind(value.kind);
}

function isSearchFilterKind(value: unknown): value is FilterOption["kind"] {
	return value === "food_category" || value === "culinary_role" || value === "physical_state" || value === "allergen" || value === "dietary_preset";
}

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null && !Array.isArray(value);
}

function hasOnlyKeys(value: Record<string, unknown>, allowed: readonly string[]): boolean {
	return Object.keys(value).every((key) => allowed.includes(key));
}

function isBoundedText(value: unknown): value is string {
	if (typeof value !== "string") return false;
	const length = [...value].length;
	return length >= 1 && length <= MAX_TEXT_LENGTH;
}

async function readBoundedJson(response: Response, signal?: AbortSignal): Promise<unknown> {
	const contentLength = response.headers.get("content-length");
	if (contentLength !== null && /^\d+$/.test(contentLength) && Number(contentLength) > MAX_RESPONSE_BYTES) throw new RangeError("filter option response is too large");
	if (!response.body) return JSON.parse(await response.text()) as unknown;

	const reader = response.body.getReader();
	const chunks: Uint8Array[] = [];
	let length = 0;
	try {
		while (true) {
			if (signal?.aborted) throw signal.reason ?? new DOMException("Aborted", "AbortError");
			const { done, value } = await reader.read();
			if (done) break;
			length += value.byteLength;
			if (length > MAX_RESPONSE_BYTES) {
				await reader.cancel();
				throw new RangeError("filter option response is too large");
			}
			chunks.push(value);
		}
	} finally {
		reader.releaseLock();
	}

	const bytes = new Uint8Array(length);
	let offset = 0;
	for (const chunk of chunks) {
		bytes.set(chunk, offset);
		offset += chunk.byteLength;
	}
	return JSON.parse(new TextDecoder().decode(bytes)) as unknown;
}
