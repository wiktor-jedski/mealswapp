# Task 269 Preparation — Backend Exported Go Documentation Gate

## Outcome and task control

- Task: **269 — Phase 08.01 Backend Exported Go Documentation Gate**.
- Result: **prepared directly under the `developer.toml` role**.
- Fixed baseline and current `HEAD`: `e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f`.
- Dependencies `242`, `243`, `244`, `245`, `246`, `249`, `250`, `252`, and `260` were inspected in their preparation reports and in `docs/implementation/02_TASK_LIST.md`; all are `PASSED`.
- Task 269 remains `OPEN`. This preparation did not edit `docs/implementation/02_TASK_LIST.md` or any task status.
- The task adds 44 previously missing identifier-led comments. Together with the already compliant `DefaultExternalSearchPageSize` and `ErrForbidden` comments, the gate now checks 46 exported Phase 08 constants and sentinel errors.

The worktree was shared and dirty before this preparation. Existing concurrent changes in `api/openapi.yaml`, `frontend/src/lib/admin-workflows.ts`, `frontend/src/lib/api/admin-client.ts`, `frontend/src/lib/api/generated.ts`, `scripts/generate-api-types.py`, and `scripts/test_generate_api_types.py` were preserved. Additional concurrent Task 264 and Task 270 edits appeared while verification ran; they were neither edited nor attributed to Task 269. No reset, checkout, clean, staging operation, task-status transition, generated-client rewrite, or unrelated formatting was performed.

## Design and dependency evidence

- `docs/implementation/preparations/task-243.md` and `task-244.md`: USDA and OpenFoodFacts public bounds, endpoints, and provider error vocabulary.
- `docs/implementation/preparations/task-245.md`: provider retry and warning vocabulary.
- `docs/implementation/preparations/task-246.md`: density provenance and normalization warning vocabulary.
- `docs/implementation/preparations/task-249.md`: curated-import sentinel errors.
- `docs/implementation/preparations/task-250.md`: manual-item curator sentinel errors.
- `docs/implementation/preparations/task-252.md`: bounded user lookup constants and authorization sentinel.
- `docs/implementation/preparations/task-260.md`: Phase 08 administration and external-data metric names.
- Existing adjacent traces remain byte-for-byte present: `DESIGN-009` for importer, curator, and user administration; `DESIGN-012` for external data; and `DESIGN-014` for Phase 08 observability.

## Exact changed paths

| Path | Task 269 change |
| --- | --- |
| `backend/internal/externaldata/usda.go` | Added identifier-led Go Doc for USDA configuration bounds and all provider error codes. |
| `backend/internal/externaldata/openfoodfacts.go` | Added identifier-led Go Doc for the endpoint and page-size bounds. |
| `backend/internal/externaldata/normalizer.go` | Added identifier-led Go Doc for density provenance and normalization warnings. |
| `backend/internal/externaldata/rate_limit.go` | Added identifier-led Go Doc for retry and provider warning constants. |
| `backend/internal/dataimporter/service.go` | Added identifier-led Go Doc for all import sentinel errors. |
| `backend/internal/itemcurator/service.go` | Added identifier-led Go Doc for both curator sentinel errors. |
| `backend/internal/useradmin/service.go` | Added identifier-led Go Doc for both page-size constants; the existing `ErrForbidden` comment was retained. |
| `backend/internal/observability/admin_external.go` | Added identifier-led Go Doc for all eight Phase 08 metric names. |
| `scripts/validate-phase07-go-doc.py` | Extended the existing aggregate gate to Phase 08 packages and the Phase 08 observability file; added single-declaration checks and an exact identifier boundary. |
| `docs/implementation/preparations/task-269.md` | Recorded scope, symbols, commands, criteria evidence, and risks. |

No other path is owned by Task 269.

## Documented declaration inventory

### `externaldata`

- USDA configuration: `USDAAPIKeyEnvironment`, `DefaultUSDAEndpoint`, `MaxUSDAPageSize`.
- Provider errors: `ProviderErrorInvalidInput`, `ProviderErrorNotConfigured`, `ProviderErrorRejected`, `ProviderErrorRateLimited`, `ProviderErrorUnavailable`, `ProviderErrorInvalidPayload`, `ProviderErrorResponseTooLarge`, `ProviderErrorTimeout`, `ProviderErrorCanceled`.
- OpenFoodFacts configuration: `DefaultOpenFoodFactsEndpoint`, `MaxOpenFoodFactsPageSize`.
- Density provenance: `DensitySourceImported`, `DensitySourceManual`, `DensitySourceEstimated`.
- Normalization warnings: `WarningMissingImage`, `WarningMissingMacros`, `WarningMissingMicronutrients`, `WarningMissingLiquidDensity`, `WarningUncertainUnitConversion`, `WarningSuspiciousLiquidMacroSum`.
- Retry warnings and bound: `MaxProviderRetries`, `WarningRateLimited`, `WarningUnavailable`, `WarningTimeout`, `WarningRetryExhausted`.
- Already compliant and now package-gated without modification: `DefaultExternalSearchPageSize`.

### `dataimporter`, `itemcurator`, and `useradmin`

- Import sentinels: `dataimporter.ErrMissingIdempotencyKey`, `dataimporter.ErrIdempotencyConflict`, `dataimporter.ErrProviderConflict`, `dataimporter.ErrNameConfirmation`.
- Curator sentinels: `itemcurator.ErrMissingIdempotencyKey`, `itemcurator.ErrIdempotencyConflict`.
- User administration bounds: `useradmin.DefaultPageSize`, `useradmin.MaxPageSize`.
- Already compliant and now single-declaration-gated without modification: `useradmin.ErrForbidden`.

### `observability`

- `MetricExternalProviderCalls`, `MetricExternalProviderLatency`, `MetricExternalProviderRetries`, `MetricExternalProviderQuota`, `MetricExternalNormalization`, `MetricAdminImportOutcomes`, `MetricAdminMutationOutcomes`, `MetricCustomItemLifecycleOutcomes`.

## Added or modified executable symbols

The Go changes modify comments and `gofmt` alignment only. **No Go function, method, type, variable initializer, constant expression, error text, public API, or runtime behavior was added or modified.**

The executable validator changes are:

- `validate_comment` — added; checks that the first contiguous Go Doc line starts with the exact exported identifier followed by a space or end-of-line.
- `validate_file` — modified; retains grouped declaration validation and adds exported single `const`/`var` validation.
- `main` — modified; retains Phase 07 package checks, adds all non-test Go files in `externaldata`, `dataimporter`, `itemcurator`, and `useradmin`, and adds `observability/admin_external.go`.

Validator configuration additions are `PHASE08_PACKAGES`, `PHASE08_FILES`, and `SINGLE_DECLARATION`.

## Verification commands and results

| Command | Result |
| --- | --- |
| `git rev-parse HEAD` | PASS; exactly `e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f`. |
| Dependency-row and preparation-report inspection for `242,243,244,245,246,249,250,252,260` | PASS; all dependency rows are `PASSED` and their exported vocabularies match the current declarations. |
| `python3 scripts/validate-phase07-go-doc.py` | PASS; `Phase 07 and Phase 08 exported Go Doc validation passed.` |
| `python3 scripts/validate-traceability.py` | PASS; `Traceability validation passed.` |
| `gofmt -w` on the eight task-owned Go files, followed by `gofmt -d` on the same files | PASS; final diff was empty. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./internal/externaldata ./internal/dataimporter ./internal/itemcurator ./internal/useradmin ./internal/observability` | PASS for all five packages. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -count=1 ./...` | PASS for every backend package. |
| `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` | PASS. |
| `git diff --ignore-all-space --unified=0 e8e7d7ee6b3c1fb37e159f3c8e7933a7aa53395f -- <eight Go paths>` | PASS; the only non-header additions are the 44 Go Doc comment lines. No declaration, initializer, literal, type, or error text differs from baseline. |
| `git diff --check` | PASS. |

## Criteria evidence

| Task criterion | Evidence | Result |
| --- | --- | --- |
| No missing or non-identifier-led exported Phase 08 constant/error comment | The expanded validator checks all four dedicated Phase 08 packages, the Phase 08 observability file, grouped declarations, and single declarations; all 46 target declarations pass. | PASS |
| Names, values, behavior, and public APIs unchanged | Whitespace-insensitive baseline diff contains comments only; focused and full backend tests pass. | PASS |
| Exact adjacent design traceability retained | Existing `Implements DESIGN-009`, `DESIGN-012`, and `DESIGN-014` lines were not removed or rewritten; traceability validation passes. | PASS |
| Formatting and static analysis | Task-owned `gofmt -d` is empty and `go vet ./...` passes. | PASS |
| Aggregate validation integration | `scripts/check.py` already invokes `scripts/validate-phase07-go-doc.py`; extending that script automatically extends the root static gate without another runner or dependency. | PASS |

## Risks and boundaries

- The historical validator filename still mentions Phase 07 to preserve all existing aggregate-gate and evidence references. Its module description and success output now state that it validates both phases.
- `observability` contains Phase 07 optimization constants in a separate file. Task 269 deliberately gates only `admin_external.go`, the Phase 08 observability surface, avoiding unrelated documentation churn while covering every Task 260 metric constant.
- The validator is intentionally scoped to exported constants and exported variables used as sentinel errors. It does not broaden this task into package-wide exported type/function documentation.
- No runtime risk is expected because production diffs are comments and formatter alignment only. The remaining integration risk is future Phase 08 observability constants being added outside `admin_external.go`; such a change must extend `PHASE08_FILES` or consolidate the Phase 08 surface.
