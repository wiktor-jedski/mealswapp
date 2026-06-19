# Phase 05 User Acceptance Tests

## Scope

Phase 05 implements the frontend search experience and the Phase 04 result-contract follow-up covered by Tasks 138–154. The delivered surface includes generated search contracts, typed client state, a 20-entry local LRU cache, the credentialed search/autocomplete client, search settings, Catalog/Substitution/Daily Diet Alternative controls, debounced autocomplete, result cards and pagination, activity sidebar, offline cache feedback, theme persistence, responsive styling, and browser accessibility gates.

The implementation traces to `DESIGN-001` SearchView, LocalStorageManager, SettingsPanel, AutocompleteDropdown, ResultsGrid, SidebarComponent, and OfflineBanner; `DESIGN-002` SearchController; `DESIGN-005` MacroNormalizer; `DESIGN-011` LocalStorageCache; `DESIGN-014` MetricsCollector; `DESIGN-016` ThemeProvider, ColorPalette, TypographySystem, LayoutGrid, and ComponentStyles; and `DESIGN-017` ErrorMessageMapper.

## Requirement Traceability

| Requirement | Phase 05 evidence |
|---|---|
| SW-REQ-001 | Catalog is the default state and all macro toggles start enabled. |
| SW-REQ-002 | Autocomplete uses a tested 150 ms trailing debounce. |
| SW-REQ-003 | localStorage retains the 20 most recent unique normalized request/result pairs with LRU refresh and stale timestamps. |
| SW-REQ-004 | The UI preserves the server-ranked autocomplete order. |
| SW-REQ-005 | Enter selects autocomplete entries into the quantity-bearing Substitution Input list. |
| SW-REQ-007 | Search-mode controls render above the search bar and macro controls. |
| SW-REQ-008 | Autocomplete expands in document flow. |
| SW-REQ-009 | Tab and Shift+Tab traverse suggestions; Enter selects and Escape dismisses. |
| SW-REQ-010 | Result pages render no more than 10 items and expose bounded Previous/Next controls. |
| SW-REQ-011 | Cards show image, name, Food Categories, macros with `100g`/`100ml` basis, calories, and available similarity score/tier. |
| SW-REQ-012 | Missing or broken images use deterministic primary-category placeholders. |
| SW-REQ-013 | The left activity sidebar provides collapse/mobile controls, history, favorites, modes, and settings. |
| SW-REQ-014 | The UI uses a 12-column desktop layout and one column below 640 px with no overflow at 320 px. |
| SW-REQ-015 | System/light/dark selection lives in the sidebar and persists across reloads. |
| SW-REQ-089 | Exact style-guide color tokens, Inter, Roboto Mono, responsive screenshots, keyboard checks, and axe checks are verified. |

SW-REQ-006 multi-meal Daily Diet aggregation is explicitly Phase 07 scope. Phase 05 only submits the designed Daily Diet Alternative request shape and renders its structured Phase 04 rejection; it does not create optimization jobs or claim saved-diet behavior.

## Automated Verification Evidence

The following commands were run successfully on 2026-06-18:

```sh
cd frontend && bun test
cd frontend && bun test --coverage
cd frontend && bun run build
cd frontend && bun run check:api-types
cd frontend && bun run test:browser
npx --no-install redocly lint api/openapi.yaml
python3 scripts/validate-task-list.py
python3 scripts/validate-traceability.py
MEALSWAPP_POSTGRES_PORT=55432 MEALSWAPP_REDIS_PORT=56379 \
MEALSWAPP_DATABASE_URL='postgres://mealswapp:mealswapp@localhost:55432/mealswapp?sslmode=disable' \
MEALSWAPP_REDIS_URL='redis://localhost:56379/0' \
python3 scripts/check.py --output logs/check-report.html
```

Results:

- 43 Bun tests passed; instrumented Phase 05 TypeScript reports 100.00% line coverage.
- 17 Chromium Playwright tests passed across Catalog, Substitution, Daily Diet rejection, autocomplete, cache/offline, history/favorites, retry, theme, responsive, and accessibility workflows.
- Desktop and mobile axe scans reported no serious or critical violations.
- Generated API types were current; frontend build, OpenAPI lint, task-list validation, and traceability validation passed.
- The aggregate check passed migrations, local-stack smoke tests, backend tests, race detection, vet, vulnerability scan, frontend verification screenshots, coverage, and browser tests.
- Aggregate evidence is stored in `logs/check-report.html` for this working session. The Phase 05 report and screenshots are copied into `docs/implementation/implemented/` as committed UAT evidence.

## Project-Owner Acceptance Tests

These checks are proposed and were not claimed as performed by the project owner in this session.

1. Catalog search
   - Open the application at desktop width.
   - Confirm Catalog is active and all three macro toggles are enabled.
   - Search for a known food and inspect no more than 10 cards.
   - Accept when cards show name, Food Category, image/fallback, protein, carbohydrate, fat, basis, calories, and any similarity metadata supplied by the API.

2. Substitution and autocomplete keyboard flow
   - Select Substitution using only the keyboard.
   - Type a partial food name, wait for suggestions, use Tab/Shift+Tab, then Enter.
   - Set quantity/unit, add and remove inputs, and submit.
   - Accept when server rank is preserved, the dropdown expands downward, selected quantities reach the request, and results remain keyboard operable.

3. Pagination, empty, and retry states
   - Search a dataset with more than 10 results and traverse pages.
   - Exercise a zero-result query and a retryable 503/timeout response.
   - Accept when page boundaries disable correctly, prior results remain during page loading, empty text is clear, and Retry preserves the request.

4. Activity and authenticated data
   - Test signed-in and anonymous sessions.
   - Collapse/expand the sidebar, select a history item, inspect favorites, then simulate activity API failure.
   - Accept when history restores search state, anonymous guidance is clear, and activity failure never disables public search.

5. Offline local-cache behavior
   - Complete a search online, disable network connectivity, and repeat it.
   - Try an uncached query, reconnect, and retry.
   - Accept when cached data remains visible with offline/stale labeling, uncached feedback is actionable, and reconnection permits a fresh request.

6. Theme, responsive, and visual acceptance
   - Verify system, light, and dark selection across reloads.
   - Inspect 1280×900, 390×844, and 320 px widths in light and dark modes.
   - Accept when the sidebar/layout transitions correctly, cards are stable, focus is visible, contrast is readable, typography matches the style guide, and no horizontal scrolling occurs.

7. Daily Diet boundary
   - Select Daily Diet Alternative, provide a valid UUID, and submit.
   - Accept when the structured Phase 07 availability rejection is shown and no optimization-job workflow is started.

## Coverage, Deferred Scope, and Known Notes

- Bun reports 100.00% line coverage for instrumented TypeScript. Bun does not report Svelte source-line percentages; the file-specific exception and Playwright substitution are recorded in `docs/implementation/04_OPEN.md`.
- Phase 09 remains responsible for service-worker API/image interception and broader offline production hardening. Phase 05 implements localStorage query/result caching and browser online/offline feedback only.
- The host's default PostgreSQL/Redis ports were occupied by another workspace. The successful aggregate run used configurable host ports `55432` and `56379`; default compose ports remain unchanged when overrides are absent.
- No unresolved Phase 05 clarification or immediate project-owner action is recorded.

## Acceptance Decision

Automated implementation evidence is complete and Tasks 138–154 are `PREPARED`. The phase is ready for project-owner acceptance. Change the phase tasks to `PASSED` only after the project owner completes or explicitly accepts the relevant checks above.
