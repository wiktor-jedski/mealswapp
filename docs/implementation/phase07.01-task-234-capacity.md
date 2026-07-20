# Phase 07.01 Task 234: Observability and Capacity Regression Gate

<!-- Implements DESIGN-014 MetricsCollector and LogAggregator. -->

This gate protects the repaired Phase 07.01 submission, queue, worker, and
solver boundaries. It is repeatable at two levels: deterministic local
failure/load fixtures and an authenticated deployment-facing responsiveness
check.

## Deterministic regression profile

Run from the repository root with PostgreSQL and Redis available:

```sh
bash scripts/start-services.sh
python3 scripts/verify-phase0701-observability-capacity.py
```

The gate runs normal and race-detector variants and fails if a required Redis
or isolated Redis-restart fixture is skipped. Its fixed workload covers:

- eight concurrent unrelated submissions and eight exact same-key replays;
- original submission, exact replay, and polling batches below the 2-second
  critical latency boundary, with every processing job polled while a
  background worker fixture remains active;
- failed admission release preserving the primary `503 queue_unavailable`
  response and emitting only bounded cleanup telemetry;
- one waiting plus one pending Redis delivery with positive authoritative ages,
  three retry attempts, exact `retry`, `retry`, `exhausted` outcomes, consumer
  group recovery, and a final empty queue with zero ages;
- a solver deadline followed by failed admission release, preserving the
  published `solver_timeout` state and exact worker/solve/job telemetry;
- bounded solver-directory and queue-lock cleanup failures, real child-process
  timeout cleanup, and Redis restart recovery;
- exact metric/log allowlists and generic sink-failure fallback records with no
  user, diet, meal, job, stream, idempotency-key, body, or diagnostic content.

## Authenticated capacity profile

The operator check defaults to 32 unrelated submissions at concurrency 8. Each
accepted submission is immediately replayed with the exact same idempotency key
and body, then polled while `/ready` is sampled in parallel:

```sh
MEALSWAPP_CAPACITY_COOKIE='...' \
MEALSWAPP_CAPACITY_CSRF_TOKEN='...' \
python3 scripts/verify-optimization-capacity.py \
  --body-file /path/to/optimization-fixture.json \
  --output logs/optimization-capacity.json
```

The report contains only counts, booleans, durations, terminal status counts,
and queue/worker readiness aggregates. It never includes credentials,
idempotency keys, request bodies, poll URLs, job IDs, diet IDs, or user IDs.

The check fails unless every original and replay returns `202`, every replay
matches the original acknowledgement data, original/replay/poll P95 latency is
strictly below 2 seconds, poll samples exist, every readiness sample is healthy
and well-formed, and queue/worker evidence is present. The warning threshold
remains P95 above 1.5 seconds; 2 seconds is the critical/failing boundary from
DESIGN-014. The default is an isolation regression gate, not a production
throughput claim for SW-REQ-082; production sizing must rerun the authenticated
profile against the deployed API and worker topology.
