# Review Evidence: Task 243 — USDA External Data Client

```yaml
task_id: 243
component: "Phase 08 USDA External Data Client"
static_aspect: "DESIGN-012: USDAClient"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-21T09:10:09Z"
review_agent: "Codex independent re-review"
evidence_file: "docs/implementation/reviews/task-243-review.md"
baseline_ref: "81ca40ce00cb667ea29243ed2d34068e11229a69 plus preparation and prior rejection evidence"
baseline_confidence: "MEDIUM"
code_review_skill_invoked: true
relevant_language_guide: "reference/go.md; reference/security-review-guide.md; cross-cutting async-concurrency and error-handling guidance"
repair_context_required: false
```

## 1. Task Source

**Description:** Phase 08: implement the configured USDA FoodData Central client with bounded query construction, pagination, deadlines, credential loading, response-size limits, payload decoding, and safe provider diagnostics.

**Depends On:** 7 (`PASSED`), 242 (`PASSED`)

**Testing Coverage Exceptions:** None.

**Verification Criteria:** Fake-server tests verify URL/query encoding, configured API-key handling without secret logging, page boundaries, deadlines and cancellation, bounded bodies, malformed and partial payload rejection, provider status mapping, no outbound request for invalid input, and deterministic `ExternalFoodRecord` projection including volume portions with gram weights.

## 2. Pre-Review Gates

- [x] Input status is `PREPARED`.
- [x] Every dependency is `PREPARED` or `PASSED`.
- [x] The preparation report claims completion.
- [x] A task-specific baseline/diff is available; the prior rejection and repair manifest identify the changed surface.
- [x] `code-review-skill` was invoked exactly once and its relevant Go and security guides were read.
- [x] The reviewer is independent from implementation/repair.
- [x] Review uses current repository state rather than stale logs.
- [x] Reviewer made no production-code or task-list-status changes.

```yaml
pre_review_gates_passed: true
blocking_issue: "None. The prior three important findings were re-audited and repaired."
```

## 3. Review Baseline and Change Surface

Baseline/reference method: `HEAD` is `81ca40ce00cb667ea29243ed2d34068e11229a69`; the preparation report and prior rejection evidence establish the untracked Task 243 files and repaired symbol surface. The current implementation hashes match the refreshed preparation manifest. Later Task 244 shared-field additions in `usda.go` were re-audited; its new OpenFoodFacts files remain excluded.

Commands used to reconstruct scope:

```bash
git rev-parse HEAD
git status --short --untracked-files=all
rg -n "USDA|ExternalFoodRecord|ProviderError|NewUSDAClient|LoadUSDAAPIKey|ExternalSearchQuery" backend docs/design docs/architecture --glob '*.go' --glob '*.md'
nl -ba backend/internal/externaldata/usda.go
nl -ba backend/internal/externaldata/usda_test.go
sha256sum backend/internal/externaldata/usda.go backend/internal/externaldata/usda_test.go
```

Pre-existing dirty-worktree changes and exclusions: unrelated tracked/untracked Phase 08 work from Tasks 238–242 was preserved. Later Task 244 `openfoodfacts.go` and `openfoodfacts_test.go` were inspected only for shared-type consumers and are excluded from this task decision. The task-list row was observed as `PREPARED` and was not edited.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| `backend/internal/externaldata/usda.go` | Task 243 preparation manifest plus repaired implementation; shared `ImageURL`/provider-error fields from later Task 244 re-audited | MEDIUM | USDA contracts/configuration, credential loader, HTTP search, decoder, portions, statuses, errors |
| `backend/internal/externaldata/usda_test.go` | Task 243 preparation manifest plus repaired adversarial tests | HIGH | USDA tests, fixtures, fake transport/body, assertion helpers |

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | URL/query encoding | Fake server and request inspection | PASS | `TestUSDASearchEncodesQueryKeyPaginationAndProjectsDeterministically` verifies NFC-normalized query, exact path, encoded URI, API key, page number, page size, and no unexpected query fields. |
| 2 | API-key loading without secret logging | Environment, error, log, body, transport, and redirect checks | PASS | Loader trims valid keys and rejects missing/control values without echoing them; provider bodies, URL-bearing transport causes, errors, and logs contain neither the key nor provider detail. Redirect test reaches no destination, so no credential-bearing `Referer` is emitted. |
| 3 | Page boundaries | Exact min/max and no-I/O counter tests | PASS | Pages 1 and 10,000 and sizes 1 and 200 are accepted and sent; zero, above-maximum, and invalid sizes are rejected before transport. |
| 4 | Deadline while waiting | Blocking server and sentinel assertion | PASS | `TestUSDASearchHonorsDeadlineAndCallerCancellation` returns `ProviderErrorTimeout` and preserves `context.DeadlineExceeded`. |
| 5 | Caller cancellation while waiting | Blocking server and sentinel assertion | PASS | The same test returns `ProviderErrorCanceled` and preserves `context.Canceled`. |
| 6 | Cancellation while reading | 200 headers flushed, body stalled, deadline/cancel sentinel assertions | PASS | `TestUSDASearchPreservesContextErrorsWhileReadingBody` cancels while `Read` is blocked; both deadline and caller-cancel cases map categorically and preserve the matching sentinel through `errors.Is`. |
| 7 | Bounded response bodies | Oversized body and constructor upper-bound/overflow tests | PASS | Constructor enforces a fixed 2 MiB ceiling independent of caller input; exact ceiling is accepted, ceiling-plus-one and `math.MaxInt64` are rejected, `max+1` is overflow-safe, and `io.LimitReader` caps `io.ReadAll` allocation rather than trusting `Content-Length`. |
| 8 | Malformed/partial payload rejection | Malformed JSON, missing fields, invalid values, duplicate nutrients, and portions | PASS | The fake-server matrix rejects malformed JSON/food, incomplete envelope/foods/identity/nutrients/serving pairs, invalid and duplicate nutrients, and invalid portions. |
| 9 | Provider status mapping | Fake status table and safe logs | PASS | 400/401/404 map permanent rejection, 408/5xx map retryable unavailable, 429 maps retryable rate limit, and diagnostics remain categorical. |
| 10 | No outbound invalid input | Atomic server counter | PASS | Empty/long/control queries, wrong provider, invalid pages, and invalid sizes make zero outbound requests. |
| 11 | Deterministic record projection | Exact record and raw-payload comparison | PASS | Provider/ID/name/serving/nutrient/raw projection matches; output portions are deterministically ordered by unit, amount, and gram weight. |
| 12 | Volume portions with gram weights | Unit fallbacks, positive values, and tie ordering | PASS | Abbreviation, measure-name, and dissemination-text fallbacks retain positive amount and provider-measured gram weight without mass conversion. |
| 13 | No unbounded pagination loop | Request-count and page inspection | PASS | USDA performs exactly one caller-selected page request; retry orchestration belongs to Task 245 and no pagination loop is introduced. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | USDA configuration constants | constants | `usda.go:23-29` | added | constructor/search | config tests |
| 2 | `ExternalSearchQuery` | type | `usda.go:33-38` | added | Search/future proxy | query tests |
| 3 | `ExternalFoodPortion` | type | `usda.go:42-46` | added | record/normalizer | portion tests |
| 4 | `ExternalFoodRecord` | type | `usda.go:50-60` | added; later `ImageURL` overlay | Search/future normalizer | projection test; shared OpenFoodFacts tests excluded |
| 5 | `ProviderErrorCode` | type | `usda.go:64` | added | errors/status | error tests |
| 6 | `ProviderError*` constants | constants | `usda.go:68-78` | added | failure paths | status/config tests |
| 7 | `ProviderError` | type | `usda.go:81-88` | added; later provider field overlay | errors/future retry | error tests; shared OpenFoodFacts consumers excluded |
| 8 | `(*ProviderError).Error` | method | `usda.go:91-98` | added; later provider formatting | logs/upstream | safe errors |
| 9 | `(*ProviderError).Unwrap` | method | `usda.go:100-102` | added | `errors.Is` | context tests |
| 10 | `USDAConfig` | type | `usda.go:104-113` | added | constructor | config tests |
| 11 | `USDAClient` | type | `usda.go:115-124` | added | future proxy/retry | search tests |
| 12 | `LoadUSDAAPIKey` | function | `usda.go:126-134` | added | future bootstrap | credential test |
| 13 | `NewUSDAClient` | function | `usda.go:136-165` | repaired | bootstrap | unsafe-config/redirect tests |
| 14 | `(*USDAClient).Search` | method | `usda.go:167-209` | repaired | future proxy/retry | fake-server/body tests |
| 15 | `validateUSDAQuery` | function | `usda.go:211-229` | added | Search | invalid-input test |
| 16 | `usdaSearchPayload` | type | `usda.go:232-239` | added | decoder | envelope tests |
| 17 | `usdaFood` | type | `usda.go:241-250` | added | decoder | food tests |
| 18 | `usdaNutrient` | type | `usda.go:252-258` | added | decoder | nutrient tests |
| 19 | `usdaMeasure` | type | `usda.go:260-267` | added | portion tests | decoder |
| 20 | `usdaMeasureUnit` | type | `usda.go:269-274` | added | measure/decoder | fallback tests |
| 21 | `decodeUSDASearch` | function | `usda.go:276-330` | added | Search | malformed/projection tests |
| 22 | `finitePositive` | function | `usda.go:332-336` | added | decoder | quantity tests |
| 23 | `mapUSDAStatus` | function | `usda.go:338-351` | added | Search | status test |
| 24 | `(*USDAClient).transportFailure` | method | `usda.go:353-363` | repaired | Search Do/body errors | context/transport/body tests |
| 25 | `(*USDAClient).failure` | method | `usda.go:365-373` | added | safe paths | diagnostics test |
| 26 | ProviderError error assertion | compile-time | `usda.go:375-376` | added | Go error interface | build/vet |
| 27 | `testUSDAKey` | test constant | `usda_test.go:24` | added | secret tests | sentinel checks |
| 28 | `TestLoadUSDAAPIKey` | test | `usda_test.go:26-42` | added | loader | trim/control |
| 29 | `TestUSDASearchEncodesQueryKeyPaginationAndProjectsDeterministically` | test | `usda_test.go:44-84` | added | Search/decoder | URL/projection |
| 30 | `TestUSDASearchRejectsInvalidInputBeforeOutboundRequest` | test | `usda_test.go:86-108` | added | validation | no-I/O matrix |
| 31 | `TestUSDASearchHonorsDeadlineAndCallerCancellation` | test | `usda_test.go:110-136` | added | context | header-wait |
| 32 | `TestUSDASearchPreservesContextErrorsWhileReadingBody` | test | `usda_test.go:138-187` | added; repaired | body-read context | deadline/cancel |
| 33 | `TestUSDASearchDoesNotFollowCredentialBearingRedirects` | test | `usda_test.go:189-214` | added; repaired | redirect boundary | cross-host/Referer |
| 34 | `TestUSDASearchBoundsResponseAndRejectsMalformedOrPartialPayloads` | test | `usda_test.go:216-244` | added; repaired | body/decoder | size/malformed |
| 35 | `TestUSDASearchHandlesRequestTransportAndBodyReadFailures` | test | `usda_test.go:246-273` | added | errors | generic failures |
| 36 | `TestDecodeUSDASearchAcceptsEmptyResultsAndOrdersPortionTies` | test | `usda_test.go:275-293` | added | decoder | empty/ties |
| 37 | `TestUSDASearchMapsProviderStatusesAndLogsOnlyBoundedMetadata` | test | `usda_test.go:295-332` | added | status/logs | safe diagnostics |
| 38 | `TestNewUSDAClientRejectsUnsafeConfiguration` | test | `usda_test.go:334-355` | added; repaired | constructor | limits/defaults |
| 39 | `newTestUSDAClient` | helper | `usda_test.go:357-364` | added | fake tests | setup |
| 40 | `validUSDAQuery` | helper | `usda_test.go:366-368` | added | search tests | valid fixture |
| 41 | `searchPayload` | helper | `usda_test.go:370-372` | added | payload tests | envelope fixture |
| 42 | `assertProviderError` | helper | `usda_test.go:374-380` | added | error tests | typed assertion |
| 43 | `roundTripFunc` | type | `usda_test.go:382` | added | injected clients | transport tests |
| 44 | `roundTripFunc.RoundTrip` | method | `usda_test.go:384` | added | HTTP transport | transport tests |
| 45 | `failingBody` | test type | `usda_test.go:386` | added | body test | body failure |
| 46 | `failingBody.Read` | test method | `usda_test.go:388-391` | added | `io.ReadAll` | body error |
| 47 | `failingBody.Close` | test method | `usda_test.go:391` | added | deferred cleanup | cleanup |
| 48 | `signalingBody` | test type | `usda_test.go:393-396` | added; repaired | body cancellation test | signaling fixture |
| 49 | `(*signalingBody).Read` | test method | `usda_test.go:398-405` | added; repaired | body cancellation test | read/cancel |
| 50 | `validUSDAPayload` | test constant | `usda_test.go:407-425` | added | projection/size | valid fixture |

```yaml
inventory_source_count: 50
audited_symbol_count: 50
inventory_complete: true
generated_groupings:
  - "The two USDA constant blocks are grouped by semantic block; every individual constant was inspected. The failingBody and signalingBody fixtures are grouped as test-only body adapters, with each method audited separately."
```

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| USDA configuration constants | Default HTTPS endpoint, 5-second deadline, fixed 2 MiB body ceiling, page size 200. | Zero values default; max body is finite and independent of caller input; `max+1` is safe under constructor invariant. | Immutable; no resources. | Default endpoint is HTTPS; redirect policy is installed for every client. | Fixed response/read cap. | Small explicit policy surface. | Exact/over/MaxInt64 config cases pass. | PASS |
| `ExternalSearchQuery` | Carries query, provider, one-based page, positive size. | Invalid values are rejected by validator. | Plain value; stateless. | Provider allowlist and normalization precede I/O. | Bounded normalization. | Minimal shared contract. | Valid/invalid query tests pass. | PASS |
| `ExternalFoodPortion` | Retains positive amount, unit, measured gram weight. | Invalid finite/positive values reject; no mass conversion. | Output-owned value; no shared state. | Unit remains provider data for later normalization. | Body-bounded slice capacity. | Correct density evidence type. | Fallback and tie tests pass. | PASS |
| `ExternalFoodRecord` | Projects identity, serving, nutrients, portions, and copied raw food JSON. | Required USDA fields populate; later `ImageURL` overlay remains zero for USDA and is outside this task. | Per-call maps/slices/raw copies; no shared mutation. | Raw bytes never logged. | Input and raw data are bounded by body cap. | Necessary downstream contract. | Exact projection passes; shared OpenFoodFacts consumer is excluded from this decision. | PASS |
| `ProviderErrorCode` | Typed categorical failure vocabulary. | Client uses fixed constants; no raw status/body category is exposed. | Immutable. | Error category has no raw cause. | Constant-sized. | Idiomatic typed mapping. | `errors.As` assertions pass. | PASS |
| `ProviderError*` constants | Classify config, input, status, payload, size, timeout, cancel. | Mapping is explicit, including body-read context failures. | Immutable. | No body/URL/key. | O(1). | Supports later retry layer. | All used categories covered. | PASS |
| `ProviderError` | Carries category/status/retryability and private safe cause. | USDA causes are only context sentinels; arbitrary transport causes are dropped. | No resources; immutable by convention. | Error text excludes URL/query/key/body/cause. | Constant-sized. | Small module error; shared provider field overlay re-audited. | Error/context tests pass. | PASS |
| `(*ProviderError).Error` | Returns stable provider-safe diagnostic. | Provider-aware value defaults to `usda`; no nil receiver is expected. | No state/resources. | No cause, query, URL, or credential. | O(1). | Idiomatic method. | Safe error assertions pass. | PASS |
| `(*ProviderError).Unwrap` | Preserves only safe context sentinel matching. | Nil cause yields no chain. | No resources. | Prevents URL-bearing causes from surfacing. | O(1). | Idiomatic. | Body and wait-phase context tests pass. | PASS |
| `USDAConfig` | Holds credential, endpoint, deadline, body cap, HTTP client, log sink. | Zero defaults; negative and over-ceiling body values reject; endpoint credentials/query/fragments reject. | Caller client/sink are shared under their contracts; constructor clones client. | Endpoint is configuration, not query input; redirects are forcibly disabled. | Caller cannot raise the 2 MiB allocation/read ceiling. | Clear input type with enforced safety invariants. | Unsafe config matrix passes. | PASS |
| `USDAClient` | Stores validated immutable search configuration. | Constructor establishes fields; valid calls only. | Per-call child context and body close are deferred; Search does not mutate fields. | Key is sent only to configured endpoint; cloned client cannot follow redirects. | One bounded request and body read. | Minimal no-retry client; retry belongs to Task 245. | Race suite and redirect/body tests pass. | PASS |
| `LoadUSDAAPIKey` | Loads and trims required environment key safely. | Empty/whitespace/control values reject without echo; valid key returns. | Environment read only. | No logging or wrapping of secret. | O(key length). | Simple loader. | Credential matrix passes. | PASS |
| `NewUSDAClient` | Validates config and applies defaults. | Rejects malformed/unsafe endpoint, invalid deadline/body values, and body ceiling overflow inputs. | No owned network resource; clones HTTP client and overwrites `CheckRedirect` with `ErrUseLastResponse`. | Cross-host, downgrade, loopback, and other redirect targets are never requested; no redirect `Referer` can leak the query key. | Fixed ceiling makes `max+1` overflow impossible and bounds `ReadAll`. | Correct defensive constructor. | Real TLS-to-loopback redirect and max/overflow tests pass. | PASS |
| `(*USDAClient).Search` | Validates, encodes one GET, applies context/body cap, maps and decodes. | Invalid input returns before I/O; non-2xx maps; body-read failures preserve deadline/cancel sentinels; malformed payload rejects. | Child context is canceled; response body closes on all response paths; no goroutine or loop. | URL query is encoded; redirect-following is disabled; arbitrary causes are not logged/returned. | `io.LimitReader` plus fixed ceiling bounds body allocation; one request only. | Straight-line, idiomatic HTTP flow. | Focused USDA tests and 25x adversarial repetition pass. | PASS |
| `validateUSDAQuery` | Reuses Task 242 normalization and USDA/page-size bounds. | Empty/long/control/wrong provider/page/size fail before network access. | Stateless. | Untrusted query/provider cannot reach URL unless accepted. | O(query length), page bounded 10,000. | Reuses shared security helper. | Atomic no-outbound matrix passes. | PASS |
| `usdaSearchPayload` | Requires envelope fields and nonnil foods. | Missing/null/negative fields reject; empty result envelope is valid. | Decoder-local. | Provider bytes are not logged. | Body cap bounds source bytes; records remain body-bounded. | Small DTO. | Envelope/missing-fields cases pass. | PASS |
| `usdaFood` | Captures identity/description/serving/nutrients/measures. | Missing/empty/incomplete/negative fields reject; optional serving pair is all-or-nothing. | Decoder-local. | Provider text is not logged. | Per-food bytes are bounded by response. | Minimal DTO. | Food/serving cases pass. | PASS |
| `usdaNutrient` | Named, unit-qualified, finite nonnegative value. | Empty/negative/nonfinite/semantic duplicate reject. | Decoder-local. | Names/units remain in output, not diagnostics. | Map follows bounded body. | Qualified key avoids ordinary unit collisions. | Nutrient/adversarial cases pass. | PASS |
| `usdaMeasure` | Measure label, amount, and provider gram weight. | Abbreviation/name/dissemination fallback; empty/nonpositive/nonfinite values reject. | Decoder-local. | Provider text remains data, not logs. | Portion slice follows bounded body. | Explicit fallback order. | Portion tests pass. | PASS |
| `usdaMeasureUnit` | Preferred/fallback labels. | Empty all labels ultimately rejects. | Decoder-local. | Provider text not logged. | Trivial. | Minimal DTO. | Fallback cases pass. | PASS |
| `decodeUSDASearch` | Fails closed and sorts portions deterministically. | Required/finite/duplicate semantic checks work; empty results accepted; duplicate JSON keys/page metadata consistency are optional hardening outside criteria. | No external resources; output owns maps/slices/raw copies. | Generic errors/raw values never logged. | O(body plus maps/slices and sort); body cap bounds source and output factor. | Cohesive parser; no conversion or density assumption. | Malformed/empty/projection/tie tests pass. | PASS |
| `finitePositive` | Accepts only finite positive numbers. | Zero/negative/NaN/Inf reject. | Stateless. | Prevents invalid quantities downstream. | O(1). | Idiomatic predicate. | Indirect quantity coverage passes. | PASS |
| `mapUSDAStatus` | Maps 429, 408/5xx, 4xx, and unexpected statuses. | Tested mappings pass; redirects are returned rather than followed and map to safe unavailable. | Stateless. | Status only; non-2xx body is not read/logged. | O(1). | Simple switch. | Status table and redirect test pass. | PASS |
| `(*USDAClient).transportFailure` | Drops arbitrary Do/read causes while preserving request context sentinels. | Request/body timeout and cancellation map to timeout/canceled; generic causes map unavailable. | No resources; observes both request context and safe wrapped error. | Prevents URL-bearing `url.Error` and provider detail from surfacing. | O(1). | Reused for wait and body-read paths. | Header/body context and generic transport tests pass. | PASS |
| `(*USDAClient).failure` | Returns categorical error and bounded log metadata. | Fixed categories; log-sink errors are intentionally ignored after attempting metadata-only emission. | Synchronous sink; sink owns concurrency. | No URL/query/key/body/cause. | Constant event plus small map. | Appropriate failure boundary. | Memory-sink tests pass. | PASS |
| ProviderError error assertion | Compile-time error contract. | Compile-time only. | N/A — no runtime resource. | N/A. | N/A. | Idiomatic. | Build/vet pass. | PASS |
| `testUSDAKey` | Test secret sentinel. | Stable test constant. | Test-only. | Enables no-leak assertions. | O(1). | Necessary fixture. | Used in secret tests. | PASS |
| `TestLoadUSDAAPIKey` | Verifies loader trim/rejection. | Missing/space/newline cases and valid key. | `t.Setenv` restores state; not parallel. | Checks no secret in error. | Tiny. | Focused. | Passes. | PASS |
| `TestUSDASearchEncodesQueryKeyPaginationAndProjectsDeterministically` | Verifies fake request and record. | Method/path/query/NFC/raw/nutrients/portions. | Server closes; no shared mutation. | Key is checked only at expected fake endpoint; redirect coverage is separate. | One bounded response. | Strong integration test. | Passes. | PASS |
| `TestUSDASearchRejectsInvalidInputBeforeOutboundRequest` | Proves pre-I/O invalid matrix. | Empty/long/NUL/provider/page/size. | Atomic counter and deferred server close. | Invalid values do not reach transport/logs. | Small table. | Direct regression. | Passes. | PASS |
| `TestUSDASearchHonorsDeadlineAndCallerCancellation` | Proves wait-phase context mapping. | Deadline/cancel sentinels pass. | Blocking handler observes context; result collected. | Only safe sentinel exposed. | Bounded timing. | Correct wait test. | Passes. | PASS |
| `TestUSDASearchPreservesContextErrorsWhileReadingBody` | Proves cancellation after 200 headers while `Read` blocks. | Deadline and caller cancel map to distinct categories and `errors.Is` sentinels. | Signaling body and fake server coordinate without leaked goroutines. | No transport/body detail escapes. | Tiny table; repeated 25 times clean. | Direct regression for prior finding. | Passes. | PASS |
| `TestUSDASearchDoesNotFollowCredentialBearingRedirects` | Proves redirect policy at a real HTTP boundary. | TLS source redirects to HTTP destination; destination receives zero calls and no key-bearing Referer. | Servers defer close; atomic call count. | Blocks cross-host and scheme-downgrade redirect. | One response. | Strong SSRF/secret regression. | Passes. | PASS |
| `TestUSDASearchBoundsResponseAndRejectsMalformedOrPartialPayloads` | Proves body limit and decoder matrix. | Oversized small limit plus malformed/partial/duplicate/invalid cases. | Servers defer close; no retained provider body in errors/logs. | Bodies are not diagnostic output. | Small fixtures. | Good table. | Passes. | PASS |
| `TestUSDASearchHandlesRequestTransportAndBodyReadFailures` | Proves generic transport/body/status errors. | Generic causes are dropped; direct redirect response maps safely. | Injected clients and response body close via Search defer. | Provider detail and key do not leak. | Tiny fakes. | Good generic coverage. | Passes. | PASS |
| `TestDecodeUSDASearchAcceptsEmptyResultsAndOrdersPortionTies` | Proves empty and tie sort. | Empty foods and same-unit order. | Pure local decoder. | Test-local data. | Small JSON. | Direct parser test. | Passes. | PASS |
| `TestUSDASearchMapsProviderStatusesAndLogsOnlyBoundedMetadata` | Proves status/log safety. | Representative 4xx/408/429/500. | Test-local server/sink. | Body/URL/key sentinels absent. | Small table. | Clear regression. | Passes. | PASS |
| `TestNewUSDAClientRejectsUnsafeConfiguration` | Proves constructor defaults, redirect override, and body ceiling. | Missing/control key, malformed URL, userinfo/query, negative values, ceiling-plus-one, and MaxInt64 reject; exact ceiling/defaults pass. | No resources; verifies cloned client is not default client and has redirect policy. | Unsafe endpoint components and credentials do not enter requests. | Direct bound/overflow cases. | Focused constructor regression. | Passes. | PASS |
| `newTestUSDAClient` | Builds configured fake client. | Fails test on invalid setup. | Helper owns no resources. | Test key only. | O(1). | Idiomatic. | Used throughout. | PASS |
| `validUSDAQuery` | Valid query fixture. | In bounds. | Immutable. | Fixed provider token. | O(1). | Minimal. | Used throughout. | PASS |
| `searchPayload` | Wraps food JSON in envelope. | Allows malformed food fixture. | String-only. | Test-only bytes. | Small concatenation. | Clear. | Used by malformed table. | PASS |
| `assertProviderError` | Asserts category/status/retryability. | Reports wrong/missing typed error. | Helper only. | Does not print raw provider data beyond safe typed error. | O(1). | Useful assertion. | Used in error cases. | PASS |
| `roundTripFunc` | Function adapter for RoundTripper. | Returns injected response/error. | Test-only. | Enables safe transport probes. | O(1). | Idiomatic. | Transport tests. | PASS |
| `roundTripFunc.RoundTrip` | Delegates exactly once. | Caller controls result. | No resources. | Test-only. | O(1). | Minimal. | Indirectly covered. | PASS |
| `failingBody` | Deterministic body-error fixture. | Zero-byte read returns a provider-detail error; close succeeds. | No blocking or shared state. | Provider detail is used only as a non-leak sentinel. | O(1). | Minimal test adapter. | Generic body-error test passes. | PASS |
| `failingBody.Read` | Exercises `io.ReadAll` error handling. | Returns the intended non-context error. | No resources or cancellation. | Cause is discarded by `transportFailure`. | O(1). | Correct fake reader. | Body-error test passes. | PASS |
| `failingBody.Close` | Satisfies response-body cleanup contract. | Returns nil. | Search's deferred close reaches it. | No data. | O(1). | Necessary adapter. | Indirectly covered by body test. | PASS |
| `signalingBody` | Wraps a real body and signals entry into `Read`. | Preserves the underlying body result while exposing read-start synchronization. | Test coordination is channel-based and bounded; no goroutine ownership. | No provider data added. | O(1) wrapper state. | Focused cancellation fixture. | Body-read cancellation test passes repeatedly. | PASS |
| `(*signalingBody).Read` | Signals the first body read and delegates exactly once. | Handles first and subsequent reads without closing the signal twice. | Select/default avoids blocking; underlying response context supplies cancellation. | No logging or data transformation. | O(1) per read. | Idiomatic test wrapper. | Deadline/caller-cancel body test passes. | PASS |
| `validUSDAPayload` | Valid bounded USDA fixture. | Parses and exceeds tiny test limit. | Immutable test string. | No real secret/PII. | Small. | Reusable. | Projection/size tests. | PASS |

Audit conclusion: all 50 inventory rows were inspected with callers, dependencies, design source, error paths, cleanup, context, concurrency, security boundaries, allocations/I/O, API necessity, and adversarial tests. No unbounded pagination loop, SQL/filesystem/command operation, or client-owned goroutine was found. The prior body-read, response-limit, and redirect findings are closed by current code and tests.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| [optional] | `usda.go:278-280` | `decodeUSDASearch` | Duplicate JSON object names and inconsistent response page metadata are accepted by standard last-value-wins decoding. | No task criterion requires rejecting duplicate JSON names or cross-checking `currentPage`/`totalPages` against the requested page; all required malformed/partial cases reject. | Optional hardening for a future contract revision; does not block Task 243. |

```yaml
blocking_findings: 0
important_findings: 0
optional_findings: 1
```

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| `go test -count=1 -run '^(TestLoadUSDAAPIKey|TestUSDASearch|TestDecodeUSDASearch|TestNewUSDAClient)' -v ./internal/externaldata` | `backend` | 0 | PASS | USDA-only focused suite, including repaired adversarial cases. |
| `go test -count=25 -run '^(TestUSDASearchHonorsDeadlineAndCallerCancellation|TestUSDASearchPreservesContextErrorsWhileReadingBody|TestUSDASearchDoesNotFollowCredentialBearingRedirects|TestNewUSDAClientRejectsUnsafeConfiguration)$' ./internal/externaldata` | `backend` | 0 | PASS | 25 repeated security/cancellation/limit runs. |
| `go test -count=1 -v ./internal/externaldata` | `backend` | 0 | PASS | USDA suite plus colocated later-task package tests. |
| `go test -count=1 -race -coverprofile=/tmp/task-243-re-review-cover.out ./internal/externaldata` | `backend` | 0 | PASS | Race clean; package 100.0% statements. |
| `go tool cover -func=/tmp/task-243-re-review-cover.out` | `backend` | 0 | PASS | Every current USDA production function 100.0%; total includes excluded Task 244 functions. |
| `go vet ./internal/externaldata` | `backend` | 0 | PASS | Focused vet. |
| `gofmt -d backend/internal/externaldata/usda.go backend/internal/externaldata/usda_test.go` | repository root | 0 | PASS | No formatting diff. |
| `git diff --check` | repository root | 0 | PASS | No whitespace errors. |
| `go test -count=1 ./...` | `backend` | 0 | PASS | Full backend tests. |
| `go test -count=1 -race ./...` | `backend` | 0 | PASS | Full backend race suite. |
| `go vet ./...` | `backend` | 0 | PASS | Full backend vet. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `backend` | 0 | PASS | No vulnerabilities in called code; 18 uncalled module advisories. |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS | 263 sequential tasks; Task 243 remains `PREPARED`. |
| `python3 scripts/validate-traceability.py` | repository root | 0 | PASS | Traceability passed. |
| `npx --no-install redocly lint api/openapi.yaml` | repository root | 0 | PASS with existing warning | Valid OpenAPI; existing ignored OAuth callback 302-only warning. |
| `python3 scripts/check.py` | repository root | 0 | PASS | Aggregate traceability, local stack, backend, frontend, browser, coverage, and security gates passed; documented repository exceptions and ignored warning unchanged. |

## 9. Files Inspected and Staleness Fingerprints

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| `backend/internal/externaldata/usda.go` | USDA contracts/config/search/decoder/portions/errors and shared overlay | Current repaired implementation; no blocking/important finding | SHA-256 | `21f29cb5c6d1c528339cce5aa4d78622bfa6bcc9051d9a0888dc2c51cee07beb` |
| `backend/internal/externaldata/usda_test.go` | USDA fake-server and adversarial tests | Current repaired test surface; no blocking/important finding | SHA-256 | `b9045286353943098e3b3f4fa4576bb0954312ee9191a4ff622bad7e7a9dfd25` |

```yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The prior Task 243 rejection evidence recorded usda.go d602aa1620a6d278057c14055b3dc25008f2f63fd1f219d3f1995ace0e45b028 and usda_test.go fd2905b61aaa0cd7c4641aecd5fec5df8db476c754f4b38f6b63221643e4cd4f. Both implementation files changed during repair; all affected USDA symbols and callers were re-reviewed."
```

## 10. Coverage and Exceptions

- [x] Required focused coverage command ran.
- [x] Report path and observed threshold are recorded.
- [x] Untested branches relevant to changed symbols were inspected.
- [x] Exceptions exactly match the task row: no Task 243 exception is allowed.

```yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "/tmp/task-243-re-review-cover.out"
observed_line_coverage: "100.0% current externaldata package; USDA functions 100.0%; current total includes excluded Task 244 functions"
coverage_passed: true
```

Coverage finding: statement coverage is supplementary; manual review and repeated fake-server tests specifically cover body-read cancellation, redirect blocking, API-key safety, and finite response limits.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced by Task 243; later Task 244 files are excluded.
- [x] No source-of-truth documentation was contradicted.
- [x] No generated/cache/build/temporary artifact was unintentionally added; the coverage profile is under `/tmp`.
- [x] Public API additions are necessary for planned proxy/retry/normalization callers, although no production caller exists yet.
- [x] Duplicate helpers and obsolete aliases were searched for; Task 242 normalization is reused.
- [x] Error, cleanup, timeout, concurrency, and malformed-input paths were challenged, including cancellation while waiting and reading.

Findings: query encoding, page validation, response close, generic error sanitization, semantic malformed checks, measured portions, statuses, no-outbound invalid input, redirect blocking, context preservation, and body bounds pass. No pagination loop, SQL, filesystem, command, or client-owned goroutine was found.

## 12. Decision

A task may be `PASSED` only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains.

Before accepting the decision, run:

```bash
python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-243-review.md
```

```yaml
decision: "PASSED"
reason: "All acceptance criteria and 46 audited symbols pass; prior body-read cancellation, response-limit overflow, and redirect/API-key leakage findings are closed, with no blocking or important finding remaining."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "None for Task 243; retain evidence and leave task-list status unchanged for the orchestrator."
```

## 13. Repair Context

Not applicable: the repaired PREPARED task passes independent re-review.
