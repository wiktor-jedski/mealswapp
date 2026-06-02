# Phase 03 CSRF Foundation UAT

## Scope

Phase 03 tasks `73`-`75` replace the custom CSRF hook with Fiber v2 CSRF middleware, bind synchronizer tokens to Fiber sessions, expose `GET /api/v1/auth/csrf-token` for SPA delivery, and provide login/logout/refresh/reset lifecycle methods for later authentication handlers.

Traceability: `DESIGN-010: CSRFValidator`, `DESIGN-006: SessionManager`, and `DESIGN-006: AuthController`.

## Automated Verification

Run:

```sh
python3 scripts/check.py --output docs/implementation/implemented/03_PHASE_REPORT.html
cd backend
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi/... -coverprofile=/tmp/httpapi-cover.out
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go tool cover -func=/tmp/httpapi-cover.out
cd ..
python3 scripts/generate-api-types.py --check
python3 scripts/validate-traceability.py
python3 scripts/validate-task-list.py
```

Expected result: all commands pass, the aggregate report is generated, and backend internal packages plus frontend source report `100.0%` coverage.

## Project-Owner Acceptance Tests

1. Start the API and request `GET /api/v1/auth/csrf-token`.
2. Confirm the JSON envelope contains `data.csrfToken`.
3. Confirm the browser receives HttpOnly `mealswapp_csrf` and `mealswapp_session` cookies with `SameSite=Strict`.
4. Submit a protected mutation with browser credentials and the returned `X-CSRF-Token`; accept only when the request reaches its handler.
5. Repeat without the header, with another session's token, and with a token retained across logout or authorization-state rotation; accept only when each request returns structured `403 csrf_failed`.

## Notes

- Redis-backed Fiber session storage is deferred before horizontally scaled deployment.
- Authentication route handlers are still future work and must invoke the supplied lifecycle methods.
- `python3 scripts/check.py --output docs/implementation/implemented/03_PHASE_REPORT.html` passed during phase completion.

## Acceptance Decision

Accept this phase when the automated checks pass and the project owner confirms the browser-oriented checks above.
