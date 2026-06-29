# Task 151 Review — Phase 05 Search Workflow Integration

## Task row
| 151 | Phase 05 Search Workflow Integration | DESIGN-001: SearchView | PREPARED | 0 | Phase 05: compose stores, API client, controls, autocomplete, results, sidebar, cache, offline status, and theming into the production search shell. | 142,143,144,145,146,147,148,149,150 | None | Integration tests with a controlled API cover initial Catalog search, debounced autocomplete, Substitution Input search, Daily Diet Alternative rejection, filters, pagination, cache reuse, history/favorites, offline cached display, timeout retry, empty state, and theme restoration without handwritten contract fixtures drifting from generated types. |

## Preconditions
- Task 151 status: PREPARED (confirmed in `docs/implementation/02_TASK_LIST.md:158`).
- Dependencies 142–150: all PASSED (confirmed in `docs/implementation/02_TASK_LIST.md:149-157`).
- Task 152 (a11y gate): OPEN. Task 153 (aggregate gate): OPEN. Task 154 (UAT doc): OPEN. No `docs/implementation/implemented/05_PHASE_UAT.md` exists — Task 151 does not implement 154.

## Checklist (Verification Criteria)
- PASS — C1 Initial Catalog search: `search-workflow.spec.ts:110` "initial Catalog search renders ranked results after typing a query" fills the search bar and asserts 5 ranked `[data-result-card]`s.
- PASS — C2 Debounced autocomplete: `search-workflow.spec.ts:120` "autocomplete shows ranked suggestions after the 150ms debounce" asserts listbox visibility and server-ranked option order (Apple, Applesauce).
- PASS — C3 Substitution Input search: `search-workflow.spec.ts:132` switches to Substitution mode, adds an input via `#substitution-food-object-id` + `#substitution-quantity` + Add button, fills the search bar, and asserts `seenRequestBody.mode === "substitution"` with the food object id and quantity.
- PASS — C4 Daily Diet Alternative rejection (422): `search-workflow.spec.ts:155` stubs `/api/v1/search` with a 422 `SearchRejectionEnvelope`, switches to Daily Diet Alternative mode, fills `#daily-diet-id`, and asserts `[data-rejection-message]` contains "No daily diet alternative". `SearchResults.svelte:53-59` derives the rejection from `SearchClientError` status 422 and lifts it to `DailyDietControls` via `onRejection`.
- PASS — C5 Filters: `search-workflow.spec.ts:174` fills `#filter-id`, clicks "Add filter", fills the search bar, and asserts the active-filter chip is visible and `seenRequestBody.filters[0]` has `filterId` + `kind`.
- PASS — C6 Pagination: `search-workflow.spec.ts:195` asserts `Page 1 of 2`, clicks `[data-results-next]`, asserts `Page 2 of 2`, and `lastPage === 2`.
- PASS — C7 Cache reuse: `search-workflow.spec.ts:212` fills "apple", asserts `appleRequestCount === 1`, clears and re-enters "apple", asserts the same count and 4 cards remain visible.
- PASS — C8 History/favorites: `search-workflow.spec.ts:234` stubs `/api/v1/profile`, `/api/v1/search-history`, `/api/v1/saved-items?kind=favorite`, opens the mobile sidebar toggle when needed, and asserts `[data-sidebar-history-entry='hist-1']` and `[data-sidebar-favorite='food-apple']` are visible.
- PASS — C9 Offline cached display: `search-workflow.spec.ts:251` fills "apple", asserts 6 cards, goes offline via `page.context().setOffline(true)`, asserts `[data-offline-banner]` visible and 6 cards remain visible.
- PASS — C10 Timeout retry: `search-workflow.spec.ts:266` stubs search with an 11s delay (exceeds the 10s `SEARCH_TIMEOUT_MS`), asserts `[data-results-error]` visible within 15s, then re-routes with a fast stub, fills "banana", and asserts 3 cards render.
- PASS — C11 Empty state: `search-workflow.spec.ts:287` stubs a zero-item envelope and asserts `[data-results-empty]` has text "No results found.".
- PASS — C12 Theme restoration: `search-workflow.spec.ts:297` emulates light colorScheme, selects "dark", asserts `html[data-theme="dark"]`, reloads, and asserts the attribute and select value persist.
- PASS — C13 No handwritten contract fixtures drifting from generated types: `search-workflow.spec.ts:2-11` imports `AutocompleteEnvelope`, `ProfileEnvelope`, `SavedItemsEnvelope`, `SearchHistoryEnvelope`, `SearchRequest`, `SearchResponse`, `SearchResponseEnvelope`, `SearchRejectionEnvelope` as `type` from `../src/lib/api/generated`. All eight types exist in `frontend/src/lib/api/generated.ts` (lines 144, 174, 193, 269, 341, 353, 357, 383). No `interface`/`type` duplicates of API contract types exist anywhere under `frontend/src` (grep confined to generated.ts). The `foodObject` helper is typed as `SearchResponse["items"][number]`, so any drift fails the build.

## Implementation review
- `frontend/src/App.svelte` (36 lines): constructs `QueryClient` (retry:false, refetchOnWindowFocus:false), wraps `<SearchShell />` in `<QueryClientProvider>`, runs `initPreferences`/`initTheme`/`initOffline` in a `$effect` with `cleanupTheme`/`cleanupOffline` on destroy, and registers the service-worker seam gated by `import.meta.env.PROD`. Traceability comments cite DESIGN-001 SearchView and DESIGN-016 ThemeProvider.
- `frontend/src/lib/components/SearchShell.svelte` (174 lines): composes `<SidebarComponent />`, `<SearchModes />`, `<AutocompleteDropdown>` (bound to `searchStore.query`/`setQuery`/`onAutocompleteSelect`), `<SubstitutionInputs />`/`<DailyDietControls {rejection}>` conditionally on mode, a filter composer (id/kind/include + Add filter + active filter chips with Remove), `<SettingsPanel />`, `<SearchResults onRejection>`, and `<OfflineBanner />`. Documented visual order matches the spec. Traceability comments cite DESIGN-001 SearchView, SidebarComponent, SettingsPanel, and DESIGN-016 LayoutGrid (12-column desktop, single-column below 640px).
- `frontend/src/lib/components/SearchResults.svelte` (95 lines): new TanStack Query host. Builds `createSearchQueryOptions(searchStore, localCache)`, bridges the derived store + search store + offline store to runes, drives `createQuery(() => currentOptions)`, derives `rejection` from a 422 `SearchClientError`, lifts it to the parent via `onRejection`, reflects offline cached state into the OfflineBanner store, and wires `<ResultsGrid>` with `query.data` items/metadata/scores, `query.isFetching` loading, derived error message, `state.page`, and `setPage`. The `rejection` derivation omits `field` (documented inline as a follow-up); the test only asserts the message, so C4 passes. Traceability comments cite DESIGN-001 SearchView.
- `frontend/src/lib/components/SearchShell.test.ts` (65 lines): static-source assertions verify all eight composed components are present, the documented visual order (modes → autocomplete → settings → results → offline), the search-bar binding to `setQuery`, conditional mode-specific controls, the `onRejection`/`{rejection}` wiring, and the DESIGN-001 traceability comment.
- `frontend/tests/search-workflow.spec.ts` (308 lines): new file with the 12 integration scenarios mapped to C1–C12 above. All fixtures are typed from `generated.ts` (C13).
- `frontend/tests/autocomplete.spec.ts`, `results.spec.ts`, `smoke.spec.ts`, `responsive.spec.ts`: un-skipped / updated to use the wired shell selectors (`getByLabel("Food search")`, `[data-result-card]`, `[data-results-*]`, `getByRole("navigation", { name: "Search modes" })`). The smoke axe check is a pre-existing smoke-level scan, not the Task 152 a11y gate (no WCAG 4.5:1, keyboard-only workflow, focus-visible, or accessible-names verification). `autocomplete.spec.ts:67` explicitly references the Task 152 a11y gate as a follow-up, confirming 152 is not implemented here.
- No later-task implementation: no `docs/implementation/implemented/05_PHASE_UAT.md` (154), no `scripts/check.py` modifications (153), no full WCAG/keyboard-workflow/focus-visible a11y gate (152). `registerServiceWorker` in `App.svelte` is a registration seam (`enabled: import.meta.env.PROD`), not Phase 09 service-worker interception.
- Code smells: minor — `SearchResults.svelte` subscribes to writable stores inside `$effect` blocks (valid Svelte 5 pattern returning the unsubscribe function); the `rejection.field` omission is documented inline as a follow-up rather than a silent defect. Neither blocks the verification criteria.

## Commands
- `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` (frontend/) -> 195 pass, 0 fail, 656 expect() calls across 18 files.
- `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` (frontend/) -> vite v7.3.3, 184 modules transformed, built in 7.65s, dist/assets/index-CgV1Hd-5.js 122.55 kB.
- `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e` (frontend/) -> 64 passed (1.0m); all 12 search-workflow scenarios pass on both desktop-chromium and mobile-chromium projects; the 11s-timeout scenario passes at ~12.6s desktop / 13.0s mobile.
- `git -C /home/wiktor/Work/glm status` -> On branch multistep-phase-05-glm; modified files include App.svelte, SearchShell.svelte, theme.ts, app.css, generated.ts, package.json, package.json-trace.md, 02_TASK_LIST.md, 04_OPEN.md; untracked evidence/, frontend/tests/, SearchResults.svelte, SearchShell.test.ts, and the Phase 05 component/store/API client files from deps 142–150.
- `python3 /home/wiktor/Work/glm/scripts/validate-task-list.py` -> "Task-list validation passed: 154 sequential tasks with ordered dependencies."
- `python3 /home/wiktor/Work/glm/scripts/validate-traceability.py` -> "Traceability validation passed."

## Files inspected
- `docs/implementation/02_TASK_LIST.md:149-161` — confirm Task 151 PREPARED, deps 142–150 PASSED, tasks 152/153/154 OPEN.
- `frontend/src/App.svelte` — bootstrap: QueryClientProvider, init/cleanup lifecycle, service-worker seam.
- `frontend/src/lib/components/SearchShell.svelte` — composition of all eight components + filter composer + visual order.
- `frontend/src/lib/components/SearchResults.svelte` — TanStack Query host, ResultsGrid wiring, 422 rejection derivation.
- `frontend/src/lib/components/SearchShell.test.ts` — static-source composition/order/wiring assertions.
- `frontend/tests/search-workflow.spec.ts` — 12 integration scenarios (C1–C12) with generated-type fixtures (C13).
- `frontend/tests/autocomplete.spec.ts`, `results.spec.ts`, `smoke.spec.ts`, `responsive.spec.ts` — un-skipped/updated selectors.
- `frontend/src/lib/api/search-client.ts` — confirms 10s timeout, credentialed requests, local-cache read/write, 422 mapping.
- `frontend/src/lib/api/generated.ts:144,174,193,269,296,314,333,341,353,357,365,376,383` — confirms all imported types exist and no handwritten duplicates.
- `frontend/src/lib/stores/search.ts`, `preferences.ts`, `offline.ts` — confirms composed stores.
- `frontend/src/lib/components/SubstitutionInputs.svelte`, `DailyDietControls.svelte`, `SearchModes.svelte`, `AutocompleteDropdown.svelte`, `ResultsGrid.svelte` — confirms selectors and IDs used by the integration tests.
- `frontend/src/lib/cache/service-worker.ts` — confirms registration seam, no Phase 09 policy.
- `frontend/playwright.config.ts`, `bunfig.toml`, `package.json` — confirms e2e config and test scripts.

## Decision reason
All 13 verification criteria are satisfied. The production search shell composes every Phase 05 component (SidebarComponent, SearchModes, AutocompleteDropdown, SubstitutionInputs/DailyDietControls, SettingsPanel, SearchResults+ResultsGrid, OfflineBanner) with TanStack Query context, theme/preferences/offline lifecycle, and a filter composer. The 12 integration scenarios in `frontend/tests/search-workflow.spec.ts` cover C1–C12 and run green on both desktop and mobile Chromium; all fixtures are typed from `generated.ts` with zero handwritten contract duplicates (C13). Traceability comments cite DESIGN-001 SearchView (and DESIGN-016 where relevant) across App.svelte, SearchShell.svelte, and SearchResults.svelte. No later-task work is present: 152 a11y gate is OPEN (smoke axe check is smoke-level only), 153 aggregate gate is OPEN (scripts/check.py untouched), 154 UAT is OPEN (no 05_PHASE_UAT.md). All six verification commands pass: 195 unit tests, 64 e2e tests, vite build, task-list validation, and traceability validation.

## Repair instructions
N/A — recommended status PASSED.
