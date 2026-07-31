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
| `npm exec --yes --package=@redocly/cli@2.31.5 -- redocly lint api/openapi.yaml` | PASS; one pre-existing ignored OAuth callback 302 warning |
| `cd backend && ... go test ./internal/repository ./internal/httpapi ./internal/itemcurator` | PASS |
| `cd backend && ... go test -race ./internal/repository` with isolated `mealswapp_test` | PASS; repository DB/race lane completed in 43.4s |
| `cd backend && ... go test -race ./...` with isolated `mealswapp_test` | PASS; all backend packages completed |
| `cd backend && ... go vet ./...` | PASS |
| `cd backend && ... go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS for called code; scanner reports dependency vulnerabilities outside called code |
| `python3 scripts/check.py --quick` | FAIL, unrelated phase-wide acceptance/UAT fixtures: stale `docs/implementation/02_TASK_LIST.md` hash and unsynchronized `P08-SWR054-ACCEPT-01`; the Task 292 validators and changed-area checks pass |

The retained managed Task 283 evidence (`logs/phase08-acceptance/task283-38b14c5e78cc60d1a3b46df3-sw-req-056/` and the paired `sw-req-033` report) produced real-stack evidence without the admin private-item POST 403. Its non-pass criteria remain linked to `P08-FIND-283-001` (global/private-owner acceptance) and `P08-FIND-283-005` (standardized-storage/discovery acceptance). Neither finding is closed by this preparation; Task 292's picker implementation is the planned remediation surface for `P08-FIND-283-005`, while Task 302 owns the final isolated retest and closure decision.

## Criterion results

- Global ownerless item search and bounded pagination: PASS by implementation and repository/API/frontend regression coverage; managed Task 283 acceptance remains non-pass where its separate owner/private-partition criteria apply.
- Canonical name persistence and search, including repeated internal whitespace in legacy rows: PASS by PostgreSQL regression.
- Blank-query stale response cancellation and later-page deletion recovery: PASS by focused frontend regression coverage from the implementation repair cycle.
- Macro projection contract: PASS; picker uses the canonical `MacroProfile` contract.
- Private-item exclusion and administrator authorization: implementation/API boundary PASS; managed Task 283 acceptance still records `P08-FIND-283-001` for the unresolved owner/private-partition criterion. The harness seeds a private fixture read-only boundary instead of attempting an unauthorized admin private-item mutation.
- OpenAPI lint: PASS with the documented pre-existing ignored OAuth callback 302 warning; generated API drift check passes.

## Risks and blockers

The current acceptance ledger still records `P08-FIND-283-001` and `P08-FIND-283-005` as open/non-pass findings; this report does not hide or close them. The phase-wide quick gate also exposes an unrelated stale UAT hash and unsynchronized `P08-SWR054-ACCEPT-01`; those are not Task 292 evidence and remain outside this repair. Task 292 requires global ownerless manual items and therefore cannot resolve the private-owner tracker without contradicting its own scope. No additional Task 292 implementation blocker remains.
