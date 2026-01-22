# [ARCH-003] - Similarity Engine

**Description:** Core computational service that calculates cosine similarity between food items based on macronutrient vectors (Protein, Carbohydrates, Fat), applies threshold filtering, and provides visual indicator mappings.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service |
| **Static Aspects** | CosineSimilarityCalculator, MacroVectorNormalizer, ThresholdFilter, SimilarityIndicatorMapper, SimilarityAssetResolver |
| **Dependencies** | ARCH-005 (Data Repository) |
| **Traceability** | SW-REQ-016, SW-REQ-017, SW-REQ-018, SW-REQ-026, SW-REQ-027, SW-REQ-028 |

**Dynamic Behavior:**

- **Vector Calculation:** Normalizes macronutrient values to unit vectors. For recipes, aggregates constituent ingredient macros before normalization.
- **Similarity Scoring:** Computes cosine similarity using dot product of normalized vectors. Filters results below 0.40 threshold.
- **Visual Indicator Mapping (SW-REQ-018):** Assigns tier indicators based on score thresholds. Returns both color code and server-hosted image URL for the indicator icon. Indicator images are stored as static assets on the server (not client-side Unicode emojis) to ensure consistent cross-platform rendering.
  - Green + `/assets/indicators/star.png` for >=85%
  - Light Green + `/assets/indicators/sparkle.png` for 70-84%
  - Yellow + `/assets/indicators/thumbs-up.png` for 55-69%
  - Red + `/assets/indicators/thumbs-down.png` for <55%
- **Quantity Matching:** Calculates replacement quantities to match original calorie or protein counts.

**Interface Definition:**

- `Input`: ComparisonRequest { sourceItem: MacroVector, targetItems: MacroVector[], matchType: 'calorie' | 'protein' }
- `Output`: SimilarityResult { itemId: string, score: number, tier: SimilarityTier, matchingQuantity: number }[]

**Alternative Analysis (BP6):**

- *Chosen Approach:* Three-dimensional cosine similarity (P, C, F)
- *Alternative Considered:* Euclidean distance in macro space, or weighted similarity including calories
- *Trade-off:* Cosine similarity measures directional alignment of macro ratios regardless of magnitude, which aligns with nutritional replacement goals (same macro profile at any quantity). Euclidean would penalize magnitude differences inappropriately. Adding calories as 4th dimension would over-weight it since calories derive from macros.
