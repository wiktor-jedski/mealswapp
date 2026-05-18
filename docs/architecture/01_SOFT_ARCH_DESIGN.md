# Software Architecture Design (SAD)
**Project:** Mealswapp
**Process:** SWE.2 Software Architectural Design
**Version:** 1.1 (ASPICE 4.0 Compliant)

## 1. Introduction

This document defines the software architecture for the Mealswapp application, decomposing the Software Requirements Specification into software components, interfaces, and design decisions. All architectural elements trace back to one or more [SW-REQ-XXX] requirements.

### 1.1 Architectural Overview

The Mealswapp application follows a **three-tier architecture** with a responsive Svelte web frontend, a Fiber-based Go RESTful API backend, and a persistent data layer using PostgreSQL and Redis. The system is designed for horizontal scalability, security, and future mobile platform integration.

```
┌─────────────────────────────────────────────────────────────────┐
│                     CLIENT TIER                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              ARCH-001: Web Application                   │    │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────────┐   │    │
│  │  │Search UI│ │Sidebar  │ │Results  │ │Offline Cache│   │    │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────────┘   │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                              │ HTTPS/TLS 1.3
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     SERVICE TIER                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              ARCH-010: API Gateway                       │    │
│  │         (Rate Limiting, CSRF, Security Headers)          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│  ┌─────────┬─────────┬───────┴───────┬─────────┬─────────┐     │
│  │         │         │               │         │         │     │
│  ▼         ▼         ▼               ▼         ▼         ▼     │
│ ┌───┐   ┌───┐     ┌───┐           ┌───┐     ┌───┐     ┌───┐   │
│ │002│   │003│     │004│           │006│     │007│     │009│   │
│ │Srch│  │Sim│     │LP │           │Auth│    │Sub │    │Admn│  │
│ └───┘   └───┘     └───┘           └───┘     └───┘     └───┘   │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              ARCH-005: Data Repository                   │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     DATA TIER                                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐     │
│  │ PostgreSQL  │  │    Redis    │  │  External APIs      │     │
│  │  (Primary)  │  │   (Cache)   │  │  (USDA/OpenFood)    │     │
│  └─────────────┘  └─────────────┘  └─────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Resource Goals

| Resource | Target | Rationale |
| :--- | :--- | :--- |
| **API Response Time** | < 2s (P95) | SW-REQ-080 |
| **Concurrent Users** | 1000+ | SW-REQ-082 |
| **System Availability** | 99.9% | SW-REQ-081 |
| **Memory (Client)** | < 50MB heap | Mobile browser optimization |
| **Network Payload** | < 100KB initial load | Mobile data efficiency |

### 1.3 Hosting Infrastructure

The application is deployed on Google Cloud Platform (GCP) using managed services to minimize operational overhead:

| Service | Purpose |
| :--- | :--- |
| **Cloud Run** | Containerized backend API deployment with automatic scaling |
| **Cloud SQL** | Managed PostgreSQL database |
| **Memorystore** | Managed Redis for caching and session management |
| **Cloud Storage + Cloud CDN** | Static asset hosting (images, similarity indicators) |
| **Cloud Monitoring** | Logging, metrics, and uptime monitoring |
| **Secret Manager** | Secure storage of API keys, database credentials, and secrets |
| **GitHub Actions** | CI/CD pipeline for automated testing and deployment |

Container orchestration is not required as managed services handle scaling and availability.

---

## 2. Architectural Components

---

## [ARCH-001] - Web Application Module

**Description:** The responsive single-page application (SPA) built with Svelte that serves as the primary user interface, handling all client-side rendering, state management with Svelte stores + TanStack Query, local caching via Service Worker + localStorage, and offline functionality.

| Attribute | Value |
| :--- | :--- |
| **Type** | Module |
| **Static Aspects** | SearchView, SidebarComponent, ResultsGrid, AutocompleteDropdown, ThemeProvider, OfflineBanner, SettingsPanel, LocalStorageManager, ServiceWorker |
| **Dependencies** | ARCH-010 (API Gateway), ARCH-011 (Caching Layer), TanStack Query |
| **Traceability** | SW-REQ-001, SW-REQ-002, SW-REQ-003, SW-REQ-005, SW-REQ-007, SW-REQ-008, SW-REQ-009, SW-REQ-011, SW-REQ-012, SW-REQ-013, SW-REQ-014, SW-REQ-015, SW-REQ-018, SW-REQ-025, SW-REQ-048, SW-REQ-077, SW-REQ-085, SW-REQ-086, SW-REQ-087, SW-REQ-088, SW-REQ-089 |

**Dynamic Behavior:**

- **Initialization:** On application load, initializes search mode to 'Single Item' and enables all macronutrient toggles. Detects system theme preference and applies user-stored preference override.
- **Search Input:** Debounces user input by 150ms before triggering API calls. Manages focus states for keyboard navigation (Tab/Shift+Tab).
- **Offline Detection:** Monitors browser online/offline events. Switches to cached data display and shows offline indicator when disconnected.
- **Theme Switching:** Real-time CSS variable updates when user toggles light/dark mode. Persists selection to localStorage.

**Interface Definition:**

- `Input`: User interactions (keyboard, mouse, touch), system events (online/offline), API responses (JSON)
- `Output`: HTTP requests to API Gateway, localStorage writes, DOM updates

**Alternative Analysis (BP6):**

- *Chosen Approach:* Single-Page Application with client-side routing and state management
- *Alternative Considered:* Server-Side Rendering (SSR) with hydration
- *Trade-off:* SPA provides better offline capability (SW-REQ-087, SW-REQ-088) and reduces server load. SSR would improve initial load SEO but adds server complexity and breaks offline-first design. Since Mealswapp is an authenticated app (not SEO-critical), SPA is superior.

**Reference Documentation:** 
- 02_APPENDIX_A.md

---

## [ARCH-002] - Search Module

**Description:** Backend service responsible for processing search queries, implementing autocomplete ranking, and coordinating with the Similarity Engine for result retrieval and filtering.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | SearchController, AutocompleteRanker, QueryParser, PaginationHandler, FilterProcessor, FunctionalityTagWeighter |
| **Dependencies** | ARCH-003 (Similarity Engine), ARCH-005 (Data Repository), ARCH-011 (Caching Layer) |
| **Traceability** | SW-REQ-004, SW-REQ-010, SW-REQ-017, SW-REQ-019, SW-REQ-024, SW-REQ-026, SW-REQ-029, SW-REQ-031 |

**Dynamic Behavior:**

- **Query Processing:** Receives search terms, applies tag whitelist/blacklist filters, and routes to appropriate search strategy (text-based or similarity-based).
- **Autocomplete Ranking:** Implements three-tier priority: (1) Exact match, (2) Levenshtein distance, (3) String length. Executes in < 100ms.
- **Implicit Trigger:** Detects empty search bar with 2+ ingredients to automatically initiate similarity search.
- **Pagination:** Returns max 10 results per page, sorted by cosine similarity descending.
- **Functionality Tag Weighting (SW-REQ-031):** During replacement searches, applies a relevance boost multiplier to items sharing the same Functionality Tags as the source item. Sorting combines cosine similarity score with tag match weight (e.g., `finalScore = similarityScore * (1 + 0.2 * tagMatchCount)`) to prioritize contextually appropriate replacements.

**Interface Definition:**

- `Input`: SearchRequest { query: string, mode: SearchMode, filters: TagFilter[], page: number, ingredients?: string[] }
- `Output`: SearchResponse { items: FoodItem[], totalCount: number, page: number, similarityScores: number[] }

**Alternative Analysis (BP6):**

- *Chosen Approach:* Dedicated Search Module with in-memory ranking algorithms
- *Alternative Considered:* Elasticsearch/Algolia for full-text search
- *Trade-off:* Custom module provides precise control over ranking algorithm (SW-REQ-004) and cosine similarity integration. External search services would require synchronization overhead and may not support custom similarity scoring. For the current scale (1000 users), custom solution is more cost-effective and controllable.

**Reference Documentation:** 
- 02_APPENDIX_A.md

---

## [ARCH-003] - Similarity Engine

**Description:** Core computational service that calculates cosine similarity between food items based on macronutrient vectors (Protein, Carbohydrates, Fat), applies threshold filtering, and provides visual indicator mappings.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | CosineSimilarityCalculator, MacroVectorNormalizer, ThresholdFilter, SimilarityIndicatorMapper, SimilarityAssetResolver |
| **Dependencies** | ARCH-005 (Data Repository) |
| **Traceability** | SW-REQ-016, SW-REQ-017, SW-REQ-018, SW-REQ-026, SW-REQ-027, SW-REQ-028 |

**Dynamic Behavior:**

- **Vector Calculation:** Normalizes macronutrient values to unit vectors. For recipes, aggregates constituent ingredient macros before normalization.
- **Micronutrient Exclusion:** Ignores micronutrient key-value pairs entirely; similarity vectors contain only Protein, Carbohydrates, and Fat.
- **Similarity Scoring:** Computes cosine similarity using dot product of normalized vectors. Filters results below 0.40 threshold.
- **Visual Indicator Mapping (SW-REQ-018):** Assigns tier indicators based on score thresholds. Returns both color code and server-hosted image URL for the indicator icon. Indicator images are stored as static assets on the server (not client-side Unicode emojis) to ensure consistent cross-platform rendering.
  - Green + `/assets/indicators/star.png` for >=85%
  - Light Green + `/assets/indicators/sparkle.png` for 70-84%
  - Yellow + `/assets/indicators/thumbs-up.png` for 55-69%
  - Red + `/assets/indicators/thumbs-down.png` for <55%
- **Quantity Matching:** Calculates replacement quantities to match original calorie or protein counts.

**Interface Definition:**

- `Input`: ComparisonRequest { sourceItem: MacroVector, targetItems: MacroVector[], matchType: 'calorie' | 'protein' }
- `Output`: SimilarityResult { itemId: string, score: number, tier: SimilarityTier, matchingQuantity: number }[]

**Alternative Analysis (BP6):**

- *Chosen Approach:* Three-dimensional cosine similarity (P, C, F)
- *Alternative Considered:* Euclidean distance in macro space, or weighted similarity including calories
- *Trade-off:* Cosine similarity measures directional alignment of macro ratios regardless of magnitude, which aligns with nutritional replacement goals (same macro profile at any quantity). Euclidean would penalize magnitude differences inappropriately. Adding calories as 4th dimension would over-weight it since calories derive from macros.

---

## [ARCH-004] - Linear Programming Optimizer

**Description:** Asynchronous optimization service that uses linear programming to generate alternative diet combinations matching target macronutrient profiles while minimizing total calories. Operates as a job queue to handle CPU-intensive calculations without blocking the API.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service (Asynchronous Job Queue) |
| **Static Aspects** | LPSolverWrapper (go-coinor/clp), ConstraintBuilder, ObjectiveFunction, DiversityPenalizer, SolutionValidator, JobQueueManager, JobStatusTracker |
| **Dependencies** | ARCH-003 (Similarity Engine), ARCH-005 (Data Repository), Redis (Job Queue via go-redis/queue or machinery), ARCH-010 (API Gateway) |
| **Traceability** | SW-REQ-021, SW-REQ-022, SW-REQ-023, SW-REQ-030, SW-REQ-080, SW-REQ-082 |

**Dynamic Behavior:**

- **Job Submission:** Client submits optimization request. API returns `202 Accepted` with a `jobId` immediately, without blocking. Job is queued in Redis-backed queue (go-redis/queue or machinery).
- **Asynchronous Processing:** Worker processes pick up jobs from queue. LP solving occurs asynchronously, preventing blocking under concurrent load.
- **Constraint Setup:** Builds linear constraints for target Protein, Carbohydrate, and Fat values with configurable tolerance bands.
- **Objective Minimization:** Defines calorie count as primary objective function to minimize.
- **Diversity Weighting:** Applies penalty weights to meal IDs present in original diet to encourage diverse alternatives.
- **Multi-Solution Generation:** Iteratively solves LP with exclusion constraints to produce up to 3 distinct alternatives.
- **Result Retrieval:** Client polls `GET /jobs/{jobId}` endpoint or subscribes via WebSocket for completion notification. Results cached in Redis with 1-hour TTL.
- **Timeout Handling:** Jobs exceeding 30 seconds are terminated and marked as failed. Client receives partial results if available.

**Interface Definition:**

- `Input`: POST /api/v1/diet/optimize -> DietOptimizationRequest { originalMeals: Meal[], targetMacros: MacroTarget, excludedIds: string[] }
- `Output (Immediate)`: { jobId: string, status: 'queued', pollUrl: string }
- `Output (Poll)`: GET /api/v1/jobs/{jobId} -> { status: 'queued'|'processing'|'completed'|'failed', result?: DietAlternative[], error?: string }
- `Output (WebSocket)`: Event { jobId: string, status: 'completed', result: DietAlternative[] }

**Alternative Analysis (BP6):**

- *Chosen Approach:* Asynchronous Job Queue with Redis-backed go-redis/queue or machinery and worker pool
- *Alternative Considered:* Synchronous LP execution within API request lifecycle
- *Trade-off:* Synchronous execution would block the Go Fiber event loop during CPU-intensive LP solving. With 1000 concurrent users (SW-REQ-082) and 200+ simultaneous diet searches, this creates a self-inflicted DoS condition, failing SW-REQ-080 (<2s response) and SW-REQ-081 (99.9% availability). Asynchronous queue isolates CPU work, maintains API responsiveness, and allows horizontal scaling of worker processes independently.

**Reference Documentation:** 
- 02_APPENDIX_A.md

---

## [ARCH-005] - Data Repository Module

**Description:** Central data access layer implementing the domain data model, handling all database operations, unit conversions, and data normalization for food items, meals, and recipes.

| Attribute | Value |
| :--- | :--- |
| **Type** | Module |
| **Static Aspects** | FoodItemEntity, MealEntity, RecipeEntity, TagEntity, MicronutrientVocabulary, UnitConverter, MacroNormalizer, RepositoryInterfaces |
| **Dependencies** | PostgreSQL (primary datastore, via lib/pq or pgx) |
| **Traceability** | SW-REQ-032, SW-REQ-033, SW-REQ-034, SW-REQ-035, SW-REQ-036, SW-REQ-037, SW-REQ-038, SW-REQ-039, SW-REQ-040, SW-REQ-041, SW-REQ-090 |

**Dynamic Behavior:**

- **Normalization:** All macronutrient values stored per 100g (solids) or 100ml (liquids). Conversion applied on read based on user preference.
- **Unit Conversion:** Metric-to-Imperial conversion (g->oz, ml->fl oz) performed at repository boundary, never in storage.
- **Recipe Aggregation:** Dynamically calculates total macros for recipe-based meals by summing constituent ingredients.
- **Real-time Scaling:** Provides calculation methods for quantity-based macro scaling.
- **Micronutrient Validation:** Validates all micronutrient keys against a centrally managed vocabulary before storage. Micronutrients are persisted for display/export only and are never passed into similarity vector calculations.

**Interface Definition:**

- `Input`: CRUD operations, unit preference context, quantity parameters
- `Output`: Domain entities with macros in requested unit system

**Data Model (Core Entities):**

```
FoodItem {
  id: UUID
  name: string
  physicalState: 'solid' | 'liquid'        // SW-REQ-035
  prepTime: minutes                         // SW-REQ-035
  averageUnitWeight: grams                  // SW-REQ-036
  macros: { protein, carbs, fat } per 100g  // SW-REQ-033
  micros: { sodium, fiber, ... }            // SW-REQ-038, keys validated by SW-REQ-090 vocabulary
  categoryTags: Tag[]                       // SW-REQ-012
  functionalityTags: Tag[]                  // SW-REQ-037
  imageUrl: string?
}

Meal {
  id: UUID
  type: 'single' | 'recipe'                 // SW-REQ-034
  items?: FoodItem                          // single dish
  recipe?: { item: FoodItem, qty: number }[] // recipe composition
  physicalState: 'solid' | 'liquid'        // SW-REQ-035
  prepTime: minutes                         // SW-REQ-035
  averageUnitWeight: grams                  // SW-REQ-036
  categoryTags: Tag[]                       // SW-REQ-012
  functionalityTags: Tag[]                  // SW-REQ-037
}

SimilarityIndicatorAsset {                   // SW-REQ-018
  tier: 'excellent' | 'good' | 'fair' | 'poor'
  colorHex: string                           // e.g., "#22C55E" for green
  imageUrl: string                           // Server-hosted image path
  minScore: number                           // Lower bound threshold
  maxScore: number                           // Upper bound threshold
}

MicronutrientVocabularyEntry {               // SW-REQ-090
  key: string                                 // canonical key, e.g., "Sodium"
  displayName: string
  unit: string                                // e.g., "mg", "mcg", "g"
  active: boolean
}
```

**Alternative Analysis (BP6):**

- *Chosen Approach:* PostgreSQL relational database with normalized schema
- *Alternative Considered:* MongoDB document store for flexible food item schema
- *Trade-off:* Relational model ensures data integrity for macronutrient calculations and enforces consistent schema across all items (critical for SW-REQ-033). Recipe composition with foreign keys prevents orphaned ingredients. PostgreSQL JSONB columns can handle variable micronutrient fields while maintaining relational benefits.

**Reference Documentation:** 
- 02_APPENDIX_A.md

---

## [ARCH-006] - Authentication Module

**Description:** Security service handling user authentication via email/password and social providers (Google, Apple), session management, and token lifecycle.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | AuthController, PasswordHasher (Argon2), JWTManager, OAuthHandler (goth), SessionManager (Fiber session middleware), AccountLockoutTracker |
| **Dependencies** | ARCH-005 (Data Repository), ARCH-013 (Security Middleware), External OAuth Providers (Google, Apple via github.com/markbates/goth) |
| **Traceability** | SW-REQ-046, SW-REQ-058, SW-REQ-059, SW-REQ-060, SW-REQ-061, SW-REQ-062, SW-REQ-063, SW-REQ-064, SW-REQ-065, SW-REQ-066, SW-REQ-069, SW-REQ-070 |

**Dynamic Behavior:**

- **Registration:** Validates email uniqueness, hashes password with Argon2 (golang.org/x/crypto/argon2, unique salt), sends verification email. Blocks paid features until verified.
- **Login:** Validates credentials, tracks failed attempts per account (5 max -> 15min lockout) and per IP (10 max/10min).
- **Token Lifecycle:** Issues 15-minute access tokens and 7-day refresh tokens in HttpOnly/Secure/SameSite=Strict cookies. Manages sessions via Fiber session middleware. Rotates refresh token on use.
- **Social Login:** Handles OAuth2 flows for Google/Apple using github.com/markbates/goth, creates or links user accounts, grants 7-day trial on first authentication.
- **Password Reset:** Generates cryptographically random single-use tokens valid for 1 hour.

**Interface Definition:**

- `Input`: Credentials (email/password or OAuth tokens), session cookies
- `Output`: JWT tokens (access/refresh), session state, verification emails

**Alternative Analysis (BP6):**

- *Chosen Approach:* Custom JWT-based authentication with HttpOnly cookies and Fiber session middleware, using github.com/markbates/goth for OAuth providers
- *Alternative Considered:* Third-party auth service (Auth0, Firebase Auth)
- *Trade-off:* Custom implementation provides full control over security requirements (SW-REQ-062, SW-REQ-063, SW-REQ-065) and avoids vendor lock-in. Third-party services simplify development but may not support exact lockout policies or cookie configurations required. For a subscription-based app with specific security needs, custom implementation ensures compliance.

**Reference Documentation:** 
- 02_APPENDIX_A.md

---

## [ARCH-007] - Subscription Module

**Description:** Service managing subscription tiers, payment processing via Stripe, entitlement enforcement, and trial period logic.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | SubscriptionController, StripeWebhookHandler, EntitlementManager, TrialTracker, UsageLimiter |
| **Dependencies** | ARCH-006 (Authentication), ARCH-005 (Data Repository), Stripe API |
| **Traceability** | SW-REQ-042, SW-REQ-044, SW-REQ-045, SW-REQ-050, SW-REQ-051, SW-REQ-052, SW-REQ-053 |

**Dynamic Behavior:**

- **Tier Enforcement:** Checks user entitlement on each request. Free tier: 3 searches/24h, single-item only. Paid/Trial: unlimited, all features.
- **Payment Flow:** Client uses Stripe Elements (PCI-DSS compliant tokenization). Server creates Payment Intents, never handles raw card data.
- **Webhook Processing:** Asynchronously processes payment_intent.succeeded/failed events to update entitlement status reliably.
- **Trial Management:** Activates 7-day trial on first social login. Tracks expiration timestamp. Auto-downgrades to Free tier on expiry.

**Interface Definition:**

- `Input`: Subscription requests, Stripe webhook events, entitlement checks
- `Output`: Entitlement status, payment session URLs, feature access decisions

**Alternative Analysis (BP6):**

- *Chosen Approach:* Stripe with server-side webhook processing for entitlement sync
- *Alternative Considered:* Client-side payment confirmation with polling
- *Trade-off:* Webhook-based sync (SW-REQ-045) ensures reliable entitlement updates even if user closes browser during payment. Polling would miss events and create inconsistent states. Stripe Elements ensure PCI-DSS scope reduction (SW-REQ-044) by tokenizing at client.

### Webhook Handling

**Idempotency:**
- Store `event.id` in `processed_events` table before processing
- On duplicate webhook delivery, return 200 OK without reprocessing
- Prevents double-crediting or duplicate entitlement updates

**Retry Policy Awareness:**
- Stripe retries failed webhooks for up to 3 days with exponential backoff
- Handler must be idempotent to safely handle retries
- Return 2xx status only after successful processing; 4xx/5xx triggers retry

**Signature Verification:**
- Validate `Stripe-Signature` header using webhook signing secret
- Reject webhooks with invalid or missing signatures (return 400)
- Prevents spoofed webhook attacks

### Partial Failure Recovery

**Scenario:** Payment succeeds at Stripe, but local entitlement database write fails.

**Solution:**
1. Webhook handler wraps entitlement update in database transaction
2. On transaction failure, log event to dead-letter queue with full payload
3. Return 500 to Stripe (triggers automatic retry)
4. Reconciliation job runs hourly: queries Stripe API for active subscriptions, compares with local entitlements, fixes discrepancies

**User-Facing Behavior:**
- During payment processing, UI shows "Payment processing..." state
- Entitlement confirmed only after webhook successfully processed
- If webhook fails repeatedly, reconciliation job catches within 1 hour

### Payment Flow Diagram

```
┌──────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Client  │────>│ Stripe      │────>│ ARCH-007    │────>│ ARCH-005    │
│          │     │ Checkout    │     │ Webhook     │     │ Repository  │
└──────────┘     └─────────────┘     └─────────────┘     └─────────────┘
     │                  │                   │                   │
     │  1. Redirect     │                   │                   │
     │─────────────────>│                   │                   │
     │                  │                   │                   │
     │  2. User pays    │                   │                   │
     │                  │                   │                   │
     │                  │ 3. payment_intent │                   │
     │                  │    .succeeded     │                   │
     │                  │──────────────────>│                   │
     │                  │                   │                   │
     │                  │                   │ 4. Verify         │
     │                  │                   │    signature      │
     │                  │                   │                   │
     │                  │                   │ 5. Check          │
     │                  │                   │    idempotency    │
     │                  │                   │                   │
     │                  │                   │ 6. BEGIN TXN      │
     │                  │                   │──────────────────>│
     │                  │                   │                   │
     │                  │                   │ 7. Update         │
     │                  │                   │    entitlement    │
     │                  │                   │<──────────────────│
     │                  │                   │                   │
     │                  │                   │ 8. Log event      │
     │                  │                   │──────────────────>│
     │                  │                   │                   │
     │                  │                   │ 9. COMMIT         │
     │                  │                   │                   │
     │                  │   10. HTTP 200    │                   │
     │                  │<──────────────────│                   │
     │                  │                   │                   │
     │ 11. Return URL   │                   │                   │
     │<─────────────────│                   │                   │
     │                  │                   │                   │
     │ 12. Fetch        │                   │                   │
     │    entitlement   │                   │                   │
     │─────────────────────────────────────>│                   │
     │                  │                   │                   │
     │ 13. Confirmed    │                   │                   │
     │<─────────────────────────────────────│                   │
```

---

## [ARCH-008] - User Profile Module

**Description:** Service managing user preferences, saved data, search history, favorites, and data export/deletion for GDPR compliance.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | ProfileController, PreferenceManager, SavedDataRepository, DataExporter, AccountDeleter |
| **Dependencies** | ARCH-005 (Data Repository), ARCH-006 (Authentication) |
| **Traceability** | SW-REQ-043, SW-REQ-047, SW-REQ-048, SW-REQ-049, SW-REQ-072, SW-REQ-073, SW-REQ-074 |

**Dynamic Behavior:**

- **Data Isolation:** Enforces user-scoped queries for all custom items and saved data. Cross-user access prevented at repository level.
- **Preference Propagation:** Updates to unit preferences trigger real-time recalculation across all displayed data (SW-REQ-041).
- **Data Export:** Generates JSON and CSV exports containing all user PII, saved items, diets, and history.
- **Account Deletion:** Permanently removes all PII and associated data from production database. Cascades to all related records.

**Interface Definition:**

- `Input`: User ID context, preference updates, export/delete requests
- `Output`: User profiles, exported data files, deletion confirmations

**Alternative Analysis (BP6):**

- *Chosen Approach:* Server-side profile storage with client-side history caching
- *Alternative Considered:* Fully client-side profile storage (localStorage only)
- *Trade-off:* Server-side storage enables cross-device sync and proper GDPR compliance (SW-REQ-072, SW-REQ-073). Pure client-side would lose data on device change and complicate data export requests. Hybrid approach uses localStorage for recent history (SW-REQ-048) while server stores persistent data.

---

## [ARCH-009] - Administration Module

**Description:** Restricted backend service providing administrative functions for data curation, user management, and global tag management. Acts as a proxy for external data searches to enable admin-curated imports.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | AdminController, DataImporter, ItemCurator, TagManager, UserAdminPanel, ExternalSearchProxy |
| **Dependencies** | ARCH-005 (Data Repository), ARCH-006 (Authentication), ARCH-012 (External Data Integration) |
| **Traceability** | SW-REQ-054, SW-REQ-055, SW-REQ-056, SW-REQ-057 |

**Dynamic Behavior:**

- **Access Control:** Validates 'Admin' role on all requests. Returns 403 Forbidden for non-admin users.
- **External Data Search (SW-REQ-055):** Admin UI provides a dedicated search interface that queries external APIs (not the local database). Flow:
  1. Admin enters search term in "External Import" panel
  2. `ExternalSearchProxy` routes request to ARCH-012 (External Data Integration)
  3. ARCH-012 queries USDA and/or OpenFoodFacts APIs
  4. Results displayed in admin UI with "Import" action for each item
  5. Admin selects item, edits fields (name, tags, macros), and confirms import
  6. `DataImporter` saves curated item to local database via ARCH-005
- **Item CRUD:** Full create/update/delete capabilities for food items including macros, images, and tags.
- **Tag Management:** Creates and manages global Category Tags and Functionality Tags used across all items.

**Interface Definition:**

- `Input`: Admin-authenticated requests, external search queries, item definitions
- `Output`: External search results (uncurated), curated items (post-import), tag hierarchies, admin audit logs

**Admin External Search Flow:**

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Admin UI   │────>│  ARCH-009   │────>│  ARCH-012   │────>│ USDA/OFF    │
│ (Search)    │     │ (Proxy)     │     │ (External)  │     │ (APIs)      │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
       │                   │                   │                   │
       │<──────────────────┴───────────────────┴───────────────────│
       │              Normalized results for curation              │
       │                                                           │
       ▼                                                           │
┌─────────────┐     ┌─────────────┐                               │
│ Edit & Tag  │────>│  ARCH-005   │  (Save curated item)          │
│ (Admin)     │     │ (Repository)│                               │
└─────────────┘     └─────────────┘                               │
```

**Alternative Analysis (BP6):**

- *Chosen Approach:* Integrated admin module within main application backend
- *Alternative Considered:* Separate admin microservice with dedicated database access
- *Trade-off:* Integrated module simplifies deployment and shares data models with main application. Separate microservice would add network latency and deployment complexity for minimal security benefit (RBAC already enforces access). Admin operations are low-frequency and don't require independent scaling.

---

## [ARCH-010] - API Gateway

**Description:** Entry point for all client requests, implementing routing, rate limiting, security header injection, CSRF protection, and request validation.

| Attribute | Value |
| :--- | :--- |
| **Type** | Middleware |
| **Static Aspects** | RouteHandler, RateLimiter, SecurityHeaderMiddleware, CSRFValidator, RequestValidator, CORSHandler |
| **Dependencies** | All backend services |
| **Traceability** | SW-REQ-064, SW-REQ-067, SW-REQ-068, SW-REQ-076, SW-REQ-078 |

**Dynamic Behavior:**

- **Rate Limiting:** Enforces 10 failed login attempts per IP per 10-minute window. Configurable limits per endpoint.
- **Security Headers:** Injects CSP, X-Frame-Options (DENY), X-Content-Type-Options (nosniff), Referrer-Policy, Permissions-Policy on all responses.
- **CSRF Protection:** Validates synchronizer tokens on all state-changing requests (POST, PUT, DELETE).
- **Timeout Management:** Enforces 10-second timeout on all API requests, returns 504 on timeout.
- **API Versioning:** Routes requests based on version prefix (e.g., /api/v1/) for future mobile integration.

**Interface Definition:**

- `Input`: HTTP requests from clients
- `Output`: Routed requests to services, HTTP responses with security headers

**Alternative Analysis (BP6):**

- *Chosen Approach:* Application-level API gateway (Fiber middleware)
- *Alternative Considered:* Dedicated API gateway service (Kong, AWS API Gateway)
- *Trade-off:* Application-level gateway reduces infrastructure complexity and latency for current scale. Dedicated gateway would provide advanced features (API keys, analytics) but adds operational overhead. For 1000 concurrent users (SW-REQ-082), application-level gateway is sufficient and simpler to deploy.

**Reference Documentation:** 
- 02_APPENDIX_A.md

---

## [ARCH-011] - Caching Layer

**Description:** Multi-tier caching system using client-side Service Worker with Cache API, localStorage for metadata, and server-side Redis to optimize performance and enable full offline functionality including images.

| Attribute | Value |
| :--- | :--- |
| **Type** | Middleware |
| **Static Aspects** | ServiceWorkerCache (client), LocalStorageCache (client), RedisCache (server), CacheInvalidator, LRUEvictionPolicy, UserCachePurger |
| **Dependencies** | Redis (via github.com/redis/go-redis/v9), Browser Service Worker API, Browser localStorage API, ARCH-008 (User Profile) |
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

---

## [ARCH-012] - External Data Integration

**Description:** Integration layer for fetching and normalizing food data from external APIs (USDA FoodData Central, OpenFoodFacts).

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | USDAClient, OpenFoodFactsClient, DataNormalizer, RateLimitHandler |
| **Dependencies** | External APIs (USDA, OpenFoodFacts), ARCH-005 (Data Repository) |
| **Traceability** | SW-REQ-055 |

**Dynamic Behavior:**

- **API Fetching:** Queries external APIs based on search terms, handles pagination and rate limits.
- **Data Normalization:** Converts external formats to internal schema, maps to standard units (per 100g/ml).
- **Error Handling:** Graceful degradation when external APIs are unavailable (returns empty results with warning).

**Interface Definition:**

- `Input`: Search queries, item identifiers
- `Output`: Normalized FoodItem candidates for admin curation

**Alternative Analysis (BP6):**

- *Chosen Approach:* On-demand fetching with admin curation workflow
- *Alternative Considered:* Bulk data import with scheduled synchronization
- *Trade-off:* On-demand fetching with curation (SW-REQ-055) ensures data quality and proper functionality tagging. Bulk import would populate database faster but with uncurated, potentially inconsistent data. Quality over quantity is critical for accurate similarity matching.

**Reference Documentation:** 
- 02_APPENDIX_A.md

---

## [ARCH-013] - Security Middleware

**Description:** Cross-cutting security services implementing encryption, input validation, and audit logging across all components.

| Attribute | Value |
| :--- | :--- |
| **Type** | Middleware |
| **Static Aspects** | EncryptionService (AES-256 via crypto/aes), InputSanitizer, AuditLogger, TLSEnforcer, RateLimiter (Fiber built-in limiter), CSRFValidator (Fiber csrf middleware) |
| **Dependencies** | All services |
| **Traceability** | SW-REQ-059, SW-REQ-068, SW-REQ-075, SW-REQ-084 |

**Dynamic Behavior:**

- **Encryption at Rest:** AES-256 encryption (crypto/aes) for PII fields in database.
- **Encryption in Transit:** TLS 1.3 enforced for all connections. HTTP redirects to HTTPS.
- **Input Validation:** Sanitizes all user inputs to prevent XSS, SQL injection, and command injection.
- **Rate Limiting:** Enforces rate limits using Fiber built-in limiter middleware.
- **CSRF Protection:** Validates synchronizer tokens on all state-changing requests using Fiber csrf middleware.
- **Audit Logging:** Logs all authentication events, API requests, errors, and admin actions with timestamps and user IDs.

**Interface Definition:**

- `Input`: Raw data for encryption, user inputs for validation
- `Output`: Encrypted data, sanitized inputs, audit log entries

**Alternative Analysis (BP6):**

- *Chosen Approach:* Application-level encryption with database-native TDE as backup
- *Alternative Considered:* Full database-level Transparent Data Encryption (TDE) only
- *Trade-off:* Application-level encryption provides field-level control over which data is encrypted and allows encryption keys to be managed separately from database. TDE-only would encrypt entire database but not protect against application-level data leaks. Layered approach provides defense in depth.

---

## [ARCH-014] - Logging & Monitoring Module

**Description:** Centralized logging and monitoring infrastructure for system health, performance tracking, and security auditing.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | LogAggregator, MetricsCollector, AlertManager, UptimeMonitor, FiberLogger (Fiber logger middleware) |
| **Dependencies** | All services (Architectural Overhead) |
| **Traceability** | SW-REQ-081, SW-REQ-083, SW-REQ-084 |

**Dynamic Behavior:**

- **Log Aggregation:** Collects structured logs from all services using Fiber logger middleware. Integrates with GCP Cloud Monitoring for log aggregation. Retains for minimum 90 days.
- **Metrics Collection:** Tracks response times, error rates, concurrent users for P95 latency monitoring via GCP Cloud Monitoring.
- **Uptime Monitoring:** Continuous health checks for 99.9% availability tracking via GCP Cloud Monitoring.
- **Backup Verification:** Monitors daily backup completion and tests restore capability.

**Interface Definition:**

- `Input`: Log events from all services, metrics data points
- `Output`: Aggregated dashboards, alerts, audit reports

**Alternative Analysis (BP6):**

- *Chosen Approach:* Centralized logging with GCP Cloud Monitoring (cloud-native equivalent)
- *Alternative Considered:* Distributed logging with per-service log files
- *Trade-off:* Centralized logging enables correlation across services for debugging and security auditing (SW-REQ-084). Distributed logs would be simpler but make cross-service analysis difficult. Centralized approach is essential for maintaining 99.9% availability (SW-REQ-081) through proactive monitoring.

**Reference Documentation:** 
- 02_APPENDIX_A.md

---

## [ARCH-015] - Compliance Module

**Description:** Service handling legal and regulatory requirements including GDPR compliance, consent management, and disclaimer display.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | ConsentManager, DisclaimerRenderer, DataRetentionPolicy, BackupManager |
| **Dependencies** | ARCH-005 (Data Repository), ARCH-008 (User Profile) |
| **Traceability** | SW-REQ-071, SW-REQ-072, SW-REQ-073, SW-REQ-074, SW-REQ-083 |

**Dynamic Behavior:**

- **Consent Capture:** Blocks registration completion until Privacy Policy and ToS checkboxes are explicitly checked.
- **Disclaimer Display:** Renders medical disclaimer on login screen and in About section.
- **Data Retention:** Enforces 30-day backup retention with point-in-time recovery capability.
- **Erasure Processing:** Coordinates complete data deletion across primary database and schedules backup purge.

**Interface Definition:**

- `Input`: Consent status, deletion requests, backup schedules
- `Output`: Consent records, disclaimer content, backup status

**Alternative Analysis (BP6):**

- *Chosen Approach:* Integrated compliance module with automated retention policies
- *Alternative Considered:* Manual compliance processes with external legal review
- *Trade-off:* Automated compliance ensures consistent enforcement of GDPR requirements (SW-REQ-072, SW-REQ-073, SW-REQ-074) without human error. Manual processes would require dedicated staff and risk non-compliance. Automation also enables faster response to data subject requests.

---

## [ARCH-016] - Theme & Style Module

**Description:** Client-side theming system implementing the Style Guide specifications for consistent visual presentation across light and dark modes.

| Attribute | Value |
| :--- | :--- |
| **Type** | Module |
| **Static Aspects** | ThemeProvider, ColorPalette, TypographySystem, LayoutGrid, ComponentStyles |
| **Dependencies** | ARCH-001 (Web Application) |
| **Traceability** | SW-REQ-014, SW-REQ-015, SW-REQ-085, SW-REQ-089 |

**Dynamic Behavior:**

- **Theme Detection:** Reads system prefers-color-scheme on load. User preference overrides system setting.
- **Variable Switching:** Updates CSS custom properties for all color tokens when theme changes.
- **Responsive Layout:** 12-column grid collapses to single column below 640px breakpoint.
- **Accessibility Enforcement:** Validates all color combinations meet WCAG 2.1 AA 4.5:1 contrast ratio.

**Interface Definition:**

- `Input`: Theme preference (system or user), viewport dimensions
- `Output`: CSS custom property values, responsive layout classes

**Color Tokens (Light Mode):**

| Token | Value | Usage |
| :--- | :--- | :--- |
| --bg-primary | #F7FCF7 | Main background |
| --bg-surface | #FFFFFF | Cards, containers |
| --color-primary | #166534 | Buttons, headers |
| --color-secondary | #DCFCE7 | Badges, highlights |
| --color-accent | #F97316 | Special offers |
| --color-error | #DC2626 | Validation errors |
| --text-primary | #111827 | Body text |
| --text-muted | #6B7280 | Secondary labels |

**Color Tokens (Dark Mode):**

| Token | Value | Usage |
| :--- | :--- | :--- |
| --bg-primary | #0A0F0A | Main background |
| --bg-surface | #161D16 | Cards, containers |
| --color-primary | #4ADE80 | Buttons, active states |
| --color-secondary | #86EFAC | Secondary actions |
| --color-accent | #FFB86C | Best match badges |
| --color-error | #F87171 | Alerts |
| --text-primary | #F3F4F6 | Body text |
| --text-muted | #9CA3AF | Descriptions |

**Alternative Analysis (BP6):**

- *Chosen Approach:* CSS Custom Properties with theme provider context
- *Alternative Considered:* CSS-in-JS with runtime theme switching (Styled Components, Emotion)
- *Trade-off:* CSS Custom Properties provide zero-runtime theme switching with native browser support. CSS-in-JS adds JavaScript bundle size and runtime overhead. For the defined color palette (SW-REQ-089), native CSS variables are simpler and more performant.

---

## [ARCH-017] - Error Handling Module

**Description:** Centralized error handling system implementing graceful degradation, user-friendly error messages, and automatic retry logic.

| Attribute | Value |
| :--- | :--- |
| **Type** | Module |
| **Static Aspects** | ErrorBoundary (client), GlobalExceptionHandler (server), RetryManager, ErrorMessageMapper |
| **Dependencies** | ARCH-001 (Web Application), ARCH-010 (API Gateway - Fiber) |
| **Traceability** | SW-REQ-077, SW-REQ-078, SW-REQ-079 |

**Dynamic Behavior:**

- **Network Failure:** Preserves application state, displays retry option, auto-retries on connectivity restoration.
- **Timeout Handling:** Shows timeout notification after 10 seconds, offers manual retry.
- **Graceful Degradation:** Isolates non-critical feature failures (history sync, recommendations) from core functionality (search, auth).
- **Error Classification:** Maps technical errors to user-friendly messages without exposing system internals.

**Interface Definition:**

- `Input`: Error events, network status changes
- `Output`: User-facing error messages, retry triggers, degraded feature flags

**Alternative Analysis (BP6):**

- *Chosen Approach:* Centralized error boundary with feature-level isolation
- *Alternative Considered:* Per-component error handling
- *Trade-off:* Centralized handling ensures consistent user experience and prevents full application crashes (SW-REQ-079). Per-component handling would require duplicated logic and risk inconsistent error messages. Feature isolation at the boundary level provides both centralization and graceful degradation.

---

## 3. Interface Definitions

### 3.1 External Interfaces

| Interface | Protocol | Description | Security |
| :--- | :--- | :--- | :--- |
| **Client <-> API** | HTTPS (TLS 1.3) | RESTful API endpoints via Fiber | JWT in HttpOnly cookies, Fiber CSRF middleware |
| **API <-> Stripe** | HTTPS | Payment processing | Webhook signatures |
| **API <-> OAuth** | OAuth 2.0 | Google/Apple login via github.com/markbates/goth | PKCE flow |
| **API <-> USDA** | HTTPS | Food data retrieval | API key |
| **API <-> OpenFoodFacts** | HTTPS | Food data retrieval | Public API |

### 3.2 Internal Interfaces

| Interface | Type | Data Flow |
| :--- | :--- | :--- |
| **Search -> Similarity** | Function call | MacroVectors -> SimilarityScores |
| **Search -> Repository** | Query interface | Filters -> FoodItems |
| **Optimizer -> Similarity** | Function call | ItemPairs -> Scores |
| **Auth -> Repository** | Query interface | Credentials -> Users |
| **Subscription -> Auth** | Event | EntitlementUpdates |

---

## 4. Traceability Matrix

| Requirement | Architectural Component(s) |
| :--- | :--- |
| SW-REQ-001 | ARCH-001 |
| SW-REQ-002 | ARCH-001 |
| SW-REQ-003 | ARCH-001, ARCH-011 |
| SW-REQ-004 | ARCH-002 |
| SW-REQ-005 | ARCH-001 |
| SW-REQ-006 | ARCH-001, ARCH-002 |
| SW-REQ-007 | ARCH-001 |
| SW-REQ-008 | ARCH-001 |
| SW-REQ-009 | ARCH-001 |
| SW-REQ-010 | ARCH-002 |
| SW-REQ-011 | ARCH-001 |
| SW-REQ-012 | ARCH-001 |
| SW-REQ-013 | ARCH-001 |
| SW-REQ-014 | ARCH-001, ARCH-016 |
| SW-REQ-015 | ARCH-001, ARCH-016 |
| SW-REQ-016 | ARCH-003 |
| SW-REQ-017 | ARCH-002, ARCH-003 |
| SW-REQ-018 | ARCH-001, ARCH-003 |
| SW-REQ-019 | ARCH-002 |
| SW-REQ-020 | ARCH-001, ARCH-005 |
| SW-REQ-021 | ARCH-004 |
| SW-REQ-022 | ARCH-004 |
| SW-REQ-023 | ARCH-004 |
| SW-REQ-024 | ARCH-002 |
| SW-REQ-025 | ARCH-001 |
| SW-REQ-026 | ARCH-002, ARCH-003 |
| SW-REQ-027 | ARCH-003 |
| SW-REQ-028 | ARCH-003 |
| SW-REQ-029 | ARCH-002 |
| SW-REQ-030 | ARCH-004 |
| SW-REQ-031 | ARCH-002 |
| SW-REQ-032 | ARCH-005 |
| SW-REQ-033 | ARCH-005 |
| SW-REQ-034 | ARCH-005 |
| SW-REQ-035 | ARCH-005 |
| SW-REQ-036 | ARCH-005 |
| SW-REQ-037 | ARCH-005 |
| SW-REQ-038 | ARCH-005 |
| SW-REQ-039 | ARCH-005 |
| SW-REQ-040 | ARCH-005 |
| SW-REQ-041 | ARCH-005, ARCH-008 |
| SW-REQ-042 | ARCH-007 |
| SW-REQ-043 | ARCH-008 |
| SW-REQ-044 | ARCH-007 |
| SW-REQ-045 | ARCH-007 |
| SW-REQ-046 | ARCH-006 |
| SW-REQ-047 | ARCH-008 |
| SW-REQ-048 | ARCH-001, ARCH-011 |
| SW-REQ-049 | ARCH-008 |
| SW-REQ-050 | ARCH-007 |
| SW-REQ-051 | ARCH-007 |
| SW-REQ-052 | ARCH-007 |
| SW-REQ-053 | ARCH-007 |
| SW-REQ-054 | ARCH-009 |
| SW-REQ-055 | ARCH-009, ARCH-012 |
| SW-REQ-056 | ARCH-009 |
| SW-REQ-057 | ARCH-009 |
| SW-REQ-058 | ARCH-006 |
| SW-REQ-059 | ARCH-006, ARCH-013 |
| SW-REQ-060 | ARCH-006 |
| SW-REQ-061 | ARCH-006 |
| SW-REQ-062 | ARCH-006 |
| SW-REQ-063 | ARCH-006 |
| SW-REQ-064 | ARCH-006, ARCH-010 |
| SW-REQ-065 | ARCH-006 |
| SW-REQ-066 | ARCH-006 |
| SW-REQ-067 | ARCH-010 |
| SW-REQ-068 | ARCH-010, ARCH-013 |
| SW-REQ-069 | ARCH-006 |
| SW-REQ-070 | ARCH-006 |
| SW-REQ-071 | ARCH-015 |
| SW-REQ-072 | ARCH-008, ARCH-015 |
| SW-REQ-073 | ARCH-008, ARCH-011, ARCH-015 |
| SW-REQ-074 | ARCH-015 |
| SW-REQ-075 | ARCH-013 |
| SW-REQ-076 | ARCH-010 |
| SW-REQ-077 | ARCH-001, ARCH-017 |
| SW-REQ-078 | ARCH-010, ARCH-017 |
| SW-REQ-079 | ARCH-017 |
| SW-REQ-080 | ARCH-002, ARCH-011 |
| SW-REQ-081 | ARCH-014 |
| SW-REQ-082 | ARCH-010 |
| SW-REQ-083 | ARCH-014, ARCH-015 |
| SW-REQ-084 | ARCH-013, ARCH-014 |
| SW-REQ-085 | ARCH-001, ARCH-016 |
| SW-REQ-086 | ARCH-001 |
| SW-REQ-087 | ARCH-001, ARCH-017 |
| SW-REQ-088 | ARCH-001, ARCH-011 |
| SW-REQ-089 | ARCH-001, ARCH-016 |
| SW-REQ-090 | ARCH-005 |

---

## 5. Changelog

### 2026-01-21 (Rev 1.1)

**Changed (Post-Review Remediation):**
- **ARCH-004:** Converted from synchronous service to asynchronous job queue pattern (go-redis/queue or machinery/Redis) to prevent CPU blocking under concurrent load. Added job submission, polling, and WebSocket notification interfaces. Addresses performance risk for SW-REQ-080, SW-REQ-082.
- **ARCH-011:** Replaced localStorage-only approach with Service Worker + Cache API for offline image caching. Added `UserCachePurger` for GDPR-compliant Redis cache invalidation on account deletion (SW-REQ-073).
- **ARCH-002:** Added `FunctionalityTagWeighter` component and explicit dynamic behavior for relevance boosting based on Functionality Tag matches during replacement searches (SW-REQ-031).
- **ARCH-003:** Added `SimilarityAssetResolver` and explicit server-hosted image URLs for similarity tier indicators. Addresses SW-REQ-018 requirement to store emojis as server images.
- **ARCH-005:** Added `SimilarityIndicatorAsset` entity to data model for storing tier indicator images (SW-REQ-018).
- **ARCH-009:** Added `ExternalSearchProxy` component and detailed flow diagram showing how admin searches external APIs (USDA/OpenFoodFacts) via ARCH-012 for data curation workflow (SW-REQ-055).
- **Traceability Matrix:** Updated SW-REQ-073 mapping to include ARCH-011.

### 2026-01-21 (Rev 1.0)

**Added:**
- Document created with 17 architectural components
- Full traceability to 89 software requirements
- Alternative analysis for all major design decisions
- Resource goals and interface definitions
