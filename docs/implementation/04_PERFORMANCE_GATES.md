# Performance Gates

Task 93 gates the current production build and deterministic backend paths with these budgets:

| Area | Target | Evidence |
| --- | --- | --- |
| Search API handler P95 | <= 2 seconds | `go test ./internal/http -run TestSearchControllerP95LatencyGate -count=1` |
| Autocomplete ranking | <= 100 ms | `go test ./internal/services/search -run TestRankAutocompleteLatencyOnSeededData -count=1` |
| Initial JS payload | <= 150 KiB gzip | `python scripts/performance_gates.py` after `bun run build` |
| Initial CSS payload | <= 30 KiB gzip | `python scripts/performance_gates.py` after `bun run build` |
| Initial total HTML/CSS/JS payload | <= 200 KiB gzip | `python scripts/performance_gates.py` after `bun run build` |
| Client heap | <= 64 MiB after initial search workflow | Manual/staging browser measurement until Playwright/browser automation exists in task 99 |

The local API P95 gate uses in-memory dependencies and proves the HTTP envelope, validation, routing, middleware, and serialization path stays under the hard 2-second budget. It is not a substitute for staging load tests against PostgreSQL/Redis/provider-backed data.

The heap target is documented but not enforced in local CI yet because the repository does not currently include browser automation or a Chrome DevTools Protocol harness. Task 99 should promote this to an automated browser measurement.
