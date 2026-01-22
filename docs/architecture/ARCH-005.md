# [ARCH-005] - Data Repository Module

**Description:** Central data access layer implementing the domain data model, handling all database operations, unit conversions, and data normalization for food items, meals, and recipes.

| Attribute | Value |
| :--- | :--- |
| **Type** | Module |
| **Static Aspects** | FoodItemEntity, MealEntity, RecipeEntity, TagEntity, UnitConverter, MacroNormalizer, RepositoryInterfaces |
| **Dependencies** | PostgreSQL (primary datastore) |
| **Traceability** | SW-REQ-032, SW-REQ-033, SW-REQ-034, SW-REQ-035, SW-REQ-036, SW-REQ-037, SW-REQ-038, SW-REQ-039, SW-REQ-040, SW-REQ-041 |

**Dynamic Behavior:**

- **Normalization:** All macronutrient values stored per 100g (solids) or 100ml (liquids). Conversion applied on read based on user preference.
- **Unit Conversion:** Metric-to-Imperial conversion (g->oz, ml->fl oz) performed at repository boundary, never in storage.
- **Recipe Aggregation:** Dynamically calculates total macros for recipe-based meals by summing constituent ingredients.
- **Real-time Scaling:** Provides calculation methods for quantity-based macro scaling.

**Interface Definition:**

- `Input`: CRUD operations, unit preference context, quantity parameters
- `Output`: Domain entities with macros in requested unit system

**Data Model (Core Entities):**

```
FoodItem {
  id: UUID
  name: string
  physicalState: 'solid' | 'liquid'        // SW-REQ-035
  prepTime: minutes                         // SW-REQ-035
  averageUnitWeight: grams                  // SW-REQ-036
  macros: { protein, carbs, fat } per 100g  // SW-REQ-033
  micros: { sodium, fiber, ... }            // SW-REQ-038
  categoryTags: Tag[]                       // SW-REQ-012
  functionalityTags: Tag[]                  // SW-REQ-037
  imageUrl: string?
}

Meal {
  id: UUID
  type: 'single' | 'recipe'                 // SW-REQ-034
  items?: FoodItem                          // single dish
  recipe?: { item: FoodItem, qty: number }[] // recipe composition
  physicalState: 'solid' | 'liquid'        // SW-REQ-035
  prepTime: minutes                         // SW-REQ-035
  averageUnitWeight: grams                  // SW-REQ-036
  categoryTags: Tag[]                       // SW-REQ-012
  functionalityTags: Tag[]                  // SW-REQ-037
}

SimilarityIndicatorAsset {                   // SW-REQ-018
  tier: 'excellent' | 'good' | 'fair' | 'poor'
  colorHex: string                           // e.g., "#22C55E" for green
  imageUrl: string                           // Server-hosted image path
  minScore: number                           // Lower bound threshold
  maxScore: number                           // Upper bound threshold
}
```

**Alternative Analysis (BP6):**

- *Chosen Approach:* PostgreSQL relational database with normalized schema
- *Alternative Considered:* MongoDB document store for flexible food item schema
- *Trade-off:* Relational model ensures data integrity for macronutrient calculations and enforces consistent schema across all items (critical for SW-REQ-033). Recipe composition with foreign keys prevents orphaned ingredients. PostgreSQL JSONB columns can handle variable micronutrient fields while maintaining relational benefits.

**Reference Documentation:** 
- 02_APPENDIX_A.md
