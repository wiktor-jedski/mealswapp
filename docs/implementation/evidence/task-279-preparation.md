# Task 279 re-review repair evidence

prepared_at_utc: 2026-07-28T02:13:47Z
task_id: 279
task_status_observed: PREPARED
component: Phase 08.02 Isolated Real-Stack Administration E2E Harness
static_aspect: DESIGN-005 RepositoryInterfaces
baseline_ref: f9a646ffede5c8a63ad787a07eaba7efc0433677
baseline_confidence: HIGH
review_source: docs/implementation/evidence/task-279-review.md

## Outcome

The second re-review findings are repaired without changing the Task 279 row or
removing unrelated worktree changes.

- `stop_owned_process` now fails closed when the directly owned leader's live
  `/proc` start token cannot be read. It does not signal that process group,
  even if a group with the old numeric ID exists.
- A failed spawn publication followed by failed rollback raises both failures,
  retains the `OwnedProcess` in memory, and publishes `rollback_failed` state
  when possible. Tracking is removed only after checked rollback or
  `retire_process` confirms that the group is gone.
- `Harness.run` rechecks handled signals after final diagnostics, changes a
  previously successful result to failed, exports failed diagnostics, then ends
  teardown mode so any still-later signal interrupts rather than returning
  success.
- API/frontend startup treats a listener exit during readiness as a possible
  release-to-bind collision. It tears down the complete attempted stack with
  ownership checks, allocates two new held loopback reservations, and retries a
  maximum of three attempts. An adversarial test claims the released socket
  before child exec and proves the collision is detected; a separate contract
  test proves whole-stack retry and Vite proxy rebinding.
- `process_group_exists` ignores zombie-only groups and parses `/proc/*/stat`
  after the parenthesized command field.
- The full-gate browser failure was reproduced under repetition. Two races were
  identified: the test returned before detailed selected-item classifications
  loaded, and a stale 100 ms blur timer could close a newly refocused filter
  dropdown. The fixture now waits for the classification and retry response;
  `SubstitutionInputs` cancels pending close timers on refocus.

The prior report's statement that the latest exact full gate passed was false:
the independent re-review observed 308 browser passes, one failure, and five
skips. The current exact full gate was rerun after repair and passed with 309
browser passes and five expected skips.

## Baseline and confidence

- Repository: `/home/wiktor/Work/mealswapp`
- Branch: `multistep-phase-08`
- HEAD: `f9a646ffede5c8a63ad787a07eaba7efc0433677`
- Worktree baseline: dirty with existing Phase 08 changes; they were preserved.
- Authoritative task source: row 279 in
  `docs/implementation/02_TASK_LIST.md`.
- Dependencies observed by the re-review: Tasks 258, 259, 261, and 276 passed.
- Confidence: **HIGH**. Every finding has an adversarial regression test; the
  focused, stress, quick, exact full, real-stack, failure-injection, cleanup,
  traceability, static, coverage, and browser boundaries were available and
  executed locally.
- Task-list SHA-256 remained
  `e96e5bbf3902ff354cfa180b84d113b3cefd7e3a55d9f710d9b71d2491b52da1`.
  No task-list status edit was made by this repair.

## Exact changed paths and symbols

- `scripts/run-real-stack-e2e.py`
  - `ListenerExitedError`
  - `SignalController.end_teardown`
  - `process_group_exists`
  - `stop_owned_process`
  - `wait_http`
  - `Harness.start_process`
  - `Harness.run`
  - `Harness.retire_process`
  - `Harness.start_application_stack`
  - `Harness.execute`
  - `main` error handling
- `scripts/test_run_real_stack_e2e.py`
  - `test_spawn_rollback_failure_keeps_process_tracked_and_reports_both_failures`
  - `test_cleanup_never_kills_group_when_managed_leader_token_is_unavailable`
  - `test_signal_received_during_final_diagnostics_cannot_return_success`
  - `test_listener_exit_after_reservation_release_retries_whole_stack_with_new_ports`
  - `test_competitor_claim_between_release_and_exec_is_detected`
  - the descendant process-group regression now accepts reaped zombie-only
    groups as terminated
- `frontend/src/lib/components/SubstitutionInputs.svelte`
  - `openFilterOptions`
  - `scheduleFilterOptionsClose`
  - include/exclude focus, input, blur, and unmount timer handling
- `frontend/tests/dynamic-substitution-filters.spec.ts`
  - `addSelectedItem`
  - schema-invalid inventory retry synchronization
- `docs/implementation/evidence/task-279-preparation.md`
  - replaced stale findings, results, criteria, risks, and hashes with this
    observed report

No other path was intentionally changed for this repair.

## Re-review finding coverage

| Finding | Repair and regression evidence |
|---|---|
| Blocking: unavailable leader ownership token | `stop_owned_process` raises `SafetyError` before `killpg`; adversarial mock asserts no signal. |
| Blocking: suppressed spawn rollback failure | `Harness.start_process` raises a `BaseExceptionGroup`, retains tracking, and records `rollback_failed`; regression asserts both errors and retained ownership. |
| Important: late signal after result snapshot | `Harness.run` rechecks after export and before return; injected `TERM` during the passed export raises `InterruptedError`. |
| Important: release-before-bind port race | Real competitor claims the socket between release and exec and causes `ListenerExitedError`; bounded whole-stack retry uses fresh API/frontend ports and rewrites the Vite target. |
| Important: process-group cleanup | Current-token loss fails closed; zombie-only groups do not trigger unsafe retries; timeout descendants are reaped. |
| Important: full-gate browser flake | Reproduction was 115 passed/5 failed; classification readiness plus blur-timer cancellation produced 120/120 repeated passes and 309/309 runnable full-gate browser passes. |
| Important: stale preparation evidence | This report distinguishes the rejected full-gate result from the new successful run and records current hashes. |

## Task criteria coverage

| Criterion | Current evidence |
|---|---|
| Fail-closed process ownership and cleanup | Start tokens are validated before signaling. Missing leader identity never authorizes a process-group signal. Rollback failures remain tracked for later cleanup. |
| Independent API/frontend ports and Vite target | Two loopback reservations are distinct and held through publication. Native listeners then use bounded bind-and-readiness retry with a complete attempted-stack rollback and fresh proxy target. |
| Consistent handled-signal teardown | `INT`/`TERM`/`HUP` remain queued during teardown; a signal injected during final result export cannot return success; after teardown mode ends, a signal interrupts immediately. |
| Diagnostics and teardown ordering | Export failure cannot skip cleanup or replace the original execution failure; final result is recomputed for late signals. |
| Strict database/Redis safety retained | Exact generated database/comment run IDs, loopback target, environment guard, exact Redis label, and stale-age checks remain covered by focused tests. |
| Task 261 isolation retained | Contract test still proves no direct SQL or `node:child_process` promotion, and Vite uses the allocated API target. |
| Browser reproducibility | Formerly flaky spec passed 120 repetitions across desktop/mobile; exact full browser lane passed 309 with five expected skips. |
| Real-stack repeatability and failures | A concurrent pair and two later sequential runs passed. Injected assertion and API-readiness timeout exited 1 and cleaned. |
| Post-run residue | Age-bounded stale cleanup reported `resources=0`; PostgreSQL query returned no generated E2E database; Docker returned no run-labeled container. |
| Repository gates | Focused tests, quick gate, exact full gate, frontend build, Playwright stress, traceability, coverage contracts, vulnerability scan, and `git diff --check` passed. |

## Commands and exact observed results

All commands ran on 2026-07-28.

| Command/scenario | Result |
|---|---|
| `python3 -m unittest -v scripts/test_run_real_stack_e2e.py` | PASS, 28 tests. |
| `python3 -m py_compile scripts/run-real-stack-e2e.py scripts/test_run_real_stack_e2e.py` | PASS. |
| `cd frontend && bun test src/lib/components/SubstitutionInputs.test.ts` | PASS, 11 tests. |
| `cd frontend && bun run build` | PASS, 219 modules transformed. |
| Pre-fix stress: `MEALSWAPP_PLAYWRIGHT_REUSE_BUILD=1 MEALSWAPP_PLAYWRIGHT_WORKERS=4 bunx playwright test tests/dynamic-substitution-filters.spec.ts --repeat-each=20` | FAIL, 115 passed and 5 failed; all failures were missing filter options after focus. |
| Same Playwright stress command after repair | PASS, 120 tests in 1.1 minutes. |
| `python3 scripts/check.py --quick` | PASS: 28 harness tests, 536 frontend tests, 24 changed Playwright passes with 2 expected skips, changed backend packages, and all static/contract checks. |
| `python3 scripts/check.py` | PASS: 309 Playwright passes with 5 expected skips; browser lane 113.3 s; backend lane 378.2 s; package coverage 87.4%; Phase 08 coverage 4639/4980 (93.2%). |
| Two concurrently started `python3 scripts/run-real-stack-e2e.py --timeout-seconds 180` runs | PASS/PASS, exit 0/0. |
| Two subsequent sequential runs of the same real-stack command | PASS/PASS, 8.256 s and 8.330 s. |
| `--inject-assertion-failure` and `--inject-api-timeout` | Expected FAIL/FAIL, exit 1/1 (`CalledProcessError`, `TimeoutError`); owned cleanup completed. |
| `python3 scripts/run-real-stack-e2e.py --cleanup-stale --stale-age-seconds 900` | PASS, `resources=0`. |
| PostgreSQL generated-name residue query over loopback | PASS, zero rows. |
| Docker exact run-label residue query | PASS, zero containers. |
| `git diff --check` | PASS. |

The OpenAPI lint lane retained its existing redirect-only OAuth callback
warning and passed. The vulnerability lane reported no called
vulnerabilities.

## Current SHA-256 hashes

| Path | SHA-256 |
|---|---|
| `docs/implementation/02_TASK_LIST.md` | `e96e5bbf3902ff354cfa180b84d113b3cefd7e3a55d9f710d9b71d2491b52da1` |
| `scripts/run-real-stack-e2e.py` | `52647965aead4bf16f87e6c1ae4fc4a8c6a7a9c20af4b1ea45921089f8aa140d` |
| `scripts/test_run_real_stack_e2e.py` | `545501b661902524faadf19ae3f208d7a278fe2f1f78a261a24cce12be469031` |
| `frontend/src/lib/components/SubstitutionInputs.svelte` | `7ddf4f5a66ae76a5133bc487f7b25c877a65bab35a571a5836e9037b4b41c3ce` |
| `frontend/tests/dynamic-substitution-filters.spec.ts` | `8a62e7b9104459a98d08242e2268bc14fa6b8113eb5f11a07c03b992abe3449f` |
| `scripts/verify-task-261-ui.sh` | `1799f1d533203fc929cf69edeb4095b6d83d8aea10f2266ed001b564e058aafa` |
| `frontend/tests/task261-real-admin-flow.spec.ts` | `a1ed430a6511e04436b31417b86608b5174354d4219bd72d15bca94ed76c1210` |
| `frontend/playwright.real-stack.config.ts` | `11e3530338347f87788fdfb792d224971fab965dfd79b7a51ccf5e81dc230ead` |
| `frontend/vite.config.ts` | `49b59bf06e2a1ffb79851354cbd5344a8c6716ffd91125d793e65a9a81152332` |
| `docs/operations/real-stack-e2e.md` | `85268f77839f072866bbe98c0d2555a8d97fa2bc8f527adda14da9a7ac006c0c` |
| `docs/implementation/04_OPEN.md` | `f65d915e4cc012e4afef0616c91179c9eaac323f961d6b0012900ebafcc8fecb` |
| `scripts/check.py` | `d9ff7da8c7a5c4b5be223a2e9699bd7bf004c6f2be38531634cc950081d20d38` |
| `scripts/test_check_coverage.py` | `20dbbc53a9df76a9f19a4784fd7f01dccc4e5990717e46f4898f74282a6a0379` |

## Residual risks

- The Go API and Vite do not accept an inherited listening socket. The harness
  therefore cannot atomically transfer its reservation; it closes the
  release-to-bind window with bounded listener-exit detection, complete
  ownership-checked rollback, and fresh-port retry. A continuously hostile
  local process can exhaust all three attempts, in which case the run fails
  safely.
- Linux process ownership recovery depends on `/proc/<pid>/stat`. If the
  directly owned leader token is unavailable while its numeric process group
  still exists, cleanup intentionally fails closed and leaves marked state for
  operator inspection rather than risking an unrelated process.
- The development database retains the independent review's pre-existing Task
  261 fixture pollution. This repair neither targeted nor changed those rows.
- Persistent diagnostics remain deliberately sanitized and therefore trade raw
  request/body detail for the task's credential and PII non-retention
  requirement.
