# [ARCH-011] - Caching Layer

**Description:** Multi-tier caching system using client-side Service Worker with Cache API, localStorage for metadata, and server-side Redis to optimize performance and enable full offline functionality including images.

| Attribute | Value |
| :--- | :--- |
| **Type** | Middleware |
| **Static Aspects** | ServiceWorkerCache (client), LocalStorageCache (client), RedisCache (server), CacheInvalidator, LRUEvictionPolicy, UserCachePurger |
| **Dependencies** | Redis, Browser Service Worker API, Browser localStorage API, ARCH-008 (User Profile) |
| **Traceability** | SW-REQ-003, SW-REQ-048, SW-REQ-073, SW-REQ-080, SW-REQ-088 |

**Dynamic Behavior:**

- **Service Worker Registration:** On first load, registers Service Worker to intercept network requests and manage Cache API storage.
- **Image Caching:** Service Worker caches all food item images referenced in search results. Cache-first strategy serves images offline. Respects Cache-Control headers for freshness.
- **Query Result Cache:** localStorage stores 20 most recent unique queries with JSON result metadata (LRU eviction). Stores 5 recent search queries for history display.
- **Server Cache:** Redis caches frequently accessed food items, similarity calculations, session data, and LP job results.
- **Cache Invalidation:** Admin data updates trigger cache invalidation for affected items across Redis. Service Worker receives push notification to purge stale image URLs.
- **User Data Purge (GDPR):** On account deletion (SW-REQ-073), ARCH-008 triggers `UserCachePurger` which: (1) Deletes all Redis keys prefixed with user ID, (2) Invalidates user session tokens, (3) Clears server-side search history cache for user.
- **Offline Serving:** Service Worker serves cached images and API responses when offline. Displays staleness indicator and "offline mode" banner.

**Interface Definition:**

- `Input`: Cache keys (query hashes, item IDs, user IDs), TTL configurations, deletion events
- `Output`: Cached data, cache miss signals, purge confirmations

**Alternative Analysis (BP6):**

- *Chosen Approach:* Three-tier caching (Service Worker + localStorage + Redis) with GDPR-aware purging
- *Alternative Considered:* localStorage-only client caching without Service Worker
- *Trade-off:* localStorage has a 5MB limit and cannot cache binary assets (images). SW-REQ-088 requires displaying "cached search results" offline, and SW-REQ-011 mandates images in results. Without Service Worker, offline mode would show broken image links, degrading UX. Service Worker enables full offline visual experience while localStorage handles structured query data within its size constraints.

**Reference Documentation:** 
- 02_APPENDIX_A.md
