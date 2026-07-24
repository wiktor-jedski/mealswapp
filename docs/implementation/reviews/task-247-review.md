# Review Evidence: Task 247 — DESIGN-009 AdminController

~~~~yaml
task_id: 247
component: "AdminController"
static_aspect: "DESIGN-009 admin gateway, authorization, CSRF, validation, rate limiting, request-correlated audit, and transactional mutation boundary"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-21T13:20:05Z"
review_agent: "Codex independent reviewer"
evidence_file: "docs/implementation/reviews/task-247-review.md"
baseline_ref: "HEAD 81ca40ce00cb667ea29243ed2d34068e11229a69 plus current task-247 worktree files"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "Go + HTTP/auth/CSRF + transaction/security + concurrency + error handling"
repair_context_required: true
~~~~

## 1. Task Source

**Description:** Register the versioned admin route group with server-verified JWT-cookie admin-role enforcement, CSRF and validation on mutations, scoped rate limits, request-correlated audit coordination, safe reads when audit is unavailable, and fail-closed transactional audit behavior for every admin mutation.

**Depends On:** 61 PASSED; 65 PASSED; 77 PASSED; 93 PASSED; 242 PASSED.

**Testing Coverage Exceptions:** The task row says `None`. No task-247 coverage exception is authorized or required; the complete task-owned executable surface reaches 100% in the focused package profiles.

**Verification Criteria:** Anonymous requests receive 401; authenticated non-admin users receive 403; spoofed role or identity input is ignored; admins reach only documented routes; CSRF, validation, rate, and audit controls execute in the required order; reads degrade safely when allowed; mutations abort and roll back when audit persistence fails; and audit/log/error envelopes contain request IDs without raw PII, secrets, or provider payloads.

## 2. Pre-Review Gates

- [x] Input status is PREPARED.
- [x] Every dependency is PASSED or otherwise available as a current prerequisite.
- [x] The preparation report was read and its current implementation fingerprints were checked.
- [x] The task-owned diff was reconstructed from HEAD and current files despite unrelated Phase 08 worktree changes.
- [x] The review template was read fully.
- [x] `code-review-skill` was invoked exactly once; Go, HTTP/auth/CSRF, transaction/security, concurrency, error-handling, and review-checklist guidance was read.
- [x] The reviewer independently reran the repaired behavior and did not rely on the preparation report's claimed results.
- [x] No production code or task-list file was edited by this review.

~~~~yaml
pre_review_gates_passed: true
blocking_issue: "None. The unrelated full-suite task-240 failure is recorded as an external worktree failure, not a task-247 finding."
~~~~

## 3. Review Baseline and Change Surface

The review used branch `multistep-phase-08` at `81ca40ce00cb667ea29243ed2d34068e11229a69` (`phase 08 planned`). The worktree is intentionally dirty with unrelated Phase 08 implementation, migration, API, frontend, and review work. Task 247 remains `PREPARED` in `docs/implementation/02_TASK_LIST.md`; its status cell and the task-list file were not changed.

The task-owned surface was reconstructed with `git rev-parse HEAD`, `git status --short`, tracked `git diff HEAD`, untracked-file inspection, `git diff --no-index /dev/null` for the new controller/tests, and targeted `rg` call-site searches. The review bounded the shared-file inspection to the task-247 symbols and directly called gateway/auth/CSRF/rate/audit/transaction/error paths. Unrelated changes in shared repository files were not attributed to task 247.

Reviewed implementation and test paths:

| Changed file | Task-owned surface | Review scope |
|---|---|---|
| `backend/internal/httpapi/admin_controller.go` | Admin types, route construction/validation, authorization, transactional mutation | Full file |
| `backend/internal/httpapi/admin_controller_test.go` | HTTP authorization, route collisions, ordering, request IDs, rollback, response serialization | Full file |
| `backend/internal/httpapi/router.go` | Server request ID and admin middleware ordering | Relevant symbols/hunks |
| `backend/internal/httpapi/router_test.go` | Signed role-aware JWT test helper | Relevant helper hunk |
| `backend/internal/repository/types.go` | Audit DTO and transactional audit contract | Relevant symbols |
| `backend/internal/repository/errors.go` | Audit-persistence sentinel | Relevant symbol |
| `backend/internal/repository/compliance_repository.go` | Audit persistence, strict snapshot sanitation, transaction coordinator | Relevant symbols; unrelated Phase 08 changes excluded |
| `backend/internal/repository/postgres_repository_test.go` | PostgreSQL audit and rollback coverage | Relevant test hunks |
| `backend/internal/repository/admin_audit_security_test.go` | Snapshot privacy, rollback, cause-chain, and success-path tests | Full file |

Relevant contracts inspected included `DESIGN-009`, `DESIGN-010`, `DESIGN-013`, the preparation report, `requireAuth`, `authenticatedUser`, `JWTAuthenticator.Authenticate`, `registerV1Routes`, `requestID`, `requireAudit`, `rateLimitHandler`, `writeError`, `withTransaction`, and parameterized admin-audit SQL.

## 4. Acceptance Criteria Checklist

| # | Criterion | Result | Current evidence |
|---:|---|---|---|
| 1 | Anonymous admin requests receive 401. | PASS | Signed-cookie HTTP integration and direct `RequireAdmin`/wrapper tests. |
| 2 | Authenticated non-admin users receive 403. | PASS | Server-signed non-admin JWT-cookie test and exact `UserRoleAdmin` comparison. |
| 3 | Spoofed role or identity input is ignored. | PASS | Headers/body/query are not auth sources; tests send spoofed role, user ID, and request ID values. |
| 4 | Admins reach only documented, semantically collision-free routes. | PASS | Startup rejects wildcard/optional/malformed paths, exact duplicates, parameter aliases, and static/parameter overlaps in both registration orders. |
| 5 | CSRF, validation, rate, and audit controls execute in the required order. | PASS | `registerV1Routes` composes auth → admin role → CSRF → validation → rate limit → required audit → handler; short-circuit counters verify order. |
| 6 | Reads degrade safely when audit is unavailable. | PASS | Read route definitions do not require mutation audit coordination; nil audit is accepted for the read path. |
| 7 | Mutations abort and roll back when audit persistence fails, including response serialization failure. | PASS | Real PostgreSQL and fake-coordinator tests prove rollback; response envelope marshaling occurs inside the transaction callback before audit insert/commit. |
| 8 | Audit, log, and error envelopes contain server-owned request IDs without raw PII, secrets, or provider payloads. | PASS | UUID request IDs replace client headers; strict entity/action schemas retain only booleans/fixed status codes and canonicalize output; logs/errors are sanitized. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Result | Main callers/consumers | Tests/evidence |
|---:|---|---|---|---|---|---|
| 1 | `AdminContext` | behavioral type | `admin_controller.go:17` | PASS | `RequireAdmin`, admin handlers | 401/403/correlation tests |
| 2 | `AdminMutationResult` | response/audit DTO | `admin_controller.go:25` | PASS | Transactional wrapper and mutation callbacks | Response/status/serialization tests |
| 3 | `AdminMutationHandler` | function type | `admin_controller.go:33` | PASS | `AdminRouteDefinition`, wrapper | Mutation tests and compile contract |
| 4 | `AdminRouteDefinition` | route configuration | `admin_controller.go:37` | PASS | `AdminController.Routes` | Control and collision rejection tests |
| 5 | `AdminController` | controller type | `admin_controller.go:50` | PASS | Gateway composition | Constructor/route/mutation tests |
| 6 | `NewAdminController` | constructor | `admin_controller.go:61` | PASS | Route ownership and audit dependency | Focused controller tests |
| 7 | `AdminController.Routes` | route builder | `admin_controller.go:67` | PASS | `registerV1Routes` | Semantic collision and control tests |
| 8 | `adminRoutePathsCollide` | route matcher | `admin_controller.go:94` | PASS | `Routes` | Alias/static-parameter both-order tests |
| 9 | `RequireAdmin` | authorization method | `admin_controller.go:112` | PASS | Role middleware and mutation wrapper | Anonymous/non-admin/spoof tests |
| 10 | `requireAdminRole` | middleware | `admin_controller.go:125` | PASS | `registerV1Routes` | Gateway ordering tests |
| 11 | `validateRoute` | configuration validator | `admin_controller.go:134` | PASS | `Routes` | Missing-control/grammar tests |
| 12 | `isSafeAdminRoutePath` | path grammar | `admin_controller.go:158` | PASS | `validateRoute` | Empty, slash, duplicate, wildcard, boundary tests |
| 13 | `isAdminRouteLiteral` | segment validator | `admin_controller.go:187` | PASS | Path grammar | Allowed-character boundary tests |
| 14 | `isAdminRouteIdentifier` | parameter validator | `admin_controller.go:202` | PASS | Path grammar | Lower-camel and invalid-name tests |
| 15 | `transactionalMutation` | transaction/response wrapper | `admin_controller.go:214` | PASS | Mutation route handlers | Commit, domain-error, status, marshal, rollback tests |
| 16 | `RouteDefinition.RequiresAdmin` | route policy field | `router.go:97` | PASS | `registerV1Routes` | Admin-without-auth startup rejection |
| 17 | `serverRequestID` | request boundary | `router.go:170` | PASS | `NewRouter` | Client-header replacement and concurrent correlation tests |
| 18 | `registerV1Routes` | route registration | `router.go:189` | PASS | `NewRouter` | Auth/CSRF/validation/rate/audit ordering tests |
| 19 | `AdminAuditEntry/AdminAuditChanges` | audit DTOs | `types.go:503,537` | PASS | Repository/controller boundary | Persistence, correlation, safe snapshot, and rollback tests |
| 20 | `AdminMutationExecutor/AdminMutationAuditRepository` | transaction contracts | `types.go:820,824` | PASS | Controller and PostgreSQL implementation | Compile assertion and commit/rollback tests |
| 21 | `ErrAdminAuditPersistence` | sentinel error | `errors.go:8` | PASS | HTTP classification and repository wrapper | Cause-chain/error-envelope tests |
| 22 | `PersistAuditEntry` | persistence method | `compliance_repository.go:353` | PASS | PostgreSQL audit writes | Parameterized SQL and strict sanitization tests |
| 23 | `WithMutationAudit` | transaction method | `compliance_repository.go:391` | PASS | Controller mutation wrapper | Real rollback, success, invalid audit, cause tests |
| 24 | `validateAdminAuditEntry` | audit validator | `compliance_repository.go:527` | PASS | Audit persistence | Required fields, before/after, schema tests |
| 25 | `adminAuditSnapshotRule/Schemas` | strict metadata policy | `compliance_repository.go:546` | PASS | Snapshot sanitizer | Explicit entity/action schemas |
| 26 | `sanitizeAdminAuditSnapshot` | privacy/size sanitizer | `compliance_repository.go:567` | PASS | Audit validation/persistence | Forbidden keys, types, enums, size, canonicalization tests |
| 27 | `testJWTAuthRole` | signed-auth test helper | `router_test.go:109` | PASS | Admin HTTP tests | Admin/non-admin signed role fixtures |
| 28 | HTTP route and security tests | route/order/correlation tests | `admin_controller_test.go:97,181,223` | PASS | Gateway stack | Controls, semantic collisions, request IDs, envelopes |
| 29 | Repository audit security and PostgreSQL tests | transaction/privacy tests | `admin_audit_security_test.go`, `postgres_repository_test.go` | PASS | Repository boundary | Strict snapshot, rollback, commit, cause, and DB persistence |

~~~~yaml
inventory_source_count: 29
audited_symbol_count: 29
inventory_complete: true
generated_groupings:
  - "No generated artifacts are in the task-owned surface."
~~~~

## 6. Function-Level Audit

| Symbol/unit | Contract/invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundary | Performance/I/O | Result and evidence |
|---|---|---|---|---|---|---|
| `AdminContext` | Server user, exact admin role, request ID | Missing/admin/non-admin outcomes safe | Per-request value | Headers cannot set identity | Small value | PASS; HTTP tests |
| `AdminMutationResult` | Deferred safe response plus audit changes | Invalid status/data/marshal paths fail closed | Request-local | Audit bytes reach repository gate | Bounded sanitizer | PASS; response tests |
| `AdminMutationHandler` | Receives transaction-scoped executor | Callback errors propagate | Repository owns transaction | Wrapper controls auth/audit boundary | Handler-owned SQL | PASS; contract and tests |
| `AdminRouteDefinition` | Read/mutation controls are explicit | Unsafe combinations panic at startup | Registration-only | No bypassable route shape | Linear config | PASS; control tests |
| `AdminController` | Owns route and audit policy | Nil audit remains safe for reads | Route config is startup-owned | Collision and policy checks | O(n²) startup-only | PASS; route tests |
| `NewAdminController` | Captures explicit dependencies | Valid fixtures construct safely | No request state | Caller cannot bypass `Routes` checks | O(n) storage | PASS |
| `Routes` | Emits `/admin` routes and control metadata | Semantically colliding same-method routes panic | Local `seen`; no goroutines | Prevents registration-order dispatch ambiguity | O(n²) bounded startup work | PASS; both-order tests |
| `adminRoutePathsCollide` | Equal-length templates overlap if each segment can match | Static/static mismatch remains valid | Pure function | Rejects aliases and static/parameter overlap | O(segments) | PASS; adversarial tests |
| `RequireAdmin` | Exact signed role and server identity | 401/403 classified safely | No blocking I/O | Client role/ID/request headers ignored | Constant-time lookup | PASS |
| `requireAdminRole` | Runs after verified authentication | Propagates `RequireAdmin` errors | No shared state | Admin gate before mutation controls | Negligible | PASS |
| `validateRoute` | Only documented safe paths and controls | Wildcard/optional/empty/duplicate/missing-control branches covered | Startup-only | No unsafe handler registration | Small | PASS |
| `isSafeAdminRoutePath` | Explicit literals and required named parameters | Grammar boundaries and duplicate names rejected | Pure function | Prevents catchall/optional dispatch | O(segments) | PASS; 100% |
| `isAdminRouteLiteral` | Lower kebab-case literals | Character/index boundaries tested | Pure function | No matcher metacharacters | O(segment) | PASS; 100% |
| `isAdminRouteIdentifier` | Lower-camel required parameters | Invalid/empty/upper-leading cases tested | Pure function | Parameter grammar is constrained | O(identifier) | PASS; 100% |
| `transactionalMutation` | Mutation, audit, and response are atomic | Domain/status/no-content/marshal/audit errors fail closed | Context propagated; no goroutine leak | Safe generic dependency errors and server ID | One callback plus marshal | PASS; 100% |
| `RouteDefinition.RequiresAdmin` | Admin flag is explicit route policy | Missing auth panics at registration | Static metadata | Prevents accidental unauthenticated admin route | None | PASS |
| `serverRequestID` | Every request receives a fresh server UUID | Client header is overwritten | Per-request local; UUID generation is concurrency-safe | Untrusted correlation input cannot control logs | Constant-time | PASS; 100% |
| `registerV1Routes` | Auth → admin → CSRF → validation → rate → audit → handler | Short-circuit paths prevent later work | Fiber middleware lifecycle | Every mutation has required controls | Expected middleware cost | PASS; 100% |
| `AdminAuditEntry/Changes` | Gateway identity immutable; mutation fields callback-derived | Before/after validation covers both fields | Transaction-local values | No raw free text reaches storage | Bounded payloads | PASS |
| `AdminMutationExecutor` | Callback receives transaction executor | Mutation errors return before audit/commit | Same tx required by repository | Prevents separate-pool mutation contract | SQL interface only | PASS |
| `AdminMutationAuditRepository` | Callback and audit commit atomically | Nil/prefilled seed and insert errors fail | Deferred rollback and context | Fail-closed audit boundary | One transaction | PASS |
| `ErrAdminAuditPersistence` | Audit failure remains classifiable | Wrapped sentinel and root cause survive | No shared state | Client gets generic 503, not internal cause | None | PASS |
| `PersistAuditEntry` | Validates and canonicalizes before SQL | Invalid snapshots do not reach query | Uses request context | Parameterized insert and strict schema | One insert | PASS; 100% |
| `WithMutationAudit` | Mutation and audit use one DB transaction | Mutation, validation, persistence, commit errors roll back | Deferred rollback; cancellation propagated | Audit failure cannot become best effort | One transaction | PASS; 100% |
| `validateAdminAuditEntry` | Required identity/action/entity and safe snapshots | Before and after independently checked | Pure validation | Fixed schema is final repository gate | Bounded JSON parse | PASS; 100% |
| `sanitizeAdminAuditSnapshot` | Explicit entity/action fields only | Malformed, unknown, nested, wrong type, invalid enum, oversized input rejected | No external state | No names/reasons/IDs/provider text/free text | 4096-byte input cap; canonical output | PASS; 100% |
| `testJWTAuthRole` | Test roles are server-signed | Admin/user variants exercised | Test-local session state | Models production claim source | Test-only | PASS |
| Route/security HTTP tests | Acceptance order and reachability are observable | Unauthorized, forbidden, CSRF, validation, rate, success, error branches | Test counters and responses | Spoof/privacy assertions | Small | PASS |
| Repository audit tests | Privacy and atomicity are regression-protected | Unsafe `Before`/`After`, duplicate-key canonicalization, rollback, cause, success | Fake and real DB paths | No secret/PII/provider text persists | Integration cost bounded | PASS |

## 7. Findings

No blocking, important, or optional findings remain in the task-247 review surface.

The prior findings were independently rechecked and are closed:

- Semantic route collision rejection now compares safe templates by matching segment semantics, independent of registration order.
- Audit snapshots now use explicit entity/action schemas, fixed booleans/enums, a 4096-byte input cap, and canonical output; free-text fields are rejected.
- Complete package coverage profiles report 100% for every task-owned executable function; no exception is used.
- Server-generated request IDs, auth/CSRF/validation/rate/audit order, safe reads, role-spoof resistance, fail-closed rollback, response serialization before commit, cleanup, concurrency, and privacy checks pass.

~~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 0
~~~~

## 8. Commands Run

| Command | Working directory | Exit code | Result | Evidence |
|---|---|---:|---|---|
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race -coverprofile=/tmp/task-247-review.cover ./internal/httpapi ./internal/repository` | `backend/` | 1 | Environment/test-isolation failure | HTTP passed; repository hit unrelated concurrent migration bootstrap error in `TestPostgresMealRepositorySingleRecipeAndMacros` (`pg_type_typname_nsp_index`). |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race -run 'TestAdmin|TestPostgresComplianceAndAdmin|TestAdminMutation' ./internal/httpapi ./internal/repository` | `backend/` | 0 | PASS | Task-specific HTTP/repository tests pass under race with no race report. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race -coverprofile=/tmp/task-247-http-full.cover ./internal/httpapi` | `backend/` | 0 | PASS | HTTP package 88.5%; every task-owned HTTP function, including `registerV1Routes`, is 100.0%. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -p 1 -race -coverprofile=/tmp/task-247-repository-full.cover ./internal/repository` | `backend/` | 0 | PASS | Repository package 93.4%; every task-owned repository function is 100.0%. |
| `go tool cover -func=/tmp/task-247-http-full.cover` and `go tool cover -func=/tmp/task-247-repository-full.cover` filtered to task symbols | `backend/` | 0 | PASS | 100.0% for all task-owned executable functions. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./...` | `backend/` | 1 | External failure only | All task-247 packages pass; untouched `internal/app.TestTask240CustomItemErasureIntegration` leaves 2 owner custom items. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./...` | `backend/` | 1 | External failure only | Same untouched task-240 cleanup assertion; task-247 packages pass and no race is reported. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | `backend/` | 0 | PASS | No diagnostics. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `backend/` | 0 | PASS | No vulnerabilities in called/imported code; 18 unreachable required-module advisories reported. |
| `npx --no-install redocly lint api/openapi.yaml` | repository root | 0 | PASS with existing warning | OpenAPI valid; existing ignored warning is the OAuth callback's 302-only response. |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS | 263 ordered tasks; task 247 remains `PREPARED`. |
| `python3 scripts/validate-traceability.py` | repository root | 0 | PASS | Traceability validation passed. |
| `gofmt -d` on all nine reviewed Go files | repository root | 0 | PASS | No formatting diff. |
| `git diff --check` | repository root | 0 | PASS | No whitespace errors. |
| `python3 scripts/check.py` | repository root | 1 | External failure only | Traceability, task-list, docs, OpenAPI, static/security, local stack, UAT, frontend, and focused suites pass; serial full Go suite stops on untouched task-240 cleanup failure. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-247-review.md` | repository root | 0 | PASS | Final structural review-evidence validator passes after this refresh. |

## 9. Files Inspected and Staleness Fingerprints

Hashes were computed after the final verification commands and before this evidence refresh. They match the latest preparation fingerprints where applicable. The task table and open-items document were hashed as unchanged controls; neither was edited by this review.

| File | Purpose | SHA-256 |
|---|---|---|
| `backend/internal/httpapi/admin_controller.go` | Admin route policy, authorization, and mutation boundary | `94d3841f21c30bd2939e2be1ac46d8d984c68a95765e488854335bff6ed6fe3c` |
| `backend/internal/httpapi/admin_controller_test.go` | Task-specific HTTP and transaction tests | `0718be6b3969ef020da66b3256520e76cc1b8a7aa8a74b624548ed4a08236f3a` |
| `backend/internal/httpapi/router.go` | Request ID and route middleware ordering | `a98d15348d69f6fdf4d5076c2c12f203b20b10c033a81a6fcc25cd7dfacbcb48` |
| `backend/internal/httpapi/router_test.go` | Signed JWT role test helper | `26e5c52e6d0826916664c9dba1a33147baa073b7d4d15be3362ea6862698f7d0` |
| `backend/internal/repository/types.go` | Transaction and audit contracts | `5534be37a865c95390f84687ed82007e0adbca63a94fbd8c7e849ccb8cc40ac6` |
| `backend/internal/repository/errors.go` | Audit-persistence sentinel | `2cb72f6d57578da51f99866053c6e7d285d7446c21ca32231b2761a67a6b915e` |
| `backend/internal/repository/compliance_repository.go` | PostgreSQL audit transaction and sanitation | `d185aed065dd59ade5d3f7330efa5defc1e4acabd5958f2a8ed1e9c83f111f88` |
| `backend/internal/repository/postgres_repository_test.go` | PostgreSQL integration/validation tests | `ceae02b7ee90824286ed621d3f00b756a6ba6101083620a344fb48e74f19d7cd` |
| `backend/internal/repository/admin_audit_security_test.go` | Snapshot/privacy/cause-chain tests | `06aa705afc27beeee0f6de781279d3a92d46cdc36ff9ed9b2773311a00e84d39` |
| `docs/implementation/02_TASK_LIST.md` | Unchanged task status control | `a4381061f6f1e13115521b2baab83396cfb28abf52f78a6fc2e74aee83096ea8` |
| `docs/implementation/04_OPEN.md` | Unchanged coverage-exception control | `9a91d4cfe32ed9eb1f85d08dd668cb44cac3635f5a9d4dff368fdd425c22b5fa` |
| `docs/implementation/preparations/task-247.md` | Latest repair report | `58f723e8b141ce9245ba3688573ad32fb3bf912f192697786c1c6b384485ea70` |

~~~~yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The earlier rejected review was stale: its pre-repair findings and hashes were rechecked against current source and are closed by the latest repair."
  - "The dirty worktree contains unrelated Phase 08 changes; only task-247 symbols and directly shared controls were attributed to this review."
~~~~

## 10. Coverage and Exceptions

- [x] Complete focused HTTP package race/coverage profile ran.
- [x] Complete serialized repository package race/coverage profile ran.
- [x] Coverage profile paths and package measurements are recorded.
- [x] Every task-owned executable function reports 100.0% line coverage.
- [x] No task-247 coverage exception is used.

~~~~yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_paths:
  - "/tmp/task-247-http-full.cover"
  - "/tmp/task-247-repository-full.cover"
observed_package_coverage: "HTTP 88.5%; repository 93.4%"
observed_task_local_function_coverage: "100.0% for every task-owned executable function"
coverage_passed: true
~~~~

The package percentages include unrelated functions in shared packages. The acceptance measurement is the complete function-level result for the task-owned symbols listed in sections 5 and 6; all are 100.0%.

## 11. Negative and Regression Checks

- [x] Wildcard, catchall, optional, malformed, empty-segment, and duplicate-parameter route attacks are rejected.
- [x] Exact duplicate routes are rejected.
- [x] Semantic parameter aliases and static/parameter overlaps are rejected in both registration orders.
- [x] Different methods, incompatible static paths, and different segment counts remain valid.
- [x] Server-generated UUID request IDs ignore inbound `X-Request-ID` on success and error paths; concurrent requests remain correlated and unique.
- [x] Snapshot input is size-bounded, object-shaped, schema-bound, type-checked, enum-checked, and canonicalized.
- [x] Free-text names, reasons, identifiers, provider payloads, PII, secrets, nested values, unknown schemas, malformed JSON, and unsafe `Before`/`After` values are rejected.
- [x] Auth role comes from a verified signed JWT/session; client role and identity spoofing are ignored.
- [x] Middleware order is authentication, admin role, CSRF, validation, rate limit, required security audit, then handler.
- [x] Nil audit permits safe reads but mutation execution fails closed.
- [x] Mutation response serialization and status validation occur before audit persistence/transaction commit.
- [x] PostgreSQL audit persistence uses parameterized SQL, context propagation, deferred rollback, and sentinel/cause wrapping.
- [x] Audit failure, invalid snapshots, mutation errors, and response serialization failures roll back without committing the mutation.
- [x] Focused race tests and complete package race profiles report no task-247 race or goroutine leak.
- [x] No production code or task-list file was changed by this review; only this evidence document was refreshed.

The unrelated full-suite/aggregate failure is `internal/app.TestTask240CustomItemErasureIntegration`, where transactional account cleanup leaves two owner custom items in the already-dirty Phase 08 worktree. It is not attributed to task 247.

## 12. Decision

A task may be `PASSED` only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, task-local coverage is complete, and no blocking/important finding remains. Those conditions are met for task 247.

~~~~yaml
decision: "PASSED"
reason: "All task-247 acceptance criteria and audited symbols pass. The repaired route registry rejects semantic collisions independent of registration order; audit snapshots are strict, bounded, and privacy-safe; task-local executable coverage is 100%; and the prior auth, CSRF, validation, rate, request-ID, rollback, serialization, read-safety, role-spoofing, cleanup, concurrency, and envelope checks remain passing."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "No task-247 repair is required. Keep task status PREPARED until the phase orchestrator applies its normal status transition; address the unrelated task-240 integration failure in its own scope."
~~~~

## 13. Repair Context

This is a fresh independent re-review of repaired PREPARED task 247. The latest repair report was used only to identify the changed boundary; source, tests, hashes, and command results were independently checked.

- **Semantic route collisions:** `Routes` validates each safe path and rejects same-method templates whose segments can match the same request. Parameter aliases (`/:id` vs `/:name`) and static/parameter overlaps (`/search` vs `/:id`) panic in either registration order; exact duplicates remain rejected.
- **Strict audit snapshots:** `sanitizeAdminAuditSnapshot` uses explicit entity/action schemas: fixture update permits only boolean `active`/`deleted` and status codes `draft`/`published`; food update permits only fixed status codes. Unknown/free-text/nested/provider/PII/secret content and wrong types are rejected, input is capped at 4096 bytes, and persisted bytes are canonicalized.
- **Coverage:** Complete HTTP and serialized repository race profiles report 100% for every task-owned executable function; no exception was added to `docs/implementation/04_OPEN.md`.
- **Request IDs and privacy:** `serverRequestID` overwrites client correlation headers with a server UUID before gateway processing. Logs, security audit entries, errors, and mutation envelopes use that server-owned ID and do not expose raw causes or sensitive payloads.
- **Transaction boundary:** Mutation callbacks, response status checks, and JSON envelope serialization execute inside `WithMutationAudit` before audit insert and commit. Invalid response/status, mutation, snapshot, audit, and commit paths fail closed and roll back.
- **Original controls:** Verified JWT-cookie admin role enforcement, role/identity spoof resistance, auth → admin → CSRF → validation → rate → audit ordering, nil-audit safe reads, context propagation, cleanup, parameterized reads/writes, and concurrency behavior remain passing.

No production-code or task-list repair was performed in this review. The only intended write was this evidence document.

### Do Not Change

Do not weaken the server-signed JWT-cookie role check, accept client headers as identity or role, reorder auth/admin before CSRF/validation/rate/audit controls, make audit failure best-effort for mutations, remove transaction rollback cleanup, expose raw audit/database/provider errors, or broaden the task-247 scope into unrelated dirty-worktree tasks.
