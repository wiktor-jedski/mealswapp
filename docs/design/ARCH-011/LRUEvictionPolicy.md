# LRUEvictionPolicy

**Traceability:** ARCH-011

## 1. Data Structures & Types

### 1.1 Cache Entry
```typescript
interface CacheEntry<T> {
  key: string;
  value: T;
  timestamp: number;
  accessCount: number;
  lastAccessedAt: number;
  sizeBytes: number;
}
```

### 1.2 LRU Cache Configuration
```typescript
interface LRUCacheConfig {
  maxEntries: number;
  maxSizeBytes: number;
  ttlMs: number;
  evictionPercentage: number;
}
```

### 1.3 Eviction Result
```typescript
interface EvictionResult {
  evictedKeys: string[];
  freedBytes: number;
  remainingEntries: number;
  timestamp: number;
}
```

### 1.4 Cache Statistics
```typescript
interface CacheStatistics {
  totalEntries: number;
  usedBytes: number;
  hitRate: number;
  missRate: number;
  evictionCount: number;
  lastEvictionAt: number;
}
```

## 2. Logic & Algorithms

### 2.1 LRU Eviction Algorithm

**Pseudocode:**
```
FUNCTION evictLeastRecentlyUsed(cache: LRUCache, count: number): EvictionResult
  IF cache.entries.length <= 0 THEN
    RETURN { evictedKeys: [], freedBytes: 0, remainingEntries: 0, timestamp: NOW() }
  END IF

  entriesToEvict := MIN(count, cache.entries.length)
  entries := SORT(cache.entries BY lastAccessedAt ASC)
  evictedEntries := TAKE_FIRST(entries, entriesToEvict)
  freedBytes := SUM(entry.sizeBytes FOR entry IN evictedEntries)

  FOR EACH entry IN evictedEntries DO
    REMOVE entry.key FROM cache.entries
    REMOVE entry.key FROM cache.accessOrder
  END FOR

  RETURN {
    evictedKeys: MAP(entry.key FOR entry IN evictedEntries),
    freedBytes: freedBytes,
    remainingEntries: cache.entries.length,
    timestamp: NOW()
  }
END FUNCTION
```

### 2.2 Access Pattern Tracking

**Pseudocode:**
```
FUNCTION recordAccess(cache: LRUCache, key: string): void
  entry := cache.entries.get(key)
  IF entry IS null THEN
    RETURN
  END IF

  entry.lastAccessedAt := NOW()
  entry.accessCount := entry.accessCount + 1
  REMOVE key FROM cache.accessOrder
  PUSH key TO cache.accessOrder
END FUNCTION
```

### 2.3 Cache Entry Insertion with Overflow Handling

**Pseudocode:**
```
FUNCTION insertWithEviction[T](cache: LRUCache, entry: CacheEntry[T]): EvictionResult
  key := entry.key

  IF cache.entries.has(key) THEN
    REMOVE key FROM cache.entries
    REMOVE key FROM cache.accessOrder
  END IF

  currentSizeBytes := cache.usedBytes + entry.sizeBytes
  entriesNeeded := 1
  IF cache.entries.length >= cache.config.maxEntries THEN
    entriesNeeded := 1
  END IF

  WHILE cache.entries.length + entriesNeeded > cache.config.maxEntries OR
        currentSizeBytes > cache.config.maxSizeBytes DO
    evictionResult := evictLeastRecentlyUsed(cache, cache.config.evictionPercentage)
    currentSizeBytes := currentSizeBytes - evictionResult.freedBytes
    IF evictionResult.evictedKeys.length == 0 THEN
      BREAK
    END IF
  END WHILE

  entry.timestamp := NOW()
  entry.lastAccessedAt := NOW()
  entry.accessCount := 0

  SET cache.entries[key] := entry
  PUSH key TO cache.accessOrder
  cache.usedBytes := cache.usedBytes + entry.sizeBytes

  RETURN evictionResult
END FUNCTION
```

### 2.4 Size-Based Eviction Trigger

**Pseudocode:**
```
FUNCTION checkAndEvictBySize(cache: LRUCache): EvictionResult
  IF cache.usedBytes <= cache.config.maxSizeBytes THEN
    RETURN { evictedKeys: [], freedBytes: 0, remainingEntries: cache.entries.length, timestamp: NOW() }
  END IF

  targetSize := cache.config.maxSizeBytes * 0.8
  freedBytes := 0
  evictedKeys := []

  WHILE cache.usedBytes > targetSize DO
    result := evictLeastRecentlyUsed(cache, 10)
    freedBytes := freedBytes + result.freedBytes
    APPEND result.evictedKeys TO evictedKeys
    IF cache.entries.length == 0 THEN
      BREAK
    END IF
  END WHILE

  RETURN {
    evictedKeys: evictedKeys,
    freedBytes: freedBytes,
    remainingEntries: cache.entries.length,
    timestamp: NOW()
  }
END FUNCTION
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error State | Condition | Handling Strategy |
| :--- | :--- | :--- |
| `CACHE_FULL` | `entries.length >= maxEntries` AND `evictionPercentage == 0` | Throw error, prevent insertion |
| `SIZE_EXCEEDED` | `usedBytes > maxSizeBytes` AND no entries can be evicted | Log warning, reject entry |
| `INVALID_KEY` | Key is null, undefined, or empty string | Throw validation error |
| `STORAGE_QUOTA_EXCEEDED` | localStorage quota exceeded | Catch exception, trigger cleanup, retry once |
| `ENTRY_NOT_FOUND` | Attempt to access non-existent key | Return null, log debug message |
| `CORRUPTED_CACHE` | Checksum mismatch on deserialization | Clear cache, log error |
| `CONCURRENT_MODIFICATION` | Cache modified during iteration | Retry operation with lock |

### 3.2 State Transitions

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            INITIAL STATE                                     │
│                              entries: {}                                     │
│                              usedBytes: 0                                    │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         OPERATION: INSERT                                    │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │ Check size constraints                                                │   │
│  │     │                                                               │   │
│  │     ▼                                                               │   │
│  │ Check if key exists ──Yes──► Remove old entry                       │   │
│  │     │                                                               │   │
│  │     No                                                              │   │
│  │     │                                                               │   │
│  │     ▼                                                               │   │
│  │ Check capacity ──Full──► Trigger eviction (evictionPercentage)      │   │
│  │     │                    │                                          │   │
│  │     │                    ▼                                          │   │
│  │     │              Still full?                                      │   │
│  │     │                    │                                          │   │
│  │     │              Yes──►───Error: CACHE_FULL                       │   │
│  │     │                                                               │   │
│  │     No                                                               │   │
│  │     │                                                               │   │
│  │     ▼                                                               │   │
│  │ Insert entry, update metadata, record access                         │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         OPERATION: GET                                       │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │ Check if key exists ──No──► Return null, log miss                    │   │
│  │     │                                                               │   │
│  │     Yes                                                              │   │
│  │     │                                                               │   │
│  │     Check TTL (if configured)                                        │   │
│  │     │                                                               │   │
│  │     Expired?                                                         │   │
│  │     │     │                                                          │   │
│  │     Yes   No                                                         │   │
│  │     │     │                                                          │   │
│  │     ▼     ▼                                                          │   │
│  │  Delete  Return value, update access metadata                        │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                      OPERATION: CHECK SIZE                                   │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │ usedBytes > maxSizeBytes?                                            │   │
│  │     │                                                               │   │
│  │     Yes                                                              │   │
│  │     │                                                               │   │
│  │     Trigger eviction (target 80% capacity)                           │   │
│  │     │                                                               │   │
│  │     Still over limit?                                                │   │
│  │     │     │                                                          │   │
│  │     Yes   No                                                         │   │
│  │     │     │                                                          │   │
│  │     ▼     ▼                                                          │   │
│  │  Log    Done                                                         │   │
│  │  warning                                                             │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.3 Cache Corruption Recovery

```
FUNCTION recoverFromCorruption(cache: LRUCache): void
  TRY
    serializedData := localStorage.getItem(cache.name)
    IF serializedData IS null THEN
      RETURN
    END IF

    data := JSON.parse(serializedData)
    checksum := calculateChecksum(data.entries)

    IF checksum !== data.checksum THEN
      CLEAR cache.entries
      CLEAR cache.accessOrder
      cache.usedBytes := 0
      localStorage.removeItem(cache.name)
      logError("Cache corruption detected, cleared cache", { cacheName: cache.name })
    END IF
  CATCH error
    CLEAR cache.entries
    CLEAR cache.accessOrder
    cache.usedBytes := 0
    localStorage.removeItem(cache.name)
    logError("Cache recovery failed", { cacheName: cache.name, error: error.message })
  END TRY
END FUNCTION
```

## 4. Component Interfaces

### 4.1 Public Interface

```typescript
interface LRUEvictionPolicy {
  insert<T>(key: string, value: T, sizeBytes: number): EvictionResult;
  get<T>(key: string): CacheEntry<T> | null;
  remove(key: string): boolean;
  clear(): void;
  evict(count: number): EvictionResult;
  evictBySize(maxBytes: number): EvictionResult;
  getStatistics(): CacheStatistics;
  get(key: string): CacheEntry<T> | null;
}
```

### 4.2 Configuration Factory

```typescript
function createQueryResultCacheConfig(): LRUCacheConfig {
  return {
    maxEntries: 20,
    maxSizeBytes: 5 * 1024 * 1024,
    ttlMs: 24 * 60 * 60 * 1000,
    evictionPercentage: 25
  };
}

function createRedisCacheConfig(): LRUCacheConfig {
  return {
    maxEntries: 10000,
    maxSizeBytes: 100 * 1024 * 1024,
    ttlMs: 60 * 60 * 1000,
    evictionPercentage: 10
  };
}
```

### 4.3 Usage Example (Client-Side)

```typescript
const cacheConfig = createQueryResultCacheConfig();
const cache = new LRUCache<SearchResult>(cacheConfig);

// Insert search result
const entry: CacheEntry<SearchResult> = {
  key: hashQuery("pasta recipes"),
  value: searchResults,
  timestamp: Date.now(),
  accessCount: 0,
  lastAccessedAt: Date.now(),
  sizeBytes: calculateSize(searchResults)
};

const evictionResult = cache.insert(entry.key, entry.value, entry.sizeBytes);

// Retrieve cached result
const cached = cache.get(hashQuery("pasta recipes"));
if (cached) {
  recordAccess(cache, cached.key);
  return cached.value;
}
```

### 4.4 Server-Side Integration (Go)

```go
type LRUCache struct {
    mu    sync.RWMutex
    items map[string]*list.Element
    list  *list.List
    cap   int
    maxBytes int64
}

type cacheEntry struct {
    key       string
    value     interface{}
    sizeBytes int64
    accessAt  time.Time
}

func NewLRUCache(maxEntries int, maxSizeBytes int64) *LRUCache {
    return &LRUCache{
        items:    make(map[string]*list.Element),
        list:     list.New(),
        cap:      maxEntries,
        maxBytes: maxSizeBytes,
    }
}

func (c *LRUCache) Evict(count int) []string {
    c.mu.Lock()
    defer c.mu.Unlock()

    evicted := []string{}
    for i := 0; i < count && c.list.Len() > 0; i++ {
        elem := c.list.Back()
        c.list.Remove(elem)
        entry := elem.Value.(*cacheEntry)
        delete(c.items, entry.key)
        evicted = append(evicted, entry.key)
    }
    return evicted
}
```
