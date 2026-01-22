# Detailed Design: LocalStorageManager

**Traceability:** [ARCH-001]

---

## 1. Data Structures & Types

### 1.1 Storage Key Constants

```typescript
const STORAGE_KEYS = {
  QUERY_CACHE: 'mealswapp_query_cache',
  SEARCH_HISTORY: 'mealswapp_search_history',
  THEME: 'mealswapp_theme',
  LAST_ONLINE: 'mealswapp_last_online',
  USER_PREFERENCES: 'mealswapp_preferences',
  CACHE_METADATA: 'mealswapp_cache_meta'
} as const;

type StorageKey = typeof STORAGE_KEYS[keyof typeof STORAGE_KEYS];
```

### 1.2 Query Cache Types

```typescript
interface CachedQueryResult {
  queryHash: string;              // SHA-256 hash of normalized query
  query: NormalizedQuery;         // Original query parameters for display
  results: FoodItemSummary[];     // Cached search results (summary data only)
  totalCount: number;             // Total results count from API
  timestamp: number;              // Unix timestamp when cached
  ttl: number;                    // Time-to-live in milliseconds
  version: number;                // Cache format version for migration
}

interface NormalizedQuery {
  searchTerm: string;             // Lowercase, trimmed search term
  searchMode: 'single' | 'multi'; // Search mode at time of query
  filters: QueryFilters;          // Applied filters (macros, categories)
  page: number;                   // Pagination page
  pageSize: number;               // Results per page
}

interface QueryFilters {
  macroToggles: MacroToggleState;
  categoryFilter: string | null;
  sortBy: SortOption;
}

interface MacroToggleState {
  calories: boolean;
  protein: boolean;
  carbs: boolean;
  fat: boolean;
}

type SortOption = 'relevance' | 'name' | 'calories_asc' | 'calories_desc';

interface FoodItemSummary {
  id: string;
  name: string;
  brand: string | null;
  calories: number;
  protein: number;
  carbs: number;
  fat: number;
  imageUrl: string | null;
  similarityScore: number | null;
}
```

### 1.3 Query Cache Container

```typescript
interface QueryCacheContainer {
  version: number;                        // Container format version
  entries: CachedQueryResult[];           // LRU-ordered list (most recent first)
  maxEntries: number;                     // Max entries (default: 20)
  totalSizeBytes: number;                 // Approximate size tracking
  lastCleanup: number;                    // Timestamp of last eviction
}

const DEFAULT_QUERY_CACHE: QueryCacheContainer = {
  version: 1,
  entries: [],
  maxEntries: 20,
  totalSizeBytes: 0,
  lastCleanup: 0
};
```

### 1.4 Search History Types

```typescript
interface SearchHistoryEntry {
  query: string;                  // Display string (user's original input)
  normalizedQuery: string;        // Lowercase, trimmed for deduplication
  timestamp: number;              // When search was performed
  resultCount: number;            // Number of results returned
}

interface SearchHistoryContainer {
  version: number;
  entries: SearchHistoryEntry[];
  maxEntries: number;             // Max entries (default: 5)
}

const DEFAULT_SEARCH_HISTORY: SearchHistoryContainer = {
  version: 1,
  entries: [],
  maxEntries: 5
};
```

### 1.5 User Preferences Types

```typescript
interface UserPreferences {
  version: number;
  searchMode: 'single' | 'multi';
  defaultMacroToggles: MacroToggleState;
  defaultSort: SortOption;
  resultsPerPage: number;
}

const DEFAULT_USER_PREFERENCES: UserPreferences = {
  version: 1,
  searchMode: 'single',
  defaultMacroToggles: {
    calories: true,
    protein: true,
    carbs: true,
    fat: true
  },
  defaultSort: 'relevance',
  resultsPerPage: 10
};
```

### 1.6 Cache Metadata Types

```typescript
interface CacheMetadata {
  version: number;
  totalStorageUsed: number;       // Estimated bytes used by Mealswapp
  storageQuota: number;           // Browser storage quota (if available)
  lastFullCleanup: number;        // Timestamp of last full cleanup
  errorCount: number;             // Consecutive storage errors
  degradedMode: boolean;          // True if storage is unreliable
}

const DEFAULT_CACHE_METADATA: CacheMetadata = {
  version: 1,
  totalStorageUsed: 0,
  storageQuota: 5 * 1024 * 1024,  // 5MB default
  lastFullCleanup: 0,
  errorCount: 0,
  degradedMode: false
};
```

### 1.7 Storage Operation Results

```typescript
type StorageResult<T> =
  | { success: true; data: T }
  | { success: false; error: StorageError };

interface StorageError {
  type: StorageErrorType;
  message: string;
  recoverable: boolean;
  originalError?: Error;
}

type StorageErrorType =
  | 'QUOTA_EXCEEDED'
  | 'SECURITY_ERROR'
  | 'PARSE_ERROR'
  | 'SERIALIZE_ERROR'
  | 'KEY_NOT_FOUND'
  | 'STORAGE_UNAVAILABLE'
  | 'VERSION_MISMATCH'
  | 'CORRUPTED_DATA';
```

### 1.8 Configuration Constants

```typescript
const CONFIG = {
  // Cache limits
  MAX_QUERY_CACHE_ENTRIES: 20,
  MAX_SEARCH_HISTORY_ENTRIES: 5,

  // TTL values (milliseconds)
  QUERY_CACHE_TTL: 30 * 60 * 1000,        // 30 minutes
  STALE_CACHE_TTL: 24 * 60 * 60 * 1000,   // 24 hours (offline fallback)

  // Storage limits
  MAX_STORAGE_BYTES: 4 * 1024 * 1024,     // 4MB (leave 1MB buffer)
  WARNING_THRESHOLD_BYTES: 3 * 1024 * 1024, // 3MB warning

  // Cleanup thresholds
  CLEANUP_INTERVAL: 60 * 60 * 1000,       // 1 hour
  ERROR_THRESHOLD: 3,                      // Errors before degraded mode

  // Format versions
  CURRENT_QUERY_CACHE_VERSION: 1,
  CURRENT_HISTORY_VERSION: 1,
  CURRENT_PREFERENCES_VERSION: 1
} as const;
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Initialization Flow

```
FUNCTION initializeLocalStorageManager(): StorageResult<void>
  1. Check storage availability
     1.1. TRY:
          testKey = '__mealswapp_test__'
          localStorage.setItem(testKey, 'test')
          localStorage.removeItem(testKey)
     1.2. CATCH (SecurityError):
          RETURN { success: false, error: {
            type: 'SECURITY_ERROR',
            message: 'localStorage access denied (private browsing?)',
            recoverable: false
          }}
     1.3. CATCH (Error):
          RETURN { success: false, error: {
            type: 'STORAGE_UNAVAILABLE',
            message: 'localStorage not available',
            recoverable: false
          }}

  2. Load and validate cache metadata
     2.1. metadata = loadCacheMetadata()
     2.2. IF metadata.errorCount >= CONFIG.ERROR_THRESHOLD:
          Log warning: 'Storage in degraded mode'
          metadata.degradedMode = true

  3. Perform startup cleanup if needed
     3.1. IF Date.now() - metadata.lastFullCleanup > CONFIG.CLEANUP_INTERVAL:
          CALL performFullCleanup()

  4. Estimate current storage usage
     4.1. CALL updateStorageEstimate()

  5. RETURN { success: true, data: undefined }
```

### 2.2 Query Hash Generation

```
FUNCTION generateQueryHash(query: NormalizedQuery): string
  1. Create canonical representation
     canonical = {
       term: query.searchTerm.toLowerCase().trim(),
       mode: query.searchMode,
       filters: {
         macros: sortedMacroKeys(query.filters.macroToggles),
         category: query.filters.categoryFilter || '',
         sort: query.filters.sortBy
       },
       page: query.page,
       pageSize: query.pageSize
     }

  2. Serialize to deterministic JSON
     jsonString = JSON.stringify(canonical, Object.keys(canonical).sort())

  3. Generate hash (simple hash for localStorage key)
     // Use FNV-1a for fast hashing (no crypto needed for cache keys)
     hash = fnv1aHash(jsonString)

  4. RETURN hash as hex string
```

### 2.3 Query Cache Operations

#### 2.3.1 Get Cached Query

```
FUNCTION getCachedQuery(query: NormalizedQuery): StorageResult<CachedQueryResult | null>
  1. Generate query hash
     hash = generateQueryHash(query)

  2. Load cache container
     containerResult = loadQueryCacheContainer()
     IF NOT containerResult.success:
       RETURN containerResult

  3. Find matching entry
     container = containerResult.data
     entry = container.entries.find(e => e.queryHash === hash)

  4. IF entry not found:
     RETURN { success: true, data: null }

  5. Check TTL validity
     age = Date.now() - entry.timestamp
     isOnline = navigator.onLine

     5.1. IF isOnline AND age > entry.ttl:
          // Entry expired, remove from cache
          CALL removeQueryFromCache(hash)
          RETURN { success: true, data: null }

     5.2. IF NOT isOnline AND age > CONFIG.STALE_CACHE_TTL:
          // Even stale cache has expired
          RETURN { success: true, data: null }

  6. Move entry to front (LRU update)
     CALL touchQueryCacheEntry(hash)

  7. RETURN { success: true, data: entry }
```

#### 2.3.2 Store Query Result

```
FUNCTION storeQueryResult(
  query: NormalizedQuery,
  results: FoodItemSummary[],
  totalCount: number
): StorageResult<void>
  1. Check degraded mode
     metadata = loadCacheMetadata()
     IF metadata.degradedMode:
       Log warning: 'Storage in degraded mode, skipping cache write'
       RETURN { success: true, data: undefined }

  2. Generate query hash
     hash = generateQueryHash(query)

  3. Create cache entry
     entry: CachedQueryResult = {
       queryHash: hash,
       query: query,
       results: results,
       totalCount: totalCount,
       timestamp: Date.now(),
       ttl: CONFIG.QUERY_CACHE_TTL,
       version: CONFIG.CURRENT_QUERY_CACHE_VERSION
     }

  4. Estimate entry size
     entrySize = estimateSize(entry)

  5. Load current container
     containerResult = loadQueryCacheContainer()
     container = containerResult.success
       ? containerResult.data
       : DEFAULT_QUERY_CACHE

  6. Check if entry already exists (update case)
     existingIndex = container.entries.findIndex(e => e.queryHash === hash)
     IF existingIndex >= 0:
       // Remove existing entry (will be re-added at front)
       oldEntry = container.entries.splice(existingIndex, 1)[0]
       container.totalSizeBytes -= estimateSize(oldEntry)

  7. Ensure capacity
     WHILE container.entries.length >= container.maxEntries:
       // Remove oldest entry (LRU eviction)
       evicted = container.entries.pop()
       container.totalSizeBytes -= estimateSize(evicted)

  8. Check storage quota
     newTotalSize = container.totalSizeBytes + entrySize
     IF newTotalSize > CONFIG.MAX_STORAGE_BYTES:
       // Need to evict more entries
       WHILE newTotalSize > CONFIG.WARNING_THRESHOLD_BYTES AND container.entries.length > 0:
         evicted = container.entries.pop()
         newTotalSize -= estimateSize(evicted)
         container.totalSizeBytes -= estimateSize(evicted)

  9. Add new entry at front (most recent)
     container.entries.unshift(entry)
     container.totalSizeBytes = newTotalSize

  10. Save container
      saveResult = saveQueryCacheContainer(container)
      IF NOT saveResult.success:
        CALL handleStorageError(saveResult.error)
        RETURN saveResult

  11. RETURN { success: true, data: undefined }
```

#### 2.3.3 Touch Cache Entry (LRU Update)

```
FUNCTION touchQueryCacheEntry(queryHash: string): void
  1. Load container
     containerResult = loadQueryCacheContainer()
     IF NOT containerResult.success:
       RETURN
     container = containerResult.data

  2. Find and move entry to front
     index = container.entries.findIndex(e => e.queryHash === queryHash)
     IF index > 0:
       entry = container.entries.splice(index, 1)[0]
       container.entries.unshift(entry)
       saveQueryCacheContainer(container)
```

#### 2.3.4 Remove Query from Cache

```
FUNCTION removeQueryFromCache(queryHash: string): StorageResult<boolean>
  1. Load container
     containerResult = loadQueryCacheContainer()
     IF NOT containerResult.success:
       RETURN { success: false, error: containerResult.error }
     container = containerResult.data

  2. Find and remove entry
     index = container.entries.findIndex(e => e.queryHash === queryHash)
     IF index < 0:
       RETURN { success: true, data: false }

  3. Remove entry and update size
     removed = container.entries.splice(index, 1)[0]
     container.totalSizeBytes -= estimateSize(removed)

  4. Save container
     saveResult = saveQueryCacheContainer(container)
     RETURN { success: true, data: saveResult.success }
```

### 2.4 Search History Operations

#### 2.4.1 Add to Search History

```
FUNCTION addToSearchHistory(
  query: string,
  resultCount: number
): StorageResult<void>
  1. Normalize query for deduplication
     normalizedQuery = query.toLowerCase().trim()

  2. Skip empty queries
     IF normalizedQuery.length === 0:
       RETURN { success: true, data: undefined }

  3. Load history container
     containerResult = loadSearchHistoryContainer()
     container = containerResult.success
       ? containerResult.data
       : DEFAULT_SEARCH_HISTORY

  4. Remove duplicate if exists
     container.entries = container.entries.filter(
       e => e.normalizedQuery !== normalizedQuery
     )

  5. Create new entry
     entry: SearchHistoryEntry = {
       query: query,                // Preserve original casing for display
       normalizedQuery: normalizedQuery,
       timestamp: Date.now(),
       resultCount: resultCount
     }

  6. Add at front (most recent)
     container.entries.unshift(entry)

  7. Enforce max entries
     IF container.entries.length > container.maxEntries:
       container.entries = container.entries.slice(0, container.maxEntries)

  8. Save container
     RETURN saveSearchHistoryContainer(container)
```

#### 2.4.2 Get Search History

```
FUNCTION getSearchHistory(): StorageResult<SearchHistoryEntry[]>
  1. Load container
     containerResult = loadSearchHistoryContainer()
     IF NOT containerResult.success:
       RETURN { success: true, data: [] }  // Return empty on error

  2. RETURN { success: true, data: containerResult.data.entries }
```

#### 2.4.3 Clear Search History

```
FUNCTION clearSearchHistory(): StorageResult<void>
  1. Reset to default container
     RETURN saveSearchHistoryContainer(DEFAULT_SEARCH_HISTORY)
```

#### 2.4.4 Remove Single History Entry

```
FUNCTION removeFromSearchHistory(normalizedQuery: string): StorageResult<boolean>
  1. Load container
     containerResult = loadSearchHistoryContainer()
     IF NOT containerResult.success:
       RETURN { success: false, error: containerResult.error }

  2. Filter out matching entry
     container = containerResult.data
     originalLength = container.entries.length
     container.entries = container.entries.filter(
       e => e.normalizedQuery !== normalizedQuery
     )

  3. Save if changed
     IF container.entries.length < originalLength:
       saveResult = saveSearchHistoryContainer(container)
       RETURN { success: saveResult.success, data: true }

  4. RETURN { success: true, data: false }
```

### 2.5 User Preferences Operations

#### 2.5.1 Get Preferences

```
FUNCTION getUserPreferences(): StorageResult<UserPreferences>
  1. TRY:
       raw = localStorage.getItem(STORAGE_KEYS.USER_PREFERENCES)
       IF raw === null:
         RETURN { success: true, data: DEFAULT_USER_PREFERENCES }

       parsed = JSON.parse(raw)
       IF NOT isValidPreferences(parsed):
         Log warning: 'Invalid preferences, using defaults'
         RETURN { success: true, data: DEFAULT_USER_PREFERENCES }

       // Merge with defaults for forward compatibility
       merged = { ...DEFAULT_USER_PREFERENCES, ...parsed }
       RETURN { success: true, data: merged }
  2. CATCH (error):
       RETURN { success: true, data: DEFAULT_USER_PREFERENCES }
```

#### 2.5.2 Update Preferences

```
FUNCTION updateUserPreferences(
  updates: Partial<UserPreferences>
): StorageResult<UserPreferences>
  1. Load current preferences
     currentResult = getUserPreferences()
     current = currentResult.success
       ? currentResult.data
       : DEFAULT_USER_PREFERENCES

  2. Merge updates
     updated = { ...current, ...updates }

  3. Validate merged result
     IF NOT isValidPreferences(updated):
       RETURN { success: false, error: {
         type: 'SERIALIZE_ERROR',
         message: 'Invalid preferences update',
         recoverable: true
       }}

  4. Save to storage
     TRY:
       localStorage.setItem(STORAGE_KEYS.USER_PREFERENCES, JSON.stringify(updated))
       RETURN { success: true, data: updated }
     CATCH (error):
       RETURN handleWriteError(error)
```

### 2.6 Storage Utilities

#### 2.6.1 Size Estimation

```
FUNCTION estimateSize(data: unknown): number
  1. Serialize to JSON
     jsonString = JSON.stringify(data)

  2. Calculate byte size (UTF-16 in localStorage)
     // Each character is 2 bytes in UTF-16
     byteSize = jsonString.length * 2

  3. RETURN byteSize
```

#### 2.6.2 Storage Quota Check

```
FUNCTION checkStorageQuota(): StorageResult<{ used: number; available: number }>
  1. TRY using Storage API (modern browsers):
     IF navigator.storage AND navigator.storage.estimate:
       estimate = await navigator.storage.estimate()
       RETURN {
         success: true,
         data: {
           used: estimate.usage || 0,
           available: estimate.quota || CONFIG.MAX_STORAGE_BYTES
         }
       }

  2. Fallback: estimate from key sizes
     totalSize = 0
     FOR key IN Object.values(STORAGE_KEYS):
       value = localStorage.getItem(key)
       IF value:
         totalSize += (key.length + value.length) * 2

     RETURN {
       success: true,
       data: {
         used: totalSize,
         available: CONFIG.MAX_STORAGE_BYTES
       }
     }
```

#### 2.6.3 Full Cleanup

```
FUNCTION performFullCleanup(): StorageResult<number>
  1. Track bytes freed
     bytesFreed = 0

  2. Clean expired query cache entries
     containerResult = loadQueryCacheContainer()
     IF containerResult.success:
       container = containerResult.data
       now = Date.now()

       container.entries = container.entries.filter(entry => {
         age = now - entry.timestamp
         maxAge = navigator.onLine ? entry.ttl : CONFIG.STALE_CACHE_TTL

         IF age > maxAge:
           bytesFreed += estimateSize(entry)
           RETURN false
         RETURN true
       })

       container.lastCleanup = now
       saveQueryCacheContainer(container)

  3. Update metadata
     metadata = loadCacheMetadata()
     metadata.lastFullCleanup = now
     metadata.totalStorageUsed -= bytesFreed
     saveCacheMetadata(metadata)

  4. RETURN { success: true, data: bytesFreed }
```

#### 2.6.4 Emergency Cleanup

```
FUNCTION emergencyCleanup(): StorageResult<number>
  // Called when quota is exceeded

  1. Log warning
     Log warning: 'Emergency storage cleanup initiated'

  2. Clear entire query cache (preserve history and preferences)
     bytesFreed = 0

     cacheRaw = localStorage.getItem(STORAGE_KEYS.QUERY_CACHE)
     IF cacheRaw:
       bytesFreed = cacheRaw.length * 2
       localStorage.removeItem(STORAGE_KEYS.QUERY_CACHE)

  3. Reset query cache to empty
     localStorage.setItem(
       STORAGE_KEYS.QUERY_CACHE,
       JSON.stringify(DEFAULT_QUERY_CACHE)
     )

  4. Update metadata
     metadata = loadCacheMetadata()
     metadata.totalStorageUsed -= bytesFreed
     metadata.lastFullCleanup = Date.now()
     saveCacheMetadata(metadata)

  5. RETURN { success: true, data: bytesFreed }
```

### 2.7 Error Handling

#### 2.7.1 Handle Storage Error

```
FUNCTION handleStorageError(error: StorageError): void
  1. Log error
     Log error: `Storage error: ${error.type} - ${error.message}`

  2. Update error count
     metadata = loadCacheMetadata()
     metadata.errorCount += 1

  3. Check for degraded mode trigger
     IF metadata.errorCount >= CONFIG.ERROR_THRESHOLD:
       metadata.degradedMode = true
       Log warning: 'Entering degraded storage mode'

  4. Handle specific error types
     SWITCH error.type:
       CASE 'QUOTA_EXCEEDED':
         CALL emergencyCleanup()
         metadata.errorCount = 0  // Reset after cleanup

       CASE 'SECURITY_ERROR':
         metadata.degradedMode = true

       CASE 'CORRUPTED_DATA':
         // Try to recover by clearing corrupted key
         CALL clearCorruptedData()

  5. Save metadata (best effort)
     TRY:
       saveCacheMetadata(metadata)
     CATCH:
       // Cannot save metadata, storage is seriously broken
```

#### 2.7.2 Handle Write Error

```
FUNCTION handleWriteError(error: Error): StorageResult<never>
  1. Detect error type
     IF error.name === 'QuotaExceededError' OR
        error.message.includes('quota'):
       CALL emergencyCleanup()
       RETURN {
         success: false,
         error: {
           type: 'QUOTA_EXCEEDED',
           message: 'Storage quota exceeded',
           recoverable: true,
           originalError: error
         }
       }

     IF error.name === 'SecurityError':
       RETURN {
         success: false,
         error: {
           type: 'SECURITY_ERROR',
           message: 'Storage access denied',
           recoverable: false,
           originalError: error
         }
       }

  2. Generic error
     RETURN {
       success: false,
       error: {
         type: 'STORAGE_UNAVAILABLE',
         message: error.message,
         recoverable: false,
         originalError: error
       }
     }
```

### 2.8 Data Migration

```
FUNCTION migrateStorageFormat(): void
  // Called during initialization to handle version upgrades

  1. Check query cache version
     containerResult = loadQueryCacheContainer()
     IF containerResult.success:
       container = containerResult.data
       IF container.version < CONFIG.CURRENT_QUERY_CACHE_VERSION:
         migratedContainer = migrateQueryCache(container)
         saveQueryCacheContainer(migratedContainer)

  2. Check history version
     historyResult = loadSearchHistoryContainer()
     IF historyResult.success:
       history = historyResult.data
       IF history.version < CONFIG.CURRENT_HISTORY_VERSION:
         migratedHistory = migrateHistory(history)
         saveSearchHistoryContainer(migratedHistory)

  3. Check preferences version
     prefsResult = getUserPreferences()
     IF prefsResult.success:
       prefs = prefsResult.data
       IF prefs.version < CONFIG.CURRENT_PREFERENCES_VERSION:
         migratedPrefs = migratePreferences(prefs)
         updateUserPreferences(migratedPrefs)
```

### 2.9 Clear All Mealswapp Data

```
FUNCTION clearAllMealswappData(): StorageResult<void>
  // For GDPR data deletion or user-initiated clear

  1. Remove all Mealswapp keys
     FOR key IN Object.values(STORAGE_KEYS):
       TRY:
         localStorage.removeItem(key)
       CATCH (error):
         Log warning: `Failed to remove key ${key}: ${error.message}`

  2. Dispatch event for other components
     window.dispatchEvent(new CustomEvent('mealswapp:storagecleared'))

  3. RETURN { success: true, data: undefined }
```

---

## 3. State Management & Error Handling

### 3.1 State Diagram

```
                    ┌─────────────────────────────────────┐
                    │           INITIALIZING              │
                    │  (Test access, load metadata,       │
                    │   perform startup cleanup)          │
                    └──────────────────┬──────────────────┘
                                       │
                      ┌────────────────┴────────────────┐
                      │                                 │
                      ▼                                 ▼
             ┌────────────────┐                ┌────────────────┐
             │    AVAILABLE   │                │  UNAVAILABLE   │
             │                │                │                │
             │ Normal ops     │                │ All ops return │
             │ Cache enabled  │                │ safe defaults  │
             └───────┬────────┘                └────────────────┘
                     │
       ┌─────────────┴─────────────┐
       │                           │
       ▼                           ▼
┌──────────────┐          ┌──────────────┐
│   HEALTHY    │          │   DEGRADED   │
│              │          │              │
│ Full caching │          │ Read-only    │
│ All features │          │ No writes    │
└──────┬───────┘          └──────┬───────┘
       │                          │
       │ 3+ consecutive errors    │ Successful write
       └─────────────────────────►└──────────────────►
```

### 3.2 Cache Entry Lifecycle

```
┌─────────────────┐
│    CREATED      │
│ (storeQuery)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│     FRESH       │
│ age < TTL       │◄─────────────┐
└────────┬────────┘              │
         │                       │ Access (LRU touch)
         │ time passes           │
         ▼                       │
┌─────────────────┐              │
│     STALE       │──────────────┘
│ TTL < age < 24h │
│ (offline only)  │
└────────┬────────┘
         │
         │ age > 24h OR
         │ online + age > TTL
         ▼
┌─────────────────┐
│    EXPIRED      │
│ (removed on     │
│  next access)   │
└─────────────────┘
```

### 3.3 Error States

| Error State | Trigger | User Impact | Recovery Action |
|:------------|:--------|:------------|:----------------|
| **QUOTA_EXCEEDED** | localStorage full | Cache write fails | Emergency cleanup, retry write |
| **SECURITY_ERROR** | Private browsing, iframe sandbox | All storage unavailable | Enter degraded mode, use in-memory fallback |
| **PARSE_ERROR** | Corrupted JSON data | Specific key unusable | Clear corrupted key, use default |
| **SERIALIZE_ERROR** | Object cannot be stringified | Write fails | Skip caching for this entry |
| **KEY_NOT_FOUND** | Expected key missing | Read returns null | Return default value |
| **VERSION_MISMATCH** | Old cache format | Data incompatible | Migrate or clear data |
| **CORRUPTED_DATA** | Invalid structure in valid JSON | Key unusable | Clear and reinitialize key |

### 3.4 Graceful Degradation Strategy

| Scenario | Degraded Behavior | Core Functionality |
|:---------|:------------------|:-------------------|
| **localStorage unavailable** | All reads return defaults, writes are no-ops | App works without caching/history |
| **Quota exceeded (persistent)** | Query cache disabled, preferences preserved | Search works, no cached results |
| **Corrupted single key** | That feature uses defaults | Other storage features work |
| **High error rate** | Read-only mode (no writes) | Existing cache still serves reads |

### 3.5 Error Handling Decision Tree

```
ON Storage Operation Error:
  │
  ├─► Is it a QuotaExceededError?
  │     │
  │     ├─► YES: Perform emergency cleanup
  │     │         Retry operation once
  │     │         If still fails → enter degraded mode
  │     │
  │     └─► NO: Continue
  │
  ├─► Is it a SecurityError?
  │     │
  │     └─► YES: Enter degraded mode immediately
  │               Log warning to console
  │               Set degradedMode = true
  │
  ├─► Is it a parse/corruption error?
  │     │
  │     └─► YES: Clear affected key
  │               Reinitialize with default
  │               Reset error count for this key
  │
  └─► Otherwise:
        Increment error count
        If errorCount >= 3 → enter degraded mode
        Return safe default value
```

---

## 4. Component Interfaces

### 4.1 LocalStorageManager Class

```typescript
class LocalStorageManager {
  private static instance: LocalStorageManager | null = null;
  private metadata: CacheMetadata;
  private isInitialized: boolean = false;

  // Singleton access
  static getInstance(): LocalStorageManager;

  // Initialization
  initialize(): Promise<StorageResult<void>>;
  isAvailable(): boolean;
  isDegraded(): boolean;

  // Query Cache
  getCachedQuery(query: NormalizedQuery): StorageResult<CachedQueryResult | null>;
  storeQueryResult(
    query: NormalizedQuery,
    results: FoodItemSummary[],
    totalCount: number
  ): StorageResult<void>;
  invalidateQueryCache(): StorageResult<void>;
  getQueryCacheStats(): StorageResult<QueryCacheStats>;

  // Search History
  getSearchHistory(): StorageResult<SearchHistoryEntry[]>;
  addToSearchHistory(query: string, resultCount: number): StorageResult<void>;
  removeFromSearchHistory(query: string): StorageResult<boolean>;
  clearSearchHistory(): StorageResult<void>;

  // User Preferences
  getUserPreferences(): StorageResult<UserPreferences>;
  updateUserPreferences(updates: Partial<UserPreferences>): StorageResult<UserPreferences>;
  resetUserPreferences(): StorageResult<void>;

  // Storage Management
  getStorageUsage(): StorageResult<StorageUsage>;
  performCleanup(): StorageResult<number>;
  clearAllData(): StorageResult<void>;
}
```

### 4.2 Query Cache Stats Interface

```typescript
interface QueryCacheStats {
  entryCount: number;
  maxEntries: number;
  totalSizeBytes: number;
  oldestEntryAge: number;     // Milliseconds
  newestEntryAge: number;     // Milliseconds
  hitRate: number;            // 0-1 (if tracking enabled)
}
```

### 4.3 Storage Usage Interface

```typescript
interface StorageUsage {
  totalUsed: number;          // Bytes used by Mealswapp
  quota: number;              // Available quota (estimated)
  percentUsed: number;        // 0-100
  breakdown: {
    queryCache: number;
    searchHistory: number;
    preferences: number;
    metadata: number;
  };
}
```

### 4.4 React Hook: useLocalStorage

```typescript
interface UseLocalStorageOptions<T> {
  key: StorageKey;
  defaultValue: T;
  serialize?: (value: T) => string;
  deserialize?: (raw: string) => T;
}

function useLocalStorage<T>(
  options: UseLocalStorageOptions<T>
): [T, (value: T | ((prev: T) => T)) => void, () => void];

// Returns: [currentValue, setValue, clearValue]
```

### 4.5 React Hook: useQueryCache

```typescript
interface UseQueryCacheResult {
  getCached: (query: NormalizedQuery) => CachedQueryResult | null;
  setCached: (
    query: NormalizedQuery,
    results: FoodItemSummary[],
    totalCount: number
  ) => void;
  invalidate: () => void;
  stats: QueryCacheStats | null;
}

function useQueryCache(): UseQueryCacheResult;
```

### 4.6 React Hook: useSearchHistory

```typescript
interface UseSearchHistoryResult {
  history: SearchHistoryEntry[];
  addEntry: (query: string, resultCount: number) => void;
  removeEntry: (query: string) => void;
  clearHistory: () => void;
}

function useSearchHistory(): UseSearchHistoryResult;
```

### 4.7 Utility Functions (Exported)

```typescript
/**
 * Generate deterministic hash for query caching.
 */
function generateQueryHash(query: NormalizedQuery): string;

/**
 * Normalize search query for deduplication.
 */
function normalizeSearchQuery(query: string): string;

/**
 * Estimate byte size of data when stored in localStorage.
 */
function estimateStorageSize(data: unknown): number;

/**
 * Check if localStorage is available and writable.
 */
function isLocalStorageAvailable(): boolean;

/**
 * Format storage size for display.
 * @example formatStorageSize(1536000) => "1.5 MB"
 */
function formatStorageSize(bytes: number): string;
```

### 4.8 Event Types (for non-React listeners)

```typescript
interface StorageChangeEventDetail {
  key: StorageKey;
  action: 'set' | 'remove' | 'clear';
  oldValue?: unknown;
  newValue?: unknown;
}

// Usage: window.addEventListener('mealswapp:storagechange', handler)
type StorageChangeEvent = CustomEvent<StorageChangeEventDetail>;

interface StorageClearedEventDetail {
  reason: 'user_initiated' | 'gdpr' | 'emergency_cleanup';
  bytesFreed: number;
}

type StorageClearedEvent = CustomEvent<StorageClearedEventDetail>;
```

---

## 5. Integration Requirements

### 5.1 Application Initialization

```typescript
// main.tsx or App.tsx
import { LocalStorageManager } from './services/LocalStorageManager';

async function initializeApp() {
  const storage = LocalStorageManager.getInstance();
  const result = await storage.initialize();

  if (!result.success) {
    console.warn('Storage initialization failed:', result.error);
    // App continues with degraded storage (safe defaults)
  }

  // Continue with app render
  render(<App />);
}
```

### 5.2 SearchView Integration

```typescript
// SearchView.tsx
function SearchView() {
  const { getCached, setCached } = useQueryCache();
  const { addEntry } = useSearchHistory();
  const { isOffline } = useNetwork();

  async function handleSearch(query: NormalizedQuery) {
    // Check cache first
    const cached = getCached(query);
    if (cached) {
      // Use cached results
      setResults(cached.results);
      setTotalCount(cached.totalCount);
      return;
    }

    if (isOffline) {
      // Cannot fetch, no cache available
      setError('No cached results available offline');
      return;
    }

    // Fetch from API
    const response = await searchApi.search(query);
    setResults(response.results);
    setTotalCount(response.totalCount);

    // Cache results
    setCached(query, response.results, response.totalCount);

    // Add to history
    addEntry(query.searchTerm, response.totalCount);
  }

  // ...
}
```

### 5.3 AutocompleteDropdown Integration

```typescript
// AutocompleteDropdown.tsx
function AutocompleteDropdown({ inputValue, onSelect }) {
  const { history } = useSearchHistory();

  // Filter history based on current input
  const suggestions = useMemo(() => {
    if (!inputValue) return history;

    const normalized = inputValue.toLowerCase().trim();
    return history.filter(entry =>
      entry.normalizedQuery.includes(normalized)
    );
  }, [history, inputValue]);

  return (
    <ul>
      {suggestions.map(entry => (
        <li key={entry.normalizedQuery} onClick={() => onSelect(entry.query)}>
          {entry.query}
          <span className="result-count">{entry.resultCount} results</span>
        </li>
      ))}
    </ul>
  );
}
```

### 5.4 SettingsPanel Integration

```typescript
// SettingsPanel.tsx
function SettingsPanel() {
  const storage = LocalStorageManager.getInstance();
  const [usage, setUsage] = useState<StorageUsage | null>(null);

  useEffect(() => {
    const result = storage.getStorageUsage();
    if (result.success) {
      setUsage(result.data);
    }
  }, []);

  async function handleClearCache() {
    const result = await storage.invalidateQueryCache();
    if (result.success) {
      showToast('Cache cleared');
      refreshUsage();
    }
  }

  async function handleClearHistory() {
    const result = await storage.clearSearchHistory();
    if (result.success) {
      showToast('Search history cleared');
    }
  }

  async function handleClearAllData() {
    if (confirm('This will clear all cached data and preferences. Continue?')) {
      await storage.clearAllData();
      showToast('All data cleared');
      window.location.reload();
    }
  }

  return (
    <section>
      <h2>Storage</h2>

      {usage && (
        <div className="storage-usage">
          <ProgressBar value={usage.percentUsed} max={100} />
          <span>{formatStorageSize(usage.totalUsed)} of {formatStorageSize(usage.quota)}</span>
        </div>
      )}

      <button onClick={handleClearCache}>Clear Search Cache</button>
      <button onClick={handleClearHistory}>Clear Search History</button>
      <button onClick={handleClearAllData}>Clear All Data</button>
    </section>
  );
}
```

### 5.5 Service Worker Coordination

```typescript
// When Service Worker caches API responses, coordinate with LocalStorageManager

// In main thread:
navigator.serviceWorker.addEventListener('message', (event) => {
  if (event.data.type === 'CACHE_UPDATED') {
    // Service Worker cached a new API response
    // LocalStorageManager doesn't need to duplicate this
    const storage = LocalStorageManager.getInstance();
    storage.markQueryAsServiceWorkerCached(event.data.queryHash);
  }
});
```

---

## 6. Performance Considerations

| Optimization | Implementation | Impact |
|:-------------|:---------------|:-------|
| **Singleton pattern** | Single instance manages all storage | No redundant reads/parsing |
| **Lazy loading** | Cache containers loaded on first access | Faster app startup |
| **Batch writes** | Debounce rapid successive writes | Reduce serialization overhead |
| **Size estimation** | Track sizes without re-serializing | Fast quota checks |
| **LRU efficiency** | Array shift/unshift O(n) acceptable for 20 items | Simple, fast enough |
| **Hash caching** | Memoize query hashes during session | Avoid repeated hashing |
| **Incremental cleanup** | Clean expired entries on access | Spread cleanup cost |

### 6.1 Size Optimization Strategies

```typescript
// Minimize stored data size

// DON'T store full food item details:
interface BadCacheEntry {
  results: FullFoodItem[];  // Includes all details, descriptions, etc.
}

// DO store summaries only:
interface GoodCacheEntry {
  results: FoodItemSummary[];  // Only fields needed for display
}

// Summary size: ~200 bytes per item
// 20 queries × 10 results × 200 bytes = 40KB (well under 5MB limit)
```

---

## 7. Security Considerations

| Concern | Mitigation |
|:--------|:-----------|
| **XSS data theft** | localStorage is same-origin only; CSP headers prevent XSS |
| **Sensitive data exposure** | Never store auth tokens, passwords, or PII in localStorage |
| **Data tampering** | Version checks detect format corruption; invalid data cleared |
| **Quota exhaustion attack** | Max storage limit enforced; emergency cleanup available |
| **Private browsing detection** | Graceful degradation; app works without storage |

### 7.1 Data Stored (Privacy Review)

| Data Type | Contains PII? | Retention | User Control |
|:----------|:--------------|:----------|:-------------|
| Query cache | No (food data only) | 30min-24h TTL | Clear via settings |
| Search history | Potentially (search terms) | Until cleared | Clear via settings |
| Theme preference | No | Indefinite | Change anytime |
| User preferences | No | Indefinite | Reset via settings |

---

## 8. Testing Requirements

### 8.1 Unit Test Cases

| Test Case | Input | Expected Output |
|:----------|:------|:----------------|
| Initialize with available storage | Normal browser | `{ success: true }` |
| Initialize in private browsing | SecurityError on access | `{ success: false, error.type: 'SECURITY_ERROR' }` |
| Store and retrieve query | Valid query + results | Same results returned |
| Query cache LRU eviction | Store 21 queries | Oldest query evicted |
| Query cache TTL expiration | Cached 31 min ago, online | Returns null, entry removed |
| Query cache stale serving | Cached 2h ago, offline | Returns stale data |
| Search history deduplication | Add same query twice | Single entry, updated timestamp |
| Search history max entries | Add 6 entries | Only 5 most recent kept |
| Quota exceeded handling | Simulate QuotaExceededError | Emergency cleanup triggered |
| Corrupted data handling | Invalid JSON in storage | Default value returned, key cleared |
| generateQueryHash determinism | Same query twice | Same hash both times |
| generateQueryHash uniqueness | Different queries | Different hashes |

### 8.2 Integration Test Cases

| Test Case | Scenario | Expected Behavior |
|:----------|:---------|:------------------|
| Cache persists across refresh | Store query, refresh page | Cached query available |
| History persists across refresh | Add search, refresh page | History entry present |
| Preferences persist | Change setting, refresh | Setting retained |
| Offline cache serving | Go offline, search cached query | Results returned |
| Storage quota warning | Fill to 3MB | Warning logged |
| Multi-tab consistency | Update in tab A | Tab B sees update (on next read) |
| Migration on version bump | Old format data | Migrated to new format |

### 8.3 Edge Case Test Cases

| Test Case | Scenario | Expected Behavior |
|:----------|:---------|:------------------|
| Empty localStorage | First-time user | All defaults work |
| Partially corrupted storage | One key invalid | Other keys work, invalid key reset |
| Concurrent writes | Rapid fire cache updates | No data corruption |
| Very large result set | 100 items in response | Truncated to fit quota |
| Unicode search terms | Japanese/emoji queries | Properly stored and retrieved |
| Browser storage disabled | User disabled localStorage | Graceful degradation |

---

## Changelog

### 2026-01-22 (Rev 1.0)

**Added:**
- Initial detailed design document for LocalStorageManager
- Query cache with LRU eviction (20 entries max)
- Search history storage (5 entries max)
- User preferences persistence
- Storage quota management with emergency cleanup
- Graceful degradation for unavailable/unreliable storage
- TypeScript interfaces for all data structures
- React hooks: useLocalStorage, useQueryCache, useSearchHistory
- Integration examples with SearchView, AutocompleteDropdown, SettingsPanel
- Comprehensive error handling with typed results
- Data migration framework for version upgrades
- Security and privacy considerations
- Performance optimization strategies
- Full test case specifications

**Design Decisions:**
- Singleton pattern chosen for centralized storage management
- LRU eviction preferred over FIFO for better cache hit rates
- 30-minute TTL balances freshness with offline utility
- 4MB soft limit leaves buffer for browser overhead
- Emergency cleanup preserves preferences over cache
- Query hash uses FNV-1a (fast, non-cryptographic) since security not required for cache keys
