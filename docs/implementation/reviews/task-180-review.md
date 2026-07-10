# Task 180 Review Evidence

Task ID: 180

Task: Phase 06.01 Login View / DESIGN-018: LoginView

Recommended status: PASSED

## Status And Dependency Check

- Task 180 status in `docs/implementation/02_TASK_LIST.md`: PREPARED
- Dependency 178 status: PREPARED
- Dependency 179 status: PREPARED
- Dependency requirement satisfied: yes, dependencies are PREPARED or PASSED.

## Checklist

- Login form validation: PASS. Submit is blocked unless trimmed email and password are present, and the form reports missing credentials.
- Accessible labels and focus order: PASS. Email and password controls have explicit labels and safe autocomplete hints; Playwright verifies email -> password -> submit focus order.
- Generic invalid-credential copy without account enumeration: PASS. 401/`invalid_credentials` maps to `Email or password is incorrect.` and browser coverage verifies server enumeration text is not rendered.
- Lockout/rate-limit feedback with safe retry timing: PASS. `Retry-After` is parsed as strict unsigned seconds or HTTP-date, malformed and negative values are suppressed, and display seconds are clamped to 3600 in the client and defensively in `LoginView`.
- Disabled duplicate submissions: PASS. Submit is disabled while pending; Playwright force-clicks the loading button and verifies the route receives only the intended attempts.
- Password field clearing after submit: PASS. Password state is cleared on success and failure, and the generated auth client also clears the caller-owned request password in `finally`.
- Successful cookie-backed login session projection: PASS. Login flows through `loginWithEmail`/auth session store using generated DTOs and credentialed fetch behavior from dependency 178.
- Successful-session handoff to pending protected actions: PASS. The queued checkout action runs once after successful login.
- Preserved search state after closing auth surface: PASS. Browser coverage verifies the search input value remains after the sign-in surface is closed.

## Repair Verification

- `frontend/src/lib/api/auth-client.ts:295` to `frontend/src/lib/api/auth-client.ts:317` now normalizes retry metadata by accepting only strict digit seconds or valid HTTP-date values, suppressing malformed/past/negative values, and clamping display metadata to 3600 seconds.
- `frontend/src/lib/components/LoginView.svelte:64` to `frontend/src/lib/components/LoginView.svelte:85` maps lockout/rate-limit errors through `normalizeRetryAfterSeconds`, providing a second defensive finite/non-negative check and the same 3600 second cap before rendering.
- `frontend/src/lib/api/auth-client.test.ts:365` to `frontend/src/lib/api/auth-client.test.ts:396` covers malformed, negative, huge, and HTTP-date retry metadata.
- `frontend/tests/login.spec.ts:154` to `frontend/tests/login.spec.ts:205` verifies browser behavior for suppressed malformed/negative values, clamped huge values, and HTTP-date display. The same spec also covers focus order, generic invalid credentials, duplicate-submit prevention, password clearing, session handoff, and preserved search state.

No blocking findings remain for task 180.

## Commands

| Command | Cwd | Exit Code | Result |
| --- | --- | --- | --- |
| `rg -n "\| (178\|179\|180) \|" docs/implementation/02_TASK_LIST.md` | `/home/wiktor/Work/worktrees/gpt` | 0 | Confirmed task 180 and dependencies 178/179 are PREPARED. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/auth-client.test.ts src/lib/components/LoginView.test.ts` | `/home/wiktor/Work/worktrees/gpt/frontend` | 0 | 15 focused Bun tests passed. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun x playwright test tests/login.spec.ts` | `/home/wiktor/Work/worktrees/gpt/frontend` | 0 | 6 Playwright tests passed across desktop and mobile Chromium. Vite logged backend proxy ECONNREFUSED for unstubbed autocomplete calls, but the login assertions completed successfully. |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | `/home/wiktor/Work/worktrees/gpt/frontend` | 0 | Vite production build passed. |
| `git diff --check -- frontend/src/lib/api/auth-client.ts frontend/src/lib/api/auth-client.test.ts frontend/src/lib/components/LoginView.svelte frontend/src/lib/components/LoginView.test.ts frontend/tests/login.spec.ts` | `/home/wiktor/Work/worktrees/gpt` | 0 | No whitespace errors. |

## Files Inspected

- `docs/implementation/02_TASK_LIST.md`
- `evidence/reviews/task-180-review.md`
- `frontend/src/lib/api/auth-client.ts`
- `frontend/src/lib/api/auth-client.test.ts`
- `frontend/src/lib/components/LoginView.svelte`
- `frontend/src/lib/components/LoginView.test.ts`
- `frontend/tests/login.spec.ts`

## Decision Reason

The repaired implementation satisfies the task's verification criteria. The prior blocking retry-timing issue is fixed in both the auth client and LoginView rendering path, with unit and browser coverage for malformed, negative, huge, and HTTP-date `Retry-After` values. Focused tests, Playwright login coverage, production build, and whitespace checks passed.

## Repair Instructions

None.
