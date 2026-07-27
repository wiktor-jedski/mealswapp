# Task 266 preparation — External Import Lifecycle and Draft Boundary

## Outcome and baseline

Task 266 is prepared against the fixed baseline commit `e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f`. Dependency 255 was inspected in `docs/implementation/02_TASK_LIST.md` and `docs/implementation/preparations/task-255.md`; it is `PASSED` and supplies the external search, curation, conflict, and retry-stable import workflow extended here.

`docs/implementation/02_TASK_LIST.md` was not edited. The six restored concurrent paths named at delegation were preserved. Other task preparations expanded the shared worktree while Task 266 was running; those changes were not reverted. In particular, concurrent Task 268 styling changes overlap `ExternalImportWorkflow.svelte` and remain intact beside the Task 266 lifecycle changes.

## Exact changed paths

- `frontend/src/lib/components/ExternalImportWorkflow.svelte`
- `frontend/src/lib/components/ExternalImportWorkflow.test.ts`
- `frontend/tests/external-import-workflow.spec.ts`
- `docs/implementation/preparations/task-266.md`

No API, generated contract, backend, task-list, or dependency implementation was changed for Task 266.

## Added and modified executable symbols

### `frontend/src/lib/components/ExternalImportWorkflow.svelte`

| Symbol or executable unit | Change |
| --- | --- |
| `onMount` callback | Invalidates import ownership on teardown in addition to aborting provider search. |
| `requestSearch` | Captures an immutable query/provider/page request, blocks search while importing, and opens the unsaved-draft boundary when curation owns the workflow. A draft-free completed workflow resets directly without a warning. |
| `runSearch` | Consumes the captured `SearchRequest` instead of reading mutable query/provider/page state after the boundary decision. |
| `selectCandidate` | Invalidates any older import token before installing the candidate, draft, and new idempotency key. |
| `keepEditing` | Cancels the pending search without changing the draft or key and restores keyboard focus to the draft name. |
| `handleDraftBoundaryKeydown` | Cancels on Escape and traps forward/reverse Tab navigation within the two modal decisions. |
| `discardDraftAndSearch` | Captures the pending search, clears all curation/import state through `resetCuration`, invalidates old ownership, and starts the requested provider search. |
| `submitImport` | Rejects duplicate activation, allocates a monotonic ownership token, snapshots the full nested draft and idempotency key before asynchronous work, and commits success/conflict/ambiguity/failure only while its token still owns the workflow. |
| `snapshotDraft` | Copies the draft, macro/micronutrient maps, classification arrays, and conflict-confirmation decision into one immutable attempt value. |
| `invalidateImportOwnership` | Monotonically advances the owner counter so an older completion cannot commit. |
| `resetCompletedDraft` | Clears draft-only state and the spent key after success while retaining the completed result for local-search handoff. |
| `resetCuration` | Invalidates ownership and clears pending search, draft, warnings, key, confirmation/conflict state, message, and result. |
| External-search form/retry/pagination/candidate handlers | Route searches through `requestSearch`; search, provider, pagination, and candidate controls are disabled while importing. |
| Draft-boundary key/click handlers | Expose labelled `alertdialog` semantics, focus the safe “Keep editing” action, support Escape, and provide keyboard-native keep/discard actions. |
| Curation form submit/editing controls | Disable the complete editable fieldset and submit action while importing. |
| Provider-conflict refresh handler | Uses the same explicit draft-to-search boundary as every other provider search. |

`SearchRequest`, `pendingSearch`, `importOwnershipToken`, `keepEditingButton`, `discardSearchButton`, and `draftNameInput` are added state/type units rather than executable symbols; they support the executable units above.

### `frontend/src/lib/components/ExternalImportWorkflow.test.ts`

| Test callback | Change |
| --- | --- |
| `covers provider selection, pagination, and all safe external states` | Updated pagination assertions for the guarded `requestSearch` boundary. |
| `keeps one idempotency key through conflict and ambiguous retry paths` | Verifies the immutable draft/key snapshot call. |
| `owns deferred imports and explicit draft-to-search transitions` | Added source-level contract checks for monotonic ownership, stale-outcome guards, resets, labelled keep/discard actions, and disabled controls. |

### `frontend/tests/external-import-workflow.spec.ts`

| Symbol or test callback | Change |
| --- | --- |
| `lifecycleEnvelope` | Added deterministic old/current/replacement candidate fixtures for lifecycle races. |
| `ignores a superseded import {success, conflict, ambiguity, failure} without overwriting the active draft` matrix callback | Defers each outcome, proves candidate selection is disabled, deliberately bypasses that presentation guard to exercise defensive ownership invalidation, and verifies the newer draft/result/message/key-owning state survives. |
| `disables incompatible controls during import and starts a completed workflow without a draft warning` | Verifies search/provider/pagination/candidate/draft/classification/import controls are disabled, success clears the spent draft, and the next provider search starts directly and clears the old result. |
| `keeps or discards an unsaved draft explicitly and invalidates discarded import ownership` | Verifies Escape and Enter keyboard flows, safe initial focus and focus return, exact draft retention, no search on keep, complete curation reset on discard, stale-success suppression after discard, and zero serious/critical axe violations. |

## Criteria evidence

| Task criterion | Evidence |
| --- | --- |
| Monotonic ownership and immutable snapshots | `submitImport` increments `importOwnershipToken`, captures `snapshotDraft(...)` and `keySnapshot`, and checks ownership before both success and failure commits. Candidate selection, discard/reset, and teardown invalidate old ownership. |
| Superseded success/conflict/ambiguity/failure cannot overwrite active state | The four-outcome Playwright matrix passes on desktop and mobile and retains the current candidate draft with no stale result, conflict, blocked conflict, error, or message. |
| Incompatible controls disabled while importing | Browser assertions cover search text, provider, submit, pagination, candidate selection, draft name/state/density, classification, and import submit controls. |
| Explicit keep/discard before a new provider search | `requestSearch` opens a labelled boundary only for an active draft. “Keep editing” performs no search and preserves the edited draft. “Discard draft and search” resets curation/import ownership before using the captured search request. |
| Discard invalidates the token | The discard browser test starts a deferred import behind the boundary, discards, completes the new search, then releases the old success and proves it cannot install a result or remove the replacement search state. |
| Completed import resets cleanly | Success clears draft/key state but keeps the result handoff; the next search opens no discard warning and removes the completed result. |
| Ambiguous retry retains the correct key | Existing browser coverage still proves two ambiguous/replay requests use the same non-empty key; the new snapshot path passes that key by value. |
| Keyboard and screen-reader flows | The boundary has `alertdialog`, `aria-modal`, labelled/described relationships, deterministic safe-action focus, forward/reverse Tab containment, Escape cancellation, Enter activation, draft focus restoration, and passing desktop/mobile axe checks. |

## Verification commands and results

| Command | Result |
| --- | --- |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/components/ExternalImportWorkflow.test.ts` | PASS after the final focus-containment change: 5 tests, 0 failures, 56 expectations. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/external-import-workflow.spec.ts --project=desktop-chromium` | Initial run: 12 passed and 1 failed because the lifecycle fixture inherited the pre-existing accent-text contrast issue assigned to Task 268. The unrelated warning fixture was removed; no lifecycle assertion failed. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/external-import-workflow.spec.ts` | PASS: 26 tests across desktop and mobile, including four superseded outcome variants, locking, clean completion, keep/discard, ambiguous retry, keyboard, and axe. Expected unstubbed Account Export proxy failures were non-fatal. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/external-import-workflow.spec.ts -g 'keeps or discards'` | PASS after final focus/tabindex/Tab-containment repair: 2 desktop/mobile tests. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck` | PASS after the final focus-containment change. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check` | PASS: generated API drift check, TypeScript typecheck, production build, and 533 tests with 2,784 expectations. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS: 275 sequential tasks with ordered dependencies; no status edit. |
| `git diff --check -- frontend/src/lib/components/ExternalImportWorkflow.svelte frontend/src/lib/components/ExternalImportWorkflow.test.ts frontend/tests/external-import-workflow.spec.ts` | PASS. |

## Reviewer repair — modal draft boundary

The blocking finding in `docs/implementation/reviews/task-266-review.md` was repaired without changing the task list or any non-Task-266 implementation path.

### Repair paths

- `frontend/src/lib/components/ExternalImportWorkflow.svelte`
- `frontend/src/lib/components/ExternalImportWorkflow.test.ts`
- `frontend/tests/external-import-workflow.spec.ts`
- `docs/implementation/preparations/task-266.md`

### Repair symbols and behavior

| Symbol or surface | Repair |
| --- | --- |
| `draftBoundaryOpener` / `requestSearch` | Captures the control that opened the draft boundary so the discard path can restore focus after the modal closes. |
| `handleDraftBoundaryKeydown` | Prevents native Escape handling from racing the explicit keep-editing transition while retaining forward/reverse Tab containment. |
| `discardDraftAndSearch` | Restores focus to the connected opener after the inert background is released and the replacement search starts. |
| `openDraftBoundary` | Opens the boundary with native `HTMLDialogElement.showModal()`, handles native cancel, contains Tab, recovers pointer/programmatic focus to the safe Keep action through a document `focusin` listener, removes listeners on teardown, and closes the dialog if still open. |
| Draft-boundary markup | Moves the decision into a native `<dialog role="alertdialog" aria-modal="true">`, adds a modal backdrop, and wraps every background workflow control in `data-draft-search-background` with `inert` while `pendingSearch` exists. |
| `owns deferred imports and explicit draft-to-search transitions` | Adds source-contract assertions for the native modal action, `showModal`, focus recovery, and conditional inert background. |
| `keeps or discards an unsaved draft explicitly and invalidates discarded import ownership` | Adds desktop/mobile regression coverage that attempts to activate a background Curate control, proves native modal interception, preserves the edited draft and boundary, attempts focus escape to the inert search input, and proves focus remains on the safe action. |

### Repair verification

| Command | Result |
| --- | --- |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/external-import-workflow.spec.ts -g 'keeps or discards' --project=desktop-chromium` | RED before repair: failed because `data-draft-search-background`/`inert` did not exist, reproducing the reviewer finding. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/external-import-workflow.spec.ts -g 'keeps or discards'` | PASS after repair: 2/2 desktop/mobile cases. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/components/ExternalImportWorkflow.test.ts` | PASS: 5 tests, 0 failures, 60 expectations. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/external-import-workflow.spec.ts` | PASS: 26/26 desktop/mobile cases. Expected unstubbed Account Export proxy failures remained non-fatal. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check` | PASS: generated API drift, TypeScript typecheck, production build, and 533 tests with 2,789 expectations. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS: 275 sequential tasks with ordered dependencies; the concurrent task-list changes were not edited. |
| `git diff --check -- frontend/src/lib/components/ExternalImportWorkflow.svelte frontend/src/lib/components/ExternalImportWorkflow.test.ts frontend/tests/external-import-workflow.spec.ts docs/implementation/preparations/task-266.md` | PASS. |

## Reviewer repair — post-discard focus restoration

The important pagination/disappearing-opener finding in `docs/implementation/reviews/task-266-review.md` was repaired using the simplicity and native-platform guidance from `/home/wiktor/.codex/agents/developer.toml`. `docs/implementation/02_TASK_LIST.md`, the review artifact, and unrelated implementation were not edited.

### Exact repair paths

- `frontend/src/lib/components/ExternalImportWorkflow.svelte`
- `frontend/tests/external-import-workflow.spec.ts`
- `docs/implementation/preparations/task-266.md`

### Exact repair symbols and behavior

| Symbol or executable unit | Repair |
| --- | --- |
| `searchInput` | Binds the stable external-search input as the enabled in-context fallback when a transient opener cannot receive focus. |
| `discardDraftAndSearch` | Waits for the modal/inert state and replacement-search rendering to settle, then delegates focus restoration instead of focusing the captured opener unconditionally. |
| `restoreDraftBoundaryFocus` | Preserves a connected enabled opener when possible; skips disabled, `aria-disabled`, disconnected, and inert candidates; verifies actual focus movement; and falls back to `searchInput`. |
| External food search input | Adds `bind:this={searchInput}` without changing search or disabled-state behavior. |
| `restores visible focus after discarding from pagination or a disappearing refresh action` | Adds the focused Playwright regression. It opens the boundary from Next, proves discard returns focus to the stable search input after pagination disables the opener, then opens from provider-conflict refresh and proves the same result after that opener is removed. The existing desktop/mobile projects execute both paths. |

### Exact repair commands and results

| Command | Result |
| --- | --- |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/external-import-workflow.spec.ts -g 'restores visible focus' --project=desktop-chromium` | RED before production repair: 1 failed at the pagination `toBeFocused` assertion after Page 2 rendered; the search input was inactive, reproducing the reviewer finding. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/external-import-workflow.spec.ts -g 'restores visible focus'` | PASS after repair: 2/2 desktop/mobile cases, including pagination and disappearing provider-refresh openers. Expected unstubbed Account Export proxy failures were non-fatal. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/components/ExternalImportWorkflow.test.ts` | PASS: 5 tests, 0 failures, 60 expectations. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/external-import-workflow.spec.ts` | PASS: 28/28 desktop/mobile cases. Expected unstubbed Account Export proxy failures were non-fatal. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check` | PASS: generated API drift check, TypeScript typecheck, production build, and 533 tests with 2,789 expectations. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS: 275 sequential tasks with ordered dependencies; no task-list edit. |
| `git diff --check -- frontend/src/lib/components/ExternalImportWorkflow.svelte frontend/src/lib/components/ExternalImportWorkflow.test.ts frontend/tests/external-import-workflow.spec.ts docs/implementation/preparations/task-266.md` | PASS after the final preparation-evidence update. |

## Risks

- `ExternalImportWorkflow.svelte` is shared with concurrent Task 268 styling work. Both changes currently coexist and pass the frontend gate, but later reconciliation must preserve the lifecycle handlers, disabled bindings, focus bindings, and ownership checks listed above.
- The supersession matrix intentionally removes a disabled attribute in the browser before invoking candidate selection. This is not a user flow; it proves the defensive token remains correct even if a future caller or regression bypasses presentation-level locking.
- Imports are not transport-aborted on candidate invalidation or discard because the existing client call has no component-owned import controller. The monotonic token prevents stale UI commits, while the backend idempotency contract safely handles a request that already reached the server.
