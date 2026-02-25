PARENT="$1"
CHILD="$2"

echo "Run coding task"
opencode run "read docs/implementation/coder-prompt.md\\
Task ID: $CHILD\\
Implementation phase: $PARENT"

echo "Push changes and create a PR"
git push -u origin "$PARENT"-"$CHILD"
gh pr create --base "$PARENT" --head "$PARENT"-"$CHILD" --fill

echo "Update task list"
sh scripts/update_task_status.sh "$PARENT" "$CHILD" "PREPARED"
