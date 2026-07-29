# Review Evidence: Task 277 — DESIGN-009: ItemCurator

```yaml
task_id: 277
component: "ItemCurator"
static_aspect: "Phase 08.02 JSON Global Catalog Import Operator"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-27T22:26:38Z"
review_agent: "Codex independent re-review"
evidence_file: "docs/implementation/evidence/task-277-review.md"
baseline_ref: "f9a646ffede5c8a63ad787a07eaba7efc0433677"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "code-review-skill Python, Go, TypeScript, and Svelte guides"
repair_context_required: true
```

## 1. Task Source

**Description:** Add the explicit `scripts/import-global-catalog.py` operator for versioned, metric-only, at-most-500-item global catalog imports through the existing authenticated manual-item route. Validate and preflight the complete document before mutation; use interactive credentials, in-memory cookies, fresh CSRF, pacing, bounded retry handling, stable idempotency keys, resumable reporting, and PII-safe output. Extend the manual-item request, response, and persistence contract with active canonical `allergenKeys` atomically. Do not add a bulk endpoint, direct PostgreSQL importer, imperial persistence fields, position-derived keys, or credential/PII output.

**Depends On:** 250, 251, 253, 276

**Testing Coverage Exceptions:** None in the authoritative task row. Existing Phase 08 coverage exceptions are separately machine-checked in `docs/implementation/04_OPEN.md`.

**Verification Criteria:** Version/schema and duplicate-key rejection; malformed/type/overflow-safe validation; unique operator keys and safe fingerprints; metric solid/liquid/density rules; canonical micronutrients/allergens; classification name and optional UUID resolution; complete no-mutation preflight; authenticated dry run; interactive password, memory-only cookies, fresh CSRF; one paced mutation per item; bounded `Retry-After`; same-key/body ambiguity retries; permanent/auth failure policy; deterministic reports and resumable replay; no bulk route/direct database/imperial/position-derived keys/PII; end-to-end active allergen persistence, audit, cache invalidation, idempotency, and contract-conforming responses; moderate-volume documentation; OpenAPI/generated contracts; focused Python, backend, security, traceability, and diff checks.

## 2. Pre-Review Gates

- [x] Input status is `PREPARED`.
- [x] Every dependency is `PREPARED` or `PASSED`; dependencies 250, 251, 253, and 276 are recorded as passed.
- [x] The preparation report claims completion.
- [x] A task-specific baseline and current diff are available and trustworthy.
- [x] `code-review-skill` was invoked exactly once and its relevant Python/Go/TypeScript/Svelte guides were read.
- [x] The reviewer is independent from implementation and repair.
- [x] Review uses current repository state rather than stale logs.
- [x] No production-code or task-list changes were made by this reviewer; only this evidence file is being refreshed.

```yaml
pre_review_gates_passed: true
blocking_issue: "none"
```

## 3. Review Baseline and Change Surface

Baseline/reference method: The fixed baseline is commit `f9a646ffede5c8a63ad787a07eaba7efc0433677`. I independently captured the current dirty status, reconstructed `git diff HEAD`, compared the task-owned paths with `task-277-preparation.md`, and inspected callers/dependencies at the current worktree. The preparation report is not treated as proof of the repaired state.

Commands used to reconstruct the diff:

```bash
git log -1 --oneline
git status --short
git diff --name-status HEAD
git ls-files --others --exclude-standard
git diff --stat HEAD
git diff --check
git diff HEAD -- api/openapi.yaml backend/internal/httpapi/manual_item_controller.go backend/internal/itemcurator/service.go backend/internal/repository/manual_food_repository.go backend/internal/repository/types.go frontend/src/lib/admin-workflows.ts frontend/src/lib/api/admin-client.ts frontend/src/lib/api/generated.ts scripts/check.py scripts/generate-api-types.py
git diff HEAD -- frontend/src/lib/components/AdminDataManagement.svelte frontend/src/lib/components/AdminDataManagement.test.ts
```

The starting tree was already dirty with Task 276 and earlier Phase 08 changes: 52 tracked changed paths plus 28 untracked paths. The current Task 277 surface is the preparation report’s operator, backend contract/persistence, generated/API/frontend, seed compatibility, browser fixture, design, operations, and gate paths. Shared overlap is attributed symbol-by-symbol: `repository/types.go` contains Task 276 audit/bootstrap additions and the Task 277 `FoodItemEntity.AllergenKeys` field; `seed/development.sql` and `seed_test.go` are shared Phase 08 seed boundaries repaired for the Task 277 response contract; `docs/design/DESIGN-009.md` and the task list are shared planning surfaces. Task 276 auth/bootstrap files and unrelated hunks were inspected for boundary compatibility but were not counted as Task 277 implementation symbols. The task-list row remains externally `PREPARED`; this review did not edit it.

| Changed file or surface | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| `scripts/import-global-catalog.py` | Task 277 new operator | HIGH | loader, validation, client, retry, preflight, report, CLI |
| `scripts/test_import_global_catalog.py` | Task 277 new tests | HIGH | validation, preflight, retry, HTTP fake-server, reporting units |
| `backend/internal/httpapi/manual_item_controller.go` | Task 277 shared manual-item boundary | HIGH | request decoder, duplicate key guard, response projection |
| `backend/internal/itemcurator/service.go` | Task 277 global request/entity contract | HIGH | request validation/normalization and response mapping |
| `backend/internal/repository/manual_food_repository.go` | Task 277 persistence boundary | HIGH | active allergen validation, replacement, hydration |
| `backend/internal/repository/sql/food_allergens_*.sql` | Task 277 new SQL statements | HIGH | validate, clear, replace, list |
| `backend/internal/repository/types.go` | Task 277 field plus Task 276 shared changes | HIGH | `FoodItemEntity.AllergenKeys`; other changed symbols excluded as Task 276 |
| `backend/internal/httpapi/manual_item_controller_test.go` | Task 277 HTTP regression assertions | HIGH | valid CRUD/replay, invalid input, invalidation tests |
| `backend/internal/repository/manual_food_repository_test.go` | Task 277 persistence regression assertions | HIGH | CRUD/allergen/rollback/replay test |
| `backend/internal/app/task271_backend_regression_integration_test.go` | Task 277 propagated fixture boundary | HIGH | global/private request helpers and production gate |
| `api/openapi.yaml`, `scripts/generate-api-types.py`, `frontend/src/lib/api/generated.ts` | Task 277 source/generated contract | HIGH | `AdminItemRequest`, `AdminItem`, strict allergen rules |
| `frontend/src/lib/admin-workflows.ts`, `frontend/src/lib/api/admin-client.ts` | Task 277 frontend request/response boundary | HIGH | form key validation, request projection, response decoding |
| `frontend/src/lib/components/AdminDataManagement.svelte` | Repair-cycle Task 277 caller change | HIGH | form initialization, hydration, allergen select, submission caller |
| `frontend/src/lib/components/AdminDataManagement.test.ts` | Repair-cycle regression test | HIGH | allergen caller contract assertion |
| `frontend/src/lib/admin-workflows.test.ts`, `frontend/src/lib/api/admin-client.test.ts`, `scripts/test_generate_api_types.py` | Task 277 propagated tests | HIGH | form, decoder, generator drift units |
| `scripts/check.py`, `docs/design/DESIGN-009.md`, `docs/operations/global-catalog-import.md` | Task 277 gate/design/operator documentation | HIGH | focused operator gate, design clauses, moderate-volume runbook |
| `backend/internal/seed/development.sql`, `backend/internal/seed/seed_test.go` | second-cycle classification UUID/reference migration repair | HIGH | canonical seed map, reference migration, idempotency and UUID contract tests |
| `frontend/tests/admin-data-management.spec.ts` | second-cycle browser fixture repair | HIGH | allergen-bearing manual-item response fixtures and request assertions |
| `scripts/test_check_coverage.py`, `docs/implementation/04_OPEN.md` | second-cycle coverage metadata/checker repair | HIGH | exact Phase 08 measurements, Markdown parser alignment, regression checks |

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Closed `mealswapp.global-catalog.v1` root and 1–500 item limit | `load_document`, focused validation tests, quick gate | PASS | Closed root and limit are enforced; 17 focused Python tests pass. |
| 2 | Duplicate JSON keys are rejected at every depth | `object_pairs_hook`, duplicate test, CLI probe | PASS | Duplicate root probe exits 2 with only `catalog contains duplicate JSON keys`; no traceback. |
| 3 | Malformed JSON/types/nulls/overflow are sanitized before mutation | validation and CLI probes | PASS | `load_document` converts parser `ValueError`/overflow failures to `CatalogError`; every accepted optional field has explicit type, bound, and URI/provenance validation, and the focused no-login/no-client probes cover malformed values plus a 10,000-digit integer. |
| 4 | Stable unique 8–255 printable operator keys; never position-derived | key validation, duplicate-key tests, report test | PASS | Duplicate/short keys fail; reports use stable SHA-256 fingerprints; no positional key generation exists. |
| 5 | Metric-only nutrition/serving/density and solid/liquid rules | validation tests, backend service validation | PASS | Solid/liquid, density provenance, bounds, and unknown-field rejection are implemented and tested. |
| 6 | Canonical micronutrient keys and active canonical allergens | validation, preflight, SQL, integration tests | PASS | Micronutrient allowlist is local; allergens are resolved remotely and transactionally revalidated as active. |
| 7 | Portable classification names resolve exactly one active item | `preflight` and ambiguity/unknown tests | PASS | Name normalization is deterministic; unknown and ambiguous names fail before creates. |
| 8 | Optional classification UUID alternative accepts canonical UUIDs | UUID validation/preflight tests | PASS | UUID alternative is exclusive with names, canonicalized, sorted, and active-response checked. |
| 9 | Uppercase classification UUIDs canonicalize | focused uppercase-UUID test and direct probe | PASS | `AAAAAAAA-...` resolves to lowercase canonical output. |
| 10 | Complete preflight precedes first mutation; dry run has zero creates | fake-server tests | PASS | Login, CSRF, both classification lists, and allergen options are read before create; dry-run create count is zero. |
| 11 | Interactive password, memory-only cookies, fresh CSRF | fake HTTP server and PII assertions | PASS | No password CLI argument; cookie jar is in memory; CSRF is fetched after login; fake server confirms cookie/CSRF behavior. |
| 12 | One create per item, deterministic ordering, and at least 2.1-second pacing | fake clock/server tests and code audit | PASS | Every attempt passes through `pace`; identical request sequence and stable report order are tested. |
| 13 | Bounded `Retry-After` is honored on retryable 5xx | 5xx `Retry-After` test and direct probe | PASS | 503 with `Retry-After: 4` sleeps 4 seconds before same-key retry; 31 is ignored and pacing remains bounded. |
| 14 | Network/5xx ambiguity retries identical key and body | retry tests | PASS | Same key/body set remains one element across 500/429/201 and 503/201; attempts are capped. |
| 15 | 400/409 are permanent; 401/403 abort; partial failure is nonzero | retry/partial tests | PASS | Focused tests prove status policy and exit 1 for partial item failure. |
| 16 | Reports are sanitized, stable, and resumable | report/PII/fake-server tests | PASS | Only index, fingerprint, outcome, and status are emitted; credentials, email, cookies, CSRF, names, keys, and bodies are absent. |
| 17 | Existing authenticated manual route only; no bulk endpoint/direct DB/imperial/position keys | route/source audit, OpenAPI diff | PASS | No public bulk route or PostgreSQL client was added; importer calls only `POST /api/v1/admin/items`. |
| 18 | `allergenKeys` is required, canonical, unique, and carried through frontend request parsing | OpenAPI, Go decoder/service, frontend workflow/component tests | PASS | Form, parser, generated request, HTTP decoder, service normalization, and persistence all carry keys; repaired component initializes/hydrates/binds them. |
| 19 | Active allergens atomically persist with ownerless item, audit, and cache behavior | Go HTTP/repository/app integration tests | PASS | Full backend and isolated repository/app suites pass; SQL is parameterized and transaction-owned. |
| 20 | Manual-item/classification/allergen responses decode against production contracts | backend response inspection plus frontend decoder probe | PASS | `manualItemData` now omits absent optional fields while retaining required arrays; canonical v4/RFC4122 seed IDs and the reference migration pass isolated PostgreSQL/idempotency tests; repaired browser fixtures and frontend decoder tests pass. |
| 21 | Moderate-volume threshold and larger offline path are documented | operations doc inspection | PASS | Runbook caps this workflow at 500 and directs larger datasets to separately designed offline ingestion. |
| 22 | OpenAPI/generated drift, focused Python, backend, security, traceability, and diff checks pass | commands in section 8 | PASS | Focused and full aggregate gates pass; 309 browser tests passed/5 skipped, frontend coverage is machine-current at 95.19% functions/96.06% lines, Phase 08 Go coverage is the exact documented 4638/4979 (93.2%) exception, and static/security/traceability/diff checks pass. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | `CatalogError`, `DuplicateJSONKey`, `AuthorizationError` | exceptions | `scripts/import-global-catalog.py:42-52` | added | importer error boundary | Python validation/auth |
| 2 | `reject_duplicate_keys` | function | `scripts/import-global-catalog.py:54` | added | `load_document` | duplicate JSON |
| 3 | `load_document` | function | `scripts/import-global-catalog.py:69` | added | `run` | schema/limit/malformed |
| 4 | `canonical_name` | function | `scripts/import-global-catalog.py:87` | added | `preflight` | name resolution |
| 5 | `finite_nonnegative` | function | `scripts/import-global-catalog.py:92` | added | `validate_item` | overflow/metric |
| 6 | `validate_item` | function | `scripts/import-global-catalog.py:102` | added | `validate_document` | malformed/schema rules |
| 7 | `validate_document` | function | `scripts/import-global-catalog.py:190` | added | `run` | complete preflight |
| 8 | `key_fingerprint` | function | `scripts/import-global-catalog.py:201` | added | report creation | fingerprint |
| 9 | `APIResponse` | data type | `scripts/import-global-catalog.py:207` | added | API client/preflight | fake HTTP |
| 10 | `APIClient.__init__` | method | `scripts/import-global-catalog.py:218` | added | `run` | URL/cookie |
| 11 | `APIClient.request` | method | `scripts/import-global-catalog.py:230` | added | login/preflight/create | HTTP fake server |
| 12 | `APIClient.login` | method | `scripts/import-global-catalog.py:253` | added | `run` | CSRF/cookie |
| 13 | `APIClient.pace` | method | `scripts/import-global-catalog.py:267` | added | `create` | fake clock |
| 14 | `APIClient.create` | method | `scripts/import-global-catalog.py:273` | added | import loop | retry/status |
| 15 | `retry_after_seconds` | function | `scripts/import-global-catalog.py:307` | added | `create` | bounded retry |
| 16 | `response_data` | function | `scripts/import-global-catalog.py:325` | added | `preflight` | envelope errors |
| 17 | `preflight` | function | `scripts/import-global-catalog.py:335` | added | `run` | names/UUID/allergen |
| 18 | `write_report` | function | `scripts/import-global-catalog.py:388` | added | `run` | report |
| 19 | `run` | function | `scripts/import-global-catalog.py:394` | added | CLI entry point | e2e/partial/sanitize |
| 20 | `valid_entry`, `ValidationTests` | test group | `scripts/test_import_global_catalog.py:26-153` | added/extended | importer validation | 9 methods |
| 21 | `StubClient`, `PreflightTests` | test group | `scripts/test_import_global_catalog.py:155-196` | added/extended | preflight | 3 methods |
| 22 | `FakeClock`, `SequenceClient`, `RetryTests` | test group | `scripts/test_import_global_catalog.py:198-255` | added/extended | retry/pacing | 3 methods |
| 23 | `OperatorServer`, `EndToEndTests` | test group | `scripts/test_import_global_catalog.py:257-374` | added | authenticated e2e | 3 methods |
| 24 | `repository.FoodItemEntity` | type | `backend/internal/repository/types.go:85` | modified | itemcurator/repository | Go integration |
| 25 | `itemcurator.Request` | type | `backend/internal/itemcurator/service.go:27` | modified | HTTP/service | Go service/HTTP |
| 26 | `itemcurator.Item` | type | `backend/internal/itemcurator/service.go:54` | modified | controller/frontend | response tests |
| 27 | `itemcurator.Service.Create` | method | `backend/internal/itemcurator/service.go:112` | modified | manual controller | replay/live |
| 28 | `itemcurator.Service.Update` | method | `backend/internal/itemcurator/service.go:165` | modified | manual controller | CRUD/live |
| 29 | `itemcurator.requestHash` | function | `backend/internal/itemcurator/service.go:211` | modified | idempotency | replay |
| 30 | `itemcurator.validateRequest` | function | `backend/internal/itemcurator/service.go:222` | added | Create/Update | validation |
| 31 | `itemcurator.toEntity` | function | `backend/internal/itemcurator/service.go:260` | modified | repository | persistence |
| 32 | `itemcurator.fromEntity` | function | `backend/internal/itemcurator/service.go:280` | modified | response/audit | readback |
| 33 | `httpapi.validateManualItemBody` | function | `backend/internal/httpapi/manual_item_controller.go:177` | modified | admin route | HTTP tests |
| 34 | `httpapi.manualItemRequest` | function | `backend/internal/httpapi/manual_item_controller.go:191` | modified | Create/Update | HTTP tests |
| 35 | `httpapi.decodeManualItemRequest` | function | `backend/internal/httpapi/manual_item_controller.go:200` | added | middleware/controller | duplicate/invalid |
| 36 | `httpapi.hasDuplicateString` | function | `backend/internal/httpapi/manual_item_controller.go:234` | added | decoder | duplicate |
| 37 | `httpapi.manualItemData` | function | `backend/internal/httpapi/manual_item_controller.go:264` | modified | response envelope | optional-field decoder/browser contract |
| 38 | `repository.PostgresManualFoodItemRepository.Update` | method | `backend/internal/repository/manual_food_repository.go:124` | modified | itemcurator | PostgreSQL |
| 39 | `repository.createManualFoodItem` | function | `backend/internal/repository/manual_food_repository.go:165` | modified | ClaimCreate | PostgreSQL |
| 40 | `repository.getManualFoodByID` | function | `backend/internal/repository/manual_food_repository.go:187` | modified | read/audit | PostgreSQL |
| 41 | `repository.validateManualFoodAllergens` | function | `backend/internal/repository/manual_food_repository.go:203` | added | create/update | PostgreSQL |
| 42 | `repository.replaceManualFoodAllergens` | function | `backend/internal/repository/manual_food_repository.go:216` | added | transaction | PostgreSQL |
| 43 | `repository.hydrateManualFoodAllergens` | function | `backend/internal/repository/manual_food_repository.go:228` | added | read/audit | PostgreSQL |
| 44 | `food_allergens_validate` | SQL | `backend/internal/repository/sql/food_allergens_validate.sql:1` | added | active validation | PostgreSQL |
| 45 | `food_allergens_clear` | SQL | `backend/internal/repository/sql/food_allergens_clear.sql:1` | added | replacement | PostgreSQL |
| 46 | `food_allergens_replace` | SQL | `backend/internal/repository/sql/food_allergens_replace.sql:1` | added | replacement | PostgreSQL |
| 47 | `food_allergens_list` | SQL | `backend/internal/repository/sql/food_allergens_list.sql:1` | added | hydration | PostgreSQL |
| 48 | `AdminItemForm` | interface | `frontend/src/lib/admin-workflows.ts:6` | modified | Svelte form | workflow tests |
| 49 | `parseAdminItemForm` | function | `frontend/src/lib/admin-workflows.ts:27` | modified | manual save | workflow tests |
| 50 | `decodeItem` | function | `frontend/src/lib/api/admin-client.ts:173` | modified | admin API consumers | client tests |
| 51 | `allergenKeys` | function | `frontend/src/lib/api/admin-client.ts:281` | added | `decodeItem` | client tests |
| 52 | generated `AdminItemRequest` | interface | `frontend/src/lib/api/generated.ts:1246` | modified | workflow/client | generator/typecheck |
| 53 | `phase08_contract_mismatches` | function | `scripts/generate-api-types.py:512` | modified | generator gate | generator tests |
| 54 | `validate_generator_tests` | function | `scripts/check.py:788` | modified | static lane | quick/full gate |
| 55 | `TRACEABLE_FILES` | configuration | `scripts/check.py:692` | modified | traceability scanner | traceability |
| 56 | `validate_global_catalog_operator_tests` | function | `scripts/check.py:793` | added | static lane | Python suite |
| 57 | `task271ItemBody` | function | `backend/internal/app/task271_backend_regression_integration_test.go:184` | modified | production gate | live integration |
| 58 | `task271CustomItemBody` | function | `backend/internal/app/task271_backend_regression_integration_test.go:188` | added | private fixture | live integration |
| 59 | `task271CreateCustomItem` | function | `backend/internal/app/task271_backend_regression_integration_test.go:311` | modified | production gate | live integration |
| 60 | `AdminItemRequest/AdminItem` schemas | contract | `api/openapi.yaml:1916` | modified | generator/runtime clients | OpenAPI lint |
| 61 | manual HTTP test group | test group | `backend/internal/httpapi/manual_item_controller_test.go:91-220` | modified | manual route | backend suite |
| 62 | manual PostgreSQL test group | test group | `backend/internal/repository/manual_food_repository_test.go:17-220` | modified | persistence | backend suite |
| 63 | Task 271 regression test group | test group | `backend/internal/app/task271_backend_regression_integration_test.go:184-319` | modified | production composition | backend suite |
| 64 | frontend workflow test group | test group | `frontend/src/lib/admin-workflows.test.ts:6-52` | modified | parser | Bun suite |
| 65 | frontend admin-client test group | test group | `frontend/src/lib/api/admin-client.test.ts:16-158` | modified | decoder/generated request | Bun suite |
| 66 | generator strict-envelope test | test method | `scripts/test_generate_api_types.py:86-93` | modified | generated contract | generator suite |
| 67 | `AdminDataManagement` allergen form boundary | component unit | `frontend/src/lib/components/AdminDataManagement.svelte:12-99,289` | modified in repair | AdminItemForm/parser/API | Bun/build/source test |
| 68 | allergen caller regression test | test method | `frontend/src/lib/components/AdminDataManagement.test.ts:48-54` | added in repair | component boundary | Bun suite |
| 69 | canonical classification seed migration | SQL/data boundary | `backend/internal/seed/development.sql:10-80` | modified in repair | admin classification responses and all known references | isolated PostgreSQL seed test |
| 70 | seed UUID/reference/idempotency tests | test group | `backend/internal/seed/seed_test.go:95-190` | modified in repair | development seed contract | PostgreSQL seed suite |
| 71 | admin browser response fixtures | browser fixture group | `frontend/tests/admin-data-management.spec.ts:1-280` | modified in repair | AdminDataManagement response decoder | 18 desktop/mobile cases |
| 72 | coverage metadata and checker contract | gate/test group | `docs/implementation/04_OPEN.md:475-531`, `scripts/test_check_coverage.py` | modified in repair | `scripts/check.py` Phase 08 checker | 21 coverage-contract tests/full gate |
| 73 | importer validation bounds | constants/configuration | `scripts/import-global-catalog.py:34-36` | added in F1 repair | `bounded_string`, `valid_image_url`, `validate_item` | focused optional-field probes |
| 74 | `bounded_string` | function | `scripts/import-global-catalog.py:107` | added in F1 repair | `validate_item`, `valid_image_url` | every bounded optional string probe |
| 75 | `valid_image_url` | function | `scripts/import-global-catalog.py:112` | added in F1 repair | `validate_item` | wrong type, length, scheme, URI probes |
| 76 | F1 importer regression tests | test methods | `scripts/test_import_global_catalog.py:133-213` | added in F1 repair | loader/validator/CLI boundary | 17-test focused suite |

```yaml
inventory_source_count: 76
audited_symbol_count: 76
inventory_complete: true
generated_groupings:
  - "Generated frontend output is represented by the source OpenAPI schemas, generator drift guard, generated AdminItemRequest unit, and strict client tests; no generated implementation was hand-edited outside the generator output."
```

## 6. Function-Level Audit

Every inventory unit was inspected line by line against its callers, dependencies, tests, and trust boundary. F1–F7 are closed in the current tree; the repaired loader, optional-field validation, and pre-login/no-mutation regressions were rechecked directly.

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| 1 `CatalogError` family | safe operator-facing errors | fixed duplicate/auth/validation paths | no retained state | suppresses causes | pure | minimal hierarchy | malformed/auth tests | PASS |
| 2 `reject_duplicate_keys` | rejects duplicates at every object depth | duplicate key is deterministic error | local map only | no body output | linear object traversal | standard JSON hook | nested/root tests | PASS |
| 3 `load_document` | closed versioned root and item count | malformed/schema/limit/parser overflow paths become safe `CatalogError` | file read only | no secrets in message | count bounded; file size not explicitly bounded | simple loader | schema/duplicate/limit/10,000-digit/CLI probes | PASS |
| 4 `canonical_name` | case/whitespace normalization | string inputs are prevalidated | pure | no trust crossing | linear string work | deterministic | ambiguity/name tests | PASS |
| 5 `finite_nonnegative` | bounded finite non-boolean number | catches huge-int `OverflowError` | pure | no output | constant work | explicit bound | overflow/nonfinite tests | PASS |
| 6 `validate_item` | complete typed metric item | every optional metric, provenance, classification, and image field has type/bound/URI validation before remote state | pre-mutation pure path | no remote state | bounded collections | closed allowlist | 17-test suite covers malformed optional values and no-login/no-client behavior | PASS |
| 7 `validate_document` | all items/keys validated before auth/mutation | invalid document and duplicate keys observable | no mutation | no key output | <=500 entries | deterministic | document tests | PASS |
| 8 `key_fingerprint` | stable irreversible report identity | fixed SHA prefix | pure | raw key excluded | constant | appropriate API | stable/safe test | PASS |
| 9 `APIResponse` | status/body/header carrier | malformed body delegated to safe protocol handling | no resource ownership | no logging | small carrier | idiomatic dataclass | fake HTTP | PASS |
| 10 `APIClient.__init__` | rejects credential-bearing URL; creates memory jar | invalid URL fails safe | CookieJar memory-only | no persistent cookie path | bounded URL/session setup | standard urllib | e2e | PASS |
| 11 `APIClient.request` | JSON request/response boundary | HTTPError decoded; malformed body becomes None | response closed in finally | body not printed | 30s timeout; response bytes not capped | standard library | fake server | PASS with bounded-body follow-up gap |
| 12 `APIClient.login` | login then fresh CSRF | bad status/envelope aborts | CSRF set only after valid response | password only request body, never report | two requests | clear sequence | cookie/CSRF e2e | PASS |
| 13 `APIClient.pace` | at least 2.1s between attempts | first immediate; clock skew clamps | injectable monotonic clock | none | bounded delay | simple | fake clock | PASS |
| 14 `APIClient.create` | same key/body ambiguity retry, status policy | 5xx/transport/429 retry; 400/409 permanent; auth abort | max 3 attempts; pacing every attempt | key only header, no logging | bounded attempts/delay | explicit policy | retry/status tests | PASS |
| 15 `retry_after_seconds` | delta/date and <=30s bound | invalid/negative/overbound safely ignored | pure | no output | constant | standard parser | 2/31s tests; valid date path not separately asserted | PASS with coverage gap |
| 16 `response_data` | exact success envelope | TypeError/KeyError maps safe | pure | remote text not exposed | constant | small boundary helper | preflight fake | PASS |
| 17 `preflight` | all active classifications/allergens resolved before create | unknown/ambiguous/malformed refs fail; UUIDs canonicalized | no mutation until complete | remote data treated untrusted | maps rebuilt deterministically | clear two alternatives | name/UUID/allergen tests | PASS |
| 18 `write_report` | only schema/results safe fields | report path/write failure reaches CLI safe error | file write not atomic | no PII fields supplied | report size bounded by 500 records | simple but non-atomic | report e2e | PASS with interrupted-write gap |
| 19 `run` | exit 0/1/2 and sanitized errors | catches arbitrary operator exceptions without traceback | synchronous process-local state | credentials/email/body never printed | <=500 item loop | direct CLI | e2e and malformed probes | PASS |
| 20 Python validation tests | adversarial importer contract | includes malformed runtime types/overflow/nonfinite | temp files cleaned | asserts safe output | bounded fixtures | focused | 11 validation methods, including every optional-field type/bound family | PASS |
| 21 Python preflight tests | remote resolution contract | unknown/ambiguous/allergen failures | stub state local | no secrets | small | focused stubs | 3 methods | PASS |
| 22 Python retry tests | pacing/retry identity | 5xx Retry-After, 429, permanent/auth | fake clock | key/body equality only | bounded | focused | 3 methods | PASS |
| 23 Python HTTP e2e tests | auth/dry-run/create/report/partial | server responses and teardown | in-memory cookie observed | PII assertions | one item and small partial set | focused fake server | 3 methods | PASS |
| 24 `FoodItemEntity` | carries canonical allergen keys | nil projection normalized later | entity passed through tx | ownerless global boundary | slice-sized | minimal field | live repository | PASS |
| 25 `itemcurator.Request` | global request adds keys without private ownership | validator rejects invalid keys | normalized clone/sort | server-side authority | <=100 keys | explicit type | service/HTTP | PASS |
| 26 `itemcurator.Item` | response carries non-null keys | nil normalized empty | response clone | no owner/audit fields | bounded | contract type | HTTP/client | PASS |
| 27 `Service.Create` | validates, hashes, claims, creates ownerless item | conflict/replay/error propagation | tx/audit/invalidation caller-owned | authenticated admin boundary | one item | existing idempotency design | live replay | PASS |
| 28 `Service.Update` | validates and atomically replaces classifications/allergens | validation and persistence errors propagate | tx-owned replacement | admin route only | bounded relationships | explicit workflow | live CRUD | PASS |
| 29 `requestHash` | stable normalized body hash | JSON encoder/error path | no mutation | key/body conflict integrity | small body | existing pattern | replay/conflict | PASS |
| 30 `validateRequest` | metric domain plus canonical sorted keys | blank/case/duplicate/too-many reject | clone avoids caller mutation | backend validation | <=100 | reuses customitem validator | service tests/live | PASS |
| 31 `toEntity` | maps keys to ownerless entity | classifications preserve kind | no commit | no owner field | slices proportional to input | direct map | repository integration | PASS |
| 32 `fromEntity` | returns canonical non-null arrays | nil micros/allergens empty | allergen clone | excludes owner/audit | bounded read | direct map | readback | PASS |
| 33 `validateManualItemBody` | duplicate JSON and global request schema | malformed/duplicate safe 400 | locals request only | CSRF/auth middleware upstream | body already bounded upstream | reuses boundary | HTTP invalid tests | PASS |
| 34 `manualItemRequest` | validated request reuse | fallback decodes same contract | no state | authenticated route | constant | minimal | HTTP | PASS |
| 35 `decodeManualItemRequest` | required array, no duplicate strings | null/nonarray/duplicate reject | temporary raw map only | untrusted JSON | bounded by upstream body | explicit extension | HTTP tests | PASS |
| 36 `hasDuplicateString` | exact duplicate detection | empty/unique/duplicate | local map | no output | O(n) | idiomatic | duplicate test | PASS |
| 37 `manualItemData` | response omits absent optionals and includes empty required arrays | conditional projection now omits zero/empty optional fields | no resource | contract boundary | small map | typed projection is compatible | HTTP/frontend/browser contract tests | PASS |
| 38 repository `Update` | active allergen validation and atomic replacement | every failure returns to audit tx | no own commit | parameterized SQL | bounded keys | existing repository seam | live rollback | PASS |
| 39 `createManualFoodItem` | active validation before row and relationship writes | failures roll back outer tx | tx-owned | parameterized SQL | one item | direct helper | live create | PASS |
| 40 `getManualFoodByID` | authoritative item plus classifications/allergens | hydration errors propagate | rows owned/closed by helper | response data trusted only after decode | bounded rows | explicit hydration | live read | PASS |
| 41 `validateManualFoodAllergens` | every key active | DB errors mapped; inactive false | query only | DB vocabulary authority | one aggregate query | parameterized | live inactive test | PASS |
| 42 `replaceManualFoodAllergens` | clear/insert in caller transaction | either error aborts tx | no own commit | parameterized IDs/arrays | <=100 rows | simple SQL boundary | live atomicity | PASS |
| 43 `hydrateManualFoodAllergens` | sorted canonical response | scan/rows errors propagate | rows closed | no PII | bounded item relations | deterministic | live readback | PASS |
| 44 `food_allergens_validate` | active-count equality | empty array valid; inactive rejects | transaction caller | no injection | one query | parameterized | PostgreSQL | PASS |
| 45 `food_allergens_clear` | delete only target item assignments | DB error propagates | tx caller | item ID parameter | one delete | simple | PostgreSQL | PASS |
| 46 `food_allergens_replace` | insert supplied keys | DB constraints/errors propagate | tx caller | parameters | <=100 rows | simple | PostgreSQL | PASS |
| 47 `food_allergens_list` | deterministic sorted hydration | empty returns empty | rows caller closes | target ID parameter | bounded | simple | PostgreSQL | PASS |
| 48 `AdminItemForm` | form must carry required keys | type-level field present | component-owned state | no private fields | small | generated request boundary | workflow/component tests | PASS |
| 49 `parseAdminItemForm` | validates and emits keys | valid empty/nonempty and invalid paths are covered; exact machine metrics are current | pure | canonical regex | small | existing parser | Bun/full coverage contract | PASS |
| 50 `decodeItem` | strict response contract | missing/invalid keys reject | bounded body reader | untrusted API response | bounded response | fail-closed | client suite | PASS, exposes F5 |
| 51 frontend `allergenKeys` decoder | canonical unique bounded array | malformed types/keys reject | pure | untrusted API response | O(n) | helper matches OpenAPI | client tests | PASS |
| 52 generated `AdminItemRequest` | source contract requires keys | generator drift fails | type-only | no runtime secrets | none | generated source | generator/typecheck | PASS |
| 53 `phase08_contract_mismatches` | detects allergen schema drift | missing schema/pattern/unique reports | pure scan | source contract guard | small | explicit guard | generator tests | PASS |
| 54 `validate_generator_tests` | quality lane executes generator tests | subprocess failures propagate | no shared state | none | gate only | quick/full | PASS |
| 55 `TRACEABLE_FILES` | new operator/tests are traceable | missing mapping fails | static | none | scan | traceability | PASS |
| 56 `validate_global_catalog_operator_tests` | focused Python gate is mandatory | test failure propagates | subprocess | none | one suite | quick/full | PASS |
| 57 `task271ItemBody` | global fixture includes keys | private fixture separated | test data only | no production data | small | explicit helper | live regression | PASS |
| 58 `task271CustomItemBody` | private fixture excludes global allergen extension | request remains compatible | test data only | private/global isolation | small | explicit helper | live regression | PASS |
| 59 `task271CreateCustomItem` | regression exercises both paths | errors propagate | test tx/Redis | no leakage | bounded | explicit call path | live regression | PASS |
| 60 OpenAPI schemas | required unique canonical keys | closed schema and response array | contract only | excludes owner fields | bounded arrays | source of truth | lint/generated | PASS |
| 61 manual HTTP tests | request/persistence/audit/invalidation behavior | invalid/replay and optional-field omission paths | fake tx/audit | safe response assertions include strict decoder projection | small | focused | backend suite | PASS |
| 62 manual PostgreSQL tests | atomic allergen persistence/replay/rollback | inactive and error paths | isolated DB | parameterized SQL | bounded | integration seam | backend suite | PASS |
| 63 Task 271 regression tests | full app/auth/cache composition | global/private and replay paths | isolated PostgreSQL/Redis | auth/ownerless boundary | production composition | integration | backend suite | PASS |
| 64 frontend workflow tests | request includes keys | empty, valid nonempty, and invalid key paths | pure | no secrets | small | focused | Bun/full coverage suite | PASS |
| 65 frontend admin-client tests | strict key response and request | fixtures use contract-valid optional omission and UUID values | fetch mock | response fail-closed | bounded body | focused | client/full suite | PASS |
| 66 generator strict-envelope test | generated success envelope stays strict | drift mutation paths | pure | contract | small | focused | Python generator suite | PASS |
| 67 `AdminDataManagement` allergen form boundary | initialize, hydrate, bind, submit keys | repaired field is present across caller path | component state/authoritative refresh | server remains authority | static options match current seven-key seed vocabulary | build/source test; no rendered runtime test | PASS for F2 |
| 68 allergen caller regression test | protects repaired caller strings | static assertions only | no runtime component state | no data output | negligible | source-level test | Bun suite | PASS with behavioral coverage gap |
| 69 canonical classification seed migration | all legacy seed IDs map to valid canonical UUIDs and known references migrate before old rows are removed | two-run/idempotency and old-reference paths | one transaction/temporary map | response contract compatibility | bounded four-row seed | explicit migration | isolated PostgreSQL | PASS |
| 70 seed UUID/reference/idempotency tests | prove version/variant, reference migration, and replay stability | legacy rows, canonical rows, second run | isolated database reset | no PII | bounded fixtures | focused integration | seed suite | PASS |
| 71 admin browser response fixtures | every manual-item response matches required `allergenKeys` and optional omission contract | CRUD/refresh/PUT assertions | Playwright fixture-local state | no secrets | small responses | existing fixture seam | 18 admin desktop/mobile cases | PASS |
| 72 coverage metadata and checker contract | exact measured rows and summaries remain machine-checked | Markdown whitespace/parser and current metrics | subprocess gate only | no secrets | profile parsing bounded by source set | explicit checker tests | 21 contract tests/full gate | PASS |
| 73 importer validation bounds | optional-field limits are centralized and applied consistently | constants bound text, classifications, and image URLs | immutable module constants | no output | constant lookup | small explicit bounds | focused malformed optional probes | PASS |
| 74 `bounded_string` | rejects nonstrings, NUL, and overlong values | all malformed string shapes fail safely | pure | no output | linear length/NUL scan | reusable helper | provider/FoodId/classification/name probes | PASS |
| 75 `valid_image_url` | accepts only bounded HTTP(S)/relative URI references | list/object/bool, controls, overlength, bad scheme, and malformed URI fail | pure | no remote state | bounded parse | standard `urlsplit` | focused imageUrl probes | PASS |
| 76 F1 importer regression tests | lock complete pre-login rejection and sanitized overflow | malformed optional values and 10,000-digit integer exit 2 without prompts/client | temporary files/mocks cleaned | assert no credential/API path | bounded fixtures | focused adversarial tests | 17 tests pass | PASS |

## 7. Findings

F1–F4 and the second-cycle F5–F7 repairs are closed in the current code. The final F1 repair converts parser integer-limit failures to `CatalogError`, validates every accepted optional field before authentication, and proves malformed optional inputs cannot reach login or the API client. No blocking, important, or optional findings remain.

```yaml
blocking_findings: 0
important_findings: 0
optional_findings: 0
```

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| `git diff --check` | repository root | 0 | PASS | no whitespace errors |
| `python3 -m unittest -v scripts/test_import_global_catalog.py` | repository root | 0 | PASS | 17 tests, including malformed optional-field/no-login probes and 10,000-digit overflow |
| `python3 -m py_compile scripts/import-global-catalog.py scripts/test_import_global_catalog.py` | repository root | 0 | PASS | importer and focused tests compile |
| `python3 -m unittest -v scripts/test_check_coverage.py` | repository root | 0 | PASS | 21 coverage-contract tests |
| F1 CLI and direct loader probe for malformed JSON, wrong types, overflow, and duplicate keys | repository root | 0 | PASS | `load_document` raises sanitized `CatalogError`; every malformed optional case exits 2 without prompts or `APIClient` construction |
| F3/F4 direct probe for 503 `Retry-After` and uppercase UUID | repository root | 0 | PASS | 4-second bounded delay and lowercase UUID output |
| `python3 scripts/check.py --quick` | repository root | 0 | PASS | static, changed backend/frontend, generator, OpenAPI, 17-test operator suite, vet, govulncheck |
| `python3 scripts/validate-traceability.py` | repository root | 0 | PASS | traceability passed |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS | 286 ordered tasks |
| `python3 scripts/generate-api-types.py --check` | repository root | 0 | PASS | generated types current |
| `npx --no-install redocly lint api/openapi.yaml` | repository root | 0 | PASS with existing warning | intentional OAuth callback 302-only warning |
| `bunx playwright test tests/admin-data-management.spec.ts --workers=1` | `frontend` | 0 | PASS | 18 repaired desktop/mobile admin cases |
| `cd backend && ... go test ./internal/seed -run 'TestDevelopmentClassificationIDsMatchPublicUUIDContract|TestRunIsIdempotentAndSeedsRepositoryFixtures' -count=1` | `backend` | 0 | PASS | canonical UUID/reference migration and two-run replay |
| `cd frontend && ... bun test --coverage` | `frontend` | 0 | PASS | 536 tests, 2803 expectations; 95.19% functions/96.06% lines |
| `cd frontend && ... bun run build` | `frontend` | 0 | PASS | Vite production build |
| `cd backend && ... go test ./...` | `backend` | 0 | PASS | all packages, isolated PostgreSQL/Redis suites included |
| `cd backend && ... go vet ./...` | `backend` | 0 | PASS | run in quick gate |
| `cd backend && ... go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `backend` | 0 | PASS | no vulnerabilities in called code |
| `cd backend && ... go test -race ./...` | `backend` | 0 | PASS | full race suite |
| `cd backend && ... go test ./internal/... -coverprofile=/tmp/task-277-coverage.out && go tool cover -func=/tmp/task-277-coverage.out` | `backend` | 0 | PASS with existing documented exceptions | aggregate package line coverage 87.4%; changed packages: httpapi 87.4%, itemcurator 74.2%, repository 86.3% |
| `python3 scripts/check.py` | repository root | 0 | PASS | all four lanes pass; browser 309 passed/5 skipped; exact Phase 08 coverage contracts pass; backend lane completed 371.8s |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/evidence/task-277-review.md` | repository root | 0 | PASS | this evidence structurally validates after writing |

The `golang-security` skill was read and applied to the Go HTTP/auth/CSRF, SQL parameterization, cookie/secret, error/logging, and resource-boundary review. The `code-review-skill` was invoked exactly once; its Python, Go, TypeScript, and Svelte guides were read.

## 9. Files Inspected and Staleness Fingerprints

All current Task 277 implementation and directly reviewed dependency files were hashed after inspection. SHA-256 is used throughout.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| `api/openapi.yaml` | source request/response contract | none | SHA-256 | `69397349b60bd9744effea07cc249f8a6a65e626bf3e83b8d682c89a189372fc` |
| `backend/internal/app/task271_backend_regression_integration_test.go` | production integration fixture | none | SHA-256 | `606ee1c347602bf7e17574832f3ede551669a6df43479ada924b6a8068186a3c` |
| `backend/internal/httpapi/manual_item_controller.go` | manual request/response boundary | F1–F2 pass; optional response repair verified | SHA-256 | `9611e4b4303c2f10b2508a7d88bf592257d7f76a5b31e973712c5a80aef5b01a` |
| `backend/internal/httpapi/manual_item_controller_test.go` | manual HTTP tests | optional response/allergen tests | SHA-256 | `9ad6c9c34d0d103cf634044a9269e5aee96027992f0b2887f93b4c64d4152466` |
| `backend/internal/itemcurator/service.go` | service request/entity/response | none | SHA-256 | `c1e1a5f8309152ec65cf588c49a50c9a5fdf4ab7c24f1bf63b02abcd0521abd7` |
| `backend/internal/itemcurator/service_test.go` | adjacent service tests | none | SHA-256 | `5e3932a2ecfff6d3fe6d1f4c822f9fcdbc6620e755a07bf1103dfdd5fdee7bfe` |
| `backend/internal/repository/manual_food_repository.go` | atomic allergen persistence | none | SHA-256 | `140c197b6a3bfc24be809309c702ebf7da0456fd639f762cf8dcb38ac042b203` |
| `backend/internal/repository/manual_food_repository_test.go` | persistence integration | none | SHA-256 | `674e9dea370b5231fe8e845011f386b456e1588f89b06c95785ec23a3a3978f3` |
| `backend/internal/repository/types.go` | entity boundary | none in Task 277 field | SHA-256 | `ba165efac8cb76ba71e3b7f537abbd0455ba4a20b10447450d1e8bc32422b6d3` |
| `backend/internal/repository/sql/food_allergens_clear.sql` | clear statement | none | SHA-256 | `3625eb03dbc07707aa37f77aad3f0ff7be30d0a79636351db8d50592af4b3052` |
| `backend/internal/repository/sql/food_allergens_list.sql` | list statement | none | SHA-256 | `fc70951c71f0cda4c889bb15ef9d2f79c1959f64a39a1941a34e6624b64eb86c` |
| `backend/internal/repository/sql/food_allergens_replace.sql` | replace statement | none | SHA-256 | `aa93bf9b7843abe7a1b327bc697de7981de15e0e2a69fadfcb3ac82ca5d1c161` |
| `backend/internal/repository/sql/food_allergens_validate.sql` | active validation | none | SHA-256 | `84283481a6aca98238c8e94431815fe12484d10627dd4d7e26846d1d29d4ab01` |
| `frontend/src/lib/admin-workflows.test.ts` | parser tests | F7 repair verification | SHA-256 | `f90688587108af37de1e90ea630a783a5d277bd17372d02d97b656407cb89a3e` |
| `frontend/src/lib/admin-workflows.ts` | request parser | F7 repair verification; F1 is importer-only | SHA-256 | `ff2db65f18a045eae9ab7017851ab9162739744dd7aeea800f74a36fbe41ad24` |
| `frontend/src/lib/api/admin-client.test.ts` | strict client tests | repaired optional/UUID contract | SHA-256 | `68ab30fbefd4b3ed5b014d0168f96170809fdad82d39b9d1577aacefff973884` |
| `frontend/src/lib/api/admin-client.ts` | response decoder | F5/F6 boundary | SHA-256 | `71b76d457ff4c8379cdabab6b5e3cb6f56cf0e22ab9249abe06e3dd8678be880` |
| `frontend/src/lib/api/generated.ts` | generated contract | none | SHA-256 | `c9f6d7e8d8ff0b991b40168a0b7d707d70558b8ce36e886dc4118e1f722df649` |
| `frontend/src/lib/components/AdminDataManagement.svelte` | repaired allergen caller | F2 pass; response dependency | SHA-256 | `0064ebe05b99f29d288d05edaab4614a5355de4c93d29eb8727df852c21ab14a` |
| `frontend/src/lib/components/AdminDataManagement.test.ts` | repaired caller regression | behavioral gap only | SHA-256 | `b6b009f88c8acdb6e75e73df656a18d16ed4397207bbe9a84a36d4a69ed94b3f` |
| `scripts/check.py` | quality/coverage/operator gate | current exact coverage/full-gate pass | SHA-256 | `b7618066a073b6b406c4a841c2a537f211462200319048a4bb9c5c3f286e856c` |
| `scripts/generate-api-types.py` | generator | none | SHA-256 | `01ae86e27af53cbeeaa76cd95f0b85ad45e8659f9448704eb8bfbd607e3a49b9` |
| `scripts/test_generate_api_types.py` | generator tests | none | SHA-256 | `cbed90622dd05718eb7bc0f2a43b58d38bc5aa9cc668518d2a104f9ddb194348` |
| `scripts/import-global-catalog.py` | operator implementation | F1–F4 pass | SHA-256 | `c34b49f753accb403c1672688c99ac3497379346128a5a383e218f251f8dd014` |
| `scripts/test_import_global_catalog.py` | operator tests | F1–F4 focused coverage | SHA-256 | `ca68191084f7cd061c3ae641e4ae17ae43e40a10d46778abe1fd8e83412702a6` |
| `docs/design/DESIGN-009.md` | design source | none | SHA-256 | `22ca65baf347c8c1598a4a71439ca5091548fc3a8f9b5dc906406a3fdcc8c814` |
| `docs/operations/global-catalog-import.md` | operator runbook | none | SHA-256 | `97256538264b77e6027ecc26d63a30c3ebc60b650da6af66ded8412446fe15d9` |
| `backend/internal/seed/development.sql` | canonical classification seed/reference migration | F6 repaired | SHA-256 | `8948996e646c4f89c96c3bcda1ea9a85a883d8a3fc467998403def263e726dbe` |
| `backend/internal/seed/seed_test.go` | canonical UUID/reference/idempotency tests | F6 repaired | SHA-256 | `183e8d53fd28b1127173d7dd48a9f5305b0d6302b7004f4e626a945cab68e53c` |
| `frontend/tests/admin-data-management.spec.ts` | repaired browser fixtures | F6/F2 response/request fixtures | SHA-256 | `b80676a865c30017fb40ed847ab5658a421d9d0d0e88703df69ee9e27d41f4db` |
| `scripts/test_check_coverage.py` | coverage metadata regressions | F7 repaired | SHA-256 | `4e6bbf9b012b625a8a69b8d79d54a1775e2103d013afa69c2b4a0c0501b2040d` |
| `docs/implementation/04_OPEN.md` | Phase 08 coverage contract and repair notes | F7 repaired/current | SHA-256 | `dd4cf76b93601b05b4c570132c4a7c62af290e4f23c346d95419c13ee60a9584` |
| `docs/implementation/evidence/task-277-preparation.md` | updated preparation evidence checked for staleness | current claim, not proof | SHA-256 | `e4499640355b899ea95afab5f3c70d87026aaf1c92a6081e1e81a28ba195f68c` |
| `docs/implementation/02_TASK_LIST.md` | current status reference | externally changed, not edited | SHA-256 | `41d5c06f705630e8d734195c3b60505f2dd825e9d7a95bdb8a68211440974679` |

```yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "docs/implementation/evidence/task-277-preparation.md: second-cycle claims were treated as claims and independently checked; its content hash is recorded above."
  - "docs/implementation/evidence/task-277-review.md: prior REJECTED F1-F4/F5-F7 evidence is superseded by this fresh current-tree review and refreshed hashes."
  - "docs/implementation/04_OPEN.md: coverage metadata was rechecked against the current measured output; prior F5/F6 open notes were not treated as proof until response/seed tests passed."
```

## 10. Coverage and Exceptions

- [x] Required focused and backend coverage commands ran.
- [x] Report paths and observed thresholds are recorded.
- [x] Untested changed-symbol branches were inspected.
- [x] Full Phase 08 frontend and backend coverage contracts pass with their exact documented exceptions.
- [x] No new Task 277 exception was authorized.

```yaml
coverage_required: true
coverage_exception_allowed: true
coverage_report_path: "/tmp/task-277-coverage.out; Bun coverage captured by scripts/check.py"
observed_line_coverage: "Phase 08 Go 4638/4979 (93.2%); frontend 536 tests/2803 expectations, 95.19% functions and 96.06% lines; admin-workflows.ts 83.33% functions/98.55% lines"
coverage_passed: true
```

The exact machine-checked Phase 08 exceptions are current. The measured frontend function percentage is intentionally represented in the documented row; the repaired checker accepts Markdown alignment whitespace and the 21 regression tests protect the contract.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No public bulk-import endpoint or direct PostgreSQL importer was introduced.
- [x] No imperial persistence fields or position-derived idempotency keys were introduced.
- [x] No credential, cookie, CSRF, email, raw key, item name, body, or request ID was added to operator output/report paths.
- [x] SQL uses embedded parameterized statements and remains transaction-owned.
- [x] Idempotency, audit, ownerless persistence, and cache invalidation integration paths pass.
- [x] Duplicate helpers and generated drift were searched and checked.
- [x] Race, vet, vulnerability, OpenAPI, generated, traceability, and diff checks pass in their focused commands.
- [x] Full aggregate check passes: all static, backend, frontend, browser, coverage, security, and traceability lanes completed successfully; 5 documented browser skips remain.
- [x] F1 complete-validation contract passes: parser overflow is a sanitized `CatalogError`, every optional field is type/bound validated, and malformed values trigger no login, preflight, or mutation.
- [x] F2–F4 prior repair contracts pass, including allergenKeys end-to-end, bounded 5xx Retry-After, and uppercase UUID canonicalization.
- [x] F5–F7 second-cycle repairs pass: optional response omission, valid canonical seed UUID/reference migration, coverage metadata/checker, and browser fixtures.

Finding status: none. F1–F7 repairs and all prior boundary checks are independently confirmed in the current worktree.

## 12. Decision

A task may be `PASSED` only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains. That condition is met.

```yaml
decision: "PASSED"
reason: "F1 is repaired and independently verified; F2-F7 remain closed, all 76 inventory units pass audit, all reviewed files have current hashes, and the full aggregate gate passes."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "none"
```

## 13. Repair Context

### Failure Summary

This final review verifies the F1 repair after the prior rejection: `load_document` now converts parser integer-limit failures to `CatalogError`, `validate_item` validates every optional field before authentication, and focused tests prove malformed optional inputs perform no login or mutation. The prior F2–F4 and F5–F7 repairs remain independently confirmed.

### Minimal Repair Goal

1. Completed: catch JSON integer-limit `ValueError`/parser failures in `load_document` and convert them to the safe `CatalogError` contract.
2. Completed: validate every accepted optional input type and bound in `validate_item`, including `imageUrl` as a bounded NUL-free URI-reference string; direct tests prove malformed input causes no login, preflight, or POST.
3. Completed: preserve F2–F7 repairs and re-run the full aggregate gate, focused importer suite, and PII-safe output checks.

### Evidence to Reuse

Use the passing F2-F4 and F5-F7 probes and 17-test Python suite in section 8, the two repaired F1 probes, current hashes in section 9, the current `docs/implementation/04_OPEN.md` coverage contract, full `go test ./...`, full `go test -race ./...`, and the existing Task 276 overlap/dependency evidence.

### Required Re-Review Surface

`scripts/import-global-catalog.py` and tests at the loader/validator/CLI boundary; manual-item response projection and tests; frontend admin client/workflow/component and tests; `backend/internal/seed/development.sql` plus all classification references/migrations; coverage checker/metadata; browser fixtures; OpenAPI/generated contracts; full browser/admin refresh paths; all prior F2-F7 paths and affected hashes.

### Do Not Change

Do not add a bulk endpoint or direct PostgreSQL importer; do not accept imperial persistence fields or derive keys from positions; do not persist/print credentials, cookies, CSRF tokens, email, request bodies, item names, or other PII; preserve the existing authenticated `POST /api/v1/admin/items` route, atomic item/classification/allergen/audit/cache behavior, and the repaired F2-F7 semantics.
