PARENT="$1"
CHILD="$2"

echo "Perform code review"
opencode run "read docs/implementation/reviewer-prompt.md\\
Task ID: $CHILD\\
Implementation phase: $PARENT"

