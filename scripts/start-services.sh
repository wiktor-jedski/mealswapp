#!/usr/bin/env bash
# Implements DESIGN-011 RedisCache and DESIGN-005 RepositoryInterfaces local service startup.
set -euo pipefail

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    docker compose up -d postgres redis
else
    echo "Starting PostgreSQL with system service..."
    service postgresql start
    until pg_isready -q; do
        echo "Waiting for PostgreSQL..."
        sleep 1
    done

    echo "Starting Redis with system service..."
    service redis-server start
    until redis-cli ping | grep -q PONG; do
        echo "Waiting for Redis..."
        sleep 1
    done
fi

echo "Development services ready:"
echo "PostgreSQL: localhost:5432 database=mealswapp user=mealswapp"
echo "Redis: localhost:6379"
