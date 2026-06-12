# Phase 04 UAT: Search, Similarity, Cache Core

## Scope

Phase 04 covers tasks `115`-`133`. The backend now exposes search input
normalization, search DTO parsing, dietary-preset expansion, autocomplete
ranking, Redis-backed search/autocomplete/similarity caching, macro similarity
ranking, similarity indicator metadata, catalog search, substitution search,
Daily Daily Diet Alternative Search request-shape handling, search error
mapping, authenticated search-history persistence, search routes, OpenAPI
contracts, generated frontend API types, production bootstrap composition, and
aggregate gate evidence.

The implemented public API surface is `/api/v1/search` and
`/api/v1/search/autocomplete`. Search is optionally authenticated: anonymous
users can search without history persistence, while authenticated successful
searches append encrypted user-scoped history after valid results return.
Daily-diet alternative requests are accepted at the API boundary in Phase 04,
but Phase 07 saved-diet/LP optimization data is not implemented yet, so missing
daily-diet data returns a deterministic `SearchRejection`.

## Automated Evidence

Run from the repository root unless noted:

```sh
gofmt -l $(find backend -name '*.go' -not -path '*/.go-cache/*' -not -path '*/.go-mod-cache/*')
python3 scripts/validate-task-list.py
python3 scripts/validate-traceability.py
npx --no-install redocly lint api/openapi.yaml
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run generate:api-types
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check:api-types
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi/... ./internal/search/...
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./cmd/... ./internal/app/... ./internal/httpapi/...
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/... -count=1 -coverprofile=coverage.out
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go tool cover -func=coverage.out
python3 scripts/check.py
```

Observed results from task reviews `128`-`132`:

- Task-list validation passed with `133` sequential tasks and ordered
  dependencies.
- Traceability validation passed.
- Redocly lint passed; the OpenAPI description is valid with one explicitly
  ignored existing OAuth redirect warning.
- Frontend API type generation produced `frontend/src/lib/api/generated.ts`,
  `check:api-types` passed, frontend tests passed with `9 pass`, and frontend
  build passed.
- Focused backend search workflow tests passed for catalog search, substitution
  similarity sorting, autocomplete determinism, cache hit/miss behavior,
  authenticated history persistence, anonymous no-history behavior, daily-diet
  structured rejection, OpenAPI lint, and generated-type drift checks.
- API bootstrap tests passed for `./cmd/...`, `./internal/app/...`, and
  `./internal/httpapi/...`; smoke verification showed `/ready` returning
  PostgreSQL and Redis readiness, `/api/v1/search` returning cache metadata with
  namespace `search`, schema version `search-response-v1`, and TTL `300`, and
  autocomplete returning namespace `autocomplete`, schema version
  `autocomplete-response-v1`, and TTL `120`.
- `go vet ./...` passed.
- Backend internal coverage passed with the accepted Phase 04 deviation:
  aggregate coverage `88.5%`, `backend/internal/repository` `95.8%`,
  `backend/internal/search` `89.3%`, and `backend/internal/cache` `83.3%`.
- `python3 scripts/check.py` passed after sequential execution, including local
  stack verification, Phase 02/03 UAT checks, frontend screenshot verification,
  backend tests, `go test -race ./...`, backend coverage, frontend build,
  generated API type drift check, frontend tests, and frontend coverage.

Task `133` preparation additionally ran:

```sh
python3 scripts/validate-traceability.py
```

Result: passed.

## Project-Owner Checks

1. Start dependencies with `bash scripts/start-services.sh`, then run
   `cd backend && go run ./cmd/migrate up && go run ./cmd/api`.
2. Catalog search: POST `/api/v1/search` with
   `{"query":"milk","mode":"catalog","page":1,"filters":[]}` and confirm a
   `200` envelope with at most `10` items, deterministic `page` and
   `totalCount`, no `rejection`, and `cache.status` present.
3. Repeat the same catalog search and confirm cache metadata changes to, or
   remains observably compatible with, a Redis cache-hit path without changing
   the result shape.
4. Substitution search: POST `/api/v1/search` with `mode:"substitution"` and
   one or more `substitutionInputs`; confirm returned items include sorted
   descending similarity scores and similarity metadata for tier/color/image
   where available.
5. Autocomplete: GET `/api/v1/search/autocomplete?query=milk`; confirm exact
   matches rank ahead of fuzzy matches, equal scores remain deterministic across
   repeated calls, and the response includes autocomplete cache metadata.
6. Authenticated history: register/login, perform a successful catalog search,
   then GET `/api/v1/search-history`; confirm the completed search appears for
   the authenticated account and duplicate searches are retained.
7. Anonymous no-history behavior: perform the same search without auth cookies,
   then login as a test user and confirm the anonymous query was not appended to
   that user's history.
8. Clear-history behavior: with auth cookies and CSRF token, DELETE
   `/api/v1/search-history`; confirm the list endpoint returns an empty history
   for that user.
9. Daily-diet rejection: POST `/api/v1/search` with
   `{"query":"lentil","mode":"daily_diet_alternative","page":1,"dailyDietId":"<uuid>","filters":[]}`
   before Phase 07 data exists; confirm `422`, rejection code
   `phase_07_saved_diet_unavailable`, field `dailyDietId`, no worker/job side
   effects, and no history append.
10. Invalid request handling: submit malformed filters, invalid pages, empty
    queries, and bad substitution quantities; confirm structured `400` errors
    include request IDs and do not echo raw rejected query text in logs.

## Traceability

Primary design sources:

- `docs/design/DESIGN-002.md`: SearchController, AutocompleteRanker,
  QueryParser, PaginationHandler, FilterProcessor, and CulinaryRoleWeighter.
- `docs/design/DESIGN-003.md`: CosineSimilarityCalculator,
  MacroVectorNormalizer, ThresholdFilter, SimilarityIndicatorMapper, and
  SimilarityAssetResolver.
- `docs/design/DESIGN-008.md`: SearchHistoryRepository and user-scoped
  authenticated history behavior.
- `docs/design/DESIGN-011.md`: RedisCache, cache key schema versions, TTLs,
  cache hit/miss metadata, and fallback behavior.
- `docs/design/DESIGN-014.md`: MetricsCollector, aggregate checks, local stack
  verification, and accepted coverage evidence.
- `docs/design/DESIGN-017.md`: ErrorMessageMapper and stable search error
  envelopes.

Related Phase 04 task IDs:

- `115` search request normalization.
- `116` search contracts and query parser.
- `117` dietary presets and filter processing.
- `118` autocomplete ranking.
- `119` Redis search cache core.
- `120` macro similarity calculator.
- `121` similarity indicators and assets.
- `122` catalog search service.
- `123` substitution search service.
- `124` daily-diet alternative request shape.
- `125` search degradation and error mapping.
- `126` authenticated search-history persistence.
- `127` search API routes.
- `128` search OpenAPI contract.
- `129` frontend search contract generation.
- `130` API bootstrap composition.
- `131` search workflow integration gate.
- `132` coverage and aggregate gate.
- `133` acceptance documentation.

## Known Deviations

- Phase 04 backend internal coverage is accepted at `88.5%` and documented in
  `docs/implementation/04_OPEN.md`. The aggregate gate accepts this only because
  the documented deviation is present. This is an aggregate-gate exception, not
  a blanket carry-forward of the Phase 03 coverage exception; Task 134 must audit
  the remaining uncovered lines and close or justify active behavior gaps.
  Frontend coverage remains `100%`.
- Concrete search boundary limits are implementation assumptions: search
  queries are limited to `200` runes, autocomplete queries to `120` runes, and
  page values to a maximum of `10000`. The project owner should adjust these
  before public launch if product UX requires different bounds.
- Redis cache schema versions and TTLs are concrete Phase 04 choices:
  `search-response-v1` with `300` seconds, `autocomplete-response-v1` with
  `120` seconds, and `similarity-calculation-v1` with `900` seconds.
- Dietary Presets are backend-owned bundles that expand into Exclusion Rules;
  they are not persisted as Food Category or Culinary Role classification rows.
- Daily-diet alternative search is only a request shape in Phase 04. Full
  saved-diet persistence and LP optimization remain Phase 07 work.
- Substitution response DTOs keep final ranking scores in `SimilarityScores`;
  full tier/color/image metadata is verified internally until the API response
  field design is expanded.

## Acceptance

Accept Phase 04 after the automated evidence remains green and the
project-owner checks confirm catalog search, substitution ranking,
autocomplete determinism, Redis cache metadata, authenticated search-history
persistence, anonymous no-history behavior, clear-history behavior, structured
daily-diet rejection, and stable validation/error envelopes.
