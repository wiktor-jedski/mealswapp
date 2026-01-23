## Phase 3: Similarity Engine & Data Repository Completion

**Goal:** Implement cosine similarity calculation and complete data layer

### Components & Static Aspects

#### ARCH-003 - Similarity Engine
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **CosineSimilarityCalculator** | Dot product of normalized P/C/F vectors | `similarity/calculator.go` |
| **MacroVectorNormalizer** | Normalize macros to unit vectors | `similarity/normalizer.go` |
| **ThresholdFilter** | Exclude results with score < 0.40 | `similarity/threshold_filter.go` |
| **SimilarityIndicatorMapper** | Map scores to tiers (excellent/good/fair/poor) | `similarity/indicator_mapper.go` |
| **SimilarityAssetResolver** | Resolve tier to color hex and image URL | `similarity/asset_resolver.go` |

#### ARCH-005 (complete) - Data Repository Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **UnitConverter** | Metric<->Imperial (g->oz: ×0.035, ml->fl oz: ×0.033) | `services/unit_converter.go` |
| **MacroNormalizer** | Normalize all values to per 100g/100ml | `services/macro_normalizer.go` |
| **FoodItemRepository** | FoodItem CRUD with tag associations | `repository/food_item_repo.go` |
| **MealRepository** | Meal CRUD with recipe aggregation | `repository/meal_repo.go` |
| **RecipeRepository** | Recipe ingredient CRUD | `repository/recipe_repo.go` |
| **TagRepository** | Tag CRUD operations | `repository/tag_repo.go` |

### Testing
- [ ] Cosine similarity math correctness (known test vectors)
- [ ] Threshold filter excludes scores < 0.40
- [ ] Visual indicator assignment by tier
- [ ] Recipe macro aggregation (sum of ingredients)
- [ ] Unit conversion accuracy (g->oz, ml->fl oz)
- [ ] Quantity-based macro scaling
- [ ] Matching quantity calculation for calorie/protein match

---

