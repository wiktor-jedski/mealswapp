# Review Evidence: Task 238 — DESIGN-005: FoodItemEntity

~~~yaml
task_id: 238
component: "FoodItemEntity"
static_aspect: "DESIGN-005"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-20T22:59:20Z"
review_agent: "independent-reviewer-rereview"
evidence_file: "docs/implementation/reviews/task-238-review.md"
baseline_ref: "81ca40ce00cb667ea29243ed2d34068e11229a69"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "code-review-skill reference/go.md, reference/cross-cutting/sql-injection-prevention.md, reference/cross-cutting/n-plus-one-queries.md, reference/cross-cutting/error-handling-principles.md, reference/cross-cutting/async-concurrency-patterns.md"
repair_context_required: false
~~~

## 1. Task Source

**Description:** Phase 08 User-Owned Custom Item Persistence: add an explicit user-owned custom food-item persistence model structurally distinct from global curated food_items, with mandatory owner predicates, repository CRUD, and existing food-item invariants.

**Depends On:** 33, 34, 43, 112

**Testing Coverage Exceptions:** None.

**Verification Criteria:** Forward/down migrations and repository integration tests prove owner-scoped create/read/update/delete, same-user visibility, cross-user non-disclosure, global curated-item isolation, duplicate-name behavior within the documented scope, canonical units, active micronutrient validation, required liquid density provenance, parameterized SQL loaded from backend/internal/repository/sql/, and no ownerless private row can be persisted.

## 2. Pre-Review Gates

- [x] Input status is PREPARED. Task-list row 238 is PREPARED.
- [x] Every dependency is PREPARED or PASSED. Direct inspection shows 33, 34, 43, and 112 are PASSED.
- [x] The preparation report claims completion and records the repair delta.
- [x] A task-specific baseline and diff are available and trustworthy.
- [x] code-review-skill was invoked exactly once for this re-review and its complete Go guide plus relevant SQL/database guidance was read.
- [x] The reviewer is independent from implementation and repair.
- [x] Review uses current repository state rather than stale preparation claims.
- [x] Reviewer made no production-code or task-list changes.

~~~yaml
pre_review_gates_passed: true
blocking_issue: "NONE"
~~~

## 3. Review Baseline and Change Surface

Baseline/reference method: HEAD equals the fixed baseline. The current task-owned implementation is therefore the working-tree change set against 81ca40ce00cb667ea29243ed2d34068e11229a69. Tracked diffs, untracked paths, preparation evidence, task status, and current source contents were independently inspected.

Commands used to reconstruct the diff:

~~~bash
git rev-parse HEAD
git status --short
git diff --name-status 81ca40ce00cb667ea29243ed2d34068e11229a69
git diff --stat 81ca40ce00cb667ea29243ed2d34068e11229a69
git diff -- docs/design/DESIGN-005.md backend/internal/repository/types.go backend/internal/repository/sql/classification_is_in_use.sql docs/implementation/02_TASK_LIST.md
rg --files backend/internal/repository/sql database/migrations docs/implementation/preparations
rg -n "CustomFoodItem|custom_food_items|customFood|owner_id" backend database docs/design/DESIGN-005.md
~~~

Pre-existing dirty-worktree changes and exclusions:

The current worktree contains the Task 238 implementation, its preparation report, and the status-only transition of row 238 from OPEN to PREPARED. The status transition predates this re-review and was inspected but not edited. The prior review evidence was stale and was overwritten as requested. Generated caches, frontend build output, coverage output, and temporary screenshots were excluded from the task diff. No task-owned implementation change could not be distinguished.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| backend/internal/repository/sql/classification_is_in_use.sql | working-tree modification | HIGH | custom classification EXISTS branch |
| backend/internal/repository/types.go | working-tree modification | HIGH | CustomFoodItemEntity; CustomFoodItemRepository |
| docs/design/DESIGN-005.md | working-tree modification | HIGH | private entity, ownership, name, and repository contract rules |
| docs/implementation/02_TASK_LIST.md | pre-existing status transition | HIGH | Task 238 PREPARED status only; no executable unit |
| backend/internal/repository/custom_food_repository.go | working-tree addition | HIGH | seven embedded SQL variables; repository type; assertion; constructor; four methods; two helpers |
| backend/internal/repository/custom_food_repository_test.go | working-tree addition and repair | HIGH | fixture; CRUD, error-branch, slice-immutability, and validation tests; fixture helper |
| backend/internal/repository/sql/custom_food_create.sql | working-tree addition | HIGH | parameterized create statement |
| backend/internal/repository/sql/custom_food_get_by_id.sql | working-tree addition | HIGH | owner-scoped read statement |
| backend/internal/repository/sql/custom_food_update.sql | working-tree addition | HIGH | owner-scoped update statement |
| backend/internal/repository/sql/custom_food_soft_delete.sql | working-tree addition | HIGH | owner-scoped soft-delete statement |
| backend/internal/repository/sql/custom_food_clear_classifications.sql | working-tree addition | HIGH | classification clear statement |
| backend/internal/repository/sql/custom_food_attach_classification.sql | working-tree addition | HIGH | classification attach statement |
| backend/internal/repository/sql/custom_food_list_classifications.sql | working-tree addition | HIGH | classification hydration statement |
| backend/internal/repository/sql/testdata/custom_food_ownerless_create.sql | working-tree addition | HIGH | ownerless negative fixture |
| database/migrations/000025_user_owned_custom_food_items.up.sql | working-tree addition | HIGH | table, checks, indexes, join table, schema version |
| database/migrations/000025_user_owned_custom_food_items.down.sql | working-tree addition | HIGH | reverse migration |
| docs/implementation/preparations/task-238.md | working-tree addition and repair | HIGH | current preparation, repair, coverage, and command evidence |

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Forward migration creates the private persistence model. | Migration inspection and aggregate migration cycle. | PASS | Migration 000025 up creates custom_food_items, constraints, indexes, classification join table, and schema version 25; scripts/check.py migration up/down/up passed. |
| 2 | Down migration reverses the private model. | Down migration inspection and rollback execution. | PASS | Migration 000025 down removes the reverse index, join table, owner indexes, table, and schema version; aggregate migration cycle passed. |
| 3 | Create is owner-scoped and owner is mandatory. | Repository integration, implementation audit, and DDL. | PASS | Create rejects nil OwnerID, binds owner_id, and the database column is NOT NULL; API and direct-SQL ownerless tests pass. |
| 4 | Read is owner-scoped. | Cross-owner integration and SQL predicate inspection. | PASS | GetByID binds owner_id and id; same-owner read succeeds and wrong-owner read maps to not_found. |
| 5 | Update is owner-scoped. | Same-owner and cross-owner integration plus SQL inspection. | PASS | Update binds owner_id and id, maps zero affected rows to not_found, and persists same-owner changes. |
| 6 | Delete is owner-scoped. | Same-owner and cross-owner integration plus SQL inspection. | PASS | Soft-delete binds owner_id and id; wrong-owner deletion is not_found and same-owner deletion hides the active row. |
| 7 | Same-user visibility works. | Real PostgreSQL CRUD integration. | PASS | Owner A creates, reads, updates, and rereads the item and hydrated classifications. |
| 8 | Cross-user access does not disclose existence. | Cross-owner read, update, and delete tests plus error mapping audit. | PASS | Owner B receives typed not_found for all three operations; owner predicates are present in every relevant statement. |
| 9 | Private items are isolated from global curated items. | Separate repositories, same-name integration, and ID checks. | PASS | Same-named global and private rows coexist in separate tables and neither repository reads the other ID. |
| 10 | Duplicate names follow owner scope and soft-delete release. | Unique partial index and integration tests. | PASS | Same-owner normalized duplicate conflicts, different owner and global same-name rows succeed, and soft deletion releases the active reservation. |
| 11 | Name comparison trims and case-folds. | Generated normalized column, index, and mixed-case/whitespace integration. | PASS | lower(btrim(name)) is indexed; My Tofu versus two-space mixed-case mY tOfU conflicts for one owner. |
| 12 | Canonical units and repository conversion are preserved. | Metric and imperial integration plus shared conversion audit. | PASS | Metric values round-trip and imperial reads convert solid weight and liquid volume through existing boundary helpers. |
| 13 | Active micronutrient vocabulary validation is preserved. | Validation integration and shared validator audit. | PASS | Canonical Sodium succeeds; alias Na and inactive Legacy are rejected with typed errors. |
| 14 | Liquid density and provenance are required. | Shared validation, DDL checks, and valid/invalid liquid integration. | PASS | Valid density and manual provenance round-trip; missing density and missing provenance fail. |
| 15 | SQL is parameterized. | SQL source inspection and static interpolation scan. | PASS | All custom SQL uses positional placeholders; no interpolation or concatenation was found. |
| 16 | SQL is loaded from backend/internal/repository/sql/. | go:embed inspection and file inventory. | PASS | All seven production custom statements are embedded from the required directory. |
| 17 | No ownerless private row can be persisted. | NOT NULL DDL, API validation, direct SQL fixture, and integration test. | PASS | Nil owner fails before I/O and direct ownerless insert fails at the database NOT NULL boundary. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | CustomFoodItemEntity | type | backend/internal/repository/types.go:94 | added | custom repository and future services | CRUD and validation integration |
| 2 | CustomFoodItemRepository | interface | backend/internal/repository/types.go:577 | added | compile assertion and future services | package compile; CRUD integration |
| 3 | customFoodCreateSQL | embedded SQL | backend/internal/repository/custom_food_repository.go:10-13 | added | Create | CRUD integration |
| 4 | customFoodGetByIDSQL | embedded SQL | backend/internal/repository/custom_food_repository.go:15-18 | added | GetByID | CRUD and error-branch integration |
| 5 | customFoodUpdateSQL | embedded SQL | backend/internal/repository/custom_food_repository.go:20-23 | added | Update | CRUD and error-branch integration |
| 6 | customFoodSoftDeleteSQL | embedded SQL | backend/internal/repository/custom_food_repository.go:25-28 | added | Delete | CRUD and error-branch integration |
| 7 | customFoodClearClassificationsSQL | embedded SQL | backend/internal/repository/custom_food_repository.go:30-33 | added | replaceClassifications | CRUD and error-branch integration |
| 8 | customFoodAttachClassificationSQL | embedded SQL | backend/internal/repository/custom_food_repository.go:35-38 | added | replaceClassifications | CRUD and error-branch integration |
| 9 | customFoodListClassificationsSQL | embedded SQL | backend/internal/repository/custom_food_repository.go:40-43 | added | hydrateClassifications | CRUD and error-branch integration |
| 10 | PostgresCustomFoodItemRepository | type | backend/internal/repository/custom_food_repository.go:45-49 | added | all custom repository methods | CRUD, validation, and error-branch integration |
| 11 | CustomFoodItemRepository compile assertion | compile-time contract | backend/internal/repository/custom_food_repository.go:51-52 | added | Go compiler | package tests |
| 12 | NewPostgresCustomFoodItemRepository | constructor | backend/internal/repository/custom_food_repository.go:54-58 | added | repository tests and future callers | package coverage |
| 13 | GetByID | method | backend/internal/repository/custom_food_repository.go:60-78 | added | CustomFoodItemRepository consumers | CRUD and error-branch integration |
| 14 | Create | method | backend/internal/repository/custom_food_repository.go:80-106 | added | CustomFoodItemRepository consumers | CRUD and validation integration |
| 15 | Update | method | backend/internal/repository/custom_food_repository.go:108-135 | added | CustomFoodItemRepository consumers | CRUD and error-branch integration |
| 16 | Delete | method | backend/internal/repository/custom_food_repository.go:137-151 | added | CustomFoodItemRepository consumers | CRUD and error-branch integration |
| 17 | hydrateClassifications | method | backend/internal/repository/custom_food_repository.go:153-180 | added | GetByID | CRUD and error-branch integration |
| 18 | replaceClassifications | method | backend/internal/repository/custom_food_repository.go:182-196 | modified by repair | Create and Update | CRUD, error-branch, and immutability integration |
| 19 | validateCustomFoodIdentity | function | backend/internal/repository/custom_food_repository.go:198-207 | added | GetByID, Update, Delete | error-branch integration |
| 20 | testCustomFoodOwnerlessCreateSQL | embedded test SQL | backend/internal/repository/custom_food_repository_test.go:14-17 | added | direct ownerless insert test | CRUD integration |
| 21 | TestPostgresCustomFoodItemRepositoryOwnerScopedCRUD | integration test | backend/internal/repository/custom_food_repository_test.go:19-158 | added and repaired | test runner | direct acceptance evidence |
| 22 | TestPostgresCustomFoodItemRepositoryErrorBranches | integration/fault-injection test | backend/internal/repository/custom_food_repository_test.go:160-250 | added by repair | test runner | changed error and cleanup paths |
| 23 | TestReplaceCustomFoodClassificationsPreservesInputs | regression test | backend/internal/repository/custom_food_repository_test.go:252-273 | added by repair | test runner | slice aliasing regression |
| 24 | customFoodFixtureValues | test helper | backend/internal/repository/custom_food_repository_test.go:275-284 | added by repair | error-branch test | helper exercised |
| 25 | TestPostgresCustomFoodItemRepositoryValidation | integration test | backend/internal/repository/custom_food_repository_test.go:286-340 | added | test runner | invariant acceptance evidence |
| 26 | custom_food_items table definition | SQL DDL | database/migrations/000025_user_owned_custom_food_items.up.sql:2-36 | added | migration runner and repositories | migration cycle; integration |
| 27 | custom_food_items_owner_active_name_idx | SQL index | database/migrations/000025_user_owned_custom_food_items.up.sql:38-40 | added | PostgreSQL name uniqueness | normalized duplicate integration |
| 28 | custom_food_items_owner_idx | SQL index | database/migrations/000025_user_owned_custom_food_items.up.sql:42-44 | added | owner-scoped access | SQL and migration inspection |
| 29 | custom_food_item_classifications table | SQL DDL | database/migrations/000025_user_owned_custom_food_items.up.sql:46-51 | added | classification persistence | CRUD and IsInUse integration |
| 30 | custom_food_item_classifications_classification_idx | SQL index | database/migrations/000025_user_owned_custom_food_items.up.sql:53-54 | added | classification usage query | IsInUse integration |
| 31 | schema_migrations version 25 insert | migration bookkeeping | database/migrations/000025_user_owned_custom_food_items.up.sql:56-58 | added | migration runner | migration cycle |
| 32 | 000025 down migration | SQL migration | database/migrations/000025_user_owned_custom_food_items.down.sql:1-8 | added | migration rollback | migration cycle |
| 33 | classificationIsInUseSQL custom EXISTS branch | embedded SQL statement | backend/internal/repository/sql/classification_is_in_use.sql:1-4 | modified | ClassificationRepository.IsInUse | IsInUse integration |

~~~yaml
inventory_source_count: 33
audited_symbol_count: 33
inventory_complete: true
generated_groupings:
  - "None; every added or modified executable unit and persistence statement is listed separately. The task-list status-only row is excluded from executable inventory."
~~~

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| CustomFoodItemEntity | Embeds FoodItemEntity and carries mandatory OwnerID for private persistence. | Nil owner is rejected by repository; database enforces non-null. | N/A — data type only; no owned resource. | Owner is explicit in entity boundary. | N/A — no runtime allocation or I/O. | Minimal composition is idiomatic. | Ownerless API and direct-row tests pass. | PASS |
| CustomFoodItemRepository | Small owner-aware CRUD contract matching DESIGN-005. | Interface has no runtime branches; concrete methods audit all errors. | Context and transaction requirements are implemented by concrete methods. | Owner appears in read and delete signatures and entity update contract. | N/A — interface only. | Necessary, focused public API with compile assertion. | Future API callers are later-task scope. | PASS |
| customFoodCreateSQL | Inserts owner and all persisted food fields with bound values. | Constraints and conflicts are surfaced to Create. | N/A — statement text; transaction is caller-owned. | No interpolated input. | Single insert with bounded columns. | Separate embedded SQL follows repository convention. | Real create and placeholder scan pass. | PASS |
| customFoodGetByIDSQL | Selects only owner-matching row and applies include-deleted policy. | Missing or deleted rows produce not_found; boolean parameter is explicit. | Query context supports cancellation; hydration closes rows. | Owner predicate prevents cross-user disclosure. | Owner/id index supports lookup. | Simple static query. | Same-owner, wrong-owner, deleted, and include-deleted tests pass. | PASS |
| customFoodUpdateSQL | Updates only active row matching owner and ID. | Zero rows maps to not_found; constraints map through repository. | Executed in transaction with context; PostgreSQL serializes concurrent row updates. | Owner predicate prevents cross-owner mutation. | One indexed update. | No dynamic clauses or interpolation. | Same-owner, wrong-owner, and injected execution/rollback paths pass. | PASS |
| customFoodSoftDeleteSQL | Soft-deletes only active owner-owned row. | Repeated or wrong-owner delete produces not_found. | Context reaches one atomic Exec; no cleanup resource. | Owner predicate protects deletion. | One indexed update. | Minimal statement. | Same-owner, wrong-owner, and database-error paths pass. | PASS |
| customFoodClearClassificationsSQL | Clears assignments for one custom item. | Database errors are returned by replacement helper. | Runs inside Create or Update transaction; rollback protects partial writes. | ID is reached only through owner-scoped operation. | One delete bounded to one item. | Simple replacement primitive. | Clear-error and successful replacement paths pass. | PASS |
| customFoodAttachClassificationSQL | Adds a global classification assignment under composite-PK and FK constraints. | Duplicate or invalid assignment errors are observable. | Runs in transaction; partial writes roll back. | IDs are bound and FK-constrained. | One insert per caller-supplied classification. | Static parameterized statement. | Attach-error and normal category/role paths pass. | PASS |
| customFoodListClassificationsSQL | Hydrates active global classifications assigned to item. | Deleted classifications are excluded; scan and iteration errors are handled. | Query context and row close are handled by hydrateClassifications. | Data is reached from owner-scoped item read. | One join query with indexed assignment key. | Deterministic ordering. | Normal hydration, query, scan, rows.Err, and unknown-kind paths pass. | PASS |
| PostgresCustomFoodItemRepository | Holds transactional executor and isolates private table access. | Nil dependency is not a specified constructor error; use with nil would fail at I/O. | Shared pgx executor is concurrency-safe by contract; methods pass contexts and use transactions. | No public SQL bypass or global-table merge. | Stateless and constant-size repository. | Matches existing PostgreSQL repositories. | Real database and injected error tests pass. | PASS |
| CustomFoodItemRepository compile assertion | Enforces interface conformance at compile time. | Contract drift fails compilation. | N/A — compile-time only. | N/A — no data path. | N/A — no runtime cost. | Idiomatic Go guard. | Package tests compile. | PASS |
| NewPostgresCustomFoodItemRepository | Stores the supplied executor. | Nil is not validated because constructor contract assumes configured dependency. | Caller owns executor lifecycle; constructor owns none. | No user input handling. | One small allocation. | Consistent constructor pattern. | Package coverage and real construction pass. | PASS |
| GetByID | Validates identity, owner-scopes query, hydrates classes, converts units, and returns owner. | Validation, scan, mapping, hydration, and conversion paths are intentional. | Context reaches both database calls; hydration defers row close; no shared mutable state. | Wrong owner returns not_found without row data. | Two bounded queries and linear classification hydration. | Reuses shared scan and conversion helpers. | Real CRUD plus invalid row, query, and identity fault tests pass. | PASS |
| Create | Validates owner and food invariants, inserts, and atomically replaces classifications. | Validation, insert conflict/database, attach, rollback, and commit errors propagate. | withTransaction owns begin/rollback/commit; context reaches validation and transaction operations. | Owner is bound and database owner is mandatory. | One insert plus one clear and bounded assignment loop. | Reuses shared validation. | Valid, duplicate, normalized duplicate, different owner, ownerless, validation, and injected errors pass. | PASS |
| Update | Validates identity and food fields, updates active owner row, and replaces classifications atomically. | Validation, SQL, zero-row, attach, rollback, and commit paths are observable. | Transaction and context are used; concurrent writes rely on database row/transaction semantics. | Both owner and ID are bound. | One update plus bounded assignment replacement. | Consistent with Create and existing food repository. | Same-owner, wrong-owner, invalid identity, SQL error, rollback, and normal paths pass. | PASS |
| Delete | Validates identity and soft-deletes owner-owned active row. | Database failure, zero-row, and success are distinct. | Context reaches Exec; one statement is atomic and owns no resources. | Owner predicate protects deletion. | One indexed Exec. | Minimal method. | Same-owner, wrong-owner, invalid identity, and SQL error paths pass. | PASS |
| hydrateClassifications | Clears lists, scans rows, separates known kinds, checks rows.Err, and closes rows. | Query, scan, iteration, unknown-kind, and normal paths are intentional. | defer Close handles all post-query returns; context reaches Query; no shared state. | Returned IDs are global FK-backed classifications. | Linear assignment count and bounded per-item slices. | Clear switch and shared mapper. | Query, scan, rows.Err, unknown kind, and normal paths pass. | PASS |
| replaceClassifications | Clears then independently iterates categories and roles, avoiding caller-slice aliasing. | Clear, attach, empty, and normal paths return observable errors. | Caller transaction protects partial replacement; context propagates to every Exec. | IDs are bound and FK-constrained. | Linear in classifications; two-slice header allocation is constant. | Repair removes append aliasing and remains simple. | Clear/attach errors, empty inputs, normal replacement, and sentinel backing-array regression pass. | PASS |
| validateCustomFoodIdentity | Rejects nil owner or nil item ID before database access. | Both malformed cases return typed validation errors; valid UUIDs pass. | N/A — pure function. | Prevents ownerless or unidentified operations. | Constant time. | Small non-duplicative helper. | GetByID, Update, and Delete invalid identity paths pass. | PASS |
| testCustomFoodOwnerlessCreateSQL | Directly omits owner_id to test the database boundary. | NOT NULL failure is expected and asserted. | N/A — embedded fixture. | Verifies database enforcement beyond API validation. | One insert. | Appropriate negative fixture. | CRUD integration passes. | PASS |
| TestPostgresCustomFoodItemRepositoryOwnerScopedCRUD | End-to-end proof of CRUD, ownership, isolation, names, units, classes, deletion, and ownerless rejection. | Happy path and typed conflict/not-found/validation paths are asserted. | Real reset database and test cleanup are helper-owned; no leaked goroutines or shared mutable test state. | Cross-owner non-disclosure and curated isolation are direct assertions. | Bounded fixture and classification count. | Strong vertical integration scenario. | Mixed-case duplicate and prior adversarial paths are now covered. | PASS |
| TestPostgresCustomFoodItemRepositoryErrorBranches | Injects scan, query, iteration, identity, execution, and transaction replacement failures. | Every changed repository error branch asserted with typed error kind. | Rollback is explicitly asserted after attach failure. | No production boundary is bypassed except controlled test fakes. | Small bounded fake fixtures. | Focused fault tests avoid production test seams. | Commit and context cancellation are not separately injected, but production context propagation and aggregate/race checks pass; no uncovered changed line remains. | PASS |
| TestReplaceCustomFoodClassificationsPreservesInputs | Proves replacement does not mutate caller backing arrays. | Normal category and role inputs are checked after replacement. | N/A — synchronous helper test. | N/A — no trust boundary. | Constant small fixture. | Direct regression for repaired aliasing behavior. | Sentinel capacity case catches the prior append bug. | PASS |
| customFoodFixtureValues | Supplies correctly typed row values for fault-injection tests. | Covers valid row and deliberately malformed micros through caller tests. | N/A — pure test helper; time values are test-local. | N/A — test-only data. | Fixed-size slice allocation. | Keeps fault tests readable and avoids production helpers. | Exercised by ErrorBranches. | PASS |
| TestPostgresCustomFoodItemRepositoryValidation | Proves macros, active micros, liquid density/provenance, and conversion. | Table-driven typed validation cases cover negative macro, alias, inactive key, missing density, and missing provenance. | Rejected writes are isolated in real database test lifecycle. | Canonical vocabulary boundary is enforced. | Small bounded fixture. | Reuses existing validation contract. | Valid liquid round-trip and all invalid cases pass. | PASS |
| custom_food_items table definition | Requires owner, valid state, quantities, macros, JSON object micros, name, and liquid/solid density invariants. | Direct malformed writes are rejected by checks. | Owner FK cascade and timestamps are database-managed. | NOT NULL and FK prevent ownerless rows and invalid owners. | Constraints and indexes support access. | Dedicated table preserves curated isolation. | Migration cycle, repository validation, and ownerless direct SQL pass. | PASS |
| custom_food_items_owner_active_name_idx | Enforces lower(trimmed) active-name uniqueness per owner. | Normalized active duplicates conflict; tombstones are excluded. | PostgreSQL index is process-safe and race-safe. | Owner is uniqueness boundary. | Efficient unique lookup. | Correct generated-key partial index. | Exact, mixed-case/whitespace, different-owner, global, and post-delete tests pass. | PASS |
| custom_food_items_owner_idx | Supports active owner/id access. | N/A — index only. | Process-safe database structure. | Does not weaken owner predicate. | Supports scoped read/update/delete. | Appropriate supporting index. | Migration and SQL inspection pass. | PASS |
| custom_food_item_classifications table | Relates private items to global classifications with composite PK and FK policy. | Invalid and duplicate relationships are rejected by database constraints. | Item deletion cascades; classification deletion is restricted. | Parent item ownership protects access without redundant owner column. | Composite and reverse indexes support lookups. | Normal relational model. | CRUD and IsInUse paths pass. | PASS |
| custom_food_item_classifications_classification_idx | Supports reverse classification usage lookup. | N/A — index only. | Process-safe database structure. | No boundary change. | Efficient EXISTS lookup. | Minimal supporting index. | IsInUse integration passes. | PASS |
| schema_migrations version 25 insert | Records migration application idempotently. | ON CONFLICT handles repeated application. | Migration runner owns transaction. | N/A — bookkeeping. | Constant work. | Existing migration convention. | Aggregate migration cycle passes. | PASS |
| 000025 down migration | Reverses objects in dependency order. | IF EXISTS supports expected rollback state. | Migration runner owns transaction and cancellation. | Drops only Task 238 objects. | Bounded DDL/index work. | Correct reverse order. | Aggregate migration cycle passes. | PASS |
| classificationIsInUseSQL custom EXISTS branch | Treats custom assignments as classification usage. | Parameterized ID and EXISTS semantics are correct. | N/A — query text; caller context handles execution. | No interpolated data. | EXISTS and reverse index stop efficiently. | Small additive change. | Custom assignment IsInUse integration passes. | PASS |

## 7. Findings

No blocking, important, or optional finding remains after the repaired re-review. The prior status-gate, changed-symbol coverage, append-aliasing, and normalized-name findings were all independently verified as resolved.

~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 0
~~~

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| git rev-parse HEAD; git status; git diff/discovery commands | repository root | 0 | PASS | HEAD equals fixed baseline; current scope reconstructed. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 -v ./internal/repository -run 'TestPostgresCustomFoodItemRepository\|TestReplaceCustomFoodClassifications' | backend | 0 | PASS | Four Task 238 top-level tests and validation subtests passed. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/repository | backend | 0 | PASS | Full repository package passed. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race -count=1 -v ./internal/repository -run 'TestPostgresCustomFoodItemRepository\|TestReplaceCustomFoodClassifications' | backend | 0 | PASS | Focused race run passed. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/repository -coverprofile=/tmp/task-238-rereview.coverage.out | backend | 0 | PASS | Current coverage profile written. |
| go tool cover -func=/tmp/task-238-rereview.coverage.out | backend | 0 | PASS | All eight executable implementation symbols are 100.0%; repository total 93.6%. |
| gofmt -d internal/repository/custom_food_repository.go internal/repository/custom_food_repository_test.go | backend | 0 | PASS | No formatting diff. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./... | backend | 0 | PASS | No vet findings. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./... | backend | 0 | PASS | No vulnerabilities in called code; 18 uncalled module vulnerabilities reported by the tool. |
| GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./... | backend | 0 | PASS | All backend packages passed. |
| python3 scripts/validate-task-list.py | repository root | 0 | PASS | 263 sequential tasks; row 238 PREPARED. |
| python3 scripts/validate-traceability.py | repository root | 0 | PASS | Traceability passed. |
| git diff --check | repository root | 0 | PASS | No whitespace errors. |
| rg SQL interpolation scan over custom repository and SQL files | repository root | 0 | PASS | Placeholder-only matches; no dynamic SQL construction. |
| python3 scripts/check.py | repository root | 0 | PASS | Aggregate migration, readiness, backend, frontend, browser, traceability, and static checks passed; 237 browser tests passed and 3 were skipped by existing test markers. Existing documented Phase 07 coverage deviations and ignored OAuth 302-only warning were reported. |
| python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-238-review.md | repository root | 0 | PASS | Current review evidence passed structural validation. |

Required commands not run: none.

## 9. Files Inspected and Staleness Fingerprints

All hashes below are SHA256 values taken after the final review commands. The prior review evidence was checked and found stale against the repaired implementation; it was overwritten.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| docs/implementation/02_TASK_LIST.md | PREPARED gate and dependency status | none | SHA256 | f043e4a8c9cf2a5971566d132a8d36d6efa32535f27812ae9f7aae598cc3b4c6 |
| docs/design/DESIGN-005.md | source design contract | none | SHA256 | 91e9f1e152554e5d6eb62093018d57464ac3d38ca2add217215281927f885d31 |
| docs/implementation/preparations/task-238.md | preparation and repair scope | none | SHA256 | 1abbf808c82538353a7b28dca32daa3bda49877550e638fd4493fca0ab92b796 |
| backend/internal/repository/types.go | entity and interface | none | SHA256 | 05e6b89d355078c1d69a7d01a7ee7278ef977bc7e57a5b13541f4dc29e9ce41f |
| backend/internal/repository/custom_food_repository.go | implementation | none | SHA256 | 5adb40a45499c0c5b21fe295a87c3d325c28ac7cfb3f1a2fa90346c98055b472 |
| backend/internal/repository/custom_food_repository_test.go | integration and repair tests | none | SHA256 | 5d4b73d06ace7575d235469eeabf688d45f26ffa95bef929b7b5d66ba7a9e250 |
| backend/internal/repository/food_repository.go | shared validation and conversion | none | SHA256 | e37c943ee99bb260c5710a8738f21ea1c1709690ba40d610f91336816a24ade0 |
| backend/internal/repository/classification_repository.go | IsInUse consumer | none | SHA256 | 2d5073f1fe54d0e7a6a291bdbf9ee6e38531fe96ece7575d7fcf1c56325c0368 |
| backend/internal/repository/postgres.go | transaction and error mapping | none | SHA256 | ea59cfa009486ac4d37c620b8051005a925125d4d4ebdc4f63e506b9e84ef637 |
| backend/internal/repository/errors.go | typed error mapping dependency | none | SHA256 | 7545f079c82b24a650d72f7685f03c195f9efd5bbcf632363f0a4736634512d8 |
| backend/internal/repository/macros.go | macro and unit dependency | none | SHA256 | fe08f2fe0a693b99b413153ca190ec1db0e40e2140bf8d79cdfd86c186381af2 |
| backend/internal/repository/vocabulary_repository.go | active micronutrient dependency | none | SHA256 | c27715ce33cf4da3a3715f1d8019489dcdb5e81998321dfd2a9c1dd7d27153e6 |
| backend/internal/repository/postgres_repository_test.go | shared test fakes and helpers | none | SHA256 | 3db1938e60074ff110832e1e30ec8d4ded2fdd79045cc9ad5f79f4711d51c403 |
| backend/internal/testdatabase/testdatabase.go | integration database lifecycle | none | SHA256 | ec7bb939b7862e9db39274f8728cf9bfb27f19cca975d5803253f8329524414f |
| backend/internal/repository/sql/classification_is_in_use.sql | modified usage SQL | none | SHA256 | 2d381174a19601807474b77675b42da39e00b1c5ebed1e0f6cd2785e9edad1c5 |
| backend/internal/repository/sql/custom_food_attach_classification.sql | assignment SQL | none | SHA256 | 8870aa780d2b53fd5f01045460a86bac1487a3e2db8c0a597503ed2991e15a69 |
| backend/internal/repository/sql/custom_food_clear_classifications.sql | clear SQL | none | SHA256 | e593ef412c7aeb073af1448a31186fe6886ea529cc38109185eb8fcb417eb6f4 |
| backend/internal/repository/sql/custom_food_create.sql | create SQL | none | SHA256 | 8d5d14547a7b3c6364cd167378f8e5b28e4e21b113ce5292c35c1f7d3cd99857 |
| backend/internal/repository/sql/custom_food_get_by_id.sql | read SQL | none | SHA256 | 04ddfa9e5572e74cd8f0b89967460fb6207af04c7e0714aaf45471389f724c6c |
| backend/internal/repository/sql/custom_food_list_classifications.sql | hydration SQL | none | SHA256 | ebc16d07c6e51c181468935532a776407bdde334b781f83c503fd5fdac804617 |
| backend/internal/repository/sql/custom_food_soft_delete.sql | soft-delete SQL | none | SHA256 | 3a90aa2ff72a61aaff589ed695e824a0023bf2b48223cfc133eae8c1be0d95e1 |
| backend/internal/repository/sql/custom_food_update.sql | update SQL | none | SHA256 | 410fd9a0bd210557593d4c16c053bf21291359d4f438881079911861ab55079e |
| backend/internal/repository/sql/testdata/custom_food_ownerless_create.sql | ownerless negative fixture | none | SHA256 | fbf4b5fd21c7dfb9f6d1ff808a4ec116a0a3022290cb9bf74a064f4bad453081 |
| database/migrations/000002_food_items.up.sql | curated-table dependency | none | SHA256 | 13012e45e3e4b20d71e19a1c913d2f4b20f060a61d43f1cb5c1b53b8ccc160f6 |
| database/migrations/000003_classifications.up.sql | classification FK dependency | none | SHA256 | 768cc69f5a030b2e81ae9ac9715a233abfd0221247ed1e1fa887d1a93b099987 |
| database/migrations/000006_user_identity.up.sql | owner FK dependency | none | SHA256 | 90ff321007b8ee87371f1978b6a632213ad65a1a3470898e28ed07114f145cb7 |
| database/migrations/000011_liquid_density_constraint.up.sql | liquid invariant dependency | none | SHA256 | b27777e85809b7b144000b525f8fbddfc70fda8255b9c5a1ae5ccd16550b2878 |
| database/migrations/000025_user_owned_custom_food_items.up.sql | forward migration | none | SHA256 | dc3e479dd9ba72d39ceeb0a93c3d99b6097c4a862e5800f93ed3612d4f5c5091 |
| database/migrations/000025_user_owned_custom_food_items.down.sql | reverse migration | none | SHA256 | 8e9ba4712f5ee253bd2425557433cbbcebced083780dbce411cb7c5c3789091f |

~~~yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "docs/implementation/reviews/task-238-review.md was stale after the repair and was overwritten."
~~~

## 10. Coverage and Exceptions

- [x] Required coverage command ran.
- [x] Report path and observed threshold are recorded.
- [x] Untested branches relevant to changed symbols were inspected.
- [x] Exceptions exactly match the task row and are justified; Task 238 declares None and every changed executable implementation symbol is at 100.0% line coverage.

~~~yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "/tmp/task-238-rereview.coverage.out"
observed_line_coverage: "all eight executable symbols in custom_food_repository.go 100.0%; repository package total 93.6%"
coverage_passed: true
~~~

Coverage finding: No Task 238 coverage finding. Repository-package statements outside this task remain below 100% under existing documented Phase 07 exceptions, while every Task 238 executable implementation symbol is fully covered.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced.
- [x] No source-of-truth documentation was contradicted.
- [x] No generated/cache/build/temporary artifact was unintentionally added.
- [x] Public API additions are necessary and compile-checked; later API/export callers belong to later Phase 08 tasks.
- [x] Duplicate helpers and obsolete aliases were searched for.
- [x] Error, cleanup, timeout, concurrency, and malformed-input paths were challenged; changed executable paths are covered.

Findings: No Task 238 implementation finding. scripts/check.py reported only existing documented Phase 07 coverage deviations, an ignored OAuth 302-only OpenAPI warning, normal browser proxy errors for intentionally unavailable backend routes, and three explicitly skipped browser scenarios. These did not affect Task 238.

## 12. Decision

A task may be PASSED only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains.

Before accepting the decision, run:

~~~bash
python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-238-review.md
~~~

~~~yaml
decision: "PASSED"
reason: "PREPARED gate, dependencies, all 17 acceptance clauses, all 33 symbol audits, current hashes, full coverage of changed implementation symbols, and required validation commands pass with no blocking or important finding."
failed_criteria:
  - ""
failed_or_unaudited_symbols:
  - ""
recommended_next_action: "NONE"
~~~

## 13. Repair Context

Not applicable because the current decision is PASSED.
