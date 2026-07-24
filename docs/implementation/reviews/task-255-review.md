# Review Evidence: Task 255 — External Search and Import UI

```yaml
task_id: 255
component: "External Search and Import UI"
static_aspect: "External Search and Import UI"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-21T21:33:27Z"
review_agent: "independent-final-re-review-task-255"
evidence_file: "docs/implementation/reviews/task-255-review.md"
baseline_ref: "81ca40ce00cb667ea29243ed2d34068e11229a69 plus preparation manifest and current dirty-worktree hashes"
baseline_confidence: "MEDIUM"
code_review_skill_invoked: true
relevant_language_guide: "TypeScript, Svelte 5, security, async/concurrency, and error-handling guidance"
repair_context_required: false
```

## 1. Task Source

**Description:** Phase 08 administrator external-source search, provider selection, pagination, candidate warning display, editable curation draft, conflict confirmation, import-result workflow, retry-stable idempotency keys, local-search visibility, keyboard operation, and safe provider/error messaging.

**Depends On:** 253 (`PASSED`), 254 (`PASSED`)

**Testing Coverage Exceptions:** None in the task row. Repository guidance targets 100% line coverage by phase; Svelte source-line instrumentation is unavailable in the current Bun profile and is recorded as optional evidence debt with browser evidence.

**Verification Criteria:** Component and Playwright tests cover USDA, OpenFoodFacts, and combined searches; loading, empty, partial-success, rate-limit, timeout, and unavailable states; candidate edits; normalization and liquid-density warnings; classification selection; conflict confirmation; ambiguous import-response replay creating one item; successful local-search visibility; keyboard-only operation; and safe messages with no raw provider diagnostics.

## 2. Pre-Review Gates

- [x] Input status is `PREPARED`.
- [x] Dependencies 253 and 254 are `PASSED`.
- [x] The preparation report claims completion of the timeout-signal repair and F-255-001 through F-255-008 evidence.
- [x] A task-specific baseline is available. The Phase 08 frontend files are untracked in the shared worktree, so scope is reconstructed from fixed `HEAD`, the preparation manifest, prior-review fingerprints, current source, and explicit task boundaries.
- [x] `code-review-skill` was invoked exactly once and the TypeScript and Svelte guides plus applicable security and async guidance were read.
- [x] This review is independent of the implementation/repair agent.
- [x] Review uses current source, fresh commands, fresh adversarial probes, and fresh hashes rather than preparation claims alone.
- [x] No production code or task-list status was changed; only this review evidence file is written.

```yaml
pre_review_gates_passed: true
blocking_issue: "None. Six optional hardening or instrumentation observations remain visible; no blocking or important finding remains."
```

## 3. Review Baseline and Change Surface

Baseline/reference method: `git rev-parse HEAD` returned `81ca40ce00cb667ea29243ed2d34068e11229a69`. The worktree contains concurrent Phase 08 changes, including untracked frontend files. The repaired Task 255 surface was reconstructed from `docs/implementation/preparations/task-255.md`, the rejected predecessor review, current file inspection, focused source discovery, and post-review SHA-256 hashes. Shared Task 254 shell files and generated/backend contract files were inspected as boundaries and hashed as context, not treated as repaired Task 255 production edits.

Commands used to reconstruct the diff and surface:

```bash
git status --short
git log -1 --oneline
git diff --stat
git rev-parse HEAD
sed -n '250,266p' docs/implementation/02_TASK_LIST.md
sed -n '1,220p' docs/implementation/preparations/task-255.md
rg -n 'function|test\\(' <Task-255 source and test files>
nl -ba <Task-255 implementation and test files>
sha256sum <all files listed in section 9>
```

Pre-existing dirty-worktree changes and exclusions: backend Phase 08 work, OpenAPI and generated contracts, Task 254/256/257 frontend files, prior preparation/review evidence, caches, and build output were preserved. Task 255 review scope includes the external client, external workflow, focused client/component/browser tests, the AdministrationPanel composition boundary, and the SearchShell local-search handoff. Backend custom-item/data-importer/import-controller files, `api/openapi.yaml`, `docs/design/DESIGN-009.md`, and generated API types are contract context. No task-owned scope was unverifiable at the symbol level, but the shared untracked worktree limits baseline confidence to `MEDIUM`.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| `frontend/src/lib/api/external-admin-client.ts` | preparation manifest plus current source | HIGH | public operations, transport, bounded decoder, CSRF preflight, safe mappers, runtime guards |
| `frontend/src/lib/api/external-admin-client.test.ts` | preparation manifest plus current tests | HIGH | transport identity, timeout, status, bounds, CSRF, request-ID, payload, and security tests |
| `frontend/src/lib/components/ExternalImportWorkflow.svelte` | preparation manifest plus current source | HIGH | workflow state machine, lifecycle, draft/provenance handlers, conflict branches, templates |
| `frontend/src/lib/components/ExternalImportWorkflow.test.ts` | current source-contract tests | HIGH | workflow contract assertions |
| `frontend/tests/external-import-workflow.spec.ts` | current browser tests | HIGH | fixtures, route stubs, desktop/mobile workflow tests |
| `frontend/src/lib/components/AdministrationPanel.svelte` | shared Task 254 composition boundary | MEDIUM | external workflow composition in allowed branch |
| `frontend/src/lib/components/AdministrationPanel.test.ts` | shared Task 254 composition test | MEDIUM | composition and authorization-boundary assertions |
| `frontend/src/lib/components/SearchShell.svelte` | shared shell local-search handoff boundary | MEDIUM | imported-item handoff and callback binding |
| `frontend/src/lib/components/SearchShell.test.ts` | shared shell handoff assertions | MEDIUM | local-search handoff contract assertions |

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | USDA search | focused provider matrix and browser route | PASS | `searchExternalFoods` builds the generated provider query; focused tests and 14 desktop/mobile Playwright tests pass. |
| 2 | OpenFoodFacts search | focused provider matrix and browser route | PASS | OpenFoodFacts URL/provider projection is exercised in focused and browser tests. |
| 3 | Combined search | focused provider matrix and browser route | PASS | Provider `all` is exercised, including merged warning rendering and pagination. |
| 4 | Loading state | pending-response browser assertion | PASS | The pending route shows `[data-external-loading]`. |
| 5 | Empty state | released-empty browser assertion | PASS | An empty candidate collection renders `[data-external-empty]`. |
| 6 | Partial-success state | warning plus candidate retention | PASS | Closed provider warnings render fixed labels while candidates remain usable. |
| 7 | Rate-limit state | 429 and Retry-After tests | PASS | Status mapping and bounded retry copy render without server text. |
| 8 | Timeout state | 504 and transport-timeout tests | PASS | Provider 504 and genuine timeout signals render safe timeout copy. |
| 9 | Unavailable state | 503 tests | PASS | Provider-unavailable copy is fixed and raw diagnostics are suppressed. |
| 10 | Candidate edits | submitted draft inspection | PASS | Browser evidence checks edited name, macros, liquid density, provenance, and classifications. |
| 11 | Normalization and liquid-density warnings | warning labels, draft invariant, backend contract | PASS | Missing density, unit-conversion, and suspicious-macro warnings are visible; positive manual density resolves only the density warning and the backend accepts the resulting contract. |
| 12 | Classification selection | UUID checkbox and request-body assertions | PASS | Backend-owned classification UUIDs, not labels, reach the import body. |
| 13 | Conflict confirmation | subtype-aware 409 matrix | PASS | Only `name_conflict_confirmation_required` renders `Confirm merge`; provider, idempotency, and unknown conflicts are blocked. |
| 14 | Ambiguous replay | connection reset, same key, one result | PASS | A transport ambiguity retries with the same memory-only key and displays one validated local item identity. |
| 15 | Successful local-search visibility | SearchShell handoff and browser assertion | PASS | The validated imported name enters the ordinary Catalog search and is visible. |
| 16 | Keyboard-only operation | native controls, Enter activation, axe | PASS | Native forms/buttons and the local handoff are keyboard-operable; focused browser axe checks have no serious or critical violations. |
| 17 | Safe messages without raw diagnostics | hostile unit/browser payloads | PASS | Raw provider payloads, response messages, status codes, socket text, and malformed nested values do not enter visible UI copy. |
| 18 | F-255-001 liquid provenance | source inspection and browser/backend request body | PASS | `validDraft`, `hasValidLiquidDensity`, `updateDensity`, and `updatePhysicalState` require positive density with manual/estimated or evidenced imported provenance and clear stale liquid fields when solid. |
| 19 | F-255-002 conflict subtype behavior | unit and browser conflict matrix | PASS | Allowlisted name/provider/idempotency codes retain distinct safe copy; merge is name-only and fresh-key rotation is explicit. |
| 20 | F-255-003 nested payload guards | malformed candidate, warning, classification, and result payloads | PASS | Exact nested guards reject malformed ordinary JSON before Svelte state; malformed browser payloads recover safely. One non-native mocked-object coercion edge is optional and cannot be produced by JSON transport. |
| 21 | F-255-004 successful-status allowlists | wrong-2xx matrix and body-cancel probe | PASS | Search and classifications require 200, import requires 201, CSRF requires 200; wrong successful statuses fail closed and are canceled. |
| 22 | F-255-005 body and correlation bounds | declared and streamed overflow plus request-ID matrix | PASS | Success bodies are capped at 256 KiB, error bodies at 16 KiB, overflow cancels/releases readers, and request IDs are limited to 1–120 safe printable characters. |
| 23 | F-255-006 cancellation identity and cleanup | fetch/body AbortError identity and reader cleanup | PASS | Caller and success/error body-reader `AbortError` values preserve identity; cleanup occurs on read failure and overflow; genuine timeout is mapped separately. |
| 24 | F-255-007 strict CSRF preflight | no-token import, wrong status, oversized body, hostile ID | PASS | The generated credentialed CSRF request uses the strict Task 255 decoder, requires HTTP 200, bounds both body classes, validates the envelope and token, and prevents import after preflight rejection. |
| 25 | F-255-008 generic transport AbortError timeout | real `AbortSignal.timeout`, substituted generic transport error, identity matrix, browser copy | PASS | A timed-out signal plus a fresh generic `DOMException(..., "AbortError")` maps to `external_request_timeout`; caller AbortError and custom cancellation objects remain identical; Chromium verifies fixed timeout copy. |

## 5. Changed-Symbol Inventory

Test-only fixtures and test callbacks are grouped by owning test file in rows 52–54. The grouping is limited to test executables and is justified because their behavior is audited together by their named test file; no production executable symbol is omitted. Generated artifacts are not grouped.

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | `ExternalAdminClientError.constructor` | class constructor | `external-admin-client.ts:36-47` | modified | all external client failures | status and transport tests |
| 2 | `searchExternalFoods` | async API | `external-admin-client.ts:51-67` | modified | `runSearch` | provider, malformed, bounds, cancellation |
| 3 | `loadAdminClassifications` | async API | `external-admin-client.ts:70-83` | modified | `loadClassifications` | classification and browser shell tests |
| 4 | `importCuratedItem` | async API | `external-admin-client.ts:86-107` | modified | `submitImport` | import, conflict, replay, CSRF |
| 5 | `createImportIdempotencyKey` | key factory | `external-admin-client.ts:109-112` | added | candidate selection and fresh retry | UUID and conflict tests |
| 6 | `safeFetch` | transport wrapper | `external-admin-client.ts:114-141` | repaired | all public APIs | timeout and cancellation identity |
| 7 | `decodeResponse` | bounded operation decoder | `external-admin-client.ts:143-161` | modified | all public APIs including CSRF | status, body, envelope |
| 8 | `fetchImportCsrfToken` | CSRF preflight | `external-admin-client.ts:163-168` | added | `importCuratedItem` | no-token and CSRF adversarial tests |
| 9 | `safeResponseError` | safe error mapper | `external-admin-client.ts:170-183` | modified | `decodeResponse` | status, conflict, hostile body |
| 10 | `malformedResponse` | safe error factory | `external-admin-client.ts:185-195` | modified | decoder and guards | malformed/status tests |
| 11 | `safeMessageForStatus` | safe copy mapper | `external-admin-client.ts:197-208` | modified | error mapper | safe messages and conflict matrix |
| 12 | `safeCodeForStatus` | code allowlist | `external-admin-client.ts:210-217` | modified | error mapper and workflow | conflict matrix |
| 13 | `categoryForStatus` | status mapper | `external-admin-client.ts:219-226` | used | error mapper | status tests |
| 14 | `parseRetryAfter` | bounded retry parser | `external-admin-client.ts:228-232` | used | error mapper | rate-limit tests |
| 15 | `readBoundedText` | stream reader | `external-admin-client.ts:234-258` | added | `decodeResponse` | overflow, reader AbortError, CSRF |
| 16 | `safeRequestId` | correlation-token guard | `external-admin-client.ts:260-262` | added | success/error decoders | hostile and max ID tests |
| 17 | `isCsrfTokenData` | CSRF data guard | `external-admin-client.ts:264-266` | added | CSRF preflight | exact token tests and hostile-token probe |
| 18 | `isErrorEnvelope` | error envelope guard | `external-admin-client.ts:268-270` | used | error mapper | hostile errors |
| 19 | `isExternalSearchData` | search data guard | `external-admin-client.ts:272-277` | modified | search API | nested and page bounds |
| 20 | `isImportResult` | import result guard | `external-admin-client.ts:279-284` | modified | import API | malformed result |
| 21 | `isClassificationData` | classification guard | `external-admin-client.ts:286-293` | modified | classification API | malformed classification |
| 22 | `isExternalCandidate` | candidate guard | `external-admin-client.ts:295-302` | added | search data guard | malformed candidate and URI |
| 23 | `isExternalDataWarning` | provider warning guard | `external-admin-client.ts:304-308` | added | search data guard | warning allowlist |
| 24 | `isMacroProfile` | macro guard | `external-admin-client.ts:310-313` | added | candidate guard | NaN and negative values |
| 25 | `isNumericMap` | micronutrient guard | `external-admin-client.ts:315-319` | added | candidate guard | malformed maps |
| 26 | `isCandidateWarning` | candidate warning enum | `external-admin-client.ts:321-323` | added | candidate guard | hostile warning |
| 27 | `isProviderWarningCode` | provider warning enum | `external-admin-client.ts:325-327` | added | provider warning guard | hostile warning |
| 28 | `exact` | exact-object guard | `external-admin-client.ts:329-333` | added | all runtime guards | malformed payload suite |
| 29 | `boundedString` | string bound helper | `external-admin-client.ts:335-337` | added | all runtime guards | bounds suite |
| 30 | `nonnegativeFiniteNumber` | numeric guard | `external-admin-client.ts:339-341` | added | nutrition/map guards | NaN and negative suite |
| 31 | `positiveInteger` | page guard | `external-admin-client.ts:343-345` | added | search data guard | page validation |
| 32 | `uuid` | identifier guard | `external-admin-client.ts:347-349` | added | import/classification guards | malformed UUID suite |
| 33 | `isUri` | URI guard | `external-admin-client.ts:351-358` | used | candidate guard | malformed URI |
| 34 | `isRecord` | object guard | `external-admin-client.ts:360-362` | added | all guards | malformed payload suite |
| 35 | `isAbort` | cancellation guard | `external-admin-client.ts:364-366` | added | transport and decoder | AbortError identity |
| 36 | `isTimeout` | timeout guard | `external-admin-client.ts:368-370` | repaired | transport | direct and signal timeout |
| 37 | `isDOMExceptionNamed` | branded DOMException guard | `external-admin-client.ts:372-379` | repaired | abort and timeout guards | cross-brand identity matrix |
| 38 | `onMount` callback | lifecycle callback | `ExternalImportWorkflow.svelte:66-69` | modified | Svelte mount/unmount | stale search browser test |
| 39 | `loadClassifications` | async workflow operation | `ExternalImportWorkflow.svelte:71-81` | modified | mount and retry | classification shell fixtures |
| 40 | `runSearch` | async workflow operation | `ExternalImportWorkflow.svelte:83-109` | modified | form, paging, retry | stale and safe-state browser tests |
| 41 | `selectCandidate` | draft initializer | `ExternalImportWorkflow.svelte:111-130` | modified | Curate buttons | main browser flow |
| 42 | `toggleClassification` | draft updater | `ExternalImportWorkflow.svelte:132-137` | used | classification checkboxes | submitted UUID assertions |
| 43 | `updatePhysicalState` | draft updater | `ExternalImportWorkflow.svelte:139-146` | added | physical-state select | source and density flow |
| 44 | `updateDensity` | draft updater | `ExternalImportWorkflow.svelte:148-152` | added | density input | liquid provenance flow |
| 45 | `updateDensitySourceKind` | provenance updater | `ExternalImportWorkflow.svelte:154-157` | added | provenance select | source and browser flow |
| 46 | `submitImport` | async workflow operation | `ExternalImportWorkflow.svelte:159-187` | modified | form and retry handlers | conflict and replay browser tests |
| 47 | `validDraft` | draft validator | `ExternalImportWorkflow.svelte:189-195` | modified | `submitImport` | backend-faithful liquid flow |
| 48 | `hasValidLiquidDensity` | liquid invariant | `ExternalImportWorkflow.svelte:197-201` | added | derived warning and template | density warning browser test |
| 49 | `startFreshImport` | key rotation handler | `ExternalImportWorkflow.svelte:203-208` | added | idempotency conflict action | key-difference browser assertion |
| 50 | `withRetryAfter` | safe copy formatter | `ExternalImportWorkflow.svelte:210-212` | used | search error rendering | rate-limit browser test |
| 51 | `visibleSelectedWarnings` | derived state | `ExternalImportWorkflow.svelte:45` | added | warning template | density warning browser test |
| 52 | external client test fixtures and cases | test executable group | `external-admin-client.test.ts:16-358` | added | client boundary | 3 helpers and 18 tests; grouping justified above |
| 53 | workflow source-contract test cases | test executable group | `ExternalImportWorkflow.test.ts:7-56` | added | workflow source | 4 tests; grouped by file |
| 54 | external workflow browser fixtures/routes/cases | browser test executable group | `external-import-workflow.spec.ts:15-336` | added | rendered admin workflow | 7 helpers and 7 tests across desktop/mobile; grouped by file |
| 55 | AdministrationPanel external-workflow branch | Svelte composition boundary | `AdministrationPanel.svelte:52-53` | context | allowed admin branch | AdministrationPanel test and browser shell |
| 56 | AdministrationPanel composition tests | test executable group | `AdministrationPanel.test.ts:7-38` | context | composition boundary | 4 source-contract tests |
| 57 | `viewImportedItemInLocalSearch` | shell handoff | `SearchShell.svelte:324-330` | context | workflow success callback | SearchShell and browser local visibility |
| 58 | SearchShell handoff assertions and callback binding | test executable group | `SearchShell.test.ts:45-65` | context | shell boundary | 2 task-specific source-contract tests |

```yaml
inventory_source_count: 58
audited_symbol_count: 58
inventory_complete: true
generated_groupings:
  - "None. Test-only callbacks are grouped by owning test file with explicit justification; no generated executable artifact is grouped."
```

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| `ExternalAdminClientError.constructor` | Stores only already-mapped safe error, status, and retry metadata. | Fixed message and optional retry value; no fallthrough. | Readonly fields; no resources. | Provider diagnostics cannot enter through this constructor from mappers. | Constant. | Minimal public error type. | Status and transport tests. | PASS |
| `searchExternalFoods` | Builds trimmed provider/page GET with cookies and signal and returns validated search data. | Wrong status, malformed payload, overflow, and cancellation fail closed. | Signal reaches fetch; `runSearch` owns stale-response sequence. | Only validated candidates/warnings reach state. | One bounded request. | Thin generated-contract boundary. | Provider, malformed, bounds, timeout, identity tests. | PASS |
| `loadAdminClassifications` | Loads one closed classification kind and returns exact guarded projections. | Bad status, IDs, names, kind, overflow, and cancellation fail closed. | Caller signal supported; overlap/unmount suppression is optional lifecycle hardening. | IDs and labels guarded before rendering. | Max 1000 entries. | Narrow API. | Classification and browser shell tests. | PASS |
| `importCuratedItem` | Sends draft, stable key, cookies, CSRF, JSON, and signal; returns exact import identity. | Strict preflight without token; all invalid statuses/data fail safely. | Shared caller signal; memory-only key. | Explicit CSRF/idempotency headers; exact response guard. | At most two bounded requests. | Correct mutation boundary. | Import, conflict, replay, malformed, status, CSRF tests. | PASS |
| `createImportIdempotencyKey` | Generates one canonical opaque browser UUID per attempt. | Native generation has no weak fallback. | Memory-only; explicit fresh action rotates. | Not rendered or logged. | Constant. | Appropriate public factory. | UUID and browser key tests. | PASS |
| `safeFetch` | Classifies genuine timeout before preserving cancellation; maps other transport errors to safe ambiguity. | Timed-out signal plus generic AbortError maps timeout; caller AbortError, custom objects, and plain object named TimeoutError preserve identity; ordinary errors map safely. | Fetch uses supplied signal; no detached timer. | Raw transport reason/message never reaches app error. | One fetch and constant classification. | Fix ordering is narrow and correct. | Generic transport test and fresh timeout/caller/custom matrix. | PASS |
| `decodeResponse` | Enforces operation-specific success status, bounded JSON, exact success envelope, safe IDs, and error mapping. | Wrong 2xx cancels; malformed/overflow fails; body AbortError rethrows; error responses map safely. | Reader cleanup delegated; status rejection cancels body. | Untrusted error fields are allowlisted/replaced. | 256 KiB success and 16 KiB error. | Shared decoder prevents duplication. | Status, bounds, malformed, cancellation tests. | PASS |
| `fetchImportCsrfToken` | Uses generated credentialed GET and admits exact HTTP 200 token envelope. | Wrong status, overflow, extra fields, unsafe ID, bad token, cancellation fail before import. | Caller signal covers preflight; reader cleanup verified. | Cookie and token boundary explicit. | One bounded request. | Avoids permissive auth decoder. | No-token, wrong-status, overflow, hostile-ID tests. | PASS |
| `safeResponseError` | Creates fixed category/code/message/retryability and retains only safe request ID. | Missing source and unknown conflict use fixed fallbacks. | Pure, no resources. | Raw server fields never rendered. | Constant after bounded parse. | Centralized safe mapping. | Status/conflict/hostile-body tests. | PASS |
| `malformedResponse` | Produces bounded retryable operation-specific server error. | Covers all malformed/status/guard failures. | No resources. | Operation is internal. | Constant. | Simple factory. | Malformed/status tests. | PASS |
| `safeMessageForStatus` | Maps statuses and allowlisted conflict subtypes to fixed copy. | Handles validation, auth, rate, timeout, dependency, and fallback. | Pure. | No response text interpolation. | Constant. | Explicit policy branches. | Safe-message and browser tests. | PASS |
| `safeCodeForStatus` | Retains only name/provider/idempotency conflict codes. | Unknown 409 and other statuses use closed values. | Pure. | Server code cannot create arbitrary workflow branch. | Constant. | Minimal allowlist. | Four-code conflict matrix. | PASS |
| `categoryForStatus` | Maps status to closed error category. | Every status has deterministic fallback. | Pure. | Ignores untrusted source category. | Constant. | Idiomatic. | Focused status tests. | PASS |
| `parseRetryAfter` | Accepts positive decimal seconds and clamps at 3600. | Null, invalid, zero, unsafe, oversized return undefined. | Pure. | Header treated as untrusted. | Constant. | Bounded parser. | Rate-limit evidence. | PASS |
| `readBoundedText` | Reads fatal UTF-8 bytes within cap and flushes decoder. | Declared/stream overflow, invalid UTF-8, reader errors, and absent body handled. | Cancels on failure and releases lock in finally. | Body not logged or exposed. | Byte cap and bounded string. | Correct stream lifecycle. | Fresh overflow and cleanup probes plus reader AbortError tests. | PASS |
| `safeRequestId` | Allows 1–120 ASCII printable correlation-token characters. | Empty, oversized, whitespace, newline, NUL, and disallowed punctuation reject. | Pure. | Prevents correlation/log injection. | Constant regex. | Narrow shared policy. | Hostile/max-ID tests. | PASS |
| `isCsrfTokenData` | Requires exact one token field bounded to 4096 chars. | Wrong/empty/oversized/extra fields reject; C0 controls are optional hardening because native Fetch rejects them. | Pure. | Token not rendered. | Constant. | Exact guard. | CSRF tests and hostile-token probe. | PASS |
| `isErrorEnvelope` | Recognizes error status, string ID, object error for mapping. | Invalid shape becomes source-less safe error. | Pure. | Source is never directly rendered. | Constant. | Appropriate error guard. | Hostile error tests. | PASS |
| `isExternalSearchData` | Requires exact bounded candidate/warning/page data. | Wrong fields, excess arrays, bad page, duplicate/unknown warnings reject. | Pure bounded traversal. | Closed values before state. | Max 40 candidates and 4 warnings. | Clear composition. | Nested/provider tests. | PASS |
| `isImportResult` | Requires exact UUID/name/state and boolean decisions. | Any wrong field rejects. | Pure. | Prevents untrusted identity/flags. | Constant. | Direct checks. | Malformed result/browser recovery. | PASS |
| `isClassificationData` | Requires exact bounded classification collection and UUID/name/kind/parent. | Bad nested values and excess entries reject. | Pure bounded traversal. | Backend IDs/labels guarded. | Max 1000. | Correct projection guard. | Malformed class tests. | PASS |
| `isExternalCandidate` | Requires exact provider candidate, bounded text/nutrition/warnings, state, optional URI. | Bad provider/state/number/warning/URI rejects; broader URI schemes are optional because backend rejects unsafe mutation URLs. | Pure bounded traversal. | Svelte escapes; backend validates image URL. | Bounded maps/warnings. | Explicit guard. | Candidate/URI tests. | PASS |
| `isExternalDataWarning` | Requires exact provider/code/message equal to closed code. | Unknown JSON values reject; hostile non-native coercion can throw, optional mock-only gap. | Pure for JSON. | Raw warning message never renders. | Constant. | Literal checks could be tighter. | Ordinary malformed warning/browser tests. | PASS |
| `isMacroProfile` | Exact three nonnegative finite numbers. | Missing/extra/negative/NaN/infinite reject. | Pure. | Numeric only. | Constant. | Small guard. | Candidate numeric tests. | PASS |
| `isNumericMap` | Bounded record of bounded keys and nonnegative finite numbers. | Null/excess/NUL/negative/NaN/infinite reject. | Pure bounded traversal. | Keys/values do not reach trusted SQL/HTML. | Max 512. | Reusable helper. | Malformed map tests. | PASS |
| `isCandidateWarning` | Closed candidate-warning enum. | Unknown/non-string reject. | Pure. | Component uses fixed label map. | Constant. | Idiomatic. | Hostile-warning tests. | PASS |
| `isProviderWarningCode` | Closed provider-warning enum. | Unknown/non-string reject. | Pure. | Prevents raw code rendering. | Constant. | Minimal. | Warning/browser tests. | PASS |
| `exact` | Requires fields and rejects unknown keys on ordinary records. | Non-record/missing/extra/array-shaped values reject via checks; non-native accessors are outside JSON transport. | Pure. | DTO shape before state. | Bounded by body cap. | Shared exact guard. | All malformed suites. | PASS |
| `boundedString` | Enforces runtime string min/max. | Wrong/empty/oversized rejects. | Pure. | Reused at all text boundaries. | Constant. | Simple. | Indirect complete client coverage. | PASS |
| `nonnegativeFiniteNumber` | Accepts finite numbers at least zero. | NaN/infinity/negative/non-number reject. | Pure. | No text injection. | Constant. | Idiomatic. | Numeric suite. | PASS |
| `positiveInteger` | Accepts page integer at least one. | Fractional/zero/negative/non-number/infinite reject. | Pure. | Bounds page state. | Constant. | Small helper. | Provider/page tests. | PASS |
| `uuid` | Accepts canonical UUID variant/version form. | Malformed/wrong type rejects. | Pure. | Identity guarded. | Constant regex. | Narrow. | Class/import malformed tests. | PASS |
| `isUri` | Rejects syntactically invalid URI. | Broader schemes remain optional defense in depth; backend rejects unsafe mutation schemes. | Pure URL parse. | URL not rendered. | Constant. | Native parser. | Invalid URI test. | PASS |
| `isRecord` | Narrows non-null objects. | Null/primitives reject; arrays fail later exact checks. | Pure. | Primitive boundary for decoders. | Constant. | Idiomatic type guard. | Indirect malformed suite. | PASS |
| `isAbort` | Detects branded AbortError DOMException. | Plain name-only objects do not trigger special handling. | Pure. | Preserves genuine cancellation. | Constant. | Cross-realm defensive helper. | Fetch/body identity. | PASS |
| `isTimeout` | Detects branded TimeoutError, including `AbortSignal.timeout` reason. | Generic AbortError is not timeout; plain object named TimeoutError remains custom; branded timeout maps. | Pure. | Correct deadline/caller distinction. | Constant. | Paired with `safeFetch` ordering. | Direct/signal/generic/custom matrix. | PASS |
| `isDOMExceptionNamed` | Checks branded DOMException name without trusting arbitrary `.name`. | Same/cross-compatible brands pass; forged plain objects fail safely. | Pure guarded getter. | Cancellation boundary cannot be forged by text. | Constant. | Defensive. | Identity matrix. | PASS |
| `onMount` callback | Loads classifications and aborts search on unmount. | Cleanup is returned; classification generation suppression is optional. | Search controller aborts; sequence not incremented on cleanup. | No data exposure. | At most two reads plus cleanup. | Valid Svelte lifecycle. | Stale search browser evidence. | PASS |
| `loadClassifications` | Loads both backend kinds or fixed recoverable message. | Promise failure safe; retry available. | Parallel requests; overlap can commit older pair, optional. | Guarded IDs/labels. | Two bounded requests. | Straightforward Promise.all. | Classification fixtures/browser shell. | PASS |
| `runSearch` | Latest nonempty query owns visible state. | Empty no-op; success/empty/error fixed; stale paths ignored. | Aborts prior controller and uses sequence ownership; unmount cleanup. | Only validated data/fixed copy. | One bounded request. | Correct latest-wins. | Stale/safe-state browser tests. | PASS |
| `selectCandidate` | Copies guarded candidate into draft and creates one key. | Optional image and warning state handled; prior import state resets. | Immutable, no network. | Guarded input and escaped output. | Bounded shallow copies. | Clear initializer. | Main browser flow. | PASS |
| `toggleClassification` | Adds/removes backend classification IDs. | Null no-op; controlled checkbox values. | Immutable synchronous update. | IDs, not labels, submitted. | Bounded list. | Idiomatic. | UUID request body. | PASS |
| `updatePhysicalState` | Solid clears all liquid fields; liquid retains draft. | Null no-op; closed state values. | Prevents stale provenance crossing state. | Trust invariant preserved. | Constant. | Explicit. | Source/browser flow. | PASS |
| `updateDensity` | Stores finite density and defaults positive absent provenance to manual. | NaN clears; nonpositive remains invalid. | Synchronous. | No provider evidence invented. | Constant. | Correct. | Browser/backend density. | PASS |
| `updateDensitySourceKind` | Accepts manual/estimated and clears provider evidence. | Controlled closed kinds; null no-op. | Synchronous. | Prevents false provider claim. | Constant. | Minimal. | Source/browser. | PASS |
| `submitImport` | Validates draft, name-only confirmation, safe 409 branches, and stable retries. | Invalid local; name confirms; other 409 blocked; ambiguity same-key; unknown generic. | Disabled while importing; key stable except explicit fresh. | Only safe mapped error branches. | One/two bounded requests. | Correct state machine. | Conflict/replay/malformed/status browser. | PASS |
| `validDraft` | Enforces nonblank name, nonnegative finite macros, solid cleanup, liquid provenance. | Invalid stops I/O; backend retains authoritative other bounds. | Pure. | Density trust boundary. | Constant. | Appropriate preflight. | Browser/backend. | PASS |
| `hasValidLiquidDensity` | Positive finite density plus valid manual/estimated or imported source pair. | Wrong/missing/invalid fields reject. | Pure derived predicate. | No unsupported source evidence. | Constant. | Clear. | Density warning browser. | PASS |
| `startFreshImport` | Explicitly rotates key and retries without merge. | Clears conflict state; submit maps result. | New key before request. | Prevents changed-body replay. | Constant plus request. | Correct explicit action. | Browser key difference. | PASS |
| `withRetryAfter` | Appends only bounded positive retry seconds. | Absent/zero unchanged. | Pure. | Header not directly shown. | Constant. | Minimal. | Rate-limit browser. | PASS |
| `visibleSelectedWarnings` | Hides only resolved missing-density warning. | Other warnings retained; stale solid warning is minor UX only. | Pure `$derived`. | Closed labels. | Max eight warnings. | Correct derived state. | Density browser. | PASS |
| external client test fixtures and cases | Deterministic generated-shaped mocks challenge all client boundaries. | Covers success, malformed, wrong status, overflow, cancellation, timeout identity, CSRF, conflicts, and diagnostics. | Fetch restored; reader cleanup asserted. | Hostile data synthetic. | Small fixtures. | Direct focused tests. | 18 tests pass; control token is optional gap. | PASS |
| workflow source-contract test cases | Assert provider/state/provenance/conflict/key/handoff source contracts. | Static tests supplement runtime browser evidence. | No resources. | Assert raw warning message absent. | Small reads. | Appropriate setup. | 4 tests pass. | PASS |
| external workflow browser fixtures/routes/cases | Generated-shaped fixtures and controlled routes cover full vertical workflow. | Includes providers, degraded states, malformed payloads, stale response, timeout, conflicts, replay, local handoff, keyboard, axe. | Route abort/release deterministic. | No real secrets/payloads. | Bounded fixtures. | Strong vertical slice. | 14 focused desktop/mobile and full suite pass. | PASS |
| AdministrationPanel external-workflow branch | Renders workflow only in allowed non-denied branch and passes handoff. | Loading/error do not mount; visibility not authorization. | Child owns search cleanup. | Parent is fail-closed; backend authoritative. | One instance. | Correct composition. | Panel/browser tests. | PASS |
| AdministrationPanel composition tests | Verify loading/error shell, responsive regions, auth notice, branch placement. | Static assertions catch accidental API/auth logic. | No resources. | Explicit server-auth notice. | Constant source reads. | Focused. | 4 tests pass. | PASS |
| `viewImportedItemInLocalSearch` | Converts validated name into ordinary Catalog query, submit, view, and route. | Name comes from validated import result. | Existing Search store owns request lifecycle. | No raw IDs/diagnostics. | One local request. | Reuses workflow. | SearchShell and browser visibility. | PASS |
| SearchShell handoff assertions and callback binding | Verify callback wiring and search-state preservation on denied admin route. | Static assertions cover no reset and correct callback. | No resources. | Visibility not auth. | Constant reads. | Necessary boundary. | Task-specific tests pass. | PASS |

Mandatory audit conclusion: boundary and malformed inputs, error classification, cleanup, cancellation while fetch/body work is in progress, concurrency, security boundaries, bounded work, API necessity, and adversarial tests were inspected for every inventory row. All audited rows pass; optional findings below do not affect the stated task contract.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| optional | `frontend/src/lib/api/external-admin-client.ts:264-266` | `isCsrfTokenData` | Bounded exact CSRF tokens still accept C0 controls before header construction. | A mocked CR/LF token reached a second mocked import call; native Fetch rejects invalid header values, so no demonstrated header injection or unsafe request occurred. | Prefer the server token alphabet or reject C0 controls in a future hardening pass. Non-blocking. |
| optional | `frontend/src/lib/components/ExternalImportWorkflow.svelte:66-81` | `onMount` and `loadClassifications` | Classification loads lack component-owned AbortController/generation. | Static lifecycle inspection; search itself has abort plus sequence ownership and visible workflow tests pass. | Add classification cancellation/generation cleanup later. Non-blocking. |
| optional | `frontend/src/lib/api/external-admin-client.ts:295-302,351-358` | `isExternalCandidate` and `isUri` | URI parsing accepts schemes broader than backend HTTP(S). | `new URL` accepts `javascript:` and `data:`; workflow does not render the URL and backend rejects unsafe mutation URLs. | Align client URI scheme validation later. Non-blocking defense in depth. |
| optional | `frontend/src/lib/components/ExternalImportWorkflow.svelte:189-299` | `validDraft` and controls | Client validation does not mirror every backend bound. | Invalid values receive safe backend 422, with no unsafe success path. | Add field-local mirrors while retaining backend authority. Non-blocking UX hardening. |
| optional | `frontend/src/lib/api/external-admin-client.ts:304-308` | `isExternalDataWarning` | `String(value.provider)` can throw for a non-native object with non-callable `toString`. | Synthetic mocked object produced raw TypeError; ordinary JSON malformed warnings fail closed and UI fallback is safe. | Use literal checks or guarded coercion later. Non-blocking mock-only robustness. |
| optional | coverage tooling | Svelte workflow line coverage | Bun V8 profile does not instrument `.svelte` source lines. | Direct client is 100% functions/lines; 14 focused and 283 full Playwright tests pass; full TypeScript is 95.99% lines. | Add Svelte instrumentation or record Phase 08 tooling exception. Non-blocking evidence debt. |

```yaml
blocking_findings: 0
important_findings: 0
optional_findings: 6
```

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/external-admin-client.test.ts src/lib/components/ExternalImportWorkflow.test.ts src/lib/components/AdministrationPanel.test.ts src/lib/components/SearchShell.test.ts --coverage --coverage-reporter=text --coverage-dir=/tmp/mealswapp-task-255-final-review-coverage` | `frontend/` | 0 | PASS | 50 tests, 275 expectations; external client 100.00% functions and lines. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check` | `frontend/` | 0 | PASS | API drift, typecheck, Vite build, 519 tests, 2415 expectations. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage --coverage-reporter=text --coverage-dir=/tmp/mealswapp-task-255-final-review-full-coverage` | `frontend/` | 0 | PASS | 519 tests; 95.99% full TypeScript line coverage; direct client 100.00%. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/external-import-workflow.spec.ts` | `frontend/` | 0 | PASS | 14 desktop/mobile tests, including generic transport AbortError timeout copy. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e --reporter=line` | `frontend/` | 0 | PASS | 283 passed, 3 intentional skips, 286 total; expected unstubbed-backend proxy noise non-fatal. |
| `python3 scripts/verify-frontend.py --artifact-dir /tmp/mealswapp-task-255-final-review-frontend-verifier --screenshot-stem task-255-final-review` | repository | 0 | PASS | Desktop/mobile verification and screenshots under artifact directory. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/customitem ./internal/dataimporter ./internal/externaldata ./internal/httpapi -count=1` | `backend/` | 0 | PASS | Relevant Task 255 contract packages. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/customitem ./internal/dataimporter ./internal/externaldata ./internal/httpapi -count=1` | `backend/` | 0 | PASS | Relevant race packages. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./internal/dataimporter ./internal/customitem ./internal/externaldata ./internal/httpapi` | `backend/` | 0 | PASS | Relevant backend contract packages. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -count=1` | `backend/` | 1 | ENVIRONMENT/UNRELATED FAIL | Task 240 erasure integration leaves two owner items because Redis endpoints `127.0.0.1:1` and `127.0.0.1:63999` are unavailable; relevant Task 255 packages pass. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1` | `backend/` | 1 | ENVIRONMENT/UNRELATED FAIL | Same Redis-dependent Task 240 failure; relevant Task 255 race packages pass. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `backend/` | 1 | PRE-EXISTING DEPENDENCY FINDING | GO-2026-5970 in `golang.org/x/text@v0.29.0`, fixed in v0.39.0; outside Task 255. |
| `npx --no-install redocly lint api/openapi.yaml` | repository | 0 | PASS WITH PRE-EXISTING WARNING | OpenAPI valid; OAuth callback 302-only warning at line 235. |
| `python3 scripts/validate-traceability.py` | repository | 0 | PASS | Traceability validation passed. |
| `python3 scripts/validate-task-list.py` | repository | 0 | PASS | 263 sequential tasks; task 255 remains `PREPARED`. |
| `git diff --check` | repository | 0 | PASS | No whitespace errors. |
| Inline timeout identity matrix using `bun -e` | `frontend/` | 0 | PASS | Timed-out signal plus generic transport AbortError mapped to `external_request_timeout`; caller AbortError and custom cancellation object remained identical. |
| Inline bounded stream matrix using `bun -e` | `frontend/` | 0 | PASS | Success 256 KiB plus one byte and error 16 KiB plus one byte fail safely; CSRF overflow makes one preflight call only; wrong 2xx cancels body. |
| Inline reader cleanup matrix using `bun -e` | `frontend/` | 0 | PASS | Success/error/CSRF overflow each called reader cancel and release exactly once. |
| Inline hostile CSRF control probe using `bun -e` | `frontend/` | 0 | OPTIONAL GAP CONFIRMED | Synthetic CR/LF token reached a mocked second call; native Fetch rejects such a header. |

## 9. Files Inspected and Staleness Fingerprints

Hashes were recomputed after source inspection, probes, tests, browser runs, and validators. The evidence file is excluded to avoid a self-referential hash.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| `frontend/src/lib/api/external-admin-client.ts` | Task 255 client boundary | F-255-001–008 rechecked; optional hardening only | SHA-256 | `f0cacba9063fb1dae4bfc8b212e6e04a8d3aba2a174f1c7611afdcdb31176c95` |
| `frontend/src/lib/api/external-admin-client.test.ts` | client adversarial tests | timeout/status/body/CSRF identity evidence | SHA-256 | `268539b80293448fca74c68b4917aaa856717e90f797fcc5545b26a7cb417480` |
| `frontend/src/lib/components/ExternalImportWorkflow.svelte` | workflow state/templates | optional lifecycle/client-bound/instrumentation notes | SHA-256 | `eee68537f6780b7fee370455e8992383a508f647a9e549cc63689b13d4e7fe55` |
| `frontend/src/lib/components/ExternalImportWorkflow.test.ts` | workflow source tests | source-contract evidence | SHA-256 | `81ad0e4be588cb8a13fcb934e3f0ea1a6b9592034e2aecc20ede73358d12db14` |
| `frontend/tests/external-import-workflow.spec.ts` | workflow browser tests | desktop/mobile acceptance evidence | SHA-256 | `60fd01a828bb6e5978f222ab7a0765fce5a2b162ac0f955016f3c033f70b323c` |
| `frontend/src/lib/components/AdministrationPanel.svelte` | admin composition context | no repair finding | SHA-256 | `cd758ecf302e2b8d722b0be5bd82ac6cb6e457490a3759261bb620769bdd858f` |
| `frontend/src/lib/components/AdministrationPanel.test.ts` | composition test context | no finding | SHA-256 | `07dbc8d90fbf3d28429ab6acac6754f78ee29a981b5526446edf2facc64540a6` |
| `frontend/src/lib/components/SearchShell.svelte` | local-search handoff context | no finding | SHA-256 | `f7bdfae6ec146f0db01136318d0c27bb07ca1fd287b66aacc620c850b103c7f3` |
| `frontend/src/lib/components/SearchShell.test.ts` | handoff assertion context | no finding | SHA-256 | `88f065d461baa8f7a7a1b21730355801ed9d9da177c9d945dd84363b01a4b51a` |
| `frontend/src/lib/api/generated.ts` | generated endpoint and DTO source | contract context | SHA-256 | `46c7a49e096ce66720b598c06bca76d4444c3e718940eabc31c48dbe8b1ac9c0` |
| `frontend/src/lib/api/auth-client.ts` | permissive decoder comparison | final path does not use it | SHA-256 | `5fa89c0b2d71fab4edbc0395d402b09e6f27ccf9ced5d6fa422917c382c57c3e` |
| `api/openapi.yaml` | endpoint/status/DTO source | contract context | SHA-256 | `f3143eaa58c136fbe8f7db6652ef030cb851d831f62dded65935544bb460ab46` |
| `docs/design/DESIGN-009.md` | ExternalSearchProxy/DataImporter design | design context | SHA-256 | `85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b` |
| `backend/internal/customitem/service.go` | density/image/name contract | F-255-001 context | SHA-256 | `28d9981b711f94f57c864b27daf4c83e34952acc088c1f98ed22caf910f0793d` |
| `backend/internal/dataimporter/service.go` | normalization/conflict contract | F-255-001/002 context | SHA-256 | `1d2f801934ed6f0cdcf101384acfe508f83124311f7f3a3b5b0dbb2fb7be66ae` |
| `backend/internal/httpapi/import_controller.go` | HTTP 409/201/CSRF contract | F-255-002/007 context | SHA-256 | `04e0e65035302d15501dd44e0ba1327ee1af71f22db97ce733d0df5dd4483de1` |
| `docs/implementation/preparations/task-255.md` | preparation manifest | current evidence source | SHA-256 | `c6243d5b152e6ca8a8afa0d1aed2ba8c50d6dc3bb571ae743af82300ba0c0112` |
| `docs/implementation/02_TASK_LIST.md` | current task row/status | read-only status boundary | SHA-256 | `d520c8413a2b3df8c0f569fafa5fe3224be93d459c3970f665c75f48e22e45af` |

```yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The predecessor Task 255 review was REJECTED and its implementation fingerprints predate the timeout repair; repaired files were re-inspected and rehashed here."
  - "The preparation report is an input claim, not a substitute for current evidence; its current hash and all implementation hashes were checked after this review."
  - "The shared worktree contains concurrent task changes; context files are hashed and excluded from repair ownership where noted."
```

## 10. Coverage and Exceptions

- [x] Focused coverage ran against current Task 255 sources.
- [x] Full frontend coverage ran against current repository sources.
- [x] Direct `external-admin-client.ts` coverage is 100.00% functions and 100.00% lines.
- [x] Changed branches were manually inspected; generic transport timeout, caller/custom identity, body cleanup, status allowlists, request IDs, and CSRF rejection were directly challenged.
- [x] Desktop/mobile browser coverage supplements source-only Svelte tests with 14 focused and 283 full Playwright passes.
- [ ] Svelte component source lines were instrumented; Bun does not include `.svelte` files in its V8 profile. This is the optional tooling finding above, not a silent task-row exception.

```yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "/tmp/mealswapp-task-255-final-review-coverage and /tmp/mealswapp-task-255-final-review-full-coverage"
observed_line_coverage: "100.00% direct external-admin-client.ts; 95.99% full TypeScript aggregate; Svelte source lines not instrumented"
coverage_passed: true
```

Coverage finding: all instrumented Task 255 client code is covered, and component behavior is exercised in Chromium at desktop/mobile sizes. The tooling gap is explicit and non-blocking because all task component and Playwright assertions pass.

## 11. Negative and Regression Checks

- [x] F-255-001 through F-255-008 were rechecked against current source and current tests.
- [x] Genuine timeout signal plus generic transport AbortError maps to fixed timeout error; caller AbortError and custom cancellation reasons preserve identity.
- [x] Response-reader AbortErrors preserve identity and reader cancel/release happen on abort, decode, and overflow paths.
- [x] Success, error, and CSRF body limits are byte-based; declared and streamed overflow fail closed; wrong 2xx bodies are canceled.
- [x] Search/classification/import/CSRF status allowlists are operation-specific and reject undocumented 2xx statuses.
- [x] CSRF preflight is generated, credentialed, strict HTTP 200, exact, bounded, request-ID checked, and completed before import.
- [x] Request IDs are bounded printable correlation tokens; unsafe error IDs are discarded and unsafe success IDs reject the envelope.
- [x] Raw provider diagnostics, error messages, socket details, unknown conflict codes, and malformed nested payloads do not reach UI text.
- [x] Idempotency keys are browser-generated and memory-only, stable across ambiguity/name confirmation, and rotated only by explicit fresh recovery.
- [x] Search supersession aborts prior work and sequence ownership prevents stale responses from overwriting current visible results.
- [x] No new dependency, persistence, authorization bypass, SQL/command/path boundary, secret, or raw external payload exposure was introduced by Task 255.
- [x] Shared auth/admin shell remains fail-closed; client visibility is not treated as backend authorization.
- [x] No generated/cache/build/temporary artifact was intentionally added by this review.
- [x] Required validators, focused backend tests, race tests, vet, frontend build/typecheck, browser, and frontend verifier passed where applicable.
- [x] Full backend/race gates are not green only because unrelated Redis-dependent Task 240 integration is configured with unavailable endpoints; relevant Task 255 packages pass.
- [x] Vulnerability scan has a pre-existing GO-2026-5970 dependency finding outside Task 255 scope.

Findings: only the six optional observations in section 7 remain. They are defense-in-depth, lifecycle hardening, client-feedback, mock-totality, or tooling issues with no blocking or important impact on the stated task criteria.

## 12. Decision

A task may be `PASSED` only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains. Those gates pass for Task 255 after the generic transport AbortError timeout repair.

Before accepting the decision, the evidence validator was run:

```bash
python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-255-review.md
```

```yaml
decision: "PASSED"
reason: "F-255-001 through F-255-008, bounded/status/CSRF/request-ID/security behavior, acceptance tests, symbol audits, and current hashes pass; only six non-blocking observations remain."
failed_criteria:
  - ""
failed_or_unaudited_symbols:
  - ""
recommended_next_action: "None for Task 255 review. Track the six optional hardening/instrumentation observations separately."
```

## 13. Repair Context

Not applicable: this independent final re-review is `PASSED`. No repair instructions are required.
