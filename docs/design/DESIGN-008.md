## FILE: DESIGN-008.md
**Traceability:** ARCH-008

**Static aspects covered:** ProfileController, PreferenceManager, SavedDataRepository, SearchHistoryRepository, DataExporter, AccountDeleter.

### 0. Static Aspect Responsibilities
- `ProfileController`: owns profile, preferences, export, deletion, saved data, and history endpoints.
- `PreferenceManager`: owns unit, theme, and recalculation-hint behavior.
- `SavedDataRepository`: owns user-scoped favorites, saved meals, and saved diets.
- `SearchHistoryRepository`: owns optional user-scoped recent-search persistence with a bounded retention policy.
- `DataExporter`: owns JSON/CSV export bundle generation.
- `AccountDeleter`: owns production data deletion, account write lockout, and cache purge coordination.

### 1. Data Structures & Types
- `interface UserProfile { userId: UUID; displayName?: string; unitSystem: "metric" | "imperial"; themePreference: "system" | "light" | "dark"; createdAt: time.Time; updatedAt: time.Time }`
- `interface SavedItem { id: UUID; userId: UUID; itemId: UUID; kind: "favorite" | "saved_meal" | "saved_diet"; createdAt: time.Time }`
- `interface SearchHistoryEntry { id: UUID; userId: UUID; query: string; mode: string; filtersHash: string; createdAt: time.Time }`
- `interface ExportBundle { user: UserProfile; savedItems: SavedItem[]; history: SearchHistoryEntry[]; customItems: FoodItemEntity[]; format: "json" | "csv" }`
- `interface DeletionPlan { userId: UUID; piiTables: string[]; cascadeTables: string[]; cachePrefixes: string[]; requestedAt: time.Time }`

### 2. Logic & Algorithms (Step-by-Step)
1. Require authenticated user context from ARCH-006 for every profile route.
2. Read and write preferences through ARCH-005 using `user_id` predicates on every query.
3. When unit preference changes, persist the value and return recalculation hints for currently displayed data.
4. Save favorites, meals, diets, and optional history with the authenticated user ID supplied by the server, never by the client.
5. Data export loads profile, PII, saved data, custom items, diets, and history into an `ExportBundle`.
6. JSON export writes a structured object; CSV export writes separate sections/files for tabular data.
7. Account deletion builds a deletion plan, deletes production records in a transaction, and calls ARCH-011 to purge user cache keys.
8. Return deletion confirmation only after database deletion and cache purge are complete or explicitly queued for retry.

### 3. State Management & Error Handling
- `profile_missing`: create default profile after first authentication.
- `preference_saved`: return updated profile and recalculation hints.
- `export_pending`: export generation is running for larger accounts.
- `export_ready`: file payload or signed download URL is available.
- `delete_requested`: deletion workflow has started and account is locked from new writes.
- `delete_completed`: production data and cache entries are removed.
- `cross_user_access`: return 403 and audit log the attempt.
- `export_failed`: retain account state and provide retryable error.
- `cache_purge_failed`: queue retry and include operational warning for monitoring.

### 4. Component Interfaces
- `func (c *ProfileController) GetProfile(ctx *fiber.Ctx) error`
- `func (c *ProfileController) UpdatePreferences(ctx *fiber.Ctx) error`
- `func (c *ProfileController) ExportData(ctx *fiber.Ctx) error`
- `func (c *ProfileController) DeleteAccount(ctx *fiber.Ctx) error`
- `func SaveHistory(ctx context.Context, userID UUID, entry SearchHistoryEntry) error`
- `func ListSavedData(ctx context.Context, userID UUID) ([]SavedItem, error)`
- `func BuildExportBundle(ctx context.Context, userID UUID, format string) (ExportBundle, error)`
- `func ExecuteAccountDeletion(ctx context.Context, plan DeletionPlan) error`
