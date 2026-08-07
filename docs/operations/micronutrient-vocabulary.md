# Micronutrient vocabulary operator

`scripts/manage-micronutrient-vocabulary.py` manages the canonical micronutrient vocabulary only through the authenticated administration API described by DESIGN-009. It never connects to PostgreSQL.

## Target and authentication

Every invocation requires both `--environment` and `--base-url`. HTTP is accepted only for a loopback `development` target. Staging and production require HTTPS, and production also requires `--confirm-production`. The target must be an origin without credentials, a path, query parameters, or a fragment.

The administrator email and password are read interactively. The password is not echoed. Cookies and the fresh CSRF token remain only in process memory; credentials, cookies, tokens, email addresses, response bodies, database URLs, and stack diagnostics are never written to output.

List active and inactive entries:

```sh
python3 scripts/manage-micronutrient-vocabulary.py \
  --environment development \
  --base-url http://127.0.0.1:8080 \
  list
```

The output is deterministic tab-separated text ordered by canonical key.

## Mutations

Add an entry:

```sh
python3 scripts/manage-micronutrient-vocabulary.py \
  --environment staging \
  --base-url https://staging-api.example.test \
  add --key VitaminK --display-name "Vitamin K" --unit mcg
```

The remaining commands are:

```text
update-display-name --key VitaminK --display-name "Vitamin K1"
update-unit         --key VitaminK --unit mg
deactivate          --key VitaminK
reactivate          --key VitaminK
```

Append `--dry-run` to a mutation command to authenticate, read authoritative vocabulary state, and validate the operation without sending a mutation. Unit updates and deactivation can still be rejected by the server with an in-use conflict because food-item usage is intentionally not exposed by the list API.

Every mutation carries a fresh CSRF token. After an ambiguous network or 5xx outcome, the operator reads authoritative vocabulary state and accepts an already-applied result before considering a retry; this prevents duplicate audit mutations after a lost success response. Otherwise, ambiguous outcomes and valid bounded `Retry-After` responses are retried at most three times with the identical method, path, and body. Authentication or authorization failure aborts immediately. Validation, missing-entry, in-use/concurrent conflict, malformed response, exhausted retry, and permanent HTTP failures exit nonzero with a fixed safe diagnostic.

## Disposable-stack acceptance

After starting a throwaway PostgreSQL/Redis/API stack and bootstrapping an administrator, run the real API acceptance with `MEALSWAPP_TASK290_DISPOSABLE=1`, `MEALSWAPP_TASK290_ADMIN_EMAIL`, and `MEALSWAPP_TASK290_ADMIN_PASSWORD` supplied only through the environment:

```sh
MEALSWAPP_TASK290_DISPOSABLE=1 \
MEALSWAPP_TASK290_ADMIN_EMAIL="$ADMIN_EMAIL" \
MEALSWAPP_TASK290_ADMIN_PASSWORD="$ADMIN_PASSWORD" \
python3 scripts/run-task290-acceptance.py \
  --environment development \
  --base-url http://127.0.0.1:8080 \
  --database-url "$DISPOSABLE_DATABASE_URL"
```

The runner performs list, dry-run, add, display-name update, unit update, deactivate, and reactivate through the real HTTP API, then checks the final projection and (when `--database-url` is supplied) read-only audit action names. Tear down the disposable database after the run; no operator mutation is directed at a persistent fixture.

To let the runner own the complete disposable lifecycle, use `--start-disposable`. It creates and migrates an owned database, starts Redis and the controlled Task 289 API, bootstraps a run-owned administrator, runs the commands, records inspectable lifecycle artifacts under `logs/real-stack-e2e/`, verifies cleanup, and drops the owned database and Redis container. When Task 289 is still isolated, point `--api-source-root` at its task worktree; after integration, omit that option.

```sh
MEALSWAPP_TASK290_DISPOSABLE=1 \
python3 scripts/run-task290-acceptance.py \
  --environment development \
  --start-disposable \
  --api-source-root /home/wiktor/Work/worktrees/mealswapp/289
```
