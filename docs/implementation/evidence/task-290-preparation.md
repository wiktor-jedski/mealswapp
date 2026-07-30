# Task 290 final acceptance evidence

## Controlled disposable-stack run

The final acceptance runner was executed from the Task 290 worktree against the isolated Task 289 API source tree:

```text
MEALSWAPP_TASK290_DISPOSABLE=1 python3 scripts/run-task290-acceptance.py \
  --environment development \
  --start-disposable \
  --api-source-root /home/wiktor/Work/worktrees/mealswapp/289
```

Run ID: `ce39997eb0e77209dc4b89c7`

Result: `task290_real_api=passed audit=verified_or_not_requested`

The run-owned artifact directory was `logs/real-stack-e2e/ce39997eb0e77209dc4b89c7/` and recorded `status: cleaned`, `result: passed`, and these bounded lifecycle events:

- `redis_ready`
- `task290_api_started`
- `task290_migrations_applied`
- `task290_commands_passed`
- `task290_audit_verified`
- `task290_final_state_verified`

The runner created and migrated an owned PostgreSQL database, started Redis and the API from the selected Task 289 source root, bootstrapped a run-owned administrator, performed list, add dry-run, add, existing-key dry-run, display-name update, unit update, deactivate, and reactivate through the operator, verified all five `micronutrient.*` actions on entity type `micronutrient_vocabulary`, checked the final `(display name, unit, active)` projection, and dropped the owned database and Redis container during cleanup.

## Focused validation

- 18 Task 290 operator and acceptance-harness tests passed.
- Python compilation passed.
- Traceability validation passed.
- Task-list validation passed without editing status rows.
- `git diff --check` passed.

The dependency row for Task 289 remains `PREPARED`; this evidence does not transition either task-list row.
