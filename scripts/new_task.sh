PARENT="$1"
CHILD="$2"

echo "Run coding task"
opencode run "read docs/implementation/coder-prompt.md\\
Task ID: $CHILD\\
Tech stack: docs/design/01_TECH_STACK.md\\
Implementation phase: $PARENT"

echo "Push changes and create a PR"
git push -u origin "$PARENT"-"$CHILD"
gh pr create --base "$PARENT" --head "$PARENT"-"$CHILD" --fill
