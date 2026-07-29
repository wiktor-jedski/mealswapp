# Task 280 review-repair preparation evidence

prepared_at_utc: 2026-07-28T03:56:38Z
task_id: 280
task_status_observed: PREPARED
component: Phase 08.02 Requirement Acceptance Manifest and Finding Synchronization
static_aspect: DESIGN-014 MetricsCollector
baseline_ref: f9a646ffede5c8a63ad787a07eaba7efc0433677
baseline_confidence: HIGH
review_source: docs/implementation/evidence/task-280-review.md

## Outcome

All three blocking and four important Task 280 review findings are repaired.
The authoritative task-list row was not edited by this repair.

- A producer nonzero exit, missing result, or spawn failure is retained as a
  safe report-level producer failure and forces report/process exit code `1`.
  Later producers still execute.
- Validated regular evidence is copied into the atomic report staging tree
  before the temporary producer directory is removed. Missing evidence,
  symlinks, unsafe paths, and copy failures fail closed.
- Finding history is now an unconditional acceptance context input through the
  append-only `finding-history.json` registry. The current ledger and history
  must have exactly the same IDs; deletion and unregistered additions fail.
- Manifest source identity now includes `kind`, so changing a verification step
  into an acceptance criterion, or the reverse, creates source drift.
- Embedded ledger JSON uses the shared duplicate-key-rejecting decoder and
  rejects unsupported fields.
- Evidence links reject percent escapes, query/fragment syntax, control
  characters, hidden components, literal traversal, absolute paths, URI-like
  paths, and backslashes.
- Filesystem, malformed UTF-8, and report-finalization failures cross the CLI as
  bounded messages without traceback, private path, byte, or exception text.
- Producer stdout and stderr are never inherited by the acceptance CLI. An
  uncaught producer exception, its traceback, and arbitrary producer secret
  text are discarded; only fixed producer status codes are emitted.
- Every retained producer artifact is published under `evidence/`. Producer
  artifacts named `report.json` or `report.html` therefore remain available as
  `evidence/report.json` and `evidence/report.html` without overwriting the
  generated top-level report pair.

Task 280 remains test-delivery infrastructure. These repairs do not implement
later requirement-specific scenarios or claim Phase 08.02 UAT acceptance.

## Baseline and confidence

- Repository: `/home/wiktor/Work/mealswapp`
- Branch: `multistep-phase-08`
- HEAD: `f9a646ffede5c8a63ad787a07eaba7efc0433677`
- Worktree: dirty with prior Phase 08 work. Unrelated changes were preserved.
- Review input SHA-256:
  `docs/implementation/evidence/task-280-review.md`
  `8b6fc3078e07e65fadcc8cf89b277143cfd7fcf0306c997a293be17d0b62f889`.
- Repair-start implementation hashes:
  - `scripts/phase08_acceptance.py`:
    `674836de6b2cfbb28074bc30eb09ecdf9f63dbd9fa1a8599df72b5dd2c76ebe1`
  - `scripts/test_phase08_acceptance.py`:
    `133e21caf52167a3ca9186a83e182fbeff4c9c69e4bfcc62c9bd2531894d7de6`
  - `docs/implementation/04_OPEN.md`:
    `08475c75c4092d6bfad9aa9e4c244f1f379a9c07c540297ac05029e568e4a728`
- Remaining-important-repair start hashes:
  - `scripts/phase08_acceptance.py`:
    `f0ecfbccd00b0b10bf5dd1d2b3ed458a6f3b6f0c856101e8e5ff9762b81fd525`
  - `scripts/test_phase08_acceptance.py`:
    `cedc2daa4251f78d604c6ad184c8da991d822cb9e899fe83fe30721a3962c480`
  - `docs/implementation/evidence/task-280-preparation.md`:
    `f727041345350bd6c77714fb35b4b69d9ff7c7265783283f46e66025e60c4bbf`
- Current task-list row is `PREPARED`, changed externally before this repair.
  Its repair-start/current SHA-256 is
  `0097d216ae5b99f2373b365df1fedc900d96b7c0b237f670dc8e9030fc08dab4`;
  this repair did not edit it.
- Confidence: **HIGH**. Every mandatory review finding has a deterministic
  adversarial regression. Focused, quick, full, task-list, traceability,
  compile, and whitespace gates pass.

## Exact changed paths and executable symbols

The remaining-important repair changed exactly:

- `scripts/phase08_acceptance.py`
  - added `EVIDENCE_NAMESPACE` and `namespace_evidence`
  - changed `write_report` and `execute_producers`
- `scripts/test_phase08_acceptance.py`
  - added
    `test_producer_stdout_stderr_and_traceback_are_never_emitted_by_cli`
    and `test_protected_report_names_are_preserved_in_evidence_namespace`
  - updated report-evidence path assertions for the `evidence/` namespace
- `docs/testing/phase08/acceptance-manifest.json-trace.md`
  - documented suppressed producer streams and namespaced retained evidence
- `docs/implementation/evidence/task-280-preparation.md`
  - recorded this repair, verification results, risks, and hashes

The following symbol inventory also retains the earlier review-repair work so
the preparation record remains complete.

### `scripts/phase08_acceptance.py`

New symbols:

- `EVIDENCE_NAMESPACE`
- `strict_json_loads`
- `read_text`
- `validate_finding_history`
- `evidence_paths`
- `namespace_evidence`
- `copy_evidence`
- `acceptance_context`

Changed symbols:

- `load_json`
- `req_test_entries`
- `validate_manifest`
- `extract_finding_ledger`
- `validate_findings`
- `safe_relative_path`
- `write_report`
- `execute_producers`
- `command_validate`
- `command_report`
- `command_run`
- `build_parser`
- `main`

Unchanged executable symbols re-exercised by the repair suite:

- `assert_safe_value`
- `normalize_evidence`
- `normalize_results`
- `synchronize_results`
- `overall_status`
- `load_result_files`

### `scripts/test_phase08_acceptance.py`

New helper:

- `Phase08AcceptanceTests.validate_ledger`

New regression tests:

- `test_report_rejects_missing_and_symlink_evidence`
- `test_embedded_ledger_rejects_duplicate_json_keys`
- `test_run_preserves_evidence_and_nonzero_producer_cannot_return_zero`
- `test_missing_and_spawn_failed_producers_never_return_zero`
- `test_history_registry_is_mandatory_for_normal_cli`
- `test_cli_external_errors_are_sanitized_without_traceback`
- `test_producer_stdout_stderr_and_traceback_are_never_emitted_by_cli`
- `test_protected_report_names_are_preserved_in_evidence_namespace`

Expanded regression tests:

- `test_manifest_rejects_duplicate_and_orphan_mappings`
- `test_manifest_has_required_json_sidecar_traceability`
- `test_mixed_report_is_ordered_correlated_grouped_and_exit_one`
- `test_evidence_links_must_be_safe_and_relative`
- finding-validation/closure tests now always supply complete history
- `test_failure_during_finalization_publishes_no_partial_run`
- `test_all_producers_run_after_individual_failures`
- `test_current_open_document_ledger_is_valid`

All 24 `test_*` methods passed.

### New control files

- `docs/testing/phase08/finding-history.json`
  - mandatory append-only `mealswapp.phase08-finding-history.v1` registry
- `docs/testing/phase08/finding-history.json-trace.md`
  - DESIGN-014 and history-retention sidecar traceability

### Updated documentation/control paths

- `docs/testing/phase08/acceptance-manifest.json-trace.md`
  - documents producer-failure exit semantics, unconditional history
    validation, suppressed producer streams, and the retained `evidence/`
    namespace
- `docs/implementation/04_OPEN.md`
  - adds the mandatory history-registry rule beside the existing Phase 08
    finding ledger
- `docs/implementation/evidence/task-280-preparation.md`
  - replaces pre-review claims with this repair evidence

### Read/verified but unchanged by this repair

- `docs/testing/phase08/acceptance-manifest.json`
- `scripts/check.py`
  - `validate_phase08_acceptance_contracts` already runs the normal validator;
    the validator now loads the mandatory default history registry itself
- `req_tests.md`
- `docs/implementation/02_TASK_LIST.md`
- `docs/implementation/evidence/task-280-review.md`

No backend, frontend, OpenAPI, migration, requirement-specific acceptance
scenario, later task, or task-list status was changed.

## Review-finding closure

| Review finding | Repair and regression |
|---|---|
| Blocking: producer evidence deleted | `copy_evidence` copies each validated regular artifact into report staging. The real two-producer test confirms the screenshot remains readable after `TemporaryDirectory` cleanup. |
| Blocking: producer failure returns 0 | `write_report` accepts safe producer-failure codes and overrides aggregate status/code to `FAIL`/`1`. End-to-end tests cover complete PASS evidence plus exit 7, missing result, spawn failure, and later producer execution. |
| Blocking: history check optional | `acceptance_context` always loads `DEFAULT_FINDING_HISTORY`; `validate_findings` requires exact current/history ID equality. Normal CLI tests reject deletion/unregistered findings and missing history without any optional baseline mode. |
| Important: manifest kind tampering | Source mapping key is now requirement, line, kind, and exact text. Step-to-criterion mutation is rejected as unmapped source drift. |
| Important: duplicate ledger keys | `extract_finding_ledger` uses `strict_json_loads`; duplicate `findings` keys fail. Ledger and finding field sets are exact. |
| Important: encoded traversal | `%`, `?`, `#`, controls, hidden components, absolute/traversal/URI/backslash paths fail; missing files and symlink components also fail before publication. |
| Important: unsanitized CLI exceptions | UTF-8/filesystem reads and finalization are wrapped; `main` handles intended external exception classes with bounded text. Tests prove malformed bytes and injected private write errors emit no traceback/path/payload. |
| Remaining important: producer output/exception leakage | `execute_producers` sends producer stdout and stderr to `subprocess.DEVNULL`; the parent emits only fixed producer diagnostic codes. A real child writes secret text to both streams and then raises an uncaught secret-bearing exception. The CLI output contains neither the secrets nor `Traceback`, returns `1`, and still finalizes the report. |
| Remaining important: protected report artifact overwrite | `namespace_evidence` rewrites every retained link to `evidence/<source-path>` and `write_report` copies into that namespace before writing generated top-level files. The regression proves producer `report.json` and `report.html` bytes survive while generated reports remain valid and link to the retained copies. |

The review's optional bounded producer timeout/process-group enhancement was not
part of the user's requested blocking/important repair. Producer commands still
require an external bounded execution contract; this is recorded below and was
not represented as closed.

## Task acceptance coverage retained

- The 12-scenario/91-criterion manifest remains byte-for-byte unchanged and
  validates every `req_tests.md` source entry exactly once.
- Mixed `PASS`/`FAIL`/`BLOCKED`, deterministic ordering, request IDs, evidence
  links, root grouping, finding linkage, sensitive-data rejection, atomic
  report publication, and criterion exit codes `0/1/2` remain passing.
- Producer output is an untrusted channel: it cannot reach acceptance CLI
  stdout/stderr. Evidence links in JSON and HTML consistently target the
  namespaced retained copies.
- Producer infrastructure failure is intentionally distinct from criterion
  acceptance and can only strengthen the final exit from `0/2` to `1`.
- The finding ledger remains empty because no product acceptance run occurred;
  the mandatory history registry is correspondingly empty. The first finding
  must be added to both in one reviewed change.

## Commands and exact observed results

All commands ran from the repository root on 2026-07-28.

| Command | Result |
|---|---|
| `python3 -m unittest -v scripts/test_phase08_acceptance.py` | PASS, 24 tests in 0.331 seconds. Includes real uncaught producer traceback/secret suppression and protected-name overwrite prevention. |
| `python3 scripts/phase08_acceptance.py validate` | PASS, `scenarios=12 criteria=91`; mandatory history loaded. |
| `python3 -m py_compile scripts/phase08_acceptance.py scripts/test_phase08_acceptance.py scripts/check.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS, 286 sequential tasks with ordered dependencies. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/check.py --quick` | PASS in 80.2 seconds. The 24-test Task 280 lane, 536 frontend tests, 24 changed Playwright tests with 2 expected skips, changed backend packages, static contracts, OpenAPI, vet, vulnerability scan, and generated drift passed. |
| `python3 scripts/check.py` | PASS. Static lane 11.8s; frontend build/unit lane passed with 536 tests; browser lane 113.9s with 309 passes and 5 expected skips; backend lane 374.8s; Phase 08 coverage `4639/4980` (`93.2%`) under the documented exception. |
| `git diff --check` | PASS. |

The existing OpenAPI OAuth callback redirect-only `2XX` warning remains
non-failing and unrelated.

## Current SHA-256 hashes

| Path | SHA-256 |
|---|---|
| `docs/implementation/02_TASK_LIST.md` | `0097d216ae5b99f2373b365df1fedc900d96b7c0b237f670dc8e9030fc08dab4` |
| `req_tests.md` | `241779579f568eecfe40a8d41b97ec2b6a87e6201bd25b8ea76c4bac5c2a8902` |
| `docs/implementation/04_OPEN.md` | `c2f21ddf39942efe3c4e2d3f3b45580ba0e27a72891ee4e2bd14aca6ab76116b` |
| `docs/implementation/evidence/task-280-review.md` | `a7db3a91a907fc10aba4ea648934a10d56d18b1820132198cad4dffe40fde979` |
| `scripts/phase08_acceptance.py` | `cd74a1ab8e1c16d6c7e85425a84e9083a594ff69674ce808654482476d993287` |
| `scripts/test_phase08_acceptance.py` | `ce8962bece793702c2f71af09dc470a7a56187b18662338c8c36b7d3e42198b5` |
| `docs/testing/phase08/acceptance-manifest.json` | `2f597b9545bb176e45a03df02d1b05f01fc424b479ffb4b64c43ee3b6991fa8c` |
| `docs/testing/phase08/acceptance-manifest.json-trace.md` | `2b8f681a27e559b909c7f5d4614d685627050d59c9bdc2d1962ed82b7c0559a5` |
| `docs/testing/phase08/finding-history.json` | `0963e472b2cf2001cd4fe369c75bb4a6ec85f836e821a6f440c523a62d64b105` |
| `docs/testing/phase08/finding-history.json-trace.md` | `f9136c93b2cded40a92ce0d1b2bc92152f43654f596b83fa3a3aa084cce4d2f7` |
| `scripts/check.py` | `6f9ec0c6af49fdf49c79a93d7076aee5cb03f67678943d7da13b634b26f67595` |

This evidence document intentionally omits a self-referential hash.

## Residual risks

- Producer subprocesses still have no internal timeout/process-group
  cancellation. The review classified this as optional. Callers must enforce a
  bounded external timeout until a separately authorized lifecycle enhancement
  is implemented.
- Producer stdout/stderr are intentionally unavailable for diagnostics because
  they are untrusted and cannot be safely classified after emission. Producers
  must place sanitized diagnostic evidence in declared result artifacts.
- Evidence copying validates path syntax, existence, regular-file type, and
  symlink-free ancestry immediately before copying. As with any local
  filesystem check/copy pair, a hostile same-user process could race a path
  after validation; Task 279's run-owned isolated artifact directory is the
  expected trust boundary.
- The append-only history registry is the agreed repository source of
  historical finding IDs. Deleting or rewriting both the ledger and registry
  in one malicious change still requires code-review/VCS history detection;
  ordinary validator execution rejects one-sided deletion or addition.
