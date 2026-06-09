# Phase 03 UAT: Authentication, Profile, Consent

## Scope

Phase 03 covers tasks `83`-`106`. The backend now has account configuration,
password hashing, JWT access tokens, hashed refresh/reset tokens, lockout,
session cookies, registration with consent, login, refresh, logout, password
reset, authenticated verification hook, OAuth provider boundary, profile and
preferences, saved data/history reads and deletes, account export, account
deletion coordination, disclaimer content, OpenAPI contracts, and generated
frontend API types.

Protected `/api/v1` routes derive user identity only from validated JWT cookies
and repository session state. Client identity headers and body-supplied user IDs
are ignored for user scoping. Account export decrypts PII only at the export
boundary, and reset/refresh tokens persist only hashes.

## Automated Evidence

Run from the repository root unless noted:

```sh
git diff --check
python3 scripts/validate-task-list.py
python3 scripts/validate-traceability.py
npx --no-install redocly lint api/openapi.yaml
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check:api-types
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test
cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi/... ./internal/auth/... ./internal/profile/...
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...
cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./...
python3 scripts/check.py --output docs/implementation/implemented/03_PHASE_REPORT.html
```

Observed results:

- Task-list validation, traceability validation, and `git diff --check` passed.
- Redocly lint passed with one warning for the OAuth start route's `302`
  redirect-only response.
- Frontend API types regenerated deterministically and `check:api-types`
  passed.
- Frontend tests, build, and coverage passed.
- Backend HTTP/auth/profile package tests passed.
- `go vet`, `govulncheck`, and `go test -race ./...` passed.
- `python3 scripts/check.py --output docs/implementation/implemented/03_PHASE_REPORT.html`
  passed, including migrations, local stack probes, frontend screenshot
  verification, backend tests, race detection, generated type drift detection,
  coverage checks, and completed-phase HTML report generation.

## Project-Owner Checks

1. Start PostgreSQL and Redis with `bash scripts/start-services.sh`, then run
   `cd backend && go run ./cmd/migrate up && go run ./cmd/api`.
2. Register a new account with current privacy and terms versions; confirm the
   response contains an auth envelope without token values and the browser
   receives HttpOnly auth cookies.
3. Request `GET /api/v1/auth/csrf-token`, then submit authenticated profile
   updates with `X-CSRF-Token`; confirm missing or mismatched tokens return
   structured `403` errors.
4. Login, refresh, and logout; confirm refresh rotates cookies, logout clears
   auth cookies, and protected profile/export routes reject the cleared session.
5. Request password reset for an existing and a missing email; confirm both
   return the same accepted envelope and no reset token appears in the response.
6. Read and update `/api/v1/profile`; confirm display name, unit system, theme,
   and recalculation hint match the request.
7. Call saved-data and history endpoints; confirm only the authenticated user's
   saved items/history are visible and deletes require CSRF.
8. Download `/api/v1/account/export?format=json` and `format=csv`; confirm the
   export contains account/profile/consent/saved/history sections and no data
   from other users.
9. Delete the account with CSRF; confirm a pending deletion request envelope,
   cleared auth cookies, failed subsequent login/profile/export attempts, and a
   pseudonymous receipt after the deletion executor completes.
10. Request `/api/v1/disclaimers?location=login` and `location=account`; confirm
    stable Markdown content and fallback alert behavior when configured content
    is unavailable.

## Traceability

Primary design sources:

- `docs/design/DESIGN-006.md`: AuthController, SessionManager, JWTManager,
  PasswordHasher, AccountLockoutTracker, OAuthHandler.
- `docs/design/DESIGN-008.md`: ProfileController, PreferenceManager,
  SavedDataRepository, SearchHistoryRepository, DataExporter, AccountDeleter.
- `docs/design/DESIGN-010.md`: RouteHandler, CSRFValidator, RateLimiter,
  request validation and protected-route composition.
- `docs/design/DESIGN-013.md`: EncryptionService, AuditLogger, secure cookies,
  logs, token and PII handling.
- `docs/design/DESIGN-014.md`: MetricsCollector, aggregate checks, local stack
  verification, coverage reporting.
- `docs/design/DESIGN-015.md`: ConsentManager, DisclaimerRenderer,
  DataRetentionPolicy.
- `docs/design/DESIGN-017.md`: AppError and shared API envelopes.

## Known Deviations

- Phase 03 backend internal line coverage is accepted at 90.3% and documented in
  `docs/implementation/04_OPEN.md`. The aggregate gate now requires that
  documented deviation marker before accepting coverage below 100%.
- Phase 03 email verification is an authenticated, CSRF-protected hook that
  marks only the server-derived user. Signed single-use email-verification
  tokens and outbound email delivery remain required before production paid
  feature unlocks.
- Account export returns an empty `customItems` array until a user-owned custom
  item persistence model exists.
- Pseudonymous deletion receipt fields and the provisional three-year retention
  period require privacy-law review before production.

## Acceptance

Accept Phase 03 after the automated evidence remains green and the
project-owner checks confirm registration through deletion behavior, CSRF and
cookie handling, export contents, disclaimer fallback behavior, and the listed
legal/product follow-ups are accepted for later phases.
