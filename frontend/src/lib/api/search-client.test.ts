import { afterEach, beforeEach, expect, test } from "bun:test";
import { QueryClient, QueryObserver, keepPreviousData } from "@tanstack/query-core";
import { get } from "svelte/store";
import { writable } from "svelte/store";

import type {
	AppError,
	AutocompleteEnvelope,
	AutocompleteResponse,
	SearchRequest,
	SearchResponse,
	SearchResponseEnvelope
} from "./generated";
import {
	addSubstitutionInput,
	createInitialSearchState,
	requestSubstitutionSearch,
	resetSearch,
	searchStore,
	setMode,
	setQuery,
	type SearchState
} from "../stores/search";
import { LocalQueryCache } from "../cache/local-query-cache";
import {
	LOCAL_CACHE_STALE_MS,
	SearchClientError,
	buildAutocompleteQueryOptions,
	buildSearchQueryOptions,
	createSearchQueryOptions,
	fetchAutocomplete,
	fetchFoodObject,
	fetchSearch,
	mapAppError
} from "./search-client";
import type { SearchQueryKey } from "./search-client";

// Implements DESIGN-001 SearchView generated search API client verification.
// Implements DESIGN-017 ErrorMessageMapper 400/422/429/503 mapping verification.

type MockResponseProvider = (init: RequestInit) => Response | Promise<Response>;

class FetchMock {
	calls: Array<{ url: string; init: RequestInit }> = [];
	private providers: MockResponseProvider[] = [];
	private index = 0;

	enqueueResponse(response: Response): void {
		this.providers.push(() => response);
	}

	enqueueProvider(provider: MockResponseProvider): void {
		this.providers.push(provider);
	}

	reset(): void {
		this.calls = [];
		this.providers = [];
		this.index = 0;
	}

	fetch = (input: string | URL | Request, init?: RequestInit): Promise<Response> => {
		const url = typeof input === "string" ? input : input.toString();
		const requestInit = init ?? {};
		this.calls.push({ url, init: requestInit });
		const provider = this.providers[this.index++];
		if (provider === undefined) {
			throw new Error(`FetchMock: no response queued for ${url}`);
		}
		return Promise.resolve(provider(requestInit));
	};
}

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { "Content-Type": "application/json" }
	});
}

function pendingUntilAbort(signal: AbortSignal): Promise<Response> {
	return new Promise<Response>((_resolve, reject) => {
		if (signal.aborted) {
			reject(new DOMException("Aborted", "AbortError"));
			return;
		}
		signal.addEventListener(
			"abort",
			() => reject(new DOMException("Aborted", "AbortError")),
			{ once: true }
		);
	});
}

function makeSearchResponse(seed: number, page: number): SearchResponse {
	return {
		items: [{ id: `food-${seed}`, name: `Item ${seed}`, physicalState: "solid" }],
		totalCount: 30,
		page,
		similarityScores: [0.9],
		similarityMetadata: [
			{
				itemId: `food-${seed}`,
				score: 0.9,
				tier: "excellent",
				imageUrl: `https://example.com/${seed}.png`,
				matchingQuantity: 100
			}
		],
		warnings: []
	};
}

function makeSearchEnvelope(seed: number, page: number, requestId = "req-1"): SearchResponseEnvelope {
	return {
		status: "ok",
		requestId,
		data: makeSearchResponse(seed, page)
	};
}

function makeAutocompleteResponse(): AutocompleteResponse {
	return {
		items: [
			{ itemId: "food-1", label: "Apple", exactMatch: true, levenshteinDistance: 0, length: 5, rank: 1 }
		]
	};
}

function makeAutocompleteEnvelope(): AutocompleteEnvelope {
	return { status: "ok", requestId: "req-auto-1", data: makeAutocompleteResponse() };
}

function makeFoodObjectEnvelope() {
	return {
		status: "ok",
		requestId: "req-food-1",
		data: {
			id: "food-1",
			name: "Apple",
			physicalState: "solid",
			imageUrl: null,
			classifications: [{ id: "cat-1", name: "Fruit", kind: "food_category" }],
			primaryFoodCategory: { id: "cat-1", name: "Fruit", kind: "food_category" },
			macros: { protein: 0.5, carbohydrates: 14, fat: 0.3 },
			macroBasis: "100g",
			calories: 60.7
		}
	} as const;
}

const originalFetch = globalThis.fetch;
const originalNavigator = globalThis.navigator;
const fetchMock = new FetchMock();

beforeEach(() => {
	fetchMock.reset();
	globalThis.fetch = fetchMock.fetch as typeof fetch;
});

afterEach(() => {
	globalThis.fetch = originalFetch;
	if (originalNavigator === undefined) {
		delete (globalThis as { navigator?: unknown }).navigator;
	} else {
		Object.defineProperty(globalThis, "navigator", {
			configurable: true,
			value: originalNavigator
		});
	}
	resetSearch();
});

function catalogState(query: string, page = 1): SearchState {
	const state = createInitialSearchState();
	state.query = query;
	state.page = page;
	return state;
}

function lastCall(): { url: string; init: RequestInit } {
	const call = fetchMock.calls[fetchMock.calls.length - 1];
	if (!call) {
		throw new Error("FetchMock: no fetch was recorded");
	}
	return call;
}

function asJson(init: RequestInit): Record<string, unknown> {
	if (init.body === undefined || init.body === null) {
		throw new Error("FetchMock: request body missing");
	}
	return JSON.parse(init.body as string) as Record<string, unknown>;
}

function setNavigatorOnline(onLine: boolean): void {
	Object.defineProperty(globalThis, "navigator", {
		configurable: true,
		value: { onLine }
	});
}

async function tick(): Promise<void> {
	await new Promise<void>((resolve) => setTimeout(resolve, 0));
}

/**
 * Invokes the `queryFn` of a TanStack Query options object with a synthesized context.
 * Accepts a minimal structural shape so the `SkipToken | undefined` union and query-key
 * contravariance do not block calls to the function the builder always assigns.
 */
function invokeQueryFn<TData>(options: { queryFn?: unknown }, signal: AbortSignal): Promise<TData> {
	const fn = options.queryFn as (ctx: { signal: AbortSignal }) => Promise<TData>;
	return fn({ signal });
}

async function waitForResult(
	observer: QueryObserver<SearchResponse, SearchClientError, SearchResponse, SearchResponse, SearchQueryKey>,
	predicate: (result: { data: SearchResponse | undefined; isPlaceholderData: boolean; isFetching: boolean }) => boolean,
	timeoutMs = 1000
): Promise<void> {
	const start = Date.now();
	while (Date.now() - start < timeoutMs) {
		const result = observer.getCurrentResult();
		if (predicate({ data: result.data, isPlaceholderData: result.isPlaceholderData, isFetching: result.isFetching })) {
			return;
		}
		await tick();
	}
	throw new Error("waitForResult: predicate never satisfied");
}

// Implements DESIGN-001 SearchView credentialed POST search request verification.
test("fetchSearch POSTs to /api/v1/search with credentials and JSON body matching SearchRequest", async () => {
	const request: SearchRequest = { query: "apple", mode: "catalog", page: 1 };
	fetchMock.enqueueResponse(jsonResponse(200, makeSearchEnvelope(1, 1)));

	const result = await fetchSearch(request, new AbortController().signal);

	expect(result).toEqual(makeSearchResponse(1, 1));
	const call = lastCall();
	expect(call.url).toBe("/api/v1/search");
	expect(call.init.method).toBe("POST");
	expect(call.init.credentials).toBe("include");
	expect(asJson(call.init)).toEqual({ query: "apple", mode: "catalog", page: 1 });
	expect((call.init.headers as Record<string, string>)["Content-Type"]).toBe("application/json");
});

// Implements DESIGN-001 SearchView generated envelope decoding verification.
test("fetchSearch decodes SearchResponseEnvelope.data into SearchResponse", async () => {
	fetchMock.enqueueResponse(jsonResponse(200, makeSearchEnvelope(7, 2, "req-decode")));
	const result = await fetchSearch({ query: "kale", mode: "catalog", page: 2 }, new AbortController().signal);
	expect(result.page).toBe(2);
	expect(result.items[0]?.id).toBe("food-7");
});

// Implements DESIGN-017 ErrorMessageMapper 400 validation mapping verification.
test("fetchSearch maps 400 to validation SearchClientError and preserves requestId and retryable", async () => {
	const envelope = {
		status: "error",
		requestId: "req-400",
		error: {
			category: "validation",
			code: "query_too_short",
			message: "Query must be at least 2 characters.",
			retryable: false,
			requestId: "req-400"
		}
	};
	fetchMock.enqueueResponse(jsonResponse(400, envelope));
	fetchMock.enqueueResponse(jsonResponse(400, envelope));

	await expect(
		fetchSearch({ query: "a", mode: "catalog", page: 1 }, new AbortController().signal)
	).rejects.toBeInstanceOf(SearchClientError);

	try {
		await fetchSearch({ query: "a", mode: "catalog", page: 1 }, new AbortController().signal);
	} catch (error) {
		const clientError = error as SearchClientError;
		expect(clientError.status).toBe(400);
		expect(clientError.appError.category).toBe("validation");
		expect(clientError.appError.retryable).toBe(false);
		expect(clientError.appError.requestId).toBe("req-400");
		expect(clientError.appError.code).toBe("query_too_short");
	}
});

// Implements DESIGN-017 ErrorMessageMapper 422 search rejection mapping verification.
test("fetchSearch maps 422 to validation category with search_rejected default code", async () => {
	const envelope = {
		status: "error",
		requestId: "req-422",
		error: { category: "validation", code: "filter_conflict", message: "Filters conflict.", retryable: false }
	};
	fetchMock.enqueueResponse(jsonResponse(422, envelope));

	try {
		await fetchSearch({ query: "x", mode: "catalog", page: 1 }, new AbortController().signal);
	} catch (error) {
		const clientError = error as SearchClientError;
		expect(clientError.status).toBe(422);
		expect(clientError.appError.category).toBe("validation");
		expect(clientError.appError.code).toBe("filter_conflict");
		expect(clientError.appError.requestId).toBe("req-422");
	}
});

// Implements DESIGN-017 ErrorMessageMapper 429 rate-limit mapping verification.
test("fetchSearch maps 429 to retryable server category and preserves requestId", async () => {
	const envelope = {
		status: "error",
		requestId: "req-429",
		error: { category: "server", code: "rate_limited", message: "Slow down.", retryable: true }
	};
	fetchMock.enqueueResponse(jsonResponse(429, envelope));

	try {
		await fetchSearch({ query: "x", mode: "catalog", page: 1 }, new AbortController().signal);
	} catch (error) {
		const clientError = error as SearchClientError;
		expect(clientError.status).toBe(429);
		expect(clientError.appError.category).toBe("server");
		expect(clientError.appError.retryable).toBe(true);
		expect(clientError.appError.requestId).toBe("req-429");
	}
});

// Implements DESIGN-017 ErrorMessageMapper 503 dependency mapping verification.
test("fetchSearch maps 503 to retryable dependency category", async () => {
	const envelope = {
		status: "error",
		requestId: "req-503",
		error: { category: "dependency", code: "search_index_down", message: "Search index unavailable.", retryable: true }
	};
	fetchMock.enqueueResponse(jsonResponse(503, envelope));

	try {
		await fetchSearch({ query: "x", mode: "catalog", page: 1 }, new AbortController().signal);
	} catch (error) {
		const clientError = error as SearchClientError;
		expect(clientError.status).toBe(503);
		expect(clientError.appError.category).toBe("dependency");
		expect(clientError.appError.retryable).toBe(true);
		expect(clientError.appError.requestId).toBe("req-503");
	}
});

// Implements DESIGN-017 ErrorMessageMapper stack-trace and URL leak prevention verification.
test("mapAppError falls back to safe message when server message leaks stack or URL", () => {
	const fallback = "Safe fallback.";
	const unsafeStack: AppError = {
		category: "server",
		code: "boom",
		message: "runtime error: at /src/search.go:42\nmain.run()",
		retryable: true
	};
	const mapped = mapAppError(unsafeStack, 500, fallback);
	expect(mapped.message).toBe(fallback);
	expect(mapped.category).toBe("server");
	expect(mapped.retryable).toBe(true);

	const unsafeUrl: AppError = {
		category: "validation",
		code: "bad",
		message: "Failed to reach https://internal.svc/search",
		retryable: false
	};
	const mappedUrl = mapAppError(unsafeUrl, 400, fallback);
	expect(mappedUrl.message).toBe(fallback);
});

// Implements DESIGN-017 ErrorMessageMapper default category and retryability verification.
test("mapAppError derives category, code, and retryability from status when envelope error is missing", () => {
	const fallback = "Fallback.";

	expect(mapAppError(undefined, 400, fallback).category).toBe("validation");
	expect(mapAppError(undefined, 400, fallback).retryable).toBe(false);
	expect(mapAppError(undefined, 400, fallback).code).toBe("invalid_request");

	expect(mapAppError(undefined, 422, fallback).category).toBe("validation");
	expect(mapAppError(undefined, 422, fallback).code).toBe("search_rejected");

	expect(mapAppError(undefined, 404, fallback).category).toBe("validation");
	expect(mapAppError(undefined, 404, fallback).code).toBe("not_found");

	expect(mapAppError(undefined, 429, fallback).category).toBe("server");
	expect(mapAppError(undefined, 429, fallback).retryable).toBe(true);
	expect(mapAppError(undefined, 429, fallback).code).toBe("rate_limited");

	expect(mapAppError(undefined, 503, fallback).category).toBe("dependency");
	expect(mapAppError(undefined, 503, fallback).retryable).toBe(true);
	expect(mapAppError(undefined, 503, fallback).code).toBe("dependency_unavailable");
});

// Implements DESIGN-001 SearchView credentialed GET autocomplete request verification.
test("fetchAutocomplete GETs /api/v1/search/autocomplete with query param and credentials", async () => {
	fetchMock.enqueueResponse(jsonResponse(200, makeAutocompleteEnvelope()));

	const result = await fetchAutocomplete("app", new AbortController().signal);

	expect(result).toEqual(makeAutocompleteResponse());
	const call = lastCall();
	expect(call.url).toBe("/api/v1/search/autocomplete?query=app");
	expect(call.init.method).toBe("GET");
	expect(call.init.credentials).toBe("include");
	expect(call.init.body).toBeUndefined();
});

// Implements DESIGN-001 SearchView autocomplete envelope decoding verification.
test("fetchAutocomplete decodes AutocompleteEnvelope.data into AutocompleteResponse", async () => {
	fetchMock.enqueueResponse(jsonResponse(200, makeAutocompleteEnvelope()));
	const result = await fetchAutocomplete("apple", new AbortController().signal);
	expect(result.items[0]?.label).toBe("Apple");
});

// Implements DESIGN-001 SearchView autocomplete error mapping verification.
test("fetchAutocomplete maps 429 to retryable server category", async () => {
	const envelope = {
		status: "error",
		requestId: "req-auto-429",
		error: { category: "server", code: "rate_limited", message: "Slow down.", retryable: true }
	};
	fetchMock.enqueueResponse(jsonResponse(429, envelope));

	try {
		await fetchAutocomplete("app", new AbortController().signal);
	} catch (error) {
		const clientError = error as SearchClientError;
		expect(clientError.status).toBe(429);
		expect(clientError.appError.category).toBe("server");
		expect(clientError.appError.retryable).toBe(true);
		expect(clientError.appError.requestId).toBe("req-auto-429");
	}
});

// Implements DESIGN-001 SearchView selected Substitution Input hydration request verification.
test("fetchFoodObject GETs /api/v1/food-objects/{id} with credentials", async () => {
	fetchMock.enqueueResponse(jsonResponse(200, makeFoodObjectEnvelope()));

	const result = await fetchFoodObject("food 1", new AbortController().signal);

	expect(result.name).toBe("Apple");
	expect(result.macros.carbohydrates).toBe(14);
	const call = lastCall();
	expect(call.url).toBe("/api/v1/food-objects/food%201");
	expect(call.init.method).toBe("GET");
	expect(call.init.credentials).toBe("include");
	expect(call.init.body).toBeUndefined();
});

// Implements DESIGN-001 SearchView selected Substitution Input hydration error mapping verification.
test("fetchFoodObject maps 404 to not found SearchClientError", async () => {
	fetchMock.enqueueResponse(jsonResponse(404, {
		status: "error",
		requestId: "req-food-404",
		error: { category: "validation", code: "not_found", message: "resource not found", retryable: false }
	}));

	try {
		await fetchFoodObject("missing", new AbortController().signal);
	} catch (error) {
		const clientError = error as SearchClientError;
		expect(clientError.status).toBe(404);
		expect(clientError.appError.code).toBe("not_found");
		expect(clientError.appError.requestId).toBe("req-food-404");
	}
});

// Implements DESIGN-001 SearchView stable query-key derivation verification (step 6).
test("buildSearchQueryOptions uses [search, searchRequestKey] as stable query key", () => {
	const state = catalogState("apple", 1);
	const options = buildSearchQueryOptions(state, new LocalQueryCache({ storage: null }));
	expect(options.queryKey[0]).toBe("search");
	expect(options.queryKey[1]).toBe(JSON.stringify({
		mode: "catalog",
		query: "apple",
		filters: [],
		page: 1,
		inputs: [],
		dailyDietId: ""
	}));
});

// Implements DESIGN-001 SearchView no duplicate request for equivalent query keys verification.
test("buildSearchQueryOptions produces identical query keys for equivalent search states", () => {
	const stateA = catalogState("apple", 1);
	const stateB = catalogState("apple", 1);
	const optionsA = buildSearchQueryOptions(stateA, new LocalQueryCache({ storage: null }));
	const optionsB = buildSearchQueryOptions(stateB, new LocalQueryCache({ storage: null }));
	expect(optionsA.queryKey).toEqual(optionsB.queryKey);
});

// Implements DESIGN-001 SearchView submitted-search loading isolation verification.
test("buildSearchQueryOptions does not keep previous result pages as placeholder data", () => {
	const options = buildSearchQueryOptions(catalogState("apple", 1), new LocalQueryCache({ storage: null }));
	expect(options.placeholderData).toBeUndefined();
});

// Implements DESIGN-001 SearchView local-cache hit bypasses fetch verification.
test("queryFn returns cached response on local cache hit without fetching", async () => {
	const state = catalogState("apple", 1);
	const localCache = new LocalQueryCache({ storage: null });
	const request: SearchRequest = { query: "apple", mode: "catalog", page: 1 };
	const cached = makeSearchResponse(5, 1);
	const requestKey = buildSearchQueryOptions(state, localCache).queryKey[1];
	localCache.set(requestKey, request, cached);

	const options = buildSearchQueryOptions(state, localCache);
	const result = await invokeQueryFn(options, new AbortController().signal);

	expect(result).toEqual(cached);
	expect(fetchMock.calls.length).toBe(0);
});

// Implements DESIGN-001 SearchView local-cache miss triggers fetch and writes to cache verification.
test("queryFn fetches on cache miss and writes the decoded response to local cache", async () => {
	const state = catalogState("apple", 1);
	const localCache = new LocalQueryCache({ storage: null });
	fetchMock.enqueueResponse(jsonResponse(200, makeSearchEnvelope(9, 1, "req-miss")));

	const options = buildSearchQueryOptions(state, localCache);
	const result = await invokeQueryFn(options, new AbortController().signal);

	expect(result).toEqual(makeSearchResponse(9, 1));
	expect(fetchMock.calls.length).toBe(1);
	const requestKey = options.queryKey[1];
	expect(localCache.get(requestKey)?.response).toEqual(makeSearchResponse(9, 1));
});

// Implements DESIGN-001 SearchView stale cache entry triggers fetch verification.
test("queryFn fetches when local cache entry is stale", async () => {
	const now = { value: 1_000_000 };
	const localCache = new LocalQueryCache({ storage: null, now: () => now.value });
	const state = catalogState("apple", 1);
	const options = buildSearchQueryOptions(state, localCache);
	const requestKey = options.queryKey[1];
	localCache.set(requestKey, { query: "apple", mode: "catalog", page: 1 }, makeSearchResponse(1, 1));

	now.value += LOCAL_CACHE_STALE_MS + 1;
	fetchMock.enqueueResponse(jsonResponse(200, makeSearchEnvelope(2, 1, "req-refresh")));

	const result = await invokeQueryFn(options, new AbortController().signal);
	expect(result).toEqual(makeSearchResponse(2, 1));
	expect(fetchMock.calls.length).toBe(1);
});

// Implements DESIGN-001 SearchView offline stale-cache fallback verification.
test("queryFn returns stale cached response when browser is offline and fetch fails", async () => {
	const now = { value: 1_000_000 };
	const localCache = new LocalQueryCache({ storage: null, now: () => now.value });
	const state = catalogState("apple", 1);
	const options = buildSearchQueryOptions(state, localCache);
	const requestKey = options.queryKey[1];
	const staleCached = makeSearchResponse(1, 1);
	localCache.set(requestKey, { query: "apple", mode: "catalog", page: 1 }, staleCached);

	now.value += LOCAL_CACHE_STALE_MS + 1;
	setNavigatorOnline(false);
	fetchMock.enqueueProvider(() => Promise.reject(new TypeError("Failed to fetch")));

	const result = await invokeQueryFn(options, new AbortController().signal);

	expect(result).toEqual(staleCached);
	expect(fetchMock.calls.length).toBe(1);
	expect(localCache.isStale(requestKey, LOCAL_CACHE_STALE_MS)).toBe(true);
});

// Implements DESIGN-001 SearchView 10-second timeout cancellation verification.
test("queryFn aborts fetch after timeout and throws retryable timeout SearchClientError", async () => {
	const state = catalogState("apple", 1);
	const localCache = new LocalQueryCache({ storage: null });
	fetchMock.enqueueProvider((init) => pendingUntilAbort(init.signal ?? new AbortController().signal));

	const options = buildSearchQueryOptions(state, localCache, 50);
	await expect(
		invokeQueryFn(options, new AbortController().signal)
	).rejects.toBeInstanceOf(SearchClientError);

	try {
		fetchMock.enqueueProvider((init) => pendingUntilAbort(init.signal ?? new AbortController().signal));
		await invokeQueryFn(buildSearchQueryOptions(state, localCache, 50), new AbortController().signal);
	} catch (error) {
		const clientError = error as SearchClientError;
		expect(clientError.appError.category).toBe("timeout");
		expect(clientError.appError.retryable).toBe(true);
		expect(clientError.appError.code).toBe("search_timeout");
	}
});

// Implements DESIGN-001 SearchView queryFn propagates parent abort without timeout mapping verification.
test("queryFn rethrows parent abort as AbortError without mapping to timeout", async () => {
	const state = catalogState("apple", 1);
	const localCache = new LocalQueryCache({ storage: null });
	fetchMock.enqueueProvider((init) => pendingUntilAbort(init.signal ?? new AbortController().signal));

	const parent = new AbortController();
	const options = buildSearchQueryOptions(state, localCache, 5000);
	const pending = invokeQueryFn(options, parent.signal);
	parent.abort(new DOMException("User navigated", "AbortError"));

	await expect(pending).rejects.toMatchObject({ name: "AbortError" });
});

// Implements DESIGN-001 SearchView queryFn maps HTTP errors to SearchClientError verification.
test("queryFn surfaces 429 envelope as SearchClientError with retryable server category", async () => {
	const state = catalogState("apple", 1);
	const localCache = new LocalQueryCache({ storage: null });
	const envelope = {
		status: "error",
		requestId: "req-qfn-429",
		error: { category: "server", code: "rate_limited", message: "Slow down.", retryable: true }
	};
	fetchMock.enqueueResponse(jsonResponse(429, envelope));

	const options = buildSearchQueryOptions(state, localCache);
	try {
		await invokeQueryFn(options, new AbortController().signal);
	} catch (error) {
		const clientError = error as SearchClientError;
		expect(clientError.appError.category).toBe("server");
		expect(clientError.appError.retryable).toBe(true);
		expect(clientError.appError.requestId).toBe("req-qfn-429");
	}
});

// Implements DESIGN-001 SearchView createSearchQueryOptions derived store verification.
test("createSearchQueryOptions derives options reactively from a SearchState store", async () => {
	const store = writable<SearchState>(catalogState("apple", 1));
	const localCache = new LocalQueryCache({ storage: null });
	const optionsStore = createSearchQueryOptions(store, localCache);

	let options = get(optionsStore);
	expect(options.queryKey[1]).toContain('"query":"apple"');

	store.set(catalogState("banana", 2));
	options = get(optionsStore);
	expect(options.queryKey[1]).toContain('"query":"banana"');
	expect(options.queryKey[1]).toContain('"page":2');
});

// Implements DESIGN-001 SearchView QueryClient cache hit avoids duplicate fetch verification.
test("QueryClient fetchQuery with equivalent query keys performs fetch only once", async () => {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
	const localCache = new LocalQueryCache({ storage: null });
	fetchMock.enqueueResponse(jsonResponse(200, makeSearchEnvelope(1, 1, "req-once")));

	const first = await queryClient.fetchQuery(buildSearchQueryOptions(catalogState("apple", 1), localCache));
	expect(first).toEqual(makeSearchResponse(1, 1));
	expect(fetchMock.calls.length).toBe(1);

	const second = await queryClient.fetchQuery(buildSearchQueryOptions(catalogState("apple", 1), localCache));
	expect(second).toEqual(makeSearchResponse(1, 1));
	expect(fetchMock.calls.length).toBe(1);
});

// Implements DESIGN-001 SearchView QueryClient local cache hit bypasses fetch verification.
test("QueryClient fetchQuery bypasses network when local cache holds a fresh entry", async () => {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
	const localCache = new LocalQueryCache({ storage: null });
	const state = catalogState("apple", 1);
	const requestKey = buildSearchQueryOptions(state, localCache).queryKey[1];
	localCache.set(requestKey, { query: "apple", mode: "catalog", page: 1 }, makeSearchResponse(8, 1));

	const result = await queryClient.fetchQuery(buildSearchQueryOptions(state, localCache));
	expect(result).toEqual(makeSearchResponse(8, 1));
	expect(fetchMock.calls.length).toBe(0);
});

// Implements DESIGN-001 SearchView submitted-search loading isolation verification.
test("QueryObserver clears previous page data while next page loads", async () => {
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
	const localCache = new LocalQueryCache({ storage: null });

	fetchMock.enqueueResponse(jsonResponse(200, makeSearchEnvelope(1, 1, "req-page-1")));
	const observer = new QueryObserver<SearchResponse, SearchClientError, SearchResponse, SearchResponse, SearchQueryKey>(
		queryClient,
		buildSearchQueryOptions(catalogState("apple", 1), localCache)
	);
	const unsubscribe = observer.subscribe(() => {});

	await waitForResult(observer, (r) => r.data?.page === 1 && !r.isPlaceholderData);
	expect(observer.getCurrentResult().data).toEqual(makeSearchResponse(1, 1));

	let resolvePage2!: (value: Response) => void;
	fetchMock.enqueueProvider(() => new Promise<Response>((resolve) => {
		resolvePage2 = resolve;
	}));

	observer.setOptions(buildSearchQueryOptions(catalogState("apple", 2), localCache));
	await tick();

	const during = observer.getCurrentResult();
	expect(during.isFetching).toBe(true);
	expect(during.isPlaceholderData).toBe(false);
	expect(during.data).toBeUndefined();

	resolvePage2(jsonResponse(200, makeSearchEnvelope(2, 2, "req-page-2")));
	await waitForResult(observer, (r) => r.data?.page === 2 && !r.isPlaceholderData);
	expect(observer.getCurrentResult().data).toEqual(makeSearchResponse(2, 2));

	unsubscribe();
});

// Implements DESIGN-001 SearchView autocomplete query options verification.
test("buildAutocompleteQueryOptions uses [autocomplete, query] key and decodes envelope", async () => {
	fetchMock.enqueueResponse(jsonResponse(200, makeAutocompleteEnvelope()));
	const options = buildAutocompleteQueryOptions("app", 5000);
	expect(options.queryKey[0]).toBe("autocomplete");
	expect(options.queryKey[1]).toBe("app");
	expect(options.placeholderData).toBe(keepPreviousData);

	const result = await invokeQueryFn(options, new AbortController().signal);
	expect(result).toEqual(makeAutocompleteResponse());
	const call = lastCall();
	expect(call.url).toBe("/api/v1/search/autocomplete?query=app");
	expect(call.init.credentials).toBe("include");
});

// Implements DESIGN-001 SearchView autocomplete timeout cancellation verification.
test("buildAutocompleteQueryOptions aborts after timeout and throws timeout SearchClientError", async () => {
	fetchMock.enqueueProvider((init) => pendingUntilAbort(init.signal ?? new AbortController().signal));
	const options = buildAutocompleteQueryOptions("app", 50);
	try {
		await invokeQueryFn(options, new AbortController().signal);
	} catch (error) {
		const clientError = error as SearchClientError;
		expect(clientError.appError.category).toBe("timeout");
		expect(clientError.appError.retryable).toBe(true);
	}
});

// Implements DESIGN-001 SearchView searchStore integration stable key verification.
test("equivalent searchStore states share a query key via searchRequestKey", () => {
	setQuery("apple");
	const keyA = buildSearchQueryOptions(get(searchStore), new LocalQueryCache({ storage: null })).queryKey;
	resetSearch();
	setQuery("apple");
	const keyB = buildSearchQueryOptions(get(searchStore), new LocalQueryCache({ storage: null })).queryKey;
	expect(keyA).toEqual(keyB);
});

// Implements DESIGN-001 SearchView distinct pages produce distinct query keys verification.
test("buildSearchQueryOptions produces distinct query keys for distinct pages", () => {
	const localCache = new LocalQueryCache({ storage: null });
	const key1 = buildSearchQueryOptions(catalogState("apple", 1), localCache).queryKey;
	const key2 = buildSearchQueryOptions(catalogState("apple", 2), localCache).queryKey;
	expect(key1).not.toEqual(key2);
});

// Implements DESIGN-001 SearchView execution guard so the shell does not fire premature requests.
test("buildSearchQueryOptions enables Catalog text searches and explicit Substitution searches only", () => {
	const localCache = new LocalQueryCache({ storage: null });
	expect(buildSearchQueryOptions(catalogState("", 1), localCache).enabled).toBe(false);
	expect(buildSearchQueryOptions(catalogState("   ", 1), localCache).enabled).toBe(false);
	expect(buildSearchQueryOptions(catalogState("apple", 1), localCache).enabled).toBe(true);

	setMode("substitution");
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" }, "Apple");
	expect(buildSearchQueryOptions(get(searchStore), localCache).enabled).toBe(false);
	requestSubstitutionSearch();
	expect(buildSearchQueryOptions(get(searchStore), localCache).enabled).toBe(true);
});
