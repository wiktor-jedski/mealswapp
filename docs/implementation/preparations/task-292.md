# Task 292 Preparation

- Task: 292, Phase 08.02 Searchable Global Item Picker
- Status: `PREPARED`
- Preparation commit: `d0dcf521` (`repair legacy picker search evidence`)
- Preparation refresh: 2026-07-30
- Worktree: `/home/wiktor/Work/worktrees/mealswapp/292`
- Task-list status rows other than Task 292 were not changed.

## Scope and dependencies

Task 292 implements the administrator-only paginated global manual-item picker and its search, authoritative loading, mutation refresh, and privacy boundaries. Dependencies 250, 253, 256, 257, 280, and 283 are `PASSED` in `docs/implementation/02_TASK_LIST.md`.

## Changed task surface

| File | Symbols or executable surface | Evidence |
| --- | --- | --- |
| `backend/internal/repository/sql/manual_food_search.sql` | Legacy-compatible canonical search predicate and deterministic ordering | Repeated-whitespace legacy PostgreSQL regression |
| `backend/internal/repository/sql/manual_food_search_count.sql` | Legacy-compatible canonical count predicate | Search/count parity inspection and repository regression |
| `backend/internal/repository/manual_food_repository_test.go` | `TestPostgresManualFoodItemCRUD` | Direct PostgreSQL create/search/update and legacy-row discovery |
| `frontend/tests/task283-manual-catalog.spec.ts` | Private-partition assertions in managed acceptance flow | Task 283 isolated real-stack evidence |
| `scripts/run-task283-acceptance.py` | Run-owned private fixture seeding and fixture environment export | Managed acceptance harness |

Earlier committed Task 292 implementation and repair commits remain preserved:

- `685ca9f8` — add global item picker
- `01995b97` — repair global item picker regressions
- `e24b285b` — align picker macros contract
- `d0dcf521` — repair legacy picker search evidence

## Verification evidence

| Command | Result |
| --- | --- |
| `python3 scripts/validate-task-list.py` | PASS; 302 sequential tasks and ordered dependencies |
| `python3 scripts/validate-traceability.py` | PASS |
| `python3 scripts/generate-api-types.py --check` | PASS; generated API types current |
| `git diff --check` | PASS |
| `cd backend && ... go test ./...` | PASS; all backend packages |
| `cd backend && ... go vet ./...` | PASS |
| `cd backend && ... go test ./internal/repository -run '^TestPostgresManualFoodItemCRUD$' -count=1` | PASS |
| `cd frontend && ... bun run typecheck` | PASS |
| `python3 -m unittest scripts/test_run_task283_acceptance.py` | PASS; 28 tests |
| `python3 -m py_compile scripts/run-task283-acceptance.py` | PASS |
| `npx --no-install redocly lint api/openapi.yaml` | NOT RUN; local npx resolution returned `This is not the package you're looking for` |

The managed Task 283 run produced real-stack create/update/search evidence and no admin private-item POST 403. Its remaining non-pass criteria are linked to the pre-existing `P08-FIND-283-001` global/private-owner finding, which is outside Task 292's picker/search implementation scope.

## Criterion results

- Global ownerless item search and bounded pagination: PASS by implementation and repository/API/frontend regression coverage.
- Canonical name persistence and search, including repeated internal whitespace in legacy rows: PASS by PostgreSQL regression.
- Blank-query stale response cancellation and later-page deletion recovery: PASS by focused frontend regression coverage from the implementation repair cycle.
- Macro projection contract: PASS; picker uses the canonical `MacroProfile` contract.
- Private-item exclusion and administrator authorization: PASS by backend/API and managed browser evidence; the Task 283 harness now seeds a private fixture read-only boundary instead of attempting an unauthorized admin private-item mutation.
- OpenAPI lint: unavailable in this environment; generated API drift check passes.

## Risks and blockers

The Task 283 managed acceptance report still records `P08-FIND-283-001` for the separate requirement that created manual items be private/owner-scoped; Task 292 requires global ownerless manual items and therefore cannot resolve that product-contract finding without contradicting its own scope. The OpenAPI lint executable is unavailable through the configured local `npx --no-install` path. No Task 292 implementation blocker remains.

