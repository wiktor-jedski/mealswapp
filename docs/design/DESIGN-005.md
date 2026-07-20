## FILE: DESIGN-005.md
**Traceability:** ARCH-005

**Static aspects covered:** FoodItemEntity, MealEntity, RecipeEntity, ClassificationEntity, MicronutrientVocabulary, UnitConverter, MacroNormalizer, RepositoryInterfaces.

### 0. Static Aspect Responsibilities
- `FoodItemEntity`: owns persisted food item fields, physical state, macro/micro values, classifications, and image metadata.
- `MealEntity`: owns opaque single meals and ingredient-derived composite meals plus meal-level physical and preparation metadata.
- `RecipeEntity`: owns ingredient composition, quantities, and aggregate macro calculation inputs.
- `ClassificationEntity`: owns Food Category and Culinary Role classification identity, hierarchy, and uniqueness constraints.
- `MicronutrientVocabulary`: owns canonical micronutrient keys, display names, units, active/inactive state, and validation before storage.
- `UnitConverter`: owns metric/imperial and serving-to-base conversions at repository boundaries.
- `MacroNormalizer`: owns per-100g/per-100ml storage normalization and quantity scaling.
- `RepositoryInterfaces`: owns typed data access contracts for services and modules.

### 1. Data Structures & Types
- `type PhysicalState = "solid" | "liquid"`
- `type UnitSystem = "metric" | "imperial"`
- `interface MacroValues { protein: number; carbs: number; fat: number }`
- `interface MicroValues { [canonicalKey: string]: number | undefined }`
- `interface MicronutrientVocabularyEntry { key: string; displayName: string; unit: string; active: boolean }`
- `interface FoodItemEntity { id: UUID; name: string; physicalState: PhysicalState; prepTimeMinutes: number; averageUnitWeightGrams?: number; averageServingVolumeMilliliters?: number; densityGramsPerMilliliter: number when liquid; densitySourceProvider?: string; densitySourceFoodId?: string; densitySourceKind: "imported" | "manual" | "estimated" when liquid; macrosPer100: MacroValues; micros: MicroValues; foodCategories: ClassificationEntity[]; culinaryRoles: ClassificationEntity[]; imageUrl?: string }`
- `interface MealEntity { id: UUID; type: "single" | "composite"; name: string; recipeItems?: RecipeIngredientEntity[]; physicalState: PhysicalState; prepTimeMinutes: number; averageUnitWeightGrams?: number; macrosPer100: MacroValues; normalizedMacrosAvailable: boolean }`
- `interface RecipeIngredientEntity { foodItemId: UUID; quantity: number; unit: string }`
- `interface ClassificationEntity { id: UUID; name: string; kind: "food_category" | "culinary_role"; parentId?: UUID }`
- `interface RepositoryContext { userId?: UUID; unitSystem: UnitSystem; includeDeleted: boolean }`

### 2. Logic & Algorithms (Step-by-Step)
1. Store all base macro values per 100g for solids and per 100ml for liquids.
2. Convert user-entered quantities to the storage basis before insert or update.
3. Convert solid servings with `averageUnitWeightGrams` and liquid servings with `averageServingVolumeMilliliters`; never treat grams as a milliliter proxy.
4. Validate every micronutrient key against active `MicronutrientVocabularyEntry` records before insert or update; reject aliases such as `Na` when the canonical key is `Sodium`.
5. Store micronutrients as supplemental display/export data only; repository methods that build similarity inputs must return only protein, carbohydrates, and fat.
6. Use repository methods as the only data access path for domain services.
7. Apply user scoping in repository queries whenever custom items, saved meals, profile data, or history are requested.
8. For composite meals, load ingredients, sum each ingredient macro after scaling by ingredient quantity, and normalize the total to the meal's per-100g basis. Convert liquid ingredient volume to mass using required density. Missing persisted liquid density is invalid data and returns an error.
9. Use `g`, `ml`, `oz`, and `fl_oz` as the canonical physical quantity vocabulary. Retain the internal `serving` token only for recipe ingredients as the persisted implementation of SW-REQ-036's user-facing “per unit” calculation: convert solid servings through `averageUnitWeightGrams` and liquid servings through `averageServingVolumeMilliliters` before macro scaling. Accept `g`, `oz`, and `serving` for solid recipe ingredients; accept `ml`, `fl_oz`, and `serving` for liquid recipe ingredients. Saved-diet and substitution contracts never accept `serving`. Reject cross-basis units at service, repository, HTTP, and database boundaries.
10. Convert metric values to imperial only at the repository boundary when `RepositoryContext.unitSystem = "imperial"`.
11. Use raw SQL with parameter binding through `pgx` or `lib/pq`; never concatenate user input into SQL.
12. Maintain indexes for item name, food_category classifications, culinary_role classifications, micronutrient vocabulary keys, and common filter columns.
13. Return domain entities with normalized macros, validated micronutrients, and hydrated classification lists for callers.
14. Apply meal-search `LIMIT` and `OFFSET` in SQL after a filter-equivalent count query; hydrate only IDs in the requested page so iterative consumers never rehydrate earlier pages.

### 3. State Management & Error Handling
- `not_found`: repository returns typed not-found errors; controllers map to 404.
- `validation_error`: invalid macro values, physical state, quantities, or missing required classifications.
- `invalid_micronutrient_key`: micronutrient key is not present as an active canonical vocabulary entry; reject the write.
- `constraint_violation`: duplicate names or foreign-key failures; map to conflict or bad request.
- `connection_error`: database unavailable; fail fast for callers.
- `read_replica_lag`: route critical reads to primary or return degraded warning.
- `unit_conversion_error`: unsupported unit; reject before persistence.

### 4. Component Interfaces
- `type FoodItemRepository interface { GetByID(ctx context.Context, id UUID, rc RepositoryContext) (FoodItemEntity, error); Search(ctx context.Context, q RepositoryQuery) ([]FoodItemEntity, int, error); Create(ctx context.Context, item FoodItemEntity) (UUID, error); Update(ctx context.Context, item FoodItemEntity) error; Delete(ctx context.Context, id UUID) error }`
- `type MealRepository interface { GetByID(ctx context.Context, id UUID, rc RepositoryContext) (MealEntity, error); Search(ctx context.Context, q RepositoryQuery) ([]MealEntity, int, error); CalculateMacros(ctx context.Context, mealID UUID) (MacroValues, error); Create(ctx context.Context, meal MealEntity) (UUID, error); Update(ctx context.Context, meal MealEntity) error; Delete(ctx context.Context, id UUID) error }`
- `type ClassificationRepository interface { List(ctx context.Context, kind string) ([]ClassificationEntity, error); Upsert(ctx context.Context, classification ClassificationEntity) (UUID, error); IsInUse(ctx context.Context, id UUID) (bool, error); SoftDelete(ctx context.Context, id UUID) error }`
- `type MicronutrientVocabularyRepository interface { ListActive(ctx context.Context) ([]MicronutrientVocabularyEntry, error); IsAllowed(ctx context.Context, key string) (bool, error); Upsert(ctx context.Context, entry MicronutrientVocabularyEntry) error }`
- `func NormalizeMacros(value MacroValues, quantity float64, state PhysicalState) MacroValues`
- `func ValidateMicronutrientKeys(values MicroValues, vocabulary []MicronutrientVocabularyEntry) error`
- `func ConvertUnit(value float64, fromUnit string, toUnit string) (float64, error)`
- `func ScaleMacros(base MacroValues, quantity float64, basis float64) MacroValues`
