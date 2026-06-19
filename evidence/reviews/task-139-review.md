# Review Evidence: Task 139

## Decision

**PASSED**

## Task Reviewed

| Field | Value |
|-------|-------|
| ID | 139 |
| Component | Phase 05 Frontend Search Tooling |
| Static Aspect | DESIGN-001: SearchView |
| Status (in 02_TASK_LIST.md) | PREPARED |
| Retries | 0 |
| Description | Phase 05: add TanStack Query for Svelte, Playwright, and `@axe-core/playwright` with deterministic Bun unit, component, and browser-test commands for the search experience. |
| Depends On | 16, 129 |
| Testing Coverage Exceptions | None |

## Dependency Check

| Dep ID | Title | Status | OK? |
|--------|-------|--------|-----|
| 16 | Frontend Test and Quality Commands | PASSED | yes |
| 129 | Phase 04 Frontend Search Contract Generation | PASSED | yes |

Task 139 itself is `PREPARED` in `docs/implementation/02_TASK_LIST.md:146`, matching the precondition. Review concerns only Task 139.

## Verification Checklist

| ID | Criterion | Result | Evidence |
|----|-----------|--------|----------|
| C1 | `frontend/package.json` contains pinned search-client dependency (TanStack Query for Svelte) | PASS | `frontend/package.json:18` adds `@tanstack/svelte-query: ^6.1.34`; `frontend/bun.lock` pins `@tanstack/svelte-query@6.1.34` (+ `@tanstack/query-core@5.101.0`). |
| C2 | `frontend/package.json` contains pinned browser-test dependencies (Playwright + @axe-core/playwright) | PASS | `frontend/package.json:23-24` adds `@axe-core/playwright: ^4.11.3` and `@playwright/test: ^1.61.0`; `frontend/bun.lock` pins `@axe-core/playwright@4.11.3`, `@playwright/test@1.61.0`, and transitive `axe-core@4.11.4`. |
| C3 | `frontend/package.json` contains the required commands (Playwright command + unit/build/test commands) | PASS | `frontend/package.json:9` adds `preview: "vite preview --port 4173 --strictPort"`; `:13` adds `test:e2e: "playwright test"`; existing `build`, `test`, `check` scripts remain. |
| C4 | A browser smoke fixture exists and can render the app against controlled API responses | PASS | `frontend/tests/smoke.spec.ts` intercepts `/api/v1/search` and `/api/v1/search/autocomplete` with controlled, generated-type-checked `SearchResponseEnvelope` / `AutocompleteEnvelope` payloads, navigates to `/`, and asserts the SearchView shell renders (`Mealswapp` h1, `Catalog` button, `Food search` label). |
| C5 | `bun test` passes | PASS | `bun test` (scoped to `src/` via `frontend/bunfig.toml`) -> 9 pass / 0 fail / 18 expect calls across 2 files. |
| C6 | `bun run build` passes | PASS | `vite build` -> 113 modules transformed, `dist/index.html` + CSS + JS emitted, built in 894ms, exit 0. |
| C7 | The Playwright command passes | PASS | `bun run test:e2e` -> `playwright test` ran 4 tests (desktop-chromium + mobile-chromium x 2 specs), 4 passed in 6.4s, exit 0. |
| C8 | JSON traceability is recorded in the required sidecar document | PASS | `frontend/package.json-trace.md` updated with a `## Phase 05 Frontend Search Tooling (Task 139)` section listing added deps, scripts, `bunfig.toml`, `playwright.config.ts`, and the smoke fixture; `package.json` is valid JSON (no inline comments). |

## Commands Run

| Command | Working dir | Exit | Result |
|---------|-------------|------|--------|
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun install` | `frontend/` | 0 | `Checked 81 installs across 157 packages (no changes)` — idempotent. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | `frontend/` | 0 | 9 pass / 0 fail, 18 expect() calls, 2 files (service-worker + theme). |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | `frontend/` | 0 | 113 modules transformed; `dist/index.html` 0.51 kB, `dist/assets/index-*.css` 8.79 kB, `dist/assets/index-*.js` 38.17 kB; built in 894ms. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e` | `frontend/` | 0 | 4 passed (6.4s): desktop + mobile x {search shell renders, axe smoke}. Playwright chromium-1228 already installed under `~/.cache/ms-playwright/`. |
| `git status --porcelain` | repo root | 0 | Modified: `.gitignore`, `docs/implementation/02_TASK_LIST.md`, `frontend/bun.lock`, `frontend/package.json`, `frontend/package.json-trace.md`. Untracked: `frontend/bunfig.toml`, `frontend/playwright.config.ts`, `frontend/tests/smoke.spec.ts`. Only intended files. |
| `git check-ignore frontend/dist frontend/node_modules frontend/test-results frontend/.bun-tmp frontend/.bun-install frontend/playwright-report/` | repo root | 0 | All generated artifacts are ignored; Playwright `test-results/`, `playwright-report/`, `/playwright/.cache/` patterns present in `.gitignore:31-35`. |
| `python3 scripts/validate-task-list.py` | repo root | 0 | `Task-list validation passed: 154 sequential tasks with ordered dependencies.` |
| `python3 scripts/validate-traceability.py` | repo root | 0 | `Traceability validation passed.` |
| `bun -e "JSON.parse(...)"` (package.json validity) | `frontend/` | 0 | `valid JSON` — no inline comments. |

## Files Inspected

| Path | Reason |
|------|--------|
| `frontend/package.json` | Confirm added deps (`@tanstack/svelte-query`, `@playwright/test`, `@axe-core/playwright`) and scripts (`preview`, `test:e2e`); confirm no inline JSON comments; confirm caret ranges are backed by exact `bun.lock` pins. |
| `frontend/bun.lock` | Confirm exact pins for added packages and their transitive deps (`@tanstack/query-core`, `axe-core`, `playwright`). |
| `frontend/bunfig.toml` | New file: `[test] root = "src"` scopes Bun unit/component tests to `src/`, excluding Playwright specs under `tests/`. Includes DESIGN-001 traceability comment. |
| `frontend/playwright.config.ts` | New file: Chromium desktop + mobile projects, `webServer: bun run build && bun run preview` on `http://localhost:4173`, `reporter: [["list"]]`, trace on first retry. Includes DESIGN-001 + tech-stack traceability comment. |
| `frontend/tests/smoke.spec.ts` | New file: route interception for `/api/v1/search` and `/api/v1/search/autocomplete` with controlled typed envelopes; asserts shell renders + axe reports no serious/critical violations. Minimal scaffold — does not implement search state, request builder, API client, controls, or results grid (later tasks 140-144). |
| `frontend/package.json-trace.md` | Sidecar updated with Task 139 section listing added deps, scripts, `bunfig.toml`, `playwright.config.ts`, and smoke fixture. |
| `.gitignore` | Playwright artifact patterns added (`test-results/`, `playwright-report/`, `/playwright/.cache/`). |
| `frontend/src/App.svelte` | Confirm shell composition unchanged (`<SearchShell />`); smoke assertions align with existing shell. |
| `frontend/src/lib/components/SearchShell.svelte` | Confirm `Mealswapp` h1, `Catalog` button, and `Food search` label (via `for="search"`/`id="search"`) exist — matches smoke spec selectors. Search input is still disabled with Phase 00 placeholder, so no later-task search behavior is implemented. |
| `frontend/src/lib/api/generated.ts` | Confirm `SearchResponseEnvelope`, `AutocompleteEnvelope`, `SearchResponse`, `AutocompleteResponse`, `FoodObject`, `SimilarityMetadata`, `RankedAutocomplete` types referenced by the smoke spec exist (from Task 129). |
| `docs/implementation/02_TASK_LIST.md` | Confirm Task 139 row is `PREPARED` and dependency tasks 16/129 are `PASSED`. |

## Coverage / Exception Review

- Testing Coverage Exceptions: `None` — no exceptions claimed.
- The implementation is a minimal tooling scaffold: dependency additions, script additions, Bun test scoping, Playwright config, and one smoke spec. It does not implement search state, request building, the API client, search controls, or result rendering (those are tasks 140-144 and remain `OPEN`). No coverage gate is triggered by this task.
- The smoke spec intentionally stubs `/api/v1/search` and `/api/v1/search/autocomplete` even though the current shell does not yet issue those requests; this establishes the controlled-response harness for later tasks and still exercises the shell render + axe scan now.
- No JSON inline comments were introduced. The `frontend/package.json` remains valid JSON.
- No generated artifacts (`node_modules/`, `dist/`, `.bun-tmp/`, `.bun-install/`, `test-results/`, `playwright-report/`, `/playwright/.cache/`) are staged; all are gitignored.

## Failure Details

None. All eight verification criteria are satisfied with direct evidence.

## Notes

- "Pinned" in the verification criterion is satisfied by the combination of caret ranges in `package.json` and exact-version resolution recorded in `frontend/bun.lock` (standard Bun pinning model). The lockfile pins `@tanstack/svelte-query@6.1.34`, `@playwright/test@1.61.0`, and `@axe-core/playwright@4.11.3`.
- The Playwright `webServer` runs `bun run build && bun run preview`, so `test:e2e` implicitly re-validates the build gate (C6) as part of C7.
- Playwright chromium browser was already installed (`~/.cache/ms-playwright/chromium-1228`); no `bunx playwright install` was needed.
