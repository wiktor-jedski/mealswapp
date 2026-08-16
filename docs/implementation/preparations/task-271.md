# Task 271 Preparation — Backend Security, Integration, and Functional Regression Gate

## Outcome and task control

- Task: **271 — Phase 08.01 Backend Security, Integration, and Functional Regression Gate** (`ARCH-009: AdminController`).
- Fixed baseline and current `HEAD`: `e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f`.
- Role followed: `/home/wiktor/.codex/agents/developer.toml`.
- The installed `golang-security` skill was applied in coding mode to the authenticated HTTP, hostile JSON, transaction/audit, PostgreSQL, Redis, and telemetry boundaries.
- Dependencies 264, 265, and 269 were inspected through their current implementation, preparation reports, and independent review reports. All three have `PASSED` review decisions; their task rows are `PREPARED` in the concurrently managed task list.
- `docs/implementation/02_TASK_LIST.md` was read only. Task 271 remains `OPEN`; no task status was edited.
- The shared worktree was already heavily dirty. All concurrent OpenAPI, backend dependency, frontend, generator, script, task-list, preparation, and review changes were preserved. No reset, checkout, clean, staging, commit, dependency update, migration, generated-client edit, or unrelated formatting was performed.

## Exact task-owned paths

| Path | Task 271 change |
|---|---|
| `backend/internal/app/task271_backend_regression_integration_test.go` | Added the live production-composition regression gate and its test-only helpers. |
| `docs/implementation/04_OPEN.md` | Superseded only the stale Task 262 backend coverage disposition with the exact Task 271 measurement and refreshed the existing machine-checked Phase 08 backend rows. |
| `docs/implementation/preparations/task-271.md` | Added this preparation record. |

Task 271 adds no production runtime code, SQL migration, API contract, frontend code, dependency, or task-list change. The implementation under test is owned by PASSED dependencies 264, 265, and 269.

## Design and production path

- `DESIGN-009 AdminController/ItemCurator`: verified server-derived admin role checks, audited mutation coordination, idempotent manual create, update/delete, bounded audit snapshots, and fail-closed audit persistence.
- `DESIGN-010 RequestValidator`: verified recursive duplicate-key rejection on authenticated private custom-item create and update before the custom-item service emits a lifecycle metric.
- `DESIGN-011 RedisCache/CacheInvalidator`: verified shared-generation search response and similarity caches through two independently composed production app instances.
- `DESIGN-008 ProfileController` and `DESIGN-005 RepositoryInterfaces`: verified owner-scoped private-item access, global/private separation, parameterized PostgreSQL paths, and atomic mutation/audit rollback.

`TestTask271ProductionBackendRegressionGate` calls `NewProduction` twice with the same live PostgreSQL pool and Redis service. Requests cross the production Fiber router, cookie JWT authentication, CSRF middleware, admin role gate, request validators, controller/service/repository boundaries, audited PostgreSQL transaction, Redis generation/cache stores, Catalog Search, and Substitution Search. The test database remains the fail-closed `_test` database selected by `testdatabase.Reset`; Redis uses `MEALSWAPP_REDIS_URL=redis://localhost:6379/12` during verification.

## Added executable symbols

| Symbol | Kind | Purpose |
|---|---|---|
| `TestTask271ProductionBackendRegressionGate` | integration test | Runs the complete production-path security and functional gate. |
| `task271SafeBuffer` | test type | Provides race-safe concurrent metric/log capture from production telemetry. |
| `(*task271SafeBuffer).Write` | test helper | Serializes JSON telemetry writes. |
| `(*task271SafeBuffer).String` | test helper | Returns a synchronized telemetry snapshot. |
| `task271RegisterAdmin` | test helper | Registers, promotes, and reauthenticates a live administrator. |
| `task271ItemBody` | test fixture | Builds one strict global/private item request. |
| `task271CatalogBody` | test fixture | Builds a unique Catalog Search request. |
| `task271SubstitutionBody` | test fixture | Builds a single-source Substitution Search request. |
| `task271AssertSearchNames` | test helper | Executes HTTP search and verifies exact visible item names. |
| `task271AssertCacheHit` | test helper | Proves both search paths were prewarmed in Redis. |
| `task271Generation` | test helper | Reads the shared Redis food-data generation with a bounded context. |
| `task271AssertGeneration` | test helper | Verifies exact generation advancement or non-advancement. |
| `task271AssertAuditCount` | test helper | Verifies action/entity-specific PostgreSQL audit atomicity. |
| `task271AssertCount` | test helper | Verifies fixed internal PostgreSQL count queries. |
| `task271InstallAuditFailure` | test fixture | Installs an isolated test trigger that rejects manual-create audit insertion after the item mutation runs. |
| `task271DropAuditFailure` | test cleanup | Removes the isolated trigger/function before subsequent mutations. |
| `task271AssertError` | test helper | Verifies generic error status/code/request-ID envelopes. |
| `task271CreateCustomItem` | test helper | Creates the owner-scoped control item through production HTTP. |
| `task271AssertDuplicateCustomJSON` | test helper | Sends the six create/update top-level, macro, and micronutrient duplicate-key cases and verifies unchanged persistence. |

No production symbol was added or modified by Task 271.

## Acceptance evidence

| Criterion | Production-path evidence | Result |
|---|---|---|
| Post-commit Catalog/Substitution visibility | Both paths are called twice through app instance two and report Redis `hit`; create/update/delete through app instance one advance the generation once and app two immediately observes created, renamed, and removed state. | **PASS** |
| Replay non-invalidation | Exact manual-create replay returns the same item ID, leaves shared generation unchanged, and leaves one `manual_create` audit. | **PASS** |
| Rollback non-invalidation and atomic audit | A PostgreSQL trigger rejects only the audit insert after the food mutation. The HTTP request returns safe retryable `503 dependency_unavailable`; the item, idempotency side effect, and audit roll back; generation remains unchanged. | **PASS** |
| Recursive duplicate-key rejection before dispatch | Authenticated POST and PUT each send duplicate top-level `name`, nested macro `protein`, and nested micronutrient `Sodium`. All return safe `400 invalid_json`; custom-item row count/name remain unchanged, and the production custom-item lifecycle metric count does not increase, proving the service was not dispatched. | **PASS** |
| Authorization and isolation unchanged | A non-admin manual-item mutation returns `403 forbidden` and writes no global item. A second authenticated user receives `404 not_found` for the owner's private item. | **PASS** |
| Atomic successful audit | Committed manual create/update/delete each persist exactly one entity/action audit; replay adds none. Audit counts are checked after each response. | **PASS** |
| Safe envelopes and telemetry | Every error has the stable shared envelope and server request ID. The forced PostgreSQL message, rollback item text, duplicate key, and duplicate value are absent from responses; forced audit detail is also absent from captured telemetry. | **PASS** |
| Race safety | The live gate and complete backend suite pass under `go test -race`. Telemetry capture uses a mutex because production metric/log lanes intentionally dispatch concurrently. | **PASS** |

## Security review

- **Trust boundaries:** hostile JSON enters only through authenticated, CSRF-protected HTTP mutation routes; admin identity and private-item ownership come from verified session context, never request JSON.
- **Tampering:** repeated object members are rejected from raw bytes before typed decode or service dispatch. The six live cases cover both mutation routes and every required nesting shape.
- **Elevation of privilege/information disclosure:** non-admin global mutation remains forbidden; cross-owner private reads remain not-found; error envelopes contain no database, duplicate-key, item-name, or trigger detail.
- **Repudiation/atomicity:** successful global mutations and audit rows commit together. Forced audit failure rolls back the item and claim before response and cannot trigger post-commit invalidation.
- **Injection:** task code adds no production query. Test assertions use fixed SQL or parameter binding; the isolated trigger definition contains no untrusted interpolation and is removed in cleanup.
- **Availability/concurrency:** Redis reads use a one-second context; integration database resets are serialized by the existing advisory lock; telemetry capture is synchronized; the full race detector passes.
- **Dependency security:** `govulncheck@v1.3.0` reports no called-code vulnerabilities.

Review conclusion: **PASS**. No security or behavior exception was added.

## Baseline and verification commands

Commands ran from `backend/` with repository-local `GOCACHE=$PWD/.go-cache`, `GOMODCACHE=$PWD/.go-mod-cache`, and, for live Redis suites, `MEALSWAPP_REDIS_URL=redis://localhost:6379/12`, unless stated otherwise.

| Command | Result |
|---|---|
| Baseline `go test -count=1 ./internal/app ./internal/httpapi ./internal/cache ./internal/customitem ./internal/itemcurator` | **PASS** before Task 271 changes: app `15.638s`, HTTP `2.765s`, cache `1.036s`, customitem `0.009s`, itemcurator `0.008s`. |
| `go test ./internal/app -run '^TestTask271ProductionBackendRegressionGate$' -count=1 -v` | **PASS**, live PostgreSQL/Redis test ran rather than skipped; final focused non-race run `1.67s`. |
| Initial focused `go test -race ./internal/app -run '^TestTask271ProductionBackendRegressionGate$' -count=1 -v` | **PASS** before telemetry-capture strengthening. |
| First complete `go test -race ./... -count=1` after telemetry capture was added | **FAIL as useful fixture diagnosis:** concurrent production metric/log writes exposed that the test used a plain `bytes.Buffer`. No production race was found. The capture was replaced with `task271SafeBuffer`. |
| Final focused `go test -race ./internal/app -run '^TestTask271ProductionBackendRegressionGate$' -count=1 -v` | **PASS**, `4.267s` package time; no race report. |
| `go test ./... -count=1` | **PASS** for every backend command/package. |
| `go test ./internal/... -p 1 -count=1 -coverprofile=coverage.out` and `go tool cover -func=coverage.out` | **PASS**; repository aggregate statement coverage is exactly `87.5%`. |
| `go test ./internal/... -p 1 -count=1 -coverpkg=./internal/... -coverprofile=phase08-coverage.out` | **PASS**; exact deduplicated Phase 08 runtime scope is `4,537/4,849` statements (`93.6%`). |
| Direct `validate_phase08_go_coverage` against `phase08-coverage.out` | **PASS:** `Phase 08 Go coverage below 100% with exact measured exceptions: 4537/4849 (93.6%).` |
| `python3 -m unittest scripts.test_check_coverage` from repository root | **PASS:** 17 tests. |
| `go vet ./...` | **PASS**, no findings. |
| Final `go test -race ./... -count=1` | **PASS** for every backend command/package; no race report. |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` | **PASS:** no vulnerabilities in called code; one imported-package and 18 required-module advisories are not called. |
| Go formatting check using `gofmt -l` over every backend `*.go` file | **PASS:** `All Go files formatted.` |
| `python3 scripts/validate-task-list.py` | **PASS:** 275 sequential tasks with ordered dependencies. |
| `python3 scripts/validate-traceability.py` | **PASS**. |
| `python3 scripts/validate-phase07-go-doc.py` | **PASS:** Phase 07 and Phase 08 exported Go Doc validation passed. |
| `git diff --check -- backend/internal/app/task271_backend_regression_integration_test.go docs/implementation/04_OPEN.md` | **PASS** before this preparation file was added. |

## Exact coverage disposition

- Pre-gate machine contract: `4,525/4,841` (`93.5%`), 316 uncovered statements.
- Task 271 measurement: `4,537/4,849` (`93.6%`), 312 uncovered statements.
- Delta: eight runtime statements were added by PASSED remediation dependencies; twelve additional statements are covered; uncovered scope decreases by four.
- The exact current file/range contract is in `docs/implementation/04_OPEN.md` and passes the repository validator. The reason IDs remain the existing closed catalog `B1`–`B4`; no new reason, source exception, behavior waiver, or broadened range was accepted.
- Generated `backend/coverage.out` and `backend/phase08-coverage.out` remain ignored local artifacts and are not task-owned deliverables.

## Final hashes

| Path | SHA-256 |
|---|---|
| `backend/internal/app/task271_backend_regression_integration_test.go` | `2206f5fdf67aa4b4d5c4af267c9b34191c6545fe9b1e7bdd387f711058a5494a` |
| `docs/implementation/04_OPEN.md` | `265ccbc283d4cc516a1dcb1d2372de66ddacd1999acd7a905ec58d1612e8b3ae` |

The preparation file's self-referential hash is intentionally omitted. Final hashes above were captured before adding this report; any later evidence-only correction must refresh the affected hash.
