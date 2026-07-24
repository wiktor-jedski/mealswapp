# Review Evidence: Task 248 — DESIGN-009 ExternalSearchProxy

~~~yaml
task_id: 248
component: ExternalSearchProxy
static_aspect: DESIGN-009 ExternalSearchProxy provider orchestration, normalized curation projection, warnings, and read-only admin boundary
input_status: PREPARED
review_decision: PASSED
decision: PASSED
reviewed_at_utc: 2026-07-21T15:22:40Z
review_agent: Codex independent reviewer
baseline_ref: 81ca40ce00cb667ea29243ed2d34068e11229a69 plus current task-248 worktree surface
baseline_confidence: HIGH
code_review_skill_invoked: true
relevant_language_guide: Go, HTTP/authentication, serialization, security, concurrency, bounded external-data handling, and error handling
pre_review_gates_passed: true
inventory_complete: true
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
inventory_source_count: 34
audited_symbol_count: 34
blocking_findings: 0
important_findings: 0
optional_findings: 0
~~~

## 1. Task Source

Task 248 is the exact PREPARED row in docs/implementation/02_TASK_LIST.md, titled Phase 08 External Search Proxy and covering DESIGN-009: ExternalSearchProxy. Dependencies 245, 246, and 247 are PASSED. The task-list row remains PREPARED; this review did not edit the task list or production code.

The task owns admin external search across USDA, OpenFoodFacts, or both; deterministic pagination and merge ordering; normalized bounded candidates and warnings; partial and complete outage degradation; cancellation propagation; admin-only access; and a read-only boundary until the later explicit import task.

The full docs/implementation/reviewer-prompt.md template was read. DESIGN-009, DESIGN-012, ARCH-009, ARCH-012, the documented Go/Fiber stack, the relevant SW-REQ-054, SW-REQ-055, and SW-REQ-090 requirements, implementation planning, open-items control, task-248 preparation, and PASSED evidence for tasks 245–247 were checked. The template requests merging the main branch, but no merge was performed because this is a verification-only review in a heavily dirty shared worktree and merging would mutate unrelated user work.

The preparation report and previous review were checked for staleness. Their earlier HTTP rerun, traceability, aggregate, and warning-serialization conclusions were not treated as proof; current source, current hashes, and fresh focused tests are authoritative. The repaired F-248-1 serialization boundary now passes.

## 2. Pre-Review Gates

| Gate | Result | Evidence |
|---|---|---|
| Exact task status | PASS | Current task row 248 is PREPARED; dependencies 245–247 are PASSED. |
| Review template | PASS | docs/implementation/reviewer-prompt.md was read fully. |
| Review skill | PASS | code-review-skill was read and applied exactly once, including Go correctness, security, concurrency, error-handling, architecture, and review-checklist guidance. |
| Go security guidance | PASS | The repository-required golang-security guidance was applied to provider trust boundaries, HTTP authorization, serialization, cancellation, rate limiting, secret/payload handling, and read-only persistence boundaries. |
| Preparation evidence | PASS | Preparation and previous-review claims were compared with current source, current task status, current hashes, focused tests, coverage, and fresh gate results. |
| Scope control | PASS | Only task-248 behavior and its direct callers, dependencies, route controls, designs, requirements, and evidence controls were reviewed. |
| Mutation boundary | PASS | No production file or task-list file was edited. The only intended write is this review evidence file. |

## 3. Review Baseline and Change Surface

The review baseline is HEAD 81ca40ce00cb667ea29243ed2d34068e11229a69. The worktree contains unrelated modified and untracked Phase 08 implementation, migrations, generated API files, task preparations, and reviews. Those changes were preserved and were not attributed to task 248 except where a shared symbol was a direct dependency of the reviewed route.

The task-248 implementation boundary is:

- ExternalSearchProxy.Search and its bounded candidate/warning projection.
- searchExternalRecords and its task-248 record retention through shared rate-limit orchestration.
- DataNormalizer.NormalizeRecordsWithWarnings and its one-vocabulary-snapshot handoff.
- WithExternalSearch, SearchExternal, curation query validation, admin route controls, and shared router composition.
- NewProduction provider/normalizer composition and route registration.
- The external-search OpenAPI operation and schemas.
- Focused task tests, direct provider callers, the vocabulary read boundary, and route-composition tests.

No import, local-food persistence, or audit mutation workflow is reachable from external search. Curated import remains the explicit later task-249 boundary.

## 4. Acceptance Criteria Checklist

| Criterion | Result | Evidence |
|---|---|---|
| Provider selection | PASS | searchExternalRecords selects only the requested provider or both providers for all; focused tests verify USDA-only, OpenFoodFacts-only, and combined calls. |
| Pagination | PASS | ExternalSearchProxy.Search forces page size 20; provider queries receive the requested page and bounded size; each provider result is capped before merge. |
| Deterministic merge ordering | PASS | Candidates sort by case-folded name, provider, then external ID; focused tests verify the exact combined order. |
| Normalization | PASS | One active micronutrient vocabulary snapshot feeds all records; provider aliases, units, density, serving/package conversion, missing data, and warnings are covered. |
| Bounded candidates and warnings | PASS | Provider output is capped, response candidates are bounded, fields and nutrient maps are revalidated, raw payloads are omitted, and warning identities are closed, sorted, and deduplicated. |
| Partial outage | PASS | A successful sibling’s candidates survive an unavailable or rate-limited provider and receive a categorical warning. |
| Complete outage | PASS | Missing selected providers produce no candidates and bounded unavailable warnings without a service error. |
| Cancellation | PASS | Caller cancellation propagates through an in-flight provider call and retry sleep boundary; focused race tests pass. |
| Admin authorization | PASS | Verified JWT-cookie admin middleware returns 403 for authenticated non-admin users and does not dispatch the service; anonymous access remains 401. |
| HTTP validation and dependency errors | PASS | Strict query validation rejects missing, duplicate, extra, unsupported, and invalid page inputs before dispatch; dependency errors map to sanitized 503 responses. |
| Read-only search boundary | PASS | The proxy has no food/import/audit dependency; vocabulary use is ListActive only; HTTP tests prove no admin audit call. |
| Route wiring | PASS | WithExternalSearch supplies typed validation and user-scoped rate limiting; NewProduction composes providers, shared rate limits, vocabulary normalization, and the versioned admin route. |
| Shared rate-limit interaction | PASS | SearchExternalFoods and task-248 orchestration share the mutex-protected, provider-isolated RateLimitHandler; bounded response headers are recorded on success and error, retries preserve caller context, and existing direct callers pass. |
| HTTP warning serialization | PASS | ExternalDataWarning emits exactly lowercase provider, code, and message keys. The real Fiber response regression passes and rejects extra/default Go field names. |
| OpenAPI and traceability gates | PASS | OpenAPI validates with only the existing OAuth 302-only warning; generated types, task-list validation, and traceability validation pass. |
| Focused coverage | PASS | internal/externaldata reports 100.0% statement coverage; task proxy, rate-limit, normalizer, and provider functions report 100.0%; combined curation/external HTTP coverage reports WithExternalSearch, SearchExternal, and ValidateExternalSearchQuery at 100.0%. |

## 5. Changed-Symbol Inventory

| # | File | Reviewed surface | SHA-256 |
|---:|---|---|---|
| 1 | backend/internal/externaldata/search_proxy.go | Proxy response types, constructor, search orchestration, warning and string ordering | d9bb2eb6c389302e792aab29802296fc4fb8f9904f6aa408dca3ebe91242a6c9 |
| 2 | backend/internal/externaldata/search_proxy_test.go | Provider selection, pagination, merge, outages, cancellation, bounds, mutation boundary | 0f363626c5aa9797924737222837ea68ca5e0a5edf4d006a07a0c25183c71e6c |
| 3 | backend/internal/externaldata/rate_limit.go | Provider set, record projection, retry/quota boundary, task-248 record retention, JSON tags | ef8cf75d159a64cf58170d243ed1656857c49cd38b2a92734936ceeb3372d846 |
| 4 | backend/internal/externaldata/rate_limit_test.go | Provider selection, retry, quota, timeout, cancellation, and shared callers | f7ea7b2e01e041cbff3c40688e929214b2919669715009a96ff4f339536ef025 |
| 5 | backend/internal/externaldata/normalizer.go | One-snapshot normalization, validation, bounded warnings, density and unit conversion | ce77b1a0b255d8685d95a481dcb7090ae3597729b051ea83640c4ed68c143ac1 |
| 6 | backend/internal/externaldata/normalizer_test.go | Normalization, warning, density, unit, vocabulary, and overflow coverage | a615b5dacb7635f769e279e979c618655b6e22df288f4d07abc5f331f6b2e8b9 |
| 7 | backend/internal/externaldata/usda.go | USDA provider interface, bounded projection, page request, status/error boundary | 78b198fba01bd7da8ff665b401e5b95b1fa64fdd86fc3df030e9c63864aac116 |
| 8 | backend/internal/externaldata/openfoodfacts.go | OpenFoodFacts provider interface, bounded projection, page request, status/error boundary | e839c6594ab265a11c8ab1b7bb2d295911696a86adc08326bf789e9ba1e0ef57 |
| 9 | backend/internal/httpapi/external_search_controller.go | Read route registration, validation/rate metadata, service dispatch, safe error mapping | 32086389a2a6ac1d27162ec17cac197f5a103d62cb0d591423ff0344534ac864 |
| 10 | backend/internal/httpapi/external_search_controller_test.go | Admin 403, service dispatch, no-audit read, validation, dependency/cancellation errors, warning serialization | d6f92d11f5d30fd34a5ebdaf5eacaf93fdb3bbb1103fe083bbb29bc84eac7e79 |
| 11 | backend/internal/httpapi/admin_controller.go | Admin route allowlist, role enforcement, read/mutation classification | cb1f9bcd0896fadad29c29b8ede663b13c3a2250d99c13c5efe70d05739f730f |
| 12 | backend/internal/httpapi/curation_validation.go | Typed external-search query handoff and strict parameter boundary | 14cd4a46838d84fb643fd9448e8f61c6909f09bf66203ae00d7570d7664c46f1 |
| 13 | backend/internal/httpapi/curation_validation_test.go | Invalid/duplicate/extra query rejection and normalized handoff | f541715892e9d4ecbfabce62e602e029d3a8977e552c9d38bdd35a7780e32292 |
| 14 | backend/internal/httpapi/router.go | Versioned route registration, auth/admin ordering, timeout and error mapping | a98d15348d69f6fdf4d5076c2c12f203b20b10c033a81a6fcc25cd7dfacbcb48 |
| 15 | backend/internal/httpapi/router_test.go | Shared authorization, request-context, error, and route-control tests | 26e5c52e6d0826916664c9dba1a33147baa073b7d4d15be3362ea6862698f7d0 |
| 16 | backend/internal/app/app.go | Production provider/normalizer/admin route composition | 33c22fd95422fe5fbd41b5090c23fcf33a8e4cbf94a6dacf6f0464a869ad0f99 |
| 17 | backend/internal/app/app_test.go | Production route composition smoke coverage | f267d7813831a91e664355094959c3cfa8ff57e1d096360932f7c9bd503ca9e2 |
| 18 | backend/internal/repository/vocabulary_repository.go | Read-only active vocabulary repository boundary | c27715ce33cf4da3a3715f1d8019489dcdb5e81998321dfd2a9c1dd7d27153e6 |
| 19 | backend/internal/repository/sql/vocabulary_list_active.sql | Parameter-free active vocabulary read SQL | 3c949bf8bd2cb92a504a4411da5dc6a272151eda0c876751ffec5bb5eda85950 |
| 20 | api/openapi.yaml | External-search operation, response schemas, bounds, warning vocabulary | 4bbd3ef34268e41a4a37599aa2729ac8589ba0fd34274ae80b27a7e0bbff72f7 |
| 21 | docs/architecture/ARCH-009.md | Admin/external-search architecture and read-before-import flow | 153607ef21b23caad6805f8c0f77e3ad9584dd8ab20dc7c86a54134905a95e91 |
| 22 | docs/architecture/ARCH-012.md | External provider integration, normalization, pagination, degradation | 8377243ff9409b27ac9c43de556f0ff094c163ca465abe562ee8cec5721ed435 |
| 23 | docs/design/DESIGN-009.md | ExternalSearchProxy responsibilities, interfaces, errors, import handoff | 85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b |
| 24 | docs/design/DESIGN-012.md | Provider, normalizer, warning, rate-limit, and partial-result contracts | 53ac9bd6a34bd07216666d4beaae6533a0281c905fc2dc5c474f48f614746eddf |
| 25 | docs/design/01_TECH_STACK.md | Go/Fiber/PostgreSQL/OpenAPI/security stack constraints | 64e2cf45ec039db597244678b17e8028f4705b86dcad01e7051e3e686d6f9338 |
| 26 | docs/requirements/01_SOFT_REQ_SPEC.md | SW-REQ-054 admin access, SW-REQ-055 external curation, SW-REQ-090 vocabulary | 80b2f57a8c1caebd8b37cdb949cc7e928f3a128a2b8ed81313637b919cafba8b |
| 27 | docs/implementation/01_PLAN.md | Phase 08 implementation scope and sequencing | 59fef9bf6f8c1cf058533ab296e87d9264d091cbcf204b56a2ff6b8dbfa4ba1d |
| 28 | docs/implementation/04_OPEN.md | Current open assumptions/deviations control | 4b703eca5a8b6207ce0e87fc0ea23c9df255ac8923ab8d93e8b4f8ff1f318e4d |
| 29 | docs/implementation/reviewer-prompt.md | Review role, template, merge/scope/output instructions | 92c9b71361a50868becf0b9a9895071bdd657e8c092afb8c1b19691cb569386d |
| 30 | docs/implementation/preparations/task-248.md | Preparation scope, claims, commands, and fingerprints | 45276fa0626bbcdf673d0293395c0b13a978c45dc8e55f66a39ad0b2ac0e624a |
| 31 | docs/implementation/reviews/task-245-review.md | PASSED provider retry/rate-limit prerequisite evidence | 22424777517d81ff005ea9e2c6fc1b726ef2fa83b511970fcacae7b0524780d9 |
| 32 | docs/implementation/reviews/task-246-review.md | PASSED normalization prerequisite evidence | 2e2abfd0b47d158f2ed00d51cb33c4bf62c6aaa5c35d936aae4a274f83ba72bc |
| 33 | docs/implementation/reviews/task-247-review.md | PASSED admin-gateway prerequisite evidence | 6b53354c55d63d06327876eb8ed5fe9dc450125addee4a59ff901ba4306dea10 |
| 34 | docs/implementation/02_TASK_LIST.md | Current status/dependency/verification control; unchanged by review | 9085f355eb060181a16c04ef439c6f30afd67889d3550b971580643d6c379a7d |

## 6. Function-Level Audit

| # | Symbol/surface | Result | Evidence |
|---:|---|---|---|
| 1 | ExternalSearchProxy.Search | PASS | Validates composition, forces page size, calls bounded orchestration and one normalization snapshot, shapes candidates, sorts results, and deduplicates warnings. |
| 2 | searchExternalRecords | PASS | Selects providers deterministically, checks quota, propagates caller cancellation, retries through the shared handler, preserves partial results, and caps each provider page. |
| 3 | sortedUniqueWarnings | PASS | Sorts by provider/code and deduplicates the closed warning identity without exposing provider diagnostics. |
| 4 | sortedUniqueStrings | PASS | Candidate warning lists are sorted, deduplicated, and non-nil when empty. |
| 5 | NormalizeRecordsWithWarnings | PASS | Loads ListActive once, drops malformed records with one bounded warning identity, and returns normalized candidates without persistence. |
| 6 | validateExternalRecord and boundedProvider | PASS | Revalidate provider identity, IDs, names, URLs, nutrient count/keys/values, and bound malformed-provider warning identity. |
| 7 | resolveDensity and trustedUSDADensity | PASS | Preserve explicit density provenance and use documented trusted USDA volume priority without a silent 1 ml = 1 g assumption. |
| 8 | RateLimitHandler and SearchExternalFoods boundary | PASS | Quota state is mutex-protected and provider-isolated; bounded response headers update the correct provider; retry sleeps and calls retain caller cancellation/deadline identity. |
| 9 | USDAClient.SearchResult | PASS | Fixed server endpoint, bounded query/page/body, no redirects, safe status mapping, response-header projection, and loss-bounded records. |
| 10 | OpenFoodFactsClient.SearchResult | PASS | Fixed server endpoint, caller identification, bounded query/page/body, no redirects, safe status mapping, and loss-bounded records. |
| 11 | AdminController.WithExternalSearch | PASS | Registers only a GET read route with typed validation and a user-scoped limit; it declares no mutation/audit metadata. |
| 12 | AdminController.SearchExternal | PASS | Dispatches only the normalized request, maps safe dependency/cancellation errors, and serializes the repaired warning type through the live JSON response. |
| 13 | ValidateExternalSearchQuery | PASS | Requires exactly query/provider/page, rejects duplicates and extras, normalizes typed values, and prevents provider dispatch on invalid input. |
| 14 | AdminController.Routes and RequireAdmin | PASS | The route is classified as read-only and receives verified cookie authentication plus server-derived admin role enforcement. |
| 15 | registerV1Routes | PASS | Middleware ordering places auth, admin role, validation, and rate limiting before the handler; no audit middleware is attached to this read route. |
| 16 | NewProduction | PASS | Composes optional USDA, OpenFoodFacts, shared rate limits, PostgreSQL vocabulary reads, and the versioned admin route without importing or persisting candidates. |
| 17 | PostgresMicronutrientVocabularyRepository.ListActive | PASS | The proxy’s only repository call is the active-vocabulary read; no Upsert, food write, import, or audit method is reachable from search. |
| 18 | Direct callers/tests/OpenAPI projection | PASS | Existing shared rate-limit callers, provider tests, curation HTTP tests, live warning serialization, generated types, and strict OpenAPI schema all pass. |
| 19 | vocabulary_list_active.sql | PASS | The task path uses a parameter-free active-vocabulary read statement and no repository write statement. |
| 20 | api/openapi.yaml external-search operation and schemas | PASS | The route, auth/status responses, candidate bounds, warning bounds, lowercase warning properties, and strict additional-property policy match the live projection. |
| 21 | ARCH-009 administration flow | PASS | The admin proxy is role-restricted, external-only during search, and separated from later explicit import persistence. |
| 22 | ARCH-012 external-data integration | PASS | Provider fetching, normalization, graceful degradation, pagination, and rate-limit responsibilities align with the implementation. |
| 23 | DESIGN-009 static responsibilities and interfaces | PASS | Admin routing, ExternalSearchProxy shaping, typed request flow, and import handoff boundaries align with the design. |
| 24 | DESIGN-012 provider/normalizer/rate-limit contracts | PASS | Closed warnings, provider projections, retries, density/unit conversion, one vocabulary snapshot, and partial results align with the design. |
| 25 | DESIGN-01_TECH_STACK constraints | PASS | Go/Fiber, PostgreSQL, OpenAPI, HTTP security, and provider integration constraints are followed by the reviewed path. |
| 26 | SW-REQ-054, SW-REQ-055, and SW-REQ-090 | PASS | Admin access, external curation search, and canonical vocabulary use are covered by route, service, and repository evidence. |
| 27 | implementation plan Phase 08 scope | PASS | Search precedes import, remains read-only, and composes with the planned Phase 08 sequence. |
| 28 | docs/implementation/04_OPEN.md control | PASS | No task-248 deviation or unauthorized coverage exception is required by the reviewed behavior. |
| 29 | reviewer-prompt.md control instructions | PASS | Required task scope, design alignment, traceability, merge decision, and review output were followed within the non-mutating worktree constraint. |
| 30 | task-248 preparation report | PASS | The repaired claim was independently checked; stale warning and aggregate claims were superseded by fresh evidence. |
| 31 | task-245 PASSED prerequisite evidence | PASS | Shared provider retry/rate-limit behavior was rechecked at the current caller boundary. |
| 32 | task-246 PASSED prerequisite evidence | PASS | Normalization, vocabulary, density, unit, and warning prerequisites remain compatible with task 248. |
| 33 | task-247 PASSED prerequisite evidence | PASS | Verified authentication, admin role enforcement, route gateway ordering, and read-only controls remain compatible. |
| 34 | current task-list row 248 and dependency control | PASS | Task 248 is still PREPARED, dependencies are satisfied, and the control hash is unchanged by this review. |

## 7. Findings

No blocking, important, or optional findings remain.

The previous review’s F-248-1 is closed. ExternalDataWarning now declares JSON tags mapping Provider to provider, Code to code, and Message to message.

The documented warning response is exercised through the real Fiber route by TestAdminExternalSearchSerializesBoundedWarningsWithContractKeys. For representative values, the live JSON shape is:

~~~json
{"provider":"usda","code":"provider_unavailable","message":"provider_unavailable"}
~~~

The test requires exactly those three lowercase keys for bounded provider warnings; it would fail against Go’s former default Provider/Code/Message encoding. Warning generation remains categorical and bounded, provider and normalizer warnings are sorted/deduplicated, and no raw payload, URL, secret, or underlying error is serialized.

## 8. Commands Run

| Command | Result | Evidence |
|---|---|---|
| GOCACHE="$PWD/.go-cache" GOMODCACHE="$PWD/.go-mod-cache" go test -count=1 -coverprofile=/tmp/task-248-external-rereview.cover ./internal/externaldata | PASS | Package statement coverage 100.0%; task proxy, rate-limit, normalizer, and provider functions report 100.0%. |
| GOCACHE="$PWD/.go-cache" GOMODCACHE="$PWD/.go-mod-cache" go test -count=1 -coverprofile=/tmp/task-248-http-rereview.cover ./internal/httpapi -run TestAdminExternalSearch | PASS | Live warning regression passes; WithExternalSearch and SearchExternal report 100.0%. |
| GOCACHE="$PWD/.go-cache" GOMODCACHE="$PWD/.go-mod-cache" go test -count=1 -coverprofile=/tmp/task-248-http-combined-rereview.cover ./internal/httpapi -run TestAdminExternalSearch\|TestCuration | PASS | ValidateExternalSearchQuery, WithExternalSearch, and SearchExternal report 100.0%. |
| GOCACHE="$PWD/.go-cache" GOMODCACHE="$PWD/.go-mod-cache" go test -count=1 ./internal/httpapi -run ExternalSearch\|CurationRequestValidator | PASS | External-search route, validation, dependency, authorization, and typed-handoff tests pass. |
| GOCACHE="$PWD/.go-cache" GOMODCACHE="$PWD/.go-mod-cache" go test -count=1 ./internal/app -run ^TestNewProductionExposesProductionRoutes$ | PASS | Production external-search route composition passes. |
| GOCACHE="$PWD/.go-cache" GOMODCACHE="$PWD/.go-mod-cache" go test -race -count=1 ./internal/externaldata ./internal/httpapi | PASS | Complete task-local externaldata and HTTP packages pass with no race report. |
| GOCACHE="$PWD/.go-cache" GOMODCACHE="$PWD/.go-mod-cache" go test -race -count=1 ./internal/app -run ^TestNewProductionExposesProductionRoutes$ | PASS | Production route composition passes under race detection. |
| GOCACHE="$PWD/.go-cache" GOMODCACHE="$PWD/.go-mod-cache" go test -count=1 ./internal/repository -run External\|Vocabulary\|Admin\|Search | PASS | Direct vocabulary/admin/search-related repository tests pass. |
| GOCACHE="$PWD/.go-cache" GOMODCACHE="$PWD/.go-mod-cache" go vet ./... | PASS | No vet findings. |
| GOCACHE="$PWD/.go-cache" GOMODCACHE="$PWD/.go-mod-cache" go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./... | PASS | No vulnerabilities in called/imported code; 18 required-module advisories are not called. |
| npx --no-install redocly lint api/openapi.yaml | PASS with existing warning | OpenAPI is valid; only the pre-existing OAuth callback 302-only warning remains. |
| python3 scripts/validate-task-list.py | PASS | 263 ordered tasks; task 248 remains PREPARED. |
| python3 scripts/validate-traceability.py | PASS | No traceability failures in the current worktree at review time. |
| python3 scripts/generate-api-types.py --check | PASS | Generated API types are current. |
| git diff --check | PASS | No whitespace errors. |
| go test -count=1 ./... | FAIL outside task 248 | Task-owned externaldata and httpapi packages pass. Full run reports unrelated task-206 Redis integration timeout and task-240 custom-item erasure cleanup failure. |
| go test -race -count=1 ./... | FAIL outside task 248 | Task-owned packages pass with no race report; full run reports unrelated task-240 custom-item erasure cleanup failure. |
| python3 scripts/check.py | FAIL outside task 248 | Static, task-list, OpenAPI, security, vet, and focused tests pass; local-stack verification stops on the existing PostgreSQL duplicate pg_type error while applying migration 000004_micronutrient_vocabulary. |
| python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-248-review.md | PASS | Final evidence is structurally valid with 34 inventory rows, 18 audited-symbol rows, and zero findings. |

The repository-wide normal and race commands were initially launched concurrently against the shared test database, so a serial -p 1 rerun was also attempted. It reproduced the unrelated task-240 failure; stale orphaned repository test binaries from the interrupted concurrent run were terminated, so that serial aggregate was not used as a task-248 gate. No task-local result depended on the database lock contention.

## 9. Files Inspected and Staleness Fingerprints

Section 5 is the complete 34-entry SHA-256 inventory for the reviewed source, tests, direct callers, provider clients, route composition, API contract, architecture/design/requirements, preparation, prerequisite evidence, reviewer template, and current task-list control.

The current worktree task-list hash is 9085f355eb060181a16c04ef439c6f30afd67889d3550b971580643d6c379a7d; task 248 is still PREPARED and this review did not change it. The repaired shared rate_limit.go hash is ef8cf75d159a64cf58170d243ed1656857c49cd38b2a92734936ceeb3372d846, and the repaired HTTP regression test hash is d6f92d11f5d30fd34a5ebdaf5eacaf93fdb3bbb1103fe083bbb29bc84eac7e79.

Preparation-era evidence was stale where it described the old uppercase JSON output, a blocked final HTTP rerun, or traceability failures from concurrent unrelated declarations. Those claims were superseded by the current source, current hashes, live HTTP serialization test, complete task-local race run, and current validators.

The dirty worktree contains unrelated later-task files and shared-file edits. Shared files were reviewed only for task-248 symbols and direct behavior called by external search. No unrelated implementation was modified or attributed to this task.

## 10. Coverage and Exceptions

The task row declares no testing coverage exception. The focused externaldata package reaches 100.0% statement coverage, with all task-owned proxy, shared rate-limit, normalization, and provider boundary functions at 100.0%. The combined external-search/curation HTTP profile reports ValidateExternalSearchQuery, WithExternalSearch, and SearchExternal at 100.0%. Complete task-local race coverage passes for externaldata and HTTP packages.

The failed whole-repository gates are not task-248 coverage exceptions: they are unrelated dirty-worktree integration behavior and a pre-existing local PostgreSQL migration-state failure. They do not implicate task-248 symbols, and all relevant task-local tests, race checks, static analysis, security scan, OpenAPI validation, traceability, and coverage checks pass.

## 11. Negative and Regression Checks

| Check | Result | Evidence |
|---|---|---|
| No local food mutation | PASS | Proxy composition contains providers, rate limits, and a vocabulary reader only; no food repository write is passed to the service. |
| No import mutation | PASS | No importer or confirmation route is called or introduced; task 249 remains the explicit import boundary. |
| No audit mutation | PASS | GET route has no mutation/audit metadata; HTTP tests leave the admin audit coordinator untouched. |
| No raw provider payload | PASS | Provider projections discard raw payloads, normalizer copies bounded canonical fields, and API candidate types omit raw payloads. |
| No admin spoofing | PASS | Authorization derives from verified JWT-cookie claims; role and identity headers/body/query are not accepted. |
| Provider outage degradation | PASS | Partial results survive; complete provider outage returns empty candidates and categorical warnings. |
| Cancellation | PASS | In-flight cancellation returns context.Canceled through provider orchestration; retry sleeps use caller context and deadlines remain bounded. |
| Cardinality/field bounds | PASS | Per-provider page cap, combined response cap, bounded strings/maps/warnings, finite values, and strict query limits are enforced. |
| Warning JSON contract | PASS | Lowercase provider, code, and message are emitted through the live response with no extra default Go keys. |
| Route status and path | PASS | Production route smoke test passes; route is versioned and admin-gated. |
| Worktree safety | PASS | No production or task-list edit was made; only this review evidence file was refreshed. |

## 12. Decision

~~~yaml
review_decision: PASSED
decision: PASSED
blocking_findings: 0
important_findings: 0
optional_findings: 0
recommended_next_action: Task 248 satisfies the reviewed criteria. A project owner may transition the unchanged task-list row from PREPARED when the repository-wide unrelated integration and local-stack state are separately resolved.
task_status_after_review: PREPARED
~~~

Task 248 passes this independent re-review. No production repair or task-list status transition was performed.

## 13. Repair Context

This is an independent re-review of the current repaired PREPARED task. The previous F-248-1 finding required documented lowercase warning keys and a live HTTP serialization regression. The current ExternalDataWarning tags and TestAdminExternalSearchSerializesBoundedWarningsWithContractKeys satisfy both requirements without expanding the response surface.

The reviewer did not merge branches, edit production code, edit docs/implementation/02_TASK_LIST.md, change open-items documentation, or alter unrelated dirty-worktree work. The only intended write is docs/implementation/reviews/task-248-review.md.
