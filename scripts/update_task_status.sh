#!/bin/bash
set -e

PARENT="$1"
CHILD="$2"
NEW_STATUS="$3"

echo "Updating task $CHILD to $NEW_STATUS on main branch..."

# Save current branch name
CURRENT_BRANCH=$(git branch --show-current)

# Switch to the phase/main branch where the task list lives
git checkout "$PARENT"
git clean -fd
git restore .

# Retry loop for concurrency (max 5 attempts)
ATTEMPTS=0
while [ $ATTEMPTS -lt 5 ]; do
  # Pull latest changes to avoid conflicts
  git pull origin "$PARENT" --rebase
  
  # Ask opencode to ONLY update the markdown table
  opencode run "In the file docs/implementation/02_TASK_LIST.md, find the row for Task ID: $CHILD. Task IDs are in the first column. Change its status to $NEW_STATUS. Status is in the fourth column. Do not touch anything else."
  
  git add docs/implementation/02_TASK_LIST.md
  
  # Check if there are actually changes to commit
  if git diff-index --quiet HEAD; then
    echo "No changes needed or status already updated."
    break
  fi

  git commit -m "chore: update $CHILD status to $NEW_STATUS"
  
  # Try to push. If successful, break the loop.
  if git push origin "$PARENT"; then
    echo "✅ Status updated successfully!"
    break
  else
    echo "⚠️ Git push failed due to concurrent update. Retrying..."
    # Discard failed local commit, reset to remote, and loop again
    git reset --hard origin/"$PARENT"
    sleep $((RANDOM % 5 + 2)) # Random sleep to prevent race conditions
    ((ATTEMPTS++))
  fi
done

# Go back to the feature branch so the rest of the script continues normally
git checkout "$CURRENT_BRANCH"
