# Task 267 Preparation — Authoritative Account Export Refresh

## Result

Task 267 is implemented and practically verified. The Administration Panel now fails closed whenever an Account Export operation begins or fails: prior success feedback, pending deletion confirmation, stale private items, and item deletion controls are removed before network work. A DELETE accepted before its authoritative follow-up export fails receives verification-required feedback, while deletion success is shown only after both DELETE and export refresh succeed.

`docs/implementation/02_TASK_LIST.md` was not edited. Task 267 remains `OPEN`.

## Baseline and preservation

- Fixed baseline commit: `e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f`.
- Role: `/home/wiktor/.codex/agents/developer.toml`.
- Repository: `/home/wiktor/Work/mealswapp`.
- Task-list SHA-256 before and after preparation: `9b478ce8f10ead7c87a995b05ab10621fa5ad041e0229aed3a63e793fed08e02`.
- Baseline component SHA-256:
  - `frontend/src/lib/components/AdminPrivateData.svelte`: `0acdf79df7617030b228c8578bc02474b841857f8aeaa7363b41b38de5761719`.
  - `frontend/src/lib/components/AdminPrivateData.test.ts`: `eaac9091e7d27e4e238f401fb82d2df20e15f7bf926d5a91c6f549afe106595b`.
- Initial dirty paths were preserved: `api/openapi.yaml`, `frontend/src/lib/admin-workflows.ts`, `frontend/src/lib/api/admin-client.ts`, `frontend/src/lib/api/generated.ts`, `scripts/generate-api-types.py`, `scripts/test_generate_api_types.py`, and untracked `docs/implementation/preparations/task-270.md`.
- Concurrent work expanded while verification was running, including Task 266/268/269/270 files. No unrelated path was reverted or repaired. Concurrent visual Task 268 edits also landed in `AdminPrivateData.svelte`; its current hash therefore represents the safely combined file, while the executable symbols below identify Task 267 ownership precisely.

## Dependency and design inspection

- Task 239 preparation/review: authenticated owner-scoped custom-item deletion and owner-free Account Export custom-item projection. Its client boundary rejects `ownerId`.
- Task 256 preparation/review: `AdministrationPanel` composition, strict generated-client conventions, operation generations, abort handling, and authoritative follow-up patterns.
- `docs/design/DESIGN-008.md`: `DataExporter`, JSON Account Export, custom-item inclusion, retryable export failure, and retained account state.
- `docs/design/DESIGN-009.md`: `UserAdminPanel` restricted administration composition.
- `docs/requirements/01_SOFT_REQ_SPEC.md`: SW-REQ-043 private user data, SW-REQ-054 restricted Administration Panel, SW-REQ-071 machine-readable personal-data export, and SW-REQ-072 account/private-data deletion.
- Task row: `docs/implementation/02_TASK_LIST.md:274`.

## Exact changed paths

| Path | Task-267 responsibility | Final SHA-256 |
| --- | --- | --- |
| `frontend/src/lib/components/AdminPrivateData.svelte` | Added one fail-closed operation boundary, current-operation guards, precise post-delete verification failure, and success only after authoritative refresh. Concurrent Task 268 classes are preserved in the same file. | `18361a1fa0a42194e1032bcfc5ba700536bd58320cb2ac002bdbc2ec183328b1` |
| `frontend/src/lib/components/AdminPrivateData.test.ts` | Added component-source regression assertions for fail-closed clearing, operation guards, and precise verification feedback. | `a5c5ee38a527aa5afc49733b3e5daf3e5e208c72ddf60e6f8859055c9737cc02` |
| `frontend/tests/admin-private-data.spec.ts` | Added desktop/mobile browser scenarios for failed refresh after load, deletion accepted before refresh failure, successful retry, later complete deletion cycle, stale-success clearing, and owner-field rejection. | `15562390201e5c56e4d5084336acdd1adda88689c4de0bab3ce7e623383a8ac3` |
| `docs/implementation/preparations/task-267.md` | This preparation, verification, and risk record. | Self-referential hash omitted. |

No API, generated type, account-data client, backend, task-list, or dependency implementation was changed for Task 267.

## Added or modified executable symbols

### `frontend/src/lib/components/AdminPrivateData.svelte`

- Added reactive operation counter: `operation`.
- Modified lifecycle callback: `onDestroy` now invalidates the operation generation and aborts the active controller.
- Modified function: `refresh`.
- Modified function: `confirmDelete`.
- Added function: `beginOperation`.
- Added function: `isCurrent`.

`beginOperation` is the single transition into authoritative work. It aborts the previous operation, advances ownership, sets loading, clears error and success feedback, clears `pendingDelete`, and empties `items`. `refresh` and `confirmDelete` commit only when `isCurrent` confirms that their generation still owns state.

### `frontend/src/lib/components/AdminPrivateData.test.ts`

- Added test: `fails closed before every authoritative refresh and distinguishes deletion verification failure`.

### `frontend/tests/admin-private-data.spec.ts`

- Added constants/helpers: `firstItemId`, `secondItemId`, `ok`, `item`, `bundle`, `json`, `stubAdminShell`, and `openPrivateData`.
- Added browser test: `failed refresh clears loaded private objects and controls until an owner-free retry succeeds`.
- Added browser test: `accepted deletion reports verification-required failure and claims success only after a later complete cycle`.

## Criteria evidence

| Criterion | Evidence |
| --- | --- |
| Successful load followed by failed refresh | The first browser scenario loads a private item, opens its deletion confirmation, starts a delayed refresh, and proves the item, delete button, and confirmation disappear before the malformed export is released. |
| Fail closed during loading/failure | `beginOperation` clears `message`, `pendingDelete`, and `items` before either API call. Both browser scenarios assert no stale item or destructive control while loading and after failure. |
| Clear prior success feedback | The second browser scenario completes a deletion and authoritative refresh, then starts a delayed failing refresh and proves the prior success text disappears during loading and remains absent after failure. |
| Deletion accepted before refresh failure | The DELETE returns 204, the follow-up export returns 503, and the panel displays: “The private item was deleted, but current account data could not be verified. Refresh the export before continuing.” |
| Success only when both operations succeed | No deletion success appears after the accepted DELETE/failed export. A later independent deletion whose DELETE and export both succeed displays the success message and authoritative empty state. |
| Successful retry | Manual retry after failure restores the current remaining private item and its delete control, clears the alert, and does not resurrect the previous item. |
| No cross-user or owner-field leakage | A failed refresh returns a custom item carrying `ownerId` and the name `Foreign private secret`. The existing strict account-data client rejects the bundle; the browser proves neither the foreign name nor `ownerId` renders. The successful retry uses an owner-free current projection. |
| Stale/asynchronous operation containment | Every completion checks the captured generation and abort signal through `isCurrent`; teardown advances the generation before aborting. |

## Commands and results

Commands ran from the repository root unless a frontend working directory is stated.

1. `git rev-parse HEAD`, `git status --short`, repository/file searches, task-row search, role read, dependency preparation reads, design/requirements searches, and focused source inspection.
   - Confirmed baseline commit, initial concurrent edits, Task 267 scope, dependencies 239/256, and the existing stale-state defect.
2. From `frontend/`: `bun test src/lib/components/AdminPrivateData.test.ts`
   - First post-change run: 1 pass, 1 fail. The new source-order assertion accidentally matched the initial state declaration rather than the operation boundary; the assertion was corrected without production behavior changes.
   - Final focused result: 2 pass, 0 fail, 13 expectations.
3. From `frontend/`: `bun run typecheck`
   - PASS.
4. From `frontend/`: `bunx playwright test tests/admin-private-data.spec.ts --project=desktop-chromium`
   - PASS: 2/2.
5. From `frontend/`: `bun test src/lib/components/AdminPrivateData.test.ts src/lib/api/account-data-client.test.ts --coverage`
   - PASS: 4/4, 20 expectations.
   - `account-data-client.ts`: 100% functions, 98% lines. The source-only component test does not instrument compiled Svelte component lines; browser behavior supplies the component execution evidence.
6. From `frontend/`: `bunx playwright test tests/admin-private-data.spec.ts`
   - PASS: 4/4 across desktop Chromium and Pixel 5/mobile Chromium.
7. From `frontend/`: `bun run check`
   - Generated API type check PASS, TypeScript PASS, production build PASS.
   - Final unit phase: 525 pass, 2 fail, 2446 expectations. Both failures are pre-existing/concurrent Task 266 source assertions in `frontend/src/lib/components/ExternalImportWorkflow.test.ts`: they still expect `runSearch(page ± 1)` and the prior retry expression after concurrent implementation renamed those paths to `requestSearch(...)`. Task 267 files and focused tests pass; Task 266 files were intentionally not modified.
   - Build emitted two concurrent `ExternalImportWorkflow.svelte` warnings (`alertdialog` on a `section`, and non-reactive `keepEditingButton`); neither warning is in Task 267 code.
8. Final from `frontend/`: `MEALSWAPP_PLAYWRIGHT_REUSE_BUILD=1 bunx playwright test tests/admin-private-data.spec.ts`
   - PASS: 4/4 across desktop and mobile after adding explicit post-success failure coverage.
9. `python3 scripts/validate-traceability.py`
   - PASS: `Traceability validation passed.`
10. `python3 scripts/validate-task-list.py`
    - PASS: `275 sequential tasks with ordered dependencies.`
11. `git diff --check`
    - PASS.
12. `rg -n '^\| 267 \|' docs/implementation/02_TASK_LIST.md`
    - Task 267 remains `OPEN`; no status edit was made.

The commands used repository-local Bun temporary/install directories as required by `AGENTS.md`.

## Risks and handoff notes

- The account export client validates only the private custom-item fields needed by this panel (`id`, bounded nonblank `name`, and absence of `ownerId`) while checking top-level export collections. Task 267 relies on that Task 239 boundary and does not broaden export decoding.
- `AdminPrivateData.svelte` became a shared concurrent path because Task 268 added visual classes while this preparation was active. The merged file typechecks and focused browser tests pass. Review should attribute only the Task 267 symbols listed above.
- The full frontend gate is not green because concurrent Task 266 source assertions lag its implementation. Focused Task 267 component/client/browser checks, generated-contract check, typecheck, build, traceability, task-list validation, and whitespace validation pass.
- No real backend browser stack was required for this frontend state-machine repair. Existing Task 261 real-stack coverage already verifies the successful generated-client deletion/export path; Task 267 adds deterministic failure and recovery coverage with browser-controlled transport responses.
