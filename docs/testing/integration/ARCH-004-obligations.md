# ARCH-004 Integration Verification Obligations

## Purpose

This document defines the SWE.5 integration verification obligations for ARCH-004, the Linear Programming Optimizer. The obligations verify collaboration across the saved-diet aggregate, repository meal data, authenticated entitlement gateway, Redis Streams queue, dedicated worker, native CLP boundary, polling API, frontend workflow, safe failure mapping, and bounded observability.

The Phase 07 client completion mechanism is polling. The WebSocket notification interface remains optional and is not part of these obligations.

## Component Information

| Field | Value |
| --- | --- |
| Architecture Component | ARCH-004 |
| Name | Linear Programming Optimizer |
| Source Documents | `docs/architecture/ARCH-004.md`, `docs/architecture/01_SOFT_ARCH_DESIGN.md`, `docs/design/DESIGN-004.md`, `docs/design/DESIGN-001.md`, `docs/design/DESIGN-008.md`, `docs/design/DESIGN-014.md` |
| Related Units | SavedDataRepository, MealRepository, EntitlementRepository, auth/CSRF gateway, OptimizationController, JobStatusTracker, JobQueueManager, RepositoryOptimizationInputLoader, ConstraintBuilder, SolutionValidator, LPSolverWrapper, optimization worker, generated API client, OptimizationWorkflow, optimization store, OptimizationTelemetry |
| Collaborating Architecture | ARCH-005, ARCH-006, ARCH-007, ARCH-008, ARCH-010, ARCH-014 |
| Related Requirements | SW-REQ-006, SW-REQ-021, SW-REQ-022, SW-REQ-023, SW-REQ-030, SW-REQ-042, SW-REQ-043, SW-REQ-080, SW-REQ-082 |

## Integration Test Conventions

- Tests use real PostgreSQL, Redis, and the installed native CLP executable where those boundaries are available.
- Deterministic tests may replace only an external boundary: the CLP runner, Stripe-like entitlement source, or browser HTTP responses. They do not replace the ARCH-004 orchestration, queue, job-state, or polling units under test.
- Redis stream payloads contain only the server-created logical job ID. Job input and authoritative terminal state remain in the Redis job store and are reloaded by the worker.
- Every test listed below carries an `IT-ARCH-004-*`, `ARCH-004`, `DESIGN-*`, and `SW-REQ-*` trace in its source file or test comment.

## IT-ARCH-004-001 Authenticated Saved-Diet Submission and User-Scoped Polling

### Intent

Verify that a signed-in trial or paid user can select a persisted Daily Diet, submit an optimization request through the authenticated gateway, and poll a server-owned job without trusting client-supplied diet totals or exposing another user's job.

### System Under Test

ARCH-004 JobStatusTracker and OptimizationController at the authenticated API boundary.

### Real Components

- Fiber authentication, CSRF, and entitlement middleware
- OptimizationController and JobStatusTracker response mapping
- Daily Diet repository contract and server-owned job envelope
- Idempotency boundary and generated polling response shape

### Allowed Test Doubles

- Controller-level repository/queue doubles for deterministic denial, idempotency, and response-state interleavings.
- The complete backend gate uses real PostgreSQL, Redis, and the worker path for the end-to-end version of this obligation.

### Trigger / Stimulus

An authenticated user submits a saved-diet optimization request containing only the saved-diet ID, tolerance, and exclusions, then polls the job as the owner and as a different authenticated user. A legacy request containing client target macros is rejected before side effects. Anonymous and free-tier users also attempt submission.

### Expected Integrated Behavior

1. Anonymous requests are rejected before a job or queue side effect.
2. An active trial or paid entitlement is accepted with `202 Accepted`, a server-created job ID, and a poll URL.
3. The saved diet is reloaded under the authenticated owner and current repository meal data derives the authoritative target.
4. Poll responses transition monotonically through queued/processing/completed or failed.
5. A different user receives an indistinguishable not-found response and no job data.

### Required Evidence

- HTTP submission, entitlement, ownership, and polling assertions.
- The real saved-diet-to-worker path asserts that incorrect client totals do not alter the result.

### Requirement Traceability

- SW-REQ-006
- SW-REQ-042
- SW-REQ-043
- SW-REQ-021

### Verification Status

Implemented by:

- `backend/internal/httpapi/optimization_controller_test.go::TestOptimizationHTTPSubmissionAndPolling`
- `backend/internal/httpapi/optimization_controller_test.go::TestOptimizationHTTPEntitlementAndOwnershipGuards`
- `backend/internal/httpapi/optimization_controller_test.go::TestOptimizationHTTPAnonymousSubmissionIsDeniedBeforeSideEffects`
- `backend/internal/app/task206_backend_integration_test.go::TestTask206BackendIntegrationGate`
- `frontend/tests/phase07-browser-acceptance.spec.ts::paid fixture completes the keyboard-only meal selection, save, polling, alternatives, axe, responsive, and screenshot path`
- `frontend/tests/phase07-browser-acceptance.spec.ts::anonymous fixture gives sign-in guidance without protected Daily Diet requests`
- `frontend/tests/phase07-browser-acceptance.spec.ts::free fixture shows entitlement guidance and disables Daily Diet save and optimization`

Status: PASS.

## IT-ARCH-004-002 Repository Meal Data, Worker/Solver, Validation, and Partial Results

### Intent

Verify that the dedicated worker reloads the saved diet and eligible repository meals, builds the LP input from server data, invokes the solver boundary, validates recalculated alternatives, and publishes valid partial alternatives when a later solve fails.

### System Under Test

ARCH-004 worker orchestration: RepositoryOptimizationInputLoader, ConstraintBuilder, SolutionValidator, LPSolverWrapper, and JobStatusTracker.

### Real Components

- RedisOptimizationJobStore
- JobQueueManager delivery and terminal acknowledgement
- OptimizationProcessor
- ConstraintBuilder and SolutionValidator
- Redis state transition script

### Allowed Test Doubles

- Injected solver implementation for a deterministic first-valid/second-failure sequence. The solver remains at the explicit LPSolverWrapper architecture boundary.
- The complete backend gate uses the packaged native CLP executable for the nominal path.

### Trigger / Stimulus

A queued job is reserved from Redis. The first solve produces one valid alternative and the next solve returns a timeout.

### Expected Integrated Behavior

1. The worker reads job identity and owner-scoped input from job state rather than stream payload meal data.
2. Repository macro values are used to validate and recalculate the alternative.
3. The first valid alternative is retained when a later alternative solve fails.
4. The job is published as failed with the stable `solver_timeout` code and no solver diagnostic.
5. The queue delivery is acknowledged only after authoritative terminal publication.

### Required Evidence

- Real Redis stream reservation, job state, worker processor, validation, partial result, and terminal acknowledgement assertions.
- Nominal native-CLP evidence also verifies macro matching, calorie ordering, exclusions, diversity, and the three-result ceiling.

### Requirement Traceability

- SW-REQ-021
- SW-REQ-022
- SW-REQ-023
- SW-REQ-030

### Verification Status

Implemented by:

- `backend/internal/worker/task210_swe5_integration_test.go::TestTask210WorkerPublishesPartialAlternativesOnLaterSolverFailure`
- `backend/internal/app/task206_backend_integration_test.go::TestTask206BackendIntegrationGate`
- `backend/internal/worker/worker_integration_test.go::TestRunPublishesAlternativeBeforeAcknowledgingQueuedJob`
- `backend/internal/optimization/validator_test.go::TestGenerateValidatedAlternativesPreservesValidPartialResultsAfterLaterSolveFails`

Status: PASS.

## IT-ARCH-004-003 Redis Streams Delivery, Idempotency, Retry, and Worker Recovery

### Intent

Verify that API/worker queue participants use one Redis Stream consumer group with at-least-once delivery, logical-job deduplication, bounded retries, abandoned-delivery recovery, and terminal acknowledgement.

### System Under Test

ARCH-004 JobQueueManager and its collaboration with the worker processor boundary.

### Real Components

- Redis 7 Streams
- `XADD`, `XREADGROUP`, `XAUTOCLAIM`, `XACK`, and Redis scripts
- Multiple JobQueueManager consumers
- Queue stats and pending-delivery state

### Allowed Test Doubles

- A counting processor may stand in for the worker's solver orchestration when the queue behavior itself is the system under test.

### Trigger / Stimulus

Bootstrap the queue repeatedly, enqueue the same logical job from concurrent callers, reserve it with multiple consumers, abandon a delivery, and force processing failures through the retry budget.

### Expected Integrated Behavior

1. Stream/group bootstrap is idempotent and creates one consumer group.
2. Repeated publication of one logical job returns one stream entry.
3. Concurrent consumers process one authoritative logical job and acknowledge duplicate deliveries safely.
4. `XAUTOCLAIM` returns abandoned work with an incremented attempt.
5. The third failed attempt invokes terminal handling and acknowledges the delivery.
6. Queue depth, pending depth, and entry age remain observable without payload or user data.

### Required Evidence

- Real Redis integration assertions over stream entries, group state, pending state, attempt count, and terminal state.

### Requirement Traceability

- SW-REQ-021
- SW-REQ-080
- SW-REQ-082

### Verification Status

Implemented by:

- `backend/internal/queue/job_queue_integration_test.go::TestJobQueueBootstrapIsIdempotentAndUsesOneConsumerGroup`
- `backend/internal/queue/job_queue_integration_test.go::TestJobQueueEnqueueReserveAndAckUseRedisStreams`
- `backend/internal/queue/job_queue_integration_test.go::TestJobQueueEnqueueIsIdempotentPerLogicalJob`
- `backend/internal/queue/job_queue_integration_test.go::TestJobQueueConcurrentConsumersDoNotProcessDuplicateLogicalJob`
- `backend/internal/queue/job_queue_integration_test.go::TestJobQueueReclaimsAbandonedDeliveryWithXAUTOCLAIM`
- `backend/internal/queue/job_queue_integration_test.go::TestJobQueueRetriesAndTerminallyFailsAfterThreeAttempts`
- `backend/internal/worker/worker_integration_test.go::TestRedisOptimizationJobStoreTerminalTransitionsAreAtomic`

Status: PASS.

## IT-ARCH-004-004 End-to-End Nominal, Infeasible, Duplicate, Outage, and Ownership Collaboration

### Intent

Verify the complete saved-diet-to-alternative path across PostgreSQL persistence, repository meal data, entitlement/authentication, API submission, Redis queue, worker, native solver, polling, and duplicate/outage/error paths.

### System Under Test

ARCH-004 as a composed asynchronous service, with ARCH-005 persistence and ARCH-010 gateway boundaries.

### Real Components

- PostgreSQL saved-diet and meal repositories
- Authenticated Fiber API and entitlement repository
- Redis Streams queue and Redis job store
- Dedicated worker and native CLP executable
- OptimizationController polling API

### Allowed Test Doubles

- An unreachable Redis endpoint stands in for the queue outage condition.
- No solver or persistence double is used in the nominal, infeasible, duplicate, and ownership paths.

### Trigger / Stimulus

Register two users, grant one an active trial entitlement, persist meals and a Daily Diet, submit optimization jobs, run the worker, poll completion, redeliver a completed job, submit an infeasible request, and submit through an unavailable Redis endpoint.

### Expected Integrated Behavior

1. A saved Daily Diet is persisted and reloaded with current server meal data.
2. Nominal jobs complete with one to three validated alternatives, matching macros, ordered calories, exclusions, and distinct meal sets.
3. Infeasible constraints become a safe terminal failure.
4. A duplicate delivery does not rerun the processor or alter authoritative alternatives.
5. Cross-user polling does not disclose data.
6. Redis outage returns `503 queue_unavailable` and never invokes synchronous solving.

### Required Evidence

- The test must cross the PostgreSQL, Redis, API, worker, solver, and polling boundaries in one fixture.
- Assertions must include status, result values, ownership, queue side effects, and safe failure mapping.

### Requirement Traceability

- SW-REQ-006
- SW-REQ-021
- SW-REQ-022
- SW-REQ-023
- SW-REQ-030
- SW-REQ-080

### Verification Status

Implemented by `backend/internal/app/task206_backend_integration_test.go::TestTask206BackendIntegrationGate`.

Status: PASS.

## IT-ARCH-004-005 Solver Timeout, Cancellation, and Safe Error Boundary

### Intent

Verify that the worker's solver boundary enforces a deadline, maps timeout/infeasible outcomes to safe public failures, and keeps the API process free of synchronous solver work.

### System Under Test

ARCH-004 LPSolverWrapper and worker-to-JobStatusTracker failure collaboration.

### Real Components

- PostgreSQL saved-diet and meal repositories
- Redis job state and stream delivery
- OptimizationProcessor
- LPSolverWrapper timeout and failure mapping
- Polling response mapper

### Allowed Test Doubles

- A context-aware child-process runner shortens the integration fixture's wait while exercising the production wrapper deadline/cancellation boundary.

### Trigger / Stimulus

A valid queued optimization job reaches a solver runner that blocks until its bounded context is canceled.

### Expected Integrated Behavior

1. The worker cancels the solver within the configured deadline.
2. The job is terminally published with `solver_timeout` and a user-safe message.
3. Polling exposes no child-process diagnostic or internal path.
4. No synchronous solver fallback occurs in the API process.

### Required Evidence

- A real repository/Redis/processor path must reach the injected timeout runner.
- HTTP failure mapping and frontend timeout retry must be verified separately at their boundaries.

### Requirement Traceability

- SW-REQ-021
- SW-REQ-022
- SW-REQ-080

### Verification Status

Implemented by:

- `backend/internal/app/task206_backend_integration_test.go::TestTask206TimeoutAndOwnershipGate`
- `backend/internal/httpapi/optimization_controller_test.go::TestOptimizationHTTPFailedPollingUsesSafeSolverMessages`
- `frontend/tests/phase07-browser-acceptance.spec.ts::timeout fixture supports a keyboard retry with a new safe submission`
- `frontend/src/lib/stores/optimization.test.ts::projects queue-unavailable and solver-timeout messages without exposing infrastructure details`

Status: PASS.

## IT-ARCH-004-006 Frontend Saved-Diet Selection, Polling, Retry, and Terminal Workflow

### Intent

Verify that the frontend's saved-diet selection, generated optimization client, bounded polling store, workflow component, and accessible result/error presentation collaborate with the ARCH-004 HTTP contract.

### System Under Test

ARCH-004 JobStatusTracker as consumed by the DESIGN-001 SearchView OptimizationWorkflow.

### Real Components

- Daily Diet collection and selection UI
- OptimizationWorkflow component
- generated API DTOs and optimization client
- optimization store/controller
- keyboard and accessibility behavior

### Allowed Test Doubles

- Playwright route interception may provide API-compatible authenticated, entitlement, queue, worker, and terminal responses. The browser renders the real frontend components, stores, and generated clients.

### Trigger / Stimulus

Paid/trial users select a saved Daily Diet, submit by keyboard, observe queued/processing progress, receive alternatives, retry an ambiguous submission, change diets, and encounter infeasible, timeout, expired, anonymous, and free-tier states.

### Expected Integrated Behavior

1. The request contains the selected server diet ID and server-projected macro target fields.
2. One idempotency key is reused only for an ambiguous retry; a deliberate fresh retry receives a new key.
3. Polling uses bounded backoff, stops at terminal state, and is canceled on unmount or diet change.
4. The UI renders zero to three validated alternatives and never leaves stale results after a diet change or expiry.
5. Safe queue, timeout, infeasible, expiry, anonymous, and entitlement messages are keyboard-accessible and do not expose infrastructure details.

### Required Evidence

- Frontend unit tests for client/store/controller behavior.
- Playwright/axe assertions for the real workflow, responsive layout, keyboard operation, and terminal states.

### Requirement Traceability

- SW-REQ-006
- SW-REQ-021
- SW-REQ-030
- SW-REQ-042
- SW-REQ-080

### Verification Status

Implemented by:

- `frontend/src/lib/api/optimization-client.test.ts`
- `frontend/src/lib/stores/optimization.test.ts`
- `frontend/src/lib/components/OptimizationWorkflow.test.ts`
- `frontend/tests/optimization-workflow.spec.ts`
- `frontend/tests/phase07-browser-acceptance.spec.ts`

Status: PASS.

## IT-ARCH-004-007 Safe Errors, Queue/Worker Observability, and Capacity Evidence

### Intent

Verify that submission, queue, worker, solver, terminal, retry, expiry, readiness, and capacity signals cross the ARCH-004 boundaries using bounded labels and sanitized operational fields only.

### System Under Test

ARCH-004 observability collaboration with DESIGN-014 MetricsCollector/LogAggregator and the API readiness surface.

### Real Components

- OptimizationController telemetry hooks
- JobQueueManager queue-depth/age reporting
- worker heartbeat and solve/job outcome telemetry
- Redis job-store expiry telemetry
- readiness checks and capacity evidence script
- MemorySink/JSONSink observability adapters

### Allowed Test Doubles

- MemorySink and deterministic capacity fixtures may stand in for the external metrics backend.

### Trigger / Stimulus

Emit accepted, rejected, queue, worker, solve, timeout/infeasible, retry, and result-expiry events; collect queue/readiness samples; attempt to inject user, diet, and job identifiers into telemetry.

### Expected Integrated Behavior

1. Metrics and logs use fixed low-cardinality labels and bounded statuses.
2. User IDs, diet IDs, job IDs, meal data, Redis URLs, and solver diagnostics do not cross the telemetry boundary.
3. Queue depth/age, active worker/utilization, solve duration/status, retries, outcomes, and result expiry are observable.
4. Redis/worker degradation is reflected in readiness and fail-closed capacity evidence.

### Required Evidence

- Telemetry assertions, queue stats, worker heartbeat lifecycle, readiness tests, and deterministic capacity-gate tests.
- The capacity document records the local evidence scope for SW-REQ-080 and SW-REQ-082.

### Requirement Traceability

- SW-REQ-080
- SW-REQ-082

### Verification Status

Implemented by:

- `backend/internal/observability/observability_test.go::TestOptimizationTelemetryUsesBoundedLabelsAndNoPrivateData`
- `backend/internal/queue/job_queue_integration_test.go::TestJobQueueStatsExposeDepthAndAge`
- `backend/internal/worker/worker_integration_test.go::TestOptimizationWorkerHeartbeatIsRefreshableAndRemovedOnStop`
- `backend/internal/httpapi/optimization_controller_test.go::TestOptimizationHTTPSubmissionAndPolling`
- `scripts/test_verify_optimization_capacity.py`
- `scripts/verify-optimization-capacity.py`

Status: PASS.

## IT-ARCH-004-008 Result TTL Expiry and Cross-User Expiration Isolation

### Intent

Verify that completed results expire according to the JobStatusTracker TTL, that an owner-independent marker preserves the stable expired/not-found classification, and that expiry cannot disclose ownership to another user.

### System Under Test

ARCH-004 RedisOptimizationJobStore and polling ownership/error mapping.

### Real Components

- Redis job-state store and terminal transition script
- Result TTL and owner marker keys
- OptimizationController poll mapping

### Allowed Test Doubles

- Controller-level store fixtures may deterministically represent an expired owner marker while preserving the production HTTP/error mapping.

### Trigger / Stimulus

Publish a completed job with a short integration TTL, wait for the result key to expire, and poll it as the owner and another authenticated user.

### Expected Integrated Behavior

1. The completed result is unavailable after its TTL.
2. The store returns an expired classification that remains `ErrOptimizationJobNotFound` compatible.
3. The owner marker does not expose the owner through the API.
4. Owner and cross-user polls return safe, stable responses and the frontend offers a fresh retry without stale results.

### Required Evidence

- Real Redis expiry and owner-marker assertion.
- HTTP owner/cross-user mapping and browser expired-result assertions.

### Requirement Traceability

- SW-REQ-006
- SW-REQ-043

### Verification Status

Implemented by:

- `backend/internal/worker/task210_swe5_integration_test.go::TestTask210RedisJobStoreExpiresResultsWithOwnerMarker`
- `backend/internal/httpapi/optimization_controller_test.go::TestOptimizationHTTPExpiryKeepsOwnerIsolation`
- `frontend/tests/phase07-browser-acceptance.spec.ts::expired-result fixture presents the retryable expired state and no stale result`
- `frontend/src/lib/stores/optimization.test.ts::expired results retry as a fresh submission instead of polling the expired job again`

Status: PASS.

## SWE.5 Checklist Evaluation

| Obligation | ARCH Trace | DESIGN Trace | SW-REQ Trace | Two or more collaborating units | Real boundary evidence | Nominal/failure/recovery as applicable | Decision |
| --- | --- | --- | --- | --- | --- | --- | --- |
| IT-ARCH-004-001 | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| IT-ARCH-004-002 | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| IT-ARCH-004-003 | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| IT-ARCH-004-004 | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| IT-ARCH-004-005 | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| IT-ARCH-004-006 | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| IT-ARCH-004-007 | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| IT-ARCH-004-008 | PASS | PASS | PASS | PASS | PASS | PASS | PASS |

## Required Focused Verification

The Task 210 focused gate is:

```bash
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/app ./internal/httpapi ./internal/queue ./internal/worker ./internal/observability -run 'Test(Task206|OptimizationHTTP|JobQueue|RunPublishes|RedisOptimization|OptimizationWorker|OptimizationTelemetry|Task210)' -count=1
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/optimization-client.test.ts src/lib/stores/optimization.test.ts src/lib/components/OptimizationWorkflow.test.ts
python3 -m unittest scripts/test_verify_optimization_capacity.py
python3 scripts/validate-task-list.py
python3 scripts/validate-traceability.py
```

The real PostgreSQL/Redis/CLP gate requires local services and `/usr/bin/clp` (or `MEALSWAPP_CLP_EXECUTABLE`) with the supported `1.17.11` version. Browser evidence is run with the Phase 07 Playwright configuration and uses API-compatible worker fixtures at the frontend boundary.

## Completion Decision

ARCH-004 SWE.5 is complete when every obligation above has a passing implementation, all listed traceability IDs remain present, the focused backend/frontend/script checks pass, and both repository validators pass. No task-list row is changed by this document.
