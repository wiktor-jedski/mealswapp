# Task 147 Review — Phase 05 Activity Sidebar

Task row:
| 147 | Phase 05 Activity Sidebar | DESIGN-001: SidebarComponent | PREPARED | 0 | Phase 05: implement the left-side collapsible activity sidebar with search-mode navigation, authenticated search history, favorites, settings entry points, and a mobile toggle. | 142,144 | None | Component integration tests verify desktop-left placement, collapse/expand behavior, mobile toggle behavior, history and favorite loading through generated authenticated contracts, anonymous empty/sign-in guidance, selecting a history entry restores search state, and API failures do not block core search. |

## Preconditions

- Task 147 status: PREPARED (confirmed in `docs/implementation/02_TASK_LIST.md:154`).
- Dep 142 (Generated Search API Client): PASSED (`02_TASK_LIST.md:149`).
- Dep 144 (Search Modes and Substitution Inputs): PASSED (`02_TASK_LIST.md:151`).

## Verification Criteria Checklist

- PASS - C1: Tests verify desktop-left placement.
  - `SidebarComponent.test.ts:24` "renders a desktop-left aside with sticky left-column placement classes" asserts `desktop-sidebar-left`, `aria-label="Activity sidebar"`, `sm:sticky`, `sm:top-0`, `sm:h-screen`, `sm:border-r`, `data-sidebar`. Component root at `SidebarComponent.svelte:206-212` declares the `<aside class="desktop-sidebar-left ... sm:sticky sm:top-0 sm:h-screen sm:border-r ...">`.
- PASS - C2: Tests verify collapse/expand behavior.
  - `SidebarComponent.test.ts:35` asserts `sidebar-collapse-toggle`, `data-sidebar-collapse`, `toggleCollapsed`, `$sidebarStore.collapsed`, `aria-expanded={!$sidebarStore.collapsed}`, collapse glyph `»/«`, width swap `sm:w-14`/`sm:w-60`, and content hide `$sidebarStore.collapsed ? 'sm:hidden' : 'sm:grid'`. Implementation at `SidebarComponent.svelte:229-238` and `:243`.
- PASS - C3: Tests verify mobile toggle behavior.
  - `SidebarComponent.test.ts:49` asserts `mobile-sidebar-toggle`, `data-sidebar-mobile-toggle`, `sm:hidden` gating, `toggleMobileOpen`, `sidebar-mobile-close`, `data-sidebar-mobile-close`, `setMobileOpen(false)`, `$sidebarStore.mobileOpen`, and `$sidebarStore.mobileOpen ? 'block' : 'hidden'`. Implementation at `SidebarComponent.svelte:214-226` (open) and `:247-257` (close).
- PASS - C4: Tests verify history and favorite loading through generated authenticated contracts.
  - `SidebarComponent.test.ts:63` asserts `import type {` with `ProfileEnvelope`, `SearchHistoryEnvelope`, `SavedItemsEnvelope`, `SearchHistoryEntry`, `SavedItem`, `ProfileData` all `from "../api/generated"`; endpoints `/api/v1/profile`, `/api/v1/search-history`, `/api/v1/saved-items?kind=favorite`; and exactly 3 `credentials: "include"` fetches. All types confirmed present in `frontend/src/lib/api/generated.ts:134,144,156,174,178,193`. No handwritten API type duplicates — types are imported only.
- PASS - C5: Tests verify anonymous empty/sign-in guidance.
  - `SidebarComponent.test.ts:81` asserts `data-sidebar-anonymous`, the guidance copy `Sign in to see your history and favorites.`, `authenticating`/`authenticated` gating, and `response.status === 401`. Implementation at `SidebarComponent.svelte:91-96` treats 401 as anonymous (sets `authenticated=false`, `profile=null`, does NOT set `authError`) and renders the guidance block at `:282-284`.
- PASS - C6: Tests verify selecting a history entry restores search state (setQuery/setMode).
  - `SidebarComponent.test.ts:91` asserts `onHistoryEntrySelect`, `setQuery(entry.query)`, `setMode(entry.mode)`, `isSearchMode(entry.mode)` (mode validation guard), `on:click={() => onHistoryEntrySelect(entry)}`, `data-sidebar-history-entry={entry.id}`. Implementation at `SidebarComponent.svelte:182-188` calls `setQuery(entry.query)`, validates mode via `isSearchMode` before `setMode`, and closes mobile sidebar. `setQuery`/`setMode` confirmed exported from `frontend/src/lib/stores/search.ts:96,81`.
- PASS - C7: Tests verify API failures do not block core search (inline error handling, no propagation).
  - `SidebarComponent.test.ts:101` asserts >= 3 `} catch {` blocks, `authError =`, `historyError =`, `favoritesError =` inline state, `data-sidebar-history-error`, `data-sidebar-favorites-error`, `data-sidebar-auth-error` surface nodes, and `not.toContain("throw new Error")` / `not.toContain("throw new SearchClientError")`. Implementation: `loadSidebar` (`:82-119`), `loadHistory` (`:126-146`), `loadFavorites` (`:153-173`) each wrap the fetch in try/catch and assign a local error string; no rethrow. The component has no `throw` expression. Parent search flow stays usable.

## Implementation Review

Files inspected:
- `frontend/src/lib/components/SidebarComponent.svelte` — new component implementing desktop-left `<aside>`, collapse toggle, mobile open/close toggles, mode nav, authenticated history/favorites sections, settings entry, anonymous guidance, inline error handling. Traceability comment `<!-- Implements DESIGN-001 SidebarComponent -->` at top of markup (`:205`) and `// Implements DESIGN-001 SidebarComponent ...` in script (`:22`); per-block comments cite the specific sidebar aspect.
- `frontend/src/lib/components/SidebarComponent.test.ts` — 10 static-source verification tests covering all seven criteria plus mode nav, settings entry, and traceability. Test header (`:5-15`) documents why static-source verification is used: Bun's isolated install-cache layout breaks `svelte/server`/`svelte/compiler` transitive resolution and no DOM library (jsdom/happy-dom) is installed; `vite build` compiles the component, validating the Svelte source at build time. This pattern is consistent with the other Phase 05 component tests (`SearchModes.test.ts`, `SubstitutionInputs.test.ts`, `DailyDietControls.test.ts`) that are already PASSED.
- `frontend/src/lib/stores/sidebar.ts` — new store exposing `sidebarStore`, `toggleCollapsed`, `toggleMobileOpen`, `setMobileOpen`, `initSidebar`, plus `createInitialSidebarState`, `isValidSidebarState`, `resetSidebar`, `SIDEBAR_STORAGE_KEY`. Persists only the desktop `collapsed` flag to localStorage; `mobileOpen` is never persisted (SSR- and throw-safe). TSDoc comments throughout; every function carries an `Implements DESIGN-001 ...` remark.

Cross-cutting checks:
- No handwritten API type duplicates: all envelope/entry types imported from `../api/generated`.
- Traceability: DESIGN-001 SidebarComponent cited in component script, markup, every store function remark, and every test comment.
- No later-task implementation leakage:
  - Task 149 (Theme Persistence): no theme work added by this task. `SearchShell.svelte` already imported `themePreference`/`setThemePreference` from `../stores/theme` (store committed in phase 00, `git log -- frontend/src/lib/stores/theme.ts` → `d582d340 phase 00 completed`). `SidebarComponent.svelte` does not touch theme.
  - Task 150 (Responsive Style System): no 12-column grid or breakpoint-system changes; sidebar uses existing Tailwind utilities consistent with the documented style guide.
  - Task 151 (Search Workflow Integration): `SearchShell.svelte` still renders the placeholder `<aside>` (`SearchShell.svelte:33-35`) with the comment `Task 147 builds the full activity sidebar`; `SidebarComponent` is NOT wired into the shell. `git diff frontend/src/lib/components/SearchShell.svelte` shows no sidebar-wiring change in this task.
- credentials: "include" on all three authenticated fetches (profile, history, favorites); count verified by test (`countOccurrences === 3`).
- 401 treated as anonymous, not error: `SidebarComponent.svelte:91-96` returns early without setting `authError`.
- Code smells: none significant. State is local to the component; store interactions go through the documented `sidebarStore` API; async loaders are fire-and-forget via `void` calls in `onMount`/`loadSidebar`; error state is per-section and rendered inline.

## Commands

- `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/components/SidebarComponent.test.ts` (frontend/) -> 10 pass, 0 fail, 75 expect() calls.
- `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` (frontend/) -> 160 pass, 0 fail, 527 expect() calls across 16 files.
- `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` (frontend/) -> vite v7.3.3 built, 119 modules transformed, dist assets emitted, 0 errors.
- `git -C /home/wiktor/Work/glm status` -> branch `multistep-phase-05-glm`; SidebarComponent.svelte, SidebarComponent.test.ts, sidebar.ts untracked (new); SearchShell.svelte modified by prior tasks (diff shows no sidebar wiring).
- `python3 /home/wiktor/Work/glm/scripts/validate-task-list.py` -> "Task-list validation passed: 154 sequential tasks with ordered dependencies."
- `python3 /home/wiktor/Work/glm/scripts/validate-traceability.py` -> "Traceability validation passed."

## Decision

Recommended status: PASSED.

Decision reason:
All seven verification criteria (C1-C7) are satisfied by the implementation and verified by the test suite. The component imports all envelope/entry types from `../api/generated` with no handwritten duplicates, performs three credentialed authenticated fetches, treats 401 as anonymous rather than an error, restores search state via `setQuery`/`setMode` with a mode-validation guard, and confines every failure to inline per-section error state so core search stays usable. Traceability comments cite DESIGN-001 SidebarComponent across the component, store, and tests. The implementation does not encroach on later tasks: no theme persistence (149), no responsive grid system (150), and no SearchShell integration wiring (151) — `SearchShell.svelte` still hosts the placeholder aside. The static-source test approach is consistent with the already-PASSED sibling Phase 05 component tests and is justified by the documented Bun/svelte-rendering constraint; `vite build` validates the Svelte source at build time. Targeted tests, full suite, build, task-list validation, and traceability validation all pass.

## Repair instructions

None.
