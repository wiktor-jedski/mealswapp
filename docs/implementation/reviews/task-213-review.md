# Review Evidence: Task 213 — Daily Diet and Optimization Response Contract Audit

task_id: 213
component: "DESIGN-004: JobStatusTracker"
static_aspect: "Daily Diet and Optimization Response Contract Audit"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-14T21:43:35Z"
review_agent: "Codex independent re-review"
evidence_file: "docs/implementation/reviews/task-213-review.md"
baseline_ref: "a4e31367485b03269e90b5607f2057c9568bb5b1"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "code-review-skill/reference/python.md; code-review-skill/reference/typescript.md; /home/wiktor/.codex/.tmp/plugins/plugins/zoom/skills/rest-api/references/openapi.md; repository OpenAPI guidance in docs/implementation/05_DEVELOPMENT.md"
repair_context_required: false

## 1. Task Source

Description: Phase 07.01: define one endpoint-versus-shared-middleware response-documentation policy, audit every Daily Diet and optimization route against controller, middleware, dependency, rate-limit, timeout, and generic failure paths, update OpenAPI responses and the generated-type drift enforcement needed to keep the matrix current.

Depends On: 212

Testing Coverage Exceptions: None.

Verification Criteria: A route/status matrix accounts for every reachable success and error response; api/openapi.yaml contains the responses required by the chosen policy, including applicable 429/500/503/504 responses; response-aware generation or a focused drift test fails on a deliberate mismatch; npx --no-install redocly lint api/openapi.yaml and cd frontend with the repository Bun environment, bun run check:api-types pass; focused API-client error tests pass.

## 2. Pre-Review Gates

- [x] Input status is PREPARED; task 213 remains PREPARED in docs/implementation/02_TASK_LIST.md:220.
- [x] Dependency 212 is PASSED.
- [x] The preparation report claims completion and records the three repaired findings.
- [x] A trustworthy task-specific baseline and scope boundary are available.
- [x] code-review-skill was invoked exactly once; the Python and TypeScript guides were read completely.
- [x] The available OpenAPI guide at /home/wiktor/.codex/.tmp/plugins/plugins/zoom/skills/rest-api/references/openapi.md and repository OpenAPI guidance were read; Redocly was used for structural validation.
- [x] The reviewer is independent from implementation and repair.
- [x] Review uses current repository state, not stale command logs.
- [x] No production code or task-list cell was edited during this review.

pre_review_gates_passed: true
blocking_issue: "NONE"

## 3. Review Baseline and Change Surface

Baseline/reference method: The fixed baseline commit is a4e31367485b03269e90b5607f2057c9568bb5b1, confirmed as HEAD. The preparation report recorded that the initial dirty paths were only the task-list edit and review.txt; it also recorded later concurrent task-214/215/216/217 changes and their overlap boundaries. I reconstructed the baseline-to-current diff, read every task-owned hunk line-by-line, and used the preparation manifest only to distinguish concurrent overlapping changes.

Commands used to reconstruct the diff:

    git status --short
    git rev-parse HEAD
    git diff --name-status a4e31367485b03269e90b5607f2057c9568bb5b1
    git diff --unified=0 a4e31367485b03269e90b5607f2057c9568bb5b1 -- api/openapi.yaml scripts/generate-api-types.py scripts/test_generate_api_types.py backend/internal/dailydiet/service_test.go backend/internal/httpapi/daily_diet_controller_test.go frontend/src/lib/api/daily-diet-client.test.ts frontend/src/lib/api/optimization-client.test.ts
    rg -n 'TestServiceListReturnsNotFoundWhenSavedMealIsUnavailable|TestProfileControllerDailyDietListMapsUnavailableMealToNotFound|class OperationResponseDriftTest|def test_|shared 500|submission maps audited' scripts/test_generate_api_types.py backend/internal/dailydiet/service_test.go backend/internal/httpapi/daily_diet_controller_test.go frontend/src/lib/api/daily-diet-client.test.ts frontend/src/lib/api/optimization-client.test.ts

Pre-existing dirty-worktree changes and exclusions:

The shared worktree contains concurrent task-214/215/216/217 implementation changes, including changes in backend/internal/dailydiet/service.go, backend/internal/httpapi/daily_diet_controller.go, repository files, migrations, frontend/src/lib/api/generated.ts, and the later canonical-unit/idempotency portions of shared files. Those changes were preserved and are not attributed to task 213. The task-owned scope is:

- api/openapi.yaml: the seven Daily Diet and optimization operation response maps. The concurrent canonical-unit schema hunk is excluded.
- scripts/generate-api-types.py: the response matrix, operation discovery, response extraction, mismatch enforcement, wildcard rejection, and the response-drift portion of main(). The concurrent generated_contract() and quantity-type hunk is excluded.
- scripts/test_generate_api_types.py: the untracked focused response-contract test module.
- backend/internal/dailydiet/service_test.go and backend/internal/httpapi/daily_diet_controller_test.go: only the unavailable-meal list regressions; unrelated concurrent test changes are excluded.
- The two frontend client test callbacks added for the audited error statuses.

The current runtime callers, SQL, migrations, frontend clients, generated types, package script, and design sources were inspected as context and hashed below. No task-owned change could not be distinguished reliably; task-owned confidence is high.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| api/openapi.yaml | Baseline diff plus preparation scope; concurrent canonical-unit hunk excluded | HIGH | Seven audited operation response maps and shared Error description |
| scripts/generate-api-types.py | Baseline diff plus preparation scope; concurrent generated-contract hunk excluded | HIGH | Matrix, discovery constants/function, status extraction, mismatch function, CLI response-drift path |
| scripts/test_generate_api_types.py | Preparation-listed untracked file and current source inspection | HIGH | Test class and five response-contract tests |
| backend/internal/dailydiet/service_test.go | Preparation-listed regression hunk; concurrent idempotency changes excluded | HIGH | Unavailable-meal list regression |
| backend/internal/httpapi/daily_diet_controller_test.go | Preparation-listed regression hunk; concurrent unit-validation changes excluded | HIGH | Collection-list 404 controller regression |
| frontend/src/lib/api/daily-diet-client.test.ts | Baseline diff | HIGH | Added 500/503/504 error test |
| frontend/src/lib/api/optimization-client.test.ts | Baseline diff | HIGH | Added 429/500/503/504 submission error test |

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | A route/status matrix accounts for every reachable success and error response. | Current route registration, gateway middleware, auth/CSRF/CORS, controllers, service/dependency mappings, SQL soft-delete behavior, and design sources. | PASS | The chosen explicit per-operation policy has exactly seven operations. The live audit accounts for success, validation, auth/security, resource/conflict, admission, dependency/generic, and timeout paths. GET /api/v1/daily-diets includes the reachable unavailable-meal 404, proven through service and controller regressions. OPTIONS preflight and TLS redirect remain shared transport behavior outside operation responses. |
| 2 | OpenAPI contains the responses required by the chosen policy, including applicable 429/500/503/504. | Exact extraction from every audited operation plus shared Error inspection. | PASS | Current extracted sets exactly equal the matrix: collection GET 200/401/403/404/500/503/504; collection POST 201/400/401/403/404/409/500/503/504; item GET 200/400/401/403/404/500/503/504; item PUT 200/400/401/403/404/409/500/503/504; item DELETE 204/400/401/403/404/500/503/504; optimization POST 202/400/401/403/404/409/429/500/503/504; poll GET 200/400/401/403/404/410/500/503/504. components.responses.Error now has a non-enumerating, operation-level-policy-consistent description. |
| 3 | Response-aware generation or a focused drift test fails on a deliberate mismatch. | Focused tests and direct mutation probes. | PASS | Five Python tests pass. The checker rejects removed optimization POST 429, an extra PATCH /api/v1/daily-diets, each valid 1XX/2XX/3XX/4XX/5XX wildcard, and a renamed/missing operation. |
| 4 | OpenAPI lint passes. | Redocly exit status and output. | PASS | npx --no-install redocly lint api/openapi.yaml exits 0. The only output is the existing explicitly ignored OAuth callback warning requiring a 2XX response. |
| 5 | Generated API type drift check passes. | Frontend command exit status and generator output. | PASS | bun run check:api-types exits 0 and the generator reports current generated API types after response-matrix validation. |
| 6 | Focused API-client error tests pass. | Focused Bun tests and coverage output. | PASS | Both client files pass: 13 tests and 42 expectations. The focused coverage run also passes, observing 82.93% lines for Daily Diet client, 90.59% for optimization client, and 60.24% aggregate across loaded files. |

## 5. Changed-Symbol Inventory

Inventory every task-owned added or modified executable/configuration unit. The OpenAPI response maps are behavioral contract units. Concurrent symbols are excluded from the task-owned count but are listed as reviewed context in Section 9.

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | REQUIRED_OPERATION_RESPONSES | Python behavioral configuration | scripts/generate-api-types.py:91-99 | Added and repaired with collection-list 404 | operation_response_mismatches, CLI, focused tests | Current, mismatch, collection, extra-operation, wildcard tests |
| 2 | AUDITED_OPERATION_PREFIXES | Python behavioral configuration | scripts/generate-api-types.py:101 | Added | audited_operation_keys | Current and extra-operation tests |
| 3 | HTTP_METHODS | Python behavioral configuration | scripts/generate-api-types.py:102 | Added | audited_operation_keys | Extra-operation and direct discovery probes |
| 4 | audited_operation_keys | Python function | scripts/generate-api-types.py:105-118 | Added | operation_response_mismatches | Current, extra-operation, missing-operation probes |
| 5 | operation_response_statuses | Python function | scripts/generate-api-types.py:121-145 | Added and modified for wildcard keys | operation_response_mismatches, direct matrix audit | Current, mismatch, collection, wildcard tests |
| 6 | operation_response_mismatches | Python function | scripts/generate-api-types.py:148-162 | Added and repaired for operation-key drift | main, focused tests | Current, mismatch, extra-operation, wildcard tests |
| 7 | Response-drift path in main | Python CLI entrypoint | scripts/generate-api-types.py:1365-1368 | Modified | Root generator and frontend package check:api-types script | Current CLI check; callee mutation probes |
| 8 | OperationResponseDriftTest | Python unittest class | scripts/test_generate_api_types.py:19 | Added | Python unittest runner | Five methods below |
| 9 | OperationResponseDriftTest.test_current_contract_matches_audited_response_matrix | Python test method | scripts/test_generate_api_types.py:20-22 | Added | Unittest runner | Self |
| 10 | OperationResponseDriftTest.test_collection_list_404_is_audited | Python test method | scripts/test_generate_api_types.py:24-28 | Added | Unittest runner | Self |
| 11 | OperationResponseDriftTest.test_deliberate_response_mismatch_is_rejected | Python test method | scripts/test_generate_api_types.py:30-40 | Added | Unittest runner | Self |
| 12 | OperationResponseDriftTest.test_deliberate_extra_audited_operation_is_rejected | Python test method | scripts/test_generate_api_types.py:42-50 | Added | Unittest runner | Self |
| 13 | OperationResponseDriftTest.test_wildcard_response_keys_are_rejected_by_exact_policy | Python test method | scripts/test_generate_api_types.py:52-63 | Added | Unittest runner | Self |
| 14 | GET /api/v1/daily-diets response map | OpenAPI behavioral contract | api/openapi.yaml:491-511 | Modified and repaired with 404 | Generator, API documentation, clients | Current matrix and service/controller regressions |
| 15 | POST /api/v1/daily-diets response map | OpenAPI behavioral contract | api/openapi.yaml:512-545 | Modified | Generator, API documentation, clients | Current matrix and client tests |
| 16 | GET /api/v1/daily-diets/{dietId} response map | OpenAPI behavioral contract | api/openapi.yaml:547-571 | Modified | Generator, API documentation, clients | Current matrix and client tests |
| 17 | PUT /api/v1/daily-diets/{dietId} response map | OpenAPI behavioral contract | api/openapi.yaml:572-603 | Modified | Generator, API documentation, clients | Current matrix and client tests |
| 18 | DELETE /api/v1/daily-diets/{dietId} response map | OpenAPI behavioral contract | api/openapi.yaml:604-626 | Modified | Generator, API documentation, clients | Current matrix and client tests |
| 19 | POST /api/v1/optimization/jobs response map | OpenAPI behavioral contract | api/openapi.yaml:628-664 | Modified; stale 422 removed | Generator, API documentation, clients | Current matrix and optimization tests |
| 20 | GET /api/v1/optimization/jobs/{jobId} response map | OpenAPI behavioral contract | api/openapi.yaml:666-693 | Modified | Generator, API documentation, clients | Current matrix and optimization tests |
| 21 | TestServiceListReturnsNotFoundWhenSavedMealIsUnavailable | Go regression test | backend/internal/dailydiet/service_test.go:405-416 | Added | Service.List projection behavior | Self; focused package test |
| 22 | TestProfileControllerDailyDietListMapsUnavailableMealToNotFound | Go HTTP regression test | backend/internal/httpapi/daily_diet_controller_test.go:275-295 | Added | ListDailyDiets and dailyDietError | Self; focused HTTP tests |
| 23 | Daily Diet shared 500/503/504 test callback | TypeScript/Bun test | frontend/src/lib/api/daily-diet-client.test.ts:155-179 | Added | fetchDailyDiets through client error mapping | Self; focused Bun tests |
| 24 | Optimization submission 429/500/503/504 test callback | TypeScript/Bun test | frontend/src/lib/api/optimization-client.test.ts:159-185 | Added | submitOptimization through client error mapping | Self; focused Bun tests |

inventory_source_count: 24
audited_symbol_count: 24
inventory_complete: true
generated_groupings:
  - "None; every task-owned behavioral unit and test symbol is listed separately."

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| REQUIRED_OPERATION_RESPONSES | Encodes exact status sets for all seven audited operations, including collection-list 404 and optimization 429/410 where applicable. | Exact comparison exposes missing and extra status keys; no wildcard or default silently satisfies the exact policy. | Immutable module data; N/A for cleanup, cancellation, and concurrency because it is static configuration. | Reads only repository source; no user data. | Seven small bounded sets. | Clear, minimally public module constant; completeness is checked separately. | Current matrix, removed-429, extra-operation, wildcard, and direct probes pass. | PASS |
| AUDITED_OPERATION_PREFIXES | Defines the two documented path families in scope. | Exact prefix or prefix-plus-slash matching excludes lookalikes and includes item subpaths. | Static tuple; no resources or concurrency. | No external input. | Constant-time comparison per path. | Narrow and readable for fixed task scope. | Extra path/method probes and current seven-key discovery pass. | PASS |
| HTTP_METHODS | Contains every OpenAPI operation method accepted by the source format. | Valid new methods such as PATCH are discovered; unknown extension keys are intentionally ignored. | Static set; no lifecycle concerns. | No external input. | Bounded membership lookup. | Idiomatic set. | Deliberate PATCH operation is rejected. | PASS |
| audited_operation_keys | Discovers every valid HTTP operation under audited path families instead of trusting only expected matrix keys. | Handles path items with parameters and multiple methods; missing or renamed operations become diagnostics through caller. Fixed indentation parsing matches checked source layout; Redocly validates YAML. | Pure function; no resources, cancellation, or shared state. | Source is local and not user-controlled. | One bounded source scan and set. | Small single-purpose parser. | Current seven keys, extra PATCH, and missing/renamed probes pass. | PASS |
| operation_response_statuses | Extracts explicit response keys for one exact path and method, including numeric, default, and uppercase wildcard keys. | Missing sections yield empty set and fail exact comparison; wildcard keys are retained and rejected by policy. Relies on canonical indentation while Redocly validates YAML. | Pure function; no resources, cancellation, or concurrency. | No user-controlled path is used for I/O. | Re-splits a small source per operation; bounded. | Simple auditable regex for repository layout. | Current, removed-429, collection, all five wildcard, and direct checks pass. | PASS |
| operation_response_mismatches | Compares discovered operation keys and exact response sets to policy, reporting missing and unexpected operations before status drift. | Reports missing expected operations, unexpected scoped operations, and status differences; empty sections fail closed as mismatches. | Pure function; no resources or concurrency. | No external data. | Seven operations and bounded source. | Diagnostics include method and path and feed CLI. | Unit tests and direct probes pass. | PASS |
| Response-drift path in main | Rejects response drift before generated output is checked or written. | Required-marker failure, response mismatch, generated-contract failure, stale output, and current output have observable returns; current check returns 0 only after all checks. | Synchronous CLI; check mode does not write. | Fixed paths; no shell execution. | Reads bounded repository files. | Correct placement before generation; concurrent generator portion excluded and inspected. | Current check and callee mutation probes pass. | PASS |
| OperationResponseDriftTest | Standard unittest grouping for response-contract regressions. | Import failure is explicit; each test uses current source and deterministic mutation. | No persistent state or external resources. | Local repository files only. | Small bounded strings and five cases. | Idiomatic standard-library test class. | All five methods pass. | PASS |
| test_current_contract_matches_audited_response_matrix | Proves current OpenAPI extraction equals exact policy. | Detects current missing or extra status/operation; paired runtime audit prevents a jointly incomplete matrix. | One local read; no cleanup or concurrency. | No user data. | Bounded source read. | Direct assertion. | Passes against seven current operations. | PASS |
| test_collection_list_404_is_audited | Locks collection-list 404 in both matrix and OpenAPI extraction. | Fails if either policy or route loses 404. | Local source only. | No external input. | Two membership checks. | Focused regression for prior finding. | Passes with paired Go regressions. | PASS |
| test_deliberate_response_mismatch_is_rejected | Proves removing optimization admission 429 creates an actionable mismatch. | Mutates only targeted 429 and checks operation and status diagnostic. | Local strings only. | No external input. | Bounded replacement. | Clear negative test. | Passes. | PASS |
| test_deliberate_extra_audited_operation_is_rejected | Proves a new scoped PATCH operation cannot escape the matrix. | Inserts 418 operation and requires exact unexpected-operation diagnostic. | Local strings only. | No external input. | One bounded mutation. | Direct complete-operation regression. | Passes; missing-operation probe also passes. | PASS |
| test_wildcard_response_keys_are_rejected_by_exact_policy | Proves all valid wildcard classes are rejected when not in exact matrix. | Subtests insert 1XX, 2XX, 3XX, 4XX, 5XX and require one status mismatch each. | Deterministic local subtests. | No external input. | Five bounded mutations. | Idiomatic table-driven subtests. | All five pass. | PASS |
| GET /api/v1/daily-diets response map | Documents collection success and every reachable operation error. | Includes 200, 401, 403, 404 unavailable meal, 500, 503, and 504. | Static contract; no runtime lifecycle. | Shared safe Error envelope. | Static YAML. | Exact and explicit; no inheritance. | Matrix, Go regressions, and lint pass. | PASS |
| POST /api/v1/daily-diets response map | Documents create success, validation, auth/security, missing meal, conflict, generic, dependency, and timeout. | Includes 201, 400, 401, 403, 404, 409, 500, 503, 504; no unsupported 429. | Static contract. | Ownership and details remain behind safe Error. | Static YAML. | Matches controller/service behavior. | Matrix and focused client tests pass. | PASS |
| GET /api/v1/daily-diets/{dietId} response map | Documents item read success, UUID validation, auth/security, ownership-safe not-found, dependency/generic, and timeout. | Includes 200, 400, 401, 403, 404, 500, 503, 504. | Static contract. | Cross-user access uses safe policy. | Static YAML. | Exact map. | Matrix and source inspection pass. | PASS |
| PUT /api/v1/daily-diets/{dietId} response map | Documents replacement success, validation, auth/CSRF/CORS, not-found, conflict, dependency/generic, and timeout. | Includes 200, 400, 401, 403, 404, 409, 500, 503, 504. | Static contract. | Shared safe Error envelope. | Static YAML. | Matches route middleware and service. | Matrix and source audit pass. | PASS |
| DELETE /api/v1/daily-diets/{dietId} response map | Documents deletion success, validation, auth/CSRF/CORS, not-found, dependency/generic, and timeout. | Includes 204, 400, 401, 403, 404, 500, 503, 504. | Static contract. | Shared safe Error envelope. | Static YAML. | Explicit 204 description. | Matrix and source audit pass. | PASS |
| POST /api/v1/optimization/jobs response map | Documents async acceptance and submission failures, including admission and queue behavior. | Includes 202, 400, 401, 403, 404, 409, 429, 500, 503, 504; stale 422 removed because infeasibility is a polled failed state. | Static contract; lifecycle inspected at controller boundary. | User and queue details remain behind safe errors. | Static YAML. | Agrees with DESIGN-004 and controller. | Matrix, client tests, and source audit pass. | PASS |
| GET /api/v1/optimization/jobs/{jobId} response map | Documents polling success, validation, auth/security, ownership-safe not-found, expiry, dependency/generic, and timeout. | Includes 200, 400, 401, 403, 404, 410, 500, 503, 504. | Static contract; lifecycle remains controller/worker boundary. | Cross-user jobs hidden as 404. | Static YAML. | Exact map. | Matrix and polling source audit pass. | PASS |
| TestServiceListReturnsNotFoundWhenSavedMealIsUnavailable | Proves Service.List preserves repository not-found when projecting a diet whose meal is unavailable. | Owned diet with one missing meal must return ErrorKindNotFound; runtime behavior unchanged. | Single-threaded in-memory fixture; no external resources. | User-owned fixture. | One diet and one entry. | Focused service regression. | Passes focused and full package tests. | PASS |
| TestProfileControllerDailyDietListMapsUnavailableMealToNotFound | Proves authenticated collection route maps service not-found to HTTP 404 and stable not_found code. | Router/auth/controller boundary is exercised and response status/envelope checked. | Fiber app per test; body closed; no shared mutable state. | User is server-derived; no sensitive text asserted. | One in-memory request. | Focused integration regression. | Passes focused and full HTTP package tests. | PASS |
| Daily Diet shared 500/503/504 test callback | Proves valid generic, dependency, and timeout envelopes become bounded client errors. | One response per status; asserts status, category, code, request ID, retryability. | Existing afterEach restores fetch and resets mock; no timers or concurrency. | Bounded typed error fields. | Three small responses. | Deterministic table-driven Bun test. | Passes; malformed/empty bodies are optional gap only. | PASS |
| Optimization submission 429/500/503/504 test callback | Proves admission, generic, dependency, and timeout submission envelopes become bounded client errors. | Four valid envelopes; asserts status, code, request ID, retryability. | Existing cleanup restores fetch and resets mock. | Safe envelope fields pass through sanitization. | Four small responses. | Clear table-driven Bun test. | Passes; malformed/empty bodies are optional gap only. | PASS |

Mandatory audit questions were answered in every row. Static YAML/configuration units have no runtime cleanup, cancellation, or concurrency. Pure Python units have no external resources. Tests use existing fixture cleanup. Runtime callers were inspected line-by-line in router, auth, CSRF, profile, Daily Diet, optimization, service, repository, SQL, and frontend client sources.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| nit | frontend/src/lib/api/daily-diet-client.test.ts:155-179; frontend/src/lib/api/optimization-client.test.ts:159-185 | Added client error tests | New tests use valid JSON error envelopes, so empty or malformed 500/503/504 bodies are not directly exercised. | responseError in both clients was read line-by-line and has status-derived fallback behavior; all required criteria pass. | Optional follow-up: add malformed/empty-body cases if proxy/upstream body loss is in scope. Not blocking for task 213. |

blocking_findings: 0
important_findings: 0
optional_findings: 1

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| git status --short | /home/wiktor/Work/mealswapp | 0 | PASS; dirty concurrent work and untracked preparation/review files captured | Scope capture |
| git rev-parse HEAD | /home/wiktor/Work/mealswapp | 0 | PASS; baseline confirmed | Baseline capture |
| git diff --name-status a4e31367485b03269e90b5607f2057c9568bb5b1 | /home/wiktor/Work/mealswapp | 0 | PASS; shared-worktree surface enumerated | Scope reconstruction |
| git diff --unified=0 baseline -- task-owned paths | /home/wiktor/Work/mealswapp | 0 | PASS; task hunks separated from concurrent overlap | Scope reconstruction |
| python3 -m unittest scripts/test_generate_api_types.py | /home/wiktor/Work/mealswapp | 0 | PASS; 5 tests | Python drift evidence |
| python3 -m py_compile scripts/generate-api-types.py scripts/test_generate_api_types.py | /home/wiktor/Work/mealswapp | 0 | PASS | Python syntax evidence |
| python3 scripts/generate-api-types.py --check | /home/wiktor/Work/mealswapp | 0 | PASS; generated types current | API drift evidence |
| npx --no-install redocly lint api/openapi.yaml | /home/wiktor/Work/mealswapp | 0 | PASS; one existing explicitly ignored OAuth 2XX warning | OpenAPI lint |
| python3 scripts/validate-task-list.py | /home/wiktor/Work/mealswapp | 0 | PASS; 237 sequential tasks | Repository validator |
| python3 scripts/validate-traceability.py | /home/wiktor/Work/mealswapp | 0 | PASS | Repository validator |
| BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check:api-types | /home/wiktor/Work/mealswapp/frontend | 0 | PASS | Frontend API drift |
| BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/daily-diet-client.test.ts src/lib/api/optimization-client.test.ts | /home/wiktor/Work/mealswapp/frontend | 0 | PASS; 13 tests and 42 expectations | Focused client tests |
| BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage src/lib/api/daily-diet-client.test.ts src/lib/api/optimization-client.test.ts | /home/wiktor/Work/mealswapp/frontend | 0 | PASS; 60.24% aggregate, 82.93% Daily Diet, 90.59% optimization | Coverage stdout |
| BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck | /home/wiktor/Work/mealswapp/frontend | 0 | PASS | TypeScript check |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/dailydiet -run 'TestServiceListReturnsNotFoundWhenSavedMealIsUnavailable' -count=1 | /home/wiktor/Work/mealswapp/backend | 0 | PASS | Focused Go test |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -run 'TestProfileControllerDailyDietListMapsUnavailableMealToNotFound|Test(ProfileControllerDailyDiet|OptimizationHTTP)' -count=1 | /home/wiktor/Work/mealswapp/backend | 0 | PASS | Focused Go test |
| GOCACHE and GOMODCACHE go test ./internal/dailydiet ./internal/httpapi -count=1 | /home/wiktor/Work/mealswapp/backend | 0 | PASS | Affected Go packages |
| gofmt -d internal/dailydiet/service_test.go internal/httpapi/daily_diet_controller_test.go | /home/wiktor/Work/mealswapp/backend | 0 | PASS; no diff | Go formatting |
| GOCACHE and GOMODCACHE go vet ./internal/dailydiet ./internal/httpapi | /home/wiktor/Work/mealswapp/backend | 0 | PASS | Go static analysis |
| Direct Python audit of discovered operations, exact sets, removed 429, extra PATCH, all wildcards, renamed operation | /home/wiktor/Work/mealswapp | 0 | PASS; exact seven keys and expected diagnostics | Manual probes |
| git diff --check task-owned implementation/test paths | /home/wiktor/Work/mealswapp | 0 | PASS | Whitespace check |
| sha256sum over every reviewed implementation/context/source file | /home/wiktor/Work/mealswapp | 0 | PASS; hashes recorded below | Staleness fingerprints |
| python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-213-review.md | /home/wiktor/Work/mealswapp | 0 | PASS; structurally valid evidence | Final evidence validator |

The full aggregate scripts/check.py was not re-run because the shared worktree contains unrelated concurrent phase changes; all task-required checks and affected package/static checks were run directly. No command failure is being suppressed as evidence for this task.

## 9. Files Inspected and Staleness Fingerprints

All reviewed implementation, caller, SQL, configuration, and design-source files were hashed after inspection with SHA-256.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| api/openapi.yaml | Seven operation response maps and shared Error component | None | SHA-256 | 6368ee9c1321104d0e645ed8a3e6b73f8f14c1a161835a5f388a0e6e2fa3da4a |
| scripts/generate-api-types.py | Matrix, discovery, extraction, mismatch enforcement, CLI caller | None | SHA-256 | b6b241afb9c13c4206e30eb1e12ae4adcd32d6734cc065b33ece5d47ed4f4e87 |
| scripts/test_generate_api_types.py | Five focused drift tests | None | SHA-256 | 475c5f9fe28e4bba538ced96faa40b07223708cd37ae2a227f7049ff900ad4d5 |
| backend/internal/dailydiet/service_test.go | Collection-list service regression plus concurrent test context | None | SHA-256 | 8974b57ca08281025d247b92a1f81edf6d922aee1b5684653273e2e5f9389907 |
| backend/internal/httpapi/daily_diet_controller_test.go | Collection-list controller regression plus concurrent test context | None | SHA-256 | 34485172a7e6ada598863b6ff01e7c3e8ffcc290bc44961866e629bbcb40d777 |
| frontend/src/lib/api/daily-diet-client.test.ts | Added Daily Diet error tests | Optional malformed-body gap | SHA-256 | bb3e33b04093e8650e3843556de3b0d6a418c2ff3d9e86b14dc23d12aca19c90 |
| frontend/src/lib/api/optimization-client.test.ts | Added optimization error tests | Optional malformed-body gap | SHA-256 | 8935ee1e3e571d91648bd582da96fc5fdbfc0cb02f0f426412a004dcace1e5f8 |
| backend/internal/httpapi/router.go | Gateway timeout, CORS, validation, generic classification | None | SHA-256 | cd4c888689151d66051561c3aaedd6ac379149df950a09eec30d0f7591566125 |
| backend/internal/httpapi/auth_middleware.go | Protected-route 401 behavior | None | SHA-256 | 71b8edaa05c479e31703871ac7066dc384d534e40dfb4cb565bee5cfc8f0f7b9 |
| backend/internal/httpapi/csrf.go | Mutation 403 behavior | None | SHA-256 | 9c4e5f40018e23a5cc1956b0bbd9959a288e9ec0dc6fbc1e937b266c4dbfded1 |
| backend/internal/httpapi/profile_controller.go | Daily Diet route registration | None | SHA-256 | d1c6599efd07921e09092651c6732e23bfe6de66e3cb08d5fa69faf0cf910676 |
| backend/internal/httpapi/daily_diet_controller.go | List mapping and Daily Diet errors | None | SHA-256 | d2e0e9346968402a2453012857c15432b363b9c8aef68d00d5349cb43aaf7dea |
| backend/internal/httpapi/optimization_controller.go | Optimization submission/poll statuses | None | SHA-256 | ef7c3eb939e40ce3f75ef9c8ec912ba75bb2acd871520c2b876b86e2ea717855 |
| backend/internal/dailydiet/service.go | List projection and unavailable meal propagation | None | SHA-256 | 191c17f3cdc84dacf03a0c3007ea29adbfd3c02b05a0396f533d37ebc6820d6c |
| backend/internal/repository/meal_repository.go | GetByID soft-delete context | None | SHA-256 | 4f034eb4245c83617c8662d212022f34265eb71635bb4c46bc6f0539430d2088 |
| backend/internal/repository/user_data_repository.go | Saved-diet list and entry loading callers | None | SHA-256 | 41bf37f97e5dfb35b5a79620452e360b346e3d3368a15358301145765054651e |
| backend/internal/repository/sql/meal_get_by_id.sql | Excludes soft-deleted meals unless requested | None | SHA-256 | c01a3bcc44f4f5479e4a0b3779d84b9219593fdea88066fab35758f6db448b94 |
| backend/internal/repository/sql/meal_soft_delete.sql | Reproduction precondition retaining FK reference | None | SHA-256 | 9dbc51200a795ab825a9e70c54b53defe1cc227ddf871feb643d06678613cbb5 |
| backend/internal/repository/sql/saved_diet_list.sql | User-scoped saved-diet list | None | SHA-256 | 3a54bea21a31ad5d97fad6834a288ad59874e6de1d03e09e4580a2e2caff2de8 |
| backend/internal/repository/sql/saved_diet_entry_list.sql | Saved-diet meal-entry loading | None | SHA-256 | 172303c0226eaa855979b9a1107081449e446fa9f4df5582b16f121b3465c292 |
| database/migrations/000019_saved_diet_persistence.up.sql | Confirms meal FK without cascade | None | SHA-256 | b063c95f567b3073fc9160b6f52b1f84f3d863a9712aa8f394342918cf570999 |
| frontend/src/lib/api/daily-diet-client.ts | List/client error consumer | Optional malformed-body gap | SHA-256 | d109f5bd39d786362693f653b2478174e71ead2e2b424dc6b0e0704d1fadbd0c |
| frontend/src/lib/api/optimization-client.ts | Submission/poll client error consumer | Optional malformed-body gap | SHA-256 | 145e51d097243affd72513c7de1174e7e49324aa2b2a4b460db86e6e7de91635 |
| frontend/src/lib/api/generated.ts | Concurrent generated-type boundary | None; excluded concurrent change | SHA-256 | 361ce14d3cde8ae90afe0bc074ffb3e301c751a8a9603fc7a300e7d6b49cf20b |
| frontend/package.json | check:api-types caller | None | SHA-256 | 1819d69ba01bcf8282812eb67ad492f4c7892127c6e6b4666b78b9ce27e22138 |
| docs/design/DESIGN-004.md | Optimization statuses, admission, queue, expiry, timeout | None | SHA-256 | 47ab62398f77413f295ac9e0b56d1d9cf92000f7f3edeed3355cf5c56a550410 |
| docs/design/DESIGN-008.md | Profile/SavedDataRepository ownership and errors | None | SHA-256 | 551880a70a3b42698e11632a471957bbecc6011900cf121c34f1ff3c3db18b87 |
| docs/design/DESIGN-010.md | Route, validation, CORS, CSRF, rate-limit, timeout | None | SHA-256 | fabf99b19e918272ffd711122662b67174a7b2e24e4febe87101b505ec7b |
| docs/design/DESIGN-017.md | Error envelope, safe messages, 500/timeout policy | None | SHA-256 | 5c92a6776af7bb9584c73098b4c53e9c53e182b7388c990a4fd42f043f44464c |
| docs/design/01_TECH_STACK.md | OpenAPI, Go/Fiber, Bun/TypeScript stack | None | SHA-256 | 64e2cf45ec039db597244678b17e8028f4705b86dcad01e7051e3e686d6f9338 |
| docs/implementation/05_DEVELOPMENT.md | OpenAPI source-of-truth and generator commands | None | SHA-256 | 74187d4ebd337d7d2374ca987b36c0695137f941945d97df0565fbf9043c7806 |

all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The previous task-213-review.md was a rejected review with pre-repair hashes and findings; it was re-read and replaced."
  - "No current implementation hash changed after the hash capture above."

## 10. Coverage and Exceptions

- [x] The focused frontend coverage command ran.
- [x] Its stdout measurements and uncovered changed-client branches were inspected.
- [x] No task-specific coverage exception was claimed; the task adds no production Go or TypeScript symbols.
- [x] The repository phase-level aggregate coverage gate was not substituted by this focused command; it is outside task-specific verification criteria and concurrent worktree scope.

coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "stdout from focused Bun coverage command; no committed report"
observed_line_coverage: "60.24% focused aggregate; 82.93% daily-diet-client.ts; 90.59% optimization-client.ts"
coverage_passed: true

Coverage finding: The focused command passes. Uncovered client branches are existing fallback, key-generation, and malformed-body paths; the malformed-body gap is recorded as the single optional finding and does not affect required response-contract criteria.

## 11. Negative and Regression Checks

- [x] Existing focused Python, frontend, and affected Go package tests pass.
- [x] Collection-list 404 was challenged through unavailable-meal service path, controller mapping, SQL soft-delete query, and retained meal foreign key.
- [x] Operation discovery was challenged with an extra PATCH operation and a renamed/missing operation.
- [x] Wildcard response keys 1XX, 2XX, 3XX, 4XX, and 5XX were each challenged and rejected by the exact policy.
- [x] Removed optimization submission 429 was rejected by drift enforcement.
- [x] No unrelated dependency or architectural boundary was introduced by task 213; concurrent changes are explicitly excluded.
- [x] No source-of-truth documentation is contradicted: operation maps are exact and shared Error no longer enumerates a stale status list.
- [x] No generated/cache/build/temporary artifact was intentionally added by this review.
- [x] Public additions are used: discovery and mismatch functions are called by main and focused tests; response maps document reachable behavior.
- [x] Duplicate response matrices/helpers were searched for; the task-owned checker is unique.
- [x] Error, cleanup, timeout, concurrency, and malformed-input paths were challenged. Cleanup/concurrency are N/A for static/Python/test units; gateway timeout, generic classification, safe client fallback, and malformed source behavior were inspected. One optional malformed-body test gap remains visible in Section 7.

## 12. Decision

A task may be PASSED only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking or important finding remains. Those conditions are satisfied. The one remaining finding is optional and does not block acceptance.

Before accepting the decision, this command was run successfully:

    python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-213-review.md

decision: "PASSED"
reason: "All seven audited operations have complete exact response documentation, collection-list 404 behavior is proven, operation discovery and wildcard rejection are enforced, shared Error documentation is consistent, and all required checks pass."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "NONE"

## 13. Repair Context

Not applicable to the final PASSED decision. The previous rejection's three important findings were independently re-verified as repaired in current source: collection-list 404 policy and regressions, complete discovered operation-key comparison, and non-contradictory shared Error documentation.
