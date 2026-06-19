# Task 138 Review — Phase 04 Search Contract and Cleanup Follow-up

**Static Aspect:** DESIGN-002: SearchController
**Status reviewed:** PREPARED
**Dependencies:** 115,116,117,118,120,121,122,123,124,125,127,128,129,135,136,137 — all PASSED
**Reviewer verdict:** PASSED

## Preconditions

- Task 138 status is PREPARED in `docs/implementation/02_TASK_LIST.md` (line 145). PASS
- All 16 dependencies (115-137) are PASSED in the task list. PASS

## Checklist (from Verification Criteria)

- PASS - C1: Search service/HTTP tests verify every result exposes classifications as `id`, `name`, `kind`. `TestSearchControllerFoodObjectDTOExposesClassificationMacrosAndCalories` asserts category (`id`/`name`/`kind=food_category`) and role (`id`/`name`/`kind=culinary_role`) on the solid item and an empty classifications array on the liquid item.
- PASS - C2: Tests verify an explicit primary Food Category. Same test asserts `primaryFoodCategory` equals the first food category for the solid item and is `nil` for the liquid item with no categories.
- PASS - C3: Tests verify non-negative protein/carbohydrate/fat macros. Test asserts each macro `>= 0` and exact values; OpenAPI `MacroProfile` sets `minimum: 0` on protein/carbohydrates/fat.
- PASS - C4: Tests verify a physical-state-consistent `100g` or `100ml` basis. Test asserts `macroBasis == "100g"` for solid and `"100ml"` for liquid; `macroBasisForState` derives the basis from `repository.PhysicalState`.
- PASS - C5: Tests verify non-negative server-calculated calories. Test asserts `calories >= 0` and `calories == protein*4 + carbs*4 + fat*9` for both items; OpenAPI `calories` sets `minimum: 0`; `search.CalculateCalories` implements the 4/4/9 rule.
- PASS - C6: OpenAPI lint passes. `npx --no-install redocly lint api/openapi.yaml` reported "valid" with no errors.
- PASS - C7: Frontend type regeneration exposes fields without handwritten duplicates. `bun run check:api-types` reported "Generated API types are current." Grep confirms `ClassificationSummary`, `MacroProfile`, `FoodObject` and the new fields exist only in `frontend/src/lib/api/generated.ts`.
- PASS - C8: `python3 scripts/validate-task-list.py` passed: "154 sequential tasks with ordered dependencies."
- PASS - C9: `python3 scripts/validate-traceability.py` passed: "Traceability validation passed."
- PASS - C10: Each other selected `04_OPEN.md` action point is implemented or intentionally split. The only documented Phase 04 action point in `04_OPEN.md` is the FoodObject OpenAPI/generated-type extension, which is implemented. The other items in the task description (naming cleanup, similarity presentation cleanup, deterministic ordering cleanup, test hygiene) are not recorded as `04_OPEN.md` action points and are treated as intentionally split/out of scope for this task.

## Commands

- `npx --no-install redocly lint api/openapi.yaml` -> valid, no errors
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/httpapi/... ./internal/search/...` -> ok (httpapi), ok (search)
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` -> no issues
- `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/search/... -count=1` -> ok (2.996s)
- `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun run check:api-types` -> "Generated API types are current."
- `cd frontend && BUN_TMPDIR=$PWD/.bun-tmp BUN_INSTALL=$PWD/.bun-install bun test` -> 172 pass, 0 fail
- `python3 scripts/validate-task-list.py` -> passed
- `python3 scripts/validate-traceability.py` -> passed
- `gofmt -l backend/internal/httpapi/search_controller.go backend/internal/search/substitution_service.go` -> clean
- `gofmt -l backend/internal/httpapi/search_controller_test.go` -> flagged (struct field alignment drift; see Notes)

## Files inspected

- `api/openapi.yaml` (lines 891-955) — new `ClassificationSummary`, `MacroProfile` schemas; `FoodObject` extended with `classifications`, `primaryFoodCategory`, `macros`, `macroBasis`, `calories`; `minimum: 0` constraints on macros and calories; `macroBasis` enum `[100g, 100ml]`.
- `backend/internal/httpapi/search_controller.go` — added `classificationSummaryDTO`, `macroProfileDTO`; extended `foodObjectDTO`; `foodItemsData` mapper populates all new fields; `classificationSummariesData` combines food categories + culinary roles; `primaryFoodCategoryData` returns first food category or nil; `macroBasisForState` derives 100ml for liquid, 100g for solid; `Calories: search.CalculateCalories(...)`.
- `backend/internal/search/substitution_service.go` — `macroCalories` exported as `CalculateCalories`; 4/4/9 rule (`Protein*4 + Carbohydrates*4 + Fat*9`); both call sites updated.
- `backend/internal/httpapi/search_controller_test.go` — new `TestSearchControllerFoodObjectDTOExposesClassificationMacrosAndCalories` covering C1-C5 for solid and liquid items.
- `frontend/src/lib/api/generated.ts` — new `ClassificationSummary`, `MacroProfile` interfaces; `FoodObject` extended with the five new fields; tab indentation consistent with existing file.
- `scripts/generate-api-types.py` — templates added for `ClassificationSummary`, `MacroProfile`, and extended `FoodObject`; output produces tabs consistent with existing generated output (drift check passes).
- `docs/implementation/04_OPEN.md` — contains stale text (added by preparation/planning) stating Task 138 is "not implemented despite PREPARED status"; now contradicted by working-tree implementation.

## Implementation review

- Calorie calculation is correct: `CalculateCalories` uses the 4/4/9 kcal/g rule (protein*4 + carbohydrates*4 + fat*9). PASS
- Classifications combine `food_category` and `culinary_role` entries via `classificationSummariesData`, preserving kind. PASS
- `primaryFoodCategory` is the first food category, or nil when none exist. PASS
- `macroBasis` is derived from `physicalState` (`100ml` for liquid, `100g` otherwise), consistent with the per-100 storage basis. PASS
- No handwritten frontend type duplicates: the new interfaces and fields exist only in `generated.ts`. PASS
- Traceability comments (`Implements DESIGN-002 SearchController ...`) present in openapi.yaml, search_controller.go, search_controller_test.go, generated.ts, and generate-api-types.py. PASS

## Decision reason

All nine explicit verification criteria (C1-C9) are satisfied with passing commands and inspectable implementation. The only documented `04_OPEN.md` Phase 04 action point — extending `FoodObject` with classification summaries, primary Food Category, macros with 100g/100ml basis, and server-calculated calories — is implemented end-to-end across the OpenAPI source of truth, the backend DTO/mapper, the exported calorie calculator, the generated frontend types, and the type-generation script. The new HTTP test verifies every required field for both solid and liquid items, including the 4/4/9 calorie calculation and physical-state-derived macro basis. OpenAPI lint, generated-type drift check, backend tests, go vet, frontend unit tests, task-list validation, and traceability validation all pass. The remaining task-description items (naming cleanup, similarity presentation cleanup, deterministic ordering cleanup, test hygiene) are not recorded as `04_OPEN.md` action points and are acceptably treated as intentionally split. Two non-blocking follow-ups are noted below for the phase-completion step.

## Notes (non-blocking follow-ups for phase-completion)

1. `gofmt -l` flags `backend/internal/httpapi/search_controller_test.go` for struct-field alignment drift in the new test. `go test` and `go vet` are unaffected, and gofmt is not in the Task 138 verification criteria, but the aggregate gate (`scripts/check.py`) enforces gofmt and would flag it. Recommend `gofmt -w backend/internal/httpapi/search_controller_test.go` before the Phase 04 aggregate/UAT gate.
2. `docs/implementation/04_OPEN.md` still records (lines 127, 138, 171) that Task 138 OpenAPI/generated-type extensions are "not implemented despite PREPARED status". This text was added during preparation/planning and is now stale given the working-tree implementation. The phase-completion step should update `04_OPEN.md` to mark the Phase 04 action point resolved and clear the stale Phase 05 carryover note.
