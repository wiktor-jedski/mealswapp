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
- `type FoodObjectType = "food_item" | "meal"`
- `interface DailyDietEntry { id: UUID; foodObjectId: UUID; foodObjectType: FoodObjectType; quantity: number; unit: CanonicalQuantityUnit; position: number }`
- `interface DailyDiet { id: UUID; userId: UUID; name: string; entries: DailyDietEntry[]; createdAt: time.Time; updatedAt: time.Time }`
- `interface SearchHistoryEntry { id: UUID; userId: UUID; query: string; mode: string; filtersHash: string; createdAt: time.Time }`
- `interface ExportUser { userId: UUID; email: string; role: "user" | "admin"; displayName: string; unitSystem: "metric" | "imperial"; themePreference: "system" | "light" | "dark" }`
- `interface ExportSavedItem { id: UUID; itemId: UUID; kind: "favorite" | "saved_meal" | "saved_diet"; createdAt: time.Time }`
- `interface ExportSavedDiet { id: UUID; name: string; entries: ExportSavedDietEntry[]; createdAt: time.Time; updatedAt: time.Time }`
- `interface ExportSearchHistoryEntry { id: UUID; query: string; mode: string; filtersHash: string; createdAt: time.Time }`
- `interface ExportCustomItem { id: UUID; name: string; physicalState: "solid" | "liquid"; prepTimeMinutes: number; metric serving and density fields; macrosPer100: ExportMacros; micros: map<string, number>; foodCategories: ExportClassificationSummary[]; culinaryRoles: ExportClassificationSummary[]; imageUrl?: string }`
- `interface ExportBundle { user: ExportUser; consent: ExportConsent[]; savedItems: ExportSavedItem[]; savedDiets: ExportSavedDiet[]; history: ExportSearchHistoryEntry[]; customItems: ExportCustomItem[] }`
- `interface DeletionPlan { userId: UUID; piiTables: string[]; cascadeTables: string[]; cachePrefixes: string[]; requestedAt: time.Time }`

### 2. Logic & Algorithms (Step-by-Step)
1. Require authenticated user context from ARCH-006 for every profile route.
2. Read and write preferences through ARCH-005 using `user_id` predicates on every query.
3. The authenticated profile is authoritative for unit preference. When it changes, persist the value and return the confirmed profile plus recalculation hints; the frontend applies that confirmed value and performs display conversion exactly once. Anonymous device preference remains browser-local and is not copied into an account profile.
4. Save favorites, meals, diets, and optional history with the authenticated user ID supplied by the server, never by the client. Daily Diet names are unique per user after trimming and case folding. A Daily Diet entry identifies exactly one Food Item or Meal, and aggregate nutrition is derived from that authoritative Food Object.
5. Data export passes the authenticated user ID to every repository query, decrypts the owner's portable data, and copies repository results into export-only projections before they reach a serialization function. Repository entities and persistence ownership fields never form part of `ExportBundle`.
6. JSON export writes one strict `ExportBundle`. CSV writes one three-column document (`section,field,value`): top-level account identity uses individual `user` rows, while each nested resource row carries the export projection as escaped JSON. Empty collection sections carry an explicit `count,0` row. Repository ordering and JSON map-key ordering make unchanged owner data deterministic.
7. The authenticated account identity appears exactly once in the top-level `user` object or CSV `user` section. Nested projections recursively exclude `userId`, `UserID`, `ownerId`, `owner_id`, and equivalent persistence-only ownership spellings while retaining IDs, content, classifications, nutrition, and timestamps.
8. Account deletion builds a deletion plan, deletes production records in a transaction, and calls ARCH-011 to purge user cache keys.
9. Return deletion confirmation only after database deletion and cache purge are complete or explicitly queued for retry.

### 3. State Management & Error Handling
- `profile_missing`: create default profile after first authentication.
- `preference_saved`: return updated profile and recalculation hints.
- `export_pending`: export generation is running for larger accounts.
- `export_ready`: file payload or signed download URL is available.
- `delete_requested`: deletion workflow has started and account is locked from new writes.
- `delete_completed`: production data and cache entries are removed.
- `cross_user_access`: return 403 and audit log the attempt.
- `duplicate_daily_diet_name`: return 409 without creating or replacing a collection.
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
