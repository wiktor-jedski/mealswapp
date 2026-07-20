# Review Evidence: Task 220 — Alternative Generation and Canonical Validation Pipeline

~~~yaml
task_id: 220
component: "Phase 07.01 Alternative Generation and Canonical Validation Pipeline"
static_aspect: "DESIGN-004: DiversityPenalizer"
input_status: "PREPARED"
review_decision: "PASSED"
reviewed_at_utc: "2026-07-17T12:30:34Z"
review_agent: "Codex independent owner review"
evidence_file: "docs/implementation/reviews/task-220-review.md"
baseline_ref: "HEAD a4e31367485b03269e90b5607f2057c9568bb5b1"
baseline_confidence: "MEDIUM"
code_review_skill_invoked: true
relevant_language_guide: "/home/wiktor/.agents/skills/code-review-skill/reference/go.md; /home/wiktor/.agents/skills/code-review-skill/reference/cross-cutting/async-concurrency-patterns.md; DESIGN-004"
repair_context_required: true
~~~

## 1. Task Source

**Description:** Phase 07.01: canonicalize and model-validate every solver result before deduplication or iteration-state mutation, use one scale-aware quantity tolerance and deterministic evaluation order, define partial-result behavior when the attempt budget is exhausted, and build one immutable indexed repository snapshot for one authoritative validation/publication pass per result.

**Depends On:** 219 (PREPARED in the current task table; its required behavior was inspected as dependency context).

**Testing Coverage Exceptions:** No Task 220 exception is recorded in the task row. The repository-level Phase 07 coverage exception in docs/implementation/04_OPEN.md remains the only exception considered.

**Verification criteria:** Instrumented tests prove one index build and one validation/projection per accepted solver result; invalid duplicates are rejected before retry/state mutation; previous/current solutions share canonicalization; exact-model validation precedes deduplication; evaluation is deterministic; malformed inputs fail safely; the 100/101 OpenAPI boundary is enforced; approved partial results survive later failures or exhaustion; snapshots are mutation-safe; metric/liquid, tolerance, exclusion, ordering, race, and signature checks pass.

## 2. Pre-Review Gates

- [x] Input status is PREPARED.
- [x] Dependency 219 is eligible in the current task table.
- [x] Refreshed Task 220 preparation was read in full.
- [x] The prior rejected review was read in full, including F-1/F-2/F-3 and repair context.
- [x] Current production callers, tests, DESIGN-004, OpenAPI, and task criteria were inspected.
- [x] code-review-skill was invoked exactly once; its full template/checklist and applicable Go/concurrency guidance were read.
- [x] This review is independent of the repair and uses the current worktree rather than preparation claims as test substitutes.
- [x] No production code, task status, or unrelated concurrent work was edited.

~~~yaml
pre_review_gates_passed: true
blocking_issue: "None. Prior F-1, F-2, and F-3 findings are closed by current source and public-path tests."
~~~

## 3. Review Baseline and Change Surface

The fixed repository reference is a4e31367485b03269e90b5607f2057c9568bb5b1. The worktree is intentionally shared and dirty, so Task 220 ownership was reconstructed from the fixed HEAD, the refreshed preparation report, the previous review, current symbols/callers, and focused tests. Task 221+ overlays in validator.go/validator_test.go and other concurrent files were inspected only where they overlap the Task 220 boundary and were not attributed to this task.

The repaired production sequence is now:

1. GenerateAlternatives and GenerateValidatedAlternatives cap the requested result count before deriving the attempt budget.
2. newSolutionValidator creates one detached meal map and UUID-sorted ID sequence; real context-scoped instrumentation observes this constructor boundary.
3. Every attempt calls buildConstraintsFromIndex with that same map and sorted IDs, rather than rebuilding a meal index.
4. Primary and secondary solver outputs are canonicalized and exact-model checked before final publication validation.
5. SolutionValidator.Validate projects once from the same snapshot; only then are duplicate/state/output mutations committed.

Prior findings were specifically rechecked:

| Prior finding | Reproduction/condition | Current result |
|---|---|---|
| F-1 blocking: limit*3 overflow and uncapped over-limit budget | Call either public generator with limit=4 or MaxInt. | Closed. alternativeGenerationLimits caps to 3 first and returns budget 9; public tests cover 4, MaxInt, 0, and -1 for both generators. |
| F-2 important: model construction rebuilt a map per attempt | Generate multiple alternatives and inspect the model-builder boundary. | Closed. The generation loop calls buildConstraintsFromIndex(validator.meals, validator.orderedMealIDs, ...) directly. BuildConstraints retains a separate compatibility boundary for standalone callers. |
| F-3 important: build counter was disconnected from production | Exercise both exported generators with a constructor/index probe. | Closed. immutableMealSnapshot invokes the context probe, and the public-path test asserts one build and one projection per accepted result for both generators. |

The current worktree briefly showed a concurrent overlay during an early test invocation (missing ValidatePublishedAlternatives/duplicate test declaration, then a missing traceability comment); immediate reruns after the overlay settled passed. These were outside Task 220 ownership and were not used to mask a Task 220 result.

## 4. Acceptance Criteria Checklist

| # | Criterion | Result | Current evidence |
|---:|---|---|---|
| 1 | One immutable repository index is built for a generation call | PASS | newSolutionValidator calls immutableMealSnapshot once; the loop consumes its map/IDs through buildConstraintsFromIndex; TestPublicAlternativeGeneratorsBuildOneIndexAndProjectEachResultOnce observes the real constructor boundary through both exports. |
| 2 | One validation/projection occurs per accepted solver result | PASS | The common pipeline invokes validator.Validate once after the primary/secondary policy; the public instrumentation test records exactly two projections for two accepted results in each adapter. |
| 3 | Invalid duplicate output is rejected before retry/state mutation | PASS | solveObjectivePolicy exact-model-validates before the pipeline computes/commits deduplication state; TestAlternativePipelineRejectsInvalidDuplicateBeforeStateMutation expects one retained result, one projection, and a validation failure on the invalid duplicate. |
| 4 | Previous and current solutions share zero/residue canonicalization | PASS | canonicalQuantities is used for solver outputs and publication; the residue test proves prior exclusion contains only the canonical positive meal. |
| 5 | Exact-model validation precedes deduplication | PASS | Both primary and secondary results pass solutionSatisfiesModel; the duplicate key is not checked or committed until after validator.Validate. |
| 6 | Constraint/objective evaluation is deterministic | PASS | Meal IDs and coefficient IDs are sorted before model/evaluation accumulation; the deterministic test repeats the adversarial numeric fixture 100 times. |
| 7 | Nil context and malformed snapshots fail safely | PASS | Focused cases cover nil context, nil/empty snapshot, duplicate IDs, and nil IDs; no solver call occurs and the result is empty with failed_validation. |
| 8 | Selected-meal cardinality matches OpenAPI 1..100 | PASS | Validation canonicalizes residue before counting; the 100-plus-residue case passes and 101 positive meals fails. OpenAPI confirms minItems 1, maxItems 100. |
| 9 | Approved later failures preserve only prior valid partial results | PASS | State is appended only after all validation/projection stages; existing worker/validator partial-failure tests preserve one valid alternative and return a safe terminal code. |
| 10 | Attempt-budget exhaustion returns approved partial results | PASS | alternativeGenerationLimits returns (3,9) after capping; the private exhaustion fixture returns one accepted result with nil error; public large-limit tests prove six solver calls for three results in both adapters. |
| 11 | Caller mutation cannot change the snapshot | PASS | Request entries/exclusions and meal slices are detached before solving; the public mutation fixture changes caller values during the first solve and still receives the original valid result. |
| 12 | Metric and liquid projections are canonical | PASS | Projection uses repository data, maps solid to g and liquid to ml, sorts UUIDs, and is exercised concurrently. |
| 13 | One scale-aware tolerance is used at quantity/model/macro/parser boundaries | PASS | quantityTolerance is shared by canonicalization, model checks, publication quantity/macro checks, and CLP residue parsing; boundary and residue tests pass. |
| 14 | Exclusions remain enforced | PASS | Excluded IDs are omitted from model variables and rejected at publication; prior exclusion constraints also reject solver assignments that violate them. |
| 15 | Deterministic output ordering remains stable | PASS | Snapshot IDs, coefficient IDs, prior constraints, and projected meals use deterministic ordering; existing and focused tests pass. |
| 16 | Shared snapshot reads are race-safe | PASS | Snapshot/index data is read-only after construction; affected-package and full-backend go test -race pass. |
| 17 | DESIGN-004 public signatures match the implementation | PASS | BuildConstraints, ValidateSolution, GenerateAlternatives, GenerateValidatedAlternatives, and injected AlternativeSolveFunc match current Go callers and DESIGN-004. |

## 5. Changed-Symbol Inventory

| # | Symbol/unit | File:line | Surface and consumer | Tests/evidence |
|---:|---|---|---|---|
| 1 | BuildConstraints | constraints.go:103 | Standalone compatibility index boundary | Existing constraint tests; source audit |
| 2 | buildConstraintsFromIndex | constraints.go:114 | Shared indexed model assembly per generation attempt | Public generation tests; source audit |
| 3 | GenerateAlternatives | diversity.go:74 | Raw solver-result generation adapter | Public generator tests |
| 4 | alternativeGenerationLimits | diversity.go:88 | Capped result/attempt policy | Large, zero, and negative limit tests |
| 5 | validatedAlternative | diversity.go:99 | Couples one canonical solution with one projection | Pipeline tests |
| 6 | generateAlternativePipeline | diversity.go:110 | Canonical validation, projection, deduplication, and state ordering | Pipeline tests and source audit |
| 7 | solveObjectivePolicy | diversity.go:166 | Primary/secondary exact-model solver policy | Objective/diversity and CLP tests |
| 8 | objectiveValueForSolution | diversity.go:220 | Deterministic objective evaluation | Deterministic test |
| 9 | canonicalSolution | diversity.go:239 | Model-known-ID and solver-output canonicalization | Residue/duplicate tests |
| 10 | canonicalQuantities | diversity.go:256 | Shared sparse quantity normalization/keying | Residue/cardinality tests |
| 11 | solutionSatisfiesModel | diversity.go:277 | Exact bounds/constraint validation | Duplicate/determinism tests |
| 12 | sortedCoefficientIDs | diversity.go:309 | Stable coefficient traversal | Deterministic test |
| 13 | cloneOptimizationRequest | diversity.go:344 | Caller-owned request detachment | Mutation test |
| 14 | NewSolutionValidator | validator.go:211 | Public snapshot validator constructor | Existing validator tests |
| 15 | newSolutionValidator | validator.go:217 | Instrumented generation constructor | Public index instrumentation test |
| 16 | immutableMealSnapshot | validator.go:417 | Detached map and sorted ID construction | Malformed/mutation/race tests |
| 17 | SolutionValidator.Validate | validator.go:236 | One authoritative publication projection | Projection and boundary tests |
| 18 | GenerateValidatedAlternatives | validator.go:360 | Publication-safe generation adapter | Public generator/worker tests |
| 19 | generationInstrumentation | validator.go:188 | Real constructor/projection test probes | Public index instrumentation test |
| 20 | generationInstrumentationFromContext | validator.go:201 | Context-scoped probe lookup | Source audit; public test |
| 21 | quantityTolerance | validator.go:439 | Shared numeric tolerance | Tolerance/residue tests |
| 22 | macroWithinTolerance | validator.go:451 | Scale-aware macro boundary | Validator tolerance tests |
| 23 | parseCLPSolutionLine | clp_wrapper.go:623 | Signed solver residue boundary | CLP parser tests |
| 24 | TestPublicAlternativeGeneratorsBuildOneIndexAndProjectEachResultOnce | task220_pipeline_test.go:16 | Public real-path index/projection evidence | Direct |
| 25 | TestPublicAlternativeGeneratorsCapAttemptBudgetBeforeMultiplication | task220_pipeline_test.go:53 | Public overflow/cap evidence | Direct |
| 26 | TestPublicAlternativeGeneratorsDoNotSolveNonPositiveLimits | task220_pipeline_test.go:85 | Non-positive boundary evidence | Direct |
| 27 | TestAlternativePipelineRejectsInvalidDuplicateBeforeStateMutation | task220_pipeline_test.go:115 | Invalid duplicate ordering | Direct |
| 28 | TestAlternativePipelineCanonicalizesResidueForCurrentAndPreviousSolutions | task220_pipeline_test.go:136 | Shared canonicalization | Direct |
| 29 | TestAlternativePipelineAttemptExhaustionReturnsValidPartialResults | task220_pipeline_test.go:159 | Exhaustion/partial contract | Direct |
| 30 | TestSolutionValidatorSelectedMealCountMatchesOpenAPIBoundary | task220_pipeline_test.go:169 | 100/101 publication boundary | Direct |
| 31 | TestAlternativePipelineRejectsNilContextAndMalformedSnapshots | task220_pipeline_test.go:192 | Safe malformed-input behavior | Direct |
| 32 | TestAlternativePipelineSnapshotIgnoresLaterCallerMutation | task220_pipeline_test.go:218 | Caller mutation isolation | Direct |
| 33 | TestDeterministicObjectiveAndConstraintEvaluation | task220_pipeline_test.go:237 | Stable numeric evaluation | Direct |
| 34 | TestSolutionValidatorConcurrentMetricAndLiquidProjection | task220_pipeline_test.go:256 | Concurrent read/projection evidence | Direct/race |
| 35 | TestTask219PackagedCLPLexicographicObjective | diversity_test.go:188 | Shared tolerance compatibility regression | Native CLP conditional test |

~~~yaml
inventory_source_count: 35
audited_symbol_count: 35
inventory_complete: true
generated_groupings:
  - "No executable Task 220 symbol was omitted; later Task 221-only publication symbols are excluded from attribution."
~~~

## 6. Function-Level Audit

| Symbol/unit | Correctness and edge paths | Security/concurrency/performance | Result |
|---|---|---|---|
| BuildConstraints | Creates one detached standalone snapshot and delegates to indexed assembly; malformed input returns typed validation errors. | No caller mutation; standalone allocation is bounded by supplied meals. | PASS |
| buildConstraintsFromIndex | Reuses the supplied map/ID order, validates request/targets/exclusions, and emits stable variables/constraints. | Read-only map use prevents per-attempt aliasing; no external I/O. | PASS |
| GenerateAlternatives | Uses capped limits, cloned request, common pipeline, and returns canonical raw solutions plus safe errors. | One constructor/index per call; context is propagated. | PASS |
| alternativeGenerationLimits | Non-positive limits produce zero attempts; positive values cap at 3 before multiplication, including MaxInt. | Arithmetic is bounded and overflow-safe. | PASS |
| validatedAlternative | Prevents a solution and publication projection from diverging after acceptance. | Small immutable value wrapper. | PASS |
| generateAlternativePipeline | Checks context/solver/snapshot, builds indexed models, validates solver outputs before projection/dedup/state commit, and preserves prior results on later safe failure. | Read-only snapshot, cloned prior maps, bounded 9 attempts; no goroutines. | PASS |
| solveObjectivePolicy | Validates both primary and secondary outputs against their exact models before returning. | Solver receives caller context; secondary model copies constraints. | PASS |
| objectiveValueForSolution | Rejects non-finite inputs/results and traverses coefficients deterministically. | Linear time in coefficient count. | PASS |
| canonicalSolution | Rejects unknown/duplicate model IDs and delegates residue/non-finite/positive-selection checks. | Solver output is treated as untrusted data. | PASS |
| canonicalQuantities | Removes signed residue, rejects material negatives/unknown IDs, and derives stable selected-set keys. | No mutation of solver-owned map. | PASS |
| solutionSatisfiesModel | Checks variable bounds, coefficient references, finite bounds, and all constraints with shared tolerance. | Deterministic accumulation; solver IDs are validated against the model. | PASS |
| sortedCoefficientIDs | Makes map traversal reproducible. | O(n log n), appropriate for bounded model construction/evaluation. | PASS |
| cloneOptimizationRequest | Detaches request slices used by the generation loop. | Prevents caller races on entries/exclusions after the boundary. | PASS |
| NewSolutionValidator | Builds the public validator snapshot boundary. | Ordinary public validation has no instrumentation side effect. | PASS |
| newSolutionValidator | Wires the single snapshot and optional real test instrumentation. | Constructor probe is context-scoped and absent in ordinary production calls. | PASS |
| immutableMealSnapshot | Rejects nil/duplicate IDs and creates deterministic read-only map/order data; copies optimization-relevant nested slices. | Shared reads are race-safe after construction. | PASS |
| SolutionValidator.Validate | Canonicalizes before cardinality/exclusion/quantity checks, recalculates macros/calories from repository data, and projects deterministic g/ml meals. | No solver totals or client totals are trusted. | PASS |
| GenerateValidatedAlternatives | Uses the same common pipeline and avoids a second raw-result projection loop. | Maintains worker context/partial-result semantics. | PASS |
| generationInstrumentation | Observes actual constructor/projection boundaries without changing normal behavior. | Probe callbacks are supplied only by package tests. | PASS |
| generationInstrumentationFromContext | Reads the package-private context probe safely. | No global mutable counters or exported instrumentation seam. | PASS |
| quantityTolerance | Applies one scale-aware policy at quantity/model/parser boundaries. | Tolerance is explicit and bounded by compared magnitudes. | PASS |
| macroWithinTolerance | Applies the shared scale-aware policy to recomputed macro bands. | Rejects non-finite macro inputs and margins. | PASS |
| parseCLPSolutionLine | Accepts only valid CLP rows and canonicalizes tiny negative residue through the shared tolerance. | Bounded parser fields and finite-number checks. | PASS |
| TestPublicAlternativeGeneratorsBuildOneIndexAndProjectEachResultOnce | Covers the repaired real public constructor/projection boundaries in both adapters. | Deterministic injected solver fixture. | PASS |
| TestPublicAlternativeGeneratorsCapAttemptBudgetBeforeMultiplication | Covers 4 and MaxInt without overflow and checks the bounded six solver calls. | Deterministic injected solver fixture. | PASS |
| TestPublicAlternativeGeneratorsDoNotSolveNonPositiveLimits | Covers zero and negative limits without invoking the solver. | Deterministic no-call assertion. | PASS |
| TestAlternativePipelineRejectsInvalidDuplicateBeforeStateMutation | Covers exact-model rejection before projection/dedup/state commit. | Invalid solver output cannot reach publication. | PASS |
| TestAlternativePipelineCanonicalizesResidueForCurrentAndPreviousSolutions | Covers shared signed-residue normalization in output and prior constraints. | Canonical map is copied before state use. | PASS |
| TestAlternativePipelineAttemptExhaustionReturnsValidPartialResults | Covers successful exhaustion with an accepted partial result. | Bounded private seam. | PASS |
| TestSolutionValidatorSelectedMealCountMatchesOpenAPIBoundary | Covers 100 positive meals plus residue and rejects 101. | Publication cardinality is explicit. | PASS |
| TestAlternativePipelineRejectsNilContextAndMalformedSnapshots | Covers nil context, empty, nil-ID, and duplicate-ID snapshots without solver calls. | Safe validation code and no diagnostic leak. | PASS |
| TestAlternativePipelineSnapshotIgnoresLaterCallerMutation | Covers request and meal mutation after snapshot construction. | Caller-owned data cannot race the read-only snapshot. | PASS |
| TestDeterministicObjectiveAndConstraintEvaluation | Repeats sorted numeric objective/constraint evaluation 100 times. | Deterministic map traversal. | PASS |
| TestSolutionValidatorConcurrentMetricAndLiquidProjection | Covers concurrent read-only projection and solid/liquid units. | Full race run passes. | PASS |
| Existing constraint/objective/validator/CLP regressions | Preserve prior eligibility, exclusion, objective, metric/liquid, tolerance, and packaged-CLP behavior. | Full backend and race suites pass. | PASS |

## 7. Findings

### Required Changes

None. No blocking correctness, security, regression, or acceptance-coverage finding remains for Task 220.

### Important Suggestions

None.

### Minor Suggestions

🟢 **[nit] Post-solve cancellation assertion:** generateAlternativePipeline checks ctx.Err() before each attempt and passes the context into both solver passes, but has no separate post-solver check before the current result is projected/committed. A deliberately context-ignoring injected solver could therefore commit a valid result after cancellation. Add a focused cancellation-after-solve fixture if the worker contract requires cancellation to win that race; current worker deadline/shutdown tests and the documented solver context boundary make this non-blocking for Task 220.

🟢 **[nit] Deep-copy scope:** immutableMealSnapshot copies the nested recipe/classification slices but classification ParentID pointers remain aliased. Optimization currently never reads those fields; either deep-copy those pointers or narrow the comment/test claim to optimization-relevant snapshot data in a future cleanup.

~~~yaml
blocking_findings: 0
important_findings: 0
optional_findings: 2
~~~

## 8. Commands Run

| Command | Result | Evidence |
|---|---|---|
| cd backend && GOCACHE=... GOMODCACHE=... go test ./internal/optimization -run 'Test(PublicAlternativeGenerators\|AlternativePipeline\|SolutionValidatorSelected\|DeterministicObjective\|SolutionValidatorConcurrentMetric\|ValidateSolution)' -count=1 | PASS | Current focused repaired surface. |
| cd backend && ... go test ./internal/optimization -count=10 | PASS | Repeated optimization package run. |
| cd backend && ... go test -race ./internal/optimization ./internal/worker -count=1 | PASS | Affected-package race run. |
| cd backend && ... go test ./... -count=1 | PASS | Full backend packages, including optimization/worker/httpapi. |
| cd backend && ... go test -race ./... -count=1 | PASS | Full backend race run. |
| cd backend && ... go vet ./... | PASS | Backend static analysis. |
| cd backend && ... go test ./internal/optimization -coverprofile=/tmp/task-220-optimization-current.coverage.out -count=1 && go tool cover -func=... | PASS | Fresh optimization statement coverage: 85.3%; repository exception remains documented. |
| python3 scripts/validate-task-list.py | PASS | 237 sequential tasks with ordered dependencies; task statuses unchanged. |
| python3 scripts/validate-traceability.py | PASS | Final rerun after concurrent overlay settled. |
| git diff --check | PASS | No whitespace errors. |

An earlier concurrent-worktree test invocation failed before package execution because Task 221 test edits were briefly inconsistent; the immediate rerun passed. The same occurred once for traceability while a concurrent declaration was mid-edit. These observations are recorded for reproducibility, not treated as Task 220 failures.

## 9. Files Inspected and Staleness Fingerprints

SHA-256 fingerprints below were captured after the final verification commands. Shared files include concurrent later-task overlays where noted; the overlapping Task 220 symbols were re-read from the same snapshot.

| File | Purpose | Attribution | SHA-256 |
|---|---|---|---|
| backend/internal/optimization/constraints.go | Indexed model construction and prior constraints | Task 220/shared | 9f1d72435bac344e8e5c0b4140c19d87392a993e8834c43f689cd24e86627db1 |
| backend/internal/optimization/diversity.go | Generation pipeline, canonicalization, model checks | Task 220/shared | 647547e6488f23455ab56f5042d3aa2ffbae1caee0f56b43cdb00fb99ae7ffd7 |
| backend/internal/optimization/validator.go | Immutable snapshot, projection, tolerance | Task 220 with later overlay | 4119ce67e0353c725d9e0feca9f13379966f1d89eaee5477bee706929ae6b09a |
| backend/internal/optimization/clp_wrapper.go | Solver residue parser boundary | Task 220 compatibility/shared | cc5079bf7475f8bea0e7d97327a9f511a7ca17c4fbdd11564da2bf2bf3e48996 |
| backend/internal/optimization/objective.go | Objective policy dependency | Dependency/source audit | 03461f5deb4b76e673216658c78269cd37042a7c0476baf8f459ee15a9764c35 |
| backend/internal/optimization/task220_pipeline_test.go | New repaired acceptance tests | Task 220 | 0704646e1bd48048dc95ca2320dd2018d6c5242cf8c0092b166b646f30eccea5 |
| backend/internal/optimization/diversity_test.go | Objective/CLP regression | Shared compatibility | 0a00afe09117ccf468477989b9beda2704db8e12d02eb09d860c5f2d797ae8fc |
| backend/internal/optimization/constraints_test.go | Constraint/model regressions | Shared/dependency | 3bb53ea9bdce760eec5a07aa6ca0d7a11c0c41006baf360940f6ab6a93a3514d |
| backend/internal/optimization/validator_test.go | Validator/partial-result regressions | Task 220 boundary with later overlay | 950349bf26c0a28df9b2f3037243b26d0ccd0c0203f69fb5d76b612ad1742f73 |
| backend/internal/worker/optimization_processor.go | Worker caller and partial publication boundary | Caller audit | cc5a4509725adcd751e043ac3f377aadf48be6796abad618b25af61a20f50807 |
| docs/design/DESIGN-004.md | Algorithm and interface source of truth | Design source | 1ee9b87165cb5b1eff43cf6a39b32c20ebab02ae8ab1cddc9f98c3c6152caf0d |
| api/openapi.yaml | Alternative meal cardinality contract | Publication source | 3387a030adc71fdf5d470077b1a5737b32b47059658ff25596dba03256afcbd5 |
| docs/implementation/02_TASK_LIST.md | Task/status/dependency source | Read-only status boundary | e57ae220a9a603aeba610f3e58992701b63ef5c42d2406bcd3bbac16ff79a1eb |
| docs/implementation/preparation/task-220-preparation.md | Refreshed preparation evidence | Input evidence | 32657f553c0419e80655a4d488f0724d60fc69faa6081223a054756a18752f0a |

~~~yaml
all_reviewed_files_hashed: true
prior_evidence_checked_for_staleness: true
stale_prior_evidence:
  - "The prior rejected review hashes describe the pre-repair snapshot and are not reused as current evidence."
  - "The refreshed preparation hashes for shared files can differ when concurrent Task 221+ edits land; current overlapping symbols were re-read and current hashes are recorded above."
  - "Transient compile/traceability failures were observed while concurrent edits were incomplete; final reruns passed."
~~~

## 10. Coverage and Exceptions

- [x] Fresh focused optimization coverage command ran.
- [x] Coverage artifact path and observed percentage are recorded.
- [x] Relevant defensive/error branches were source-audited.
- [x] No new Task 220 coverage exception was invented.

~~~yaml
coverage_required: true
coverage_exception_allowed: true
coverage_report_path: "/tmp/task-220-optimization-current.coverage.out"
observed_line_coverage: "85.3% internal/optimization statements"
coverage_passed: true
~~~

The percentage is not a standalone approval gate because the repository’s documented Phase 07 exception covers defensive branches; the acceptance-specific tests and full backend/race runs are the deciding evidence.

## 11. Negative and Regression Checks

### Strengths

- The repaired limit arithmetic is simple and bounded at the public adapters.
- Model construction and publication validation now have one explicit indexed snapshot boundary.
- Canonicalization, exact-model validation, projection, deduplication, and state mutation have a clear order.
- The public-path instrumentation test no longer counts a test helper; it observes the actual constructor/projection hooks.
- Partial results are coupled with their validated projections and do not expose solver diagnostics.
- Full backend tests and full backend race tests pass in the final snapshot.

### Architecture and Performance

- [x] Separation of concerns: snapshot/indexing, model assembly, objective policy, canonical validation, and publication projection remain distinct.
- [x] Dependency direction: the pure model builder consumes an explicit index; the worker injects the solver.
- [x] No per-attempt meal-map rebuild in the generation path.
- [x] Attempt count is bounded by 9 and result count by 3.
- [x] Map traversals used in numerical evaluation are sorted; no N+1 or new external I/O is introduced.

### Security Considerations

- [x] No secrets or credentials introduced.
- [x] Solver IDs are checked against the model and repository snapshot before publication.
- [x] Non-finite, negative, over-limit, excluded, duplicate, and malformed assignments fail safely.
- [x] Projection recalculates macros/calories/units from server-side meal data.
- [x] Internal solver/repository diagnostics remain behind typed safe failure codes.

### Test Coverage

- [x] Unit tests cover changed canonical/model/index/tolerance behavior.
- [x] Public adapter tests cover both raw and validated generation paths.
- [x] Edge cases cover overflow, non-positive limits, residue, duplicate state ordering, 100/101 cardinality, malformed snapshots, caller mutation, metric/liquid order, and concurrency.
- [x] Error and partial-result behavior is covered by optimization and worker tests.

No unrelated production file, task status, or concurrent task work was changed by this review.

## 12. Decision

The repaired Task 220 meets its task-row acceptance criteria. The prior blocking/important findings are closed, no new blocking or important finding remains, current focused and aggregate backend/race checks pass, and review evidence is complete.

~~~yaml
decision: "PASSED"
reason: "The attempt budget is capped before multiplication, the generation path shares one real immutable index between model construction and projection, and all Task 220 acceptance criteria are supported by current source/tests."
failed_criteria: []
failed_or_unaudited_symbols: []
recommended_next_action: "Keep the two documented nits for a later focused cancellation/structural-snapshot test; do not change Task 220 status as part of this review."
~~~

## 13. Repair Context

The prior review at 2026-07-17T10:34:38Z rejected Task 220 for F-1 (attempt-budget overflow) and F-2/F-3 (shared-index implementation/evidence defects). The refreshed preparation explicitly claimed repairs for all three. This independent review re-read the repaired symbols and exercised both exported generation paths.

Repair verification:

- F-1 is closed by alternativeGenerationLimits: positive limits are capped to MaxAlternativeCount before *3; zero/negative limits produce no solver calls.
- F-2 is closed by buildConstraintsFromIndex, which receives the validator’s immutable map and UUID order directly on each attempt; BuildConstraints is only the standalone compatibility entry point.
- F-3 is closed by generationInstrumentation wired into immutableMealSnapshot and SolutionValidator.Validate, with the public-path test asserting one real index build and one projection per accepted result for both adapters.

The current task-table row remains PREPARED; this review intentionally leaves it unchanged. Concurrent Task 221+ edits were preserved and excluded from Task 220 attribution except where their overlay was necessary to interpret current shared files.
