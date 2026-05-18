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
- Exit criteria: repository tests pass for CRUD, search primitives, tag filters, unit conversion, recipe macro summation, and micronutrient validation.

### Phase 02: API Gateway, Security, Errors, Observability Baseline

- Implement ARCH-010, ARCH-013, ARCH-014, and ARCH-017 foundations.
- Add versioned /api/v1/* routing, request IDs, timeouts, validation, CORS, security headers, CSRF hooks, structured errors, health/readiness, logging, and basic metrics.
- Exit criteria: all API responses use consistent envelopes/errors, protected mutations enforce middleware, health/readiness are testable.

### Phase 03: Authentication, Profile, Consent

- Implement ARCH-006, ARCH-008, and ARCH-015 minimum account flows.
- Add registration with consent, login/logout, refresh, password hashing, lockout, password reset, email verification hooks, profile/preferences, saved data, export, and account deletion coordination.
- Exit criteria: authenticated session lifecycle works end to end; profile preferences persist; consent blocks registration when missing.

### Phase 04: Search, Similarity, Cache Core

- Implement ARCH-002, ARCH-003, and server-side ARCH-011 search cache.
- Add search/autocomplete endpoints, query parsing, pagination limit of 10, filters, Levenshtein ranking, cosine similarity, similarity tiers/assets, Redis cache keys, and graceful similarity degradation.
- Exit criteria: API supports single/replacement/diet query shapes; autocomplete order is deterministic; similarity threshold and sorting match design.

### Phase 05: Frontend Search Experience

- Implement ARCH-001 and ARCH-016 user-facing search shell.
- Add Svelte stores, TanStack Query API client, sidebar, search mode controls, macro toggles, autocomplete keyboard navigation, results grid, pagination, theme provider, responsive layout, placeholder images, and local query cache.
- Exit criteria: default search state, 150ms debounce, local cache LRU, responsive UI, light/dark persistence, and result rendering satisfy SW-REQ-001 through SW-REQ-015 and SW-REQ-089.

### Phase 06: Subscription and Entitlement Enforcement

- Implement ARCH-007.
- Add free/trial/paid entitlement model, 3-search free limit, mode gating, Stripe checkout/webhooks, webhook idempotency, trial creation on social login, and reconciliation job.
- Exit criteria: free users are limited to single mode and usage caps; trial/paid users unlock ingredient, meal, and diet features; webhook tests cover duplicate and failure cases.

### Phase 07: Diet Optimization Worker

- Implement ARCH-004.
- Add Redis-backed optimization jobs, LP constraint/objective construction, worker process, status polling, 30-second solver timeout, infeasible handling, and up to 3 alternatives.
- Exit criteria: API returns 202 with job ID, worker stores completed/failed results, and LP tests validate macro tolerance, exclusions, diversity penalty, and timeout behavior.

### Phase 08: Admin Curation and External Data

- Implement ARCH-009 and ARCH-012.
- Add admin-only endpoints/UI, external search proxy for USDA/OpenFoodFacts, normalization warnings, curated import, manual item CRUD, tag management, user admin actions, and audit persistence.
- Exit criteria: non-admin users receive 403; admins can search external sources, edit/tag/import items, and all mutations create audit entries.

### Phase 09: Offline, Degradation, Accessibility, Production Hardening

- Complete client ARCH-011 service worker behavior plus cross-cutting requirements.
- Add offline cached search/image behavior, stale indicators, retry manager integration, accessibility pass, Playwright browser coverage, monitoring alerts, backup/retention checks, and deployment config for GCP services.
- Exit criteria: offline cached searches render, connection loss preserves state, WCAG/keyboard checks pass, performance and readiness gates are documented.

## Public APIs and Interfaces

- Backend exposes versioned REST under /api/v1: auth, profile/preferences, search/autocomplete, optimization jobs, subscription/billing, saved data, admin, external search, health, and readiness.
- Shared request/response contracts should be generated or mirrored from Go/OpenAPI into frontend types for SearchRequest, SearchResponse, AppError, Entitlement, UserProfile, OptimizationJob, and admin import types.
- Redis namespaces follow ARCH-011: search, item, similarity, session, job, and user, each with schema-versioned keys.
- Task list rows should map each task to one ARCH/DESIGN static aspect, using the phase ID in the description or traceability header.

## Test Plan

- Unit tests: repository validation, unit conversion, macro normalization, autocomplete ranking, cosine similarity, entitlement decisions, LP constraints, cache key stability, error classification.
- Integration tests: API middleware, auth/session flows, profile/preferences, search pagination/filtering, Stripe webhook idempotency, optimization job lifecycle, admin import workflow.
- Frontend tests: Svelte component state, debounce, localStorage LRU, keyboard navigation, theme persistence, responsive rendering, error/offline states.
- E2E tests: registration/login, basic search, replacement search, paid-mode gating, saved favorites/history, admin import, account export/deletion.
- Operational checks: python scripts/check.py, backend go test ./..., frontend bun test, Playwright suite, health/readiness, and migration up/down validation.

## Assumptions

- Phase output will be written later as a documentation file, then expanded into docs/implementation/02_TASK_LIST.md.
- The first shippable MVP is Phase 05 plus minimal Phase 03 authentication; subscription, optimization, admin curation, and full production hardening can follow.
- External services use test/sandbox modes until Phase 09.
- No real secrets, provider credentials, Stripe keys, or production data are committed.
