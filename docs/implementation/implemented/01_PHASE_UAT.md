# Phase 01 UAT - Repository Foundation

## Scope

Phase 01 implements the backend repository foundation for Mealswapp:

- PostgreSQL migrations for food items, tags, micronutrient vocabulary, meals, recipes, users, profiles, saved data, entitlements, consent/deletion, curated imports, and admin audit.
- Repository contracts and PostgreSQL implementations for food, meals, tags, vocabulary, user profiles, saved data, entitlements, compliance, curated imports, and admin audit.
- Macro normalization, unit conversion, search primitives, recipe macro calculation, and deterministic seed data.
- Development seed command at `backend/cmd/seed`.
- Phase 01 quality reporting at `docs/implementation/implemented/01_PHASE_REPORT.html` with frontend verification screenshots in `docs/implementation/implemented/screenshots/`.

## Automated Verification

Run from repository root:

```sh
python3 scripts/check.py --output docs/implementation/implemented/01_PHASE_REPORT.html
```

Verified result:

- Requirement and design traceability validation passed.
- Local PostgreSQL and Redis stack verification passed.
- Migration `up`, `down`, and `up` sequence passed.
- Backend health and readiness smoke tests passed.
- Frontend screenshot verification passed for desktop and mobile.
- `go fmt ./...` passed.
- `go test ./...` passed.
- Backend internal package coverage reported `total: ... 100.0%`.
- Frontend build, unit tests, and coverage passed with `All files | 100.00 | 100.00`.

## Project Owner Acceptance Tests

### 1. Database Foundation

Steps:

```sh
bash scripts/start-services.sh
cd backend
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/migrate down
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/migrate up
```

Accept when:

- All Phase 01 tables and indexes are created.
- Re-running `up` is safe.
- Running `down` removes Phase 01 schema cleanly.

### 2. Repository Test Gate

Steps:

```sh
cd backend
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/... -coverprofile=coverage.out
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go tool cover -func=coverage.out
```

Accept when:

- All internal package tests pass.
- Coverage total is `100.0%`.
- Repository integration tests cover CRUD, search, saved data, entitlements, consent/deletion, curated imports, and audit rollback behavior.

### 3. Seed Data

Steps:

```sh
cd backend
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/seed
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/seed
```

Accept when:

- Both runs succeed.
- Seeded users use fixture-only non-secret password hashes.
- Seeded foods, tags, meals, entitlements, saved data, consent, curated import, and audit rows are not duplicated.

### 4. Search and Recipe Behavior

Steps:

```sh
cd backend
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/repository -run 'TestPostgres(FoodItemRepositorySearch|MealRepositorySearch|MealRepositorySingleRecipeAndMacros)'
```

Accept when:

- Food and meal search return deterministic paginated results.
- Name, tag, macro, prep-time, and deleted-row filters behave as expected.
- User-provided search text is parameterized.
- Seeded and test recipe macro sums match calculated values.

### 5. Compliance and Audit Workflows

Steps:

```sh
cd backend
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/repository -run 'TestPostgresComplianceAndAdminRepositories|TestPostgresEntitlementRepository|TestPostgresUserDataRepositories'
```

Accept when:

- Consent persistence is idempotent by user/version pair.
- Deletion requests keep auditable status transitions.
- User data reads and writes stay scoped by server-supplied user ID.
- Entitlement history is preserved when status changes.
- Admin audit writes can rollback mutating work when audit persistence fails.

## Phase 01 Acceptance Decision

Phase 01 can be accepted when the automated verification command and the project-owner acceptance tests above pass in the project owner environment.

Known notes are tracked in `docs/implementation/04_OPEN.md` under Phase 01.
