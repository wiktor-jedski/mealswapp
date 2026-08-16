# Review Evidence: Task 245 — RateLimitHandler

task_id: 245
component: "RateLimitHandler"
static_aspect: "DESIGN-012: RateLimitHandler"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-21T12:17:35Z"
review_agent: "Codex GPT-5 independent re-review"
evidence_file: "docs/implementation/reviews/task-245-review.md"
baseline_ref: "81ca40ce00cb667ea29243ed2d34068e11229a69 plus task-245 preparation manifest"
baseline_confidence: "MEDIUM"
code_review_skill_invoked: true
relevant_language_guide: "Go, async/concurrency patterns, and Go security review guides"
repair_context_required: true

## 1. Task Source

Description: Phase 08 Provider Rate Limits and Retry Orchestration. Implement per-provider quota state, configured call deadlines, response-header updates, exponential backoff with jitter for transient failures up to three retries, and partial-success behavior when one provider is unavailable or rate limited.

Depends On: 243, 244 — both are PASSED in the current task list.

Testing Coverage Exceptions: None. The task row explicitly says None and no exception was added to docs/implementation/04_OPEN.md.

Verification Criteria: Deterministic clock/jitter tests and fake-provider integration tests cover available, rate-limited, unavailable, timeout, cancellation, reset, retry exhaustion, concurrent provider isolation, combined-provider partial success, bounded warning codes, permanent failures are not retried, and no provider secrets or raw payloads enter logs or metrics.

## 2. Pre-Review Gates

- [x] Input status is PREPARED.
- [x] Every dependency is PREPARED or PASSED; dependencies 243 and 244 are PASSED.
- [x] The preparation report claims completion.
- [x] A task-specific baseline/diff is available and trustworthy. The externaldata files are untracked relative to the fixed baseline, so the repair preparation manifest and current file hashes are used as the scope manifest.
- [x] code-review-skill was invoked exactly once in this review turn and its Go, concurrency, and security guides were read.
- [x] The reviewer is independent from implementation/repair for this review pass.
- [x] Review uses current repository state rather than stale logs. Focused tests, full package coverage, package race, repository tests, vet, vulnerability, and validators were rerun.
- [x] Reviewer made no production-code or task-list changes.

pre_review_gates_passed: true
blocking_issue: "None"

## 3. Review Baseline and Change Surface

Baseline/reference method: Start at fixed reference 81ca40ce00cb667ea29243ed2d34068e11229a69. Inspect the task-list row and dependency statuses, compare tracked changes, confirm that backend/internal/externaldata is absent from the fixed tree, then use docs/implementation/preparations/task-245.md as the repair scope manifest. Reconstruct the current task-owned surface from that manifest, current source, all callers, tests, and fresh SHA-256 hashes. The shared NormalizedFoodCandidate consumer and normalizer were inspected as an adjacent boundary.

Commands used to reconstruct the diff:

    git status --short
    git ls-tree -r --name-only 81ca40ce00cb667ea29243ed2d34068e11229a69 -- backend/internal/externaldata
    git diff --stat 81ca40ce00cb667ea29243ed2d34068e11229a69 -- backend/internal/externaldata docs/design/DESIGN-012.md docs/implementation/02_TASK_LIST.md
    rg -n "SearchExternalFoods|ResultProvider|NormalizedFoodCandidate|SearchResult|ProviderSet" backend docs/design
    sha256sum backend/internal/externaldata/{rate_limit.go,rate_limit_test.go,usda.go,usda_test.go,openfoodfacts.go,openfoodfacts_test.go,normalizer.go,normalizer_test.go}

The corrected focused command used unescaped Go-regexp alternation:

    cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/externaldata -run 'Test(SearchExternalFoods|RateLimitHandler|ExternalSearch|USDASearchResult|OpenFoodFactsSearchResult)' -coverprofile=/tmp/task245-named-review.cover

Pre-existing dirty-worktree changes and exclusions:

The worktree contains broad unrelated Phase 08 changes, including api/openapi.yaml, backend application/repository/security/userdata files, frontend generated types, migrations, docs/implementation/04_OPEN.md, and task-list transitions for tasks 238–244 and 247. Those changes were not attributed to Task 245. The task row 245 remains PREPARED and was not edited. The new backend/internal/externaldata files are the Task 243/244/245/246 area; only rate_limit.go, rate_limit_test.go, the repaired provider result boundary and its tests are in this review surface. normalizer.go and normalizer_test.go were inspected only as the shared NormalizedFoodCandidate boundary and are not attributed as Task 245 changes.

If any task-owned change cannot be distinguished reliably, stop and recommend REJECTED. In this pass the preparation manifest, fixed reference, scope-specific source inspection, callers, tests, and matching current hashes provide sufficient MEDIUM-confidence attribution.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| backend/internal/externaldata/rate_limit.go | Task 245 implementation and repaired orchestration boundary | HIGH | Quota handler, result contract, bounded headers, ProviderSet, candidate boundary, orchestration, validation |
| backend/internal/externaldata/rate_limit_test.go | Task 245 deterministic, adversarial, and integration tests | HIGH | Four test doubles and thirteen orchestration tests |
| backend/internal/externaldata/usda.go | Task 245 repair required by the real provider boundary; Search wrapper and SearchResult | MEDIUM | USDA Search compatibility wrapper and result-bearing SearchResult |
| backend/internal/externaldata/usda_test.go | Task 245 repair regression test for bounded headers on success and error | MEDIUM | TestUSDASearchResultProjectsBoundedHeadersOnSuccessAndFailure |
| backend/internal/externaldata/openfoodfacts.go | Task 245 repair required by the real provider boundary; Search wrapper and SearchResult | MEDIUM | OpenFoodFacts Search compatibility wrapper and result-bearing SearchResult |
| backend/internal/externaldata/openfoodfacts_test.go | Task 245 repair regression test for bounded headers on success and error | MEDIUM | TestOpenFoodFactsSearchResultProjectsBoundedHeadersOnSuccessAndFailure |
| backend/internal/externaldata/normalizer.go | Adjacent shared NormalizedFoodCandidate consumer, read-only boundary audit | N/A — no Task 245 change attributed | NormalizeRecords and normalization consumers inspected |
| backend/internal/externaldata/normalizer_test.go | Adjacent shared candidate tests, read-only boundary audit | N/A — no Task 245 change attributed | Candidate construction and serving-measure tests inspected |
| docs/design/DESIGN-012.md | Read-only source-of-truth comparison | N/A — no Task 245 change attributed | RateLimitHandler, provider, candidate, and state/error criteria |

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Per-provider quota state is isolated. | rate_limit.go state map and concurrent state test; package race suite | PASS | RateLimitHandler keys all state by provider under one mutex. TestSearchExternalFoodsUsesConfiguredDeadlineAndConcurrentProviderIsolation concurrently records USDA and OpenFoodFacts state and verifies no crossover. |
| 2 | Configured provider-call deadlines are applied. | Configure, blocking-provider test, provider deadline tests | PASS | SearchExternalFoods snapshots the configured deadline and wraps every attempt with context.WithTimeout. The configured 2 ms blocking-provider test produces the bounded timeout warning; both real clients also retain their own defensive deadlines. |
| 3 | Response headers update quota state on success and error. | real USDA/OpenFoodFacts fake-server tests and orchestration header test | PASS | SearchResult returns projected headers for both HTTP success and non-2xx outcomes. TestSearchExternalFoodsPropagatesRealProviderHeadersOnErrorAndSuccess proves USDA 429 state is recorded and OpenFoodFacts success state is recorded; the direct provider tests prove both real adapters. |
| 4 | Exponential backoff with jitter is used for transient failures. | deterministic clock/jitter test and backoff implementation | PASS | TestRateLimitHandlerDeterministicRetryDeadlineAndHeaderIsolation observes 150 ms, 300 ms, and 600 ms sleeps for a 100 ms base with half-duration jitter. |
| 5 | Retry count is capped at three retries. | four-outcome retry exhaustion test | PASS | The deterministic test supplies four retryable failures and observes exactly four provider calls and three sleeps, then retry_exhausted. |
| 6 | Available provider results are returned. | fake-provider and real OpenFoodFacts success path | PASS | Successful ProviderResult records are projected into candidates, and the real-provider combined test returns one OpenFoodFacts item. |
| 7 | Rate-limited providers are skipped and reported with bounded warning codes. | error-header quota test, reset test, combined real-provider test | PASS | A 429 with remaining 0 and reset 200 records state before classification; the next call is skipped until reset and emits provider_rate_limited. |
| 8 | Unavailable or missing providers produce bounded warnings. | missing-provider and permanent-unavailable tests | PASS | Nil selected providers emit provider_unavailable while preserving sibling results. Non-retryable unavailable errors make one call and emit one bounded warning. |
| 9 | Provider timeouts use the timeout outcome. | configured-deadline blocking-provider test and full externaldata tests | PASS | Child deadline expiry is mapped by the provider boundary to retryable ProviderErrorTimeout and the orchestrator emits timeout after the retry policy; package coverage exercises the path. |
| 10 | Caller cancellation and deadline identity are preserved. | in-flight cancellation and deadline tests, provider cancellation test | PASS | Parent context cancellation or deadline is checked immediately after each call and returned as context.Canceled or context.DeadlineExceeded with errors.Is identity, without retry sleep or warning substitution. Provider-originated cancellation is also returned without retry. |
| 11 | Reset windows block before reset and permit the boundary after reset. | deterministic mutable-clock quota reset test | PASS | TestSearchExternalFoodsQuotaResetSkipsBeforeResetAndAllowsAfterReset observes zero provider calls before reset and one call at the exact reset timestamp. |
| 12 | Combined-provider partial success is retained. | all-provider partial-success test and real USDA 429 plus OpenFoodFacts success test | PASS | Successful candidates remain in the result while the failing or missing sibling contributes only a bounded warning; no sibling quota state is shared. |
| 13 | Concurrent access to handler state is safe. | race-enabled externaldata package and concurrent state test | PASS | go test -race ./internal/externaldata passes; CheckRateLimit, RecordRateLimit, blocked, and backoff use the handler mutex and return copies rather than shared mutable state. |
| 14 | Permanent failures are not retried. | non-retryable ProviderError test and rejected-provider branch | PASS | TestSearchExternalFoodsNonRetryableUnavailableIsWarningAndSingleCall and the partial-outcomes test both observe one call for permanent failures. |
| 15 | Warnings, logs, and telemetry contain no secrets or raw provider payloads. | telemetry negative test, real provider log tests, bounded projection tests, vulnerability scan | PASS | Orchestration warnings use a fixed four-code vocabulary and message equals code. Provider diagnostics retain only provider, code, status, retryable, and bounded counts; projected headers discard X-Provider-Secret and raw payloads. No Task 245 metrics or raw payload logging exists. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | MaxProviderRetries and warning constants | constants | backend/internal/externaldata/rate_limit.go:17-24 | Added | SearchExternalFoods and warning assertions | Rate-limit orchestration tests |
| 2 | ProviderRateLimit | type | backend/internal/externaldata/rate_limit.go:26-33 | Added | CheckRateLimit callers and tests | Quota and isolation tests |
| 3 | rateState | type | backend/internal/externaldata/rate_limit.go:35-37 | Added | RateLimitHandler.states | Handler tests |
| 4 | RateLimitHandler | type | backend/internal/externaldata/rate_limit.go:39-49 | Added | SearchExternalFoods and direct handler tests | All rate-limit tests |
| 5 | NewRateLimitHandler | function | backend/internal/externaldata/rate_limit.go:51-61 | Added | SearchExternalFoods default and tests | TestRateLimitHandlerAdversarialBranches |
| 6 | RateLimitHandler.Configure | method | backend/internal/externaldata/rate_limit.go:63-76 | Added | SearchExternalFoods configuration and tests | Adversarial, deadline, deterministic tests |
| 7 | contextSleep | function | backend/internal/externaldata/rate_limit.go:78-89 | Added | RateLimitHandler default sleep | Adversarial and cancellation tests |
| 8 | RateLimitHandler.CheckRateLimit | method | backend/internal/externaldata/rate_limit.go:91-97 | Added | Tests and quota inspection | Isolation, header, reset tests |
| 9 | RateLimitHandler.RecordRateLimit | method | backend/internal/externaldata/rate_limit.go:99-117 | Added | SearchExternalFoods and direct tests | Header, invalid-provider, isolation tests |
| 10 | RateLimitHandler.blocked | method | backend/internal/externaldata/rate_limit.go:119-126 | Added | SearchExternalFoods preflight and retry loop | Quota/reset/partial tests |
| 11 | RateLimitHandler.backoff | method | backend/internal/externaldata/rate_limit.go:128-140 | Added | SearchExternalFoods retry loop | Deterministic retry tests |
| 12 | ResultProvider | interface | backend/internal/externaldata/rate_limit.go:142-146 | Added and repaired | ProviderSet and SearchExternalFoods | Fake providers and real client compile/use |
| 13 | ProviderResult | type | backend/internal/externaldata/rate_limit.go:148-153 | Added and repaired | ResultProvider boundary and candidate projection | Header success/error tests |
| 14 | projectRateLimitHeaders | function | backend/internal/externaldata/rate_limit.go:155-165 | Added | USDA and OpenFoodFacts SearchResult | Real-provider header tests |
| 15 | ProviderSet | type | backend/internal/externaldata/rate_limit.go:167-172 | Added and repaired | SearchExternalFoods caller boundary | Selection, missing-provider, partial tests |
| 16 | ExternalDataWarning | type | backend/internal/externaldata/rate_limit.go:174-180 | Added | SearchExternalFoods result boundary | Warning assertions |
| 17 | NormalizedFoodCandidate | type | backend/internal/externaldata/rate_limit.go:182-204 | Added/shared | SearchExternalFoods output and DataNormalizer input | Rate-limit and normalizer tests |
| 18 | SearchExternalFoods | function | backend/internal/externaldata/rate_limit.go:206-307 | Added and repaired | No current production caller; reserved for Task 248 proxy | Thirteen orchestration tests |
| 19 | validateExternalSearchQuery | function | backend/internal/externaldata/rate_limit.go:309-332 | Added and repaired | SearchExternalFoods before provider selection | Invalid-input matrix |
| 20 | USDAClient.Search | method | backend/internal/externaldata/usda.go:169-174 | Modified as compatibility wrapper | Existing USDA direct callers and tests; no orchestrator caller | Existing USDA Search tests |
| 21 | USDAClient.SearchResult | method | backend/internal/externaldata/usda.go:176-220 | Added for repaired ResultProvider contract | ProviderSet through SearchExternalFoods | USDA bounded-header test and client suite |
| 22 | TestUSDASearchResultProjectsBoundedHeadersOnSuccessAndFailure | test | backend/internal/externaldata/usda_test.go:334-359 | Added | USDA SearchResult boundary | Direct fake-server success/error assertions |
| 23 | OpenFoodFactsClient.Search | method | backend/internal/externaldata/openfoodfacts.go:87-92 | Modified as compatibility wrapper | Existing OpenFoodFacts direct callers and tests; no orchestrator caller | Existing OpenFoodFacts Search tests |
| 24 | OpenFoodFactsClient.SearchResult | method | backend/internal/externaldata/openfoodfacts.go:94-146 | Added for repaired ResultProvider contract | ProviderSet through SearchExternalFoods | OpenFoodFacts bounded-header test and client suite |
| 25 | TestOpenFoodFactsSearchResultProjectsBoundedHeadersOnSuccessAndFailure | test | backend/internal/externaldata/openfoodfacts_test.go:362-387 | Added | OpenFoodFacts SearchResult boundary | Direct fake-server success/error assertions |
| 26 | fakeProvider | test type | backend/internal/externaldata/rate_limit_test.go:15-21 | Added | SearchExternalFoods tests | Its method and count helper |
| 27 | fakeProvider.SearchResult | test method | backend/internal/externaldata/rate_limit_test.go:23-38 | Added | ResultProvider test double | Selection, retry, partial, invalid-input tests |
| 28 | fakeProvider.count | test method | backend/internal/externaldata/rate_limit_test.go:39 | Added | Test call-count assertions | Retry and no-I/O assertions |
| 29 | blockingProvider | test type | backend/internal/externaldata/rate_limit_test.go:41 | Added | Configured deadline test | Uses context cancellation |
| 30 | blockingProvider.SearchResult | test method | backend/internal/externaldata/rate_limit_test.go:43-46 | Added | SearchExternalFoods configured deadline path | Configured deadline test |
| 31 | resultErrorProvider | test type | backend/internal/externaldata/rate_limit_test.go:48-52 | Added | Header/error and cancellation tests | Its method and call assertions |
| 32 | resultErrorProvider.SearchResult | test method | backend/internal/externaldata/rate_limit_test.go:54-57 | Added | ResultProvider error-result boundary | Error header and provider cancellation tests |
| 33 | cancelingProvider | test type | backend/internal/externaldata/rate_limit_test.go:59-62 | Added | In-flight caller cancellation/deadline tests | Its method and call assertions |
| 34 | cancelingProvider.SearchResult | test method | backend/internal/externaldata/rate_limit_test.go:64-69 | Added | In-flight context path | Caller cancellation and deadline tests |
| 35 | TestSearchExternalFoodsRecordsErrorHeadersAndSkipsUntilReset | test | backend/internal/externaldata/rate_limit_test.go:71-88 | Added | SearchExternalFoods | Error headers and skip behavior |
| 36 | TestSearchExternalFoodsPropagatesRealProviderHeadersOnErrorAndSuccess | test | backend/internal/externaldata/rate_limit_test.go:90-133 | Added | USDA/OpenFoodFacts SearchResult and orchestrator | Real fake-server boundary and sibling isolation |
| 37 | TestSearchExternalFoodsPreservesInFlightCallerCancellation | test | backend/internal/externaldata/rate_limit_test.go:135-153 | Added | SearchExternalFoods | Caller cancellation identity and no retry sleep |
| 38 | TestSearchExternalFoodsPreservesInFlightCallerDeadline | test | backend/internal/externaldata/rate_limit_test.go:155-163 | Added | SearchExternalFoods | Caller deadline identity |
| 39 | TestSearchExternalFoodsRejectsInvalidInputWithoutProviderCalls | test | backend/internal/externaldata/rate_limit_test.go:165-184 | Added | validateExternalSearchQuery | Unknown provider, empty query, page, and page-size matrix |
| 40 | TestSearchExternalFoodsReportsMissingSelectedProviders | test | backend/internal/externaldata/rate_limit_test.go:186-201 | Added | ProviderSet and SearchExternalFoods | One, selected, and both missing providers |
| 41 | TestRateLimitHandlerAdversarialBranches | test | backend/internal/externaldata/rate_limit_test.go:203-243 | Added | Handler, validation, and orchestration branches | Default jitter, invalid config, canceled sleep, provider selection, sleep error, provider cancellation |
| 42 | TestSearchExternalFoodsUsesConfiguredDeadlineAndConcurrentProviderIsolation | test | backend/internal/externaldata/rate_limit_test.go:245-270 | Added | Configure, SearchExternalFoods, handler state | Child deadline and concurrent state access |
| 43 | TestSearchExternalFoodsQuotaResetSkipsBeforeResetAndAllowsAfterReset | test | backend/internal/externaldata/rate_limit_test.go:272-291 | Added | RecordRateLimit and blocked | Reset boundary |
| 44 | TestSearchExternalFoodsNonRetryableUnavailableIsWarningAndSingleCall | test | backend/internal/externaldata/rate_limit_test.go:293-299 | Added | Retry classifier | Permanent failure no-retry |
| 45 | TestSearchExternalFoodsEmitsNoTelemetryAndDoesNotLeakPayloadOrSecrets | test | backend/internal/externaldata/rate_limit_test.go:301-317 | Added | Warning boundary | Secret and raw-payload negative assertion |
| 46 | TestRateLimitHandlerDeterministicRetryDeadlineAndHeaderIsolation | test | backend/internal/externaldata/rate_limit_test.go:319-353 | Added | Backoff, RecordRateLimit, CheckRateLimit | Jitter schedule, retry cap, state isolation |
| 47 | TestExternalSearchAllOutcomesPartialSuccessCancellationAndSafeWarnings | test | backend/internal/externaldata/rate_limit_test.go:355-389 | Added | Combined provider orchestration | Partial success, rate limit, permanent failure, cancellation, warning safety |

inventory_source_count: 47
audited_symbol_count: 47
inventory_complete: true
generated_groupings:
  - "None. Test support types and methods are listed separately because they exercise distinct context, result, and call-count behavior."

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| MaxProviderRetries and warning constants | Fixed retry cap and four stable warning codes. | All classifier paths use the closed vocabulary. | Stateless. | Codes contain no provider-controlled data. | O(1). | Minimal package constants. | Warning assertions cover every code used by Task 245. | PASS |
| ProviderRateLimit | Returns a copyable per-provider quota snapshot. | Zero state is safe for unknown providers. | Embedded in mutex-protected map. | Provider name is selected from fixed orchestration names. | O(1), no I/O. | Small value type. | Check, reset, header, and isolation tests. | PASS |
| rateState | Internal mutable wrapper for one provider. | Zero state preserves no quota block. | Only accessed under RateLimitHandler.mu. | No external serialization. | O(1). | Avoids exposing mutable map internals. | Handler tests exercise updates. | PASS |
| RateLimitHandler | Owns clock, jitter, sleep, deadline, and per-provider states. | Constructor supplies safe defaults; configuration is bounded. | Mutex covers state and mutable config; each handler has isolated state. | No raw provider data stored. | O(1) map operations. | Injectable seams are appropriate for deterministic tests. | Full rate-limit suite and race package pass. | PASS |
| NewRateLimitHandler | Creates an isolated usable handler with five-second default deadline and 100 ms base. | Nil clock and jitter are replaced; default jitter is bounded inclusive. | New map prevents cross-handler state. | No inputs logged or retained except function values. | O(1); no I/O. | Idiomatic dependency injection. | Adversarial default-jitter assertion. | PASS |
| RateLimitHandler.Configure | Accepts only deadlines greater than zero and at most one minute; optional sleep override. | Invalid values leave prior configuration unchanged; nil sleep retains default. | Mutex protects config while searches snapshot it. | Error is fixed and safe. | O(1). | Small explicit configuration API. | Invalid bounds, nil sleep, custom sleep, and deadline tests. | PASS |
| contextSleep | Waits for duration or returns caller context error. | Canceled context wins; timer is stopped on return. | Cancellation applies while waiting; no goroutine leak. | No data crosses boundary. | One bounded timer. | Standard timer/select idiom. | Canceled sleep and retry sleep paths. | PASS |
| RateLimitHandler.CheckRateLimit | Returns a stable value snapshot for one provider. | Unknown provider returns zero value. | Mutex prevents map races and caller cannot mutate internal state. | Only state metadata is exposed. | O(1). | Clear read API. | Concurrent isolation and header/reset assertions. | PASS |
| RateLimitHandler.RecordRateLimit | Parses only remaining and reset headers and stores nonnegative values. | Blank provider errors; nil or malformed headers are ignored without panic; stale values remain conservatively. | Mutex serializes map mutation. | Header allowlist is consumed; no raw header map is stored. | O(1). | Fixed names and numeric parsing are simple. | Valid headers, blank provider, malformed/empty header coverage through full package tests. | PASS |
| RateLimitHandler.blocked | Blocks only during backoff or zero remaining before reset. | Exact reset timestamp is allowed; zero state is unblocked. | Reads under mutex and uses injected clock. | Provider key is fixed at orchestration boundary. | O(1). | Direct predicate. | Before-reset, exact-reset, and retry paths. | PASS |
| RateLimitHandler.backoff | Computes base times 2 to the retry attempt plus jitter and records the window. | Task loop bounds attempt to zero through two; no overflow at the supported cap. | State update is mutex-protected; handler clock/jitter are injected. | No payload data. | O(1), no I/O. | Straightforward exponential calculation. | 150, 300, 600 ms deterministic schedule. | PASS |
| ResultProvider | Requires one call to return records and safe headers on both success and error. | Real clients and fakes implement the same boundary; error results can carry headers. | Call context is propagated by orchestrator. | Contract prevents arbitrary response metadata from being required. | One provider call per attempt. | Removes the repaired optional split and stale records-only branch. | Fake and real provider implementations compile and are exercised. | PASS |
| ProviderResult | Carries records plus projected response headers. | Zero result is safe; error paths may still return headers. | Result is local to one attempt. | Header projection removes secrets before this type crosses boundary. | Records remain bounded by client body/page limits. | Minimal result carrier. | Direct success/error header tests and orchestration test. | PASS |
| projectRateLimitHeaders | Copies exactly two quota header names into a new header map. | Nil or absent headers return an empty map; unrelated headers are dropped. | New map avoids aliasing response headers. | X-Provider-Secret and all non-allowlisted metadata are discarded. | Fixed two-name loop, O(1). | Private helper keeps allowlist centralized. | USDA and OpenFoodFacts success/error tests. | PASS |
| ProviderSet | Names USDA and OpenFoodFacts implementations independently. | Nil slots are representable and handled as warnings. | No shared mutable provider state imposed. | Provider selection is normalized before use. | At most two selected providers. | Explicit typed boundary. | Selection, missing, sibling, and real-client tests. | PASS |
| ExternalDataWarning | Bounded provider outcome with fixed provider, code, and message. | Partial and complete outage outcomes are observable without failing the whole search. | No retained mutable error or payload. | Message is a fixed code, not an error string. | At most two warnings per search plus no unbounded retry output. | Small API DTO. | Warning code and secret negative tests. | PASS |
| NormalizedFoodCandidate | Shared non-persisted candidate shape supports normalizer fields and Task 245 raw nutrient bridge. | Search populates provider identity, external ID, name, nutrients, and image; later normalizer fills canonical fields. | Candidate is local output; no repository write. | Provider text is later normalized by DataNormalizer; raw payload bytes are not copied. | Output bounded by provider body/page limits, though future proxy owns final DTO cap. | Shared type is consumed by normalizer and avoids duplicate candidate shapes. | normalizer.go and normalizer_test.go inspected; rate-limit output and normalizer boundary tests pass. | PASS |
| SearchExternalFoods | Validates, selects, quota-checks, calls providers, retries transient failures, preserves partial success, and returns safe warnings. | Handles available, nil, preflight limited, nonretryable, retryable, quota-blocked, retry-exhausted, provider cancellation, caller cancellation, and sleep errors. | Parent context is checked before each provider and after each attempt; child timeout is canceled; handler state is synchronized; providers are processed independently. | Query is normalized before selection; only fixed provider names and warning codes cross output; no error strings, secrets, payloads, or telemetry are emitted. | At most two providers and four attempts each; retry sleeps are bounded and cancellable; provider clients cap response bodies. | Linear orchestration with no duplicate provider branch; compatibility is provided at client boundary. | Thirteen tests plus full package coverage and race. No untested Task 245 branch remains. | PASS |
| validateExternalSearchQuery | Enforces normalized query, supported provider, one-based page, and provider-specific page-size bounds before I/O. | Empty/invalid/unknown values return typed invalid input; all uses the stricter 100-item bound. | Pure function with no I/O or state. | Uses security.NormalizeInput and does not return rejected raw values. | O(input length), bounded by security normalizer. | Shared provider-neutral validation avoids relying only on real clients. | Matrix covers unknown, empty, page zero, USDA 201, and all 101; zero provider calls observed. | PASS |
| USDAClient.Search | Preserves existing records-only API while delegating to SearchResult. | Returns records on success and the same safe error on failure. | Delegated context and response cleanup remain in SearchResult. | No new metadata leak. | One delegation and no duplicate HTTP request. | Compatibility wrapper is minimal. | Existing USDA Search tests and full package coverage. | PASS |
| USDAClient.SearchResult | Validates before I/O, applies nested deadline, projects headers immediately after response, and returns headers on HTTP/body/decode errors. | Covers success, non-2xx, body read, size, invalid payload, transport, cancellation, and deadline. | defer closes body and cancel; request context propagates caller cancellation. | API key remains request-only; response metadata is allowlisted and diagnostics are categorical. | Response body is limited; one request per attempt. | Result-bearing contract is direct and typed. | New success/error bounded-header test plus existing fake-server suite. | PASS |
| TestUSDASearchResultProjectsBoundedHeadersOnSuccessAndFailure | Regression test requires two allowlisted headers on both outcomes and excludes secret header. | Runs success and 429 subtests. | httptest server and defer Close isolate resources. | Explicit secret-header negative assertion. | Small bounded fixtures. | Focused direct boundary test. | Covers repaired real adapter branch. | PASS |
| OpenFoodFactsClient.Search | Preserves existing records-only API while delegating to SearchResult. | Returns records and safe error unchanged. | Delegated context and cleanup are centralized. | No metadata leak. | One delegation. | Minimal compatibility wrapper. | Existing OpenFoodFacts Search tests and full package coverage. | PASS |
| OpenFoodFactsClient.SearchResult | Validates before I/O, sets caller identification, projects headers on every response outcome, and parses bounded body. | Covers success, status, body read, size, decode, transport, cancellation, and deadline. | defer closes body and cancel; context is passed to HTTP request. | Only allowlisted quota headers leave response; logs contain provider/category/count only. | Response body is limited; one request per attempt. | Same contract as USDA, avoiding optional boundary divergence. | New success/error bounded-header test plus existing client suite. | PASS |
| TestOpenFoodFactsSearchResultProjectsBoundedHeadersOnSuccessAndFailure | Regression test requires two allowlisted headers on success and 429 and excludes secret header. | Runs both statuses with valid or empty bounded body. | httptest server is closed per subtest. | Explicit X-Provider-Secret assertion. | Small deterministic fixtures. | Symmetric provider-boundary coverage. | Covers repaired real adapter branch. | PASS |
| fakeProvider | Mutex-protected queued ResultProvider double with call count. | Empty queues return zero success; queued errors/results drive each branch. | Mutex protects queues and count. | Fixtures contain no production telemetry. | O(1) queue pop per call. | Small deterministic test seam. | Used by retry, partial, invalid-I/O, reset, warning tests. | PASS |
| fakeProvider.SearchResult | Implements result-bearing contract for controlled success/error outcomes. | Error queue has priority; result queue supports headers and records. | Mutex held only over local queue operation; no context wait needed for those tests. | Test-only raw fixture does not enter production warnings. | O(1) pop. | Idiomatic fake. | Header, retry, permanent, partial, and selection tests. | PASS |
| fakeProvider.count | Returns synchronized call count. | Safe after zero or many calls. | Mutex protects read. | No external data. | O(1). | Tiny helper. | All no-retry/no-I/O assertions. | PASS |
| blockingProvider | Test provider that waits for its call context to end. | Returns typed timeout with context cause. | Proves configured child deadline cancellation. | Safe fixed provider value. | One blocked call. | Focused behavior double. | Configured deadline test. | PASS |
| blockingProvider.SearchResult | Honors context Done and returns retryable ProviderErrorTimeout. | No success path is needed for the configured-timeout fixture. | No goroutine remains after context deadline. | No payload. | One wait. | Correct context-aware fake. | TestSearchExternalFoodsUsesConfiguredDeadlineAndConcurrentProviderIsolation. | PASS |
| resultErrorProvider | Test provider carrying a result and error together. | Supports headers with rate-limited error and raw context cancellation. | Single-threaded call count is scoped to tests. | Error cause is not rendered by production warning. | O(1). | Directly models repaired error-result boundary. | Error-header, real-adjacent, and cancellation tests. | PASS |
| resultErrorProvider.SearchResult | Returns configured ProviderResult and error without dropping headers. | Tests both typed provider error and context.Canceled. | No I/O or wait. | Fixtures challenge safe warning behavior. | O(1). | Minimal boundary double. | Header-before-classification and cancellation assertions. | PASS |
| cancelingProvider | Test provider exposing in-flight start and call count. | Waits until canceled and returns typed canceled error. | Started channel synchronizes the test; production call context controls exit. | Fixed provider identity. | One blocked call. | Deterministic in-flight seam. | Caller cancellation and deadline tests. | PASS |
| cancelingProvider.SearchResult | Signals actual dispatch, waits for context, and returns cancellation cause. | Parent cancel and parent deadline produce respective sentinel through orchestrator ctx.Err. | No retry sleep or later provider dispatch after return. | No payload. | One wait. | Direct context propagation fixture. | In-flight cancellation and deadline identity. | PASS |
| TestSearchExternalFoodsRecordsErrorHeadersAndSkipsUntilReset | Requires error-carried quota headers to block a subsequent call. | First typed rate-limit error warns; second call skips. | Handler state is reused deliberately; no goroutine. | Fixed warning assertion. | Two calls at most. | Focused regression for original finding. | Passes with fresh current code. | PASS |
| TestSearchExternalFoodsPropagatesRealProviderHeadersOnErrorAndSuccess | Proves actual USDA and OpenFoodFacts adapters, not only a fake, feed state. | USDA 429 and OpenFoodFacts success are combined; next search skips only USDA. | Two httptest servers are closed; sequential provider isolation is explicit. | Secret header is absent from returned projection. | Tiny valid payload and fixed calls. | Strong boundary integration test. | Covers success/error header propagation and sibling quota isolation. | PASS |
| TestSearchExternalFoodsPreservesInFlightCallerCancellation | Requires context.Canceled identity after dispatch has started and no retry sleep. | Cancellation returns error with prior partial values empty. | Channel synchronization prevents pre-entry false positive. | No error text exposure. | One call. | Direct regression for swallowed cancellation. | Passes current package and race runs. | PASS |
| TestSearchExternalFoodsPreservesInFlightCallerDeadline | Requires context.DeadlineExceeded identity after an in-flight call. | Deadline path is distinct from configured timeout warning. | Provider observes parent deadline and exits. | No payload. | One short call. | Direct deadline identity check. | Passes current package and race runs. | PASS |
| TestSearchExternalFoodsRejectsInvalidInputWithoutProviderCalls | Requires all invalid matrix inputs to return typed invalid input before provider access. | Unknown provider, empty query, page zero, USDA oversize, all oversize. | Pure preflight makes call count zero. | Security normalizer boundary is exercised. | No I/O on failures. | Atomic matrix is easy to extend. | Five cases and shared fake call count. | PASS |
| TestSearchExternalFoodsReportsMissingSelectedProviders | Requires selected nil providers to be observable and sibling success retained. | All, one selected, and both missing cases. | No provider call for nil slot; no state mutation. | Fixed warning fields. | At most two warnings. | Directly closes prior silent omission. | Partial and complete outage assertions. | PASS |
| TestRateLimitHandlerAdversarialBranches | Exercises constructor defaults, invalid configuration, canceled sleep, blank provider, provider selection, sleep failure, and provider cancellation. | Covers rejected and propagated errors as well as OpenFoodFacts-only success. | Canceled sleep exits without waiting; provider cancellation is not retried. | Fixed errors and warnings. | Bounded small fixtures. | Consolidates residual branch checks without production complexity. | Fresh focused command passes. | PASS |
| TestSearchExternalFoodsUsesConfiguredDeadlineAndConcurrentProviderIsolation | Requires configured deadline warning and concurrent per-provider state isolation. | Blocking timeout plus 100 synchronized records/checks per provider. | Race-sensitive state access is tested. | Provider names are fixed. | Bounded 200-iteration test loop. | Clear separation between timeout and state checks. | Package race passes. | PASS |
| TestSearchExternalFoodsQuotaResetSkipsBeforeResetAndAllowsAfterReset | Requires conservative skip before reset and exact-boundary allowance. | Mutable clock moves from 100 to 200. | Handler state intentionally persists between calls. | Numeric header fixtures only. | O(1) calls. | Deterministic reset test. | Passes current package. | PASS |
| TestSearchExternalFoodsNonRetryableUnavailableIsWarningAndSingleCall | Requires permanent unavailable not be retried and not fail the aggregate. | Typed nonretryable error returns empty items and one warning. | No sleep or extra call. | Cause includes a secret-like fixture but is not output. | One call. | Direct classifier assertion. | Passes current package. | PASS |
| TestSearchExternalFoodsEmitsNoTelemetryAndDoesNotLeakPayloadOrSecrets | Requires warning message to exclude raw cause, secret, and payload. | ProviderError cause contains both secret and JSON payload. | No telemetry sink is invoked by orchestrator. | Negative substring assertions and fixed code. | One call. | Focused security regression. | Complements real provider log tests. | PASS |
| TestRateLimitHandlerDeterministicRetryDeadlineAndHeaderIsolation | Requires deterministic retry schedule, cap, and per-provider header state. | Four retryable failures then quota update and clock movement. | Sleep hook advances injected clock; state remains provider-specific. | Warning is fixed; no raw errors. | Exactly four calls and three sleeps. | High-signal deterministic orchestration test. | Passes current package. | PASS |
| TestExternalSearchAllOutcomesPartialSuccessCancellationAndSafeWarnings | Covers available, rate-limited, permanent, cancellation, partial success, and warning safety together. | USDA success plus OpenFoodFacts retries; later permanent failure and pre-canceled context. | Partial results survive sibling failure; cancellation returns sentinel. | Raw payload bytes and secret-like values stay out of warnings. | Four retry max and two providers. | Broad regression complement to focused tests. | Passes current package and full coverage. | PASS |

Mandatory questions were applied to every row: malformed and boundary inputs are covered where relevant; all return paths are intentional; HTTP bodies and timers are released on success, error, and cancellation; context applies during provider execution and retry waiting; handler state is mutex-protected; trusted boundaries use security normalization and allowlisted headers; provider body/page and retry counts are bounded; APIs are minimal and typed; adversarial tests exercise the non-happy paths.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| None | N/A | N/A | No blocking, important, or optional finding remains after repair. | All 15 criteria and all 47 inventory rows pass; fresh focused/full/race/coverage/security/validator evidence is current. | None. |

blocking_findings: 0
important_findings: 0
optional_findings: 0

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| go test -count=1 ./internal/externaldata -run Task 245 named suite with corrected alternation and coverprofile | backend | 0 | PASS | /tmp/task245-named-review.cover; package selection reports 49.8 percent because unrelated externaldata tests are excluded |
| go tool cover -func=/tmp/task245-named-review.cover | backend | 0 | PASS | All ten task-owned rate_limit.go functions are 100.0 percent |
| go test -count=1 ./internal/externaldata -coverprofile=/tmp/task245-full-review.cover | backend | 0 | PASS | Full externaldata package reports 100.0 percent |
| go tool cover -func=/tmp/task245-full-review.cover | backend | 0 | PASS | Total statements 100.0 percent |
| go test -count=1 -race ./internal/externaldata | backend | 0 | PASS | Task package race gate passes |
| go test -count=1 ./... | backend | 1 | FAIL outside Task 245 | internal/app/TestTask240CustomItemErasureIntegration failed because transactional account cleanup left 2 owner custom items; externaldata passed |
| go test -count=1 -race ./... | backend | 1 | FAIL outside Task 245 | internal/app/TestTask240CustomItemErasureIntegration left 2 owner custom items and internal/deletionworker/TestRunAccountDeletionProcessorRetriesAndReportsBoundedMetrics observed unexpected calls/metrics; externaldata passed |
| go vet ./... | backend | 0 | PASS | No vet findings |
| go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./... | backend | 0 | PASS | No reachable vulnerabilities; 18 uncalled module advisories reported by tool |
| gofmt -d six reviewed Go implementation/test files | repository root | 0 | PASS | No formatting diff |
| git diff --check -- backend/internal/externaldata | repository root | 0 | PASS | No whitespace errors |
| python3 scripts/validate-task-list.py | repository root | 0 | PASS | 263 sequential tasks; task 245 remains PREPARED |
| python3 scripts/validate-traceability.py | repository root | 0 | PASS | Traceability validation passed |
| python3 scripts/check.py | repository root | Not rerun | N/A for Task 245 decision | Preparation evidence records the aggregate gate stopping at an unrelated local migration failure: 000003_classifications.up.sql could not find relation food_items, SQLSTATE 42P01; task-specific gates were rerun here |
| python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-245-review.md | repository root | 0 | PASS | Review evidence is structurally valid |

The exact focused command was:

    cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/externaldata -run 'Test(SearchExternalFoods|RateLimitHandler|ExternalSearch|USDASearchResult|OpenFoodFactsSearchResult)' -coverprofile=/tmp/task245-named-review.cover

The full package coverage command was:

    cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/externaldata -coverprofile=/tmp/task245-full-review.cover

The full race command was run after the package race command. Its failures are in unrelated Task 240/deletion-worker surfaces; the Task 245 package passed within that same run.

## 9. Files Inspected and Staleness Fingerprints

Hash the current contents of every reviewed implementation file after review. SHA-256 values below were recomputed after all source inspection and test runs.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| backend/internal/externaldata/rate_limit.go | Task 245 quota handler and orchestration | PASS | SHA-256 | d7649887ac6fb8c960a2df2734f33ca842fc680cce94722a150a52d31a08660b |
| backend/internal/externaldata/rate_limit_test.go | Task 245 orchestration tests | PASS | SHA-256 | f7ea7b2e01e041cbff3c40688e929214b2919669715009a96ff4f339536ef025 |
| backend/internal/externaldata/usda.go | Repaired real ResultProvider boundary | PASS | SHA-256 | 78b198fba01bd7da8ff665b401e5b95b1fa64fdd86fc3df030e9c63864aac116 |
| backend/internal/externaldata/usda_test.go | USDA boundary and security tests | PASS | SHA-256 | d76cfc8a6ae122c8b7a182dcfb85a4c4567803ccf25d810db3addbe5d3364b58 |
| backend/internal/externaldata/openfoodfacts.go | Repaired real ResultProvider boundary | PASS | SHA-256 | e839c6594ab265a11c8ab1b7bb2d295911696a86adc08326bf789e9ba1e0ef57 |
| backend/internal/externaldata/openfoodfacts_test.go | OpenFoodFacts boundary and security tests | PASS | SHA-256 | c41107a76b9651fccb98d4eead9d579cd6d4080923715a756cc8e7e8a6c06742 |
| backend/internal/externaldata/normalizer.go | Shared NormalizedFoodCandidate consumer | PASS — adjacent boundary only | SHA-256 | 3dd9c9fac82375056a88de1aedb5699782dbccba1808d1054868219e833a8fe0 |
| backend/internal/externaldata/normalizer_test.go | Shared candidate boundary tests | PASS — adjacent boundary only | SHA-256 | 7d19639b3bbe3ee241eb2a80ea2cdc2f1f47c4ff34929eca0d5b183a7afa01d9 |
| docs/design/DESIGN-012.md | Source-of-truth design criteria | PASS | SHA-256 | 53ac9bd6a34bd07216666d4beaae6533a0281c905fc2d5c474f48f614746eddf |
| docs/implementation/02_TASK_LIST.md | Task status and criteria | PASS — read-only | SHA-256 | 63856b885be37011e7de9fae2af65cdf390ecceee6c4ac38e875480f66b9a9fa |
| docs/implementation/preparations/task-245.md | Repair scope and prior command manifest | PASS — read-only | SHA-256 | b01bfc16819b4613b1bd65d78397493e280971bb832bf7b5bd5031e29a5f010c |

all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "docs/implementation/reviews/task-245-review.md at pre-repair hash 99b007a0d0a0cbe973073579f24465094e523db4186c8a91ab374b5e2ae39894 was the rejected review and is superseded by this refreshed artifact."
  - "docs/implementation/preparations/task-245.md was checked against current source and fresh command output; its earlier logs were not treated as sufficient without rerunning."

## 10. Coverage and Exceptions

- [x] Required coverage command ran, unless explicitly excepted.
- [x] Report path and observed threshold are recorded.
- [x] Untested branches relevant to changed symbols were inspected.
- [x] Exceptions exactly match the task row and are justified.

coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "/tmp/task245-named-review.cover and /tmp/task245-full-review.cover"
observed_line_coverage: "100.0% for full backend/internal/externaldata; 100.0% for all ten task-owned rate_limit.go functions in named profile"
coverage_passed: true

Coverage finding: No exception is needed or allowed. The named profile reports 100.0 percent for NewRateLimitHandler, Configure, contextSleep, CheckRateLimit, RecordRateLimit, blocked, backoff, projectRateLimitHeaders, SearchExternalFoods, and validateExternalSearchQuery. The full externaldata profile reports 100.0 percent statements, including both repaired real-provider SearchResult paths. The 49.8 percent package total in the narrowed named test command is expected because unrelated normalizer/provider tests are excluded; it is not the task-owned function result.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced. The only cross-file repair is the required real USDA/OpenFoodFacts ResultProvider boundary.
- [x] No source-of-truth documentation was contradicted. DESIGN-012 and the task row remain aligned.
- [x] No generated/cache/build/temporary artifact was unintentionally added by this review.
- [x] Public API additions are necessary and used. ResultProvider is consumed by ProviderSet and the real provider clients; Search remains a compatibility wrapper.
- [x] Duplicate helpers and obsolete aliases were searched for. No records-only optional orchestrator branch remains; all provider orchestration uses SearchResult.
- [x] Error, cleanup, timeout, concurrency, and malformed-input paths were challenged.

Findings: The shared NormalizedFoodCandidate retains the raw Nutrients bridge field for the later normalization/curation workflow, but Task 245 does not log, metric, persist, or expose it. Future Task 248 owns the final external-search DTO cap and safe shaping. This is consistent with the task dependency boundary and is not a Task 245 finding.

## 12. Decision

A task may be PASSED only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking or important finding remains. Those conditions are satisfied.

Before accepting the decision, run:

    python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-245-review.md

decision: "PASSED"
reason: "The repaired real-provider result boundary, quota state, deadlines, retries, cancellation, partial success, bounded warnings, security behavior, and task-owned coverage all pass current evidence with no blocking or important finding."
failed_criteria:
  - "None"
failed_or_unaudited_symbols:
  - "None"
recommended_next_action: "None. Keep task 245 PREPARED until the phase orchestrator applies its normal status workflow; do not change the task list in this review."

## 13. Repair Context

N/A — this is a PASSED re-review. The prior rejected review’s five important findings were verified repaired:

- Real USDA/OpenFoodFacts SearchResult implementations now return allowlisted quota headers on both success and error.
- SearchExternalFoods records error headers before classification and preserves per-provider reset isolation.
- In-flight caller cancellation and deadline identity are returned without retry or warning substitution.
- Nil selected providers emit bounded provider_unavailable warnings.
- Provider-neutral query/provider/page/page-size validation occurs before provider selection or I/O.
- Fresh current coverage reaches 100.0 percent for every Task 245 rate-limit/orchestration function, with no coverage exception.
