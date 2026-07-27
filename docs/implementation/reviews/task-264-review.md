# Review Evidence: Task 264 — Manual Item Search Cache Invalidation

```yaml
task_id: 264
component: "ItemCurator"
static_aspect: "Manual Item Search Cache Invalidation"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-24T20:43:27Z"
review_agent: "Codex"
evidence_file: "docs/implementation/reviews/task-264-review.md"
baseline_ref: "e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "code-review-skill/reference/go.md and security-review-guide.md"
repair_context_required: true
```

## 1. Task Source

**Description:** Wire manual global-item create, update, and soft-delete to the existing shared Redis food-data generation only after the audited transaction commits; skip create replays and validation, conflict, transaction, or audit failures.

**Depends On:** `250`, `251`; both dependency rows are `PASSED`.

**Testing Coverage Exceptions:** `None` in task row. The repository-wide coverage contract permits exact measured defensive exceptions documented in `docs/implementation/04_OPEN.md`; no task-specific behavior exception was accepted.

**Verification Criteria:** Focused controller and app-composition tests prove each committed create, update, and delete advances the shared generation exactly once; create replay and failed or rolled-back mutations advance it zero times; invalidation is registered through `AfterCommit`. Redis integration prewarms Catalog and Substitution Search caches, mutates the item, proves fresh peer-instance results without stale in-flight repopulation, and focused tests pass under `go test -race`.

The literal repository path `templates/review_checklist.md` was absent. The canonical available template at `/home/wiktor/.agents/skills/phase-orchestrator/templates/review_checklist.md` was read completely (225 lines). The code-review skill was invoked exactly once; its complete Go and security guides were read and applied.

## 2. Pre-Review Gates

- [x] Input status is `PREPARED`. The concurrent phase workflow transitioned authoritative row 264 from `OPEN` to `PREPARED`; this review did not edit it.
- [x] Dependencies `250` and `251` are `PASSED`.
- [x] The preparation report claims completion and identifies the task-owned paths and symbols.
- [x] The fixed baseline is available and the task-owned diff can be reconstructed.
- [x] `code-review-skill` was invoked exactly once and relevant Go/security guidance was read.
- [x] The reviewer is independent from the preparation/implementation work in this turn.
- [x] Review evidence uses current source, tests, and hashes rather than stale preparation logs.
- [x] No production code, test implementation, task-list status, or concurrent change was edited by this review.

```yaml
pre_review_gates_passed: true
blocking_issue: null
```

## 3. Review Baseline and Change Surface

Baseline/reference method: `HEAD` equals the fixed baseline commit. The worktree is intentionally dirty with concurrent Phase 08 changes. Ownership was reconstructed from the preparation’s six-path claim, the baseline diff, the untracked-file diff, symbol-level edits, and caller searches. The only task-owned executable changes are the five implementation/test paths below; the preparation file is evidence only.

Commands used to reconstruct the diff:

```bash
git status --short --branch
git show -s --format='%H %P %s' e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f
git diff --stat e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f -- <task-owned tracked paths>
git diff --unified=80 e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f -- <task-owned tracked paths>
git diff --no-index /dev/null backend/internal/cache/manual_item_generation_integration_test.go
rg -n 'NewManualItemAdminController|ManualItemCacheInvalidator|AfterCommit|NewClassificationInvalidator\(' backend --glob '*.go'
```

Pre-existing dirty-worktree changes and exclusions: `api/openapi.yaml`, external-data/admin/frontend/generated-type/generator paths, `backend/internal/itemcurator/service.go` documentation changes from Task 269, `backend/internal/httpapi/custom_item_controller.go` changes from Task 265, later task preparation/review files, and other concurrent work were preserved and excluded. `app.go` contributes one task-owned constructor argument; unrelated callers and shared dependencies were inspected but not attributed. No SQL, migration, OpenAPI, generated client, frontend, task-list, or dependency implementation path was changed for Task 264.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| `backend/internal/httpapi/manual_item_controller.go` | Baseline tracked diff | HIGH | `ManualItemCacheInvalidator`, `ManualItemController`, constructor, `Create`, `Update`, `Delete`, `invalidate` |
| `backend/internal/httpapi/manual_item_controller_test.go` | Baseline tracked diff | HIGH | test service/doubles and three focused tests |
| `backend/internal/app/app.go` | One baseline tracked constructor call change | HIGH | `newProduction` |
| `backend/internal/app/app_test.go` | Baseline tracked diff | HIGH | AST composition test and `selectorName` |
| `backend/internal/cache/manual_item_generation_integration_test.go` | New untracked task-owned integration test | HIGH | repository double, search runner, live Redis test, assertion helper |
| `docs/implementation/preparations/task-264.md` | New preparation evidence | HIGH | Documentation only; no executable unit |

The task-owned change is distinguishable. No implementation-path ambiguity was found.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | A committed create advances the shared generation exactly once. | Focused HTTP test and controller callback inspection. | PASS | `TestManualItemAdminHTTPValidCRUDReplayAndAuditSnapshots` observes count `1`; `Create` installs one callback. |
| 2 | A committed update advances the shared generation exactly once. | Focused HTTP test and callback inspection. | PASS | The same test observes count `2`; `Update` returns `AfterCommit: c.invalidate`. |
| 3 | A committed soft-delete advances the shared generation exactly once. | Focused HTTP test and callback inspection. | PASS | The same test observes count `3`; `Delete` returns `AfterCommit: c.invalidate`. |
| 4 | An idempotent create replay advances zero times. | Replay HTTP test and repository audit contract inspection. | PASS | Replay leaves count at `1`; `Create` gates the callback on `!result.Replayed`; `WithMutationAudit` skips replay audit persistence. |
| 5 | Validation failures advance zero times. | Invalid-body HTTP cases and pre-dispatch validation inspection. | PASS | Seven invalid create bodies return `400` and leave the invalidator at zero; validators run before mutation dispatch. |
| 6 | Conflict failures advance zero times. | Create and update conflict HTTP cases. | PASS | Idempotency, duplicate-name, and update conflict cases leave the count at zero; service errors return before producing `AfterCommit`. |
| 7 | Transaction and audit rollback failures advance zero times. | Audit observer plus transaction implementation inspection. | PASS | Audit and simulated transaction failures return errors with zero invalidations; `withTransaction` commits before returning and the gateway invokes the callback only after successful return. |
| 8 | Invalidation is registered through `AfterCommit`, not during the transaction. | `AdminController.transactionalMutation`, `WithMutationAudit`, and ordering test. | PASS | Gateway invokes `result.AfterCommit()` only after `WithMutationAudit` succeeds; `manualItemAuditObserver` rejects any pre-return invalidation. |
| 9 | Production composition uses the existing shared Redis generation invalidator. | App AST test and `newProduction` inspection. | PASS | `NewManualItemAdminController(..., cache.NewClassificationInvalidator(nil, redisClient))`; the test asserts the constructor shape. |
| 10 | Redis-backed Catalog Search cache refreshes on create across peer instances. | Live Redis integration with two Catalog services. | PASS | Test prewarms `firstCatalog`, advances generation, and observes the created item through `secondCatalog`. |
| 11 | Redis-backed Catalog Search cache refreshes on update and delete. | Live Redis integration. | PASS | The same test observes renamed `tempeh` through the first instance and empty results after deletion through the second. |
| 12 | Redis-backed Substitution Search response and similarity caches refresh across peer instances. | Live Redis integration with two Substitution services. | PASS | The test prewarms both response and similarity paths and observes fresh create/update/delete state through the peer service. |
| 13 | Stale in-flight Catalog response writes are rejected. | Live Redis guarded-write assertions. | PASS | `SetSearchResponse` with the pre-invalidation token returns `stored=false`. |
| 14 | Stale in-flight similarity writes are rejected. | Live Redis guarded-write assertions. | PASS | `SetSimilarityCalculation` with the pre-invalidation token returns `stored=false`. |
| 15 | Focused tests pass under race detection. | Focused and complete affected-package race commands. | PASS | HTTP, app, cache focused race tests and `go test -race ./internal/httpapi ./internal/app ./internal/cache` all pass. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | `ManualItemCacheInvalidator` | interface | `manual_item_controller.go:24` | Added | `ManualItemController` and production invalidator | Constructor and HTTP tests |
| 2 | `ManualItemController` | behavioral type | `manual_item_controller.go:30` | Modified | Manual admin route handlers | HTTP tests |
| 3 | `NewManualItemAdminController` | constructor | `manual_item_controller.go:37` | Modified | `app.newProduction`, test constructors | App and HTTP tests |
| 4 | `(*ManualItemController).Create` | mutation handler | `manual_item_controller.go:54` | Modified | POST `/admin/items` through `AdminController.transactionalMutation` | Valid/replay/failure HTTP tests |
| 5 | `(*ManualItemController).Update` | mutation handler | `manual_item_controller.go:107` | Modified | PUT `/admin/items/:itemId` through gateway | Valid/conflict/failure HTTP tests |
| 6 | `(*ManualItemController).Delete` | mutation handler | `manual_item_controller.go:130` | Modified | DELETE `/admin/items/:itemId` through gateway | Valid/failure HTTP tests |
| 7 | `(*ManualItemController).invalidate` | callback helper | `manual_item_controller.go:149` | Added | Update/delete `AfterCommit` callbacks | Exactly-once and ordering tests |
| 8 | `fakeManualItemService` | test double | `manual_item_controller_test.go:21` | Modified | Focused HTTP tests | Three focused tests |
| 9 | `(*fakeManualItemService).Update` | test-double method | `manual_item_controller_test.go:44` | Modified | Update success/conflict tests | Focused HTTP tests |
| 10 | `(*fakeManualItemService).Delete` | test-double method | `manual_item_controller_test.go:54` | Modified | Delete success test | Focused HTTP tests |
| 11 | `manualItemInvalidatorStub` | test double | `manual_item_controller_test.go:62` | Added | Controller callback assertions | Focused HTTP tests |
| 12 | `(*manualItemInvalidatorStub).Invalidate` | test-double method | `manual_item_controller_test.go:64` | Added | Controller callbacks | Atomic count assertions |
| 13 | `manualItemAuditObserver` | test double | `manual_item_controller_test.go:66` | Added | Ordering/rollback test | Post-commit test |
| 14 | `(*manualItemAuditObserver).WithMutationAudit` | test-double method | `manual_item_controller_test.go:71` | Added | Shared gateway callback | Post-commit test |
| 15 | `TestManualItemAdminHTTPValidCRUDReplayAndAuditSnapshots` | test | `manual_item_controller_test.go:90` | Modified | Test runner | Exactly-once/replay assertions |
| 16 | `TestManualItemAdminHTTPRejectsConflictsDuplicatesInvalidFieldsAndOwnership` | test | `manual_item_controller_test.go:130` | Modified | Test runner | Failed-request zero assertions |
| 17 | `TestManualItemInvalidationRunsOnlyAfterSuccessfulAuditCommit` | test | `manual_item_controller_test.go:181` | Added | Test runner | Ordering/audit/transaction/conflict assertions |
| 18 | `newProduction` | application composition | `backend/internal/app/app.go:53,191` | Modified | `NewProduction` startup | App composition test and full app tests |
| 19 | `TestNewProductionComposesManualItemSharedGenerationInvalidation` | test | `backend/internal/app/app_test.go:127` | Added | Test runner | AST composition assertion |
| 20 | `selectorName` | test helper | `backend/internal/app/app_test.go:150` | Added | AST composition test | Direct consumer |
| 21 | `manualItemSearchRepository` | integration test double | `manual_item_generation_integration_test.go:18` | Added | Catalog/Substitution services | Live Redis integration |
| 22 | `(*manualItemSearchRepository).GetByID` | test-double method | `manual_item_generation_integration_test.go:23` | Added | Substitution source loading | Live Redis integration |
| 23 | `(*manualItemSearchRepository).Search` | test-double method | `manual_item_generation_integration_test.go:34` | Added | Catalog and candidate loading | Live Redis integration |
| 24 | `(*manualItemSearchRepository).replace` | fixture mutator | `manual_item_generation_integration_test.go:46` | Added | Create/update/delete fixture stages | Live Redis integration |
| 25 | `TestManualItemGenerationLiveRedisRefreshesPeerCatalogAndSubstitutionSearch` | integration test | `manual_item_generation_integration_test.go:54` | Added | Test runner | Live Redis refresh and stale-write assertions |
| 26 | `searchRunner` | test interface | `manual_item_generation_integration_test.go:145` | Added | Search assertion helper | Live Redis integration |
| 27 | `assertSearchItemName` | test helper | `manual_item_generation_integration_test.go:149` | Added | Live Redis integration test | Direct consumer |

```yaml
inventory_source_count: 27
audited_symbol_count: 27
inventory_complete: true
generated_groupings:
  - "None; the integration test and test doubles are hand-written and individually inventoried."
```

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| `ManualItemCacheInvalidator` | Minimal post-commit `Invalidate()` contract; no mutation data crosses the seam. | N/A — interface has no branches; nil is handled by caller. | No state or goroutine ownership. | No user input or sensitive data. | No allocation or I/O contract. | Small consumer-defined interface, idiomatic Go. | Used by constructor and stub; no implementation gap. | PASS |
| `ManualItemController` | Retains service and optional invalidator for four manual routes. | Nil service remains fail-closed in existing handlers. | Fields are construction-time immutable; no shared mutable state added. | Existing admin gateway, CSRF, validation, and rate limits remain in front. | No new query or loop. | Narrow composition, with compatibility-preserving optional argument. | Constructor and route tests pass; variadic extra-argument misuse is not product-reachable. | PASS |
| `NewManualItemAdminController` | Wires one optional invalidator into route-bound controller. | Zero or nil optional invalidator is safe; first supplied invalidator is used. | No resources or cancellation. | Does not bypass the admin gateway. | O(1) construction. | Direct route composition; optional seam preserves existing test callers. | App AST and HTTP composition tests pass. | PASS |
| `(*ManualItemController).Create` | Non-replay committed create gets exactly one post-commit invalidation; replay gets none. | Auth, dependency, decode, service, and mapped error returns produce no callback. | Callback is deferred by gateway until audit transaction success; no goroutine. | Admin identity and idempotency key remain gateway/service-owned; no cache payload includes input. | One closure; invalidation is the existing bounded best-effort Redis operation. | Callback captures immutable result and uses explicit replay guard. | Valid/replay, invalid, conflict, audit, and transaction tests pass; no direct injected nil-controller call is needed. | PASS |
| `(*ManualItemController).Update` | Successful authoritative update returns one `AfterCommit` callback. | Parse, dependency, decode, and service errors return before callback creation. | Gateway owns commit ordering; invalidator has its own bounded timeout. | UUID and body validation remain unchanged; no authorization scope is weakened. | No additional database work. | Reuses one private helper rather than duplicate callback code. | Success/conflict/rollback tests pass; direct update-validation failure is covered by shared middleware inspection rather than a dedicated count assertion. | PASS |
| `(*ManualItemController).Delete` | Successful soft-delete returns one post-commit callback and 204 response. | Parse, dependency, and service errors return before callback creation. | No goroutine; callback runs only after audit/transaction return. | Existing admin route and UUID boundary remain intact. | No additional database work. | Same callback shape as update. | Success/rollback tests pass; a direct delete-service-error count test is an optional coverage gap. | PASS |
| `(*ManualItemController).invalidate` | Calls injected invalidator exactly once when non-nil. | Nil invalidator is an intentional no-op. | No new state; underlying invalidator owns its 2-second context. | No trusted-boundary crossing beyond the existing cache abstraction. | Synchronous existing invalidation behavior may add bounded post-commit latency. | Tiny nil-safe helper. | Atomic stub observes exactly-once behavior. | PASS |
| `fakeManualItemService` | Supplies deterministic service outcomes without persistence coupling. | Added update/delete errors model domain failure paths. | Test-only state is request-serial and not shared across goroutines. | Test fixture does not alter auth boundary. | No production I/O. | Small focused double. | Used by all changed HTTP tests. | PASS |
| `(*fakeManualItemService).Update` | Returns success or injected error for callback tests. | Conflict error exits before `AfterCommit`; success supplies before/after. | No concurrency in test fixture. | No security surface. | No I/O. | Behavior is minimal and deterministic. | Success and conflict paths pass; validation is gateway-owned. | PASS |
| `(*fakeManualItemService).Delete` | Returns before state or injected error for delete callback tests. | Success and injected error are explicit; only success reaches callback. | No concurrency in fixture. | No security surface. | No I/O. | Minimal test seam. | Success tested; injected delete error is defined but not directly exercised, optional gap. | PASS |
| `manualItemInvalidatorStub` | Counts callback calls atomically. | Zero and positive counts are observable. | `atomic.Int32` is race-safe. | No security data. | No I/O. | Appropriate test double. | Focused race tests pass. | PASS |
| `(*manualItemInvalidatorStub).Invalidate` | One invocation increments one count. | No error path by contract. | Atomic increment avoids test races. | N/A — test-only. | O(1), no I/O. | Idiomatic. | Exactly-once and zero assertions pass. | PASS |
| `manualItemAuditObserver` | Models the audit boundary and rejects pre-commit invalidation. | Mutation errors and configured audit/transaction errors return without callback. | Checks ordering while callback is still inside audit function. | No production trust boundary. | No I/O. | Focused test-only observer. | Post-commit and rollback test pass; real commit ordering is also inspected in repository code. | PASS |
| `(*manualItemAuditObserver).WithMutationAudit` | Returns only after mutation and simulated audit outcome. | Handles mutation error, early invalidation, configured failure, and replay. | No goroutine; ordering assertion is deterministic. | Test-only. | O(1). | Implements the narrow repository interface. | Covers callback timing; real `withTransaction` path is separately inspected and package-tested. | PASS |
| `TestManualItemAdminHTTPValidCRUDReplayAndAuditSnapshots` | Proves create/update/delete counts 1/2/3 and replay count remains 1. | Exercises normal CRUD and replay. | Request sequence is serial; atomic count is safe. | Uses admin JWT, CSRF, route validation, and audit gateway. | Local Fiber test only. | Assertions remain focused on behavior plus existing audit snapshots. | Direct acceptance evidence passes. | PASS |
| `TestManualItemAdminHTTPRejectsConflictsDuplicatesInvalidFieldsAndOwnership` | Failed request paths must not invalidate. | Covers duplicate JSON keys, duplicate IDs, invalid macros/state/micros, owner/image input, idempotency and duplicate conflicts. | Serial test; atomic counter. | Verifies private route is not exposed and safe status mapping. | Local Fiber test only. | Reuses existing valid fixture. | Direct failure evidence passes; no separate update-validation case. | PASS |
| `TestManualItemInvalidationRunsOnlyAfterSuccessfulAuditCommit` | Callback is post-commit and absent on audit/transaction/conflict failure. | Covers committed create, audit failure, transaction failure, update conflict. | Observer checks ordering; atomic count avoids race. | Failure responses remain safe. | Local Fiber test only. | Small targeted regression test. | Direct ordering evidence passes; delete-service error remains optional gap. | PASS |
| `newProduction` | Manual admin routes share the Redis generation used by Catalog/Substitution caches. | Nil Redis remains safe through existing no-op invalidator behavior. | Production wiring owns client lifetime; invalidation timeout is inside existing cache implementation. | No new route or auth bypass. | One value construction; no extra dependency. | Reuses existing `ClassificationInvalidator`, no new cache primitive. | AST test plus full app tests; live Redis behavior is tested in cache package. | PASS |
| `TestNewProductionComposesManualItemSharedGenerationInvalidation` | Fails if the manual controller is not passed the Redis-backed invalidator. | Parser failure and missing composition fail test. | AST inspection is deterministic and resource-free. | Verifies composition without opening production dependencies. | O(size of app AST), negligible. | Appropriate for concrete production constructor. | Direct static evidence passes; it does not replace a live production HTTP mutation test. | PASS |
| `selectorName` | Extracts selector identifier for the AST assertion. | Non-selector returns empty string. | No state/resources. | N/A — test helper. | O(1). | Small helper. | Consumed by composition test. | PASS |
| `manualItemSearchRepository` | Provides shared mutable food source for two search-service instances. | Search and lookup return not-found or matching entities. | `sync.RWMutex` protects fixture reads/replacements. | No external input beyond test query. | Bounded fixture slices; no external I/O. | Narrow test repository. | Live Redis test exercises create/update/delete states. | PASS |
| `(*manualItemSearchRepository).GetByID` | Returns matching source item or typed not-found. | Handles absent IDs. | Read lock is released with defer. | Test-only. | O(n) bounded fixture scan. | Simple repository contract implementation. | Used by substitution source loading. | PASS |
| `(*manualItemSearchRepository).Search` | Case-insensitive name filtering returns items and count. | Empty query returns all; no match returns empty. | Read lock covers snapshot iteration. | Test-only. | O(n) bounded fixture scan; no unbounded production path. | Deterministic enough for fixture assertions. | Catalog and substitution cache paths pass; pagination breadth is not the subject. | PASS |
| `(*manualItemSearchRepository).replace` | Atomically replaces fixture source state. | Empty replacement models delete. | Write lock and cloned slice prevent aliasing. | Test-only. | O(n) copy bounded by fixture. | Clear mutation fixture helper. | Drives create/update/delete stages. | PASS |
| `TestManualItemGenerationLiveRedisRefreshesPeerCatalogAndSubstitutionSearch` | Shared Redis generation makes old cache generations unreachable and rejects stale writes. | Redis unavailable skips; create, update, delete, response miss, similarity miss, and stale writes are exercised. | 10-second search context and 2-second cleanup context are cancelled; no goroutine. | Uses random query/IDs and isolated Redis DB convention; no secrets or user data. | Real Redis, bounded keys and cleanup scans; invalidator has 2-second bound. | Directly exercises existing cache contract without production DB coupling. | Live test passes under normal and race runs; direct production app transaction integration is deferred to Task 271. | PASS |
| `searchRunner` | Narrows search services to the method under assertion. | N/A — interface only. | No state. | No security surface. | No I/O contract. | Consumer-defined minimal interface. | Used by helper. | PASS |
| `assertSearchItemName` | Asserts empty or one expected search result. | Handles empty expected state and mismatched result count/name. | No state/resources. | Test-only. | O(result count). | Focused helper. | Covers create/update/delete peer observations. | PASS |

Mandatory cross-cutting answers: malformed inputs remain rejected before changed callbacks; every callback/error path is intentional; resources and cancellation are owned by the existing invalidator; no new goroutine or shared mutable production state exists; Redis generation and guarded writes are concurrency-safe; user-controlled data does not enter cache keys or SQL through this change; loops are existing bounded invalidation scans; the interface is minimal and used; adversarial replay, failure, stale-write, and race paths are tested.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| optional | `backend/internal/httpapi/manual_item_controller_test.go:54-59,167-171` | Delete/update failure coverage | The focused suite does not directly assert zero invalidations for a delete service error or an update validation failure, although code inspection proves both return before callback creation. | `deleteErr` is defined but not exercised; invalid-body cases are create-only. | Add table-driven per-mutation failure assertions if stronger direct coverage is desired; no production repair is indicated. |

```yaml
blocking_findings: 0
important_findings: 0
optional_findings: 1
```

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| `gofmt -d backend/internal/httpapi/manual_item_controller.go backend/internal/httpapi/manual_item_controller_test.go backend/internal/app/app.go backend/internal/app/app_test.go backend/internal/cache/manual_item_generation_integration_test.go` | repository root | 0 | PASS | No formatting diff. |
| `git diff --check` | repository root | 0 | PASS | No whitespace errors. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -run 'TestManualItem(...)$' -count=1` | `backend/` | 0 | PASS | Focused HTTP tests. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/app -run '^TestNewProductionComposesManualItemSharedGenerationInvalidation$' -count=1` | `backend/` | 0 | PASS | App composition test. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/cache -run '^TestManualItemGenerationLiveRedisRefreshesPeerCatalogAndSubstitutionSearch$' -count=1` | `backend/` | 0 | PASS | Live Redis test ran, not skipped. |
| Focused HTTP/app/cache `go test -race` commands | `backend/` | 0 | PASS | All focused race lanes pass. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi ./internal/app ./internal/cache -count=1` | `backend/` | 0 | PASS | All affected packages; app 104.502s while concurrent repository tests were running. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/httpapi ./internal/app ./internal/cache -count=1` | `backend/` | 0 | PASS | All affected packages under race detector. |
| `GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./internal/httpapi ./internal/app ./internal/cache` | `backend/` | 0 | PASS | No diagnostics. |
| Full-package coverage commands writing `/tmp/task-264-httpapi-full.cover`, `/tmp/task-264-app-full.cover`, `/tmp/task-264-cache-full.cover` | `backend/` | 0 | PASS | HTTP 87.4%, app 84.8%, cache 90.1%; changed `newProduction` 92.9%, `NewClassificationInvalidator` 100%, `Invalidate` 85.7%. |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS | 275 ordered tasks; row 264 is `PREPARED` at final review. |
| `python3 scripts/validate-traceability.py` | repository root | 0 | PASS | Traceability valid. |
| `python3 scripts/check.py --quick` | repository root | 0 | PASS | 533 frontend unit tests, 30 browser tests, changed backend packages, vet, vulnerability, OpenAPI/type drift, and coverage-contract lanes pass. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-264-review.md` | repository root | 0 | PASS | Run after writing this evidence file. |

The full aggregate `scripts/check.py` release gate was not run; the task-specific `--quick` gate covered all changed-area and static lanes, while the worktree contains unrelated concurrent Phase 08 changes.

## 9. Files Inspected and Staleness Fingerprints

The following hashes are SHA-256 of current contents captured after review inspection. The review evidence file is intentionally omitted from its own manifest.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| `backend/internal/httpapi/manual_item_controller.go` | Task implementation boundary | PASS; post-commit callbacks | SHA-256 | `48659ad69db18fbb3f830a4e0f833786dcd0c4c8c51f777211dbfe8001324382` |
| `backend/internal/httpapi/manual_item_controller_test.go` | HTTP regression evidence | PASS; optional failure-case gap recorded | SHA-256 | `2f953697baabed40e6d7539f0a99499a5a6dac3810123699d2745a5a79fbf36e` |
| `backend/internal/app/app.go` | Production composition | PASS | SHA-256 | `3370985462e78ec3d2a037494ffed0ed101cdf056290c62aaa41a9eaf6dccd29` |
| `backend/internal/app/app_test.go` | Composition evidence | PASS; AST boundary noted | SHA-256 | `1a2a336ab8c95fb54ede6d313ea860a3ec7c9732715b08f2e4781a35ea93d9f1` |
| `backend/internal/cache/manual_item_generation_integration_test.go` | Live Redis peer/stale-write evidence | PASS | SHA-256 | `fc8add31d60b67b3e90b85bc973619f450109fd00147c29f1f4eff24ea408a31` |
| `docs/implementation/preparations/task-264.md` | Preparation boundary and claimed evidence | Current preparation record | SHA-256 | `689c8b34c453c06b9a6b3d74b297a1970106b0f4aba565536a25005684f3ff97` |
| `backend/internal/httpapi/admin_controller.go` | `AfterCommit` caller and gateway ordering | PASS | SHA-256 | `763e21a7f4df99afafd82870d0470c9f6d080070c27ef657f76fe8674f883b82` |
| `backend/internal/repository/postgres.go` | Commit/rollback implementation | PASS | SHA-256 | `2dc903f7954876014f6f94b2ae399680c950d47acda634ec970af6329d507046` |
| `backend/internal/repository/compliance_repository.go` | Audited transaction contract | PASS | SHA-256 | `56d69c43de27d8ff2056be0764125ee1682f911a148cb7dc6fc809843dffdb38` |
| `backend/internal/repository/types.go` | Audit and mutation interfaces | PASS | SHA-256 | `57f27717e00d382e03225f1c2a903c66604c4ce4fe02cefa2a950951623f4c83` |
| `backend/internal/itemcurator/service.go` | Direct service dependency; concurrent docs excluded | PASS; no Task 264 behavior attributed | SHA-256 | `b3b3b699a6f62ec4c6b51d803a103744cbd01c0bb294ccced38b35167b920407` |
| `backend/internal/repository/manual_food_repository.go` | Global CRUD/idempotency dependency | PASS | SHA-256 | `7cce8d565c88161e3639253cd397a736dccf9915d612255a4c237089ca91b6fa` |
| `backend/internal/cache/classification_invalidator.go` | Existing invalidator behavior | PASS | SHA-256 | `f378e9daef4e183645548cf7cd319a34dccd613561028597ae2ba8ca84eed989` |
| `backend/internal/cache/classification_generation.go` | Shared generation and CAS guard | PASS | SHA-256 | `5ebeafb2d1e0ec6fd0944679d2725c2d43e580d18cf35b77927db59d952bc3e7` |
| `backend/internal/cache/search_cache.go` | Catalog/Substitution cache consumers | PASS | SHA-256 | `5160bdafd92c2b964328c5828978b957da8fd640d5597a0fc11e47513de20a9c` |
| `backend/internal/search/catalog_service.go` | Catalog cache caller | PASS | SHA-256 | `f31d095dadde7f4cef48b47c8f404dedd4c2b5652e796dd53b236e7072cac982` |
| `backend/internal/search/substitution_service.go` | Substitution response/similarity caller | PASS | SHA-256 | `8d5f789a7a5aa4f57318769a8675dc59bd504e547cdb6c0d860712b6fcd6d3a6` |
| `backend/internal/cache/classification_invalidator_test.go` | Existing invalidator regression tests | PASS | SHA-256 | `b149b8a2f3fd25ef311adced0554c74f753b34ca48620bc1f4e7c67b749125be` |
| `backend/internal/cache/classification_generation_integration_test.go` | Existing shared-generation precedent | PASS | SHA-256 | `dfa24e7962aea6f2478d5d7aacab9e67ff7d8b0d19d74f98a90ff7230bfd5f3c` |
| `docs/implementation/02_TASK_LIST.md` | Authoritative task status and criteria | Blocking status gate | SHA-256 | `9b478ce8f10ead7c87a995b05ab10621fa5ad041e0229aed3a63e793fed08e02` |
| `docs/implementation/04_OPEN.md` | Phase coverage/staleness control | Pre-change Task 264 action text remains stale | SHA-256 | `81b097f1ec9503964a714864cf35dc988f4ddef5c0c6e18d9a2011c100aea6c4` |
| `docs/design/DESIGN-009.md` | ItemCurator/admin contract | PASS | SHA-256 | `85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b` |
| `docs/design/DESIGN-011.md` | Cache invalidation/generation contract | PASS | SHA-256 | `6b10db5e2060efda5df11b4fbb78aecbe1c42603d2ac77413400d55cf4a3bfc5` |
| `docs/architecture/ARCH-009.md` | Administration architecture | PASS | SHA-256 | `153607ef21b23caad6805f8c0f77e3ad9584dd8ab20dc7c86a54134905a95e91` |
| `docs/architecture/ARCH-011.md` | Caching architecture | PASS | SHA-256 | `8d98eb6d6e3043ef2acc9bb6a74450406acf1d3b03d96c3c7860a70f05f58073` |
| `docs/architecture/ARCH-013.md` | Security/audit architecture | PASS | SHA-256 | `6a532ffd96b433bf460e4adca46f2567a28b7d412646381d791b91653db5f751` |
| `docs/requirements/01_SOFT_REQ_SPEC.md` | SW-REQ-056/057 source | PASS | SHA-256 | `80b2f57a8c1caebd8b37cdb949cc7e928f3a128a2b8ed81313637b919cafba8b` |
| `docs/implementation/reviews/task-250-review.md` | Dependency review staleness comparison | Checked; no conflicting behavior found | SHA-256 | `ce86eae0f4834d362d236fded9188f4060f8d5330bfc086804bed51b6cb91823` |
| `docs/implementation/reviews/task-251-review.md` | Dependency review staleness comparison | Checked; no conflicting behavior found | SHA-256 | `0d186e0e3cf1449618c704e6f5670dd8491a927dabe05c682dd6061b13b82522` |
| `/home/wiktor/.agents/skills/code-review-skill/SKILL.md` | Review method | Read completely | SHA-256 | `500eee0a40ebfc32741937dc70b1e038ebf81763e26b8bc426dc026477842c80` |
| `/home/wiktor/.agents/skills/code-review-skill/reference/go.md` | Go review guidance | Read completely | SHA-256 | `c183c3ab9440f4f4ab6f2e6862648781042f5cd4de2ca505ee6a9af3acd528c4` |
| `/home/wiktor/.agents/skills/code-review-skill/reference/security-review-guide.md` | Security review guidance | Read completely | SHA-256 | `a0271fe590ff17cbf983477e82603fdc184943b70dd6a6805d7bc6390c7ae02c` |
| `/home/wiktor/.agents/skills/phase-orchestrator/templates/review_checklist.md` | Required review template | Read completely | SHA-256 | `ae4bf4e40498de95912b899d067586158f1c792864cc38aedd50b1ab2d02803` |
| `/home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py` | Required evidence validator | Inspected and run | SHA-256 | `be2c89cf06838a33019dd6458367602ac0b943f0eb14a8b58c7743812a0fcd46` |

```yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "Task 264 preparation has no final hash manifest; current hashes above were captured independently."
  - "docs/implementation/04_OPEN.md still describes the pre-change Task 264 implementation as missing AfterCommit wiring; the action remains OPEN and this review did not edit the control document."
  - "The worktree gained concurrent Task 265-270 preparation/review files during review; they were preserved and excluded."
```

## 10. Coverage and Exceptions

- [x] Focused and complete affected-package tests ran.
- [x] Focused and complete affected-package race tests ran.
- [x] Full package coverage reports were generated and recorded.
- [x] Changed-symbol branches were inspected; task behavior is directly exercised.
- [x] Existing repository coverage deviations are explicit in `docs/implementation/04_OPEN.md`; no new behavior exception is claimed.

```yaml
coverage_required: true
coverage_exception_allowed: true
coverage_report_path: "/tmp/task-264-httpapi-full.cover, /tmp/task-264-app-full.cover, /tmp/task-264-cache-full.cover"
observed_line_coverage: "internal/httpapi 87.4%; internal/app 84.8%; internal/cache 90.1%; newProduction 92.9%; NewClassificationInvalidator 100.0%; Invalidate 85.7%"
coverage_passed: true
```

Coverage finding: The task row names no task-specific exception, but the repository contract explicitly accepts exact measured defensive branches through `04_OPEN.md`; the task’s changed behavior and callback paths are covered. The app composition assertion is intentionally AST-based, while full app tests cover `newProduction` and the live cache test covers Redis behavior independently.

## 11. Negative and Regression Checks

- [x] Existing focused and complete affected-package tests pass.
- [x] No unrelated dependency or architectural boundary was introduced.
- [x] No SQL, migration, OpenAPI, generated client, frontend, or private-item boundary was changed.
- [x] Design sources remain behaviorally compatible: admin CRUD remains audited, cache invalidation remains post-commit, and shared Redis generation/CAS semantics are reused.
- [x] No generated, cache, build, or temporary artifact was added as repository evidence; only the requested review file is being added by this turn.
- [x] The new interface is used by production composition and tests; duplicate constructor/invalidation helpers were searched for.
- [x] Error, cleanup, timeout, concurrency, malformed-input, replay, conflict, audit-failure, and stale-write paths were inspected.

Findings: No blocking or important production defect was found. The optional per-method failure-test gap is recorded without treating it as an implementation regression. Concurrent modifications were not reset, staged, overwritten, or attributed to Task 264.

## 12. Decision

A task may be `PASSED` only when every acceptance criterion and symbol audit passes, current evidence is complete, every reviewed file is hashed, and all pre-review gates are satisfied. The implementation, acceptance criteria, evidence, and final authoritative input-status gate all pass.

```yaml
decision: "PASSED"
reason: "Task 264 implementation evidence is acceptance-positive, the final authoritative task row is PREPARED, all reviewed files are hashed, and all required validation passed."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "Proceed with the phase orchestrator's next review/acceptance step; no production repair is required."
```

## 13. Repair Context

### Review Summary

The implementation review is complete and found no blocking or important production defect. The task-control precondition is satisfied: `docs/implementation/02_TASK_LIST.md:271` is `PREPARED` at final review.

### Minimal Repair Goal

No production repair is required. The optional coverage improvement is to add direct per-mutation assertions for update validation and delete service failure; it is not a release-blocking gap.

### Evidence to Reuse

Reuse the current implementation hashes, `/tmp/task-264-httpapi-full.cover`, `/tmp/task-264-app-full.cover`, `/tmp/task-264-cache-full.cover`, focused test outputs, full affected-package race output, `scripts/check.py --quick`, and the current preparation report.

### Required Re-Review Surface

If the optional tests are added or any task-owned implementation changes, recheck the same 27 inventoried symbols, their direct callers (`AdminController`, `WithMutationAudit`, `withTransaction`, `newProduction`, Catalog Search, Substitution Search, and the shared generation store), the six task-owned paths, the task row, and current concurrent-worktree hashes.

### Do Not Change

Do not alter manual-item behavior, callback ordering, Redis generation semantics, SQL, generated/API/frontend files, unrelated concurrent changes, or any task status other than authorized workflow transitions.
