# Task 218 Preparation Evidence

## Assignment, Current Snapshot, and Attribution

- Task: **218 — Phase 07.01 Constraint Domain and Eligibility Model**.
- Task source: row 218 of `docs/implementation/02_TASK_LIST.md`.
- Design source: `docs/design/DESIGN-004.md`, static aspect `ConstraintBuilder`, with adjacent `SolutionValidator` and `DiversityPenalizer` consumers.
- Dependencies: tasks 215 and 217 are `PASSED`.
- Git baseline and current `HEAD`: `a4e31367485b03269e90b5607f2057c9568bb5b1`.
- Current task status: **PREPARED**. This repair did not edit `docs/implementation/02_TASK_LIST.md` or any task status.
- Snapshot basis: the current worktree after the prepared Task 219 implementation and the Task 218 validator repair requested by `docs/implementation/reviews/task-218-review.md`.
- Attribution method: baseline diff, the prior Task 218 preparation, the Task 218 review, and the current `task-219-preparation.md` session-owned inventory and hashes.

The prior Task 218 preparation snapshot was stale because Task 219 subsequently edited shared DESIGN-004, constraint, diversity, validator-test, and worker-test paths. This document supersedes that snapshot. It inventories shared symbols explicitly and excludes Task 219-only semantics. No Task 219 implementation was reverted or edited during this repair.

The only implementation changes made by this repair are:

- `backend/internal/optimization/validator.go`: `(*SolutionValidator).Validate` now calls the shared `validateRequest` boundary before validating solver output. Missing saved-diet ID or owner therefore fails closed for both direct method calls and `ValidateSolution`.
- `backend/internal/optimization/validator_test.go`: `TestSolutionValidatorRequiresSavedDietIdentity` adds a valid control and adversarial nil diet-ID and nil owner-ID cases against the direct reusable validator boundary.
- This preparation evidence was refreshed after all code and validation commands completed.

## Current Scope and Shared-Symbol Inventory

### Task 218-owned domain and validation surface

| Path | Task 218 symbols/surface | Current disposition |
|---|---|---|
| `docs/design/DESIGN-004.md` | ConstraintBuilder responsibility; canonical request/macro vocabulary; exact saved-diet loading; eligible physical states and normalized bases; 10,000 g/ml bounds; authoritative target; validation; deterministic hard meal-set distinctness | Current shared design retained; Task 219 objective text excluded below |
| `backend/internal/optimization/constraints.go` | `MaximumMealQuantity`, `MacroTarget`, `DietOptimizationRequest`, `LPVariable` domain fields other than objective policy, `BuildConstraints` request/target/bounds/exclusion behavior, `BuildFromSavedDiet`, `LoadFromSavedDiet`, `validateRequest`, `validateMeal`, `targetForRequest`, `quantityInNutritionBasis`, `validateMacroTarget`, `excludedMealIDs`, `originalMealIDs`, `buildAlternativeConstraints` | Current and validated; Task 219 zero-information filtering inside `BuildConstraints` is shared-file but excluded |
| `backend/internal/optimization/diversity.go` | `NewDiversityPenalizer` saved-diet identity source and `GenerateAlternatives` use of prior accepted solutions/hard alternative constraints | Shared symbol; Task 219 objective-pass implementation excluded |
| `backend/internal/optimization/validator.go` | `ValidateSolution`, `(*SolutionValidator).Validate`, explicit repository snapshot, request identity, typed exclusions, 10,000 ceiling, authoritative target recomputation | Repaired and validated |
| `backend/internal/worker/optimization_processor.go` | `OptimizationJob` removal of divergent target, repository input loading, explicit saved-diet builder/validator inputs | Current and validated |

Removed Task 218 vocabulary remains absent: `DefaultMaxQuantity`, `alternativeOverlapLoss`, `originalMealsToEntries`, `canonicalTarget`, `solutionMaxQuantity`, `MacroTarget.Carbs`, request-carried targets/repository meals/prior solutions/max quantity, string exclusions, and duplicate solver meal identity.

### Task 218 tests on the current shared tree

| Path | Task 218 evidence retained in current file | Shared/later boundary |
|---|---|---|
| `backend/internal/optimization/constraints_test.go` | Canonical vocabulary, authoritative target, unit normalization, eligibility, identity, bounds, hard distinctness, deterministic fixtures, repository paging, invalid inputs, packaged CLP constraint fixture | Task 219 additions to eligibility and `ObjectivePolicy.Primary` fixture adaptation are excluded |
| `backend/internal/optimization/diversity_test.go` | Original-meal identity source, hard exclusion, deterministic distinct result generation | Task 219 two-pass counts, lexicographic assertions, and packaged objective fixture are excluded |
| `backend/internal/optimization/validator_test.go` | Recalculation, tolerance, malformed output, safe partial results, validator fixtures, and repaired direct identity adversaries | Task 219 changed partial-result solver timing; that timing is excluded while current test pass is integration evidence |
| `backend/internal/optimization/objective_test.go` | No current Task 218 behavior is claimed beyond compatibility with Task 218 domain types | Current objective policy bodies are Task 219-only |
| `backend/internal/worker/optimization_processor_deadline_test.go` | Exact diet/owner and normalized-meal worker fixture | Task 219 two-pass solver compatibility is excluded |
| `backend/internal/worker/worker_integration_test.go` | Repository-derived target and eligible saved-diet fixture | Task 219 two-pass solver compatibility is excluded |
| `backend/internal/app/task206_backend_integration_test.go` | Task 218 compatibility is limited to the excluded-domain validation assertion | Other hunks belong to Task 217 |

### Task 219-only changes explicitly excluded from Task 218

The following current behavior belongs only to Task 219 and is neither claimed nor modified by this repair:

- `docs/design/DESIGN-004.md`: lexicographic primary-calorie/secondary-original-quantity policy, secondary units/range, exact primary-optimum equality, and zero-secondary short path.
- `constraints.go`: filtering unrelated zero-information candidates and rejecting zero-information originals as objective inputs.
- `objective.go`: `ObjectivePolicy`, current `ObjectiveFunction` shape, separate coefficient maps, and `BuildObjective` policy construction.
- `diversity.go`: `DefaultDiversityPenalty = 1`, `solveObjectivePolicy`, `hasPositiveCoefficient`, `objectiveValueForSolution`, and the primary/secondary solve sequence within shared `GenerateAlternatives`.
- Task 219 objective, lexicographic, packaged-CLP, solver-call-count, and two-pass worker fixture changes in `objective_test.go`, `diversity_test.go`, `constraints_test.go`, `validator_test.go`, `optimization_processor_deadline_test.go`, `task210_swe5_integration_test.go`, and `worker_integration_test.go`.

Preservation check: current hashes for DESIGN-004, `constraints.go`, `objective.go`, `diversity.go`, their three principal test files, and the three Task 219 worker test files match the final hashes recorded in `task-219-preparation.md`. The expected exception is `validator_test.go`, changed only by the new Task 218 identity regression test. Task 219 production code is byte-for-byte preserved.

## Prepared Outcome After Repair

The Task 218 domain has one production vocabulary: `Carbohydrates`, typed UUID exclusions, and an authoritative persisted saved diet. Builder and validator boundaries now both require non-nil saved-diet and owner identity. Targets derive only from normalized persisted entries; unsupported or unusable originals fail, unrelated ineligible catalog candidates are filtered, and all-zero targets fail deliberately.

Eligible recommendation variables remain bounded to `10,000` grams or millilitres in LP bounds, without duplicate row bounds. Repeated alternatives use the documented deterministic hard exclusion of the greatest positive base-unit quantity, with ascending UUID tie-break, so quantity drift over the same selected meal IDs is not distinct.

The current Task 219 lexicographic objective composes with this domain without changing Task 218 ownership. Current focused optimization/worker tests and full backend/race suites pass, replacing the stale worker failures recorded by the Task 218 review.

## Verification-Criteria Mapping

| Task 218 criterion | Current implementation evidence | Current verification |
|---|---|---|
| One carbohydrate field | `MacroTarget.Carbohydrates`; no `Carbs` alias | Exact reflection/vocabulary test passes in optimization suite |
| One typed exclusion representation | `ExcludedMealIDs []uuid.UUID`; nil and duplicates rejected | Invalid typed-exclusion tests pass |
| One authoritative saved-diet target | `targetForRequest` uses only `OriginalDiet.Entries`; worker payload has no target | Authoritative target and worker suites pass |
| Explicit non-nil diet and owner identity | `validateRequest` checks both; builder and repaired `SolutionValidator.Validate` invoke it | `TestSolutionValidatorRequiresSavedDietIdentity` valid control passes and nil diet/owner adversaries fail with `failed_validation` |
| Supported states and normalized bases | `validateMeal`, unit conversion, and loader filtering | Eligibility and metric/imperial fixtures pass |
| Approved bounded quantity | `MaximumMealQuantity = 10_000` in model and validator | Deterministic bound and packaged CLP fixtures pass |
| No divergent solver identity | One item ID and explicit meal snapshot; obsolete request fields absent | Domain vocabulary and compile-time callers pass |
| Unusable originals fail; ineligible candidates filter | Original target entries validate; catalog loop filters invalid candidates | Eligibility mutation and loader paging fixtures pass; Task 219 zero-information policy excluded from attribution |
| All-zero target deliberate | `targetForRequest` rejects exact zero target | Explicit all-zero fixture passes |
| No duplicate bounds/model inflation | Variable bounds only; three macro rows before alternative rows | Constraint-shape fixture passes |
| Approved meal-set distinctness | `buildAlternativeConstraints` hard-excludes deterministic greatest selected meal | Quantity-drift/reversed-order fixtures and current generation tests pass |
| Deterministic fixtures | UUID sorting and deterministic tie-break | Optimization suite passes on current shared state |
| Repository/worker integration | Current repository and worker packages pass in full backend run | Full backend and race suites pass |
| Packaged CLP | Task 218 constraint model is serialized and solved by installed CLP | Included in passing optimization focused/full runs |
| Traceability/task integrity | Specific DESIGN-004 comments remain; task rows unchanged by repair | Traceability, Phase 07 Go Doc, task-list, and diff checks pass |

## Commands and Results

- Regression reproduction before the fix: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization -run '^TestSolutionValidatorRequiresSavedDietIdentity$' -count=1` -> **FAIL as expected**; nil saved-diet ID and nil owner were both accepted.
- Post-fix validator loop: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization -run '^(TestSolutionValidatorRequiresSavedDietIdentity|TestValidateSolution)' -count=1` -> **PASS**.
- Focused Task 218/current-overlap suites: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./internal/optimization ./internal/worker -count=1` -> **PASS** (`optimization` 0.118s; `worker` 3.192s).
- Full backend: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test ./... -count=1` -> **PASS** for all packages (`repository` 31.078s; `worker` 3.277s).
- Full backend race detector: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go test -race ./... -count=1` -> **PASS** for all packages (`repository` 27.501s; `worker` 4.231s).
- Static analysis: `cd backend && GOCACHE=$PWD/.go-cache GOMODCACHE=$PWD/.go-mod-cache go vet ./...` -> **PASS**.
- `python3 scripts/validate-traceability.py` -> **PASS**: `Traceability validation passed.`
- `python3 scripts/validate-phase07-go-doc.py` -> **PASS**: `Phase 07 exported Go Doc validation passed.`
- `python3 scripts/validate-task-list.py` -> **PASS**: 237 sequential tasks with ordered dependencies.
- `git diff --check` -> **PASS**.
- `rg -n '^\| 21(8|9|20) \|' docs/implementation/02_TASK_LIST.md` -> tasks 218 and 219 are `PREPARED`; no status was edited by this repair.

## Current SHA-256 Snapshot

| Path | SHA-256 | Attribution note |
|---|---|---|
| `docs/design/DESIGN-004.md` | `856be5aff72d1e9728e8870206d04bd59209924376cbe0e9b609cb7c5331aeea` | Shared 218/219; unchanged from Task 219 final |
| `backend/internal/optimization/constraints.go` | `3e13e8140a487af67dfd596b88d6db5a0dada708fba666742cdc92fdd393f84b` | Shared 218/219; unchanged from Task 219 final |
| `backend/internal/optimization/constraints_test.go` | `3bb53ea9bdce760eec5a07aa6ca0d7a11c0c41006baf360940f6ab6a93a3514d` | Shared 218/219; unchanged from Task 219 final |
| `backend/internal/optimization/diversity.go` | `f11b10b9bcd4314332583d80062292c67db41168cd1909910fc21c0188f88b6f` | Shared 218/219; unchanged from Task 219 final |
| `backend/internal/optimization/diversity_test.go` | `58ae20f87513951b1f856a885733b5c54ec2648b9036855c673954d606a7755a` | Shared 218/219; unchanged from Task 219 final |
| `backend/internal/optimization/objective.go` | `03461f5deb4b76e673216658c78269cd37042a7c0476baf8f459ee15a9764c35` | Task 219-only; excluded and preserved |
| `backend/internal/optimization/objective_test.go` | `5957da3993b36ea7aa20ef99fc3ffb7cd2e7a224a60c3c495cc7aacaa625a979` | Task 219-only/current compatibility; excluded and preserved |
| `backend/internal/optimization/validator.go` | `8b26b9fbe50e5ec9e1d3b78f970614373777ea71561d19f20963a8aa60c587fd` | Task 218 repair |
| `backend/internal/optimization/validator_test.go` | `79b49bf7f11eac0f1526748681213b1cb95665d0b48976103825879ca8db3c96` | Shared file; Task 218 repair test added after Task 219 |
| `backend/internal/worker/optimization_processor.go` | `4db1a08c09598077cea9ffdbde347df9dab01a3f76cd61617ac0330a20b202b0` | Task 218; unchanged during repair |
| `backend/internal/worker/optimization_processor_deadline_test.go` | `e7660af899156cec6e07b52cee3cc1b2c43a3b31a465e4d735406082b848f180` | Shared fixture; Task 219 final preserved |
| `backend/internal/worker/task210_swe5_integration_test.go` | `28ff63596079557f1b9995a4a2ae6af0cdd8c33c4e884156b806747e83b83448` | Task 219 compatibility only; excluded and preserved |
| `backend/internal/worker/worker_integration_test.go` | `935c3caa093d220942a7e468355881ad38f94d29d36f18bc983bde08bda1615c` | Shared fixture; Task 219 final preserved |
| `backend/internal/app/task206_backend_integration_test.go` | `7ba576d1700d7b8f19cc1b99ca57ca69728aedcda0ee091dfcb3c20d3fa58042` | Shared Task 217/218; unchanged during repair |
| `docs/implementation/02_TASK_LIST.md` | `5cea9418e48077e1a2fadf9516d9768cc5228866e9fc2ee441a9e76c48d987de` | Status-preservation reference; not edited by repair |

The preparation document omits its own hash. Hashes were captured after the implementation repair and requested validation commands. Task 219 preservation is evidenced by equality with its recorded final production and shared-file hashes, except for the intentionally repaired Task 218 validator test file.

## Risks and Deliberate Boundaries

- The deterministic hard-exclusion heuristic guarantees a changed selected meal-ID set but can sacrifice completeness; this is the approved Task 218 policy.
- Comparing solid grams and liquid millilitres is only a deterministic base-unit ordering policy, not physical equivalence.
- The 10,000 g/ml optimizer ceiling is intentionally stricter than persistence limits.
- Task 219's lexicographic objective requires extra solver passes inside the existing deadline; that cost and its objective semantics are outside Task 218 attribution.
- Task 220's model-validation-before-deduplication concern and later interface work remain outside this repair.
- Aggregate package coverage remains below the phase-end 100% goal; no Task 218 exception or task-list edit was introduced.

No task status was changed.
