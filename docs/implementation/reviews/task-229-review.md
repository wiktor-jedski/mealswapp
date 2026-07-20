# Review Evidence: Task 229 — Daily Diet State Coordination and Authoritative Selection

~~~yaml
task_id: 229
phase: "07.01"
component: "DESIGN-001: SearchView"
static_aspect: "SearchView Daily Diet state coordination and authoritative selection"
input_status: "OPEN (preserved; task-status edits were prohibited)"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-18T11:49:35Z"
review_agent: "Codex independent owner re-review after F-229-01/F-229-02 repair"
evidence_file: "docs/implementation/reviews/task-229-review.md"
baseline_ref: "HEAD a4e31367485b03269e90b5607f2057c9568bb5b1 plus cumulative dirty Phase 07.01 worktree"
baseline_confidence: "HIGH"
inventory_symbol_count: 27
audited_symbol_count: 27
inventory_source_count: 27
code_review_skill_invoked: true
code_review_skill_invocation_count: 1
relevant_language_guides: "TypeScript, Svelte, security, async/concurrency, common-bugs, architecture, performance, and universal-quality guidance applied"
review_template_path: "docs/implementation/reviews/REVIEW_TEMPLATE.md"
review_template_available: false
fallback_review_template_path: "review.txt"
fallback_review_template_sha256: "f741e00f06e76a90c26ee263fc4485698f4fde182711352138179369f3186b20"
prior_task_review_sha256_before_refresh: "f057676408866ec9110628641293dccc983107fd9d6ba2c66ef6035f4a4e6da0"
repair_context_required: true
~~~

## 1. Task Source

**Description:** Phase 07.01: replace the latest-operation counter with an explicit cancellable read/mutation lifecycle, prevent overlapping mutations and lost confirmed writes, remove the stale-macro optimistic replacement projection, and establish one coherent selected-diet source across collection, search modes, optimization input, logout, and account changes.

**Task row:** `docs/implementation/02_TASK_LIST.md:236`; Task 229 remains `OPEN` and was not changed.

**Depends On:** Task 228, `PASSED` in the current task list. Its refreshed preparation and review were read as inherited Daily Diet client/retry-key evidence, not as proof of Task 229 behavior.

**Testing Coverage Exceptions:** `None` in the Task 229 row. The repository's accepted Phase 07 frontend coverage deviation is recorded in `docs/implementation/04_OPEN.md:330-337`; this review adds no exception and verifies every Task 229-specific repair branch.

**Design and architecture sources:** Full `docs/design/DESIGN-001.md` (`SearchView`), `docs/architecture/ARCH-001.md`, and `docs/design/01_TECH_STACK.md` were read. The relevant boundary is the Svelte stores + TanStack Query + SearchView component coordination layer.

**Verification criteria:** deterministic deferred-promise coverage for load/load, load/create, create/load, replace/select/failure, replace/load, delete/load, clear/logout, duplicate activation, abort propagation, and out-of-order completion; no lost confirmed writes or newer user-state overwrite; no stale-macro replacement projection; one selected-diet source across collection, search, optimization, reload/deletion/empty state, identity changes, and Catalog/Substitution round trips; correct Alternative request/query-key behavior; mutation-gated optimization; component tests; and frontend validation/build/browser evidence.

**Template note:** `docs/implementation/reviews/REVIEW_TEMPLATE.md` is absent in this checkout. The complete root `review.txt` fallback and `docs/implementation/reviewer-prompt.md` were read, and the repository's 13-section evidence schema was used without creating the missing template. The prior rejected Task 229 review was read in full, its pre-refresh hash was recomputed, and its findings were independently rechecked against current source.

**Decision at a glance:** F-229-01 and F-229-02 are repaired and closed. The Alternative query is disabled until the authoritative selected ID exists, and pre-aborted/synchronously settling Daily Diet operations no longer install ghost ownership or leave the store wedged. All original coordination, serialization, cancellation, stale-macro, identity, selection, and race requirements pass.

## 2. Pre-Review Gates

- [x] The exact Task 229 row was read; it remains `OPEN`.
- [x] Dependency Task 228 is `PASSED`; its preparation and review boundary were read.
- [x] The refreshed `docs/implementation/preparation/task-229-preparation.md` was read in full; its current implementation hashes were recomputed and match.
- [x] Full `docs/design/DESIGN-001.md`, `docs/architecture/ARCH-001.md`, and `docs/design/01_TECH_STACK.md` were read.
- [x] Full fallback `review.txt` and `docs/implementation/reviewer-prompt.md` were read; the requested template path was checked and is absent.
- [x] The prior rejected Task 229 review was read in full and its stale claims were checked against current source and tests.
- [x] The current Daily Diet lifecycle, selection store, search state/query bridge, SearchResults execution bridge, components, identity boundary, supporting client, and listed tests were audited at grouped-symbol level.
- [x] `code-review-skill` was invoked exactly once; its complete guidance and relevant TypeScript, Svelte, security, concurrency, common-bug, architecture, performance, and quality guidance were applied.
- [x] F-229-01 was directly probed and its deterministic no-selection/selection regression passed.
- [x] F-229-02 was directly probed for pre-aborted read/mutation recovery and synchronous mutation settlement; all passed.
- [x] Focused and full frontend tests, coverage, typecheck, build, generated API check, traceability, task-list validation, browser verification, and scoped diff checks passed.
- [x] One unrelated generated-contract drift test was recorded as out of scope: it fails on a concurrent optimization/OpenAPI `requestId` rule, not on any Task 229 source or contract.
- [x] No production file, task row/status, unrelated implementation, or dependency evidence was edited; only this review artifact was refreshed.
- [x] All Task 229 acceptance criteria and audited symbols pass; no blocking or important finding remains.

~~~yaml
pre_review_gates_passed: true
blocking_issue: "None. F-229-01 and F-229-02 are repaired, independently verified, and closed."
~~~

## 3. Review Baseline and Change Surface

The baseline is `HEAD a4e31367485b03269e90b5607f2057c9568bb5b1` plus the cumulative dirty Phase 07.01 worktree. `git status --short --branch` shows many concurrent backend, API, frontend, migration, preparation, and review changes. Task 229 attribution was reconstructed from the authoritative task row, refreshed preparation manifest, prior rejected review, current content hashes, direct call-site searches, and function-level behavior. Nearby aggregate changes were excluded unless they were a necessary supporting boundary for Task 229.

No merge, reset, checkout, staging, cleanup, production-code edit, unrelated-file edit, or task-status edit was performed. Merging into this shared dirty worktree would change state outside the requested review scope, so the supplied worktree was preserved as the review baseline.

The audited contract is:

1. `daily-diet.ts` owns explicit read and mutation lifecycles; newer reads supersede older reads, reads queue abortably behind mutations, mutations cancel reads and cannot overlap, and only the lifecycle owner commits.
2. Create, replace, and delete install decoded server DTO state only; replacement never pairs draft entries with old aggregate macros.
3. `selected-daily-diet.ts` is the one memory-only selected identity consumed by collection selection, Alternative request/key construction, optimization input, logout, and account changes.
4. Search mode state has no competing selected-diet copy; Catalog and Substitution omit the ID, while Daily Diet Alternative emits it only when authoritative selection exists.
5. Parent/auth identity cleanup clears Daily Diet state even when Daily Diet mode components are unmounted.

The repaired change surface satisfies both earliest-boundary failures from the prior review: `buildSearchQueryOptions` gates Alternative execution on `selectedId !== null`, and `load`/`runMutation` reject already-aborted callers before state/owner mutation while installing lifecycle ownership before synchronous execution can settle.

## 4. Acceptance Criteria Checklist

| # | Criterion | Result | Evidence and review conclusion |
|---:|---|---|---|
| 1 | Replace the latest-operation counter with explicit cancellable read/mutation lifecycles. | PASS | `ReadLifecycle` and `MutationLifecycle` own controllers and promises; ownership is installed before async execution; no obsolete Daily Diet counter remains. |
| 2 | Load/load and out-of-order completion preserve the newer server snapshot. | PASS | `load/load aborts the older read and only the newer server snapshot is installed`; lifecycle identity rejects stale success and failure commits. |
| 3 | Load/create cancels stale reads without losing the confirmed create. | PASS | `load/create cancels the stale read and preserves the confirmed create`; the older signal is aborted and the created DTO remains authoritative. |
| 4 | Create/load waits for mutation confirmation before reading authoritative state. | PASS | `create/load queues the read behind mutation confirmation without losing the write`; list I/O remains at zero until the mutation settles. |
| 5 | Replace/select/failure preserves newer selection and the last authoritative diet. | PASS | `replace/select/failure keeps newer selection and the last authoritative diet`; failure does not restore an obsolete selection snapshot. |
| 6 | Replace/load and delete/load serialize reads after authoritative mutation settlement. | PASS | Both named deferred tests pass; neither read can commit a pre-mutation snapshot or resurrect a confirmed deletion. |
| 7 | Duplicate activation and distinct overlapping mutations cannot lose or fork confirmed writes. | PASS | Duplicate create returns the same active promise and one API call; distinct replacement is rejected with `daily_diet_mutation_in_progress` before I/O. |
| 8 | Caller abort propagates while reading, mutating, or waiting behind a mutation, retaining authoritative state. | PASS | Started-read, started-mutation, queued-read, pre-aborted read, and pre-aborted create/replace/delete paths pass; abort listeners reach API signals and settlement leaves idle state. |
| 9 | Clear/logout aborts active work and ignores late completions. | PASS | `clear/logout aborts read and mutation lifecycles and ignores late completions`; store and selection reset to initial state despite late API resolution. |
| 10 | Replacement never projects stale macros/draft data; success installs server DTO and failure retains authoritative state. | PASS | Pending replacement leaves old DTO/macros visible; failed replacement retains it; success installs only decoded server entries, name, timestamps, and aggregate macros. Optimization requires idle mutation and a selected server DTO. |
| 11 | Selection is one coherent source across collection, reload, deletion, empty state, logout, account changes, and optimization. | PASS | `selectedDailyDietId` is memory-only; reload disappearance, selected/unselected deletion, empty reload, clear, identity changes, selector, and optimization all use/reconcile the same ID. |
| 12 | Catalog/Substitution round trips preserve selection while omitting it; Alternative emits the ID, reacts in its key, and cannot execute without it. | PASS | Mode/request/key tests pass; the repaired query test observes `enabled=false` and zero fetches before selection, then one selected-ID request after selection. |
| 13 | Component and repository gates cover the repaired behavior without source-contract drift. | PASS | Focused 105-test run, full 414-test run, typecheck, build, generated API check, traceability/task validators, browser desktop/mobile verification, and scoped diff checks pass. |

The repaired acceptance evidence adds exactly the missing boundary coverage from the prior rejection: `Daily Diet Alternative performs no query until authoritative selection exists`, `pre-aborted read settles without API I/O and a later read remains usable`, `pre-aborted mutations settle without API I/O and later mutations remain usable`, and `synchronous mutation execution failure settles ownership and permits retry`.

## 5. Changed-Symbol Inventory

| # | Grouped symbol/unit | File:line | Task 229 surface audited | Result |
|---:|---|---|---|---|
| 1 | `DailyDietState`, lifecycle types, controller options/interface, public exports | `frontend/src/lib/stores/daily-diet.ts:21-75,226-253` | Server-owned collection state, memory-only selection boundary, and public controller API | PASS |
| 2 | `load` | `frontend/src/lib/stores/daily-diet.ts:82-116` | Read supersession, mutation queue, caller abort, and lifecycle-owned commit | PASS |
| 3 | `create` and create-intent ownership | `frontend/src/lib/stores/daily-diet.ts:118-142` | Duplicate activation, authoritative create commit, and retry-key lifecycle | PASS |
| 4 | `replace` | `frontend/src/lib/stores/daily-diet.ts:144-156` | Authoritative replacement reconciliation and overlap guard | PASS |
| 5 | `select` and `remove` | `frontend/src/lib/stores/daily-diet.ts:158-178` | Valid selection, deletion reconciliation, and mutation overlap guard | PASS |
| 6 | `clear` | `frontend/src/lib/stores/daily-diet.ts:180-188` | Logout/account clear, abort, and late-completion invalidation | PASS |
| 7 | `runMutation`, `finishMutation`, `finishCancelledMutation` | `frontend/src/lib/stores/daily-diet.ts:190-225,263-275` | Mutation ownership, commit/failure/cancel settlement, and synchronous execution safety | PASS |
| 8 | `projectError`, `retainSelection`, `upsert`, overlap and abort helpers | `frontend/src/lib/stores/daily-diet.ts:255-341` | Safe error projection, selection retention, cancellation, and reconciliation helpers | PASS |
| 9 | Deferred/abortable controller fixtures and controller tests | `frontend/src/lib/stores/daily-diet.test.ts:57-687` | All 27 controller tests, including races, selection, macro, clear, duplicate, pre-abort, and sync-throw cases | PASS |
| 10 | `selectedDailyDietId` writable | `frontend/src/lib/stores/selected-daily-diet.ts:1-6` | Sole memory-only selected identity | PASS |
| 11 | Search mode union, `setMode`, and mode projections | `frontend/src/lib/stores/search.ts:13-158` | No mode-local duplicate selection and compatible round trips | PASS |
| 12 | `setDailyDietId`, `buildSearchRequest`, `searchRequestKey` | `frontend/src/lib/stores/search.ts:371-470` | Shared selection emission, deterministic key construction, and mode omission rules | PASS |
| 13 | Compile-time mode-shape assertions | `frontend/src/lib/stores/search-state.types.ts:21-204` | Incompatible mode fields and duplicate Alternative ID prevention | PASS |
| 14 | Search mode/selection/request tests | `frontend/src/lib/stores/search.test.ts:77-534` | Mode transitions, omission/emission, key determinism, and round trips | PASS |
| 15 | Inherited Daily Diet client operations and signal forwarding | `frontend/src/lib/api/daily-diet-client.ts:46-97,130-153,288-300` | List/create/replace/delete caller signals and safe errors | PASS; inherited Task 228 boundary |
| 16 | Inherited Daily Diet client exact/error/abort tests | `frontend/src/lib/api/daily-diet-client.test.ts:1-430` | Exact contract, status, error, and signal behavior | PASS; inherited Task 228 evidence |
| 17 | `buildSearchQueryOptions`, `createSearchQueryOptions` | `frontend/src/lib/api/search-client.ts:165-209` | Selected-ID request/key reactivity and Alternative execution enablement | PASS; F-229-01 repaired |
| 18 | Search query option/request/key tests | `frontend/src/lib/api/search-client.test.ts:629-811` | Query construction, reactive selection, no-selection guard, and zero-fetch proof | PASS |
| 19 | `committedSearchStore`, options bridge, `createQuery` | `frontend/src/lib/components/SearchResults.svelte:35-60,80-108` | Actual SearchView query execution path and enabled propagation | PASS |
| 20 | SearchResults source assertions | `frontend/src/lib/components/SearchResults.test.ts:1-90` | Committed query, result execution, and cache indicator wiring | PASS |
| 21 | Daily Diet draft identity, save/reset, pending-save UI | `frontend/src/lib/components/DailyDietCollection.svelte:69-212,254-351` | Draft reset, create intent invalidation, and no optimistic DTO projection | PASS |
| 22 | Daily Diet collection component tests | `frontend/src/lib/components/DailyDietCollection.test.ts:1-35` | Identity reset, retry ownership, and pending-click suppression | PASS |
| 23 | Alternative loading, selector, identity effect | `frontend/src/lib/components/DailyDietControls.svelte:30-43,79-123` | Collection loading, shared selection, and optimization handoff | PASS |
| 24 | Daily Diet controls source/mutation-gate tests | `frontend/src/lib/components/DailyDietControls.test.ts:1-18` | One selected source and mutation-gated optimization | PASS |
| 25 | `OptimizationWorkflow` selected DTO, `canSubmit`, diet-change effect | `frontend/src/lib/components/OptimizationWorkflow.svelte:23-75` | Server-derived selected macros and pending-mutation gate | PASS |
| 26 | Optimization controller and receiving lifecycle tests | `frontend/src/lib/stores/optimization.ts:35-145`; `optimization.test.ts:1-420` | Shared selected-ID handoff without a competing Daily Diet selection source | PASS; supporting interaction boundary |
| 27 | SearchShell/auth identity clear and round-trip support | `frontend/src/lib/components/SearchShell.svelte:82-100,297-305`; `auth-session.ts:112-160`; `SubstitutionRequest.test.ts:1-74` | Parent-owned clear on logout/account change and request-shape support | PASS |

~~~yaml
inventory_symbol_count: 27
inventory_complete: true
audited_symbol_count: 27
inventory_source_count: 27
generated_groupings:
  - "Rows group only tightly coupled symbols with one contract and one evidence boundary; lifecycle, query-guard, selection, component, and supporting-client boundaries remain distinct."
  - "Task 228 client and auth/optimization files are supporting interaction boundaries inspected for Task 229 signal, selection, and identity behavior, not re-attributed as new Task 229 implementation."
~~~

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| `DailyDietState` and exports | State contains server collections, lifecycle status, safe error, and no persisted identity/key data; one controller exposes operations. | Initial, success, empty, error, and clear projections are intentional. | Selection is a separate writable; no owner or promise is exposed in state. | No secrets or raw server errors are placed in the store. | Collection reconciliation is bounded by the server response and uses one upsert/filter pass. | Public API is typed and TSDoc/design-traced. | Initial-state, storage, and full controller tests pass. | PASS |
| `load` | One read lifecycle owns commits; newer reads supersede; reads wait behind mutation settlement. | Success, empty, rejection, stale completion, queued abort, clear, and pre-abort paths settle intentionally. | Caller signal is chained; pre-aborted input rejects before aborting existing work or installing state; `finally` releases listeners/owner. | Only decoded client DTOs enter state; errors are projected through safe AppError. | One list request per accepted read; superseded/queued reads avoid unnecessary API I/O. | Explicit lifecycle is clearer and more local than the removed counter. | Load/load, load/create, create/load, delete/load, abort, queued abort, pre-abort, and direct probes pass. | PASS |
| `create` | One request fingerprint owns one in-memory retry key and one mutation owner; successful state is server DTO only. | Duplicate active create shares its promise; key generation failure, abort, API failure, success, clear, and changed intent are handled. | Pre-aborted mutation reaches no API and does not install a ghost owner; active writes cannot overlap. | Key never enters Svelte state/browser storage; client contract and safe errors remain inherited boundaries. | Duplicate activation produces one API request; no optimistic collection projection. | Caller-owned retry intent remains closure-local and minimally public. | Replay, duplicate, rotation, storage, random failure, abort, pre-abort, and race tests pass. | PASS |
| `replace` | Only one mutation owner may replace; commit is the decoded server DTO for the same ID. | Pending, success, failure, abort, overlap, pre-abort, and synchronous execute throw are covered through `runMutation`. | Caller signal is chained; ownership is installed before execute and cleared in `finally`. | Request/response remain at the strict Daily Diet client boundary; raw errors do not render. | One replace request and one upsert on success; no draft projection. | Small wrapper delegates common lifecycle semantics. | DTO/macro, failure/selection, replace/load, pre-abort, sync-throw/retry, and full tests pass. | PASS |
| `select` and `remove` | Selection accepts only an ID present in authoritative collections; delete reconciles only after server success. | Invalid selection is ignored; selected/unselected delete, failure, abort, pre-abort, empty, and clear paths are intentional. | Delete is serialized by the mutation owner; selected ID is cleared only when the deleted ID disappears. | No client-supplied macros or identity are trusted for selection state. | Delete performs one API call and one collection filter; selection update is separate and memory-only. | Selection is centralized rather than duplicated in search state. | Selection, deletion/reload, empty reload, round-trip, and mutation tests pass. | PASS |
| `clear` | Clear invalidates every active owner, create intent, collection state, and selected ID. | Logout/account change during read or mutation ignores late success and failure. | Internal controllers abort; pointers are nulled before late continuations can commit. | User-owned state does not cross identities. | No network call is introduced by clear. | One explicit teardown path is used by parent/auth boundaries. | Clear/logout late-completion and SearchShell identity tests pass. | PASS |
| `runMutation` and settlement helpers | Active mutation is installed before async execution; only the current lifecycle can commit or settle visible state. | Pre-abort rejects before state mutation; synchronous throw, async rejection, normal success, caller abort, clear, and late completion all settle. | Chained listeners are canceled in `finally`; overlap is rejected; ownership cannot be left installed after sync failure. | `projectError` exposes only safe AppError data. | No extra retry or mutation I/O is performed on pre-abort/overlap. | Shared generic helper removes duplicated lifecycle branches. | Pre-aborted mutation recovery and synchronous-execution-failure/retry test directly covers the repair. | PASS |
| `projectError`, `retainSelection`, `upsert`, overlap and abort helpers | Helpers preserve authoritative DTO identity, selection membership, and fixed safe errors. | Unknown errors, client errors, empty collections, existing/new IDs, and abort reasons are deliberate. | Abort listeners are removed; stale owner identity prevents late writes. | Raw error fields and diagnostics are not exposed. | Linear collection operations; no unbounded retained work. | Helpers are small, typed, and not duplicated in components. | Indirect controller tests plus client error tests cover paths. | PASS |
| Deferred/abortable controller fixtures and controller tests | Fixtures model out-of-order resolution, ignored abort, caller abort, queued waiting, and synchronous throw. | The full 27-test controller matrix includes nominal, stale, failure, empty, pre-aborted, and recovery paths. | Assertions inspect signals, API-call counts, state settlement, and late completion behavior. | Safe-error and memory-only key assertions remain present. | Deferred fixtures make ordering deterministic without sleeps or external services. | Test helpers are local and readable. | Focused run passes 27 controller tests; no unresolved Task 229 edge gap remains. | PASS |
| `selectedDailyDietId` | One memory-only writable is the selected-diet identity source. | Set, retain, clear, deletion, disappearance, empty reload, and identity clear are handled. | It is independent of mode state and mutation snapshots. | No browser storage or session credential is stored. | O(1) writable updates and derived subscriptions. | Minimal public store prevents parallel selection copies. | Store coverage is 100%; integration and source tests pass. | PASS |
| Search mode union and `setMode` | Discriminated modes cannot carry incompatible fields; selection survives mode changes outside mode-owned state. | Catalog/Substitution/Alternative transitions reset submitted page state without dropping authoritative selection. | Mode transition does not mutate the selected-diet store. | No untrusted mode fields bypass generated request construction. | Small immutable projections; no duplicate collection fetch. | Typed union and projections match DESIGN-001. | Type assertions and mode-transition tests pass. | PASS |
| `setDailyDietId` / `buildSearchRequest` / `searchRequestKey` | One shared selected ID is emitted only for Alternative; other modes omit it; key includes selected identity. | Null selection produces no Alternative ID and a deterministic no-selection key; selected changes alter key/request. | Derived query options react to both search state and selected store. | Generated optional field is used only under the correct mode. | Deterministic sorting/serialization avoids duplicate cache entries. | Explicit selected-ID parameter is testable while defaulting to the shared store. | Request, key, mode, selection, and round-trip tests pass. | PASS |
| Type-level mode assertions | Search mode types reject duplicate IDs, substitution fields, and collection fields in incompatible variants. | Positive complete states and negative `@ts-expect-error` constructions compile as intended. | Compile-time prevention complements runtime query guard. | No client identity is added to mode state. | Type-only checks add no runtime cost. | Discriminated union is idiomatic and minimal. | Typecheck and compile-time assertions pass. | PASS |
| Search mode/store tests | Tests document selection retention, request omission/emission, and deterministic keys. | Mode re-entry, page reset, duplicate fields, filters, substitutions, and null/selected ID behavior pass. | Selection remains independent of mode transitions. | Generated request shape is asserted rather than handwritten client state. | No network or timers required. | Test names map directly to design contracts. | Full search-store suite and focused suite pass. | PASS |
| Inherited Daily Diet client | Canonical operations forward signals and return strict decoded DTOs; controller owns coordination above it. | Exact statuses, safe errors, malformed payloads, and abort are handled by Task 228 boundary. | API signal receives chained controller abort. | Strict decoder prevents hostile response data entering lifecycle state. | Bounded response acquisition and one request per operation are inherited. | No alias/wrapper drift found. | Task 228 review/preparation and current client tests pass. | PASS |
| Inherited Daily Diet client tests | Client fixtures prove the controller's API seam is strict and signal-aware. | Exact list/create/replace/delete, malformed payload, error, and abort cases pass. | No controller repair depends on a weak client cast. | Safe mapper and ownership-safe errors remain intact. | Bounded body tests cover response acquisition. | Supporting tests are isolated from Task 229 attribution. | Focused/full suites pass. | PASS |
| `buildSearchQueryOptions` / `createSearchQueryOptions` | An enabled query must be executable under authoritative state; Alternative requires non-null selected ID. | Catalog/Substitution guards remain correct; Alternative null selection is disabled; selected state enables a request. | Derived store reacts to selection changes and changes query key/request together. | Backend mode-shape contract requires `dailyDietId`; invalid no-ID execution is blocked. | No premature network fetch; cache keys remain deterministic. | Guard is a local invariant at the query option boundary. | F-229-01 regression, direct probe, and full search tests pass. | PASS |
| Search query tests | Query options prove request, key, enablement, cache, timeout, and selection behavior. | Zero-fetch no-selection, selected activation, abort, cache hit/miss/stale, timeout, and error paths pass. | Query observer receives reactive option changes and selection key invalidation. | Safe error mapping and bounded fetch signal remain covered. | Fresh cache bypasses network; timeout clears its timer/listener. | Tests use deterministic QueryObserver/fetch fixtures. | Added guard test closes the prior coverage defect. | PASS |
| SearchResults query bridge | Committed SearchView state reaches TanStack Query and passes the combined enabled guard. | Empty/unsubmitted text is hidden; selected Alternative execution follows options; result/error/cache projections remain safe. | Query options react to committed search and authoritative selection; local component flag cannot enable an unsafe option. | Results render decoded generated data and safe mapped errors. | One query/cache path; no duplicate fetch bridge introduced. | Runes/store bridge remains typed and simple. | SearchResults source tests, full suite, build, and browser verification pass. | PASS |
| SearchResults tests | Source assertions protect committed query, visible state, cache indicator, and result wiring. | Empty, loading, error, page, Catalog, and source-summary paths are asserted. | The downstream query-option test covers the actual no-selection guard boundary. | No raw ID or error leak is introduced. | Static tests avoid unnecessary DOM dependency; build verifies compilation. | Assertions are scoped to the component contract. | Full component suite passes; no untested Task 229 execution boundary remains. | PASS |
| `DailyDietCollection` | Draft state is distinct from server DTO state; server aggregate is authoritative after save. | Identity, edit, success, failure, reset, pending-click, and selection-hydration paths are intentional. | Identity reset clears local state and controller state; create intent is discarded on edits. | No idempotency key or user identity is rendered/persisted. | Aggregate calculation is local draft-only; save has one mutation call. | Component owns draft UI while store owns server coordination. | Component source assertions, full tests, build, and browser flows pass. | PASS |
| Daily Diet collection component tests | Tests protect basis-aware draft setup, identity cleanup, retry invalidation, and pending suppression. | Source contracts cover all local draft fields and save behavior. | No stale draft can survive logout/account switch. | No sensitive state is asserted in rendered output. | Static assertions are deterministic. | Test scope matches current Bun/Svelte setup. | Full suite and browser verification pass. | PASS |
| `DailyDietControls` | Selector and optimization receive the shared selected ID; collection load is identity-scoped. | Auth loading/anonymous/error/empty/selection/entitlement paths are intentional. | Parent clear plus local identity effect prevents cross-account state; downstream query guard handles no selection. | Only server collection DTOs populate labels/macros. | Selection is a simple store update; no duplicate list request on mode changes. | Component reads one selected source directly. | Controls source and full component/browser tests pass. | PASS |
| Daily Diet controls tests | Source assertions lock selector and optimization to the shared source and idle mutation gate. | Both source-level invariants pass. | The query-option regression separately covers downstream execution. | No raw data boundary is added. | No runtime dependency needed for source assertion. | Small focused test is appropriate for current environment. | Full focused/full suites pass. | PASS |
| `OptimizationWorkflow` selected DTO, `canSubmit`, diet-change effect | Optimization request uses selected ID and selected server DTO; pending mutation disables submit. | Missing selection, identity change, entitlement, invalid tolerance, busy, terminal, and retry display paths are intentional. | Controller is reset on selected-diet change/disposal; mutation state gates activation. | Macro targets are read-only server-derived values; no draft macros enter optimization. | At most three alternatives render; polling ownership is in controller. | Component delegates lifecycle and keeps form logic local. | Optimization component/store tests and full suite pass. | PASS |
| Optimization controller and receiving lifecycle tests | Supporting controller consumes the same selected ID without creating another Daily Diet selection source. | Diet changes invalidate stale jobs/results; late polling is ignored; submission and terminal states remain bounded. | Its internal operation counter is an unrelated optimization lifecycle, not the removed Daily Diet counter. | Strict optimization client owns its own contract/error boundary. | Polling is bounded by its own task scope. | Supporting boundary was inspected but not re-attributed. | Current optimization tests pass; no Task 229 regression found. | PASS |
| SearchShell/auth identity clear and request round trips | Parent shell clears Daily Diet state on auth identity change; Catalog/Substitution preserve selection while omitting ID. | Logout, anonymous/expired transition, account switch, and request-mode round trips are intentional. | Clear runs even if Daily Diet controls are unmounted; late controller work loses commit authority. | Auth session exposes only sanitized projection; no token/ID persistence is added by Task 229. | One identity effect and shared writable subscriptions; no extra request. | Parent-owned cleanup matches the architecture boundary. | SearchShell/auth/request tests, full suite, and browser desktop/mobile verification pass. | PASS |

All mandatory audit questions pass for the Task 229 surface: malformed/edge inputs at the repaired boundaries are handled; return/error paths are intentional; listeners and owners settle on success, failure, and cancellation; cancellation applies while waiting and executing; concurrent operations cannot commit stale state; user-controlled data crosses only strict generated/client boundaries; query, collection, and request work is bounded; no duplicate selection/lifecycle helper remains; and adversarial tests cover the prior failures.

## 7. Findings

| ID | Prior severity | Status | File:line | Symbol | Repair verification |
|---|---|---|---|---|---|
| F-229-01 | `[blocking]` | CLOSED / REPAIRED | `frontend/src/lib/api/search-client.ts:171-209`; caller `frontend/src/lib/components/SearchResults.svelte:44-60` | `buildSearchQueryOptions` / `createSearchQueryOptions` | The repaired enablement is `state.query.trim().length > 0 && (state.mode !== "daily_diet_alternative" || selectedId !== null)`. The deterministic test at `search-client.test.ts:759-789` observes `enabled=false` and zero fetches with Alternative text and no authoritative selection, then one fetch with `dailyDietId: "diet-1"` after selection; direct runtime probing confirms `alternativeEnabledWithoutSelection: false`. Backend validation at `search_validation.go:182-185` still requires the ID, so the client no longer issues the guaranteed invalid request. |
| F-229-02 | `[blocking]` | CLOSED / REPAIRED | `frontend/src/lib/stores/daily-diet.ts:82-115,190-225` | `load` / `runMutation` lifecycle installation and settlement | `load` and `runMutation` now reject already-aborted signals before aborting existing owners or installing visible state. Accepted operations install lifecycle ownership/state before the async body can synchronously settle. Tests at `daily-diet.test.ts:559-624` prove no API I/O, initial idle state, later read/mutations remaining usable, and synchronous execution failure releasing ownership for retry; direct probing confirms `AbortError`, zero create I/O, idle state, successful later read, and no lifecycle wedge. |

~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 0
decision_basis: "PASSED because both prior blocking findings are repaired and independently verified, and the full original Task 229 coordination/serialization/abort/stale-macro/identity/selection/race matrix passes."
~~~

## 8. Commands Run

| Command | Working directory | Exit code | Result | Evidence |
|---|---|---:|---|---|
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/stores/daily-diet.test.ts src/lib/stores/search.test.ts src/lib/api/search-client.test.ts src/lib/components/SubstitutionRequest.test.ts src/lib/components/DailyDietCollection.test.ts src/lib/components/DailyDietControls.test.ts src/lib/components/OptimizationWorkflow.test.ts` | `frontend/` | 0 | PASS | 105 tests, 426 expectations; all Task 229 deferred, selection, component, query, and repair tests pass. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | `frontend/` | 0 | PASS | 414 tests, 1,877 expectations across 37 files. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage` | `frontend/` | 0 | PASS | 94.01% aggregate lines; `daily-diet.ts` 99.55%; selected-diet store 100%; search client 100%. The accepted Phase 07 exception remains unchanged; all Task 229 repair branches are covered. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck` | `frontend/` | 0 | PASS | TypeScript no-emit check. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | `frontend/` | 0 | PASS | Vite production build; 205 modules transformed. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check:api-types` | `frontend/` | 0 | PASS | Generated API types current. |
| `python3 scripts/validate-traceability.py` | repository root | 0 | PASS | Traceability validation passed. |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS | 237 sequential tasks with ordered dependencies; Task 229 remains `OPEN`. |
| `python3 scripts/verify-frontend.py` | repository root | 0 | PASS | Chromium desktop/mobile scenarios and screenshots passed; expected anonymous 401 console messages were classified as pass. |
| `git diff --check -- <Task 229 scoped frontend files>` | repository root | 0 | PASS | No whitespace errors. |
| `rg` obsolete Daily Diet lifecycle/optimistic-selection tokens in scoped production files | repository root | 0 | PASS | No `operation = 0`, `currentOperation`, `optimisticReplace`, `canOptimisticallyReplace`, or `$dailyDietStore.selectedId` remains in the Task 229 scope. The unrelated optimization controller's own operation counter was inspected and excluded. |
| Direct Bun F-229-01/F-229-02 runtime probe | `frontend/` | 0 | PASS | Alternative no-selection options report `enabled=false`; pre-aborted read/create reject with `AbortError` without I/O; later read settles successfully and store remains usable. |
| `python3 scripts/test_generate_api_types.py` | repository root | 1 | OUT OF SCOPE / NOT A TASK 229 FAILURE | One unrelated `OperationResponseDriftTest` assertion fails because the concurrent optimization/OpenAPI acknowledgement `requestId` schema lacks bounds/pattern. No Task 229 file, selection contract, or Daily Diet lifecycle is implicated; `bun run check:api-types` passes. |
| `/home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-229-review.md` | repository root | 0 | PASS | Final structural evidence is validator-clean. |

The repository-wide `scripts/check.py`, backend gates, OpenAPI lint, and unrelated Phase 07 aggregate gates were not used as Task 229 decision gates. The scoped frontend, repository, browser, direct-runtime, and evidence-validator checks provide direct coverage of this frontend coordination task. The single failing generator-drift test is retained above for transparent scope accounting and belongs to concurrent optimization/OpenAPI work.

## 9. Files Inspected and Staleness Fingerprints

These SHA-256 values identify the exact reviewed content in the shared dirty worktree. The current review file is intentionally not self-hashed; its pre-refresh content was hashed as the prior rejected artifact in the metadata and below.

| File | Purpose / audited surface | Hash algorithm | Current SHA-256 |
|---|---|---|---|
| `review.txt` | Complete fallback review template | SHA-256 | `f741e00f06e76a90c26ee263fc4485698f4fde182711352138179369f3186b20` |
| `docs/implementation/reviewer-prompt.md` | Repository review instruction | SHA-256 | `92c9b71361a50868becf0b9a9895071bdd657e8c092afb8c1b19691cb569386d` |
| `docs/implementation/02_TASK_LIST.md` | Exact Task 229 row/status; not edited by review | SHA-256 | `a44ed4b1ed8bdaebba1510b1b18c5214c43051e77ba307ae1ddab2d1fa3dc6f4` |
| `docs/design/DESIGN-001.md` | Full SearchView design source | SHA-256 | `34d699ae93a8e5465199f3494ed41813c675f1cb3b9c1c6b6e611ba66c6142c7` |
| `docs/architecture/ARCH-001.md` | SPA/SearchView architecture boundary | SHA-256 | `03fcbae9676ecf278e72a621c7fd9911d0c3dc6ee15eaebcec308d010cc76833` |
| `docs/design/01_TECH_STACK.md` | Svelte/TanStack Query/Bun stack | SHA-256 | `64e2cf45ec039db597244678b17e8028f4705b86dcad01e7051e3e686d6f9338` |
| `docs/implementation/04_OPEN.md` | Accepted Phase 07 coverage deviation and related open actions | SHA-256 | `c60ad3f42f5bcb66ed6d143c0f05b761ae80b3fb8be5f1541ea340d65c26c527` |
| `docs/implementation/preparation/task-229-preparation.md` | Refreshed Task 229 repair attribution, acceptance matrix, commands, and hashes | SHA-256 | `44ec1670ac0c402873eefdafb3bf1a7e6e43a1033d8cc4d1114379b771e0367e` |
| `docs/implementation/preparation/task-228-preparation.md` | Inherited strict client/retry boundary | SHA-256 | `2b1c3201303d03f2a15d80b4743a78a3cdf3fc56b4b9dda444c4cb76de5f5352` |
| `docs/implementation/reviews/task-228-review.md` | Inherited dependency review | SHA-256 | `f089287fed28666c6526400f3141973606454acd1a17ce190c5cc30cf8e13539` |
| `docs/implementation/reviews/task-229-review.md` before refresh | Prior rejected Task 229 review; superseded artifact | SHA-256 | `f057676408866ec9110628641293dccc983107fd9d6ba2c66ef6035f4a4e6da0` |
| `frontend/src/lib/stores/daily-diet.ts` | Controller and lifecycle | SHA-256 | `9a321420f52aadbc924870098a194fe6fe2b57c844076bbece0746a83572cf20` |
| `frontend/src/lib/stores/daily-diet.test.ts` | Deferred/abortable controller tests | SHA-256 | `86694403519286d0da6a6cda8839571506167d14fa1afac5c92247b857a1a7d0` |
| `frontend/src/lib/stores/selected-daily-diet.ts` | Shared selection store | SHA-256 | `75435238ff8c0a17107ce7b2be601531e3edc636c94a937a7ae995170201ef0d` |
| `frontend/src/lib/stores/search.ts` | Mode and request/key bridge | SHA-256 | `32ea31c61bafd59f92cb28013fcd646ef097400cb18d48d5359c346057947778` |
| `frontend/src/lib/stores/search.test.ts` | Selection/mode/request tests | SHA-256 | `84b2663ef9a788a17802e0ed52ba0a772afca2faf87ddd20e7a57c31a4b66911` |
| `frontend/src/lib/stores/search-state.types.ts` | Compile-time mode checks | SHA-256 | `b5ad04da63f8d1ce7f33151a674a4a426462dace59cc3bb8f90f13c40cda55c0` |
| `frontend/src/lib/api/search-client.ts` | Query enablement and selection bridge | SHA-256 | `25ffbf5ad14c75614478363bed774ae22b8d2e77682c90c5bf35464afe75ce1f` |
| `frontend/src/lib/api/search-client.test.ts` | Query option/key and F-229-01 tests | SHA-256 | `532bd87a173f6eacfbe038ac128b8d6e15af6582a61ad96bdfcf4664ace41a83` |
| `frontend/src/lib/api/daily-diet-client.ts` | Inherited client/signal boundary | SHA-256 | `35d60162f1f5e9a3db350b95d93e6b2c894e9926be5305b406a2815e9ad03db6` |
| `frontend/src/lib/api/daily-diet-client.test.ts` | Inherited client tests | SHA-256 | `72ae560716e8abf580cc173e9f603f238de45029f7ce7170cda659d6960cd941` |
| `frontend/src/lib/api/generated.ts` | Generated SearchRequest contract | SHA-256 | `166722ae537251aeb09ec58f9b53557d929bc4f75c258dc770dc93090f0882ae` |
| `frontend/src/lib/components/SearchResults.svelte` | SearchView query execution bridge | SHA-256 | `d3e55d4eacd2ef1b1116e9d290e4c6766292f4b8c2da65e51e9cf8e150b4ba85` |
| `frontend/src/lib/components/SearchResults.test.ts` | SearchResults assertions | SHA-256 | `655478ed9a9bff02119aec54e143be46847a1f6055cd97b4b2d4456f8af68b26` |
| `frontend/src/lib/components/DailyDietCollection.svelte` | Draft/create boundary | SHA-256 | `1428689f367cd04f32e562f132c39b79f609fa0ae7fa9fd104b69e9b20d8ca04` |
| `frontend/src/lib/components/DailyDietCollection.test.ts` | Draft/pending-save tests | SHA-256 | `c0869a7ec40af0806231e72bf900d320d60ca39a3742f302fb5e43a48ab6cf65` |
| `frontend/src/lib/components/DailyDietControls.svelte` | Selector/optimization handoff | SHA-256 | `6e0c9327665d07d8c1f0e03d058254c577c327f9e49ca3a1d24a45ae14296de4` |
| `frontend/src/lib/components/DailyDietControls.test.ts` | Selector/mutation-gate tests | SHA-256 | `d2e170cc0e1ac8553c45faffb21f4bbbec078d71e807895c1e73625b7b2a3712` |
| `frontend/src/lib/components/OptimizationWorkflow.svelte` | Authoritative DTO/mutation gate | SHA-256 | `620e825cd23e258fee69ccb42899e00c01f2dc7a53df5d5b8e3d9cc3c6f00b33` |
| `frontend/src/lib/components/OptimizationWorkflow.test.ts` | Optimization component tests | SHA-256 | `022b8e15728f1808c7397ee2ffd9b31f9c56d8af0915c28ab6b3da1ceb87a28d` |
| `frontend/src/lib/stores/optimization.ts` | Supporting selected-ID receiving boundary | SHA-256 | `a2e959c819daa0a0a1d1cf685e13c36926bcf24d9e55786205a3b46c3301019e` |
| `frontend/src/lib/stores/optimization.test.ts` | Supporting optimization lifecycle tests | SHA-256 | `d1c25017a9a48f1fb576b549f20a1b8c6e46d5a9854a7c4d8758004a9f9a8efb` |
| `frontend/src/lib/components/SearchShell.svelte` | Parent identity/logout clear | SHA-256 | `584b0e0dba4ec6a8d38217816daa910b09a2bdefc7f3d0d26cf11adbba5fc6e8` |
| `frontend/src/lib/components/SearchShell.test.ts` | Identity assertions | SHA-256 | `a8282445a4b7b08571598dd8bb768ede8c5aea20f80e38df0b519619fa3a55a6` |
| `frontend/src/lib/stores/auth-session.ts` | Supporting logout/auth identity boundary | SHA-256 | `97944edf13db85c71873e0dcd1a93a5a62335df1f26e3bce7f04995341be1323` |
| `frontend/src/lib/stores/auth-session.test.ts` | Supporting identity tests | SHA-256 | `aa36b0c304e310819a7f57fe2097e499d1b1c3cc9cb17ed9a93ed4b6ce28779e` |
| `frontend/src/lib/components/SubstitutionRequest.test.ts` | Request round trips | SHA-256 | `52eee8b0a6dd2e0f2b3921e7b8f6382f1b62c5fda533329a182193f817665f2c` |
| `backend/internal/httpapi/search_validation.go` | Backend required-ID contract used by F-229-01 proof | SHA-256 | `1c01b989d2d469425f945592c0e65cd76c1c5d9d35bede4b8ff4720760029b` |

~~~yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The requested docs/implementation/reviews/REVIEW_TEMPLATE.md is absent; complete review.txt was used as the documented fallback."
  - "The prior rejected Task 229 artifact had a stale F-229-01/F-229-02 result and an inaccurate prior metadata hash; its current pre-refresh content was rehashed as f057676408866ec9110628641293dccc983107fd9d6ba2c66ef6035f4a4e6da0 before replacement."
  - "The refreshed preparation hash is 44ec1670ac0c402873eefdafb3bf1a7e6e43a1033d8cc4d1114379b771e0367e; all listed Task 229 implementation/test hashes were recomputed and match its manifest."
  - "The shared worktree remains cumulatively dirty across Phase 07.01; attribution is symbol-level and unrelated changes were excluded."
~~~

## 10. Coverage and Exceptions

- [x] Focused Task 229 tests ran: 105 tests and 426 expectations.
- [x] Full frontend tests ran: 414 tests and 1,877 expectations.
- [x] Frontend typecheck, production build, generated API check, traceability, task-list validation, browser verification, and scoped diff checks passed.
- [x] Coverage ran: 94.01% aggregate lines; Daily Diet store 99.55%; selected-diet store 100%; search client 100%.
- [x] The repaired branches are directly covered: no-selection Alternative zero-fetch, reactive selected-ID activation, pre-aborted read recovery, pre-aborted create/replace/delete recovery, and synchronous mutation throw/retry.
- [x] The accepted Phase 07 frontend coverage deviation in `docs/implementation/04_OPEN.md:330-337` remains unchanged. Task 229 adds no new exception and raises its principal lifecycle store to 99.55% line coverage.

~~~yaml
coverage_required: true
coverage_exception_allowed: true
coverage_report_path: "Bun test --coverage stdout; no persistent coverage artifact committed"
observed_line_coverage: "94.01% aggregate; 99.55% daily-diet.ts; 100% selected-daily-diet.ts; 100% search-client.ts"
coverage_passed: true
coverage_reason: "All Task 229-specific changed-symbol and repaired-edge branches are covered and pass; the pre-existing Phase 07 frontend deviation is documented and unchanged."
~~~

## 11. Negative and Regression Checks

- [x] No latest-operation counter or optimistic replacement helper remains in the scoped Daily Diet lifecycle/selection production surface.
- [x] Replacement does not mutate collection entries or aggregate macros before server success.
- [x] Successful replacement installs the decoded server DTO; failed replacement retains the prior authoritative collection.
- [x] Mutations serialize and reads queue behind mutation settlement; load/load supersession and stale completion cannot overwrite newer state.
- [x] Caller abort after operation start reaches API signals; queued and already-aborted callers settle without leaving visible loading/mutation state.
- [x] Synchronous mutation execution failure releases ownership and permits a later retry.
- [x] Clear/logout aborts active read/mutation work and late completions cannot repopulate state.
- [x] Selected identity is memory-only, retained only when present after reload, cleared on selected deletion/disappearance/empty reload/clear, and preserved across Catalog/Substitution round trips.
- [x] Selector and optimization use the shared selected ID; pending Daily Diet mutation gates optimization and optimization reads server-derived macros.
- [x] Alternative execution is safe without a selection: enabled is false and no fetch occurs; after selection one request includes the selected ID.
- [x] Strict Daily Diet client signal forwarding and inherited decoder/error boundaries pass their current tests.
- [x] No new dependency, API, storage, secret, generated-contract, or architectural boundary was introduced by Task 229.
- [x] No obsolete aliases/counters/duplicate selection fields remain in the scoped production paths.
- [x] Task row remains `OPEN`; no task status or unrelated implementation was changed by this review.

The only unrelated negative check failure is the recorded Python generated-contract drift test for a concurrent optimization/OpenAPI acknowledgement `requestId` schema. It is outside Task 229, while the current generated output check and all Task 229 request/selection checks pass.

## 12. Decision

A task may be `PASSED` only when all acceptance criteria and audited symbols pass, evidence is current, every reviewed file is hashed, and no blocking or important finding remains. Those conditions are satisfied for Task 229.

~~~yaml
decision: "PASSED"
reason: "F-229-01 and F-229-02 are repaired: Alternative execution is gated on authoritative selection, pre-aborted and synchronously settling Daily Diet lifecycles settle safely, and every original Task 229 coordination, mutation/read serialization, abort, stale-macro, identity, selection, and race requirement passes."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "Leave Task 229 OPEN until the phase orchestrator or project owner performs the separate status transition; do not change task status in this review."
~~~

## 13. Repair Context

This is a repaired re-review of the prior rejected Task 229 evidence. The prior review's two blocking findings were:

1. **F-229-01:** Alternative search options remained enabled with text and no authoritative selected Daily Diet ID, allowing a guaranteed backend-invalid request.
2. **F-229-02:** Already-aborted read/mutation signals could run `throwIfAborted` before lifecycle installation, leaving a rejected owner and stuck loading/mutation state; synchronous execution failure had no explicit ownership regression test.

The refreshed preparation and current source repair those boundaries:

- `buildSearchQueryOptions` and `createSearchQueryOptions` now require a non-null shared selection for Alternative enablement, while preserving reactive query-key/request changes after selection. The added QueryObserver test proves zero premature fetches and one selected-ID request.
- `load` and `runMutation` short-circuit already-aborted callers before touching active work or visible state, install lifecycle ownership before async execution, and settle ownership in `finally`. Added tests cover read recovery, create/replace/delete recovery, API-call suppression, and synchronous mutation failure followed by retry.

No task status, unrelated Phase 07.01 implementation, generated contract, preparation evidence, or dependency review was changed. The only requested write is this review artifact.
