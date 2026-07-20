# Task 236 Preparation — ARCH-004 SWE.5 Integration Verification

## Scope, preservation, and decision

- Task: **236 — Phase 07.01 SWE.5 Integration Verification**.
- Architecture component: `ARCH-004`, Linear Programming Optimizer; task-row static aspect: `JobStatusTracker`.
- Verification date/time: `2026-07-18T18:21:13+02:00` (Europe/Warsaw).
- Skill applied: `/home/wiktor/.agents/skills/swe5-integration-test/SKILL.md`, obligation template, and mandatory `CHECKLIST.md`.
- Task 236 remains **OPEN**. This work did not edit a task status or any task row.
- The shared worktree already contained cumulative concurrent Phase 07.01 production, test, API, frontend, script, planning, review, and evidence changes. No path was cleaned, reverted, staged, or rewritten. Task attribution below is limited to the ARCH-004 obligation rewrite, adjacent test trace comments, capacity-evidence trace comments, and this preparation document.
- No production executable symbol or behavior was added or changed. Existing architecture-level tests already exercised every required scenario, so duplicating them would have reduced clarity. Task 236 maintained those tests by adding exact obligation/architecture/design/requirement traces and executed them as one SWE.5 gate.
- The rejected review at `docs/implementation/reviews/task-236-review.md` was repaired only at its three blocking trace locations. Test bodies, obligation content, production code, and the Task 236 row remain unchanged.
- Decision: all eight `ARCH-004` obligations pass the mandatory SWE.5 checklist and focused verification. No follow-up implementation defect or incomplete obligation was found.

## Authoritative sources read

The requested generic files do not exist under those names in this repository. Their authoritative repository equivalents were used:

| Requested source | Repository source used |
| --- | --- |
| `docs/implementation/tasks.md` | `docs/implementation/02_TASK_LIST.md`, Task 236 row at line 243 |
| `docs/design/software-architecture.md` | `docs/architecture/01_SOFT_ARCH_DESIGN.md` and component source `docs/architecture/ARCH-004.md` |
| `docs/design/code-design.md` | `docs/design/DESIGN-004.md` plus collaborating `DESIGN-001`, `DESIGN-008`, `DESIGN-014`, and `DESIGN-017` |

Also read:

- all existing `docs/testing/integration/*.md`, with `ARCH-004-obligations.md` as the changed component document;
- `docs/requirements/01_SOFT_REQ_SPEC.md` for `SW-REQ-006`, `021`, `022`, `023`, `030`, `042`, `043`, `080`, and `082`;
- Phase 07.01 preparation evidence `task-213-preparation.md` through `task-235-preparation.md`;
- the current integration/browser/capacity tests cited by the obligations;
- the skill's complete `SKILL.md`, `CHECKLIST.md`, obligation template, and generator script.

## Phase 07.01 evidence manifest

These SHA-256 values identify the exact prerequisite preparation evidence inspected before the SWE.5 decision:

| Task | Preparation evidence SHA-256 |
| --- | --- |
| 213 | `83bb5de5f8c4138c9e15d1f8a64725cc0fe5b7ee31f2ae3ed3f68dafa28ccea0` |
| 214 | `7ca71c5a30f6de282bd0b3bb4f09840f2e65d382558b08dcf3035ab252270aa4` |
| 215 | `b59e6de8ea56809c7d33ab57e36bb6719aa9f8bbc1cbdbbeccfac8d18a8829cf` |
| 216 | `3917278b0fcff5ebddc53ad007a88cadb3795337a9960e3836c6ee2de7690c52` |
| 217 | `ead0e0c6498e07ad1845a968e95fe2fc6ed13988c9a24d3cbee3437549e7b0e1` |
| 218 | `3cd31a1fc483e2928187ca6b99d45d14f48a68b6b1b57e99bb9804698680f4b7` |
| 219 | `df2d94aacc193aba9bcdffc7ea78396ba46d71eb649af9db4c815674f87c97a6` |
| 220 | `32657f553c0419e80655a4d488f0724d60fc69faa6081223a054756a18752f0a` |
| 221 | `ecf646e5b92139608ac4b74326f7d921064a24d420deb22b764a2a3e6657a632` |
| 222 | `199b902b6436355af973d341bab914ef3ad6d4575c25028c53b016cf1690a548` |
| 223 | `45570f5a43d91144666280e3830f7a41be6208827e1fcc1aa14a125437c222a3` |
| 224 | `8b7022dd052faeb2c47c6f50c446a25b2e65e7fe2694689b01e3cd8c4fd72680` |
| 225 | `254d1aa5082183e71f86f9517e4aab8cf4c7eb4fadf30625fbe6be6173a8b78c` |
| 226 | `7648b4629aad277ac1633769839cba7f87408242bc659a4e5917e24de04051d9` |
| 227 | `dd4b4e05d2c665fcc461ebbe964a8b127dafb55e1813ec9e240235697109944d` |
| 228 | `2b1c3201303d03f2a15d80b4743a78a3cdf3fc56b4b9dda444c4cb76de5f5352` |
| 229 | `44ec1670ac0c402873eefdafb3bf1a7e6e43a1033d8cc4d1114379b771e0367e` |
| 230 | `11849b62ffacb742e7bf6a0269729bfa8a36d9e4b9aa4a353f05e3fc9f2b9964` |
| 231 | `1cc8c542cc273059d6fb655faeca5173f8abed21735e393d1b50db8aee82708a` |
| 232 | `e635320f1020d8272d4647d0b1e4438c05c291b79b0a70c2450a684f42748099` |
| 233 | `3e1589f6d8df32a809e828c47983f6831d125b6fb20b95a3c3c3a53de1351b64` |
| 234 | `0641f414c2f440fc27fd0e2233abe7ccc729c945137733fc08f14037ebe4249a` |
| 235 | `9021391564bd377d2b8a5d4703a4a39a95ceb28f718602eca19f832694d1b4c8` |

## Obligation set and scenario disposition

`docs/testing/integration/ARCH-004-obligations.md` retains the established IDs and updates their content to the current Phase 07.01 architecture:

| Obligation | Architecture behavior | Required Task 236 scenarios | Result |
| --- | --- | --- | --- |
| `IT-ARCH-004-001` | Saved-diet identity, immutable replay, normalized submission/repair, owner polling | nominal, replay, replacement/deletion, concurrency, cancellation, malformed contract, identity | PASS |
| `IT-ARCH-004-002` | Repository input, solver output, canonical validation, distinct alternatives | nominal, concurrency, solver output, distinct alternative, malformed contract, partial recovery | PASS |
| `IT-ARCH-004-003` | Redis ownership, retry/finalization, cancellation, malformed delivery, loss recovery | concurrency, cancellation, queue loss/recovery, malformed contract, observability, degraded | PASS |
| `IT-ARCH-004-004` | Composed PostgreSQL/API/Redis/worker/native-CLP service | nominal, replay, replacement/deletion, concurrency, solver output, distinct alternative, identity, degraded | PASS |
| `IT-ARCH-004-005` | Whole-job deadline, child/worker cancellation, partial failure, safe finalization | cancellation, solver output, recovery, degraded | PASS |
| `IT-ARCH-004-006` | Strict frontend contracts, authoritative diet state, retry/identity lifecycle | nominal, replay, replacement/deletion, cancellation, malformed contract, identity, degraded | PASS |
| `IT-ARCH-004-007` | Bounded telemetry, accurate queue ages/outcomes, concurrency/capacity | replay, concurrency, queue loss/recovery, observability, degraded | PASS |
| `IT-ARCH-004-008` | Result TTL expiry and cross-identity isolation | identity, recovery, degraded | PASS |

Every required scenario named in the Task 236 row is covered by at least one obligation and passing integration fixture. Every obligation ID occurs in both the obligation document and executable test/capacity evidence.

## Maintained test symbols and traceability

No test body was changed. Adjacent comments were added or completed on these existing architecture-level tests:

### Rejected-review trace repairs

| Finding | Cited obligation evidence | Adjacent trace and declaration | SHA-256 |
| --- | --- | --- | --- |
| `F-236-01` | `docs/testing/integration/ARCH-004-obligations.md:374` | `backend/internal/worker/worker_integration_test.go:173` starts the adjacent `IT-ARCH-004-007`/`ARCH-004`/`DESIGN-004`/`DESIGN-014`/`SW-REQ-080`/`SW-REQ-082` trace; the cited test declaration is line 176. | `989ec5b09aa2e0934ba18bc4d357455004b82ad3450babb84237d6a434e82dd1` |
| `F-236-02` | `docs/testing/integration/ARCH-004-obligations.md:418` | `frontend/tests/phase07-browser-acceptance.spec.ts:450` starts the adjacent `IT-ARCH-004-008`/`ARCH-004`/`DESIGN-001`/`DESIGN-004`/`DESIGN-017`/`SW-REQ-006`/`SW-REQ-043`/`SW-REQ-080` trace; the cited test declaration is line 452. | `701eccab8cdd63a5f45c23db467ca9c8c02f597fb57ea6f96869a2f6fe526022` |
| `F-236-R01` | `docs/testing/integration/ARCH-004-obligations.md:129` | `backend/internal/app/task206_backend_integration_test.go:36` starts the adjacent `IT-ARCH-004-001`/`IT-ARCH-004-002`/`IT-ARCH-004-004`/`ARCH-004`/`DESIGN-004`/`SW-REQ-006`/`SW-REQ-021`/`SW-REQ-022`/`SW-REQ-023`/`SW-REQ-030` trace; the cited test declaration is line 40. | `cb0c2643b11b92da3d9436d84739f2d5b66ae25a39404ff83f914d872589ca0b` |
| `F-236-R02` | `docs/testing/integration/ARCH-004-obligations.md:273` | `backend/internal/worker/task210_swe5_integration_test.go:17` starts the adjacent `IT-ARCH-004-002`/`IT-ARCH-004-005`/`ARCH-004`/`DESIGN-004`/`SW-REQ-021`/`SW-REQ-022`/`SW-REQ-030` trace; the cited test declaration is line 19. | `176770344bc412ae0d925d74c76e3769d8ca04467cd7b58e155acba158e0b701` |
| `F-236-R03` | `docs/testing/integration/ARCH-004-obligations.md:419` | `frontend/tests/task233-frontend-gate.spec.ts:411` starts the adjacent `IT-ARCH-004-006`/`IT-ARCH-004-008`/`ARCH-004`/`DESIGN-001`/`DESIGN-004`/`DESIGN-017`/`SW-REQ-006`/`SW-REQ-043`/`SW-REQ-080` trace; the cited test declaration is line 414. | `77ede0fdc7dd4482b16316f4d020d1aa0db48f69132d0b39b114207de0afaa96` |

The Task 236 source remains `docs/implementation/02_TASK_LIST.md:243`, status `OPEN`, at SHA-256 `afea5492b9c526a1160f24e2cdf8a72e53133665267c9d0487ba3a575629ee3d`.

### Backend composed/API

- `app.TestTask232PostgresRedisAPIReplayRepairAndConcurrentUsers` — `IT-ARCH-004-001/002/003/004/005/007`.
- `httpapi.TestOptimizationHTTPDifferentUsersProceedIndependentlyAndSameKeyIsUserScoped` — `001/004`.
- `httpapi.TestOptimizationHTTPSubmissionHonorsRequestCancellation` — `001/005`.
- `httpapi.TestOptimizationHTTPPublishedAcknowledgementReplayHasNoCurrentStateSideEffects` — `001/004`.
- `httpapi.TestOptimizationHTTPUnpublishedRepairRevalidatesAndPublishesOnce` — `001`.
- `httpapi.TestOptimizationHTTPConcurrentPublishedRepairDoesNotStrandAdmission` — `001`.
- `httpapi.TestOptimizationHTTPRejectsMalformedPersistedAcknowledgementWithoutLocationFallback` — `001`.
- `httpapi.TestTask234ConcurrentSubmissionsReplayCleanupAndPollResponsiveness` — `007`.

### Backend optimization/queue/worker

- `optimization.TestPublicAlternativeGeneratorsBuildOneIndexAndProjectEachResultOnce` — `002`.
- `optimization.TestAlternativePipelineRejectsInvalidDuplicateBeforeStateMutation` — `002`.
- `optimization.TestAlternativePipelineAttemptExhaustionReturnsValidPartialResults` — `002`.
- `optimization.TestAlternativePipelineSnapshotIgnoresLaterCallerMutation` — `002`.
- `optimization.TestSolutionValidatorConcurrentMetricAndLiquidProjection` — `002`.
- `queue.TestJobQueueCancellationReachesProcessorAndLeavesDeliveryRecoverable` — `003/005`.
- `queue.TestTask224RejectsNonCanonicalJobIDsAndRemovesMalformedDeliveries` — `003`.
- `queue.TestTask225RequiresExplicitTerminalPublication` — `003`.
- `queue.TestTask225DistinctFinalizationAndZeroAckSemantics` — `003`.
- `queue.TestTask225AtomicDuplicateCleanupUnderRace` — `003`.
- `queue.TestTask225LiveManagerRecoversGroupAndDataLoss` — `003`.
- `queue.TestTask225LiveManagerRecoversAfterRedisRestart` — `003`.
- `queue.TestTask225AuthorizationAndConnectivityErrorsFailClosed` — `003`.
- `queue.TestTask234MixedAgesRetriesAndStreamRecoveryMatchFinalState` — `007`.
- `worker.TestOptimizationProcessorAppliesOneDeadlineAcrossAlternativeSolves` — `005`.
- `worker.TestOptimizationProcessorLeavesShutdownCancellationPendingForRetry` — `005`.
- `worker.TestTask234ProductionWorkerQueueTelemetryIsWiredAndPrivacySafe` — `007`.
- `worker.TestTask234SolverTimeoutAndAdmissionReleaseFailureKeepSafeFinalTelemetry` — `005/007`.
- `worker.TestOptimizationWorkerHeartbeatIsRefreshableAndRemovedOnStop` — `007`.

### Frontend/browser/capacity

- Task 233 browser tests for lost-response replay/authoritative replacement, malformed collection/poll contracts, queue ambiguity/timeout retry, remount/logout/account switch, and delayed identity cancellation — `001/005/006/008`.
- Daily Diet store tests for delete/load non-resurrection and reload/deletion/identity selection reconciliation — `006`.
- Optimization store tests for expiry/fresh retry, malformed polling rejection, and logout/account teardown — `006/008`.
- Phase 07 browser expired-result test for retryable expiry without stale result — `008`.
- `scripts/test_verify_optimization_capacity.py` and `scripts/verify-phase0701-observability-capacity.py` — `007`.

All comments include `ARCH-004`, applicable `DESIGN-*` static aspects, and `SW-REQ-*` IDs. The Task 206, Task 210, and Task 233 cited traces now name every obligation under which those tests are cited.

## SWE.5 checklist execution

The skill checklist was evaluated obligation by obligation against the obligation document and cited source tests. The full matrix and evidence notes are in `ARCH-004-obligations.md`.

| Mandatory checklist group | 001 | 002 | 003 | 004 | 005 | 006 | 007 | 008 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| ARCH and SW requirement trace; architectural behavior, not one method | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| Identified SUT; two or more units; interaction and data exchange | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| Real implementations practical; boundary-only doubles; not all mocked | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| Sequence/state/data/failure/recovery architecture behavior | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| Observable state/result/payload/side-effect evidence | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| Every obligation referenced by tests and every cited test references an obligation | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| No isolated validation/bounds/helper-only SWE.4 leakage | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |
| All implementations passing and no obligation uncovered | PASS | PASS | PASS | PASS | PASS | PASS | PASS | PASS |

Final sanity check: replacing every collaborator except one with mocks would remove the PostgreSQL, Redis, stream/pending/terminal, worker/native-CLP, HTTP, browser/store, or telemetry outcomes asserted by these fixtures. The tests would not still pass, so they are genuine SWE.5 evidence.

## Commands and exact results

All final commands ran against the current shared worktree with PostgreSQL and Redis started by `bash scripts/start-services.sh`.

| Command | Result |
| --- | --- |
| `bash scripts/start-services.sh` | PASS — existing `mealswapp-postgres-1` and `mealswapp-redis-1` containers running and ready. |
| Focused backend normal command from `ARCH-004-obligations.md` over `internal/app`, `httpapi`, `optimization`, `queue`, `worker`, and `observability` | PASS — all six packages; live app 6.278s, queue 2.199s. |
| Focused backend race command from `ARCH-004-obligations.md` | PASS on final rerun — all six packages; app 3.820s, worker 1.709s. |
| `go test -race ./internal/worker -run '^TestTask234ProductionWorkerQueueTelemetryIsWiredAndPrivacySafe$' -count=3 -v` | PASS — 3/3 consecutive isolated race runs. |
| Focused Bun command over both strict clients, both stores, and `OptimizationWorkflow` | PASS — 93 tests, 472 expectations, 0 failures. |
| `bunx playwright test tests/task233-frontend-gate.spec.ts --workers=2 --reporter=dot` | PASS — 14/14 desktop/mobile executions in 32.0s. |
| `python3 scripts/verify-phase0701-observability-capacity.py` | PASS — 10 Python tests and all selected normal/race Go fixtures; real Redis restart and child-process cleanup ran, with no required skip. |
| `go test ./internal/worker -run '^TestOptimizationWorkerHeartbeatIsRefreshableAndRemovedOnStop$' -count=1 -v` | PASS — focused heartbeat lifecycle test. |
| `go test -race ./internal/worker -run '^TestOptimizationWorkerHeartbeatIsRefreshableAndRemovedOnStop$' -count=1 -v` | PASS — focused heartbeat lifecycle race test. |
| `bunx playwright test tests/phase07-browser-acceptance.spec.ts --grep 'expired-result fixture presents the retryable expired state and no stale result' --workers=2 --reporter=dot` | PASS — 2/2 configured browser projects in 6.7s. |
| `go test ./internal/app -run '^TestTask206BackendIntegrationGate$' -count=1 -v` | PASS — cited Task 206 integration gate in 3.739s. |
| `go test ./internal/worker -run '^TestTask210WorkerPublishesPartialAlternativesOnLaterSolverFailure$' -count=1 -v` | PASS — cited Task 210 partial-alternative gate in 0.015s. |
| `bunx playwright test tests/task233-frontend-gate.spec.ts --grep 'remount resumes acknowledged polling, then logout and account change clear every prior-user artifact' --workers=2 --reporter=dot` | PASS — 2/2 configured browser projects in 12.1s. |
| `python3 scripts/validate-task-list.py` | PASS — 237 sequential tasks with ordered dependencies; Task 236 remains OPEN. |
| `python3 scripts/validate-traceability.py` | PASS — design-comment/file traceability valid. |
| `python3 -c 'import scripts.check as c; checked,total=c.validate_requirements(); ...'` | PASS — requirement traceability 91/91. |
| Scoped `validate_design_coverage()` assertion for `DESIGN-004` | PASS — 7/7 static aspects implemented. |
| `gofmt -l` over all maintained Go test files | PASS — no output. |
| `git diff --check -- <Task 236 surface>` | PASS — no whitespace errors. |
| Independent cited-obligation audit (`IT-ARCH-004-001` through `-008`) | PASS — 59/59 cited sources resolve with exact adjacent obligation traces. |

## Deviations and diagnostic history

1. **Repository source names:** the three generic filenames expected by the skill/user are absent. The repository equivalents listed above were used; this changes no verification intent.
2. **Generic obligation generator:** the default invocation failed with `required file not found: docs/implementation/tasks.md`. The repository-adapted invocation using `02_TASK_LIST.md`, `coder-prompt.md`, and `01_SOFT_ARCH_DESIGN.md` then failed with `could not parse any UNIT -> ARCH mappings from code design` because this repository models static aspects directly and has no `UNIT-* → ARCH-*` matrix. The skill template/checklist were therefore applied manually to the existing stable ARCH-004 obligation IDs. No generated JSON artifact was created.
3. **Initial focused race run:** every package except `internal/worker` passed; `TestTask234ProductionWorkerQueueTelemetryIsWiredAndPrivacySafe` reached its 3-second polling limit under the first combined race load. The unchanged test passed 3/3 isolated race runs and the identical full focused race command then passed all six packages. The complete observability/capacity normal/race gate also passed it. This is recorded as transient timing evidence, not hidden or treated as a product failure.
4. **Supplemental aggregate design-completeness probe:** an exploratory `validate_design_coverage()` assertion across every design reported unrelated planned aspects such as `DESIGN-001 ServiceWorker` and `DESIGN-015 BackupManager`. This is not one of the repository's two traceability validators and is outside Task 236. The scoped `DESIGN-004` result is 7/7 and passes.
5. **Expected degraded logs:** capacity verification printed Redis connection-refused lines while intentionally testing restart and bounded cleanup against unavailable endpoints. All assertions passed and no private data appeared.
6. **Coverage/aggregate gates:** Task 235 already executed and recorded the aggregate `scripts/check.py`, full coverage, security, browser, and quality gates. Task 236 did not rerun that entire gate; it reran the exact affected integration, race, browser, capacity, formatting, obligation, requirement, design, and task-list surfaces. No coverage deviation is introduced by comment/document-only Task 236 changes.

## Current SHA-256 fingerprints

### Task 236 obligation/test surface

| Path | SHA-256 |
| --- | --- |
| `docs/testing/integration/ARCH-004-obligations.md` | `a4739a69f68e1286e0db31c1bc0de6384913acb4e9c773292f53fc7932808b9d` |
| `backend/internal/app/task206_backend_integration_test.go` | `cb0c2643b11b92da3d9436d84739f2d5b66ae25a39404ff83f914d872589ca0b` |
| `backend/internal/app/task232_backend_regression_test.go` | `3fd6f01245e162e1f45840d56570c29d9d8f49c2d546b12c64cfd50fa8997076` |
| `backend/internal/httpapi/task222_optimization_submission_integration_test.go` | `87fcb2f12391378e12cdbe3d553b4d25a49e629fbeff5a4f0cf10cc9adcc97a8` |
| `backend/internal/httpapi/task234_observability_capacity_test.go` | `315d30cbc3adfca6d227c1848b157be376671fa953371a70812a47354b7f64c3` |
| `backend/internal/optimization/task220_pipeline_test.go` | `3a3b8798e7a752a555dd5b3a6cdc9aa509377ee0fe88a392af33f58eaeaa9d76` |
| `backend/internal/queue/job_queue_integration_test.go` | `8a72f388ab7a5537f269f608e625017b32932eacd1eb40c2c38660cb6717ff2f` |
| `backend/internal/queue/task225_queue_test.go` | `94356fbd74a110bac7efbc782011d71c195d5b43fe0926f971ed6b125f4c8ea5` |
| `backend/internal/queue/task234_regression_test.go` | `6c0d3b48933ec97b3bf3e2a99142a2cd6a6f0aa7e0382f87c294482d9264fbec` |
| `backend/internal/worker/optimization_processor_deadline_test.go` | `189fcfc88140f76a5913fd91bc1106106b47b34ba53153b662b4c8c5bc46dccd` |
| `backend/internal/worker/task210_swe5_integration_test.go` | `176770344bc412ae0d925d74c76e3769d8ca04467cd7b58e155acba158e0b701` |
| `backend/internal/worker/task234_regression_test.go` | `06e826f63dba8cc23e9a39fd49f760a370bf3f5cf7b373fc4c77701f4d965d56` |
| `backend/internal/worker/worker_integration_test.go` | `989ec5b09aa2e0934ba18bc4d357455004b82ad3450babb84237d6a434e82dd1` |
| `frontend/tests/task233-frontend-gate.spec.ts` | `77ede0fdc7dd4482b16316f4d020d1aa0db48f69132d0b39b114207de0afaa96` |
| `frontend/tests/phase07-browser-acceptance.spec.ts` | `701eccab8cdd63a5f45c23db467ca9c8c02f597fb57ea6f96869a2f6fe526022` |
| `frontend/src/lib/stores/daily-diet.test.ts` | `565c8e326dd70a16665c20efc8d17f95a554a7cfb70900b9f3a2a10f0f83e8f5` |
| `frontend/src/lib/stores/optimization.test.ts` | `639b0871f8f6e5474e226702000fe2746fc7a7c0746a570b224a12c4faf3a801` |
| `scripts/test_verify_optimization_capacity.py` | `c265b6ad7506082b54c8c82e920137ad71f67ddff02532c9fe45712e5eb424a0` |
| `scripts/verify-phase0701-observability-capacity.py` | `84813a0c52f7676ac003e9de228f7f8906f92947a1317e7fd193bc92f86554a6` |

### Authoritative source/checklist snapshots

| Path | SHA-256 |
| --- | --- |
| `docs/implementation/02_TASK_LIST.md` | `afea5492b9c526a1160f24e2cdf8a72e53133665267c9d0487ba3a575629ee3d` — read-only Task 236 snapshot |
| `docs/architecture/01_SOFT_ARCH_DESIGN.md` | `eb45a090af681f6dff6a44a0eee51c36719da60ad4a3e06f01d1adf083e0998c` |
| `docs/architecture/ARCH-004.md` | `bedcf45c79f50cfe24313345ac2ba664130ee44d7d2f35c7b1de0a06e0abe867` |
| `docs/design/DESIGN-004.md` | `45dd31e2afe1480ec54f540031ec45289f8287fbd0ef1a2d2cee86dea16a5474` |
| `docs/design/DESIGN-001.md` | `34d699ae93a8e5465199f3494ed41813c675f1cb3b9c1c6b6e611ba66c6142c7` |
| `docs/design/DESIGN-008.md` | `551880a70a3b42698e11632a471957bbecc6011900cf121c34f1ff3c3db18b87` |
| `docs/design/DESIGN-014.md` | `f9f6521d89e6d31306422017e07af5630ba4d8da56907174f3653ea0d72e9fe4` |
| `docs/design/DESIGN-017.md` | `5c92a6776af7bb9584c73098b4c53e9c53e182b7388c990a4fd42f043f44464c` |
| `docs/requirements/01_SOFT_REQ_SPEC.md` | `244749423b0bab26a0f25be4d2be8babfd78aa2d85f163739895e68f0c9e69a9` |
| `docs/implementation/reviews/task-236-review.md` | `13dc93984581526af26d98a9b827d069fb0acc84dd116e2b2c2db509c9a9d4bf` |
| `scripts/validate-traceability.py` | `78b62585204e3530027fe9194693503912c4b40cc74f950ca03842620e1589fd` |
| `scripts/validate-task-list.py` | `9fc5f34f548af84720a29adb22d5367e3a5541aa097dde74c77a11e3a39d811c` |
| skill `CHECKLIST.md` | `1f5393a352ed840e78c2541aa85ac056b400a9a03d04be726a4744666a58ca9f` |
| skill `templates/obligation.md` | `099e3b8269146ee0a7daa8f804890a275294ca5cd958b79b99020609c4cbcd07` |

This preparation document is intentionally not self-hashed because embedding its final digest would change that digest.

## Final Task 236 conclusion

`ARCH-004` has complete Phase 07.01 SWE.5 evidence for nominal, replay, replacement/deletion, concurrency, cancellation, solver-output, distinct-alternative, queue-loss/recovery, malformed-contract, identity, observability, and degraded collaborations. All eight obligations pass, all focused normal/race/frontend/browser/capacity tests pass, both repository traceability validations pass, and no task status or unrelated implementation was changed.
