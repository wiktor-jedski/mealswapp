## FILE: MacroVectorNormalizer.md
**Traceability:** ARCH-003

### 1. Data Structures & Types

```go
// MacroValues represents raw macronutrient values (per 100g or 100ml)
type MacroValues struct {
    Protein float64 // grams
    Carbs   float64 // grams
    Fat     float64 // grams
}

// MacroVector represents a normalized unit vector in 3D macro space
type MacroVector struct {
    P         float64 // normalized protein component
    C         float64 // normalized carbohydrate component
    F         float64 // normalized fat component
    Magnitude float64 // original vector magnitude (preserved for reference)
}

// RecipeIngredient represents a single ingredient with quantity
type RecipeIngredient struct {
    ItemID   string
    Macros   MacroValues
    Quantity float64 // grams
}

// NormalizationError indicates why normalization failed
type NormalizationError struct {
    Code    string
    Message string
}

// Error codes
const (
    ErrCodeZeroVector     = "ZERO_VECTOR"      // All macro values are zero
    ErrCodeNegativeValues = "NEGATIVE_VALUES"  // One or more macro values negative
    ErrCodeNaNInput       = "NAN_INPUT"        // Input contains NaN
    ErrCodeInfInput       = "INF_INPUT"        // Input contains infinity
)
```

### 2. Logic & Algorithms (Step-by-Step)

#### 2.1 NormalizeMacros - Single Item Normalization

**Purpose:** Convert raw macronutrient values into a unit vector.

**Algorithm:**

1. **Validate Input**
   - Check if any value is NaN → return error `ErrCodeNaNInput`
   - Check if any value is Inf → return error `ErrCodeInfInput`
   - Check if any value is negative → return error `ErrCodeNegativeValues`

2. **Calculate Magnitude**
   ```
   magnitude = sqrt(protein^2 + carbs^2 + fat^2)
   ```

3. **Handle Zero Vector**
   - If magnitude == 0 (all macros are zero) → return error `ErrCodeZeroVector`
   - Note: A zero vector cannot be normalized and has no direction

4. **Normalize Components**
   ```
   P = protein / magnitude
   C = carbs / magnitude
   F = fat / magnitude
   ```

5. **Return Result**
   - Return MacroVector with P, C, F components and original magnitude

#### 2.2 AggregateRecipeMacros - Recipe Aggregation

**Purpose:** Calculate total macronutrients for a recipe by summing scaled ingredient contributions.

**Algorithm:**

1. **Initialize Accumulators**
   ```
   totalProtein = 0.0
   totalCarbs = 0.0
   totalFat = 0.0
   ```

2. **Iterate Over Ingredients**
   - For each ingredient in the recipe:
     ```
     scaleFactor = ingredient.Quantity / 100.0
     totalProtein += ingredient.Macros.Protein * scaleFactor
     totalCarbs += ingredient.Macros.Carbs * scaleFactor
     totalFat += ingredient.Macros.Fat * scaleFactor
     ```

3. **Return Aggregated Values**
   - Return MacroValues with totalProtein, totalCarbs, totalFat

#### 2.3 NormalizeRecipe - Full Recipe Normalization

**Purpose:** Aggregate recipe ingredients and normalize to unit vector.

**Algorithm:**

1. **Validate Ingredients List**
   - If ingredients list is empty → return error (no direction can be computed)

2. **Aggregate Macros**
   - Call `AggregateRecipeMacros(ingredients)`

3. **Normalize Aggregated Values**
   - Call `NormalizeMacros(aggregatedMacros)`
   - Propagate any errors from normalization

4. **Return Result**
   - Return the normalized MacroVector

#### 2.4 Numerical Stability Considerations

- Use `math.IsNaN()` and `math.IsInf()` for input validation
- Magnitude calculation uses standard Euclidean norm
- Division by magnitude is safe after zero-check
- No special handling for very small magnitudes (they still represent valid directions)

### 3. State Management & Error Handling

#### 3.1 Error States

| Error Condition | Error Code | Cause | Handling |
|----------------|------------|-------|----------|
| All macros are zero | `ZERO_VECTOR` | Food item with no macronutrients (e.g., pure water) | Cannot compute similarity; exclude from comparison |
| Negative macro value | `NEGATIVE_VALUES` | Data corruption or invalid input | Reject input; log for investigation |
| NaN in input | `NAN_INPUT` | Computational error upstream | Reject input; log for investigation |
| Infinity in input | `INF_INPUT` | Overflow upstream | Reject input; log for investigation |
| Empty recipe | `EMPTY_RECIPE` | Recipe with no ingredients | Cannot compute similarity; exclude from comparison |

#### 3.2 State Transitions

The normalizer is stateless. Each call is independent:

```
Input → Validation → Calculation → Output/Error
```

No internal state is maintained between calls.

#### 3.3 Error Propagation

- Errors are returned as structured `NormalizationError` values
- Callers (CosineSimilarityCalculator) must handle errors appropriately
- Zero-vector items should be filtered out before similarity comparison

### 4. Component Interfaces

```go
// MacroVectorNormalizer provides normalization services for macro vectors
type MacroVectorNormalizer interface {
    // NormalizeMacros converts raw macro values to a unit vector
    // Returns error if input is invalid or zero vector
    NormalizeMacros(macros MacroValues) (MacroVector, error)

    // AggregateRecipeMacros sums scaled ingredient macros
    // Returns total macros for the recipe (not normalized)
    AggregateRecipeMacros(ingredients []RecipeIngredient) MacroValues

    // NormalizeRecipe aggregates and normalizes recipe ingredients
    // Returns error if recipe is empty or aggregated macros are zero
    NormalizeRecipe(ingredients []RecipeIngredient) (MacroVector, error)
}

// Error implements the error interface for NormalizationError
func (e *NormalizationError) Error() string
```

#### 4.1 Function Signatures

```go
// NewMacroVectorNormalizer creates a new normalizer instance
func NewMacroVectorNormalizer() MacroVectorNormalizer

// NormalizeMacros normalizes a single set of macro values
// Parameters:
//   - macros: raw macronutrient values (per 100g/100ml)
// Returns:
//   - MacroVector: normalized unit vector with P, C, F components
//   - error: NormalizationError if validation fails or zero vector
func (n *macroVectorNormalizer) NormalizeMacros(macros MacroValues) (MacroVector, error)

// AggregateRecipeMacros calculates total macros for a recipe
// Parameters:
//   - ingredients: list of recipe ingredients with quantities
// Returns:
//   - MacroValues: summed macros for entire recipe
func (n *macroVectorNormalizer) AggregateRecipeMacros(ingredients []RecipeIngredient) MacroValues

// NormalizeRecipe aggregates and normalizes recipe ingredients
// Parameters:
//   - ingredients: list of recipe ingredients with quantities
// Returns:
//   - MacroVector: normalized unit vector for the recipe
//   - error: NormalizationError if empty or zero vector
func (n *macroVectorNormalizer) NormalizeRecipe(ingredients []RecipeIngredient) (MacroVector, error)
```

#### 4.2 Usage Example (Pseudocode)

```
// Single food item
chickenMacros := MacroValues{Protein: 31.0, Carbs: 0.0, Fat: 3.6}
normalizer := NewMacroVectorNormalizer()
vector, err := normalizer.NormalizeMacros(chickenMacros)
// vector.P ≈ 0.993, vector.C = 0.0, vector.F ≈ 0.115

// Recipe
ingredients := []RecipeIngredient{
    {Macros: chickenMacros, Quantity: 200.0},
    {Macros: MacroValues{Protein: 13.0, Carbs: 71.0, Fat: 1.5}, Quantity: 150.0}, // rice
}
recipeVector, err := normalizer.NormalizeRecipe(ingredients)
```
