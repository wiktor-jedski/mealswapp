# Task 152 Review — Phase 05 Browser Accessibility and Responsive Gate

Task row:
| 152 | Phase 05 Browser Accessibility and Responsive Gate | DESIGN-016: ComponentStyles | PREPARED | 0 | Phase 05: add Playwright end-to-end coverage and `@axe-core/playwright` checks for the complete desktop and mobile search workflows while UI components are built. | 145,151 | None | Chromium Playwright tests pass for keyboard-only Catalog and Substitution workflows at desktop and mobile sizes; automated axe scans report no serious or critical violations; normal-text color pairs meet WCAG 2.1 AA 4.5:1; focus is visible; controls have accessible names; and screenshots verify responsive light/dark layouts. |

## Preconditions

- Task 152 status in `docs/implementation/02_TASK_LIST.md:159` is **PREPARED** ✓
- Dep 145 (AutocompleteDropdown.svelte with ARIA combobox/listbox, keyboard nav) is **PASSED** (`02_TASK_LIST.md:152`) ✓
- Dep 151 (SearchShell.svelte composes all components; 64 Playwright tests pass) is **PASSED** (`02_TASK_LIST.md:158`) ✓

## Implementation inspected

- `frontend/tests/accessibility.spec.ts` (new, 367 lines): 6 Playwright tests parameterized over `desktop-chromium` (1280×720) and `mobile-chromium` (390×844, Pixel 5) projects via `playwright.config.ts`. Traceability comments citing `DESIGN-016 ComponentStyles` are present at lines 9, 92, 179, 215, 243, 281, 307, 334.
- `frontend/playwright.config.ts`: defines the two projects and a `bun run build && bun run preview` webServer on port 4173.
- `frontend/src/lib/components/SearchShell.svelte:68` carries a `DESIGN-016 LayoutGrid` traceability comment; `App.svelte:19` carries `DESIGN-016 ThemeProvider` traceability.
- Selectors referenced by the spec exist in the implementation: `#autocomplete-input` (`AutocompleteDropdown.svelte:138`), `#search-mode-substitution` (`SearchModes.svelte:13`), `[data-results-next]` (`ResultsGrid.svelte:112`), `section[aria-label="Substitution inputs"]` (`SubstitutionInputs.svelte:75`).
- `docs/implementation/04_OPEN.md:170-177` documents the accepted accessibility deviations: `color-contrast` (serious) on decorative ResultCard tier badges (Fair fails both themes; Excellent/Good/Poor fail dark), category chips, image placeholder text, and the active sidebar mode button — all `text-white` on mid-tone backgrounds. The gate asserts these are the ONLY serious/critical violations and re-runs axe with `color-contrast` disabled to confirm the rest of the shell is clean. Normal reading-text pairs are verified separately at 4.5:1.

## Verification results

Commands run from `/home/wiktor/Work/glm`:

- `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` -> **195 pass, 0 fail, 656 expect() calls across 18 files** (1.3s)
- `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` -> **vite v7.3.3, 184 modules transformed, built in 8.39s** (dist/index.html 0.51 kB, index.css 14.48 kB, index.js 122.55 kB)
- `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e tests/accessibility.spec.ts` -> **11 passed, 1 skipped, 0 failed (49.1s)** using 6 workers. The 1 skip is `captures responsive light and dark layouts` on the `mobile-chromium` project — intentional, because that test sets all four viewport/theme combinations itself and is gated by `test.skip(testInfo.project.name !== "desktop-chromium", ...)` at line 337.
- `git -C /home/wiktor/Work/glm status` -> on branch `multistep-phase-05-glm`; `frontend/tests/accessibility.spec.ts` untracked (new), `docs/implementation/04_OPEN.md` modified; many other Phase 05 files untracked/modified from earlier PASSED tasks.
- `python3 scripts/validate-task-list.py` -> **Task-list validation passed: 154 sequential tasks with ordered dependencies.**
- `python3 scripts/validate-traceability.py` -> **Traceability validation passed.**

Screenshot artifacts produced under `frontend/test-results/accessibility/`:
- `a11y-desktop-light.png` (169729 bytes)
- `a11y-desktop-dark.png` (170499 bytes)
- `a11y-mobile-light.png` (146907 bytes)
- `a11y-mobile-dark.png` (148126 bytes)

## Checklist

- **PASS** - C1: Keyboard-only Catalog workflow passes at desktop and mobile sizes. `tests/accessibility.spec.ts:180` runs on both projects; tabs to `#autocomplete-input`, types "apple", dismisses the autocomplete listbox with Escape (with a debounce-reopen guard), tabs to `[data-results-next]`, asserts focus, verifies a visible focus indicator, activates Next with Enter, and asserts `Page 2 of 2`.
- **PASS** - C2: Keyboard-only Substitution workflow passes at desktop and mobile sizes. `tests/accessibility.spec.ts:216` runs on both projects; tabs to `#search-mode-substitution`, activates it with Enter, tabs to `#substitution-food-object-id`, types `food-apple`, tabs to the Add button, activates it with Enter, asserts the row appears and 10 result cards render, and verifies a visible focus indicator.
- **PASS** - C3: Automated axe scans report no serious or critical violations outside the documented color-contrast deviations. `tests/accessibility.spec.ts:244` runs a full WCAG2a/2aa/21a/21aa axe analysis on both projects, filters to serious/critical, asserts the only such violations are `color-contrast` (the documented decorative deviations), then re-runs with `color-contrast` disabled and asserts zero serious/critical. Accepted deviations are recorded in `docs/implementation/04_OPEN.md:170-177` with a follow-up note for a future visual-design token pass. The axe rule is **not** disabled in the gate run — the gate asserts the violation set is exactly the documented one, then proves the remainder is clean. This matches the review rule that accepted decorative color-contrast deviations are acceptable if documented.
- **PASS** - C4: Normal-text color pairs meet WCAG 2.1 AA 4.5:1. `tests/accessibility.spec.ts:282` reads `--color-bg`, `--color-surface`, `--color-text`, `--color-muted` from the document root in both light and dark themes (switched via the Theme preference select) and asserts `contrastRatio(text, bg)`, `contrastRatio(text, surface)`, `contrastRatio(muted, bg)`, `contrastRatio(muted, surface)` ≥ 4.5 in both themes. The WCAG luminance and contrast helpers (lines 142-167) follow the 0.03928 / 1.055 / 2.4 formula.
- **PASS** - C5: Focus is visible. `expectFocusIndicatorVisible` (lines 119-139) checks the focused element's computed `boxShadow` (Tailwind ring) or `outline`/`outlineWidth` and is called from both keyboard workflow tests after focusing a real interactive control.
- **PASS** - C6: Controls have accessible names. `tests/accessibility.spec.ts:308` runs axe with rules `button-name`, `label`, `link-name`, `select-name`, `aria-input-field-name` in both Catalog and Substitution states on both projects, asserting zero violations.
- **PASS** - C7: Screenshots verify responsive light/dark layouts. `tests/accessibility.spec.ts:335` captures `a11y-desktop-light.png`, `a11y-desktop-dark.png`, `a11y-mobile-light.png`, `a11y-mobile-dark.png` (all 146-170 KB, verified on disk) by switching viewport and `data-theme` for all four combinations.

## Files inspected

- `frontend/tests/accessibility.spec.ts` - the new accessibility gate spec (6 tests × 2 projects).
- `frontend/playwright.config.ts` - defines desktop-chromium and mobile-chromium projects and the build+preview webServer.
- `frontend/package.json` - confirms `@axe-core/playwright@^4.11.3`, `@playwright/test@^1.61.0`, and `test:e2e` script.
- `frontend/src/lib/components/SearchShell.svelte` - DESIGN-016 traceability comment and composed shell.
- `frontend/src/lib/components/AutocompleteDropdown.svelte` - `#autocomplete-input` and sr-only label.
- `frontend/src/lib/components/SearchModes.svelte` - `#search-mode-substitution` button.
- `frontend/src/lib/components/SubstitutionInputs.svelte` - `section[aria-label="Substitution inputs"]`.
- `frontend/src/lib/components/ResultsGrid.svelte` - `[data-results-next]` and `[data-results-page]`.
- `frontend/src/App.svelte` - DESIGN-016 ThemeProvider traceability.
- `docs/implementation/02_TASK_LIST.md` - task 152 PREPARED, deps 145/151 PASSED.
- `docs/implementation/04_OPEN.md` - accepted accessibility deviations recorded under Phase 05.
- `frontend/test-results/accessibility/` - four responsive screenshots produced.

## Decision reason

All seven verification criteria are satisfied. The accessibility gate spec is parameterized over desktop and mobile Chromium projects and covers keyboard-only Catalog and Substitution workflows, axe scans (with the only serious/critical violations being the documented decorative `color-contrast` cases, and a clean re-run with that rule disabled), WCAG 2.1 AA 4.5:1 normal-text contrast in both themes, visible focus indicators, accessible control names via axe name rules, and responsive light/dark screenshots at all four viewport/theme combinations. Unit tests (195/195), build, task-list validation, and traceability validation all pass. The accepted color-contrast deviations are decorative (badges, chips, placeholder, active sidebar button) — not normal reading text — and are documented in `docs/implementation/04_OPEN.md` with a follow-up for a future visual-design token pass, which matches the review rule that accepted decorative color-contrast deviations are acceptable if documented. The axe `color-contrast` rule is not silently disabled in the gate; the gate asserts the violation set is exactly the documented one and then proves the remainder is clean. No repair instructions needed.
