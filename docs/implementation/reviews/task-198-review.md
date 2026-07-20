# Review Evidence: Task 198 — DESIGN-004: ConstraintBuilder

## Decision

Recommended status: `PASSED`

Task 198 is `PREPARED`, dependency 192 is `PREPARED`, the repaired normalized iterative exclusion rejects near-duplicate quantities while permitting a materially different selected set, deterministic feasible/infeasible matrix fixtures are present, and all relevant checks pass.

## Task and Dependency Check

| Item | Required | Observed | Result |
|---|---|---|---|
| Task 198 status | PREPARED | PREPARED | PASS |
| Dependency 192 | PREPARED | PREPARED | PASS |
| Retry count | 0 | 0 | PASS |

## Repair Verification

### Normalized iterative exclusion

`buildAlternativeConstraints` now assigns each previously selected meal coefficient `1 / previousQuantity` and sets the upper bound to `selectedMealCount - 0.05`. The previous solution therefore evaluates to the selected-meal count and is rejected. The focused regression test confirms:

- `{A: 24.999, B: 100}` is rejected against previous `{A: 25, B: 100}` because its normalized overlap remains above `1.95`.
- `{A: 0, B: 100}` is allowed because removing meal A is a materially different selected set and evaluates below the bound.
- Coefficients and the complete model remain deterministic when repository candidates arrive in reverse order.

Result: PASS.

### Deterministic feasible/infeasible fixtures

`TestBuildConstraintsMatrixFixturesAreDeterministicAndClassifiable` is table-driven and contains:

- A feasible exact macro intersection with a known accepted assignment.
- A bounded infeasible intersection whose maximum reachable macros remain below required lower bounds.
- Reverse-input reconstruction and `reflect.DeepEqual` checks for deterministic matrices in both fixtures.

Result: PASS.

## Acceptance Criteria

| Criterion | Result | Evidence |
|---|---|---|
| Lower/upper macro bounds across tolerance | PASS | Protein, carbohydrate, and fat bands are constructed from targets and tolerance and asserted in focused tests. |
| Excluded IDs have zero eligibility | PASS | Excluded variables have zero upper bounds and explicit zero-valued exclusion constraints. |
| Invalid/non-finite rejection | PASS | Table-driven tests cover non-finite/negative targets, tolerance, meal macros, bounds, and prior quantities. |
| Repository macro scaling | PASS | Per-100 repository macros are converted to per-unit coefficients and asserted. |
| Persisted original diet | PASS | Targets are derived from persisted quantities and repository macro values. |
| Iterative alternatives | PASS | Normalized overlap rejects near duplicates and permits a materially different meal set. |
| Deterministic feasible/infeasible matrices | PASS | Table-driven fixtures classify both cases and compare reversed-input models. |

## Commands Run

Commands used repository-local Go caches where applicable.

| Command | Working directory | Result |
|---|---|---|
| `go test -count=1 ./...` | `backend/` | PASS |
| `go test -race -count=1 ./internal/optimization` | `backend/` | PASS |
| `go vet ./internal/optimization` | `backend/` | PASS |
| `git diff --check` | repository root | PASS |

## Files Inspected

- `docs/implementation/02_TASK_LIST.md`
- `docs/implementation/reviews/task-192-review.md`
- `docs/design/DESIGN-004.md`
- `backend/internal/optimization/constraints.go`
- `backend/internal/optimization/constraints_test.go`

## Findings

No blocking correctness, security, regression, or test-coverage findings remain within Task 198's scope.
