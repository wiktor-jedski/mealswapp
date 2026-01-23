# FILE: USDAClient.md
**Traceability:** ARCH-012

## 1. Data Structures & Types

```go
package usda

import "time"

// USDAAPIConfig holds configuration for the USDA FoodData Central API
type USDAAPIConfig struct {
    APIKey     string
    BaseURL    string
    Timeout    time.Duration
    MaxRetries int
}

// USDAFoodItem represents raw response from USDA FoodData Central API
type USDAFoodItem struct {
    FDCID         int                    `json:"fdcId"`
    Description   string                 `json:"description"`
    DataType      string                 `json:"dataType"`
    PublicationDate string               `json:"publicationDate"`
    FoodNutrients []USDAFoodNutrient     `json:"foodNutrients"`
    ServingSize   *float64               `json:"servingSize"`
    ServingSizeUnit string               `json:"servingSizeUnit"`
    HouseholdServingFullText string      `json:"householdServingFullText"`
    IngredientList string                `json:"ingredientList"`
    LastUpdated   time.Time              `json:"lastUpdated"`
}

// USDAFoodNutrient represents nutrient data from USDA API
type USDAFoodNutrient struct {
    NutrientID    int     `json:"nutrientId"`
    NutrientName  string  `json:"nutrientName"`
    NutrientNumber string `json:"nutrientNumber"`
    UnitName      string  `json:"unitName"`
    Value         float64 `json:"value"`
    Rank          int     `json:"rank"`
    DisplayName   string  `json:"displayName"`
}

// USDASearchResponse represents the search API response
type USDASearchResponse struct {
    TotalHits          int           `json:"totalHits"`
    CurrentPage        int           `json:"currentPage"`
    TotalPages         int           `json:"totalPages"`
    Foods              []USDAFoodItem `json:"foods"`
    Aggregations       interface{}   `json:"aggregations"`
}

// USDASearchParams defines search parameters for USDA API
type USDASearchParams struct {
    Query           string
    Page            int
    PageSize        int
    DataType        []string // e.g., "Branded", "Foundation", "Survey"
    SortField       string   // "fdcId", "description", "publishedDate"
    SortOrder       string   // "asc", "desc"
    Format          string   // "full", "basic"
}

// NormalizedFoodItem represents normalized food data for internal use
type NormalizedFoodItem struct {
    ExternalID        string
    Source            string // "USDA"
    Name              string
    DefaultServingGrams float64
    Calories          *float64
    Protein           *float64
    Carbs             *float64
    Fat               *float64
    Fiber             *float64
    Sugar             *float64
    Sodium            *float64
    Cholesterol       *float64
    SaturatedFat      *float64
    TransFat          *float64
    Potassium         *float64
    VitaminA          *float64
    VitaminC          *float64
    Calcium           *float64
    Iron              *float64
    Ingredients       []string
    ServingSizeText   string
    LastUpdated       time.Time
}

// Client wraps USDA API interactions
type Client struct {
    config USDAAPIConfig
    httpClient *http.Client
    rateLimiter *RateLimitHandler
}

// RateLimitHandler manages API rate limits
type RateLimitHandler struct {
    requestsPerSecond float64
    lastRequestTime   time.Time
    mu                sync.Mutex
}

// USDAError represents errors from USDA API
type USDAError struct {
    Code    int
    Message string
    Details interface{}
}

func (e *USDAError) Error() string {
    return fmt.Sprintf("USDA API error (%d): %s", e.Code, e.Message)
}
```

## 2. Logic & Algorithms

### 2.1 Search Foods

```
FUNCTION SearchFoods(params USDASearchParams) -> ([]NormalizedFoodItem, error)

1. Validate input parameters
   - IF params.Query is empty THEN RETURN error "search query required"
   - IF params.Page < 1 THEN params.Page = 1
   - IF params.PageSize not in [1, 50, 100] THEN params.PageSize = 25
   - IF params.DataType is empty THEN params.DataType = ["Branded", "Foundation"]

2. Construct API endpoint
   - endpoint = config.BaseURL + "/foods/search"
   - queryParams = {
       "query": params.Query,
       "page": params.Page,
       "pageSize": params.PageSize,
       "dataType": join(params.DataType, ","),
       "sort": params.SortField,
       "ascending": params.SortOrder == "asc",
       "format": params.Format
     }

3. Wait for rate limit availability
   - CALL rateLimiter.Wait()

4. Execute HTTP GET request
   - timeout = config.Timeout
   - IF request fails THEN
     - retry up to config.MaxRetries times with exponential backoff
     - IF all retries fail THEN RETURN error

5. Parse response body into USDASearchResponse
   - IF parse fails THEN RETURN error "invalid response format"

6. FOR EACH food in response.Foods:
   - normalized = CALL NormalizeFoodItem(food)
   - append normalized to results

7. RETURN results, nil
```

### 2.2 Get Food By ID

```
FUNCTION GetFoodByID(fdcID int) -> (*NormalizedFoodItem, error)

1. Construct API endpoint
   - endpoint = config.BaseURL + "/food/" + string(fdcID)
   - queryParams = {"format": "full"}

2. Wait for rate limit availability
   - CALL rateLimiter.Wait()

3. Execute HTTP GET request
   - timeout = config.Timeout
   - IF request fails THEN
     - retry up to config.MaxRetries times
     - IF all retries fail THEN RETURN error

4. Parse response body into USDAFoodItem
   - IF parse fails THEN RETURN error "invalid food item format"

5. Normalize the food item
   - normalized = CALL NormalizeFoodItem(foodItem)

6. RETURN normalized, nil
```

### 2.3 Normalize Food Item

```
FUNCTION NormalizeFoodItem(usdaItem USDAFoodItem) -> NormalizedFoodItem

1. Initialize normalized item
   - normalized = NormalizedFoodItem{
       ExternalID:  strconv.Itoa(usdaItem.FDCID),
       Source:      "USDA",
       Name:        usdaItem.Description,
       LastUpdated: usdaItem.LastUpdated,
       ServingSizeText: usdaItem.HouseholdServingFullText
     }

2. Calculate default serving in grams
   - IF usdaItem.ServingSize != nil THEN
     - normalized.DefaultServingGrams = *usdaItem.ServingSize
   - ELSE IF usdaItem.HouseholdServingFullText contains measurement THEN
     - normalized.DefaultServingGrams = CALL EstimateGramsFromText(usdaItem.HouseholdServingFullText)
   - ELSE
     - normalized.DefaultServingGrams = 100.0 // fallback to 100g

3. Extract nutrients and convert to per 100g
   - FOR EACH nutrient in usdaItem.FoodNutrients:
     - convertedValue = nutrient.Value * (100.0 / normalized.DefaultServingGrams)
     - SWITCH nutrient.NutrientID or nutrient.NutrientName:
       - CASE "208", "Calories": normalized.Calories = &convertedValue
       - CASE "203", "Protein": normalized.Protein = &convertedValue
       - CASE "205", "Carbohydrate": normalized.Carbs = &convertedValue
       - CASE "204", "Total lipid (fat)": normalized.Fat = &convertedValue
       - CASE "205", "Fiber": normalized.Fiber = &convertedValue
       - CASE "269", "Sugars": normalized.Sugar = &convertedValue
       - CASE "307", "Sodium": normalized.Sodium = &convertedValue
       - CASE "601", "Cholesterol": normalized.Cholesterol = &convertedValue
       - CASE "606", "Fatty acids, total saturated": normalized.SaturatedFat = &convertedValue
       - CASE "605", "Fatty acids, total trans": normalized.TransFat = &convertedValue
       - CASE "306", "Potassium": normalized.Potassium = &convertedValue
       - CASE "320", "Vitamin A": normalized.VitaminA = &convertedValue
       - CASE "401", "Vitamin C": normalized.VitaminC = &convertedValue
       - CASE "301", "Calcium": normalized.Calcium = &convertedValue
       - CASE "303", "Iron": normalized.Iron = &convertedValue

4. Parse ingredients
   - IF usdaItem.IngredientList is not empty THEN
     - normalized.Ingredients = CALL ParseIngredients(usdaItem.IngredientList)

5. RETURN normalized
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
| `ErrEmptyQuery` | Search query is empty | Validate input before API call, return user-friendly error |
| `ErrRateLimitExceeded` | Too many requests in short window | Implement exponential backoff, return retry-after header |
| `ErrAPITimeout` | USDA API not responding within timeout | Retry with backoff, then return timeout error |
| `ErrInvalidAPIKey` | API key is missing or invalid | Log error, return authentication error to caller |
| `ErrInvalidResponse` | API returned unexpected format | Log response body, return parsing error |
| `ErrServiceUnavailable` | USDA API returns 503 | Retry up to MaxRetries, then graceful degradation |
| `ErrNetworkError` | Network connectivity issue | Retry with backoff, return network error after retries |
| `ErrMaxRetriesExceeded` | All retry attempts failed | Log final error, return wrapped error with attempt count |

### 3.2 State Transitions

```
Initial State: IDLE

IDLE -> VALIDATING: SearchFoods() or GetFoodByID() called
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

When USDA API is unavailable after all retries:
- Log warning with context (search query, error details)
- Return empty `[]NormalizedFoodItem{}` with a warning indicator
- Do NOT propagate API errors to upstream callers
- Allow application to continue functioning with data from other sources

## 4. Component Interfaces

### 4.1 Public Functions

```go
// NewClient creates a new USDA API client
func NewClient(config USDAAPIConfig) *Client {
    return &Client{
        config: USDAAPIConfig{
            APIKey:          config.APIKey,
            BaseURL:         config.BaseURL,
            Timeout:         config.Timeout,
            MaxRetries:      config.MaxRetries,
        },
        httpClient: &http.Client{
            Timeout: config.Timeout,
        },
        rateLimiter: &RateLimitHandler{
            requestsPerSecond: 10.0, // USDA limit is 10 req/sec for free tier
        },
    }
}

// SearchFoods searches the USDA database for food items
func (c *Client) SearchFoods(ctx context.Context, params USDASearchParams) ([]NormalizedFoodItem, error)

// GetFoodByID retrieves a specific food item by its FDC ID
func (c *Client) GetFoodByID(ctx context.Context, fdcID int) (*NormalizedFoodItem, error)
```

### 4.2 Internal Helper Functions

```go
// waitForRateLimit ensures the rate limit is respected
func (c *Client) waitForRateLimit() error

// executeRequest performs HTTP request with retry logic
func (c *Client) executeRequest(ctx context.Context, endpoint string, queryParams map[string]string) (*http.Response, error)

// normalizeFoodItem converts USDA format to internal schema
func (c *Client) normalizeFoodItem(item USDAFoodItem) NormalizedFoodItem

// extractNutrientValue finds and converts a specific nutrient
func (c *Client) extractNutrientValue(nutrients []USDAFoodNutrient, nutrientIDs []string, servingGrams float64) *float64

// parseIngredients splits ingredient string into slice
func (c *Client) parseIngredients(ingredientList string) []string

// estimateGramsFromText estimates serving size in grams from text description
func (c *Client) estimateGramsFromText(text string) float64
```

### 4.3 Configuration Defaults

```go
const (
    DefaultBaseURL    = "https://api.nal.usda.gov/fdc/v1"
    DefaultTimeout    = 30 * time.Second
    DefaultMaxRetries = 3
    DefaultPageSize   = 25
    MaxPageSize       = 100
)
```
