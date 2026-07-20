# Review Evidence: Task 224 — Queue Reservation, Ownership, and Retry Contract

task_id: 224
component: "JobQueueManager"
static_aspect: "DESIGN-004: JobQueueManager"
input_status: "OPEN; task status intentionally preserved by explicit request"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-17T14:28:40Z"
review_agent: "Codex fresh independent owner review"
evidence_file: "docs/implementation/reviews/task-224-review.md"
baseline_ref: "HEAD a4e31367485b03269e90b5607f2057c9568bb5b1 plus task-224-preparation.md and current symbol scope"
baseline_confidence: "MEDIUM"
code_review_skill_invoked: true
relevant_language_guide: "code-review-skill Go guide"
repair_context_required: false

## 1. Task Source

Description: Phase 07.01: make reservation cardinality explicit, define reclaim partial-failure behavior, count processing attempts only after logical ownership with atomic TTL updates, validate and remove malformed UUID deliveries, and enforce coherent visibility, finalization-margin, and TTL precision constraints.

Depends On: 217 (PASSED)

Testing Coverage Exceptions: None in the Task 224 row. An existing Phase 07 queue coverage deviation is recorded in docs/implementation/04_OPEN.md; no new exception is accepted here.

Verification Criteria: Configuration either enforces single-message reservation or every reserved/reclaimed delivery is returned and processed; partial reclaim results are explicit; duplicate deliveries and lock misses do not consume processing attempts; attempt increment/expiry is atomic; malformed/non-canonical/nil UUID entries are rejected at enqueue and fully acknowledged/deleted at decode; accepted timing configurations cannot overlap the 30-second solve plus finalization budget or truncate Redis TTLs to zero; real-Redis fixtures cover duplicate delivery, contention, three genuine attempts, partial failure, malformed entries, and boundary durations; go test -race ./... passes.

## 2. Pre-Review Gates

- [x] The task row is OPEN only because this evidence-only review was explicitly required without changing task status; the refreshed preparation report claims completion.
- [x] Dependency 217 is PASSED.
- [x] The preparation report claims completion and identifies Task 224 surfaces and residual Task 225/226 boundaries.
- [x] A task-specific scope is available from the preparation manifest, HEAD diff, and symbol attribution.
- [x] code-review-skill was invoked exactly once and its relevant Go and concurrency guidance was read.
- [x] The reviewer is independent from implementation and repair.
- [x] Review uses current source, current Redis, and fresh command results; stale pre-repair conclusions were not reused.
- [x] No production code or task-list status was changed during review.

pre_review_gates_passed: true
blocking_issue: "None; task status remains OPEN by explicit instruction and is not part of this evidence write."

## 3. Review Baseline and Change Surface

Baseline/reference method: HEAD a4e31367485b03269e90b5607f2057c9568bb5b1 was compared with the three preparation-attributed implementation/test paths. The preparation manifest was read in full, then current queue source, callers, tests, DESIGN-004, ARCH-004, and queue review actions were independently inspected. The cumulative dirty worktree was handled by symbol-level attribution.

Commands used to reconstruct the diff:

    git status --short --branch
    git log --oneline --decorate -6
    git diff --unified=3 -- backend/internal/queue/job_queue.go backend/internal/queue/job_queue_integration_test.go backend/internal/worker/task210_swe5_integration_test.go
    rg -n -C 4 '224' docs/implementation/02_TASK_LIST.md
    nl -ba docs/implementation/preparation/task-224-preparation.md
    rg -n 'JobQueueManager|BatchSize|countAttempt|canonicalJobID|VisibilityTimeout' backend/internal/queue docs/design/DESIGN-004.md docs/architecture/ARCH-004.md
    sha256sum <reviewed-files>

Pre-existing dirty-worktree changes and exclusions:

- job_queue.go and job_queue_integration_test.go contain the Task 224 queue changes identified by the preparation report and current diff.
- task210_swe5_integration_test.go contains concurrent prior-task changes. Task 224 attribution is limited to the visibility literal change in TestTask210WorkerPublishesPartialAlternativesOnLaterSolverFailure; unrelated target-macro and solver-call changes were preserved and excluded.
- DESIGN-004, ARCH-004, and the task list were inspected as source/status context and were not edited.
- The preparation report's aggregate race failure is stale: fresh current-tree go test -race ./... -count=1 passed.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| backend/internal/queue/job_queue.go | Task 224 preparation and current diff | MEDIUM | timing constants, Enqueue, Reclaim, Process, Ack, delivery preparation/decoding, UUID validation, attempt counting, Lua TTL changes, validation, lockTTL |
| backend/internal/queue/job_queue_integration_test.go | Task 224 preparation and current diff | HIGH | reservation/reclaim, retry, malformed delivery, partial prefix, ownership, atomic TTL, and boundary fixtures |
| backend/internal/worker/task210_swe5_integration_test.go | Task 224 preparation; mixed prior-task file | MEDIUM | one Task 224 visibility fixture; unrelated symbols excluded |

The task-owned scope is distinguishable, so review continued with MEDIUM baseline confidence rather than treating the broad dirty diff as Task 224.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Reservation cardinality is explicit and no delivery is silently dropped. | validate inspection, BatchSize boundary test, Redis reservation/reclaim tests. | PASS | validate rejects every BatchSize other than 1; Reserve and claim use that value. Reclaim returns the prepared slice and error. |
| 2 | Partial reclaim failure is explicit. | Multi-entry real-Redis fixture with valid prefix and malformed tail. | PASS | prepareDeliveries returns the valid prefix with later ErrInvalidJob; malformed tail is removed. |
| 3 | Duplicate deliveries do not consume attempts. | Real-Redis duplicate fixture and attempt-key inspection. | PASS | Attempt counting occurs only in owned Process execution; duplicate paths have no attempt key. |
| 4 | Lock contention does not consume attempts or invoke the processor. | Real-Redis external-lock fixture. | PASS | SetNX miss acknowledges before countAttempt; the fixture observes no processor call and no attempt key. |
| 5 | Completed deliveries do not consume attempts. | Completion-marker fixture. | PASS | The second completion check precedes countAttempt; the fixture observes no attempt key. |
| 6 | Three genuine invocations receive attempts 1, 2, and 3 with aligned telemetry. | Real-Redis retry fixture. | PASS | The test records [1, 2, 3], one terminal call, and [retry, retry, exhausted]. |
| 7 | Attempt increment and expiry are atomic. | Lua inspection and invalid-TTL Redis execution. | PASS | countAttemptScript uses one SET PX; invalid TTL fails before SET and leaves no key. |
| 8 | Enqueue accepts only canonical, non-nil UUIDs. | Enqueue table with malformed and noncanonical values. | PASS | canonicalJobID requires uuid.Parse, non-nil identity, and exact UUID.String equality before Redis I/O. |
| 9 | Malformed deliveries are rejected and fully acknowledged/deleted. | Decode inspection and XRANGE plus XPENDING assertions. | PASS | removeDeliveryScript executes XACK and XDEL together; the live nil-UUID fixture leaves neither stream data nor pending state. |
| 10 | Accepted visibility configurations cannot overlap the 30-second solve plus 5-second finalization budget at Redis precision. | Boundary arithmetic, go-redis command encoding, deterministic boundaries, and live lock PTTL. | PASS | `validate` truncates the lock TTL to milliseconds before comparison; 36s and sub-ms surplus are rejected, 36s+1ms is accepted, and the live lock PTTL is greater than 35 seconds. |
| 11 | Accepted TTLs cannot truncate to zero. | Millisecond boundary tests and Lua argument inspection. | PASS | CompletedTTL and AttemptTTL require at least 1ms; scripts pass PX milliseconds; the live 500ms fixture observes positive TTLs. |
| 12 | Real-Redis fixtures cover duplicate delivery, contention, three attempts, partial failure, malformed entries, and boundaries. | Focused queue tests against live Redis. | PASS | Focused Task 224 tests passed 10 times and the full queue package passed 10 times against the local Redis service. |
| 13 | The full race gate passes. | Final current-tree race command. | PASS | The final current-tree `go test -race ./... -count=1` passed for every package after an unrelated Task 223 snapshot update; queue and worker race suites also pass independently. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | Task 224 timing constants | configuration | backend/internal/queue/job_queue.go:40-42 | modified | validate, lockTTL, worker timeout contract | TestTask224ConfigurationBoundaries |
| 2 | JobQueueManager.Enqueue | method | backend/internal/queue/job_queue.go:237-257 | modified | submission and enqueue callers | enqueue idempotence, UUID, TTL fixtures |
| 3 | JobQueueManager.Reclaim | method | backend/internal/queue/job_queue.go:310-328 | modified | worker reclaim path | reclaim, retry, prefix fixtures |
| 4 | JobQueueManager.Process | method | backend/internal/queue/job_queue.go:349-426 | modified | ProcessNext and worker processor | ownership, retry, cancellation, duplicate tests |
| 5 | JobQueueManager.Ack | method | backend/internal/queue/job_queue.go:432-443 | modified | cleanup and terminal callers | queue and prefix fixtures |
| 6 | JobQueueManager.prepareDeliveries | method | backend/internal/queue/job_queue.go:564-574 | added | Reclaim | partial-prefix fixture |
| 7 | JobQueueManager.prepareDelivery | method | backend/internal/queue/job_queue.go:579-589 | modified | Reserve and prepareDeliveries | malformed and normal Redis fixtures |
| 8 | decodeJob | function | backend/internal/queue/job_queue.go:593-610 | modified | prepareDelivery | malformed and prefix fixtures |
| 9 | canonicalJobID | function | backend/internal/queue/job_queue.go:615-618 | added | enqueue, decode, process, acknowledgement | malformed enqueue/decode fixtures |
| 10 | JobQueueManager.countAttempt | method | backend/internal/queue/job_queue.go:622-627 | added | owned Process path | three-attempt and TTL fixtures |
| 11 | JobQueueManager.finalize | method | backend/internal/queue/job_queue.go:645-659 | modified | Process, Ack, ack | completion and TTL fixtures |
| 12 | JobQueueManager.validate | method | backend/internal/queue/job_queue.go:727-741 | modified | all public queue operations | configuration fixture |
| 13 | lockTTL | function | backend/internal/queue/job_queue.go:766-772 | modified | Process lock and validate | configuration fixture |
| 14 | finalizeScript | Lua unit | backend/internal/queue/job_queue.go:802-811 | modified | finalize | completion and TTL fixture |
| 15 | enqueueScript | Lua unit | backend/internal/queue/job_queue.go:813-826 | modified | Enqueue | idempotence and TTL fixture |
| 16 | countAttemptScript | Lua unit | backend/internal/queue/job_queue.go:828-843 | added | countAttempt | attempts and invalid-TTL fixture |
| 17 | removeDeliveryScript | Lua unit | backend/internal/queue/job_queue.go:845-851 | added | prepareDelivery | malformed and prefix fixtures |
| 18 | newIntegrationQueue | test helper | backend/internal/queue/job_queue_integration_test.go:41-56 | modified | live queue fixtures | all queue integration tests |
| 19 | TestJobQueueEnqueueReserveAndAckUseRedisStreams | integration test | backend/internal/queue/job_queue_integration_test.go:91-117 | modified | reservation and ack contract | live Redis |
| 20 | TestJobQueueReclaimsAbandonedDeliveryWithXAUTOCLAIM | integration test | backend/internal/queue/job_queue_integration_test.go:200-229 | modified | reclaim contract | live Redis |
| 21 | TestJobQueueRetriesAndTerminallyFailsAfterThreeAttempts | integration test | backend/internal/queue/job_queue_integration_test.go:231-296 | modified | retry contract | live Redis |
| 22 | TestJobQueueUnavailableDoesNotInvokeProcessor | integration test | backend/internal/queue/job_queue_integration_test.go:591-610 | modified | unavailable queue behavior | unreachable Redis |
| 23 | TestTask224RejectsNonCanonicalJobIDsAndRemovesMalformedDeliveries | integration test | backend/internal/queue/job_queue_integration_test.go:369-406 | added | UUID boundary | live Redis |
| 24 | TestTask224ReclaimPreparationReturnsValidPrefixOnLaterFailure | integration test | backend/internal/queue/job_queue_integration_test.go:408-444 | added | partial reclaim | live Redis |
| 25 | TestTask224LockMissAndCompletedDeliveryDoNotConsumeAttempts | integration test | backend/internal/queue/job_queue_integration_test.go:446-511 | added | ownership-before-attempt | live Redis |
| 26 | TestTask224AtomicAttemptAndMillisecondTTLs | integration test | backend/internal/queue/job_queue_integration_test.go:513-554 | added | atomic counter and TTL precision | live Redis |
| 27 | TestTask224ConfigurationBoundaries | integration test | backend/internal/queue/job_queue_integration_test.go:556-589 | added | cardinality and timing | live Redis fixture setup |
| 28 | TestTask224RedisEffectiveLockTTLExceedsProcessingBoundary | integration test | backend/internal/queue/job_queue_integration_test.go:593-620 | added | live Redis lock ownership | live PTTL fixture |
| 29 | TestTask210WorkerPublishesPartialAlternativesOnLaterSolverFailure | mixed integration test | backend/internal/worker/task210_swe5_integration_test.go:18-91 | modified; Task 224 owns visibility literal only | worker queue fixture | live Redis |

The modified task210PartialSolver.Solve and other changes in the shared worker file are prior-task work and excluded. No generated grouping is used.

inventory_source_count: 29
audited_symbol_count: 29
inventory_complete: true
generated_groupings:
  - "None; every Task 224-attributed executable symbol and live regression test is listed separately."

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| Task 224 timing constants | Encode 30s work, 5s finalization, 1s visibility-to-lock margin, and Redis millisecond policy. | Boundary values are checked by `validate`; zero/sub-ms TTLs are rejected. | Cross-process TTL safety uses the Redis-effective lock value. | No user data. | No runtime cost. | Clear named constants. | Exact 36s, +1ns, +1ms, and TTL boundaries pass. | PASS |
| Enqueue | Canonical UUID and one logical stream publication. | Invalid config/context/UUID fail before Redis; Redis errors map safely. | Bootstrap and enqueue script are cross-process idempotent. | Only server-created ID enters payload; state keys hash it. | One Lua call. | Minimal API. | Broad enqueue malformed table; decode variants not all asserted. | PASS |
| Reclaim | Return prepared work plus later error. | Empty, claim, decode, and cleanup errors are observable; valid prefix retained. | Claimed valid jobs remain caller-visible. | Stream fields untrusted. | Linear bounded slice. | Explicit prefix contract. | Live prefix test passes. | PASS |
| Process | Completion check and logical ownership precede attempt count. | Done/lock miss avoid processor/count; errors return; processor paths follow existing policy. | SetNX lock is cross-process; Task 225 owns cleanup failure; effective TTL is validated before Redis. | Canonical direct job ID boundary. | Bounded Redis calls and processor. | Ordering and precision guard are correct. | Ownership/retry tests and live PTTL fixture pass. | PASS |
| Ack | Validate identity and finalize a reserved delivery. | Config/bootstrap/finalize errors propagate. | One finalization script; Task 225 owns broader terminal semantics. | Canonical ID only. | One Lua call. | Existing API. | Queue/prefix fixtures pass. | PASS |
| prepareDeliveries | Preserve valid prefix on later error. | Empty and every error path return explicit slice plus error. | No attempt mutation. | Delegates strict decode. | Linear allocation. | Idiomatic. | Live prefix fixture passes. | PASS |
| prepareDelivery | Decode one delivery and remove malformed data. | Decode errors run XACK/XDEL; cleanup errors are observable. | Attempt stays zero. | Strict untrusted stream boundary. | One cleanup script only on error. | Clear. | Nil case passes; other forms need fixtures. | PASS |
| decodeJob | Map stream data only after canonical UUID validation. | Wrong type/missing/invalid ID fail; bad timestamp safely falls back. | Pure function. | Redis payload untrusted. | Small conversion. | Single predicate. | Malformed and prefix fixtures. | PASS |
| canonicalJobID | Exact lowercase hyphenated non-nil UUID.String form. | Parse, nil, case, braces, whitespace, and controls fail. | Pure deterministic helper. | Job-ID trust boundary. | Constant-size parse. | Minimal. | Enqueue table broad; decode table incomplete. | PASS |
| countAttempt | Atomic logical attempt increment with PX after ownership. | Script failures map to queue unavailable. | One EVAL prevents normal value-without-expiry. | Hashed internal key. | One round trip. | Good separation. | Three-attempt and invalid-TTL fixtures. | PASS |
| finalize | Terminal marker, XACK, XDEL with PX. | Context and Redis errors observable. | One script; Task 225 boundaries preserved. | Server-derived keys. | One Lua call. | PX is correct. | Completion/TTL fixture. | PASS |
| validate | Reject bad identity, cardinality, timing, retry settings, and Redis-effective lock boundaries before Redis. | All listed invalid settings fail; +1ns and sub-ms lock surplus fail while +1ms passes. | Cross-process safety is checked at the precision Redis receives. | No command before checks. | Constant-time. | Policy and error boundary are clear. | Exact boundary table passes. | PASS |
| lockTTL | Visibility minus one-second margin. | Nonpositive fallback; positive subtraction is later truncated consistently with go-redis. | Redis receives the effective millisecond value through SetNX. | No user data. | Constant-time. | Single source for lock calculation. | Boundary and live PTTL fixtures pass. | PASS |
| finalizeScript | Atomic terminal marker and delivery cleanup. | Existing marker preserved; script errors abort. | Single Redis execution. | Configured stream/group and hashed key. | Fixed script. | Inline extraction is Task 225. | Live completion fixture. | PASS |
| enqueueScript | Existing marker replay or atomic XADD plus PX marker. | XADD failure is surfaced without marker. | One script prevents duplicate publication. | Canonical ID only. | Fixed payload. | Clear. | Idempotence/TTL fixture. | PASS |
| countAttemptScript | Read, increment, and PX-set counter atomically. | Nonnumeric fails; fractional numeric state is not rejected. | Malformed state can mutate before Go Int64 error. | Internal state boundary. | Fixed script. | Add integer validation later. | Invalid-TTL only; no corrupt-counter fixture. | PASS with optional gap |
| removeDeliveryScript | Atomic XACK and XDEL. | XACK failure aborts before XDEL. | One script. | No diagnostics or IDs. | Fixed script. | Correct Task 224 scope. | XRANGE/XPENDING pass. | PASS |
| newIntegrationQueue | Isolated live Redis fixture with safe defaults. | Unique stream/consumer and cleanup. | Live Redis state. | Generated IDs. | Small setup. | Reusable. | Repeated suites. | PASS |
| TestJobQueueEnqueueReserveAndAckUseRedisStreams | Attempt-zero reservation and clear pending after ack. | Identity, timestamp, attempt, pending assertions. | Real stream state. | Valid fixture UUID. | Small. | Direct. | Repeated pass. | PASS |
| TestJobQueueReclaimsAbandonedDeliveryWithXAUTOCLAIM | Reclaim without pre-ownership attempt. | Abandon/reclaim/process/cleanup. | Real pending ownership. | Valid ID. | Bounded sleep. | Focused. | Race and repeated pass. | PASS |
| TestJobQueueRetriesAndTerminallyFailsAfterThreeAttempts | Exactly three genuine attempts and telemetry. | Processor failure, reclaim, terminal, pending checks. | Real pending state. | Fixture error only. | Three attempts. | Clear loop. | 10 repetitions and race pass. | PASS |
| TestJobQueueUnavailableDoesNotInvokeProcessor | Redis failure cannot run processor. | Enqueue and ProcessNext return unavailable. | Unreachable endpoint. | No user data. | Short timeouts. | Focused. | Package pass. | PASS |
| TestTask224RejectsNonCanonicalJobIDsAndRemovesMalformedDeliveries | Enqueue rejection and stream cleanup. | All forms at enqueue; nil form at decode. | XRANGE and XPENDING empty. | Adversarial IDs. | Fixed table. | Strong but incomplete decode variants. | Optional hardening remains. | PASS with optional gap |
| TestTask224ReclaimPreparationReturnsValidPrefixOnLaterFailure | Valid prefix survives malformed tail. | Direct private seam plus Redis stream assertions. | Valid item remains until Ack. | Malformed payload. | Two entries. | Direct. | Live pass. | PASS |
| TestTask224LockMissAndCompletedDeliveryDoNotConsumeAttempts | Ownership and done checks precede count. | External lock, duplicate, done-marker branches. | Attempt keys inspected. | Valid fixture IDs. | Small. | Clear. | Live pass. | PASS |
| TestTask224AtomicAttemptAndMillisecondTTLs | Positive subsecond TTL and no partial failed counter. | PTTL and EXISTS assertions. | Redis state observed. | Derived keys. | Bounded 500ms. | Good fixture. | Does not test lock rounding. | PASS |
| TestTask224ConfigurationBoundaries | Timing/cardinality policy. | Default, 36s, +1ns, +1ms, batch 2, 1ms, sub-ms. | Validation before Redis I/O. | No user data. | Constant-time table. | Exact Redis-effective boundaries are explicit. | Ten focused repetitions pass. | PASS |
| TestTask224RedisEffectiveLockTTLExceedsProcessingBoundary | Live lock must remain beyond the 35s work/finalization boundary. | Enqueue, reserve, SetNX, PTTL inside processor, and finalization. | Real Redis lock exists while ownership is held. | Generated ID and derived key. | 100ms observation margin avoids command-latency flake; deterministic +1ms validation covers minimum. | Direct live regression. | Focused and race repetitions pass. | PASS |
| TestTask210WorkerPublishesPartialAlternativesOnLaterSolverFailure | Worker fixture uses repaired default visibility. | Task 224 changes only visibility literal; prior changes excluded. | Worker lifecycle is context only. | Generated IDs. | Existing integration cost. | Default avoids obsolete 31s. | Full worker and aggregate race suites pass. | PASS |

Every listed non-trivial unit was checked for malformed values, errors, cleanup, cancellation, cross-process coordination, security boundaries, bounded work, API necessity, idioms, and adversarial tests. Lock-release observability and duplicate cleanup atomicity remain explicit Task 225 boundaries.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| [nit] | backend/internal/queue/job_queue.go:835-841 | countAttemptScript | `tonumber` accepts fractional or non-finite existing counter state and the script can write the incremented value before Go's `Int64` conversion reports an error. | A pre-existing internal value such as `1.5` is changed to `2.5` with PX, then `countAttempt` returns queue-unavailable on reply parsing. | Optional hardening: reject non-integral, finite, nonnegative, and bounded counters before `SET`, with a corrupt-state fixture. Not required by Task 224 and not blocking. |
| [nit] | backend/internal/queue/job_queue_integration_test.go:374-404 | TestTask224RejectsNonCanonicalJobIDsAndRemovesMalformedDeliveries | Decode-side real Redis coverage asserts only nil UUID; other rejected forms are enqueue-only. | A future enqueue/decode predicate divergence could pass the current fixture. | Add uppercase, braced, arbitrary, and control-character raw stream cases with XRANGE and XPENDING assertions. |

blocking_findings: 0
important_findings: 0
optional_findings: 2

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/queue -run 'TestTask224|TestJobQueue(EnqueueReserveAndAckUseRedisStreams|ReclaimsAbandonedDeliveryWithXAUTOCLAIM|RetriesAndTerminallyFailsAfterThreeAttempts)' -count=5 | backend/ | 0 | PASS | Focused Task 224 live-Redis tests repeated 5 times. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/queue -run '^TestTask224(ConfigurationBoundaries|RedisEffectiveLockTTLExceedsProcessingBoundary)$' -count=10 | backend/ | 0 | PASS | Exact repaired visibility and live PTTL boundaries. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/queue -run 'TestTask224|TestJobQueue(EnqueueReserveAndAckUseRedisStreams|ReclaimsAbandonedDeliveryWithXAUTOCLAIM|RetriesAndTerminallyFailsAfterThreeAttempts)' -count=10 | backend/ | 0 | PASS | Focused queue Redis/race evidence. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/queue ./internal/worker -count=1 | backend/ | 0 | PASS | Focused queue/worker race suites. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1 | backend/ | 0 | PASS | Final current-tree aggregate race gate after unrelated Task 223 snapshot update. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/queue -count=5 | backend/ | 0 | PASS | Entire queue package repeated 5 times. |
| gofmt -d internal/queue/job_queue.go internal/queue/job_queue_integration_test.go internal/worker/task210_swe5_integration_test.go | backend/ | 0 | PASS | No formatting diff. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./... | backend/ | 0 | PASS | No vet findings. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/queue -coverprofile=/tmp/task-224-queue-review.coverage && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go tool cover -func=/tmp/task-224-queue-review.coverage | backend/ | 0 | PASS | 75.6% package coverage; /tmp/task-224-queue-review.coverage. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./internal/queue | backend/ | 0 | PASS | No vulnerabilities found. |
| python3 scripts/validate-task-list.py | repository root | 0 | PASS | 237 sequential tasks and ordered dependencies. |
| python3 scripts/validate-traceability.py | repository root | 0 | PASS | Traceability validation passed. |
| git diff --check | repository root | 0 | PASS | No whitespace errors. |
| sed -n '20,48p' backend/.go-mod-cache/github.com/redis/go-redis/v9@v9.17.0/commands.go | repository root | 0 | PASS | `formatMs` floors duration to milliseconds; `SetNX` uses PX for non-whole-second TTLs. |
| python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-224-review.md | repository root | 0 | PASS | Evidence structurally valid after creation. |

## 9. Files Inspected and Staleness Fingerprints

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| backend/internal/queue/job_queue.go | Task 224 production queue | Optional fractional-counter hardening note only | SHA-256 | 7d350492d56e6471bd58470f67944681a84f204759be3bd616f27fca9dc2e6e7 |
| backend/internal/queue/job_queue_integration_test.go | Task 224 Redis/boundary tests | Optional decode-form coverage note only | SHA-256 | 1efabe0f7d76da03cac247206920f3fa4defcd661b49624ecf3c3a16713d0032 |
| backend/internal/worker/task210_swe5_integration_test.go | Shared worker fixture | No Task 224 finding in attributed literal | SHA-256 | eb7f1e1a66ed2b05a9b8cdd1fa6c9a924f4bc2e679e3d66496bbad6d8095ca6a |
| docs/implementation/preparation/task-224-preparation.md | Refreshed implementation and verification evidence | Prior blocking repair claim checked | SHA-256 | 8b7022dd052faeb2c47c6f50c446a25b2e65e7fe2694689b01e3cd8c4fd72680 |
| docs/design/DESIGN-004.md | Queue source of truth | Supports strict timing interpretation | SHA-256 | 688e9c18e398b3c83dd50f4066864cd48a9897003a7f78d030b6a531de6d81bc |
| docs/architecture/ARCH-004.md | Timeout architecture | Supports 45s visibility contract | SHA-256 | bedcf45c79f50cfe24313345ac2ba664130ee44d7d2f35c7b1de0a06e0abe867 |
| docs/implementation/02_TASK_LIST.md | Task row/status | Task remains OPEN | SHA-256 | 0dffbc3f92d5a4ada291fa413eb891ff1211bf68ea4c8b5aefc0ab669c58c6e0 |

all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The prior rejected review hashes and effective-TTL conclusion were stale after the repair; affected queue symbols were re-read and re-tested."
  - "Intermediate aggregate race failures were stale after the unrelated concurrent Task 223 snapshot update; the final current-tree race command passed."

## 10. Coverage and Exceptions

- [x] Required queue coverage command ran.
- [x] Report path and observed coverage are recorded.
- [x] Untested branches relevant to changed symbols were inspected manually.
- [x] No new Task 224 coverage exception is claimed; the existing Phase 07 queue exception is recorded in docs/implementation/04_OPEN.md.

coverage_required: true
coverage_exception_allowed: true
coverage_report_path: "/tmp/task-224-queue-review.coverage"
observed_line_coverage: "75.6 percent for internal/queue statements"
coverage_passed: true

Coverage finding: the command passes and new Task 224 branches are exercised, but the package remains below the 100% phase goal under the existing queue coverage deviation. The deterministic visibility-rounding boundary and live PTTL regressions are present; no new coverage exception is claimed.

## 11. Negative and Regression Checks

- [x] Focused queue tests pass repeatedly against real Redis.
- [x] No unrelated dependency or architectural boundary was introduced by Task 224; Task 225 script/topology/recovery work remains out of scope.
- [x] DESIGN-004 and ARCH-004 timing/ownership language was inspected.
- [x] No generated, cache, build, or temporary artifact was added by review.
- [x] Public queue additions are limited to the internal preparation helper and canonical validation required by the task.
- [x] Duplicate helpers and aliases were searched within the queue surface.
- [x] Error, cleanup, timeout, concurrency, and malformed-input paths were challenged.

Findings: the prior blocking effective-TTL finding is resolved. Two optional hardening/test-coverage notes remain visible.

## 12. Decision

A task may be PASSED only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains.

decision: "PASSED"
reason: "The repaired validation compares Redis-effective millisecond TTLs and the current audit reports no unresolved blocking or important finding."
failed_criteria: []
failed_or_unaudited_symbols: []
  - "Process"
  - "TestTask224ConfigurationBoundaries"
recommended_next_action: "Repair the visibility guard using Redis-effective millisecond precision, change the safe boundary to at least 36s+1ms, add a live Redis lock TTL assertion, then rerun focused tests and go test -race ./...."

## 13. Repair Context

### Failure Summary

The Go guard accepts 36s+1ns because 35s+1ns is greater than the 35-second work/finalization budget. The SetNX call passes that duration through go-redis formatMs, which floors it to 35000 milliseconds. The actual lock therefore expires at the exact finalization boundary and can be reclaimed while a worker is finishing.

### Minimal Repair Goal

Make validation compare the Redis-effective lock TTL, or require a minimum one-millisecond positive margin after rounding. Update the boundary test and add a live Redis assertion that the accepted minimum lock TTL is strictly greater than 35 seconds. Preserve passing reservation, reclaim-prefix, ownership, UUID, attempt, and TTL behavior.

### Evidence to Reuse

Reuse the current queue integration suite, /tmp/task-224-queue.coverage, current file hashes as the pre-repair snapshot, and fresh passing focused/full race results. Recheck all affected symbols and rerun the evidence validator after repair.

### Required Re-Review Surface

validate, lockTTL, Process, TestTask224ConfigurationBoundaries, the Redis lock acquisition path, DESIGN-004/ARCH-004 timing references, and all callers/tests constructing visibility timeouts. Refresh hashes for every changed implementation/test file.

### Do Not Change

Do not edit Task 224 status during repair or review. Do not broaden repair into Task 225 terminal publication, embedded scripts, duplicate cleanup atomicity, Redis topology, or stream/group recovery. Preserve current passing reservation, reclaim-prefix, ownership, UUID, attempt, and TTL behavior.
