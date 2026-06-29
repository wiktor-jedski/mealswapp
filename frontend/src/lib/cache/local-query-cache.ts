import type { SearchRequest, SearchResponse } from "../api/generated";

// Implements DESIGN-001 LocalStorageManager query metadata persistence and LRU refresh.

/**
 * Schema version stamped on every persisted cache entry. Entries that do not match
 * this version on load are discarded so cache shape changes do not crash the UI.
 *
 * @remarks Implements DESIGN-001 LocalStorageManager schema evolution safety.
 */
export const SCHEMA_VERSION = "local-query-cache-v1";

/**
 * Maximum number of unique normalized search requests kept in localStorage.
 *
 * @remarks Implements DESIGN-001 LocalStorageManager 20 most recent unique query persistence.
 */
export const MAX_ENTRIES = 20;

/**
 * Storage key under which the cache blob is persisted as a single JSON string.
 *
 * @remarks Implements DESIGN-001 LocalStorageManager client persistence key.
 */
export const STORAGE_KEY = "mealswapp.local-query-cache";

/**
 * Schema-versioned cache entry persisted to localStorage and refreshed on read.
 *
 * @remarks Implements DESIGN-001 LocalStorageManager CachedQuery shape.
 */
export interface LocalQueryCacheEntry {
	version: string;
	requestKey: string;
	request: SearchRequest;
	response: SearchResponse;
	storedAt: number;
	lastAccessedAt: number;
}

/**
 * Narrow storage seam accepted by the cache so tests can pass lightweight fakes
 * without satisfying the full DOM `Storage` interface.
 */
export interface LocalQueryCacheStorage {
	getItem(key: string): string | null;
	setItem(key: string, value: string): void;
	removeItem(key: string): void;
}

/**
 * Constructor options for {@link LocalQueryCache}.
 */
export interface LocalQueryCacheOptions {
	/** Persistence backing. Pass `null` for an in-memory-only cache. */
	storage?: LocalQueryCacheStorage | null;
	/** Clock injection for deterministic tests; defaults to `Date.now`. */
	now?: () => number;
}

interface StoredPayload {
	entries: LocalQueryCacheEntry[];
}

/**
 * LRU cache of the twenty most recent unique normalized search requests and their
 * `SearchResponse` result sets, persisted to localStorage with schema validation
 * and a storage-unavailable fallback to an in-memory map.
 *
 * @remarks Implements DESIGN-001 LocalStorageManager 20 most recent unique query persistence with LRU refresh.
 */
export class LocalQueryCache {
	private readonly storage: LocalQueryCacheStorage | null;
	private readonly now: () => number;
	private readonly entriesByKey: Map<string, LocalQueryCacheEntry> = new Map();

	constructor(options: LocalQueryCacheOptions = {}) {
		this.storage = options.storage ?? null;
		this.now = options.now ?? (() => Date.now());
		this.loadFromStorage();
	}

	/**
	 * Returns the entry for `requestKey` and refreshes its LRU recency, or `null` on miss.
	 *
	 * @remarks Implements DESIGN-001 LocalStorageManager cache hit LRU refresh.
	 */
	get(requestKey: string): LocalQueryCacheEntry | null {
		const existing = this.entriesByKey.get(requestKey);
		if (existing === undefined) {
			return null;
		}
		const refreshed: LocalQueryCacheEntry = {
			...existing,
			lastAccessedAt: this.now()
		};
		this.entriesByKey.set(requestKey, refreshed);
		this.persist();
		return refreshed;
	}

	/**
	 * Returns the entry for `requestKey` without refreshing LRU recency. Used when
	 * callers need to preserve stale metadata while rendering an offline fallback.
	 *
	 * @remarks Implements DESIGN-001 LocalStorageManager stale state reporting.
	 */
	peek(requestKey: string): LocalQueryCacheEntry | null {
		return this.entriesByKey.get(requestKey) ?? null;
	}

	/**
	 * Inserts or replaces the entry for `requestKey`, evicting the least-recently-accessed
	 * entry when the cache exceeds {@link MAX_ENTRIES}.
	 *
	 * @remarks Implements DESIGN-001 LocalStorageManager cache write and LRU eviction.
	 */
	set(requestKey: string, request: SearchRequest, response: SearchResponse): void {
		const now = this.now();
		const existing = this.entriesByKey.get(requestKey);
		const entry: LocalQueryCacheEntry = {
			version: SCHEMA_VERSION,
			requestKey,
			request,
			response,
			storedAt: existing?.storedAt ?? now,
			lastAccessedAt: now
		};
		this.entriesByKey.set(requestKey, entry);
		this.evictIfNeeded();
		this.persist();
	}

	/**
	 * Returns `true` when an entry exists for `requestKey` without refreshing recency.
	 *
	 * @remarks Implements DESIGN-001 LocalStorageManager cache presence check.
	 */
	has(requestKey: string): boolean {
		return this.entriesByKey.has(requestKey);
	}

	/**
	 * Removes all entries and clears the persisted blob when storage is available.
	 *
	 * @remarks Implements DESIGN-001 LocalStorageManager UserCachePurger local query purge.
	 */
	clear(): void {
		this.entriesByKey.clear();
		this.persist();
	}

	/**
	 * Returns `true` when the entry is missing or its `lastAccessedAt` is older than `maxAgeMs`.
	 *
	 * @remarks Implements DESIGN-001 LocalStorageManager stale state reporting.
	 */
	isStale(requestKey: string, maxAgeMs: number): boolean {
		const entry = this.entriesByKey.get(requestKey);
		if (entry === undefined) {
			return true;
		}
		return this.now() - entry.lastAccessedAt > maxAgeMs;
	}

	/**
	 * Returns all entries ordered most-recent first for debugging and inspection.
	 *
	 * @remarks Implements DESIGN-001 LocalStorageManager cache inspection.
	 */
	entries(): LocalQueryCacheEntry[] {
		return Array.from(this.entriesByKey.values()).sort(
			(a, b) => b.lastAccessedAt - a.lastAccessedAt
		);
	}

	private loadFromStorage(): void {
		if (this.storage === null) {
			return;
		}
		let raw: string | null;
		try {
			raw = this.storage.getItem(STORAGE_KEY);
		} catch {
			// Storage reads are optional; the cache remains usable in memory.
			return;
		}
		if (raw === null) {
			return;
		}
		let parsed: unknown;
		try {
			parsed = JSON.parse(raw);
		} catch {
			// Malformed persisted cache data is ignored instead of blocking search.
			return;
		}
		if (!isStoredPayload(parsed)) {
			return;
		}
		for (const entry of parsed.entries) {
			if (isValidEntry(entry) && entry.version === SCHEMA_VERSION) {
				this.entriesByKey.set(entry.requestKey, entry);
			}
		}
	}

	private evictIfNeeded(): void {
		while (this.entriesByKey.size > MAX_ENTRIES) {
			let lruKey: string | null = null;
			let lruTime = Number.POSITIVE_INFINITY;
			for (const [key, entry] of this.entriesByKey) {
				if (entry.lastAccessedAt < lruTime) {
					lruTime = entry.lastAccessedAt;
					lruKey = key;
				}
			}
			if (lruKey === null) {
				break;
			}
			this.entriesByKey.delete(lruKey);
		}
	}

	private persist(): void {
		if (this.storage === null) {
			return;
		}
		const payload: StoredPayload = {
			entries: Array.from(this.entriesByKey.values())
		};
		try {
			this.storage.setItem(STORAGE_KEY, JSON.stringify(payload));
		} catch {
			// Quota exceeded or storage disabled; the in-memory map still serves callers.
			return;
		}
	}
}

/**
 * Creates a localStorage-backed {@link LocalQueryCache}, falling back to an in-memory
 * cache when `window` is undefined or the localStorage probe throws.
 *
 * @remarks Implements DESIGN-001 LocalStorageManager storage-unavailable fallback.
 */
export function createLocalQueryCache(): LocalQueryCache {
	if (typeof window === "undefined") {
		return new LocalQueryCache({ storage: null });
	}
	try {
		const probeKey = "mealswapp.local-query-cache.probe";
		window.localStorage.setItem(probeKey, "1");
		window.localStorage.removeItem(probeKey);
		return new LocalQueryCache({ storage: window.localStorage });
	} catch {
		// Storage is unavailable or disabled; fall back to an in-memory cache.
		return new LocalQueryCache({ storage: null });
	}
}

function isStoredPayload(value: unknown): value is StoredPayload {
	if (typeof value !== "object" || value === null) {
		return false;
	}
	const candidate = value as { entries?: unknown };
	return Array.isArray(candidate.entries);
}

/**
 * Runtime guard for persisted local-query cache entries loaded from localStorage.
 *
 * @remarks Implements DESIGN-001 LocalStorageManager cache-entry schema validation.
 */
function isValidEntry(value: unknown): value is LocalQueryCacheEntry {
	if (typeof value !== "object" || value === null) {
		return false;
	}
	const entry = value as Record<string, unknown>;
	return (
		typeof entry.version === "string" &&
		typeof entry.requestKey === "string" &&
		typeof entry.request === "object" &&
		entry.request !== null &&
		typeof entry.response === "object" &&
		entry.response !== null &&
		typeof entry.storedAt === "number" &&
		typeof entry.lastAccessedAt === "number"
	);
}
