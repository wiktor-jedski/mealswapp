# Review Evidence: Task 168 — SearchView

## Decision

Recommended status: `REJECTED`

Reason: Frontend unit tests are missing verification for 402 error mapping, which is explicitly required by the verification criteria.

## Task Reviewed

- ID: 168
- Component: Phase 06 Frontend Entitlement Client
- Static Aspect: DESIGN-001: SearchView
- Input Status: PREPARED
- Retries: 0
- Depends On: 167

## Dependency Check

| Dependency ID | Expected Status | Observed Status | Result |
|---|---|---|---|
| 167 | PASSED | PASSED | PASS |

## Verification Checklist

| # | Criterion | Evidence Type | Result | Evidence Summary |
|---|---|---|---|---|
| 1 | Frontend unit tests verify credentialed entitlement fetches | command/file | PASS | `bun test` passed and `entitlement-client.test.ts` covers credentialed requests |
| 2 | Frontend unit tests verify 401 anonymous handling | command/file | PASS | `bun test` passed and test `maps 401 anonymous requests to auth AppError` exists |
| 3 | Frontend unit tests verify 402/409/503 error mapping | file | FAIL | Tests cover 409 and 503, but there is no test covering 402 error mapping |
| 4 | Frontend unit tests verify stable query keys | command/file | PASS | Test `returns stable query keys and exact retry behavior` exists |
| 5 | Frontend unit tests verify checkout idempotency-key generation and exact retry behavior | command/file | PASS | Tests `creates checkout session with idempotency key` and exact retry logic exist |
| 6 | No handwritten duplicate billing DTOs drift from generated types | file | PASS | `entitlement-client.ts` imports DTOs directly from `./generated` without redeclaring them |

## Commands Run

| Command | Working Directory | Exit Code | Result |
|---|---|---:|---|
| `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/entitlement-client.test.ts` | `/home/wiktor/Work/worktrees/gemini` | 0 | PASS |

## Files Inspected

| File | Reason | Finding |
|---|---|---|
| `frontend/src/lib/api/entitlement-client.ts` | Code implementation review | Properly uses generated types and correctly implements logic |
| `frontend/src/lib/api/entitlement-client.test.ts` | Test coverage review | Missing test for 402 error mapping |

## Coverage / Exception Review

Testing Coverage Exceptions from task:

> None

Coverage finding:

Tests ran correctly and passed, but do not fully cover the explicitly requested error codes (402).

## Failure Details

### Failed Criteria

- Frontend unit tests verify 402/409/503 error mapping: Missing test for 402 error mapping.

### Missing Evidence

- No missing evidence, the evidence proves the test does not exist.

### Repair Instructions

A repair agent should:
- Add a test for 402 error mapping in `frontend/src/lib/api/entitlement-client.test.ts` to ensure it correctly maps to the 'entitlement' AppError category.
- Run `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test src/lib/api/entitlement-client.test.ts` to verify the tests.

The repair agent should not:
- Modify `entitlement-client.ts` (unless a bug is found during test addition), as the mapping logic appears correct (just untested).
- Touch unrelated files or tasks.
