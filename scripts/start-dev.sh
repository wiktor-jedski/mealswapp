#!/usr/bin/env bash
# Implements DESIGN-001 SearchView and DESIGN-010 RouteHandler local development startup.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=dev-processes.sh
source "$ROOT_DIR/scripts/dev-processes.sh"
BACKEND_PID=""
WORKER_PID=""
FRONTEND_PID=""
STRIPE_PID=""
STRIPE_LOG=""
ENV_FILE="${MEALSWAPP_ENV_FILE:-$ROOT_DIR/.env}"
STRIPE_WEBHOOK_URL=""
START_STRIPE_CLI="${MEALSWAPP_START_STRIPE_CLI:-false}"

cleanup() {
    local pids=()
    trap - EXIT INT TERM
    stop_dev_process "$BACKEND_PID"
    stop_dev_process "$WORKER_PID"
    stop_dev_process "$FRONTEND_PID"
    stop_dev_process "$STRIPE_PID"
    [[ -n "$BACKEND_PID" ]] && pids+=("$BACKEND_PID")
    [[ -n "$WORKER_PID" ]] && pids+=("$WORKER_PID")
    [[ -n "$FRONTEND_PID" ]] && pids+=("$FRONTEND_PID")
    [[ -n "$STRIPE_PID" ]] && pids+=("$STRIPE_PID")
    [[ "${#pids[@]}" -gt 0 ]] && wait "${pids[@]}" 2>/dev/null || true
    [[ -n "$STRIPE_LOG" ]] && rm -f "$STRIPE_LOG"
    return 0
}

trap cleanup EXIT INT TERM

usage() {
    cat <<EOF
Usage: $0 [--stripe]

Options:
  --stripe   Start Stripe CLI webhook forwarding and inject its local webhook secret into the backend.

Environment:
  MEALSWAPP_START_STRIPE_CLI=true   Equivalent to --stripe.
  MEALSWAPP_ENV_FILE=path           Env file loaded for local dev. Defaults to $ROOT_DIR/.env.
  MEALSWAPP_STRIPE_WEBHOOK_URL=url  Forwarding target. Defaults to backend /api/v1/billing/stripe/webhook.
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --stripe)
            START_STRIPE_CLI="true"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown argument: $1" >&2
            usage >&2
            exit 1
            ;;
    esac
done

if [[ -f "$ENV_FILE" ]]; then
    echo "Loading local development environment from ${ENV_FILE}..."
    set -a
    # shellcheck disable=SC1090
    source "$ENV_FILE"
    set +a
fi

STRIPE_WEBHOOK_URL="${MEALSWAPP_STRIPE_WEBHOOK_URL:-http://127.0.0.1:${MEALSWAPP_HTTP_PORT:-8080}/api/v1/billing/stripe/webhook}"
START_STRIPE_CLI="${MEALSWAPP_START_STRIPE_CLI:-$START_STRIPE_CLI}"

for command in go bun setsid; do
    if ! command -v "$command" >/dev/null 2>&1; then
        echo "Required command not found: $command" >&2
        exit 1
    fi
done

if ! command -v "${MEALSWAPP_CLP_EXECUTABLE:-clp}" >/dev/null 2>&1; then
    echo "Required CLP executable not found: ${MEALSWAPP_CLP_EXECUTABLE:-clp}" >&2
    exit 1
fi

if [[ "$START_STRIPE_CLI" == "true" ]] && ! command -v stripe >/dev/null 2>&1; then
    echo "Required command not found: stripe" >&2
    echo "Install and authenticate the Stripe CLI, or run without --stripe." >&2
    exit 1
fi

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

if [[ "$START_STRIPE_CLI" == "true" ]]; then
    if [[ -z "${MEALSWAPP_STRIPE_SECRET_KEY:-}" ]]; then
        echo "MEALSWAPP_STRIPE_SECRET_KEY is required when starting Stripe CLI forwarding." >&2
        exit 1
    fi
    STRIPE_LOG="$(mktemp)"
    echo "Starting Stripe CLI forwarding to ${STRIPE_WEBHOOK_URL}..."
    start_dev_process "$ROOT_DIR" stripe listen --api-key "$MEALSWAPP_STRIPE_SECRET_KEY" --forward-to "$STRIPE_WEBHOOK_URL" >"$STRIPE_LOG" 2>&1
    STRIPE_PID=$DEV_PROCESS_PID

    for _ in {1..60}; do
        if ! kill -0 "$STRIPE_PID" 2>/dev/null; then
            echo "Stripe CLI exited before producing a webhook secret:" >&2
            sed -n '1,120p' "$STRIPE_LOG" >&2
            exit 1
        fi
        if grep -Eo 'whsec_[A-Za-z0-9_]+' "$STRIPE_LOG" >/dev/null 2>&1; then
            export MEALSWAPP_STRIPE_WEBHOOK_SECRET
            MEALSWAPP_STRIPE_WEBHOOK_SECRET="$(grep -Eo 'whsec_[A-Za-z0-9_]+' "$STRIPE_LOG" | tail -n 1)"
            break
        fi
        sleep 1
    done

    if [[ -z "${MEALSWAPP_STRIPE_WEBHOOK_SECRET:-}" ]]; then
        echo "Timed out waiting for Stripe CLI webhook secret. Recent Stripe CLI output:" >&2
        sed -n '1,120p' "$STRIPE_LOG" >&2
        exit 1
    fi
    echo "Stripe CLI forwarding is running. Webhook secret injected into backend environment."
fi

echo "Starting backend at http://localhost:${MEALSWAPP_HTTP_PORT:-8080}..."
start_dev_process "$ROOT_DIR/backend" go run ./cmd/api
BACKEND_PID=$DEV_PROCESS_PID

echo "Starting optimization worker..."
start_dev_process "$ROOT_DIR/backend" go run ./cmd/worker
WORKER_PID=$DEV_PROCESS_PID

echo "Starting frontend at http://localhost:5173..."
start_dev_process "$ROOT_DIR/frontend" bun run dev
FRONTEND_PID=$DEV_PROCESS_PID

if [[ "$START_STRIPE_CLI" == "true" ]]; then
    echo "Development environment running. Press Ctrl-C to stop Stripe CLI, backend, worker, and frontend."
else
    echo "Development environment running. Press Ctrl-C to stop the backend, worker, and frontend."
fi

set +e
if [[ -n "$STRIPE_PID" ]]; then
    wait -n "$BACKEND_PID" "$WORKER_PID" "$FRONTEND_PID" "$STRIPE_PID"
else
    wait -n "$BACKEND_PID" "$WORKER_PID" "$FRONTEND_PID"
fi
status=$?
set -e

exit "$status"
