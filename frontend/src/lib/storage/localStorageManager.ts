import type { SearchRequest, SearchResponse } from '../api/types';

export const localCacheSchemaVersion = 'v1';
export const localQueryCacheKey = 'mealswapp.localQueryCache';
export const maxRecentQueries = 20;
export const maxHistoryEntries = 5;
export const defaultMaxBytes = 512 * 1024;

export interface CachedQuery {
  key: string;
  request: SearchRequest;
  response: SearchResponse;
  storedAt: string;
  staleAt: string;
}

export interface LocalQueryCache {
  schemaVersion: string;
  recentQueries: CachedQuery[];
  maxEntries: number;
  history: string[];
  maxHistory: number;
}

export interface QueryHydrationEntry {
  queryKey: readonly unknown[];
  state: {
    data: SearchResponse;
    dataUpdatedAt: number;
  };
}

export interface BrowserStorage {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
}

export class LocalStorageManager {
  private storage: BrowserStorage;
  private maxBytes: number;

  constructor(storage: BrowserStorage = defaultStorage(), maxBytes = defaultMaxBytes) {
    this.storage = storage;
    this.maxBytes = maxBytes;
  }

  load(): LocalQueryCache {
    const fallback = emptyCache();
    try {
      const raw = this.storage.getItem(localQueryCacheKey);
      if (!raw) {
        return fallback;
      }
      const cache = JSON.parse(raw) as LocalQueryCache;
      if (cache.schemaVersion !== localCacheSchemaVersion) {
        this.storage.removeItem(localQueryCacheKey);
        return fallback;
      }
      return normalizeCache(cache);
    } catch {
      this.storage.removeItem(localQueryCacheKey);
      return fallback;
    }
  }

  save(cache: LocalQueryCache): LocalQueryCache {
    let normalized = normalizeCache(cache);
    while (encodedSize(normalized) > this.maxBytes && normalized.recentQueries.length > 0) {
      normalized = { ...normalized, recentQueries: normalized.recentQueries.slice(0, -1) };
    }
    this.storage.setItem(localQueryCacheKey, JSON.stringify(normalized));
    return normalized;
  }

  readQuery(request: SearchRequest, now = new Date()): CachedQuery | null {
    const key = buildQueryCacheKey(request);
    const entry = this.load().recentQueries.find((candidate) => candidate.key === key);
    if (!entry) {
      return null;
    }
    if (Date.parse(entry.staleAt) <= now.getTime()) {
      return { ...entry };
    }
    return { ...entry };
  }

  writeQuery(request: SearchRequest, response: SearchResponse, now = new Date(), ttlMs = 15 * 60 * 1000): CachedQuery {
    const cache = this.load();
    const key = buildQueryCacheKey(request);
    const entry: CachedQuery = {
      key,
      request,
      response,
      storedAt: now.toISOString(),
      staleAt: new Date(now.getTime() + ttlMs).toISOString()
    };
    const recentQueries = [entry, ...cache.recentQueries.filter((candidate) => candidate.key !== key)].slice(0, cache.maxEntries);
    const historyLabel = request.query.trim();
    const history = historyLabel
      ? [historyLabel, ...cache.history.filter((candidate) => candidate.toLowerCase() !== historyLabel.toLowerCase())].slice(0, cache.maxHistory)
      : cache.history;
    this.save({ ...cache, recentQueries, history });
    return entry;
  }

  hydrateQueryEntries(): QueryHydrationEntry[] {
    return this.load().recentQueries.map((entry) => ({
      queryKey: buildTanStackQueryKey(entry.request),
      state: {
        data: entry.response,
        dataUpdatedAt: Date.parse(entry.storedAt)
      }
    }));
  }

  purge(): void {
    this.storage.removeItem(localQueryCacheKey);
  }
}

export function buildQueryCacheKey(request: SearchRequest): string {
  return stableStringify({
    query: request.query.trim().toLowerCase(),
    mode: request.mode,
    page: request.page,
    filters: request.filters ?? [],
    ingredients: request.ingredients ?? [],
    sourceItemId: request.sourceItemId ?? '',
    enabledMacros: request.enabledMacros,
    dietaryTagIds: request.dietaryTagIds ?? [],
    allergenTagIds: request.allergenTagIds ?? [],
    sourceProviders: request.sourceProviders ?? []
  });
}

export function buildTanStackQueryKey(request: SearchRequest): readonly unknown[] {
  return ['search', JSON.parse(buildQueryCacheKey(request))] as const;
}

export function emptyCache(): LocalQueryCache {
  return {
    schemaVersion: localCacheSchemaVersion,
    recentQueries: [],
    maxEntries: maxRecentQueries,
    history: [],
    maxHistory: maxHistoryEntries
  };
}

function normalizeCache(cache: LocalQueryCache): LocalQueryCache {
  return {
    schemaVersion: localCacheSchemaVersion,
    recentQueries: (cache.recentQueries ?? []).slice(0, cache.maxEntries || maxRecentQueries),
    maxEntries: cache.maxEntries || maxRecentQueries,
    history: (cache.history ?? []).slice(0, cache.maxHistory || maxHistoryEntries),
    maxHistory: cache.maxHistory || maxHistoryEntries
  };
}

function stableStringify(value: unknown): string {
  return JSON.stringify(sortValue(value));
}

function sortValue(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(sortValue);
  }
  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>)
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([key, nested]) => [key, sortValue(nested)])
    );
  }
  return value;
}

function encodedSize(cache: LocalQueryCache): number {
  return new TextEncoder().encode(JSON.stringify(cache)).length;
}

function defaultStorage(): BrowserStorage {
  if (typeof localStorage !== 'undefined') {
    return localStorage;
  }
  return new MemoryStorage();
}

class MemoryStorage implements BrowserStorage {
  private values = new Map<string, string>();

  getItem(key: string): string | null {
    return this.values.get(key) ?? null;
  }

  setItem(key: string, value: string): void {
    this.values.set(key, value);
  }

  removeItem(key: string): void {
    this.values.delete(key);
  }
}
