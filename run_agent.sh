PARENT="$1"
CHILD="$2"
TASK="$3"

docker run -dit --name agent \
  -v ~/.local/share/opencode/auth.json:/home/ubuntu/.local/share/opencode/auth.json \
  -v ~/.config/opencode:/home/ubuntu/.config/opencode \
  --env-file .env.docker \
  mealswapp-dev:latest "$PARENT" "$CHILD" "$TASK"
