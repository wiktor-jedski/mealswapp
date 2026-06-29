import { afterEach, expect, test } from "bun:test";
import { get } from "svelte/store";
import type { SearchResponse } from "../api/generated";
import { resetSearch, searchRequestKey, searchStore, setQuery } from "../stores/search";
import {
	LocalQueryCache,
	MAX_ENTRIES,
	SCHEMA_VERSION,
	STORAGE_KEY,
	createLocalQueryCache
} from "./local-query-cache";

// Implements DESIGN-001 LocalStorageManager LRU cache verification.

afterEach(() => {
	resetSearch();
});

function makeResponse(seed: number): SearchResponse {
	return {
		items: [{ id: `food-${seed}`, name: `Item ${seed}`, physicalState: "solid" }],
		totalCount: 1,
		page: 1,
		similarityScores: [0.9],
		similarityMetadata: [
			{
				itemId: `food-${seed}`,
				score: 0.9,
				tier: "excellent",
				imageUrl: `https://example.com/${seed}.png`,
				matchingQuantity: 100
			}
		],
		warnings: []
	};
}

class FakeStorage {
	private data = new Map<string, string>();
	getItem(key: string): string | null {
		return this.data.has(key) ? (this.data.get(key) as string) : null;
	}
	setItem(key: string, value: string): void {
		this.data.set(key, value);
	}
	removeItem(key: string): void {
		this.data.delete(key);
	}
}

class ThrowingStorage {
	getItem(): string | null {
		throw new Error("storage unavailable");
	}
	setItem(): void {
		throw new Error("storage unavailable");
	}
	removeItem(): void {
		throw new Error("storage unavailable");
	}
}

class SetItemThrowingStorage {
	private data = new Map<string, string>();
	getItem(key: string): string | null {
		return this.data.has(key) ? (this.data.get(key) as string) : null;
	}
	setItem(): void {
		throw new Error("quota exceeded");
	}
	removeItem(key: string): void {
		this.data.delete(key);
	}
}

// Implements DESIGN-001 LocalStorageManager equivalent request key sharing verification.
test("equivalent search states share a cache entry via searchRequestKey", () => {
	setQuery("apple");
	const keyA = searchRequestKey(get(searchStore));
	resetSearch();
	setQuery("apple");
	const keyB = searchRequestKey(get(searchStore));
	expect(keyA).toBe(keyB);

	const cache = new LocalQueryCache({ storage: null });
	cache.set(keyA, { query: "apple", mode: "catalog", page: 1 }, makeResponse(0));

	const hit = cache.get(keyB);
	expect(hit).not.toBeNull();
	expect(hit?.response.items[0]?.id).toBe("food-0");
});

// Implements DESIGN-001 LocalStorageManager LRU refresh on cache hit verification.
test("cache hits move entries to most-recent by updating lastAccessedAt", () => {
	const now = { value: 1000 };
	const cache = new LocalQueryCache({ storage: null, now: () => now.value });

	now.value = 1000;
	cache.set("key-a", { query: "a", mode: "catalog", page: 1 }, makeResponse(0));
	now.value = 2000;
	cache.set("key-b", { query: "b", mode: "catalog", page: 1 }, makeResponse(1));

	now.value = 3000;
	const hit = cache.get("key-a");
	expect(hit).not.toBeNull();
	expect(hit?.lastAccessedAt).toBe(3000);

	const ordered = cache.entries();
	expect(ordered[0]?.requestKey).toBe("key-a");
	expect(ordered[1]?.requestKey).toBe("key-b");
});

// Implements DESIGN-001 LocalStorageManager twenty-entry LRU eviction verification.
test("the twenty-first unique entry evicts the least-recently-accessed entry", () => {
	const now = { value: 1000 };
	const cache = new LocalQueryCache({ storage: null, now: () => now.value });

	for (let i = 0; i <= MAX_ENTRIES; i++) {
		now.value = 1000 + i * 100;
		cache.set(`key-${i}`, { query: `q${i}`, mode: "catalog", page: 1 }, makeResponse(i));
	}

	// key-0 was inserted first and never accessed again, so it is the LRU victim.
	expect(cache.has("key-0")).toBe(false);
	expect(cache.has("key-1")).toBe(true);
	expect(cache.has(`key-${MAX_ENTRIES}`)).toBe(true);
	expect(cache.entries()).toHaveLength(MAX_ENTRIES);
});

// Implements DESIGN-001 LocalStorageManager malformed JSON tolerance verification.
test("malformed JSON in storage is ignored and the cache starts empty", () => {
	const storage = new FakeStorage();
	storage.setItem(STORAGE_KEY, "{not valid json");
	const cache = new LocalQueryCache({ storage });
	expect(cache.entries()).toEqual([]);
	expect(cache.has("anything")).toBe(false);
});

// Implements DESIGN-001 LocalStorageManager schema-version mismatch tolerance verification.
test("schema-version-mismatched entries are ignored on load", () => {
	const storage = new FakeStorage();
	const stalePayload = {
		entries: [
			{
				version: "local-query-cache-v0",
				requestKey: "stale-key",
				request: { query: "a", mode: "catalog", page: 1 },
				response: makeResponse(0),
				storedAt: 1000,
				lastAccessedAt: 1000
			}
		]
	};
	storage.setItem(STORAGE_KEY, JSON.stringify(stalePayload));
	const cache = new LocalQueryCache({ storage });
	expect(cache.has("stale-key")).toBe(false);
	expect(cache.entries()).toEqual([]);
});

// Implements DESIGN-001 LocalStorageManager schema-version match acceptance verification.
test("schema-version-matched entries are loaded from storage", () => {
	const storage = new FakeStorage();
	const payload = {
		entries: [
			{
				version: SCHEMA_VERSION,
				requestKey: "good-key",
				request: { query: "a", mode: "catalog", page: 1 },
				response: makeResponse(7),
				storedAt: 1000,
				lastAccessedAt: 1000
			}
		]
	};
	storage.setItem(STORAGE_KEY, JSON.stringify(payload));
	const cache = new LocalQueryCache({ storage });
	expect(cache.has("good-key")).toBe(true);
	const hit = cache.get("good-key");
	expect(hit?.response.items[0]?.id).toBe("food-7");
});

// Implements DESIGN-001 LocalStorageManager stale state reporting verification.
test("isStale reports stale for missing or aged entries and fresh for recent ones", () => {
	const now = { value: 1000 };
	const cache = new LocalQueryCache({ storage: null, now: () => now.value });

	expect(cache.isStale("missing", 1000)).toBe(true);

	now.value = 1000;
	cache.set("key", { query: "a", mode: "catalog", page: 1 }, makeResponse(0));

	now.value = 1300;
	expect(cache.isStale("key", 500)).toBe(false);

	now.value = 1600;
	expect(cache.isStale("key", 500)).toBe(true);
});

// Implements DESIGN-001 LocalStorageManager stale metadata preservation verification.
test("peek returns a cached entry without refreshing stale metadata", () => {
	const now = { value: 1000 };
	const cache = new LocalQueryCache({ storage: null, now: () => now.value });
	cache.set("key", { query: "a", mode: "catalog", page: 1 }, makeResponse(0));

	now.value = 1600;

	expect(cache.peek("key")?.response.items[0]?.id).toBe("food-0");
	expect(cache.isStale("key", 500)).toBe(true);
});

// Implements DESIGN-001 LocalStorageManager storage-unavailable fallback verification.
test("localStorage setItem failures degrade to in-memory cache without throwing", () => {
	const storage = new SetItemThrowingStorage();
	const cache = new LocalQueryCache({ storage });

	expect(() =>
		cache.set("key", { query: "a", mode: "catalog", page: 1 }, makeResponse(0))
	).not.toThrow();

	const hit = cache.get("key");
	expect(hit).not.toBeNull();
	expect(hit?.response.items[0]?.id).toBe("food-0");
});

// Implements DESIGN-001 LocalStorageManager storage-read failure tolerance verification.
test("localStorage read failures start with an empty in-memory cache", () => {
	const storage = new ThrowingStorage();
	const cache = new LocalQueryCache({ storage });
	expect(cache.entries()).toEqual([]);
	expect(() =>
		cache.set("key", { query: "a", mode: "catalog", page: 1 }, makeResponse(0))
	).not.toThrow();
	expect(cache.get("key")).not.toBeNull();
});

// Implements DESIGN-001 LocalStorageManager clear verification.
test("clear removes all entries", () => {
	const storage = new FakeStorage();
	const cache = new LocalQueryCache({ storage });
	cache.set("key-a", { query: "a", mode: "catalog", page: 1 }, makeResponse(0));
	cache.set("key-b", { query: "b", mode: "catalog", page: 1 }, makeResponse(1));
	expect(cache.entries()).toHaveLength(2);

	cache.clear();
	expect(cache.entries()).toEqual([]);
	expect(cache.has("key-a")).toBe(false);
});

// Implements DESIGN-001 LocalStorageManager persistence round-trip verification.
test("entries persist across cache instances backed by the same storage", () => {
	const storage = new FakeStorage();
	const first = new LocalQueryCache({ storage });
	first.set("key", { query: "a", mode: "catalog", page: 1 }, makeResponse(3));

	const second = new LocalQueryCache({ storage });
	const hit = second.get("key");
	expect(hit).not.toBeNull();
	expect(hit?.response.items[0]?.id).toBe("food-3");
});

// Implements DESIGN-001 LocalStorageManager createLocalQueryCache SSR fallback verification.
test("createLocalQueryCache returns an in-memory cache when window is undefined", () => {
	// Bun's test environment has no `window` global, so this exercises the SSR path directly.
	expect(typeof globalThis.window).toBe("undefined");
	const cache = createLocalQueryCache();
	expect(() =>
		cache.set("key", { query: "a", mode: "catalog", page: 1 }, makeResponse(0))
	).not.toThrow();
	expect(cache.get("key")).not.toBeNull();
});

// Implements DESIGN-001 LocalStorageManager createLocalQueryCache probe-throws fallback verification.
test("createLocalQueryCache falls back to in-memory when the localStorage probe throws", () => {
	const throwingLocalStorage = {
		getItem(): string | null {
			return null;
		},
		setItem(): void {
			throw new Error("denied");
		},
		removeItem(): void {
			throw new Error("denied");
		}
	};
	const originalDescriptor = Object.getOwnPropertyDescriptor(globalThis, "window");
	Object.defineProperty(globalThis, "window", {
		configurable: true,
		value: { localStorage: throwingLocalStorage }
	});
	try {
		const cache = createLocalQueryCache();
		expect(() =>
			cache.set("key", { query: "a", mode: "catalog", page: 1 }, makeResponse(0))
		).not.toThrow();
		expect(cache.get("key")).not.toBeNull();
	} finally {
		if (originalDescriptor === undefined) {
			delete (globalThis as { window?: unknown }).window;
		} else {
			Object.defineProperty(globalThis, "window", originalDescriptor);
		}
	}
});
