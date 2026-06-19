import { describe, expect, test } from "bun:test";
import type { SearchRequest, SearchResponse } from "../api/generated";
import { searchRequestKey } from "../search/search-state";
import { SearchLRUCache, type KeyValueStorage } from "./search-lru";

class MemoryStorage implements KeyValueStorage {
  values = new Map<string, string>();
  getItem(key: string) { return this.values.get(key) ?? null; }
  setItem(key: string, value: string) { this.values.set(key, value); }
  removeItem(key: string) { this.values.delete(key); }
}

const request = (query: string): SearchRequest => ({ query, mode: "catalog", filters: [], page: 1 });
const response = (query: string): SearchResponse => ({ items: [], totalCount: 0, page: 1, similarityScores: [], similarityMetadata: [], warnings: [query] });

// Implements DESIGN-001 LocalStorageManager cache verification.
describe("SearchLRUCache", () => {
  test("equivalent requests share a key and reads refresh recency", () => {
    const storage = new MemoryStorage();
    const cache = new SearchLRUCache({ storage, maxEntries: 2 });
    cache.set(request(" Apple "), response("apple"));
    cache.set(request("banana"), response("banana"));
    expect(cache.get(request("apple"))?.response.warnings).toEqual(["apple"]);
    cache.set(request("carrot"), response("carrot"));
    expect(cache.get(request("banana"))).toBeNull();
    expect(cache.get(request("APPLE"))?.response.warnings).toEqual(["apple"]);
  });

  test("the twenty-first unique entry evicts the least recent", () => {
    const cache = new SearchLRUCache({ storage: new MemoryStorage() });
    for (let index = 0; index < 21; index += 1) cache.set(request(`food-${index}`), response(`food-${index}`));
    expect(cache.size()).toBe(20);
    expect(cache.get(request("food-0"))).toBeNull();
    expect(cache.get(request("food-20"))).not.toBeNull();
  });

  test("malformed and version-mismatched persisted data is ignored", () => {
    const malformed = new MemoryStorage();
    malformed.setItem("mealswapp.search-cache.v1", "not-json");
    expect(new SearchLRUCache({ storage: malformed }).size()).toBe(0);

    const wrongVersion = new MemoryStorage();
    wrongVersion.setItem("mealswapp.search-cache.v1", JSON.stringify({ version: 99, entries: [] }));
    expect(new SearchLRUCache({ storage: wrongVersion }).size()).toBe(0);
    expect(wrongVersion.getItem("mealswapp.search-cache.v1")).toBeNull();
  });

  test("rejects persisted requests with malformed nested fields", () => {
    const malformedRequests = [
      { query: "apple", mode: "catalog", page: 1 },
      { ...request("apple"), filters: [{ filterId: "fruit", kind: "unknown", include: true }] },
      { ...request("apple"), substitutionInputs: [{ foodObjectId: "apple", quantity: "100", unit: "g" }] }
    ];
    for (const persistedRequest of malformedRequests) {
      expect(restoredSize(persistedRequest, response("apple"))).toBe(0);
    }
  });

  test("rejects persisted responses with incomplete or malformed generated fields", () => {
    const validItem = {
      id: "apple", name: "Apple", physicalState: "solid", classifications: [], primaryFoodCategory: null,
      macros: { protein: 0.3, carbohydrate: 14, fat: 0.2, basis: "100g" }, calories: 52
    };
    const malformedResponses = [
      { ...response("apple"), items: [{ id: "apple", name: "Apple" }] },
      { ...response("apple"), items: [{ ...validItem, classifications: [{ id: "fruit", name: "Fruit" }] }] },
      { ...response("apple"), items: [{ ...validItem, macros: { ...validItem.macros, protein: "0.3" } }] },
      { ...response("apple"), similarityMetadata: [{ itemId: "apple", score: 0.9, tier: "excellent" }] },
      { ...response("apple"), cache: { status: "hit", namespace: "search", schemaVersion: "v1", ttlSeconds: -1 } }
    ];
    for (const persistedResponse of malformedResponses) {
      expect(restoredSize(request("apple"), persistedResponse)).toBe(0);
    }
  });

  test("reports stale timestamps deterministically", () => {
    let now = new Date("2026-01-01T00:00:00.000Z");
    const cache = new SearchLRUCache({ storage: new MemoryStorage(), staleAfterMs: 1000, now: () => now });
    cache.set(request("apple"), response("apple"));
    expect(cache.get(request("apple"))?.stale).toBe(false);
    now = new Date("2026-01-01T00:00:01.000Z");
    expect(cache.get(request("apple"))?.stale).toBe(true);
  });

  test("reloads schema-valid persisted entries", () => {
    const storage = new MemoryStorage();
    new SearchLRUCache({ storage }).set(request("apple"), response("apple"));
    const restored = new SearchLRUCache({ storage });
    expect(restored.get(request("apple"))?.response).toEqual(response("apple"));
  });

  test("default browser storage detection is safe without browser globals", () => {
    expect(new SearchLRUCache().size()).toBe(0);
  });

  test("storage failures retain an online-usable in-memory cache", () => {
    const throwing: KeyValueStorage = {
      getItem: () => { throw new Error("denied"); },
      setItem: () => { throw new Error("full"); },
      removeItem: () => { throw new Error("denied"); }
    };
    const cache = new SearchLRUCache({ storage: throwing });
    cache.set(request("apple"), response("apple"));
    expect(cache.get(request("apple"))?.response.warnings).toEqual(["apple"]);
  });
});

function restoredSize(persistedRequest: unknown, persistedResponse: unknown): number {
  const storage = new MemoryStorage();
  const key = searchRequestKey(persistedRequest as SearchRequest);
  storage.setItem("mealswapp.search-cache.v1", JSON.stringify({
    version: 1,
    entries: [{ key, request: persistedRequest, response: persistedResponse, storedAt: "2026-01-01T00:00:00.000Z", staleAt: "2026-01-01T00:05:00.000Z" }]
  }));
  return new SearchLRUCache({ storage }).size();
}
