# Task 299 Preparation — Trusted Density Provenance Repair Evidence

## Prior findings carried into this repair

Task 299’s initial implementation and follow-up repairs exposed four integration findings that remain in scope here:

1. Task 297’s saved-diet custom-food migration and owner-scoped custom-food picker had been omitted from the phase-head implementation. Migration `000030`, custom-food repository/API/UI support, and regression tests were restored in commit `71d11016`.
2. External record evidence was initially process-local, which rejected valid search/import flows across API instances. PostgreSQL-backed evidence coordination and migration `000031` were added in commit `1bc95a73`.
3. Evidence cleanup initially used the requested token expiry as its cleanup clock, deleting a still-valid token during a retry with a later expiry. Cleanup was changed to use current time and regression-tested in commit `4ce9d176`.
4. The PostgreSQL cleanup and insert were initially sent as one multi-statement prepared operation with incompatible parameter positions. They were split into embedded statements and executed in one transaction in commit `1bc95a73`.

## Current repair finding

The shared evidence resolver collapsed PostgreSQL connection failures and request cancellation into the same invalid-token sentinel used for malformed, unknown, and expired evidence. The HTTP boundary therefore returned validation `422` for a retryable dependency failure.

The repair preserves `422` for invalid evidence, classifies database outage and cancellation as retryable `503`, and adds regression tests at the external-data, importer-service, and HTTP mapping seams.

## Verification evidence

- Direct PostgreSQL evidence store/resolve, expiry, cleanup, and retry-identity tests pass against the isolated Task 299 database.
- Full backend tests pass with migrations `000030` and `000031` applied.
- Traceability, generated API drift, `git diff --check`, and Go vet pass.

Task-list status rows remain unchanged.
