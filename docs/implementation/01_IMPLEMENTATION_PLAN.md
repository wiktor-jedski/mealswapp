# Mealswapp Implementation Plan

## Overview

This plan organizes the 17 architectural components (ARCH-001 to ARCH-017) into 8 implementation phases. Each phase is designed to be independently testable and builds upon previous phases.

**Tech Stack:** Go (Fiber) backend, Svelte frontend, PostgreSQL, Redis
**Target:** 1000 concurrent users, <2s P95 response time, 99.9% availability

---

## Phase 1: Foundation Infrastructure

**Goal:** Establish database schema, project structure, and core middleware

### Components & Static Aspects

#### ARCH-005 (partial) - Data Repository Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **FoodItemEntity** | Food item with macros, physical state, prep time, tags | `models/food_item.go` |
| **MealEntity** | Single dish or recipe reference | `models/meal.go` |
| **RecipeEntity** | Ingredient composition with quantities | `models/recipe.go` |
| **TagEntity** | Category and functionality tags | `models/tag.go` |
| **SimilarityIndicatorAsset** | Tier images/colors for similarity display | `models/similarity_indicator.go` |
| **RepositoryInterfaces** | Interface definitions for all repositories | `repository/interfaces.go` |

#### ARCH-013 (partial) - Security Middleware
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **EncryptionService** | AES-256 encryption via crypto/aes | `middleware/encryption.go` |
| **InputSanitizer** | XSS, SQL injection prevention | `middleware/sanitizer.go` |
| **TLSEnforcer** | TLS 1.3 enforcement, HTTP->HTTPS redirect | `middleware/tls.go` |

#### ARCH-014 (partial) - Logging & Monitoring Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **FiberLogger** | Fiber logger middleware integration | `middleware/logger.go` |
| **AuditLogger** | Structured audit logging for security events | `middleware/audit.go` |

### Testing
- [ ] Database migrations run successfully
- [ ] Connection pool handles 100+ concurrent connections
- [ ] Schema validation for all entity types
- [ ] Encryption service encrypts/decrypts correctly
- [ ] Input sanitizer blocks XSS/SQL injection patterns

---

## Phase 2: Authentication & User Management

**Goal:** Complete authentication system with email/password and OAuth

### Components & Static Aspects

#### ARCH-006 - Authentication Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **AuthController** | HTTP handlers for auth endpoints | `auth/controller.go` |
| **PasswordHasher** | Argon2 hashing with unique salts (golang.org/x/crypto/argon2) | `auth/password_hasher.go` |
| **JWTManager** | JWT issue, validate, refresh (15min access, 7-day refresh) | `auth/jwt_manager.go` |
| **OAuthHandler** | Google/Apple OAuth via github.com/markbates/goth | `auth/oauth_handler.go` |
| **SessionManager** | Fiber session middleware integration | `auth/session_manager.go` |
| **AccountLockoutTracker** | Track failed attempts, enforce 5-failure/15min lockout | `auth/lockout_tracker.go` |

#### ARCH-010 (partial) - API Gateway
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **RouteHandler** | Fiber route definitions and grouping | `routes/router.go` |
| **RateLimiter** | Fiber built-in limiter (10 failed/IP/10min) | `middleware/rate_limiter.go` |
| **SecurityHeaderMiddleware** | CSP, X-Frame-Options, X-Content-Type-Options, etc. | `middleware/headers.go` |
| **CSRFValidator** | Fiber csrf middleware for state-changing requests | `middleware/csrf.go` |
| **CORSHandler** | CORS configuration for allowed origins | `middleware/cors.go` |

#### ARCH-005 (partial) - Data Repository (Users)
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **UserEntity** | User model with email, password hash, role, verification status | `models/user.go` |
| **UserRepository** | User CRUD operations | `repository/user_repo.go` |

### Testing
- [ ] User registration with email validation
- [ ] Password hashing with unique salts (verify hash format)
- [ ] JWT token lifecycle (15min access, 7-day refresh)
- [ ] OAuth flow with Google/Apple mock
- [ ] Rate limiting: 10 failed attempts/IP/10min
- [ ] Account lockout: 5 failures -> 15min lock
- [ ] Security headers present on all responses
- [ ] CSRF tokens validate correctly
- [ ] Session timeout after 30min inactivity

---

## Phase 3: Similarity Engine & Data Repository Completion

**Goal:** Implement cosine similarity calculation and complete data layer

### Components & Static Aspects

#### ARCH-003 - Similarity Engine
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **CosineSimilarityCalculator** | Dot product of normalized P/C/F vectors | `similarity/calculator.go` |
| **MacroVectorNormalizer** | Normalize macros to unit vectors | `similarity/normalizer.go` |
| **ThresholdFilter** | Exclude results with score < 0.40 | `similarity/threshold_filter.go` |
| **SimilarityIndicatorMapper** | Map scores to tiers (excellent/good/fair/poor) | `similarity/indicator_mapper.go` |
| **SimilarityAssetResolver** | Resolve tier to color hex and image URL | `similarity/asset_resolver.go` |

#### ARCH-005 (complete) - Data Repository Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **UnitConverter** | Metric<->Imperial (g->oz: ×0.035, ml->fl oz: ×0.033) | `services/unit_converter.go` |
| **MacroNormalizer** | Normalize all values to per 100g/100ml | `services/macro_normalizer.go` |
| **FoodItemRepository** | FoodItem CRUD with tag associations | `repository/food_item_repo.go` |
| **MealRepository** | Meal CRUD with recipe aggregation | `repository/meal_repo.go` |
| **RecipeRepository** | Recipe ingredient CRUD | `repository/recipe_repo.go` |
| **TagRepository** | Tag CRUD operations | `repository/tag_repo.go` |

### Testing
- [ ] Cosine similarity math correctness (known test vectors)
- [ ] Threshold filter excludes scores < 0.40
- [ ] Visual indicator assignment by tier
- [ ] Recipe macro aggregation (sum of ingredients)
- [ ] Unit conversion accuracy (g->oz, ml->fl oz)
- [ ] Quantity-based macro scaling
- [ ] Matching quantity calculation for calorie/protein match

---

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

## Phase 5: Web Application & Theme System

**Goal:** Build the Svelte frontend with responsive UI

### Components & Static Aspects

#### ARCH-001 - Web Application Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **SearchView** | Main search input with mode toggles | `components/SearchView.svelte` |
| **SidebarComponent** | Collapsible sidebar with history/favorites | `components/SidebarComponent.svelte` |
| **ResultsGrid** | Display search results with images, macros, similarity | `components/ResultsGrid.svelte` |
| **AutocompleteDropdown** | Dropdown list of suggestions with keyboard nav | `components/AutocompleteDropdown.svelte` |
| **OfflineBanner** | Visual indicator when offline | `components/OfflineBanner.svelte` |
| **SettingsPanel** | User preferences (unit, theme) | `components/SettingsPanel.svelte` |
| **LocalStorageManager** | Manage localStorage for queries/history | `api/cache.ts` |

#### ARCH-016 - Theme & Style Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **ThemeProvider** | Svelte context for theme state | `theme/ThemeProvider.svelte` |
| **ColorPalette** | CSS custom property definitions | `theme/colors.ts` |
| **TypographySystem** | Font families, sizes, weights | `theme/typography.ts` |
| **LayoutGrid** | 12-column grid, responsive breakpoints | `theme/layout.ts` |
| **ComponentStyles** | Shared component style utilities | `theme/components.ts` |

#### ARCH-017 - Error Handling Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **ErrorBoundary** | Svelte error boundary component | `components/ErrorBoundary.svelte` |
| **RetryManager** | Automatic retry logic on connectivity restore | `api/retry.ts` |
| **ErrorMessageMapper** | Map technical errors to user-friendly messages | `api/error_mapper.ts` |

#### ARCH-011 (partial) - Caching Layer (Client)
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **ServiceWorkerCache** | Service Worker for offline caching | `static/sw.js` |
| **LocalStorageCache** | 20 recent queries, 5 search history | `api/cache.ts` |

### Testing
- [ ] Responsive layout: desktop (12-column), mobile (single column < 640px)
- [ ] Theme switching updates CSS variables
- [ ] System theme preference detection (prefers-color-scheme)
- [ ] Search debounce delays API calls by 150ms
- [ ] Keyboard navigation (Tab/Shift+Tab) through autocomplete
- [ ] Offline banner appears when disconnected
- [ ] Service Worker caches images and API responses
- [ ] localStorage stores 20 recent queries, 5 search history (LRU)
- [ ] WCAG 2.1 AA color contrast compliance (4.5:1 ratio)
- [ ] Error boundary catches component errors without full crash

---

## Phase 6: Subscriptions & User Profile

**Goal:** Implement payment processing and user data management

### Components & Static Aspects

#### ARCH-007 - Subscription Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **SubscriptionController** | HTTP handlers for subscription endpoints | `subscription/controller.go` |
| **StripeWebhookHandler** | Process payment_intent.succeeded/failed events | `subscription/webhook_handler.go` |
| **EntitlementManager** | Check/update user entitlement status | `subscription/entitlement_manager.go` |
| **TrialTracker** | 7-day trial activation and expiration | `subscription/trial_tracker.go` |
| **UsageLimiter** | 3 searches/24h for free tier | `subscription/usage_limiter.go` |

#### ARCH-008 - User Profile Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **ProfileController** | HTTP handlers for profile endpoints | `profile/controller.go` |
| **PreferenceManager** | Unit preferences, theme persistence | `profile/preference_manager.go` |
| **SavedDataRepository** | Saved items, diets, favorites | `profile/saved_data_repo.go` |
| **DataExporter** | Export user data to JSON/CSV | `profile/exporter.go` |
| **AccountDeleter** | Cascade delete all user data | `profile/deleter.go` |

#### ARCH-015 - Compliance Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **ConsentManager** | Track Privacy Policy/ToS consent | `compliance/consent_manager.go` |
| **DisclaimerRenderer** | Medical disclaimer content | `compliance/disclaimer_renderer.go` |
| **DataRetentionPolicy** | 30-day backup retention rules | `compliance/retention_policy.go` |
| **BackupManager** | Daily backup coordination | `compliance/backup_manager.go` |

### Testing
- [ ] Stripe Elements tokenization (no raw card data on server)
- [ ] Webhook idempotency (duplicate events ignored)
- [ ] Webhook signature verification (reject invalid signatures)
- [ ] Free tier enforces 3 searches/24h
- [ ] Paid features blocked for free users
- [ ] 7-day trial activates on first OAuth
- [ ] Trial auto-downgrades after 7 days
- [ ] Data export includes all PII, saved items, history
- [ ] Account deletion cascades to all related data
- [ ] Consent checkbox required for registration
- [ ] Medical disclaimer displayed on login

---

## Phase 7: Linear Programming Optimizer

**Goal:** Implement async diet optimization with job queue

### Components & Static Aspects

#### ARCH-004 - Linear Programming Optimizer
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **LPSolverWrapper** | Wrapper for go-coinor/clp solver | `optimizer/solver.go` |
| **ConstraintBuilder** | Build P, C, F constraints with tolerance bands | `optimizer/constraints.go` |
| **ObjectiveFunction** | Minimize total calories | `optimizer/objective.go` |
| **DiversityPenalizer** | Penalty weight for overlapping meal IDs | `optimizer/diversity.go` |
| **SolutionValidator** | Validate LP solutions meet all constraints | `optimizer/validator.go` |
| **JobQueueManager** | Redis-backed job queue (go-redis/queue or machinery) | `optimizer/job_manager.go` |
| **JobStatusTracker** | Track job status (queued/processing/completed/failed) | `optimizer/status_tracker.go` |

### Testing
- [ ] Job submission returns 202 immediately (non-blocking)
- [ ] Worker processes jobs from Redis queue
- [ ] LP constraints enforce P, C, F tolerance bands
- [ ] Calorie minimization objective works
- [ ] Diversity penalty reduces overlap with original diet
- [ ] Max 3 alternative combinations returned
- [ ] 30-second timeout terminates long jobs (marks as failed)
- [ ] Job status polling returns correct states
- [ ] WebSocket notifications fire on completion
- [ ] Results cached with 1-hour TTL

---

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

## Verification Checklist

After all phases complete, verify end-to-end:

### Functional
- [ ] User can register, login, and search
- [ ] Free tier limits enforced (3/24h, single-item only)
- [ ] Payment flow upgrades to paid tier
- [ ] Diet optimization returns up to 3 alternatives
- [ ] Admin can import and curate external data

### Performance
- [ ] Search P95 < 2 seconds
- [ ] Concurrent user load test (1000 users)
- [ ] Cache hit rate > 80% for repeat queries
- [ ] Autocomplete < 100ms

### Security
- [ ] OWASP Top 10 vulnerabilities addressed
- [ ] PCI-DSS compliance (Stripe tokenization)
- [ ] GDPR compliance (export, deletion, consent)
- [ ] All PII encrypted at rest (AES-256)

### Reliability
- [ ] Graceful degradation when Redis down
- [ ] LP job timeout and error handling
- [ ] Automatic retries for external APIs (3x with backoff)
- [ ] 99.9% uptime over 30-day period

---

## File Structure Summary

```
mealswapp/
├── backend/
│   ├── cmd/
│   │   ├── api/main.go
│   │   └── worker/main.go
│   └── internal/
│       ├── config/
│       ├── database/
│       │   ├── connection.go
│       │   └── migrations/
│       ├── middleware/
│       │   ├── logger.go
│       │   ├── security.go
│       │   ├── encryption.go
│       │   ├── sanitizer.go
│       │   ├── rate_limiter.go
│       │   ├── csrf.go
│       │   ├── cors.go
│       │   └── headers.go
│       ├── models/
│       ├── repository/
│       ├── routes/
│       ├── auth/
│       ├── similarity/
│       ├── search/
│       ├── cache/
│       ├── subscription/
│       ├── profile/
│       ├── compliance/
│       ├── optimizer/
│       ├── admin/
│       ├── external/
│       └── monitoring/
├── frontend/
│   ├── src/
│   │   ├── lib/
│   │   │   ├── components/
│   │   │   ├── stores/
│   │   │   ├── api/
│   │   │   └── theme/
│   │   └── routes/
│   └── static/
│       └── sw.js
├── assets/
│   └── indicators/
└── docker-compose.yml
```
