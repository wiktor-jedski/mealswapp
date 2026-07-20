# Task 232 Preparation — Backend Integration and Functional Regression Gate

## Scope and conclusion

- Task: **232 — Phase 07.01 Backend Integration and Functional Regression Gate**.
- Architecture source: `ARCH-004: JobStatusTracker`; collaborating static aspects are `JobQueueManager`, `LPSolverWrapper`, `ConstraintBuilder`, `ObjectiveFunction`, `DiversityPenalizer`, and `SolutionValidator`.
- Requirements traced by the gate: `SW-REQ-006`, `SW-REQ-021`, `SW-REQ-022`, `SW-REQ-023`, `SW-REQ-030`, `SW-REQ-080`, and `SW-REQ-082`.
- Current task status remains `OPEN`. This work did not edit `docs/implementation/02_TASK_LIST.md` or any task status.
- The shared worktree already contained cumulative concurrent Phase 07.01 changes. No existing path was cleaned, reverted, staged, or rewritten. Task 232 adds only `backend/internal/app/task232_backend_regression_test.go` and this evidence document.
- Conclusion: the focused live-dependency suite, focused race suite, serial aggregate backend normal/race gates, static/security checks, API/traceability checks, and no-cache packaged-worker image check pass.

## Sources and passed evidence inspected

The implementation was derived from the Task 232 row, `docs/architecture/ARCH-004.md`, `docs/design/DESIGN-004.md`, `docs/design/DESIGN-008.md`, `docs/design/DESIGN-014.md`, `docs/testing/integration/ARCH-004-obligations.md`, the current backend integration layout, `scripts/start-services.sh`, `scripts/check.py`, and `scripts/verify-clp-worker-image.sh`.

Current preparation/review evidence for the completed Phase 07.01 backend prerequisites was checked before implementation:

- Task 216: immutable typed Daily Diet create response, PostgreSQL claim/replay, cross-instance atomicity, rollback, and malformed persistence rejection.
- Task 221: bounded failure vocabulary, similarity projection, terminal publication validation, timeout/cancellation policy, and safe HTTP projection.
- Task 223: exact submission outcomes, bounded cleanup, privacy-safe telemetry, and race-safe sink behavior.
- Task 225: publication-before-acknowledgement, atomic terminal finalization and duplicate cleanup, embedded scripts, Redis topology, and live group/restart recovery.
- Task 226: distinct waiting/pending populations, authoritative Redis age sources, nonnegative skew handling, and exact metric units/labels.

Earlier Phase 07 integration evidence from Tasks 206 and 210 and the current PASSED evidence through Task 230 were used as regression inventory rather than copied into a second giant test. Existing tests remain the authoritative focused evidence for timeout child termination, cancellation/reclaim, duplicate delivery, retry exhaustion, result expiry, partial publication, queue recovery, and telemetry boundaries.

## Added integrated test

`TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers` composes real PostgreSQL, real Redis, authenticated Fiber routes, the Redis stream queue, the dedicated worker, repository reload, the solver pipeline, and pinned CLP `1.17.11`. It adds the integration seam that was missing from the prerequisite tests:

1. Creates a Daily Diet through the production API, replaces and deletes it, then proves the same create key returns the immutable original `201` data without recreating persistence.
2. Submits semantically equivalent optimization JSON with reordered set-like exclusions and negative zero, proves exact `202` acknowledgement replay, one stream publication, stable `Location`, stable job ID/poll URL, and a changed-body `409`.
3. Proves a free user receives exact `403 entitlement_denied`, an entitled non-owner receives ownership-safe `404 not_found`, and cross-user polling discloses no job state.
4. Runs the normalized job through the real worker and packaged CLP boundary, then validates one-to-three server-recalculated alternatives, target macros, calorie ordering, exclusions, and distinct meal sets.
5. Seeds the real PostgreSQL durable pending-publication shape, proves repair rechecks current entitlement before side effects, restores entitlement, retains the server-created job ID, publishes once, and makes later exact replay side-effect free.
6. Submits two unrelated entitled users concurrently and proves independent `202` jobs both complete through the real worker.
7. Uses an unreachable Redis endpoint to prove exact retryable `503 queue_unavailable` behavior without synchronous API-process solving.
8. Checks exact root JSON key sets, exact error key sets, request-ID correspondence, fixed safe messages, retryability, and exact acknowledgement data cardinality.

### Added symbols

- `TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers`
- `task232Response`
- `task232Call`
- `task232AssertSuccess`
- `task232AssertAcknowledgement`
- `task232AssertError`
- `task232AssertJSONKeys`
- `task232AssertKeySet`
- `task232SortedKeys`
- `task232OptimizationBody`
- `task232StorePendingOptimization`
- `task232AppendEntitlement`
- `task232StreamLength`
- `task232ResetRedis`
- `task232ConcurrentRequest`
- `task232SubmitConcurrentUsers`

No production symbol, interface, migration, OpenAPI source, generated client, task row, or status was changed by Task 232.

## Focused regression matrix

| Required behavior | Exact test evidence | Boundary |
| --- | --- | --- |
| Immutable Daily Diet create replay | `TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers`; `TestDailyDietProductionAPIWithLivePostgres`; `TestPostgresDailyDietCreateClaimReplaysImmutableResponseAndCascades` | PostgreSQL + API |
| Malformed/legacy persisted replay rejection | `TestDecodeDailyDietCreateResponseRejectsInvalidPersistedBodies`; `TestPostgresDailyDietCreateClaimRejectsLegacyDualAndRollsBackWrites` | codec + PostgreSQL |
| Normalized optimization replay and repair | `TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers`; `TestOptimizationHTTPPublishedAcknowledgementReplayHasNoCurrentStateSideEffects`; `TestOptimizationHTTPUnpublishedRepairRevalidatesAndPublishesOnce` | PostgreSQL + Redis + API |
| Ownership and entitlement isolation | `TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers`; `TestOptimizationHTTPEntitlementAndOwnershipGuards` | auth + entitlement + repository + API |
| Strict terminal publication/failure | `TestTask210WorkerPublishesPartialAlternativesOnLaterSolverFailure`; `TestTask221RedisStoreValidatesSimilarityScoreAtPublicationAndDecode`; `TestRedisOptimizationJobStoreTerminalTransitionsAreAtomic` | worker + Redis + HTTP projection |
| Solver model/codec/process boundaries | `TestTask218PackagedCLPConstraintFixture`; `TestTask219PackagedCLPLexicographicObjective`; all `TestLPSolverWrapper*`, `TestCLPVersion*`, `TestSerializeLP*`, and `TestParseCLPSolution*` | model + parser + child process + CLP |
| Distinct alternatives and canonical validation | `TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers`; `TestPublicAlternativeGenerators*`; `TestAlternativePipeline*`; `TestSolutionValidator*` | PostgreSQL snapshot + solver + validator |
| Attempts, terminal finalization, duplicate delivery | `TestJobQueueConcurrentConsumersDoNotProcessDuplicateLogicalJob`; `TestJobQueueRetriesAndTerminallyFailsAfterThreeAttempts`; all `TestTask224*` and `TestTask225*` | real Redis Streams + embedded Lua |
| Stream/group/restart recovery | `TestTask225LiveManagerRecoversGroupAndDataLoss`; `TestTask225LiveManagerRecoversAfterRedisRestart`; `TestTask225AuthorizationAndConnectivityErrorsFailClosed` | live manager + isolated real Redis restart |
| Accurate waiting/pending ages and bounded metrics | all queue/observability `TestTask226*`; `TestOptimizationTelemetryUsesBoundedLabelsAndNoPrivateData`; all `TestTask223*` | Redis metadata + telemetry sink |
| Timeout, cancellation, outage, duplicate delivery | `TestTask206TimeoutAndOwnershipGate`; `TestOptimizationProcessorAppliesOneDeadlineAcrossAlternativeSolves`; `TestOptimizationProcessorLeavesShutdownCancellationPendingForRetry`; `TestJobQueueCancellationReachesProcessorAndLeavesDeliveryRecoverable`; Task 206 duplicate/outage path; Task 232 outage path | API + queue + worker + process boundary |
| Concurrent users | `TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers`; `TestOptimizationHTTPDifferentUsersProceedIndependentlyAndSameKeyIsUserScoped` | PostgreSQL + Redis + API + worker |
| Exact statuses and safe envelopes | Task 232 exact key/message/status assertions; `TestOptimizationHTTPAdmission429UsesSharedRetryContract`; `TestOptimizationHTTPFailedPollingUsesSafeSolverMessages` | HTTP/OpenAPI contract |

## Commands and results

Local dependencies were started with `bash scripts/start-services.sh`. Focused Redis tests used database 14 through `MEALSWAPP_REDIS_URL=redis://localhost:6379/14`; packages were run sequentially so independent integration fixtures did not contend for one stream.

| Command | Result |
| --- | --- |
| `cd backend && MEALSWAPP_REDIS_URL=redis://localhost:6379/14 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -v ./internal/app -run '^TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers$' -count=1` | PASS; live PostgreSQL/Redis/API/worker/CLP replay, repair, isolation, concurrent-user, and outage gate. |
| Sequential focused `go test` commands over `internal/repository`, `internal/optimization`, `internal/queue`, `internal/worker`, `internal/observability`, `internal/httpapi`, and `internal/app` with the exact test families in the matrix above | PASS in all seven packages. |
| The same seven focused package selections under `go test -race ... -count=1` | PASS in all seven packages. |
| `cd backend && ... go test ./... -p 1 -count=1` | PASS; complete backend suite. |
| `cd backend && ... go test -race ./... -p 1 -count=1` | PASS; complete backend race suite. |
| `cd backend && ... go vet ./...` | PASS. |
| `cd backend && ... go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS; zero called vulnerabilities. |
| `cd backend && ... go test ./internal/app -run '^TestTask232' -coverprofile=/tmp/task-232-app.coverage.out -count=1` plus `go tool cover -func=...` | PASS; Task 232 selection covers 57.1% of the complete app package statements. |
| `cd backend && ... go test ./internal/optimization -run 'Test(LPSolverWrapper\|CLPVersion\|SerializeLP\|ParseCLPSolution)' -count=1` | PASS; exact solver process, version, serialization, and parser boundary selection. |
| `bash scripts/verify-clp-worker-image.sh` | PASS; no-cache `linux/amd64` image built as SHA-256 `1cddbbfdd92614312908708416da105f045639b88e62039afb06f000705c84f7`, contained executable worker, and reported `Coin LP version 1.17.11`. |
| `python3 scripts/validate-task-list.py` | PASS; 237 sequential tasks with ordered dependencies; Task 232 remains OPEN. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-phase07-go-doc.py` | PASS. |
| `python3 scripts/generate-api-types.py --check` | PASS; generated API types current. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS; one pre-existing explicitly ignored OAuth callback 2XX warning. |
| `git diff --check -- backend/internal/app/task232_backend_regression_test.go docs/implementation/preparation/task-232-preparation.md` | PASS after this document was added. |

## Execution corrections and deviations

- First live attempt failed because the manually seeded test claim used the external `/api/v1/optimization/jobs` path instead of the controller's durable route scope `/optimization/jobs`. The test fixture was corrected; production code was not changed.
- Second live attempt reached the concurrent-user section but associated unordered channel responses with ordered owner cookies. The harness now carries the original request index and owner-scoped polling is deterministic. Production code was not changed.
- One initial coverage command was invoked from the repository root and printed the expected “cannot find main module” error. It did not run tests or change files; the command was rerun from `backend/` and passed at 57.1%.
- `python3 scripts/check.py` was not run because it includes frontend/browser/coverage work owned by later Tasks 233–235. Task 232 instead ran the complete backend normal/race/static/security gates, focused live integration suites, API drift/lint, traceability, and packaged-image verification directly.
- The Task 232 row has no numeric package-coverage threshold. The measured 57.1% is for the single composed app selection; complete-package and aggregate Phase 07.01 100% coverage disposition remains Task 235. No new coverage exception is claimed here.
- A live multi-node Redis Cluster was not introduced. Task 225's current passed evidence structurally verifies common hash-slot topology and executes the same scripts against standalone Redis; Task 232 reran that focused surface.
- The no-cache worker build creates local Docker image `mealswapp-optimizer:task-212`; it creates no repository artifact.

## Current SHA-256 fingerprints

| Path | SHA-256 |
| --- | --- |
| `backend/internal/app/task232_backend_regression_test.go` | `c7483d952cb84e139f6ca945c200fa54dbb0e38ee12eacd291733e89e5edb567` |
| `backend/internal/app/task206_backend_integration_test.go` | `f9e0de887d0670b914267730d36a8f897bff42635db7d41c5f89fd9865ff6629` |
| `backend/internal/app/daily_diet_api_integration_test.go` | `c58009446a62bdfff9fcbcccb003ad66ab25a3f242687169619098b456ce6eb0` |
| `backend/internal/repository/daily_diet_create_claim_test.go` | `9dd7069aae8f8c0247eb46ba54ba0240ebd311c9a94c22164b2edc874e46e4d2` |
| `backend/internal/httpapi/task222_optimization_submission_integration_test.go` | `37f390e9f7fd006a492cb9a43c593307134a1e510bab93fb02d3affe52255e55` |
| `backend/internal/httpapi/task223_submission_observability_test.go` | `542629229075b1c8f3e6d80dee7036cd6c1cc3e15405dd525e1138c72541e563` |
| `backend/internal/optimization/clp_wrapper_test.go` | `ad201e23848593fe5f783dda419b7ffc5ea9d969f9f98e6152422e535e18664f` |
| `backend/internal/optimization/task220_pipeline_test.go` | `0704646e1bd48048dc95ca2320dd2018d6c5242cf8c0092b166b646f30eccea5` |
| `backend/internal/queue/job_queue_integration_test.go` | `4eeb0a386fcc6fdc52b2a60e38920f3f0e7cb9233a94e381d9671a502263b420` |
| `backend/internal/queue/task225_queue_test.go` | `b9c35c96fb1972de5c48a96daa28ae0a26b9a4f8b909ab26bcfae5435d7ad9c5` |
| `backend/internal/queue/task226_queue_age_test.go` | `4f7c8ce3ce102c0f5f7e52814c6c1d5b7cd895489538c61223668afa9753b58d` |
| `backend/internal/worker/task210_swe5_integration_test.go` | `3f76d08b34d74a3ac965cb75e285d60003e3d638ef843b420ef84ab47f2599f7` |
| `backend/internal/worker/task221_publication_test.go` | `1a91f30779bf8cc3139c10d71a9a6520785d9698d6b1d8ced9a067838c4e3d4c` |
| `backend/internal/worker/worker_integration_test.go` | `935c3caa093d220942a7e468355881ad38f94d29d36f18bc983bde08bda1615c` |
| `backend/internal/worker/optimization_processor_deadline_test.go` | `2c7601ef63fc4c6e46d256c958251f8cfc655cdec6488a707e87339a767c1d24` |
| `backend/internal/observability/task223_optimization_test.go` | `251deca606e3836d73508576b3f076567c3db43e0b09ebb14c29c47a44e09ac1` |
| `backend/internal/observability/task226_queue_age_test.go` | `5dc75753fc82fabed79d442b1457c2e21da63f28e82216875c21fe8d8a4ce67b` |
| `docs/architecture/ARCH-004.md` | `bedcf45c79f50cfe24313345ac2ba664130ee44d7d2f35c7b1de0a06e0abe867` |
| `docs/design/DESIGN-004.md` | `45dd31e2afe1480ec54f540031ec45289f8287fbd0ef1a2d2cee86dea16a5474` |
| `docs/design/DESIGN-008.md` | `551880a70a3b42698e11632a471957bbecc6011900cf121c34f1ff3c3db18b87` |
| `docs/design/DESIGN-014.md` | `f9f6521d89e6d31306422017e07af5630ba4d8da56907174f3653ea0d72e9fe4` |
| `docs/testing/integration/ARCH-004-obligations.md` | `ffc3c036ad32a58fc340a9dddcd5cedbefa37ee687c2b200e99ac9b53cca91b0` |
| `docs/implementation/02_TASK_LIST.md` | `3bdabe886facb2b96875489dcce7186d13acecc0a1582ff8e2fc8cc1dff62ebf` — read-only Task 232 snapshot |
| `backend/Dockerfile.worker` | `dbf7af9f61f8d7ac0aaf9a5c42a9f34c841ee1897e61f1eeca8172d4a39dd273` |
| `scripts/verify-clp-worker-image.sh` | `1d56d9472d77390ce564664ee0ea1cd7fdcebd42a62e60a71ad0e79526c8fd36` |

The Task 232 test hash above is the final post-verification content hash. This preparation document intentionally does not self-hash.

## Preparation decision

Task 232's implementation and verification criteria are satisfied by the new composed live-dependency regression plus the current focused queue/worker/solver/metrics suites. No unresolved Task 232 correctness, security, race, contract, or degraded-path finding remains. Per explicit instruction, the task status remains `OPEN` for separate review/orchestration.
