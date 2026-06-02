# Open Points

## Phase XX - Template - do not edit, add next after this one

Note: not all fields need to be filled. Don't add unnecessary information.

### Assumptions
- Write here anything that has been assumed about the implementation AND is missing in the design.

### Clarifications
- Write here anything that needs to be clarified - insert here all of your questions.

### Actions needed
- Immediate problems that need to be solved and will block us in the future.

## Phase 00

No project-owner action is required for Phase 00 at this time.

### Assumptions

- The implementation task list at `docs/implementation/02_TASK_LIST.md` is the authoritative task list. The root `docs/02_TASK_LIST.md` file is a pointer to avoid duplicate task status sources. - resolved, removed pointer docs
- Phase 00 uses development-only local credentials from `.env.example` and `docker-compose.yml`; production secrets remain outside the repository. - OK, accepted
- OpenAPI type generation is documented as deferred until domain request and response contracts exist in later phases. - added to phase 02

### Testing coverage deviations

- Resolved for Phase 00 testable source. `python3 scripts/check.py` now enforces 100% coverage for backend internal packages with `go test ./internal/... -coverprofile=coverage.out` and frontend source with `bun test --coverage`.
- Backend `cmd/*` entrypoints remain covered by build/smoke verification rather than line coverage, because they are process bootstrap commands that bind ports, connect to local services, or run migrations.

## Phase 01

### Assumptions

- Resolved: add an optional `averageServingVolumeMilliliters` field for liquids because food-data sources provide milliliters per serving. Keep `averageUnitWeightGrams` for solid serving-to-gram conversion. Do not use grams as a 1:1 milliliter proxy.
- Resolved: require positive `densityGramsPerMilliliter` and `densitySourceKind` for liquids. Derive density from trusted USDA volume portions with gram weights when available, preferring `ml`, `cup`, `tbsp`, `tsp`, then `fl_oz`. Provider and source food ID remain optional for manual or estimated values. Do not silently assume `1 ml = 1 g`.
- Resolved: defer the dedicated saved-diet table until Phase 07 diet optimization. `saved_items.item_id` remains polymorphic for `favorite`, `saved_meal`, and future `saved_diet` rows. Until Phase 07 adds the target table, repositories must reject attempts to save the reserved `saved_diet` kind.
- Resolved: composite-meal ingredients remain limited to food-item references. Nested composite meals are intentionally unsupported, so recipe-cycle prevention is unnecessary.
- Accepted: `RecipeIngredientEntity` includes `position`, although `DESIGN-005` does not explicitly list it. SQL row order is undefined, so the field preserves deterministic ingredient display and editing order.
- Accepted: `RepositoryQuery` includes normalized-name, tag, preparation-time, pagination, and repository-context fields, although `DESIGN-005` does not explicitly list them. These fields are Phase 01 persistence-layer candidate-retrieval primitives for later search orchestration; they do not perform similarity ranking. Macro-range fields were intentionally removed because target-based similarity search accepts a concrete macro vector instead of speculative advanced filters.
- Accepted: OAuth login identities are stored separately from `AuthUser` password credentials in `oauth_identities`. `AuthUser.passwordHash` and `AuthUser.passwordSalt` are optional as a pair, and OAuth-only users do not store placeholder password credentials. This supports linking multiple authentication methods to one account.
- Resolved: keep ingredient classification tag-based and add named dietary rules in Phase 04 search as constraints composed from tag inclusions and exclusions. For example, `pescatarian` allows fish while excluding meat.
- Resolved: persist search history for authenticated users only after a completed search returns valid results. Keep duplicate searches, retain the latest 100 rows per user, expose clear-history behavior, and do not persist anonymous searches. Duplicate collapsing is deferred unless the UI becomes noisy.
- Resolved: keep rolling 24-hour usage counters PostgreSQL-backed for Phase 06. The timestamped usage rows and caller-supplied cutoff provide durable enforcement. Consider Redis only if measurements show entitlement checks are a bottleneck.
- Resolved: account deletion uses `pending -> processing -> completed|failed`, with `failed -> processing` permitted for retry and `completed` terminal. Store a sanitized transient, permanent, or unknown failure category. Retry transient failures automatically with exponential backoff up to 3 attempts. Require admin-triggered retry after investigation for permanent, unknown, or exhausted failures, and alert when requests fail or exhaust retries.
- Resolved provisionally: retain a minimal pseudonymous deletion receipt after account erasure for GDPR accountability. Store a random receipt ID, request and completion timestamps, final outcome, and sanitized failure category when applicable. Do not retain the deleted user ID, email, or account data. Use a provisional three-year retention period pending legal review before production.
- Resolved: validate liquid macros as non-negative, but do not apply the solid `protein + carbohydrates + fat <= 100 g` rule to values stored per `100 ml`. Without density data, values above that threshold can be legitimate. Add external-import warnings for suspicious liquid values instead of rejecting them solely for exceeding `100 g`.
- Resolved: normalize mixed solid/liquid composite meals using required liquid density. Missing persisted density is invalid data and returns an error. Do not invent density conversions.

### Actions needed

- Obtain privacy-law review before production for the pseudonymous deletion-receipt fields and provisional three-year retention period.

## Phase 02

### Assumptions

- Add dedicated security-audit persistence for request-correlated authentication, API error, CSRF, rate-limit, and future admin events. Keep it separate from the Phase 01 admin-audit table because the event shapes and fail-closed security-mutation behavior differ.
- Implement AES-256-GCM envelope encryption, key versions, key-loader interfaces, and a production GCP Secret Manager adapter boundary in Phase 02. Use explicit local test fixtures for development and tests. Defer wiring encryption into concrete PII repository fields until Phase 03 authentication and profile services define the plaintext service boundaries.
- Implement local structured JSON logging, in-memory metrics test sinks, emitted metric names, probe cadence, and alert-rule configuration in Phase 02. Defer deployed GCP Cloud Monitoring resources, notification channels, backup monitoring resources, and dashboards until Phase 09 production hardening.
- SW-REQ-091 requires proxy-provided scheme headers to be trusted for TLS enforcement only when deployed ingress restricts direct application access to the configured reverse proxy or load balancer. Local development keeps direct HTTP enabled with trusted-proxy mode disabled.
- Provide CSRF token generation and validation hooks in Phase 02 using test-only mutation routes for verification. Wire token issuance into browser session or authentication flows when those routes are added in Phase 03.
- Resolved: Phase 03 tasks 73-75 replace the custom Phase 02 CSRF hook with Fiber v2 CSRF middleware, bind tokens to Fiber sessions, and expose `GET /api/v1/auth/csrf-token` for safe SPA token delivery. Future pre-authentication auth routes must explicitly choose CSRF protection or an exemption when they are added.

### Actions needed

- Before Phase 03 implementation, enumerate the concrete user and profile fields treated as PII and confirm which service boundaries may decrypt each field.
- Before Phase 09 deployment work, confirm the production reverse proxy or load balancer topology and implement the SW-REQ-091 ingress restriction so trusted forwarded-scheme handling cannot accept spoofed headers from arbitrary clients.

## Phase 03

### Assumptions

- Fiber session storage remains in-process for the CSRF foundation. Before horizontally scaled deployment, wire the Fiber session store to the documented Redis session namespace so CSRF state is shared between API instances.
- Login, refresh, password-reset completion, and logout handlers are not implemented yet. Their future authorization-state transitions must call `RegenerateAuthorizationState` or `InvalidateAuthorizationState` before returning success.

### Testing coverage deviations

- None. `python3 scripts/check.py --output docs/implementation/implemented/03_PHASE_REPORT.html` passes with 100% backend internal-package and frontend source coverage.
