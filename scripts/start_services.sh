#!/bin/bash
set -e
set -u

PARENT="$1"
CHILD="$2"
TASK="$3"

TARGET_DIR="/workspace"
REQUIRED_REPO="wiktor-jedski/mealswapp"
CURRENT_REMOTE=$(git remote get-url origin 2>/dev/null || echo "")

if [[ "$PWD" != "$TARGET_DIR" ]]; then
  echo "Wrong directory: need to be in /workspace"
  exit 1
fi

echo "Starting PostgreSQL..."
sudo service postgresql start

until pg_isready -q; do
  echo "Waiting for PostgreSQL..."
  sleep 1
done

echo "Starting Redis..."
sudo service redis-server start

until redis-cli ping | grep -q PONG; do
  echo "Waiting for Redis..."
  sleep 1
done

echo "================================"
echo "Development environment ready!"
echo "PostgreSQL: localhost:5432"
echo "Redis: localhost:6379"
echo "================================"

echo "Set up git config"
git config --global user.name "$USER_NAME"
git config --global user.email "$USER_EMAIL"
gh auth setup-git

echo "Set up git repo"
if [[ "$CURRENT_REMOTE" == *"$REQUIRED_REPO"* ]]; then
  echo "Correct repo detected. Fetching..."
  git fetch --all
  git reset --hard origin/"$PARENT"
else
  echo "Wrong repo or empty dir. Cleaning and cloning..."
  ls -A1 | xargs rm -rf
  gh repo clone "$REQUIRED_REPO" . -- -b "$PARENT"
fi

echo "Run chosen task"
if [[ $TASK == "NEW" ]]; then
  git checkout -B "$PARENT"-"$CHILD"
  bash scripts/new_task.sh "$PARENT" "$CHILD"
elif [[ $TASK == "REVIEW" ]]; then
  git fetch origin
  git checkout "$PARENT"-"$CHILD"
  bash scripts/review_task.sh "$PARENT" "$CHILD"
elif [[ $TASK == "REVISE" ]]; then
  git fetch origin
  git checkout "$PARENT"-"$CHILD"
  bash scripts/revise_task.sh "$PARENT" "$CHILD"
else
  echo "Error: unknown task passed"
  exit 1
fi

