PARENT="$1"
CHILD="$2"

echo "Run revise task"
opencode run "read docs/implementation/reviser-prompt.md\\
Task ID: $CHILD\\
Implementation phase: $PARENT"

echo "Update task list"
sh scripts/update_task_status.sh "$PARENT" "$CHILD" "PREPARED"
