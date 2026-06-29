# Task 142 Review — Phase 05 Generated Search API Client

## Decision

**Recommended status: PASSED**

## Task Reviewed

| Field | Value |
| --- | --- |
| Task ID | 142 |
| Component | Phase 05 Generated Search API Client |
| Static Aspect | DESIGN-001: SearchView |
| Status (in task list) | PREPARED |
| Retries | 0 |
| Depends On | 129, 139, 140, 141 |
| Testing Coverage Exceptions | None |
| Description | Implement the TanStack Query client over generated search/autocomplete envelopes with credentialed requests, 10-second timeout behavior, stable query keys, previous-page retention, local-cache reads/writes, and safe `AppError` mapping. |

## Dependency Check

| Dep ID | Required | Found | Notes |
| --- | --- | --- | --- |
| 129 | PASSED or PREPARED | PASSED | `02_TASK_LIST.md:136` shows Task 129 PASSED. `frontend/src/lib/api/generated.ts` exports `SearchRequest` (`:269`), `SearchResponse`/`SearchResponseEnvelope` (`:320`/`:332`), `AutocompleteResponse`/`AutocompleteEnvelope` (`:355`/`:362`), `AppError`/`ErrorCategory` (`:16`/`:4`), `Envelope` (`:26`) imported by `search-client.ts:5-13`. |
| 139 | PASSED or PREPARED | PASSED | `02_TASK_LIST.md:146` shows Task 139 PASSED. `frontend/package.json` provides `@tanstack/svelte-query` and `@tanstack/query-core` (imported at `search-client.ts:2-3`, `search-client.test.ts:2`). |
| 140 | PASSED or PREPARED | PASSED | `02_TASK_LIST.md:147` shows Task 140 PASSED. `frontend/src/lib/stores/search.ts` exports `searchRequestKey` (`:294`), `buildSearchRequest` (`:269`), `SearchState`, `createInitialSearchState` (`:51`), `searchStore` (`:74`), `setQuery`, `resetSearch` (`:261`) consumed by `search-client.ts:14` and the tests. |
| 141 | PASSED or PREPARED | PASSED | `02_TASK_LIST.md:148` shows Task 141 PASSED. `frontend/src/lib/cache/local-query-cache.ts` exports `LocalQueryCache` with `has`/`isStale`/`get`/`set` (`:129`/`:148`/`:88`/`:108`) consumed by `search-client.ts:15` and exercised by the cache hit/miss tests. |

## Verification Checklist

| ID | Criterion | Result | Evidence |
| --- | --- | --- | --- |
| C1 | Tests verify request URLs and bodies (POST `/api/v1/search` with `SearchRequest` body; GET `/api/v1/search/autocomplete` with query). | PASS | `search-client.test.ts:190` asserts `call.url === "/api/v1/search"`, `method === "POST"`, and `asJson(call.init)` deep-equals `{ query:"apple", mode:"catalog", page:1 }` with `Content-Type: application/json`. `search-client.test.ts:350` asserts `call.url === "/api/v1/search/autocomplete?query=app"`, `method === "GET"`, `body === undefined`. Implementation: `fetchSearch` at `search-client.ts:101` POSTs `JSON.stringify(request)` to `SEARCH_ENDPOINT`; `fetchAutocomplete` at `search-client.ts:122` builds `URLSearchParams` and GETs `AUTOCOMPLETE_ENDPOINT`. |
| C2 | Tests verify generated response decoding (envelope.data extracted). | PASS | `search-client.test.ts:206` decodes `makeSearchEnvelope(7,2)` and asserts `result.page === 2` and `result.items[0].id === "food-7"` — i.e. `envelope.data` is extracted, not the wrapper. `search-client.test.ts:364` decodes `makeAutocompleteEnvelope()` and asserts `result.items[0].label === "Apple"`. Implementation: `decodeSearchResponse` at `search-client.ts:295` returns `envelope.data` (`:326`); `decodeAutocompleteResponse` at `search-client.ts:329` returns `envelope.data` (`:360`). Both reject bodies where `envelope.data` is missing/null (`:316`, `:350`). |
| C3 | Tests verify credentials (`credentials: "include"`). | PASS | `search-client.test.ts:200` asserts `call.init.credentials === "include"` for POST search. `search-client.test.ts:359` asserts the same for GET autocomplete. `search-client.test.ts:616` re-verifies credentials on the autocomplete query-options path. Implementation: `fetchSearch` (`:104`) and `fetchAutocomplete` (`:131`) both set `credentials: "include"`. |
| C4 | Tests verify timeout cancellation (10-second AbortController). | PASS | `search-client.test.ts:469` enqueues a `pendingUntilAbort` provider, builds options with `timeoutMs=50`, and asserts the query rejects with `SearchClientError` whose `appError.category === "timeout"`, `retryable === true`, `code === "search_timeout"`. `search-client.test.ts:620` covers autocomplete timeout. `search-client.test.ts:491` confirms a parent abort is rethrown as `AbortError` without being misclassified as a timeout. Implementation: `SEARCH_TIMEOUT_MS = 10_000` (`search-client.ts:27`); `createTimeoutSignal` (`:205`) chains a child `AbortController` that aborts with `DOMException("Search timeout","TimeoutError")` after `timeoutMs`; `mapAbortError` (`:277`) converts the `TimeoutError` reason into a retryable `SearchClientError` with status 408. |
| C5 | Tests verify cache hit/miss behavior (local cache read bypasses fetch; miss triggers fetch and writes). | PASS | `search-client.test.ts:421` pre-seeds `localCache.set(requestKey, request, cached)`, invokes `queryFn`, asserts `result` equals the cached response and `fetchMock.calls.length === 0`. `search-client.test.ts:437` enqueues a 200 envelope, invokes `queryFn` on an empty cache, asserts the decoded response is returned, `fetchMock.calls.length === 1`, and `localCache.get(requestKey).response` equals the decoded response. `search-client.test.ts:452` advances `now` past `LOCAL_CACHE_STALE_MS` and asserts a stale entry triggers a fetch. `search-client.test.ts:557` repeats the hit-bypasses-fetch check through a real `QueryClient.fetchQuery`. Implementation: `runSearchQueryFn` (`:227`) reads `localCache.has` + `localCache.isStale` before fetch (`:235`) and writes `localCache.set` after success (`:246`). |
| C6 | Tests verify previous results during page loads (keepPreviousData/placeholderData). | PASS | `search-client.test.ts:415` asserts `options.placeholderData === keepPreviousData`. `search-client.test.ts:570` runs a full `QueryObserver` integration: loads page 1, switches options to page 2 with a deferred provider, and asserts `during.isFetching === true`, `during.isPlaceholderData === true`, and `during.data` still equals the page-1 response while page 2 loads; after resolving, `data` equals the page-2 response. Implementation: `buildSearchQueryOptions` sets `placeholderData: keepPreviousData` (`search-client.ts:159`); `buildAutocompleteQueryOptions` does the same (`:193`). |
| C7 | Tests verify 400/422/429/503 mapping to `AppError` with retryability. | PASS | `search-client.test.ts:214` maps 400 to `category:"validation"`, `retryable:false`, `code:"query_too_short"`. `:246` maps 422 to `category:"validation"`, `code:"filter_conflict"`. `:266` maps 429 to `category:"server"`, `retryable:true`. `:286` maps 503 to `category:"dependency"`, `retryable:true`. `:330` covers default derivation when the envelope error is missing: 400→`invalid_request`/not-retryable, 422→`search_rejected`, 429→`rate_limited`/retryable, 503→`dependency_unavailable`/retryable. `:505` confirms the queryFn path surfaces a 429 as `SearchClientError`. Implementation: `mapAppError` (`:69`) + `categoryForStatus` (`:380`) + `defaultRetryableFor` (`:394`) + `defaultCodeForStatus` (`:398`). |
| C8 | Tests verify request IDs preserved from server envelope. | PASS | `search-client.test.ts:240` asserts `appError.requestId === "req-400"`. `:261` asserts `"req-422"`. `:281` asserts `"req-429"`. `:301` asserts `"req-503"`. `:386` asserts `"req-auto-429"` for autocomplete. `:522` asserts `"req-qfn-429"` through the queryFn path. Implementation: `mapAppError` copies `envelopeError.requestId` (`:88-90`); `decodeSearchResponse`/`decodeAutocompleteResponse` additionally call `attachRequestId(appError, envelope.requestId)` (`:311`/`:345`) so the envelope-level `requestId` is preserved even when the server error object omits it. |
| C9 | Tests verify no duplicate request for equivalent query keys. | PASS | `search-client.test.ts:406` builds options from two independent but equivalent `catalogState("apple",1)` states and asserts `optionsA.queryKey === optionsB.queryKey`. `:542` calls `queryClient.fetchQuery` twice with equivalent options and asserts `fetchMock.calls.length === 1` after both. `:633` confirms equivalent `searchStore` states share a key via `searchRequestKey`. `:643` confirms distinct pages produce distinct keys (negative case). Implementation: `buildSearchQueryOptions` derives the key from `searchRequestKey(state)` (`:152-154`), so key stability is delegated to the already-PASSED Task 140 normalizer. |

Additional coverage beyond required criteria: stack-trace/URL leak prevention (`:306`), parent-abort pass-through without timeout misclassification (`:491`), `createSearchQueryOptions` reactive derived store (`:527`), autocomplete envelope decoding + credentials on the query-options path (`:605`), and `QueryClient` local-cache hit bypass through the real TanStack cache (`:557`).

## Commands Run

| Command | Working dir | Exit | Result |
| --- | --- | --- | --- |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/search-client.test.ts` | `frontend/` | 0 | 28 pass, 0 fail, 94 expect() calls, 1 file, 225 ms. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | `frontend/` | 0 | 93 pass, 0 fail, 275 expect() calls across 7 files (search-client + Task 140/141/143 stores/cache/preferences + SettingsPanel). |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | `frontend/` | 0 | Vite build OK: 116 modules, `dist/index.html` 0.51 kB, CSS 11.31 kB, JS 41.24 kB; built in 734 ms. |
| `git -C /home/wiktor/Work/glm status --short` | repo root | 0 | Task-142 deliverables: `frontend/src/lib/api/search-client.ts`, `frontend/src/lib/api/search-client.test.ts` (both untracked, new). Other untracked/modified files belong to Task 143 (SettingsPanel, preferences) or earlier tasks (Playwright config, search store, local-query-cache, evidence/). `frontend/tsconfig.tsbuildinfo` untracked (TS incremental cache, not staged, not a Task 142 deliverable). No `dist/`, `node_modules/`, `.bun-tmp/`, `.bun-install/` staged (gitignored). |
| `python3 /home/wiktor/Work/glm/scripts/validate-task-list.py` | repo root | 0 | "Task-list validation passed: 154 sequential tasks with ordered dependencies." |
| `python3 /home/wiktor/Work/glm/scripts/validate-traceability.py` | repo root | 0 | "Traceability validation passed." |

## Files Inspected

| Path | Reason |
| --- | --- |
| `frontend/src/lib/api/search-client.ts` (470 lines, new) | Implementation: `SEARCH_TIMEOUT_MS = 10_000` (`:27`), `LOCAL_CACHE_STALE_MS` (`:30`); `SearchClientError` (`:51`); `mapAppError` (`:69`) with `categoryForStatus`/`defaultRetryableFor`/`defaultCodeForStatus`/`looksSafe` helpers (`:380`/`:394`/`:398`/`:453`); `fetchSearch` (`:101`) and `fetchAutocomplete` (`:122`) credentialed fetch + envelope decoders (`:295`/`:329`); `buildSearchQueryOptions`/`createSearchQueryOptions`/`buildAutocompleteQueryOptions` (`:147`/`:172`/`:186`); `createTimeoutSignal` abort chaining (`:205`); `runSearchQueryFn`/`runAutocompleteQueryFn` (`:227`/`:255`) with local-cache read/write and timeout/abort mapping. Imports types only from `./generated` (`:5-13`) — no handwritten duplicates. |
| `frontend/src/lib/api/search-client.test.ts` (648 lines, 28 tests, new) | Bun tests covering C1-C9 plus stack-trace leak prevention, parent-abort pass-through, reactive options store, and `QueryClient`/`QueryObserver` integration. Uses `FetchMock` with queued providers, `pendingUntilAbort` for timeout tests, injectable `now` for staleness, and real TanStack `QueryClient`/`QueryObserver` for duplicate-request and placeholderData verification. |
| `frontend/src/lib/api/generated.ts` | Confirms `SearchRequest` (`:269`), `SearchResponse`/`SearchResponseEnvelope` (`:320`/`:332`), `AutocompleteResponse`/`AutocompleteEnvelope` (`:355`/`:362`), `AppError`/`ErrorCategory` (`:16`/`:4`), `Envelope` (`:26`) are generated types re-used by the client; `ErrorCategory` union (`:4-12`) matches the `isCategory` guard in `search-client.ts:434`. No duplicate declarations introduced. |
| `frontend/src/lib/stores/search.ts` | Dependency Task 140: exports `searchRequestKey` (`:294`), `buildSearchRequest` (`:269`), `createInitialSearchState` (`:51`), `searchStore` (`:74`), `setQuery`, `resetSearch` (`:261`) consumed by `search-client.ts:14` and tests. |
| `frontend/src/lib/cache/local-query-cache.ts` | Dependency Task 141: `LocalQueryCache` `has`/`isStale`/`get`/`set` API (`:129`/`:148`/`:88`/`:108`) consumed by `runSearchQueryFn`. |
| `docs/implementation/02_TASK_LIST.md` | Task 142 row (`:149`) is `PREPARED`; deps 129 (`:136`), 139 (`:146`), 140 (`:147`), 141 (`:148`) all `PASSED`. |
| `.gitignore` | `dist/`, `.bun-tmp/`, `.bun-install/` ignored. `.gitignore` diff adds Playwright artifacts only — no Task 142 dependency. |

## Coverage / Exception Review

- **Testing Coverage Exceptions:** None declared by the task.
- **Traceability comments:** 15 comments in `search-client.ts` and 21 in `search-client.test.ts` cite `Implements DESIGN-001 SearchView ...` or `Implements DESIGN-017 ErrorMessageMapper ...` at module, function, TSDoc `@remarks`, and test level — meets AGENTS.md requirement.
- **TSDoc:** Every exported constant, type, class, and function has TSDoc with `@remarks` tying it to DESIGN-001/DESIGN-017 — meets AGENTS.md requirement.
- **Generated-type discipline:** `SearchRequest`, `SearchResponse`, `SearchResponseEnvelope`, `AutocompleteResponse`, `AutocompleteEnvelope`, `AppError`, `Envelope` are imported as types from `./generated` (`:5-13`); no handwritten `type`/`interface` redeclarations of those names. The only locally-defined types are `SearchQueryKey`/`AutocompleteQueryKey` (TanStack key shapes), which are client concerns, not API contracts.
- **Scope discipline:** Implementation is a pure API/client layer. It does not render UI (Tasks 143-147, 150), does not wire offline banner (Task 148), theme (149), or full integration (151). The `createSearchQueryOptions` store bridge is the reactive seam Task 144+ will consume. No `SearchShell.svelte`/`SettingsPanel`/sidebar/results code is touched by the Task 142 files.
- **Simplicity:** Single `fetch` wrapper per endpoint, one `createTimeoutSignal` helper reused by both query functions, `keepPreviousData` for placeholder data (TanStack built-in), envelope decoding centralized in two small functions. No redundant state or speculative abstraction.
- **Security:** `looksSafe` (`:453`) rejects server messages containing URLs, file:line fragments, newlines, or stack/panic/goroutine/traceback keywords — prevents infrastructure detail leakage per DESIGN-017. Credentials use `include` for cookie-based auth. No secrets, tokens, or PII handled.
- **No generated artifacts staged:** `frontend/dist/`, `node_modules/`, `.bun-tmp/`, `.bun-install/` are gitignored. The only new Task 142 files are the two search-client source files.

## Failure Details

None. All nine verification criteria (C1-C9) are satisfied with passing test evidence, implementation inspection, build success, and aggregate validators passing.
