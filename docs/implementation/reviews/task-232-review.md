# Review Evidence: Task 232 — Backend Integration and Functional Regression Gate

~~~yaml
task_id: 232
phase: "07.01"
component: "ARCH-004: JobStatusTracker"
static_aspect: "JobStatusTracker backend integration and functional regression gate"
input_status: "OPEN (preserved; task-status edits were prohibited)"
review_decision: "PASSED"
decision: "PASSED"
reviewed_at_utc: "2026-07-18T14:45:00Z"
review_agent: "Codex independent owner review"
evidence_file: "docs/implementation/reviews/task-232-review.md"
baseline_ref: "HEAD a4e31367485b03269e90b5607f2057c9568bb5b1 plus cumulative dirty Phase 07.01 worktree"
baseline_confidence: "HIGH"
inventory_symbol_count: 20
audited_symbol_count: 20
inventory_source_count: 20
code_review_skill_invoked: true
code_review_skill_invocation_count: 1
relevant_language_guides: "Go, security, async/concurrency, common-bugs, architecture, performance, and universal-quality guidance applied"
review_template_path: "docs/implementation/reviews/REVIEW_TEMPLATE.md"
review_template_available: false
fallback_review_template_path: "review.txt"
fallback_review_template_sha256: "f741e00f06e76a90c26ee263fc4485698f4fde182711352138179369f3186b20"
prior_evidence_checked_for_staleness: true
blocking_findings: 0
important_findings: 0
optional_findings: 1
repair_context_required: false
~~~

## 1. Task Source

**Description:** Phase 07.01: add integrated PostgreSQL, Redis, API, queue, worker, and packaged-CLP coverage that exercises the repaired Daily Diet and optimization contracts across nominal, replay, concurrency, recovery, and degraded paths.

**Task row:** `docs/implementation/02_TASK_LIST.md:239`; Task 232 remains `OPEN` and was not changed.

**Dependencies:** Tasks 216, 221, 223, 225, and 226 are `PASSED` in the current task table. Their preparation/review evidence was used only for the inherited boundaries explicitly named by Task 232; no later-task frontend or aggregate gate was attributed to this review.

**Architecture and design sources read in full:** `docs/architecture/ARCH-004.md`, `docs/design/DESIGN-004.md`, `docs/design/DESIGN-008.md`, `docs/design/DESIGN-014.md`, `docs/design/01_TECH_STACK.md`, and `docs/testing/integration/ARCH-004-obligations.md`.

**Preparation source:** `docs/implementation/preparation/task-232-preparation.md` was read in full. Its current hashes for the Task 232 test and inherited evidence files match the current worktree. Its PASS conclusion was treated as a claim to re-audit, not as review evidence by itself.

**Template note:** `docs/implementation/reviews/REVIEW_TEMPLATE.md` is absent both from this checkout and the `HEAD` tree. The complete root `review.txt`, `docs/implementation/reviewer-prompt.md`, and the repository's established 13-section review-evidence structure were read and used. No missing template was created.

**Decision at a glance:** The live Task 232 API/worker gate, inherited PostgreSQL/API/queue/worker/solver/publication/metrics suites, native CLP 1.17.11 boundary, packaged-worker image verification, normal and race backend gates, static/security/contract validators, and exact status/envelope assertions pass. One non-blocking test-isolation hardening note remains for the broad Redis cleanup helper; the documented serial command and a direct concurrent app/queue probe both pass.

## 2. Pre-Review Gates

- [x] The exact Task 232 row was read; it remains `OPEN`.
- [x] Full ARCH-004, DESIGN-004, DESIGN-008, DESIGN-014, tech-stack, and ARCH-004 integration-obligation sources were read.
- [x] Full `docs/implementation/preparation/task-232-preparation.md` was read, including its matrix, commands, deviations, and hashes.
- [x] Full fallback `review.txt` and `docs/implementation/reviewer-prompt.md` were read; the requested `REVIEW_TEMPLATE.md` path was checked and is absent.
- [x] `code-review-skill` was invoked exactly once; its Go, security, async/concurrency, common-bug, architecture, performance, and universal-quality guidance was applied.
- [x] The added Task 232 live test and every inherited integration/regression family named by the task preparation were read at source level.
- [x] Real PostgreSQL, Redis DB 14, authenticated Fiber routes, Redis Streams, the worker, and CLP 1.17.11 were exercised by the new composed test.
- [x] Nominal, replay, repair, concurrency, ownership/entitlement, outage, timeout, cancellation, duplicate-delivery, retry/finalization, recovery, solver, alternative, metric, and safe-envelope evidence was checked against actual assertions rather than test names alone.
- [x] The Task 232 test passed normally, under `-race`, and three repeated `-race` executions; the full backend normal suite passed serially, and the full backend race suite passed serially and with default package scheduling.
- [x] Current source hashes match the preparation manifest for all files it fingerprints, including the new Task 232 test.
- [x] No-cache packaged worker image verification passed and reported CLP 1.17.11 plus an executable worker.
- [x] No production code, task row/status, unrelated implementation, or prior review was edited; only this review document was added.

~~~yaml
pre_review_gates_passed: true
~~~

## 3. Review Baseline and Change Surface

The baseline is commit `a4e31367485b03269e90b5607f2057c9568bb5b1` plus the cumulative dirty Phase 07.01 worktree. The worktree contains many concurrent backend, frontend, API, migration, preparation, and review changes. Task 232 attribution was reconstructed from the exact task row, preparation manifest, current symbols, direct callers, test comments, commands, and SHA-256 fingerprints. The aggregate dirty diff was not treated as Task 232 ownership.

The new Task 232 surface is `backend/internal/app/task232_backend_regression_test.go`. It composes `NewProduction`, live PostgreSQL saved-diet persistence, authenticated Fiber requests, real Redis queue/job state, a dedicated worker, and native CLP. Its helper asserts exact top-level JSON key sets, acknowledgement cardinality, `Location`/poll URL identity, safe error vocabulary, error request-ID correspondence, and the expected HTTP statuses.

The inherited evidence surface is deliberately grouped by contract boundary:

1. Daily Diet CRUD, immutable create replay, PostgreSQL claim/replay, ownership, and rollback.
2. Optimization submission, canonical request hashing, durable acknowledgement replay/repair, admission/entitlement, safe HTTP projection, and telemetry.
3. CLP invocation/version/parser/serialization, constraints, lexicographic objective, diversity, validation, immutable snapshots, and distinct alternatives.
4. Redis Streams bootstrap, enqueue deduplication, reservation/reclaim, ownership-based attempts, terminal publication, duplicate cleanup, live group/restart recovery, and queue ages.
5. Worker publication ordering, partial failures, expiry, timeout/cancellation policy, terminal state monotonicity, and bounded telemetry.

The preparation's claim that Task 232 adds only the new test and this review artifact is consistent with the current task-specific hash manifest. Shared inherited files remain cumulative worktree files and are not re-attributed as new Task 232 production changes.

No merge, reset, checkout, staging, cleanup, production edit, or task-status edit was performed. The reviewer prompt asks for a branch merge, but merging into this shared dirty worktree would violate the user's narrower review-only scope, so the supplied baseline was preserved.

## 4. Acceptance Criteria Checklist

| # | Criterion | Result | Evidence and review conclusion |
|---:|---|---|---|
| 1 | Live PostgreSQL, Redis, authenticated API, queue, worker, and CLP compose in one focused gate. | PASS | `TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers` starts from live PostgreSQL/Redis, calls production Fiber routes, starts the production worker processor, and completes jobs through CLP 1.17.11. |
| 2 | Daily Diet create is immutable and replay-safe after replacement and deletion. | PASS | Task 232 creates, replaces, deletes, then replays the same key; the replay is exact `201`, data equals the original response, and PostgreSQL has no recreated diet. Existing live CRUD and repository claim tests cover cross-instance and malformed persistence boundaries. |
| 3 | Optimization request normalization and replay are stable and changed bodies conflict. | PASS | Reordered exclusions, negative zero, and reordered JSON members replay one `202`, one job, one stream entry, stable `Location`, and stable poll URL; changed tolerance returns exact `409 idempotency_key_conflict`. |
| 4 | Durable pending acknowledgement repair revalidates current entitlement/ownership and publishes once. | PASS | Task 232 seeds the real PostgreSQL pending claim, denies expired entitlement without stream publication, restores entitlement, reuses the server job ID, publishes one entry, and proves post-repair replay is side-effect free. Task 222 covers controller interleavings and cancellation. |
| 5 | Ownership and entitlement isolation are fail-closed. | PASS | Free submission is exact `403 entitlement_denied`; entitled non-owner submission and cross-user polling are exact `404 not_found`; existing HTTP tests cover anonymous denial, expiry isolation, and admission side-effect ordering. |
| 6 | Nominal worker results are server-derived, valid, ordered, excluded, bounded, and distinct. | PASS | Real worker/CLP assertions require one-to-three alternatives, exact target macros, nondecreasing calories, allowed meal IDs, no excluded IDs, and unique selected meal-ID sets. Domain tests cover quantity-weighted similarity and authoritative repository recalculation. |
| 7 | Solver model, parser, native child-process, version, and packaged image boundaries are covered. | PASS | CLP wrapper tests cover generated arguments, bounded output, parser statuses, cancellation/timeout, child termination, cleanup, version, LP serialization, and malformed references; packaged constraint/objective tests and the no-cache image check pass with CLP 1.17.11. |
| 8 | Queue enqueue/reservation/retry/finalization and duplicate delivery semantics are covered. | PASS | Real Redis tests cover stream/group bootstrap, logical enqueue deduplication, reservation, XAUTOCLAIM, attempts one/two/three, terminal publication-before-ack, zero-ack behavior, lock misses, malformed delivery removal, atomic duplicate cleanup, and cancellation-pending behavior. The concurrent-consumer test itself uses one idempotent stream entry; duplicate stream delivery credit comes from Task 206's explicit second `XADD` and Task 224/225 tests. |
| 9 | Redis group deletion, restart, authorization, and connectivity recovery are covered. | PASS | Task 225 live fixtures recover a deleted group and a Redis restart, verify post-recovery publication, and fail closed on invalid authorization/connectivity. |
| 10 | Waiting/pending queue ages and bounded metrics are covered. | PASS | Task 226 real Redis fixtures cover mixed, queued-only, pending-only, empty, and future-clock populations; observability tests enforce nonnegative values, exact names/units, and fixed labels. Task 223 covers bounded submission outcomes and privacy-safe telemetry. |
| 11 | Publication/failure, partial alternatives, expiry, timeout, cancellation, and safe failure mapping are covered. | PASS | Task 210 covers partial alternatives on later solver timeout and expiry owner markers; worker deadline tests cover one whole-job deadline and shutdown cancellation; HTTP tests cover all four canonical terminal failures, malformed persistence, and safe messages. |
| 12 | Redis outage does not invoke synchronous solving and returns a retryable safe response. | PASS | Task 232 uses an unreachable Redis endpoint and asserts exact `503 dependency/queue_unavailable`, retryability, fixed message, exact envelope keys, and request-ID correspondence. Queue tests independently assert no processor call when Redis is unavailable. |
| 13 | Concurrent unrelated users complete independent jobs and the complete backend race gate passes. | PASS | Task 232 submits two entitled users concurrently, asserts distinct job IDs and owner-scoped completion, then the focused race run and full `go test -race ./...` runs pass. |

## 5. Changed-Symbol Inventory

| # | Grouped symbol/unit | File:line | Task 232 surface audited | Result |
|---:|---|---|---|---|
| 1 | `TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers`, response/assertion/body/cleanup/concurrency helpers | `backend/internal/app/task232_backend_regression_test.go:35-397` | New composed live gate, exact statuses/envelopes, normalization, repair, ownership, concurrency, outage, and test cleanup | PASS; F-232-01 optional isolation note |
| 2 | `TestDailyDietProductionAPIWithLivePostgres` and live API/database/auth helpers | `backend/internal/app/daily_diet_api_integration_test.go:32-380` | PostgreSQL CRUD, CSRF, immutable replay, cross-user isolation, and concurrent API instances | PASS |
| 3 | `TestTask206BackendIntegrationGate`, `TestTask206TimeoutAndOwnershipGate`, and worker/live fixture helpers | `backend/internal/app/task206_backend_integration_test.go:40-536` | Nominal saved-diet-to-worker path, wrong client targets, exclusions, duplicate delivery, admission contention, infeasible path, outage, and timeout child boundary | PASS |
| 4 | Optimization controller submission/polling/ownership/failure/expiry tests | `backend/internal/httpapi/optimization_controller_test.go:22-445` | Exact HTTP status behavior, server-owned input, safe terminal projection, replay, queue repair, cross-user/expiry isolation, malformed result rejection | PASS |
| 5 | Task 222 submission integration suite | `backend/internal/httpapi/task222_optimization_submission_integration_test.go:23-348` | Cross-user progress, cancellation, published replay, pending repair, concurrent repair, admission retry headers, malformed acknowledgement, and side-effect gates | PASS |
| 6 | Task 223 submission observability integration suite | `backend/internal/httpapi/task223_submission_observability_test.go:22-414` | Final response/metric correspondence, failed repair, bounded cleanup, repeated blocked release, and blocking sink behavior | PASS |
| 7 | CLP wrapper/parser/process tests | `backend/internal/optimization/clp_wrapper.go`; `clp_wrapper_test.go:19-509` | Native child invocation, fixed args, generated names, output bounds/sanitization, deadlines, cancellation, parser, serialization, cleanup, and pinned version | PASS |
| 8 | Constraint builder and model eligibility tests | `backend/internal/optimization/constraints.go`; `constraints_test.go:24-314` | Authoritative diet identity, physical-state units, exclusion, numeric bounds, zero targets, page loading, distinctness, packaged CLP fixture | PASS |
| 9 | Diversity and lexicographic objective tests | `backend/internal/optimization/diversity.go`, `objective.go`; corresponding tests | Calorie-first ordering, secondary original-meal objective, hard exclusions, deterministic alternative generation, and packaged CLP objective | PASS |
| 10 | Solution validator and failure-classification tests | `backend/internal/optimization/validator.go`, `validator_test.go:23-346` | Repository recalculation, quantity-weighted similarity, finite bounds, malformed solutions, typed-nil/unknown failures, tolerance, and partial results | PASS |
| 11 | Task 220 public alternative pipeline tests | `backend/internal/optimization/task220_pipeline_test.go:16-282` | One snapshot/index, one projection per accepted result, attempt cap, duplicate rejection before state mutation, residue, immutable caller snapshot, concurrency | PASS |
| 12 | Redis Streams integration suite | `backend/internal/queue/job_queue.go`; `job_queue_integration_test.go:20-641` | Bootstrap, enqueue/reserve/ack, logical deduplication, reclaim, three attempts, stats, cancellation, malformed deliveries, lock misses, TTLs, configuration, and outage | PASS |
| 13 | Task 225 finalization/recovery suite | `backend/internal/queue/task225_queue_test.go:23-377` | Explicit terminal publication, completed/failed distinction, zero-ack semantics, embedded Lua/NOSCRIPT, atomic cleanup, group/restart recovery, fail-closed errors, bounded cleanup | PASS |
| 14 | Task 226 queue-age suite | `backend/internal/queue/task226_queue_age_test.go:15-132` | Waiting/pending population separation, authoritative idle age, empty/skew fixtures, and real Redis metadata | PASS |
| 15 | Task 210 worker SWE.5 suite | `backend/internal/worker/task210_swe5_integration_test.go:19-151` | Real Redis partial publication on later timeout and result TTL/owner marker | PASS |
| 16 | Worker bootstrap/terminal/heartbeat suite | `backend/internal/worker/worker_integration_test.go:25-289` | Publish-before-ack, atomic terminal transitions under interleaving, distinct alternatives, and heartbeat lifecycle | PASS |
| 17 | Worker deadline/cancellation/failure-policy tests | `backend/internal/worker/optimization_processor_deadline_test.go:15-259` | One whole-job deadline, visibility margin, shutdown cancellation pending, canonical failure vocabulary, typed nil, invalid failure normalization | PASS |
| 18 | Task 221 publication/decode suite | `backend/internal/worker/task221_publication_test.go:15-138` | Similarity-score publication/decode bounds and omitted/null/string/zero raw JSON presence/type cases | PASS |
| 19 | Task 223/226 observability suites | `backend/internal/observability/task223_optimization_test.go:11-58`; `task226_queue_age_test.go:11-75` | Bounded outcome vocabulary, label ownership, exact queue metric names/units/labels, negative and invalid metric inputs | PASS |
| 20 | Packaged worker image verification boundary | `scripts/verify-clp-worker-image.sh:1-29`; `backend/Dockerfile.worker:1-58` | No-cache linux/amd64 image, checksum-pinned CLP 1.17.11, executable worker, and runtime version check | PASS |

~~~yaml
inventory_source_count: 20
audited_symbol_count: 20
inventory_complete: true
generated_groupings:
  - "Rows group only tightly coupled symbols/tests with one contract and one evidence boundary; solver model, parser, queue, worker, and API boundaries remain separate because their failure modes differ."
  - "Inherited Task 206/210/221/222/223/225/226 files are supporting evidence for Task 232, not new Task 232 production ownership."
~~~

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | Isolation/state/concurrency | Security/performance/API | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|
| Task 232 composed test and helpers | Uses production construction and server-created identifiers; API data and queue payloads are not fabricated for the nominal path. | Exercises replay, repair denial/success, owner/not-found, outage, exact 202/201/204/403/404/409/503, and safe messages. | Worker is cancelled and joined; response-index association is deterministic; Redis reset is broad but scoped to the integration DB, recorded as F-232-01. | Exact root/error key sets and request-ID nesting prevent envelope drift; no raw diagnostics are asserted. | Real CLP/Redis/PG path passes; polling success envelope field-level exactness remains covered by the existing HTTP suite rather than this helper. | PASS |
| Live Daily Diet API suite | CRUD projections and immutable create claim are exercised through authenticated production routes. | CSRF, missing meal, replay, changed key body, cross-user get/replace/delete, replace, delete, and replay after deletion are covered. | PostgreSQL advisory lock serializes destructive migration reset; concurrent production instances use shared durable state. | Tests ensure rejected writes have no persisted diet and replay does not recreate data. | Full live path passes; exact nested JSON keys are less strict than Task 232 acknowledgement/error assertions. | PASS |
| Task 206 app gate | Worker reloads server-owned diet/meal data and does not trust client target macros. | Nominal, exclusion, cross-user poll, duplicate stream entry, same-user admission contention, infeasible, Redis outage, and timeout are covered. | Explicit duplicate `XADD` is processed after terminal state; cancellation and worker join are bounded. | Timeout fixture executes a child shell process through the production wrapper; outage asserts no processor fallback. | The helper requires CLP and skips only Redis/PG service setup; all current paths pass. | PASS |
| Optimization HTTP controller tests | HTTP projection is owner-scoped and terminal state is monotonic; public errors are fixed vocabulary. | Covers queued/processing/completed/failed, replay/conflict, queue repair, expiry, auth/entitlement, malformed raw result/failure/score, and 429 retry headers. | Fakes deliberately block repository/idempotency seams to test cancellation and cross-controller races. | Client-authored macros never reach side effects; diagnostics and foreign expiry ownership are hidden. | Poll envelope exactness is represented through decoded fields and data assertions; raw exact-key coverage is supplied by Task 232 for submission/error envelopes. | PASS |
| Task 222 submission suite | Durable claim/published acknowledgement is the cross-controller idempotency authority. | Published replay bypasses current dependency state; pending repair revalidates and handles cancellation, ownership, queue, and concurrent publication. | Barriers model cross-controller interleavings and admission ownership transfer. | No external identity is trusted from request body; retry headers use positive whole seconds. | Deterministic fakes cover interleavings not practical in one live app test. | PASS |
| Task 223 HTTP observability suite | Final response classification and telemetry outcome must agree without exposing identifiers. | Accepted/replayed/rejected/dependency/queue/unexpected paths plus blocked and failing sinks are covered. | Cleanup lanes are bounded and independent; repeated blocking is asserted. | Telemetry labels/messages are fixed and bounded. | Noncooperative sink fixtures exercise lifecycle bounds; package race passes. | PASS |
| CLP wrapper | Only validated LP models and fixed subprocess arguments cross the native boundary; output is bounded. | Optimal, infeasible, timeout, cancellation, malformed/missing/nonzero output, cleanup failure, version, and parser cases pass. | Context reaches child runner and private temp directories are cleaned. | Generated names keep caller IDs out of args/files; output sanitization limits diagnostics. | Native packaged executable test and image check pass; no cluster-specific solver dependency is claimed. | PASS |
| Constraint builder | Targets derive from authoritative saved-diet entries; physical-state base units and exclusions are explicit. | Invalid identity, zero targets, unsupported/unusable meals, invalid numbers, duplicate data, and bounded paging are covered. | Repository loads are bounded and source snapshots are validated before solve. | Client cannot supply authoritative macro totals; LP bounds are bounded. | Packaged fixture and unit tables pass; no additional production finding. | PASS |
| Diversity/objective | Calories are primary; original-meal base-unit quantity is secondary and cannot overturn calorie ordering. | Equal-calorie ties, hard exclusions, repeated selections, caps, and infeasible alternative paths are covered. | Generation state changes only after validated projection; canonical ordering is deterministic. | No scaling constant or mixed-unit penalty leaks into calorie objective. | Packaged lexicographic fixture passes; solver doubles are used only at the documented solver boundary. | PASS |
| Solution validator | Every accepted alternative is recalculated from repository data and has bounded meals/macros/quantities/similarity. | NaN/infinity, negative/oversized quantities, malformed IDs, wrong units, tolerance boundaries, typed nils, and later solve failure are covered. | One immutable snapshot/index is shared; concurrent validation test passes. | Solver/client-provided macro and similarity values are not authoritative. | Domain tests cover malformed outputs and partial-result semantics; live Task 232 confirms end-to-end projected values. | PASS |
| Task 220 pipeline | One snapshot/index and one projection per accepted result; duplicate results do not mutate exclusion state. | Nonpositive limits, capped attempts, residue, duplicate, malformed snapshot, caller mutation, and attempt exhaustion pass. | Caller mutation and 16-way concurrent validation are directly challenged. | Result count is capped at three. | The test explicitly proves the inherited queue “concurrent consumer” test is not the duplicate-delivery proof; separate explicit duplicate tests are credited. | PASS |
| Queue integration suite | Redis Streams group lag/pending state, ownership, and attempts are authoritative; malformed entries are removed. | Bootstrap, enqueue, reserve, reclaim, retry one/two/three, stats, cancellation, malformed UUID, lock miss, TTL, config, and outage paths pass. | Real Redis is used with unique streams/groups; Task 232's wildcard cleanup can cross these key prefixes if package tests overlap. | Stream payload is only logical job ID/timestamp; no user/diet body enters queue. | Direct concurrent-consumer case uses one deduplicated entry; explicit duplicate delivery is covered by Task 206/224/225. | PASS; F-232-01 |
| Task 225 queue suite | Terminal publication is required before acknowledgement; atomic Lua state is monotonic and topology-safe. | Failed/completed conflict, zero ack, NOSCRIPT fallback, duplicate cleanup, NOGROUP recovery, Redis restart, authorization, and cleanup timeout pass. | Atomic scripts and concurrent recovery/cleanup are exercised against live Redis. | Embedded scripts carry design trace; fixed hash tags support cluster execution. | Isolated restart fixture uses a local Redis process/container; no multi-node cluster is claimed. | PASS |
| Task 226 queue age suite | Waiting uses consumer-group lag/stream timestamp; pending uses Redis idle metadata. | Empty, queued-only, pending-only, mixed, and future-clock cases pass. | Unique streams/groups and real pending metadata avoid injected ages. | Age labels are not identifiers. | Pagination beyond 100 pending entries is not exercised, but source cursor is correct and this is outside the Task 232 acceptance failure set. | PASS |
| Task 210 worker suite | Valid partial alternatives persist before terminal failure and expiry retains owner classification. | Later timeout preserves the first result; completed TTL produces owner marker and not-found-compatible load error. | Real Redis queue delivery is acknowledged only after publication. | Failure code/message is safe and bounded. | Timeout solver is injected at the documented solver boundary; nominal native CLP is supplied by Task 232/packaged checks. | PASS |
| Worker bootstrap/terminal/heartbeat | Terminal store transitions are monotonic and worker publishes before ack. | Concurrent completed/failed transitions, distinct alternatives, heartbeat refresh/removal, and child worker loop pass. | Redis hooks create a deterministic terminal race; worker cancellation is joined. | No terminal diagnostic or private state is projected. | Real Redis integration passes normal and race suites. | PASS |
| Worker deadline/failure policy | One deadline spans load and all solves; parent shutdown cancellation remains pending; only valid canonical failures publish. | Timeout, visibility margin, cancellation, canonical four-code messages, typed nil, and invalid failure normalization pass. | Finalization uses detached bounded context only for live-parent timeout; admission release is checked. | Safe failure messages and telemetry statuses are fixed. | Injected loaders/solvers cover policy branches; Task 232 covers native nominal. | PASS |
| Task 221 publication/decode | Persisted alternatives must preserve required score presence/type and valid rounded range. | Negative/above-one/unrounded, omitted/null/string, and explicit zero cases pass. | Redis round-trip and raw payload injection are real. | Invalid persisted data is rejected before HTTP projection. | Explicit zero distinguishes a valid zero from absent/null. | PASS |
| Observability Task 223/226 | Metrics/events use bounded vocabulary, exact queue units, and fixed labels. | Unsupported outcomes, negative ages, wrong units, unknown kinds, and extra labels are dropped. | MemorySink assertions plus full race suite cover ownership and concurrent delivery. | No IDs, URLs, keys, bodies, diagnostics, or PII labels. | Cross-package telemetry coverage is inherited and sufficient for this gate; later aggregate coverage owns global percentages. | PASS |
| Packaged image boundary | Dedicated linux/amd64 worker image contains executable worker and pinned native CLP. | No-cache build, checksum validation, runtime version output, executable check, and CGO-disabled build pass. | Image test is independent from local API/Redis state. | CLP is an in-container child, not a network service; API image is not used as solver. | Current image digest is recorded in command evidence; no repository artifact was created. | PASS |

## 7. Findings

| ID | Severity | Status | File:line | Symbol | Problem / trigger | Evidence and disposition |
|---|---|---|---|---|---|---|
| F-232-01 | 🟢 [nit] | OPEN / NON-BLOCKING | `backend/internal/app/task232_backend_regression_test.go:315-332` | `task232ResetRedis` | The helper scans `mealswapp:optimization:*` and deletes every matching key. Queue managers using custom streams still derive attempt, lock, done, and enqueue keys under that same prefix, so overlapping package-level integration processes can delete unrelated queue state. This is a test-isolation/flakiness risk and can also remove local integration jobs outside this test's logical fixture. | The documented focused/aggregate commands run package tests serially where destructive PostgreSQL/Redis fixtures are shared; serial normal/race gates pass. The default-package `go test -race ./...` also passed in this run, and a direct concurrent app Task 232 plus queue integration probe passed. No failure was reproduced. Recommended hardening is exact per-test key cleanup or a dedicated Redis DB/namespace; this does not block the current Task 232 acceptance decision. |

~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 1
~~~

No blocking or important correctness, security, behavior-regression, solver, queue, worker, API-contract, race, or coverage finding remains. The queue duplicate-delivery claim was deliberately not over-counted: the named concurrent-consumer test uses one idempotent publication, while explicit duplicate stream delivery is verified elsewhere in the audited inherited suites.

## 8. Commands Run

All commands below exited 0 unless a warning is explicitly noted.

| Command | Working directory | Exit | Result |
|---|---|---:|---|
| `MEALSWAPP_REDIS_URL=redis://localhost:6379/14 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -v ./internal/app -run '^TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers$' -count=1` | `backend/` | 0 | PASS; live PostgreSQL/Redis/API/worker/CLP gate in 2.27s. |
| `MEALSWAPP_REDIS_URL=redis://localhost:6379/14 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/app -run '^TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers$' -count=1` | `backend/` | 0 | PASS in 3.78s. |
| `MEALSWAPP_REDIS_URL=redis://localhost:6379/14 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/app -run '^TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers$' -count=3` | `backend/` | 0 | PASS; three repeated live race executions in 8.87s. |
| `MEALSWAPP_REDIS_URL=redis://localhost:6379/14 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -p 1 -count=1` | `backend/` | 0 | PASS; serial complete backend suite. |
| `MEALSWAPP_REDIS_URL=redis://localhost:6379/14 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -p 1 -count=1` | `backend/` | 0 | PASS; serial complete backend race suite. |
| `MEALSWAPP_REDIS_URL=redis://localhost:6379/14 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1` | `backend/` | 0 | PASS; default package scheduling also completed all packages. |
| Concurrent `go test` probe: Task 232 app test and queue `TestJobQueue|TestTask224|TestTask225|TestTask226` suite | `backend/` | 0 | PASS; no isolation failure reproduced. |
| `gofmt -d` over all reviewed Go integration/regression files | `backend/` | 0 | PASS; no formatting diff. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | `backend/` | 0 | PASS. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `backend/` | 0 | PASS; zero called vulnerabilities; 18 uncalled required-module vulnerabilities reported by the tool. |
| `MEALSWAPP_REDIS_URL=redis://localhost:6379/14 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/app -run '^TestTask232' -coverprofile=/tmp/task-232-review-app.coverage.out -count=1 && go tool cover -func=/tmp/task-232-review-app.coverage.out | tail -1` | `backend/` | 0 | PASS; 57.1% of the complete app package statements for the single Task 232 selection. No Task 232 coverage threshold or new exception is claimed. |
| `bash scripts/verify-clp-worker-image.sh` | repository root | 0 | PASS; no-cache linux/amd64 image, executable worker, and `Coin LP version 1.17.11`; current image digest `sha256:a10d623223135083ee9e40542bed85691e8553db3959e8ccd9d2c696a9745893`. |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS; 237 sequential tasks; Task 232 remains `OPEN`. |
| `python3 scripts/validate-traceability.py` | repository root | 0 | PASS. |
| `python3 scripts/validate-phase07-go-doc.py` | repository root | 0 | PASS. |
| `python3 scripts/generate-api-types.py --check` | repository root | 0 | PASS; generated API types current. |
| `npx --no-install redocly lint api/openapi.yaml` | repository root | 0 | PASS with one pre-existing ignored OAuth callback 2XX warning; no Task 232 contract failure. |
| `git diff --check` | repository root | 0 | PASS for tracked worktree changes; new Go test was separately checked with `gofmt -d` and hashed. |

`python3 scripts/check.py` was not run because it includes the later frontend/browser/aggregate coverage gates owned by Tasks 233–235. Task 232's required backend normal/race/static/security, focused live integration, API drift/lint, traceability, and packaged-image evidence was run directly. No frontend result is used as Task 232 evidence.

## 9. Files Inspected and Staleness Fingerprints

SHA-256 fingerprints below were captured from the current worktree after verification. The preparation-listed hashes for Task 232 and inherited evidence match exactly; shared production files are recorded independently because they contain cumulative Phase 07.01 edits.

| File | Purpose | SHA-256 |
|---|---|---|
| `backend/internal/app/task232_backend_regression_test.go` | New composed Task 232 gate | `c7483d952cb84e139f6ca945c200fa54dbb0e38ee12eacd291733e89e5edb567` |
| `backend/internal/app/task206_backend_integration_test.go` | Inherited live app/worker evidence | `f9e0de887d0670b914267730d36a8f897bff42635db7d41c5f89fd9865ff6629` |
| `backend/internal/app/daily_diet_api_integration_test.go` | Live Daily Diet API evidence | `c58009446a62bdfff9fcbcccb003ad66ab25a3f242687169619098b456ce6eb0` |
| `backend/internal/httpapi/optimization_controller.go` | HTTP submission/polling contract | `b30d1f00bdc946908300f1f704b64f1f5ba4bd60f7590529f71a6b377a43965c` |
| `backend/internal/httpapi/optimization_controller_test.go` | HTTP contract regression suite | `3aae2bb23b667f5a18bd221a9d7959bd80def77b3e7e0582739766d651e6e214` |
| `backend/internal/httpapi/task222_optimization_submission_integration_test.go` | Durable replay/repair evidence | `37f390e9f7fd006a492cb9a43c593307134a1e510bab93fb02d3affe52255e55` |
| `backend/internal/httpapi/task223_submission_observability_test.go` | Submission telemetry evidence | `542629229075b1c8f3e6d80dee7036cd6c1cc3e15405dd525e1138c72541e563` |
| `backend/internal/optimization/clp_wrapper.go` | Native solver boundary | `cc5079bf7475f8bea0e7d97327a9f511a7ca17c4fbdd11564da2bf2bf3e48996` |
| `backend/internal/optimization/clp_wrapper_test.go` | Solver/parser/process tests | `ad201e23848593fe5f783dda419b7ffc5ea9d969f9f98e6152422e535e18664f` |
| `backend/internal/optimization/constraints.go` | Constraint builder | `9f1d72435bac344e8e5c0b4140c19d87392a993e8834c43f689cd24e86627db1` |
| `backend/internal/optimization/constraints_test.go` | Constraint tests | `3bb53ea9bdce760eec5a07aa6ca0d7a11c0c41006baf360940f6ab6a93a3514d` |
| `backend/internal/optimization/diversity.go` | Diversity generation | `647547e6488f23455ab56f5042d3aa2ffbae1caee0f56b43cdb00fb99ae7ffd7` |
| `backend/internal/optimization/diversity_test.go` | Diversity/packaged objective tests | `0a00afe09117ccf468477989b9beda2704db8e12d02eb09d860c5f2d797ae8fc` |
| `backend/internal/optimization/objective.go` | Lexicographic objective | `03461f5deb4b76e673216658c78269cd37042a7c0476baf8f459ee15a9764c35` |
| `backend/internal/optimization/objective_test.go` | Objective tests | `5957da3993b36ea7aa20ef99fc3ffb7cd2e7a224a60c3c495cc7aacaa625a979` |
| `backend/internal/optimization/validator.go` | Alternative/result validation | `5ceb96bf19396ff9bc33e1de54fc879a62c5c948815b7fc1ac2b1468f68c6efd` |
| `backend/internal/optimization/validator_test.go` | Validator/failure tests | `d7ac4c7b1dfde12def8f49a435e9d2530d568255c198eeb984346f68835e6075` |
| `backend/internal/optimization/task220_pipeline_test.go` | Pipeline regression tests | `0704646e1bd48048dc95ca2320dd2018d6c5242cf8c0092b166b646f30eccea5` |
| `backend/internal/queue/job_queue.go` | Queue implementation boundary | `df27156ee125c8e4e62f090eb6be9506afc8bd35d76c7b61d3c3c77800d22e07` |
| `backend/internal/queue/job_queue_integration_test.go` | Redis Streams integration | `4eeb0a386fcc6fdc52b2a60e38920f3f0e7cb9233a94e381d9671a502263b420` |
| `backend/internal/queue/task225_queue_test.go` | Finalization/recovery tests | `b9c35c96fb1972de5c48a96daa28ae0a26b9a4f8b909ab26bcfae5435d7ad9c5` |
| `backend/internal/queue/task226_queue_age_test.go` | Queue-age tests | `4f7c8ce3ce102c0f5f7e52814c6c1d5b7cd895489538c61223668afa9753b58d` |
| `backend/internal/worker/optimization_processor.go` | Worker processor boundary | `f3e42d4eadbb1ade39410da510c6144a3204a749c45353ab5a44fb39ad6971e0` |
| `backend/internal/worker/task210_swe5_integration_test.go` | Partial/expiry worker tests | `3f76d08b34d74a3ac965cb75e285d60003e3d638ef843b420ef84ab47f2599f7` |
| `backend/internal/worker/task221_publication_test.go` | Persisted result tests | `1a91f30779bf8cc3139c10d71a9a6520785d9698d6b1d8ced9a067838c4e3d4c` |
| `backend/internal/worker/worker_integration_test.go` | Worker queue integration | `935c3caa093d220942a7e468355881ad38f94d29d36f18bc983bde08bda1615c` |
| `backend/internal/worker/optimization_processor_deadline_test.go` | Deadline/cancellation tests | `2c7601ef63fc4c6e46d256c958251f8cfc655cdec6488a707e87339a767c1d24` |
| `backend/internal/observability/optimization.go` | Metrics boundary | `88c0e988d043abf4b44266302b74e68780c3af589eae0022e21608974332d042` |
| `backend/internal/observability/task223_optimization_test.go` | Bounded telemetry tests | `251deca606e3836d73508576b3f076567c3db43e0b09ebb14c29c47a44e09ac1` |
| `backend/internal/observability/task226_queue_age_test.go` | Queue metric tests | `5dc75753fc82fabed79d442b1457c2e21da63f28e82216875c21fe8d8a4ce67b` |
| `api/openapi.yaml` | Documented status/envelope contract | `392a3d531301a937b001bc7561b6e5cdef76a6a786d2073d739ab81cd1161c4a` |
| `scripts/generate-api-types.py` | Generated-contract drift check | `c2fdf54b8280eedf91b149ae9f94fd8d1f9a01d22095b57bb53309f792313acc` |
| `scripts/verify-clp-worker-image.sh` | Packaged image verification | `1d56d9472d77390ce564664ee0ea1cd7fdcebd42a62e60a71ad0e79526c8fd36` |
| `backend/Dockerfile.worker` | Packaged worker image | `dbf7af9f61f8d7ac0aaf9a5c42a9f34c841ee1897e61f1eeca8172d4a39dd273` |
| `docs/architecture/ARCH-004.md` | Architecture source | `bedcf45c79f50cfe24313345ac2ba664130ee44d7d2f35c7b1de0a06e0abe867` |
| `docs/design/DESIGN-004.md` | Optimizer design source | `45dd31e2afe1480ec54f540031ec45289f8287fbd0ef1a2d2cee86dea16a5474` |
| `docs/design/DESIGN-008.md` | Saved-data design source | `551880a70a3b42698e11632a471957bbecc6011900cf121c34f1ff3c3db18b87` |
| `docs/design/DESIGN-014.md` | Observability design source | `f9f6521d89e6d31306422017e07af5630ba4d8da56907174f3653ea0d72e9fe4` |
| `docs/testing/integration/ARCH-004-obligations.md` | SWE.5 obligation source | `ffc3c036ad32a58fc340a9dddcd5cedbefa37ee687c2b200e99ac9b53cca91b0` |
| `docs/implementation/preparation/task-232-preparation.md` | Preparation claim/evidence | `e635320f1020d8272d4647d0b1e4438c05c291b79b0a70c2450a684f42748099` |
| `docs/implementation/02_TASK_LIST.md` | Task/status source | `3bdabe886facb2b96875489dcce7186d13acecc0a1582ff8e2fc8cc1dff62ebf` |
| `review.txt` | Full fallback template | `f741e00f06e76a90c26ee263fc4485698f4fde182711352138179369f3186b20` |

~~~yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The preparation conclusion is implementation evidence only; all named current files and commands were independently re-read or rerun."
  - "Shared Phase 07.01 files contain cumulative edits; their current hashes, not preparation hashes alone, are used for this review."
  - "The current packaged image digest differs from the preparation's prior local image digest because the no-cache image was rebuilt; the source/image verification still passed."
~~~

## 10. Coverage and Exceptions

- [x] The focused Task 232 coverage command ran and its artifact path is recorded.
- [x] The complete backend normal and race suites passed; a separate default-scheduling race run also passed.
- [x] Solver, queue, worker, HTTP, PostgreSQL, Redis, telemetry, outage, timeout, cancellation, and malformed-boundary branches were source-audited and mapped to tests.
- [x] The Task 232 row declares no numeric package-coverage threshold; the measured 57.1% is not presented as a phase-wide gate.
- [x] No new coverage exception was added to `docs/implementation/04_OPEN.md`.

The single-selection app coverage profile is `/tmp/task-232-review-app.coverage.out` and reports 57.1% of the complete `internal/app` package statements. This is expected for one composed integration test and is not evidence that the package or Phase 07.01 has reached the later aggregate 100% line-coverage gate. Task 235 owns that aggregate coverage decision and its documented exceptions. Task 232's criteria are integration behavior and backend regression gates, not package-wide line coverage.

The optional F-232-01 is test isolation debt, not a coverage exception. The inherited queue age suite also has no >100-entry pagination fixture, but the implementation cursor is source-audited and the Task 232 row does not require that specific fixture.

~~~yaml
coverage_required: true
coverage_exception_allowed: true
coverage_report_path: "/tmp/task-232-review-app.coverage.out"
observed_line_coverage: "57.1% of complete internal/app statements for the single Task 232 selection"
coverage_passed: true
~~~

## 11. Negative and Regression Checks

- [x] Added and inherited focused integration tests pass against local PostgreSQL/Redis/CLP dependencies.
- [x] The full backend normal suite passes serially; the full backend race suite passes serially and with default package scheduling, plus a direct concurrent app/queue probe.
- [x] `go vet` and `govulncheck` pass; no called vulnerability was reported.
- [x] Generated API types, OpenAPI lint, task-list, traceability, Go-doc, and formatting checks pass.
- [x] The one OpenAPI warning is the existing intentionally `302`-only OAuth callback and is explicitly ignored; no Task 232 path is implicated.
- [x] Duplicate delivery was not falsely credited to the one-entry concurrent-consumer test; explicit duplicate publication/delivery tests were independently located and passed.
- [x] Exact acknowledgement/error root and nested key sets are asserted; 204 delete is asserted to have an empty body.
- [x] No solver diagnostics, Redis addresses, identifiers, request bodies, credentials, or PII are asserted as public telemetry/error content.
- [x] Task status and all unrelated dirty worktree paths were preserved.

The only open note is F-232-01: exact Redis key cleanup or a dedicated test namespace would make package-level integration execution independent. It did not fail the current serial or default race gates and does not invalidate the Task 232 behavior evidence.

## 12. Decision

A Task 232 review may be `PASSED` when every row criterion and audited symbol passes, evidence is current and hashed, normal/race backend gates are green, and no blocking or important finding remains. Those conditions are met.

~~~yaml
decision: "PASSED"
reason: "The live composed Task 232 gate and inherited backend integration/regression boundaries pass against PostgreSQL, Redis, API, queue, worker, native/packaged CLP, and observability contracts. Exact statuses/envelopes, replay/repair, ownership/entitlement, concurrency, publication/failure, solver/distinct alternatives, recovery/outage/timeout/cancellation/duplicate delivery, race, static, security, and traceability evidence are current."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "Keep Task 232 OPEN for the separate orchestrator status transition; optionally replace task232ResetRedis wildcard deletion with exact fixture-owned cleanup or a dedicated Redis namespace before increasing package-level integration parallelism."
~~~

## 13. Repair Context

~~~yaml
repair_context_required: false
~~~

This is a first independent Task 232 review of the current implementation evidence; no prior Task 232 review or repair finding was supplied. The preparation document's implementation conclusion is confirmed for the required gate, with F-232-01 retained as a non-blocking test-isolation hardening note.

Do not change: Task 232 status, unrelated Phase 07.01 code, shared backend implementation, task-list rows, or dependency evidence. The only requested write was this review artifact.
