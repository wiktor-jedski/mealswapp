# Phase 08 finding history traceability

`finding-history.json` implements the append-only finding-history boundary for
Task 280 and `DESIGN-014` `MetricsCollector`.

Every finding ID in the Phase 08 ledger in
`docs/implementation/04_OPEN.md` must be retained here permanently, including
after a passing retest closes the finding. The acceptance validator loads this
registry unconditionally and rejects missing, duplicate, unsorted, deleted, or
unregistered finding IDs.
