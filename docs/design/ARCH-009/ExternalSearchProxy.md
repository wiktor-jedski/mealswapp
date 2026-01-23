# FILE: ExternalSearchProxy.md
**Traceability:** ARCH-009

## 1. Data Structures & Types

```go
package externalsearchproxy

import "github.com/gofiber/fiber/v2"

type SearchRequest struct {
	Query      string   `json:"query" validate:"required,min=1,max=200"`
	Sources    []string `json:"sources"` // Optional: ["usda", "openfoodfacts"], empty means all
	Page       int      `json:"page"`    // 1-indexed, default 1
	PageSize   int      `json:"pageSize"` // Default 20, max 100
	Category   string   `json:"category,omitempty"` // Optional filter
}

type SearchResultItem struct {
	ExternalID     string            `json:"externalId"`
	Source         string            `json:"source"` // "usda" or "openfoodfacts"
	Name           string            `json:"name"`
	Brand          string            `json:"brand,omitempty"`
	Description    string            `json:"description,omitempty"`
	ImageURL       string            `json:"imageUrl,omitempty"`
	Category       string            `json:"category,omitempty"`
	Nutrients      []NutrientEntry   `json:"nutrients"`
	ServingSize    float64           `json:"servingSize"` // grams
	ServingUnit    string            `json:"servingUnit"` // "g" or "ml"
	Calories       float64           `json:"calories"`
	Protein        float64           `json:"protein"`
	Carbohydrates  float64           `json:"carbohydrates"`
	Fat            float64           `json:"fat"`
	Fiber          float64           `json:"fiber,omitempty"`
	Sugar          float64           `json:"sugar,omitempty"`
	Sodium         float64           `json:"sodium,omitempty"`
	Ingredients    string            `json:"ingredients,omitempty"`
	RawData        map[string]any    `json:"rawData,omitempty"` // Preserved for debugging
}

type NutrientEntry struct {
	Name        string  `json:"name"`
	Amount      float64 `json:"amount"`
	Unit        string  `json:"unit"`
	Per100g     float64 `json:"per100g"` // Normalized value per 100g
}

type SearchResponse struct {
	Results     []SearchResultItem `json:"results"`
	TotalCount  int                `json:"totalCount"`
	Page        int                `json:"page"`
	PageSize    int                `json:"pageSize"`
	TotalPages  int                `json:"totalPages"`
	SourcesUsed []string           `json:"sourcesUsed"`
	CachedAt    *time.Time         `json:"cachedAt,omitempty"`
	Warnings    []string           `json:"warnings,omitempty"`
}

type ImportCandidate struct {
	SearchResultItem
	Selected    bool     `json:"selected"`
	EditedName  string   `json:"editedName,omitempty"`
	EditedTags  []string `json:"editedTags,omitempty"`
	Notes       string   `json:"notes,omitempty"`
}

type ProxyConfig struct {
	CacheTTL          time.Duration `mapstructure:"cache_ttl"`
	RateLimitRequests int           `mapstructure:"rate_limit_requests"`
	RateLimitWindow   time.Duration `mapstructure:"rate_limit_window"`
	Timeout           time.Duration `mapstructure:"timeout"`
	MaxRetries        int           `mapstructure:"max_retries"`
}

type AdminContext struct {
	UserID    string `json:"userId"`
	Username  string `json:"username"`
	IsSuperAdmin bool `json:"isSuperAdmin"`
	RequestID string `json:"requestId"`
}
```

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 External Search Flow

```
ALGORITHM: HandleExternalSearch
INPUT: ctx (Fiber context), adminCtx (AdminContext), searchReq (SearchRequest)
OUTPUT: SearchResponse or error

1. VALIDATE INPUT
   1.1 IF searchReq.Query is empty or length < 1 THEN
       RETURN ErrInvalidQuery ("Search query is required")
   1.2 IF searchReq.Query length > 200 THEN
       RETURN ErrQueryTooLong ("Query must be 200 characters or less")
   1.3 IF searchReq.Page < 1 THEN set searchReq.Page = 1
   1.4 IF searchReq.PageSize < 1 OR > 100 THEN set searchReq.PageSize = 20
   1.5 IF searchReq.Sources is empty THEN set searchReq.Sources = ["usda", "openfoodfacts"]
   1.6 FOR EACH source IN searchReq.Sources DO
       IF source NOT IN ["usda", "openfoodfacts"] THEN
           RETURN ErrInvalidSource ("Invalid source: " + source)
       END IF
   END FOR

2. CHECK RATE LIMIT
   2.1 rateKey = "ratelimit:extsearch:" + adminCtx.UserID
   2.2 currentCount = Redis.GET(rateKey)
   2.3 IF currentCount >= ProxyConfig.RateLimitRequests THEN
       RETURN ErrRateLimited ("Rate limit exceeded. Try again later.")
   2.4 Redis.INCR(rateKey)
   2.5 IF currentCount == 0 THEN Redis.EXPIRE(rateKey, ProxyConfig.RateLimitWindow)

3. CHECK CACHE (for identical queries)
   3.1 cacheKey = "extsearch:" + hash(searchReq.Query + sources + page)
   3.2 cachedResponse = Redis.GET(cacheKey)
   3.3 IF cachedResponse exists AND cachedResponse.age < ProxyConfig.CacheTTL THEN
       3.3.1 cachedResponse.CachedAt = current_time
       RETURN cachedResponse
   3.4 IF cachedResponse exists AND cachedResponse.age >= ProxyConfig.CacheTTL THEN
       3.4.1 Continue with fresh search (cache will be updated)

4. ROUTE TO ARCH-012 (External Data Integration)
   4.1 results = []SearchResultItem
   4.2 warnings = []string
   4.3 FOR EACH source IN searchReq.Sources DO
       4.3.1 sourceResults, err = ARCH012_Search(source, searchReq.Query, searchReq.Page, searchReq.PageSize)
       4.3.2 IF err IS ErrSourceUnavailable THEN
           4.3.2.1 warnings.append("Source temporarily unavailable: " + source)
           CONTINUE
       4.3.3 IF err IS ErrRateLimitedSource THEN
           4.3.3.1 warnings.append("Rate limited by source: " + source)
           CONTINUE
       4.3.4 IF err != nil THEN
           4.3.4.1 warnings.append("Error from " + source + ": " + err.message)
           CONTINUE
       4.3.5 results.append(sourceResults)
   END FOR

5. NORMALIZE AND MERGE RESULTS
   5.1 normalizedResults = []SearchResultItem
   5.2 FOR EACH item IN results DO
       5.2.1 normalizedItem = NormalizeToInternalSchema(item)
       5.2.2 normalizedItem.Per100g = ConvertToPer100g(normalizedItem)
       5.2.3 normalizedResults.append(normalizedItem)
   5.3 Sort normalizedResults by relevance score (descending)

6. CONSTRUCT RESPONSE
   6.1 response = SearchResponse{
       Results: normalizedResults,
       TotalCount: sum(all source totalCounts),
       Page: searchReq.Page,
       PageSize: searchReq.PageSize,
       TotalPages: calculate based on totalCount and PageSize,
       SourcesUsed: sources with successful results,
       Warnings: warnings,
   }

7. CACHE RESPONSE
   7.1 IF warnings is empty OR warnings.length < searchReq.Sources.length THEN
       7.1.1 Redis.SET(cacheKey, response, ProxyConfig.CacheTTL)

8. LOG AUDIT
   8.1 AuditLog.create({
       action: "EXTERNAL_SEARCH",
       adminId: adminCtx.UserID,
       query: searchReq.Query,
       resultCount: len(normalizedResults),
       sourcesUsed: response.SourcesUsed,
       requestId: adminCtx.RequestID,
   })

9. RETURN response
```

### 2.2 Import Validation Flow

```
ALGORITHM: ValidateImportCandidate
INPUT: candidate (ImportCandidate)
OUTPUT: ValidationResult

1. CHECK REQUIRED FIELDS
   1.1 IF candidate.ExternalID is empty THEN
       RETURN ErrMissingExternalID
   1.2 IF candidate.Selected AND candidate.Name is empty THEN
       RETURN ErrMissingName ("Selected item must have a name")
   1.3 IF candidate.Selected AND candidate.Calories < 0 THEN
       RETURN ErrInvalidCalories ("Calories cannot be negative")

2. VALIDATE NUTRIENTS
   2.1 FOR EACH nutrient IN candidate.Nutrients DO
       2.1.1 IF nutrient.Amount < 0 THEN
           RETURN ErrInvalidNutrient ("Nutrient amount cannot be negative")
       2.1.2 IF nutrient.Unit not IN allowedUnits THEN
           RETURN ErrInvalidUnit ("Invalid nutrient unit: " + nutrient.Unit)
   2.2 IF candidate.Protein < 0 OR candidate.Protein > 1000 THEN
       RETURN ErrInvalidProtein ("Invalid protein value")
   2.3 IF candidate.Carbohydrates < 0 OR candidate.Carbohydrates > 1000 THEN
       RETURN ErrInvalidCarbs ("Invalid carbohydrates value")
   2.4 IF candidate.Fat < 0 OR candidate.Fat > 1000 THEN
       RETURN ErrInvalidFat ("Invalid fat value")

3. VALIDATE TAGS (if provided)
   3.1 FOR EACH tag IN candidate.EditedTags DO
       3.1.1 IF tag is empty OR len(tag) > 50 THEN
           RETURN ErrInvalidTag ("Tag must be 1-50 characters")
       3.1.2 IF tag contains special characters THEN
           RETURN ErrInvalidTag ("Tag contains invalid characters")
   3.2 Validate tag count (max 20 tags per item)

4. RETURN success
   RETURN ValidationResult{IsValid: true, Warnings: []}
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error Code | Condition | HTTP Status | Recovery Action |
| :--- | :--- | :--- | :--- |
| `ErrInvalidQuery` | Empty or whitespace query | 400 | User provides valid search term |
| `ErrQueryTooLong` | Query > 200 characters | 400 | User shortens query |
| `ErrInvalidSource` | Unknown source in sources list | 400 | User selects valid sources |
| `ErrRateLimited` | Admin exceeded rate limit | 429 | Wait for rate limit reset (per-window) |
| `ErrSourceUnavailable` | External API returned error | 502 | Retry later or use different source |
| `ErrRateLimitedSource` | External API rate limited | 502 | Retry with backoff |
| `ErrTimeout` | External API timeout | 504 | Retry with longer timeout |
| `ErrCacheUnavailable` | Redis connection failed | 503 | Retry request (bypass cache) |
| `ErrUnauthorized` | Missing/invalid admin token | 401 | Re-authenticate |
| `ErrForbidden` | User lacks Admin role | 403 | Request admin access |
| `ErrMissingExternalID` | Import without external ID | 400 | Item was not properly selected |
| `ErrMissingName` | Selected import without name | 400 | Provide item name |
| `ErrInvalidNutrient` | Negative nutrient value | 400 | Correct nutrient values |
| `ErrInvalidTag` | Invalid tag format | 400 | Fix tag format |

### 3.2 State Transitions

```
STATE: Idle
  --> VALIDATE_INPUT [success] --> Validating
  --> VALIDATE_INPUT [fail] --> Error (return 400)

STATE: Validating
  --> CHECK_RATE_LIMIT [pass] --> RateLimitChecked
  --> CHECK_RATE_LIMIT [exceeded] --> RateLimited (return 429)

STATE: RateLimitChecked
  --> CHECK_CACHE [hit] --> Cached
  --> CHECK_CACHE [miss] --> FetchingExternal

STATE: Cached
  --> RETURN_RESPONSE --> Idle

STATE: FetchingExternal
  --> ARCH012_SEARCH [success] --> ProcessingResults
  --> ARCH012_SEARCH [partial] --> ProcessingResults (with warnings)
  --> ARCH012_SEARCH [all failed] --> Error (return 502)

STATE: ProcessingResults
  --> NORMALIZE_RESULTS [success] --> ConstructingResponse
  --> NORMALIZE_RESULTS [fail] --> Error (return 500)

STATE: ConstructingResponse
  --> CACHE_RESPONSE [success] --> Idle
  --> CACHE_RESPONSE [fail] --> Idle (continue without cache)
  --> RETURN_RESPONSE --> Idle

STATE: Error
  --> LOG_ERROR --> Idle (return error to caller)

STATE: RateLimited
  --> TIMEOUT (window expires) --> Idle
```

### 3.3 Circuit Breaker Pattern

For each external source, maintain a circuit breaker state:

```
CIRCUIT BREAKER STATES:
- CLOSED: Normal operation, requests allowed
- OPEN: Recent failures detected, requests fail-fast
- HALF_OPEN: Testing if service recovered

TRANSITIONS:
1. CLOSED --> OPEN: When failure count >= threshold (e.g., 5) within window
2. OPEN --> HALF_OPEN: After timeout period (e.g., 30 seconds)
3. HALF_OPEN --> CLOSED: When test request succeeds
4. HALF_OPEN --> OPEN: When test request fails

APPLY TO: USDA API, OpenFoodFacts API
PURPOSE: Prevent cascade failures, allow recovery
```

### 3.4 Retry Strategy

```
RETRY CONFIGURATION:
- MaxRetries: 3 (configurable)
- InitialBackoff: 1 second
- MaxBackoff: 30 seconds
- BackoffMultiplier: 2x

RETRY LOGIC:
1. First attempt: immediate
2. Second attempt: wait 1 second + jitter (±100ms)
3. Third attempt: wait 2 seconds + jitter
4. If still fails: return error

RETRY ON:
- Network timeouts
- 5xx server errors (except 501, 505)
- Rate limit responses (with header-guided backoff)

DO NOT RETRY ON:
- 4xx client errors
- Invalid query format
- Authentication failures
```

## 4. Component Interfaces

### 4.1 External Search Handler

```go
// HandleExternalSearch processes admin external search requests
// POST /api/admin/external-search
// Requires: Admin role
func (p *Proxy) HandleExternalSearch(c *fiber.Ctx) error {
    adminCtx := c.Locals("adminContext").(AdminContext)
    var req SearchRequest

    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
            Code:    "INVALID_REQUEST",
            Message: "Failed to parse request body",
        })
    }

    if err := validator.New().Struct(req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
            Code:    "VALIDATION_ERROR",
            Message: err.Error(),
        })
    }

    response, err := p.HandleSearch(adminCtx, req)
    if err != nil {
        return mapErrorToResponse(c, err)
    }

    return c.JSON(response)
}
```

### 4.2 Search Orchestration

```go
// HandleSearch orchestrates the external search flow
func (p *Proxy) HandleSearch(adminCtx AdminContext, req SearchRequest) (*SearchResponse, error) {
    // Step 1: Validate input
    if err := p.validateSearchRequest(req); err != nil {
        return nil, err
    }

    // Step 2: Check rate limit
    if !p.checkRateLimit(adminCtx.UserID) {
        return nil, ErrRateLimited{}
    }

    // Step 3: Check cache
    if cached, found := p.checkCache(req); found {
        return cached, nil
    }

    // Step 4: Fetch from external sources
    results, warnings := p.fetchFromSources(req)

    // Step 5: Normalize results
    normalized := p.normalizeResults(results)

    // Step 6: Construct response
    response := p.constructResponse(normalized, warnings, req)

    // Step 7: Cache response
    p.cacheResponse(req, response)

    // Step 8: Audit log
    p.logSearch(adminCtx, req, response)

    return response, nil
}
```

### 4.3 Source Router

```go
// fetchFromSources queries all specified external sources
func (p *Proxy) fetchFromSources(req SearchRequest) ([]SearchResultItem, []string) {
    var results []SearchResultItem
    var warnings []string
    var mu sync.Mutex
    var wg sync.WaitGroup

    sourceChan := make(chan SourceResult, len(req.Sources))

    for _, source := range req.Sources {
        wg.Add(1)
        go func(src string) {
            defer wg.Done()
            items, err := p.externalClient.Search(src, req.Query, req.Page, req.PageSize)
            mu.Lock()
            if err != nil {
                warnings = append(warnings, formatSourceError(src, err))
            } else {
                sourceChan <- SourceResult{Source: src, Items: items}
            }
            mu.Unlock()
        }(source)
    }

    wg.Wait()
    close(sourceChan)

    for result := range sourceChan {
        results = append(results, result.Items...)
    }

    return results, warnings
}
```

### 4.4 Data Normalizer

```go
// normalizeResults converts external API formats to internal schema
func (p *Proxy) normalizeResults(items []SearchResultItem) []SearchResultItem {
    normalized := make([]SearchResultItem, 0, len(items))

    for _, item := range items {
        normalizedItem := SearchResultItem{
            ExternalID:  p.generateExternalID(item.Source, item.ExternalID),
            Source:      item.Source,
            Name:        strings.TrimSpace(item.Name),
            Brand:       strings.TrimSpace(item.Brand),
            Description: strings.TrimSpace(item.Description),
            ImageURL:    p.resolveImageURL(item.Source, item.ImageURL),
            Category:    p.mapCategory(item.Source, item.Category),
            Nutrients:   p.normalizeNutrients(item.Nutrients),
            ServingSize: item.ServingSize,
            ServingUnit: normalizeUnit(item.ServingUnit),
            Calories:    p.extractNutrient(item.Nutrients, "Energy"),
            Protein:     p.extractNutrient(item.Nutrients, "Protein"),
            Carbohydrates: p.extractNutrient(item.Nutrients, "Carbohydrate"),
            Fat:         p.extractNutrient(item.Nutrients, "Total lipid"),
            Fiber:       p.extractNutrient(item.Nutrients, "Fiber"),
            Sugar:       p.extractNutrient(item.Nutrients, "Sugars"),
            Sodium:      p.extractNutrient(item.Nutrients, "Sodium"),
            Ingredients: strings.TrimSpace(item.Ingredients),
        }

        normalizedItem = p.calculatePer100g(normalizedItem)
        normalized = append(normalized, normalizedItem)
    }

    return p.deduplicateResults(normalized)
}

// calculatePer100g converts nutrient values to per-100g basis
func (p *Proxy) calculatePer100g(item SearchResultItem) SearchResultItem {
    if item.ServingSize <= 0 {
        return item
    }

    multiplier := 100.0 / item.ServingSize

    item.Calories *= multiplier
    item.Protein *= multiplier
    item.Carbohydrates *= multiplier
    item.Fat *= multiplier
    item.Fiber *= multiplier
    item.Sugar *= multiplier
    item.Sodium *= multiplier

    for i := range item.Nutrients {
        item.Nutrients[i].Per100g = item.Nutrients[i].Amount * multiplier
    }

    return item
}
```

### 4.5 Import Handler

```go
// HandleImportCandidates processes import candidates from admin UI
// POST /api/admin/import-candidates
// Requires: Admin role
func (p *Proxy) HandleImportCandidates(c *fiber.Ctx) error {
    adminCtx := c.Locals("adminContext").(AdminContext)
    var candidates []ImportCandidate

    if err := c.BodyParser(&candidates); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
            Code:    "INVALID_REQUEST",
            Message: "Failed to parse candidates",
        })
    }

    var importResults []ImportResult
    var validationErrors []ValidationError

    for i, candidate := range candidates {
        result, err := p.ValidateImportCandidate(candidate)
        if err != nil {
            validationErrors = append(validationErrors, ValidationError{
                Index: i,
                Error: err,
            })
            continue
        }

        if candidate.Selected {
            savedItem, err := p.DataRepository.CreateItem(adminCtx, candidate)
            if err != nil {
                return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
                    Code:    "IMPORT_FAILED",
                    Message: err.Error(),
                })
            }
            importResults = append(importResults, ImportResult{
                ExternalID: candidate.ExternalID,
                ItemID:     savedItem.ID,
                Success:    true,
            })
        }
    }

    // Log import audit
    p.logImport(adminCtx, importResults, validationErrors)

    return c.JSON(ImportResponse{
        Imported:  importResults,
        Errors:    validationErrors,
        Total:     len(candidates),
        Succeeded: len(importResults),
        Failed:    len(validationErrors),
    })
}
```

### 4.6 Configuration Loader

```go
// LoadConfig loads proxy configuration from environment/config
func LoadConfig(path string) (*ProxyConfig, error) {
    var config ProxyConfig

    viper.SetConfigFile(path)
    viper.SetDefault("cache_ttl", 15*time.Minute)
    viper.SetDefault("rate_limit_requests", 100)
    viper.SetDefault("rate_limit_window", 1*time.Minute)
    viper.SetDefault("timeout", 30*time.Second)
    viper.SetDefault("max_retries", 3)

    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }

    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }

    return &config, nil
}
```

### 4.7 Middleware Registration

```go
// RegisterRoutes registers ExternalSearchProxy routes on Fiber app
func (p *Proxy) RegisterRoutes(app *fiber.App, authMiddleware fiber.Handler) {
    admin := app.Group("/api/admin")
    admin.Use(authMiddleware)
    admin.Use(p.requireAdminRole)

    admin.Post("/external-search", p.HandleExternalSearch)
    admin.Post("/import-candidates", p.HandleImportCandidates)
    admin.Get("/search-sources", p.HandleListSources)
    admin.Get("/search-history", p.HandleSearchHistory)
}
```
