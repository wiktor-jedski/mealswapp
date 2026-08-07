# Task 302 Preparation — Final Remediation Acceptance Gate

## Scope

- Task: 302, `Phase 08.02 Final Remediation Acceptance Gate`
- Worktree: `/home/wiktor/Work/worktrees/mealswapp/302`
- Branch: `task-302-final-remediation-acceptance`
- Baseline after phase synchronization: `ce4c14db`
- Design traceability: `DESIGN-014: MetricsCollector`
- Delivery status: `PREPARED`; the Phase 08.02 acceptance gate remains `FAIL`.

## Phase synchronization

Fetched `origin/multistep-phase-08`, fast-forwarded the local phase branch, and
published the recent Task 303 integration/status commit `c626e23f`. Merged that
phase tip into Task 302 as `ce4c14db`, resolving only the whitespace conflict in
`frontend/src/lib/components/AdminDataManagement.svelte`.

## Acceptance repair and evidence

The latest isolated Task 283 run is
`d6953d11779c18e38ba0c44c`. It reached the application and produced 40 PASS and
4 BLOCKED criteria. The only blocked criteria are
`P08-SWR033-STEP-01` through `P08-SWR033-STEP-04`, which remain explicit
provider-storage capability blockers. Task 303 mobile global discovery and
Task 304 vocabulary evidence pass in the same run. Earlier committed Task 282
and Task 284 evidence is preserved; the new per-requirement reports and run
context are under `docs/implementation/implemented/08.02_PHASE_EVIDENCE/`.

The acceptance-test repair in commit `3b37c647` narrows the solid and liquid
catalog assertions to the exact item names created by the scenario. This removes
the broad shared-query pagination/order false negative without changing product
code or weakening any assertion.

The synchronized finding state is:

- Closed: `P08-FIND-283-001` and `P08-FIND-283-006`, with latest passing reports.
- Still open: `P08-FIND-283-005` for the four blocked SW-REQ-033 criteria.
- Unchanged: `P08-FIND-281-002` and the three deployed-environment Task 285
  findings.

No Task 302 review checklist was present in `.git/reviews`; therefore this
preparation does not claim review-gate completion or change the task-list
status to `PASSED`.

## Verification evidence

- Task 283 isolated real-stack rerun: exit code `2` by design for truthful
  blocked results; no product/browser failure after the query repair.
- `frontend` typecheck: passed.
- Playwright test discovery for `task283-manual-catalog.spec.ts`: passed.
- `python3 scripts/phase08_acceptance.py validate`: passed.
- `python3 scripts/validate-task-list.py`: passed.
- `python3 scripts/validate-traceability.py`: passed.
- `python3 scripts/check.py --quick`: passed, including 109 acceptance-contract
  tests and the Go vulnerability scan.
- `python3 scripts/check.py`: failed only in the static lane because the
  concurrent run hit the existing temporary-child-process race in
  `test_run_command_timeout_reaps_descendant_process_group`; frontend/browser
  completed with 349 passed and 81 skipped, and the backend lane completed.
- A focused rerun of `python3 -m unittest scripts/test_run_real_stack_e2e.py`
  passed all 28 tests, so that full-gate failure was not reproducible.
- Final `git diff --check`: passed.
