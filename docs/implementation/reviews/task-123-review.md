# Task 123 Re-Review - Phase 04 Substitution Search Service

Recommended status: PASSED

## Gate Checks

- Task 123 status verified as `PREPARED` in `docs/implementation/02_TASK_LIST.md`.
- Dependencies verified in `docs/implementation/02_TASK_LIST.md`:
  - 117: `PASSED`
  - 119: `PASSED`
  - 120: `PASSED`
  - 121: `PASSED`
- Scope respected: no task-list status or implementation code was edited.
- Re-review focused on the repaired task-123 route/service surface and the previous rejection about production `/api/v1/search` not reaching `SubstitutionService`.

## Practical Verification

- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/search ./internal/httpapi ./internal/app` passed.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/search ./internal/httpapi ./internal/app` passed.

## Checklist

- PASS: Multi-input macro combination is implemented and covered by `TestSubstitutionServiceCombinesMultipleInputsWithoutPerInputCulinaryOrdering`.
- PASS: Single-input Culinary Role weighting uses `finalScore = similarityScore * (1 + 0.2 * tagMatchCount)` and is covered by `TestSubstitutionServiceAppliesSingleInputCulinaryRoleWeightThresholdWarningsAndTieSort`.
- PASS: Multi-input searches do not apply per-input Culinary Role ordering; role weighting is only enabled when `len(req.SubstitutionInputs) == 1`.
- PASS: Threshold filtering is exercised through `CompareMacros` diagnostics and substitution service tests.
- PASS: Tier metadata is preserved and serialized through `SimilarityMetadata`.
- PASS: Graceful diagnostic warnings are returned for skipped source loads and skipped targets.
- PASS: Deterministic sorting is implemented by final score, then candidate name, then ID.
- PASS: Previous rejection is repaired. Production `NewProduction` now composes `NewSearchController(search.NewSearchDispatcher(search.NewCatalogService(...), search.NewSubstitutionService(...)))`, so `/api/v1/search` substitution requests are no longer forced through `CatalogService` only.
- PASS: Repair tests cover dispatcher routing, real HTTP route substitution dispatch, and production composition reaching substitution-specific validation instead of catalog-only rejection.

## Review Notes

`backend/internal/search/dispatcher.go` resolves the parsed search strategy and delegates substitution strategies to the substitution service while preserving catalog/default behavior. `backend/internal/app/app.go` now wires that dispatcher into the production search controller with a shared PostgreSQL food repository and optional Redis-backed search response cache.

The prior failure mode is covered by `TestNewProductionSearchRouteDispatchesSubstitutionPastCatalog`: a production `/api/v1/search` substitution request returns a substitution-input rejection from `SubstitutionService` because the fake production repository cannot load the source, and it specifically does not return the previous catalog-only mode rejection. The HTTP integration test `TestSearchControllerRealRouteSubstitutionDispatchesToSubstitutionService` also verifies the catalog service is not called for a valid substitution request and that similarity metadata is returned.

No blocking findings remain for task 123.
