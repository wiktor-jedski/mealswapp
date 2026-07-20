# Task 233 Preparation Evidence

## Scope and decision

- Task: **233 — Phase 07.01 Frontend Functional, End-to-End, and Accessibility Gate**.
- Design source: `docs/design/DESIGN-001.md`, static aspect `SearchView`.
- Repair source: `docs/implementation/reviews/task-233-review.md`, findings `F-233-01`, `F-233-02`, and `F-233-03` only.
- Task 233 remains **OPEN**. `docs/implementation/02_TASK_LIST.md`, task statuses, `docs/implementation/04_OPEN.md`, prior review evidence, and unrelated code were not edited for this repair.
- The shared worktree contains concurrent Phase 07.01 changes. They were preserved; the repair surface is limited to the SearchShell hydration owner, its focused tests, the Task 233 browser gate, the deterministic screenshot fixture/assertion, and this evidence file.

## Sources inspected

- Task 233 row in `docs/implementation/02_TASK_LIST.md`.
- `docs/implementation/reviews/task-233-review.md` and the prior complete `docs/implementation/preparation/task-233-preparation.md`.
- `CONTEXT.md`, `docs/agents/domain.md`, `docs/architecture/ARCH-001.md`, and `docs/design/DESIGN-001.md` in full.
- Current `SearchShell.svelte`, its source-level test, Task 233 Playwright gate, verifier/capture scripts, frontend package scripts, and Playwright configuration.

## Repaired findings and exact symbols

### F-233-01 — delayed Daily Diet hydration ownership

`frontend/src/lib/components/SearchShell.svelte`:

- `dailyDietSelectionGeneration` invalidates every prior hydration continuation.
- `dailyDietHydrationControllers` retains every active hydration request without suppressing legitimate concurrent meal selections.
- the mode lifecycle effect calls `clearIdentityOwnedDailyDietSelections` when a request, selection, or error leaves `daily_diet` mode;
- the identity `$effect.pre` continues to clear parent-owned Daily Diet state before the next identity loads;
- `hydrateDailyDietMeal` captures the initiating User Account ID, generation, mode owner, and `AbortController`;
- `dailyDietHydrationIsCurrent` guards both success and error continuations before state or query mutation;
- `clearIdentityOwnedDailyDietSelections` increments the generation, aborts all retained controllers, clears their set, and clears selection/error state on logout, User Account change, mode exit, or explicit lifecycle clear.

`frontend/src/lib/components/SearchShell.test.ts`:

- `cancels and generation-guards delayed Daily Diet hydration` locks the cancellation and identity/mode-generation guard wiring.

`frontend/tests/task233-frontend-gate.spec.ts`:

- `installGate` exposes deterministic `waitForDelayedHydration`, `releaseDelayedHydration`, and `delayedHydrationWasAborted` controls.
- `delayed Daily Diet hydration cannot cross logout and account change` holds User Account A's food-object response, logs out, authenticates User Account B, releases the response, verifies network abort, and verifies B's draft remains empty.
- `delayed Daily Diet hydration cannot cross a mode change` holds the same response across a switch to Catalog, verifies network abort, returns to Daily Diet, and verifies the draft remains empty.
- Both regressions execute in the desktop and mobile Playwright projects, producing four deterministic delayed-response executions.

The two delayed browser regressions were run before the production repair and both failed because `delayedHydrationWasAborted()` remained false. After the repair, both abort and empty-draft assertions pass.

### F-233-02 — deterministic screenshot fixture safety

`scripts/capture-frontend-scenarios.mjs`:

- `task233Diet` now contains exactly two distinct meals with distinct entries and positions.
- `task233CompletedJob` represents the corresponding two-meal completed alternative.
- `authSessionEnvelope` uses access/refresh expiries on 2027-07-18 and 2027-07-25; `entitlementEnvelope` uses an active trial expiring 2027-07-25. All are valid on the 2026-07-18 verification date.
- `assertTask233FixtureSafe` runs before any capture and again at each Task 233 safety check. It rejects mismatched fixture identity, malformed/expired session dates, inactive/expired trial entitlement, and any Daily Diet other than exactly two distinct meals.
- `assertTask233SafeState` retains unsafe-text and stale progress/error rejection and additionally requires exactly one rendered Task 233 Daily Diet summary showing `2 meals`.

The regenerated light desktop/mobile Daily Diet and dark desktop/mobile optimization screenshots were visually inspected. All four show `2 meals`; no stale progress/error or unsafe backend text is visible.

### F-233-03 — corrected provenance

The previous preparation's 1,991-expectation count is replaced by the current exact **438 tests / 1,998 expectations** result. The current maintained browser inventory does contain `frontend/tests/theme.spec.ts`; the exact current command below passes **75 tests with one intentional duplicate-screenshot skip** after adding the four delayed-response project executions. No absent file or obsolete 65/71-pass snapshot is claimed.

## Exact test inventory

The dedicated `frontend/tests/task233-frontend-gate.spec.ts` contains seven scenarios, each run in desktop and mobile projects:

1. lost create response, retry-stable write, replace, authoritative macros/selection, optimization, themes, axe, and layout;
2. malformed Daily Diet/optimization payload rejection and safe recovery;
3. queue ambiguity key reuse and terminal-timeout key rotation;
4. terminal infeasible guidance without stale alternatives;
5. optimization remount, logout, User Account change, and prior-result cleanup;
6. delayed selected-meal hydration across logout/User Account change;
7. delayed selected-meal hydration across mode exit.

The maintained related browser command also runs:

- `tests/daily-diet-workflow.spec.ts`
- `tests/optimization-workflow.spec.ts`
- `tests/phase07-browser-acceptance.spec.ts`
- `tests/accessibility.spec.ts`
- `tests/responsive.spec.ts`
- `tests/theme.spec.ts`

The sole skip is `accessibility.spec.ts`'s duplicate responsive screenshot execution in the second project; no Task 233 scenario is skipped.

## Commands and exact results

| Command | Exact result |
|---|---|
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck` | PASS. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage` | PASS — **438 tests, 1,998 expectations**, 0 failures; **94.01% functions, 94.86% lines**. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | PASS — Vite 7.3.3 transformed **205 modules** and emitted the production bundle. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/task233-frontend-gate.spec.ts --workers=2 --reporter=dot` | PASS — **14/14** desktop/mobile executions, 0 skipped. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/daily-diet-workflow.spec.ts tests/optimization-workflow.spec.ts tests/phase07-browser-acceptance.spec.ts tests/task233-frontend-gate.spec.ts tests/accessibility.spec.ts tests/responsive.spec.ts tests/theme.spec.ts --workers=2 --reporter=dot` | PASS — **75 passed, 1 intentional skip**, 0 failures. Expected proxy-noise from deliberately unstubbed anonymous probes did not fail assertions. |
| `python3 scripts/verify-frontend.py --artifact-dir /tmp/mealswapp-task-233-repair-final --screenshot-stem task-233-repair` | PASS — shell DOM, base desktop/mobile images, and **18 deterministic scenario captures**, including four hardened Task 233 images. |
| `node --check scripts/capture-frontend-scenarios.mjs` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS — 237 sequential tasks with ordered dependencies; Task 233 remains OPEN. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `git diff --check -- frontend/src/lib/components/SearchShell.svelte frontend/src/lib/components/SearchShell.test.ts frontend/tests/task233-frontend-gate.spec.ts scripts/capture-frontend-scenarios.mjs docs/implementation/preparation/task-233-preparation.md` | PASS. |

## Coverage disposition

The repair adds one source-level component test and four real-browser executions. Bun does not line-instrument `.svelte` files; the changed lifecycle is therefore covered at the real SearchShell/client/auth/mode seam by deterministic Playwright route control, plus typecheck and production compilation. Aggregate testable TypeScript remains 94.01% function and 94.86% line coverage. Task 235 owns the aggregate Phase 07.01 coverage disposition; this repair adds no exception and does not edit `docs/implementation/04_OPEN.md`.

## Screenshot evidence

| Scenario | Dimensions | SHA-256 |
|---|---:|---|
| Daily Diet light desktop | 1280×900 | `7c3acd2855fa14affcca76a6be2c2c54b4780441fd8292c063fa107372bbe2cd` |
| Daily Diet light mobile | 390×1019 | `17a590b4a3556b518c11282a4dbd857af680872311e28203490282d09b6fb8aa` |
| Optimization dark desktop | 1280×1203 | `a736384f0fb1c56940a7f0052b2d85c09e31a0468f6c96ac2bd63d1c9f72bf90` |
| Optimization dark mobile | 390×1454 | `60c1e110747dcaf06cb7e031e868e4de69257933f097d11dd05148c4b2e8836b` |

Artifact directory: `/tmp/mealswapp-task-233-repair-final/`. The generated verification artifacts remain outside the repository.

## Current fingerprints

| Path | Exact symbols/surface | SHA-256 |
|---|---|---|
| `docs/design/DESIGN-001.md` | `SearchView` source contract | `34d699ae93a8e5465199f3494ed41813c675f1cb3b9c1c6b6e611ba66c6142c7` |
| `docs/implementation/02_TASK_LIST.md` | read-only Task 233 row; OPEN | `ab4c293b379394fe573aaa1cd67d89a996a0a07e363c1b31752a1d220b0b3adb` |
| `frontend/src/lib/components/SearchShell.svelte` | generation/controller owner, `hydrateDailyDietMeal`, `dailyDietHydrationIsCurrent`, `clearIdentityOwnedDailyDietSelections` | `aa7a7e697445ff1dfcf54a2d6c75b54169e8680411f74279bcaa97db89545c81` |
| `frontend/src/lib/components/SearchShell.test.ts` | cancellation/generation source contract | `4d3d6b8b4960fa555e6a7a3f3f921977db13b074d884aba015928869ba2e74a7` |
| `frontend/tests/task233-frontend-gate.spec.ts` | delayed route owner and seven desktop/mobile scenarios | `9dd7c1f714b3ae6baa6528265b62b999adf487c9ec160c50275c51348961df1f` |
| `scripts/capture-frontend-scenarios.mjs` | two-meal fixture, current session/entitlement, fixture/render safety assertions | `538cbac2a2421820ddf9542beaec28822f9f3dd98070756676239fc1b2ec5e87` |

This preparation document intentionally does not self-hash.
