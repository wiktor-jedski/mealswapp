# Task 251 preparation — Global Classification Management

## Outcome and scope

- Task: 251, `DESIGN-009: TagManager`.
- Fixed Git reference: `81ca40ce00cb667ea29243ed2d34068e11229a69`.
- Dependencies 23, 34, 241, and 247 were `PASSED` at preparation start.
- Task 251 remains `PREPARED`. The repair did not edit `docs/implementation/02_TASK_LIST.md`; its repair before/after SHA-256 is `9085f355eb060181a16c04ef439c6f30afd67889d3550b971580643d6c379a7d`.
- Scope is limited to backend global Food Category/Culinary Role CRUD, hierarchy and duplicate integrity, audited admin routes, classification-derived cache invalidation, and direct tests/evidence.

## Baseline and preservation

The worktree was already substantially dirty with overlapping Phase 08 tasks. The complete initial `git status --short` was captured before edits. In the task-251 surface, `app.go`, `types.go`, `compliance_repository.go`, `classification_is_in_use.sql`, and the task list were already modified; curation/admin gateway files were untracked dependency work. Task-251 additions were absent. Unrelated task 248/250/252 work continued in shared files during preparation and was preserved at symbol/hunk level. No reset, checkout, clean, generated-client change, OpenAPI edit, task transition, or unrelated repair was performed.

| Baseline path | Initial SHA-256 |
|---|---|
| `backend/internal/repository/classification_repository.go` | `2d5073f1fe54d0e7a6a291bdbf9ee6e38531fe96ece7575d7fcf1c56325c0368` |
| `backend/internal/repository/types.go` | `5534be37a865c95390f84687ed82007e0adbca63a94fbd8c7e849ccb8cc40ac6` |
| `backend/internal/repository/sql/classification_list.sql` | `d6c41d6512fbd7459fd34a51285d76a45615bf1832917ee134f93c2740d899b6` |
| `backend/internal/repository/sql/classification_is_in_use.sql` | `2d381174a19601807474b77675b42da39e00b1c5ebed1e0f6cd2785e9edad1c5` |
| `backend/internal/httpapi/admin_controller.go` | `94d3841f21c30bd2939e2be1ac46d8d984c68a95765e488854335bff6ed6fe3c` |
| `backend/internal/curation/validation.go` | `8b66ed5241864693c7634b0d4dd41aa30535625daa57976455871ba19a3274f6` |
| `backend/internal/app/app.go` | `42dadf16664b29dcfff32cf4b6b1643a2b9e5bdaa0a776b9d63eabd8498b0243` |
| `docs/implementation/02_TASK_LIST.md` | `689954f8dc9a17c2344db0e03be72e1555aabc9d8756c0d8213763e9ba7c3a96` |

Baseline confidence is high for new files and the exact in-turn patches; shared-file ownership is recorded by symbol because those files contain preserved dependency and concurrent task work.

## Implemented behavior

- Admin routes: `GET /api/v1/admin/classifications?kind=...`, `POST /api/v1/admin/classifications/:kind`, `PUT /api/v1/admin/classifications/:classificationId`, and `DELETE /api/v1/admin/classifications/:classificationId`.
- Verified JWT-cookie admin authorization, CSRF, strict normalized bodies, UUID/kind validation, scoped rate limiting, and generic safe errors are inherited from the PASSED admin gateway. Non-admin mutations return 403.
- Explicit create rejects normalized duplicates; list uses recursive name/ID hierarchy paths for deterministic parent-before-descendant ordering; update supports rename/reparent without kind changes; delete soft-deletes only unused leaves.
- Service validation walks ancestors to reject cycles and cross-kind parents. Migration 28 repeats these invariants in PostgreSQL and serializes hierarchy edits, classification assignments, and soft deletion with a transaction advisory lock, covering concurrent application instances.
- Food, meal, private custom-item, and active-child references block deletion. Assignment triggers reject attaching inactive classifications.
- Mutation and privacy-safe before/after audit snapshots commit in the same transaction. Administrator-authored labels are represented by SHA-256 digests rather than persisted raw text. Audit failure rolls the mutation back.
- Post-commit invalidation atomically advances a shared Redis classification generation before clearing the mutating process's filter snapshot and bounded batches of the versioned Redis catalog-search namespace. Every API instance compares its filter snapshot with that shared generation, so create, rename, and delete are observed by non-mutating instances.
- Catalog and substitution searches carry the generation observed on the initial cache lookup across the database load. Search keys include that generation, and an atomic Redis Lua compare-and-set writes the response only if the generation is still current. An invalidation therefore prevents an older in-flight miss from restoring stale hydrated labels.

## Changed paths and symbols

### Production

- `backend/internal/tagmanager/service.go` (new): `Service`, `NewService`, `List`, `Create`, `Update`, `Delete`, `validateParent`.
- `backend/internal/httpapi/classification_admin_controller.go` (new): `ClassificationRepositoryFactory`, `ClassificationCacheInvalidator`, `ClassificationAdminController`, constructor, `AdminRoutes`, create/update validators, `List`, `Create`, `Update`, `Delete`, UUID/kind helpers, `classificationAuditJSON`, and post-commit `invalidate`.
- `backend/internal/httpapi/admin_controller.go`: added `AdminMutationResult.AfterCommit`; modified `transactionalMutation` to run it only after successful mutation/audit commit.
- `backend/internal/repository/types.go`: added `ClassificationAdminRepository`.
- `backend/internal/repository/classification_repository.go`: added embedded CRUD statements and the admin contract; modified `List`, `Upsert`, and `SoftDelete`; added `GetByID`, `Create`, `Update`, mutation/kind validation, and guarded conflict mapping.
- `backend/internal/repository/compliance_repository.go`: extended `adminAuditSnapshotRule` and fixed schemas/canonicalization for classification UUID/digest/kind/boolean metadata; added `classificationAuditSnapshotSchema`.
- `backend/internal/curation/validation.go`: extended `ClassificationRequest` with `ParentID`; modified `NormalizeClassification` to preserve the typed hierarchy input.
- `backend/internal/cache/classification_invalidator.go` (new): `FilterOptionInvalidator`, `ClassificationInvalidator`, `classificationRedisInvalidator`, `NewClassificationInvalidator`, and `Invalidate`.
- `backend/internal/cache/classification_generation.go` (repair): Redis-backed `ClassificationGeneration`, shared generation reads/advances, and atomic `SetIfCurrent` guarded writes.
- `backend/internal/cache/search_cache.go` (repair): generation-versioned search keys, explicit `SearchResponseCacheToken`, guarded catalog/substitution writes, and matching protection for `GetOrLoadSearchResponse`.
- `backend/internal/search/filter_options.go` (repair): optional shared-generation source, generation-tagged process-local snapshots, cross-instance refresh, Redis-down source fallback, and guarded in-flight local cache population.
- `backend/internal/search/catalog_service.go` and `substitution_service.go` (repair): carry lookup tokens across source loads and advertise miss metadata only when the guarded write succeeds.
- `backend/internal/app/app.go`: modified `NewProduction` to compose TagManager, transaction repository factory, validator, audit gateway, shared classification generation, versioned filter options, and guarded Redis search storage alongside preserved concurrent admin controllers. Redis-disabled composition retains the existing process-local generation guard.
- `backend/internal/repository/sql/classification_create.sql`, `classification_get_by_id.sql`, `classification_update.sql` (new): parameterized CRUD statements.
- `backend/internal/repository/sql/classification_list.sql`: recursive deterministic hierarchy statement.
- `backend/internal/repository/sql/classification_is_in_use.sql`: active child hierarchy usage check, preserving the pre-existing custom-item check.
- `database/migrations/000028_classification_hierarchy_guards.up.sql` / `.down.sql` (new): hierarchy, assignment, and in-use concurrency guards and reversible trigger/function lifecycle.

### Tests

- `backend/internal/tagmanager/service_test.go` (new): memory repository helpers; service CRUD, duplicate/cross-kind/cycle/failure tests; filter rename/invalidation test. Task-owned service statements reach 100% coverage.
- `backend/internal/httpapi/classification_admin_controller_test.go` (new): HTTP CRUD/list, duplicate conflict, in-use 409, audit snapshots, post-commit invalidation, audit-failure non-invalidation, and non-admin 403.
- `backend/internal/repository/classification_admin_repository_test.go` (new): live PostgreSQL deterministic CRUD/hierarchy/duplicates, cross-kind rejection, rename propagation into food search results, in-use/parent delete conflicts, unused delete, and audit rollback.
- `backend/internal/cache/classification_invalidator_test.go` (new): filter and paginated search-namespace deletion plus nil/Redis-failure behavior.
- `backend/internal/cache/search_cache_test.go` (repair): deterministic channel-gated search miss/invalidation interleaving proving an old generation cannot repopulate cache.
- `backend/internal/cache/classification_generation_integration_test.go` (repair): live Redis, two-service-instance create/rename/delete refresh plus stale-token write rejection.
- `backend/internal/search/filter_options_test.go` (repair): two service instances sharing one generation prove peer refresh after create, rename, and delete.
- `backend/internal/search/catalog_service_test.go` and `backend/internal/httpapi/search_controller_test.go` (repair): updated guarded cache contract fixtures without changing search, authorization, or usage-gate behavior.
- `backend/internal/repository/admin_audit_security_test.go`: classification audit format allow/reject cases.
- `backend/internal/curation/validation_test.go`: normalized parent-ID preservation.

## Review repair

The repair addresses both important findings in `docs/implementation/reviews/task-251-review.md`:

1. F-251-1: filter-option invalidation is now shared across API instances through the Redis classification generation. Both deterministic and live Redis tests cover create, rename, and delete on a non-mutating instance.
2. F-251-2: an in-flight search miss now retains its pre-load generation and can write only through an atomic compare-and-set. Generation-versioned keys also make all older entries unreachable immediately after invalidation.

Mutation/audit transaction boundaries, `AfterCommit` ordering, privacy-safe audit snapshots, hierarchy and duplicate guards, rename hydration, in-use deletion protection, and non-admin 403 behavior were not changed. Their existing focused repository/service/HTTP tests pass.

## Verification

Commands ran on 2026-07-21 with local PostgreSQL and Redis services running.

| Command | Result |
|---|---|
| `MEALSWAPP_REDIS_URL=redis://localhost:6379/13 go test -count=1 -p 1 ./internal/cache ./internal/search ./internal/tagmanager ./internal/httpapi ./internal/repository` (`backend/`, repository-local caches) | PASS, including live Redis multi-instance create/rename/delete and stale-write rejection plus live PostgreSQL audit/rename/in-use coverage. |
| Same focused command with `-race` | PASS; no race report. |
| `go test ./... -p 1 -count=1` with Redis DB 10 | All task-251 and other packages PASS; overall FAIL only in unrelated Task 240 `TestTask240CustomItemErasureIntegration`: `transactional account cleanup left 2 owner custom items`. |
| `go test -race ./... -p 1 -count=1` with Redis DB 11 | All task-251 and other packages PASS with no race report; overall FAIL only at the same unrelated Task 240 assertion. |
| `go vet ./...` (`backend/`) | PASS. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` (`backend/`) | PASS: no vulnerabilities in called code or imported packages; 18 required-module advisories are not called. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS: 263 sequential tasks with ordered dependencies; task 251 remains `PREPARED`. |
| `python3 scripts/check.py` | Traceability, task-list, Go Doc, OpenAPI, script tests, vet, vulnerability scan, local stack, phase UAT, frontend verification, and focused integration gates PASS. Aggregate stops at the same unrelated Task 240 full-backend failure before its built-in race stage; the full race command above was run separately. |
| `git diff --check` | PASS. |

## Verification-criteria assessment

Every task-251 criterion is directly covered: create/list/update/delete, duplicate and cycle rejection, in-use 409, renamed labels in filter options and hydrated search results, post-commit cross-instance invalidation, stale in-flight write rejection, before/after audits, audit rollback, and non-admin 403. The only full/aggregate exception is outside task 251 and was preserved rather than repaired.

## Repair final hashes

| Path | SHA-256 |
|---|---|
| `backend/internal/cache/classification_generation.go` | `5ebeafb2d1e0ec6fd0944679d2725c2d43e580d18cf35b77927db59d952bc3e7` |
| `backend/internal/cache/classification_generation_integration_test.go` | `4109cb819aa9e8a200f1d299c88ece7adca10bb652782a00cfc79bc6b7e75cfe` |
| `backend/internal/cache/classification_invalidator.go` | `f378e9daef4e183645548cf7cd319a34dccd613561028597ae2ba8ca84eed989` |
| `backend/internal/cache/classification_invalidator_test.go` | `b149b8a2f3fd25ef311adced0554c74f753b34ca48620bc1f4e7c67b749125be` |
| `backend/internal/cache/search_cache.go` | `28ea8f2ce78a7a8e0d326c9e6ec162bad4e1c8e24a6a38cf2e88d49a4db522c2` |
| `backend/internal/cache/search_cache_test.go` | `4fbe3d3da6157c74da4f4337c0e82f47e4bbd70129d17c2b8cfcb2f43c9cc9ac` |
| `backend/internal/search/filter_options.go` | `38d834756b43bbeb9bf5365a53eebc58193e30485f9372b5b336a10cf390b1b7` |
| `backend/internal/search/filter_options_test.go` | `17c090ad015553a6bb586b6735592efece263e2c1b55d38147476cae41306735` |
| `backend/internal/search/catalog_service.go` | `54162486ba57cecb05667a669333921912afe25ea838628dc65db732fae484f8` |
| `backend/internal/search/catalog_service_test.go` | `30ad6bbd43b261a23a0fee3fcf8f0281001071de27297fc4bd1a46c1bb03fdb2` |
| `backend/internal/search/substitution_service.go` | `57a829cdbe7f76f15e63b9ba7cffd2227735146c7cd885c2564dacd5af189bed` |
| `backend/internal/httpapi/search_controller_test.go` | `41e24daa374c079d66e9d903e52cd288d69c5db517144e258a9479d1d6093e1c` |
| `backend/internal/app/app.go` | `33c22fd95422fe5fbd41b5090c23fcf33a8e4cbf94a6dacf6f0464a869ad0f99` |
| unchanged during repair `docs/implementation/02_TASK_LIST.md` | `9085f355eb060181a16c04ef439c6f30afd67889d3550b971580643d6c379a7d` |
