import { LocalStorageManager } from './localStorageManager';

export const serviceWorkerCachePrefix = 'mealswapp-sw-';

export interface CacheStorageLike {
  keys(): Promise<string[]>;
  delete(cacheName: string): Promise<boolean>;
}

export interface ServiceWorkerRegistrationLike {
  active?: { postMessage(message: unknown): void } | null;
}

export interface ServiceWorkerContainerLike {
  ready?: Promise<ServiceWorkerRegistrationLike>;
  controller?: { postMessage(message: unknown): void } | null;
}

export interface UserCachePurgeTarget {
  localStorageManager?: Pick<LocalStorageManager, 'purge'>;
  caches?: CacheStorageLike;
  serviceWorker?: ServiceWorkerContainerLike;
}

export interface UserCachePurgeResult {
  localStoragePurged: boolean;
  deletedCacheNames: string[];
  errors: string[];
}

export async function purgeUserCaches(target: UserCachePurgeTarget = {}): Promise<UserCachePurgeResult> {
  const result: UserCachePurgeResult = { localStoragePurged: false, deletedCacheNames: [], errors: [] };
  const localStorageManager = target.localStorageManager ?? new LocalStorageManager();

  try {
    localStorageManager.purge();
    result.localStoragePurged = true;
  } catch (error) {
    result.errors.push(errorMessage(error));
  }

  await postServiceWorkerPurge(target.serviceWorker, result);
  await deleteCacheBuckets(target.caches ?? defaultCaches(), result);
  return result;
}

export function purgeCachesForLogout(target: UserCachePurgeTarget = {}): Promise<UserCachePurgeResult> {
  return purgeUserCaches(target);
}

export function purgeCachesForAccountDeletion(target: UserCachePurgeTarget = {}): Promise<UserCachePurgeResult> {
  return purgeUserCaches(target);
}

async function postServiceWorkerPurge(serviceWorker: ServiceWorkerContainerLike | undefined, result: UserCachePurgeResult): Promise<void> {
  try {
    serviceWorker?.controller?.postMessage({ type: 'PURGE_USER_CACHE' });
    const registration = await serviceWorker?.ready;
    registration?.active?.postMessage({ type: 'PURGE_USER_CACHE' });
  } catch (error) {
    result.errors.push(errorMessage(error));
  }
}

async function deleteCacheBuckets(caches: CacheStorageLike | undefined, result: UserCachePurgeResult): Promise<void> {
  if (!caches) {
    return;
  }
  try {
    const names = await caches.keys();
    for (const name of names.filter((candidate) => candidate.startsWith(serviceWorkerCachePrefix))) {
      try {
        if (await caches.delete(name)) {
          result.deletedCacheNames.push(name);
        }
      } catch (error) {
        result.errors.push(errorMessage(error));
      }
    }
  } catch (error) {
    result.errors.push(errorMessage(error));
  }
}

function defaultCaches(): CacheStorageLike | undefined {
  if (typeof globalThis.caches !== 'undefined') {
    return globalThis.caches;
  }
  return undefined;
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'cache purge failed';
}
