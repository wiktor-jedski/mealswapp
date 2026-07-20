# Task 229 Preparation — Daily Diet State Coordination and Authoritative Selection

## Scope and worktree safety

- Authoritative task: Phase 07.01 Task 229, `DESIGN-001: SearchView`, status `OPEN`.
- Work is limited to the Daily Diet read/mutation lifecycle, authoritative DTO reconciliation, selected-diet coordination across search/optimization/auth identity, rejected-review repairs F-229-01/F-229-02, deterministic tests, and this evidence file.
- The worktree already contained concurrent Phase 07.01 changes, including Task 228 Daily Diet client/store work. No unrelated path was cleaned, reverted, staged, or rewritten.
- No task status or unrelated task row was changed. Task 229 remains `OPEN` as requested.

## Authoritative sources read

| Source | Relevant contract | SHA-256 |
|---|---|---|
| `docs/implementation/02_TASK_LIST.md` | Task 229 description, dependency 228, complete race/selection acceptance matrix, preserved `OPEN` status | `a44ed4b1ed8bdaebba1510b1b18c5214c43051e77ba307ae1ddab2d1fa3dc6f4` |
| `docs/design/DESIGN-001.md` | `SearchView`, `SearchState`, request construction, Daily Diet Alternative orchestration, authenticated routing | `34d699ae93a8e5465199f3494ed41813c675f1cb3b9c1c6b6e611ba66c6142c7` |
| `docs/architecture/ARCH-001.md` | Web Application Module and `SearchView` ownership boundary | `03fcbae9676ecf278e72a621c7fd9911d0c3dc6ee15eaebcec308d010cc76833` |
| `docs/implementation/preparation/task-228-preparation.md` | inherited strict decoder, retry-stable create intent, identity lifecycle, and residual Task 229 ordering boundary | `2b1c3201303d03f2a15d80b4743a78a3cdf3fc56b4b9dda444c4cb76de5f5352` |
| `docs/implementation/reviews/task-228-review.md` | prior evidence that broader operation ordering and selection coordination belong to Task 229 | `f089287fed28666c6526400f3141973606454acd1a17ce190c5cc30cf8e13539` |

## Implemented symbols and files

| File | Exact symbols / behavior | SHA-256 |
|---|---|---|
| `frontend/src/lib/stores/daily-diet.ts` | `ReadLifecycle`, `MutationLifecycle`, `activeRead`, `activeMutation`, `load`, `create`, `replace`, `select`, `remove`, `clear`, `runMutation`, `finishMutation`, `finishCancelledMutation`, `mutationInProgress`, `waitForSettlement`, `chainAbortSignal`; pre-aborted calls reject before touching active work, ownership/state are installed before synchronous execution can settle, superseded reads abort, reads queue abortably behind mutations, mutations cancel reads and cannot overlap, clear aborts both lifecycles, and commits are lifecycle-owned | `9a321420f52aadbc924870098a194fe6fe2b57c844076bbece0746a83572cf20` |
| `frontend/src/lib/stores/daily-diet.test.ts` | deterministic `deferred`/`abortable` fixtures and 27 controller tests, including every Task 229 race and selection reconciliation case plus pre-aborted read/mutation recovery and synchronous execution failure recovery | `86694403519286d0da6a6cda8839571506167d14fa1afac5c92247b857a1a7d0` |
| `frontend/src/lib/stores/selected-daily-diet.ts` | sole memory-only `selectedDailyDietId` writable used by collection, search, optimization, and identity clearing | `75435238ff8c0a17107ce7b2be601531e3edc636c94a937a7ae995170201ef0d` |
| `frontend/src/lib/stores/search.ts` | Daily Diet Alternative state no longer duplicates `dailyDietId`; `setDailyDietId` writes the authoritative store; `buildSearchRequest` and `searchRequestKey` consume one selected ID only in alternative mode | `32ea31c61bafd59f92cb28013fcd646ef097400cb18d48d5359c346057947778` |
| `frontend/src/lib/stores/search.test.ts` | mode transitions preserve authoritative selection; Catalog/Substitution omit it; alternative requests emit it; state has no duplicate selection field | `84b2663ef9a788a17802e0ed52ba0a772afca2faf87ddd20e7a57c31a4b66911` |
| `frontend/src/lib/stores/search-state.types.ts` | compile-time proof that `DailyDietAlternativeSearchState` cannot carry a duplicate `dailyDietId` field | `b5ad04da63f8d1ce7f33151a674a4a426462dace59cc3bb8f90f13c40cda55c0` |
| `frontend/src/lib/api/search-client.ts` | `buildSearchQueryOptions` accepts the selected ID explicitly and disables Daily Diet Alternative execution until it is non-null; `createSearchQueryOptions` derives from both search state and `selectedDailyDietId`, keeping enablement/request/query-key emission reactive | `25ffbf5ad14c75614478363bed774ae22b8d2e77682c90c5bf35464afe75ce1f` |
| `frontend/src/lib/api/search-client.test.ts` | `createSearchQueryOptions reacts to authoritative selection changes` proves query-key/request invalidation follows selected identity; `Daily Diet Alternative performs no query until authoritative selection exists` proves zero premature fetches and reactive activation with the selected ID | `532bd87a173f6eacfbe038ac128b8d6e15af6582a61ad96bdfcf4664ace41a83` |
| `frontend/src/lib/components/DailyDietControls.svelte` | selector checked state and `OptimizationWorkflow` prop both read `selectedDailyDietId`; no `$dailyDietStore.selectedId` remains | `6e0c9327665d07d8c1f0e03d058254c577c327f9e49ca3a1d24a45ae14296de4` |
| `frontend/src/lib/components/DailyDietControls.test.ts` | source-level component proof of one selected source and mutation-gated optimization | `d2e170cc0e1ac8553c45faffb21f4bbbec078d71e807895c1e73625b7b2a3712` |
| `frontend/src/lib/components/OptimizationWorkflow.svelte` | `canSubmit` requires an authoritative selected DTO and `dailyDietStore.mutation === "idle"`; pending replacement data cannot activate optimization | `620e825cd23e258fee69ccb42899e00c01f2dc7a53df5d5b8e3d9cc3c6f00b33` |
| `frontend/src/lib/components/SearchShell.svelte` | `dailyDietStateUserId` effect clears the controller and selected source on logout/account changes even when Daily Diet mode components are unmounted | `584b0e0dba4ec6a8d38217816daa910b09a2bdefc7f3d0d26cf11adbba5fc6e8` |
| `frontend/src/lib/components/SearchShell.test.ts` | shell-level identity lifecycle wiring assertion | `a8282445a4b7b08571598dd8bb768ede8c5aea20f80e38df0b519619fa3a55a6` |
| `frontend/src/lib/components/SubstitutionRequest.test.ts` | generated request round-trip confirms alternative emits and Substitution omits the authoritative ID | `52eee8b0a6dd2e0f2b3921e7b8f6382f1b62c5fda533329a182193f817665f2c` |

Inherited Task 228 component files were read and retained unchanged by Task 229: `DailyDietCollection.svelte` (`1428689f367cd04f32e562f132c39b79f609fa0ae7fa9fd104b69e9b20d8ca04`) and `DailyDietCollection.test.ts` (`c0869a7ec40af0806231e72bf900d320d60ca39a3742f302fb5e43a48ab6cf65`).

## Lifecycle decisions

- A read is an explicit object with its own internal `AbortController` and promise. Starting a newer read aborts the older read; stale success/failure cannot commit.
- A mutation is an explicit object with kind, controller, and promise. Starting it aborts any read. A load requested during a mutation waits for mutation settlement before reading authoritative server state.
- Only one mutation can own the lifecycle. Duplicate create activation shares the same promise and idempotency intent; a distinct overlapping create/replace/delete fails before API I/O with fixed safe `daily_diet_mutation_in_progress` data.
- External abort signals are chained into internal read/mutation signals and listeners are removed on settlement. Aborted work retains the last authoritative collections and does not project a generic failure.
- An already-aborted caller is rejected before it can cancel an existing owner, install loading/mutation state, or reach API I/O. For accepted operations, lifecycle ownership and visible state are established before synchronous API execution, so synchronous rejection settles the owner and future operations remain usable.
- Logout/account clear aborts read and mutation lifecycles before resetting collection and selection state. Late API completion has no commit authority.
- Create, replace, and delete mutate the collection only after their decoded API operation succeeds while still owning the lifecycle. Replacement has no optimistic entry/name projection and therefore never combines draft entries with old aggregate macros.
- Selection is deliberately independent of mode state and mutation snapshots. Reload retains it only if the ID still exists; selected deletion, disappearance, empty reload, logout, and account changes clear it; unselected deletion preserves it.
- Search state carries mode-specific query/input fields but no selected-diet copy. Catalog/Substitution round trips preserve the authoritative selection while omitting it from their requests; Daily Diet Alternative request/key construction reads the shared selected ID. Optimization receives that same ID and resolves its DTO from server-returned collections.
- Daily Diet Alternative query execution requires both non-empty text and the non-null authoritative selected ID. Selection changes still update the query key and request reactively; no request can execute during the no-selection interval.

## Rejected-review repair evidence

| Finding | Repaired symbols | Deterministic regression evidence |
|---|---|---|
| `F-229-01` missing authoritative selection execution guard | `buildSearchQueryOptions`, `createSearchQueryOptions` | `Daily Diet Alternative performs no query until authoritative selection exists` observes disabled options and zero fetches before selection, then one request containing the newly selected `dailyDietId`. |
| `F-229-02` pre-aborted/synchronous lifecycle wedge | `load`, `runMutation`, `ReadLifecycle`, `MutationLifecycle` | `pre-aborted read settles without API I/O and a later read remains usable`; `pre-aborted mutations settle without API I/O and later mutations remain usable`; `synchronous mutation execution failure settles ownership and permits retry`. |

## Deterministic acceptance matrix

| Task 229 criterion | Exact test evidence |
|---|---|
| load/load and out-of-order completion | `load/load aborts the older read and only the newer server snapshot is installed` |
| load/create | `load/create cancels the stale read and preserves the confirmed create` |
| create/load | `create/load queues the read behind mutation confirmation without losing the write` |
| replace/select/failure | `replace/select/failure keeps newer selection and the last authoritative diet` |
| replace/load | `replace/load serializes the read after authoritative replacement` |
| delete/load | `delete/load serializes the read and cannot resurrect a confirmed deletion` |
| clear/logout and late completion | `clear/logout aborts read and mutation lifecycles and ignores late completions` |
| duplicate activation / overlap | `duplicate activation shares create and rejects overlapping distinct mutations` |
| abort propagation | `caller abort propagates to reads and mutations and leaves authoritative state unchanged` |
| abort while queued behind mutation | `caller abort promptly cancels a read queued behind a mutation` |
| pre-aborted read and recovery | `pre-aborted read settles without API I/O and a later read remains usable` |
| pre-aborted create/replace/delete and recovery | `pre-aborted mutations settle without API I/O and later mutations remain usable` |
| synchronous mutation execution failure and recovery | `synchronous mutation execution failure settles ownership and permits retry` |
| no stale macro projection; authoritative success/failure | `replacement keeps the last authoritative DTO and macros until server success or failure`; `successful replacement installs only the decoded server-derived DTO state`; component mutation-gate test |
| selection before/after mode and Catalog/Substitution round trips | `authoritative selection survives mode round trips and drives one emitted diet id`; `setMode preserves authoritative selection when leaving daily_diet_alternative and resets page` |
| disappearance, selected/unselected deletion, empty reload, identity clear | `reload, deletion, empty state, and identity clear reconcile authoritative selection` |
| emitted search request/query key and no-selection guard | `buildSearchRequest includes dailyDietId in daily_diet_alternative mode`; `createSearchQueryOptions reacts to authoritative selection changes`; `Daily Diet Alternative performs no query until authoritative selection exists`; `daily_diet_alternative mode exposes dailyDietId on SearchRequest without substitutionInputs` |
| UI and optimization consume one ID | `selector and optimization consume one selected Daily Diet source`; `optimization cannot activate while a Daily Diet mutation is pending`; SearchShell identity lifecycle assertion |

## Verification evidence

| Command | Result |
|---|---|
| `cd frontend && bun test src/lib/stores/daily-diet.test.ts src/lib/stores/search.test.ts src/lib/api/search-client.test.ts src/lib/components/SubstitutionRequest.test.ts src/lib/components/DailyDietCollection.test.ts src/lib/components/DailyDietControls.test.ts src/lib/components/OptimizationWorkflow.test.ts` | PASS — 105 tests, 426 expectations |
| `cd frontend && bun test` | PASS — 414 tests, 1,877 expectations |
| `cd frontend && bun run typecheck` | PASS |
| `cd frontend && bun run build` | PASS — Vite production build, 205 modules transformed |
| `cd frontend && bun test --coverage` | PASS — 414 tests; aggregate 94.01% lines; Daily Diet store 99.55%; selected-diet store 100%; search client 100% |
| `cd frontend && bun run check:api-types` | PASS — generated API types current |
| `python3 scripts/validate-traceability.py` | PASS |
| `python3 scripts/validate-task-list.py` | PASS — 237 sequential tasks with ordered dependencies |
| `python3 scripts/verify-frontend.py` | PASS — real Chromium desktop/mobile scenarios and screenshots |
| `git diff --check -- <Task 229 files>` | PASS |
| search for `operation = 0`, `currentOperation`, `optimisticReplace`, `canOptimisticallyReplace`, `$dailyDietStore.selectedId` in scoped production files | PASS — none remain |

## Status and residuals

- No task status was changed. Task 229 remains `OPEN` in `docs/implementation/02_TASK_LIST.md`.
- Repository policy targets 100% phase coverage. The existing phase-level frontend coverage exception remains outside this task; Task 229 raises the Daily Diet store to 99.55% line coverage and gives the selected-diet store and search client 100% coverage.
- Expected 401 resource messages appeared only in anonymous login/register browser scenarios; `scripts/verify-frontend.py` classified the run as passed.
- Backend, OpenAPI, generated API contracts, task rows, and unrelated Phase 07.01 implementation were not changed by Task 229.
