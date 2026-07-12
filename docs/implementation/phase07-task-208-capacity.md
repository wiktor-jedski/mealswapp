# Phase 07 Task 208: Queue Observability and Capacity Gate

This document records the operational limits and repeatable evidence for
`DESIGN-014: MetricsCollector`, relevant to `SW-REQ-080`, `SW-REQ-081`, and
`SW-REQ-082`.

## Bounded telemetry

Optimization metrics use fixed names and allow-listed values only. The labels
are limited to `outcome`, `kind`, `pool`, and `status`; no metric or
optimization log contains a job ID, diet ID, user ID, request body, diet
contents, meal IDs, solver output, cookie, or token.

The production API records submission outcomes, queue depth and oldest queued /
pending age during readiness checks, worker heartbeat readiness, and result-TTL
expiry. The dedicated worker records active workers, utilization, solver
duration/status, job outcomes, retries, timeout outcomes, and infeasible
outcomes.

`GET /ready` reports `postgres`, `redis`, `worker`, and
`optimization_queue` independently when those dependencies are configured. A
missing or stale worker heartbeat is `worker: unavailable` and returns HTTP
503; Redis and queue failures use the same degraded readiness response.

## Alert thresholds

| Signal | Warning | Critical | Window |
| --- | ---: | ---: | ---: |
| API P95 latency | > 1.5 s | > 2 s | 60 s |
| Optimization queue depth | > 20 jobs | deployment-specific | 60 s |
| Oldest queue age | > 5 s | > 15 s | 60 s |
| Worker utilization | > 70% | > 90% | 300 s |
| Solver duration | — | >= 30 s | 1 s |

The 30-second solver alert is aligned with the hard CLP deadline. Queue-depth
thresholds are local single-worker defaults and must be recalibrated when the
worker pool is scaled; the bounded metric vocabulary does not change.

## Accepted environment limits

- API request timeout: `MEALSWAPP_API_TIMEOUT`, positive, development default
  10 seconds.
- CLP solve deadline: 30 seconds maximum.
- Redis visibility timeout: greater than 30 seconds, production default 45
  seconds.
- Logical optimization retry budget: 3 attempts.
- Job/result retention: 1 hour; expired polling is reported as `410` to the
  owner and `404` to other users.
- Worker heartbeat: refresh every 5 seconds; stale after 15 seconds.
- The local capacity check defaults to 32 submissions at concurrency 8. This
  is a repeatable isolation/responsiveness gate, not a claim that one local
  process proves 1,000 users; `SW-REQ-082` production capacity requires
  horizontal API and worker scaling with the same queue evidence.

## Repeatable check

Provision one authenticated saved-diet fixture, then provide its cookie,
CSRF token, and JSON submission body without putting them in the report:

```sh
MEALSWAPP_CAPACITY_COOKIE='...' \
MEALSWAPP_CAPACITY_CSRF_TOKEN='...' \
python3 scripts/verify-optimization-capacity.py \
  --body-file /path/to/optimization-fixture.json \
  --output logs/optimization-capacity.json
```

The check submits distinct idempotency keys concurrently, polls every accepted
job while the worker is solving, probes `/ready` in parallel, and reports only
submission/poll P95 latency, HTTP status counts, terminal status counts, and
the observed Redis/worker/queue readiness tuple. It exits non-zero when all
submissions are not accepted, either submission/poll P95 reaches 2 seconds,
poll samples are absent, the readiness monitor fails, any readiness sample is
degraded/malformed, or queue/worker evidence is absent. A valid readiness
sample requires HTTP 200, `redis: ok`, `worker: ok`,
`optimization_queue: ok`, and non-negative queue depth/age values.
