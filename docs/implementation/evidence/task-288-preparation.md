# Task 288 Preparation Evidence

Task 288 implements DESIGN-012 `USDAClient` optional-portion tolerance for
SW-REQ-033.

## Controlled-provider acceptance

The disposable Task 282 harness run `c03d6213957858826c10e15e` executed the
production USDA client, backend API, generated frontend client, and
Administration UI against controlled HTTP fixtures.

`P08-SWR033-STEP-02` passed with request-correlated evidence:

- `provider_state=partial_normalization`
- `metric_basis=100ml`
- the USDA candidate remained visible
- no provider-unavailability warning was shown
- the candidate warning reported that unusable optional measures were ignored

The overall Task 282 harness returned nonzero because the independent
OpenFoodFacts metadata criterion remained assigned to Task 287 and the existing
micronutrient-disable criterion remained blocked. Neither non-pass was mapped to
`P08-FIND-282-002`.

## Focused verification

- `go test -count=1 ./internal/externaldata`: PASS
- `go test -race -count=1 ./internal/externaldata`: PASS
- changed USDA decoder symbols: 100% statement coverage; external-data
  package: 99.8% with pre-existing package gaps
- focused external-admin client and ExternalImportWorkflow Bun tests: PASS,
  24 tests
- full frontend tests and coverage: PASS, 538 tests; 95.19% functions and
  96.06% lines under the existing Phase 08 coverage contract
- generated API type drift check: PASS
- frontend production build: PASS
- Task 282 harness contract and API generator Python tests: PASS, 34 tests
- OpenAPI lint: PASS with the pre-existing OAuth callback `2XX` warning
- focused `go vet`: PASS
- `govulncheck ./...`: PASS, no reachable vulnerabilities
- traceability validation: PASS
- `git diff --check`: PASS

The repository-wide `go test ./...` attempt passed the changed external-data,
HTTP, importer, application, and other completed packages, but the repository
package did not complete after 407 seconds and the run was interrupted. The
quick aggregate reached and passed its changed-area Go/frontend, typecheck,
Playwright, vet, and vulnerability lanes, then failed on the phase branch's
pre-existing stale `02_TASK_LIST.md` report hash and missing root `logs/`
directory in unrelated Phase 08 UAT tests.

The decoder fixtures directly cover null amount/unit pairs, valid and invalid
text-only cup/tablespoon evidence, mixed measures, top-level serving pairs,
invalid identity, malformed supported nutrients, ignored malformed unsupported
nutrients, valid-peer isolation, exact FDC-ID and broad-name searches. Existing
focused provider tests retain timeout and rate-limit coverage.
