#!/bin/bash
set -e

PARENT="$1"
CHILD="$2"

# Start PostgreSQL
echo "Starting PostgreSQL..."
sudo service postgresql start

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
sudo service redis-server start

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

echo "Set up git config"
git config --global user.name "WiktorBot"
git config --global user.email "jedrzejewskiwiktor@gmail.com"
gh auth setup-git

echo "Clone repo"
gh repo clone wiktor-jedski/mealswapp . -- -b "$PARENT" --single-branch
if [[ $PARENT != $CHILD ]]; then
    git checkout -b "$PARENT"-"$CHILD"
fi

echo "Run coding task"
# opencode run "create a hello world app in go and commit changes"
echo "henlo" > hello.txt
git add .
git commit -m "hello test"

echo "Push changes and create a PR"
if [[ $PARENT != $CHILD ]]; then
    git push -u origin "$PARENT"-"$CHILD"
    gh pr create --base "$PARENT" --head "$PARENT"-"$CHILD" --fill
else
    git push
fi

# If a task was provided, execute it
# if [ -n "$TASK" ]; then
#     echo "Executing task: $TASK"
#     # Clone repo if workspace is empty
#     if [ -z "$(ls -A /workspace)" ]; then
#         git clone "$REPO_URL" . 2>/dev/null || true
#     fi
#
#     # Execute task (customize this to your agent setup)
#     # Example: run a script, call an API, etc.
#     exec $TASK
# else
#     # Keep container running for interactive use
#     exec tail -f /dev/null
# fi
