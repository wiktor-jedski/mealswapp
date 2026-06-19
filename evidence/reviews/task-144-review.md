# Task 144 Review — Phase 05 Search Modes and Substitution Inputs

## Decision

**Recommended status: PASSED**

## Task Reviewed

| Field | Value |
| --- | --- |
| Task ID | 144 |
| Component | Phase 05 Search Modes and Substitution Inputs |
| Static Aspect | DESIGN-001: SearchView |
| Status (in task list) | PREPARED |
| Retries | 0 |
| Depends On | 140, 143 |
| Testing Coverage Exceptions | None |
| Description | Phase 05: replace the shell placeholders with Catalog, Substitution, and Daily Diet Alternative controls, position mode controls above the search bar and macro controls, and support quantity-bearing Substitution Input accumulation and removal. |

## Dependency Check

| Dep ID | Required | Found | Notes |
| --- | --- | --- | --- |
| 140 | PASSED | PASSED | `02_TASK_LIST.md:147` shows Task 140 PASSED. `frontend/src/lib/stores/search.ts` exports `searchStore` (`:74`), `setQuery` (`:96`), `setMode` (`:81`), `addSubstitutionInput` (`:160`), `removeSubstitutionInput` (`:173`), `updateSubstitutionInput` (`:188`), `setDailyDietId` (`:208`), `buildSearchRequest` (`:269`), `resetSearch` (`:260`). All consumed by Task 144 components and `SubstitutionRequest.test.ts`. Reused, not duplicated. |
| 143 | PASSED | PASSED | `02_TASK_LIST.md:150` shows Task 143 PASSED. `frontend/src/lib/stores/preferences.ts` and `SettingsPanel.svelte` exist; `SearchShell.svelte:75` composes `<SettingsPanel />` as the macro-controls layer required by C1's visual order. No additional coupling required. |

## Verification Checklist

| ID | Criterion | Result | Evidence |
| --- | --- | --- | --- |
| C1 | Tests verify documented visual order (mode controls above search bar above macro controls). | PASS | `SearchShell.test.ts:19` "visual order: mode controls above search bar above macro controls above results area" asserts `indexOf("<SearchModes />") < indexOf('id="search"') < indexOf("<SettingsPanel />") < indexOf('aria-label="Search results"')`. Implementation `SearchShell.svelte:56-80` renders `<SearchModes />` → search `<form>` → conditional mode-specific controls → `<SettingsPanel />` → results placeholder, in that document order. |
| C2 | Tests verify mode-specific controls (Catalog/Substitution/Daily Diet each show appropriate controls). | PASS | `SearchShell.test.ts:38` "mode-specific controls render conditionally based on searchStore.mode" asserts source contains `$searchStore.mode === "substitution"`, `$searchStore.mode === "daily_diet_alternative"`, `<SubstitutionInputs />`, `<DailyDietControls />` (`SearchShell.svelte:69-73`). `SearchModes.test.ts:21` asserts all three mode options `catalog`/`substitution`/`daily_diet_alternative` with ids `search-mode-catalog`/`search-mode-substitution`/`search-mode-daily-diet` and visible labels. Catalog mode correctly shows no extra mode-specific panel (uses the shared search bar). |
| C3 | Tests verify selected autocomplete + Enter adds one Substitution Input (autocomplete is Task 145; verify Enter-to-add mechanism exists). | PASS | `SubstitutionInputs.test.ts:28` "Enter on the foodObjectId input and Add button both add one Substitution Input via addSubstitutionInput" asserts `addSubstitutionInput`, `addInput`, `on:keydown`, `event.key === "Enter"`, `on:click={addInput}`. Implementation `SubstitutionInputs.svelte:52-57` `onFoodObjectIdKeydown` calls `addInput()` on Enter with `preventDefault`; `:44` calls `addSubstitutionInput(...)` once per add. Autocomplete itself is correctly deferred to Task 145 (`:17,51,82` comments mark the raw text input as a stand-in). |
| C4 | Tests verify duplicate/removal behavior is deterministic. | PASS | `SubstitutionInputs.test.ts:37` "duplicate foodObjectId is rejected with a message before reaching the store" asserts `some((existing) => existing.foodObjectId === trimmedId)` and `"Duplicate"` (`SubstitutionInputs.svelte:40-43` rejects with `draftMessage` and returns before touching the store). `SubstitutionInputs.test.ts:50` "removal calls removeSubstitutionInput and row edits call updateSubstitutionInput" asserts `removeSubstitutionInput(input.foodObjectId)` (`:147`) and update bindings (`:131,138`). Store helpers `mergeSubstitutionInput` (`search.ts:327`) and `removeSubstitutionInput` (`search.ts:173`) are deterministic by `foodObjectId`. |
| C5 | Tests verify quantities and canonical units (g/ml/oz/fl_oz) reach SearchRequest. | PASS | `SubstitutionInputs.test.ts:20` asserts the source declares all four canonical `SubstitutionUnit` options `g`/`ml`/`oz`/`fl_oz` (`SubstitutionInputs.svelte:10-15`). `SubstitutionRequest.test.ts:23` "each canonical SubstitutionUnit reaches SearchRequest.substitutionInputs via buildSearchRequest" round-trips all four units through `addSubstitutionInput` → `buildSearchRequest` and asserts the exact `SearchRequest` shape. `SubstitutionRequest.test.ts:49` "Substitution Input quantities are preserved" asserts quantity 250 survives. Implementation `search.ts:280-281` attaches `state.substitutionInputs` to `request.substitutionInputs` in substitution mode. |
| C6 | Tests verify Daily Diet Alternative requests expose Phase 04 structured rejection without creating Phase 07 job behavior. | PASS | `DailyDietControls.test.ts:14` asserts generated `SearchRejection` import (no handwritten duplicate). `:29` asserts the rejection surface exposes `rejection?.code`, `rejection?.message`, `rejection?.field` via `role="alert"` `aria-label="Search rejection"` (`DailyDietControls.svelte:37-49`). `:40` "does not create Phase 07 job or worker behavior" asserts absence of `createJob`/`startWorker`/`queueJob`/`optimizeDiet`. `SubstitutionRequest.test.ts:61` "daily_diet_alternative mode exposes dailyDietId on SearchRequest without substitutionInputs" verifies the request shape. No job/worker code present in any Task 144 file (grep confirmed). |

## Commands Run

| Command | Working dir | Exit | Result |
| --- | --- | --- | --- |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | `frontend/` | 0 | 137 pass, 0 fail, 392 expect() calls across 14 files. Task 144 files: 23 pass (SearchShell 4, SearchModes 4, SubstitutionInputs 7, DailyDietControls 5, SubstitutionRequest 3). |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | `frontend/` | 0 | Vite build OK: 119 modules transformed, `dist/index.html` 0.51 kB, CSS 11.62 kB, JS 53.73 kB; built in 1.14s. All Task 144 components compile (wired via `SearchShell.svelte`). |
| `git -C /home/wiktor/Work/glm status` | repo root | 0 | Task 144 deliverables present: `SearchShell.svelte` (modified), `SearchModes.svelte`/`SubstitutionInputs.svelte`/`DailyDietControls.svelte` (new), and 5 new test files. Other untracked files belong to Tasks 140-143 (PASSED) or 148 (PREPARED); not Task 144 concerns. No `dist/`, `node_modules/`, `.bun-tmp/`, `.bun-install/` staged (gitignored). |
| `python3 /home/wiktor/Work/glm/scripts/validate-task-list.py` | repo root | 0 | "Task-list validation passed: 154 sequential tasks with ordered dependencies." |
| `python3 /home/wiktor/Work/glm/scripts/validate-traceability.py` | repo root | 0 | "Traceability validation passed." |

## Files Inspected

| Path | Reason |
| --- | --- |
| `frontend/src/lib/components/SearchShell.svelte` (83 lines, modified) | Shell composition: imports `SearchModes`, `SubstitutionInputs`, `DailyDietControls`, `SettingsPanel` (`:5-8`); renders `<SearchModes />` (`:56`) above the search `<form>` (`:58-67`) above the conditional mode-specific controls (`:69-73`) above `<SettingsPanel />` (`:75`) above the results placeholder (`:78-80`). Mode-aware placeholder (`:13-22`). Traceability comments at `:10,29,32,77`. Sidebar (`:33-35`) and ResultsGrid (`:78-80`) are explicit placeholders for Tasks 147/146 — not implemented. |
| `frontend/src/lib/components/SearchModes.svelte` (32 lines, new) | Mode toggle `<nav aria-label="Search modes">` with three `<button>`s bound to `setMode(option.value)`, `aria-pressed={$searchStore.mode === option.value}`, `class:border-[var(--color-primary)]` active state, and `focus:outline-none focus:ring-2` focus state. Uses generated `SearchMode` (`:3`). Traceability comments at `:5,18`. No later-task surface. |
| `frontend/src/lib/components/SubstitutionInputs.svelte` (155 lines, new) | Substitution Input composition: draft row (foodObjectId text input + quantity number input + unit `<select>` with g/ml/oz/fl_oz + Add button) and rendered input rows with quantity/unit editing and Remove button. `addInput` (`:30-49`) validates non-empty id, positive finite quantity, and duplicate id before calling `addSubstitutionInput`. `onFoodObjectIdKeydown` (`:52-57`) adds on Enter. Row updates call `updateSubstitutionInput` (`:60-71`). Uses generated `SubstitutionUnit` (`:3`). sr-only labels and visible focus states on all controls. Traceability comments at `:5,17,22,25,51,59,74`. Autocomplete deferred to Task 145 (placeholder text + comments). |
| `frontend/src/lib/components/DailyDietControls.svelte` (50 lines, new) | Daily Diet Alternative controls: UUID-pattern `<input>` bound to `setDailyDietId` (`:18-21,27-35`) and a `SearchRejection` display surface (`:37-49`) exposing `code`/`message`/`field` via `role="alert"` `aria-label="Search rejection"`. `export let rejection: SearchRejection \| null = null` is the typed prop (Task 151 wires the actual 422 envelope); falls back to `searchStore.error` so the UI shape exists without Phase 07 job behavior. Uses generated `SearchRejection` (`:3`). Traceability comments at `:5,7,24`. |
| `frontend/src/lib/components/SearchShell.test.ts` (48 lines, new) | 4 static-source tests for C1/C2: visual order via `indexOf` position comparisons; search bar enabled and bound to `setQuery`; conditional mode-specific controls; DESIGN-001 traceability comment. Header documents the no-DOM-lib fallback and that `vite build` validates the Svelte source. |
| `frontend/src/lib/components/SearchModes.test.ts` (52 lines, new) | 4 static-source tests for C2: three mode options declared with ids/values/labels; `setMode` binding + `aria-pressed` + `$searchStore.mode`; labelled `nav` landmark + traceability; Tailwind focus states. |
| `frontend/src/lib/components/SubstitutionInputs.test.ts` (70 lines, new) | 7 static-source tests for C3/C4/C5: canonical `SubstitutionUnit` set; Enter + Add button call `addSubstitutionInput`; duplicate rejection; positive finite quantity validation; removal + update bindings; labelled section landmark + traceability; focus states (≥4 occurrences each). |
| `frontend/src/lib/components/DailyDietControls.test.ts` (51 lines, new) | 5 static-source tests for C6: generated `SearchRejection` import (no handwritten duplicate); UUID-pattern input bound to `setDailyDietId`; rejection surface exposes code/message/field; absence of `createJob`/`startWorker`/`queueJob`/`optimizeDiet` (Phase 07 exclusion); labelled section landmark + traceability. |
| `frontend/src/lib/components/SubstitutionRequest.test.ts` (71 lines, new) | 3 store-level tests for C5/C6 exercising the Task 140 search store: all four canonical units round-trip to `SearchRequest.substitutionInputs` via `buildSearchRequest`; quantity preservation; `daily_diet_alternative` mode exposes `dailyDietId` without `substitutionInputs`. Uses `afterEach(resetSearch)` for isolation. |
| `frontend/src/lib/stores/search.ts` (356 lines, Task 140) | Dependency: confirms `setMode` (`:81`) clears incompatible state and resets page; `addSubstitutionInput` (`:160`) uses deterministic `mergeSubstitutionInput` (`:327`); `removeSubstitutionInput` (`:173`) filters by `foodObjectId`; `buildSearchRequest` (`:269`) attaches `substitutionInputs` (substitution mode) or `dailyDietId` (daily_diet_alternative mode). No handwritten API type duplicates — all types imported from `../api/generated`. |
| `frontend/src/lib/api/generated.ts` | Confirms generated types: `SearchMode` (`:236`), `SubstitutionUnit` (`:257`), `SubstitutionInput` (`:261`), `SearchRequest` (`:269`), `SearchRejection` (`:312`). All Task 144 components import these — no handwritten duplicates. |
| `docs/design/DESIGN-001.md` | Confirms `SearchView` static aspect (`:7`) owns "catalog query input, Substitution Input composition, filter composition, debounce timing, and result loading orchestration"; `SidebarComponent` (`:8`) owns navigation between Catalog/Substitution/Daily Diet Alternative search. Task description's visual order (mode controls above search bar above macro controls) is reflected in the implementation and verified by `SearchShell.test.ts`. |
| `docs/implementation/02_TASK_LIST.md` | Task 144 row (`:151`) is `PREPARED`; Task 140 (`:147`) PASSED; Task 143 (`:150`) PASSED. Tasks 145/146/147 remain OPEN — confirming Task 144 must not implement autocomplete, results grid, or sidebar. |

## Coverage / Exception Review

- **Testing Coverage Exceptions:** None declared by the task.
- **Traceability comments:** 42 `Implements DESIGN-001` citations across the 9 Task 144 files (components + tests). Meets AGENTS.md requirement.
- **TSDoc:** Every exported prop, function, and constant in the new components has TSDoc explaining purpose and tying it to DESIGN-001 SearchView. Meets AGENTS.md requirement.
- **Generated-type discipline:** All API types (`SearchMode`, `SubstitutionUnit`, `SubstitutionInput`, `SearchRequest`, `SearchRejection`) are imported from `../api/generated`. No handwritten API type duplicates. Component-local types are limited to UI draft state and option arrays.
- **Scope discipline:** No Task 145 (autocomplete), 146 (results grid), or 147 (sidebar) implementation is introduced. `SearchShell.svelte:32,77` and `SubstitutionInputs.svelte:17,51,82` carry explicit placeholder comments deferring to those tasks. grep for `createJob`/`startWorker`/`queueJob`/`optimizeDiet` returned no matches — no Phase 07 job behavior.
- **Static-source test approach:** Consistent with Tasks 140-143, component tests use static-source assertions because no DOM library (jsdom/happy-dom) is installed and Bun's isolated cache breaks `svelte/server` rendering. `SubstitutionRequest.test.ts` exercises the real store (Task 140 dependency) at the contract boundary, proving quantities/units reach `SearchRequest`. `vite build` compiles all components (119 modules, exit 0), validating the Svelte source. Per review rules, source inspection + build compilation + store-level round-trip is acceptable.
- **Simplicity:** Each component is a minimal declarative Svelte component driven by the typed search store. No premature abstraction. WCAG AA: visible focus states (`focus:ring-2`), sr-only labels with `for`/`id` associations, `aria-pressed` for toggle buttons, `role="alert"`/`role="status"` for rejection/draft messages, `aria-label` on landmarks.
- **No generated artifacts staged:** `frontend/dist/`, `node_modules/`, `.bun-tmp/`, `.bun-install/`, `tsconfig.tsbuildinfo` are gitignored.

## Notes / Risks (not rejection reasons)

- **Daily Diet rejection prop not wired to a live 422 response:** `DailyDietControls.svelte:12` declares `export let rejection: SearchRejection | null = null` and falls back to `searchStore.error` for the message. The actual `SearchRejectionEnvelope` wiring from the 422 response is Task 151's responsibility; the component's TSDoc (`:7-11`) states this explicitly. The rejection *surface* (code/message/field via `role="alert"`) is in place and verified, satisfying C6's "expose the Phase 04 structured rejection" without creating Phase 07 behavior. The wiring gap is expected for this phase.

## Failure Details

None. All six verification criteria (C1-C6) are satisfied with passing test evidence (23 Task-144 tests across 5 files), implementation inspection, build success, and aggregate validators (task-list and traceability) passing. No later-task implementation leaked into Task 144; no handwritten API type duplicates; traceability comments present throughout.

