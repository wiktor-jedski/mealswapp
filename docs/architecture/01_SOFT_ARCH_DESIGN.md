# Software Architecture Design (SWE.2)
**Project:** Mealswapp
**Version:** 1.0
**Methodology:** ASPICE 4.0

## 1. System Overview
The Mealswapp system is designed as a **Client-Server Architecture** utilizing a **Layered Pattern**.
- **Frontend:** A Single Page Application (SPA) facilitating responsive UI, local caching, and state management.
- **Backend:** A RESTful API Service handling business logic, complex mathematical modeling (Linear Programming), authentication, and database abstraction.
- **Infrastructure:** Cloud-hosted containerized environment with managed database services.

---

### [ARCH-FE-CORE] - Single Page Application Shell
**Description:** The main entry point of the web application. It initializes the React/Framework instance, handles global error boundaries, and manages the client-side routing.

| Attribute | Value |
| :--- | :--- |
| **Type** | Module (Application Root) |
| **Static Aspects** | `App.js`, `Router`, `ErrorBoundary` |
| **Dependencies** | [ARCH-FE-THEME], [ARCH-FE-AUTH], [ARCH-FE-NET] |
| **Traceability** | [SW-REQ-014], [SW-REQ-071], [SW-REQ-077], [SW-REQ-079] |

**Dynamic Behavior:**
- **Initialization:** On load, triggers [ARCH-FE-AUTH] to validate session.
- **Routing:** Maps URLs to specific Views (Search, Login, Profile).
- **Error Handling:** If a child component crashes, displays the "Graceful Degradation" UI ([SW-REQ-079]).

**Interface Definition:**
- `Input`: Browser URL, User Events.
- `Output`: DOM updates, Route changes.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Client-side routing (SPA).
- *Alternative Considered:* Multi-page Application (Server-side rendering for every page).
- *Trade-off:* SPA is selected to support the "App-like" feel and offline capabilities ([SW-REQ-088]) required, which is harder to achieve with traditional SSR.

---

### [ARCH-FE-THEME] - Theme & Style Manager
**Description:** Manages the CSS variables and global context for the visual identity, including Light/Dark mode toggling and responsiveness.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service / Context |
| **Static Aspects** | `ThemeContext`, `GlobalStyles.css`, `TailwindConfig` |
| **Dependencies** | None |
| **Traceability** | [SW-REQ-015], [SW-REQ-089], [SW-REQ-007], [SW-REQ-012], [SW-REQ-018], [SW-REQ-085] |

**Dynamic Behavior:**
- **State Change:** Triggered by user toggle in Sidebar. Updates CSS variables for `#F7FCF7` (Light) or `#0A0F0A` (Dark) per Style Guide.
- **Persistence:** On change, writes preference to `localStorage`.

**Interface Definition:**
- `Input`: User Toggle Event, System Preference (`prefers-color-scheme`).
- `Output`: CSS Variable injection.

**Alternative Analysis (BP6):**
- *Chosen Approach:* CSS Variables (Custom Properties) managed by JS Context.
- *Alternative Considered:* Hardcoded CSS classes for every element.
- *Trade-off:* CSS Variables allow instant runtime switching without page reloads, essential for [SW-REQ-015].

---

### [ARCH-FE-SEARCH] - Search Logic Controller
**Description:** Handles the search bar input, debounce logic, autocomplete suggestions, result data formatting and orchestrates query execution.

| Attribute | Value |
| :--- | :--- |
| **Type** | Component Logic |
| **Static Aspects** | `useSearch` hook, `DebounceUtil`, `ResultFormatter`, `TriggerOrchestrator` |
| **Dependencies** | [ARCH-FE-CACHE], [ARCH-FE-NET], [ARCH-FE-LIST-MGR] |
| **Traceability** | [SW-REQ-001], [SW-REQ-002], [SW-REQ-004], [SW-REQ-008], [SW-REQ-009], [SW-REQ-024], [SW-REQ-025], [SW-REQ-011], [SW-REQ-006] |

**Dynamic Behavior:**
- **Debounce:** Wraps user input in a 150ms timer. If new input arrives <150ms, reset timer ([SW-REQ-002]).
- **Trigger:** If input stops >150ms, calls [ARCH-FE-NET].
- **Implicit Trigger:** Monitors Ingredient List state; if count >= 2, triggers Similarity Search ([SW-REQ-024]).
- **Data Mapping:** Parses API response to ensure Image, Name, Tags, Macros, and Similarity Score are present for the UI ([SW-REQ-011]).
- **Implicit Orchestration:** Subscribes to `ListContext`. IF `Mode == IngredientList` AND `List.length >= 2` AND `SearchInput == Empty`, THEN automatically fire `API.findSimilarity(List)` ([SW-REQ-024]).
- **Explicit Trigger:** Listens for "Search Button" click ([SW-REQ-025]).
    - If `Input != Empty`: Execute Standard Text Search.
    - If `Input == Empty` AND `List != Empty`: Force execute Similarity Search (Redundancy).
- **Mode Handling:** If `Mode == MealList`, modifies the search payload to request aggregatable meal objects rather than ingredients ([SW-REQ-006]).

**Interface Definition:**
- `Input`: Raw text string, Keydown events, List context state.
- `Output`: Filtered Data Array (Results), UI Expansion Events, Mode Switches.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Custom Hook with Debounce.
- *Alternative Considered:* Immediate API call on every keystroke.
- *Trade-off:* Debounce reduces server load and API costs significantly. Immediate calls would violate [SW-REQ-002].

---

### [ARCH-FE-CACHE] - Local Persistence Manager
**Description:** Wraps the Browser `localStorage` API to handle offline data, query caching, and search history.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service (Client-Side) |
| **Static Aspects** | `StorageManager` class |
| **Dependencies** | None |
| **Traceability** | [SW-REQ-003], [SW-REQ-048], [SW-REQ-088] |

**Dynamic Behavior:**
- **Write:** When a search returns successfully, push Query+Result to a Stack (Max 20).
- **Eviction:** LRU (Least Recently Used) logic removes oldest entry when >20 ([SW-REQ-003]).
- **Read:** Checks existence of key before requesting network.

**Interface Definition:**
- `Input`: Key (Query String), Data (JSON).
- `Output`: Cached JSON or `null`.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Browser `localStorage`.
- *Alternative Considered:* IndexedDB.
- *Trade-off:* `localStorage` is synchronous and simpler for the small dataset defined (20 queries). IndexedDB is overkill complexity for this phase.

---

### [ARCH-FE-NET] - Network & API Client
**Description:** A centralized Axios/Fetch wrapper that handles HTTP requests, timeout logic, headers, and retry mechanisms.

| Attribute | Value |
| :--- | :--- |
| **Type** | Middleware |
| **Static Aspects** | `ApiClient.js`, `Interceptors` |
| **Dependencies** | External REST API |
| **Traceability** | [SW-REQ-077], [SW-REQ-078], [SW-REQ-087] |

**Dynamic Behavior:**
- **Interception:** Injects Authorization Bearer Token (JWT).
- **Timeout:** Aborts requests >10,000ms ([SW-REQ-078]).
- **Retry:** On network error (status 0/503), queues retry logic if configured ([SW-REQ-087]).

**Interface Definition:**
- `Input`: Endpoint, Method, Payload.
- `Output`: Promise<Response> or Error (Timeout/Network).

**Alternative Analysis (BP6):**
- *Chosen Approach:* Centralized Wrapper.
- *Alternative Considered:* `fetch()` calls scattered inside components.
- *Trade-off:* Centralization ensures consistent security headers, timeout logic, and error handling across the app.

---

### [ARCH-BE-GATEWAY] - API Gateway & Security Controller
**Description:** The entry point for the backend. Handles routing, rate limiting, feature gating, and request validation before passing to services.

| Attribute | Value |
| :--- | :--- |
| **Type** | Middleware / Controller |
| **Static Aspects** | `Routes.js`, `RateLimiter.middleware`, `SecurityHeaders.middleware`, `FeatureGuard.middleware` |
| **Dependencies** | [ARCH-BE-AUTH], [ARCH-BE-SEARCH], [ARCH-BE-USER] |
| **Traceability** | [SW-REQ-042], [SW-REQ-064], [SW-REQ-067], [SW-REQ-068], [SW-REQ-076], [SW-REQ-080], [SW-REQ-051], [SW-REQ-052], [SW-REQ-053] |

**Dynamic Behavior:**
- **Rate Limit:** Tracks IP in Redis. If >10 failures/10min, block 429 ([SW-REQ-064]).
- **Business Limit:** Checks User Tier. If Free & Searches > 3, block ([SW-REQ-042]).
- **Headers:** Injects `Content-Security-Policy`, `Strict-Transport-Security` ([SW-REQ-068]).
- **Trial Logic:** If `(Now - FirstLogin) < 7 days && AccountType == Social`, grant Paid Access ([SW-REQ-051]).
- **Blocking:** If `Role == Free` AND Endpoint is `/api/diet/generate`, return 403 Forbidden ([SW-REQ-053]).

**Interface Definition:**
- `Input`: HTTP Requests + User Context.
- `Output`: JSON Responses or HTTP Errors.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Application-level Middleware (e.g., Express/Gin Middleware).
- *Alternative Considered:* Hardware Load Balancer rules only.
- *Trade-off:* App-level middleware allows logic based on User ID (Free/Paid), not just IP address.

---

### [ARCH-BE-MATH] - Similarity & Optimization Engine
**Description:** The core mathematical computation unit. Handles Vector Cosine Similarity and Linear Programming solver.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | `SimilarityCalculator`, `LPSolver` (Python/C++ binding), `WeightVector` |
| **Dependencies** | [ARCH-DB-MAIN] |
| **Traceability** | [SW-REQ-016], [SW-REQ-017], [SW-REQ-021], [SW-REQ-022], [SW-REQ-023], [SW-REQ-026], [SW-REQ-030] |

**Dynamic Behavior:**
- **Vector Search:** Converts Item Macros [P, C, F] to vector. Calculates Cosine Similarity. Filters <0.40 ([SW-REQ-017]).
- **Optimization:** Runs Simplex or Interior-Point algorithm to solve: Minimize(Calories) subject to Constraints(Target Macros) ([SW-REQ-021], [SW-REQ-022]).
- **Weighted Similarity**: Accepts an optional weights vector {P: float, C: float, F: float} from the user request. Applies these weights to the item vectors before calculating Cosine Similarity, allowing users to prioritize specific macros (e.g., "Protein is 2x more important") ([SW-REQ-017]).
- **Constraint Modeling (SW-REQ-023):** Constructs a matrix where variables are Meal Quantities and constraints are Total P, C, F.
- **Tolerance Handling (SW-REQ-026):** Implements "Relaxed Constraints." Instead of `Sum(Protein) == Target`, it creates inequalities: `Target * 0.95 <= Sum(Protein) <= Target * 1.05`. This ±5% window ensures the solver finds valid solutions even if exact matches don't exist.

**Interface Definition:**
- `Input`: Source Diet (Array of Items), Constraints, Macro Weights.
- `Output`: 3 Optimized Meal Sets (Arrays).

**Alternative Analysis (BP6):**
- *Chosen Approach:* Server-side Library (e.g., Python SciPy/PuLP).
- *Alternative Considered:* Client-side JS Math.
- *Trade-off:* LP solvers are computationally intensive. Server-side execution ensures consistent performance across mobile/desktop ([SW-REQ-080]) and protects the proprietary algorithm.

---

### [ARCH-BE-SEARCH] - Search Service
**Description:** Manages text-based queries, filtering, and data aggregation (e.g., fetching images/macros).

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | `SearchService`, `FilterBuilder` |
| **Dependencies** | [ARCH-DB-MAIN], [ARCH-BE-MATH] |
| **Traceability** | [SW-REQ-010], [SW-REQ-019], [SW-REQ-027], [SW-REQ-029], [SW-REQ-031], [SW-REQ-020] |

**Dynamic Behavior:**
- **Query:** Executes Fuzzy Search on DB.
- **Filter:** Applies Whitelist/Blacklist tags ([SW-REQ-019]).
- **Pagination:** Limits result set to 10 offset N ([SW-REQ-010]).
- **Enrichment:** Calls [ARCH-BE-MATH] to append similarity scores.
- **Zero-Match Handling:** IF a text query returns 0 results:
    1. Extract potential "Category" keywords from the search term (using simple NLP or lookup).
    2. Automatically execute a secondary query for items with that `CategoryTag`.
    3. Return these results with a metadata flag `is_fallback: true` ([SW-REQ-020]).

**Interface Definition:**
- `Input`: Search Term, Tag Filters, Pagination Offset.
- `Output`: Paginated List of Meal Objects.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Database Full-Text Search + Code Filtering.
- *Alternative Considered:* Dedicated Search Engine (ElasticSearch).
- *Trade-off:* Given the requirement for strict mathematical filtering and LP, a combined approach is simpler for the current scale. ElasticSearch can be added later if text search latency degrades.

---

### [ARCH-BE-AUTH] - Authentication & Identity Service
**Description:** Manages user registration, login (Social + Email), JWT issuance, password security and verification workflows.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | `AuthService`, `TokenManager`, `SocialProviderInterface`, `EmailVerifier`, `IdentityManager`, `PasswordHasher`, `OAuthVerifier` |
| **Dependencies** | [ARCH-DB-MAIN], SMTP Service |
| **Traceability** | [SW-REQ-046], [SW-REQ-058], [SW-REQ-059], [SW-REQ-062], [SW-REQ-063], [SW-REQ-065], [SW-REQ-066], [SW-REQ-060], [SW-REQ-061], [SW-REQ-069], [SW-REQ-070], [SW-REQ-074] |

**Dynamic Behavior:**
- **Hashing:** Uses Argon2 for password storage ([SW-REQ-059]).
- **Token Rotation:** Issues Access (15m) and Refresh (7d) tokens ([SW-REQ-063]).
- **Lockout:** Monitors failed attempts; locks account after 5 failures ([SW-REQ-065]).
- **Registration:** Checks DB for existing email. If found, reject ([SW-REQ-060]). If new, create pending record.
- **Verification:** Sends Email. Blocks paid features until link clicked ([SW-REQ-070]).
- **Reset:** Generates 1hr token. Emails link. Updates password hash ([SW-REQ-069]).
- **Consent:** Records boolean `consent_given` and timestamp at signup ([SW-REQ-074]).
- **Registration (SW-REQ-058/060):**
    1. `SELECT count(*) FROM users WHERE email = ?`. If > 0, throw "Duplicate" Error.
    2. Generate Salt (16 bytes).
    3. Hash Password using **Argon2id** (Memory: 64MB, Iterations: 3) ([SW-REQ-059]).
    4. Insert User with `verified = false`.
    5. Generate unique `verification_token` (UUID) and trigger Email.
- **Verification (SW-REQ-070):**
    - Endpoint `/verify?token=XYZ` sets `verified = true` in DB.
- **Social Login (SW-REQ-046):**
    - Verifies the incoming `id_token` signature against Google/Apple public keys.
    - If valid and email not in DB, Auto-Register.

**Interface Definition:**
- `Input`: Credentials (Email/Pass or OAuth Token).
- `Output`: HttpOnly Cookies (JWT), Emails

**Alternative Analysis (BP6):**
- *Chosen Approach:* JWT in HttpOnly Cookies.
- *Alternative Considered:* JWT in LocalStorage.
- *Trade-off:* HttpOnly Cookies prevent XSS attacks from stealing tokens, adhering to the high security priority of [SW-REQ-062].

---

### [ARCH-BE-PAY] - Payment Service Interface
**Description:** Encapsulates Stripe interactions to manage subscriptions and payment intents without touching raw card data.

| Attribute | Value |
| :--- | :--- |
| **Type** | Interface / Adapter |
| **Static Aspects** | `StripeAdapter`, `WebhookHandler` |
| **Dependencies** | Stripe API (External) |
| **Traceability** | [SW-REQ-044], [SW-REQ-045], [SW-REQ-050] |

**Dynamic Behavior:**
- **Handover:** Generates `client_secret` for Frontend Stripe Element.
- **Webhook:** Listens for `invoice.payment_succeeded`. Updates User Role in DB asynchronously ([SW-REQ-045]).

**Interface Definition:**
- `Input`: Plan ID, Webhook Event.
- `Output`: Payment Intent Secret, DB Update.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Webhook-based synchronization.
- *Alternative Considered:* Synchronous update after client-side success callback.
- *Trade-off:* Webhooks are robust against browser crashes/closes during the payment redirection, ensuring [SW-REQ-045].

---


### [ARCH-DB-MAIN] - Relational Database
**Description:** The primary storage for Users, Meals, Ingredients, and Subscriptions.

| Attribute | Value |
| :--- | :--- |
| **Type** | Database (PostgreSQL recommended) |
| **Static Aspects** | Schemas: `Users`, `Meals`, `Tags`, `History`, `Micronutrients` |
| **Dependencies** | None |
| **Traceability** | [SW-REQ-033], [SW-REQ-034], [SW-REQ-035], [SW-REQ-043], [SW-REQ-047], [SW-REQ-073], [SW-REQ-075], [SW-REQ-083], [SW-REQ-036], [SW-REQ-037], [SW-REQ-038] |

**Dynamic Behavior:**
- **Encryption:** Storage encrypted at rest (AES-256) ([SW-REQ-075]).
- **Backup:** Automated daily snapshots ([SW-REQ-083]).
- **Isolation:** Queries for custom items always include `WHERE user_id = X` ([SW-REQ-043]).
- **Schema Enforcement:** `Meals` table includes columns for `unit_weight_g` ([SW-REQ-036]) and `functionality_tags` (Array/JSONB) ([SW-REQ-037]).
- **Separation:** Micronutrients stored in separate column/table to exclude from Vector Search ([SW-REQ-038]).

**Interface Definition:**
- `Input`: SQL Queries.
- `Output`: Result Sets.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Relational (SQL).
- *Alternative Considered:* NoSQL (Document Store).
- *Trade-off:* The data model relies heavily on relationships (Recipe -> Ingredients) and strict structure (Macros per 100g). SQL ensures integrity better than NoSQL here.

---

### [ARCH-INFRA-LOG] - Centralized Logging
**Description:** Aggregates logs from all backend services for auditing and debugging.

| Attribute | Value |
| :--- | :--- |
| **Type** | Infrastructure Service |
| **Static Aspects** | Logger Interface (e.g., Winston/Zap) |
| **Dependencies** | None |
| **Traceability** | [SW-REQ-084] |

**Dynamic Behavior:**
- **Trigger:** On every API Request, Error, or Auth Event.
- **Masking:** Scrub PII/Passwords before writing.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Structured Logging (JSON) to centralized collector.
- *Alternative Considered:* Text files on server disk.
- *Trade-off:* Centralized JSON logs allow querying and alerts, essential for meeting security audit requirements ([SW-REQ-084]).

---

## [ARCH-FE-LIST-MGR] - List State Manager
**Description:** A client-side state container (e.g., Redux Slice or Context) responsible for managing the user's active "Ingredient List" and "Meal List," including accumulation and aggregation logic.

| Attribute | Value |
| :--- | :--- |
| **Type** | Module / State Manager |
| **Static Aspects** | `ListContext`, `AggregatorReducer` |
| **Dependencies** | None |
| **Traceability** | [SW-REQ-005], [SW-REQ-006] |

**Dynamic Behavior:**
- **Accumulation:** Listens for 'Enter' key events on autocomplete; pushes selected item ID and default quantity to the active array ([SW-REQ-005]).
- **Aggregation:** When in 'Meal List' mode, groups multiple meal objects into a simplified "Day" object for diet generation ([SW-REQ-006]).

**Interface Definition:**
- `Input`: Selected Item Object, Mode (Ingredient/Meal).
- `Output`: Updated State Array.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Global Client-Side State (Context/Redux).
- *Alternative Considered:* Component Local State (`useState`).
- *Trade-off:* Global state is required because the list must persist when navigating between the Search view and the Diet Generation view.

---

## [ARCH-FE-UNIT-ENGINE] - Unit Conversion & Math Engine
**Description:** A pure utility library responsible for all client-side numerical transformations, including metric/imperial swapping, real-time scaling, and comparative quantity math.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service (Utility) |
| **Static Aspects** | `UnitConverter.js`, `MacroScaler.js` |
| **Dependencies** | [ARCH-FE-CACHE] (for User Preference) |
| **Traceability** | [SW-REQ-020], [SW-REQ-028], [SW-REQ-032], [SW-REQ-036], [SW-REQ-039], [SW-REQ-040], [SW-REQ-041] |

**Dynamic Behavior:**
- **Reactive Scaling:** On quantity input change, multiplies base (100g) macros by `(UserQty / 100)` ([SW-REQ-020]).
- **Contextual Display:** Checks Global Preference (`imperial` vs `metric`). If Imperial, applies `x 0.035` (solids) or `x 0.033` (liquids) ([SW-REQ-032]).
- **Comparison:** Calculates `(TargetCal / ItemCal) * 100g` to find replacement weight ([SW-REQ-028]).

**Interface Definition:**
- `Input`: Base Value, Input Quantity, User Preference Enum.
- `Output`: Formatted Value (String), Scaled Value (Float).

**Alternative Analysis (BP6):**
- *Chosen Approach:* Centralized Math Engine.
- *Alternative Considered:* Inline calculations in UI components.
- *Trade-off:* Centralization prevents "floating point drift" errors and ensures [SW-REQ-041] (Global update) is applied consistently across all views.

---

## [ARCH-FE-SIDEBAR] - Sidebar & Favorites Controller
**Description:** Manages the visibility and content of the collateral UI sidebar, including the "Favorites" list and integration with the Search History.

| Attribute | Value |
| :--- | :--- |
| **Type** | Component / UI Controller |
| **Static Aspects** | `SidebarContainer`, `FavoritesManager` |
| **Dependencies** | [ARCH-FE-CACHE], [ARCH-FE-THEME], [ARCH-FE-SEARCH] |
| **Traceability** | [SW-REQ-013], [SW-REQ-049] |

**Dynamic Behavior:**
- **Toggle:** Responds to burger-menu click to slide in/out ([SW-REQ-013]).
- **Pinning:** On "Star" click, moves item from "History" stack to permanent "Favorites" list in LocalStorage ([SW-REQ-049]).

**Interface Definition:**
- `Input`: Toggle Events, History Data.
- `Output`: Sidebar UI, Trigger Search Event.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Collapsible Overlay (Mobile) / Persistent Column (Desktop).
- *Alternative Considered:* Separate "History" page.
- *Trade-off:* Keeping it as a sidebar maintains the user's context within the Search interface, improving workflow efficiency.

---

## [ARCH-FE-A11Y] - Accessibility & Focus Manager
**Description:** A dedicated hook/service to manage global keyboard traps, focus navigation, and screen reader announcements.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service (Frontend) |
| **Static Aspects** | `useFocusTrap`, `KeyboardNavigator` |
| **Dependencies** | DOM APIs |
| **Traceability** | [SW-REQ-086] |

**Dynamic Behavior:**
- **Navigation:** Listens for Arrow Keys, Tab, Enter, and Esc to move focus logically between Search Results and Filters ([SW-REQ-086]).
- **Announcements:** Injects updates into an `aria-live` region when search results load.

**Interface Definition:**
- `Input`: Keydown Events.
- `Output`: `document.activeElement` updates.

**Alternative Analysis (BP6):**
- *Chosen Approach:* programmatic Focus Management.
- *Alternative Considered:* Native Tab order only.
- *Trade-off:* Native tab order is insufficient for complex grids like "Search Results," requiring manual arrow-key logic for WCAG compliance.

---

## [ARCH-BE-ADMIN] - Administration Service
**Description:** Provides the backend logic for the Admin Panel, handling CRUD operations for manual items, external imports, and tag management.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | `AdminController`, `ImportSanitizer` |
| **Dependencies** | [ARCH-DB-MAIN], [ARCH-BE-AUTH] |
| **Traceability** | [SW-REQ-054], [SW-REQ-055], [SW-REQ-056], [SW-REQ-057] |

**Dynamic Behavior:**
- **RBAC:** Verifies `Role == 'Admin'` before execution ([SW-REQ-054]).
- **Import:** Fetches data from external source, maps fields to local schema, and saves ([SW-REQ-055]).
- **Tagging:** Updates the global `Tags` table ([SW-REQ-057]).

**Interface Definition:**
- `Input`: Admin API Token, Item Data.
- `Output`: Database Confirmation.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Dedicated Service.
- *Alternative Considered:* Mixing Admin logic into Search Service.
- *Trade-off:* Separation of concerns prevents admin-only operations (like `DELETE item`) from accidentally being exposed to public endpoints.

---

## [ARCH-BE-USER] - User Data Service
**Description:** Manages the user's private data profile, including GDPR data export requests and account deletion cleanup.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | `UserController`, `DataExporter` |
| **Dependencies** | [ARCH-DB-MAIN] |
| **Traceability** | [SW-REQ-072] |

**Dynamic Behavior:**
- **Export:** Queries all tables (History, Lists, Preferences) linked to `UserID`, serializes to JSON, and initiates download ([SW-REQ-072]).

**Interface Definition:**
- `Input`: User ID.
- `Output`: JSON Blob.

**Alternative Analysis (BP6):**
- *Chosen Approach:* On-demand generation.
- *Alternative Considered:* Pre-generated nightly exports.
- *Trade-off:* On-demand ensures the data is real-time and reduces storage costs for stale export files.

---

## [ARCH-INFRA-SCALE] - Scaling & Reliability Config
**Description:** Defines the infrastructure-level configuration for load balancers and container orchestration to meet SLA and capacity requirements.

| Attribute | Value |
| :--- | :--- |
| **Type** | Infrastructure / Middleware |
| **Static Aspects** | `K8sHPA` (Horizontal Pod Autoscaler), `NginxConfig` |
| **Dependencies** | Cloud Provider |
| **Traceability** | [SW-REQ-081], [SW-REQ-082] |

**Dynamic Behavior:**
- **Autoscale:** Monitors CPU/RAM. If >70%, spins up new Container Replicas to handle >1000 concurrent users ([SW-REQ-082]).
- **Health Check:** Liveness probes ensure failed pods are restarted to maintain 99.9% uptime ([SW-REQ-081]).

**Interface Definition:**
- `Input`: Metrics (CPU/Memory).
- `Output`: Replica Count.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Kubernetes Autoscaling.
- *Alternative Considered:* Static Server Provisioning.
- *Trade-off:* Static provisioning wastes money during low-traffic periods; autoscaling is cost-effective and compliant.

---

## [ARCH-FE-LAYOUT] - Global Layout Orchestrator
**Description:** Implements the application's responsive structural grid, enforcing the "Skeleton" defined in the Style Guide. It manages the positioning of the Sidebar (Left), Search (Middle), and Content Areas based on viewport breakpoints.

| Attribute | Value |
| :--- | :--- |
| **Type** | Component / Layout Engine |
| **Static Aspects** | `AppLayout.tsx`, `GridSystem.css` |
| **Dependencies** | [ARCH-FE-SIDEBAR], [ARCH-FE-SEARCH] |
| **Traceability** | [SW-REQ-089] (Style Guide Sec 4), [SW-REQ-007], [SW-REQ-013], [SW-REQ-014] |

**Dynamic Behavior:**
- **Breakpoint Logic:** Monitors viewport width.
    - `< 640px` (Mobile): Switches to Single Column; Sidebar becomes an overlay drawer.
    - `> 1024px` (Desktop): Enforces 12-column Grid; Sidebar is persistent in left columns; Max-width restricted to `1280px`.
- **Z-Ordering:** Ensures Search Interface sits vertically on top of Macronutrient toggles ([SW-REQ-007]).

**Interface Definition:**
- `Input`: Window Resize Events.
- `Output`: CSS Grid Class assignments.

**Alternative Analysis (BP6):**
- *Chosen Approach:* CSS Grid + Flexbox Wrapper.
- *Alternative Considered:* JavaScript-based window resize listeners calculating pixel widths.
- *Trade-off:* CSS Grid is native, hardware-accelerated, and prevents layout thrashing, ensuring the responsive requirements ([SW-REQ-014]) are met performantly.

---

## [ARCH-FE-UI-KIT] - Atomic Design System
**Description:** A library of reusable, standardized UI components that strictly enforce the Typography, Shape, and Interaction states defined in the Style Guide.

| Attribute | Value |
| :--- | :--- |
| **Type** | Module / Component Library |
| **Static Aspects** | `Button`, `Input`, `Typography`, `SkeletonLoader` |
| **Dependencies** | [ARCH-FE-THEME] |
| **Traceability** | [SW-REQ-089] (Style Guide Sec 3, 5), [SW-REQ-008], [SW-REQ-012] |

**Dynamic Behavior:**
- **Typography Enforcement:** Global injection of `Inter` (Body) and `Roboto Mono` (Data) fonts.
- **Input States:** Inputs render with `#E0E0E0` border; Focus state applies Primary Color (`#166534`/`#4ADE80`) ([Style Guide Sec 5]).
- **Loading:** Renders animated "Skeleton" blocks instead of spinners for search results ([Style Guide Sec 5]).

**Interface Definition:**
- `Input`: Props (Variant, Label, State).
- `Output`: Rendered HTML Elements with scoped styles.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Custom Component Library.
- *Alternative Considered:* Third-party library (e.g., Material UI, Bootstrap) out of the box.
- *Trade-off:* A custom kit ensures exact compliance with the specific Hex codes and Border Radius (4px) defined in the Style Guide without fighting framework overrides.

---

## [ARCH-FE-PAGINATION] - Search Result Pagination Control
**Description:** A stateless UI component responsible for calculating page offsets and rendering navigation controls (Next/Prev/Page Numbers) based on total result count and current limit.

| Attribute | Value |
| :--- | :--- |
| **Type** | Component (UI) |
| **Static Aspects** | `PaginationControls.tsx`, `usePagination` |
| **Dependencies** | [ARCH-FE-SEARCH] |
| **Traceability** | [SW-REQ-010] |

**Dynamic Behavior:**
- **State Calculation:** Computes `TotalPages = Math.ceil(TotalCount / 10)`.
- **Interaction:** On 'Next' click, increments internal index and emits `onPageChange(newOffset)` event to the parent controller.
- **Disabling:** Disables 'Next' button if `CurrentPage == TotalPages` to prevent out-of-bounds requests.

**Interface Definition:**
- `Input`: Total Count (Int), Current Offset (Int), Loading State (Bool).
- `Output`: New Offset Event.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Controlled Component.
- *Alternative Considered:* Infinite Scroll.
- *Trade-off:* Numbered pagination ([SW-REQ-010]) allows users to bookmark or return to specific result sets more reliably than infinite scroll, which resets on navigation.

---

## [ARCH-FE-MEDIA-HANDLER] - Intelligent Media Resolver
**Description:** A wrapper component for image rendering that implements the fallback logic for missing assets based on item metadata.

| Attribute | Value |
| :--- | :--- |
| **Type** | Component (Utility) |
| **Static Aspects** | `SmartImage.tsx`, `CategoryPlaceholderMap` |
| **Dependencies** | [ARCH-FE-THEME] |
| **Traceability** | [SW-REQ-011], [SW-REQ-012] |

**Dynamic Behavior:**
- **Source Validation:** Attempts to load the primary `image_url` from the item record.
- **Fallback Logic:** If `image_url` is null OR triggers an `onError` event, the component reads the Item's `CategoryTag`.
- **Resolution:** Selects the corresponding static asset (e.g., `assets/placeholders/dairy.png`) from the `CategoryPlaceholderMap` ([SW-REQ-012]).

**Interface Definition:**
- `Input`: Image Source URL, Category Tag.
- `Output`: Rendered `<img>` element.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Client-Side Fallback.
- *Alternative Considered:* Server-Side Placeholder Injection.
- *Trade-off:* Client-side handling allows the server to send `null`, reducing bandwidth if the user already has the placeholder assets cached locally.

---

## [ARCH-FE-SCORE-INDICATOR] - Similarity Visualizer
**Description:** Responsible for mapping the raw Cosine Similarity score to the specific visual hierarchy (Colors and Icons) defined in the requirements.

| Attribute | Value |
| :--- | :--- |
| **Type** | Component (UI) |
| **Static Aspects** | `ScoreBadge.tsx`, `ScoreThresholds` |
| **Dependencies** | [ARCH-BE-SEARCH] (for server-hosted emoji URLs) |
| **Traceability** | [SW-REQ-018] |

**Dynamic Behavior:**
- **Thresholding:** Evaluates score `S`:
    - `S ≥ 0.85` -> Applies Class `text-green-600` + Fetches `star_icon.png`.
    - `0.70 ≤ S < 0.84` -> Applies Class `text-green-400` + Fetches `sparkle_icon.png`.
    - `0.55 ≤ S < 0.69` -> Applies Class `text-yellow-500` + Fetches `thumbs_up_icon.png`.
    - `S < 0.55` -> Applies Class `text-red-500` + Fetches `thumbs_down_icon.png`.
- **Asset Retrieval:** Uses server-hosted URLs for icons as mandated by [SW-REQ-018] ("save emojis as images on server").

**Interface Definition:**
- `Input`: Similarity Score (Float 0.0-1.0).
- `Output`: Styled Badge UI.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Discrete Component with Server Assets.
- *Alternative Considered:* CSS Pseudo-elements / System Emojis.
- *Trade-off:* Using server assets ensures cross-platform consistency (Android vs iOS emojis differ), strictly adhering to the "UI stored properly" clause of [SW-REQ-018].

---

## [ARCH-FE-LIST-MGR] - List State & Aggregation Manager
**Description:** A client-side state container (Context/Redux) that manages the user's active building lists. It distinguishes between "Ingredient Mode" (adding atomic items) and "Meal List Mode" (aggregating full meal objects).

| Attribute | Value |
| :--- | :--- |
| **Type** | Module / State Manager |
| **Static Aspects** | `ListContext.tsx`, `AggregationReducer.ts` |
| **Dependencies** | None |
| **Traceability** | [SW-REQ-005], [SW-REQ-006] |

**Dynamic Behavior:**
- **Event Listener:** Attaches a global listener for the 'Enter' key within the search context.
- **Ingredient Mode (SW-REQ-005):** When 'Enter' is pressed on an autocomplete item, pushes the Item ID + Default Qty (100g) to the `activeIngredients` array.
- **Meal List Mode (SW-REQ-006):** When in Meal Mode, selected items are pushed to the `activeDay` array. The manager calculates the sum of all macros in `activeDay` to represent a full day's diet.

**Interface Definition:**
- `Input`: Selected Item, Mode Enum (`Ingredient` | `Meal`).
- `Output`: Updated State Arrays, Calculated Daily Totals.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Client-side Accumulation.
- *Alternative Considered:* Server-side "Cart" session.
- *Trade-off:* Client-side management allows for instant UI updates and accumulation without network latency for every item added, providing a snappier UX for building lists quickly.

---

## [ARCH-FE-PAGINATION] - Search Result Pagination Control
**Description:** A stateless UI component responsible for calculating page offsets and rendering navigation controls based on the total result count returned by the API.

| Attribute | Value |
| :--- | :--- |
| **Type** | Component (UI) |
| **Static Aspects** | `PaginationControls.tsx`, `usePagination` hook |
| **Dependencies** | [ARCH-FE-SEARCH] |
| **Traceability** | [SW-REQ-010] |

**Dynamic Behavior:**
- **Offset Calculation:** Computes the API `offset` parameter: `(CurrentPage - 1) * 10`.
- **Limit Enforcement:** Enforces the hard limit of 10 items per page view ([SW-REQ-010]).
- **Navigation:** Emits event `onPageChange(newOffset)` which triggers [ARCH-FE-SEARCH] to re-fetch data.

**Interface Definition:**
- `Input`: Total Item Count, Current Page Index.
- `Output`: UI Events (Next/Prev).

**Alternative Analysis (BP6):**
- *Chosen Approach:* Indexed Pagination.
- *Alternative Considered:* "Load More" (Append) button.
- *Trade-off:* Indexed pagination is required to allow users to navigate back and forth through search results without losing their place, which is critical when comparing nutritional values across many items.

---

## [ARCH-FE-MEDIA-HANDLER] - Placeholder Resolution Service
**Description:** A robust media handling component that intercepts image load failures and injects category-specific assets to ensure visual consistency.

| Attribute | Value |
| :--- | :--- |
| **Type** | Component (Utility) |
| **Static Aspects** | `SmartImage.tsx`, `CategoryAssetMap.json` |
| **Dependencies** | [ARCH-FE-THEME] |
| **Traceability** | [SW-REQ-012] |

**Dynamic Behavior:**
- **Detection:** Checks if the item object has a null `image_url` OR listens for the DOM `onError` event on the `<img>` tag.
- **Resolution:** If missing/error, reads the Item's `category_tag` (e.g., "dairy", "meat").
- **Injection:** Swaps the `src` attribute with a local static asset (e.g., `/assets/placeholders/dairy_generic.png`) ([SW-REQ-012]).

**Interface Definition:**
- `Input`: Image URL (nullable), Category Tag.
- `Output`: Valid Image Source (Remote or Local).

**Alternative Analysis (BP6):**
- *Chosen Approach:* Client-side Logic.
- *Alternative Considered:* Backend Default Image URL.
- *Trade-off:* Client-side logic reduces bandwidth (no need to download a placeholder image if it's already in the app bundle) and prevents broken images if the backend logic fails to identify a missing external image.

---

## [ARCH-FE-PREF-WEIGHTS] - Macro Priority Controller
**Description:** A UI/Logic component that allows users to define the relative importance of Proteins, Carbs, and Fats via sliders or input fields.

| Attribute | Value |
| :--- | :--- |
| **Type** | Component (UI + State) |
| **Static Aspects** | `PrioritySliders.tsx`, `WeightContext` |
| **Dependencies** | [ARCH-FE-CACHE] |
| **Traceability** | [SW-REQ-017] |

**Dynamic Behavior:**
- **Capture:** Listens for slider changes (Range 0.1 - 2.0).
- **Persistence:** Saves the preferred weight vector to `localStorage` (via [ARCH-FE-CACHE]).
- **Injection:** Injects the current weights into every Search Request payload to trigger the Weighted Similarity logic in [ARCH-BE-MATH].

**Interface Definition:**
- `Input`: User Slider Events.
- `Output`: Weight Vector `{p, c, f}`.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Client-side Preference Injection.
- *Alternative Considered:* Server-side User Profile setting.
- *Trade-off:* Keeping this in client state allows for rapid experimentation ("What if I prioritize Fat now?") without needing permanent DB writes for every adjustment.

---

## [ARCH-FE-BADGE-SYS] - Visual Hierarchy & Badge System
**Description:** A specialized display component responsible for rendering "Best Match" badges and visual hierarchy indicators based on strict score thresholds.

| Attribute | Value |
| :--- | :--- |
| **Type** | Component (UI) |
| **Static Aspects** | `MatchBadge.tsx`, `HierarchyResolver` |
| **Dependencies** | [ARCH-FE-THEME] |
| **Traceability** | [SW-REQ-018], [SW-REQ-019] |

**Dynamic Behavior:**
- **Hierarchy Rendering:**
    - If Score > 0.90: Renders "Best Match" Badge with Accent Color (Soft Amber) ([SW-REQ-018]).
    - Else: Renders standard similarity indicators.
- **Score Display:** Formats the raw float (e.g., `0.876`) into a readable percentage or decimal (`0.88` / `88%`) based on configuration ([SW-REQ-019]).

**Interface Definition:**
- `Input`: Similarity Score (Float).
- `Output`: Badge UI Element.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Component-level Logic.
- *Alternative Considered:* CSS Classes only.
- *Trade-off:* A dedicated component is needed to handle the logic of "Best Match" exclusivity (only showing it on the top item) vs standard coloring.

---

## [ARCH-FE-PROMO] - Engagement & Offer Controller
**Description:** Monitors search results to inject "Special Offer" notifications or marketing prompts when high-quality matches are found.

| Attribute | Value |
| :--- | :--- |
| **Type** | Controller / Interceptor |
| **Static Aspects** | `OfferTrigger.ts`, `PromoModal.tsx` |
| **Dependencies** | [ARCH-FE-SEARCH] |
| **Traceability** | [SW-REQ-021] |

**Dynamic Behavior:**
- **Trigger Condition:** Listens to search results. IF `BestItem.score > 0.95` AND `User.hasSeenOffer == false`, THEN trigger the "Special Offer" UI ([SW-REQ-021]).
- **Dismissal:** Updates local session state to prevent spamming the offer on every search.

**Interface Definition:**
- `Input`: Search Result Set.
- `Output`: UI Modal / Toast Notification.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Client-side Trigger.
- *Alternative Considered:* Backend "Marketing" Flag in response.
- *Trade-off:* Client-side triggers allow for UI-specific state tracking (session/local storage) to manage frequency capping without bloating the API response.

---

## [ARCH-FE-DIET-WIZARD] - Diet Configuration Interface
**Description:** A multi-step form wizard allowing the user to define the boundary conditions for the diet generation algorithm, including numeric macro targets and dietary exclusions.

| Attribute | Value |
| :--- | :--- |
| **Type** | Component (UI/Form) |
| **Static Aspects** | `DietWizard.tsx`, `TargetValidator` |
| **Dependencies** | [ARCH-FE-CACHE] (Last used targets) |
| **Traceability** | [SW-REQ-022], [SW-REQ-027] |

**Dynamic Behavior:**
- **Target Capture:** Accepts user inputs for Daily Protein, Carbs, and Fat (in grams). Validates that values are positive integers ([SW-REQ-022]).
- **Constraint Selection:** Provides toggles for Dietary Restrictions (e.g., Gluten-Free, Vegan). These are mapped to internal Tag IDs for API transmission ([SW-REQ-027]).
- **Persistence:** Saves the configuration to `localStorage` so repeat generations don't require re-entry.

**Interface Definition:**
- `Input`: User Form Data.
- `Output`: JSON Payload `{ targets: {p,c,f}, filters: [tags] }`.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Explicit Wizard Step.
- *Alternative Considered:* Inferring targets from user profile stats (Height/Weight).
- *Trade-off:* Explicit entry is chosen because users often cycle diet types (Bulking vs Cutting) rapidly; profile-based inference is too rigid for a "Meal Swapp" use case.

---

## [ARCH-BE-DIET-SVC] - Diet Orchestration Service
**Description:** The dedicated service layer responsible for gathering candidate meals, applying pre-optimization filters, and orchestrating the Linear Programming workflow.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | `DietGenerator`, `CandidateSelector` |
| **Dependencies** | [ARCH-DB-MAIN], [ARCH-BE-MATH] |
| **Traceability** | [SW-REQ-023], [SW-REQ-027] |

**Dynamic Behavior:**
- **Candidate Fetching:** Queries [ARCH-DB-MAIN] for a pool of "Valid Meals." Applies User Constraints (e.g., `WHERE tag NOT IN ('gluten')`) *before* optimization to reduce the solution space ([SW-REQ-027]).
- **Orchestration:** Passes the Candidate Pool and User Targets to [ARCH-BE-MATH].
- **Response Formatting:** Maps the Solver's mathematical output (Meal IDs) back to full Meal Objects for the client.

**Interface Definition:**
- `Input`: User Targets, Constraints.
- `Output`: Structured Diet Plan (List of Meals).

**Alternative Analysis (BP6):**
- *Chosen Approach:* Dynamic Candidate Filtering.
- *Alternative Considered:* Solving on the entire database.
- *Trade-off:* Filtering candidates by dietary restriction *before* the math step significantly reduces CPU time and ensures strict compliance with safety requirements (Allergies).

---

## [ARCH-FE-DIET-RESULT] - Diet Solution Viewer
**Description:** A complex display component responsible for rendering the generated diet, breaking it down meal-by-meal, and showing the "Total vs Target" comparison.

| Attribute | Value |
| :--- | :--- |
| **Type** | Component (UI) |
| **Static Aspects** | `DietBreakdown.tsx`, `MacroSummaryChart` |
| **Dependencies** | [ARCH-FE-THEME] |
| **Traceability** | [SW-REQ-028] |

**Dynamic Behavior:**
- **Iterative Rendering:** Maps through the returned solution array to display each meal card (Image, Name, Qty) ([SW-REQ-028]).
- **Summary Calculation:** client-side recalculation of the total P/C/F of the solution to display a "Match Accuracy" chart (e.g., "98% of Protein Target").

**Interface Definition:**
- `Input`: Diet Plan Object.
- `Output`: Rendered List & Charts.

**Alternative Analysis (BP6):**
- *Chosen Approach:* List View with Summary Header.
- *Alternative Considered:* Calendar View.
- *Trade-off:* The requirement focuses on a "One-Day Diet" ([SW-REQ-006]), so a vertical list is more space-efficient on mobile than a full calendar grid.

---

## [ARCH-FE-AUTH-WIDGET] - Authentication Interface & Logic
**Description:** The client-side controller for all user identification flows. It manages the Toggle between "Login" and "Register" forms, input validation (Password Strength), and the Social Login SDK integrations.

| Attribute | Value |
| :--- | :--- |
| **Type** | Component (UI + Logic) |
| **Static Aspects** | `AuthModal.tsx`, `SocialLoginButton.tsx`, `ValidationUtils` |
| **Dependencies** | [ARCH-FE-NET], Google/Apple SDKs |
| **Traceability** | [SW-REQ-046], [SW-REQ-058], [SW-REQ-060], [SW-REQ-061] |

**Dynamic Behavior:**
- **Password Strength:** Real-time regex validation on the registration input. Requires: `Min 8 chars, 1 Uppercase, 1 Number`. Visual feedback provided before submission.
- **Social Flow:**
    1. User clicks "Continue with Google".
    2. Component calls `GoogleAuth.signIn()`.
    3. On success, receives `id_token`.
    4. POSTs `id_token` to Backend `/api/auth/social`.
- **Error Handling:** Displays specific messages for "User already exists" ([SW-REQ-060]) or "Invalid Credentials."

**Interface Definition:**
- `Input`: User Credentials or OAuth Token.
- `Output`: Auth Success Event (triggers Router redirect).

**Alternative Analysis (BP6):**
- *Chosen Approach:* Dedicated Auth Widget/Modal.
- *Alternative Considered:* Redirect to separate `/login` page.
- *Trade-off:* Using a Modal/Widget allows the user to log in contextually (e.g., when trying to "Save" a diet) without losing their current work/search state.

---

## [ARCH-BE-SESSION] - Token & Session Manager
**Description:** Dedicated to the generation, validation, rotation, and invalidation of JSON Web Tokens (JWT). It isolates session security logic from identity logic.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service / Middleware |
| **Static Aspects** | `TokenSigner`, `RefreshRotationStrategy` |
| **Dependencies** | Redis (Optional, for blocklist) or DB |
| **Traceability** | [SW-REQ-062], [SW-REQ-063], [SW-REQ-066] |

**Dynamic Behavior:**
- **Minting (SW-REQ-063):**
    - Creates `AccessToken` (Exp: 15 mins). Payload: `{uid, role}`. Signed with HS256/RS256.
    - Creates `RefreshToken` (Exp: 7 days). Stored as Hash in DB linked to User.
    - Sets Cookies: `HttpOnly; Secure; SameSite=Strict` ([SW-REQ-062]).
- **Rotation:**
    - On `/refresh` call: Validates old RefreshToken. Checks DB.
    - If valid: **Revokes old token**, Mints NEW Access + NEW Refresh tokens (Rotation).
    - If invalid (Reuse Attempt): **Revokes ALL tokens** for that user (Theft detection).
- **Logout (SW-REQ-066):**
    - Deletes RefreshToken from DB.
    - Clears Cookies on Client Response.

**Interface Definition:**
- `Input`: User ID, Roles.
- `Output`: Cookie Set Headers.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Refresh Token Rotation.
- *Alternative Considered:* Long-lived Access Tokens.
- *Trade-off:* Rotation limits the damage of a stolen refresh token to a single use and allows the server to detect token theft (reuse) immediately, significantly improving security over static tokens.

---

## [ARCH-FE-CHECKOUT] - Payment & Subscription UI
**Description:** A secure UI component responsible for rendering the pricing tiers and encapsulating the Stripe Elements (Input Fields). It handles the client-side tokenization of payment credentials.

| Attribute | Value |
| :--- | :--- |
| **Type** | Component (UI + Security) |
| **Static Aspects** | `CheckoutForm.tsx`, `PricingTable.tsx`, `StripeProvider` |
| **Dependencies** | [ARCH-FE-NET], Stripe JS SDK |
| **Traceability** | [SW-REQ-050] (Original), [SW-REQ-044] (Tokenization), [SW-REQ-043] (Checkout Integration) |

**Dynamic Behavior:**
- **Tier Selection:** Displays "Monthly ($3)" and "Yearly ($25)" options. On selection, requests a `PaymentIntent` from the backend ([SW-REQ-050]).
- **Tokenization:** Mounts the Stripe Elements `CardElement`. On form submit, calls `stripe.confirmCardPayment()`. This sends card data directly to Stripe, never hitting the App Server ([SW-REQ-044]).
- **Success Handling:** On promise resolution (success), displays the Success State and triggers a session refresh to update User Role.

**Interface Definition:**
- `Input`: User Selection, Card Data (Hosted Iframe).
- `Output`: Payment Success Event.

**Alternative Analysis (BP6):**
- *Chosen Approach:* Stripe Elements (Client-side Tokenization).
- *Alternative Considered:* Raw Form POST to Backend.
- *Trade-off:* Elements is mandatory for PCI-DSS SAQ A compliance. Sending raw data to the backend would drastically increase compliance scope and security risk.

---

## [ARCH-BE-SUB-MGR] - Subscription Lifecycle Manager
**Description:** The domain service responsible for the business logic of subscriptions, including handling cancellations, auto-renewal monitoring, and tier transitions (e.g., Paid -> Free upon expiration).

| Attribute | Value |
| :--- | :--- |
| **Type** | Service (Domain) |
| **Static Aspects** | `SubscriptionService`, `PlanManager` |
| **Dependencies** | [ARCH-DB-MAIN], [ARCH-BE-PAY] (Adapter) |
| **Traceability** | [SW-REQ-046] (Cancellation), [SW-REQ-047] (Auto-Renewal), [SW-REQ-045] (Failure Handling) |

**Dynamic Behavior:**
- **Cancellation (SW-REQ-046):**
    - API Endpoint `/api/subs/cancel`.
    - Calls Stripe Adapter to set `cancel_at_period_end = true`.
    - Updates DB status to "Pending Cancellation". Access remains active until `current_period_end`.
- **Renewal (SW-REQ-047):**
    - Passive: Relies on Webhooks from `[ARCH-BE-PAY]`.
    - Active Check: If a user logs in and `subscription_end < NOW`, downgrades User Role to 'Free' immediately.
- **Failure Handling (SW-REQ-045):**
    - Listens for `invoice.payment_failed` webhook.
    - Triggers "Dunning" email flow and marks DB status as "Past Due".

**Interface Definition:**
- `Input`: User Requests, Webhook Events.
- `Output`: Subscription Status Updates.

**Alternative Analysis (BP6):**
- *Chosen Approach:* State Machine based on Webhooks.
- *Alternative Considered:* Polling Stripe API cron job.
- *Trade-off:* Webhook-driven architecture is real-time and avoids API rate limits, whereas polling is inefficient and can lead to sync delays where a user pays but doesn't get access immediately.
