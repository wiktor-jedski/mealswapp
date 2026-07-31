# Task 290 acceptance evidence

This sanitized bundle records run `e565f3b003c09aa4d1c1af6a`. `manifest.json` links the exact run to the committed acceptance artifact and the independently inspectable state, diagnostics, audit-row, and final-state documents. `trace.ndjson` contains only safe event categories and the run ID.

The bundle is validated by `validate_run_evidence` in `scripts/run-task290-acceptance.py` and its regression test. No credentials, cookies, CSRF values, URLs, process IDs, or raw API bodies are included.
