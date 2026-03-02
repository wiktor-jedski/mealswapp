#!/bin/bash
PARENT="$1"
CHILD="$2"

echo "Perform code review"
opencode run "read docs/implementation/reviewer-prompt.md\\
Task ID: $CHILD\\
Implementation phase: $PARENT\\
Write your final decision (exactly PASSED or REJECTED) to a new file named REVIEW_RESULT.txt"

RESULT=$(cat REVIEW_RESULT.txt)
echo "$RESULT"

if [ "$RESULT" = "PASSED" ]; then
  echo "Code passed review! Merging PR..."
  rm REVIEW*.md
  rm review*.md
  git add . && git commit -m "removed review" && git push
  gh pr merge "$PARENT"-"$CHILD" --merge --delete-branch
  ./scripts/update_task_status.sh "$PARENT" "$CHILD" "PASSED"
  
elif [ "$RESULT" = "REJECTED" ]; then
  echo "Code rejected!"
  git add . && git commit -m "added review" && git push
  ./scripts/update_task_status.sh "$PARENT" "$CHILD" "REJECTED"
fi

rm -f REVIEW_RESULT.txt
