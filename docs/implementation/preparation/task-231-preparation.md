# Task 231 Preparation Evidence

## Scope and authoritative contract

- Task: **231 — Phase 07.01 Optimization Retry and Identity Lifecycle**.
- Authoritative row: `docs/implementation/02_TASK_LIST.md:238`; its status remained `OPEN` throughout this work.
- Design owner: `docs/design/DESIGN-017.md`, `RetryManager`.
- Required behavior: explicit failure-code/stage retry actions using current input; safe disposal/remount and authenticated-identity lifecycle; one controller owner per shared store; bounded polling configuration; abortable, leak-free delays; and the complete task-row test matrix.
- Scope boundary: frontend optimization controller/runtime, the optimization workflow identity/current-input handoff, focused component tests, and this evidence. No task status, backend, API schema, generated client, design source, or unrelated Phase 07.01 behavior was changed.

The worktree already contained cumulative concurrent Phase 07.01 work, including Task 229 selected-diet changes and Task 230 optimization client/controller changes. Those contents were treated as the implementation baseline and preserved. Attribution below uses pre-edit hashes captured before Task 231, exact symbols, final hashes, and focused verification rather than the aggregate dirty-worktree diff.

This evidence was refreshed after repairing only the three findings in `docs/implementation/reviews/task-231-review.md`: F-231-01 authenticated teardown, F-231-02 live successor ownership handoff, and F-231-03 stale completed artifacts after secure key-generation failure. The accepted retry table, current-input comparison, polling bounds, abortable delay behavior, and concurrent Phase 07.01 work were preserved.

## Sources read

- `docs/implementation/02_TASK_LIST.md:238`: exact Task 231 description, dependency, and acceptance criteria.
- `docs/design/DESIGN-017.md`: `RetryManager` ownership of policy, timing, connectivity recovery, timeout state, and preserved client state.
- `docs/design/DESIGN-004.md` and `frontend/src/lib/api/generated.ts`: queued/processing/completed/failed/cancelled job variants and the four persisted terminal failure codes.
- `frontend/src/lib/api/error-message-mapper.ts`: approved optimization transport codes and retryability for validation, authentication, entitlement, not-found, expiry, rate limit, server, queue, and timeout outcomes.
- `frontend/src/lib/api/optimization-client.ts` and tests: strict acknowledgement/poll boundary, caller-owned in-memory key, abort signals, and safe errors.
- `frontend/src/lib/stores/optimization.ts` and tests: current Task 230 controller behavior and existing ambiguity/key lifecycle.
- `frontend/src/lib/components/OptimizationWorkflow.svelte`, `DailyDietControls.svelte`, their tests, and browser optimization fixtures: form input, retry/new button visibility, selected diet, authenticated user, mount/disposal, and terminal presentation.
- `frontend/src/lib/stores/auth-session.ts` and `SearchShell.svelte`: authenticated `userId`, logout/account transitions, and protected Daily Diet clearing.
- `docs/implementation/preparation/task-221-preparation.md`: terminal code vocabulary and safe projection evidence.
- `docs/implementation/preparation/task-227-preparation.md`: shared runtime-safe error mapping evidence.
- `docs/implementation/preparation/task-229-preparation.md`: authoritative selected Daily Diet identity and concurrent component baseline.
- `docs/implementation/preparation/task-230-preparation.md` and `docs/implementation/reviews/task-230-review.md`: strict client, exact key ownership, ambiguity, and prior hashes/tests.

## Explicit retry policy

`optimizationRetryAction(stage, code, retryable)` is the single policy boundary. `OptimizationController.retry(currentRequest)` then compares the current request with the preserved request snapshot. Any tolerance or ordered exclusion change converts an otherwise reusable action into `new_submission`, with a fresh idempotency key and the current request.

| Outcome | Policy action | Job/key/request behavior | Retry button |
|---|---|---|---|
| Retryable network, queue, timeout, rate-limit, server, dependency, or unknown failure before acknowledgement | `replay_submission` | Exact request and exact key are replayed because acceptance is ambiguous | Visible |
| Retryable transport/queue/timeout failure after acknowledgement | `poll_job` | Exact acknowledged job is polled; no submission and no new key | Visible |
| `result_expired`, `optimization_not_found`, or `not_found` | `new_submission` | Fresh job and key; current request is submitted | Visible |
| Terminal `solver_timeout`, `worker_crash`, or `cancelled` | `new_submission` | Fresh job and key; current request is submitted | Visible |
| Terminal `solver_infeasible` | `new_submission` | Never exact-replayed; a deliberate retry uses a fresh key and current/wider tolerance | Visible |
| Terminal `failed_validation` | `none` | No replay; user must correct the underlying saved-diet input | Hidden |
| Submission validation error or any non-retryable unclassified operation error | `none` | No replay | Hidden |
| `unauthorized`, `session_expired`, or `entitlement_denied` at any stage | `none` | No protected retry under the invalid identity/entitlement | Hidden |
| Completed result | internal fresh-submission intent only | Dedicated “Generate fresh alternatives” action creates a fresh key | Retry hidden; fresh action visible |

Repeated submit clicks during `submitting`, `queued`, or `processing` are ignored. Exact replay is allowed only while the original key still exists before acknowledgement. Once acknowledged, retry can only poll that job or create a deliberately fresh submission.

## Shared ownership and lifecycle

- `runtimes: WeakMap<Writable<OptimizationState>, SharedOptimizationRuntime>` gives every writable store one memory-only runtime containing identity, selected diet, pending request/key, exact retry action, owner token, live controller registrations, operation generation, and active abort controller.
- `claim` permits only one live controller to mutate a shared store. A second controller cannot submit, poll, reset, or overwrite state while the owner remains mounted.
- `cancelOperation` increments the shared generation and aborts the shared controller, so late submission/poll completion fails `isCurrent` and cannot commit stale state.
- `dispose` unregisters the old owner, preserves the approved submission or acknowledged-job remount state, and hands ownership to the first still-mounted controller. That successor reapplies its requested identity/diet scope and resumes the exact pending submission or acknowledged job once.
- `resume` records a mounted controller's desired identity/diet even while another owner is active. On direct remount or ownership handoff it exact-replays an interrupted pre-acknowledgement submission or resumes polling the same queued/processing job.
- `setIdentity` aborts work and clears pending key/request, job, result, error, retry action, and selected-diet scope when logout or account identity changes.
- `clearOptimizationIdentity` is the auth-boundary teardown used by `DailyDietControls`; it aborts active work, invalidates late completions, clears all registered scopes, pending private key/request, retry action, job/result/error state, and selected diet before the authenticated workflow is removed or an account is replaced.
- `setDiet` performs the equivalent reset for a diet-scope change.
- Every deliberate fresh intent clears the prior operation artifacts before secure key allocation. Key-generation failure therefore fails closed with no stale job, DTO, alternatives, or reusable retry material.
- `OptimizationWorkflow` receives `identityId={userId}`, applies identity before diet, calls `resume()` after scope configuration, passes current `activeRequest` to `retry`, and disposes on teardown; `DailyDietControls` invokes authenticated teardown before logout/account replacement removes or re-scopes it.

All pending keys remain closure/module-memory-only. They are absent from `OptimizationState`, DOM output, browser storage, and preparation evidence values.

## Polling bounds and settlement

- Delay schedules must be non-empty; every delay must be finite, non-negative, and at most `60,000 ms`.
- `maxPolls` must be an integer in `1..1,000`.
- The expanded delay schedule must be a safe integer and at most `600,000 ms` total. The production defaults remain 60 polls with capped delays `[500, 1000, 2000, 4000, 8000, 10000]`, totaling `565,500 ms`.
- `pollJob` performs exactly `maxPolls` polls and reuses the final configured delay after the schedule is exhausted.
- `waitForOptimizationPoll` checks pre-abort, owns one timer and one abort listener, uses an exactly-once settlement guard, clears the timer, and removes the listener on both timer and abort settlement.

## Exact symbols and tests

| File | Exact symbols / tests | Task 231 evidence |
|---|---|---|
| `frontend/src/lib/stores/optimization.ts` | `OptimizationControllerRegistration`, `clearOptimizationIdentity`, `OptimizationRetryAction`, `OptimizationFailureStage`, `SharedOptimizationRuntime`, `optimizationRetryAction`, `createOptimizationController`, `claim`, `setIdentity`, `setDiet`, `resetScope`, `resume`, `resumeOwned`, `submit`, `retry`, `submitFresh`, `runSubmission`, `pollExistingJob`, `pollJob`, `beginOperation`, `cancelOperation`, `isCurrent`, `setOperationFailure`, `dispose`, controller registration/handoff, `runtimeFor`, `sameRequest`, `retryMode`, `validatePollingConfiguration`, `waitForOptimizationPoll` | Auth-boundary destruction of protected/private artifacts, live successor handoff, fresh-intent atomic clearing, explicit code/stage policy, exact replay/repoll/rotation, shared generation/abort lifecycle, validated bounds, and listener-safe settlement. |
| `frontend/src/lib/stores/optimization.test.ts` | 31 tests, including focused regressions at lines 211, 244, and 801 | Existing strict-client/key tests plus direct-fresh and policy-retry completed-result secure-key failure cleanup, pre-ack auth teardown/private-key destruction, mounted successor handoff, exact policy, both ambiguity boundaries, repeated outage, bounded polling, queue outcomes, all terminal codes, edited input, remount, logout/account switch, controller race, invalid config, and timer/listener settlement. |
| `frontend/src/lib/components/OptimizationWorkflow.svelte` | `identityId`, scope `$effect`, `controller.setIdentity`, `controller.setDiet`, `controller.resume`, `retryOptimization` | Authenticated identity reaches the controller, remount resumes deliberately, and retry receives current form input. |
| `frontend/src/lib/components/OptimizationWorkflow.test.ts` | 6 tests at lines 11-75 | Lifecycle wiring and policy-approved retry/fresh-button visibility are source-verified alongside existing form/progress/result/accessibility checks. |
| `frontend/src/lib/components/DailyDietControls.svelte` | `clearOptimizationIdentity`, authenticated identity `$effect`, `OptimizationWorkflow identityId={userId}` | Logout/account replacement clears shared optimization scope before workflow removal; the authenticated user owning Daily Diet data also owns optimization runtime state. |
| `frontend/src/lib/components/DailyDietControls.test.ts` | 3 tests, including `logout and account teardown clear shared optimization identity before workflow removal` | Verifies selected-diet/authenticated-identity handoff and the concrete auth-boundary reset wiring. |

## Acceptance matrix

| Task-row criterion | Result | Focused evidence |
|---|---|---|
| Ambiguity before/after acknowledgement | PASS | Exact key/request replay before acknowledgement; exact job-only repoll after acknowledgement. |
| Poll timeout and stable bounded schedule | PASS | Four configured polls produce `[5, 10, 10, 10]`, then `optimization_poll_timeout` with job reuse. |
| Queue outage | PASS | Pre-ack queue outage reuses one key; post-ack queue outage performs no new submission. |
| Expiry/not-found | PASS | Both rotate from job/key 1 to fresh job/key 2. |
| Validation/infeasible/auth/entitlement/cancellation/worker failure | PASS | Table-driven operation and terminal tests assert `none` versus `new_submission` and exact key sequences. |
| Edited tolerance/exclusions | PASS | Both pre-ack replay and post-ack repoll become fresh key/current-request submissions. |
| Repeated clicks | PASS | Concurrent submit test observes one API call; deliberate later intent rotates key. |
| Correct retry/new button visibility | PASS | Policy states drive `retryMode`; component condition hides retry for `none` and completed, while completed exposes fresh generation. Browser timeout, infeasible, expiry, anonymous, and free fixtures pass on desktop/mobile. |
| Submit/queued/processing unmount/remount | PASS | Submission remount exact-replays the same key; queued and processing remounts poll the same job with one submission. |
| Logout/account switch | PASS | The actual parent auth boundary invokes `clearOptimizationIdentity`; active submission/poll aborts, job, DTO, alternatives, failure, pending private key/request, retry action, registered scopes, and selected scope reset; stale retry/resume performs no I/O. |
| Two controllers cannot race or wedge | PASS | First owner performs one submission; second controller cannot mutate concurrently, is retained as a live successor, then receives ownership and polls the same acknowledged job exactly once after disposal. |
| Secure key failure over an existing result | PASS | Direct fresh generation and policy-driven retry whose second key allocation fails leave `jobId`, job DTO, alternatives, pending submission, and retry action cleared while exposing only the safe `secure_random_unavailable` failure. |
| Invalid polling configuration | PASS | Empty, negative, NaN, infinity, over-minute delay, zero/fractional/over-1,000 polls, and over-ten-minute total all throw `RangeError`. |
| Fake timer/listener settlement | PASS | Exact max polls/delays, one listener removal per timer/abort path, one timer clear per path, repeated callback no-op, and pre-abort no-listener behavior. |

## Verification commands

| Command | Result |
|---|---|
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/stores/optimization.test.ts src/lib/components/OptimizationWorkflow.test.ts src/lib/components/DailyDietControls.test.ts` | PASS — 40 tests, 190 expectations. |
| Same focused command with `bun test --coverage` | PASS — 40 tests / 190 expectations; `optimization.ts` 100.00% lines and 98.00% functions. |
| `cd frontend && ... bun run typecheck` | PASS — no TypeScript errors. |
| `cd frontend && ... bun run build` | PASS — Vite production build, 205 modules transformed. |
| `cd frontend && ... bun test --coverage` | PASS — 436 tests / 1,985 expectations; 94.01% functions and 94.86% lines overall; `optimization.ts` is 100.00% lines and 98.00% functions. Existing phase-wide coverage exceptions are unchanged. |
| `cd frontend && ... bunx playwright test tests/optimization-workflow.spec.ts tests/phase07-browser-acceptance.spec.ts --reporter=line` | PASS — 18 desktop/mobile Chromium tests. |
| `python3 scripts/validate-task-list.py` | PASS — 237 sequential tasks with ordered dependencies; Task 231 remains `OPEN`. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `git diff --check -- <Task 231 frontend files>` | PASS. |
| `python3 scripts/check.py` | Reached and passed traceability, task-list, Go Doc, OpenAPI validity (one explicitly ignored existing OAuth 302 warning), verifier unit tests, Go vet, vulnerability scan, focused backend checks, and local-stack verification. It stopped in the pre-existing/concurrent Phase 02 UAT migration cycle because `000021_mutation_idempotency.down.sql` attempted to drop absent `mutation_idempotency_keys`; this database/migration-state failure is outside frontend Task 231 and occurred before aggregate frontend checks, which passed separately above. |

## Staleness and content fingerprints

### Pre-edit hashes captured before Task 231

| File | SHA-256 |
|---|---|
| `docs/implementation/02_TASK_LIST.md` | `3bdabe886facb2b96875489dcce7186d13acecc0a1582ff8e2fc8cc1dff62ebf` |
| `docs/design/DESIGN-017.md` | `5c92a6776af7bb9584c73098b4c53e9c53e182b7388c990a4fd42f043f44464c` |
| `frontend/src/lib/stores/optimization.ts` | `a2e959c819daa0a0a1d1cf685e13c36926bcf24d9e55786205a3b46c3301019e` |
| `frontend/src/lib/stores/optimization.test.ts` | `d1c25017a9a48f1fb576b549f20a1b8c6e46d5a9854a7c4d8758004a9f9a8efb` |
| `frontend/src/lib/components/OptimizationWorkflow.svelte` | `620e825cd23e258fee69ccb42899e00c01f2dc7a53df5d5b8e3d9cc3c6f00b33` |
| `frontend/src/lib/components/OptimizationWorkflow.test.ts` | `022b8e15728f1808c7397ee2ffd9b31f9c56d8af0915c28ab6b3da1ceb87a28d` |
| `frontend/src/lib/stores/auth-session.ts` | `97944edf13db85c71873e0dcd1a93a5a62335df1f26e3bce7f04995341be1323` |

### Final audited hashes

| File | SHA-256 |
|---|---|
| `docs/implementation/02_TASK_LIST.md` | `3bdabe886facb2b96875489dcce7186d13acecc0a1582ff8e2fc8cc1dff62ebf` |
| `docs/design/DESIGN-017.md` | `5c92a6776af7bb9584c73098b4c53e9c53e182b7388c990a4fd42f043f44464c` |
| `frontend/src/lib/stores/optimization.ts` | `766d11e87663108381dae36dfc9cc4705d6b73e8d46c531933f101f25083f4bc` |
| `frontend/src/lib/stores/optimization.test.ts` | `db3e6ce541bf5e83f726281fd8d9d4ee9eb62ba58fc8e4caa07461addf761800` |
| `frontend/src/lib/components/OptimizationWorkflow.svelte` | `c8d7428c9280ec5e5379f006bfb1c6a5c6bb4184915ee64d32c5c2952e0df82c` |
| `frontend/src/lib/components/OptimizationWorkflow.test.ts` | `0682288b529d27bd394c1ffb9bfb6c94fef04bc3b96334fd17dd9b4be61289a3` |
| `frontend/src/lib/components/DailyDietControls.svelte` | `a3a4a6111fa6a8b5a6dff97fb27e8e68f1f7ce5df567f91c7999cbdecd20a98d` |
| `frontend/src/lib/components/DailyDietControls.test.ts` | `b1db99593ac911b4e41682bbf51b399883212995967614a28139ad1c3cb6ce28` |
| `frontend/src/lib/stores/auth-session.ts` | `97944edf13db85c71873e0dcd1a93a5a62335df1f26e3bce7f04995341be1323` |

The identical task-list and DESIGN-017 hashes prove no status/design edit. The unchanged auth-store hash proves Task 231 consumes the existing identity projection without changing authentication behavior. No unrelated working-tree change was reverted, staged, or rewritten.
