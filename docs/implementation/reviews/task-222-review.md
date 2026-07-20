# Task 222 Review Evidence

task_id: 222
component: "JobStatusTracker"
static_aspect: "DESIGN-004 optimization submission idempotency and concurrency"
input_status: "OPEN; task status intentionally preserved"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-17T15:30:00Z"
review_agent: "fresh independent repository review"
evidence_file: "docs/implementation/reviews/task-222-review.md"
baseline_ref: "HEAD a4e31367485b03269e90b5607f2057c9568bb5b1 plus current Task 222 preparation manifest and symbol-level scope"
baseline_confidence: "MEDIUM"
code_review_skill_invoked: true
relevant_language_guide: "Go, TypeScript, asynchronous concurrency, and security"
repair_context_required: false

## 1. Task Source

Task 222 is the Phase 07.01 optimization submission idempotency and concurrency task for DESIGN-004. The task row remains OPEN by explicit instruction. The current preparation document is the refreshed post-repair evidence source and was read in full.

The acceptance scope reviewed here is the task row and DESIGN-004: independent concurrent submissions, one durable job and logical queue publication for same-key submissions across controller instances, canonical semantic request hashing, exact acknowledgement replay, safe pending repair and revalidation, cancellation, shared 429 behavior, removal of obsolete helpers, and the current test and contract gates.

## 2. Pre-Review Gates

pre_review_gates_passed: true

- The full phase-orchestrator review template and checklist were read before review.
- The refreshed preparation document was read in full.
- The code-review-skill was invoked exactly once, with Go, TypeScript, asynchronous-concurrency, and security guidance applied.
- The prior Task 222 rejection was treated as stale evidence. Its stale-pending admission race was independently rechecked against current code and the new regression test.
- Production code and task status were not edited. The existing Task 221 review was not edited.
- The review uses current source fingerprints and command results recorded below.

## 3. Review Baseline and Change Surface

The review baseline is the current worktree at HEAD a4e31367485b03269e90b5607f2057c9568bb5b1, together with the refreshed Task 222 preparation manifest. The worktree contains cumulative changes from earlier tasks, so this is a symbol-level acceptance review rather than an attribution review of every dirty diff.

The principal change surface is Submit and repairOptimizationPublication in the optimization controller; canonical request parsing and hashing; persisted typed acknowledgement decoding and completion; optimization admission acquire and job-scoped release; idempotent queue publication; durable job state transitions and worker terminal cleanup; the idempotency repository update; the shared 429 helper; OpenAPI and generated API contracts; and focused HTTP and frontend tests.

The repaired stale-pending sequence was explicitly exercised: a repairer reads pending, another controller publishes and releases the original worker-owned slot, the repairer acquires a fresh slot, rereads the durable acknowledgement, observes the published record, and releases its fresh slot. The resulting state has no active admission slot and one logical queue entry.

## 4. Acceptance Criteria Checklist

| Criterion | Evidence | Result |
|---|---|---|
| Different-user submissions proceed independently and request cancellation is honored | Submit has no controller-wide mutex; focused HTTP tests cover independent users and cancellation, with repeated race runs | PASS |
| Same-key submissions across controller instances create one durable job | Durable claim conflict rereads the authoritative record; concurrent controller test asserts one job and two accepted responses | PASS |
| Same-key submissions create one logical queue publication | Queue marker and job save are idempotent; focused and repaired stale-race tests assert one logical queue entry | PASS |
| Semantically equivalent JSON replays | Typed parsing sorts set-like exclusions, normalizes UUID spelling and signed zero, and hashes the parsed request; focused tests cover ordering, whitespace, UUID case, numeric spelling, and signed zero | PASS |
| Real request changes return conflict | Canonical body hash mismatch returns the shared idempotency conflict error; changed tolerance is tested | PASS |
| Published acknowledgement replay is exact and has no current-state side effects | Published replay writes the persisted job, status, poll URL, and Location without entitlement, diet, queue, or admission reads; degradation and mutable-state test passes | PASS |
| Pending repair revalidates required ownership and input before publication | Repair checks current entitlement and exact owned diet, handles dependency failure, and tests invalidation and recovery | PASS |
| Pending repair closes the stale-pending admission race | Fresh durable reread occurs after newly acquired capacity; queued or processing transfers ownership, terminal state retains deferred release; two-controller regression test proves no stranded slot and one publication | PASS |
| Admission errors and Retry-After follow the shared 429 contract | Shared helper emits 429 and integer Retry-After rounded up with minimum one; boundary tests and OpenAPI lint pass | PASS |
| Accepted and replayed acknowledgements retain exact response data | One typed persisted acknowledgement is decoded strictly and projected to exact 202, Location, jobId, status, and pollUrl; malformed persisted data cannot silently fall back to Location | PASS |
| Obsolete helper behavior is absent | No raw-body replay hash, ignored replay parameter, controller serialization, silent Location fallback, or redundant UUID and retry clamp helper remains in the audited surface | PASS |
| Current verification gates pass | Focused tests, repeated race tests, full backend tests and race tests, vet, vulnerability reachability scan, frontend tests and build, generation tests, OpenAPI lint, task-list validation, traceability, and diff check pass | PASS |

## 5. Changed-Symbol Inventory

inventory_source_count: 25
audited_symbol_count: 25
inventory_complete: true

| # | Source and symbol surface | Scope reviewed |
|---|---|---|
| 1 | optimization_controller.go Submit | Admission, durable lookup, claim, save, enqueue, completion, and cancellation ownership |
| 2 | optimization_controller.go repairOptimizationPublication | Pending repair validation, fresh reread, ownership transfer, completion, and release |
| 3 | optimization_controller.go parseOptimizationSubmission | Strict request parsing and canonical parsed representation |
| 4 | optimization_controller.go optimizationRequestHash | Hash input and canonical serialization |
| 5 | optimization_controller.go lookupOptimizationIdempotency | User, method, route, key, and body-hash identity |
| 6 | optimization_controller.go persisted acknowledgement types and decoder | Strict durable state and exact response data |
| 7 | optimization_controller.go completeOptimizationPublication | Pending-to-published persistence |
| 8 | optimization_controller.go writeOptimizationAcknowledgement | Exact accepted response and Location projection |
| 9 | optimization_controller.go key, UUID, and finite-number validators | Input bounds and validation behavior |
| 10 | auth_errors.go retryableTooManyRequests | Shared 429 and Retry-After construction |
| 11 | optimization_admission.go Acquire | Slot ownership, replay, conflict, active, rate, and cancellation behavior |
| 12 | optimization_admission.go Release and existingDecision | Job-scoped compare-and-delete cleanup |
| 13 | job_queue.go Enqueue and enqueueScript | Logical queue publication idempotence |
| 14 | optimization_processor.go RedisOptimizationJobStore Save and Load | Durable job existence and authoritative state reads |
| 15 | optimization_processor.go transition and state-transition script | Monotonic job state behavior |
| 16 | optimization_processor.go terminal processing and releaseAdmission | Terminal cleanup and admission ownership |
| 17 | checkout_idempotency_repository.go UpdateResponse and tests | Durable acknowledgement update semantics and repository coverage |
| 18 | checkout_idempotency_update_response.sql | Update key scope and publication persistence |
| 19 | optimization_controller_test.go focused tests and fakes | Canonicalization, idempotency, queue failure, and concurrent claims |
| 20 | task222_optimization_submission_integration_test.go and ownership fake | HTTP acceptance, repair, cancellation, 429, and stale-pending regression |
| 21 | api/openapi.yaml optimization responses and TooManyRequests | Contract response and Retry-After schema |
| 22 | generate-api-types.py and its tests | Contract matrix and generated-type consistency |
| 23 | frontend/src/lib/api/generated.ts | Generated error-category contract |
| 24 | frontend/src/lib/api/optimization-client.ts | Safe 429 and error-category response mapping |
| 25 | frontend/src/lib/api/optimization-client.test.ts | Frontend contract and 429 behavior tests |

## 6. Function-Level Audit

| # | Symbol | Correctness and concurrency audit | Negative or regression audit | Result |
|---|---|---|---|---|
| 1 | Submit | Performs auth, parse, canonical hash, durable lookup, admission, claim, save, enqueue, and publication completion in the intended order; detached cleanup prevents cancellation leaks | No global serialization; claim conflicts release owned capacity before reread | PASS |
| 2 | repairOptimizationPublication | Revalidates dependencies, acquires same-key capacity without rate counting, rereads durable state after newly acquiring, and distinguishes worker-owned versus controller-owned terminal work | Fresh-slot path cannot strand capacity; invalid or changed diet is rejected before publication | PASS |
| 3 | parseOptimizationSubmission | Uses strict JSON decoding, rejects trailing data and invalid bounds, deduplicates by validation, and sorts exclusions | Rejects unknown fields, duplicate exclusions, invalid UUIDs, NaN, infinity, and over-limit input | PASS |
| 4 | optimizationRequestHash | Hashes the typed parsed request rather than raw JSON, including canonical exclusion order and normalized values | Reordered and equivalent JSON tests replay; real semantic changes conflict | PASS |
| 5 | lookupOptimizationIdempotency | Scopes lookup to user, method, route, key, and canonical body hash | Body mismatch is conflict and lookup does not ignore replay parameters | PASS |
| 6 | acknowledgement types and decoder | Persists one typed acknowledgement with state, job, status, and poll URL; strict decoder verifies accepted state and canonical poll URL | Unknown fields, malformed values, and wrong state fail instead of falling back to Location | PASS |
| 7 | completeOptimizationPublication | Updates the exact durable record from pending to published and preserves response fields | Update failure leaves controller ownership for safe cleanup and test coverage exercises it | PASS |
| 8 | writeOptimizationAcknowledgement | Projects exact status 202, Location, jobId, status, and pollUrl from the durable acknowledgement | No silent generated Location or response reconstruction from mutable current state | PASS |
| 9 | validators | Clones and bounds idempotency keys, validates nonnil UUIDs, and accepts only finite tolerance values | Boundary and malformed input paths are covered | PASS |
| 10 | retryableTooManyRequests | Uses the shared error envelope and rounds Retry-After up with a minimum of one second | Negative, zero, fractional, subsecond, and whole-second cases are tested | PASS |
| 11 | Acquire | Uses Redis SetNX per user and admission key, distinguishes replay from conflicts and active work, and releases rate-limited reservations | CountRate false repair does not consume rate quota; cancellation is tested | PASS |
| 12 | Release and existingDecision | Watches the reservation and deletes only when the stored job ID matches, with bounded transaction retry | A stale owner cannot delete another job's slot; no broad delete is used | PASS |
| 13 | Enqueue and enqueueScript | Validates and atomically inserts one stream entry with a marker | Existing marker returns the same logical entry and prevents duplicate publication | PASS |
| 14 | Job store Save and Load | Save and Load provide durable job state and authoritative reread for repair ownership decisions | Terminal records are not overwritten by duplicate save paths | PASS |
| 15 | transition and script | State transitions are guarded and monotonic | Terminal state remains terminal under repeated or late transitions | PASS |
| 16 | terminal processing and releaseAdmission | Worker terminal paths release the job-scoped admission reservation | Release is tied to the same job and integrates with stale-pending regression setup | PASS |
| 17 | repository update and tests | Update is key-scoped and requires exactly one affected row | Wrong body or missing record cannot be silently treated as published | PASS |
| 18 | update SQL | Predicate includes user, method, route, key, and body hash | No cross-user or cross-request acknowledgement mutation is possible through this statement | PASS |
| 19 | focused controller tests and fakes | Cover canonicalization, queue outage, durable claim conflict, and concurrent controller behavior | Repeated normal and race executions pass | PASS |
| 20 | Task 222 integration tests and ownership fake | Exercise HTTP-level replay, repair, cancellation, 429, exact response, and stale-pending ownership sequence | Regression asserts no active slot, same job, and one logical queue entry after interleaving | PASS |
| 21 | OpenAPI responses | Optimization POST references reusable TooManyRequests with required integer Retry-After minimum one | Redocly lint passes with only an unrelated pre-existing OAuth callback warning | PASS |
| 22 | generator and tests | Error-category and response matrix includes rate_limit and stays synchronized | Generator check and seven generator tests pass | PASS |
| 23 | generated TypeScript | Contains the current rate_limit error category and generated response types | Generated output is current according to generator check | PASS |
| 24 | optimization client | Preserves error category, code, request ID, and retryability in safe mapping | 429 handling is covered and does not discard contract metadata | PASS |
| 25 | frontend tests | Verifies optimization client error and 429 projection | Full frontend suite and build pass | PASS |

## 7. Findings

blocking_findings: 0
important_findings: 0
optional_findings: 0

| Severity | Finding | Reproduction | Impact | Disposition |
|---|---|---|---|---|
| None | No unresolved correctness, security, behavior, or required-coverage finding identified | Not applicable | No acceptance criterion remains failed or unaudited | None |

## 8. Commands Run

The following scoped commands were run against the current worktree and passed unless noted:

- Focused HTTP idempotency, repair, cancellation, 429, and stale-race tests with count 20.
- The same focused HTTP family with race detection and count 25.
- Full backend test suite with count 1.
- Full backend race suite with count 1.
- Go vet for all backend packages.
- govulncheck v1.3.0 for all backend packages; no vulnerabilities were reachable from called symbols. The tool reported module advisories outside the reachable call graph.
- HTTP package coverage with coverprofile at /tmp/task-222-httpapi-repaired.coverage.out; package line coverage was 89.6 percent.
- Frontend optimization-client focused tests; 8 tests and 22 expectations passed.
- Full frontend tests; 365 tests and 1603 expectations passed.
- Frontend production build.
- API type generation check and scripts/test_generate_api_types.py; 7 tests passed.
- Redocly lint for api/openapi.yaml; valid with one unrelated pre-existing OAuth callback warning.
- scripts/validate-task-list.py.
- scripts/validate-traceability.py.
- git diff --check.

scripts/check.py was not used as the review gate because this intentionally cumulative worktree requires external Docker Compose and a Chromium-compatible browser for unrelated aggregate checks. Its scoped backend, frontend, contract, traceability, and static checks were run directly above.

## 9. Files Inspected and Staleness Fingerprints

all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true

| File | SHA-256 |
|---|---|
| backend/internal/httpapi/optimization_controller.go | 551fa4826675bcb926c64e1673bdfdbc9cb3be96881451ffc1c31c6a027501d3 |
| backend/internal/httpapi/auth_errors.go | ef2c5ada2b3d8c916d9ec624b867b773bc2c53a625f379bc500286389d2d775f |
| backend/internal/httpapi/optimization_controller_test.go | 3aae2bb23b667f5a18bd221a9d7959bd80def77b3e7e0582739766d651e6e214 |
| backend/internal/httpapi/task222_optimization_submission_integration_test.go | 37f390e9f7fd006a492cb9a43c593307134a1e510bab93fb02d3affe52255e55 |
| backend/internal/repository/checkout_idempotency_repository.go | 520ebeb234ec337c9714333731dc17b4ba9a3166d2e72f144fe481304be31ef1 |
| backend/internal/repository/checkout_idempotency_repository_test.go | a45b608b26111df8ab5ea3f7205b4d0625146be63f7b38dbab1c6657a72c9317 |
| backend/internal/repository/sql/checkout_idempotency_update_response.sql | 3b67b43570fd6d5978f447d722634c48153dbb7c59fdf6fdf3b2e0a92075707e |
| backend/internal/worker/optimization_admission.go | 6c27535a913f83b2d103093d227aaec8134eadaa625389469ad491e79252e756 |
| backend/internal/worker/optimization_processor.go | 50ea0a2165cb6ec19f4d4fcb7f83d1ce51ff1f65f569dcf788d652b2d8933427 |
| backend/internal/queue/job_queue.go | dd7779df174114b3dc6a9104ebaa3428101bd942a7c5f3559e7f5a18dc932497 |
| api/openapi.yaml | b6fe1f1322016ad96fa8eba9c09eb83211f698ab940ff36772e75468c0861585 |
| scripts/generate-api-types.py | 1d3df961971558688facf503afdf1d014e802ba0bca9f12e7d786f6ab7752954 |
| scripts/test_generate_api_types.py | b21cee3080b7e93f8827b690e88076dcc91d18c219714665b1add1df720f1ff0 |
| frontend/src/lib/api/generated.ts | 166722ae537251aeb09ec58f9b53557d929bc4f75c258dc770dc93090f0882ae |
| frontend/src/lib/api/optimization-client.ts | 71ca43db9783f42c59d6523290b2e60be24f25c51ce1b1d5df73955755c330f7 |
| frontend/src/lib/api/optimization-client.test.ts | fab6abd590530acaf2d735ffc8c35f787d880f10bc6c1d887e70f83332aad744 |
| docs/design/DESIGN-004.md | 688e9c18e398b3c83dd50f4066864cd48a9897003a7f78d030b6a531de6d81bc |
| docs/implementation/preparation/task-222-preparation.md | 199b902b6436355af973d341bab914ef3ad6d4575c25028c53b016cf1690a548 |
| docs/implementation/02_TASK_LIST.md | ff97c9908298a6215b3211cce5ebb8931569940d2e534b3387b1c8b60374f6d4 |
| docs/implementation/reviews/task-221-review.md | 729f275360a645a92ba1cfcb64e0e8f55e753a643b458c5ee5fbdd7224aec2e0 |

stale_prior_evidence:

- The old Task 222 review described the pre-repair stale-pending race and was not reused as current evidence.
- The refreshed preparation hashes match the repaired source and test files reviewed here.
- The Task 221 review and task-list fingerprints were captured before writing and must remain unchanged.

## 10. Coverage and Exceptions

coverage_required: true
coverage_exception_allowed: true
coverage_report_path: "/tmp/task-222-httpapi-repaired.coverage.out"
observed_line_coverage: "89.6 percent for internal/httpapi statements; Submit 71.4 percent; repairOptimizationPublication 72.9 percent; acknowledgement decoder and projection 100 percent"
coverage_passed: true

The task-specific integration tests cover the repaired interleaving, including fresh admission acquisition, durable published reread, worker terminal release, deferred fresh-slot release, one logical queue entry, and exact acknowledgement identity. The lower function-level percentages are not an acceptance failure because the relevant branches are covered by the focused and race-tested integration suite and the full backend suite passes.

## 11. Negative and Regression Checks

- No controller-wide submission mutex or other global submission serialization remains.
- No raw request-body hash is used for replay identity; hashing uses the canonical parsed request.
- Set-like exclusions are sorted and validated before hashing.
- No ignored replay parameter, silent Location fallback, or redundant UUID or retry clamp helper remains in the audited path.
- Published replay does not re-read current entitlement, diet, queue, or admission state.
- Pending repair revalidates entitlement and exact diet ownership before save or enqueue.
- Admission release is job-scoped and compare-and-delete protected.
- The repaired stale-pending test demonstrates that a newly acquired slot is released when a concurrent publication is found, while queued or processing work transfers ownership and terminal work does not.
- Cancellation tests cover initial submission and pending repair; cleanup uses a cancellation-detached request context.
- Full backend race testing, frontend testing and build, API generation, OpenAPI lint, task-list validation, traceability, vet, vulnerability reachability scanning, and diff checks pass.

## 12. Decision

decision: "PASSED"

The repaired durable pending reread and explicit queued or processing versus terminal ownership check close the stale-pending admission leak while preserving one logical publication, exact replay and repair semantics, cancellation behavior, and the shared 429 contract. Current acceptance, symbol, race, static, security, frontend, and OpenAPI evidence passes.

failed_criteria:

- ""

failed_or_unaudited_symbols:

- ""

recommended_next_action: "None for this review; the owner may separately transition task status when authorized."

## 13. Repair Context

Repair context was reviewed because this is a fresh review after the stale-pending race repair. The prior rejection is superseded for this review by current source, refreshed preparation evidence, and the new two-controller regression test. No additional repair is required. Task 222 remains OPEN because this review was explicitly instructed not to change task status.
