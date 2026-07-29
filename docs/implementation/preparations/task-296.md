# Task 296 Preparation Evidence

Task 296, Phase 08.02 Authoritative Unit Preference, is prepared on the
isolated `task-296-authoritative-unit-preference` branch. Dependency Task 295
is `PASSED` on the synchronized phase branch.

## Owned implementation

- `frontend/src/lib/stores/preferences.ts` now separates anonymous device
  preference from authenticated profile authority, cancels stale account
  operations, applies confirmed server values only, and exposes recoverable
  load/save state.
- `frontend/src/lib/stores/auth-session.ts` hydrates the preference after
  profile/session authentication and restores the anonymous device value on
  sign-out or expired/anonymous transitions.
- `frontend/src/lib/api/auth-client.ts`, the generated API contract, and the
  generator add the CSRF-protected `PUT /api/v1/profile` client path.
- `frontend/src/lib/components/SidebarComponent.svelte` keeps the confirmed
  value visible during saves and exposes accessible loading, error, and retry
  states.
- Component, store, generated-client, auth-session, focused workflow, and
  desktop/mobile Playwright regressions cover persistence, hydration,
  confirmation, account switching, sign-out restoration, cancellation, and
  recoverable failures.

## Verification

| Command | Result |
| --- | --- |
| `bun test src/lib/stores/preferences.test.ts --coverage` | PASS; `23` tests, `preferences.ts` `100.00%` functions / `100.00%` lines |
| `bun run check` | PASS; `549` frontend tests, typecheck, generated drift, and production build |
| `bunx playwright test tests/task296-unit-preference.spec.ts` | PASS; `8/8` desktop/mobile cases |
| changed-area Playwright lane | PASS; `62/62` cases |
| `python3 -m unittest scripts/test_generate_api_types.py` | PASS; `25/25` |
| `python3 scripts/validate-traceability.py` | PASS |
| `python3 scripts/validate-task-list.py` | PASS |
| `git diff --check` | PASS |

The aggregate quick gate reaches and passes the changed-area, typecheck,
OpenAPI/generated, Go static/security, and browser lanes. Its inherited Phase
08 acceptance-contract lane remains nonzero because historical
`P08-SWR054-ACCEPT-01` evidence has no synchronized unresolved finding; the
same failure reproduces on the untouched phase worktree and is outside Task
296's scope.

## Traceability

The implementation cites `DESIGN-001` SettingsPanel/LocalStorageManager and
`DESIGN-008` PreferenceManager at the store, auth-client, component, and
design-contract surfaces. Task 296's task-list row is the only status row
promoted by this preparation.
