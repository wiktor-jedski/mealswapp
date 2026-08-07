# Task 298 Preparation Evidence

Date: 2026-08-07
Task: 298 — Permanent Custom Food Item Deletion
Branch: `task-298-permanent-custom-food-deletion`
Preserved implementation commit: `164e9f56`

## Scope and sources

This preparation covers only Task 298. The task-list file was not modified.
The implementation follows `DESIGN-008: AccountDeleter`, with the private-item
visibility and account-erasure boundaries in SW-REQ-043 and SW-REQ-073. The
dependency contract from Task 297 is consumed but not changed.

Task 298 implementation commits, in order, are:

- `6cd6bc47` — implement task 298 permanent custom food deletion
- `ec022859` — repair task 298 deletion concurrency and markers
- `531f8373` — wire task 298 marker maintenance contract
- `3bbd86af` — finish task 298 production deletion safeguards
- `d83f912f` — complete task 298 worker and behavioral coverage
- `41174bd1` — harden task 298 create purge and migration races
- `164e9f56` — repair task 298 deletion safeguards

## Implementation inventory

| File | Task 298 surface |
|---|---|
| `api/openapi.yaml` | Permanent custom-item DELETE response and account-export saved-diet contract |
| `database/migrations/000031_permanent_custom_food_deletion.up.sql` | Retry-marker table, legacy soft-delete purge, legacy marker materialization, claim cleanup, expiry index |
| `database/migrations/000031_permanent_custom_food_deletion.down.sql` | Marker-table rollback |
| `backend/internal/repository/custom_food_repository.go` | `ClaimCreate`, `Delete`, `PurgeExpiredDeletedCustomFoodCreateKeys`, `rejectDeletedCustomFoodCreateRetry`, and atomic create/delete coordination |
| `backend/internal/repository/types.go` | `CustomFoodDeletionConflict`, `SavedDietDeletionReference`, and maintenance repository contract |
| `backend/internal/repository/sql/custom_food_*.sql` | Owner lock, bounded affected-diet projection, hard delete, payload-free retry marker, marker check/purge, and active-only reads/lists |
| `backend/internal/repository/custom_food_deletion_contract_test.go` | SQL contract guards for ownership, erasure precedence, marker scope, and expiry |
| `backend/internal/repository/custom_food_repository_test.go` | Repository deletion, classification cascade, and `IncludeDeleted` regression coverage |
| `backend/internal/repository/task298_permanent_custom_food_deletion_integration_test.go` | PostgreSQL migration legacy-purge and delayed-replay proof |
| `backend/internal/httpapi/custom_item_controller.go` | Authenticated permanent DELETE route and bounded `custom_item_in_use` error projection |
| `backend/internal/httpapi/router.go` | Protected custom-item DELETE route registration |
| `backend/internal/httpapi/custom_item_controller_test.go` | HTTP conflict projection coverage |
| `backend/internal/maintenance/custom_food_markers.go` | Expired-marker purge and scheduler |
| `backend/internal/maintenance/custom_food_markers_test.go` | Purge scheduler and dependency/error coverage |
| `backend/cmd/purge-custom-food-markers/main.go` | Operational marker-purge command |
| `backend/cmd/worker/main.go` | Production worker maintenance wiring |
| `backend/internal/app/task298_permanent_custom_food_deletion_integration_test.go` | PostgreSQL/API/export owner isolation, conflict, hard delete, replay, expiry, reuse, and erasure proof |
| `frontend/src/lib/api/account-data-client.ts` | `custom_food_item` export decoding and bounded deletion-conflict error mapping |
| `frontend/src/lib/api/account-data-client.test.ts` | Export and conflict-decoder regression coverage |
| `frontend/src/lib/api/generated.ts` | Generated account-data contract updates |
| `frontend/src/lib/components/AdminPrivateData.svelte` | Irreversible confirmation, conflict guidance, bounded affected-diet display, and fail-closed refresh |
| `frontend/src/lib/components/AdminPrivateData.test.ts` | Component contract and traceability coverage |
| `frontend/tests/admin-private-data.spec.ts` | Desktop/mobile deletion confirmation and conflict recovery scenarios |
| `frontend/tests/task261-real-admin-flow.spec.ts` | Updated real admin deletion confirmation |
| `scripts/generate-api-types.py` | Source-contract generation for the updated API surface |
| `docs/implementation/evidence/task-298-repair.md` | Repair findings and earlier focused evidence |

## Verification criteria

| Criterion | Direct evidence | Result |
|---|---|---|
| Referenced deletion returns bounded owner-scoped conflict without mutation | `TestTask298PermanentCustomFoodDeletionPostgresAndExport`; `custom_food_delete_references.sql` | PASS |
| Cross-owner and repeated DELETE reveal no item or diet data | Same PostgreSQL/API test; owner predicates in `custom_food_delete_lock.sql` and `custom_food_delete_references.sql` | PASS |
| Unreferenced deletion hard-deletes the item and classification links | Same PostgreSQL/API test; `custom_food_delete_hard.sql` and FK cascade | PASS |
| Active/legacy deleted items cannot be recovered and same-name creation succeeds | Same PostgreSQL/API test; `TestTask298MigrationMaterializesLegacyRetryMarker`; active-only read/list SQL | PASS |
| Retry marker is payload-free, blocks exact replay, expires, purges, and cascades on erasure | Same tests; `deleted_custom_food_create_keys` FK and expiry purge | PASS |
| JSON/CSV account export removes deleted content immediately | Same PostgreSQL/API test and `account-data-client.test.ts` | PASS |
| UI requires irreversible confirmation and gives actionable diet guidance | `frontend/tests/admin-private-data.spec.ts` in desktop and mobile projects | PASS |
| Focused backend/frontend/browser, contract, traceability, and diff checks pass | Verification table below | PASS, except external Task 284 predecessor gate |

## Verification commands

| Command | Result |
|---|---|
| `python3 scripts/run-task284-acceptance.py` | BLOCKED/exit 1 by pre-existing unsynchronized predecessor findings: `P08-SWR043-ACCEPT-01`, `P08-SWR072-STEP-01`, `P08-SWR073-STEP-01` |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/repository ./internal/app -run 'TestTask298' -count=1` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/repository ./internal/app -run 'TestTask298' -count=1` | PASS |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | PASS — 568 tests |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck` | PASS |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | PASS |
| `cd frontend && MEALSWAPP_PLAYWRIGHT_REUSE_BUILD=1 BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/admin-private-data.spec.ts` | PASS — 6 tests across desktop/mobile projects |
| `python3 scripts/check.py --quick` | PASS |
| `python3 scripts/validate-traceability.py` | PASS |
| `python3 scripts/generate-api-types.py --check` | PASS |
| `npm exec --yes --package=@redocly/cli@2.31.5 -- redocly lint api/openapi.yaml` | PASS with the repository’s pre-existing ignored OAuth callback `302` warning |
| `git diff --check` | PASS |

## Risks and blockers

- `docs/implementation/02_TASK_LIST.md` remains unchanged and still records
  Task 298 as `OPEN`, as required by the task instructions. Task 297 is also
  still `OPEN`, so no task-list status transition is claimed here.
- The default Task 284 acceptance runner is not a fresh passing predecessor:
  its report validator exits 1 because the three named non-pass criteria lack
  synchronized unresolved findings. This is an external predecessor/report
  blocker; no Task 284 or finding-ledger files were changed.
- The full aggregate release gate was not requested; the supported quick gate
  passed. No review or integration was performed.
