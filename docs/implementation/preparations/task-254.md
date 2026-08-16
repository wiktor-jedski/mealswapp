# Task 254 preparation — Admin Frontend Access Shell

## Outcome

Task 254 (`DESIGN-009: UserAdminPanel`) is implemented without changing its task-list status. The Svelte shell now exposes `/admin` and an Administration navigation entry only after the startup session probe returns a verified `admin` role. Anonymous, standard-user, unverified, malformed, changed-account, and failed-session states remain fail-closed.

The Administration Panel is presentation-only in this task. It has feature-local loading and safe error boundaries plus a responsive shell for the later task 255/256 features. It performs no admin API calls and explicitly records that client visibility is not authorization; the backend remains authoritative for every `/api/v1/admin/*` request.

## Preparation baseline

- Fixed reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Initial worktree: dirty with extensive pre-existing Phase 08 backend, OpenAPI, generated-contract, design, implementation, script, preparation, and review changes. These were preserved.
- Task-owned frontend candidates were clean at baseline. `frontend/src/lib/api/generated.ts` was already modified by task 253 and `frontend/src/lib/api/generated.phase08-typecheck.ts` was already untracked; neither is task 254 work.
- Preparation evidence path: `docs/implementation/preparations/task-254.md` (absent at baseline).
- Writable preparation subagents were unavailable in this session, so the parent implemented the single requested task directly while retaining task-owned baseline and verification evidence.

### Baseline fingerprints

| Path | Baseline SHA-256 |
| --- | --- |
| `docs/implementation/02_TASK_LIST.md` | `e725a7ccf1a4e9365264befb4ef6cfe82637bdbc40386e63787b8742accccf3c` |
| `docs/design/DESIGN-009.md` | `85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b` |
| `frontend/src/lib/api/generated.ts` | `46c7a49e096ce66720b598c06bca76d4444c3e718940eabc31c48dbe8b1ac9c0` |
| `frontend/src/lib/components/SearchShell.svelte` | `9cb0a0bc9ac362583ba44cfdde330fcb095ff334bcdd0b99b44687f7ebed5513` |
| `frontend/src/lib/components/SearchShell.test.ts` | `e584c62991a5a583f9e8fb9e3ba73bad97be9d29ac6ebeb4faac3e871c0fe3af` |
| `frontend/src/lib/components/SidebarComponent.svelte` | `b4b946d2068ee268f2900d9519c054d8aadb78e0e43c08f3b5cad77e0366a745` |
| `frontend/src/lib/components/SidebarComponent.test.ts` | `e12314eba0ca884b049f86bea85a68e11c76b88928f16966f08aecd6d2c70a41` |
| `frontend/src/lib/shell-routing.ts` | `b640293003d0a79da155519e013e772b8e471eb4f0f7c25b37b126481462eef3` |
| `frontend/src/lib/shell-routing.test.ts` | `a217afbf675b705c0f44e71415f9f9826daf2a85257116d9acaef8c735e320fa` |
| New task files | Absent |

Dependencies 191 and 253 were both `PASSED` at selection. Task 254 remained `OPEN` throughout preparation.

## Task-owned surface and symbols

| Path | Added or modified surface |
| --- | --- |
| `frontend/src/lib/admin-access.ts` | Added `AdminAccessState`, `resolveAdminAccess`, and `verifiedAdminIdentity`. The decision requires authenticated status, a non-empty user ID, verified login method, and the server-projected `admin` role; unresolved and probe-error states use local boundaries. |
| `frontend/src/lib/admin-access.test.ts` | Added table-style store/decision tests for allowed, loading, denied, malformed, and error states. |
| `frontend/src/lib/components/AdministrationPanel.svelte` | Added the presentation-only responsive panel with loading, error, allowed-shell, and backend-authorization notice regions. No admin endpoint or later-task workflow is present. |
| `frontend/src/lib/components/AdministrationPanel.test.ts` | Added component-source assertions for local boundaries, responsive layout, task scope, backend-authorization messaging, and design traceability. |
| `frontend/src/lib/shell-routing.ts` | Extended `ShellView`, `parseShellRoute`, and `shellViewRoute` with canonical `/admin` routing. |
| `frontend/src/lib/shell-routing.test.ts` | Added `/admin` parse/generation assertions. |
| `frontend/src/lib/components/SidebarComponent.svelte` | Extended `Props` and `onSidebarNavigationSelect`; added a verified-admin-only Administration button with keyboard focus and active-page semantics. |
| `frontend/src/lib/components/SidebarComponent.test.ts` | Added static assertions for role/verification gating and callback wiring. |
| `frontend/src/lib/components/SearchShell.svelte` | Added administration access/identity/denial state, account-change reset effect, `openAdministrationView`, `denyAdministrationRoute`, popstate integration, local-boundary composition, and safe denial feedback. Search state is never reset. |
| `frontend/src/lib/components/SearchShell.test.ts` | Added composition assertions for fail-closed routing, identity reset wiring, denial feedback, and search-state preservation. |
| `frontend/tests/admin-access-shell.spec.ts` | Added Playwright coverage for verified admin navigation/panel, anonymous/user absence, direct denial, logout/account replacement, preserved search state, keyboard activation, responsive columns, light/dark themes, feature-local boundaries, and axe. |

No task 255 external search/import workflow, task 256 CRUD/user-management workflow, admin API client, backend code, OpenAPI contract, generated API type, architecture document, or task-list cell was changed.

## Verification criteria

| Criterion | Evidence |
| --- | --- |
| Verified admin navigation and panel | Browser test activates the Administration button with keyboard Enter and verifies canonical `/admin`, panel heading, shell regions, and server-authorization notice. |
| Anonymous and standard-user absence | Decision tests and browser tests prove no Administration nav or panel for either role. |
| Direct-route denial | `/admin` is replaced with the preserved Search route after anonymous/user resolution; no protected panel or controls render. |
| Logout and account reset | Browser test logs an admin out, signs in as a standard account, and proves the panel/nav remain cleared; identity tracking also rejects a changed admin identity. |
| Preserved non-admin search state | The same browser flow retains `preserved apples`; neither admin route function calls `resetSearch`. |
| Keyboard navigation | Browser test focuses the admin navigation entry and opens it with Enter; the control has the repository's visible focus ring. |
| Responsive layout | Browser test proves one panel column on mobile and three on desktop; shell uses `sm`/`lg` responsive grids without new fixed widths. |
| Light/dark themes and accessibility | Task-specific axe scans run in both themes for both browser projects with no serious or critical violations. |
| Feature-local loading/error | Delayed and failed profile probes render `data-admin-loading` then `data-admin-error` without exposing admin navigation. |
| Backend authorization remains authoritative | The component states this boundary; task 254 adds no admin API client and uses the role only for presentation and routing. |

## Commands and results

| Command | Result |
| --- | --- |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck` | PASS. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | PASS: 487 tests, 0 failures. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | PASS: 215 modules transformed. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage --coverage-reporter=text --coverage-dir=/tmp/mealswapp-task-254-repair-coverage src/lib/admin-access.test.ts src/lib/shell-routing.test.ts src/lib/components/AdministrationPanel.test.ts src/lib/components/SearchShell.test.ts src/lib/components/SidebarComponent.test.ts` | PASS: 50 tests; `admin-access.ts` and `shell-routing.ts` both report 100% line/function coverage. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/admin-access-shell.spec.ts` | PASS: 10 tests across desktop/mobile, including the malformed-session regression and four light/dark axe scans. |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e` | PASS: 262 scheduled tests with only the suite's intentional skips. Existing tests that intentionally leave API routes unstubbed emitted expected local-backend connection-refused proxy diagnostics but did not fail. |
| `python3 scripts/verify-frontend.py --artifact-dir /tmp/mealswapp-task-254-repair-frontend-verifier --screenshot-stem task-254-repair` | PASS; desktop/mobile evidence captured under `/tmp/mealswapp-task-254-repair-frontend-verifier/`. Expected anonymous-route 401 console messages were observed. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks with ordered dependencies. |
| `python3 scripts/generate-api-types.py --check` | PASS: generated API types current. |
| `git diff --check -- frontend/src/lib/components/SidebarComponent.svelte frontend/src/lib/components/SidebarComponent.test.ts frontend/tests/admin-access-shell.spec.ts docs/implementation/preparations/task-254.md` | PASS. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-254-review.md` | PASS: rejected review evidence remains structurally valid for independent re-review. |

## Rejected-review repair: F-254-001

The repair is intentionally limited to the rejected Sidebar access finding. `SidebarComponent` now imports and consumes the canonical `resolveAdminAccess` predicate instead of duplicating role and login-method checks. Therefore a visible Administration entry requires the resolver's complete `allowed` contract: authenticated status, non-empty user ID, admin role, verified login method, and no session error.

Exact changed symbols and test units:

| Path | Symbol or unit | Repair |
| --- | --- | --- |
| `frontend/src/lib/components/SidebarComponent.svelte` | `administrationAllowed` derived binding | Added the single fail-closed result `resolveAdminAccess($authSessionStore) === "allowed"`. |
| `frontend/src/lib/components/SidebarComponent.svelte` | Administration nav conditional/template | Replaced the duplicated role/verification expression with `{#if administrationAllowed}`; callback dispatch, active-page semantics, and mobile close behavior are unchanged. |
| `frontend/src/lib/components/SidebarComponent.test.ts` | `uses the centralized fail-closed predicate for malformed and error-bearing admin sessions` | Proves the Sidebar imports/calls `resolveAdminAccess`, gates on `administrationAllowed`, and contains no direct admin-role predicate. Together with `admin-access.test.ts`, this covers the authenticated+error projection rejected by review. |
| `frontend/tests/admin-access-shell.spec.ts` | `a malformed admin-shaped session exposes no administration control` | Supplies an authenticated admin-shaped refresh response without `userId` and proves the nav is absent in rendered desktop and mobile shells. |

No production symbol in `admin-access.ts`, `SearchShell.svelte`, routing, or the panel was changed by this repair. The task-list file was read for validation only and was not edited.

## Final fingerprints

| Path | SHA-256 |
| --- | --- |
| `frontend/src/lib/admin-access.ts` | `8e2a53aad61b2fedabf9f5ddb343360a20389bdb71b3af750e7e580dafe88aac` |
| `frontend/src/lib/admin-access.test.ts` | `366e41d9265c654e2284e12fd5d01cd1c459909a2b6f4c1509ef17c84e8e14ff` |
| `frontend/src/lib/components/AdministrationPanel.svelte` | `cd758ecf302e2b8d722b0be5bd82ac6cb6e457490a3759261bb620769bdd858f` |
| `frontend/src/lib/components/AdministrationPanel.test.ts` | `07dbc8d90fbf3d28429ab6acac6754f78ee29a981b5526446edf2facc64540a6` |
| `frontend/src/lib/components/SearchShell.svelte` | `f7bdfae6ec146f0db01136318d0c27bb07ca1fd287b66aacc620c850b103c7f3` |
| `frontend/src/lib/components/SearchShell.test.ts` | `88f065d461baa8f7a7a1b21730355801ed9d9da177c9d945dd84363b01a4b51a` |
| `frontend/src/lib/components/SidebarComponent.svelte` | `d24fd0959609b123550d62e4b309ec1948f1d40d9860ca977a6b2117217762ad` |
| `frontend/src/lib/components/SidebarComponent.test.ts` | `7f98738814bbdbe27a6a030f64c83c9cdecdce8f555cfc31d2b144bafd4e40f3` |
| `frontend/src/lib/shell-routing.ts` | `12964536dd4385381a9f75175daadfd2383723d672e484069bf48e923edf11f4` |
| `frontend/src/lib/shell-routing.test.ts` | `6e851caae71917d4d848352c9eed1aef0c80b29fd932d54987298bd123c7da23` |
| `frontend/tests/admin-access-shell.spec.ts` | `091437277211e1675e5b19afe833f33d0bde3f402fe07a66ad4f0dff6f22b0df` |
| `docs/implementation/02_TASK_LIST.md` | `541ae9c525d4c53fd8462c16cbb124783d37690943636800e572aeab9a1e1506` |
| `docs/design/DESIGN-009.md` | `85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b` |
| `frontend/src/lib/api/generated.ts` | `46c7a49e096ce66720b598c06bca76d4444c3e718940eabc31c48dbe8b1ac9c0` |

The design and generated-contract fingerprints remain unchanged from the preparation baseline. The shared task list changed elsewhere in the worktree and currently records task 254 as `PREPARED`; this repair did not edit it.
