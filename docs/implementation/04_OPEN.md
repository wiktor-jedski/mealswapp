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
- Resolved: add optional `densityGramsPerMilliliter` and density provenance for liquids. Derive density from trusted USDA volume portions with gram weights when available, preferring `ml`, `cup`, `tbsp`, `tsp`, then `fl_oz`. Persist source provider, source food ID, and whether the value was imported, manually entered, or estimated. Do not silently assume `1 ml = 1 g`.
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
- Resolved: normalize mixed solid/liquid composite meals using liquid density when available. If any required liquid density is unavailable, retain full-recipe and per-serving nutrition but exclude the meal from normalized similarity ranking. Do not invent density conversions.

### Actions needed

- Obtain privacy-law review before production for the pseudonymous deletion-receipt fields and provisional three-year retention period.
