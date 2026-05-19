# Backend

Go/Fiber API application.

## Commands

- `go test ./...`
- `go run ./cmd/api`
- `go run ./cmd/worker`
- `DATABASE_URL=postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable go run ./cmd/seed`

The API listens on `API_ADDR` when set, otherwise `:8080`.
The worker performs no-op Redis/job initialization and exits by default in local mode. Set `WORKER_IDLE=true` to keep it running for signal handling checks.

Copy `.env.example` from the repository root when local dependency URLs are needed. `DATABASE_URL` and `REDIS_URL` are read from the environment by the shared config loader.
