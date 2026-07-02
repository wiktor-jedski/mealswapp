# Review Evidence: Task 170 — DESIGN-007: SubscriptionController

## Decision

Recommended status: `REJECTED`

Reason: The implementation is missing the required axe accessibility checks for the Subscription UI.

## Task Reviewed

- ID: 170
- Component: Phase 06 Subscription UI and Checkout Flow
- Static Aspect: DESIGN-007: SubscriptionController
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
| 1 | Playwright tests verify monthly and annual buttons call checkout creation with generated contracts | test result | PASS | `tests/subscription.spec.ts` verifies button clicks map to correct payload |
| 2 | loading and retry states are visible | test result | PASS | Verified in `tests/subscription.spec.ts` |
| 3 | Stripe redirect URL is followed only from the server response | test result | PASS | Verified in `tests/subscription.spec.ts` |
| 4 | cancel and success return routes refresh entitlement state | test result | PASS | Verified in `tests/subscription.spec.ts` |
| 5 | past_due/cancelled states show recovery actions | test result | PASS | Verified in `tests/subscription.spec.ts` |
| 6 | axe checks report no serious or critical violations | test result | FAIL | No axe scan is performed on the subscription UI in Playwright tests |
| 7 | no application form captures PAN/CVC fields | file inspection, test result | PASS | Verified in test and source code inspection |

## Commands Run

| Command | Working Directory | Exit Code | Result |
|---|---|---:|---|
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e tests/subscription.spec.ts` | `/home/wiktor/Work/worktrees/gemini` | 0 | PASS |

## Files Inspected

| File | Reason | Finding |
|---|---|---|
| `frontend/src/lib/components/SubscriptionController.svelte` | Inspect implementation | Implemented the subscription UI correctly without capturing raw card data. |
| `frontend/src/lib/components/SidebarComponent.svelte` | Inspect usage | Included `SubscriptionController` inside authenticated sidebar content. |
| `frontend/tests/subscription.spec.ts` | Review tests | All non-axe criteria are tested successfully. |
| `frontend/tests/accessibility.spec.ts` | Look for axe checks | The global accessibility test does not mock the authenticated state, leaving `SubscriptionController` untested. |

## Coverage / Exception Review

Testing Coverage Exceptions from task:

> None

Coverage finding:

Not verified as the task fails on explicit Playwright testing criteria.

## Failure Details

### Failed Criteria

- `axe checks report no serious or critical violations` — The `frontend/tests/subscription.spec.ts` file lacks an `@axe-core/playwright` check and the component is not tested in the global accessibility tests because those do not mock the signed-in state.

### Missing Evidence

- Missing Playwright axe scan report for the Subscription UI.

### Repair Instructions

A repair agent should:
- Import `@axe-core/playwright` and add an axe accessibility scan to the tests in `frontend/tests/subscription.spec.ts` while the Subscription UI is visible.
- Run `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e tests/subscription.spec.ts` to ensure the axe checks pass.
- Regenerate the test run output as evidence.

The repair agent should not:
- Remove the accessibility criterion from the task.
- Modify the `DESIGN-007` requirements.
