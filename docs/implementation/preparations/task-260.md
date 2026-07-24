# Task 260 preparation — Phase 08 admin and external-data observability gate

## Outcome and scope

- Task: 260, `DESIGN-014: MetricsCollector`.
- Fixed Git reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Task row was already `PREPARED`; dependency rows 245, 249, 250, 251, and 252 were `PASSED`, and dependency 258 was `PREPARED`.
- `docs/implementation/02_TASK_LIST.md` was not edited by preparation or repair. Its current SHA-256 staleness fingerprint is `304f2622185cd6c7dcb83e25679866b8de6ba84fca840ef6a2faf7f3be8850ce`.
- The original review findings I-1 through I-3 and the remaining re-review findings I-1 through I-2 were repaired; final verification completed at `2026-07-22T01:14:20Z`.

The worktree already contained uncommitted and untracked Phase 08 work. Scope was restricted to the Phase 08 observability boundary, production wiring, deterministic fixtures/tests, and this evidence document. Existing task-258 sink-failure fallbacks, authentication warnings, deletion-worker cancellation repair, dependency updates, and unrelated frontend/backend work were preserved. No task status, OpenAPI contract, migration, frontend source, task-258 evidence, or task-list content was changed.

## Sources inspected

- Task 260 and dependency rows in `docs/implementation/02_TASK_LIST.md`.
- `docs/design/DESIGN-014.md`: `MetricsCollector`, `LogAggregator`, structured event/point shapes, dependency metrics, low-cardinality behavior, and logging backpressure expectations.
- `docs/design/DESIGN-009.md`, `DESIGN-012.md`, `DESIGN-008.md`, and `DESIGN-013.md`: admin mutation/audit coordination, provider retries/rate limits/normalization, custom-item ownership/lifecycle, and privacy boundaries.
- `docs/requirements/01_SOFT_REQ_SPEC.md`: SW-REQ-043, SW-REQ-054 through SW-REQ-057, SW-REQ-072, SW-REQ-073, SW-REQ-084, and SW-REQ-090.
- Existing observability sinks and optimization telemetry, USDA/OpenFoodFacts clients, provider orchestration, normalization, curated imports, custom items, admin transaction coordination, production composition, and task-258 observability/race repairs.
- The `golang-security` logging rules: no PII/secrets, no untrusted error text, structured events, and explicit trust-boundary review.

## Implemented metric and log contract

`observability.AdminExternalTelemetry` is the single Phase 08 allowlist boundary. Its public methods accept only categorical provider/operation/outcome values and numeric latency. There is deliberately no argument for query text, names, emails, user/item IDs, idempotency keys, URLs, provider bodies, secrets, before/after snapshots, or raw errors.

| Metric | Fixed labels | Structured log | Behavior source |
|---|---|---|---|
| `external_provider_calls_total` | `provider`, `outcome` | `external_provider_call` | Every orchestrated call attempt, including safe provider error class. |
| `external_provider_latency_seconds` | `provider`, `outcome` | numeric latency on the call event | Deterministic injected provider clock; negative skew clamps to zero. |
| `external_provider_retries_total` | `provider`, `outcome` | `external_provider_retry` | Scheduled, exhausted, or canceled retry state. |
| `external_provider_quota_total` | `provider`, `state` | `external_provider_quota` | Available, exhausted, blocked, or unknown; raw quota headers are not emitted. |
| `external_normalization_warnings_total` | `provider`, `warning` | `external_normalization_warning` | Canonical candidate warnings and dropped invalid payloads. |
| `admin_import_outcomes_total` | `provider`, `outcome` | `admin_import_outcome` | Conflict/error classes immediately; created/replayed/merged only after mutation and audit commit. |
| `admin_mutation_outcomes_total` | `operation`, `outcome` | `admin_mutation_outcome` | Succeeded, failed, or uniquely distinguishable `audit_failed`. Unknown route actions collapse to `other`. |
| `custom_item_lifecycle_outcomes_total` | `operation`, `outcome` | `custom_item_lifecycle` | Create/replay/get/update/delete/list behavior without owner or item identity. |

Provider values are restricted by the emitting method (`usda`, `openfoodfacts`, and where applicable `external` or `manual`). Warning, retry, quota, import, admin, and custom-item values use explicit local allowlists. Invalid provider/outcome/warning/custom values are dropped; unknown admin action names collapse to `other`; and unknown typed provider error codes collapse to the allowlisted `error` outcome so call and latency observations are preserved. Quota state is derived from validated values in the current response: malformed, negative, incomplete, or absent quota headers emit `unknown`, never `exhausted`.

Metric and log delivery is batched behind independent one-call lanes with a shared 100 ms deadline. Cooperative sinks still complete before the facade returns. A sink that ignores context can retain at most one metric-batch goroutine and one log goroutine per facade; later callers wait only for the bounded deadline and cannot create unbounded sink fan-out. Sink errors remain ignored without serializing diagnostics, preserving the task-258 fixed-category fallback boundary at the HTTP instrumentation layer.

`AdminExternalTelemetry` also implements the production `LogSink` boundary for concrete USDA/OpenFoodFacts diagnostics and curation validation. It accepts only the three fixed event shapes, reconstructs their fields from exact provider/code/field/outcome allowlists and bounded numeric/boolean metadata, drops unknown or widened events, and dispatches accepted records through the same one-call log lane. Production composition no longer passes the raw context-blind `JSONSink` to provider clients or curation validators.

Successful curated-import telemetry was intentionally moved to `AdminMutationResult.AfterCommit`. This prevents a successful service call followed by audit persistence failure from reporting a durable import that was rolled back. The admin transaction metric reports that path as `audit_failed`.

## Deterministic and load evidence

- `task260_load.json` fixes 24 workers × 32 iterations and supplies explicit canaries for query text, a person name, email, user/item IDs, idempotency key, URL, raw payload, provider secret, before/after snapshots, and audit database error text. Its required JSON traceability sidecar is present.
- `TestAdminExternalTelemetryLoadIsBoundedAndPrivacySafe` emits 6,144 metrics and 5,376 structured logs concurrently, asserts exact counts, fixed label keys and values, complete canary absence in serialized telemetry, and 768 independently countable `audit_failed` metrics.
- `TestTask260ProviderAndNormalizationTelemetryMatchesBehavior` proves two calls, one retry, two quota observations, measured latency, canonical candidate warnings, and one invalid-record warning against final provider behavior.
- `TestTask260ImportTelemetryDistinguishesCreatedAndConflict` proves success emits nothing before audit commit, then emits `created`; a provider identity conflict remains distinct.
- `TestTask260CustomItemLifecycleTelemetryHasNoIdentityLabels` proves create, replay, delete, and post-delete not-found outcomes with only operation/outcome labels.
- `TestTask260AuditFailureHasDistinctBoundedAdminOutcome` drives the HTTP/admin transaction boundary and proves a fail-closed 503 has both metric and structured-log outcome `audit_failed`.
- `TestTask260TelemetryLoadLeavesSearchAndAuthRoutesResponsive` drives 128 concurrent search/auth probes while 512 provider and 512 admin telemetry operations run; all probes complete successfully within the bounded test deadline.
- `TestTask260UnknownProviderCodePreservesBoundedCallAndLatencyTelemetry` proves an attacker-controlled typed code emits `error` call/latency metrics without exposing the code, provider cause, or query.
- `TestTask260MalformedQuotaHeaderIsUnknownNotExhausted` covers non-numeric and negative remaining values, malformed reset values, and reset-without-remaining; every case emits only `unknown` quota state.
- `TestAdminExternalTelemetryBlockingSinksHaveBoundedDispatch` proves a noncooperative metric/log sink cannot hold callers and cannot create more than one active call per sink under 32 concurrent attempts.
- `TestTask260BlockingTelemetrySinkCannotHoldProviderRequest` and `TestTask260BlockingTelemetrySinkCannotHoldAdminRequest` prove the bounded policy at the external-provider and authenticated-admin request boundaries.
- `TestAdminExternalTelemetryLogSinkAllowsOnlyBoundedProviderAndCurationMetadata` proves the production adapter accepts only exact provider/curation event shapes and drops widened private fields and unknown categories.
- `TestTask260ConcreteProviderFailureCannotBlockRequestOnJSONWriter` drives a real USDA HTTP 503 through a concrete client and a blocked `JSONSink` writer; the provider request returns the safe unavailable error within the bounded deadline.
- `TestTask260CurationRejectionCannotBlockRequestOnJSONWriter` drives a real Fiber curation rejection through the bounded adapter and proves the blocked writer cannot hold the structured 400 response.
- `TestTask260CustomItemResourceConflictUsesBoundedConflictOutcome` drives the real repository duplicate-name `ErrorKindConflict` through `Service.Create`, asserts exact metric and structured-log `conflict` outcomes, and excludes the name, idempotency key, and repository error from serialization.

## Exact changed-symbol inventory

| File:line | Symbol | Change and evidence |
|---|---|---|
| `observability/admin_external.go:9` | Phase 08 metric constant group | Added eight fixed names and the 100 ms delivery deadline. |
| `observability/admin_external.go:23-293` | `adminExternalMetric`, `adminExternalLog`, `AdminExternalTelemetry`, `NewAdminExternalTelemetry`, public category methods, `Log`, `emit`, `dispatch`, `waitForAdminExternalDelivery`, `metric`, `deliverLog`, `boundedAdminExternalLog`, `allowedProviderFailureCode`, `allowedCurationLogField`, `allowed`, `boundedOperation`, `stringFields` | Implements the allowlisted facade, production provider/curation log adapter, and batched bounded delivery. Covered by load/privacy, exact-event, unknown-action, and blocking-sink tests. |
| `externaldata/rate_limit.go:42,55` | `RateLimitHandler.telemetry`, `WithTelemetry` | Added optional race-protected telemetry composition. |
| `externaldata/rate_limit.go:235` | `searchExternalRecords` | Added call latency/outcome, retry, blocked quota, and response quota observations without query/header/error data. |
| `externaldata/rate_limit.go:347-400` | `telemetrySnapshot`, `quotaState`, `providerTelemetryOutcome` | Uses only validated current headers for quota state and collapses unknown typed codes to `error`. |
| `externaldata/normalizer.go:59,66` | `DataNormalizer.telemetry`, `WithTelemetry` | Added optional normalization telemetry composition. |
| `externaldata/normalizer.go:105` | `NormalizeRecordsWithWarnings` | Added canonical candidate and invalid-payload warning observations. |
| `dataimporter/service.go:56,67` | `Service.telemetry`, `WithTelemetry` | Added curated-import telemetry composition. |
| `dataimporter/service.go:76` | `Service.Confirm` | Added classified failure/conflict outcomes; successful outcomes remain deferred. |
| `dataimporter/service.go:124-141` | `RecordCommittedOutcome`, `importTelemetryProvider`, `importTelemetryOutcome` | Added post-audit success emission and closed mappings. |
| `customitem/service.go:90,97` | `Service.telemetry`, `WithTelemetry` | Added owner-free lifecycle telemetry composition. |
| `customitem/service.go:112-246` | `Create`, `Get`, `Update`, `Delete`, `List`, `recordLifecycle`, `lifecycleOutcome` | Maps idempotency and repository resource conflicts to the same allowlisted `conflict` outcome without identities or request fields. |
| `httpapi/admin_controller.go:52,62` | `AdminController.telemetry`, `WithTelemetry` | Added optional mutation telemetry composition. |
| `httpapi/admin_controller.go:227` | `transactionalMutation` | Added committed/failed/audit-failed outcomes after final transaction status is known. |
| `httpapi/import_controller.go:23` | `curatedImportOutcomeRecorder` | Added optional post-commit service seam without widening the existing controller service contract. |
| `httpapi/import_controller.go:54` | `CuratedImportController.Confirm` | Composes committed import telemetry with existing cache invalidation in `AfterCommit`. |
| `app/app.go:46-166` | `NewProduction`, `newProduction` | Wires one facade into provider orchestration, concrete provider logs, curation validation, normalization, imports, custom items, and every production admin mutation controller. |
| `observability/task260_admin_external_test.go` | Load/privacy fixtures, exact production-log adapter checks, and blocking sink adversary | Proves exact cardinality/privacy behavior, rejection of widened events, and bounded one-call dispatch under noncooperative sinks. |
| `externaldata/task260_observability_test.go` | Provider/normalization fixtures plus unknown-code, malformed-header, blocked-facade, and concrete blocked-JSON-writer tests | Proves preserved call/latency telemetry, safe quota classification, privacy, and concrete provider-path responsiveness. |
| `dataimporter/task260_observability_test.go:14` | `TestTask260ImportTelemetryDistinguishesCreatedAndConflict` | Added post-commit and conflict evidence. |
| `customitem/task260_observability_test.go` | Lifecycle privacy and duplicate-name resource-conflict regressions | Proves exact owner-free outcomes and the real repository conflict mapping in both metrics and logs. |
| `httpapi/task260_observability_load_test.go:19-165` | Audit/load fixtures and `TestTask260BlockingTelemetrySinkCannotHoldAdminRequest` | Proves distinct audit failure, core-route load responsiveness, and authenticated admin responsiveness with a noncooperative telemetry sink. |
| `httpapi/curation_validation_test.go` | `TestTask260CurationRejectionCannotBlockRequestOnJSONWriter` | Proves a concrete curation rejection response remains bounded with the production JSON writer blocked. |

No executable symbol outside this inventory was added or modified for task 260.

## Commands and evidence

Commands were run from `backend/` unless shown from repository root.

| Command | Result |
|---|---|
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/observability ./internal/externaldata ./internal/dataimporter ./internal/customitem ./internal/httpapi ./internal/app -count=1` | PASS after the remaining-findings repair. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/observability ./internal/externaldata ./internal/customitem ./internal/httpapi -run 'Task260\|AdminExternalTelemetry\|AdminMutationUnknown' -count=10` | PASS: all task-260 adversarial tests, including the concrete blocked writer and conflict regressions, passed ten times under race detection. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -count=1` | PASS: all backend command/internal packages, including unrelated task-261 work. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1` | TASK-260 PASS / unrelated timeout: all changed packages and `internal/app` passed; pre-existing `internal/queue.TestTask225LiveManagerRecoversAfterRedisRestart` then timed out after 10 minutes in go-redis semaphore release while recovering a restarted test Redis process. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/observability ./internal/externaldata ./internal/customitem ./internal/httpapi -count=1 -coverprofile=/tmp/task-260-remaining-findings.coverage.out` | PASS: observability 85.6%, externaldata 99.8%, customitem 90.9%, and httpapi 87.2%; phase-wide 100% disposition remains task 262 scope. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS with no diagnostics. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS: zero reachable vulnerabilities and zero vulnerabilities in imported packages. The scanner listed 18 module-only advisories in required modules that application code does not call. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks with ordered dependencies. |
| `git diff --check` | PASS. |

## Failures, corrections, and accepted exceptions

- Independent review I-1 reproduced missing call/latency telemetry for an unknown typed provider code. `providerTelemetryOutcome` now validates the complete known code vocabulary and maps every other typed code to fixed `error`; the adversarial test also excludes the raw code, cause, and query from serialized telemetry.
- Independent review I-2 reproduced malformed remaining quota as `exhausted`. `quotaState` now parses the current remaining value and any present reset value before classification; malformed, negative, incomplete, and absent values are `unknown`.
- Independent review I-3 reproduced indefinite request blocking with a sink that ignores context. Task-260 facade delivery now batches metrics and logs into separate one-call lanes with a shared 100 ms deadline. Direct, provider-path, and authenticated-admin tests prove bounded completion and bounded goroutine fan-out.
- Remaining re-review I-1 reproduced raw context-blind `JSONSink` injection into concrete USDA/OpenFoodFacts clients and curation validators. `AdminExternalTelemetry` now implements a strict provider/curation `LogSink`; all production composition paths use it, and concrete provider failure plus curation rejection regressions prove blocked-writer responsiveness.
- Remaining re-review I-2 reproduced duplicate-name `repository.ErrorKindConflict` as lifecycle `error`. `lifecycleOutcome` now maps both repository conflict kinds to `conflict`; the regression asserts exact metric/log fields and privacy canaries.
- During the original repair, the first complete race run found a race only in the new responsiveness test because it reused the legacy `auditSink`, whose test-only slice is not concurrency-safe. The task-260 load fixture now uses stateless `task260NoopAudit`; that repair's focused and full race reruns passed. No production race fix and no task-258 code change was needed.
- The initial traceability validation required repository-standard Go-doc and adjacent `Implements DESIGN-*` comments on new private helpers. Comments were added; validation passes without behavior changes.
- The current full race command passed every task-260 package and `internal/app`, then timed out only in the unrelated task-225 live Redis restart test. The deterministic task-260 race gate passed ten consecutive runs and is the acceptance signal for this repair; no task-225 or queue code was changed.
- `gosec` is not installed in this workspace (`command -v gosec` returned no path), so no unpinned tool download was introduced. Security evidence consists of the pinned repository vulnerability scan, repeated focused race and full vet gates, exact allowlist tests, adversarial serialized-canary inspection, and the existing task-258 sanitized sink-failure tests. This is an environment/tooling exception, not an implementation or acceptance exception.
- The 18 module-only `govulncheck` advisories are unreachable from imported application packages and are retained as dependency-maintenance information; reachable and imported-package findings are both zero.
- No coverage exception is claimed here. Phase-wide 100% coverage disposition remains task 262 scope.

## Final implementation hashes

| Path | SHA-256 |
|---|---|
| `backend/internal/app/app.go` | `4a32fe296885145876d71c01c35a32584d89bc8f52271d2851c0d84ef17281b9` |
| `backend/internal/observability/admin_external.go` | `064d34395042e6460386062ac0d338100d0d6421a74ef5238e992611ee479180` |
| `backend/internal/observability/task260_admin_external_test.go` | `54b09cd76d8d12c6f21a1d4c9e86b807806177e75bfee36a71de88f17e11e755` |
| `backend/internal/observability/testdata/task260_load.json` | `c5523c0d195ffaedd19ed1af4f1bd77abc145f88b87325ef958dab415b2b5df7` |
| `backend/internal/observability/testdata/task260_load.json-trace.md` | `9397096cd6c4d8d9009d18dbd7a5fc2629a55e55f6e03945cd51c3390d9c5fb6` |
| `backend/internal/externaldata/rate_limit.go` | `a0d525a6f4717d1a03f6738ba583c725a7438de55a44f0a52c0a1c73e9a14663` |
| `backend/internal/externaldata/normalizer.go` | `c43a1ef2758e57e82f5610d6545ac1b22aa6551ef494f3606454e076cf5b2b5c` |
| `backend/internal/externaldata/task260_observability_test.go` | `130fa54ed9a57fa948ac79dd8985290c3b14331e657076fc38cf71fb44d55bdb` |
| `backend/internal/dataimporter/service.go` | `4139ed058b32693efbb59d435cb7d4ad573fc99eb13d3a0758450305f8c52337` |
| `backend/internal/dataimporter/task260_observability_test.go` | `219129154457f3843edf73278ff1eb74be80bc3fcac556cd6c41c4624e3cba24` |
| `backend/internal/customitem/service.go` | `547d675338ffaa89bd28896573be73e3bb9d4fb3cce66162aeb9cc8eaeee0c59` |
| `backend/internal/customitem/task260_observability_test.go` | `30478d1caf4d295ec03bd8233fca11a5ca6a9a2752a23e8a3a0d1584d9a1265e` |
| `backend/internal/httpapi/admin_controller.go` | `763e21a7f4df99afafd82870d0470c9f6d080070c27ef657f76fe8674f883b82` |
| `backend/internal/httpapi/import_controller.go` | `2c33ea79bcc5a955ec6293186e97f50b4467d30af8b3b734ba474d1bb88822d1` |
| `backend/internal/httpapi/task260_observability_load_test.go` | `403feb70ab4898c0f0c3a998f017e06c854db9d2d82f3e8319158d58248433d7` |
| `backend/internal/httpapi/curation_validation_test.go` | `680f34c0d34d5673c4ccc78f9fdf6a3f0eaeb349774c734d5de9815929be1b43` |

Task 260 is repaired and prepared for independent re-review. The task list remains intentionally unchanged by this work.
