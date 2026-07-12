#!/usr/bin/env bash
set -euo pipefail

# Implements DESIGN-004 LPSolverWrapper packaged worker verification.
docker build --file backend/Dockerfile.worker --tag mealswapp-worker:task-201 .
