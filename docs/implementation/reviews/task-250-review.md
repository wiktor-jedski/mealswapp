# Review Evidence: Task 250 — DESIGN-009 ItemCurator

~~~~yaml
task_id: 250
component: "ItemCurator"
static_aspect: "DESIGN-009 manual global food-item CRUD, idempotency, validation, search, isolation, and audit transaction"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-21T14:10:00Z"
review_agent: "Codex independent reviewer"
evidence_file: "docs/implementation/reviews/task-250-review.md"
baseline_ref: "HEAD 81ca40ce00cb667ea29243ed2d34068e11229a69 plus current Task-250 worktree files"
baseline_confidence: "MEDIUM"
code_review_skill_invoked: true
pre_review_gates_passed: true
relevant_language_guide: "Go + HTTP/auth/CSRF + PostgreSQL transactions + security + concurrency"
repair_context_required: false
~~~~

## 1. Task Source

**Description:** Implement administrator-only manual create, read, update, and soft-delete for global curated food items, with macros, micronutrients, images, physical state, density provenance, classifications, creation idempotency, search propagation, audit rollback, and strict separation from user-owned custom items.

**Depends On:** 33 PASSED; 34 PASSED; 43 PASSED; 238 PASSED; 247 PASSED.

**Task row:** `250 | Phase 08 Manual Global Item CRUD | DESIGN-009: ItemCurator | PREPARED`.

**Verification criteria:** The task table’s valid CRUD, replay/conflict, duplicate-name, field-validation, density, audit, search, rollback, authentication, and global/private isolation criteria were checked against source and tests. No OpenAPI change is attributed to this task; that contract is task 253.

## 2. Pre-Review Gates

- [x] The selected row is exactly task 250 and is `PREPARED`.
- [x] All listed dependencies are `PASSED` in the current task table.
- [x] `docs/design/01_TECH_STACK.md`, `docs/architecture/ARCH-009.md`, and `docs/design/DESIGN-009.md` were read.
- [x] The full repository reviewer template `docs/implementation/reviewer-prompt.md` was read.
- [x] The worktree was inspected as dirty shared Phase 08 work; no merge, reset, checkout, cleanup, production edit, or task-list edit was performed.
- [x] Task-owned and directly shared source hashes match the preparation fingerprints, except the task table’s expected concurrent status transition from `OPEN` to `PREPARED`.

The template requests a branch merge, but no phase branch/PR target was supplied and merging would mutate this deliberately shared worktree. The fixed HEAD and current task-scoped files were reviewed instead.

## 3. Review Baseline and Change Surface

The fixed baseline is `81ca40ce00cb667ea29243ed2d34068e11229a69`. The worktree contains concurrent Phase 08 changes for tasks 238–252. Task 250 attribution is limited to the ItemCurator symbols, the adjacent `DESIGN-009` shared transaction/route changes, their callers, and their direct validation/isolation dependencies. Concurrent task-240 and task-252 failures are not attributed to this task.

The implementation writes only `food_items` and `food_item_classifications`. `custom_food_items` remains a separate owner-required table and is reached only through the existing profile/custom-item route and repository. Manual creation claims use the existing `mutation_idempotency_keys` table with fixed administrator, method, route, and key scope.

## 4. Acceptance Criteria Checklist

| Criterion | Result | Evidence |
|---|---|---|
| Valid global create, read, update, and soft-delete | PASS | `Service` delegates to the global repository; PostgreSQL integration covers hydrated CRUD and active/deleted visibility. |
| Stable create replay | PASS | The normalized request SHA-256 is stored with the administrator-scoped `POST /admin/items` claim; the immutable stored 201 response is replayed and no second audit is written. |
| Conflicting key reuse | PASS | A different normalized body returns the typed idempotency conflict before item or audit side effects. |
| Duplicate global name | PASS | The active normalized-name unique index and transaction rollback map duplicate names to safe 409 conflict behavior. |
| Macros and physical state | PASS | HTTP and repository validation reject invalid/non-finite/negative values, reject solid macro totals over 100, allow dense liquids, and reject unsupported states. |
| Micronutrients | PASS | Active vocabulary lookup occurs in the mutation executor transaction; unknown keys are rejected and classified safely. |
| Images | PASS | Request decoding rejects malformed/unsafe schemes and bounds the field; persistence is parameterized and responses are owner-free. |
| Density and provenance | PASS | Solid liquid-only fields are rejected; liquid items require positive density and imported/manual/estimated provenance. |
| Classifications | PASS | Duplicate JSON/UUID inputs are rejected at HTTP, IDs are kind-checked against active classifications, and assignments are replaced atomically. |
| Before/after audit snapshots | PASS | Create, update, and delete emit only bounded booleans and the closed `solid|liquid` enum; repository schemas reject free text and unsafe fields. |
| Search propagation | PASS | PostgreSQL search sees a created item, loses the old name after update, sees the new name, and excludes a soft-deleted item. |
| Audit rollback | PASS | Invalid audit metadata rolls back both item state and provisional idempotency claim; no rolled-back item remains searchable. |
| Global/private isolation | PASS | The global table has no owner column, private routes use a distinct owner predicate/table, same names may coexist, and neither repository reads the other’s IDs. |
| Auth, CSRF, rate limits, and errors | PASS | `NewAdminController` marks every route admin-only; mutation routes require verified cookie auth, CSRF, validation, rate, security audit, and fail-closed mutation audit. Error envelopes are generic and request-correlated. |
| Design and traceability | PASS | Task-specific modules and generated SQL carry precise `DESIGN-009 ItemCurator` comments; traceability validation passes. |

## 5. Changed-Symbol Inventory

| # | Symbol or unit | Role | Direct callers/tests | Result |
|---:|---|---|---|---|
| 1 | `AdminAuditChanges`, `ManualFoodItemCreateClaim*` | Transaction and claim contracts | Admin gateway, ItemCurator, repository tests | PASS |
| 2 | `WithMutationAudit` | Mutation-plus-audit atomic boundary | Admin controller and repository integration/security tests | PASS |
| 3 | `adminAuditSnapshotSchemas`, `sanitizeAdminAuditSnapshot` | Bounded privacy schema | Audit repository and security tests | PASS |
| 4 | `validateFoodItemWithExecutor` | Global item invariant validation in caller transaction | Manual repository create/update | PASS |
| 5 | Classification and micronutrient executor helpers | Active-kind/vocabulary validation | Manual repository and food repository tests | PASS |
| 6 | `PostgresManualFoodItemRepository` constructor and `GetByID*` | Global-only reads | ItemCurator service and repository integration | PASS |
| 7 | `ClaimCreate` | Durable create claim, create, response completion, replay | ItemCurator service and PostgreSQL integration | PASS |
| 8 | Manual repository `Update` and `Delete` | Transaction-scoped replacement and soft delete | ItemCurator service and PostgreSQL integration | PASS |
| 9 | `createManualFoodItem`, `getManualFoodByID` | Global persistence and hydrated state | ClaimCreate and CRUD integration | PASS |
| 10 | Claim scanner and validator helpers | Scope/hash/response integrity | Claim paths and repository coverage | PASS |
| 11 | Embedded manual claim SQL trio | Parameterized idempotency persistence | `ClaimCreate` | PASS |
| 12 | ItemCurator `Request`, `Item`, `CreateResult`, `MutationResult` | Typed owner-free API boundary | Service and HTTP controller | PASS |
| 13 | `Service`, `NewService` | Global-only application service | Production wiring and service tests | PASS |
| 14 | `Service.Create` | Normalize, hash, claim, decode immutable response | Manual create handler and service tests | PASS |
| 15 | `Service.Get` | Active global read | Manual GET handler and service tests | PASS |
| 16 | `Service.Update` | Authoritative before/update/after state | Manual PUT handler and service tests | PASS |
| 17 | `Service.Delete` | Authoritative before/soft-delete state | Manual DELETE handler and service tests | PASS |
| 18 | `requestHash`, `toEntity`, `fromEntity`, `validationError` | Stable mapping and projection helpers | Service methods | PASS |
| 19 | `ManualItemService`, `ManualItemController`, route constructor | Admin controller seam and route policy | Production app and HTTP tests | PASS |
| 20 | Manual `Create` handler | Admin identity, create response, replay audit behavior | POST integration test | PASS |
| 21 | Manual `Get` handler | Global active read projection | GET integration test | PASS |
| 22 | Manual `Update` and `Delete` handlers | Mutation projection and audit snapshot handoff | PUT/DELETE integration test | PASS |
| 23 | Manual validators and UUID parsing | Strict body, duplicate-key, ownership, key, and path checks | HTTP rejection tests | PASS |
| 24 | Manual response, audit, dependency, and error helpers | Safe DTO/envelope boundaries | HTTP tests and global error classifier | PASS |
| 25 | `NewProduction` manual service/controller wiring | Production caller composition | App compile and route construction | PASS |
| 26 | Admin route registration and transactional wrapper; router admin middleware | Auth/CSRF/validation/rate/audit order | Admin gateway tests | PASS |
| 27 | `service_test.go` | Unit replay/conflict/CRUD/validation coverage | ItemCurator package | PASS |
| 28 | `manual_item_controller_test.go` | HTTP CRUD/security/error coverage | HTTP package | PASS |
| 29 | `manual_food_repository_test.go` | PostgreSQL CRUD/search/rollback/isolation coverage | Repository package | PASS |
| 30 | `admin_audit_security_test.go` and shared audit tests | Snapshot privacy/cause/replay/rollback coverage | Repository package | PASS |
| 31 | `customitem.ValidateRequest` dependency | Shared normalization and physical/image validation | Manual service plus private-item tests | PASS |
| 32 | Private custom-item callers and catalog search/schema SQL | Isolation and visibility boundary | Profile/custom controllers, food search, migrations | PASS |

~~~~yaml
inventory_source_count: 32
audited_symbol_count: 32
inventory_complete: true
generated_groupings:
  - "No generated artifacts are in the Task-250-owned surface; OpenAPI remains task 253."
~~~~

## 6. Function-Level Audit

| # | Unit | Correctness and edge paths | Security/state/concurrency | Evidence |
|---:|---|---|---|---|
| 1 | Claim/audit types | Replay is explicit and mutation-derived fields are separate from gateway identity. | Prevents replay from carrying a second audit. | Source and contract compilation PASS. |
| 2 | `WithMutationAudit` | Callback, audit validation, commit, and rollback are ordered correctly. | Same PostgreSQL transaction and context are used; audit failure is fail-closed. | Repository rollback/cause/replay tests PASS. |
| 3 | Snapshot schema/sanitizer | Malformed, oversized, unknown, wrong-type, wrong-enum, and invalid digest/UUID snapshots reject. | Only bounded booleans/enums/digests/UUIDs persist. | Security tests PASS. |
| 4 | Food invariant validation | Checks name, state, macros, density, classifications, and active micro vocabulary. | Runs against supplied transaction executor. | Full coverage profile and integration PASS. |
| 5 | Classification/micro helpers | Wrong kind, inactive, nonexistent, and unsupported micro keys reject. | No caller-owned transaction is escaped. | Repository tests PASS. |
| 6 | Global repository reads | Active reads are global-only; deleted rows are hidden unless explicitly requested internally. | No owner parameter or private table access exists. | Isolation integration PASS. |
| 7 | `ClaimCreate` | First writer creates; replay locks and checks body; incomplete/corrupt claims reject. | Unique scope serializes concurrent retries and rolls back on failure. | Replay/conflict/rollback integration PASS. |
| 8 | Manual update/delete | Active-row predicates produce typed not-found; classifications and soft-delete are atomic. | Mutation executor prevents split-brain audit state. | CRUD/search integration PASS. |
| 9 | Create/get helpers | Create validates before insert, attaches classifications, then hydrates authoritative response. | Parameterized SQL and same transaction are used. | PostgreSQL integration PASS. |
| 10 | Claim scanner/validator | UUID/scope/key/hash/encoder checks prevent malformed internal claims. | Stored response is immutable and JSON-valid before replay. | Repository tests and coverage profile PASS. |
| 11 | Claim SQL | Insert-once, `FOR UPDATE` lookup, and body-matched completion are parameterized. | Administrator/method/route/key scope cannot cross private claims. | SQL inspection and integration PASS. |
| 12 | DTOs | Response includes macros, micros, measures, density, classifications, and image without owner. | No private owner field is representable. | Service/HTTP tests PASS. |
| 13 | Service construction | Store dependency is explicit and nil service paths classify as unavailable. | Store interface exposes global-only operations. | Unit tests and production compile PASS. |
| 14 | `Service.Create` | Trimmed key, normalized request, stable hash, replay decode, and typed conflict are handled. | Server-derived admin ID is passed; no request owner is accepted. | Unit and HTTP tests PASS. |
| 15 | `Service.Get` | Nil ID and missing item are typed; active item is projected. | Global repository boundary prevents IDOR into private items. | Unit/integration PASS. |
| 16 | `Service.Update` | Reads before state, updates validated entity, rereads after state. | All operations share gateway transaction for audit authority. | Unit/repository integration PASS. |
| 17 | `Service.Delete` | Reads authoritative before state then soft-deletes active row. | No hard delete or private-table mutation. | Unit/repository integration PASS. |
| 18 | Mapping helpers | Classification kinds and nil micros are normalized deterministically. | Request hash is based on normalized request, not raw JSON. | Service tests PASS. |
| 19 | Controller seam/routes | Four documented routes have explicit read/mutation shapes and rate limits. | `NewAdminController` adds admin/auth/CSRF/audit controls. | Route registration tests PASS. |
| 20 | POST handler | Status and DTO are deferred until commit; replay returns `Replayed` with no audit changes. | Identity comes from verified admin context; idempotency key is header-only. | HTTP replay/audit tests PASS. |
| 21 | GET handler | UUID parsing and service errors map to safe envelopes. | Route is admin-only and global-only. | HTTP read/isolation tests PASS. |
| 22 | PUT/DELETE handlers | Before/after snapshots are authoritative and delete returns 204. | Snapshot contains no name, PII, raw payload, or secret. | HTTP and audit tests PASS. |
| 23 | Validators/parsing | Duplicate JSON keys, unknown/owner fields, malformed macros/micros, duplicate IDs, images, density, and path IDs reject. | Validation runs after auth/CSRF and before rate/audit/mutation. | HTTP rejection/order tests PASS. |
| 24 | DTO/audit/error helpers | Empty micros and safe statuses serialize consistently; dependency/domain errors map generically. | Internal causes are omitted by `writeError`. | HTTP/error tests PASS. |
| 25 | Production wiring | Manual repository, service, audit repository, and controller are composed once. | No private service is injected into the manual controller. | App compile and caller search PASS. |
| 26 | Gateway/router | Auth then admin then CSRF then validation then rate then security audit then mutation is enforced. | Signed cookie claims and server request IDs defeat spoofing; mutations fail closed. | Admin gateway tests PASS. |
| 27 | Service tests | Cover first create, exact replay, changed body, duplicate name, CRUD, invalid macros/image/micro, and liquid density. | Global-only memory store contract is explicit. | `go test` and race PASS. |
| 28 | HTTP tests | Cover valid CRUD, replay audit count, duplicate keys/IDs, invalid fields, ownership injection, conflicts, and private route absence. | Admin auth/CSRF/security gateway is exercised. | `go test` and race PASS. |
| 29 | PostgreSQL integration | Covers hydrated macros/micros/image/classifications, search create/update/delete, duplicate, rollback, and private/global cross-reads. | Real constraints and transaction behavior are tested. | Focused `go test` and race PASS. |
| 30 | Audit security tests | Unsafe snapshots, persistence cause, rollback, commit, and replay no-op are covered. | No duplicate audit on replay; cause does not become a client message. | Focused tests PASS. |
| 31 | Shared custom-item validation | Manual routes reuse strict field normalization and repository validation without adding owner input. | Private route remains authenticated and owner-predicate based. | Dependency source/tests and manual HTTP tests PASS. |
| 32 | Private/catalog boundary | Private routes point to `custom_food_items`; catalog search points to `food_items` and excludes deleted rows. | Same-name global/private records are distinct and cross-repository IDs return not-found. | SQL, migration, route, and integration inspection PASS. |

## 7. Findings

No blocking, important, or optional findings remain in the Task-250 review surface.

The below-100 package/function lines are the documented Task 250 coverage deviation in `docs/implementation/04_OPEN.md`: remaining lines are defensive nil/dependency, malformed internal claim/encoder, impossible JSON marshal, and repeated repository error branches. Every task acceptance behavior listed in the task table has direct tests; this is not treated as an implementation finding.

~~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 0
~~~~

## 8. Commands Run

| Command | Working directory | Result | Evidence |
|---|---|---|---|
| `go test -count=1 ./internal/itemcurator ./internal/httpapi` | `backend/` | PASS | ItemCurator and HTTP packages pass. |
| `go test -count=1 ./internal/repository -run 'TestPostgresManualFoodItemCRUD|TestAdminMutationAuditReplayCommitsWithoutDuplicateAudit|TestAdminAuditSnapshotsRejectUnsafeOrUnboundedData|TestAdminAuditSnapshotValidationRollsBackTransaction|TestAdminAuditPersistenceErrorPreservesCause|TestAdminMutationAuditSuccessfulCommitPath'` | `backend/` | PASS | Focused PostgreSQL CRUD, audit, rollback, privacy, and replay tests pass. |
| `go test -count=1 -race ./internal/itemcurator ./internal/httpapi ./internal/repository -run 'TestServiceCreateReplayConflictDuplicateAndCRUD|TestServiceRejectsInvalidFieldsAndLiquidDensity|TestManualItemAdminHTTP|TestPostgresManualFoodItemCRUD|TestAdminMutationAuditReplayCommitsWithoutDuplicateAudit|TestAdminAudit'` | `backend/` | PASS | Selected service, HTTP, repository, and audit tests pass with no race report. |
| `go test -count=1 -coverprofile=/tmp/task-250-itemcurator-review.cover ./internal/itemcurator` | `backend/` | PASS | Package coverage 74.7%. |
| `go test -count=1 -coverprofile=/tmp/task-250-httpapi-review.cover ./internal/httpapi` | `backend/` | PASS | Package coverage 87.4%. |
| Retained full profile `/tmp/task-250.cover` plus `go tool cover -func` | `backend/` | PASS | Preparation-matched full profile: itemcurator 74.7%, HTTP 87.4%, repository 91.9%, combined 89.3%; task behavior branches covered, deviation documented. |
| `go test -count=1 -coverprofile=/tmp/task-250-repository-focused.cover ./internal/repository -run 'TestPostgresManualFoodItemCRUD|TestAdminMutationAuditReplayCommitsWithoutDuplicateAudit|TestAdminAuditSnapshotsRejectUnsafeOrUnboundedData|TestAdminAuditSnapshotValidationRollsBackTransaction|TestAdminAuditPersistenceErrorPreservesCause|TestAdminMutationAuditSuccessfulCommitPath'` | `backend/` | PASS | Selected-test profile 18.2%; this is supplemental, not the full package measurement. |
| `gofmt -d` on reviewed Go files | repository root | PASS | No formatting diff. |
| `git diff --check` | repository root | PASS | No whitespace errors. |
| `go vet ./...` | `backend/` | PASS | No diagnostics. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `backend/` | PASS | Zero reachable vulnerabilities; 18 unreachable required-module advisories. |
| `npx --no-install redocly lint api/openapi.yaml` | repository root | PASS with existing warning | OpenAPI valid; one ignored OAuth callback 302-only 2XX warning. |
| `python3 scripts/validate-task-list.py` | repository root | PASS | 263 ordered tasks; task 250 remains PREPARED. |
| `python3 scripts/validate-traceability.py` | repository root | PASS | Traceability validation passed. |
| `python3 scripts/check.py` | repository root | ENVIRONMENT FAILURE | Reached local-stack migration; shared concurrent PostgreSQL migration contention produced `pg_type_typname_nsp_index`. Earlier aggregate stages for traceability, task list, docs, OpenAPI, scripts, vet, and security passed. |
| `go test -count=1 ./...` | `backend/` | ENVIRONMENT/CONCURRENT FAILURE | Unrelated task-240 erasure test failed its cleanup assertion; the run was stopped while concurrent unrelated Phase 08 tests held the shared database. Task-250 focused packages had already passed. |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-250-review.md` | repository root | PASS | Structural review-evidence validator passes. |

## 9. Files Inspected and Staleness Fingerprints

Hashes were computed from the current worktree after verification. The task-list hash is intentionally the current `PREPARED` row, not the preparation’s earlier `OPEN` control. No file in this review was edited after hashing except this review evidence file itself.

| File | Purpose | SHA-256 |
|---|---|---|
| `backend/internal/app/app.go` | Production ItemCurator wiring | `e9fa64094fdbff1b2b8e88857dd119b400d337f9401464ee284cffb2e17c5409` |
| `backend/internal/repository/types.go` | Claim and audit contracts | `7a0069590989b8fe0311c960021c5da87cc3d568b9ca708a7e586b620511f730` |
| `backend/internal/repository/compliance_repository.go` | Transactional audit and sanitizer | `57790181b840c7f494e44dbd62a7c69be6210cff79987652db6bd6a3852712f6` |
| `backend/internal/repository/food_repository.go` | Global validation and hydration helpers | `01e10872bc32ec184ed42a17941294f51bd235110983615dd4a41c81b244453f` |
| `backend/internal/repository/manual_food_repository.go` | Manual global CRUD/idempotency repository | `7cce8d565c88161e3639253cd397a736dccf9915d612255a4c237089ca91b6fa` |
| `backend/internal/repository/sql/manual_food_create_claim.sql` | First-writer claim SQL | `cb268b43f6301fe68598e14f811ecd0f38f3b92ff9ed1cbe945631e7663cb4fc` |
| `backend/internal/repository/sql/manual_food_create_claim_get.sql` | Replay/conflict lock SQL | `085972e305710dc289ecf32dffcbcad0de0b9845b6cb65c9375a02efa33d5b90` |
| `backend/internal/repository/sql/manual_food_create_claim_complete.sql` | Immutable response completion SQL | `d3385f705c9ba162aad090a6ff1c448ab9d445834cd3e391d3d02b171dd4a80d` |
| `backend/internal/repository/sql/food_create.sql` | Global insert SQL | `d59f729f616f9b044ec39855013f0cb8349a705cb59fe28255e01224cc96958d` |
| `backend/internal/repository/sql/food_update.sql` | Global update SQL | `8d7e31eebf9129ab1dbfc57ea25768255065a28df51ce4455d77a23bc4a0c6fb` |
| `backend/internal/repository/sql/food_soft_delete.sql` | Global soft-delete SQL | `fd708efc321ffab11926d2da5cf2d9257cb3ff68804b427d77d94bf18cc83edd` |
| `backend/internal/repository/sql/food_get_by_id.sql` | Active global read SQL | `35dadc1e0c351401c91b1c798744e7edbc41eec343f204218f6c78c6e26e14d4` |
| `backend/internal/repository/sql/food_search.sql` | Catalog search SQL | `1d1dcd005ab745ee3da32d5dca8ded558e6931b03d834ecdf43e837253f966d4` |
| `backend/internal/repository/sql/food_search_count.sql` | Catalog count SQL | `552f2d6350bbb0ce84636e04db37a0087032192b55b7189bb5de7a1a6dd9a78b` |
| `backend/internal/repository/sql/food_clear_classifications.sql` | Classification replacement clear | `71e5a3d73a59dd8e22e50f5937d4b74335112b47f4329fcb35d7257e354614b2` |
| `backend/internal/repository/sql/food_attach_classification.sql` | Classification assignment | `68809d640c394a2eb31c59cae4f1576bad0c4f7a1a4794d195f38f524d2f0707` |
| `backend/internal/repository/sql/food_list_classifications.sql` | Classification hydration | `2704a8a06223dd336d77e2d8ced494360377512addb1f6386df0eabcd97d12ae` |
| `backend/internal/repository/sql/food_validate_classification.sql` | Kind/active validation | `eced28dcd61cb749d93fbd95e70ca4f10ef49b6955218b95b367828e510873f5` |
| `backend/internal/itemcurator/service.go` | ItemCurator service | `7bd8e1a99c795d318e8dc7e4988b571f4aa6ddefe35730d22057bf02324a3729` |
| `backend/internal/httpapi/manual_item_controller.go` | Manual admin HTTP boundary | `b7ec1af1f64a48461922915ff5d2aa012ee870f42e0983fb407d00e6f1496b4c` |
| `backend/internal/httpapi/admin_controller.go` | Admin routing/transaction wrapper | `cb1f9bcd0896fadad29c29b8ede663b13c3a2250d99c13c5efe70d05739f730f` |
| `backend/internal/httpapi/router.go` | Gateway auth/CSRF/request ID order | `a98d15348d69f6fdf4d5076c2c12f203b20b10c033a81a6fcc25cd7dfacbcb48` |
| `backend/internal/customitem/service.go` | Shared request normalization dependency | `4bc9eb6ae297aec1b1030a23084143d27edf9f807e02f07073d4bce541b3975b` |
| `backend/internal/httpapi/custom_item_controller.go` | Private item validation and handlers | `4ea8018aa044b3ab34ee54d8391e9dd4cd3a08dc911a800888d9daec791d4d0a` |
| `backend/internal/httpapi/profile_controller.go` | Private route caller registration | `38b8a2bebab80c3079dce54d57fe0157e55a8e448c42d7814b05d618150c4965` |
| `backend/internal/itemcurator/service_test.go` | Unit coverage | `5e3932a2ecfff6d3fe6d1f4c822f9fcdbc6620e755a07bf1103dfdd5fdee7bfe` |
| `backend/internal/httpapi/manual_item_controller_test.go` | HTTP coverage | `b62b8a8a714be320bf3fffeff93325125e5f19a3880dca8fa8444dbddc19046e` |
| `backend/internal/httpapi/admin_controller_test.go` | Gateway and rollback coverage | `a946248885c974cb44d4abba90157f8a66bbad1bcd6fa4b1a90e582e17e8ac13` |
| `backend/internal/repository/manual_food_repository_test.go` | PostgreSQL CRUD/isolation coverage | `bad81f579e53756bcf168b00cfd631c776eb24cc5a259296e749f0fb51067c60` |
| `backend/internal/repository/admin_audit_security_test.go` | Audit privacy/rollback coverage | `6398fd2c0a680a5c985dde54f31994cd23766304dc9d1812807785d5466adbda` |
| `database/migrations/000002_food_items.up.sql` | Global food table/unique search index | `13012e45e3e4b20d71e19a1c913d2f4b20f060a61d43f1cb5c1b53b8ccc160f6` |
| `database/migrations/000003_classifications.up.sql` | Global classification tables | `768cc69f5a030b2e81ae9ac9715a233abfd0221247ed1e1fa887d1a93b099987` |
| `database/migrations/000021_mutation_idempotency.up.sql` | Shared idempotency table migration | `354e77c40320c5c31711241e130a460ee9be86b974f20baa6c4266d3f609a08b` |
| `database/migrations/000025_user_owned_custom_food_items.up.sql` | Separate private table and owner constraints | `dc3e479dd9ba72d39ceeb0a93c3d99b6097c4a862e5800f93ed3612d4f5c5091` |
| `docs/implementation/02_TASK_LIST.md` | Current PREPARED task control | `a659cdcf0bbdd8e00d83c6f08167f1ea262dfe60837890505b3683615a4d12d6` |
| `docs/implementation/04_OPEN.md` | Coverage deviation control | `4b703eca5a8b6207ce0e87fc0ea23c9df255ac8923ab8d93e8b4f8ff1f318e4d` |
| `docs/implementation/preparations/task-250.md` | Preparation evidence | `1dc47478ca284e18099ceb2e46fb55b80fcd68bc5350249dbe8512e34e5e4776` |
| `docs/design/01_TECH_STACK.md` | Stack contract | `64e2cf45ec039db597244678b17e8028f4705b86dcad01e7051e3e686d6f9338` |
| `docs/design/DESIGN-009.md` | ItemCurator design contract | `85119dd44195103f75d2297751304299ddf5f1c4713dc81108b731af6e438b3b` |
| `docs/architecture/ARCH-009.md` | Administration architecture | `153607ef21b23caad6805f8c0f77e3ad9584dd8ab20dc7c86a54134905a95e91` |
| `docs/implementation/reviewer-prompt.md` | Review template | `92c9b71361a50868becf0b9a9895071bdd657e8c092afb8c1b19691cb569386d` |

~~~~yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "Preparation reported task 250 as OPEN at its capture point; the current task table is PREPARED and its current hash is recorded above."
  - "The dirty worktree contains concurrent Phase 08 changes; only Task-250 symbols and direct shared controls are attributed here."
~~~~

## 10. Coverage and Exceptions

- [x] Focused ItemCurator, HTTP, and repository tests ran.
- [x] Focused race tests ran with no race report.
- [x] Full retained Task-250 coverage profile was inspected.
- [x] Package measurements are recorded: itemcurator 74.7%, HTTP 87.4%, repository 91.9%, combined 89.3%.
- [x] The precise below-100 deviation is recorded in `docs/implementation/04_OPEN.md` and waives no required behavior.
- [x] All task-owned executable functions and defensive branches were reviewed against the profile; lower lines are non-acceptance defensive/error paths.

~~~~yaml
coverage_required: true
coverage_exception_allowed: true
coverage_report_paths:
  - "/tmp/task-250.cover"
  - "/tmp/task-250-itemcurator-review.cover"
  - "/tmp/task-250-httpapi-review.cover"
  - "/tmp/task-250-repository-focused.cover"
observed_package_coverage: "ItemCurator 74.7%; HTTP 87.4%; repository 91.9%; combined 89.3%"
observed_task_local_function_coverage: "All required behavior branches directly covered; remaining defensive/dependency/error branches are documented"
coverage_passed: true
~~~~

## 11. Negative and Regression Checks

- [x] Anonymous and non-admin requests cannot reach manual routes; spoofed role, user, and request-ID headers are ignored.
- [x] CSRF is required for POST, PUT, and DELETE and precedes validation and mutation dispatch.
- [x] Idempotency key is required, trimmed for scope, normalized-body hashed, locked for replay, and conflict-safe across concurrent writers.
- [x] Replay commits no second item or audit; replay carrying mutation-derived audit changes is rejected.
- [x] Duplicate active names conflict; a failed duplicate or audit write leaves no provisional claim or food row.
- [x] Invalid macros, micros, images, state, density, classifications, duplicate JSON keys, duplicate UUIDs, owner fields, and malformed IDs reject before mutation.
- [x] Global CRUD uses parameterized embedded SQL and never queries `custom_food_items`.
- [x] Private CRUD predicates on owner and private routes are not registered by the manual controller.
- [x] Search visibility follows active global rows and soft-delete; old updated names disappear and new names appear.
- [x] Audit snapshots reject raw names, PII, secrets, provider payloads, free text, wrong types, malformed JSON, and oversized payloads.
- [x] Context is passed through gateway, repository, claim, validation, search, and audit operations; no Task-250 goroutines are introduced.
- [x] `go vet`, `govulncheck`, OpenAPI lint, traceability, task-list validation, formatting, and diff checks pass.
- [x] No production code or task-list file was changed by this review; only this review evidence file is being added.

## 12. Decision

A task may be `PASSED` only when the current row is PREPARED, all acceptance criteria and audited symbols pass, evidence is current, all reviewed files are hashed, task-local coverage is complete or precisely documented, and no blocking/important finding remains. Those conditions are met for task 250.

~~~~yaml
decision: "PASSED"
reason: "Manual ItemCurator global CRUD is transactionally audited, idempotent, validated, searchable, ownerless by construction, isolated from private custom items, and protected by the verified admin gateway. Focused tests and race tests pass; the only aggregate failures are shared-worktree database contention and unrelated task-240 cleanup behavior."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "Keep task 250 PREPARED until the phase orchestrator applies its normal status transition; address shared PostgreSQL contention and the unrelated task-240 failure in their own scopes."
~~~~

## 13. Repair Context

This is an independent review of the PREPARED Task-250 surface. The preparation evidence was used to identify the intended boundary and prior command results, then current source, callers, tests, design documents, hashes, and focused commands were rechecked. No repair was requested or performed.

The current source hashes match the preparation’s final Task-250 fingerprints. The task table’s current `PREPARED` status is a concurrent control update and was not edited by this review.
