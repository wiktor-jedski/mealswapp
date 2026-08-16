# Review Evidence: Task 249 — Curated Import Workflow

~~~yaml
task_id: 249
component: DataImporter
static_aspect: DESIGN-009: DataImporter
input_status: PREPARED
review_decision: PASSED
decision: PASSED
reviewed_at_utc: 2026-07-21T17:39:03Z
review_agent: Codex independent reviewer
baseline_ref: 81ca40ce00cb667ea29243ed2d34068e11229a69
baseline_confidence: HIGH
code_review_skill_invoked: true
relevant_language_guide: Go, SQL, idempotency, transaction, concurrency, and security guidance
pre_review_gates_passed: true
inventory_complete: true
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
inventory_source_count: 31
audited_symbol_count: 31
blocking_findings: 0
important_findings: 0
optional_findings: 0
~~~

## 1. Task Source

This is a fresh independent final review of PREPARED task 249, `DESIGN-009: DataImporter`. The full `docs/implementation/reviewer-prompt.md` template was read. The task row, preparation report, prior rejected review, `docs/design/01_TECH_STACK.md`, `docs/architecture/ARCH-009.md`, `docs/design/DESIGN-009.md`, and the relevant implementation/open-items controls were read and compared with current source.

Task 249 remains exactly `PREPARED` at `docs/implementation/02_TASK_LIST.md:256`; dependencies 30, 33, 34, 43, 246, 247, and 248 are `PASSED`. The task-list content/status and all production files were preserved. The template requests merging the main branch, but no merge was performed because this is a non-mutating review in a shared dirty worktree and the user explicitly prohibited production/task-list edits.

The fixed preparation reference and current `HEAD` are `81ca40ce00cb667ea29243ed2d34068e11229a69`. The previous review was rejected for three important findings. Its claims were rechecked against current implementation and fresh tests rather than accepted as proof.

## 2. Pre-Review Gates

| Gate | Result | Evidence |
|---|---|---|
| Exact task status | PASS | Task 249 is `PREPARED`; dependencies 30, 33, 34, 43, 246, 247, and 248 are `PASSED`. |
| Review template | PASS | `docs/implementation/reviewer-prompt.md` was read fully. |
| Review skill | PASS | `code-review-skill` was invoked exactly once and its Go, SQL parameterization, transaction, idempotency, concurrency, error-handling, and security guidance was applied. |
| Preparation/prior evidence | PASS | Preparation and prior rejected-review claims were checked for staleness against current source, hashes, fresh focused/live tests, and aggregate gates. |
| Scope control | PASS | Task 249 behavior, direct callers, transaction/audit boundary, search-cache invalidation dependency, designs, architecture, and evidence controls were reviewed. |
| Mutation boundary | PASS | No production file or task-list cell was edited. Only this review evidence file is refreshed. |
| Traceability | PASS | `python3 scripts/validate-traceability.py` passes; reviewed implementation has nearby `Implements DESIGN-*` comments. |

## 3. Review Baseline and Change Surface

The worktree contains broad, user-owned Phase 08 and later-task changes. Whole-file diffs against `HEAD` are therefore not treated as task ownership proof. The task-249 surface was reconstructed from the preparation inventory, current shared-file hunks, design comments, symbol reachability, and direct tests.

The reviewed boundary is the strict admin import request, `dataimporter.Service`, the gateway-owned PostgreSQL transaction and audit coordination, curated-import SQL, food/classification/vocabulary validation helpers, durable natural/key replay, canonical-name serialization, production route composition, and the shared generation used by catalog/substitution caches. OpenAPI remains task 253-owned and was inspected unchanged.

The mutation sequence is `AdminController.transactionalMutation` → `WithMutationAudit` → `Service.Confirm` → `ConfirmCuratedImport`. Food rows, classifications, curated-import metadata, key completion, and the request-correlated audit entry commit together. Replays return immutable stored response DTOs and do not write audit rows or invalidate caches.

## 4. Acceptance Criteria Checklist

| Acceptance behavior | Result | Evidence |
|---|---|---|
| Strict editable-draft validation | PASS | Strict UTF-8/object decoding, duplicate/unknown-key rejection, required fields/macros, typed normalizer, and HTTP negative tests. |
| Successful natural import | PASS | Live PostgreSQL workflow creates the food, classifications, imported metadata, and audit entry. |
| Provider/external-ID natural uniqueness | PASS | Parameterized transaction-scoped advisory identity lock, locked lookup, unique metadata row, and changed-body conflict test. |
| Immutable natural replay after mutable/deleted food | PASS | Stored response DTO is returned without mutable food reload; live test changes name/state, replays, soft-deletes, and replays again. |
| Durable key replay after mutable/deleted food | PASS | Shared mutation-key claim stores the complete immutable DTO; live test proves stable IDs/name/state/merge and no duplicate effects after mutation and soft delete. |
| Different-body key conflict | PASS | Normalized SHA-256 claim comparison returns the documented conflict and leaves counts unchanged. |
| Normalized-name confirmation | PASS | Typed canonical name, transaction-scoped canonical-name advisory lock, `FOR UPDATE` active-row lookup, and sequential/concurrent confirmation tests. |
| Absent-name concurrency | PASS | Repeated live concurrent imports produce one create plus one confirmation-required result, or one create plus one merge when confirmed. |
| Natural replay status handling | PASS | `imported` is the only replayable status; live fixtures for `draft`, `conflict`, and `rejected` return provider conflict. |
| Classification validation | PASS | One transaction-scoped classification validation path checks existence and required kind before writes. |
| Micronutrient validation | PASS | One active vocabulary snapshot is loaded in the transaction; unknown/inactive keys fail before persistence. |
| Liquid density correction | PASS | Missing density fails; positive imported density without trusted provider/source-food evidence fails; manual, estimated, and provider-evidenced paths pass. |
| Atomic repository rollback | PASS | Invalid classification leaves no food/import/key state. |
| Atomic audit rollback | PASS | Invalid audit snapshot returns fail-closed audit error and leaves no food/import/key state. |
| Privacy-safe audit | PASS | Import audit snapshot contains only `status` and `physicalState`; provider identity, body, name, key, and payload are excluded. |
| Admin authorization and mutation controls | PASS | Verified server-derived admin identity, CSRF, scoped rate limit, strict validation, fixed audit metadata, and route composition. |
| Typed curation normalization before hash/dispatch | PASS | `NormalizeRequest` runs before `customitem.ValidateRequest`, body hashing, locals dispatch, and repository confirmation; Unicode/whitespace, control, URL, and numeric regressions pass. |
| SQL and transaction security | PASS | Embedded SQL is parameterized; advisory locks are transaction-scoped; errors roll back all mutation state; no user data reaches SQL syntax or commands. |
| Immediate catalog visibility | PASS | Post-commit shared generation advances and live catalog search finds the imported item. |
| Immediate substitution visibility | PASS | Post-commit generation-versioned similarity cache recomputes after a confirmed merge with changed macros; live Redis regression passes. |
| Replay side effects | PASS | Natural/key replay creates no food/import/audit rows and does not invoke post-commit invalidation. |
| Traceability/evidence controls | PASS | Current hashes, validators, OpenAPI lint, generated-type drift, static/security checks, and evidence validator pass. |

## 5. Changed-Symbol Inventory

| # | File or source set | Reviewed surface |
|---:|---|---|
| 1 | `docs/implementation/reviewer-prompt.md` | Full review template and required output controls |
| 2 | `docs/design/01_TECH_STACK.md` | Go/Fiber/PostgreSQL/Redis/security stack |
| 3 | `docs/architecture/ARCH-009.md` | Administration and DataImporter architecture |
| 4 | `docs/design/DESIGN-009.md` | DataImporter responsibilities, interfaces, states, and errors |
| 5 | `docs/implementation/02_TASK_LIST.md` | Task status, dependencies, and acceptance source |
| 6 | `docs/implementation/04_OPEN.md` | Open assumptions and coverage controls |
| 7 | `docs/implementation/preparations/task-249.md` | Preparation scope, repair claims, commands, and fingerprints |
| 8 | `docs/implementation/reviews/task-249-review.md` | Prior rejected evidence and finding staleness comparison |
| 9 | `backend/internal/app/app.go`, `app_test.go` | Production composition, route wiring, shared generation, and smoke coverage |
| 10 | `backend/internal/dataimporter/service.go` | Normalization, identity selection, hash, mapping, conflict conversion |
| 11 | `backend/internal/dataimporter/service_test.go`, `integration_test.go` | Unit/live import, replay, conflict, rollback, concurrency, density, and search tests |
| 12 | `backend/internal/httpapi/import_controller.go` | Strict boundary, admin mutation, audit projection, safe errors, invalidation |
| 13 | `backend/internal/httpapi/import_controller_test.go`, `curation_validation.go` | HTTP normalization, strict JSON, safe conflict, audit, and replay tests |
| 14 | `backend/internal/httpapi/admin_controller.go` | Admin authorization, middleware ordering, transaction/audit gateway |
| 15 | `backend/internal/curation/validation.go`, `validation_test.go` | Typed curation field normalization and bounds |
| 16 | `backend/internal/customitem/service.go`, `service_test.go` | Shared editable-item validation and density invariant |
| 17 | `backend/internal/security/normalizer.go`, `curation_normalizer_test.go` | Unicode/control/image/provider/identifier security rules |
| 18 | `backend/internal/repository/curated_import_repository.go` | Transactional import, locks, persistence, status, replay DTO |
| 19 | `backend/internal/repository/compliance_repository.go` | Mutation/audit transaction and privacy-safe snapshot schemas |
| 20 | `backend/internal/repository/food_repository.go`, `postgres.go`, `types.go` | Single-pass validation, transaction executor, and contracts |
| 21 | `backend/internal/repository/repository_test.go`, `postgres_repository_test.go` | Repository invariants, SQL behavior, density, and audit rollback evidence |
| 22 | `backend/internal/repository/sql/food_*.sql`, `vocabulary_list_active.sql` | Parameterized food/classification/vocabulary persistence/read statements |
| 23 | `backend/internal/repository/sql/curated_import_create_claim.sql` | Durable absent-identity key claim |
| 24 | `backend/internal/repository/sql/curated_import_create_claim_complete.sql` | Immutable key response completion |
| 25 | `backend/internal/repository/sql/curated_import_create_claim_get.sql` | Locked key replay lookup |
| 26 | `backend/internal/repository/sql/curated_import_find_for_update.sql`, `curated_import_insert.sql` | Natural replay and immutable metadata insert |
| 27 | `backend/internal/repository/sql/curated_import_food_by_name_for_update.sql` | Active normalized-name conflict row lock |
| 28 | `backend/internal/repository/sql/curated_import_lock_identity.sql`, `curated_import_lock_name.sql` | Transaction-scoped identity/name advisory locks |
| 29 | `database/migrations/000002_*`, `000010_*`, `000011_*`, `000021_*` | Food, import/audit, density, and shared idempotency schema contracts |
| 30 | `backend/internal/cache/classification_generation.go`, `classification_invalidator.go`, `search_cache.go` | Shared mutation generation, guarded cache writes, search/similarity keys |
| 31 | `backend/internal/cache/*_test.go`, `backend/internal/search/catalog_service.go`, `substitution_service.go`, `substitution_service_test.go` | Cache generation/rejection, catalog/substitution freshness, and score behavior |

## 6. Function-Level Audit

| # | Audited unit | Result and evidence |
|---:|---|---|
| 1 | Task source/design/architecture controls | PASS — task row and design responsibilities match the reviewed import workflow. |
| 2 | `NewProduction` and route smoke test | PASS — DataImporter and post-commit invalidator are composed on the versioned admin route. |
| 3 | `Service.Confirm` | PASS — normalizes before validation/hash/store, derives admin identity from caller, and maps stable conflicts. |
| 4 | `NormalizeRequest` | PASS — invokes typed curation rules for name, image, providers, identifiers, density evidence/kind, macros, measures, and bounded finite micronutrient values before dispatch. |
| 5 | `requestHash` and `toEntity` | PASS — hash receives the normalized request and entity mapping carries only validated fields. |
| 6 | `validateCuratedImport` | PASS — rejects malformed UTF-8/JSON, duplicates, unknown keys, missing required fields, and invalid typed values before locals/service dispatch. |
| 7 | `CuratedImportController.Confirm` | PASS — uses verified admin, gateway transaction, safe DTO, replay audit suppression, and after-commit invalidation. |
| 8 | `curatedImportError` and dependency mapping | PASS — conflict/validation/dependency responses are explicit and sanitized. |
| 9 | `AdminController.transactionalMutation` | PASS — mutation response is deferred until mutation and audit commit; failed audit aborts the transaction. |
| 10 | `ConfirmCuratedImport` | PASS — rejects malformed claim state and routes natural/key paths through the supplied transaction. |
| 11 | `confirmNaturalCuratedImport` | PASS — locks identity, locks/reloads metadata, requires `imported`, compares body hash, and reconstructs immutable replay without mutable food reads. |
| 12 | `confirmIdempotentCuratedImport` | PASS — durable administrator/method/route/key claim serializes first writer, rejects body drift, and replays immutable response data. |
| 13 | `persistCuratedImport` | PASS — validates once, locks canonical name before lookup, requires explicit merge confirmation, writes food/import metadata atomically, and stores immutable response fields. |
| 14 | `createValidatedCuratedFood` / `updateValidatedCuratedFood` | PASS — uses the caller transaction for food and classification writes and does not repeat full validation. |
| 15 | Replay DTO helpers | PASS — persisted import/key payloads contain and validate the complete public response identity; mutable food state is not reloaded. |
| 16 | Classification/micronutrient validation helpers | PASS — classification kind/existence and one active vocabulary snapshot execute through the gateway transaction. |
| 17 | Density validation in service/custom-item/repository | PASS — imported density requires supported provider plus source-food ID; manual/estimated positive corrections remain accepted. |
| 18 | Curated-import SQL | PASS — values are bound parameters; natural/key/name locks are transaction-scoped and lookup rows use `FOR UPDATE`. |
| 19 | Food/classification/vocabulary SQL | PASS — create/update/read/association/vocabulary statements are embedded and parameterized. |
| 20 | `WithMutationAudit` / audit sanitizer | PASS — audit state is atomic and the import schema allows only bounded status/physical-state metadata. |
| 21 | Transactional live workflow | PASS — successful create, exact replay, body conflicts, classification/micro failures, density paths, and counts pass. |
| 22 | Immutable replay regression | PASS — both natural and absent-identity key replay survive mutable fields and soft deletion with stable results. |
| 23 | Status regression | PASS — `draft`, `conflict`, and `rejected` natural rows never replay as successful imports. |
| 24 | Name-lock concurrency regression | PASS — repeated live concurrent normalized-name requests produce the specified confirmation/merge outcomes. |
| 25 | HTTP and admin security tests | PASS — strict input, admin route controls, safe responses, audit privacy, and replay no-op behavior pass. |
| 26 | Curation/security normalizer tests | PASS — NFC/whitespace, controls, safe HTTPS URLs, provider identifiers, numeric bounds, and metadata-only rejection logging pass. |
| 27 | `ClassificationGeneration` / invalidator | PASS — committed imports advance the cross-instance generation; old search generations are not served. |
| 28 | `SearchResponseStore` | PASS — similarity keys include food-data generation and stale cache-miss writes are rejected atomically. |
| 29 | Catalog/Substitution cache integration | PASS — live confirmed merge with changed macros recomputes the score and exposes updated food state. |
| 30 | SQL/security/error review | PASS — no SQL interpolation, command execution, raw payload logging, admin spoofing, or unsafe error exposure is reachable in the task path. |
| 31 | Full gate and evidence control | PASS — task packages pass; only the preserved Task-240 integration assertion fails in repository-wide aggregate commands. |

## 7. Findings

No blocking, important, or optional findings remain.

The three previous important findings are closed:

1. Immutable natural and key replay now returns the stored response DTO after food mutation and soft deletion without reading mutable food state.
2. Every confirmation acquires the canonical normalized-name advisory lock before the active-row lookup, and live concurrent tests prove confirmation-required versus confirmed merge outcomes.
3. Natural replay is gated on `status == imported`; `draft`, `conflict`, and `rejected` fixtures return provider conflict.

The three findings from the prior review are also repaired and independently reverified:

1. Typed curation normalization runs before request hashing and repository dispatch, with service/HTTP adversarial tests.
2. Positive `imported` liquid density requires supported provider and source-food evidence in both shared validation layers, with live rejected/accepted paths.
3. Similarity calculations use the shared mutation generation in keys and guarded writes, with a live Redis confirmed-merge score recomputation.

## 8. Commands Run

All commands below ran against the current worktree on 2026-07-21. Go commands used repository-local caches. The single repository-wide failure is preserved out-of-scope Task-240 evidence, not a Task-249 failure.

| Command | Result |
|---|---|
| `bash scripts/start-services.sh` | PASS; local PostgreSQL and Redis available. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/dataimporter ./internal/httpapi ./internal/cache ./internal/search ./internal/app -run 'CuratedImport\|ServiceConfirm\|Similarity\|ClassificationGenerationLiveRedis\|NewProductionExposesProductionRoutes'` | PASS; every selected package passed. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=10 ./internal/dataimporter -run 'CuratedImportTransactionalWorkflow\|ConfirmedMergeInvalidatesRedisSimilarity'` | PASS 10/10; PostgreSQL replay/concurrency and Redis merge freshness passed. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race -count=1 ./internal/dataimporter ./internal/httpapi ./internal/cache ./internal/search ./internal/app -run 'CuratedImport\|ServiceConfirm\|Similarity\|ClassificationGenerationLiveRedis\|NewProductionExposesProductionRoutes'` | PASS; no race report. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./...` | FAIL only at preserved `internal/app.TestTask240CustomItemErasureIntegration`: `transactional account cleanup left 2 owner custom items`; all Task-249 and repaired shared packages pass. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race -count=1 ./...` | FAIL at the same preserved Task-240 assertion; all Task-249 and repaired shared packages pass with no race report. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS; zero vulnerabilities in called/imported code; 18 uncalled required-module advisories reported. |
| `npx --no-install redocly lint api/openapi.yaml` | VALID; one existing OAuth callback 302-only warning. |
| `python3 scripts/validate-task-list.py` | PASS; 263 sequential tasks and task 249 remains `PREPARED`. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/generate-api-types.py --check` | PASS; generated API types current. |
| `git diff --check` | PASS. |
| `go test -count=1 -coverprofile=/tmp/task249-rereview.cover ...` plus `go tool cover -func` | PASS; selected packages run; focused selected-package statement reports are dataimporter 88.9%, HTTP 22.2%, cache 57.8%, search 41.6%, and app 54.5%; no Task-249 coverage waiver is claimed. |
| `python3 scripts/check.py --output /tmp/task-249-rereview-check.html` | FAIL only at its repository-wide `go test ./...` stage for the same preserved Task-240 assertion; local stack, UAT, frontend verification, traceability, task list, OpenAPI, vet, vulnerability, and focused gates pass. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-249-review.md` | PASS after this refresh; structural evidence is valid. |

## 9. Files Inspected and Staleness Fingerprints

The current preparation fingerprint is `fa202f74d38af58a4ea8e8ee0924c309427e45ff8906069fabffac333f0a1a1b`. The prior rejected review fingerprint before this refresh is `e4d8528ff36c09046b452e15a18c81249d30a04f6d550240d87a412c8c3a4267`. The task-list fingerprint is `68e19810450d0348b844e3c89eb83977a0382821186d03f6d2474fb02f872151` and remained unchanged during this review.

All reviewed implementation, direct dependency, SQL, test, design, preparation, task-list, and template files were fingerprinted. Hashes are authoritative for this dirty shared worktree.

| File | SHA-256 |
|---|---|
| `docs/implementation/reviewer-prompt.md` | `92c9b71361a50868becf0b9a9895071bdd657e8c092afb8c1b19691cb569386d` |
| `docs/design/01_TECH_STACK.md` | `64e2cf45ec039db597244678b17e8028f4705b86dcad01e7051e3e686d6f9338` |
| `docs/architecture/ARCH-009.md` | `153607ef21b23caad6805f8c0f77e3ad9584dd8ab20dc7c86a54134905a95e91` |
| `docs/design/DESIGN-009.md` | `85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b` |
| `docs/implementation/02_TASK_LIST.md` | `68e19810450d0348b844e3c89eb83977a0382821186d03f6d2474fb02f872151` |
| `docs/implementation/04_OPEN.md` | `4b703eca5a8b6207ce0e87fc0ea23c9df255ac892ab8d93e8b4f8ff1f318e4d` |
| `docs/implementation/preparations/task-249.md` | `fa202f74d38af58a4ea8e8ee0924c309427e45ff8906069fabffac333f0a1a1b` |
| `backend/internal/app/app.go` | `2f74a8ae0d4880f757ea05bea31f763dd2c3cc61c1bc6127a7294a289d050a26` |
| `backend/internal/app/app_test.go` | `7ba6924dc9e04ec8d663b17672ad2e0519c40b285ffaababba38d525f77dcdd4` |
| `backend/internal/cache/classification_generation.go` | `5ebeafb2d1e0ec6fd0944679d2725c2d43e580d18cf35b77927db59d952bc3e7` |
| `backend/internal/cache/classification_generation_integration_test.go` | `4b5ea8d467a74e5722ece4649c860864d3615cead8ab99cb670ea830a836ccc1` |
| `backend/internal/cache/classification_invalidator.go` | `f378e9daef4e183645548cf7cd319a34dccd613561028597ae2ba8ca84eed989` |
| `backend/internal/cache/classification_invalidator_test.go` | `b149b8a2f3fd25ef311adced0554c74f753b34ca48620bc1f4e7c67b749125be` |
| `backend/internal/cache/search_cache.go` | `5160bdafd92c2b964328c5828978b957da8fd640d5597a0fc11e47513de20a9c` |
| `backend/internal/cache/search_cache_test.go` | `7a24328fa0f8ce2b529a6851307a75468dafd4ceecf6cb2575e04d216d26de1b` |
| `backend/internal/curation/validation.go` | `114bd3e16d2046964a9aeb594ebd52efcce7e649cb45f934807cdfa457fd9a16` |
| `backend/internal/curation/validation_test.go` | `c3d127aa322dcf639acbe1b31a020617eee7457b0048b9533aca7e98c57072d1` |
| `backend/internal/customitem/service.go` | `28d9981b711f94f57c864b27daf4c83e34952acc088c1f98ed22caf910f0793d` |
| `backend/internal/dataimporter/service.go` | `1d2f801934ed6f0cdcf101384acfe508f83124311f7f3a3b5b0dbb2fb7be66ae` |
| `backend/internal/dataimporter/service_test.go` | `fa07d14e71557f4cd49080b059de416d1feb183b53bbdd2849712c204ccbc761` |
| `backend/internal/dataimporter/integration_test.go` | `7241fbd2483356bcf9267cfce57ef7cde855902c2e2bb26a6825ae6cea661515` |
| `backend/internal/httpapi/import_controller.go` | `04e0e65035302d15501dd44e0ba1327ee1af71f22db97ce733d0df5dd4483de1` |
| `backend/internal/httpapi/import_controller_test.go` | `d2433595cc3168f9ea2afc806ca3be0c371b453f88f899078a92364f73e4da31` |
| `backend/internal/httpapi/curation_validation.go` | `14cd4a46838d84fb643fd9448e8f61c6909f09bf66203ae00d7570d7664c46f1` |
| `backend/internal/httpapi/admin_controller.go` | `cb1f9bcd0896fadad29c29b8ede663b13c3a2250d99c13c5efe70d05739f730f` |
| `backend/internal/repository/compliance_repository.go` | `56d69c43de27d8ff2056be0764125ee1682f911a148cb7dc6fc809843dffdb38` |
| `backend/internal/repository/curated_import_repository.go` | `c033ecac1bfd3fcee7947d13e6c16c4f2234d1ad7a56f300177d7fec835e7c65` |
| `backend/internal/repository/food_repository.go` | `f5f06aca0f8da39b2be26c1fa9f6148958db88b2c4e51b6f2d464e1ef5565844` |
| `backend/internal/repository/postgres.go` | `2dc903f7954876014f6f94b2ae399680c950d47acda634ec970af6329d507046` |
| `backend/internal/repository/types.go` | `57f27717e00d382e03225f1c2a903c66604c4ce4fe02cefa2a950951623f4c83` |
| `backend/internal/repository/sql/curated_import_create_claim.sql` | `d5a1181aad29e748bb152898d50b2bfa7445c925d417edb1c8d57457a30ba975` |
| `backend/internal/repository/sql/curated_import_create_claim_complete.sql` | `b78aaf66d2c2198e2f8c9f58fac8eb00cfa5b37e20be509e6f11d3b58f5875fb` |
| `backend/internal/repository/sql/curated_import_create_claim_get.sql` | `e025bc3c366495ae061466b889b6706bb8b775a621bcae4f31cbab8a0087ff6c` |
| `backend/internal/repository/sql/curated_import_find_for_update.sql` | `f6118fd595dcabe3eaa8047b51f11f93e172f687833b0a40a4942eb7bf26b9a4` |
| `backend/internal/repository/sql/curated_import_food_by_name_for_update.sql` | `290f34871bfe5d395765a32249502208d436e2cdbe9fd0286bbe4810661a797b` |
| `backend/internal/repository/sql/curated_import_insert.sql` | `18f35fc7a30ae996783ffc9501c1bddb1437c958e10283522e807e22cd4f1e53` |
| `backend/internal/repository/sql/curated_import_lock_identity.sql` | `d705ddbcc230b1fe9bf83a5046d163ec60282f9e419afb5c8a8bbfd0d7f017be` |
| `backend/internal/repository/sql/curated_import_lock_name.sql` | `88e61932f17d8747ddf7b80478bd6722f738404634eb9bcb677ab51c9dabad47` |
| `backend/internal/search/catalog_service.go` | `f31d095dadde7f4cef48b47c8f404dedd4c2b5652e796dd53b236e7072cac982` |
| `backend/internal/search/substitution_service.go` | `8d5f789a7a5aa4f57318769a8675dc59bd504e547cdb6c0d860712b6fcd6d3a6` |
| `backend/internal/search/substitution_service_test.go` | `fd006f1075b8a18679a4f7d64ad357a19c35df0e21eb05b366ccb7caec873929` |
| `backend/internal/security/normalizer.go` | `f87732321090d144229227b4573cf5ff1155d80f95c4e68da44a513c55802607` |
| `backend/internal/security/curation_normalizer_test.go` | `28ea79df82b789d677cd5a4f1649afb51311e757cd936782fe3b7b3e1191b749` |
| `database/migrations/000002_food_items.up.sql` | `13012e45e3e4b20d71e19a1c913d2f4b20f060a61d43f1cb5c1b53b8ccc160f6` |
| `database/migrations/000010_admin_import_audit.up.sql` | `e5a8b6d756b6bf7511f609e05381af74f2cff8ae3a123db6cdbfb5cdca1dd40e` |
| `database/migrations/000011_liquid_density_constraint.up.sql` | `b27777e85809b7b144000b525f8fbddfc70fda8255b9c5a1ae5ccd16550b2878` |
| `database/migrations/000021_mutation_idempotency.up.sql` | `354e77c40320c5c31711241e130a460ee9be86b974f20baa6c4266d3f609a08b` |
| `api/openapi.yaml` | `f3143eaa58c136fbe8f7db6652ef030cb851d831f62dded65935544bb460ab46` |

## 10. Coverage and Exceptions

The selected command reports package statement coverage of 88.9% for `dataimporter`; the other selected packages include broad shared code and are not used as a task-wide coverage claim. The task’s direct behavior is covered by live/unit/HTTP tests for every listed acceptance criterion. No Task-249 coverage exception or waiver is claimed; `optional_findings` is zero.

The repository-wide ordinary and race test commands, and the aggregate `scripts/check.py`, stop only at the preserved `internal/app.TestTask240CustomItemErasureIntegration` assertion (`transactional account cleanup left 2 owner custom items`). Task-249 and its repaired direct cache/search dependencies pass in those runs. This unrelated failure is recorded as an environment/worktree exception, not waived Task-249 behavior and not a review finding.

## 11. Negative and Regression Checks

- Natural and key replay return the original immutable import ID, food ID, name, physical state, and merge outcome after mutable food updates and soft deletion; no mutable food reload is reachable from replay branches.
- Natural identity uses a transaction-scoped advisory lock followed by a locked metadata row; key identity uses the durable administrator/method/route/key claim and `FOR UPDATE` replay lookup.
- Canonical normalized-name locking precedes the active-row lookup. Repeated live concurrency runs reject an unconfirmed loser and merge a confirmed loser, without duplicate active food/import/audit rows.
- `draft`, `conflict`, and `rejected` natural metadata rows are not replayable, even with unchanged body hash.
- NFC/collapsed whitespace name normalization occurs before both request storage and SHA-256 hashing; control characters, unsafe loopback/HTTP image URLs, and over-bound numeric values fail before store dispatch.
- Imported liquid density without provider/source-food evidence fails in service and repository validation. Positive manual, estimated, and trusted USDA/OpenFoodFacts evidence paths pass.
- Classifications are checked for required kind/existence; active micronutrient vocabulary is loaded once in the transaction; invalid rows roll back.
- Audit failure and repository validation failure leave no food, import, idempotency claim, classification, or audit row.
- Audit snapshots exclude provider identity, external ID, request body, idempotency key, food name, and raw provider payload.
- SQL is embedded and parameterized; advisory-lock expressions contain only SQL constants around bound values. No command execution or dynamic SQL identifier path is reachable.
- The shared mutation generation is included in similarity keys and guarded writes. A confirmed merge with changed macros produces a different score on the next substitution search through live Redis.
- New imports are immediately visible in catalog and substitution search; cache replays do not trigger invalidation.
- Admin identity is server-derived from verified authentication context; CSRF, rate limit, validation, audit, and safe response mapping remain in the route gateway.

## 12. Decision

`PASSED` — no blocking or important findings remain for Task 249. All prior findings are repaired and directly regression-tested. The full repository test command has one preserved unrelated Task-240 assertion; every Task-249 package and repaired shared package passes, including repeated live PostgreSQL/Redis and race runs. The task-list row remains `PREPARED` because this review was explicitly forbidden from editing task-list status.

## 13. Repair Context

No further Task-249 repair is required. The exact repaired behaviors verified by this review are:

1. Immutable natural/key replay DTOs survive mutable and soft-deleted food state; identity and canonical-name advisory locks serialize concurrent confirmations; only `imported` natural rows replay.
2. Typed curation normalization runs before hashing and repository dispatch, including canonical names, public HTTPS image URLs, provider identities/IDs, finite bounded values, and density provenance.
3. Imported liquid density requires trusted provider plus source-food evidence, while positive manual/estimated corrections remain valid.
4. A shared Redis-backed mutation generation versions similarity calculations and rejects stale in-flight writes after committed import merges.

No production source or task-list edit was made during this re-review. The task-list SHA-256 remained `68e19810450d0348b844e3c89eb83977a0382821186d03f6d2474fb02f872151`.
