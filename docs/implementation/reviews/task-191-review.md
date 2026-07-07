# Review Evidence: Task 191 — DESIGN-006: OAuthHandler

## Decision

Recommended status: `REJECTED`

Reason: Required validation includes a failing task-list validator, and required frontend/preparation evidence is missing or stale.

## Task Reviewed

- ID: 191
- Component: Phase 09 Live Google OAuth Enablement
- Static Aspect: DESIGN-006: OAuthHandler
- Input Status: PREPARED
- Retries: 0
- Depends On: 188

## Dependency Check

| Dependency ID | Expected Status | Observed Status | Result |
|---|---|---|---|
| 188 | PASSED | PASSED | PASS |

## Verification Checklist

| # | Criterion | Evidence Type | Result | Evidence Summary |
|---|---|---|---|---|
| 1 | Selected task status is `PREPARED`. | file | PASS | `docs/implementation/02_TASK_LIST.md:198` lists task 191 as `PREPARED`. |
| 2 | Dependency task 188 is `PREPARED` or `PASSED`. | file | PASS | `docs/implementation/02_TASK_LIST.md:195` lists task 188 as `PASSED`. |
| 3 | Preparation report claims task 191 is complete or ready for review. | file search | FAIL | No task-191 preparation report was found under `evidence/` or `docs/implementation/implemented/`; only the task-list row marks the task `PREPARED`. |
| 4 | Backend tests verify Google start redirects to provider URL with a cryptographically random state cookie. | command/file | PASS | `go test ./internal/...` passed; `oauth_controller.go:65-75` generates and stores state, `oauth_controller.go:114-119` uses `crypto/rand`, and `oauth_controller_test.go` checks redirect/state cookie presence. |
| 5 | Backend tests verify callback rejects missing or mismatched state and clears state cookie on success/failure. | command/file | FAIL | Implementation rejects missing/mismatched state via `validOAuthState`, and mismatch clearing is tested, but no backend test was found for missing state specifically. |
| 6 | Backend tests verify callback maps Google profile through OAuth account linking/session creation and sets auth cookies. | command/file | PASS | `go test ./internal/...` passed; `oauth_controller_test.go` covers service mapping and auth cookie creation on callback. |
| 7 | Backend tests verify missing Google config fails closed without leaking secrets. | command/file | PASS | `go test ./internal/...` passed; `app_test.go` covers `NewGoogleOAuthGateway(config.OAuthConfig{})` fail-closed and provider URL without client secret. |
| 8 | Backend tests verify Apple start remains unavailable. | command/file | PASS | `go test ./internal/...` passed; `app_test.go` checks `StartOAuth(..., "apple", ...)` returns an error. |
| 9 | OpenAPI documents optional `return_to` on OAuth start. | file/command | PASS | `api/openapi.yaml:202-208` documents optional `return_to`; Redocly lint passed with one warning. |
| 10 | Frontend tests verify auth modal renders `Continue with Google` but not Apple. | file/command | FAIL | `bun test` passed, but `OAuthEntryPoint.svelte:19-20` labels the visible provider as `Google`, and tests assert `label: "Google"` rather than `Continue with Google`; Apple is absent. |
| 11 | Frontend tests verify Google start uses `/api/v1/auth/oauth/google/start?return_to=<safe-current-path>`. | file/command | PASS | `bun test` passed; `oauth-entry-point.test.ts` verifies `/api/v1/auth/oauth/google/start?return_to=%2Fsubscription%3Fplan%3Dannual`. |
| 12 | Frontend tests verify unsafe OAuth URLs are rejected. | file/command | PASS | `bun test` passed; `oauth-entry-point.test.ts` rejects a provider start URL containing `client_secret`. |
| 13 | Frontend tests verify session state refresh after redirect. | file/command | PASS | `bun test` passed; `auth-session.test.ts` and `oauth-entry-point.test.ts` cover OAuth-return refresh. |
| 14 | Backend `go test ./internal/...` passes. | command | PASS | Exit code 0. |
| 15 | Backend `go vet ./...` passes. | command | PASS | Exit code 0. |
| 16 | `govulncheck ./...` runs when available. | command | PASS | Exit code 0; no called vulnerabilities found. |
| 17 | Frontend `bun test` passes. | command | PASS | Exit code 0; 324 tests passed. |
| 18 | Frontend build passes. | command | PASS | Exit code 0; Vite production build completed. |
| 19 | OpenAPI lint passes. | command | PASS | Exit code 0; API valid with one warning about 302-only callback responses. |
| 20 | `python3 scripts/validate-task-list.py` passes. | command | FAIL | Exit code 1; validator reports task IDs start at 100 and many dependencies reference missing older IDs. |
| 21 | `python3 scripts/validate-traceability.py` passes. | command | PASS | Exit code 0; traceability validation passed. |
| 22 | Implementation stays scoped to task 191 and does not implement later task IDs. | file inspection | PASS | Reviewed OAuth backend/frontend/OpenAPI surfaces; no obvious later Apple OAuth implementation was found. |

## Commands Run

| Command | Working Directory | Exit Code | Result |
|---|---|---:|---|
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/...` | `/home/wiktor/Work/worktrees/gpt/backend` | 0 | PASS |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | `/home/wiktor/Work/worktrees/gpt/backend` | 0 | PASS |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `/home/wiktor/Work/worktrees/gpt/backend` | 0 | PASS |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` | `/home/wiktor/Work/worktrees/gpt/frontend` | 0 | PASS |
| `BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build` | `/home/wiktor/Work/worktrees/gpt/frontend` | 0 | PASS |
| `npx --no-install redocly lint api/openapi.yaml` | `/home/wiktor/Work/worktrees/gpt` | 0 | PASS |
| `python3 scripts/validate-task-list.py` | `/home/wiktor/Work/worktrees/gpt` | 1 | FAIL |
| `python3 scripts/validate-traceability.py` | `/home/wiktor/Work/worktrees/gpt` | 0 | PASS |

## Files Inspected

| File | Reason | Finding |
|---|---|---|
| `docs/implementation/02_TASK_LIST.md` | Task and dependency status | Task 191 is `PREPARED`; dependency 188 is `PASSED`. |
| `backend/internal/httpapi/oauth_controller.go` | OAuth start/callback behavior | Generates state with `crypto/rand`, stores short-lived cookies, validates state, clears OAuth cookies via defers, sets auth cookies, and redirects to sanitized relative frontend path. |
| `backend/internal/httpapi/oauth_controller_test.go` | Backend OAuth verification | Covers Google start/callback, mismatched state, link-required/provider mismatch, unavailable gateway, safe return path. Missing-state callback is not directly tested. |
| `backend/internal/app/oauth_gateway.go` | Goth provider gateway | Configures Google-only `goth` gateway; rejects non-Google providers and missing config. |
| `backend/internal/app/app.go` | Production wiring and unavailable gateway | Wires `NewGoogleOAuthGateway(cfg.OAuth)` into `OAuthController`; unavailable gateway fails closed. |
| `backend/internal/app/app_test.go` | Gateway configuration tests | Verifies missing config fail-closed, Apple unavailable, Google URL includes provider and state without secret. |
| `backend/internal/config/config.go` | OAuth configuration | Loads Google client ID, secret, callback URL and validates callback scheme. |
| `backend/internal/config/config_test.go` | OAuth config tests | Verifies Google config load and invalid callback rejection. |
| `api/openapi.yaml` | OAuth API contract | Documents optional `return_to`, but summaries and provider enum still mention/allow Apple. |
| `frontend/src/lib/components/OAuthEntryPoint.svelte` | Auth modal provider UI | Renders only Google provider, but visible label is `Google`, not `Continue with Google`. |
| `frontend/src/lib/components/oauth-entry-point.ts` | Frontend OAuth routing safety | Builds provider start URL from generated client, rejects unsafe URL shapes and sensitive query names. |
| `frontend/src/lib/components/oauth-entry-point.test.ts` | Frontend provider tests | Verifies Google route, Apple fail-closed, unsafe URL rejection, OAuth-return refresh; tests assert `Google`, not `Continue with Google`. |
| `frontend/src/lib/stores/auth-session.ts` | OAuth-return session refresh | Refreshes server session and entitlement without trusting URL parameters. |
| `frontend/src/lib/stores/auth-session.test.ts` | Session refresh tests | Verifies OAuth return refresh succeeds/fails based on server session refresh. |

## Coverage / Exception Review

Testing Coverage Exceptions from task:

> None

Coverage finding:

The task verification criteria did not require coverage commands, and no coverage exception is allowed. Required backend/frontend test commands passed, but coverage was not regenerated for this review. The review still fails because required command validation and specific evidence criteria failed.

## Failure Details

### Failed Criteria

- Preparation report evidence is missing: no task-191 preparation report was found under `evidence/` or `docs/implementation/implemented/`.
- Backend test coverage is incomplete for the stated criterion: callback rejects missing state is implemented but not directly verified by a test found in `backend/internal/httpapi/oauth_controller_test.go`.
- Frontend visible provider label does not match the verification criterion: the component renders `Google`, while the task requires tests to verify `Continue with Google`.
- `python3 scripts/validate-task-list.py` fails with exit code 1.

### Missing Evidence

- A task-191 preparation report claiming the task is complete or ready for review.
- Direct backend test evidence for a missing-state callback rejection.
- Frontend test evidence for visible `Continue with Google` text.
- Passing task-list validation evidence.

### Repair Instructions

A repair agent should:
- Add or restore task-191 preparation evidence that explicitly claims the task is complete and ready for review.
- Add a backend test for callback without an OAuth state cookie and/or without a callback `state` query, asserting rejection and OAuth cookie clearing.
- Change the frontend OAuth action text and tests to require `Continue with Google`, while keeping Apple hidden.
- Fix or document the task-list validator failure so `python3 scripts/validate-task-list.py` exits 0 under the current task-list shape.
- Rerun the full task-required command set: backend `go test ./internal/...`, `go vet ./...`, `govulncheck ./...`, frontend `bun test`, frontend build, OpenAPI lint, `python3 scripts/validate-task-list.py`, and `python3 scripts/validate-traceability.py`.

The repair agent should not:
- Mark task 191 `PASSED` directly.
- Enable Apple OAuth.
- Work on later task IDs.
- Touch unrelated dirty phase work except where needed to make the required task-list validator pass.
