## FILE: DESIGN-009.md
**Traceability:** ARCH-009

**Static aspects covered:** AdminController, DataImporter, ItemCurator, TagManager, UserAdminPanel, ExternalSearchProxy.

### 0. Static Aspect Responsibilities
- `AdminController`: owns admin-only endpoint routing, role checks, and audit coordination.
- `DataImporter`: owns validated persistence of curated external candidates into ARCH-005.
- `ItemCurator`: owns draft editing rules, required fields, and duplicate handling.
- `TagManager`: owns global category/functionality tag CRUD and in-use safeguards.
- `UserAdminPanel`: owns restricted user lookup and administrative user actions.
- `ExternalSearchProxy`: owns calls from admin UI to ARCH-012 and result shaping for curation.

### 1. Data Structures & Types
- `interface AdminContext { userId: UUID; role: "admin"; requestId: string }`
- `interface ExternalSearchRequest { query: string; provider: "usda" | "openfoodfacts" | "all"; page: number }`
- `interface ExternalCandidate { provider: string; externalId: string; name: string; macrosPer100: MacroValues; imageUrl?: string; raw: map[string]any }`
- `interface CuratedItemDraft { sourceProvider?: string; externalId?: string; name: string; physicalState: PhysicalState; macrosPer100: MacroValues; categoryTagIds: UUID[]; functionalityTagIds: UUID[]; imageUrl?: string }`
- `interface AdminAuditEntry { adminUserId: UUID; action: string; entityType: string; entityId?: UUID; before?: any; after?: any; createdAt: time.Time }`

### 2. Logic & Algorithms (Step-by-Step)
1. API gateway authenticates the request; `AdminController` verifies role `admin`.
2. External search requests are sent to `ExternalSearchProxy`, which calls ARCH-012 instead of the local repository.
3. Normalize provider results enough for admin display but do not persist them until curation is confirmed.
4. Admin edits required fields, tags, macro values, physical state, and image URL.
5. `DataImporter` validates the curated draft against repository rules and saves it through ARCH-005.
6. Item CRUD operations load current state, apply the mutation, and write an `AdminAuditEntry`.
7. `TagManager` creates and updates global category and functionality tags, preventing duplicate names within each kind.
8. User admin actions are role-restricted and audited with before/after snapshots where appropriate.

### 3. State Management & Error Handling
- `forbidden`: non-admin user receives 403.
- `external_search_loading`: external query in progress.
- `external_source_unavailable`: return empty candidate list with warning.
- `draft_invalid`: required curated fields are missing or inconsistent.
- `import_conflict`: external item or normalized name already exists; require admin confirmation to merge.
- `audit_write_failed`: abort mutating operation unless audit can be persisted in the same transaction.
- `tag_in_use`: block destructive tag deletion or require replacement tag.

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
