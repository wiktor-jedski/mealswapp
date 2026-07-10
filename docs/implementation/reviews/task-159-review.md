# Review Evidence: Task 159 - DESIGN-007 UsageLimiter

Task ID: 159

Evidence path: `docs/implementation/reviews/task-159-review.md`

Recommended status: PASSED

Evidence date: 2026-07-01

## Checklist Summary

| Criterion | Result | Evidence |
| --- | --- | --- |
| Selected task is `PREPARED` and dependencies are `PREPARED` or `PASSED`. | PASS | `docs/implementation/02_TASK_LIST.md` rows 158 and 159 are both `PREPARED`. |
| Free users are capped at 3 counted searches per rolling 24 hours. | PASS | `CheckSearchAllowed` reads usage since `now - 24h`; `RecordCompletedSearch` calls `RecordUsageWithinLimit`; `TestUsageLimiterCapsFreeUsersAtThreeCountedSearchesPerRolling24Hours` verifies two in-window records allow one more and the next check is denied. |
| Usage is recorded only after allowed completed searches. | PASS | `CheckSearchAllowed` sets `CountUsageOnFinish` but does not write; `RecordCompletedSearch` is the write boundary; `TestUsageLimiterDoesNotCountDeniedAttemptsOrBeforeCompletion` verifies denied and pre-completion attempts are not counted. |
| Deterministic limit errors occur before paid-mode dispatch. | PASS | Exhausted free users receive `UsageDenyReasonFreeLimitReached` and `IsUsageLimitError`; free users requesting paid-only modes are denied before usage persistence in `TestUsageLimiterBlocksPaidModesBeforeUsageDispatchForFreeUsers`. |
| Trial/paid active users are not capped. | PASS | `CheckSearchAllowed` bypasses usage reads/writes for non-free active tiers; `TestUsageLimiterDoesNotCapTrialOrPaidActiveUsers` covers active `trial` and `paid` users. |
| Anonymous Catalog Search remains available without entitlement usage writes. | PASS | `anonymousDecision` allows `FeatureCatalog` with nil user and no count flag; `TestUsageLimiterAllowsAnonymousCatalogWithoutUsageWrites` verifies no usage repository calls. |
| Concurrent same-user attempts cannot exceed the persisted limit. | PASS | `RecordUsageWithinLimit` uses a transaction plus `pg_advisory_xact_lock` and re-counts the rolling window before insert; `TestUsageLimiterPostgresConcurrentSeparateInstancesCannotExceedPersistedLimit` verifies stale allowed decisions completed concurrently through separate limiter instances persist only 3 records. |
| Service and integration tests verify all required behavior. | PASS | Service tests cover the decision and counting criteria; PostgreSQL integration test covers persisted concurrent enforcement across separate limiter instances. |
| Traceability and task-list validation pass. | PASS | `python3 scripts/validate-task-list.py` and `python3 scripts/validate-traceability.py` both passed. |

## Commands Run / Results

| Command | Result |
| --- | --- |
| `rg -n "\| 15[89] \|" docs/implementation/02_TASK_LIST.md` | PASS: task 158 and task 159 are both `PREPARED`. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/entitlement ./internal/repository` | PASS, cached package results. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/entitlement ./internal/repository` | PASS: entitlement `0.115s`, repository `13.274s`. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -run TestUsageLimiterPostgresConcurrentSeparateInstancesCannotExceedPersistedLimit -v ./internal/entitlement` | PASS: PostgreSQL concurrency integration test executed and passed. |
| `python3 scripts/validate-task-list.py` | PASS: `Task-list validation passed: 175 sequential tasks with ordered dependencies.` |
| `python3 scripts/validate-traceability.py` | PASS: `Traceability validation passed.` |
| `gofmt -l backend/internal/entitlement/usage_limiter.go backend/internal/entitlement/usage_limiter_test.go backend/internal/entitlement/usage_limiter_integration_test.go backend/internal/repository/entitlement_repository.go backend/internal/repository/types.go backend/internal/repository/repository_test.go backend/internal/repository/postgres_repository_test.go` | PASS: no output, inspected Go files are formatted. |

## Files Inspected

- `docs/implementation/02_TASK_LIST.md`
- `backend/internal/entitlement/manager.go`
- `backend/internal/entitlement/usage_limiter.go`
- `backend/internal/entitlement/usage_limiter_test.go`
- `backend/internal/entitlement/usage_limiter_integration_test.go`
- `backend/internal/repository/entitlement_repository.go`
- `backend/internal/repository/types.go`
- `backend/internal/repository/repository_test.go`
- `backend/internal/repository/postgres.go`
- `backend/internal/repository/postgres_repository_test.go`
- `backend/internal/repository/sql/usage_window_advisory_lock.sql`
- `backend/internal/repository/sql/usage_window_record_within_limit.sql`
- `backend/internal/repository/sql/usage_window_get_since.sql`
- `backend/internal/repository/sql/usage_window_record.sql`
- `database/migrations/000008_entitlements.up.sql`

## Decision Reason

Recommend `PASSED`.

The repaired implementation directly satisfies the task 159 verification criteria. The service enforces free-tier rolling-window usage before dispatch, records only completed allowed searches, leaves active trial/paid users unmetered, permits anonymous Catalog Search without usage writes, and returns deterministic free-limit denials. PostgreSQL-backed enforcement is now atomic at completion time through transaction-scoped advisory locking and a limit recheck, and the dedicated integration test confirms concurrent same-user completions through separate limiter instances cannot persist more than 3 counted searches.

## Repair Instructions If Rejected

None.
