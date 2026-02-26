#!/bin/bash

PARENT="$1"
CHILD="$2"
TASK="$3"

docker run --name "$PARENT"-"$CHILD"-"$TASK" \
  -v ~/.local/share/opencode/auth.json:/home/ubuntu/.local/share/opencode/auth.json \
  -v ~/.config/opencode:/home/ubuntu/.config/opencode \
  --env-file ../.env.docker \
  mealswapp-dev:latest "$PARENT" "$CHILD" "$TASK"
