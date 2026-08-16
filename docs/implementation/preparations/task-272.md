# Task 272 Preparation — Frontend Functional, End-to-End, and Accessibility Regression Gate

## Outcome and baseline

Task 272 is implemented and verified against fixed baseline commit `e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f` using `/home/wiktor/.codex/agents/developer.toml`.

Dependencies 266, 267, 268, and 270 were inspected through their preparation and independent review evidence. Each review decision is `PASSED`. Their existing component and browser tests provide the detailed external-import lifecycle, Account Export, administration styling, and frontend documentation coverage aggregated by this task.

The shared worktree contains concurrent Phase 08.01 backend, frontend, API, script, task-list, open-item, preparation, and review changes. They were preserved. `docs/implementation/02_TASK_LIST.md` was read but not edited; Task 272 remains `OPEN`.

## Exact Task 272 changed paths

| Path | Task 272 responsibility |
| --- | --- |
| `frontend/tests/task272-frontend-gate.spec.ts` | Adds the aggregate rendered administration regression gate and four desktop/mobile light/dark screenshot artifacts under `/tmp/mealswapp-task-272/`. |
| `docs/implementation/preparations/task-272.md` | Records baseline, scope, executable symbols, commands, results, screenshots, and exact coverage disposition. |

No production component, client, generated contract, API source, checker, task-list status, phase UAT, or coverage-policy file was changed for Task 272.

## Added executable symbols

### `frontend/tests/task272-frontend-gate.spec.ts`

| Symbol | Kind | Purpose |
| --- | --- | --- |
| `privateItemId` | constant | Supplies one valid owner-free private-item identity for the authoritative export fixture. |
| `screenshotDirectory` | constant | Keeps Task 272 visual evidence outside the repository under `/tmp/mealswapp-task-272`. |
| `ok` | function | Builds deterministic successful gateway envelopes. |
| `json` | function | Fulfils browser-controlled HTTP responses. |
| `stubAdmin` | function | Installs bounded authenticated-admin, Account Export, classification, and external-search fixtures. |
| `tabTo` | function | Proves keyboard reachability without relying on a fixed tab count. |
| `setTheme` | function | Selects light or dark through keyboard activation and restores the mobile administration viewport after sidebar use. |
| `administration regressions remain keyboard-safe, responsive, accessible, themed, and motion-reduced` | Playwright test | Executes the Task 272 rendered style, keyboard, responsive, reduced-motion, axe, clipping, current-export, safe-search-state, and screenshot gate in both configured projects. |

The test module carries exact `DESIGN-009 UserAdminPanel` traceability.

## Verification evidence

### Functional and browser criteria

| Criterion | Evidence |
| --- | --- |
| Deferred import completions cannot overwrite current state | `external-import-workflow.spec.ts` passes the superseded success, conflict, ambiguity, and failure matrix in desktop and mobile projects. |
| Incompatible import controls are disabled | The same suite passes search/provider/pagination/candidate/draft/classification/import locking and clean completed-workflow reset. |
| Draft discard, cancellation, search transition, and focus restoration | Desktop/mobile tests pass native modal containment, Escape/keep, Enter/discard, stale-token invalidation, pagination fallback, and disappearing-opener fallback. |
| Authoritative Account Export refresh and deletion recovery | `admin-private-data.spec.ts` passes successful load followed by failed refresh, fail-closed loading/error state, accepted deletion followed by failed verification, successful retry, complete deletion cycle, stale-success clearing, and owner-field rejection. |
| Administration styling | `AdminVisualCompliance.test.ts` inventories all Phase 08 administration buttons and affected form controls, checks heading-only Bold 700 use, verifies the dedicated destructive foreground, and measures light/dark contrast. The aggregate browser test also checks rendered heading weight `700`, non-transparent form Surface, `1px` border, and reduced-motion `transition-property: none` with `0s` duration. |
| Exported frontend documentation | `validate-phase08-tsdoc.py`, all 24 generator tests, generated-type drift checking, typecheck, component tests, and production build pass. |
| Desktop/mobile, keyboard, themes, and reduced motion | The five-suite Playwright matrix passes all 62 cases across desktop Chromium and Pixel 5. The aggregate test enters and submits external search by keyboard, selects both themes by keyboard, runs with `reducedMotion: "reduce"`, and checks visible rendered controls for horizontal clipping. |
| Accessibility and visual safety | Axe reports zero serious or critical violations. Four screenshots were visually inspected and show current Account Export data, a safe empty external-search result, readable destructive controls, no stale success/error content, and no clipped controls. |

### Commands and results

Commands ran from the repository root unless the command begins with `cd frontend`. Repository-local Bun temporary and install directories were used.

| Command | Result |
| --- | --- |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/task272-frontend-gate.spec.ts` | PASS: 2/2 desktop/mobile cases. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck` | PASS: TypeScript emitted no errors. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage` | PASS: 533 tests, 0 failures, 2,789 expectations; `All files` is `95.46%` functions and `96.06%` lines. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | PASS: Vite transformed 219 modules and produced the production bundle. |
| `python3 scripts/validate-phase08-tsdoc.py` | PASS: every hand-written Phase 08 frontend export has adjacent concise TSDoc. |
| `python3 -m unittest scripts/test_generate_api_types.py` | PASS: 24 tests, 0 failures. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check:api-types` | PASS: generated API types are current. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/components/AdminVisualCompliance.test.ts src/lib/components/AdministrationPanel.test.ts src/lib/components/AdminDataManagement.test.ts src/lib/components/AdminPrivateData.test.ts src/lib/components/ExternalImportWorkflow.test.ts` | PASS: 21 tests, 0 failures, 426 expectations. |
| `cd frontend && MEALSWAPP_PLAYWRIGHT_REUSE_BUILD=1 BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/external-import-workflow.spec.ts tests/admin-private-data.spec.ts tests/admin-data-management.spec.ts tests/admin-access-shell.spec.ts tests/task272-frontend-gate.spec.ts` | PASS: 62/62 across desktop/mobile. Expected proxy-refusal logs come from older fixtures that deliberately leave unrelated panel requests unstubbed; no assertion or rendered safety state failed. |
| `python3 scripts/verify-frontend.py` | PASS: standard desktop/mobile DOM and scenario captures completed under `/tmp/mealswapp-frontend-verifier/`. The expected fixture 500/401 browser console responses remained contained and the verifier exited successfully. |
| `git diff --check -- frontend/tests/task272-frontend-gate.spec.ts docs/implementation/preparations/task-272.md` | PASS. |

## Precise coverage disposition

Task 272 does not broaden the existing Phase 08 frontend coverage exception in `docs/implementation/04_OPEN.md`.

The fresh measurement is exactly `All files | 95.46% funcs | 96.06% lines`. Relevant Phase 08 runtime rows remain:

| Runtime row | Functions | Lines | Uncovered lines | Existing disposition |
| --- | ---: | ---: | --- | --- |
| `src/lib/admin-access.ts` | 100.00% | 100.00% | none | fully covered |
| `src/lib/admin-workflows.ts` | 90.91% | 98.51% | Bun emits no stable line range | `F1` callback instrumentation; component/browser evidence |
| `src/lib/api/account-data-client.ts` | 100.00% | 98.00% | Bun emits no stable line range | `F1` callback instrumentation; client/browser evidence |
| `src/lib/api/admin-client.ts` | 97.22% | 100.00% | function instrumentation only | `F2` |
| `src/lib/api/external-admin-client.ts` | 100.00% | 100.00% | none | fully covered |
| `src/lib/api/filter-options-client.ts` | 100.00% | 100.00% | none | fully covered |
| `src/lib/api/generated.ts` | 100.00% | 98.98% | `185` | `F3` generated unreachable fallback; generator drift/client evidence |
| `src/lib/shell-routing.ts` | 100.00% | 100.00% | none | fully covered |
| `src/lib/substitution-filter-options.ts` | 100.00% | 100.00% | none | fully covered |

Svelte components do not emit Bun coverage rows. Their Task 272 disposition is the passing 21-test component inventory plus the 62-case desktop/mobile Playwright/axe matrix and production compilation. No external-import lifecycle, draft ownership, authoritative refresh, deletion recovery, generated-contract documentation, administration styling, keyboard, theme, reduced-motion, responsive, contrast, or axe behavior is waived.

## Screenshot evidence

Screenshots are temporary verification artifacts and were not added to the repository.

| Screenshot | Dimensions | SHA-256 |
| --- | --- | --- |
| `/tmp/mealswapp-task-272/task-272-desktop-chromium-light.png` | `1280 × 1988` | `085cd4154a4cbdf6640ee4421889291f1ee5556956a939401219b61da89fe15b` |
| `/tmp/mealswapp-task-272/task-272-desktop-chromium-dark.png` | `1280 × 1988` | `78f7a251fa598540ce7b219680f9ad0bfaf4d5dfdfbdc1b4fa5bd507505a9ff2` |
| `/tmp/mealswapp-task-272/task-272-mobile-chromium-light.png` | `1081 × 8891` device pixels | `e9d817d9966ef097e42a5dc4232cb900709ed8bc085a7241139b7b6d4bb07e5e` |
| `/tmp/mealswapp-task-272/task-272-mobile-chromium-dark.png` | `1081 × 8891` device pixels | `fad5163c9d7f3667f6ee7c04997c79b40978fb6caa4e24a98853903075d599f6` |

## Residual notes

- The Playwright browser fixtures verify the frontend boundary against deterministic representative responses; Task 271 owns live backend/PostgreSQL/Redis regression integration.
- The existing exact frontend coverage disposition remains machine-checked by the later aggregate Task 274 gate.
- No task status was changed and no other agent was contacted.
