# Task 238 Preparation — User-Owned Custom Item Persistence

## Current scope, status, and baseline confidence

- Task: **238 — Phase 08 User-Owned Custom Item Persistence**.
- Fixed implementation reference and current `HEAD`: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Baseline confidence: **HIGH**. The original Task 238 preparation began from that exact `HEAD` with an empty `git status --short`; all Task 238 implementation paths remain distinguishable as working-tree changes against it.
- Current task-list state is `PREPARED`. The repair agent did not edit `docs/implementation/02_TASK_LIST.md`; its existing `OPEN -> PREPARED` change appeared after the original preparation and was preserved unchanged.
- Independent review source: `docs/implementation/reviews/task-238-review.md`.
- Repair date: 2026-07-21, Europe/Warsaw.
- No unrelated file was cleaned, reverted, staged, or rewritten.

## Independent-review repairs

1. **Changed-symbol coverage:** added focused fault-injection tests for invalid identity, invalid row data, row/query/iteration failures, update validation/SQL/rollback failures, delete SQL failure, classification clear/attach failures, and the ignored-unknown-kind path. Fresh coverage reports every executable symbol in `custom_food_repository.go` at **100.0% line coverage**. Repository-package coverage increased from **92.8% to 93.6%**. No Task 238 exception is needed, so `docs/implementation/04_OPEN.md` was not edited.
2. **Input-slice mutation:** `replaceClassifications` no longer combines caller slices with `append(foodCategories, culinaryRoles...)`. It iterates each input independently, preventing spare capacity in `foodCategories` from being overwritten. A regression test supplies a sentinel in that spare capacity and proves both inputs remain unchanged.
3. **Normalized duplicate names:** the real PostgreSQL integration test now creates `"My Tofu"` and attempts the same-owner duplicate `"  mY tOfU  "`, proving the migration's `lower(btrim(name))` uniqueness contract. Different-owner, curated-table, and post-soft-delete behavior remain covered.

## Exact repair delta

| Path | Added/modified symbols and behavior |
| --- | --- |
| `backend/internal/repository/custom_food_repository.go` | Modified `(*PostgresCustomFoodItemRepository).replaceClassifications`: independent nested iteration over `foodCategories` and `culinaryRoles`, with no aliasing `append`. |
| `backend/internal/repository/custom_food_repository_test.go` | Modified `TestPostgresCustomFoodItemRepositoryOwnerScopedCRUD` with mixed-case/whitespace duplicate coverage. Added `TestPostgresCustomFoodItemRepositoryErrorBranches`, `TestReplaceCustomFoodClassificationsPreservesInputs`, and test helper `customFoodFixtureValues`. |
| `docs/implementation/preparations/task-238.md` | Replaced stale OPEN/coverage evidence with this current repair record. |

## Complete Task 238 implementation surface

| Path | Symbols or persistence units |
| --- | --- |
| `docs/design/DESIGN-005.md` | Added private `CustomFoodItemEntity`, mandatory owner/non-disclosure rules, per-owner normalized-name scope, curated isolation, and `CustomFoodItemRepository` contract. |
| `backend/internal/repository/types.go` | Added `CustomFoodItemEntity` and `CustomFoodItemRepository`. |
| `backend/internal/repository/custom_food_repository.go` | Embedded SQL variables `customFoodCreateSQL`, `customFoodGetByIDSQL`, `customFoodUpdateSQL`, `customFoodSoftDeleteSQL`, `customFoodClearClassificationsSQL`, `customFoodAttachClassificationSQL`, `customFoodListClassificationsSQL`; type `PostgresCustomFoodItemRepository`; compile assertion; constructor `NewPostgresCustomFoodItemRepository`; methods `GetByID`, `Create`, `Update`, `Delete`, `hydrateClassifications`, `replaceClassifications`; helper `validateCustomFoodIdentity`. |
| `backend/internal/repository/custom_food_repository_test.go` | Embedded fixture variable `testCustomFoodOwnerlessCreateSQL`; tests `TestPostgresCustomFoodItemRepositoryOwnerScopedCRUD`, `TestPostgresCustomFoodItemRepositoryErrorBranches`, `TestReplaceCustomFoodClassificationsPreservesInputs`, `TestPostgresCustomFoodItemRepositoryValidation`; helper `customFoodFixtureValues`. |
| `backend/internal/repository/sql/custom_food_create.sql` | Parameterized private-item create. |
| `backend/internal/repository/sql/custom_food_get_by_id.sql` | Parameterized owner-scoped read. |
| `backend/internal/repository/sql/custom_food_update.sql` | Parameterized owner-scoped active-row update. |
| `backend/internal/repository/sql/custom_food_soft_delete.sql` | Parameterized owner-scoped soft deletion. |
| `backend/internal/repository/sql/custom_food_clear_classifications.sql` | Classification replacement clear. |
| `backend/internal/repository/sql/custom_food_attach_classification.sql` | Parameterized classification assignment. |
| `backend/internal/repository/sql/custom_food_list_classifications.sql` | Parameterized classification hydration. |
| `backend/internal/repository/sql/classification_is_in_use.sql` | Modified embedded `classificationIsInUseSQL` content to include custom assignments. |
| `backend/internal/repository/sql/testdata/custom_food_ownerless_create.sql` | Direct ownerless-row rejection fixture. |
| `database/migrations/000025_user_owned_custom_food_items.up.sql` | Added `custom_food_items`; constraints `custom_food_items_name_not_blank`, `custom_food_items_micronutrients_object`, `custom_food_items_liquid_density_required`; indexes `custom_food_items_owner_active_name_idx`, `custom_food_items_owner_idx`; table `custom_food_item_classifications`; index `custom_food_item_classifications_classification_idx`; schema version 25. |
| `database/migrations/000025_user_owned_custom_food_items.down.sql` | Reverses all migration-25 objects and bookkeeping. |
| `docs/implementation/preparations/task-238.md` | Current preparation and repair evidence. |

The independently created `docs/implementation/reviews/task-238-review.md` and the existing task-list status transition are part of the dirty worktree but were not authored or edited by this repair.

## Changed executable-symbol coverage

Fresh command:

`GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/repository -coverprofile=/tmp/task-238-after.coverage.out`

| Symbol | Before review repair | Current |
| --- | ---: | ---: |
| `NewPostgresCustomFoodItemRepository` | 100.0% | 100.0% |
| `(*PostgresCustomFoodItemRepository).GetByID` | 81.8% | 100.0% |
| `(*PostgresCustomFoodItemRepository).Create` | 100.0% | 100.0% |
| `(*PostgresCustomFoodItemRepository).Update` | 72.7% | 100.0% |
| `(*PostgresCustomFoodItemRepository).Delete` | 75.0% | 100.0% |
| `(*PostgresCustomFoodItemRepository).hydrateClassifications` | 81.2% | 100.0% |
| `(*PostgresCustomFoodItemRepository).replaceClassifications` | 66.7% | 100.0% |
| `validateCustomFoodIdentity` | 80.0% | 100.0% |

Repository-package total is 93.6%; its remaining uncovered statements pre-exist outside Task 238 and are governed by existing project/phase dispositions. Task 238 introduces no below-100 executable symbol and needs no new exception.

## Acceptance criteria after repair

| Criterion | Current evidence | Result |
| --- | --- | --- |
| Forward/down migrations | Focused PostgreSQL reset and aggregate local-stack migration up/down/up cycle include migration 25. | PASS |
| Owner-scoped CRUD and same-user visibility | Real PostgreSQL CRUD integration plus focused error tests. | PASS |
| Cross-user non-disclosure | Cross-owner read/update/delete return typed `not_found`; every statement binds owner and item ID. | PASS |
| Global curated isolation | Same-named rows coexist in separate tables and neither repository reads the other's ID. | PASS |
| Duplicate-name scope and normalization | Same-owner `"My Tofu"` versus `"  mY tOfU  "` conflicts; different owner/global succeeds; soft deletion releases the active reservation. | PASS |
| Canonical units | Metric values persist and established `g -> oz`/`ml -> fl_oz` boundary conversion remains covered. | PASS |
| Macro and active micronutrient validation | Valid canonical data persists; negative macros, alias `Na`, and inactive `Legacy` fail with typed errors. | PASS |
| Liquid-density provenance | Valid density/provenance persists; missing density or provenance fails at repository and schema boundaries. | PASS |
| Classification invariants | Kind/existence validation, hydration, in-use protection, transactional replacement failure, and caller-slice immutability are covered. | PASS |
| Soft deletion | Active reads hide tombstones, `IncludeDeleted` returns them, and duplicate-name release is covered. | PASS |
| Parameterized embedded SQL | All seven production statements remain positional-parameter files under `backend/internal/repository/sql/`. | PASS |
| No ownerless private row | Nil owner fails before I/O and `owner_id NOT NULL` rejects direct SQL. | PASS |
| Changed executable coverage | Every executable symbol in `custom_food_repository.go` reports 100.0%. | PASS |

Every Task 238 criterion and all three requested review repairs are satisfied.

## Commands and exact results

| Command | Result |
| --- | --- |
| `git rev-parse HEAD` | `81ca40ce00cb667ea29243ed2d34068e11229a69`; fixed reference unchanged. |
| `gofmt -w internal/repository/custom_food_repository.go internal/repository/custom_food_repository_test.go` | PASS. |
| Focused normal test: `go test -count=1 -v ./internal/repository -run 'TestPostgresCustomFoodItemRepository|TestReplaceCustomFoodClassifications'` | PASS — four top-level tests plus all validation subtests. |
| Repository coverage command writing `/tmp/task-238-after.coverage.out` | PASS — package 93.6%; every `custom_food_repository.go` symbol 100.0%. |
| `go tool cover -func=/tmp/task-238-after.coverage.out` filtered to Task 238 symbols | PASS — all eight executable symbols 100.0%. |
| Focused race command with the same test expression | PASS — all focused tests and subtests. |
| `go test -count=1 ./...` from `backend/` | PASS — no failing package output. The execution tool did not expose a numeric exit field after completion, but the command emitted only `ok`/`[no test files]`; the subsequent aggregate rerun also passed the affected repository/migration gates. |
| `go vet ./...` | PASS — no output. |
| `govulncheck@v1.3.0 ./...` | PASS — no vulnerabilities in called code; 18 vulnerable module versions are present but not called. |
| `python3 scripts/validate-task-list.py` | PASS — 263 sequential tasks; current Task 238 state `PREPARED` preserved. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `git diff --check` | PASS. |
| First `python3 scripts/check.py` repair run | FAIL environmental/concurrent — migration 4 hit PostgreSQL `pg_type_typname_nsp_index` duplicate-key creation; process inspection found no remaining overlapping migration afterward. No Task 238 test failed. |
| Quiescent rerun of `python3 scripts/check.py` | PASS — traceability/task-list/Go Doc/OpenAPI/security, migration up/down/up, local readiness, Phase 02/03 UAT, migration tests, and repository tests passed. The only OpenAPI message is the existing ignored OAuth `302`-only warning. |

## Risks and blockers

- No Task 238 implementation, coverage, correctness, or security blocker remains.
- No `docs/implementation/04_OPEN.md` exception was added because all changed executable symbols reached 100% line coverage.
- Account export/deletion integration, authenticated API/idempotency, and owner-scoped listing belong to later Phase 08 tasks and remain outside Task 238.
- The independent review file is now stale with respect to repaired code/tests and should be regenerated by an independent reviewer; this preparation does not rewrite review evidence or change task status.
