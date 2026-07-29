#!/usr/bin/env bash
set -euo pipefail

# Implements DESIGN-001 PreferenceManager isolated Task 296 real-stack verification.
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
: "${MEALSWAPP_DATABASE_URL:?set MEALSWAPP_DATABASE_URL to the disposable controlled-stack database}"
: "${MEALSWAPP_REDIS_URL:?set MEALSWAPP_REDIS_URL to the controlled-stack Redis}"

api_port="${MEALSWAPP_HTTP_PORT:-18080}"
frontend_origin="${MEALSWAPP_FRONTEND_ORIGIN:-http://localhost:5173}"
export MEALSWAPP_ENV="${MEALSWAPP_ENV:-development}"
export MEALSWAPP_HTTP_PORT="$api_port"
export MEALSWAPP_FRONTEND_ORIGIN="$frontend_origin"
export MEALSWAPP_ALLOWED_ORIGINS="${MEALSWAPP_ALLOWED_ORIGINS:-$frontend_origin}"
export MEALSWAPP_VITE_API_TARGET="${MEALSWAPP_VITE_API_TARGET:-http://127.0.0.1:$api_port}"

api_log="${MEALSWAPP_TASK296_API_LOG:-$(mktemp -t mealswapp-task296-api.XXXXXX.log)}"
result_file="${MEALSWAPP_TASK296_RESULT_FILE:-}"
if [[ -n "$result_file" && "$result_file" != /* ]]; then
	result_file="$repo_dir/$result_file"
fi
api_pid=""
cleanup() {
	if [[ -n "$api_pid" ]]; then
		kill "$api_pid" 2>/dev/null || true
		wait "$api_pid" 2>/dev/null || true
	fi
}
trap cleanup EXIT

(
	cd "$repo_dir/backend"
	GOCACHE="${GOCACHE:-$PWD/.go-cache}" \
	GOMODCACHE="${GOMODCACHE:-$PWD/.go-mod-cache}" \
	go run ./cmd/migrate up
) >"$api_log" 2>&1
(
	cd "$repo_dir/backend"
	GOCACHE="${GOCACHE:-$PWD/.go-cache}" \
	GOMODCACHE="${GOMODCACHE:-$PWD/.go-mod-cache}" \
	go run ./cmd/api
) >>"$api_log" 2>&1 &
api_pid=$!

for _ in $(seq 1 60); do
	if curl --fail --silent --show-error "http://127.0.0.1:$api_port/health" >/dev/null; then
		break
	fi
	sleep 1
done
curl --fail --silent --show-error "http://127.0.0.1:$api_port/health" >/dev/null

cd "$repo_dir/frontend"
MEALSWAPP_REAL_STACK_E2E=1 \
	MEALSWAPP_REAL_STACK_RESULT_FILE="$result_file" \
	bunx playwright test -c playwright.real-stack.config.ts tests/task296-real-stack-preference.spec.ts "$@"

if [[ -n "$result_file" ]]; then
	python3 - "$result_file" "$api_port" <<'PY'
import json
import sys
from pathlib import Path

Path(sys.argv[1]).write_text(json.dumps({
    "schema": "mealswapp.task296.real-stack-result.v1",
    "apiHealth": "passed",
    "apiPort": int(sys.argv[2]),
    "playwright": "passed",
    "tests": 1,
}, indent=2) + "\n", encoding="utf-8")
PY
fi
