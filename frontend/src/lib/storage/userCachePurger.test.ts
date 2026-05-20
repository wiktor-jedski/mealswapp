import { describe, expect, it } from 'bun:test';
import { purgeCachesForAccountDeletion, purgeCachesForLogout, purgeUserCaches } from './userCachePurger';

describe('UserCachePurger', () => {
  it('purges local search storage and service worker cache buckets', async () => {
    const deleted: string[] = [];
    let localPurged = false;
    const messages: unknown[] = [];

    const result = await purgeUserCaches({
      localStorageManager: {
        purge: () => {
          localPurged = true;
        }
      },
      caches: {
        keys: async () => ['mealswapp-sw-v1:shell', 'mealswapp-sw-v1:api', 'other-cache'],
        delete: async (name) => {
          deleted.push(name);
          return true;
        }
      },
      serviceWorker: {
        controller: { postMessage: (message) => messages.push(message) },
        ready: Promise.resolve({ active: { postMessage: (message) => messages.push(message) } })
      }
    });

    expect(localPurged).toBe(true);
    expect(result.localStoragePurged).toBe(true);
    expect(result.deletedCacheNames).toEqual(['mealswapp-sw-v1:shell', 'mealswapp-sw-v1:api']);
    expect(deleted).toEqual(['mealswapp-sw-v1:shell', 'mealswapp-sw-v1:api']);
    expect(messages).toEqual([{ type: 'PURGE_USER_CACHE' }, { type: 'PURGE_USER_CACHE' }]);
    expect(result.errors).toEqual([]);
  });

  it('reports partial purge failures without throwing', async () => {
    const result = await purgeUserCaches({
      localStorageManager: {
        purge: () => {
          throw new Error('local blocked');
        }
      },
      caches: {
        keys: async () => ['mealswapp-sw-v1:images'],
        delete: async () => {
          throw new Error('cache blocked');
        }
      }
    });

    expect(result.localStoragePurged).toBe(false);
    expect(result.deletedCacheNames).toEqual([]);
    expect(result.errors).toEqual(['local blocked', 'cache blocked']);
  });

  it('uses the same purge behavior for logout and account deletion', async () => {
    let logoutPurged = false;
    let deletionPurged = false;

    await purgeCachesForLogout({ localStorageManager: { purge: () => (logoutPurged = true) } });
    await purgeCachesForAccountDeletion({ localStorageManager: { purge: () => (deletionPurged = true) } });

    expect(logoutPurged).toBe(true);
    expect(deletionPurged).toBe(true);
  });
});
