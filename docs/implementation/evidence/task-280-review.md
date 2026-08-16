# Review Evidence: Task 280 — DESIGN-014 MetricsCollector

```yaml
task_id: 280
component: "Phase 08.02 Requirement Acceptance Manifest and Finding Synchronization"
static_aspect: "DESIGN-014: MetricsCollector"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-28T06:20:00Z"
review_agent: "independent-task-280-final-reviewer"
evidence_file: "docs/implementation/evidence/task-280-review.md"
baseline_ref: "f9a646ffede5c8a63ad787a07eaba7efc0433677"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "Python; security; HTML/XSS output; JSON serialization; universal error handling."
repair_context_required: true
```

## 1. Task Source

The authoritative source is row 280 in `docs/implementation/02_TASK_LIST.md`:
Phase 08.02 Requirement Acceptance Manifest and Finding Synchronization,
DESIGN-014 MetricsCollector, status PREPARED, dependency 279 PASSED. The task
requires exact `req_tests.md` coverage, sanitized deterministic JSON/HTML
reports, producer continuation, evidence retention, finding synchronization and
historical closure, exact exit codes 0/1/2, and the listed repository gates.
Delivery infrastructure may pass while product requirements fail; this review
does not claim the Phase 08.02 UAT gate.

The updated preparation evidence was read completely. Task 279 preparation and
review evidence were read for dependency and overlap. The phase-orchestrator
review checklist was read completely. `code-review-skill` was invoked exactly
once; its Python, security, HTML/XSS, JSON, and error-handling guidance was
applied. No production code or task-list file was edited by this reviewer.

## 2. Pre-Review Gates

- [x] Input status is PREPARED.
- [x] Dependency Task 279 is PASSED.
- [x] Baseline `f9a646ffede5c8a63ad787a07eaba7efc0433677` and the current
      task-owned working-tree surface were reconstructed independently.
- [x] Manifest, reporter, parser, sanitizer, evidence-link, exit-code,
      finding, caller, test, design, documentation, and prior-task boundaries
      were inspected.
- [x] `code-review-skill` was invoked exactly once.
- [x] Requested repairs and all prior fixes were independently re-exercised.
- [x] No production-code or task-list file was changed by this reviewer.

```yaml
pre_review_gates_passed: true
```

## 3. Review Baseline and Change Surface

`HEAD` is `f9a646ffede5c8a63ad787a07eaba7efc0433677` on
`multistep-phase-08`. Task 280’s working-tree surface is the acceptance script,
its tests, manifest and trace sidecar, finding-history registry and sidecar,
Task 280 portions of `scripts/check.py` and `04_OPEN.md`, and the preparation
evidence. The worktree also contains prior Phase 08 implementation and Tasks
276–279 changes; those were preserved and were not attributed to Task 280
except at the unique acceptance symbols, markers, and trace entries.

Independent reconstruction used `git status --short --untracked-files=all`,
`git log`, `git diff --numstat`, targeted `git diff`/`rg`, AST symbol listing,
source inspection, manifest parsing, and SHA-256 fingerprints. The current
source has 12 scenarios and 91 source entries. The repaired `execute_producers`
boundary uses `subprocess.DEVNULL` for both child streams. The repaired
`write_report` boundary prefixes every published evidence path with
`evidence/`, copies it below that directory before writing the top-level
reports, and renders the namespaced links.

## 4. Acceptance Criteria Checklist

Every one of the 91 flattened manifest entries is listed below. `PASS` means
the source/manifest contract and its focused negative or positive regression
were independently checked; it does not assert that a future real-stack
product run will pass its requirement.

| # | Stable ID | Requirement | Kind | Result | Evidence |
|---:|---|---|---|---|---|
| 1 | P08-SWR019-STEP-01 | SW-REQ-019 | step | PASS | exact source mapping, validator/test suite |
| 2 | P08-SWR019-STEP-02 | SW-REQ-019 | step | PASS | exact source mapping, validator/test suite |
| 3 | P08-SWR019-STEP-03 | SW-REQ-019 | step | PASS | exact source mapping, validator/test suite |
| 4 | P08-SWR019-STEP-04 | SW-REQ-019 | step | PASS | exact source mapping, validator/test suite |
| 5 | P08-SWR032-STEP-01 | SW-REQ-032 | step | PASS | exact source mapping, validator/test suite |
| 6 | P08-SWR032-STEP-02 | SW-REQ-032 | step | PASS | exact source mapping, validator/test suite |
| 7 | P08-SWR032-STEP-03 | SW-REQ-032 | step | PASS | exact source mapping, validator/test suite |
| 8 | P08-SWR033-STEP-01 | SW-REQ-033 | step | PASS | exact source mapping, validator/test suite |
| 9 | P08-SWR033-STEP-02 | SW-REQ-033 | step | PASS | exact source mapping, validator/test suite |
| 10 | P08-SWR033-STEP-03 | SW-REQ-033 | step | PASS | exact source mapping, validator/test suite |
| 11 | P08-SWR033-STEP-04 | SW-REQ-033 | step | PASS | exact source mapping, validator/test suite |
| 12 | P08-SWR033-STEP-05 | SW-REQ-033 | step | PASS | exact source mapping, validator/test suite |
| 13 | P08-SWR043-STEP-01 | SW-REQ-043 | step | PASS | exact source mapping, validator/test suite |
| 14 | P08-SWR043-STEP-02 | SW-REQ-043 | step | PASS | exact source mapping, validator/test suite |
| 15 | P08-SWR043-STEP-03 | SW-REQ-043 | step | PASS | exact source mapping, validator/test suite |
| 16 | P08-SWR043-STEP-04 | SW-REQ-043 | step | PASS | exact source mapping, validator/test suite |
| 17 | P08-SWR043-ACCEPT-01 | SW-REQ-043 | criterion | PASS | exact source mapping, validator/test suite |
| 18 | P08-SWR054-STEP-01 | SW-REQ-054 | step | PASS | exact source mapping, validator/test suite |
| 19 | P08-SWR054-STEP-02 | SW-REQ-054 | step | PASS | exact source mapping, validator/test suite |
| 20 | P08-SWR054-STEP-03 | SW-REQ-054 | step | PASS | exact source mapping, validator/test suite |
| 21 | P08-SWR054-STEP-04 | SW-REQ-054 | step | PASS | exact source mapping, validator/test suite |
| 22 | P08-SWR054-ACCEPT-01 | SW-REQ-054 | criterion | PASS | exact source mapping, validator/test suite |
| 23 | P08-SWR054-ACCEPT-02 | SW-REQ-054 | criterion | PASS | exact source mapping, validator/test suite |
| 24 | P08-SWR054-ACCEPT-03 | SW-REQ-054 | criterion | PASS | exact source mapping, validator/test suite |
| 25 | P08-SWR054-ACCEPT-04 | SW-REQ-054 | criterion | PASS | exact source mapping, validator/test suite |
| 26 | P08-SWR054-ACCEPT-05 | SW-REQ-054 | criterion | PASS | exact source mapping, validator/test suite |
| 27 | P08-SWR055-STEP-01 | SW-REQ-055 | step | PASS | exact source mapping, validator/test suite |
| 28 | P08-SWR055-STEP-02 | SW-REQ-055 | step | PASS | exact source mapping, validator/test suite |
| 29 | P08-SWR055-STEP-03 | SW-REQ-055 | step | PASS | exact source mapping, validator/test suite |
| 30 | P08-SWR055-STEP-04 | SW-REQ-055 | step | PASS | exact source mapping, validator/test suite |
| 31 | P08-SWR055-STEP-05 | SW-REQ-055 | step | PASS | exact source mapping, validator/test suite |
| 32 | P08-SWR055-STEP-06 | SW-REQ-055 | step | PASS | exact source mapping, validator/test suite |
| 33 | P08-SWR055-STEP-07 | SW-REQ-055 | step | PASS | exact source mapping, validator/test suite |
| 34 | P08-SWR055-ACCEPT-01 | SW-REQ-055 | criterion | PASS | exact source mapping, validator/test suite |
| 35 | P08-SWR055-ACCEPT-02 | SW-REQ-055 | criterion | PASS | exact source mapping, validator/test suite |
| 36 | P08-SWR055-ACCEPT-03 | SW-REQ-055 | criterion | PASS | exact source mapping, validator/test suite |
| 37 | P08-SWR055-ACCEPT-04 | SW-REQ-055 | criterion | PASS | exact source mapping, validator/test suite |
| 38 | P08-SWR055-ACCEPT-05 | SW-REQ-055 | criterion | PASS | exact source mapping, validator/test suite |
| 39 | P08-SWR055-ACCEPT-06 | SW-REQ-055 | criterion | PASS | exact source mapping, validator/test suite |
| 40 | P08-SWR055-ACCEPT-07 | SW-REQ-055 | criterion | PASS | exact source mapping, validator/test suite |
| 41 | P08-SWR056-STEP-01 | SW-REQ-056 | step | PASS | exact source mapping, validator/test suite |
| 42 | P08-SWR056-STEP-02 | SW-REQ-056 | step | PASS | exact source mapping, validator/test suite |
| 43 | P08-SWR056-STEP-03 | SW-REQ-056 | step | PASS | exact source mapping, validator/test suite |
| 44 | P08-SWR056-STEP-04 | SW-REQ-056 | step | PASS | exact source mapping, validator/test suite |
| 45 | P08-SWR056-STEP-05 | SW-REQ-056 | step | PASS | exact source mapping, validator/test suite |
| 46 | P08-SWR056-STEP-06 | SW-REQ-056 | step | PASS | exact source mapping, validator/test suite |
| 47 | P08-SWR056-ACCEPT-01 | SW-REQ-056 | criterion | PASS | exact source mapping, validator/test suite |
| 48 | P08-SWR056-ACCEPT-02 | SW-REQ-056 | criterion | PASS | exact source mapping, validator/test suite |
| 49 | P08-SWR056-ACCEPT-03 | SW-REQ-056 | criterion | PASS | exact source mapping, validator/test suite |
| 50 | P08-SWR056-ACCEPT-04 | SW-REQ-056 | criterion | PASS | exact source mapping, validator/test suite |
| 51 | P08-SWR056-ACCEPT-05 | SW-REQ-056 | criterion | PASS | exact source mapping, validator/test suite |
| 52 | P08-SWR056-ACCEPT-06 | SW-REQ-056 | criterion | PASS | exact source mapping, validator/test suite |
| 53 | P08-SWR056-ACCEPT-07 | SW-REQ-056 | criterion | PASS | exact source mapping, validator/test suite |
| 54 | P08-SWR057-STEP-01 | SW-REQ-057 | step | PASS | exact source mapping, validator/test suite |
| 55 | P08-SWR057-STEP-02 | SW-REQ-057 | step | PASS | exact source mapping, validator/test suite |
| 56 | P08-SWR057-STEP-03 | SW-REQ-057 | step | PASS | exact source mapping, validator/test suite |
| 57 | P08-SWR057-STEP-04 | SW-REQ-057 | step | PASS | exact source mapping, validator/test suite |
| 58 | P08-SWR057-STEP-05 | SW-REQ-057 | step | PASS | exact source mapping, validator/test suite |
| 59 | P08-SWR057-STEP-06 | SW-REQ-057 | step | PASS | exact source mapping, validator/test suite |
| 60 | P08-SWR057-STEP-07 | SW-REQ-057 | step | PASS | exact source mapping, validator/test suite |
| 61 | P08-SWR057-STEP-08 | SW-REQ-057 | step | PASS | exact source mapping, validator/test suite |
| 62 | P08-SWR057-ACCEPT-01 | SW-REQ-057 | criterion | PASS | exact source mapping, validator/test suite |
| 63 | P08-SWR057-ACCEPT-02 | SW-REQ-057 | criterion | PASS | exact source mapping, validator/test suite |
| 64 | P08-SWR057-ACCEPT-03 | SW-REQ-057 | criterion | PASS | exact source mapping, validator/test suite |
| 65 | P08-SWR057-ACCEPT-04 | SW-REQ-057 | criterion | PASS | exact source mapping, validator/test suite |
| 66 | P08-SWR057-ACCEPT-05 | SW-REQ-057 | criterion | PASS | exact source mapping, validator/test suite |
| 67 | P08-SWR057-ACCEPT-06 | SW-REQ-057 | criterion | PASS | exact source mapping, validator/test suite |
| 68 | P08-SWR072-STEP-01 | SW-REQ-072 | step | PASS | exact source mapping, validator/test suite |
| 69 | P08-SWR072-STEP-02 | SW-REQ-072 | step | PASS | exact source mapping, validator/test suite |
| 70 | P08-SWR072-STEP-03 | SW-REQ-072 | step | PASS | exact source mapping, validator/test suite |
| 71 | P08-SWR072-STEP-04 | SW-REQ-072 | step | PASS | exact source mapping, validator/test suite |
| 72 | P08-SWR072-STEP-05 | SW-REQ-072 | step | PASS | exact source mapping, validator/test suite |
| 73 | P08-SWR072-STEP-06 | SW-REQ-072 | step | PASS | exact source mapping, validator/test suite |
| 74 | P08-SWR073-STEP-01 | SW-REQ-073 | step | PASS | exact source mapping, validator/test suite |
| 75 | P08-SWR073-STEP-02 | SW-REQ-073 | step | PASS | exact source mapping, validator/test suite |
| 76 | P08-SWR073-STEP-03 | SW-REQ-073 | step | PASS | exact source mapping, validator/test suite |
| 77 | P08-SWR073-STEP-04 | SW-REQ-073 | step | PASS | exact source mapping, validator/test suite |
| 78 | P08-SWR073-STEP-05 | SW-REQ-073 | step | PASS | exact source mapping, validator/test suite |
| 79 | P08-SWR073-STEP-06 | SW-REQ-073 | step | PASS | exact source mapping, validator/test suite |
| 80 | P08-SWR084-STEP-01 | SW-REQ-084 | step | PASS | exact source mapping, validator/test suite |
| 81 | P08-SWR084-STEP-02 | SW-REQ-084 | step | PASS | exact source mapping, validator/test suite |
| 82 | P08-SWR084-STEP-03 | SW-REQ-084 | step | PASS | exact source mapping, validator/test suite |
| 83 | P08-SWR084-ACCEPT-01 | SW-REQ-084 | criterion | PASS | exact source mapping, validator/test suite |
| 84 | P08-SWR084-ACCEPT-02 | SW-REQ-084 | criterion | PASS | exact source mapping, validator/test suite |
| 85 | P08-SWR084-ACCEPT-03 | SW-REQ-084 | criterion | PASS | exact source mapping, validator/test suite |
| 86 | P08-SWR084-ACCEPT-04 | SW-REQ-084 | criterion | PASS | exact source mapping, validator/test suite |
| 87 | P08-SWR090-STEP-01 | SW-REQ-090 | step | PASS | exact source mapping, validator/test suite |
| 88 | P08-SWR090-STEP-02 | SW-REQ-090 | step | PASS | exact source mapping, validator/test suite |
| 89 | P08-SWR090-STEP-03 | SW-REQ-090 | step | PASS | exact source mapping, validator/test suite |
| 90 | P08-SWR090-STEP-04 | SW-REQ-090 | step | PASS | exact source mapping, validator/test suite |
| 91 | P08-SWR090-ACCEPT-01 | SW-REQ-090 | criterion | PASS | exact source mapping, validator/test suite |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Change | Caller/consumer | Test or evidence |
|---:|---|---|---|---|---|---|
| 1 | EVIDENCE_NAMESPACE | configuration | scripts/phase08_acceptance.py:28 | added | report publication | protected-name test |
| 2 | ValidationError | exception | scripts/phase08_acceptance.py:82 | added | validators and CLI | negative fixtures |
| 3 | SourceEntry | dataclass | scripts/phase08_acceptance.py:87 | added | source parser/manifest | manifest inventory |
| 4 | strict_json_loads | function | scripts/phase08_acceptance.py:96 | added | JSON/ledger loaders | duplicate-key test |
| 5 | reject_duplicates | nested function | scripts/phase08_acceptance.py:99 | added | JSON object hook | duplicate-key test |
| 6 | read_text | function | scripts/phase08_acceptance.py:116 | added | source/docs loader | sanitized read errors |
| 7 | load_json | function | scripts/phase08_acceptance.py:125 | changed | manifest/plan/results/history | validator and CLI |
| 8 | req_test_entries | function | scripts/phase08_acceptance.py:131 | changed | manifest validator | 91-entry assertion |
| 9 | validate_manifest | function | scripts/phase08_acceptance.py:159 | changed | all CLI commands | orphan/duplicate/kind tests |
| 10 | extract_finding_ledger | function | scripts/phase08_acceptance.py:249 | changed | acceptance context | duplicate-ledger test |
| 11 | validate_finding_history | function | scripts/phase08_acceptance.py:265 | added | acceptance context | mandatory-baseline test |
| 12 | validate_findings | function | scripts/phase08_acceptance.py:282 | changed | context/synchronizer | completeness/history tests |
| 13 | safe_relative_path | function | scripts/phase08_acceptance.py:368 | changed | evidence/results/findings | encoded-traversal matrix |
| 14 | assert_safe_value | function | scripts/phase08_acceptance.py:387 | re-exercised | finding/backend/report | sensitive-value test |
| 15 | normalize_evidence | function | scripts/phase08_acceptance.py:404 | re-exercised | result normalization | safe-link tests |
| 16 | normalize_results | function | scripts/phase08_acceptance.py:420 | re-exercised | report/run | result completeness tests |
| 17 | synchronize_results | function | scripts/phase08_acceptance.py:493 | re-exercised | report/run | finding synchronization |
| 18 | overall_status | function | scripts/phase08_acceptance.py:526 | re-exercised | finalization | exit fixtures |
| 19 | evidence_paths | function | scripts/phase08_acceptance.py:537 | added | report finalization | retention tests |
| 20 | namespace_evidence | function | scripts/phase08_acceptance.py:549 | added | report links/copy | collision test |
| 21 | copy_evidence | function | scripts/phase08_acceptance.py:564 | added | report staging | missing/symlink tests |
| 22 | write_report | function | scripts/phase08_acceptance.py:589 | changed | report/run commands | mixed/atomic/retention tests |
| 23 | load_result_files | function | scripts/phase08_acceptance.py:681 | re-exercised | report/run | result completeness tests |
| 24 | execute_producers | function | scripts/phase08_acceptance.py:694 | changed | command_run | exit/continuation/stream tests |
| 25 | acceptance_context | function | scripts/phase08_acceptance.py:735 | added | all commands | history-baseline test |
| 26 | command_validate | function | scripts/phase08_acceptance.py:746 | changed | parser | normal validation |
| 27 | command_report | function | scripts/phase08_acceptance.py:754 | changed | parser | report fixtures |
| 28 | command_run | function | scripts/phase08_acceptance.py:771 | changed | parser | producer e2e |
| 29 | build_parser | function | scripts/phase08_acceptance.py:796 | changed | main | CLI shape |
| 30 | main | function | scripts/phase08_acceptance.py:820 | changed | process entry | sanitized CLI errors |
| 31 | Phase08AcceptanceTests | test class | scripts/test_phase08_acceptance.py:28 | changed | unittest loader | 24 tests |
| 32 | setUpClass | fixture | scripts/test_phase08_acceptance.py:32 | changed | test class | manifest setup |
| 33 | pass_results | helper | scripts/test_phase08_acceptance.py:39 | changed | report fixtures | complete results |
| 34 | ledger | helper | scripts/test_phase08_acceptance.py:54 | changed | finding fixtures | history fixtures |
| 35 | finding | helper | scripts/test_phase08_acceptance.py:60 | changed | finding fixtures | closure/completeness |
| 36 | validate_ledger | helper | scripts/test_phase08_acceptance.py:86 | added | finding tests | complete history |
| 37 | test_manifest_maps_every_source_entry_exactly_once | test | scripts/test_phase08_acceptance.py:97 | changed | unittest | 91-entry coverage |
| 38 | test_manifest_rejects_duplicate_and_orphan_mappings | test | scripts/test_phase08_acceptance.py:108 | changed | unittest | duplicate/orphan/kind |
| 39 | test_manifest_has_required_json_sidecar_traceability | test | scripts/test_phase08_acceptance.py:122 | changed | unittest | sidecars |
| 40 | test_mixed_report_is_ordered_correlated_grouped_and_exit_one | test | scripts/test_phase08_acceptance.py:136 | changed | unittest | mixed report |
| 41 | test_fixture_exit_codes_are_zero_one_two | test | scripts/test_phase08_acceptance.py:205 | changed | unittest | exit contract |
| 42 | test_report_rejects_missing_duplicate_and_orphan_results | test | scripts/test_phase08_acceptance.py:230 | re-exercised | unittest | result completeness |
| 43 | test_evidence_links_must_be_safe_and_relative | test | scripts/test_phase08_acceptance.py:241 | changed | unittest | traversal matrix |
| 44 | test_report_rejects_missing_and_symlink_evidence | test | scripts/test_phase08_acceptance.py:260 | added | unittest | evidence boundary |
| 45 | test_protected_report_names_are_preserved_in_evidence_namespace | test | scripts/test_phase08_acceptance.py:283 | added | unittest | collision boundary |
| 46 | test_sensitive_fields_and_values_are_rejected | test | scripts/test_phase08_acceptance.py:318 | changed | unittest | sanitization |
| 47 | test_finding_requires_complete_unresolved_metadata | test | scripts/test_phase08_acceptance.py:345 | changed | unittest | finding metadata |
| 48 | test_nonpass_requires_matching_unresolved_finding | test | scripts/test_phase08_acceptance.py:362 | changed | unittest | synchronization |
| 49 | test_duplicate_root_cause_findings_are_rejected | test | scripts/test_phase08_acceptance.py:377 | re-exercised | unittest | root uniqueness |
| 50 | test_embedded_ledger_rejects_duplicate_json_keys | test | scripts/test_phase08_acceptance.py:385 | added | unittest | duplicate ledger |
| 51 | test_passing_retest_closes_and_retains_historical_finding | test | scripts/test_phase08_acceptance.py:395 | changed | unittest | historical closure |
| 52 | test_closed_finding_requires_date_and_passing_evidence | test | scripts/test_phase08_acceptance.py:411 | re-exercised | unittest | closure fields |
| 53 | test_failure_during_finalization_publishes_no_partial_run | test | scripts/test_phase08_acceptance.py:420 | changed | unittest | atomic cleanup |
| 54 | fail_second | nested test helper | scripts/test_phase08_acceptance.py:426 | re-exercised | finalization test | injected failure |
| 55 | test_all_producers_run_after_individual_failures | test | scripts/test_phase08_acceptance.py:443 | changed | unittest | continuation |
| 56 | test_run_preserves_evidence_and_nonzero_producer_cannot_return_zero | test | scripts/test_phase08_acceptance.py:470 | added | unittest | exit/evidence e2e |
| 57 | test_producer_stdout_stderr_and_traceback_are_never_emitted_by_cli | test | scripts/test_phase08_acceptance.py:536 | added | unittest | stream sanitization |
| 58 | test_missing_and_spawn_failed_producers_never_return_zero | test | scripts/test_phase08_acceptance.py:580 | added | unittest | missing/spawn |
| 59 | test_history_registry_is_mandatory_for_normal_cli | test | scripts/test_phase08_acceptance.py:625 | added | unittest | baseline CLI |
| 60 | test_cli_external_errors_are_sanitized_without_traceback | test | scripts/test_phase08_acceptance.py:680 | added | unittest | bounded errors |
| 61 | test_current_open_document_ledger_is_valid | test | scripts/test_phase08_acceptance.py:716 | changed | unittest | current docs |
| 62 | validate_phase08_acceptance_contracts | gate function | scripts/check.py:637 | changed | static lane | focused suite/validator |
| 63 | CheckStep | orchestration class | scripts/check.py:27 | re-exercised | full/quick lanes | aggregate pass |
| 64 | TRACEABLE_FILES acceptance entries | configuration | scripts/check.py:705 | changed | traceability | validator pass |
| 65 | acceptance-manifest.json | JSON contract | docs/testing/phase08/acceptance-manifest.json:1-206 | added | manifest loader | 12/91 inventory |
| 66 | manifest JSON trace sidecar | documentation contract | docs/testing/phase08/acceptance-manifest.json-trace.md:1-33 | added | traceability/reviewer | sidecar test |
| 67 | finding-history.json | JSON historical baseline | docs/testing/phase08/finding-history.json:1-4 | added | acceptance context | mandatory history |
| 68 | finding-history JSON trace sidecar | documentation contract | docs/testing/phase08/finding-history.json-trace.md:1-10 | added | traceability/reviewer | sidecar inspection |
| 69 | Phase 08 finding ledger markers | documentation data contract | docs/implementation/04_OPEN.md:563-570 | added | ledger parser | current ledger test |

```yaml
inventory_source_count: 69
audited_symbol_count: 69
inventory_complete: true
generated_groupings:
  - "Manifest/history JSON are audited as data contracts; sidecars and the 04_OPEN ledger markers are audited as documentation/control surfaces."
```

## 6. Function-Level Audit

| # | Symbol/unit | Contract, boundary, and adversarial audit | Result |
|---:|---|---|---|
| 1 | EVIDENCE_NAMESPACE | Fixed report-local namespace is used for every published evidence path. | PASS |
| 2 | ValidationError | Bounded validation exception is caught by `main`; direct exception text is not exposed as a traceback. | PASS |
| 3 | SourceEntry | Frozen source identity preserves requirement, kind, line, and exact text. | PASS |
| 4 | strict_json_loads | Rejects malformed JSON, non-object roots, and duplicate keys at every decoded object boundary. | PASS |
| 5 | reject_duplicates | Object-pairs hook prevents last-write-wins duplicate-key parsing. | PASS |
| 6 | read_text | UTF-8 and filesystem failures become bounded validation messages. | PASS |
| 7 | load_json | All JSON documents use strict decoding before schema validation. | PASS |
| 8 | req_test_entries | Preserves authoritative source kind, line, and exact text for one-to-one comparison. | PASS |
| 9 | validate_manifest | Enforces schema, stable IDs, exact kind-aware source identity, metadata, and exact requirement set. | PASS |
| 10 | extract_finding_ledger | Checks unique markers, fences, schema, exact fields, and duplicate keys. | PASS |
| 11 | validate_finding_history | Requires the mandatory sorted unique safe-ID registry; current empty baseline is valid. | PASS |
| 12 | validate_findings | Checks complete metadata, safe links, root uniqueness, closure, and current/history equality. | PASS |
| 13 | safe_relative_path | Rejects literal and encoded traversal, absolute/URI/query/fragment/control/hidden/backslash paths. | PASS |
| 14 | assert_safe_value | Recurses through structured values and rejects prohibited fields and recognizable sensitive values. | PASS |
| 15 | normalize_evidence | Allows only manifest evidence types and deterministic safe paths. | PASS |
| 16 | normalize_results | Rejects orphan/duplicate/missing criteria, invalid statuses/IDs/summaries, and unsafe evidence. | PASS |
| 17 | synchronize_results | Requires unresolved findings for non-pass roots and closed findings for resolved roots. | PASS |
| 18 | overall_status | Implements PASS=0, FAIL=1 precedence, and BLOCKED=2. | PASS |
| 19 | evidence_paths | Deduplicates and sorts source evidence paths before publication. | PASS |
| 20 | namespace_evidence | Rewrites every report link into `evidence/`, preventing top-level report collisions. | PASS |
| 21 | copy_evidence | Rejects missing/symlink components and copies validated regular files before producer cleanup. | PASS |
| 22 | write_report | Atomically stages evidence under `evidence/`, writes top-level reports afterward, and cleans failed staging. | PASS |
| 23 | load_result_files | Strictly loads all result arrays; malformed collected output cannot create a partial report. | PASS |
| 24 | execute_producers | Runs every producer, redirects stdout/stderr to `DEVNULL`, and emits only fixed safe failure codes. | PASS |
| 25 | acceptance_context | Every command loads manifest, source, mandatory history, and current ledger before reporting. | PASS |
| 26 | command_validate | Normal validation has bounded success/error output. | PASS |
| 27 | command_report | Normalizes, synchronizes, finalizes, and propagates report status. | PASS |
| 28 | command_run | Continues after producer failure, retains evidence, and forces nonzero producer exit to report/process code 1. | PASS |
| 29 | build_parser | Exposes explicit manifest/source/history/result/plan/output controls. | PASS |
| 30 | main | Catches validation/filesystem/Unicode/subprocess failures without traceback or private diagnostics. | PASS |
| 31 | Phase08AcceptanceTests | Isolated temporary fixtures cover the full manifest and report boundaries. | PASS |
| 32 | setUpClass | Loads the current manifest through production validators. | PASS |
| 33 | pass_results | Generates one complete PASS result for every manifest criterion. | PASS |
| 34 | ledger | Builds schema-shaped finding documents for current/history cases. | PASS |
| 35 | finding | Supplies complete sanitized open and closed finding fixtures. | PASS |
| 36 | validate_ledger | Couples fixtures to the complete historical ID set without an optional-baseline shortcut. | PASS |
| 37 | manifest coverage test | Confirms 12 scenarios, 91 entries, exact requirements, and metadata. | PASS |
| 38 | duplicate/orphan/kind test | Mutates criterion ID, source line, and source kind; all tampering rejects. | PASS |
| 39 | sidecar test | Checks JSON sidecars for DESIGN-014 and traceability requirements. | PASS |
| 40 | mixed report test | Exercises mixed statuses, deterministic ordering, request IDs, evidence, HTML links, and grouping. | PASS |
| 41 | exit fixture test | Proves direct report codes 0/1/2. | PASS |
| 42 | result completeness test | Rejects missing, duplicate, and orphan result IDs. | PASS |
| 43 | safe-link test | Exercises literal/encoded traversal, URI, hidden, query, fragment, control, colon, and backslash paths. | PASS |
| 44 | missing/symlink evidence test | Fail-closed missing and symlink publication passes. | PASS |
| 45 | protected report-name test | `report.json` and `report.html` evidence survive under `evidence/` and links are rewritten. | PASS |
| 46 | sensitive-value test | Covers prohibited fields/values and raw provider summary rejection. | PASS |
| 47 | finding completeness test | Deletes each required finding field and expects rejection. | PASS |
| 48 | non-pass synchronization test | Missing and CLOSED findings cannot satisfy a non-pass root. | PASS |
| 49 | duplicate-root test | Two finding IDs for one root cause reject. | PASS |
| 50 | duplicate-ledger test | Embedded duplicate finding keys reject through the shared decoder. | PASS |
| 51 | historical closure test | Passing retest closes a finding while deleted current records reject against history. | PASS |
| 52 | closed-field test | CLOSED findings require date and passing evidence. | PASS |
| 53 | finalization test | Injected write failure leaves no final or temporary partial run. | PASS |
| 54 | fail_second | Deterministic write-failure injection covers cleanup. | PASS |
| 55 | producer continuation test | Later producers run after an individual failure and safe codes collect. | PASS |
| 56 | producer/evidence e2e test | Exit 7 forces code 1, later producer runs, and ordinary evidence survives cleanup. | PASS |
| 57 | producer stream test | Child stdout, stderr, traceback, and exception secret never reach CLI output. | PASS |
| 58 | missing/spawn producer test | Missing result and spawn failure never return zero with a passing producer. | PASS |
| 59 | mandatory history CLI test | Missing, deleted, and unregistered IDs reject; a valid registry passes. | PASS |
| 60 | CLI sanitization test | Malformed UTF-8 and external write errors are bounded without traceback. | PASS |
| 61 | current ledger test | Current `04_OPEN.md` ledger and mandatory history validate against current scenarios. | PASS |
| 62 | validate_phase08_acceptance_contracts | Static lane runs the focused suite and normal validator. | PASS |
| 63 | CheckStep | Full and quick orchestration include the acceptance contract lane. | PASS |
| 64 | TRACEABLE_FILES entries | Both acceptance scripts are registered traceability inputs. | PASS |
| 65 | acceptance manifest | Stable 12-scenario/91-entry source contract is exact. | PASS |
| 66 | manifest sidecar | Documents DESIGN-014 and source/report boundaries. | PASS |
| 67 | finding history | Mandatory sorted unique historical baseline exists and matches current empty ledger. | PASS |
| 68 | history sidecar | Documents retention and DESIGN-014 traceability. | PASS |
| 69 | 04_OPEN ledger markers | Machine-readable ledger markers sit beside human actions and are parsed. | PASS |

## 7. Findings

The two previously important findings are repaired and closed by this review:

- Producer stdout/stderr is redirected to `subprocess.DEVNULL`; the regression
  producer prints secrets and an uncaught traceback, but the parent emits only
  `producer diagnostics: BACKEND:EXIT-1` and the report line.
- Evidence is copied beneath `evidence/` before top-level `report.json` and
  `report.html` are written; both same-named producer files retain their bytes
  and report/HTML links point to the namespace.

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| OPTIONAL | scripts/phase08_acceptance.py:694 | execute_producers | There is no internal producer timeout or process-group cancellation for a hung producer. | Source audit; the preparation evidence explicitly records this as a residual external execution-contract risk. | Keep the optional residual documented, or add a bounded timeout/process-group cleanup in a later hardening task. It does not fail Task 280’s stated producer-failure/retention criteria. |

```yaml
blocking_findings: 0
important_findings: 0
optional_findings: 1
```

## 8. Commands Run

All commands ran in `/home/wiktor/Work/mealswapp` on 2026-07-28. Expected
negative probes are PASS when rejection is the expected result.

| Command or scenario | Exit code | Result | Observed evidence |
|---|---:|---|---|
| Read phase-orchestrator `review_checklist.md` completely | 0 | PASS | Required review structure applied. |
| Read `code-review-skill` exactly once with Python/security/HTML/JSON/error guidance | 0 | PASS | Guidance applied to source audit. |
| `python3 -m unittest -v scripts/test_phase08_acceptance.py` | 0 | PASS | 24 tests passed in 0.338s; stream and collision regressions included. |
| `python3 scripts/phase08_acceptance.py validate` | 0 | PASS | 12 scenarios and 91 criteria valid; mandatory history loaded. |
| `python3 -m py_compile scripts/phase08_acceptance.py scripts/test_phase08_acceptance.py scripts/check.py` | 0 | PASS | All reviewed Python scripts compile. |
| `python3 scripts/validate-task-list.py` | 0 | PASS | 286 sequential tasks valid. |
| `python3 scripts/validate-traceability.py` | 0 | PASS | Traceability valid. |
| `git diff --check` | 0 | PASS | No whitespace errors. |
| Manifest kind-tamper probe | 0 | PASS | Source kind mutation rejected as an unmapped entry. |
| Embedded duplicate-ledger-key probe | 0 | PASS | Duplicate JSON key rejected before validation. |
| Encoded traversal/path matrix | 0 | PASS | Percent-encoded traversal and unsafe path classes rejected. |
| Producer exit/evidence/later-producer e2e | 0 | PASS | Exit 7 produces report/process code 1; later producer runs; evidence survives cleanup. |
| Producer stdout/stderr/traceback probe | 0 | PASS | Secret stdout, secret stderr, traceback, and exception text absent from CLI output. |
| Protected `report.json`/`report.html` evidence probe | 0 | PASS | Original bytes retained at `evidence/report.json` and `evidence/report.html`; links namespaced. |
| Mandatory history baseline CLI probe | 0 | PASS | Missing/deleted/unregistered cases reject; valid empty registry passes. |
| Sanitized CLI external-error probe | 0 | PASS | No traceback, private path, bytes, or injected exception text. |
| `python3 scripts/check.py --quick` | 0 | PASS | Static, Task 280, frontend, changed browser, backend, vet, vulnerability, OpenAPI, and validators passed; 77.8s. |
| `python3 scripts/check.py` | 0 | PASS | Full lanes passed; 309 Playwright passes/5 expected skips, race/security passed, Phase 08 coverage 4639/4980 (93.2%). |
| AST symbol listing and source inspection | 0 | PASS | All 69 inventory units audited; repaired boundaries inspected at current lines. |
| `sha256sum` reviewed-file fingerprint command | 0 | PASS | Current hashes recorded in section 9. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/evidence/task-280-review.md` | 0 | PASS | Final evidence structurally valid. |

## 9. Files Inspected and Staleness Fingerprints

The prior rejected Task 280 review hash was
`a7db3a91a907fc10aba4ea648934a10d56d18b1820132198cad4dffe40fde979`; its
producer-stream and evidence-collision findings were re-exercised after repair.
Task 279 dependency evidence was also checked for overlap.

The preparation file is current at the hash below, and its embedded fingerprint
for `docs/implementation/04_OPEN.md` matches the current file. The prior review
fingerprint was stale because it predates the repair; current validators and the
current ledger were rerun against the repaired source.

| File | Purpose | SHA-256 |
|---|---|---|
| docs/implementation/02_TASK_LIST.md | authoritative task row | 0097d216ae5b99f2373b365df1fedc900d96b7c0b237f670dc8e9030fc08dab4 |
| req_tests.md | authoritative Phase 08 source entries | 241779579f568eecfe40a8d41b97ec2b6a87e6201bd25b8ea76c4bac5c2a8902 |
| docs/implementation/04_OPEN.md | current finding ledger and coverage control | c2f21ddf39942efe3c4e2d3f3b45580ba0e27a72891ee4e2bd14aca6ab76116b |
| docs/design/DESIGN-014.md | MetricsCollector design contract | f9f6521d89e6d31306422017e07af5630ba4d8da56907174f3653ea0d72e9fe4 |
| docs/architecture/ARCH-018.md | overlap/reference architecture | 52eb914b44fe2ec105483b94d0a74225a1a6a9072d122c3db25a4b49828c57b9 |
| scripts/phase08_acceptance.py | parser, validator, reporter, producer and CLI | cd74a1ab8e1c16d6c7e85425a84e9083a594ff69674ce808654482476d993287 |
| scripts/test_phase08_acceptance.py | Task 280 regression suite | ce8962bece793702c2f71af09dc470a7a56187b18662338c8c36b7d3e42198b5 |
| scripts/check.py | aggregate acceptance gate and traceability registration | 6f9ec0c6af49fdf49c79a93d7076aee5cb03f67678943d7da13b634b26f67595 |
| docs/testing/phase08/acceptance-manifest.json | scenario/criterion contract | 2f597b9545bb176e45a03df02d1b05f01fc424b479ffb4b64c43ee3b6991fa8c |
| docs/testing/phase08/acceptance-manifest.json-trace.md | manifest sidecar | 2b8f681a27e559b909c7f5d4614d685627050d59c9bdc2d1962ed82b7c0559a5 |
| docs/testing/phase08/finding-history.json | mandatory historical ID baseline | 0963e472b2cf2001cd4fe369c75bb4a6ec85f836e821a6f440c523a62d64b105 |
| docs/testing/phase08/finding-history.json-trace.md | history sidecar | f9136c93b2cded40a92ce0d1b2bc92152f43654f596b83fa3a3aa084cce4d2f7 |
| docs/implementation/evidence/task-280-preparation.md | repaired preparation claims | ed3755d1bb9ff9eed7c204aa6dbf23b22aa3c9882c6e8b5ba1e3f85e56853d22 |
| docs/implementation/evidence/task-279-preparation.md | dependency preparation evidence | 5c57523da3092b8cc3bb9fd1d8cf9076c1ec23364dc1d57015723165fc73e03d |
| docs/implementation/evidence/task-279-review.md | dependency final review | 156c91717ba74a0e680a94c3030cd90d4a748905bfadae581b8bc25c98c79ab1 |

```yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "Previous rejected Task 280 review hash a7db3a91a907fc10aba4ea648934a10d56d18b1820132198cad4dffe40fde979 was superseded and its findings were re-exercised."
  - "The prior rejected review’s implementation hashes were stale after repair; the updated preparation fingerprints and current source hashes were independently checked."
review_evidence_self_hash: "intentionally omitted because the evidence file contains its own refreshed content"
```

## 10. Coverage and Exceptions

- [x] Focused Task 280 suite: 24 tests passed.
- [x] Quick aggregate gate passed.
- [x] Full aggregate gate passed, including backend race/security lanes,
      frontend build/unit/coverage, browser, local-stack, UAT, and static lanes.
- [x] Full Playwright result: 309 passed, 5 expected skips.
- [x] Phase 08 Go coverage: 4639/4980 statements (93.2%), matching the exact
      documented repository exception contract.
- [x] Frontend coverage and Phase 07/08 exception contracts passed.
- [x] The task row declares no Task 280 coverage exception; the optional
      producer-timeout residual is a hardening item, not a coverage waiver.

```yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "backend/phase08-coverage.out generated by scripts/check.py"
observed_line_coverage: "Phase 08 Go 4639/4980 statements (93.2%); exact current exception contract validated"
coverage_passed: true
```

## 11. Negative and Regression Checks

- [x] Producer nonzero, missing-result, and spawn-failure paths cannot return 0.
- [x] All later producers execute after an individual failure.
- [x] Child stdout/stderr/traceback and exception secrets do not reach CLI output.
- [x] Producer evidence is retained after the temporary producer directory is
      removed.
- [x] `report.json` and `report.html` evidence cannot collide with top-level
      reports because publication is namespaced.
- [x] Manifest source-kind tampering, duplicate ledger keys, encoded traversal,
      hidden/control/URI paths, sensitive fields, and raw provider summaries
      fail closed.
- [x] Mandatory historical finding baseline rejects deletions and unregistered
      IDs; passing retests close rather than delete findings.
- [x] Atomic finalization leaves no partial report on failure.
- [x] Existing focused, quick, full, task-list, traceability, whitespace,
      static, race, vulnerability, coverage, browser, and local-stack gates
      pass.
- [x] No task-list or production-code edit was made by this reviewer.

The only residual is optional: a producer that hangs indefinitely is not
internally timed out or process-group cancelled. The preparation evidence
explicitly scopes that as an external execution-contract hardening risk, and it
is not one of Task 280’s stated failure/retention acceptance criteria.

## 12. Decision

```yaml
review_decision: PASSED
decision: PASSED
reason: "All Task 280 acceptance criteria and repaired boundaries pass independent source audit, focused regressions, and the full repository gates; no blocking or important finding remains."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "Proceed to the downstream Phase 08.02 real-stack acceptance tasks and keep the optional producer-timeout hardening risk tracked."
```

Task 280 is PASSED for delivery of the acceptance manifest/reporter/finding
synchronization infrastructure. This is not a claim that the future Phase
08.02 UAT gate has accepted all product requirements.

## 13. Repair Context

The prior review’s important findings F-280-08 and F-280-09 were repaired:

1. `execute_producers` now sends both child stdout and stderr to
   `subprocess.DEVNULL`; only fixed producer ID/status codes are emitted by the
   parent. The new regression proves secrets, traceback text, and uncaught
   exception text are absent.
2. `namespace_evidence` and the staging order in `write_report` publish every
   retained artifact below `evidence/` before top-level report files are
   written. The new regression proves same-named `report.json` and
   `report.html` evidence remains byte-for-byte intact and safely linked.

No code or task-list repair was performed by this reviewer. The final review
evidence is the only file written in this review.
