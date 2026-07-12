#!/usr/bin/env bash
set -euo pipefail

# Implements DESIGN-004 amd64 optimizer image and packaged CLP verification.
readonly image="mealswapp-optimizer:task-212"

if ! command -v docker >/dev/null 2>&1; then
    echo "Docker is required to verify the optimizer image." >&2
    exit 1
fi
if ! docker info >/dev/null 2>&1; then
    echo "Docker is installed but its daemon is unavailable." >&2
    exit 1
fi

docker build \
    --no-cache \
    --platform linux/amd64 \
    --file backend/Dockerfile.worker \
    --tag "${image}" \
    .

version_output="$(docker run --rm --platform linux/amd64 --entrypoint /usr/local/bin/clp "${image}" -version 2>&1)"
grep --fixed-strings "Coin LP version 1.17.11" <<<"${version_output}"

docker run --rm --platform linux/amd64 --entrypoint /bin/sh "${image}" -c \
    'test -x /usr/local/bin/mealswapp-worker'
