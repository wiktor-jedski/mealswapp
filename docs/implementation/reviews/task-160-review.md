# Task 160 Review

Task ID: 160

Evidence path: `docs/implementation/reviews/task-160-review.md`

Recommended status: `PASSED`

## Checklist Summary

- PASS: Task 160 is `PREPARED` in `docs/implementation/02_TASK_LIST.md`.
- PASS: Dependencies are eligible: task 127 is `PASSED`; tasks 158 and 159 are `PREPARED`.
- PASS: `daily_diet` is now present in DESIGN-002, OpenAPI, backend search contracts, validation, parser strategy selection, normalizer, and generated frontend API types.
- PASS: Production composition wires `WithSearchUsageGate(usageLimiter)` into `POST /api/v1/search`.
- PASS: Anonymous Catalog Search remains available and does not load entitlement state or write usage.
- PASS: Free authenticated single-input Substitution within usage limit dispatches search, appends history, and records usage after completion.
- PASS: Free authenticated multi-input Substitution returns stable `entitlement_denied` metadata before search/cache/history/usage-record side effects.
- PASS: Free authenticated Daily Diet returns stable `entitlement_denied` metadata before search/cache/history/usage-record side effects.
- PASS: Free authenticated Daily Diet Alternative returns stable `entitlement_denied` metadata before search/cache/history/usage-record side effects.
- PASS: Free usage-limit denial returns stable `free_usage_limit_reached` metadata before search/cache/history side effects.

## Commands Run / Results

- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi ./internal/search ./internal/entitlement`
  - Result: PASS.
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi -run 'TestSearchControllerEntitlementGate|TestSearchWorkflowIntegrationGateGeneratedTypesAreCurrent' -count=1 -v`
  - Result: PASS. All task-160 entitlement gate tests and generated-type drift test passed.
- `python3 scripts/generate-api-types.py --check`
  - Result: PASS, `Generated API types are current.`
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/app -run 'TestNewProductionSearchRouteBlocksAnonymousSubstitutionBeforeCatalog|TestNewProductionRoutes' -count=1 -v`
  - Result: PASS. The matching production search route test passed.
- `python3 scripts/validate-task-list.py`
  - Result: PASS, `Task-list validation passed: 175 sequential tasks with ordered dependencies.`
- `python3 scripts/validate-traceability.py`
  - Result: PASS, `Traceability validation passed.`
- `npx --no-install redocly lint api/openapi.yaml`
  - Result: exit code 0, output was `This is not the package you're looking for.` rather than a normal lint report, so schema verification relied on direct inspection plus the generated API type drift check.

## Files Inspected

- `docs/implementation/02_TASK_LIST.md`
- `api/openapi.yaml`
- `docs/design/DESIGN-002.md`
- `backend/internal/app/app.go`
- `backend/internal/httpapi/router.go`
- `backend/internal/httpapi/search_controller.go`
- `backend/internal/httpapi/search_controller_test.go`
- `backend/internal/httpapi/search_validation.go`
- `backend/internal/httpapi/search_validation_test.go`
- `backend/internal/search/contracts.go`
- `backend/internal/search/parser.go`
- `backend/internal/search/parser_test.go`
- `backend/internal/security/normalizer.go`
- `backend/internal/entitlement/manager.go`
- `backend/internal/entitlement/usage_limiter.go`
- `frontend/src/lib/api/generated.ts`
- `scripts/generate-api-types.py`

## Decision Reason

The repair directly satisfies task 160. The controller now calls `checkSearchUsage` before `service.Search`, and `searchFeature` maps single-input Substitution, multi-input Substitution, Daily Diet, Daily Diet Alternative, and Catalog to the expected entitlement features. Production app wiring installs the usage limiter on the real search route.

HTTP integration tests directly cover the task criteria, including denial before search service/cache/history/usage-record side effects for multi-input Substitution, Daily Diet, Daily Diet Alternative, and free usage-limit failures. The repaired `daily_diet` contract is present across backend validation/parsing, OpenAPI/design docs, generated frontend types, and type generation.

## Repair Instructions If Rejected

None.
