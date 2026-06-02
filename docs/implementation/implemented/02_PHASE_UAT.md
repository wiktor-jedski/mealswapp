# Phase 02 UAT: API Gateway Repair

## Scope

Phase 02 covers tasks `50`-`82`. The repaired gateway uses Fiber CSRF and limiter
middleware, session-bound SPA CSRF token delivery, explicit mutation CSRF
policies, cooperative request deadlines, and request-correlated security audits.
Flagged sensitive mutations persist an audit before handler dispatch. Low-risk
reads remain available during audit outages.

Phase 02 redirects deployed HTTP traffic to HTTPS but rejects
`MEALSWAPP_TRUST_PROXY=true` and ignores `X-Forwarded-Proto`. Phase 09 owns TLS
1.3 edge enforcement and restricted trusted ingress.

## Automated Evidence

Run from the repository root:

```sh
git diff --check
python3 scripts/validate-task-list.py
python3 scripts/validate-traceability.py
cd frontend && bun run check:api-types
cd ..
python3 scripts/check.py --output docs/implementation/implemented/02_PHASE_REPORT.html
```

The aggregate gate verifies migrations, probes, screenshots, formatting,
backend tests and 100% coverage, frontend build and 100% coverage, generated
API-type drift, requirement coverage including `SW-REQ-090` and `SW-REQ-091`,
and traceability.

## Project-Owner Checks

1. Start services and the API with `bash scripts/start-services.sh`, then run
   migrations and `go run ./cmd/api` from `backend/`.
2. Request `/health`, `/ready`, `/api/v1/health`, and `/api/v1/ready`; confirm
   envelopes contain `requestId` and browser security headers.
3. Request `GET /api/v1/auth/csrf-token`; confirm the body contains `csrfToken`
   and cookies are HttpOnly with `SameSite=Strict`.
4. Run `cd backend && go test ./internal/httpapi/...`; confirm protected
   mutations reject missing, stale, and cross-session tokens, audit outages
   block flagged mutations before dispatch, reads continue, limiter responses
   contain `Retry-After`, and cooperative timeouts return structured `504`.
5. Run the API with deployed TLS redirects enabled and send HTTP requests with
   spoofed `X-Forwarded-Proto: https`; confirm requests still redirect.

## Deferred Work

- Phase 03 account handlers must call the authorization-state rotation and
  invalidation helpers on login, refresh, password-reset completion, and logout.
- Before horizontal scaling, move Fiber session storage to the documented Redis
  session namespace.
- Phase 09 must deploy and verify restricted ingress before adding trusted
  forwarded-scheme support or enabling TLS 1.3 edge enforcement.

## Acceptance

Accept Phase 02 after the automated gate passes and the project-owner checks
confirm the repaired gateway behavior.
