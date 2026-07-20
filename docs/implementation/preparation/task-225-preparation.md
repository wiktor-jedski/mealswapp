# Task 225 Preparation — Atomic Queue Finalization and Recovery

## Scope and attribution

- Task: `225`, Phase 07.01 Atomic Queue Finalization and Recovery.
- Design source: `docs/design/DESIGN-004.md`, static aspects `JobQueueManager` and `JobStatusTracker`; `docs/architecture/ARCH-004.md` supplies the asynchronous worker boundary.
- Preparation date: 2026-07-17 (Europe/Warsaw).
- Retry inputs read: task row 225, the complete relevant DESIGN-004 queue contract, `task-221-preparation.md`, `task-223-preparation.md`, `task-224-preparation.md`, and `task-224-review.md`. No prior `task-225-preparation.md` or `task-225-review.md` existed.
- The worktree already contained concurrent Phase 07.01 edits and a partial untracked Task 225 implementation. Those files were audited and completed in place. No unrelated path was cleaned, reverted, staged, or rewritten.
- `docs/implementation/02_TASK_LIST.md` was not edited; Task 225 remains `OPEN` as explicitly required.

## Implemented contract

### Publication-before-acknowledgement

`Processor` and `TerminalHandler` return the closed `TerminalPublication` vocabulary: `PublishedCompleted` or `PublishedFailed`. `Process` finalizes only after a valid publication confirmation. A nil/unknown confirmation, a publication paired with an error, failed exhaustion publication, or a missing exhaustion handler leaves the delivery pending. The third unknown worker failure calls the terminal handler, requires `PublishedFailed`, and only then runs queue finalization. `OptimizationProcessor` returns the confirmation only after the Redis job store has published the authoritative completed/failed state and admission ownership has been released.

`AckCompleted` and `AckFailed` are separate entry points. The former generic `Ack` API and the dead `ErrJobInProgress` branch are absent.

### Atomic finalization and duplicate cleanup

`finalize.lua` checks an immutable terminal marker, executes `XACK`, conditionally writes the matching marker with millisecond TTL, and executes `XDEL` in one Redis script. `XACK=0` succeeds only when the same marker already exists. Without a marker it returns `-1`, writes nothing, and retains the stream entry; a contradictory marker returns `-2` before mutation.

`remove_delivery.lua` performs duplicate/malformed `XACK` and `XDEL` together. Repeated `{0,0}` execution is explicit idempotent success. Concurrent real-Redis cleanup proves no pending or stream entry survives.

### Bounded observable ownership cleanup

`releaseLock` executes the owner-token Lua script under `context.WithTimeout(context.WithoutCancel(ctx), 100*time.Millisecond)`. A Redis failure cannot replace the processing result and records only `optimization_queue_cleanup_total{outcome="failed"}` plus the matching bounded event. The telemetry has no job, stream entry, consumer, user, key, or payload label. It reuses Task 223's capped cleanup-delivery lane so a non-cooperative sink cannot create unbounded goroutines.

### Embedded scripts and Redis topology

Five nonempty, DESIGN-004-traced Lua resources are compiled into the worker with `go:embed`: enqueue, finalization, attempt counting, duplicate removal, and owner-safe lock release. `redis.NewScript` provides `EVALSHA`-first cached execution and automatic `EVAL` fallback on `NOSCRIPT`; a real `SCRIPT FLUSH` fixture verifies the fallback without a runtime source-file dependency.

The approved topology supports standalone Redis and Redis Cluster. The default stream is `mealswapp:optimization:{queue-v1}:jobs`; custom streams must contain a nonempty hash tag. Every queue-owned enqueue, terminal, attempt, and lock key derives the same tag, so every multi-key script remains in one Redis Cluster slot. Real standalone Redis executes enqueue/finalize/attempt scripts and the fixture checks all generated keys carry the stream tag.

### Live stream/group recovery

Bootstrap is serialized per manager and treats `BUSYGROUP` as idempotent success. Only `NOGROUP` invalidates cached bootstrap state. Reserve/reclaim recreates the stream/group at `0` and retries the failed Redis command once; authorization, connectivity, script, and other errors remain `ErrQueueUnavailable` and do not trigger permissive recovery.

Recovery evidence covers deletion of a live group with surviving entries, concurrent recovery calls, empty stream/group state, and a real non-persistent Redis process restart. The restart fixture uses an isolated local `redis-server` or an already-installed `redis:7-alpine` image on a private port; it does not stop, flush, or restart the shared project Redis.

## Exact changed files and symbols

| Path | Task 225 symbols or surface |
| --- | --- |
| `backend/internal/queue/job_queue.go` | `ErrTerminalPublicationRequired`; `TerminalPublication`; `PublishedCompleted`; `PublishedFailed`; typed `Processor`/`TerminalHandler`; serialized `Bootstrap`; `Process`; `AckCompleted`; `AckFailed`; `ackPublished`; `claim`; `autoClaim`; `readNew`; `recoverGroup`; `finalize`; `removeDelivery`; bounded `releaseLock`; hash-tagged `key`; strict topology `validate`; `isNoGroup`; `streamHashTag`; removal of `ErrJobInProgress`, generic `Ack`, inline Lua, and pending-marker branch |
| `backend/internal/queue/queue_scripts.go` | embedded Lua resources and five cached `redis.Script` values |
| `backend/internal/queue/lua/enqueue.lua` | atomic idempotent enqueue using stream/marker keys in one slot |
| `backend/internal/queue/lua/finalize.lua` | immutable terminal marker plus atomic `XACK`/`SET PX`/`XDEL`, including zero-ack/conflict return codes |
| `backend/internal/queue/lua/count_attempt.lua` | embedded Task 224 atomic attempt/TTL script |
| `backend/internal/queue/lua/remove_delivery.lua` | atomic duplicate/malformed `XACK`/`XDEL` |
| `backend/internal/queue/lua/release_lock.lua` | owner-token lock deletion |
| `backend/internal/queue/task225_queue_test.go` | eight Task 225 real-Redis, restart, fallback, race, topology, failure, and observability tests |
| `backend/internal/queue/job_queue_integration_test.go` | existing queue tests migrated to typed terminal publication and distinct acknowledgement; exhaustion verifies failed publication before zero pending |
| `backend/internal/observability/optimization.go` | `MetricOptimizationQueueCleanup`; `queueCleanup`; `QueueCleanupFailed`; fixed metric/event allowlists |
| `backend/internal/worker/optimization_processor.go` | `ProcessOptimizationJob`, `Process`, `Terminal`, `publishFailure`, and `confirmTerminal` return deliberate terminal publication after authoritative status persistence |
| `backend/internal/worker/worker.go` | worker composition accepts one typed terminal handler and uses typed processor publication |
| `backend/internal/worker/optimization_processor_deadline_test.go` | publication assertions for terminal deadline behavior |
| `backend/internal/worker/worker_integration_test.go` | worker startup composes `processor.Terminal` with the queue manager |
| `backend/internal/worker/task210_swe5_integration_test.go` | compatibility call site for the typed queue processor contract |
| `backend/internal/app/task206_backend_integration_test.go` | compatibility duplicate-processor fixture returns explicit completed publication |
| `docs/design/DESIGN-004.md` | queue topology, script caching/fallback, terminal publication, zero-ack, cleanup, and live recovery policy plus distinct interfaces |
| `docs/implementation/preparation/task-225-preparation.md` | this scoped implementation and verification evidence |

Shared worker/design files also contain retained Task 220–224 work. The table attributes only the named Task 225 symbols and compatibility call sites, not each whole-file diff.

## Verification criteria

| Task 225 criterion | Evidence | Result |
| --- | --- | --- |
| Exhaustion cannot ACK before failed publication | missing/invalid terminal handler retains one pending delivery and no marker; explicit failed publication then finalizes | PASS |
| Completed and failed ACKs are distinct | separate APIs, matching-marker idempotency, and conflicting completed-after-failed rejection | PASS |
| Lock cleanup is bounded and observable | dead endpoint returns within the bound and fixed cleanup telemetry is observed | PASS |
| Duplicate acknowledgement/deletion is atomic | one Lua script; 16 concurrent real-Redis removals leave zero pending and zero stream entry | PASS |
| Zero-ack behavior is explicit | matching marker succeeds; no marker returns queue unavailable without marker creation or deletion; conflict fails before mutation | PASS |
| Dead branches are absent | repository search finds no `ErrJobInProgress`, generic `Ack`, `__pending__`, or inline queue script constants | PASS |
| Lua is embedded, cached, and self-contained | five traced nonempty embedded resources; `SCRIPT FLUSH` followed by successful enqueue/process proves fallback | PASS |
| Standalone/cluster topology is approved | all multi-key paths share the required stream hash tag; real standalone Redis executes all atomic paths | PASS |
| Live manager recovers safely | group deletion, concurrent recovery, data loss, and isolated Redis process restart pass; authorization/connectivity fail closed | PASS |
| Real Redis and race behavior | focused Task 225 race suite and complete backend race suite pass | PASS |

## Commands and results

| Working directory | Command | Result |
| --- | --- | --- |
| `backend/` | `go test ./internal/queue` | PASS |
| `backend/` | `go test -v ./internal/queue -run 'TestTask225LiveManagerRecoversAfterRedisRestart' -count=1` | PASS using isolated `redis:7-alpine`; real process stop/start and reconnect |
| `backend/` | `go test -race ./internal/queue -run 'TestTask225' -count=1` | PASS |
| `backend/` | `go test -race ./internal/queue ./internal/worker` | PASS |
| `backend/` | `go test ./...` | PASS |
| `backend/` | `go test -race ./... -count=1` | PASS |
| `backend/` | `go vet ./...` | PASS |
| `backend/` | focused Task 225/reclaim/exhaustion tests with `-coverprofile=/tmp/task-225-queue.coverage` | PASS; focused selection covers 56.1% of the complete queue package |
| repository root | `python3 scripts/validate-task-list.py` | PASS; 237 sequential tasks with ordered dependencies; no status edit |
| repository root | `python3 scripts/validate-traceability.py` | PASS after adding adjacent Go Doc/design comments to new declarations |
| repository root | `git diff --check` | PASS |

## Current SHA-256 snapshot

| Path | SHA-256 |
| --- | --- |
| `backend/internal/queue/job_queue.go` | `11b8432e23042c47de4416e5025ecdf949e03bdd62594e5dc9c30feb368b188f` |
| `backend/internal/queue/job_queue_integration_test.go` | `4eeb0a386fcc6fdc52b2a60e38920f3f0e7cb9233a94e381d9671a502263b420` |
| `backend/internal/queue/queue_scripts.go` | `7c34e5d78d1c73c8af3a3e74c751776cf57cd62f15024a61aeb3a357137f75d0` |
| `backend/internal/queue/task225_queue_test.go` | `b9c35c96fb1972de5c48a96daa28ae0a26b9a4f8b909ab26bcfae5435d7ad9c5` |
| `backend/internal/queue/lua/count_attempt.lua` | `ee8e8dfad99e8769334b68af3a3f30b9079b2951c975499a1c6cfaebdd30babb` |
| `backend/internal/queue/lua/enqueue.lua` | `f19c60dc07acaff742ebe303c7e93e046715a4c4891beb10e53c87a293e481f0` |
| `backend/internal/queue/lua/finalize.lua` | `068940df68ff162eebf99ab9504fa0414c56afbb96092114104059060f2aef1c` |
| `backend/internal/queue/lua/release_lock.lua` | `7d1168dbd643bc876cf9667e9ba84f921808815bc9286fc22eb483645d405a06` |
| `backend/internal/queue/lua/remove_delivery.lua` | `1bd216e4f0b34ed6f85361cc8e88f0656e9b15351e3c950a12727394f7b3460a` |
| `backend/internal/observability/optimization.go` | `6e158b2b8f476ab63c25dbd55c2980a8b6ba20ef252b35fccc69c5c610239897` |
| `backend/internal/worker/worker.go` | `5bcafbac328f824c26a1fd298a6a7dbd4091350a5e6f0b65fb25056847a86663` |
| `backend/internal/worker/optimization_processor.go` | `f3e42d4eadbb1ade39410da510c6144a3204a749c45353ab5a44fb39ad6971e0` |
| `backend/internal/worker/optimization_processor_deadline_test.go` | `2c7601ef63fc4c6e46d256c958251f8cfc655cdec6488a707e87339a767c1d24` |
| `backend/internal/worker/worker_integration_test.go` | `935c3caa093d220942a7e468355881ad38f94d29d36f18bc983bde08bda1615c` |
| `backend/internal/worker/task210_swe5_integration_test.go` | `3f76d08b34d74a3ac965cb75e285d60003e3d638ef843b420ef84ab47f2599f7` |
| `backend/internal/app/task206_backend_integration_test.go` | `f9e0de887d0670b914267730d36a8f897bff42635db7d41c5f89fd9865ff6629` |
| `docs/design/DESIGN-004.md` | `7fcd70141966404cab51a014512e32b4e8681551e2a0248ebaf01f18d0fab547` |
| `docs/implementation/02_TASK_LIST.md` | `bc8e6eff2a09cd85f3a1135e81ce7abe657caf347cc5d91a325528121a787fbe` (read-only concurrent snapshot; Task 225 remains `OPEN`) |

This preparation document intentionally omits its own hash.

## Remaining concerns and boundaries

- The focused Task 225 selection covers 56.1% of the whole queue package; the complete package and backend suites pass, but the phase-wide 100% coverage gate remains Task 235 ownership. No new coverage exception is accepted here.
- Cluster safety is proven structurally by one required hash tag across every script key and by standalone execution of the same scripts. A live multi-node Redis Cluster fixture is not present in this repository and was not introduced solely for this task.
- Queue-age semantics remain Task 226. This task does not alter waiting-versus-pending age calculation.
- The restart test skips only when neither a local `redis-server` nor an already-installed `redis:7-alpine` Docker image can run. In this verification environment the Docker-backed real restart path ran and passed.

## Preparation decision

The implementation satisfies Task 225's publication ordering, distinct terminal acknowledgements, bounded observable cleanup, atomic finalization/duplicate removal, embedded cached scripts, cluster-safe key topology, fail-closed recovery, and real Redis/race criteria. No task-list status was changed.
