# Task 240 Preparation and Repair Evidence

## Outcome

Task 240 is implemented, repaired against both independent-review passes, and verified. The task-list row remains `PREPARED`; this work did not edit the task list or any status cell.

> 240 | Phase 08 Custom Item Erasure Integration | DESIGN-008: AccountDeleter | PREPARED | deps 113,238,239

Dependencies 113, 238, and 239 remain `PASSED`.

## Baseline and scope control

- Repository: `/home/wiktor/Work/mealswapp`
- Fixed repository reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`
- Independent review: `docs/implementation/reviews/task-240-review.md`
- Current review hash at remaining-findings repair start: `542a4ee4bd053cd8e2b3f08f1beb55be551ec995da8b75c9838c5ccb9a59b835`
- Task-list hash at repair start and completion: `e651504f51193ca89e90cf99e144600280bdcfc44b83266a4347327ed1bd76ac`
- Preparation hash at repair start: `babbf403f93016cbf74644ab350957faa6b83463ddfed2b168996e9eb715762e`
- Migration-26 up hash at repair start: `ee1caaebdd189dfe6566a7f878f1613596edb2aaaa933cb6267fb4e13c8d74b7`
- Deletion claim SQL hash at repair start: `151059d6fcb8a0a15da8b58ebc10a5e304c92649d02c124fd0fe8924530893c4`
- Production worker hash at repair start: `a4704e209f30b173fccdce7ada0821ab411aa837d56c9f8c51ed11ef06049521`
- Fixed baseline commit for both repair passes: `81ca40ce00cb667ea29243ed2d34068e11229a69`

The working tree was already dirty with task 238/239 implementation, generated API changes, design/open-point edits, and review/preparation files. Those changes were preserved. This repair changed only task-240 production, migration, test, and evidence paths listed below. Writable subagents were unavailable, so the phase-orchestrator preparation contract was executed directly; no review decision or status transition was made.

## Sources inspected

- `docs/implementation/reviews/task-240-review.md`, including all three exact findings and locations.
- `docs/design/DESIGN-008.md` AccountDeleter ownership, lifecycle, erasure, cache purge, and completion rules.
- `docs/requirements/01_SOFT_REQ_SPEC.md`, especially SW-REQ-043 and SW-REQ-073.
- Account deletion service, repository contracts, request/claim/failure/completion SQL, migrations, API controller, worker bootstrap, Redis cache code, session/account deletion, export/login/profile access, and custom-item service/repository/SQL.
- Task 238/239 preparation and review evidence, to preserve their overlap and ownership boundaries.

## Independent-review repairs

### Migration upgrade lockout

Migration 26 now idempotently backfills `users.deletion_requested_at` from the earliest request time for every non-completed deletion request before custom-item writes can resume. Including `failed` as well as the review's `pending`/`processing` cases preserves the established invariant that deletion remains locked until completion. The migration recreates the deletion-claim index with stale `processing` support; the down migration restores the original predicate.

`TestTask240MigrationUpgradeBackfillsDeletionLockout` creates users and custom items at schema version 25, inserts pending and processing requests, applies migration 26, verifies the exact marker timestamps, and proves direct create/update/delete are blocked without mutating existing items.

### Production deletion worker

The production `cmd/worker` process now composes `AccountDeletionService` with PostgreSQL compliance/session/account repositories and the shared Redis user purger. A cancellable scheduler runs immediately and every 30 seconds alongside the optimization worker under one `errgroup`; operational cycle failures emit fixed low-cardinality metrics and retry on the next cycle. The scheduler lives in the Phase 08 `internal/deletionworker` package, so it does not alter the historical Phase 07 worker coverage exception.

`TestTask240APIToProductionDeletionWorker` submits `DELETE /account` through the production API and lets the production scheduler boundary claim and finish it. It never calls `ProcessDueDeletionRequests` directly. Completion and cache erasure are awaited and asserted.

### Stale processing recovery

Claiming a request now assigns a five-minute processing lease in `next_attempt_at`. The claim query can reclaim a `processing` row only when that lease expires. Thus, if deletion execution fails and failure-state persistence also fails, the request is temporarily leased rather than permanently stranded. A normal failure record still replaces the lease with the existing classified retry schedule.

`TestTask240ProcessingLeaseRecoversFailureRecordOutage` injects both an execution failure and the first `RecordDeletionFailure` outage, proves no early reclaim, then reclaims at the lease boundary and completes safely. The repository hardening test also verifies stale processing recovery and non-reclaim after completion.

## Remaining-review repairs

### Deletion-marker/write serialization

The four custom-item write statements now acquire `FOR NO KEY UPDATE` on the owner row before inserting, updating, or soft-deleting. PostgreSQL conflicts that lock with the deletion request's owner-marker `UPDATE`; consequently, a write already holding the owner lock serializes before marker creation, while every write starting after marker creation observes the marker and fails. The prior `FOR KEY SHARE` mode was compatible with a non-key `UPDATE` and did not establish that boundary.

`TestTask240ConcurrentCustomWriteSerializesBeforeDeletionMarker` holds the production create SQL inside a live transaction while production `DELETE /api/v1/account` runs concurrently. It proves the marker update waits, then commits the pre-marker custom item, completes deletion initiation, and rejects a subsequent repository create.

### Attempt-owned processing lease

Each claim now records a configurable lease deadline in `next_attempt_at`. That exact timestamp is the attempt token: failure and completion updates include `WHERE next_attempt_at = claimed_lease`, so an expired processor cannot finalize after another worker reclaims the request. `ExecuteDeletion` runs all destructive work under a context deadline equal to its claimed lease and preserves cancellation/deadline errors. The production lease remains five minutes; the duration option exists only to make the live overlap regression deterministic and fast.

`TestTask240ExpiredAttemptCannotFinalizeReclaimedWork` starts two workers against one live request. It proves worker two cannot claim an active lease, worker one is cancelled at expiry, worker two reclaims and completes, and worker one's stale failure update is rejected without overwriting completion.

### Direct Redis purger verification

`UserPurger` now uses a narrow Redis client interface while production construction remains typed as `*redis.Client`. Direct tests cover multi-page SCAN pagination, exact owner-prefix deletion, cross-user isolation, context cancellation during SCAN and DEL, Redis errors from both operations, nil/empty inputs, and a live Redis run with 150 owner keys. `NewUserPurger` and `UserPurger.PurgeUser` both have 100% direct function coverage.

## Complete task-240 changed-path and executable-symbol ledger

| Path | Added or modified executable symbols/statements |
|---|---|
| `backend/cmd/worker/main.go` | modified `main`: production deletion service composition and concurrent scheduler lifecycle |
| `backend/internal/app/app.go` | modified `redisCachePurger.PurgeUser`: delegates through `cache.NewUserPurger`, preserving nil-client behavior |
| `backend/internal/app/task240_custom_item_erasure_integration_test.go` | added/modified `task240FailingCachePurger`, `task240FailingCachePurger.PurgeUser`, `task240ExpiredAttemptPurger`, `task240ExpiredAttemptPurger.PurgeUser`, `task240FailureRecordRepository`, `(*task240FailureRecordRepository).RecordDeletionFailure`, `TestTask240CustomItemErasureIntegration`, `TestTask240MigrationUpgradeBackfillsDeletionLockout`, `TestTask240APIToProductionDeletionWorker`, `TestTask240ProcessingLeaseRecoversFailureRecordOutage`, `TestTask240ConcurrentCustomWriteSerializesBeforeDeletionMarker`, `TestTask240ExpiredAttemptCannotFinalizeReclaimedWork`, and the task-240 test helpers |
| `backend/internal/cache/user_purger.go` | added/modified `userPurgeClient`, `UserPurger`, `NewUserPurger`, `UserPurger.PurgeUser` |
| `backend/internal/cache/user_purger_test.go` | added `userPurgeScanPage`, `userPurgeClientStub`, `userPurgeClientStub.Scan`, `userPurgeClientStub.Del`, `TestNewUserPurger`, `TestUserPurgerRejectsMissingInputs`, `TestUserPurgerPaginatesAndIsolatesNamespace`, `TestUserPurgerPropagatesRedisErrors`, `TestUserPurgerHonorsCancellation`, `TestUserPurgerLiveRedisPaginationAndIsolation` |
| `backend/internal/deletionworker/account_deletion.go` | added `MetricAccountDeletionCycles`, `MetricAccountDeletionRequests`, `AccountDeletionProcessor`, `RunAccountDeletionProcessor`, `recordAccountDeletionMetric` |
| `backend/internal/deletionworker/account_deletion_test.go` | added scheduler doubles and four `TestRunAccountDeletionProcessor...` tests; every production function is 100% covered |
| `backend/internal/repository/compliance_repository.go` | modified `PostgresComplianceRepository`, `NewPostgresComplianceRepository`, `ClaimDeletionRequests`, `RecordDeletionFailure`, `CompleteDeletionRequest`; added `WithDeletionLeaseDuration` |
| `backend/internal/repository/types.go` | modified `DeletionRequestRepository` attempt-token contract |
| `backend/internal/repository/account_repository_coverage_test.go` | modified compliance repository coverage tests for lease configuration and token validation |
| `backend/internal/repository/postgres_repository_test.go` | modified `TestPostgresComplianceRepositoryDeletionHardening` for lease-token stale-finalization rejection, stale-processing reclaim, and completed-row exclusion |
| `backend/internal/repository/repository_test.go` | modified deletion repository contract fake and assertions for lease tokens |
| `backend/internal/repository/sql/deletion_claim.sql` | modified `deletionClaimSQL`: parameterized claim lease and due stale-processing reclaim |
| `backend/internal/repository/sql/deletion_fail.sql` | modified `deletionFailSQL`: failure finalization guarded by the claimed lease timestamp |
| `backend/internal/repository/sql/deletion_complete.sql` | modified `deletionCompleteSQL`: completion guarded by the claimed lease timestamp |
| `backend/internal/repository/sql/deletion_request.sql` | modified `deletionRequestSQL`: atomic durable deletion marker/request creation |
| `backend/internal/repository/sql/session_revoke_user.sql` | modified `sessionRevokeUserSQL`: retry-safe no-row completion |
| `backend/internal/repository/sql/custom_food_create.sql` | modified `customFoodCreateSQL`: active-owner lockout plus `FOR NO KEY UPDATE` deletion-marker serialization |
| `backend/internal/repository/sql/custom_food_create_claim.sql` | modified `customFoodCreateClaimSQL`: active-owner idempotent-create lockout plus `FOR NO KEY UPDATE` serialization |
| `backend/internal/repository/sql/custom_food_update.sql` | modified `customFoodUpdateSQL`: active-owner update lockout plus `FOR NO KEY UPDATE` serialization |
| `backend/internal/repository/sql/custom_food_soft_delete.sql` | modified `customFoodSoftDeleteSQL`: active-owner delete lockout plus `FOR NO KEY UPDATE` serialization |
| `backend/internal/userdata/deletion.go` | modified `AccountDeletionService.ExecuteDeletion`, `AccountDeletionService.ProcessDueDeletionRequests`: lease-deadline execution and attempt-token finalization |
| `backend/internal/userdata/deletion_test.go` | modified deletion repository/cache doubles and `TestAccountDeletionService`; added lease/deadline/cancellation regression subtests |
| `database/migrations/000026_custom_item_erasure_integration.up.sql` | added/modified migration-26 statements: marker, upgrade backfill, FK removal, stale-processing claim index |
| `database/migrations/000026_custom_item_erasure_integration.down.sql` | added/modified rollback statements: FK restoration, original claim index, marker/version removal |
| `docs/implementation/preparations/task-240.md` | evidence only; no executable symbol |

No API schema, frontend, design, open-point, review, or task-list path was edited for this repair.

## Acceptance evidence

| Criterion | Verified evidence |
|---|---|
| Pending deletion blocks custom-item writes | Original live integration proves API/idempotent/direct create, update, and delete lockout. Upgrade regression proves pending and processing rows created before migration 26 receive the marker and all direct writes remain blocked. Concurrent live regression proves the owner lock establishes a strict write-before-marker or marker-before-write order. |
| Completion removes all owned custom items and owner cache | Original live integration proves owner rows and exact/collection/detail Redis keys disappear. API-to-worker regression proves the production scheduler reaches completion and purges cache. |
| Retries and recovery are safe | Classified cache failure remains retryable; the processing lease prevents early duplicate work and permits reclaim after expiry; lease-token guards prevent expired workers from recording failure or completion after reclaim; completed rows are never reclaimed. |
| Cross-user/global data survives | Original integration compares another user's custom row and a curated `food_items` row byte-for-byte before/after deletion. Claim predicates and cache prefix remain owner-scoped. |
| Access fails after completion | Original integration proves stale export/profile/custom-item access and password login return 401 after identity/account deletion. |
| Receipt is pseudonymous | Original integration proves completed request has null `user_id`, a receipt UUID, and no owner UUID, custom-item UUID/name, `owner_id`, or `custom_food` content. |
| Production path exists | `cmd/worker/main.go` is now a real caller of `RunAccountDeletionProcessor`; local-stack verification starts that binary successfully. |

## Security and concurrency review

- All SQL remains parameterized; no dynamic identifiers or user-controlled Redis glob syntax were added.
- Upgrade backfill is set-based, idempotent, and preserves an existing marker with `coalesce`.
- Custom-item owner predicates acquire `FOR NO KEY UPDATE`, which conflicts with the deletion-marker update while never targeting global curated storage.
- Processing reclaim requires a server-assigned lease deadline; concurrent claimers retain `FOR UPDATE SKIP LOCKED`, and completion/failure require ownership of the exact claimed deadline.
- Active execution is bounded by that lease deadline, preventing an expired attempt from continuing cache or database work as the current owner.
- Worker metric names, units, and labels are fixed and contain no owner/request data. Operational errors are retried without logging PII.
- Scheduler cancellation propagates through the shared signal context. Redis scan/delete operations receive that context.
- `govulncheck` found no reachable vulnerability in called code or imported packages.

## Verification commands and results

| Command | Result |
|---|---|
| `go test ./internal/cache ./internal/app ./internal/userdata ./internal/repository -run 'TestUserPurger\|TestTask240\|TestAccountDeletionService\|TestPostgresComplianceRepositoryDeletionHardening' -count=1` | PASS; concurrent write, two-worker lease ownership, direct Redis purger, upgrade, API-to-worker, recovery, and original task tests included |
| focused command above with `-race` | PASS |
| `python3 scripts/verify-local-stack.py` | PASS; real migration up/down/up, API, worker, health, and readiness |
| `go test ./...` | PASS for all backend packages |
| `go test -race ./...` | PASS for all backend packages, including `internal/deletionworker` |
| `go vet ./...` | PASS |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS; zero reachable vulnerabilities |
| focused function coverage profiles | PASS; `NewUserPurger`, `UserPurger.PurgeUser`, `NewPostgresComplianceRepository`, `WithDeletionLeaseDuration`, `ClaimDeletionRequests`, `RecordDeletionFailure`, `CompleteDeletionRequest`, and every function in `userdata/deletion.go` are 100% covered |
| `python3 scripts/validate-traceability.py` | PASS |
| `python3 scripts/validate-task-list.py` | PASS; 263 sequential tasks with ordered dependencies; task 240 remains `PREPARED` |
| `git diff --check` and `gofmt` on task paths | PASS |
| `python3 scripts/check.py` | PASS, exit 0; aggregate includes traceability/task validation, Go Doc, OpenAPI lint, security, migration/local-stack/UAT, full backend tests/race/vet/coverage, frontend build/unit/coverage, and Playwright. Frontend unit: 459 passed. |

The aggregate retains the pre-existing, explicitly ignored OpenAPI 302 warning. The dedicated Phase 08 deletion worker is 100% covered; the historical Phase 07 `internal/worker` package remains exactly at its documented 67.4% exception.

## Final implementation hashes

| Path | SHA-256 |
|---|---|
| `backend/cmd/worker/main.go` | `07211c3c0a6bdeef0fbfb12e43fa34fa1aaeb7ee0ad08b1fbf8f103775e7c74d` |
| `backend/internal/app/app.go` | `45de7f3b7de3515120e2a94c9eb3c40dfbbed95d8e29f64fa19f94f83e92d97b` |
| `backend/internal/app/task240_custom_item_erasure_integration_test.go` | `7e6a0af9cbe2e17937e3cf6b85f5d874820f08f8a4401d132fac117c1c998948` |
| `backend/internal/cache/user_purger.go` | `1ee0ccc70faf86a731d770cf549a3f9e55fff4aa00c5e2027f3929bf4a9e6e62` |
| `backend/internal/cache/user_purger_test.go` | `6f07cf7eb09cb925e2759157d8b3302719fc03d26f73976821367b96e58f6d72` |
| `backend/internal/deletionworker/account_deletion.go` | `4474c633b2cc1c4724cbf3ef2df980a95429df3b68af6a8a7f5a48c598b78467` |
| `backend/internal/deletionworker/account_deletion_test.go` | `f55c1be75603c96c40383c6ed3e4523b9dc8d97fa7c56cfe627f2a5982f4e661` |
| `backend/internal/repository/compliance_repository.go` | `c864ec7ca992b3182f6778f71a2308a7934ea13dd6b76be01439be9830b506e3` |
| `backend/internal/repository/types.go` | `a959e11147b0ff709b885ae0c58870c3628ad545cfd99f8e5c53efb5f1116cde` |
| `backend/internal/repository/postgres_repository_test.go` | `1340709b50d180000495275c8c1209f87b5bb4f56d3b5824b80fdec8bd1d137a` |
| `backend/internal/repository/sql/deletion_claim.sql` | `84aa4f1083f989e160b82f26b5eb87f4e0121545a9f5c74599f3430b38bbdd27` |
| `backend/internal/repository/sql/deletion_fail.sql` | `c40e27694a16ffac97ae3ba1c9ffea9f51a3bef8dbe613622dafd66be02d3fa2` |
| `backend/internal/repository/sql/deletion_complete.sql` | `4b79c009fe562bdf08db513710cf97e08ad000f0d34be220c2c24e65b49a1d34` |
| `backend/internal/repository/sql/deletion_request.sql` | `f32989a58246943dda9fb74159e49d7f436c3a7a101662b008a65d2558bdf842` |
| `backend/internal/repository/sql/session_revoke_user.sql` | `6a11f789b8037bff4149b6d0c09bb8e65e553b25d493c8c9c52844ef765d7960` |
| `backend/internal/repository/sql/custom_food_create.sql` | `303e7e223ea997dcdad08d8129cd98981599b814e850cbc2a0caa849766f64e0` |
| `backend/internal/repository/sql/custom_food_create_claim.sql` | `424b3a4eb0f54eb3fcd2af919ba254cfe4618e88e9d526327f6db7f9f0186a8f` |
| `backend/internal/repository/sql/custom_food_update.sql` | `4e862b2fffd3b198e47a8d342e6612f7c68fc3d436d25bfc5b365f4cd28bed42` |
| `backend/internal/repository/sql/custom_food_soft_delete.sql` | `9fd2a13a0701a2822d875bb334f70ef3d3e15e88fe126e7c559ef2281013916e` |
| `backend/internal/userdata/deletion.go` | `3959b9bec47cd5f461e7ac47b79be5a68faa5f0e3339bbb5a831520f56db9f47` |
| `backend/internal/userdata/deletion_test.go` | `7dba8003ca03a87836ef86f47d9a1a18be21c4aced25cef798a7279071054bb0` |
| `database/migrations/000026_custom_item_erasure_integration.up.sql` | `219fffeefbe247c298b48c88fc1d880782744da722e828ace2d9b0989d415f91` |
| `database/migrations/000026_custom_item_erasure_integration.down.sql` | `ca178712ba75ef96b9912f4b0aaf0c0820e39f407642bebd76da48f0177e1eca` |
| `docs/implementation/02_TASK_LIST.md` | `e651504f51193ca89e90cf99e144600280bdcfc44b83266a4347327ed1bd76ac` |
| `docs/implementation/reviews/task-240-review.md` | `542a4ee4bd053cd8e2b3f08f1beb55be551ec995da8b75c9838c5ccb9a59b835` |

## Residual risk

No task-240 blocker remains. Redis SCAN/DEL is bounded per batch but is not a server-side atomic namespace delete; account/session revocation, durable deletion lockout, and account deletion prevent current task-owned entries from being recreated. Any future cache writer must continue to honor the deletion marker or an equivalent generation/tombstone policy.
