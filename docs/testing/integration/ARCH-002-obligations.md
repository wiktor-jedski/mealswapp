# ARCH-002 Integration Verification Obligations

## Purpose

This document defines the SWE.5 integration verification obligations for architecture component ARCH-002, the Search Module.

The goal is to verify that SearchController, QueryParser, FilterProcessor, Catalog Search, Substitution Search, AutocompleteRanker, Redis cache integration, repository access, similarity ranking, authenticated history, and API gateway routing collaborate according to the architecture.

## Component Information

| Field | Value |
| --- | --- |
| Architecture Component | ARCH-002 |
| Name | Search Module |
| Source Documents | `docs/architecture/ARCH-002.md`, `docs/design/DESIGN-002.md` |
| Related Units | SearchController, SearchDispatcher, CatalogService, SubstitutionService, AutocompleteService, FilterProcessor, RedisCache, FoodItemRepository, MealRepository, SearchHistoryRepository |
| Collaborating Architecture | ARCH-003, ARCH-005, ARCH-008, ARCH-010, ARCH-011 |
| Related Requirements | SW-REQ-004, SW-REQ-010, SW-REQ-017, SW-REQ-019, SW-REQ-024, SW-REQ-026, SW-REQ-029, SW-REQ-031 |

## IT-ARCH-002-001 Catalog Search Gateway, Cache, Repository, and History Flow

### Intent

Verify that an authenticated catalog search passes through the API gateway into the Search Module, retrieves repository results, writes cache metadata, appends server-derived search history, and later serves a cache hit without writing anonymous history.

### System Under Test

ARCH-002 Search Module, centered on SearchController and SearchDispatcher.

### Real Components

- SearchController
- SearchDispatcher
- CatalogService
- FilterProcessor
- Redis-compatible search cache test double at the ARCH-011 boundary
- Food repository test double preserving repository semantics at the ARCH-005 boundary
- SearchHistoryRepository-compatible appender at the ARCH-008 boundary
- Fiber router and optional-auth middleware from ARCH-010

### Allowed Test Doubles

- Repository/cache/history test doubles may be used to observe architecture side effects without requiring full PostgreSQL/Redis setup.

### Trigger / Stimulus

Authenticated and anonymous `POST /api/v1/search` catalog requests with normalized search input.

### Expected Integrated Behavior

1. Gateway validation accepts the request and derives user identity from JWT cookies when present.
2. SearchDispatcher routes to CatalogService.
3. CatalogService misses cache, reads repository candidates, sorts deterministic results, and writes cache metadata.
4. SearchController appends history only for the authenticated successful response using the server-derived user ID.
5. A subsequent anonymous request returns cached data and does not append history.

### Required Evidence

- Test verifies HTTP response, repository calls, cache miss/hit side effects, cache metadata, authenticated history append, and anonymous no-history behavior.
- Test traceability comment references `IT-ARCH-002-001`, `ARCH-002`, and related SW requirements.

### Requirement Traceability

- SW-REQ-004
- SW-REQ-010
- SW-REQ-019
- SW-REQ-024
- SW-REQ-029

### Verification Status

Implemented by `backend/internal/httpapi/search_controller_test.go::TestSearchWorkflowIntegrationGateCatalogCacheHistoryAndDailyDiet`.

Status: PASS.

## IT-ARCH-002-002 Substitution Search Dispatch, Similarity, and Ranking Flow

### Intent

Verify that a substitution-mode search passes through the real route and dispatcher into SubstitutionService, combines source inputs, invokes similarity behavior, preserves tier metadata, and sorts results by final score.

### System Under Test

ARCH-002 Search Module, centered on SearchDispatcher and SubstitutionService.

### Real Components

- SearchController
- SearchDispatcher
- SubstitutionService
- Similarity calculator from ARCH-003
- Repository-compatible food source/candidate provider at the ARCH-005 boundary
- Fiber router from ARCH-010

### Allowed Test Doubles

- Food repository test double may provide deterministic macro and culinary-role fixtures.

### Trigger / Stimulus

`POST /api/v1/search` substitution request with one Substitution Input.

### Expected Integrated Behavior

1. Gateway validation accepts the substitution input.
2. SearchDispatcher routes to SubstitutionService instead of CatalogService.
3. SubstitutionService combines source macro profile and ranks target candidates through ARCH-003 similarity.
4. Single-input Culinary Role weighting is applied where roles match.
5. Response exposes ordered items, final scores, and similarity tier metadata.

### Required Evidence

- Test verifies route-level substitution dispatch, result ordering by similarity/final score, and response metadata.
- Test traceability comment references `IT-ARCH-002-002`, `ARCH-002`, `ARCH-003`, and related SW requirements.

### Requirement Traceability

- SW-REQ-017
- SW-REQ-026
- SW-REQ-031

### Verification Status

Implemented by:

- `backend/internal/httpapi/search_controller_test.go::TestSearchWorkflowIntegrationGateSubstitutionSortsBySimilarity`
- `backend/internal/httpapi/search_controller_test.go::TestSearchControllerRealRouteSubstitutionDispatchesToSubstitutionService`

Status: PASS.

## IT-ARCH-002-003 Daily Diet Alternative Boundary and No-Side-Effects Rejection

### Intent

Verify that Phase 04 supports the Daily Diet Alternative request shape at the Search Module boundary without invoking Phase 07 optimization, worker jobs, cache writes, repository reads, or history persistence when saved-diet data is unavailable.

### System Under Test

ARCH-002 Search Module, centered on QueryParser, SearchController, and CatalogService daily-diet boundary handling.

### Real Components

- SearchController
- QueryParser
- CatalogService daily-diet preparation path
- Fiber router from ARCH-010

### Allowed Test Doubles

- Counting repository/cache/history test doubles may be used to verify absence of side effects.

### Trigger / Stimulus

`POST /api/v1/search` request with mode `daily_diet_alternative` and a valid `dailyDietId`.

### Expected Integrated Behavior

1. Gateway validation accepts the request shape.
2. QueryParser selects Daily Diet Alternative strategy.
3. Phase 07 saved-diet unavailability is returned as deterministic `SearchRejection`.
4. Repository, cache, worker/job, and history side effects do not occur.

### Required Evidence

- Test verifies 422 response, rejection code/field, and zero side effects.
- Test traceability comment references `IT-ARCH-002-003`, `ARCH-002`, and related SW requirements.

### Requirement Traceability

- SW-REQ-024
- SW-REQ-029

### Verification Status

Implemented by:

- `backend/internal/httpapi/search_controller_test.go::TestSearchWorkflowIntegrationGateCatalogCacheHistoryAndDailyDiet`
- `backend/internal/httpapi/search_controller_test.go::TestSearchControllerProductionPathDailyDietUnavailableReturns422WithoutSideEffects`

Status: PASS.

## IT-ARCH-002-004 Autocomplete Repository-to-Ranking Integration

### Intent

Verify that autocomplete uses real food and meal repositories, excludes deleted rows, preserves SQL parameterization for special characters, and ranks candidates by exact match, Levenshtein distance, length, and stable tie-breakers.

### System Under Test

ARCH-002 Search Module, centered on AutocompleteService and AutocompleteRanker.

### Real Components

- AutocompleteService
- AutocompleteRanker
- PostgresFoodItemRepository from ARCH-005
- PostgresMealRepository from ARCH-005
- PostgreSQL migration-backed schema

### Allowed Test Doubles

- None for repository behavior; PostgreSQL is used when available.

### Trigger / Stimulus

Autocomplete queries against seeded food and meal rows, including deleted rows and special-character input.

### Expected Integrated Behavior

1. Real repositories retrieve candidate names from migrated tables.
2. Deleted food and meal rows are excluded even when repository context includes deleted rows.
3. Special-character input is parameterized safely and does not return injected rows.
4. Results are deterministic and page bounded.

### Required Evidence

- Repository-backed integration test verifies ranking order, deleted-row exclusion, parameterized special-character behavior, and deterministic repeated calls.
- Test traceability comment references `IT-ARCH-002-004`, `ARCH-002`, `ARCH-005`, and related SW requirements.

### Requirement Traceability

- SW-REQ-004
- SW-REQ-010
- SW-REQ-019

### Verification Status

Implemented by `backend/internal/search/autocomplete_integration_test.go::TestAutocompleteServiceUsesRealRepositoriesForRankingAndSafety`.

Status: PASS.

## SWE.5 Checklist Evaluation

| Obligation | ARCH Trace | SW-REQ Trace | Collaborating Units | Real Components Practical | Behavior Type | Test Evidence | Decision |
| --- | --- | --- | --- | --- | --- | --- | --- |
| IT-ARCH-002-001 | PASS | PASS | PASS | PASS | Data flow, sequence, side effects | PASS | PASS |
| IT-ARCH-002-002 | PASS | PASS | PASS | PASS | Dispatch, ranking, data flow | PASS | PASS |
| IT-ARCH-002-003 | PASS | PASS | PASS | PASS | Failure handling, no-side-effects | PASS | PASS |
| IT-ARCH-002-004 | PASS | PASS | PASS | PASS | Repository-to-ranking integration | PASS | PASS |

## Verification Commands

The Phase 04 SWE.5 evidence is covered by:

```bash
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi/... ./internal/search/...
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/search -run TestAutocompleteServiceUsesRealRepositoriesForRankingAndSafety -count=1
python3 scripts/validate-task-list.py
python3 scripts/validate-traceability.py
```

## Completion Decision

SWE.5 coverage for ARCH-002 is complete for the Phase 04 scope.

All obligations are implemented, tests contain traceability comments, practical verification passes, and no ARCH-002 Phase 04 obligation remains uncovered.
