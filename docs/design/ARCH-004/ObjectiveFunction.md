## FILE: ObjectiveFunction.md
**Traceability:** ARCH-004

### 1. Data Structures & Types

```go
// MealVariable represents a decision variable in the LP problem
// Each meal in the candidate pool becomes a variable with value 0 or 1
type MealVariable struct {
    MealID        string  // Unique identifier of the candidate meal
    Calories      float64 // Calories per 100g of the meal
    ServingWeight float64 // Standard serving weight in grams
    Index         int     // Variable index in the LP problem (0-based)
}

// ObjectiveCoefficient represents the coefficient for a single variable
// in the objective function
type ObjectiveCoefficient struct {
    VariableIndex int     // Index of the variable in the LP problem
    Coefficient   float64 // Calorie contribution when variable = 1
}

// ObjectiveFunctionSpec defines the complete objective function for the LP
type ObjectiveFunctionSpec struct {
    Coefficients []ObjectiveCoefficient // One per decision variable
    Sense        OptimizationSense      // Minimize or Maximize
}

// OptimizationSense indicates the optimization direction
type OptimizationSense int

const (
    Minimize OptimizationSense = iota // Minimize objective value (default for calories)
    Maximize                          // Maximize objective value (not used for calories)
)

// ObjectiveFunctionConfig holds configuration for building objectives
type ObjectiveFunctionConfig struct {
    CalorieScalingFactor float64 // Multiplier for calorie values (default: 1.0)
    RoundingPrecision    int     // Decimal places for coefficients (default: 2)
}

// CandidateMeal represents a meal from the database eligible for selection
type CandidateMeal struct {
    MealID        string     // Unique identifier
    Name          string     // Display name (for debugging/logging)
    Macros        MacrosPer100g
    ServingWeight float64    // Standard serving weight in grams
}

// MacrosPer100g contains nutritional values per 100g
type MacrosPer100g struct {
    Protein  float64 // grams per 100g
    Carbs    float64 // grams per 100g
    Fat      float64 // grams per 100g
    Calories float64 // kcal per 100g
}

// ObjectiveFunctionError indicates why objective building failed
type ObjectiveFunctionError struct {
    Code    string
    Message string
}

// Error codes
const (
    ErrCodeEmptyCandidates    = "EMPTY_CANDIDATES"     // No candidate meals provided
    ErrCodeInvalidCalories    = "INVALID_CALORIES"     // Negative calorie value
    ErrCodeZeroServingWeight  = "ZERO_SERVING_WEIGHT"  // Serving weight is zero
    ErrCodeNaNValue           = "NAN_VALUE"            // NaN detected in input
)
```

### 2. Logic & Algorithms (Step-by-Step)

#### 2.1 BuildObjectiveFunction - Main Entry Point

**Purpose:** Construct the objective function coefficients that represent total calories to minimize.

**Mathematical Formulation:**
```
Minimize: Z = sum(c_i * x_i) for i in 1..n

Where:
  c_i = calories contributed by meal i at serving weight
  x_i = binary decision variable (0 or 1, meal selected or not)
  n   = number of candidate meals
```

**Algorithm:**

```
FUNCTION BuildObjectiveFunction(
    candidates []CandidateMeal,
    config ObjectiveFunctionConfig
) -> (ObjectiveFunctionSpec, error):

    1. VALIDATE INPUT
       - IF candidates is empty:
           RETURN error with code EMPTY_CANDIDATES
       - FOR each candidate in candidates:
           IF candidate.Macros.Calories < 0:
               RETURN error with code INVALID_CALORIES
           IF candidate.ServingWeight <= 0:
               RETURN error with code ZERO_SERVING_WEIGHT
           IF math.IsNaN(candidate.Macros.Calories):
               RETURN error with code NAN_VALUE

    2. INITIALIZE COEFFICIENT ARRAY
       - coefficients = make([]ObjectiveCoefficient, len(candidates))

    3. CALCULATE COEFFICIENTS
       - FOR index, candidate in enumerate(candidates):
           a. Calculate calories at serving weight:
              caloriesPerServing = (candidate.Macros.Calories / 100.0)
                                   * candidate.ServingWeight

           b. Apply scaling factor (if configured):
              scaledCalories = caloriesPerServing * config.CalorieScalingFactor

           c. Apply rounding:
              roundedCalories = RoundToDecimalPlaces(
                  scaledCalories,
                  config.RoundingPrecision
              )

           d. Store coefficient:
              coefficients[index] = ObjectiveCoefficient{
                  VariableIndex: index,
                  Coefficient:   roundedCalories,
              }

    4. BUILD SPEC
       - spec = ObjectiveFunctionSpec{
           Coefficients: coefficients,
           Sense:        Minimize,
       }

    5. RETURN spec, nil
```

#### 2.2 CalorieCoefficient - Single Meal Calculation

**Purpose:** Calculate the objective coefficient for a single meal.

**Algorithm:**

```
FUNCTION CalorieCoefficient(
    meal CandidateMeal,
    config ObjectiveFunctionConfig
) -> float64:

    1. CALCULATE BASE CALORIES
       // Convert from per-100g to actual serving
       caloriesPerServing = (meal.Macros.Calories / 100.0) * meal.ServingWeight

    2. APPLY SCALING
       scaledCalories = caloriesPerServing * config.CalorieScalingFactor

    3. ROUND RESULT
       coefficient = RoundToDecimalPlaces(scaledCalories, config.RoundingPrecision)

    4. RETURN coefficient
```

#### 2.3 ApplyToSolver - Integration with CLP

**Purpose:** Apply the objective function specification to the CLP solver instance.

**Algorithm:**

```
FUNCTION ApplyToSolver(
    spec ObjectiveFunctionSpec,
    solver *clp.Simplex
) -> error:

    1. SET OPTIMIZATION DIRECTION
       IF spec.Sense == Minimize:
           solver.SetOptimizationDirection(clp.Minimize)
       ELSE:
           solver.SetOptimizationDirection(clp.Maximize)

    2. SET OBJECTIVE COEFFICIENTS
       FOR each coef in spec.Coefficients:
           solver.SetObjectiveCoefficient(coef.VariableIndex, coef.Coefficient)

    3. RETURN nil
```

#### 2.4 CalculateTotalCalories - Solution Evaluation

**Purpose:** Calculate total calories for a given solution vector.

**Algorithm:**

```
FUNCTION CalculateTotalCalories(
    spec ObjectiveFunctionSpec,
    solutionVector []float64
) -> float64:

    1. VALIDATE LENGTHS
       IF len(solutionVector) != len(spec.Coefficients):
           PANIC "solution vector length mismatch"

    2. COMPUTE DOT PRODUCT
       totalCalories = 0.0
       FOR index, coef in enumerate(spec.Coefficients):
           totalCalories += coef.Coefficient * solutionVector[index]

    3. RETURN totalCalories
```

#### 2.5 RoundToDecimalPlaces - Utility Function

**Purpose:** Round a float to specified decimal places for coefficient precision.

**Algorithm:**

```
FUNCTION RoundToDecimalPlaces(value float64, places int) -> float64:
    multiplier = math.Pow(10, float64(places))
    RETURN math.Round(value * multiplier) / multiplier
```

### 3. State Management & Error Handling

#### 3.1 Error States

| Error State | Code | Cause | Detection | Response |
|:------------|:-----|:------|:----------|:---------|
| Empty Candidates | `EMPTY_CANDIDATES` | No meals provided to optimization | len(candidates) == 0 | Return error; cannot build objective |
| Invalid Calories | `INVALID_CALORIES` | Negative calorie value in meal | calories < 0 | Return error; reject invalid data |
| Zero Serving Weight | `ZERO_SERVING_WEIGHT` | Meal has zero or negative serving | servingWeight <= 0 | Return error; cannot compute coefficient |
| NaN Value | `NAN_VALUE` | Computational error in input data | math.IsNaN(value) | Return error; reject corrupted data |
| Solver Integration Failure | (wrapped CLP error) | CLP solver rejects coefficient | CLP returns error | Propagate error to job handler |

#### 3.2 Input Validation

```
FUNCTION ValidateCandidateMeal(meal CandidateMeal) -> error:
    IF meal.MealID == "":
        RETURN error "meal ID cannot be empty"

    IF meal.ServingWeight <= 0:
        RETURN ObjectiveFunctionError{
            Code:    ErrCodeZeroServingWeight,
            Message: fmt.Sprintf("meal %s has invalid serving weight: %f",
                     meal.MealID, meal.ServingWeight),
        }

    IF meal.Macros.Calories < 0:
        RETURN ObjectiveFunctionError{
            Code:    ErrCodeInvalidCalories,
            Message: fmt.Sprintf("meal %s has negative calories: %f",
                     meal.MealID, meal.Macros.Calories),
        }

    IF math.IsNaN(meal.Macros.Calories):
        RETURN ObjectiveFunctionError{
            Code:    ErrCodeNaNValue,
            Message: fmt.Sprintf("meal %s has NaN calories", meal.MealID),
        }

    IF math.IsInf(meal.Macros.Calories, 0):
        RETURN ObjectiveFunctionError{
            Code:    ErrCodeNaNValue,
            Message: fmt.Sprintf("meal %s has Inf calories", meal.MealID),
        }

    RETURN nil
```

#### 3.3 Error Definitions

```go
var (
    ErrEmptyCandidates   = errors.New("no candidate meals provided")
    ErrInvalidCalories   = errors.New("invalid calorie value")
    ErrZeroServingWeight = errors.New("serving weight must be positive")
    ErrNaNValue          = errors.New("NaN value detected")
)

// Error implements the error interface for ObjectiveFunctionError
func (e *ObjectiveFunctionError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
```

#### 3.4 State Flow Diagram

```
                    ┌─────────────────────┐
                    │   RECEIVE_MEALS     │
                    │  (candidate list)   │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │  VALIDATE_INPUTS    │
                    └──────────┬──────────┘
                               │
              ┌────────────────┼────────────────┐
              │ Empty/Invalid  │ Valid          │
              ▼                ▼                │
    ┌─────────────────┐ ┌─────────────────────┐│
    │  RETURN_ERROR   │ │ ITERATE_CANDIDATES  ││
    └─────────────────┘ └──────────┬──────────┘│
                                   │           │
                          ┌────────┴────────┐  │
                          │ For each meal   │  │
                          ▼                 │  │
                  ┌───────────────────┐     │  │
                  │ VALIDATE_MEAL     │     │  │
                  └────────┬──────────┘     │  │
                           │                │  │
              ┌────────────┼────────────┐   │  │
              │ Invalid    │ Valid      │   │  │
              ▼            ▼            │   │  │
    ┌─────────────────┐ ┌──────────────┐│   │  │
    │  RETURN_ERROR   │ │CALC_CALORIES ││   │  │
    └─────────────────┘ └───────┬──────┘│   │  │
                                │       │   │  │
                                ▼       │   │  │
                        ┌──────────────┐│   │  │
                        │APPLY_SCALING ││   │  │
                        └───────┬──────┘│   │  │
                                │       │   │  │
                                ▼       │   │  │
                        ┌──────────────┐│   │  │
                        │ROUND_VALUE   ││   │  │
                        └───────┬──────┘│   │  │
                                │       │   │  │
                                ▼       │   │  │
                        ┌──────────────┐│   │  │
                        │STORE_COEF    ││   │  │
                        └───────┬──────┘│   │  │
                                │       │   │  │
                                └───────┴───┘  │
                                        │      │
                                        ▼      │
                            ┌─────────────────┐│
                            │  BUILD_SPEC     ││
                            └────────┬────────┘│
                                     │         │
                                     ▼         │
                            ┌─────────────────┐│
                            │  RETURN_SPEC    ││
                            └─────────────────┘│
```

#### 3.5 Relationship to LP Problem Structure

The objective function integrates with the overall LP problem as follows:

```
LP Problem Structure:
────────────────────
Minimize: Z = c₁x₁ + c₂x₂ + ... + cₙxₙ    ← ObjectiveFunction builds this

Subject to:
  Protein constraint:  p₁x₁ + p₂x₂ + ... ≥ targetProtein - tolerance
  Protein constraint:  p₁x₁ + p₂x₂ + ... ≤ targetProtein + tolerance
  Carbs constraint:    c₁x₁ + c₂x₂ + ... ≥ targetCarbs - tolerance
  Carbs constraint:    c₁x₁ + c₂x₂ + ... ≤ targetCarbs + tolerance
  Fat constraint:      f₁x₁ + f₂x₂ + ... ≥ targetFat - tolerance
  Fat constraint:      f₁x₁ + f₂x₂ + ... ≤ targetFat + tolerance

  ↑ ConstraintBuilder builds these (separate component)

  x₁, x₂, ..., xₙ ∈ {0, 1}  (binary variables)

Where:
  cᵢ = calories of meal i at serving weight (built by ObjectiveFunction)
  pᵢ = protein of meal i at serving weight
  carbsᵢ = carbohydrates of meal i at serving weight
  fᵢ = fat of meal i at serving weight
  xᵢ = 1 if meal i is selected, 0 otherwise
```

### 4. Component Interfaces

#### 4.1 Public Interface

```go
// ObjectiveFunctionBuilder constructs LP objective functions for calorie minimization
type ObjectiveFunctionBuilder interface {
    // Build creates an objective function specification from candidate meals.
    // Each candidate meal contributes its serving calories as a coefficient.
    // Parameters:
    //   - candidates: List of meals eligible for selection
    // Returns:
    //   - ObjectiveFunctionSpec with coefficients for each meal
    //   - error if validation fails or candidates is empty
    Build(candidates []CandidateMeal) (ObjectiveFunctionSpec, error)

    // CalorieCoefficient calculates the objective coefficient for a single meal.
    // Used for incremental updates or debugging.
    // Parameters:
    //   - meal: The candidate meal
    // Returns:
    //   - Coefficient value (calories at serving weight)
    CalorieCoefficient(meal CandidateMeal) float64

    // ApplyToSolver sets the objective function on a CLP solver instance.
    // Parameters:
    //   - spec: The objective function specification
    //   - solver: CLP Simplex solver instance
    // Returns:
    //   - error if solver rejects the configuration
    ApplyToSolver(spec ObjectiveFunctionSpec, solver *clp.Simplex) error

    // CalculateTotalCalories evaluates the objective value for a solution.
    // Parameters:
    //   - spec: The objective function specification
    //   - solution: Variable values from LP solution
    // Returns:
    //   - Total calories (sum of coefficient * variable value)
    CalculateTotalCalories(spec ObjectiveFunctionSpec, solution []float64) float64
}
```

#### 4.2 Constructor

```go
// NewObjectiveFunctionBuilder creates a new builder with the given configuration.
// Parameters:
//   - config: Builder configuration (scaling, rounding)
// Returns:
//   - ObjectiveFunctionBuilder implementation
func NewObjectiveFunctionBuilder(config ObjectiveFunctionConfig) ObjectiveFunctionBuilder
```

#### 4.3 Internal Functions

```go
// validateCandidate checks that a CandidateMeal has valid values.
// Parameters:
//   - meal: The meal to validate
// Returns:
//   - error if any value is invalid, nil otherwise
func validateCandidate(meal CandidateMeal) error

// calculateCaloriesAtServing computes calories for a meal at its serving weight.
// Parameters:
//   - caloriesPer100g: Calorie density
//   - servingWeight: Serving size in grams
// Returns:
//   - Total calories for the serving
func calculateCaloriesAtServing(caloriesPer100g, servingWeight float64) float64

// roundToDecimalPlaces rounds a value to the specified number of decimal places.
// Parameters:
//   - value: The value to round
//   - places: Number of decimal places
// Returns:
//   - Rounded value
func roundToDecimalPlaces(value float64, places int) float64

// buildMealVariables creates MealVariable structs with assigned indices.
// Parameters:
//   - candidates: List of candidate meals
// Returns:
//   - Slice of MealVariable with assigned indices
func buildMealVariables(candidates []CandidateMeal) []MealVariable
```

#### 4.4 Default Configuration

```go
var DefaultObjectiveFunctionConfig = ObjectiveFunctionConfig{
    CalorieScalingFactor: 1.0, // No scaling (use raw calorie values)
    RoundingPrecision:    2,   // Round to 2 decimal places
}
```

#### 4.5 Usage Example (Pseudocode)

```go
// Define candidate meals
candidates := []CandidateMeal{
    {
        MealID:        "meal-001",
        Name:          "Grilled Chicken Breast",
        Macros:        MacrosPer100g{Protein: 31.0, Carbs: 0.0, Fat: 3.6, Calories: 165},
        ServingWeight: 150.0,
    },
    {
        MealID:        "meal-002",
        Name:          "Brown Rice",
        Macros:        MacrosPer100g{Protein: 2.6, Carbs: 23.0, Fat: 0.9, Calories: 111},
        ServingWeight: 200.0,
    },
    {
        MealID:        "meal-003",
        Name:          "Steamed Broccoli",
        Macros:        MacrosPer100g{Protein: 2.8, Carbs: 7.0, Fat: 0.4, Calories: 34},
        ServingWeight: 100.0,
    },
}

// Build objective function
builder := NewObjectiveFunctionBuilder(DefaultObjectiveFunctionConfig)
spec, err := builder.Build(candidates)
if err != nil {
    // Handle error
}

// spec.Coefficients:
//   [0]: 247.50 (165 * 150/100)  - Chicken
//   [1]: 222.00 (111 * 200/100)  - Rice
//   [2]:  34.00 (34 * 100/100)   - Broccoli

// Apply to CLP solver
solver := clp.NewSimplex()
// ... add variables and constraints via ConstraintBuilder ...
err = builder.ApplyToSolver(spec, solver)

// After solving, evaluate solution
solution := solver.PrimalColumnSolution() // e.g., [1.0, 1.0, 0.0]
totalCalories := builder.CalculateTotalCalories(spec, solution)
// totalCalories = 247.50 + 222.00 + 0 = 469.50 kcal
```

#### 4.6 Integration with ARCH-004 Components

| Component | Interaction |
|:----------|:------------|
| **ConstraintBuilder** | Receives same `candidates` list; builds macro constraints for same variable indices |
| **DiversityPenalizer** | May modify coefficients to add penalty weights for original diet meals |
| **LPSolverWrapper** | Receives `ObjectiveFunctionSpec` via `ApplyToSolver`; executes optimization |
| **SolutionValidator** | Uses `CalculateTotalCalories` to verify solution quality |

#### 4.7 Performance Constraints

| Constraint | Target | Enforcement |
|:-----------|:-------|:------------|
| Build time for 1000 candidates | < 10ms | O(n) single-pass coefficient calculation |
| Memory per coefficient | 24 bytes | Single `ObjectiveCoefficient` struct |
| Total memory for 1000 candidates | < 24KB | Pre-allocated coefficient slice |
| Floating-point precision | 15 significant digits | float64 type |
