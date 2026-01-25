#!/bin/bash
set -e
set -u

PARENT="$1"
CHILD="$2"
TASK="$3"

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

if [[ $TASK == "NEW" ]]; then
    rm -rf *
    gh repo clone wiktor-jedski/mealswapp . -- -b "$PARENT" --single-branch
    sh scripts/new_task.sh "$PARENT" "$CHILD"
elif [[ $TASK == "REVIEW" ]]; then
    if [ -d ".git" ]; then
        git pull
    else
        gh repo clone wiktor-jedski/mealswapp . -- -b "$PARENT"-"$CHILD" --single-branch
    fi
    sh scripts/reviewed_task.sh
else
    echo "Error: unkown task passed"
    exit 1
fi

