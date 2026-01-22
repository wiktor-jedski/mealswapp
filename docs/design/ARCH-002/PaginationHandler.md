## FILE: PaginationHandler.md
**Traceability:** ARCH-002

### 1. Data Structures & Types

```go
// PaginationRequest represents input parameters for paginated search results
type PaginationRequest struct {
    Query           string       // Search query string (may be empty for similarity search)
    Mode            SearchMode   // Search mode (text-based or similarity-based)
    Filters         []TagFilter  // Tag whitelist/blacklist filters
    Page            int          // Requested page number (1-indexed)
    Ingredients     []string     // Optional: ingredients for similarity search
    SourceItemID    string       // Optional: source item ID for replacement searches
    PageSize        int          // Results per page (default: 10, max: 10)
}

// SearchMode indicates the type of search to perform
type SearchMode int

const (
    SearchModeText       SearchMode = 1 // Text-based search using query string
    SearchModeSimilarity SearchMode = 2 // Cosine similarity search using ingredients
)

// TagFilter represents a filter on functionality tags
type TagFilter struct {
    TagID     string    // Tag identifier
    FilterType FilterType // Include or exclude
}

// FilterType indicates whether to include or exclude matching items
type FilterType int

const (
    FilterTypeWhitelist FilterType = 1 // Include items with this tag
    FilterTypeBlacklist FilterType = 2 // Exclude items with this tag
)

// PaginatedResult represents a single result item with its score
type PaginatedResult struct {
    Item            FoodItem // The food item
    SimilarityScore float64  // Cosine similarity score (0.0 - 1.0)
    FinalScore      float64  // Score after functionality tag weighting
}

// FoodItem represents a food item in search results
type FoodItem struct {
    ID              string   // Unique identifier
    Name            string   // Display name
    CategoryTags    []string // Category tag IDs
    FunctionalityTags []string // Functionality tag IDs
}

// PaginationResponse contains paginated search results
type PaginationResponse struct {
    Items            []FoodItem  // Food items for current page
    SimilarityScores []float64   // Corresponding similarity scores
    TotalCount       int         // Total matching items across all pages
    Page             int         // Current page number (1-indexed)
    TotalPages       int         // Total number of pages
    HasNextPage      bool        // Whether more pages exist
    HasPrevPage      bool        // Whether previous pages exist
}

// PaginationConfig holds configuration for the pagination handler
type PaginationConfig struct {
    DefaultPageSize       int     // Default results per page (10)
    MaxPageSize           int     // Maximum allowed page size (10)
    MaxTotalResults       int     // Maximum total results to consider (1000)
    TagMatchWeightBoost   float64 // Weight multiplier per matching tag (0.2)
    CacheTTLSeconds       int     // TTL for cached results (300)
    QueryTimeoutMs        int     // Database query timeout (500)
}

// PageBounds represents calculated offset and limit for database query
type PageBounds struct {
    Offset int // Number of items to skip
    Limit  int // Number of items to fetch
}
```

### 2. Logic & Algorithms (Step-by-Step)

#### 2.1 Main Pagination Flow

```
FUNCTION Paginate(ctx context.Context, request PaginationRequest) -> PaginationResponse:
    1. INPUT VALIDATION
       - IF request.Page < 1:
           request.Page = 1
       - IF request.PageSize < 1 OR request.PageSize > MaxPageSize:
           request.PageSize = DefaultPageSize (10)

    2. CALCULATE PAGE BOUNDS
       - offset = (request.Page - 1) * request.PageSize
       - limit = request.PageSize

    3. GENERATE CACHE KEY
       - cacheKey = GenerateCacheKey(request)
       - Check Redis cache (github.com/redis/go-redis/v9)
       - IF cache hit for this specific page:
           RETURN cached PaginationResponse

    4. FETCH TOTAL COUNT (parallel with step 5)
       - totalCount = FetchTotalCount(ctx, request)
       - IF totalCount == 0:
           RETURN empty PaginationResponse with Page=1, TotalCount=0

    5. FETCH RESULTS WITH SCORES
       - IF request.Mode == SearchModeSimilarity:
           results = FetchSimilarityResults(ctx, request, offset, limit)
       - ELSE:
           results = FetchTextSearchResults(ctx, request, offset, limit)

    6. APPLY FUNCTIONALITY TAG WEIGHTING (if replacement search)
       - IF request.SourceItemID != "":
           results = ApplyTagWeighting(ctx, results, request.SourceItemID)

    7. SORT BY FINAL SCORE
       - Sort results by FinalScore DESC (highest similarity first)

    8. BUILD RESPONSE
       - totalPages = CEIL(totalCount / request.PageSize)
       - response = PaginationResponse{
           Items:            ExtractItems(results),
           SimilarityScores: ExtractScores(results),
           TotalCount:       totalCount,
           Page:             request.Page,
           TotalPages:       totalPages,
           HasNextPage:      request.Page < totalPages,
           HasPrevPage:      request.Page > 1,
       }

    9. CACHE RESPONSE
       - Store in Redis with TTL = CacheTTLSeconds
       - Key: cacheKey

    10. RETURN response
```

#### 2.2 Calculate Page Bounds

```
FUNCTION CalculatePageBounds(page int, pageSize int, totalCount int) -> PageBounds:
    1. VALIDATE INPUTS
       - IF pageSize > MaxPageSize:
           pageSize = MaxPageSize
       - IF pageSize < 1:
           pageSize = DefaultPageSize

    2. CALCULATE OFFSET
       - offset = (page - 1) * pageSize
       - IF offset >= totalCount AND totalCount > 0:
           // Requested page beyond available data, return last page
           page = CEIL(totalCount / pageSize)
           offset = (page - 1) * pageSize

    3. CALCULATE LIMIT
       - limit = pageSize
       - IF offset + limit > totalCount:
           limit = totalCount - offset

    4. RETURN PageBounds{Offset: offset, Limit: limit}
```

#### 2.3 Fetch Similarity Results

```
FUNCTION FetchSimilarityResults(ctx context.Context, request PaginationRequest, offset int, limit int) -> []PaginatedResult:
    1. BUILD BASE QUERY
       - Query ARCH-003 (Similarity Engine) for cosine similarity scores
       - Pass request.Ingredients as input vector

    2. APPLY TAG FILTERS
       - FOR each filter in request.Filters:
           IF filter.FilterType == FilterTypeWhitelist:
               Add WHERE clause: item.functionality_tags @> ARRAY[$tag]
           ELSE IF filter.FilterType == FilterTypeBlacklist:
               Add WHERE clause: NOT (item.functionality_tags @> ARRAY[$tag])

    3. EXECUTE QUERY WITH PAGINATION
       - Query PostgreSQL:
         SELECT fi.id, fi.name, fi.category_tags, fi.functionality_tags,
                similarity_score
         FROM food_items fi
         JOIN similarity_scores ss ON fi.id = ss.item_id
         WHERE ss.query_vector_id = $queryVectorId
           [AND applied_filters]
         ORDER BY similarity_score DESC
         LIMIT $limit OFFSET $offset

    4. MAP TO RESULTS
       - FOR each row in query results:
           result = PaginatedResult{
               Item: FoodItem from row,
               SimilarityScore: row.similarity_score,
               FinalScore: row.similarity_score,  // Will be adjusted in step 6 of main flow
           }
           ADD result to results

    5. RETURN results
```

#### 2.4 Fetch Text Search Results

```
FUNCTION FetchTextSearchResults(ctx context.Context, request PaginationRequest, offset int, limit int) -> []PaginatedResult:
    1. NORMALIZE QUERY
       - query = LOWERCASE(TRIM(request.Query))

    2. APPLY TAG FILTERS
       - Same logic as FetchSimilarityResults

    3. EXECUTE QUERY WITH PAGINATION
       - Query PostgreSQL:
         SELECT id, name, category_tags, functionality_tags,
                1.0 AS similarity_score  -- Text search uses relevance, normalized to 1.0
         FROM food_items
         WHERE LOWER(name) LIKE '%' || $query || '%'
           [AND applied_filters]
         ORDER BY
           CASE WHEN LOWER(name) = $query THEN 0
                WHEN LOWER(name) LIKE $query || '%' THEN 1
                ELSE 2
           END,
           LENGTH(name) ASC
         LIMIT $limit OFFSET $offset

    4. MAP TO RESULTS
       - Same mapping as FetchSimilarityResults with similarity_score = 1.0

    5. RETURN results
```

#### 2.5 Apply Functionality Tag Weighting

```
FUNCTION ApplyTagWeighting(ctx context.Context, results []PaginatedResult, sourceItemID string) -> []PaginatedResult:
    1. FETCH SOURCE ITEM TAGS
       - Query PostgreSQL:
         SELECT functionality_tags FROM food_items WHERE id = $sourceItemID
       - sourceTags = result.functionality_tags
       - IF sourceTags is empty:
           RETURN results (no weighting applied)

    2. CALCULATE WEIGHTED SCORES
       - FOR each result in results:
           tagMatchCount = 0
           FOR each tag in result.Item.FunctionalityTags:
               IF tag IN sourceTags:
                   tagMatchCount++

           // Apply formula: finalScore = similarityScore * (1 + 0.2 * tagMatchCount)
           result.FinalScore = result.SimilarityScore * (1.0 + TagMatchWeightBoost * float64(tagMatchCount))

    3. RE-SORT BY FINAL SCORE
       - Sort results by FinalScore DESC

    4. RETURN results
```

#### 2.6 Fetch Total Count

```
FUNCTION FetchTotalCount(ctx context.Context, request PaginationRequest) -> int:
    1. CHECK COUNT CACHE
       - countCacheKey = "count:" + GenerateFilterHash(request)
       - Check Redis cache
       - IF cache hit:
           RETURN cached count

    2. BUILD COUNT QUERY
       - IF request.Mode == SearchModeSimilarity:
           SELECT COUNT(*) FROM food_items fi
           JOIN similarity_scores ss ON fi.id = ss.item_id
           WHERE ss.query_vector_id = $queryVectorId
             [AND applied_filters]
       - ELSE:
           SELECT COUNT(*) FROM food_items
           WHERE LOWER(name) LIKE '%' || $query || '%'
             [AND applied_filters]

    3. EXECUTE QUERY
       - count = Execute count query with QueryTimeoutMs timeout

    4. CACHE COUNT
       - Store count in Redis with TTL = CacheTTLSeconds
       - Key: countCacheKey

    5. RETURN MIN(count, MaxTotalResults)
```

#### 2.7 Generate Cache Key

```
FUNCTION GenerateCacheKey(request PaginationRequest) -> string:
    1. BUILD KEY COMPONENTS
       - components = []string{
           "pagination",
           request.Mode.String(),
           request.Query,
           fmt.Sprintf("page:%d", request.Page),
           fmt.Sprintf("size:%d", request.PageSize),
           SortedFilterHash(request.Filters),
           SortedIngredientsHash(request.Ingredients),
           request.SourceItemID,
       }

    2. GENERATE HASH
       - concatenated = strings.Join(components, ":")
       - hash = MD5(concatenated)

    3. RETURN "pagination:" + hash
```

### 3. State Management & Error Handling

#### 3.1 Error States

| Error State | Cause | Detection | Response | HTTP Status |
|:------------|:------|:----------|:---------|:------------|
| Invalid Page Number | Page < 1 | Validation check | Default to page 1, proceed | 200 OK |
| Page Beyond Range | Page > TotalPages | offset >= totalCount | Return last available page | 200 OK |
| Invalid Page Size | PageSize < 1 or > 10 | Validation check | Default to 10, proceed | 200 OK |
| Empty Query (Text Mode) | Query empty in text mode | len(query) == 0 | Return empty results | 200 OK |
| No Results Found | Query/filters return no items | totalCount == 0 | Return empty PaginationResponse | 200 OK |
| Database Timeout | PostgreSQL query exceeds 500ms | Context deadline exceeded | Return cached results if available, else 503 | 503 Service Unavailable |
| Database Connection Error | PostgreSQL connection failed | lib/pq or pgx connection error | Log error, return 503 | 503 Service Unavailable |
| Redis Unavailable | Redis connection failed | github.com/redis/go-redis/v9 error | Proceed without cache (degrade gracefully) | 200 OK |
| Similarity Engine Unavailable | ARCH-003 not responding | Timeout or connection error | Fall back to text search if query provided, else return error | 200 OK or 503 |
| Source Item Not Found | SourceItemID doesn't exist | No rows returned | Skip tag weighting, proceed with base scores | 200 OK |

#### 3.2 State Transitions

```
                    ┌─────────────┐
                    │    IDLE     │
                    └──────┬──────┘
                           │ Receive PaginationRequest
                           ▼
                    ┌─────────────┐
                    │ VALIDATING  │
                    └──────┬──────┘
                           │
              ┌────────────┴────────────┐
              │ Valid                   │
              ▼                         ▼
    ┌─────────────────┐     ┌─────────────────┐
    │ CHECK_CACHE     │     │ (normalize page │
    └────────┬────────┘     │  to valid range)│
             │              └─────────────────┘
    ┌────────┴────────┐
    │ Cache Hit       │ Cache Miss
    ▼                 ▼
┌─────────┐   ┌─────────────┐
│RETURN   │   │FETCH_COUNT  │◄──────┐
│CACHED   │   └──────┬──────┘       │
└─────────┘          │              │
                     ▼              │
              ┌─────────────┐       │
              │ Count == 0? │       │
              └──────┬──────┘       │
              │ Yes  │ No           │
              ▼      ▼              │
    ┌─────────┐ ┌─────────────┐     │
    │RETURN   │ │FETCH_RESULTS│     │
    │EMPTY    │ └──────┬──────┘     │
    └─────────┘        │            │
                       ▼            │
              ┌─────────────┐       │
              │SourceItem?  │       │
              └──────┬──────┘       │
              │ Yes  │ No           │
              ▼      │              │
    ┌─────────────┐  │              │
    │APPLY_TAG    │  │              │
    │WEIGHTING    │  │              │
    └──────┬──────┘  │              │
           │         │              │
           ▼         ▼              │
        ┌─────────────┐             │
        │   SORTING   │             │
        └──────┬──────┘             │
               │                    │
               ▼                    │
        ┌─────────────┐             │
        │BUILD_RESPONSE│            │
        └──────┬──────┘             │
               │                    │
               ▼                    │
        ┌─────────────┐             │
        │   CACHING   │─────────────┘
        └──────┬──────┘   (on error, skip)
               │
               ▼
        ┌─────────────┐
        │  RETURN_OK  │
        └─────────────┘
```

#### 3.3 Performance Constraints

| Constraint | Target | Enforcement |
|:-----------|:-------|:------------|
| Total Response Time | < 500ms | Context timeout of 500ms on entire operation |
| Cache Lookup | < 5ms | Redis GET with 5ms timeout |
| Count Query | < 100ms | Query timeout + indexed columns |
| Results Query | < 300ms | Query timeout + LIMIT/OFFSET + indexes |
| Tag Weighting | < 50ms | In-memory computation, source tags cached |
| Max Results Per Page | 10 | Hardcoded MaxPageSize limit |
| Max Total Results | 1000 | Cap on totalCount to prevent expensive deep pagination |

#### 3.4 Pagination Edge Cases

| Scenario | Behavior |
|:---------|:---------|
| Page 0 requested | Normalize to page 1 |
| Negative page requested | Normalize to page 1 |
| Page beyond total pages | Return empty items with correct TotalCount/TotalPages |
| PageSize > 10 | Cap at 10 |
| PageSize = 0 | Default to 10 |
| TotalCount = 0 | Return Page=1, TotalPages=0, HasNextPage=false, HasPrevPage=false |
| Exactly 10 results total | Page 1: 10 items, HasNextPage=false |
| 11 results total | Page 1: 10 items, HasNextPage=true; Page 2: 1 item, HasNextPage=false |

### 4. Component Interfaces

#### 4.1 Public Interface

```go
// PaginationHandler handles pagination of search results
type PaginationHandler interface {
    // Paginate returns a page of search results sorted by similarity score descending.
    // Respects the max 10 results per page constraint.
    // Applies functionality tag weighting for replacement searches.
    // Parameters:
    //   - ctx: Context with timeout (should have 500ms deadline)
    //   - request: Pagination request parameters
    // Returns:
    //   - PaginationResponse with items and pagination metadata
    //   - error if database operation fails
    Paginate(ctx context.Context, request PaginationRequest) (PaginationResponse, error)
}
```

#### 4.2 Internal Functions

```go
// NewPaginationHandler creates a new pagination handler instance.
// Parameters:
//   - repo: Data repository for fetching food items (ARCH-005)
//   - similarityEngine: Similarity engine client (ARCH-003)
//   - cache: Redis cache client (ARCH-011)
//   - config: Pagination configuration
// Returns:
//   - PaginationHandler implementation
func NewPaginationHandler(
    repo FoodItemRepository,
    similarityEngine SimilarityEngine,
    cache *redis.Client,
    config PaginationConfig,
) PaginationHandler

// SimilarityEngine defines the interface for cosine similarity calculations
type SimilarityEngine interface {
    // CalculateSimilarity computes similarity scores for all items against input ingredients.
    // Parameters:
    //   - ctx: Context with timeout
    //   - ingredients: List of ingredient names for the query vector
    // Returns:
    //   - queryVectorID: Identifier for cached query vector
    //   - error if calculation fails
    CalculateSimilarity(ctx context.Context, ingredients []string) (queryVectorID string, error)
}

// calculatePageBounds computes offset and limit for database query.
// Parameters:
//   - page: 1-indexed page number
//   - pageSize: Number of items per page
//   - totalCount: Total number of matching items
// Returns:
//   - PageBounds with calculated offset and limit
func calculatePageBounds(page, pageSize, totalCount int) PageBounds

// fetchSimilarityResults retrieves results from similarity search with pagination.
// Parameters:
//   - ctx: Context with timeout
//   - request: Pagination request
//   - bounds: Calculated page bounds
// Returns:
//   - slice of PaginatedResult
//   - error if database query fails
func fetchSimilarityResults(
    ctx context.Context,
    request PaginationRequest,
    bounds PageBounds,
) ([]PaginatedResult, error)

// fetchTextSearchResults retrieves results from text search with pagination.
// Parameters:
//   - ctx: Context with timeout
//   - request: Pagination request
//   - bounds: Calculated page bounds
// Returns:
//   - slice of PaginatedResult
//   - error if database query fails
func fetchTextSearchResults(
    ctx context.Context,
    request PaginationRequest,
    bounds PageBounds,
) ([]PaginatedResult, error)

// applyTagWeighting applies functionality tag boost to similarity scores.
// Parameters:
//   - ctx: Context with timeout
//   - results: Results to weight
//   - sourceItemID: Source item for tag comparison
// Returns:
//   - slice of PaginatedResult with updated FinalScore
//   - error if source item lookup fails
func applyTagWeighting(
    ctx context.Context,
    results []PaginatedResult,
    sourceItemID string,
) ([]PaginatedResult, error)

// fetchTotalCount retrieves the total count of matching items.
// Parameters:
//   - ctx: Context with timeout
//   - request: Pagination request (for filters and mode)
// Returns:
//   - total count (capped at MaxTotalResults)
//   - error if database query fails
func fetchTotalCount(ctx context.Context, request PaginationRequest) (int, error)

// generateCacheKey creates a deterministic cache key for a pagination request.
// Parameters:
//   - request: Pagination request
// Returns:
//   - Cache key string in format "pagination:<md5_hash>"
func generateCacheKey(request PaginationRequest) string

// buildFilterClauses generates SQL WHERE clauses from tag filters.
// Parameters:
//   - filters: Tag filters to apply
// Returns:
//   - SQL clause string
//   - slice of parameter values
func buildFilterClauses(filters []TagFilter) (string, []interface{})
```

#### 4.3 Default Configuration Values

```go
var DefaultPaginationConfig = PaginationConfig{
    DefaultPageSize:     10,
    MaxPageSize:         10,
    MaxTotalResults:     1000,
    TagMatchWeightBoost: 0.2,
    CacheTTLSeconds:     300,
    QueryTimeoutMs:      500,
}
```

#### 4.4 SQL Queries

```sql
-- FetchTotalCount (similarity mode): Count matching items for similarity search
SELECT COUNT(*)
FROM food_items fi
JOIN similarity_scores ss ON fi.id = ss.item_id
WHERE ss.query_vector_id = $1
  AND ($2::text[] IS NULL OR fi.functionality_tags @> $2)
  AND ($3::text[] IS NULL OR NOT fi.functionality_tags && $3);

-- FetchTotalCount (text mode): Count matching items for text search
SELECT COUNT(*)
FROM food_items
WHERE LOWER(name) LIKE '%' || $1 || '%'
  AND ($2::text[] IS NULL OR functionality_tags @> $2)
  AND ($3::text[] IS NULL OR NOT functionality_tags && $3);

-- FetchSimilarityResults: Paginated similarity search results
SELECT fi.id, fi.name, fi.category_tags, fi.functionality_tags,
       ss.score AS similarity_score
FROM food_items fi
JOIN similarity_scores ss ON fi.id = ss.item_id
WHERE ss.query_vector_id = $1
  AND ($2::text[] IS NULL OR fi.functionality_tags @> $2)
  AND ($3::text[] IS NULL OR NOT fi.functionality_tags && $3)
ORDER BY ss.score DESC
LIMIT $4 OFFSET $5;

-- FetchTextSearchResults: Paginated text search results
SELECT id, name, category_tags, functionality_tags,
       1.0 AS similarity_score
FROM food_items
WHERE LOWER(name) LIKE '%' || $1 || '%'
  AND ($2::text[] IS NULL OR functionality_tags @> $2)
  AND ($3::text[] IS NULL OR NOT functionality_tags && $3)
ORDER BY
  CASE
    WHEN LOWER(name) = $1 THEN 0
    WHEN LOWER(name) LIKE $1 || '%' THEN 1
    ELSE 2
  END,
  LENGTH(name) ASC
LIMIT $4 OFFSET $5;

-- FetchSourceItemTags: Get functionality tags for tag weighting
SELECT functionality_tags
FROM food_items
WHERE id = $1;
```

**Required Indexes:**
```sql
-- Support text search with filter clauses
CREATE INDEX idx_food_items_name_lower ON food_items (LOWER(name));
CREATE INDEX idx_food_items_functionality_tags ON food_items USING GIN (functionality_tags);

-- Support similarity score joins
CREATE INDEX idx_similarity_scores_query_vector ON similarity_scores (query_vector_id);
CREATE INDEX idx_similarity_scores_item_score ON similarity_scores (item_id, score DESC);
```

#### 4.5 Redis Cache Schema

| Key Pattern | Value Type | TTL | Description |
|:------------|:-----------|:----|:------------|
| `pagination:<md5_hash>` | JSON | 300s | Serialized PaginationResponse |
| `count:<md5_hash>` | int | 300s | Total count for filter combination |

**Example:**
```
Key: pagination:f4a3b2c1d0...
Value: {
  "items": [{"id":"uuid1","name":"Apple",...},...],
  "similarityScores": [0.95, 0.87, 0.82, ...],
  "totalCount": 156,
  "page": 1,
  "totalPages": 16,
  "hasNextPage": true,
  "hasPrevPage": false
}
TTL: 300

Key: count:a1b2c3d4...
Value: 156
TTL: 300
```
