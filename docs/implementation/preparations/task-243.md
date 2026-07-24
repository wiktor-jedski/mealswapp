# Task 243 Preparation — USDA External Data Client

## Outcome and task control

- Task: **243 — Phase 08 USDA External Data Client**.
- Result: **review findings repaired and verified; every row criterion has direct fake-server or adversarial boundary-test evidence**.
- Fixed implementation reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Preparation refreshed: 2026-07-21, Europe/Warsaw, after `docs/implementation/reviews/task-243-review.md` rejection repair.
- Dependencies 7 and 242 were re-read from `docs/implementation/02_TASK_LIST.md` and remain `PASSED`.
- Task 243 was observed as `PREPARED`. This repair did not edit its status or any task-list content.
- Final observed task-list SHA-256 control: `573cbd8d36d099a9951f1fbf2ea8fc797862d0a974fdc8e90c64a6d27cc841f6`.
- The phase-orchestrator skill required delegation when a writable subagent is available. No writable subagent capability was available in this session, so the parent executed the single-task preparation contract directly.

## Baseline and scope ownership

- Baseline commit: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Baseline `git status --short` already contained Tasks 238–242 and unrelated tracked/untracked changes in API, application, cache, curation, custom-item, deletion, HTTP, repository, search, security, userdata, database migration, frontend, scripts, design, task-list, open-issue, preparation, and review paths.
- `backend/internal/externaldata/` and `docs/implementation/preparations/task-243.md` did not exist as Task 243 paths at baseline. No pre-existing file was modified by Task 243.
- Original Task 243 candidate hashes were therefore absence controls. The repair did not edit the already-modified task list; its current observed control hash is recorded below.
- No unrelated file was cleaned, reverted, staged, overwritten, or reformatted.
- Task 243 intentionally does not compose the client into the application, add OpenFoodFacts, retries/rate-limit state, normalization, admin routes, persistence, or OpenAPI. Those surfaces belong to Tasks 244–253.

## Design, requirement, and provider boundary inspected

- `docs/design/DESIGN-012.md`: `USDAClient`, `ExternalSearchQuery`, `ExternalFoodRecord`, request construction, pagination, parsing, timeout, invalid-payload, unavailable, and rate-limited states.
- `docs/architecture/ARCH-012.md` and `docs/architecture/01_SOFT_ARCH_DESIGN.md`: USDA HTTPS/API-key boundary, on-demand curation, pagination, and graceful provider failure.
- `docs/architecture/02_APPENDIX_A.md`: external API timeout/status failure policy and later retry ownership.
- `docs/requirements/01_SOFT_REQ_SPEC.md`: SW-REQ-055 external-data curation.
- `docs/design/01_TECH_STACK.md`: USDA FoodData Central as the selected food-data provider.
- `docs/design/DESIGN-005.md`, repository unit code, and Task 246: volume must not be treated as mass; USDA portions therefore preserve amount, unit, and provider-measured gram weight for later density derivation.
- Task 242's `security.NormalizeInput` contract: normalized external query/provider and bounded one-based page validation are reused before outbound I/O.

## Security assessment

The `golang-security` skill was applied in coding mode.

- Trust boundary: query/provider/page/page-size and all provider bytes are untrusted. Input is validated before `http.Client.Do`; payload envelope, identities, serving pairs, nutrients, portions, finite values, and duplicates fail closed.
- Credential/redirect boundary: `MEALSWAPP_USDA_API_KEY` is loaded and trimmed without logging. The constructor clones the configured HTTP client and unconditionally disables redirects, so credential-bearing requests cannot follow cross-host, HTTPS-to-HTTP, or private-target redirects and cannot disclose the query API key through `Referer`. The caller's client is not mutated.
- A manual security pass found that generic `http.Client` transport errors may embed the complete URL and query-string API key. The implementation discards all arbitrary underlying causes after categorical mapping and unwraps only `context.Canceled` or `context.DeadlineExceeded` sentinels.
- Cancellation boundary: transport and body-read failures share safe context classification. Deadline and caller-cancellation sentinels are preserved through `errors.Is` while arbitrary URL- or provider-bearing causes remain discarded.
- Denial-of-service controls: query/page/page-size bounds, per-call context deadline, a fixed 2 MiB maximum response configuration independent of caller input, overflow-safe `limit + 1` after constructor enforcement, bounded raw payload retention, and no retries in this task. Exact maximum is accepted; maximum-plus-one and `math.MaxInt64` are rejected.
- Diagnostics use a closed provider/code/status/retryable field set. Fake-provider bodies and secret sentinels are proven absent from errors and structured logs.
- Endpoint configuration accepts only absolute HTTP(S) URLs without credentials, query strings, or fragments. HTTP remains injectable for loopback fake servers; the production default is HTTPS and redirects are disabled for every endpoint/client configuration.
- No SQL, filesystem writes, command execution, authentication state, cookies, PII, persistence, or cryptography were added.

## Exact changed paths and symbols

| Path | Task 243 surface |
| --- | --- |
| `backend/internal/externaldata/usda.go` | New external-data package, USDA request/response contracts, immutable configuration, environment credential loader, bounded HTTP search, strict projection, safe errors/logs, and status/context mapping. |
| `backend/internal/externaldata/usda_test.go` | Fake-server and direct-boundary tests for every verification clause plus 100% package statement coverage. |
| `docs/implementation/preparations/task-243.md` | Baseline, scope, security, symbol, acceptance, command, and hash evidence. |

Production declarations added in `usda.go`:

- Constants: `USDAAPIKeyEnvironment`, `DefaultUSDAEndpoint`, `MaxUSDAPageSize`, `defaultUSDADeadline`, `defaultUSDABodyLimit`, `maxUSDABodyLimit`; all `ProviderError*` codes.
- Contracts/types: `ExternalSearchQuery`, `ExternalFoodPortion`, `ExternalFoodRecord`, `ProviderErrorCode`, `ProviderError`, `USDAConfig`, `USDAClient`.
- Provider payload types: `usdaSearchPayload`, `usdaFood`, `usdaNutrient`, `usdaMeasure`, `usdaMeasureUnit`.
- Exported behavior: `(*ProviderError).Error`, `(*ProviderError).Unwrap`, `LoadUSDAAPIKey`, `NewUSDAClient`, `(*USDAClient).Search`.
- Private behavior: `validateUSDAQuery`, `decodeUSDASearch`, `finitePositive`, `mapUSDAStatus`, `(*USDAClient).transportFailure` (now accepts the transport/body error and HTTP status), `(*USDAClient).failure`, and the `ProviderError` compile-time assertion.

Test declarations added in `usda_test.go`:

- Tests: `TestLoadUSDAAPIKey`, `TestUSDASearchEncodesQueryKeyPaginationAndProjectsDeterministically`, `TestUSDASearchRejectsInvalidInputBeforeOutboundRequest`, `TestUSDASearchHonorsDeadlineAndCallerCancellation`, `TestUSDASearchPreservesContextErrorsWhileReadingBody`, `TestUSDASearchDoesNotFollowCredentialBearingRedirects`, `TestUSDASearchBoundsResponseAndRejectsMalformedOrPartialPayloads`, `TestUSDASearchHandlesRequestTransportAndBodyReadFailures`, `TestDecodeUSDASearchAcceptsEmptyResultsAndOrdersPortionTies`, `TestUSDASearchMapsProviderStatusesAndLogsOnlyBoundedMetadata`, `TestNewUSDAClientRejectsUnsafeConfiguration`.
- Test fixtures/helpers: `testUSDAKey`, `validUSDAPayload`, `newTestUSDAClient`, `validUSDAQuery`, `searchPayload`, `assertProviderError`, `roundTripFunc`, `roundTripFunc.RoundTrip`, `failingBody`, `failingBody.Read`, `failingBody.Close`, `signalingBody`, and `(*signalingBody).Read`.

## Verification criteria

| Criterion | Direct evidence | Result |
| --- | --- | --- |
| URL/query encoding | Fake server receives NFC-normalized `crème & apple`, exact endpoint path, encoded URI, and only expected USDA query fields. | PASS |
| API-key handling without secret logging | Environment/config tests cover trim, missing/control credentials, query delivery, provider-body secret sentinel, error formatting, log serialization, dropped URL-bearing causes, and an HTTPS source redirect toward a cross-host HTTP loopback/private destination. Redirect is returned to the caller; destination calls remain zero, preventing `Referer` leakage. | PASS |
| Page boundaries | Page 1..10,000 and size 1..200 are enforced; exact maximum reaches the fake server; zero/above-maximum values make no request. | PASS |
| Deadlines and cancellation | Blocking fake servers cover waiting and body-reading phases. A signaling body proves cancellation occurs during `Read`; deadline and caller cancellation map to `timeout`/`canceled` and preserve only the matching safe context sentinel. | PASS |
| Bounded bodies | A fixed 2 MiB constructor ceiling makes `max+1` overflow impossible and independently caps response allocation. Small oversized bodies are rejected; exact maximum configuration is accepted; maximum-plus-one and `math.MaxInt64` are rejected. | PASS |
| Malformed/partial payload rejection | Tests reject malformed JSON/food, missing envelope/foods/identity/nutrients, incomplete serving pair, invalid/duplicate nutrients, and invalid portions. | PASS |
| Provider status mapping | 400/401/404 map permanent rejection, 408/5xx map retryable unavailable, 429 maps retryable rate limit, and unexpected non-success maps unavailable. | PASS |
| No outbound invalid input | Atomic fake-server counter remains zero across empty/long/control queries, wrong provider, invalid pages, and invalid page sizes. | PASS |
| Deterministic record projection | Exact provider/ID/name/serving/nutrient/raw projection is compared; portions sort by unit, amount, then gram weight. | PASS |
| Volume portions with gram weights | `cup`, `tbsp`, measure-name, and dissemination-text fallbacks retain positive amount and gram weight, including tie ordering. | PASS |

## Commands and results

| Command | Result |
| --- | --- |
| Focused adversarial `go test -count=1 -run ... ./internal/externaldata` | PASS for body-read deadline/cancellation, credential-bearing redirect rejection, body bounds, and unsafe configuration maxima. |
| `go test -count=1 -v ./internal/externaldata` | PASS; all USDA and colocated OpenFoodFacts fake-server/boundary cases. |
| `go test -count=1 -race -coverprofile=task-243-cover.out ./internal/externaldata` plus `go tool cover -func=task-243-cover.out` | PASS; race clean; **100.0% statements** for every production function. The temporary coverage profile was removed after recording results. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS for every backend package. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./...` | PASS for every backend package. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./...` | PASS for every backend package. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS: no vulnerabilities in called code or imported packages; 18 required-module advisories are not called. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks with ordered dependencies; Task 243 remains `PREPARED`; no task-list edit was made. |
| `python3 scripts/validate-traceability.py` | PASS with required adjacent `DESIGN-012` comments. |
| `git diff --check` | PASS. |
| First `python3 scripts/check.py` repair run | FAIL outside Task 243: unrelated `TestTask225LiveManagerRecoversAfterRedisRestart` deadlocked in go-redis semaphore release and hit its 10-minute timeout. All earlier aggregate gates and Task 243 passed. |
| `MEALSWAPP_REDIS_URL=redis://localhost:6379/10 ... go test -count=1 -v -run '^TestTask225LiveManagerRecoversAfterRedisRestart$' -timeout=2m ./internal/queue` | PASS in 0.78s, confirming the first aggregate failure was transient infrastructure behavior. |
| Second `python3 scripts/check.py` repair run | PASS, exit 0: traceability/task list/Go Doc, OpenAPI/security, migrations and local-stack readiness, Phase 02/03 UAT, backend tests/race/coverage, frontend API drift/typecheck/build/unit/coverage, 72 + 30 focused Playwright checks, full Playwright 237 passed/3 expected skipped, and 459 frontend unit tests. The existing OAuth 302-only warning and documented repository-wide coverage deviations remain unchanged. |

## Final implementation hashes

| Path | SHA-256 |
| --- | --- |
| `backend/internal/externaldata/usda.go` | `21f29cb5c6d1c528339cce5aa4d78622bfa6bcc9051d9a0888dc2c51cee07beb` |
| `backend/internal/externaldata/usda_test.go` | `b9045286353943098e3b3f4fa4576bb0954312ee9191a4ff622bad7e7a9dfd25` |
| `docs/implementation/02_TASK_LIST.md` (observed control; not edited by repair) | `573cbd8d36d099a9951f1fbf2ea8fc797862d0a974fdc8e90c64a6d27cc841f6` |

## Risks and handoff

- No Task 243 acceptance, race, security, traceability, or aggregate-check blocker remains. The first aggregate attempt's unrelated Task 225 Redis-restart timeout passed immediately in isolation and the complete aggregate rerun passed.
- `ExternalFoodRecord.RawPayload` is bounded and retained because DESIGN-012 declares it. Provider payloads are never logged or exposed by this client; later persistence/API work must continue that boundary.
- Task 245 owns retry and rate-limit state. This client performs exactly one bounded request and exposes categorical retryability/status metadata for that layer.
- Task 246 owns nutrient aliasing, canonical unit conversion, liquid density priority, and warnings. This client deliberately preserves unit-qualified nutrient names and measured volume-portion evidence without making a 1 ml = 1 g assumption.
- Independent review should recompute the hashes above and inspect all listed declarations. Task-list status must remain untouched by the preparation agent.
