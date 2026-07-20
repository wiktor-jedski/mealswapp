# Task 224 Preparation — Queue Reservation, Ownership, and Retry Contract

## Scope and attribution

- Task: `224`, Phase 07.01 Queue Reservation, Ownership, and Retry Contract.
- Design source: `docs/design/DESIGN-004.md`, static aspect `JobQueueManager`, plus the 30-second processing and bounded-finalization contract in `docs/architecture/ARCH-004.md`.
- Review-action source: queue actions 307–310 in `docs/implementation/04_OPEN.md`.
- Repair source: validated review `docs/implementation/reviews/task-224-review.md`, blocking effective-Redis-TTL finding only. The two optional review findings remain out of scope.
- Preparation date: 2026-07-17 (Europe/Warsaw).
- The worktree already contained concurrent Phase 07.01 changes. Existing Tasks 220–223 changes were preserved; no unrelated path was cleaned, reverted, staged, or rewritten.
- Task 224 remains `OPEN`. No task-list status or other task row was changed.
- Later Task 225 atomic finalization, bounded lock cleanup, embedded Lua resources/topology, dead-branch cleanup, and stream/group recovery remain out of scope. Task 226 queue-age correction also remains out of scope.

## Implemented contract

### Reservation and reclaim cardinality

`JobQueueManager` now validates `BatchSize == 1`. Both `XREADGROUP` and `XAUTOCLAIM` therefore assign at most the one delivery represented by `Reserve` and the configured reclaim operation. Values greater than one fail configuration validation before Redis I/O.

The shared delivery-preparation boundary returns the successfully decoded prefix together with a later error. This makes partial failure explicit and prevents already claimed/prepared work from disappearing behind a `nil` result. With the enforced production cardinality the public reclaim path normally has zero or one item; the real-Redis multi-entry fixture directly proves the prefix contract for a future/internal multi-message result and verifies malformed tail cleanup.

### Ownership-based processing attempts

Reservation and reclaim decode delivery metadata but leave `Job.Attempt` at zero. `Process` performs completion detection, acquires the logical-job lock, repeats completion detection under ownership, and only then advances the logical processing-attempt counter. The processor and terminal handler receive the owned copy with attempt `1`, `2`, or `3`.

Duplicate deliveries, lock misses, and deliveries already covered by a completion marker are acknowledged without creating or advancing attempt state. Retry telemetry is emitted only after an owned second/third processor invocation; exhaustion follows the third genuine invocation.

The counter update uses one Redis Lua execution whose sole mutation is `SET value PX ttl`. Existing state is parsed before mutation, and value plus expiry are installed by the same Redis command. An invalid TTL/script failure therefore cannot leave an incremented key without expiry.

### UUID trust boundary and malformed delivery removal

Enqueue, stream decode, direct processing, and acknowledgement accept only the lowercase hyphenated canonical form returned by `uuid.UUID.String()`, and reject the nil UUID. Empty, uppercase, braced, arbitrary, control-character, and nil IDs cannot reach the processor.

A malformed pending stream entry is removed with one Lua execution containing `XACK` and `XDEL`. The entry therefore leaves both the pending-entry list and stream instead of being acknowledged but retained. The operation uses no job ID in telemetry or errors.

### Visibility and TTL precision

The queue validates the complete ownership window using the millisecond precision Redis receives: the processing lock (`visibility - 1 second`) is truncated to whole milliseconds before comparison with the documented 30-second whole-job deadline plus 5-second finalization budget. A 36-second visibility timeout, `36s + 1ns`, and every sub-millisecond surplus are rejected because their Redis-effective lock expires exactly at the 35-second boundary. `36s + 1ms` is the minimum accepted visibility and encodes a 35,001-millisecond lock. The production default remains 45 seconds visibility and 44 seconds lock ownership.

The live Redis regression acquires the production `SET NX` lock through `Process` and observes its `PTTL` inside the processor while ownership is held. It verifies a near-boundary accepted configuration retains a measurable TTL strictly beyond 35 seconds; deterministic validation cases separately pin the exact rejected and accepted millisecond boundaries without depending on command latency.

Completed, attempt, and enqueue-marker TTL arguments now use Redis millisecond precision (`PX`). TTLs from 1 millisecond upward are accepted, including sub-second values; sub-millisecond values are rejected before Redis I/O. No accepted duration can become `EX 0` through integer truncation.

## Changed Task 224 surfaces

| Path | Task 224 surface |
| --- | --- |
| `backend/internal/queue/job_queue.go` | strict single-message batch validation; explicit prepared-prefix return; canonical UUID validation; atomic malformed `XACK`/`XDEL`; ownership-before-attempt ordering; atomic attempt value/TTL update; millisecond enqueue/finalization TTLs; Redis-effective millisecond visibility/lock/finalization validation |
| `backend/internal/queue/job_queue_integration_test.go` | real-Redis UUID rejection/removal, partial prepared-prefix, duplicate/lock/completed no-attempt, exact three-attempt/telemetry, atomic failure, millisecond TTL, exact visibility boundaries, and live lock `PTTL` fixture |
| `backend/internal/worker/task210_swe5_integration_test.go` | fixture-only replacement of the obsolete 31-second visibility value with `queue.DefaultVisibilityTimeout`; all pre-existing Task 210/220–223 changes in the file are preserved |
| `docs/implementation/preparation/task-224-preparation.md` | this scoped implementation and verification evidence |

`docs/design/DESIGN-004.md`, `docs/implementation/02_TASK_LIST.md`, and every Task 220–223 implementation surface were read and preserved without Task 224 edits.

## Verification-criteria mapping

| Task 224 criterion | Evidence | Result |
| --- | --- | --- |
| Reservation cardinality is explicit | `validate` requires `BatchSize == 1`; boundary test rejects `2` | PASS |
| Reclaim partial failure is explicit | `prepareDeliveries` returns prepared prefix plus error; real Redis fixture prepares a valid pending entry before a malformed tail and retains the valid job | PASS |
| Duplicate, contention, and completed deliveries consume no attempt | real Redis duplicate stream entry, external lock, and completion-marker fixture; processor is not called and attempt keys remain absent | PASS |
| Three genuine attempts | retry fixture observes processor attempts exactly `[1, 2, 3]`, one terminal call, and telemetry `[retry, retry, exhausted]` | PASS |
| Attempt and expiry are atomic | `countAttemptScript` performs one `SET ... PX`; invalid-TTL execution leaves no key | PASS |
| UUIDs are canonical and non-nil | enqueue table and malformed stream fixture reject empty/nil/uppercase/braced/arbitrary/control-character values | PASS |
| Malformed entries are fully removed | real Redis `XRANGE` and `XPENDING` both show no malformed delivery after decode rejection | PASS |
| Visibility/finalization/lock window is coherent at Redis precision | 36 seconds, `+1ns`, and sub-millisecond surplus rejected; `+1ms` accepted; live production `SET NX` lock has `PTTL > 35s`; default remains 45/44 seconds | PASS |
| TTL precision cannot truncate to zero | 500-millisecond production execution produces positive attempt/completion/enqueue PTTLs; 1 millisecond accepted and smaller durations rejected | PASS |
| Race safety | focused queue and worker race suites pass | PASS |

## Commands and results

| Working directory | Command | Result |
| --- | --- | --- |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/queue -run '^TestTask224(ConfigurationBoundaries\|RedisEffectiveLockTTLExceedsProcessingBoundary)$' -count=1` before the production repair | FAIL as intended: `+1ns` and sub-millisecond surplus were accepted |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/queue -run '^TestTask224(ConfigurationBoundaries\|RedisEffectiveLockTTLExceedsProcessingBoundary)$' -count=10` after the repair | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/queue -run 'TestTask224\|TestJobQueue(EnqueueReserveAndAckUseRedisStreams\|ReclaimsAbandonedDeliveryWithXAUTOCLAIM\|RetriesAndTerminallyFailsAfterThreeAttempts)' -count=10` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/queue -coverprofile=/tmp/task-224-queue.coverage` | PASS; 75.6% statements under the existing Phase 07 queue coverage deviation |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS |
| repository root | `python3 scripts/validate-task-list.py` | PASS; 237 sequential tasks with ordered dependencies |
| repository root | `python3 scripts/validate-traceability.py` | PASS |
| repository root | `git diff --check` | PASS |

The preparation's previously recorded aggregate-race failure no longer reproduces on the current worktree. No Task 223 implementation or test surface was edited during this repair, and the passing full race suite exercises those retained changes.

## Current SHA-256 snapshot

| Path | SHA-256 | Attribution |
| --- | --- | --- |
| `backend/internal/queue/job_queue.go` | `7d350492d56e6471bd58470f67944681a84f204759be3bd616f27fca9dc2e6e7` | Task 224 production implementation including effective-millisecond validation repair |
| `backend/internal/queue/job_queue_integration_test.go` | `1efabe0f7d76da03cac247206920f3fa4defcd661b49624ecf3c3a16713d0032` | Task 224 focused real-Redis, live PTTL, and exact boundary evidence |
| `backend/internal/worker/task210_swe5_integration_test.go` | `eb7f1e1a66ed2b05a9b8cdd1fa6c9a924f4bc2e679e3d66496bbad6d8095ca6a` | shared pre-existing Task 210/220+ file; Task 224 changes only its visibility fixture |
| `docs/design/DESIGN-004.md` | `688e9c18e398b3c83dd50f4066864cd48a9897003a7f78d030b6a531de6d81bc` | preserved shared design source |
| `docs/implementation/02_TASK_LIST.md` | `0dffbc3f92d5a4ada291fa413eb891ff1211bf68ea4c8b5aefc0ab669c58c6e0` | preserved shared task/status source; Task 224 remains `OPEN` |

This preparation document intentionally omits its own hash.

## Residual boundaries

- Task 225 still owns requiring terminal publication, modeling completed/failed acknowledgement separately, bounded observable lock cleanup, atomic duplicate cleanup, zero-ack semantics, script embedding/cache fallback, Redis topology, and live stream/group recovery. Task 224 adds only the atomic scripts necessary for attempt accounting and malformed-delivery removal; extraction is deliberately deferred.
- Task 226 still owns accurate waiting-versus-pending queue age measurement.
- Redis Streams delivery count remains distinct from processing-attempt count. Reservation/reclamation alone does not consume retry budget.
- The queue uses Redis Lua and multi-key scripts under the repository's current standalone Redis assumption. Cluster hash-slot design remains explicitly assigned to Task 225.

## Preparation decision

The validated review's sole blocking finding is repaired: accepted visibility configurations now preserve a strictly positive Redis-effective millisecond lock margin beyond solve/finalization, with deterministic and live-Redis regression evidence. Existing reservation, reclaim-prefix, ownership, UUID, attempt, TTL, Task 223, and later-task boundaries are preserved. Per explicit instruction, `docs/implementation/02_TASK_LIST.md` was not edited and Task 224 remains `OPEN`.
