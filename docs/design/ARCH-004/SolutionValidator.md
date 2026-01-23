# FILE: SolutionValidator.md
**Traceability:** ARCH-004

---

## 1. Data Structures & Types

```go
package optimizer

import (
    "time"
)

type MacroTolerance struct {
    ProteinTolerance float64 // e.g., 0.05 for ±5%
    CarbTolerance    float64 // e.g., 0.05 for ±5%
    FatTolerance     float64 // e.g., 0.05 for ±5%
    CalorieTolerance float64 // e.g., 0.10 for ±10%
}

type MacroTarget struct {
    Protein float64
    Carbs   float64
    Fat     float64
    Calories float64
}

type Meal struct {
    ID           string
    Name         string
    Protein      float64
    Carbs        float64
    Fat          float64
    Calories     float64
    MealType     string // "breakfast", "lunch", "dinner", "snack"
    Ingredients  []string
}

type DietAlternative struct {
    ID               string
    Meals            []Meal
    TotalProtein     float64
    TotalCarbs       float64
    TotalFat         float64
    TotalCalories    float64
    DiversityScore   float64
    SimilarityScore  float64
}

type LPSolution struct {
    SelectedMealIDs  []string
    SelectedMeals    []Meal
    ObjectiveValue   float64
    Status           SolutionStatus
    Constraints      map[string]float64
}

type SolutionStatus string

const (
    SolutionStatusValid   SolutionStatus = "valid"
    SolutionStatusInvalid SolutionStatus = "invalid"
    SolutionStatusPartial SolutionStatus = "partial"
)

type ValidationResult struct {
    IsValid         bool
    Status          SolutionStatus
    MacroDeviations []MacroDeviation
    ConstraintGaps  map[string]float64
    Warnings        []ValidationWarning
    Timestamp       time.Time
}

type MacroDeviation struct {
    MacroType    string
    TargetValue  float64
    ActualValue  float64
    DeviationPct float64
    IsAcceptable bool
}

type ValidationWarning struct {
    Code        string
    Message     string
    Severity    string // "low", "medium", "high"
}

type ValidationConfig struct {
    Tolerance               MacroTolerance
    MinMealsCount           int
    MaxMealsCount           int
    MaxMacroDeviationPct    float64
    MinDiversityThreshold   float64
    EnablePartialValidation bool
    Timeout                 time.Duration
}

var DefaultValidationConfig = ValidationConfig{
    Tolerance: MacroTolerance{
        ProteinTolerance:  0.05,
        CarbTolerance:     0.05,
        FatTolerance:      0.05,
        CalorieTolerance:  0.10,
    },
    MinMealsCount:         3,
    MaxMealsCount:         6,
    MaxMacroDeviationPct:  0.15,
    MinDiversityThreshold: 0.3,
    EnablePartialValidation: true,
    Timeout:               5 * time.Second,
}
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 ValidateSolution

**Algorithm:** ValidateSolution(solution LPSolution, target MacroTarget, config ValidationConfig) ValidationResult

```
1. Initialize ValidationResult with timestamp and SolutionStatusInvalid

2. IF solution.SelectedMealIDs is empty THEN
       Add ValidationWarning(W001, "No meals selected in solution")
       RETURN ValidationResult with IsValid=false
   END IF

3. IF solution.Status indicates infeasible/unbounded THEN
       Add ValidationWarning(W002, "LP solution is infeasible or unbounded")
       RETURN ValidationResult with IsValid=false
   END IF

4. Validate meal count:
   4.1 mealsCount = COUNT(solution.SelectedMeals)
   4.2 IF mealsCount < config.MinMealsCount THEN
           Add ValidationWarning(W003, "Meal count below minimum")
           RETURN ValidationResult with IsValid=false
       END IF
   4.3 IF mealsCount > config.MaxMealsCount THEN
           Add ValidationWarning(W004, "Meal count exceeds maximum")
           RETURN ValidationResult with IsValid=false
       END IF

5. Calculate total macros from selected meals:
   5.1 totalProtein = SUM(m.Protein FOR m IN solution.SelectedMeals)
   5.2 totalCarbs   = SUM(m.Carbs FOR m IN solution.SelectedMeals)
   5.3 totalFat     = SUM(m.Fat FOR m IN solution.SelectedMeals)
   5.4 totalCalories = SUM(m.Calories FOR m IN solution.SelectedMeals)

6. Validate each macro against target with tolerance:
   6.1 proteinDeviation = ABS(totalProtein - target.Protein) / target.Protein
   6.2 IF proteinDeviation > config.Tolerance.ProteinTolerance THEN
           Add MacroDeviation("protein", target.Protein, totalProtein, proteinDeviation, false)
       ELSE
           Add MacroDeviation("protein", target.Protein, totalProtein, proteinDeviation, true)
       END IF

   6.3 carbsDeviation = ABS(totalCarbs - target.Carbs) / target.Carbs
   6.4 IF carbsDeviation > config.Tolerance.CarbTolerance THEN
           Add MacroDeviation("carbs", target.Carbs, totalCarbs, carbsDeviation, false)
       ELSE
           Add MacroDeviation("carbs", target.Carbs, totalCarbs, carbsDeviation, true)
       END IF

   6.5 fatDeviation = ABS(totalFat - target.Fat) / target.Fat
   6.6 IF fatDeviation > config.Tolerance.FatTolerance THEN
           Add MacroDeviation("fat", target.Fat, totalFat, fatDeviation, false)
       ELSE
           Add MacroDeviation("fat", target.Fat, totalFat, fatDeviation, true)
       END IF

   6.7 calorieDeviation = ABS(totalCalories - target.Calories) / target.Calories
   6.8 IF calorieDeviation > config.Tolerance.CalorieTolerance THEN
           Add MacroDeviation("calories", target.Calories, totalCalories, calorieDeviation, false)
       ELSE
           Add MacroDeviation("calories", target.Calories, totalCalories, calorieDeviation, true)
       END IF

7. Determine overall validity:
   7.1 allMacrosAcceptable = ALL(d.IsAcceptable FOR d IN MacroDeviations)
   7.2 IF allMacrosAcceptable THEN
           Set Status = SolutionStatusValid, IsValid = true
       ELSE IF config.EnablePartialValidation AND hasMajorMacrosAcceptable() THEN
           Set Status = SolutionStatusPartial, IsValid = true
       ELSE
           Set Status = SolutionStatusInvalid, IsValid = false
       END IF

8. Validate diversity requirements:
   8.1 diversityScore = CalculateDiversityScore(solution.SelectedMeals)
   8.2 IF diversityScore < config.MinDiversityThreshold THEN
           Add ValidationWarning(W005, "Solution lacks sufficient diversity")
       END IF

9. Validate meal type coverage:
   9.1 mealTypes = EXTRACT(m.MealType FOR m IN solution.SelectedMeals)
   9.2 IF breakfast NOT IN mealTypes THEN
           Add ValidationWarning(W006, "Missing breakfast meal")
       END IF
   9.3 IF lunch NOT IN mealTypes AND dinner NOT IN mealTypes THEN
           Add ValidationWarning(W007, "Missing main meals (lunch/dinner)")
       END IF

10. Populate ConstraintGaps with deviation values:
    10.1 ConstraintGaps["protein_gap"] = totalProtein - target.Protein
    10.2 ConstraintGaps["carbs_gap"] = totalCarbs - target.Carbs
    10.3 ConstraintGaps["fat_gap"] = totalFat - target.Fat
    10.4 ConstraintGaps["calories_gap"] = totalCalories - target.Calories

11. RETURN ValidationResult
```

### 2.2 ValidateMultipleSolutions

**Algorithm:** ValidateMultipleSolutions(solutions []LPSolution, target MacroTarget, config ValidationConfig) []ValidationResult

```
1. results = EMPTY_LIST

2. FOR i, solution IN solutions DO
       result = ValidateSolution(solution, target, config)
       result.SolutionIndex = i
       APPEND result TO results
   END FOR

3. RETURN results
```

### 2.3 CalculateDiversityScore

**Algorithm:** CalculateDiversityScore(meals []Meal) float64

```
1. IF meals is empty THEN RETURN 0.0 END IF

2. ingredientSet = EMPTY_SET
3. FOR meal IN meals DO
       FOR ingredient IN meal.Ingredients DO
           ADD ingredient TO ingredientSet
       END FOR
   END FOR

4. totalIngredients = SIZE(ingredientSet)

5. mealCategorySet = EMPTY_SET
6. FOR meal IN meals DO
       ADD meal.MealType TO mealCategorySet
   END FOR

7. categoryCount = SIZE(mealCategorySet)

8. calorieVariance = CalculateVariance(meals.Calories)

9. diversityScore = (categoryCount / 4.0) * 0.3 + (totalIngredients / 50.0) * 0.4 + (calorieVariance / 10000.0) * 0.3

10. RETURN MIN(diversityScore, 1.0)
```

### 2.4 ValidateConstraintsFeasibility

**Algorithm:** ValidateConstraintsFeasibility(target MacroTarget, availableMeals []Meal, config ValidationConfig) (bool, error)

```
1. IF availableMeals is empty THEN
       RETURN false, Error("No available meals for validation")
   END IF

2. minProtein = MIN(m.Protein FOR m IN availableMeals)
3. maxProtein = MAX(m.Protein FOR m IN availableMeals)
4. IF target.Protein < minProtein OR target.Protein > maxProtein * 10 THEN
       RETURN false, nil
   END IF

5. minCarbs = MIN(m.Carbs FOR m IN availableMeals)
6. maxCarbs = MAX(m.Carbs FOR m IN availableMeals)
7. IF target.Carbs < minCarbs OR target.Carbs > maxCarbs * 10 THEN
       RETURN false, nil
   END IF

8. minFat = MIN(m.Fat FOR m IN availableMeals)
9. maxFat = MAX(m.Fat FOR m IN availableMeals)
10. IF target.Fat < minFat OR target.Fat > maxFat * 10 THEN
        RETURN false, nil
    END IF

11. minCalories = MIN(m.Calories FOR m IN availableMeals)
12. maxCalories = MAX(m.Calories FOR m IN availableMeals)
13. IF target.Calories < minCalories OR target.Calories > maxCalories * 10 THEN
        RETURN false, nil
    END IF

14. RETURN true, nil
```

---

## 3. State Management & Error Handling

### 3.1 Possible Error States

| Error Code | Condition | Severity | Recovery Action |
| :--- | :--- | :--- | :--- |
| E001 | Empty solution provided | high | Re-run optimization with relaxed constraints |
| E002 | Infeasible LP solution | high | Adjust target macros or add more meals to pool |
| E003 | Unbounded objective | high | Add upper bounds to meal quantities |
| E004 | Meal count below minimum | medium | Reduce min meals threshold or add more meals |
| E005 | Meal count exceeds maximum | low | Truncate to max meals, keep highest diversity |
| E006 | Protein deviation exceeded | medium | Log warning, may still be valid if partial allowed |
| E007 | Carbs deviation exceeded | medium | Log warning, may still be valid if partial allowed |
| E008 | Fat deviation exceeded | medium | Log warning, may still be valid if partial allowed |
| E009 | Calorie deviation exceeded | low | Log warning (calorie minimization may cause deviation) |
| E010 | Diversity below threshold | low | Suggest more diverse meal pool |
| E011 | Missing breakfast meal | medium | Warn user, may be acceptable for some diets |
| E012 | Missing main meals | high | Require at least one main meal |
| E013 | Validation timeout | medium | Return partial result if enabled |
| E014 | No available meals | high | Abort, require meal pool population |
| E015 | Target out of feasible range | high | Return error, suggest adjusted targets |

### 3.2 State Transitions

```
Initial State: Not Validated

    [ValidateSolution called]
            |
            v
    [Validating] --(empty solution)--> [Invalid: E001]
            |
            --(LP infeasible)---------> [Invalid: E002]
            |
            --(LP unbounded)---------> [Invalid: E003]
            |
            --(meal count invalid)----> [Invalid: E004/E005]
            |
            --(all macros acceptable)--> [Valid]
            |
            --(partial allowed + major OK)--> [Partial]
            |
            --(otherwise)------------> [Invalid]
```

### 3.3 Error Recovery Strategies

- **Partial Validation:** When `EnablePartialValidation=true`, solutions with minor macro deviations (≤15%) are marked as partial success rather than failure
- **Graceful Degradation:** On timeout (E013), return best validation achieved with timeout warning
- **Feedback Loop:** Constraint gaps are returned to caller for iterative constraint relaxation

---

## 4. Component Interfaces

### 4.1 Public Functions

```go
// NewSolutionValidator creates a new SolutionValidator with given config
func NewSolutionValidator(config ValidationConfig) *SolutionValidator

// ValidateSolution validates a single LP solution against target macros
func (sv *SolutionValidator) ValidateSolution(
    solution LPSolution,
    target MacroTarget,
) ValidationResult

// ValidateMultipleSolutions validates multiple solutions and returns results
func (sv *SolutionValidator) ValidateMultipleSolutions(
    solutions []LPSolution,
    target MacroTarget,
) []ValidationResult

// ValidateConstraintsFeasibility checks if target is achievable with available meals
func (sv *SolutionValidator) ValidateConstraintsFeasibility(
    target MacroTarget,
    availableMeals []Meal,
) (bool, error)

// CalculateDiversityScore calculates diversity metric for a set of meals
func (sv *SolutionValidator) CalculateDiversityScore(
    meals []Meal,
) float64

// GetConstraintGaps returns the deviation of actual macros from targets
func (sv *SolutionValidator) GetConstraintGaps(
    solution LPSolution,
    target MacroTarget,
) map[string]float64

// IsSolutionAcceptable checks if validation result meets acceptance criteria
func (sv *SolutionValidator) IsSolutionAcceptable(
    result ValidationResult,
) bool
```

### 4.2 Internal Helper Functions

```go
// validateProtein validates protein macro
func (sv *SolutionValidator) validateProtein(
    totalProtein float64,
    target Protein float64,
) MacroDeviation

// validateCarbs validates carbohydrate macro
func (sv *SolutionValidator) validateCarbs(
    totalCarbs float64,
    target Carbs float64,
) MacroDeviation

// validateFat validates fat macro
func (sv *SolutionValidator) validateFat(
    totalFat float64,
    target Fat float64,
) MacroDeviation

// validateCalories validates calorie macro
func (sv *SolutionValidator) validateCalories(
    totalCalories float64,
    target Calories float64,
) MacroDeviation

// hasMajorMacrosAcceptable checks if critical macros are within tolerance
func (sv *SolutionValidator) hasMajorMacrosAcceptable(
    deviations []MacroDeviation,
) bool

// buildValidationWarnings creates warnings from validation result
func (sv *SolutionValidator) buildValidationWarnings(
    result *ValidationResult,
)

// CalculateVariance calculates statistical variance of a slice
func (sv *SolutionValidator) CalculateVariance(
    values []float64,
) float64
```

### 4.3 Usage Example

```go
func ExampleSolutionValidator_Usage() {
    config := DefaultValidationConfig
    validator := NewSolutionValidator(config)

    target := MacroTarget{
        Protein:  150,
        Carbs:    200,
        Fat:      65,
        Calories: 2000,
    }

    solution := LPSolution{
        SelectedMealIDs: []string{"meal1", "meal2", "meal3"},
        SelectedMeals: []Meal{
            {ID: "meal1", Protein: 50, Carbs: 60, Fat: 20, Calories: 600},
            {ID: "meal2", Protein: 40, Carbs: 70, Fat: 25, Calories: 650},
            {ID: "meal3", Protein: 60, Carbs: 70, Fat: 20, Calories: 700},
        },
        ObjectiveValue: 1950,
        Status:         SolutionStatusOptimal,
    }

    result := validator.ValidateSolution(solution, target)

    if result.IsValid {
        fmt.Printf("Solution is valid with %s status\n", result.Status)
    } else {
        fmt.Printf("Solution invalid: %v\n", result.Warnings)
    }
}
```
