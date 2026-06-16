# Task 117 Review

Recommended status: PASSED

## Scope

Reviewed exactly task 117:

`Phase 04 Dietary Presets and Filter Processing` / `DESIGN-002: FilterProcessor`

Source-of-truth verification criteria:

> Unit and repository-backed tests verify preset expansion, include and exclude filter translation, contradictory filter rejection, Exclusion Rule conflict rejection with 422 `SearchRejection`, allergen exclusion behavior, and no misleading classification rows are created for dietary presets.

## Status and Dependency Check

- Task 117 status in `docs/implementation/02_TASK_LIST.md`: `PREPARED`.
- Dependency 34 status: `PASSED`.
- Dependency 116 status: `PASSED`.

## Files Inspected

- `docs/implementation/02_TASK_LIST.md`
- `docs/design/DESIGN-002.md`
- `backend/internal/search/contracts.go`
- `backend/internal/search/parser.go`
- `backend/internal/search/filter_processor.go`
- `backend/internal/search/filter_processor_test.go`
- `backend/internal/search/filter_processor_integration_test.go`
- `backend/internal/repository/types.go`
- `backend/internal/repository/food_repository.go`
- `backend/internal/repository/sql/food_search.sql`
- `backend/internal/repository/sql/food_search_count.sql`
- `backend/internal/repository/postgres_repository_test.go`
- `database/migrations/000016_food_item_allergens.up.sql`
- `database/migrations/000016_food_item_allergens.down.sql`

## Verification Run

Commands run:

```sh
rg -n "\| (34|116|117) \|" docs/implementation/02_TASK_LIST.md
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/search -run 'TestApplyFilters' -count=1
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/search -count=1
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/repository -run 'TestPostgresFoodItemRepository' -count=1
```

Results:

- Task/dependency status check passed.
- `go test ./internal/search -run 'TestApplyFilters' -count=1` passed.
- `go test ./internal/search -count=1` passed, covering the repository-backed filter processor integration test.
- `go test ./internal/repository -run 'TestPostgresFoodItemRepository' -count=1` failed in `TestPostgresFoodItemRepositorySearch` at `backend/internal/repository/postgres_repository_test.go:937`: after deleting Apple, the test still sees Apple Juice for `Name: "Ap"`. This matches the repair report's existing deleted-row expectation and is not caused by dietary presets or allergen filtering.

## Checklist

- Preset expansion: PASS. `filter_processor.go` defines named preset bundles and `TestApplyFiltersExpandsDietaryPresetsToExclusionRules` verifies expansion.
- Include and exclude filter translation: PASS. `TestApplyFiltersTranslatesIncludeAndExcludeFilters` covers category, role, object type, and allergen include/exclude query translation.
- Contradictory filter rejection: PASS. `TestApplyFiltersRejectsContradictoryFilters` returns `SearchRejection{Code: "rejected_search", Field: "filters"}`.
- Exclusion Rule conflict rejection with 422 `SearchRejection`: PASS for task 117 scope. The filter processor returns the `rejected_search` `SearchRejection`; HTTP 422 mapping is owned by later SearchController/service tasks in the task list.
- Allergen exclusion behavior: PASS. Named allergen preset rules now populate `RepositoryQuery.ExcludedAllergenKeys`; repository SQL applies `food_item_allergens` include/exclude key filters.
- No misleading classification rows for dietary presets: PASS. The integration test checks Food Category and Culinary Role listings and fails if `dairy_free` appears as a classification row.
- Repository-backed tests: PASS for task 117 criteria. `TestApplyFiltersDietaryPresetExcludesRepositoryAllergenKeys` seeds an allowed food and a dairy-tagged food, applies `dairy_free`, and verifies only the allowed food remains.

## Notes

The repaired implementation addresses the prior no-op named allergen issue by adding `AllergenKeys` and `ExcludedAllergenKeys` to `RepositoryQuery`, adding `food_item_allergens`, and filtering those keys in `food_search.sql` and `food_search_count.sql`.

Residual risk: direct UUID allergen filters still use the classification join path, while named preset allergens use `food_item_allergens`. That is acceptable for this task's repaired criteria because the dietary preset repository-backed behavior is now covered, and no misleading preset classification rows are created.

## Recommendation

Mark task 117 as `PASSED`.
