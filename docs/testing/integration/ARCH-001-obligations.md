# ARCH-001 Integration Verification Obligations

## Purpose

This document defines the SWE.5 integration verification obligations for architecture component ARCH-001, the Web Application Module.

The goal is to verify that SearchView, SidebarComponent, ResultsGrid, AutocompleteDropdown, ThemeProvider, OfflineBanner, SettingsPanel, LocalStorageManager, TanStack Query, and generated API contracts collaborate according to the architecture.

## Component Information

| Field | Value |
| --- | --- |
| Architecture Component | ARCH-001 |
| Name | Web Application Module |
| Source Documents | `docs/architecture/ARCH-001.md`, `docs/design/DESIGN-001.md`, `docs/design/DESIGN-016.md`, `docs/design/DESIGN-017.md` |
| Related Units | SearchView, SidebarComponent, ResultsGrid, AutocompleteDropdown, ThemeProvider, OfflineBanner, SettingsPanel, LocalStorageManager, generated search API client, TanStack Query |
| Collaborating Architecture | ARCH-010, ARCH-011, ARCH-016, ARCH-017 |
| Related Requirements | SW-REQ-001, SW-REQ-002, SW-REQ-003, SW-REQ-005, SW-REQ-007, SW-REQ-008, SW-REQ-009, SW-REQ-010, SW-REQ-011, SW-REQ-012, SW-REQ-013, SW-REQ-014, SW-REQ-015, SW-REQ-018, SW-REQ-025, SW-REQ-077, SW-REQ-085, SW-REQ-086, SW-REQ-087, SW-REQ-088, SW-REQ-089 |

## IT-ARCH-001-001 Catalog Search Shell Request and Result Flow

### Intent

Verify that the composed web application initializes in Catalog mode, accepts user search input, builds the generated `SearchRequest`, executes it through the API client/TanStack Query boundary, renders result cards, and paginates the response without handwritten contract drift.

### System Under Test

ARCH-001 Web Application Module, centered on SearchView.

### Real Components

- SearchView
- AutocompleteDropdown
- generated search API client
- TanStack Query composition
- ResultsGrid
- ResultCard
- LocalStorageManager

### Allowed Test Doubles

- Playwright route interception may stand in for ARCH-010 API Gateway responses while preserving generated frontend response types.

### Trigger / Stimulus

User opens the SPA, enters a Catalog query, submits the query, and navigates to the next result page.

### Expected Integrated Behavior

1. The SPA initializes to Catalog mode with no initial result grid.
2. User-entered query text is not sent as a final search until explicit submission.
3. SearchView sends a generated-contract search request through the API client boundary.
4. ResultsGrid renders result cards from the generated response contract.
5. Pagination requests carry the requested page and update the visible page state.

### Required Evidence

- Test verifies initial state, request payload, rendered results, result count, and pagination request state.
- Test traceability comment references `IT-ARCH-001-001`, `ARCH-001`, and related SW requirements.

### Requirement Traceability

- SW-REQ-001
- SW-REQ-010
- SW-REQ-011
- SW-REQ-012

### Verification Status

Implemented by:

- `frontend/tests/search-workflow.spec.ts::initial Catalog view hides search results until the user enters a query`
- `frontend/tests/search-workflow.spec.ts::initial Catalog search renders ranked results after typing a query`
- `frontend/tests/search-workflow.spec.ts::typing a query waits for Enter before sending the final search text`
- `frontend/tests/search-workflow.spec.ts::pagination loads page 2 and reflects the page in the request`

Status: PASS.

## IT-ARCH-001-002 Autocomplete Ranking, Overlay, Selection, and Keyboard Flow

### Intent

Verify that AutocompleteDropdown integrates with SearchView and generated autocomplete responses so ranked suggestions appear after the debounce interval, remain a positioned overlay, expose correct ARIA state, and support keyboard navigation, selection, and dismissal.

### System Under Test

ARCH-001 Web Application Module, centered on AutocompleteDropdown within SearchView.

### Real Components

- SearchView
- AutocompleteDropdown
- autocomplete controller
- generated autocomplete API client boundary
- Search mode focus management

### Allowed Test Doubles

- Playwright route interception may stand in for ARCH-010 autocomplete responses while preserving generated frontend response types.

### Trigger / Stimulus

User types into the search bar, waits for autocomplete suggestions, navigates with keyboard controls, and selects or dismisses suggestions.

### Expected Integrated Behavior

1. A 150ms debounced autocomplete query retrieves server-ranked suggestions.
2. Suggestions display in server rank order.
3. The dropdown is positioned without shifting existing results.
4. Tab and Shift+Tab move focus through options.
5. Arrow and Enter selection update the search bar while Escape dismisses the dropdown.
6. Combobox/listbox ARIA state remains valid.

### Required Evidence

- Test verifies debounce behavior, ranking order, layout position, keyboard navigation, selection, dismissal, and ARIA state.
- Test traceability comment references `IT-ARCH-001-002`, `ARCH-001`, and related SW requirements.

### Requirement Traceability

- SW-REQ-002
- SW-REQ-008
- SW-REQ-009
- SW-REQ-086

### Verification Status

Implemented by:

- `frontend/tests/search-workflow.spec.ts::autocomplete shows ranked suggestions after the 150ms debounce`
- `frontend/tests/autocomplete.spec.ts::types in the search bar and verifies ranked suggestions appear after the 150ms debounce`
- `frontend/tests/autocomplete.spec.ts::opening autocomplete suggestions does not push results down`
- `frontend/tests/autocomplete.spec.ts::Tab moves focus forward through options and Shift+Tab moves it backward`
- `frontend/tests/autocomplete.spec.ts::ArrowDown moves the active suggestion instead of moving the text caret`
- `frontend/tests/autocomplete.spec.ts::Escape dismisses the dropdown and returns focus to the combobox`
- `frontend/tests/autocomplete.spec.ts::combobox exposes aria-expanded, aria-controls, and inactive suggestions before navigation`

Status: PASS.

## IT-ARCH-001-003 Substitution Input Hydration and Explicit Search Flow

### Intent

Verify that Substitution mode composes mode controls, autocomplete selection, FoodObject hydration, Substitution Input state, quantity/unit controls, explicit search triggering, generated request construction, and ResultsGrid similarity rendering.

### System Under Test

ARCH-001 Web Application Module, centered on SearchView and Substitution Inputs.

### Real Components

- SearchView
- SearchModes
- AutocompleteDropdown
- generated food-object detail API client boundary
- SubstitutionInputs
- SettingsPanel unit handling
- ResultsGrid

### Allowed Test Doubles

- Playwright route interception may stand in for ARCH-010 search, autocomplete, and food-object detail responses while preserving generated frontend response types.

### Trigger / Stimulus

User switches to Substitution mode, selects an autocomplete item, confirms hydrated FoodObject data, adjusts quantity, and activates the explicit substitution search.

### Expected Integrated Behavior

1. Search mode controls switch to Substitution and reset incompatible results.
2. Autocomplete selection hydrates a rich FoodObject through `/api/v1/food-objects/{id}`.
3. Substitution Input controls render macros, category, calories, basis, quantity, and unit.
4. Explicit search sends mode `substitution`, empty query, selected FoodObject ID, quantity, and canonical unit.
5. ResultsGrid renders similarity metadata for the substitution results.

### Required Evidence

- Test verifies mode transition, hydration, displayed food data, request body, explicit search, and result rendering.
- Test traceability comment references `IT-ARCH-001-003`, `ARCH-001`, and related SW requirements.

### Requirement Traceability

- SW-REQ-005
- SW-REQ-007
- SW-REQ-011
- SW-REQ-018
- SW-REQ-025

### Verification Status

Implemented by:

- `frontend/tests/search-workflow.spec.ts::Substitution Input search sends inputs and renders ranked results`
- `frontend/tests/search-workflow.spec.ts::Catalog results can add full item data to the substitution input list`
- `frontend/tests/accessibility.spec.ts::keyboard-only Substitution workflow switches mode, adds an input, and searches via keyboard`

Status: PASS.

## IT-ARCH-001-004 Cache, Offline Indicator, Timeout, and Recovery Flow

### Intent

Verify that LocalStorageManager, TanStack Query, OfflineBanner, generated API client timeout handling, and error display collaborate so cached results remain usable offline and timeout recovery does not lose application state.

### System Under Test

ARCH-001 Web Application Module, centered on SearchView resilience behavior.

### Real Components

- SearchView
- LocalStorageManager
- generated search API client
- TanStack Query composition
- OfflineBanner
- ResultsGrid
- ErrorMessageMapper-facing UI state

### Allowed Test Doubles

- Playwright route interception may stand in for ARCH-010 responses and delayed network behavior.
- Browser offline emulation may stand in for connection-loss conditions.

### Trigger / Stimulus

User performs a search, repeats it, loses browser connectivity, and later encounters and recovers from a timed-out search.

### Expected Integrated Behavior

1. Repeating an equivalent search reuses cached state without an extra network call.
2. Offline status is detected through browser events and the OfflineBanner becomes visible.
3. Previously cached results remain visible while offline.
4. A 10-second timeout surfaces an error state instead of hanging indefinitely.
5. A subsequent successful request recovers the result grid.

### Required Evidence

- Test verifies cache reuse, network-call suppression, offline indicator, retained result cards, timeout error state, and recovery result rendering.
- Test traceability comment references `IT-ARCH-001-004`, `ARCH-001`, and related SW requirements.

### Requirement Traceability

- SW-REQ-003
- SW-REQ-077
- SW-REQ-087
- SW-REQ-088

### Verification Status

Implemented by:

- `frontend/tests/search-workflow.spec.ts::repeating the same search reuses the cache without a second network call`
- `frontend/tests/search-workflow.spec.ts::going offline shows the OfflineBanner while cached results remain visible`
- `frontend/tests/search-workflow.spec.ts::a slow search surfaces an error and a retry succeeds`

Status: PASS.

## IT-ARCH-001-005 Sidebar, Activity Data, Unit Preference, and Theme Persistence Flow

### Intent

Verify that SidebarComponent integrates with generated authenticated activity contracts, settings controls, unit preference state, ThemeProvider, local persistence, and responsive sidebar visibility without blocking core search behavior.

### System Under Test

ARCH-001 Web Application Module, centered on SidebarComponent and ThemeProvider.

### Real Components

- SidebarComponent
- SettingsPanel unit preference control
- generated profile/search-history/saved-items API client boundaries
- ThemeProvider
- LocalStorageManager
- SearchView shell

### Allowed Test Doubles

- Playwright route interception may stand in for ARCH-010 authenticated profile, history, and saved-item responses while preserving generated frontend response types.

### Trigger / Stimulus

User opens the app, observes authenticated sidebar activity data, changes unit preference, toggles theme, and reloads the SPA.

### Expected Integrated Behavior

1. Sidebar loads authenticated search history and favorites from generated contract responses.
2. Unit preference changes between metric and imperial in the sidebar settings area.
3. ThemeProvider applies an explicit light/dark override.
4. Theme selection persists across reload and restores visible control state.

### Required Evidence

- Test verifies activity entries, favorite entries, unit setting changes, document theme token state, reload persistence, and toggle state.
- Test traceability comment references `IT-ARCH-001-005`, `ARCH-001`, and related SW requirements.

### Requirement Traceability

- SW-REQ-013
- SW-REQ-015
- SW-REQ-048

### Verification Status

Implemented by:

- `frontend/tests/search-workflow.spec.ts::authenticated sidebar loads search history and favorites`
- `frontend/tests/search-workflow.spec.ts::sidebar unit preference changes between metric and imperial`
- `frontend/tests/search-workflow.spec.ts::explicit theme selection restores across a reload`

Status: PASS.

## IT-ARCH-001-006 Responsive Layout, Keyboard Accessibility, and Style System Flow

### Intent

Verify that LayoutGrid, ComponentStyles, ThemeProvider, SearchView, SidebarComponent, AutocompleteDropdown, ResultsGrid, and accessibility behavior collaborate at desktop and mobile breakpoints with keyboard-only operation and documented style tokens.

### System Under Test

ARCH-001 Web Application Module, centered on the composed browser UI.

### Real Components

- SearchView
- SidebarComponent
- AutocompleteDropdown
- ResultsGrid
- ThemeProvider
- LayoutGrid
- ComponentStyles

### Allowed Test Doubles

- Playwright route interception may stand in for ARCH-010 search/autocomplete responses while preserving generated frontend response types.

### Trigger / Stimulus

Browser runs the composed shell at desktop and mobile viewports, performs keyboard-only Catalog and Substitution workflows, runs axe checks, inspects normal-text contrast, and captures light/dark screenshots.

### Expected Integrated Behavior

1. Mobile layout avoids horizontal scrolling at 320px.
2. Desktop layout places the sidebar at the viewport-left edge and main content beside it.
3. Mobile layout stacks sidebar and main content in one column.
4. Keyboard-only Catalog and Substitution workflows reach controls, submit searches, and keep focus visible.
5. Interactive controls have accessible names and no serious or critical axe violations outside documented color-contrast deviations.
6. Normal text contrast and documented style tokens match the style guide.

### Required Evidence

- Test verifies layout measurements, keyboard-only workflows, focus indicators, axe results, accessible names, contrast ratios, design tokens, typography, and screenshots.
- Test traceability comment references `IT-ARCH-001-006`, `ARCH-001`, and related SW requirements.

### Requirement Traceability

- SW-REQ-014
- SW-REQ-085
- SW-REQ-086
- SW-REQ-089

### Verification Status

Implemented by:

- `frontend/tests/responsive.spec.ts`
- `frontend/tests/accessibility.spec.ts`

Status: PASS.

## SWE.5 Checklist Evaluation

| Obligation | ARCH Trace | SW-REQ Trace | Collaborating Units | Real Components Practical | Behavior Type | Test Evidence | Decision |
| --- | --- | --- | --- | --- | --- | --- | --- |
| IT-ARCH-001-001 | PASS | PASS | PASS | PASS | Data flow, sequence, state transition | PASS | PASS |
| IT-ARCH-001-002 | PASS | PASS | PASS | PASS | Sequence, data flow, keyboard interaction | PASS | PASS |
| IT-ARCH-001-003 | PASS | PASS | PASS | PASS | State transition, data flow, explicit trigger | PASS | PASS |
| IT-ARCH-001-004 | PASS | PASS | PASS | PASS | Failure handling, recovery, state retention | PASS | PASS |
| IT-ARCH-001-005 | PASS | PASS | PASS | PASS | Data flow, persistence, state transition | PASS | PASS |
| IT-ARCH-001-006 | PASS | PASS | PASS | PASS | Responsive state, accessibility, style integration | PASS | PASS |

## Verification Commands

The Phase 05 SWE.5 evidence is covered by:

```bash
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build
python3 scripts/validate-task-list.py
python3 scripts/validate-traceability.py
```

## Completion Decision

SWE.5 coverage for ARCH-001 is complete for the Phase 05 scope.

All obligations are implemented by existing Playwright browser integration tests, the tests now contain obligation traceability comments, practical verification passes, and no ARCH-001 Phase 05 obligation remains uncovered. Full ServiceWorker API/cache interception remains deferred to Phase 09 as recorded in the Phase 05 UAT and is not claimed here.
