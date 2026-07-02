# Review Evidence: Task 169 — DESIGN-001: SearchView

## Decision

Recommended status: `REJECTED`

Reason: The implementation breaks an existing E2E test (`search-workflow.spec.ts`) because it does not mock the new `/api/v1/entitlements` endpoint in the core test fixtures, causing the Daily Diet Alternative input to be disabled indefinitely during the test.

## Task Reviewed

- ID: 169
- Component: DESIGN-001: SearchView
- Static Aspect: PREPARED
- Input Status: PREPARED
- Retries: 0
- Depends On: 168

## Dependency Check

| Dependency ID | Expected Status | Observed Status | Result |
|---|---|---|---|
| 168 | PASSED | PASSED | PASS |

## Verification Checklist

| # | Criterion | Evidence Type | Result | Evidence Summary |
|---|---|---|---|---|
| 1 | Component and Playwright tests verify free-user usage counter display | Test | PASS | `entitlement.spec.ts` and `SearchModes.test.ts` verify the counter conditionally renders. |
| 2 | single-input Substitution remains usable until the limit | File inspection | PASS | `SubstitutionInputs.svelte` computes `isBlocked` using `isPremiumBlocked && $searchStore.substitutionInputs.length > 1`, ensuring single input remains usable. |
| 3 | multi-input Substitution and Daily Diet modes show entitlement feedback without sending blocked searches | Test | PASS | Playwright E2E tests and component tests ensure `[data-entitlement-feedback]` appears when conditions are met and inputs are disabled. |
| 4 | trial/paid fixtures unlock paid modes | Test | PASS | Playwright E2E tests mock paid tier and verify `[data-entitlement-feedback]` is hidden. |
| 5 | anonymous Catalog Search stays usable | Test | PASS | Playwright E2E tests verify a 401 response keeps Catalog mode visible and hides entitlement usage counters. |
| 6 | keyboard/focus behavior remains accessible | Test | PASS | Playwright E2E tests verify `getByRole('button', { name: 'Substitution' }).focus()` behaves correctly. |
| 7 | Existing E2E tests continue to pass | Test | FAIL | `search-workflow.spec.ts` fails because the Daily Diet Alternative input is disabled due to missing entitlement mocks. |

## Commands Run

| Command | Working Directory | Exit Code | Result |
|---|---|---:|---|
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun install && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | `/home/wiktor/Work/worktrees/gemini` | 0 | PASS |
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e` | `/home/wiktor/Work/worktrees/gemini` | 1 | FAIL |

## Files Inspected

| File | Reason | Finding |
|---|---|---|
| `frontend/src/lib/components/SearchShell.svelte` | Verify entitlement gating integration | Integrates `entitlementQuery.data` and passes it down. |
| `frontend/src/lib/components/DailyDietControls.svelte` | Verify premium blocking | Derives `isBlocked` based on `allowedModes.includes("daily_diet_alternative")`, disabling the input if true or loading. |
| `frontend/tests/entitlement.spec.ts` | Verify Playwright criteria | Has test cases asserting required E2E entitlement flows. |
| `frontend/tests/search-workflow.spec.ts` | Debug E2E test failure | The test `Daily Diet Alternative search shows the structured 422 rejection` times out filling `#daily-diet-id` because `stubCoreApi` and the test itself do not mock `/api/v1/entitlements`, leaving the input disabled (`isLoading` or `isError`). |

## Coverage / Exception Review

Testing Coverage Exceptions from task:

> None

Coverage finding:

Tests ran but the E2E suite failed due to existing tests not mocking the newly introduced entitlement endpoint.

## Failure Details

### Failed Criteria

- Verification cannot be trusted because existing E2E tests in `search-workflow.spec.ts` fail due to missing mocks for the new entitlement API, causing inputs to be disabled.

### Missing Evidence

- None

### Repair Instructions

A repair agent should:
- Update `stubCoreApi` in `frontend/tests/search-workflow.spec.ts` (and any other relevant E2E tests) to mock the `/api/v1/entitlements` endpoint with a paid/trial entitlement so that premium features like Daily Diet Alternative remain enabled for existing workflow tests.
- Re-run `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e` to ensure all tests pass.
- Regenerate the review evidence.

The repair agent should not:
- Modify `frontend/tests/entitlement.spec.ts` as it correctly tests the specific entitlement gating behaviors.
- Change the functional implementation of entitlement gating in the Svelte components.
