# Task 213 Preparation Evidence

## Scope and baseline

- Assigned task: **213 — Phase 07.01 Daily Diet and Optimization Response Contract Audit**.
- Baseline commit: `a4e31367485b03269e90b5607f2057c9568bb5b1`.
- Baseline confidence: **high**. The supplied baseline was confirmed before editing, and the initial dirty paths were exactly `docs/implementation/02_TASK_LIST.md` plus untracked `review.txt`.
- Preservation: neither initial dirty path was edited by task 213, and no task status was changed.
- Concurrent-work note: while task 213 was in progress, other implementation streams added task-215/216 files and overlapping canonical-unit/idempotency hunks. Those changes were preserved. In particular, the `CanonicalQuantityUnit`, `SubstitutionUnit`, and `generated_contract` changes visible in shared-file diffs are not task-213 work.
- Scope boundary: no task after 213 was implemented. This task documents and enforces response status presence; shared runtime-safe error mapping remains task 227.

## Sources inspected

- `docs/design/DESIGN-004.md`: `JobStatusTracker`, submission admission, queue failure, polling ownership, expiry, and timeout behavior.
- `docs/design/DESIGN-008.md`: `ProfileController` and `SavedDataRepository` ownership/error responsibilities.
- `backend/internal/httpapi/router.go`: gateway context, CORS, authentication composition, validation, timeout, generic classification, and dependency classification.
- `backend/internal/httpapi/auth_middleware.go` and `backend/internal/httpapi/csrf.go`: 401 and mutation 403 paths.
- `backend/internal/httpapi/profile_controller.go` and `backend/internal/httpapi/daily_diet_controller.go`: five Daily Diet route definitions and controller mappings.
- `backend/internal/dailydiet/service.go`: validation, missing-resource, idempotency-conflict, repository/dependency, and generic failures.
- `backend/internal/httpapi/optimization_controller.go`: submission/poll route definitions, entitlement, admission 429, idempotency conflict, queue/dependency, ownership, and expiry paths.
- `frontend/src/lib/api/daily-diet-client.ts` and `frontend/src/lib/api/optimization-client.ts`: response-envelope consumption and safe fallback behavior.

## Response-documentation policy

Every audited OpenAPI operation explicitly declares every JSON HTTP status reachable after matching that operation. This includes statuses produced by route middleware (authentication, CSRF, body/path validation), controllers, application dependencies, admission/rate limiting, gateway deadlines, and generic server-error classification. There is no implicit response inheritance for audited operations.

Transport behavior that does not represent execution of an OpenAPI operation remains shared gateway behavior: HTTP-to-HTTPS redirect and CORS `OPTIONS` preflight are not repeated as operation responses. A rejected cross-origin request is a JSON 403 and is therefore included on every audited operation. The shared `components.responses.Error` envelope remains the single schema reference for all audited error statuses.

## Route/status audit matrix

| Route | Success | Validation | Auth/security | Resource/conflict | Rate limit | Generic/dependency/timeout | Result |
|---|---:|---:|---:|---:|---:|---:|---|
| `GET /api/v1/daily-diets` | 200 | — | 401, 403 | 404 | — | 500, 503, 504 | Complete |
| `POST /api/v1/daily-diets` | 201 | 400 | 401, 403 | 404, 409 | — | 500, 503, 504 | Complete |
| `GET /api/v1/daily-diets/{dietId}` | 200 | 400 | 401, 403 | 404 | — | 500, 503, 504 | Complete |
| `PUT /api/v1/daily-diets/{dietId}` | 200 | 400 | 401, 403 | 404, 409 | — | 500, 503, 504 | Complete |
| `DELETE /api/v1/daily-diets/{dietId}` | 204 | 400 | 401, 403 | 404 | — | 500, 503, 504 | Complete |
| `POST /api/v1/optimization/jobs` | 202 | 400 | 401, 403 | 404, 409 | 429 | 500, 503, 504 | Complete |
| `GET /api/v1/optimization/jobs/{jobId}` | 200 | 400 | 401, 403 | 404, 410 | — | 500, 503, 504 | Complete |

Status-source details:

- 400: body/header validation, UUID path validation, and validation-classified service/repository failures.
- 401: required authentication middleware (plus defensive controller checks).
- 403: shared CORS rejection; additionally CSRF on mutations and entitlement denial on optimization submission.
- 404: missing meal/diet/job resources, including a collection projection whose referenced meal was soft-deleted, and ownership-safe optimization polling.
- 409: create idempotency conflict, replacement persistence conflict, and optimization idempotency/admission conflict.
- 410: owned optimization result expired.
- 429: optimization active-job and fixed-hour admission decisions. No audited Daily Diet route has a route limiter or domain admission limiter, so 429 is not declared there.
- 500: panic/unclassified generic failure handling.
- 503: unavailable Daily Diet repository/meal dependencies, optimization dependencies, admission/store, and queue.
- 504: shared gateway deadline handling.
- The former optimization-submission 422 declaration was removed because no submission middleware, controller, or dependency path produces it; solver infeasibility is a polled job state rather than a submission response.

## Exact task-213 changed paths

- `api/openapi.yaml`
  - Added the audited explicit response statuses to all seven Daily Diet/optimization operations.
  - Removed stale optimization submission 422.
- `scripts/generate-api-types.py`
  - Added an exact audited operation/status matrix and made `--check` fail before generation when an audited operation key is missing/extra or any exact status, `default`, or valid `1XX`/`2XX`/`3XX`/`4XX`/`5XX` wildcard key is missing/extra.
- `scripts/test_generate_api_types.py`
  - Added focused current-contract, collection-list 404, deliberate exact-status mismatch, extra-audited-operation, and wildcard-response mismatch tests.
- `frontend/src/lib/api/daily-diet-client.test.ts`
  - Added focused 500/503/504 envelope error coverage.
- `frontend/src/lib/api/optimization-client.test.ts`
  - Added focused 429/500/503/504 envelope error coverage.
- `docs/implementation/preparation/task-213-preparation.md`
  - Added this policy, matrix, traceability, and verification record.
- `backend/internal/dailydiet/service_test.go` and `backend/internal/httpapi/daily_diet_controller_test.go`
  - Added focused regressions for unavailable-meal list projection and its stable HTTP 404 mapping; production Go was not changed for task 213.

## Added or modified executable symbols

### `scripts/generate-api-types.py`

- Added `REQUIRED_OPERATION_RESPONSES`.
- Added `AUDITED_OPERATION_PREFIXES` and `HTTP_METHODS`.
- Added `audited_operation_keys(source)`.
- Added `operation_response_statuses(source, path, method)`; repaired its response-key recognizer to include OpenAPI wildcard keys so the exact policy rejects them as extras.
- Added `operation_response_mismatches(source)` and repaired it to compare the complete discovered audited-operation key set before exact status sets.
- Modified `main()` to reject response matrix drift before checking generated output.

### `scripts/test_generate_api_types.py`

- Added `OperationResponseDriftTest`.
- Added `OperationResponseDriftTest.test_current_contract_matches_audited_response_matrix()`.
- Added `OperationResponseDriftTest.test_deliberate_response_mismatch_is_rejected()`.
- Added `OperationResponseDriftTest.test_collection_list_404_is_audited()`.
- Added `OperationResponseDriftTest.test_deliberate_extra_audited_operation_is_rejected()`.
- Added `OperationResponseDriftTest.test_wildcard_response_keys_are_rejected_by_exact_policy()`.

### Focused backend regressions

- Added `TestServiceListReturnsNotFoundWhenSavedMealIsUnavailable()`.
- Added `TestProfileControllerDailyDietListMapsUnavailableMealToNotFound()`.

### Frontend test registrations

- Added `shared 500, 503, and 504 failures remain bounded client errors`.
- Added `submission maps audited 429, 500, 503, and 504 envelopes to bounded errors`.

No production Go or TypeScript executable symbol was added or modified by task 213.

## Commands and results

| Command | Result |
|---|---|
| `python3 -m unittest scripts/test_generate_api_types.py` (repository root) | PASS — 5 tests. The tests lock collection-list 404, reject a removed optimization 429, reject a deliberate extra audited PATCH operation, and reject each valid wildcard response key. |
| `python3 scripts/generate-api-types.py --check` | PASS — generated API types current and response matrix current. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS (exit 0) — API valid; one existing ignored warning for OAuth callback lacking a 2XX response. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check:api-types` | PASS. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/daily-diet-client.test.ts src/lib/api/optimization-client.test.ts` | PASS — 13 tests, 42 expectations. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -run 'Test(ProfileControllerDailyDiet\|OptimizationHTTP)' -count=1` | PASS, including the collection-list 404 controller regression. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/dailydiet -run 'TestServiceListReturnsNotFoundWhenSavedMealIsUnavailable' -count=1` | PASS. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck` | PASS. |
| `python3 -m py_compile scripts/generate-api-types.py scripts/test_generate_api_types.py` | PASS. |
| `git diff --check -- api/openapi.yaml scripts/generate-api-types.py scripts/test_generate_api_types.py frontend/src/lib/api/daily-diet-client.test.ts frontend/src/lib/api/optimization-client.test.ts` | PASS. |
| Focused Python status extraction using `REQUIRED_OPERATION_RESPONSES` and `operation_response_statuses` | PASS — printed the exact seven status sets recorded in the matrix. |
| `python3 scripts/validate-traceability.py` | PASS in the final shared-worktree state. |
| `git diff --check -- docs/implementation/preparation/task-213-preparation.md` | PASS. |

Two grouped command attempts were initially launched with `frontend/` or `backend/` as the working directory while using root-relative Python/OpenAPI paths. Those root-relative subcommands failed with file-not-found errors; the frontend tests, focused Go tests, and frontend typecheck in the same attempts passed. Every mistaken invocation was rerun from its correct directory with the passing results above.

## Verification criteria

| Criterion | Satisfied | Evidence |
|---|---|---|
| Route/status matrix accounts for every reachable success and error response | Yes | Policy and seven-row matrix above, derived from route composition, controllers, dependencies, admission, and gateway classification. |
| OpenAPI contains required responses, including applicable 429/500/503/504 | Yes | Exact status sets in `api/openapi.yaml`; response drift test confirms equality. |
| Response-aware generation or focused drift test fails on deliberate mismatch | Yes | `check:api-types` invokes `operation_response_mismatches`; focused tests reject removed POST optimization 429, an extra audited PATCH operation, and wildcard response keys. |
| Redocly lint passes | Yes | Exit 0; only the repository's existing explicitly ignored OAuth callback warning remains. |
| `check:api-types` passes | Yes | Generated types and audited response matrix current. |
| Focused API-client error tests pass | Yes | 13/13 Daily Diet and optimization client tests pass. |

All assigned task-213 verification criteria are satisfied.

## Three important review findings repaired

### F-213-1 — collection-list 404 policy

- Chosen policy: retain explicit per-operation status documentation. `GET /api/v1/daily-diets` now declares 404 in `api/openapi.yaml`, and `REQUIRED_OPERATION_RESPONSES[("/api/v1/daily-diets", "get")]` requires it.
- Runtime evidence: `TestServiceListReturnsNotFoundWhenSavedMealIsUnavailable()` stores a diet entry whose meal cannot be projected and proves `Service.List()` preserves `repository.ErrorKindNotFound`. `TestProfileControllerDailyDietListMapsUnavailableMealToNotFound()` proves `ListDailyDiets()` returns status 404 with stable code `not_found`.
- Contract evidence: `OperationResponseDriftTest.test_collection_list_404_is_audited()` independently asserts 404 in both the policy matrix and extracted OpenAPI responses.

### F-213-2 — complete audited operation keys

- Root cause: `operation_response_mismatches()` iterated only hard-coded keys, so an added operation was invisible.
- Repair symbols: `AUDITED_OPERATION_PREFIXES`, `HTTP_METHODS`, and `audited_operation_keys()` discover every HTTP operation under the Daily Diet and optimization-job audited path families. `operation_response_mismatches()` compares discovered and required operation-key sets, reports missing/extra operations, then performs exact status-set comparison for every common key.
- Regression evidence: `OperationResponseDriftTest.test_deliberate_extra_audited_operation_is_rejected()` inserts `PATCH /api/v1/daily-diets` with response 418 and requires the exact diagnostic `unexpected audited operation: PATCH /api/v1/daily-diets`.
- Current direct audit discovered exactly seven operation keys and returned `mismatches: []`; the extracted collection-list set is `['200', '401', '403', '404', '500', '503', '504']`.

### F-213-3 — shared Error response policy

- The enumerated shared description was removed. `components.responses.Error.description` now states only that it is the user-safe gateway error envelope reused by operation-level responses; exact statuses remain authoritative at each operation.
- Because the shared component no longer enumerates statuses, no duplicate status-list assertion is needed. Exact operation statuses remain enforced by `operation_response_mismatches()` and its focused tests.

### Final reviewer-requested verification

| Command | Exact result |
|---|---|
| `python3 -m unittest scripts/test_generate_api_types.py` | PASS — 5 tests. |
| `python3 scripts/generate-api-types.py --check` | PASS — generated API types current. |
| `python3 -m py_compile scripts/generate-api-types.py scripts/test_generate_api_types.py` | PASS. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS — API valid; one pre-existing explicitly ignored OAuth callback warning. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check:api-types` | PASS. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/daily-diet-client.test.ts src/lib/api/optimization-client.test.ts` | PASS — 13 tests, 42 expectations. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage src/lib/api/daily-diet-client.test.ts src/lib/api/optimization-client.test.ts` | PASS — 13 tests; aggregate 60.24% lines, `daily-diet-client.ts` 82.93%, `optimization-client.ts` 90.59%. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck` | PASS. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -run 'Test(ProfileControllerDailyDiet\|OptimizationHTTP)' -count=1` | PASS. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/dailydiet -run 'TestServiceListReturnsNotFoundWhenSavedMealIsUnavailable' -count=1` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS — 237 sequential tasks with ordered dependencies. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-213-review.md` | PASS — structurally valid. |
| `git diff --check -- api/openapi.yaml scripts/generate-api-types.py scripts/test_generate_api_types.py backend/internal/dailydiet/service_test.go backend/internal/httpapi/daily_diet_controller_test.go frontend/src/lib/api/daily-diet-client.test.ts frontend/src/lib/api/optimization-client.test.ts` | PASS. |

The first focused-check attempt ran from `backend/` while retaining root-relative `backend/...` paths and stopped at `gofmt` with `lstat ... no such file or directory`; it executed no tests and changed no files. The command was immediately rerun from the repository root, and all focused checks passed.

### Final symbols and SHA-256 fingerprints

| File/symbol surface | SHA-256 |
|---|---|
| `api/openapi.yaml` — collection GET response map and shared `Error` description | `6368ee9c1321104d0e645ed8a3e6b73f8f14c1a161835a5f388a0e6e2fa3da4a` |
| `scripts/generate-api-types.py` — matrix, audited-operation discovery, status extraction/mismatch, `main()` enforcement | `b6b241afb9c13c4206e30eb1e12ae4adcd32d6734cc065b33ece5d47ed4f4e87` |
| `scripts/test_generate_api_types.py` — five focused contract tests | `475c5f9fe28e4bba538ced96faa40b07223708cd37ae2a227f7049ff900ad4d5` |
| `backend/internal/dailydiet/service_test.go` — unavailable-meal list regression | `8974b57ca08281025d247b92a1f81edf6d922aee1b5684653273e2e5f9389907` |
| `backend/internal/httpapi/daily_diet_controller_test.go` — list 404 mapping regression | `34485172a7e6ada598863b6ff01e7c3e8ffcc290bc44961866e629bbcb40d777` |
| `frontend/src/lib/api/daily-diet-client.test.ts` — inspected unchanged task-213 client test | `bb3e33b04093e8650e3843556de3b0d6a418c2ff3d9e86b14dc23d12aca19c90` |
| `frontend/src/lib/api/optimization-client.test.ts` — inspected unchanged task-213 client test | `8935ee1e3e571d91648bd582da96fc5fdbfc0cb02f0f426412a004dcace1e5f8` |
| `frontend/src/lib/api/generated.ts` — concurrent overlap boundary, unchanged by this repair | `361ce14d3cde8ae90afe0bc074ffb3e301c751a8a9603fc7a300e7d6b49cf20b` |

Preservation evidence: task 213 remains `PREPARED` at `docs/implementation/02_TASK_LIST.md:220`. This repair did not edit any task status or any production Go/TypeScript symbol, and unrelated concurrent task-214/215/216/217 work remains intact.
