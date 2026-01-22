## FILE: SimilarityIndicatorMapper.md

**Traceability:** ARCH-003, SW-REQ-018

### 1. Data Structures & Types

```go
// SimilarityTier represents the quality tier of a food similarity match
type SimilarityTier string

const (
    TierExcellent SimilarityTier = "excellent"
    TierGood      SimilarityTier = "good"
    TierFair      SimilarityTier = "fair"
    TierPoor      SimilarityTier = "poor"
)

// SimilarityIndicator contains the visual representation data for a similarity tier
type SimilarityIndicator struct {
    Tier     SimilarityTier `json:"tier"`
    ColorHex string         `json:"colorHex"`
    ImageURL string         `json:"imageUrl"`
}

// tierConfig holds the threshold boundaries and indicator data for each tier
type tierConfig struct {
    tier     SimilarityTier
    minScore float64 // inclusive lower bound
    maxScore float64 // exclusive upper bound (except for top tier)
    colorHex string
    imageURL string
}
```

**Configuration Constants:**

```go
const (
    // Color codes for each tier
    ColorExcellent = "#22C55E" // Green
    ColorGood      = "#84CC16" // Light Green
    ColorFair      = "#EAB308" // Yellow
    ColorPoor      = "#EF4444" // Red

    // Asset paths (server-hosted)
    AssetPathExcellent = "/assets/indicators/star.png"
    AssetPathGood      = "/assets/indicators/sparkle.png"
    AssetPathFair      = "/assets/indicators/thumbs-up.png"
    AssetPathPoor      = "/assets/indicators/thumbs-down.png"

    // Score thresholds (as decimals, not percentages)
    ThresholdExcellent = 0.85
    ThresholdGood      = 0.70
    ThresholdFair      = 0.55
)
```

### 2. Logic & Algorithms (Step-by-Step)

**MapScoreToIndicator(score float64) SimilarityIndicator**

1. Validate input score is within valid range [0.0, 1.0]
2. Compare score against threshold boundaries in descending order:
   - If score >= 0.85: return Excellent indicator
   - If score >= 0.70: return Good indicator
   - If score >= 0.55: return Fair indicator
   - Otherwise: return Poor indicator
3. Return SimilarityIndicator with tier, colorHex, and imageURL

```
FUNCTION MapScoreToIndicator(score: float64) -> SimilarityIndicator:
    IF score < 0.0 THEN
        score = 0.0
    END IF
    IF score > 1.0 THEN
        score = 1.0
    END IF

    IF score >= ThresholdExcellent THEN
        RETURN SimilarityIndicator{
            Tier:     TierExcellent,
            ColorHex: ColorExcellent,
            ImageURL: AssetPathExcellent
        }
    ELSE IF score >= ThresholdGood THEN
        RETURN SimilarityIndicator{
            Tier:     TierGood,
            ColorHex: ColorGood,
            ImageURL: AssetPathGood
        }
    ELSE IF score >= ThresholdFair THEN
        RETURN SimilarityIndicator{
            Tier:     TierFair,
            ColorHex: ColorFair,
            ImageURL: AssetPathFair
        }
    ELSE
        RETURN SimilarityIndicator{
            Tier:     TierPoor,
            ColorHex: ColorPoor,
            ImageURL: AssetPathPoor
        }
    END IF
END FUNCTION
```

**GetTierForScore(score float64) SimilarityTier**

1. Delegate to MapScoreToIndicator
2. Return only the Tier field

```
FUNCTION GetTierForScore(score: float64) -> SimilarityTier:
    indicator = MapScoreToIndicator(score)
    RETURN indicator.Tier
END FUNCTION
```

**GetAllTierConfigs() []tierConfig**

Returns the complete tier configuration table for reference or validation purposes.

```
FUNCTION GetAllTierConfigs() -> []tierConfig:
    RETURN [
        {TierExcellent, 0.85, 1.00, ColorExcellent, AssetPathExcellent},
        {TierGood,      0.70, 0.85, ColorGood,      AssetPathGood},
        {TierFair,      0.55, 0.70, ColorFair,      AssetPathFair},
        {TierPoor,      0.00, 0.55, ColorPoor,      AssetPathPoor}
    ]
END FUNCTION
```

### 3. State Management & Error Handling

**Error States:**

| Error Condition | Handling Strategy |
| :--- | :--- |
| Score < 0.0 | Clamp to 0.0, return Poor indicator |
| Score > 1.0 | Clamp to 1.0, return Excellent indicator |
| NaN score | Treat as 0.0, return Poor indicator |

**State Transitions:**

This component is stateless. Each call to MapScoreToIndicator is a pure function with no side effects. The mapping is deterministic: the same input score always produces the same output indicator.

**Boundary Conditions:**

| Score Value | Expected Tier |
| :--- | :--- |
| 0.00 | Poor |
| 0.54 | Poor |
| 0.55 | Fair |
| 0.69 | Fair |
| 0.70 | Good |
| 0.84 | Good |
| 0.85 | Excellent |
| 1.00 | Excellent |

### 4. Component Interfaces

```go
// SimilarityIndicatorMapper provides tier mapping for similarity scores
type SimilarityIndicatorMapper interface {
    // MapScoreToIndicator converts a similarity score [0.0, 1.0] to a visual indicator
    // Scores outside valid range are clamped to boundaries
    MapScoreToIndicator(score float64) SimilarityIndicator

    // GetTierForScore returns only the tier classification for a given score
    GetTierForScore(score float64) SimilarityTier

    // GetAllTierConfigs returns the complete tier configuration table
    GetAllTierConfigs() []tierConfig
}

// NewSimilarityIndicatorMapper creates a new mapper instance with default configuration
func NewSimilarityIndicatorMapper() SimilarityIndicatorMapper
```

**Usage Example:**

```go
mapper := NewSimilarityIndicatorMapper()

// Get full indicator for display
indicator := mapper.MapScoreToIndicator(0.78)
// Returns: SimilarityIndicator{Tier: "good", ColorHex: "#84CC16", ImageURL: "/assets/indicators/sparkle.png"}

// Get just the tier for filtering/sorting
tier := mapper.GetTierForScore(0.92)
// Returns: TierExcellent
```

**Integration with SimilarityResult:**

The mapper is called by the Similarity Engine after cosine similarity calculation:

```go
type SimilarityResult struct {
    ItemID           string              `json:"itemId"`
    Score            float64             `json:"score"`
    Tier             SimilarityTier      `json:"tier"`
    Indicator        SimilarityIndicator `json:"indicator"`
    MatchingQuantity float64             `json:"matchingQuantity"`
}
```
