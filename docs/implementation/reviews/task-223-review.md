# Review Evidence: Task 223 — MetricsCollector

~~~yaml
task_id: 223
component: "DESIGN-014: MetricsCollector"
static_aspect: "MetricsCollector"
input_status: "OPEN (preserved because the user prohibited task-status edits)"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-17T15:26:15Z"
review_agent: "fresh independent owner review"
evidence_file: "docs/implementation/reviews/task-223-review.md"
baseline_ref: "HEAD a4e31367485b03269e90b5607f2057c9568bb5b1 plus refreshed task-223-preparation.md manifest"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "Go guide; error handling, context, goroutines, interfaces, race safety, testing, and performance guidance applied"
repair_context_required: true
~~~

## 1. Task Source

**Description:** Phase 07.01: replace raw submission outcome strings with one bounded typed vocabulary, classify every final controller path accurately, use maps.Copy without changing label clone semantics, and make failed admission-slot cleanup time-bounded and observable without sensitive labels.

**Depends On:** 222 (PASSED in the current task list)

**Testing Coverage Exceptions:** None in the Task 223 row. The repository has a prior Phase 07 coverage disposition in docs/implementation/04_OPEN.md; this review records current scoped coverage and manually audits every changed symbol.

**Verification Criteria:** Table-driven HTTP/metrics tests cover accepted, successful replayed, rejected, dependency_error, queue_error, and unexpected error; failed repair is never replayed; all labels pass a fixed allowlist and contain no user, diet, job, key, or body data; nil/empty/populated label cloning preserves nil behavior and mutation independence; release failure cannot replace the primary HTTP response or block indefinitely and emits sanitized telemetry; focused observability tests and race tests pass.

## 2. Pre-Review Gates

- [x] The user explicitly required Task 223 to remain OPEN; the refreshed preparation report claims completion and is independently reviewable without changing the status cell.
- [x] Dependency 222 is PASSED.
- [x] The refreshed preparation report lists the Task 223 surfaces, criteria, commands, and current hashes.
- [x] A task-specific baseline is available despite cumulative dirty-worktree overlap.
- [x] code-review-skill was invoked exactly once and its complete Go guide was read.
- [x] This reviewer is independent from the implementation/repair work.
- [x] Current source and current command results were used; the earlier rejected review was treated as historical evidence only.
- [x] No production file or task-status cell was changed.

~~~yaml
pre_review_gates_passed: true
blocking_issue: "Task status remains OPEN only because the user prohibited status edits; the independent review decision is recorded here."
~~~

## 3. Review Baseline and Change Surface

Baseline/reference method: HEAD is a4e31367485b03269e90b5607f2057c9568bb5b1. The worktree contains cumulative uncommitted Tasks 220–222 and other phase changes, so Task 223 ownership was reconstructed from the refreshed preparation manifest, current hashes, current source, and the Task 223-specific untracked tests. The prior review's pre-repair hashes and findings were not reused as current proof.

Commands used to reconstruct the diff:

~~~bash
git status --short
git rev-parse HEAD && git log -8 --oneline --decorate
git diff -- backend/internal/httpapi/optimization_controller.go backend/internal/observability/optimization.go backend/internal/observability/observability.go
rg -n 'OptimizationSubmissionOutcome|AdmissionCleanupFailed|deliverSubmission|deliverCleanup|releaseAdmission|cloneLabels|JSONSink' backend/internal
sha256sum <current Task 223 files, dependency files, design files, task list, and preparation report>
~~~

Pre-existing dirty-worktree changes and exclusions:

The worktree contains modified backend, frontend, API, migration, design, and task-list paths owned by Tasks 213–222 and concurrent phase work. Those changes were preserved and excluded from Task 223 attribution except where their current callers or contracts were necessary to audit behavior. backend/internal/worker/optimization_admission.go was inspected for the release interface and owner semantics; backend/internal/httpapi/router.go was inspected for ClassifyServerError and request-context behavior; neither was attributed as a Task 223 implementation change. No task-list status cell was edited.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| backend/internal/httpapi/optimization_controller.go | preparation manifest and current source | HIGH | submission classification, failed-publication repair result, bounded admission cleanup lane |
| backend/internal/httpapi/task223_submission_observability_test.go | Task-specific untracked test listed by preparation | HIGH | six outcomes, failed repair, release cap, selective and universal blocking sinks, privacy assertions |
| backend/internal/observability/optimization.go | preparation manifest and current source | HIGH | typed vocabulary, allowlists, clone, submission lanes, cleanup lanes |
| backend/internal/observability/observability.go | preparation manifest and current source | HIGH | race-safe MemorySink.Snapshot; JSON sink contract audited for context ignorance |
| backend/internal/observability/observability_test.go | preparation manifest and current source | HIGH | typed-outcome/race, clone, and privacy regressions |
| backend/internal/observability/task223_optimization_test.go | Task-specific untracked test listed by preparation | HIGH | vocabulary and clone ownership tests |

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | HTTP and metrics cover the six exact final outcomes: accepted, successful replay, rejected, dependency error, queue error, and unexpected error. | TestTask223SubmissionHTTPOutcomesMatchFinalResponse; manual audit of Submit and optimizationSubmissionFailure. | PASS | Corrected focused test passed; each table case asserted HTTP status and final submission outcome. |
| 2 | A failed pending-publication repair is never classified or returned as replayed. | TestTask223FailedRepairIsNotReplayed; every error return in repairOptimizationPublication; named-result defer in Submit. | PASS | Initial queue failure creates a pending claim; later repair revalidation failure returns 503 and dependency_error, never replayed. |
| 3 | Submission and cleanup labels/events use fixed allowlists and contain no user, diet, job, key, body, primary-error, or release-diagnostic data. | Typed outcome boundary, validOptimizationMetric, validOptimizationMessage, fixed cleanup adapter, serialized HTTP assertions. | PASS | Unsupported typed values are dropped; cleanup accepts no sensitive argument; current tests reject the listed values. |
| 4 | Nil, empty, and populated label cloning preserves nil behavior and mutation independence using maps.Copy. | TestTask223CloneLabelsPreservesSemanticsAndOwnership; existing clone regression; source inspection. | PASS | Nil and empty maps remain nil; populated maps are equal but independent in both mutation directions. |
| 5 | Failed admission cleanup is best effort, time-bounded, observable, and cannot replace or indefinitely block the primary HTTP response. | Permanent-release cap, independent cleanup-sink cap, universal JSONSink writer regression, defer/source audit. | PASS | Eight requests retain safe 503 queue_unavailable; current universal-blocking test passes 10 repetitions and observes one outstanding call per cleanup and submission lane. |
| 6 | Final submission telemetry remains bounded when a production JSONSink ignores context or shares a blocked writer with cleanup telemetry. | Submission, deliverSubmission, waitForOptimizationDelivery, JSONSink, universal writer test. | PASS | Each of the four adapter-owned lanes admits at most one blocked sink call; the shared 100 ms final telemetry deadline returns the request. |
| 7 | Focused observability tests and race tests pass. | Focused tests, 10× and 25× Task 223 race repetitions, full backend tests/race, vet, and coverage profiles. | PASS | All current commands exited 0; full backend race suite passed. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | optimizationAdmissionCleanupTimeout, optimizationAdmissionCleanupLimit | controller constants | backend/internal/httpapi/optimization_controller.go:34-35 | added | releaseAdmission and constructor | release timing and one-slot tests |
| 2 | OptimizationController | behavioral type | backend/internal/httpapi/optimization_controller.go:68-79 | modified | HTTP router and controller methods | HTTP integration tests |
| 3 | NewOptimizationController | constructor | backend/internal/httpapi/optimization_controller.go:84-88 | modified | production app wiring and tests | all controller fixtures |
| 4 | (*OptimizationController).Submit | route handler | backend/internal/httpapi/optimization_controller.go:110-302 | modified | Fiber POST route | outcome, repair, cleanup, race tests |
| 5 | (*OptimizationController).repairOptimizationPublication | method | backend/internal/httpapi/optimization_controller.go:304-413 | modified | Submit pending-claim paths | failed-repair and Task 222 dependency tests |
| 6 | optimizationSubmissionFailure | classifier | backend/internal/httpapi/optimization_controller.go:415-429 | added | Submit final-result defer and error paths | six-outcome table |
| 7 | (*OptimizationController).releaseAdmission | cleanup method | backend/internal/httpapi/optimization_controller.go:431-463 | added | Submit release defers and conflict path | release error, permanent-block, sink tests |
| 8 | MetricOptimizationAdmissionCleanup, optimizationTelemetryTimeout | observability constants | backend/internal/observability/optimization.go:11-26 | added | cleanup allowlist and adapter deadlines | cleanup and universal writer tests |
| 9 | OptimizationSubmissionOutcome and six constants | typed vocabulary | backend/internal/observability/optimization.go:28-40 | added | controller and telemetry boundary | vocabulary and HTTP outcome tests |
| 10 | OptimizationTelemetry | behavioral type | backend/internal/observability/optimization.go:50-62 | modified | app, queue, worker, controller telemetry | telemetry and race tests |
| 11 | NewOptimizationTelemetry | constructor | backend/internal/observability/optimization.go:64-75 | modified | production app and worker wiring | package and HTTP fixtures |
| 12 | (*OptimizationTelemetry).Submission | method | backend/internal/observability/optimization.go:77-96 | modified | Submit defer | typed outcome, universal sink, race tests |
| 13 | (*OptimizationTelemetry).deliverSubmission | method | backend/internal/observability/optimization.go:98-115 | added | Submission metric/log lanes | universal sink and race tests |
| 14 | waitForOptimizationDelivery | helper | backend/internal/observability/optimization.go:118-129 | added | Submission final wait | universal sink and race tests |
| 15 | (*OptimizationTelemetry).AdmissionCleanupFailed | method | backend/internal/observability/optimization.go:131-143 | added | releaseAdmission failure path | cleanup and universal sink tests |
| 16 | (*OptimizationTelemetry).deliverCleanup | method | backend/internal/observability/optimization.go:145-159 | added | cleanup metric/log lanes | cleanup sink and race tests |
| 17 | validOptimizationMetric | allowlist helper | backend/internal/observability/optimization.go:267-297 | modified | record and Submission | vocabulary/privacy tests |
| 18 | optimizationSubmissionOutcomes | allowlist helper | backend/internal/observability/optimization.go:299-307 | added | typed boundary and metric validation | vocabulary test |
| 19 | validOptimizationMessage | allowlist helper | backend/internal/observability/optimization.go:315-324 | modified | event cleanup/submission paths | cleanup/privacy tests |
| 20 | cloneLabels | map clone helper | backend/internal/observability/optimization.go:326-335 | modified | record sink delivery | clone ownership tests |
| 21 | (*MemorySink).Snapshot | method | backend/internal/observability/observability.go:100-109 | added | asynchronous test observation | HTTP cleanup tests |
| 22 | JSONSink.Log, JSONSink.RecordMetric | sink methods | backend/internal/observability/observability.go:70-79 | dependency audit | adapter sink boundary; methods ignore context | JSON sink and universal writer tests |
| 23 | TestCloneLabelsPreservesNilSemanticsAndMutationIndependence | test | backend/internal/observability/observability_test.go:16-45 | added | clone helper | itself |
| 24 | TestOptimizationSubmissionOutcomesAreBoundedAndRaceSafe | test | backend/internal/observability/observability_test.go:47-82 | added | typed submission adapter | itself and race runs |
| 25 | TestOptimizationTelemetryUsesBoundedLabelsAndNoPrivateData | test | backend/internal/observability/observability_test.go:98-137 | modified | full optimization telemetry | itself |
| 26 | TestTask223SubmissionHTTPOutcomesMatchFinalResponse | test | backend/internal/httpapi/task223_submission_observability_test.go:22-78 | added | controller route | itself |
| 27 | TestTask223FailedRepairIsNotReplayed | test | backend/internal/httpapi/task223_submission_observability_test.go:80-101 | added | pending repair | itself |
| 28 | TestTask223AdmissionCleanupIsBoundedObservableAndBestEffort | test | backend/internal/httpapi/task223_submission_observability_test.go:103-139 | added | release path | itself |
| 29 | TestTask223RepeatedPermanentlyBlockedReleaseHasBoundedOutstandingWork | test | backend/internal/httpapi/task223_submission_observability_test.go:141-159 | added | release lane | itself |
| 30 | TestTask223BlockingCleanupSinksDoNotBlockPrimaryResponse | test | backend/internal/httpapi/task223_submission_observability_test.go:161-192 | added | cleanup metric/log lanes | itself |
| 31 | TestTask223UniversallyBlockingSinkDoesNotBlockPrimaryResponse | test | backend/internal/httpapi/task223_submission_observability_test.go:194-210 | added | cleanup plus final submission lanes | itself |
| 32 | task223Admission | test double type | backend/internal/httpapi/task223_submission_observability_test.go:212-216 | added | controller admission boundary | cleanup tests |
| 33 | (*task223Admission).Acquire | test-double method | backend/internal/httpapi/task223_submission_observability_test.go:244-246 | added | controller | cleanup tests |
| 34 | (*task223Admission).Release | test-double method | backend/internal/httpapi/task223_submission_observability_test.go:248-254 | added | controller cleanup | release cap tests |
| 35 | task223BlockingCleanupSink | test double type | backend/internal/httpapi/task223_submission_observability_test.go:218-226 | added | cleanup sink boundary | selective blocking test |
| 36 | newTask223BlockingCleanupSink | test helper | backend/internal/httpapi/task223_submission_observability_test.go:256-258 | added | cleanup sink fixture | selective blocking test |
| 37 | (*task223BlockingCleanupSink).RecordMetric | test-double method | backend/internal/httpapi/task223_submission_observability_test.go:260-271 | added | metrics adapter | selective blocking test |
| 38 | (*task223BlockingCleanupSink).Log | test-double method | backend/internal/httpapi/task223_submission_observability_test.go:274-285 | added | log adapter | selective blocking test |
| 39 | (*task223BlockingCleanupSink).waitForCleanup | test helper | backend/internal/httpapi/task223_submission_observability_test.go:288-301 | added | cleanup observations | selective blocking test |
| 40 | (*task223BlockingCleanupSink).unblock | test helper | backend/internal/httpapi/task223_submission_observability_test.go:303-305 | added | test cleanup | selective blocking test |
| 41 | (*task223BlockingCleanupSink).blockedCalls | test helper | backend/internal/httpapi/task223_submission_observability_test.go:307-312 | added | cap assertion | selective blocking test |
| 42 | task223UniversallyBlockingWriter | test double type | backend/internal/httpapi/task223_submission_observability_test.go:228-236 | added | production JSONSink writer | universal sink test |
| 43 | (*task223UniversallyBlockingWriter).Write | test-double method | backend/internal/httpapi/task223_submission_observability_test.go:314-329 | added | JSONSink | universal sink test |
| 44 | (*task223UniversallyBlockingWriter).unblock | test helper | backend/internal/httpapi/task223_submission_observability_test.go:331-333 | added | test cleanup | universal sink test |
| 45 | (*task223UniversallyBlockingWriter).assertCappedDeliveries | test helper | backend/internal/httpapi/task223_submission_observability_test.go:335-340 | added | four lane cap assertion | universal sink test |
| 46 | task223FailingJobStore | test double type | backend/internal/httpapi/task223_submission_observability_test.go:238 | added | unexpected error case | outcome table |
| 47 | (task223FailingJobStore).Save | test-double method | backend/internal/httpapi/task223_submission_observability_test.go:240-242 | added | Submit job persistence | outcome table |
| 48 | assertTask223PrimaryQueueResponse | test helper | backend/internal/httpapi/task223_submission_observability_test.go:342-350 | added | primary response checks | release/sink tests |
| 49 | lastTask223SubmissionOutcome | test helper | backend/internal/httpapi/task223_submission_observability_test.go:352-360 | added | MemorySink metric scan | outcome/repair tests |
| 50 | task223CleanupMetric | test helper | backend/internal/httpapi/task223_submission_observability_test.go:363-370 | added | sanitized cleanup check | cleanup test |
| 51 | assertTask223TelemetryIsBounded | test helper | backend/internal/httpapi/task223_submission_observability_test.go:372-397 | added | serialized privacy check | outcome/cleanup tests |
| 52 | waitForTask223CleanupTelemetry | test helper | backend/internal/httpapi/task223_submission_observability_test.go:399-414 | added | asynchronous MemorySink observation | cleanup test |

~~~yaml
inventory_source_count: 52
audited_symbol_count: 52
inventory_complete: true
generated_groupings:
  - "Typed outcome constants and immutable metric/deadline constants are grouped only where every named constant is listed explicitly. JSONSink methods are included as a dependency audit because the repaired regression depends on their context-ignoring behavior."
~~~

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| optimizationAdmissionCleanupTimeout, optimizationAdmissionCleanupLimit | Set the documented 100 ms cleanup bound and one controller lane. | Positive fixed values; no runtime input. | Bound release wait and outstanding release goroutines. | No data. | Constant time and space. | Minimal local configuration. | Timing and permanent-block tests. | PASS |
| OptimizationController | Owns admission cleanup state alongside controller collaborators. | Nil collaborators remain handled by existing guards. | Lane is controller-local and buffered to one; cross-instance Redis ownership remains in the gate. | User/job IDs stay in admission call only. | One channel per controller. | Necessary state, no extra mutex. | Controller fixtures and race suite. | PASS |
| NewOptimizationController | Initializes the cleanup lane for every production/test controller. | All constructor inputs remain injectable. | Prevents zero-capacity lane in normal construction. | Does not copy telemetry data. | One fixed channel allocation. | Idiomatic constructor. | All controller construction sites and tests. | PASS |
| (*OptimizationController).Submit | Classifies the final returned error and emits one typed final outcome while preserving the authoritative response. | Covers auth, validation, dependency, replay, admission, persistence, queue, repair, and response-write failures; defer maps unassigned errors. | Release defer runs before the final telemetry defer; both are bounded after repair. | Telemetry receives only typed outcome and context. | One final metric/log pair with fixed payloads. | Named result makes defer classification explicit; current Task 222 flow remains intact. | Six-outcome, repair, cleanup, universal sink, full integration, and race tests. | PASS |
| (*OptimizationController).repairOptimizationPublication | Never returns replay for unsuccessful repair; only a published acknowledgement is replayed. | Revalidates entitlement, diet, claim, admission, job state, save, queue, publication, and response paths. | Owned slot is released on every failed owned path; queue publication transfers ownership only after state check. | User ID is used for ownership but not telemetry. | Bounded request-side operations; no solver work. | Typed result separates classification from HTTP error. | Failed-repair regression plus prior Task 222 matrix. | PASS |
| optimizationSubmissionFailure | Maps final public error category to exactly one allow-listed outcome, prioritizing queue unavailable. | Queue, dependency/timeout, 4xx, and unknown errors are distinct; nil is not called by the defer. | Pure function, no state or waits. | Only fixed enum returned. | Constant-size classification. | Small, explicit switch using shared error classifier. | Six-outcome table and source branch audit. | PASS |
| (*OptimizationController).releaseAdmission | Best-effort release cannot replace the primary result and reports only fixed cleanup failure. | Immediate release error, cooperative success, timeout, and occupied lane all handled. | One admission lane; one release goroutine per admitted lane; detached 100 ms context; later failures do not spawn more work. | IDs go only to gate; cleanup telemetry takes no IDs or diagnostics. | At most one retained noncooperative release call per controller. | select default is a clear capacity gate. | Release error, permanent block, and universal writer tests; full race. | PASS |
| MetricOptimizationAdmissionCleanup, optimizationTelemetryTimeout | Define fixed cleanup metric and common 100 ms telemetry deadline. | Immutable values only. | Deadline is applied separately to cleanup and submission lifecycles. | Names contain no request data. | Constant. | Centralized bounded configuration. | Cleanup and universal sink regressions. | PASS |
| OptimizationSubmissionOutcome and six constants | Define the complete submission vocabulary. | Unsupported aliases are rejected before sink delivery. | Immutable values; no shared mutation. | Caller cannot inject arbitrary label text through the typed boundary. | Fixed six-entry map when validated. | Narrow typed API. | All six plus attacker-like value. | PASS |
| OptimizationTelemetry | Holds sinks, worker gauge, and independent one-slot submission/cleanup state. | Constructor-created channels are valid; nil sinks are tolerated by lower methods. | Atomic cleanup flags and channels are race-safe; one lane per sink kind. | No identity fields are stored. | Constant state per adapter. | Small focused adapter. | Race suite and blocking fixtures. | PASS |
| NewOptimizationTelemetry | Normalizes nonpositive worker capacity and initializes all lanes. | Zero/negative capacity becomes one; nil sinks remain allowed. | Channels have capacity one; atomics start clear. | No sensitive input. | Fixed allocations. | Simple constructor. | Existing telemetry tests and all blocking tests. | PASS |
| (*OptimizationTelemetry).Submission | Emits one valid metric and event with one shared 100 ms caller lifecycle. | Nil receiver and unsupported outcome return; valid outcome starts metric/log deliveries. | WithoutCancel preserves best-effort delivery; waits stop at deadline; each sink lane caps noncooperative work. | Payload contains only fixed outcome. | Two goroutines at most per admitted call, with global per-adapter lane caps. | Clear separation between admission and waiting. | Typed, privacy, universal sink, and repeated race tests; nil-context misuse is outside Go Context contract. | PASS |
| (*OptimizationTelemetry).deliverSubmission | Admit at most one metric or log delivery and always release the lane when delivery returns. | Deadline while waiting drops delivery; valid lane starts a delivery goroutine. | Deferred release and done close handle cooperative completion; context cannot stop a sink that ignores it, but caller bound remains. | Closure payload is fixed by Submission. | One goroutine per occupied lane; no unbounded retry loop. | Channel semaphore is idiomatic for a one-slot cap. | Universal writer and race repetitions. | PASS |
| waitForOptimizationDelivery | Wait for cooperative completion or the shared deadline without blocking forever. | Nil done channel is a no-op; both completion and timeout are handled. | Does not own resources or leak waiters. | N/A. | One select. | Minimal helper. | Universal sink exercises timeout; cooperative submissions exercise completion. | PASS |
| (*OptimizationTelemetry).AdmissionCleanupFailed | Emit fixed failed-cleanup metric/event without request details. | Nil receiver no-op; both sinks independently attempted. | Each sink uses its own atomic one-slot lane and does not wait on caller context. | No IDs, errors, keys, or body accepted. | At most one retained goroutine per sink kind. | Small best-effort façade. | Cleanup and universal writer tests; nil/sink-absent direct branch remains optional gap. | PASS |
| (*OptimizationTelemetry).deliverCleanup | Admit one cleanup delivery per sink lane and clear the lane after return. | CAS rejects duplicate while blocked; admitted delivery runs with background 100 ms context. | Atomic CAS prevents races; context bounds cooperative sinks, not a sink that ignores it. | Delivery closure is fixed before entry. | One goroutine per lane, no request blocking. | Explicit bounded async lane. | Blocking cleanup and 25× race tests. | PASS |
| validOptimizationMetric | Fail closed on unknown metric names, label keys, counts, or values. | Nil/empty exact-label metrics and fixed-label metrics are distinguished; malformed maps reject. | Pure local allowlist construction. | Arbitrary PII labels cannot pass. | Small bounded maps. | Auditable explicit policy. | Vocabulary and serialized privacy assertions. | PASS |
| optimizationSubmissionOutcomes | Derive the submission metric allowlist from the six typed constants. | Exactly six values; no caller-provided extension. | Fresh immutable-by-convention map per call. | Prevents arbitrary outcome labels. | Six-entry allocation, acceptable for telemetry. | Avoids duplicated raw strings in the production allowlist. | Vocabulary test and raw-string search. | PASS |
| validOptimizationMessage | Allow only fixed optimization log messages, including cleanup. | Unknown message rejects; cleanup and submission names accept. | Pure switch. | No user-controlled message path from Task 223. | Constant-time. | Simple switch is idiomatic. | Cleanup and privacy tests. | PASS |
| cloneLabels | Preserve nil/empty-to-nil behavior and copy populated maps independently. | Nil, empty, and populated inputs handled; source/destination mutations do not alias. | No shared state after copy. | Copies only already-validated labels. | O(n) map allocation. | maps.Copy is the requested standard helper. | Both clone tests. | PASS |
| (*MemorySink).Snapshot | Return race-safe slice copies for asynchronous test observation. | Nil receiver returns nil; empty/nonempty sinks copy safely. | Mutex covers reads against sink append; returned slices do not alias backing slices. | Test-only sink; records remain fixed by producer. | O(n) slice copy. | Small synchronization façade. | Cleanup tests and full race; shallow record-map copy is not mutated by these producers. | PASS |
| JSONSink.Log, JSONSink.RecordMetric | Encode one structured record to the configured writer. | Writer errors propagate; context is intentionally ignored by this sink. | A noncooperative writer can block the call, so adapter lanes must provide lifecycle bounds. | Records are sanitized before this boundary. | One JSON encode/write per delivery. | Existing sink contract is preserved; adapter is the boundary repair. | JSON sink unit test and universal blocking writer regression. | PASS |
| TestCloneLabelsPreservesNilSemanticsAndMutationIndependence | Regression for clone shapes and ownership. | Table covers nil, empty, populated, and both mutation directions. | No asynchronous state. | Fixed test values. | Tiny maps. | Direct assertions. | Self-covering. | PASS |
| TestOptimizationSubmissionOutcomesAreBoundedAndRaceSafe | Prove concurrent valid outcomes emit exactly once and attacker outcome emits nothing. | All six outcomes plus unsupported value. | WaitGroup synchronizes concurrent calls; MemorySink is observed after waits. | Rejects user-like outcome. | Six fixed goroutines. | Focused table/parallel test. | Run under race detector. | PASS |
| TestOptimizationTelemetryUsesBoundedLabelsAndNoPrivateData | Preserve existing broad telemetry privacy contract after typed API change. | Submission, queue, worker, solve, job, retry, expiry, invalid record, and invalid labels. | Cooperative sink; calls complete before inspection. | Serializes records and rejects PII-like values. | Fixed fixture. | Regression retained. | Package tests and full race. | PASS |
| TestTask223SubmissionHTTPOutcomesMatchFinalResponse | Map six controller outcomes to authoritative HTTP responses and telemetry. | Accepted, replay, entitlement rejection, missing dependency, queue failure, unexpected persistence failure. | Each case uses isolated fakes and waits for synchronous MemorySink delivery. | Serialized telemetry rejects IDs/key/body/diagnostics. | One request or replay pair per case. | Table-driven and direct. | Focused test and race repetitions. | PASS |
| TestTask223FailedRepairIsNotReplayed | Prove queue failure leaves repairable claim but failed revalidation is dependency error. | Queue outage then diet dependency failure. | Deferred release is cooperative in fixture. | No sensitive telemetry fields. | Two bounded requests. | Narrow regression. | Focused test. | PASS |
| TestTask223AdmissionCleanupIsBoundedObservableAndBestEffort | Prove release error/timeout retains primary queue response and emits sanitized cleanup. | Immediate error and context-ignoring release. | Test unblocks intentional fake in cleanup; production timeout is source-audited. | Rejects IDs, body, key, primary error, release diagnostic. | Two fixed cases with 5× deadline margin. | Clear adversarial fixture. | Focused and race runs. | PASS |
| TestTask223RepeatedPermanentlyBlockedReleaseHasBoundedOutstandingWork | Prove repeated failures do not create release goroutine growth. | Eight requests against a fake that ignores context. | Atomic call count; final cleanup unblocks the one retained fake. | Test-only IDs. | Fixed eight requests. | Direct cap assertion. | Focused and race runs. | PASS |
| TestTask223BlockingCleanupSinksDoNotBlockPrimaryResponse | Prove metric and log cleanup lanes are independent and fixed. | Each sink kind selectively blocks cleanup deliveries; eight requests follow. | Buffered observations and atomics avoid test-side races; deferred unblock. | Exact fixed cleanup metric/event asserted. | Fixed eight-request regression per sink. | Good lane-isolation fixture. | Focused and race runs. | PASS |
| TestTask223UniversallyBlockingSinkDoesNotBlockPrimaryResponse | Regress the previously rejected production JSONSink universal-blocking behavior. | One writer blocks cleanup and submission metric/log writes across eight requests. | Four adapter lanes retain one blocked call each; writer unblocks after assertions. | Writer classifies only fixed record names; no unknown output. | Eight bounded requests; response bound is 500 ms versus 100 ms component deadlines. | Tests actual JSONSink, not only a selective fake. | 10× focused and full race runs. | PASS |
| task223Admission | Model owned admission with configurable error or context-ignoring release. | Acquire always grants; release can return error or block. | Atomic call count; test cleanup closes block. | Test-only UUIDs. | Constant-time or intentional block. | Minimal interface fake. | Cleanup and cap tests. | PASS |
| (*task223Admission).Acquire | Return an acquired decision for cleanup paths. | No error branch by design. | No shared mutation. | Test-only. | Constant-time. | Minimal fake. | Cleanup tests. | PASS |
| (*task223Admission).Release | Count and model cooperative/error/noncooperative release. | Error, success, and indefinite wait until test cleanup. | Atomic count; block intentionally ignores context to challenge production bound. | No output. | One call per admitted lane. | Appropriate adversarial fake. | Release and cap tests. | PASS |
| task223BlockingCleanupSink | Model one independently blocked cleanup sink while observing fixed records. | Metric or log selector blocks; other record kinds return. | Atomic counts, buffered channels, once-only close. | Only fixed records are captured. | Bounded channels. | Focused fake. | Selective sink test. | PASS |
| newTask223BlockingCleanupSink | Initialize deterministic blocking/observation channels. | Test table supplies valid selector. | Channels are bounded and independent. | Fixed test state. | Constant allocations. | Simple constructor. | Selective sink test. | PASS |
| (*task223BlockingCleanupSink).RecordMetric | Capture cleanup metric and optionally block. | Cleanup metric path blocks only when selected; all else returns. | Atomic count and buffered send. | Caller asserts exact fixed label. | Constant-size operation. | Minimal fake. | Selective sink test. | PASS |
| (*task223BlockingCleanupSink).Log | Capture cleanup event and optionally block. | Cleanup event path blocks only when selected; all else returns. | Atomic count and buffered send. | Caller asserts exact fixed fields. | Constant-size operation. | Minimal fake. | Selective sink test. | PASS |
| (*task223BlockingCleanupSink).waitForCleanup | Wait for both fixed cleanup observations with a bounded test timeout. | Two records required; timeout fails test. | Select synchronizes observations. | Fixed records only. | 500 ms upper wait. | Clear helper. | Selective sink test. | PASS |
| (*task223BlockingCleanupSink).unblock | Release blocked fake exactly once. | Repeated calls safe. | sync.Once prevents close panic. | Test-only. | Constant time. | Idiomatic cleanup. | Deferred in blocking test. | PASS |
| (*task223BlockingCleanupSink).blockedCalls | Read selected call count. | Metric/log selector returns matching counter. | Atomic load. | Test-only. | Constant time. | Minimal assertion helper. | Selective sink test. | PASS |
| task223UniversallyBlockingWriter | Model a writer that blocks every JSON delivery and classify fixed records. | Four expected record categories plus unknown. | Atomic counters and once-only unblock; all calls wait on block channel. | Unknown writes are counted and fail the assertion. | Constant-size classification. | Direct production sink fixture. | Universal writer test. | PASS |
| (*task223UniversallyBlockingWriter).Write | Count record category before deliberately ignoring context. | All expected JSON payload markers classify; unknown fails later. | Atomic counter and intentional block. | Ensures only fixed telemetry records are emitted. | Bounded marker checks. | Small deterministic writer. | Universal writer test. | PASS |
| (*task223UniversallyBlockingWriter).unblock | Release every blocked writer call once. | Repeated calls safe. | sync.Once protects close. | Test-only. | Constant time. | Idiomatic cleanup. | Deferred in universal test. | PASS |
| (*task223UniversallyBlockingWriter).assertCappedDeliveries | Require one outstanding call per cleanup/submission lane and no unknown output. | Exact four counters and zero unknown. | Atomic loads after all requests; writer remains blocked during assertion. | Fixed category assertion. | Constant time. | Direct cap evidence. | Universal writer test. | PASS |
| task223FailingJobStore | Override only Save to force an uncategorized internal failure. | Embedded store keeps other fixture behavior. | No resources. | Error is not forwarded to telemetry. | Constant-time. | Minimal wrapper. | Outcome table. | PASS |
| (task223FailingJobStore).Save | Return a deliberate internal error. | Always errors. | No resources. | Test diagnostic remains outside response/telemetry assertions. | Constant-time. | Minimal fake. | Unexpected outcome case. | PASS |
| assertTask223PrimaryQueueResponse | Lock the safe 503 queue response and response-time bound. | Requires code queue_unavailable; rejects replacement error. | No shared state. | Primary error not serialized by telemetry. | Fixed 500 ms assertion margin. | Clear helper. | Blocking and universal tests. | PASS |
| lastTask223SubmissionOutcome | Find latest submission metric while skipping cleanup metrics. | Missing metric fails; reverse scan bounded by fixture size. | Reads after synchronous delivery. | Fixed label only. | O(fixed metrics). | Simple helper. | Outcome and repair tests. | PASS |
| task223CleanupMetric | Recognize exact fixed cleanup metric. | Requires exact name, one label, and failed. | Pure slice read after snapshot. | Rejects extra labels. | O(fixed metrics). | Exact assertion. | Cleanup wait helper. | PASS |
| assertTask223TelemetryIsBounded | Serialize captured telemetry and reject extra labels and sensitive values. | Marshal failure fails; checks IDs, key, body, primary error, release diagnostic. | Uses race-safe Snapshot; no concurrent map mutation in fixture. | Strong representative privacy boundary. | Small payload. | Useful integration assertion. | Outcome and cleanup tests. | PASS |
| waitForTask223CleanupTelemetry | Wait for both asynchronous cleanup records within a bounded window. | Polls snapshots until both appear or fails after 500 ms. | Snapshot lock prevents races. | Fixed cleanup records. | 1 ms polling, bounded. | Deterministic async helper. | Cleanup test. | PASS |

Mandatory audit conclusion: the typed outcome classifier, failed-repair handling, fixed privacy vocabulary, maps.Copy ownership, one-slot release cap, independent cleanup lanes, and the repaired universal JSONSink response bound all pass. The final telemetry defer is now bounded and independently capped, so the prior important finding is closed.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| 🟢 [nit] | backend/internal/observability/optimization.go:133-159; /tmp/task-223-final.coverage.out | AdmissionCleanupFailed, deliverCleanup | The ordinary two-package coverage profile does not instrument imported observability code from HTTP tests, so these async methods appear at 0.0% in the owning-package report. The cross-package Task 223 profile executes them at 83.3% and 100.0%; nil/sink-absent branches remain untested. | Fresh package profile: HTTP 89.1%, observability 73.1%, combined 87.9%. Fresh cross-package HTTP coverpkg profile: async cleanup methods 83.3% and 100.0%. | Optional: add direct observability-package tests for nil receiver, nil sinks, and duplicate-lane admission if the later phase gate requires owning-package branch visibility. No repair is required for Task 223 acceptance. |

~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 1
~~~

The prior important universal-sink finding is resolved by the lifecycle-bounded Submission path and the passing production JSONSink regression. The earlier permanent-release and selective-cleanup findings remain resolved by the controller lane and independent cleanup lanes. The coverage observation is optional and does not fail any Task 223 criterion.

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| gofmt -d <six reviewed Go files> | backend/ | 0 | PASS | No formatting diff. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/observability ./internal/httpapi -run 'Task223|OptimizationTelemetryUsesBoundedLabels|OptimizationHTTPSubmissionAndPolling|OptimizationHTTPIdempotencyAndQueueFailure' -count=1 | backend/ | 0 | PASS | Focused intended tests selected. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -run '^TestTask223UniversallyBlockingSinkDoesNotBlockPrimaryResponse$' -count=10 | backend/ | 0 | PASS | Repaired production JSONSink regression, 10 repetitions. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/httpapi ./internal/observability -run 'Task223|OptimizationTelemetryUsesBoundedLabels|OptimizationHTTPSubmissionAndPolling|OptimizationHTTPIdempotencyAndQueueFailure' -count=10 | backend/ | 0 | PASS | Focused race repetitions. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/httpapi ./internal/observability -run 'Task223' -count=25 | backend/ | 0 | PASS | Task 223 race repetitions. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/observability ./internal/httpapi -count=1 | backend/ | 0 | PASS | Scoped package tests. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./internal/observability ./internal/httpapi | backend/ | 0 | PASS | Scoped vet. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -count=1 | backend/ | 0 | PASS | Full backend suite. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1 | backend/ | 0 | PASS | Full backend race suite. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi ./internal/observability -coverprofile=/tmp/task-223-final.coverage.out -count=1 && go tool cover -func=/tmp/task-223-final.coverage.out | backend/ | 0 | PASS with optional coverage observation | /tmp/task-223-final.coverage.out; HTTP 89.1%, observability 73.1%, total 87.9%. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -coverpkg=github.com/wiktor-jedski/mealswapp/backend/internal/observability -coverprofile=/tmp/task-223-final-crosspkg.coverage.out -run 'Task223' -count=1 && go tool cover -func=/tmp/task-223-final-crosspkg.coverage.out | backend/ | 0 | PASS | /tmp/task-223-final-crosspkg.coverage.out; AdmissionCleanupFailed 83.3%, deliverCleanup 100.0%. |
| python3 scripts/validate-task-list.py | repository root | 0 | PASS | 237 sequential tasks; Task 223 remains OPEN. |
| python3 scripts/validate-traceability.py | repository root | 0 | PASS | Traceability validator. |
| git diff --check | repository root | 0 | PASS | No whitespace errors. |
| sha256sum <reviewed files> | repository root | 0 | PASS | Current hashes recorded in Section 9. |

The aggregate scripts/check.py was not run because its documented Docker/Chromium local-stack checks are outside this scoped review and would exercise unrelated dirty-worktree work. Backend tests/race, focused coverage, vet, formatting, task-list, traceability, and diff checks were run directly. The phase-orchestrator review evidence validator is run after this document is written.

## 9. Files Inspected and Staleness Fingerprints

Current contents were hashed after the review commands. The old rejected review was checked for staleness; its pre-repair hashes differ for the repaired controller, observability implementation, and Task 223 test surface, so its rejection evidence was not reused. observability.go, the admission dependency, router classifier, design source, preparation report, and task list were also rehashed.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| backend/internal/httpapi/optimization_controller.go | typed final classification, repair result, release cleanup lane | prior important finding repaired | SHA-256 | b30d1f00bdc946908300f1f704b64f1f5ba4bd60f7590529f71a6b377a43965c |
| backend/internal/httpapi/task223_submission_observability_test.go | Task 223 HTTP, release, sink, and privacy regressions | universal writer regression | SHA-256 | 542629229075b1c8f3e6d80dee7036cd6c1cc3e15405dd525e1138c72541e563 |
| backend/internal/observability/optimization.go | typed outcomes, allowlists, clone, submission and cleanup lanes | prior important finding repaired; optional coverage nit | SHA-256 | 92cf748939933adbf73eae1b463935689bf8e34ea5b8cb9a7f261fff6f5f7433 |
| backend/internal/observability/observability.go | MemorySink snapshot and JSONSink contract | dependency behavior audited | SHA-256 | 8e4ab1928b6b995dea55a49b4fa364a6e1b02367cc6983106e5610804b5b3eba |
| backend/internal/observability/observability_test.go | existing clone, typed/race, privacy, and JSON sink tests | optional coverage context | SHA-256 | 58ea7bb18432ff1b5a0b6161cf5e5389cdeb55c91aea6a51921bcc5d679ae5cd |
| backend/internal/observability/task223_optimization_test.go | Task 223 vocabulary and clone tests | none | SHA-256 | 251deca606e3836d73508576b3f076567c3db43e0b09ebb14c29c47a44e09ac1 |
| backend/internal/worker/optimization_admission.go | production release interface and owner-scoped semantics | dependency context | SHA-256 | 6c27535a913f83b2d103093d227aaec8134eadaa625389469ad491e79252e756 |
| backend/internal/httpapi/router.go | error classification and request-context behavior | dependency context | SHA-256 | cd4c888689151d66051561c3aaedd6ac379149df950a09eec30d0f7591566125 |
| docs/design/DESIGN-014.md | MetricsCollector and LogAggregator source of truth | no contradiction | SHA-256 | f913c8087efc1d9e928b316d821cb88f2710316e232d482e7bfe6f963019a2ee |
| docs/implementation/preparation/task-223-preparation.md | refreshed scope, repair evidence, and hashes | current preparation checked | SHA-256 | 45570f5a43d91144666280e3830f7a41be6208827e1fcc1aa14a125437c222a3 |
| docs/implementation/02_TASK_LIST.md | Task 223 status and criteria | status intentionally preserved | SHA-256 | 5e33b75edd838e98be60a0a0e734dc33b46bef19a09bd2a96accbe7d0c1fbab0 |

~~~yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "docs/implementation/reviews/task-223-review.md before this rewrite; its pre-repair controller/observability/test hashes and important finding were superseded by the refreshed repair and current review"
~~~

## 10. Coverage and Exceptions

- [x] Required scoped coverage command ran.
- [x] Report paths and observed thresholds are recorded.
- [x] Untested branches relevant to changed symbols were manually inspected.
- [x] The Task 223 row has no explicit coverage exception; the existing Phase 07 coverage disposition and the cross-package async evidence are distinguished from the Task 223 acceptance decision.

~~~yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "/tmp/task-223-final.coverage.out; /tmp/task-223-final-crosspkg.coverage.out"
observed_line_coverage: "HTTP 89.1%; observability 73.1%; combined 87.9%; cross-package Task 223 execution covers AdmissionCleanupFailed 83.3% and deliverCleanup 100.0%"
coverage_passed: true
~~~

Coverage finding: the profile does not meet the repository's eventual 100% phase goal, and direct owning-package tests for nil/sink-absent async branches are absent. The documented Phase 07 coverage disposition accepts the broader defensive/dependency gaps; the Task 223 acceptance criterion requires focused tests and race evidence rather than a numerical threshold. The uncovered branches were manually inspected and remain visible as the optional finding in Section 7.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced by the reviewed Task 223 symbols.
- [x] No source-of-truth documentation was contradicted.
- [x] No generated/cache/build/temporary artifact was unintentionally added by this review.
- [x] The typed public telemetry boundary is necessary and used; unsupported values fail closed.
- [x] Raw submission outcome duplication was searched; production values are centralized in the typed vocabulary and allowlist.
- [x] Error, cleanup, timeout, concurrency, malformed-input, response-preservation, and privacy paths were challenged.
- [x] The previous universal JSONSink blocking regression was reproduced as a current regression target and passes after repair.

Findings: no blocking or important correctness, security, behavior, lifecycle, or race issue remains. The only retained observation is optional coverage visibility for async observability branches.

## 12. Decision

A task may be PASSED only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains. Those conditions are satisfied.

Before accepting the decision, run:

~~~bash
python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-223-review.md
~~~

~~~yaml
decision: "PASSED"
reason: "The repaired typed telemetry and cleanup implementation satisfies every Task 223 criterion, including the previously failing universal JSONSink response bound, with current race, coverage, privacy, and hash evidence."
failed_criteria:
  - ""
failed_or_unaudited_symbols:
  - ""
recommended_next_action: "None for Task 223; keep the task status OPEN as instructed and optionally add direct owning-package async cleanup coverage before the phase-wide gate."
~~~

## 13. Repair Context

Not applicable to the current PASSED decision. The prior important finding was nevertheless re-audited: final submission delivery is now admitted through independent one-slot metric/log lanes with one shared 100 ms deadline, while cleanup delivery remains independently capped and controller release remains one-slot and time-bounded. The current universal JSONSink regression and full race suite provide fresh post-repair evidence.
