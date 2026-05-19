#!/bin/bash
set -e

TASK="$1"

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    echo "Starting PostgreSQL and Redis with Docker Compose..."
    docker compose up -d postgres redis

    echo "Waiting for PostgreSQL..."
    until docker compose exec -T postgres pg_isready -U "${POSTGRES_USER:-mealswapp}" -d "${POSTGRES_DB:-mealswapp}" >/dev/null 2>&1; do
        sleep 1
    done

    echo "Waiting for Redis..."
    until docker compose exec -T redis redis-cli ping | grep -q PONG; do
        sleep 1
    done
else
    echo "Docker Compose not available; falling back to local system services."

    echo "Starting PostgreSQL..."
    service postgresql start

    until pg_isready -q; do
        echo "Waiting for PostgreSQL..."
        sleep 1
    done

    echo "Starting Redis..."
    service redis-server start

    until redis-cli ping | grep -q PONG; do
        echo "Waiting for Redis..."
        sleep 1
    done
fi

echo "================================"
echo "Development environment ready!"
echo "PostgreSQL: localhost:5432"
echo "Redis: localhost:6379"
echo "DATABASE_URL=postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable"
echo "REDIS_URL=redis://localhost:6379/0"
echo "================================"

# If a task was provided, execute it
if [ -n "$TASK" ]; then
    echo "Executing task: $TASK"
    # Clone repo if workspace is empty
    if [ -z "$(ls -A /workspace)" ]; then
        git clone "$REPO_URL" . 2>/dev/null || true
    fi

    # Execute task (customize this to your agent setup)
    # Example: run a script, call an API, etc.
    exec $TASK
else
    # Keep container running for interactive use
    exec tail -f /dev/null
fi
