## FILE: DESIGN-009.md
**Traceability:** ARCH-009

**Static aspects covered:** AdminController, DataImporter, ItemCurator, TagManager, UserAdminPanel, ExternalSearchProxy.

### 0. Static Aspect Responsibilities
- `AdminController`: owns admin-only endpoint routing, role checks, and audit coordination.
- Administrator bootstrap is not an `AdminController` HTTP route. It is a dedicated operator-only CLI that can change only the first administrator role.
- `DataImporter`: owns validated persistence of curated external candidates into ARCH-005.
- `ItemCurator`: owns draft editing rules, required fields, and duplicate handling.
- `TagManager`: owns global food_category/culinary_role classification CRUD and in-use safeguards.
- `UserAdminPanel`: owns restricted user lookup and administrative user actions.
- `ExternalSearchProxy`: owns calls from admin UI to ARCH-012 and result shaping for curation.

### 1. Data Structures & Types
- `interface AdminContext { userId: UUID; role: "admin"; requestId: string }`
- `interface ExternalSearchRequest { query: string; provider: "usda" | "openfoodfacts" | "all"; page: number }`
- `interface ExternalCandidate { provider: string; externalId: string; recordToken: string; name: string; macrosPer100: MacroValues; imageUrl?: string }`
- `interface CuratedItemDraft { externalRecordToken?: string; name: string; physicalState: PhysicalState; densitySourceKind?: "imported" | "manual" | "estimated"; macrosPer100: MacroValues; foodCategoryIds: UUID[]; culinaryRoleIds: UUID[]; imageUrl?: string }`
- `interface AdminAuditEntry { actorKind: "administrator" | "operator"; adminUserId?: UUID; action: string; entityType: string; entityId?: UUID; before?: any; after?: any; createdAt: time.Time }`
- `interface GlobalCatalogDocumentV1 { schema: "mealswapp.global-catalog.v1"; items: GlobalCatalogEntryV1[] }`
- `interface GlobalCatalogEntryV1 { idempotencyKey: string; item: metric AdminItemRequest plus portable classification names or explicit classification UUIDs and optional informational export metadata }`

### 2. Logic & Algorithms (Step-by-Step)
1. API gateway authenticates the request; `AdminController` verifies role `admin`.
2. External search requests are sent to `ExternalSearchProxy`, which calls ARCH-012 instead of the local repository.
3. Normalize provider results enough for admin display and issue an opaque, short-lived server-record token; do not persist food or audit state until curation is confirmed.
4. Admin edits required fields, classifications, macro values, physical state, and image URL.
5. `DataImporter` resolves the selected server record, derives canonical import and imported-density identity from it, validates the curated draft against repository rules, and saves it through ARCH-005. Private and manual-administrator item requests accept only manual or estimated density and cannot submit provider identity.
6. Item CRUD operations load current state, apply the mutation, and write an `AdminAuditEntry`.
7. `TagManager` creates and updates global Food Category and Culinary Role classifications, preventing duplicate names within each kind.
8. User admin actions are role-restricted and audited with before/after snapshots where appropriate. Exact email lookup uses the same canonical-plus-exact-legacy resolver as authentication and bootstrap, reindexing only an unambiguous legacy account and refusing collisions.
9. Administrator bootstrap requires an operator with database and lookup-key access to name the configured environment explicitly. Production additionally requires a separate confirmation flag.
10. Bootstrap resolves exactly one existing account through DESIGN-013's canonical lower-case email HMAC lookup digest. It may resolve the exact trimmed pre-canonical digest only to atomically reindex that same account; canonical/legacy different-account collisions fail closed. UUID is an advanced alternative; selectors cannot be combined.
11. The repository takes a transaction-scoped PostgreSQL advisory lock, then requires a verified account with a password credential accepted by the exact DESIGN-006 authentication parser or a populated encrypted OAuth credential. SQL does not parse password hashes. It promotes only when no administrator exists, commits the role and audit together, treats replay for the same target as a no-op, and refuses a different target.
12. Bootstrap audit uses `actor_kind = "operator"` with no `admin_user_id`; the promoted account is the `entity_id`. The shared audit model, list query, and scanner preserve both actor kind and nullable administrator identity so readback remains truthful.
13. Existing access and refresh sessions retain their signed role and verification claims. The promoted user must sign out and sign in again before administrator access is available.
14. The global-catalog operator validates the complete versioned JSON document, canonical micronutrients, metric solid/liquid rules, stable item keys, and the 500-item ceiling before login. It then logs in with an interactively entered password, keeps cookies in memory, obtains a fresh CSRF token, and resolves all active classification names/UUIDs and allergen keys before the first mutation.
15. The operator sends exactly one existing `POST /api/v1/admin/items` mutation per validated item, paced by at least 2.1 seconds. Ambiguous transport and 5xx outcomes retry with the identical key and normalized body; bounded `Retry-After` is honored. Deterministic input ordering plus immutable per-item keys makes interrupted reruns resumable.
16. ItemCurator validates active allergen keys and stores `food_item_allergens` together with the ownerless food item, classifications, idempotency response, and administrator audit in one transaction. Successful commit advances the existing food-data cache generation exactly once.
17. `GET /api/v1/admin/catalog-export` requires verified administrator claims and accepts only optional boolean `includeDeleted`, defaulting false. It never shares Account Export code or queries private custom items, users, saved data, consent, billing, sessions, or encrypted identity.
18. The export repository streams UUID-ordered `food_items` from one PostgreSQL repeatable-read, read-only transaction. Classification relationships are ordered by name and UUID; allergens, micronutrient keys, and curated-source identity are deterministic. The representation includes no generated timestamp, so unchanged state produces byte-identical compact JSON.
19. Export entries derive their stable idempotency key from the item UUID. Item timestamps, deletion state, image alt text, direct source identity, classification IDs/names, and `curated_imports` identity are informational metadata. The manual-item import path validates and reports that metadata but strips it before mutation and does not recreate `curated_imports`.
20. The export operator reuses the import operator's interactive credential and memory-only authenticated session helper. It streams the API response to a private same-directory temporary file, validates the complete JSON, optionally emits pretty JSON or deterministic inspection CSV, flushes and fsyncs, and atomically replaces the destination only after success.

### 3. State Management & Error Handling
- `forbidden`: non-admin user receives 403.
- `external_search_loading`: external query in progress.
- `external_source_unavailable`: return empty candidate list with warning.
- `draft_invalid`: required curated fields are missing or inconsistent.
- `external_record_evidence_invalid`: selected provider evidence is malformed, unknown, expired, mismatched, or no longer registered; no mutation or audit occurs.
- `import_conflict`: external item or normalized name already exists; require admin confirmation to merge.
- `audit_write_failed`: abort mutating operation unless audit can be persisted in the same transaction.
- `bootstrap_denied`: reject environment mismatch, missing production confirmation, ambiguous/missing target, canonical email collision, absent target, unverified target, unusable credential, or an existing different administrator without exposing PII.
- `classification_in_use`: block destructive classification deletion or require replacement classification.
- `catalog_preflight_failed`: reject the complete run before mutation when any schema, metric, micronutrient, classification, allergen, or key rule fails.
- `catalog_partial_failure`: continue after permanent per-item 400/409 failures, report only safe key fingerprints and bounded status metadata, and exit nonzero; 401/403 aborts immediately.
- `catalog_export_failed`: cancel the read-only snapshot and leave any prior operator destination untouched; emit only a safe error category.

### 4. Component Interfaces
- `func (c *AdminController) SearchExternal(ctx *fiber.Ctx) error`
- `func (c *AdminController) ImportItem(ctx *fiber.Ctx) error`
- `func (c *AdminController) UpdateItem(ctx *fiber.Ctx) error`
- `func (c *AdminController) DeleteItem(ctx *fiber.Ctx) error`
- `func (c *AdminController) ManageTags(ctx *fiber.Ctx) error`
- `func RequireAdmin(ctx *fiber.Ctx) (AdminContext, error)`
- `func NormalizeExternalCandidate(raw ExternalCandidate) CuratedItemDraft`
- `func PersistAuditEntry(ctx context.Context, entry AdminAuditEntry) error`
- `type CuratedImportRepository interface { UpsertCuratedImport(ctx context.Context, item CuratedImport) (UUID, error); FindCuratedImport(ctx context.Context, provider string, externalID string) (CuratedImport, error) }`
- `type AdminAuditRepository interface { PersistAuditEntry(ctx context.Context, entry AdminAuditEntry) (UUID, error); WithAudit(ctx context.Context, entry AdminAuditEntry, fn func(sqlExecutor) error) error; ListAuditForEntity(ctx context.Context, entityType string, entityID UUID) ([]AdminAuditEntry, error) }`
- `type AdministratorBootstrapRepository interface { BootstrapAdministrator(ctx context.Context, selector AdministratorBootstrapSelector, requestId string) (AdministratorBootstrapResult, error) }`
- `type GlobalCatalogExportRepository interface { Stream(ctx context.Context, includeDeleted bool, consume CatalogExportRowConsumer) (count int, err error) }`
- `func (s *catalogexport.Service) Write(ctx context.Context, output io.Writer, includeDeleted bool) (count int, err error)`
