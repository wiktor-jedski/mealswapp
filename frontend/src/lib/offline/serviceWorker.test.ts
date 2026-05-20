import { describe, expect, it } from 'bun:test';
import {
  registerServiceWorker,
  serviceWorkerCacheLimits,
  serviceWorkerPath,
  shouldCacheAPIGet,
  shouldCacheImage,
  shouldCacheShellAsset,
  staleHeader
} from './serviceWorker';

describe('service worker registration and cache policy', () => {
  it('registers the service worker with root scope when supported', async () => {
    const calls: unknown[] = [];
    const result = await registerServiceWorker({
      serviceWorker: {
        register: async (scriptURL, options) => {
          calls.push([scriptURL, options]);
        }
      }
    });

    expect(result.registered).toBe(true);
    expect(result.scriptURL).toBe(serviceWorkerPath);
    expect(calls[0]).toEqual(['/sw.js', { scope: '/' }]);
  });

  it('reports unsupported and failed registration states without throwing', async () => {
    expect((await registerServiceWorker({})).supported).toBe(false);
    const failed = await registerServiceWorker({
      serviceWorker: {
        register: async () => {
          throw new Error('blocked');
        }
      }
    });
    expect(failed.registered).toBe(false);
    expect(failed.error).toBe('blocked');
  });

  it('classifies shell assets, search API GETs, and offline images', () => {
    expect(shouldCacheShellAsset('/')).toBe(true);
    expect(shouldCacheShellAsset('/assets/index.js')).toBe(true);
    expect(shouldCacheShellAsset('/api/v1/search')).toBe(false);

    expect(shouldCacheAPIGet('/api/v1/search?query=tofu', 'GET')).toBe(true);
    expect(shouldCacheAPIGet('/api/v1/autocomplete?query=tof', 'GET')).toBe(true);
    expect(shouldCacheAPIGet('/api/v1/search', 'POST')).toBe(false);

    expect(shouldCacheImage('/static/similarity/green.svg')).toBe(true);
    expect(shouldCacheImage('/images/placeholder-food.png')).toBe(true);
    expect(staleHeader(true)['X-MealSwapp-Stale']).toBe('true');
  });

  it('documents service worker cache size limits for LRU trimming', () => {
    expect(serviceWorkerCacheLimits).toEqual({ shell: 20, images: 60, api: 30 });
  });
});
