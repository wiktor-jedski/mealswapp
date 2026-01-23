# FILE: OpenFoodFactsClient.md
**Traceability:** ARCH-012

## 1. Data Structures & Types

```go
package openfoodfacts

import "time"

// OpenFoodFactsConfig holds configuration for the OpenFoodFacts API
type OpenFoodFactsConfig struct {
    BaseURL     string
    Timeout     time.Duration
    MaxRetries  int
    UserAgent   string
    Country     string // e.g., "us", "world"
    Language    string // e.g., "en"
}

// OpenFoodFactsProduct represents raw response from OpenFoodFacts API
type OpenFoodFactsProduct struct {
    ProductName      string                   `json:"product_name"`
    GenericName      string                   `json:"generic_name"`
    Code             string                   `json:"code"`
    Barcode          string                   `json:"_id"`
    Brands           string                   `json:"brands"`
    Categories       string                   `json:"categories"`
    CategoriesTags   []string                 `json:"categories_tags"`
    Nutrients        OpenFoodFactsNutrients   `json:"nutriments"`
    ServingSize      string                   `json:"serving_size"`
    ServingQuantity  *float64                 `json:"serving_quantity"`
    ImageURL         string                   `json:"image_url"`
    ImageSmallURL    string                   `json:"image_small_url"`
    Ingredients      []OpenFoodFactsIngredient `json:"ingredients"`
    IngredientsText  string                   `json:"ingredients_text"`
    IngredientsTags  []string                 `json:"ingredients_tags"`
    AdditivesTags    []string                 `json:"additives_tags"`
    NutrientLevels   *OpenFoodFactsNutrientLevels `json:"nutrient_level"`
    NutritionGrade   string                   `json:"nutrition_grade"`
    nutriments       OpenFoodFactsNutrients   `json:"nutriments"`
    CreatedAt        time.Time                `json:"created_t"`
    LastModified     time.Time                `json:"last_modified_t"`
    Countries        string                   `json:"countries"`
    CountriesTags    []string                 `json:"countries_tags"`
}

// OpenFoodFactsNutrients represents nutrient values from OpenFoodFacts
type OpenFoodFactsNutrients struct {
    EnergyKcal          *float64 `json:"energy-kcal"`
    EnergyKcal100       *float64 `json:"energy-kcal_100g"`
    EnergyKj            *float64 `json:"energy"`
    EnergyKj100         *float64 `json:"energy_100g"`
    Proteins            *float64 `json:"proteins"`
    Proteins100         *float64 `json:"proteins_100g"`
    Carbohydrates       *float64 `json:"carbohydrates"`
    Carbohydrates100    *float64 `json:"carbohydrates_100g"`
    Sugars              *float64 `json:"sugars"`
    Sugars100           *float64 `json:"sugars_100g"`
    Fiber               *float64 `json:"fiber"`
    Fiber100            *float64 `json:"fiber_100g"`
    Fat                 *float64 `json:"fat"`
    Fat100              *float64 `json:"fat_100g"`
    SaturatedFat        *float64 `json:"saturated-fat"`
    SaturatedFat100     *float64 `json:"saturated-fat_100g"`
    TransFat            *float64 `json:"trans-fat"`
    TransFat100         *float64 `json:"trans-fat_100g"`
    Cholesterol         *float64 `json:"cholesterol"`
    Cholesterol100      *float64 `json:"cholesterol_100g"`
    Sodium              *float64 `json:"sodium"`
    Sodium100           *float64 `json:"sodium_100g"`
    VitaminA            *float64 `json:"vitamin-a"`
    VitaminA100         *float64 `json:"vitamin-a_100g"`
    VitaminC            *float64 `json:"vitamin-c"`
    VitaminC100         *float64 `json:"vitamin-c_100g"`
    Calcium             *float64 `json:"calcium"`
    Calcium100          *float64 `json:"calcium_100g"`
    Iron                *float64 `json:"iron"`
    Iron100             *float64 `json:"iron_100g"`
    Potassium           *float64 `json:"potassium"`
    Potassium100        *float64 `json:"potassium_100g"`
    Salt                *float64 `json:"salt"`
    Salt100             *float64 `json:"salt_100g"`
}

// OpenFoodFactsIngredient represents a single ingredient
type OpenFoodFactsIngredient struct {
    ID        string  `json:"id"`
    Text      string  `json:"text"`
    Percent   *float64 `json:"percent"`
    PercentMin *float64 `json:"percent_min"`
    PercentMax *float64 `json:"percent_max"`
    Vegan     string  `json:"vegan"`
    Vegetarian string  `json:"vegetarian"`
}

// OpenFoodFactsNutrientLevels represents nutrient level indicators
type OpenFoodFactsNutrientLevels struct {
    Salt       string `json:"salt"`
    Saturated  string `json:"saturated-fat"`
    Sugars     string `json:"sugars"`
    Fat        string `json:"fat"`
}

// OpenFoodFactsSearchResponse represents the search API response
type OpenFoodFactsSearchResponse struct {
    Count        int                     `json:"count"`
    Page         int                     `json:"page"`
    PageSize     int                     `json:"page_size"`
    Products     []OpenFoodFactsProduct `json:"products"`
    Skip         int                     `json:"skip"`
    Status       int                     `json:"status"`
    Version      string                  `json:"version"`
}

// OpenFoodFactsSearchParams defines search parameters for OpenFoodFacts API
type OpenFoodFactsSearchParams struct {
    Query        string
    Page         int
    PageSize     int
    Category     string
    Brand        string
    Country      string
    Language     string
    AdditiveTags []string
    NutritionGrades []string // "a", "b", "c", "d", "e"
}

// NormalizedFoodItem represents normalized food data for internal use
type NormalizedFoodItem struct {
    ExternalID         string
    Source             string // "OpenFoodFacts"
    Name               string
    DefaultServingGrams float64
    Calories           *float64
    Protein            *float64
    Carbs              *float64
    Fat                *float64
    Fiber              *float64
    Sugar              *float64
    Sodium             *float64
    Cholesterol        *float64
    SaturatedFat       *float64
    TransFat           *float64
    Potassium          *float64
    VitaminA           *float64
    VitaminC           *float64
    Calcium            *float64
    Iron               *float64
    Ingredients        []string
    ServingSizeText    string
    NutritionGrade     string
    ImageURL           string
    Categories         []string
    Brands             []string
    Barcode            string
    LastUpdated        time.Time
}

// Client wraps OpenFoodFacts API interactions
type Client struct {
    config     OpenFoodFactsConfig
    httpClient *http.Client
    rateLimiter *RateLimitHandler
}

// RateLimitHandler manages API rate limits
type RateLimitHandler struct {
    requestsPerSecond float64
    lastRequestTime   time.Time
    mu                sync.Mutex
}

// OpenFoodFactsError represents errors from OpenFoodFacts API
type OpenFoodFactsError struct {
    Code    int
    Message string
    Details interface{}
}

func (e *OpenFoodFactsError) Error() string {
    return fmt.Sprintf("OpenFoodFacts API error (%d): %s", e.Code, e.Message)
}
```

## 2. Logic & Algorithms

### 2.1 Search Foods

```
FUNCTION SearchFoods(params OpenFoodFactsSearchParams) -> ([]NormalizedFoodItem, error)

1. Validate input parameters
   - IF params.Query is empty AND params.Category is empty AND params.Brand is empty
     THEN RETURN error "at least one search parameter required"
   - IF params.Page < 1 THEN params.Page = 1
   - IF params.PageSize not in [1, 100] THEN params.PageSize = 25
   - IF params.PageSize > 100 THEN params.PageSize = 100

2. Construct API endpoint
   - endpoint = config.BaseURL + "/cgi/search.pl"
   - queryParams = {
        "search_terms": params.Query,
        "search_simple": 1,
        "action": "process",
        "page": params.Page,
        "page_size": params.PageSize,
        "json": 1,
        "lang": params.Language,
        "country": params.Country,
      }
   - IF params.Category is not empty THEN add "tagtype_0=categories&tag_contains_0=" + params.Category
   - IF params.Brand is not empty THEN add "tagtype_0=brands&tag_contains_0=" + params.Brand
   - IF params.NutritionGrades not empty THEN add "nutriscores=" + join(params.NutritionGrades, ",")

3. Set request headers
   - "User-Agent": config.UserAgent + " (Mealswapp)"
   - "Accept-Language": config.Language

4. Wait for rate limit availability
   - CALL rateLimiter.Wait()

5. Execute HTTP GET request
   - timeout = config.Timeout
   - IF request fails THEN
     - retry up to config.MaxRetries times with exponential backoff
     - IF all retries fail THEN RETURN error

6. Parse response body into OpenFoodFactsSearchResponse
   - IF parse fails THEN RETURN error "invalid response format"

7. FOR EACH product in response.Products:
   - IF product.ProductName is empty THEN skip
   - normalized = CALL NormalizeProduct(product)
   - append normalized to results

8. RETURN results, nil
```

### 2.2 Get Product By Barcode

```
FUNCTION GetProductByBarcode(barcode string) -> (*NormalizedFoodItem, error)

1. Construct API endpoint
   - endpoint = config.BaseURL + "/api/v0/product/" + barcode + ".json"

2. Set request headers
   - "User-Agent": config.UserAgent + " (Mealswapp)"
   - "Accept-Language": config.Language

3. Wait for rate limit availability
   - CALL rateLimiter.Wait()

4. Execute HTTP GET request
   - timeout = config.Timeout
   - IF request fails THEN
     - retry up to config.MaxRetries times
     - IF all retries fail THEN RETURN error

5. Parse response body into OpenFoodFactsProduct
   - IF parse fails THEN RETURN error "invalid product format"

6. Check API status
   - IF response.Status != 1 THEN RETURN error "product not found"

7. Normalize the product
   - normalized = CALL NormalizeProduct(product)

8. RETURN normalized, nil
```

### 2.3 Normalize Product

```
FUNCTION NormalizeProduct(product OpenFoodFactsProduct) -> NormalizedFoodItem

1. Initialize normalized item
   - normalized = NormalizedFoodItem{
        ExternalID:    product.Code,
        Source:        "OpenFoodFacts",
        Name:          product.ProductName,
        Barcode:       product.Barcode,
        LastUpdated:   product.LastModified,
        ServingSizeText: product.ServingSize,
        NutritionGrade: strings.ToUpper(product.NutritionGrade),
        ImageURL:      product.ImageURL,
      }

2. Calculate default serving in grams
   - IF product.ServingQuantity != nil THEN
     - normalized.DefaultServingGrams = *product.ServingQuantity
   - ELSE IF product.ServingSize is not empty THEN
     - normalized.DefaultServingGrams = CALL EstimateGramsFromText(product.ServingSize)
   - ELSE
     - normalized.DefaultServingGrams = 100.0 // fallback to 100g

3. Extract nutrients from per 100g values
   - nutrients = product.Nutrients
   - IF nutrients.EnergyKcal100 != nil THEN
     - normalized.Calories = nutrients.EnergyKcal100
   - ELSE IF nutrients.EnergyKcal != nil THEN
     - IF normalized.DefaultServingGrams > 0 THEN
       - normalized.Calories = *nutrients.EnergyKcal * (100.0 / normalized.DefaultServingGrams)

   - normalized.Protein = nutrients.Proteins100
   - normalized.Carbs = nutrients.Carbohydrates100
   - normalized.Fat = nutrients.Fat100
   - normalized.Fiber = nutrients.Fiber100
   - normalized.Sugar = nutrients.Sugars100
   - normalized.Sodium = multiplyOrNil(nutrients.Sodium100, 2.5) // convert sodium to salt equivalent if needed
   - normalized.Cholesterol = nutrients.Cholesterol100
   - normalized.SaturatedFat = nutrients.SaturatedFat100
   - normalized.TransFat = nutrients.TransFat100
   - normalized.Potassium = nutrients.Potassium100
   - normalized.VitaminA = nutrients.VitaminA100
   - normalized.VitaminC = nutrients.VitaminC100
   - normalized.Calcium = nutrients.Calcium100
   - normalized.Iron = nutrients.Iron100

4. Parse ingredients
   - IF product.IngredientsText is not empty THEN
     - normalized.Ingredients = CALL ParseIngredients(product.IngredientsText)
   - ELSE IF product.Ingredients is not empty THEN
     - FOR EACH ingredient in product.Ingredients:
       - append ingredient.Text to normalized.Ingredients

5. Parse categories
   - normalized.Categories = product.CategoriesTags

6. Parse brands
   - IF product.Brands is not empty THEN
     - normalized.Brands = strings.Split(product.Brands, ",")

7. RETURN normalized
```

### 2.4 Rate Limiter Wait

```
FUNCTION RateLimitHandler.Wait()

1. Acquire lock
   - rateLimiter.mu.Lock()
   - DEFER rateLimiter.mu.Unlock()

2. Calculate time since last request
   - elapsed = time.Since(rateLimiter.lastRequestTime)
   - requiredInterval = 1.0 / rateLimiter.requestsPerSecond

3. IF elapsed < requiredInterval THEN
   - sleepDuration = requiredInterval - elapsed
   - time.Sleep(sleepDuration)

4. Update last request time
   - rateLimiter.lastRequestTime = time.Now()
```

## 3. State Management & Error Handling

### 3.1 Possible Error States

| Error | Cause | Handling |
|-------|-------|----------|
| `ErrEmptySearch` | No search parameters provided | Validate input before API call, return user-friendly error |
| `ErrRateLimitExceeded` | Too many requests in short window | Implement exponential backoff, return retry-after header |
| `ErrAPITimeout` | OpenFoodFacts API not responding within timeout | Retry with backoff, then return timeout error |
| `ErrProductNotFound` | Product with barcode doesn't exist | Return nil result (not an error for lookups) |
| `ErrInvalidResponse` | API returned unexpected format | Log response body, return parsing error |
| `ErrServiceUnavailable` | OpenFoodFacts API returns 503 | Retry up to MaxRetries, then graceful degradation |
| `ErrNetworkError` | Network connectivity issue | Retry with backoff, return network error after retries |
| `ErrMaxRetriesExceeded` | All retry attempts failed | Log final error, return wrapped error with attempt count |

### 3.2 State Transitions

```
Initial State: IDLE

IDLE -> VALIDATING: SearchFoods() or GetProductByBarcode() called
VALIDATING -> API_CALL: Parameters validated successfully
VALIDATING -> ERROR: Invalid parameters

API_CALL -> RATE_LIMIT_WAIT: Need to wait for rate limit
RATE_LIMIT_WAIT -> API_CALL: Rate limit satisfied

API_CALL -> PARSING: HTTP response received (status 200)
API_CALL -> ERROR: HTTP status >= 400 or network error

PARSING -> NORMALIZING: Response parsed successfully
PARSING -> ERROR: Parse failure

NORMALIZING -> IDLE: Normalization complete, return result
NORMALIZING -> ERROR: Unexpected normalization failure

ERROR -> IDLE: Error returned to caller, client ready for next request
```

### 3.3 Graceful Degradation

When OpenFoodFacts API is unavailable after all retries:
- Log warning with context (search query, error details)
- Return empty `[]NormalizedFoodItem{}` with a warning indicator
- Do NOT propagate API errors to upstream callers
- Allow application to continue functioning with data from other sources

### 3.4 Data Quality Handling

OpenFoodFacts is community-contributed, so data quality varies:
- Skip products without a product name
- Skip products with incomplete nutrient data (all zeros or missing)
- Flag products with unusual nutrient values (> 200% DV per 100g) for review
- Prefer products with nutrition grades (a-e) over ungraded products
- Prefer products with images over those without

## 4. Component Interfaces

### 4.1 Public Functions

```go
// NewClient creates a new OpenFoodFacts API client
func NewClient(config OpenFoodFactsConfig) *Client {
    return &Client{
        config: OpenFoodFactsConfig{
            BaseURL:     config.BaseURL,
            Timeout:     config.Timeout,
            MaxRetries:  config.MaxRetries,
            UserAgent:   config.UserAgent,
            Country:     config.Country,
            Language:    config.Language,
        },
        httpClient: &http.Client{
            Timeout: config.Timeout,
        },
        rateLimiter: &RateLimitHandler{
            requestsPerSecond: 5.0, // Conservative rate limit for OpenFoodFacts
        },
    }
}

// SearchFoods searches the OpenFoodFacts database for food products
func (c *Client) SearchFoods(ctx context.Context, params OpenFoodFactsSearchParams) ([]NormalizedFoodItem, error)

// GetProductByBarcode retrieves a specific product by its barcode
func (c *Client) GetProductByBarcode(ctx context.Context, barcode string) (*NormalizedFoodItem, error)

// GetProductByCode is an alias for GetProductByBarcode for compatibility
func (c *Client) GetProductByCode(ctx context.Context, code string) (*NormalizedFoodItem, error)
```

### 4.2 Internal Helper Functions

```go
// waitForRateLimit ensures the rate limit is respected
func (c *Client) waitForRateLimit() error

// executeRequest performs HTTP request with retry logic
func (c *Client) executeRequest(ctx context.Context, endpoint string, headers map[string]string) (*http.Response, error)

// normalizeProduct converts OpenFoodFacts format to internal schema
func (c *Client) normalizeProduct(product OpenFoodFactsProduct) NormalizedFoodItem

// extractNutrientValue safely extracts nutrient value from optional pointer
func (c *Client) extractNutrientValue(value *float64, servingGrams float64) *float64

// parseIngredients splits ingredient string into slice
func (c *Client) parseIngredients(ingredientText string) []string

// estimateGramsFromText estimates serving size in grams from text description
func (c *Client) estimateGramsFromText(text string) float64

// multiplyOrNil multiplies a pointer value by a factor, returning nil if input is nil
func multiplyOrNil(value *float64, factor float64) *float64
```

### 4.3 Configuration Defaults

```go
const (
    DefaultBaseURL       = "https://world.openfoodfacts.org"
    DefaultTimeout       = 30 * time.Second
    DefaultMaxRetries    = 3
    DefaultPageSize      = 25
    MaxPageSize          = 100
    DefaultUserAgent     = "Mealswapp/1.0"
    DefaultCountry       = "world"
    DefaultLanguage      = "en"
    RateLimitRequestsPS  = 5.0
)
```

### 4.4 Environment Variables

```go
// Config is loaded from environment variables
type Config struct {
    OpenFoodFactsBaseURL string  // OPENFOODFACTS_BASE_URL (default: https://world.openfoodfacts.org)
    OpenFoodFactsTimeout int     // OPENFOODFACTS_TIMEOUT_SECONDS (default: 30)
    OpenFoodFactsMaxRetries int  // OPENFOODFACTS_MAX_RETRIES (default: 3)
}
```
