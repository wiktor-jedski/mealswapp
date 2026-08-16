# Task 258 preparation — Phase 08 backend security, integration, and functional gate

## Outcome and scope

- Task: 258, `ARCH-009: AdminController`.
- Fixed Git reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Task row status observed: `OPEN`; dependencies 240, 249, 250, 251, 252, and 253 were `PASSED` when preparation began.
- `docs/implementation/02_TASK_LIST.md` was not edited. Current SHA-256 after preserving concurrent changes: `0edb3e8e4ba15356f98cb4c04bd64ab08ef5ec23d521bf565940e046d5ddab4a`.
- The preparation is restricted to the Phase 08 backend gate, two operational race fixes found by that gate, one reachable normalization dependency vulnerability found by the required security scan, and the task-258 review repair for bounded observability fallbacks. No frontend, OpenAPI, migration, task-status, or later task-260 observability implementation was added.

The worktree already contained extensive concurrent Phase 08 changes and untracked task-240 files. Those changes were preserved. The task-240 review hashes provide the best available baseline for the two shared untracked files changed here:

| Shared path | Pre-task SHA-256 from task-240 review | Task-258 change |
|---|---|---|
| `backend/internal/app/task240_custom_item_erasure_integration_test.go` | `7e6a0af9cbe2e17937e3cf6b85f5d874820f08f8a4401d132fac117c1c998948` | Replace a noon-fixed processing lease with current UTC time truncated to PostgreSQL microsecond precision. |
| `backend/internal/deletionworker/account_deletion.go` | `4474c633b2cc1c4724cbf3ef2df980a95429df3b68af6a8a7f5a48c598b78467` | Stop immediately when cancellation occurs during a successful processing cycle instead of racing the next ticker event. |

## Sources inspected

- Task 258 and dependency rows in `docs/implementation/02_TASK_LIST.md`.
- `docs/design/DESIGN-008.md`: private export/erasure composition, write lockout, cache-purge retry, and completion behavior.
- `docs/design/DESIGN-009.md`: admin authorization, external curation, transactional mutation-plus-audit, classification in-use safeguards, and restricted user actions.
- `docs/design/DESIGN-012.md`: provider partial success, deadlines/retries, normalization, safe warnings, and no direct provider persistence.
- `docs/design/DESIGN-013.md`: parameterized SQL, server-derived identity, input normalization, CSRF/rate controls, and fail-closed security-sensitive audit.
- `docs/design/DESIGN-014.md`: structured request logs, bounded metrics, request correlation, dependency health, and operational backpressure expectations.
- `docs/design/DESIGN-015.md`: auditable erasure transitions, immediate active-data removal, retries, and privacy-minimized receipts.
- `docs/design/DESIGN-017.md`: sanitized server errors, generic dependency envelopes, retry metadata, and server-error logging.
- `docs/requirements/01_SOFT_REQ_SPEC.md`: SW-REQ-043, SW-REQ-054 through SW-REQ-057, SW-REQ-072, SW-REQ-073, SW-REQ-084, and SW-REQ-090.
- Preparation and review evidence for dependency tasks 240 and 247-253, plus the callers, repositories, SQL files, HTTP gateway, provider orchestrator, normalization boundary, and deletion worker exercised by the gate.

## Prepared behavior and operational safeguards

### Audit-safe administration, correlation, metrics, and logging

`TestTask258AdminAuditFailureIsCorrelatedObservableAndSanitized` composes the real Fiber gateway middleware with verified admin JWT cookies, CSRF, rate controls, the admin transaction coordinator contract, security audit, structured logging, and metrics. It proves:

- an admin mutation whose audit persistence fails returns retryable `503 dependency_unavailable`, never the uncommitted success body;
- the transaction boundary is treated as rolled back and the response waits for the audit outcome;
- the client-supplied request ID is rejected and one server-generated request ID correlates the admin audit entry, security audit, safe response, and error-level request log;
- `http_response_total` and `http_error_total` contain only the fixed route template and numeric status labels;
- provider payload text, an internal database hostname, the spoofed trace ID, and uncommitted response data do not enter the response, logs, or metrics.

This is focused task-258 integration coverage. Task 260 remains responsible for the broader provider/admin outcome metric inventory and load fixtures.

### Parameterized PostgreSQL administration

`TestTask258AdminPersistenceTreatsInputAsData` sends a SQL-control-string classification name through `PostgresClassificationRepository.Create`, reloads the exact value, and confirms `public.users` still exists. The test exercises the live PostgreSQL repository boundary and proves the admin value is passed as data rather than executable SQL.

### Erasure lease and scheduler cancellation safeguards

- `TestTask240CustomItemErasureIntegration` previously used `2026-07-21 12:00 UTC` as a processing time. After that instant, the real context deadline was expired before account erasure, making the test falsely report retained private items. It now uses current UTC time at PostgreSQL precision. The original full API/PostgreSQL/Redis erasure workflow passes three consecutive runs and under the complete race suite.
- `RunAccountDeletionProcessor` previously returned `true` after a successful cycle even when that cycle canceled its context. If the ticker was simultaneously ready, the select could run an extra deletion cycle and emit duplicate completion/request metrics. The loop now returns `ctx.Err() == nil` after both failed and successful cycles. The regression passes 100 consecutive race-enabled runs.

### Reachable normalization vulnerability

The first `govulncheck` run found reachable `GO-2026-5970` in `golang.org/x/text v0.29.0`, including a direct trace through `security.normalizeVisibleText`. The dependency was upgraded to fixed `golang.org/x/text v0.39.0`; its required `golang.org/x/sync` version is `v0.21.0`. `go mod tidy` retained concurrent direct dependencies and normalized the module graph. The final scan reports zero called or imported vulnerabilities.

### Review repair: bounded observability fallback diagnostics

The important finding in `docs/implementation/reviews/task-258-review.md` was repaired without changing the task list or weakening the previously verified backend behavior:

- `reportObservabilityFailure` now accepts a closed category type and emits only fixed, bounded diagnostics for log, metric, and audit-identity failures. Sink error values are never formatted or written.
- `AuthController.warn` emits only the fixed refresh-cookie cleanup category and delegates sink failures to the bounded log-failure category. Refresh-cookie cleanup warnings no longer put `clearErr.Error()` into structured fields.
- `TestObservabilitySinkFailureUsesStderrFallback` now injects secret-bearing provider/token errors independently through router log and metric sinks, requires the fixed categories, and rejects secret, token, and provider text.
- `TestAuthWarningAndRefreshCleanupDoNotExposeErrors` injects a secret-bearing auth log-sink failure and refresh-cookie storage cleanup failure, then verifies fallback and structured logs contain no raw error text or secret-bearing field.

## Verification-criteria evidence

| Task 258 criterion | Direct evidence | Result |
|---|---|---|
| Non-admin 403 behavior | `TestAdminGatewayAuthorizationAllowlistAndSafeReadDegradation`, `TestClassificationAdminHTTPRejectsNonAdminMutationAndAuditFailureInvalidation`, `TestAdminExternalSearchForbidsNonAdminAndDoesNotAuditRead` | PASS under focused race suite. |
| Custom-item owner isolation, export, and erasure | `TestPostgresCustomFoodItemRepositoryOwnerScopedCRUD`, `TestServiceDerivesOwnershipAndKeepsCrossUserItemsNotFound`, `TestExportServiceBuildsJSONAndCSV`, `TestTask240CustomItemErasureIntegration` | PASS, including three-run erasure regression and full race. |
| Provider partial success and normalization | `TestExternalSearchAllOutcomesPartialSuccessCancellationAndSafeWarnings`, `TestExternalSearchProxyPartialAndCompleteOutageWarnings`, `TestNormalizeUSDAAliasesAndTrustedDensityPriority` and the full `externaldata` package | PASS under focused and full race suites. |
| Import/manual-create idempotency | `TestCuratedImportTransactionalWorkflow`, `TestPostgresManualFoodItemCRUD`, `TestManualItemAdminHTTPValidCRUDReplayAndAuditSnapshots`, `TestServiceCreateReplayConflictDuplicateAndCRUD` | PASS under focused and full race suites. |
| Transactional mutation-plus-audit rollback | `TestAdminMutationRollsBackWhenTransactionalAuditFails`, `TestClassificationAdminMutationRollsBackWhenAuditFails`, `TestTask258AdminAuditFailureIsCorrelatedObservableAndSanitized` | PASS; no false success and safe 503 telemetry. |
| Classification invalidation and in-use safeguards | `TestClassificationAdminHTTPCRUDConflictsAuditAndInvalidation`, `TestClassificationAdminRepositoryCRUDHierarchyConflictsAndSearchRename`, `TestCommittedRenameReplacesFilterOptionLabelAfterInvalidation` | PASS under focused and full race suites. |
| Authorized deletion retry | `TestPostgresAdminDeletionRetryPermitsOnlyLegalScopedFailures`, `TestPostgresAdminDeletionConcurrentRetryClaimsOnceWithAtomicAudits`, user-admin HTTP tests | PASS under focused and full race suites. |
| Immediate search visibility | Live search assertions in `TestCuratedImportTransactionalWorkflow`, `TestPostgresManualFoodItemCRUD`, and classification rename tests | PASS. |
| Provider outage degradation | `TestExternalSearchProxyPartialAndCompleteOutageWarnings`, `TestExternalSearchAllOutcomesPartialSuccessCancellationAndSafeWarnings`, admin external-search HTTP tests | PASS. |
| Concurrent mutation behavior | `TestPostgresAdminDeletionConcurrentRetryClaimsOnceWithAtomicAudits`, `TestServiceConcurrentCreateHasOneSideEffect`, import and custom-item concurrency tests, complete `-race` run | PASS. |
| Parameterized persistence | `TestTask258AdminPersistenceTreatsInputAsData` plus embedded parameterized repository SQL and existing parameterized search assertions | PASS against live PostgreSQL. |
| Sanitized responses/logs and operational correlation | `TestTask258AdminAuditFailureIsCorrelatedObservableAndSanitized`, `TestAdminMutationControlOrderAtomicAuditAndSanitizedEnvelopes`, provider safe-diagnostic tests, `govulncheck` | PASS. |

## Exact changed paths and symbols

| Path | Task-258 symbols or manifest surface |
|---|---|
| `backend/internal/httpapi/task258_admin_operations_integration_test.go` | Added `task258FailingAdminAudit`; `(*task258FailingAdminAudit).WithMutationAudit`; `TestTask258AdminAuditFailureIsCorrelatedObservableAndSanitized`; `task258HasHTTPMetric`. |
| `backend/internal/repository/task258_admin_persistence_integration_test.go` | Added `TestTask258AdminPersistenceTreatsInputAsData`. |
| `backend/internal/httpapi/router.go` | Repaired `reportObservabilityFailure` and its log, metric, and audit-identity callers to use fixed bounded categories. Preserved concurrent server request-ID/admin-route changes. |
| `backend/internal/httpapi/auth_controller.go` | Repaired `AuthController.warn` and refresh-cookie cleanup warning so raw errors do not cross fallback or structured-log boundaries. |
| `backend/internal/httpapi/router_test.go` | Updated the raw-output regression and added secret-bearing router/auth fallback and cleanup coverage. Preserved concurrent router/auth tests. |
| `backend/internal/app/task240_custom_item_erasure_integration_test.go` | Modified only `TestTask240CustomItemErasureIntegration` processing-clock initialization. All other pre-existing task-240 symbols were preserved. |
| `backend/internal/deletionworker/account_deletion.go` | Modified only `RunAccountDeletionProcessor`: successful cycles now stop if the cycle canceled the context. |
| `backend/go.mod` | Upgraded direct `golang.org/x/text` to `v0.39.0`; upgraded/marked direct required `golang.org/x/sync v0.21.0`; preserved and normalized concurrent direct `github.com/markbates/goth`. |
| `backend/go.sum` | Replaced vulnerable module checksums with `x/text v0.39.0` and required `x/sync v0.21.0`; retained tidy-required checksums. |

No JSON file changed, so no JSON traceability sidecar was required.

## Commands and evidence

| Command | Result |
|---|---|
| `go test ./internal/app -run '^TestTask240CustomItemErasureIntegration$' -count=3` | PASS; live PostgreSQL/Redis/API erasure is stable across three resets. |
| `go test ./internal/httpapi -run '^TestTask258AdminAuditFailureIsCorrelatedObservableAndSanitized$' -count=1` | PASS. |
| `go test ./internal/repository -run '^TestTask258AdminPersistenceTreatsInputAsData$' -count=1` | PASS against live PostgreSQL. |
| Focused Phase 08 package set with `go test -race -count=1 ... -run '<task-258 criterion regex>'` | PASS across app, HTTP, repository, external-data, import, item-curation, tag-management, user-admin, custom-item, and userdata packages. |
| `go test -race ./internal/deletionworker -run '^TestRunAccountDeletionProcessorRetriesAndReportsBoundedMetrics$' -count=100` | PASS 100/100; no extra post-cancellation cycles or race reports. |
| `go test -race -count=1 ./...` | PASS for every backend command and package after all fixes. |
| `go vet ./...` | PASS. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS after upgrade: zero called/imported vulnerabilities; 18 required-module advisories are not called. |
| `npx --no-install redocly lint api/openapi.yaml` | VALID; only the existing OAuth callback 302-only warning remains. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks with ordered dependencies. |
| `git diff --check` | PASS before preparation-document creation. |
| `python3 scripts/check.py` | Backend formatting, tests, 87.2% aggregate backend coverage, vet, vulnerability scan, local stack, Phase 02/03 UAT, OpenAPI, frontend build, browser verification, and 519 frontend tests PASS. The aggregate command exits 1 only at the existing/concurrent frontend 100% coverage policy: `src/lib/admin-workflows.ts` and `src/lib/api/admin-client.ts` are below 100%; task 258 did not change them. |
| `go test ./internal/httpapi -run '^(TestObservabilitySinkFailureUsesStderrFallback\|TestAuthWarningAndRefreshCleanupDoNotExposeErrors\|TestAuthControllerFailures)$' -count=1` | PASS; router log/metric fallback and auth warning/cleanup secret regressions pass. |
| `go test -race ./...` | PASS for every backend command and package after the observability repair. |
| `go test ./internal/... -coverprofile=/tmp/task-258-repair.coverage.out` and `go tool cover -func=...` | PASS; aggregate backend statement coverage is 87.4%. |
| `go vet ./...` | PASS after the observability repair. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS after the observability repair: zero called/imported vulnerabilities; 18 required-module advisories are not called. |
| `python3 scripts/validate-traceability.py` | PASS after the observability repair. |
| `python3 scripts/validate-task-list.py` | PASS after the observability repair: 263 sequential tasks with ordered dependencies; task list was not edited. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-258-review.md` | PASS; review evidence remains structurally valid. |
| `git diff --check` | PASS after the observability repair and preparation update. |

During diagnosis, one orphaned deletion request created by the intentionally interrupted test run was removed from the isolated `mealswapp_test` database before repeating migration-reset verification. No development or production database was targeted.

## Final implementation hashes

| Path | SHA-256 |
|---|---|
| `backend/go.mod` | `51031c31b5d3cc7119b2b26cc5670f088f75c96ef44c3b4cf8c2b4cd24f74e0d` |
| `backend/go.sum` | `962451b58ba814ed99f7a5dc88c6c02fb5579525be0965f2d916576e209ac29f` |
| `backend/internal/app/task240_custom_item_erasure_integration_test.go` | `ad51601fa941da6959edd0a364f76720e534a10dbe60a8367fc47daed35c9ff8` |
| `backend/internal/deletionworker/account_deletion.go` | `bd79a43c18839ea84aa883168fcfd036d591ecca592cd07561fedade3ec14014` |
| `backend/internal/httpapi/task258_admin_operations_integration_test.go` | `a5630710cff3d09eb7d5e891af4298be7ab0fd9a3773a815216b4ca5aff966e2` |
| `backend/internal/repository/task258_admin_persistence_integration_test.go` | `3cfd1c2618e743abf4b691b46ef0f38ebf409ec52e7ab7f1999b968caa588b32` |
| `backend/internal/httpapi/router.go` | `5e2095d29a6dc295ba004fee000f5f7a4c79f70de7381b147f5a56e917a73b3c` |
| `backend/internal/httpapi/auth_controller.go` | `96a5ad4c85029acf073111142d67f267b9b7a41f17e265aa27e155024d323796` |
| `backend/internal/httpapi/router_test.go` | `9a36b032794a14fe5e23f3041ac8fbf2c02ca116088c24600f61454d5dd79e11` |

Task 258 remains `OPEN` because this preparation intentionally did not edit `docs/implementation/02_TASK_LIST.md`.
