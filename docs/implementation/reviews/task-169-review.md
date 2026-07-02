# Review for Task 169: Phase 06 Search UI Entitlement Gating

## Decision
**REJECTED**

## Verification Criteria Checklist
- [x] Playwright and unit tests verify free users see disabled UI states or upsell modals for paid modes
- [ ] Playwright and unit tests verify unauthenticated users see disabled UI states or upsell modals for paid modes
- [ ] API requests are blocked proactively without relying solely on backend 402/403 errors
- [ ] TanStack Query correctly transitions from loading to error or upsell states upon entitlement-check failures
- [x] Authenticated paid users can access all search modes

## Findings
The implementation fails to handle the `undefined` state of the `entitlement` data (which occurs during loading and upon 401 unauthorized or network errors). This leads to several verification failures:

1. **Unauthenticated/Error fallback allows premium access:**
   In both `SubstitutionInputs.svelte` and `DailyDietControls.svelte`, the UI is disabled via `isBlocked` checks that are incorrectly permissive when data is missing:
   ```typescript
   let isBlocked = $derived(entitlement !== undefined && !entitlement.allowedModes.includes("daily_diet_alternative"));
   ```
   If the entitlement fetch fails (e.g., a 401 for an unauthenticated user), `entitlement` is `undefined`, making `isBlocked` evaluate to `false`. This leaves the UI open and allows the user to trigger proactive API requests, relying solely on the backend to reject them with 402/403.

2. **TanStack Query error state is ignored:**
   `SearchShell.svelte` maps `let entitlement = $derived(entitlementQuery.data);` but ignores `entitlementQuery.error` or `entitlementQuery.isError`. The UI never transitions to a visual error or upsell state if the entitlement check fails; it simply acts as if there are no gating restrictions.

3. **Incomplete Playwright verification:**
   `tests/entitlement.spec.ts` verifies disabled states for a mocked `"free"` user but has an incomplete test for anonymous users. The `"anonymous Catalog Search stays usable"` test asserts only that the Catalog mode is visible and usage is hidden; it fails to verify that the unauthenticated user is actually blocked from using paid modes (like Daily Diet or Multi-input Substitution).
