# Task 251 Review — Global Classification Management

review_decision: "PASSED"
decision: "PASSED"
inventory_source_count: 47
audited_symbol_count: 47
blocking_findings: 0
important_findings: 0
optional_findings: 2
code_review_skill_invoked: true
pre_review_gates_passed: true
inventory_complete: true
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
baseline_confidence: "HIGH"

## 1. Task Source

Task 251 is the current `PREPARED` row in `docs/implementation/02_TASK_LIST.md`, titled “Phase 08 Global Classification Management”, with fixed preparation reference `81ca40ce00cb667ea29243ed2d34068e11229a69`. The preparation is `docs/implementation/preparations/task-251.md`.

The requested surface is global `food_category` and `culinary_role` CRUD, hierarchy and duplicate integrity, deterministic listing, in-use deletion protection, audited admin mutations, cache/filter invalidation, rename propagation, rollback safety, and non-admin 403 behavior. Dependencies 23, 34, 241, and 247 were recorded as `PASSED` by preparation.

The repository-level review template named by the reviewer prompt was not present. I read the full available `code-review-skill` PR template, the full reviewer prompt, and the full evidence validator, then used the validator’s required 13-section evidence format. The code-review skill was invoked exactly once. This is a re-review of the repaired PREPARED task; the prior review findings were checked against the current source and tests.

## 2. Pre-Review Gates

- [x] Current task-list row is `PREPARED`; it was not changed.
- [x] The preparation report, fixed baseline, dependency claims, design references, changed paths, prior review evidence, and current worktree were inspected.
- [x] `code-review-skill` was invoked exactly once; its relevant Go, SQL, HTTP/API, security, caching, concurrency, and testing guidance was applied.
- [x] No production source, migration, task-list row, or task status was edited by this review. The only repository write is this review evidence file.
- [x] Prior Task 241 evidence was checked for staleness. Its explicit handoff says cross-process invalidation of the process-local filter cache belongs to Task 251.
- [x] Current source hashes were captured after inspection; the task-list hash is recorded separately because the dirty worktree already changed it from the preparation hash.
- [x] All Task 251-relevant gates passed. The repository aggregate and full race run still report only the unrelated `TestTask240CustomItemErasureIntegration` failure; no Task 251 package or test failed. The existing OAuth callback lint warning is unrelated and explicitly ignored by the OpenAPI lint configuration.

```yaml
pre_review_gates_passed: true
gate_exception: "Repository-wide aggregate and race commands retain the unrelated Task 240 failure; task-scoped gates pass."
```

## 3. Review Baseline and Change Surface

The baseline is the fixed Git reference from preparation plus the preparation’s symbol-level preservation notes for already-dirty shared files. The worktree contains overlapping Phase 08 implementation, tests, migrations, generated files, and documentation changes from other tasks. Those changes were preserved and were not treated as Task 251 defects unless they were direct callers or required policy dependencies.

Reviewed designs and architecture: `docs/design/DESIGN-009.md` (TagManager and admin policy), `docs/design/DESIGN-005.md` (ClassificationEntity and repository ownership), `docs/architecture/ARCH-009.md` (restricted administration), `docs/architecture/ARCH-011.md` (cache invalidation), `docs/architecture/ARCH-013.md` (security and audit), and the corresponding sections of `docs/architecture/01_SOFT_ARCH_DESIGN.md`. The direct implementation handoff from Task 241 was also reviewed.

The main call graph is:

`NewProduction` → `NewClassificationAdminController` → shared `AdminController` route gateway → controller mutation → `tagmanager.Service` → transaction-bound `ClassificationAdminRepository` → PostgreSQL CRUD/trigger guards; after the audited transaction commits, `ClassificationInvalidator` advances the shared Redis classification generation, clears the local filter projection, and best-effort deletes old search-response keys. Every API instance reads the same generation from Redis. Public filter options and catalog/substitution search caches use generation snapshots and guarded writes.

The repaired implementation is simple and well-separated. CRUD, hierarchy validation, duplicate constraints, in-use checks, audit canonicalization, transaction commit ordering, authorization, shared generation invalidation, versioned keys, and guarded cache writes are correct in the audited paths. The prior two important findings are resolved by the current implementation and deterministic/live tests.

## 4. Acceptance Criteria Checklist

| # | Criterion | Result | Evidence |
|---:|---|---|---|
| 1 | Create, list, update, and delete unused classifications. | PASS | Service, HTTP, and live PostgreSQL CRUD tests pass; repository statements are parameterized and active-row scoped. |
| 2 | Reject duplicate classifications with deterministic hierarchy listing. | PASS | Existing sibling-scoped unique constraint and create conflict mapping reject duplicates; recursive SQL orders parents before descendants with normalized-name/ID tie breaks. |
| 3 | Reject cycles, self-parenting, and cross-kind parents. | PASS | Service ancestor walk and migration 28 trigger guards reject these cases; live repository and service tests pass. |
| 4 | Block deletion of classifications in use by food, meal, private custom item, or active child. | PASS | `classification_is_in_use.sql` checks all four references; migration trigger repeats checks under the transaction advisory lock; live tests cover parent and referenced deletion conflicts. |
| 5 | Serialize hierarchy, assignments, and deletion safely under concurrency. | PASS with test-gap | Migration 28 takes the same global transaction advisory lock and rechecks active parent/assignment/in-use state in triggers. Full and focused race/vet checks pass; no high-contention two-connection stress test exists, retained as optional evidence follow-up. |
| 6 | Propagate a rename into hydrated search results. | PASS | Food hydration and current-label search integration pass. Search entries include the shared classification generation, and `SetIfCurrent` rejects a stale in-flight loader after a committed rename; deterministic interleaving and live Redis tests pass. |
| 7 | Invalidate filter options consistently after create/update/delete. | PASS | Each API instance reads the shared Redis generation. Filter-option cache snapshots are discarded after peer create/rename/delete invalidation, and an in-flight load cannot repopulate an old generation; live two-instance tests pass. |
| 8 | Persist privacy-safe before/after audits atomically and roll back on audit failure. | PASS | SHA-256 label digests and fixed schemas are enforced; live audit tests prove mutation rollback and the HTTP path defers invalidation until commit. |
| 9 | Return 403 for non-admin mutation attempts. | PASS | Admin route middleware verifies the JWT-derived role before CSRF, validation, rate limiting, or handler execution; the HTTP security test passes. |
| 10 | Preserve safe errors, input normalization, and SQL/traceability contracts. | PASS | Strict curation validation, UUID/kind validation, generic error mapping, embedded SQL, design comments, `go vet`, OpenAPI lint, and traceability checks pass. |
| 11 | Meet the repository’s testing and evidence gates. | PASS with documented exceptions | Focused and live integration, full/focused race, vet, vulnerability, OpenAPI, traceability, task-list, evidence-validator, and aggregate checks were run. The aggregate/full-race exception is the unrelated Task 240 failure; coverage and contention-test gaps are optional follow-ups in Section 10. |

## 5. Changed-Symbol Inventory

The 47 rows below enumerate the changed units and their direct callers/consumers. Grouped rows list every symbol in the group, rather than hiding related symbols behind a file-only entry. Test rows list every Task 251 test function inspected.

| # | Symbol or unit | Kind | Location | Direct caller/consumer | Test/evidence |
|---:|---|---|---|---|---|
| 1 | `Service` | type | `backend/internal/tagmanager/service.go:13` | TagManager composition and HTTP controller | service unit tests |
| 2 | `NewService` | constructor | `service.go:19` | `app.NewProduction` | service construction/use |
| 3 | `Service.List` | method | `service.go:25` | classification admin `List` | CRUD/list tests |
| 4 | `Service.Create` | method | `service.go:31` | classification admin `Create` | duplicate/parent tests |
| 5 | `Service.Update` | method | `service.go:40` | classification admin `Update` | rename/reparent tests |
| 6 | `Service.Delete` | method | `service.go:54` | classification admin `Delete` | in-use/unused tests |
| 7 | `validateParent` | helper | `service.go:67` | `Create` and `Update` | cross-kind/cycle tests |
| 8 | `ClassificationRepositoryFactory`, `ClassificationCacheInvalidator`, `ClassificationAdminController` | boundary types | `classification_admin_controller.go:16-32` | shared admin gateway and app composition | compile/use tests |
| 9 | `NewClassificationAdminController` | constructor | `classification_admin_controller.go:35` | `app.NewProduction` | HTTP integration |
| 10 | `AdminRoutes` | route declarations | `classification_admin_controller.go:41` | `AdminController.Routes` | route graph and HTTP tests |
| 11 | `validateCreate`, `validateUpdate` | validators | `classification_admin_controller.go:54-70` | route validation middleware | valid/invalid HTTP bodies |
| 12 | `ClassificationAdminController.List` | handler | `classification_admin_controller.go:72` | GET admin route | list HTTP test |
| 13 | `ClassificationAdminController.Create` | mutation handler | `classification_admin_controller.go:86` | transactional admin gateway | create/conflict/audit tests |
| 14 | `ClassificationAdminController.Update` | mutation handler | `classification_admin_controller.go:108` | transactional admin gateway | rename/reparent tests |
| 15 | `ClassificationAdminController.Delete` | mutation handler | `classification_admin_controller.go:134` | transactional admin gateway | in-use/unused tests |
| 16 | `classificationAuditJSON` | audit projection | `classification_admin_controller.go:156` | mutation audit entry | digest/security/rollback tests |
| 17 | `validateClassificationID` | route validator | `classification_admin_controller.go:167` | PUT/DELETE route middleware | malformed-ID branch audit |
| 18 | `classificationID` | parser | `classification_admin_controller.go:176` | update/delete handlers | UUID validation |
| 19 | `classificationKind` | parser | `classification_admin_controller.go:186` | list/create handlers | kind validation |
| 20 | `ClassificationAdminController.invalidate` | post-commit helper | `classification_admin_controller.go:196` | shared `AfterCommit` callback | invalidation-count test |
| 21 | `AdminMutationResult.AfterCommit` | transaction result field | `admin_controller.go:25-31` | shared mutation gateway | commit/failure tests |
| 22 | `AdminController.transactionalMutation` | transaction gateway | `admin_controller.go:216` | all admin routes | audit, rollback, 403 tests |
| 23 | `ClassificationAdminRepository` | repository contract | `repository/types.go:698` | TagManager and controller factory | compile/live repository tests |
| 24 | `PostgresClassificationRepository.List`, `GetByID` | repository reads | `classification_repository.go:72-107` | TagManager/filter options | live list/not-found tests |
| 25 | `PostgresClassificationRepository.Create`, `Update` | repository mutations | `classification_repository.go:109-127` | TagManager transaction | CRUD/conflict tests |
| 26 | `Upsert`, `validateClassificationMutation`, `validClassificationKind`, `mapClassificationError` | compatibility/validation | `classification_repository.go:129-215` | existing curation plus admin CRUD | existing and focused repository tests |
| 27 | `IsInUse`, `SoftDelete` | deletion safeguards | `classification_repository.go:170-196` | admin delete and DB trigger | live in-use/unused tests |
| 28 | create/get/update classification SQL | embedded SQL | `repository/sql/classification_{create,get_by_id,update}.sql` | repository methods | live PostgreSQL CRUD |
| 29 | recursive classification list SQL | embedded SQL | `repository/sql/classification_list.sql` | `List` and filter options | deterministic hierarchy test |
| 30 | in-use and soft-delete SQL | embedded SQL | `repository/sql/classification_{is_in_use,soft_delete}.sql` | `IsInUse` and `SoftDelete` | deletion tests |
| 31 | `classificationAuditSnapshotSchema`, `sanitizeAdminAuditSnapshot`, canonical JSON rules | audit policy | `compliance_repository.go:551-689` | `WithMutationAudit` | security and rollback tests |
| 32 | `ClassificationRequest.ParentID`, `NormalizeClassification` | input contract | `curation/validation.go:45-50,169` | HTTP validators and TagManager | normalization tests |
| 33 | `FilterOptionInvalidator`, `ClassificationGeneration`, `NewClassificationGeneration`, `ClassificationInvalidator`, `classificationRedisInvalidator`, `NewClassificationInvalidator` | cache boundary/constructor | `cache/classification_generation.go:14-88`, `cache/classification_invalidator.go:10-37` | `NewProduction` and controller | generation, invalidator, and live Redis tests |
| 34 | `ClassificationInvalidator.Invalidate` | cache invalidation | `cache/classification_invalidator.go:40-74` | controller post-commit callback | paginated scan/error and cross-instance generation tests |
| 35 | `NewProduction` classification composition | application wiring | `app/app.go:80-89` | process startup | app route/composition tests |
| 36 | hierarchy, assignment, and in-use trigger functions and triggers | database concurrency guards | `database/migrations/000028_classification_hierarchy_guards.up.sql` | every classification write/assignment/delete | migration replay and live integration |
| 37 | `TestServiceCreateListUpdateDeleteAndHierarchyValidation`, `TestServicePropagatesRepositoryFailures`, `TestCommittedRenameReplacesFilterOptionLabelAfterInvalidation`, and memory-repository methods | service tests/doubles | `tagmanager/service_test.go` | test runner | 100% package coverage |
| 38 | `TestClassificationAdminHTTPCRUDConflictsAuditAndInvalidation`, `TestClassificationAdminHTTPRejectsNonAdminMutationAndAuditFailureInvalidation`, and HTTP repository/invalidator doubles | HTTP tests/doubles | `httpapi/classification_admin_controller_test.go` | test runner | CRUD, 403, audit, invalidation |
| 39 | `TestClassificationAdminRepositoryCRUDHierarchyConflictsAndSearchRename`, `TestClassificationAdminMutationRollsBackWhenAuditFails` | live repository tests | `repository/classification_admin_repository_test.go` | test runner | PostgreSQL CRUD/rename/rollback |
| 40 | `TestClassificationGeneration...`, `TestClassificationInvalidatorClearsFilterAndSearchNamespaces`, `TestClassificationInvalidatorNilAndRedisFailuresAreSafe` | cache tests | `cache/classification_generation_integration_test.go`, `cache/classification_invalidator_test.go` | test runner | shared generation, stale-write guard, local and Redis failure behavior |
| 41 | classification portions of `TestAdminAuditSnapshotsRejectUnsafeOrUnboundedData`, `TestAdminAuditSnapshotValidationRollsBackTransaction`, `TestAdminMutationAuditSuccessfulCommitPath`, `TestAdminMutationAuditReplayCommitsWithoutDuplicateAudit` | audit security tests | `repository/admin_audit_security_test.go` | test runner | schema, rollback, replay |
| 42 | `TestInputNormalizerNormalizesTypedCurationRequests` plus typed-validation tests | normalization tests | `curation/validation_test.go` | test runner | ParentID preservation and safety |
| 43 | `FilterOptionService.Options`, `Invalidate`, generation guard, and classification projection | direct consumer | `search/filter_options.go:50-155` | public filter-options route; controller invalidator | service, peer-instance, and race tests |
| 44 | `SearchResponseStore`, `GetOrLoadSearchResponse`, `SetIfCurrent`, and `SearchSchemaVersion` | direct consumer | `cache/search_cache.go:19-255` | catalog and substitution search | cache, deterministic interleaving, and live Redis tests |
| 45 | food, meal, and private custom-item classification hydration queries | rename consumers | repository food/meal/custom SQL and methods | search responses and filter projections | live rename/hydration tests |
| 46 | `RequireAdmin`, route registration, CSRF/rate-limit/validation middleware | security caller | `httpapi/admin_controller.go`, `router.go`, `curation_validation.go` | all four admin classification routes | non-admin 403 and route tests |
| 47 | `app.NewProduction` route assertions and direct design/traceability links | integration/evidence caller | `backend/internal/app/app_test.go`, `DESIGN-009`, `DESIGN-005`, `ARCH-009`, `ARCH-011`, `ARCH-013` | production composition and static review | app, traceability, OpenAPI checks |

## 6. Function-Level Audit

| # | Audited unit | Correctness and caller audit | Security/concurrency/cache audit | Test result |
|---:|---|---|---|---|
| 1 | `Service` | Owns classification orchestration only; reader and transaction repository are separate seams. | No raw input or persistence bypass. | PASS |
| 2 | `NewService` | Composition is direct and nil behavior is not used by production wiring. | Narrow dependency boundary. | PASS |
| 3 | `Service.List` | Delegates the requested kind to the active-row repository. | Kind validation remains repository-owned. | PASS |
| 4 | `Service.Create` | Validates parent before insert and passes the full typed entity to the transaction repository. | DB trigger is the authoritative concurrent backstop. | PASS |
| 5 | `Service.Update` | Loads the current entity, retains its kind, validates a new parent, then updates name/parent. | Prevents cross-kind reparenting and service-level cycles. | PASS |
| 6 | `Service.Delete` | Loads active entity before soft deletion and returns repository conflicts. | DB rechecks in-use state under advisory lock. | PASS |
| 7 | `validateParent` | Ancestor walk detects self/cycle and cross-kind parent; missing parent errors propagate. | Bounded by cycle detection; DB guard handles races. | PASS |
| 8 | Controller boundary types | Factories force mutation repositories to use the transaction executor; invalidation is post-commit-capable. | Interfaces do not expose raw labels to the cache/audit seam. | PASS |
| 9 | Controller constructor | Stores service, factory, validator, and invalidator without hidden global state. | Production wiring is auditable. | PASS |
| 10 | `AdminRoutes` | Exact GET/POST/PUT/DELETE paths and methods are declared. | Shared gateway adds admin, CSRF, validation, rate limit, audit ordering. | PASS |
| 11 | `validateCreate`/`validateUpdate` | Strict decoder and normalizer reject unknown/duplicate/malformed fields and preserve ParentID. | Safe error envelopes are produced before handler dispatch. | PASS; defensive branches under-covered |
| 12 | `List` handler | Parses kind, calls read service, returns deterministic entities. | Read remains admin-scoped by route. | PASS |
| 13 | `Create` handler | Parses kind/body, builds entity, calls service, and builds safe audit before/after. | Mutation is transaction-bound; invalidation is deferred. | PASS |
| 14 | `Update` handler | Parses UUID/body and returns before/after entity snapshots. | Existing kind cannot be changed by request body. | PASS |
| 15 | `Delete` handler | Parses UUID, loads/deletes active entity, and emits deleted audit state. | In-use conflict is returned without invalidation. | PASS |
| 16 | `classificationAuditJSON` | Emits fixed kind/status/parent metadata and SHA-256 name digest. | Raw administrator labels are not persisted in the snapshot. | PASS |
| 17 | `validateClassificationID` | Rejects malformed route UUIDs before mutation. | Prevents ambiguous identifier handling. | PASS; invalid branch under-covered |
| 18 | `classificationID` | Parses the normalized UUID from Fiber route params. | No SQL interpolation. | PASS; invalid branch under-covered |
| 19 | `classificationKind` | Accepts only the two canonical kinds. | Rejects arbitrary table/column selectors. | PASS; invalid branch under-covered |
| 20 | controller `invalidate` | Calls the composed invalidator after a successful transaction only. | Does not invalidate on audit/mutation failure. | PASS |
| 21 | `AfterCommit` | Carries a callback out of the transaction result without executing it prematurely. | Correct commit boundary. | PASS |
| 22 | `transactionalMutation` | Runs mutation plus audit atomically, serializes response before commit, invokes callback after commit. | Admin role and CSRF middleware precede handler; errors are generic. | PASS |
| 23 | repository interface | Separates read and transaction mutation contracts. | No caller receives an unscoped database handle. | PASS |
| 24 | repository reads | Active-row filtering and kind-scoped hierarchy reads are correct; not-found maps safely. | Context is passed to queries; parameters are bound. | PASS; some error branches under-covered |
| 25 | repository create/update | Inserts/updates normalized labels and parent IDs under DB constraints. | `kind` is retained on update; SQL has no dynamic identifiers. | PASS |
| 26 | compatibility/validation/error helpers | Existing upsert behavior remains compatible while new CRUD validates kind, IDs, and self-parent. | Constraint names are mapped to safe conflict kinds. | PASS; some branches under-covered |
| 27 | in-use/delete methods | Checks food, meal, custom, and child references before soft delete. | Trigger repeats the check after the advisory lock, closing the TOCTOU window. | PASS |
| 28 | CRUD SQL | Statements are static, parameterized, and return current state. | No injection surface. | PASS |
| 29 | recursive list SQL | Parent-before-descendant order is deterministic and kind-scoped. | Active rows only. | PASS |
| 30 | in-use/delete SQL | Explicitly checks all required references and active children. | Soft delete is idempotent for active rows only. | PASS |
| 31 | audit schema/sanitizer | Fixed fields, types, sizes, canonical ordering, and digest format are enforced. | Unknown/raw fields are rejected; transaction rollback preserves atomicity. | PASS |
| 32 | curation request/normalizer | Parent UUID survives normalization while names are normalized. | Strict validator rejects malformed/unknown input. | PASS |
| 33 | invalidator/generation boundary | Shared `ClassificationGeneration` reads, atomically advances, and conditionally writes the Redis generation; invalidator invokes it before local clear and bounded key cleanup. | Redis failures disable unsafe local caching or leave versioned keys unreachable; labels are not exposed to audit/cache metadata. | PASS; defensive constructor/Redis branches are under-covered |
| 34 | `ClassificationInvalidator.Invalidate` | Advances shared generation before post-commit cache cleanup and clears the mutating process’s filter projection. | Versioned keys make scan failure safe; Lua CAS prevents stale in-flight writes from restoring an old generation. | PASS |
| 35 | app wiring | Uses one shared Redis generation for admin invalidation, filter options, catalog search, and substitution search. | All API instances pointed at the same Redis database observe the same generation; similarity cache remains classification-independent. | PASS wiring/live integration |
| 36 | migration guards | Advisory lock serializes hierarchy writes, assignment validation, and deletion; trigger queries recheck active state. | Database enforces invariants even if another caller bypasses Go. | PASS static/live; no stress test |
| 37 | service tests/doubles | Cover CRUD, duplicates, cross-kind, cycles, repository errors, and local rename invalidation. | Memory double is only supplemental; DB guard is separately tested. | PASS; 100% package coverage |
| 38 | HTTP tests/doubles | Cover normal CRUD, conflict, audit digest, invalidation, audit failure, and non-admin 403. | Does not exercise every malformed/error branch or two deployed instances. | PASS behavior; coverage gap |
| 39 | live repository tests | Cover hierarchy order, duplicate/cross-kind conflicts, rename hydration, in-use/child delete, unused delete, and rollback. | PostgreSQL constraints are exercised. | PASS |
| 40 | cache tests | Cover generation current/advance/CAS behavior, local invalidation, paginated Redis scan, nil client, Redis failure, and stale-write interleaving. | Live Redis test exercises separate generation readers and stale token rejection. | PASS |
| 41 | audit security tests | Cover accepted/rejected classification snapshots, rollback, successful commit, and replay. | Digest/raw-label policy is enforced. | PASS |
| 42 | normalization tests | Cover ParentID preservation and normalized typed input plus rejection branches. | Raw sensitive values are not logged. | PASS |
| 43 | filter-options consumer | Captures a shared generation before load, rechecks it after load, and only caches a current snapshot. | Peer invalidation changes the Redis generation; Redis-down mode avoids unsafe local caching. | PASS; peer create/rename/delete integration and race tests |
| 44 | search cache consumer | Captures generation in the key/token and guards post-load writes with atomic `SetIfCurrent`; catalog and substitution callers pass the token. | A stale in-flight loader cannot write an old generation after commit; key scan is only cleanup, not the consistency barrier. | PASS; deterministic interleaving and live Redis tests |
| 45 | hydration consumers | Queries join active classifications and select the current name, so a cache miss sees a rename. | Inactive classifications are excluded. | PASS on miss |
| 46 | admin gateway/router | Route metadata and middleware order enforce admin role, CSRF, strict input, rate limits, and safe errors. | Non-admin mutation is rejected before handler execution. | PASS |
| 47 | app/design/evidence links | Composition and traceability align with DESIGN-009, DESIGN-005, ARCH-009, ARCH-011, and ARCH-013. | Shared Redis generation and guarded writes now satisfy the cross-instance cache consistency requirement. | PASS |

## 7. Findings

No blocking or important findings remain after the repair. The prior F-251-1 cross-instance filter invalidation finding is closed by the shared Redis classification generation and peer-instance create/rename/delete integration test. The prior F-251-2 stale in-flight search write finding is closed by generation-versioned keys/tokens, atomic Lua `SetIfCurrent`, and the deterministic loader interleaving test.

### O-251-1 — Optional: task-owned branch coverage remains below the repository phase target

The relevant package totals are reported in Section 10. Task-owned service behavior is fully covered, while defensive HTTP, constructor, Redis-error, and repository compatibility branches remain below 100%. This is a follow-up evidence improvement, not a correctness or security finding.

### O-251-2 — Optional: add high-contention PostgreSQL stress evidence

Migration 28 uses one global transaction advisory lock for hierarchy writes, assignment validation, and deletion, and the triggers recheck state after taking that lock. Full and focused race runs pass, and live integration tests exercise the invariants, but there is no repeated two-connection contention stress test. This is additional regression evidence, not a blocking design defect.

## 8. Commands Run

All commands below were run on 2026-07-21. Paths are relative to the repository unless noted.

| Command | Result |
|---|---|
| `MEALSWAPP_REDIS_URL=redis://localhost:6379/13 go test -count=1 -p 1 ./internal/cache ./internal/search ./internal/tagmanager ./internal/httpapi` | PASS; focused cache, filter, catalog/substitution, service, and HTTP behavior. |
| `MEALSWAPP_REDIS_URL=redis://localhost:6379/13 go test -count=1 -race ./internal/cache -run 'Test(ClassificationGenerationLiveRedis\|InFlightSearchMiss\|SearchResponseStoreGetSet)'` | PASS; shared generation and stale in-flight search-write interleaving under the race detector. |
| `MEALSWAPP_REDIS_URL=redis://localhost:6379/15 go test -count=1 -race ./internal/search -run 'Test(FilterOptionServiceInvalidationReachesPeerInstance\|FilterOptionServiceCachesCopiesAndInvalidatesAfterAdministration)'` | PASS; peer filter create/rename/delete invalidation and local invalidation behavior. |
| `go test -count=1 -race ./internal/tagmanager` | PASS; service CRUD, duplicate, hierarchy, and repository-error paths. |
| `MEALSWAPP_REDIS_URL=redis://localhost:6379/15 go test -count=1 -race ./internal/httpapi -run TestClassificationAdminHTTP` | PASS; CRUD, audit, invalidation, auth, and failure behavior. |
| `MEALSWAPP_REDIS_URL=redis://localhost:6379/15 go test -count=1 -race ./internal/repository -run 'TestClassificationAdmin\|TestAdminAuditSnapshot'` | PASS; live PostgreSQL CRUD/hierarchy/in-use/rename/hydration/rollback and audit tests. |
| `go test ./... -race -p 1 -count=1` | FAIL only `internal/app.TestTask240CustomItemErasureIntegration`; every task-relevant package passed, including cache, HTTP, repository, search, and TagManager. Redis connection-refused logs came from deliberate failure-path tests. |
| `go test -count=1 -coverprofile=/tmp/task251-tagmanager.cover ./internal/tagmanager` | PASS; 100.0% statements. |
| `go test -count=1 -coverprofile=/tmp/task251-cache.cover ./internal/cache` | PASS; 90.7% package statements; generation and guarded-write functions are covered on tested paths. |
| `go test -count=1 -coverprofile=/tmp/task251-search.cover ./internal/search` | PASS; 96.7% package statements. |
| `go test -count=1 -coverprofile=/tmp/task251-httpapi.cover ./internal/httpapi` | PASS; 87.4% package statements. |
| `go test -count=1 -race -coverprofile=/tmp/task251-repository.cover ./internal/repository -run 'TestClassificationAdmin\|TestAdminAuditSnapshot'` | PASS; 13.1% whole-package total because the repository package contains unrelated surfaces; task-specific classification/audit functions are shown in Section 10. |
| `go vet ./...` | PASS. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS; no vulnerabilities in called code; the tool reported 18 vulnerabilities in required but unreachable module code. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS with the existing ignored OAuth callback 3xx-only warning at `api/openapi.yaml:235`. |
| `python3 scripts/validate-task-list.py` | PASS; 263 sequential tasks with ordered dependencies. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/check.py` | FAIL only at the aggregate backend test stage: `internal/app.TestTask240CustomItemErasureIntegration` reports unrelated Task 240 custom-item cleanup failure. Local stack migration replay, UAT, frontend verification, and earlier aggregate stages passed. |
| `git diff --check -- . ':(exclude)docs/implementation/reviews/task-251-review.md'` | PASS; no whitespace errors in the pre-existing/shared production worktree. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-251-review.md` | Run after this file is written; required structural validator gate. |

## 9. Files Inspected and Staleness Fingerprints

SHA-256 values are current after review inspection. The review file itself is intentionally omitted from its own hash manifest. The current task-list hash differs from the preparation’s claimed before/after hash because the worktree already contains concurrent task changes; this review did not edit it.

| File | SHA-256 |
|---|---|
| `backend/internal/tagmanager/service.go` | `81dec89bcf8122238350f58a0bd218111ed47c9bc8bbf87749a0c91e6fe90ae8` |
| `backend/internal/tagmanager/service_test.go` | `ef71a8f79b3cfed489f0b92171aa64ac362ca8b1c9c395f5594a08bedf1a20ad` |
| `backend/internal/httpapi/classification_admin_controller.go` | `1584656419a549fe7e3975304a7e30feca623e50599043110117a33ff428df08` |
| `backend/internal/httpapi/classification_admin_controller_test.go` | `5393fb3fd2512970f240ca307a07f44663cfc22c91d8cf6626fcf868c2edfc30` |
| `backend/internal/httpapi/admin_controller.go` | `cb1f9bcd0896fadad29b8ede663b13c3a2250d99c13c5efe70d05739f730f` |
| `backend/internal/httpapi/router.go` | `a98d15348d69f6fdf4d5076c2c12f203b20b10c033a81a6fcc25cd7dfacbcb48` |
| `backend/internal/httpapi/curation_validation.go` | `14cd4a46838d84fb643fd9448e8f61c6909f09bf66203ae00d7570d7664c46f1` |
| `backend/internal/repository/types.go` | `7a0069590989b8fe0311c960021c5da87cc3d568b9ca708a7e586b620511f730` |
| `backend/internal/repository/classification_repository.go` | `656c3c86f2694e348298f52c4ecabc278e857d8a9ab9545b36a45c28f38dd841` |
| `backend/internal/repository/compliance_repository.go` | `57790181b840c7f494e44dbd62a7c69be6210cff79987652db6bd6a3852712f6` |
| `backend/internal/repository/classification_admin_repository_test.go` | `cb24919687d86105791395f8046a35b3427d9d744500e52efe29235f5d47d6b6` |
| `backend/internal/repository/admin_audit_security_test.go` | `6398fd2c0a680a5c985dde54f31994cd23766304dc9d1812807785d5466adbda` |
| `backend/internal/repository/food_repository.go` | `01e10872bc32ec184ed42a17941294f51bd235110983615dd4a41c81b244453f` |
| `backend/internal/repository/meal_repository.go` | `14e5a89572b5d8b63e18d5b37f9215da00e97cacd4c8f11e1d51cc21a2e6d0b9` |
| `backend/internal/repository/custom_food_repository.go` | `0b7035b27b6270afe532289c65f1a08ea7547302ee823e1a79c5ae9c0cb0dc5b` |
| `backend/internal/repository/postgres_repository_test.go` | `ceae02b7ee90824286ed621d3f00b756a6ba6101083620a344fb48e74f19d7cd` |
| `backend/internal/repository/sql/classification_create.sql` | `aced3001ff147060d0608c9c3aa11ffc9a37875a2b8f71ef370c065d9c9e6443` |
| `backend/internal/repository/sql/classification_get_by_id.sql` | `c903a61f383450ad44c06537899d103ef30e09492ea15230ef46bbecb89ff976` |
| `backend/internal/repository/sql/classification_update.sql` | `5c3cec4401408545a33d599d7d790c1a599cf8d5084ce94789ac697a35c9f969` |
| `backend/internal/repository/sql/classification_list.sql` | `781b283ef704c36c2f269acfd728b9c8b3128a7e77a975ac893b31b1e2683b0d` |
| `backend/internal/repository/sql/classification_is_in_use.sql` | `515c90403c002b335a21bfb704bad5466a2237ca7ecaf1e9284eb488dacef1b2` |
| `backend/internal/repository/sql/classification_soft_delete.sql` | `dc53f4ab17d0826f9c070f58c689274c569ea05027ccfa86ff610c86d6f771ea` |
| `backend/internal/cache/classification_generation.go` | `5ebeafb2d1e0ec6fd0944679d2725c2d43e580d18cf35b77927db59d952bc3e7` |
| `backend/internal/cache/classification_generation_integration_test.go` | `4109cb819aa9e8a200f1d299c88ece7adca10bb652782a00cfc79bc6b7e75cfe` |
| `backend/internal/cache/classification_invalidator.go` | `f378e9daef4e183645548cf7cd319a34dccd613561028597ae2ba8ca84eed989` |
| `backend/internal/cache/classification_invalidator_test.go` | `b149b8a2f3fd25ef311adced0554c74f753b34ca48620bc1f4e7c67b749125be` |
| `backend/internal/cache/search_cache.go` | `28ea8f2ce78a7a8e0d326c9e6ec162bad4e1c8e24a6a38cf2e88d49a4db522c2` |
| `backend/internal/cache/search_cache_test.go` | `4fbe3d3da6157c74da4f4337c0e82f47e4bbd70129d17c2b8cfcb2f43c9cc9ac` |
| `backend/internal/search/filter_options.go` | `38d834756b43bbeb9bf5365a53eebc58193e30485f9372b5b336a10cf390b1b7` |
| `backend/internal/search/filter_options_test.go` | `17c090ad015553a6bb586b6735592efece263e2c1b55d38147476cae41306735` |
| `backend/internal/search/filter_options_integration_test.go` | `e843a4a53e6c6d2ba7c60312c834b7ddcbad34f0336adc8682c92e9459e04d7b` |
| `backend/internal/search/catalog_service.go` | `54162486ba57cecb05667a669333921912afe25ea838628dc65db732fae484f8` |
| `backend/internal/search/catalog_service_test.go` | `30ad6bbd43b261a23a0fee3fcf8f0281001071de27297fc4bd1a46c1bb03fdb2` |
| `backend/internal/search/substitution_service.go` | `57a829cdbe7f76f15e63b9ba7cffd2227735146c7cd885c2564dacd5af189bed` |
| `backend/internal/curation/validation.go` | `7c3cfdcb6ed41cad3c9b40ec814bbf6f9b6b15990a9b09bb238165e64571b31f` |
| `backend/internal/curation/validation_test.go` | `c3d127aa322dcf639acbe1b31a020617eee7457b0048b9533aca7e98c57072d1` |
| `backend/internal/app/app.go` | `33c22fd95422fe5fbd41b5090c23fcf33a8e4cbf94a6dacf6f0464a869ad0f99` |
| `backend/internal/app/app_test.go` | `f267d7813831a91e664355094959c3cfa8ff57e1d096360932f7c9bd503ca9e2` |
| `database/migrations/000028_classification_hierarchy_guards.up.sql` | `959b8bd85a92a2d3fdb485145107b166a81b61b63729e3d983d5d454fd3c6766` |
| `database/migrations/000028_classification_hierarchy_guards.down.sql` | `b17dd0e8b39c5984b96f8ce3816411d94a4be2a8def336db7dd5b3cbcc831fa4` |
| `docs/design/DESIGN-009.md` | `85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b` |
| `docs/design/DESIGN-005.md` | `91e9f1e152554e5d6eb62093018d57464ac3d38ca2add217215281927f885d31` |
| `docs/architecture/ARCH-009.md` | `153607ef21b23caad6805f8c0f77e3ad9584dd8ab20dc7c86a54134905a95e91` |
| `docs/architecture/ARCH-011.md` | `8d98eb6d6e3043ef2acc9bb6a74450406acf1d3b03d96c3c7860a70f05f58073` |
| `docs/architecture/ARCH-013.md` | `6a532ffd96b433bf460e4adca46f2567a28b7d412646381d791b91653db5f751` |
| `docs/architecture/01_SOFT_ARCH_DESIGN.md` | `eb45a090af681f6dff6a44a0eee51c36719da60ad4a3e06f01d1adf083e0998c` |
| `docs/implementation/preparations/task-251.md` | `3a92f9bf196f3340a17cd74ea68f060fcb7b7823f0916747fb2eb32f6dcf08f9` |
| `docs/implementation/preparations/task-241.md` | `b0f7ce961ff0d5f44b592b52f7d2d5f537e10dea802944e79ca89e3128c5a031` |
| `docs/implementation/reviews/task-241-review.md` | `6cc5964ae93466158f861f2133a0be1be6e69217367a5a5593e400f6c14c7c6d` |
| `docs/implementation/reviewer-prompt.md` | `92c9b71361a50868becf0b9a9895071bdd657e8c092afb8c1b19691cb569386d` |
| `docs/implementation/02_TASK_LIST.md` | `9085f355eb060181a16c04ef439c6f30afd67889d3550b971580643d6c379a7d` |
| `docs/implementation/04_OPEN.md` | `4b703eca5a8b6207ce0e87fc0ea23c9df255ac8923ab8d93e8b4f8ff1f318e4d` |
| `/home/wiktor/.agents/skills/code-review-skill/SKILL.md` | `500eee0a40ebfc32741937dc70b1e038ebf81763e26b8bc426dc026477842c80` |
| `/home/wiktor/.agents/skills/code-review-skill/assets/pr-review-template.md` | `a440ec7aa749aeb3164f0a7ddded9a0ede40aa83786383e28d841cfb09f37eb3` |
| `/home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py` | `be2c89cf06838a33019dd6458367602ac0b943f0eb14a8b58c7743812a0fcd46` |

## 10. Coverage and Exceptions

Coverage was measured from the current focused profiles `/tmp/task251-cache.cover`, `/tmp/task251-search.cover`, `/tmp/task251-httpapi.cover`, `/tmp/task251-repository.cover`, and `/tmp/task251-tagmanager.cover`. The TagManager service package is 100.0%; the relevant package totals are cache 90.7%, HTTP 87.4%, search 96.7%, and the task-specific repository run is 13.1% for the whole repository package because unrelated repository surfaces are intentionally excluded from that run.

Task-owned function coverage is:

- `tagmanager.Service`, `NewService`, `List`, `Create`, `Update`, `Delete`, and `validateParent`: 100.0% each.
- `curation.NormalizeClassification`: 100.0%.
- `ClassificationGeneration`: `NewClassificationGeneration` 66.7%, `Current` 75.0%, `Advance` 66.7%, and `SetIfCurrent` 75.0%; `ClassificationInvalidator.Invalidate` 85.7% and its constructor 100.0%.
- `SearchResponseStore`: `GetSearchResponse` 92.9%, `SetSearchResponse` 80.0%, generation-key construction 100.0%, and `GetOrLoadSearchResponse` 73.9%; the generic adapter’s `Current`/`SetIfCurrent` methods are not directly covered because production uses the explicit `SearchResponseStore.Generation` seam.
- `classification_admin_controller.go`: constructor and route/invalidate helpers reach 100.0%, while `validateCreate` 66.7%, `validateUpdate` 66.7%, `List` 71.4%, `Create` 76.9%, `Update` 68.8%, `Delete` 76.9%, `classificationAuditJSON` 80.0%, `validateClassificationID` 66.7%, `classificationID` 75.0%, and `classificationKind` 75.0% remain below 100.0%.
- Shared `AdminController.transactionalMutation`, `RequireAdmin`, and router registration reach 100.0% in the full HTTP profile.

This is recorded as optional finding O-251-1 rather than a blocking correctness defect because the covered paths prove the main behavior and the missing lines are mostly defensive/error branches. The repository guidance still sets a 100% phase goal, the task row declares no exception, and `docs/implementation/04_OPEN.md` has no Task 251 coverage deviation; add focused tests or document the exception before an eventual phase-completion review.

The full-backend race run and aggregate check’s Task 240 failure is an existing unrelated worktree exception, not evidence of a Task 251 package defect. It remains visible in the command record rather than being repaired here. No migration deadlock occurred in this run; local-stack migration replay passed.

## 11. Negative and Regression Checks

| Check | Result |
|---|---|
| Duplicate sibling create | PASS; database uniqueness and repository conflict mapping. |
| Same label in a different kind | PASS; kind-scoped uniqueness and cross-kind parent rejection. |
| Self-parent and multi-node cycle | PASS; service and migration guard. |
| Missing/inactive/cross-kind parent | PASS; service/repository validation plus database trigger. |
| Food, meal, private custom item, and active-child deletion use | PASS; SQL and live integration. |
| Concurrent hierarchy/assignment/delete reasoning | PASS static audit and full/focused race runs; advisory lock is consistent, but no contention stress test exists. |
| Rename after filter cache warm-up in the same process | PASS; invalidator and integration tests. |
| Rename after search cache warm-up | PASS; generation-versioned key/token and guarded write reject the stale in-flight interleaving. |
| Cross-process filter cache invalidation | PASS; separate Redis-backed instances observe create, rename, and delete generation changes. |
| Audit contains no raw label and rejects unsafe fields | PASS; digest/schema tests. |
| Mutation rollback when audit persistence fails | PASS; live PostgreSQL transaction test. |
| Invalidation is not called after mutation/audit failure | PASS; HTTP test and post-commit gateway inspection. |
| Non-admin mutation | PASS; 403 before handler. |
| SQL injection/dynamic identifier review | PASS; embedded static parameterized SQL and canonical kind allowlist. |
| Race detector/vet/vulnerability/OpenAPI/traceability | PASS for relevant commands; aggregate Task 240 exception is isolated above. |

Optional finding O-251-2: add a deterministic two-connection contention test for migration 28. The trigger design is sound on inspection and live cache tests now cover two Redis-backed instances, but repeated database contention evidence would further protect the concurrency claim from future changes.

## 12. Decision

**PASSED.**

There are no blocking or important findings. The repaired implementation satisfies CRUD, deterministic hierarchy, duplicate and cross-kind/cycle guards, all in-use deletion protections, rename propagation, privacy-safe atomic audits, admin authorization, shared Redis generation invalidation across API instances, versioned cache keys, and guarded cache writes against stale in-flight repopulation. Targeted/live/full race evidence passes for Task 251. The aggregate and full race commands retain an unrelated Task 240 failure; coverage and database contention stress gaps remain optional follow-ups.

```yaml
review_decision: "PASSED"
decision: "PASSED"
blocking_findings: 0
important_findings: 0
optional_findings: 2
```

## 13. Repair Context

The PREPARED repair changed the cache design and added shared Redis generation/CAS evidence. This re-review did not edit production source or the task list. The only repository write was this report.

1. Closed F-251-1: shared Redis generation now invalidates filter projections on peer API instances; live create/rename/delete tests pass.
2. Closed F-251-2: generation-versioned search keys and atomic `SetIfCurrent` prevent stale in-flight cache repopulation; deterministic interleaving tests pass.
3. Optional follow-up: add defensive tests or document the coverage exception in `docs/implementation/04_OPEN.md` before phase completion.
4. Optional follow-up: add repeated two-connection PostgreSQL contention evidence for migration 28.
