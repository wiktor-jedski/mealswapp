# Task 241 Preparation — Backend-Owned Search Filter Options

## Outcome and task state

- Task: **241 — Phase 08 Backend-Owned Search Filter Options**.
- Result: **prepared and repaired in code and verification evidence; task-list status intentionally remains `PREPARED`**.
- Fixed implementation reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Preparation and repair date: 2026-07-21, Europe/Warsaw.
- Dependencies 34, 117, and 138 were re-read from `docs/implementation/02_TASK_LIST.md` and remain `PASSED`.
- The repair did not edit `docs/implementation/02_TASK_LIST.md`. Its pre-repair and final SHA-256 are both `ca6252275e342f390e767783df7b884f88f7ab1b7df169976c218bd55df148a5`; row 241 remains `PREPARED`.
- Baseline confidence: **HIGH**. Every new Task 241 path was absent at baseline. `backend/internal/httpapi/search_validation.go` and `backend/internal/app/app_test.go` were clean at baseline. `backend/internal/app/app.go` was already modified; its exact pre-task hash was captured and only the two Task 241 composition statements identified below were added.
- The phase-orchestrator skill required delegation, but no writable subagent tool was available in this session. The parent applied the preparation contract directly and did not perform a status transition or acceptance review.

## Baseline capture

Baseline command results:

- `git rev-parse HEAD`: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- `git status --short`: the worktree already contained Task 238–240 and unrelated changes. No baseline path was cleaned, reverted, staged, or overwritten.
- Pre-existing tracked changes: `api/openapi.yaml`, `backend/cmd/worker/main.go`, `backend/internal/app/app.go`, `backend/internal/httpapi/profile_controller.go`, repository/account/compliance/deletion files and SQL, `backend/internal/repository/types.go`, userdata files, `docs/design/DESIGN-005.md`, `docs/implementation/02_TASK_LIST.md`, `docs/implementation/04_OPEN.md`, `frontend/src/lib/api/generated.ts`, and API generation scripts/tests.
- Pre-existing untracked work: Task 238 custom-item persistence/migrations, Task 239 custom-item API files, Task 240 deletion/cache integration files, preparation/review evidence for Tasks 238–240, and migrations 25–26.
- Pre-task SHA-256 for the overlapping `backend/internal/app/app.go`: `45de7f3b7de3515120e2a94c9eb3c40dfbbed95d8e29f64fa19f94f83e92d97b`.
- Pre-task SHA-256 for already-modified `backend/internal/repository/types.go`: `a959e11147b0ff709b885ae0c58870c3628ad545cfd99f8e5c53efb5f1116cde`. Task 241 briefly considered this shared file, then removed its additions; the final hash is exactly the same, so it is not part of the Task 241 surface.
- All other Task 241 implementation paths listed below were either clean tracked files or did not exist at baseline.

## Review repair

`docs/implementation/reviews/task-241-review.md` rejected the preparation because the real migration runner executes every up migration on every invocation and migration 27's conflict branch set `deleted_at = NULL`. Repeating `migrate up` therefore reactivated a soft-deleted `dairy` allergen.

The repair is limited to two implementation/test paths:

- `database/migrations/000027_allergen_vocabulary.up.sql:23-26`, allergen seed conflict update: removed only `deleted_at = NULL`. Existing canonical `name` and `label_key` refresh plus `updated_at` behavior remain unchanged.
- `backend/internal/repository/allergen_vocabulary_repository_test.go:58`, `TestAllergenVocabularyMigrationReplayPreservesInactiveDairy`: soft-deletes `dairy`, replays all up migrations through `migrations.Run`, proves `deleted_at IS NOT NULL`, and proves `ListActive` still excludes dairy.

Red/green evidence:

- Before the SQL repair, the new focused test failed with `dairy became active after repeat migrations up`.
- After the SQL repair, the same test passed, followed by the focused, full, race, vet, vulnerability, migration, traceability, OpenAPI, and aggregate checks recorded below.
- Pre-repair hashes from the independent review were `46e9dc30cf9d6481e7291608926da50da2099cf8fe86860e91e460cf47d0a828` for migration 27 and `e297fed65ce32e4522c44348ff57777c1753e55069e2c78730a2f1207adca764` for the repository test. Final hashes are in the manifest below.

## Design and requirement inspection

The implementation was bounded by:

- `docs/design/DESIGN-009.md`: `TagManager` owns global classification identity and administration-facing invalidation responsibilities.
- `docs/design/DESIGN-002.md` and `backend/internal/search/filter_processor.go`: canonical filter kinds, physical-state IDs, Dietary Preset IDs, and preset-to-allergen exclusion policy.
- `docs/design/DESIGN-005.md` and the existing classification repository: active Food Category and Culinary Role persistence and deterministic listing.
- `docs/requirements/01_SOFT_REQ_SPEC.md`: SW-REQ-019 classification filtering and SW-REQ-057 global classification management.
- `docs/implementation/01_PLAN.md` and `docs/implementation/04_OPEN.md`: the explicit Phase 08 requirement to replace frontend-hardcoded substitution options with persisted vocabularies and backend policy. The open notes confirmed that a dedicated persisted allergen vocabulary did not yet exist.

Scope decisions:

1. Migration 27 adds the missing active allergen vocabulary and seeds only the allergen keys already accepted by the backend FilterProcessor. It does not add later admin CRUD.
2. The service owns fallback labels, localization keys, include/exclude policy, preset dependency projection, deterministic group/label/ID ordering, and an explicit process-local invalidation seam for Task 251 to invoke after committed administration changes.
3. Food Category and Culinary Role IDs come directly from persisted UUIDs; allergen IDs come directly from persisted keys; physical-state and Dietary Preset IDs come from existing backend domain constants/policy. The HTTP route only maps service values and invents no frontend-oriented IDs.
4. The route is intentionally anonymous and read-only. It has query validation and endpoint rate limiting, but no auth, CSRF, audit-mutation, or role-header trust.
5. Task 253 explicitly owns the combined Phase 08 OpenAPI and generated-client contract, and Task 257 owns frontend consumption. Neither later task was implemented here. The current OpenAPI source remains untouched by Task 241 and still lints successfully.

## Exact changed paths and symbols

| Path | Task 241 additions or modifications |
| --- | --- |
| `database/migrations/000027_allergen_vocabulary.up.sql` | Adds `allergen_vocabulary`, active/deleted state, normalized keys, fallback names, localization keys, seven backend-supported seed entries, and schema version 27; repeat seed conflicts refresh canonical metadata without changing `deleted_at`. |
| `database/migrations/000027_allergen_vocabulary.down.sql` | Drops the vocabulary and schema-version record. |
| `backend/internal/repository/sql/allergen_vocabulary_list.sql` | Embedded parameter-free active vocabulary read ordered by normalized display name and persisted key. |
| `backend/internal/repository/allergen_vocabulary_repository.go` | `allergenVocabularyListSQL`; `AllergenVocabularyEntry`; `AllergenVocabularyRepository`; `PostgresAllergenVocabularyRepository`; compile assertion; `NewPostgresAllergenVocabularyRepository`; `ListActive`. |
| `backend/internal/repository/allergen_vocabulary_repository_test.go` | `TestPostgresAllergenVocabularyRepositoryListsOnlyActiveEntries`; `TestAllergenVocabularyMigrationReplayPreservesInactiveDairy`; `TestPostgresAllergenVocabularyRepositoryClassifiesFailures`. |
| `backend/internal/search/filter_options.go` | `FilterOptionReference`; `FilterOption`; `FilterOptionsResponse`; repository seams; `FilterOptionService`; `NewFilterOptionService`; `Options`; `Invalidate`; `cachedOptions`; `load`; `classificationFilterOptions`; `physicalStateFilterOptions`; `dietaryPresetFilterOptions`; `sortFilterOptions`; `cloneFilterOptionsResponse`. |
| `backend/internal/search/filter_options_test.go` | Thread-safe repository doubles and methods; `TestFilterOptionServiceProjectsDeterministicLocalizedPolicy`; `TestFilterOptionServiceCachesCopiesAndInvalidatesAfterAdministration`; `TestFilterOptionServiceValidatesModeAndReturnsDependencyFailures`; `TestSortFilterOptionsBreaksEqualLabelTiesByPersistedID`; `findFilterOption`. |
| `backend/internal/search/filter_options_integration_test.go` | `TestFilterOptionServiceReloadsActivePersistedVocabularyAfterAdministration`; `hasFilterOption`. |
| `backend/internal/httpapi/filter_option_controller.go` | `FilterOptionReader`; `FilterOptionController`; DTO types; compile assertion; `NewFilterOptionController`; `Routes`; `Get`. |
| `backend/internal/httpapi/filter_option_controller_test.go` | `filterOptionReaderStub`; its `Options` method; anonymous/public projection test; invalid-mode and safe structured-dependency-error test. |
| `backend/internal/httpapi/search_validation.go` | Added `ValidateFilterOptionQueryParams` only. |
| `backend/internal/app/app.go` | Modified `NewProduction` only: constructs `FilterOptionService` from PostgreSQL classification/allergen repositories and registers `FilterOptionController`. All other visible diff in this already-dirty file predates Task 241. |
| `backend/internal/app/app_test.go` | Modified `TestNewProductionExposesProductionRoutes` with the filter-options route composition assertion. |
| `docs/implementation/preparations/task-241.md` | This baseline, implementation, verification, symbol, hash, and risk record. |

All Go and SQL additions carry concise `DESIGN-009 TagManager` traceability comments. No JSON file was changed, so no JSON sidecar was required.

## Verification criteria

| Task criterion | Direct evidence | Result |
| --- | --- | --- |
| Deterministic localized-label-ready DTO ordering | Service test starts with deliberately unsorted classifications/allergens and verifies physical → category → role → allergen → preset groups, case-insensitive labels, ID tie-breaks, fallback labels, and policy localization keys. SQL independently orders active allergens. | PASS |
| Supported mode validation | Service rejects non-substitution mode before repository I/O; HTTP validation rejects absent, `catalog`, and extra-parameter requests before dispatch. | PASS |
| Inactive classification exclusion | PostgreSQL integration creates and soft-deletes a Food Category, then proves it never enters service options before or after invalidation. | PASS |
| Active persisted Allergens | Migration/repository integration verifies seven persisted entries, localized metadata, soft-deleted exclusion, a non-nil empty result after all are inactive, and preservation of dairy's inactive state across real-runner repeat migration-up. | PASS |
| Allergen and Dietary Preset policy projection | Unit test verifies allergens are exclusion choices and Vegan projects the exact `animal_product`, `dairy`, and `egg` rules used by `ApplyFilters`; all five existing backend presets are present. | PASS |
| Physical-state options | Service projects repository-owned `solid` and `liquid` identities with localization keys and include/exclude policy. | PASS |
| Empty vocabulary | Empty classification and allergen doubles return exactly the two physical states plus five backend Dietary Presets, without nil or invented persisted choices. | PASS |
| Cache invalidation after administration changes | Unit and live PostgreSQL tests prove cached isolation, unchanged cached reads after an upsert, explicit `Invalidate`, and refreshed active labels/IDs afterward. Generation checking prevents a load concurrent with invalidation from repopulating stale cache state. | PASS |
| Public-read security | Route metadata proves GET, no required/optional auth, and no CSRF. Anonymous HTTP succeeds; a spoofed role header has no authority or behavioral effect. Endpoint rate limiting is present. | PASS |
| Structured dependency failures | Repository maps query/scan/iteration failures to `connection`; service returns classification/allergen failures; HTTP returns safe retryable 503 `dependency_unavailable` without repository messages or socket details. | PASS |
| No hardcoded frontend IDs invented by route | HTTP test injects sentinel repository/policy IDs and receives those exact IDs. Controller code performs projection only. Service obtains persisted IDs from repositories and policy IDs from established backend constants/rules. | PASS |
| Production route composition | `TestNewProductionExposesProductionRoutes` verifies `/api/v1/search/filter-options?mode=substitution` is registered rather than returning 404. | PASS |

## Security assessment

The `golang-security` skill was applied in coding mode.

- Trust boundary: only the `mode` query value is attacker-controlled. It is allow-listed to one exact value before service dispatch; unknown/extra query fields are rejected.
- Persistence: the new read SQL contains no interpolated input and no dynamic identifiers. Existing classification reads remain parameterized by backend constants.
- Authorization: the data is intentionally global/public. The route does not inspect or trust client role/identity headers and exposes no private/user-owned data.
- Error disclosure: typed dependency failures become the repository-wide generic 503 envelope; tests prove sensitive causes/messages are absent.
- DoS/concurrency: endpoint rate limiting is configured; service caching is mutex-protected; callers receive deep-enough slice copies; invalidation uses a generation counter; full race verification passed.
- Secrets/PII: the vocabulary and policy contain neither. No new logs, credentials, cookies, crypto, filesystem access, command execution, or outbound requests were introduced.

No security blocker or Task 241-specific vulnerability was found.

## Commands and results

| Command | Result |
| --- | --- |
| Red regression: `go test -count=1 -v ./internal/repository -run TestAllergenVocabularyMigrationReplayPreservesInactiveDairy` before SQL repair | EXPECTED FAIL: `dairy became active after repeat migrations up`; reproduced the exact review finding. |
| Green regression: the same focused command after SQL repair | PASS through the real `migrations.Run` path; dairy retained non-null `deleted_at` and remained absent from `ListActive`. |
| Focused service: `go test -count=1 -v ./internal/search -run 'TestFilterOption'` | PASS, including live PostgreSQL administration/inactive test. |
| Focused HTTP: `go test -count=1 -v ./internal/httpapi -run 'TestFilterOption'` | PASS. |
| Focused repository: `go test -count=1 -v ./internal/repository -run 'Test(AllergenVocabularyMigrationReplay|PostgresAllergenVocabulary)'` | PASS, including repeat migration-up inactive-state preservation. |
| Focused race: `go test -count=1 -race ./internal/search ./internal/httpapi ./internal/repository -run 'Test(FilterOption|PostgresAllergenVocabulary|AllergenVocabularyMigrationReplay)'` | PASS. |
| Production composition race: `go test -count=1 -race ./internal/app -run TestNewProductionExposesProductionRoutes` | PASS. |
| Aggregate backend coverage | PASS under documented exceptions: backend total 87.6%, search 96.8%, HTTP 87.5%, repository 93.1%, and migrations 100.0%; no Task 241 exception added. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./...` | PASS for every backend package. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./...` | PASS for every backend package. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS, no output. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS: no vulnerabilities in called code; 18 vulnerable required-module versions are present but not called. |
| `python3 scripts/verify-local-stack.py` | PASS: isolated PostgreSQL migration up/down/up plus API/worker health and readiness. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks with ordered dependencies; Task 241 remains PREPARED. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS with the existing explicitly ignored OAuth callback 302-only warning. |
| `git diff --check` | PASS. |
| `python3 scripts/check.py` | PASS, exit 0: traceability/task-list/Go Doc, OpenAPI/security, migration up/down/up including migration 27, local stack/readiness, Phase 02/03 UAT, repository/backend tests, full backend race, frontend verification/build/unit/coverage, 72 + 30 focused Playwright checks, full Playwright 237 passed/3 skipped, and 459 frontend unit tests all succeeded. Existing documented package/frontend coverage exceptions were reported, with no new Task 241 exception needed. |

The aggregate browser logs contained expected mocked/proxy `401`/`ECONNREFUSED` diagnostics while their owning tests passed; they did not affect the exit result or Task 241.

## Final implementation hashes

| Path | SHA-256 |
| --- | --- |
| `backend/internal/app/app.go` | `42dadf16664b29dcfff32cf4b6b1643a2b9e5bdaa0a776b9d63eabd8498b0243` |
| `backend/internal/app/app_test.go` | `6bcc71758113a7426356021b9965ef6cdc646c2b85042e7e39b1b631068faf04` |
| `backend/internal/httpapi/search_validation.go` | `46e58df720077aa284e747366c172fe3a649322871b5fc7c8bec8a876f54fb29` |
| `backend/internal/httpapi/filter_option_controller.go` | `380673327db3acc6043c284109ba66a37b89ef9dd40a0c1dd500ab5092b0e78a` |
| `backend/internal/httpapi/filter_option_controller_test.go` | `8d234aeb377e5cc9a144fcf1ce553bd6c01c806b14edca2078f1647ef525a5a9` |
| `backend/internal/repository/allergen_vocabulary_repository.go` | `cf5e1f7fe7059740fa683ceb8fa150a44d3d48cc1b982f7d005a32a50e896635` |
| `backend/internal/repository/allergen_vocabulary_repository_test.go` | `3617d6359918af5e0a25674ed71b3df425b1be4cb42fb92f952bb1ccafdaaeb1` |
| `backend/internal/repository/sql/allergen_vocabulary_list.sql` | `9fee8de0b360fc6e4906f0a707eeca4472e63102caedb560c80da0c00e100042` |
| `backend/internal/search/filter_options.go` | `6415996d2ab56641416c999bfe93c5df7f21bf5c7c4551b3448e4ad7b452a31f` |
| `backend/internal/search/filter_options_test.go` | `9a80e286cb21251955c607ffe59f8396cc30fe6d156781100755284c3e82c375` |
| `backend/internal/search/filter_options_integration_test.go` | `e843a4a53e6c6d2ba7c60312c834b7ddcbad34f0336adc8682c92e9459e04d7b` |
| `database/migrations/000027_allergen_vocabulary.up.sql` | `c7182d47efeb78a99ac5b7a650665147405dc1dfe79fde12f32ab9607a7ce0bb` |
| `database/migrations/000027_allergen_vocabulary.down.sql` | `987a21d7c0775f0d35df1caaf72817b9b99c9bf7c85a6052dfe224a5c2d61694` |
| `docs/implementation/02_TASK_LIST.md` (unchanged repair control) | `ca6252275e342f390e767783df7b884f88f7ab1b7df169976c218bd55df148a5` |

## Risks and handoff

- No Task 241 acceptance, correctness, race, security, migration, or coverage blocker remains.
- The in-memory cache invalidation seam is correct for the current single-process service composition and is ready for Task 251 to call after committed classification administration. If deployment later runs multiple API instances, Task 251 or infrastructure design must add a cross-instance invalidation signal or remove the process-local cache; that expansion is outside Task 241.
- Migration 27 persists allergen vocabulary but deliberately does not expose allergen administration; Task 251 is scoped only to Food Categories and Culinary Roles.
- Task 253 must publish this route in OpenAPI and regenerate frontend types; Task 257 must remove the frontend hardcoded inventory and consume the generated contract.
- Existing unrelated dirty changes remain untouched. Task 241 status remains `PREPARED`; this repair did not edit the task list.
