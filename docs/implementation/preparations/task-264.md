# Task 264 Preparation — Manual Item Search Cache Invalidation

## Scope and baseline

- Task: `264`, Phase 08.01 Manual Item Search Cache Invalidation.
- Fixed baseline: `e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f`.
- Role: `/home/wiktor/.codex/agents/developer.toml`.
- Dependencies inspected: task `250` and task `251` are both `PASSED`; their preparation records establish manual global-item CRUD, the audited transaction boundary, `AdminMutationResult.AfterCommit`, the shared Redis generation, generation-versioned Catalog/Substitution responses, and guarded stale writes.
- Task-list status was not edited.
- Concurrent worktree changes, including the specifically identified OpenAPI, frontend admin workflow/client/generated-type, and generator paths, were preserved. Additional concurrent backend/frontend/script edits appeared while verification ran and were also left untouched.

## Outcome

Manual global-item create, update, and soft-delete now register the existing Redis-backed shared generation invalidator through `AdminMutationResult.AfterCommit`. A committed non-replay create, committed update, and committed delete each invoke it once. Create replay, request validation failure, conflict, transaction failure, and audit persistence failure invoke it zero times.

`newProduction` supplies `cache.NewClassificationInvalidator(nil, redisClient)` to the manual-item admin controller. This is the same generation mechanism already consumed by generation-versioned Catalog Search responses and Substitution Search response/similarity caches. The generation advances before best-effort old search-key deletion, so peer instances immediately address the new generation and stale in-flight writes fail the existing Redis compare-and-set.

## Exact changed paths

| Path | Change |
| --- | --- |
| `backend/internal/httpapi/manual_item_controller.go` | Added the manual-item invalidator boundary, optional constructor composition, non-replay create callback, update/delete callbacks, and the nil-safe post-commit invalidation method. |
| `backend/internal/httpapi/manual_item_controller_test.go` | Added exactly-once, replay-zero, failure-zero, and callback-order assertions at the audited HTTP boundary. |
| `backend/internal/app/app.go` | Composed manual-item administration with the shared Redis generation invalidator. |
| `backend/internal/app/app_test.go` | Added focused AST-backed production-composition verification. |
| `backend/internal/cache/manual_item_generation_integration_test.go` | Added live Redis, two-instance Catalog/Substitution refresh and stale-write rejection coverage for create/update/delete. |
| `docs/implementation/preparations/task-264.md` | Added this preparation and verification record. |

No SQL, migration, OpenAPI, generated client, frontend, task-list, or dependency implementation path was changed for task 264.

## Added and modified executable symbols

### `backend/internal/httpapi/manual_item_controller.go`

- Added `ManualItemCacheInvalidator`.
- Modified `ManualItemController` to retain the invalidator.
- Modified `NewManualItemAdminController`.
- Modified `(*ManualItemController).Create`.
- Modified `(*ManualItemController).Update`.
- Modified `(*ManualItemController).Delete`.
- Added `(*ManualItemController).invalidate`.

### `backend/internal/httpapi/manual_item_controller_test.go`

- Modified `fakeManualItemService`.
- Modified `(*fakeManualItemService).Update`.
- Modified `(*fakeManualItemService).Delete`.
- Added `manualItemInvalidatorStub`.
- Added `(*manualItemInvalidatorStub).Invalidate`.
- Added `manualItemAuditObserver`.
- Added `(*manualItemAuditObserver).WithMutationAudit`.
- Modified `TestManualItemAdminHTTPValidCRUDReplayAndAuditSnapshots`.
- Modified `TestManualItemAdminHTTPRejectsConflictsDuplicatesInvalidFieldsAndOwnership`.
- Added `TestManualItemInvalidationRunsOnlyAfterSuccessfulAuditCommit`.

### `backend/internal/app/app.go`

- Modified `newProduction`.

### `backend/internal/app/app_test.go`

- Added `TestNewProductionComposesManualItemSharedGenerationInvalidation`.
- Added `selectorName`.

### `backend/internal/cache/manual_item_generation_integration_test.go`

- Added `manualItemSearchRepository`.
- Added `(*manualItemSearchRepository).GetByID`.
- Added `(*manualItemSearchRepository).Search`.
- Added `(*manualItemSearchRepository).replace`.
- Added `TestManualItemGenerationLiveRedisRefreshesPeerCatalogAndSubstitutionSearch`.
- Added `searchRunner`.
- Added `assertSearchItemName`.

## Verification commands and results

| Command | Result |
| --- | --- |
| `gofmt -w backend/internal/httpapi/manual_item_controller.go backend/internal/httpapi/manual_item_controller_test.go backend/internal/cache/manual_item_generation_integration_test.go backend/internal/app/app_test.go backend/internal/app/app.go` | PASS. Task-owned Go paths formatted. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -run 'TestManualItem(AdminHTTPValidCRUDReplayAndAuditSnapshots\|AdminHTTPRejectsConflictsDuplicatesInvalidFieldsAndOwnership\|InvalidationRunsOnlyAfterSuccessfulAuditCommit)$' -count=1` plus equivalent focused `internal/app` and `internal/cache` commands | PASS: `internal/httpapi`, `internal/app`, and `internal/cache`. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/httpapi -run 'TestManualItem(AdminHTTPValidCRUDReplayAndAuditSnapshots\|AdminHTTPRejectsConflictsDuplicatesInvalidFieldsAndOwnership\|InvalidationRunsOnlyAfterSuccessfulAuditCommit)$' -count=1` | PASS, final rerun `1.118s`. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/app -run '^TestNewProductionComposesManualItemSharedGenerationInvalidation$' -count=1` | PASS, final rerun `1.034s`. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/cache -run '^TestManualItemGenerationLiveRedisRefreshesPeerCatalogAndSubstitutionSearch$' -count=1` | PASS, ran rather than skipped against live Redis, final rerun `1.032s`. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi ./internal/app ./internal/cache` | PASS: `internal/httpapi` `2.653s`, `internal/app` `47.305s`, `internal/cache` `1.031s`. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./internal/httpapi ./internal/app ./internal/cache` | PASS. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS: 275 sequential tasks with ordered dependencies. |
| `git diff --check` | PASS. |
| `python3 scripts/check.py --quick` | PARTIAL/UNRELATED FAILURE. Static checks, format, Go Doc/TSDoc, OpenAPI lint, generated drift, vulnerability scan, all changed backend package tests, frontend type/unit tests (`528` pass), and desktop executions passed. The concurrently added/modified frontend Playwright lane failed when its preview server stopped accepting connections before all mobile executions (`net::ERR_CONNECTION_REFUSED` on `localhost:4173`). No task-264 path is frontend or Playwright-owned. |

Backend test commands used repository-local `GOCACHE` and `GOMODCACHE`. The live Redis test used `MEALSWAPP_REDIS_URL` when set, otherwise `redis://localhost:6379/13`, and passed against the available local service.

## Criteria evidence

| Criterion | Evidence |
| --- | --- |
| Invalidation only after audited commit | `Create`, `Update`, and `Delete` return `AfterCommit`; `AdminController.transactionalMutation` executes that callback only after `WithMutationAudit` returns successfully. `manualItemAuditObserver` asserts zero invalidations while the transaction callback is active. |
| Exactly one generation advancement per committed mutation | `TestManualItemAdminHTTPValidCRUDReplayAndAuditSnapshots` observes counts `1`, `2`, and `3` after committed create, update, and delete. The production invalidator performs one `ClassificationGeneration.Advance` per `Invalidate`. |
| Replay advances zero | The create callback checks `!result.Replayed`; the replay assertion leaves the count at `1` and produces no second audit commit. |
| Validation/conflict/transaction/audit failure advances zero | Invalid request bodies, idempotency conflict, duplicate-name conflict, update conflict, simulated transaction commit failure, and simulated `ErrAdminAuditPersistence` all retain an invalidation count of zero. |
| Production app uses shared Redis generation | `newProduction` passes `cache.NewClassificationInvalidator(nil, redisClient)` into `NewManualItemAdminController`; the focused app AST test locks this composition. Search response and similarity stores in the same constructor use `classificationGeneration` from the same Redis client. |
| Cross-instance cache refresh | The live Redis integration test prewarms Catalog and Substitution Search through one service composition, changes a shared repository, invalidates, and reads fresh create/update/delete state through a second composition sharing Redis. |
| No stale in-flight repopulation | The live Redis test captures old response and similarity tokens before create, advances generation, and proves both guarded writes return `stored=false`. |
| Race safety | All focused controller, app-composition, and live Redis tests pass under `go test -race`. Test repositories and counters use `sync.RWMutex` and `atomic.Int32`. |

## Risks and residual notes

- `ClassificationInvalidator.Invalidate` is intentionally best-effort and has no error return. Redis failure after a database/audit commit cannot roll back the committed mutation or change its response; this is existing task-251 behavior. When generation advancement succeeds, old generations become unreachable before key cleanup. When Redis is unavailable, search already follows its documented cache-degraded path.
- The existing implementation names the shared Redis primitive `ClassificationGeneration` and its key `classification:cache-generation:v1`, although it now coordinates classification, curated-import, and manual food-data mutations. Renaming or migrating that key was outside task 264 and would add compatibility risk without changing behavior.
- The app-composition test intentionally verifies constructor syntax through Go AST because `newProduction` owns concrete production collaborators. Runtime behavior is separately exercised through the controller and live Redis integration tests.
- The aggregate quick-check failure is isolated to concurrent frontend Playwright changes and preview-server availability; all task-264-focused and complete affected backend package checks pass.
