# Development Setup

## Prerequisites

- Go 1.24 or newer
- Bun 1.3 or newer
- Docker Compose, or local PostgreSQL and Redis services

## Configuration

Copy `.env.example` into a local `.env` if shell-based loading is preferred. The application also works with built-in development defaults:

- `MEALSWAPP_HTTP_PORT=8080`
- `MEALSWAPP_DATABASE_URL=postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable`
- `MEALSWAPP_TEST_DATABASE_URL=postgres://mealswapp:mealswapp@localhost:5432/mealswapp_test?sslmode=disable` for destructive PostgreSQL integration tests; test database names must end in `_test`.
- `MEALSWAPP_REDIS_URL=redis://localhost:6379/0`
- `MEALSWAPP_FRONTEND_ORIGIN=http://localhost:5173`
- `MEALSWAPP_ALLOWED_ORIGINS=http://localhost:5173`
- `MEALSWAPP_API_TIMEOUT=10s`
- `MEALSWAPP_ENFORCE_TLS=false`
- `MEALSWAPP_TRUST_PROXY=false`
- `MEALSWAPP_TLS_MIN_VERSION=1.3`

## Local Services

Start PostgreSQL, Redis, the backend, and the frontend together from the repository root:

```sh
bash scripts/start-dev.sh
```

The command starts local services, applies backend migrations, seeds deterministic
development data, and keeps the backend and frontend attached to the terminal.
Press `Ctrl-C` to stop both application processes. PostgreSQL and Redis remain
available for the next development session.

To start only PostgreSQL and Redis:

```sh
bash scripts/start-services.sh
```

This starts PostgreSQL and Redis through Docker Compose when Docker is available. The script falls back to system services on environments that provide `service`.

## Frontend

```sh
cd frontend
bun install
bun run dev
bun test
bun run build
```

The Vite dev server proxies `/api` to the backend at `http://127.0.0.1:8080`
(see `frontend/vite.config.ts`), so start the backend before `bun run dev`
when running the full search experience locally.

The Phase 00 frontend renders the application shell, theme selector, and disabled search placeholder. Search behavior is implemented in later phases.

## Backend

```sh
cd backend
go run ./cmd/migrate up
go run ./cmd/seed
go run ./cmd/api
go test ./...
```

Health endpoints:

- `GET http://localhost:8080/health`
- `GET http://localhost:8080/ready`
- `GET http://localhost:8080/api/v1/health`
- `GET http://localhost:8080/api/v1/ready`

Browser clients obtain a session-bound CSRF synchronizer token with
`GET http://localhost:8080/api/v1/auth/csrf-token`. The response body provides
the token for the `X-CSRF-Token` header. The matching CSRF and session cookies
are HttpOnly and must be sent by the browser with credentials enabled.

Phase 02 rejects `MEALSWAPP_TRUST_PROXY=true` and does not consume
`X-Forwarded-Proto`. Phase 09 owns restricted trusted ingress and TLS 1.3 edge
enforcement. Gateway deadlines are cooperative: handlers and dependencies must
honor request-context cancellation and return propagated deadline errors.

## Worker

```sh
cd backend
go run ./cmd/worker
```

The Phase 00 worker starts, checks Redis connectivity, and waits for shutdown. Job processing is implemented in Phase 07.

## Aggregate Checks

```sh
python3 scripts/check.py
```

The aggregate check validates requirement and design traceability, checks generated API types, formats and tests Go code, builds the frontend package, and runs Bun tests from `frontend/`.

## API Contract

The OpenAPI source of truth lives at `api/openapi.yaml`. It covers shared gateway envelopes, `AppError`, health, readiness, request IDs, retry metadata, and future cookie-auth plus CSRF hooks.

Generate or verify frontend contracts:

```sh
cd frontend
bun run generate:api-types
bun run check:api-types
```

Generated shared types live at `frontend/src/lib/api/generated.ts`. Phase 04 extends the same OpenAPI input with search-domain contracts before the Phase 05 API client consumes them.

## Observability Baseline

Phase 02 emits structured request logs with request ID, route template, method, status, and latency. Metrics cover request latency, concurrent requests, and PostgreSQL or Redis dependency health. Deployed monitoring should probe `/health` and `/ready` every 30 seconds and configure P95 latency warnings above 1.5 seconds and critical alerts above 2 seconds. GCP resources, notification channels, dashboards, and backup monitoring are deferred to Phase 09.
