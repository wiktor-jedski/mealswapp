# Task 178 Review Evidence

## Task

- ID: 178
- Component: Phase 06.01 Auth API Client
- Static aspect: DESIGN-018: AuthApiClient
- Status reviewed: PREPARED
- Retries: 0
- Dependency reviewed: 177 is PREPARED
- Recommendation: PASSED

## Checklist

- PASS - Selected task row is PREPARED.
- PASS - Dependency task 177 is PREPARED.
- PASS - Preparation report claims task 178 is complete and ready for review.
- PASS - Client wrappers cover CSRF, register, login, logout, session refresh, profile probing, OAuth start URL construction, disclaimer loading, and entitlement refresh coordination.
- PASS - Request URLs, HTTP methods, and `credentials: include` are verified by unit tests and generated request helper inspection.
- PASS - CSRF token retrieval and CSRF header handling are verified for protected mutations.
- PASS - Generated request/response DTOs are used by `auth-client.ts`; no duplicate `RegisterRequest`, `LoginRequest`, `AuthSessionData`, `ProfileData`, `DisclaimerData`, or `EntitlementStatusData` declarations were found in the auth client.
- PASS - 400, 401, 403, 409, 429, and 503 error mapping is verified by unit tests.
- PASS - Session and profile responses strip unexpected token-like fields before returning JavaScript-visible data.
- PASS - Register and login helpers clear caller-owned raw password fields after success and failure.
- PASS - Implementation remains scoped to task 178 and does not implement task 179 auth session store behavior.
- PASS - Traceability comments cite DESIGN-018 and related DESIGN-017 error mapping.

## Commands

- `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/auth-client.test.ts` -> exit 0; 9 pass, 0 fail.
- `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` -> exit 0; 273 pass, 0 fail.
- `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` -> exit 0; Vite production build succeeded.
- `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check:api-types` -> exit 0; generated API types are current.
- `python3 scripts/validate-traceability.py` -> exit 0; traceability validation passed.
- `python3 scripts/validate-task-list.py` -> exit 0; task-list validation passed.

## Files Inspected

- `docs/implementation/02_TASK_LIST.md` - verified task 178 status and dependency 177 status.
- `frontend/src/lib/api/auth-client.ts` - reviewed implementation scope, generated contract usage, credentialed fetch wrappers, safe error mapping, token stripping, and password clearing.
- `frontend/src/lib/api/auth-client.test.ts` - reviewed task-specific verification coverage.
- `frontend/src/lib/api/generated.ts` - verified generated endpoint constants, DTOs, request helpers, and credential inclusion.
- `frontend/src/lib/api/generated.test.ts` - checked dependency coverage around generated auth and entitlement contracts.

## Decision Reason

Task 178 satisfies the verification criteria with direct unit-test coverage and supporting file inspection. The auth API client delegates request shapes and DTOs to generated contracts, includes credentials on all relevant calls, maps the required error statuses to safe `AppError` values, strips token-like response fields from session/profile results, clears caller-owned password values after submission helpers resolve or reject, and stays within the selected task scope. All practical verification commands passed.
