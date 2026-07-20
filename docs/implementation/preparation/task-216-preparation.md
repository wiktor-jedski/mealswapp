# Task 216 Preparation Evidence

## Assignment and current conclusion

- Task: **216 — Phase 07.01 Durable Daily Diet Create Idempotency**.
- Design source: `DESIGN-008: SavedDataRepository`.
- Baseline/HEAD: `a4e31367485b03269e90b5607f2057c9568bb5b1`.
- Current task status: `PREPARED`; this preparation refresh did not edit `docs/implementation/02_TASK_LIST.md` or any task status.
- Independent review source: `docs/implementation/reviews/task-216-review.md` (`REJECTED` on 2026-07-14 for one important malformed-response finding).
- Current conclusion: the existing presence-aware production decoder and malformed-response tests close that finding. No further production or test edit was needed during this refresh.

The reviewer found that decoding required `aggregateMacros` directly into a value struct allowed an absent field, JSON `null`, or `{}` to become a valid all-zero macro value. The current `decodeDailyDietCreateResponse` first performs the strict typed decode and then decodes the required macro surface through pointers. It requires a non-nil `aggregateMacros` object and non-nil `protein`, `carbohydrates`, `fat`, and `calories` fields. This rejects all three malformed shapes while preserving valid explicit zero values.

## Exact Task 216 changed paths

Added:

- `backend/internal/repository/daily_diet_create_claim_test.go`
- `backend/internal/repository/sql/daily_diet_create_claim.sql`
- `backend/internal/repository/sql/daily_diet_create_claim_get.sql`
- `backend/internal/repository/sql/saved_diet_create_snapshot.sql`
- `backend/internal/repository/sql/saved_diet_entry_insert_snapshot.sql`
- `database/migrations/000021_mutation_idempotency.up.sql`
- `database/migrations/000021_mutation_idempotency.down.sql`
- `docs/implementation/preparation/task-216-preparation.md`

Modified:

- `backend/internal/app/app.go`
- `backend/internal/app/daily_diet_api_integration_test.go`
- `backend/internal/dailydiet/service.go`
- `backend/internal/dailydiet/service_test.go`
- `backend/internal/repository/saved_diet_mutation_repository.go`
- `backend/internal/repository/types.go`
- `backend/internal/repository/sql/checkout_idempotency_get.sql`
- `backend/internal/repository/sql/checkout_idempotency_store.sql`

Deleted:

- `backend/internal/repository/sql/checkout_idempotency_claim.sql`
- `backend/internal/repository/sql/checkout_idempotency_get_for_update.sql`

The shared worktree also contains Tasks 213, 215, 217, and later Phase 07.01 changes. They are not Task 216 evidence. In particular, `docs/implementation/02_TASK_LIST.md`, `docs/implementation/preparation/task-217-preparation.md`, unrelated optimization/frontend files, and Task 215 canonical-unit changes were preserved.

## Exact symbols

Production contracts/types:

- `repository.DailyDietCreateResponse`
- `repository.DailyDietCreateResponseEntry`
- `repository.DailyDietCreateResponseMacros`
- `repository.DailyDietCreateClaim`
- `repository.DailyDietCreateClaimResult`
- `repository.DailyDietMutationRepository`
- `repository.dailyDietCreateRecord`

Production behavior:

- `repository.(*PostgresSavedDataRepository).GetDailyDietCreateClaim`
- `repository.(*PostgresSavedDataRepository).ClaimDailyDietCreate`
- `repository.(*PostgresSavedDataRepository).DeleteIfOwned`
- `repository.scanDailyDietCreateClaim`
- `repository.decodeDailyDietCreateResponse`
- `repository.validateDailyDietCreateClaim`
- `repository.validateDailyDietCreateScope`
- `repository.validateDailyDietCreateResponse`
- `repository.createSavedDietSnapshot`
- `repository.dailyDietCreateMethod`, `repository.dailyDietCreateRoute`
- `repository.dailyDietCreateClaimSQL`, `repository.dailyDietCreateClaimGetSQL`
- `repository.savedDietCreateSnapshotSQL`, `repository.savedDietEntryInsertSnapshotSQL`
- `dailydiet.Service`, `dailydiet.NewService`
- `dailydiet.(*Service).Create`, `dailydiet.(*Service).prepareCreate`
- `dailydiet.createResultFromClaim`
- `app.NewProduction`

Tests/fakes:

- `repository.TestDecodeDailyDietCreateResponseRejectsInvalidPersistedBodies`
- `repository.TestPostgresDailyDietCreateClaimReplaysImmutableResponseAndCascades`
- `repository.TestPostgresDailyDietCreateClaimRejectsLegacyDualAndRollsBackWrites`
- `repository.staticDailyDietCreateResponsePayload`
- `repository.mutatedDailyDietCreateResponsePayload`
- `repository.validDailyDietCreateResponsePayload`
- `repository.responseEntry`, `repository.responseMacros`
- `repository.testDailyDietCreateClaim`, `repository.assertNoDailyDietClaimWrites`
- `dailydiet.memoryDietRepository`, `dailydiet.memoryDailyDietClaim`, `dailydiet.memoryMealRepository`
- `dailydiet.TestServiceCreateAggregatesMultipleMealsAndReplaysIdempotently`
- `dailydiet.TestServiceCreateLooksUpEachDistinctMealOnceAtMaximumEntries`
- `dailydiet.TestServiceCreateSameKeyIsAtomicAcrossInstances`
- `dailydiet.TestServiceCreateDoesNotBlockIndependentUsersAndHonorsCancellation`
- `app.TestDailyDietProductionAPIWithLivePostgres`

Removed Task 216 production surface:

- `repository.AtomicDailyDietMutationResult`
- `repository.(*PostgresSavedDataRepository).CreateWithIdempotency`
- `repository.dailyDietIDFromIdempotencyResponse`
- `dailydiet.(*Service).replay`, `dailydiet.(*Service).replayRecord`
- `dailydiet.dailyDietIdempotencyResponse`, `dailydiet.dailyDietIDFromIdempotencyResponse`
- Daily Diet's checkout-idempotency dependency, process-wide create mutex, and checkout-specific test fake.

Repository searches found none of those retired Daily Diet helpers and no checkout-named symbol in the current Daily Diet service or typed create-claim implementation.

## Reviewer-finding repair scope

The post-review repair is intentionally narrow:

1. `backend/internal/repository/saved_diet_mutation_repository.go:168` — `decodeDailyDietCreateResponse` adds a presence-aware required-field decode for the `aggregateMacros` object and its four numeric members. This is a concrete production behavior correction: malformed missing/null/empty macro objects now return `ErrorKindInternal`; explicit zero-valued fields remain valid.
2. `backend/internal/repository/daily_diet_create_claim_test.go:17` — `TestDecodeDailyDietCreateResponseRejectsInvalidPersistedBodies` proves explicit zero macros succeed and missing, null, and empty macro objects fail, alongside unknown, trailing, malformed, wrong-type, domain-invalid, legacy-ID, and dual-ID bodies.
3. `backend/internal/repository/daily_diet_create_claim_test.go:110` — `TestPostgresDailyDietCreateClaimRejectsLegacyDualAndRollsBackWrites` inserts missing/null/empty macro JSONB rows, verifies `GetDailyDietCreateClaim` returns `ErrorKindInternal`, retains the original mutation row, and creates no diet.

No other production symbol was changed to address the reviewer finding. This evidence refresh itself overwrote only this document.

## Verification-criteria mapping

| Task 216 criterion | Current evidence | Result |
|---|---|---|
| First claim and immutable exact replay | Direct PostgreSQL, service, and production API tests compare the stored response. | PASS |
| Replay after replacement, deletion, and macro changes | Repository/service/API fixtures mutate or delete current state and still obtain the original response without recreation. | PASS |
| Changed-body conflict | Repository/service/API tests reject the changed hash without another create. | PASS |
| Atomic same-key behavior across instances | Service and live production-app concurrency fixtures converge on one response ID and one persisted diet. | PASS |
| Independent-user progress and cancellation | Blocking meal-repository fixture allows the unrelated user through and settles cancellation. | PASS |
| Rollback at each write stage | Claim, parent, entry, and saved-item failures leave no claim/diet residue. | PASS |
| Account cascade and ownership-aware deletion | PostgreSQL cascade and user-scoped deletion assertions pass. | PASS |
| Exact method/route/status/hash/ID/response shape | Fixed `POST /daily-diets`, exact 201, SHA-256, UUID/correspondence validation, strict unknown/trailing/type/domain checks, and presence-aware macro checks are implemented. | PASS |
| Reject malformed, dual-ID, and legacy bodies without writes | Pure decoder and persisted JSONB matrices include the reviewer's missing/null/empty macro cases and verify safe internal errors/no writes. | PASS |
| No duplicate meal lookups at 100 entries | Instrumented maximum-entry service test observes one lookup for one distinct meal. | PASS |
| No checkout-specific Daily Diet helper | Removed-symbol and checkout-name searches are empty for the Daily Diet create path; shared storage is explicitly generalized. | PASS |
| Backend/full/race/vet and repository validators | Commands below all exited 0. | PASS |

## Commands and current results

All commands were run against the current shared worktree on 2026-07-15.

| Command | Result |
|---|---|
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/repository -run 'TestDecodeDailyDietCreateResponseRejectsInvalidPersistedBodies|TestPostgresDailyDietCreateClaimRejectsLegacyDualAndRollsBackWrites' -count=1` | PASS; repository package completed in 1.147s, including the local PostgreSQL fixture. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./...` | PASS; every backend package passed. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./...` | PASS; every backend package passed, repository completed in 32.207s. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS. |
| `python3 scripts/validate-traceability.py` | PASS: `Traceability validation passed.` |
| `python3 scripts/validate-task-list.py` | PASS: 237 sequential tasks with ordered dependencies. |
| `git diff --check` | PASS; no whitespace errors. |
| `git status --short` plus scoped `git diff -- docs/implementation/02_TASK_LIST.md` inspection | PASS; shared changes remain present and Task 216 remains `PREPARED`; this refresh did not edit the task list. |
| Removed-symbol and Daily Diet checkout-name `rg` searches | PASS; no matches. |

## SHA-256 fingerprints

These hashes capture the current Task 216 implementation/test/schema surface after the reviewer repair. Deleted files use their baseline-content hashes from the independent review. The preparation document is not self-hashed because embedding its own final digest would change that digest.

| Path | SHA-256 |
|---|---|
| `backend/internal/app/app.go` | `bf4b26213e9c3e6ce856d9793c980152975e178f86a1da74367f93d5a68d2066` |
| `backend/internal/app/daily_diet_api_integration_test.go` | `c58009446a62bdfff9fcbcccb003ad66ab25a3f242687169619098b456ce6eb0` |
| `backend/internal/dailydiet/service.go` | `191c17f3cdc84dacf03a0c3007ea29adbfd3c02b05a0396f533d37ebc6820d6c` |
| `backend/internal/dailydiet/service_test.go` | `8974b57ca08281025d247b92a1f81edf6d922aee1b5684653273e2e5f9389907` |
| `backend/internal/repository/saved_diet_mutation_repository.go` | `548ecc8a2aa6aa69272d4cbc98c67867c62c2ef9bbe7c70093cf1b00c318581d` |
| `backend/internal/repository/types.go` | `c1c2ce654f89100b093efdf0dfa5182f535b549c2c8c2a34c6a8ed8689d0511f` |
| `backend/internal/repository/daily_diet_create_claim_test.go` | `9dd7069aae8f8c0247eb46ba54ba0240ebd311c9a94c22164b2edc874e46e4d2` |
| `backend/internal/repository/sql/daily_diet_create_claim.sql` | `1054668ac9fc21e6e963b66372f6ad2e9ed3ccd5dcad79550211d4e4e58e5c8c` |
| `backend/internal/repository/sql/daily_diet_create_claim_get.sql` | `70c7e78e6c805b86b0d4ad3ea81530e9bc0707dd91e37b669745774227dbba57` |
| `backend/internal/repository/sql/saved_diet_create_snapshot.sql` | `03980279c4eb3ad34c64b5cf26dcf27ed0f8d26850c4891b55e8c94bb0d5e6aa` |
| `backend/internal/repository/sql/saved_diet_entry_insert_snapshot.sql` | `d7a10a84a49bbb8c12c987901daf5b76c90911cba82367cd7cd50faa29c94fac` |
| `backend/internal/repository/sql/checkout_idempotency_get.sql` | `296e99d52ec96c58e43ace452a9e42b304ddf0d53c43ff0a8c91d9c3f36d0ae0` |
| `backend/internal/repository/sql/checkout_idempotency_store.sql` | `4b023025deac6ee642f1038c7ae6d1fe4c506076f0116e5f02773ae538632efb` |
| `database/migrations/000021_mutation_idempotency.up.sql` | `354e77c40320c5c31711241e130a460ee9be86b974f20baa6c4266d3f609a08b` |
| `database/migrations/000021_mutation_idempotency.down.sql` | `4c915288ba2299adc4ec3c3e2c728db12be795f4df9c4b4ebb22f53e59a4f7ba` |
| deleted `backend/internal/repository/sql/checkout_idempotency_claim.sql` baseline content | `628fd0182fb0e09b6dbe9e2fac399799b0a84b174cf66d3d05f199081bcb256c` |
| deleted `backend/internal/repository/sql/checkout_idempotency_get_for_update.sql` baseline content | `55d18718401c7784e0579c6f18336ccbae2787865dcad8d62c17b27dbe7b191d` |

Repair staleness evidence: the independent review recorded pre-repair hashes `aa8fb95c...` for `saved_diet_mutation_repository.go` and `e7922d53...` for `daily_diet_create_claim_test.go`; the current hashes above differ exactly because the presence-aware production hunk and its missing/null/empty malformed tests now exist.

## Final repair boundary

Task 216 preparation is complete for re-review. The important reviewer finding has direct production and adversarial regression coverage, all requested gates pass, no concrete defect remains, no unrelated production code was edited, and Task 216's status remains `PREPARED`.
