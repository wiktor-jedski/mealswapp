import type { SearchRequest, SearchResponse } from "../api/generated";
import { searchRequestKey } from "../search/search-state";

const CACHE_SCHEMA_VERSION = 1;
const DEFAULT_STORAGE_KEY = "mealswapp.search-cache.v1";
const DEFAULT_MAX_ENTRIES = 20;
const DEFAULT_STALE_AFTER_MS = 5 * 60 * 1000;

export interface KeyValueStorage {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
}

interface CacheEntry {
  key: string;
  request: SearchRequest;
  response: SearchResponse;
  storedAt: string;
  staleAt: string;
}

interface PersistedCache {
  version: number;
  entries: CacheEntry[];
}

export interface CachedSearchResult {
  response: SearchResponse;
  storedAt: string;
  stale: boolean;
}

export interface SearchLRUOptions {
  storage?: KeyValueStorage | null;
  storageKey?: string;
  maxEntries?: number;
  staleAfterMs?: number;
  now?: () => Date;
}

// Implements DESIGN-001 LocalStorageManager versioned 20-entry search cache.
export class SearchLRUCache {
  private entries: CacheEntry[] = [];
  private readonly storage: KeyValueStorage | null;
  private readonly storageKey: string;
  private readonly maxEntries: number;
  private readonly staleAfterMs: number;
  private readonly now: () => Date;

  constructor(options: SearchLRUOptions = {}) {
    this.storage = options.storage === undefined ? browserStorage() : options.storage;
    this.storageKey = options.storageKey ?? DEFAULT_STORAGE_KEY;
    this.maxEntries = options.maxEntries ?? DEFAULT_MAX_ENTRIES;
    this.staleAfterMs = options.staleAfterMs ?? DEFAULT_STALE_AFTER_MS;
    this.now = options.now ?? (() => new Date());
    this.entries = this.load();
  }

  get(request: SearchRequest): CachedSearchResult | null {
    const key = searchRequestKey(request);
    const index = this.entries.findIndex((entry) => entry.key === key);
    if (index < 0) return null;
    const [entry] = this.entries.splice(index, 1);
    this.entries.push(entry);
    this.persist();
    return { response: entry.response, storedAt: entry.storedAt, stale: this.now().getTime() >= Date.parse(entry.staleAt) };
  }

  set(request: SearchRequest, response: SearchResponse): void {
    const key = searchRequestKey(request);
    this.entries = this.entries.filter((entry) => entry.key !== key);
    const stored = this.now();
    this.entries.push({
      key,
      request,
      response,
      storedAt: stored.toISOString(),
      staleAt: new Date(stored.getTime() + this.staleAfterMs).toISOString()
    });
    if (this.entries.length > this.maxEntries) this.entries.splice(0, this.entries.length - this.maxEntries);
    this.persist();
  }

  size(): number {
    return this.entries.length;
  }

  private load(): CacheEntry[] {
    if (!this.storage) return [];
    try {
      const raw = this.storage.getItem(this.storageKey);
      if (!raw) return [];
      const parsed: unknown = JSON.parse(raw);
      if (!isPersistedCache(parsed)) {
        this.storage.removeItem(this.storageKey);
        return [];
      }
      return parsed.entries.slice(-this.maxEntries);
    } catch {
      return [];
    }
  }

  private persist(): void {
    if (!this.storage) return;
    try {
      this.storage.setItem(this.storageKey, JSON.stringify({ version: CACHE_SCHEMA_VERSION, entries: this.entries } satisfies PersistedCache));
    } catch {
      // In-memory entries remain usable when localStorage is denied or full.
    }
  }
}

function browserStorage(): KeyValueStorage | null {
  try {
    return typeof window === "undefined" ? null : window.localStorage;
  } catch {
    return null;
  }
}

function isPersistedCache(value: unknown): value is PersistedCache {
  if (!isRecord(value) || value.version !== CACHE_SCHEMA_VERSION || !Array.isArray(value.entries)) return false;
  return value.entries.every(isCacheEntry);
}

function isCacheEntry(value: unknown): value is CacheEntry {
  return isRecord(value) && typeof value.key === "string" && isSearchRequest(value.request) && isSearchResponse(value.response) &&
    typeof value.storedAt === "string" && !Number.isNaN(Date.parse(value.storedAt)) &&
    typeof value.staleAt === "string" && !Number.isNaN(Date.parse(value.staleAt)) && value.key === searchRequestKey(value.request);
}

function isSearchRequest(value: unknown): value is SearchRequest {
  return isRecord(value) && typeof value.query === "string" &&
    ["catalog", "substitution", "daily_diet_alternative"].includes(String(value.mode)) &&
    Array.isArray(value.filters) && value.filters.every(isSearchFilter) &&
    Number.isInteger(value.page) && Number(value.page) > 0 &&
    (value.substitutionInputs === undefined || (Array.isArray(value.substitutionInputs) && value.substitutionInputs.every(isSubstitutionInput))) &&
    (value.dailyDietId === undefined || typeof value.dailyDietId === "string");
}

function isSearchResponse(value: unknown): value is SearchResponse {
  return isRecord(value) && Array.isArray(value.items) && value.items.every(isFoodObject) &&
    Number.isInteger(value.totalCount) && Number(value.totalCount) >= 0 && Number.isInteger(value.page) && Number(value.page) > 0 &&
    isNumberArray(value.similarityScores) && Array.isArray(value.similarityMetadata) && value.similarityMetadata.every(isSimilarityMetadata) &&
    Array.isArray(value.warnings) && value.warnings.every((warning) => typeof warning === "string") &&
    (value.cache === undefined || isCacheMetadata(value.cache));
}

function isSearchFilter(value: unknown): boolean {
  return isRecord(value) && typeof value.filterId === "string" &&
    ["food_category", "culinary_role", "physical_state", "allergen", "dietary_preset"].includes(String(value.kind)) &&
    typeof value.include === "boolean";
}

function isSubstitutionInput(value: unknown): boolean {
  return isRecord(value) && typeof value.foodObjectId === "string" && typeof value.quantity === "number" && Number.isFinite(value.quantity) &&
    ["g", "ml", "oz", "fl_oz"].includes(String(value.unit));
}

function isClassification(value: unknown): boolean {
  return isRecord(value) && typeof value.id === "string" && typeof value.name === "string" &&
    ["food_category", "culinary_role"].includes(String(value.kind));
}

function isFoodObject(value: unknown): boolean {
  return isRecord(value) && typeof value.id === "string" && typeof value.name === "string" &&
    ["solid", "liquid"].includes(String(value.physicalState)) &&
    (value.imageUrl === undefined || value.imageUrl === null || typeof value.imageUrl === "string") &&
    Array.isArray(value.classifications) && value.classifications.every(isClassification) &&
    (value.primaryFoodCategory === null || isClassification(value.primaryFoodCategory)) &&
    isRecord(value.macros) && isFiniteNumber(value.macros.protein) && isFiniteNumber(value.macros.carbohydrate) &&
    isFiniteNumber(value.macros.fat) && ["100g", "100ml"].includes(String(value.macros.basis)) && isFiniteNumber(value.calories);
}

function isSimilarityMetadata(value: unknown): boolean {
  return isRecord(value) && typeof value.itemId === "string" && isFiniteNumber(value.score) &&
    ["excellent", "good", "fair", "poor"].includes(String(value.tier)) && typeof value.imageUrl === "string" &&
    isFiniteNumber(value.matchingQuantity);
}

function isCacheMetadata(value: unknown): boolean {
  return isRecord(value) && ["hit", "miss"].includes(String(value.status)) && typeof value.namespace === "string" &&
    typeof value.schemaVersion === "string" && Number.isInteger(value.ttlSeconds) && Number(value.ttlSeconds) >= 0;
}

function isNumberArray(value: unknown): boolean {
  return Array.isArray(value) && value.every(isFiniteNumber);
}

function isFiniteNumber(value: unknown): boolean {
  return typeof value === "number" && Number.isFinite(value);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
