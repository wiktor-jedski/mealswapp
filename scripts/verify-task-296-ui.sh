#!/usr/bin/env bash
set -euo pipefail

# Implements DESIGN-001 PreferenceManager isolated Task 296 real-stack verification.
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir/frontend"
exec env MEALSWAPP_REAL_STACK_E2E=1 bunx playwright test -c playwright.real-stack.config.ts tests/task296-real-stack-preference.spec.ts "$@"
