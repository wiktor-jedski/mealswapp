# Task 287 preparation evidence

Task 287 re-review repair: refresh the Phase 08 backend coverage contract and
inventory the acceptance-gate symbols added or changed by the raw
OpenFoodFacts boundary repair.

## Coverage contract

The current focused `internal/externaldata` profile records:

| Source | Covered/statements | Exact zero-count ranges |
|---|---:|---|
| `backend/internal/externaldata/openfoodfacts.go` | `184/192` (`95.8%`) | `206.62-208.12,228.10-230.4,248.14-250.12,252.26-254.12,260.2-260.17` |
| `backend/internal/externaldata/rate_limit.go` | `186/188` (`98.9%`) | `357.14-359.3,407.46-409.3` |

The OpenFoodFacts rows cover the malformed-key raw JSON boundary and its
defensive scanner. The rate-limit rows are the exact current uncovered ranges;
no broader file or package exception is claimed.

## Acceptance-gate symbol inventory

The following hand-written symbols are inventoried under DESIGN-014
traceability and are exercised by the Phase 08 acceptance/UAT contract tests:

| Source | Symbols |
|---|---|
| `scripts/phase08_acceptance.py` | `strict_json_loads`, `validate_manifest`, `extract_finding_ledger`, `validate_finding_history`, `validate_findings`, `normalize_results`, `synchronize_results`, `write_report`, `execute_producers`, `command_validate`, `command_report`, `command_run` |
| `scripts/phase08_uat.py` | `sha256`, `validate_source_report`, `canonical_source_reports`, `validate_build_input`, `aggregate`, `verify_special_evidence`, `evidence_hashes`, `render_html`, `render_uat`, `build`, `validate_final`, `command_validate`, `main` |
| `scripts/test_phase08_acceptance.py` | `test_nonpass_requires_matching_unresolved_finding` and the malformed-ledger, evidence-boundary, and report-sanitization regression methods |
| `scripts/test_phase08_uat.py` | `test_committed_report_is_current_and_complete` and the nested-schema, source-linkage, hash, and CLI fail-closed regression methods |

Each source has an adjacent `Implements DESIGN-014 MetricsCollector` module
trace, and `python3 scripts/validate-traceability.py` verifies the inventory
boundary.

## Verification

- `python3 -m unittest scripts/test_phase08_acceptance.py scripts/test_phase08_uat.py`
- `python3 scripts/validate-task-list.py`
- `python3 scripts/validate-traceability.py`
- `python3 scripts/check.py --quick`

PostgreSQL availability check: `pg_isready` reported no response on the local
socket during preparation, so the full database-backed coverage lane was not
claimed as complete. The focused externaldata profile completed successfully.
