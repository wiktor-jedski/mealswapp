# Review Evidence: Task 219 — Primary Calorie and Secondary Diversity Objective

~~~yaml
task_id: 219
component: "Phase 07.01 Primary Calorie and Secondary Diversity Objective"
static_aspect: "DESIGN-004: ObjectiveFunction"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-14T23:19:47Z"
review_agent: "Codex fresh independent re-review"
evidence_file: "docs/implementation/reviews/task-219-review.md"
baseline_ref: "HEAD a4e31367485b03269e90b5607f2057c9568bb5b1 plus current worktree; Task 218 ownership excluded using its preparation/review manifests"
baseline_confidence: "MEDIUM"
code_review_skill_invoked: true
relevant_language_guide: "/home/wiktor/.agents/skills/code-review-skill/reference/go.md"
repair_context_required: false
~~~

## 1. Task Source

**Description:** Phase 07.01: define and implement an explicit objective policy in which calorie minimization remains primary and diversity is a secondary best-effort preference, filter zero-information candidates without rejecting unrelated valid meals, and remove objective fields and validation wrappers that have no solver semantics.

**Depends On:** 218

**Testing Coverage Exceptions:** None in the task row. The repository-level accepted Phase 07 exception in docs/implementation/04_OPEN.md:332 covers the documented defensive and optimization branches; no new Task 219 exception was added.

**Verification Criteria:** Nontrivial fixtures prove diversity cannot overturn calorie ordering; original meals remain eligible unless excluded; the diversity term has documented units, bounds, and numerical behavior or uses lexicographic solving; unusable original entries fail safely while unrelated zero-information catalog meals are filtered; every retained objective field changes serialized solver behavior and cannot diverge from it; deterministic unit, validation, and packaged-CLP tests pass; DESIGN-004 interfaces match the validated implementation contract.

## 2. Pre-Review Gates

- [x] Input status is PREPARED.
- [x] Every dependency is PREPARED or PASSED; task 218 is PREPARED in the current task table.
- [x] The preparation report claims completion and records the repair scope.
- [x] A task-specific baseline/diff is available and trustworthy at medium confidence after explicit Task 218 exclusion.
- [x] code-review-skill was invoked exactly once and its relevant Go guide was read completely.
- [x] The reviewer is independent from implementation/repair.
- [x] Review uses current repository state rather than stale logs.
- [x] Reviewer made no production-code or task-list changes; only this evidence file is being overwritten.

~~~yaml
pre_review_gates_passed: true
blocking_issue: "NONE"
~~~

## 3. Review Baseline and Change Surface

Baseline/reference method: HEAD is a4e31367485b03269e90b5607f2057c9568bb5b1. The worktree has concurrent Phase 07.01 edits without per-task commits. I reconstructed the Task 219 surface from the HEAD diff, the Task 218 preparation/review ownership exclusions, the current preparation report, current symbol searches, and direct line-by-line inspection. The stale prior Task 219 review was not used as acceptance evidence; it was used only to identify the repaired findings for re-checking.

Commands used to reconstruct the diff:

~~~bash
git rev-parse HEAD
git status --short
git diff --stat HEAD -- docs/design/DESIGN-004.md backend/internal/optimization backend/internal/worker
git diff --unified=80 HEAD -- <Task 219 candidate paths>
git diff --unified=20 HEAD -- backend/internal/optimization/objective.go backend/internal/optimization/diversity.go backend/internal/optimization/validator.go backend/internal/worker/optimization_processor.go
sed -n '1,300p' docs/implementation/preparation/task-219-preparation.md
sed -n '1,320p' docs/implementation/reviews/task-219-review.md
rg -n 'DiversityPenalties|VariableIDs|objectiveValidationError|ObjectivePolicy|BuildObjective|AlternativeSolveFunc|GenerateAlternatives|GenerateValidatedAlternatives|ValidateSolution' backend docs/design/DESIGN-004.md
nl -ba <objective, diversity, validator, constraints, worker, CLP, and design files>
sha256sum <all reviewed files>
~~~

Pre-existing dirty-worktree changes and exclusions:

The worktree contains unrelated API, repository, SQL, frontend, worker, and documentation changes. Task 218 changes are excluded from Task 219 ownership: saved-diet vocabulary and identity enforcement, request and meal domain shape, target derivation and unit conversion, hard alternative constraints, repository loading, direct validator identity validation, and their domain fixtures. Task 220 remains OPEN and is excluded; its future canonical one-pass validation and immutable-snapshot work is not attributed to Task 219. The current Task 219 repair changed the objective/diversity ownership text and exact interfaces in DESIGN-004; it did not change production Go, tests, or the task list. During this review, unrelated later task rows 221–237 were appended by the shared worktree. After the review decision, another process changed Task 219's status from PREPARED to PASSED; the reviewer did not edit the task list. The current task-list hash is recorded below.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| docs/design/DESIGN-004.md | HEAD diff plus Task 219 repair manifest | HIGH | ObjectiveFunction, DiversityPenalizer, lexicographic algorithm, exact current interfaces |
| backend/internal/optimization/constraints.go | Mixed HEAD diff; only zero-information eligibility branch attributed here | MEDIUM | BuildConstraints zero-information filtering; Task 218 domain and hard-constraint hunks excluded |
| backend/internal/optimization/objective.go | Task 219 objective diff | HIGH | ObjectiveFunction, ObjectivePolicy, BuildObjective, removed wrapper |
| backend/internal/optimization/diversity.go | Mixed HEAD diff; objective-policy hunks attributed here | MEDIUM | DefaultDiversityPenalty, GenerateAlternatives objective calls, lexicographic helpers |
| backend/internal/optimization/validator.go | Task 218 boundary file inspected, not attributed | HIGH exclusion | ValidateSolution identity guard and GenerateValidatedAlternatives caller boundary |
| backend/internal/optimization/clp_wrapper.go | Task 217 solver boundary inspected, not attributed | HIGH exclusion | Solve and serializeLPWithLimit objective serialization boundary |
| backend/internal/optimization/constraints_test.go | Mixed compatibility and Task 219 objective fixture | MEDIUM | zero-information eligibility and packaged primary-objective fixture |
| backend/internal/optimization/objective_test.go | Task 219 objective tests | HIGH | policy construction, validation, serialization mutation |
| backend/internal/optimization/diversity_test.go | Mixed compatibility and Task 219 lexicographic tests | MEDIUM | pass ordering, cap, adversarial ordering, packaged CLP |
| backend/internal/optimization/validator_test.go | Mixed Task 218 identity fixture and Task 219 partial-result fixture | MEDIUM | two-pass publication compatibility; identity test is dependency evidence only |
| backend/internal/worker/optimization_processor.go | Task 218 worker boundary plus current Task 219 caller inspection | HIGH exclusion | OptimizationSolver seam and ProcessOptimizationJob call path |
| backend/internal/worker/optimization_processor_deadline_test.go | Task 219 two-pass deadline compatibility | HIGH | deadlineSolver.Solve |
| backend/internal/worker/task210_swe5_integration_test.go | Task 219 two-pass partial-result compatibility | HIGH | partial-result test and solver fixture |
| backend/internal/worker/worker_integration_test.go | Mixed fixture migration and Task 219 two-pass solver fixture | MEDIUM | integrationSolver.Solve |

The current Task 219 diff is therefore auditable despite the shared worktree: Task 218 symbols are explicitly excluded, while their current interfaces and identity guard are inspected as dependency boundaries.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | Diversity cannot overturn calorie ordering | BuildObjective, solveObjectivePolicy, adversarial injected solver, and packaged CLP narrow-difference fixture | PASS | Primary calories are solved first; the finite primary value is added as an equality constraint before the secondary solve. The injected higher-calorie secondary result is rejected, and native CLP chooses the lower-calorie meal at a 0.000001 difference. |
| 2 | Original meals remain eligible unless excluded | BuildConstraints, DiversityPenalizer.Apply, bounds, typed exclusions, and original-meal fixture | PASS | Original variables retain the 0..10,000 bound and receive only the secondary coefficient. Exclusions omit variables; otherwise original meals remain available. |
| 3 | Diversity units, bounds, and numerical behavior are documented or lexicographic | DESIGN-004 lines 20 and 33 plus objective and constraint source | PASS | Coefficient 1 counts repository base units, grams for solids and millilitres for liquids. The range is bounded by 10,000 per eligible original variable, and lexicographic equality avoids mixed-unit weighting. |
| 4 | Unusable originals fail safely | BuildConstraints, target derivation, validator boundary, and eligibility fixtures | PASS | Invalid state, unavailable normalization, invalid macros, zero-information originals, and incomplete saved-diet identity fail with safe validation errors. The direct validator now calls the shared validateRequest guard; this is verified as Task 218 dependency behavior, not re-attributed. |
| 5 | Unrelated zero-information catalog meals are filtered | BuildConstraints candidate loop and eligibility fixture | PASS | Unrelated invalid, unavailable, and zero-calorie candidates are skipped while the valid original remains. An all-zero original is rejected rather than silently filtered. |
| 6 | Every retained objective field changes serialized behavior and cannot diverge | ObjectiveFunction, ObjectivePolicy, BuildObjective, LPSolverWrapper.Solve, serializeLPWithLimit, all callers, and serialization mutation test | PASS | Only Coefficients remains. Primary and secondary maps are each supplied to a solver pass; the mutation test produces different serialized LP objectives, and the generated-name serializer covers every model variable. |
| 7 | Deterministic unit, validation, and packaged-CLP tests pass | focused tests, full backend tests, race detector, coverage, vet, and validators | PASS | Focused tests, go test ./..., go test -race ./..., go vet ./..., traceability, task-list, Go-doc, formatting, diff, and backend coverage commands all exit 0. Native packaged CLP is installed and both lexicographic subtests pass. |
| 8 | DESIGN-004 interfaces match the validated implementation contract | direct source comparison of design lines 64–70 with current Go declarations and callers | PASS | BuildConstraints, BuildObjective, LPSolverWrapper.Solve, ValidateSolution, raw GenerateAlternatives, and publication-safe GenerateValidatedAlternatives match current signatures. The injected AlternativeSolveFunc and worker OptimizationSolver boundaries were also inspected. |

## 5. Changed-Symbol Inventory

Inventory includes every Task 219-added or Task 219-owned modified executable unit reconstructed from the mixed diff. Shared files are represented only for the Task 219 hunk. Task 218 identity/domain symbols and unchanged Task 218 boundaries are audited separately below and are not counted here.

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | BuildConstraints Task 219 eligibility branch | function | backend/internal/optimization/constraints.go:104-196 | Modified shared function; zero-information candidate policy retained | GenerateAlternatives, worker preflight, eligibility tests | TestBuildConstraintsEligibilityPolicy, TestBuildConstraintsRejectsAllZeroAuthoritativeTargetAndInvalidIdentity |
| 2 | ObjectiveFunction | behavioral type | backend/internal/optimization/objective.go:3-7 | Modified to contain only serialized coefficients | BuildObjective, CLP serializer, worker solver seam | objective construction and serialization tests |
| 3 | ObjectivePolicy | behavioral type | backend/internal/optimization/objective.go:9-15 | Added primary and secondary objectives | GenerateAlternatives, solveObjectivePolicy, worker preflight | objective and diversity tests |
| 4 | BuildObjective | function | backend/internal/optimization/objective.go:21-49 | Modified to build separate validated maps | GenerateAlternatives, worker preflight, CLP callers | objective validation and serialization tests |
| 5 | DefaultDiversityPenalty | constant | backend/internal/optimization/diversity.go:13-18 | Modified from additive weight to one base-unit coefficient | NewDiversityPenalizer, BuildObjective | original-meal penalty fixture |
| 6 | GenerateAlternatives objective-policy path | function | backend/internal/optimization/diversity.go:69-122 | Modified to run lexicographic passes and preserve partial results | GenerateValidatedAlternatives, worker processor | generation and worker tests |
| 7 | solveObjectivePolicy | function | backend/internal/optimization/diversity.go:124-167 | Added true primary-then-secondary solve | GenerateAlternatives, packaged CLP test | adversarial and native CLP tests |
| 8 | hasPositiveCoefficient | function | backend/internal/optimization/diversity.go:169-178 | Added zero-secondary short path | solveObjectivePolicy | generation and policy tests |
| 9 | objectiveValueForSolution | function | backend/internal/optimization/diversity.go:180-195 | Added finite primary optimum evaluation | solveObjectivePolicy | narrow-difference and model checks |
| 10 | TestBuildConstraintsEligibilityPolicy | test | backend/internal/optimization/constraints_test.go:109-143 | Modified shared eligibility fixture for zero-information policy | BuildConstraints | direct table and original mutation cases |
| 11 | TestTask218PackagedCLPConstraintFixture | test | backend/internal/optimization/constraints_test.go:282-307 | Modified objective consumer to use policy.Primary | CLP wrapper | packaged primary solve |
| 12 | TestBuildObjectiveRanksFeasibleFixturesByServerCalories | test | backend/internal/optimization/objective_test.go:19-53 | Modified for separate primary and secondary assertions | BuildObjective | direct |
| 13 | TestBuildObjectiveUsesServerCalculatedCaloriesAndStableEqualTies | test | backend/internal/optimization/objective_test.go:55-79 | Modified for policy maps | BuildObjective | direct |
| 14 | TestBuildObjectiveRejectsMissingInvalidAndNegativeCoefficients | test | backend/internal/optimization/objective_test.go:101-119 | Modified for retained objective fields | BuildObjective | direct table |
| 15 | TestObjectivePolicyFieldsDriveDistinctSerializedObjectives | test | backend/internal/optimization/objective_test.go:121-142 | Added serialization mutation proof | serializeLP | direct |
| 16 | TestBuildObjectiveRejectsEmptyAndDuplicateVariables | test | backend/internal/optimization/objective_test.go:144-155 | Modified for policy return type | BuildObjective | direct |
| 17 | TestBuildConstraintsPenalizesOriginalMealsWithoutForbiddingThem | test | backend/internal/optimization/diversity_test.go:24-53 | Modified to assert separate primary and secondary maps | BuildConstraints, BuildObjective | direct |
| 18 | TestGenerateAlternativesReturnsDeterministicOneOrTwoResults | test | backend/internal/optimization/diversity_test.go:55-96 | Modified for two solver passes per result | GenerateAlternatives | direct call-count and result assertions |
| 19 | TestGenerateAlternativesUsesLexicographicPassesAndCapsResults | test | backend/internal/optimization/diversity_test.go:98-153 | Added pass-order, equality-row, and three-result cap proof | GenerateAlternatives | direct |
| 20 | TestGenerateAlternativesRejectsSolverOutputThatViolatesHardExclusion | test | backend/internal/optimization/diversity_test.go:155-167 | Modified for lexicographic caller | GenerateAlternatives | direct |
| 21 | TestSolveObjectivePolicyRejectsDiversityThatOverturnsCalorieOrdering | test | backend/internal/optimization/diversity_test.go:169-186 | Added adversarial higher-calorie secondary output | solveObjectivePolicy | direct |
| 22 | TestTask219PackagedCLPLexicographicObjective | test | backend/internal/optimization/diversity_test.go:188-220 | Added native CLP narrow-difference and tie fixtures | LPSolverWrapper.Solve | direct |
| 23 | lexicographicFixtureModel | test helper | backend/internal/optimization/diversity_test.go:222-230 | Added bounded two-variable model | packaged and injected tests | direct consumers |
| 24 | TestGenerateValidatedAlternativesPreservesValidPartialResultsAfterLaterSolveFails | test | backend/internal/optimization/validator_test.go:139-161 | Modified to allow two successful objective passes before failure | GenerateValidatedAlternatives | direct |
| 25 | deadlineSolver.Solve | test method | backend/internal/worker/optimization_processor_deadline_test.go:119-125 | Modified to model two passes under one deadline | OptimizationProcessor | deadline integration |
| 26 | TestTask210WorkerPublishesPartialAlternativesOnLaterSolverFailure | test | backend/internal/worker/task210_swe5_integration_test.go:19-88 | Modified expected calls to two passes plus later failure | worker processor | Redis integration |
| 27 | task210PartialSolver.Solve | test method | backend/internal/worker/task210_swe5_integration_test.go:145-151 | Modified to return two passes before timeout | worker processor | direct consumer |
| 28 | integrationSolver.Solve | test method | backend/internal/worker/worker_integration_test.go:256-259 | Modified to keep one selected meal across both passes | worker processor | Redis worker integration |
| 29 | objectiveValidationError | removed function | backend/internal/optimization/objective.go baseline lines 66-70 | Removed redundant validation wrapper | no current callers; canonical validationError remains | repository-wide symbol search and tests |

~~~yaml
inventory_source_count: 29
audited_symbol_count: 29
inventory_complete: true
generated_groupings:
  - "None; every Task 219 executable unit is listed. Task 218 identity/domain units and unchanged solver/validator boundaries are explicitly excluded from this count."
~~~

## 6. Function-Level Audit

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| BuildConstraints Task 219 eligibility branch | Eligible variables have finite positive server calories and 0..10,000 bounds; original entries are authoritative. | Unrelated invalid, unavailable, and zero-calorie candidates are skipped; original invalid or zero-information candidates return typed validation errors; empty eligible set fails. | Pure function; no resources, goroutines, or shared mutation. | Repository meal data is trusted only after physical-state, normalization, and macro validation; IDs remain UUID-derived. | One indexed map and stable sorted IDs; bounded model size follows catalog input and fixed quantity policy. | Minimal branch layered onto the Task 218 builder; no duplicate bound rows. | Eligibility table covers zero-information and malformed original mutations; direct mixed-state boundary is covered by dependency tests. | PASS |
| ObjectiveFunction | Contains only the coefficient map consumed by LP serialization. | Empty or malformed maps are rejected by BuildObjective or serializer; no stale diagnostic fields remain. | Immutable-by-convention value passed to solver; no resources or concurrency. | Coefficients originate from repository-derived model data; solver later maps IDs to generated names. | One map per objective, linear in variables. | Smaller idiomatic value type; removed unused VariableIDs and DiversityPenalties. | Serialization mutation test and obsolete-symbol search pass. | PASS |
| ObjectivePolicy | Primary is calorie minimization; Secondary is original-meal base-unit quantity at the same model variables. | Both maps are built together and validated before solving; missing or non-finite coefficients fail. | Passed by value; no shared mutation or resource ownership. | No caller-controlled value is used as a subprocess argument. | Two maps and at most two solver calls per accepted alternative. | Explicit policy is clearer and safe for mixed units. | Objective and native CLP tests prove both fields are consumed. | PASS |
| BuildObjective | Emits positive finite calorie coefficients and non-negative finite diversity coefficients for every unique variable. | Empty, missing ID, duplicate ID, non-finite, negative, and zero calorie inputs fail through canonical validation. | Pure map construction. | Server-calculated calories and model IDs only. | Linear time and one allocation per objective map. | Minimal public API with a typed policy result. | Validation table, feasible ranking, equal-tie, and serialization tests pass; no missing branch remains relevant to acceptance. | PASS |
| DefaultDiversityPenalty | Value 1 means one gram or millilitre of an original meal in the secondary objective. | Applied only as a non-negative secondary coefficient; it is never added to calories. | Immutable constant. | N/A — no external input boundary. | No runtime cost. | Unit is documented in code and DESIGN-004. | Solid and liquid unit policy is covered by the model/design audit; a dedicated mixed-state sum fixture is optional. | PASS |
| GenerateAlternatives objective-policy path | Returns at most three distinct selected-meal sets and preserves validated partial results on later solve failure. | Handles non-positive and over-limit requests, nil solver, context cancellation between attempts, model/objective/solver/validation errors, duplicates, and bounded attempt exhaustion. | No goroutines; shared context is passed to both objective passes and checked before attempts. | Solver output is canonicalized and validated before publication. | At most 3 × limit attempts and two solver calls per attempt; copies prior solutions. | Explicit raw-result boundary supports the validated wrapper and worker seam. | Pass-order, cap, hard exclusion, partial failure, and worker deadline tests pass. The all-zero-secondary one-call path lacks a direct call-count test but is source-verified and optional. | PASS |
| solveObjectivePolicy | Secondary solve cannot alter the finite primary optimum because it adds an exact primary equality row. | Primary output is canonicalized and model-checked; finite objective is computed; secondary output is canonicalized and checked against the augmented model; all errors return. | Delegates cancellation to the injected solver; owns no files, processes, locks, or goroutines. | Untrusted solver output cannot bypass model bounds or equality constraints. | Copies the constraint slice once; maximum two solves. | Correct lexicographic formulation instead of a mixed-unit weight. | Adversarial injected output and packaged CLP narrow-difference and tie tests pass; nil solver is protected by the public generator. | PASS |
| hasPositiveCoefficient | Detects whether any secondary coefficient can affect optimization. | All-zero or empty maps take the primary-only path; BuildObjective prevents negative or non-finite production coefficients. | Read-only iteration. | N/A — internal numeric helper. | O(n), no allocations. | Small nonduplicative helper. | Positive secondary behavior is covered; direct all-zero call-count coverage is optional. | PASS |
| objectiveValueForSolution | Computes the finite primary value of a canonical sparse assignment. | Missing quantities act as zero; non-finite coefficients or quantities and overflowed totals return validation errors. | Pure arithmetic. | N/A — values are already model-validated. | O(number of objective coefficients), no I/O. | Necessary equality-row helper. | Narrow numerical difference passes; an explicit overflow fixture is optional because the finite result guard is present. | PASS |
| TestBuildConstraintsEligibilityPolicy | Proves valid original retention and unrelated candidate filtering. | Exercises unsupported state, unavailable basis, negative macro basis, zero-information catalog candidate, and zero-information original. | Test-only, no resources. | Local UUID fixtures. | Small deterministic table. | Direct table-style assertions. | Covers the Task 219 zero-information criterion and dependency boundary. | PASS |
| TestTask218PackagedCLPConstraintFixture | Proves the changed policy shape reaches native CLP through the primary objective. | Skips only when the executable is unavailable; current environment has packaged CLP and the solve passes. | Solver owns child cleanup; test is synchronous. | Generated solver names isolate repository IDs. | Tiny model. | Direct compatibility fixture. | Primary objective solve passes. | PASS |
| TestBuildObjectiveRanksFeasibleFixturesByServerCalories | Primary objective ranks feasible assignments by server calories and secondary ranks original quantity separately. | Both feasible assignments and both objective maps are asserted. | Test-only. | Repository-derived fixture values. | Constant-size arithmetic. | Direct. | Lower calorie and original/non-original assertions pass. | PASS |
| TestBuildObjectiveUsesServerCalculatedCaloriesAndStableEqualTies | Uses calculated calories and retains equal primary coefficients without accidental secondary mixing. | Excluded original and equal calorie candidates are checked; map cardinality is asserted. | Test-only. | Typed UUID fixtures. | Constant-size. | Direct. | Server calorie and separate-map assertions pass. | PASS |
| TestBuildObjectiveRejectsMissingInvalidAndNegativeCoefficients | Invalid objective inputs cannot enter solver serialization. | Missing ID, NaN, infinity, negative calorie, and invalid diversity cases fail. | Test-only. | N/A. | Small table. | Idiomatic table test. | All cases pass; zero calorie is also rejected by the production builder. | PASS |
| TestObjectivePolicyFieldsDriveDistinctSerializedObjectives | Every retained policy field changes the actual serialized solver objective. | Serializes primary and secondary maps and rejects identical LP bytes. | Test-only; no persistent files. | Serializer generated names prevent ID injection. | Two-variable model. | Direct mutation proof. | Passes against current serializeLP. | PASS |
| TestBuildObjectiveRejectsEmptyAndDuplicateVariables | Empty, missing-ID, and duplicate-variable models fail before solving. | Direct malformed cases assert errors. | Test-only. | N/A. | Constant-size. | Clear. | All cases pass. | PASS |
| TestBuildConstraintsPenalizesOriginalMealsWithoutForbiddingThem | Original variables retain bounds and only receive secondary penalty. | Checks original, unrelated, primary equality, and secondary coefficients. | Test-only. | Typed IDs. | Tiny model. | Direct. | Original eligibility and separate objectives pass. | PASS |
| TestGenerateAlternativesReturnsDeterministicOneOrTwoResults | Each accepted result uses a primary and secondary pass and stable result IDs. | Limits one and two, result cardinality, call count, and selected IDs are asserted. | Test-only injected solver; no goroutines. | Trusted test seam is local. | At most four solver calls. | Deterministic fixture. | Passes. | PASS |
| TestGenerateAlternativesUsesLexicographicPassesAndCapsResults | Primary objective precedes secondary objective, equality row is present, and cap is three. | Six calls, three results, objective coefficients, equality row, and hard exclusions are asserted. | Test-only. | Solver output boundary is challenged through model inspection. | Bounded six-call fixture. | Strong orchestration assertion. | Passes; native CLP covers numeric selection. | PASS |
| TestGenerateAlternativesRejectsSolverOutputThatViolatesHardExclusion | Excluded output cannot become a raw alternative. | Injected excluded assignment produces safe error and no partial result. | Test-only. | Direct solver-output trust boundary. | Tiny. | Direct. | Passes. | PASS |
| TestSolveObjectivePolicyRejectsDiversityThatOverturnsCalorieOrdering | A more diverse but higher-calorie secondary result is invalid. | Injected secondary assignment violates the exact primary equality and is rejected. | Test-only synchronous seam. | Adversarial solver output is model-checked. | Two calls. | Focused regression. | Passes. | PASS |
| TestTask219PackagedCLPLexicographicObjective | Native CLP preserves calorie primary ordering and uses diversity only for exact calorie ties. | Runs narrow positive calorie difference and exact tie subtests. | Test-only native child process with wrapper cleanup. | Generated names isolate UUID fixtures. | Two tiny models and two passes each. | Strong end-to-end proof. | Both subtests pass. | PASS |
| lexicographicFixtureModel | Produces a bounded two-variable model with controllable calorie difference. | Caller supplies narrow difference or tie; bounds and equality constraint are finite. | Test-only pure value. | N/A. | O(1). | Minimal fixture helper. | Used by injected and packaged tests. | PASS |
| TestGenerateValidatedAlternativesPreservesValidPartialResultsAfterLaterSolveFails | A valid alternative remains publishable when a later two-pass attempt fails. | Two successful passes then a private solver error; one result and safe worker-crash code are asserted. | Test-only context; no goroutines. | Error diagnostic is checked not to leak. | Small. | Direct worker-facing regression. | Passes. | PASS |
| deadlineSolver.Solve | Models two successful lexicographic passes followed by a shared deadline. | Calls one and two return a valid assignment; later call waits for cancellation and returns context error. | Uses the supplied context and no goroutines of its own. | Test-only. | O(1). | Correct adapter for one whole-job deadline. | Deadline integration passes under race. | PASS |
| TestTask210WorkerPublishesPartialAlternativesOnLaterSolverFailure | Worker retains a validated partial result after a later objective pass failure. | Expects two successful passes plus timeout, terminal failure, and queue behavior. | Redis integration exercises publication and acknowledgment ordering. | Safe failure code boundary asserted. | Small bounded job. | Existing SWE.5 regression adapted to two passes. | Passes in full and race suites. | PASS |
| task210PartialSolver.Solve | Returns a valid pair of objective passes before a classified timeout. | Call count controls success and timeout deterministically. | Test-only, no shared concurrency. | N/A. | O(1). | Minimal fixture. | Consumed by worker integration test. | PASS |
| integrationSolver.Solve | Returns the same chosen meal for both passes, then advances the selected meal per alternative. | Atomic call count maps pass pairs to three distinct results. | Uses atomic.Int32, safe under worker execution. | Test-only generated UUIDs. | O(1). | Correct two-pass test seam. | Redis worker integration passes under race. | PASS |
| objectiveValidationError | Removed obsolete wrapper had no solver semantics or current callers. | Repository-wide search finds no definition or caller; canonical validationError is used. | N/A — removed. | N/A. | Removes dead code and duplicate error path. | Simplifies API surface. | Objective tests and symbol search pass. | PASS |

Mandatory dependency-boundary inspection, excluded from the Task 219 inventory: SolutionValidator.Validate at backend/internal/optimization/validator.go:114-196 now calls validateRequest at line 118, so missing saved-diet ID and owner fail with FailureCodeValidation; this is Task 218 repair behavior and was rerun as regression evidence. GenerateValidatedAlternatives at validator.go:198-215 consumes raw results and maps them to publication-safe alternatives. OptimizationProcessor.ProcessOptimizationJob at backend/internal/worker/optimization_processor.go:422-498 passes the shared processing context and injected OptimizationSolver through both objective passes. OptimizationSolver at lines 365-369 matches the injected function shape. LPSolverWrapper.Solve at clp_wrapper.go:207-294 and serializeLPWithLimit at clp_wrapper.go:395-479 serialize each supplied objective with generated names and validate objective coverage. These current callers and boundaries were read line by line and are not silently treated as Task 219-owned changes.

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| OPTIONAL | backend/internal/optimization/diversity.go:140-142 | hasPositiveCoefficient short path | No direct call-count fixture proves an explicitly excluded-original request performs only one solver pass. | Source returns the validated primary assignment when every secondary coefficient is zero; existing positive-secondary tests and exclusion behavior pass. | Optional future test; no effect on current acceptance. |
| OPTIONAL | backend/internal/optimization/diversity.go:182-194 | objectiveValueForSolution | No dedicated adversarial test supplies a finite coefficient and quantity whose product overflows before equality-row construction. | The helper rejects a non-finite accumulated value, and production quantity bounds plus finite model coefficients constrain normal inputs. | Optional overflow regression test; no current defect found. |

~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 2
~~~

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| git rev-parse HEAD | repository | 0 | PASS | a4e31367485b03269e90b5607f2057c9568bb5b1 |
| git status --short | repository | 0 | PASS | Shared Phase 07.01 dirty worktree recorded; no reviewer production/task-list edit |
| gofmt -d Task 219 Go paths | repository | 0 | PASS | No formatting output |
| git diff --check | repository | 0 | PASS | No whitespace errors |
| focused optimization tests including identity, lexicographic, native CLP, malformed input, and partial results | backend | 0 | PASS | go test ./internal/optimization -run ... -count=1 -v |
| go test ./... -count=1 | backend | 0 | PASS | All backend packages, including repository and worker integrations |
| go test -race ./... -count=1 | backend | 0 | PASS | All backend packages race-clean |
| go vet ./... | backend | 0 | PASS | No diagnostics |
| go test ./internal/... -coverprofile=/tmp/task-219-backend.coverage.out -count=1 | backend | 0 | PASS | Coverage profile written; optimization 83.3%, aggregate backend 88.0% |
| go tool cover -func=/tmp/task-219-backend.coverage.out | backend | 0 | PASS | Aggregate 88.0%; below-100% branches are covered by accepted Phase 07 exception |
| python3 scripts/validate-traceability.py | repository | 0 | PASS | Traceability validation passed |
| python3 scripts/validate-task-list.py | repository | 0 | PASS | 237 sequential tasks with ordered dependencies; task 219 remains PREPARED |
| python3 scripts/validate-phase07-go-doc.py | repository | 0 | PASS | Phase 07 exported Go Doc validation passed |
| sha256sum reviewed files | repository | 0 | PASS | Current fingerprints recorded in section 9 |
| python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-219-review.md | repository | 0 | PASS | Run after this file was written; structural evidence validated |

## 9. Files Inspected and Staleness Fingerprints

The current contents were hashed after code inspection and before the evidence validator run. The old Task 219 rejection evidence was checked for staleness; its design and implementation fingerprints no longer describe the repaired current source and are not reused for this decision.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| docs/design/DESIGN-004.md | Objective/diversity and current interface source of truth | No blocking finding; repaired signatures match | SHA-256 | e944ab313d017d827f17304f6e0bf8a73f1d403dd79e236b9c97932bed81595b |
| docs/implementation/02_TASK_LIST.md | Status and dependency reference | Unrelated rows and post-review external status transition; not edited by reviewer | SHA-256 | acd3962e832a7a93d63ff1e5ab698d0dc5bce289a1a1d7eb1c5da2df58688406 |
| docs/implementation/04_OPEN.md | Accepted Phase 07 coverage exception source | No direct finding | SHA-256 | c60ad3f42f5bcb66ed6d143c0f05b761ae80b3fb8be5f1541ea340d65c26c527 |
| backend/internal/optimization/constraints.go | Eligibility and zero-information filtering boundary | No blocking finding; shared Task 218 symbols excluded | SHA-256 | 3e13e8140a487af67dfd596b88d6db5a0dada708fba666742cdc92fdd393f84b |
| backend/internal/optimization/objective.go | Objective policy and dead-field removal | No blocking finding | SHA-256 | 03461f5deb4b76e673216658c78269cd37042a7c0476baf8f459ee15a9764c35 |
| backend/internal/optimization/diversity.go | Lexicographic solve and diversity units | Optional test gaps only | SHA-256 | f11b10b9bcd4314332583d80062292c67db4119910fc21c0188f88b6f |
| backend/internal/optimization/validator.go | Validator identity and raw/validated boundary audit | Task 218 identity guard present; no Task 219 finding | SHA-256 | 8b26b9fbe50e5ec9e1d3b78f970614373777ea71561d19f20963a8aa60c587fd |
| backend/internal/optimization/clp_wrapper.go | Solver objective serialization boundary audit | No blocking finding | SHA-256 | da9ae4b9862c67ab18848a2829b763034d54d684add4eefbe08dae868b8451c7 |
| backend/internal/optimization/constraints_test.go | Eligibility and packaged compatibility tests | No blocking finding | SHA-256 | 3bb53ea9bdce760eec5a07aa6ca0d7a11c0c41006baf360940f6ab6a93a3514d |
| backend/internal/optimization/objective_test.go | Objective validation and serialization tests | No blocking finding | SHA-256 | 5957da3993b36ea7aa20ef99fc3ffb7cd2e7a224a60c3c495cc7aacaa625a979 |
| backend/internal/optimization/diversity_test.go | Lexicographic and native CLP tests | No direct test failure | SHA-256 | 58ae20f87513951b1f856a885733b5c54ec2648b9036855c673954d606a7755a |
| backend/internal/optimization/validator_test.go | Identity dependency regression and partial-result tests | Identity test passes; no Task 219 blocking finding | SHA-256 | 79b49bf7f11eac0f1526748681213b1cb95665d0b48976103825879ca8db3c96 |
| backend/internal/worker/optimization_processor.go | Worker caller and shared-context boundary audit | No blocking finding; Task 218 domain edits excluded | SHA-256 | 4db1a08c09598077cea9ffdbde347df9dab01a3f76cd61617ac0330a20b202b0 |
| backend/internal/worker/optimization_processor_deadline_test.go | Deadline two-pass fixture | No blocking finding | SHA-256 | e7660af899156cec6e07b52cee3cc1b2c43a3b31a465e4d735406082b848f180 |
| backend/internal/worker/task210_swe5_integration_test.go | Partial-failure two-pass fixture | No blocking finding | SHA-256 | 28ff63596079557f1b9995a4a2ae6af0cdd8c33c4e884156b806747e83b83448 |
| backend/internal/worker/worker_integration_test.go | Worker integration two-pass solver fixture | No blocking finding | SHA-256 | 935c3caa093d220942a7e468355881ad38f94d29d36f18bc983bde08bda1615c |

~~~yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The prior Task 219 rejection is stale for DESIGN-004, the objective/diversity source, validator dependency boundary, and all current hashes; it was overwritten by this fresh review."
  - "Task 218 evidence overlaps shared optimization files and is excluded from Task 219 attribution; current hashes and current symbols were inspected directly."
~~~

## 10. Coverage and Exceptions

- [x] Required coverage command ran.
- [x] Report path and observed threshold are recorded.
- [x] Untested branches relevant to changed symbols were inspected.
- [x] The only coverage exception used is the accepted repository Phase 07 exception; no new task-row exception was invented.

~~~yaml
coverage_required: true
coverage_exception_allowed: true
coverage_report_path: "/tmp/task-219-backend.coverage.out"
observed_line_coverage: "88.0% aggregate backend; 83.3% internal/optimization; 59.8% internal/worker"
coverage_passed: true
~~~

Coverage finding: The command passed. Aggregate coverage is below the project phase-end goal, but docs/implementation/04_OPEN.md:332 explicitly accepts the Phase 07 backend coverage gaps and names the optimization and worker branches involved. Task 219 adds no unrecorded coverage deviation; the changed objective path itself is covered by focused tests and BuildObjective reports 100% in the current profile.

## 11. Negative and Regression Checks

- [x] Existing focused objective, lexicographic, validator, and packaged-CLP tests pass.
- [x] No unrelated dependency or architectural boundary was introduced by the Task 219 objective change.
- [x] No source-of-truth documentation is contradicted; DESIGN-004 matches current interfaces and semantics.
- [x] No generated, cache, build, or temporary artifact was unintentionally added by this review.
- [x] Public objective additions are necessary and used by the raw generator and worker solver seam.
- [x] Duplicate helpers and obsolete aliases were searched for; DiversityPenalties, VariableIDs, and objectiveValidationError have no current callers.
- [x] Error, malformed-input, identity, cleanup, timeout, and concurrency paths were challenged at the relevant boundaries; Task 218 identity and Task 217 CLP behavior were rerun as dependency regression evidence.

Findings: No unresolved blocking or important issue. The two optional test-strengthening notes are retained in section 7. Nil-context handling and canonical one-pass validation remain Task 220 scope and were not misattributed to this task.

## 12. Decision

A task may be PASSED only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking or important finding remains.

Before accepting the decision, run:

~~~bash
python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-219-review.md
~~~

~~~yaml
decision: "PASSED"
reason: "The repaired Task 219 objective is lexicographically calorie-primary, safely secondary-diverse, zero-information aware, interface-aligned with DESIGN-004, identity-safe at the shared validator boundary, fully audited, and green on required tests and validators."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "NONE; retain the two optional test-strengthening notes and proceed to the independent Task 220 review when its work is PREPARED."
~~~

## 13. Repair Context

Not applicable for a passed review. The previous Task 219 rejection findings were rechecked after repair:

- DESIGN-004 now documents the exact current ValidateSolution, raw GenerateAlternatives, and GenerateValidatedAlternatives interfaces, including repository meal snapshots and the injected solver seam.
- The shared Task 218 repair is present: SolutionValidator.Validate calls validateRequest before processing a solution, and the valid, missing-ID, and missing-owner regression cases pass with FailureCodeValidation.

The old rejection is not current evidence; this file is the current validated decision.
