# Task 255 preparation — External Search and Import UI

## Outcome

Task 255 remains `PREPARED`; `docs/implementation/02_TASK_LIST.md` was not edited. The repair addresses all required remaining findings in `docs/implementation/reviews/task-255-review.md`:

- Liquid drafts now require positive density plus explicit `manual`, `estimated`, or fully evidenced `imported` provenance. Editing density defaults to auditable administrator-supplied provenance, the UI displays the resolved curation state, and switching to solid clears all liquid-only fields.
- Only `name_conflict_confirmation_required` enters the merge-confirmation path. Provider-identity, idempotency-key, and unknown conflicts use bounded non-merge recovery actions; a fresh idempotency attempt rotates its key.
- External candidates, provider warnings, classifications, and import decisions are decoded through exact nested runtime guards before entering Svelte state. Malformed values produce a safe recoverable message instead of reaching template assumptions.
- Superseded searches and unmounts abort in-flight work, and sequence ownership prevents stale responses from overwriting a newer result.
- Search and classification responses now admit only HTTP 200, while imports admit only HTTP 201; every other 2xx response fails closed.
- Success and error response bodies are read through cancellation-aware 256 KiB and 16 KiB limits respectively, including declared-length and streamed overflow handling.
- Request IDs are retained only when they match the shared 1–120 character printable correlation-token policy; malformed or oversized IDs are discarded from errors and rejected in success envelopes.
- Caller and response-reader `AbortError` values now pass through unchanged; only a genuine `TimeoutError` is mapped to the external timeout application error. Reader cancellation and lock release remain guaranteed on failed reads and overflow.
- The default import CSRF preflight now uses the task-owned strict decoder: only documented HTTP 200 is accepted, success and error bodies are bounded and canceled on rejection, envelope/request-ID/token data is validated, and an unsafe request ID cannot reach the import request.
- A genuine `TimeoutError` carried by an aborted signal, including `AbortSignal.timeout()` and branded `DOMException` values, now maps to `external_request_timeout` before generic signal-reason preservation. Caller `AbortError` and custom cancellation reasons, including an object merely named `TimeoutError`, retain identity unchanged; the workflow renders only the fixed safe timeout copy.

No backend, OpenAPI, generated API contract, shared Task 254/256 component, or task-list change belongs to this repair.

## Repair baseline

- Fixed implementation reference: `81ca40ce00cb667ea29243ed2d34068e11229a69` plus the dirty shared Phase 08 worktree.
- Review evidence: `docs/implementation/reviews/task-255-review.md`, decision `REJECTED`, reviewed at `2026-07-21T20:38:00Z`.
- Repair started from the review fingerprints below. Concurrent unrelated edits were left intact.

| Path | Review SHA-256 |
| --- | --- |
| `frontend/src/lib/api/external-admin-client.ts` | `c2839474011934d8048252497829c7e6d3ba3657e4291953c9ab08c78e54bf0f` |
| `frontend/src/lib/api/external-admin-client.test.ts` | `1818ff3945d71a5e0c3757b24c2db311412acd2c3b70275e5320f1fc15ba474d` |
| `frontend/src/lib/components/ExternalImportWorkflow.svelte` | `eee68537f6780b7fee370455e8992383a508f647a9e549cc63689b13d4e7fe55` |
| `frontend/src/lib/components/ExternalImportWorkflow.test.ts` | `81ad0e4be588cb8a13fcb934e3f0ea1a6b9592034e2aecc20ede73358d12db14` |
| `frontend/tests/external-import-workflow.spec.ts` | `6cf64a1534ae95bd508982517ddc8f909c5e91319b56d8d3b376ded3650134c0` |

## Exact repaired symbols

### Client boundary

`frontend/src/lib/api/external-admin-client.ts` contains these executable symbols after repair:

| Symbol | Repair evidence |
| --- | --- |
| `ExternalAdminClientError.constructor` | Retains bounded application errors and safe retry metadata. |
| `searchExternalFoods` | Passes cancellation and admits data only through `isExternalSearchData`. |
| `loadAdminClassifications` | Admits exact nested classifications only through `isClassificationData`. |
| `importCuratedItem` | Retains CSRF/cookies/idempotency behavior and admits exact import decisions only through `isImportResult`. |
| `createImportIdempotencyKey` | Creates canonical browser UUID keys, including deliberate fresh-attempt rotation. |
| `safeFetch` | Maps a genuine timeout from either fetch rejection or an aborted signal reason before preserving every other caller cancellation reason; hides ambiguous transport diagnostics. |
| `decodeResponse` | Enforces each operation's documented success status and parses only bounded success/error bodies. |
| `fetchImportCsrfToken` | Uses the generated credentialed CSRF request with exact HTTP 200, bounded decoding, strict envelope/request-ID checks, and exact bounded token data. |
| `safeResponseError` | Preserves only allowlisted 409 source codes and only safe request IDs. |
| `malformedResponse` | Produces bounded safe malformed-response failures. |
| `safeMessageForStatus` | Supplies distinct local copy for name, provider, idempotency, and unknown conflicts. |
| `safeCodeForStatus` | Allows only `name_conflict_confirmation_required`, `provider_identity_conflict`, and `idempotency_key_conflict` through a 409. |
| `categoryForStatus`, `parseRetryAfter`, `isErrorEnvelope` | Preserve existing safe status behavior. |
| `readBoundedText` | Rejects declared or streamed overflow, invalid UTF-8, and cancels failed reads. |
| `safeRequestId` | Admits only 1–120 character printable correlation tokens. |
| `isCsrfTokenData` | Requires one non-empty bounded CSRF token and rejects additional preflight data. |
| `isExternalSearchData` | Enforces exact search fields, candidate/warning limits, nested decoders, and page bounds. |
| `isImportResult` | Enforces exact UUID/name/state and boolean `merged`/`replayed` decisions. |
| `isClassificationData` | Enforces exact UUID/name/kind/parent classification projections. |
| `isExternalCandidate` | Enforces exact provider identity, text bounds, state, macros, micronutrients, URI, and closed warnings. |
| `isExternalDataWarning` | Enforces exact provider/code/message and requires the closed message decision to equal its code. |
| `isMacroProfile`, `isNumericMap` | Reject non-finite, negative, oversized, and malformed nutrition maps. |
| `isCandidateWarning`, `isProviderWarningCode` | Enforce closed warning vocabularies. |
| `exact`, `boundedString`, `nonnegativeFiniteNumber`, `positiveInteger`, `uuid`, `isUri`, `isRecord`, `isAbort`, `isTimeout`, `isDOMExceptionNamed` | Primitive defensive-decoder and branded cancellation helpers. |

### Workflow component

`frontend/src/lib/components/ExternalImportWorkflow.svelte` contains these executable units after repair:

| Symbol/unit | Repair evidence |
| --- | --- |
| `onMount` callback | Loads classifications and aborts the current search on unmount. |
| `loadClassifications` | Retains safe classification loading. |
| `runSearch` | Aborts superseded requests and commits state only for the latest sequence. |
| `selectCandidate` | Starts one curation identity without inventing provider density evidence. |
| `toggleClassification` | Retains classification selection. |
| `updatePhysicalState` | Clears density, provenance, and serving-volume fields when changing to solid. |
| `updateDensity` | Persists positive edited density and defaults missing provenance to `manual`. |
| `updateDensitySourceKind` | Allows only administrator-supplied or administrator-estimated provenance and clears incompatible provider evidence. |
| `submitImport` | Routes only normalized-name conflicts to confirmation; blocks all other 409s from merge. |
| `validDraft`, `hasValidLiquidDensity` | Enforce backend-compatible solid/liquid density invariants before POST. |
| `startFreshImport` | Rotates the key only for an explicit fresh idempotency recovery. |
| `withRetryAfter` | Retains bounded retry guidance. |
| `visibleSelectedWarnings` | Removes the missing-density warning only after valid density and provenance exist. |
| density curation template | Requires density/provenance and renders `data-density-curation-state`. |
| conflict templates | Separate `data-import-conflict` from `data-import-blocked-conflict`; only the former renders `Confirm merge`. |

### Focused tests

`frontend/src/lib/api/external-admin-client.test.ts` contains 18 tests. In addition to the prior provider, classification, import, conflict, nested-payload, status, body-bound, request-ID, transport, idempotency, and URI coverage, focused adversarial tests now preserve fetch-level caller abort, preserve success/error body-reader abort while asserting cancel/release cleanup, map direct and `AbortSignal.timeout()` DOM timeout failures, map a timed-out signal even when the transport substitutes a fresh generic `AbortError`, preserve branded `AbortError` and custom cancellation reasons by identity, exercise the no-token happy path, and reject/cancel oversized or HTTP 201 CSRF responses plus a hostile request ID before import.

`frontend/src/lib/components/ExternalImportWorkflow.test.ts` contains 4 component-contract tests: provider/states/cancellation; editable density provenance and classifications; conflict/idempotency branches; and keyboard-native local handoff.

`frontend/tests/external-import-workflow.spec.ts` contains these helpers and browser tests:

| Exact symbol/test | Coverage |
| --- | --- |
| `sessionEnvelope`, `profileEnvelope`, `entitlementEnvelope`, `json`, `stubAdminShell`, `externalEnvelope`, `importEnvelope`, `localSearchEnvelope` | Contract-faithful controlled shell and DTO fixtures with valid UUIDs. |
| `searches every provider, paginates partial results, curates warnings/classifications, confirms conflict, and opens the local result` | Backend-faithful liquid request validation, manual provenance, resolved warning state, name-only confirmation, same-key replay, local handoff, keyboard, axe. |
| `keeps provider, idempotency, and unknown conflicts out of the merge-confirmation path` | No merge action for non-name conflicts; provider refresh; fresh-key rotation; `confirmNameConflict: false`. |
| `contains malformed nested search and import payloads without crashing the workflow` | Malformed candidate warnings and malformed import decisions remain safe and recoverable. |
| `ignores a stale external search response after a newer query wins` | Abort/sequence runtime regression. |
| `shows loading, empty, rate-limit, timeout, and unavailable states without raw diagnostics` | Existing safe state matrix. |
| `shows safe timeout copy when the search signal times out` | Browser-level aborted-signal regression proves the genuine DOM timeout maps to fixed workflow copy while raw reason text stays hidden. |
| `replays one ambiguous import with the same key and displays one local item identity` | Backend-valid liquid retry with one stable key. |

## Finding disposition

| Finding | Disposition and regression evidence |
| --- | --- |
| F-255-001 liquid provenance | Fixed by `updateDensity`, `updateDensitySourceKind`, `updatePhysicalState`, `validDraft`, and `hasValidLiquidDensity`. Main browser route returns 422 for invalid liquid provenance and asserts the exact valid body. |
| F-255-002 409 collapse | Fixed by `safeCodeForStatus`, `safeMessageForStatus`, subtype-aware `submitImport`, `startFreshImport`, and separate conflict templates. Unit and browser matrices cover all backend codes plus unknown 409. |
| F-255-003 shallow payload guard | Fixed by exact nested candidate, warning, classification, macro, micronutrient, URI, UUID, and import-decision decoders. Unit and browser malformed payloads fail safely. |
| F-255-004 undocumented 2xx statuses | Fixed by the `SUCCESS_STATUS` operation allowlist in `decodeResponse`; valid envelopes at search/classification 201 and import 200 now fail closed. |
| F-255-005 unbounded bodies/request IDs | Fixed by bounded stream decoding, cancellation on overflow, strict success-envelope request IDs, and discard-on-error for unsafe IDs. Focused tests cover >256 KiB success, >16 KiB error, 121-character, whitespace, newline, NUL, and valid 120-character IDs. |
| F-255-006 cancellation classification | Fixed by `safeFetch`, `decodeResponse`, `readBoundedText`, and `isAbort`. Fetch and success/error body-reader `AbortError` objects are preserved by identity; `TimeoutError` alone receives timeout mapping; failed body reads still cancel and release the reader. |
| F-255-007 permissive CSRF preflight | Fixed by `fetchImportCsrfToken` using the generated credentialed request plus `safeFetch`/`decodeResponse`. The preflight requires HTTP 200, applies the 256 KiB/16 KiB bounds, cancels rejected bodies, validates request IDs and exact bounded token data, and never starts import after oversized, 201, or hostile-ID responses. |
| F-255-008 timeout signal classification | Fixed by ordering branded signal-timeout classification before generic transport `AbortError` and aborted-signal preservation in `safeFetch`. Focused coverage uses real `AbortSignal.timeout()`, substitutes a fresh generic transport `AbortError`, and identity-checks caller `AbortError` plus custom reasons; desktop/mobile Playwright verifies the workflow shows `The request timed out. Try again.` without the raw timeout reason. |
| O-255-004 stale search/unmount | Fixed with component-owned `AbortController`, sequence ownership, unmount cleanup, and desktop/mobile stale-response browser coverage. |
| O-255-005 source-only component confidence | Runtime desktop/mobile browser tests now exercise provenance resolution, subtype rendering, malformed payload containment, and stale search behavior. |
| O-255-006 client coverage | Focused Bun coverage reports `100.00%` functions and `100.00%` lines for `external-admin-client.ts`. |

## Verification evidence

| Command | Result |
| --- | --- |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/external-admin-client.test.ts src/lib/components/ExternalImportWorkflow.test.ts src/lib/components/AdministrationPanel.test.ts src/lib/components/SearchShell.test.ts --coverage --coverage-reporter=text --coverage-dir=/tmp/mealswapp-task-255-timeout-repair-coverage` | PASS: 50 tests, 0 failures, 275 expectations; task-owned client 100% functions and 100% lines. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/external-import-workflow.spec.ts` | PASS: 14 tests across desktop/mobile, including safe timeout-signal workflow copy. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check` | PASS: generated API drift check, typecheck, build, and 519 unit/component tests with 2415 expectations. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e --reporter=line` | PASS on clean rerun: 283 passed, 3 intentional skips. Expected deliberately unstubbed proxy failures were non-fatal. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/dynamic-substitution-filters.spec.ts:107 --project=mobile-chromium --repeat-each=3` | PASS: the unrelated aggregate failure passed 3 consecutive isolated repetitions, confirming a transient cross-suite race. |
| `python3 scripts/verify-frontend.py --artifact-dir /tmp/mealswapp-task-255-timeout-repair --screenshot-stem task-255-timeout-repair` | PASS: desktop/mobile verification and screenshots. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/dataimporter -run 'TestServiceConfirmRejectsInvalidDraftsBeforePersistence' -count=1` | PASS: real backend contract rejects invalid liquid density/provenance. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks with ordered dependencies. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS with the pre-existing OAuth 302-only warning at line 235. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-255-review.md` | PASS: rejected review evidence remains structurally valid for independent re-review. |
| `git diff --check` | PASS. |

The aggregate browser run logged expected proxy failures for deliberately unstubbed local backend requests; all assertions passed. No coverage exception is required for the repaired task-owned client.

## Final SHA-256 fingerprints

Captured after all implementation and verification work at `2026-07-21T21:24:06Z`.

| Path | SHA-256 | Ownership/state |
| --- | --- | --- |
| `frontend/src/lib/api/external-admin-client.ts` | `f0cacba9063fb1dae4bfc8b212e6e04a8d3aba2a174f1c7611afdcdb31176c95` | Repaired Task 255 client. |
| `frontend/src/lib/api/external-admin-client.test.ts` | `268539b80293448fca74c68b4917aaa856717e90f797fcc5545b26a7cb417480` | Repaired Task 255 unit tests. |
| `frontend/src/lib/components/ExternalImportWorkflow.svelte` | `eee68537f6780b7fee370455e8992383a508f647a9e549cc63689b13d4e7fe55` | Repaired Task 255 workflow. |
| `frontend/src/lib/components/ExternalImportWorkflow.test.ts` | `81ad0e4be588cb8a13fcb934e3f0ea1a6b9592034e2aecc20ede73358d12db14` | Repaired Task 255 component tests. |
| `frontend/tests/external-import-workflow.spec.ts` | `60fd01a828bb6e5978f222ab7a0765fce5a2b162ac0f955016f3c033f70b323c` | Repaired Task 255 browser tests. |
| `frontend/src/lib/components/AdministrationPanel.svelte` | `cd758ecf302e2b8d722b0be5bd82ac6cb6e457490a3759261bb620769bdd858f` | Shared file unchanged by repair. |
| `frontend/src/lib/components/AdministrationPanel.test.ts` | `07dbc8d90fbf3d28429ab6acac6754f78ee29a981b5526446edf2facc64540a6` | Shared file unchanged by repair. |
| `frontend/src/lib/components/SearchShell.svelte` | `f7bdfae6ec146f0db01136318d0c27bb07ca1fd287b66aacc620c850b103c7f3` | Shared file unchanged by repair. |
| `frontend/src/lib/components/SearchShell.test.ts` | `88f065d461baa8f7a7a1b21730355801ed9d9da177c9d945dd84363b01a4b51a` | Shared file unchanged by repair. |
| `frontend/src/lib/api/generated.ts` | `46c7a49e096ce66720b598c06bca76d4444c3e718940eabc31c48dbe8b1ac9c0` | Generated contract unchanged. |
| `docs/design/DESIGN-009.md` | `85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b` | Design unchanged. |
| `api/openapi.yaml` | `f3143eaa58c136fbe8f7db6652ef030cb851d831f62dded65935544bb460ab46` | Shared OpenAPI unchanged by repair. |
| `backend/internal/customitem/service.go` | `28d9981b711f94f57c864b27daf4c83e34952acc088c1f98ed22caf910f0793d` | Backend density contract unchanged. |
| `backend/internal/httpapi/import_controller.go` | `04e0e65035302d15501dd44e0ba1327ee1af71f22db97ce733d0df5dd4483de1` | Backend conflict contract unchanged. |
| `docs/implementation/02_TASK_LIST.md` | `d520c8413a2b3df8c0f569fafa5fe3224be93d459c3970f665c75f48e22e45af` | Concurrent modification preserved; not edited by this repair. |
| `docs/implementation/reviews/task-255-review.md` | `c8049226e02fd62854b7e4c2608185a780c0c06defa8bfd56cab888a63874904` | Rejected review repaired by this preparation evidence. |
