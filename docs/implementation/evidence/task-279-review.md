# Task 279 Final Independent Review Evidence

review_decision: PASSED
decision: PASSED
task_id: 279
component: Phase 08.02 Isolated Real-Stack Administration E2E Harness
static_aspect: DESIGN-005: RepositoryInterfaces
input_status: PREPARED
reviewed_at_utc: 2026-07-28T02:31:34Z
review_agent: independent-final-re-reviewer
evidence_file: docs/implementation/evidence/task-279-review.md
baseline_ref: f9a646ffede5c8a63ad787a07eaba7efc0433677
baseline_confidence: HIGH
code_review_skill_invoked: true
pre_review_gates_passed: true
inventory_source_count: 100
audited_symbol_count: 100
inventory_complete: true
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
blocking_findings: 0
important_findings: 0
optional_findings: 0

## 1. Task Source

Authoritative source is row 279 in `docs/implementation/02_TASK_LIST.md`:
“Phase 08.02 Isolated Real-Stack Administration E2E Harness”, DESIGN-005
`RepositoryInterfaces`, status `PREPARED`, dependencies 258, 259, 261, 276,
and 278 observed PASSED. Task 280 remains OPEN. The updated preparation report
was read from `docs/implementation/evidence/task-279-preparation.md` and
challenged against current source, current hashes, live resources, focused
tests, browser gates, and the full repository gate.

The mandatory review checklist at
`/home/wiktor/.agents/skills/phase-orchestrator/templates/review_checklist.md`
was read completely. `code-review-skill` was invoked exactly once, with its
Python, Svelte, Go, and security guidance applied. No implementation file or
task-list file was edited by this reviewer.

## 2. Pre-Review Gates

pre_review_gates_passed: true

- The task row is still `PREPARED`; its SHA-256 is recorded below.
- The repaired preparation evidence was independently compared with the prior
  rejected review and current source; stale claims were not accepted without
  fresh execution.
- The worktree is intentionally dirty with broader Phase 08 work. Its existing
  changes were preserved and Task 279 scope was separated by path and symbol.
- Focused Python, frontend, build, traceability, task-list, static, quick, full,
  real-stack, failure, signal, stale-recovery, residue, and diff checks ran.

## 3. Review Baseline and Change Surface

The baseline is commit `f9a646ffede5c8a63ad787a07eaba7efc0433677` on branch
`multistep-phase-08`. Task-owned implementation surfaces are the reusable Python
harness, its unit/contract tests, the shell compatibility caller, the managed
Playwright config and Task 261 flow, the configurable Vite proxy, the repaired
dynamic-filter browser synchronization/component timer logic, gate registration
and coverage contract evidence, DESIGN-005/operations documentation, and the
updated preparation evidence. Related backend/bootstrap code is existing Phase
08 dependency work and was exercised by the real stack; it was not reclassified
as Task 279 implementation.

The final repair changed the previously rejected boundaries as follows:

- `stop_owned_process` refuses to signal when the directly owned leader token is
  unavailable, even when the old numeric process-group ID exists.
- `start_process` retains the published owner and records `rollback_failed`
  while reporting both publication and rollback failures through an exception
  group; it removes tracking only after checked rollback.
- `Harness.run` rechecks signals after final diagnostics and before returning,
  exports a failed result for a late signal, and only then ends teardown mode.
- Listener exit during the release-to-bind window causes complete attempted-stack
  retirement, fresh held ports, bounded retry, and a rewritten Vite target.
- The browser flake repair waits for selected classifications/retry responses and
  cancels delayed filter-close timers when the input is refocused.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence | Result |
|---:|---|---|---|
| 1 | Reusable harness owns one isolated complete stack | Source audit and ordinary runs | PASS |
| 2 | Strict opaque database name and matching ownership comment | Regex/source/unit/live catalog checks | PASS |
| 3 | Loopback PostgreSQL maintenance target only | Unsafe target matrix and live runs | PASS |
| 4 | Migrations target the generated database without development seeds | `Harness.execute` source and live runs | PASS |
| 5 | Run-labeled disposable Redis only | Docker label/loopback/readiness tests and live runs | PASS |
| 6 | Independent API/frontend ports and configurable Vite target | Held reservation, competitor, retry, config, and browser tests | PASS |
| 7 | Task 276 administrator bootstrap rather than direct SQL promotion | Task 261 contract and live browser flow | PASS |
| 8 | Created IDs are retained for assertions | Fixture state and Playwright response-ID assertions | PASS |
| 9 | Sanitized artifacts survive teardown | Artifact test and persistent scan | PASS |
| 10 | Teardown covers success, assertion failure, timeout, cancellation, and handled signals | Live injections and INT/TERM/HUP probes | PASS |
| 11 | SIGKILL recovery is explicit, age-bounded, and idempotent | KILL plus mature stale cleanup/repeat | PASS |
| 12 | Production and unsafe targets fail before external boundaries | Production stale and unsafe matrix | PASS |
| 13 | Database deletion requires exact generated name and exact live comment | Unit and source audit | PASS |
| 14 | Redis cleanup requires exact run label | Unit and live residue checks | PASS |
| 15 | No broad truncation, prefix cleanup, or `FLUSHALL` | Repository search/source audit | PASS |
| 16 | Process ownership tokens fail closed | Missing/reused token tests and source audit | PASS |
| 17 | Process groups are owned and descendants are reaped on timeout | `start_new_session`, timeout descendant test, live cleanup | PASS |
| 18 | Spawn gate prevents execution before state publication | Gate regression test | PASS |
| 19 | Spawn publication/rollback failure preserves ownership state | Exception-group regression test | PASS |
| 20 | Diagnostics failure cannot bypass teardown or replace original failure | Diagnostics regression test | PASS |
| 21 | Signals arriving during teardown cannot return success | Late-final-diagnostics regression and live signals | PASS |
| 22 | Port release-to-bind collisions are detected and retried | Adversarial competitor and whole-stack retry tests | PASS |
| 23 | Task 261 has no direct SQL or child-process promotion | Contract test and source | PASS |
| 24 | Browser flake is repaired and full browser lane passes | 120/120 dynamic-filter stress and 309/309 runnable full lane | PASS |
| 25 | Repository, security, traceability, coverage, and diff gates pass | Quick/full gate outputs and validators | PASS |

## 5. Changed-Symbol Inventory

inventory_source_count: 100

| # | Symbol or surface | File | Boundary/consumer |
|---:|---|---|---|
| 1 | SafetyError | scripts/run-real-stack-e2e.py | safety failures |
| 2 | ListenerExitedError | scripts/run-real-stack-e2e.py | port-race retry |
| 3 | PostgresTarget | scripts/run-real-stack-e2e.py | database boundary |
| 4 | PostgresTarget.parse | scripts/run-real-stack-e2e.py | URL validation |
| 5 | PostgresTarget.command | scripts/run-real-stack-e2e.py | psql argv |
| 6 | PostgresTarget.environment | scripts/run-real-stack-e2e.py | child secrets |
| 7 | PostgresTarget.database_url | scripts/run-real-stack-e2e.py | generated app URL |
| 8 | OwnedProcess | scripts/run-real-stack-e2e.py | PID/token state |
| 9 | PortReservation | scripts/run-real-stack-e2e.py | held loopback socket |
| 10 | PortReservation.release | scripts/run-real-stack-e2e.py | reservation lifecycle |
| 11 | SignalController | scripts/run-real-stack-e2e.py | handled signals |
| 12 | SignalController.handle | scripts/run-real-stack-e2e.py | interrupt/queue |
| 13 | SignalController.begin_teardown | scripts/run-real-stack-e2e.py | teardown mode |
| 14 | SignalController.end_teardown | scripts/run-real-stack-e2e.py | post-cleanup signal mode |
| 15 | validate_run_id | scripts/run-real-stack-e2e.py | opaque ID guard |
| 16 | database_name | scripts/run-real-stack-e2e.py | exact DB name |
| 17 | ownership_comment | scripts/run-real-stack-e2e.py | exact DB marker |
| 18 | validate_database_name | scripts/run-real-stack-e2e.py | DB target guard |
| 19 | validate_environment | scripts/run-real-stack-e2e.py | development guard |
| 20 | validate_stale_age | scripts/run-real-stack-e2e.py | recovery age guard |
| 21 | run_command | scripts/run-real-stack-e2e.py | grouped subprocess boundary |
| 22 | process_group_exists | scripts/run-real-stack-e2e.py | group/zombie check |
| 23 | rollback_spawned_process | scripts/run-real-stack-e2e.py | spawn rollback |
| 24 | psql | scripts/run-real-stack-e2e.py | PostgreSQL command boundary |
| 25 | create_database | scripts/run-real-stack-e2e.py | DB create/comment |
| 26 | database_comment | scripts/run-real-stack-e2e.py | live ownership marker |
| 27 | drop_owned_database | scripts/run-real-stack-e2e.py | exact DB deletion |
| 28 | ownership_comment_run | scripts/run-real-stack-e2e.py | marker parser |
| 29 | parse_ownership_comment | scripts/run-real-stack-e2e.py | marker syntax |
| 30 | validate_ownership_comment | scripts/run-real-stack-e2e.py | stale state validation |
| 31 | docker_inspect | scripts/run-real-stack-e2e.py | Docker boundary |
| 32 | inspect_labels | scripts/run-real-stack-e2e.py | Redis ownership |
| 33 | remove_owned_redis | scripts/run-real-stack-e2e.py | exact Redis deletion |
| 34 | reserve_ports | scripts/run-real-stack-e2e.py | paired allocation |
| 35 | reserve_port | scripts/run-real-stack-e2e.py | timeout injection |
| 36 | process_start_token | scripts/run-real-stack-e2e.py | Linux PID identity |
| 37 | validate_process_start_token | scripts/run-real-stack-e2e.py | fail-closed token guard |
| 38 | stop_process | scripts/run-real-stack-e2e.py | stale process cleanup |
| 39 | stop_owned_process | scripts/run-real-stack-e2e.py | live process cleanup |
| 40 | wait_http | scripts/run-real-stack-e2e.py | readiness/exit race |
| 41 | request_json | scripts/run-real-stack-e2e.py | fixture API boundary |
| 42 | register_fixture | scripts/run-real-stack-e2e.py | registration/IDs |
| 43 | data_field | scripts/run-real-stack-e2e.py | response projection |
| 44 | safe_diagnostics | scripts/run-real-stack-e2e.py | redaction allowlist |
| 45 | Harness | scripts/run-real-stack-e2e.py | lifecycle owner |
| 46 | Harness.__init__ | scripts/run-real-stack-e2e.py | resource identity |
| 47 | Harness.write_state | scripts/run-real-stack-e2e.py | atomic state publication |
| 48 | Harness.start_process | scripts/run-real-stack-e2e.py | gated managed spawn |
| 49 | Harness.run | scripts/run-real-stack-e2e.py | unconditional teardown |
| 50 | Harness.retire_process | scripts/run-real-stack-e2e.py | retry retirement |
| 51 | Harness.start_application_stack | scripts/run-real-stack-e2e.py | listener retry |
| 52 | Harness.execute | scripts/run-real-stack-e2e.py | stack orchestration |
| 53 | Harness.application_environment | scripts/run-real-stack-e2e.py | per-run config |
| 54 | Harness.start_redis | scripts/run-real-stack-e2e.py | container/readiness |
| 55 | Harness.export_diagnostics | scripts/run-real-stack-e2e.py | persistent artifacts |
| 56 | Harness.cleanup | scripts/run-real-stack-e2e.py | ownership teardown |
| 57 | cleanup_stale | scripts/run-real-stack-e2e.py | killed-run recovery |
| 58 | remove_owned_raw_directory | scripts/run-real-stack-e2e.py | exact temp cleanup |
| 59 | write_sanitized_png | scripts/run-real-stack-e2e.py | screenshot sanitizer |
| 60 | write_sanitized_png.chunk | scripts/run-real-stack-e2e.py | PNG encoding |
| 61 | install_signal_handlers | scripts/run-real-stack-e2e.py | OS signal setup |
| 62 | restore_signal_handlers | scripts/run-real-stack-e2e.py | OS signal restore |
| 63 | parse_args | scripts/run-real-stack-e2e.py | CLI mode guard |
| 64 | main | scripts/run-real-stack-e2e.py | operator entrypoint |
| 65 | SafetyValidationTests | scripts/test_run_real_stack_e2e.py | safety test group |
| 66 | DiagnosticsTests | scripts/test_run_real_stack_e2e.py | lifecycle test group |
| 67 | IntegrationContractTests | scripts/test_run_real_stack_e2e.py | integration contracts |
| 68 | stale/unsafe target test matrix | scripts/test_run_real_stack_e2e.py | pre-boundary rejection |
| 69 | exact DB/comment test matrix | scripts/test_run_real_stack_e2e.py | deletion/create guards |
| 70 | exact Redis label test | scripts/test_run_real_stack_e2e.py | ownership guard |
| 71 | diagnostics failure test | scripts/test_run_real_stack_e2e.py | cleanup guarantee |
| 72 | diagnostic allowlist test | scripts/test_run_real_stack_e2e.py | PII/secret exclusion |
| 73 | token absence/reuse tests | scripts/test_run_real_stack_e2e.py | no unsafe signal |
| 74 | spawn publication rollback test | scripts/test_run_real_stack_e2e.py | exact child rollback |
| 75 | rollback-failure state test | scripts/test_run_real_stack_e2e.py | retained ownership |
| 76 | spawn gate test | scripts/test_run_real_stack_e2e.py | pre-publication isolation |
| 77 | held-port publication test | scripts/test_run_real_stack_e2e.py | reservation ordering |
| 78 | exited-leader fail-closed test | scripts/test_run_real_stack_e2e.py | process-group safety |
| 79 | late-signal final-diagnostics test | scripts/test_run_real_stack_e2e.py | no false success |
| 80 | timeout descendant test | scripts/test_run_real_stack_e2e.py | process-group reaping |
| 81 | paired reservation collision tests | scripts/test_run_real_stack_e2e.py | port safety |
| 82 | listener-exit whole-stack retry test | scripts/test_run_real_stack_e2e.py | fresh ports/proxy |
| 83 | release-to-exec competitor test | scripts/test_run_real_stack_e2e.py | TOCTOU detection |
| 84 | Task 261 boundary contract | scripts/test_run_real_stack_e2e.py | no SQL/child promotion |
| 85 | Redis readiness contract | scripts/test_run_real_stack_e2e.py | exact container |
| 86 | shell compatibility caller | scripts/verify-task-261-ui.sh | operator forwarding |
| 87 | Task 261 real-admin flow | frontend/tests/task261-real-admin-flow.spec.ts | isolated browser behavior |
| 88 | requiredFixture | frontend/tests/task261-real-admin-flow.spec.ts | fixture env guard |
| 89 | responseID | frontend/tests/task261-real-admin-flow.spec.ts | created-ID assertion |
| 90 | createPrivateItemFixture | frontend/tests/task261-real-admin-flow.spec.ts | API fixture creation |
| 91 | managedByHarness/baseURL | frontend/playwright.real-stack.config.ts | managed server target |
| 92 | apiTarget | frontend/vite.config.ts | configurable proxy |
| 93 | Vite proxy export | frontend/vite.config.ts | API routing |
| 94 | openFilterOptions | frontend/src/lib/components/SubstitutionInputs.svelte | timer cancellation |
| 95 | scheduleFilterOptionsClose | frontend/src/lib/components/SubstitutionInputs.svelte | delayed blur close |
| 96 | dynamic-filter addSelectedItem | frontend/tests/dynamic-substitution-filters.spec.ts | classification readiness |
| 97 | dynamic-filter retry synchronization | frontend/tests/dynamic-substitution-filters.spec.ts | response wait |
| 98 | real-stack gate helper | scripts/check.py | quick/static gate |
| 99 | coverage contract additions | scripts/test_check_coverage.py | measured Phase 08 scope |
| 100 | DESIGN/operations/evidence contracts | docs/design/DESIGN-005.md; docs/operations/real-stack-e2e.md; docs/implementation/04_OPEN.md; docs/implementation/evidence/task-279-preparation.md | source of truth |

## 6. Function-Level Audit

audited_symbol_count: 100

| # | Audited symbol/surface | Audit result and evidence |
|---:|---|---|
| 1 | SafetyError | PASS: unsafe boundary failures are explicit and stop work. |
| 2 | ListenerExitedError | PASS: readiness exit is distinguished for bounded retry. |
| 3 | PostgresTarget | PASS: validated loopback maintenance target carries password only to child environment. |
| 4 | PostgresTarget.parse | PASS: scheme, loopback host, `/postgres`, user, and port checks run before psql. |
| 5 | PostgresTarget.command | PASS: argv form avoids shell expansion. |
| 6 | PostgresTarget.environment | PASS: `PGPASSWORD` is not persisted in diagnostics. |
| 7 | PostgresTarget.database_url | PASS: generated DB name is validated before URL construction. |
| 8 | OwnedProcess | PASS: state carries name, PID, and validated start token. |
| 9 | PortReservation | PASS: loopback socket remains bound until controlled release. |
| 10 | PortReservation.release | PASS: deterministic socket close is used by normal and exceptional paths. |
| 11 | SignalController | PASS: handled signals are queued during teardown. |
| 12 | SignalController.handle | PASS: pre-teardown signals interrupt; teardown signals only record. |
| 13 | SignalController.begin_teardown | PASS: entered before diagnostics and cleanup. |
| 14 | SignalController.end_teardown | PASS: later signals interrupt instead of being reported as success. |
| 15 | validate_run_id | PASS: exactly 24 lowercase hex characters. |
| 16 | database_name | PASS: exact `mealswapp_e2e_<id>_test` pattern. |
| 17 | ownership_comment | PASS: exact owner/run/timestamp marker. |
| 18 | validate_database_name | PASS: rejects development, unmarked, and non-test names. |
| 19 | validate_environment | PASS: only development is permitted. |
| 20 | validate_stale_age | PASS: minimum 900 seconds is enforced. |
| 21 | run_command | PASS: every command uses a new process group and timeout rollback. |
| 22 | process_group_exists | PASS: `/proc/*/stat` parsing handles parenthesized comm fields and ignores zombie-only groups. |
| 23 | rollback_spawned_process | PASS: exact newly-created group receives bounded TERM/KILL and is waited on. |
| 24 | psql | PASS: only validated target command is invoked. |
| 25 | create_database | PASS: DB-name and comment run IDs cross-check before PostgreSQL. |
| 26 | database_comment | PASS: only exact generated names are inspected. |
| 27 | drop_owned_database | PASS: exact name, run ID, expected comment, and live comment all match before DROP. |
| 28 | ownership_comment_run | PASS: only strict owner comments parse. |
| 29 | parse_ownership_comment | PASS: malformed/ambiguous comments are rejected. |
| 30 | validate_ownership_comment | PASS: stale state timestamp and run marker must match. |
| 31 | docker_inspect | PASS: exact container regex precedes Docker inspection. |
| 32 | inspect_labels | PASS: only Docker Config labels are trusted. |
| 33 | remove_owned_redis | PASS: exact run label is checked before forced removal. |
| 34 | reserve_ports | PASS: two distinct bound loopback ports are allocated and collision cleanup closes both. |
| 35 | reserve_port | PASS: one-port helper is bounded and used only for timeout injection. |
| 36 | process_start_token | PASS: missing `/proc` identity is represented as unavailable. |
| 37 | validate_process_start_token | PASS: missing/malformed tokens fail closed. |
| 38 | stop_process | PASS: stale cleanup refuses unavailable/reused identity before signaling. |
| 39 | stop_owned_process | PASS: unavailable live leader token raises without `killpg`; descendants are handled only after ownership validation. |
| 40 | wait_http | PASS: bounded readiness detects listener exit and timeout. |
| 41 | request_json | PASS: fixture request body/cookies remain in memory and errors propagate. |
| 42 | register_fixture | PASS: registration/verification occurs over run-owned API and returns safe IDs/request IDs. |
| 43 | data_field | PASS: malformed response data becomes empty and callers validate it. |
| 44 | safe_diagnostics | PASS: only run IDs, exact request IDs, safe events, and result cross the persistent boundary. |
| 45 | Harness | PASS: one generated identity owns DB, Redis, processes, ports, and artifacts. |
| 46 | Harness.__init__ | PASS: environment guard precedes resource creation. |
| 47 | Harness.write_state | PASS: atomic same-directory state replacement publishes ownership metadata. |
| 48 | Harness.start_process | PASS: gate child cannot exec before ownership state publication; rollback failures preserve tracking/state. |
| 49 | Harness.run | PASS: execution failure, diagnostics failure, cleanup, late signal, and final status are ordered safely. |
| 50 | Harness.retire_process | PASS: retry teardown confirms group absence before removing tracking. |
| 51 | Harness.start_application_stack | PASS: listener exit retires the attempted stack and retries with fresh reservations at most twice. |
| 52 | Harness.execute | PASS: create/migrate/build/start/bootstrap/browser order uses generated targets and releases reservations in finally. |
| 53 | Harness.application_environment | PASS: generated DB, Redis, API/frontend ports, origins, and Vite target are isolated. |
| 54 | Harness.start_redis | PASS: run label, loopback binding, exact container, and PONG readiness are verified. |
| 55 | Harness.export_diagnostics | PASS: persistent diagnostics are sanitized and synthetic screenshots survive raw teardown. |
| 56 | Harness.cleanup | PASS: every owned category is attempted, failures are recorded, and state is updated. |
| 57 | cleanup_stale | PASS: production/age/state/name/comment/token/label guards precede owned recovery; repeated recovery is idempotent. |
| 58 | remove_owned_raw_directory | PASS: only exact run-prefixed directories below the system temp directory are removed. |
| 59 | write_sanitized_png | PASS: source-free one-pixel PNG contains no screenshot pixels. |
| 60 | write_sanitized_png.chunk | PASS: bounded length/CRC output is deterministic. |
| 61 | install_signal_handlers | PASS: INT/TERM/HUP handlers are installed for the harness. |
| 62 | restore_signal_handlers | PASS: prior handlers are restored after either CLI mode. |
| 63 | parse_args | PASS: stale age is rejected outside explicit stale mode. |
| 64 | main | PASS: environment/target guards precede work and handled failures return nonzero safely. |
| 65 | SafetyValidationTests | PASS: unsafe targets, production, DB/comment, Redis, and age guards all pass. |
| 66 | DiagnosticsTests | PASS: diagnostics, tokens, rollback, gate, signal, timeout, and artifact tests all pass. |
| 67 | IntegrationContractTests | PASS: ports, retry, direct-boundary, Vite, and Redis contracts all pass. |
| 68 | stale/unsafe target matrix | PASS: no boundary mock is called for rejected production/host/name inputs. |
| 69 | exact DB/comment matrix | PASS: mismatches reject before psql/drop. |
| 70 | exact Redis label test | PASS: mismatched label rejects before Docker removal. |
| 71 | diagnostics failure test | PASS: cleanup is called and original failure remains authoritative. |
| 72 | diagnostic allowlist test | PASS: credentials, cookies, CSRF, PII, bodies, and DB URL are excluded. |
| 73 | token absence/reuse tests | PASS: neither missing nor reused token issues a signal. |
| 74 | spawn publication rollback test | PASS: exact child rollback runs after publication failure. |
| 75 | rollback-failure state test | PASS: both failures are reported, owner remains tracked, and `rollback_failed` is published. |
| 76 | spawn gate test | PASS: executable marker is absent when state publication fails. |
| 77 | held-port publication test | PASS: state publication precedes reservation release. |
| 78 | exited-leader fail-closed test | PASS: unavailable leader token raises and does not call `killpg`. |
| 79 | late-signal final-diagnostics test | PASS: a signal injected during passed-result export raises instead of returning 0. |
| 80 | timeout descendant test | PASS: descendant process is gone after grouped timeout rollback. |
| 81 | paired reservation collision tests | PASS: duplicate reservations close safely and held ports reject competitors. |
| 82 | listener-exit retry test | PASS: failed attempt is retired and fresh API/frontend ports/proxy target are used. |
| 83 | release-to-exec competitor test | PASS: adversarial bind is detected as listener exit and cannot silently pass. |
| 84 | Task 261 boundary contract | PASS: no SQL or `node:child_process` promotion remains. |
| 85 | Redis readiness contract | PASS: readiness invokes `docker exec` on the exact run-owned container. |
| 86 | shell compatibility caller | PASS: strict shell wrapper forwards only to the harness. |
| 87 | Task 261 real-admin flow | PASS: fixture env, generated API/browser actions, deletion, IDs, and dynamic filters pass. |
| 88 | requiredFixture | PASS: missing fixture values fail immediately without fallback shared state. |
| 89 | responseID | PASS: created classification/item IDs are asserted as UUIDs. |
| 90 | createPrivateItemFixture | PASS: private fixture uses authenticated API/CSRF/idempotency, not SQL. |
| 91 | managedByHarness/baseURL | PASS: managed runs omit webServer and use allocated frontend URL. |
| 92 | apiTarget | PASS: Vite proxy target is environment-configurable. |
| 93 | Vite proxy export | PASS: `/api` routes to the allocated API port in live runs. |
| 94 | openFilterOptions | PASS: refocus cancels stale close timer and keeps dropdown open. |
| 95 | scheduleFilterOptionsClose | PASS: blur close is delayed only long enough for option interaction. |
| 96 | dynamic-filter addSelectedItem | PASS: selected classification rendering is awaited before assertion. |
| 97 | dynamic-filter retry synchronization | PASS: retry waits for its response before asserting empty state. |
| 98 | real-stack gate helper | PASS: harness unit tests are part of quick/static/full orchestration. |
| 99 | coverage contract additions | PASS: exact current Phase 08 measurements are machine-checked. |
| 100 | DESIGN/operations/evidence contracts | PASS: design, operational safety, coverage, task, and preparation records agree with observed behavior. |

## 7. Findings

blocking_findings: 0
important_findings: 0
optional_findings: 0

No blocking, important, or optional findings remain. The prior rejected findings
were specifically re-tested: production stale-mode guard, diagnostics-failure
teardown, fail-closed missing/reused process tokens, spawn gate/publication and
rollback state preservation, process groups/zombie handling, exact database and
comment matching, queued and late signals, release-to-bind port races, stale
recovery, and the formerly flaky browser/full gate.

## 8. Commands Run

All commands ran in `/home/wiktor/Work/mealswapp` on 2026-07-28. Results below are
the exact observed outcomes, with expected negative scenarios explicitly marked.

| Command or scenario | Result |
|---|---|
| `sed -n '1,260p' /home/wiktor/.agents/skills/phase-orchestrator/templates/review_checklist.md` | PASS; checklist read completely |
| `sed -n '1,260p' /home/wiktor/.agents/skills/code-review-skill/SKILL.md` | PASS; invoked exactly once |
| Relevant Python/Svelte/Go/security guide reads | PASS |
| `python3 -m unittest -v scripts/test_run_real_stack_e2e.py` | PASS; 28 tests |
| `python3 -m py_compile scripts/run-real-stack-e2e.py scripts/test_run_real_stack_e2e.py` | PASS |
| `python3 -m unittest -v scripts/test_check_coverage.py` | PASS; 21 tests |
| `python3 scripts/validate-traceability.py` | PASS |
| `python3 scripts/validate-task-list.py` | PASS; 286 sequential tasks |
| `git diff --check` | PASS |
| `bun test frontend/src/lib/components/SubstitutionInputs.test.ts` | PASS; 11 tests, 90 expectations |
| `cd frontend && bun run build` | PASS; 219 modules transformed |
| `python3 scripts/check.py --quick` | PASS; 28 harness, 536 frontend, 24 changed-browser passes plus 2 expected skips, changed backend/static lanes |
| `python3 scripts/check.py` | PASS; all lanes; 309 browser passes plus 5 expected skips; Phase 08 4639/4980 (93.2%) |
| Dynamic-filter Playwright stress, 120 repetitions after repair | PASS; 120/120 |
| Two concurrent `python3 scripts/run-real-stack-e2e.py --timeout-seconds 180` | PASS/PASS; exit 0/0 |
| Two sequential ordinary harness runs | PASS/PASS; exit 0/0 |
| `--inject-assertion-failure` | Expected FAIL; exit 1, `CalledProcessError` |
| `--inject-api-timeout` | Expected FAIL; exit 1, `TimeoutError` |
| Harness child sent `SIGINT` | Expected FAIL; exit 1, `InterruptedError` |
| Harness child sent `SIGTERM` | Expected FAIL; exit 1, `InterruptedError` |
| Harness child sent `SIGHUP` | Expected FAIL; exit 1, `InterruptedError` |
| Harness child sent `SIGKILL` | Expected kill; exit -9 and marked stale state remained |
| Direct `cleanup_stale(..., age=900, now=created+900)` after SIGKILL | PASS; removed 1 |
| Repeated mature stale cleanup | PASS; removed 0 |
| Production CLI stale mode | Expected FAIL; exit 1 before boundary |
| CLI stale age 899 | Expected FAIL; exit 1 |
| Generated DB residue query | PASS; zero rows |
| Run-labeled Docker container query | PASS; none |
| Run process query after scenarios | PASS; none |
| Development E2E fixture-name query | PASS; `global|0`, `private|0` |
| Development pre-existing Task 261 row check | `global|19`, `private|21`; unchanged pre-existing baseline, not created by this review |
| Persistent forbidden diagnostic scan | PASS; no matches |

## 9. Files Inspected and Staleness Fingerprints

The Python harness and tests were inspected line by line, including every
function/method and nested test helper listed in the inventory. Changed frontend
symbols, shell/config callers, gate registration, coverage contract, design,
operations, task row, preparation evidence, and prior review findings were also
inspected. The AST refresh found 64 function/class symbols in the harness and 34
in its test module; the cross-file inventory below includes the complete changed
surface and was audited 100/100.

| Path | SHA-256 |
|---|---|
| `docs/implementation/02_TASK_LIST.md` | `e96e5bbf3902ff354cfa180b84d113b3cefd7e3a55d9f710d9b71d2491b52da1` |
| `docs/implementation/evidence/task-279-preparation.md` | `5c57523da3092b8cc3bb9fd1d8cf9076c1ec23364dc1d57015723165fc73e03d` |
| `scripts/run-real-stack-e2e.py` | `52647965aead4bf16f87e6c1ae4fc4a8c6a7a9c20af4b1ea45921089f8aa140d` |
| `scripts/test_run_real_stack_e2e.py` | `545501b661902524faadf19ae3f208d7a278fe2f1f78a261a24cce12be469031` |
| `scripts/verify-task-261-ui.sh` | `1799f1d533203fc929cf69edeb4095b6d83d8aea10f2266ed001b564e058aafa` |
| `frontend/tests/task261-real-admin-flow.spec.ts` | `a1ed430a6511e04436b31417b86608b5174354d4219bd72d15bca94ed76c1210` |
| `frontend/playwright.real-stack.config.ts` | `11e3530338347f87788fdfb792d224971fab965dfd79b7a51ccf5e81dc230ead` |
| `frontend/vite.config.ts` | `49b59bf06e2a1ffb79851354cbd5344a8c6716ffd91125d793e65a9a81152332` |
| `frontend/src/lib/components/SubstitutionInputs.svelte` | `7ddf4f5a66ae76a5133bc487f7b25c877a65bab35a571a5836e9037b4b41c3ce` |
| `frontend/tests/dynamic-substitution-filters.spec.ts` | `8a62e7b9104459a98d08242e2268bc14fa6b8113eb5f11a07c03b992abe3449f` |
| `scripts/check.py` | `d9ff7da8c7a5c4b5be223a2e9699bd7bf004c6f2be38531634cc950081d20d38` |
| `scripts/test_check_coverage.py` | `20dbbc53a9df76a9f19a4784fd7f01dccc4e5990717e46f4898f74282a6a0379` |
| `docs/design/DESIGN-005.md` | `796dbabc2d474ce1c05068ee4b4ad7a17aec803c1db7d55c8d7a829e03a58f3b` |
| `docs/operations/real-stack-e2e.md` | `85268f77839f072866bbe98c0d2555a8d97fa2bc8f527adda14da9a7ac006c0c` |
| `docs/implementation/04_OPEN.md` | `f65d915e4cc012e4afef0616c91179c9eaac323f961d6b0012900ebafcc8fecb` |
| prior `docs/implementation/evidence/task-279-review.md` | `be8b916876e0122725c582d7c51c2c1856fc7ed66b0ee3937220b3e5dcc772ae` before this report |

The prior report was rejected for five concrete boundary gaps and one failed
full-gate result. Its preparation claims were treated as stale until the fresh
tests and live probes above passed.

## 10. Coverage and Exceptions

The exact full gate passed. Its relevant measured results are:

- Phase 08 Go: `4639/4980` statements (`93.2%`) under the machine-checked
  accepted exception contract in `docs/implementation/04_OPEN.md`.
- Repository aggregate Go coverage: `87.4%`.
- Frontend unit coverage: 536 tests, 2,803 expectations, `95.19%` functions and
  `96.06%` lines; current Phase 08 exceptions are machine-checked.
- Full browser lane: 309 passed and 5 expected skips.
- No vulnerability was reported by `govulncheck`; the tool noted dependency
  vulnerabilities not called by the code.
- The existing OpenAPI redirect-only OAuth callback warning remained the known
  warning; Redocly passed.

## 11. Negative and Regression Checks

- No code or task-list status changed during this review.
- The development database has no generated `E2E ...` fixture names. The
  `Task 261 global`/`Task 261 private` counts are 19/21 pre-existing rows
  recorded by preparation evidence; this review did not add, delete, or mutate
  them.
- No generated E2E database, run-labeled Redis container, harness process, or
  forbidden persistent diagnostic value remains.
- No harness source or Task 261 flow contains `FLUSHALL`, broad schema/table
  truncation, prefix-based cleanup, direct SQL promotion, or
  `node:child_process`.
- The production stale guard is checked before filesystem enumeration and the
  age guard rejects 899 seconds.
- Artifacts are persistent only as safe metadata; raw logs, traces, screenshots,
  cookies, CSRF values, credentials, fixture PII, and database URLs are removed
  with the temporary run directory.

## 12. Decision

review_decision: PASSED
decision: PASSED

Task 279 is PASSED after final independent re-review. The repaired process,
database, Redis, port, signal, diagnostics, stale-recovery, browser, and gate
boundaries satisfy the authoritative row. No repair instructions remain.

## 13. Repair Context

This is the final re-review after the repair cycle described in the updated
preparation evidence. The prior review's blocking/important findings were all
closed and re-proved:

1. production stale cleanup rejects before any boundary;
2. diagnostics failure cannot skip unconditional teardown;
3. process token absence/reuse fails closed before group signaling;
4. spawn gates and publication rollback preserve ownership state, including
   reporting rollback failure;
5. all managed commands use process groups and timeout descendants are reaped;
6. database deletion requires exact generated name and matching comment;
7. handled and late signals cannot produce false success;
8. release-to-bind port races are detected and retried with fresh ports;
9. stale state recovery is age-bounded and idempotent;
10. the dynamic-filter browser race is stable under repetition and the exact
    full gate passes.

The only file written by this review is this evidence report. Implementation,
task-list, and unrelated worktree changes were not edited.
