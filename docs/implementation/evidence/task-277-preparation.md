# Task 277 preparation evidence

## Outcome

Task 277, Phase 08.02 JSON Global Catalog Import Operator, is implemented and verified. `docs/implementation/02_TASK_LIST.md` was not edited and Task 277 remains `OPEN`.

The implementation adds one explicit standard-library Python operator for at most 500 items and no bulk endpoint or direct database path. It authenticates through the existing login route, acquires fresh CSRF state, preflights the complete document and all remote references, and sends one paced, audited, idempotent `POST /api/v1/admin/items` per item. The existing manual-administration request now requires canonical `allergenKeys`; active keys, classifications, the ownerless item, immutable replay response, administrator audit, and post-commit cache invalidation are handled by the existing transaction/commit boundary.

## Baseline and preservation

- Worktree: `/home/wiktor/Work/mealswapp`
- Baseline commit: `f9a646ffede5c8a63ad787a07eaba7efc0433677`
- Baseline status: dirty with 33 modified and 17 untracked status entries from Task 276 and earlier Phase 08 work. The complete `git status --short` was captured before edits. Overlapping shared files included `backend/internal/repository/types.go`, `docs/design/DESIGN-009.md`, and the externally modified task list.
- Dependency evidence read: Task 250, 251, and 253 preparation/review evidence and Task 276 preparation/review evidence. Their authenticated admin gateway, item curation, classification list, strict success envelopes, active allergen vocabulary, idempotency, bootstrap, and auth behavior were treated as the implemented baseline.
- Preservation: no reset, checkout, clean, commit, task transition, public bulk route, database migration, or direct PostgreSQL import was performed. Existing Task 276 auth/bootstrap and unrelated Phase 08 hunks remain intact.
- Reference confidence: **HIGH** for new files and exact in-turn patches; **HIGH** for existing gateway/repository behavior because dependency evidence and focused/live tests agree. Shared dirty-file ownership is attributed below by symbol.

### Pre-work task-surface fingerprints

| Path | Baseline SHA-256 |
|---|---|
| `api/openapi.yaml` | `02c6657a15ce63d3f5e7f6ea7c35c84938bce6592b2c06dd8fac37f0b268fd9c` |
| `backend/internal/httpapi/manual_item_controller.go` | `48659ad69db18fbb3f830a4e0f833786dcd0c4c8c51f777211dbfe8001324382` |
| `backend/internal/httpapi/manual_item_controller_test.go` | `2f953697baabed40e6d7539f0a99499a5a6dac3810123699d2745a5a79fbf36e` |
| `backend/internal/itemcurator/service.go` | `b3b3b699a6f62ec4c6b51d803a103744cbd01c0bb294ccced38b35167b920407` |
| `backend/internal/repository/manual_food_repository.go` | `7cce8d565c88161e3639253cd397a736dccf9915d612255a4c237089ca91b6fa` |
| `backend/internal/repository/manual_food_repository_test.go` | `e83642d13869e416dbc0c8681412322e43d10938d3543137185e39735583184d` |
| `backend/internal/repository/types.go` | `d52db09e9dd3d8caa266f52c0bc718e9d707ffc267760e30c2f0d83cee5b02af` |
| `docs/design/DESIGN-009.md` | `be8e978538a24db6d8e216fae4a5a5c8b5b5a942c29f04c2c7d6fda756c8ee94` |
| `frontend/src/lib/api/generated.ts` | `e08701b57ede329f92e2437411c7599856b4864ca2ce4e282e9dd3e4f16cb264` |
| `scripts/generate-api-types.py` | `71fa504eedd60e2807483e309684938db829f9a283b7f660076f2d4f4a68278e` |
| `scripts/test_generate_api_types.py` | `f8e1291f1433b0764b5ca9c01644a54d3f044872d3eda3e54def4d5f9312bbc6` |
| `backend/internal/app/task271_backend_regression_integration_test.go` | `819d036475cd3ad24f30e245b57ac31c09836d7ce8200cfe5434b5668561ec74` |
| `frontend/src/lib/admin-workflows.test.ts` | `68a6f890cba079d16b2b5db1cc2872084b107417762e8809be8af030fcf822b6` |
| `frontend/src/lib/admin-workflows.ts` | `8cbb52e965d4f05c725b424deba76136e52a3f235dddb2281cfd2a8e2822c0c9` |
| `frontend/src/lib/api/admin-client.test.ts` | `931918cd3c0e10eddb823f680385c175b667675eede8cc962a315c831bcda83a` |
| `frontend/src/lib/api/admin-client.ts` | `7e3b4d109dd08d28a5cb7e384c1cf5f60d206b13be86dcc8ccee00ee328a62dc` |
| `scripts/check.py` | `f843434d561d884cc90fd10829dddbccedca02046165483f5dbf27b80cd0b5ea` |
| New operator, operator test/doc/evidence, and four allergen SQL files | absent |
| `docs/implementation/02_TASK_LIST.md` | `e93b4df39b52897f0ae87299edb0fba7c082ca5446d80da02ae025acdd658683` |

Clean-file baselines added to the task surface during implementation were verified against the fixed commit; overlapping dirty-file hashes were captured directly before editing.

## Implemented behavior

- Closed `mealswapp.global-catalog.v1` root and entry objects; duplicate JSON keys at every depth are rejected.
- One nonblank operator-supplied 8–255 printable idempotency key per item; duplicate keys are rejected and keys are never derived from positions.
- Complete local validation covers required/unknown/null fields, 500-item maximum, finite nonnegative macros/micros, solid 100 g macro ceiling, metric-only weights/volumes/density, solid/liquid density/provenance rules, canonical micronutrient/allergen vocabularies, and name-versus-UUID classification alternatives.
- Portable classification names are whitespace/case-normalized only for lookup and must match exactly one active item of the required kind. Advanced UUIDs are parsed canonically and must exist in the active response. Remote malformed UUIDs fail preflight.
- Login email and password are interactive and have no command-line option. Password entry is non-echoing. `CookieJar` is memory-only; no cookie file exists. CSRF is acquired only after successful login.
- Both classification lists and the existing allergen filter vocabulary are decoded before mutation. Any local or remote preflight failure produces zero create requests.
- Dry run performs login, fresh CSRF, and complete remote preflight but no mutation.
- Create attempts preserve input order and are at least 2.1 seconds apart. Network/TLS/timeout and 5xx ambiguity retries use the identical key and deterministic JSON body. `Retry-After` accepts only a nonnegative delta/date no greater than 30 seconds. Retry attempts are capped at three.
- HTTP 400/409 are permanent item failures; 401/403 abort; other exhausted failures remain item failures. Partial failure returns status 1, run/setup failure status 2.
- Reports contain only schema, one-based index, a 12-hex SHA-256 key fingerprint, bounded outcome, and HTTP status. Stdout contains counts only; stderr contains fixed/index-only errors. Tests assert credentials, email, cookies, CSRF, full keys, item names, and bodies are absent.
- Deterministic order and unchanged keys make interruption recovery a normal rerun. The existing immutable idempotency claim proves exact replay creates no duplicate and changed-body key reuse conflicts.
- `allergenKeys` is required only by `AdminItemRequest`, not private custom-item requests. Go service normalization sorts keys; PostgreSQL verifies every key is active, then clears/replaces and hydrates assignments in the same audited transaction.
- `AdminItem` success responses and the generated/frontend strict decoder now require a non-null canonical allergen array. Classification preflight rejects non-UUID response IDs, closing the known response defect at the operator boundary.
- Operator documentation establishes 500 items as the moderate-volume ceiling and directs larger datasets to a separately designed offline ingestion path.

## Task-owned changed paths

### Operator, tests, and gate

- `scripts/import-global-catalog.py` (new, executable mode 0755)
- `scripts/test_import_global_catalog.py` (new)
- `scripts/check.py`

### Backend contract and persistence

- `backend/internal/repository/types.go`
- `backend/internal/itemcurator/service.go`
- `backend/internal/httpapi/manual_item_controller.go`
- `backend/internal/repository/manual_food_repository.go`
- `backend/internal/repository/sql/food_allergens_clear.sql` (new)
- `backend/internal/repository/sql/food_allergens_list.sql` (new)
- `backend/internal/repository/sql/food_allergens_replace.sql` (new)
- `backend/internal/repository/sql/food_allergens_validate.sql` (new)

### Backend tests and propagated regression fixture

- `backend/internal/itemcurator/service_test.go` was exercised but not changed.
- `backend/internal/httpapi/manual_item_controller_test.go`
- `backend/internal/repository/manual_food_repository_test.go`
- `backend/internal/app/task271_backend_regression_integration_test.go`

### OpenAPI, generated, frontend, and documentation

- `api/openapi.yaml`
- `scripts/generate-api-types.py`
- `scripts/test_generate_api_types.py`
- `frontend/src/lib/api/generated.ts`
- `frontend/src/lib/api/admin-client.ts`
- `frontend/src/lib/api/admin-client.test.ts`
- `frontend/src/lib/admin-workflows.ts`
- `frontend/src/lib/admin-workflows.test.ts`
- `frontend/src/lib/components/AdminDataManagement.svelte`
- `frontend/src/lib/components/AdminDataManagement.test.ts`
- `docs/design/DESIGN-009.md`
- `docs/operations/global-catalog-import.md` (new)
- `docs/implementation/evidence/task-277-preparation.md` (new)

## Added or modified executable symbols

### Python production additions

- Exceptions: `CatalogError`, `DuplicateJSONKey`, `AuthorizationError`
- Data/class: `APIResponse`, `APIClient`
- `reject_duplicate_keys`, `load_document`, `canonical_name`, `finite_nonnegative`, `validate_item`, `validate_document`, `key_fingerprint`
- `APIClient.__init__`, `APIClient.request`, `APIClient.login`, `APIClient.pace`, `APIClient.create`
- `retry_after_seconds`, `response_data`, `preflight`, `write_report`, `run`

### Go production additions/modifications

- Modified declarations: `repository.FoodItemEntity`, `itemcurator.Request`, `itemcurator.Item`
- Modified functions/methods: `itemcurator.Service.Create`, `itemcurator.Service.Update`, `itemcurator.requestHash`, `itemcurator.toEntity`, `itemcurator.fromEntity`, `httpapi.validateManualItemBody`, `httpapi.manualItemRequest`, `httpapi.manualItemData`, `repository.PostgresManualFoodItemRepository.Update`, `repository.createManualFoodItem`, `repository.getManualFoodByID`
- Added functions: `itemcurator.validateRequest`, `httpapi.decodeManualItemRequest`, `httpapi.hasDuplicateString`, `repository.validateManualFoodAllergens`, `repository.replaceManualFoodAllergens`, `repository.hydrateManualFoodAllergens`
- Added embedded SQL variables: `foodAllergensValidateSQL`, `foodAllergensClearSQL`, `foodAllergensReplaceSQL`, `foodAllergensListSQL`

### Frontend/generator/gate modifications

- Modified `parseAdminItemForm`, `decodeItem`, `phase08_contract_mismatches`, `validate_generator_tests`, and generated `AdminItemRequest`
- Added `allergenKeys` decoder helper and `validate_global_catalog_operator_tests`
- Added `task271CustomItemBody`; modified `task271ItemBody` and `task271CreateCustomItem`

### Test executable symbols

- Added Python fixtures/classes: `valid_entry`, `ValidationTests`, `StubClient`, `PreflightTests`, `FakeClock`, `SequenceClient`, `RetryTests`, `OperatorServer`, `EndToEndTests`
- Added fifteen Python test methods covering document/schema/key/metric/reference/retry/auth/dry-run/report/partial-failure behavior.
- Modified `TestManualItemAdminHTTPValidCRUDReplayAndAuditSnapshots`, `TestManualItemAdminHTTPRejectsConflictsDuplicatesInvalidFieldsAndOwnership`, `TestManualItemInvalidationRunsOnlyAfterSuccessfulAuditCommit`, `TestPostgresManualFoodItemCRUD`, frontend admin workflow tests, frontend strict admin-client tests, and Phase 08 generator tests.

## Criteria coverage

| Task 277 criterion | Evidence |
|---|---|
| Version/schema, duplicates, malformed items, 500 limit | Python validation tests and closed object checks. |
| Stable unique keys and safe fingerprints | Validation/fingerprint tests; report excludes raw keys. |
| Metric solid/liquid/density and no imperial persistence | Local validation tests plus existing Go domain validation; unknown imperial fields are rejected. |
| Canonical micronutrients/allergens | Fixed v1 micronutrient vocabulary validation; active allergen remote preflight and transactional PostgreSQL validation. |
| Classification names and UUID alternative | Name/UUID preflight tests; unknown, ambiguous, wrong-kind, malformed UUID cases fail before mutation. |
| Complete preflight and authenticated dry run | Fake HTTP server proves login, cookie use, fresh CSRF, all reads, and zero create calls. |
| One create per item and 30/minute ceiling | Create loop has one call per resolved item; all attempts share the 2.1-second pacer. |
| Retry-After and ambiguous retry | Fake client proves bounded delay and byte-identical key/body over 500/429/201. |
| Replay/conflict/resume | Existing live PostgreSQL idempotency test proves one row/audit on replay and changed-body conflict; deterministic operator ordering/fingerprints and partial report test prove resumable sequencing. |
| Permanent/auth/partial failures | Client tests cover 400/409 permanence and 403 abort; run test proves partial failure exits 1 and stable report ordering. |
| PII/credential safety | Fake-server test captures stdout, stderr, report and rejects email, password, cookie, CSRF, full key, and item name. Email/password have no CLI arguments. |
| Atomic item/classification/allergen/audit/cache effects | Live `TestPostgresManualFoodItemCRUD`, manual HTTP audit/invalidation tests, and Task 271 production multi-instance gate pass. Unknown allergen and audit failure leave no item. |
| Contract-conforming responses | Manual HTTP test decodes strict `status/requestId/data`; generated/OpenAPI tests guard allergen arrays; operator parses classification UUIDs and allergen options. |
| Moderate-volume documentation | `docs/operations/global-catalog-import.md` defines the ≤500 boundary and separate offline design requirement. |
| No prohibited path | No public bulk API, database connection, imperial field, position-derived key, persistent cookie, or sensitive report field exists. |

## Security review

`golang-security` coding/review mode was applied. Trust boundaries reviewed were catalog JSON, remote admin/reference responses, authentication cookies/CSRF, idempotency headers, SQL parameters, and report/error output.

- **PASS, no open security finding.**
- STRIDE summary: authentication and server-side admin authorization prevent spoofing/elevation; duplicate-key and closed-schema parsing, active-vocabulary validation, immutable idempotency hashes, and transaction-scoped SQL prevent tampering; existing admin audit provides repudiation evidence; interactive secrets, memory-only cookies, closed output, and generic failures prevent disclosure; 500-item, body, pace, retry-delay, attempt, and timeout bounds reduce denial-of-service risk.
- SQL is parameterized and embedded from repository-local files. No shell, subprocess, raw SQL string construction, direct database import, token logging, or persistent credential/cookie storage was introduced.
- `go vet ./...`: PASS.
- `govulncheck@v1.3.0 ./...`: PASS, no vulnerability in called code; one package and nineteen module advisories are not called.

## Verification commands and results

Commands ran on 2026-07-27.

| Command | Result |
|---|---|
| `python3 -m unittest scripts/test_import_global_catalog.py` | PASS, 15 operator tests. |
| `python3 -m py_compile scripts/import-global-catalog.py scripts/test_import_global_catalog.py` | PASS. |
| `python3 scripts/generate-api-types.py --check` | PASS. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS with the existing OAuth callback 302-only warning. |
| `bun run typecheck` (`frontend/`) | PASS. |
| Focused Bun admin workflow/client tests | PASS, 14 tests. |
| `go test -count=1 ./internal/repository -run TestPostgresManualFoodItemCRUD -v` (`backend/`) | PASS against isolated PostgreSQL. |
| `go test -count=1 ./internal/app -run TestTask271ProductionBackendRegressionGate -v` (`backend/`) | PASS; production composition, audited mutation, replay, rollback, cache generation, and private/global separation. |
| `go test -count=1 -race ./internal/itemcurator ./internal/httpapi ./internal/repository -run 'TestManualItem\|TestPostgresManualFoodItemCRUD'` | PASS, no race report. |
| `go test -count=1 ./...` (`backend/`) | PASS across every command and backend package. An earlier run exposed the Task 271 fixture contract propagation; after separating global/private bodies, the fresh full run passed. |
| `python3 scripts/check.py --quick` | PASS after independent-review repair: operator lane (15), API generator (24), all 534 frontend tests, typecheck, changed backend packages/live integrations, formatting, vet, vulnerability scan, OpenAPI, generated drift, Go Doc/TSDoc, task list, and traceability. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `git diff --check` | PASS. |

Coverage measurement: focused ItemCurator package coverage is 74.2%; focused HTTP/repository aggregate package totals are 17.1%/16.4% because those packages contain broad unrelated surfaces. New critical helpers measured by the focused profile include `hasDuplicateString` and `manualItemData` at 100%, `decodeManualItemRequest` at 82.4%, `validateManualFoodAllergens` at 83.3%, and allergen hydration at 76.9%. Python behavioral branches are directly exercised by 11 focused tests. The repository’s existing Phase 08 coverage gate reports its documented measured exception and passes; no new undocumented gate exception was added.

## Final task-surface fingerprints

| Path | Final SHA-256 |
|---|---|
| `api/openapi.yaml` | `69397349b60bd9744effea07cc249f8a6a65e626bf3e83b8d682c89a189372fc` |
| `backend/internal/app/task271_backend_regression_integration_test.go` | `606ee1c347602bf7e17574832f3ede551669a6df43479ada924b6a8068186a3c` |
| `backend/internal/httpapi/manual_item_controller.go` | `4e9fcf3e5a9113f5d34fa71203c4e74b5446be254b48a30d0563f647794519e0` |
| `backend/internal/httpapi/manual_item_controller_test.go` | `e60e442d6865b6a027228fe28aedc3bd83a6b28de89d955744f84e4cda25c253` |
| `backend/internal/itemcurator/service.go` | `c1e1a5f8309152ec65cf588c49a50c9a5fdf4ab7c24f1bf63b02abcd0521abd7` |
| `backend/internal/repository/manual_food_repository.go` | `140c197b6a3bfc24be809309c702ebf7da0456fd639f762cf8dcb38ac042b203` |
| `backend/internal/repository/manual_food_repository_test.go` | `674e9dea370b5231fe8e845011f386b456e1588f89b06c95785ec23a3a3978f3` |
| `backend/internal/repository/types.go` | `ba165efac8cb76ba71e3b7f537abbd0455ba4a20b10447450d1e8bc32422b6d3` |
| `backend/internal/repository/sql/food_allergens_clear.sql` | `3625eb03dbc07707aa37f77aad3f0ff7be30d0a79636351db8d50592af4b3052` |
| `backend/internal/repository/sql/food_allergens_list.sql` | `fc70951c71f0cda4c889bb15ef9d2f79c1959f64a39a1941a34e6624b64eb86c` |
| `backend/internal/repository/sql/food_allergens_replace.sql` | `aa93bf9b7843abe7a1b327bc697de7981de15e0e2a69fadfcb3ac82ca5d1c161` |
| `backend/internal/repository/sql/food_allergens_validate.sql` | `84283481a6aca98238c8e94431815fe12484d10627dd4d7e26846d1d29d4ab01` |
| `docs/design/DESIGN-009.md` | `22ca65baf347c8c1598a4a71439ca5091548fc3a8f9b5dc906406a3fdcc8c814` |
| `docs/operations/global-catalog-import.md` | `97256538264b77e6027ecc26d63a30c3ebc60b650da6af66ded8412446fe15d9` |
| `frontend/src/lib/admin-workflows.test.ts` | `f90688587108af37de1e90ea630a783a5d277bd17372d02d97b656407cb89a3e` |
| `frontend/src/lib/admin-workflows.ts` | `ff2db65f18a045eae9ab7017851ab9162739744dd7aeea800f74a36fbe41ad24` |
| `frontend/src/lib/api/admin-client.test.ts` | `311f17d57883e572b584169bd2c71ebdfedff466e9eb3e437168369c7c19cbb5` |
| `frontend/src/lib/api/admin-client.ts` | `71b76d457ff4c8379cdabab6b5e3cb6f56cf0e22ab9249abe06e3dd8678be880` |
| `frontend/src/lib/api/generated.ts` | `c9f6d7e8d8ff0b991b40168a0b7d707d70558b8ce36e886dc4118e1f722df649` |
| `scripts/check.py` | `fcdc83a3782a959d7963ff8c552db7ea08e38d2fff03dd7198194fedbc6b0cf6` |
| `scripts/generate-api-types.py` | `01ae86e27af53cbeeaa76cd95f0b85ad45e8659f9448704eb8bfbd607e3a49b9` |
| `scripts/test_generate_api_types.py` | `cbed90622dd05718eb7bc0f2a43b58d38bc5aa9cc668518d2a104f9ddb194348` |
| `scripts/import-global-catalog.py` | `db296a12e9ce291edf192127453f9cb181710b34c4072289d837512201432823` |
| `scripts/test_import_global_catalog.py` | `0b1724a3e9fa95783595e3efc4f9e31cbbc7726069595d898786936e6e454e63` |
| `frontend/src/lib/components/AdminDataManagement.svelte` | `0064ebe05b99f29d288d05edaab4614a5355de4c93d29eb8727df852c21ab14a` |
| `frontend/src/lib/components/AdminDataManagement.test.ts` | `b6b009f88c8acdb6e75e73df656a18d16ed4397207bbe9a84a36d4a69ed94b3f` |
| unchanged-by-Task-277-repair `docs/implementation/02_TASK_LIST.md` | `41d5c06f705630e8d734195c3b60505f2dd825e9d7a95bdb8a68211440974679` |

The evidence file hash is intentionally omitted from its own body to avoid a self-referential unstable value.

## Risks and blockers

- Blockers: none.
- The v1 operator deliberately fixes the canonical micronutrient vocabulary to the eight keys seeded by migration 4. A future vocabulary expansion requires an explicit operator/schema version update; unknown keys fail safely today.
- Portable classification matching uses collapsed whitespace plus Unicode case-folding. Environments containing two names that normalize alike are rejected as ambiguous rather than guessed.
- HTTP is accepted for local development/fake-server operation. Production operators must use HTTPS as documented; credentials in URL user-info are rejected.
- The operator does not provide rollback across already successful per-item API transactions. Its safety model is preflight plus deterministic idempotent continuation; a future large offline ingestion design must define staging and rollback separately.

## Independent-review repair

### Repair baseline and preservation

- Repair baseline commit: `f9a646ffede5c8a63ad787a07eaba7efc0433677`.
- Repair baseline status remained dirty with prior Phase 08 work. Task 277 operator/evidence files were untracked and the frontend administration files were already modified; all unrelated changes were preserved.
- The task list was independently modified before this repair. Its repair-baseline SHA-256 was `41d5c06f705630e8d734195c3b60505f2dd825e9d7a95bdb8a68211440974679`; this repair did not edit it.
- Repair-baseline SHA-256 values:
  - `scripts/import-global-catalog.py`: `5421685fde1b50a0f670d8efa78cf1be4b07e43d37458aa1243009bccfe18e99`
  - `scripts/test_import_global_catalog.py`: `53700da5261c73f5a351217903305784d1e34a7359e8edbb468cf896723ec004`
  - `frontend/src/lib/components/AdminDataManagement.svelte`: `375bf03b2ca4d8ba48020a082259e22c50c8cb1ba8e0ebc9f23d698f5bafe0ea`
  - `frontend/src/lib/components/AdminDataManagement.test.ts`: `7e121a18099da6ec2e453c009a33b33a816a498bb7236241737a19f0363a856e`
  - `docs/implementation/evidence/task-277-preparation.md`: `61bd227f3a7279ec2c80080da3177d1464b12a3d828a726082ae53fa38872d9d`
- Reference confidence: **HIGH**. Each independent-review finding was reproduced by a failing focused regression before its implementation was changed; focused tests, the aggregate quick gate, and the production frontend build then passed.

### Repair-owned changed paths and symbols

- `scripts/import-global-catalog.py`
  - Added constant `MAX_NUTRITION_VALUE`.
  - Added executable symbol `reject_nonfinite_number`.
  - Modified executable symbols `load_document`, `finite_nonnegative`, `validate_item`, `validate_document`, `APIClient.create`, and `run`.
- `scripts/test_import_global_catalog.py`
  - Added/modified executable test symbols `ValidationTests.test_malformed_runtime_types_and_nonfinite_numbers_raise_only_catalog_errors`, `ValidationTests.test_unexpected_operator_failure_is_sanitized_without_traceback`, `PreflightTests.test_uppercase_classification_uuid_resolves_and_is_canonicalized`, and `RetryTests.test_5xx_honors_valid_bounded_retry_after_before_identical_retry`.
- `frontend/src/lib/components/AdminDataManagement.svelte`
  - Added `allergenOptions`; modified `emptyForm` and `applyItem`; added the `form.allergenKeys` multi-select binding used by both create and update submission.
- `frontend/src/lib/components/AdminDataManagement.test.ts`
  - Added the executable regression `initializes, loads, edits, and submits the required allergen key contract`.
- `docs/implementation/evidence/task-277-preparation.md`
  - Updated paths, symbols, commands/results, criteria coverage, risks, and hashes. No other path was changed by this repair.

### Independent-review criteria coverage

| Finding | Repair and regression evidence |
|---|---|
| F1 malformed JSON/type/overflow failures | Non-standard `NaN`/infinities are rejected during parsing; numeric values are finite and bounded; type checks precede set membership, comparisons, sums, and deduplication; classification UUIDs are canonicalized before deduplication; imported density evidence is type checked; malformed top-level shapes fail as `CatalogError`; the process boundary converts unexpected exceptions to one fixed, body-free operator error. Tests cover unhashable state/allergen/classification values, invalid density/provenance types, a 10,001-digit integer, all non-finite constants, malformed roots, duplicate JSON keys, and fixed sanitized stderr. |
| F2 missing frontend `allergenKeys` | New forms initialize `allergenKeys: []`; loaded items copy `item.allergenKeys`; the administration form binds a canonical allergen selection to `form.allergenKeys`; existing `parseAdminItemForm` includes the field for create and replace requests. Component and workflow regressions pass, and production TypeScript/build validation catches contract omissions. |
| F3 `Retry-After` on 5xx ambiguity | A retryable 5xx now parses and sleeps for a valid bounded `Retry-After` before the next paced attempt. The regression proves a 503/201 sequence sleeps four seconds and reuses the identical idempotency key and body. Existing tests retain the 30-second bound. |
| F4 uppercase classification UUID | Local validation canonicalizes operator UUIDs using `uuid.UUID`, rejects nil/canonical duplicates, and passes lowercase canonical IDs into preflight/request bodies. The regression resolves a valid uppercase UUID against the lowercase active response and asserts canonical output. |

### Repair verification

Commands ran on 2026-07-27.

| Command | Result |
|---|---|
| `python3 -m unittest scripts/test_import_global_catalog.py` | PASS, 15 tests. |
| `python3 -m py_compile scripts/import-global-catalog.py` | PASS. |
| Focused Bun component/workflow tests | PASS, 11 tests. |
| `python3 scripts/check.py --quick` | PASS in 73.1 seconds: 15 operator tests, 24 generator tests, 534 frontend tests, frontend typecheck, changed backend packages including live repository integration, Go formatting/vet/vulnerability scan, OpenAPI, generated drift, task-list and traceability validation. |
| `bun run build` (`frontend/`) | PASS, 219 modules transformed. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS, 286 sequential tasks. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS with the pre-existing OAuth callback 302-only warning. |
| `git diff --check` | PASS before and after evidence update. |

### Repair risks and blockers

- Blockers: none.
- The frontend allergen selector reflects the seven canonical v1 keys. Backend request validation and atomic repository validation remain authoritative if a key is inactive; vocabulary expansion still requires an explicit contract/operator revision as already documented.
- The broad process-boundary exception handler deliberately emits only `operator request or report failed`; this prevents malformed input or dependency exceptions from exposing request bodies, credentials, or internal diagnostics on operator output.

## Second re-review repair

### Baseline and preservation

- Baseline commit: `f9a646ffede5c8a63ad787a07eaba7efc0433677`.
- The worktree was already dirty with prior Phase 08 work. The repair preserved every unrelated tracked and untracked path.
- `docs/implementation/02_TASK_LIST.md` was already modified before this repair. Its baseline and final SHA-256 are both `41d5c06f705630e8d734195c3b60505f2dd825e9d7a95bdb8a68211440974679`; this repair did not edit task status or task-list content.
- Repair-baseline SHA-256 values:
  - `backend/internal/httpapi/manual_item_controller.go`: `4e9fcf3e5a9113f5d34fa71203c4e74b5446be254b48a30d0563f647794519e0`
  - `backend/internal/httpapi/manual_item_controller_test.go`: `e60e442d6865b6a027228fe28aedc3bd83a6b28de89d955744f84e4cda25c253`
  - `backend/internal/seed/development.sql`: `d64ce7970576eee75899d290b792d36258f9a9d102cecf1014feacf14bd450e3`
  - `backend/internal/seed/seed_test.go`: `c1779635ed57c6fa310c5a0e3bd53768d9520b11fef1039295229c6030f99fae`
  - `frontend/src/lib/api/admin-client.ts`: `71b76d457ff4c8379cdabab6b5e3cb6f56cf0e22ab9249abe06e3dd8678be880`
  - `frontend/src/lib/api/admin-client.test.ts`: `311f17d57883e572b584169bd2c71ebdfedff466e9eb3e437168369c7c19cbb5`
  - `frontend/src/lib/admin-workflows.ts`: `ff2db65f18a045eae9ab7017851ab9162739744dd7aeea800f74a36fbe41ad24`
  - `frontend/tests/admin-data-management.spec.ts`: this path was identified after the first full-gate run; no pre-repair hash was captured, so baseline hash confidence for this one path is unavailable rather than inferred.
  - `scripts/check.py`: `fcdc83a3782a959d7963ff8c552db7ea08e38d2fff03dd7198194fedbc6b0cf6`
  - `scripts/test_check_coverage.py`: `7dc23ddfced75d9af3204eab18fd1185939d79b6990956c6efad4b92202a3349`
  - `docs/implementation/evidence/task-277-preparation.md`: `83ed9bdb2274a5ee4610ea9160fc97d229cc3301a9efeea00a89acddb20f6e6c`
- `docs/implementation/04_OPEN.md` was inspected at baseline but inadvertently omitted from the baseline hash command. Its exact changes are reviewable in the diff; no baseline hash is invented.
- Reference confidence: **HIGH** for each reported defect and repair. All three blockers were reproduced; focused unit, PostgreSQL, frontend-decoder, browser, coverage-contract, and aggregate tests then passed. Baseline hash confidence is explicitly unavailable only where noted.

### Changed paths and executable symbols

- `backend/internal/httpapi/manual_item_controller.go`
  - Modified `manualItemData` to emit required fields and `allergenKeys`, but omit absent optional positive numeric and nonempty string fields.
- `backend/internal/httpapi/manual_item_controller_test.go`
  - Added `TestManualItemDataOmitsAbsentOptionalFields`.
- `backend/internal/seed/development.sql`
  - Added the transaction-local `seed_classification_uuid_repairs` mapping and reference-preserving migration statements for classification hierarchy, global/custom food assignments, meal assignments, and classification audit entity IDs.
  - Replaced all four development classification fixture IDs and their assignment references with deterministic RFC 4122 version-4/variant-compatible UUIDs.
- `backend/internal/seed/seed_test.go`
  - Added `TestDevelopmentClassificationIDsMatchPublicUUIDContract` and `insertLegacyClassificationFixture`.
  - Modified `TestRunIsIdempotentAndSeedsRepositoryFixtures` to prove legacy-reference migration, canonical IDs, and repeatable seeding.
- `frontend/src/lib/api/admin-client.test.ts`
  - Added `accepts omitted optional item fields and rejects zero or empty placeholders`.
  - Added `decodes every canonical development-seed classification UUID`.
  - Production `admin-client.ts` was intentionally unchanged because its strict optional-field and UUID decoder behavior was already correct.
- `frontend/tests/admin-data-management.spec.ts`
  - Updated manual-item response fixtures in `keyboard cancellation restores focus for every destructive confirmation and uses a safe fallback`, `item replacement preserves all fields and renders the differing authoritative follow-up`, `confirmation target cannot race mutable item state`, and `older item reads and user lookups cannot overwrite newer state` so required `allergenKeys` traverse the production browser contract.
- `docs/implementation/04_OPEN.md`
  - Remeasured the Task 277 frontend and backend coverage source-of-truth, added the exact compliance repository row, and recorded the resolved portions of the optional-field and seed-UUID review findings while retaining their separate unresolved UX portions.
- `scripts/check.py`
  - Modified `validate_phase07_go_coverage` so exact coverage evidence tolerates ordinary Markdown table alignment whitespace without weakening path/line/function/percentage equality.
- `scripts/test_check_coverage.py`
  - Added `Phase08BackendCoverageContractTests.test_task277_backend_metrics_are_current` and `FrontendCoverageContractTests.test_task277_frontend_metrics_are_current`.
  - Modified `test_phase07_exact_functions_come_from_isolated_package_profiles` to regress aligned Markdown table cells.

### Criteria coverage

| Re-review blocker | Repair and evidence |
|---|---|
| Optional response omission/null semantics | `manualItemData` no longer serializes absent optional fields as invalid zero or empty placeholders. Required empty collections remain present, including `allergenKeys`. Go projection tests and strict production-client tests cover omission, valid present values, and rejection of explicit invalid placeholders; browser fixtures exercise the same required contract. |
| Canonical seed classification UUIDs | Four deterministic fixtures now use version-4 and RFC-variant bits. The idempotent seed transaction migrates every known classification reference before deleting legacy rows. PostgreSQL integration runs seed twice and proves preserved assignment/no legacy IDs; the production frontend decoder accepts every canonical fixture. |
| Full-gate coverage metadata | Frontend metadata now records `admin-workflows.ts 83.33%/98.55%` and `admin-client.ts 95.95%/100.00%`. Backend metadata records `4638/4979 (93.2%)`, exact Task 277 rows, and `compliance_repository.go 277/280`. The Phase 07 parser accepts formatting whitespace but still matches the complete exact tuple. Dedicated metadata regressions and the complete aggregate gate pass. |

### Commands and results

Commands ran on 2026-07-27.

| Command | Result |
|---|---|
| Focused Go HTTP test for `TestManualItemDataOmitsAbsentOptionalFields` | Initially failed because `averageUnitWeightGrams` was emitted as `0`; PASS after repair. |
| `go test ./internal/seed -run 'TestDevelopmentClassificationIDsMatchPublicUUIDContract|TestRunIsIdempotentAndSeedsRepositoryFixtures' -count=1 -v` | Initially rejected the legacy seed UUID; final PASS against isolated PostgreSQL, including two-run idempotency and migrated references. |
| Focused Bun admin-client/component/workflow tests | PASS; strict omission/placeholder and all four canonical seed UUID regressions included. |
| `bun test --coverage` | PASS, 536 tests, 2,803 expectations; all files `95.19%` functions / `96.06%` lines; exact Task 277 rows recorded above. |
| Focused `frontend/tests/admin-data-management.spec.ts` Playwright run | PASS, 18 desktop/mobile cases after required allergen fixture repair. |
| `python3 -m unittest scripts/test_check_coverage.py` | PASS, 21 tests. |
| Direct `validate_phase07_go_coverage` | PASS with exact documented package/function exceptions. |
| Direct `validate_phase08_go_coverage` over fresh `phase08-coverage.out` | PASS, `4638/4979 (93.2%)`. |
| First `python3 scripts/check.py` repair run | Interrupted after exposing stale browser fixtures that omitted required `allergenKeys`; those fixtures were repaired. |
| Second `python3 scripts/check.py` repair run | Reached backend coverage and exposed brittle Phase 07 Markdown whitespace matching, then the stale Phase 08 backend inventory; both source-of-truth defects received focused regressions. |
| Final `python3 scripts/check.py` | PASS in 372.6 seconds for the backend lane; all static, frontend build/unit/coverage, browser (309 passed, 5 skipped), integration, race, vulnerability, OpenAPI, generated-contract, traceability, and coverage-contract lanes passed. |

### Final repair fingerprints

| Path | Final SHA-256 |
|---|---|
| `backend/internal/httpapi/manual_item_controller.go` | `9611e4b4303c2f10b2508a7d88bf592257d7f76a5b31e973712c5a80aef5b01a` |
| `backend/internal/httpapi/manual_item_controller_test.go` | `9ad6c9c34d0d103cf634044a9269e5aee96027992f0b2887f93b4c64d4152466` |
| `backend/internal/seed/development.sql` | `8948996e646c4f89c96c3bcda1ea9a85a883d8a3fc467998403def263e726dbe` |
| `backend/internal/seed/seed_test.go` | `183e8d53fd28b1127173d7dd48a9f5305b0d6302b7004f4e626a945cab68e53c` |
| `frontend/src/lib/api/admin-client.test.ts` | `68ab30fbefd4b3ed5b014d0168f96170809fdad82d39b9d1577aacefff973884` |
| `frontend/tests/admin-data-management.spec.ts` | `b80676a865c30017fb40ed847ab5658a421d9d0d0e88703df69ee9e27d41f4db` |
| `docs/implementation/04_OPEN.md` | `dd4cf76b93601b05b4c570132c4a7c62af290e4f23c346d95419c13ee60a9584` |
| `scripts/check.py` | `b7618066a073b6b406c4a841c2a537f211462200319048a4bb9c5c3f286e856c` |
| `scripts/test_check_coverage.py` | `4e6bbf9b012b625a8a69b8d79d54a1775e2103d013afa69c2b4a0c0501b2040d` |
| unchanged-by-repair `docs/implementation/02_TASK_LIST.md` | `41d5c06f705630e8d734195c3b60505f2dd825e9d7a95bdb8a68211440974679` |

The evidence file hash is intentionally omitted from its own body to avoid a self-referential unstable value.

### Risks and blockers

- Blockers: none.
- The seed repair is intentionally confined to deterministic development fixtures. It migrates every current foreign-key/audit reference surface atomically; it is not a production classification-ID migration.
- Optional fields remain represented by Go zero values internally, so response omission is an explicit boundary responsibility and is regression-locked in `manualItemData`.
- Coverage exceptions remain measured evidence, not waived Task 277 behavior; the full behavioral and integration gates pass.

## Final F1 importer repair

### Baseline and preservation

- Repair date: 2026-07-28.
- Baseline commit remained `f9a646ffede5c8a63ad787a07eaba7efc0433677`.
- The pre-existing dirty Phase 08 worktree was preserved. This repair changed only the importer, its focused tests, and this evidence file.
- Baseline SHA-256:
  - `scripts/import-global-catalog.py`: `db296a12e9ce291edf192127453f9cb181710b34c4072289d837512201432823`
  - `scripts/test_import_global_catalog.py`: `0b1724a3e9fa95783595e3efc4f9e31cbbc7726069595d898786936e6e454e63`
  - `docs/implementation/evidence/task-277-preparation.md`: `3ed1bd799b4f919486453da818cecd510b79e3f447462eef1b8f5139fd07b68e`
  - `docs/implementation/02_TASK_LIST.md`: `41d5c06f705630e8d734195c3b60505f2dd825e9d7a95bdb8a68211440974679`
- The task-list final hash is unchanged at `41d5c06f705630e8d734195c3b60505f2dd825e9d7a95bdb8a68211440974679`; no task status or task-list content was edited.
- Reference confidence: **HIGH**. Both reported failures and sibling optional-field gaps were reproduced through the real `run` pre-authentication boundary before implementation changes.

### Changed paths and executable symbols

- `scripts/import-global-catalog.py`
  - Added constants `MAX_TEXT_LENGTH`, `MAX_CLASSIFICATION_NAME_LENGTH`, and `MAX_IMAGE_URL_LENGTH`.
  - Added executable symbols `bounded_string` and `valid_image_url`.
  - Modified `load_document` to preserve deliberate `CatalogError` subclasses while converting JSON parser `ValueError`/`OverflowError`, malformed encoding, and read failures to the fixed `catalog document is unreadable or invalid JSON` error.
  - Modified `validate_item` to enforce type and bounds before authentication for preparation time, all metric measures, density provenance strings/kind, both classification name/UUID alternatives, and `imageUrl`.
- `scripts/test_import_global_catalog.py`
  - Added `ValidationTests.test_optional_fields_are_typed_and_bounded_before_login_or_mutation`.
  - Added `ValidationTests.test_json_integer_digit_overflow_is_a_safe_prelogin_catalog_error`.
- `docs/implementation/evidence/task-277-preparation.md`
  - Added this baseline, symbols/paths, criteria, commands/results, hashes, and risk record.

### F1 criteria coverage

| Criterion | Repair evidence |
|---|---|
| `imageUrl: []` cannot reach POST | `valid_image_url` first requires a bounded string, rejects NUL/control/whitespace, accepts only relative or HTTP(S) URI references, requires an authority for absolute URLs, and contains parser `ValueError`. The end-to-end validation regression asserts `input`, `getpass`, and `APIClient` are never called. |
| 10,000-digit integer is sanitized | `load_document` catches parser-origin `ValueError` and `OverflowError` after allowing intentional `CatalogError` to pass through. Direct and `run` regressions assert the exact body-free invalid-JSON message and no login/client construction. |
| Every optional field is typed and bounded | The table-driven regression covers wrong types and upper bounds for `prepTimeMinutes`, all three metric measures, density provider/food ID/kind, food-category and culinary-role name/UUID alternatives, and `imageUrl`. Preparation and numeric fields share the existing finite `99_999_999.9999` bound; provenance is 200 characters, portable names 120, classification arrays 100, and image URI references 2,048. |
| No login or mutation on malformed input | Every malformed optional-field and parser-overflow case asserts zero calls to the email prompt, password prompt, and `APIClient`; therefore neither login, preflight, nor create can occur. |
| Safe duplicate/non-finite behavior retained | `CatalogError` is re-raised before the broad parser exception tuple, preserving duplicate-key and non-finite-number classifications. The complete 17-test operator suite passes. |

### Commands and results

| Command | Result |
|---|---|
| Focused new F1 tests before implementation | Reproduced eight validation escapes plus one raw 10,000-digit `ValueError`; `imageUrl: []`, oversized preparation/provenance/image values, malformed provenance, and unsupported image scheme reached authentication/preflight. |
| Focused new F1 tests after implementation | PASS, including all no-login/no-client assertions. |
| `python3 -m unittest scripts/test_import_global_catalog.py` | PASS, 17 tests. |
| `python3 -m py_compile scripts/import-global-catalog.py scripts/test_import_global_catalog.py` | PASS. |
| `python3 scripts/check.py --quick` | PASS in 75.6 seconds; operator, static/security, generated-contract, frontend unit/type, focused browser, and changed backend integration lanes passed. |
| `python3 scripts/check.py` | PASS; backend lane completed in 367.8 seconds, browser lane reported 309 passed/5 skipped, and all static, security, coverage, frontend, integration, race, OpenAPI, generated-contract, and traceability lanes passed. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS, 286 sequential tasks. |
| `git diff --check` | PASS. |

### Final fingerprints

| Path | Final SHA-256 |
|---|---|
| `scripts/import-global-catalog.py` | `c34b49f753accb403c1672688c99ac3497379346128a5a383e218f251f8dd014` |
| `scripts/test_import_global_catalog.py` | `ca68191084f7cd061c3ae641e4ae17ae43e40a10d46778abe1fd8e83412702a6` |
| unchanged-by-repair `docs/implementation/02_TASK_LIST.md` | `41d5c06f705630e8d734195c3b60505f2dd825e9d7a95bdb8a68211440974679` |

The evidence file hash is intentionally omitted from its own body to avoid a self-referential unstable value.

### Risks and blockers

- Blockers: none.
- URI-reference validation intentionally mirrors the existing manual-item contract: relative references and absolute HTTP(S) references are accepted; FTP, authority-less absolute URLs, control characters, whitespace, NULs, and over-2,048-character values fail before authentication.
- Python's interpreter-configured integer digit limit may vary, but both parser `ValueError` and `OverflowError` are safely normalized. Numeric values that parse successfully remain subject to the explicit finite upper bound.
