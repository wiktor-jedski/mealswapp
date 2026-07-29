# Review Evidence: Task 286 — DESIGN-014 MetricsCollector

~~~yaml
task_id: 286
component: "Phase 08.02 Separate UAT and Acceptance Decision"
static_aspect: "DESIGN-014: MetricsCollector"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-29T16:09:05Z"
review_agent: "independent-task-286-final-reviewer"
evidence_file: "docs/implementation/evidence/task-286-final-review.md"
baseline_ref: "f9a646ffede5c8a63ad787a07eaba7efc0433677 plus current task-owned working-tree paths"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "Python, security review, and common-bugs guidance"
repair_context_required: true
~~~

## 1. Task Source

Description: Create separate Phase 08.02 UAT/report artifacts, preserve the historical UAT, trace Tasks 276-286 and all Task 280 criteria to Tasks 281-285 evidence, synchronize the complete open-finding ledger, keep the requirement gate truthful, and separate task delivery from project-owner acceptance.

Depends On: 281, 282, 283, 284, 285. All five dependency rows are currently PASSED.

Testing Coverage Exceptions: None for Task 286. Repository-wide documented coverage deviations remain enforced by the aggregate coverage contracts.

Verification Criteria: The new UAT and report are self-contained, preserve historical Phase 08 UAT unchanged, trace Tasks 276-286 and all required SW-REQ IDs, distinguish supporting from mandatory evidence, and match the Task 280 manifest/report plus 04_OPEN.md. Automated integrity tests must reject missing criteria, contradictory statuses, orphan evidence, stale findings, unaccepted mandatory non-pass criteria, console-only logging, and unsupported erasure claims. All scenarios must pass or have explicit owner-accepted deviations while the decision remains unchecked otherwise. UAT/report validation, task-list/traceability validation, sanitized-artifact scanning, and git diff --check must pass.

## 2. Pre-Review Gates

- [x] Input status is PREPARED.
- [x] Every dependency is PASSED or PREPARED.
- [x] The preparation report claims completion and records the final-review repair scope.
- [x] A task-specific baseline and task-owned working-tree surface are available; earlier Phase 08 work remains excluded.
- [x] code-review-skill was invoked exactly once and its Python/security/common-bugs guidance was read.
- [x] The reviewer is independent from implementation and repair.
- [x] Review uses current source, current hashes, and fresh commands.
- [x] Reviewer made no production-code changes and did not edit the task-list status.

~~~yaml
pre_review_gates_passed: true
blocking_issue: "NONE"
~~~

## 3. Review Baseline and Change Surface

Baseline/reference method: HEAD is f9a646ffede5c8a63ad787a07eaba7efc0433677. The worktree was already dirty with Tasks 276-285 and shared-gate changes. Task 286 ownership follows the preparation report: scripts/phase08_uat.py, its focused tests, the Task 286 registration in scripts/check.py, and the committed UAT/report/evidence artifacts. This repair additionally covers the rejected quick-gate assertion in frontend/tests/admin-data-management.spec.ts. The PREPARED Task 286 row and its status transition predate this review; the row was read but not edited.

Commands used to reconstruct the surface:

    git status --short --untracked-files=all
    git rev-parse HEAD
    git diff -- scripts/check.py
    rg -n '^(class|def) |^    def ' scripts/phase08_uat.py scripts/test_phase08_uat.py
    rg -n 'phase08|Task 286' scripts/check.py
    sha256sum on every reviewed implementation and report/control artifact

Pre-existing dirty-worktree changes and exclusions: Application/backend/frontend work and Tasks 276-285 files are outside this review except for the shared Phase 08 acceptance registration and the repaired stale-category assertion. The current task-list diff contains the pre-existing OPEN-to-PREPARED transition for Task 286; this review did not create or alter it. Generated JSON/HTML/UAT, sanitized evidence, run-context evidence, and screenshots are reviewed as committed contract/evidence artifacts, not as executable symbols.

| Changed file | Change source | Task-owned confidence | Symbols or units discovered |
|---|---|---|---|
| scripts/phase08_uat.py | Task 286 acceptance builder and validator | HIGH | UATError plus 25 functions |
| scripts/test_phase08_uat.py | Task 286 adversarial regression suite | HIGH | Phase08UATTests plus 28 executable methods |
| scripts/check.py | shared static-lane Task 286 registration | MEDIUM | validation function and registration/configuration |
| frontend/tests/admin-data-management.spec.ts | rejected quick-gate stale-category assertion repair | HIGH | post-fulfill completion barrier in the delayed classification route |
| docs/testing/phase08/uat-input.json and sidecar | Task 286 input contract and traceability | HIGH | declarative JSON/config artifacts |
| docs/implementation/implemented/08.02_PHASE_REPORT.json | Task 286 final machine report | HIGH | generated report artifact |
| docs/implementation/implemented/08.02_PHASE_REPORT.html | Task 286 rendered report | HIGH | generated report artifact |
| docs/implementation/implemented/08.02_PHASE_UAT.md | Task 286 owner UAT | HIGH | generated UAT artifact |
| docs/implementation/implemented/08.02_PHASE_EVIDENCE/ | sanitized copied evidence tree | HIGH | generated evidence artifact tree |
| docs/implementation/implemented/screenshots/08.02_PHASE_CHECK-* | full-gate UAT screenshots | HIGH | generated evidence artifacts |

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Separate UAT/report artifacts are self-contained and the historical 08_PHASE_UAT.md is unchanged. | Current UAT/report inspection, historical SHA-256, final validator. | PASS | Historical hash is 371361f141ec2fe883e448e21480a63ddf9e8fa1d37910ea7fd9209bf511a9c0 and matches the report and fresh validator. |
| 2 | Tasks 276-286, all required SW-REQ IDs, source reports, traces, screenshots, backend evidence, and log-sink evidence are linked. | Report tree inspection, source aggregation, evidence hashes, task-list and traceability validators. | PASS | 91 criteria aggregate from Tasks 281-285; task trace covers 276-286; all 12 required SW-REQ IDs are present; validators pass. |
| 3 | Supporting mocked/component evidence is distinct from mandatory real-stack/deployed evidence and the requirement gate remains truthful. | UAT/report text, source projection, status recomputation, special-evidence checks. | PASS | The report preserves FAIL/BLOCKED observations and rejects console-only logging or erasure without worker/backend evidence. |
| 4 | Integrity validation rejects malformed nested types before every set/dict/hash/path/status operation, including exact result.criterionId, source.report, and source.status regressions, broad malformed values, and CLI no-traceback behavior. | 28 focused tests, exact source-report/build-input probes, and CLI regressions. | PASS | Source result `criterionId=[]` and `status=[]`, build `historicalUat=[]`, `reports=[[]]`, report path `=[]`, decision status `=[]`, and deviation `criterionId=[]` all raise UATError; real CLI invocations return one structured exit-1 diagnostic without Traceback. |
| 5 | The report preserves 67 PASS, 13 FAIL, and 11 BLOCKED; it lists 14 findings and leaves project-owner acceptance PENDING/unchecked. | Direct report inspection and current validator. | PASS | Counts are exactly 67/13/11, gateStatus is FAIL, openFindings contains 14 IDs, and acceptanceDecision is PENDING with empty owner/date/deviations. |
| 6 | UAT/report, task-list, traceability, sanitized-artifact scan, cleanup, quick, and full gates pass. | Fresh commands, full-gate report, cleanup probes, hashes. | PASS | The stale-category assertion passes 40/40 focused stress iterations across desktop and mobile. Two consecutive aggregate quick gates pass, each with 30 changed-area browser passes and 74 intentional skips. Task-286 static/acceptance lanes, validators, sanitized scan, cleanup, diff check, and the preserved full gate pass. The full gate reports 537 frontend tests, 309 browser passes/77 skips, race/vulnerability/coverage lanes, 87.4% backend coverage, and 93.2% Phase 08 coverage. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | UATError | exception type | scripts/phase08_uat.py:127 | added | all validation and CLI paths | all focused tests |
| 2 | sha256 | function | scripts/phase08_uat.py:131 | added | evidence/hash validators | hash and committed-validator tests |
| 3 | relative | function | scripts/phase08_uat.py:147 | added | JSON errors, report rendering, hashes | validator paths |
| 4 | repo_path | function | scripts/phase08_uat.py:156 | added | input/hash/final path validation | malformed path tests |
| 5 | load_json | function | scripts/phase08_uat.py:167 | added | controls, source, input, final report loading | validator and CLI tests |
| 6 | load_controls | function | scripts/phase08_uat.py:176 | added | build and final validation | committed-validator tests |
| 7 | task_rows | function | scripts/phase08_uat.py:191 | added | build/final task trace | current report test |
| 8 | expected_report_status | function | scripts/phase08_uat.py:217 | added | source report validation | source aggregation paths |
| 9 | validate_string_list | function | scripts/phase08_uat.py:228 | added | IDs, request IDs, backend summaries, findings | malformed list tests |
| 10 | validate_backend_evidence | function | scripts/phase08_uat.py:244 | added | source/final evidence validation | malformed backend tests |
| 11 | validate_source_report | function | scripts/phase08_uat.py:257 | added and repaired | build and canonical_source_reports | source result criterionId/status regressions |
| 12 | canonical_source_reports | function | scripts/phase08_uat.py:422 | added and reviewed | validate_final trusted-source reconstruction | final validator and source CLI regression |
| 13 | validate_decision | function | scripts/phase08_uat.py:454 | added and repaired | build and final validation | decision/deviation tests |
| 14 | valid_date | function | scripts/phase08_uat.py:509 | added | decision/deviation validation | calendar-date tests |
| 15 | validate_build_input | function | scripts/phase08_uat.py:518 | added and repaired | build CLI | malformed build-input matrix |
| 16 | aggregate | function | scripts/phase08_uat.py:574 | added | build and final trusted aggregation | status weakening tests |
| 17 | verify_special_evidence | function | scripts/phase08_uat.py:626 | added | build and final acceptance checks | logging/erasure tests |
| 18 | copy_report_tree | function | scripts/phase08_uat.py:647 | added | build evidence publication | build/evidence paths |
| 19 | evidence_hashes | function | scripts/phase08_uat.py:659 | added | build report and task trace | validator/hash checks |
| 20 | render_html | function | scripts/phase08_uat.py:668 | added | build report output | committed HTML artifact |
| 21 | render_uat | function | scripts/phase08_uat.py:694 | added | build UAT output | committed UAT artifact |
| 22 | build | function | scripts/phase08_uat.py:785 | added | build CLI | validator and gate |
| 23 | validate_hash_rows | function | scripts/phase08_uat.py:881 | added | final schema/final validation | malformed hash/path tests |
| 24 | validate_final_schema | function | scripts/phase08_uat.py:899 | added and repaired | validate_final | broad nested final-tree tests |
| 25 | validate_final | function | scripts/phase08_uat.py:1034 | added and repaired | command_validate/tests | focused and trusted-source tests |
| 26 | command_validate | function | scripts/phase08_uat.py:1152 | added | main | CLI validator |
| 27 | main | function | scripts/phase08_uat.py:1160 | added | module entry point | CLI no-traceback tests |
| 28 | Phase08UATTests | test class | scripts/test_phase08_uat.py:21 | added | unittest discovery | 25 test methods |
| 29 | setUpClass | test fixture method | scripts/test_phase08_uat.py:25 | added | unittest lifecycle | all tests |
| 30 | validate | test helper method | scripts/test_phase08_uat.py:29 | added | mutation tests | all validator mutations |
| 31 | recount | test helper method | scripts/test_phase08_uat.py:36 | added | status mutation tests | weakening tests |
| 32 | test_committed_report_is_current_and_complete | test method | scripts/test_phase08_uat.py:41 | added | unittest discovery | current report |
| 33 | test_missing_criterion_is_rejected | test method | scripts/test_phase08_uat.py:51 | added | unittest discovery | missing criterion |
| 34 | test_weakened_or_contradictory_status_is_rejected | test method | scripts/test_phase08_uat.py:57 | added | unittest discovery | status weakening |
| 35 | test_malformed_nested_source_types_are_rejected | test method | scripts/test_phase08_uat.py:65 | added | unittest discovery | nested final source types |
| 36 | test_list_valued_hash_operands_are_bounded_validation_errors | test method | scripts/test_phase08_uat.py:78 | added | unittest discovery | exact result/source list regressions |
| 37 | test_all_nested_operation_fields_are_type_guarded | test method | scripts/test_phase08_uat.py:94 | added | unittest discovery | broad final nested mutations |
| 38 | test_cli_rejects_malformed_nested_values_without_traceback | test method | scripts/test_phase08_uat.py:153 | added | unittest discovery | CLI final-report regression |
| 39 | test_duplicate_sources_are_rejected | test method | scripts/test_phase08_uat.py:173 | added | unittest discovery | duplicate source identity |
| 40 | test_coordinated_fail_source_and_result_weakening_is_rejected | test method | scripts/test_phase08_uat.py:181 | added | unittest discovery | FAIL weakening |
| 41 | test_coordinated_blocked_source_and_result_weakening_is_rejected | test method | scripts/test_phase08_uat.py:194 | added | unittest discovery | BLOCKED weakening |
| 42 | test_nonpass_without_synchronized_finding_is_rejected | test method | scripts/test_phase08_uat.py:207 | added | unittest discovery | finding parity |
| 43 | test_closed_finding_without_exact_retest_evidence_is_rejected | test method | scripts/test_phase08_uat.py:214 | added | unittest discovery | closure evidence |
| 44 | test_orphan_source_evidence_is_rejected | test method | scripts/test_phase08_uat.py:220 | added | unittest discovery | orphan evidence |
| 45 | test_source_result_hash_operands_are_bounded_validation_errors | test method | scripts/test_phase08_uat.py:234 | added and repaired | unittest discovery | source criterionId/status regressions |
| 46 | test_malformed_build_input_is_bounded_before_operations | test method | scripts/test_phase08_uat.py:251 | added and repaired | unittest discovery | build input regression matrix |
| 47 | test_build_cli_rejects_malformed_input_and_source_without_traceback | test method | scripts/test_phase08_uat.py:290 | added and repaired | unittest discovery | build CLI no-traceback regressions |
| 48 | test_all_open_findings_are_required | test method | scripts/test_phase08_uat.py:332 | added | unittest discovery | 14-finding parity |
| 49 | test_recursive_sensitive_evidence_metadata_is_rejected | test method | scripts/test_phase08_uat.py:338 | added | unittest discovery | recursive source metadata |
| 50 | test_recursive_sensitive_final_report_metadata_is_rejected | test method | scripts/test_phase08_uat.py:357 | added | unittest discovery | recursive final metadata |
| 51 | test_normalized_sensitive_aliases_are_rejected_at_any_final_depth | test method | scripts/test_phase08_uat.py:365 | added | unittest discovery | normalized aliases |
| 52 | test_invalid_calendar_dates_are_rejected | test method | scripts/test_phase08_uat.py:375 | added | unittest discovery | real calendar dates |
| 53 | test_acceptance_without_owner_and_date_is_rejected | test method | scripts/test_phase08_uat.py:386 | added | unittest discovery | owner/date |
| 54 | test_mandatory_nonpass_prevents_plain_acceptance | test method | scripts/test_phase08_uat.py:392 | added | unittest discovery | mandatory non-pass |
| 55 | test_incomplete_accepted_deviations_are_rejected | test method | scripts/test_phase08_uat.py:403 | added | unittest discovery | deviation completeness |
| 56 | test_centralized_logging_cannot_pass_from_blocked_console_evidence | test method | scripts/test_phase08_uat.py:414 | added | unittest discovery | log sink boundary |
| 57 | test_erasure_pass_requires_worker_backend_evidence | test method | scripts/test_phase08_uat.py:427 | added | unittest discovery | erasure evidence |
| 58 | test_historical_phase_uat_hash_is_immutable | test method | scripts/test_phase08_uat.py:438 | added | unittest discovery | historical hash |
| 59 | test_uat_requires_unchecked_owner_decision_fields | test method | scripts/test_phase08_uat.py:444 | added | unittest discovery | UAT decision fields |
| 60 | validate_phase08_acceptance_contracts | gate function | scripts/check.py:672 | modified | static lane | quick/full gates |
| 61 | Phase 08 acceptance gate registration and TRACEABLE_FILES entries | gate configuration | scripts/check.py:759-764, 877 | modified | static lane and traceability | quick/full gates |

~~~yaml
inventory_source_count: 61
audited_symbol_count: 61
inventory_complete: true
generated_groupings:
  - "The generated JSON/HTML/UAT/evidence/screenshot artifacts are audited as contract outputs and hashed below; no executable generated unit is hidden in a grouping."
~~~

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| UATError | Bounded validation exception for incomplete or contradictory evidence. | Intentional error type; callers convert it to safe diagnostics. | Stateless; N/A for cancellation/concurrency. | Does not carry private exception details by itself. | O(1). | Idiomatic ValueError subclass. | Focused and CLI probes observe it on malformed source/input/final data. | PASS |
| sha256 | Hash only regular in-repository files with no symlink component. | Missing, escaping, symlink, and non-file paths reject. | Read-only file I/O; no retained resource; N/A for cancellation. | Prevents evidence path escape and symlink substitution. | Reads each bounded artifact once. | Small helper and standard hashlib. | Hash/path mutations pass. | PASS |
| relative | Return repository-relative POSIX evidence path. | Outside-root paths reject. | No shared state. | Keeps diagnostics and references repository scoped. | O(1) path resolution. | Minimal helper. | Indirect validator coverage. | PASS |
| repo_path | Accept only strict relative POSIX strings. | Non-string/absolute/traversal values reject before filesystem use. | No resource state. | Path traversal and absolute-path boundary is explicit. | O(parts). | Idiomatic PurePosixPath use. | Final and build-input malformed path probes pass. | PASS |
| load_json | Load duplicate-key-free JSON objects. | JSON/schema errors become UATError; object shape comes from strict loader. | File read only; no cancellation. | Duplicate-key rejection supports integrity. | Bounded by artifact inputs but no explicit size cap. | Reuses Task 280 loader. | Final and control loads pass. | PASS |
| load_controls | Validate manifest, history, and open ledger together. | Dependency validation errors are wrapped as UATError. | Read-only files. | Trusted finding/criterion controls define later comparisons. | Fixed 91-criterion control set. | Good reuse of Task 280 source of truth. | Validators pass. | PASS |
| task_rows | Parse exact Tasks 276-286 without editing the task list. | Missing/malformed row count rejects; duplicate row IDs are not explicitly rejected. | Read-only file state. | Task status trace is a control boundary. | Linear file scan. | Simple parser; duplicate-ID gap is defense-in-depth. | Task-list validator passes current file. | PASS |
| expected_report_status | Map PASS/BLOCKED/FAIL to truthful status/exit code. | Called only after every source result status is a validated string; rank mapping is exhaustive. | Stateless. | Prevents status weakening after validation. | O(results). | Clear rank policy. | Current source reports and malformed source status probe pass. | PASS |
| validate_string_list | Require sorted unique string arrays and optional regex. | Non-list, nested, non-string, unsorted, duplicate values reject before set. | Stateless. | ID/request/backend fields remain bounded. | O(n log n). | Correct short-circuit type guard. | Broad final list mutations pass. | PASS |
| validate_backend_evidence | Enforce safe summary keys and syntax. | Wrong type or unsupported summary rejects. | Stateless. | Backend output cannot carry raw payloads. | Small bounded list. | Reuses common list helper. | Backend malformed test passes. | PASS |
| validate_source_report | Validate source schema, result linkage, evidence, findings, status, and orphan files. | Exact top-level/result/root/evidence contracts guard criterionId/status and every set/dict/path/status operand; malformed values become UATError. | Read-only report tree; no cancellation/concurrency. | Untrusted producer reports fail closed and sensitive metadata is rejected recursively. | Bounded by the selected report/evidence tree. | Central contract is cohesive and now complete at the producer boundary. | Direct `criterionId=[]` and `status=[]` probes both return UATError; orphan/sensitive tests pass. | PASS |
| canonical_source_reports | Select exactly Tasks 281-285 copied reports and validate them. | Validates selected input shape before task/path/set operations and delegates to guarded source validation. | Read-only source selection and sets. | Trusted-source reconstruction prevents final projection weakening. | Linear selected reports. | Good canonical-source design. | Final coordinated weakening and malformed source CLI tests pass. | PASS |
| validate_decision | Enforce PENDING/accepted/deviation ownership and real dates. | Guards decision/result/deviation scalar types before status or criterion set membership; malformed build decision inputs become UATError. | Stateless. | Prevents unowned mandatory acceptance. | O(criteria + deviations). | Clear acceptance state machine. | Final decision/date plus malformed status/criterion probes pass. | PASS |
| valid_date | Accept only exact real ISO calendar dates. | Non-string and impossible dates return false. | Stateless. | Prevents false dated deviations. | O(1). | Correct stdlib date use. | Impossible-date test passes. | PASS |
| validate_build_input | Exact-schema/type-check fresh UAT input before dictionary, status, set, hash, path, or publication operations. | Rejects malformed root/history/reports/support/task-evidence/decision/deviation values with UATError. | Stateless; runs before any evidence copy or generated write. | Input is an untrusted publication boundary; paths remain repository-relative. | Linear over bounded configured lists. | Small explicit schema gate. | Five direct malformed-input cases and real CLI no-traceback probes pass. | PASS |
| aggregate | Preserve all source observations and strongest status. | Receives source reports already validated by `validate_source_report`; strongest status is exhaustive and cannot weaken failures. | Stateless output construction. | FAIL cannot be weakened by duplicate PASS evidence. | Linear bounded source criteria. | Explicit strongest-status aggregation. | Coordinated FAIL/BLOCKED tests pass. | PASS |
| verify_special_evidence | Enforce independent deployed logging and worker/backend erasure evidence. | Validated result/source shapes required; unsupported direct input can raise. | Stateless. | Blocks console-only and browser-only claims. | Linear result/evidence scan. | Focused security boundary. | Logging and erasure tests pass. | PASS |
| copy_report_tree | Publish one validated source tree through temporary staging. | Copy failure occurs before replacement; existing destination is removed only after copy succeeds. | TemporaryDirectory cleans staging on errors; N/A for cancellation. | Destination is derived from input and needs the upstream strict path boundary. | Copies each selected tree once. | Atomic same-parent replacement is appropriate. | Build artifacts and cleanup probes pass. | PASS |
| evidence_hashes | Produce deterministic path/digest rows. | Assumes Path inputs; malformed caller values can fail before UATError. | Read-only files. | Hashes bind evidence to contents. | Sort/set over bounded path list. | Deterministic helper. | Current hash validation passes. | PASS |
| render_html | Render the 91-criterion report. | Trusted validated report fields are escaped where HTML-visible. | Pure string generation. | html.escape protects visible report values. | O(criteria). | Simple renderer. | Current HTML artifact validates. | PASS |
| render_uat | Render UAT, task trace, findings, and owner decision fields. | Trusted report fields only; malformed direct caller is outside its contract. | Pure string generation. | Explicit unchecked decision prevents false acceptance. | O(criteria + tasks). | Readable output. | UAT artifact and decision tests pass. | PASS |
| build | Validate fresh input, copy evidence, aggregate, render, write, and revalidate. | `validate_build_input` rejects malformed input before path/hash/publication; source and final validators reject malformed reports with UATError. | Temporary staging is context-managed and no staging directory remains on failure; no cancellation/concurrency needed. | Input JSON and producer report files are integrity boundaries. | Linear bounded artifacts and evidence copies. | Cohesive orchestration with explicit preflight validation. | Fresh malformed build matrix and CLI probes all return bounded failures; valid committed report rebuild/validation remains current. | PASS |
| validate_hash_rows | Validate exact path/hash rows and current content. | Dict/string guards precede repo_path and sha256. | Read-only file I/O. | Prevents stale or escaped evidence. | Linear hashes and file reads. | Good strict helper. | Broad hash/path mutations pass. | PASS |
| validate_final_schema | Type-check the complete final report before keyed/status/path/hash operations. | Final nested fields, exact schemas, strings, IDs, lists, task IDs, hashes, decision and deviations are guarded. | Stateless; file hashes are delegated after shape checks. | Final report boundary fails closed for malformed values. | Linear bounded tree plus hashes. | Repair is systematic and readable. | Exact result.criterionId/source.report/source.status, broad malformed, and CLI tests pass. | PASS |
| validate_final | Recompute final results from trusted source reports and enforce parity, hashes, findings, special evidence, history, and UAT text. | Final projection and delegated source reports are fully guarded; trusted results/counts/gate are recomputed from validated evidence. | Read-only; no cancellation/concurrency needed. | Strong anti-weakening, sensitive-value, and historical-UAT controls. | Multiple bounded scans and hashes. | Correct trust/recompute architecture. | Current report, exact malformed final/source probes, and coordinated weakening tests pass. | PASS |
| command_validate | Run committed report/UAT validation. | Loads strict objects and delegates to guarded final validation; expected invalid states become UATError. | Read-only. | CLI is intended to fail closed without private traceback. | Bounded report validation. | Small command wrapper. | Committed validator and CLI malformed-value probes pass. | PASS |
| main | Provide build/validate CLI with structured nonzero diagnostics. | Catches OSError, UnicodeError, UATError, and Task 280 ValidationError; all challenged malformed source/input cases enter this bounded path. | No external long-running resources. | Diagnostic boundary emits one safe nonzero line without traceback. | O(validator). | Idiomatic narrow exception boundary for typed validation. | Final, source, and build CLI probes pass. | PASS |
| Phase08UATTests | Own 28 Task 286 adversarial tests and current-artifact assertions. | Tests pass and use isolated deep copies/temp directories for malformed reports and inputs. | Context managers clean temporary report trees and no run-owned process/staging remains. | Covers final, producer-source, build-input, CLI, sensitive, erasure, and log-sink boundaries. | Fast focused suite. | Standard unittest class. | 28 Task 286 tests pass. | PASS |
| setUpClass | Load committed report and UAT once for test fixtures. | Current artifact read failures fail setup. | Class-scoped immutable fixture data. | Uses committed evidence only. | One JSON/text read. | Simple. | Current report test passes. | PASS |
| validate | Run complete final validator against copied mutations. | UATError is asserted; raw TypeError would fail the test. | Deep copy prevents shared mutation. | Exercises final integrity boundary. | Bounded test data. | Useful helper. | Exact final mutations pass. | PASS |
| recount | Recompute mutated counts/gate to reach deeper checks. | Assumes mutation values are valid statuses; test-only helper. | Mutates isolated test copy only. | Cannot influence production report. | O(results). | Narrowly scoped. | Weakening tests pass. | PASS |
| test_committed_report_is_current_and_complete | Prove current 91-result report, counts, findings, and PREPARED trace. | Validator and exact assertions fail on stale output. | Read-only fixture. | Binds task status and requirement report. | O(report). | Direct acceptance test. | Passes with 67/13/11 and 14 findings. | PASS |
| test_missing_criterion_is_rejected | Reject incomplete manifest coverage. | UATError expected. | Isolated copy. | Prevents omission. | O(report). | Direct negative test. | Passes. | PASS |
| test_weakened_or_contradictory_status_is_rejected | Reject final status weakening. | UATError expected after recomputed counts. | Isolated copy. | Prevents false PASS. | O(report). | Direct negative test. | Passes. | PASS |
| test_malformed_nested_source_types_are_rejected | Reject malformed final nested task/source/request/backend values. | UATError expected. | Isolated copy. | Final source boundary. | O(report). | Direct negative test. | Passes, but source report files are not mutated. | PASS |
| test_list_valued_hash_operands_are_bounded_validation_errors | Exact result.criterionId, source.report, and source.status list regressions. | UATError expected, no raw TypeError. | Isolated copy. | Final set/dict/path/status boundary. | O(3 mutations). | Precise regression. | Passes exactly at final-report boundary. | PASS |
| test_all_nested_operation_fields_are_type_guarded | Broad malformed final count/list/object/scalar coverage. | UATError expected for 30 mutations. | Isolated copies. | Final nested operation safety. | O(mutations × report). | Comprehensive targeted test. | Passes; does not include source producer reports or build input. | PASS |
| test_cli_rejects_malformed_nested_values_without_traceback | CLI must emit one bounded diagnostic for malformed final report. | Exit 1, structured stderr, no Traceback. | Temporary report/UAT files cleaned. | Prevents diagnostics leak. | O(one CLI invocation). | Direct CLI regression. | Passes for final projection criterionId. | PASS |
| test_duplicate_sources_are_rejected | Reject repeated final source identity. | UATError expected. | Isolated copy. | Prevents ambiguous provenance. | O(sources). | Direct negative test. | Passes. | PASS |
| test_coordinated_fail_source_and_result_weakening_is_rejected | Recompute a source FAIL and final result mutation. | UATError expected. | Isolated copy. | Prevents coordinated false PASS. | O(report). | Strong regression. | Passes. | PASS |
| test_coordinated_blocked_source_and_result_weakening_is_rejected | Preserve BLOCKED strongest status. | UATError expected. | Isolated copy. | Prevents blocked-to-pass weakening. | O(report). | Strong regression. | Passes. | PASS |
| test_nonpass_without_synchronized_finding_is_rejected | Require open finding linkage. | UATError expected. | Isolated copy. | Keeps failure ledger truthful. | O(report/findings). | Direct negative test. | Passes. | PASS |
| test_closed_finding_without_exact_retest_evidence_is_rejected | Require exact closure evidence. | UATError expected. | Isolated copy. | Prevents deletion of historical defects. | O(findings). | Direct negative test. | Passes. | PASS |
| test_orphan_source_evidence_is_rejected | Reject unreferenced source files. | UATError expected. | Isolated copied tree. | Evidence completeness. | O(tree). | Direct negative test. | Passes. | PASS |
| test_source_result_hash_operands_are_bounded_validation_errors | Reject list-valued producer result criterionId/status before dictionary/set/status operations. | UATError expected. | Isolated copied source tree; context manager cleans it. | Protects the untrusted producer boundary. | O(2 source mutations). | Precise regression. | Both exact mutations pass. | PASS |
| test_malformed_build_input_is_bounded_before_operations | Reject malformed historical/report/path/decision/deviation input before publication operations. | UATError expected. | Temporary input and any staged artifacts are scoped. | Protects build publication and decision boundary. | O(5 input mutations). | Direct preflight regression. | All five direct cases pass. | PASS |
| test_build_cli_rejects_malformed_input_and_source_without_traceback | Emit one structured nonzero diagnostic for malformed build input/source. | Exit 1 and no Traceback. | Temporary input/source trees are cleaned. | Prevents diagnostic leakage. | O(6 subprocess-equivalent invocations). | Direct CLI regression. | All source/input cases pass. | PASS |
| test_all_open_findings_are_required | Require all 14 current unresolved ledger IDs. | UATError expected for omission. | Isolated copy. | No hidden unresolved defect. | O(findings). | Direct parity test. | Passes. | PASS |
| test_recursive_sensitive_evidence_metadata_is_rejected | Scan nested source evidence metadata and aliases. | UATError expected. | Isolated copy. | Sensitive artifact boundary. | O(nested value). | Direct security test. | Passes. | PASS |
| test_recursive_sensitive_final_report_metadata_is_rejected | Scan nested final metadata. | UATError expected. | Isolated copy. | Sensitive report boundary. | O(nested value). | Direct security test. | Passes. | PASS |
| test_normalized_sensitive_aliases_are_rejected_at_any_final_depth | Reject punctuation/case aliases at arbitrary final depth. | UATError expected. | Isolated copies. | Sensitive-key normalization. | Bounded depth probes. | Good adversarial breadth. | Passes. | PASS |
| test_invalid_calendar_dates_are_rejected | Reject impossible dates. | UATError expected. | Isolated copy. | Owner/deviation accountability. | O(1). | Direct negative test. | Passes. | PASS |
| test_acceptance_without_owner_and_date_is_rejected | Reject ownerless acceptance. | UATError expected. | Isolated copy. | Accountability gate. | O(report). | Direct negative test. | Passes. | PASS |
| test_mandatory_nonpass_prevents_plain_acceptance | Reject acceptance with non-pass criteria. | UATError expected. | Isolated copy. | Prevents false requirement acceptance. | O(criteria). | Direct state-machine test. | Passes. | PASS |
| test_incomplete_accepted_deviations_are_rejected | Require complete deviation records. | UATError expected. | Isolated copy. | Owner/retest/expiry boundary. | O(deviations). | Direct negative test. | Passes. | PASS |
| test_centralized_logging_cannot_pass_from_blocked_console_evidence | Reject console-only SW-REQ-084 PASS. | UATError expected. | Isolated copy. | Independent sink evidence. | O(evidence). | Direct security test. | Passes. | PASS |
| test_erasure_pass_requires_worker_backend_evidence | Reject browser-only erasure PASS. | UATError expected. | Isolated copy. | High-risk deletion evidence. | O(evidence). | Direct security test. | Passes. | PASS |
| test_historical_phase_uat_hash_is_immutable | Reject historical UAT rewrite. | UATError expected. | Isolated copy/text. | Historical audit integrity. | One hash. | Direct integrity test. | Passes. | PASS |
| test_uat_requires_unchecked_owner_decision_fields | Require unchecked UAT acceptance fields until owner action. | UATError expected. | Isolated text. | Prevents accidental acceptance. | O(text). | Direct artifact test. | Passes. | PASS |
| validate_phase08_acceptance_contracts | Register Task 280 plus Task 286 focused/committed validators. | Child nonzero exits propagate through gate runner. | Runs as static-lane step; no owned resources. | Ensures integrity checks cannot be skipped. | Runs bounded focused suites and validators. | Correct shared-gate wiring. | Quick/full gates pass. | PASS |
| Phase 08 acceptance gate registration and TRACEABLE_FILES entries | Make changed Task 286 Python files traceable and run the validator. | Missing registration would skip checks; current entries are present. | Static configuration only. | Traceability boundary. | No runtime allocation. | Declarative configuration. | Quick/full and traceability validators pass. | PASS |

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| none | N/A | N/A | No unresolved blocking, important, or optional finding remains in the Task 286-owned surface after repair. | Fresh source-report, build-input, final-report, and CLI probes all fail closed with UATError or one structured diagnostic; current artifacts and trusted aggregation validate. | No repair required. |

~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 0
~~~

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| python3 -m unittest -v scripts/test_phase08_uat.py scripts/test_phase08_acceptance.py | repo | 0 | PASS: 53 tests (28 Task 286, 25 Task 280) | focused acceptance suites |
| python3 -m py_compile scripts/phase08_uat.py scripts/test_phase08_uat.py scripts/check.py | repo | 0 | PASS | Python syntax gate |
| python3 scripts/phase08_uat.py validate | repo | 0 | PASS: 91 criteria, current hashes | committed validator |
| python3 scripts/phase08_acceptance.py validate | repo | 0 | PASS: 12 scenarios, 91 criteria | manifest validator |
| python3 scripts/validate-task-list.py | repo | 0 | PASS: 286 sequential tasks | task-list validator |
| python3 scripts/validate-traceability.py | repo | 0 | PASS | traceability validator |
| git diff --check | repo | 0 | PASS | whitespace check |
| BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/admin-data-management.spec.ts --grep "older classification mutations and refreshes" --repeat-each=20 | frontend | 0 | PASS: 40/40, 20 desktop and 20 mobile iterations | focused deterministic stress |
| python3 scripts/check.py --quick (run 1) | repo | 0 | PASS: all static and changed-area lanes; 30 browser passes, 74 intentional skips | first consecutive quick gate |
| python3 scripts/check.py --quick (run 2) | repo | 0 | PASS: all static and changed-area lanes; 30 browser passes, 74 intentional skips | second consecutive quick gate |
| python3 scripts/check.py --output docs/implementation/implemented/08.02_PHASE_CHECK.html | repo | 1 | TRANSIENT: first run backend local-stack lane overlapped a concurrent quick run; browser completed 309/309 with 77 skips | `/tmp/mealswapp-task286-full.log` |
| python3 scripts/verify-local-stack.py --keep-services | repo | 0 | PASS: local PostgreSQL/Redis/API/worker health and readiness | independent rerun after transient full-lane failure |
| python3 scripts/check.py --output docs/implementation/implemented/08.02_PHASE_CHECK.html | repo | 0 | PASS: full static/backend/frontend/browser lanes; 537 frontend tests, 309 browser passes, 77 skips, race, coverage, vulnerability, and artifact lanes | `docs/implementation/implemented/08.02_PHASE_CHECK.html` |
| exact source/build-input malformed guard probe | repo | 0 | PASS: source criterionId/status and five build-input mutations each raise UATError | fresh reviewer probe |
| exact source/build-input CLI no-traceback probe | repo | 0 | PASS: six real CLI invocations exit 1 with one structured diagnostic and no Traceback | fresh reviewer probe |
| python3 artifact sensitive-value/symlink scan | repo | 0 | PASS: 139 evidence files; 128 UTF-8 text files clean, 11 image files skipped, no symlinks | sanitized evidence scan |
| cleanup process/staging/container probe | repo | 0 | PASS after gates: no Task 281-286 process, UAT staging directory, or run-owned container; only expected shared PostgreSQL/Redis services remain | cleanup probe |
| sha256sum reviewed files and controls | repo | 0 | PASS: fresh hashes recorded below | current content fingerprints |
| python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/evidence/task-286-final-review.md | repo | 0 | PASS: structural review evidence | this file |

## 9. Files Inspected and Staleness Fingerprints

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| scripts/phase08_uat.py | Task 286 source validator, builder, renderer, and CLI | repaired source/input guards pass | SHA256 | 2edce59eef387b2b8423111eb2b22d1a5723bbd0013c4cf2f915ff89a7172537 |
| scripts/test_phase08_uat.py | Task 286 focused regressions | final, source, and build-input boundaries pass | SHA256 | 2af8e9726092a375f741655ad3ad0c96bea6995b1969f2b9e871a1aeeb214b4e |
| scripts/check.py | shared Task 286 gate registration | no finding | SHA256 | 5061b3594e71e5d96a9725b2942eba432f5d0a136538d7e3fedde886baefbd4f |
| frontend/tests/admin-data-management.spec.ts | changed-area stale-category regression | post-fulfill barrier passes desktop/mobile stress | SHA256 | 78cbd6337dc669b4e6746569b35828929386f0c3da0c005bf2552c63ca626e41 |
| docs/implementation/02_TASK_LIST.md | task status and trace source | authorized post-review Task 286 PASSED state; not edited in review | SHA256 | 3339d96fa4da0c6e2e0174ab0aa3da9544dce93dff99006eb1e624b1c536df59 |
| docs/implementation/04_OPEN.md | synchronized Phase 08 findings | current 14 unresolved IDs | SHA256 | de951074a8dc70f49c01eedc99af2cf56c0d7b5894733c838190fa93defbdf15 |
| docs/testing/phase08/acceptance-manifest.json | 91-criterion source contract | current manifest | SHA256 | 2f597b9545bb176e45a03df02d1b05f01fc424b479ffb4b64c43ee3b6991fa8c |
| docs/testing/phase08/finding-history.json | append-only finding registry | current historical IDs | SHA256 | d18286a98a3e957cf7a2f4ece91c517337513c2b78a3c5dbd1a9983492d77e5b |
| docs/testing/phase08/uat-input.json | selected source/evidence input | current input | SHA256 | d14f7883bff4a4a4304e2b6f6f3c5168f905c06f54d309e2a846f75787b43980 |
| docs/testing/phase08/uat-input.json-trace.md | JSON sidecar traceability | current sidecar | SHA256 | 28bf796af77d5260fb5b7b9b475ac7dde08c693214ad2d657da4d7d8d72cd8fb |
| docs/implementation/implemented/08_PHASE_UAT.md | historical UAT | required immutable baseline | SHA256 | 371361f141ec2fe883e448e21480a63ddf9e8fa1d37910ea7fd9209bf511a9c0 |
| docs/implementation/implemented/08.02_PHASE_UAT.md | separate owner UAT | current unchecked PENDING artifact | SHA256 | 6d913897fb5e26f279326408906fa64d3514859374ab3a10fedac69c4d5fb7e8 |
| docs/implementation/implemented/08.02_PHASE_REPORT.json | final 91-criterion report | current 67/13/11 and 14 findings | SHA256 | 0a4b91480f34fe8e7793897c21c2af59055e0bac61320088ea5750ef5389d66d |
| docs/implementation/implemented/08.02_PHASE_REPORT.html | rendered report | current artifact | SHA256 | 5e0823f47e752f91ac7b7de1803c1a0b4cb29c5bc77c2abfefe7b1bc901951d4 |
| docs/implementation/implemented/08.02_PHASE_CHECK.html | fresh full-gate report | full rerun passed | SHA256 | 4db7b5583c5e28df2a56d4b57f9038e935374fd3aa630a28768f84a32ac7aae1 |

all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "docs/implementation/evidence/task-286-preparation.md is preparation evidence, not independent review evidence; its prior report hash claims were rechecked against current files."

## 10. Coverage and Exceptions

- [x] Required focused and aggregate coverage/test commands ran.
- [x] Full gate recorded backend aggregate coverage 87.4% and Phase 08 coverage 93.2% (4645/4986) with repository-documented exceptions.
- [x] Untested branches relevant to changed symbols were inspected.
- [x] The task row has no coverage exception.
- [x] Source-report/build-input malformed branches have direct regression coverage.

coverage_required: true
coverage_exception_allowed: false
coverage_report_path: docs/implementation/implemented/08.02_PHASE_CHECK.html
observed_line_coverage: "backend 87.4%; Phase 08 93.2% (4645/4986)"
coverage_passed: true

Coverage finding: Aggregate coverage passes with repository-documented exceptions; direct source/input regressions cover the repaired boundary beyond line coverage.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] Final exact result.criterionId/source.report/source.status list regressions raise UATError.
- [x] Source producer result criterionId/status and malformed build-input regressions raise UATError.
- [x] Broad final nested malformed coverage passes.
- [x] Final/source/build CLI malformed-value regressions exit nonzero with one structured diagnostic and no traceback.
- [x] Report preserves 67 PASS, 13 FAIL, 11 BLOCKED, 14 open findings, FAIL gate, and PENDING unchecked decision.
- [x] Historical 08_PHASE_UAT.md is byte/hash unchanged.
- [x] Task-list, traceability, UAT, Task 280, diff, full gate, 40-iteration focused stress, and two consecutive aggregate quick gates pass.
- [x] Cleanup probes find no Task 281-286 process, phase08 UAT staging directory, or run-owned container; shared local PostgreSQL/Redis services are expected and were not targeted.

Findings: The final-report trust/recompute design, producer-source/input type guards, sensitive-value rejection, status preservation, open-ledger synchronization, historical hash, owner decision distinction, validators, cleanup, focused desktop/mobile stress, two consecutive quick gates, and the preserved full gate pass.

## 12. Decision

A task may be PASSED only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking or important finding remains.

~~~yaml
decision: "PASSED"
reason: "The repaired producer-source and build-input boundaries now reject malformed nested values before set/dict/hash/path/status operations, CLI diagnostics are bounded, current artifacts and all task-owned audits pass, and no blocking or important finding remains."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "None for Task 286 review; retain the PREPARED task-list status for project-owner acceptance."
~~~

## 13. Repair Context

The rejected stale-category check now awaits an explicit completion promise resolved only after the delayed route handler's awaited `route.fulfill` returns. Focused desktop/mobile stress and two consecutive quick gates pass. The repaired source-report, decision, build-input, and CLI surfaces remain covered by fresh malformed-input probes; the truthful 67/13/11 report, 14 findings, FAIL gate, PENDING decision, historical UAT, cleanup, and PREPARED task row remain unchanged.
