# Phase 00 UAT - Repository Bootstrap

## Scope

Phase 00 establishes the empty Mealswapp application foundation:

- Frontend package in `frontend/` with Svelte, Bun, Vite, Tailwind, theme state, app shell, and service-worker bootstrap.
- Backend package in `backend/` with Go/Fiber app construction, config loading, request IDs, recovery, health/readiness endpoints, PostgreSQL and Redis connectivity wrappers, migration command, and worker bootstrap.
- Local development dependencies through Docker Compose and `scripts/start-services.sh`.
- Bootstrap OpenAPI contract in `api/openapi.yaml`.
- CI and aggregate verification through `python3 scripts/check.py`.
- Design traceability comments in generated source, with sidecar trace docs for JSON files.

## Automated Verification

Run from repository root:

```sh
python3 scripts/check.py
```

Expected result:

- Go formatting succeeds.
- Backend tests pass.
- Backend internal package coverage reports `total: ... 100.0%`.
- Frontend build succeeds.
- Frontend Bun tests pass.
- Frontend coverage reports `All files | 100.00 | 100.00`.

## Project Owner Acceptance Tests

### 1. Local Services Start

Steps:

```sh
bash scripts/start-services.sh
docker compose ps
```

Accept when:

- PostgreSQL and Redis containers are running.
- Both services become healthy.
- No production secrets are required.

### 2. Database Migration Runs

Steps:

```sh
cd backend
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/migrate up
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/migrate down
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/migrate up
```

Accept when:

- All commands exit successfully.
- Re-running `up` remains idempotent.
- The schema migration table exists after the final `up`.

### 3. Backend Health and Readiness

Steps:

```sh
cd backend
MEALSWAPP_HTTP_PORT=18080 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/api
```

In another terminal:

```sh
curl -sS http://127.0.0.1:18080/health
curl -sS http://127.0.0.1:18080/ready
curl -sS http://127.0.0.1:18080/api/v1/health
curl -sS http://127.0.0.1:18080/api/v1/ready
```

Accept when:

- Health responses return `status: "ok"`.
- Readiness responses return `status: "ready"`.
- Readiness checks include `postgres: "ok"` and `redis: "ok"`.
- Each response includes a non-empty `requestId`.

### 4. Frontend Shell Boots

Steps:

```sh
cd frontend
BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun install
BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run dev
```

Open `http://localhost:5173`.

Accept when:

- The page renders the Mealswapp shell.
- The sidebar shows `Single Item`, `Replacement`, and `Diet`.
- The search input is visible and disabled with Phase 00 placeholder text.
- The theme selector offers `System`, `Light`, and `Dark`.
- Switching themes changes the page theme without a reload.

### 5. Worker Starts and Stops

Steps:

```sh
cd backend
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run ./cmd/worker
```

Accept when:

- Worker logs startup with the current environment.
- Worker connects to Redis successfully.
- Pressing `Ctrl+C` stops the worker cleanly.

### 6. API Contract Is Present

Steps:

```sh
test -f api/openapi.yaml
rg "/health|/ready|/api/v1/health|/api/v1/ready" api/openapi.yaml
```

Accept when:

- The OpenAPI file exists.
- Health and readiness endpoints are represented for root and `/api/v1`.

### 7. Traceability Is Present

Steps:

```sh
rg "Implements DESIGN-" backend frontend api database scripts docker-compose.yml .github/workflows/ci.yml
find frontend -name '*-trace.md' -maxdepth 1 -type f -print
```

Accept when:

- Generated source and operational artifacts include relevant `Implements DESIGN-*` comments.
- JSON files have sidecar trace documents such as `frontend/package.json-trace.md`.

## Phase 00 Acceptance Decision

Phase 00 can be accepted when all automated verification and project-owner acceptance tests above pass in the project owner environment.

Known note:

- Backend `cmd/*` entrypoints are verified by build and smoke tests instead of line coverage because they bind ports, connect to local services, or execute process-level commands.
