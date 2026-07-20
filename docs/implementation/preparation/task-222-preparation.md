# Task 222 Preparation — Optimization Submission Idempotency and Concurrency

## Scope and attribution

- Task: `222`, Phase 07.01 Optimization Submission Idempotency and Concurrency.
- Design source: `docs/design/DESIGN-004.md`, static aspect `JobStatusTracker`.
- Dependencies preserved: task `220` remains `PASSED` and task `221` remains `PREPARED`.
- Task status intentionally remains `OPEN` per the implementation request; no task-list cell was changed.
- Preparation date: 2026-07-17 (Europe/Warsaw).
- The worktree already contained uncommitted Tasks 220/221 work and a partially implemented Task 222 stream when this continuation began. Existing changes were preserved. Attribution therefore uses symbol-level inspection and the Task 222 review actions in `docs/implementation/04_OPEN.md`, not a whole-worktree diff against `HEAD`.
- Later Task 223 observability/cleanup typing and every Task 224+ queue change remain out of scope.

## Blocking-review repair refresh

The validated independent review in `docs/implementation/reviews/task-222-review.md` rejected the earlier preparation because controller B could retain a stale `pending` acknowledgement while controller A published the same job and its worker released the admission slot. B could then acquire a fresh slot, observe only idempotent job/queue state, and incorrectly hand that slot to a worker that had already finished.

The repaired pending path now re-reads the durable acknowledgement immediately after a repairer newly acquires capacity. A concurrently persisted `published` acknowledgement is replayed and deferred cleanup releases the fresh reservation. If the acknowledgement is still pending, the repairer saves/enqueues idempotently and loads the authoritative job before ownership transfer. Only `queued` or `processing` accepts the handoff; a preserved terminal job remains controller-owned and is released. A later terminal transition remains safe because the worker's job-scoped compare-and-delete release targets the same reservation.

`TestOptimizationHTTPConcurrentPublishedRepairDoesNotStrandAdmission` uses two controller instances and explicit barriers. Controller B pauses after reading `pending`; controller A publishes; the fixture marks the worker terminal and releases its slot; then B resumes. Before the repair this test failed with `stale pending repair stranded a newly acquired admission slot`. After the repair it passes repeatedly and proves the slot is empty, both controllers return the same durable job acknowledgement, and the idempotent queue contains one logical publication.

## Implemented contract

### Canonical request identity

`OptimizationController.Submit` parses and validates the request before hashing it. The parsed diet UUID is canonical, tolerance is normalized to one decimal place including signed zero, and `excludedMealIds` is a duplicate-free set sorted by canonical UUID. JSON member order, whitespace, numeric spelling, UUID letter case, and exclusion order therefore replay one request; changes to diet, tolerance, or exclusion membership conflict with `409`.

The idempotency key is user-scoped in durable storage and cloned from the Fiber header buffer before use. No controller-wide mutex remains.

### Durable acknowledgement and publication state

`optimizationPersistedAcknowledgement` is the sole strict persisted shape:

- `jobId`: non-nil UUID;
- `status`: exactly `queued`;
- `pollUrl`: exactly the canonical poll URL for `jobId`;
- `publicationState`: exactly `pending` or `published`.

The decoder rejects unknown fields, trailing JSON, invalid status, mismatched/missing poll URLs, nil IDs, unsupported status codes, and unsupported publication states. Public acknowledgement projection returns only `jobId`, `status`, and `pollUrl`, with exact `202` and `Location`; internal publication state is not exposed.

Published exact replay occurs before entitlement, diet, admission, job-store, queue, or result-TTL access. A published acknowledgement therefore remains immutable after diet mutation/deletion, entitlement change, dependency outage, or result expiry and performs no publication/rate side effect.

### Concurrent claim and repair

The PostgreSQL user/method/route/key uniqueness boundary is the cross-controller authority. A claim winner stores one pending acknowledgement. A concurrent loser loads that exact typed acknowledgement and either replays a published result or repairs the pending claim using the original job ID. Redis job save and queue enqueue are idempotent by job ID, so concurrent controllers converge on one job and one queue entry.

Only pending claims enter repair. Repair rechecks current entitlement, reloads the exact diet under the authenticated owner, honors cancellation, and reacquires the same user slot with `CountRate=false` before save/enqueue. Denied, missing, foreign, cancelled, or dependency-failed repair publishes nothing. A newly acquired repair slot triggers a durable acknowledgement re-read so a concurrent published winner is replayed and the fresh slot is released.

Successful enqueue transfers active-slot ownership only when the authoritative job remains queued or processing. A terminal job behind an existing queue marker cannot own a new delivery, so deferred controller cleanup releases that slot. If the job transitions after the check, the worker releases the same job-scoped reservation. If acknowledgement update fails for active work, the pending durable claim remains retryable through idempotent publication repair without admitting overlap.

### Shared admission error contract

Active-job and hourly-limit rejection use the shared retryable 429 helper. `Retry-After` uses `ceil` and Go's built-in `max`, producing a positive base-10 whole second for negative, zero, sub-second, fractional, and whole-second inputs. OpenAPI uses the shared `TooManyRequests` response with a required integer header whose minimum is one.

`rate_limit` is now included consistently in OpenAPI `AppError.category`, generated `ErrorCategory`, the optimization client's runtime guard, and frontend error tests. The client preserves category, code, request ID, and retryability instead of falling back by status.

## Changed Task 222 surfaces

| Path | Symbols or contract surface |
| --- | --- |
| `backend/internal/httpapi/optimization_controller.go` | `OptimizationIdempotencyRepository`; `OptimizationController.Submit`; `repairOptimizationPublication`, including post-acquisition durable re-read and queued/processing worker-delivery ownership check; `optimizationSubmissionRequest`; `parseOptimizationSubmission`; `optimizationRequestHash`; `lookupOptimizationIdempotency`; `optimizationPublicationState`; `optimizationPersistedAcknowledgement`; `decodeOptimizationAcknowledgement`; `completeOptimizationPublication`; `writeOptimizationAcknowledgement`; direct `validateUUIDValue` route validation; removal of controller mutex, raw-body hashing, ignored replay parameter, map/string fallback, forwarding UUID helpers, manual retry helper, and one-use acknowledgement constructor |
| `backend/internal/httpapi/auth_errors.go` | shared `retryableTooManyRequests`; `AccountLockedError` reuse |
| `backend/internal/httpapi/optimization_controller_test.go` | canonical replay, changed-body conflict, cross-controller concurrency fakes, terminal-preserving job-store behavior, acknowledgement update failure injection, cancellation-aware dependency fakes |
| `backend/internal/httpapi/task222_optimization_submission_integration_test.go` | barriered two-controller stale-pending/publication/terminal-release regression; stateful admission ownership; different-user progress; submit/repair cancellation; exact replay under changed/failed current state; repair revalidation; queue-to-worker admission handoff; 429/OpenAPI/retry boundaries; malformed acknowledgement rejection |
| `backend/internal/repository/checkout_idempotency_repository.go` | embedded update query; `UpdateCheckoutIdempotencyResponse` body-matched durable transition |
| `backend/internal/repository/checkout_idempotency_repository_test.go` | successful update and missing/mismatched target tests |
| `backend/internal/repository/sql/checkout_idempotency_update_response.sql` | scoped, parameterized acknowledgement response update |
| `api/openapi.yaml` | optimization `429` uses `TooManyRequests`; required positive `Retry-After`; `rate_limit` error category |
| `scripts/generate-api-types.py` | generated `rate_limit` category and audited optimization response matrix |
| `scripts/test_generate_api_types.py` | response-reference mutation test and `rate_limit` generation drift test |
| `frontend/src/lib/api/generated.ts` | generated `ErrorCategory` includes `rate_limit` |
| `frontend/src/lib/api/optimization-client.ts` | runtime `isErrorCategory` accepts the shared category |
| `frontend/src/lib/api/optimization-client.test.ts` | 429 category/code/request/retryability preservation |
| `docs/design/DESIGN-004.md` | canonical identity, typed acknowledgement, replay/repair split, post-acquisition durable re-read, cross-controller authority, and explicit queued/processing worker ownership handoff |
| `docs/implementation/preparation/task-222-preparation.md` | this evidence |

No Task 220/221 solver-generation or terminal-publication behavior was reverted or widened.

## Verification-criteria mapping

| Task 222 criterion | Evidence | Result |
| --- | --- | --- |
| Different users proceed independently; cancellation is honored | blocked-repository two-user HTTP test; initial and repair timeout tests | PASS |
| Same key across controllers creates one job/publication | two Fiber apps race through one durable claim fake; one job, one idempotency row, one idempotent queue entry | PASS |
| Semantic JSON replay and real-change conflict | property order, whitespace/numeric decoding, UUID case, reordered exclusions, and changed tolerance fixtures | PASS |
| Published replay ignores current mutable/degraded state | exact three-field acknowledgement replay after expiry marker, diet change and repository failure, entitlement failure, queue failure, and admission failure; call counts unchanged | PASS |
| Repair revalidates and never publishes invalid work or strands capacity | entitlement denial, ownership miss, cancellation, queue outage/recovery, post-repair replay, and barriered stale-pending/publication/terminal-release fixtures | PASS |
| 429 and `Retry-After` match shared contract | backend boundary table, OpenAPI response component, Redocly, generator drift, frontend runtime test | PASS |
| Exact acknowledgement retained | strict persisted decoder and accepted/replayed assertions for `202`, `Location`, job ID, queued status, poll URL, and exactly three public data fields | PASS |
| Obsolete helpers absent | repository search confirms no controller mutex, ignored replay parameter, `stringValue`, `validUUIDString`, forwarding optimization UUID validator, manual retry clamp, or one-use acknowledgement constructor | PASS |
| Concurrency/security | barriered two-controller stale-read regression, one logical queue publication and no active slot after terminal release, parameterized SQL, authenticated user scope, no raw identifier/key telemetry labels, race suite, vet, and vulnerability scan | PASS |

## Commands and results

| Working directory | Command | Result |
| --- | --- | --- |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi ./internal/repository ./internal/worker -count=1` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -run '^TestOptimizationHTTPConcurrentPublishedRepairDoesNotStrandAdmission$' -count=1` before production repair | EXPECTED FAIL; reproduced stranded admission slot |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -run '^TestOptimizationHTTPConcurrentPublishedRepairDoesNotStrandAdmission$' -count=20` after repair | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/httpapi -run 'TestOptimizationHTTP(ConcurrentPublishedRepairDoesNotStrandAdmission\|ConcurrentControllersClaimOneJob\|DifferentUsersProceedIndependentlyAndSameKeyIsUserScoped\|IdempotencyAndQueueFailure\|UnpublishedRepairRevalidatesAndPublishesOnce\|PublishedAcknowledgementReplayHasNoCurrentStateSideEffects\|SubmissionHonorsRequestCancellation\|Admission429)' -count=25` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -coverprofile=/tmp/task-222-httpapi-repair.coverage.out -count=1` | PASS; 89.6% of statements |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -count=1` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS; no called vulnerabilities |
| `frontend/` | `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/optimization-client.test.ts` | PASS; 8 tests, 22 expectations |
| `frontend/` | `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | PASS; 365 tests, 1603 expectations |
| `frontend/` | `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | PASS |
| repository root | `python3 scripts/generate-api-types.py --check` | PASS |
| repository root | `python3 -m unittest scripts/test_generate_api_types.py` | PASS; 7 tests |
| repository root | `npx --no-install redocly lint api/openapi.yaml` | PASS with the pre-existing OAuth callback 302/no-2xx warning |
| repository root | `python3 scripts/validate-task-list.py` | PASS; 237 sequential tasks with ordered dependencies |
| repository root | `python3 scripts/validate-traceability.py` | PASS |
| repository root | `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-222-review.md` | PASS; validated review remains the repair source |
| repository root | `git diff --check` | PASS |

## Refreshed repair hashes

| File | SHA-256 after repair |
| --- | --- |
| `backend/internal/httpapi/optimization_controller.go` | `551fa4826675bcb926c64e1673bdfdbc9cb3be96881451ffc1c31c6a027501d3` |
| `backend/internal/httpapi/optimization_controller_test.go` | `3aae2bb23b667f5a18bd221a9d7959bd80def77b3e7e0582739766d651e6e214` |
| `backend/internal/httpapi/task222_optimization_submission_integration_test.go` | `37f390e9f7fd006a492cb9a43c593307134a1e510bab93fb02d3affe52255e55` |
| `docs/design/DESIGN-004.md` | `688e9c18e398b3c83dd50f4066864cd48a9897003a7f78d030b6a531de6d81bc` |

## Security and residual boundaries

- Trust boundary: authenticated JSON, idempotency header, and persisted acknowledgement JSON are untrusted until validated. SQL remains parameterized and all durable lookup/update scopes include authenticated user, method, route, and key.
- Idempotency keys are not returned, logged, or placed in telemetry labels. Redis admission stores only SHA-256 hashes of user/key identity plus the canonical body hash.
- Queue publication and PostgreSQL acknowledgement confirmation cannot be one cross-store transaction. The deliberate pending/published state plus idempotent save/enqueue provides safe retry without duplicate logical work.
- Failed admission cleanup observability and a local cleanup deadline remain Task 223 ownership. Queue marker lifecycle, reservation, and finalization remain Tasks 224/225 ownership.
- The repository-wide phase coverage target remains a later Phase 07.01 gate; Task 222 adds focused branch and race coverage but does not claim aggregate 100% coverage.

## Preparation decision

Task 222 implementation and evidence satisfy its row criteria. Per explicit instruction, `docs/implementation/02_TASK_LIST.md` was not edited and Task 222 remains `OPEN` for the owner's later status transition/review workflow.
