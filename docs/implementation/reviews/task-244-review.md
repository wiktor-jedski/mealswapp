# Review Evidence: Task 244 — OpenFoodFacts External Data Client

task_id: 244
component: "OpenFoodFacts External Data Client"
static_aspect: "External provider HTTP client"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-21T10:50:26Z"
review_agent: "Codex independent re-review"
evidence_file: "docs/implementation/reviews/task-244-review.md"
baseline_ref: "Current worktree after numeric/null repair, compared with prior review ea0db088 and preparation hash 38b1783add328db4f8fcfc73325e9072717d141ffcb3d7ad4f36f2b2ff42e9e2"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "/home/wiktor/.agents/skills/code-review-skill/reference/go.md; /home/wiktor/.agents/skills/code-review-skill/reference/security-review-guide.md"
repair_context_required: false

## 1. Task Source

**Description:** Implement the OpenFoodFacts client with bounded query construction, pagination/page boundaries, deadlines, required caller identification, response-size limits, payload decoding, safe provider diagnostics, deterministic ExternalFoodRecord projection, and no raw provider payload persistence.

**Depends On:** 7, 242

**Testing Coverage Exceptions:** None. The focused externaldata race/coverage run reports 100% statement coverage, including the repaired numeric/null/malformed cases.

**Verification Criteria:** Fake-server tests verify URL/query encoding, caller-identification headers, page boundaries, deadlines and cancellation, bounded bodies, malformed and partial payload handling, provider status mapping, no outbound request for invalid input, and deterministic ExternalFoodRecord projection without persisting raw provider payloads. This re-review additionally verifies redirect/SSRF boundaries, safe diagnostics, and interaction with the stabilized Task 243 shared error and record contracts.

## 2. Pre-Review Gates

- [x] Input status is PREPARED.
- [x] Every dependency is PREPARED or PASSED (7 and 242 are PASSED).
- [x] The preparation report claims completion.
- [x] A task-specific baseline/diff is available and trustworthy.
- [x] code-review-skill was invoked exactly once and its relevant Go/security guides were read.
- [x] The reviewer is independent from implementation/repair.
- [x] Review uses current repository state rather than stale logs.
- [x] Reviewer made no production-code changes or task-list status changes.

pre_review_gates_passed: true
blocking_issue: "None."

## 3. Review Baseline and Change Surface

Baseline/reference method: Read the full task preparation report and prior review, checked current task/dependency status, re-read current OpenFoodFacts source and tests line by line, re-read the repaired shared Task 243 contracts and shared test helpers, searched all callers/consumers, and compared current hashes with the prior review and preparation hashes. The numeric/null repair changed the prior implementation hashes 880bef2f/e06b... to 43373399/c13dbb...; current shared usda.go and usda_test.go remain byte-identical to the stabilized Task 243 controls. The repaired source was audited line by line, including strict JSON token classification for nutriments and valid-sibling retention.

Commands used to reconstruct the diff:

    git status --short
    rg -n "OpenFoodFacts|OpenFoodFactsClient|NewOpenFoodFactsClient|ExternalFoodRecord|ExternalFoodPortion|ProviderError" backend docs/design docs/architecture
    git cat-file -t 81ca40ce00cb667ea29243ed2d34068e11229a69
    git show --stat --oneline 81ca40ce00cb667ea29243ed2d34068e11229a69
    nl -ba backend/internal/externaldata/openfoodfacts.go
    nl -ba backend/internal/externaldata/openfoodfacts_test.go
    nl -ba backend/internal/externaldata/usda.go
    nl -ba backend/internal/externaldata/usda_test.go
    sed -n '1,180p' docs/design/DESIGN-012.md
    sed -n '1,100p' docs/architecture/ARCH-012.md
    sed -n '1,120p' backend/internal/curation/validation.go
    sha256sum <reviewed files>

Pre-existing dirty-worktree changes and exclusions:

The worktree contains extensive unrelated tracked and untracked work from Tasks 238–243 and other repository areas. The task-list row 244 remains PREPARED and rows 7 and 242 remain PASSED; those statuses were read but not edited. The task-owned OpenFoodFacts files are untracked relative to HEAD, so the preparation report, previous review hashes, current file hashes, and complete source inspection establish the change surface. The repaired shared usda.go/usda_test.go files are dependency controls and were not edited in this review. A temporary null-probe test had been added during the prior rejection only to reproduce the finding and was removed immediately; it is not part of the reviewed tree. The current repaired regression matrix is in the task-owned test file.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| backend/internal/externaldata/openfoodfacts.go | Task 244 implementation plus repair | HIGH | Configuration/constants, client construction, Search, query validation, wire types, decoding/projection, status/error/log helpers |
| backend/internal/externaldata/openfoodfacts_test.go | Task 244 tests plus repair regressions | HIGH | Request/header, invalid-input, deadline/cancel, body-limit, malformed/partial, projection, status/logging, transport, redirect, and configuration tests/helpers |
| backend/internal/externaldata/usda.go | Stabilized Task 243 shared dependency | HIGH | ExternalSearchQuery, ExternalFoodPortion, ExternalFoodRecord, ProviderError, Error, Unwrap, USDA client interactions |
| backend/internal/externaldata/usda_test.go | Stabilized Task 243 test dependency | HIGH | Shared error assertion, transport/body helpers, and USDA regression tests used to verify cross-client behavior |

No task-owned executable unit was ambiguous. No production caller currently composes OpenFoodFacts; the only direct callers are the focused tests and the future curation boundary documented in DESIGN-012.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | URL and query values are encoded safely. | Source audit and TestOpenFoodFactsSearchEncodesQueryIdentifiesCallerAndProjectsDeterministically. | PASS | url.Values encodes NFC-normalized query text and all fixed query parameters; the test asserts exact decoded values/path and rejects spaces in RequestURI. |
| 2 | Required caller-identification headers are present and safe. | Header assertions and constructor validation. | PASS | Search sets Accept application/json and the validated User-Agent caller ID; caller IDs reject empty, control-bearing, invalid UTF-8, and overlong values. |
| 3 | Page and page-size boundaries are enforced before outbound I/O. | Validation source audit and atomic invalid-input matrix. | PASS | Page 1..10000 and page size 1..100 are enforced; exact maxima reach the fake server and invalid values produce zero calls. |
| 4 | Deadlines and caller cancellation work while waiting for response headers. | Header-stall fake-server test. | PASS | A bounded child context is created and canceled; timeout/cancellation categories and errors.Is sentinels are preserved. |
| 5 | Deadlines and caller cancellation preserve sentinel identity during body reads and transport failures. | Body-stall and direct/wrapped transport sentinel tests. | PASS | Search passes the original body/transport error to transportFailure; request context and direct/wrapped context sentinels map to safe timeout/canceled ProviderErrors with errors.Is identity. |
| 6 | Response size and allocations are finite, safe, and overflow-resistant. | Exact-limit, over-limit, MaxInt64, and counting-infinite-body tests plus source audit. | PASS | A fixed 2 MiB policy cap rejects larger configuration including MaxInt64; max+1 is safe under that cap; an infinite body is read only through the fixed cap plus one byte. |
| 7 | Malformed envelopes and partial/malformed product candidates are rejected or safely omitted. | Malformed envelope/partial matrix, projection audit, and adversarial malformed nutrient cases. | PASS | Envelopes fail categorically; null, overflowed, boolean, object, array, empty, negative, and numeric-field string nutriments drop the candidate; supported label/unit strings are ignored; valid siblings survive and only the dropped count is logged. |
| 8 | Provider statuses map to safe categorical errors and retryability. | Status matrix and safe diagnostics assertions. | PASS | 429 maps rate-limited; 408/5xx unavailable retryable; 4xx rejected permanent; redirects/unexpected statuses unavailable; bodies and URLs are not logged. |
| 9 | Invalid input makes no outbound request. | Atomic fake-server counter matrix and constructor validation. | PASS | Invalid query/provider/page/page-size returns before transport invocation; unsafe endpoint/caller/limits fail construction. |
| 10 | Valid provider data projects deterministically into ExternalFoodRecord. | Deterministic projection test and shared record audit. | PASS | Provider, external ID, normalized name, valid serving pair, numeric nutrients, and public image URL are deterministic; provider order is retained and RawPayload is nil. OpenFoodFacts does not populate Task 243 USDA-specific Portions because its selected wire fields do not supply measured gram weights. |
| 11 | Redirects cannot create cross-host SSRF or leak credentials/API-key data. | Client construction audit, direct redirect response, and real two-server redirect test. | PASS | CheckRedirect is replaced with ErrUseLastResponse and the destination receives zero calls. No API-key or Referer header is added; the endpoint is trusted configuration rather than caller input. |
| 12 | No raw provider payload is returned or persisted. | Projection/source/caller/persistence search. | PASS | RawPayload is never assigned by OpenFoodFacts; no repository or persistence caller was introduced. |
| 13 | The repaired client remains compatible with Task 243 shared contracts. | Shared usda.go/usda_test.go re-read, USDA regression tests, and errors.Is audit. | PASS | ProviderError.Error remains safe with provider fallback; Unwrap exposes only context sentinels; USDA portion/gram-weight behavior and credential/redirect controls remain unchanged and all shared tests pass. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | OpenFoodFacts constants/defaults | configuration logic | backend/internal/externaldata/openfoodfacts.go:24-31 | added/modified | constructor/Search | focused search/config/limit tests |
| 2 | OpenFoodFactsConfig | type | backend/internal/externaldata/openfoodfacts.go:34-43 | added | constructor/tests | configuration tests |
| 3 | OpenFoodFactsClient | type | backend/internal/externaldata/openfoodfacts.go:45-54 | added | constructor/Search/tests | all OpenFoodFacts tests |
| 4 | NewOpenFoodFactsClient | function | backend/internal/externaldata/openfoodfacts.go:56-85 | added/modified | test fixture/future composition | configuration and redirect tests |
| 5 | (*OpenFoodFactsClient).Search | method | backend/internal/externaldata/openfoodfacts.go:87-137 | added/modified | focused tests; no runtime caller | request, context, body, status, transport tests |
| 6 | validateOpenFoodFactsQuery | function | backend/internal/externaldata/openfoodfacts.go:139-155 | added | Search | invalid-input tests |
| 7 | openFoodFactsSearchPayload | type | backend/internal/externaldata/openfoodfacts.go:157-165 | added | decoder | malformed envelope tests |
| 8 | openFoodFactsProduct | type | backend/internal/externaldata/openfoodfacts.go:167-176 | added | decoder/projector | malformed/projection tests |
| 9 | decodeOpenFoodFactsSearch | function | backend/internal/externaldata/openfoodfacts.go:178-196 | added | Search | malformed/partial tests |
| 10 | projectOpenFoodFactsProduct | function | backend/internal/externaldata/openfoodfacts.go:198-268 | added/modified | decoder/tests | deterministic/optional/malformed tests |
| 11 | decodeJSONString | function | backend/internal/externaldata/openfoodfacts.go:270-279 | added | projector | projection tests |
| 12 | containsUnsafeProviderText | function | backend/internal/externaldata/openfoodfacts.go:281-293 | added | projector/caller validation | safety/config tests |
| 13 | validCallerID | function | backend/internal/externaldata/openfoodfacts.go:295-299 | added | constructor | configuration tests |
| 14 | mapOpenFoodFactsStatus | function | backend/internal/externaldata/openfoodfacts.go:301-314 | added | Search | status tests |
| 15 | (*OpenFoodFactsClient).transportFailure | method | backend/internal/externaldata/openfoodfacts.go:316-326 | added/modified | Search | body/transport/context tests |
| 16 | (*OpenFoodFactsClient).failure | method | backend/internal/externaldata/openfoodfacts.go:328-336 | added | Search/helpers | status/log tests |
| 17 | (*OpenFoodFactsClient).logDropped | method | backend/internal/externaldata/openfoodfacts.go:338-344 | added | Search | partial/log tests |
| 18 | openFoodFactsError | function | backend/internal/externaldata/openfoodfacts.go:346-350 | added | constructor/helpers | provider error tests |
| 19 | ExternalFoodRecord | shared type | backend/internal/externaldata/usda.go:49-61 | Task 243 shared contract consumed here | OpenFoodFacts/USDA/future normalizer | OpenFoodFacts/USDA tests |
| 20 | ProviderError | shared type | backend/internal/externaldata/usda.go:80-88 | Task 243 shared contract consumed here | both provider clients/callers | provider error tests |
| 21 | (*ProviderError).Error | method | backend/internal/externaldata/usda.go:92-98 | Task 243 shared behavior control | all provider errors | provider error assertions |
| 22 | (*ProviderError).Unwrap | method | backend/internal/externaldata/usda.go:100-102 | Task 243 shared behavior control | errors.Is callers | context tests |
| 23 | TestOpenFoodFactsSearchEncodesQueryIdentifiesCallerAndProjectsDeterministically | test function | backend/internal/externaldata/openfoodfacts_test.go:27-63 | added | OpenFoodFacts Search | itself |
| 24 | TestOpenFoodFactsSearchRejectsInvalidInputBeforeOutboundRequest | test function | backend/internal/externaldata/openfoodfacts_test.go:65-86 | added | Search validation | itself |
| 25 | TestOpenFoodFactsSearchHonorsDeadlineAndCallerCancellation | test function | backend/internal/externaldata/openfoodfacts_test.go:88-114 | added | Search context | itself |
| 26 | TestOpenFoodFactsSearchPreservesContextErrorsWhileReadingBody | test function | backend/internal/externaldata/openfoodfacts_test.go:116-150 | added by repair | Search body mapper | itself |
| 27 | TestOpenFoodFactsSearchPreservesTransportContextSentinels | test function | backend/internal/externaldata/openfoodfacts_test.go:152-176 | added by repair | transportFailure | itself |
| 28 | TestOpenFoodFactsSearchBoundsBodiesAndHandlesMalformedOrPartialPayloads | test function | backend/internal/externaldata/openfoodfacts_test.go:178-222 | added | Search/decode/logging | itself |
| 29 | TestOpenFoodFactsSearchEnforcesFiniteAllocationBound | test function | backend/internal/externaldata/openfoodfacts_test.go:224-244 | added by repair | Search body limit | itself |
| 30 | TestProjectOpenFoodFactsProductHandlesOptionalAndMalformedFields | test function | backend/internal/externaldata/openfoodfacts_test.go:247-282 | added | projector | itself |
| 31 | TestProjectOpenFoodFactsProductRejectsMalformedNumericNutriments | test function | backend/internal/externaldata/openfoodfacts_test.go:284-323 | added by repair | projector numeric token policy | itself |
| 32 | TestOpenFoodFactsSearchMapsStatusesAndLogsOnlyBoundedMetadata | test function | backend/internal/externaldata/openfoodfacts_test.go:325-358 | added | status/failure/logging | itself |
| 33 | TestOpenFoodFactsSearchHandlesRequestTransportAndBodyReadFailures | test function | backend/internal/externaldata/openfoodfacts_test.go:360-397 | added | Search transport/body cleanup | itself |
| 34 | TestNewOpenFoodFactsClientRejectsUnsafeConfiguration | test function | backend/internal/externaldata/openfoodfacts_test.go:400-421 | added/modified by repair | constructor bounds | itself |
| 35 | testOpenFoodFactsCallerID | test fixture | backend/internal/externaldata/openfoodfacts_test.go:25 | added | constructor/header tests | all request tests |
| 36 | newTestOpenFoodFactsClient | test helper | backend/internal/externaldata/openfoodfacts_test.go:424-431 | added | all client tests | all OpenFoodFacts tests |
| 37 | validOpenFoodFactsQuery | test helper | backend/internal/externaldata/openfoodfacts_test.go:433-435 | added | all Search tests | all Search tests |
| 38 | contextBlockingBody | test type | backend/internal/externaldata/openfoodfacts_test.go:437-440 | added by repair | body cancellation test | body cancellation test |
| 39 | (*contextBlockingBody).Read | test method | backend/internal/externaldata/openfoodfacts_test.go:442-450 | added by repair | body cancellation test | body cancellation test |
| 40 | (*contextBlockingBody).Close | test method | backend/internal/externaldata/openfoodfacts_test.go:452 | added by repair | HTTP cleanup | body cancellation test |
| 41 | countingInfiniteBody | test type | backend/internal/externaldata/openfoodfacts_test.go:454 | added by repair | allocation bound test | allocation bound test |
| 42 | (*countingInfiniteBody).Read | test method | backend/internal/externaldata/openfoodfacts_test.go:456-462 | added by repair | allocation bound test | allocation bound test |
| 43 | (*countingInfiniteBody).Close | test method | backend/internal/externaldata/openfoodfacts_test.go:464 | added by repair | HTTP cleanup | allocation bound test |
| 44 | paddedOpenFoodFactsPayload | test helper | backend/internal/externaldata/openfoodfacts_test.go:466-477 | added by repair | exact-limit test | allocation bound test |
| 45 | validOpenFoodFactsPayload | test fixture | backend/internal/externaldata/openfoodfacts_test.go:479-498 | added | deterministic/malformed tests | request/projection tests |

inventory_source_count: 45
audited_symbol_count: 45
inventory_complete: true
generated_groupings:
  - "None; runtime symbols and every task-owned OpenFoodFacts test/helper unit are listed individually. Shared roundTripFunc, failingBody, signalingBody, and assertProviderError were re-read as unchanged Task 243 test dependencies and are not counted as Task 244 changes."

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| OpenFoodFacts constants/defaults | Fixed endpoint, field projection, deadline, page maximum, and finite 2 MiB policy. | Zero config defaults; policy cap is explicit. | Immutable after construction. | HTTPS default and trusted endpoint validation. | Fixed body policy makes max+1 safe. | Minimal provider constants. | Exact/over-limit tests. | PASS |
| OpenFoodFactsConfig | Carries validated caller, endpoint, deadline, body limit, client, logs. | Invalid caller/endpoint/deadline/limit rejected. | No mutable per-call state. | Caller and endpoint are constrained. | Public limit cannot exceed fixed cap. | Small config surface. | Unsafe configuration matrix; null nutrient is outside config. | PASS |
| OpenFoodFactsClient | Holds immutable validated request dependencies. | Constructor rejects unsafe configuration. | Per-call child context; cloned HTTP client; response close deferred. | Redirect policy is local to clone. | Finite body setting. | Private fields preserve invariants. | Used by all tests. | PASS |
| NewOpenFoodFactsClient | Builds safe configured client and cannot mutate caller HTTP client redirect policy. | Defaults and all invalid settings handled. | Shallow HTTP client clone; no goroutine ownership. | Rejects userinfo/query/fragment; disables redirects; no credential fields. | Rejects MaxInt64 and all values above fixed cap. | Idiomatic constructor. | Config and real redirect tests. | PASS |
| (*OpenFoodFactsClient).Search | One validated, bounded request returns safe records/errors. | Handles validation, URL/request creation, transport, statuses, body reads, size, decode, partial logs. | Context.WithTimeout and defer cancel; response close on all response paths; body errors preserve context sentinels. | URL values/header safe; no Referer/API key; redirects stopped. | Read capped at max+1; no retries; finite allocation input. | Linear and focused. | All focused tests pass; null nutrient coercion is exposed through projector. | PASS |
| validateOpenFoodFactsQuery | Normalizes provider/query and checks one-based page/page-size boundaries. | Empty/control/oversize/wrong provider and boundary values rejected. | Pure/no I/O. | Stops unsafe input before URL/transport. | Bounded normalization. | Reuses shared security normalizer. | Atomic no-outbound matrix. | PASS |
| openFoodFactsSearchPayload | Required legacy envelope with pointers for presence validation. | Missing/negative/zero invalid metadata rejected; nil products rejected. | Decode-local. | No raw body field. | Input body is fixed bounded before decode. | Private wire type. | Malformed envelope matrix. | PASS |
| openFoodFactsProduct | Selected raw provider fields only. | Optional fields can be omitted; malformed required fields reach projector. | Decode-local; no persistence. | Image and text normalized before record. | Wire values bounded by response cap. | Narrow projection schema. | Optional/partial tests. | PASS |
| decodeOpenFoodFactsSearch | Returns ordered valid records and dropped count or invalid-payload error. | Envelope invalid fails; bad siblings drop; good siblings survive. | Pure after body read; no shared state. | Raw body not returned. | Product allocation bounded by finite body, though metadata consistency/duplicate-key rejection is optional. | Preserves provider order. | Envelope/partial tests and strict malformed-nutrient sibling matrix; no page metadata cross-check. | PASS |
| projectOpenFoodFactsProduct | Converts a candidate into safe deterministic record. | Required code/name/nutrients invalidates; optional serving/image can be omitted. | Pure; no I/O. | Provider identifier/text/image/units normalized. | Nutrient map bounded by response. | Clear projection. | Strict token classification rejects null, overflow, boolean/object/array, empty, negative, and numeric-field strings; supported textual label/unit metadata is ignored. | PASS |
| decodeJSONString | Accepts only non-empty JSON strings. | Missing/wrong/blank values reject. | Pure. | Prevents type coercion for IDs/names/images. | Tiny bounded helper. | Idiomatic. | Projection matrix. | PASS |
| containsUnsafeProviderText | Rejects invalid UTF-8/control/format characters. | Safe nutrient keys/caller IDs pass; unsafe reject. | Pure. | Prevents header/log key injection. | Bounded field usage. | Centralized predicate. | Direct safety assertions. | PASS |
| validCallerID | Bounded safe visible caller ID. | Empty/oversize/control invalid. | Pure. | Header injection prevented. | Max 256 bytes. | Minimal predicate. | Constructor matrix. | PASS |
| mapOpenFoodFactsStatus | Closed provider status mapping. | 429, 408, 4xx, 5xx, redirects handled. | Pure. | Never reads body for diagnostics. | O(1). | Explicit switch. | Status matrix. | PASS |
| (*OpenFoodFactsClient).transportFailure | Maps only safe context causes and otherwise unavailable. | request context and direct/wrapped sentinels handled; arbitrary cause discarded. | Preserves cancellation while waiting/reading; no goroutine leak. | Does not expose URL-bearing transport error. | No extra I/O. | Shared Task 243 pattern reused. | Direct/wrapped sentinel and generic error tests. | PASS |
| (*OpenFoodFactsClient).failure | Creates categorical ProviderError and bounded log. | Code/status/retryability retained. | No owned resources. | Constant provider/code/status only; no body/query/secret. | Constant-size log. | Small helper. | Status/log tests. | PASS |
| (*OpenFoodFactsClient).logDropped | Logs only count/provider metadata. | Positive dropped count logged. | Sink errors intentionally ignored like shared provider diagnostics. | No product values. | O(1). | Minimal. | Partial log assertions. | PASS |
| openFoodFactsError | Constructs provider-specific safe error. | Provider identity fixed. | Cause only safe context sentinel supplied by mapper. | No raw provider data. | Small. | Thin adapter. | Error assertions. | PASS |
| ExternalFoodRecord | Shared Task 243 record includes serving, Portions, image, raw slot. | OpenFoodFacts fills supported fields and leaves Portions/RawPayload nil; USDA still fills portions/raw payload. | Value passed to later normalizer. | Shared downstream validation remains intact. | No raw OpenFoodFacts retention. | Reused shared type. | Both OpenFoodFacts and USDA tests. | PASS |
| ProviderError | Shared typed safe error with provider/status/retryability/cause. | Error and Unwrap preserve safe category/context. | errors.Is works for mapper-provided sentinels. | No cause text exposed. | Small. | Correct shared contract. | USDA/OpenFoodFacts tests. | PASS |
| (*ProviderError).Error | Stable provider/category text. | USDA fallback retained; OpenFoodFacts identity set. | Pure. | No secrets/body/URL. | Small. | Idiomatic. | Provider assertions. | PASS |
| (*ProviderError).Unwrap | Exposes cause for errors.Is only. | Context sentinels match. | Preserves Task 243 cancellation contract. | Cause is a closed sentinel. | O(1). | Idiomatic. | Both client context tests. | PASS |
| TestOpenFoodFactsSearchEncodesQueryIdentifiesCallerAndProjectsDeterministically | Proves exact request and valid projection. | Max page/page-size and Unicode/space query covered. | Test server closes via defer. | Checks headers/encoding/raw nil. | Small fixture. | Direct assertion. | Valid sibling and textual metadata projection is covered by the focused malformed-nutrient regression too. | PASS |
| TestOpenFoodFactsSearchRejectsInvalidInputBeforeOutboundRequest | Proves validation short-circuit. | Empty/long/control/provider/page/size matrix. | Atomic counter. | No unsafe input reaches transport. | O(1) cases. | Table-like loop. | Strong no-outbound evidence. | PASS |
| TestOpenFoodFactsSearchHonorsDeadlineAndCallerCancellation | Proves header-wait context mapping. | Timeout and explicit cancel. | Fake server waits on request context. | Safe errors only. | Small. | Deterministic channels. | Complements body test. | PASS |
| TestOpenFoodFactsSearchPreservesContextErrorsWhileReadingBody | Proves repaired body-read sentinel mapping. | Deadline and caller cancel after headers. | Body blocks on request context; cleanup deferred. | Cause text not exposed. | No large allocation. | Focused regression. | Passes. | PASS |
| TestOpenFoodFactsSearchPreservesTransportContextSentinels | Proves direct/wrapped transport sentinel mapping. | Deadline/cancel direct and wrapped. | No leaked goroutines. | Error string excludes transport details. | Tiny. | Table-driven subtests. | Passes. | PASS |
| TestOpenFoodFactsSearchBoundsBodiesAndHandlesMalformedOrPartialPayloads | Proves size and envelope/partial behavior. | Too large, malformed/missing/negative envelope, mixed candidates. | Server/client body cleanup deferred. | Logs only count. | Small except bounded test separate. | Good adversarial matrix. | The focused numeric-token regression adds null/overflow and valid-sibling survival coverage. | PASS |
| TestOpenFoodFactsSearchEnforcesFiniteAllocationBound | Proves finite policy and read cap. | Exact 2 MiB succeeds; infinite body stops at cap+1. | Body close deferred. | No body retention after failure. | Read count bounded. | Direct cap regression. | Does not measure allocator internals, but fixed finite cap is source-proven. | PASS |
| TestProjectOpenFoodFactsProductHandlesOptionalAndMalformedFields | Proves projection filters optional/bad fields. | Invalid IDs/names/keys, optional serving/image. | Pure. | Provider text/image safe. | Small maps. | Direct table. | Numeric-token edge cases are covered by the dedicated strict malformed-nutrient regression. | PASS |
| TestProjectOpenFoodFactsProductRejectsMalformedNumericNutriments | Proves strict numeric token policy. | Null, overflow, boolean, object, array, empty, negative, and numeric-field string tokens reject the candidate; label/unit string metadata is ignored. | Pure subtests; no resources. | Confirms malformed provider data cannot become nutrition values. | Tiny table. | Focused regression. | Passes and verifies supported textual metadata plus the invalid-candidate path. | PASS |
| TestOpenFoodFactsSearchMapsStatusesAndLogsOnlyBoundedMetadata | Proves mapping and log secrecy. | Redirect, 4xx, timeout, 429, 5xx. | Deferred server cleanup. | Body/URL/USDA text excluded. | Constant log. | Clear matrix. | Passes. | PASS |
| TestOpenFoodFactsSearchHandlesRequestTransportAndBodyReadFailures | Proves generic errors, invalid request, redirect non-following. | Generic transport/body, direct 3xx, real two-server redirect. | Bodies close; destination counter. | No cause/credential/referrer leak. | Small. | Good boundary test. | Passes. | PASS |
| TestNewOpenFoodFactsClientRejectsUnsafeConfiguration | Proves endpoint/caller/deadline/body policy. | Missing/unsafe/negative/over-limit/MaxInt64 and defaults. | Cloned client redirect behavior. | Userinfo/query/file endpoint rejected. | Fixed cap. | Focused constructor test. | Passes. | PASS |
| testOpenFoodFactsCallerID | Safe bounded caller fixture. | Valid header identity. | N/A fixture. | No secret. | Constant. | Simple. | Used throughout. | PASS |
| newTestOpenFoodFactsClient | Central test constructor. | Fails test on invalid config. | Test helper no persistent resources. | Uses safe fixture. | Small. | Avoids duplicate setup. | Used broadly. | PASS |
| validOpenFoodFactsQuery | Valid normalized request fixture. | One-based page/size. | Immutable return value. | Safe provider token. | Tiny. | Simple. | Used broadly. | PASS |
| contextBlockingBody | Body test double that follows request context. | Blocks until deadline/cancel. | No leaked goroutine; Close is no-op. | No provider data. | One read. | Correct fake. | Body context test. | PASS |
| (*contextBlockingBody).Read | Returns context sentinel after unblock. | Deadline/cancel propagated. | Selectively closes started channel. | N/A. | Constant. | Idiomatic fake. | Body test. | PASS |
| (*contextBlockingBody).Close | Implements cleanup contract. | Always nil. | No resources. | N/A. | O(1). | Minimal. | HTTP defer path. | PASS |
| countingInfiniteBody | Infinite provider body test double. | Never EOF; read count measures bound. | No shared cross-test state. | N/A. | Bounded by client. | Simple. | Allocation bound test. | PASS |
| (*countingInfiniteBody).Read | Fills requested buffer and increments count. | Arbitrary read sizes. | Test-local state. | N/A. | Exposes read bound. | Valid Reader behavior. | Allocation bound test. | PASS |
| (*countingInfiniteBody).Close | Implements cleanup contract. | Always nil. | No resources. | N/A. | O(1). | Minimal. | HTTP defer path. | PASS |
| paddedOpenFoodFactsPayload | Creates exact policy-sized valid body. | Fails if base exceeds target. | Test-local allocation. | N/A. | 2 MiB bounded fixture. | Test helper. | Exact-limit test. | PASS |
| validOpenFoodFactsPayload | Valid selected-field provider fixture. | Includes numeric/text nutriments, serving/image. | Immutable literal. | Safe public image. | Small. | Deterministic fixture. | Request/projection tests. | PASS |

Mandatory audit conclusion: all mandatory source and test units pass. The repaired numeric-token policy rejects null and malformed numeric values, ignores only supported textual metadata, and preserves valid sibling products. Transport/body cancellation, finite limits, redirect/SSRF, safe diagnostics, and the stabilized Task 243 shared contracts also pass. One optional page-metadata consistency hardening item remains documented below.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| optional | backend/internal/externaldata/openfoodfacts.go:180-196 | decodeOpenFoodFactsSearch | The decoder validates local page metadata shape but does not compare returned page/page_size/page_count with the requested values and accepts duplicate envelope keys. | A provider can return a structurally valid but mismatched envelope; no current test exercises this. The current client performs one bounded page request and later pagination orchestration owns cross-request policy. | Optional hardening: pass expected page/page size to the decoder and reject inconsistent metadata/duplicate keys, or document this as the later pagination-composition boundary. It is not a blocker for the current one-page bounded client. |

blocking_findings: 0
important_findings: 0
optional_findings: 1

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| sed/nl/rg source and caller discovery commands | repository root | 0 | PASS | Current OpenFoodFacts, shared externaldata, callers, design, and test dependencies inspected line by line. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -v ./internal/externaldata | backend | 0 | PASS | All current OpenFoodFacts tests, including strict malformed numeric/null cases, and all stabilized USDA tests passed. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race -coverprofile=/tmp/task-244-final-rereview-cover.out ./internal/externaldata && go tool cover -func=/tmp/task-244-final-rereview-cover.out | backend | 0 | PASS | Focused externaldata race clean; statement coverage 100%; every OpenFoodFacts production function 100%, including numeric-token branches. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./... | backend | 0 | PASS | Full backend tests passed. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -race ./... | backend | 143 | NOT COMPLETED | The full-repository race run was terminated after the unrelated internal/queue Redis restart integration test stalled in an idle futex; focused externaldata race passed. This is an environment/test-harness limitation, not a Task 244 finding. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./... | backend | 0 | PASS | No vet diagnostics. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./... | backend | 0 | PASS | No vulnerabilities in called/imported code; 18 uncalled module advisories reported. |
| python3 scripts/validate-task-list.py && python3 scripts/validate-traceability.py | repository root | 0 | PASS | 263 sequential tasks and traceability passed; task list untouched. |
| npx --no-install redocly lint api/openapi.yaml | repository root | 0 | PASS | Valid; existing ignored OAuth callback 302-only warning only. |
| python3 scripts/check.py | repository root | 0 | PASS | Aggregate repository checks completed successfully, including backend/frontend/UAT checks; existing documented coverage deviations unchanged. |
| gofmt -d backend/internal/externaldata/openfoodfacts.go backend/internal/externaldata/openfoodfacts_test.go backend/internal/externaldata/usda.go backend/internal/externaldata/usda_test.go | repository root | 0 | PASS | No formatting diff. |
| git diff --check | repository root | 0 | PASS | No whitespace diagnostics. |
| python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-244-review.md | repository root | 0 | PASS | Refreshed evidence file validates structurally. |

No task-relevant command was omitted. The standalone full-repository race command did not complete because of an unrelated queue/Redis integration stall; focused task-package race and the aggregate check completed successfully.

## 9. Files Inspected and Staleness Fingerprints

Hash the current contents of every reviewed implementation/dependency file after review.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| backend/internal/externaldata/openfoodfacts.go | OpenFoodFacts client, request bounds, decoding, projection, diagnostics | Optional response metadata consistency hardening only | SHA-256 | 433733998bf73d63dd7dce66152ff24bb6ffb3cd4b80817d595b78949758b53d |
| backend/internal/externaldata/openfoodfacts_test.go | OpenFoodFacts adversarial and regression tests | Numeric/null/malformed regression coverage present | SHA-256 | c13dbb6309a8041476a5d32b03db432ac8b2d8a6863c84897d07f9a977ea98b4 |
| backend/internal/externaldata/usda.go | Stabilized shared record/error contracts and USDA client | No Task 243 regression finding | SHA-256 | 21f29cb5c6d1c528339cce5aa4d78622bfa6bcc9051d9a0888dc2c51cee07beb |
| backend/internal/externaldata/usda_test.go | Shared Task 243 helpers/regression controls | No regression finding | SHA-256 | b9045286353943098e3b3f4fa4576bb0954312ee9191a4ff622bad7e7a9dfd25 |
| docs/implementation/preparations/task-244.md | Current repair preparation and claimed evidence | Re-read; current source/hash values supersede embedded older claims where applicable | SHA-256 | 38b1783add328db4f8fcfc73325e9072717d141ffcb3d7ad4f36f2b2ff42e9e2 |
| docs/implementation/reviews/task-244-review.md | Prior rejection evidence and current refreshed evidence | Prior pre-refresh content was re-read; this row records the prior evidence hash because a file cannot contain its own final hash without recursion | SHA-256 | ea0db08816d9cba6c8f005192efa814746759192a704a6f7604fb3f10107fd83 |
| docs/implementation/02_TASK_LIST.md | Current PREPARED input/dependency status control | Inspected only; no status edit | SHA-256 | 6435b85a88d9c5df80176cea7b2edc1fd109e5ad8f54a3be78f3cca4d7f691d1 |

all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "docs/implementation/reviews/task-244-review.md hash c23ede7c... is superseded by this refreshed evidence."
  - "docs/implementation/preparations/task-244.md contains prior repair claims; current hashes and independent audit above are authoritative."
  - "docs/implementation/reviews/task-243-review.md was checked; stabilized shared file hashes match usda.go 21f29cb5... and usda_test.go b9045286..."

## 10. Coverage and Exceptions

- [x] Required coverage command ran.
- [x] Report path and observed threshold are recorded.
- [x] Untested branches relevant to changed symbols were inspected.
- [x] Exceptions exactly match the task row; none are claimed.

coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "/tmp/task-244-final-rereview-cover.out"
observed_line_coverage: "100% of statements in backend/internal/externaldata"
coverage_passed: true

Coverage finding: Statement coverage is 100% and includes the repaired JSON null, overflow, non-number, empty-token, numeric-field string, supported textual metadata, and valid-sibling cases.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced.
- [x] No source-of-truth documentation was contradicted.
- [x] No generated/cache/build/temporary artifact was unintentionally added.
- [x] Public API additions are necessary and used.
- [x] Duplicate helpers and obsolete aliases were searched for.
- [x] Error, cleanup, timeout, concurrency, and malformed-input paths were challenged.

Findings: All prior important findings are repaired: context sentinels survive transport/body reads, the fixed finite body policy rejects over-limit/MaxInt64 values while bounding reads, and strict nutriment token classification rejects null/malformed numeric data without losing valid siblings. Redirects are not followed, no API key/Referer is introduced, and safe diagnostics exclude bodies/URLs/causes. Query encoding, headers, page boundaries, invalid no-outbound behavior, status mapping, deterministic valid projection, textual metadata handling, nil RawPayload, and Task 243 USDA behavior all pass. Only the optional response page-metadata/duplicate-key hardening item remains.

## 12. Decision

A task may be PASSED only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains.

Before accepting the decision, run:

    python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-244-review.md

decision: "PASSED"
reason: "Current OpenFoodFacts implementation and tests satisfy all mandatory criteria with no blocking or important findings. The repaired numeric/null/malformed handling is strict, valid siblings survive, textual metadata remains supported, and the stabilized Task 243 transport/error/record controls remain intact."
failed_criteria:
  []
failed_or_unaudited_symbols:
  []
recommended_next_action: "No repair required. Optionally harden response page metadata and duplicate-key rejection at the later pagination-composition boundary."
    
## 13. Repair Context

Not applicable — this independent re-review is PASSED and no further repair context is required.
