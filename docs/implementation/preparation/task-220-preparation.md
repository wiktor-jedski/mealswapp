# Task 220 Preparation Evidence

## Assignment and Baseline

- Task: **220 — Phase 07.01 Alternative Generation and Canonical Validation Pipeline**.
- Task source: row 220 of `docs/implementation/02_TASK_LIST.md`.
- Design source: `docs/design/DESIGN-004.md`, static aspects `DiversityPenalizer` and `SolutionValidator`, with the adjacent `ObjectiveFunction`, `ConstraintBuilder`, and `LPSolverWrapper` result-codec boundaries inspected.
- Dependency: task 219 was `PREPARED` and eligible when preparation started.
- Fixed Git reference: `a4e31367485b03269e90b5607f2057c9568bb5b1`.
- Initial worktree: dirty with concurrent Phase 07.01 work. The complete initial `git status --short` was captured before edits. No unrelated path was cleaned, reverted, staged, or rewritten.
- Repair baseline: `docs/implementation/reviews/task-220-review.md` (REJECTED at `2026-07-17T10:34:38Z`) identified F-1 attempt-budget overflow and F-2/F-3 missing shared-index production instrumentation. The repair began from the current shared worktree, not from a clean checkout.
- Attribution confidence: high for the repair hunks in `constraints.go`, `diversity.go`, `validator.go`, and `task220_pipeline_test.go`; medium for whole-file Task 220 attribution because Tasks 221+ are interleaved in the same files.
- Scope boundary: concurrent Tasks 221+ implementation, tests, documentation, API, worker, repository, and frontend changes were preserved. This repair did not alter their symbols or revert their files.
- Status boundary: row 220 remains `PREPARED`; no task-list status was changed by this repair.

## Prepared Outcome

Alternative generation now has one authoritative sequence:

1. Clone caller-owned request slices and build one detached repository meal snapshot, one UUID index, and one UUID-sorted ID sequence.
2. Build every attempted model through `buildConstraintsFromIndex` from that exact map/ID sequence; no generation attempt rebuilds a meal map.
3. Canonicalize and exact-model-validate each primary and secondary solver output.
4. Validate and project the final assignment once from the same index.
5. Reject duplicates.
6. Only then append the result to the seen set, previous-solution state, and output.

Signed solver residue within `1e-9 × max(1, compared magnitudes)` canonicalizes to zero. The same scale-aware function now governs parser residue, variable bounds, constraint bounds, maximum quantity, and macro tolerance. Objective and constraint coefficient maps are evaluated in sorted meal-ID order.

The capped public result count remains three. Both public functions now call `alternativeGenerationLimits`, which handles non-positive limits and caps positive limits before multiplying by three. Consequently `limit = 4` and `limit = MaxInt` both produce `(3, 9)` without integer overflow. Exhaustion returns accepted partial results without error. A later context, solver, model, duplicate, or projection failure returns accepted partial results with a safe terminal error; no current invalid result is retained. A duplicate that ignores a prior hard exclusion fails exact-model validation and is not silently retried.

The publication boundary enforces the OpenAPI `OptimizationAlternative.meals` cardinality of 1..100 after zero/residue canonicalization. Solid and liquid meals project respectively to `g` and `ml`.

## Changed Paths and Symbols

| Path | Added/modified symbol or surface | Task 220 change |
|---|---|---|
| `backend/internal/optimization/constraints.go` | `BuildConstraints` | Compatibility entry point now builds one immutable index per standalone call and delegates indexed model assembly. |
| same | `buildConstraintsFromIndex` | New internal boundary consumes an existing immutable map and sorted IDs without rebuilding a per-attempt index. |
| `backend/internal/optimization/diversity.go` | `GenerateAlternatives` | Uses overflow-safe capped limits, context-scoped real instrumentation, and the shared snapshot/pipeline. |
| same | `alternativeGenerationLimits` | Caps before multiplication; returns a zero attempt budget for non-positive limits. |
| same | `validatedAlternative` | Couples one canonical solution with its sole publication projection. |
| same | `generateAlternativePipeline` | Reuses `validator.meals` and `validator.orderedMealIDs` for every model attempt and owns canonical/model/projection/deduplication/state-commit ordering. |
| same | `solveObjectivePolicy` | Retains canonical and exact-model checks for every primary/secondary solver result. |
| same | `objectiveValueForSolution` | Evaluates coefficients in deterministic sorted order. |
| same | `canonicalSolution`, `canonicalQuantities` | Validate model IDs, reject duplicate variables, remove signed residue, reject malformed quantities, and create the canonical key. |
| same | `solutionSatisfiesModel` | Uses the shared scale-aware tolerance, deterministic coefficient order, and validates bounds/coefficient references. |
| same | `sortedCoefficientIDs`, `cloneOptimizationRequest` | Add deterministic map traversal and detached request slices. |
| same | `diversityValidationEpsilon` | Removed; no second fixed model tolerance remains. |
| `backend/internal/optimization/validator.go` | `SolutionValidator` | Stores one immutable map and sorted ID sequence shared by model construction and projection. |
| same | `generationInstrumentation`, `generationInstrumentationContextKey`, `generationInstrumentationFromContext` | Context-scoped, race-safe package instrumentation observes actual public-constructor index builds and validation/projection calls. |
| same | `NewSolutionValidator`, `newSolutionValidator`, `immutableMealSnapshot` | Deep-copy repository nested slices, reject malformed/duplicate IDs, sort/index once, and wire optional real instrumentation. |
| same | `(*SolutionValidator).Validate` | Canonicalizes before cardinality/exclusion/projection and reuses the existing index. |
| same | `GenerateValidatedAlternatives` | Uses the same overflow-safe capped limits and instrumented common pipeline, eliminating raw-generation plus second validation/projection. |
| same | `quantityTolerance`, `macroWithinTolerance` | Establish one scale-aware tolerance and apply it at macro boundaries. |
| same | `quantityEpsilon` | Removed in favor of `quantityTolerance`. |
| `backend/internal/optimization/clp_wrapper.go` | `parseCLPSolutionLine` | Uses the shared scale-aware tolerance for tiny negative CLP residue. No other Task 217 behavior changed. |
| `backend/internal/optimization/task220_pipeline_test.go` | 11 focused tests | Real public-path instrumentation covers both generators; public limit tables cover 4, MaxInt, zero, and negative values; retained tests cover invalid duplicate ordering, residue, private exhaustion, 100/101, malformed snapshots, mutation, deterministic evaluation, metric/liquid ordering, and concurrency. |
| `backend/internal/optimization/diversity_test.go` | `TestTask219PackagedCLPLexicographicObjective` | Replaces the removed fixed epsilon with the shared tolerance; Task 219 behavior is unchanged. |
| `docs/design/DESIGN-004.md` | responsibilities; algorithm steps 9–10; partial-result policy | Documents the immutable snapshot, exact sequence, tolerance, deterministic evaluation, cardinality, attempts, and partial-result contract. Public signatures remain aligned with Go. |
| `docs/implementation/02_TASK_LIST.md` | row 220 `Status` only | Transitioned `OPEN` to `PREPARED` after all evidence passed; every other cell/status is preserved. |

`backend/internal/optimization/validator_test.go` was not changed; its existing tolerance, exclusion, repository recalculation, metric/liquid, and deterministic-order fixtures are retained as Task 220 evidence.

## Verification-Criteria Mapping

| Row 220 criterion | Implementation evidence | Test evidence |
|---|---|---|
| One index build and one validation/projection per result | `newSolutionValidator` builds `immutableMealSnapshot` once; `generateAlternativePipeline` passes its map/IDs to `buildConstraintsFromIndex`; each accepted loop calls `validator.Validate` once | `TestPublicAlternativeGeneratorsBuildOneIndexAndProjectEachResultOnce` exercises both exported generators and observes the actual index-build and projection boundaries |
| Invalid duplicate rejected, not silently retried | `solveObjectivePolicy` canonicalizes and exact-model-validates before pipeline dedup/state writes | `TestAlternativePipelineRejectsInvalidDuplicateBeforeStateMutation` observes one retained result, one projection, and failure on the third solver call |
| Shared zero/residue canonicalization | `canonicalQuantities` is used for objective passes and final projection; previous state contains only canonical solutions | `TestAlternativePipelineCanonicalizesResidueForCurrentAndPreviousSolutions`; CLP codec tests |
| Exact-model checks before deduplication | Every return from `solveObjectivePolicy` has passed `solutionSatisfiesModel`; dedup occurs later in `generateAlternativePipeline` | Invalid duplicate/hard-exclusion fixtures |
| Deterministic constraints/evaluation | UUID-sorted snapshot plus `sortedCoefficientIDs` for objective and constraint accumulation | `TestDeterministicObjectiveAndConstraintEvaluation` repeated 100 times; existing deterministic generation tests |
| Nil context and malformed snapshots fail safely | Pipeline checks nil context, nil/empty index, nil IDs, and duplicates before solver invocation | `TestAlternativePipelineRejectsNilContextAndMalformedSnapshots` |
| 100/101 selected-meal boundary matches OpenAPI | Cardinality is checked after canonicalization against `maxAlternativeMealCount = 100`; OpenAPI has `maxItems: 100` | `TestSolutionValidatorSelectedMealCountMatchesOpenAPIBoundary` accepts 100 plus residue and rejects 101 |
| Approved partial results and attempt exhaustion | `alternativeGenerationLimits` derives `(3, 9)` only after capping; state is committed only after acceptance; later safe failures return prior results; exhaustion returns prior results with nil | `TestPublicAlternativeGeneratorsCapAttemptBudgetBeforeMultiplication`, `TestPublicAlternativeGeneratorsDoNotSolveNonPositiveLimits`, existing partial-failure test, and private exhaustion test |
| Caller mutation cannot change snapshot | Request slices and meal values/nested slices are detached before solving | `TestAlternativePipelineSnapshotIgnoresLaterCallerMutation` mutates caller meals/request during the first solve |
| Metric/liquid | `mealBaseUnit` and one indexed repository projection remain authoritative | `TestValidateSolutionRecomputesEveryAcceptedAlternativeFromRepositoryMeals`; `TestSolutionValidatorConcurrentMetricAndLiquidProjection` |
| Scale-aware tolerance | `quantityTolerance` is the sole quantity/model/macro tolerance and is used by CLP residue parsing | Existing tolerance-boundary test plus residue/model fixtures |
| Exclusion | Canonical positive excluded IDs fail projection; prior hard exclusions fail exact-model validation | Existing malformed/excluded tests and invalid-duplicate test |
| Ordering | Sorted repository IDs and coefficient IDs determine projection and numeric evaluation | Existing repository-order assertion and deterministic evaluation test |
| Race safety | Snapshot/index are read-only after construction; generation does not retain caller slices | Concurrent validator test and affected-package `go test -race` pass |
| DESIGN-004 signatures match | Existing public `ValidateSolution`, `GenerateAlternatives`, and `GenerateValidatedAlternatives` signatures are retained and documented | Go-doc, focused compile/tests, and Task 220 traceability checks |

## Commands and Results

- `git rev-parse HEAD` -> `a4e31367485b03269e90b5607f2057c9568bb5b1`.
- Focused repaired surface: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization -run 'PublicAlternativeGenerators|AlternativePipeline|SolutionValidatorSelected|DeterministicObjective|SolutionValidatorConcurrent|ValidateSolution' -count=1` -> **PASS**.
- Optimization repetition: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization -count=10` -> **PASS** (`1.383s`).
- Affected-package race: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./internal/optimization ./internal/worker -count=1` -> **PASS** (`optimization 1.201s`, `worker 9.185s`).
- Static analysis: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` -> **PASS**.
- Full backend: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -count=1` -> **TASK-220 PACKAGES PASS; AGGREGATE FAILS OUTSIDE SCOPE**. `internal/optimization` and `internal/worker` passed; `internal/httpapi` failed concurrent Task 222 assertions in `TestOptimizationHTTPIdempotencyAndQueueFailure` and `TestOptimizationHTTPRejectsMalformedPersistedAcknowledgementWithoutLocationFallback` (`202` expected, `409` received). The chained full-race command therefore did not start.
- `python3 scripts/validate-traceability.py` -> **TASK-220 DECLARATIONS PASS; AGGREGATE FAILS OUTSIDE SCOPE** on concurrent declarations in `backend/internal/httpapi/auth_errors.go:28` and `backend/internal/httpapi/optimization_controller.go:33`.
- `python3 scripts/validate-task-list.py` -> **PASS**: 237 sequential tasks with ordered dependencies.
- `python3 scripts/validate-phase07-go-doc.py` -> **PASS**.
- `git diff --check` -> **PASS**.

## Current SHA-256 Snapshot

| Path | SHA-256 | Attribution |
|---|---|---|
| `backend/internal/optimization/clp_wrapper.go` | `cc5079bf7475f8bea0e7d97327a9f511a7ca17c4fbdd11564da2bf2bf3e48996` | Task 220 changes only parser tolerance call; remainder is prior Task 217 work |
| `backend/internal/optimization/constraints.go` | `9f1d72435bac344e8e5c0b4140c19d87392a993e8834c43f689cd24e86627db1` | Shared prior 218 file plus Task 220 shared-index repair |
| `backend/internal/optimization/diversity.go` | `647547e6488f23455ab56f5042d3aa2ffbae1caee0f56b43cdb00fb99ae7ffd7` | Shared prior 218/219 file plus Task 220 overflow/shared-index repair |
| `backend/internal/optimization/validator.go` | `a9d646b671aef88746c1030b14a3976999b068c111e85515e4a7e04c4212a79a` | Includes preserved Task 221 overlay plus Task 220 snapshot/instrumentation repair |
| `backend/internal/optimization/diversity_test.go` | `0a00afe09117ccf468477989b9beda2704db8e12d02eb09d860c5f2d797ae8fc` | One compatibility assertion changed to shared tolerance |
| `backend/internal/optimization/validator_test.go` | `097338ed951cdcb7c43e6c93d75524be6d9e3aa603592583e82fc7a6c3255a20` | Includes preserved Task 221 tests; inherited Task 220 validation fixtures remain evidence |
| `backend/internal/optimization/task220_pipeline_test.go` | `0704646e1bd48048dc95ca2320dd2018d6c5242cf8c0092b166b646f30eccea5` | Task 220 focused tests with repaired public instrumentation/boundaries |
| `docs/design/DESIGN-004.md` | `e4e9e3cdde5f8715c586ae4a7f1c4a3da11697574203881e5b0df8a959e37782` | Shared design contract; unchanged by repair |
| `docs/implementation/02_TASK_LIST.md` | `e57ae220a9a603aeba610f3e58992701b63ef5c42d2406bcd3bbac16ff79a1eb` | Current shared status fingerprint; row 220 remains PREPARED |

This preparation document intentionally omits its own hash.

## Risks and Deliberate Boundaries

- A `1e-9` relative tolerance can accept absolute drift up to `1e-5` at the `10,000` quantity ceiling. This is deliberate scale-aware floating-point tolerance and remains far below API quantity precision.
- Deterministic sorted accumulation is reproducible but is not compensated summation. Current model bounds and finite validation make this sufficient; changing numerical algorithms would require separate solver-compatibility evidence.
- The three-times-capped-limit attempt budget is normally not exhausted because current invalid outputs terminate instead of retrying. Public tests prove safe boundary calculation and non-positive behavior; the private seam remains the direct partial-result exhaustion fixture.
- `SolutionValidator` is safe for concurrent reads after construction. Instrumentation is context-scoped and wired before use; production callers without the private context key pay only a nil check.
- Partial alternatives remain exposed only under the existing failed-result contract. Concurrent Task 221 failure-code/similarity work was preserved and is not attributed to this repair.
- Aggregate phase coverage policy is unchanged; no row-level coverage exception was added. Current aggregate HTTP/traceability failures are recorded above and remain outside Task 220.
