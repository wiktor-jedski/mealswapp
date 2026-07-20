# Task 234 Preparation — Observability and Capacity Regression Gate

## Scope and attribution

- Task: `234`, Phase 07.01 Observability and Capacity Regression Gate.
- Authoritative static aspects: `DESIGN-014: MetricsCollector` and `LogAggregator`; queue and worker fixture collaborations retain their existing `DESIGN-004` ownership.
- The Task 234 row in `docs/implementation/02_TASK_LIST.md` remains `OPEN`. This work did not edit the task list or any status.
- Preparation date: 2026-07-18 (Europe/Warsaw).
- The worktree already contained extensive concurrent Phase 07.01 work, including the Task 223/226 observability implementation and queue/worker/solver repairs. Those changes were preserved. The rejected-review repair attribution is limited to durable-terminal confirmation in `RedisOptimizationJobStore`, worker queue telemetry composition in `RunWithProcessorAndTelemetry`/`runWithProcessor` and `cmd/worker/main.go`, the Task 234 worker regressions, and the corrected opposite-terminal race assertion. Earlier Task 234 attribution in `backend/internal/observability/optimization.go` remains limited to `validOptimizationEvent`, generic serialized fallback reporting in `reportSinkFailure`, and `optimizationFallbackMu`.
- No frontend, OpenAPI, repository, migration, task-status, or unrelated review/open-point surface was changed for Task 234.

## Sources and prior evidence read

The Task 234 row, `docs/implementation/reviews/task-234-review.md`, `docs/design/DESIGN-014.md`, `docs/design/DESIGN-004.md`, and their architecture sources were read in full. Current submission, queue, worker, solver, metric, and log implementation/tests were traced through:

- `backend/internal/observability/observability.go`, `observability_test.go`, `optimization.go`, `task223_optimization_test.go`, and `task226_queue_age_test.go`;
- `backend/internal/httpapi/optimization_controller.go`, `optimization_controller_test.go`, `task222_optimization_submission_integration_test.go`, and `task223_submission_observability_test.go`;
- `backend/internal/queue/job_queue.go`, `job_queue_integration_test.go`, `task225_queue_test.go`, and `task226_queue_age_test.go`;
- `backend/internal/worker/optimization_processor.go`, `optimization_processor_deadline_test.go`, `worker.go`, and `worker_integration_test.go`;
- `backend/internal/optimization/clp_wrapper.go` and `clp_wrapper_test.go`;
- `scripts/verify-optimization-capacity.py`, `scripts/test_verify_optimization_capacity.py`, `scripts/check.py`, and `docs/implementation/phase07-task-208-capacity.md`.

Prior prerequisite evidence was used as a regression inventory rather than copied:

| Evidence | Relevant inherited contract | SHA-256 read |
|---|---|---|
| `docs/implementation/preparation/task-223-preparation.md` | typed final submission outcomes, bounded admission cleanup, privacy-safe labels, noncooperative sink/release bounds | `45570f5a43d91144666280e3830f7a41be6208827e1fcc1aa14a125437c222a3` |
| `docs/implementation/preparation/task-226-preparation.md` | separate waiting/pending populations, authoritative ages, fixed queue units/kinds, nonnegative skew | `7648b4629aad277ac1633769839cba7f87408242bc659a4e5917e24de04051d9` |
| `docs/implementation/preparation/task-232-preparation.md` | real PostgreSQL/Redis/API/worker/CLP regression inventory and recovery boundaries | `e635320f1020d8272d4647d0b1e4438c05c291b79b0a70c2450a684f42748099` |

## Implemented regression surfaces

### Exact files, symbols, and tests

| Path | Exact Task 234 symbols/tests | Purpose |
|---|---|---|
| `backend/internal/observability/optimization.go` | `validOptimizationEvent`; `reportSinkFailure`; `optimizationFallbackMu` | Enforce exact message-specific field shapes and bounded string vocabularies; never copy sink errors into fallback logs; serialize concurrent metric/log fallback writes. |
| `backend/internal/observability/task234_regression_test.go` | `TestTask234TelemetryRejectsIdentifiersAndSanitizesSinkFailures`; `task234FailingSink` | Reject identifier/PII/body/diagnostic values in outcomes, labels, retry status, and solve status; prove exactly one generic metric and log fallback record under failure. |
| `backend/internal/httpapi/task234_observability_capacity_test.go` | `TestTask234ConcurrentSubmissionsReplayCleanupAndPollResponsiveness`; `assertTask234CapacityWindow`; `runTask234Concurrently` | Eight unrelated concurrent submissions, eight exact same-key acknowledgement replays, no duplicate publication, exact accepted/replayed metrics, bounded admission-release failure, and submission/replay/poll batch responsiveness below 2 seconds while processing jobs are active. |
| `backend/internal/queue/task234_regression_test.go` | `TestTask234MixedAgesRetriesAndStreamRecoveryMatchFinalState` | Real-Redis one-waiting/one-pending mixture, positive authoritative ages, emitted values equal returned state, three attempts with exact retry outcomes, failed/completed final markers, group recovery, and final empty zero-age queue. |
| `backend/internal/worker/optimization_processor.go` | `RedisOptimizationJobStore.PublishCompleted`; `RedisOptimizationJobStore.PublishFailed`; `requireDurableTerminalStatus` | Treat a rejected terminal compare-and-set as success only when a reload proves the requested durable terminal status; reject queued and opposite-terminal state so no terminal queue publication can precede durable state. |
| `backend/internal/worker/worker.go` | `RunWithProcessorAndTelemetry`; `runWithProcessor` | Attach the caller's bounded telemetry instance to the queue manager created by the actual worker runtime while preserving the existing telemetry-free compatibility entry point. |
| `backend/cmd/worker/main.go` | production `RunWithProcessorAndTelemetry` composition | Pass the same production telemetry instance used by the job store and processor into the worker queue runtime. |
| `backend/internal/worker/task234_regression_test.go` | `TestTask234QueuedDurableJobRemainsPendingWhenProcessingAndFinalizationFail`; `TestTask234TerminalPublicationRequiresProcessingAndMatchesDurableState`; `TestTask234ProductionWorkerQueueTelemetryIsWiredAndPrivacySafe`; existing timeout/release fixture; supporting timeout store and Redis hook | Prove queued processing-timeout finalization cannot ACK, processing can reach each durable terminal state, opposite terminal publication fails closed, and production queue cleanup telemetry retains exact bounded name/value/unit/label/event shapes without IDs, PII, keys, entries, or sink diagnostics. |
| `backend/internal/worker/worker_integration_test.go` | `TestRedisOptimizationJobStoreTerminalTransitionsAreAtomic` assertion | Require one winner and one rejected conflict when completed and failed publication race, while retaining the first durable terminal result. |
| `scripts/verify-optimization-capacity.py` | `submit_and_poll`; `capacity_gate_passes`; report `replay` section | Exact-replay every accepted key/body, compare acknowledgement data without reporting it, enforce replay count/P95, and retain only aggregate privacy-safe evidence. |
| `scripts/test_verify_optimization_capacity.py` | `test_gate_rejects_same_key_replay_drift`; `test_submit_and_poll_replays_same_key_without_reporting_identifiers`; updated `valid_report` | Prove replay drift fails and no key, cookie, body-derived identifier, job ID, or poll URL enters result evidence. |
| `scripts/verify-phase0701-observability-capacity.py` | `run`; `main` | One fail-closed normal/race gate across Task 234, queue cleanup, solver cleanup, child timeout, and isolated Redis restart; required skips and empty test selection fail. |
| `docs/implementation/phase07.01-task-234-capacity.md` | deterministic and authenticated profiles | Fixed workload, thresholds, privacy contract, commands, and production-sizing limitation. |
| `docs/implementation/phase07-task-208-capacity.md` | updated repeatable-check contract | Align prior operational evidence with exact same-key replay, acknowledgement matching, and replay P95. |

### Acceptance mapping

| Task 234 criterion | Exact evidence | Result |
|---|---|---|
| Concurrent unrelated submissions | HTTP fixture issues eight distinct keys concurrently and receives eight `202` acknowledgements with eight queue publications | PASS |
| Same-key replay | The same eight keys/bodies replay concurrently to the original job acknowledgements with zero additional queue publications; live checker now repeats every accepted key | PASS |
| Admission release failure | HTTP cleanup fixture preserves bounded `503 queue_unavailable`; worker fixture preserves already-published `solver_timeout` while returning release failure for retry | PASS |
| Waiting/pending mixture and ages | Real Redis reports depth 2, pending depth 1, positive separate ages; six emitted queue metrics exactly equal mixed then final zero state | PASS |
| Retries | Real Redis records attempts 1–3 and exact telemetry sequence `retry`, `retry`, `exhausted`, then failed marker and no pending delivery | PASS |
| Solver timeout | Deadline solver emits two completed solves, one timeout solve, and one timeout job outcome; durable failure code is `solver_timeout` | PASS |
| Cleanup failure | Focused gate includes queue lock cleanup failure and solver-directory cleanup with primary-result preservation; the production worker runtime fixture proves its queue manager receives the bounded telemetry adapter; child timeout confirms unconditional directory/process cleanup | PASS |
| Stream recovery | Task 234 recovers a destroyed consumer group and completes waiting work; gate also requires the isolated Redis restart recovery fixture | PASS |
| Final outcomes and ages | HTTP accepted/replayed counts, Redis failed/completed markers, worker timeout state, and mixed/final queue metric values are asserted exactly; queued/pre-processing timeout and rejected finalization retain one pending delivery, while processing publication is durable before the queue can return a terminal marker | PASS |
| Bounded privacy-safe labels/logs | Message-specific field validation and all four fixtures reject or serialize-check PII, IDs, keys, bodies, stream entries, provider/solver diagnostics, and extra labels | PASS |
| Endpoint responsiveness | Original submission, exact replay, and active-worker poll batches each complete below DESIGN-014's 2-second critical boundary; authenticated checker enforces original/replay/poll P95 `< 2s` | PASS |
| Focused documented gate | `python3 scripts/verify-phase0701-observability-capacity.py` passes normal and race variants with no skipped required fixture | PASS |

## Security findings closed during implementation

1. The initial privacy regression showed that invalid retry and solver status strings were dropped as metric labels but still reached known structured log messages. `validOptimizationEvent` now requires the exact fields, types, and fixed values for every optimization message before delivery.
2. Sink errors previously reached the fallback writer verbatim and could include provider diagnostics, identifiers, or PII. Fallback records now contain only fixed `metric`/`log` failure text.
3. The first race run showed simultaneous submission metric/log failures could write concurrently to a non-thread-safe fallback writer. `optimizationFallbackMu` now serializes that bounded fallback boundary.
4. The first standalone script run revealed an anchored selector that omitted `TestTask234…` prefix tests. The selector now uses `TestTask234.*`, and the gate fails on skipped or empty test selection.
5. Review F-234-02 is closed without widening telemetry data: the production worker queue manager receives the existing typed adapter, and the runtime regression accepts only `optimization_queue_cleanup_total`, unit `cleanups`, fixed `outcome="failed"`, and the matching generic event; injected job/user identifiers and sink diagnostics are absent.

## Verification commands and final results

| Working directory | Command | Final result |
|---|---|---|
| repository root | `python3 scripts/verify-phase0701-observability-capacity.py` | PASS; 10 capacity-script unit tests, all Task 234 fixtures, queue/solver cleanup, child timeout, live Redis, isolated Redis restart, and the same set under `-race`; no required skip. |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -count=1` | PASS; complete backend normal suite. |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1` | PASS; complete backend race suite. |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS. |
| repository root | `python3 scripts/validate-task-list.py` | PASS; 237 sequential tasks with ordered dependencies; Task 234 remains `OPEN`. |
| repository root | `python3 scripts/validate-traceability.py` | PASS. |
| repository root | `gofmt -l backend/cmd/worker/main.go backend/internal/worker/optimization_processor.go backend/internal/worker/worker.go backend/internal/worker/task234_regression_test.go backend/internal/worker/worker_integration_test.go` | PASS; no output. |
| repository root | `git diff --check` | PASS. |

## Final hash evidence

| Path | SHA-256 |
|---|---|
| `docs/implementation/reviews/task-234-review.md` | `0def85668c5bdeb669810425d7d9dd049b33620c87c91495296e9036719a576a` |
| `docs/design/DESIGN-014.md` | `f9f6521d89e6d31306422017e07af5630ba4d8da56907174f3653ea0d72e9fe4` |
| `docs/design/DESIGN-004.md` | `45dd31e2afe1480ec54f540031ec45289f8287fbd0ef1a2d2cee86dea16a5474` |
| `docs/implementation/02_TASK_LIST.md` | `ab4c293b379394fe573aaa1cd67d89a996a0a07e363c1b31752a1d220b0b3adb` |
| `backend/internal/observability/optimization.go` | `793f008dd6760e65908d97beefcc556e2f6cb2d46e56a6ad2d6e9eca4ea6e08c` |
| `backend/internal/observability/task234_regression_test.go` | `adcaa1ff62ab0a04059e218df70b5e22f1488a6e93fc5d10b8e7dc88a79476dc` |
| `backend/internal/httpapi/task234_observability_capacity_test.go` | `b087144c0e87d6fa59a78be04e75603fc8a821ecd9c7a5aefb16e5c73e5a1e7b` |
| `backend/internal/queue/task234_regression_test.go` | `0012c7123869c8f06b30a5594b27a52f7c093c27da933172ea2a51d85818481c` |
| `backend/cmd/worker/main.go` | `c55fe688f187f186807407fe2b5f97bea94a819642d0c30d143795b0a43663e9` |
| `backend/internal/worker/optimization_processor.go` | `d53512126d70e0b8f4af3558548faf3d7ab3f0b2eb37c7152630e3de390490d8` |
| `backend/internal/worker/worker.go` | `c1a7a8352b3a4c26a3b92d52edd25c5b9aa5d38ec9a3ad5d094c64d949183b65` |
| `backend/internal/worker/task234_regression_test.go` | `8ca59f53b25532fcd90bf8bb3e445b04b60958af7b7a3d0f579473ccfff750c9` |
| `backend/internal/worker/worker_integration_test.go` | `01245327c46ddf7f13420ffe2a9200f630ca01e448450dd41512de5753a9f434` |
| `scripts/verify-phase0701-observability-capacity.py` | `2962cb8d607ee433431579bc285ee137446906faf017e91d59322a914decd09e` |
| `scripts/verify-optimization-capacity.py` | `b0055d5287d0909a77ab0c267c09ca76365612e2bd867b8c33053e40c4cd4e95` |
| `scripts/test_verify_optimization_capacity.py` | `b9e3548a11c9f6e11892be64d53ab581bcce58780cc1e95457df16a69b0590af` |
| `docs/implementation/phase07-task-208-capacity.md` | `25fbe6edde4360d7a6116b52ffe6678ccda5b774d5887cd7270b90b078da889e` |
| `docs/implementation/phase07.01-task-234-capacity.md` | `376aaba4cfb1f9426aaebe538b14663a2fd79714ed380faf80d4baf76ab3e96a` |

## Deviations and boundaries

- The credentialed deployment profile in `scripts/verify-optimization-capacity.py` was not executed because this worktree provides no real authenticated cookie, CSRF token, or private saved-diet request fixture, and Task 234 must not invent or expose those secrets. Its report/gate logic passes 10 unit tests, while the production Fiber/controller, worker-active polling, real Redis, cleanup, timeout, and recovery behavior passes the deterministic normal/race gate. The exact credentialed command remains documented for deployment capacity evidence.
- `python3 scripts/check.py` and frontend/browser checks are intentionally deferred to Task 235's aggregate quality gate. Task 234 ran the complete backend normal, race, and vet suites plus task-list/traceability/format/diff checks; it does not claim Phase 07.01 aggregate coverage or frontend acceptance.
- Redis connection-refused lines during the cleanup and restart fixtures are expected injected failure evidence; the fixtures pass only after bounded cleanup/recovery is observed.
- No coverage deviation is introduced or accepted by Task 234. Any phase-wide coverage disposition remains Task 235 scope.
- No task status, unrelated dirty-worktree edit, or prerequisite evidence was changed.

## Preparation decision

The rejected F-234-01 and F-234-02 paths now fail closed under focused normal/race and full backend normal/race verification: queue ACK follows a matching durable terminal state, and production worker queue telemetry is wired with bounded privacy-safe output. The deterministic gate and documented authenticated profile satisfy Task 234's focused observability, privacy, failure-load, and responsiveness criteria. Per explicit instruction, Task 234 remains `OPEN` and no task status was changed.
