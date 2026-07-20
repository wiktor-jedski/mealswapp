# Task 226 Preparation — Queue Age Measurement Correctness

## Scope and worktree attribution

- Task: `226`, Phase 07.01 Queue Age Measurement Correctness.
- Authoritative static aspect: `DESIGN-014: MetricsCollector`, with the Redis population boundary owned by `DESIGN-004: JobQueueManager`.
- The authoritative row in `docs/implementation/02_TASK_LIST.md` remained `OPEN`; this implementation did not edit the row or any task status.
- The worktree already contained concurrent Phase 07.01 edits, including the Task 223 observability surface and Tasks 224–225 queue surface. They were preserved. Attribution below uses hashes captured before Task 226 edits rather than the aggregate dirty-worktree diff.
- An initially considered attempt/outcome-contract expansion was stopped after scope correction. Its patch failed before application, so no processor, retry-outcome, terminal-publication, worker, or acknowledgement interface was changed by Task 226.

## Implemented behavior

`JobQueueManager.Stats` now separates the two Redis consumer-group populations:

- Waiting depth is the group's Redis-reported `Lag`; pending depth is Redis `XPENDING` count; total queue depth remains their sum.
- Waiting age is read from the first `XRANGE` entry strictly after the group's `LastDeliveredID`. A pending stream entry can no longer become `OldestQueuedAge`.
- Pending age is the greatest `Idle` duration returned by extended `XPENDING` metadata. It is no longer recomputed from a stream ID and the application clock.
- Extended pending metadata is scanned in pages of 100, preserving exact oldest-idle selection without issuing one unbounded Redis response.
- Empty waiting or pending populations report zero.
- A Redis stream timestamp ahead of the application clock produces zero waiting age. `OptimizationTelemetry.QueueStats` independently clamps negative durations before metric or event delivery.
- Missing, incomplete, or indeterminate consumer-group metadata fails closed as `ErrQueueUnavailable` instead of publishing plausible zero measurements.

Queue telemetry remains low-cardinality:

- Depth: `optimization_queue_depth`, unit `jobs`, no labels.
- Ages: `optimization_queue_age_seconds`, unit `seconds`, exactly one `kind` label.
- The only age kinds are `oldest_queued` and `oldest_pending`; unknown values and additional labels are rejected by the metric allowlist.
- `OptimizationTelemetry.Record` enforces `optimization_queue_depth` → `jobs` and `optimization_queue_age_seconds` → `seconds` before reaching the sink, so direct adapters cannot override the Task 226 units.

## Rejected finding repair

Review finding `F-226-01` reproduced because `OptimizationTelemetry.Record` passed the caller-supplied unit through `record`, while `validOptimizationMetric` checked only the metric name and labels. A focused red test proved that valid queue names and bounded labels still emitted depth with `seconds` or an empty unit and age with `milliseconds` or `jobs`.

The repair passes the unit into `validOptimizationMetric` and rejects a queue metric unless its name/unit pair is exactly one of the two documented Task 226 pairs. Existing fixed callers, queue measurements, non-queue telemetry validation, and the bounded queue label allowlist remain unchanged. The regression test also calls `OptimizationTelemetry.Record` directly with all three valid queue points before proving both age kinds and depth cannot be recorded with invalid or mismatched units.

## Exact files and symbols

| File | Task 226 symbols/surfaces |
|---|---|
| `backend/internal/queue/job_queue.go` | `(*JobQueueManager).Stats`; new `(*JobQueueManager).oldestPendingIdle`; retained `streamEntryAge` nonnegative skew clamp |
| `backend/internal/queue/task226_queue_age_test.go` | `TestTask226StatsSeparateWaitingAndPendingAges`; `TestTask226StatsPopulationAndClockSkewFixtures`; `task226Stats`; `addTask226Delivery` |
| `backend/internal/observability/optimization.go` | `(*OptimizationTelemetry).QueueStats` clamps both age inputs before fixed-name/unit/label emission; `(*OptimizationTelemetry).Record`, `(*OptimizationTelemetry).record`, and `validOptimizationMetric` enforce the queue name/unit/label contract at the public recording boundary |
| `backend/internal/observability/task226_queue_age_test.go` | `TestTask226QueueAgeMetricsUseExactUnitsAndBoundedLabels`; `TestTask226RecordRejectsMismatchedQueueMetricUnits` |
| `docs/design/DESIGN-004.md` | Queue waiting/pending population, metadata source, empty-state, and skew policy |
| `docs/design/DESIGN-014.md` | Exact queue metric names, units, labels, metadata sources, nonnegative policy, and `QueueStats` interface |

## Acceptance evidence

| Required behavior | Real/focused evidence | Result |
|---|---|---|
| Queued-only | One old stream entry remains after `LastDeliveredID`; queued age tracks its stream timestamp and pending age is zero | PASS |
| Pending-only | Two deliveries are reserved at different times; queued age is zero and oldest pending age is the greater Redis idle duration, not either 19–20 second stream age | PASS |
| Mixed | A 10-second-old entry is pending and a 1-second-old entry waits; queued age follows only the waiting entry while pending age remains near Redis idle time | PASS |
| Empty | Bootstrapped empty stream/group returns zero depths and ages | PASS |
| Skewed clock | A future explicit Redis stream ID remains waiting and reports zero, never a negative duration | PASS |
| Authoritative pending age | Assertions distinguish tens-of-milliseconds Redis idle from deliberately old stream timestamps | PASS |
| Bounded telemetry | Exact depth/age names and units are asserted; direct `Record` calls accept only `jobs` for depth and `seconds` for both bounded age kinds; empty/mismatched units, unknown kinds, and extra labels are dropped | PASS |
| Regression/race safety | Full backend normal and race suites pass, including existing duplicate-delivery, atomic finalization, and bounded lock-cleanup tests | PASS |

## Verification performed

| Command | Result |
|---|---|
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/queue -run '^TestTask226' -count=10` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/observability -run '^TestTask226' -count=10` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/queue ./internal/observability -count=1` | PASS; real Redis queue fixtures executed |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/queue ./internal/observability -count=1` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./internal/queue ./internal/observability` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -count=1` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1` | PASS |
| `python3 scripts/validate-traceability.py` | PASS — traceability validation passed |
| `python3 scripts/validate-task-list.py` | PASS — 237 sequential tasks with ordered dependencies |

### Focused rejected-finding repair verification

| Command | Result |
|---|---|
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/observability -run '^TestTask226RecordRejectsMismatchedQueueMetricUnits$' -count=1` before the production repair | EXPECTED FAIL — all four mismatched/empty-unit cases reached `MemorySink`, reproducing `F-226-01` |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/observability -run '^TestTask226' -count=10` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/observability -count=1` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/observability -run '^TestTask226' -count=1` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./internal/observability` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -count=1` | PASS |

## Hash evidence

### Pre-Task-226 baseline

| File | SHA-256 before Task 226 |
|---|---|
| `backend/internal/queue/job_queue.go` | `11b8432e23042c47de4416e5025ecdf949e03bdd62594e5dc9c30feb368b188f` |
| `backend/internal/observability/optimization.go` | `6e158b2b8f476ab63c25dbd55c2980a8b6ba20ef252b35fccc69c5c610239897` |
| `docs/design/DESIGN-004.md` | `7fcd70141966404cab51a014512e32b4e8681551e2a0248ebaf01f18d0fab547` |
| `docs/design/DESIGN-014.md` | `f913c8087efc1d9e928b316d821cb88f2710316e232d482e7bfe6f963019a2ee` |
| `backend/internal/queue/task226_queue_age_test.go` | absent |
| `backend/internal/observability/task226_queue_age_test.go` | absent |

### Verified Task-226 state before rejected finding repair

| File | SHA-256 |
|---|---|
| `backend/internal/queue/job_queue.go` | `df27156ee125c8e4e62f090eb6be9506afc8bd35d76c7b61d3c3c77800d22e07` |
| `backend/internal/queue/task226_queue_age_test.go` | `4f7c8ce3ce102c0f5f7e52814c6c1d5b7cd895489538c61223668afa9753b58d` |
| `backend/internal/observability/optimization.go` | `eabe9aaeadb8699be7dfcf9590109e7d3c42d0429efa5bf394f2374e06bb9906` |
| `backend/internal/observability/task226_queue_age_test.go` | `564cf38e7bd3b6fe31625ebd74ec325e96f09fd844b4259009bdb05ea8b14f7c` |
| `docs/design/DESIGN-004.md` | `45dd31e2afe1480ec54f540031ec45289f8287fbd0ef1a2d2cee86dea16a5474` |
| `docs/design/DESIGN-014.md` | `f9f6521d89e6d31306422017e07af5630ba4d8da56907174f3653ea0d72e9fe4` |

### Repaired state

| File | Repaired SHA-256 |
|---|---|
| `backend/internal/observability/optimization.go` | `88c0e988d043abf4b44266302b74e68780c3af589eae0022e21608974332d042` |
| `backend/internal/observability/task226_queue_age_test.go` | `5dc75753fc82fabed79d442b1457c2e21da63f28e82216875c21fe8d8a4ce67b` |

## Concerns and boundaries

- Exact oldest pending idle requires inspecting each pending entry because Redis does not return the maximum idle duration in the summary response. The implementation pages `XPENDING` metadata 100 entries at a time; command response size is bounded per page, while total work remains linear in pending depth.
- Queue state can change between Redis metadata calls. Each age is authoritative for the metadata returned by its own command, but `Stats` is an operational snapshot rather than a cross-command Redis transaction.
- Waiting age necessarily compares the Redis-generated stream timestamp with the application clock. The documented fail-safe is a zero clamp for future timestamps; pending age avoids this clock comparison entirely by using Redis idle duration.
- Unit enforcement is deliberately limited to the two Task 226 queue metric names. Non-queue metric contracts and behavior are outside this rejected-finding repair.
- No Task 225 finalization, duplicate-delivery, lock-cleanup, retry, worker, or terminal-publication behavior was changed. No unrelated task or task status was modified.
