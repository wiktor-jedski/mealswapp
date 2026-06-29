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

- Backend coverage is measured and reported by `python3 scripts/check.py` with `go test ./internal/... -coverprofile=coverage.out`; the aggregate check does not enforce a 100% threshold. Deviations are reviewed and recorded per phase instead of being hidden behind a pass/fail percentage gate.
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
- Accepted for planning: Phase 04 supports the Daily Diet Alternative Search request shape at the search API boundary, but does not implement Phase 07 LP optimization jobs or saved-diet persistence. When required Phase 07 data is unavailable, the API returns a deterministic user-facing `SearchRejection` instead of creating worker/job side effects.
- Task 115 implementation assumes defensive request-boundary limits of 200 runes for search queries, 120 runes for autocomplete queries, and maximum page `10000`; project owner should adjust these before public launch if product search UX requires different bounds.
- Task 115 implementation validates substitution units as canonical API units only: `g`, `ml`, `oz`, and `fl_oz`. It also validates `dailyDietId` as UUID-shaped when present; required-vs-optional daily-diet semantics remain deferred to Task 124.
- Task 119 implementation uses schema versions `search-response-v1`, `autocomplete-response-v1`, and `similarity-calculation-v1`, with default TTLs of 5 minutes for search responses, 2 minutes for autocomplete responses, and 15 minutes for similarity calculations. DESIGN-011 requires schema versions and TTLs but does not specify concrete values.
- Task 117 implementation treats Dietary Preset rule IDs such as `dairy`, `gluten`, and `meat` as backend-owned Exclusion Rule names, not classification rows. Until a dedicated allergen persistence model exists, repository-backed allergen filters use existing classification association IDs as the only available persisted exclusion surface.
- Task 123 implementation keeps the current `SearchResponse` DTO stable: substitution results expose final ranking scores in `SimilarityScores`, while DESIGN-003 tier and image metadata remains verified through the internal `SimilarityResult` path until the public API response field is finalized.
- Task 138 OpenAPI extensions are now implemented: `api/openapi.yaml` `FoodObject` schema includes classification summaries (`id`, `name`, `kind`), an explicit primary Food Category, protein/carbohydrate/fat macros with `100g`/`100ml` basis, and non-negative server-calculated calories. Backend `foodObjectDTO`, `foodItemsData` mapper, and exported `search.CalculateCalories` populate the fields. Frontend generated types regenerated and drift check passes. Task 146 is unblocked.

### Testing coverage deviations

- Phase 04 coverage audit updated on 2026-06-18: backend internal line coverage is 99.2% using `go test ./internal/... -coverprofile=coverage.out`. `app`, `cache`, `compliance`, `config`, `database`, `migrations`, `observability`, `profile`, `repository`, `search`, `seed`, and `worker` are at 100%; `auth` is 99.2%, `httpapi` 97.0%, `security` 99.5%, and `userdata` 99.3%.
- `cmd/*` remains intentionally outside the coverage profile. Command packages are process bootstrap code that bind ports, connect to external services, or run migrations; they remain verified by build, migration, and API smoke checks.
- Remaining non-HTTP gaps are implementation-impossible error guards under current concrete types: JSON marshaling fixed internal DTOs, the standard-library hash writer returning an error, and the default password-hasher constructor rejecting its own compile-time-valid defaults. Adding injection solely to force these branches would increase production complexity without testing product behavior.
- Remaining `httpapi` gaps are primarily error returns from Fiber response serialization/cookie writes, observability fallback reporting, `runtime.Caller` failure, and middleware-preempted controller guards. Active authentication, OAuth, profile, export, deletion, user-data, search, cache, validation, repository-failure, and malformed-input paths are covered. These lines are deferred unless their collaborators gain practical injectable failure modes; they are not exempted from future targeted tests when behavior changes.

### Actions needed

- Task 138 OpenAPI extensions are now implemented: `api/openapi.yaml` `FoodObject` schema includes classification summaries (`id`, `name`, `kind`), an explicit primary Food Category, protein/carbohydrate/fat macros with `100g`/`100ml` basis, and non-negative server-calculated calories. Backend `foodObjectDTO`, `foodItemsData` mapper, and exported `search.CalculateCalories` populate the fields. Frontend generated types regenerated and drift check passes. Task 146 is unblocked.
- Remaining Task 138 description items (similarity presentation cleanup, deterministic ordering cleanup, naming cleanup, focused test hygiene) are intentionally split and not tracked as 04_OPEN.md action points.

### Code Review Findings

- No unresolved Phase 04 code review findings remain at this time.

## Phase 05

### Assumptions

- Phase 05 uses TanStack Query for server state and localStorage for the SW-REQ-003 cache of the 20 most recent unique normalized request/result pairs. Cache recency is updated on reads and writes; malformed or schema-version-mismatched entries are discarded. Phase 09 remains responsible for service-worker API/image interception and broader offline hardening.
- Phase 05 renders authenticated history and favorites in the Activity Sidebar from the existing Phase 03 generated contracts. Anonymous users receive empty/sign-in guidance, and activity API failure does not block public Catalog Search.
- Task 139 implementation added pinned `@tanstack/svelte-query@6.1.34`, `@playwright/test@1.61.0`, and `@axe-core/playwright@4.11.3` to `frontend/package.json` with `preview` and `test:e2e` scripts. `bunfig.toml` scopes `bun test` to `src/` to keep Playwright specs under `tests/` out of the unit test runner. The `check` script intentionally excludes `test:e2e` to keep the unit/build/drift gate deterministic and not dependent on browser binaries.
- Project-owner decision: Phase 05 does not implement macro visibility toggles. Result cards always show the required protein, carbohydrate, and fat values from the generated search contract; SettingsPanel scope is limited to unit and theme preferences.
- Project-owner decision: Phase 05 keeps the Daily Diet Alternative UI as a scaffold only. Full Daily Diet Alternative execution, field-level rejection UX, saved-diet data, and optimization job behavior move to Phase 07.
- Project-owner decision: the sidebar theme control intentionally exposes only explicit light/dark switching. The underlying ThemeProvider still supports `system` for defaults, invalid-value fallback, and live system-theme resolution, but a visible `system` sidebar option is out of scope.
- Task 142 implementation maps HTTP 429 to `server` category and 422 to `validation` category (code `search_rejected`) since the generated `ErrorCategory` enum has no `rate_limit`/`rejection` category. This stays within the generated contract.
- Task 143 component tests use static-source assertions rather than DOM rendering because no DOM library (jsdom/happy-dom) is installed in the Bun environment. `vite build` validates Svelte source compilation. A future phase may add happy-dom + render tests for stronger behavioral coverage.
- Task 145 implementation added `autocomplete-controller.ts` as a helper module (injectable timers/fetch) because Bun lacks Jest-style `useFakeTimers` for `setTimeout`. This enables deterministic 150ms debounce testing. Playwright autocomplete flows are `test.skip` scaffolds pending Task 151 wiring the dropdown into `SearchShell`.
- Task 147 implementation uses `GET /api/v1/profile` to detect signed-in state (401 = anonymous, no error), `GET /api/v1/search-history` for history, and `GET /api/v1/saved-items?kind=favorite` for favorites. All fetches use `credentials: "include"` and inline try/catch error handling that never propagates to the parent so core search stays usable.
- Task 148 implementation is SSR-safe and delegates staleness to Task 141's `LocalQueryCache.isStale`. `OfflineBanner` is not yet wired into `SearchShell` (Task 151 scope). Tests include explicit Phase 09 service-worker non-coverage disclaimers.
- Task 149 implementation extends the Phase 00 `theme.ts` store (preserving the public API) with live `matchMedia` system-theme subscription, `cleanupTheme()` for listener teardown, and storage-unavailable try/catch fallback. `App.svelte` calls `initTheme()` at startup but not `cleanupTheme()` — Task 151 should confirm no double-subscription on route changes.

### Clarifications

- Resolved by project-owner decision: Task 138 extends each search-result item with server-derived classification summaries (`id`, `name`, and `kind`), an explicit primary Food Category, protein/carbohydrate/fat macros with a `100g` or `100ml` basis, and calories. OpenAPI and generated frontend types must expose these fields before Task 146 implements result cards and category placeholders.
- Resolved by project-owner decision: SW-REQ-006 multi-meal Daily Diet aggregation moves to Phase 07 alongside the saved-diet model and optimization worker. Phase 05 keeps only the Daily Diet Alternative scaffold and does not claim Daily Diet Alternative execution or field-level rejection UX compliance.
- Step B UI iteration adds `GET /api/v1/food-objects/{id}` as a small UX support endpoint so autocomplete-selected Substitution Inputs can hydrate into the same rich FoodObject card view as Catalog-added items. The endpoint reuses the existing FoodObject DTO/OpenAPI schema and does not change autocomplete ranking or substitution-search behavior.

### Testing coverage deviations

- None accepted during planning. Phase 05 targets 100% line coverage for testable frontend source; any implementation-time exception must identify the specific file/function and rationale here.
- Task 143, 144, 147 component tests use static-source assertions rather than DOM rendering because no DOM library (jsdom/happy-dom) is installed in the Bun environment. `vite build` validates Svelte source compilation. A future phase may add happy-dom + render tests for stronger behavioral coverage.
- Task 153 closed the remaining `.ts` line- and function-coverage gaps so `bun test --coverage` reports `All files | 100.00 | 100.00` (functions and lines). Targeted tests added: `AutocompleteController` dispose-while-in-flight abort, `currentQuery` getter, and default `setTimeout`/`clearTimeout` fallback arrows (`autocomplete-controller.ts`); `updateSubstitutionInput` no-match branch and `compareFilter`/`compareSubstitutionInput` equal-id comparators (`search.ts`); `readSystemTheme`/`ensureSystemThemeSubscription` `matchMedia`-unavailable fallbacks and the subscribe-exactly-once early return (`theme.ts`). Svelte `.svelte` components remain outside Bun's line-coverage report and are verified by static-source assertions plus Playwright e2e (75 passed, 1 scaffold skipped) and `vite build`.

### Accepted accessibility deviations (Task 152)

- Task 152 automated axe scans (WCAG 2.1 A/AA, `frontend/tests/accessibility.spec.ts`) report `color-contrast` (serious) violations on decorative elements that use `text-white` on mid-tone backgrounds. These are accepted visual-design limitations, not normal reading-text pairs:
  - ResultCard similarity tier badges (`ResultCard.svelte` `tierStyles`): the "Fair" badge `bg-[var(--color-accent)] text-white` fails in both light and dark themes; the "Excellent"/"Good" badges (`bg-[var(--color-primary)]`) and the "Poor" badge (`bg-[var(--color-muted)]`) fail in dark theme.
  - ResultCard category chips (`bg-[var(--color-muted)] text-white`) and the image placeholder text (`text-white` on `bg-[var(--color-muted)]`) fail in dark theme.
  - SidebarComponent active search-mode button (`bg-[var(--color-primary)] text-white`) fails in dark theme.
- The gate asserts the ONLY serious/critical axe violations are these `color-contrast` cases, then re-runs axe with `color-contrast` disabled to confirm the rest of the composed shell is clean. Normal reading-text pairs (body `--color-text` and muted `--color-muted` labels on `--color-bg`/`--color-surface`) are explicitly verified to meet WCAG 2.1 AA 4.5:1 in both light and dark themes.
- Follow-up: a future visual-design pass should introduce theme-aware on-accent/on-muted text tokens (or darker/lighter badge backgrounds) so the decorative badges, chips, placeholder, and active sidebar button meet 4.5:1 in both themes. This is a design change, not a Task 152 minor a11y fix (aria-labels, focus styles, and labels were already in place from earlier Phase 05 tasks).

### Actions needed

- `frontend/src/lib/stores/search.ts` currently uses one broad `SearchState` interface with mode-specific fields (`substitutionInputs`, `dailyDietId`) guarded by `setMode()` and `buildSearchRequest()`. This keeps Phase 05 wiring simple, but invalid cross-mode states remain representable. Before Phase 07 implements full Daily Diet behavior, review whether search state should become a discriminated union or nested per-mode state so impossible Catalog/Substitution/Daily Diet combinations are prevented by types instead of only by transition helpers and tests.
- `frontend/src/lib/api/search-client.ts` `categoryForStatus()` maps `400`, `404`, and `422` to the same `validation` category, but groups only `400` and `422` while leaving `404` as a separate identical branch. Clean this up by grouping all validation statuses together, or add a comment if `404` is intentionally kept separate because missing food-object detail is conceptually different from request/search validation.
- Scan handwritten frontend `.ts` files, starting with `frontend/src/lib/api/search-client.ts`, and add missing TSDoc comments plus precise `docs/design` implementation traceability comments where functions/types implement design surfaces. Exclude generated files such as `frontend/src/lib/api/generated.ts`; their traceability belongs in the sidecar document.
- `frontend/src/lib/components/AutocompleteDropdown.svelte` `onOptionClick(item, index)` does not use the `item` parameter; simplify the handler to accept only `index` and update the click binding to avoid carrying an unused argument.
- `frontend/src/lib/components/SubstitutionInputs.svelte` hardcodes allergen, dietary preset, and physical-state filter options in the component. This is acceptable as Phase 05 UI scaffolding, but it couples frontend rendering to backend-owned filter IDs and domain vocabulary. Before admin curation/localization or broader filter management work, introduce a backend-owned filter-option source, ideally exposed through an API such as `/api/v1/search/filter-options?mode=substitution`, with persisted vocabularies for classifications/allergens and backend-defined policy for dietary presets; keep the frontend responsible for rendering and merging selected-item classification options.
- `frontend/src/lib/components/OfflineBanner.svelte` and `frontend/src/lib/stores/offline.ts` model both `showingCached` and `showingStale`, but `SearchResults.svelte` appears to update only `showingCached`. Verify the intended offline result states and wire stale-cache detection through `setShowingStale()` when local cached data exists but exceeds `LOCAL_CACHE_STALE_MS`, or remove the stale state/message if Phase 05 does not actually support it.
- `frontend/src/lib/components/SidebarComponent.svelte` mobile sidebar trigger currently renders the visible text `Activity` next to the menu icon. Remove the visible label and leave the icon-only button while preserving an accessible `aria-label` such as `Open activity sidebar`.
- `frontend/src/lib/components/AutocompleteDropdown.svelte` search input placeholder strings can be cut off awkwardly on mobile. Add responsive truncation (`text-overflow: ellipsis`/Tailwind `truncate`) and, if native input placeholder ellipsis remains inconsistent across browsers, introduce shorter mobile-specific placeholder strings so the search bar stays polished at narrow widths.
- Audit empty frontend `catch` branches in handwritten `.ts` and `.svelte` files. For each branch, confirm the swallowed error is intentional, document the fallback behavior in a short comment where useful, and verify the webpage remains usable when storage, browser APIs, profile/history/favorites calls, or other optional dependencies fail.
- Review Svelte component reactivity style for consistency. Some components use classic reactive declarations such as `$: activeMode = $searchStore.mode`, while others use Svelte 5 runes such as `$state` and `$derived`; choose a consistent style per component or document why both forms are intentionally mixed.
- Run a full LSP diagnostic pass across handwritten frontend `.svelte` files and fix reported Svelte/TypeScript errors. If any diagnostics are language-server configuration false positives while `vite build` and tests are correct, document the root cause and required Svelte/Vite/tsconfig adjustment instead of leaving editor-only errors unexplained.
- Investigate the LSP type mismatch reported for `App.svelte` `registerServiceWorker({ enabled: import.meta.env.PROD })`. `vite build` passes, but the language server may not be loading Vite `ImportMetaEnv` typings for `.svelte` files; verify whether a Vite environment declaration or tsconfig adjustment is needed, and keep any JSON traceability sidecar updates aligned if config files change.
