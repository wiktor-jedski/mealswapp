# FILE: RepositoryInterfaces.md

**Traceability:** ARCH-005

## 1. Data Structures & Types

### 1.1 Domain Entities

```go
type UUID string

type PhysicalState string

const (
    PhysicalStateSolid  PhysicalState = "solid"
    PhysicalStateLiquid PhysicalState = "liquid"
)

type MealType string

const (
    MealTypeSingle   MealType = "single"
    MealTypeRecipe   MealType = "recipe"
)

type SimilarityTier string

const (
    SimilarityTierExcellent SimilarityTier = "excellent"
    SimilarityTierGood      SimilarityTier = "good"
    SimilarityTierFair      SimilarityTier = "fair"
    SimilarityTierPoor      SimilarityTier = "poor"
)

type UnitSystem string

const (
    UnitSystemMetric    UnitSystem = "metric"
    UnitSystemImperial  UnitSystem = "imperial"
)

type Macros struct {
    Protein float64 `json:"protein"`
    Carbs   float64 `json:"carbs"`
    Fat     float64 `json:"fat"`
}

type Micros struct {
    Sodium float64 `json:"sodium"`
    Fiber  float64 `json:"fiber"`
}

type Tag struct {
    ID   UUID   `json:"id"`
    Name string `json:"name"`
}

type FoodItem struct {
    ID                  UUID         `json:"id"`
    Name                string       `json:"name"`
    PhysicalState       PhysicalState `json:"physicalState"`
    PrepTime            int          `json:"prepTime"`
    AverageUnitWeight   float64      `json:"averageUnitWeight"`
    Macros              Macros       `json:"macros"`
    Micros              Micros       `json:"micros"`
    CategoryTags        []Tag        `json:"categoryTags"`
    FunctionalityTags   []Tag        `json:"functionalityTags"`
    ImageUrl            *string      `json:"imageUrl,omitempty"`
}

type RecipeIngredient struct {
    FoodItem  FoodItem `json:"foodItem"`
    Quantity  float64  `json:"quantity"`
}

type Meal struct {
    ID                 UUID           `json:"id"`
    Type               MealType       `json:"type"`
    Items              *FoodItem      `json:"items,omitempty"`
    Recipe             []RecipeIngredient `json:"recipe,omitempty"`
    PhysicalState      PhysicalState  `json:"physicalState"`
    PrepTime           int            `json:"prepTime"`
    AverageUnitWeight  float64        `json:"averageUnitWeight"`
    CategoryTags       []Tag          `json:"categoryTags"`
    FunctionalityTags  []Tag          `json:"functionalityTags"`
}

type SimilarityIndicatorAsset struct {
    Tier     SimilarityTier `json:"tier"`
    ColorHex string         `json:"colorHex"`
    ImageUrl string         `json:"imageUrl"`
    MinScore float64        `json:"minScore"`
    MaxScore float64        `json:"maxScore"`
}
```

### 1.2 Database Row Types

```go
type FoodItemRow struct {
    ID                 UUID
    Name               string
    PhysicalState      PhysicalState
    PrepTime           int
    AverageUnitWeight  float64
    ProteinPer100g     float64
    CarbsPer100g       float64
    FatPer100g         float64
    SodiumPer100g      float64
    FiberPer100g       float64
    ImageUrl           *string
}

type TagRow struct {
    ID   UUID
    Name string
}

type MealRow struct {
    ID                UUID
    Type              MealType
    PhysicalState     PhysicalState
    PrepTime          int
    AverageUnitWeight float64
}
```

### 1.3 Repository Configuration

```go
type RepositoryConfig struct {
    DBConnectionString string
    MaxOpenConnections int
    MaxIdleConnections int
    ConnectionTimeout  time.Duration
    QueryTimeout       time.Duration
}

type UnitPreference struct {
    System   UnitSystem
    VolumeUnit string
    WeightUnit string
}
```

## 2. Logic & Algorithms

### 2.1 Unit Conversion Algorithm

```
ALGORITHM ConvertWeight(value float64, fromUnit string, toUnit string) -> float64
    CONSTANTS:
        GRAMS_PER_OUNCE = 28.3495

    IF fromUnit == toUnit THEN
        RETURN value

    IF fromUnit == "g" AND toUnit == "oz" THEN
        RETURN value / GRAMS_PER_OUNCE

    IF fromUnit == "oz" AND toUnit == "g" THEN
        RETURN value * GRAMS_PER_OUNCE

    RETURN value
END ALGORITHM

ALGORITHM ConvertVolume(value float64, fromUnit string, toUnit string) -> float64
    CONSTANTS:
        ML_PER_FL_OZ = 29.5735

    IF fromUnit == toUnit THEN
        RETURN value

    IF fromUnit == "ml" AND toUnit == "fl_oz" THEN
        RETURN value / ML_PER_FL_OZ

    IF fromUnit == "fl_oz" AND toUnit == "ml" THEN
        RETURN value * ML_PER_FL_OZ

    RETURN value
END ALGORITHM
```

### 2.2 Macro Normalization Algorithm

```
ALGORITHM NormalizeMacrosToUserUnit(
    storedValue float64,
    baseQuantity float64,
    requestedQuantity float64,
    physicalState PhysicalState
) -> float64
    ratio = requestedQuantity / baseQuantity
    normalizedValue = storedValue * ratio

    IF physicalState == PhysicalStateSolid AND userPrefersImperial() THEN
        normalizedValue = ConvertWeight(normalizedValue, "g", "oz")
    ELSE IF physicalState == PhysicalStateLiquid AND userPrefersImperial() THEN
        normalizedValue = ConvertVolume(normalizedValue, "ml", "fl_oz")
    END IF

    RETURN normalizedValue
END ALGORITHM
```

### 2.3 Recipe Aggregation Algorithm

```
ALGORITHM CalculateRecipeMacros(recipe []RecipeIngredient) -> Macros
    totalProtein = 0.0
    totalCarbs = 0.0
    totalFat = 0.0

    FOR EACH ingredient IN recipe DO
        item = ingredient.FoodItem
        quantity = ingredient.Quantity

        totalProtein += NormalizeMacrosToUserUnit(
            item.Macros.Protein,
            100,
            quantity,
            item.PhysicalState
        )
        totalCarbs += NormalizeMacrosToUserUnit(
            item.Macros.Carbs,
            100,
            quantity,
            item.PhysicalState
        )
        totalFat += NormalizeMacrosToUserUnit(
            item.Macros.Fat,
            100,
            quantity,
            item.PhysicalState
        )
    END FOR

    RETURN Macros{
        Protein: totalProtein,
        Carbs: totalCarbs,
        Fat: totalFat
    }
END ALGORITHM
```

### 2.4 Quantity Scaling Algorithm

```
ALGORITHM ScaleMacrosForQuantity(
    baseMacros Macros,
    baseQuantity float64,
    targetQuantity float64
) -> Macros
    scaleFactor = targetQuantity / baseQuantity

    RETURN Macros{
        Protein: baseMacros.Protein * scaleFactor,
        Carbs: baseMacros.Carbs * scaleFactor,
        Fat: baseMacros.Fat * scaleFactor
    }
END ALGORITHM
```

### 2.5 Food Item Retrieval Flow

```
ALGORITHM GetFoodItemByID(id UUID, unitPref UnitPreference) -> FoodItem
    row = ExecuteQuery("SELECT * FROM food_items WHERE id = $1", id)
    IF row IS NULL THEN
        RAISE ItemNotFoundError
    END IF

    foodItem = MapRowToFoodItem(row)

    tags = ExecuteQuery(
        "SELECT t.id, t.name FROM tags t " +
        "JOIN food_item_tags fit ON t.id = fit.tag_id " +
        "WHERE fit.food_item_id = $1",
        id
    )
    foodItem.CategoryTags = FilterTagsByType(tags, "category")
    foodItem.FunctionalityTags = FilterTagsByType(tags, "functionality")

    foodItem.Macros.Protein = NormalizeMacrosToUserUnit(
        foodItem.Macros.Protein, 100, 100, foodItem.PhysicalState
    )
    foodItem.Macros.Carbs = NormalizeMacrosToUserUnit(
        foodItem.Macros.Carbs, 100, 100, foodItem.PhysicalState
    )
    foodItem.Macros.Fat = NormalizeMacrosToUserUnit(
        foodItem.Macros.Fat, 100, 100, foodItem.PhysicalState
    )

    RETURN foodItem
END ALGORITHM
```

### 2.6 Meal Retrieval Flow

```
ALGORITHM GetMealByID(id UUID, unitPref UnitPreference) -> Meal
    mealRow = ExecuteQuery("SELECT * FROM meals WHERE id = $1", id)
    IF mealRow IS NULL THEN
        RAISE ItemNotFoundError
    END IF

    meal = MapRowToMeal(mealRow)

    IF meal.Type == MealTypeSingle THEN
        meal.Items = GetFoodItemByID(mealRow.ItemID, unitPref)
    ELSE IF meal.Type == MealTypeRecipe THEN
        ingredients = ExecuteQuery(
            "SELECT fi.*, mi.quantity FROM meal_ingredients mi " +
            "JOIN food_items fi ON mi.food_item_id = fi.id " +
            "WHERE mi.meal_id = $1",
            id
        )
        FOR EACH row IN ingredients DO
            ingredient = MapRowToRecipeIngredient(row)
            ingredient.FoodItem = MapRowToFoodItem(row)
            meal.Recipe = append(meal.Recipe, ingredient)
        END FOR
    END IF

    tags = ExecuteQuery(
        "SELECT t.id, t.name FROM tags t " +
        "JOIN meal_tags mt ON t.id = mt.tag_id " +
        "WHERE mt.meal_id = $1",
        id
    )
    meal.CategoryTags = FilterTagsByType(tags, "category")
    meal.FunctionalityTags = FilterTagsByType(tags, "functionality")

    RETURN meal
END ALGORITHM
```

## 3. State Management & Error Handling

### 3.1 Error Types

```go
var (
    ErrItemNotFound = fmt.Errorf("food item or meal not found")
    ErrTagNotFound = fmt.Errorf("tag not found")
    ErrInvalidQuantity = fmt.Errorf("quantity must be positive")
    ErrInvalidUnitPreference = fmt.Errorf("invalid unit system preference")
    ErrDatabaseConnection = fmt.Errorf("database connection failed")
    ErrQueryTimeout = fmt.Errorf("database query timed out")
    ErrTransactionFailed = fmt.Errorf("transaction failed")
    ErrConstraintViolation = fmt.Errorf("database constraint violation")
    ErrConversionNotSupported = fmt.Errorf("unit conversion not supported")
)
```

### 3.2 Error Handling Strategy

| Error Condition | Handler Action | User Impact |
| :--- | :--- | :--- |
| ItemNotFoundError | Return nil entity with error code 404 | "Item not found" displayed |
| DatabaseConnection | Retry 3 times with exponential backoff | "Service temporarily unavailable" |
| QueryTimeout | Cancel context, return timeout error | "Request took too long" |
| ConstraintViolation | Log details, return 400 with violation type | "Invalid data submitted" |
| InvalidQuantity | Return validation error before DB call | "Enter a valid quantity" |
| UnitConversionError | Fallback to metric units, log warning | Uses default metric display |

### 3.3 State Transitions

```
STATE: EmptyResultSet
  -> Trigger: Query returns no rows
  -> Action: Return empty slice, nil error
  -> Next State: Idle

STATE: Loading
  -> Trigger: Repository method invoked
  -> Action: Acquire DB connection from pool
  -> Next State: QueryExecuting

STATE: QueryExecuting
  -> Trigger: Query completes successfully
  -> Action: Map rows to domain entities, normalize values
  -> Next State: Ready

STATE: QueryExecuting
  -> Trigger: Query fails with connection error
  -> Action: Release connection, attempt reconnect
  -> Next State: Loading

STATE: QueryExecuting
  -> Trigger: Query fails with timeout
  -> Action: Cancel context, release connection
  -> Next State: Error

STATE: Error
  -> Trigger: Error is retryable
  -> Action: Increment retry counter, backoff
  -> Next State: Loading

STATE: Error
  -> Trigger: Error is non-retryable
  -> Action: Return error to caller
  -> Next State: Idle
```

### 3.4 Connection Pool Management

```go
type ConnectionPool struct {
    mu           sync.Mutex
    openCount    int
    idleCount    int
    waitCount    int
    maxOpen      int
    maxIdle      int
    maxLifetime  time.Duration
}

ALGORITHM AcquireConnection(pool *ConnectionPool) -> *DBConnection
    pool.mu.Lock()

    IF pool.openCount >= pool.maxOpen THEN
        IF pool.idleCount > 0 THEN
            connection = PopFromIdlePool()
            pool.idleCount--
            pool.mu.Unlock()
            RETURN connection
        END IF

        pool.waitCount++
        pool.mu.Unlock()

        WaitForConnection(timeout: pool.config.QueryTimeout)

        pool.mu.Lock()
        pool.waitCount--
        pool.mu.Lock()
    END IF

    connection = CreateNewConnection()
    pool.openCount++
    pool.mu.Unlock()

    RETURN connection
END ALGORITHM

ALGORITHM ReleaseConnection(pool *ConnectionPool, connection *DBConnection)
    pool.mu.Lock()

    IF pool.openCount > pool.maxIdle THEN
        CloseConnection(connection)
        pool.openCount--
    ELSE
        PushToIdlePool(connection)
        pool.idleCount++
    END IF

    IF pool.waitCount > 0 AND pool.idleCount > 0 THEN
        SignalWaitingGoroutine()
    END IF

    pool.mu.Unlock()
END ALGORITHM
```

## 4. Component Interfaces

### 4.1 FoodItemRepository Interface

```go
type FoodItemRepository interface {
    Create(ctx context.Context, item *FoodItem) error
    GetByID(ctx context.Context, id UUID, unitPref UnitPreference) (*FoodItem, error)
    Update(ctx context.Context, item *FoodItem) error
    Delete(ctx context.Context, id UUID) error
    List(ctx context.Context, limit, offset int) ([]FoodItem, error)
    ListByCategory(ctx context.Context, categoryID UUID, limit, offset int) ([]FoodItem, error)
    ListByFunctionality(ctx context.Context, functionalityID UUID, limit, offset int) ([]FoodItem, error)
    SearchByName(ctx context.Context, query string, limit, offset int) ([]FoodItem, error)
    AddTag(ctx context.Context, foodItemID, tagID UUID) error
    RemoveTag(ctx context.Context, foodItemID, tagID UUID) error
    GetTags(ctx context.Context, foodItemID UUID) ([]Tag, error)
    GetMacrosPer100g(ctx context.Context, id UUID) (Macros, error)
}
```

### 4.2 MealRepository Interface

```go
type MealRepository interface {
    Create(ctx context.Context, meal *Meal) error
    GetByID(ctx context.Context, id UUID, unitPref UnitPreference) (*Meal, error)
    Update(ctx context.Context, meal *Meal) error
    Delete(ctx context.Context, id UUID) error
    List(ctx context.Context, limit, offset int) ([]Meal, error)
    ListByType(ctx context.Context, mealType MealType, limit, offset int) ([]Meal, error)
    AddRecipeIngredient(ctx context.Context, mealID, foodItemID UUID, quantity float64) error
    RemoveRecipeIngredient(ctx context.Context, mealID, foodItemID UUID) error
    UpdateRecipeIngredientQuantity(ctx context.Context, mealID, foodItemID UUID, quantity float64) error
    GetRecipeIngredients(ctx context.Context, mealID UUID) ([]RecipeIngredient, error)
    AddTag(ctx context.Context, mealID, tagID UUID) error
    RemoveTag(ctx context.Context, mealID, tagID UUID) error
    GetTags(ctx context.Context, mealID UUID) ([]Tag, error)
}
```

### 4.3 TagRepository Interface

```go
type TagRepository interface {
    Create(ctx context.Context, tag *Tag) error
    GetByID(ctx context.Context, id UUID) (*Tag, error)
    GetByName(ctx context.Context, name string) (*Tag, error)
    Update(ctx context.Context, tag *Tag) error
    Delete(ctx context.Context, id UUID) error
    List(ctx context.Context, tagType string, limit, offset int) ([]Tag, error)
    ListAll(ctx context.Context) ([]Tag, error)
    GetCategoryTags(ctx context.Context) ([]Tag, error)
    GetFunctionalityTags(ctx context.Context) ([]Tag, error)
}
```

### 4.4 SimilarityIndicatorRepository Interface

```go
type SimilarityIndicatorRepository interface {
    GetByTier(ctx context.Context, tier SimilarityTier) (*SimilarityIndicatorAsset, error)
    GetAll(ctx context.Context) ([]SimilarityIndicatorAsset, error)
    GetByScoreRange(ctx context.Context, minScore, maxScore float64) (*SimilarityIndicatorAsset, error)
    UpdateAsset(ctx context.Context, asset *SimilarityIndicatorAsset) error
}
```

### 4.5 UnitConverter Interface

```go
type UnitConverter interface {
    ConvertWeight(value float64, fromUnit, toUnit string) (float64, error)
    ConvertVolume(value float64, fromUnit, toUnit string) (float64, error)
    ConvertTemperature(value float64, fromUnit, toUnit string) (float64, error)
    IsWeightUnit(unit string) bool
    IsVolumeUnit(unit string) bool
    GetDefaultUnitForSystem(system UnitSystem, physicalState PhysicalState) string
}
```

### 4.6 MacroNormalizer Interface

```go
type MacroNormalizer interface {
    NormalizeToPer100g(value float64, currentUnit string, physicalState PhysicalState) (float64, error)
    NormalizeFromPer100g(value float64, targetUnit string, physicalState PhysicalState) (float64, error)
    ScaleMacros(macros Macros, scaleFactor float64) Macros
    CalculateTotalMacros(ingredients []RecipeIngredient) (Macros, error)
    ConvertMacrosToUnitSystem(macros Macros, fromSystem UnitSystem, toSystem UnitSystem, physicalState PhysicalState) (Macros, error)
}
```

### 4.7 Repository Factory

```go
type RepositoryFactory struct {
    db           *pgx.Conn
    config       RepositoryConfig
    converter    UnitConverter
    normalizer   MacroNormalizer
}

func NewRepositoryFactory(db *pgx.Conn, config RepositoryConfig) *RepositoryFactory {
    return &RepositoryFactory{
        db:         db,
        config:     config,
        converter:  NewUnitConverter(),
        normalizer: NewMacroNormalizer(),
    }
}

func (f *RepositoryFactory) FoodItemRepository() FoodItemRepository {
    return NewFoodItemRepository(f.db, f.converter, f.normalizer)
}

func (f *RepositoryFactory) MealRepository() MealRepository {
    return NewMealRepository(f.db, f.converter, f.normalizer)
}

func (f *RepositoryFactory) TagRepository() TagRepository {
    return NewTagRepository(f.db)
}

func (f *RepositoryFactory) SimilarityIndicatorRepository() SimilarityIndicatorRepository {
    return NewSimilarityIndicatorRepository(f.db)
}
```

### 4.8 Transaction Support

```go
type Transaction interface {
    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
    FoodItemRepository() FoodItemRepository
    MealRepository() MealRepository
    TagRepository() TagRepository
}

type Repository interface {
    BeginTx(ctx context.Context) (Transaction, error)
    WithTransaction(ctx context.Context, fn func(tx Transaction) error) error
}
```

### 4.9 Repository Configuration Methods

```go
type RepositoryConfigurator struct {
    config RepositoryConfig
}

func NewRepositoryConfigurator() *RepositoryConfigurator {
    return &RepositoryConfigurator{
        config: RepositoryConfig{
            MaxOpenConnections: 25,
            MaxIdleConnections: 5,
            ConnectionTimeout:  30 * time.Second,
            QueryTimeout:       15 * time.Second,
        },
    }
}

func (c *RepositoryConfigurator) WithConnectionString(connStr string) *RepositoryConfigurator {
    c.config.DBConnectionString = connStr
    return c
}

func (c *RepositoryConfigurator) WithMaxOpenConnections(n int) *RepositoryConfigurator {
    c.config.MaxOpenConnections = n
    return c
}

func (c *RepositoryConfigurator) WithMaxIdleConnections(n int) *RepositoryConfigurator {
    c.config.MaxIdleConnections = n
    return c
}

func (c *RepositoryConfigurator) WithQueryTimeout(timeout time.Duration) *RepositoryConfigurator {
    c.config.QueryTimeout = timeout
    return c
}

func (c *RepositoryConfigurator) Build() RepositoryConfig {
    return c.config
}
```
