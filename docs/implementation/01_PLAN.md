# Mealswapp Implementation Phase Plan

## Summary

Build Mealswapp as a greenfield Svelte + Go/Fiber application in dependency order: foundations, data model, backend core, search value loop, frontend shell, paid/user features, admin curation, offline/error handling, and production readiness. This plan is
intended as the phase-level source for expanding docs/implementation/02_TASK_LIST.md into concrete tasks.

## Development Phases

### Phase 00: Repository Bootstrap

- Create frontend, backend, worker, database migration, and local development structure.
- Add Bun/Svelte/Tailwind setup, Go module, Fiber app skeleton, shared config loading, Docker/local service wiring, and baseline CI checks.
- Exit criteria: empty app boots locally, backend health endpoint responds, frontend renders shell, tests/check commands exist.

### Phase 01: Data Repository Foundation

- Implement ARCH-005 core entities, PostgreSQL schema, migrations, repository interfaces, unit conversion, macro normalization, tag model, micronutrient vocabulary, and seed data.
- Cover food items, meals, recipes, tags, users, preferences, entitlements, saved data, audit logs, and admin imports enough for later phases.
- Require positive `densityGramsPerMilliliter` and `densitySourceKind` for liquids. Use density when normalizing mixed solid/liquid composite meals. Missing persisted liquid density is invalid data and returns an error.
- Exit criteria: repository tests pass for CRUD, search primitives, tag filters, unit conversion, recipe macro summation, and micronutrient validation.

### Phase 02: API Gateway, Security, Errors, Observability Baseline

- Implement ARCH-010, ARCH-013, ARCH-014, and ARCH-017 foundations.
- Add versioned /api/v1/* routing, request IDs, timeouts, validation, CORS, security headers, CSRF hooks, structured errors, health/readiness, logging, and basic metrics.
- Establish the OpenAPI source of truth and frontend type-generation command for shared gateway envelopes, health/readiness, and `AppError` contracts. The command may remain a placeholder in this phase if no domain API contracts are ready to generate yet, but its intended location, inputs, outputs, and validation path must be documented.
- Lint the OpenAPI source of truth with Redocly CLI as contracts are added or changed.
- Exit criteria: all API responses use consistent envelopes/errors, protected mutations enforce middleware, health/readiness are testable.

### Phase 03: Authentication, Profile, Consent

- Implement ARCH-006, ARCH-008, and ARCH-015 minimum account flows.
- Add registration with consent, login/logout, refresh, password hashing, lockout, password reset, email verification hooks, profile/preferences, saved data, export, and account deletion coordination.
- Replace the Phase 02 `X-Test-User-ID` protected-route placeholder with verified JWT-cookie authentication. Derive authenticated user UUIDs only from validated token claims for authorization, audit metadata, and user-scoped rate limits; never trust client-supplied identity headers.
- Extend `InputNormalizer` with typed field-specific rules as account and profile controllers introduce new string inputs. Keep email normalization from Phase 02 and add only the fields used by these flows.
- Harden account-deletion processing before exposing it: enforce the selected status-transition rules, lock request rows during transitions, and add a worker claim query using `FOR UPDATE SKIP LOCKED` or an equivalent concurrency-safe approach.
- For account deletion, classify sanitized failures as transient, permanent, or unknown. Retry transient failures automatically with exponential backoff up to 3 attempts. Require admin-triggered retry after investigation for permanent, unknown, or exhausted failures, and alert when requests fail or exhaust retries.
- Retain a minimal pseudonymous deletion receipt after account erasure: random receipt ID, request and completion timestamps, final outcome, and sanitized failure category when applicable. Do not retain the deleted user ID, email, or account data. Use a provisional three-year retention period pending pre-production legal review.
- Use the installed `golang-security` agent skill during implementation and review. Add `go vet ./...`, `govulncheck ./...`, and `go test -race ./...` to the backend quality gate before completing this phase.
- Lint auth/profile OpenAPI contract additions with Redocly CLI.
- Exit criteria: authenticated session lifecycle works end to end; profile preferences persist; consent blocks registration when missing.

### Phase 04: Search, Similarity, Cache Core

- Implement ARCH-002, ARCH-003, and server-side ARCH-011 search cache.
- Add search/autocomplete endpoints, query parsing, pagination limit of 10, filters, Levenshtein ranking, cosine similarity, similarity tiers/assets, Redis cache keys, and graceful similarity degradation.
- Extend `InputNormalizer` with typed search-query normalization when the search controller is added.
- Add named dietary rules as search constraints composed from ingredient-tag inclusions and exclusions, for example allowing fish while excluding meat for `pescatarian`. Keep ingredient classification tag-based.
- Persist completed authenticated-user searches only after valid results are returned. Retain duplicate searches, cap history at the latest 100 rows per user, expose clear-history behavior, and do not persist anonymous searches.
- Implement required OpenAPI-to-frontend type generation for the first domain contracts, including `SearchRequest`, `SearchResponse`, autocomplete responses, search errors, and cache-related response metadata. This is the latest phase where type generation may remain incomplete, because Phase 05 frontend API work consumes these generated types.
- Lint search-domain OpenAPI contract additions with Redocly CLI before generating frontend types.
- Exit criteria: API supports single/replacement/diet query shapes; autocomplete order is deterministic; similarity threshold and sorting match design.

### Phase 05: Frontend Search Experience

- Implement ARCH-001 and ARCH-016 user-facing search shell.
- Add Svelte stores, TanStack Query API client, sidebar, search mode controls, macro toggles, autocomplete keyboard navigation, results grid, pagination, theme provider, responsive layout, placeholder images, and local query cache.
- Add Playwright browser coverage and `@axe-core/playwright` accessibility checks while building the search UI so keyboard and WCAG regressions are caught before Phase 09 hardening.
- Exit criteria: default search state, 150ms debounce, local cache LRU, responsive UI, light/dark persistence, and result rendering satisfy SW-REQ-001 through SW-REQ-015 and SW-REQ-089.

### Phase 06: Subscription and Entitlement Enforcement

- Implement ARCH-007.
- Add free/trial/paid entitlement model, 3-search free limit, mode gating, Stripe checkout/webhooks, webhook idempotency, trial creation on social login, and reconciliation job.
- Use the installed `golang-security` agent skill during implementation and review of entitlement enforcement, Stripe webhook verification, and billing endpoints.
- Use Stripe CLI sandbox forwarding and event triggers to verify webhook signatures, retries, duplicate delivery, and failure handling locally.
- Lint subscription/billing OpenAPI contract additions with Redocly CLI.
- Exit criteria: free users are limited to single mode and usage caps; trial/paid users unlock ingredient, meal, and diet features; webhook tests cover duplicate and failure cases.

### Phase 07: Diet Optimization Worker

- Implement ARCH-004.
- Add Redis-backed optimization jobs, LP constraint/objective construction, worker process, status polling, 30-second solver timeout, infeasible handling, and up to 3 alternatives.
- Add the dedicated saved-diet persistence model and enable `saved_items` rows with kind `saved_diet`; until this phase, repositories must reject attempts to save that reserved kind.
- Lint optimization-job OpenAPI contract additions with Redocly CLI.
- Exit criteria: API returns 202 with job ID, worker stores completed/failed results, and LP tests validate macro tolerance, exclusions, diversity penalty, and timeout behavior.

### Phase 08: Admin Curation and External Data

- Implement ARCH-009 and ARCH-012.
- Add admin-only endpoints/UI, external search proxy for USDA/OpenFoodFacts, normalization warnings, curated import, manual item CRUD, tag management, user admin actions, and audit persistence.
- Extend `InputNormalizer` with typed rules for admin-authored names and provider text introduced by curation flows.
- Normalize provider-specific serving-unit aliases to canonical repository units (`g`, `ml`, `oz`, `fl_oz`, or `serving`) at the external-import boundary before persistence.
- Warn during external import when liquid macro totals per `100 ml` look suspicious, but do not reject them solely for exceeding `100 g`; without density data, that threshold is not a valid hard constraint for liquids.
- Derive required liquid density from trusted USDA volume portions with gram weights when available, preferring `ml`, `cup`, `tbsp`, `tsp`, then `fl_oz`. Persist whether the value was imported, manually entered, or estimated. Keep source provider and source food ID optional for manual or estimated values. Do not silently assume `1 ml = 1 g`.
- Optimize curated-import micronutrient validation: replace per-item full active-vocabulary loading with supplied-key lookup such as `ListAllowed(ctx, keys)`, or load and reuse the active vocabulary once within an import workflow. Keep ordinary CRUD simple unless measurements justify sharing the optimized path.
- Use the installed `golang-security` agent skill during implementation and review of admin authorization, provider input handling, and audit persistence.
- Lint admin and external-data OpenAPI contract additions with Redocly CLI.
- Exit criteria: non-admin users receive 403; admins can search external sources, edit/tag/import items, and all mutations create audit entries.

### Phase 09: Offline, Degradation, Accessibility, Production Hardening

- Complete client ARCH-011 service worker behavior plus cross-cutting requirements.
- Add offline cached search/image behavior, stale indicators, retry manager integration, accessibility pass, Playwright browser coverage, monitoring alerts, backup/retention checks, and deployment config for GCP services.
- Extend Playwright and `@axe-core/playwright` coverage for offline, degradation, keyboard, responsive, and WCAG acceptance paths.
- Install and use the `gcp-cloud-run` agent skill when implementing Cloud Run deployment, restricted ingress, Cloud SQL, Memorystore, Secret Manager, and monitoring configuration.
- Enforce SW-REQ-091: when trusted forwarded-scheme handling is enabled, restrict direct application ingress to the configured reverse proxy or load balancer and verify that arbitrary public clients cannot reach the application instance or spoof `X-Forwarded-Proto`.
- Exit criteria: offline cached searches render, connection loss preserves state, WCAG/keyboard checks pass, performance and readiness gates are documented, and trusted-proxy deployment tests pass before enabling `MEALSWAPP_TRUST_PROXY=true`.

## Public APIs and Interfaces

- Backend exposes versioned REST under /api/v1: auth, profile/preferences, search/autocomplete, optimization jobs, subscription/billing, saved data, admin, external search, health, and readiness.
- OpenAPI/type generation is planned but not required immediately for Phase 00 or Phase 01.
- Shared request/response contracts should be generated from OpenAPI into frontend types. Phase 02 establishes the OpenAPI source of truth, type-generation command, and shared envelope/error contracts; Phase 04 must generate frontend types for `SearchRequest`, `SearchResponse`, autocomplete responses, search errors, and cache metadata before the Phase 05 frontend API client consumes them; later phases add generated auth/profile, entitlement, optimization, billing, saved-data, and admin import types as their APIs are implemented.
- Redis namespaces follow ARCH-011: search, item, similarity, session, job, and user, each with schema-versioned keys.
- Task list rows should map each task to one ARCH/DESIGN static aspect, using the phase ID in the description or traceability header.

## Test Plan

- Unit tests: repository validation, unit conversion, macro normalization, autocomplete ranking, cosine similarity, entitlement decisions, LP constraints, cache key stability, error classification.
- Integration tests: API middleware, auth/session flows, profile/preferences, search pagination/filtering, Stripe webhook idempotency, optimization job lifecycle, admin import workflow.
- Frontend tests: Svelte component state, debounce, localStorage LRU, keyboard navigation, theme persistence, responsive rendering, error/offline states.
- E2E tests: registration/login, basic search, replacement search, paid-mode gating, saved favorites/history, admin import, account export/deletion.
- Operational checks: python scripts/check.py, backend go test ./..., frontend bun test, Playwright suite, health/readiness, and migration up/down validation.
- Goal: 100% line coverage by the end of each phase. Each deviation from that has to be documented in AGENTS.md and accepted.
- After each phase suggest integration, functional, end to end and acceptance tests that have to be performed by the project owner.

## Assumptions

- Phase output will be written later as a documentation file, then expanded into docs/implementation/02_TASK_LIST.md.
- The first shippable MVP is Phase 05 plus minimal Phase 03 authentication; subscription, optimization, admin curation, and full production hardening can follow.
- External services use test/sandbox modes until Phase 09.
- No real secrets, provider credentials, Stripe keys, or production data are committed.
