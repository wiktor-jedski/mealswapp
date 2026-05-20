import { describe, expect, it } from 'bun:test';
import { ApiClientError } from '../api/client';
import type { SearchRequest, SearchResponse } from '../api/types';
import { buildQueryCacheKey, type CachedQuery } from '../storage/localStorageManager';
import { buildSearchRequest, createDefaultSearchState, createSearchController } from './searchState';

describe('SearchView state', () => {
  it('starts in single-item mode with all macros enabled', () => {
    const state = createDefaultSearchState();

    expect(state.mode).toBe('single');
    expect(state.enabledMacros).toEqual({ protein: true, carbs: true, fat: true });
    expect(state.status).toBe('idle');
  });

  it('builds trimmed search request payloads', () => {
    const request = buildSearchRequest({
      ...createDefaultSearchState(),
      query: ' tofu ',
      page: 2,
      filters: [{ tagId: 'tag-1', kind: 'functionality', include: true }],
      enabledMacros: { protein: true, carbs: false, fat: true }
    });

    expect(request).toEqual({
      query: 'tofu',
      mode: 'single',
      page: 2,
      filters: [{ tagId: 'tag-1', kind: 'functionality', include: true }],
      enabledMacros: { protein: true, carbs: false, fat: true }
    });
  });

  it('debounces query updates before calling the API', async () => {
    const timers: Array<() => void> = [];
    const calls: unknown[] = [];
    const controller = createSearchController({
      api: { search: async (request) => {
        calls.push(request);
        return emptyResponse();
      } },
      setTimeoutFn: (callback) => {
        timers.push(callback);
        return callback;
      },
      clearTimeoutFn: () => undefined
    });

    controller.setQuery('to');
    controller.setQuery('tofu');

    expect(controller.getState().status).toBe('debouncing');
    expect(calls.length).toBe(0);
    await timers[timers.length - 1]?.();
    expect(calls).toEqual([{ query: 'tofu', mode: 'single', page: 1, filters: [], enabledMacros: { protein: true, carbs: true, fat: true } }]);
  });

  it('sets loading then empty state for empty responses', async () => {
    const seen: string[] = [];
    const controller = createSearchController({
      api: { search: async () => emptyResponse() }
    });
    controller.subscribe((state) => seen.push(state.status));
    controller.setQuery('missing');

    await controller.execute();

    expect(seen).toContain('loading');
    expect(controller.getState().status).toBe('empty');
    expect(controller.getState().isLoading).toBe(false);
  });

  it('sets success state for non-empty responses', async () => {
    const controller = createSearchController({
      api: { search: async () => ({ ...emptyResponse(), items: [{ id: 'food-1', name: 'Tofu', tags: [], macros: { protein: 10, carbs: 2, fat: 4, unitBasis: '100g' } }] }) }
    });
    controller.setQuery('tofu');

    await controller.execute();

    expect(controller.getState().status).toBe('success');
    expect(controller.getState().response?.items[0].name).toBe('Tofu');
  });

  it('sets error state for typed API errors', async () => {
    const controller = createSearchController({
      api: {
        search: async () => {
          throw new ApiClientError({ category: 'dependency', code: 'dependency_unavailable', message: 'Dependency unavailable', retryable: true });
        }
      }
    });
    controller.setQuery('tofu');

    await controller.execute();

    expect(controller.getState().status).toBe('error');
    expect(controller.getState().error?.code).toBe('dependency_unavailable');
  });

  it('updates mode, macro toggles, and filters from sidebar events', () => {
    const controller = createSearchController({ api: { search: async () => emptyResponse() } });

    controller.setMode('diet');
    controller.setMacro('carbs', false);
    controller.setFilters([{ tagId: 'diet-vegan', kind: 'diet', include: true }]);

    expect(controller.getState().mode).toBe('diet');
    expect(controller.getState().enabledMacros.carbs).toBe(false);
    expect(controller.getState().filters).toEqual([{ tagId: 'diet-vegan', kind: 'diet', include: true }]);
  });

  it('updates page and issues a paginated request', async () => {
    const calls: unknown[] = [];
    const controller = createSearchController({
      api: { search: async (request) => {
        calls.push(request);
        return emptyResponse();
      } }
    });
    controller.setQuery('tofu');

    controller.setPage(2);
    await Promise.resolve();
    await Promise.resolve();

    expect(controller.getState().page).toBe(2);
    expect(calls[calls.length - 1]).toEqual({ query: 'tofu', mode: 'single', page: 2, filters: [], enabledMacros: { protein: true, carbs: true, fat: true } });
  });

  it('writes successful results and hydrates offline searches from local cache', async () => {
    const cachedResponse = { ...emptyResponse(), items: [{ id: 'cached', name: 'Cached tofu', tags: [], macros: { protein: 10, carbs: 2, fat: 4, unitBasis: '100g' as const } }] };
    const cache = {
      stored: undefined as SearchResponse | undefined,
      readQuery: (request: unknown) =>
        cache.stored
          ? ({
              key: buildQueryCacheKey(request as SearchRequest),
              request: request as SearchRequest,
              response: cache.stored,
              storedAt: new Date(0).toISOString(),
              staleAt: new Date(1000).toISOString()
            } satisfies CachedQuery)
          : null,
      writeQuery: (request: unknown, response: SearchResponse) => {
        cache.stored = response;
        return {
          key: buildQueryCacheKey(request as SearchRequest),
          request: request as SearchRequest,
          response,
          storedAt: new Date(0).toISOString(),
          staleAt: new Date(1000).toISOString()
        } satisfies CachedQuery;
      }
    };
    const controller = createSearchController({
      api: { search: async () => cachedResponse },
      localStorageManager: cache
    });
    controller.setQuery('tofu');
    await controller.execute();
    controller.setOnline(false);
    controller.setQuery('tofu');

    await controller.execute();

    expect(controller.getState().status).toBe('success');
    expect(controller.getState().response?.items[0].name).toBe('Cached tofu');
  });
});

function emptyResponse(): SearchResponse {
  return {
    items: [],
    totalCount: 0,
    page: 1,
    pageSize: 10,
    similarityScores: [],
    warnings: []
  };
}
