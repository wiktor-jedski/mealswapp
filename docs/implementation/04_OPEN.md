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

## Phase 03

### Assumptions

- Accepted for planning: normalized-email uniqueness and lookup use a versioned keyed HMAC-SHA-256 digest stored alongside encrypted email. Phase 03 defines lookup-key metadata and a digest reindex command or repository method so rotation can add a new digest version and rebuild existing account lookup digests without decrypting or logging PII outside the service boundary.
- Accepted for planning: OAuth first-login trial activation is represented in Phase 03 as an explicit no-op entitlement hook because ARCH-007 subscription and trial persistence are implemented in Phase 06. Phase 06 must replace this hook with real trial creation and entitlement reconciliation.
- Accepted for Phase 03.1: account export returns an empty-but-typed `customItems` section until the repository gains a user-owned custom item schema. Current `food_items` rows are global and do not include a `user_id` owner predicate, so exporting them as account data would violate user scoping. Do not invent fake custom item rows in real account exports.
- Phase 03 email verification is limited to an authenticated, CSRF-protected verification hook that updates the server-derived user only. Production email delivery and signed email-verification tokens are deferred until the email provider integration is introduced.
- Accepted for Phase 03.1: keep the Phase 02 Fiber failed-login IP limiter as the Phase 03 IP-level brute-force protection. Do not add a separate persisted IP failure counter in Phase 03.1 unless the design changes; persisted account lockout remains required.
- Resolved for Phase 03.1: account deletion is implemented with account write lockout, transactional production deletion and receipt persistence where possible, sanitized failure classification, exponential retry scheduling up to three attempts, admin-retry state for permanent/unknown/exhausted failures, alert metadata, session invalidation, cache purge handling, and pseudonymous receipt constraints.
- Accepted for Phase 03.1 evidence: generate separate `docs/implementation/implemented/03.1_PHASE_UAT.md` and `docs/implementation/implemented/03.1_PHASE_REPORT.html` artifacts so the original Phase 03 UAT/report remain available.
- Phase 03.1 production bootstrap uses `MEALSWAPP_LOCAL_SECRET_KEY` for local JWT signing, PII encryption, and deterministic lookup digest key material. Development has a local fallback; production fails closed unless the variable is set. Replace this local loader with the documented Secret Manager-backed key loaders before deployment.
- Accepted: `localKeyLoader` uses a static `"local-v1"` version. Since it only holds a single active key, configuring a dynamic version via environment variables would not enable testing multi-version key rotation (as it cannot serve historical keys concurrently). Multi-version key rotation testing is deferred to unit test mocks or the production Secret Manager key loader integration.
- Resolved: `localKeyLoader.LookupKey` and `localKeyLoader.SigningKey` now forward the incoming context to the shared key lookup instead of replacing it with `context.Background()`.
- Resolved: `AuthController.Refresh` now logs a warning when best-effort authenticated-cookie clearing fails after refresh-token rejection, while preserving the original authentication error response.
- Resolved: `httpapi.Controller` now formalizes the `Routes() []RouteDefinition` contract, all Phase 03 HTTP controllers have compile-time guards for it, and `NewProduction` registers routes by flattening a typed controller slice.
- Resolved: `GenericInvalidCredentialMessage` is now a constant instead of a function, keeping the failed-login message reusable without exposing a function-shaped API.
- Resolved: deterministic password test-fixture hash and salt generation now live in `password_test.go` instead of production `password.go`.
- Resolved: `parseHashParams` now ranges over `strings.SplitSeq` instead of allocating a slice with `strings.Split`.
- Resolved: `ExportBundle` no longer carries the transport-level `format` field; the export format enum now lives at the API query-parameter/type boundary.
- Resolved: fallback disclaimer Markdown strings now live in named package-level constants.
- Resolved: database repositories, HTTP controllers, and local infrastructure adapters were audited for compile-time interface guards. Missing guards were added for concrete PostgreSQL repositories, security audit logging, observability sinks, local key loading, OAuth fail-closed gateway, and Redis cache purging. Cross-package service-boundary guards that would introduce import cycles remain intentionally omitted.
- Resolved: the pre-production `users` schema no longer includes plaintext `email` or generated `normalized_email` columns. User email uniqueness and lookup now rely on the encrypted-email metadata and `normalized_email_digest`, and `encrypted_user_create.sql` no longer writes a placeholder legacy email value.
- Resolved: all current database mutation SQL was audited for idempotency and duplication safety. Registration, saved items, consent, profile creation, deletion requests, OAuth identity linking, curated imports, classification/vocabulary upserts, food item creation, Stripe event recording, and usage windows rely on unique constraints, `ON CONFLICT`, or state-machine transitions. Profile/password/verification/session revocation updates are absolute or repeat-safe; password-reset token consume and refresh-token reuse intentionally reject replay; audit/security/history rows and login counters are intentionally append-only event records. Raw create primitives such as meal creation and future admin/custom-item creation must not be exposed through retryable REST flows without the cross-phase `Idempotency-Key` standard now recorded in `docs/implementation/01_PLAN.md`; Phase 06 checkout/webhook and Phase 08 admin/custom-item creation are called out explicitly.
- Phase 03.1 production bootstrap composes auth, OAuth, profile, saved-data, export, account-deletion, disclaimer, CSRF, and JWT routes from real repositories. OAuth routes fail closed until Google/Apple provider credentials and callback exchange are configured.
- Deferred to Phase 08: add an explicit user-owned custom item persistence model before relying on `customItems` in account export or account deletion. Until then, Phase 03 account export keeps `customItems` empty and typed.
- Deferred to Phase 09: add signed, single-use email-verification tokens and outbound email delivery before production paid-feature unlocks can rely on email-and-password verification.
- Deferred to Phase 09: add and validate production Google and Apple OAuth provider gateway configuration before enabling live external login. Until then, Phase 03 production bootstrap fails OAuth routes closed.

### Security review notes

- `golang-security` review on 2026-06-03 found and fixed one verification authorization issue: `/api/v1/auth/verify-email` no longer trusts a client-supplied `userId`; it requires JWT cookies plus CSRF and marks only the authenticated user.
- Review confirmed account routes ignore identity headers, auth/profile/export/delete routes derive user scope from validated JWT/session state, refresh and reset tokens persist only hashes, PII fields cross plaintext boundaries only in auth/profile/export/delete services, and auth/CSRF cookies are HttpOnly with SameSite=Strict and Secure when TLS enforcement is enabled.

### Testing coverage deviations

- Phase 03 coverage deviation accepted on 2026-06-03: backend internal coverage is 90.3% after auth, OAuth, profile, export, deletion, and repository hardening. The uncovered lines are primarily defensive error branches, constructor defaults, and future-provider failure paths; Phase 04 should add targeted tests when those branches become active product behavior instead of inflating brittle tests solely to satisfy line coverage.

### Actions needed

No Phase 03 project-owner action is required at this time.

### Code Review Findings

No unresolved Phase 03 code review findings remain at this time.

## Phase 04

### Assumptions

- Accepted for planning: Dietary Presets are deterministic backend-owned named bundles that expand into Exclusion Rules at search time. They are not stored as Food Object classifications and should not create misleading Food Category or Culinary Role rows.
- Accepted for planning: Phase 04 supports the Daily Daily Diet Alternative Search request shape at the search API boundary, but does not implement Phase 07 LP optimization jobs or saved-diet persistence. When required Phase 07 data is unavailable, the API returns a deterministic user-facing `SearchRejection` instead of creating worker/job side effects.
- Task 115 implementation assumes defensive request-boundary limits of 200 runes for search queries, 120 runes for autocomplete queries, and maximum page `10000`; project owner should adjust these before public launch if product search UX requires different bounds.
- Task 115 implementation treats substitution units as compact letter, hyphen, or underscore tokens up to 32 runes and validates `dailyDietId` as UUID-shaped when present; required-vs-optional daily-diet semantics remain deferred to Task 124.
- Task 119 implementation uses schema versions `search-response-v1`, `autocomplete-response-v1`, and `similarity-calculation-v1`, with default TTLs of 5 minutes for search responses, 2 minutes for autocomplete responses, and 15 minutes for similarity calculations. DESIGN-011 requires schema versions and TTLs but does not specify concrete values.
- Task 117 implementation treats Dietary Preset rule IDs such as `dairy`, `gluten`, and `meat` as backend-owned Exclusion Rule names, not classification rows. Until a dedicated allergen persistence model exists, repository-backed allergen filters use existing classification association IDs as the only available persisted exclusion surface.
- Task 123 implementation keeps the current `SearchResponse` DTO stable: substitution results expose final ranking scores in `SimilarityScores`, while DESIGN-003 tier metadata remains verified through the internal `SimilarityResult` path until an API response field for tier/color/image metadata is designed.

### Testing coverage deviations

- Phase 04 coverage deviation accepted after Phase 04.1 coverage audit on 2026-06-12 for Task 134. Current measured backend internal line coverage is 88.9%; package highlights are `backend/internal/cache` 94.4%, `backend/internal/search` 89.3%, `backend/internal/httpapi` 85.5%, `backend/internal/repository` 95.8%, and frontend coverage remains covered by the aggregate check. The accepted exception is no longer a broad aggregate waiver; the remaining uncovered lines are classified below.
- Resolved Phase 04 active behavior gap: `backend/internal/cache/search_cache.go` `SearchResponseStore.GetSearchResponse`, `SetSearchResponse`, and default `ttl` behavior were active Redis search-cache behavior but were not directly covered. `backend/internal/cache/search_cache_test.go` now verifies miss/hit metadata, default TTL, and stripping transient cache metadata before persistence.
- Resolved Phase 03 carryover: Phase 03 auth/profile/export/deletion behavior that became relevant to Phase 04 authenticated search history is covered at the active Phase 04 boundary by `backend/internal/httpapi/search_controller_test.go` and `backend/internal/search/*` tests: authenticated successful search appends history with the server-derived user ID, anonymous/rejected/failed searches do not persist, daily-diet rejections avoid side effects, and cache-hit catalog searches bypass repository loading.
- Still-dormant Phase 03 defensive paths: `backend/internal/app/app.go` `CompleteOAuth`, local key-loader accessors, and `PurgeUser`; `backend/internal/auth/*` default password-hasher constructors, direct service `Register`/`Logout` paths, and custom `Error` stringers; `backend/internal/repository/account_verification_repository.go` password-reset/email-verification methods; `registration_repository.go`; `compliance_repository.go` deletion worker methods; `encrypted_identity_repository.go` encrypted profile/history helper methods; `session_repository.go` `RevokeUserSessions`; `userdata/deletion.go`; and `userdata/export.go` remain mostly dormant until later end-to-end account, OAuth, deletion-worker, and export flows are expanded. These are not Phase 04 search/cache/history behavior and are kept as specific carry-forward targets rather than inflated by branch-only tests.
- Active Phase 04 behavior with sufficient targeted coverage: `backend/internal/search` uncovered lines are low-frequency branches in daily-diet/search error stringers, duplicate-filter helper branches, canonical unit fallback, page normalization, and similarity tier/assets fallback; active success and rejection behavior is covered by parser, filter, catalog, substitution, daily-diet, similarity, and route integration tests. `backend/internal/httpapi/search_controller.go` uncovered `parsePage` and `parseQuantity` integer/string/default branches are behind validated JSON request bodies, while float64 JSON decoding is covered through HTTP tests; malformed and boundary request behavior is covered in `search_validation_test.go`.
- Explicit low-value defensive branches accepted for Phase 04: repository hydration/validation cleanup branches such as `food_repository.go` `physicalStatesToStrings`, `validateFoodItem`, `hydrateFoodClassifications`, and meal conversion helpers; database transaction `Begin`; auth controller warning/logging and validation fallbacks; and error `Error`/`Unwrap` helpers are retained as defensive code. Testing each remaining branch would require brittle artificial database, JSON-decoding, or impossible-state fixtures with little product-value gain in Phase 04.

### Actions needed

- No current Phase 04 aggregate-gate blocker remains after Task 134 audit verification on 2026-06-12. Future phases should add tests when any dormant account, OAuth, deletion-worker, export, repository, or defensive search branch becomes active product behavior.
