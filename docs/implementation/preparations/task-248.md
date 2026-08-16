# Task 248 preparation — External Search Proxy

## Outcome and scope

- Task: 248, `DESIGN-009: ExternalSearchProxy`.
- Fixed Git reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Dependencies 245, 246, and 247 were `PASSED`; task 248 was and remains `PREPARED`. This repair did not edit the task list.
- The implementation provides an admin-only `GET /api/v1/admin/external-search` flow across USDA, OpenFoodFacts, or both; one-snapshot normalization; deterministic bounded candidates and warnings; and no food/import/audit mutation.
- No later task was intentionally implemented. In particular, no import confirmation or persistence workflow was added.

## Baseline and unrelated-work preservation

The initial `git status --short` was captured before edits. The worktree was already substantially dirty with Phase 08 changes, including modified shared app/router/repository/API files and untracked task 245–247 implementation. Later-task files for classification, manual-item, and user-admin work appeared and changed concurrently while task 248 was being prepared. They were not edited for task 248, even when their transient compile, test, or traceability failures blocked whole-repository checks.

| Task-relevant baseline path | Baseline state / SHA-256 |
|---|---|
| `api/openapi.yaml` | pre-existing modified, `af5a676d54220079d5f852139a57e0737fee7ffa4e3ca595a6ed302417d4d0c7` |
| `backend/internal/app/app.go` | pre-existing modified, `42dadf16664b29dcfff32cf4b6b1643a2b9e5bdaa0a776b9d63eabd8498b0243` |
| `backend/internal/app/app_test.go` | pre-existing modified, `6bcc71758113a7426356021b9965ef6cdc646c2b85042e7e39b1b631068faf04` |
| `backend/internal/externaldata/normalizer.go` | untracked dependency work, `08bc5afc680300e46d83c4e9f2d59d2aba33f470ff7390fedec503693db78ec4` |
| `backend/internal/externaldata/rate_limit.go` | untracked dependency work; task-245/246 evidence hash `d7649887ac6fb8c960a2df2734f33ca842fc680cce94722a150a52d31a08660b` |
| `backend/internal/httpapi/admin_controller.go` | untracked dependency work, `94d3841f21c30bd2939e2be1ac46d8d984c68a95765e488854335bff6ed6fe3c` |
| proxy and external-search controller files | absent |
| `docs/implementation/02_TASK_LIST.md` | initial row captured as `OPEN`; file changed concurrently, not by task 248 |

Baseline confidence is high for new files and task-owned hunks. Shared-file final hashes include concurrent work and must not be treated as exclusively task-248-owned diffs.

## Implementation

### Provider orchestration and normalization

- `searchExternalRecords` preserves the complete bounded provider projection through retry/rate-limit orchestration instead of discarding serving, package, portion, and nutrient evidence before normalization.
- `ExternalSearchProxy.Search` forces a page size of 20 per selected provider, caps provider output to that page size, loads micronutrient vocabulary once, drops malformed candidates with a closed warning, and returns at most 20 candidates for one provider or 40 for both.
- Candidate order is deterministic by case-folded name, provider, then external ID. Candidate warnings and provider warnings are sorted and deduplicated.
- The API projection omits provider raw payloads and internal normalization details. Strings, maps, arrays, pages, and provider warning values are bounded by the existing normalization boundary and the OpenAPI contract.
- Caller cancellation propagates through retry waits and in-flight provider calls.

### HTTP and production composition

- `(*AdminController).WithExternalSearch` registers one allowlisted, user-rate-limited admin read route using the existing verified JWT-cookie admin middleware and typed curation validator.
- `(*AdminController).SearchExternal` maps only normalized query/provider/page values into the proxy and emits a safe envelope. It has no mutation route metadata and never invokes transactional admin audit coordination.
- `app.NewProduction` composes OpenFoodFacts, optionally configured USDA, the existing provider rate-limit handler, and a PostgreSQL-backed read-only vocabulary normalizer. A missing USDA key degrades that provider to a bounded unavailable warning instead of preventing startup.
- OpenAPI documents authentication, query bounds, response cardinality, candidate field limits, the closed warning vocabulary, and the separate-explicit-import guarantee.

### Review repair

- Repaired finding F-248-1 from `docs/implementation/reviews/task-248-review.md`: `ExternalDataWarning` now declares `json:"provider"`, `json:"code"`, and `json:"message"`, matching the strict OpenAPI schema.
- Added `TestAdminExternalSearchSerializesBoundedWarningsWithContractKeys`, which sends two bounded provider warnings through the real Fiber JSON response and requires exactly the documented lowercase keys and values. The regression fails against the former default `Provider`/`Code`/`Message` encoding.
- Provider warning generation, sorting, deduplication, cardinality bounds, and all prior search behavior remain unchanged.

## Changed paths and symbols

### Production and API

- `backend/internal/externaldata/search_proxy.go`
  - Added `DefaultExternalSearchPageSize`, `ExternalCandidate`, `ExternalSearchResponse`, `ExternalSearchProxy`, `NewExternalSearchProxy`, `(*ExternalSearchProxy).Search`, `sortedUniqueWarnings`, and `sortedUniqueStrings`.
- `backend/internal/externaldata/rate_limit.go`
  - Modified `SearchExternalFoods` to retain its existing projection contract through `searchExternalRecords`.
  - Added `searchExternalRecords`; provider results are capped to requested page size.
  - Added explicit lowercase JSON contract tags to `ExternalDataWarning`.
- `backend/internal/externaldata/normalizer.go`
  - Added `(*DataNormalizer).NormalizeRecordsWithWarnings` and `boundedProvider`.
- `backend/internal/httpapi/external_search_controller.go`
  - Added `ExternalSearchService`, `(*AdminController).WithExternalSearch`, and `(*AdminController).SearchExternal`.
- `backend/internal/httpapi/admin_controller.go`
  - Modified behavioral type `AdminController` with the task-owned `externalSearch` dependency field.
- `backend/internal/app/app.go`
  - Modified `NewProduction` to compose and register the external-search proxy.
- `api/openapi.yaml`
  - Added operation `getApiV1AdminExternalSearch` and schemas `ExternalSearchEnvelope`, `ExternalCandidate`, and `ExternalDataWarning`; extended the file traceability comment.

### Tests

- `backend/internal/externaldata/search_proxy_test.go`
  - Added fakes `proxyVocabulary`, `proxyProvider`, helper `proxyRecord`, and tests covering selection/pagination, deterministic merge, partial and complete outage, cancellation, field/cardinality bounds, warning deduplication, malformed provider records, read-only vocabulary use, and defensive composition errors.
- `backend/internal/httpapi/external_search_controller_test.go`
  - Added `externalSearchServiceStub`, `TestAdminExternalSearchForbidsNonAdminAndDoesNotAuditRead`, `TestAdminExternalSearchSerializesBoundedWarningsWithContractKeys`, and `TestAdminExternalSearchMapsDependencyFailureAndRequiresValidatedInput`.
- `backend/internal/app/app_test.go`
  - Modified `TestNewProductionExposesProductionRoutes` to assert the admin external-search route is composed.

## Security review

The Go security skill was applied in coding mode. Trust boundaries and controls inspected:

- Authorization derives only from verified cookie identity; role/user headers cannot grant admin access.
- Query values pass the typed normalization allowlist before provider dispatch.
- Provider URLs are fixed server configuration, so user input cannot select hosts or schemes; redirects remain disabled in provider clients.
- Provider bodies, record counts, page sizes, field lengths, nutrient maps, warnings, retries, deadlines, and response cardinality are bounded.
- API responses omit API keys, raw provider payloads, provider URLs, and underlying dependency errors.
- Search owns no food/import/audit mutation dependency; the vocabulary repository is read once via `ListActive`. HTTP tests prove transactional admin audit is not called.
- Cancellation is preserved during retry waits and provider execution. Shared quota state remains mutex-protected and race-clean in task packages.

Finding F-248-1 is repaired without expanding the response surface or exposing provider details. No task-248 security finding remained after inspection. `go vet` and `govulncheck` passed.

## Verification

All commands ran on 2026-07-21.

| Command | Result |
|---|---|
| `go test -count=1 -coverprofile=/tmp/task-248-external-repair.cover ./internal/externaldata` | PASS; package 100.0% statements. Every task-owned proxy, record-orchestration, and partial-normalization function reports 100.0%. |
| `go test -count=1 -coverprofile=/tmp/task-248-http-repair.cover ./internal/httpapi -run 'TestAdminExternalSearch'` | PASS before subsequent concurrent interface drift; the lowercase warning serialization regression passes, and `WithExternalSearch` and `SearchExternal` both report 100.0%. A final rerun was blocked only because unrelated `search_controller_test.go` cache doubles no longer implement the concurrently changed `SearchResponseCacheToken` interface. |
| `go test -count=1 ./internal/app -run '^TestNewProductionExposesProductionRoutes$'` | PASS. |
| `go test -count=1 ./...` (`backend/`) | Task-owned `externaldata` and `httpapi` packages PASS. Full command FAILS only in unrelated `internal/app.TestTask240CustomItemErasureIntegration`: transactional account cleanup left two owner custom items. |
| `go test -race -count=1 ./...` (`backend/`) | Task-owned packages PASS with no race report. Full command FAILS at unrelated task-206 timeout and task-240 cleanup integration assertions. |
| `go vet ./...` (`backend/`) | PASS. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` (`backend/`) | PASS: no vulnerabilities in called/imported code; 18 required-module advisories are not called. |
| `npx --no-install redocly lint api/openapi.yaml` | VALID; one existing OAuth callback 302-only warning remains. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks with ordered dependencies; task 248 remains `PREPARED`. |
| `python3 scripts/generate-api-types.py --check` | PASS: generated API types are current. |
| `python3 scripts/validate-traceability.py` | Final rerun FAILS only on concurrent, unrelated declarations in `backend/internal/cache/classification_generation.go` and `backend/internal/search/filter_options.go`. No task-248 path is reported. |
| `git diff --check` | PASS. |
| `python3 scripts/check.py` | Ran as requested; stops immediately at the same unrelated classification/filter-option traceability failures. |

## Final hashes

| Path | SHA-256 |
|---|---|
| `api/openapi.yaml` | `4bbd3ef34268e41a4a37599aa2729ac8589ba0fd34274ae80b27a7e0bbff72f7` |
| `backend/internal/app/app.go` | `c21d0dfcb1cc6ae74c8ee837c211bcaafb875e5a1a74f4724969d3fea2e17df5` |
| `backend/internal/app/app_test.go` | `f267d7813831a91e664355094959c3cfa8ff57e1d096360932f7c9bd503ca9e2` |
| `backend/internal/externaldata/normalizer.go` | `ce77b1a0b255d8685d95a481dcb7090ae3597729b051ea83640c4ed68c143ac1` |
| `backend/internal/externaldata/rate_limit.go` | `ef8cf75d159a64cf58170d243ed1656857c49cd38b2a92734936ceeb3372d846` |
| `backend/internal/externaldata/search_proxy.go` | `d9bb2eb6c389302e792aab29802296fc4fb8f9904f6aa408dca3ebe91242a6c9` |
| `backend/internal/externaldata/search_proxy_test.go` | `0f363626c5aa9797924737222837ea68ca5e0a5edf4d006a07a0c25183c71e6c` |
| `backend/internal/httpapi/admin_controller.go` | `cb1f9bcd0896fadad29c29b8ede663b13c3a2250d99c13c5efe70d05739f730f` |
| `backend/internal/httpapi/external_search_controller.go` | `32086389a2a6ac1d27162ec17cac197f5a103d62cb0d591423ff0344534ac864` |
| `backend/internal/httpapi/external_search_controller_test.go` | `d6f92d11f5d30fd34a5ebdaf5eacaf93fdb3bbb1103fe083bbb29bc84eac7e79` |
| unchanged-status control `docs/implementation/02_TASK_LIST.md` | `9085f355eb060181a16c04ef439c6f30afd67889d3550b971580643d6c379a7d` |

## Handoff

Finding F-248-1 is repaired and covered at the live HTTP serialization boundary. Every task-248 verification criterion is directly covered by focused passing tests and manual boundary inspection. Whole-repository failures are confined to preserved unrelated/concurrent work listed above. No task-list status was edited.
