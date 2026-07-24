#!/usr/bin/env bash
set -euo pipefail

# Implements DESIGN-009 UserAdminPanel real browser/API/PostgreSQL verification for task 261.
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fixture_dir="$(mktemp -d)"
api_pid=""

cleanup() {
  if [[ -n "$api_pid" ]]; then
    kill "$api_pid" 2>/dev/null || true
    wait "$api_pid" 2>/dev/null || true
  fi
  rm -r "$fixture_dir"
}
trap cleanup EXIT

cd "$repo_dir"
bash scripts/start-services.sh
mkdir -p logs

cd backend
GOCACHE="$PWD/.go-cache" GOMODCACHE="$PWD/.go-mod-cache" go run ./cmd/migrate up
GOCACHE="$PWD/.go-cache" GOMODCACHE="$PWD/.go-mod-cache" go build -o "$fixture_dir/mealswapp-api" ./cmd/api
MEALSWAPP_ENV=development MEALSWAPP_HTTP_PORT=8080 MEALSWAPP_DATABASE_URL='postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable' MEALSWAPP_REDIS_URL='redis://localhost:6379/0' "$fixture_dir/mealswapp-api" > "$repo_dir/logs/task-261-api.log" 2>&1 &
api_pid=$!

for _ in {1..80}; do
  if curl --fail --silent http://127.0.0.1:8080/health >/dev/null; then break; fi
  if ! kill -0 "$api_pid" 2>/dev/null; then
    sed -n '1,200p' "$repo_dir/logs/task-261-api.log"
    exit 1
  fi
  sleep 0.25
done
curl --fail --silent http://127.0.0.1:8080/health >/dev/null

cd ../frontend
MEALSWAPP_TASK261_REAL_E2E=1 BUN_TMPDIR="$PWD/.bun-tmp" BUN_INSTALL="$PWD/.bun-install" bunx playwright test -c playwright.real-stack.config.ts tests/task261-real-admin-flow.spec.ts
