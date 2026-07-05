## FILE: DESIGN-001.md
**Traceability:** ARCH-001

**Static aspects covered:** SearchView, SidebarComponent, ResultsGrid, AutocompleteDropdown, ThemeProvider, OfflineBanner, SettingsPanel, LocalStorageManager, ServiceWorker.

### 0. Static Aspect Responsibilities
- `SearchView`: owns catalog query input, Substitution Input composition, filter composition, debounce timing, and result loading orchestration.
- `SidebarComponent`: owns navigation between Catalog Search, Substitution Search, Daily Diet Alternative Search, saved filters, settings entry points, and responsive collapse behavior.
- `ResultsGrid`: owns result card layout, pagination controls, image fallback display, and similarity badge rendering.
- `AutocompleteDropdown`: owns ranked suggestion display, keyboard focus movement, selection, and dismissal rules.
- `ThemeProvider`: owns resolved theme state and delegates token application to ARCH-016.
- `OfflineBanner`: owns online/offline and stale-data indicators.
- `SettingsPanel`: owns unit preference and theme preference controls.
- `LocalStorageManager`: owns client persistence for settings, recent searches, and query metadata.
- `ServiceWorker`: owns offline asset/API interception and delegates cache policy to ARCH-011.
- Authenticated browser-session creation is delegated to DESIGN-018. `SearchView` and `SidebarComponent` consume its frontend-safe session projection only for display, anonymous fallbacks, and protected-action routing.

### 1. Data Structures & Types
- `type SearchMode = "catalog" | "substitution" | "daily_diet_alternative"`
- `interface SearchState { query: string; mode: SearchMode; substitutionInputs: SubstitutionInputViewModel[]; filters: SearchFilter[]; page: number; selectedIndex: number; isOnline: boolean; isLoading: boolean; authStatus: "unknown" | "anonymous" | "authenticated" | "expired"; error?: AppError }`
- `interface SubstitutionInputViewModel { foodObjectId: string; quantity: number; unit: string; label: string }`
- `interface SearchFilter { id: string; kind: "food_category" | "culinary_role" | "food_object_type" | "allergen" | "dietary_preset"; mode: "include" | "exclude"; label: string }`
- `interface FoodItemViewModel { id: string; name: string; imageUrl?: string; macros: MacroSummary; classifications: string[]; similarity?: SimilarityBadge }`
- `interface MacroSummary { protein: number; carbs: number; fat: number; unitBasis: "100g" | "100ml" | "serving" }`
- `interface SimilarityBadge { score: number; tier: "excellent" | "good" | "fair" | "poor"; colorHex: string; imageUrl: string }`
- `interface AppSettings { theme: "system" | "light" | "dark"; unitSystem: "metric" | "imperial" }`
- `interface CachedQuery { key: string; request: SearchRequest; response: SearchResponse; storedAt: string; staleAt: string }`

### 2. Logic & Algorithms (Step-by-Step)
1. On app startup, load `AppSettings` from `LocalStorageManager`; default to `mode = "catalog"` with metric units unless a saved preference exists.
2. Register the service worker and subscribe to `online` and `offline` browser events.
3. Initialize Svelte stores for search state, settings, offline status, DESIGN-018 auth session projection, and current user entitlement.
4. When the search input changes, trim the value, update state immediately, and start a 150ms debounce timer.
5. After debounce, build a `SearchRequest`; if offline, read cached results through ARCH-011 and mark the result set as stale.
6. If online, execute the request through TanStack Query against ARCH-010; use query keys derived from mode, query, filters, page, and Substitution Input IDs and quantities.
7. Render `AutocompleteDropdown` for active text input; keyboard navigation changes `selectedIndex` and Enter selects the highlighted option.
8. Render `ResultsGrid` with stable card dimensions, image fallback handling, similarity badges, pagination controls, and empty-state text.
9. Persist theme and unit preference changes to localStorage, then update CSS variables through ARCH-016.
10. Route authenticated-only actions, including saved-data and checkout entry points, through DESIGN-018 `AuthenticatedActionGuard` before calling protected APIs.
11. Surface network, timeout, entitlement, auth, and validation failures through ARCH-017 instead of local ad hoc messages.

### 3. State Management & Error Handling
- `idle`: no active request; search input can be edited.
- `debouncing`: local input has changed but no request was sent.
- `loading`: TanStack Query request is in flight; previous results stay visible unless this is the first page.
- `success`: results and total count are present; cache metadata is updated.
- `empty`: request succeeded with zero items; show an empty result view.
- `offline`: browser is offline; use cached response if available and show `OfflineBanner`.
- `stale`: cached response exists but `staleAt` has passed; display data with staleness indicator.
- `api_error`: ARCH-010 returns 4xx or 5xx; map through `ErrorMessageMapper`.
- `anonymous`: DESIGN-018 reports no authenticated session; keep Catalog Search usable and show sign-in guidance for protected sidebar, saved-data, and checkout actions.
- `session_expired`: DESIGN-018 reports expired cookies; clear authenticated-only UI state and request sign-in before protected actions.
- `timeout`: request exceeds 10 seconds; show retry action and keep previous state.
- `storage_unavailable`: localStorage or Cache API fails; continue online-only and log a client warning.

### 4. Component Interfaces
- `function buildSearchRequest(state: SearchState): SearchRequest`
- `function debounceSearchInput(value: string, delayMs: 150): void`
- `function selectAutocompleteOption(option: FoodItemViewModel): void`
- `function setSearchMode(mode: SearchMode): void`
- `function updateFilters(filters: SearchFilter[]): void`
- `function applyAuthStatus(status: "unknown" | "anonymous" | "authenticated" | "expired"): void`
- `function loadSettings(): AppSettings`
- `function saveSettings(settings: AppSettings): void`
- `async function fetchSearchResults(request: SearchRequest): Promise<SearchResponse>`
- `async function readOfflineResults(request: SearchRequest): Promise<CachedQuery | null>`
- `function registerServiceWorker(): Promise<ServiceWorkerRegistration | null>`
