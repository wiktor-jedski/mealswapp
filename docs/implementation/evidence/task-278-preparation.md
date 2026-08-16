# Task 278 preparation evidence

## Outcome

Task 278, Phase 08.02 Global Catalog Export Operator, is implemented and verified without editing `docs/implementation/02_TASK_LIST.md`. The route is an authenticated administration-only read, the repository queries only ownerless global catalog tables in a repeatable-read/read-only PostgreSQL transaction, and the operator publishes canonical JSON or inspection CSV through a same-directory fsync-and-replace boundary.

The authoritative interchange schema remains `mealswapp.global-catalog.v1`. Export-only UUID, timestamp, deletion, image-alt, source, classification identity, and curated-source fields are informational. The Task 277 importer validates their outer representation and strips them before `POST /api/v1/admin/items`; it does not recreate `curated_imports`.

## Baseline, dependency evidence, and preservation

- Worktree: `/home/wiktor/Work/mealswapp`
- Fixed baseline commit: `f9a646ffede5c8a63ad787a07eaba7efc0433677`
- Baseline status: dirty with 52 tracked modifications and the Task 276/277 untracked implementation/evidence surfaces. The full initial `git status --short`, diff stat, and task-list diff were captured before Task 278 edits.
- Authoritative row: Task 278 in `docs/implementation/02_TASK_LIST.md`, status `OPEN`.
- Dependencies read: Task 253, 276, and 277 preparation/review evidence; Tasks 276 and 277 were already externally transitioned to `PASSED`.
- Preservation: no reset, checkout, clean, commit, migration, task transition, Account Export change, private-item change, or later-task implementation was performed. The pre-existing task-list hunk changing only Tasks 276/277 to `PASSED` remains untouched.
- Baseline confidence: **HIGH**. The complete Task 278 row, dependency evidence, implemented Task 277 operator, DESIGN-009, OpenAPI, migrated PostgreSQL schema, admin route gateway, and live repository behavior agreed. New files and in-turn patches are exact. Shared dirty-file baselines came from Task 276/277 final evidence; `backend/internal/app/app.go` was clean at turn start and was hashed from the fixed commit.

### Pre-work task-surface fingerprints

| Path | SHA-256 before Task 278 |
|---|---|
| `api/openapi.yaml` | `69397349b60bd9744effea07cc249f8a6a65e626bf3e83b8d682c89a189372fc` |
| `backend/internal/app/app.go` | `3370985462e78ec3d2a037494ffed0ed101cdf056290c62aaa41a9eaf6dccd29` |
| `backend/internal/repository/postgres_repository_test.go` | `97685f5f4db8d6820ea8761d207983b360b92d47fa06e57f2c05d09a5365e241` |
| `docs/design/DESIGN-009.md` | `22ca65baf347c8c1598a4a71439ca5091548fc3a8f9b5dc906406a3fdcc8c814` |
| `docs/operations/global-catalog-import.md` | `97256538264b77e6027ecc26d63a30c3ebc60b650da6af66ded8412446fe15d9` |
| `scripts/check.py` | `b7618066a073b6b406c4a841c2a537f211462200319048a4bb9c5c3f286e856c` |
| `scripts/import-global-catalog.py` | `c34b49f753accb403c1672688c99ac3497379346128a5a383e218f251f8dd014` |
| `scripts/test_import_global_catalog.py` | `ca68191084f7cd061c3ae641e4ae17ae43e40a10d46778abe1fd8e83412702a6` |
| All other Task 278 paths | absent |

## Implemented behavior

- `GET /api/v1/admin/catalog-export` is registered through the secure `AdminController`, requires authenticated administrator claims, is read-only, has no CSRF or mutation audit policy, and accepts only optional strict boolean `includeDeleted`.
- Active items are the default. `includeDeleted=true` includes soft-deleted global rows and their explicit deletion timestamp.
- The SQL source references `food_items`, `food_item_classifications`, `classifications`, `food_item_allergens`, and informational `curated_imports` only. It does not reference users, identities, login methods, account data, saved data, consent, subscriptions, or `custom_food_items`.
- PostgreSQL starts one transaction, sets `REPEATABLE READ, READ ONLY` before querying, orders rows by UUID, emits one row at a time, commits only after complete iteration, and uses a bounded independent rollback context after query/consumer cancellation or failure.
- Classification relationships are sorted by C-locale name then UUID, allergens by canonical key, curated identities by provider/external ID/UUID, and JSON map properties (including micronutrients) by `encoding/json` key ordering.
- Numeric PostgreSQL values scan as text and encode as canonical JSON numbers without float conversion. Optional metric/source/image/deletion fields are explicit JSON `null`.
- Every entry uses `global-catalog:<item UUID>` as stable idempotency identity. The root contains only `schema` and `items`; no generated timestamp exists.
- The document includes IDs, Unicode names, physical state, preparation time, metric unit/serving/density fields and provenance, exact macros, micronutrients, image URL/alt, direct source provider/external ID, timestamps, Food Category/Culinary Role names plus UUID/name metadata, canonical allergens, deletion state, and informational curated-source identities.
- The service writes a compact JSON prefix, one encoded item at a time, and a suffix; empty and large catalogs do not require a catalog-sized Go slice.
- Streaming is detached only from Fiber's handler-return cancellation and retains the gateway deadline. Closing the response stream cancels snapshot work.
- `global_catalog_session.py` is shared by import/export. It prompts for email and non-echoing password, rejects credentials embedded in the base URL, and uses only a process-memory `CookieJar`.
- The export operator streams 64 KiB chunks to a private random temporary file in the destination directory, flushes and fsyncs it, validates duplicate-key-safe JSON, converts to optional pretty JSON or deterministic CSV when requested, fsyncs converted content, atomically replaces the destination, and fsyncs the directory entry.
- Any authentication, authorization, HTTP, interrupted body, parse, conversion, fsync, or publication failure returns nonzero, removes temporary files, and leaves a prior destination unchanged.
- Standard output contains only output path, count, and bounded request ID. Errors are fixed safe categories and never include response bodies, names, credentials, cookies, CSRF, tokens, or email.
- The import operator accepts the exported informational fields, strips null and informational values before manual-item preflight/mutation, preserves supported metric fields, classifications, micronutrients, image URL, density provenance, and allergens, and intentionally does not recreate curated import identity.

## Task-owned changed paths

### API and production composition

- `api/openapi.yaml`
- `backend/internal/app/app.go`
- `backend/internal/httpapi/catalog_export_controller.go`
- `backend/internal/httpapi/catalog_export_controller_test.go`

### Export service and persistence

- `backend/internal/catalogexport/service.go`
- `backend/internal/catalogexport/service_test.go`
- `backend/internal/repository/global_catalog_export_repository.go`
- `backend/internal/repository/global_catalog_export_repository_test.go`
- `backend/internal/repository/postgres_repository_test.go`
- `backend/internal/repository/sql/global_catalog_export.sql`
- `backend/internal/repository/sql/testdata/global_catalog_export_fixture_create.sql`

### Operators, tests, and aggregate gate

- `scripts/global_catalog_session.py`
- `scripts/export-global-catalog.py` (new executable mode 0755)
- `scripts/test_export_global_catalog.py`
- `scripts/import-global-catalog.py`
- `scripts/test_import_global_catalog.py`
- `scripts/check.py`

### Design, runbook, and evidence

- `docs/design/DESIGN-009.md`
- `docs/operations/global-catalog-export.md`
- `docs/operations/global-catalog-import.md`
- `docs/implementation/evidence/task-278-preparation.md`

## Added or modified executable symbols

### Go production

- `catalogexport.Schema`, `catalogexport.Store`, `catalogexport.Service`, `catalogexport.NewService`
- `catalogexport.catalogEntry`, `catalogexport.catalogItem`, `catalogexport.catalogMacros`, `catalogexport.catalogClassifications`
- `catalogexport.(*Service).Write`, `catalogexport.project`, `catalogexport.number`, `catalogexport.canonicalNumber`, `catalogexport.classificationNames`, `catalogexport.timestamp`
- `repository.globalCatalogExportSQL`, `repository.globalCatalogReadOnlyTransactionSQL`
- `repository.CatalogExportClassification`, `repository.CatalogExportCuratedSource`, `repository.CatalogExportItem`, `repository.CatalogExportRowConsumer`
- `repository.PostgresGlobalCatalogExportRepository`, `repository.NewPostgresGlobalCatalogExportRepository`, `repository.(*PostgresGlobalCatalogExportRepository).Stream`
- `repository.scanGlobalCatalogExportItem`, `repository.decodeCatalogJSON`
- `httpapi.GlobalCatalogExportService`, `httpapi.CatalogExportController`, `httpapi.NewGlobalCatalogExportAdminController`
- `httpapi.(*CatalogExportController).Export`, `httpapi.validateCatalogExportQuery`, `httpapi.cancelingReadCloser`, `httpapi.(*cancelingReadCloser).Close`
- Modified composition: `app.newProduction`

### Python production

- Shared session: `SessionError`, `AuthorizationError`, `APIResponse`, `prompt_credentials`, `AuthenticatedSession`, `AuthenticatedSession.__init__`, `AuthenticatedSession.open`, `AuthenticatedSession.request`, `AuthenticatedSession.login`
- Export operator: `ExportError`, `reject_duplicate_keys`, `load_export`, `temporary_path`, `flush_and_sync`, `stream_response`, `write_json`, `json_cell`, `write_csv`, `publish`, `run`
- Import integration: `INFORMATIONAL_FIELDS`, modified `validate_item`, added `validate_informational_metadata`, modified `preflight`, `APIClient` now subclasses `AuthenticatedSession`, and modified `run`
- Aggregate gate: modified `validate_global_catalog_operator_tests` and `TRACEABLE_FILES`

### Contracts and tests

- OpenAPI operation/schema additions: `getApiV1AdminCatalogExport`, `GlobalCatalogExportDocument`, `GlobalCatalogExportEntry`, `GlobalCatalogExportItem`, `GlobalCatalogExportClassification`, `GlobalCatalogExportCuratedSource`
- Go test helpers/tests: `exportStore.Stream`, `TestServiceWritesByteIdenticalCompleteCatalog`, `TestServiceEmptyLargeCancellationAndErrorPropagation`, `catalogExportService.Write`, `TestCatalogExportRouteIsAdminOnlyAndDefaultsToActive`, `TestCatalogExportRouteShapeHasNoMutationOrAccountExportControls`, `TestGlobalCatalogExportRepositoryReadOnlySnapshotAndProjection`, `TestGlobalCatalogExportRepositoryCancellationAndFailuresRollback`, `TestPostgresGlobalCatalogExportRepositorySelectionOrderingAndPrivateIsolation`, `containsAll`; modified shared `fakeSQLExecutor.Exec` and `fakeTx.Commit` to expose transaction evidence.
- Python test helpers/tests: `OperatorHandler.do_POST`, `OperatorHandler.do_GET`, `ExportOperatorTests.setUp`, `tearDown`, `run_operator`, and six `test_*` methods covering JSON/CSV/pretty/include-deleted/auth/output safety/atomicity/failure/round-trip behavior.

## Acceptance-criterion coverage

| Task 278 criterion | Evidence |
|---|---|
| Admin-only route; anonymous 401 and user 403 | `TestCatalogExportRouteIsAdminOnlyAndDefaultsToActive`; route-shape test verifies auth/admin flags and no mutation controls. |
| Default active selection; explicit deleted selection | HTTP includeDeleted call assertions and live PostgreSQL active/complete streams. |
| Unsupported parameters/formats fail | HTTP invalid/duplicate query cases; argparse CSV+pretty and XML cases. |
| No private custom items or Account Export PII | SQL allowlist/denylist assertions, live same-database custom-item fixture, route and raw-document separation. |
| Safe logs/output | HTTP memory-log inspection and Python captured stdout/stderr assertions exclude email/password/cookie/CSRF/name/payload. |
| Read-only single snapshot and bounded stream | Repository transaction assertions, embedded SQL UUID order, consumer callback iteration, large 2,000-item service test. |
| Deterministic relationships, keys, numbers, bytes | SQL aggregate ordering; map-key JSON encoding; exact text numeric projection; service byte-identical rerun test. |
| Complete fields, Unicode, null, solids/liquids, source/image | service fixture plus live PostgreSQL liquid and deleted-solid fixtures. |
| Empty/large/cancellation/error | focused service and repository failure-table tests; pipe close cancellation boundary. |
| JSON/CSV/pretty/output/include-deleted | five fake-server operator tests and parse assertions. |
| Same-directory temp, fsync, atomic replacement, interrupted preservation | mocked temp directory/fsync/replace assertions; partial-body and 403 tests preserve existing bytes and remove temp files. |
| Shared interactive in-memory session | one imported helper used by both scripts; fake server requires interactive credentials and cookie-backed login/CSRF. |
| Round trip without false curated recreation | `test_export_round_trip_preserves_supported_fields_and_strips_curated_metadata`; runbooks explicitly state `curated_imports` are not recreated. |
| OpenAPI/generated drift/security/quality | commands below. |

## Security review

The installed `golang-security` skill was applied in coding/review mode.

- Trust boundaries: verified admin cookies/JWT claims; strict query input; PostgreSQL global catalog rows; streamed HTTP body; interactive credentials; destination path and temporary-file publication.
- STRIDE result: server-side admin middleware prevents spoofing/elevation; strict query allowlisting and canonical JSON prevent parameter/payload ambiguity; read-only repeatable snapshots and UUID-derived keys protect integrity; no user/private tables and safe output prevent disclosure; gateway deadline, 10/minute read limit, row streaming, 64 KiB client chunks, and full failure propagation bound denial-of-service; this read intentionally creates no mutation audit entry.
- SQL is static and embedded; the only parameter is boolean. No SQL concatenation, shell, subprocess, direct operator database connection, credential argument, persistent cookie, token logging, response-body logging, or weak cryptography was introduced.
- Files are created by `NamedTemporaryFile` in the destination directory, content and directory entries are fsynced, and publication uses `os.replace`. Prior destination bytes remain recoverable on all pre-publication failures.
- No open Critical/High/Medium/Low security finding remains.

## Verification commands and results

Commands ran on 2026-07-28.

| Command | Result |
|---|---|
| `python3 -m unittest scripts/test_import_global_catalog.py scripts/test_export_global_catalog.py` | PASS, 23 tests. |
| `python3 -m py_compile scripts/global_catalog_session.py scripts/import-global-catalog.py scripts/export-global-catalog.py scripts/test_import_global_catalog.py scripts/test_export_global_catalog.py` | PASS. |
| `go test -count=1 ./internal/catalogexport ./internal/repository ./internal/httpapi ./internal/app` | PASS. |
| `go test -count=1 ./internal/repository -run 'TestPostgresGlobalCatalogExportRepositorySelectionOrderingAndPrivateIsolation|TestGlobalCatalogExportRepository' -v` | PASS against isolated migrated PostgreSQL. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS with the pre-existing OAuth callback 302-only warning. |
| `python3 scripts/generate-api-types.py --check` | PASS; generated output is current. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/check.py --quick` | PASS; static, OpenAPI, generated drift, Python operators, changed backend/live PostgreSQL, frontend, and Playwright lanes passed. |
| `go vet ./...` | PASS. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS; no called-code vulnerability. |
| `go test -count=1 -race ./...` | PASS; no race report. |
| `git diff --check` | PASS. |

## Coverage and residual risks

- Focused service tests cover deterministic/empty/large/cancel/error paths; repository tests cover transaction setup, scan, selection, iteration, commit/rollback, and live projection; HTTP tests cover authorization/query/route shape; Python tests cover the complete publication boundary.
- A streaming HTTP status is committed before a late database or connection failure can be represented as a JSON error envelope. Such failures terminate the body as invalid/truncated JSON. The Task 278 operator validates the entire document before replacement and therefore fails nonzero without publishing. This is the standard streaming tradeoff and does not create false-success output.
- The API gateway deadline remains applicable to a complete export. A catalog too large for the configured deadline fails safely and requires an operator/configuration decision; Task 278 does not add an unbounded endpoint.
- Task 277 intentionally limits each import run to 500 items. A complete export larger than 500 remains valid, but re-import must use separately reviewed ≤500 documents without changing item-derived keys; this task does not design later offline ingestion.
- Curated identity is informational by explicit contract. Recreating it requires the existing dedicated curated-import workflow, not manual-item import.

## Final task-surface fingerprints

| Path | Final SHA-256 |
|---|---|
| `api/openapi.yaml` | `b70e989e5260d6ea312ae5d8f927b16f50b791a4ae18203ac90c64a5675829a9` |
| `backend/internal/app/app.go` | `a1c7347b325be1031f9978726e1e69717bdad733d5199b675a009c5b1ad479af` |
| `backend/internal/catalogexport/service.go` | `3c7ae983a2ff94f9dcbdd755deabd28f891b5ae5cd2ac5babcd54104bd05acec` |
| `backend/internal/catalogexport/service_test.go` | `13e36b7abc73a6122c8f5ea913568f2d561ee896ec80eba5d67f48ae1619a622` |
| `backend/internal/httpapi/catalog_export_controller.go` | `2d1d7babb7f613af4196a15a6cd07b197c791fb4b2d979d3e74355f86729adb3` |
| `backend/internal/httpapi/catalog_export_controller_test.go` | `2d4445700ebafddcf4de83a22dfa28a7a39568a1c871eaec2ca7a0f373295cdf` |
| `backend/internal/repository/global_catalog_export_repository.go` | `96023562a6a2c3357e8324ee25932236f6c4dde392abf8aa65df9aa09d070c25` |
| `backend/internal/repository/global_catalog_export_repository_test.go` | `c6185145c068ad7b5b4c0d1dc85fa8df2aeb5bdda889cf411d6addf80be6ee28` |
| `backend/internal/repository/postgres_repository_test.go` | `6366a0d84980807b4a80113969436b2207b542a72520b1767e4fb7037ad2cbe4` |
| `backend/internal/repository/sql/global_catalog_export.sql` | `7286a00106375503a221268a2898fd9c565d0930708ac29e6468a6f553aee1dd` |
| `backend/internal/repository/sql/testdata/global_catalog_export_fixture_create.sql` | `e14e819b935a954c94504f5b69969c0b674df11b14da4acd7212c1bd33222f10` |
| `docs/design/DESIGN-009.md` | `704500175a5e465ee6d10cc1edd421f4c7c6043dfff1cd8c444a28e1d75973ea` |
| `docs/operations/global-catalog-export.md` | `8e93184cfdb5fbd480eb4e1fa91460f95de6564bb0e598d565e05907c822d207` |
| `docs/operations/global-catalog-import.md` | `548cca6e5648249bafeb4027570e2670f3300d7036c7ab94dde76c524bf7d469` |
| `scripts/check.py` | `17d0f799050382be5a8f5f95e0ed9e731e09031f6ca5b2704a1d3487aea77a8b` |
| `scripts/export-global-catalog.py` | `dd6f6c04c1715e93efbd41b79860afbc401c8ea051fccf596b321a952e7b32d7` |
| `scripts/global_catalog_session.py` | `972b0d361f7642c8f9a8bf31e582600bb0a0f6b571d6ab671eb3ac8e8da1766f` |
| `scripts/import-global-catalog.py` | `d5758aff8147d4bbabcae0ad282b9d260c1ec4e2d43da4c5da25a2e854d00de5` |
| `scripts/test_export_global_catalog.py` | `02ca8c1b67ac0d9a990aa0d6ed1a27e4890a092c1b0b3f214c13a20062ee0995` |
| `scripts/test_import_global_catalog.py` | `c88e9ae64ba1514a90bb1164aa3f8bfb0585ced3fdcf1cace7dd6a63543b7e61` |

The evidence file's self-hash is intentionally omitted because including it would be self-referential. All task-owned non-evidence paths are covered above.

## 2026-07-28 review repair

### Repair scope, baseline, and preservation

This repair addressed only findings F1-F4. No route, repository, database, generated contract, importer, shared session, or later-task behavior was changed. Task 278 remained `PREPARED`; `docs/implementation/02_TASK_LIST.md` was not edited during the repair.

- Repair baseline commit: `f9a646ffede5c8a63ad787a07eaba7efc0433677`.
- Repair baseline status: the preserved Phase 08.02 worktree was dirty and contained the externally prepared Task 278 implementation plus prior Task 276/277 changes.
- Repair baseline confidence: **HIGH**. The full authoritative Task 278 row, Tasks 253/276/277 dependency state and evidence, original Task 278 preparation evidence, implementation, tests, and review findings were read before changes. The four findings were reproducible from the operator code and existing test omissions.
- Preservation: no reset, checkout, clean, commit, migration, task-list edit, backend/API change, import behavior change, or later-task implementation was performed.

| Path | SHA-256 before repair |
|---|---|
| `scripts/export-global-catalog.py` | `dd6f6c04c1715e93efbd41b79860afbc401c8ea051fccf596b321a952e7b32d7` |
| `scripts/test_export_global_catalog.py` | `02ca8c1b67ac0d9a990aa0d6ed1a27e4890a092c1b0b3f214c13a20062ee0995` |
| `docs/operations/global-catalog-export.md` | `8e93184cfdb5fbd480eb4e1fa91460f95de6564bb0e598d565e05907c822d207` |
| `scripts/global_catalog_session.py` (unchanged control) | `972b0d361f7642c8f9a8bf31e582600bb0a0f6b571d6ab671eb3ac8e8da1766f` |
| `api/openapi.yaml` (unchanged control) | `b70e989e5260d6ea312ae5d8f927b16f50b791a4ae18203ac90c64a5675829a9` |
| `docs/implementation/evidence/task-278-preparation.md` | `2d0732efa281e14466dfae3473172652a4217e8a2b194b93e8c75925dc059f0b` |

### Finding closure and criterion coverage

| Finding | Closure and evidence |
|---|---|
| F1: missing CSV density source kind | `CSV_COLUMNS` and `write_csv` now emit `densitySourceKind` beside the other density provenance columns. `test_csv_is_deterministic_and_parseable` verifies the complete header contract, exported value, and parser result; `test_export_round_trip_preserves_supported_fields_and_strips_curated_metadata` verifies imported density provenance survives the export/import projection. The export runbook lists all three provenance columns. |
| F2: destination loss after directory fsync failure | `publish`, `restore_destination`, and `sync_directory` retain a same-directory fsynced backup when a destination already exists, replace only after successful output creation, and restore the prior bytes if publication-directory fsync fails. With no prior destination, failed publication removes the unconfirmed new file. `test_directory_fsync_failure_restores_prior_destination_or_removes_new_file` injects both failures and verifies prior-byte preservation, nonzero status, safe diagnostics, and no leaked temporary files. |
| F3: permissive/lossy JSON validation | `load_export` parses fractional numbers as `Decimal`, rejects duplicate keys and non-standard `NaN`/positive or negative infinity, and validates the closed root/entry/item representation before publication. `validate_entry` and its validator helpers enforce canonical UUID/idempotency identity, bounded exact finite numbers, physical/density rules, canonical keys, deterministic arrays/relationships, curated metadata, and UTC timestamps. `write_json_value`/`write_json_collection` preserve exact decimal tokens in pretty JSON and CSV instead of converting through binary floats. `test_exact_numeric_precision_and_adversarial_json_validation` proves long-decimal preservation through compact operator output, pretty JSON, and CSV, and rejects non-finite, duplicate, unknown, mismatched, boolean, and negative adversarial values. |
| F4: response leak when temp creation fails | `stream_response` now owns the HTTP response immediately after `open`, creates the temporary file inside a `try`, closes the response in `finally`, and removes any created temporary file on every body/write failure. `test_temp_creation_and_body_read_failures_close_response_and_cleanup` injects temp creation and body read failures and verifies body closure and cleanup. |

### Repair-changed paths and executable symbols

- `scripts/export-global-catalog.py`
  - Modified constants: `CSV_COLUMNS`.
  - Added constants: `ALLERGEN_PATTERN`, `MAX_NUMBER`, `MICRONUTRIENT_KEYS`, `ITEM_FIELDS`.
  - Added symbols: `reject_nonfinite_number`, `validate_entry`, `valid_text`, `finite_number`, `validate_number_map`, `validate_sorted_strings`, `validate_classifications`, `validate_curated_sources`, `validate_timestamp`, `write_json_value`, `write_json_collection`, `restore_destination`, `sync_directory`.
  - Modified symbols: `load_export`, `stream_response`, `write_json`, `json_cell`, `write_csv`, `publish`.
- `scripts/test_export_global_catalog.py`
  - Modified fixture `DOCUMENT`.
  - Modified tests: `test_csv_is_deterministic_and_parseable`, `test_export_round_trip_preserves_supported_fields_and_strips_curated_metadata`.
  - Added tests: `test_directory_fsync_failure_restores_prior_destination_or_removes_new_file`, `test_exact_numeric_precision_and_adversarial_json_validation`, `test_temp_creation_and_body_read_failures_close_response_and_cleanup`.
- `docs/operations/global-catalog-export.md`
  - Documented the CSV `densitySourceKind` contract, strict exact-number validation, response cleanup, and backup/restore publication boundary.
- `docs/implementation/evidence/task-278-preparation.md`
  - Added this repair baseline, finding closure, symbols, verification, risks, and final fingerprints.

### Repair verification results

| Command | Result |
|---|---|
| `python3 -m unittest scripts/test_import_global_catalog.py scripts/test_export_global_catalog.py` | PASS, 26 tests in 6.336 seconds. |
| `python3 -m py_compile scripts/export-global-catalog.py scripts/test_export_global_catalog.py` | PASS. |
| `go test ./internal/catalogexport ./internal/httpapi ./internal/repository -run 'CatalogExport\|GlobalCatalog' -count=1` | PASS. |
| `go vet ./internal/catalogexport ./internal/httpapi ./internal/repository` | PASS. |
| `npx --no-install redocly lint api/openapi.yaml` | PASS with the pre-existing OAuth callback 302-only warning. |
| `python3 scripts/generate-api-types.py --check` | PASS; generated output is current. |
| `python3 scripts/validate-traceability.py` | PASS. |
| `python3 scripts/validate-task-list.py` | PASS; 286 sequential tasks with ordered dependencies, Task 278 still `PREPARED`. |
| `python3 scripts/check.py --quick` | PASS in 77.5 seconds; all static, contract, Python operator, backend, frontend, and Playwright changed-area lanes passed. |
| `go test -race ./internal/catalogexport ./internal/httpapi ./internal/repository -run 'CatalogExport\|GlobalCatalog' -count=1` | PASS; no race report. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | PASS; no called-code vulnerability. |
| `git diff --check` | PASS after the final evidence update. |

### Repair security review and residual risks

The installed `golang-security` guidance was reapplied to the filesystem, untrusted-JSON, HTTP-body, and diagnostic boundaries. Exact closed-schema validation now fails before destination publication; no response payload or rejected value is included in diagnostics. HTTP bodies are closed even when local file allocation fails. Same-directory random private temporary files, file fsync, atomic replacement, directory fsync, and restoration preserve prior bytes for the injected durability failure. No credential, cookie, token, email, item name, or payload logging was added. No open security finding remains.

- A directory fsync failure means the operating system cannot confirm durable directory metadata. The operator therefore returns nonzero even after restoring the prior logical destination; durability across an immediate system crash remains dependent on the filesystem honoring a later sync.
- Validation intentionally accepts only the current closed `mealswapp.global-catalog.v1` representation and canonical key sets. A future schema extension must version or deliberately update this validator rather than being silently accepted.
- Compact JSON without conversion remains the server's validated byte representation; pretty JSON and CSV are deterministically re-encoded from exact `Decimal` values.

### Repair final fingerprints

| Path | SHA-256 after repair |
|---|---|
| `scripts/export-global-catalog.py` | `df418cfafa9d3112e6b84c93f57a3c533654ffad7780a4af3786721a608f7f84` |
| `scripts/test_export_global_catalog.py` | `98f3d75619b0884646e524ffcd4023c318881ac4efa65d74959453a378f3dd54` |
| `docs/operations/global-catalog-export.md` | `49f2e34918d38c0ad568ea1c246879775bba076cd92a770e82b82f31042d6fab` |
| `scripts/global_catalog_session.py` (unchanged control) | `972b0d361f7642c8f9a8bf31e582600bb0a0f6b571d6ab671eb3ac8e8da1766f` |
| `api/openapi.yaml` (unchanged control) | `b70e989e5260d6ea312ae5d8f927b16f50b791a4ae18203ac90c64a5675829a9` |

The repair evidence file's self-hash is intentionally omitted because including it would be self-referential.
