#!/usr/bin/env bash
# Implements DESIGN-001 SearchView and DESIGN-010 RouteHandler local development startup.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_PID=""
FRONTEND_PID=""

cleanup() {
    trap - EXIT INT TERM
    [[ -n "$BACKEND_PID" ]] && kill "$BACKEND_PID" 2>/dev/null || true
    [[ -n "$FRONTEND_PID" ]] && kill "$FRONTEND_PID" 2>/dev/null || true
    wait "$BACKEND_PID" "$FRONTEND_PID" 2>/dev/null || true
}

trap cleanup EXIT INT TERM

for command in go bun; do
    if ! command -v "$command" >/dev/null 2>&1; then
        echo "Required command not found: $command" >&2
        exit 1
    fi
done

bash "$ROOT_DIR/scripts/start-services.sh"

echo "Applying backend migrations..."
(
    cd "$ROOT_DIR/backend"
    go run ./cmd/migrate up
)

echo "Seeding development data..."
(
    cd "$ROOT_DIR/backend"
    go run ./cmd/seed
)

echo "Starting backend at http://localhost:${MEALSWAPP_HTTP_PORT:-8080}..."
(
    cd "$ROOT_DIR/backend"
    exec go run ./cmd/api
) &
BACKEND_PID=$!

echo "Starting frontend at http://localhost:5173..."
(
    cd "$ROOT_DIR/frontend"
    exec bun run dev
) &
FRONTEND_PID=$!

echo "Development environment running. Press Ctrl-C to stop the backend and frontend."

set +e
wait -n "$BACKEND_PID" "$FRONTEND_PID"
status=$?
set -e

exit "$status"
