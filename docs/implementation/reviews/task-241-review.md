# Review Evidence: Task 241 — Backend-Owned Search Filter Options

```yaml
task_id: 241
component: "Backend-Owned Search Filter Options"
static_aspect: "DESIGN-009 TagManager"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-21T06:21:18Z"
review_agent: "Codex independent review"
evidence_file: "docs/implementation/reviews/task-241-review.md"
baseline_ref: "81ca40ce00cb667ea29243ed2d34068e11229a69 plus preparation manifest docs/implementation/preparations/task-241.md"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "Go, SQL injection prevention, HTTP/API security, error handling, async/concurrency"
repair_context_required: false
```

## 1. Task Source

**Description:** Add the backend-owned filter-option service and public `GET /api/v1/search/filter-options?mode=substitution` contract, sourced from active persisted Food Categories, Culinary Roles, and Allergens plus backend-owned dietary-preset and physical-state policy.

**Depends On:** 34, 117, 138 — all `PASSED` at review time.

**Testing Coverage Exceptions:** The task row declares `None`. Repository-wide Phase 07 coverage exceptions remain documented for pre-existing packages; every changed Task 241 Go function is 100% in the current backend coverage profile.

**Verification Criteria:** Repository/service/HTTP tests must prove deterministic localized-label-ready DTO ordering, supported mode validation, inactive classification exclusion, allergen and dietary-preset policy projection, physical-state options, empty-vocabulary behavior, cache invalidation after administration changes, public-read security, structured dependency failures, and no hardcoded frontend-oriented IDs invented by the route.

## 2. Pre-Review Gates

- [x] Input status is `PREPARED` in the current task list.
- [x] Every dependency is `PASSED`.
- [x] The preparation report claims completion and supplies the baseline manifest.
- [x] A task-specific baseline/diff is available and trustworthy: fixed HEAD is `81ca40ce00cb667ea29243ed2d34068e11229a69`, and the preparation report distinguishes new Task 241 paths from prior dirty-worktree changes.
- [x] `code-review-skill` was invoked exactly once and its relevant Go, SQL, HTTP/API, security, error-handling, and concurrency guides were read.
- [x] The reviewer is independent from implementation and made no repair.
- [x] Review uses current repository state, current hashes, and fresh command output rather than stale preparation logs.
- [x] No production code or task-list status was changed by this review.

```yaml
pre_review_gates_passed: true
blocking_issue: "None; the prior migration-state defect was repaired and independently reverified."
```

## 3. Review Baseline and Change Surface

Baseline/reference method: I used the fixed preparation reference and inspected the preparation report, current task-list row and dependency statuses, current worktree, and all Task 241 paths. New Task 241 files were compared as absent at the preparation baseline; additions to pre-existing `app.go`, `app_test.go`, and `search_validation.go` were isolated using the preparation symbol list and current line-level inspection. Direct callers and policy dependencies were inspected separately.

Commands used to reconstruct the diff:

```bash
git status --short
git diff --check
git diff 81ca40ce00cb667ea29243ed2d34068e11229a69 -- backend/internal/app/app.go backend/internal/app/app_test.go backend/internal/httpapi/search_validation.go
rg -n "241|PREPARED|34|117|138" docs/implementation/02_TASK_LIST.md docs/implementation/preparations/task-241.md
rg -n "NewFilterOptionService|NewFilterOptionController|ValidateFilterOptionQueryParams|ListActive|Invalidate|filter-options" backend database docs
```

Pre-existing dirty-worktree changes and exclusions:

The worktree contains Task 238–240 implementation files, their review evidence, generated API changes, unrelated repository/deletion changes, and pre-existing documentation edits. Those paths were excluded unless they were a direct Task 241 dependency. Task 241-owned paths are the two migration files, one embedded SQL file, the allergen repository and test, three search files, two filter-option HTTP files, `search_validation.go`’s added validator, and the two app composition/test additions. The current review made no production-code or task-list change; it only refreshes this evidence file.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| `database/migrations/000027_allergen_vocabulary.up.sql` | Preparation manifest; new Task 241 migration | HIGH | Table DDL, seed upsert, schema marker |
| `database/migrations/000027_allergen_vocabulary.down.sql` | Preparation manifest; new Task 241 migration | HIGH | Table drop, schema marker deletion |
| `backend/internal/repository/sql/allergen_vocabulary_list.sql` | Preparation manifest; new embedded SQL | HIGH | Active vocabulary SELECT |
| `backend/internal/repository/allergen_vocabulary_repository.go` and `_test.go` | Preparation manifest; new repository boundary | HIGH | Entry, interface, implementation, constructor, read method, tests |
| `backend/internal/search/filter_options.go`, `_test.go`, `_integration_test.go` | Preparation manifest; new service/policy projection | HIGH | DTO types, service/cache, policy/order helpers, tests |
| `backend/internal/httpapi/filter_option_controller.go` and `_test.go` | Preparation manifest; new public route | HIGH | Reader, controller/DTOs, route, handler, tests |
| `backend/internal/httpapi/search_validation.go` | Task 241 addition to pre-existing validator file | HIGH | `ValidateFilterOptionQueryParams` |
| `backend/internal/app/app.go` and `app_test.go` | Task 241 additions isolated from Task 238–240 edits | HIGH | `NewProduction` composition and route assertion |

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Production exposes `GET /api/v1/search/filter-options?mode=substitution`. | App composition and controller route inspection; focused app/controller tests; aggregate API checks. | PASS | `NewProduction` composes the service/controller, `Routes` registers the exact versioned GET path, and `TestNewProductionExposesProductionRoutes` passes. |
| 2 | Categories, culinary roles, and allergens come from persisted active vocabulary. | Repository SQL inspection and PostgreSQL integration tests. | PASS | Classification SQL filters `deleted_at IS NULL`; allergen SQL does likewise; live repository/service tests read persisted rows and IDs/keys. |
| 3 | Inactive entries stay excluded and an empty vocabulary remains valid, including after administrative state changes. | Soft-delete integration tests plus migration replay test. | PASS | Normal soft-delete and empty-list tests pass. The repaired repeat-up test and direct isolated-database reproduction keep `dairy` inactive and absent from `ListActive`. |
| 4 | Dietary presets and physical state project backend policy. | Shared policy inspection, deterministic service test, filter processor cross-check. | PASS | Five preset constants reuse `dietaryPresetExclusionRules`; liquid/solid use repository constants and localization keys; Vegan and allergen exclusion behavior are asserted. |
| 5 | Labels and ordering are deterministic and localization-ready. | Service ordering test, SQL ordering inspection, coverage and focused tests. | PASS | Group order, case-insensitive label order, and ID tie-break are explicit; policy entries expose `LabelKey`; classification entries retain persisted UUID and fallback name. |
| 6 | Only supported mode is accepted and malformed/extra query input is rejected. | Validator/controller tests for missing, catalog, extra, and valid substitution mode. | PASS | All tested invalid paths return structured 400 without service dispatch; substitution dispatches the validated mode. Duplicate-key cardinality is noted as an optional adversarial test gap in Section 7. |
| 7 | Cache reads are isolated and administrative invalidation reloads current vocabulary. | Cache copy/invalidation unit test, PostgreSQL admin-change integration test, mutex/generation inspection, race test. | PASS | Caller mutation cannot alter cache; `Invalidate` clears and increments generation; in-flight loads cannot repopulate a newer generation; focused and race tests pass. Cross-process invalidation is explicitly owned by later Task 251. |
| 8 | Dependency failures are observable internally and safe/structured publicly. | Fake query/scan/iteration failures, service propagation tests, HTTP error-envelope test. | PASS | Repository maps failures to connection kind; service preserves dependency errors; HTTP returns 503 dependency envelope without sensitive dependency text. |
| 9 | Public-read security boundary is intentional and safe. | Route metadata inspection and anonymous request with spoofed role header. | PASS | Route has no auth, optional-auth, or CSRF requirement because it is read-only public vocabulary; spoofed `X-User-Role: admin` does not affect the service-owned response; endpoint rate limit is present. |
| 10 | The route invents no frontend-oriented IDs. | DTO and projection inspection plus exact-ID controller test. | PASS | Categories/roles use persisted UUID strings, allergens use persisted keys, and policy IDs come from existing backend constants; the controller only maps service values. |
| 11 | SQL/resource/cancellation behavior is safe. | Embedded SQL inspection, repository line audit, `go vet`, race, full tests, local migrations. | PASS | SQL is embedded and static, rows close on all paths after query success, scan/iteration errors are mapped, context is passed to `Query`, and no runtime file or interpolation is introduced. |
| 12 | Adversarial regressions are covered sufficiently for acceptance. | Focused, full, race, security, migration, aggregate, and source audits. | PASS | The repaired migration replay regression is covered by a live repository test and direct repeat-up reproduction; focused/full/race/security/migration/aggregate gates pass. Duplicate query cardinality remains an optional non-blocking test gap in Section 7. |

## 5. Changed-Symbol Inventory

Inventory every added or modified executable unit. The 57 entries below are the complete Task 241 implementation/test inventory after reconstruction; direct unchanged dependencies are audited in context but are not falsely counted as changed Task 241 units.

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | `allergen_vocabulary` table DDL | SQL schema | `database/migrations/000027_allergen_vocabulary.up.sql:2` | added | migration runner; repository query | repository integration |
| 2 | Allergen seed `INSERT ... ON CONFLICT` | SQL seed | `database/migrations/000027_allergen_vocabulary.up.sql:14` | added | migration runner | migration replay audit |
| 3 | Schema migration marker insert | SQL marker | `database/migrations/000027_allergen_vocabulary.up.sql:29` | added | migration runner | migration stack |
| 4 | Allergen table rollback | SQL rollback | `database/migrations/000027_allergen_vocabulary.down.sql:2` | added | migration runner | migration stack |
| 5 | Schema marker rollback | SQL rollback | `database/migrations/000027_allergen_vocabulary.down.sql:4` | added | migration runner | migration stack |
| 6 | `allergenVocabularyListSQL` | embedded SQL | `backend/internal/repository/sql/allergen_vocabulary_list.sql:2` | added | `ListActive` | repository integration/unit |
| 7 | `AllergenVocabularyEntry` | behavioral type | `backend/internal/repository/allergen_vocabulary_repository.go:15` | added | service projection | repository/service tests |
| 8 | `AllergenVocabularyRepository` | interface | `backend/internal/repository/allergen_vocabulary_repository.go:23` | added | service constructor | compile/use tests |
| 9 | `PostgresAllergenVocabularyRepository` | repository type | `backend/internal/repository/allergen_vocabulary_repository.go:29` | added | app composition | repository tests |
| 10 | repository compile assertion | contract assertion | `backend/internal/repository/allergen_vocabulary_repository.go:34` | added | compiler | build/tests |
| 11 | `NewPostgresAllergenVocabularyRepository` | constructor | `backend/internal/repository/allergen_vocabulary_repository.go:38` | added | `NewProduction`; tests | repository tests |
| 12 | `ListActive` | repository method | `backend/internal/repository/allergen_vocabulary_repository.go:44` | added | filter-option service | repository integration/error tests |
| 13 | `TestPostgresAllergenVocabularyRepositoryListsOnlyActiveEntries` | integration test | `backend/internal/repository/allergen_vocabulary_repository_test.go:11` | added | test runner | live PostgreSQL |
| 14 | `TestPostgresAllergenVocabularyRepositoryClassifiesFailures` | error test | `backend/internal/repository/allergen_vocabulary_repository_test.go:55` | added | test runner | fake query/scan/iteration |
| 15 | `FilterOptionReference` | behavioral type | `backend/internal/search/filter_options.go:14` | added | preset policy; HTTP DTO | service/controller tests |
| 16 | `FilterOption` | behavioral type | `backend/internal/search/filter_options.go:21` | added | service response; controller | service/controller tests |
| 17 | `FilterOptionsResponse` | behavioral type | `backend/internal/search/filter_options.go:33` | added | reader/controller | service/controller tests |
| 18 | `FilterOptionClassificationRepository` | interface | `backend/internal/search/filter_options.go:40` | added | service | service/integration tests |
| 19 | `FilterOptionAllergenRepository` | interface | `backend/internal/search/filter_options.go:46` | added | service | service/integration tests |
| 20 | `FilterOptionService` | service/cache type | `backend/internal/search/filter_options.go:52` | added | app/controller | service/race tests |
| 21 | `NewFilterOptionService` | constructor | `backend/internal/search/filter_options.go:62` | added | `NewProduction`; tests | service tests |
| 22 | `Options` | service method | `backend/internal/search/filter_options.go:68` | added | controller; tests | service/controller/race |
| 23 | `Invalidate` | cache method | `backend/internal/search/filter_options.go:95` | added | Task 251 seam; tests | cache/integration tests |
| 24 | `cachedOptions` | cache helper | `backend/internal/search/filter_options.go:104` | added | `Options` | cache tests |
| 25 | `load` | service helper | `backend/internal/search/filter_options.go:115` | added | `Options` | dependency tests |
| 26 | `classificationFilterOptions` | projection helper | `backend/internal/search/filter_options.go:142` | added | `load` | deterministic service test |
| 27 | `physicalStateFilterOptions` | policy helper | `backend/internal/search/filter_options.go:152` | added | `load` | deterministic/empty tests |
| 28 | `dietaryPresetFilterOptions` | policy helper | `backend/internal/search/filter_options.go:161` | added | `load` | deterministic service test |
| 29 | `sortFilterOptions` | ordering helper | `backend/internal/search/filter_options.go:187` | added | `load`; tie test | deterministic/tie tests |
| 30 | `cloneFilterOptionsResponse` | copy helper | `backend/internal/search/filter_options.go:209` | added | cache/return paths | mutation-isolation test |
| 31 | `filterOptionClassificationStub` | test double | `backend/internal/search/filter_options_test.go:14` | added | service unit tests | mutex-protected |
| 32 | classification stub `List` | test method | `backend/internal/search/filter_options_test.go:21` | added | service tests | cache/error tests |
| 33 | `filterOptionAllergenStub` | test double | `backend/internal/search/filter_options_test.go:28` | added | service unit tests | mutex-protected |
| 34 | allergen stub `ListActive` | test method | `backend/internal/search/filter_options_test.go:35` | added | service tests | cache/error tests |
| 35 | `TestFilterOptionServiceProjectsDeterministicLocalizedPolicy` | service test | `backend/internal/search/filter_options_test.go:43` | added | test runner | ordering/labels/policy |
| 36 | `TestFilterOptionServiceCachesCopiesAndInvalidatesAfterAdministration` | service test | `backend/internal/search/filter_options_test.go:104` | added | test runner | empty/cache/invalidation |
| 37 | `TestFilterOptionServiceValidatesModeAndReturnsDependencyFailures` | service test | `backend/internal/search/filter_options_test.go:137` | added | test runner | mode/dependency paths |
| 38 | `TestSortFilterOptionsBreaksEqualLabelTiesByPersistedID` | ordering test | `backend/internal/search/filter_options_test.go:163` | added | test runner | tie-break |
| 39 | `findFilterOption` | test helper | `backend/internal/search/filter_options_test.go:174` | added | service tests | lookup assertions |
| 40 | `TestFilterOptionServiceReloadsActivePersistedVocabularyAfterAdministration` | integration test | `backend/internal/search/filter_options_integration_test.go:11` | added | test runner | PostgreSQL/admin invalidation |
| 41 | `hasFilterOption` | integration helper | `backend/internal/search/filter_options_integration_test.go:53` | added | integration test | active/inactive assertions |
| 42 | `FilterOptionReader` | HTTP interface | `backend/internal/httpapi/filter_option_controller.go:12` | added | controller | HTTP tests |
| 43 | `FilterOptionController` | controller type | `backend/internal/httpapi/filter_option_controller.go:18` | added | app/router | HTTP tests |
| 44 | `filterOptionDTO` | public DTO | `backend/internal/httpapi/filter_option_controller.go:24` | added | `Get` JSON | HTTP projection test |
| 45 | `filterOptionReferenceDTO` | public DTO | `backend/internal/httpapi/filter_option_controller.go:36` | added | `Get` JSON | HTTP projection test |
| 46 | controller compile assertion | contract assertion | `backend/internal/httpapi/filter_option_controller.go:42` | added | compiler | build/tests |
| 47 | `NewFilterOptionController` | constructor | `backend/internal/httpapi/filter_option_controller.go:46` | added | app composition; tests | HTTP tests |
| 48 | `Routes` | route declaration | `backend/internal/httpapi/filter_option_controller.go:52` | added | router composition | route metadata test |
| 49 | `Get` | HTTP handler | `backend/internal/httpapi/filter_option_controller.go:58` | added | router | anonymous/error tests |
| 50 | `filterOptionReaderStub` | HTTP test double | `backend/internal/httpapi/filter_option_controller_test.go:17` | added | controller tests | request tracking |
| 51 | reader stub `Options` | HTTP test method | `backend/internal/httpapi/filter_option_controller_test.go:24` | added | controller tests | mode/error tracking |
| 52 | `TestFilterOptionControllerReturnsServiceOwnedOptionsAnonymously` | HTTP test | `backend/internal/httpapi/filter_option_controller_test.go:31` | added | test runner | public/security/ID projection |
| 53 | `TestFilterOptionControllerRejectsInvalidModeAndStructuresDependencyFailure` | HTTP test | `backend/internal/httpapi/filter_option_controller_test.go:74` | added | test runner | malformed/error envelope |
| 54 | `ValidateFilterOptionQueryParams` | validator | `backend/internal/httpapi/search_validation.go:109` | added | `ValidateQuery` middleware | HTTP tests |
| 55 | `NewProduction` filter-option composition | configuration logic | `backend/internal/app/app.go:76,125` | modified | production route graph | app composition test |
| 56 | `TestNewProductionExposesProductionRoutes` filter-option assertion | composition test | `backend/internal/app/app_test.go:67,100` | modified | test runner | route non-404 assertion |
| 57 | `TestAllergenVocabularyMigrationReplayPreservesInactiveDairy` | migration replay integration test | `backend/internal/repository/allergen_vocabulary_repository_test.go:58` | added in repair | test runner | repeat-up state preservation and active-read exclusion | Passes against the real migration runner and isolated PostgreSQL. |

```yaml
inventory_source_count: 57
audited_symbol_count: 57
inventory_complete: true
generated_groupings:
  - "None; no generated artifact is part of the Task 241 implementation surface."
```

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| `allergen_vocabulary` table DDL | Normalized key, nonblank name/label key, soft-delete timestamp, unique label key. | Valid schema and constraint failures are database-observable. | Persistent state; no hidden process state. | Stores vocabulary, not secrets or PII. | Small indexed primary-key table. | Clear schema and traceability comment. | Live migration/repository and repeat-up tests. | PASS |
| Allergen seed upsert | Seeds seven supported policy keys without changing existing soft-delete state. | Conflict path updates canonical name/label metadata and leaves `deleted_at` untouched. | Repeat execution preserves administrative inactive state. | Public readers cannot observe replay-induced resurrection. | Bounded seven-row seed. | Idempotent syntax now has state-preserving conflict semantics. | Direct repeat-up reproduction and live regression test pass. | PASS |
| Schema migration marker insert | Records version 27 without duplicate error. | Repeated marker is safe. | Persistent marker only. | No user data. | Single-row insert. | Idiomatic `ON CONFLICT DO NOTHING`. | Local migration up/down/up passes. | PASS |
| Allergen table rollback | Removes table on down. | `IF EXISTS` is safe for absent table. | Releases persistent schema. | No auth boundary. | Bounded DDL. | Minimal rollback. | Local migration stack passes. | PASS |
| Schema marker rollback | Removes version 27 marker. | Safe repeated delete. | Persistent cleanup. | No sensitive data. | Single-row delete. | Minimal rollback. | Local migration stack passes. | PASS |
| `allergenVocabularyListSQL` | Returns active rows with name/key/label key. | No malformed dynamic input. | Context is supplied by caller; query rows handled by repository. | Static SQL prevents injection. | Explicit deterministic `ORDER BY`. | Embedded SQL follows repository convention. | Repository integration and `git diff --check`. | PASS |
| `AllergenVocabularyEntry` | Carries persisted key, display fallback, localization key. | Zero values are tolerated until database/service validation. | Immutable value projection. | No frontend-generated identity. | Tiny value type. | Narrow and minimal. | Repository/service tests assert nonempty fields. | PASS |
| `AllergenVocabularyRepository` | Exposes only active vocabulary read. | Dependency errors returned. | Context-aware read contract. | No mutation/public authorization implied. | Narrow interface. | Correct dependency inversion. | Compile and service tests. | PASS |
| `PostgresAllergenVocabularyRepository` | Binds SQL executor to repository contract. | Nil dependency is construction misuse; production composition is nonnil. | No internal mutable state. | No input interpolation. | One executor field. | Idiomatic adapter. | Constructor and integration tests. | PASS |
| repository compile assertion | Enforces interface implementation. | Compile failure is immediate. | N/A — compile-time only. | N/A — compile-time only. | N/A — compile-time only. | Useful local contract. | Full build. | PASS |
| `NewPostgresAllergenVocabularyRepository` | Returns configured adapter. | No runtime I/O. | N/A — constructor only. | Does not broaden access. | O(1) allocation. | Minimal constructor. | Repository tests. | PASS |
| `ListActive` | Returns active entries in SQL order and a nonnil empty slice. | Query, scan, and iteration errors map to repository connection errors. | Passes context, defers `Close`, no goroutine or cache. | Static embedded SQL; no user data leakage. | One bounded vocabulary query; slice allocation proportional to rows. | Matches repository error/rows idioms. | Live active/inactive/empty and fake query/scan/iteration tests. | PASS |
| `TestPostgresAllergenVocabularyRepositoryListsOnlyActiveEntries` | Proves active key inventory and empty behavior. | Soft-delete is checked. | Test DB reset/cleanup is existing harness. | No sensitive fixtures. | Seven-row fixture. | Focused integration test. | Ordinary inactive and empty-vocabulary paths pass; repeat-up behavior is covered by the adjacent repair regression. | PASS |
| `TestPostgresAllergenVocabularyRepositoryClassifiesFailures` | Proves all repository dependency failure categories. | Query, scan, and iteration errors covered. | Fake rows close behavior is harness-controlled. | Sensitive text only exists in test error. | No external I/O. | Table-like focused cases. | Covers adversarial dependency failures. | PASS |
| `FilterOptionReference` | Represents backend policy dependency by ID and kind. | Zero value is only meaningful as invalid policy data. | Value copied in slices. | No client-generated ID. | Tiny value. | Narrow policy type. | Vegan exclusion assertions. | PASS |
| `FilterOption` | Carries ID, kind, label, optional localization key, include/exclude policy, references. | Empty persisted names are database-constrained; policy values are explicit. | Nested excludes require deep copying. | DTO source remains service-owned. | Small bounded object. | Clear backend contract. | Deterministic/localization/policy tests. | PASS |
| `FilterOptionsResponse` | Associates options with validated mode. | Zero response returned on errors. | Returned by value with cloned nested slice. | Public response contains no auth data. | Bounded vocabulary output. | Minimal envelope. | Service/controller tests. | PASS |
| `FilterOptionClassificationRepository` | Reads a requested classification kind. | Errors propagate. | Context-aware. | Read-only boundary. | Narrow interface. | Good dependency inversion. | Stubs and integration. | PASS |
| `FilterOptionAllergenRepository` | Reads active persisted allergens. | Errors propagate. | Context-aware. | Read-only boundary. | Narrow interface. | Good dependency inversion. | Stubs and integration. | PASS |
| `FilterOptionService` | Owns policy projection and cache. | Handles missing cache, dependency failure, invalidation. | RWMutex, generation guard, cloned cache; process-local by design, cross-process invalidation belongs to Task 251. | No auth logic because route is public read. | One cached aggregate; no unbounded retry. | Centralizes backend policy. | Unit, integration, race coverage; no dedicated high-contention in-flight test. | PASS |
| `NewFilterOptionService` | Wires the two read boundaries. | Nil dependencies are construction misuse, not production path. | No I/O. | No privilege elevation. | O(1). | Minimal constructor. | App/unit composition. | PASS |
| `Options` | Accepts only substitution and returns isolated deterministic response. | Rejects unsupported mode before dependency calls; returns load errors; cache publication guarded by generation. | Stale in-flight load may return to its caller after invalidation, but cannot repopulate the newer cache; mutex/race checks pass. | No user-controlled SQL or identity. | Cache avoids repeat DB reads; clones nested slices. | Clear sequence and error propagation. | Mode/error/cache tests and race; duplicate query behavior is HTTP-layer gap only. | PASS |
| `Invalidate` | Discards cache and advances generation. | Safe repeated calls. | Mutex-protected; prevents old load publication. | Admin caller must be supplied by later Task 251. | O(1). | Small explicit seam. | Unit and PostgreSQL admin-change integration. | PASS |
| `cachedOptions` | Returns a deep isolated copy or cache miss. | Nil cache handled. | RLock held through clone. | No data crossing beyond intended response. | Copy proportional to options. | Idiomatic read helper. | Caller mutation test. | PASS |
| `load` | Reads category, role, allergen sources and appends physical/preset policy. | Stops at first dependency failure; returns no partial response. | Context passed to each dependency; sequential ordering is deterministic. | No input interpolation. | Three reads on cold load, bounded aggregate. | Straightforward orchestration. | Dependency order and empty tests. | PASS |
| `classificationFilterOptions` | Uses persisted UUID string and persisted name. | Empty input gives empty slice. | New slice; no alias to repository input. | No invented frontend ID. | O(n) allocation. | Simple projection. | Deterministic test. | PASS |
| `physicalStateFilterOptions` | Exposes liquid and solid backend constants with label keys and both operations allowed. | Fixed complete policy set. | New slice each load. | Existing backend IDs only. | Constant-size allocation. | Explicit policy is easy to audit. | Deterministic/empty tests. | PASS |
| `dietaryPresetFilterOptions` | Exposes five existing presets and shared exclusion rules as exclude-only. | Missing rule would produce empty excludes; current map is complete. | New nested slices per load. | IDs and rule kinds are backend constants. | Constant-size allocation. | Reuses filter processor policy rather than duplicating semantics. | Vegan and option policy assertions. | PASS |
| `sortFilterOptions` | Orders physical, category, role, allergen, preset; lower-label then ID tie-break. | Unknown kinds sort after known groups by zero-map behavior; only known kinds are produced. | In-place sort of fresh aggregate. | No security-sensitive ordering input. | O(n log n). | Stable deterministic comparator. | Unsorted fixture and equal-label tie test. | PASS |
| `cloneFilterOptionsResponse` | Copies options and nested excludes. | Empty/nil nested excludes remain safe for caller use. | Prevents cache aliasing; strings immutable. | No accidental state exposure. | O(n plus excludes). | Focused helper. | Caller mutation test and coverage. | PASS |
| `filterOptionClassificationStub` | Thread-safe mutable test source. | Missing map entries yield empty result. | Mutex protects entries/errors/calls. | Test-only data. | Copy-on-read. | Appropriate fake. | Service tests. | PASS |
| classification stub `List` | Returns copy and configured error. | Error path and call count visible. | Mutex released with defer. | N/A — test-only. | O(n) copy. | Simple fake. | Cache/error tests. | PASS |
| `filterOptionAllergenStub` | Thread-safe active-allergen fake. | Empty and error states. | Mutex protects mutable fixture. | Test-only. | Copy-on-read. | Appropriate fake. | Service tests. | PASS |
| allergen stub `ListActive` | Returns copy and configured error. | Error and call count visible. | Mutex-protected. | N/A — test-only. | O(n) copy. | Simple fake. | Cache/error tests. | PASS |
| `TestFilterOptionServiceProjectsDeterministicLocalizedPolicy` | Asserts complete policy projection and order. | Unsorted persisted data, labels, keys, Vegan, allergen semantics. | No shared mutable production state. | Exact persisted IDs and policy IDs asserted. | Small deterministic fixture. | Strong table-like assertions. | Covers main adversarial projections. | PASS |
| `TestFilterOptionServiceCachesCopiesAndInvalidatesAfterAdministration` | Asserts empty vocabulary, cache hit, copy isolation, and reload. | Mutated caller and changed admin fixture. | Call counts prove cache and invalidation; no direct concurrent interleaving. | No auth concern. | Proves cold versus cached reads. | Good focused cache test. | Migration replay preservation is covered by the repository regression. | PASS |
| `TestFilterOptionServiceValidatesModeAndReturnsDependencyFailures` | Rejects catalog before reads and preserves typed dependency errors. | Category, role, allergen failure paths. | Stops subsequent reads after failure. | Sensitive error only in fake; HTTP sanitization separate. | No external I/O. | Clear failure sequencing. | Adversarial dependency coverage. | PASS |
| `TestSortFilterOptionsBreaksEqualLabelTiesByPersistedID` | Proves deterministic tie-break. | Case variation in equal labels. | In-place fixture only. | IDs are explicit. | Constant-size. | Focused comparator test. | Covers tie edge. | PASS |
| `findFilterOption` | Fails test when expected option absent. | Returns zero only after fatal test failure. | N/A — test helper. | N/A — test helper. | Linear small fixture. | Appropriate helper. | Used by policy tests. | PASS |
| `TestFilterOptionServiceReloadsActivePersistedVocabularyAfterAdministration` | Proves inactive exclusion, stale cache, invalidation, and active reload. | Creates/soft-deletes one category and creates a fresh one. | Uses live DB and explicit invalidation. | No cross-user data. | Small fixture. | Good integration seam. | Migration repeat-up preservation is covered by the repository regression. | PASS |
| `hasFilterOption` | Tests presence by kind and ID. | Returns false for absent. | N/A — test helper. | Exact IDs only. | Linear small slice. | Simple helper. | Integration assertions. | PASS |
| `FilterOptionReader` | Controller depends only on service read contract. | Service errors return to router. | Context propagated. | Does not imply auth. | Narrow interface. | Good HTTP seam. | HTTP stub tests. | PASS |
| `FilterOptionController` | Owns public DTO mapping and route registration. | Invalid input middleware; service errors pass to error mapper. | No mutable state. | Public read is intentional; no CSRF/auth on GET. | One response allocation. | Minimal controller. | Route/controller tests. | PASS |
| `filterOptionDTO` | JSON-safe public projection with optional label key. | Excludes always serialized as a slice. | Value copied from service. | No secret/user fields. | Bounded DTO. | Private type avoids accidental external API coupling. | Exact response test. | PASS |
| `filterOptionReferenceDTO` | JSON-safe policy dependency reference. | Maps all references. | Value copied. | No hidden label/identity fabrication. | Tiny DTO. | Minimal. | Exact excludes test. | PASS |
| controller compile assertion | Enforces controller contract. | Compile-time failure. | N/A — compile-time only. | N/A — compile-time only. | N/A — compile-time only. | Idiomatic. | Full build. | PASS |
| `NewFilterOptionController` | Stores reader for public handler. | Nil reader is construction misuse. | No I/O. | Does not grant auth. | O(1). | Minimal. | HTTP tests. | PASS |
| `Routes` | Registers exact GET path, validator, rate limit, and no auth/CSRF flags. | Middleware rejects invalid query before handler. | Route metadata is immutable. | Public boundary is explicit and read-only. | Endpoint rate limit bounds abuse. | Matches router conventions. | Route metadata and anonymous request. | PASS |
| `Get` | Maps service-owned response without inventing IDs and emits envelope. | Dependency/validation errors return; nested excludes map. | Uses request context; no retained state. | Does not trust spoofed role header; safe error mapping is upstream. | O(options plus excludes). | Direct field mapping. | Public/exact-ID/error tests. | PASS |
| `filterOptionReaderStub` | Captures mode/calls and returns fixture/error. | Error fixture supports 503 path. | Test-only mutable state. | N/A — test-only. | Constant-size. | Simple fake. | Controller tests. | PASS |
| reader stub `Options` | Records service dispatch. | Returns configured response/error. | N/A — test-only. | N/A — test-only. | O(1). | Simple fake. | Anonymous/malformed tests. | PASS |
| `TestFilterOptionControllerReturnsServiceOwnedOptionsAnonymously` | Proves public route and exact service-owned IDs/labels/excludes. | Spoofed admin header and JSON decode. | One request. | Auth flags and role spoofing challenged. | Small fixture. | Strong boundary test. | Covers no invented IDs. | PASS |
| `TestFilterOptionControllerRejectsInvalidModeAndStructuresDependencyFailure` | Proves 400 validation and safe 503 envelope. | Missing, unsupported, extra query and sensitive dependency error. | Invalid requests do not call service. | No sensitive error text crosses HTTP. | Small table. | Good HTTP failure test. | Duplicate-key case not included. | PASS |
| `ValidateFilterOptionQueryParams` | Accepts exactly one `mode=substitution` map entry. | Missing/catalog/extra values rejected. | No I/O. | Rejects unsupported public input before service. | O(1). | Small validator. | Focused HTTP tests; duplicate-key cardinality is an optional gap because router collapses query args to a map. | PASS |
| `NewProduction` filter-option composition | Wires real Postgres classification/allergen readers and public controller. | Constructor errors remain app errors; route graph includes controller. | Service cache is process-local; no admin invalidator wiring is claimed here. | Public-read boundary intentionally not auth-gated. | Shared DB pool; cached service. | Correct composition for Task 241; Task 251 owns admin invalidation caller. | App route test and aggregate. | PASS |
| `TestNewProductionExposesProductionRoutes` filter-option assertion | Prevents route omission in production composition. | Fails on 404; does not require live DB response. | Test app construction. | Route existence only; controller security tested separately. | Bounded route table. | Focused regression assertion. | Passes. | PASS |
| `TestAllergenVocabularyMigrationReplayPreservesInactiveDairy` | Proves a soft-deleted seeded allergen remains inactive after the real runner repeats all up migrations. | Directly checks `deleted_at IS NOT NULL` and confirms `ListActive` omits `dairy`. | Uses isolated PostgreSQL reset, migration directory resolution, and cleanup; no retained process state. | Prevents public policy resurrection through migration replay. | Seven-row migration and one repository read. | Focused regression at the actual runner boundary. | New repair test passes in focused, full, and race runs; direct reproduction reports `active_after_repeat_up=f`. | PASS |

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| nit | `backend/internal/httpapi/search_validation.go:109-113` | `ValidateFilterOptionQueryParams` | `ValidateQuery` collapses duplicate query keys into a map, while `Get` reads the first raw `mode`; duplicate cardinality is not explicitly rejected or tested. Conflicting duplicates still end in a safe 400 through service validation, and identical duplicates are harmless, so this is not an acceptance blocker. | Adversarial source audit of router `VisitAll` map construction and Fiber `Query` first-value lookup; existing tests cover missing, unsupported, and extra keys but not duplicate keys. | Optional follow-up: add table-driven duplicate/conflicting-mode tests; if strict one-key semantics are required, preserve query cardinality in validation and pass the validated canonical mode to the handler. |

```yaml
blocking_findings: 0
important_findings: 0
optional_findings: 1
```

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| `git diff --check` | repository root | 0 | PASS | clean whitespace check |
| `gofmt -l` on all Task 241 Go files | repository root | 0 | PASS | no files listed |
| `go test -count=1 -v ./internal/search -run 'TestFilterOption'` | `backend` | 0 | PASS | focused service and PostgreSQL integration tests |
| `go test -count=1 -v ./internal/httpapi -run 'TestFilterOption'` | `backend` | 0 | PASS | focused route/HTTP tests |
| `go test -count=1 -v ./internal/repository -run 'Test(AllergenVocabulary|PostgresAllergenVocabulary)'` | `backend` | 0 | PASS | focused repository tests, including repeat-up regression |
| `go test -count=1 -race ./internal/search ./internal/httpapi ./internal/repository -run 'Test(FilterOption|PostgresAllergenVocabulary)'` | `backend` | 0 | PASS | focused race coverage |
| `go test -count=1 -race ./internal/app -run TestNewProductionExposesProductionRoutes` | `backend` | 0 | PASS | production route composition race check |
| `go test -count=1 ./...` | `backend` | 0 | PASS | all backend packages |
| `go test -count=1 -race ./...` | `backend` | 0 | PASS | all backend packages under race detector |
| `go test ./internal/... -coverprofile=coverage.out` and `go tool cover -func=coverage.out` | `backend` | 0 | PASS | backend total 87.6%; changed Task 241 functions 100% |
| `go vet ./...` | `backend` | 0 | PASS | no diagnostics |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `backend` | 0 | PASS | 0 vulnerabilities in called code; 18 required-module-only findings reported by tool |
| `python3 scripts/verify-local-stack.py` | repository root | 0 | PASS | PostgreSQL/Redis, migration up/down/up, API health/readiness |
| `npx --no-install redocly lint api/openapi.yaml` | repository root | 0 | PASS | valid with one pre-existing ignored 302-only warning |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS | 263 sequential tasks and dependencies |
| `python3 scripts/validate-traceability.py` | repository root | 0 | PASS | traceability validation |
| `python3 scripts/check.py` | repository root | 0 | PASS | aggregate contract, backend, frontend, browser, coverage, race, security, and stack gates; documented coverage exceptions remain |
| `UPDATE ... deleted_at=now(); go run ./cmd/migrate up; SELECT deleted_at IS NULL` | isolated `mealswapp_test` | 0 | PASS | direct repeat-up reproduction returned `active_after_repeat_up=f`; cleanup restored the fixture |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-241-review.md` | repository root | 0 | PASS | current evidence structure validated after refresh |

## 9. Files Inspected and Staleness Fingerprints

Hashes below are SHA-256 of current contents after implementation inspection and before evidence validation. Direct policy, repository, router, and migration-runner dependencies are included because they determine the Task 241 behavior.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| `database/migrations/000027_allergen_vocabulary.up.sql` | Task migration schema/seed | repaired repeat-up state preservation | SHA-256 | `c7182d47efeb78a99ac5b7a650665147405dc1dfe79fde12f32ab9607a7ce0bb` |
| `database/migrations/000027_allergen_vocabulary.down.sql` | Task migration rollback | none | SHA-256 | `987a21d7c0775f0d35df1caaf72817b9b99c9bf7c85a6052dfe224a5c2d61694` |
| `backend/internal/repository/sql/allergen_vocabulary_list.sql` | Active allergen SQL source | none | SHA-256 | `9fee8de0b360fc6e4906f0a707eeca4472e63102caedb560c80da0c00e100042` |
| `backend/internal/repository/allergen_vocabulary_repository.go` | Allergen repository implementation | none | SHA-256 | `cf5e1f7fe7059740fa683ceb8fa150a44d3d48cc1b982f7d005a32a50e896635` |
| `backend/internal/repository/allergen_vocabulary_repository_test.go` | Repository tests | repeat-up regression added and passing | SHA-256 | `3617d6359918af5e0a25674ed71b3df425b1be4cb42fb92f952bb1ccafdaaeb1` |
| `backend/internal/search/filter_options.go` | Service/cache/policy implementation | none | SHA-256 | `6415996d2ab56641416c999bfe93c5df7f21bf5c7c4551b3448e4ad7b452a31f` |
| `backend/internal/search/filter_options_test.go` | Service tests | none beyond migration gap | SHA-256 | `9a80e286cb21251955c607ffe59f8396cc30fe6d156781100755284c3e82c375` |
| `backend/internal/search/filter_options_integration_test.go` | Live classification/invalidation test | none beyond migration gap | SHA-256 | `e843a4a53e6c6d2ba7c60312c834b7ddcbad34f0336adc8682c92e9459e04d7b` |
| `backend/internal/httpapi/filter_option_controller.go` | Public route and DTO mapping | none | SHA-256 | `380673327db3acc6043c284109ba66a37b89ef9dd40a0c1dd500ab5092b0e78a` |
| `backend/internal/httpapi/filter_option_controller_test.go` | Public/error/security tests | duplicate-key gap | SHA-256 | `8d234aeb377e5cc9a144fcf1ce553bd6c01c806b14edca2078f1647ef525a5a9` |
| `backend/internal/httpapi/search_validation.go` | Added mode validator and router caller context | duplicate-key gap | SHA-256 | `46e58df720077aa284e747366c172fe3a649322871b5fc7c8bec8a876f54fb29` |
| `backend/internal/app/app.go` | Production service/controller composition | none | SHA-256 | `42dadf16664b29dcfff32cf4b6b1643a2b9e5bdaa0a776b9d63eabd8498b0243` |
| `backend/internal/app/app_test.go` | Production route regression assertion | none | SHA-256 | `6bcc71758113a7426356021b9965ef6cdc646c2b85042e7e39b1b631068faf04` |
| `backend/internal/search/contracts.go` | Search mode/kind constants consumed by task | none | SHA-256 | `f06edbd63ab7aa3d9780a48415880916dd6f40fad571b60e6ebfe214cd0ea2ec` |
| `backend/internal/search/filter_processor.go` | Shared dietary/allergen/physical policy | none | SHA-256 | `3c9236e42342006d238f3c0555c3242e7df0dcbec1ec33200a716dcc8dad3683` |
| `backend/internal/repository/classification_repository.go` | Active classification reads and ordering | none | SHA-256 | `2d5073f1fe54d0e7a6a291bdbf9ee6e38531fe96ece7575d7fcf1c56325c0368` |
| `backend/internal/repository/types.go` | Classification and physical-state contracts | none | SHA-256 | `a959e11147b0ff709b885ae0c58870c3628ad545cfd99f8e5c53efb5f1116cde` |
| `backend/internal/repository/errors.go` | Typed dependency/validation errors | none | SHA-256 | `4423cf862534cd5612800032309386e175d889d9f1428fb2069c72b4bd4c9a09` |
| `backend/internal/repository/postgres.go` | PostgreSQL error mapping | none | SHA-256 | `2dc903f7954876014f6f94b2ae399680c950d47acda634ec970af6329d507046` |
| `backend/internal/httpapi/router.go` | Query middleware, public route, error mapping | duplicate-key audit context | SHA-256 | `cd4c888689151d66051561c3aaedd6ac379149df950a09eec30d0f7591566125` |
| `backend/internal/migrations/migrations.go` | Actual repeat-up execution semantics | confirms regression boundary and repaired behavior | SHA-256 | `001c8a13fe04a249acfee803bf5e94cde2d35ad87af780dba30d67baeda178d6` |
| `backend/internal/testdatabase/testdatabase.go` | Isolated PostgreSQL reset/migration test harness used by the replay regression | direct test dependency audited | SHA-256 | `ec7bb939b7862e9db39274f8728cf9bfb27f19cca975d5803253f8329524414f` |
| `docs/implementation/preparations/task-241.md` | Preparation baseline/control evidence | current hash recorded | SHA-256 | `b0f7ce961ff0d5f44b592b52f7d2d5f537e10dea802944e79ca89e3128c5a031` |
| `docs/implementation/02_TASK_LIST.md` | Current task/dependency status control | current status is PREPARED | SHA-256 | `ca6252275e342f390e767783df7b884f88f7ab1b7df169976c218bd55df148a5` |

```yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The prior task-241 review was REJECTED because migration 27 reset deleted_at during repeat up; its migration and repository-test hashes are stale. It was checked for the original finding, not treated as current evidence. The current task-list row remains PREPARED and was not edited."
```

## 10. Coverage and Exceptions

- [x] Required backend coverage command ran.
- [x] Report path and observed thresholds are recorded.
- [x] Changed Task 241 functions were checked individually in `backend/coverage.out` and are 100% covered.
- [x] Pre-existing package exceptions were inspected and remain documented; they are not used to excuse the migration defect.

```yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "backend/coverage.out from the current aggregate/focused run"
observed_line_coverage: "Backend total 87.6%; search 96.8%; repository 93.1%; changed Task 241 functions 100%; frontend aggregate 95.13% with documented Phase 07 exceptions."
coverage_passed: true
```

Coverage finding: Coverage is not a blocker. The repair adds a live repeat-up regression test; the current backend coverage profile reports the changed Task 241 production functions at 100% and the focused/full/race suites pass.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced by Task 241; process-local invalidation is an explicit seam and Task 251 owns the admin caller/cross-process invalidation work.
- [x] No source-of-truth design contradiction was found in the service policy, public-read decision, or ID ownership; the migration replay behavior is a persistence regression discovered against the actual migration runner.
- [x] No generated/cache/build/temporary artifact was unintentionally added to the reviewed task surface; aggregate-generated ignored artifacts were not included in the evidence inventory.
- [x] Public API additions are necessary and used by production composition.
- [x] Duplicate helpers and obsolete aliases were searched for; no competing Task 241 implementation was found.
- [x] Error, cleanup, timeout/context, concurrency, SQL, public-auth, localization, ID, malformed-input, and dependency-failure paths were challenged. The duplicate-query cardinality test gap is optional; migration replay is important.

Findings: The prior migration replay defect is repaired: the seed conflict branch no longer writes `deleted_at`, and both the new live regression and direct repeat-up reproduction preserve inactive `dairy`. The active read path, cache invalidation, public boundary, structured dependency failures, localization, ordering, and ID ownership remain correct. The only finding is an optional duplicate-query-cardinality test gap.

## 12. Decision

A task may be `PASSED` only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains. This re-review meets those conditions. The prior migration replay finding is closed by the repaired conflict semantics and regression evidence.

```yaml
decision: "PASSED"
reason: "The repaired migration preserves soft-deleted allergen state across the actual repeat-up runner, all original criteria and symbol audits pass, current hashes and command evidence are complete, and no blocking or important finding remains."
failed_criteria:
  - ""
failed_or_unaudited_symbols:
  - ""
recommended_next_action: "None; task is ready for the orchestrator's status decision."
```

## 13. Repair Context

Repair context was satisfied and is now closed. The prior review's repeat-up migration finding was rechecked against current migration-runner semantics; current migration 27, the new repository regression test, active-vocabulary SQL, service projection/cache/invalidation, production composition, and all affected callers/dependencies were re-read and rehashed. No further repair is required, and this review did not edit production code or task-list status.
