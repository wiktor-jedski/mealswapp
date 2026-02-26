#!/bin/bash

PARENT="$1"
CHILD="$2"
TASK="$3"

mkdir -p logs

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
LOG_FILE="logs/${PARENT}_${CHILD}_${TASK}_${TIMESTAMP}.log"

echo "Spawning container. Logs are being saved to: $LOG_FILE"

docker run --rm --name "$PARENT"-"$CHILD"-"$TASK" \
  -v ~/.local/share/opencode/auth.json:/home/ubuntu/.local/share/opencode/auth.json \
  -v ~/.config/opencode:/home/ubuntu/.config/opencode \
  --env-file .env.docker \
  mealswapp-dev:latest "$PARENT" "$CHILD" "$TASK" > "$LOG_FILE" 2>&1
