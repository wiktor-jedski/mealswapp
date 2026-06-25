# Phase 05 UAT: Frontend Search Experience

<!-- Implements DESIGN-001 SearchView, SidebarComponent, ResultsGrid, AutocompleteDropdown, OfflineBanner, SettingsPanel, LocalStorageManager; DESIGN-016 ThemeProvider, LayoutGrid, ComponentStyles; DESIGN-017 ErrorMessageMapper. -->

## Scope

Phase 05 covers tasks `138`-`154`. Task `138` was a Phase 04 cleanup task that
unblocked Phase 05 by extending every search-result item with server-derived
classification summaries (`id`, `name`, `kind`), an explicit primary Food
Category, protein/carbohydrate/fat macros with a `100g` or `100ml` basis, and
server-calculated calories. Tasks `139`-`154` build the frontend search
experience against the Phase 04 generated OpenAPI contracts.

The implemented frontend surface composes a `SearchView` shell: typed search
state and request construction, a 20-entry localStorage LRU query cache, a
TanStack Query generated search/autocomplete API client, settings controls
(metric/imperial unit preference), Catalog/Substitution/Daily Diet
Alternative mode controls, 150ms-debounced autocomplete with keyboard focus
navigation, selected FoodObject hydration through `GET /api/v1/food-objects/{id}`,
a paginated results grid with image fallback and similarity presentation, a
collapsible activity sidebar with history/favorites, an offline and stale-data
banner, theme persistence and resolution with an explicit light/dark sidebar
toggle, a 12-column responsive layout, Playwright end-to-end workflows with
`@axe-core/playwright` checks, and aggregate coverage gates.

The frontend consumes the Phase 04 generated search/autocomplete contracts. The
`ServiceWorker` static aspect of DESIGN-001 (offline asset/API/image
interception) is **not** implemented in Phase 05 and is explicitly deferred to
Phase 09. SW-REQ-006 (multi-meal Daily Diet aggregation) is **not** claimed by
Phase 05 and remains Phase 07 scope alongside the saved-diet model and
optimization worker; Phase 05 keeps only the Daily Diet Alternative scaffold and
does not claim Daily Diet Alternative execution, field-level rejection UX, saved
diet data, or optimization job behavior.

## Automated Evidence

Run from the repository root unless noted. These commands were actually run
during tasks `153`-`154`:

```sh
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check:api-types
python3 scripts/validate-task-list.py
python3 scripts/validate-traceability.py
python3 scripts/verify-frontend.py
npx --no-install redocly lint api/openapi.yaml
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./...
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./...
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...
```

Observed results from task `153`:

- `bun test` passed with `203 pass`, `0 fail`.
- `bun test --coverage` reported `All files | 100.00 | 100.00` line and function
  coverage for testable TypeScript frontend source. Svelte `.svelte` components
  remain outside Bun's line-coverage report and are verified by static-source
  assertions plus Playwright end-to-end tests and `vite build` (see Accepted
  Deviations).
- `bun run build` produced `184 modules`.
- `bun run test:e2e` (Playwright) passed with `75 passed`, `1 skipped`. The
  skipped case is an autocomplete scaffold pending `SearchShell` wiring noted in
  `docs/implementation/04_OPEN.md`.
- `bun run check:api-types` passed; generated types are current with the
  OpenAPI source of truth.
- `python3 scripts/validate-task-list.py` validated `154` sequential tasks with
  ordered dependencies.
- `python3 scripts/validate-traceability.py` passed.
- `python3 scripts/verify-frontend.py` passed and wrote desktop/mobile
  screenshots under `/tmp/mealswapp-frontend-verifier/`.
- `npx redocly lint api/openapi.yaml` reported the OpenAPI description valid
  with one explicitly ignored existing OAuth redirect warning.
- `go test ./...`, `go vet ./...`, and `go test -race ./...` all passed.
- `govulncheck` reported `0 vulnerabilities`.

## Project-Owner Checks

### Integration Checks (API + Frontend)

1. Start PostgreSQL and Redis with `bash scripts/start-services.sh`, then run
   `cd backend && go run ./cmd/migrate up && go run ./cmd/api`.
2. From `frontend/`, run `bun run dev` and open the served URL in a
   Chromium-compatible browser; confirm the search shell loads without console
   errors and renders against the live API.
3. Confirm `bun run check:api-types` passes so the frontend's generated search
   and autocomplete contracts have not drifted from the OpenAPI source of truth.
4. Perform a Catalog search against the running backend; confirm request and
   response shapes match the generated `SearchRequest`/`SearchResponse` types
   and the response includes cache metadata.

### Functional Checks

1. **Default search state (SW-REQ-001):** On first load, confirm the Search Mode
   is `Catalog`, unit preferences load with safe defaults, and result cards show
   protein, carbohydrate, and fat values when results are available.
2. **Debounce timing (SW-REQ-002):** Type rapidly into the search bar; confirm
   no request fires before `150ms` and exactly one request fires after the final
   keystroke.
3. **Local query cache (SW-REQ-003):** Repeat a previously executed search;
   confirm the cached result set renders immediately from localStorage. Perform
   twenty-one unique searches; confirm the least-recent entry is evicted.
4. **Autocomplete ranking (SW-REQ-004):** Type a query with an exact match;
   confirm exact matches rank ahead of fuzzy matches and equal scores stay
   deterministic across repeated calls.
5. **Substitution Inputs (SW-REQ-005):** Switch to Substitution mode, select an
   autocomplete item, and press `Enter`; confirm the selected FoodObject hydrates
   through `/api/v1/food-objects/{id}` and one Substitution Input is added. Add
   and remove inputs and confirm quantities and canonical units reach
   `SearchRequest`.
6. **Daily Diet Alternative scaffold:** Switch to Daily Diet Alternative mode;
   confirm the scaffold is visible and does not create Phase 07 job behavior.
   SW-REQ-006 multi-meal aggregation, saved-diet data, optimization jobs, and
   field-level rejection UX are Phase 07 scope.
7. **UI toggle positioning (SW-REQ-007):** Confirm search mode controls sit
   above the search bar and settings controls.
8. **Filters and pagination (SW-REQ-010):** Apply food-category/allergen/dietary
   filters; confirm results respect them. Confirm no page renders more than `10`
   items and previous results stay visible during page loads.
9. **Search history and favorites (SW-REQ-013):** Log in, perform a successful
   search, and confirm it appears in the Activity Sidebar history; select a
   history entry and confirm search state restores. Confirm anonymous users see
   empty/sign-in guidance and that activity API failure does not block public
   Catalog search.
10. **Timeout and retry:** Trigger a request that exceeds the 10-second timeout;
    confirm previous state is retained and a retry action is offered. Confirm
    `400`/`422`/`429`/`503` responses map to stable `AppError` envelopes through
    DESIGN-017 `ErrorMessageMapper`.

### End-to-End Checks (Playwright)

1. Run `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e`
   against a running API; confirm `75 passed`, `1 skipped`.
2. Covered workflows include keyboard-only Catalog and Substitution flows at
   desktop and mobile sizes, autocomplete selection, pagination, cache reuse,
   history/favorites, offline cached display, timeout retry, selected FoodObject
   hydration, and the empty state.
3. The single skipped spec is the autocomplete Playwright scaffold pending
   `SearchShell` wiring, documented in `docs/implementation/04_OPEN.md`; the
   autocomplete behavior is otherwise covered by fake-timer unit tests and
   component tests.

### Accessibility Checks (Keyboard, axe, WCAG AA)

1. Run the `frontend/tests/accessibility.spec.ts` Playwright suite; confirm the
   automated axe scans report only the accepted `color-contrast` violations on
   decorative elements (see Accepted Deviations) and that re-running axe with
   `color-contrast` disabled reports no serious or critical violations for the
   composed shell.
2. Confirm normal reading-text pairs (body `--color-text` and muted
   `--color-muted` labels on `--color-bg`/`--color-surface`) meet WCAG 2.1 AA
   4.5:1 in both light and dark themes.
3. Confirm focus is visible and all interactive controls have accessible names.
4. **Keyboard navigation (SW-REQ-009):** With the autocomplete dropdown open,
   press `Tab` to move focus forward through suggestions and `Shift+Tab` to move
   backward; press `Enter` to select the active option and `Escape` to dismiss.
   Confirm ARIA combobox/listbox state is correct.

### Responsive Checks

1. Confirm a 12-column desktop grid renders the search layout with the sidebar
   docked on the left, and a single-column layout renders below `640px`
   (SW-REQ-014).
2. Resize the viewport to `320px` width; confirm no horizontal scrolling occurs.
3. Confirm Inter is used for UI text and Roboto Mono for data labels, and that
   light/dark design tokens match `docs/requirements/02_STYLE_GUIDE.md`
   (SW-REQ-089).
4. Confirm stable result-card dimensions across breakpoints and that the sidebar
   collapse/expand and mobile toggle behave as designed.

### Visual Acceptance Tests (Light/Dark, Screenshots)

1. Inspect the screenshots produced by `python3 scripts/verify-frontend.py` and
   by the Playwright accessibility/responsive suites under
   `frontend/test-results/accessibility/` and `frontend/test-results/responsive/`:
   - `a11y-desktop-light.png`, `a11y-desktop-dark.png`
   - `a11y-mobile-light.png`, `a11y-mobile-dark.png`
   - `responsive-desktop-light.png`, `responsive-desktop-dark.png`
   - `responsive-mobile-light.png`, `responsive-mobile-dark.png`
2. Confirm light and dark themes render the search shell, result cards, sidebar,
   and offline banner with correct tokens and no layout regressions.

#### Desktop Keyboard Acceptance Test

On a desktop viewport, tab into the search bar and type a query; after
`150ms` the autocomplete dropdown opens below the input as a positioned overlay
without shifting page layout or obscuring the typed input. Press `Tab` to
advance focus through ranked suggestions and `Shift+Tab` to reverse; the active
option is visually indicated. Press
`Enter` to select the active suggestion; in Substitution mode this adds one
Substitution Input above the settings controls, and in Catalog mode this runs
the search. Press `Escape` to dismiss the dropdown without selection. Tab
through the unit preference controls, mode controls, pagination controls, and
sidebar entries; confirm visible focus and reachable, named controls throughout.

#### Mobile Keyboard/Touch Acceptance Test

On a `320px`-wide mobile viewport, tap the search bar and type; the
autocomplete dropdown opens below the input without horizontal scrolling or
layout shift. Tap a
suggestion to select it. Use the mobile sidebar toggle to open and close the
activity sidebar; confirm single-column layout and that history/favorites
restore search state. Toggle the light/dark theme switcher and confirm the
preference persists across a reload. Confirm no content is clipped and all
controls remain operable by touch.

## Traceability

Primary design sources:

- `docs/design/DESIGN-001.md`: SearchView, SidebarComponent, ResultsGrid,
  AutocompleteDropdown, OfflineBanner, SettingsPanel, LocalStorageManager. The
  `ServiceWorker` static aspect of DESIGN-001 is deferred to Phase 09.
- `docs/design/DESIGN-016.md`: ThemeProvider, LayoutGrid, ComponentStyles, color
  and typography tokens, 12-column grid and single-column mobile layout.
- `docs/design/DESIGN-017.md`: ErrorMessageMapper and stable search error
  envelopes mapped from `400`/`422`/`429`/`503` responses.

Related Phase 05 task IDs:

- `138` Phase 04 search contract and cleanup follow-up (unblocked Phase 05
  result rendering).
- `139` frontend search tooling (TanStack Query, Playwright, axe).
- `140` search state and request builder.
- `141` local query LRU cache (SW-REQ-003).
- `142` generated search API client.
- `143` search settings controls (unit preference).
- `144` search modes and Substitution Inputs (SW-REQ-005, SW-REQ-007).
- `145` autocomplete interaction (SW-REQ-002, SW-REQ-004, SW-REQ-008,
  SW-REQ-009).
- `146` search results grid (SW-REQ-010, SW-REQ-011, SW-REQ-012).
- `147` activity sidebar (SW-REQ-013).
- `148` offline and stale indicator.
- `149` theme persistence with explicit light/dark sidebar toggle (SW-REQ-015).
- `150` responsive style system (SW-REQ-014, SW-REQ-089).
- `151` search workflow integration.
- `152` browser accessibility and responsive gate.
- `153` coverage and aggregate gate.
- `154` acceptance documentation (this document).

Requirement coverage:

- `SW-REQ-001` Default Search State: tasks `140`, `151`.
- `SW-REQ-002` Search Debounce Timing: tasks `145`, `151`.
- `SW-REQ-003` Local Query Caching: tasks `141`, `151`.
- `SW-REQ-004` Autocomplete Ranking Priority: tasks `145`, `151` (server ranking
  consumed by the frontend).
- `SW-REQ-005` Ingredient List Accumulation: tasks `142`, `144`, `151`.
- `SW-REQ-006` Search Mode: Daily Diet: **Phase 07 scope.** Phase 05 exposes
  only the Daily Diet Alternative scaffold.
- `SW-REQ-007` UI Toggle Positioning: tasks `144`, `150`.
- `SW-REQ-008` Search Bar Expansion: tasks `145`, `151`.
- `SW-REQ-009` Keyboard Navigation: Autocomplete: tasks `145`, `152`.
- `SW-REQ-010` Search Result Pagination: tasks `146`, `151`.
- `SW-REQ-011` Search Result Data Fields: tasks `138`, `146`.
- `SW-REQ-012` Category-Based Placeholders: tasks `138`, `146`.
- `SW-REQ-013` Activity Sidebar: tasks `147`, `151`.
- `SW-REQ-014` Responsive Web Interface: tasks `150`, `152`.
- `SW-REQ-015` Light/Dark Mode Toggle: tasks `149`, `150`.
- `SW-REQ-089` Style Guide: tasks `150`, `152`.

## Deferred Phase 09 Behavior

- The DESIGN-001 `ServiceWorker` static aspect (offline asset/API/image
  interception and cache policy delegation to ARCH-011) is **not** implemented
  in Phase 05. Phase 05 only subscribes to browser `online`/`offline` events and
  shows cached-result, stale-result, and online-only fallback status from the
  in-memory/localStorage query cache.
- Phase 09 remains responsible for service-worker API and image interception,
  broader offline hardening, and edge TLS/ingress enforcement. The Phase 05
  `OfflineBanner` tests explicitly disclaim service-worker interception
  coverage.

## Known Deviations

- **Frontend coverage:** `bun test --coverage` reports `100%` line and function
  coverage for testable TypeScript frontend source. Svelte `.svelte` components
  are outside Bun's line-coverage report and are verified by static-source
  assertions, Playwright end-to-end tests (`75 passed`, `1 skipped`), and
  `vite build`. This is the accepted Svelte-component coverage approach and is
  documented in `docs/implementation/04_OPEN.md`.
- **Accepted color-contrast deviations (Task 152):** Automated axe scans report
  `color-contrast` (serious) violations on decorative elements that use
  `text-white` on mid-tone backgrounds. These are accepted visual-design
  limitations, not normal reading-text pairs: ResultCard similarity tier badges
  (Fair badge in both themes; Excellent/Good/Poor badges in dark theme),
  ResultCard category chips and image placeholder text (dark theme), and the
  SidebarComponent active search-mode button (dark theme). The gate asserts
  these are the only serious/critical axe violations and re-runs axe with
  `color-contrast` disabled to confirm the rest of the composed shell is clean.
  Follow-up: a future visual-design pass should introduce theme-aware
  on-accent/on-muted text tokens so the decorative badges, chips, placeholder,
  and active sidebar button meet 4.5:1 in both themes.
- **Macro visibility controls:** Phase 05 intentionally does not implement
  macro visibility toggles. Result cards always show the required protein,
  carbohydrate, and fat values from the generated search contract; SettingsPanel
  scope is limited to unit and theme preferences.
- **Daily Diet Alternative behavior:** Phase 05 intentionally keeps Daily Diet
  Alternative as a scaffold only. Full execution, saved-diet data, optimization
  jobs, and field-level rejection UX move to Phase 07.
- **Theme sidebar option:** The sidebar intentionally exposes a simple
  light/dark toggle. `system` remains supported inside `ThemeProvider` for
  first-load defaults and live system-theme resolution, but is not a visible
  sidebar option.
- **HTTP status mapping:** Task `142` maps `429` to `server` category and `422`
  to `validation` category (`search_rejected`) because the generated
  `ErrorCategory` enum has no `rate_limit`/`rejection` category. This stays
  within the generated contract.
- **Skipped Playwright spec:** One autocomplete Playwright scaffold is skipped
  pending `SearchShell` wiring; autocomplete behavior is otherwise covered by
  fake-timer unit tests and component tests.

## Acceptance

Accept Phase 05 after the automated evidence remains green and the
project-owner checks confirm: the search shell loads against the live API;
Catalog is the initial mode; unit preferences load and persist; 150ms debounce
and 20-entry LRU caching behave; autocomplete ranking and keyboard navigation work;
selected FoodObject hydration works for Substitution Inputs; Substitution Inputs
accumulate and reach `SearchRequest`; the Daily Diet Alternative scaffold does
not start Phase 07 job behavior (SW-REQ-006 remains Phase 07); filters and
pagination respect the 10-item limit; the
activity sidebar shows history/favorites and restores search state; offline and
stale indicators show without claiming service-worker interception; the explicit
light/dark theme preference persists; the 12-column desktop grid and `320px`
no-scroll mobile layout render with the documented style guide; desktop and
mobile keyboard and visual acceptance tests pass; and the accepted color-contrast and Svelte
component coverage deviations remain documented in
`docs/implementation/04_OPEN.md`.
