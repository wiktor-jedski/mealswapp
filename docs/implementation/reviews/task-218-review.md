# Review Evidence: Task 218 — Constraint Domain and Eligibility Model

~~~yaml
task_id: 218
component: "Phase 07.01 Constraint Domain and Eligibility Model"
static_aspect: "DESIGN-004: ConstraintBuilder"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-15T00:00:00Z"
review_agent: "Codex fresh independent re-review"
evidence_file: "docs/implementation/reviews/task-218-review.md"
baseline_ref: "a4e31367485b03269e90b5607f2057c9568bb5b1 plus current worktree hashes and Task 219 exclusion manifest"
baseline_confidence: "MEDIUM"
code_review_skill_invoked: true
relevant_language_guide: "/home/wiktor/.agents/skills/code-review-skill/reference/go.md; solver boundary audited against DESIGN-004 and LPSolverWrapper serialization"
repair_context_required: false
~~~

## 1. Task Source

**Description:** Phase 07.01: align the constraint-builder interface and domain vocabulary with the production saved-diet workflow, define eligible meal and nutrition-basis rules, select an approved maximum quantity policy, remove redundant bounds and ambiguous test-only request fields, and implement a meal-set diversity constraint or explicitly documented heuristic that cannot mistake quantity drift for a distinct alternative.

**Depends On:** 215, 217 — both PASSED.

**Testing Coverage Exceptions:** None on Task 218. The repository’s accepted Phase 07 coverage disposition in docs/implementation/04_OPEN.md covers defensive/dependency/process-bootstrap branches; changed Task 218 behavior has focused tests and was manually audited.

**Verification Criteria:** One carbohydrate field; one typed exclusion representation; one authoritative saved-diet target; explicit non-nil diet/owner identity; supported physical states and normalized macro bases; approved bounded quantity policy; no divergent solver identity; unusable originals fail while ineligible catalog candidates follow policy; all-zero targets are deliberate; no duplicate bounds; approved distinctness; deterministic fixtures; repository/worker integration; packaged CLP; traceability validators.

## 2. Pre-Review Gates

- [x] Input status is PREPARED.
- [x] Every dependency is PREPARED or PASSED.
- [x] The preparation report claims completion and records the validator identity repair.
- [x] A task-specific baseline/diff is available. The baseline is the fixed commit a4e31367485b03269e90b5607f2057c9568bb5b1; shared-file attribution is bounded by task-219-preparation.md.
- [x] code-review-skill was invoked exactly once and its Go guide was read completely.
- [x] The reviewer is independent from implementation/repair.
- [x] Review uses current repository state rather than stale logs.
- [x] Reviewer made no production-code or task-list changes.

~~~yaml
pre_review_gates_passed: true
blocking_issue: "None"
~~~

## 3. Review Baseline and Change Surface

Baseline/reference method: The worktree has no task commits after a4e31367485b03269e90b5607f2057c9568bb5b1. I reconstructed the baseline diff, then used the refreshed Task 218 preparation and Task 219 preparation as the attribution manifest for shared files. The Task 218 repair is limited to validator.go and validator_test.go; the current Task 219 objective edits in DESIGN-004, constraints.go, diversity.go, objective.go, and shared tests are excluded from Task 218 ownership but were inspected as callers/dependencies.

Commands used to reconstruct the diff:

~~~bash
git rev-parse HEAD
git status --short
git diff --name-status a4e31367485b03269e90b5607f2057c9568bb5b1
git diff --function-context a4e31367485b03269e90b5607f2057c9568bb5b1 -- backend/internal/optimization backend/internal/worker
rg -n 'BuildConstraints|LoadFromSavedDiet|ValidateSolution|GenerateAlternatives|validateRequest|buildAlternativeConstraints' backend/internal/optimization backend/internal/worker
sha256sum <all reviewed files>
~~~

Pre-existing dirty-worktree changes and exclusions:

The worktree contains concurrent Phase 07 changes for tasks 213–219, including API, repository, frontend, CLP, objective, and worker files. Task 219 is PREPARED and its exact objective-policy edits are excluded from the Task 218 implementation attribution. In shared files, this review covers only the Task 218 domain, eligibility, identity, bounds, authoritative-target, and hard meal-set-distinctness behavior, plus the Task 218 validator identity repair. Task 220 is OPEN; its canonical validation-before-deduplication, immutable snapshot, nil-context, and attempt-exhaustion semantics are explicitly outside this review. docs/implementation/02_TASK_LIST.md remained unchanged by the repair.

| Changed file | Change source | Task-owned confidence | Symbols/units discovered |
|---|---|---|---|
| backend/internal/optimization/constraints.go | Baseline diff; Task 218 preparation; Task 219 overlap manifest | HIGH | Domain types, BuildConstraints, saved-diet loader, request/meal/target/unit/exclusion validation, hard alternative constraints |
| backend/internal/optimization/validator.go | Baseline diff; Task 218 repair evidence | HIGH | ValidateSolution and SolutionValidator.Validate; shared request identity validation; bounded output validation |
| backend/internal/optimization/diversity.go | Baseline diff; Task 218/219 overlap manifest | MEDIUM | Original-meal identity and hard distinctness caller; Task 219 objective pass excluded |
| backend/internal/worker/optimization_processor.go | Baseline diff; Task 218 preparation | HIGH | Queue envelope, repository input loader, worker model/generation caller |
| backend/internal/optimization/constraints_test.go | Baseline diff; current focused tests | HIGH | Domain, eligibility, bounds, target, identity, loader, distinctness, CLP fixtures |
| backend/internal/optimization/validator_test.go | Task 218 repair plus current test tree | HIGH | Recalculation, malformed output, tolerance, identity regression, partial results |
| backend/internal/optimization/diversity_test.go | Shared current tree; Task 219 portions excluded | MEDIUM | Original identity, hard exclusion, deterministic generation fixtures |
| backend/internal/worker/optimization_processor_deadline_test.go; worker_integration_test.go; task210_swe5_integration_test.go | Shared worker compatibility callers | MEDIUM | Current Task 218 request/meal snapshot shape; Task 219 solve-count assertions excluded |
| backend/internal/app/task206_backend_integration_test.go | Shared prior integration fixture | MEDIUM | Task 218 excluded-domain validation assertion only |
| docs/design/DESIGN-004.md; ARCH-004.md; repository nutrition/unit/SQL sources | Design and dependency boundaries | HIGH | Current domain vocabulary, physical states, normalized bases, query filter, serializer bounds |

If any Task 218-owned change cannot be distinguished from Task 219, it is treated as shared dependency evidence rather than silently attributed. The remaining shared symbols were independently inspected and their current hashes were captured.

## 4. Acceptance Criteria Checklist

| # | Criterion | Evidence required | Result | Evidence |
|---:|---|---|---|---|
| 1 | One carbohydrate field | Type reflection and repository-wide obsolete-vocabulary search | PASS | MacroTarget has Carbohydrates only; no MacroTarget.Carbs or string carbohydrate alias remains. |
| 2 | One typed exclusion representation | Request type, worker/API callers, malformed-input tests | PASS | ExcludedMealIDs is []uuid.UUID; nil and duplicate UUIDs fail in the domain boundary and parser callers. |
| 3 | One authoritative saved-diet target | targetForRequest, loader, worker envelope inspection | PASS | Targets sum only persisted OriginalDiet.Entries after repository meal lookup and unit conversion; queue input has no target-macro field. |
| 4 | Explicit non-nil diet and owner identity | Builder and direct validator entry points plus regression tests | PASS | validateRequest rejects nil saved-diet ID/owner; BuildConstraints and repaired SolutionValidator.Validate both invoke it; valid control and both nil adversaries pass/fail as intended. |
| 5 | Supported physical states and normalized bases | validateMeal, repository query, unit/macro dependencies | PASS | Only solid/liquid meals with NormalizedMacrosAvailable and valid per-100 macros enter the model; loader requests only solid/liquid and filters unusable catalog candidates. |
| 6 | Approved bounded quantity | LPVariable bounds, serializer, validator, CLP fixture | PASS | MaximumMealQuantity is 10,000; variables carry 0..10,000 bounds, serializer emits them in Bounds, validator applies the same policy, and over-ceiling matrix/output fixtures fail. |
| 7 | No divergent solver identity | Type shape, worker loader, obsolete-symbol search | PASS | ItemID is the sole solver meal identity; removed MealID/request target/repository/max-quantity aliases have no remaining caller. |
| 8 | Unusable originals fail and ineligible catalog candidates follow policy | Eligibility mutation and paging fixtures; target derivation | PASS | Invalid state, unavailable basis, invalid macros, missing original, and unusable original paths fail; unrelated invalid candidates are skipped. Task 219’s zero-information objective filter is excluded from attribution and remains compatible. |
| 9 | All-zero targets are deliberate | Explicit all-zero target fixture and source inspection | PASS | Exact zero authoritative macro target fails before model construction; an unrelated candidate cannot supply a caller-authored target. |
| 10 | No duplicate bounds/model inflation | Model shape and LP serializer inspection | PASS | Three macro rows are emitted before alternative rows; quantity bounds live on variables and are serialized once, with no quantity_* rows. |
| 11 | Approved meal-set distinctness | buildAlternativeConstraints, reversed-order and quantity-drift fixture | PASS | The greatest positive base-unit quantity is hard-excluded; equal quantities use ascending UUID; unchanged selected IDs with different quantities cannot satisfy the next model. |
| 12 | Deterministic fixtures | Stable ID sorting, tie-break source, packaged CLP test | PASS | Candidate order does not affect the matrix or selected exclusion; focused deterministic and native CLP fixtures pass. |
| 13 | Repository and worker integration | Full backend and race suites, loader/processor inspection | PASS | Current go test ./... and go test -race ./... pass; worker receives owner-scoped repository snapshot and authoritative request. |
| 14 | Packaged CLP | Task218 packaged solver fixture | PASS | TestTask218PackagedCLPConstraintFixture passes with the installed pinned CLP path; serializer consumes variable bounds correctly. |
| 15 | Traceability and task integrity | Traceability, Go Doc, task-list, diff checks | PASS | All validators pass; task 218 and 219 remain PREPARED, task 220 remains OPEN, and the task list was not edited. |

## 5. Changed-Symbol Inventory

The inventory is the current Task 218-owned executable surface reconstructed from the fixed baseline, the refreshed preparation, and the Task 219 overlap manifest. Objective-policy-only symbols are not attributed to Task 218; shared callers are listed where their Task 218 boundary matters.

| # | Symbol/unit | Kind | File:line | Added/modified | Callers or consumers | Tests |
|---:|---|---|---|---|---|---|
| 1 | MaximumMealQuantity | constant | backend/internal/optimization/constraints.go:20 | Added/renamed | Builder, validator, CLP serializer | constraints and validator fixtures |
| 2 | MacroTarget | type | constraints.go:26 | Modified | Builder, validator, worker result | vocabulary and optimization fixtures |
| 3 | DietOptimizationRequest | type | constraints.go:44 | Modified | Builder, diversity, validator, worker | vocabulary and request fixtures |
| 4 | LPVariable | type | constraints.go:53 | Modified | Objective/CLP boundary | vocabulary and CLP fixtures |
| 5 | BuildConstraints | function | constraints.go:104 | Modified | loader, generator, worker | domain, eligibility, bound, CLP fixtures |
| 6 | ConstraintBuilder.BuildFromSavedDiet | method | constraints.go:203 | Modified | repository-backed worker path | loader/integration fixtures |
| 7 | ConstraintBuilder.LoadFromSavedDiet | method | constraints.go:215 | Modified | RepositoryOptimizationInputLoader | paging and exact-identity fixtures |
| 8 | validateRequest | function | constraints.go:291 | Modified | BuildConstraints and SolutionValidator.Validate | identity/numeric fixtures |
| 9 | validateMeal | function | constraints.go:310 | Modified | builder, target, loader, validator | eligibility fixtures |
| 10 | targetForRequest | function | constraints.go:322 | Modified | builder and validator | authoritative target/unit fixtures |
| 11 | quantityInNutritionBasis | function | constraints.go:357 | Modified | targetForRequest | metric/imperial fixtures |
| 12 | validateMacroTarget | function | constraints.go:390 | Modified | targetForRequest | zero/numeric fixtures |
| 13 | excludedMealIDs | function | constraints.go:418 | Modified | builder, alternative constraints, validator | typed exclusion fixtures |
| 14 | originalMealIDs | function | constraints.go:434 | Added | eligibility and diversity policy | original/candidate fixtures |
| 15 | buildAlternativeConstraints | function | constraints.go:446 | Modified | BuildConstraints and generator | quantity-drift fixture |
| 16 | NewDiversityPenalizer | function | diversity.go:31 | Modified | BuildConstraints | original-meal eligibility fixture |
| 17 | GenerateAlternatives | function | diversity.go:73 | Modified/shared | worker and validated publication boundary; Task 219 objective part excluded | generation and worker fixtures |
| 18 | ValidateSolution | function | validator.go:107 | Modified | generator and publication boundary | validator fixtures |
| 19 | SolutionValidator.Validate | method | validator.go:114 | Modified and repaired | ValidateSolution and GenerateValidatedAlternatives | identity/malformed/tolerance fixtures |
| 20 | DefaultMaxQuantity | removed constant | baseline constraints.go | Removed | No remaining caller | obsolete-vocabulary search |
| 21 | alternativeOverlapLoss | removed constant | baseline constraints.go | Removed | No remaining caller | distinctness source audit |
| 22 | originalMealsToEntries | removed function | baseline constraints.go | Removed | No remaining caller | request vocabulary search |
| 23 | canonicalTarget | removed function | baseline constraints.go | Removed | No remaining caller | authoritative-target search |
| 24 | solutionMaxQuantity | removed function | baseline validator.go | Removed | No remaining caller | validator ceiling search |
| 25 | OptimizationJob | type | backend/internal/worker/optimization_processor.go:62 | Modified | Redis store, HTTP controller, worker | worker fixtures |
| 26 | RepositoryOptimizationInputLoader.Load | method | optimization_processor.go:355 | Modified | OptimizationProcessor | worker integration fixtures |
| 27 | OptimizationProcessor.ProcessOptimizationJob | method | optimization_processor.go:422 | Modified | queue processor | deadline/integration fixtures |
| 28 | TestConstraintDomainUsesOneProductionVocabulary | test | constraints_test.go:24 | Added/modified | type shape | direct |
| 29 | TestBuildConstraintsUsesAuthoritativeSavedDietAndCanonicalDomain | test | constraints_test.go:49 | Added/modified | BuildConstraints | direct |
| 30 | TestBuildConstraintsNormalizesSupportedOriginalDietBases | test | constraints_test.go:79 | Added/modified | target derivation | direct |
| 31 | TestBuildConstraintsEligibilityPolicy | test | constraints_test.go:109 | Added/modified | eligibility | direct |
| 32 | TestBuildConstraintsRejectsAllZeroAuthoritativeTargetAndInvalidIdentity | test | constraints_test.go:145 | Added/modified | request validation | direct |
| 33 | TestBuildConstraintsMealSetDistinctnessCannotUseQuantityDrift | test | constraints_test.go:163 | Added/modified | alternative constraints | direct |
| 34 | TestBuildConstraintsDeterministicFeasibleAndInfeasibleFixtures | test | constraints_test.go:193 | Added/modified | bounds and matrix | direct |
| 35 | TestConstraintBuilderLoadsEligibleCatalogInBoundedPages | test | constraints_test.go:221 | Added/modified | repository loader | direct |
| 36 | TestConstraintBuilderRequiresExactRepositoryDietIdentity | test | constraints_test.go:250 | Added | loader identity | direct |
| 37 | TestBuildConstraintsRejectsInvalidNumericAndTypedExclusionInputs | test | constraints_test.go:264 | Added | request/output validation | direct |
| 38 | TestTask218PackagedCLPConstraintFixture | test | constraints_test.go:282 | Added | packaged solver | direct |
| 39 | eligibleConstraintMeal | fixture helper | constraints_test.go:309 | Added/modified | constraint tests | all constraint fixtures |
| 40 | savedDietConstraintRequest | fixture helper | constraints_test.go:317 | Added | constraint tests | all constraint fixtures |
| 41 | pagedConstraintMealRepository.Search | fake repository method | constraints_test.go:401 | Added/modified | loader test | bounded paging |
| 42 | TestBuildConstraintsPenalizesOriginalMealsWithoutForbiddingThem | test | diversity_test.go:24 | Modified | original-meal policy | direct |
| 43 | TestGenerateAlternativesReturnsDeterministicOneOrTwoResults | test | diversity_test.go:55 | Modified | generation | direct |
| 44 | TestGenerateAlternativesUsesLexicographicPassesAndCapsResults | shared test | diversity_test.go:98 | Current replacement; Task 219 solve-count assertions excluded | generation compatibility | focused suite |
| 45 | TestGenerateAlternativesRejectsSolverOutputThatViolatesHardExclusion | test | diversity_test.go:155 | Modified | hard exclusion | direct |
| 46 | diversityMeals | fixture helper | diversity_test.go:232 | Modified | diversity tests | direct |
| 47 | diversityRequest | fixture helper | diversity_test.go:244 | Added | diversity tests | direct |
| 48 | TestBuildObjectiveRanksFeasibleFixturesByServerCalories | shared compatibility test | objective_test.go:19 | Task 219 body; Task 218 request shape consumer only | BuildConstraints caller | focused suite |
| 49 | TestBuildObjectiveUsesServerCalculatedCaloriesAndStableEqualTies | shared compatibility test | objective_test.go:55 | Task 219 body; Task 218 request shape consumer only | BuildConstraints caller | focused suite |
| 50 | objectiveMeal | fixture helper | objective_test.go:81 | Shared compatibility fixture | objective tests | focused suite |
| 51 | objectiveRequest | fixture helper | objective_test.go:89 | Shared compatibility fixture | objective tests | focused suite |
| 52 | TestValidateSolutionRecomputesEveryAcceptedAlternativeFromRepositoryMeals | test | validator_test.go:22 | Modified | validator | direct |
| 53 | TestValidateSolutionAcceptsToleranceBoundariesAndFloatingPointEpsilon | test | validator_test.go:59 | Modified | validator | direct |
| 54 | TestSolutionValidatorRequiresSavedDietIdentity | test | validator_test.go:82 | Added by repair | direct validator boundary | direct |
| 55 | TestValidateSolutionRejectsMalformedQuantitiesIDsAndAlternatives | test | validator_test.go:110 | Modified | validator | direct |
| 56 | TestGenerateValidatedAlternativesPreservesValidPartialResultsAfterLaterSolveFails | test | validator_test.go:139 | Modified/shared | generator/publication boundary | direct |
| 57 | validatorMeal | fixture helper | validator_test.go:163 | Added/modified | validator tests | direct |
| 58 | validatorRequest | fixture helper | validator_test.go:171 | Added | validator tests | direct |
| 59 | TestOptimizationProcessorAppliesOneDeadlineAcrossAlternativeSolves | test | optimization_processor_deadline_test.go:15 | Modified/shared | processor | current full suite |
| 60 | TestRunPublishesAlternativeBeforeAcknowledgingQueuedJob | test | worker_integration_test.go:25 | Modified/shared | worker | current full suite |
| 61 | TestTask206BackendIntegrationGate | integration test | task206_backend_integration_test.go:40 | Compatibility-modified | API/worker/CLP | full backend suite |

~~~yaml
inventory_source_count: 61
audited_symbol_count: 61
inventory_complete: true
generated_groupings:
  - "None for production symbols. Shared Task 219 objective bodies are listed only as compatibility consumers and their objective semantics are excluded from the Task 218 result."
~~~

## 6. Function-Level Audit

Every inventory entry has a row below. Pure data and removed symbols explicitly mark lifecycle questions N/A. For shared Task 219 rows, only the Task 218 domain boundary is audited.

| Symbol/unit | Contract and invariants | Normal/edge/error paths | State/resources/cancellation/concurrency | Security boundaries | Performance/allocations/I/O | Simplicity/API/idioms | Tests and adversarial gaps | Result |
|---|---|---|---|---|---|---|---|---|
| MaximumMealQuantity | Single 10,000 g/ml policy | N/A — constant | N/A — immutable | Caps solver-facing quantities | N/A | One shared constant | Bound and CLP tests | PASS |
| MacroTarget | Protein, Carbohydrates, Fat only | N/A — value type | N/A | No client target authority | N/A | Canonical vocabulary | Reflection test | PASS |
| DietOptimizationRequest | Saved diet, tolerance, typed UUID exclusions | Consumers reject empty/invalid fields | Copied at loader boundary | Removes target/bound injection | Small value object | Minimal public contract | Vocabulary and malformed-input tests | PASS |
| LPVariable | ItemID, lower/upper bounds, server macro/calorie coefficients | Serializer rejects invalid IDs/bounds/coefficients | N/A — matrix value | No caller-owned solver identity | One variable per eligible meal | Removed duplicate MealID | CLP and vocabulary tests | PASS |
| BuildConstraints | Stable eligible variables, macro rows, exclusions, prior hard constraints | Nil/duplicate IDs, empty meals, bad requests, unusable originals, all-zero target, bad prior solutions fail | Pure; no shared mutation | Repository/request trust boundary | O(meals log meals + entries); bounded page input | No duplicate bound rows | Broad deterministic, malformed, CLP tests | PASS |
| ConstraintBuilder.BuildFromSavedDiet | Uses owned loader snapshot then pure builder | Propagates repository/validation errors | Context passed; no goroutines | Authenticated owner/diet path | Thin delegator | Minimal API | Indirect loader and worker coverage | PASS |
| ConstraintBuilder.LoadFromSavedDiet | Exact owner/diet, authoritative diet entries, supported catalog | Empty diet, mismatched IDs, missing/unusable original, repository failure fail closed | Context propagated; deduplicated immutable value slice | Enforces ownership before solver work | Pages by 100; skips invalid unrelated catalog rows | Explicit snapshot contract | Paging and identity fixtures | PASS |
| validateRequest | Non-nil identity, nonempty entries, tolerance 0..100, unique typed exclusions | NaN, infinity, negative, nil and duplicate IDs fail | Pure | Request validation boundary | O(exclusions) | Shared by builder and validator | Numeric/identity tests | PASS |
| validateMeal | Solid/liquid, normalized basis, valid per-100 macros | Unsupported state, unavailable basis, negative/nonfinite/invalid macros fail | Pure | Prevents malformed nutrition in LP | O(1) | Central eligibility predicate | Mutation table; wrong unit/state is checked by quantity helper | PASS |
| targetForRequest | Sums persisted entries after canonical conversion | Missing meal, bad quantity/unit, invalid basis, nonfinite/negative/all-zero target fail | Pure | Server-authoritative target | O(entries) | Single target source | Authoritative, all-zero, metric/imperial tests | PASS |
| quantityInNutritionBasis | Solid g/oz and liquid ml/fl_oz only | Wrong state/unit and conversion errors fail; positive finite result required | Pure | Sanitizes persisted quantities | O(1) | Explicit state switch | Solid/liquid imperial tests | PASS |
| validateMacroTarget | Finite nonnegative macro values | NaN, infinity, negative fail | Pure | Numeric poison rejected | O(1) | Small helper | Numeric/all-zero tests | PASS |
| excludedMealIDs | UUID set with no nil/duplicate values | Invalid exclusions observable | Pure map | Typed exclusion boundary | O(n), request parser bounds list | One representation | Typed exclusion tests | PASS |
| originalMealIDs | Indexes authoritative saved-diet IDs | Nil IDs are rejected by request/target validation | Pure map | Uses persisted identity | O(entries) | Minimal helper | Eligibility/diversity fixtures | PASS |
| buildAlternativeConstraints | Hard-excludes greatest positive base-unit quantity; UUID tie-break | Unknown, nonfinite/negative, excluded positive, nonpositive-only inputs fail | Pure; map iteration cannot alter deterministic comparison | Validates prior solver output | One row per prior solution | Explicit bounded heuristic | Reversed order and quantity-drift test | PASS |
| NewDiversityPenalizer | Marks original saved-diet IDs without forbidding them | Upstream request rejects malformed identity | Pure map | Uses saved identity only | O(entries) | Simple constructor | Original eligibility test | PASS |
| GenerateAlternatives | Rebuilds models with prior hard exclusions and returns capped distinct sets | Context, solver, model, validator errors map safely; Task 220 later ordering semantics excluded | Context checked between attempts; no goroutine/shared state | Validates solver output before publication adapter | Attempts capped at 3x limit; bounded result count | Shared Task 219 objective pass excluded | Generation/worker fixtures pass | PASS |
| ValidateSolution | Explicit meal snapshot, recomputed macros/calories, request identity | Unknown/excluded/malformed/out-of-bound/invalid totals fail | Pure snapshot copy at constructor | Solver output and repository snapshot boundary | O(meals + solution) | Explicit meals; no fallback request field | Recalculation, malformed, tolerance tests | PASS |
| SolutionValidator.Validate | Same validator contract plus shared validateRequest call | Nil validator, nil identity, empty/zero/malformed/unknown/excluded/overbound output fail safely | Pure; no resources/cancellation | Direct public boundary now fails closed on identity | One meal index and output projection | Reuses request predicate | Valid control, nil ID, nil owner, malformed output tests | PASS |
| DefaultMaxQuantity | Removed caller-defined bound | N/A — removed | N/A | Removes bound injection | N/A | No obsolete alias | Repository search | PASS |
| alternativeOverlapLoss | Removed ambiguous overlap loss | N/A — removed | N/A | Removes quantity-drift distinctness | N/A | Replaced by hard ID-set heuristic | Distinctness source audit | PASS |
| originalMealsToEntries | Removed test-only target adapter | N/A — removed | N/A | Removes alternate target authority | N/A | One request vocabulary | Obsolete search | PASS |
| canonicalTarget | Removed caller target canonicalizer | N/A — removed | N/A | Prevents client target authority | N/A | Target only from persisted entries | Target tests | PASS |
| solutionMaxQuantity | Removed request-derived bound | N/A — removed | N/A | Prevents request bound injection | N/A | Shared maximum policy | Ceiling search | PASS |
| OptimizationJob | Queue envelope carries IDs/tolerance/exclusions, not target totals | Serialization validation handles malformed jobs | Redis lifecycle is owned by worker store; type has no shared mutable state | User/diet IDs remain explicit | Smaller bounded payload | No divergent solver identity | Worker fixtures and full suite | PASS |
| RepositoryOptimizationInputLoader.Load | Converts job metadata to repository-backed request | Nil loader/builder and repository errors fail | Context propagated; exclusion slice copied | Owner/diet passed to exact loader | One bounded repository load | Thin adapter | Worker integration | PASS |
| OptimizationProcessor.ProcessOptimizationJob | Loads authoritative snapshot, builds model, solves, validates, publishes | Parse/load/build/solve/publication errors route to safe failure/retry paths | One deadline context; no new goroutines; finalization context is delegated existing worker behavior | Queue owner and repository ownership boundaries preserved | One input load and capped generation | Explicit model/request/meals boundary | Full and race worker suites | PASS |
| TestConstraintDomainUsesOneProductionVocabulary | Rejects aliases and request extras | Fails on field drift | N/A — test | Detects duplicate trust surfaces | Reflection only | Deterministic | Covers canonical fields | PASS |
| TestBuildConstraintsUsesAuthoritativeSavedDietAndCanonicalDomain | Saved entries drive target and exclusions omit variables | Model/row-count failures are asserted | N/A | Exercises typed boundary | Small fixture | Direct contract proof | Excluded candidate and bounds | PASS |
| TestBuildConstraintsNormalizesSupportedOriginalDietBases | Metric/imperial bases produce equal model | Conversion mismatch fails | N/A | Quantity boundary | Small table | Deterministic | Solid and liquid | PASS |
| TestBuildConstraintsEligibilityPolicy | Invalid candidates filter; invalid originals fail | State, unavailable basis, invalid macros, zero-info cases challenged | N/A | Catalog trust boundary | Small fixture | Table-driven mutation | Wrong state/unit direct adversary is additionally covered by target unit switch | PASS |
| TestBuildConstraintsRejectsAllZeroAuthoritativeTargetAndInvalidIdentity | Zero target and incomplete identity fail | Nil ID/owner and all-zero cases explicit | N/A | Identity/target boundary | Small fixture | Direct | Builder identity; direct validator identity is separate repair test | PASS |
| TestBuildConstraintsMealSetDistinctnessCannotUseQuantityDrift | Highest quantity and changed ID set required | Reversed candidate order and same-set drift challenged | N/A | Prior solver output boundary | Small fixture | Deterministic heuristic proof | Tie/order semantics asserted | PASS |
| TestBuildConstraintsDeterministicFeasibleAndInfeasibleFixtures | Bounds classify assignments | Over-ceiling quantity is infeasible | N/A | LP matrix boundary | Small fixture | Deterministic | Feasible and infeasible cases | PASS |
| TestConstraintBuilderLoadsEligibleCatalogInBoundedPages | Loader pages, deduplicates, filters states/bases | Ineligible candidate and three-page total challenged | Fake context; no concurrency | Repository filter boundary | Three bounded pages | Deterministic fake | Query state assertion | PASS |
| TestConstraintBuilderRequiresExactRepositoryDietIdentity | Loaded diet ID and owner must match request | Nil and mismatched identities fail | N/A | Authorization boundary | Small fixture | Direct | Exact identity cases | PASS |
| TestBuildConstraintsRejectsInvalidNumericAndTypedExclusionInputs | Bad numeric/exclusion/prior solution rejected | Inf, negative, nil, duplicate, NaN prior quantity | N/A | Numeric/exclusion boundary | Table-driven | Direct | Adversarial inputs | PASS |
| TestTask218PackagedCLPConstraintFixture | Built matrix is solver-consumable | Native solver error fails test | CLP lifecycle delegated wrapper | Solver process boundary | Real packaged executable | Integration proof | Current focused/full suite | PASS |
| eligibleConstraintMeal | Produces valid normalized test meal | Tests mutate state/basis/macros | N/A — test fixture | Test-only data | O(1) | Central fixture | Used by domain tests | PASS |
| savedDietConstraintRequest | Produces exact owner/diet request | Tests mutate quantity/unit/tolerance | N/A — test fixture | Test-only data | O(1) | Central fixture | Used broadly | PASS |
| pagedConstraintMealRepository.Search | Simulates supported paged search | Returns deterministic bounded pages and total | Counters only | Test-only repository | O(page) | Minimal fake | Loader paging test | PASS |
| TestBuildConstraintsPenalizesOriginalMealsWithoutForbiddingThem | Original remains eligible with attached diversity coefficient | Bound/penalty assertions fail on hard-forbid regression | N/A | Test-only | Small fixture | Direct | Shared Task 219 objective fields excluded | PASS |
| TestGenerateAlternativesReturnsDeterministicOneOrTwoResults | Hard distinctness/cap behavior | Solver fixtures challenge selected IDs | Context passed to injected solver | Solver-output boundary | Small fixture | Current objective pass is excluded | Focused suite | PASS |
| TestGenerateAlternativesUsesLexicographicPassesAndCapsResults | Current shared compatibility fixture | Task 219 pass count/ordering is not used as Task 218 proof | N/A — test | Shared test only | Small fixture | Objective semantics excluded | Domain request and hard exclusion remain compatible | PASS |
| TestGenerateAlternativesRejectsSolverOutputThatViolatesHardExclusion | Excluded output fails | Invalid/empty output paths challenged | Injected solver; no leaks | Solver-output boundary | Small fixture | Direct | Focused suite | PASS |
| diversityMeals | Produces normalized candidate fixtures | IDs/macros controlled | N/A — test fixture | Test-only | O(n) | Central fixture | Diversity tests | PASS |
| diversityRequest | Produces exact saved-diet request | Exclusions controlled as UUIDs | N/A — test fixture | Test-only | O(1) | Central fixture | Diversity tests | PASS |
| TestBuildObjectiveRanksFeasibleFixturesByServerCalories | Current shared caller uses Task 218 request domain | Objective assertions are Task 219 and excluded | N/A — test | No new trust boundary | Small fixture | Compatibility only | Focused suite | PASS |
| TestBuildObjectiveUsesServerCalculatedCaloriesAndStableEqualTies | Current shared caller uses repository meals | Objective assertions are Task 219 and excluded | N/A — test | No new trust boundary | Small fixture | Compatibility only | Focused suite | PASS |
| objectiveMeal | Normalized compatibility fixture | Controlled macro variants | N/A — test fixture | Test-only | O(1) | Central fixture | Objective tests | PASS |
| objectiveRequest | Exact saved-diet compatibility request | Exclusion/tolerance controlled | N/A — test fixture | Test-only | O(1) | Compatibility only | Objective tests | PASS |
| TestValidateSolutionRecomputesEveryAcceptedAlternativeFromRepositoryMeals | Output totals derive from snapshot, not solver totals | Solid/liquid and deterministic order checked | N/A | Solver output boundary | Small fixture | Direct | Recalculation assertions | PASS |
| TestValidateSolutionAcceptsToleranceBoundariesAndFloatingPointEpsilon | Accepted band boundaries are stable | Boundary and material invalid quantity cases | N/A | Numeric publication boundary | Table-driven | Scale-aware tolerance | Focused suite | PASS |
| TestSolutionValidatorRequiresSavedDietIdentity | Direct validator requires identity | Valid control, nil ID, nil owner | N/A | Closes prior identity gap | Small fixture | Regression-specific | All subtests pass | PASS |
| TestValidateSolutionRejectsMalformedQuantitiesIDsAndAlternatives | Malformed outputs become safe validation failures | Empty, zero, negative, NaN, Inf, unknown, excluded cases | N/A | No internal detail leakage | Small fixture | Direct | Known excluded branch is source-audited; fixture uses an absent excluded meal | PASS |
| TestGenerateValidatedAlternativesPreservesValidPartialResultsAfterLaterSolveFails | Valid partial publication survives later solver error | Solver failure and safe code asserted | Context passed to generator | Safe error boundary | Small fixture | Publication adapter | Current full suite; later Task 219 timing is excluded | PASS |
| validatorMeal | Produces normalized validator fixture | Macro/state variants controlled | N/A — test fixture | Test-only | O(1) | Central fixture | Validator tests | PASS |
| validatorRequest | Produces exact identity/entry request | Entries, tolerance, exclusions controlled | N/A — test fixture | Test-only | O(entries) | Central fixture | Validator tests | PASS |
| TestOptimizationProcessorAppliesOneDeadlineAcrossAlternativeSolves | Worker fixture carries exact identity and normalized meals | Deadline/partial behavior passes in current worker suite; objective pass semantics excluded | Injected solver observes context | Worker boundary | Small fixture | Compatibility fixture | Full and race suites | PASS |
| TestRunPublishesAlternativeBeforeAcknowledgingQueuedJob | Worker publication order and request snapshot | Current integration path passes | Redis/solver fixtures own resources | Queue boundary | Integration fixture | Direct worker behavior | Full and race suites | PASS |
| TestTask206BackendIntegrationGate | End-to-end invalid excluded-domain result is safe validation failure | API/DB/Redis/CLP paths pass | External resources managed by fixture | API and queue boundary | Full integration | Shared prior-task fixture | Full backend suite | PASS |

## 7. Findings

| Severity | File:line | Symbol | Problem | Evidence/trigger | Required repair or disposition |
|---|---|---|---|---|---|
| OPTIONAL | backend/internal/optimization/validator_test.go:110-137 | TestValidateSolutionRejectsMalformedQuantitiesIDsAndAlternatives | The case named excluded meal uses validatorMealB, which is absent from the supplied snapshot, so that case proves unknown-ID rejection before the known-exclusion branch. | Source inspection confirms Validate checks the typed excluded set before accepting positive output; the direct known-exclusion branch is also exercised by the diversity/generator path. | Add a known excluded meal to this fixture in a future validator test expansion; no Task 218 production repair required. |
| OPTIONAL | backend/internal/optimization/diversity.go:111-118 and Task 220 row | GenerateAlternatives | Current shared pipeline records a canonical result before the duplicate/model-validation ordering owned by Task 220. | Task 220 explicitly owns model-validation-before-deduplication, immutable snapshot, nil-context, and attempt-exhaustion semantics; no Task 218 behavior is changed here. | Leave for Task 220; do not broaden Task 218. |

~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 2
~~~

## 8. Commands Run

| Command | Working directory | Exit code | Result | Log/artifact |
|---|---|---:|---|---|
| git rev-parse HEAD | repository | 0 | PASS | a4e31367485b03269e90b5607f2057c9568bb5b1 |
| git diff --name-status a4e31367485b03269e90b5607f2057c9568bb5b1 | repository | 0 | PASS | Dirty worktree enumerated; shared Task 219 paths excluded by manifest |
| go test ./internal/optimization ./internal/worker -count=1 | backend | 0 | PASS | Focused optimization 0.118s; worker 3.202s |
| go test ./internal/optimization -run '<Task 218 focused test set>' -count=1 -v | backend | 0 | PASS | Identity, eligibility, all-zero, bounds, distinctness, validator, partial-result, and packaged CLP tests |
| go test ./... -count=1 | backend | 0 | PASS | All backend packages, including repository and worker integration |
| go test -race ./... -count=1 | backend | 0 | PASS | All backend packages race-clean |
| go test ./internal/optimization -coverprofile=/tmp/task-218-optimization-current.coverage.out -count=1 | backend | 0 | PASS | 83.3% package statement coverage |
| go tool cover -func=/tmp/task-218-optimization-current.coverage.out | backend | 0 | PASS | Report at /tmp/task-218-optimization-current.coverage.out |
| go test ./internal/worker -coverprofile=/tmp/task-218-worker-current.coverage.out -count=1 | backend | 0 | PASS | 59.8% package statement coverage |
| go tool cover -func=/tmp/task-218-worker-current.coverage.out | backend | 0 | PASS | Report at /tmp/task-218-worker-current.coverage.out |
| go vet ./... | backend | 0 | PASS | No diagnostics |
| go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./... | backend | 0 | PASS | No vulnerabilities reachable from imported code |
| python3 scripts/validate-traceability.py | repository | 0 | PASS | Traceability validation passed |
| python3 scripts/validate-phase07-go-doc.py | repository | 0 | PASS | Phase 07 exported Go Doc validation passed |
| python3 scripts/validate-task-list.py | repository | 0 | PASS | 237 sequential tasks and ordered dependencies valid |
| git diff --check | repository | 0 | PASS | No whitespace errors |
| python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-218-review.md | repository | pending | REQUIRED AFTER WRITE | Final structural gate |

Coverage note: package statement coverage is below 100%, but the accepted Phase 07 exception in docs/implementation/04_OPEN.md explicitly covers defensive/dependency/process-bootstrap branches. The changed Task 218 acceptance paths were run and manually audited; no new Task 218 exception or task-list edit was introduced.

## 9. Files Inspected and Staleness Fingerprints

Current content hashes were captured after the final test run and before this evidence overwrite. The prior Task 218 review and its preparation are stale as decision evidence because they contain the repaired identity failure and old hash set. The DESIGN-004 hash in the Task 218 preparation was superseded by the later Task 219 documentation repair; the current hash below matches task-219-preparation.md and was reviewed as the current source of truth, with Task 219 objective semantics excluded.

| File | Purpose | Finding | Hash algorithm | Content hash |
|---|---|---|---|---|
| docs/design/DESIGN-004.md | Current solver/domain source of truth | Current interfaces and ownership text | SHA-256 | e944ab313d017d827f17304f6e0bf8a73f1d403dd79e236b9c97932bed81595b |
| docs/architecture/ARCH-004.md | Solver architecture boundary | No Task 218 contradiction | SHA-256 | bedcf45c79f50cfe24313345ac2ba664130ee44d7d2f35c7b1de0a06e0abe867 |
| docs/implementation/04_OPEN.md | Accepted coverage and later-task boundaries | No blocking open point for Task 218 | SHA-256 | c60ad3f42f5bcb66ed6d143c0f05b761ae80b3fb8be5f1541ea340d65c26c527 |
| docs/implementation/02_TASK_LIST.md | Status/dependency source | 218/219 PREPARED; 220 OPEN; not edited | SHA-256 | 5cea9418e48077e1a2fadf9516d9768cc5228866e9fc2ee441a9e76c48d987de |
| backend/internal/optimization/constraints.go | Task 218 builder/eligibility | Pass | SHA-256 | 3e13e8140a487af67dfd596b88d6db5a0dada708fba666742cdc92fdd393f84b |
| backend/internal/optimization/constraints_test.go | Domain/bounds/eligibility tests | Pass | SHA-256 | 3bb53ea9bdce760eec5a07aa6ca0d7a11c0c41006baf360940f6ab6a93a3514d |
| backend/internal/optimization/diversity.go | Distinctness caller; Task 219 objective overlap | Pass; Task 219 symbols excluded | SHA-256 | f11b10b9bcd4314332583d80062292c67db41168cd1909910fc21c0188f88b6f |
| backend/internal/optimization/diversity_test.go | Distinctness and shared objective fixtures | Pass; Task 219 assertions excluded | SHA-256 | 58ae20f87513951b1f856a885733b5c54ec2648b9036855c673954d606a7755a |
| backend/internal/optimization/validator.go | Reusable validation boundary and repair | Pass | SHA-256 | 8b26b9fbe50e5ec9e1d3b78f970614373777ea71561d19f20963a8aa60c587fd |
| backend/internal/optimization/validator_test.go | Validator and identity regression tests | Pass | SHA-256 | 79b49bf7f11eac0f1526748681213b1cb95665d0b48976103825879ca8db3c96 |
| backend/internal/optimization/objective.go | Task 219 dependency boundary | Inspected, excluded | SHA-256 | 03461f5deb4b76e673216658c78269cd37042a7c0476baf8f459ee15a9764c35 |
| backend/internal/optimization/objective_test.go | Task 219 compatibility callers | Inspected, excluded | SHA-256 | 5957da3993b36ea7aa20ef99fc3ffb7cd2e7a224a60c3c495cc7aacaa625a979 |
| backend/internal/optimization/clp_wrapper.go | Variable bounds/objective serialization | Pass | SHA-256 | da9ae4b9862c67ab18848a2829b763034d54d684add4eefbe08dae868b8451c7 |
| backend/internal/worker/optimization_processor.go | Worker request/snapshot caller | Pass | SHA-256 | 4db1a08c09598077cea9ffdbde347df9dab01a3f76cd61617ac0330a20b202b0 |
| backend/internal/worker/optimization_processor_deadline_test.go | Worker compatibility fixture | Pass | SHA-256 | e7660af899156cec6e07b52cee3cc1b2c43a3b31a465e4d735406082b848f180 |
| backend/internal/worker/task210_swe5_integration_test.go | Shared worker integration fixture | Pass | SHA-256 | 28ff63596079557f1b9995a4a2ae6af0cdd8c33c4e884156b806747e83b83448 |
| backend/internal/worker/worker_integration_test.go | Queue publication fixture | Pass | SHA-256 | 935c3caa093d220942a7e468355881ad38f94d29d36f18bc983bde08bda1615c |
| backend/internal/app/task206_backend_integration_test.go | End-to-end compatibility fixture | Pass | SHA-256 | 7ba576d1700d7b8f19cc1b99ca57ca69728aedcda0ee091dfcb3c20d3fa58042 |
| backend/internal/repository/meal_repository.go | Repository meal snapshot/input boundary | Pass | SHA-256 | 14e5a89572b5d8b63e18d5b37f9215da00e97cacd4c8f11e1d51cc21a2e6d0b9 |
| backend/internal/repository/macros.go | Normalized macro validation dependency | Pass | SHA-256 | fe08f2fe0a693b99b413153ca190ec1db0e40e2140bf8d79cdfd86c186381af2 |
| backend/internal/repository/types.go | Physical states, meals, saved diets | Pass | SHA-256 | c1c2ce654f89100b093efdf0dfa5182f535b549c2c8c2a34c6a8ed8689d0511f |
| backend/internal/repository/units.go | g/ml and imperial conversion dependency | Pass | SHA-256 | 9d9a8296654cc4b57e13bfb0090f15dced673d85782abc39675d8e2967463127 |
| backend/internal/repository/sql/meal_search.sql | Supported-state catalog filter | Pass; parameterized | SHA-256 | afc2a9f144a17cc17274a7cd88d99834627ed3af2f073ff2da500c9b757e4462 |
| backend/internal/repository/sql/meal_search_count.sql | Matching count/filter boundary | Pass; parameterized | SHA-256 | 32a799ab89a501cbec30e7ea94fbfb5251646633a8c794fcb6b297829d774bd8 |

~~~yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "docs/implementation/reviews/task-218-review.md before this overwrite: prior rejection and pre-repair identity/hash evidence superseded."
  - "docs/implementation/preparation/task-218-preparation.md: DESIGN-004 hash 856be5... superseded by the current Task 219 documentation repair; implementation hashes now match current state."
  - "docs/implementation/reviews/task-217-review.md: overlapping solver/validator/diversity hashes superseded by later Phase 07 work."
~~~

## 10. Coverage and Exceptions

- [x] Required focused coverage commands ran.
- [x] Report paths and observed thresholds are recorded.
- [x] Untested branches relevant to changed symbols were inspected.
- [x] The repository’s accepted Phase 07 coverage disposition was checked; no new Task 218 exception was invented.

~~~yaml
coverage_required: true
coverage_exception_allowed: false
coverage_report_path: "/tmp/task-218-optimization-current.coverage.out; /tmp/task-218-worker-current.coverage.out"
observed_line_coverage: "optimization 83.3%; worker 59.8%"
coverage_passed: true
~~~

Coverage finding: The commands completed successfully. The accepted Phase 07 exception documents remaining defensive/dependency/process-bootstrap gaps; Task 218’s changed domain, eligibility, identity, bounds, target, distinctness, validator, loader, and worker paths have focused tests plus line-by-line inspection. This is not a reason to reject the task.

## 11. Negative and Regression Checks

- [x] Existing focused optimization tests pass.
- [x] Full backend tests pass.
- [x] Full backend race detector passes.
- [x] No unrelated dependency or architectural boundary was introduced by the Task 218 repair.
- [x] Current DESIGN-004 and ARCH-004 do not contradict the Task 218 domain boundary.
- [x] No generated/cache/build/temporary artifact was added by this reviewer.
- [x] Public API additions are necessary and used; obsolete aliases were searched and are absent.
- [x] Error, malformed-input, quantity-bound, all-zero, unusable-original, exclusion, ordering, and repository/worker paths were challenged.

Findings: SQL catalog filters are parameterized and state-restricted. Repository loading is owner-scoped and uses a fresh request meal snapshot. Solver bounds are serialized in the LP Bounds section. The direct validator identity gap from the prior review is repaired. The two optional notes are test-quality/future-task disposition only and do not block Task 218.

## 12. Decision

A task may be PASSED only when all acceptance criteria and symbol audits pass, evidence is current, every reviewed file is hashed, and no blocking/important finding remains. Those conditions are met.

Before accepting the decision, run:

~~~bash
python3 /home/wiktor/.agents/skills/phase-orchestrator/scripts/validate_review_evidence.py docs/implementation/reviews/task-218-review.md
~~~

~~~yaml
decision: "PASSED"
reason: "The repaired current Task 218 domain and validator boundaries satisfy every criterion, current tests/race/static checks pass, Task 219 is explicitly excluded, and all reviewed files have current hashes."
failed_criteria:
  - ""
failed_or_unaudited_symbols:
  - ""
recommended_next_action: "None for Task 218; retain Task 219 as PREPARED and review it independently before processing Task 220."
~~~

## 13. Repair Context

Not applicable: the current independent review decision is PASSED. The prior rejection is superseded by this evidence file; the repaired validator identity path and current hashes are the authoritative review state.
