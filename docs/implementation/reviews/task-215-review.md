# Review Evidence: Task 215 — UnitConverter

```yaml
task_id: 215
component: "DESIGN-005: UnitConverter"
static_aspect: "UnitConverter"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-14T22:03:18Z"
review_agent: "Codex independent re-review"
evidence_file: "docs/implementation/reviews/task-215-review.md"
baseline_ref: "a4e31367485b03269e90b5607f2057c9568bb5b1"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "Go, SQL injection prevention, TypeScript; no OpenAPI-specific guide exists, so the OpenAPI source and Redocly validation were used"
repair_context_required: false
```

## 1. Task Source

**Description:** Phase 07.01: obtain and implement the recorded product/design disposition for `serving`, consolidate saved-diet, repository, HTTP, database, OpenAPI, and frontend validation around one canonical mass/volume unit vocabulary, and isolate any approved recipe/per-unit serving path under a context-specific name and conversion boundary.

**Depends On:** Task 212 — PASSED.

**Testing Coverage Exceptions:** None for task 215. Existing Phase 07 exceptions remain documented in `docs/implementation/04_OPEN.md`; no new exception was introduced.

**Verification Criteria:**

1. One shared validator owns `g`, `ml`, `oz`, and `fl_oz`.
2. Retained `serving` is limited to the approved recipe/per-unit boundary with explicit conversion.
3. Saved-diet and substitution contracts are not broadened to accept `serving`.
4. Cross-basis and unsupported values fail at service, repository, HTTP, and database boundaries.
5. OpenAPI and generated frontend enums agree.
6. Unit, integration, contract-drift, requirement-traceability, and design-traceability checks pass.

## 2. Pre-Review Gates

- [x] Input status is `PREPARED`.
- [x] Every dependency is `PREPARED` or `PASSED`.
- [x] The preparation report claims completion.
- [x] A task-specific baseline/diff is available and trustworthy.
- [x] `code-review-skill` was invoked exactly once and its relevant Go, SQL, and TypeScript guides were read; no OpenAPI-specific guide is present in the skill.
- [x] The reviewer is independent from implementation/repair.
- [x] Review uses current repository state rather than stale logs.
- [x] Reviewer made no production-code or task-list changes.

```yaml
pre_review_gates_passed: true
blocking_issue: "None"
```

## 3. Review Baseline and Change Surface

Baseline/reference method: `HEAD` and `git merge-base` both equal `a4e31367485b03269e90b5607f2057c9568bb5b1`. The preparation manifest defined task ownership for overlapping files; current tracked diffs and untracked task files were then inspected directly. The repaired symbols were re-audited instead of accepting the previous review's result.

Commands used to reconstruct the diff:

```bash
git rev-parse HEAD
git merge-base a4e31367485b03269e90b5607f2057c9568bb5b1 HEAD
git status --short
git diff --stat a4e31367485b03269e90b5607f2057c9568bb5b1 -- <task-owned paths>
git diff --unified=5 a4e31367485b03269e90b5607f2057c9568bb5b1 -- <task-owned paths>
rg -n 'ConvertUnit|ConvertRecipeServingToBase|ValidateQuantityUnit|ValidateRecipeIngredientUnit|serving|validUnit' backend api frontend scripts database docs/design/DESIGN-005.md
sha256sum <all reviewed files>
```

Pre-existing dirty-worktree changes and exclusions:

The worktree contains concurrent task 213, 214, 216, and 217 changes, including overlapping files. Task 215 ownership was limited to the quantity-unit hunks and files in `docs/implementation/preparation/task-215-preparation.md`; unrelated response-matrix, durable-idempotency, repository-surface, and worker changes were excluded from the task inventory. The pre-existing edit to `docs/implementation/02_TASK_LIST.md`, `review.txt`, other task review/preparation files, and concurrent untracked files were not edited. `api/openapi.yaml`, `service.go`, `user_data_repository.go`, `saved_diet_mutation_repository.go`, and the generator contain excluded concurrent hunks; only the canonical-unit symbols below were audited.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| `api/openapi.yaml` | Canonical substitution reference; other response edits excluded | HIGH | `CanonicalQuantityUnit` and three references |
| `backend/internal/dailydiet/service.go` | Shared validator call in `normalizeRequest`; idempotency edits excluded | HIGH | `normalizeRequest` |
| `backend/internal/dailydiet/task215_units_test.go` | New task-specific test | HIGH | `TestTask215SavedDietQuantityUnitBoundaries` |
| `backend/internal/httpapi/daily_diet_controller.go` | Shared validator call | HIGH | `validateDailyDietBodyMap` |
| `backend/internal/httpapi/daily_diet_controller_test.go` | New canonical-unit test; unrelated list test excluded | HIGH | `TestValidateDailyDietBodyMapUsesCanonicalQuantityUnits` |
| `backend/internal/httpapi/search_validation.go` | Shared validator call | HIGH | `validateSubstitutionUnit` |
| `backend/internal/repository/meal_repository.go` | Recipe unit and finite-quantity boundary | HIGH | `validateIngredients`, `ingredientBasisQuantity` |
| `backend/internal/repository/postgres_repository_test.go` | Saved-diet and recipe database boundary tests | HIGH | two PostgreSQL integration tests |
| `backend/internal/repository/repository_test.go` | Converter, validator, and non-finite tests | HIGH | seven unit tests |
| `backend/internal/repository/saved_diet_mutation_repository.go` | Response-unit validation hunk; durable idempotency excluded | HIGH | `validateDailyDietCreateResponse` |
| `backend/internal/repository/units.go` | Shared vocabulary and repaired numeric conversion boundary | HIGH | four repository functions |
| `backend/internal/repository/user_data_repository.go` | Saved-diet validator call; alias removal excluded | HIGH | `validateSavedDietInput` |
| `backend/internal/search/substitution_service.go` | Public substitution unit rejection | HIGH | `sourceBaseQuantity` |
| `backend/internal/search/substitution_service_test.go` | Canonical substitution regression tests | HIGH | two substitution tests |
| `database/migrations/000022_canonical_quantity_units.down.sql` | Task migration rollback | HIGH | rollback statements |
| `database/migrations/000022_canonical_quantity_units.up.sql` | Task canonical constraints and triggers | HIGH | preflight, four trigger functions, four triggers |
| `docs/design/DESIGN-005.md` | Recorded serving disposition | HIGH | canonical vocabulary and recipe boundary |
| `frontend/src/lib/api/generated.ts` | Generated substitution alias | HIGH | `SubstitutionUnit` |
| `scripts/generate-api-types.py` | Canonical OpenAPI enum/reference generation checks | HIGH | `generated_contract`, canonical path in `main` |

If any task-owned change cannot be distinguished reliably, stop and recommend `REJECTED`. The task-owned symbols were distinguishable; concurrent hunks were not reused as task evidence.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | One shared validator owns `g`, `ml`, `oz`, and `fl_oz`. | `units.go`, callers, duplicate-helper search, validator tests | PASS | `ValidateQuantityUnit` is the exact four-value allowlist; former duplicate predicates are absent; all changed Go boundaries route vocabulary checks through it directly or through `ConvertUnit`. |
| 2 | Retained `serving` is recipe/per-unit only with explicit conversion. | `DESIGN-005`, recipe validator/converter, integration tests | PASS | `serving` is accepted only by `ValidateRecipeIngredientUnit`; `ConvertRecipeServingToBase` selects grams for solids and milliliters for liquids using the matching persisted measure. |
| 3 | Saved-diet and substitution contracts do not accept `serving`. | OpenAPI, HTTP/service/repository tests, generated frontend type, substitution service | PASS | Public schemas and generated types contain only the four physical units; saved-diet, HTTP, and substitution paths reject `serving`. |
| 4 | Cross-basis, unsupported, and non-finite values fail at relevant boundaries. | Unit, service, repository, HTTP, SQL, and direct trigger checks | PASS | Solid/liquid mismatches and unsupported units fail in Go and SQL; `NaN`, both infinities, and conversion overflow fail in converters and recipe validation before lookup/persistence. |
| 5 | OpenAPI and generated frontend enums agree. | generator drift check, generator tests, generated output, Redocly lint | PASS | All three public fields reference `CanonicalQuantityUnit`; its enum and generated `CanonicalQuantityUnit`/`SubstitutionUnit` agree. |
| 6 | Unit, integration, contract-drift, requirement-traceability, and design-traceability checks pass. | Focused/full tests and validators | PASS | Focused and full backend tests, race, coverage, frontend tests/build/coverage, generator, OpenAPI, task-list, traceability, diff, and vulnerability checks passed. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | `ConvertUnit` | function | `backend/internal/repository/units.go:11` | modified and repaired | `quantityInMealBase`, `quantityInNutritionBasis`, `ingredientBasisQuantity`, `sourceBaseQuantity`, display converters | `TestConvertUnit`, `TestConvertUnitRejectsUnsupportedAndNegativeValues` |
| 2 | `ConvertRecipeServingToBase` | function | `backend/internal/repository/units.go:46` | added and repaired | `ingredientBasisQuantity` | `TestConvertRecipeServingToBase`, `TestConvertRecipeServingToBaseRejectsInvalidInput` |
| 3 | `ValidateQuantityUnit` | function | `backend/internal/repository/units.go:81` | added | `ConvertUnit`, service, HTTP, saved-diet, substitution boundaries | `TestValidateQuantityUnit` |
| 4 | `ValidateRecipeIngredientUnit` | function | `backend/internal/repository/units.go:92` | added | `ingredientBasisQuantity`, recipe repository validation | `TestValidateRecipeIngredientUnit` |
| 5 | `(*PostgresMealRepository).validateIngredients` | method | `backend/internal/repository/meal_repository.go:314` | modified and repaired | meal create/update validation | `TestPostgresMealRepositoryRejectsNonFiniteRecipeQuantities`, PostgreSQL validation tests |
| 6 | `ingredientBasisQuantity` | function | `backend/internal/repository/meal_repository.go:459` | modified and repaired | composite macro calculation and recipe validation | `TestPostgresMealRepositoryRejectsNonFiniteRecipeQuantities`, recipe integration tests |
| 7 | `normalizeRequest` | function | `backend/internal/dailydiet/service.go:342` | modified | daily-diet create/replace | service tests, `TestTask215SavedDietQuantityUnitBoundaries` |
| 8 | `validateDailyDietBodyMap` | function | `backend/internal/httpapi/daily_diet_controller.go:159` | modified | daily-diet HTTP middleware | `TestValidateDailyDietBodyMapUsesCanonicalQuantityUnits` |
| 9 | `validateSubstitutionUnit` | function | `backend/internal/httpapi/search_validation.go:390` | modified | substitution HTTP validation | existing HTTP validation coverage |
| 10 | `validateSavedDietInput` | function | `backend/internal/repository/user_data_repository.go:535` | modified | saved-diet create/replace | saved-diet repository tests |
| 11 | `validateDailyDietCreateResponse` | function | `backend/internal/repository/saved_diet_mutation_repository.go:232` | modified | durable create claim validation | daily-diet mutation tests |
| 12 | `sourceBaseQuantity` | function | `backend/internal/search/substitution_service.go:207` | modified | substitution source macro calculation | substitution regression tests |
| 13 | `generated_contract` | function | `scripts/generate-api-types.py:1342` | added | generator output and `--check` path | generator check and tests |
| 14 | canonical generation path in `main` | CLI logic | `scripts/generate-api-types.py:1365` | modified | OpenAPI drift and generated output | generator check and tests |
| 15 | `recipe_ingredients_unit_canonical` and migration preflight | SQL constraint/preflight | `database/migrations/000022_canonical_quantity_units.up.sql:3-44` | added | recipe and existing-row migration boundary | PostgreSQL recipe boundary test; direct migration run |
| 16 | `validate_saved_diet_entry_unit_basis` | SQL trigger function | `database/migrations/000022_canonical_quantity_units.up.sql:46-61` | added | saved-diet entry insert/update | PostgreSQL saved-diet test |
| 17 | `saved_diet_entry_unit_basis_trigger` | SQL trigger | `database/migrations/000022_canonical_quantity_units.up.sql:63-67` | added | saved-diet entry writes | PostgreSQL saved-diet test |
| 18 | `validate_saved_diet_meal_unit_basis` | SQL trigger function | `database/migrations/000022_canonical_quantity_units.up.sql:69-86` | added | meal physical-state updates | direct transactional trigger probe |
| 19 | `saved_diet_meal_unit_basis_trigger` | SQL trigger | `database/migrations/000022_canonical_quantity_units.up.sql:88-92` | added | meal physical-state updates | direct transactional trigger probe |
| 20 | `validate_recipe_ingredient_unit_basis` | SQL trigger function | `database/migrations/000022_canonical_quantity_units.up.sql:94-109` | added | recipe ingredient insert/update | PostgreSQL recipe boundary test |
| 21 | `recipe_ingredient_unit_basis_trigger` | SQL trigger | `database/migrations/000022_canonical_quantity_units.up.sql:111-115` | added | recipe ingredient writes | PostgreSQL recipe boundary test |
| 22 | `validate_food_recipe_unit_basis` | SQL trigger function | `database/migrations/000022_canonical_quantity_units.up.sql:117-134` | added | food physical-state updates | direct transactional trigger probe |
| 23 | `food_recipe_unit_basis_trigger` | SQL trigger | `database/migrations/000022_canonical_quantity_units.up.sql:136-140` | added | food physical-state updates | direct transactional trigger probe |
| 24 | `000022` rollback statements | SQL migration | `database/migrations/000022_canonical_quantity_units.down.sql:1-28` | added | migration downgrade | static inspection; migration runner exercised up/down by repository fixtures |
| 25 | `CanonicalQuantityUnit` schema and three references | OpenAPI contract | `api/openapi.yaml:1236-1238,1254,1276,1710` | modified | saved-diet and substitution clients/servers | generator check and Redocly lint |
| 26 | `SubstitutionUnit = CanonicalQuantityUnit` | generated TypeScript type | `frontend/src/lib/api/generated.ts:1035` | modified | frontend substitution request builders and controls | frontend tests/build |
| 27 | `TestTask215SavedDietQuantityUnitBoundaries` | test | `backend/internal/dailydiet/task215_units_test.go:11` | added | daily-diet service quantity boundary | self |
| 28 | `TestValidateDailyDietBodyMapUsesCanonicalQuantityUnits` | test | `backend/internal/httpapi/daily_diet_controller_test.go:297` | added | daily-diet HTTP boundary | self |
| 29 | `TestConvertUnit` | test | `backend/internal/repository/repository_test.go:419` | modified | conversion behavior | self |
| 30 | `TestConvertUnitRejectsUnsupportedAndNegativeValues` | test | `backend/internal/repository/repository_test.go:447` | modified and repaired | converter error paths | self |
| 31 | `TestValidateQuantityUnit` | test | `backend/internal/repository/repository_test.go:469` | added | canonical allowlist | self |
| 32 | `TestValidateRecipeIngredientUnit` | test | `backend/internal/repository/repository_test.go:482` | added | recipe context/unit basis | self |
| 33 | `TestConvertRecipeServingToBase` | test | `backend/internal/repository/repository_test.go:506` | added and modified | serving conversion normal/zero paths | self |
| 34 | `TestConvertRecipeServingToBaseRejectsInvalidInput` | test | `backend/internal/repository/repository_test.go:542` | added and repaired | serving conversion error paths | self |
| 35 | `TestPostgresMealRepositoryRejectsNonFiniteRecipeQuantities` | test | `backend/internal/repository/repository_test.go:579` | added | pre-lookup and direct recipe quantity validation | self |
| 36 | `TestPostgresSavedDietRepository` | integration test | `backend/internal/repository/postgres_repository_test.go:1326` | modified | saved-diet service/repository/database boundary | self |
| 37 | `TestPostgresMealRepositoryLiquidRecipeNormalizationAndUnits` | integration test | `backend/internal/repository/postgres_repository_test.go:2862` | modified | recipe conversion and database boundary | self |
| 38 | `TestSubstitutionServiceCachesRejectionsAndSkippedSources` | test | `backend/internal/search/substitution_service_test.go:287` | modified | substitution rejection/degradation behavior | self |
| 39 | `TestSubstitutionServiceFailureAndDegradationPaths` | test | `backend/internal/search/substitution_service_test.go:441` | modified | substitution conversion failure behavior | self |

```yaml
inventory_source_count: 39
audited_symbol_count: 39
inventory_complete: true
generated_groupings:
  - "CanonicalQuantityUnit and its three OpenAPI references are one source-of-truth contract unit; generated TypeScript is listed separately."
```

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| `ConvertUnit` | Converts only the four physical units; rejects negative and non-finite input and non-finite output. | Same-unit and four metric/imperial paths pass; unsupported, negative, `NaN`, infinities, and overflow return typed errors. | Pure; no shared state or cancellation. | Only allowlisted strings reach arithmetic; no SQL or command boundary. | Constant-time arithmetic with no I/O. | Small typed repository helper; shared validator removes duplicate vocabulary. | Tests cover normal values, all invalid classes, and overflow; callers propagate conversion errors except display-only conversion helpers whose source fields are repository-validated. | PASS |
| `ConvertRecipeServingToBase` | Recipe-only serving maps solid to grams and liquid to milliliters using the matching positive measure. | Zero servings succeeds; negative, invalid state, absent/non-finite measures, non-finite servings, and overflow fail. | Pure; no shared state or cancellation. | Internal recipe boundary; no user data execution. | Constant-time arithmetic. | Context-specific name prevents public `serving` leakage. | Table tests cover both states, zero, all non-finite inputs including unused measure fields, missing measures, and overflow. | PASS |
| `ValidateQuantityUnit` | Exact allowlist is `g`, `ml`, `oz`, `fl_oz`; `serving` is excluded. | Empty, aliases, `cup`, and `serving` fail with unit-conversion kind. | Pure. | Safe string comparison. | Constant-time switch. | One shared predicate with minimal API. | Direct accepted/rejected table is complete for vocabulary. | PASS |
| `ValidateRecipeIngredientUnit` | Physical units must match solid/liquid state; recipe `serving` is allowed for either state. | Invalid state, unknown, and cross-basis values fail; matching physical units and serving pass. | Pure. | Context boundary is explicit and isolated from public contracts. | Constant-time. | Reuses shared physical validator and canonical predicate. | Tests cover all accepted bases, serving in both states, cross-basis, and unknown unit. | PASS |
| `(*PostgresMealRepository).validateIngredients` | Recipe quantities must be positive finite and units valid for the looked-up food. | Nil food ID and non-finite values fail before lookup; lookup errors and unit/state errors propagate. | Uses caller context for food lookup; no goroutines or leaked rows. | Validates before persistence and before arbitrary food-derived values enter conversion. | One food lookup per ingredient, existing bounded recipe list. | Clear validation ordering. | New test proves `NaN`, positive infinity, and negative infinity do not reach fake lookup; integration tests cover normal/cross-basis paths. | PASS |
| `ingredientBasisQuantity` | Returns a finite macro-basis quantity for a valid recipe ingredient. | Direct non-finite quantity fails validation; physical units convert; serving delegates to recipe converter; invalid state/unit fails. | Pure; no resources or cancellation. | Repository boundary revalidates state and unit independently of callers. | Constant-time conversion. | Explicit switch is readable and non-duplicative. | Direct non-finite test plus liquid/imperial/serving integration cases; negative direct helper input is guarded by its production caller and database check. | PASS |
| `normalizeRequest` | Saved-diet API entries require finite positive quantities and canonical units. | Rejects malformed IDs, quantities, positions, duplicates, and `serving`; accepts four units. | Pure; bounded maps only. | User input is normalized before repository/solver use. | O(n) bounded by 100 entries. | Shared validator call replaces local predicate. | Existing service tests plus task-specific basis test; quantity finiteness is explicitly checked. | PASS |
| `validateDailyDietBodyMap` | HTTP JSON shape and unit vocabulary match the public daily-diet contract. | Wrong types, invalid UUIDs, non-finite/non-positive/out-of-range quantities, duplicate positions, and unsupported units fail. | Pure; no resources. | JSON is rejected before controller dispatch. | Bounded by 100 entries. | Reuses repository vocabulary without broadening API. | New test covers all four accepted units and rejects `serving` and `cup`; existing coverage exercises malformed fields. | PASS |
| `validateSubstitutionUnit` | Public substitution units are canonical and security-normalized. | Normalized valid units pass; aliases, `serving`, and unknown values fail. | Pure; no resources. | Security normalization precedes shared allowlist. | Constant-time after normalization. | Minimal adapter from HTTP error contract to shared policy. | Existing HTTP coverage rejects `serving`, aliases, and invalid types. | PASS |
| `validateSavedDietInput` | Persistence input has authenticated identity and canonical positive finite entries. | Nil IDs, non-finite/non-positive quantities, and `serving` fail before transaction; valid entries continue. | Pure validation before transaction. | Final repository input boundary plus SQL constraints/triggers. | O(n) entries. | Shared predicate removes obsolete alias. | PostgreSQL repository tests cover serving, zero, cross-basis, and valid metric/imperial entries. | PASS |
| `validateDailyDietCreateResponse` | Immutable create response contains finite positive canonical entries and valid macros. | Invalid IDs, units, quantities, positions, duplicates, and non-finite/negative macros fail. | Pure; no resources. | Protects durable response decoding and publication. | O(n) bounded entries. | Exact response validator is reused by claim/decode paths. | Mutation tests and current full suite cover response validation; changed unit branch is covered. | PASS |
| `sourceBaseQuantity` | Substitution input converts to the food's metric macro basis and never accepts public `serving`. | Canonical unit validation precedes conversion; solid/liquid base selection and conversion errors are returned. | Pure; caller owns repository lookup and context. | Service revalidates beyond HTTP, preventing forged internal `serving`. | Constant-time. | Clear separation between lookup and conversion. | Substitution rejection, skipped-source, and degradation tests pass; non-finite conversion is rejected by `ConvertUnit`. | PASS |
| `generated_contract` | OpenAPI has exactly three canonical public references and the exact four-value enum. | Missing refs/schema or changed enum raises `ValueError`; generated TypeScript is rendered from validated source. | Pure source transformation; caller owns file I/O. | Contract source is checked before client generation. | One bounded source scan and replacement. | Explicit fail-closed contract check. | Current generator check and five tests pass; failure branches are statically inspected. | PASS |
| canonical generation path in `main` | Generator refuses required-marker, response-drift, canonical-schema, or generated-output drift. | `--check` compares current output; normal mode writes only generated contract; failures return nonzero. | CLI-local file I/O; no cancellation requirement. | Prevents stale or broadened client contract. | Single source/output read and comparison. | Idiomatic small CLI path. | Generator check/tests and frontend API check pass. | PASS |
| `recipe_ingredients_unit_canonical` and migration preflight | Recipe storage allows four physical units plus recipe `serving`, while existing cross-basis rows block migration. | Invalid existing solid/liquid rows raise `23514`; constraint rejects unsupported future units. | Migration runner supplies transaction; no runtime resource leak. | Database is final direct-write boundary. | Bounded catalog scans and one check constraint. | Parentheses/SQL precedence inspected. | PostgreSQL recipe test and migration runner pass. | PASS |
| `validate_saved_diet_entry_unit_basis` | Entry unit matches referenced meal state. | Solid accepts `g`/`oz`; liquid accepts `ml`/`fl_oz`; mismatches raise `23514`. | Row trigger is transaction-local. | Protects direct SQL inserts/updates. | One indexed meal lookup. | Clear trigger function. | Repository integration tests cover direct insert and service path. | PASS |
| `saved_diet_entry_unit_basis_trigger` | Entry validator fires on insert and meal/unit updates. | Relevant writes are intercepted; unrelated columns do not broaden the trigger scope. | Database-owned lifecycle. | Final persistence enforcement. | One trigger call per changed row. | Idiomatic `BEFORE` trigger. | Direct insert test plus SQL inspection. | PASS |
| `validate_saved_diet_meal_unit_basis` | A meal state change cannot invalidate existing saved-diet units. | Opposite-state transition raises `23514`; compatible state remains valid. | Transaction-local `EXISTS` query; state update remains unchanged on error. | Protects state mutation boundary. | One relationship query. | Correct trigger-level invariant. | Direct transactional probe verified rejection and unchanged state. | PASS |
| `saved_diet_meal_unit_basis_trigger` | Meal physical-state updates invoke the saved-diet validator. | Fires only for `physical_state` updates and blocks incompatible rows. | Database-owned lifecycle. | Prevents persisted cross-basis state. | One trigger invocation. | Minimal trigger declaration. | Direct transactional probe. | PASS |
| `validate_recipe_ingredient_unit_basis` | Recipe unit matches referenced food state, with serving allowed for recipes. | Solid/liquid cross-basis and unknown units fail; valid serving/physical units pass. | Transaction-local food lookup. | Protects direct recipe writes. | One food lookup. | Mirrors Go policy at final boundary. | PostgreSQL recipe integration and migration inspection. | PASS |
| `recipe_ingredient_unit_basis_trigger` | Recipe validator fires on ingredient insert and food/unit changes. | Relevant writes are intercepted before commit. | Database-owned lifecycle. | Prevents bypass through direct SQL. | One trigger per row. | Idiomatic trigger declaration. | Direct invalid inserts and SQL inspection. | PASS |
| `validate_food_recipe_unit_basis` | Food state changes cannot invalidate attached recipe units. | Opposite-state transition raises `23514`; compatible state remains valid. | Transaction-local `EXISTS` query; rejected update leaves state unchanged. | Protects food physical-state mutation. | One relationship query. | Correct reciprocal invariant. | Direct transactional probe verified rejection and unchanged state. | PASS |
| `food_recipe_unit_basis_trigger` | Food physical-state updates invoke recipe-state validation. | Fires only on `physical_state` updates and blocks invalidating transitions. | Database-owned lifecycle. | Final direct-write boundary. | One trigger invocation. | Minimal trigger declaration. | Direct transactional probe. | PASS |
| `000022` rollback statements | Downgrade removes task triggers/functions and restores prior recipe non-blank constraint. | `IF EXISTS`/`IF NOT EXISTS` guards support expected repeated migration states. | Migration runner owns transaction. | Schema-only boundary. | Constant catalog operations. | Symmetric, minimal rollback. | Migration runner exercised down/up during repository fixtures; static inspection complete. | PASS |
| `CanonicalQuantityUnit` schema and three references | OpenAPI source declares exactly four public units and reuses one schema. | Saved-diet request/response and substitution inputs cannot advertise `serving`. | Static contract. | Client/server boundary is explicit. | No runtime cost. | Reusable schema avoids drift. | Redocly lint and generator checks pass with only accepted pre-existing OAuth warning. | PASS |
| `SubstitutionUnit = CanonicalQuantityUnit` | Frontend substitution input type is exactly the generated public vocabulary. | Type alias cannot widen independently from canonical generated type. | Type-only; no runtime state. | Client contract mirrors server source. | No runtime cost. | Minimal alias. | Full frontend tests, coverage, and build pass. | PASS |
| `TestTask215SavedDietQuantityUnitBoundaries` | Service helper accepts state-compatible canonical units only. | Covers solid/liquid accepted units and serving/cross-basis/unknown rejection. | No production resources. | Exercises service boundary. | Small table. | Focused traceable test. | Covers canonical service behavior; non-finite conversion is tested at repository boundary. | PASS |
| `TestValidateDailyDietBodyMapUsesCanonicalQuantityUnits` | HTTP map validation uses one canonical vocabulary. | All four units pass; `serving` and `cup` fail. | No resources. | Exercises user JSON boundary. | Bounded one-entry cases. | Direct focused assertions. | Adversarial public-unit cases are explicit. | PASS |
| `TestConvertUnit` | Supported conversions and same-unit behavior remain stable. | Metric/imperial directions and rounding pass. | No resources. | No external boundary. | Small table. | Appropriate unit test. | Normal paths complement invalid/repair test. | PASS |
| `TestConvertUnitRejectsUnsupportedAndNegativeValues` | Converter rejects all malformed numeric/unit inputs. | Covers negative, unsupported, `NaN`, both infinities, and overflow. | No resources. | No external boundary. | Small table. | Table-driven adversarial test. | Directly closes prior important finding. | PASS |
| `TestValidateQuantityUnit` | Exact canonical allowlist is regression-protected. | Four accepted values and `serving`/unknown/empty rejected. | No resources. | Safe strings only. | Constant-time. | Minimal test. | Vocabulary coverage complete. | PASS |
| `TestValidateRecipeIngredientUnit` | Recipe-only context rules remain explicit. | Covers physical state, serving, matching, cross-basis, and unknown values. | No resources. | Safe strings only. | Small table. | Table-driven. | Context boundary adversarial cases covered. | PASS |
| `TestConvertRecipeServingToBase` | Solid/liquid serving conversion and zero-serving semantics remain stable. | Positive and zero serving cases return expected finite base quantity/unit. | No resources. | No external boundary. | Small table. | Focused conversion test. | Both physical bases covered. | PASS |
| `TestConvertRecipeServingToBaseRejectsInvalidInput` | Recipe converter rejects malformed metadata and arithmetic. | Covers negative, missing, non-finite servings/measures, unused non-finite fields, bad state, and overflow. | No resources. | No external boundary. | Table-driven. | Directly proves repaired numeric boundary. | All prior finding triggers are represented. | PASS |
| `TestPostgresMealRepositoryRejectsNonFiniteRecipeQuantities` | Recipe validation rejects non-finite quantities before lookup and direct conversion. | `NaN`, positive infinity, and negative infinity return validation kind in both paths. | Fake executor confirms early path; no I/O occurs. | Persistence boundary is challenged. | Three values, bounded. | Focused regression test. | Covers the repaired caller and helper independently. | PASS |
| `TestPostgresSavedDietRepository` | Saved-diet repository and SQL enforce state-compatible canonical units. | Valid metric/imperial entries pass; serving, cross-basis service input, and direct SQL insert fail. | PostgreSQL fixture resets migrations and cleans up. | Direct SQL bypass is challenged. | Small fixture. | Relevant integration test. | Does not need non-finite SQL because repair is Go pre-persistence validation. | PASS |
| `TestPostgresMealRepositoryLiquidRecipeNormalizationAndUnits` | Recipe conversion and SQL state/unit rules agree. | Liquid `ml`, `fl_oz`, and `serving` normalize; solid/liquid mismatches and direct unsupported inserts fail. | PostgreSQL fixture manages migration/test database lifecycle. | Direct recipe writes are challenged. | Small fixture. | Good end-to-end recipe boundary test. | Update trigger behavior additionally probed transactionally. | PASS |
| `TestSubstitutionServiceCachesRejectionsAndSkippedSources` | Canonical source-unit rejection preserves substitution degradation behavior. | Rejected/skipped source paths and cache behavior remain safe. | Existing fixture lifecycle passes. | Service remains defensive after HTTP. | Bounded fixture. | Regression-focused. | Updated away from public `serving`; existing failure paths pass. | PASS |
| `TestSubstitutionServiceFailureAndDegradationPaths` | Unsupported source units become safe conversion degradation, not success. | `serving` is rejected and warning/rejection behavior remains bounded. | Existing fixture lifecycle passes. | No unsafe output crosses search response. | Bounded fixture. | Clear regression coverage. | Explicit unsupported-unit case added. | PASS |

Mandatory audit conclusion: all changed non-trivial units have explicit malformed-input, error-path, state/resource, security, performance, API, and test coverage assessments above. No cancellation or concurrency behavior was introduced by the task-owned unit changes.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| optional | `database/migrations/000022_canonical_quantity_units.up.sql:69-140` | Physical-state update triggers | The repository does not currently contain a permanent integration test for both reciprocal state-update triggers. | A direct transactional `psql` probe created valid saved-diet/recipe rows, attempted incompatible meal/food state updates, observed `23514`, and verified state remained unchanged. | No repair required for acceptance; add permanent regression coverage in a future migration-test refinement. |

```yaml
blocking_findings: 0
important_findings: 0
optional_findings: 1
```

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| `git rev-parse HEAD`, `git merge-base`, task-owned diff/status discovery | repository root | 0 | PASS | Baseline exactly matched `a4e31367485b03269e90b5607f2057c9568bb5b1`; scope reconstructed from preparation manifest and current diff. |
| `go test ./internal/repository -run 'Test(ConvertUnit\|ValidateQuantityUnit\|ValidateRecipeIngredientUnit\|ConvertRecipeServingToBase\|PostgresMealRepositoryRejectsNonFiniteRecipeQuantities\|PostgresSavedDietRepository\|PostgresMealRepositoryLiquidRecipeNormalizationAndUnits)$' -count=1` | `backend` | 0 | PASS | Focused converter, repair, saved-diet, and recipe tests. |
| `go test ./internal/dailydiet ./internal/httpapi ./internal/search -count=1` | `backend` | 0 | PASS | Changed service, HTTP, and substitution packages passed. |
| `go test ./... -count=1` with Redis URL | `backend` | 0 | PASS | Full backend suite passed sequentially. |
| `go test -race ./... -count=1` with Redis URL | `backend` | 0 | PASS | Full backend race suite passed sequentially. |
| `go vet ./...` | `backend` | 0 | PASS | Static analysis passed. |
| `go test ./internal/... -coverprofile=/tmp/task-215-backend-coverage-rereview.out && go tool cover -func=...` | `backend` | 0 | PASS | 88.0% aggregate backend statements; all four `units.go` functions and `validateIngredients` were 100%; report at `/tmp/task-215-backend-coverage-rereview.out`. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `backend` | 0 | PASS | No reachable vulnerabilities found. |
| `go run ./cmd/migrate up` | `backend` | 0 | PASS | Prepared local database for direct migration-boundary probe. |
| Direct transactional `psql` probe of meal and food physical-state update triggers | repository root | 0 | PASS | Both incompatible updates raised `23514`; both states remained unchanged; transaction rolled back. |
| `python3 scripts/generate-api-types.py --check` | repository root | 0 | PASS | Generated API types current. |
| `python3 scripts/test_generate_api_types.py` | repository root | 0 | PASS | Five generator tests passed. |
| `npx --no-install redocly lint api/openapi.yaml` | repository root | 0 | PASS | Valid; one pre-existing accepted OAuth callback `302`-only warning. |
| `bun run check:api-types` | `frontend` | 0 | PASS | Generator drift check passed. |
| `bun test` | `frontend` | 0 | PASS | 364 tests passed, 0 failed. |
| `bun test --coverage` | `frontend` | 0 | PASS | 92.11% line coverage; existing phase exceptions only. |
| `bun run build` | `frontend` | 0 | PASS | Vite production build passed. |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS | 237 sequential tasks validated; no task status was edited. |
| `python3 scripts/validate-traceability.py` | repository root | 0 | PASS | Design traceability passed. |
| `git diff --check a4e31367485b03269e90b5607f2057c9568bb5b1 -- <task-owned tracked paths>` | repository root | 0 | PASS | No whitespace errors. |

## 9. Files Inspected and Staleness Fingerprints

All 20 current task preparation, implementation, contract, migration, design, generated, and test files reviewed for this decision were hashed after the audit.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| `api/openapi.yaml` | Public canonical unit contract | None | SHA-256 | `6368ee9c1321104d0e645ed8a3e6b73f8f14c1a161835a5f388a0e6e2fa3da4a` |
| `backend/internal/dailydiet/service.go` | Daily-diet normalization | None | SHA-256 | `191c17f3cdc84dacf03a0c3007ea29adbfd3c02b05a0396f533d37ebc6820d6c` |
| `backend/internal/dailydiet/task215_units_test.go` | Task service boundary test | None | SHA-256 | `bc701f8c767c45aa5a6af877e0bedbd61fd62cf5071d86328462523927cb003f` |
| `backend/internal/httpapi/daily_diet_controller.go` | Daily-diet HTTP validation | None | SHA-256 | `d2e0e9346968402a2453012857c15432b363b9c8aef68d00d5349cb43aaf7dea` |
| `backend/internal/httpapi/daily_diet_controller_test.go` | HTTP unit boundary test | None | SHA-256 | `34485172a7e6ada598863b6ff01e7c3e8ffcc290bc44961866e629bbcb40d777` |
| `backend/internal/httpapi/search_validation.go` | Substitution HTTP validation | None | SHA-256 | `1c01b989d2d469425f945592c0e65cd76c1c5d9d35bede4b8ff4720760029b` |
| `backend/internal/repository/meal_repository.go` | Recipe numeric and unit validation | None | SHA-256 | `0219f44cc3dc10350ec72ebeed1e338e314c233d5d7283bd53dff9112e43fddc` |
| `backend/internal/repository/postgres_repository_test.go` | PostgreSQL boundary tests | Optional coverage gap | SHA-256 | `6f363df8b0c0d4af71559dbc8baefa1a05128512f6d360c41c329a2ee9b8eaba` |
| `backend/internal/repository/repository_test.go` | Converter/validator tests | None | SHA-256 | `5d3cdae031d3dae35ede6f2659fa8e26433ed1b5faef7dd3998d36a8a031369b` |
| `backend/internal/repository/saved_diet_mutation_repository.go` | Durable response unit validation hunk | None | SHA-256 | `aa8fb95cad4b611bbabbf533d9731a7ae595305420777952b7bd93fcc3229c78` |
| `backend/internal/repository/units.go` | Shared unit conversion boundary | None | SHA-256 | `9d9a8296654cc4b57e13bfb0090f15dced673d85782abc39675d8e2967463127` |
| `backend/internal/repository/user_data_repository.go` | Saved-diet input validation | None | SHA-256 | `41bf37f97e5dfb35b5a79620452e360b346e3d3368a15358301145765054651e` |
| `backend/internal/search/substitution_service.go` | Substitution conversion | None | SHA-256 | `f576300c020175a7678c9747c03ad01afc383445cf38244ed984932efd4d3805` |
| `backend/internal/search/substitution_service_test.go` | Substitution regression tests | None | SHA-256 | `8e75154447f2a43751379ecd58b95fa722f9d2aa3e89c484009664f7d8ac2b14` |
| `database/migrations/000022_canonical_quantity_units.down.sql` | Migration rollback | None | SHA-256 | `5ec614ac457fb671dd3ff0c55ca8d74962c4025095f7a1dc5f0c797029d78fc9` |
| `database/migrations/000022_canonical_quantity_units.up.sql` | Database unit/state boundaries | Optional coverage gap | SHA-256 | `e93dfe861e244f46a707600e87d89fbc9b680e55e5e9c9ec3aa4fefc8c1469ba1f1` |
| `docs/design/DESIGN-005.md` | Product/design serving disposition | None | SHA-256 | `bf206afdc0b9a35c56e7114192fdc5b5e1bbbd8441fa5c2efddef13f6ec9d6b0` |
| `frontend/src/lib/api/generated.ts` | Generated frontend units | None | SHA-256 | `361ce14d3cde8ae90afe0bc074ffb3e301c751a8a9603fc7a300e7d6b49cf20b` |
| `scripts/generate-api-types.py` | OpenAPI-to-TypeScript generator | None | SHA-256 | `b6b241afb9c13c4206e30eb1e12ae4adcd32d6734cc065b33ece5d47ed4f4e87` |
| `docs/implementation/preparation/task-215-preparation.md` | Current preparation and repair evidence | Prior evidence superseded where hashes differ | SHA-256 | `b59e6de8ea56809c7d33ab57e36bb6719aa9f8bbc1cbdbbeccfac8d18a8829cf` |

```yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "docs/implementation/reviews/task-215-review.md was rejected on the pre-repair hashes and is overwritten by this current evidence."
  - "Prior task-213 and task-214 evidence was not reused for overlapping files; current contents and task-owned symbols were audited independently."
```

## 10. Coverage and Exceptions

- [x] Required backend and frontend coverage commands ran.
- [x] Report paths and observed thresholds are recorded.
- [x] Untested branches relevant to changed symbols were inspected.
- [x] Exceptions exactly match the task row; no task-specific exception was introduced.

```yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "/tmp/task-215-backend-coverage-rereview.out; frontend bun test --coverage output"
observed_line_coverage: "88.0% backend aggregate; 92.11% frontend; task-changed units.go and validateIngredients 100%"
coverage_passed: true
```

Coverage finding: aggregate percentages are below the repository's aspirational 100% target because existing Phase 07 exceptions remain active. The changed conversion, shared validators, recipe finite guard, and response validator were directly covered; the optional SQL update-trigger gap is explicitly recorded rather than hidden as a coverage exception.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced by task 215.
- [x] No source-of-truth documentation was contradicted; the serving disposition is recorded in DESIGN-005.
- [x] No generated/cache/build/temporary artifact was unintentionally added; Vite output remained ignored and no status change was introduced by review.
- [x] Public API additions are necessary and used; `SubstitutionUnit` is a generated alias, not a new runtime API.
- [x] Duplicate helpers and obsolete aliases were searched for; `validUnit`, `validSavedDietUnit`, and `ConvertServingToBase` are absent from executable code.
- [x] Error, malformed-input, non-finite, overflow, cross-basis, direct-SQL, cleanup, and race paths were challenged.

Findings: The repaired numeric boundary now fails closed for `NaN`, positive/negative infinity, and conversion overflow. Canonical public units remain exactly `g`, `ml`, `oz`, and `fl_oz`; `serving` remains confined to recipe conversion. The only remaining observation is optional permanent coverage for state-update triggers; direct transactional behavior passed.

## 12. Decision

A task may be `PASSED` only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains.

```yaml
decision: "PASSED"
reason: "The NaN/Inf repair is verified at both converters and recipe validation, all canonical unit boundaries agree, and no blocking or important finding remains."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "None; optionally add permanent integration assertions for both physical-state update triggers."
```

## 13. Repair Context

Not applicable for a passed re-review. The prior important non-finite-conversion finding was independently verified as repaired; no further repair context is required.
