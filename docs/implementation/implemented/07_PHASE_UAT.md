# Phase 07 UAT: Daily Diet Optimization Worker

<!-- Implements DESIGN-001 SearchView, DESIGN-004 JobStatusTracker, DESIGN-008 SavedDataRepository, and DESIGN-014 MetricsCollector. -->

## Scope

Phase 07 covers Tasks `192`-`211`: user-owned saved daily-diet persistence,
generated contracts and mode-safe frontend state, multi-meal collection and
aggregation, LP constraint/objective/diversity/validation behavior, the native
CLP worker boundary, Redis Streams delivery, authenticated submission and
polling, frontend optimization workflow, integration and browser acceptance
coverage, queue observability/capacity evidence, aggregate verification, and
SWE.5 integration obligations.

The implementation uses:

- `DESIGN-008` saved-diet parent and ordered-entry persistence, with totals
  recalculated from server-side meal data;
- `DESIGN-004` Redis Streams through `github.com/redis/go-redis/v9`, a
  dedicated worker, and a pure-Go wrapper around native COIN-OR CLP `1.17.11`;
- authenticated owner predicates for saved diets, optimization submission, and
  polling; and
- `DESIGN-001` in-memory frontend collection/optimization state with bounded
  polling and safe retry behavior.

## Automated Verification

The command/result records below are the reproducible evidence cited by the
Task 209 and Task 210 reviews. Commands that require local services use the
repository defaults and isolated Redis databases; no cookies, tokens, or
fixture request bodies are committed.

### Aggregate gate

Exact recorded command from the repository root:

```sh
python3 scripts/check.py
```

Result: **PASS**, exit code `0` (Task 209 review). The aggregate used the
following effective environment and service assumptions:

```text
GOCACHE=backend/.go-cache
GOMODCACHE=backend/.go-mod-cache
BUN_TMPDIR=frontend/.bun-tmp
BUN_INSTALL=frontend/.bun-install
MEALSWAPP_DATABASE_URL=postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable
MEALSWAPP_REDIS_URL=redis://localhost:6379/0 for local-stack/API processes
MEALSWAPP_CLP_EXECUTABLE=clp
MEALSWAPP_CLP_VERSION=1.17.11
```

`scripts/check.py` isolates its Go checks on Redis databases `10` (serial
tests), `11` (race tests), `12` (aggregate coverage and queue coverage), `13`
(worker coverage), `14` (HTTP/app focused workflows), and `15` (focused
queue/worker workflows). It supplies the cache variables above internally;
the CLP values come from the configuration defaults and the local executable
reported `Coin LP version 1.17.11`.

Recorded aggregate results:

- OpenAPI lint passed with one existing explicitly ignored OAuth callback
  `302`-only `operation-2xx-response` warning.
- Backend aggregate coverage was `86.2%`; Phase 07 packages were
  `dailydiet 71.1%`, `optimization 74.6%`, `queue 68.4%`, and `worker 48.2%`.
- Frontend aggregate coverage was `93.19%` functions and `92.79%` lines.
- The complete browser/axe run was `215 passed, 3 intentionally skipped,
  0 failed` (`218` total).

Evidence/report paths: [Task 209 review](../reviews/task-209-review.md),
local Go coverage output `backend/coverage.out`, [Phase 07 open-point and
deviation register](../04_OPEN.md), and [Task 210 review](../reviews/task-210-review.md).
The recorded passing invocation was the bare command above, so it did not
write an HTML phase report; the review and coverage paths are the authoritative
recorded evidence for that run.

### Backend integration and coverage

The Phase 07 focused backend block executed by `scripts/check.py` was:

```sh
cd backend
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/migrations -run '^TestRun' -count=1
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/repository -run '^TestPostgresSavedDiet' -count=1
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/dailydiet -count=1
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization -run '^(TestBuild|TestGenerate|TestLPSolver|TestValidate|TestSafe)' -count=1
MEALSWAPP_REDIS_URL=redis://localhost:6379/15 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/queue -run '^TestJobQueue' -count=1
MEALSWAPP_REDIS_URL=redis://localhost:6379/15 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/worker -run '^(TestRun|TestRedis|TestOptimization)' -count=1
MEALSWAPP_REDIS_URL=redis://localhost:6379/14 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -run '^(TestProfileControllerDailyDiet|TestOptimizationHTTP)' -count=1
MEALSWAPP_REDIS_URL=redis://localhost:6379/14 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/app -run '^(TestDailyDietProductionAPIWithLivePostgres|TestTask206)' -count=1
```

Result: **PASS** for the focused Phase 07 block against local PostgreSQL and
Redis. The Task 206 production integration crosses saved-diet persistence,
authenticated API, Redis, worker, native CLP, polling, ownership, duplicate
delivery, infeasible, timeout, queue-outage, exclusion, and concurrent paths.
The Task 210 SWE.5 integration evidence covers partial-result publication,
terminal acknowledgement, Redis result expiry, and retained owner markers.

The aggregate backend serial/race/coverage commands were:

```sh
cd backend
MEALSWAPP_REDIS_URL=redis://localhost:6379/10 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -p 1 -count=1
MEALSWAPP_REDIS_URL=redis://localhost:6379/11 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -p 1 -count=1
MEALSWAPP_REDIS_URL=redis://localhost:6379/12 GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/... -p 1 -count=1 -coverprofile=coverage.out
GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go tool cover -func=coverage.out
```

Result: **PASS**. The coverage measurements and specifically accepted
below-100% branches are recorded in [04_OPEN.md](../04_OPEN.md:268).

### Frontend unit, type, build, and coverage

Exact recorded commands from `frontend/` (with the repository-local Bun
directories shown explicitly) were:

```sh
cd frontend
BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check:api-types
BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run typecheck
BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run build
BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test
BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test --coverage
```

Results: generated API types, typecheck, and production build **PASS**;
`bun test` **PASS**, `350` tests and `0` failures; coverage **PASS with the
documented Phase 07 exceptions**, `All files | 93.19 funcs | 92.79 lines`.
The exact uncovered source groups and focused follow-up rule are in
[04_OPEN.md](../04_OPEN.md:269). The recorded coverage evidence is in the
[Task 209 review](../reviews/task-209-review.md:23).

### Playwright and axe

The exact scoped browser commands and results were:

```sh
cd frontend
BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e -- tests/daily-diet-workflow.spec.ts
BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/optimization-workflow.spec.ts
BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bunx playwright test tests/phase07-browser-acceptance.spec.ts
BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run test:e2e
```

Results: Daily Diet workflow **PASS**, `12` tests across desktop/mobile
Chromium including axe; optimization workflow **PASS**, `4` tests across
desktop/mobile Chromium; Phase 07 browser acceptance **PASS**, `14` tests
across desktop/mobile Chromium with responsive, keyboard, and axe checks; full
Playwright/axe **PASS**, `215 passed, 3 intentionally skipped, 0 failed`.
The browser harness uses `frontend/playwright.config.ts`; the deterministic
Task 207 screenshot is written by Playwright under
`frontend/test-results/**/task-207-daily-diet.png` and is not committed as a
phase report artifact. Evidence paths: [Task 197 review](../reviews/task-197-review.md:50),
[Task 205 review](../reviews/task-205-review.md:39), [Task 207 review](../reviews/task-207-review.md:39),
and [Task 209 review](../reviews/task-209-review.md:24).

### Capacity verification

The deterministic fail-closed capacity regression command was:

```sh
python3 -m unittest scripts/test_verify_optimization_capacity.py
```

Result: **PASS**, `8` tests. Evidence: [Task 208 review](../reviews/task-208-review.md:38).

The live authenticated responsiveness check is reproducible with operator-only
credentials and a fixture body that must not be committed:

```sh
MEALSWAPP_CAPACITY_COOKIE='...' \
MEALSWAPP_CAPACITY_CSRF_TOKEN='...' \
python3 scripts/verify-optimization-capacity.py \
  --body-file /path/to/optimization-fixture.json \
  --output logs/optimization-capacity.json
```

Result/status: **operator-executed evidence required**; the repository does
not claim a committed live credentialed run. On a passing run the report path
is `logs/optimization-capacity.json`; the gate requires every submission to
return `202`, at least one poll sample, P95 submission/poll latency below `2`
seconds, valid `/ready` samples, and Redis/worker/optimization-queue evidence.
The local gate is not a claim of the production `SW-REQ-082` 1,000-user target.
The thresholds and accepted local limit are in
[phase07-task-208-capacity.md](../phase07-task-208-capacity.md).

### Docker optimizer-image check

Exact command:

```sh
bash scripts/verify-clp-worker-image.sh
```

Result: **PASS**. The no-cache `linux/amd64` build downloads COIN-OR's
official Ubuntu 24 CLP `1.17.11` release artifact, verifies its pinned SHA-256,
builds the Go worker with `CGO_ENABLED=0`, and packages both in a minimal Ubuntu
24.04 optimizer runtime under a non-root user. Runtime verification reports:

```text
Coin LP version 1.17.11, build Mar 11 2026
```

The image is deliberately amd64-only because the upstream `1.17.11` release
does not publish an Ubuntu ARM64 artifact. CLP runs as a child process inside
the dedicated optimizer container; it is not deployed as a separate service
and is not present in the Fiber API process. Evidence:
[04_OPEN.md](../04_OPEN.md:270), `backend/Dockerfile.worker`, and
`scripts/verify-clp-worker-image.sh`.

### Task 211 documentation validation

The repair validation commands are:

```sh
python3 scripts/validate-task-list.py
python3 scripts/validate-traceability.py
git diff --check
```

Final Task 211 repair results:

- `python3 scripts/validate-task-list.py` — **PASS**, `211` sequential tasks
  with ordered dependencies.
- `python3 scripts/validate-traceability.py` — **PASS**.
- `git diff --check` — **PASS**.

Task-list status is intentionally not changed: Task 210 remains `PASSED` and
Task 211 remains `PREPARED`; no later task row was edited.

## Project Owner Acceptance Tests

Run with local PostgreSQL, Redis, the supported CLP `1.17.11` executable, the
backend migrations, the API, the dedicated worker, and the frontend dev server.
Use test accounts and fixture data only; do not place cookies, CSRF tokens,
secrets, or personal data in logs or evidence.

### 1. Saved daily-diet collection

Steps:

1. Sign in as an entitled user and select at least two meals from autocomplete.
2. Add, reorder, edit quantities, and remove a meal; confirm the one-day
   aggregate updates from the server response.
3. Save the collection, reload it, replace it, and delete it.
4. Repeat the create request with the same `Idempotency-Key` and identical
   body; then reuse the key with a changed body.
5. Attempt to read or mutate the diet as a different user.

Accept when:

- the collection contains ordered meal entries with positive canonical
  quantities;
- protein, carbohydrates, fat, and calories are server-derived and match the
  current meal data;
- the exact create retry replays one saved diet and changed-body reuse
  conflicts;
- `PUT` and `DELETE` are safe absolute/stable-resource mutations; and
- another user cannot read or mutate the collection.

Traceability: `192`, `194`, `196`, `197`, `DESIGN-001`, `DESIGN-008`, and
`SW-REQ-006`.

### 2. Optimization submission and polling

Steps:

1. Select the saved collection as Daily Diet Alternative input and submit a
   tolerance and exclusions with a fresh `Idempotency-Key`; confirm its current
   server-calculated aggregate macros are displayed read-only.
2. Confirm the API returns `202 Accepted`, a server-created job ID, and a poll
   URL without running a solver in the API process.
3. Poll as the owner through queued, processing, and terminal states.
4. Retry the same request after an ambiguous response and then retry with a
   changed body using the same key.
5. Poll the job as another user and after the one-hour result TTL fixture has
   expired.

Accept when:

- active trial/paid users are accepted and anonymous/free users are denied
  before queue side effects;
- exact retries return the original acknowledgement, changed-body reuse
  conflicts, and states never regress;
- only the submitting owner can see job data;
- a completed job returns one to three validated alternatives; and
- expiry, cross-user access, and queue outage produce stable safe responses.

Traceability: `193`, `204`, `205`, `210`, `ARCH-004`, `DESIGN-004`, and
`SW-REQ-021`/`SW-REQ-022`/`SW-REQ-023`/`SW-REQ-030`.

### 3. Solver, constraints, and degraded paths

Steps:

1. Run a feasible fixture with a non-zero macro tolerance and verify every
   returned alternative remains within the tolerance band.
2. Compare two feasible fixtures and confirm lower-calorie alternatives rank
   first.
3. Confirm repeated alternatives are distinct, original-meal overlap is only
   penalized, exclusions are honored, and no more than three alternatives are
   returned.
4. Run infeasible, 30-second timeout, malformed/invalid-input, and unavailable
   queue fixtures.
5. Inspect worker readiness and bounded queue/worker metrics.

Accept when:

- invalid, non-finite, negative, excluded, unknown, and tolerance-violating
  quantities are never published;
- infeasible and timeout failures are safe and preserve valid partial results
  where available;
- queue failure returns without synchronous solving;
- the worker uses the packaged CLP `1.17.11` child process, cleans its private
  temporary directory, and stops at the hard 30-second deadline; and
- metrics/readiness expose only bounded operational labels and no diet contents,
  job IDs, user IDs, or solver diagnostics.

Traceability: `198`-`203`, `206`, `208`, `ARCH-004`, `DESIGN-004`,
`DESIGN-014`, and `SW-REQ-021`/`SW-REQ-022`/`SW-REQ-023`/`SW-REQ-030`/
`SW-REQ-080`/`SW-REQ-082`.

### 4. Browser, responsive, and accessibility workflow

Steps:

1. Run the real desktop and mobile Chromium workflows from authenticated meal
   selection through saved collection, optimization, polling, retry, and
   result display.
2. Repeat anonymous, free, trial, paid, infeasible, timeout, and expired-result
   fixtures.
3. Complete the workflow with keyboard-only navigation and inspect focus,
   mobile layout, loading/error states, and axe output.

Accept when:

- the old manual UUID-entry scaffold is absent from the finished user workflow;
- no horizontal overflow or clipped controls occur on mobile;
- keyboard operation and focus order remain usable;
- no serious or critical axe violations are reported; and
- results, macros, calories, safe errors, and retry states do not leak stale
  data after changing the selected diet.

Traceability: `195`-`197`, `205`, `207`, `DESIGN-001`, and `SW-REQ-006`.

## Phase 07 Acceptance Decision

The automated evidence and documentation are ready for project-owner review
when the two validators above pass and the owner accepts the four checks in
this document. Task-list status is intentionally unchanged by Task 211.

The Phase 07 open-point register is dispositioned, but the following explicit
follow-ups remain outside the completed Phase 07 implementation:

- account-deletion cancellation/invalidation of optimization Redis keys is
  deferred to the backend/platform maintainer for 2026-07-18;
- the optional WebSocket notification path is deferred to the
  product/architecture owner for review on 2026-07-18; and
- the dedicated optimizer image is currently limited to `linux/amd64`; an
  ARM64 deployment would require a separately verified source build because
  upstream does not publish a matching Ubuntu artifact.

These are not hidden exceptions: each is recorded with its owner, target date,
and focused verification in `docs/implementation/04_OPEN.md`.

## Traceability

Primary design sources:

- `docs/design/DESIGN-001.md`: `SearchView`, saved-diet selection, and browser
  workflow state.
- `docs/design/DESIGN-004.md`: `LPSolverWrapper`, `ConstraintBuilder`,
  `ObjectiveFunction`, `DiversityPenalizer`, `SolutionValidator`,
  `JobQueueManager`, and `JobStatusTracker`.
- `docs/design/DESIGN-008.md`: `SavedDataRepository` and authenticated
  ownership.
- `docs/design/DESIGN-014.md`: `MetricsCollector` and readiness/telemetry.
- `docs/architecture/ARCH-004.md`: asynchronous LP optimization architecture.
- `docs/testing/integration/ARCH-004-obligations.md`: SWE.5 obligations
  `IT-ARCH-004-001` through `IT-ARCH-004-008`.

Phase task coverage: `192`-`211`.

Requirement coverage: `SW-REQ-006`, `SW-REQ-021`, `SW-REQ-022`, `SW-REQ-023`,
`SW-REQ-030`, `SW-REQ-080`, and `SW-REQ-082`.

## Known Notes

- The Phase 07 coverage deviations are accepted only for the specific
  defensive, dependency-failure, process-bootstrap, timer, abort, rollback,
  and teardown branches listed in `docs/implementation/04_OPEN.md`.
- The dedicated `linux/amd64` optimizer image packages the CGO-disabled Go
  worker and the checksum-pinned official Ubuntu 24 CLP `1.17.11` artifact;
  its no-cache focused verification passes.
- Account deletion currently purges the generic user cache namespace; it does
  not yet proactively invalidate optimization job namespaces. The owner/date
  follow-up is documented above and is not represented as completed evidence.
