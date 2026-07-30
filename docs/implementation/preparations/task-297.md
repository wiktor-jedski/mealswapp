# Task 297 Preparation Evidence — Saved Diet Custom Food Entries Repair

## Outcome and control

- Task: **297 — Saved Diet Custom Food Entries** (`DESIGN-008: SavedDataRepository`).
- Existing implementation was preserved. This repair adds the missing browser owner-isolation scenario and refreshes the machine-checked coverage contracts.
- `docs/implementation/02_TASK_LIST.md` was read only; its Task 297 status row was not edited.
- Worktree: `task-297-saved-diet-custom-food-entries`.

## Task-owned repair paths

| Path | Repair surface |
|---|---|
| `frontend/tests/daily-diet-workflow.spec.ts` | Added two authenticated browser pages with disjoint custom-food responses; the Playwright project context runs this test as Desktop Chrome and Pixel 5 mobile. It asserts each owner’s picker contains only its own private item. Existing loading, empty, retry, duplicate-name, edit, and reload scenarios remain covered. |
| `scripts/check.py` | Added `src/lib/api/custom-item-client.ts` to the Phase 08 frontend coverage source set. |
| `docs/implementation/04_OPEN.md` | Refreshed exact backend/frontend measured coverage rows and summaries after Task 297 sources changed. |
| `docs/implementation/preparations/task-297.md` | Added this scoped preparation and verification ledger. |

## Coverage contract refresh

- Backend command: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/... -p 1 -count=1 -coverpkg=./internal/... -coverprofile=phase08-coverage.out`.
- Result: PASS; Phase 08 contract scope is `4,698/5,042` statements (`93.2%`). Exact changed rows include `custom_item_controller.go` `110/139`, `custom_food_repository.go` `133/152`, `profile_controller.go` `34/36`, `food_repository.go` `192/193`, and `rate_limit.go` `186/188`.
- Frontend command: `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage`.
- Result: PASS; `543` tests, `2,841` expectations, `95.10%` functions, `95.93%` lines. New contract row: `src/lib/api/custom-item-client.ts` `92.31%` functions / `91.67%` lines, uncovered `73-75,94-95,109-110`, justified as `F4` transport/decoder fallback coverage. Generated client coverage is `100.00%` functions / `99.07%` lines.

## Behavioral verification

| Command | Result |
|---|---|
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/daily-diet-workflow.spec.ts` | PASS: `20` desktop/mobile tests. Includes empty-state resolution, retry recovery, and two-owner custom-food isolation. |
| `python3 scripts/generate-api-types.py --check` | PASS: generated API types are current. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS: 302 sequential tasks; Task 297 row unchanged. |
| `git diff --check` | PASS. |
| `go test -race ./internal/dailydiet ./internal/repository ./internal/httpapi ./internal/app` | PASS, previously run for the preserved implementation. |
| `go vet ./...` and `govulncheck@v1.3.0 ./...` | PASS; no called vulnerabilities. |

## Aggregate disposition

The changed-area quick lane passed the Task 297 Playwright suite, Go vet, vulnerability scan, generated drift, OpenAPI, traceability, and task-list checks. The aggregate Phase 08 acceptance-contract lane remains blocked by pre-existing Task 280/281 evidence synchronization failures and the stale historical task-list hash in the committed Phase 08 UAT artifact; no Task 297 implementation or task-list status was changed to mask those failures.

## Residual control

The worktree remains isolated and clean except for the committed Task 297 repair. No implementation behavior was removed or weakened, no task status was changed, and no coverage exception was broadened without an exact measured row.
