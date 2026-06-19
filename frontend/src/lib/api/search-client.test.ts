import { describe, expect, test } from "bun:test";
import { QueryClient, keepPreviousData } from "@tanstack/svelte-query";
import type { AppError, SearchRequest, SearchResponse } from "./generated";
import { SearchLRUCache } from "../cache/search-lru";
import { AppClientError, SearchAPIClient, type FetchLike } from "./search-client";

const request: SearchRequest = { query: "apple", mode: "catalog", filters: [], page: 1 };
const searchResponse: SearchResponse = { items: [], totalCount: 0, page: 1, similarityScores: [], similarityMetadata: [], warnings: [] };
const envelope = (data: unknown) => ({ status: "ok", requestId: "req-1", data });
const jsonResponse = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { "content-type": "application/json" } });

// Implements DESIGN-001 SearchView generated-contract API client verification.
describe("SearchAPIClient", () => {
  test("sends credentialed generated requests and decodes search responses", async () => {
    let url = "";
    let init: RequestInit | undefined;
    const fetcher: FetchLike = async (input, options) => { url = String(input); init = options; return jsonResponse(envelope(searchResponse)); };
    const client = new SearchAPIClient({ baseURL: "https://example.test/", fetch: fetcher, cache: new SearchLRUCache({ storage: null }) });
    expect(await client.search(request)).toEqual(searchResponse);
    expect(url).toBe("https://example.test/api/v1/search");
    expect(init?.method).toBe("POST");
    expect(init?.credentials).toBe("include");
    expect(JSON.parse(String(init?.body))).toEqual(request);
  });

  test("encodes autocomplete queries and decodes generated envelopes", async () => {
    let url = "";
    const client = new SearchAPIClient({ fetch: async (input) => { url = String(input); return jsonResponse(envelope({ items: [] })); }, cache: new SearchLRUCache({ storage: null }) });
    expect(await client.autocomplete("green apple")).toEqual({ items: [] });
    expect(url).toBe("/api/v1/search/autocomplete?query=green%20apple");
  });

  test("builds stable autocomplete query options and retry policy", async () => {
    let calls = 0;
    const client = new SearchAPIClient({ fetch: async () => { calls += 1; return jsonResponse(envelope({ items: [] })); }, cache: new SearchLRUCache({ storage: null }) });
    const options = client.autocompleteQueryOptions(" Apple ");
    expect(options.queryKey).toEqual(["autocomplete", "apple"]);
    expect(options.enabled).toBe(true);
    expect(await options.queryFn!({} as never)).toEqual({ items: [] });
    expect(options.retry?.(0, new AppClientError({ category: "network", code: "offline", message: "offline", retryable: true }))).toBe(true);
    expect(client.autocompleteQueryOptions(" ").enabled).toBe(false);
    expect(calls).toBe(1);
  });

  test("aborts at the configured timeout and maps a retryable timeout error", async () => {
    const fetcher: FetchLike = async (_input, init) => await new Promise((_resolve, reject) => init?.signal?.addEventListener("abort", () => reject(new DOMException("aborted", "AbortError"))));
    const client = new SearchAPIClient({ fetch: fetcher, timeoutMs: 5, cache: new SearchLRUCache({ storage: null }) });
    await expect(client.search(request)).rejects.toMatchObject({ detail: { category: "timeout", code: "request_timeout", retryable: true } });
  });

  test("maps 400, 422, 429, and 503 errors with request IDs and retryability", async () => {
    for (const [status, category, retryable] of [[400, "validation", false], [422, "validation", false], [429, "entitlement", true], [503, "dependency", true]] as const) {
      const detail: AppError = { category, code: `status_${status}`, message: "safe", retryable, requestId: `req-${status}` };
      const client = new SearchAPIClient({ fetch: async () => jsonResponse({ status: "error", requestId: `req-${status}`, error: detail }, status), cache: new SearchLRUCache({ storage: null }) });
      try { await client.search(request); throw new Error("expected rejection"); } catch (error) {
        expect(error).toBeInstanceOf(AppClientError);
        expect((error as AppClientError).detail).toEqual(detail);
      }
    }
  });

  test("preserves a generated structured search rejection", async () => {
    const rejection = { code: "daily_diet_phase_07_required", message: "Daily Diet Alternative requires Phase 07 data", field: "dailyDietId" };
    const detail: AppError = { category: "validation", code: "daily_diet_unavailable", message: "Search rejected", retryable: false };
    const client = new SearchAPIClient({ fetch: async () => jsonResponse({ status: "error", requestId: "daily", data: { rejection }, error: detail }, 422), cache: new SearchLRUCache({ storage: null }) });
    await expect(client.search(request)).rejects.toMatchObject({ detail, rejection });
  });

  test("rejects malformed success envelopes safely", async () => {
    const client = new SearchAPIClient({ fetch: async () => jsonResponse(envelope({ unexpected: true })), cache: new SearchLRUCache({ storage: null }) });
    await expect(client.search(request)).rejects.toMatchObject({ detail: { code: "invalid_response", requestId: "req-1", retryable: false } });
  });

  test("uses local cache, stable keys, and previous-page placeholder data", async () => {
    const cache = new SearchLRUCache({ storage: null });
    cache.set(request, searchResponse);
    let calls = 0;
    const client = new SearchAPIClient({ fetch: async () => { calls += 1; return jsonResponse(envelope(searchResponse)); }, cache });
    const options = client.searchQueryOptions({ ...request, query: " Apple " });
    expect(options.queryKey).toEqual(["search", expect.any(String)]);
    expect(options.initialData).toEqual(searchResponse);
    expect(options.placeholderData).toBe(keepPreviousData);
    const queryClient = new QueryClient();
    await queryClient.fetchQuery(options);
    expect(calls).toBe(0);
  });

  test("deduplicates concurrent equivalent query keys", async () => {
    let calls = 0;
    const client = new SearchAPIClient({ fetch: async () => { calls += 1; await Bun.sleep(5); return jsonResponse(envelope(searchResponse)); }, cache: new SearchLRUCache({ storage: null }) });
    const queryClient = new QueryClient();
    await Promise.all([
      queryClient.fetchQuery(client.searchQueryOptions(request)),
      queryClient.fetchQuery(client.searchQueryOptions({ ...request, query: " APPLE " }))
    ]);
    expect(calls).toBe(1);
  });

  test("serves cached results offline and rejects uncached offline searches", async () => {
    const cache = new SearchLRUCache({ storage: null });
    cache.set(request, searchResponse);
    const client = new SearchAPIClient({ fetch: async () => { throw new Error("must not fetch"); }, cache });
    expect(await client.searchWithCache(request, false)).toEqual({ response: searchResponse, cached: true, stale: false });
    await expect(client.searchWithCache({ ...request, query: "banana" }, false)).rejects.toMatchObject({ detail: { code: "offline_cache_miss" } });
  });
});
