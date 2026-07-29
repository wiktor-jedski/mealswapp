# Isolated real-stack E2E operation

<!-- Implements DESIGN-005 RepositoryInterfaces isolated real-stack test persistence. -->

`python3 scripts/run-real-stack-e2e.py` runs the Task 261 administration browser flow without using the development database, development Redis data, or fixed application ports. Each invocation owns one database, one Redis container, and one API/frontend process group. Concurrent invocations are supported.

## Safety contract

- The PostgreSQL maintenance URL defaults to loopback `postgres`; override it only with `MEALSWAPP_E2E_POSTGRES_ADMIN_URL`. Non-loopback hosts and maintenance databases other than `postgres` are rejected before `psql` runs.
- Databases must match `mealswapp_e2e_<24 lowercase hex characters>_test`. Creation writes `mealswapp-e2e-owner run=<run-id> created=<unix-seconds>` as the database comment. Deletion requires both markers to match exactly.
- Redis runs in a disposable `redis:7-alpine` container bound to a random loopback port. Cleanup requires the exact `mealswapp.e2e.run` label. The harness never runs `FLUSHALL`.
- Migrations target only the generated database and do not invoke development seeds. The fixture is registered and verified over the run-owned API, then promoted by the Task 276 `admin-bootstrap` CLI. No browser-side SQL or role-promotion subprocess exists.
- API, frontend, and Playwright output first goes to a run-private temporary directory. Persistent diagnostics under `logs/real-stack-e2e/<run-id>/` contain only run IDs, request IDs, lifecycle event names, and cleanup metadata. Raw logs, HTTP bodies, traces, screenshots, cookies, CSRF tokens, fixture PII, credentials, and database URLs are destroyed.

Handled `INT`, `TERM`, and `HUP` signals, assertion failures, readiness timeouts, and ordinary cancellation all enter the same ownership-checked teardown. `SIGKILL` cannot run in-process cleanup; use the recovery mode after its age guard.

## Stale-run recovery

Run:

```sh
python3 scripts/run-real-stack-e2e.py --cleanup-stale --stale-age-seconds 3600
```

Recovery is explicit and refuses ages below 900 seconds. It reads only harness state directories, validates the exact run ID, database name/comment, Redis name/label, process start token, and run-prefixed system temporary directory, then removes those owned resources and any raw killed-run output. A young run is left untouched. Repeating cleanup is safe.

Do not edit state metadata to bypass the guard. If ownership markers disagree, inspect manually; never drop or flush a broader target.
