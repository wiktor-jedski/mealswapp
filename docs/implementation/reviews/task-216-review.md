# Review Evidence: Task 216 — Durable Daily Diet Create Idempotency

```yaml
task_id: 216
component: "Daily Diet"
static_aspect: "Durable Daily Diet Create Idempotency"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-14T22:31:25Z"
review_agent: "Codex independent current-state task-216 review"
evidence_file: "docs/implementation/reviews/task-216-review.md"
baseline_ref: "a4e31367485b03269e90b5607f2057c9568bb5b1"
baseline_confidence: "HIGH"
code_review_skill_invoked: true
relevant_language_guide: "code-review-skill/reference/go.md; code-review-skill/reference/cross-cutting/sql-injection-prevention.md; code-review-skill/reference/cross-cutting/async-concurrency-patterns.md"
repair_context_required: true
```

## 1. Task Source

**Description:** Phase 07.01: replace checkout-named and permissive Daily Diet mutation claims with one typed persistence contract, persist and replay the immutable original create response, remove process-wide create serialization, and avoid duplicate meal lookups while preserving atomic same-key behavior across service instances.

**Depends On:** Tasks 214 and 215. Both are `PASSED` in the current task list.

**Testing Coverage Exceptions:** `None` in the task row.

**Verification Criteria:** Direct PostgreSQL and service tests prove first claim, exact replay after replacement/deletion/macro changes, changed-body conflict, concurrent same-key atomicity across service instances, independent-user progress, cancellation behind unrelated work, rollback at every write stage, account cascade, ownership-aware deletion, one exact decoded response shape with validated method/route/status/body hash/ID, rejection of malformed/dual/legacy bodies, and no duplicate per-meal lookups at the maximum entry count; no Daily Diet helper retains a checkout-specific name unless the storage is explicitly generalized; backend tests and `go test -race ./...` pass.

## 2. Pre-Review Gates

- [x] Input status is `PREPARED`.
- [x] Every dependency is `PREPARED` or `PASSED`.
- [x] The preparation report claims completion.
- [x] A task-specific baseline/diff is available and trustworthy.
- [x] `code-review-skill` was invoked exactly once and its relevant guide read.
- [x] The reviewer is independent from implementation/repair.
- [x] Review uses current repository state rather than stale logs.
- [x] Reviewer made no production-code changes.

```yaml
pre_review_gates_passed: true
blocking_issue: "NONE"
```

## 3. Review Baseline and Change Surface

Baseline/reference method: `git rev-parse HEAD` matched `a4e31367485b03269e90b5607f2057c9568bb5b1`. The dirty worktree contains concurrent Phase 07 work. Task-owned paths were reconstructed from the current baseline diff, the authoritative preparation manifest, untracked-file inspection, deleted-file baseline content, and symbol-level hunk ownership. The presence-aware repair was independently re-read in the current production decoder and repository tests.

Commands used to reconstruct the diff:

```bash
git rev-parse HEAD
git status --short
git diff --name-status HEAD -- <Task-216 tracked paths>
git diff --stat HEAD -- <Task-216 tracked paths>
git diff --unified=0 HEAD -- <Task-216 tracked paths>
git diff --no-index -- /dev/null <Task-216 untracked path>
git show HEAD:<deleted Task-216 SQL path>
git ls-files --others --exclude-standard
rg -n 'CreateWithIdempotency|AtomicDailyDietMutationResult|dailyDietIDFromIdempotencyResponse|checkout-specific Daily Diet symbols' backend docs database
```

Pre-existing dirty-worktree changes and exclusions:

`api/openapi.yaml`, unrelated backend packages, frontend files, scripts, `docs/design/DESIGN-005.md`, `docs/implementation/02_TASK_LIST.md`, `review.txt`, Task-215-only changes in shared files, and other task reviews/preparations were preserved and not treated as Task-216 implementation changes. The OpenAPI source, Daily Diet controller, controller tests, shared saved-diet helpers, PostgreSQL transaction helper, checkout repository, migration runner, DESIGN-007, and DESIGN-008 were inspected as callers, dependencies, or source contracts. `TestServiceListReturnsNotFoundWhenSavedMealIsUnavailable` is Task-215-only and was excluded from the Task-216 inventory. The only file overwritten by this review is this evidence file.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| `backend/internal/app/app.go` | baseline diff | HIGH | `NewProduction` Daily Diet wiring |
| `backend/internal/app/daily_diet_api_integration_test.go` | baseline diff plus immutable-replay assertions | HIGH | live API create/replay/concurrency test |
| `backend/internal/dailydiet/service.go` | baseline diff with Task-215 unit hunk excluded | HIGH | typed dependencies, create orchestration, projection mapper |
| `backend/internal/dailydiet/service_test.go` | baseline diff with Task-215-only test excluded | HIGH | memory claim fake and service tests |
| `backend/internal/repository/saved_diet_mutation_repository.go` | baseline diff plus presence-aware repair | HIGH | typed claim, scanner, decoder, validators, transaction |
| `backend/internal/repository/types.go` | baseline diff | HIGH | response, claim, result, interface contract |
| `backend/internal/repository/daily_diet_create_claim_test.go` | untracked task file plus presence-aware repair | HIGH | decoder, persisted-row, rollback, cascade tests and fixtures |
| `backend/internal/repository/sql/daily_diet_create_claim.sql` | untracked task file | HIGH | typed claim INSERT |
| `backend/internal/repository/sql/daily_diet_create_claim_get.sql` | untracked task file | HIGH | typed locked SELECT |
| `backend/internal/repository/sql/saved_diet_create_snapshot.sql` | untracked task file | HIGH | parent snapshot INSERT |
| `backend/internal/repository/sql/saved_diet_entry_insert_snapshot.sql` | untracked task file | HIGH | entry snapshot INSERT |
| `backend/internal/repository/sql/checkout_idempotency_get.sql` | baseline diff | HIGH | generalized shared-table checkout read |
| `backend/internal/repository/sql/checkout_idempotency_store.sql` | baseline diff | HIGH | generalized shared-table checkout write |
| `database/migrations/000021_mutation_idempotency.up.sql` | untracked task file | HIGH | shared-table upgrade rename |
| `database/migrations/000021_mutation_idempotency.down.sql` | untracked task file | HIGH | rollback rename/copy |
| `backend/internal/repository/sql/checkout_idempotency_claim.sql` | baseline deletion | HIGH | obsolete Daily Diet claim SQL |
| `backend/internal/repository/sql/checkout_idempotency_get_for_update.sql` | baseline deletion | HIGH | obsolete Daily Diet locked-read SQL |

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | First claim persists one immutable response and exact replay survives replacement, deletion, and meal-macro changes. | service, repository, live API tests and mapper/transaction inspection | PASS | `TestServiceCreateAggregatesMultipleMealsAndReplaysIdempotently`, direct PostgreSQL replay, and live API replay compare the original response after replacement, deletion, and macro mutation; replay maps the claim without reload. |
| 2 | Reusing a key with a changed body conflicts without another create. | service and PostgreSQL conflict assertions | PASS | Body hash comparison exists in both initial read and insert-race paths; service and repository tests assert conflict and unchanged create count. |
| 3 | Same-key creates are atomic across service and production instances. | fake concurrency test, live two-app test, unique-key and row-lock inspection | PASS | Two service instances and two production apps converge on one response ID and one persisted diet; `ON CONFLICT DO NOTHING` plus `FOR UPDATE` serializes contenders. |
| 4 | An unrelated user progresses and cancellation is honored while a create is blocked. | blocking meal fake and context-path inspection | PASS | The independent-user test completes within one second while another lookup blocks; cancellation reaches the blocking select and returns `context.Canceled`; no process mutex remains. |
| 5 | Claim, parent, entry, and saved-item failures roll back all writes. | direct PostgreSQL fault injection and residue queries | PASS | Claim-stage, parent-stage, entry-stage, and saved-item-stage failures all pass; the claim and all snapshot writes share `withTransaction`. |
| 6 | Account cascade and ownership-aware deletion remain correct. | FK migration, direct cascade, service/API ownership tests, SQL inspection | PASS | The renamed table retains the user cascade; `DeleteIfOwned` uses the user predicate and distinguishes foreign existence; direct, service, and live API tests pass. |
| 7 | Only the exact response shape and fixed method/route/status/hash/ID scope are accepted. | decoder, validator, scanner, SQL scope, OpenAPI source inspection | PASS | Strict unknown/trailing decoding, fixed `POST` `/daily-diets`, status `201`, 32-byte SHA-256 hash, nonnil IDs, entry correspondence, canonical units, timestamps, and domain validation are enforced. |
| 8 | Missing, null, and empty macro objects are rejected, explicit numeric zero is accepted, and malformed, dual-ID, or legacy bodies cause no writes. | pure decoder matrix and persisted JSONB read/residue assertions | PASS | `aggregateMacros` and all four members are pointer-checked at decoder lines 178-188; a complete all-zero macro object passes, while missing, `null`, `{}`, wrong type, unknown, trailing, malformed, legacy, dual, and domain-invalid bodies return internal; persisted tests verify the row remains and no diet is written. |
| 9 | Maximum-entry create performs one lookup per distinct meal. | instrumented 100-entry service test and loop inspection | PASS | The 100-entry test records one lookup for 100 repeated entries; `prepareCreate` caches by meal UUID and performs O(distinct meals) lookups. |
| 10 | Daily Diet no longer uses checkout-specific helpers unless shared storage is explicitly generalized. | repository-wide symbol search and migration inspection | PASS | Retired Daily Diet claim/replay helpers and SQL are removed; checkout consumers use the explicitly generalized `mutation_idempotency_keys` table. |
| 11 | Backend tests and race gate pass. | focused/full/race/vet/coverage/security/traceability commands | PASS | Every required command listed in Section 8 exited zero. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | `DailyDietCreateResponse` | behavioral type | `backend/internal/repository/types.go:340` | added | service and repository result | service and repository tests |
| 2 | `DailyDietCreateResponseEntry` | behavioral type | `backend/internal/repository/types.go:351` | added | service mapper and snapshot validator | service and repository tests |
| 3 | `DailyDietCreateResponseMacros` | behavioral type | `backend/internal/repository/types.go:361` | added | service mapper and decoder | service and repository tests |
| 4 | `DailyDietCreateClaim` | behavioral type | `backend/internal/repository/types.go:370` | added | service-to-repository boundary | repository rollback tests |
| 5 | `DailyDietCreateClaimResult` | behavioral type | `backend/internal/repository/types.go:381` | added | repository-to-service boundary | replay tests |
| 6 | `DailyDietMutationRepository` | interface contract | `backend/internal/repository/types.go:617` | modified | service, PostgreSQL implementation, memory fake | compile assertions and service tests |
| 7 | `dailyDietCreateMethod` and `dailyDietCreateRoute` | constants | `backend/internal/repository/saved_diet_mutation_repository.go:20` | added | all typed claim SQL calls | repository tests |
| 8 | `dailyDietCreateClaimSQL` | embedded SQL unit | `backend/internal/repository/saved_diet_mutation_repository.go:27` | added | `ClaimDailyDietCreate` | PostgreSQL tests |
| 9 | `dailyDietCreateClaimGetSQL` | embedded SQL unit | `backend/internal/repository/saved_diet_mutation_repository.go:32` | added | get and race paths | PostgreSQL tests |
| 10 | `savedDietCreateSnapshotSQL` | embedded SQL unit | `backend/internal/repository/saved_diet_mutation_repository.go:37` | added | snapshot helper | rollback tests |
| 11 | `savedDietEntryInsertSnapshotSQL` | embedded SQL unit | `backend/internal/repository/saved_diet_mutation_repository.go:42` | added | snapshot helper | rollback tests |
| 12 | `GetDailyDietCreateClaim` | repository method | `backend/internal/repository/saved_diet_mutation_repository.go:54` | added | `Service.Create` | replay, conflict, malformed tests |
| 13 | `ClaimDailyDietCreate` | repository method | `backend/internal/repository/saved_diet_mutation_repository.go:70` | added | `Service.Create` | claim, rollback, concurrency tests |
| 14 | `DeleteIfOwned` | repository method | `backend/internal/repository/saved_diet_mutation_repository.go:108` | re-audited in changed surface | `Service.Delete` | service/API ownership tests |
| 15 | `dailyDietCreateRecord` | internal type | `backend/internal/repository/saved_diet_mutation_repository.go:136` | added | scanner and claim methods | repository tests |
| 16 | `scanDailyDietCreateClaim` | scanner | `backend/internal/repository/saved_diet_mutation_repository.go:144` | added | get and claim methods | replay and malformed tests |
| 17 | `decodeDailyDietCreateResponse` | decoder | `backend/internal/repository/saved_diet_mutation_repository.go:168` | added and repaired | scanner | table-driven decoder test |
| 18 | `validateDailyDietCreateClaim` | validator | `backend/internal/repository/saved_diet_mutation_repository.go:197` | added | claim method | valid and rollback tests |
| 19 | `validateDailyDietCreateScope` | validator | `backend/internal/repository/saved_diet_mutation_repository.go:227` | added | get, claim, scanner | repository tests |
| 20 | `validateDailyDietCreateResponse` | validator | `backend/internal/repository/saved_diet_mutation_repository.go:243` | added | decoder and claim validator | malformed/domain tests |
| 21 | `createSavedDietSnapshot` | transaction helper | `backend/internal/repository/saved_diet_mutation_repository.go:268` | added | claim method | rollback tests |
| 22 | `Service` durable dependencies | behavioral type | `backend/internal/dailydiet/service.go:98` | modified | controller and app composition | service/API tests |
| 23 | `NewService` | constructor | `backend/internal/dailydiet/service.go:105` | modified | `NewProduction` and tests | compile and service tests |
| 24 | `Service.Create` | service method | `backend/internal/dailydiet/service.go:111` | modified | `ProfileController.CreateDailyDiet` | service/API tests |
| 25 | `prepareCreate` | service helper | `backend/internal/dailydiet/service.go:267` | added | `Service.Create` | aggregation/lookup tests |
| 26 | `createResultFromClaim` | service mapper | `backend/internal/dailydiet/service.go:304` | added | `Service.Create` | replay tests |
| 27 | `NewProduction` Daily Diet wiring | application wiring | `backend/internal/app/app.go:69-73` | modified | production composition | live API test |
| 28 | `memoryDietRepository` | test fake | `backend/internal/dailydiet/service_test.go:16` | modified | service tests | service suite |
| 29 | `memoryDailyDietClaim` | test type | `backend/internal/dailydiet/service_test.go:25` | added | memory claim fake | service suite |
| 30 | memory `GetDailyDietCreateClaim` | test fake method | `backend/internal/dailydiet/service_test.go:46` | added | `Service.Create` | replay/conflict tests |
| 31 | memory `ClaimDailyDietCreate` | test fake method | `backend/internal/dailydiet/service_test.go:61` | added | `Service.Create` | atomicity test |
| 32 | `memoryMealRepository` | test fake | `backend/internal/dailydiet/service_test.go:141` | modified | service tests | service suite |
| 33 | memory `GetByID` | test fake method | `backend/internal/dailydiet/service_test.go:150` | modified | prepare/cancel paths | lookup/cancel tests |
| 34 | `TestServiceCreateAggregatesMultipleMealsAndReplaysIdempotently` | service test | `backend/internal/dailydiet/service_test.go:189` | modified | create contract | focused service test |
| 35 | `TestServiceCreateLooksUpEachDistinctMealOnceAtMaximumEntries` | service test | `backend/internal/dailydiet/service_test.go:268` | added | lookup contract | focused service test |
| 36 | `TestServiceCreateSameKeyIsAtomicAcrossInstances` | service test | `backend/internal/dailydiet/service_test.go:285` | added | same-key contract | race-tested service test |
| 37 | `TestServiceCreateDoesNotBlockIndependentUsersAndHonorsCancellation` | service test | `backend/internal/dailydiet/service_test.go:327` | added | concurrency contract | race-tested service test |
| 38 | `TestServiceRejectsMissingMealsBeforeWritesAndScopesOwnership` | service test | `backend/internal/dailydiet/service_test.go:370` | modified | create/delete behavior | service suite |
| 39 | `TestServiceValidationRejectsInvalidInputs` | service test | `backend/internal/dailydiet/service_test.go:418` | modified | constructor migration and validation | service suite |
| 40 | `TestDailyDietProductionAPIWithLivePostgres` | integration test | `backend/internal/app/daily_diet_api_integration_test.go:32` | modified | live controller/repository stack | live PostgreSQL test |
| 41 | `TestDecodeDailyDietCreateResponseRejectsInvalidPersistedBodies` | repository test | `backend/internal/repository/daily_diet_create_claim_test.go:17` | added and repaired | decoder | table-driven decoder test |
| 42 | `TestPostgresDailyDietCreateClaimReplaysImmutableResponseAndCascades` | repository test | `backend/internal/repository/daily_diet_create_claim_test.go:70` | added | repository contract | focused PostgreSQL test |
| 43 | `TestPostgresDailyDietCreateClaimRejectsLegacyDualAndRollsBackWrites` | repository test | `backend/internal/repository/daily_diet_create_claim_test.go:110` | added and repaired | decoder and transaction | focused PostgreSQL test |
| 44 | `staticDailyDietCreateResponsePayload` | test fixture helper | `backend/internal/repository/daily_diet_create_claim_test.go:195` | added | decoder tests | decoder test |
| 45 | `mutatedDailyDietCreateResponsePayload` | test fixture helper | `backend/internal/repository/daily_diet_create_claim_test.go:199` | added | decoder tests | decoder test |
| 46 | `validDailyDietCreateResponsePayload` | test fixture helper | `backend/internal/repository/daily_diet_create_claim_test.go:215` | added | decoder tests | decoder test |
| 47 | `responseEntry` | test fixture helper | `backend/internal/repository/daily_diet_create_claim_test.go:224` | added | malformed entry cases | decoder test |
| 48 | `responseMacros` | test fixture helper | `backend/internal/repository/daily_diet_create_claim_test.go:228` | added | malformed macro cases | decoder test |
| 49 | `testDailyDietCreateClaim` | test fixture | `backend/internal/repository/daily_diet_create_claim_test.go:232` | added | claim/replay/rollback tests | repository suite |
| 50 | `assertNoDailyDietClaimWrites` | test assertion helper | `backend/internal/repository/daily_diet_create_claim_test.go:241` | added | rollback and persisted-row tests | repository suite |
| 51 | daily-diet claim INSERT | SQL statement | `backend/internal/repository/sql/daily_diet_create_claim.sql:2-5` | added | claim method | PostgreSQL tests |
| 52 | daily-diet claim locked SELECT | SQL statement | `backend/internal/repository/sql/daily_diet_create_claim_get.sql:2-5` | added | get and race paths | PostgreSQL tests |
| 53 | saved-diet parent snapshot INSERT | SQL statement | `backend/internal/repository/sql/saved_diet_create_snapshot.sql:2-3` | added | snapshot helper | rollback tests |
| 54 | saved-diet entry snapshot INSERT | SQL statement | `backend/internal/repository/sql/saved_diet_entry_insert_snapshot.sql:2-3` | added | snapshot helper | rollback tests |
| 55 | checkout idempotency SELECT | SQL statement | `backend/internal/repository/sql/checkout_idempotency_get.sql:2-4` | modified | checkout repository | full backend suite |
| 56 | checkout idempotency INSERT | SQL statement | `backend/internal/repository/sql/checkout_idempotency_store.sql:2-3` | modified | checkout repository | full backend suite |
| 57 | mutation idempotency up migration | migration logic | `database/migrations/000021_mutation_idempotency.up.sql:2-10` | added | migration runner and both consumers | migration/integration tests |
| 58 | mutation idempotency down migration | migration logic | `database/migrations/000021_mutation_idempotency.down.sql:2-18` | added | migration runner | migration/integration tests |
| 59 | obsolete checkout claim SQL | deleted SQL unit | `backend/internal/repository/sql/checkout_idempotency_claim.sql` | deleted | no current consumer | removed-symbol search |
| 60 | obsolete checkout locked-read SQL | deleted SQL unit | `backend/internal/repository/sql/checkout_idempotency_get_for_update.sql` | deleted | no current consumer | removed-symbol search |

```yaml
inventory_source_count: 60
audited_symbol_count: 60
inventory_complete: true
generated_groupings:
  - "None; each changed production, SQL, migration, test, and deleted SQL unit is listed separately."
```

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| `DailyDietCreateResponse` | Typed immutable projection with required identity, entries, macros, and timestamps. | Zero macro values are valid; missing fields are rejected by the decoder before this value is trusted. | Value type; no resources. | Only server projection fields cross the replay boundary. | Entries are bounded by validator. | Minimal typed replacement for generic JSON. | Service and repository replay plus malformed tests. | PASS |
| `DailyDietCreateResponseEntry` | Carries entry/meal IDs, positive quantity, canonical unit, and position. | Decoder and validator reject bad IDs, quantity, unit, position, and duplicates. | Value type; no shared state. | No direct SQL construction. | One value per bounded entry. | Explicit JSON contract. | Matrix covers invalid entry fields. | PASS |
| `DailyDietCreateResponseMacros` | Four nonnegative finite aggregate values, with zero preserved. | Presence is enforced through pointer fields in `decodeDailyDietCreateResponse`; negative and nonfinite values fail validation. | Value type; no resources. | Persisted macro JSON is not trusted until presence and domain checks pass. | Fixed four-value check. | Clear typed shape with a narrow presence-aware decoder seam. | Explicit all-zero success and missing/null/empty failures pass. | PASS |
| `DailyDietCreateClaim` | Couples user, key, SHA-256 body hash, persisted diet, response, and 201 status. | Claim validation handles scope, status, identity, entries, response, and all error returns before transaction. | Passed to one transaction; no goroutine state. | User ID and SQL values are bound parameters. | At most 100 entries. | Narrow typed boundary. | Valid and all write-stage rollback fixtures pass. | PASS |
| `DailyDietCreateClaimResult` | Returns stored response/status and replay marker. | First and replay branches map deliberately. | Immutable value result. | PostgreSQL results are scanner-validated. | Bounded response copy. | Minimal result surface. | Service and repository replay tests pass. | PASS |
| `DailyDietMutationRepository` | Extends CRUD with typed get/claim and ownership-aware delete. | Compile-time implementation and fake assertions catch contract drift. | Cross-instance coordination is implementation-owned. | User-scoped method signatures. | No extra abstraction cost. | Necessary focused interface. | Service and full backend tests pass. | PASS |
| `dailyDietCreateMethod` and `dailyDietCreateRoute` | Fixed `POST` and `/daily-diets` scope. | Scanner rejects persisted method, route, or status drift. | Constants eliminate caller-selected scope. | Prevents cross-route replay. | No I/O. | Simple canonical boundary. | Direct claim and live API tests pass. | PASS |
| `dailyDietCreateClaimSQL` | Inserts first claim and returns it; unique conflict returns no row. | DB errors map through scanner and conflict proceeds to locked read. | Runs in transaction; unique key serializes instances. | Fully parameterized. | One indexed insert. | Correct PostgreSQL pattern. | Rollback and concurrency tests pass. | PASS |
| `dailyDietCreateClaimGetSQL` | Reads exact claim scope with `FOR UPDATE`. | Missing row maps not-found; malformed row maps internal. | Existing row is locked in the claim transaction; context reaches query. | Exact user/method/route/key predicates. | One indexed read and bounded decode. | Minimal SQL. | Replay, concurrency, and malformed tests pass. | PASS |
| `savedDietCreateSnapshotSQL` | Inserts explicit response-identical parent identity and timestamps. | Constraint errors propagate to transaction rollback. | Transaction-owned; no leaked resource. | User ID is bound. | One insert. | Dedicated colocated SQL. | Parent-stage rollback passes. | PASS |
| `savedDietEntryInsertSnapshotSQL` | Inserts explicit entry identity and persisted values. | FK and constraint errors propagate. | Runs inside parent transaction. | IDs and values are parameterized. | At most 100 inserts. | Dedicated SQL avoids inline statements. | Entry-stage rollback passes. | PASS |
| `GetDailyDietCreateClaim` | Returns only an exact body-hash match and canonical stored response. | Validates caller scope; not-found, conflict, and internal paths are intentional. | Query context is propagated; read has no write side effect. | User, key, and hash are validated and parameterized. | One query plus bounded decode. | Typed API. | Replay, conflict, and persisted malformed/no-write tests pass. | PASS |
| `ClaimDailyDietCreate` | First claimant atomically persists claim plus snapshot; loser replays exact row. | Validation, insert, scan, snapshot, conflict, and commit errors return. | Transaction plus unique key and `FOR UPDATE` handles process instances; cancellation reaches DB. | No SQL interpolation; persisted response is validated before use. | Bounded entries, one claim operation, and up to 100 entry writes. | Clear first-or-replay flow. | Full, race, rollback, and live tests pass. | PASS |
| `DeleteIfOwned` | Deletes only a user-owned diet and reports foreign existence. | Nil IDs validate; delete, existence, query, and commit failures return. | Transaction cleanup is delegated to shared helper. | User predicate prevents cross-account deletion. | Delete plus one existence query on miss. | Existing minimal API retained. | Ownership and cascade tests pass. | PASS |
| `dailyDietCreateRecord` | Private validated row carries hash, status, and response. | Constructed only after scanner checks. | Value type. | Scope is not exposed before validation. | Small decoded value. | Correctly private. | Repository tests pass. | PASS |
| `scanDailyDietCreateClaim` | Scans row and enforces fixed scope/status, response shape, and SHA-256 scope. | DB errors map; invalid method, route, status, payload, or scope become internal. | Row owns no resources; caller controls transaction. | Database row is treated as untrusted input. | One JSON decode and one bounded validator pass. | Centralized trust-boundary gate. | Repaired matrix covers malformed, domain, and macro-presence branches. | PASS |
| `decodeDailyDietCreateResponse` | Accepts one exact JSON object with no unknown or trailing data and valid domain values. | `DisallowUnknownFields` and the second decode reject malformed forms; pointer members reject missing, null, and empty macros while explicit zero pointers pass. | Pure function; no cancellation or resource issue. | Prevents persisted JSON from becoming trusted API state. | Fixed-field presence check plus O(entries) validation. | Strict decoder is centralized and idiomatic. | Pure matrix and persisted JSONB tests pass for zero, missing, null, empty, wrong type, legacy, dual, and domain cases. | PASS |
| `validateDailyDietCreateClaim` | Ensures request scope, 201, diet identity/timestamps, response validity, and entry correspondence. | Rejects mismatched IDs, timestamps, lengths, fields, and invalid domain values. | Pure validation before transaction. | User and resource identity are checked. | O(entries), bounded by service input. | Single claim invariant. | Valid and rollback tests pass. | PASS |
| `validateDailyDietCreateScope` | Requires nonnil user, bounded key, and 32-byte hex body hash. | Rejects malformed hash, user, and key; service trims caller keys before use. | Pure function. | Prevents invalid persisted scope and key misuse. | Constant-size hash decode. | Small helper. | Normal, invalid-input, and conflict tests pass. | PASS |
| `validateDailyDietCreateResponse` | Enforces IDs, nonblank name, timestamps, 1-100 entries, canonical values, unique positions, and nonnegative finite macros. | Handles malformed/domain values; required macro-object presence is enforced one layer earlier, preserving explicit zero. | Pure O(entries) validation and bounded map. | Untrusted persisted JSON crosses into replay only after both checks. | Linear and bounded for valid rows. | Single response-domain validator. | Matrix covers negative, zero, wrong-type, ID, quantity, unit, and position cases. | PASS |
| `createSavedDietSnapshot` | Writes parent, entries, and saved-item index matching the response inside claim transaction. | Any stage error maps and aborts transaction. | No goroutines; rollback covers partial writes. | All SQL inputs are bound and prevalidated. | At most 102 writes including index. | Clear helper. | Parent, entry, and saved-item rollback passes. | PASS |
| `Service` durable dependencies | Holds only typed diet mutation and meal repositories; no process-wide mutex or checkout dependency. | Nil dependencies fail closed in service methods. | Removes cross-user lock contention; repository supplies cross-process coordination. | Service passes authenticated user ID. | No unnecessary dependency. | Simpler API. | Constructor and concurrency tests pass. | PASS |
| `NewService` | Constructs saved-diet behavior with exactly the two required dependencies. | No hidden global state. | No resource ownership. | Dependency boundary is explicit. | No unnecessary work. | Idiomatic constructor. | App and service suites compile and pass. | PASS |
| `Service.Create` | Validates request, replays exact claim, prepares projection, and atomically claims first create. | Validation, not-found, conflict, repository, meal, and claim errors are intentional; conflict maps to service sentinel. | Context passes through reads, meal lookup, and transaction; no process mutex. | User-scoped claim and no client-owned response fields. | One pre-read plus one lookup per distinct meal. | Correct orchestration. | Service, live, and race tests pass. | PASS |
| `prepareCreate` | Builds one immutable diet and response using distinct meal projections. | Meal and physical-unit errors return before writes; claim validation rejects invalid computed values. | No shared mutable state; context reaches each lookup. | Meal IDs are input but bound only later. | O(entries), O(distinct meals), max 100. | Reuses projection rather than reloading. | 100-entry lookup and aggregate tests pass. | PASS |
| `createResultFromClaim` | Maps persisted typed response into API-safe result without reload. | Copies entries, macros, timestamps, status, and replay marker. | Pure value conversion. | Only repository-validated response is exposed. | O(entries), bounded. | No duplicate decoder or resource lookup. | Exact replay tests pass. | PASS |
| `NewProduction` Daily Diet wiring | Uses saved repository and meal repository with the new constructor. | Checkout repository remains wired for optimization/subscription consumers. | No Daily Diet process mutex/dependency. | Composition preserves route boundary. | No additional runtime work. | Correct dependency graph. | Live API and full suite pass. | PASS |
| `memoryDietRepository` | Test fake models typed claims and diets. | Missing, conflict, replay, and CRUD paths are represented. | Claim map and create count are mutex-protected for concurrent tests. | User/key composite scope. | O(1) fake operations. | Faithful focused seam. | Service suite and race gate pass. | PASS |
| `memoryDailyDietClaim` | Stores hash and immutable claim result. | Changed hash conflicts. | Value held under fake mutex. | User scope is outer key. | Small value. | Minimal fake state. | Replay/conflict tests pass. | PASS |
| memory `GetDailyDietCreateClaim` | Replays exact fake result and marks replay. | Missing and changed-body paths are intentional. | Mutex is released with defer. | User/key-scoped lookup. | O(1). | Matches repository contract. | Replay/conflict tests pass. | PASS |
| memory `ClaimDailyDietCreate` | First fake claimant stores once; same key replays. | Existing hash conflict and exact replay handled. | Mutex makes two service instances atomic. | User/key-scoped map key. | O(1). | Appropriate test double. | Cross-instance service test passes. | PASS |
| `memoryMealRepository` | Test fake exposes meal data, lookup counters, and cancellable blocking. | Missing meal and cancellation paths represented. | Mutex protects counters; blocking select honors context. | Test-only IDs. | Counter map reveals duplicate work. | Narrow useful seam. | Lookup and cancellation tests pass. | PASS |
| memory `GetByID` | Returns meal or not-found and counts actual calls. | Context cancellation returns `ctx.Err()` while blocked. | No leaked test goroutine; mutex protects counter. | No SQL boundary. | O(1) lookup. | Faithful fake. | Maximum-entry and cancellation tests pass. | PASS |
| `TestServiceCreateAggregatesMultipleMealsAndReplaysIdempotently` | Covers first result, aggregate math, CRUD, exact replay, conflict, deletion, and macro mutation. | Replayed result remains equal after resource deletion and macro change. | No mutable reload on replay is observable. | User scope exercised. | Checks create count. | Strong regression. | Service test passes; repository test owns malformed persistence. | PASS |
| `TestServiceCreateLooksUpEachDistinctMealOnceAtMaximumEntries` | Exercises 100 entries sharing one meal and asserts one lookup. | Maximum cardinality and unique positions are valid. | Counter read is locked. | No external trust boundary. | Direct O(distinct) assertion. | Focused deterministic test. | Test passes. | PASS |
| `TestServiceCreateSameKeyIsAtomicAcrossInstances` | Two service instances converge on one response ID and one create. | Both errors and IDs are checked. | Two bounded goroutines and mutex-backed fake. | Same user/key scope. | Minimal concurrent fixture. | Direct regression. | Test and race suite pass. | PASS |
| `TestServiceCreateDoesNotBlockIndependentUsersAndHonorsCancellation` | Unrelated create progresses while one meal lookup blocks and cancellation settles. | Timeout assertions catch blocking and wrong cancellation. | Blocking fake selects on context; no production goroutine leaks. | User separation exercised. | Two bounded operations. | Direct concurrency regression. | Test and race suite pass. | PASS |
| `TestServiceRejectsMissingMealsBeforeWritesAndScopesOwnership` | Missing meals do not write and foreign users cannot read/replace/delete. | Not-found and mutation counts checked. | Sequential fake use. | Ownership boundary exercised. | Small fixture. | Preserves existing coverage. | Test and full suite pass. | PASS |
| `TestServiceValidationRejectsInvalidInputs` | Existing key and cross-basis validation cases compile against typed service. | Missing key, short key, and wrong physical basis reject. | No resources. | Input validation remains before write. | Small table. | No duplicated fake dependency. | Test passes. | PASS |
| `TestDailyDietProductionAPIWithLivePostgres` | End-to-end authenticated create/replay/conflict/delete/cascade/concurrency behavior. | Replay after deletion and macro mutation is exact and does not recreate. | Two production apps exercise shared DB boundary. | Auth, CSRF, and ownership routes included. | Live integration cost bounded. | High-value system test. | Full app and race suites pass. | PASS |
| `TestDecodeDailyDietCreateResponseRejectsInvalidPersistedBodies` | Table-driven decoder regression for the persisted response contract. | Explicit all-zero macros pass; missing, null, empty, wrong-type, unknown, trailing, malformed, ID, domain, legacy, and dual cases reject. | Pure test. | Exercises persisted-row trust boundary. | Small fixtures. | Focused adversarial matrix. | Focused repository test passes. | PASS |
| `TestPostgresDailyDietCreateClaimReplaysImmutableResponseAndCascades` | Direct first claim, exact replay after mutation/deletion, conflict, and account cascade. | Errors and exact equality checked. | Real PostgreSQL transactions and FK cascade. | User-scoped key and deletion. | Small fixture. | Strong persistence regression. | Focused repository test passes. | PASS |
| `TestPostgresDailyDietCreateClaimRejectsLegacyDualAndRollsBackWrites` | Direct persisted-body rejection plus each write-stage rollback. | Legacy, dual, macro-presence, and representative invalid rows return internal; residue is checked. | Real transaction rollback assertions. | Invalid DB JSON is treated internal. | Deterministic bounded fixture. | Good persistence coverage. | Focused repository test passes. | PASS |
| `staticDailyDietCreateResponsePayload` | Supplies exact malformed/trailing/legacy bytes without transformation. | Preserves fixture bytes. | Pure helper. | Test-only. | No meaningful allocation risk. | Simple helper. | Used by decoder matrix. | PASS |
| `mutatedDailyDietCreateResponsePayload` | Starts from valid JSON and applies one controlled mutation. | Marshal errors fail test setup. | Pure helper. | Test-only representation of persisted payload. | Small bounded fixture. | Reusable. | Supports parent-field and macro-presence mutations. | PASS |
| `validDailyDietCreateResponsePayload` | Produces one valid canonical response body. | Fixture encoding failure fails the test. | Pure helper. | Test-only. | Small. | Reusable baseline. | Used by mutation cases. | PASS |
| `responseEntry` | Locates the valid first entry for mutations. | Panics only on a broken test baseline, correctly failing setup. | Pure helper. | Test-only. | O(1). | Minimal. | Used by entry matrix. | PASS |
| `responseMacros` | Locates aggregate macro map for numeric mutations. | Panics only on a broken valid fixture; parent missing/null cases mutate the parent directly. | Pure helper. | Test-only. | O(1). | Minimal. | Numeric and presence cases are covered. | PASS |
| `testDailyDietCreateClaim` | Builds internally consistent fixed-time valid claim. | IDs, timestamps, response, entries, status, and hash align. | Pure fixture. | Test-only. | One entry. | Stable reusable fixture. | Claim, replay, and rollback tests pass. | PASS |
| `assertNoDailyDietClaimWrites` | Asserts no claim or parent residue after failed claim. | Query errors fail; zero counts required. | Pure DB assertions. | User/key-scoped checks. | Two indexed counts. | Clear rollback helper. | Rollback tests pass. | PASS |
| daily-diet claim INSERT | Parameterized shared-table insert with unique no-op. | Conflict and return/no-row branches are handled by caller. | Transactional. | No injection or dynamic identifiers. | Indexed and bounded. | Correct SQL idiom. | PostgreSQL and live tests pass. | PASS |
| daily-diet claim locked SELECT | Parameterized exact scope with row lock. | Missing and malformed rows map through scanner. | Row lock supports contenders. | No dynamic identifiers. | Indexed read. | Minimal SQL. | PostgreSQL and live tests pass. | PASS |
| saved-diet parent snapshot INSERT | Explicit parent identity and timestamps. | Constraint error reaches rollback. | Transactional. | Parameterized IDs. | One insert. | Minimal colocated SQL. | Parent rollback passes. | PASS |
| saved-diet entry snapshot INSERT | Explicit entry identity and value insert. | FK and constraint error reaches rollback. | Transactional. | Parameterized fields. | At most 100 inserts. | Minimal colocated SQL. | Entry rollback passes. | PASS |
| checkout idempotency SELECT | Existing checkout consumer reads the renamed shared table. | Existing scope and scan behavior remain intact. | No Daily Diet dependency. | Parameterized. | One indexed read. | Checkout name remains valid at consumer boundary. | Full backend suite passes. | PASS |
| checkout idempotency INSERT | Existing checkout consumer writes the renamed shared table. | Existing validation and conflict behavior remain intact. | No Daily Diet dependency. | Parameterized. | One indexed write. | Explicit shared storage. | Full backend suite passes. | PASS |
| mutation idempotency up migration | Renames old table when old exists and new does not, then records version. | Guard avoids duplicate rename; migration runner controls order. | Schema operation is atomic per migration execution. | No user data interpolation. | Constant-time rename. | Explicit generalized storage. | Full migration reset/up passes. | PASS |
| mutation idempotency down migration | Renames back or copies into an existing checkout table and drops generic table. | Both table-presence branches are guarded; copy is conflict-safe. | Migration runner controls order. | Schema-only. | Copy is row-bounded. | Explicit rollback path. | Fresh migration paths pass; populated dual-table branch is inspected but not directly exercised. | PASS |
| obsolete checkout claim SQL | Retired Daily Diet claim statement is removed. | N/A — deleted. | N/A — deleted. | N/A — deleted. | N/A — deleted. | Removes stale consumer surface. | Repository search finds no current reference. | PASS |
| obsolete checkout locked-read SQL | Retired Daily Diet locked-read statement is removed. | N/A — deleted. | N/A — deleted. | N/A — deleted. | N/A — deleted. | Removes stale consumer surface. | Repository search finds no current reference. | PASS |

Mandatory cross-cutting conclusions: nil, empty, and boundary values were inspected for all nontrivial units; every repository error and transaction path was traced; DB queries are parameterized; no production goroutine is introduced; context reaches meal and database waits; unique-key and row-lock coordination is valid across process instances; writes are bounded by 100 entries; no new public API is unnecessary; duplicate helpers and obsolete aliases were searched; and the prior presence-aware response-shape defect is closed by current code and tests.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| optional | `backend/internal/repository/sql/saved_diet_create_with_id.sql:1-4` | stale explicit-ID SQL artifact | The retired explicit-ID SQL remains in the repository without a current embed or caller. | `rg` finds no current consumer; the file is unchanged from the task baseline and is not a runtime defect. | Delete in a cleanup task or document/register its deliberate consumer. Not a decision blocker. |

```yaml
blocking_findings: 0
important_findings: 0
optional_findings: 1
```

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| `git diff --check` | repository root | 0 | PASS | no whitespace errors |
| `gofmt -d` on current Task-216 Go implementation/test files | repository root | 0 | PASS | no formatting diff |
| current task-owned path reconstruction with `git diff --name-status`, `git diff --stat`, untracked inspection, deleted baseline inspection, and removed-symbol `rg` search | repository root | 0 | PASS | 17 task-owned implementation/test/SQL/migration paths reconstructed; unrelated worktree paths excluded |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/repository -run 'TestDecodeDailyDietCreateResponseRejectsInvalidPersistedBodies|TestPostgresDailyDietCreateClaimReplaysImmutableResponseAndCascades|TestPostgresDailyDietCreateClaimRejectsLegacyDualAndRollsBackWrites' -count=1` | `backend` | 0 | PASS | focused decoder, replay, malformed-row, and rollback tests passed |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/dailydiet ./internal/app ./internal/httpapi -count=1` | `backend` | 0 | PASS | service, live API, and controller caller packages passed |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -count=1 -p=1` | `backend` | 0 | PASS | full backend suite passed |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1` | `backend` | 0 | PASS | full race suite passed, including service and repository concurrency tests |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | `backend` | 0 | PASS | vet passed |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/... -coverprofile=coverage.out && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go tool cover -func=coverage.out` | `backend` | 0 | PASS | report `backend/coverage.out`; total 88.0%, internal/repository 93.4%, Daily Diet decoder and response validator 100% |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | `backend` | 0 | PASS | no reachable vulnerabilities found |
| `python3 scripts/validate-task-list.py` | repository root | 0 | PASS | 237 sequential tasks/dependencies validated |
| `python3 scripts/validate-traceability.py` | repository root | 0 | PASS | traceability validation passed |
| `npx --no-install redocly lint api/openapi.yaml` | repository root | 0 | PASS | API valid; one existing ignored OAuth callback 302-only warning |
| `sha256sum` over all current reviewed implementation, test, SQL, migration, caller, and source-contract files plus `git show HEAD:<deleted SQL> | sha256sum` | repository root | 0 | PASS | hashes recorded in Section 9 |
| `python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-216-review.md` | repository root | 0 | PASS | structural evidence validation passed |

No required command was skipped. The coverage command is recorded as a gate and measurement; the task row has no numeric coverage threshold, and the repository’s accepted Phase 07 coverage disposition remains recorded in `docs/implementation/04_OPEN.md`.

## 9. Files Inspected and Staleness Fingerprints

Hashes are SHA-256 of current contents after the audit. Deleted SQL files use SHA-256 of their baseline contents. Non-task files in this table are caller, dependency, or source-contract files inspected to validate the changed surface.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| `api/openapi.yaml` | persisted/API response source contract | no finding | SHA-256 | `6368ee9c1321104d0e645ed8a3e6b73f8f14c1a161835a5f388a0e6e2fa3da4a` |
| `backend/internal/app/app.go` | production wiring | no finding | SHA-256 | `bf4b26213e9c3e6ce856d9793c980152975e178f86a1da74367f93d5a68d2066` |
| `backend/internal/app/daily_diet_api_integration_test.go` | live API caller/test | no finding | SHA-256 | `c58009446a62bdfff9fcbcccb003ad66ab25a3f242687169619098b456ce6eb0` |
| `backend/internal/dailydiet/service.go` | service implementation | no finding | SHA-256 | `191c17f3cdc84dacf03a0c3007ea29adbfd3c02b05a0396f533d37ebc6820d6c` |
| `backend/internal/dailydiet/service_test.go` | service fake/tests | no finding | SHA-256 | `8974b57ca08281025d247b92a1f81edf6d922aee1b5684653273e2e5f9389907` |
| `backend/internal/httpapi/daily_diet_controller.go` | create caller and API projection | no finding | SHA-256 | `d2e0e9346968402a2453012857c15432b363b9c8aef68d00d5349cb43aaf7dea` |
| `backend/internal/httpapi/daily_diet_controller_test.go` | controller caller tests | no finding | SHA-256 | `34485172a7e6ada598863b6ff01e7c3e8ffcc290bc44961866e629bbcb40d777` |
| `backend/internal/repository/saved_diet_mutation_repository.go` | typed persistence, scanner, decoder, validators | no finding after repair | SHA-256 | `548ecc8a2aa6aa69272d4cbc98c67867c62c2ef9bbe7c70093cf1b00c318581d` |
| `backend/internal/repository/types.go` | response/claim types and interface | no finding | SHA-256 | `c1c2ce654f89100b093efdf0dfa5182f535b549c2c8c2a34c6a8ed8689d0511f` |
| `backend/internal/repository/daily_diet_create_claim_test.go` | decoder, persistence, and rollback tests | no finding after repair | SHA-256 | `9dd7069aae8f8c0247eb46ba54ba0240ebd311c9a94c22164b2edc874e46e4d2` |
| `backend/internal/repository/user_data_repository.go` | shared saved-diet helpers and saved-item transaction consumer | no finding | SHA-256 | `41bf37f97e5dfb35b5a79620452e360b346e3d3368a15358301145765054651e` |
| `backend/internal/repository/postgres.go` | transaction/error helper | no finding | SHA-256 | `ea59cfa009486ac4d37c620b8051005a925125d4d4ebdc4f63e506b9e84ef637` |
| `backend/internal/repository/checkout_idempotency_repository.go` | generalized-table checkout consumer | no finding | SHA-256 | `bfaab608100986199789a9374e74a2868de7fd1baa26186912887b0777b60fed` |
| `backend/internal/repository/sql/daily_diet_create_claim.sql` | claim INSERT | no finding | SHA-256 | `1054668ac9fc21e6e963b66372f6ad2e9ed3ccd5dcad79550211d4e4e58e5c8c` |
| `backend/internal/repository/sql/daily_diet_create_claim_get.sql` | locked claim SELECT | no finding | SHA-256 | `70c7e78e6c805b86b0d4ad3ea81530e9bc0707dd91e37b669745774227dbba57` |
| `backend/internal/repository/sql/saved_diet_create_snapshot.sql` | parent snapshot INSERT | no finding | SHA-256 | `03980279c4eb3ad34c64b5cf26dcf27ed0f8d26850c4891b55e8c94bb0d5e6aa` |
| `backend/internal/repository/sql/saved_diet_entry_insert_snapshot.sql` | entry snapshot INSERT | no finding | SHA-256 | `d7a10a84a49bbb8c12c987901daf5b76c90911cba82367cd7cd50faa29c94fac` |
| `backend/internal/repository/sql/checkout_idempotency_get.sql` | checkout shared-table read | no finding | SHA-256 | `296e99d52ec96c58e43ace452a9e42b304ddf0d53c43ff0a8c91d9c3f36d0ae0` |
| `backend/internal/repository/sql/checkout_idempotency_store.sql` | checkout shared-table write | no finding | SHA-256 | `4b023025deac6ee642f1038c7ae6d1fe4c506076f0116e5f02773ae538632efb` |
| `backend/internal/repository/sql/saved_diet_exists.sql` | ownership existence query | no finding | SHA-256 | `e016f9c120e519ac80410948c205ff61422b5b0c57f1ea9a6c084cb91e3cd7cc` |
| `backend/internal/repository/sql/saved_diet_delete.sql` | ownership delete query | no finding | SHA-256 | `95176d8074a9ba81b1ea3f2da890544f617a9f9db380cf56b49b38a103c691ff` |
| `backend/internal/repository/sql/saved_diet_saved_item.sql` | saved-item snapshot consumer | no finding | SHA-256 | `a43c8fa4677b877562b9571ba0e1529feb540d1d1dec4f5e8901e64e44ee4be5` |
| `backend/internal/repository/sql/saved_diet_create.sql` | existing parent create consumer | no finding | SHA-256 | `c126d753bcf827f6d1cd73cabd68a06f42cb5ffd37608ffdd089a657b68c688d` |
| `backend/internal/repository/sql/saved_diet_entry_insert.sql` | existing entry insert consumer | no finding | SHA-256 | `32783cf8863d0aed2b3ee70227e5e876c6fa755cfa9402fc1266ed1ff4ba8b58` |
| `backend/internal/repository/sql/saved_diet_entry_clear.sql` | existing entry replacement consumer | no finding | SHA-256 | `9bbdb410c2996b869d51b7ba65d47f0be0ed17da86cfd71734404b98f61a0cc0` |
| `backend/internal/repository/sql/saved_diet_get.sql` | ownership read consumer | no finding | SHA-256 | `5379cddef4fab43153353b084f14a3ca0111cea0b5209e94b6312b9280133d70` |
| `backend/internal/repository/sql/saved_diet_entry_list.sql` | ordered entry read consumer | no finding | SHA-256 | `172303c0226eaa855979b9a1107081449e446fa9f4df5582b16f121b3465c292` |
| `backend/internal/repository/sql/saved_diet_create_with_id.sql` | stale artifact | optional finding recorded | SHA-256 | `bb100e99827a1f9995440fd9e6dbf5c45de389bf2173c76a852f4f07ed64daaa` |
| `database/migrations/000017_checkout_idempotency.up.sql` | original table/FK source | no finding | SHA-256 | `41f65a67022980f7da44221c6bf711d717ae2293e63607b7e03c5daae7a08532` |
| `database/migrations/000021_mutation_idempotency.up.sql` | shared-table upgrade | no finding | SHA-256 | `354e77c40320c5c31711241e130a460ee9be86b974f20baa6c4266d3f609a08b` |
| `database/migrations/000021_mutation_idempotency.down.sql` | shared-table rollback | no finding | SHA-256 | `4c915288ba2299adc4ec3c3e2c728db12be795f4df9c4b4ebb22f53e59a4f7ba` |
| `backend/internal/migrations/migrations.go` | migration execution order and context | no finding | SHA-256 | `001c8a13fe04a249acfee803bf5e94cde2d35ad87af780dba30d67baeda178d6` |
| `docs/design/DESIGN-007.md` | checkout shared-storage context | no contradiction | SHA-256 | `875d1e1479d600da71b99459ec118a486bd7d621ea42f40e81ea958a456bd6bc` |
| `docs/design/DESIGN-008.md` | SavedDataRepository source context | no contradiction | SHA-256 | `551880a70a3b42698e11632a471957bbecc6011900cf121c34f1ff3c3db18b87` |
| deleted `backend/internal/repository/sql/checkout_idempotency_claim.sql` | obsolete baseline SQL | deleted; baseline fingerprint | SHA-256 | `628fd0182fb0e09b6dbe9e2fac399799b0a84b174cf66d3d05f199081bcb256c` |
| deleted `backend/internal/repository/sql/checkout_idempotency_get_for_update.sql` | obsolete baseline SQL | deleted; baseline fingerprint | SHA-256 | `55d18718401c7784e0579c6f18336ccbae2787865dcad8d62c17b27dbe7b191d` |

```yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The prior review recorded pre-repair hashes aa8fb95c... for saved_diet_mutation_repository.go and e7922d53... for daily_diet_create_claim_test.go; current hashes 548ecc8a... and 9dd7069a... were independently re-read and audited."
  - "The prior review recorded an earlier preparation-document hash; current preparation evidence was re-read and current task status/dependencies were independently checked."
  - "Prior evidence for Task 214 and Task 215 was not reused for the changed repository contract; current types.go, all callers, scanners, validators, and design sources were re-audited."
```

## 10. Coverage and Exceptions

- [x] Required coverage command ran.
- [x] Report path and observed threshold are recorded.
- [x] Untested branches relevant to changed symbols were inspected.
- [x] Exceptions exactly match the task row and are justified.

```yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "backend/coverage.out"
observed_line_coverage: "88.0% aggregate; 93.4% internal/repository; 80.1% internal/dailydiet; decoder and response validator 100.0%"
coverage_passed: true
```

Coverage finding: The command and all package tests pass. The task row defines no numeric coverage threshold, and the repository’s accepted Phase 07 below-100% coverage disposition is recorded in `docs/implementation/04_OPEN.md`; the changed decoder, response validator, replay, malformed-row, and rollback branches are directly exercised or manually audited.

## 11. Negative and Regression Checks

- [x] Existing focused tests pass.
- [x] No unrelated dependency or architectural boundary was introduced.
- [x] No source-of-truth documentation was contradicted.
- [x] No generated/cache/build/temporary artifact was unintentionally added.
- [x] Public API additions are necessary and used.
- [x] Duplicate helpers and obsolete aliases were searched for.
- [x] Error, cleanup, timeout, concurrency, and malformed-input paths were challenged.

Findings: The decoder is now presence-aware without changing the typed public response shape. SQL uses bound values only. Transaction rollback and context propagation were traced. Race testing found no shared-state defects. The optional stale `saved_diet_create_with_id.sql` artifact is visible in Section 7 and does not affect the decision.

## 12. Decision

A task may be `PASSED` only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains.

Before accepting the decision, run:

```bash
python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-216-review.md
```

```yaml
decision: "PASSED"
reason: "The current presence-aware aggregateMacros decoder rejects missing, null, and empty macro objects while accepting explicit numeric zero, and all task criteria, symbol audits, and validation gates pass."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "None for Task 216; retain the optional stale-SQL cleanup note for a separate cleanup task."
```

## 13. Repair Context

Repair context is recorded because the prior review was rejected. The prior important finding is closed and no further repair is requested.

### Failure Summary

The prior decoder decoded `aggregateMacros` into a non-pointer value, allowing missing, `null`, and `{}` to become an accepted all-zero struct.

### Minimal Repair Goal

Require a non-null macro object and all four required numeric members while preserving explicit numeric zero values, and prove both pure decoding and persisted JSONB no-write behavior.

### Evidence to Reuse

Current `saved_diet_mutation_repository.go:168-192`, current `daily_diet_create_claim_test.go:17-68` and `110-193`, focused PostgreSQL tests, full backend tests, race test, vet, coverage, vulnerability scan, and current file hashes in Section 9.

### Required Re-Review Surface

`decodeDailyDietCreateResponse`, `validateDailyDietCreateResponse`, `scanDailyDietCreateClaim`, `GetDailyDietCreateClaim`, the persisted malformed-row matrix, service replay mapping, controller caller, and all current task-owned implementation/test/SQL/migration units listed in Sections 5 and 9.

### Do Not Change

Do not edit production code or `docs/implementation/02_TASK_LIST.md` during this review. Preserve immutable replay behavior, typed claim scope, transaction rollback, cross-instance coordination, canonical units, and checkout’s existing consumer contract.
