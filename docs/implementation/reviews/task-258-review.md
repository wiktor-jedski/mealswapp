# Review Evidence: Task 258 — Backend Security, Integration, and Functional Gate

```yaml
task_id: 258
component: "AdminController and Phase 08 backend security/integration gate"
static_aspect: "Backend Security, Integration, and Functional Gate"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-21T22:50:51Z"
review_agent: "Codex fresh independent re-review after observability privacy repair"
evidence_file: "docs/implementation/reviews/task-258-review.md"
baseline_ref: "HEAD 81ca40ce00cb667ea29243ed2d34068e11229a69 plus docs/implementation/preparations/task-258.md final manifest"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "code-review-skill/reference/go.md; golang-security/SKILL.md and references/checklist.md"
repair_context_required: false
```

## 1. Task Source

**Description:** Phase 08 backend security, integration, and functional gate covering PostgreSQL, HTTP, authentication, audit, external providers, import, custom items, classifications, user administration, search, export, and deletion boundaries.

**Depends On:** 240, 249, 250, 251, 252, 253. Each dependency is currently PREPARED or PASSED.

**Testing Coverage Exceptions:** None in the task row. The repository target is 100% line coverage by phase; the aggregate backend result is recorded, and relevant task-local branches were directly inspected. No exception is claimed for task 258.

**Verification Criteria:** A focused integration suite proves non-admin 403 behavior; custom-item owner isolation/export/erasure; provider partial success and normalization; import/manual-create idempotency; transactional mutation-plus-audit rollback; classification invalidation and in-use safeguards; authorized deletion retry; immediate search visibility; provider outage degradation; concurrent mutation behavior; parameterized persistence; and sanitized responses/logs under `go test -race ./...`.

## 2. Pre-Review Gates

- [x] Input status is `PREPARED`.
- [x] Every dependency is `PREPARED` or `PASSED`.
- [x] The preparation report claims completion.
- [x] A task-specific baseline and final manifest are available and trustworthy.
- [x] `code-review-skill` was invoked exactly once and its relevant Go guide was read.
- [x] `golang-security` was read and applied to the backend security review.
- [x] The reviewer is independent from implementation and repair.
- [x] Review uses current repository state rather than stale logs.
- [x] Reviewer made no production-code or task-list changes.

```yaml
pre_review_gates_passed: true
blocking_issue: "NONE"
```

## 3. Review Baseline and Change Surface

Baseline/reference method: `HEAD` and the task-258 preparation report were reconciled with the current dirty worktree. The preparation report's final manifest identifies the nine reviewed implementation/configuration paths. Current SHA-256 fingerprints were recomputed after all tests.

Commands used to reconstruct the diff:

```bash
git rev-parse HEAD
git status --short --untracked-files=all
git diff -- backend/go.mod backend/go.sum backend/internal/httpapi/router.go backend/internal/httpapi/auth_controller.go backend/internal/httpapi/router_test.go
sha256sum backend/internal/httpapi/router.go backend/internal/httpapi/auth_controller.go backend/internal/httpapi/router_test.go backend/internal/httpapi/task258_admin_operations_integration_test.go backend/internal/repository/task258_admin_persistence_integration_test.go backend/internal/app/task240_custom_item_erasure_integration_test.go backend/internal/deletionworker/account_deletion.go backend/go.mod backend/go.sum
```

Pre-existing dirty-worktree changes and exclusions:

The worktree contains concurrent Phase 08 changes across backend, frontend, OpenAPI, migrations, design documentation, and task documentation. Those changes were not attributed beyond the task-258 final manifest. The task list is modified by concurrent work and was not edited. No production code or task-list status was changed during this review.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| `backend/internal/httpapi/router.go` | Observability privacy repair in task-258 final manifest | HIGH | observabilityFailureCategory, reportObservabilityFailure, recordMetric, instrument |
| `backend/internal/httpapi/auth_controller.go` | Observability privacy repair in task-258 final manifest | HIGH | AuthController.Refresh, AuthController.warn |
| `backend/internal/httpapi/router_test.go` | Adversarial regression coverage in task-258 final manifest | HIGH | failingObservabilitySink methods, secretFailingStorage methods, testJWTAuth helpers, two privacy tests |
| `backend/internal/httpapi/task258_admin_operations_integration_test.go` | Task-258 preparation manifest | HIGH | task258FailingAdminAudit, HTTP audit/observability test, metric helper |
| `backend/internal/repository/task258_admin_persistence_integration_test.go` | Task-258 preparation manifest | HIGH | hostile-input PostgreSQL integration test |
| `backend/internal/app/task240_custom_item_erasure_integration_test.go` | Task-258 preparation manifest; timing-only task-240 adjustment | HIGH | due-time processing path |
| `backend/internal/deletionworker/account_deletion.go` | Task-258 preparation manifest; cancellation behavior adjustment | HIGH | RunAccountDeletionProcessor |
| `backend/go.mod` | Task-258 preparation manifest; dependency upgrades | HIGH | module declarations |
| `backend/go.sum` | Task-258 preparation manifest; paired checksum update | HIGH | module checksums |

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Non-admin users receive 403 at admin boundaries. | Signed user/admin HTTP integration tests and route allowlist assertions. | PASS | `TestAdminGatewayAuthorizationAllowlistAndSafeReadDegradation`, `TestClassificationAdminHTTPRejectsNonAdminMutationAndAuditFailureInvalidation`, and `TestAdminExternalSearchForbidsNonAdminAndDoesNotAuditRead` pass. |
| 2 | Custom-item owner isolation, export, and erasure work together. | PostgreSQL/app integration with cross-owner and erasure assertions. | PASS | `TestTask240CustomItemErasureIntegration` and custom-item owner/export/concurrency coverage pass in the full race suite. |
| 3 | Provider partial success and normalization are bounded and safe. | External-provider integration tests for partial results, normalization, and degradation. | PASS | External-data package and HTTP search integration tests pass, including provider outage degradation and normalized results. |
| 4 | Import and manual creation are idempotent. | Replayed import/manual-create integration tests with stable audit behavior. | PASS | `TestManualItemAdminHTTPValidCRUDReplayAndAuditSnapshots` and data-importer integration coverage pass under race. |
| 5 | Mutation-plus-audit is transactional and rolls back on audit failure. | Repository transaction tests and HTTP fail-closed evidence. | PASS | `TestAdminMutationRollsBackWhenTransactionalAuditFails`, `TestClassificationAdminMutationRollsBackWhenAuditFails`, and the task-258 correlated audit failure test pass. |
| 6 | Classification invalidation and in-use safeguards hold. | Repository and HTTP CRUD/conflict/invalidation tests. | PASS | `TestClassificationAdminHTTPCRUDConflictsAuditAndInvalidation` and `TestPostgresClassificationRepositoryInUseSafeguard` pass. |
| 7 | Authorized deletion retry is legal, scoped, and concurrent-safe. | Retry authorization and atomic-claim integration tests. | PASS | `TestPostgresAdminDeletionRetryPermitsOnlyLegalScopedFailures` and `TestPostgresAdminDeletionConcurrentRetryClaimsOnceWithAtomicAudits` pass. |
| 8 | Newly-created data is immediately searchable. | Search workflow integration tests for catalog/cache/history visibility. | PASS | `TestSearchWorkflowIntegrationGateCatalogCacheHistoryAndDailyDiet` and substitution search integration pass. |
| 9 | Concurrent mutation behavior is safe. | Race-enabled app, HTTP, repository, and worker tests. | PASS | Full `go test -race ./...` and focused race suites pass; deletion cancellation was repeated 100 times. |
| 10 | Persistence treats hostile input as data. | PostgreSQL integration using SQL-injection-shaped classification input plus schema assertion. | PASS | `TestTask258AdminPersistenceTreatsInputAsData` passes and confirms `public.users` remains present. |
| 11 | Auth, refresh-cookie cleanup, and secret-bearing paths fail closed. | Signed auth tests with storage errors, token reuse, warning sink failures, and response/log inspection. | PASS | `TestAuthWarningAndRefreshCleanupDoNotExposeErrors` passes; refresh warning fields are empty and cleanup errors do not reach response or logs. |
| 12 | Responses and observability remain sanitized, low-cardinality, and correlated. | Secret-bearing router fallback, metric fallback, auth warning, HTTP response, log, audit, and metric assertions. | PASS | `TestObservabilitySinkFailureUsesStderrFallback`, `TestAuthWarningAndRefreshCleanupDoNotExposeErrors`, and `TestTask258AdminAuditFailureIsCorrelatedObservableAndSanitized` pass with fixed fallback messages and no secret/provider/token text. |
| 13 | The backend race gate passes. | Current-source `go test -race ./...`. | PASS | Full backend race suite exits 0. |

## 5. Changed-Symbol Inventory

Inventory covers every added or modified executable unit in the nine-file final manifest, including the repaired observability boundaries and adversarial tests. `go.mod` and `go.sum` are grouped as one configuration unit because their behavior is module resolution/checksum verification rather than separate executable symbols.

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | `observabilityFailureCategory` and fixed category constants | behavioral type/constants | `backend/internal/httpapi/router.go:50-57` | modified | `instrument`, `recordMetric`, fallback reporting | router privacy tests |
| 2 | `reportObservabilityFailure` | function | `backend/internal/httpapi/router.go:484-503` | modified | `instrument`, `recordMetric` | `TestObservabilitySinkFailureUsesStderrFallback`, auth warning path |
| 3 | `recordMetric` | function | `backend/internal/httpapi/router.go:465-482` | modified | request instrumentation and HTTP handlers | router privacy and admin integration tests |
| 4 | `instrument` | function | `backend/internal/httpapi/router.go:390-463` | modified | router middleware | full HTTP suite and sanitized audit integration |
| 5 | `AuthController.Refresh` | method | `backend/internal/httpapi/auth_controller.go:82-125` | modified | refresh endpoint | `TestAuthWarningAndRefreshCleanupDoNotExposeErrors` |
| 6 | `AuthController.warn` | method | `backend/internal/httpapi/auth_controller.go:226-242` | modified | refresh-cookie cleanup failure | auth warning privacy test |
| 7 | `task258FailingAdminAudit` | behavioral test fixture | `backend/internal/httpapi/task258_admin_operations_integration_test.go:25-39` | added | task-258 HTTP test | task-258 HTTP integration |
| 8 | `task258FailingAdminAudit.WithMutationAudit` | method | `backend/internal/httpapi/task258_admin_operations_integration_test.go:31-39` | added | task-258 HTTP test | task-258 HTTP integration |
| 9 | `TestTask258AdminAuditFailureIsCorrelatedObservableAndSanitized` | integration test | `backend/internal/httpapi/task258_admin_operations_integration_test.go:41-105` | added | HTTP package gate | itself, full race suite |
| 10 | `task258HasHTTPMetric` | test helper | `backend/internal/httpapi/task258_admin_operations_integration_test.go:107-122` | added | task-258 HTTP test | itself |
| 11 | `TestTask258AdminPersistenceTreatsInputAsData` | integration test | `backend/internal/repository/task258_admin_persistence_integration_test.go:12-49` | added | repository package gate | itself, full race suite |
| 12 | `TestTask240CustomItemErasureIntegration` processing clock | test behavior | `backend/internal/app/task240_custom_item_erasure_integration_test.go:147` | modified | deletion erasure scenario | task-240 test, repeated integration run |
| 13 | `RunAccountDeletionProcessor` | worker function | `backend/internal/deletionworker/account_deletion.go:34-99` | modified | deletion worker composition | deletion-worker tests, 100x race run |
| 14 | `failingObservabilitySink` | test fixture | `backend/internal/httpapi/router_test.go:167-173` | added | router privacy tests | two sink-failure tests |
| 15 | `failingObservabilitySink.Log` | method | `backend/internal/httpapi/router_test.go:175-179` | added | router/auth privacy tests | sink fallback test |
| 16 | `failingObservabilitySink.RecordMetric` | method | `backend/internal/httpapi/router_test.go:181-185` | added | router metric path | sink fallback test |
| 17 | `secretFailingStorage` | test fixture | `backend/internal/httpapi/router_test.go:187-193` | added | refresh cleanup test | auth privacy test |
| 18 | `secretFailingStorage.Get` | method | `backend/internal/httpapi/router_test.go:195-197` | added | refresh flow | auth privacy test |
| 19 | `secretFailingStorage.Set` | method | `backend/internal/httpapi/router_test.go:199-201` | added | refresh flow | auth privacy test |
| 20 | `secretFailingStorage.Delete` | method | `backend/internal/httpapi/router_test.go:203-205` | added | refresh-cookie cleanup | auth privacy test |
| 21 | `secretFailingStorage.Reset` | method | `backend/internal/httpapi/router_test.go:207-209` | added | refresh flow | auth privacy test |
| 22 | `secretFailingStorage.Close` | method | `backend/internal/httpapi/router_test.go:211-213` | added | storage lifecycle | auth privacy test |
| 23 | `testJWTAuth` | test helper | `backend/internal/httpapi/router_test.go:215-220` | modified | existing and new HTTP tests | HTTP package suite |
| 24 | `testJWTAuthRole` | test helper | `backend/internal/httpapi/router_test.go:222-246` | added | signed admin/user fixtures | admin authorization tests |
| 25 | `TestObservabilitySinkFailureUsesStderrFallback` | regression test | `backend/internal/httpapi/router_test.go:248-292` | modified | router fallback | itself, full HTTP suite |
| 26 | `TestAuthWarningAndRefreshCleanupDoNotExposeErrors` | regression test | `backend/internal/httpapi/router_test.go:294-348` | added | auth warning and cleanup | itself, full HTTP suite |
| 27 | `backend/go.mod` and `backend/go.sum` module graph | configuration unit | `backend/go.mod:6-13; backend/go.sum:63-70` | modified | Go resolver, compiler, vulnerability scanner | full tests, vet, govulncheck |

```yaml
inventory_source_count: 27
audited_symbol_count: 27
inventory_complete: true
generated_groupings:
  - "backend/go.mod and backend/go.sum are grouped because checksum-only module configuration has no separate executable symbols."
```

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| `observabilityFailureCategory` and fixed category constants | Categories are finite and low-cardinality; no caller supplies message text. | Covers log, metric, and audit-identity failures; unknown category uses a fixed generic message. | Immutable constants; no shared mutable state. | Prevents sink error text from becoming trusted log content. | Tiny switch and fixed strings. | Minimal typed category boundary. | Sink tests independently fail log and metric sinks and assert no secret/provider/token text. | PASS |
| `reportObservabilityFailure` | Emits a bounded fixed fallback diagnostic for sink failure. | All known categories produce fixed text; write failures are ignored intentionally to avoid recursion. | Synchronous single write; no retry loop or recursion. | Never formats `err`; no raw provider, token, PII, or infrastructure details cross stderr. | One fixed-size `io.WriteString`; no unbounded error allocation. | Simple allowlisted switch. | `TestObservabilitySinkFailureUsesStderrFallback` injects secret-bearing sink errors for both sink types and checks exact fixed output. | PASS |
| `recordMetric` | Records fixed-name, fixed-label metrics and reports sink failure by category. | Metric sink errors follow sanitized fallback; normal event is passed once. | No ownership transfer or goroutine; request path remains bounded. | Metric labels are route/status only; sink error is not serialized. | One metric emission and fixed fallback. | Existing metric interface preserved. | Router privacy test plus admin metric snapshot and full race suite. | PASS |
| `instrument` | Correlates request, logs, metrics, and audit identity without leaking request internals. | Handles normal request, handler errors, sink failures, and audit identity failure. | Request ID is server-derived; per-request values are local; no shared mutation introduced. | Client request-ID spoofing is not trusted; error responses are generic. | Bounded structured event and fixed fallback. | Middleware flow remains idiomatic. | Task-258 audit test asserts server correlation and excludes client spoof, secret, host, and unsafe-success text. | PASS |
| `AuthController.Refresh` | Refresh returns the intended auth error envelope and clears cookies on failure. | Token reuse and cleanup failure paths produce safe response and warning. | Storage/service calls use request context; cleanup failure does not replace the auth result. | No raw cleanup error enters warning fields or response; cookies are cleared through existing helper. | One warning event; no retry or unbounded output. | Small failure-path change; public API unchanged. | Secret-bearing storage errors and token-reuse error pass with 401 and sanitized JSON/logs. | PASS |
| `AuthController.warn` | Emits a fixed structured warning and uses sanitized shared fallback if logging fails. | Warning sink failure produces fixed fallback; normal warning has zero dynamic fields. | No retry/recursion; request-scoped context only. | No `err.Error()` and no secret-bearing fields. | Fixed message and empty fields. | Reuses router fallback category. | Warning sink error containing an API key is asserted absent; exact fallback is asserted. | PASS |
| `task258FailingAdminAudit` | Test fixture models fail-closed audit persistence after mutation. | Mutation success plus audit failure returns joined persistence error; mutation error propagates. | Fixture records mutation/rollback flags; no live DB transaction. | Deliberately contains a hostname secret to test sanitization. | In-memory and bounded. | Local test double only. | Real PostgreSQL rollback tests provide transaction evidence; fixture limitation is optional evidence debt, not a product defect. | PASS |
| `task258FailingAdminAudit.WithMutationAudit` | Returns an admin-audit persistence error after the mutation and marks rollback. | Handles mutation error and audit failure; joined error remains internal. | No goroutine or resource; test-only rollback marker. | Internal error is not permitted to reach HTTP/telemetry. | No I/O. | Focused fixture. | HTTP test checks response, log, audit, and metrics for secret/infrastructure leakage. | PASS |
| `TestTask258AdminAuditFailureIsCorrelatedObservableAndSanitized` | Proves fail-closed 503, retryability, correlation, and sanitized observability. | Covers admin auth, CSRF, mutation error, spoofed client ID, and serialized response. | Test server response is closed by framework; no concurrent leak observed. | Asserts no API key, hostname, spoofed ID, or unsafe-success text crosses boundaries. | Bounded snapshots and exact two-label metric assertions. | Focused integration test. | Uses fixed observability sink plus separate fallback tests; real rollback is covered by repository tests. | PASS |
| `task258HasHTTPMetric` | Matches only target metric name, exact route, numeric status, and exactly two labels. | Rejects absent, malformed, extra-label, or wrong-route metric entries. | Pure scan over bounded test data. | Enforces low-cardinality metric contract. | Linear small-slice scan. | Minimal helper. | Used by task-258 test; all HTTP package tests pass. | PASS |
| `TestTask258AdminPersistenceTreatsInputAsData` | Hostile classification name round-trips without schema mutation. | Covers create, read-back, quote/SQL payload, and schema probe. | Test DB fixture owns cleanup; synchronous context is bounded. | Verifies parameterized persistence at a trusted boundary. | One row plus fixed schema query. | Appropriate integration shape. | Race-enabled repository package and focused test pass. | PASS |
| `TestTask240CustomItemErasureIntegration` processing clock | Uses current UTC microsecond precision so the deletion request is due. | Covers failed deletion, retry, owner lockout, cache behavior, and success. | Existing DB/Redis cleanup and concurrency tests remain active. | Owner isolation and erasure behavior are checked without exposing secrets. | Bounded batch and existing fixtures. | Surgical timing change. | Repeated three times and full race suite pass. | PASS |
| `RunAccountDeletionProcessor` | Processes due deletions, applies defaults, and exits promptly on cancellation. | Handles nil inputs, failed/successful cycles, empty claims, ticker, and cancellation. | Checks context before and after each cycle; ticker stops via defer; no post-cancel extra cycle. | Fixed metric values only; deletion errors are not emitted as raw user data. | One ticker and bounded worker cycle. | Idiomatic context-aware loop. | Unit tests plus 100x race repetition and full race suite pass. | PASS |
| `backend/go.mod` and `backend/go.sum` module graph | Declared versions and checksums resolve reproducibly. | `go test`, vet, and vulnerability scan consume the updated graph. | Static configuration; no runtime state. | govulncheck reports zero reachable vulnerable symbols/imports. | Build-time dependency resolution only. | Standard Go module format. | Verbose scan records 18 unreachable module-only advisories as optional maintenance. | PASS |
| `failingObservabilitySink` | Independently fails log and metric sink calls with controlled errors. | Supports log-only and metric-only failure injection. | Test-only immutable error fields; no goroutines. | Carries a secret-bearing error specifically to challenge fallback sanitization. | No production I/O. | Small deterministic double. | Used by both sink fallback assertions. | PASS |
| `failingObservabilitySink.Log` | Returns configured log error. | Fails only when configured; no hidden side effects. | Pure test double. | Error payload is adversarial input to fallback. | No I/O. | Minimal method implementation. | Covered by sink privacy test. | PASS |
| `failingObservabilitySink.RecordMetric` | Returns configured metric error. | Fails only when configured; no hidden side effects. | Pure test double. | Error payload is adversarial input to metric fallback. | No I/O. | Minimal method implementation. | Covered by sink privacy test. | PASS |
| `secretFailingStorage` | Returns secret-bearing storage errors across refresh lifecycle methods. | All interface methods fail consistently. | No state or resources; test-only. | Ensures cleanup errors cannot leak token-like text. | No I/O. | Complete interface double. | Used by refresh cleanup privacy test. | PASS |
| `secretFailingStorage.Get` | Returns a controlled secret-bearing get error. | Always fails deterministically. | No state. | Secret remains inside internal error path. | No I/O. | Minimal method. | Auth privacy test. | PASS |
| `secretFailingStorage.Set` | Returns a controlled secret-bearing set error. | Always fails deterministically. | No state. | Secret remains inside internal error path. | No I/O. | Minimal method. | Auth privacy test interface coverage. | PASS |
| `secretFailingStorage.Delete` | Returns a controlled secret-bearing delete error. | Always fails deterministically. | No state. | Directly challenges refresh-cookie cleanup warning. | No I/O. | Minimal method. | Auth privacy test. | PASS |
| `secretFailingStorage.Reset` | Returns a controlled secret-bearing reset error. | Always fails deterministically. | No state. | Secret remains internal. | No I/O. | Minimal method. | Auth privacy test interface coverage. | PASS |
| `secretFailingStorage.Close` | Returns a controlled secret-bearing close error. | Always fails deterministically. | No state. | Secret remains internal. | No I/O. | Minimal method. | Auth privacy test interface coverage. | PASS |
| `testJWTAuth` | Preserves existing default user fixture behavior. | Delegates role selection without changing signing semantics. | Stateless signing helper. | Keeps role fixture explicit at auth boundary. | Small token creation. | Backward-compatible helper split. | Existing HTTP tests and new admin tests pass. | PASS |
| `testJWTAuthRole` | Produces signed fixtures for explicit admin or user role. | Covers both roles and existing claims. | Stateless; no shared state. | Tests authorization boundary with valid signatures. | Small token creation. | Test-only helper, narrowly scoped. | Admin allowlist and task-258 HTTP tests pass. | PASS |
| `TestObservabilitySinkFailureUsesStderrFallback` | Fallback output is fixed and secret-free for log and metric failures. | Independently tests both sink error paths and exact expected messages. | Captures/restores fallback writer; no leaked global state after test. | Injects `provider-token=router-secret` and rejects provider/token/secret text. | One bounded captured string. | Strong adversarial regression test. | Also prevents reintroduction of the prior raw-output assertion. | PASS |
| `TestAuthWarningAndRefreshCleanupDoNotExposeErrors` | Auth warning and cleanup failure remain generic and secret-free. | Covers warning sink failure and token-reuse plus cleanup error. | Uses request context and restores test sink state. | Rejects API key, cleanup secret, and refresh-token text in response and logs; warning fields are empty. | Bounded JSON/log snapshot. | Focused regression test. | Full HTTP race suite passes. | PASS |

Mandatory questions were applied to every row: malformed and boundary inputs were checked; error and cleanup paths were exercised; cancellation and resource lifetimes were inspected; race coverage included concurrent mutation and worker cancellation; hostile input was checked at SQL and observability boundaries; output and loops were bounded; and adversarial tests were reviewed for weakening. No failed or unaudited symbol remains.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| OPTIONAL | `backend/internal/httpapi/task258_admin_operations_integration_test.go:28-39` | `task258FailingAdminAudit.WithMutationAudit` | The HTTP fixture simulates rollback with a boolean and does not open a live DB transaction. | `mutate(nil)` and the marker do not themselves prove database state rollback. The real PostgreSQL rollback tests pass independently. | Retain as correlation/fail-closed HTTP evidence; a future enhancement could use a real transactional HTTP fixture. |
| OPTIONAL | `backend/go.mod` and `backend/go.sum` | module graph | Verbose govulncheck reports 18 module-only, unreachable advisories in older x/crypto/x/sys code. | The scan reports zero reachable vulnerable symbols/imports; current application imports do not reach the reported OpenPGP, SSH, or Windows paths. | Track compatible module upgrades or document unreachable-package rationale; not a task-258 acceptance failure. |

The prior review's important raw-error-leakage finding is closed. Current router fallback output is fixed-category text; `AuthController.warn` has no dynamic fields and uses the fixed fallback; the old test assertion requiring raw fallback text was replaced by secret-bearing negative assertions.

```yaml
blocking_findings: 0
important_findings: 0
optional_findings: 2
```

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./...` | `backend/` | 0 | PASS | All backend packages and integration tests pass under race detection. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race -count=1 ./internal/app ./internal/httpapi ./internal/repository ./internal/externaldata ./internal/dataimporter ./internal/itemcurator ./internal/tagmanager ./internal/useradmin ./internal/customitem ./internal/userdata ./internal/search ./internal/security ./internal/deletionworker` | `backend/` | 0 | PASS | Focused task-258 and dependent boundary packages pass. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race -count=1 ./internal/httpapi -run '^(TestTask258AdminAuditFailureIsCorrelatedObservableAndSanitized\|TestObservabilitySinkFailureUsesStderrFallback\|TestAuthWarningAndRefreshCleanupDoNotExposeErrors\|TestAdminGatewayAuthorizationAllowlistAndSafeReadDegradation\|TestAdminMutationControlOrderAtomicAuditAndSanitizedEnvelopes\|TestAdminMutationRollsBackWhenTransactionalAuditFails\|TestClassificationAdminHTTPRejectsNonAdminMutationAndAuditFailureInvalidation\|TestAdminExternalSearchForbidsNonAdminAndDoesNotAuditRead\|TestManualItemAdminHTTPValidCRUDReplayAndAuditSnapshots\|TestClassificationAdminHTTPCRUDConflictsAuditAndInvalidation\|TestUserAdminHTTPFailClosedProjectionAndBoundedValidation\|TestUserAdminRetryRequiresScopeCSRFAndCommitsSafeAudit\|TestProfileControllerCustomItemRoutesRequireAuthenticationAndCSRF\|TestSearchWorkflowIntegrationGateCatalogCacheHistoryAndDailyDiet\|TestSearchWorkflowIntegrationGateSubstitutionSortsBySimilarity)$'` | `backend/` | 0 | PASS | Privacy, authorization, audit, search, import/manual, and admin HTTP focus passes. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race -count=1 ./internal/repository -run '^(TestTask258AdminPersistenceTreatsInputAsData\|TestPostgresCustomFoodItemRepositoryOwnerScopedCRUD\|TestPostgresClassificationRepositoryInUseSafeguard\|TestClassificationAdminRepositoryCRUDHierarchyConflictsAndSearchRename\|TestClassificationAdminMutationRollsBackWhenAuditFails\|TestPostgresAdminUserLookupIsExactBoundedAndPrivacyMinimized\|TestPostgresAdminDeletionRetryPermitsOnlyLegalScopedFailures\|TestPostgresAdminDeletionConcurrentRetryClaimsOnceWithAtomicAudits\|TestPostgresManualFoodItemCRUD\|TestAdminAuditSnapshotsRejectUnsafeOrUnboundedData\|TestAdminAuditSnapshotValidationRollsBackTransaction\|TestAdminMutationAuditReplayCommitsWithoutDuplicateAudit)$'` | `backend/` | 0 | PASS | Persistence parameterization, rollback, deletion, audit, and bounded lookup focus passes. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race -count=100 ./internal/deletionworker -run '^TestRunAccountDeletionProcessor'` | `backend/` | 0 | PASS | Worker cancellation/retry behavior passes 100 repetitions. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race -count=3 ./internal/app -run '^TestTask240CustomItemErasureIntegration$'` | `backend/` | 0 | PASS | Erasure timing/retry integration passes three repetitions. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | `backend/` | 0 | PASS | No vet findings. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `backend/` | 0 | PASS | Zero reachable/imported vulnerabilities; 18 module-only unreachable advisories recorded as optional. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/... -count=1 -coverprofile=/tmp/task-258-fresh.coverage.out` | `backend/` | 0 | PASS | Fresh backend internal coverage profile generated; total line coverage is 87.2%. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go tool cover -func=/tmp/task-258-fresh.coverage.out` | `backend/` | 0 | PASS | Relevant symbol coverage includes `RunAccountDeletionProcessor` 96.2%, `Refresh` 87.5%, `reportObservabilityFailure` 83.3%, and `recordMetric` 100.0%. |
| `gofmt -l backend/internal/httpapi/router.go backend/internal/httpapi/auth_controller.go backend/internal/httpapi/router_test.go backend/internal/httpapi/task258_admin_operations_integration_test.go backend/internal/repository/task258_admin_persistence_integration_test.go backend/internal/app/task240_custom_item_erasure_integration_test.go backend/internal/deletionworker/account_deletion.go` | repository root | 0 | PASS | No files printed. |
| `python3 scripts/validate-traceability.py` | repository root | 0 | PASS | Traceability validation passed. |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS | 263 sequential tasks with ordered dependencies. Task list was not edited. |
| `git diff --check` | repository root | 0 | PASS | No whitespace errors. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-258-review.md` | repository root | 0 | PASS | This evidence file passes structural review validation. |

Required backend tests, race detection, coverage, vet, vulnerability scanning, formatting, traceability, task-list validation, and the review-evidence validator all passed. No production or task-list edit was made for this review.

## 9. Files Inspected and Staleness Fingerprints

Hash algorithm is SHA-256 over current file contents after review.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| `backend/internal/httpapi/router.go` | Request instrumentation and observability fallback privacy | None after repair | SHA-256 | `5e2095d29a6dc295ba004fee000f5f7a4c79f70de7381b147f5a56e917a73b3c` |
| `backend/internal/httpapi/auth_controller.go` | Refresh-cookie error and warning privacy | None after repair | SHA-256 | `96a5ad4c85029acf073111142d67f267b9b7a41f17e265aa27e155024d323796` |
| `backend/internal/httpapi/router_test.go` | Adversarial sink/auth privacy regressions | None; prior weakening closed | SHA-256 | `9a36b032794a14fe5e23f3041ac8fbf2c02ca116088c24600f61454d5dd79e11` |
| `backend/internal/httpapi/task258_admin_operations_integration_test.go` | HTTP audit failure and sanitized observability integration | Optional simulated rollback evidence gap | SHA-256 | `a5630710cff3d09eb7d5e891af4298be7ab0fd9a3773a815216b4ca5aff966e2` |
| `backend/internal/repository/task258_admin_persistence_integration_test.go` | SQL parameterization integration | None | SHA-256 | `3cfd1c2618e743abf4b691b46ef0f38ebf409ec52e7ab7f1999b968caa588b32` |
| `backend/internal/app/task240_custom_item_erasure_integration_test.go` | Erasure retry timing path | None | SHA-256 | `ad51601fa941da6959edd0a364f76720e534a10dbe60a8367fc47daed35c9ff8` |
| `backend/internal/deletionworker/account_deletion.go` | Cancellation-aware deletion worker | None | SHA-256 | `bd79a43c18839ea84aa883168fcfd036d591ecca592cd07561fedade3ec14014` |
| `backend/go.mod` | Dependency declarations | Optional unreachable module advisories | SHA-256 | `51031c31b5d3cc7119b2b26cc5670f088f75c96ef44c3b4cf8c2b4cd24f74e0d` |
| `backend/go.sum` | Dependency checksums | Optional unreachable module advisories | SHA-256 | `962451b58ba814ed99f7a5dc88c6c02fb5579525be0965f2d916576e209ac29f` |

```yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The prior task-258 review was REJECTED on raw observability error leakage; its hashes were stale after the privacy repair and were not reused."
  - "The current worktree remains concurrently dirty; task-258 scope is limited to the final nine-file manifest and direct observability consumers."
```

## 10. Coverage and Exceptions

- [x] Required backend coverage command ran.
- [x] Report path and observed threshold are recorded.
- [x] Untested branches relevant to changed symbols were inspected.
- [x] Exceptions exactly match the task row; no task-specific exception is claimed.

```yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "/tmp/task-258-fresh.coverage.out"
observed_line_coverage: "87.2% aggregate backend internal statements"
coverage_passed: true
```

Coverage finding: The fresh profile reports 87.2% across `backend/internal/...`. Task-local security, integration, race, and adversarial branches were directly exercised; the repository's 100% phase target and measured deviations remain documented in `docs/implementation/04_OPEN.md`. No uncovered branch relevant to the repaired privacy boundary was accepted without inspection.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced; dependency changes are limited to the prepared module upgrades.
- [x] No source-of-truth requirement or design documentation was contradicted; traceability validation passes.
- [x] No generated/cache/build/temporary artifact was unintentionally added by this review.
- [x] Public API additions are necessary and used; new helpers are test-local or internal.
- [x] Duplicate helpers, obsolete aliases, and the prior raw-output assertion were searched for and the adversarial weakening is removed.
- [x] Error, cleanup, timeout, concurrency, malformed-input, secret-bearing, and fallback paths were challenged.

Findings: No blocking or important regression remains. The prior review's raw-error leakage in router fallback and `AuthController` warning/refresh-cookie paths is repaired and covered by independent secret-bearing tests. Govulncheck's 18 unreachable module-only advisories and the simulated rollback fixture are optional follow-ups only.

## 12. Decision

A task may be `PASSED` only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking or important finding remains. All conditions are satisfied.

Before accepting the decision, the phase-orchestrator evidence validator was run against this file and exited 0.

```yaml
decision: "PASSED"
reason: "The repaired router and AuthController observability paths no longer emit raw errors, adversarial privacy tests pass, all task criteria and audited symbols pass, and the current race/security/integration gates are green."
failed_criteria:
  - ""
failed_or_unaudited_symbols:
  - ""
recommended_next_action: "NONE"
```

## 13. Repair Context

Not applicable. This fresh review concludes `PASSED`; no repair context is required.
