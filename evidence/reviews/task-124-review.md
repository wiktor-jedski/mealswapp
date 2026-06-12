# Task 124 Re-Review: Phase 04 Daily Diet Alternative Search Shape

Recommended status: PASSED

## Scope

Reviewed exactly task 124 against the supplied row, dependency context, and previous rejection.

- Task 124 currently remains `REJECTED` in `docs/implementation/02_TASK_LIST.md`.
- Dependencies 116, 117, and 120 are `PASSED`.
- Dependency 123 is now `PREPARED`; its previous substitution-test failure no longer blocks the requested package verification.
- No task-list status, implementation code, or later task IDs were edited.

## Checklist

- [x] Production `CatalogService.Search` calls `PrepareSearchRequest(req, DailyDietDataUnavailable)` before cache or repository access.
- [x] Daily-diet mode with valid `dailyDietId` returns deterministic 422-style `SearchRejection` when Phase 07 saved-diet data is unavailable.
- [x] Daily-diet mode with missing `dailyDietId` returns validation failure before repository/cache side effects.
- [x] HTTP production-path tests cover valid daily-diet 422 and missing `dailyDietId` 400 with zero repository/cache side effects.
- [x] Service tests cover valid daily-diet rejection and missing-ID side-effect avoidance.
- [x] Available daily-diet preparation path still honors filters, pagination, and similarity eligibility when Phase 07 data is marked available.
- [x] Substitution-shaped requests are not incorrectly treated as daily-diet jobs.
- [x] Requested package verification passes.

## Implementation Evidence

- `backend/internal/search/catalog_service.go`: `Search` invokes `PrepareSearchRequest(req, DailyDietDataUnavailable)` before cache lookup and repository search, then returns the structured rejection immediately when present.
- `backend/internal/search/daily_diet.go`: daily-diet alternative parsing requires `DailyDietID`; unavailable Phase 07 data returns `phase_07_saved_diet_unavailable` on field `dailyDietId`.
- `backend/internal/httpapi/search_controller.go`: service-side `ErrDailyDietIDRequired` maps to 400 validation failure; structured search rejections map to 422 envelopes.
- `backend/internal/search/daily_diet_test.go`: covers missing ID, deterministic Phase 07 rejection, available-data filter/pagination path, similarity eligibility, and substitution precedence.
- `backend/internal/search/catalog_service_test.go`: covers production service daily-diet rejection and missing-ID handling with zero repository/cache calls.
- `backend/internal/httpapi/search_controller_test.go`: covers production HTTP daily-diet 422 and missing-ID 400 with zero repository/cache calls.

## Verification

Command run from `backend/`:

```sh
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/search ./internal/httpapi -count=1
```

Result:

```text
ok  	github.com/wiktor-jedski/mealswapp/backend/internal/search	1.075s
ok  	github.com/wiktor-jedski/mealswapp/backend/internal/httpapi	1.142s
```

## Conclusion

The previous blocking failure in `backend/internal/search` is resolved in the current workspace. Task 124 satisfies its verification criteria at the API boundary without Phase 07 optimization or saved-diet persistence and should be promoted from `REJECTED` to `PASSED`.
