PARENT="$1"
CHILD="$2"

echo "Run coding task"
opencode run "create a hello world app in go and commit changes"

echo "Push changes and create a PR"
git push -u origin "$PARENT"-"$CHILD"
gh pr create --base "$PARENT" --head "$PARENT"-"$CHILD" --fill
