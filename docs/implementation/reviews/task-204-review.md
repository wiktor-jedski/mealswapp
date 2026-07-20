# Task 204 Review

## Recommendation

**PASSED**

No blocking correctness, security, regression, or test-coverage findings remain in the Task 204 scope.

## Repair verification

### Durable claim and recovery — Pass

Submission persists the idempotency record, including the server-created job ID and exact response, before saving or publishing the job (`backend/internal/httpapi/optimization_controller.go:122-174`). A competing controller that loses the durable insert reloads the winning record and reuses its job ID. A retry after a crash or queue outage repeats the same save/enqueue sequence, allowing publication to recover without allocating a second logical job.

The cross-controller test synchronizes two independent controllers at the claim boundary and verifies both receive the same acknowledgement while only one idempotency record and one job remain.

### Idempotent queue publication — Pass

`JobQueueManager.Enqueue` uses one Redis Lua operation to check the logical-job publication marker, append to the stream, and retain the resulting stream entry ID (`backend/internal/queue/job_queue.go:215-235,755-768`). Concurrent or recovery publication for the same server job ID therefore returns the original entry rather than appending another entry. Failed `XADD` does not create the marker, so a later retry can recover.

### Atomic monotonic transitions — Pass

All Redis job writes go through `optimizationStateTransitionScript` (`backend/internal/worker/optimization_processor.go:180-295,534-569`). The script atomically reads and guards the current state: save preserves existing queued state, processing accepts only queued/processing, terminal publication accepts only processing, and terminal states cannot be replaced or regressed. Terminal publication and the owner-scoped expiry marker are written in the same Redis operation.

The real-Redis integration test forces competing completed/failed publications from separate clients after both have loaded processing state and verifies that the first terminal result remains authoritative.

### Exact request-byte conflict semantics — Pass

The idempotency digest is SHA-256 over the validated raw HTTP request bytes (`backend/internal/httpapi/optimization_controller.go:97,299-304`). An exact retry replays the stored `202` acknowledgement. Any byte-level body change—including field order or reordered exclusions—produces a different digest and returns the stable `409 idempotency_key_conflict` response without another publication. Focused HTTP tests cover exact replay, changed values, reordered exclusions, and syntactically different JSON.

## Original criteria

| Criterion | Result | Evidence |
|---|---|---|
| Entitlement and ownership | Pass | Authentication and active trial/paid entitlement are enforced before job or queue effects; the saved diet is loaded using the session-derived user ID; cross-user polling is indistinguishable from missing data. |
| `202` and poll URL | Pass | Initial and exact-replay responses preserve `202`, job ID, poll URL, and `Location`. |
| Exact idempotency replay/conflict | Pass | Durable first-writer claim, one logical job ID, byte-exact hashing, stable replay, and changed-body `409` are implemented and tested. |
| Monotonic states and one-hour TTL | Pass | Redis-side transition guards make queued/processing/terminal progression monotonic; production job/result TTL is one hour; owner-scoped expiry produces stable `410`, while other users receive `404`. |
| Safe errors | Pass | Infeasible and timeout outcomes expose only stable public codes/messages; dependency, ownership, and internal diagnostics are not leaked. |
| Queue outage `503` and no synchronous solving | Pass | Publication failure returns retryable `503`, retains recoverable claimed/job state, and retries publication later; the API has no solver path or synchronous fallback. |

## Verification performed

- `cd backend && go test ./...` with repository-local Go caches — passed
- `cd backend && go test -race ./...` with repository-local Go caches — passed
- `cd backend && go vet ./...` with repository-local Go caches — passed
- `python3 scripts/validate-traceability.py` — passed

The Task 204 implementation and its HTTP/Redis integration coverage satisfy the acceptance criteria.
