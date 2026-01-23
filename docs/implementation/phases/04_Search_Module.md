## Phase 4: Search Module

**Goal:** Implement full search functionality with autocomplete and filtering

### Components & Static Aspects

#### ARCH-002 - Search Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **SearchController** | HTTP handlers for search endpoints | `search/controller.go` |
| **AutocompleteRanker** | Three-tier priority: exact match, Levenshtein, string length | `search/ranker.go` |
| **QueryParser** | Parse search terms, extract filters | `search/parser.go` |
| **FilterProcessor** | Apply tag whitelist/blacklist, prep time filters | `search/filter.go` |
| **PaginationHandler** | Max 10 results per page, offset/limit handling | `search/pagination.go` |
| **FunctionalityTagWeighter** | Boost score for matching functionality tags | `search/tag_weighter.go` |

#### ARCH-011 (partial) - Caching Layer (Server)
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **RedisCache** | Redis client wrapper (github.com/redis/go-redis/v9) | `cache/redis_cache.go` |
| **CacheInvalidator** | Invalidate cache on data updates | `cache/invalidator.go` |

### API Endpoints
| Method | Endpoint | Handler |
|:-------|:---------|:--------|
| GET | `/api/v1/search` | `SearchController.Search` |
| GET | `/api/v1/autocomplete` | `SearchController.Autocomplete` |
| POST | `/api/v1/search/similarity` | `SearchController.SimilaritySearch` |
| GET | `/api/v1/items/{id}` | `SearchController.GetItem` |
| GET | `/api/v1/items/{id}/similar` | `SearchController.GetSimilarItems` |

### Testing
- [ ] Autocomplete ranking: exact > Levenshtein > length
- [ ] Autocomplete executes in < 100ms
- [ ] Pagination returns max 10 items
- [ ] Results sorted by cosine similarity descending
- [ ] Tag whitelist/blacklist filtering
- [ ] Prep time filtering
- [ ] Functionality tag weighting boosts relevant results
- [ ] Implicit search trigger (empty bar + 2 ingredients)
- [ ] Redis cache hit returns <10ms
- [ ] Cache invalidation on data updates

---

