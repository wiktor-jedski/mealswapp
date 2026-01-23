# LocalStorageCache (client)

**Traceability:** ARCH-011

## 1. Data Structures & Types

```typescript
interface CachedQueryResult {
  queryHash: string;
  queryText: string;
  timestamp: number;
  resultCount: number;
  resultIds: string[];
  metadata: {
    totalCalories?: number;
    totalProtein?: number;
    totalCarbs?: number;
    totalFat?: number;
  };
}

interface CacheEntry<T> {
  value: T;
  timestamp: number;
  accessCount: number;
}

interface SearchHistoryEntry {
  queryText: string;
  timestamp: number;
  resultCount: number;
}

interface LocalStorageCacheConfig {
  maxQueryResults: number;
  maxSearchHistory: number;
  maxCacheSizeBytes: number;
  defaultTTLMs: number;
}

const DEFAULT_CONFIG: LocalStorageCacheConfig = {
  maxQueryResults: 20,
  maxSearchHistory: 5,
  maxCacheSizeBytes: 4 * 1024 * 1024,
  defaultTTLMs: 24 * 60 * 60 * const1000
};

 CACHE_KEYS = {
  QUERY_RESULTS: 'cache:query_results',
  SEARCH_HISTORY: 'cache:search_history',
  LAST_SYNC: 'cache:last_sync',
  CACHE_METADATA: 'cache:metadata'
} as const;
```

## 2. Logic & Algorithms

### 2.1 LRU Eviction Algorithm

```
Algorithm: evictIfNeeded()
1. entries ← readAllEntries()
2. currentSize ← getCacheSize()
3. maxSize ← config.maxCacheSizeBytes

4. IF currentSize ≤ maxSize THEN
5.   RETURN
6. END IF

7. sortedEntries ← SORT entries BY (accessCount ASC, timestamp ASC)
8. entriesToEvict ← FIRST entries WHERE accumulatedSize > (currentSize - maxSize * 0.8)

9. FOR EACH entry IN entriesToEvict DO
10.   DELETE entry.key FROM localStorage
11.   UPDATE cacheMetadata SET size = cacheMetadata.size - entry.size
12. END FOR

13. UPDATE searchHistoryToMatchQueryResults()
```

### 2.2 Store Query Result Algorithm

```
Algorithm: storeQueryResult(query: string, resultIds: string[], metadata: object)
1. queryHash ← sha256(query)
2. existingEntry ← readEntry(queryHash)
3. timestamp ← Date.now()

4. newEntry ← {
5.   queryHash,
6.   queryText: query,
7.   timestamp,
8.   resultCount: resultIds.length,
9.   resultIds,
10.  metadata
11. }

12. serializedEntry ← JSON.stringify(newEntry)
13. entrySize ← byteLength(serializedEntry)

14. IF entrySize > config.maxCacheSizeBytes * 0.5 THEN
15.   RETURN error("Query result too large for cache")
16. END IF

17. IF existingEntry EXISTS THEN
18.   UPDATE existingEntry SET value = newEntry, timestamp, accessCount + 1
19. ELSE
20.   INSERT newEntry INTO queryResults
21. END IF

22. UPDATE cacheMetadata SET lastUpdate = timestamp
23. evictIfNeeded()
24. addToSearchHistory(query, resultIds.length)
```

### 2.3 Retrieve Query Result Algorithm

```
Algorithm: retrieveQueryResult(query: string): CachedQueryResult | null
1. queryHash ← sha256(query)
2. entry ← readEntry(queryHash)

3. IF entry NOT EXISTS THEN
4.   RETURN null
5. END IF

6. IF isExpired(entry.timestamp) THEN
7.   DELETE entry.key FROM localStorage
8.   RETURN null
9. END IF

10. UPDATE entry SET accessCount = accessCount + 1
11. RETURN entry.value
```

### 2.4 Add to Search History Algorithm

```
Algorithm: addToSearchHistory(query: string, resultCount: number)
1. history ← readSearchHistory()
2. existingIndex ← FIND history WHERE queryText = query

3. newEntry ← {
4.   queryText: query,
5.   timestamp: Date.now(),
6.   resultCount
7. }

8. IF existingIndex EXISTS THEN
9.   REMOVE history[existingIndex]
10. END IF

11. PREPEND newEntry TO history

12. IF length(history) > config.maxSearchHistory THEN
13.   REMOVE last element FROM history
14. END IF

15. writeSearchHistory(history)
```

### 2.5 Cache Invalidation Algorithm

```
Algorithm: invalidateByItemIds(itemIds: string[])
1. allEntries ← readAllQueryEntries()

2. FOR EACH entry IN allEntries DO
3.   entryContainsStaleItem ← ANY itemIds IN entry.value.resultIds

4.   IF entryContainsStaleItem THEN
5.     DELETE entry.key FROM localStorage
6.   END IF
7. END FOR

8. UPDATE cacheMetadata SET lastInvalidation = Date.now()
```

### 2.6 GDPR User Purge Algorithm

```
Algorithm: purgeUserData(userId: string)
1. CLEAR queryResults FROM localStorage
2. CLEAR searchHistory FROM localStorage
3. CLEAR lastSync FROM localStorage
4. CLEAR cacheMetadata FROM localStorage

5. dispatchEvent('userCachePurged', { userId, timestamp: Date.now() })
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Condition | Error Code | Handling Strategy |
| :--- | :--- | :--- |
| localStorage quota exceeded | `QUOTA_EXCEEDED` | Trigger LRU eviction, retry operation |
| Cache entry too large | `ENTRY_TOO_LARGE` | Skip caching, log warning |
| Corrupted JSON data | `PARSE_ERROR` | Delete corrupted entry, continue |
| localStorage unavailable | `STORAGE_UNAVAILABLE` | Graceful degradation, log error |
| Entry expired | `ENTRY_EXPIRED` | Return null, delete entry on read |
| Encryption failure | `ENCRYPTION_FAILED` | Skip caching, alert user |

### 3.2 State Transitions

```
Initial State: UNINITIALIZED
  → Initialize cache metadata → READY

READY State:
  → Store query → WRITING → READY (success) or ERROR (failure)
  → Retrieve query → READING → READY (with result) or READY (null)
  → Invalidate → UPDATING → READY
  → Purge user data → PURGING → READY

ERROR State:
  → Retry operation → READY (if recoverable) or stay in ERROR
  → Clear cache → READY
```

### 3.3 Recovery Strategies

```
Strategy: handleQuotaExceeded()
1. TRY
2.   evictIfNeeded()
3.   RETRY failed store operation
4. CATCH error
5.   IF cache still full THEN
6.     CLEAR all cached query results
7.     RETRY operation
8.   END IF
9.   RETURN error to caller
10. END TRY
```

## 4. Component Interfaces

```typescript
class LocalStorageCache {
  private config: LocalStorageCacheConfig;
  private initialized: boolean = false;

  constructor(config?: Partial<LocalStorageCacheConfig>);

  initialize(): Promise<void>;

  storeQueryResult(
    query: string,
    resultIds: string[],
    metadata?:>
  ): Promise Record<string, unknown<void>;

  retrieveQueryResult(query: string): Promise<CachedQueryResult | null>;

  getSearchHistory(): Promise<SearchHistoryEntry[]>;

  clearSearchHistory(): Promise<void>;

  invalidateByItemIds(itemIds: string[]): Promise<void>;

  purgeUserData(userId: string): Promise<void>;

  clearAll(): Promise<void>;

  getCacheStats(): Promise<{
    sizeBytes: number;
    entryCount: number;
    lastUpdate: number;
  }>;
}

function createLocalStorageCache(
  config?: Partial<LocalStorageCacheConfig>
): LocalStorageCache;

function sha256(str: string): string;

function isExpired(timestamp: number, ttlMs?: number): boolean;

function byteLength(str: string): number;
```

## 5. Integration Points

- **ARCH-008 (User Profile):** Receives `userCachePurged` event for GDPR compliance
- **Service Worker:** Coordinates with Service Worker for image cache invalidation
- **TanStack Query:** Provides caching layer for React Query cache keys
- **Search Component:** Stores and retrieves search query results for offline display
