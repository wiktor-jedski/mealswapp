# Task 298 Repair Evidence

Date: 2026-07-31

## Repair scope

This repair addresses the actionable Task 298 review findings F-298-001,
F-298-002, and F-298-003. It does not modify the task list or unrelated task
implementation. The migration now materializes payload-free retry markers for
legacy soft-deleted custom items before removing their completed create
claims. The private-data UI now discloses permanent irreversible deletion and
retains bounded saved-diet references with concrete removal guidance after a
409 conflict.

## Evidence map

| Requirement | Direct evidence |
|---|---|
| Legacy migration creates durable retry markers and removes legacy rows/claims | `backend/internal/repository/task298_permanent_custom_food_deletion_integration_test.go`, `TestTask298MigrationMaterializesLegacyRetryMarker` |
| PostgreSQL conflict is owner-scoped, bounded, and leaves the item/reference unchanged | `backend/internal/app/task298_permanent_custom_food_deletion_integration_test.go`, `TestTask298PermanentCustomFoodDeletionPostgresAndExport` |
| Hard deletion, classification cascade, repeated DELETE, cross-owner non-disclosure, and same-name recreation | `TestTask298PermanentCustomFoodDeletionPostgresAndExport` |
| Marker is payload-free, blocks exact delayed replay, expires, purges, and follows account erasure | `TestTask298PermanentCustomFoodDeletionPostgresAndExport` and `TestTask298MigrationMaterializesLegacyRetryMarker` |
| JSON and CSV exports contain the item before deletion and cannot recover it afterward | `TestTask298PermanentCustomFoodDeletionPostgresAndExport` |
| Account-export parsing accepts saved-diet references to custom food items | `frontend/src/lib/api/account-data-client.test.ts` |
| Explicit irreversible confirmation, cancellation, conflict guidance, and fail-closed recovery | `frontend/tests/admin-private-data.spec.ts`, `requires irreversible confirmation and retains actionable saved-diet references after conflict` |

The Playwright test runs in both configured desktop and mobile projects. The
bounded diet names are rendered as escaped text in an accessible list, and the
guidance directs the user to Daily Diets to remove each reference, save the
diets, and retry deletion.

## Validation

| Command | Result |
|---|---|
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/repository ./internal/app -count=1` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/repository ./internal/app -run 'TestTask298' -count=1` | PASS |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | PASS — 568 tests, 0 failures |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck` | PASS |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | PASS |
| `cd frontend && MEALSWAPP_PLAYWRIGHT_REUSE_BUILD=1 BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/admin-private-data.spec.ts` | PASS — 6 tests |
| `python3 scripts/check.py --quick` | PASS |
| `python3 scripts/validate-task-list.py` | PASS — 302 sequential tasks; task list unchanged |
| `python3 scripts/validate-traceability.py` | PASS |
| `python3 scripts/generate-api-types.py --check` | PASS |
| `npm exec --yes --package=@redocly/cli@2.31.5 -- redocly lint api/openapi.yaml` | PASS with pre-existing ignored OAuth 302-only warning |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS |

## Remaining blockers

The following review findings are external blockers and were intentionally
not changed in this Task 298 repair:

- Task 297 remains `OPEN` in `docs/implementation/02_TASK_LIST.md`.
- Task 284 is not a fresh passing predecessor: its committed acceptance
  contains failures for `SW-REQ-043` and `SW-REQ-072`, while the fresh runner
  fails closed because `P08-SWR043-ACCEPT-01`, `P08-SWR072-STEP-01`, and
  `P08-SWR073-STEP-01` lack synchronized findings.
- `python3 scripts/check.py` remains red for the documented baseline lanes:
  the frontend coverage exception is stale/over-broad for
  `src/lib/api/account-data-client.ts`, the browser lane has 30 API-proxy
  failures with `ECONNREFUSED 127.0.0.1:8080`, and the backend race lane
  retains unrelated data-importer baseline failures. No unrelated baseline
  or task-list files were changed.

These blockers prevent claiming a green aggregate release gate; the focused
Task 298 repair evidence above is passing.
