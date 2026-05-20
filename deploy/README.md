# Deploy

This directory contains deployment assets for GCP Cloud Run, Cloud SQL,
Memorystore, Secret Manager, frontend hosting, monitoring, and CI/CD.

This guide covers a local deployment from source. Locally, PostgreSQL and Redis
run as services, while the API, worker, and frontend run from the repository.

## Local Deployment

### 1. Install prerequisites

Install these tools before starting:

- Docker with Docker Compose
- Go
- Bun
- PostgreSQL client tools, including `psql`

Check that they are available:

```bash
docker compose version
go version
bun --version
psql --version
```

### 2. Prepare environment variables

Create a local environment file from the example:

```bash
cp .env.example .env
```

The default local values are:

```bash
APP_ENV=local
API_ADDR=:8080
DATABASE_URL=postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable
REDIS_URL=redis://localhost:6379/0
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
POSTGRES_DB=mealswapp
POSTGRES_USER=mealswapp
POSTGRES_PASSWORD=mealswapp
```

Load them into each shell that runs backend commands:

```bash
set -a
. ./.env
set +a
```

Do not put real production secrets in `.env`.

### 3. Start PostgreSQL and Redis

From the repository root, start the local backing services:

```bash
docker compose up -d postgres redis
```

Wait until both services are healthy:

```bash
docker compose ps
```

You can also use the helper script, which starts the same services and waits for
readiness:

```bash
bash scripts/start-services.sh
```

### 4. Install frontend dependencies

From the frontend directory:

```bash
cd frontend
bun install
cd ..
```

Go dependencies are resolved by the Go toolchain when backend commands run.

### 5. Apply database migrations

The repository stores PostgreSQL migrations as ordered SQL files in
`db/migrations/`. Apply the `*.up.sql` files in order:

```bash
for migration in db/migrations/*.up.sql; do
  echo "Applying ${migration}"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$migration"
done
```

This is safe for a fresh local database. If you need to reset local data, remove
the Compose volume and start again:

```bash
docker compose down -v
docker compose up -d postgres redis
```

Then rerun the migration command.

### 6. Seed local data

Apply idempotent seed data after migrations:

```bash
cd backend
go run ./cmd/seed
cd ..
```

The seed command uses `DATABASE_URL` and inserts local development data such as
sample food items, tags, an admin user, and a recipe.

### 7. Run the API

Open a new shell, load `.env`, and start the API:

```bash
set -a
. ./.env
set +a
cd backend
go run ./cmd/api
```

The API listens on `http://localhost:8080` by default.

Verify it from another shell:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

### 8. Run the worker

Open another shell, load `.env`, and start the worker:

```bash
set -a
. ./.env
set +a
cd backend
WORKER_IDLE=true go run ./cmd/worker
```

`WORKER_IDLE=true` keeps the worker process running for local signal handling and
readiness checks. Without it, the worker initializes and exits in local mode.

### 9. Run the frontend

Open another shell and start Vite:

```bash
cd frontend
bun run dev
```

Open the application at:

```text
http://localhost:5173
```

The frontend is configured for the local API origin through
`CORS_ALLOWED_ORIGINS`.

### 10. Validate the deployment

Run the repository check script before considering the local deployment ready:

```bash
python scripts/check.py
```

This validates documentation traceability, Docker Compose config, backend tests,
frontend tests and build, deployment contracts, monitoring, backup policy, and
migration file naming.

## Useful Commands

Show local service status:

```bash
docker compose ps
```

View backing service logs:

```bash
docker compose logs -f postgres redis
```

Stop local backing services while keeping data:

```bash
docker compose down
```

Stop local backing services and delete local database data:

```bash
docker compose down -v
```

Run backend tests:

```bash
cd backend
GOCACHE="$PWD/.go-cache" go test ./...
```

Run frontend tests and build:

```bash
cd frontend
bun test
bun run build
```

## Troubleshooting

If `psql` cannot connect, confirm PostgreSQL is healthy with `docker compose ps`
and that `DATABASE_URL` is loaded in the current shell.

If `/ready` returns an error, check that migrations have been applied, seed data
has run successfully, and Redis is reachable at `redis://localhost:6379/0`.

If the frontend cannot call the API, confirm the API is running on port `8080`
and that `CORS_ALLOWED_ORIGINS` includes `http://localhost:5173`.

If ports `5432`, `6379`, `8080`, or `5173` are already in use, stop the
conflicting process or adjust the relevant Compose port, `API_ADDR`, or Vite
port before starting services again.
