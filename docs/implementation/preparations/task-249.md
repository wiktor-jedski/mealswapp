# Task 249 preparation — Curated Import Workflow

## Outcome and scope

- Task: 249, `DESIGN-009: DataImporter`.
- Fixed Git reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Dependencies 30, 33, 34, 43, 246, 247, and 248 were `PASSED`.
- Task 249 remains exactly `PREPARED`; this repair did not edit any task-list cell.
- The implementation adds strict typed-normalized editable-draft confirmation at `POST /api/v1/admin/imports`, natural provider identity replay, durable key-based replay when natural identity is absent, explicit normalized-name merge confirmation, one-pass repository validation, atomic import/audit persistence, trusted imported-density evidence, and generation-guarded post-commit search/similarity invalidation.
- Task 253 owns the consolidated OpenAPI contract, so task 249 did not modify `api/openapi.yaml`; the current source was linted unchanged.

## Baseline and unrelated-work preservation

The initial worktree was already substantially dirty with PASSED Phase 08 dependency work and unrelated later-task work. The initial `git status --short` included modified shared app, repository, search, API, frontend, scripts, and documentation paths plus untracked task 238–248 and 250–252 implementation/review files. Those changes were treated as user-owned. Task ownership is high-confidence for new files and for the explicitly listed shared-file hunks; whole-file diffs against Git `HEAD` are not exclusively task 249 because the shared files were already modified.

| Task-relevant baseline path | Initial state / SHA-256 |
|---|---|
| `backend/internal/app/app.go` | pre-existing modified, `33c22fd95422fe5fbd41b5090c23fcf33a8e4cbf94a6dacf6f0464a869ad0f99` |
| `backend/internal/app/app_test.go` | pre-existing modified, `f267d7813831a91e664355094959c3cfa8ff57e1d096360932f7c9bd503ca9e2` |
| `backend/internal/repository/compliance_repository.go` | pre-existing modified, `57790181b840c7f494e44dbd62a7c69be6210cff79987652db6bd6a3852712f6` |
| `backend/internal/repository/types.go` | pre-existing modified, `7a0069590989b8fe0311c960021c5da87cc3d568b9ca708a7e586b620511f730` |
| `docs/implementation/02_TASK_LIST.md` | pre-existing modified, `206fd6de6aefc40d163c838e34f88486bd25067b22356a62d826fe32c10905c5` |
| `api/openapi.yaml` | pre-existing modified and preserved, `4bbd3ef34268e41a4a37599aa2729ac8589ba0fd34274ae80b27a7e0bbff72f7` |
| `backend/internal/dataimporter/` | absent |
| `backend/internal/httpapi/import_controller.go` and test | absent |
| `backend/internal/repository/curated_import_repository.go` | absent |
| task-249 curated-import SQL statements | absent |

No migration or schema rewrite was required. Imports without external natural identity use the existing shared mutation-idempotency table and an opaque deterministic internal curated-import identity; external provider values remain restricted to `usda` and `openfoodfacts`.

## Review repair — 2026-07-21

The three important findings in `docs/implementation/reviews/task-249-review.md` are repaired without changing `docs/implementation/02_TASK_LIST.md` or weakening the prior immutable replay, canonical-name advisory-lock, and non-imported-status protections:

- F-249-001: `validateCuratedImport` now passes the strictly decoded draft through `dataimporter.NormalizeRequest`, which invokes the typed `curation.InputNormalizer` before the request is stored in Fiber locals or reaches the service/repository. `Service.Confirm` repeats the same boundary defensively. The normalized name, public HTTPS image URL, provider identities, density evidence, provenance kind, macros, measures, and bounded micronutrient values are used for hashing and persistence.
- F-249-002: imported liquid density now requires both a supported density provider (`usda` or `openfoodfacts`) and a non-empty provider source-food identifier at the custom-item and repository invariants. Missing evidence fails before mutation; positive `manual` and `estimated` corrections remain accepted. Provider evidence text is normalized by typed provider rules before persistence.
- F-249-003: similarity calculations now use the same cross-instance mutation generation as search responses. The generation is included in similarity keys, and writes carry the generation observed at lookup through `SimilarityCalculationCacheToken`; `SetIfCurrent` rejects in-flight stale writes after a committed import advances the generation. A live PostgreSQL/Redis test primes a score, confirms a normalized-name merge with changed macros, invalidates after commit, and proves the repeated Substitution Search returns a recomputed score with the new Macro Profile.

Adversarial regressions reject decomposed/multi-space bypasses by persisting the NFC/collapsed result, control characters, loopback HTTP image URLs, over-bound numeric values, imported density without evidence, and stale similarity writes. The live workflow also proves accepted manual, estimated, and trusted-provider density paths and retains sequential/concurrent replay and merge coverage.

## Implementation behavior

### Editable draft and identity validation

- The HTTP boundary strictly rejects malformed UTF-8/JSON, duplicate or unknown keys, missing/null required item/classification/micronutrient fields, and missing macro members.
- The service trims and canonicalizes editable food fields, allowlists provider identity, bounds provider/external identifiers through the shared security normalizer, requires provider and external ID together, and requires a bounded `Idempotency-Key` only when natural identity is absent.
- All standard global food invariants are reused: physical state, finite/nonnegative macros and micros, classification kind/existence, active micronutrient keys, image URL, and liquid density/provenance. A liquid candidate without provider density evidence must be explicitly corrected with positive density and `manual` or `estimated` provenance before confirmation.

### Atomic confirmation, replay, and conflict handling

- Provider/external-ID confirmations acquire a transaction-scoped PostgreSQL advisory lock, then lock/read the natural import row. Completed `imported` rows replay a persisted immutable response DTO containing import ID, food ID, name, physical state, and merge outcome without reloading mutable food data. A changed body or an existing `draft`, `conflict`, or `rejected` row returns the documented provider conflict.
- Imports without natural identity claim the existing administrator-scoped `POST /admin/imports` mutation key. The completed claim stores the same complete immutable response DTO. Same-key/same-body is a no-op replay even after the food is updated or soft-deleted; same-key/different-body is an idempotency conflict.
- Food validation happens once inside the gateway-owned transaction. The active micronutrient vocabulary is loaded once for the single import workflow rather than once per micronutrient or once again during persistence.
- Every confirmation acquires a transaction-scoped advisory lock for the canonical normalized food name after its natural-identity or idempotency-key lock and before lookup. Concurrent absent-name imports therefore serialize: an unconfirmed loser receives confirmation-required, while a confirmed loser merges the winner.
- Food/classification writes, curated-import metadata, idempotency completion, and the request-correlated audit entry commit in one transaction. Replays emit neither an audit entry nor cache invalidation.
- After a new confirmation commits, the shared Redis search generation advances and old search response keys are removed best-effort. Catalog and substitution readers therefore cannot continue serving the pre-import generation.

### Security review

The Go security skill was applied in coding mode. The relevant trust boundaries and controls were inspected:

- Authorization and administrator identity remain server-derived from the verified JWT cookie; body/header role spoofing cannot reach the route.
- CSRF, scoped mutation rate limiting, strict body validation, and required security/admin audit middleware execute before persistence.
- Provider identity, idempotency keys, text, numeric maps, classifications, and URLs are bounded before SQL. SQL remains embedded and parameterized; no user value reaches SQL syntax, paths, or commands.
- Parameterized transaction-scoped advisory locks coordinate identity/key and canonical-name decisions across processes in documented lock order. Every transactional error path rolls back locks, claims, food rows, classifications, imports, and audit state.
- Audit snapshots contain only `status` and `physicalState`; provider payloads, external identifiers, food names, idempotency keys, and request bodies are excluded.
- `govulncheck` found no called vulnerabilities, focused/full race runs found no task-249 race, and `go vet` passed.

## Changed paths and symbols

### Production and persistence

- `backend/internal/dataimporter/service.go`
  - Added errors `ErrMissingIdempotencyKey`, `ErrIdempotencyConflict`, `ErrProviderConflict`, and `ErrNameConfirmation`.
  - Added `Request`, `Result`, `Store`, `Service`, `NewService`, `(*Service).Confirm`, `requestHash`, `toEntity`, and `validationError`.
- `backend/internal/repository/curated_import_repository.go`
  - Added conflict sentinels, eight embedded SQL variables, `(*PostgresAdminImportAuditRepository).ConfirmCuratedImport`, natural/key confirmation paths, canonical-name serialization, immutable replay/metadata DTO encoding and validation, persistence helpers, body hashing, synthetic identity, and boundary validation.
- `backend/internal/repository/sql/curated_import_lock_identity.sql`
  - Added transaction-scoped natural-identity advisory lock statement.
- `backend/internal/repository/sql/curated_import_find_for_update.sql`
  - Added locked natural-identity replay lookup.
- `backend/internal/repository/sql/curated_import_insert.sql`
  - Added immutable imported-state insert.
- `backend/internal/repository/sql/curated_import_food_by_name_for_update.sql`
  - Added locked normalized-name conflict lookup.
- `backend/internal/repository/sql/curated_import_lock_name.sql`
  - Added transaction-scoped canonical normalized-name advisory locking for absent-row serialization.
- `backend/internal/repository/sql/curated_import_create_claim.sql`
  - Added durable key claim for identity-absent imports.
- `backend/internal/repository/sql/curated_import_create_claim_get.sql`
  - Added locked key replay lookup.
- `backend/internal/repository/sql/curated_import_create_claim_complete.sql`
  - Added immutable completed-response update.
- `backend/internal/repository/types.go`
  - Added `CuratedImportConfirmation`, `CuratedImportConfirmationResult`, and `CuratedImportConfirmationRepository`.
- `backend/internal/repository/compliance_repository.go`
  - Added the transactional confirmation compile-time contract and the `food_item/import_food` privacy-safe audit snapshot schema.
- `backend/internal/httpapi/import_controller.go`
  - Added `CuratedImportService`, `CuratedImportInvalidator`, `CuratedImportController`, `NewCuratedImportAdminController`, `(*CuratedImportController).Confirm`, `validateCuratedImport`, `curatedImportAuditSnapshot`, `curatedImportError`, and `curatedImportDependencyError`.
- `backend/internal/app/app.go`
  - Modified `NewProduction` only to compose `dataimporter.Service`, register the audited import controller, and supply post-commit shared search invalidation.

### Tests

- `backend/internal/dataimporter/service_test.go`
  - Added `importStoreStub`, its confirmation method, `importExecutorStub`, three service test functions, and `validRequest`.
- `backend/internal/dataimporter/integration_test.go`
  - Added the live workflow plus immutable replay, replay comparison, concurrent name-confirmation, count, and absence helpers.
  - The PostgreSQL test now covers natural and key replay after mutation and soft deletion, every non-imported natural status, simultaneous confirmed/unconfirmed normalized-name collisions, both changed-body conflicts, explicit name merge, classification/micronutrient/liquid validation, rollback, audit counts, and immediate catalog/substitution visibility.
- `backend/internal/httpapi/import_controller_test.go`
  - Added `curatedImportServiceStub`, its confirmation method, `curatedImportInvalidatorStub`, its invalidation method, and two HTTP tests covering successful audited confirmation, replay no-op behavior, post-commit invalidation, strict body rejection, and safe conflict codes.
- `backend/internal/app/app_test.go`
  - Modified `TestNewProductionExposesProductionRoutes` only to assert `POST /api/v1/admin/imports` is composed.

## Original preparation verification (superseded by post-review evidence below)

All commands ran on 2026-07-21.

| Command | Result |
|---|---|
| `go test -count=1 ./internal/dataimporter ./internal/httpapi -run 'CuratedImport|ServiceConfirm'` | PASS, including the live PostgreSQL workflow. |
| `go test -count=10 ./internal/dataimporter -run TestCuratedImportTransactionalWorkflow` | PASS 10/10, including concurrent absent-name arbitration. |
| `go test -count=1 ./internal/dataimporter ./internal/httpapi ./internal/app -run 'CuratedImport|ServiceConfirm|TestNewProductionExposesProductionRoutes'` | PASS. |
| focused `-coverpkg` run over dataimporter/repository/httpapi | PASS; core service 90%, natural confirmation 76.9%, key confirmation 77.1%, persistence 75.8%, and immutable replay helpers 75–100%. The unselected dependency-error constructor and defensive DB/encoding branches account for uncovered statements. |
| `go test -race -count=1 ./internal/dataimporter ./internal/httpapi -run 'CuratedImport|ServiceConfirm'` | PASS; no race report. |
| `go vet ./...` | PASS. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS: no vulnerabilities in called/imported code; 18 required-module advisories are not called. |
| `go test -count=1 ./...` | Every task-249 package PASS. Full command FAILS only in preserved `internal/app.TestTask240CustomItemErasureIntegration`: `transactional account cleanup left 2 owner custom items`. |
| `go test -race -count=1 ./...` | Every task-249 package PASS with no race. Full command has the same unrelated task-240 assertion only. |
| `npx --no-install redocly lint api/openapi.yaml` | VALID; one existing OAuth callback 302-only warning remains. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks with ordered dependencies; task 249 remains `PREPARED`. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/generate-api-types.py --check` | PASS; generated API types are current. |
| `git diff --check` | PASS. |
| `python3 scripts/check.py` | Traceability, task list, Go Doc, OpenAPI, script tests, vet, vulnerability scan, local stack, Phase 02/03 UAT, focused backend checks, and frontend verification PASS. Aggregate backend tests stop only at the preserved task-240 cleanup assertion. |

The standalone full and race commands were run before the aggregate local-stack verifier started services and reproduced the same task-240 assertion. Task 249's live integration, repository, service, HTTP, search, app route, focused race, vet, security, API, and traceability evidence is passing.

## Original preparation hashes (superseded by post-review hashes below)

| Path | SHA-256 |
|---|---|
| `backend/internal/app/app.go` | `17d81fe7ce4684f4ca91253abe521d2471224f5895fe1022a72dc3755ee02967` |
| `backend/internal/app/app_test.go` | `7ba6924dc9e04ec8d663b17672ad2e0519c40b285ffaababba38d525f77dcdd4` |
| `backend/internal/dataimporter/integration_test.go` | `a0a94f798f16049fdcafbd749dd98015dbac42f520a1192b273c45246fbb0506` |
| `backend/internal/dataimporter/service.go` | `758f61c6dabc47faaeb59f7332556464e12cb627b7a739ae517b9884c1dcd986` |
| `backend/internal/dataimporter/service_test.go` | `83f8d9d6460cebd35a05f317aeee62141eabf510c8a69f05509aa7c5aa9ef153` |
| `backend/internal/httpapi/import_controller.go` | `275b8e26d3750c89cd14e82569747c79911bc896a381839f2e8666c5597b372a` |
| `backend/internal/httpapi/import_controller_test.go` | `500103ddc90430f6b3222f654af8396c4335b8119e10625a144770b870879bf1` |
| `backend/internal/repository/compliance_repository.go` | `56d69c43de27d8ff2056be0764125ee1682f911a148cb7dc6fc809843dffdb38` |
| `backend/internal/repository/curated_import_repository.go` | `c033ecac1bfd3fcee7947d13e6c16c4f2234d1ad7a56f300177d7fec835e7c65` |
| `backend/internal/repository/types.go` | `57f27717e00d382e03225f1c2a903c66604c4ce4fe02cefa2a950951623f4c83` |
| `backend/internal/repository/sql/curated_import_create_claim.sql` | `d5a1181aad29e748bb152898d50b2bfa7445c925d417edb1c8d57457a30ba975` |
| `backend/internal/repository/sql/curated_import_create_claim_complete.sql` | `b78aaf66d2c2198e2f8c9f58fac8eb00cfa5b37e20be509e6f11d3b58f5875fb` |
| `backend/internal/repository/sql/curated_import_create_claim_get.sql` | `e025bc3c366495ae061466b889b6706bb8b775a621bcae4f31cbab8a0087ff6c` |
| `backend/internal/repository/sql/curated_import_find_for_update.sql` | `f6118fd595dcabe3eaa8047b51f11f93e172f687833b0a40a4942eb7bf26b9a4` |
| `backend/internal/repository/sql/curated_import_food_by_name_for_update.sql` | `290f34871bfe5d395765a32249502208d436e2cdbe9fd0286bbe4810661a797b` |
| `backend/internal/repository/sql/curated_import_insert.sql` | `18f35fc7a30ae996783ffc9501c1bddb1437c958e10283522e807e22cd4f1e53` |
| `backend/internal/repository/sql/curated_import_lock_identity.sql` | `d705ddbcc230b1fe9bf83a5046d163ec60282f9e419afb5c8a8bbfd0d7f017be` |
| `backend/internal/repository/sql/curated_import_lock_name.sql` | `88e61932f17d8747ddf7b80478bd6722f738404634eb9bcb677ab51c9dabad47` |
| unchanged status control `docs/implementation/02_TASK_LIST.md` | `68e19810450d0348b844e3c89eb83977a0382821186d03f6d2474fb02f872151` |

## Post-review repair verification

All commands below ran on 2026-07-21 after the repair.

| Command | Result |
|---|---|
| `go test -count=1 ./internal/dataimporter ./internal/httpapi ./internal/cache ./internal/search ./internal/app -run 'CuratedImport\|ServiceConfirm\|Similarity\|ClassificationGenerationLiveRedis\|NewProductionExposesProductionRoutes'` | PASS. |
| `go test -count=10 ./internal/dataimporter -run 'CuratedImportTransactionalWorkflow\|ConfirmedMergeInvalidatesRedisSimilarity'` | PASS 10/10, including PostgreSQL concurrency and Redis-backed confirmed-merge score recomputation. |
| `go test -race -count=1 ./internal/dataimporter ./internal/httpapi ./internal/cache ./internal/search ./internal/app -run 'CuratedImport\|ServiceConfirm\|Similarity\|ClassificationGenerationLiveRedis\|NewProductionExposesProductionRoutes'` | PASS; no race report. |
| `go test -count=1 ./...` | Every task-249 and repaired shared package PASS. Full command stops only at preserved task-240 `TestTask240CustomItemErasureIntegration`: `transactional account cleanup left 2 owner custom items`. |
| `go test -race -count=1 ./...` | Every task-249 and repaired shared package PASS with no race. Full command has the same unrelated task-240 assertion only. |
| `go vet ./...` | PASS. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS: zero called/imported vulnerabilities; 18 required-module advisories are not called. |
| `npx --no-install redocly lint api/openapi.yaml` | VALID; the existing OAuth callback 302-only warning remains. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks with ordered dependencies; task 249 remains `PREPARED`. |
| `python3 scripts/generate-api-types.py --check` | PASS. |
| `git diff --check` | PASS. |
| `python3 scripts/check.py` | Traceability, task-list, Go Doc, OpenAPI, script tests, vet, vulnerability scan, local stack, Phase 02/03 UAT, focused backend gates, and frontend verification PASS. Aggregate backend tests stop only at the same preserved task-240 cleanup assertion. |

### Post-review repair hashes

| Path | SHA-256 |
|---|---|
| `backend/internal/app/app.go` | `2f74a8ae0d4880f757ea05bea31f763dd2c3cc61c1bc6127a7294a289d050a26` |
| `backend/internal/cache/classification_generation_integration_test.go` | `4b5ea8d467a74e5722ece4649c860864d3615cead8ab99cb670ea830a836ccc1` |
| `backend/internal/cache/search_cache.go` | `5160bdafd92c2b964328c5828978b957da8fd640d5597a0fc11e47513de20a9c` |
| `backend/internal/cache/search_cache_test.go` | `7a24328fa0f8ce2b529a6851307a75468dafd4ceecf6cb2575e04d216d26de1b` |
| `backend/internal/curation/validation.go` | `114bd3e16d2046964a9aeb594ebd52efcce7e649cb45f934807cdfa457fd9a16` |
| `backend/internal/customitem/service.go` | `28d9981b711f94f57c864b27daf4c83e34952acc088c1f98ed22caf910f0793d` |
| `backend/internal/dataimporter/integration_test.go` | `7241fbd2483356bcf9267cfce57ef7cde855902c2e2bb26a6825ae6cea661515` |
| `backend/internal/dataimporter/service.go` | `1d2f801934ed6f0cdcf101384acfe508f83124311f7f3a3b5b0dbb2fb7be66ae` |
| `backend/internal/dataimporter/service_test.go` | `fa07d14e71557f4cd49080b059de416d1feb183b53bbdd2849712c204ccbc761` |
| `backend/internal/httpapi/import_controller.go` | `04e0e65035302d15501dd44e0ba1327ee1af71f22db97ce733d0df5dd4483de1` |
| `backend/internal/httpapi/import_controller_test.go` | `d2433595cc3168f9ea2afc806ca3be0c371b453f88f899078a92364f73e4da31` |
| `backend/internal/repository/food_repository.go` | `f5f06aca0f8da39b2be26c1fa9f6148958db88b2c4e51b6f2d464e1ef5565844` |
| `backend/internal/repository/postgres_repository_test.go` | `a023af2cd3c041d819194f6f60017e8c3555aa07fc480c457d0a1bc0809096b2` |
| `backend/internal/repository/repository_test.go` | `6e2d1717f32c28a0181835626e2ce7a407308bdf760f93e2fa79fece6c6d8dd6` |
| `backend/internal/search/catalog_service.go` | `f31d095dadde7f4cef48b47c8f404dedd4c2b5652e796dd53b236e7072cac982` |
| `backend/internal/search/substitution_service.go` | `8d5f789a7a5aa4f57318769a8675dc59bd504e547cdb6c0d860712b6fcd6d3a6` |
| `backend/internal/search/substitution_service_test.go` | `fd006f1075b8a18679a4f7d64ad357a19c35df0e21eb05b366ccb7caec873929` |
| unchanged status control `docs/implementation/02_TASK_LIST.md` | `68e19810450d0348b844e3c89eb83977a0382821186d03f6d2474fb02f872151` |

## Handoff

Every task-249 verification clause now has direct service, HTTP, live PostgreSQL, or live Redis/search evidence. The original immutable replay DTOs, atomic canonical-name serialization, advisory-lock order, and non-imported natural-status handling remain covered. The independent review's three newer findings are repaired by typed route normalization, trusted imported-density provenance, and generation-guarded similarity recomputation. Task 249 intentionally remains `PREPARED` per instruction.
