# Review Evidence: Task 240 — AccountDeleter

```yaml
task_id: 240
component: "AccountDeleter"
static_aspect: "DESIGN-008 AccountDeleter"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-21T04:37:18Z"
review_agent: "Codex GPT-5 independent re-review"
evidence_file: "docs/implementation/reviews/task-240-review.md"
baseline_ref: "81ca40ce00cb667ea29243ed2d34068e11229a69"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "code-review-skill Go, SQL injection prevention, async concurrency patterns, and security review guide"
repair_context_required: true
```

## 1. Task Source

**Description:** Phase 08 Custom Item Erasure Integration: include user-owned custom items in account write lockout, transactional account-deletion cleanup, cache invalidation, and deletion verification without deleting or mutating global curated items.

**Depends On:** 113, 238, 239; all are PASSED.

**Testing Coverage Exceptions:** None for task-240 changed symbols. The aggregate retains the pre-existing documented Phase 07 `internal/worker` coverage exception; it is outside this task.

**Verification Criteria:** Account-deletion integration tests prove pending deletion blocks custom-item writes, completion removes every owned custom item and related user-scoped cache entry, retries remain safe, cross-user and global curated rows survive unchanged, export/login/profile/custom-item access fails after completion, and the pseudonymous receipt contains no custom-item or owner data. This re-review also verifies migration-26 upgrade backfill, production scheduler invocation, cancellation and retry recovery, stale processing leases, lock serialization, and direct Redis purger coverage.

## 2. Pre-Review Gates

- [x] Input status is `PREPARED`.
- [x] Every dependency is `PREPARED` or `PASSED`.
- [x] The preparation report claims completion.
- [x] A task-specific baseline/diff is available and trustworthy.
- [x] `code-review-skill` was invoked exactly once for this re-review and its relevant guides were read.
- [x] The reviewer is independent from implementation/repair.
- [x] Review uses current repository state rather than stale logs.
- [x] Reviewer made no production-code changes.

```yaml
pre_review_gates_passed: true
blocking_issue: "None"
```

## 3. Review Baseline and Change Surface

Baseline/reference method: fixed reference `81ca40ce00cb667ea29243ed2d34068e11229a69` from the preparation report was confirmed as the current repository baseline. The task-owned repair surface was reconstructed from that baseline, the current preparation report, current `git status`, and independent symbol/caller searches. Task 238 and 239 changes already present in the dirty worktree were treated as dependencies, not as new task-240 repair scope.

Commands used to reconstruct the diff:

```bash
git status --short
git diff --stat 81ca40ce00cb667ea29243ed2d34068e11229a69 --
git diff 81ca40ce00cb667ea29243ed2d34068e11229a69 -- <task-owned paths>
rg -n -C 3 'NewUserPurger|PurgeUser\(|RunAccountDeletionProcessor|ProcessDueDeletionRequests|RecordDeletionFailure|CompleteDeletionRequest|FOR NO KEY UPDATE|deletion_claim|deletion_complete|deletion_fail' backend docs/implementation/preparations/task-240.md
sed -n '1,260p' docs/implementation/preparations/task-240.md
sed -n '1,180p' docs/design/DESIGN-008.md
sed -n '1,120p' docs/design/DESIGN-015.md
rg -n -C 3 'SW-REQ-(043|073)|custom item|deletion|pseudonymous|receipt' docs/requirements docs/design
```

Pre-existing dirty-worktree changes and exclusions:

The worktree contains earlier Phase 08 custom-item persistence/API changes, generated API artifacts, design/open-point edits, and other review/preparation documents. Those were preserved. The task-list file was already modified by prior phase orchestration: dependencies 238 and 239 are PASSED and row 240 remains PREPARED. This re-review did not edit that file or any task status. The task-owned repair surface is the worker composition, shared purger, deletion lease/claim/failure/completion logic, four custom-item write statements, migration 26, and their direct regression tests. Unmodified task-238/239 repositories, services, routes, export, identity deletion, session revocation, and cascade migrations were inspected as callers/dependencies.

| Changed file or unit | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| `backend/cmd/worker/main.go` | task-240 production repair | HIGH | `main` deletion-service composition and scheduler goroutine |
| `backend/internal/app/app.go` | task-240 production cache wiring | HIGH | `redisCachePurger.PurgeUser` |
| `backend/internal/cache/user_purger.go` and tests | task-240 cache implementation and repair coverage | HIGH | Redis namespace scanner/deleter and direct adversarial tests |
| `backend/internal/deletionworker/account_deletion.go` and tests | task-240 production scheduler | HIGH | processor contract, polling loop, metrics, cancellation tests |
| `backend/internal/repository/compliance_repository.go`, `types.go`, and tests | task-240 lease-token repair | HIGH | claim, failure, completion, configurator, interface contract |
| deletion workflow SQL | task-240 lease and lockout repair | HIGH | request, claim, failure, completion, session-revoke statements |
| four custom-item write SQL statements | task-240 lock-compatibility repair | HIGH | create, idempotent-create claim, update, soft-delete owner CTEs |
| `backend/internal/userdata/deletion.go` and tests | task-240 bounded execution and guarded finalization | HIGH | execute/process orchestration and cancellation/retry tests |
| migration 26 up/down | task-240 upgrade/backfill repair | HIGH | marker backfill, stale-processing index, reversible rollback |
| task-240 integration tests | task-240 acceptance and previous-finding regressions | HIGH | migration, live lock race, production scheduler, outage, stale lease |

No task-owned change could not be distinguished reliably.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Pending deletion blocks create, idempotent-create claim, update, and soft-delete writes. | Live API and direct repository integration. | PASS | `TestTask240CustomItemErasureIntegration` proves all four fail while the marker is active and existing rows remain. |
| 2 | A write owning the user row lock before deletion serializes before the marker. | Real concurrent PostgreSQL transaction and `DELETE /account`. | PASS | `TestTask240ConcurrentCustomWriteSerializesBeforeDeletionMarker` observes the marker update wait, then post-marker rejection. |
| 3 | Migration 26 backfills pre-existing pending and processing lockout without mutating custom rows. | Down/up migration integration with exact timestamps. | PASS | `TestTask240MigrationUpgradeBackfillsDeletionLockout` verifies both markers and blocked writes; SQL covers all non-completed requests, including failed. |
| 4 | Completion removes every owned custom item and classification dependent transactionally with account deletion. | Cascade inspection and post-completion row count. | PASS | `user_delete_account.sql` deletes the user; migration 25 defines owner and classification cascades; live test observes zero owner rows. |
| 5 | Completion removes every owner cache key and preserves other namespaces. | Direct SCAN/DEL tests, live pagination/isolation, integration assertions. | PASS | Direct tests cover pages and prefix; live test uses 150 owner keys and another user key; owner keys disappear and other key remains. |
| 6 | Cache failure leaves a safe retryable state and retry completion is idempotent. | Failure injection and second/third processing attempts. | PASS | First failure records classified retry while owner data remains; next attempt completes and later claim returns zero. |
| 7 | Execution and failure-state persistence failure cannot strand processing permanently. | Inject both failures, inspect lease, reclaim at deadline. | PASS | `TestTask240ProcessingLeaseRecoversFailureRecordOutage` proves no early reclaim, exact-boundary reclaim, completion, and cache removal. |
| 8 | Bounded attempts cannot overlap or incorrectly reclaim another attempt. | Two-worker live test with purge blocked beyond lease. | PASS | `TestTask240ExpiredAttemptCannotFinalizeReclaimedWork` proves active lease protection, expiry reclaim, and safe completion. |
| 9 | Expired attempt cannot overwrite a reclaimed attempt. | Exact token predicates and stale finalization. | PASS | Failure and completion require claimed `next_attempt_at`; stale worker gets not-found while request stays completed. |
| 10 | Cancellation reaches database and Redis work and stops scheduler cleanly. | Context-aware purger tests, deadline block, scheduler cancellation, race suite. | PASS | SCAN and DEL cancellation are injected directly; deadline exits blocked purger; scheduler tests and full race pass. |
| 11 | Cross-user custom rows survive unchanged. | Before/after row serialization and owner predicates. | PASS | Live test compares another user's row JSON; all reads/writes/purge are owner-scoped. |
| 12 | Global curated rows survive unchanged. | Before/after curated-row serialization and storage inspection. | PASS | Live test compares a `food_items` row byte-for-byte; deletion targets user-owned data only. |
| 13 | Export fails after completion. | Stale authenticated export request. | PASS | Live test asserts JSON export returns 401. |
| 14 | Login fails after completion. | Password login with deleted account email. | PASS | Live test asserts 401. |
| 15 | Profile and custom-item access fail after completion. | Stale-cookie profile and item requests. | PASS | Live test asserts 401 for profile and custom-item access. |
| 16 | Receipt contains no owner or custom-item data. | Receipt JSON and database field assertions. | PASS | `user_id` is null, receipt ID is nonzero, and owner/item UUIDs, names, `owner_id`, and `custom_food` are absent. |
| 17 | Production worker invokes the deletion processor through a cancellable scheduler. | Production composition and API-to-scheduler integration. | PASS | `cmd/worker/main.go` composes all repositories and `RunAccountDeletionProcessor`; integration does not call `ProcessDueDeletionRequests` directly. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | `main` deletion-worker composition | function | `backend/cmd/worker/main.go:24-79` | modified | deployed worker | API-to-scheduler test and local stack |
| 2 | `redisCachePurger.PurgeUser` | method | `backend/internal/app/app.go:290-302` | modified | API deletion service | app and live tests |
| 3 | `userPurgeClient` | interface | `backend/internal/cache/user_purger.go:16-21` | added | purger, Redis client, stub | direct purger tests |
| 4 | `NewUserPurger` | function | `backend/internal/cache/user_purger.go:23-29` | added | app and worker | direct test and coverage |
| 5 | `UserPurger.PurgeUser` | method | `backend/internal/cache/user_purger.go:32-54` | added | deletion service | pagination, isolation, error, cancel, live tests |
| 6 | scheduler defaults and `AccountDeletionProcessor` | constants/interface | `backend/internal/deletionworker/account_deletion.go:12-29` | added | worker and tests | scheduler doubles |
| 7 | `RunAccountDeletionProcessor` | function | `backend/internal/deletionworker/account_deletion.go:31-75` | added | worker main | four scheduler tests and race suite |
| 8 | `recordAccountDeletionMetric` | function | `backend/internal/deletionworker/account_deletion.go:78-85` | added | scheduler | metric assertions |
| 9 | compliance repository constructor/default | type/function | `backend/internal/repository/compliance_repository.go:89-106` | modified | API and worker | integration and coverage |
| 10 | `WithDeletionLeaseDuration` | method | `backend/internal/repository/compliance_repository.go:108-115` | added | deterministic lease test | coverage test |
| 11 | `ClaimDeletionRequests` | method | `backend/internal/repository/compliance_repository.go:240-263` | modified | deletion processor | hardening and recovery tests |
| 12 | `RecordDeletionFailure` | method | `backend/internal/repository/compliance_repository.go:265-284` | modified | processor failure path | outage and stale-token tests |
| 13 | `CompleteDeletionRequest` | method | `backend/internal/repository/compliance_repository.go:286-302` | modified | deletion execution | receipt and stale-token tests |
| 14 | `DeletionRequestRepository` lease-token contract | interface | `backend/internal/repository/types.go:797-805` | modified | userdata and fakes | contract tests |
| 15 | `AccountDeletionService.ExecuteDeletion` | method | `backend/internal/userdata/deletion.go:46-72` | modified | processor loop | service, cancel, cache, stale tests |
| 16 | `AccountDeletionService.ProcessDueDeletionRequests` | method | `backend/internal/userdata/deletion.go:74-105` | modified | scheduler | recovery and two-worker tests |
| 17 | `customFoodCreateSQL` | SQL | `backend/internal/repository/sql/custom_food_create.sql:2-16` | modified | custom repository | lock and pending tests |
| 18 | `customFoodCreateClaimSQL` | SQL | `backend/internal/repository/sql/custom_food_create_claim.sql:2-12` | modified | idempotent create | pending integration |
| 19 | `customFoodUpdateSQL` | SQL | `backend/internal/repository/sql/custom_food_update.sql:2-25` | modified | custom repository | pending update and lock test |
| 20 | `customFoodSoftDeleteSQL` | SQL | `backend/internal/repository/sql/custom_food_soft_delete.sql:2-11` | modified | custom repository | pending delete and lock test |
| 21 | `deletionRequestSQL` | SQL | `backend/internal/repository/sql/deletion_request.sql:2-31` | modified | API deletion request | API and lock test |
| 22 | `deletionClaimSQL` | SQL | `backend/internal/repository/sql/deletion_claim.sql:2-30` | modified | compliance repository | hardening and recovery |
| 23 | `deletionFailSQL` | SQL | `backend/internal/repository/sql/deletion_fail.sql:2-11` | modified | failure method | stale token test |
| 24 | `deletionCompleteSQL` | SQL | `backend/internal/repository/sql/deletion_complete.sql:2-14` | modified | completion method | receipt and stale tests |
| 25 | `sessionRevokeUserSQL` | SQL | `backend/internal/repository/sql/session_revoke_user.sql:1-14` | modified | deletion execution | session tests |
| 26 | migration 26 upgrade | migration | `database/migrations/000026_custom_item_erasure_integration.up.sql:1-28` | modified | migration runner | upgrade integration and local stack |
| 27 | migration 26 rollback | migration | `database/migrations/000026_custom_item_erasure_integration.down.sql:1-23` | modified | migration runner | local stack |
| 28 | `TestTask240CustomItemErasureIntegration` | integration test | `backend/internal/app/task240_custom_item_erasure_integration_test.go:63-223` | modified | acceptance gate | live DB, Redis, API, receipt |
| 29 | `TestTask240MigrationUpgradeBackfillsDeletionLockout` | integration test | `backend/internal/app/task240_custom_item_erasure_integration_test.go:225-273` | added | migration gate | old requests and writes |
| 30 | `TestTask240ConcurrentCustomWriteSerializesBeforeDeletionMarker` | integration test | `backend/internal/app/task240_custom_item_erasure_integration_test.go:275-352` | added | lock gate | held transaction and API deletion |
| 31 | `TestTask240APIToProductionDeletionWorker` | integration test | `backend/internal/app/task240_custom_item_erasure_integration_test.go:354-402` | added | scheduler gate | API plus scheduler |
| 32 | `TestTask240ProcessingLeaseRecoversFailureRecordOutage` | integration test | `backend/internal/app/task240_custom_item_erasure_integration_test.go:404-455` | added | recovery gate | outage and lease boundary |
| 33 | `TestTask240ExpiredAttemptCannotFinalizeReclaimedWork` | integration test | `backend/internal/app/task240_custom_item_erasure_integration_test.go:457-522` | added | ownership gate | blocked purger and reclaim |
| 34 | direct `UserPurger` suite and Redis stub | tests/double | `backend/internal/cache/user_purger_test.go:1-202` | added | purger | pagination, isolation, errors, cancel, live 150 keys |
| 35 | direct scheduler suite and processor doubles | tests/double | `backend/internal/deletionworker/account_deletion_test.go:1-104` | added | scheduler | retry, defaults, cancellation, metrics |
| 36 | deletion service suite and doubles | tests/double | `backend/internal/userdata/deletion_test.go:1-230` | modified | execution and processor | lease, deadline, retry, cache, classification |
| 37 | `TestPostgresComplianceRepositoryDeletionHardening` | integration test | `backend/internal/repository/postgres_repository_test.go:2455-2585` | modified | claim/finalization | SKIP LOCKED, stale processing, token guards |
| 38 | compliance coverage/error tests | tests | `backend/internal/repository/account_repository_coverage_test.go:149-229` | modified | repository validation | lease and malformed input |
| 39 | deletion repository contract fake | test double | `backend/internal/repository/repository_test.go:233-251` | modified | interface checks | lease-aware signatures |

```yaml
inventory_source_count: 39
audited_symbol_count: 39
inventory_complete: true
generated_groupings:
  - "None; test suites are grouped only where the row names the complete adversarial suite and its doubles."
```

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| `main` deletion-worker composition | Real PostgreSQL, session, account, and Redis purger dependencies. | Startup failures are observable; signals cancel. | Errgroup shares cancellation; clients close. | No owner data in metrics/logs. | One scheduler beside optimization worker. | Minimal process composition. | Local stack and API-to-scheduler test. | PASS |
| `redisCachePurger.PurgeUser` | Delegates to one owner-prefix implementation. | Nil client and Redis errors are safe/observable. | Caller context reaches Redis. | UUID is server-derived. | No duplicate scan loop. | Narrow adapter. | App and live tests. | PASS |
| `userPurgeClient` | Only SCAN and DEL are exposed. | Command errors are returned. | Context is explicit. | No caller pattern. | Small test seam. | Minimal interface. | Stub and live tests. | PASS |
| `NewUserPurger` | Nonnil client retained; nil client no-op. | Both branches covered. | No resource acquisition. | No inputs beyond typed client. | Small value wrapper. | Idiomatic constructor. | 100% direct coverage. | PASS |
| `UserPurger.PurgeUser` | Deletes exact owner namespace in cursor batches. | Empty pages, scan/delete errors, cancellation handled. | Cursor advances after successful DEL; cancellation reaches both commands. | UUID prefix prevents cross-user selection. | SCAN count 100; no unbounded key slice. | One shared implementation. | Multi-page, isolation, errors, cancel, live 150-key tests. | PASS |
| scheduler defaults and `AccountDeletionProcessor` | Requires context and processor; defaults 30s and 10. | Nil and nonpositive inputs handled. | Context and UTC time passed per cycle. | Fixed metric contract. | One synchronous cycle. | Small boundary interface. | Validation/default tests. | PASS |
| `RunAccountDeletionProcessor` | Immediate then periodic processing until cancel. | Errors metric and retry; cancellation exits. | Ticker stops; no overlapping cycle goroutines. | Fixed labels only. | Bounded batch and one cycle at a time. | Clear loop. | Four tests and full race. | PASS |
| `recordAccountDeletionMetric` | Best-effort fixed metric emission. | Nil sink no-op; sink error ignored intentionally. | Caller context used. | No PII labels. | One synchronous call. | Small helper. | Metric assertions. | PASS |
| compliance repository constructor/default | Five-minute positive production lease. | Repository errors map consistently. | No mutable per-attempt owner state. | Parameterized SQL. | One executor. | Explicit default. | Integration and coverage. | PASS |
| `WithDeletionLeaseDuration` | Deterministic positive lease test seam. | Nonpositive preserves default; 500ms path covered. | Deadline becomes attempt token. | No user input. | Millisecond encoding. | Local internal API. | Boundary/config coverage. | PASS |
| `ClaimDeletionRequests` | Claims due work and returns new lease token. | Pending, retryable failed, stale processing, scan errors. | SKIP LOCKED prevents simultaneous claims; token changes on reclaim. | No dynamic SQL. | ORDER/LIMIT/index bound work. | Repository owns SQL. | Hardening, recovery, two-worker tests. | PASS |
| `RecordDeletionFailure` | Exact-token failure transition plus audit. | IDs, lease, category validation; stale is not-found. | Stale worker cannot overwrite reclaimed row; audit errors observable. | Categories allowlisted; note trimmed. | Two bounded statements. | Explicit token API. | Malformed, outage, retry, stale tests. | PASS |
| `CompleteDeletionRequest` | Exact-token pseudonymous completion. | Invalid IDs, stale, already completed paths. | Owner nulling and token guard are atomic; audit error visible. | No owner/custom receipt fields. | One update plus audit. | Clear final state. | Receipt and stale tests. | PASS |
| `DeletionRequestRepository` lease-token contract | Failure/completion require claimed deadline. | All fakes match signature. | Attempt ownership crosses processes explicitly. | No implicit user authority. | Timestamp only. | Minimal typed contract. | Contract fake and compile checks. | PASS |
| `ExecuteDeletion` | Revoke, cascade delete, purge, then receipt. | Missing/expired lease, DB, cache, cancellation errors. | Deadline equals claim; context reaches all dependencies; completion last and guarded. | Server-derived owner and receipt; no raw errors in receipt. | Sequential atomic DB statement and Redis scan. | Central orchestration. | 100% function coverage and adversarial tests. | PASS |
| `ProcessDueDeletionRequests` | Bounded claim/execute/classify loop. | Zero time, missing user, execution and record errors. | Carries exact token; lease recovers recording outage. | Sanitized failure categories only. | Sequential bounded batch. | Good separation of concerns. | 100% function coverage and live recovery. | PASS |
| `customFoodCreateSQL` | Null marker owner required for direct create. | Missing/marked owner yields zero rows. | NO KEY UPDATE conflicts with marker UPDATE and orders transactions. | Owner parameterized; global table untouched. | One owner lock plus insert. | Consistent CTE shape. | Live lock race and pending create. | PASS |
| `customFoodCreateClaimSQL` | Null marker owner required for idempotent create claim. | Missing/marked owner yields zero claim rows. | NO KEY UPDATE protects the claim boundary. | User ID and idempotency fields parameterized. | One owner lock plus conflict-safe insert. | Reuses direct-create lockout shape. | Pending idempotent-create test. | PASS |
| `customFoodUpdateSQL` | Null marker owner required for update. | Missing/marked owner yields zero updates. | NO KEY UPDATE orders update against marker. | Owner and item IDs parameterized. | One owner lock plus update. | Narrow SQL. | Pending update and live lock test. | PASS |
| `customFoodSoftDeleteSQL` | Null marker owner required for soft delete. | Missing/marked owner yields zero updates. | NO KEY UPDATE orders soft delete against marker. | Owner and item IDs parameterized. | One owner lock plus update. | Narrow SQL. | Pending delete and live lock test. | PASS |
| `deletionRequestSQL` | Marker and active request are one statement. | Missing user and active reuse handled. | UPDATE serializes with custom writes before API response. | Fixed audit text; parameterized user. | One CTE. | Clear lock boundary. | API and lock test. | PASS |
| `deletionClaimSQL` | Due rows become processing with future lease. | All status branches explicit. | Row lock and returned token prevent overlap/reclaim errors. | No untrusted identifiers. | Index and limit. | Readable embedded SQL. | Hardening/recovery/overlap. | PASS |
| `deletionFailSQL` | Processing plus exact token required. | Retry arithmetic and exhausted path deterministic. | Stale failure cannot change row. | Category validated before query. | Single update. | Direct guard. | Stale/retry tests. | PASS |
| `deletionCompleteSQL` | Clears owner and stores receipt metadata. | Nonprocessing/stale/completed rejected. | Exact token protects finalization. | Explicit user nulling. | Single update. | Minimal privacy boundary. | Receipt/stale tests. | PASS |
| `sessionRevokeUserSQL` | User-scoped idempotent session revoke. | Zero rows safe; DB error maps. | Context propagated. | User ID parameterized. | Single update. | Existing operation retained. | Session/deletion tests. | PASS |
| migration 26 upgrade | Marker, noncompleted backfill, FK removal, stale index. | Idempotent DDL and coalesce. | Lockout exists before writes on upgraded data. | Server-side set operation. | Set-based backfill and index. | Correct order. | Upgrade and local stack. | PASS |
| migration 26 rollback | Restores FK/index and removes marker/version. | Orphans nulled before FK. | Safe DDL order. | No external input. | Bounded migration. | Reversible. | Up/down/up. | PASS |
| `TestTask240CustomItemErasureIntegration` | End-to-end original acceptance. | Pending, retry, completion, access, privacy, survivor cases. | Real DB/Redis fixture cleanup. | Cross-user/global and receipt forbidden content asserted. | Bounded fixtures. | High-value integration. | All original criteria. | PASS |
| `TestTask240MigrationUpgradeBackfillsDeletionLockout` | Old requests receive marker. | Pending and processing branches. | Writes checked after upgrade. | Owner predicates remain. | Small fixture. | Exact time assertions. | Prior migration finding. | PASS |
| `TestTask240ConcurrentCustomWriteSerializesBeforeDeletionMarker` | Production lock compatibility. | Goroutine result, timeout probe, commit, post-marker path. | Held transaction and API request establish order; rollback cleanup. | Same owner only. | One row/request. | Direct race regression. | Prior blocking finding. | PASS |
| `TestTask240APIToProductionDeletionWorker` | API request reaches scheduler. | Worker completion and shutdown. | Shared worker context cancels cleanly. | Production owner/cache wiring. | 10ms test interval and bounded wait. | No direct processor bypass. | Production caller criterion. | PASS |
| `TestTask240ProcessingLeaseRecoversFailureRecordOutage` | Stranded processing is recoverable. | Execution and recorder outages. | No early claim; exact boundary reclaim. | Sanitized state. | One request/key. | Deterministic. | Prior recovery finding. | PASS |
| `TestTask240ExpiredAttemptCannotFinalizeReclaimedWork` | Expired worker cannot overwrite successor. | Block, deadline, reclaim, release, stale result. | Context deadline and token guard. | Same owner fixture. | 500ms test lease. | Strong concurrency regression. | Prior overlap finding. | PASS |
| direct `UserPurger` suite and stub | Directly exercises changed cache loop. | Nil, empty, pages, errors, cancellation, live behavior. | Stub controls blocking; live Redis validates command semantics. | Other namespace preserved. | 150 keys. | Narrow fake. | Changed symbols 100%. | PASS |
| direct scheduler suite and doubles | Poll, defaults, errors, metrics, shutdown. | Nil, failed cycles, retry/cancel. | No overlapping calls; ticker cleanup. | Fixed labels asserted. | Fast deterministic intervals. | Focused doubles. | Scheduler symbols 100%. | PASS |
| deletion service suite and doubles | Lease-aware service contract. | Validation, retry, cache, cancel, classification. | Fakes record context and token. | Categories checked. | In-memory. | Matches production interface. | Service symbols 100%. | PASS |
| `TestPostgresComplianceRepositoryDeletionHardening` | Real SQL state-machine regression. | Pending, failed, processing, stale, complete, errors. | SKIP LOCKED and exact guards. | Receipt owner nulling. | Bounded fixtures. | Strong repository test. | Repository methods 100%. | PASS |
| compliance coverage/error tests | Malformed leases/categories and SQL errors. | Query, scan, rows, audit branches. | Rows closed by methods. | Invalid categories rejected. | Fake executor. | Surgical extension. | Relevant branches covered. | PASS |
| deletion repository contract fake | Keeps interface implementations aligned. | Missing token signature would fail compile. | No runtime state. | No data crossing. | No I/O. | Minimal test double. | Contract checks. | PASS |

Mandatory audit conclusion: malformed inputs, errors, cleanup, cancellation, SQL parameterization, row locks, transactions, attempt ownership, retries, owner isolation, global preservation, access lockout, privacy receipt, production callers, and adversarial tests were inspected. All audited units pass.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| OPTIONAL | `backend/internal/repository/compliance_repository.go:110-112,246` | `WithDeletionLeaseDuration` | A positive sub-millisecond test duration is accepted but `Milliseconds()` serializes it as zero, creating an immediately reclaimable lease. | Production constructs the repository with five minutes; the only current configurator caller uses 500ms. | Nonblocking for this task. If exposed to production, reject below 1ms or pass microseconds and add a boundary test. |
| OPTIONAL | `backend/internal/deletionworker/account_deletion.go:78-85` | `recordAccountDeletionMetric` | Telemetry errors are ignored and emission is synchronous. | Existing sink is best-effort; tests cover nil/success/retry/cancel, not a blocking sink. | Accepted as existing best-effort telemetry contract; use a bounded sink only if prompt telemetry cancellation becomes required. |

```yaml
blocking_findings: 0
important_findings: 0
optional_findings: 2
```

No blocking or important finding remains. The two optional observations do not affect the current five-minute production lease, deletion correctness, privacy, or scheduler acceptance criteria.

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| focused task/deletion/repository test command | `backend` | 0 | PASS | Original and repaired integration, cache, service, repository, migration, scheduler-boundary, and concurrency tests. |
| focused command with `-race` | `backend` | 0 | PASS | Focused race run for repaired paths. |
| package coverage commands for cache, deletionworker, app, userdata, repository | `backend` | 0 | PASS | Changed functions 100%; package totals 99.3%, 100%, 85.7%, 97.2%, 93.1%; reports in `/tmp/task240-*.coverage.out`. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./...` | `backend` | 0 | PASS | Full backend suite. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./...` | `backend` | 0 | PASS | Full backend race suite. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | `backend` | 0 | PASS | Backend static analysis. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `backend` | 0 | PASS | No reachable vulnerabilities. |
| `python3 scripts/verify-local-stack.py` | repository root | 0 | PASS | Migration up/down/up, API, worker, health, and readiness. |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS | 263 task/dependency rows; task 240 remains PREPARED. |
| `python3 scripts/validate-traceability.py` | repository root | 0 | PASS | Requirement and design traceability. |
| `python3 scripts/check.py` | repository root | 0 | PASS | Aggregate exit 0; Go, security, migration, OpenAPI, frontend, UAT, unit, coverage, accessibility, and Playwright checks completed without failures. Existing documented warning/skips remain. |
| `git diff --check` | repository root | 0 | PASS | No whitespace errors. |
| `gofmt -l` on reviewed changed Go files | repository root | 0 | PASS | No output. |
| `sha256sum` on every file in Section 9 | repository root | 0 | PASS | Fresh hashes recorded below. |
| evidence validator command below | repository root | 0 | PASS | Review evidence structurally valid after refresh. |

No required command was omitted. The aggregate was run with visible output and a second exit capture reporting `CHECK_EXIT=0`.

## 9. Files Inspected and Staleness Fingerprints

Every current implementation, SQL, migration, direct dependency, and regression-test file below was inspected at relevant symbols or SQL units and hashed after the re-review.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| `backend/cmd/worker/main.go` | production scheduler composition | none | SHA256 | `07211c3c0a6bdeef0fbfb12e43fa34fa1aaeb7ee0ad08b1fbf8f103775e7c74d` |
| `backend/internal/app/app.go` | production cache adapter | none | SHA256 | `45de7f3b7de3515120e2a94c9eb3c40dfbbed95d8e29f64fa19f94f83e92d97b` |
| `backend/internal/app/task240_custom_item_erasure_integration_test.go` | acceptance and regressions | none | SHA256 | `7e6a0af9cbe2e17937e3cf6b85f5d874820f08f8a4401d132fac117c1c998948` |
| `backend/internal/cache/user_purger.go` | owner cache erasure | optional boundary note | SHA256 | `1ee0ccc70faf86a731d770cf549a3f9e55fff4aa00c5e2027f3929bf4a9e6e62` |
| `backend/internal/cache/user_purger_test.go` | purger tests | none | SHA256 | `6f07cf7eb09cb925e2759157d8b3302719fc03d26f73976821367b96e58f6d72` |
| `backend/internal/deletionworker/account_deletion.go` | deletion scheduler | optional telemetry note | SHA256 | `4474c633b2cc1c4724cbf3ef2df980a95429df3b68af6a8a7f5a48c598b78467` |
| `backend/internal/deletionworker/account_deletion_test.go` | scheduler tests | none | SHA256 | `f55c1be75603c96c40383c6ed3e4523b9dc8d97fa7c56cfe627f2a5982f4e661` |
| `backend/internal/httpapi/account_deletion_controller.go` | deletion route caller | none | SHA256 | `fdb9aa4d0291295009eb602d6f0bc66e90fa9ab0c958e5978359c57829a0a93f` |
| `backend/internal/httpapi/custom_item_controller.go` | custom access/write caller | none | SHA256 | `4ea8018aa044b3ab34ee54d8391e9dd4cd3a08dc911a8008888d9daec791d4d0a` |
| `backend/internal/httpapi/export_controller.go` | export caller | none | SHA256 | `4d5a956e352fd1dc031627ff18549463ca7487b26f170d5af1920f431820ecf6` |
| `backend/internal/httpapi/profile_controller.go` | profile/custom route caller | none | SHA256 | `38b8a2bebab80c3079dce54d57fe0157e55a8e448c42d7814b05d618150c4965` |
| `backend/internal/customitem/service.go` | custom-item owner service | none | SHA256 | `4bc9eb6ae297aec1b1030a23084143d27edf9f807e02f07073d4bce541b3975b` |
| `backend/internal/repository/compliance_repository.go` | deletion repository | none | SHA256 | `c864ec7ca992b3182f6778f71a2308a7934ea13dd6b76be01439be9830b506e3` |
| `backend/internal/repository/types.go` | deletion contract | none | SHA256 | `a959e11147b0ff709b885ae0c58870c3628ad545cfd99f8e5c53efb5f1116cde` |
| `backend/internal/repository/account_repository_coverage_test.go` | repository coverage | none | SHA256 | `3e61d163b844001034b05cca9678718e9db805057d707eb12c8c741bdd46fe9a` |
| `backend/internal/repository/postgres_repository_test.go` | PostgreSQL hardening | none | SHA256 | `1340709b50d180000495275c8c1209f87b5bb4f56d3b5824b80fdec8bd1d137a` |
| `backend/internal/repository/repository_test.go` | contract doubles | none | SHA256 | `9e4e8bdda6bbe3014393b687fede011ea5462642d14cc0396dc2b7b36e80ead3` |
| `backend/internal/repository/custom_food_repository.go` | custom SQL caller | none | SHA256 | `0b7035b27b6270afe532289c65f1a08ea7547302ee823e1a79c5ae9c0cb0dc5b` |
| `backend/internal/repository/custom_food_repository_test.go` | custom repository tests | none | SHA256 | `6db5e2daf6948a219468cf31b6aa4f93ed1a33cecb391052cfdd7a0c9fce6bfb` |
| `backend/internal/repository/encrypted_identity_repository.go` | account cascade delete | none | SHA256 | `cb9afe281b4fcd52ebeb5ebd774f58171c3760928acda5e3451f97552ea00761` |
| `backend/internal/repository/session_repository.go` | session revoke | none | SHA256 | `c5d6214dff0bda321fcfdb4b355f1ad4fec5d072f1cff73b7446b84de77f369a` |
| `backend/internal/repository/sql/custom_food_create.sql` | direct create lockout | none | SHA256 | `303e7e223ea997dcdad08d8129cd98981599b814e850cbc2a0caa849766f64e0` |
| `backend/internal/repository/sql/custom_food_create_claim.sql` | idempotent create lockout | none | SHA256 | `424b3a4eb0f54eb3fcd2af919ba254cfe4618e88e9d526327f6db7f9f0186a8f` |
| `backend/internal/repository/sql/custom_food_update.sql` | update lockout | none | SHA256 | `4e862b2fffd3b198e47a8d342e6612f7c68fc3d436d25bfc5b365f4cd28bed42` |
| `backend/internal/repository/sql/custom_food_soft_delete.sql` | soft-delete lockout | none | SHA256 | `9fd2a13a0701a2822d875bb334f70ef3d3e15e88fe126e7c559ef2281013916e` |
| `backend/internal/repository/sql/custom_food_get_by_id.sql` | owner read | none | SHA256 | `04ddfa9e5572e74cd8f0b89967460fb6207af04c7e0714aaf45471389f724c6c` |
| `backend/internal/repository/sql/custom_food_list.sql` | owner export read | none | SHA256 | `bbea2a62edcf65ffce98b58455e7f9ee0dcf5cc20e7da2fc8c01639204bc1f33` |
| `backend/internal/repository/sql/custom_food_clear_classifications.sql` | classification cleanup | none | SHA256 | `e593ef412c7aeb073af1448a31186fe6886ea529cc38109185eb8fcb417eb6f4` |
| `backend/internal/repository/sql/custom_food_attach_classification.sql` | classification assignment | none | SHA256 | `8870aa780d2b53fd5f01045460a86bac1487a3e2db8c0a597503ed2991e15a69` |
| `backend/internal/repository/sql/custom_food_list_classifications.sql` | classification read | none | SHA256 | `ebc16d07c6e51c181468935532a776407bdde334b781f83c503fd5fdac804617` |
| `backend/internal/repository/sql/deletion_claim.sql` | claim and stale lease | none | SHA256 | `84aa4f1083f989e160b82f26b5eb87f4e0121545a9f5c74599f3430b38bbdd27` |
| `backend/internal/repository/sql/deletion_fail.sql` | guarded failure | none | SHA256 | `c40e27694a16ffac97ae3ba1c9ffea9f51a3bef8dbe613622dafd66be02d3fa2` |
| `backend/internal/repository/sql/deletion_complete.sql` | guarded completion | none | SHA256 | `4b79c009fe562bdf08db513710cf97e08ad000f0d34be220c2c24e65b49a1d34` |
| `backend/internal/repository/sql/deletion_request.sql` | durable marker | none | SHA256 | `f32989a58246943dda9fb74159e49d7f436c3a7a101662b008a65d2558bdf842` |
| `backend/internal/repository/sql/session_revoke_user.sql` | session revoke | none | SHA256 | `6a11f789b8037bff4149b6d0c09bb8e65e553b25d493c8c9c52844ef765d7960` |
| `backend/internal/repository/sql/user_delete_account.sql` | atomic user delete | none | SHA256 | `5578f4e92a93c5ff746480c3d107b5e92f9df3fb343df770b5dc879e3c1ad52a` |
| `backend/internal/userdata/deletion.go` | deletion orchestration | none | SHA256 | `3959b9bec47cd5f461e7ac47b79be5a68faa5f0e3339bbb5a831520f56db9f47` |
| `backend/internal/userdata/deletion_test.go` | service tests | none | SHA256 | `7dba8003ca03a87836ef86f47d9a1a18be21c4aced25cef798a7279071054bb0` |
| `backend/internal/userdata/export.go` | export projection | none | SHA256 | `3035d0f177abb0417b19c4cd266b466b804b20848153ba1d092e4b9caeb855af` |
| `database/migrations/000025_user_owned_custom_food_items.up.sql` | owner cascade schema | none | SHA256 | `dc3e479dd9ba72d39ceeb0a93c3d99b6097c4a862e5800f93ed3612d4f5c5091` |
| `database/migrations/000025_user_owned_custom_food_items.down.sql` | owner schema rollback | none | SHA256 | `8e9ba4712f5ee253bd2425557433cbbcebced083780dbce411cb7c5c3789091f` |
| `database/migrations/000026_custom_item_erasure_integration.up.sql` | marker/backfill/index | none | SHA256 | `219fffeefbe247c298b48c88fc1d880782744da722e828ace2d9b0989d415f91` |
| `database/migrations/000026_custom_item_erasure_integration.down.sql` | rollback | none | SHA256 | `ca178712ba75ef96b9912f4b0aaf0c0820e39f407642bebd76da48f0177e1eca` |

```yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "docs/implementation/reviews/task-240-review.md pre-repair findings and hashes; replaced after current repair audit"
  - "preparation-report hash ledger captured repair-start contents; superseded by fresh Section 9 hashes"
```

## 10. Coverage and Exceptions

- [x] Required coverage commands ran.
- [x] Report paths and observed thresholds are recorded.
- [x] Untested branches relevant to changed symbols were inspected.
- [x] Exceptions exactly match the task row; no task-240 exception is claimed.

```yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "/tmp/task240-cache.coverage.out, /tmp/task240-worker.coverage.out, /tmp/task240-app.coverage.out, /tmp/task240-userdata.coverage.out, /tmp/task240-repository2.coverage.out"
observed_line_coverage: "Changed task-240 functions 100%; package totals cache 99.3%, deletionworker 100%, app 85.7%, userdata 97.2%, repository 93.1%"
coverage_passed: true
```

Coverage finding: no task-specific coverage gap remains. Existing package-level deficits are outside changed symbols and the aggregate's historical Phase 07 exception is documented separately.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced.
- [x] No source-of-truth documentation was contradicted.
- [x] No generated/cache/build/temporary artifact was unintentionally added by this re-review.
- [x] Public API additions are necessary and used; the lease configurator is internal and used by deterministic tests.
- [x] Duplicate helpers and obsolete aliases were searched for; one shared Redis purger is used.
- [x] Error, cleanup, timeout, concurrency, and malformed-input paths were challenged.

Findings: the prior `FOR KEY SHARE` race is repaired with `FOR NO KEY UPDATE`; direct purger coverage is repaired; lease overlap and stranded-processing concerns are repaired with bounded contexts, exact attempt tokens, stale reclaim, and guarded finalization. The current task-list status was not changed.

## 12. Decision

A task may be `PASSED` only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking or important finding remains.

Before accepting the decision, the required validator was run:

```bash
python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-240-review.md
```

```yaml
decision: "PASSED"
reason: "All original criteria and repaired lock, migration, scheduler, lease, cache, cancellation, isolation, access-lockout, and privacy gates pass with current hashes and no blocking or important finding."
failed_criteria:
  - ""
failed_or_unaudited_symbols:
  - ""
recommended_next_action: "None; leave task-list status unchanged for the phase orchestrator."
```

## 13. Repair Context

N/A — this re-review is PASSED. No further repair is required for task 240. The two optional observations are recorded for future hardening only and do not block acceptance.
