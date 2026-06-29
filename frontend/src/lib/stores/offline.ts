import { writable } from "svelte/store";
import type { LocalQueryCache } from "../cache/local-query-cache";

// Implements DESIGN-001 OfflineBanner online/offline and stale-data indicator state.

/**
 * Offline status state consumed by the OfflineBanner to render online, cached,
 * stale, and uncached-offline indicators.
 *
 * @remarks Implements DESIGN-001 OfflineBanner online/offline and stale-data indicator state.
 */
export interface OfflineStatus {
	online: boolean;
	showingCached: boolean;
	showingStale: boolean;
}

/**
 * Default offline status: online with no cached/stale flags. Used as the startup
 * value and as the reset target for tests.
 *
 * @remarks Implements DESIGN-001 OfflineBanner startup initialization.
 */
export function createInitialOfflineStatus(): OfflineStatus {
	return { online: true, showingCached: false, showingStale: false };
}

/**
 * Svelte writable store holding the current offline status consumed by the OfflineBanner.
 *
 * @remarks Implements DESIGN-001 OfflineBanner Svelte store initialization.
 */
export const offlineStatus = writable<OfflineStatus>(createInitialOfflineStatus());

let onlineHandler: (() => void) | null = null;
let offlineHandler: (() => void) | null = null;

/**
 * Subscribes to browser `online` and `offline` events, syncing the store with
 * `navigator.onLine` on init and resetting cached/stale flags when connectivity
 * returns so a fresh request is permitted. Safe during SSR where `window` is
 * undefined: returns a no-op cleanup function and registers no listeners.
 *
 * @remarks Implements DESIGN-001 OfflineBanner browser online/offline event subscription.
 */
export function initOffline(): () => void {
	if (typeof window === "undefined") {
		return () => {};
	}

	onlineHandler = () => {
		offlineStatus.update((state) => ({
			...state,
			online: true,
			showingCached: false,
			showingStale: false
		}));
	};
	offlineHandler = () => {
		offlineStatus.update((state) => ({
			...state,
			online: false
		}));
	};
	window.addEventListener("online", onlineHandler);
	window.addEventListener("offline", offlineHandler);

	const initialOnline =
		typeof navigator !== "undefined" && typeof navigator.onLine === "boolean"
			? navigator.onLine
			: true;
	offlineStatus.update((state) => ({ ...state, online: initialOnline }));

	return cleanupOffline;
}

/**
 * Removes the `online` and `offline` event listeners registered by {@link initOffline}.
 * Safe during SSR and safe to call multiple times or before `initOffline`.
 *
 * @remarks Implements DESIGN-001 OfflineBanner event listener teardown.
 */
export function cleanupOffline(): void {
	if (typeof window === "undefined") {
		return;
	}
	if (onlineHandler !== null) {
		window.removeEventListener("online", onlineHandler);
		onlineHandler = null;
	}
	if (offlineHandler !== null) {
		window.removeEventListener("offline", offlineHandler);
		offlineHandler = null;
	}
}

/**
 * Flags that the UI is showing cached results (e.g. offline with a local cache hit)
 * so the OfflineBanner renders the cached-result label.
 *
 * @remarks Implements DESIGN-001 OfflineBanner cached-result indicator.
 */
export function setShowingCached(value: boolean): void {
	offlineStatus.update((state) => ({ ...state, showingCached: value }));
}

/**
 * Flags that the UI is showing stale results (e.g. offline with a stale cache entry)
 * so the OfflineBanner renders the stale-result label.
 *
 * @remarks Implements DESIGN-001 OfflineBanner stale-result indicator.
 */
export function setShowingStale(value: boolean): void {
	offlineStatus.update((state) => ({ ...state, showingStale: value }));
}

/**
 * Resets the offline status to defaults and removes any registered event listeners.
 * Used by tests to restore a clean state between cases.
 *
 * @remarks Implements DESIGN-001 OfflineBanner default restoration.
 */
export function resetOfflineStatus(): void {
	cleanupOffline();
	offlineStatus.set(createInitialOfflineStatus());
}

/**
 * Returns `true` when the cached result for `requestKey` is stale or missing,
 * delegating to the {@link LocalQueryCache} stale check so the OfflineBanner can
 * surface a stale-result label without duplicating staleness logic.
 *
 * @remarks Implements DESIGN-001 LocalStorageManager stale state reporting for the OfflineBanner.
 */
export function isStaleResult(
	requestKey: string,
	localCache: LocalQueryCache,
	maxAgeMs: number
): boolean {
	return localCache.isStale(requestKey, maxAgeMs);
}
