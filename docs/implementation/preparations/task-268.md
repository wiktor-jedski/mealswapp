# Task 268 preparation — Administration Visual and Accessibility Compliance

## Outcome

Task 268 is prepared against fixed baseline `e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f` using the `developer.toml` role. The implementation is limited to the Phase 08 administration styling remediation and its focused source/contrast test. `docs/implementation/02_TASK_LIST.md` was not edited.

The implementation:

- uses `--color-on-error` for both filled destructive administration actions;
- gives every button in `AdminDataManagement.svelte`, `AdminPrivateData.svelte`, and `ExternalImportWorkflow.svelte` `transition-all duration-200 motion-reduce:transition-none`;
- gives every `input`, `select`, and `textarea` in `AdminDataManagement.svelte` and `ExternalImportWorkflow.svelte` the theme Surface background, one-pixel theme Border, and two-pixel Primary focus ring;
- changes only `h1`–`h3` administration headings from Semibold 600 to Bold 700, retaining Semibold on legends, status labels, buttons, and emphasized body text;
- adds source-level inventory assertions and light/dark WCAG contrast calculation.

## Dependency and design inspection

Dependencies 254, 255, and 256 are `PASSED` in the unchanged task list. Their preparation records and the controlling design/style documents were inspected:

| Input | SHA-256 at preparation |
| --- | --- |
| `docs/implementation/preparations/task-254.md` | `0294650a8e6bd07239a3fd7b2da1d6a6a2f9a5d1216b6975a28281b9c2ec4046` |
| `docs/implementation/preparations/task-255.md` | `c6243d5b152e6ca8a8afa0d1aed2ba8c50d6dc3bb571ae743af82300ba0c0112` |
| `docs/implementation/preparations/task-256.md` | `ddd6510debbd3a7a75cdaefb2d4bda19248e80618ed28b594e900db9f08858e0` |
| `docs/design/DESIGN-009.md` | `85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b` |
| `docs/requirements/02_STYLE_GUIDE.md` | `b397e3de590588ca9c7d84dddec5577555202eeb93b1c1c45356fe16d58f3620` |
| `frontend/src/app.css` | `8d9b42303e60351869a482e245a26f80b1f68d30fcbaa020e48d348f5bf73806` |

The already-defined `--color-on-error` values were retained: light `#ffffff` on Error `#dc2626`; dark `#111827` on Error `#f87171`.

## Preservation and baseline

- Fixed baseline: `e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f`.
- The preparation started with pre-existing concurrent edits in `api/openapi.yaml`, `frontend/src/lib/admin-workflows.ts`, `frontend/src/lib/api/admin-client.ts`, `frontend/src/lib/api/generated.ts`, `scripts/generate-api-types.py`, and `scripts/test_generate_api_types.py`. They were not reverted or edited for Task 268.
- Additional concurrent Phase 08 work appeared while this preparation ran, including Task 266 changes in `ExternalImportWorkflow.svelte` and Task 267 changes in `AdminPrivateData.svelte`. Task 268 classes were applied to the resulting shared markup without reverting those behavioral changes.
- No backend file, API contract, generated file, script, task status, or existing preparation record was edited for Task 268.
- No commit was made because the task-owned markup is interleaved in shared files with concurrent Task 266/267 work; committing those whole files would improperly include work owned by other preparations.

## Exact Task 268 changed paths

| Path | Task 268 change | SHA-256 at final verification |
| --- | --- | --- |
| `frontend/src/lib/components/AdministrationPanel.svelte` | Changed six heading classes from `font-semibold` to `font-bold`. | `35f27a688b9e54dd59437b9107e38bf432c053029653a9c2b9f5fdf301a7ccb5` |
| `frontend/src/lib/components/AdminDataManagement.svelte` | Standardized 12 buttons, 20 form controls, six headings, and the filled destructive foreground token. | `375bf03b2ca4d8ba48020a082259e22c50c8cb1ba8e0ebc9f23d698f5bafe0ea` |
| `frontend/src/lib/components/AdminPrivateData.svelte` | Standardized four buttons, changed one heading to Bold, and retained the dedicated destructive foreground token. This shared file also contains preserved concurrent Task 267 behavior. | `18361a1fa0a42194e1032bcfc5ba700536bd58320cb2ac002bdbc2ec183328b1` |
| `frontend/src/lib/components/ExternalImportWorkflow.svelte` | Standardized 16 buttons, 12 form controls, and five headings. This shared file also contains preserved concurrent Task 266 behavior. | `446e21a4eed864b55123ae2b12833ea0366e72592455cf4cf21f27625829efbd` |
| `frontend/src/lib/components/AdminVisualCompliance.test.ts` | Added complete source inventory and semantic-token contrast assertions. | `424fdbc134f7567a10ad616d6a475dd73f37d1b13d4386369344eae0e47042a1` |
| `docs/implementation/preparations/task-268.md` | Added this preparation, verification, and risk record. | Recorded by the parent after concurrent work is quiescent. |

The unchanged task list has SHA-256 `9b478ce8f10ead7c87a995b05ab10621fa5ad041e0229aed3a63e793fed08e02`, identical to the fixed baseline.

## Added or modified executable symbols

No production TypeScript/Svelte executable symbol was added or modified by Task 268; production changes are static component class attributes only. Concurrent behavioral symbols in the two shared components are outside Task 268.

The new test module adds:

| Symbol | Kind | Purpose |
| --- | --- | --- |
| `components` | module constant | Loads the four administration component sources. |
| `openings` | function | Inventories opening tags for a requested element pattern. |
| `expectClasses` | function | Requires exact Tailwind classes on each inventoried element. |
| `themeToken` | function | Reads a semantic color token from the selected CSS theme block. |
| `contrastRatio` | function | Calculates WCAG relative-luminance contrast. |
| `every Phase 08 administration button uses the standard reduced-motion transition` | test callback | Proves the complete three-component button inventory. |
| `every affected administration form control uses Surface, Border, and a two-pixel Primary focus ring` | test callback | Proves the complete two-component form-control inventory. |
| `administration headings alone use Bold 700 without mechanically changing other emphasized text` | test callback | Proves all headings and protects non-heading Semibold examples. |
| `filled destructive buttons use the dedicated accessible error foreground` | test callback | Proves both filled Error controls use `--color-on-error`, never `--color-on-muted`. |
| `error foreground contrast is WCAG AA in light and dark themes` | test callback | Measures both actual CSS theme token pairs against `4.5:1`. |

## Criteria evidence

| Criterion | Evidence |
| --- | --- |
| Dedicated accessible error foreground | Source assertion inventories both filled destructive buttons. `AdminDataManagement.svelte` was corrected from `--color-on-muted` to `--color-on-error`; `AdminPrivateData.svelte` retains `--color-on-error`. |
| Standard button transition and reduced-motion opt-out | Source assertion inventories 12 + 4 + 16 = 32 buttons; every opening has all three required classes. Browser tests run under `reducedMotion: "reduce"` and pass. |
| Theme Surface, Border, and Primary form styling | Source assertion inventories 20 + 12 = 32 controls, including checkboxes and textarea; every control has `border`, `border-[var(--color-border)]`, `bg-[var(--color-surface)]`, `focus:ring-2`, and `focus:ring-[var(--color-primary)]`. |
| Bold 700 only headings | Source assertion inventories 6 + 6 + 1 + 5 = 18 headings; each has `font-bold` and none has `font-semibold`. Guard assertions retain Semibold on the `Food categories` legend and `Partial results` status label. |
| Contrast at least `4.5:1` | Calculated from actual `app.css` token declarations: light `4.83:1`; dark `6.41:1`. Both pass the focused test threshold. |
| Keyboard and hidden focus | Desktop/mobile Playwright confirmation flow proves focus entry, Tab/Shift+Tab containment, Enter cancellation, and restoration to opener or safe fallback. Existing focus-ring classes and source assertions cover all affected controls. |
| Mobile and clipping | Both configured desktop and mobile Chromium projects pass responsive administration tests; expected grid columns are asserted. No assertion or axe failure reported clipping or hidden controls. |
| Light/dark theme | Administration access and data-management browser tests switch through both themes and pass their layout and axe checks. Form controls resolve through theme tokens rather than page background or transparency. |
| Reduced motion | Required `motion-reduce:transition-none` is asserted on every target button; administration browser suites pass with reduced-motion emulation active. |
| axe | Light/dark administration browser scans report zero serious/critical violations in desktop and mobile projects. |

## Commands and results

| Command | Result |
| --- | --- |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/components/AdminVisualCompliance.test.ts src/lib/components/AdministrationPanel.test.ts src/lib/components/AdminDataManagement.test.ts src/lib/components/AdminPrivateData.test.ts src/lib/components/ExternalImportWorkflow.test.ts` | **PASS:** 21 tests, 0 failures, 421 expectations. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck` | **PASS:** `tsc -p tsconfig.typecheck.json --noEmit`. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/admin-data-management.spec.ts tests/admin-access-shell.spec.ts` | **PASS:** 28/28 across desktop and mobile Chromium; keyboard, responsive columns, light/dark, reduced motion, and axe covered. Expected proxy-refusal logs came from deliberately unstubbed account-export/classification requests in unrelated shell fixtures; assertions all passed. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/external-import-workflow.spec.ts tests/admin-private-data.spec.ts` | **PASS:** 30/30 across desktop and mobile Chromium; shared Task 266/267 behavior remained intact after styling. Expected unrelated account-export proxy-refusal logs did not affect assertions. |
| WCAG token calculation (`#ffffff`/`#dc2626`, `#111827`/`#f87171`) | **PASS:** light `4.83:1`; dark `6.41:1`. |
| `git diff --check` | **PASS:** no whitespace errors. |

The Playwright web server performed a successful Vite production build before each browser run.

## Risks and handoff

- `AdminPrivateData.svelte` and `ExternalImportWorkflow.svelte` are shared with concurrent Tasks 267 and 266. Their final hashes may change if those preparations continue; rerun `AdminVisualCompliance.test.ts` after any merge or rewrite of either component.
- The browser suites prove rendered keyboard/mobile/theme/reduced-motion/axe behavior but do not perform screenshot pixel comparison. Task 272 remains the full frontend regression and screenshot gate.
- `--color-on-error` was already present in `app.css`; Task 268 validates and consumes it rather than duplicating or changing the approved token.
- Task-list status remains intentionally unchanged.
