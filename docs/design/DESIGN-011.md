## FILE: DESIGN-011.md
**Traceability:** ARCH-011

**Static aspects covered:** ServiceWorkerCache, LocalStorageCache, RedisCache, CacheInvalidator, LRUEvictionPolicy, UserCachePurger.

### 0. Static Aspect Responsibilities
- `ServiceWorkerCache`: owns Cache API interception, offline image serving, and cached API response serving.
- `LocalStorageCache`: owns structured recent query metadata and search history persistence.
- `RedisCache`: owns server-side hot data, session, similarity, and job-result caching.
- `CacheInvalidator`: owns tag/key invalidation after admin data changes.
- `LRUEvictionPolicy`: owns bounded local query cache and recent-history eviction.
- `UserCachePurger`: owns GDPR-driven user key deletion, session invalidation, and history cache purge.

### 1. Data Structures & Types
- `interface CacheEntry<T> { key: string; value: T; storedAt: string; ttlSeconds: number; staleAt?: string; tags: string[] }`
- `interface RedisCacheKey { namespace: "search" | "item" | "similarity" | "session" | "job" | "user"; id: string; version: string }`
- `interface LocalQueryCache { recentQueries: CachedQuery[]; maxEntries: 20; history: string[]; maxHistory: 5 }`
- `interface CacheInvalidationEvent { tags: string[]; itemIds?: UUID[]; userId?: UUID; reason: string; createdAt: time.Time }`
- `interface PurgeResult { redisKeysDeleted: number; localKeysDeleted: number; sessionsInvalidated: number; errors: string[] }`

### 2. Logic & Algorithms (Step-by-Step)
1. Client registers service worker during ARCH-001 startup.
2. Service worker intercepts image and cacheable API GET requests.
3. Images use cache-first behavior with Cache-Control freshness checks.
4. Query metadata and 20 most recent unique query responses are stored in localStorage with LRU eviction.
5. Server-side Redis stores hot search responses, item records, similarity calculations, session data, and LP job results.
6. Cache keys include namespace, stable hash, and schema version to avoid stale shape collisions.
7. Admin item updates publish invalidation events for affected item IDs and tags.
8. Redis invalidator deletes matching keys; client receives purge instruction when available and removes stale image/API entries.
9. Account deletion calls `UserCachePurger` to delete user-prefixed Redis keys, invalidate sessions, and clear search history cache.

### 3. State Management & Error Handling
- `fresh_hit`: return cached value.
- `stale_hit`: return cached value with stale marker only when offline or degradation policy allows it.
- `miss`: caller proceeds to source of truth.
- `redis_down`: fail fast and let caller fall back to PostgreSQL when supported.
- `local_storage_full`: evict LRU entries and retry once.
- `cache_api_unavailable`: continue without image offline cache.
- `invalidation_pending`: event accepted but not fully processed.
- `purge_partial`: return deleted counts plus errors for retry and monitoring.

### 4. Component Interfaces
- `func GetRedis[T any](ctx context.Context, key RedisCacheKey) (T, bool, error)`
- `func SetRedis[T any](ctx context.Context, key RedisCacheKey, value T, ttl time.Duration, tags []string) error`
- `func DeleteByTags(ctx context.Context, tags []string) (int, error)`
- `func BuildSearchCacheKey(req SearchRequest) RedisCacheKey`
- `func PublishInvalidation(ctx context.Context, event CacheInvalidationEvent) error`
- `func PurgeUserCache(ctx context.Context, userID UUID) (PurgeResult, error)`
- `function readLocalQueryCache(key: string): CachedQuery | null`
- `function writeLocalQueryCache(entry: CachedQuery): void`
- `function evictLocalLRU(maxEntries: number): void`
