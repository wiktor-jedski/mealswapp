import { derived, type Readable } from "svelte/store";
import { keepPreviousData, type QueryFunctionContext } from "@tanstack/query-core";
import type { CreateQueryOptions } from "@tanstack/svelte-query";

import type {
	AppError,
	AutocompleteEnvelope,
	AutocompleteResponse,
	Envelope,
	FoodObject,
	FoodObjectEnvelope,
	SearchRequest,
	SearchResponse,
	SearchResponseEnvelope
} from "./generated";
import { buildSearchRequest, searchRequestKey, type SearchState } from "../stores/search";
import { LocalQueryCache } from "../cache/local-query-cache";

// Implements DESIGN-001 SearchView TanStack Query search/autocomplete client over generated envelopes.
// Implements DESIGN-017 ErrorMessageMapper safe AppError mapping for 400/422/429/503 envelopes.

/** Base path of the POST search endpoint served by ARCH-002 SearchController. */
const SEARCH_ENDPOINT = "/api/v1/search";

/** Base path of the GET autocomplete endpoint served by ARCH-002 SearchController. */
const AUTOCOMPLETE_ENDPOINT = "/api/v1/search/autocomplete";

/** Base path of the GET food-object detail endpoint served by ARCH-002 SearchController. */
const FOOD_OBJECT_ENDPOINT = "/api/v1/food-objects";

/** Search request budget after which the client aborts and surfaces a retryable timeout. */
export const SEARCH_TIMEOUT_MS = 10_000;

/** Local query cache freshness window used before issuing a network fetch. */
export const LOCAL_CACHE_STALE_MS = 5 * 60 * 1000;

/** Stable query-key namespace prefix so search keys never collide with other queries. */
const SEARCH_QUERY_NAMESPACE = "search" as const;

/** Stable query-key namespace prefix for autocomplete queries. */
const AUTOCOMPLETE_QUERY_NAMESPACE = "autocomplete" as const;

/** TanStack query key shape for search queries: a namespace plus the deterministic request key. */
export type SearchQueryKey = readonly [typeof SEARCH_QUERY_NAMESPACE, string];

/** TanStack query key shape for autocomplete queries: a namespace plus the raw query string. */
export type AutocompleteQueryKey = readonly [typeof AUTOCOMPLETE_QUERY_NAMESPACE, string];

/**
 * Error thrown by the search client when the API returns a non-2xx envelope or the
 * request times out. Carries a user-safe {@link AppError} so callers can render
 * classified messages without touching the raw network exception.
 *
 * @remarks Implements DESIGN-017 ErrorMessageMapper client error boundary contract.
 */
export class SearchClientError extends Error {
	readonly appError: AppError;
	readonly status: number;

	constructor(appError: AppError, status: number) {
		super(appError.message);
		this.name = "SearchClientError";
		this.appError = appError;
		this.status = status;
	}
}

/**
 * Maps a server envelope error and HTTP status to a user-safe {@link AppError}, preserving
 * `requestId` and `retryable` from the server while never leaking stack traces or URLs.
 *
 * @remarks Implements DESIGN-017 ErrorMessageMapper 400/422/429/503 status classification.
 */
export function mapAppError(
	envelopeError: AppError | undefined | null,
	status: number,
	fallbackMessage: string
): AppError {
	const category = categoryForStatus(status);
	const serverCategory = envelopeError?.category;
	const resolvedCategory = isCategory(serverCategory) ? serverCategory : category;
	const retryable = envelopeError?.retryable ?? defaultRetryableFor(resolvedCategory);
	const message = looksSafe(envelopeError?.message) ? (envelopeError as { message: string }).message : fallbackMessage;
	const code = envelopeError?.code ?? defaultCodeForStatus(status);

	const appError: AppError = {
		category: resolvedCategory,
		code,
		message,
		retryable
	};

	if (envelopeError?.requestId) {
		appError.requestId = envelopeError.requestId;
	}
	return appError;
}

/**
 * POSTs a {@link SearchRequest} to `/api/v1/search` with `credentials: "include"` and the
 * provided AbortSignal, then decodes the generated {@link SearchResponseEnvelope} into a
 * {@link SearchResponse}. Non-2xx envelopes throw {@link SearchClientError}.
 *
 * @remarks Implements DESIGN-001 SearchView credentialed POST search request and envelope decoding.
 */
export async function fetchSearch(request: SearchRequest, signal: AbortSignal): Promise<SearchResponse> {
	const response = await fetch(SEARCH_ENDPOINT, {
		method: "POST",
		credentials: "include",
		headers: {
			Accept: "application/json",
			"Content-Type": "application/json"
		},
		body: JSON.stringify(request),
		signal
	});
	return decodeSearchResponse(response);
}

/**
 * GETs `/api/v1/search/autocomplete` with the query parameter, `credentials: "include"`, and the
 * provided AbortSignal, then decodes the generated {@link AutocompleteEnvelope} into an
 * {@link AutocompleteResponse}. Non-2xx envelopes throw {@link SearchClientError}.
 *
 * @remarks Implements DESIGN-001 SearchView credentialed GET autocomplete request and envelope decoding.
 */
export async function fetchAutocomplete(query: string, signal: AbortSignal): Promise<AutocompleteResponse> {
	const params = new URLSearchParams();
	if (query.length > 0) {
		params.set("query", query);
	}
	const url = params.toString().length > 0 ? `${AUTOCOMPLETE_ENDPOINT}?${params.toString()}` : AUTOCOMPLETE_ENDPOINT;

	const response = await fetch(url, {
		method: "GET",
		credentials: "include",
		headers: {
			Accept: "application/json"
		},
		signal
	});
	return decodeAutocompleteResponse(response);
}

/**
 * GETs `/api/v1/food-objects/{id}` with `credentials: "include"` and the provided AbortSignal,
 * then decodes the generated {@link FoodObjectEnvelope} into a {@link FoodObject}. Non-2xx
 * envelopes throw {@link SearchClientError}.
 *
 * @remarks Implements DESIGN-001 SearchView selected Substitution Input hydration over generated envelopes.
 */
export async function fetchFoodObject(id: string, signal: AbortSignal): Promise<FoodObject> {
	const response = await fetch(`${FOOD_OBJECT_ENDPOINT}/${encodeURIComponent(id)}`, {
		method: "GET",
		credentials: "include",
		headers: {
			Accept: "application/json"
		},
		signal
	});
	return decodeFoodObjectResponse(response);
}

/**
 * Builds TanStack Query options for the search query backed by {@link searchRequestKey} as the
 * stable query key, the local query cache for hit/miss behavior, a 10-second timeout, and
 * `placeholderData: keepPreviousData` so previous-page results remain visible during page loads.
 *
 * @remarks Implements DESIGN-001 SearchView TanStack Query search query options (step 6).
 */
export function buildSearchQueryOptions(
	state: SearchState,
	localCache: LocalQueryCache,
	timeoutMs: number = SEARCH_TIMEOUT_MS
): CreateQueryOptions<SearchResponse, SearchClientError, SearchResponse, SearchQueryKey> {
	const requestKey = searchRequestKey(state);
	const request = buildSearchRequest(state);
	const queryKey: SearchQueryKey = [SEARCH_QUERY_NAMESPACE, requestKey];
	const enabled =
		state.mode === "substitution"
			? state.searchSubmitted && state.substitutionInputs.length > 0
			: state.query.trim().length > 0;

	return {
		queryKey,
		// Implements DESIGN-001 SearchView execution guard: Catalog/Daily require submitted text; Substitution requires the explicit two-step search action with at least one input.
		enabled,
		// Implements DESIGN-001 SearchView previous-page retention via TanStack keepPreviousData.
		placeholderData: keepPreviousData,
		staleTime: LOCAL_CACHE_STALE_MS,
		gcTime: LOCAL_CACHE_STALE_MS * 2,
		queryFn: (context) => runSearchQueryFn(request, requestKey, localCache, context, timeoutMs)
	};
}

/**
 * Derives a Svelte store of TanStack Query search options from a {@link SearchState} store so
 * components can pass `() => $options` to `createQuery` and stay reactive to state changes.
 *
 * @remarks Implements DESIGN-001 SearchView Svelte store to TanStack Query options bridge.
 */
export function createSearchQueryOptions(
	state: Readable<SearchState>,
	localCache: LocalQueryCache,
	timeoutMs: number = SEARCH_TIMEOUT_MS
): Readable<CreateQueryOptions<SearchResponse, SearchClientError, SearchResponse, SearchQueryKey>> {
	return derived(state, ($state) => buildSearchQueryOptions($state, localCache, timeoutMs));
}

/**
 * Builds TanStack Query options for the autocomplete query backed by the raw query string as the
 * stable query key and the same 10-second timeout budget as the search query.
 *
 * @remarks Implements DESIGN-001 SearchView autocomplete TanStack Query options.
 */
export function buildAutocompleteQueryOptions(
	query: string,
	timeoutMs: number = SEARCH_TIMEOUT_MS
): CreateQueryOptions<AutocompleteResponse, SearchClientError, AutocompleteResponse, AutocompleteQueryKey> {
	const queryKey: AutocompleteQueryKey = [AUTOCOMPLETE_QUERY_NAMESPACE, query];
	return {
		queryKey,
		placeholderData: keepPreviousData,
		staleTime: LOCAL_CACHE_STALE_MS,
		queryFn: (context) => runAutocompleteQueryFn(query, context, timeoutMs)
	};
}

/**
 * Creates a chained AbortSignal that aborts when the parent aborts or after `timeoutMs` milliseconds,
 * returning a cancel handle that clears the timer and removes the parent listener.
 *
 * @remarks Implements DESIGN-001 SearchView 10-second timeout budget with abort chaining.
 */
function createTimeoutSignal(parent: AbortSignal, timeoutMs: number): { signal: AbortSignal; cancel: () => void } {
	const controller = new AbortController();
	const onParentAbort = () => {
		controller.abort(parent.reason ?? new DOMException("Aborted", "AbortError"));
	};
	if (parent.aborted) {
		controller.abort(parent.reason);
	} else {
		parent.addEventListener("abort", onParentAbort, { once: true });
	}
	const timer = setTimeout(() => {
		controller.abort(new DOMException("Search timeout", "TimeoutError"));
	}, timeoutMs);
	return {
		signal: controller.signal,
		cancel: () => {
			clearTimeout(timer);
			parent.removeEventListener("abort", onParentAbort);
		}
	};
}

async function runSearchQueryFn(
	request: SearchRequest,
	requestKey: string,
	localCache: LocalQueryCache,
	context: QueryFunctionContext<SearchQueryKey>,
	timeoutMs: number
): Promise<SearchResponse> {
	// Implements DESIGN-001 SearchView local-cache read before fetch.
	if (localCache.has(requestKey) && !localCache.isStale(requestKey, LOCAL_CACHE_STALE_MS)) {
		const cached = localCache.get(requestKey);
		if (cached) {
			return cached.response;
		}
	}

	const handle = createTimeoutSignal(context.signal, timeoutMs);
	try {
		const response = await fetchSearch(request, handle.signal);
		// Implements DESIGN-001 SearchView local-cache write after successful fetch.
		localCache.set(requestKey, request, response);
		return response;
	} catch (error) {
		const cached = localCache.peek(requestKey);
		if (cached && isOfflineFetchFailure(error)) {
			return cached.response;
		}
		throw mapAbortError(error, handle.signal);
	} finally {
		handle.cancel();
	}
}

async function runAutocompleteQueryFn(
	query: string,
	context: QueryFunctionContext<AutocompleteQueryKey>,
	timeoutMs: number
): Promise<AutocompleteResponse> {
	const handle = createTimeoutSignal(context.signal, timeoutMs);
	try {
		return await fetchAutocomplete(query, handle.signal);
	} catch (error) {
		throw mapAbortError(error, handle.signal);
	} finally {
		handle.cancel();
	}
}

/**
 * Converts an `AbortError` caused by the 10-second timeout into a retryable
 * {@link SearchClientError}; other aborts and errors are rethrown unchanged so TanStack
 * can handle refetch cancellation and propagate HTTP-mapped errors.
 *
 * @remarks Implements DESIGN-001 SearchView timeout to AppError mapping.
 */
function mapAbortError(error: unknown, signal: AbortSignal): unknown {
	if (error instanceof DOMException && error.name === "AbortError") {
		const reason = signal.reason;
		if (reason instanceof DOMException && reason.name === "TimeoutError") {
			throw new SearchClientError(
				{
					category: "timeout",
					code: "search_timeout",
					message: "Search took too long. Please try again.",
					retryable: true
				},
				408
			);
		}
	}
	throw error;
}

/**
 * Identifies browser-offline fetch failures that can safely fall back to a stale
 * local-cache entry without hiding server-side API errors or user navigation aborts.
 *
 * @remarks Implements DESIGN-001 SearchView offline stale-cache fallback.
 */
function isOfflineFetchFailure(error: unknown): boolean {
	if (typeof navigator === "undefined" || navigator.onLine !== false) {
		return false;
	}
	if (error instanceof SearchClientError) {
		return false;
	}
	if (error instanceof DOMException && error.name === "AbortError") {
		return false;
	}
	return true;
}

/**
 * Decodes the generated search response envelope, mapping malformed JSON, non-2xx envelopes,
 * and malformed success envelopes into {@link SearchClientError}.
 *
 * @remarks Implements DESIGN-001 SearchView generated search envelope decoding and DESIGN-017 ErrorMessageMapper error wrapping.
 */
async function decodeSearchResponse(response: Response): Promise<SearchResponse> {
	const status = response.status;
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		throw new SearchClientError(
			mapAppError(undefined, status, fallbackMessageForCategory(categoryForStatus(status))),
			status
		);
	}

	if (!response.ok) {
		const envelope = readEnvelope(body);
		const fallback = fallbackMessageForCategory(categoryForStatus(status));
		const appError = mapAppError(envelope?.error ?? undefined, status, fallback);
		attachRequestId(appError, envelope?.requestId);
		throw new SearchClientError(appError, status);
	}

	const envelope = body as SearchResponseEnvelope | null;
	if (!envelope || typeof envelope !== "object" || envelope.data === undefined || envelope.data === null) {
		const appError: AppError = {
			category: "server",
			code: "malformed_envelope",
			message: fallbackMessageForCategory("server"),
			retryable: true
		};
		attachRequestId(appError, envelope?.requestId);
		throw new SearchClientError(appError, status);
	}
	return envelope.data;
}

/**
 * Decodes the generated autocomplete response envelope and applies the shared safe error mapping
 * used by the search client.
 *
 * @remarks Implements DESIGN-001 AutocompleteDropdown generated autocomplete envelope decoding and DESIGN-017 ErrorMessageMapper error wrapping.
 */
async function decodeAutocompleteResponse(response: Response): Promise<AutocompleteResponse> {
	const status = response.status;
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		throw new SearchClientError(
			mapAppError(undefined, status, fallbackMessageForCategory(categoryForStatus(status))),
			status
		);
	}

	if (!response.ok) {
		const envelope = readEnvelope(body);
		const fallback = fallbackMessageForCategory(categoryForStatus(status));
		const appError = mapAppError(envelope?.error ?? undefined, status, fallback);
		attachRequestId(appError, envelope?.requestId);
		throw new SearchClientError(appError, status);
	}

	const envelope = body as AutocompleteEnvelope | null;
	if (!envelope || typeof envelope !== "object" || envelope.data === undefined || envelope.data === null) {
		const appError: AppError = {
			category: "server",
			code: "malformed_envelope",
			message: fallbackMessageForCategory("server"),
			retryable: true
		};
		attachRequestId(appError, envelope?.requestId);
		throw new SearchClientError(appError, status);
	}
	return envelope.data;
}

/**
 * Decodes the generated FoodObject detail envelope used to hydrate selected Substitution Inputs.
 *
 * @remarks Implements DESIGN-001 SearchView selected-item hydration envelope decoding and DESIGN-017 ErrorMessageMapper error wrapping.
 */
async function decodeFoodObjectResponse(response: Response): Promise<FoodObject> {
	const status = response.status;
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		throw new SearchClientError(
			mapAppError(undefined, status, fallbackMessageForCategory(categoryForStatus(status))),
			status
		);
	}

	if (!response.ok) {
		const envelope = readEnvelope(body);
		const fallback = fallbackMessageForCategory(categoryForStatus(status));
		const appError = mapAppError(envelope?.error ?? undefined, status, fallback);
		attachRequestId(appError, envelope?.requestId);
		throw new SearchClientError(appError, status);
	}

	const envelope = body as FoodObjectEnvelope | null;
	if (!envelope || typeof envelope !== "object" || envelope.data === undefined || envelope.data === null) {
		const appError: AppError = {
			category: "server",
			code: "malformed_envelope",
			message: fallbackMessageForCategory("server"),
			retryable: true
		};
		attachRequestId(appError, envelope?.requestId);
		throw new SearchClientError(appError, status);
	}
	return envelope.data;
}

/**
 * Reads the shared generated API envelope shape from an unknown JSON body without trusting
 * application-specific payload fields.
 *
 * @remarks Implements DESIGN-017 ErrorMessageMapper generated envelope extraction.
 */
function readEnvelope(body: unknown): Envelope | null {
	if (typeof body !== "object" || body === null) {
		return null;
	}
	const candidate = body as { status?: unknown; requestId?: unknown; error?: unknown };
	if (typeof candidate.status !== "string" && typeof candidate.requestId !== "string" && candidate.error === undefined) {
		return null;
	}
	return candidate as Envelope;
}

/**
 * Copies an envelope-level request id onto the mapped {@link AppError} when the nested error
 * did not already carry one.
 *
 * @remarks Implements DESIGN-017 ErrorMessageMapper request-id preservation.
 */
function attachRequestId(appError: AppError, requestId: string | undefined): void {
	if (!appError.requestId && typeof requestId === "string" && requestId.length > 0) {
		appError.requestId = requestId;
	}
}

/**
 * Maps HTTP status codes used by the search endpoints into the generated AppError categories.
 *
 * @remarks Implements DESIGN-017 ErrorMessageMapper 400/404/422/429/503 category defaults.
 */
function categoryForStatus(status: number): AppError["category"] {
	switch (status) {
		case 400:
		case 404:
		case 422:
			return "validation";
		case 429:
			return "server";
		case 503:
			return "dependency";
		default:
			return "unknown";
	}
}

/**
 * Supplies retryability defaults when a server envelope is absent or omits the flag.
 *
 * @remarks Implements DESIGN-017 ErrorMessageMapper retryability defaults.
 */
function defaultRetryableFor(category: AppError["category"]): boolean {
	return category === "server" || category === "dependency" || category === "network" || category === "timeout";
}

/**
 * Supplies stable fallback error codes for status-derived {@link AppError} values.
 *
 * @remarks Implements DESIGN-017 ErrorMessageMapper status-to-code defaults.
 */
function defaultCodeForStatus(status: number): string {
	switch (status) {
		case 400:
			return "invalid_request";
		case 422:
			return "search_rejected";
		case 429:
			return "rate_limited";
		case 404:
			return "not_found";
		case 503:
			return "dependency_unavailable";
		default:
			return "unknown_error";
	}
}

/**
 * Supplies user-safe fallback copy for status-derived error categories.
 *
 * @remarks Implements DESIGN-017 ErrorMessageMapper safe fallback messages.
 */
function fallbackMessageForCategory(category: AppError["category"]): string {
	switch (category) {
		case "validation":
			return "Search request could not be processed. Please adjust your query and try again.";
		case "auth":
			return "Session expired. Please sign in and try again.";
		case "entitlement":
			return "Your plan does not include this search. Please upgrade to continue.";
		case "network":
			return "Network is unavailable. Please check your connection and try again.";
		case "timeout":
			return "Search took too long. Please try again.";
		case "server":
			return "Too many requests right now. Please wait a moment and try again.";
		case "dependency":
			return "Search is temporarily unavailable. Please try again shortly.";
		default:
			return "Something went wrong. Please try again.";
	}
}

/**
 * Runtime guard for generated AppError category values received from the server.
 *
 * @remarks Implements DESIGN-017 ErrorMessageMapper generated category validation.
 */
function isCategory(value: unknown): value is AppError["category"] {
	return (
		value === "validation" ||
		value === "auth" ||
		value === "entitlement" ||
		value === "network" ||
		value === "timeout" ||
		value === "server" ||
		value === "dependency" ||
		value === "unknown"
	);
}

/**
 * Predicate that rejects server-provided messages containing stack traces, URLs, or
 * newline-delimited trace fragments so the client never leaks infrastructure detail.
 *
 * @remarks Implements DESIGN-017 ErrorMessageMapper stack-trace and URL leak prevention.
 */
function looksSafe(message: string | undefined): message is string {
	if (typeof message !== "string" || message.length === 0) {
		return false;
	}
	if (message.includes("http://") || message.includes("https://")) {
		return false;
	}
	if (/\.(ts|js|go|rs|py):\d+/.test(message)) {
		return false;
	}
	if (message.includes("\n")) {
		return false;
	}
	if (/\b(stack|panic|goroutine|traceback)\b/i.test(message)) {
		return false;
	}
	return true;
}
