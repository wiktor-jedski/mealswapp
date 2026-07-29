# Task 286 Preparation Evidence

Date: 2026-07-29  
Baseline commit: `f9a646ffede5c8a63ad787a07eaba7efc0433677`  
Branch: `multistep-phase-08`  
Task status observed before and after repair: `PREPARED`  
Task-list status edited by this preparation: **no**

## Outcome

Task 286's acceptance machinery and final Phase 08.02 artifacts are prepared.
The delivery is complete, but the product acceptance gate truthfully remains
**FAIL** and the project-owner decision remains **PENDING**:

- 12 Task 280 scenarios and all 91 criteria are represented exactly once in
  the aggregate;
- fresh Task 281-285 execution produced `67 PASS`, `13 FAIL`, and
  `11 BLOCKED`;
- duplicate cross-feature observations use the strongest status
  (`FAIL > BLOCKED > PASS`) and are never weakened;
- all 24 non-pass criteria map to an unresolved synchronized Phase 08 finding;
- the report and UAT enumerate all 14 current unresolved Phase 08 findings,
  including the three not selected by the current aggregate root-cause rows;
- no deviation is accepted and the owner/date decision fields remain
  unchecked;
- historical `docs/implementation/implemented/08_PHASE_UAT.md` remains
  byte-identical;
- committed evidence includes source JSON/HTML reports, desktop/mobile
  screenshots, backend/database/Redis proof, run traces/diagnostics/results,
  and the truthful blocked centralized-log-sink projection;
- the isolated runs left no run-owned database or Redis container behind.

Task delivery status is intentionally distinct from requirement acceptance.
This preparation does not recommend marking the Phase 08.02 UAT gate accepted.

## Final-review repair

Every finding in `task-286-final-review.md` was repaired without editing the
Task 286 row:

- report controls and task trace were rebuilt from the current `PREPARED`
  task-list state;
- report, HTML, and UAT now carry the exact 14-finding open-ledger set;
- source-report validation recursively rejects sensitive keys and values,
  restricts evidence references to exact `type`/`path` objects, and rejects
  impossible owner/deviation calendar dates;
- final-report validation now applies the same recursive sensitive-value
  policy before trusting any field, normalizes sensitive key aliases across
  punctuation/case, and exact-schema-checks every nested source/evidence
  object;
- `validate_final_schema` now type-checks the complete result, source,
  evidence, hash, trace, decision, deviation, historical-UAT, count, and
  finding tree before any set membership, dictionary lookup, identity hash,
  path resolution, or strongest-status operation can consume nested values;
- `validate_source_report` now rejects non-string producer
  `result.criterionId` and `result.status` values before dictionary, set, or
  status-rank membership;
- `validate_build_input` now exact-schema-checks the complete historical-UAT,
  report, supporting-artifact, task-evidence, and decision tree before
  dictionary access, task-range membership, path arithmetic, hashing, or
  filesystem publication;
- `validate_decision` now guards its decision, deviation, and result operands
  before every status or criterion set operation;
- malformed list/object values, including exact regressions for
  `result.criterionId=[]`, `source.report=[]`, and `source.status=[]`, now
  produce bounded `UATError` rejection, while the CLI emits one structured
  nonzero diagnostic without a traceback;
- producer-report `criterionId=[]`/`status=[]` and build-input
  `historicalUat=[]`, `reports=[[]]`, `report.path=[]`, decision `status=[]`,
  and deviation `criterionId=[]` now have exact direct and CLI no-traceback
  regressions;
- source reports now have exact top-level, result, root-cause, request-ID,
  backend-summary, finding-link, evidence-link, and aggregate contracts;
- final source identities must be unique and match the Task 281-285 report
  selection; each source projection must exactly match its selected report;
- final results, counts, and gate are recomputed from the selected validated
  report evidence, so coordinated nested-source/result status weakening
  cannot manufacture PASS;
- focused regressions cover PREPARED-state freshness, exact open-finding
  parity, recursive sensitive metadata at both source and final-report
  boundaries, normalized aliases at arbitrary depths, calendar-valid dates,
  malformed nested source types, duplicate sources, and coordinated FAIL or
  BLOCKED weakening.

## Protocol and baseline

The Phase Orchestrator preparation contract was followed without changing the
task-list row. The repository was already heavily dirty with preserved Tasks
276-285 work. Task-owned scope is high-confidence for the new Task 286 files
and generated artifact tree. The `scripts/check.py` edit is limited to
registering the Task 286 focused tests and validator.

No writable subagent capability was available in this session, so independent
delegation/review could not be performed. `code-review-skill` was invoked once
and its Python/cross-cutting guidance was applied to the implementation audit
below. This document is preparation evidence, not an independent PASSED review.

Baseline observations:

- HEAD: `f9a646ffede5c8a63ad787a07eaba7efc0433677`
- Task 286 row at repair time: `PREPARED`, dependencies 281-285 `PASSED`
- historical Phase 08 UAT SHA-256:
  `371361f141ec2fe883e448e21480a63ddf9e8fa1d37910ea7fd9209bf511a9c0`
- worktree: dirty before Task 286; all unrelated changes were preserved

## Fresh end-to-end execution

Local PostgreSQL and Redis were started with `bash scripts/start-services.sh`.
The first Task 281 attempt occurred before services were ready and truthfully
returned a synchronized infrastructure `BLOCKED` report. Services were then
started and the authoritative cross-feature sequence was rerun.

| Task | Command | Exit | Fresh run/report | Result |
|---:|---|---:|---|---|
| 281 | `python3 scripts/run-task281-acceptance.py --timeout-seconds 240` | 1 | `b59a299af330a66788a5d5b5` | 7 PASS, 2 FAIL |
| 282 | `python3 scripts/run-task282-acceptance.py --timeout-seconds 240` | 1 | `7e90a8f191987741e6caac1f` | 19 PASS, 4 FAIL, 1 BLOCKED across SW-REQ-033/055/090 supporting reports |
| 283 | `python3 scripts/run-task283-acceptance.py --timeout-seconds 240` | 1 | `f50ec3f0e8eb47a07ce0de77` | 44 authoritative criteria: 34 PASS, 7 FAIL, 3 BLOCKED |
| 284 | `python3 scripts/run-task284-acceptance.py --timeout-seconds 240` | 1 | `00807a2ec1a53daf7b4d7161` | 15 PASS, 2 FAIL |
| 285 | environment-cleared `python3 scripts/run-task285-acceptance.py` | 2 | `task285-abcdab7626a34fa9ed348a6a` | 0 PASS, 0 FAIL, 7 BLOCKED |

Task 282 and Task 283 intentionally overlap SW-REQ-033 and SW-REQ-090.
Task 286 retains both observations and aggregates by strongest status. No
source report or evidence file is orphaned.

## Requirement acceptance result

| Requirement | PASS | FAIL | BLOCKED | Disposition |
|---|---:|---:|---:|---|
| SW-REQ-019 | 4 | 0 | 0 | PASS |
| SW-REQ-032 | 3 | 0 | 0 | PASS |
| SW-REQ-033 | 1 | 1 | 3 | non-pass |
| SW-REQ-043 | 4 | 1 | 0 | non-pass |
| SW-REQ-054 | 7 | 2 | 0 | non-pass |
| SW-REQ-055 | 11 | 3 | 0 | non-pass |
| SW-REQ-056 | 10 | 3 | 0 | non-pass |
| SW-REQ-057 | 12 | 2 | 0 | non-pass |
| SW-REQ-072 | 5 | 1 | 0 | non-pass |
| SW-REQ-073 | 6 | 0 | 0 | PASS with production-worker/backend proof |
| SW-REQ-084 | 0 | 0 | 7 | blocked: no approved deployed sink authority |
| SW-REQ-090 | 4 | 0 | 1 | non-pass |
| **Total** | **67** | **13** | **11** | **FAIL; decision PENDING** |

All current unresolved Phase 08 findings included in the report and UAT:

`P08-FIND-281-002`, `P08-FIND-281-003`, `P08-FIND-282-001`,
`P08-FIND-282-002`, `P08-FIND-282-004`, `P08-FIND-282-005`,
`P08-FIND-283-001`, `P08-FIND-283-002`, `P08-FIND-283-005`,
`P08-FIND-283-006`, `P08-FIND-284-001`, `P08-FIND-285-001`,
`P08-FIND-285-002`, and `P08-FIND-285-003`.

The 11 findings selected by current non-pass root-cause rows remain exact
criterion links. `P08-FIND-281-002`, `P08-FIND-285-002`, and
`P08-FIND-285-003` remain explicitly visible as unresolved ledger findings
even though the aggregate does not select their historical roots.

## Files and executable symbols

### `scripts/phase08_uat.py`

| Symbol | Contract and evidence |
|---|---|
| `UATError` | Bounded validation failure type; CLI catches it without traceback or private diagnostics. |
| `sha256` | Hashes only regular repository files and rejects missing, escaping, or symlink-component evidence. |
| `relative` | Produces repository-relative POSIX evidence paths and rejects escape. |
| `repo_path` | Rejects absolute/traversing report paths before filesystem access. |
| `load_json` | Reuses Task 280 duplicate-key rejection and bounded JSON loading. |
| `load_controls` | Validates the 91-criterion manifest, append-only finding history, and `04_OPEN.md` ledger together. |
| `task_rows` | Parses and requires exact Task 276-286 trace without mutating the task list. |
| `expected_report_status` | Implements truthful report exit semantics: FAIL/1, BLOCKED/2, PASS/0. |
| `validate_source_report` | Audits recursive sensitive metadata, exact evidence-reference schema, criterion membership, aggregate status, open-finding synchronization, every evidence file, and orphan files. |
| `validate_decision` | Prevents acceptance without owner/date and requires complete owner/date/reason/retest/expiry deviations with real calendar dates for every non-pass. |
| `valid_date` | Strictly validates ISO calendar dates with the standard library. |
| `aggregate` | Preserves every source observation and selects strongest status without weakening. |
| `verify_special_evidence` | Rejects centralized logging from console output and erasure PASS without completed worker/backend evidence. |
| `copy_report_tree` | Copies only one selected sanitized report tree through a same-parent temporary staging directory. |
| `evidence_hashes` | Creates deterministic SHA-256 manifests for controls, reports, traces, closures, and task evidence. |
| `render_html` | Emits the separate readable 91-criterion report and complete open-finding set. |
| `render_uat` | Emits the separate UAT, complete open-finding set, owner checks, unchecked decision, task trace, and explicit delivery/acceptance distinction. |
| `build` | Validates fresh inputs, copies sanitized evidence, reconciles all criteria/findings, emits JSON/HTML/UAT, and validates the result. |
| `validate_hash_rows` | Type-checks and rechecks every recorded path/hash pair against current repository contents. |
| `validate_final_schema` | Type-checks every nested final-report field before keyed, set, membership, hashing, path, or status-ranking operations. |
| `validate_final` | Recursively rejects sensitive metadata anywhere in the final report, exact-schema-checks nested source/evidence objects, requires unique canonical source identities, recomputes results/counts/gate from validated selected reports, and revalidates open-ledger parity, manifest parity, findings, closures, special security boundaries, Tasks 276-286, historical UAT, and rendered artifacts. |
| `command_validate` | Runs the committed final gate validator. |
| `main` | Provides bounded `build`/`validate` CLI handling and nonzero invalid-state exit. |

Normal, malformed, boundary, cancellation/concurrency, resource, security, and
performance audit:

- input arrays and evidence output are bounded by the fixed 91-criterion
  manifest and selected report list;
- JSON duplicate keys, unsupported status, missing criteria, duplicate
  destinations, absolute/traversing paths, symlink components, stale hashes,
  missing files, orphan files, contradictory metadata/status/counts, and stale
  task/finding state fail closed;
- evidence copy has no network, subprocess, lock, transaction, or goroutine
  lifecycle; temporary staging is context-managed and cleaned on error;
- the only recursive deletion target is the exact generated
  `08.02_PHASE_EVIDENCE/<selected-report-id>` destination after the selected
  source has validated;
- no user-controlled data reaches SQL, commands, logs, or trusted totals;
- report aggregation is linear in bounded source results/files, and no
  unbounded external I/O is introduced;
- the implementation uses only the Python standard library and existing Task
  280 validation helpers.

### `scripts/test_phase08_uat.py`

The 28 executable test methods each passed and prove:

1. current 91-criterion/hash completeness and PREPARED Task 286 trace;
2. missing criteria rejection;
3. weakened/contradictory status rejection;
4. non-pass without synchronized finding rejection;
5. closed finding without exact retest evidence rejection;
6. orphan source evidence rejection;
7. exact 14-finding open-ledger parity;
8. recursive sensitive source evidence-metadata rejection;
9. exact nested final-report `metadata.connection.databaseUrl` rejection;
10. normalized sensitive aliases at arbitrary final-report depths;
11. impossible calendar-date rejection;
12. owner/date requirement;
13. mandatory non-pass acceptance denial;
14. incomplete deviation denial;
15. console-only centralized logging denial;
16. erasure without worker/backend proof denial;
17. historical UAT immutability;
18. required unchecked owner decision fields;
19. malformed nested `taskId`, `requestIds`, and `backendEvidence` rejection;
20. duplicate final source identity rejection;
21. coordinated FAIL source/result weakening rejection;
22. coordinated BLOCKED source/result weakening rejection;
23. exact list-valued `criterionId`, source-report path, and source-status
    regressions return `UATError`, never `TypeError`;
24. broad malformed nested scalar/list/object coverage is rejected before
    set, dictionary, key, hash, path, and status-ranking operations;
25. CLI validation rejects malformed nested values with a structured nonzero
    diagnostic and no traceback.
26. producer source-report `criterionId=[]` and `status=[]` are bounded
    `UATError` diagnostics before dictionary/set/status operations;
27. malformed historical-UAT, report selection/path, decision status, and
    deviation criterion build inputs fail before path, hash, or publication
    operations;
28. build CLI malformed-input and malformed-source cases return one structured
    nonzero diagnostic without a traceback.

`Phase08UATTests.validate` and `Phase08UATTests.recount` are test-only helpers:
the first executes the complete validator against isolated mutations; the
second updates counts/gate only so adversarial tests reach the intended deeper
boundary rather than failing on an earlier count mismatch.

### Other changed surfaces

- `scripts/check.py`
  - `validate_phase08_acceptance_contracts` now runs the Task 286 tests and
    committed validator;
  - `QUICK_PATHS` maps `phase08_uat.py` and its tests into changed-area gates.
- `docs/testing/phase08/uat-input.json`
  - strict JSON configuration selecting the fresh Task 281-285 reports,
    run-context traces, Tasks 276-285 evidence, historical UAT hash, and
    pending decision.
- `docs/testing/phase08/uat-input.json-trace.md`
  - JSON sidecar traceability to DESIGN-014, Task 286, and Task 280.
- `docs/implementation/implemented/08.02_PHASE_REPORT.json`
  - machine-checkable 91-criterion report, all 14 open findings, Task 276-286
    trace, current controls/closures/source/render hashes, and decision state.
- `docs/implementation/implemented/08.02_PHASE_REPORT.html`
  - separate human-readable report.
- `docs/implementation/implemented/08.02_PHASE_UAT.md`
  - separate project-owner UAT and unchecked acceptance decision.
- `docs/implementation/implemented/08.02_PHASE_EVIDENCE/`
  - 139 sanitized, hashed source report/evidence/run-context files.
- `docs/implementation/implemented/08.02_PHASE_CHECK.html` and
  `screenshots/08.02_PHASE_CHECK-*.png`
  - successful aggregate quality-gate report and 20 desktop/mobile scenario
    screenshots.

No application production code, requirement, architecture/design source,
finding state, historical UAT, or task-list status was changed for Task 286.

## Verification commands

| Command | Result |
|---|---|
| `python3 -m unittest -v scripts/test_phase08_uat.py scripts/test_phase08_acceptance.py` | PASS, 53 tests (28 Task 286 and 25 Task 280 acceptance tests) |
| `python3 -m unittest -v scripts/test_phase08_acceptance.py` | PASS, 25 tests |
| `python3 scripts/phase08_uat.py validate` | PASS, 91 criteria and Tasks 276-286; hashes current |
| `python3 -m py_compile scripts/phase08_uat.py scripts/test_phase08_uat.py scripts/check.py` | PASS |
| `python3 scripts/phase08_acceptance.py validate` | PASS, 12 scenarios and 91 criteria |
| `python3 scripts/validate-task-list.py` | PASS, 286 sequential tasks |
| `python3 scripts/validate-traceability.py` | PASS |
| committed evidence sensitive-value scan | PASS, 127 UTF-8 text artifacts; 11 image artifacts remain non-text |
| evidence symlink, staging-directory, and Task 281-286 process probes | PASS, no resources found |
| run-owned Docker container and PostgreSQL database query | PASS, no resources found |
| pre-repair `MEALSWAPP_PLAYWRIGHT_WORKERS=8 bun run test:e2e -- tests/admin-data-management.spec.ts --grep "older classification mutations" --repeat-each=20` | FAIL, 38 passed and 2 failed with the same unexpected `Slow category` projection |
| post-repair repeated classification-mutation regression command above | PASS, 40/40 across desktop and mobile projects |
| exact 11-spec quick-gate Playwright selection | PASS, 30 tests; 74 explicit capability-gated skips |
| `python3 scripts/check.py --quick` | PASS after replacing the scheduler-dependent classification test timer with an explicit request barrier; changed-area browser 30 passed/74 capability-gated skips and all static/backend/frontend checks passed |
| `python3 scripts/check.py --output docs/implementation/implemented/08.02_PHASE_CHECK.html` | PASS on clean rerun; frontend 537 tests, browser 309 pass/77 environment-gated skips, backend 87.4%, Phase 08 exact `4645/4986` (`93.2%`) coverage contract |
| `git diff --check` | PASS |

The original classification regression used fixed 200 ms/5 ms route-handler
timers. Under quick-gate contention, the slow handler could mutate the shared
stub before the latest mutation's authoritative GET was serviced, making the
test's assumed ordering false. The repair holds the slow request behind an
explicit promise, waits until the latest projection is visible, releases the
late mutation, proves that it reached the server-side stub, and then verifies
that its obsolete response does not overwrite the UI. No production behavior,
Task 286 report/UAT counts, requirement statuses, or findings changed.

The earlier full-gate attempt also reported a concurrent backend race-lane
nonzero result; its complete clean rerun passed every lane. The full gate's
single Redocly OAuth redirect warning remains the existing accepted warning.
The 77 aggregate browser skips are explicit opt-in/managed
real-stack/deployed scenarios; Tasks 281-285 were executed separately above
and their actual non-pass results were retained rather than skipped or
converted to success.

## SHA-256 evidence

| Path | SHA-256 |
|---|---|
| `docs/implementation/02_TASK_LIST.md` | `90b16c9e140e91f218f399f8e138e5ad380946e5e54af64d044b6843e11f1ab2` |
| `docs/implementation/04_OPEN.md` | `de951074a8dc70f49c01eedc99af2cf56c0d7b5894733c838190fa93defbdf15` |
| `docs/testing/phase08/acceptance-manifest.json` | `2f597b9545bb176e45a03df02d1b05f01fc424b479ffb4b64c43ee3b6991fa8c` |
| `docs/testing/phase08/finding-history.json` | `d18286a98a3e957cf7a2f4ece91c517337513c2b78a3c5dbd1a9983492d77e5b` |
| `docs/implementation/implemented/08_PHASE_UAT.md` | `371361f141ec2fe883e448e21480a63ddf9e8fa1d37910ea7fd9209bf511a9c0` |
| `docs/implementation/implemented/08.02_PHASE_UAT.md` | `ff1590298ef9d428cae09188a189f5f0303f33004403081d8bb08bb7af94f2a1` |
| `docs/implementation/implemented/08.02_PHASE_REPORT.json` | `aacc6c814fd35a9dc6df6fb59d0929e6c4a312b7e3ec56684938d13da71cb74d` |
| `docs/implementation/implemented/08.02_PHASE_REPORT.html` | `5e0823f47e752f91ac7b7de1803c1a0b4cb29c5bc77c2abfefe7b1bc901951d4` |
| `docs/implementation/implemented/08.02_PHASE_CHECK.html` | `13e4b47346f106419dca9d9ad20e4bb74f6aa9c1ac61acd1351af604667d0b8b` |
| `scripts/phase08_acceptance.py` | `9a551ff42880b9e452476e2ccfbdcc14bc4643c354cd88b2491387521c49795b` |
| `scripts/phase08_uat.py` | `2edce59eef387b2b8423111eb2b22d1a5723bbd0013c4cf2f915ff89a7172537` |
| `scripts/test_phase08_uat.py` | `2af8e9726092a375f741655ad3ad0c96bea6995b1969f2b9e871a1aeeb214b4e` |
| `scripts/check.py` | `5061b3594e71e5d96a9725b2942eba432f5d0a136538d7e3fedde886baefbd4f` |
| `docs/testing/phase08/uat-input.json` | `d14f7883bff4a4a4304e2b6f6f3c5168f905c06f54d309e2a846f75787b43980` |
| `docs/testing/phase08/uat-input.json-trace.md` | `28bf796af77d5260fb5b7b9b475ac7dde08c693214ad2d657da4d7d8d72cd8fb` |

Evidence-tree manifest: sort all 139 files by path and hash
`relative-path + NUL + binary SHA-256 digest`; resulting SHA-256:
`030cddbcdc77a1246ed1ae1b64f93c178d3ff8e6a559a16258717294ba5a808d`.
The complete per-file hashes are embedded in
`08.02_PHASE_REPORT.json`. This preparation document intentionally omits its
self-referential hash.

The 20 full-gate screenshot manifest digest is
`3079702cd3e221342bfdf7dc73f6d2e24d13e25ad320a6aee8bc9b3ee05bdaec`.

## Final disposition

Every Task 286 verification mechanism is implemented and current. The task is
ready for independent review as **PREPARED work**. This repair did not edit
`docs/implementation/02_TASK_LIST.md`. Phase 08.02 is **not ready for
project-owner acceptance** because 13 mandatory criteria fail and 11 remain
blocked, with no accepted deviations. The blocking finding recorded in
`task-286-final-review.md` is repaired at the producer-source, build-input,
decision, and CLI boundaries with exact regressions.
