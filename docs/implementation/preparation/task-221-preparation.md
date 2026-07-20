# Task 221 Preparation — Optimization Publication Vocabulary and Projection

## Repair refresh after owner review rejection

This section supersedes the original preparation's current-state command results and final hashes. The original sections below remain as historical baseline evidence. Repair completed at `2026-07-17T12:32:56Z` against the cumulative worktree after reading `docs/implementation/reviews/task-221-review.md`. No task status was changed: rows 220, 221, and 222 remain `PREPARED`, `PREPARED`, and `OPEN` respectively, and the task-list hash remains `e57ae220a9a603aeba610f3e58992701b63ef5c42d2406bcd3bbac16ff79a1eb`.

The worktree contains overlapping Task 222 changes in `backend/internal/httpapi/optimization_controller.go` and its test. This repair changed only Task 221 result-validation/polling symbols in those shared files; submission idempotency, acknowledgement, admission, hashing, concurrency, and publication-repair behavior remain Task 222 ownership.

### Review findings repaired

| Review finding | Repair evidence | Result |
| --- | --- | --- |
| F-1 blocking: finite but out-of-range or off-grid `similarityScore` could be persisted, decoded, and projected | `optimization.ValidateDietAlternative` is the shared authoritative predicate. It validates one-to-100 canonical meal projections, finite bounded quantities/macros/calories, and a finite `0..1` score on the four-decimal grid. `RedisOptimizationJobStore.PublishCompleted`, `PublishFailed`, and decoded `validateOptimizationJob` apply it through `validateOptimizationAlternatives`. `OptimizationController.GetJob` applies the same predicate through `validateOptimizationJobAlternatives` before calling `optimizationJobData`. | REPAIRED |
| F-2 important: typed-nil errors could panic `FailureCodeOf` or `safeOptimizationFailure` | `FailureCodeOf` checks the `*OptimizationFailure` target for nil. `safeOptimizationFailure` nil-checks `*OptimizationFailure`, `*SolverError`, and `*repository.Error` targets and maps malformed typed-nil solver/dependency errors to the generic `worker_crash` classification. `OptimizationProcessor.handleProcessingError` independently nil-checks its repository fallback and leaves unknown typed-nil errors pending for retry; telemetry returns bounded `failed`. | REPAIRED |

The shared predicate also rejects malformed meal IDs, noncanonical output units, nonpositive or over-limit quantities, nonsequential positions, nonfinite/negative/over-limit macro projections, and nonfinite scores. These checks preserve the output invariants already established by `SolutionValidator`; they do not alter Task 220 generation, canonicalization, deduplication, or attempt semantics.

### Exact repaired symbols and focused tests

| Path | Symbols or tests |
| --- | --- |
| `backend/internal/optimization/validator.go` | `FailureCodeOf`; `ValidateDietAlternative`; `boundedProjectionNumber`; `safeOptimizationFailure` |
| `backend/internal/optimization/validator_test.go` | `TestValidateDietAlternativeRejectsMalformedResultShape`; `TestValidateDietAlternativeRejectsInvalidSimilarityScores`; `TestOptimizationFailureClassificationHandlesTypedNilErrors` (uses a valid generation setup so the solve seam actually returns a typed-nil `*SolverError`) |
| `backend/internal/worker/optimization_processor.go` | `RedisOptimizationJobStore.PublishCompleted`; `RedisOptimizationJobStore.PublishFailed`; `OptimizationProcessor.handleProcessingError`; `validateOptimizationJob`; `validateOptimizationAlternatives` |
| `backend/internal/worker/optimization_processor_deadline_test.go` | `TestOptimizationProcessorTreatsTypedNilFailureAsRetryableUnknown` for typed-nil optimization and repository errors, no terminal publication, and bounded telemetry |
| `backend/internal/worker/task221_publication_test.go` | `TestTask221RedisStoreValidatesSimilarityScoreAtPublicationAndDecode` for `-0.0001`, `1.0001`, `0.12345`, and valid `0.1234` Redis round-trip |
| `backend/internal/httpapi/optimization_controller.go` | `OptimizationController.GetJob`; `validateOptimizationJobAlternatives`; guarded call to `optimizationJobData` |
| `backend/internal/httpapi/optimization_controller_test.go` | `TestOptimizationHTTPRejectsInvalidPersistedSimilarityBeforeProjection` for the three invalid fixtures and valid rounded projection |

### Repair verification

| Working directory | Command | Result |
| --- | --- | --- |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization ./internal/worker ./internal/httpapi` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/optimization ./internal/worker ./internal/httpapi -run 'TestValidateDietAlternative\|TestOptimizationFailureClassificationHandlesTypedNilErrors\|TestTask221RedisStoreValidatesSimilarityScoreAtPublicationAndDecode\|TestOptimizationProcessorTreatsTypedNilFailureAsRetryableUnknown\|TestOptimizationHTTPRejectsInvalidPersistedSimilarityBeforeProjection' -count=1` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./...` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS |
| `frontend/` | `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/optimization-client.test.ts src/lib/stores/optimization.test.ts src/lib/components/OptimizationWorkflow.test.ts` | PASS; 20 tests / 90 expectations |
| repository root | `python3 scripts/validate-task-list.py` | PASS; 237 sequential tasks with ordered dependencies |
| repository root | `python3 scripts/validate-traceability.py` | PASS |
| repository root | `python3 scripts/generate-api-types.py --check` | PASS |
| repository root | `python3 -m unittest scripts.test_generate_api_types.OperationResponseDriftTest.test_optimization_terminal_vocabulary_and_similarity_projection_match_generated_contract` | PASS |
| repository root | `npx --no-install redocly lint api/openapi.yaml` | PASS with the pre-existing OAuth callback 302/no-2xx warning |
| repository root | `git diff --check` | PASS |

Two cumulative checks remain non-Task-221 failures and are not concealed by the scoped evidence:

- `python3 -m unittest scripts/test_generate_api_types.py` runs seven tests; six pass and `test_deliberate_response_mismatch_is_rejected` fails because its pre-Task-222 mutation removes an `Error` response while the current overlapping Task 222 OpenAPI uses `TooManyRequests`. The Task 221 vocabulary/similarity drift method passes independently.
- The complete three-package race command reaches the existing worker bootstrap integration test and can fail its three-second shutdown/publication timing (`TestRunPublishesAlternativeBeforeAcknowledgingQueuedJob`). The exact Task 221 race command above passes, and the full non-race backend suite passes. No worker bootstrap timing code was changed under Task 221.
- The first `python3 scripts/check.py` attempt met a concurrent migration run on the shared PostgreSQL instance; after that process finished, `python3 scripts/verify-local-stack.py --keep-services` passed migrations down/up, API and worker startup, health, and readiness checks. A contention-free aggregate rerun passed traceability, task-list, Phase 07 Go Doc, OpenAPI lint, optimization-capacity tests, `go vet`, `govulncheck` (no called vulnerabilities), local-stack/UAT/browser checks, full backend tests, and the aggregate race run. It exited at the pre-existing phase coverage gate: total backend coverage was `88.4%`, with `dailydiet 80.1%`, `optimization 85.3%`, and `worker 64.6%` below 100% without measured exceptions. No migration, coverage policy, or unrelated package was changed by Task 221.

### Current repaired-file hashes

| Path | SHA-256 |
| --- | --- |
| `backend/internal/optimization/validator.go` | `4119ce67e0353c725d9e0feca9f13379966f1d89eaee5477bee706929ae6b09a` |
| `backend/internal/optimization/validator_test.go` | `950349bf26c0a28df9b2f3037243b26d0ccd0c0203f69fb5d76b612ad1742f73` |
| `backend/internal/worker/optimization_processor.go` | `50ea0a2165cb6ec19f4d4fcb7f83d1ce51ff1f65f569dcf788d652b2d8933427` |
| `backend/internal/worker/optimization_processor_deadline_test.go` | `fc79e1cc9329eac1e5f773ef9ba3c6acf1a226e4e8876d843bb09e9af0de5b37` |
| `backend/internal/worker/task221_publication_test.go` | `8e7270f161af93763c3e6023b48e8c73304b841aaca2548305593594e31f97f3` |
| `backend/internal/httpapi/optimization_controller.go` | `422b0232a203d05071e33c050fafe40a681120ee4544011d6c2c100405208664` |
| `backend/internal/httpapi/optimization_controller_test.go` | `4425e033ea214aee6e35bb1066cd6296243a8ed7d601e961818a305472fb811b` |

Repair conclusion: both owner-review findings are closed with one domain predicate reused at persistence, decode, and HTTP projection plus nil-safe error classification through domain, solve, worker retry, and telemetry seams. Task 221 remains `PREPARED` for independent re-review; this repair does not change any task status.

## Scope and baseline

- Task: `221`, Phase 07.01 Optimization Publication Vocabulary and Projection.
- Selected row status at start: `OPEN`; dependency `220`: `PREPARED`.
- Fixed repository reference: `a4e31367485b03269e90b5607f2057c9568bb5b1`.
- Design source: `docs/design/DESIGN-004.md`, static aspects `SolutionValidator`, `JobQueueManager`, and `JobStatusTracker`.
- Preparation date: 2026-07-17 (Europe/Warsaw).
- Scope boundary: task 222 and every later task remain unimplemented. No submission idempotency, controller serialization, request hashing, admission, or queue-publication repair behavior was changed.
- Attribution confidence: high. The worktree was already dirty with concurrent Phase 07.01 work, so task attribution uses the exact pre-task hashes below rather than the commit diff. Existing modifications were preserved. `scripts/test_generate_api_types.py` was already untracked and was extended surgically. `backend/internal/app/task206_backend_integration_test.go` required one compatibility call-site edit after the failure-code representation became non-string-constructible.

### Pre-task shared-file hashes

| Path | SHA-256 before Task 221 |
| --- | --- |
| `docs/design/DESIGN-004.md` | `3a1e0da78a644c142552c1ea404ebb5441d89f35f6e1a65d42fc493632c07d70` |
| `docs/implementation/02_TASK_LIST.md` | `62fe776fdbb2e0bb16792ed0ed0435ae065980b6da6d3468408f25d4eb483211` |
| `api/openapi.yaml` | `6368ee9c1321104d0e645ed8a3e6b73f8f14c1a161835a5f388a0e6e2fa3da4a` |
| `scripts/generate-api-types.py` | `b6b241afb9c13c4206e30eb1e12ae4adcd32d6734cc065b33ece5d47ed4f4e87` |
| `scripts/test_generate_api_types.py` | `475c5f9fe28e4bba538ced96faa40b07223708cd37ae2a227f7049ff900ad4d5` |
| `backend/internal/optimization/validator.go` | `dd771467a18fdeb6502e5452b7aee8bcbbc39284a8d5c47df626a1e8ecdde827` |
| `backend/internal/optimization/validator_test.go` | `79b49bf7f11eac0f1526748681213b1cb95665d0b48976103825879ca8db3c96` |
| `backend/internal/worker/optimization_processor.go` | `4db1a08c09598077cea9ffdbde347df9dab01a3f76cd61617ac0330a20b202b0` |
| `backend/internal/worker/optimization_processor_deadline_test.go` | `e7660af899156cec6e07b52cee3cc1b2c43a3b31a465e4d735406082b848f180` |
| `backend/internal/httpapi/optimization_controller.go` | `ef7c3eb939e40ce3f75ef9c8ec912ba75bb2acd871520c2b876b86e2ea717855` |
| `backend/internal/httpapi/optimization_controller_test.go` | `45566606bd768301c9002a26d34fdfb72c1e3cdf86f3fcc43370803c1f3e5a25` |
| `frontend/src/lib/api/generated.ts` | `361ce14d3cde8ae90afe0bc074ffb3e301c751a8a9603fc7a300e7d6b49cf20b` |
| `frontend/src/lib/api/optimization-client.ts` | `145e51d097243affd72513c7de1174e7e49324aa2b2a4b460db86e6e7de91635` |
| `frontend/src/lib/api/optimization-client.test.ts` | `8935ee1e3e571d91648bd582da96fc5fdbfc0cb02f0f426412a004dcace1e5f8` |
| `frontend/src/lib/stores/optimization.ts` | `198de27d00168dd642647c2f60963688fb1fa7151ec8ff5c7814c0fb769aa092` |
| `frontend/src/lib/components/OptimizationWorkflow.test.ts` | `309c4174081bd51d1df6b29b2ffedabacce71752dd9c215f09c5b997139bf0d4` |

The app integration test was not in the initial candidate list. Its immediate pre-edit hash, captured after the full backend compiler identified the required call-site migration, was `7ba576d1700d7b8f19cc1b99ca57ca69728aedcda0ee091dfcb3c20d3fa58042`.

## Implemented contract

### Bounded terminal vocabulary and persistence

The complete persisted terminal job-failure enum is:

1. `failed_validation`
2. `solver_timeout`
3. `solver_infeasible`
4. `worker_crash`

`OptimizationFailureCode` is now a value object with an unexported string. Callers cannot manufacture a nonzero arbitrary code. `ParseOptimizationFailureCode`, `MarshalJSON`, and `UnmarshalJSON` accept only the four retained legacy string values; empty, unknown, non-string, `queue_unavailable`, and `result_expired` values fail closed. The unavoidable Go zero value is rejected by every persistence/publication boundary.

Redis compatibility is preserved for previously produced records because all four retained values keep the same JSON strings and canonical messages. `RedisOptimizationJobStore.Load` validates decoded failed records. `Save`/`PublishFailed`/transition encoding reject invalid codes. Failed records require exactly one canonical safe message for their code, and non-failed records cannot carry failure data. The controller independently validates the loaded failure before HTTP projection so an alternate store implementation cannot return an empty/arbitrary code or diagnostic text.

`queue_unavailable` and `result_expired` were removed from `OptimizationFailureCode`: queue outage is a retryable 503 operation error and expiry is an owner-scoped retryable 410 operation error, not a persisted failed-job state. They remain supported by the API client/UI as operation errors.

### Producers, consumers, cancellation, and retry

| Condition | Producer | Persisted result | Retry/cancellation semantics | Consumer |
| --- | --- | --- | --- | --- |
| Wrapped model/repository/result validation | worker classifier | `failed_validation` plus canonical safe message | terminal; valid earlier alternatives may remain | Redis decode, HTTP poll, telemetry `validation`, generated client, UI |
| CLP infeasible | solver classifier | `solver_infeasible` | terminal; UI recommends widening tolerance | Redis/HTTP/telemetry `infeasible`/client/UI |
| Worker-owned 30-second deadline while parent is live | worker finalizer | `solver_timeout` using a bounded `context.WithoutCancel` finalization context | terminal; valid partial alternatives survive | Redis/HTTP/telemetry `timeout`/client/UI |
| Unknown worker/solver failure | queue terminal handler after three deliveries | `worker_crash` | pending/retryable before exhaustion; terminal only after exhaustion | Redis/HTTP/telemetry `worker_crash`/client/UI |
| Parent worker context cancellation | worker shutdown | none | return `context.Canceled`; no terminal publication, ACK, admission release, or detached work; pending delivery is reclaimable | queue retry/reclaim |
| Queue/persistence unavailable | submitter or worker | none | 503 on submission or pending delivery on worker publication failure | HTTP operation error/client/UI |
| Owned result TTL expired | Redis expiry marker/controller | none | 410 and fresh submission; cross-user remains not-found | HTTP operation error/client/UI |

The frontend's `AbortController` only cancels local polling and ignores late results; it does not claim to cancel server execution. Every retained terminal code has focused worker message/telemetry and controller/client coverage.

### Authoritative `similarityScore`

The field is retained and now calculated once inside `SolutionValidator` from the same immutable repository snapshot used for model validation and macro/calorie projection. Original saved-diet quantities are converted to canonical `g` or `ml`; repeated entries are summed by meal UUID. The score is quantity-weighted Jaccard intersection-over-union over the union of original and alternative meal IDs:

`sum(min(original[id], alternative[id])) / sum(max(original[id], alternative[id]))`

The score is finite, bounded to `0..1`, and rounded to four decimal places. Tests establish identical `1`, partial overlap `0.3333`, and disjoint `0` fixtures. OpenAPI documents the formula and `multipleOf: 0.0001`; the generated client rejects values outside the bounds or precision contract. No client or solver-authored score is accepted.

## Changed paths and symbols

| Path | Added or modified Task 221 surface |
| --- | --- |
| `backend/internal/optimization/validator.go` | `OptimizationFailureCode`; `FailureCodeValidation`; `FailureCodeSolverTimeout`; `FailureCodeSolverInfeasible`; `FailureCodeWorkerCrash`; `ParseOptimizationFailureCode`; `OptimizationFailureCode.String`; `Valid`; `MarshalJSON`; `UnmarshalJSON`; `OptimizationFailure.Error`; `FailureCodeOf`; `DietAlternative` contract; `SolutionValidator.Validate`; new `quantityWeightedSimilarity`; `safeOptimizationFailure` retained as the internal classifier feeding the bounded enum |
| `backend/internal/optimization/validator_test.go` | Added `TestValidateSolutionCalculatesBoundedQuantityWeightedSimilarity` and `TestOptimizationFailureCodeJSONAcceptsLegacyValuesAndRejectsUnknownValues`; migrated string assertions to `String` |
| `backend/internal/worker/optimization_processor.go` | Added `OptimizationJobFailure.Valid`; tightened `RedisOptimizationJobStore.PublishFailed`, `handleProcessingError`, and `validateOptimizationJob`; retained `Terminal`, `publishFailure`, `safeFailureMessage`, and `telemetryStatusForFailure` as audited producers/consumers |
| `backend/internal/worker/optimization_processor_deadline_test.go` | Added shutdown cancellation, persisted shape, canonical message, and telemetry-vocabulary tests |
| `backend/internal/httpapi/optimization_controller.go` | `OptimizationController.GetJob` rejects invalid persisted failures; `optimizationJobData` projects only `Code.String()` |
| `backend/internal/httpapi/optimization_controller_test.go` | Expanded all-four-code polling test; added invalid persisted failure projection rejection |
| `backend/internal/app/task206_backend_integration_test.go` | `assertTask206Failure` uses the value object's `String` compatibility method |
| `api/openapi.yaml` | `OptimizationFailureCode` reduced to the four persisted terminal codes; `OptimizationAlternative.similarityScore` documents formula, bounds, rounding, and `multipleOf` |
| `scripts/generate-api-types.py` | Generated `OptimizationFailureCode` template reduced to the exact four-code enum |
| `scripts/test_generate_api_types.py` | Added OpenAPI/generated vocabulary and similarity drift test |
| `frontend/src/lib/api/generated.ts` | Regenerated exact four-code `OptimizationFailureCode` union |
| `frontend/src/lib/api/optimization-client.ts` | Added `TERMINAL_FAILURE_MESSAGES`, tightened `normalizeJob` and `normalizeAlternatives`, added `validSimilarityScore` and `isOptimizationFailureCode`; unknown/empty codes and unsafe server messages cannot reach UI state |
| `frontend/src/lib/api/optimization-client.test.ts` | Added unknown/empty code and score precision/bounds rejection coverage |
| `frontend/src/lib/stores/optimization.ts` | `displayFailure` now has an exhaustive typed terminal-message map and separate queue/expiry operation messages |
| `frontend/src/lib/components/OptimizationWorkflow.test.ts` | Browser-surface contract verifies displayed rounded similarity projection |
| `docs/design/DESIGN-004.md` | Documented complete vocabulary, producer/consumer matrix, safe messages, cancellation/shutdown/retry policy, persisted compatibility, and authoritative similarity formula/rounding |
| `docs/implementation/02_TASK_LIST.md` | Only row 221 status transitions from `OPEN` to `PREPARED` after evidence completion |
| `docs/implementation/preparation/task-221-preparation.md` | This preparation evidence |

No SQL, migration, queue stream, submission idempotency, controller serialization, acknowledgement, or task 222+ symbol was added or modified by Task 221.

## Verification-criteria mapping

| Row 221 criterion | Evidence | Result |
| --- | --- | --- |
| Arbitrary or empty failure codes cannot be constructed, persisted, decoded, or returned | Unexported code value; strict parse/JSON codec; `OptimizationJobFailure.Valid`; store load/publication validation; controller pre-projection validation; frontend runtime enum check | PASS |
| Every retained code has a tested producer and consumer | Four-code worker safe-message/telemetry table; processor timeout and terminal handling tests; four-code HTTP polling table; generated union and client/UI exhaustive maps | PASS |
| Wrapped validation, infeasible, timeout, cancellation, queue, expiry, and unknown failures follow policy without diagnostics | Existing wrapped classifier tests plus new safe-message persistence/controller/client tests; deadline and shutdown cancellation tests; DESIGN-004 matrix; queue/expiry client tests | PASS |
| OpenAPI/generated types/frontend messages use the same enum | OpenAPI enum, generator template, generated TS union, typed `Record<OptimizationFailureCode, string>`, Python drift test, `--check` | PASS |
| `similarityScore` has bounded nontrivial fixtures and documented meaning/rounding | Validator fixtures `1`, `0.3333`, `0`; OpenAPI `0..1` and `0.0001`; client runtime precision check; DESIGN-004 formula | PASS |
| Focused worker/controller/client/browser tests and contract drift checks pass | Commands below; 20 focused frontend tests, focused Go packages, Chromium verifier, generator unittest, Redocly | PASS |
| Traceability | Exact DESIGN-004 comments beside new executable symbols; design document updated; traceability validator passes | PASS |

## Commands and results

| Working directory | Command | Result |
| --- | --- | --- |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization ./internal/worker ./internal/httpapi` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization ./internal/worker ./internal/httpapi ./internal/app` | PASS after migrating the one Task 206 string call site |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./...` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/optimization ./internal/worker ./internal/httpapi` | PASS |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/... -coverprofile=coverage.out` and `go tool cover -func=coverage.out` | PASS; repository aggregate 79.9%; generated `coverage.out` removed after inspection |
| `frontend/` | `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/optimization-client.test.ts src/lib/stores/optimization.test.ts src/lib/components/OptimizationWorkflow.test.ts` | PASS, 20 tests / 90 expectations |
| `frontend/` | `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | PASS, 365 tests / 1602 expectations |
| `frontend/` | `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | PASS |
| repository root | `python3 scripts/generate-api-types.py --check` | PASS, generated types current |
| repository root | `python3 -m unittest scripts/test_generate_api_types.py` | PASS, 6 tests |
| repository root | `npx --no-install redocly lint api/openapi.yaml` | PASS with the one pre-existing OAuth 302/no-2xx warning |
| repository root | `python3 scripts/validate-traceability.py` | PASS |
| repository root | `python3 scripts/validate-task-list.py` | PASS before transition: 237 sequential tasks with ordered dependencies |
| repository root | `python3 scripts/verify-frontend.py` | PASS; desktop/mobile Chromium screenshots written under `/tmp/mealswapp-frontend-verifier/`; expected unauthenticated scenario 401 console messages were observed |
| repository root | `python3 scripts/check.py` | PASS on the final `PREPARED` state, including traceability/task-list/Go Doc/OpenAPI/generator/vulnerability/local-stack/UAT/backend/frontend/browser aggregate checks; retained the same pre-existing Redocly OAuth 302 warning |
| repository root | `git diff --check` | PASS |

## Final implementation hashes

| Path | SHA-256 |
| --- | --- |
| `api/openapi.yaml` | `102e35466ce1506bd587bc03a02d75761ffbdf9876f66398630d17bf6b020f14` |
| `backend/internal/app/task206_backend_integration_test.go` | `117336022754a3f3008efbd07646f761a5f9d6765810a121440b03a3b7bcd757` |
| `backend/internal/httpapi/optimization_controller.go` | `71f1c92e454ec2656d0c85562e938356b96b17685fc5136382fe2fbd2bd3945d` |
| `backend/internal/httpapi/optimization_controller_test.go` | `c49eb59e93b0e625abaab15d8656059aac8804984640c9c17723ee86726b017f` |
| `backend/internal/optimization/validator.go` | `aa11317e84d010d6098eeae5ff2aab5f934680195e70bc69f8b1880de735597c` |
| `backend/internal/optimization/validator_test.go` | `097338ed951cdcb7c43e6c93d75524be6d9e3aa603592583e82fc7a6c3255a20` |
| `backend/internal/worker/optimization_processor.go` | `47a0240e86810ddd41bd34fa1ef7b746fe5992a2f0a302074f5849df36602265` |
| `backend/internal/worker/optimization_processor_deadline_test.go` | `6770a7b32367242f52f4a824fb27d128c6ef21ae13b2bc9bfed77bd9c1b6e343` |
| `docs/design/DESIGN-004.md` | `e4e9e3cdde5f8715c586ae4a7f1c4a3da11697574203881e5b0df8a959e37782` |
| `frontend/src/lib/api/generated.ts` | `7fa42d073c9298afc314306767b43c03ecd86a9bfbfa275d87aae877fe5bc779` |
| `frontend/src/lib/api/optimization-client.ts` | `1543e8e319e0f96a2f978cb52cfec257dc71dca4296b984b98396207685b50ef` |
| `frontend/src/lib/api/optimization-client.test.ts` | `c41b0f6286996e240f84322ef39a85b968b573347309b8ddb28e198011c5891e` |
| `frontend/src/lib/components/OptimizationWorkflow.test.ts` | `022b8e15728f1808c7397ee2ffd9b31f9c56d8af0915c28ab6b3da1ceb87a28d` |
| `frontend/src/lib/stores/optimization.ts` | `9a117da419eb59de2e323f0e10ea7ab9bd3d8c1505b896966ceb38d6e10146e1` |
| `scripts/generate-api-types.py` | `721d12e4caad562ff5d6c5a3eeed4cf755cac1b887411715ce95bef06ee52b47` |
| `scripts/test_generate_api_types.py` | `f91e2786eccdf9e6e96e629a50fc52d5dff336fc9a4355615b9e1edde8cf0444` |

Task-list final hash after the single-cell status transition is `e57ae220a9a603aeba610f3e58992701b63ef5c42d2406bcd3bbac16ff79a1eb`.

## Risks and follow-up boundaries

- The Go type necessarily has a zero value, but zero cannot cross JSON, store publication/load, HTTP projection, or frontend decoding. A nonzero arbitrary value cannot be constructed outside the optimization package because the representation is unexported.
- Quantity-weighted Jaccard compares canonical quantities within each meal UUID. Summing grams and millilitres across different IDs occurs only in the dimensionless numerator/denominator ratio, consistent with the diversity objective's documented base-unit policy; it does not claim nutritional or volumetric equivalence.
- Previously produced valid failed records remain readable. A manually injected or foreign record using the former non-terminal `queue_unavailable`/`result_expired` failure values now fails closed as malformed dependency state; current production code had no terminal producer for either value.
- Repository aggregate backend coverage is 79.9%, below the phase-end 100% goal. Task 221's new critical branches have focused tests, but closing inherited package-wide coverage remains a phase-completion concern rather than task 221 scope.
- The four exported canonical code variables should be treated as immutable vocabulary values. Their internal value is unforgeable, and every boundary revalidates them; changing the public API to accessor functions would add churn without improving boundary safety.
- Task 222 still owns submission idempotency/concurrency and must not infer new authority from this preparation.

## Preparation decision

Every Task 221 verification clause is directly supported by implementation inspection and passing focused/full checks. Task 221 is eligible to transition from `OPEN` to `PREPARED`; no other task status is justified for change.

Post-transition confirmation: rows 220/221/222 are respectively `PREPARED`/`PREPARED`/`OPEN`; `python3 scripts/validate-task-list.py`, `python3 scripts/validate-traceability.py`, and `git diff --check` all pass after the transition.

## Repair evidence — 2026-07-17

The rejected-review findings in `docs/implementation/reviews/task-221-review.md` were repaired without changing any task status or Task 220/222 behavior.

### Repaired symbols and boundaries

| File | Exact repaired or added symbol | Evidence |
| --- | --- | --- |
| `backend/internal/optimization/validator.go` | `ValidateDietAlternative`, `boundedProjectionNumber` | One authoritative server validator rejects malformed meal/macro/calorie projections and non-finite, out-of-range, or non-four-decimal `SimilarityScore` values. |
| `backend/internal/optimization/validator.go` | `FailureCodeOf`, `safeOptimizationFailure` | Every relevant `errors.As` pointer target is checked for nil; typed-nil optimization, solver, and repository errors cannot be dereferenced or preserved as an apparently classified failure. |
| `backend/internal/worker/optimization_processor.go` | `RedisOptimizationJobStore.PublishCompleted`, `RedisOptimizationJobStore.PublishFailed`, `validateOptimizationJob`, `validateOptimizationAlternatives` | The shared alternative validator runs before terminal persistence and after Redis decode. Completed jobs require one to three valid alternatives; failed partial results allow zero to three; non-result states cannot carry alternatives. |
| `backend/internal/worker/optimization_processor.go` | `OptimizationProcessor.handleProcessingError` | A typed-nil repository error remains an unknown retryable error and is not published as a terminal validation failure. |
| `backend/internal/httpapi/optimization_controller.go` | `OptimizationController.GetJob`, `validateOptimizationJobAlternatives`, `optimizationJobData` call boundary | Alternate store implementations cannot bypass authoritative result validation before HTTP projection. |
| `backend/internal/optimization/validator_test.go` | `TestValidateDietAlternativeRejectsMalformedResultShape`, `TestValidateDietAlternativeRejectsInvalidSimilarityScores`, `TestOptimizationFailureClassificationHandlesTypedNilErrors` | Covers invalid numeric shape, `-0.0001`, `1.0001`, `0.12345`, NaN/infinity, valid `0.1234`, direct typed nil, and the injected solve seam. |
| `backend/internal/worker/optimization_processor_deadline_test.go` | `TestOptimizationProcessorTreatsTypedNilFailureAsRetryableUnknown` | Covers worker retry and telemetry behavior for typed-nil optimization and repository errors without publication or panic. |
| `backend/internal/worker/task221_publication_test.go` | `TestTask221RedisStoreValidatesSimilarityScoreAtPublicationAndDecode` | Live Redis verifies publication rejection, malformed decoded-record rejection, and valid `0.1234` round-trip. |
| `backend/internal/httpapi/optimization_controller_test.go` | `TestOptimizationHTTPRejectsInvalidPersistedSimilarityBeforeProjection` | HTTP polling returns bounded 503 dependency errors for all three invalid finite scores and 200 for valid `0.1234`. |

### Repair verification

| Working directory | Command | Final result |
| --- | --- | --- |
| `backend/` | `go test ./internal/optimization ./internal/worker ./internal/httpapi -count=1` | PASS |
| `backend/` | `go test -race ./internal/optimization ./internal/worker ./internal/httpapi -count=1` | PASS |
| `backend/` | `go vet ./internal/optimization ./internal/worker ./internal/httpapi` | PASS |
| `backend/` | `go test ./... -count=1` | PASS |
| `backend/` | `go test ./internal/optimization ./internal/worker ./internal/httpapi -coverprofile=/tmp/task-221-repair.coverage.out -count=1` | PASS; optimization 85.3%, worker 21.3%, HTTP 89.7%, combined focused packages 77.7%; inherited phase coverage exception remains unchanged. |
| repository root | `bash scripts/start-services.sh` followed by `go test ./internal/worker -run '^TestTask221RedisStoreValidatesSimilarityScoreAtPublicationAndDecode$' -v -count=1` | PASS against live local Redis. |
| repository root | `python3 scripts/validate-task-list.py` | PASS; 237 sequential tasks with ordered dependencies. |
| repository root | `python3 scripts/validate-traceability.py` | PASS. |
| repository root | `git diff --check` | PASS. |
| repository root | `python3 scripts/check.py` | FAIL only at the existing phase-wide 100% Go coverage gate after all preceding validators, vulnerability scan, stack/UAT checks, frontend verification, and backend tests passed: daily diet 80.1%, optimization 85.3%, worker 64.6%. This inherited exception remains outside Task 221 repair scope and no new Task 221 failure was reported. |
| repository root | `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-221-review.md` | FAIL on the unchanged rejected-review document: metadata says `audited_symbol_count=30` while its audit table has 31 rows. The review file is outside the requested preparation refresh and was not edited. |

### Repair fingerprints and status boundary

| Path | SHA-256 after repair |
| --- | --- |
| `backend/internal/optimization/validator.go` | `4119ce67e0353c725d9e0feca9f13379966f1d89eaee5477bee706929ae6b09a` |
| `backend/internal/optimization/validator_test.go` | `950349bf26c0a28df9b2f3037243b26d0ccd0c0203f69fb5d76b612ad1742f73` |
| `backend/internal/worker/optimization_processor.go` | `50ea0a2165cb6ec19f4d4fcb7f83d1ce51ff1f65f569dcf788d652b2d8933427` |
| `backend/internal/worker/optimization_processor_deadline_test.go` | `fc79e1cc9329eac1e5f773ef9ba3c6acf1a226e4e8876d843bb09e9af0de5b37` |
| `backend/internal/worker/task221_publication_test.go` | `8e7270f161af93763c3e6023b48e8c73304b841aaca2548305593594e31f97f3` |
| `backend/internal/httpapi/optimization_controller.go` | `422b0232a203d05071e33c050fafe40a681120ee4544011d6c2c100405208664` |
| `backend/internal/httpapi/optimization_controller_test.go` | `4425e033ea214aee6e35bb1066cd6296243a8ed7d601e961818a305472fb811b` |
| `docs/implementation/02_TASK_LIST.md` | `ff97c9908298a6215b3211cce5ebb8931569940d2e534b3387b1c8b60374f6d4` |

The repair did not edit `docs/implementation/02_TASK_LIST.md`, the Task 220 generation pipeline, or Task 222 submission idempotency/admission symbols. During final validation, a concurrent external edit changed Task 220 from `PREPARED` to `PASSED`; it was preserved as user-owned work. Task 221 remains `PREPARED` and Task 222 remains `OPEN` exactly as they were throughout this repair.

## Rejected-review residual repair evidence — 2026-07-17

This refresh closes only the two residual findings in `docs/implementation/reviews/task-221-review.md`. It preserves the cumulative Task 222 submission work and does not change any task status.

### Exact repair surface

| Path and symbol | Repair evidence |
| --- | --- |
| `backend/internal/optimization/validator.go:149` — `DietAlternative.UnmarshalJSON` | Uses a raw JSON field to distinguish an explicit numeric `0` from omitted or `null` `similarityScore`. Omitted, null, and non-number values fail before `ValidateDietAlternative`; valid numeric zero remains `0`. |
| `backend/internal/optimization/validator.go:404` — `safeOptimizationFailure` | Preserves a non-nil existing `OptimizationFailure` only when `existing.Code.Valid()`; every invalid non-nil value is wrapped with the bounded `worker_crash` code while its cause remains internal. |
| `backend/internal/httpapi/optimization_controller.go:360` — `OptimizationController.GetJob` load-error boundary | Known polling errors retain their existing mapping; an unknown persisted decode/store error becomes the generic `503 optimization_unavailable` dependency response and cannot project malformed state. Submission, acknowledgement, admission, hashing, and publication-repair paths are unchanged. |
| `backend/internal/optimization/validator_test.go:164` — `TestSafeOptimizationFailureNormalizesInvalidExistingFailure` | Directly proves the invalid exported zero-valued error becomes `worker_crash` with a non-empty safe error string. |
| `backend/internal/worker/task221_publication_test.go:73` — `TestTask221RedisStoreRejectsMalformedRawSimilarityScore` | Injects raw completed-job payloads into live Redis and rejects omitted, null, and string scores while accepting and round-tripping numeric zero. |
| `backend/internal/httpapi/optimization_controller_test.go:415` — `TestOptimizationHTTPRejectsMalformedRawSimilarityScore` | Decodes raw persisted payloads through the polling store seam; malformed score presence/type returns bounded `503` with no projected data, while numeric zero returns `200`. |
| `backend/internal/worker/optimization_processor_deadline_test.go:154` — `TestOptimizationProcessorTreatsInvalidFailureAsRetryableWorkerCrash` | Runs the worker with a solver returning `&optimization.OptimizationFailure{}`; the returned error is classified `worker_crash`, remains retryable without terminal publication, and uses the fixed `failed` solver telemetry bucket. |

### Reproduction and final commands

| Working directory | Exact command | Result |
| --- | --- | --- |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization ./internal/worker ./internal/httpapi -run 'TestSafeOptimizationFailureNormalizesInvalidExistingFailure\|TestTask221RedisStoreRejectsMalformedRawSimilarityScore\|TestOptimizationProcessorTreatsInvalidFailureAsRetryableWorkerCrash\|TestOptimizationHTTPRejectsMalformedRawSimilarityScore' -count=1` | Expected pre-fix FAIL: classifier and worker returned an invalid empty code; Redis and HTTP accepted omitted/null score as zero; non-number already failed JSON decoding. |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization ./internal/worker ./internal/httpapi -run 'TestSafeOptimizationFailureNormalizesInvalidExistingFailure\|TestOptimizationFailureClassificationHandlesTypedNilErrors\|TestTask221RedisStoreRejectsMalformedRawSimilarityScore\|TestTask221RedisStoreValidatesSimilarityScoreAtPublicationAndDecode\|TestOptimizationProcessorTreatsInvalidFailureAsRetryableWorkerCrash\|TestOptimizationProcessorTreatsTypedNilFailureAsRetryableUnknown\|TestOptimizationHTTPRejectsMalformedRawSimilarityScore\|TestOptimizationHTTPRejectsInvalidPersistedSimilarityBeforeProjection' -count=1` | PASS. |
| `backend/` | Same focused command with `go test -race` | PASS in optimization, worker, and HTTP packages. |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization ./internal/worker ./internal/httpapi -count=1` | PASS. |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -count=1` | PASS. |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1` | PASS. |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS. |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS; no called vulnerabilities. |
| `backend/` | `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization ./internal/worker ./internal/httpapi -coverprofile=/tmp/task-221-final.coverage.out -count=1` and `go tool cover -func=/tmp/task-221-final.coverage.out` | PASS; optimization 84.3%, worker 64.8%, HTTP 89.7%, 84.1% combined scoped statements. Existing phase coverage exception is unchanged. |
| repository root | `python3 scripts/validate-task-list.py` | PASS; 237 sequential tasks with ordered dependencies. |
| repository root | `python3 scripts/validate-traceability.py` | PASS. |
| repository root | `python3 scripts/generate-api-types.py --check` | PASS; generated API types are current. |
| repository root | `python3 -m unittest scripts/test_generate_api_types.py` | PASS; 7 tests. |
| repository root | `npx --no-install redocly lint api/openapi.yaml` | PASS with the unchanged OAuth callback no-2XX warning. |
| repository root | `git diff --check` | PASS before this evidence refresh. |
| repository root | `python3 scripts/check.py` | Reached the existing Phase 07 coverage gate after traceability, task-list, Go Doc, OpenAPI, vulnerability, local-stack, UAT, browser, focused, full backend, and race checks passed; then exited 1 only because documented measured coverage remains below 100% in daily diet 80.1%, optimization 84.3%, and worker 64.8%. Aggregate backend coverage was 88.3%. |

### Final implementation fingerprints and preserved boundaries

| Path | SHA-256 |
| --- | --- |
| `backend/internal/optimization/validator.go` | `5ceb96bf19396ff9bc33e1de54fc879a62c5c948815b7fc1ac2b1468f68c6efd` |
| `backend/internal/optimization/validator_test.go` | `d7ac4c7b1dfde12def8f49a435e9d2530d568255c198eeb984346f68835e6075` |
| `backend/internal/worker/optimization_processor_deadline_test.go` | `425a4ba613d9d4d25041ae0aa6447d1a03475cc08564bb706c16b5855d4a372f` |
| `backend/internal/worker/task221_publication_test.go` | `1a91f30779bf8cc3139c10d71a9a6520785d9698d6b1d8ced9a067838c4e3d4c` |
| `backend/internal/httpapi/optimization_controller.go` | `a72cb184748e9f699a93654b091d2fa2bac524e9acafce20ed7cf0795480674b` |
| `backend/internal/httpapi/optimization_controller_test.go` | `2b3f803bfcb2d3b8210018e8c9b1d94fc463ec98f6c9852b5e9f06aa293bb023` |
| `docs/implementation/02_TASK_LIST.md` | `ff97c9908298a6215b3211cce5ebb8931569940d2e534b3387b1c8b60374f6d4` |
| `docs/implementation/preparation/task-222-preparation.md` | `036dcae0624c41899684640e205714aec0cad0a7a4f75c2d604574ab78971461` |

The task rows remain exactly `220 PASSED`, `221 PREPARED`, and `222 OPEN`. The task-list and Task 222 preparation hashes match the rejected review inventory, proving this residual repair did not alter either boundary. No later task was implemented.
