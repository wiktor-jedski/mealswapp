# DataNormalizer

**Traceability:** ARCH-012

## 1. Data Structures & Types

### 1.1 External API Response Types

```go
type USDASearchResult struct {
    Foods []USDAFoodItem `json:"foods"`
    TotalHits           int    `json:"totalHits"`
    CurrentPage         int    `json:"currentPage"`
}

type USDAFoodItem struct {
    FdcID         int               `json:"fdcId"`
    Description   string            `json:"description"`
    DataType      string            `json:"dataType"`
    PublicationDate string          `json:"publicationDate"`
    FoodNutrients []USDAFoodNutrient `json:"foodNutrients"`
}

type USDAFoodNutrient struct {
    NutrientID    int     `json:"nutrientId"`
    NutrientName  string  `json:"nutrientName"`
    UnitName      string  `json:"unitName"`
    Value         float64 `json:"value"`
}

type OpenFoodFactsResponse struct {
    Products []OpenFoodFactsProduct `json:"product"`
    Count    int                    `json:"count"`
    Page     int                    `json:"page"`
    PageSize int                    `json:"page_size"`
}

type OpenFoodFactsProduct struct {
    Code          string                   `json:"code"`
    ProductName   string                   `json:"product_name"`
    Brands        string                   `json:"brands"`
    Categories    string                   `json:"categories"`
    ImageURL      string                   `json:"image_url"`
    Nutriments     OpenFoodFactsNutriments `json:"nutriments"`
    NutrientLevels *NutrientLevels         `json:"nutrient_levels"`
    ServingSize   string                   `json:"serving_size"`
    ServingQuantity float64                `json:"serving_quantity"`
}

type OpenFoodFactsNutriments struct {
    EnergyKcal            *float64 `json:"energy-kcal"`
    EnergyKcal100         *float64 `json:"energy-kcal_100g"`
    Proteins              *float64 `json:"proteins_100g"`
    Carbohydrates         *float64 `json:"carbohydrates_100g"`
    Fat                   *float64 `json:"fat_100g"`
    Fiber                 *float64 `json:"fiber_100g"`
    Sugars                *float64 `json:"sugars_100g"`
    SaturatedFat          *float64 `json:"saturated-fat_100g"`
    Sodium                *float64 `json:"sodium_100g"`
    Cholesterol           *float64 `json:"cholesterol_100g"`
    VitaminA              *float64 `json:"vitamin-a_100g"`
    VitaminC              *float64 `json:"vitamin-c_100g"`
    Calcium               *float64 `json:"calcium_100g"`
    Iron                  *float64 `json:"iron_100g"`
}

type NutrientLevels struct {
    Low    string `json:"low"`
    Medium string `json:"medium"`
    High   string `json:"high"`
}
```

### 1.2 Internal Normalized Types

```go
type NormalizedFoodItem struct {
    ExternalSource    string              `json:"external_source"`
    ExternalID        string              `json:"external_id"`
    Name              string              `json:"name"`
    Brand             string              `json:"brand,omitempty"`
    Category          string              `json:"category,omitempty"`
    ImageURL          string              `json:"image_url,omitempty"`
    ServingSizeGrams  float64             `json:"serving_size_grams"`
    Calories          float64             `json:"calories"`
    Protein           float64             `json:"protein"`
    Carbohydrates     float64             `json:"carbohydrates"`
    Fat               float64             `json:"fat"`
    Fiber             float64             `json:"fiber"`
    Sugars            float64             `json:"sugars"`
    SaturatedFat      float64             `json:"saturated_fat"`
    Sodium            float64             `json:"sodium"`
    Cholesterol       float64             `json:"cholesterol"`
    Vitamins          map[string]float64  `json:"vitamins,omitempty"`
    Confidence        float64             `json:"confidence"`
    RawData           json.RawMessage     `json:"raw_data"`
}

type NormalizationResult struct {
    Items      []NormalizedFoodItem
    Source     string
    Page       int
    TotalCount int
    Errors     []NormalizationError
}

type NormalizationError struct {
    ItemID    string `json:"item_id"`
    Source    string `json:"source"`
    ErrorType string `json:"error_type"`
    Message   string `json:"message"`
}
```

### 1.3 Unit Conversion Types

```go
type UnitConversion struct {
    FromUnit   string
    ToGrams    float64
    IsVolume   bool
}

var UnitConversions = map[string]UnitConversion{
    "g":      {FromUnit: "g", ToGrams: 1.0, IsVolume: false},
    "gram":   {FromUnit: "gram", ToGrams: 1.0, IsVolume: false},
    "grams":  {FromUnit: "grams", ToGrams: 1.0, IsVolume: false},
    "kg":     {FromUnit: "kg", ToGrams: 1000.0, IsVolume: false},
    "oz":     {FromUnit: "oz", ToGrams: 28.3495, IsVolume: false},
    "ounce":  {FromUnit: "ounce", ToGrams: 28.3495, IsVolume: false},
    "lb":     {FromUnit: "lb", ToGrams: 453.592, IsVolume: false},
    "cup":    {FromUnit: "cup", ToGrams: 240.0, IsVolume: true},
    "tbsp":   {FromUnit: "tbsp", ToGrams: 15.0, IsVolume: true},
    "tsp":    {FromUnit: "tsp", ToGrams: 5.0, IsVolume: true},
    "ml":     {FromUnit: "ml", ToGrams: 1.0, IsVolume: true},
    "l":      {FromUnit: "l", ToGrams: 1000.0, IsVolume: true},
    "liter":  {FromUnit: "liter", ToGrams: 1000.0, IsVolume: true},
    "fl oz":  {FromUnit: "fl oz", ToGrams: 29.5735, IsVolume: true},
}
```

### 1.4 Nutrient Mapping Types

```go
type NutrientMapping struct {
    ExternalName  string
    ExternalUnit  string
    InternalField string
    ConversionFactor float64
}

var USDA nutrientMappings = []NutrientMapping{
    {ExternalName: "Energy", ExternalUnit: "KCAL", InternalField: "Calories", ConversionFactor: 1.0},
    {ExternalName: "Protein", ExternalUnit: "G", InternalField: "Protein", ConversionFactor: 1.0},
    {ExternalName: "Total lipid (fat)", ExternalUnit: "G", InternalField: "Fat", ConversionFactor: 1.0},
    {ExternalName: "Carbohydrate, by difference", ExternalUnit: "G", InternalField: "Carbohydrates", ConversionFactor: 1.0},
    {ExternalName: "Fiber, total dietary", ExternalUnit: "G", InternalField: "Fiber", ConversionFactor: 1.0},
    {ExternalName: "Sugars, total including NLEA", ExternalUnit: "G", InternalField: "Sugars", ConversionFactor: 1.0},
    {ExternalName: "Fatty acids, total saturated", ExternalUnit: "G", InternalField: "SaturatedFat", ConversionFactor: 1.0},
    {ExternalName: "Sodium, Na", ExternalUnit: "MG", InternalField: "Sodium", ConversionFactor: 0.001},
    {ExternalName: "Cholesterol", ExternalUnit: "MG", InternalField: "Cholesterol", ConversionFactor: 0.001},
    {ExternalName: "Vitamin A,RAE", ExternalUnit: "UG", InternalField: "Vitamins[vitamin_a]", ConversionFactor: 1.0},
    {ExternalName: "Vitamin C, total ascorbic acid", ExternalUnit: "MG", InternalField: "Vitamins[vitamin_c]", ConversionFactor: 0.001},
    {ExternalName: "Calcium, Ca", ExternalUnit: "MG", InternalField: "Vitamins[calcium]", ConversionFactor: 0.001},
    {ExternalName: "Iron, Fe", ExternalUnit: "MG", InternalField: "Vitamins[iron]", ConversionFactor: 0.001},
}

var OpenFoodFactsNutrientMappings = map[string]string{
    "energy-kcal":            "Calories",
    "energy-kcal_100g":       "Calories",
    "proteins_100g":          "Protein",
    "carbohydrates_100g":     "Carbohydrates",
    "fat_100g":               "Fat",
    "fiber_100g":             "Fiber",
    "sugars_100g":            "Sugars",
    "saturated-fat_100g":     "SaturatedFat",
    "sodium_100g":            "Sodium",
    "cholesterol_100g":       "Cholesterol",
    "vitamin-a_100g":         "Vitamins[vitamin_a]",
    "vitamin-c_100g":         "Vitamins[vitamin_c]",
    "calcium_100g":           "Vitamins[calcium]",
    "iron_100g":              "Vitamins[iron]",
}
```

## 2. Logic & Algorithms

### 2.1 Main Normalization Pipeline

```
Algorithm: NormalizeExternalData
Input: source (string), rawData (interface{})
Output: NormalizedFoodItem or error

1.  source ← source parameter
2.  switch source:
3.      case "usda":
4.          items ← NormalizeUSDADATA(rawData)
5.      case "openfoodfacts":
6.          items ← NormalizeOpenFoodFactsDATA(rawData)
7.      default:
8.          return UnsupportedSourceError{source}
9.  return items
```

### 2.2 USDA Data Normalization

```
Algorithm: NormalizeUSDADATA
Input: searchResult (USDASearchResult)
Output: []NormalizedFoodItem

1.  normalizedItems ← empty list
2.  for each foodItem in searchResult.Foods:
3.        normalized ← create empty NormalizedFoodItem
4.        normalized.ExternalSource ← "usda"
5.        normalized.ExternalID ← string(foodItem.FdcID)
6.        normalized.Name ← NormalizeUSDAFoodName(foodItem.Description)
7.        normalized.RawData ← marshal(foodItem)
8.        for each nutrient in foodItem.FoodNutrients:
9.              mapping ← FindUSDAMapping(nutrient.NutrientName, nutrient.UnitName)
10.             if mapping exists:
11.                   value ← ConvertToGrams(nutrient.Value, mapping.ExternalUnit)
12.                   SetField(normalized, mapping.InternalField, value)
13.        normalized.Confidence ← CalculateUSDANutrientCompleteness(normalized)
14.        normalized ← ResolveUSDAStandardPortion(normalized, foodItem)
15.        Append normalizedItems, normalized
16. return normalizedItems
```

### 2.3 OpenFoodFacts Data Normalization

```
Algorithm: NormalizeOpenFoodFactsDATA
Input: response (OpenFoodFactsResponse)
Output: []NormalizedFoodItem

1.  normalizedItems ← empty list
2.  for each product in response.Products:
3.        if product.ProductName is empty:
4.              continue to next product
5.        normalized ← create empty NormalizedFoodItem
6.        normalized.ExternalSource ← "openfoodfacts"
7.        normalized.ExternalID ← product.Code
8.        normalized.Name ← product.ProductName
9.        normalized.Brand ← ExtractBrand(product.Brands)
10.       normalized.Category ← ParseFirstCategory(product.Categories)
11.       normalized.ImageURL ← product.ImageURL
12.       normalized.RawData ← marshal(product)
13.       normalized ← MapOpenFoodFactsNutrients(normalized, product.Nutriments)
14.       normalized.ServingSizeGrams ← ParseServingSize(product.ServingSize, product.ServingQuantity)
15.       normalized.Confidence ← CalculateOpenFoodFactsQualityScore(product)
16.       if normalized.ServingSizeGrams > 0:
17.             normalized ← NormalizeToPer100g(normalized)
18.       Append normalizedItems, normalized
19. return normalizedItems
```

### 2.4 Unit Conversion Algorithm

```
Algorithm: ConvertToGrams
Input: value (float64), unit (string)
Output: float64 (value in grams) or error

1.  if unit is empty or unit equals "G":
2.        return value
3.  conversion ← UnitConversions[unit]
4.  if conversion does not exist:
5.        log warning: "Unknown unit: " + unit
6.        return value
7.  return value * conversion.ToGrams
```

### 2.5 Serving Size Parsing Algorithm

```
Algorithm: ParseServingSize
Input: servingSizeStr (string), servingQty (float64)
Output: float64 (grams)

1.  if servingQty > 0 and servingSizeStr is empty:
2.        return servingQty * 100
3.  if servingSizeStr is empty:
4.        return 100.0
5.  parsed ← ParseMeasurement(servingSizeStr)
6.  if parsed.value > 0 and parsed.unit exists:
7.        grams ← ConvertToGrams(parsed.value, parsed.unit)
8.        return grams
9.  if servingQty > 0:
10.       return servingQty * 100
11. return 100.0
```

### 2.6 Per-100g Normalization Algorithm

```
Algorithm: NormalizeToPer100g
Input: item (NormalizedFoodItem)
Output: NormalizedFoodItem

1.  if item.ServingSizeGrams equals 0:
2.        return item
3.  factor ← 100.0 / item.ServingSizeGrams
4.  item.Calories ← item.Calories * factor
5.  item.Protein ← item.Protein * factor
6.  item.Carbohydrates ← item.Carbohydrates * factor
7.  item.Fat ← item.Fat * factor
8.  item.Fiber ← item.Fiber * factor
9.  item.Sugars ← item.Sugars * factor
10. item.SaturatedFat ← item.SaturatedFat * factor
11. item.Sodium ← item.Sodium * factor
12. item.Cholesterol ← item.Cholesterol * factor
13. for each vitamin, value in item.Vitamins:
14.       item.Vitamins[vitamin] ← value * factor
15. item.ServingSizeGrams ← 100.0
16. return item
```

### 2.7 Name Normalization Algorithm

```
Algorithm: NormalizeUSDAFoodName
Input: description (string)
Output: string

1.  cleaned ← Trim(description)
2.  cleaned ← RemoveExtraSpaces(cleaned)
3.  cleaned ← strings.ToLower(cleaned)
4.  cleaned ← strings.Title(cleaned)
5.  prefixes ← ["cooked", "raw", "dried", "fresh", "frozen", "canned"]
6.  for each prefix in prefixes:
7.        if strings.HasPrefix(cleaned, prefix + " "):
8.              cleaned ← strings.TrimPrefix(cleaned, prefix + " ")
9.  return strings.TrimSpace(cleaned)
```

### 2.8 Confidence Scoring Algorithm

```
Algorithm: CalculateUSDANutrientCompleteness
Input: item (NormalizedFoodItem)
Output: float64 (0.0 to 1.0)

1.  requiredFields ← ["Calories", "Protein", "Carbohydrates", "Fat"]
2.  presentCount ← 0
3.  for each field in requiredFields:
4.        if GetField(item, field) > 0:
5.              presentCount ← presentCount + 1
6.  baseScore ← presentCount / length(requiredFields)
7.  bonusFields ← ["Fiber", "Sugars", "SaturatedFat", "Sodium"]
8.  bonusCount ← 0
9.  for each field in bonusFields:
10.       if GetField(item, field) > 0:
11.             bonusCount ← bonusCount + 1
12. bonusScore ← (bonusCount / length(bonusFields)) * 0.3
13. return min(baseScore + bonusScore, 1.0)
```

### 2.9 OpenFoodFacts Quality Scoring Algorithm

```
Algorithm: CalculateOpenFoodFactsQualityScore
Input: product (OpenFoodFactsProduct)
Output: float64 (0.0 to 1.0)

1.  score ← 0.0
2.  if product.ProductName is not empty:
3.        score ← score + 0.2
4.  if product.Brands is not empty:
5.        score ← score + 0.1
6.  if product.ImageURL is not empty:
7.        score ← score + 0.1
8.  if product.Nutriments.EnergyKcal != nil or product.Nutriments.EnergyKcal100 != nil:
9.        score ← score + 0.15
10. nonNullNutrients ← 0
11. expectedNutrients ← ["Proteins", "Carbohydrates", "Fat"]
12. for each nutrient in expectedNutrients:
13.       if GetNutrient(product.Nutriments, nutrient) != nil:
14.             nonNullNutrients ← nonNullNutrients + 1
15. score ← score + (nonNullNutrients / length(expectedNutrients)) * 0.3
16. if product.NutrientLevels != nil:
17.       score ← score + 0.15
18. return min(score, 1.0)
```

## 3. State Management & Error Handling

### 3.1 Error Types

```go
var (
    ErrUnsupportedSource = errors.New("unsupported external data source")
    ErrInvalidInput      = errors.New("invalid input data for normalization")
    ErrMissingRequiredFields = errors.New("missing required fields in external data")
    ErrUnitConversion    = errors.New("failed to convert units")
    ErrParsingError      = errors.New("failed to parse serving size or measurement")
)

type NormalizationError struct {
    Source    string
    ItemID    string
    ErrorType string
    Message   string
    Err       error
}
```

### 3.2 Error Handling States

| State | Condition | Handling |
|:------|:----------|:---------|
| **UnsupportedSource** | Source not "usda" or "openfoodfacts" | Return error, log warning, do not retry |
| **InvalidInput** | JSON unmarshal fails | Return error, log with raw payload, do not retry |
| **MissingRequiredFields** | Name or core nutrients missing | Log warning, skip item, continue with others |
| **UnitConversionFailed** | Unknown unit in conversion | Log warning, set value to 0, continue |
| **ServingSizeInvalid** | Cannot parse serving size | Use default 100g, log info |
| **PartialNormalization** | Some nutrients failed to map | Continue with available nutrients, log warnings |

### 3.3 State Transitions

```
State Machine: NormalizationSession

States:
  - IDLE (initial state)
  - FETCHING (deprecated, handled by client)
  - NORMALIZING
  - COMPLETED
  - FAILED

Transitions:

IDLE → NORMALIZING:
  Trigger: NormalizeExternalData() called
  Action: Initialize result slice, set source

NORMALIZING → COMPLETED:
  Trigger: All items processed
  Action: Return NormalizationResult with items and errors

NORMALIZING → FAILED:
  Trigger: UnsupportedSource or InvalidInput error
  Action: Return error immediately, skip processing

NORMALIZING → NORMALIZING (self-loop):
  Trigger: Individual item error
  Action: Log error, add to result.Errors, continue with next item
```

### 3.4 Error Recovery Strategy

```
Error Recovery: Normalization Process

1.  On UnsupportedSource:
    - Log error with source parameter
    - Return wrapped error immediately
    - No retry (configuration issue)

2.  On InvalidInput:
    - Log error with payload snippet
    - Return error with context
    - No retry (malformed input)

3.  On MissingRequiredFields:
    - Log warning with item ID
    - Skip item in processing
    - Add to Errors slice in result
    - Continue with remaining items

4.  On UnitConversionFailed:
    - Log warning with unit name
    - Set nutrient value to 0
    - Continue normalization
    - Add warning to Errors slice

5.  On ServingSizeInvalid:
    - Log info message
    - Use default 100g
    - Continue normalization
```

### 3.5 Validation Rules

```go
type ValidationRule struct {
    Field    string
    Check    func(item NormalizedFoodItem) bool
    Message  string
}

var ValidationRules = []ValidationRule{
    {
        Field: "Name",
        Check: func(item NormalizedFoodItem) bool {
            return len(item.Name) >= 2 && len(item.Name) <= 200
        },
        Message: "name must be between 2 and 200 characters",
    },
    {
        Field: "ServingSizeGrams",
        Check: func(item NormalizedFoodItem) bool {
            return item.ServingSizeGrams > 0 && item.ServingSizeGrams <= 10000
        },
        Message: "serving size must be between 0 and 10000 grams",
    },
    {
        Field: "Nutrients",
        Check: func(item NormalizedFoodItem) bool {
            allNutrientsNonNegative := item.Calories >= 0 &&
                item.Protein >= 0 &&
                item.Carbohydrates >= 0 &&
                item.Fat >= 0 &&
                item.Fiber >= 0 &&
                item.Sodium >= 0
            return allNutrientsNonNegative
        },
        Message: "all nutrient values must be non-negative",
    },
}
```

## 4. Component Interfaces

### 4.1 Public Interface

```go
type DataNormalizer interface {
    Normalize(source string, rawData interface{}) (*NormalizationResult, error)
    NormalizeUSDA(searchResult USDASearchResult) ([]NormalizedFoodItem, error)
    NormalizeOpenFoodFacts(response OpenFoodFactsResponse) ([]NormalizedFoodItem, error)
    Validate(item NormalizedFoodItem) []ValidationError
    CalculateConfidence(item NormalizedFoodItem) float64
}
```

### 4.2 Internal Functions

```go
type dataNormalizer struct {
    logger      *log.Logger
    unitConverter UnitConverter
    validator    Validator
}

func NewDataNormalizer(logger *log.Logger) DataNormalizer {
    return &dataNormalizer{
        logger: logger,
        unitConverter: NewUnitConverter(),
        validator: NewValidator(),
    }
}
```

### 4.3 Function Signatures

```go
func (dn *dataNormalizer) Normalize(source string, rawData interface{}) (*NormalizationResult, error)

func (dn *dataNormalizer) NormalizeUSDA(searchResult USDASearchResult) ([]NormalizedFoodItem, error)

func (dn *dataNormalizer) NormalizeOpenFoodFacts(response OpenFoodFactsResponse) ([]NormalizedFoodItem, error)

func (dn *dataNormalizer) Validate(item NormalizedFoodItem) []ValidationError

func (dn *dataNormalizer) CalculateConfidence(item NormalizedFoodItem) float64

func (dn *dataNormalizer) normalizeUSDANutrient(
    nutrient USDAFoodNutrient,
    item *NormalizedFoodItem,
)

func (dn *dataNormalizer) mapOpenFoodFactsNutrient(
    key string,
    value float64,
    item *NormalizedFoodItem,
)

func (dn *dataNormalizer) parseServingSize(sizeStr string, quantity float64) (float64, error)

func (dn *dataNormalizer) convertToGrams(value float64, unit string) (float64, error)

func (dn *dataNormalizer) normalizeToPer100g(item NormalizedFoodItem) NormalizedFoodItem

func (dn *dataNormalizer) calculateUSDANutrientCompleteness(item NormalizedFoodItem) float64

func (dn *dataNormalizer) calculateOpenFoodFactsQuality(product OpenFoodFactsProduct) float64

func (dn *dataNormalizer) normalizeName(name string) string

func (dn *dataNormalizer) extractBrand(brands string) string

func (dn *dataNormalizer) parseCategories(categories string) string
```

### 4.4 Usage Example

```go
func (dn *dataNormalizer) Normalize(source string, rawData interface{}) (*NormalizationResult, error) {
    result := &NormalizationResult{
        Source: source,
        Items:  make([]NormalizedFoodItem, 0),
        Errors: make([]NormalizationError, 0),
    }

    var err error
    switch source {
    case "usda":
        searchResult, ok := rawData.(USDASearchResult)
        if !ok {
            return nil, &NormalizationError{
                Source:    source,
                ErrorType: "InvalidInput",
                Message:   "failed to cast rawData to USDASearchResult",
            }
        }
        items, normErr := dn.NormalizeUSDA(searchResult)
        result.Items = items
        result.Errors = append(result.Errors, normErr...)
    case "openfoodfacts":
        response, ok := rawData.(OpenFoodFactsResponse)
        if !ok {
            return nil, &NormalizationError{
                Source:    source,
                ErrorType: "InvalidInput",
                Message:   "failed to cast rawData to OpenFoodFactsResponse",
            }
        }
        items, normErr := dn.NormalizeOpenFoodFacts(response)
        result.Items = items
        result.Errors = append(result.Errors, normErr...)
    default:
        return nil, &NormalizationError{
            Source:    source,
            ErrorType: "UnsupportedSource",
            Message:   fmt.Sprintf("source '%s' is not supported", source),
        }
    }

    return result, nil
}
```

### 4.5 Integration Points

```go
type ExternalDataClient interface {
    FetchUSDA(query string, page int) (*USDASearchResult, error)
    FetchOpenFoodFacts(query string, page int) (*OpenFoodFactsResponse, error)
}

type DataRepository interface {
    SaveNormalizedItems(items []NormalizedFoodItem) error
}

func ProcessExternalQuery(
    client ExternalDataClient,
    normalizer DataNormalizer,
    repo DataRepository,
    query string,
    source string,
) ([]NormalizedFoodItem, error) {
    var rawData interface{}
    var err error

    if source == "usda" {
        rawData, err = client.FetchUSDA(query, 1)
    } else {
        rawData, err = client.FetchOpenFoodFacts(query, 1)
    }

    if err != nil {
        return nil, fmt.Errorf("failed to fetch from %s: %w", source, err)
    }

    result, err := normalizer.Normalize(source, rawData)
    if err != nil {
        return nil, fmt.Errorf("normalization failed: %w", err)
    }

    validationErrors := make([]NormalizedFoodItem, 0, len(result.Items))
    for _, item := range result.Items {
        if len(normalizer.Validate(item)) == 0 {
            validationErrors = append(validationErrors, item)
        }
    }

    if len(validationErrors) > 0 {
        if saveErr := repo.SaveNormalizedItems(validationErrors); saveErr != nil {
            return nil, fmt.Errorf("failed to save normalized items: %w", saveErr)
        }
    }

    return result.Items, nil
}
```
