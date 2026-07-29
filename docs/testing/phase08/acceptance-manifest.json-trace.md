# Phase 08 acceptance manifest traceability

`acceptance-manifest.json` implements `DESIGN-014` `MetricsCollector` for Task
280. It maps every Phase 08 verification step and acceptance criterion in
`req_tests.md` to one stable criterion ID, requirement, scenario owner, required
environment, and permitted evidence type.

The manifest covers `SW-REQ-019`, `SW-REQ-032`, `SW-REQ-033`, `SW-REQ-043`,
`SW-REQ-054`, `SW-REQ-055`, `SW-REQ-056`, `SW-REQ-057`, `SW-REQ-072`,
`SW-REQ-073`, `SW-REQ-084`, and `SW-REQ-090`.
`scripts/phase08_acceptance.py validate` rejects source drift, orphan or
duplicate mappings, and incomplete scenario metadata.

Use `phase08_acceptance.py report --run-id <run-id> --result <result.json>` to
finalize already produced criterion results. Use `run --plan <plan.json>` to
invoke multiple Playwright/backend producers without stopping after a nonzero
producer exit; each producer writes its declared relative result beneath the
temporary directory exposed as `PHASE08_ACCEPTANCE_RESULT_DIR`. Validated
requirement-delivery tasks may add `--requirement <SW-REQ-ID>` to finalize one
complete manifest scenario before later requirement producers exist; manifest
and finding-history validation remains global and unchanged. Validated
regular evidence files are copied into report staging before that producer
directory is removed; missing files, symlinks, encoded traversal, and unsafe
paths fail closed. Retained artifacts are published beneath the `evidence/`
namespace, so producer artifacts named `report.json` or `report.html` cannot
overwrite generated report files. Producer stdout and stderr are discarded;
the CLI emits only bounded status codes for producer failures. The report,
HTML, and retained evidence are atomically published beneath
`logs/phase08-acceptance/<run-id>/`. Any nonzero, missing, or unspawnable
producer forces report and process exit code `1` even when all published
criterion results say `PASS`.

Every command also loads `finding-history.json`; history checking is not an
optional mode. A new finding must be added to the current Phase 08 ledger and
the append-only history registry together. Closing changes the retained ledger
record and never removes its history ID.
