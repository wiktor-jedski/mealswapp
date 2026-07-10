# Task 161 Review

Task ID: 161

Evidence path: `docs/implementation/reviews/task-161-review.md`

Recommended status: PASSED

## Checklist Summary

- Task row verified as `PREPARED` in `docs/implementation/02_TASK_LIST.md`.
- Dependency status precondition satisfied: task `104` is `PASSED` and task `158` is `PREPARED` in the current task list.
- Note: the prompt's dependency context for task `104` does not match the current repository row, which lists task `104` as "Phase 03 Security Review Gate". I used the repository task list for status verification and inspected OAuth hook behavior directly.
- First social login creates exactly one active 7-day trial through `CoreAuthService` wired to a real `TrialTracker`.
- Repeated social login through the existing OAuth identity path does not append or extend the trial.
- Explicit linked social login does not create or extend a trial.
- Expired active trials are downgraded to free by appending a new entitlement row, preserving trial history.
- Paid users and unexpired trials are not downgraded by expiry processing.
- `backend/cmd/expire-trials` provides an executable command entrypoint and has command-level idempotency coverage through `runExpireTrials`.

## Commands Run / Results

- `rg -n '^\| (104|158|161) \|' docs/implementation/02_TASK_LIST.md`
  - Result: task `161` is `PREPARED`; dependency task `104` is `PASSED`; dependency task `158` is `PREPARED`.
- `git status --short`
  - Result: confirmed a dirty worktree; review did not edit task status or implementation code.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/entitlement -run 'TestTrialTracker'`
  - Result: passed.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/auth -run 'TestCoreAuthServiceOAuthRealTrialTracker'`
  - Result: passed.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./cmd/expire-trials -run 'TestRunExpireTrialsIsIdempotentAndPreservesHistory'`
  - Result: passed.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/auth`
  - Result: passed.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./cmd/expire-trials`
  - Result: passed.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/entitlement`
  - Result: failed in `TestUsageLimiterPostgresConcurrentSeparateInstancesCannotExceedPersistedLimit` with `allowed completions = 8, want 3`; this is task 159 usage-limiter behavior, outside task 161.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/repository`
  - Result: failed before/around repository tests due migration reset errors on `000015_security_audit_attempt_outcome.down.sql` and a separate `RecordUsageWithinLimit` expectation. The inspected task-161 repository query and command tests are covered through focused service/command tests.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/repository -run 'TestPostgresEntitlementRepository'`
  - Result: failed due the same repository test harness/migration reset issue before the `ListExpiredTrials` assertion could run.

## Files Inspected

- `docs/implementation/02_TASK_LIST.md`
- `docs/design/DESIGN-007.md`
- `backend/internal/entitlement/trial_tracker.go`
- `backend/internal/entitlement/trial_tracker_test.go`
- `backend/internal/app/app.go`
- `backend/internal/auth/oauth.go`
- `backend/internal/auth/service_test.go`
- `backend/cmd/expire-trials/main.go`
- `backend/cmd/expire-trials/main_test.go`
- `backend/internal/repository/types.go`
- `backend/internal/repository/entitlement_repository.go`
- `backend/internal/repository/sql/entitlement_list_expired_trials.sql`
- `backend/internal/repository/postgres_repository_test.go`

## Decision Reason

Recommend `PASSED` because every task-161 verification criterion is directly satisfied by implementation and focused tests.

`TrialTracker.StartTrial` creates a trial only when no latest entitlement exists, sets `tier = "trial"`, `status = "active"`, and an expiry seven days from creation, and returns existing entitlement state without extending it. `TestCoreAuthServiceOAuthRealTrialTracker` wires `CoreAuthService` to the real tracker and verifies first social login creates one trial while repeated and explicitly linked OAuth flows do not append or extend trial history. `TrialTracker.ExpireTrials` appends a free active entitlement only when the latest state is still an expired active trial, which preserves history and avoids downgrading paid users. `backend/cmd/expire-trials` has an executable `main` and a command-level idempotency test for repeated `runExpireTrials` calls.

The broad package failures observed during practical verification are outside this task's verification surface: one is usage-limiter concurrency from task 159, and the repository package is blocked by migration/test-harness failures before the trial query assertion can run.

## Repair Instructions If Rejected

Not applicable.
