# Task 250 Preparation Evidence

## Result

Task 250 (`DESIGN-009: ItemCurator`) is implemented and verified without editing its task-list status. The row remains `OPEN`. Manual administrator CRUD now targets only global `food_items`; private `custom_food_items` remain owner-scoped and structurally separate. Create uses an administrator-scoped, transactionally durable idempotency claim, while create/update/delete commit with privacy-safe audit snapshots or roll back together.

The phase-orchestrator skill required a writable preparation subagent, but no subagent capability was available in this session. The preparation contract was therefore applied directly. No independent acceptance review or task-status transition was attempted.

## Task and dependency context

- Fixed Git reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Task row at start and finish: `250 | Phase 08 Manual Global Item CRUD | DESIGN-009: ItemCurator | OPEN`.
- Required dependencies 33, 34, 43, 238, and 247 were all `PASSED` at selection time.
- Preparation report: `docs/implementation/preparations/task-250.md`.
- Task-list SHA-256 at start and finish: `689954f8dc9a17c2344db0e03be72e1555aabc9d8756c0d8213763e9ba7c3a96`.
- `docs/implementation/04_OPEN.md` start SHA-256: `9a91d4cfe32ed9eb1f85d08dd668cb44cac3635f5a9d4dff368fdd425c22b5fa`; final SHA-256: `4b703eca5a8b6207ce0e87fc0ea23c9df255ac8923ab8d93e8b4f8ff1f318e4d` after adding the Task 250 coverage deviation.

## Baseline and preservation

The initial worktree was heavily dirty with concurrent Phase 08 work. The complete initial `git status --short` was captured before inspection. Pre-existing changes included API generation, custom-item persistence/API/export/erasure, external providers/search, admin gateway, filter options, classification administration, user administration, shared repository files, migrations 25-28, and preparation/review evidence for tasks 238-247. Those paths and unrelated hunks were preserved; no reset, checkout, clean, commit, migration rewrite, OpenAPI edit, generated-client edit, or task-list status edit was performed.

Initial task-relevant control hashes were:

| Path | Initial SHA-256 |
|---|---|
| `backend/internal/app/app.go` | `42dadf16664b29dcfff32cf4b6b1643a2b9e5bdaa0a776b9d63eabd8498b0243` |
| `backend/internal/repository/types.go` | `5534be37a865c95390f84687ed82007e0adbca63a94fbd8c7e849ccb8cc40ac6` |
| `backend/internal/repository/compliance_repository.go` | `d185aed065dd59ade5d3f7330efa5defc1e4acabd5958f2a8ed1e9c83f111f88` |
| `backend/internal/repository/food_repository.go` | `e37c943ee99bb260c5710a8738f21ea1c1709690ba40d610f91336816a24ade0` |
| `backend/internal/httpapi/admin_controller.go` | `94d3841f21c30bd2939e2be1ac46d8d984c68a95765e488854335bff6ed6fe3c` |
| `backend/internal/httpapi/curation_validation.go` | `14cd4a46838d84fb643fd9448e8f61c6909f09bf66203ae00d7570d7664c46f1` |
| `backend/internal/customitem/service.go` | `4bc9eb6ae297aec1b1030a23084143d27edf9f807e02f07073d4bce541b3975b` |
| `backend/internal/httpapi/router.go` | `a98d15348d69f6fdf4d5076c2c12f203b20b10c033a81a6fcc25cd7dfacbcb48` |
| `api/openapi.yaml` | `af5a676d54220079d5f852139a57e0737fee7ffa4e3ca595a6ed302417d4d0c7` |

Concurrent tasks modified shared `app.go`, `types.go`, and `compliance_repository.go` after baseline capture. Task-owned hunks remain identifiable by their adjacent `DESIGN-009 ItemCurator` comments and the symbol inventory below. Baseline confidence is high for new Task 250 files and medium for those shared files; no unrelated shared-file behavior is claimed as Task 250 work.

## Design and security decisions

- `docs/design/DESIGN-009.md` defines ItemCurator validation/duplicate handling, global food-item mutation, and before/after audit coordination.
- Dependency 33 supplies global `food_items` CRUD/search, macro/micronutrient/classification hydration, and soft deletion.
- Dependency 43 supplies the required positive liquid density plus provenance-kind invariant.
- Dependency 238 supplies physically separate, mandatory-owner `custom_food_items`; Task 250 never queries that table.
- Dependency 247 supplies verified admin-cookie authorization, CSRF, rate limits, validation ordering, server request IDs, and the fail-closed transaction/audit callback.
- The Go security skill guided the implementation: all new SQL is embedded and parameterized; untrusted JSON is strictly decoded with duplicate-key, unknown-field, ownership-field, URL, numeric, classification, and liquid-density rejection; authorization remains server-derived; errors are mapped to generic envelopes; replay cannot carry mutation-derived audit changes; and audit snapshots expose only booleans plus the closed `solid|liquid` enum.

No new migration was needed. The existing shared `mutation_idempotency_keys` table already provides the required administrator/method/route/key uniqueness and JSON response storage.

## Changed paths and symbols

### Production and persistence

- `backend/internal/repository/types.go`
  - Modified `AdminAuditChanges` with explicit `Replayed` semantics.
  - Added `ManualFoodItemCreateClaim`, `ManualFoodItemCreateClaimResult`, and `ManualFoodItemResponseEncoder`.
- `backend/internal/repository/compliance_repository.go`
  - Modified `(*PostgresAdminImportAuditRepository).WithMutationAudit` to commit exact no-op replay without a duplicate audit, while rejecting replay markers mixed with entity/snapshot changes.
  - Extended `adminAuditSnapshotSchemas` and canonical ordering for `manual_create`, `manual_update`, and `manual_delete` global-food snapshots.
- `backend/internal/repository/food_repository.go`
  - Added transaction-executor helpers `validateFoodItemWithExecutor`, `validateFoodClassificationsWithExecutor`, `validateMicronutrientsWithExecutor`, `replaceFoodClassificationsWithExecutor`, `getFoodByIDWithExecutor`, and `hydrateFoodClassificationsWithExecutor`.
  - Modified the corresponding existing repository methods to delegate to those helpers, preserving prior behavior while allowing the admin gateway to own the transaction.
- `backend/internal/repository/manual_food_repository.go`
  - Added embedded SQL variables `manualFoodCreateClaimSQL`, `manualFoodCreateClaimGetSQL`, and `manualFoodCreateClaimCompleteSQL`.
  - Added `PostgresManualFoodItemRepository`, `NewPostgresManualFoodItemRepository`, `GetByID`, `GetByIDInMutation`, `ClaimCreate`, `Update`, and `Delete`.
  - Added helpers/type `createManualFoodItem`, `getManualFoodByID`, `manualFoodCreateRecord`, `scanManualFoodCreateClaim`, and `validateManualFoodCreateClaim`.
- `backend/internal/repository/sql/manual_food_create_claim.sql`
  - Added the parameterized first-writer claim for fixed scope `POST /admin/items`.
- `backend/internal/repository/sql/manual_food_create_claim_get.sql`
  - Added the parameterized `FOR UPDATE` replay/conflict lookup.
- `backend/internal/repository/sql/manual_food_create_claim_complete.sql`
  - Added the body-matched immutable `201` response completion.

### ItemCurator service and HTTP

- `backend/internal/itemcurator/service.go`
  - Added errors `ErrMissingIdempotencyKey` and `ErrIdempotencyConflict`.
  - Added types `Request`, `ClassificationSummary`, `Item`, `CreateResult`, `MutationResult`, `Store`, and `Service`.
  - Added `NewService`, `Create`, `Get`, `Update`, `Delete`, `requestHash`, `toEntity`, `fromEntity`, and `validationError`.
- `backend/internal/httpapi/manual_item_controller.go`
  - Added `ManualItemService`, `ManualItemController`, and `NewManualItemAdminController`.
  - Added CRUD handlers `Create`, `Get`, `Update`, and `Delete`.
  - Added validators/helpers `validateManualItemCreate`, `validateManualItemUpdate`, `validateManualItemBody`, `manualItemRequest`, `validateManualItemID`, `parseManualItemID`, `manualItemData`, `manualItemAuditSnapshot`, `manualItemDependencyError`, and `manualItemError`.
  - Registered `POST /api/v1/admin/items`, `GET /api/v1/admin/items/:itemId`, `PUT /api/v1/admin/items/:itemId`, and `DELETE /api/v1/admin/items/:itemId` through the dependency-247 gateway.
- `backend/internal/app/app.go`
  - Modified `NewProduction` only for the ItemCurator import, PostgreSQL service construction, and manual admin controller registration. Concurrent classification/external-search/user-admin edits in this file are unrelated.

### Tests and planning evidence

- `backend/internal/itemcurator/service_test.go`
  - Added `memoryStore`, `memoryClaim`, `inertExecutor`, their methods, `solidRequest`, `TestServiceCreateReplayConflictDuplicateAndCRUD`, and `TestServiceRejectsInvalidFieldsAndLiquidDensity`.
- `backend/internal/httpapi/manual_item_controller_test.go`
  - Added `fakeManualItemService`, its CRUD methods, `TestManualItemAdminHTTPValidCRUDReplayAndAuditSnapshots`, `TestManualItemAdminHTTPRejectsConflictsDuplicatesInvalidFieldsAndOwnership`, and `manualItemHTTPRequest`.
- `backend/internal/httpapi/admin_controller_test.go`
  - Modified `adminAuditCoordinator.WithMutationAudit` to model production replay semantics.
- `backend/internal/repository/manual_food_repository_test.go`
  - Added `TestPostgresManualFoodItemCRUD` and `assertManualFoodSearch`.
- `backend/internal/repository/admin_audit_security_test.go`
  - Added `TestAdminMutationAuditReplayCommitsWithoutDuplicateAudit`.
- `docs/implementation/04_OPEN.md`
  - Added the precise Task 250 focused-coverage deviation; no behavior criterion is waived.
- `docs/implementation/preparations/task-250.md`
  - Added this preparation record. Its self-referential hash is intentionally omitted.

## Verification-criteria evidence

| Criterion | Evidence |
|---|---|
| Valid create/read/update/soft-delete | Service and HTTP tests cover all four operations; PostgreSQL integration loads hydrated fields and performs audited update/delete. |
| Stable create replay | Normalized request SHA-256 plus fixed admin/method/route/key scope is inserted, locked, completed, and replayed in the same audit transaction. Tests assert the same response bytes/ID and no second audit. |
| Conflicting key reuse | Changed normalized hash returns `ErrorKindIdempotencyConflict`, mapped to `409 idempotency_key_conflict`, without an item or audit side effect. |
| Duplicate names | A different key with an active normalized duplicate name returns generic resource conflict and rolls back its provisional claim. HTTP maps it to safe `409 conflict`. |
| Invalid macros/micros/images/classifications | Service/HTTP tests reject malformed macros, micronutrients, image schemes, duplicate JSON/classification IDs, and ownership fields. PostgreSQL tests reject missing active micronutrient keys and nonexistent classifications. |
| Liquid density | Tests reject liquid items without density or with unsupported provenance and accept a positive manual density record. |
| Before/after audit snapshots | Create stores a bounded after snapshot; update stores before/after; delete stores active-before/deleted-after. Persisted audit entries are asserted. |
| Search visibility | Real PostgreSQL catalog search finds create, stops finding the old name after update, finds the new name, and excludes it after soft delete. |
| Audit rollback | A forbidden audit snapshot produces `ErrAdminAuditPersistence`; the transaction rolls back the food row and idempotency claim, and search proves no item remains. |
| Global/private isolation | Global writes use only `food_items`; information-schema inspection proves it has no `owner_id`; custom and global repositories cannot read each other's IDs; the same name may exist in both tables; the manual controller does not register a private-item path and rejects `ownerId`. |

All required Task 250 criteria are satisfied.

## Commands and results

Commands ran on 2026-07-21 from the repository root unless a backend working directory is stated.

| Command | Result |
|---|---|
| `go test ./internal/itemcurator ./internal/httpapi -count=1` (`backend/`) | PASS. |
| `go test ./internal/repository -run 'TestPostgresManualFoodItemCRUD|TestAdminMutationAuditReplay|TestAdminAudit' -count=1` (`backend/`) | PASS. |
| `go test -race ./internal/itemcurator ./internal/httpapi ./internal/repository -count=1` (`backend/`) | ItemCurator and HTTP packages passed with no race report; the task-specific repository race test was rerun separately for an explicit terminal result. |
| `go test -race ./internal/repository -run TestPostgresManualFoodItemCRUD -count=1` (`backend/`) | PASS; no race report. |
| `go test ./internal/app -run '^$' -count=1` (`backend/`) | PASS; production assembly compiles. |
| `go vet ./...` (`backend/`) | PASS. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` (`backend/`) | PASS: zero reachable vulnerabilities; 18 required-module advisories are not called. |
| `go test ./internal/itemcurator ./internal/httpapi ./internal/repository -coverprofile=/tmp/task-250.cover -count=1` (`backend/`) | PASS; package coverage `74.7%`, `87.4%`, and `91.9%`; `89.3%` combined. Required behavior branches are directly covered; deviation recorded in `04_OPEN.md`. |
| `go test ./... -count=1` (`backend/`) | Task 250 packages PASS. Aggregate FAILS only in unrelated concurrent work: task-240 `TestTask240CustomItemErasureIntegration` leaves two custom items, and task-252 `TestPostgresAdminUserLookupIsExactBoundedAndPrivacyMinimized` returns the wrong concurrent fixture. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks; task 250 remains `OPEN`. |
| `python3 scripts/validate-traceability.py` | PASS. An earlier run exposed missing comments in concurrently edited `user_admin_controller.go`; its owner repaired them before the final rerun. |
| `git diff --check` | PASS. |

## Final hashes

| Path | Final SHA-256 |
|---|---|
| `backend/internal/app/app.go` | `e9fa64094fdbff1b2b8e88857dd119b400d337f9401464ee284cffb2e17c5409` |
| `backend/internal/repository/types.go` | `7a0069590989b8fe0311c960021c5da87cc3d568b9ca708a7e586b620511f730` |
| `backend/internal/repository/compliance_repository.go` | `57790181b840c7f494e44dbd62a7c69be6210cff79987652db6bd6a3852712f6` |
| `backend/internal/repository/food_repository.go` | `01e10872bc32ec184ed42a17941294f51bd235110983615dd4a41c81b244453f` |
| `backend/internal/repository/manual_food_repository.go` | `7cce8d565c88161e3639253cd397a736dccf9915d612255a4c237089ca91b6fa` |
| `backend/internal/repository/manual_food_repository_test.go` | `bad81f579e53756bcf168b00cfd631c776eb24cc5a259296e749f0fb51067c60` |
| `backend/internal/repository/sql/manual_food_create_claim.sql` | `cb268b43f6301fe68598e14f811ecd0f38f3b92ff9ed1cbe945631e7663cb4fc` |
| `backend/internal/repository/sql/manual_food_create_claim_get.sql` | `085972e305710dc289ecf32dffcbcad0de0b9845b6cb65c9375a02efa33d5b90` |
| `backend/internal/repository/sql/manual_food_create_claim_complete.sql` | `d3385f705c9ba162aad090a6ff1c448ab9d445834cd3e391d3d02b171dd4a80d` |
| `backend/internal/itemcurator/service.go` | `7bd8e1a99c795d318e8dc7e4988b571f4aa6ddefe35730d22057bf02324a3729` |
| `backend/internal/itemcurator/service_test.go` | `5e3932a2ecfff6d3fe6d1f4c822f9fcdbc6620e755a07bf1103dfdd5fdee7bfe` |
| `backend/internal/httpapi/manual_item_controller.go` | `b7ec1af1f64a48461922915ff5d2aa012ee870f42e0983fb407d00e6f1496b4c` |
| `backend/internal/httpapi/manual_item_controller_test.go` | `b62b8a8a714be320bf3fffeff93325125e5f19a3880dca8fa8444dbddc19046e` |
| `backend/internal/httpapi/admin_controller_test.go` | `a946248885c974cb44d4abba90157f8a66bbad1bcd6fa4b1a90e582e17e8ac13` |
| `backend/internal/repository/admin_audit_security_test.go` | `6398fd2c0a680a5c985dde54f31994cd23766304dc9d1812807785d5466adbda` |
| `docs/implementation/02_TASK_LIST.md` (unchanged control) | `689954f8dc9a17c2344db0e03be72e1555aabc9d8756c0d8213763e9ba7c3a96` |
| `docs/implementation/04_OPEN.md` | `4b703eca5a8b6207ce0e87fc0ea23c9df255ac8923ab8d93e8b4f8ff1f318e4d` |

## Handoff

- Task 250 is prepared but intentionally remains `OPEN`; no task-list status was edited.
- Required behavior, security boundaries, and task-specific tests pass.
- Independent review remains outstanding because the required subagent capability was unavailable.
- Aggregate failures listed above belong to concurrent tasks 240/252 and were not modified.
