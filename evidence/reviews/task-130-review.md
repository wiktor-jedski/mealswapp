# Task 130 Review: Phase 04 API Bootstrap Composition

## Recommendation

PASSED

## Scope

Reviewed exactly task 130 from `docs/implementation/02_TASK_LIST.md`.

Task row under review:

`| 130 | Phase 04 API Bootstrap Composition | DESIGN-010: RouteHandler | PREPARED | 0 | Phase 04: wire search repositories, search services, similarity services, Redis cache adapters, optional authentication, history persistence, and search controllers into the real API bootstrap. | 119,127 | None | ... |`

## Status And Dependency Checks

- Task 130 status is `PREPARED`.
- Dependency 119 status is `PASSED`.
- Dependency 127 status is `PASSED`.
- No task-list status was edited during this review.

## Implementation Inspection

Inspected `backend/internal/app/app.go` and `backend/internal/app/app_test.go`.

Observed production bootstrap composition in `app.NewProduction`:

- `repository.NewPostgresFoodItemRepository(pg)` is wired as the search food repository.
- `repository.NewPostgresMealRepository(pg)` is wired into autocomplete ranking.
- Redis is optional: when `redisClient != nil`, `cache.GoRedisStore` and `cache.SearchResponseStore` are composed.
- Search route service uses `search.NewSearchDispatcher(...)`.
- Catalog mode is handled by `search.NewCatalogService(foodRepo, searchResponseCache)`.
- Substitution mode is handled by `search.NewSubstitutionService(foodRepo, searchResponseCache)`.
- Autocomplete is exposed through `searchAutocompleteAdapter`, wrapping `search.NewAutocompleteService(foodRepo, mealRepo)` with optional Redis cache metadata.
- `httpapi.NewSearchController(...)` is registered with `.WithAutocompleteService(...)` and `.WithSearchHistoryAppender(userDataService)`.
- HTTP dependencies include optional JWT auth via `httpapi.NewJWTAuthenticator(...)`, structured logs and metrics via the JSON sink, readiness checks for PostgreSQL and Redis, CSRF manager, and composed routes.

Inspected task-specific tests added in `backend/internal/app/app_test.go`:

- `TestNewProductionExposesProductionRoutes` verifies production composition exposes existing routes plus `/api/v1/search` and `/api/v1/search/autocomplete`.
- `TestNewProductionSearchRouteDispatchesSubstitutionPastCatalog` verifies substitution requests dispatch past the catalog-only path and reach substitution validation/service behavior.

## Verification Performed

Required Go test slice:

```sh
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./cmd/... ./internal/app/... ./internal/httpapi/...
```

Result: PASS

Relevant output:

```text
?    github.com/wiktor-jedski/mealswapp/backend/cmd/api      [no test files]
?    github.com/wiktor-jedski/mealswapp/backend/cmd/migrate  [no test files]
?    github.com/wiktor-jedski/mealswapp/backend/cmd/seed     [no test files]
?    github.com/wiktor-jedski/mealswapp/backend/cmd/worker   [no test files]
ok   github.com/wiktor-jedski/mealswapp/backend/internal/app
ok   github.com/wiktor-jedski/mealswapp/backend/internal/httpapi
```

Practical smoke verification:

```sh
bash scripts/start-services.sh
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/migrate up
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/api
```

Probes:

```sh
curl -sS -i http://127.0.0.1:8080/ready
curl -sS -i -X POST http://127.0.0.1:8080/api/v1/search -H 'Content-Type: application/json' --data '{"query":"milk","mode":"catalog","page":1,"filters":[]}'
curl -sS -i 'http://127.0.0.1:8080/api/v1/search/autocomplete?query=milk'
```

Smoke results:

- `/ready` returned `200 OK` with `postgres: ok` and `redis: ok`.
- `/api/v1/search` returned `200 OK` with a normal response envelope and search cache metadata: namespace `search`, schema version `search-response-v1`, status `hit`, TTL `300`.
- `/api/v1/search/autocomplete?query=milk` returned `200 OK` with a normal response envelope and autocomplete cache metadata: namespace `autocomplete`, schema version `autocomplete-response-v1`, status `miss`, TTL `120`.
- The API process was stopped after the smoke probes.

## Checklist

- [x] Verified task 130 status is `PREPARED`.
- [x] Verified dependencies 119 and 127 are `PASSED`.
- [x] Inspected production bootstrap composition for search repositories, services, Redis cache adapters, optional auth, history persistence, and controllers.
- [x] Ran the required Go test command successfully.
- [x] Started PostgreSQL and Redis services.
- [x] Ran backend migrations.
- [x] Smoke-ran `go run ./cmd/api`.
- [x] Verified `/ready` returns 200 with PostgreSQL and Redis readiness.
- [x] Verified `/api/v1/search` returns 200 through the real API bootstrap.
- [x] Verified `/api/v1/search/autocomplete` returns 200 through the real API bootstrap.

## Repair Instructions

None. Task 130 satisfies its verification criteria.
