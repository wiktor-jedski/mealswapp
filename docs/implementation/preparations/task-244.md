# Task 244 Preparation — OpenFoodFacts External Data Client

## Outcome and task control

- Task: **244 — Phase 08 OpenFoodFacts External Data Client**.
- Result: **implemented, repaired after two independent reviews, and verified; all three important review findings and every verification clause have direct adversarial evidence**.
- Fixed implementation reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Preparation date: 2026-07-21, Europe/Warsaw.
- Dependencies 7 and 242 were re-read from `docs/implementation/02_TASK_LIST.md` and remain `PASSED`.
- Task 244 remains `PREPARED`. This repair did not edit its status or any task-list content.
- Repair-baseline and final task-list SHA-256: `6435b85a88d9c5df80176cea7b2edc1fd109e5ad8f54a3be78f3cca4d7f691d1`.
- The phase-orchestrator skill requires delegation when a writable subagent is available. No writable subagent capability was available in this session, so the parent executed the one-task preparation contract directly.

## Baseline and scope ownership

- Baseline commit: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Latest repair source: `docs/implementation/reviews/task-244-review.md`, SHA-256 `ea0db08816d9cba6c8f005192efa814746759192a704a6f7604fb3f10107fd83`.
- Repair-baseline hashes: `openfoodfacts.go` = `056fe7becab7af1686d9fc8b7a07689e1a5611e554a08bde18059d7b94a7ff8f`; `openfoodfacts_test.go` = `9a772040077a0127070607181db05db2d3ab6fa58f672d01d2fa1ea5baebf531`; this evidence = `31f7b1b4db0f0a27f6c7d964ac6793684ded7382f95e08b214e34600cfa89c35`.
- Numeric-repair baseline hashes: `openfoodfacts.go` = `880bef2ff01da3159737997b981e9853c1f77e44bedbf661b566990350f2b7cf`; `openfoodfacts_test.go` = `e06b838279f8e7b5c6c80c758463f50ca35f2ccbbcda9218ad27717cfda6c9a6`; this evidence = `631a71d215f598ac04909b0bcefef6ed0952325ecc678a80ed49dc3e1b55415f`.
- Baseline `git status --short` already contained extensive Tasks 238–243 and unrelated tracked/untracked changes in API, application, cache, curation, custom-item, deletion, HTTP, repository, search, security, userdata, database migration, frontend, scripts, design, task-list, open-issue, preparation, and review paths.
- `backend/internal/externaldata/openfoodfacts.go`, `backend/internal/externaldata/openfoodfacts_test.go`, and this evidence file did not exist at baseline.
- Task 243's current repair controls were captured and remained byte-identical through this repair: `backend/internal/externaldata/usda.go` = `21f29cb5c6d1c528339cce5aa4d78622bfa6bcc9051d9a0888dc2c51cee07beb`; `backend/internal/externaldata/usda_test.go` = `b9045286353943098e3b3f4fa4576bb0954312ee9191a4ff622bad7e7a9dfd25`.
- No unrelated file was cleaned, reverted, staged, overwritten, or reformatted.
- Task 244 intentionally does not compose clients, add retries/rate-limit state, normalize nutrient aliases/units, add routes, or persist records. Those surfaces belong to Tasks 245–253.

## Design, requirements, and provider material inspected

- `docs/design/DESIGN-012.md`: `OpenFoodFactsClient`, `ExternalSearchQuery`, `ExternalFoodRecord`, bounded request construction, pagination, payload parsing, timeout, invalid-payload, unavailable, and rate-limited states.
- `docs/architecture/ARCH-012.md`, `docs/architecture/01_SOFT_ARCH_DESIGN.md`, and `docs/architecture/02_APPENDIX_A.md`: OpenFoodFacts external boundary, graceful provider failure, and later retry ownership.
- `docs/requirements/01_SOFT_REQ_SPEC.md`: SW-REQ-033 normalized downstream storage and SW-REQ-055 administrator external-data curation.
- Task 242's `security.NormalizeInput` contract: bounded normalized query/provider/page, provider identifiers, serving units, provider text, and safe image URLs.
- Task 243's `USDAClient`, shared provider error vocabulary, record contract, fake-server style, and preparation evidence.
- Current official OpenFoodFacts [API introduction](https://openfoodfacts.github.io/openfoodfacts-server/api/) and [official Python SDK](https://openfoodfacts.github.io/openfoodfacts-python/): custom `User-Agent` identification is required; read searches are rate limited; v2/v3 do not provide plain-text search; the supported SDK text search returns the legacy `count`, `page`, `page_count`, `page_size`, and `products` envelope. The client therefore uses the documented legacy text-search endpoint with a narrow `fields` projection.

## Security assessment

The `golang-security` skill was applied in coding mode.

- Trust boundary: query/provider/page/page-size, configuration, status codes, headers, redirects, and all provider response bytes are untrusted.
- Input validation occurs before `http.Client.Do`; invalid query/provider/page/page-size and missing/unsafe caller IDs cannot produce outbound requests.
- Endpoint configuration accepts only absolute HTTP(S) URLs with no credentials, query, or fragment. Production defaults to HTTPS; HTTP remains injectable for loopback fake servers.
- Redirects are rejected by a copied `http.Client`, preventing provider redirects from changing the destination host. The caller's client is not mutated.
- Denial-of-service controls: query/page/page-size bounds, one request per call, a per-call context deadline, redirect rejection, a fixed 2 MiB maximum configuration policy, an overflow-safe `io.LimitReader(max+1)` under that policy, bounded selected fields, bounded caller ID and nutrient keys, and no retries in this task.
- Response projection retains only code, normalized name, optional canonical serving pair, finite non-negative JSON numeric nutrient values, and a validated public HTTPS image URL. String-valued `label` and `*_unit` nutriment metadata remains ignored; string values in numeric fields and null, empty, boolean, object, array, overflowed, malformed, negative, NaN, and infinite numeric values reject the candidate. Invalid candidates are dropped with only a count logged.
- `RawPayload` is always nil for OpenFoodFacts records. No response body, product value, query, caller ID, URL, transport cause, or raw provider diagnostic enters logs/errors or persistence.
- Provider errors expose only provider, closed code, HTTP status, retryability, and safe context cancellation/deadline sentinels. Transport and body-read failures use one mapper that checks both request-context state and direct/wrapped error identity while discarding arbitrary cause text.
- No SQL, filesystem write, command execution, authentication state, cookies, PII persistence, or cryptography was added.

## Exact changed paths and symbols

| Path | Task 244 surface |
| --- | --- |
| `backend/internal/externaldata/openfoodfacts.go` | Repaired finite body policy, unified transport/body context-sentinel mapping, and strict JSON nutriment token classification. |
| `backend/internal/externaldata/openfoodfacts_test.go` | Added adversarial context, body-limit, malformed numeric nutriment, textual metadata, and valid-sibling survival tests. |
| `backend/internal/externaldata/usda.go` | Unchanged task-243 behavior control during repair. |
| `docs/implementation/preparations/task-244.md` | Baseline, scope, source, security, symbol, acceptance, command, and hash evidence. |

Production declarations added in `openfoodfacts.go`:

- Constants: `DefaultOpenFoodFactsEndpoint`, `MaxOpenFoodFactsPageSize`, `defaultOpenFoodFactsDeadline`, `defaultOpenFoodFactsBodyLimit`, `maxOpenFoodFactsBodyLimit`, `openFoodFactsFields`.
- Types: `OpenFoodFactsConfig`, `OpenFoodFactsClient`, `openFoodFactsSearchPayload`, `openFoodFactsProduct`.
- Exported behavior: `NewOpenFoodFactsClient`, `(*OpenFoodFactsClient).Search`.
- Private behavior: `validateOpenFoodFactsQuery`, `decodeOpenFoodFactsSearch`, `projectOpenFoodFactsProduct`, `decodeJSONString`, `containsUnsafeProviderText`, `validCallerID`, `mapOpenFoodFactsStatus`, `(*OpenFoodFactsClient).transportFailure`, `(*OpenFoodFactsClient).failure`, `(*OpenFoodFactsClient).logDropped`, `openFoodFactsError`.

Production declarations modified by the original Task 244 preparation in `usda.go` (unchanged by this repair):

- `ExternalFoodRecord`: added `ImageURL string` from the DESIGN-012 record contract.
- `ProviderError`: added private provider identity for safe provider-specific diagnostics.
- `(*ProviderError).Error`: returns the configured safe provider label while preserving USDA as the compatibility default.

Production declarations modified by the review repair in `openfoodfacts.go`:

- Constant block: added the fixed `maxOpenFoodFactsBodyLimit` policy equal to the 2 MiB default.
- `NewOpenFoodFactsClient`: rejects configured limits above the finite policy maximum, including `MaxInt64`.
- `(*OpenFoodFactsClient).Search`: passes original transport and body-read errors plus HTTP status into the safe context-aware mapper.
- `(*OpenFoodFactsClient).transportFailure`: checks request-context state and direct/wrapped transport/body error sentinels, preserves only `context.DeadlineExceeded` or `context.Canceled`, and retains body-read HTTP status.
- `projectOpenFoodFactsProduct`: trims and classifies each raw nutriment token before decoding; preserves supported `label` and `*_unit` string metadata, rejects strings in numeric fields, non-number/non-string tokens, and numeric decode failures, and stores only finite non-negative numbers.

Test declarations added in `openfoodfacts_test.go`:

- Tests: `TestOpenFoodFactsSearchEncodesQueryIdentifiesCallerAndProjectsDeterministically`, `TestOpenFoodFactsSearchRejectsInvalidInputBeforeOutboundRequest`, `TestOpenFoodFactsSearchHonorsDeadlineAndCallerCancellation`, `TestOpenFoodFactsSearchPreservesContextErrorsWhileReadingBody`, `TestOpenFoodFactsSearchPreservesTransportContextSentinels`, `TestOpenFoodFactsSearchBoundsBodiesAndHandlesMalformedOrPartialPayloads`, `TestOpenFoodFactsSearchEnforcesFiniteAllocationBound`, `TestProjectOpenFoodFactsProductHandlesOptionalAndMalformedFields`, `TestProjectOpenFoodFactsProductRejectsMalformedNumericNutriments`, `TestOpenFoodFactsSearchMapsStatusesAndLogsOnlyBoundedMetadata`, `TestOpenFoodFactsSearchHandlesRequestTransportAndBodyReadFailures`, `TestNewOpenFoodFactsClientRejectsUnsafeConfiguration`.
- Fixtures/helpers: `testOpenFoodFactsCallerID`, `validOpenFoodFactsPayload`, `newTestOpenFoodFactsClient`, `validOpenFoodFactsQuery`, `contextBlockingBody`, `(*contextBlockingBody).Read`, `(*contextBlockingBody).Close`, `countingInfiniteBody`, `(*countingInfiniteBody).Read`, `(*countingInfiniteBody).Close`, `paddedOpenFoodFactsPayload`.

## Verification criteria

| Criterion | Direct evidence | Result |
| --- | --- | --- |
| URL/query encoding | Fake server asserts the exact path, NFC-normalized `crème & apple`, encoded URI, and exact `action`, `fields`, `json`, `page`, `page_size`, `search_simple`, and `search_terms` values. | PASS |
| Caller-identification headers | Fake server asserts exact custom `User-Agent` and JSON `Accept`; constructor rejects missing, control-bearing, and oversized caller IDs. | PASS |
| Page boundaries | Page 1..10,000 and size 1..100 are enforced; exact maxima reach the fake server; zero/above-maximum values make no request. | PASS |
| Deadlines/cancellation | Blocking fake server covers pre-header cancellation; context-bound response bodies cover deadline and caller cancellation after headers; direct and wrapped transport sentinels retain `errors.Is` identity and safe categorical status. | PASS |
| Bounded bodies | Exact 2 MiB valid payload succeeds; a counting infinite body proves exactly 2 MiB+1 bytes are read before rejection; 2 MiB+1 and `MaxInt64` configuration are rejected, making `max+1` overflow impossible. | PASS |
| Malformed/partial payload handling | Malformed/incomplete envelopes fail categorically; null, overflowed, boolean, object, array, empty, negative, NaN, infinite, and string-valued numeric nutriments reject candidates; supported `label`/`*_unit` metadata remains ignored; malformed candidates are dropped while valid siblings survive, with only the dropped count logged. | PASS |
| Provider status mapping | Redirect/unexpected statuses map unavailable; 400/401/404 map permanent rejection; 408/5xx map retryable unavailable; 429 maps retryable rate limit. | PASS |
| No outbound invalid input | Atomic fake-server counter remains zero across empty/long/control query, wrong provider, invalid pages, and invalid page sizes. Unsafe configuration fails construction. | PASS |
| Deterministic projection | Exact provider/ID/name/serving/nutrient/image projection is compared; textual nutriment metadata is ignored and provider order is preserved. | PASS |
| No raw provider payload persistence | Every projected OpenFoodFacts record has nil `RawPayload`; selected typed fields are the only provider data returned, and no repository method is called or added. | PASS |
| Safe diagnostics | Errors/logs contain only closed metadata; fake provider bodies, URL/query data, caller ID, and transport causes are absent. Redirect destination receives zero calls. | PASS |

## Commands and results

| Command | Result |
| --- | --- |
| Adversarial tests before repair | EXPECTED FAIL: body-read deadline/cancel and direct/wrapped transport sentinels mapped unavailable; 2 MiB+1 configuration was accepted. |
| Numeric adversarial test before second repair | EXPECTED FAIL: null was projected as zero; overflow, boolean, object, array, and string-valued numeric fields were silently ignored; supported unit metadata passed as expected. |
| `go test -count=1 -run '^Test(OpenFoodFacts\|NewOpenFoodFacts)' -v ./internal/externaldata` | PASS after repair; all OpenFoodFacts configuration and adversarial cases. |
| `go test -count=1 -v ./internal/externaldata` | PASS; all OpenFoodFacts and existing USDA fake-server/boundary cases. |
| `go test -count=1 -run '^TestUSDA' ./internal/externaldata` | PASS; task-243 behavior preserved. |
| `go test -count=1 -race -coverprofile=/tmp/task-244-nutriment-repair-cover.out ./internal/externaldata` | PASS; race clean; **100.0% statements** across `externaldata`, including every repaired OpenFoodFacts production function. |
| `go vet ./internal/externaldata` | PASS. |
| `go test -count=1 ./...` | PASS for every backend package. |
| `go test -count=1 -race ./...` | PASS for every backend package. |
| `go vet ./...` | PASS. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS: no vulnerabilities in called/imported code; 18 required-module advisories are not called. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks with ordered dependencies; Task 244 remains PREPARED. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `git diff --check` | PASS. |
| `python3 scripts/check.py` | PASS, exit 0: traceability/task list/Go Doc, OpenAPI/security, migrations and local-stack readiness, Phase 02/03 UAT, backend tests/race/coverage, frontend API drift/typecheck/build/unit/coverage, 72 + 30 focused Playwright checks, full Playwright 237 passed/3 expected skipped, and 459 frontend unit tests. Existing OAuth 302-only warning and documented repository-wide coverage deviations remain unchanged. |

## Final implementation hashes

| Path | SHA-256 |
| --- | --- |
| `backend/internal/externaldata/openfoodfacts.go` | `433733998bf73d63dd7dce66152ff24bb6ffb3cd4b80817d595b78949758b53d` |
| `backend/internal/externaldata/openfoodfacts_test.go` | `c13dbb6309a8041476a5d32b03db432ac8b2d8a6863c84897d07f9a977ea98b4` |
| `backend/internal/externaldata/usda.go` (unchanged repair control) | `21f29cb5c6d1c528339cce5aa4d78622bfa6bcc9051d9a0888dc2c51cee07beb` |
| `backend/internal/externaldata/usda_test.go` (unchanged repair control) | `b9045286353943098e3b3f4fa4576bb0954312ee9191a4ff622bad7e7a9dfd25` |
| `docs/implementation/reviews/task-244-review.md` (unchanged latest finding source) | `ea0db08816d9cba6c8f005192efa814746759192a704a6f7604fb3f10107fd83` |
| `docs/implementation/02_TASK_LIST.md` (unchanged repair control) | `6435b85a88d9c5df80176cea7b2edc1fd109e5ad8f54a3be78f3cca4d7f691d1` |

This evidence document's final hash is reported in the repair handoff because embedding its own digest would change that digest.

## Risks and handoff

- All three important findings recorded across the Task 244 reviews are repaired with adversarial regression coverage. No Task 244 acceptance, race, security, traceability, or aggregate-check blocker remains.
- OpenFoodFacts plain-text search currently uses the provider's legacy endpoint because current v2/v3 APIs do not expose full-text search. The endpoint is isolated behind immutable configuration and a narrow response projection for later replacement.
- Task 245 owns rate-limit state, retries, and cross-provider partial success. This client performs exactly one bounded request and exposes categorical retryability/status metadata.
- Task 246 owns nutrient aliases, canonical per-100 conversion, physical-state/density decisions, and warnings. This client preserves finite numeric provider nutrient keys without interpreting them.
- Independent review should recompute the hashes above and inspect every listed declaration. Task-list status must remain untouched by the preparation agent.
