import { keepPreviousData, queryOptions } from "@tanstack/svelte-query";
import type { AppError, AutocompleteEnvelope, AutocompleteResponse, SearchRejection, SearchRequest, SearchResponse, SearchResponseEnvelope } from "./generated";
import { SearchLRUCache } from "../cache/search-lru";
import { searchRequestKey } from "../search/search-state";

export type FetchLike = (input: string | URL | Request, init?: RequestInit) => Promise<Response>;

export interface SearchAPIClientOptions {
  baseURL?: string;
  fetch?: FetchLike;
  cache?: SearchLRUCache;
  timeoutMs?: number;
}

export interface SearchLoadResult {
  response: SearchResponse;
  cached: boolean;
  stale: boolean;
}

// Implements DESIGN-017 ErrorMessageMapper safe frontend error transport.
export class AppClientError extends Error {
  constructor(readonly detail: AppError, readonly rejection?: SearchRejection) {
    super(detail.message);
    this.name = "AppClientError";
  }
}

// Implements DESIGN-001 SearchView generated-contract TanStack Query client.
export class SearchAPIClient {
  private readonly baseURL: string;
  private readonly fetcher: FetchLike;
  private readonly cache: SearchLRUCache;
  private readonly timeoutMs: number;

  constructor(options: SearchAPIClientOptions = {}) {
    this.baseURL = (options.baseURL ?? "").replace(/\/$/, "");
    this.fetcher = options.fetch ?? globalThis.fetch.bind(globalThis);
    this.cache = options.cache ?? new SearchLRUCache();
    this.timeoutMs = options.timeoutMs ?? 10_000;
  }

  async search(request: SearchRequest): Promise<SearchResponse> {
    const envelope = await this.request<SearchResponseEnvelope>("/api/v1/search", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(request)
    });
    if (!isSearchResponse(envelope.data)) throw new AppClientError(unknownError(envelope.requestId));
    this.cache.set(request, envelope.data);
    return envelope.data;
  }

  // Implements DESIGN-001 OfflineBanner local-cache fallback without service-worker claims.
  async searchWithCache(request: SearchRequest, online = typeof navigator === "undefined" || navigator.onLine): Promise<SearchLoadResult> {
    const cached = this.cache.get(request);
    if (!online) {
      if (cached) return { response: cached.response, cached: true, stale: cached.stale };
      throw new AppClientError({ category: "network", code: "offline_cache_miss", message: "No cached results are available while offline", retryable: true });
    }
    return { response: await this.search(request), cached: false, stale: false };
  }

  async autocomplete(query: string): Promise<AutocompleteResponse> {
    const envelope = await this.request<AutocompleteEnvelope>(`/api/v1/search/autocomplete?query=${encodeURIComponent(query)}`, { method: "GET" });
    if (!isAutocompleteResponse(envelope.data)) throw new AppClientError(unknownError(envelope.requestId));
    return envelope.data;
  }

  searchQueryOptions(request: SearchRequest) {
    const cached = this.cache.get(request);
    return queryOptions({
      queryKey: ["search", searchRequestKey(request)] as const,
      queryFn: () => this.search(request),
      placeholderData: keepPreviousData,
      initialData: cached?.response,
      initialDataUpdatedAt: cached ? Date.parse(cached.storedAt) : undefined,
      staleTime: cached && !cached.stale ? 5 * 60 * 1000 : 0,
      retry: (_count, error) => error instanceof AppClientError && error.detail.retryable
    });
  }

  autocompleteQueryOptions(value: string) {
    const query = value.trim();
    return queryOptions({
      queryKey: ["autocomplete", query.toLocaleLowerCase()] as const,
      queryFn: () => this.autocomplete(query),
      enabled: query.length > 0,
      retry: (_count, error) => error instanceof AppClientError && error.detail.retryable
    });
  }

  private async request<T>(path: string, init: RequestInit): Promise<T> {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), this.timeoutMs);
    try {
      const response = await this.fetcher(`${this.baseURL}${path}`, { ...init, credentials: "include", signal: controller.signal });
      const payload: unknown = await response.json().catch(() => null);
      if (!response.ok) throw errorFromPayload(payload, response.status);
      if (!isEnvelope(payload)) throw new AppClientError(unknownError());
      return payload as T;
    } catch (error) {
      if (error instanceof AppClientError) throw error;
      if (controller.signal.aborted) throw new AppClientError({ category: "timeout", code: "request_timeout", message: "Request timed out", retryable: true });
      throw new AppClientError({ category: "network", code: "network_error", message: "Network request failed", retryable: true });
    } finally {
      clearTimeout(timeout);
    }
  }
}

function errorFromPayload(payload: unknown, status: number): AppClientError {
  if (isEnvelope(payload) && isAppError(payload.error)) {
    const rejection = isRecord(payload.data) && isSearchRejection(payload.data.rejection) ? payload.data.rejection : undefined;
    return new AppClientError(payload.error, rejection);
  }
  const category: AppError["category"] = status === 400 || status === 422 ? "validation" : status === 429 ? "entitlement" : status === 503 ? "dependency" : "server";
  return new AppClientError({ category, code: `http_${status}`, message: status >= 500 ? "Service temporarily unavailable" : "Request could not be completed", retryable: status === 429 || status >= 500 });
}

function unknownError(requestId?: string): AppError {
  return { category: "unknown", code: "invalid_response", message: "The server returned an invalid response", retryable: false, requestId };
}

function isEnvelope(value: unknown): value is { status: string; requestId: string; data?: unknown; error?: unknown } {
  return isRecord(value) && typeof value.status === "string" && typeof value.requestId === "string";
}

function isAppError(value: unknown): value is AppError {
  return isRecord(value) && typeof value.category === "string" && typeof value.code === "string" && typeof value.message === "string" && typeof value.retryable === "boolean";
}

function isSearchRejection(value: unknown): value is SearchRejection {
  return isRecord(value) && typeof value.code === "string" && typeof value.message === "string" && (value.field === undefined || typeof value.field === "string");
}

function isSearchResponse(value: unknown): value is SearchResponse {
  return isRecord(value) && Array.isArray(value.items) && Number.isInteger(value.totalCount) && Number.isInteger(value.page) && Array.isArray(value.similarityScores) && Array.isArray(value.similarityMetadata) && Array.isArray(value.warnings);
}

function isAutocompleteResponse(value: unknown): value is AutocompleteResponse {
  return isRecord(value) && Array.isArray(value.items);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
