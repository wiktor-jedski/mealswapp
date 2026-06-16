# Task 127 Review: Phase 04 Search API Routes

Recommended status: PASSED

## Scope

Reviewed exactly task 127 from `docs/implementation/02_TASK_LIST.md`.

Task row status verified as `PREPARED`.

Dependencies verified:

- Task 93: `PASSED`
- Task 115: `PASSED`
- Task 125: `PASSED`
- Task 126: `PASSED`

Implementation files inspected:

- `backend/internal/httpapi/search_controller.go`
- `backend/internal/httpapi/search_controller_test.go`
- `backend/internal/app/app.go`
- Supporting route/auth/validation wiring in `backend/internal/httpapi/router.go`, `auth_middleware.go`, and `search_validation.go`
- Design source `docs/design/DESIGN-002.md`

## Verification Criteria Checklist

- [x] `/api/v1/search` is exposed through the existing Fiber route gateway via `SearchController.Routes()` and versioned route registration.
- [x] `/api/v1/search/autocomplete` is exposed when an autocomplete service is configured.
- [x] Anonymous catalog search access is covered by HTTP tests.
- [x] Authenticated catalog search access is covered by HTTP tests.
- [x] Authenticated successful searches append history only after a valid response.
- [x] Anonymous, rejected, and failed searches do not append history.
- [x] Search history uses the server-derived JWT user ID and ignores client-supplied `userId`.
- [x] Autocomplete responses are returned in the shared envelope and include cache metadata when available.
- [x] Autocomplete receives optional auth context when valid JWT cookies are present.
- [x] Search route validation is attached before handler dispatch.
- [x] Route definitions include rate-limit metadata for search and autocomplete.
- [x] Search responses include cache metadata when the service response provides it.
- [x] Response envelopes include request IDs on successful and error responses.
- [x] Metrics are verified for the autocomplete route.
- [x] Search `POST` declares an explicit CSRF exemption for the documented GET/POST-safe public search policy.
- [x] Production app wiring registers search, autocomplete, cache, and history dependencies through the existing app composition path.

## Tests Run

```text
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi ./internal/app
```

Result: passed.

```text
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/search ./internal/httpapi ./internal/app
```

Result: passed.

```text
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/...
```

Result: failed in the known repository deleted-row test:

```text
--- FAIL: TestPostgresFoodItemRepositorySearch
postgres_repository_test.go:1040: Search() deleted exclusion total=1 ... want none
```

This matches the preparation report's known broad-suite failure and is outside task 127's route-controller scope.

## Findings

No task-127 blocking findings.

## Recommendation

Mark task 127 as `PASSED`.
