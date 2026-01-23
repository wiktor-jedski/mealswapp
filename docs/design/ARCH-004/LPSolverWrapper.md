## FILE: LPSolverWrapper.md
**Traceability:** ARCH-004

### 1. Data Structures & Types

```go
// LPVariable represents a decision variable in the LP problem
// Each variable corresponds to a quantity (in 100g units) of a candidate meal
type LPVariable struct {
    Index      int     // Column index in the LP matrix
    MealID     string  // Reference to the meal in ARCH-005
    LowerBound float64 // Minimum quantity (0.0 = optional)
    UpperBound float64 // Maximum quantity (e.g., 10.0 = 1000g max)
}

// LPConstraint represents a linear constraint in the problem
// Form: LowerBound <= sum(Coefficients[i] * Variables[i]) <= UpperBound
type LPConstraint struct {
    Name         string             // Human-readable constraint identifier
    Type         ConstraintType     // Equality, LessOrEqual, GreaterOrEqual, Range
    LowerBound   float64            // Left-hand side bound
    UpperBound   float64            // Right-hand side bound
    Coefficients map[int]float64    // Variable index -> coefficient
}

// ConstraintType defines the type of linear constraint
type ConstraintType string

const (
    ConstraintEquality       ConstraintType = "eq"    // LHS = RHS
    ConstraintLessOrEqual    ConstraintType = "le"    // LHS <= RHS
    ConstraintGreaterOrEqual ConstraintType = "ge"    // LHS >= RHS
    ConstraintRange          ConstraintType = "range" // LB <= expr <= UB
)

// ObjectiveFunction represents the objective to minimize
// Form: minimize sum(Coefficients[i] * Variables[i])
type ObjectiveFunction struct {
    Sense        ObjectiveSense     // Minimize or Maximize
    Coefficients map[int]float64    // Variable index -> coefficient (calorie per 100g)
}

// ObjectiveSense defines optimization direction
type ObjectiveSense string

const (
    ObjectiveMinimize ObjectiveSense = "minimize"
    ObjectiveMaximize ObjectiveSense = "maximize"
)

// LPProblem encapsulates the complete LP formulation
type LPProblem struct {
    Variables   []LPVariable
    Constraints []LPConstraint
    Objective   ObjectiveFunction
    variableMap map[string]int // MealID -> variable index lookup
}

// LPSolution represents the result of solving the LP problem
type LPSolution struct {
    Status       SolutionStatus
    ObjectiveVal float64            // Minimized calorie count
    Variables    map[string]float64 // MealID -> quantity in 100g units
    DualValues   map[string]float64 // Constraint name -> shadow price (for diagnostics)
}

// SolutionStatus indicates the outcome of the LP solve
type SolutionStatus string

const (
    SolutionOptimal      SolutionStatus = "optimal"
    SolutionInfeasible   SolutionStatus = "infeasible"
    SolutionUnbounded    SolutionStatus = "unbounded"
    SolutionTimeout      SolutionStatus = "timeout"
    SolutionError        SolutionStatus = "error"
)

// MacroTarget defines the target macronutrient profile with tolerances
type MacroTarget struct {
    ProteinGrams float64 // Target protein in grams
    CarbsGrams   float64 // Target carbohydrates in grams
    FatGrams     float64 // Target fat in grams
    Tolerance    float64 // Allowed deviation percentage (e.g., 0.10 = +/-10%)
}

// MealCandidate represents a meal available for selection
type MealCandidate struct {
    MealID          string
    ProteinPer100g  float64
    CarbsPer100g    float64
    FatPer100g      float64
    CaloriesPer100g float64
    DiversityWeight float64 // Penalty applied if in original diet (0.0 = neutral)
}

// SolverConfig holds configuration parameters for the LP solver
type SolverConfig struct {
    MaxIterations     int           // CLP iteration limit (default: 10000)
    TimeoutSeconds    float64       // Maximum solve time (default: 25.0)
    PrimalTolerance   float64       // Feasibility tolerance (default: 1e-7)
    DualTolerance     float64       // Optimality tolerance (default: 1e-7)
    PresolveEnabled   bool          // Enable CLP presolve (default: true)
    ScalingEnabled    bool          // Enable automatic scaling (default: true)
}

// DefaultSolverConfig returns production-ready solver settings
func DefaultSolverConfig() SolverConfig {
    return SolverConfig{
        MaxIterations:   10000,
        TimeoutSeconds:  25.0,
        PrimalTolerance: 1e-7,
        DualTolerance:   1e-7,
        PresolveEnabled: true,
        ScalingEnabled:  true,
    }
}
```

### 2. Logic & Algorithms (Step-by-Step)

#### 2.1 Wrapper Initialization (`NewLPSolverWrapper`)

```
FUNCTION NewLPSolverWrapper(config SolverConfig) *LPSolverWrapper:
    1. Validate config parameters:
       - IF config.MaxIterations <= 0: set to 10000
       - IF config.TimeoutSeconds <= 0: set to 25.0
       - IF config.PrimalTolerance <= 0: set to 1e-7
       - IF config.DualTolerance <= 0: set to 1e-7
    2. Create LPSolverWrapper struct with config
    3. Return wrapper instance
```

#### 2.2 Problem Construction (`BuildProblem`)

```
FUNCTION BuildProblem(candidates []MealCandidate, target MacroTarget, excludedIDs []string) (*LPProblem, error):
    1. Validate inputs:
       - IF len(candidates) == 0: return error "no candidate meals provided"
       - IF target.ProteinGrams < 0 OR target.CarbsGrams < 0 OR target.FatGrams < 0:
           return error "invalid macro targets"
       - IF target.Tolerance < 0 OR target.Tolerance > 1:
           return error "tolerance must be between 0 and 1"

    2. Initialize LPProblem:
       - problem.Variables = empty slice
       - problem.Constraints = empty slice
       - problem.variableMap = empty map

    3. Create decision variables (Section 2.3)

    4. Build macronutrient constraints (Section 2.4)

    5. Build objective function (Section 2.5)

    6. Apply diversity penalties for excluded meals (Section 2.6)

    7. Return problem
```

#### 2.3 Variable Creation (`createVariables`)

```
FUNCTION createVariables(problem *LPProblem, candidates []MealCandidate):
    FOR i, candidate in candidates:
        1. Create LPVariable:
           - Index = i
           - MealID = candidate.MealID
           - LowerBound = 0.0 (meal is optional)
           - UpperBound = 10.0 (max 1000g per meal)

        2. Append variable to problem.Variables

        3. Add to lookup map: problem.variableMap[candidate.MealID] = i
```

#### 2.4 Constraint Building (`buildMacroConstraints`)

```
FUNCTION buildMacroConstraints(problem *LPProblem, candidates []MealCandidate, target MacroTarget):
    // Calculate tolerance bounds
    toleranceFactor = target.Tolerance  // e.g., 0.10 for 10%

    // Protein constraint: target * (1 - tolerance) <= sum(protein_i * x_i) <= target * (1 + tolerance)
    1. proteinLower = target.ProteinGrams * (1.0 - toleranceFactor)
    2. proteinUpper = target.ProteinGrams * (1.0 + toleranceFactor)
    3. proteinCoeffs = empty map
    4. FOR i, candidate in candidates:
           proteinCoeffs[i] = candidate.ProteinPer100g
    5. Create constraint:
       - Name = "protein_target"
       - Type = ConstraintRange
       - LowerBound = proteinLower
       - UpperBound = proteinUpper
       - Coefficients = proteinCoeffs
    6. Append to problem.Constraints

    // Carbohydrate constraint
    7. carbsLower = target.CarbsGrams * (1.0 - toleranceFactor)
    8. carbsUpper = target.CarbsGrams * (1.0 + toleranceFactor)
    9. carbsCoeffs = empty map
    10. FOR i, candidate in candidates:
            carbsCoeffs[i] = candidate.CarbsPer100g
    11. Create constraint:
        - Name = "carbs_target"
        - Type = ConstraintRange
        - LowerBound = carbsLower
        - UpperBound = carbsUpper
        - Coefficients = carbsCoeffs
    12. Append to problem.Constraints

    // Fat constraint
    13. fatLower = target.FatGrams * (1.0 - toleranceFactor)
    14. fatUpper = target.FatGrams * (1.0 + toleranceFactor)
    15. fatCoeffs = empty map
    16. FOR i, candidate in candidates:
            fatCoeffs[i] = candidate.FatPer100g
    17. Create constraint:
        - Name = "fat_target"
        - Type = ConstraintRange
        - LowerBound = fatLower
        - UpperBound = fatUpper
        - Coefficients = fatCoeffs
    18. Append to problem.Constraints
```

#### 2.5 Objective Function (`buildObjectiveFunction`)

```
FUNCTION buildObjectiveFunction(problem *LPProblem, candidates []MealCandidate):
    1. objectiveCoeffs = empty map

    2. FOR i, candidate in candidates:
           // Coefficient = calories per 100g (to minimize total calories)
           objectiveCoeffs[i] = candidate.CaloriesPer100g

    3. Set problem.Objective:
       - Sense = ObjectiveMinimize
       - Coefficients = objectiveCoeffs
```

#### 2.6 Diversity Penalty Application (`applyDiversityPenalties`)

```
FUNCTION applyDiversityPenalties(problem *LPProblem, candidates []MealCandidate, excludedIDs []string):
    CONST DIVERSITY_PENALTY = 1000.0  // Large penalty to discourage original meals

    1. Create excludedSet from excludedIDs for O(1) lookup

    2. FOR i, candidate in candidates:
           IF candidate.MealID in excludedSet:
               // Add penalty to objective coefficient
               problem.Objective.Coefficients[i] += DIVERSITY_PENALTY
               // Note: This makes selecting excluded meals very expensive calorically
```

#### 2.7 LP Solve (`Solve`)

```
FUNCTION Solve(problem *LPProblem, config SolverConfig) (*LPSolution, error):
    1. Initialize CLP simplex model via go-coinor/clp:
       - model = clp.NewSimplex()

    2. Configure solver parameters:
       - model.SetMaximumIterations(config.MaxIterations)
       - model.SetMaximumSeconds(config.TimeoutSeconds)
       - model.SetPrimalTolerance(config.PrimalTolerance)
       - model.SetDualTolerance(config.DualTolerance)
       - IF config.PresolveEnabled: model.SetPresolveType(clp.PresolveOn)
       - IF config.ScalingEnabled: model.SetScaling(clp.ScalingAuto)

    3. Load variables into CLP:
       - numCols = len(problem.Variables)
       - FOR each variable in problem.Variables:
           - Set column bounds: model.SetColumnBounds(idx, var.LowerBound, var.UpperBound)

    4. Load objective function:
       - FOR idx, coeff in problem.Objective.Coefficients:
           - model.SetObjectiveCoefficient(idx, coeff)
       - IF problem.Objective.Sense == ObjectiveMinimize:
           - model.SetOptimizationDirection(clp.Minimize)

    5. Load constraints into CLP:
       - FOR each constraint in problem.Constraints:
           - rowIndices, rowValues = extract non-zero coefficients
           - Add row to model with bounds based on constraint.Type

    6. Execute solve:
       - status = model.Primal()  // Use primal simplex method

    7. Map CLP status to SolutionStatus:
       - IF status == 0: solStatus = SolutionOptimal
       - ELSE IF status == 1: solStatus = SolutionInfeasible
       - ELSE IF status == 2: solStatus = SolutionUnbounded
       - ELSE IF status == 3: solStatus = SolutionTimeout
       - ELSE: solStatus = SolutionError

    8. Extract solution values (Section 2.8)

    9. Clean up: model.Delete()

    10. Return solution
```

#### 2.8 Solution Extraction (`extractSolution`)

```
FUNCTION extractSolution(model *clp.Simplex, problem *LPProblem, status SolutionStatus) *LPSolution:
    1. Create solution struct:
       - solution.Status = status
       - solution.Variables = empty map
       - solution.DualValues = empty map

    2. IF status != SolutionOptimal:
           return solution (empty variable values)

    3. Extract objective value:
       - solution.ObjectiveVal = model.ObjectiveValue()

    4. Extract primal variable values:
       - primalValues = model.PrimalColumnSolution()
       - FOR each variable in problem.Variables:
           - value = primalValues[variable.Index]
           - IF value > 1e-6:  // Only include non-zero quantities
               solution.Variables[variable.MealID] = value

    5. Extract dual values (shadow prices) for diagnostics:
       - dualValues = model.DualRowSolution()
       - FOR i, constraint in problem.Constraints:
           solution.DualValues[constraint.Name] = dualValues[i]

    6. Return solution
```

#### 2.9 Add Exclusion Constraint (`AddExclusionConstraint`)

```
FUNCTION AddExclusionConstraint(problem *LPProblem, mealIDs []string) error:
    // Used for multi-solution generation: exclude meals from previous solutions
    1. FOR each mealID in mealIDs:
           idx, exists = problem.variableMap[mealID]
           IF NOT exists:
               continue  // Meal not in candidate set, skip

           // Force variable to zero: x_i = 0
           2. Create constraint:
              - Name = "exclude_" + mealID
              - Type = ConstraintEquality
              - LowerBound = 0.0
              - UpperBound = 0.0
              - Coefficients = {idx: 1.0}

           3. Append to problem.Constraints

    4. Return nil
```

#### 2.10 Validate Solution Feasibility (`ValidateSolution`)

```
FUNCTION ValidateSolution(solution *LPSolution, target MacroTarget, candidates []MealCandidate) ValidationResult:
    1. IF solution.Status != SolutionOptimal:
           return ValidationResult{Valid: false, Reason: "non-optimal status"}

    2. Calculate actual macros from solution:
       - totalProtein = 0.0
       - totalCarbs = 0.0
       - totalFat = 0.0
       - totalCalories = 0.0

       - FOR mealID, quantity in solution.Variables:
           - Find candidate by mealID
           - totalProtein += candidate.ProteinPer100g * quantity
           - totalCarbs += candidate.CarbsPer100g * quantity
           - totalFat += candidate.FatPer100g * quantity
           - totalCalories += candidate.CaloriesPer100g * quantity

    3. Validate against target tolerances:
       - proteinLower = target.ProteinGrams * (1.0 - target.Tolerance)
       - proteinUpper = target.ProteinGrams * (1.0 + target.Tolerance)
       - IF totalProtein < proteinLower OR totalProtein > proteinUpper:
           return ValidationResult{Valid: false, Reason: "protein out of range"}

       - (Repeat for carbs and fat)

    4. Return ValidationResult{
           Valid: true,
           ActualProtein: totalProtein,
           ActualCarbs: totalCarbs,
           ActualFat: totalFat,
           TotalCalories: totalCalories
       }
```

### 3. State Management & Error Handling

#### 3.1 Error States

| Error State | Trigger | Recovery Action | Caller Response |
|:------------|:--------|:----------------|:----------------|
| No Candidates | `len(candidates) == 0` | Return immediately with error | Job marked failed, user notified |
| Invalid Macro Target | Negative protein/carbs/fat values | Return validation error | Job marked failed, request rejected |
| Invalid Tolerance | Tolerance < 0 or > 1 | Return validation error | Job marked failed, request rejected |
| CLP Initialization Failed | Memory allocation failure | Return error with CLP message | Job marked failed, retry possible |
| Problem Infeasible | No combination meets constraints | Return SolutionInfeasible | Return partial results or "no alternatives found" |
| Problem Unbounded | Constraints allow infinite solution | Return SolutionUnbounded | Log error, investigate constraint setup |
| Solver Timeout | Exceeds TimeoutSeconds (25s) | Return SolutionTimeout with best found | Return partial solution if available |
| CLP Internal Error | Numerical instability | Return SolutionError | Job marked failed, log for investigation |
| Memory Exhaustion | Large problem size | CLP returns error | Limit candidate count, reduce problem size |

#### 3.2 Solver State Machine

```
                    ┌─────────────┐
                    │    IDLE     │
                    └──────┬──────┘
                           │ BuildProblem called
                           ▼
                    ┌─────────────┐
                    │  BUILDING   │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │ Success    │            │ Validation Error
              ▼            │            ▼
       ┌─────────────┐     │     ┌─────────────┐
       │   READY     │     │     │   ERROR     │───► Return error
       └──────┬──────┘     │     └─────────────┘
              │ Solve called
              ▼
       ┌─────────────┐
       │  LOADING    │ (Transfer to CLP)
       └──────┬──────┘
              │
              ▼
       ┌─────────────┐
       │  SOLVING    │
       └──────┬──────┘
              │
    ┌─────────┼─────────┬─────────┬─────────┐
    │         │         │         │         │
    ▼         ▼         ▼         ▼         ▼
┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐
│OPTIMAL│ │INFEAS.│ │UNBND. │ │TIMEOUT│ │ ERROR │
└───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘
    │         │         │         │         │
    └─────────┴─────────┴─────────┴─────────┘
                        │
                        ▼
                 ┌─────────────┐
                 │ EXTRACTING  │
                 └──────┬──────┘
                        │
                        ▼
                 ┌─────────────┐
                 │  CLEANUP    │ (Free CLP resources)
                 └──────┬──────┘
                        │
                        ▼
                 ┌─────────────┐
                 │  COMPLETE   │───► Return solution
                 └─────────────┘
```

#### 3.3 Resource Management

| Resource | Acquisition | Release | Failure Handling |
|:---------|:------------|:--------|:-----------------|
| CLP Simplex Model | `clp.NewSimplex()` | `model.Delete()` in defer | Panic recovery with cleanup |
| Row/Column Arrays | Go slice allocation | Garbage collected | Monitor allocation size |
| Coefficient Maps | Go map allocation | Garbage collected | Pre-allocate capacity |

#### 3.4 Timeout Handling

```
FUNCTION SolveWithTimeout(problem *LPProblem, config SolverConfig) (*LPSolution, error):
    1. Set CLP internal timeout: config.TimeoutSeconds

    2. Create context with deadline:
       - ctx, cancel = context.WithTimeout(parentCtx, config.TimeoutSeconds * time.Second)
       - defer cancel()

    3. Run solve in goroutine:
       - resultChan = make(chan *LPSolution, 1)
       - errChan = make(chan error, 1)
       - go func():
             solution, err = Solve(problem, config)
             IF err != nil:
                 errChan <- err
             ELSE:
                 resultChan <- solution

    4. Select on channels:
       - CASE solution = <-resultChan:
             return solution, nil
       - CASE err = <-errChan:
             return nil, err
       - CASE <-ctx.Done():
             // CLP should have returned due to internal timeout
             // If not, return timeout status
             return &LPSolution{Status: SolutionTimeout}, nil
```

### 4. Component Interfaces

#### 4.1 Main Wrapper Interface

```go
// LPSolverWrapper wraps go-coinor/clp for diet optimization
type LPSolverWrapper struct {
    config SolverConfig
    logger *slog.Logger
}

// NewLPSolverWrapper creates a new wrapper with the given configuration
func NewLPSolverWrapper(config SolverConfig, logger *slog.Logger) *LPSolverWrapper

// BuildProblem constructs an LP problem from candidates and targets
// Returns a reusable LPProblem that can be modified for multi-solution generation
func (w *LPSolverWrapper) BuildProblem(
    candidates []MealCandidate,
    target MacroTarget,
    excludedIDs []string,
) (*LPProblem, error)

// Solve executes the LP solver on the given problem
// Returns the solution or an error if the solver fails catastrophically
func (w *LPSolverWrapper) Solve(problem *LPProblem) (*LPSolution, error)

// SolveWithContext executes with context for cancellation support
func (w *LPSolverWrapper) SolveWithContext(
    ctx context.Context,
    problem *LPProblem,
) (*LPSolution, error)
```

#### 4.2 Problem Manipulation Functions

```go
// AddExclusionConstraint adds constraints to exclude specific meals
// Used for generating multiple distinct solutions
func (p *LPProblem) AddExclusionConstraint(mealIDs []string) error

// AddMinimumMealConstraint ensures at least N different meals are selected
// constraint: sum(binary indicators) >= minMeals
func (p *LPProblem) AddMinimumMealConstraint(minMeals int) error

// Clone creates a deep copy of the problem for modification
// Used when generating alternative solutions without modifying original
func (p *LPProblem) Clone() *LPProblem

// GetVariableIndex returns the column index for a meal ID
// Returns -1 if meal not found
func (p *LPProblem) GetVariableIndex(mealID string) int
```

#### 4.3 Solution Utilities

```go
// ValidationResult contains the outcome of solution validation
type ValidationResult struct {
    Valid         bool
    Reason        string  // Empty if valid
    ActualProtein float64
    ActualCarbs   float64
    ActualFat     float64
    TotalCalories float64
}

// ValidateSolution checks if solution meets constraints
func ValidateSolution(
    solution *LPSolution,
    target MacroTarget,
    candidates []MealCandidate,
) ValidationResult

// ToMealQuantities converts solution to user-friendly format
// Converts 100g units to actual gram quantities
func (s *LPSolution) ToMealQuantities() map[string]float64

// FilterNonZero returns only meals with quantity > threshold
func (s *LPSolution) FilterNonZero(threshold float64) map[string]float64
```

#### 4.4 Dependency Interfaces (Required from other ARCH components)

```go
// From ARCH-005 (Data Repository)
type MealRepository interface {
    // GetMealsWithMacros retrieves meals with their macronutrient data
    GetMealsWithMacros(ctx context.Context, mealIDs []string) ([]MealWithMacros, error)

    // GetAllCandidateMeals retrieves all meals available for optimization
    // Excludes meals in excludeIDs
    GetAllCandidateMeals(ctx context.Context, excludeIDs []string) ([]MealCandidate, error)
}

// MealWithMacros contains meal data needed for LP optimization
type MealWithMacros struct {
    ID              string
    Name            string
    ProteinPer100g  float64
    CarbsPer100g    float64
    FatPer100g      float64
    CaloriesPer100g float64
}
```

#### 4.5 CLP Library Interface (go-coinor/clp)

```go
// Expected interface from go-coinor/clp library
// This documents the subset of CLP functionality used by LPSolverWrapper

type Simplex interface {
    // Configuration
    SetMaximumIterations(n int)
    SetMaximumSeconds(seconds float64)
    SetPrimalTolerance(tolerance float64)
    SetDualTolerance(tolerance float64)
    SetPresolveType(presolve int)
    SetScaling(scaling int)
    SetOptimizationDirection(direction int)

    // Problem construction
    Resize(numRows, numCols int)
    SetColumnBounds(col int, lower, upper float64)
    SetObjectiveCoefficient(col int, value float64)
    AddRow(numElements int, indices []int, values []float64, lower, upper float64)

    // Solving
    Primal() int  // Returns status code
    Dual() int    // Alternative: dual simplex

    // Solution extraction
    ObjectiveValue() float64
    PrimalColumnSolution() []float64
    DualRowSolution() []float64
    Status() int

    // Cleanup
    Delete()
}

// CLP status codes
const (
    CLPOptimal      = 0
    CLPInfeasible   = 1
    CLPUnbounded    = 2
    CLPIterLimit    = 3
    CLPError        = 4
)

// CLP presolve options
const (
    PresolveOff = 0
    PresolveOn  = 1
)

// CLP scaling options
const (
    ScalingOff  = 0
    ScalingAuto = 3
)

// CLP optimization direction
const (
    Minimize = 1
    Maximize = -1
)
```
