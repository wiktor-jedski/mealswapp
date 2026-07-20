# Task 215 Preparation Evidence

## Assignment and Disposition

- Task: `215` — Phase 07.01 Canonical Quantity Unit Boundaries
- Design source: `docs/design/DESIGN-005.md`, static aspect `UnitConverter`
- Requirement source: `SW-REQ-036` per-unit calculation
- Baseline commit: `a4e31367485b03269e90b5607f2057c9568bb5b1`
- Baseline confidence: **High**. At inspection, the supplied baseline and working tree showed only the declared pre-existing edit to `docs/implementation/02_TASK_LIST.md` and the declared untracked `review.txt`. Neither path was edited by this task.

The recorded disposition retains `serving` only as the internal recipe-ingredient token implementing SW-REQ-036's user-facing “per unit” calculation. Solid recipe servings convert through `averageUnitWeightGrams`; liquid recipe servings convert through `averageServingVolumeMilliliters`. The shared physical quantity vocabulary is exactly `g`, `ml`, `oz`, and `fl_oz`; saved-diet and substitution contracts do not accept `serving`.

## Prepared Change

- Added one shared Go validator, `repository.ValidateQuantityUnit`, and routed unit conversion, saved-diet service/repository validation, HTTP request validation, substitution service validation, and immutable saved-diet response validation through it.
- Replaced the generic serving converter with the recipe-specific `ConvertRecipeServingToBase` boundary and added `ValidateRecipeIngredientUnit` for physical-state compatibility.
- Removed internal substitution-service acceptance of `serving`.
- Recorded the product/design disposition in DESIGN-005.
- Added migration 22 with canonical recipe-unit constraints plus saved-diet/recipe physical-state triggers, including checks that prevent physical-state updates from invalidating persisted quantities.
- Reused OpenAPI `CanonicalQuantityUnit` for substitution inputs; generated frontend `SubstitutionUnit` is now an alias of `CanonicalQuantityUnit`. The generator reads and validates the OpenAPI enum and requires all three saved-diet/substitution fields to reference it.

### Exact task-attributable changed paths

- `api/openapi.yaml`
- `backend/internal/dailydiet/service.go`
- `backend/internal/dailydiet/task215_units_test.go`
- `backend/internal/httpapi/daily_diet_controller.go`
- `backend/internal/httpapi/daily_diet_controller_test.go`
- `backend/internal/httpapi/search_validation.go`
- `backend/internal/repository/meal_repository.go`
- `backend/internal/repository/postgres_repository_test.go`
- `backend/internal/repository/repository_test.go`
- `backend/internal/repository/saved_diet_mutation_repository.go`
- `backend/internal/repository/units.go`
- `backend/internal/repository/user_data_repository.go`
- `backend/internal/search/substitution_service.go`
- `backend/internal/search/substitution_service_test.go`
- `database/migrations/000022_canonical_quantity_units.down.sql`
- `database/migrations/000022_canonical_quantity_units.up.sql`
- `docs/design/DESIGN-005.md`
- `frontend/src/lib/api/generated.ts`
- `scripts/generate-api-types.py`
- `docs/implementation/preparation/task-215-preparation.md`

Concurrent task agents also changed some overlapping files after task 215 began. Only the quantity-unit changes described here belong to task 215; unrelated response-matrix, repository-surface, and durable-idempotency diffs are excluded from this inventory.

### Added executable symbols

- Go production: `repository.ValidateQuantityUnit`, `repository.ValidateRecipeIngredientUnit`, `repository.ConvertRecipeServingToBase`
- Go tests: `dailydiet.TestTask215SavedDietQuantityUnitBoundaries`, `httpapi.TestValidateDailyDietBodyMapUsesCanonicalQuantityUnits`, `repository.TestValidateQuantityUnit`, `repository.TestValidateRecipeIngredientUnit`, `repository.TestConvertRecipeServingToBase`, `repository.TestConvertRecipeServingToBaseRejectsInvalidInput`
- Python: `generated_contract`
- PostgreSQL: `validate_saved_diet_entry_unit_basis`, `validate_saved_diet_meal_unit_basis`, `validate_recipe_ingredient_unit_basis`, `validate_food_recipe_unit_basis`

### Modified executable symbols

- Go production: `repository.ConvertUnit`, `repository.ingredientBasisQuantity`, `repository.validateSavedDietInput`, `repository.validateDailyDietCreateResponse`, `dailydiet.normalizeRequest`, `httpapi.validateDailyDietBodyMap`, `httpapi.validateSubstitutionUnit`, `search.sourceBaseQuantity`
- Go tests: `repository.TestConvertUnit`, `repository.TestConvertUnitRejectsUnsupportedAndNegativeValues`, `repository.TestPostgresSavedDietRepository`, `repository.TestPostgresMealRepositoryLiquidRecipeNormalizationAndUnits`, `search.TestSubstitutionServiceCachesRejectionsAndSkippedSources`, `search.TestSubstitutionServiceFailureAndDegradationPaths`
- Python: `main` in `scripts/generate-api-types.py`

### Removed/replaced executable symbols

- `repository.ConvertServingToBase` was replaced by `repository.ConvertRecipeServingToBase`.
- The duplicate predicates `repository.validUnit`, `repository.validSavedDietUnit`, and `dailydiet.validUnit` were removed.

No frontend runtime function was added or modified; `SubstitutionUnit` is a generated type alias. Migration trigger names are `saved_diet_entry_unit_basis_trigger`, `saved_diet_meal_unit_basis_trigger`, `recipe_ingredient_unit_basis_trigger`, and `food_recipe_unit_basis_trigger`.

## Verification Evidence

| Command | Result |
| --- | --- |
| Focused `go test` for repository validator/converter tests, substitution service tests, and HTTP unit validation tests | PASS. |
| `go test ./internal/dailydiet -run Task215` | PASS. |
| `go build ./internal/repository ./internal/dailydiet ./internal/search ./internal/httpapi` | PASS: all changed production packages compile. |
| `go test ./internal/dailydiet ./internal/search ./internal/httpapi` | PASS. |
| `go test ./internal/repository -run 'TestPostgres(SavedDietRepository\|MealRepositoryLiquidRecipeNormalizationAndUnits)$'` | PASS: focused saved-diet and recipe PostgreSQL integration tests completed in 2.644s. |
| `go test ./...` from `backend/` | PASS: all backend packages passed; repository completed in 51.153s. |
| Temporary PostgreSQL database; apply all migrations; insert valid physical and recipe-serving quantities; assert saved-diet `ml`/`serving` for a solid meal and recipe `ml`/`cup` for a solid ingredient raise check violations | PASS: `task-215 database boundaries passed`; temporary database was dropped. |
| `python3 scripts/generate-api-types.py --check` | PASS: generated API types are current. |
| `python3 scripts/test_generate_api_types.py` | PASS: 2 tests. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS with the pre-existing accepted OAuth `302`-only warning. |
| `bun run check:api-types` and focused `bun test src/lib/units.test.ts src/lib/components/SubstitutionRequest.test.ts` | PASS: generated drift check and 9 frontend tests. |
| `bun test` from `frontend/` | PASS: 364 tests, 0 failures. |
| `validate_requirements()` from `scripts/check.py` | PASS: 91/91 requirements traced. |
| `python3 scripts/validate-task-list.py` | PASS: 237 sequential tasks; no status was edited. |
| `git diff --check` | PASS. |
| `python3 scripts/validate-traceability.py` | BLOCKED by concurrent task-216 additions in `backend/internal/repository/saved_diet_mutation_repository.go` that currently lack required adjacent design/Go Doc comments. No task-215 source is reported. |

## Verification Criteria

| Criterion | Satisfied | Evidence |
| --- | --- | --- |
| One shared validator owns `g`, `ml`, `oz`, and `fl_oz` | Yes | `ValidateQuantityUnit` is the sole Go vocabulary predicate and every changed Go boundary calls it. |
| Retained `serving` is recipe/per-unit only and converts explicitly | Yes | DESIGN-005 disposition, `ValidateRecipeIngredientUnit`, `ConvertRecipeServingToBase`, recipe integration fixtures, and direct database checks. |
| Saved-diet and substitution contracts are not broadened | Yes | OpenAPI remains the four physical units; service, repository, HTTP, generated frontend, and substitution service reject `serving`. |
| Cross-basis and unsupported values fail at service, repository, HTTP, and database boundaries | Yes for implemented behavior | Focused service/HTTP/repository tests and isolated PostgreSQL checks cover unsupported and cross-basis values. HTTP vocabulary rejection occurs before dispatch; physical-state compatibility is enforced by the service and database because the HTTP DTO does not contain trusted meal state. |
| OpenAPI/generated frontend enums agree | Yes | One OpenAPI schema is referenced by all three fields; generated `SubstitutionUnit` aliases `CanonicalQuantityUnit`; lint and drift checks pass. |
| Unit, integration, contract-drift, requirement-traceability, and design-traceability tests pass | **Not all whole-worktree gates currently pass** | Task-215 unit/integration tests, full backend tests, contract drift, OpenAPI, frontend, and 91/91 requirement checks pass. Whole-worktree design traceability remains blocked by concurrent task-216 comments identified above. |

Therefore the functional task-215 implementation and backend aggregation are complete, but **not every aggregate verification criterion can honestly be marked satisfied in the current shared worktree**. Rerun `python3 scripts/validate-traceability.py` after the concurrent task-216 preparation adds its required trace comments.

## Reviewer-Finding Repair — Non-Finite Unit Conversion

Repair date: `2026-07-14`.

The important finding in `docs/implementation/reviews/task-215-review.md` is repaired. Unit and recipe-serving conversion now reject `NaN`, positive infinity, and negative infinity with `ErrorKindUnitConversion`; conversion overflow is rejected instead of returning infinity. Recipe ingredient validation rejects non-finite quantities with the existing `ErrorKindValidation` contract before food lookup or persistence. Canonical `g`, `ml`, `oz`, and `fl_oz` conversions, solid/liquid serving selection, and zero-serving results remain unchanged. The optional migration-trigger integration coverage was not added because it is independent of the numeric repair.

### Exact repair symbols

- Modified production symbols:
  - `repository.ConvertUnit` in `backend/internal/repository/units.go`: validates finite input and finite conversion output.
  - `repository.ConvertRecipeServingToBase` in `backend/internal/repository/units.go`: validates finite servings and both serving measures, and rejects non-finite multiplication results.
  - `repository.(*PostgresMealRepository).validateIngredients` in `backend/internal/repository/meal_repository.go`: rejects non-finite recipe quantities before repository lookup/persistence.
  - `repository.ingredientBasisQuantity` in `backend/internal/repository/meal_repository.go`: independently rejects non-finite recipe quantities on direct conversion paths.
- Modified tests:
  - `repository.TestConvertUnitRejectsUnsupportedAndNegativeValues`: added `NaN`, `+Inf`, `-Inf`, and overflow rejection coverage.
  - `repository.TestConvertRecipeServingToBase`: added zero-serving solid/liquid boundary coverage.
  - `repository.TestConvertRecipeServingToBaseRejectsInvalidInput`: added non-finite servings, solid/liquid measures, unused measures, and overflow coverage.
- Added test:
  - `repository.TestPostgresMealRepositoryRejectsNonFiniteRecipeQuantities`: verifies repository validation and direct recipe-basis conversion reject `NaN`, `+Inf`, and `-Inf` with `ErrorKindValidation`.

No task status, migration, OpenAPI contract, frontend source, or unrelated task code was edited for this repair.

### Repair verification

| Command | Result |
| --- | --- |
| Initial focused regression run for converter and recipe repository non-finite cases | RED as expected: `ConvertUnit` and `ConvertRecipeServingToBase` returned nil errors for `NaN`/`+Inf`; recipe validation bypass reached the fake repository lookup and panicked. |
| `go test ./internal/repository -run 'Test(ConvertUnit\|ConvertRecipeServingToBase\|PostgresMealRepositoryRejectsNonFiniteRecipeQuantities)$' -count=1` | PASS after repair. |
| `go test ./internal/repository -run 'Test(ValidateQuantityUnit\|ValidateRecipeIngredientUnit\|ConvertRecipeServingToBase\|ConvertUnit\|PostgresMealRepositoryRejectsNonFiniteRecipeQuantities\|PostgresSavedDietRepository\|PostgresMealRepositoryLiquidRecipeNormalizationAndUnits)$' -count=1` | PASS, including PostgreSQL-backed canonical saved-diet and recipe boundaries. |
| Focused task-215 tests in `internal/dailydiet`, `internal/httpapi`, and `internal/search` | PASS. |
| `gofmt -d internal/repository/units.go internal/repository/meal_repository.go internal/repository/repository_test.go` | PASS: no output. |
| Frontend generated-type check, focused unit/component tests, and production build | PASS: generated types current, 9 tests passed, Vite build passed. |
| `python3 scripts/generate-api-types.py --check` and `python3 scripts/test_generate_api_types.py` | PASS: generated types current and 5 generator tests passed. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS with the pre-existing accepted OAuth callback `302`-only warning. |
| `python3 scripts/validate-task-list.py` and `python3 scripts/validate-traceability.py` | PASS: 237 sequential tasks and design traceability valid; no task status was edited. |
| `git diff --check` | PASS. |

### Post-repair SHA-256 fingerprints

| File | SHA-256 |
| --- | --- |
| `backend/internal/repository/units.go` | `9d9a8296654cc4b57e13bfb0090f15dced673d85782abc39675d8e2967463127` |
| `backend/internal/repository/meal_repository.go` | `0219f44cc3dc10350ec72ebeed1e338e314c233d5d7283bd53dff9112e43fddc` |
| `backend/internal/repository/repository_test.go` | `5d3cdae031d3dae35ede6f2659fa8e26433ed1b5faef7dd3998d36a8a031369b` |

## Shared-Workspace Scope Note

During implementation, concurrent agents prepared tasks 213, 214, and 216 in the same worktree. Task 215 did not edit `docs/implementation/02_TASK_LIST.md`, `review.txt`, task statuses, or unrelated concurrent changes. The task-215 migration was moved to version 22 after the concurrent durable-idempotency migration took version 21.
