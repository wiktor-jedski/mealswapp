# Task 219 Preparation Evidence

## Assignment, Current Snapshot, and Attribution

- Task: **219 — Phase 07.01 Primary Calorie and Secondary Diversity Objective**.
- Task source: row 219 of `docs/implementation/02_TASK_LIST.md`.
- Design source: `docs/design/DESIGN-004.md`, static aspects `ObjectiveFunction` and `DiversityPenalizer`, with adjacent `ConstraintBuilder` and `SolutionValidator` boundaries.
- Dependency: task 218 is `PREPARED` on the current worktree.
- Git baseline and current `HEAD`: `a4e31367485b03269e90b5607f2057c9568bb5b1`.
- Snapshot basis: the current worktree after the Task 218 repair and this Task 219 review-finding repair.
- Review source: `docs/implementation/reviews/task-219-review.md`, SHA-256 `b7d5cde4d978f7c7ee8c2f8d7ac705f718ac90c12ac32fdd7c2d44ddd92ef1d3`.
- Task status: **PREPARED**, preserved. This repair did not edit `docs/implementation/02_TASK_LIST.md` or any task status.
- Later-task boundary: task 220 remains **OPEN** and is out of scope.

The worktree contains concurrent Phase 07.01 changes without per-task commits. Attribution therefore uses the baseline diff, prior Task 218/219 preparation evidence, both review files, current symbol searches, and current hashes. This repair changed only the stale Task 219 interface and ownership text in `docs/design/DESIGN-004.md` plus this preparation evidence. It did not change production Go, tests, or task statuses.

## Prepared Outcome

The optimizer uses a lexicographic objective rather than an additive mixed-unit weight:

1. Minimize server-derived calories.
2. Canonicalize and validate the primary result, evaluate its finite calorie value, add that value as an equality constraint, and minimize original-meal repository base-unit quantity.

The secondary coefficient is `1` for an original meal and `0` otherwise. Its value is summed grams for solids and millilitres for liquids, bounded by `0..(10,000 × eligible original variables)`. Since secondary coefficients are never added to calories, diversity cannot overturn calorie ordering. A secondary solve is skipped when every secondary coefficient is zero. Original meals remain eligible unless explicitly excluded.

Unrelated normalized catalog candidates with all-zero calorie information are filtered. A zero-information original entry fails safely because every persisted original entry contributes to the authoritative target.

`ObjectiveFunction` contains only the coefficient map consumed by LP serialization. `ObjectivePolicy.Primary` and `.Secondary` are each passed to `LPSolverWrapper.Solve`; obsolete `DiversityPenalties`, `VariableIDs`, and `objectiveValidationError` remain removed.

DESIGN-004 now matches the current public boundaries: `ValidateSolution` receives the repository meal snapshot; raw `GenerateAlternatives` receives that snapshot and an `AlternativeSolveFunc` and returns `[]LPSolution`; `GenerateValidatedAlternatives` owns projection to `[]DietAlternative`. DESIGN-004 also records that `SolutionValidator` applies shared saved-diet ID and owner validation at its public boundary.

## Current Shared-Symbol Inventory

### Task 219 production and design surface

| Path | Task 219 symbols/surface | Current disposition |
|---|---|---|
| `docs/design/DESIGN-004.md` | `ObjectiveFunction` and `DiversityPenalizer` responsibilities; lexicographic algorithm; secondary units/range; exact `BuildObjective`, `ValidateSolution`, raw generation, and validated generation interfaces; validator ownership text required by review | Repaired and current |
| `backend/internal/optimization/constraints.go` | `BuildConstraints` branch that filters unrelated zero-information candidates and rejects zero-information originals | Retained; file otherwise overlaps Task 218 |
| `backend/internal/optimization/objective.go` | `ObjectiveFunction`, `ObjectivePolicy`, `BuildObjective`; removal of `DiversityPenalties`, `VariableIDs`, and `objectiveValidationError` | Retained and validated |
| `backend/internal/optimization/diversity.go` | `DefaultDiversityPenalty = 1`; objective-policy invocation in `GenerateAlternatives`; `solveObjectivePolicy`; `hasPositiveCoefficient`; `objectiveValueForSolution` | Retained and validated; generator also overlaps Task 218 |
| `backend/internal/optimization/validator.go` | `GenerateValidatedAlternatives` consumes raw lexicographic results | Retained compatibility surface; request identity repair is Task 218-only |

### Task 219 tests and compatibility fixtures

| Path | Task 219 evidence | Current disposition |
|---|---|---|
| `backend/internal/optimization/constraints_test.go` | Zero-information eligibility cases and `ObjectivePolicy.Primary` packaged-CLP compatibility | Retained; shared with Task 218 |
| `backend/internal/optimization/objective_test.go` | Separate primary/secondary validation and `TestObjectivePolicyFieldsDriveDistinctSerializedObjectives` | Retained and focused test passes |
| `backend/internal/optimization/diversity_test.go` | Two-pass generation, adversarial calorie ordering, cap/call order, and native CLP narrow-difference/tie fixtures | Retained and focused tests pass |
| `backend/internal/optimization/validator_test.go` | Two-pass partial-result timing in `TestGenerateValidatedAlternativesPreservesValidPartialResultsAfterLaterSolveFails` | Retained; identity regression belongs to Task 218 |
| `backend/internal/worker/optimization_processor_deadline_test.go` | Two-pass deadline solver compatibility | Retained |
| `backend/internal/worker/task210_swe5_integration_test.go` | Two successful objective passes before later partial-result failure | Retained |
| `backend/internal/worker/worker_integration_test.go` | Deterministic result returned for both objective passes | Retained |

### Task 218 overlap explicitly excluded from Task 219 attribution

The following current symbols and behavior are dependencies, not Task 219 implementation claims:

- `constraints.go`: `MaximumMealQuantity`, request/domain types, saved-diet loading, non-nil diet and owner checks in `validateRequest`, target derivation, unit conversion, typed exclusions, model bounds, and deterministic hard alternative constraints.
- `diversity.go`: original-meal identity sourcing, canonical selected-meal-set behavior, and the deterministic hard-exclusion heuristic. Task 219 owns only the objective-policy portions identified above.
- `validator.go`: `ValidateSolution`, `(*SolutionValidator).Validate`, repository-backed recalculation, tolerance/exclusion/quantity checks, and the shared `validateRequest` call added by the Task 218 review repair.
- `validator_test.go`: `TestSolutionValidatorRequiresSavedDietIdentity`, including its valid control, nil saved-diet ID case, nil owner case, and `failed_validation` assertions, was added by the Task 218 repair. Task 219 retains and reruns it solely as dependency regression evidence.
- DESIGN-004 saved-diet domain vocabulary, eligibility, quantity policy, authoritative target, and hard meal-set-distinctness behavior remain Task 218-owned. This repair's validator ownership sentence and exact interfaces close Task 219 review finding F-1 without re-attributing the underlying Task 218 production fix.
- Task 220's model-validation-before-deduplication, immutable indexed snapshot, one-pass publication, and attempt-exhaustion semantics remain excluded and unchanged.

## Changed-Symbol Inventory for This Repair

| Path | Symbol or documentation unit | Repair action | Production behavior impact |
|---|---|---|---|
| `docs/design/DESIGN-004.md` | `SolutionValidator` responsibility | Documented shared saved-diet ID/owner validation and explicit repository meal snapshot ownership | None; aligns docs with retained Task 218 behavior |
| `docs/design/DESIGN-004.md` | `ValidateSolution` interface | Added `meals []repository.MealEntity` | None; matches current Go signature |
| `docs/design/DESIGN-004.md` | `GenerateAlternatives` interface | Added repository meals and `AlternativeSolveFunc`; corrected return to `[]LPSolution` | None; matches raw current Go signature |
| `docs/design/DESIGN-004.md` | `GenerateValidatedAlternatives` interface | Added the publication-safe `[]DietAlternative` boundary | None; matches current Go signature and ownership |
| `docs/implementation/preparation/task-219-preparation.md` | Current evidence | Rebuilt inventory, overlap exclusions, repair evidence, commands/results, and hashes | None |

No production or test symbol changed during this repair. `backend/internal/optimization/validator.go` and `validator_test.go` are byte-for-byte retained from the Task 218 repair.

## Review-Finding Repair Evidence

| Review finding | Repair/current evidence | Verification |
|---|---|---|
| F-1 BLOCKING: DESIGN-004 lines 68-69 documented stale validator/generator APIs | DESIGN-004 now declares the exact current `ValidateSolution`, raw `GenerateAlternatives`, and `GenerateValidatedAlternatives` signatures, including repository meal snapshots, injected solver seam, raw result ownership, and validated publication ownership | Source comparison, traceability, Phase 07 Go-doc validator, and full backend compile pass |
| F-2 IMPORTANT: direct validator accepted nil saved-diet ID/owner | Current `(*SolutionValidator).Validate` calls shared `validateRequest` before solution processing. The Task 218 repair already added this production fix and `TestSolutionValidatorRequiresSavedDietIdentity`; Task 219 retains both without duplicate edits | Focused valid control and nil ID/owner subtests pass with `FailureCodeValidation`; full backend and race suites pass |

## Verification-Criteria Mapping

| Task 219 criterion | Current implementation evidence | Current verification |
|---|---|---|
| Diversity cannot overturn calorie ordering | `solveObjectivePolicy` fixes the primary calorie optimum before the secondary solve | Adversarial injected fixture and native CLP `0.000001` calorie-difference fixture pass |
| Original meals remain eligible unless excluded | `DiversityPenalizer.Apply` changes only secondary coefficients; typed exclusions remove variables | Existing eligibility and diversity fixtures pass in full suite |
| Diversity units, bounds, and numerical behavior | Coefficient `1` counts g/ml; variables are bounded to `10,000`; exact primary equality avoids mixed-unit weighting | DESIGN-004 and native CLP tie fixture pass |
| Unusable originals fail safely | Original entries must provide the authoritative target | Existing eligibility fixture passes in full suite |
| Unrelated zero-information catalog meals are filtered | `BuildConstraints` skips only unrelated all-zero candidates | Existing eligibility fixture passes in full suite |
| Every retained objective field changes serialized behavior | Only `Coefficients` remains; both policy fields feed separate solver passes | Serialization mutation test passes |
| Deterministic validation and packaged CLP behavior | Canonical two-pass fixtures and repository-backed validator | Focused optimization set, full backend, and race suites pass |
| DESIGN-004 matches implementation | Exact raw/validated signatures and ownership now documented | Traceability and Go-doc validators pass |

## Commands and Current Results

- `git rev-parse HEAD` -> `a4e31367485b03269e90b5607f2057c9568bb5b1`.
- Focused validator and lexicographic set: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization -run '^(TestSolutionValidatorRequiresSavedDietIdentity|TestValidateSolution.*|TestGenerateValidatedAlternativesPreservesValidPartialResultsAfterLaterSolveFails|TestSolveObjectivePolicyRejectsDiversityThatOverturnsCalorieOrdering|TestTask219PackagedCLPLexicographicObjective|TestGenerateAlternativesUsesLexicographicPassesAndCapsResults|TestObjectivePolicyFieldsDriveDistinctSerializedObjectives)$' -count=1 -v` -> **PASS** (`0.039s` package time). Native CLP lower-calorie and equal-calorie subtests both passed; valid identity, nil diet ID, and nil owner subtests all passed.
- Full backend: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -count=1` -> **PASS** for all packages (`repository` `25.413s`, `worker` `3.193s`).
- Full backend race detector: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1` -> **PASS** for all packages (`repository` `29.372s`, `worker` `4.246s`).
- Static analysis: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` -> **PASS**.
- `python3 scripts/validate-traceability.py` -> **PASS**: `Traceability validation passed.`
- `python3 scripts/validate-task-list.py` -> **PASS**: 237 sequential tasks with ordered dependencies.
- `python3 scripts/validate-phase07-go-doc.py` -> **PASS**: `Phase 07 exported Go Doc validation passed.`
- `git diff --check` -> **PASS**.
- `rg -n '^\| (218|219|220) \|' docs/implementation/02_TASK_LIST.md` -> task 218 `PREPARED`, task 219 `PREPARED`, task 220 `OPEN`. The task-list file was not edited by this repair.

## Current SHA-256 Snapshot

| Path | SHA-256 | Attribution note |
|---|---|---|
| `docs/design/DESIGN-004.md` | `e944ab313d017d827f17304f6e0bf8a73f1d403dd79e236b9c97932bed81595b` | Task 219 review repair updates exact interfaces/ownership |
| `backend/internal/optimization/constraints.go` | `3e13e8140a487af67dfd596b88d6db5a0dada708fba666742cdc92fdd393f84b` | Shared 218/219; unchanged during repair |
| `backend/internal/optimization/objective.go` | `03461f5deb4b76e673216658c78269cd37042a7c0476baf8f459ee15a9764c35` | Task 219; unchanged during repair |
| `backend/internal/optimization/diversity.go` | `f11b10b9bcd4314332583d80062292c67db41168cd1909910fc21c0188f88b6f` | Shared 218/219; unchanged during repair |
| `backend/internal/optimization/validator.go` | `8b26b9fbe50e5ec9e1d3b78f970614373777ea71561d19f20963a8aa60c587fd` | Task 218 repair retained |
| `backend/internal/optimization/constraints_test.go` | `3bb53ea9bdce760eec5a07aa6ca0d7a11c0c41006baf360940f6ab6a93a3514d` | Shared 218/219; unchanged during repair |
| `backend/internal/optimization/objective_test.go` | `5957da3993b36ea7aa20ef99fc3ffb7cd2e7a224a60c3c495cc7aacaa625a979` | Task 219; unchanged during repair |
| `backend/internal/optimization/diversity_test.go` | `58ae20f87513951b1f856a885733b5c54ec2648b9036855c673954d606a7755a` | Task 219/shared; unchanged during repair |
| `backend/internal/optimization/validator_test.go` | `79b49bf7f11eac0f1526748681213b1cb95665d0b48976103825879ca8db3c96` | Task 218 identity regression retained; Task 219 timing fixture unchanged |
| `backend/internal/worker/optimization_processor_deadline_test.go` | `e7660af899156cec6e07b52cee3cc1b2c43a3b31a465e4d735406082b848f180` | Task 219 compatibility; unchanged during repair |
| `backend/internal/worker/task210_swe5_integration_test.go` | `28ff63596079557f1b9995a4a2ae6af0cdd8c33c4e884156b806747e83b83448` | Task 219 compatibility; unchanged during repair |
| `backend/internal/worker/worker_integration_test.go` | `935c3caa093d220942a7e468355881ad38f94d29d36f18bc983bde08bda1615c` | Task 219 compatibility; unchanged during repair |
| `docs/implementation/02_TASK_LIST.md` | `5cea9418e48077e1a2fadf9516d9768cc5228866e9fc2ee441a9e76c48d987de` | Status-preservation reference; not edited by repair |
| `docs/implementation/reviews/task-219-review.md` | `b7d5cde4d978f7c7ee8c2f8d7ac705f718ac90c12ac32fdd7c2d44ddd92ef1d3` | Immutable review source |

The preparation document intentionally omits its own hash. Hashes were captured after the documentation repair and requested validation commands.

## Risks and Deliberate Boundaries

- Lexicographic solving can require two native CLP invocations per alternative; both remain inside the existing job deadline, so diversity is best effort when time expires.
- The fixed equality uses the finite primary value reconstructed from CLP's sparse solution. The packaged fixture covers the pinned solver's narrow calorie difference and equal-calorie face.
- Summing grams and millilitres is a deterministic repository-base-unit preference, not a physical equivalence; it affects only solutions at the same calorie optimum.
- The deterministic hard-exclusion heuristic is Task 218 behavior and remains intentionally incomplete.
- Task 220's canonical-validation pipeline concerns remain open and unchanged.
- Aggregate package coverage remains below the phase-end 100% goal; no Task 219 coverage exception or task-list edit was introduced.

No task status was changed.
