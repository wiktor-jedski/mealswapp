# FILE: PreferenceManager.md

**Traceability:** ARCH-008

## 1. Data Structures & Types

### 1.1 Core Interfaces

```go
type UnitPreference string

const (
    UnitPreferenceMetric   UnitPreference = "metric"
    UnitPreferenceImperial UnitPreference = "imperial"
)

type WeightUnit string

const (
    WeightUnitGrams    WeightUnit = "g"
    WeightUnitOunces   WeightUnit = "oz"
    WeightUnitPounds   WeightUnit = "lb"
    WeightUnitKilograms WeightUnit = "kg"
)

type VolumeUnit string

const (
    VolumeUnitMilliliters VolumeUnit = "ml"
    VolumeUnitLiters      VolumeUnit = "l"
    VolumeUnitTeaspoons   VolumeUnit = "tsp"
    VolumeUnitTablespoons VolumeUnit = "tbsp"
    VolumeUnitCups        VolumeUnit = "cup"
    VolumeUnitFluidOunces VolumeUnit = "fl_oz"
)

type TemperatureUnit string

const (
    TemperatureUnitCelsius    TemperatureUnit = "celsius"
    TemperatureUnitFahrenheit TemperatureUnit = "fahrenheit"
)

type ThemePreference string

const (
    ThemePreferenceLight  ThemePreference = "light"
    ThemePreferenceDark   ThemePreference = "dark"
    ThemePreferenceSystem ThemePreference = "system"
)

type NotificationSettings struct {
    EmailNotifications    bool `json:"email_notifications"`
    PushNotifications     bool `json:"push_notifications"`
    MealReminders         bool `json:"meal_reminders"`
    WeeklyDigest          bool `json:"weekly_digest"`
    NewFeatures           bool `json:"new_features"`
    PriceAlerts           bool `json:"price_alerts"`
}

type UserPreferences struct {
    ID                   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    UserID               primitive.ObjectID `json:"user_id" bson:"user_id"`
    UnitSystem           UnitPreference     `json:"unit_system" bson:"unit_system"`
    WeightUnit           WeightUnit         `json:"weight_unit" bson:"weight_unit"`
    VolumeUnit           VolumeUnit         `json:"volume_unit" bson:"volume_unit"`
    TemperatureUnit      TemperatureUnit    `json:"temperature_unit" bson:"temperature_unit"`
    Theme                ThemePreference    `json:"theme" bson:"theme"`
    Language             string             `json:"language" bson:"language"`
    Timezone             string             `json:"timezone" bson:"timezone"`
    Currency             string             `json:"currency" bson:"currency"`
    Notifications        NotificationSettings `json:"notifications" bson:"notifications"`
    DietaryRestrictions  []string           `json:"dietary_restrictions" bson:"dietary_restrictions"`
    Allergies            []string           `json:"allergies" bson:"allergies"`
    DefaultServings      int                `json:"default_servings" bson:"default_servings"`
    AutoScaleRecipes     bool               `json:"auto_scale_recipes" bson:"auto_scale_recipes"`
    ShowNutritionalInfo  bool               `json:"show_nutritional_info" bson:"show_nutritional_info"`
    PreferCheaperAlternatives bool           `json:"prefer_cheaper_alternatives" bson:"prefer_cheaper_alternatives"`
    MaxBudgetPerMeal     float64            `json:"max_budget_per_meal" bson:"max_budget_per_meal"`
    CreatedAt            time.Time          `json:"created_at" bson:"created_at"`
    UpdatedAt            time.Time          `json:"updated_at" bson:"updated_at"`
}

type PreferenceUpdateRequest struct {
    UnitSystem           *UnitPreference     `json:"unit_system,omitempty"`
    WeightUnit           *WeightUnit         `json:"weight_unit,omitempty"`
    VolumeUnit           *VolumeUnit         `json:"volume_unit,omitempty"`
    TemperatureUnit      *TemperatureUnit    `json:"temperature_unit,omitempty"`
    Theme                *ThemePreference    `json:"theme,omitempty"`
    Language             *string             `json:"language,omitempty"`
    Timezone             *string             `json:"timezone,omitempty"`
    Currency             *string             `json:"currency,omitempty"`
    Notifications        *NotificationSettings `json:"notifications,omitempty"`
    DietaryRestrictions  *[]string           `json:"dietary_restrictions,omitempty"`
    Allergies            *[]string           `json:"allergies,omitempty"`
    DefaultServings      *int                `json:"default_servings,omitempty"`
    AutoScaleRecipes     *bool               `json:"auto_scale_recipes,omitempty"`
    ShowNutritionalInfo  *bool               `json:"show_nutritional_info,omitempty"`
    PreferCheaperAlternatives *bool           `json:"prefer_cheaper_alternatives,omitempty"`
    MaxBudgetPerMeal     *float64            `json:"max_budget_per_meal,omitempty"`
}

type PreferenceEvent struct {
    EventType       string          `json:"event_type"`
    UserID          primitive.ObjectID `json:"user_id"`
    ChangedFields   []string        `json:"changed_fields"`
    OldPreferences  *UserPreferences `json:"old_preferences,omitempty"`
    NewPreferences  *UserPreferences `json:"new_preferences,omitempty"`
    Timestamp       time.Time       `json:"timestamp"`
}

type ConversionResult struct {
    OriginalValue   float64   `json:"original_value"`
    ConvertedValue  float64   `json:"converted_value"`
    OriginalUnit    string    `json:"original_unit"`
    TargetUnit      string    `json:"target_unit"`
}
```

### 1.2 Storage Models

```go
type PreferencesDocument struct {
    CollectionName string
    Schema: bson.M{
        "user_id":           bson.M{"$type": "objectId", "$required": true},
        "unit_system":       bson.M{"$type": "string", "$enum": ["metric", "imperial"]},
        "weight_unit":       bson.M{"$type": "string"},
        "volume_unit":       bson.M{"$type": "string"},
        "temperature_unit":  bson.M{"$type": "string"},
        "theme":             bson.M{"$type": "string", "$enum": ["light", "dark", "system"]},
        "language":          bson.M{"$type": "string"},
        "timezone":          bson.M{"$type": "string"},
        "currency":          bson.M{"$type": "string"},
        "notifications":     bson.M{"$type": "document"},
        "dietary_restrictions": bson.M{"$type": "array"},
        "allergies":         bson.M{"$type": "array"},
        "default_servings":  bson.M{"$type": "int", "$min": 1, "$max": 20},
        "auto_scale_recipes": bson.M{"$type": "bool"},
        "show_nutritional_info": bson.M{"$type": "bool"},
        "prefer_cheaper_alternatives": bson.M{"$type": "bool"},
        "max_budget_per_meal": bson.M{"$type": "double", "$min": 0},
        "created_at":        bson.M{"$type": "date"},
        "updated_at":        bson.M{"$type": "date"},
    }
    Indexes: []mongo.IndexModel{
        {Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
    }
}
```

## 2. Logic & Algorithms

### 2.1 Preference Manager Workflow

```
PreferenceManager.GetPreferences(userID)
    |
    +---> Validate userID is not empty
    |
    +---> Check Redis cache for user preferences
    |     |
    |     +---> Cache hit: Return cached preferences
    |     |
    |     +---> Cache miss:
    |           |
    |           +---> Query MongoDB for preferences
    |           |
    |           +---> If not found:
    |                 |
    |                 +---> Create default preferences
    |                 |
    |                 +---> Insert into MongoDB
    |                 |
    |                 +---> Cache in Redis with TTL 1 hour
    |                 |
    |                 +---> Return default preferences
    |
    +---> Return preferences
```

### 2.2 Preference Update Workflow

```
PreferenceManager.UpdatePreferences(userID, updateRequest)
    |
    +---> Validate updateRequest contains at least one field
    |
    +---> Fetch current preferences from cache or MongoDB
    |
    +---> Create deep copy of old preferences
    |
    +-----> For each field in updateRequest:
    |        |
    |        +---> Apply update to new preferences object
    |        |
    |        +---> Add field name to changedFields list
    |
    +---> Validate resulting preferences
    |     |
    |     +---> Validate unit consistency
    |     |     |
    |     |     +---> If unit_system changed to "metric":
    |     |           |
    |     |           +---> Reset weight_unit to "g" or "kg" if imperial
    |     |           |
    |     |           +---> Reset volume_unit to "ml" or "l" if imperial
    |     |
    |     +---> If unit_system changed to "imperial":
    |           |
    |           +---> Reset weight_unit to "oz" or "lb" if metric
    |           |
    |           +---> Reset volume_unit to "fl_oz" or "cup" if metric
    |
    +---> Update UpdatedAt timestamp
    |
    +---> Persist to MongoDB
    |     |
    |     +---> Use atomic $set update
    |
    +---> Invalidate Redis cache
    |
    +---> If unit-related fields changed:
    |     |
    |     +---> Publish PreferenceEvent to Redis Pub/Sub
    |           |
    |           +---> Event type: "preferences.updated"
    |           |
    |           +---> Subscribers: Real-time recalculation service
    |
    +---> Return updated preferences
```

### 2.3 Unit Conversion Algorithm

```
PreferenceManager.ConvertValue(value, fromUnit, toUnit, userID)
    |
    +---> Get user preferences to determine target unit system
    |
    +---> Identify conversion category:
    |     |
    |     +---> Weight: g, kg, oz, lb
    |     |
    |     +---> Volume: ml, l, tsp, tbsp, cup, fl_oz
    |     |
    |     +---> Temperature: celsius, fahrenheit
    |
    +---> Apply conversion factors:
    |     |
    |     +---> Weight conversions (base unit: grams):
    |           |
    |           +---> g to kg: / 1000
    |           +---> g to oz: * 0.035274
    |           +---> g to lb: * 0.00220462
    |           |
    |           +---> Reverse: multiply by inverse
    |
    +---> Volume conversions (base unit: milliliters):
    |     |
    |     +---> ml to l: / 1000
    |     +---> ml to tsp: / 4.92892
    |     +---> ml to tbsp: / 14.7868
    |     +---> ml to cup: / 236.588
    |     +---> ml to fl_oz: / 29.5735
    |
    +---> Temperature conversions:
    |     |
    |     +---> Celsius to Fahrenheit: (c * 9/5) + 32
    |     +---> Fahrenheit to Celsius: (f - 32) * 5/9
    |
    +---> Return ConversionResult
```

### 2.4 Default Preferences Generation

```
PreferenceManager.CreateDefaultPreferences(userID)
    |
    +---> Create UserPreferences with defaults:
    |     |
    |     +---> UnitSystem: "metric"
    |     +---> WeightUnit: "g"
    |     +---> VolumeUnit: "ml"
    |     +---> TemperatureUnit: "celsius"
    |     +---> Theme: "system"
    |     +---> Language: "en"
    |     +---> Timezone: "UTC"
    |     +---> Currency: "USD"
    |     +---> Notifications: all true
    |     +---> DietaryRestrictions: empty array
    |     +---> Allergies: empty array
    |     +---> DefaultServings: 2
    |     +---> AutoScaleRecipes: true
    |     +---> ShowNutritionalInfo: true
    |     +---> PreferCheaperAlternatives: false
    |     +---> MaxBudgetPerMeal: 0 (no limit)
    |     +---> CreatedAt: current time
    |     +---> UpdatedAt: current time
    |
    +---> Return UserPreferences
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Code | Condition | HTTP Status | Recovery Action |
| :--- | :--- | :--- | :--- |
| `ERR_USER_NOT_FOUND` | UserID not found in system | 404 | User must complete registration |
| `ERR_INVALID_PREFERENCE_FIELD` | Unknown field in update request | 400 | Validate request against schema |
| `ERR_UNIT_INCONSISTENCY` | Unit system mismatch | 400 | Normalize units before update |
| `ERR_INVALID_UNIT_CONVERSION` | Cannot convert between units | 400 | Verify units are in same category |
| `ERR_CACHE_ERROR` | Redis unavailable | 503 | Fallback to direct DB query |
| `ERR_DATABASE_ERROR` | MongoDB operation failed | 500 | Retry with exponential backoff |
| `ERR_CONCURRENT_UPDATE` | Version conflict on update | 409 | Fetch latest, merge, retry |
| `ERR_VALUE_OUT_OF_RANGE` | Numeric value outside allowed range | 400 | Validate before submission |

### 3.2 State Transitions

```
Initial State: No preferences exist

[GetPreferences Called]
    |
    +---> Loading (cache lookup)
    |     |
    |     +---> Cache hit -> Ready (return cached)
    |     |
    |     +---> Cache miss -> Loading (DB lookup)
    |           |
    |           +---> DB found -> Ready (cache and return)
    |           |
    |           +---> DB not found -> Creating defaults
    |                 |
    |                 +---> Success -> Ready (return defaults)

[UpdatePreferences Called]
    |
    +---> Ready -> Updating
    |     |
    |     +---> Success -> Ready (emit event if unit change)
    |     |
    |     +---> Version conflict -> Loading (refetch, retry)
    |     |
    |     +---> Validation error -> Ready (return errors)
    |     |
    |     +---> DB error -> Error state (retry with backoff)

[ConvertValue Called]
    |
    +---> Ready -> Converting
    |     |
    |     +---> Success -> Ready (return result)
    |     |
    |     +---> Invalid conversion -> Ready (return error)
```

### 3.3 Retry Strategy

```go
func (pm *PreferenceManager) withRetry(ctx context.Context, operation func() error) error {
    maxRetries := 3
    baseDelay := 100 * time.Millisecond

    for attempt := 0; attempt < maxRetries; attempt++ {
        err := operation()
        if err == nil {
            return nil
        }

        if isRetryableError(err) {
            delay := baseDelay * time.Duration(math.Pow(2, float64(attempt)))
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(delay):
                continue
            }
        }

        return err
    }

    return fmt.Errorf("max retries exceeded: %w", err)
}
```

### 3.4 Cache Invalidation Strategy

```go
type CacheInvalidator struct {
    patterns map[string][]string
}

func (ci *CacheInvalidator) InvalidateOnPreferenceChange(userID primitive.ObjectID, changedFields []string) {
    patterns := []string{
        fmt.Sprintf("preferences:%s", userID.Hex()),
        fmt.Sprintf("user:%s:dashboard", userID.Hex()),
        fmt.Sprintf("recipes:%s:*", userID.Hex()),
    }

    if containsUnitField(changedFields) {
        patterns = append(patterns, fmt.Sprintf("nutrition:%s:*", userID.Hex()))
    }

    for _, pattern := range patterns {
        ci.redis.Del(ctx, pattern)
    }

    if containsUnitField(changedFields) {
        ci.redis.Publish(ctx, "preference_updates", PreferenceEvent{
            EventType:   "unit_preference_changed",
            UserID:      userID,
            Timestamp:   time.Now(),
        })
    }
}
```

## 4. Component Interfaces

### 4.1 Public Methods

```go
type PreferenceManager interface {
    // GetPreferences retrieves user preferences, creating defaults if needed
    GetPreferences(ctx context.Context, userID primitive.ObjectID) (*UserPreferences, error)

    // UpdatePreferences updates specific preference fields
    UpdatePreferences(ctx context.Context, userID primitive.ObjectID, updates *PreferenceUpdateRequest) (*UserPreferences, error)

    // ReplacePreferences completely replaces all preferences (admin use)
    ReplacePreferences(ctx context.Context, userID primitive.ObjectID, preferences *UserPreferences) (*UserPreferences, error)

    // DeletePreferences removes all user preferences (account deletion flow)
    DeletePreferences(ctx context.Context, userID primitive.ObjectID) error

    // ConvertValue converts a value between units based on user preferences
    ConvertValue(ctx context.Context, userID primitive.ObjectID, value float64, fromUnit string, toUnit string) (*ConversionResult, error)

    // ConvertRecipeUnits converts all measurements in a recipe to user's preferred units
    ConvertRecipeUnits(ctx context.Context, userID primitive.ObjectID, recipe *Recipe) (*Recipe, error)

    // GetUnitSystem returns the user's preferred unit system
    GetUnitSystem(ctx context.Context, userID primitive.ObjectID) (UnitPreference, error)

    // GetTemperaturePreference returns user's preferred temperature unit
    GetTemperaturePreference(ctx context.Context, userID primitive.ObjectID) (TemperatureUnit, error)

    // GetCurrencyPreference returns user's preferred currency
    GetCurrencyPreference(ctx context.Context, userID primitive.ObjectID) (string, error)

    // GetThemePreference returns user's preferred theme
    GetThemePreference(ctx context.Context, userID primitive.ObjectID) (ThemePreference, error)

    // GetDietaryRestrictions returns user's dietary restrictions
    GetDietaryRestrictions(ctx context.Context, userID primitive.ObjectID) ([]string, error)

    // GetAllergies returns user's allergies
    GetAllergies(ctx context.Context, userID primitive.ObjectID) ([]string, error)

    // SubscribeToPreferenceChanges returns a channel for preference change events
    SubscribeToPreferenceChanges(ctx context.Context, userID primitive.ObjectID) (<-chan PreferenceEvent, error)
}
```

### 4.2 Private Methods

```go
type preferenceManager struct {
    mongoClient     *mongo.Client
    redisClient     *redis.Client
    preferencesColl *mongo.Collection
    cache           *Cache
    eventPublisher  *EventPublisher
    converter       *UnitConverter
    logger          *log.Logger
}

func (pm *preferenceManager) fetchFromDatabase(ctx context.Context, userID primitive.ObjectID) (*UserPreferences, error) {
    var preferences UserPreferences
    err := pm.preferencesColl.FindOne(ctx, bson.M{"user_id": userID}).Decode(&preferences)
    if err != nil {
        if errors.Is(err, mongo.ErrNoDocuments) {
            return nil, nil
        }
        return nil, fmt.Errorf("database error: %w", err)
    }
    return &preferences, nil
}

func (pm *preferenceManager) createDefaults(ctx context.Context, userID primitive.ObjectID) (*UserPreferences, error) {
    defaults := pm.CreateDefaultPreferences(userID)
    defaults.UserID = userID

    _, err := pm.preferencesColl.InsertOne(ctx, defaults)
    if err != nil {
        return nil, fmt.Errorf("failed to create defaults: %w", err)
    }

    pm.cache.Set(ctx, cacheKey(userID), defaults, time.Hour)
    return defaults, nil
}

func (pm *preferenceManager) applyUpdates(current *UserPreferences, updates *PreferenceUpdateRequest) (*UserPreferences, []string) {
    changed := make([]string, 0)

    if updates.UnitSystem != nil {
        if current.UnitSystem != *updates.UnitSystem {
            changed = append(changed, "unit_system")
        }
        current.UnitSystem = *updates.UnitSystem
    }

    if updates.WeightUnit != nil {
        changed = append(changed, "weight_unit")
        current.WeightUnit = *updates.WeightUnit
    }

    if updates.VolumeUnit != nil {
        changed = append(changed, "volume_unit")
        current.VolumeUnit = *updates.VolumeUnit
    }

    if updates.TemperatureUnit != nil {
        changed = append(changed, "temperature_unit")
        current.TemperatureUnit = *updates.TemperatureUnit
    }

    if updates.Theme != nil {
        changed = append(changed, "theme")
        current.Theme = *updates.Theme
    }

    if updates.Language != nil {
        changed = append(changed, "language")
        current.Language = *updates.Language
    }

    if updates.Timezone != nil {
        changed = append(changed, "timezone")
        current.Timezone = *updates.Timezone
    }

    if updates.Currency != nil {
        changed = append(changed, "currency")
        current.Currency = *updates.Currency
    }

    if updates.Notifications != nil {
        changed = append(changed, "notifications")
        current.Notifications = *updates.Notifications
    }

    if updates.DietaryRestrictions != nil {
        changed = append(changed, "dietary_restrictions")
        current.DietaryRestrictions = *updates.DietaryRestrictions
    }

    if updates.Allergies != nil {
        changed = append(changed, "allergies")
        current.Allergies = *updates.Allergies
    }

    if updates.DefaultServings != nil {
        changed = append(changed, "default_servings")
        current.DefaultServings = *updates.DefaultServings
    }

    if updates.AutoScaleRecipes != nil {
        changed = append(changed, "auto_scale_recipes")
        current.AutoScaleRecipes = *updates.AutoScaleRecipes
    }

    if updates.ShowNutritionalInfo != nil {
        changed = append(changed, "show_nutritional_info")
        current.ShowNutritionalInfo = *updates.ShowNutritionalInfo
    }

    if updates.PreferCheaperAlternatives != nil {
        changed = append(changed, "prefer_cheaper_alternatives")
        current.PreferCheaperAlternatives = *updates.PreferCheaperAlternatives
    }

    if updates.MaxBudgetPerMeal != nil {
        changed = append(changed, "max_budget_per_meal")
        current.MaxBudgetPerMeal = *updates.MaxBudgetPerMeal
    }

    current.UpdatedAt = time.Now()

    return current, changed
}

func (pm *preferenceManager) validateUnitConsistency(prefs *UserPreferences) error {
    if prefs.UnitSystem == UnitPreferenceMetric {
        imperialUnits := []WeightUnit{WeightUnitOunces, WeightUnitPounds}
        for _, unit := range imperialUnits {
            if prefs.WeightUnit == unit {
                return ErrUnitInconsistency
            }
        }
        imperialVolumeUnits := []VolumeUnit{VolumeUnitTeaspoons, VolumeUnitTablespoons, VolumeUnitCups, VolumeUnitFluidOunces}
        for _, unit := range imperialVolumeUnits {
            if prefs.VolumeUnit == unit {
                return ErrUnitInconsistency
            }
        }
    }

    if prefs.UnitSystem == UnitPreferenceImperial {
        metricUnits := []WeightUnit{WeightUnitGrams, WeightUnitKilograms}
        for _, unit := range metricUnits {
            if prefs.WeightUnit == unit {
                return ErrUnitInconsistency
            }
        }
        metricVolumeUnits := []VolumeUnit{VolumeUnitMilliliters, VolumeUnitLiters}
        for _, unit := range metricVolumeUnits {
            if prefs.VolumeUnit == unit {
                return ErrUnitInconsistency
            }
        }
    }

    return nil
}

func (pm *preferenceManager) normalizeUnitsOnSystemChange(prefs *UserPreferences, newSystem UnitPreference) {
    if newSystem == UnitPreferenceMetric {
        switch prefs.WeightUnit {
        case WeightUnitOunces:
            prefs.WeightUnit = WeightUnitGrams
        case WeightUnitPounds:
            prefs.WeightUnit = WeightUnitKilograms
        }
        switch prefs.VolumeUnit {
        case VolumeUnitTeaspoons, VolumeUnitTablespoons, VolumeUnitCups, VolumeUnitFluidOunces:
            prefs.VolumeUnit = VolumeUnitMilliliters
        }
    }

    if newSystem == UnitPreferenceImperial {
        switch prefs.WeightUnit {
        case WeightUnitGrams:
            prefs.WeightUnit = WeightUnitOunces
        case WeightUnitKilograms:
            prefs.WeightUnit = WeightUnitPounds
        }
        switch prefs.VolumeUnit {
        case VolumeUnitMilliliters, VolumeUnitLiters:
            prefs.VolumeUnit = VolumeUnitFluidOunces
        }
    }
}

func (pm *preferenceManager) publishPreferenceEvent(ctx context.Context, event *PreferenceEvent) error {
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }

    return pm.redisClient.Publish(ctx, "preference_events", data).Err()
}
```

### 4.3 Configuration

```go
type PreferenceManagerConfig struct {
    CacheTTL           time.Duration
    MaxRetries         int
    RetryDelay         time.Duration
    MongoCollection    string
    RedisKeyPrefix     string
    EnableMetrics      bool
}
```

### 4.4 Dependencies

```go
type PreferenceManagerDependencies struct {
    MongoClient   *mongo.Client
    RedisClient   *redis.Client
    Logger        *log.Logger
    Config        *PreferenceManagerConfig
}
```

### 4.5 Factory

```go
func NewPreferenceManager(deps *PreferenceManagerDependencies) PreferenceManager {
    collection := deps.MongoClient.Database("mealswapp").Collection("preferences")

    return &preferenceManager{
        mongoClient:     deps.MongoClient,
        redisClient:     deps.RedisClient,
        preferencesColl: collection,
        cache:           NewRedisCache(deps.RedisClient, deps.Config.RedisKeyPrefix, deps.Config.CacheTTL),
        eventPublisher:  NewEventPublisher(deps.RedisClient),
        converter:       NewUnitConverter(),
        logger:          deps.Logger,
    }
}
```
