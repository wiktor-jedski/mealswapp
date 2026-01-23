## Phase 8: Administration & External Integration

**Goal:** Admin panel and external data import

### Components & Static Aspects

#### ARCH-009 - Administration Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **AdminController** | HTTP handlers for admin endpoints | `admin/controller.go` |
| **DataImporter** | Save curated items from external sources | `admin/importer.go` |
| **ItemCurator** | Edit fields (name, tags, macros) before import | `admin/curator.go` |
| **TagManager** | CRUD for global category/functionality tags | `admin/tag_manager.go` |
| **UserAdminPanel** | View/manage user accounts | `admin/user_panel.go` |
| **ExternalSearchProxy** | Proxy external API searches for admin UI | `admin/search_proxy.go` |

#### ARCH-012 - External Data Integration
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **USDAClient** | USDA FoodData Central API client | `external/usda_client.go` |
| **OpenFoodFactsClient** | OpenFoodFacts API client | `external/openfoodfacts_client.go` |
| **DataNormalizer** | Convert external formats to internal schema | `external/normalizer.go` |
| **RateLimitHandler** | Respect external API rate limits | `external/rate_limit_handler.go` |

#### ARCH-011 (complete) - Caching Layer
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **UserCachePurger** | Delete all Redis keys for user (GDPR) | `cache/user_purger.go` |
| **LRUEvictionPolicy** | LRU eviction for query cache | `cache/lru_policy.go` |

#### ARCH-014 (complete) - Logging & Monitoring Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **LogAggregator** | Aggregate logs to GCP Cloud Monitoring | `monitoring/aggregator.go` |
| **MetricsCollector** | Track response times, error rates, concurrent users | `monitoring/metrics.go` |
| **AlertManager** | Trigger alerts at P95 > 1.5s or 2s | `monitoring/alerts.go` |
| **UptimeMonitor** | Synthetic health checks every 30s | `monitoring/uptime.go` |

### Testing
- [ ] Admin endpoints require 'Admin' role (403 otherwise)
- [ ] External API search returns normalized results
- [ ] Rate limiting respects external API limits
- [ ] Item import saves to local database
- [ ] Tag CRUD operations (create, update, delete)
- [ ] Cache invalidation triggers on item updates
- [ ] User cache purge removes all Redis keys for user
- [ ] Metrics collection to GCP Cloud Monitoring
- [ ] Alert triggers at P95 > 1.5s (warning) and > 2s (critical)
- [ ] Uptime monitor reports 99.9% availability target

---

