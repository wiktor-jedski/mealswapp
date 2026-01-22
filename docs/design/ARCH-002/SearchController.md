## FILE: SearchController.md
**Traceability:** ARCH-002

### 1. Data Structures & Types

```go
// SearchMode determines the search strategy
type SearchMode string

const (
    SearchModeText       SearchMode = "text"
    SearchModeSimilarity SearchMode = "similarity"
    SearchModeAuto       SearchMode = "auto"
)

// TagFilter defines whitelist/blacklist filter criteria
type TagFilter struct {
    TagID   string `json:"tagId"`
    Include bool   `json:"include"` // true = whitelist, false = blacklist
}

// SearchRequest represents the incoming search query
type SearchRequest struct {
    Query       string      `json:"query"`
    Mode        SearchMode  `json:"mode"`
    Filters     []TagFilter `json:"filters"`
    Page        int         `json:"page"`
    Ingredients []string    `json:"ingredients,omitempty"` // For similarity search context
    SourceItemID string     `json:"sourceItemId,omitempty"` // For replacement search (tag weighting)
}

// SearchResponse contains paginated search results
type SearchResponse struct {
    Items            []FoodItemResult `json:"items"`
    TotalCount       int              `json:"totalCount"`
    Page             int              `json:"page"`
    SimilarityScores []float64        `json:"similarityScores"`
}

// FoodItemResult represents a single search result item
type FoodItemResult struct {
    ID               string   `json:"id"`
    Name             string   `json:"name"`
    ImageURL         string   `json:"imageUrl,omitempty"`
    Macros           Macros   `json:"macros"`
    CategoryTags     []string `json:"categoryTags"`
    FunctionalityTags []string `json:"functionalityTags"`
}

// Macros represents macronutrient values per 100g
type Macros struct {
    Protein float64 `json:"protein"`
    Carbs   float64 `json:"carbs"`
    Fat     float64 `json:"fat"`
}

// AutocompleteResult represents a ranked suggestion
type AutocompleteResult struct {
    ID         string `json:"id"`
    Name       string `json:"name"`
    MatchType  string `json:"matchType"` // "exact", "fuzzy", "partial"
    Score      int    `json:"score"`     // Ranking priority (lower = better)
}

// Internal: cached query key structure
type cacheKey struct {
    QueryHash string
    Page      int
    FilterHash string
}
```

### 2. Logic & Algorithms (Step-by-Step)

#### 2.1 Main Search Handler (`HandleSearch`)

```
FUNCTION HandleSearch(ctx *fiber.Ctx) error:
    1. Parse request body into SearchRequest struct
    2. Validate request:
       - If Page < 1, set Page = 1
       - If Mode is empty, set Mode = SearchModeAuto
    3. Generate cache key from (Query, Filters, Page, SourceItemID)
    4. Check Redis cache via ARCH-011:
       - IF cache hit: return cached SearchResponse
    5. Determine search strategy:
       - IF Mode == SearchModeSimilarity OR (Mode == SearchModeAuto AND Query == "" AND len(Ingredients) >= 2):
           - Execute similarity search (Section 2.3)
       - ELSE:
           - Execute text search (Section 2.2)
    6. Apply tag filters to results (Section 2.4)
    7. Apply pagination (Section 2.5)
    8. IF SourceItemID is provided:
       - Apply functionality tag weighting (Section 2.6)
       - Re-sort results by final weighted score
    9. Store result in Redis cache with TTL = 300 seconds
    10. Return SearchResponse as JSON
```

#### 2.2 Text Search (`executeTextSearch`)

```
FUNCTION executeTextSearch(query string, filters []TagFilter) ([]FoodItemResult, error):
    1. Sanitize query: trim whitespace, convert to lowercase
    2. Build SQL query via ARCH-005:
       SELECT id, name, image_url, protein, carbs, fat, category_tags, functionality_tags
       FROM food_items
       WHERE LOWER(name) LIKE '%' || $1 || '%'
       ORDER BY
         CASE WHEN LOWER(name) = $1 THEN 0
              WHEN LOWER(name) LIKE $1 || '%' THEN 1
              ELSE 2
         END,
         LENGTH(name) ASC
       LIMIT 100
    3. Execute query via repository
    4. Map database rows to []FoodItemResult
    5. Return results
```

#### 2.3 Similarity Search (`executeSimilaritySearch`)

```
FUNCTION executeSimilaritySearch(ingredients []string, sourceItemID string) ([]FoodItemResult, []float64, error):
    1. IF sourceItemID is provided:
       - Fetch source item macros from ARCH-005
       - Set sourceMacros = source item's MacroVector
    2. ELSE:
       - Aggregate macros from ingredients list via ARCH-005
       - Set sourceMacros = aggregated MacroVector
    3. Fetch all candidate items from ARCH-005 (excluding source item)
    4. Call ARCH-003 SimilarityEngine.Calculate:
       - Input: sourceMacros, candidateItems
       - Output: []SimilarityResult (itemId, score, tier)
    5. Filter results where score >= 0.40 (threshold from ARCH-003)
    6. Sort by score descending
    7. Map to []FoodItemResult with parallel []float64 scores
    8. Return (results, scores)
```

#### 2.4 Tag Filtering (`applyTagFilters`)

```
FUNCTION applyTagFilters(items []FoodItemResult, filters []TagFilter) []FoodItemResult:
    1. IF len(filters) == 0: return items unchanged
    2. Separate filters into whitelist and blacklist
    3. FOR each item in items:
       a. IF blacklist is not empty:
          - IF item has ANY tag in blacklist: exclude item, continue
       b. IF whitelist is not empty:
          - IF item has NONE of the tags in whitelist: exclude item, continue
       c. Add item to filtered results
    4. Return filtered results
```

#### 2.5 Pagination (`applyPagination`)

```
FUNCTION applyPagination(items []FoodItemResult, scores []float64, page int) ([]FoodItemResult, []float64, int):
    CONST PAGE_SIZE = 10
    1. totalCount = len(items)
    2. startIndex = (page - 1) * PAGE_SIZE
    3. IF startIndex >= totalCount: return empty slice, empty slice, totalCount
    4. endIndex = min(startIndex + PAGE_SIZE, totalCount)
    5. pagedItems = items[startIndex:endIndex]
    6. pagedScores = scores[startIndex:endIndex] (if scores not empty)
    7. Return (pagedItems, pagedScores, totalCount)
```

#### 2.6 Functionality Tag Weighting (`applyFunctionalityTagWeighting`)

```
FUNCTION applyFunctionalityTagWeighting(items []FoodItemResult, scores []float64, sourceItemID string) []float64:
    CONST TAG_WEIGHT_MULTIPLIER = 0.2
    1. Fetch source item functionality tags from ARCH-005
    2. FOR i, item in items:
       a. tagMatchCount = count of item.FunctionalityTags that exist in source tags
       b. IF len(scores) > i AND scores[i] > 0:
          - finalScore = scores[i] * (1.0 + TAG_WEIGHT_MULTIPLIER * float64(tagMatchCount))
          - scores[i] = finalScore
    3. Return modified scores
```

#### 2.7 Autocomplete Handler (`HandleAutocomplete`)

```
FUNCTION HandleAutocomplete(ctx *fiber.Ctx) error:
    1. Parse query parameter from URL
    2. IF len(query) < 2: return empty array
    3. Generate cache key: "autocomplete:" + lowercase(query)
    4. Check Redis cache:
       - IF cache hit: return cached suggestions
    5. Fetch candidate names from ARCH-005:
       SELECT DISTINCT name FROM food_items
       WHERE LOWER(name) LIKE '%' || $1 || '%'
       LIMIT 50
    6. Rank candidates using three-tier algorithm (Section 2.8)
    7. Take top 10 results
    8. Store in Redis cache with TTL = 60 seconds
    9. Return as JSON array of AutocompleteResult
```

#### 2.8 Autocomplete Ranking (`rankAutocompleteResults`)

```
FUNCTION rankAutocompleteResults(query string, candidates []string) []AutocompleteResult:
    1. normalizedQuery = lowercase(trim(query))
    2. results = empty []AutocompleteResult
    3. FOR each candidate in candidates:
       a. normalizedName = lowercase(candidate)
       b. Calculate priority score:
          - TIER 1 (Exact Match):
            IF normalizedName == normalizedQuery OR normalizedName starts with normalizedQuery:
              score = 0
              matchType = "exact"
          - TIER 2 (Levenshtein Distance):
            ELSE:
              levenshteinDist = calculateLevenshtein(normalizedQuery, normalizedName)
              IF levenshteinDist <= 3:
                score = 100 + levenshteinDist
                matchType = "fuzzy"
              ELSE:
                score = 200 + len(normalizedName)
                matchType = "partial"
       c. Append AutocompleteResult{Name: candidate, Score: score, MatchType: matchType}
    4. Sort results by Score ascending
    5. Return results
```

#### 2.9 Levenshtein Distance (`calculateLevenshtein`)

```
FUNCTION calculateLevenshtein(s1 string, s2 string) int:
    1. m = len(s1), n = len(s2)
    2. Create matrix dp[m+1][n+1]
    3. Initialize dp[i][0] = i for i in 0..m
    4. Initialize dp[0][j] = j for j in 0..n
    5. FOR i = 1 to m:
       FOR j = 1 to n:
         IF s1[i-1] == s2[j-1]:
           cost = 0
         ELSE:
           cost = 1
         dp[i][j] = min(
           dp[i-1][j] + 1,      // deletion
           dp[i][j-1] + 1,      // insertion
           dp[i-1][j-1] + cost  // substitution
         )
    6. Return dp[m][n]
```

### 3. State Management & Error Handling

#### 3.1 Error States

| Error State | Trigger | HTTP Status | Response | Recovery Action |
|:------------|:--------|:------------|:---------|:----------------|
| Invalid Request Body | Malformed JSON | 400 | `{"error": "Invalid request format"}` | Client fixes request |
| Empty Query (Text Mode) | Query empty when Mode=text | 400 | `{"error": "Query required for text search"}` | Client provides query |
| Source Item Not Found | SourceItemID does not exist | 404 | `{"error": "Source item not found"}` | Client verifies item ID |
| Database Unavailable | ARCH-005 connection failure | 503 | `{"error": "Service temporarily unavailable"}` | Automatic retry by client |
| Similarity Engine Timeout | ARCH-003 response > 5s | 200 (degraded) | Return text search results only with `"degraded": true` | Log warning, serve partial |
| Redis Cache Unavailable | ARCH-011 connection failure | N/A (internal) | Bypass cache, query database directly | Circuit breaker: retry after 30s |
| No Results Found | Valid query, zero matches | 200 | `{"items": [], "totalCount": 0}` | Client displays "No results" |
| Page Out of Range | Page > total pages | 200 | `{"items": [], "totalCount": N}` | Client adjusts page |

#### 3.2 State Transitions

```
                    ┌─────────────┐
                    │   IDLE      │
                    └──────┬──────┘
                           │ Request received
                           ▼
                    ┌─────────────┐
                    │  VALIDATING │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │ Valid      │            │ Invalid
              ▼            │            ▼
       ┌─────────────┐     │     ┌─────────────┐
       │CACHE_CHECK  │     │     │   ERROR     │───► Return 4xx
       └──────┬──────┘     │     └─────────────┘
              │            │
       ┌──────┴──────┐     │
       │ HIT    MISS │     │
       ▼             ▼     │
   ┌───────┐  ┌─────────────┐
   │RETURN │  │  SEARCHING  │
   │CACHED │  └──────┬──────┘
   └───────┘         │
                     │
         ┌───────────┼───────────┐
         │ Text      │           │ Similarity
         ▼           │           ▼
  ┌─────────────┐    │    ┌─────────────┐
  │ TEXT_SEARCH │    │    │ SIM_SEARCH  │
  └──────┬──────┘    │    └──────┬──────┘
         │           │           │
         └───────────┼───────────┘
                     │
                     ▼
              ┌─────────────┐
              │  FILTERING  │
              └──────┬──────┘
                     │
                     ▼
              ┌─────────────┐
              │ PAGINATING  │
              └──────┬──────┘
                     │
                     ▼
              ┌─────────────┐
              │CACHE_STORE  │
              └──────┬──────┘
                     │
                     ▼
              ┌─────────────┐
              │  RESPONSE   │───► Return 200
              └─────────────┘
```

#### 3.3 Timeout Configuration

| Operation | Timeout | Action on Timeout |
|:----------|:--------|:------------------|
| Redis cache read | 50ms | Skip cache, proceed to database |
| Database query | 2000ms | Return 503 |
| Similarity engine call | 5000ms | Return degraded response (text results only) |
| Redis cache write | 100ms | Log warning, continue (non-blocking) |
| Autocomplete total | 100ms | Return partial results available |

### 4. Component Interfaces

#### 4.1 HTTP Handlers (Fiber)

```go
// RegisterRoutes registers all search-related routes
func RegisterRoutes(app *fiber.App, ctrl *SearchController) {
    search := app.Group("/api/v1/search")
    search.Post("/", ctrl.HandleSearch)
    search.Get("/autocomplete", ctrl.HandleAutocomplete)
}

// HandleSearch processes search requests
// POST /api/v1/search
// Request: SearchRequest
// Response: SearchResponse
func (c *SearchController) HandleSearch(ctx *fiber.Ctx) error

// HandleAutocomplete returns ranked suggestions
// GET /api/v1/search/autocomplete?q={query}
// Response: []AutocompleteResult
func (c *SearchController) HandleAutocomplete(ctx *fiber.Ctx) error
```

#### 4.2 Internal Functions

```go
// SearchController holds dependencies for search operations
type SearchController struct {
    repo        repository.FoodItemRepository  // ARCH-005
    similarity  similarity.Engine              // ARCH-003
    cache       cache.RedisClient              // ARCH-011
    logger      *slog.Logger
}

// NewSearchController creates a new controller with injected dependencies
func NewSearchController(
    repo repository.FoodItemRepository,
    similarity similarity.Engine,
    cache cache.RedisClient,
    logger *slog.Logger,
) *SearchController

// executeTextSearch performs text-based search against the database
func (c *SearchController) executeTextSearch(
    ctx context.Context,
    query string,
    filters []TagFilter,
) ([]FoodItemResult, error)

// executeSimilaritySearch performs similarity-based search via ARCH-003
func (c *SearchController) executeSimilaritySearch(
    ctx context.Context,
    ingredients []string,
    sourceItemID string,
) ([]FoodItemResult, []float64, error)

// applyTagFilters filters results by whitelist/blacklist tags
func applyTagFilters(
    items []FoodItemResult,
    filters []TagFilter,
) []FoodItemResult

// applyPagination extracts a page of results
func applyPagination(
    items []FoodItemResult,
    scores []float64,
    page int,
) (pagedItems []FoodItemResult, pagedScores []float64, totalCount int)

// applyFunctionalityTagWeighting adjusts scores based on tag matches
func (c *SearchController) applyFunctionalityTagWeighting(
    ctx context.Context,
    items []FoodItemResult,
    scores []float64,
    sourceItemID string,
) ([]float64, error)

// rankAutocompleteResults applies three-tier ranking algorithm
func rankAutocompleteResults(
    query string,
    candidates []string,
) []AutocompleteResult

// calculateLevenshtein computes edit distance between two strings
func calculateLevenshtein(s1, s2 string) int

// generateCacheKey creates a deterministic cache key from search parameters
func generateCacheKey(req SearchRequest) string
```

#### 4.3 Dependency Interfaces (Required from other ARCH components)

```go
// From ARCH-005 (Data Repository)
type FoodItemRepository interface {
    SearchByName(ctx context.Context, query string, limit int) ([]FoodItem, error)
    GetByID(ctx context.Context, id string) (*FoodItem, error)
    GetByIDs(ctx context.Context, ids []string) ([]FoodItem, error)
    GetAllForSimilarity(ctx context.Context, excludeID string) ([]FoodItem, error)
    GetAutocompleteNames(ctx context.Context, prefix string, limit int) ([]string, error)
}

// From ARCH-003 (Similarity Engine)
type SimilarityEngine interface {
    Calculate(ctx context.Context, req ComparisonRequest) ([]SimilarityResult, error)
}

// From ARCH-011 (Caching Layer)
type CacheClient interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}
```
