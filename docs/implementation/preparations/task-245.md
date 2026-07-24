# Task 245 preparation evidence

## Scope and baseline

- Task: 245, Phase 08 Provider Rate Limits and Retry Orchestration.
- Fixed reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Repair source: `docs/implementation/reviews/task-245-review.md` (`REJECTED`, five important findings).
- Scope was limited to the Task 245 rate-limit/orchestration files, the USDA/OpenFoodFacts result boundary required by the review, their adversarial tests, and this preparation evidence.
- The repository began with extensive unrelated Phase 08 changes. `docs/implementation/02_TASK_LIST.md` and `docs/implementation/04_OPEN.md` were already modified and were not edited by this repair. Task 245 status remains unchanged.
- No coverage exception was added: task-owned production code reaches 100% statement coverage.

## Repair summary

### Provider result and response-header boundary

- Replaced the optional records-only/result-aware split with one required `ResultProvider.SearchResult` contract returning `ProviderResult` on success and failure.
- `USDAClient` and `OpenFoodFactsClient` now implement `SearchResult`; their existing `Search` methods remain compatibility wrappers.
- Provider clients project only `X-RateLimit-Remaining` and `X-RateLimit-Reset`; unrelated/provider-secret headers are discarded.
- `SearchExternalFoods` records projected headers before classifying errors. A 429 quota response therefore blocks later calls until reset instead of being retried against known exhausted quota.

### Cancellation and validation behavior

- Parent cancellation and deadline identity are checked immediately after each provider attempt and returned without retry or warning substitution.
- Per-attempt child deadlines remain retryable provider timeouts and retain the bounded `timeout` warning behavior.
- Provider-neutral query text, provider, page, and provider-specific page-size bounds are validated before selection or outbound I/O.
- Missing selected providers emit bounded `provider_unavailable` warnings, including partial-success and both-missing cases.

### Adversarial tests

- Error response headers update quota state and skip subsequent provider calls until reset.
- Real USDA and OpenFoodFacts fake servers prove bounded header projection on both successful and failed HTTP responses.
- Real-provider orchestration proves USDA 429 isolation from a successful OpenFoodFacts sibling.
- In-flight caller cancellation and caller deadline preserve `context.Canceled` and `context.DeadlineExceeded`, perform no retry sleep, and do not dispatch later work.
- Invalid query/provider/page/page-size matrices make zero provider calls.
- One and both missing providers produce bounded unavailable warnings while sibling success is retained.
- Default jitter, invalid configuration, canceled sleep, blank rate-limit provider, OpenFoodFacts-only selection, retry-sleep failure, and provider-originated cancellation cover the remaining task-owned branches.

## Changed paths and current SHA-256

| Path | Repair surface | SHA-256 |
|---|---|---|
| `backend/internal/externaldata/rate_limit.go` | Result-bearing provider contract, bounded header projection, validation, cancellation/deadline propagation, missing-provider handling | `d7649887ac6fb8c960a2df2734f33ca842fc680cce94722a150a52d31a08660b` |
| `backend/internal/externaldata/rate_limit_test.go` | Deterministic and adversarial orchestration tests | `f7ea7b2e01e041cbff3c40688e929214b2919669715009a96ff4f339536ef025` |
| `backend/internal/externaldata/usda.go` | `SearchResult` implementation with headers on success/error | `78b198fba01bd7da8ff665b401e5b95b1fa64fdd86fc3df030e9c63864aac116` |
| `backend/internal/externaldata/usda_test.go` | USDA success/error bounded-header tests | `d76cfc8a6ae122c8b7a182dcfb85a4c4567803ccf25d810db3addbe5d3364b58` |
| `backend/internal/externaldata/openfoodfacts.go` | `SearchResult` implementation with headers on success/error | `e839c6594ab265a11c8ab1b7bb2d295911696a86adc08326bf789e9ba1e0ef57` |
| `backend/internal/externaldata/openfoodfacts_test.go` | OpenFoodFacts success/error bounded-header tests | `c41107a76b9651fccb98d4eead9d579cd6d4080923715a756cc8e7e8a6c06742` |
| `docs/design/DESIGN-012.md` | Read-only design source | `53ac9bd6a34bd07216666d4beaae6533a0281c905fc2d5c474f48f614746eddf` |

## Verification

Commands were run from `/home/wiktor/Work/mealswapp` unless a working directory is shown.

| Command | Result |
|---|---|
| `gofmt -w backend/internal/externaldata/{rate_limit.go,rate_limit_test.go,usda.go,usda_test.go,openfoodfacts.go,openfoodfacts_test.go}` | PASS |
| `git diff --check -- backend/internal/externaldata` | PASS |
| Corrected named task suite (command below) | PASS; corrected Go regexp selected the task tests; every `rate_limit.go` function is 100% covered |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/externaldata -coverprofile=/tmp/task245-full.cover` | PASS; external-data package 100.0% statement coverage |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./...` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./internal/externaldata` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./...` | FAIL outside Task 245 after external-data passed: `internal/worker/TestTask234ProductionWorkerQueueTelemetryIsWiredAndPrivacySafe` did not emit queue cleanup telemetry while Redis attempted `127.0.0.1:0` |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS; no reachable vulnerabilities; 18 advisories only in uncalled module code |
| `python3 scripts/validate-task-list.py` | PASS; 263 sequential tasks; status untouched |
| `python3 scripts/validate-traceability.py` | PASS |
| `python3 scripts/check.py` | Reached local-stack verification after traceability, task-list, Go Doc, OpenAPI, script tests, vet, vulnerability, and focused backend gates passed; then failed outside Task 245 because existing local migration `000003_classifications.up.sql` could not find relation `food_items` (`SQLSTATE 42P01`) |

Corrected named task command (alternation is intentionally unescaped):

```bash
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/externaldata -run 'Test(SearchExternalFoods|RateLimitHandler|ExternalSearch|USDASearchResult|OpenFoodFactsSearchResult)' -coverprofile=/tmp/task245-named.cover
```

## Coverage assessment

`/tmp/task245-full.cover` reports 100.0% for the full external-data package. `/tmp/task245-named.cover` independently reports 100.0% for every task-owned production function: `NewRateLimitHandler`, `Configure`, `contextSleep`, `CheckRateLimit`, `RecordRateLimit`, `blocked`, `backoff`, `projectRateLimitHeaders`, `SearchExternalFoods`, and `validateExternalSearchQuery`. No exception in `docs/implementation/04_OPEN.md` is necessary or authorized.

## Remaining external blockers

- The full race command has an unrelated Task 234 worker/Redis telemetry failure; focused Task 245 race verification passes.
- The aggregate command is blocked by pre-existing local PostgreSQL migration state before aggregate frontend and later checks can run.
- Neither blocker changes Task 245 behavior, coverage, security, traceability, or task-list status.
