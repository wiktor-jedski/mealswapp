import { describe, expect, it } from 'bun:test';
import type { SearchRequest, SearchResponse } from '../api/types';
import {
  LocalStorageManager,
  buildQueryCacheKey,
  buildTanStackQueryKey,
  emptyCache,
  localCacheSchemaVersion,
  localQueryCacheKey,
  maxHistoryEntries,
  maxRecentQueries
} from './localStorageManager';

describe('LocalStorageManager', () => {
  it('stores and reuses cached search results by stable query key', () => {
    const manager = new LocalStorageManager(new MemoryStorage());
    const request = searchRequest('Tofu');
    const response = searchResponse('food-1');

    manager.writeQuery(request, response, new Date('2026-05-20T00:00:00.000Z'));
    const cached = manager.readQuery({ ...request, query: ' tofu ' });

    expect(cached?.response).toEqual(response);
    expect(cached?.key).toBe(buildQueryCacheKey(request));
  });

  it('evicts recent query cache with LRU ordering and deduplicates entries', () => {
    const manager = new LocalStorageManager(new MemoryStorage());
    for (let i = 0; i < maxRecentQueries + 3; i++) {
      manager.writeQuery(searchRequest(`query ${i}`), searchResponse(`food-${i}`), new Date(1000 + i));
    }
    manager.writeQuery(searchRequest('query 10'), searchResponse('food-new'), new Date(9999));

    const cache = manager.load();
    expect(cache.recentQueries.length).toBe(maxRecentQueries);
    expect(cache.recentQueries[0].request.query).toBe('query 10');
    expect(cache.recentQueries.filter((entry) => entry.request.query === 'query 10').length).toBe(1);
    expect(cache.recentQueries.some((entry) => entry.request.query === 'query 0')).toBe(false);
  });

  it('keeps five most recent unique history labels', () => {
    const manager = new LocalStorageManager(new MemoryStorage());
    for (let i = 0; i < maxHistoryEntries + 2; i++) {
      manager.writeQuery(searchRequest(`History ${i}`), searchResponse(`food-${i}`));
    }
    manager.writeQuery(searchRequest('history 3'), searchResponse('food-repeat'));

    expect(manager.load().history).toEqual(['history 3', 'History 6', 'History 5', 'History 4', 'History 2']);
  });

  it('purges cache on schema mismatch', () => {
    const storage = new MemoryStorage();
    storage.setItem(localQueryCacheKey, JSON.stringify({ ...emptyCache(), schemaVersion: 'old' }));

    const cache = new LocalStorageManager(storage).load();

    expect(cache).toEqual(emptyCache());
    expect(storage.getItem(localQueryCacheKey)).toBeNull();
  });

  it('evicts least recent entries until size limit is satisfied', () => {
    const storage = new MemoryStorage();
    const manager = new LocalStorageManager(storage, 2500);
    for (let i = 0; i < 10; i++) {
      manager.writeQuery(searchRequest(`large ${i}`), largeResponse(`food-${i}`));
    }

    const cache = manager.load();
    expect(cache.recentQueries.length).toBeLessThan(10);
    expect(cache.recentQueries[0].request.query).toBe('large 9');
    expect(new TextEncoder().encode(storage.getItem(localQueryCacheKey) ?? '').length).toBeLessThanOrEqual(2500);
  });

  it('builds TanStack Query hydration entries from cached results', () => {
    const manager = new LocalStorageManager(new MemoryStorage());
    const request = searchRequest('lentils');
    manager.writeQuery(request, searchResponse('food-1'), new Date('2026-05-20T00:00:00.000Z'));

    expect(manager.hydrateQueryEntries()).toEqual([
      {
        queryKey: buildTanStackQueryKey(request),
        state: {
          data: searchResponse('food-1'),
          dataUpdatedAt: Date.parse('2026-05-20T00:00:00.000Z')
        }
      }
    ]);
  });

  it('uses the current schema for new empty caches', () => {
    expect(emptyCache().schemaVersion).toBe(localCacheSchemaVersion);
  });
});

function searchRequest(query: string): SearchRequest {
  return { query, mode: 'single', page: 1, filters: [], enabledMacros: { protein: true, carbs: true, fat: true } };
}

function searchResponse(id: string): SearchResponse {
  return {
    items: [{ id, name: 'Tofu', tags: ['vegan'], macros: { protein: 10, carbs: 2, fat: 4, unitBasis: '100g' } }],
    totalCount: 1,
    page: 1,
    pageSize: 10,
    similarityScores: [],
    warnings: []
  };
}

function largeResponse(id: string): SearchResponse {
  const response = searchResponse(id);
  response.items[0].name = 'x'.repeat(500);
  return response;
}

class MemoryStorage {
  values = new Map<string, string>();

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
