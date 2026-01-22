## FILE: CosineSimilarityCalculator.md
**Traceability:** ARCH-003

### 1. Data Structures & Types

```go
// MacroVector represents a food item's macronutrient profile in grams per 100g
type MacroVector struct {
    ItemID   string  // Unique identifier of the food item
    Protein  float64 // Protein in grams per 100g
    Carbs    float64 // Carbohydrates in grams per 100g
    Fat      float64 // Fat in grams per 100g
    Calories float64 // Total calories per 100g (for quantity matching)
}

// NormalizedVector represents a unit vector in 3D macro space
type NormalizedVector struct {
    ItemID    string  // Preserved from source MacroVector
    P         float64 // Normalized protein component
    C         float64 // Normalized carbohydrates component
    F         float64 // Normalized fat component
    Magnitude float64 // Original vector magnitude (for denormalization)
}

// SimilarityScore holds the raw cosine similarity between two items
type SimilarityScore struct {
    TargetItemID string  // ID of the compared item
    Score        float64 // Cosine similarity value (0.0 to 1.0)
}

// ComparisonRequest is the input for similarity calculations
type ComparisonRequest struct {
    SourceItem  MacroVector   // The item to find replacements for
    TargetItems []MacroVector // Candidate replacement items
    MatchType   MatchType     // How to calculate matching quantity
}

// MatchType determines quantity matching strategy
type MatchType string

const (
    MatchTypeCalorie MatchType = "calorie" // Match total calories
    MatchTypeProtein MatchType = "protein" // Match protein content
)

// SimilarityResult represents a single comparison result
type SimilarityResult struct {
    ItemID           string  // Target item identifier
    Score            float64 // Cosine similarity (0.0 to 1.0)
    ScorePercent     int     // Score as percentage (0 to 100)
    MatchingQuantity float64 // Grams needed to match source (per MatchType)
}

// CalculatorConfig holds configuration for the calculator
type CalculatorConfig struct {
    MinThreshold     float64 // Minimum similarity score to include (default: 0.40)
    ZeroVectorPolicy ZeroVectorPolicy // How to handle zero vectors
}

// ZeroVectorPolicy defines behavior for zero-magnitude vectors
type ZeroVectorPolicy int

const (
    ZeroVectorExclude ZeroVectorPolicy = iota // Exclude from results
    ZeroVectorZeroScore                       // Include with 0.0 score
)
```

### 2. Logic & Algorithms (Step-by-Step)

#### 2.1 Main Calculation Flow

```
FUNCTION Calculate(request ComparisonRequest, config CalculatorConfig) -> []SimilarityResult:
    1. INPUT VALIDATION
       - IF request.SourceItem has all zero macros:
           IF config.ZeroVectorPolicy == ZeroVectorExclude:
               RETURN empty []SimilarityResult
           ELSE:
               RETURN all targets with Score = 0.0
       - IF request.TargetItems is empty:
           RETURN empty []SimilarityResult

    2. NORMALIZE SOURCE VECTOR
       - sourceNorm = NormalizeVector(request.SourceItem)
       - Store sourceNorm.Magnitude for quantity calculations

    3. CALCULATE SIMILARITY FOR EACH TARGET
       - results = empty []SimilarityResult
       - FOR each target in request.TargetItems:
           a. Normalize target vector:
              targetNorm = NormalizeVector(target)

           b. Handle zero vector case:
              IF targetNorm.Magnitude == 0:
                  IF config.ZeroVectorPolicy == ZeroVectorExclude:
                      CONTINUE to next target
                  ELSE:
                      score = 0.0
              ELSE:
                  score = DotProduct(sourceNorm, targetNorm)

           c. Apply threshold filter:
              IF score < config.MinThreshold:
                  CONTINUE to next target

           d. Calculate matching quantity:
              matchQty = CalculateMatchingQuantity(
                  request.SourceItem,
                  target,
                  request.MatchType
              )

           e. Create result:
              result = SimilarityResult{
                  ItemID:           target.ItemID,
                  Score:            score,
                  ScorePercent:     int(math.Round(score * 100)),
                  MatchingQuantity: matchQty,
              }
              APPEND result to results

    4. SORT RESULTS
       - Sort results by Score DESC (highest similarity first)
       - IF scores are equal, sort by ItemID ASC for determinism

    5. RETURN results
```

#### 2.2 Vector Normalization Algorithm

```
FUNCTION NormalizeVector(macro MacroVector) -> NormalizedVector:
    // Convert macronutrient values to a unit vector (magnitude = 1)
    // This ensures similarity measures directional alignment, not magnitude

    1. CALCULATE MAGNITUDE (Euclidean norm)
       magnitude = sqrt(macro.Protein^2 + macro.Carbs^2 + macro.Fat^2)

    2. HANDLE ZERO VECTOR
       IF magnitude == 0:
           RETURN NormalizedVector{
               ItemID:    macro.ItemID,
               P:         0,
               C:         0,
               F:         0,
               Magnitude: 0,
           }

    3. NORMALIZE COMPONENTS
       RETURN NormalizedVector{
           ItemID:    macro.ItemID,
           P:         macro.Protein / magnitude,
           C:         macro.Carbs / magnitude,
           F:         macro.Fat / magnitude,
           Magnitude: magnitude,
       }
```

#### 2.3 Cosine Similarity (Dot Product of Unit Vectors)

```
FUNCTION DotProduct(a NormalizedVector, b NormalizedVector) -> float64:
    // For unit vectors, dot product equals cosine of angle between them
    // Result range: -1.0 to 1.0 (but macros are non-negative, so 0.0 to 1.0)

    1. CALCULATE DOT PRODUCT
       dot = (a.P * b.P) + (a.C * b.C) + (a.F * b.F)

    2. CLAMP RESULT
       // Handle floating-point precision errors
       IF dot > 1.0:
           dot = 1.0
       IF dot < 0.0:
           dot = 0.0

    3. RETURN dot
```

#### 2.4 Matching Quantity Calculation

```
FUNCTION CalculateMatchingQuantity(
    source MacroVector,
    target MacroVector,
    matchType MatchType
) -> float64:
    // Calculate how many grams of target equals source (by calorie or protein)
    // All values are per 100g, so ratio gives us the quantity

    1. DETERMINE MATCHING METRIC
       SWITCH matchType:
           CASE MatchTypeCalorie:
               sourceValue = source.Calories
               targetValue = target.Calories
           CASE MatchTypeProtein:
               sourceValue = source.Protein
               targetValue = target.Protein

    2. HANDLE EDGE CASES
       IF targetValue == 0:
           // Cannot match if target has zero of the matching metric
           RETURN 0
       IF sourceValue == 0:
           // Source has zero of the metric; any quantity "matches"
           RETURN 100 // Return standard 100g serving

    3. CALCULATE RATIO
       // How many grams of target to match 100g of source?
       ratio = sourceValue / targetValue
       matchingQuantity = 100 * ratio

    4. ROUND TO PRECISION
       // Round to 1 decimal place for practical use
       matchingQuantity = math.Round(matchingQuantity * 10) / 10

    5. RETURN matchingQuantity
```

#### 2.5 Batch Processing for Performance

```
FUNCTION CalculateBatch(
    source MacroVector,
    targets []MacroVector,
    matchType MatchType,
    config CalculatorConfig,
) -> []SimilarityResult:
    // Optimized batch processing - normalize source once

    1. NORMALIZE SOURCE ONCE
       sourceNorm = NormalizeVector(source)
       IF sourceNorm.Magnitude == 0 AND config.ZeroVectorPolicy == ZeroVectorExclude:
           RETURN empty []SimilarityResult

    2. PRE-ALLOCATE RESULTS
       // Allocate with capacity to avoid reallocation
       results = make([]SimilarityResult, 0, len(targets))

    3. PROCESS TARGETS
       FOR each target in targets:
           targetNorm = NormalizeVector(target)

           IF targetNorm.Magnitude == 0:
               IF config.ZeroVectorPolicy == ZeroVectorExclude:
                   CONTINUE
               score = 0.0
           ELSE IF sourceNorm.Magnitude == 0:
               score = 0.0
           ELSE:
               score = DotProduct(sourceNorm, targetNorm)

           IF score >= config.MinThreshold:
               matchQty = CalculateMatchingQuantity(source, target, matchType)
               APPEND SimilarityResult{
                   ItemID:           target.ItemID,
                   Score:            score,
                   ScorePercent:     int(math.Round(score * 100)),
                   MatchingQuantity: matchQty,
               } to results

    4. SORT BY SCORE DESC
       sort.Slice(results, func(i, j int) bool {
           if results[i].Score != results[j].Score {
               return results[i].Score > results[j].Score
           }
           return results[i].ItemID < results[j].ItemID
       })

    5. RETURN results
```

### 3. State Management & Error Handling

#### 3.1 Error States

| Error State | Cause | Detection | Response |
|:------------|:------|:----------|:---------|
| Zero Source Vector | Source item has P=0, C=0, F=0 | magnitude == 0 after normalization | Return empty results or all zeros (per ZeroVectorPolicy) |
| Zero Target Vector | Target item has P=0, C=0, F=0 | magnitude == 0 after normalization | Exclude or include with 0.0 score (per ZeroVectorPolicy) |
| Empty Target List | No items to compare against | len(targets) == 0 | Return empty []SimilarityResult |
| Negative Macro Values | Invalid input data | Any macro value < 0 | Return error: ErrInvalidMacroValue |
| Division by Zero (Quantity) | Target has 0 calories/protein | targetValue == 0 in quantity calc | Return 0 for MatchingQuantity |
| NaN/Inf Result | Floating-point edge cases | math.IsNaN or math.IsInf | Treat as 0.0 score |

#### 3.2 Input Validation

```
FUNCTION ValidateMacroVector(v MacroVector) -> error:
    IF v.Protein < 0:
        RETURN ErrInvalidMacroValue("protein cannot be negative")
    IF v.Carbs < 0:
        RETURN ErrInvalidMacroValue("carbs cannot be negative")
    IF v.Fat < 0:
        RETURN ErrInvalidMacroValue("fat cannot be negative")
    IF v.Calories < 0:
        RETURN ErrInvalidMacroValue("calories cannot be negative")
    IF v.ItemID == "":
        RETURN ErrInvalidMacroValue("itemID cannot be empty")
    RETURN nil
```

#### 3.3 Error Definitions

```go
var (
    ErrInvalidMacroValue = errors.New("invalid macro value")
    ErrEmptyItemID       = errors.New("item ID cannot be empty")
)
```

#### 3.4 Calculation State Flow

```
                    ┌─────────────────┐
                    │ RECEIVE_REQUEST │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ VALIDATE_INPUT  │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │ Invalid      │ Valid        │
              ▼              ▼              │
    ┌─────────────────┐ ┌─────────────────┐│
    │  RETURN_ERROR   │ │NORMALIZE_SOURCE ││
    └─────────────────┘ └────────┬────────┘│
                                 │         │
              ┌──────────────────┼─────────┤
              │ Zero Vector      │ Valid   │
              ▼                  ▼         │
    ┌─────────────────┐ ┌─────────────────┐│
    │ HANDLE_ZERO_SRC │ │ ITERATE_TARGETS ││
    └────────┬────────┘ └────────┬────────┘│
             │                   │         │
             │          ┌────────┼─────────┤
             │          │ For each target  │
             │          ▼                  │
             │  ┌─────────────────┐        │
             │  │NORMALIZE_TARGET │        │
             │  └────────┬────────┘        │
             │           │                 │
             │  ┌────────┼─────────┐       │
             │  │ Zero   │ Valid   │       │
             │  ▼        ▼         │       │
             │  ┌────────┐ ┌───────┴─────┐ │
             │  │SKIP/0  │ │CALC_DOT_PROD│ │
             │  └────────┘ └───────┬─────┘ │
             │                     │       │
             │         ┌───────────┼───────┤
             │         │ < threshold│ >= threshold
             │         ▼           ▼       │
             │  ┌──────────┐ ┌───────────┐ │
             │  │  SKIP    │ │CALC_QTY   │ │
             │  └──────────┘ └─────┬─────┘ │
             │                     │       │
             │                     ▼       │
             │             ┌───────────┐   │
             │             │ADD_RESULT │   │
             │             └─────┬─────┘   │
             │                   │         │
             └───────────────────┼─────────┘
                                 ▼
                        ┌─────────────────┐
                        │  SORT_RESULTS   │
                        └────────┬────────┘
                                 │
                                 ▼
                        ┌─────────────────┐
                        │ RETURN_RESULTS  │
                        └─────────────────┘
```

#### 3.5 Performance Constraints

| Constraint | Target | Enforcement |
|:-----------|:-------|:------------|
| Single comparison | < 1ms | O(1) vector operations |
| Batch of 1000 items | < 50ms | Pre-allocated slices, single-pass sort |
| Memory per comparison | < 1KB | No intermediate allocations |
| Floating-point precision | 15 significant digits | float64 type |

### 4. Component Interfaces

#### 4.1 Public Interface

```go
// CosineSimilarityCalculator computes nutritional similarity between food items
type CosineSimilarityCalculator interface {
    // Calculate computes similarity scores between source and all target items.
    // Results are filtered by threshold and sorted by score descending.
    // Parameters:
    //   - request: Contains source item, target items, and match type
    // Returns:
    //   - Slice of SimilarityResult (may be empty if no items pass threshold)
    //   - error if input validation fails
    Calculate(request ComparisonRequest) ([]SimilarityResult, error)

    // CalculateSingle computes similarity between exactly two items.
    // Does not apply threshold filtering.
    // Parameters:
    //   - source: The reference food item
    //   - target: The item to compare against
    //   - matchType: How to calculate matching quantity
    // Returns:
    //   - Single SimilarityResult
    //   - error if input validation fails
    CalculateSingle(source, target MacroVector, matchType MatchType) (SimilarityResult, error)
}
```

#### 4.2 Constructor

```go
// NewCosineSimilarityCalculator creates a new calculator with the given configuration.
// Parameters:
//   - config: Calculator configuration (threshold, zero-vector policy)
// Returns:
//   - CosineSimilarityCalculator implementation
func NewCosineSimilarityCalculator(config CalculatorConfig) CosineSimilarityCalculator
```

#### 4.3 Internal Functions

```go
// normalizeVector converts a MacroVector to a unit NormalizedVector.
// Parameters:
//   - v: MacroVector with raw gram values
// Returns:
//   - NormalizedVector with magnitude 1 (or zero vector)
func normalizeVector(v MacroVector) NormalizedVector

// dotProduct computes the dot product of two normalized vectors.
// For unit vectors, this equals the cosine of the angle between them.
// Parameters:
//   - a: First normalized vector
//   - b: Second normalized vector
// Returns:
//   - Cosine similarity (0.0 to 1.0 for non-negative macro values)
func dotProduct(a, b NormalizedVector) float64

// calculateMatchingQuantity determines grams of target to match source.
// Parameters:
//   - source: Reference food item
//   - target: Replacement food item
//   - matchType: Whether to match by calorie or protein
// Returns:
//   - Grams of target needed (rounded to 1 decimal place)
func calculateMatchingQuantity(source, target MacroVector, matchType MatchType) float64

// validateMacroVector checks that a MacroVector has valid values.
// Parameters:
//   - v: MacroVector to validate
// Returns:
//   - error if any value is invalid, nil otherwise
func validateMacroVector(v MacroVector) error

// clampScore ensures score is within valid range [0.0, 1.0].
// Handles floating-point precision errors.
// Parameters:
//   - score: Raw dot product result
// Returns:
//   - Clamped score value
func clampScore(score float64) float64
```

#### 4.4 Default Configuration

```go
var DefaultCalculatorConfig = CalculatorConfig{
    MinThreshold:     0.40,              // 40% minimum similarity
    ZeroVectorPolicy: ZeroVectorExclude, // Exclude zero-macro items
}
```

#### 4.5 Mathematical Reference

**Cosine Similarity Formula:**
```
cos(θ) = (A · B) / (||A|| × ||B||)

Where:
  A · B = (A.P × B.P) + (A.C × B.C) + (A.F × B.F)  [dot product]
  ||A|| = sqrt(A.P² + A.C² + A.F²)                  [magnitude]
```

**For unit vectors (pre-normalized):**
```
cos(θ) = A · B
```

**Example Calculation:**
```
Source: Chicken breast (31g protein, 0g carbs, 3.6g fat)
Target: Greek yogurt (10g protein, 3.6g carbs, 0.7g fat)

Step 1 - Normalize source:
  magnitude_s = sqrt(31² + 0² + 3.6²) = sqrt(961 + 12.96) = 31.21
  normalized_s = (0.993, 0, 0.115)

Step 2 - Normalize target:
  magnitude_t = sqrt(10² + 3.6² + 0.7²) = sqrt(100 + 12.96 + 0.49) = 10.65
  normalized_t = (0.939, 0.338, 0.066)

Step 3 - Dot product:
  similarity = (0.993 × 0.939) + (0 × 0.338) + (0.115 × 0.066)
             = 0.932 + 0 + 0.008
             = 0.940 (94% similarity)
```
