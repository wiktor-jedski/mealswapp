# ARCH-004 Integration Verification Obligations

## Purpose

This document defines the Automotive SPICE SWE.5 integration verification obligations for `ARCH-004`, the Linear Programming Optimizer, after Phase 07.01 remediation. It verifies architecture behavior across saved-diet persistence, authenticated submission, durable idempotency, Redis Streams, worker orchestration, solver/validation, polling, frontend state, identity boundaries, safe errors, and observability.

The primary component under test is `ARCH-004`. Collaborations with `ARCH-005`, `ARCH-006`, `ARCH-007`, `ARCH-008`, `ARCH-010`, and `ARCH-014` are included only where they carry ARCH-004 data or control flow. Polling is the Phase 07.01 completion mechanism; the optional WebSocket interface is outside this phase.

## Component Information

| Field | Value |
| --- | --- |
| Architecture component | `ARCH-004` Linear Programming Optimizer |
| Architecture sources | `docs/architecture/01_SOFT_ARCH_DESIGN.md`, `docs/architecture/ARCH-004.md` |
| Code-design sources | `docs/design/DESIGN-004.md`, `docs/design/DESIGN-001.md`, `docs/design/DESIGN-008.md`, `docs/design/DESIGN-014.md`, `docs/design/DESIGN-017.md` |
| System under test | `JobStatusTracker`, `JobQueueManager`, `LPSolverWrapper`, `ConstraintBuilder`, `ObjectiveFunction`, `DiversityPenalizer`, `SolutionValidator` and their API/worker/frontend adapters |
| Requirements | `SW-REQ-006`, `SW-REQ-021`, `SW-REQ-022`, `SW-REQ-023`, `SW-REQ-030`, `SW-REQ-042`, `SW-REQ-043`, `SW-REQ-080`, `SW-REQ-082` |

## Integration Test Rules

- PostgreSQL, Redis, Fiber routing/middleware, worker orchestration, optimization pipeline, generated client/store/component code, and native CLP are real where practical.
- Test doubles are permitted only at explicit architecture boundaries: the solver runner for deterministic output/cancellation, entitlement/provider adapters, telemetry sink, browser HTTP boundary, or a processor when Redis delivery itself is under test.
- No obligation is satisfied by a single helper or by mock-call assertions. Evidence must verify exchanged data, durable/visible state, returned results, side effects, or recovery state across at least two collaborating units.
- Every cited test has an adjacent trace containing its `IT-ARCH-004-*` obligation, `ARCH-004`, applicable `DESIGN-*`, and `SW-REQ-*` IDs.

## Scenario Coverage Matrix

| Required Task 236 scenario | Obligation(s) | Principal evidence |
| --- | --- | --- |
| Nominal | 001, 002, 004, 006 | Live PostgreSQL/Redis/API/worker/CLP gate and browser workflow |
| Replay | 001, 004, 006, 007 | Immutable Daily Diet replay, normalized optimization replay, browser lost-response replay, telemetry outcomes |
| Replacement/deletion | 001, 004, 006 | Durable create response survives replacement/deletion; authoritative frontend replacement/deletion reconciliation |
| Concurrency | 001, 002, 003, 004, 007 | Cross-user API, concurrent validator, queue cleanup/bootstrap, capacity fixtures |
| Cancellation | 001, 003, 005, 006 | Submission cancellation, recoverable queue delivery, solver deadline, browser/store abort lifecycle |
| Solver output | 002, 004, 005 | Native CLP, malformed/duplicate output rejection, partial terminal output, timeout mapping |
| Distinct alternative | 002, 004 | Canonical meal-set exclusion, immutable snapshot, one-to-three validated alternatives |
| Queue loss/recovery | 003, 004, 007 | Group deletion, Redis restart/data loss, abandoned delivery, stream recovery |
| Malformed contract | 001, 002, 003, 006 | Durable acknowledgement, solver assignment, stream job ID, HTTP/client response decoding |
| Identity | 001, 004, 006, 008 | Owner-scoped submit/poll/expiry and logout/account-switch teardown |
| Observability | 003, 007 | Accurate queue populations/ages and bounded privacy-safe API/worker telemetry |
| Degraded collaboration | 003, 004, 005, 006, 007, 008 | Redis outage, failed cleanup/finalization, timeout/infeasible, malformed response, expiry |

## IT-ARCH-004-001 Saved-Diet Identity, Durable Replay, Submission, and Polling

### Intent

Verify that an authenticated entitled user submits an owner-scoped persisted Daily Diet, receives one durable asynchronous job acknowledgement, and polls only that user's job. Exact replay remains immutable after replacement/deletion or current entitlement/state changes; pending-publication repair revalidates current authority; cancellation and malformed durable contracts fail before side effects.

### System Under Test

`ARCH-004 JobStatusTracker` and `OptimizationController` collaborating with the `DESIGN-008 SavedDataRepository`, authenticated gateway, entitlement/admission adapters, durable idempotency repository, job store, and queue publisher.

### Real Components

- Fiber authentication, CSRF, routing, response mapping, and controller orchestration
- PostgreSQL Daily Diet and durable mutation/idempotency repositories in the live gate
- Redis admission, job state, queue publication, and owner-scoped polling in the live gate
- Dedicated worker/native CLP completion path in the live gate

### Allowed Test Doubles

- Deterministic controller repositories/admission/queue adapters for cancellation, cross-controller races, pending repair, and malformed persisted acknowledgement interleavings
- No persistence, Redis, worker, or solver double in the composed Task 232 nominal/replay path

### Trigger / Stimulus

Create and replay a Daily Diet; replace and delete its current aggregate; submit equivalent normalized optimization bodies with one key; race unrelated users and controllers; repair a pending acknowledgement; cancel before publication; poll as owner/other user; load malformed persisted acknowledgement data.

### Expected Integrated Behavior

1. Daily Diet replay returns the exact original `201` body and creates no replacement aggregate after current replacement/deletion.
2. Canonically equivalent optimization requests return the original `202`, job ID, `Location`, and poll URL with one queue publication; changed canonical input returns `409`.
3. Published replay performs no current entitlement, diet, admission, job-store, or queue side effect; pending repair revalidates these boundaries and publishes the original job once.
4. Unrelated users proceed independently; owner mismatch and cross-user polling disclose no job or diet data.
5. Cancellation or malformed durable acknowledgement produces no job/claim/admission/queue side effect and no fabricated `Location` fallback.

### Required Evidence and Implementing Tests

- `backend/internal/app/task232_backend_regression_test.go::TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers`
- `backend/internal/httpapi/task222_optimization_submission_integration_test.go::TestOptimizationHTTPDifferentUsersProceedIndependentlyAndSameKeyIsUserScoped`
- `backend/internal/httpapi/task222_optimization_submission_integration_test.go::TestOptimizationHTTPSubmissionHonorsRequestCancellation`
- `backend/internal/httpapi/task222_optimization_submission_integration_test.go::TestOptimizationHTTPPublishedAcknowledgementReplayHasNoCurrentStateSideEffects`
- `backend/internal/httpapi/task222_optimization_submission_integration_test.go::TestOptimizationHTTPUnpublishedRepairRevalidatesAndPublishesOnce`
- `backend/internal/httpapi/task222_optimization_submission_integration_test.go::TestOptimizationHTTPConcurrentPublishedRepairDoesNotStrandAdmission`
- `backend/internal/httpapi/task222_optimization_submission_integration_test.go::TestOptimizationHTTPRejectsMalformedPersistedAcknowledgementWithoutLocationFallback`

### Requirement Traceability

`SW-REQ-006`, `SW-REQ-021`, `SW-REQ-042`, `SW-REQ-043`, `SW-REQ-080`, `SW-REQ-082`.

## IT-ARCH-004-002 Repository Input, Solver Output, Validation, and Distinct Alternatives

### Intent

Verify that the worker derives authoritative targets from one immutable owner-scoped repository snapshot, builds the calorie-primary/diversity-secondary model, accepts only canonical solver output, projects each accepted solution once, and creates at most three genuinely distinct meal sets while preserving already valid partial results.

### System Under Test

`ARCH-004 ConstraintBuilder`, `ObjectiveFunction`, `DiversityPenalizer`, and `SolutionValidator`, collaborating with the repository input loader and `LPSolverWrapper` boundary.

### Real Components

- Repository meal/saved-diet domain values and canonical quantity conversion
- Constraint, objective, alternative-generation, canonicalization, validation, and projection implementations
- Native CLP in packaged/live nominal fixtures
- Worker/job-store publication in partial-result integration fixtures

### Allowed Test Doubles

- A deterministic `AlternativeSolveFunc` at the documented solver boundary for exact malformed, duplicate, partial-failure, and attempt-budget sequences
- No validation, projection, model, diversity, worker, or job-store replacement in the corresponding evidence

### Trigger / Stimulus

Run native and deterministic solves with metric/liquid data, exclusions, multiple valid alternatives, signed residue, unknown/negative/non-finite assignments, duplicate meal sets, later timeout/failure, attempt exhaustion, caller snapshot mutation, and concurrent projections.

### Expected Integrated Behavior

1. Targets and result nutrition are recalculated from current repository data and canonical units.
2. Calorie minimization remains primary; original-meal quantity is only a secondary tie-breaker.
3. Unknown, materially negative, non-finite, infeasible, duplicate, or malformed solver assignments never mutate accepted state or reach publication.
4. Every retained alternative has a distinct selected meal-ID set, valid exclusion/tolerance/quantity/cardinality constraints, authoritative similarity, and deterministic ordering.
5. Output is bounded to three alternatives; valid partial results survive later solve failure or exhausted attempts.
6. The immutable index is built once and is safe under caller mutation and concurrent projection.

### Required Evidence and Implementing Tests

- `backend/internal/app/task232_backend_regression_test.go::TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers`
- `backend/internal/app/task206_backend_integration_test.go::TestTask206BackendIntegrationGate`
- `backend/internal/worker/task210_swe5_integration_test.go::TestTask210WorkerPublishesPartialAlternativesOnLaterSolverFailure`
- `backend/internal/optimization/task220_pipeline_test.go::TestPublicAlternativeGeneratorsBuildOneIndexAndProjectEachResultOnce`
- `backend/internal/optimization/task220_pipeline_test.go::TestAlternativePipelineRejectsInvalidDuplicateBeforeStateMutation`
- `backend/internal/optimization/task220_pipeline_test.go::TestAlternativePipelineAttemptExhaustionReturnsValidPartialResults`
- `backend/internal/optimization/task220_pipeline_test.go::TestAlternativePipelineSnapshotIgnoresLaterCallerMutation`
- `backend/internal/optimization/task220_pipeline_test.go::TestSolutionValidatorConcurrentMetricAndLiquidProjection`

### Requirement Traceability

`SW-REQ-021`, `SW-REQ-022`, `SW-REQ-023`, `SW-REQ-030`.

## IT-ARCH-004-003 Queue Ownership, Retry, Cancellation, Malformed Delivery, and Loss Recovery

### Intent

Verify the Redis Streams API/worker collaboration uses one consumer group, logical-job deduplication, ownership-first attempts, explicit terminal publication, atomic finalization, recoverable cancellation, abandoned-delivery reclaim, malformed-delivery removal, and fail-closed recovery after group/data/process loss.

### System Under Test

`ARCH-004 JobQueueManager` collaborating with Redis Streams, embedded queue scripts, worker processor/terminal handler, and bounded telemetry.

### Real Components

- Redis `XADD`, `XREADGROUP`, `XAUTOCLAIM`, `XACK`, `XDEL`, pending/group metadata, and embedded Lua scripts
- Multiple real queue managers/consumers and terminal marker/lock/attempt keys
- Isolated real Redis process restart fixture when `redis-server` or the approved local image is available

### Allowed Test Doubles

- A processor/terminal handler at the worker boundary when queue delivery/finalization itself is under test
- Memory telemetry sink at the external observability boundary

### Trigger / Stimulus

Bootstrap concurrently; publish/reserve duplicates; cancel an owned processor; inject malformed UUID delivery; abandon/reclaim; exhaust retries; omit terminal publication; delete the consumer group/stream state; restart Redis; inject authorization/connectivity/cleanup failure.

### Expected Integrated Behavior

1. One logical job has one authoritative stream publication and processing owner; lock misses/duplicates do not consume attempts.
2. Cancellation leaves the delivery pending and reclaimable; three genuine failures require durable failed publication before acknowledgement.
3. Completed/failed finalization and duplicate cleanup are atomic, idempotent, and cannot invent a marker after zero acknowledgement.
4. Malformed/non-canonical jobs are rejected and removed from stream and pending state.
5. `NOGROUP` causes one bounded idempotent recovery and preserved/new work remains processable after group/data/restart loss.
6. Authorization/connectivity and contradictory publication fail closed; cleanup is bounded and observable without private labels.

### Required Evidence and Implementing Tests

- `backend/internal/queue/job_queue_integration_test.go::TestJobQueueConcurrentConsumersDoNotProcessDuplicateLogicalJob`
- `backend/internal/queue/job_queue_integration_test.go::TestJobQueueReclaimsAbandonedDeliveryWithXAUTOCLAIM`
- `backend/internal/queue/job_queue_integration_test.go::TestJobQueueRetriesAndTerminallyFailsAfterThreeAttempts`
- `backend/internal/queue/job_queue_integration_test.go::TestJobQueueCancellationReachesProcessorAndLeavesDeliveryRecoverable`
- `backend/internal/queue/job_queue_integration_test.go::TestTask224RejectsNonCanonicalJobIDsAndRemovesMalformedDeliveries`
- `backend/internal/queue/task225_queue_test.go::TestTask225RequiresExplicitTerminalPublication`
- `backend/internal/queue/task225_queue_test.go::TestTask225DistinctFinalizationAndZeroAckSemantics`
- `backend/internal/queue/task225_queue_test.go::TestTask225AtomicDuplicateCleanupUnderRace`
- `backend/internal/queue/task225_queue_test.go::TestTask225LiveManagerRecoversGroupAndDataLoss`
- `backend/internal/queue/task225_queue_test.go::TestTask225LiveManagerRecoversAfterRedisRestart`
- `backend/internal/queue/task225_queue_test.go::TestTask225AuthorizationAndConnectivityErrorsFailClosed`

### Requirement Traceability

`SW-REQ-021`, `SW-REQ-080`, `SW-REQ-082`.

## IT-ARCH-004-004 Composed Nominal, Replay, Concurrency, Ownership, and Degraded Service

### Intent

Verify the composed asynchronous service from saved-diet persistence through authenticated API, durable replay, Redis queue/store, worker/native CLP, and polling without synchronous fallback under dependency failure.

### System Under Test

`ARCH-004` as a composed service collaborating with `ARCH-005` persistence and `ARCH-010` gateway boundaries.

### Real Components

- PostgreSQL repositories/migrations and authenticated Fiber API
- Redis admission, job store, stream, consumer group, and worker
- Packaged native CLP `1.17.11` and production validation/publication path
- Owner-scoped polling/error projection

### Allowed Test Doubles

- An unreachable Redis endpoint for the explicit outage path
- No solver, persistence, queue, worker, or polling double in nominal/replay/concurrency paths

### Trigger / Stimulus

Persist users/meals/diets; replay a create after replacement/deletion; submit equivalent and conflicting optimization requests; repair pending publication; process concurrent users; poll across identities; run native alternatives; disconnect Redis.

### Expected Integrated Behavior

1. Nominal jobs return immediately, execute only in the worker, and complete with one-to-three validated alternatives.
2. Replay/repair uses one server job and one publication; replacement/deletion does not mutate the immutable Daily Diet replay.
3. Concurrent users receive distinct owner-scoped jobs without controller-wide serialization.
4. Cross-user access is indistinguishable from not-found.
5. Redis outage returns bounded retryable `503 queue_unavailable`, creates no synchronous solver path, and leaks no infrastructure diagnostic.

### Required Evidence and Implementing Tests

- `backend/internal/app/task232_backend_regression_test.go::TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers`
- `backend/internal/app/task206_backend_integration_test.go::TestTask206BackendIntegrationGate`
- `backend/internal/queue/job_queue_integration_test.go::TestJobQueueUnavailableDoesNotInvokeProcessor`

### Requirement Traceability

`SW-REQ-006`, `SW-REQ-021`, `SW-REQ-022`, `SW-REQ-023`, `SW-REQ-030`, `SW-REQ-043`, `SW-REQ-080`, `SW-REQ-082`.

## IT-ARCH-004-005 Whole-Job Deadline, Cancellation, Partial Failure, and Safe Finalization

### Intent

Verify that one bounded context covers repository load and all solver attempts, active child processing is canceled, shutdown cancellation remains recoverable, valid partial alternatives are preserved, and timeout/finalization failures produce only safe observable state.

### System Under Test

`ARCH-004 LPSolverWrapper`, `OptimizationProcessor`, `JobStatusTracker`, `JobQueueManager`, and polling/frontend failure consumers.

### Real Components

- Worker processor, shared deadline, finalization context, Redis state/queue, failure classifier, and polling projection
- Real queue cancellation/reclaim and real child-process termination fixtures
- Frontend controller/store retry behavior

### Allowed Test Doubles

- Context-aware solver runner or child executable at the documented external solver boundary to make deadline/cancellation deterministic
- Failing admission release adapter at its architecture boundary

### Trigger / Stimulus

Block solver/child execution until deadline; cancel submission/queue/worker ownership; fail a later solve after one valid alternative; fail admission release after timeout publication; poll and retry the terminal timeout.

### Expected Integrated Behavior

1. The child/solver is canceled and temporary state is cleaned within the configured deadline.
2. Worker-owned deadline publishes `solver_timeout` with canonical text and valid partial alternatives, then finalizes within visibility bounds.
3. Parent shutdown cancellation publishes no terminal state and leaves delivery reclaimable.
4. Cleanup/release failure cannot replace a durable primary outcome and remains bounded/privacy-safe.
5. Polling/browser output contains no process path, Redis URL, solver diagnostic, or stale alternatives; deliberate retry uses the defined fresh/replay policy.

### Required Evidence and Implementing Tests

- `backend/internal/app/task206_backend_integration_test.go::TestTask206TimeoutAndOwnershipGate`
- `backend/internal/worker/task210_swe5_integration_test.go::TestTask210WorkerPublishesPartialAlternativesOnLaterSolverFailure`
- `backend/internal/worker/optimization_processor_deadline_test.go::TestOptimizationProcessorAppliesOneDeadlineAcrossAlternativeSolves`
- `backend/internal/worker/optimization_processor_deadline_test.go::TestOptimizationProcessorLeavesShutdownCancellationPendingForRetry`
- `backend/internal/queue/job_queue_integration_test.go::TestJobQueueCancellationReachesProcessorAndLeavesDeliveryRecoverable`
- `backend/internal/worker/task234_regression_test.go::TestTask234SolverTimeoutAndAdmissionReleaseFailureKeepSafeFinalTelemetry`
- `frontend/tests/task233-frontend-gate.spec.ts::queue ambiguity reuses its key and terminal timeout rotates the next intentional submission`

### Requirement Traceability

`SW-REQ-021`, `SW-REQ-022`, `SW-REQ-030`, `SW-REQ-080`.

## IT-ARCH-004-006 Strict Frontend Contract, Authoritative Diet State, Retry, and Identity Lifecycle

### Intent

Verify that generated contracts, strict runtime decoders, Daily Diet/optimization stores, selected-diet coordination, retry controller, and real UI collaborate without admitting malformed payloads, stale replacement/deletion state, duplicate intent, or prior-identity artifacts.

### System Under Test

`ARCH-004 JobStatusTracker` as consumed through `DESIGN-001 SearchView` and `DESIGN-017 ErrorMessageMapper/RetryManager`.

### Real Components

- Generated DTO types, strict Daily Diet and optimization clients
- Daily Diet, selected-diet, search, and optimization stores/controllers
- `OptimizationWorkflow` and real browser DOM/accessibility/keyboard behavior

### Allowed Test Doubles

- Playwright HTTP route boundary supplying contract-shaped or deliberately malformed API responses
- No component, client decoder, store/controller, selection, rendering, or identity-lifecycle replacement

### Trigger / Stimulus

Lose/replay create response; replace/delete/reload selected diets; receive malformed collection/poll envelopes; submit and poll nominal/infeasible/timeout/queue states; unmount/remount; cancel delayed work; logout and switch account; use desktop/mobile themes and keyboard controls.

### Expected Integrated Behavior

1. Exact statuses/envelopes are decoded before state mutation; malformed/unsafe data never reaches store or DOM and recovery remains available.
2. One memory-only idempotency key survives only ambiguous intent; replacement installs server macros, deletion cannot resurrect selection, and new intent rotates ownership.
3. Polling is bounded and abortable; unmount/remount deliberately resumes an acknowledged job without duplicate submission.
4. Logout/account change aborts late work and clears job, results, errors, key, selected diet, polls, and prior-user artifacts.
5. Safe degraded messages, zero-to-three alternatives, keyboard operation, responsive themes, and serious/critical axe checks pass.

### Required Evidence and Implementing Tests

- `frontend/tests/task233-frontend-gate.spec.ts::lost create response replays one write, replace installs authoritative macros, and selected optimization is safe in both themes`
- `frontend/tests/task233-frontend-gate.spec.ts::malformed collection and optimization payloads fail closed and recover without rendering unsafe state`
- `frontend/tests/task233-frontend-gate.spec.ts::queue ambiguity reuses its key and terminal timeout rotates the next intentional submission`
- `frontend/tests/task233-frontend-gate.spec.ts::remount resumes acknowledged polling, then logout and account change clear every prior-user artifact`
- `frontend/tests/task233-frontend-gate.spec.ts::delayed Daily Diet hydration cannot cross logout and account change`
- `frontend/src/lib/stores/daily-diet.test.ts::delete/load serializes the read and cannot resurrect a confirmed deletion`
- `frontend/src/lib/stores/daily-diet.test.ts::reload, deletion, empty state, and identity clear reconcile authoritative selection`
- `frontend/src/lib/stores/optimization.test.ts::strict client rejection keeps malformed polling payloads out of store and rendering state`
- `frontend/src/lib/stores/optimization.test.ts::logout and account switch abort and clear job, results, errors, keys, polls, and retry intent`

### Requirement Traceability

`SW-REQ-006`, `SW-REQ-021`, `SW-REQ-030`, `SW-REQ-042`, `SW-REQ-043`, `SW-REQ-080`.

## IT-ARCH-004-007 Bounded Observability, Queue Accuracy, Concurrency, and Degradation

### Intent

Verify that submission/replay, queue populations/ages, retries/recovery, worker/solver outcomes, cleanup, result expiry, readiness, and endpoint capacity cross the observability boundary with accurate values and bounded privacy-safe dimensions.

### System Under Test

`ARCH-004` telemetry hooks collaborating with `DESIGN-014 MetricsCollector/LogAggregator`, queue metadata, worker lifecycle, readiness, and capacity verification.

### Real Components

- Optimization controller, queue, worker, processor, telemetry adapters, and readiness wiring
- Real Redis waiting/pending metadata, retries, final markers, group recovery, and heartbeat
- Concurrent submission/replay/poll fixture and deterministic capacity verifier

### Allowed Test Doubles

- `MemorySink`/discarding JSON sink at the external metrics/log backend boundary
- Deterministic timing/load fixture; no controller, queue state, worker telemetry hook, or label policy replacement

### Trigger / Stimulus

Run unrelated concurrent submissions and exact replays; poll active jobs; create waiting/pending mixtures; retry/exhaust/recover; timeout the solver; fail admission/queue/lock/solver cleanup; inject IDs, keys, URLs, bodies, and diagnostics into errors.

### Expected Integrated Behavior

1. Accepted/replayed/rejected and worker/solve/retry/outcome values exactly match final state.
2. Waiting and pending depths/ages come from authoritative Redis metadata, remain nonnegative, and become zero after recovery/finalization.
3. Metric labels use fixed allowlists; logs/metrics contain no user/diet/job/entry IDs, idempotency keys, request bodies, provider URLs, meal data, or solver diagnostics.
4. Cleanup failure does not block/replace primary response; Redis/worker degradation is visible through bounded outcomes/readiness.
5. Submission/replay/poll responsiveness remains below the documented two-second critical boundary in the focused capacity fixture.

### Required Evidence and Implementing Tests

- `backend/internal/httpapi/task234_observability_capacity_test.go::TestTask234ConcurrentSubmissionsReplayCleanupAndPollResponsiveness`
- `backend/internal/queue/task234_regression_test.go::TestTask234MixedAgesRetriesAndStreamRecoveryMatchFinalState`
- `backend/internal/worker/task234_regression_test.go::TestTask234ProductionWorkerQueueTelemetryIsWiredAndPrivacySafe`
- `backend/internal/worker/task234_regression_test.go::TestTask234SolverTimeoutAndAdmissionReleaseFailureKeepSafeFinalTelemetry`
- `backend/internal/observability/observability_test.go::TestOptimizationTelemetryUsesBoundedLabelsAndNoPrivateData`
- `backend/internal/queue/job_queue_integration_test.go::TestJobQueueStatsExposeDepthAndAge`
- `backend/internal/worker/worker_integration_test.go::TestOptimizationWorkerHeartbeatIsRefreshableAndRemovedOnStop`
- `scripts/test_verify_optimization_capacity.py`
- `scripts/verify-phase0701-observability-capacity.py`

### Requirement Traceability

`SW-REQ-080`, `SW-REQ-082`.

## IT-ARCH-004-008 Result Expiry and Cross-Identity Isolation

### Intent

Verify that terminal results expire at the `JobStatusTracker` TTL, the owner-independent marker preserves safe expired/not-found classification, and neither another identity nor stale frontend state can recover the prior result.

### System Under Test

`ARCH-004 RedisOptimizationJobStore`, polling ownership/error mapping, and frontend expired-result retry behavior.

### Real Components

- Redis job state, terminal transition script, result TTL, and owner marker
- Optimization polling controller and safe response mapper
- Frontend optimization store/controller and identity lifecycle

### Allowed Test Doubles

- Controller/browser HTTP boundary may deterministically present the production expired/not-found envelope after real Redis expiry is separately verified

### Trigger / Stimulus

Publish a completed result with a short TTL; wait for expiry; poll as owner/other user; retry; logout/switch identity with an acknowledged or completed job.

### Expected Integrated Behavior

1. Result data is unavailable after TTL while the safe owner-independent marker remains bounded.
2. Owner receives stable retryable expiry; another identity receives indistinguishable not-found and no ownership signal.
3. A retry creates fresh intent rather than polling the expired job.
4. Identity teardown removes stale result/error/job/key/poll state and ignores late completion.

### Required Evidence and Implementing Tests

- `backend/internal/worker/task210_swe5_integration_test.go::TestTask210RedisJobStoreExpiresResultsWithOwnerMarker`
- `backend/internal/httpapi/optimization_controller_test.go::TestOptimizationHTTPExpiryKeepsOwnerIsolation`
- `frontend/src/lib/stores/optimization.test.ts::expired results retry as a fresh submission instead of polling the expired job again`
- `frontend/tests/phase07-browser-acceptance.spec.ts::expired-result fixture presents the retryable expired state and no stale result`
- `frontend/tests/task233-frontend-gate.spec.ts::remount resumes acknowledged polling, then logout and account change clear every prior-user artifact`

### Requirement Traceability

`SW-REQ-006`, `SW-REQ-043`, `SW-REQ-080`.

## SWE.5 Checklist Execution

The mandatory skill checklist was evaluated against each obligation and its cited tests. `A` = architecture trace/behavior (Section 1), `I` = integration scope/data exchange (Section 2), `R` = real components/boundary-only doubles (Section 3), `B` = sequence/state/data/failure/recovery behavior (Section 4), `E` = observable state/result/payload/side-effect evidence (Section 5), `O` = bidirectional obligation/test traceability (Section 7), `L` = no SWE.4-only leakage (Section 8), and `C` = completion criteria (Section 9).

| Obligation | A | I | R | B | E | Nominal | Failure | Recovery | O | L | C |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `IT-ARCH-004-001` | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| `IT-ARCH-004-002` | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| `IT-ARCH-004-003` | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| `IT-ARCH-004-004` | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| `IT-ARCH-004-005` | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| `IT-ARCH-004-006` | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| `IT-ARCH-004-007` | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| `IT-ARCH-004-008` | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |

### Checklist Evidence Notes

- Sections 1–2: each obligation identifies `ARCH-004`, requirements, the SUT, at least two collaborating units, exchanged data, and an architectural sequence/state/failure/recovery outcome.
- Section 3: live PostgreSQL/Redis/CLP/browser evidence is cited where practical; doubles are limited to solver/provider/telemetry/browser/processor boundaries and no test mocks every collaborator.
- Sections 4–5: assertions cover HTTP results, exact envelopes, persisted rows, Redis stream/pending/terminal state, solver alternatives, UI/store state, metrics/log fields, cancellation, and recovery—not call counts alone.
- Section 6: every obligation includes nominal/failure/recovery when applicable; expiry has nominal publication followed by expiry/isolation/retry recovery.
- Section 7: every obligation has at least one adjacent source trace, and every specifically cited Task 236 test references at least one obligation.
- Section 8: isolated parser, validation, bounds, and helper tests are supporting evidence only; no obligation depends solely on them.
- Final sanity question: replacing all collaborators except one with mocks would remove the asserted database, Redis, worker, solver, HTTP, browser, or telemetry outcomes, so the cited integration evidence would not pass.

## Focused Verification Commands

```bash
bash scripts/start-services.sh
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/app ./internal/httpapi ./internal/optimization ./internal/queue ./internal/worker ./internal/observability -run 'Test(Task232|Task206|OptimizationHTTP(DifferentUsers|SubmissionHonors|PublishedAcknowledgement|UnpublishedRepair|ConcurrentPublishedRepair|RejectsMalformedPersisted)|PublicAlternative|AlternativePipeline|SolutionValidatorConcurrent|JobQueue|Task224|Task225|Task234|Task210|OptimizationProcessor|OptimizationWorker|OptimizationTelemetry)' -count=1
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/app ./internal/httpapi ./internal/optimization ./internal/queue ./internal/worker ./internal/observability -run 'Test(Task232|OptimizationHTTP(DifferentUsers|SubmissionHonors|PublishedAcknowledgement|UnpublishedRepair|ConcurrentPublishedRepair|RejectsMalformedPersisted)|PublicAlternative|AlternativePipeline|SolutionValidatorConcurrent|JobQueueCancellation|Task224|Task225|Task234)' -count=1
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/daily-diet-client.test.ts src/lib/api/optimization-client.test.ts src/lib/stores/daily-diet.test.ts src/lib/stores/optimization.test.ts src/lib/components/OptimizationWorkflow.test.ts
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/task233-frontend-gate.spec.ts --workers=2 --reporter=dot
python3 scripts/verify-phase0701-observability-capacity.py
python3 scripts/validate-task-list.py
python3 scripts/validate-traceability.py
python3 -c 'import scripts.check as c; checked,total=c.validate_requirements(); print(f"Requirement traceability passed: {checked}/{total}")'
```

Exact execution results, timestamps, hashes, skips/deviations, and the test-to-obligation verification ledger are recorded in `docs/implementation/preparation/task-236-preparation.md`. Task 236 does not alter the task-list status.

## Completion Decision

`ARCH-004` Phase 07.01 SWE.5 coverage is complete only while all eight obligations, cited test traces, focused commands, mandatory checklist entries, and both requirement/design traceability checks pass. Any failed mandatory item reopens the affected obligation; this document does not change Task 236 status.
