# Development Setup

## Prerequisites

- Go 1.24 or newer
- Bun 1.3 or newer
- Docker Compose, or local PostgreSQL and Redis services

## Configuration

Copy `.env.example` into a local `.env` if shell-based loading is preferred. The application also works with built-in development defaults:

- `MEALSWAPP_HTTP_PORT=8080`
- `MEALSWAPP_DATABASE_URL=postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable`
- `MEALSWAPP_REDIS_URL=redis://localhost:6379/0`
- `MEALSWAPP_FRONTEND_ORIGIN=http://localhost:5173`

## Local Services

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

The Phase 00 frontend renders the application shell, theme selector, and disabled search placeholder. Search behavior is implemented in later phases.

## Backend

```sh
cd backend
go run ./cmd/migrate up
go run ./cmd/api
go test ./...
```

Health endpoints:

- `GET http://localhost:8080/health`
- `GET http://localhost:8080/ready`
- `GET http://localhost:8080/api/v1/health`
- `GET http://localhost:8080/api/v1/ready`

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

The aggregate check validates requirement traceability, formats and tests Go code, builds the frontend package, and runs Bun tests from `frontend/`.

## API Contract

The bootstrap OpenAPI document lives at `api/openapi.yaml`. It currently covers health and readiness endpoints. Type generation is intentionally deferred until domain API contracts are introduced.
