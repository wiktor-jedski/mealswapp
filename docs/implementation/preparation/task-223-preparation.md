# Task 223 Preparation — Submission Observability and Bounded Cleanup

## Scope and attribution

- Task: `223`, Phase 07.01 Submission Observability and Bounded Cleanup.
- Design source: `docs/design/DESIGN-014.md`, static aspects `MetricsCollector` and `LogAggregator`; the controller collaboration continues to implement `DESIGN-004: JobStatusTracker`.
- Dependencies preserved: Tasks `220`, `221`, and `222` remain `PASSED` in the current task list.
- Task 223 intentionally remains `OPEN`; no task-list status or other task row was changed.
- Preparation date: 2026-07-17 (Europe/Warsaw).
- The worktree already contained overlapping uncommitted Tasks 220–222 changes. Task 223 was implemented against those current symbols without reverting their canonical request, acknowledgement, repair, publication, or admission-ownership behavior.
- Tasks 224 and later queue reservation/finalization work remain out of scope.

## Implemented contract

### One bounded submission vocabulary

`observability.OptimizationSubmissionOutcome` is the sole submission-result type. Its complete allow-listed values are `accepted`, `replayed`, `rejected`, `dependency_error`, `queue_error`, and `error`. `OptimizationTelemetry.Submission` accepts that type, and `validOptimizationMetric` derives its allowed label values from the same constants. A caller-created unsupported typed value is dropped at the telemetry boundary.

`OptimizationController.Submit` uses a named final error and classifies that final result in its defer. Successful acknowledgement writes set `accepted` or `replayed` only after the write succeeds. `optimizationSubmissionFailure` maps queue-unavailable errors first, bounded dependency errors second, all public 4xx paths to `rejected`, and uncategorized/internal failures to `error`. Pending-publication repair returns its own typed result, so an unsuccessful repair cannot retain a speculative `replayed` outcome.

### Lifecycle-bounded final outcome telemetry

`OptimizationTelemetry.Submission` admits final metric and log delivery through separate one-slot adapter-owned lanes and gives both deliveries one shared 100 ms lifecycle deadline. Cooperative sinks still complete before `Submission` returns, preserving deterministic typed-outcome observation. A sink that ignores context can retain at most one submission metric call and one submission log call; later callers wait only within their own shared deadline and cannot create unbounded sink goroutines.

The HTTP response therefore remains bounded even when failed cleanup telemetry and final submission telemetry share a universally blocking or writer-serialized sink. The timeout changes only best-effort delivery: it cannot replace the controller's typed outcome, authoritative HTTP status, or public error envelope.

### Label ownership and privacy

`cloneLabels` retains the established semantics: nil and empty inputs produce nil, while populated inputs produce an independent map. The populated copy now uses standard-library `maps.Copy`. Tests mutate both source and clone to prove no aliasing.

Submission telemetry has exactly one fixed `outcome` label. Admission cleanup telemetry has exactly one fixed `outcome=failed` label. The cleanup adapter accepts no user, diet, job, idempotency-key, request-body, or diagnostic argument. Focused tests serialize all captured metrics and logs and reject authenticated IDs, diet IDs, keys, bodies, primary errors, and release diagnostics.

### Bounded best-effort cleanup

`OptimizationController.releaseAdmission` detaches cleanup from request cancellation and applies the local 100 ms deadline, but admits release work through one controller-owned cleanup lane before starting a goroutine. A permanently noncooperative gate can retain that sole lane; repeated failures are rejected from the lane without creating more goroutines. Cooperative completion releases the lane for later owner-scoped cleanup.

`OptimizationTelemetry.AdmissionCleanupFailed` independently admits its fixed metric and event through one asynchronous lane per sink. Each delivery receives a 100 ms background timeout. A sink that honors context is lifecycle-bounded; a sink that ignores context can retain only its one lane and cannot hold the HTTP request or create unbounded delivery goroutines. Metric and log lanes are separate, so one blocked sink cannot suppress first delivery to the other sink.

Cleanup remains best effort: a failed, timed-out, capacity-rejected, or telemetry-blocked cleanup never replaces the authoritative submission error. Regressions repeat eight requests against a release that remains blocked for the complete assertion window and against independently blocking cleanup metric and log sinks. A production `JSONSink` regression additionally uses one writer that blocks every cleanup and submission metric/log write across eight requests. Every request retains the safe `503 queue_unavailable` response; the writer observes exactly one cleanup metric, one cleanup log, one submission metric, and one submission log call, proving the four adapter-owned lanes cap outstanding noncooperative work; emitted records contain only the fixed vocabularies.

## Changed Task 223 surfaces

| Path | Symbols or tests |
| --- | --- |
| `backend/internal/observability/optimization.go` | `OptimizationSubmissionOutcome` and six constants; `MetricOptimizationAdmissionCleanup`; typed and lifecycle-bounded `OptimizationTelemetry.Submission`; one-slot submission metric/log lanes; `AdmissionCleanupFailed`; independent cleanup lanes; typed submission allowlist; cleanup metric/event allowlists; `cloneLabels` using `maps.Copy` |
| `backend/internal/observability/observability.go` | race-safe `MemorySink.Snapshot` used to observe asynchronous cleanup delivery without reading sink storage concurrently |
| `backend/internal/observability/observability_test.go` | existing telemetry fixture migrated to the typed accepted outcome |
| `backend/internal/observability/task223_optimization_test.go` | complete-vocabulary rejection/acceptance and nil/empty/populated clone ownership tests |
| `backend/internal/httpapi/optimization_controller.go` | final-result classification in `OptimizationController.Submit`; typed repair outcomes; `optimizationSubmissionFailure`; 100 ms `releaseAdmission` deadline; one-slot cap for permanently noncooperative release work; sanitized failure observation |
| `backend/internal/httpapi/task223_submission_observability_test.go` | table-driven six-outcome HTTP/metric coverage; failed-repair classification; repeated permanently blocked release and selective cleanup-sink regressions; production `JSONSink`/universally blocking writer response-bound and four-lane-cap regression; label/PII assertions |
| `docs/implementation/preparation/task-223-preparation.md` | this evidence |

## Verification-criteria mapping

| Task 223 criterion | Evidence | Result |
| --- | --- | --- |
| Six exact final outcomes | `TestTask223SubmissionHTTPOutcomesMatchFinalResponse` covers accepted, successful replay, rejection, dependency failure, queue failure, and unexpected internal failure | PASS |
| Failed repair is never replayed | `TestTask223FailedRepairIsNotReplayed` creates a pending claim after queue failure, fails repair revalidation, and observes `dependency_error`, never `replayed` | PASS |
| Fixed, non-sensitive labels | typed allowlist test plus serialized HTTP telemetry assertions; unsupported caller-controlled outcome is dropped | PASS |
| Nil/empty/populated clone semantics | `TestTask223CloneLabelsPreservesSemanticsAndOwnership` verifies nil preservation, legacy empty-to-nil behavior, equality, and two-way mutation independence | PASS |
| Cleanup is bounded and observable | immediate error, eight permanently blocked release attempts capped at one call, and eight attempts against each blocking sink capped at one blocked delivery; fixed cleanup metric/event | PASS |
| Cleanup preserves the primary response | release and sink failure fixtures retain `503 queue_unavailable`; release diagnostics and identifiers are absent from response and telemetry | PASS |
| Final outcome telemetry is lifecycle-bounded | a production `JSONSink` over a writer blocking every metric and log write cannot hold `Submit` beyond the shared 100 ms telemetry deadline plus scheduling margin; all eight primary `503 queue_unavailable` responses complete and only four writes remain outstanding | PASS |
| Concurrency safety | focused Task 223 tests pass for 25 repetitions under the Go race detector; full backend race test passes | PASS |

## Commands and results

| Working directory | Command | Result |
| --- | --- | --- |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -run '^TestTask223UniversallyBlockingSinkDoesNotBlockPrimaryResponse$' -count=1` before the production repair | EXPECTED FAIL at 500 ms; reproduced the review finding because final `Submission` telemetry held the response |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -run '^TestTask223UniversallyBlockingSinkDoesNotBlockPrimaryResponse$' -count=10` after the repair and strengthened writer regression | PASS; each repetition completed eight authoritative 503 responses within the bound and retained exactly one blocked write in each of the four lifecycle-capped lanes |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/observability ./internal/httpapi -run 'Task223|OptimizationTelemetryUsesBoundedLabels|OptimizationHTTPSubmissionAndPolling|OptimizationHTTPIdempotencyAndQueueFailure' -count=1` | PASS; corrected Go regexp alternation selected the intended tests |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/observability ./internal/httpapi -run 'Task223|OptimizationTelemetryUsesBoundedLabels|OptimizationHTTPSubmissionAndPolling|OptimizationHTTPIdempotencyAndQueueFailure' -count=10` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/httpapi ./internal/observability -run 'Task223' -count=25` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/observability ./internal/httpapi -count=1` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./internal/observability ./internal/httpapi` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -count=1` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi ./internal/observability -coverprofile=/tmp/task-223-repair.coverage.out -count=1` and `go tool cover -func=/tmp/task-223-repair.coverage.out` | PASS; HTTP API 89.2%, observability 73.1%, combined 88.0% statements |
| repository root | `python3 scripts/validate-task-list.py` | PASS; 237 sequential tasks with ordered dependencies; Task 223 remains `OPEN` |
| repository root | `python3 scripts/validate-traceability.py` | PASS |
| repository root | `git diff --check` | PASS after preparation refresh |

## Current hashes

| Path | SHA-256 |
| --- | --- |
| `backend/internal/httpapi/optimization_controller.go` | `b30d1f00bdc946908300f1f704b64f1f5ba4bd60f7590529f71a6b377a43965c` |
| `backend/internal/httpapi/task223_submission_observability_test.go` | `542629229075b1c8f3e6d80dee7036cd6c1cc3e15405dd525e1138c72541e563` |
| `backend/internal/observability/optimization.go` | `92cf748939933adbf73eae1b463935689bf8e34ea5b8cb9a7f261fff6f5f7433` |
| `backend/internal/observability/observability.go` | `8e4ab1928b6b995dea55a49b4fa364a6e1b02367cc6983106e5610804b5b3eba` |
| `backend/internal/observability/observability_test.go` | `58ea7bb18432ff1b5a0b6161cf5e5389cdeb55c91aea6a51921bcc5d679ae5cd` |
| `backend/internal/observability/task223_optimization_test.go` | `251deca606e3836d73508576b3f076567c3db43e0b09ebb14c29c47a44e09ac1` |
| `docs/design/DESIGN-014.md` | `f913c8087efc1d9e928b316d821cb88f2710316e232d482e7bfe6f963019a2ee` |
| `docs/implementation/02_TASK_LIST.md` | `5e33b75edd838e98be60a0a0e734dc33b46bef19a09bd2a96accbe7d0c1fbab0` |

## Security and blockers

- HTTP request bodies and authenticated identifiers remain inside the controller trust boundary and are never accepted by the Task 223 telemetry methods. Metric values and event names are fixed vocabulary only.
- Release diagnostics are deliberately discarded rather than logged. This prevents Redis/provider details or identifier-bearing errors from entering logs while still exposing a failure counter/event to operators.
- The remaining important finding in `docs/implementation/reviews/task-223-review.md` is repaired: final submission metric/log delivery is now lifecycle-bounded and independently capped, including when cleanup delivery shares the same universally blocking sink. Permanently noncooperative release and cleanup telemetry remain independently capped. No Task 223 implementation blocker remains.
- The focused package coverage is below the phase-wide 100% goal. Task 223 adds focused branch/race evidence but does not claim the later Task 235 aggregate coverage gate; no new coverage deviation is accepted here.

## Preparation decision

The repaired Task 223 implementation and refreshed focused evidence satisfy the task-row criteria. Per explicit instruction, Task 223 remains `OPEN`; Tasks 220–224 retain their existing statuses.
