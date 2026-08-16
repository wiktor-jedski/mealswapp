#!/usr/bin/env bash
set -euo pipefail

# Implements DESIGN-005 RepositoryInterfaces isolated Task 261 real-stack verification.
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec python3 "$repo_dir/scripts/run-real-stack-e2e.py" "$@"
