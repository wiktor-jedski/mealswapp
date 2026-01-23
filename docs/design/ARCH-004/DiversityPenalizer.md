## FILE: DiversityPenalizer.md
**Traceability:** ARCH-004

### 1. Data Structures & Types

```go
// DiversityPenalizer adds penalty weights to LP objective function
// to discourage selection of meals already present in the user's diet.

// PenaltyWeight represents a penalty term for a specific meal in the LP objective
type PenaltyWeight struct {
    MealID  string  // Unique identifier of the meal to penalize
    Weight  float64 // Penalty coefficient added to objective function
}

// DiversityPenaltyRequest contains the inputs needed to compute penalties
type DiversityPenaltyRequest struct {
    OriginalMealIDs []string       // Meal IDs from the user's current diet
    CandidateMeals  []CandidateMeal // All meals being considered by the LP solver
}

// CandidateMeal represents a meal that the LP solver may select
type CandidateMeal struct {
    MealID   string  // Unique identifier
    Calories float64 // Calories per serving (used for proportional penalty scaling)
}

// DiversityPenaltyResult contains the computed penalty weights
type DiversityPenaltyResult struct {
    Penalties      []PenaltyWeight // Penalty weights to add to objective
    PenalizedCount int             // Number of meals that received penalties
}

// DiversityPenalizerConfig holds configuration parameters
type DiversityPenalizerConfig struct {
    BasePenalty          float64 // Base penalty coefficient for original meals (default: 100.0)
    CalorieScalingFactor float64 // Scale penalty proportionally to calories (default: 0.1)
    MaxPenalty           float64 // Cap to prevent extreme coefficients (default: 500.0)
}

// PenaltyMode determines how penalties are calculated
type PenaltyMode int

const (
    // PenaltyModeFixed applies the same penalty to all original meals
    PenaltyModeFixed PenaltyMode = iota

    // PenaltyModeProportional scales penalty based on meal calories
    // Higher-calorie meals get larger penalties since they have more
    // impact on the total diet
    PenaltyModeProportional
)
```

### 2. Logic & Algorithms (Step-by-Step)

#### 2.1 Main Penalty Calculation Flow

```
FUNCTION CalculatePenalties(
    request DiversityPenaltyRequest,
    config DiversityPenaltyConfig,
    mode PenaltyMode
) -> DiversityPenaltyResult:

    1. INPUT VALIDATION
       - IF request.OriginalMealIDs is empty:
           RETURN DiversityPenaltyResult{Penalties: [], PenalizedCount: 0}
       - IF request.CandidateMeals is empty:
           RETURN DiversityPenaltyResult{Penalties: [], PenalizedCount: 0}

    2. BUILD ORIGINAL MEAL LOOKUP SET
       // O(1) lookup for checking if a candidate is in original diet
       originalSet = make(map[string]bool)
       FOR each mealID in request.OriginalMealIDs:
           originalSet[mealID] = true

    3. CALCULATE PENALTIES FOR MATCHING CANDIDATES
       penalties = []PenaltyWeight{}
       penalizedCount = 0

       FOR each candidate in request.CandidateMeals:
           IF originalSet[candidate.MealID] == true:
               // This candidate is in the original diet; apply penalty

               penalty = CalculateSinglePenalty(
                   candidate,
                   config,
                   mode
               )

               APPEND PenaltyWeight{
                   MealID: candidate.MealID,
                   Weight: penalty,
               } to penalties

               penalizedCount++

    4. RETURN RESULT
       RETURN DiversityPenaltyResult{
           Penalties:      penalties,
           PenalizedCount: penalizedCount,
       }
```

#### 2.2 Single Penalty Calculation

```
FUNCTION CalculateSinglePenalty(
    candidate CandidateMeal,
    config DiversityPenalizerConfig,
    mode PenaltyMode
) -> float64:

    1. DETERMINE BASE PENALTY
       SWITCH mode:
           CASE PenaltyModeFixed:
               penalty = config.BasePenalty

           CASE PenaltyModeProportional:
               // Scale penalty based on calorie contribution
               // Higher-calorie meals get larger penalties because
               // replacing them has more impact on total diet macros
               calorieComponent = candidate.Calories * config.CalorieScalingFactor
               penalty = config.BasePenalty + calorieComponent

    2. APPLY MAXIMUM CAP
       // Prevent extremely large coefficients that could cause
       // numerical instability in the LP solver (go-coinor/clp)
       IF penalty > config.MaxPenalty:
           penalty = config.MaxPenalty

    3. ENSURE NON-NEGATIVE
       // Penalties must be positive to increase objective (minimize calories)
       IF penalty < 0:
           penalty = 0

    4. RETURN penalty
```

#### 2.3 Integration with LP Objective Function

```
FUNCTION ApplyPenaltiesToObjective(
    baseObjective map[string]float64,  // MealID -> calorie coefficient
    penalties []PenaltyWeight
) -> map[string]float64:
    // The LP solver minimizes: sum(coefficient[i] * x[i]) for all meals i
    // Base coefficient = calories of meal
    // With penalty: coefficient = calories + penalty
    // This makes selecting original meals more "costly" in the objective

    1. COPY BASE OBJECTIVE
       modifiedObjective = make(map[string]float64)
       FOR mealID, coefficient in baseObjective:
           modifiedObjective[mealID] = coefficient

    2. ADD PENALTY TERMS
       FOR each penalty in penalties:
           IF mealID exists in modifiedObjective:
               modifiedObjective[penalty.MealID] += penalty.Weight
           ELSE:
               // Meal not in candidate set; ignore penalty
               LOG warning: "Penalty for unknown meal ID: %s", penalty.MealID

    3. RETURN modifiedObjective
```

#### 2.4 Iterative Exclusion for Multi-Solution Generation

```
FUNCTION UpdatePenaltiesForIteration(
    basePenalties []PenaltyWeight,
    previousSolutions [][]string,  // Meal IDs selected in previous iterations
    iterationPenalty float64       // Additional penalty per iteration (default: 200.0)
) -> []PenaltyWeight:
    // After generating solution N, penalize those meals heavily
    // to encourage solution N+1 to be different

    1. CREATE PENALTY MAP FROM BASE
       penaltyMap = make(map[string]float64)
       FOR each p in basePenalties:
           penaltyMap[p.MealID] = p.Weight

    2. ADD ESCALATING PENALTIES FOR PREVIOUS SOLUTIONS
       FOR iterationIndex, solution in previousSolutions:
           // Each previous iteration adds more penalty
           // Iteration 0 (first solution) gets 1x penalty
           // Iteration 1 (second solution) gets 2x penalty, etc.
           multiplier = float64(iterationIndex + 1)

           FOR each mealID in solution:
               currentPenalty = penaltyMap[mealID] OR 0
               penaltyMap[mealID] = currentPenalty + (iterationPenalty * multiplier)

    3. CONVERT BACK TO SLICE
       result = []PenaltyWeight{}
       FOR mealID, weight in penaltyMap:
           APPEND PenaltyWeight{MealID: mealID, Weight: weight} to result

    4. RETURN result
```

### 3. State Management & Error Handling

#### 3.1 Error States

| Error State | Cause | Detection | Response |
|:------------|:------|:----------|:---------|
| Empty Original Meals | User has no meals in their diet | len(OriginalMealIDs) == 0 | Return empty penalties; LP runs without diversity bias |
| Empty Candidate Set | No meals available for optimization | len(CandidateMeals) == 0 | Return empty penalties; let LP handle constraint infeasibility |
| Negative Calories | Invalid meal data | candidate.Calories < 0 | Log warning; treat as 0 calories for penalty calculation |
| Penalty Overflow | Extremely high calorie values | penalty > MaxPenalty | Cap at MaxPenalty; log warning |
| Unknown Meal ID | Penalty references meal not in LP | MealID not in baseObjective | Log warning; skip this penalty (no-op) |
| Duplicate Meal IDs | Same meal appears multiple times in original | Duplicate in OriginalMealIDs | Deduplicate via set; each meal penalized once |

#### 3.2 Input Validation

```
FUNCTION ValidateDiversityPenaltyRequest(request DiversityPenaltyRequest) -> error:
    // Allow empty OriginalMealIDs (no penalty case)
    // Allow empty CandidateMeals (no-op case)

    FOR each mealID in request.OriginalMealIDs:
        IF mealID == "":
            RETURN ErrEmptyMealID("original meal ID cannot be empty string")

    FOR each candidate in request.CandidateMeals:
        IF candidate.MealID == "":
            RETURN ErrEmptyMealID("candidate meal ID cannot be empty string")
        IF candidate.Calories < 0:
            RETURN ErrInvalidCalories("calories cannot be negative for meal: %s", candidate.MealID)

    RETURN nil

FUNCTION ValidateDiversityPenalizerConfig(config DiversityPenalizerConfig) -> error:
    IF config.BasePenalty < 0:
        RETURN ErrInvalidConfig("base penalty cannot be negative")
    IF config.CalorieScalingFactor < 0:
        RETURN ErrInvalidConfig("calorie scaling factor cannot be negative")
    IF config.MaxPenalty <= 0:
        RETURN ErrInvalidConfig("max penalty must be positive")
    IF config.MaxPenalty < config.BasePenalty:
        RETURN ErrInvalidConfig("max penalty must be >= base penalty")

    RETURN nil
```

#### 3.3 Error Definitions

```go
var (
    ErrEmptyMealID      = errors.New("meal ID cannot be empty")
    ErrInvalidCalories  = errors.New("invalid calorie value")
    ErrInvalidConfig    = errors.New("invalid penalizer configuration")
)
```

#### 3.4 State Transition Diagram

```
                    ┌─────────────────────┐
                    │   RECEIVE_REQUEST   │
                    └──────────┬──────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │  VALIDATE_REQUEST   │
                    └──────────┬──────────┘
                               │
              ┌────────────────┼────────────────┐
              │ Invalid        │ Valid          │
              ▼                ▼                │
    ┌─────────────────┐  ┌─────────────────────┐│
    │  RETURN_ERROR   │  │ CHECK_EMPTY_INPUTS  ││
    └─────────────────┘  └──────────┬──────────┘│
                                    │           │
              ┌─────────────────────┼───────────┤
              │ Empty               │ Non-empty │
              ▼                     ▼           │
    ┌─────────────────┐  ┌─────────────────────┐│
    │RETURN_EMPTY_RESULT│ │ BUILD_LOOKUP_SET   ││
    └─────────────────┘  └──────────┬──────────┘│
                                    │           │
                                    ▼           │
                         ┌─────────────────────┐│
                         │  ITERATE_CANDIDATES ││
                         └──────────┬──────────┘│
                                    │           │
                    ┌───────────────┼───────────┤
                    │ For each candidate        │
                    ▼                           │
          ┌─────────────────────┐               │
          │ CHECK_IN_ORIGINAL   │               │
          └──────────┬──────────┘               │
                     │                          │
        ┌────────────┼────────────┐             │
        │ Not in     │ In original│             │
        │ original   ▼            │             │
        │  ┌─────────────────┐    │             │
        │  │CALCULATE_PENALTY│    │             │
        │  └──────────┬──────┘    │             │
        │             │           │             │
        │             ▼           │             │
        │  ┌─────────────────┐    │             │
        │  │  APPLY_CAP      │    │             │
        │  └──────────┬──────┘    │             │
        │             │           │             │
        │             ▼           │             │
        │  ┌─────────────────┐    │             │
        │  │ ADD_TO_RESULT   │    │             │
        │  └────────┬────────┘    │             │
        │           │             │             │
        └───────────┴─────────────┘             │
                    │                           │
                    └───────────────────────────┘
                    │
                    ▼
          ┌─────────────────────┐
          │   RETURN_RESULT     │
          └─────────────────────┘
```

#### 3.5 Performance Constraints

| Constraint | Target | Enforcement |
|:-----------|:-------|:------------|
| Penalty calculation (N candidates) | O(N) | Single pass over candidates with O(1) set lookup |
| Memory overhead | O(M) where M = original meals | Hash set for original meal IDs |
| Single penalty calculation | < 0.1ms | Arithmetic operations only |
| 1000 candidates with 50 originals | < 5ms | Linear time complexity |
| Numerical stability | float64 precision | Cap penalties at MaxPenalty to avoid solver issues |

### 4. Component Interfaces

#### 4.1 Public Interface

```go
// DiversityPenalizer computes penalty weights to encourage diverse meal selections
// in the LP optimization process.
type DiversityPenalizer interface {
    // CalculatePenalties computes penalty weights for meals present in the original diet.
    // These penalties are added to the LP objective function coefficients.
    // Parameters:
    //   - request: Contains original meal IDs and candidate meals
    // Returns:
    //   - DiversityPenaltyResult with computed penalties
    //   - error if input validation fails
    CalculatePenalties(request DiversityPenaltyRequest) (DiversityPenaltyResult, error)

    // ApplyToObjective modifies an LP objective function by adding diversity penalties.
    // Parameters:
    //   - baseObjective: Map of MealID to base coefficient (typically calories)
    //   - penalties: Penalty weights from CalculatePenalties
    // Returns:
    //   - Modified objective with penalties applied
    ApplyToObjective(baseObjective map[string]float64, penalties []PenaltyWeight) map[string]float64

    // UpdateForIteration adds escalating penalties for previously selected solutions.
    // Used when generating multiple distinct alternatives (up to 3 per ARCH-004).
    // Parameters:
    //   - basePenalties: Original diversity penalties
    //   - previousSolutions: Meal IDs selected in each previous iteration
    // Returns:
    //   - Updated penalties with escalation for previous solutions
    UpdateForIteration(basePenalties []PenaltyWeight, previousSolutions [][]string) []PenaltyWeight
}
```

#### 4.2 Constructor

```go
// NewDiversityPenalizer creates a new penalizer with the given configuration.
// Parameters:
//   - config: Penalizer configuration (base penalty, scaling, max)
//   - mode: Fixed or proportional penalty mode
// Returns:
//   - DiversityPenalizer implementation
//   - error if configuration is invalid
func NewDiversityPenalizer(config DiversityPenalizerConfig, mode PenaltyMode) (DiversityPenalizer, error)
```

#### 4.3 Internal Functions

```go
// buildOriginalMealSet creates a hash set for O(1) lookup of original meal IDs.
// Parameters:
//   - originalMealIDs: Slice of meal IDs in user's current diet
// Returns:
//   - Map with meal IDs as keys for fast membership testing
func buildOriginalMealSet(originalMealIDs []string) map[string]bool

// calculateSinglePenalty computes the penalty weight for one meal.
// Parameters:
//   - candidate: The meal to calculate penalty for
//   - config: Penalizer configuration
//   - mode: Fixed or proportional mode
// Returns:
//   - Penalty coefficient (capped at MaxPenalty)
func calculateSinglePenalty(candidate CandidateMeal, config DiversityPenalizerConfig, mode PenaltyMode) float64

// capPenalty ensures penalty does not exceed configured maximum.
// Parameters:
//   - penalty: Calculated penalty value
//   - maxPenalty: Maximum allowed value
// Returns:
//   - Capped penalty value
func capPenalty(penalty float64, maxPenalty float64) float64

// deduplicateMealIDs removes duplicate entries from a slice of meal IDs.
// Parameters:
//   - mealIDs: Slice that may contain duplicates
// Returns:
//   - Deduplicated slice preserving first occurrence order
func deduplicateMealIDs(mealIDs []string) []string
```

#### 4.4 Default Configuration

```go
var DefaultDiversityPenalizerConfig = DiversityPenalizerConfig{
    BasePenalty:          100.0, // Base penalty for any original meal
    CalorieScalingFactor: 0.1,   // Add 0.1 penalty per calorie
    MaxPenalty:           500.0, // Cap to prevent numerical issues in go-coinor/clp
}

const DefaultIterationPenalty = 200.0 // Additional penalty per previous selection
```

#### 4.5 Usage Example with LP Solver

```
// Example: Optimizing a diet with 5 original meals

1. User submits optimization request:
   - originalMeals: [meal_A (400 cal), meal_B (300 cal), meal_C (250 cal), ...]
   - targetMacros: { protein: 150g, carbs: 200g, fat: 70g }

2. LP solver setup (LPSolverWrapper from ARCH-004):
   - Decision variables: x[i] for each candidate meal (quantity to include)
   - Base objective: minimize sum(calories[i] * x[i])

3. DiversityPenalizer calculates penalties:
   - meal_A: 100 + (400 * 0.1) = 140 penalty
   - meal_B: 100 + (300 * 0.1) = 130 penalty
   - meal_C: 100 + (250 * 0.1) = 125 penalty

4. Modified objective applied:
   - For meal_A: coefficient = 400 + 140 = 540 (effectively more expensive)
   - For new_meal_X (not in original): coefficient = 350 (no penalty)

5. Result: LP solver prefers new_meal_X over meal_A even though
   meal_A has fewer calories, because the penalty makes meal_A
   appear more costly in the objective function.

6. Multi-solution generation (ARCH-004 requires up to 3 alternatives):
   - Solution 1: [new_meal_X, new_meal_Y, meal_C]
   - Update penalties: new_meal_X and new_meal_Y get +200 penalty each
   - Solution 2: [new_meal_Z, meal_B, new_meal_W]
   - Update penalties: escalate previous + add new
   - Solution 3: Different combination encouraged by accumulated penalties
```

#### 4.6 Mathematical Foundation

**Objective Function Modification:**
```
Original LP objective (calorie minimization):
    minimize: Σ calories[i] × x[i]

With diversity penalty:
    minimize: Σ (calories[i] + penalty[i]) × x[i]

Where:
    penalty[i] = BasePenalty + (calories[i] × ScalingFactor)  if meal[i] in original diet
    penalty[i] = 0                                             otherwise

Iterative exclusion (for solution k):
    penalty[i] += IterationPenalty × count(meal[i] in solutions 1..k-1)
```

**Rationale:**
- Adding penalty to the objective coefficient makes the LP solver "see" original meals as having a higher calorie cost
- The solver naturally avoids these meals when lower-cost alternatives exist
- Proportional scaling ensures high-calorie original meals get proportionally higher penalties
- Iterative escalation ensures each of the 3 alternative solutions is distinct
