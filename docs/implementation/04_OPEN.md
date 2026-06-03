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
- Accepted: `RepositoryQuery` includes normalized-name, classification, preparation-time, pagination, and repository-context fields, although `DESIGN-005` does not explicitly list them. These fields are Phase 01 persistence-layer candidate-retrieval primitives for later search orchestration; they do not perform similarity ranking. Macro-range fields were intentionally removed because target-based similarity search accepts a concrete macro vector instead of speculative advanced filters.
- Accepted: OAuth login identities are stored separately from `AuthUser` password credentials in `oauth_identities`. `AuthUser.passwordHash` and `AuthUser.passwordSalt` are optional as a pair, and OAuth-only users do not store placeholder password credentials. This supports linking multiple authentication methods to one account.
- Resolved: add Dietary Presets in Phase 04 search as named bundles that produce Exclusion Rules. Keep Food Object classification based on Food Categories, Culinary Roles, and Allergens.
- Resolved: persist search history for authenticated users only after a completed search returns valid results. Keep duplicate searches, retain the latest 100 rows per user, expose clear-history behavior, and do not persist anonymous searches. Duplicate collapsing is deferred unless the UI becomes noisy.
- Resolved: keep rolling 24-hour usage counters PostgreSQL-backed for Phase 06. The timestamped usage rows and caller-supplied cutoff provide durable enforcement. Consider Redis only if measurements show entitlement checks are a bottleneck.
- Resolved: account deletion uses `pending -> processing -> completed|failed`, with `failed -> processing` permitted for retry and `completed` terminal. Store a sanitized transient, permanent, or unknown failure category. Retry transient failures automatically with exponential backoff up to 3 attempts. Require admin-triggered retry after investigation for permanent, unknown, or exhausted failures, and alert when requests fail or exhaust retries.
- Resolved provisionally: retain a minimal pseudonymous deletion receipt after account erasure for GDPR accountability. Store a random receipt ID, request and completion timestamps, final outcome, and sanitized failure category when applicable. Do not retain the deleted user ID, email, or account data. Use a provisional three-year retention period; pre-production legal review is tracked in `docs/implementation/01_PLAN.md`.
- Resolved: validate liquid macros as non-negative, but do not apply the solid `protein + carbohydrates + fat <= 100 g` rule to values stored per `100 ml`. Without density data, values above that threshold can be legitimate. Add external-import warnings for suspicious liquid values instead of rejecting them solely for exceeding `100 g`.
- Resolved: normalize mixed solid/liquid composite meals using required liquid density. Missing persisted density is invalid data and returns an error. Do not invent density conversions.

## Phase 02

### Assumptions

- Add dedicated security-audit persistence for request-correlated authentication, API error, CSRF, rate-limit, and future admin events. Keep it separate from the Phase 01 admin-audit table because the event shapes and fail-closed security-mutation behavior differ.
- Implement AES-256-GCM envelope encryption, key versions, key-loader interfaces, and a production GCP Secret Manager adapter boundary in Phase 02. Use explicit local test fixtures for development and tests. Defer wiring encryption into concrete PII repository fields until Phase 03 authentication and profile services define the plaintext service boundaries.
- Implement local structured JSON logging, in-memory metrics test sinks, emitted metric names, probe cadence, and alert-rule configuration in Phase 02. Defer deployed GCP Cloud Monitoring resources, notification channels, backup monitoring resources, and dashboards until Phase 09 production hardening.
- Resolved: Phase 02 rejects `MEALSWAPP_TRUST_PROXY=true` and ignores `X-Forwarded-Proto`. Phase 09 must deploy and verify restricted trusted ingress before forwarded-scheme handling can be implemented.
- Resolved: Phase 02 uses Fiber v2 CSRF middleware, binds tokens to Fiber sessions, and exposes `GET /api/v1/auth/csrf-token` for safe SPA token delivery. Every mutation route must explicitly choose CSRF protection or an exemption.
- Accepted: Phase 02 request deadlines are cooperative. Handlers and dependencies must honor context cancellation and propagate deadline errors. Non-cooperative handlers are defects.
- Resolved: encrypt `users.email`, `oauth_identities.email`, `oauth_identities.provider_user_id`, `user_profiles.display_name`, and persisted search-history query text at rest. Permit plaintext only at the narrow service boundaries that need it: authentication and account export/deletion for account email, OAuth linking and account export/deletion for OAuth identity fields, profile and account export for display name, and history and account export for search-history query text. Password hashes, password salts, token hashes, UUIDs, preferences, roles, and timestamps do not need an additional encryption envelope.
- Resolved: support normalized-email uniqueness and lookup with a deterministic keyed HMAC-SHA-256 digest stored alongside encrypted email. Use a dedicated, versioned lookup key that is separate from AES-256-GCM encryption keys and stored outside the database. Phase 03 must define lookup-key rotation and reindexing behavior before wiring encrypted account email persistence.
- Resolved: keep Fiber session storage in-process for local development and single-instance deployment. Before horizontally scaled deployment, wire the Fiber session store to the documented Redis session namespace so CSRF state is shared between API instances.
- Resolved: Phase 03 login, refresh, password-reset completion, and logout handlers must call `RegenerateAuthorizationState` or `InvalidateAuthorizationState` before returning success.

### Clarifications

- Resolved: Phase 03 treats verification as Login Method state and keeps `users.email_verified` as an account-level projection named `hasVerifiedLoginMethod` in `DESIGN-006`. Email-and-password methods require Mealswapp verification; External Login Identities rely on provider-asserted verification. Paid feature checks use the projection, which is true when at least one linked Login Method is verified.
- Resolved: Phase 04 treats Substitution Search as one operation with one or more Substitution Inputs in `DESIGN-001` and `DESIGN-002`. Adding input Food Objects refines the same Substitution Search. Multiple-input searches combine Food Quantities into one Macro Profile for Nutritional Similarity and do not apply per-input Culinary Role ordering. Contradictory filters and Exclusion Rule conflicts return user-facing `SearchRejection` feedback instead of failing silently.
