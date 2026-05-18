## FILE: DESIGN-003.md
**Traceability:** ARCH-003

**Static aspects covered:** CosineSimilarityCalculator, MacroVectorNormalizer, ThresholdFilter, SimilarityIndicatorMapper, SimilarityAssetResolver.

### 0. Static Aspect Responsibilities
- `CosineSimilarityCalculator`: owns dot-product similarity calculation between normalized macro vectors.
- `MacroVectorNormalizer`: owns zero-vector detection, micronutrient exclusion, and unit-vector conversion.
- `ThresholdFilter`: owns the minimum 0.40 score gate and skipped-target diagnostics.
- `SimilarityIndicatorMapper`: owns score-to-tier and score-to-color mapping.
- `SimilarityAssetResolver`: owns server-hosted indicator image URL selection and fallback asset behavior.

### 1. Data Structures & Types
- `interface MacroVector { protein: number; carbs: number; fat: number }`
- `interface NormalizedMacroVector { protein: number; carbs: number; fat: number; magnitude: number }`
- `type MatchType = "calorie" | "protein"`
- `type SimilarityTier = "excellent" | "good" | "fair" | "poor"`
- `interface ComparisonRequest { sourceItem: MacroVector; targetItems: TargetMacroVector[]; matchType: MatchType }`
- `interface TargetMacroVector { itemId: string; macros: MacroVector; caloriesPerBaseUnit: number; proteinPerBaseUnit: number }`
- `interface SimilarityResult { itemId: string; score: number; tier: SimilarityTier; matchingQuantity: number; colorHex: string; imageUrl: string }`
- `interface TierRule { tier: SimilarityTier; minScore: number; maxScore: number; colorHex: string; imageUrl: string }`

### 2. Logic & Algorithms (Step-by-Step)
1. Reject source vectors with all macro values equal to zero because cosine direction cannot be calculated.
2. Discard any micronutrient data supplied by callers; only protein, carbohydrates, and fat may enter `MacroVector`.
3. Normalize the source vector to unit length: divide each macro by `sqrt(p^2 + c^2 + f^2)`.
4. For recipe targets, request aggregate macros from ARCH-005 before normalization.
5. Normalize each target vector using the same formula; skip zero-magnitude targets.
6. Compute cosine similarity as `source.p*target.p + source.c*target.c + source.f*target.f`.
7. Filter out scores below `0.40`.
8. Map remaining scores to tier rules: `>=0.85 excellent`, `>=0.70 good`, `>=0.55 fair`, otherwise `poor`.
9. Calculate replacement quantity from `matchType`: calorie matching uses source calories divided by target calories per base unit; protein matching uses source protein divided by target protein per base unit.
10. Return results sorted by score descending, preserving item IDs for repository hydration by callers.

### 3. State Management & Error Handling
- `valid`: source and target vectors contain non-negative finite macro values.
- `zero_source_vector`: return validation error.
- `micronutrients_present`: ignore micronutrient fields and continue with macronutrients only.
- `zero_target_vector`: omit target and add diagnostic warning.
- `below_threshold`: omit target from response.
- `quantity_not_calculable`: return score but set `matchingQuantity = 0` when denominator is zero.
- `repository_error`: bubble up when recipe aggregation cannot be loaded.
- `asset_missing`: return tier color and fallback indicator URL from static assets.

### 4. Component Interfaces
- `func NormalizeMacroVector(v MacroVector) (NormalizedMacroVector, error)`
- `func CosineSimilarity(a NormalizedMacroVector, b NormalizedMacroVector) float64`
- `func CompareMacros(req ComparisonRequest) ([]SimilarityResult, error)`
- `func FilterByThreshold(results []SimilarityResult, minScore float64) []SimilarityResult`
- `func MapSimilarityTier(score float64) TierRule`
- `func ResolveIndicatorAsset(tier SimilarityTier) (colorHex string, imageURL string)`
- `func CalculateMatchingQuantity(source MacroVector, target TargetMacroVector, matchType MatchType) float64`
