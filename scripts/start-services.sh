#!/bin/bash
set -e

TASK="$1"

# Start PostgreSQL
echo "Starting PostgreSQL..."
service postgresql start

# Wait for PostgreSQL to be ready
until pg_isready -q; do
    echo "Waiting for PostgreSQL..."
    sleep 1
done

# Create dev user and database
# echo "Setting up database..."
# su - postgres -c "psql -tc \"SELECT 1 FROM pg_roles WHERE rolname='mealswapp'\" | grep -q 1 || psql -c \"CREATE USER mealswapp WITH PASSWORD 'dev' CREATEDB;\""
# su - postgres -c "psql -tc \"SELECT 1 FROM pg_database WHERE datname='mealswapp'\" | grep -q 1 || psql -c \"CREATE DATABASE mealswapp OWNER mealswapp;\""

# Start Redis
echo "Starting Redis..."
service redis-server start

# Wait for Redis to be ready
until redis-cli ping | grep -q PONG; do
    echo "Waiting for Redis..."
    sleep 1
done

echo "================================"
echo "Development environment ready!"
echo "PostgreSQL: localhost:5432"
echo "Redis: localhost:6379"
# echo "Database: mealswapp / user: mealswapp / pass: dev"
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
