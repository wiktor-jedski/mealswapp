# Task 208 Re-review

## Decision

**PASSED**

No blocking correctness, security, behavior-regression, or missing-test findings remain within Task 208.

## Repair verification

### Ceiling nearest-rank P95 — Pass

`scripts/verify-optimization-capacity.py` calculates the rank with `math.ceil(len(values) * 0.95) - 1`. At the default 32-sample boundary, `[0.1] * 30 + [2.1, 2.2]` now produces P95 `2.1`, so the documented two-second limit fails rather than being hidden by floor rounding. The capacity gate rejects both submission and poll P95 values greater than or equal to `2.0` seconds. A focused regression test covers the 32-sample boundary and threshold behavior.

### Fail-closed capacity evidence — Pass

The capacity gate now requires:

- every requested submission to produce `202` and a submission latency sample;
- at least one poll latency sample;
- a stopped monitor with no recorded monitor exception;
- at least one readiness sample, with every sample valid;
- HTTP 200 readiness with `redis`, `worker`, and `optimization_queue` all `ok`;
- non-negative, finite queue depth/oldest queued age/oldest pending age evidence;
- queue samples matching the number of valid readiness samples and a non-empty Redis/worker/queue evidence tuple.

Missing polls, absent evidence, monitor failure, malformed queue values, or any degraded readiness sample therefore returns non-zero. `scripts/test_verify_optimization_capacity.py` covers each rejection path and the complete passing case.

## Original Task 208 criteria

- **Privacy labels — Pass:** optimization metric names, label keys, and label values are allow-listed. Logs retain only fixed non-sensitive fields. Tests verify attempted user/diet identifiers are dropped and emitted metrics remain low-cardinality.
- **Queue metrics — Pass:** production paths record queue depth, oldest queued age, and oldest pending age without queue payloads, job IDs, diet contents, or user PII.
- **Worker telemetry — Pass:** active-worker count, normalized utilization, solver duration/status, job outcomes, retries/exhaustion, timeout, infeasible, and result-expiry outcomes are wired into worker and job-store paths with bounded vocabularies.
- **Readiness degradation — Pass:** `/ready` reports PostgreSQL, Redis, worker heartbeat, and optimization queue independently. Redis errors, queue-stat errors, and missing/stale worker heartbeat produce unavailable checks and HTTP 503. Focused readiness and real-Redis heartbeat tests pass.
- **Thresholds and limits — Pass:** the capacity document records API latency, queue depth/age, worker utilization, and solver thresholds; solver deadline, visibility timeout, retry budget, TTL, heartbeat timing, and local load defaults; and limits the local evidence claim relative to SW-REQ-082 production scaling.
- **Repeatable load evidence — Pass:** the authenticated script concurrently submits unique idempotency keys, polls accepted jobs, probes readiness, emits only bounded operational evidence, and now fails closed. The live run remains operator-executed because fixture credentials are intentionally not stored in the repository.

## Verification performed

- `python3 -m unittest scripts/test_verify_optimization_capacity.py` — passed, 8 tests.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./...` — passed.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/observability ./internal/queue ./internal/worker ./internal/httpapi ./internal/app` — passed.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` — passed.
- `cd backend && CGO_ENABLED=0 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go build -o /tmp/mealswapp-worker ./cmd/worker` — passed.
- `python3 scripts/validate-task-list.py` — passed: 212 sequential tasks with ordered dependencies.
- `python3 scripts/validate-traceability.py` — passed.

## Scope

Re-reviewed exactly Task 208 after repair, with dependencies 203 and 204 already PASSED. Only this review document was overwritten; task list, code, and later tasks were not edited.
