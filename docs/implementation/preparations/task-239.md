# Task 239 Preparation Evidence

## Result

Task 239 is implemented, independently reviewed, repaired, and verified. The current task-list status remains `PREPARED`; this repair did not edit the task list. The implementation provides authenticated owner-scoped custom-item CRUD, transactionally atomic create idempotency, strict OpenAPI/domain validation, and owner-only JSON/CSV account export data.

## Task and baseline

- Original preparation task row: `239 | Phase 08 Custom Item API, Export, and Idempotency | DESIGN-008: ProfileController | OPEN | deps 97,109,112,238`; see the repair record for the unchanged current `PREPARED` row.
- Baseline commit captured before repository inspection: `81ca40ce00cb667ea29243ed2d34068e11229a69`
- Original-preparation baseline task-list hash: `d04a5f34dcfb732a48fdbf504d568b0c367d49ba4370a8f030a32aeb0a12fc35`.
- Original-preparation final task-list hash: `d04a5f34dcfb732a48fdbf504d568b0c367d49ba4370a8f030a32aeb0a12fc35`.
- Original-preparation row state was `OPEN`; the independent-review workflow later set it to `PREPARED` before this repair.
- Preparation evidence path: `docs/implementation/preparations/task-239.md`
- Baseline confidence: high. Tracked task-239 files were clean at the fixed commit. Task-238 repository work and documentation were already dirty/untracked; their initial hashes were captured before edits and are listed below.

### Initial dirty worktree

```text
 M backend/internal/repository/sql/classification_is_in_use.sql
 M backend/internal/repository/types.go
 M docs/design/DESIGN-005.md
 M docs/implementation/02_TASK_LIST.md
?? backend/internal/repository/custom_food_repository.go
?? backend/internal/repository/custom_food_repository_test.go
?? backend/internal/repository/sql/custom_food_attach_classification.sql
?? backend/internal/repository/sql/custom_food_clear_classifications.sql
?? backend/internal/repository/sql/custom_food_create.sql
?? backend/internal/repository/sql/custom_food_get_by_id.sql
?? backend/internal/repository/sql/custom_food_list_classifications.sql
?? backend/internal/repository/sql/custom_food_soft_delete.sql
?? backend/internal/repository/sql/custom_food_update.sql
?? backend/internal/repository/sql/testdata/custom_food_ownerless_create.sql
?? database/migrations/000025_user_owned_custom_food_items.down.sql
?? database/migrations/000025_user_owned_custom_food_items.up.sql
?? docs/implementation/preparations/
?? docs/implementation/reviews/task-238-review.md
```

Initial hashes of pre-existing files later extended by task 239:

- `backend/internal/repository/types.go`: `05e6b89d355078c1d69a7d01a7ee7278ef977bc7e57a5b13541f4dc29e9ce41f`
- `backend/internal/repository/custom_food_repository.go`: `5adb40a45499c0c5b21fe295a87c3d325c28ac7cfb3f1a2fa90346c98055b472`
- `backend/internal/repository/custom_food_repository_test.go`: `5d4b73d06ace7575d235469eeabf688d45f26ffa95bef929b7b5d66ba7a9e250`

All other initial dirty paths were preserved and were not edited for task 239.

## Design and requirements read

- `docs/design/DESIGN-008.md`: ProfileController authentication/server ownership, DataExporter JSON/CSV bundle behavior, and custom-item export inclusion.
- `docs/design/DESIGN-005.md`: dedicated custom-item persistence, mandatory owner predicates, ownership-safe not-found, macro/micronutrient/classification/liquid-density invariants, and global catalog isolation.
- `docs/design/DESIGN-010.md`: route validation, authentication, and CSRF middleware ordering.
- `docs/requirements/01_SOFT_REQ_SPEC.md`, especially the private custom-item visibility requirement at SW-REQ-043.
- `api/openapi.yaml`: shared envelope, structured errors, cookie authentication, CSRF, and cross-phase `Idempotency-Key` parameter contracts.
- Dependency implementations/evidence for tasks 97, 109, 112, and 238, including the Phase 03 export stub and Phase 08 custom-item repository.

## Changed paths and final hashes

| Path | Task-239 change | Final SHA-256 |
|---|---|---|
| `api/openapi.yaml` | Added custom-item CRUD operations, parameter/response/request/item schemas, and typed export items. | `d148a39b89ac47b676662bba9e7262cc6f690e4450770164e03ab08b931ea289` |
| `backend/internal/app/app.go` | Composed the custom-item repository/service into ProfileController and DataExporter. | `bf446fe31510b13ab47604dc444d43803b2bed1d645ae1cb082879d663f4746a` |
| `backend/internal/customitem/service.go` | Added owner-scoped CRUD, normalized idempotency, and owner-free projections. | `91aac45ce9f543a05968ae1ad457edfe2082ce2eae99d6d1237abb128a923259` |
| `backend/internal/customitem/service_test.go` | Added service replay/conflict/ownership tests. | `3e9a81260a166f1c87837cae302b5c720295265cc52232b352d8a6d32c2d93b2` |
| `backend/internal/httpapi/custom_item_controller.go` | Added authenticated CRUD handlers, strict validation, CSRF route policy, and safe error mapping. | `c28df155dc9dea9f73a794b3100e4fce4724a989a20e22032042572867de326a` |
| `backend/internal/httpapi/custom_item_controller_test.go` | Added HTTP authentication, CSRF, server ownership, conflict, and safe-not-found tests. | `dbfb23efb73c7145a2d243b83c20952682a77d994153cbe6676022a9722b36d5` |
| `backend/internal/httpapi/profile_controller.go` | Attached custom-item service and four routes. | `38b8a2bebab80c3079dce54d57fe0157e55a8e448c42d7814b05d618150c4965` |
| `backend/internal/repository/custom_food_repository.go` | Added deterministic owner-scoped active-item listing for export. | `df2fd17b43ac8f4d1aab19f1c1b3365952efed837db52eb79c371cac69b9deae` |
| `backend/internal/repository/custom_food_repository_test.go` | Extended task-238 integration coverage with owner-only listing/global isolation. | `69ad995d9e9dcb878acc5d5d0959df6cc2dafbb1a4fbd20fbdaf3e7b4e0037a7` |
| `backend/internal/repository/sql/custom_food_list.sql` | Added parameterized owner predicate and deterministic ordering. | `bbea2a62edcf65ffce98b58455e7f9ee0dcf5cc20e7da2fc8c01639204bc1f33` |
| `backend/internal/repository/types.go` | Extended custom repository with `List`; added lower-camel JSON tags used by API/export projections. | `cd82550a3581d567397663fd8cc02ac03427f95d6982a45d29d13594d8a8d2f4` |
| `backend/internal/userdata/export.go` | Replaced the empty custom-item stub with owner-scoped JSON/CSV items. | `954af1a9659b13397e345e300527480188f49a5aee6c07b669d3b0343d89e3c2` |
| `backend/internal/userdata/export_test.go` | Added exact JSON/CSV private-item inclusion and leakage checks. | `7249cd8a1675261ddc70623ebba3cb4b81ee9df0a8ae0d70abacfdf3ee99013e` |
| `docs/implementation/preparations/task-239.md` | This preparation record. | Self-referential hash intentionally omitted. |

No task-238 migration, design, classification SQL, review evidence, or task-list content was changed by task 239.

## Added or modified executable symbols

### API/configuration

- OpenAPI operations: `postApiV1CustomItem`, `getApiV1CustomItem`, `putApiV1CustomItem`, `deleteApiV1CustomItem`.
- OpenAPI components: parameter `CustomItemId`; response `CustomItem`; schemas `CustomItemFields`, `CustomItemRequest`, `CustomItem`, `CustomItemEnvelope`; modified schema `ExportBundle.customItems`.
- Route configuration in `ProfileController.Routes`: `POST /custom-items`, `GET /custom-items/:itemId`, `PUT /custom-items/:itemId`, `DELETE /custom-items/:itemId`.
- SQL statement/configuration: embedded `customFoodListSQL` and `backend/internal/repository/sql/custom_food_list.sql`.

### Production and repository

- Modified function: `app.NewProduction`.
- Modified behavioral types: `repository.MacroValues` and `repository.ClassificationEntity` JSON serialization; `repository.CustomFoodItemRepository` (`List` contract).
- Added variable: `repository.customFoodListSQL`.
- Added method: `(*repository.PostgresCustomFoodItemRepository).List`.
- Modified test: `repository.TestPostgresCustomFoodItemRepositoryOwnerScopedCRUD`.

### Custom-item service

- Added error variables: `customitem.ErrMissingIdempotencyKey`, `customitem.ErrIdempotencyConflict`.
- Added types: `customitem.Request`, `customitem.CreateRequest`, `customitem.Item`, `customitem.CreateResult`, `customitem.Service`.
- Added functions/methods: `customitem.NewService`, `(*customitem.Service).Create`, `(*customitem.Service).Get`, `(*customitem.Service).Update`, `(*customitem.Service).Delete`, `(*customitem.Service).List`, `customitem.replay`, `customitem.normalizeRequest`, `customitem.normalizedIDs`, `customitem.requestHash`, `customitem.validateIdentity`, `customitem.validationError`, `customitem.toEntity`, `customitem.fromEntity`.
- Added test support types/functions: `memoryItems` and its `GetByID`, `List`, `Create`, `Update`, `Delete`; `memoryIdempotency` and its `GetCheckoutIdempotency`, `StoreCheckoutIdempotency`; `idempotencyScope`; `solidRequest`.
- Added tests: `TestServiceCreateReplayNormalizesBodyAndRejectsKeyReuse`, `TestServiceDerivesOwnershipAndKeepsCrossUserItemsNotFound`.

### HTTP API

- Modified type: `httpapi.ProfileController` (`customItems` dependency).
- Added type: `httpapi.CustomItemService`.
- Added/modified methods: `(*httpapi.ProfileController).WithCustomItems`, `(*httpapi.ProfileController).Routes`, `(*httpapi.ProfileController).CreateCustomItem`, `(*httpapi.ProfileController).GetCustomItem`, `(*httpapi.ProfileController).UpdateCustomItem`, `(*httpapi.ProfileController).DeleteCustomItem`.
- Added helpers: `validateCustomItemCreate`, `validateCustomItemUpdate`, `validateCustomItemBody`, `validateCustomItemID`, `parseCustomItemID`, `customItemData`, `invalidCustomItemBodyError`, `customItemDependencyError`, `customItemError`.
- Added test support symbols: `fakeCustomItemService` and its `Create`, `Get`, `Update`, `Delete`; `customItemBody`.
- Added tests: `TestProfileControllerCustomItemRoutesRequireAuthenticationAndCSRF`, `TestProfileControllerCustomItemRejectsClientOwnershipAndMapsSafeErrors`.

### Export

- Modified types: `userdata.ExportService`, `userdata.ExportBundle`.
- Added type: `userdata.CustomItemExporter`.
- Added method: `(*userdata.ExportService).WithCustomItems`.
- Modified functions: `(*userdata.ExportService).buildBundle`, `userdata.encodeCSV`.
- Added test support type/method: `memoryExportCustomItems`, `(*memoryExportCustomItems).List`.
- Modified test: `TestExportServiceBuildsJSONAndCSV`.

## Verification criteria evidence

| Criterion | Evidence |
|---|---|
| Server-derived ownership | HTTP requests have no accepted owner field; strict decoding rejects `ownerId`; controller passes only JWT `UserID`; service `toEntity` supplies `OwnerID`. Covered by service and HTTP tests. |
| CSRF on mutations | All POST/PUT/DELETE routes declare `RequiresCSRF: true`; HTTP tests receive 403 without CSRF on each mutation and succeed with a valid token. |
| Stable create replay | `Service.Create` normalizes accepted fields, hashes deterministic JSON, stores/reloads the response in the shared `mutation_idempotency_keys` repository scope, and the service test proves the same item/status is replayed without a second create. |
| 409 on changed normalized body | Service returns `ErrIdempotencyConflict`; HTTP maps it to structured `409 idempotency_key_conflict`; service and HTTP tests cover both layers. |
| Exact JSON/CSV inclusion | Export test decodes JSON into `ExportBundle`, parses CSV with `encoding/csv`, decodes the custom-item JSON cell, and deep-compares the projections. |
| No global leakage | Repository integration test proves owner lists contain only that owner's custom IDs and not global curated IDs; export uses only this owner-scoped list and omits `ownerId`. |
| 401 anonymous | HTTP test exercises all four routes anonymously and verifies structured `401 unauthorized`. |
| Ownership-safe not-found | Repository/service tests prove cross-user get/update/delete return typed not-found while the owner's item remains; HTTP test verifies sanitized structured `404 not_found`. |
| OpenAPI-compatible errors | OpenAPI declares shared error envelopes for 400/401/403/404/409/500/503/504; handlers use `AppError`; Redocly validates the document. |

## Commands and results

All commands ran from `/home/wiktor/Work/mealswapp` unless a backend working directory is stated.

1. `git rev-parse HEAD && git status --short && git diff --name-only && git ls-files --others --exclude-standard`
   - Exit 0. Captured baseline commit and dirty paths before task inspection.
2. Initial focused compile/test from `backend/`: `go test ./internal/customitem ./internal/httpapi ./internal/userdata ./internal/repository ./internal/app`
   - First run identified the expected stale Phase 03 CSV-stub assertion; after replacing it with concrete custom-item assertions, all listed packages passed.
3. Focused tests and vet from `backend/`: `go test ./internal/customitem ./internal/httpapi ./internal/userdata ./internal/repository ./internal/app && go vet ./internal/customitem ./internal/httpapi ./internal/userdata ./internal/repository ./internal/app`
   - Exit 0. All focused packages passed; vet passed.
4. `python3 scripts/validate-traceability.py`
   - Final exit 0: `Traceability validation passed.` An intermediate run identified missing helper comments; those comments were added before the final pass.
5. `python3 scripts/validate-task-list.py`
   - Exit 0: `Task-list validation passed: 263 sequential tasks with ordered dependencies.`
6. `npx --no-install redocly lint api/openapi.yaml`
   - Exit 0: OpenAPI valid. One pre-existing warning remains for the OAuth callback's redirect-only 302 response (`operation-2xx-response`, line 235); task 239 added no warning.
7. Full backend tests from `backend/`: `go test ./...`
   - Exit 0. Every command/internal package passed; packages without tests were reported normally.
8. Focused race detection from `backend/`: `go test -race ./internal/customitem ./internal/httpapi ./internal/userdata`
   - Exit 0. All three packages passed with no race report.
9. Vulnerability scan from `backend/`: `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...`
   - Exit 0: zero reachable vulnerabilities. The scanner noted 18 vulnerabilities in required modules that are not called by this code.
10. Focused coverage from `backend/`: `go test ./internal/customitem ./internal/httpapi ./internal/userdata ./internal/repository -coverprofile=/tmp/task-239-coverage.out && go tool cover -func=/tmp/task-239-coverage.out`
    - Exit 0. Package coverage: customitem 63.4%, httpapi 87.2%, userdata 98.1%, repository 93.3%; combined focused profile 89.7%.
11. Final task-list integrity: `sha256sum docs/implementation/02_TASK_LIST.md` and `rg -n '^\\| 239 \\|' docs/implementation/02_TASK_LIST.md`
    - Exit 0. Hash matches baseline and row 239 remains `OPEN`.

## Security assessment and risks

- Trust boundaries: JWT identity and CSRF are enforced before handlers; JSON is strictly decoded with unknown ownership fields rejected; repository SQL is embedded and parameterized; owner IDs are present in every private read/update/delete/list predicate; API/export projections omit owner metadata and global catalog rows.
- Idempotency follows the repository's existing cross-phase completed-response contract. A residual infrastructure edge exists because custom-item creation and persistence of the completed idempotency response are separate database operations, matching the existing checkout implementation: a process/database failure in that narrow interval can commit an item without a replay record, and simultaneous first attempts handled by different application instances are not serialized as one transaction. Sequential retries and changed-body reuse are verified. Making the mutation and response claim fully atomic would require a dedicated transactional repository contract and is not introduced here to avoid changing the established cross-phase schema/API beyond task 239.
- Focused combined coverage is 89.7%, below the repository's phase-end 100% goal. The task's required behavior branches are directly tested; remaining uncovered statements are primarily existing package code and defensive dependency/error branches. Any phase-level exception decision belongs in `docs/implementation/04_OPEN.md`; this task did not alter that file.
- Redocly's single warning is pre-existing and concerns an OAuth 302 callback, not the custom-item API.

## Repair record after independent review (2026-07-21)

This section supersedes stale conclusions, hashes, symbol names, coverage figures, and risks above. Historical baseline and first-preparation evidence is retained for auditability.

### Repair baseline and task-list integrity

- Review read in full: `docs/implementation/reviews/task-239-review.md`, SHA-256 `5ae07e3cfb0d48fe607c065afc268268ecf41a40751520f929d5bb9e876395e8`.
- Fixed Git baseline remains HEAD `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Repair-start task-list SHA-256: `b43ae3a5cb4d283d317e157b1c53b0dfdf0c72e09390074806d14d0307e3f27d`.
- Repair-final task-list SHA-256: `b43ae3a5cb4d283d317e157b1c53b0dfdf0c72e09390074806d14d0307e3f27d`.
- Task 239 remains `PREPARED`. No task-list content or status was edited during repair.
- Pre-existing task-238 migration, design, classification SQL, preparation/review, and task-list changes shown by `git status` were preserved.

### Final task-239 changed paths and hashes

| Path | Final task-239/repair responsibility | Final SHA-256 |
|---|---|---|
| `api/openapi.yaml` | CRUD/export contract plus runtime-aligned required fields, limits, and collection bounds. | `a3a5e5a21cce21616e8403ce44af5c0cc0185f166d5db7aa1b33793e0d4ef6cc` |
| `backend/internal/app/app.go` | Production custom-item and exporter wiring. | `a3ef58e9f6c0c7d307bbaf07a37748bf4f82a5ab1c1fda655918b0129d88e17a` |
| `backend/internal/customitem/service.go` | Owner CRUD, normalization/validation, hash calculation, and atomic repository claim consumption. | `5484ec85693392344b797217567d0d9556fd3f9b90e50c64d4a9b6e25e8183a0` |
| `backend/internal/customitem/service_test.go` | Replay/conflict/ownership, concurrency, validation, CRUD, and error-path tests. | `87ec8d9e65cb33c8a0ae9ab80912508c0d115e37f92e44fb030a3ae64f9791f5` |
| `backend/internal/httpapi/custom_item_controller.go` | CRUD handlers, strict request validation, and structured error mapping. | `4ea8018aa044b3ab34ee54d8391e9dd4cd3a08dc911a800888d9daec791d4d0a` |
| `backend/internal/httpapi/custom_item_controller_test.go` | Auth/CSRF/ownership plus adversarial schema/domain and micronutrient-error tests. | `8736ea21c6f98e7ace97afd7df61d304c9b1b220b3244cf9cedd948d615da7a3` |
| `backend/internal/httpapi/profile_controller.go` | Four authenticated custom-item route registrations. | `38b8a2bebab80c3079dce54d57fe0157e55a8e448c42d7814b05d618150c4965` |
| `backend/internal/repository/types.go` | Atomic claim DTO/result/encoder and repository contract. | `2e1850e3ab05a65f616c11c7b2eef29b68d1f43fd9c06ee01c3d0ac08a407fc8` |
| `backend/internal/repository/custom_food_repository.go` | Owner list and one-transaction item/claim create implementation. | `d3db6f12f12fa2fff03e94598678d85821554c74d12b64d918ff2b304abc5943` |
| `backend/internal/repository/custom_food_repository_test.go` | Real-PostgreSQL concurrency, rollback, conflict, vocabulary, and owner-isolation tests. | `6aa278305a3eceffaae2e5dfba66e31c10bf466d3237c935030ff05f985923f1` |
| `backend/internal/repository/sql/custom_food_create_claim.sql` | Insert fixed-scope in-progress claim with conflict serialization. | `cee2ba752978a2f1d92fe1ed9a79a5fddd354f1ed3ddbf440bd9e1894169642f` |
| `backend/internal/repository/sql/custom_food_create_claim_get.sql` | Lock and read the existing fixed-scope claim. | `b05de071c9229e6010687c64c5ca7142319da4dc9a1795a825d8b936a4ed13dd` |
| `backend/internal/repository/sql/custom_food_create_claim_complete.sql` | Persist and return canonical JSONB response in the same transaction. | `ec446199102b97e4c9cee4c2e02774a5614fad19aa5e75ff0a4b2dc877d9a5f5` |
| `backend/internal/repository/sql/custom_food_list.sql` | Deterministic active owner-only export list. | `bbea2a62edcf65ffce98b58455e7f9ee0dcf5cc20e7da2fc8c01639204bc1f33` |
| `backend/internal/userdata/export.go` | Owner JSON export, exact empty/nonempty CSV rows, and CSV/JSON error propagation. | `3035d0f177abb0417b19c4cd266b466b804b20848153ba1d092e4b9caeb855af` |
| `backend/internal/userdata/export_test.go` | Exact empty/nonempty CSV, JSON inclusion/isolation, and encoding-failure tests. | `66716453803b88440a05ffc8ddeb62805f58ed4be30895682f934802fff1ed31` |
| `docs/implementation/04_OPEN.md` | Explicitly authorized, precise Task-239 changed-coverage exception. | `845f54856391540bf241eea52c59cf04655b8c75cb0b875d577d39383e0c3fa1` |
| `docs/implementation/preparations/task-239.md` | Complete preparation and repair evidence. | Self-referential hash omitted. |

### Final added or modified executable symbols

- Contract/configuration: OpenAPI operations `postApiV1CustomItem`, `getApiV1CustomItem`, `putApiV1CustomItem`, `deleteApiV1CustomItem`; components `CustomItemId`, `CustomItemFields`, `CustomItemRequest`, `CustomItem`, `CustomItemEnvelope`, `ExportBundle.customItems`; `ProfileController.Routes`; `app.NewProduction`; SQL bindings `customFoodListSQL`, `customFoodCreateClaimSQL`, `customFoodCreateClaimGetSQL`, and `customFoodCreateClaimCompleteSQL`.
- Repository: JSON behavior of `MacroValues` and `ClassificationEntity`; types `CustomFoodItemCreateClaim`, `CustomFoodItemCreateClaimResult`, `CustomFoodItemResponseEncoder`, `customFoodCreateRecord`; interface `CustomFoodItemRepository`; functions/methods `NewPostgresCustomFoodItemRepository`, `GetByID`, `List`, `ClaimCreate`, `Create`, `createCustomFoodItemInTransaction`, `scanCustomFoodCreateClaim`, `validateCustomFoodCreateClaim`, `Update`, `Delete`, `hydrateClassifications`, `replaceClassifications`, and `validateCustomFoodIdentity`.
- Service: `ErrMissingIdempotencyKey`, `ErrIdempotencyConflict`; types `Request`, `CreateRequest`, `Item`, `CreateResult`, `Service`; functions/methods `NewService`, `Create`, `Get`, `Update`, `Delete`, `List`, `createResultFromClaim`, `ValidateRequest`, `validNonnegative`, `validOptionalPositive`, `validateDensity`, `normalizedIDs`, `requestHash`, `validateIdentity`, `validationError`, `toEntity`, and `fromEntity`.
- HTTP: `CustomItemService`; `ProfileController.customItems`, `WithCustomItems`, `CreateCustomItem`, `GetCustomItem`, `UpdateCustomItem`, `DeleteCustomItem`; helpers `validateCustomItemCreate`, `validateCustomItemUpdate`, `validateCustomItemBody`, `decodeCustomItemRequest`, `hasDuplicateUUID`, `customItemRequest`, `validateCustomItemID`, `parseCustomItemID`, `customItemData`, `invalidCustomItemBodyError`, `customItemDependencyError`, and `customItemError`.
- Export: `CustomItemExporter`; `ExportService.customItems`, `WithCustomItems`, `BuildExport`, `buildBundle`; `ExportBundle.CustomItems`; `encodeCSV`.
- Test symbols: `memoryItems`, `memoryClaim`, their repository methods, `solidRequest`, `TestServiceCreateReplayNormalizesBodyAndRejectsKeyReuse`, `TestServiceDerivesOwnershipAndKeepsCrossUserItemsNotFound`, `TestServiceConcurrentCreateHasOneSideEffect`, `TestValidateRequestRejectsDomainViolations`, `TestServiceClaimAndDependencyErrorBranches`, `TestServiceCRUDListAndInvalidRequestsHaveExpectedSideEffects`; `fakeCustomItemService`, its methods, `customItemBody`, `TestProfileControllerCustomItemRoutesRequireAuthenticationAndCSRF`, `TestProfileControllerCustomItemRejectsClientOwnershipAndMapsSafeErrors`, `TestProfileControllerCustomItemRejectsSchemaAndDomainViolationsBeforeService`, `TestProfileControllerCustomItemMapsInvalidMicronutrientsToStructuredValidation`; `TestPostgresCustomFoodItemRepositoryAtomicCreateClaim` and modifications to the owner/error/validation repository tests; `TestEncodeCSVCustomItemsEmptyAndNonemptyExact`, `memoryExportCustomItems.List`, and modifications to export service/error tests.

### Repair behavior evidence

- `ClaimCreate` now inserts/locks the cross-phase key, validates/persists one custom item, serializes the owner-free response, and completes the claim inside one PostgreSQL transaction. Concurrent identical requests serialize; the loser replays the committed JSONB bytes. A different normalized hash returns conflict. Any repository, vocabulary, or encoder failure rolls back both item and claim.
- HTTP decoding now rejects missing/null required fields, unknown fields, incomplete/negative/non-finite macros, solid macro totals above 100, invalid optional positive measures, inconsistent liquid density metadata, invalid/unsupported image URI schemes, invalid/duplicate/oversized classifications, and invalid micronutrient names/values before mutation. Repository `ErrorKindInvalidMicronutrientKey` maps to sanitized structured `400 validation_failed`.
- Empty CSV export again emits the canonical `customItems,count,0` row. Nonempty CSV contains exact owner-only custom-item JSON cells; writer and marshal errors propagate.

### Final commands and results

All commands exited `0` unless noted.

1. Focused behavior: `cd backend && go test -count=1 ./internal/customitem ./internal/httpapi ./internal/userdata ./internal/repository ./internal/app` — all passed, including real PostgreSQL concurrent claim and rollback tests.
2. Focused changed-package coverage: `cd backend && go test -count=1 -coverpkg=./internal/customitem,./internal/httpapi,./internal/repository,./internal/userdata -coverprofile=/tmp/task239-repair.cover ./internal/customitem ./internal/httpapi ./internal/repository ./internal/userdata && go tool cover -func=/tmp/task239-repair.cover` — combined `90.5%`; package-local figures from the canonical aggregate are customitem `86.9%`, httpapi `87.4%`, repository `93.1%`, userdata `97.0%`. The owner-authorized precise exception is in `docs/implementation/04_OPEN.md`.
3. Full backend: `cd backend && go test -count=1 ./...` — every package passed.
4. Full race detector: `cd backend && go test -race -count=1 -p 1 ./...` — every package passed, no race report.
5. Static analysis: `cd backend && go vet ./...` — passed with no output.
6. Security: `cd backend && go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` — zero reachable vulnerabilities; 18 required-module findings are not called.
7. OpenAPI: `npx --no-install redocly lint api/openapi.yaml` — valid; one pre-existing accepted OAuth callback `302`-only warning.
8. Generated contract drift: `cd frontend && bun run check:api-types` — generated API types current.
9. Traceability/task-list/diff: `python3 scripts/validate-traceability.py`, `python3 scripts/validate-task-list.py`, and `git diff --check` — all passed; task-list validator reports 263 ordered tasks.
10. Canonical aggregate: `python3 scripts/check.py --output logs/task-239-repair-check.html` — passed and wrote the report. This includes local-stack checks, backend tests/coverage/security, OpenAPI/type drift, Phase 03 CSV UAT, frontend build/unit/coverage, and maintained Playwright suites. Phase 03 UAT now passes; expected frontend dev-proxy refusal logs occurred in mocked browser tests without failures.

### Residual risks

- Requests sharing one owner/key intentionally serialize for the transaction duration. This is bounded to a single key and prevents duplicate side effects; database availability/latency remains the operational dependency.
- The project 100% line-coverage goal is not met for the broad changed-package denominator. The exact measured exception and below-100 safety-relevant symbols are recorded in `docs/implementation/04_OPEN.md`; no required Task-239 behavior is waived.
- Redocly retains one unrelated, pre-existing warning for the OAuth redirect operation. No Task-239 OpenAPI warning remains.

## Second repair cycle after independent re-review (2026-07-21)

This section supersedes prior Task-239 hashes and coverage measurements for the paths named here. Earlier preparation and first-repair evidence remains as historical audit context.

### Baseline and scope integrity

- Fixed Git baseline remains HEAD `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Second-review evidence read in full: `docs/implementation/reviews/task-239-review.md`, SHA-256 `ae21ac2d610f71fb15f51e7d302ca1f88ffec85599fcdd41c9750866a889ce2a`.
- Repair-start and repair-final task-list SHA-256: `b43ae3a5cb4d283d317e157b1c53b0dfdf0c72e09390074806d14d0307e3f27d`.
- Task 239 remains `PREPARED`; no task-list content or status was edited.
- The repair is limited to duplicate-name conflict classification, persisted-text NUL validation, defensive PostgreSQL classification, their tests/contracts, coverage evidence, and this preparation record. All unrelated dirty paths and prior Task-239 fixes were preserved.

### Second-cycle changed paths and final hashes

| Path | Second-cycle change | Final SHA-256 |
|---|---|---|
| `api/openapi.yaml` | Declared NUL-excluding patterns for persisted custom-item strings and micronutrient property names. | `a0d411eeff2267e123df92556c78752b7e3fb86348f019a55f1e91c90cd78f25` |
| `backend/internal/customitem/service.go` | Preserved ordinary resource conflicts, mapped only dedicated idempotency conflicts, and added shared persisted-text validation. | `5f7c1bd6d95f875cde5dda2ac5358dc72300ab4517b9cea906bc4e8eac63ceb5` |
| `backend/internal/customitem/service_test.go` | Added generic conflict and all persisted-string NUL regressions; updated in-memory idempotency classification. | `ce9baa8fe82cb1fd239c5818af8bad58164c6a8acb7793f6bd42477436ef5527` |
| `backend/internal/httpapi/custom_item_controller_test.go` | Added generic structured duplicate conflict and escaped-NUL provenance HTTP regressions. | `8c1332b22e03c53cf3c0be46680933a13257f37359b62e178188bf07613d789b` |
| `backend/internal/repository/errors.go` | Added the dedicated `ErrorKindIdempotencyConflict` classification. | `4423cf862534cd5612800032309386e175d889d9f1428fb2069c72b4bd4c9a09` |
| `backend/internal/repository/postgres.go` | Classified SQLSTATE `22021` as validation defensively. | `2dc903f7954876014f6f94b2ae399680c950d47acda634ec970af6329d507046` |
| `backend/internal/repository/custom_food_repository.go` | Returned the dedicated kind only for body-hash key reuse. | `0b7035b27b6270afe532289c65f1a08ea7547302ee823e1a79c5ae9c0cb0dc5b` |
| `backend/internal/repository/custom_food_repository_test.go` | Added real-PostgreSQL different-key duplicate rollback/classification and NUL validation tests. | `768fa6dfdc91e83684c4393f4af78f03add618cf27c39119586ad52207f2d803` |
| `backend/internal/repository/postgres_repository_test.go` | Added explicit `22021 -> ErrorKindValidation` mapping coverage. | `14f0eb3a2585474d7ef76708b2fdce20b493bdd3ad27a4ee50f49a329061e18f` |
| `docs/implementation/04_OPEN.md` | Remeasured and validated the authorized Task-239 coverage exception. | `7096d49594e73bdef9b5a37e38294cad8b13edf53495da161d7506924d177cbf` |
| `docs/implementation/preparations/task-239.md` | Added this complete second-cycle evidence. | Self-referential hash omitted. |

### Added or modified executable symbols

- OpenAPI `CustomItemFields`: modified `name`, `densitySourceProvider`, `densitySourceFoodId`, `densitySourceKind`, `micros.propertyNames`, and `imageUrl` validation schemas.
- Repository: added `repository.ErrorKindIdempotencyConflict`; modified `repository.mapPostgresError`; modified `(*repository.PostgresCustomFoodItemRepository).ClaimCreate`.
- Service: modified `(*customitem.Service).Create` and `customitem.ValidateRequest`; added `customitem.validText`.
- Test support: modified `memoryItems` with `claimErr`; modified `(*memoryItems).ClaimCreate`.
- Tests: added `TestServiceCreatePreservesResourceConflictClassification` and `TestProfileControllerCustomItemRejectsEscapedNULProvenanceBeforeService`; extended `TestValidateRequestRejectsDomainViolations`, `TestProfileControllerCustomItemRejectsClientOwnershipAndMapsSafeErrors`, `TestPostgresCustomFoodItemRepositoryAtomicCreateClaim`, `TestPostgresCustomFoodItemRepositoryValidation`, and `TestMapPostgresError`.

### Regression behavior

- A same owner/key with a changed normalized body now returns `ErrorKindIdempotencyConflict`, which the service converts to `ErrIdempotencyConflict` and HTTP exposes as `409 idempotency_key_conflict`.
- A new key whose item name conflicts with an existing owner item remains `ErrorKindConflict`; HTTP exposes the documented sanitized `409 conflict`. The transaction rolls back the new claim and item.
- `validText` rejects NUL and enforces rune limits across name, density provider/food ID/kind, image URL, and micronutrient keys before hashing or mutation. HTTP tests submit escaped `\\u0000` separately in all three density provenance fields and prove the service is not called.
- PostgreSQL SQLSTATE `22021` maps to `ErrorKindValidation` as defense in depth. A real repository call with a NUL provenance value returns validation rather than internal error.

### Commands and results

All commands exited `0`.

1. Focused regression/full affected packages: `cd backend && go test -count=1 ./internal/customitem ./internal/httpapi ./internal/repository ./internal/userdata ./internal/app` — all passed, including real PostgreSQL rollback/classification tests.
2. Focused coverage: `cd backend && go test -count=1 -coverpkg=./internal/customitem,./internal/httpapi,./internal/repository,./internal/userdata -coverprofile=/tmp/task239-second-repair.cover ./internal/customitem ./internal/httpapi ./internal/repository ./internal/userdata` — combined `90.5%`. Package-local rerun: customitem `88.4%`, httpapi `87.5%`, repository `93.1%`, userdata `97.0%`. `Service.Create` is `85.0%`, `ValidateRequest` `95.0%`, and new `validText` `100.0%`. The precise authorized exception in `docs/implementation/04_OPEN.md` was updated to these exact measurements and confirms no required behavior is waived.
3. Full backend: `cd backend && go test -count=1 ./...` — every package passed.
4. Full race detector: `cd backend && go test -race -count=1 -p 1 ./...` — every package passed with no race report.
5. Static/security checks: `cd backend && go vet ./...` passed with no output; `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` found zero reachable vulnerabilities (18 required-module findings are not called).
6. Contract checks: `npx --no-install redocly lint api/openapi.yaml` validated the API with only the pre-existing accepted OAuth `302` warning; `cd frontend && bun run check:api-types` reported generated types current.
7. Repository validators: `python3 scripts/validate-traceability.py`, `python3 scripts/validate-task-list.py`, and `git diff --check` passed; the task validator reports 263 ordered tasks.
8. Aggregate: `python3 scripts/check.py --output logs/task-239-second-repair-check.html` passed and wrote the report. It includes local-stack/migration checks, Phase 02/03 UAT, backend normal/race/coverage/security checks, OpenAPI/type drift, frontend build/unit/coverage, frontend verification, and all maintained Playwright suites. Expected mocked-browser proxy `ECONNREFUSED`/anonymous `401` logs did not cause failures.

### Residual risks

- The accepted changed-package coverage exception remains; it is current, exact, and does not waive either repaired behavior or any prior Task-239 acceptance criterion.
- SQLSTATE `22021` is now treated as client validation throughout repository code. This is appropriate for invalid text encoding/NUL input; internal database diagnostics remain hidden by the HTTP layer.
- Redocly retains one unrelated pre-existing OAuth redirect warning. No Task-239 warning or unresolved blocking/important defect remains in this preparation cycle.

## Third repair cycle after independent re-review (2026-07-21)

This section supersedes prior Task-239 hashes and contract/projection conclusions for the paths named here. Earlier evidence remains as historical audit context.

### Baseline and scope integrity

- Fixed Git baseline remains HEAD `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Fresh review evidence read in full: `docs/implementation/reviews/task-239-review.md`, SHA-256 `492d08a050a273adceefb6032ba6c6a0e2b08121aa2af22f03bc576d3ffcc93f`.
- Repair-start and repair-final task-list SHA-256: `b43ae3a5cb4d283d317e157b1c53b0dfdf0c72e09390074806d14d0307e3f27d`.
- Task 239 remains `PREPARED`; no task-list content or status was edited.
- This cycle is limited to custom-item name contract parity, generated-contract drift protection, classification DTO/export projection, regressions, coverage remeasurement, and preparation evidence. All prior fixes and unrelated dirty paths were preserved.

### Third-cycle changed paths and final hashes

| Path | Third-cycle change | Final SHA-256 |
|---|---|---|
| `api/openapi.yaml` | Made `CustomItemFields.name` reject escaped NUL and require at least one non-whitespace character. | `af5a676d54220079d5f852139a57e0737fee7ffa4e3ca595a6ed302417d4d0c7` |
| `backend/internal/customitem/service.go` | Added an explicit public classification summary DTO and stripped repository hierarchy during entity projection. | `4bc9eb6ae297aec1b1030a23084143d27edf9f807e02f07073d4bce541b3975b` |
| `backend/internal/customitem/service_test.go` | Added direct DTO serialization and nil/empty projection regressions. | `26625eb023653d82e60c1b58e51b6994984e8178e4906c5470caa59736c4012d` |
| `backend/internal/httpapi/custom_item_controller_test.go` | Added whitespace/NUL name rejection and hierarchical classification response regressions. | `5bcb03a1265e890b7085f21725262a6fe03ff0ab3dfbf63dd8ca2c7bfb2a9836` |
| `backend/internal/repository/custom_food_repository_test.go` | Proved the internal repository entity still hydrates a real parent ID before the public boundary strips it. | `6db5e2daf6948a219468cf31b6aa4f93ed1a33cecb391052cfdd7a0c9fce6bfb` |
| `backend/internal/userdata/export_test.go` | Added hierarchical source fixtures and exact JSON/CSV assertions that `parentId` is absent. | `8c06da74ac1e0c2ff41f93e93f2b4a3f8277537a80d06e3f9fd10733e46ac264` |
| `frontend/src/lib/api/generated.ts` | Regenerated `CustomItemRequest`, `CustomItem`, and `CustomItemEnvelope` against the canonical summary contract. | `5b0fcfbfa06477e5d42685ae0568b7508e3624bad201bb330454dd93673d0fd2` |
| `scripts/generate-api-types.py` | Added custom-item name and classification schema drift validation plus generated custom-item types. | `be2dc9eafbd31916606cc726a0d98de4d83f82b57dbcbd89e335d4b2fdd71526` |
| `scripts/test_generate_api_types.py` | Added positive and adversarial name/classification contract drift tests. | `e59be27156c4a25808d243e7cb94eccea39a917e73a28a2233148a35a717c00f` |
| `docs/implementation/04_OPEN.md` | Remeasured and validated the precise Task-239 changed-coverage exception. | `9a91d4cfe32ed9eb1f85d08dd668cb44cac3635f5a9d4dff368fdd425c22b5fa` |
| `docs/implementation/preparations/task-239.md` | Added this complete third-cycle evidence. | Self-referential hash omitted. |

### Added or modified executable symbols

- Contract/generated code: modified OpenAPI `CustomItemFields.name`; modified generator constants `REQUIRED_MARKERS`, `CUSTOM_ITEM_NAME_RULE`, and `GENERATED`; added `custom_item_contract_mismatches`; modified `main`; generated TypeScript interfaces `CustomItemRequest`, `CustomItem`, and `CustomItemEnvelope`.
- Service projection: added `customitem.ClassificationSummary` and `classificationSummaries`; modified `customitem.Item` and `fromEntity`.
- Tests: added `TestFromEntityStripsClassificationHierarchyFromPublicProjection`, `TestProfileControllerCustomItemClassificationProjectionOmitsParentID`, `test_custom_item_name_and_classification_contracts_match_generated_types`, and `test_custom_item_name_or_parent_projection_drift_is_rejected`; extended `TestProfileControllerCustomItemRejectsSchemaAndDomainViolationsBeforeService`, `TestPostgresCustomFoodItemRepositoryOwnerCRUDAndClassificationValidation`, `TestEncodeCSVCustomItemsEmptyAndNonemptyExact`, and `TestExportServiceFormatsAndIsolation`.

### Regression behavior

- The OpenAPI name pattern now matches runtime trimming/validation: surrounding whitespace remains valid, whitespace-only names fail, and decoded NUL fails. Generator drift checks pin that pattern, the closed `ClassificationSummary` property set, and both custom-item classification references.
- Repository entities retain `ParentID` for internal hierarchy behavior. `fromEntity` now maps those entities to an owner-safe, hierarchy-free `ClassificationSummary` containing exactly `id`, `name`, and `kind`. HTTP JSON and both JSON/CSV exports consume that DTO, so `parentId` cannot cross the public boundary.
- Regression fixtures use an actual root/child classification pair and prove internal hydration occurs while every public serialization omits `parentId`.

### Commands and results

All commands exited `0` unless explicitly noted.

1. Focused affected packages: `cd backend && go test -count=1 ./internal/customitem ./internal/httpapi ./internal/userdata ./internal/repository ./internal/app` — all passed, including real PostgreSQL hierarchy hydration.
2. Contract regressions: `python3 -m unittest scripts.test_generate_api_types.GeneratedApiTypeTests.test_custom_item_name_and_classification_contracts_match_generated_types scripts.test_generate_api_types.GeneratedApiTypeTests.test_custom_item_name_or_parent_projection_drift_is_rejected` — both passed. The positive test evaluates the OpenAPI regex against surrounding whitespace, whitespace-only input, and decoded NUL; the adversarial test proves missing name constraints and a projected `parentId` are rejected.
3. Generated type drift/type checking: `python3 scripts/generate-api-types.py`, `python3 scripts/generate-api-types.py --check`, `cd frontend && bun run check:api-types`, and `cd frontend && bun run typecheck` — generation completed, drift checks reported current, and TypeScript passed.
4. OpenAPI: `npx --no-install redocly lint api/openapi.yaml` — valid with only the pre-existing accepted OAuth callback `302`-only warning.
5. Focused coverage: `cd backend && go test -count=1 -coverpkg=./internal/customitem,./internal/httpapi,./internal/repository,./internal/userdata -coverprofile=/tmp/task239-third-repair.cover ./internal/customitem ./internal/httpapi ./internal/repository ./internal/userdata` — combined `90.6%`; package-local customitem `90.6%`, httpapi `87.4%` (canonical aggregate `87.5%`), repository `93.1%`, userdata `97.0%`. Changed symbols: `Service.Create` `85.0%`, `ValidateRequest` `95.0%`, `fromEntity` `100.0%`, `classificationSummaries` `100.0%`, HTTP `customItemData` `100.0%`, repository `ClaimCreate` `81.2%`, and export `encodeCSV` `96.3%`. The precise accepted exception is in `docs/implementation/04_OPEN.md`; it waives neither repaired contract.
6. Full backend: `cd backend && go test -count=1 ./...` — every package passed.
7. Full race detector: `cd backend && go test -race -count=1 -p 1 ./...` — every package passed with no race report.
8. Static/security checks: `cd backend && go vet ./...` passed with no output; `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` found zero reachable vulnerabilities (18 required-module findings are not called).
9. Repository validators: `python3 scripts/validate-traceability.py`, `python3 scripts/validate-task-list.py`, and `git diff --check` passed; the task validator reports 263 ordered tasks.
10. Aggregate: `python3 scripts/check.py --output logs/task-239-third-repair-check.html` passed and wrote the report. It includes local stack/migrations, Phase 02/03 UAT, backend normal/race/coverage/security checks, OpenAPI/type drift, frontend typecheck/build/unit/coverage/verification, and maintained Playwright suites (`237` passed, `3` skipped in the complete browser run). Expected mocked-browser proxy refusal logs caused no failures.

### Residual risks

- The accepted broad changed-package coverage exception remains below the project goal. Its exact current measurements and below-100 changed symbols are documented in `docs/implementation/04_OPEN.md`; all new DTO and drift helpers are fully exercised, and neither review defect is waived.
- Internal repository consumers can still access hierarchy as designed. Public safety depends on constructing API/export payloads through `customitem.Item`; its classification field type now makes accidental `ParentID` serialization impossible without an explicit contract change.
- Redocly retains one unrelated pre-existing OAuth redirect warning. No Task-239 OpenAPI warning or unresolved blocking/important defect remains.
