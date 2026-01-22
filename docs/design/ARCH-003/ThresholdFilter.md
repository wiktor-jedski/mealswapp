## FILE: ThresholdFilter.md
**Traceability:** ARCH-003

### 1. Data Structures & Types

```go
// MinimumSimilarityThreshold defines the minimum acceptable similarity score.
// Results below this threshold are considered irrelevant and filtered out.
const MinimumSimilarityThreshold float64 = 0.40

// SimilarityResult represents a single similarity comparison result.
// This is the input/output type for the ThresholdFilter.
type SimilarityResult struct {
    ItemID           string        // Unique identifier of the compared food item
    Score            float64       // Cosine similarity score [0.0, 1.0]
    Tier             SimilarityTier // Visual indicator tier assignment
    MatchingQuantity float64       // Calculated replacement quantity (grams)
}

// SimilarityTier represents the visual indicator tier for a similarity score.
type SimilarityTier string

const (
    TierExcellent SimilarityTier = "excellent" // >= 0.85
    TierGood      SimilarityTier = "good"      // 0.70 - 0.84
    TierFair      SimilarityTier = "fair"      // 0.55 - 0.69
    TierPoor      SimilarityTier = "poor"      // 0.40 - 0.54
)

// FilterConfig holds configuration for the threshold filter.
type FilterConfig struct {
    MinThreshold float64 // Minimum similarity score to include (default: 0.40)
}
```

### 2. Logic & Algorithms (Step-by-Step)

#### Filter Function

```
FUNCTION Filter(results []SimilarityResult, config FilterConfig) []SimilarityResult

    1. INPUT VALIDATION
       1.1. IF results is nil or empty:
            RETURN empty slice []SimilarityResult{}
       1.2. IF config.MinThreshold < 0.0:
            SET config.MinThreshold = MinimumSimilarityThreshold (0.40)
       1.3. IF config.MinThreshold > 1.0:
            SET config.MinThreshold = 1.0

    2. ALLOCATE OUTPUT
       2.1. CREATE filtered slice with capacity = len(results)
            (Pre-allocate to avoid repeated memory allocations)

    3. ITERATE AND FILTER
       3.1. FOR EACH result IN results:
            3.1.1. IF result.Score >= config.MinThreshold:
                   APPEND result to filtered slice

    4. RETURN filtered slice

END FUNCTION
```

#### FilterWithDefaultThreshold Function

```
FUNCTION FilterWithDefaultThreshold(results []SimilarityResult) []SimilarityResult

    1. CREATE config with MinThreshold = MinimumSimilarityThreshold (0.40)

    2. CALL Filter(results, config)

    3. RETURN result from Filter

END FUNCTION
```

#### IsAboveThreshold Function

```
FUNCTION IsAboveThreshold(score float64) bool

    1. RETURN score >= MinimumSimilarityThreshold

END FUNCTION
```

#### IsAboveCustomThreshold Function

```
FUNCTION IsAboveCustomThreshold(score float64, threshold float64) bool

    1. IF threshold < 0.0:
       SET threshold = 0.0

    2. IF threshold > 1.0:
       SET threshold = 1.0

    3. RETURN score >= threshold

END FUNCTION
```

### 3. State Management & Error Handling

#### Error States

| Error State | Condition | Handling |
|-------------|-----------|----------|
| Nil Input | `results` parameter is nil | Return empty slice `[]SimilarityResult{}` |
| Empty Input | `results` has length 0 | Return empty slice `[]SimilarityResult{}` |
| Invalid Threshold (Negative) | `config.MinThreshold < 0.0` | Clamp to `MinimumSimilarityThreshold` (0.40) |
| Invalid Threshold (>1.0) | `config.MinThreshold > 1.0` | Clamp to `1.0` |
| All Results Below Threshold | All scores < threshold | Return empty slice `[]SimilarityResult{}` |

#### State Transitions

```
                    ┌─────────────────┐
                    │   INPUT RECEIVED │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
            ┌───────│  VALIDATE INPUT │───────┐
            │       └─────────────────┘       │
            │                                  │
      nil/empty                          valid input
            │                                  │
            ▼                                  ▼
    ┌───────────────┐               ┌─────────────────┐
    │ RETURN EMPTY  │               │ VALIDATE CONFIG │
    └───────────────┘               └────────┬────────┘
                                             │
                                             ▼
                                    ┌─────────────────┐
                                    │  CLAMP THRESHOLD│
                                    │   IF INVALID    │
                                    └────────┬────────┘
                                             │
                                             ▼
                                    ┌─────────────────┐
                                    │ FILTER RESULTS  │
                                    └────────┬────────┘
                                             │
                                             ▼
                                    ┌─────────────────┐
                                    │ RETURN FILTERED │
                                    └─────────────────┘
```

#### Invariants

1. **Score Range**: All returned results have `Score >= config.MinThreshold`
2. **No Data Modification**: Input results are not mutated; new slice is returned
3. **Order Preservation**: Relative order of results is maintained in output
4. **Deterministic**: Same input always produces same output

### 4. Component Interfaces

```go
// ThresholdFilter provides methods to filter similarity results
// based on score thresholds.
type ThresholdFilter interface {
    // Filter removes results below the configured threshold.
    // Returns a new slice containing only results with Score >= config.MinThreshold.
    // If results is nil or empty, returns an empty slice.
    // If config.MinThreshold is invalid, it is clamped to valid range [0.0, 1.0].
    Filter(results []SimilarityResult, config FilterConfig) []SimilarityResult

    // FilterWithDefaultThreshold removes results below MinimumSimilarityThreshold (0.40).
    // Convenience method equivalent to Filter(results, FilterConfig{MinThreshold: 0.40}).
    FilterWithDefaultThreshold(results []SimilarityResult) []SimilarityResult

    // IsAboveThreshold checks if a single score meets the default minimum threshold.
    // Returns true if score >= MinimumSimilarityThreshold (0.40).
    IsAboveThreshold(score float64) bool

    // IsAboveCustomThreshold checks if a score meets a custom threshold.
    // Threshold is clamped to [0.0, 1.0] if out of range.
    IsAboveCustomThreshold(score float64, threshold float64) bool
}

// NewThresholdFilter creates a new instance of the threshold filter.
func NewThresholdFilter() ThresholdFilter
```

#### Function Signatures Summary

| Function | Input | Output |
|----------|-------|--------|
| `Filter` | `(results []SimilarityResult, config FilterConfig)` | `[]SimilarityResult` |
| `FilterWithDefaultThreshold` | `(results []SimilarityResult)` | `[]SimilarityResult` |
| `IsAboveThreshold` | `(score float64)` | `bool` |
| `IsAboveCustomThreshold` | `(score float64, threshold float64)` | `bool` |
| `NewThresholdFilter` | `()` | `ThresholdFilter` |
