# Task 148 Review — Phase 05 Offline and Stale Indicator

## Decision

**Recommended status: PASSED**

## Task Reviewed

| Field | Value |
| --- | --- |
| Task ID | 148 |
| Component | Phase 05 Offline and Stale Indicator |
| Static Aspect | DESIGN-001: OfflineBanner |
| Status (in task list) | PREPARED |
| Retries | 0 |
| Depends On | 141, 142 |
| Testing Coverage Exceptions | None |
| Description | Phase 05: subscribe to browser online/offline events and show cached-result, stale-result, and online-only fallback status without implementing the Phase 09 service-worker API cache policy. |

## Dependency Check

| Dep ID | Required | Found | Notes |
| --- | --- | --- | --- |
| 141 | PASSED | PASSED | `02_TASK_LIST.md:148` shows Task 141 PASSED. `frontend/src/lib/cache/local-query-cache.ts` exports `LocalQueryCache` with `isStale` consumed by `offline.ts:140` via `isStaleResult`, and exercised by `offline.test.ts:271-277`. |
| 142 | PASSED | PASSED | `02_TASK_LIST.md:149` shows Task 142 PASSED. `frontend/src/lib/api/search-client.ts` is the search client that Task 151 will wire to the offline setters; Task 148 only exposes `setShowingCached`/`setShowingStale`/`isStaleResult` as the seam and does not modify the client. |

## Verification Checklist

| ID | Criterion | Result | Evidence |
| --- | --- | --- | --- |
| C1 | Tests toggle online state and verify cached responses remain visible with offline/stale labels. | PASS | `offline.test.ts:127` dispatches `offline` and asserts `status.online === false`. `:138` then calls `setShowingCached(true)` and asserts `showingCached === true` while `online === false` (cached label state). `:152` calls `setShowingStale(true)` and asserts `showingStale === true` while offline (stale label state). `OfflineBanner.test.ts:23`/`:29` assert the component declares `"You're offline. Showing cached results."` and `"You're offline. Results may be stale."`. Implementation: `offline.ts:51-64` toggles `online` on browser events; `setShowingCached`/`setShowingStale` (`:103`/`:113`) update the store; `OfflineBanner.svelte:11-26` resolves the message from `online`/`showingCached`/`showingStale`. |
| C2 | Tests verify uncached offline requests show actionable feedback. | PASS | `offline.test.ts:166` dispatches offline with no cache flag set and asserts `online === false`, `showingCached === false`, `showingStale === false` (uncached state). `OfflineBanner.test.ts:33` asserts the component declares `"You're offline. Search is unavailable until you reconnect."`. Implementation: `OfflineBanner.svelte:25` returns the uncached actionable message when no flag is set. |
| C3 | Tests verify reconnection permits a fresh request. | PASS | `offline.test.ts:180` dispatches offline, sets both cached/stale flags, then dispatches online and asserts `online === true`, `showingCached === false`, `showingStale === false` — flags reset so a fresh request is permitted. Implementation: `offline.ts:51-58` `onlineHandler` resets `showingCached`/`showingStale` to `false` on reconnection. |
| C4 | Tests verify event listeners are removed on teardown (ref equality). | PASS | `offline.test.ts:199` captures the registered `online`/`offline` listeners from `FakeWindow.addEventListenerCalls`, calls `cleanupOffline()`, then asserts `removedOnline === registeredOnline` and `removedOffline === registeredOffline` via `toBe` (referential equality). `:219` confirms idempotency: a second `cleanupOffline()` records no further `removeEventListener` calls. Implementation: `offline.ts:35-36` stores handler refs in module-level `onlineHandler`/`offlineHandler`; `cleanupOffline` (`:83-95`) calls `removeEventListener` with the same refs and nulls them. |
| C5 | Tests do NOT claim Phase 09 service-worker interception coverage (disclaimer present). | PASS | `offline.test.ts:17-20` NOTE: "These tests verify Phase 05 online/offline event handling and cached/stale indicator state only. They do NOT claim Phase 09 service-worker API/image interception coverage, which remains Phase 09 scope per docs/implementation/04_OPEN.md ...". `OfflineBanner.test.ts:14-15` repeats the disclaimer. `04_OPEN.md:147` confirms the deferred scope: "Phase 09 remains responsible for service-worker API/image interception and broader offline hardening." |

Additional coverage beyond required criteria: SSR safety for both `initOffline` and `cleanupOffline` when `window` is undefined (`:232`/`:242`), `navigator.onLine` initial sync in both directions (`:249`/`:260`), default-status and initial-store-value tests (`:99`/`:108`), and `isStaleResult` delegation to `LocalQueryCache.isStale` without duplicating staleness logic (`:271`).

## Commands Run

| Command | Working dir | Exit | Result |
| --- | --- | --- | --- |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/stores/offline.test.ts src/lib/components/OfflineBanner.test.ts` | `frontend/` | 0 | 21 pass, 0 fail, 35 expect() calls across 2 files (15 offline + 6 OfflineBanner). |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | `frontend/` | 0 | 137 pass, 0 fail, 392 expect() calls across 14 files. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | `frontend/` | 0 | Vite build OK: 119 modules transformed, `dist/index.html` 0.51 kB, CSS 11.62 kB, JS 53.73 kB; built in 1.40 s. |
| `git -C /home/wiktor/Work/glm status --short` | repo root | 0 | Task-148 deliverables untracked: `frontend/src/lib/stores/offline.ts`, `frontend/src/lib/stores/offline.test.ts`, `frontend/src/lib/components/OfflineBanner.svelte`, `frontend/src/lib/components/OfflineBanner.test.ts`. `SearchShell.svelte` modified by earlier Task 145 wiring (no OfflineBanner import). No `dist/`, `node_modules/`, `.bun-tmp/`, `.bun-install/` staged (gitignored). |
| `python3 /home/wiktor/Work/glm/scripts/validate-task-list.py` | repo root | 0 | "Task-list validation passed: 154 sequential tasks with ordered dependencies." |
| `python3 /home/wiktor/Work/glm/scripts/validate-traceability.py` | repo root | 0 | "Traceability validation passed." |

## Files Inspected

| Path | Reason |
| --- | --- |
| `frontend/src/lib/stores/offline.ts` (141 lines, new) | Implementation: `OfflineStatus` interface (`:12`); `createInitialOfflineStatus` (`:24`); `offlineStatus` writable store (`:33`); module-level `onlineHandler`/`offlineHandler` refs (`:35-36`); `initOffline` (`:46`) subscribes to `online`/`offline`, syncs `navigator.onLine`, returns `cleanupOffline`, SSR-safe; `cleanupOffline` (`:83`) removes same refs and nulls them, idempotent, SSR-safe; `setShowingCached`/`setShowingStale` (`:103`/`:113`); `resetOfflineStatus` (`:123`); `isStaleResult` (`:135`) delegates to `LocalQueryCache.isStale` — no duplicated staleness logic. 10 traceability comments citing DESIGN-001 OfflineBanner / LocalStorageManager. |
| `frontend/src/lib/components/OfflineBanner.svelte` (42 lines, new) | Presentation component: subscribes to `$offlineStatus` (`:6-9`); `resolveOfflineBannerMessage` (`:11-26`) returns online/cached/stale/uncached messages; renders only when `!online` with `role="status"`, `aria-live="polite"`, `data-offline-banner`, `data-online`, `data-showing-cached`, `data-showing-stale` data attributes. 2 traceability comments. WCAG AA: polite live region, status role. |
| `frontend/src/lib/stores/offline.test.ts` (277 lines, 15 tests, new) | `FakeWindow` mock records `addEventListener`/`removeEventListener` calls and dispatches `online`/`offline` events (`:45-82`); `setWindow`/`setNavigator` helpers (`:84-96`); `afterEach` restores `window`/`navigator`. Covers C1-C5 plus SSR safety, navigator sync, idempotent cleanup, and `isStaleResult` delegation. 16 traceability comments. Phase 09 disclaimer at `:17-20`. |
| `frontend/src/lib/components/OfflineBanner.test.ts` (51 lines, 6 tests, new) | Source-based assertions (component cannot be rendered in Bun due to missing `svelte/server`/DOM libs; `vite build` validates the Svelte source). Asserts cached/stale/uncached messages, `role="status"`, `aria-live="polite"`, DESIGN-001 traceability comment, and `$offlineStatus` subscription. 8 traceability comments. Phase 09 disclaimer at `:14-15`. |
| `frontend/src/lib/components/SearchShell.svelte` | Confirms Task 148 does NOT wire `OfflineBanner` into the shell — no `OfflineBanner`/`offlineStatus`/`initOffline`/`cleanupOffline` imports or references. Shell changes are from earlier Task 145 wiring (SearchModes, SubstitutionInputs, DailyDietControls, SettingsPanel). Task 151 integration scope preserved. |
| `docs/implementation/02_TASK_LIST.md` | Task 148 row (`:155`) is `PREPARED`; deps 141 (`:148`) and 142 (`:149`) are `PASSED`. |
| `docs/implementation/04_OPEN.md` | `:147` confirms the Phase 09 deferred scope referenced by the test disclaimers: "Phase 09 remains responsible for service-worker API/image interception and broader offline hardening." |

## Coverage / Exception Review

- **Testing Coverage Exceptions:** None declared by the task.
- **Traceability comments:** 36 `Implements DESIGN-001 ...` comments across the four deliverable files (10 in `offline.ts`, 2 in `OfflineBanner.svelte`, 16 in `offline.test.ts`, 8 in `OfflineBanner.test.ts`), citing `OfflineBanner` and `LocalStorageManager` static aspects — meets AGENTS.md requirement.
- **TSDoc:** Every exported interface, function, and constant in `offline.ts` has TSDoc with `@remarks` tying it to DESIGN-001 — meets AGENTS.md requirement.
- **Generated-type discipline:** `offline.ts` imports `type { LocalQueryCache }` from `../cache/local-query-cache` (`:2`) and delegates to its `isStale` method. The only locally-defined type is `OfflineStatus` (a UI state shape), not an API contract type. No handwritten duplicates of generated types.
- **Scope discipline:** Implementation is a pure store + presentation component. It does not wire into `SearchShell.svelte` (Task 151), does not implement theme (Task 149), does not add responsive layout (Task 150), and does not implement service-worker interception (Phase 09). `setShowingCached`/`setShowingStale`/`isStaleResult` are the seams Task 151 will call from the search-client integration layer.
- **Simplicity:** Single writable store, two event handlers, one delegating helper. No speculative abstraction, no duplicated staleness logic (delegates to Task 141's `LocalQueryCache.isStale`).
- **Security:** No secrets, tokens, PII, or network calls. SSR-safe (window/navigator guards). Event listeners are removed on teardown to avoid leaks across HMR/test runs.
- **WCAG AA:** `role="status"` + `aria-live="polite"` announces offline state to assistive tech without interrupting; data attributes support integration-test selectors.
- **No generated artifacts staged:** `frontend/dist/`, `node_modules/`, `.bun-tmp/`, `.bun-install/` are gitignored. The only new Task 148 files are the four source files.
- **Test rendering limitation:** `OfflineBanner.test.ts` uses source-string assertions because Bun's isolated install-cache layout breaks transitive `svelte/server`/`svelte/compiler` resolution and no DOM library (jsdom/happy-dom) is installed. `vite build` compiles the component (119 modules, exit 0), validating the Svelte source. This is consistent with the approach used by other Phase 05 component tests (SettingsPanel, SearchModes, SubstitutionInputs, DailyDietControls) and is documented inline in the test file header.

## Failure Details

None. All five verification criteria (C1-C5) are satisfied with passing test evidence, implementation inspection, build success, and aggregate validators passing.
