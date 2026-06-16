# Task 125 Review Evidence

## Task

| ID | Title | Design | Selected Status |
| --- | --- | --- | --- |
| 125 | Phase 04 Search Degradation and Error Mapping | DESIGN-017: ErrorMessageMapper | PREPARED |

## Dependency Check

- Task 119: PASSED
- Task 122: PASSED
- Task 123: PASSED
- Task 124: PASSED

Selected status and dependencies satisfy the review preconditions.

## Files Inspected

- `docs/implementation/02_TASK_LIST.md`
- `docs/design/DESIGN-017.md`
- `backend/internal/search/errors.go`
- `backend/internal/search/contracts.go`
- `backend/internal/search/catalog_service.go`
- `backend/internal/search/substitution_service.go`
- `backend/internal/search/catalog_service_test.go`
- `backend/internal/httpapi/search_controller.go`
- `backend/internal/httpapi/search_controller_test.go`
- `backend/internal/httpapi/router.go`
- `backend/internal/httpapi/search_validation.go`

## Verification Commands

```bash
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/search ./internal/httpapi
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/cache
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./internal/search ./internal/httpapi ./internal/cache
```

Results:

- `go test ./internal/search ./internal/httpapi`: PASSED
- `go test ./internal/cache`: PASSED
- `go vet ./internal/search ./internal/httpapi ./internal/cache`: PASSED

## Criteria Checklist

- 400 malformed requests: PASS. `ValidateJSON` maps malformed JSON to `invalid_json`; HTTP test covers status, stable code, request IDs, and no service dispatch.
- 422 rejected searches: PASS. `SearchController.Search` serializes `SearchRejection` as a validation-category `AppError` with rejection data; HTTP test covers code/category/field.
- 503 similarity unavailable for substitution-only failures: PASS. `SubstitutionService` wraps macro comparison failures in `SimilarityUnavailableError`; controller maps it to retryable dependency `similarity_unavailable`; HTTP test verifies frontend-compatible code/category/retryable flag.
- Cache-unavailable warnings on catalog fallback: PASS. `CatalogService` treats Redis get/set errors as `cache_unavailable` warnings while still loading repository results; service and HTTP tests cover fallback and warning propagation.
- Request IDs on errors: PASS. `writeError` and the 422 rejection branch include request IDs in envelopes and error metadata; tests cover malformed request and dependency error request IDs.
- No raw query leakage in logs: PASS. HTTP request logs contain route/method/status/latency metadata only; tests assert raw search query text does not appear in captured logs.
- Frontend-compatible error codes: PASS. Observed stable codes include `invalid_json`, `validation_failed`, `rejected_search`, `dependency_unavailable`, and `similarity_unavailable`.
- Successful catalog fallback behavior preserved: PASS. Cache failures do not prevent repository search, and successful responses retain `ok` status with warnings.

## Findings

No blocking findings for task 125.

## Recommended Status

PASSED
