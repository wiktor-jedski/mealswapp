## FILE: ConstraintBuilder.md
**Traceability:** ARCH-004

### 1. Data Structures & Types

```go
// MacroTarget represents the target macronutrient values for optimization
type MacroTarget struct {
    Protein float64 // grams
    Carbs   float64 // grams
    Fat     float64 // grams
}

// ToleranceBand defines acceptable deviation from target values
type ToleranceBand struct {
    ProteinPct float64 // percentage tolerance for protein (e.g., 0.10 = ±10%)
    CarbsPct   float64 // percentage tolerance for carbohydrates
    FatPct     float64 // percentage tolerance for fat
}

// DefaultToleranceBand provides standard tolerance values
var DefaultToleranceBand = ToleranceBand{
    ProteinPct: 0.10, // ±10%
    CarbsPct:   0.15, // ±15%
    FatPct:     0.15, // ±15%
}

// MealCandidate represents a meal available for selection in the LP
type MealCandidate struct {
    ID            string
    Protein       float64 // grams per serving
    Carbs         float64 // grams per serving
    Fat           float64 // grams per serving
    Calories      float64 // kcal per serving
    MaxServings   float64 // maximum servings allowed (default 5.0)
    IsOriginal    bool    // true if this meal was in the original diet
}

// Constraint represents a single linear constraint for the LP solver
type Constraint struct {
    Coefficients []float64       // one coefficient per decision variable
    Sense        ConstraintSense // LE, GE, or EQ
    RHS          float64         // right-hand side value
    Name         string          // human-readable constraint name
}

// ConstraintSense indicates the type of inequality
type ConstraintSense int

const (
    ConstraintLE ConstraintSense = iota // <=
    ConstraintGE                        // >=
    ConstraintEQ                        // ==
)

// ConstraintSet holds all constraints for a single LP problem
type ConstraintSet struct {
    MacroConstraints     []Constraint // protein, carbs, fat bounds
    ServingConstraints   []Constraint // min/max servings per meal
    ExclusionConstraints []Constraint // meals to exclude from solution
    VariableNames        []string     // decision variable names (meal IDs)
    VariableCount        int          // number of decision variables
}

// ExclusionRule specifies meals that must not appear in the solution
type ExclusionRule struct {
    MealIDs        []string // specific meal IDs to exclude
    ExcludeOriginal bool    // exclude all meals marked as IsOriginal
}

// ConstraintBuilderError indicates why constraint building failed
type ConstraintBuilderError struct {
    Code    string
    Message string
}

// Error codes
const (
    ErrCodeNoCandidates       = "NO_CANDIDATES"        // Empty candidate list
    ErrCodeInvalidTarget      = "INVALID_TARGET"       // Negative target values
    ErrCodeInvalidTolerance   = "INVALID_TOLERANCE"    // Tolerance outside [0, 1]
    ErrCodeInfeasibleBounds   = "INFEASIBLE_BOUNDS"    // Upper bound < lower bound
    ErrCodeAllExcluded        = "ALL_EXCLUDED"         // All candidates excluded
)
```

### 2. Logic & Algorithms (Step-by-Step)

#### 2.1 BuildConstraintSet - Main Entry Point

**Purpose:** Construct the complete set of LP constraints from target macros, candidates, and exclusion rules.

**Algorithm:**

1. **Validate Inputs**
   - Check candidates list is non-empty → error `ErrCodeNoCandidates`
   - Check target macros are non-negative → error `ErrCodeInvalidTarget`
   - Check tolerance percentages are in [0, 1] → error `ErrCodeInvalidTolerance`

2. **Filter Excluded Candidates**
   - Remove candidates matching `ExclusionRule.MealIDs`
   - If `ExclusionRule.ExcludeOriginal` is true, remove candidates where `IsOriginal == true`
   - If all candidates filtered out → error `ErrCodeAllExcluded`

3. **Build Variable Mapping**
   ```
   variableNames = []
   for each candidate in filteredCandidates:
       variableNames.append(candidate.ID)
   variableCount = len(variableNames)
   ```

4. **Build Macro Constraints**
   - Call `buildMacroConstraints(target, tolerance, filteredCandidates)`
   - Results in 6 constraints (lower and upper bound for each macro)

5. **Build Serving Constraints**
   - Call `buildServingConstraints(filteredCandidates)`
   - Results in 2 constraints per candidate (min 0, max MaxServings)

6. **Assemble ConstraintSet**
   - Combine all constraint groups into ConstraintSet
   - Return the complete set

#### 2.2 buildMacroConstraints - Macronutrient Bounds

**Purpose:** Create lower and upper bound constraints for each macronutrient.

**Algorithm:**

For each macro type (Protein, Carbs, Fat):

1. **Calculate Bounds**
   ```
   lowerBound = target * (1 - tolerancePct)
   upperBound = target * (1 + tolerancePct)
   ```

2. **Validate Bounds**
   - If upperBound < lowerBound → error `ErrCodeInfeasibleBounds`

3. **Build Coefficient Vector**
   - For lower bound constraint:
     ```
     coefficients = []
     for each candidate:
         coefficients.append(candidate.MacroValue)
     constraint = Constraint{
         Coefficients: coefficients,
         Sense: ConstraintGE,
         RHS: lowerBound,
         Name: "{MacroName}_MIN"
     }
     ```
   - For upper bound constraint:
     ```
     constraint = Constraint{
         Coefficients: coefficients,
         Sense: ConstraintLE,
         RHS: upperBound,
         Name: "{MacroName}_MAX"
     }
     ```

4. **Return Constraints**
   - Return array of 6 constraints: [PROTEIN_MIN, PROTEIN_MAX, CARBS_MIN, CARBS_MAX, FAT_MIN, FAT_MAX]

**Example:**
```
Target: Protein=100g, Tolerance=10%
Candidates: [MealA(25g protein), MealB(30g protein), MealC(20g protein)]

Lower Bound: 100 * 0.90 = 90g
Upper Bound: 100 * 1.10 = 110g

PROTEIN_MIN constraint:
  25*x_A + 30*x_B + 20*x_C >= 90

PROTEIN_MAX constraint:
  25*x_A + 30*x_B + 20*x_C <= 110
```

#### 2.3 buildServingConstraints - Variable Bounds

**Purpose:** Constrain each decision variable to valid serving ranges.

**Algorithm:**

For each candidate at index i:

1. **Build Lower Bound (Non-negativity)**
   ```
   coefficients = [0, 0, ..., 1 at index i, ..., 0]
   constraint = Constraint{
       Coefficients: coefficients,
       Sense: ConstraintGE,
       RHS: 0.0,
       Name: "{MealID}_MIN_SERVING"
   }
   ```

2. **Build Upper Bound (Max Servings)**
   ```
   constraint = Constraint{
       Coefficients: coefficients,
       Sense: ConstraintLE,
       RHS: candidate.MaxServings,
       Name: "{MealID}_MAX_SERVING"
   }
   ```

**Note:** In many LP solvers, variable bounds can be specified directly without explicit constraints. The go-coinor/clp wrapper may support `SetColumnBounds()`. If so, these can be set as bounds rather than constraints for better performance.

#### 2.4 BuildExclusionConstraint - Single Meal Exclusion

**Purpose:** Create a constraint that forces a specific meal's serving count to zero.

**Algorithm:**

1. **Find Variable Index**
   ```
   index = indexOf(mealID, variableNames)
   if index == -1:
       return nil (meal not in candidate set)
   ```

2. **Build Zero-Forcing Constraint**
   ```
   coefficients = [0, 0, ..., 1 at index, ..., 0]
   constraint = Constraint{
       Coefficients: coefficients,
       Sense: ConstraintEQ,
       RHS: 0.0,
       Name: "EXCLUDE_{MealID}"
   }
   ```

3. **Return Constraint**

**Usage:** Used for multi-solution generation. After finding solution 1, add exclusion constraints for meals in solution 1, then solve again to find solution 2.

#### 2.5 AddDiversityPenaltyConstraints - Original Diet Penalty

**Purpose:** Add constraints that encourage selecting different meals from the original diet.

**Algorithm:**

This function does NOT add constraints directly. Instead, it modifies the objective function coefficients (handled by ObjectiveFunction component). However, if hard constraints are needed:

1. **Limit Original Meal Servings**
   ```
   for each candidate where IsOriginal == true:
       // Reduce max servings for original meals
       reducedMax = min(candidate.MaxServings, 1.0)

       coefficients = [0, 0, ..., 1 at index, ..., 0]
       constraint = Constraint{
           Coefficients: coefficients,
           Sense: ConstraintLE,
           RHS: reducedMax,
           Name: "DIVERSITY_{MealID}"
       }
   ```

**Note:** The primary diversity mechanism is via penalty weights in the objective function (see ObjectiveFunction design). Hard constraints are optional.

### 3. State Management & Error Handling

#### 3.1 Error States

| Error Condition | Error Code | Cause | Handling |
|----------------|------------|-------|----------|
| Empty candidate list | `NO_CANDIDATES` | No meals available for optimization | Return error to caller; job marked as failed |
| Negative target values | `INVALID_TARGET` | Invalid input from API | Return 400 Bad Request; reject job |
| Tolerance out of range | `INVALID_TOLERANCE` | Tolerance < 0 or > 1 | Return 400 Bad Request; use default tolerance |
| Infeasible bounds | `INFEASIBLE_BOUNDS` | Upper bound < lower bound (should not occur with valid tolerance) | Log error; use wider tolerance |
| All candidates excluded | `ALL_EXCLUDED` | Exclusion rules filtered all meals | Return error; suggest fewer exclusions |

#### 3.2 State Transitions

The ConstraintBuilder is stateless. Each call to `BuildConstraintSet` is independent:

```
Inputs → Validation → Filtering → Constraint Generation → Output/Error
```

No internal state is maintained between calls.

#### 3.3 Error Propagation

- Errors are returned as structured `ConstraintBuilderError` values
- Callers (LPSolverWrapper) must handle errors appropriately
- Validation errors should be caught early and returned to the API layer
- Infeasible constraint sets will cause LP solver to report "no solution"

#### 3.4 LP Infeasibility vs Constraint Errors

| Scenario | Detection Point | Response |
|----------|-----------------|----------|
| Invalid input (negative values) | ConstraintBuilder validation | `ConstraintBuilderError` returned |
| Empty after exclusions | ConstraintBuilder filtering | `ConstraintBuilderError` returned |
| Mathematically infeasible | LP solver returns infeasible status | JobStatusTracker marks job as failed with "No feasible solution" |
| Solver timeout | go-coinor/clp timeout | JobStatusTracker marks job as failed with "Optimization timeout" |

### 4. Component Interfaces

```go
// ConstraintBuilder constructs LP constraints from optimization parameters
type ConstraintBuilder interface {
    // BuildConstraintSet creates the complete constraint set for an LP problem
    // Returns error if inputs are invalid or all candidates are excluded
    BuildConstraintSet(
        target MacroTarget,
        tolerance ToleranceBand,
        candidates []MealCandidate,
        exclusion ExclusionRule,
    ) (*ConstraintSet, error)

    // BuildExclusionConstraint creates a constraint excluding a specific meal
    // Returns nil if mealID is not in the variable list
    BuildExclusionConstraint(
        mealID string,
        variableNames []string,
    ) *Constraint

    // ValidateTarget checks if target macros are valid
    ValidateTarget(target MacroTarget) error

    // ValidateTolerance checks if tolerance values are in valid range
    ValidateTolerance(tolerance ToleranceBand) error
}

// Error implements the error interface for ConstraintBuilderError
func (e *ConstraintBuilderError) Error() string
```

#### 4.1 Function Signatures

```go
// NewConstraintBuilder creates a new ConstraintBuilder instance
func NewConstraintBuilder() ConstraintBuilder

// BuildConstraintSet constructs all constraints for the LP problem
// Parameters:
//   - target: desired macronutrient values
//   - tolerance: acceptable deviation from targets
//   - candidates: meals available for selection
//   - exclusion: rules for excluding certain meals
// Returns:
//   - *ConstraintSet: complete set of constraints and variable info
//   - error: ConstraintBuilderError if validation fails
func (b *constraintBuilder) BuildConstraintSet(
    target MacroTarget,
    tolerance ToleranceBand,
    candidates []MealCandidate,
    exclusion ExclusionRule,
) (*ConstraintSet, error)

// BuildExclusionConstraint creates a single exclusion constraint
// Parameters:
//   - mealID: ID of meal to exclude from solution
//   - variableNames: ordered list of variable names (meal IDs)
// Returns:
//   - *Constraint: exclusion constraint, or nil if mealID not found
func (b *constraintBuilder) BuildExclusionConstraint(
    mealID string,
    variableNames []string,
) *Constraint

// buildMacroConstraints creates the 6 macro bound constraints
// Parameters:
//   - target: desired macronutrient values
//   - tolerance: acceptable deviation percentages
//   - candidates: filtered meal candidates
// Returns:
//   - []Constraint: array of 6 constraints (2 per macro)
//   - error: if bounds are infeasible
func (b *constraintBuilder) buildMacroConstraints(
    target MacroTarget,
    tolerance ToleranceBand,
    candidates []MealCandidate,
) ([]Constraint, error)

// buildServingConstraints creates variable bound constraints
// Parameters:
//   - candidates: filtered meal candidates
// Returns:
//   - []Constraint: array of 2*n constraints (min and max per variable)
func (b *constraintBuilder) buildServingConstraints(
    candidates []MealCandidate,
) []Constraint

// filterCandidates removes excluded meals from candidate list
// Parameters:
//   - candidates: original meal candidates
//   - exclusion: exclusion rules to apply
// Returns:
//   - []MealCandidate: filtered candidate list
func (b *constraintBuilder) filterCandidates(
    candidates []MealCandidate,
    exclusion ExclusionRule,
) []MealCandidate

// ValidateTarget checks target values for validity
// Parameters:
//   - target: target macro values
// Returns:
//   - error: ConstraintBuilderError if any value is negative or NaN/Inf
func (b *constraintBuilder) ValidateTarget(target MacroTarget) error

// ValidateTolerance checks tolerance values for validity
// Parameters:
//   - tolerance: tolerance percentages
// Returns:
//   - error: ConstraintBuilderError if any value is outside [0, 1]
func (b *constraintBuilder) ValidateTolerance(tolerance ToleranceBand) error
```

#### 4.2 Usage Example (Pseudocode)

```go
// Create builder
builder := NewConstraintBuilder()

// Define optimization parameters
target := MacroTarget{Protein: 150.0, Carbs: 200.0, Fat: 70.0}
tolerance := DefaultToleranceBand

// Available meals
candidates := []MealCandidate{
    {ID: "meal_001", Protein: 30.0, Carbs: 45.0, Fat: 12.0, Calories: 408, MaxServings: 3.0, IsOriginal: false},
    {ID: "meal_002", Protein: 25.0, Carbs: 60.0, Fat: 8.0, Calories: 412, MaxServings: 4.0, IsOriginal: true},
    {ID: "meal_003", Protein: 40.0, Carbs: 20.0, Fat: 15.0, Calories: 375, MaxServings: 3.0, IsOriginal: false},
}

// Exclusion rules (exclude original meals from second solution)
exclusion := ExclusionRule{ExcludeOriginal: false}

// Build constraints
constraintSet, err := builder.BuildConstraintSet(target, tolerance, candidates, exclusion)
if err != nil {
    // Handle error
}

// Pass to LP solver
solver := NewLPSolverWrapper()
solution, err := solver.Solve(constraintSet, objectiveCoeffs)

// For second solution, add exclusion for meals in first solution
for _, mealID := range solution.SelectedMealIDs {
    excludeConstraint := builder.BuildExclusionConstraint(mealID, constraintSet.VariableNames)
    if excludeConstraint != nil {
        constraintSet.ExclusionConstraints = append(constraintSet.ExclusionConstraints, *excludeConstraint)
    }
}

// Solve again for alternative
solution2, err := solver.Solve(constraintSet, objectiveCoeffs)
```

#### 4.3 Integration with go-coinor/clp

The ConstraintSet structure maps to CLP solver primitives:

```go
// Mapping to CLP API (conceptual)
func (cs *ConstraintSet) ToCLPModel(model *clp.Model) {
    // Add variables
    for i, name := range cs.VariableNames {
        model.AddColumn(name, 0.0, cs.ServingConstraints[i*2+1].RHS) // bounds from serving constraints
    }

    // Add macro constraints
    for _, c := range cs.MacroConstraints {
        switch c.Sense {
        case ConstraintLE:
            model.AddRow(c.Coefficients, -clp.Infinity, c.RHS)
        case ConstraintGE:
            model.AddRow(c.Coefficients, c.RHS, clp.Infinity)
        case ConstraintEQ:
            model.AddRow(c.Coefficients, c.RHS, c.RHS)
        }
    }

    // Add exclusion constraints
    for _, c := range cs.ExclusionConstraints {
        model.AddRow(c.Coefficients, c.RHS, c.RHS) // equality constraint
    }
}
```
