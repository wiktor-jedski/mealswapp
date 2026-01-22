## FILE: FilterProcessor.md
**Traceability:** ARCH-002

### 1. Data Structures & Types

```go
// TagFilter represents a single filter criterion for food items
type TagFilter struct {
    TagName   string     // Name of the tag to filter by (e.g., "vegan", "gluten-free")
    Mode      FilterMode // Whether to include or exclude items with this tag
}

// FilterMode indicates whether a tag filter is inclusive or exclusive
type FilterMode int

const (
    FilterModeWhitelist FilterMode = iota // Include only items WITH this tag
    FilterModeBlacklist                   // Exclude items WITH this tag
)

// FilterRequest represents the input for filtering operations
type FilterRequest struct {
    ItemIDs  []string    // List of food item IDs to filter (from search results)
    Filters  []TagFilter // Tag filters to apply
}

// FilterResult represents the output of a filtering operation
type FilterResult struct {
    FilteredIDs   []string          // IDs of items that passed all filters
    ExcludedIDs   []string          // IDs of items that were filtered out
    FilterStats   FilterStatistics  // Statistics about the filtering operation
    ProcessingMs  int64             // Processing time in milliseconds
}

// FilterStatistics provides breakdown of filtering results
type FilterStatistics struct {
    TotalInput       int            // Number of items before filtering
    TotalOutput      int            // Number of items after filtering
    ExcludedByTag    map[string]int // Count of items excluded per tag
    WhitelistApplied []string       // Tags used for whitelist filtering
    BlacklistApplied []string       // Tags used for blacklist filtering
}

// FoodItemTags represents a food item with its associated tags
type FoodItemTags struct {
    ID   string   // Unique identifier of the food item
    Tags []string // List of tags associated with the item
}

// FilterProcessorConfig holds configuration for the filter processor
type FilterProcessorConfig struct {
    MaxFiltersPerRequest   int  // Maximum number of filters allowed (default: 20)
    MaxItemsPerBatch       int  // Maximum items to process in single batch (default: 1000)
    EnableFilterStats      bool // Whether to compute detailed statistics (default: true)
    CacheTagLookups        bool // Whether to cache tag lookups (default: true)
    TagCacheTTLSeconds     int  // TTL for cached tag data (default: 600)
}

// TagIndex represents an in-memory index for fast tag lookups
type TagIndex struct {
    TagToItems map[string]map[string]struct{} // tag -> set of item IDs
    ItemToTags map[string][]string            // item ID -> list of tags
}
```

### 2. Logic & Algorithms (Step-by-Step)

#### 2.1 Main Filtering Flow

```
FUNCTION ProcessFilters(ctx context.Context, request FilterRequest) -> FilterResult:
    1. INPUT VALIDATION
       - IF len(request.Filters) == 0:
           RETURN FilterResult with all input IDs as FilteredIDs (no filtering)
       - IF len(request.Filters) > MaxFiltersPerRequest:
           RETURN error: ErrTooManyFilters
       - IF len(request.ItemIDs) == 0:
           RETURN FilterResult with empty FilteredIDs
       - IF len(request.ItemIDs) > MaxItemsPerBatch:
           Process in batches of MaxItemsPerBatch

    2. SEPARATE FILTERS BY MODE
       - whitelistFilters = []TagFilter{}
       - blacklistFilters = []TagFilter{}
       - FOR each filter in request.Filters:
           IF filter.Mode == FilterModeWhitelist:
               APPEND filter to whitelistFilters
           ELSE:
               APPEND filter to blacklistFilters

    3. FETCH TAG DATA
       - Check cache for item tags (key pattern: "item_tags:<item_id>")
       - FOR uncached items:
           Query PostgreSQL for tags:
           SELECT fit.food_item_id, t.name
           FROM food_item_tags fit
           JOIN tags t ON fit.tag_id = t.id
           WHERE fit.food_item_id = ANY($1)
       - Build TagIndex for efficient lookups
       - Cache results with TTL = TagCacheTTLSeconds

    4. APPLY WHITELIST FILTERS (AND logic)
       - candidateIDs = request.ItemIDs
       - FOR each filter in whitelistFilters:
           candidateIDs = ApplyWhitelistFilter(candidateIDs, filter.TagName, tagIndex)
           Record excluded items in FilterStats.ExcludedByTag

    5. APPLY BLACKLIST FILTERS (OR logic)
       - FOR each filter in blacklistFilters:
           candidateIDs = ApplyBlacklistFilter(candidateIDs, filter.TagName, tagIndex)
           Record excluded items in FilterStats.ExcludedByTag

    6. BUILD RESULT
       - FilteredIDs = candidateIDs
       - ExcludedIDs = request.ItemIDs - FilteredIDs (set difference)
       - Compute FilterStatistics

    7. RETURN FilterResult with FilteredIDs, ExcludedIDs, FilterStats
```

#### 2.2 Whitelist Filter Application

```
FUNCTION ApplyWhitelistFilter(itemIDs []string, tagName string, tagIndex TagIndex) -> []string:
    // Whitelist: KEEP only items that HAVE the specified tag

    1. LOOKUP items with tag
       - taggedItems = tagIndex.TagToItems[tagName]
       - IF taggedItems is nil or empty:
           RETURN empty slice (no items have this tag)

    2. FILTER items
       - result = []string{}
       - FOR each itemID in itemIDs:
           IF itemID EXISTS in taggedItems:
               APPEND itemID to result

    3. RETURN result
```

#### 2.3 Blacklist Filter Application

```
FUNCTION ApplyBlacklistFilter(itemIDs []string, tagName string, tagIndex TagIndex) -> []string:
    // Blacklist: REMOVE items that HAVE the specified tag

    1. LOOKUP items with tag
       - taggedItems = tagIndex.TagToItems[tagName]
       - IF taggedItems is nil or empty:
           RETURN itemIDs unchanged (no items to exclude)

    2. FILTER items
       - result = []string{}
       - FOR each itemID in itemIDs:
           IF itemID NOT EXISTS in taggedItems:
               APPEND itemID to result

    3. RETURN result
```

#### 2.4 Tag Index Building

```
FUNCTION BuildTagIndex(items []FoodItemTags) -> TagIndex:
    1. INITIALIZE index
       - index = TagIndex{
           TagToItems: make(map[string]map[string]struct{}),
           ItemToTags: make(map[string][]string),
       }

    2. POPULATE index
       - FOR each item in items:
           index.ItemToTags[item.ID] = item.Tags
           FOR each tag in item.Tags:
               IF index.TagToItems[tag] is nil:
                   index.TagToItems[tag] = make(map[string]struct{})
               index.TagToItems[tag][item.ID] = struct{}{}

    3. RETURN index
```

#### 2.5 Filter Combination Logic

| Scenario | Whitelist Tags | Blacklist Tags | Result |
|:---------|:---------------|:---------------|:-------|
| No filters | [] | [] | All items pass |
| Single whitelist | ["vegan"] | [] | Only vegan items |
| Single blacklist | [] | ["nuts"] | All items except those with nuts |
| Multiple whitelist | ["vegan", "organic"] | [] | Items with BOTH vegan AND organic |
| Multiple blacklist | [] | ["nuts", "dairy"] | Items without nuts AND without dairy |
| Combined | ["vegan"] | ["processed"] | Vegan items that are NOT processed |

#### 2.6 Batch Processing for Large Item Sets

```
FUNCTION ProcessFiltersInBatches(ctx context.Context, request FilterRequest) -> FilterResult:
    1. SPLIT into batches
       - batches = ChunkSlice(request.ItemIDs, MaxItemsPerBatch)
       - allFilteredIDs = []string{}
       - allExcludedIDs = []string{}
       - aggregatedStats = FilterStatistics{}

    2. PROCESS each batch
       - FOR each batch in batches:
           batchRequest = FilterRequest{
               ItemIDs: batch,
               Filters: request.Filters,
           }
           batchResult = ProcessFilters(ctx, batchRequest)
           APPEND batchResult.FilteredIDs to allFilteredIDs
           APPEND batchResult.ExcludedIDs to allExcludedIDs
           MergeStats(aggregatedStats, batchResult.FilterStats)

    3. RETURN combined FilterResult
```

### 3. State Management & Error Handling

#### 3.1 Error States

| Error State | Cause | Detection | Response | HTTP Status |
|:------------|:------|:----------|:---------|:------------|
| Empty Filter List | No filters provided | len(filters) == 0 | Return input unchanged (no-op) | 200 OK |
| Too Many Filters | Filters exceed MaxFiltersPerRequest | len(filters) > MaxFiltersPerRequest | Return ErrTooManyFilters | 400 Bad Request |
| Invalid Tag Name | Tag name is empty or invalid | len(strings.TrimSpace(tagName)) == 0 | Skip invalid filter, log warning | 200 OK |
| Unknown Tag | Tag does not exist in database | Tag not found in lookup | Whitelist: exclude all; Blacklist: no-op | 200 OK |
| Database Timeout | PostgreSQL query exceeds 200ms | Context deadline exceeded | Return cached tags if available, else error | 503 Service Unavailable |
| Database Connection Error | PostgreSQL connection failed | lib/pq or pgx connection error | Log error, return 503 | 503 Service Unavailable |
| Redis Unavailable | Redis connection failed | github.com/redis/go-redis/v9 error | Proceed without cache (degrade gracefully) | 200 OK |
| Empty Result Set | All items filtered out | len(filteredIDs) == 0 | Return empty FilteredIDs with stats | 200 OK |

#### 3.2 State Transitions

```
                    ┌─────────────┐
                    │    IDLE     │
                    └──────┬──────┘
                           │ Receive FilterRequest
                           ▼
                    ┌─────────────┐
                    │ VALIDATING  │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │ Invalid    │ Valid      │ No Filters
              ▼            ▼            ▼
    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
    │RETURN_ERROR │ │ SEPARATING  │ │RETURN_INPUT │
    └─────────────┘ └──────┬──────┘ └─────────────┘
                           │ Separate whitelist/blacklist
                           ▼
                    ┌─────────────┐
                    │FETCHING_TAGS│
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │ Cache Hit  │ Cache Miss │ DB Error
              ▼            ▼            ▼
    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
    │ USE_CACHED  │ │ QUERY_DB    │ │RETURN_ERROR │
    └──────┬──────┘ └──────┬──────┘ └─────────────┘
           │               │
           └───────┬───────┘
                   ▼
            ┌─────────────┐
            │BUILDING_IDX │
            └──────┬──────┘
                   ▼
            ┌─────────────┐
            │APPLY_WLIST  │ Apply whitelist filters
            └──────┬──────┘
                   ▼
            ┌─────────────┐
            │APPLY_BLIST  │ Apply blacklist filters
            └──────┬──────┘
                   ▼
            ┌─────────────┐
            │BUILD_RESULT │
            └──────┬──────┘
                   ▼
            ┌─────────────┐
            │ RETURN_OK   │
            └─────────────┘
```

#### 3.3 Performance Constraints

| Constraint | Target | Enforcement |
|:-----------|:-------|:------------|
| Total Response Time | < 50ms for 100 items | Context timeout of 45ms |
| Tag Lookup (cached) | < 5ms | Redis GET with 5ms timeout |
| Tag Lookup (uncached) | < 30ms | PostgreSQL query with LIMIT |
| Index Building | < 10ms for 1000 items | In-memory map operations |
| Filter Application | < 5ms per filter | Set operations O(n) |
| Memory | < 2MB per request | Batch processing for large sets |

#### 3.4 Filter Validation Rules

```
FUNCTION ValidateFilter(filter TagFilter) -> (bool, error):
    1. CHECK tag name
       - IF len(strings.TrimSpace(filter.TagName)) == 0:
           RETURN false, ErrEmptyTagName
       - IF len(filter.TagName) > 100:
           RETURN false, ErrTagNameTooLong

    2. CHECK mode
       - IF filter.Mode != FilterModeWhitelist AND filter.Mode != FilterModeBlacklist:
           RETURN false, ErrInvalidFilterMode

    3. SANITIZE tag name
       - Normalize to lowercase
       - Remove leading/trailing whitespace

    4. RETURN true, nil
```

#### 3.5 Graceful Degradation

```
FUNCTION ProcessFiltersWithFallback(ctx context.Context, request FilterRequest) -> FilterResult:
    1. TRY primary path
       - result, err = ProcessFilters(ctx, request)
       - IF err == nil:
           RETURN result

    2. CHECK error type
       - IF err is Redis error:
           Log warning "Redis unavailable, proceeding without cache"
           result = ProcessFiltersWithoutCache(ctx, request)
           RETURN result

       - IF err is context.DeadlineExceeded:
           Log warning "Filter timeout, returning partial results"
           RETURN FilterResult with input as FilteredIDs (skip filtering)

       - IF err is database error:
           Log error "Database error in filter processor"
           RETURN error (propagate to caller)

    3. RETURN error
```

### 4. Component Interfaces

#### 4.1 Public Interface

```go
// FilterProcessor handles tag-based filtering of food items
type FilterProcessor interface {
    // Process applies the given filters to the item IDs and returns filtered results.
    // Whitelist filters use AND logic (item must have ALL whitelisted tags).
    // Blacklist filters use OR logic (item is excluded if it has ANY blacklisted tag).
    // Returns the original IDs if no filters are provided.
    //
    // Parameters:
    //   - ctx: Context with timeout (recommended: 50ms)
    //   - request: FilterRequest containing item IDs and filters
    //
    // Returns:
    //   - FilterResult with filtered IDs, excluded IDs, and statistics
    //   - error if validation fails or database is unavailable
    Process(ctx context.Context, request FilterRequest) (FilterResult, error)

    // ValidateFilters checks if the given filters are valid.
    // Returns nil if all filters are valid, or the first validation error.
    //
    // Parameters:
    //   - filters: Slice of TagFilter to validate
    //
    // Returns:
    //   - error if any filter is invalid
    ValidateFilters(filters []TagFilter) error
}
```

#### 4.2 Internal Functions

```go
// NewFilterProcessor creates a new filter processor instance with the given dependencies.
//
// Parameters:
//   - repo: Tag repository for fetching item tags (ARCH-005)
//   - cache: Redis cache client (ARCH-011)
//   - config: Processor configuration
//
// Returns:
//   - FilterProcessor implementation
func NewFilterProcessor(
    repo TagRepository,
    cache *redis.Client,
    config FilterProcessorConfig,
) FilterProcessor

// TagRepository defines the interface for fetching tag data
type TagRepository interface {
    // GetTagsForItems returns tags for the given item IDs.
    // Results are returned as a map from item ID to tag names.
    //
    // Parameters:
    //   - ctx: Context with timeout
    //   - itemIDs: Slice of food item IDs
    //
    // Returns:
    //   - slice of FoodItemTags
    //   - error if database operation fails
    GetTagsForItems(ctx context.Context, itemIDs []string) ([]FoodItemTags, error)

    // GetItemsWithTag returns all item IDs that have the specified tag.
    //
    // Parameters:
    //   - ctx: Context with timeout
    //   - tagName: Name of the tag (case-insensitive)
    //
    // Returns:
    //   - slice of item IDs
    //   - error if database operation fails
    GetItemsWithTag(ctx context.Context, tagName string) ([]string, error)
}

// separateFilters splits filters into whitelist and blacklist groups.
//
// Parameters:
//   - filters: Slice of TagFilter
//
// Returns:
//   - whitelistFilters: Filters with FilterModeWhitelist
//   - blacklistFilters: Filters with FilterModeBlacklist
func separateFilters(filters []TagFilter) (whitelistFilters, blacklistFilters []TagFilter)

// buildTagIndex creates an in-memory index for fast tag lookups.
//
// Parameters:
//   - items: Slice of FoodItemTags from database
//
// Returns:
//   - TagIndex for efficient filtering
func buildTagIndex(items []FoodItemTags) TagIndex

// applyWhitelistFilter filters items to include only those with the specified tag.
//
// Parameters:
//   - itemIDs: Slice of item IDs to filter
//   - tagName: Tag that items must have
//   - index: TagIndex for lookups
//
// Returns:
//   - filteredIDs: Items that have the tag
//   - excludedIDs: Items that do not have the tag
func applyWhitelistFilter(itemIDs []string, tagName string, index TagIndex) (filteredIDs, excludedIDs []string)

// applyBlacklistFilter filters items to exclude those with the specified tag.
//
// Parameters:
//   - itemIDs: Slice of item IDs to filter
//   - tagName: Tag that items must NOT have
//   - index: TagIndex for lookups
//
// Returns:
//   - filteredIDs: Items that do not have the tag
//   - excludedIDs: Items that have the tag
func applyBlacklistFilter(itemIDs []string, tagName string, index TagIndex) (filteredIDs, excludedIDs []string)

// generateTagCacheKey creates a deterministic cache key for item tags.
//
// Parameters:
//   - itemID: Food item ID
//
// Returns:
//   - Cache key string in format "item_tags:<item_id>"
func generateTagCacheKey(itemID string) string

// computeStatistics calculates filtering statistics for the result.
//
// Parameters:
//   - inputIDs: Original item IDs
//   - filteredIDs: IDs that passed filters
//   - excludedByTag: Map of tag name to excluded item IDs
//   - whitelistTags: Tags used for whitelist filtering
//   - blacklistTags: Tags used for blacklist filtering
//
// Returns:
//   - FilterStatistics with counts and breakdowns
func computeStatistics(
    inputIDs []string,
    filteredIDs []string,
    excludedByTag map[string][]string,
    whitelistTags, blacklistTags []string,
) FilterStatistics
```

#### 4.3 Default Configuration Values

```go
var DefaultFilterProcessorConfig = FilterProcessorConfig{
    MaxFiltersPerRequest: 20,
    MaxItemsPerBatch:     1000,
    EnableFilterStats:    true,
    CacheTagLookups:      true,
    TagCacheTTLSeconds:   600,
}
```

#### 4.4 SQL Queries

```sql
-- GetTagsForItems: Fetch tags for multiple items
SELECT fit.food_item_id, t.name AS tag_name
FROM food_item_tags fit
JOIN tags t ON fit.tag_id = t.id
WHERE fit.food_item_id = ANY($1)
ORDER BY fit.food_item_id, t.name;

-- GetItemsWithTag: Fetch all items with a specific tag
SELECT fit.food_item_id
FROM food_item_tags fit
JOIN tags t ON fit.tag_id = t.id
WHERE LOWER(t.name) = LOWER($1);
```

**Required Indexes:**
```sql
-- Index for fast tag lookup by item ID
CREATE INDEX idx_food_item_tags_item_id ON food_item_tags (food_item_id);

-- Index for fast item lookup by tag
CREATE INDEX idx_food_item_tags_tag_id ON food_item_tags (tag_id);

-- Index for tag name lookups (case-insensitive)
CREATE INDEX idx_tags_name_lower ON tags (LOWER(name));
```

#### 4.5 Redis Cache Schema

| Key Pattern | Value Type | TTL | Description |
|:------------|:-----------|:----|:------------|
| `item_tags:<item_id>` | JSON array | 600s | List of tag names for an item |
| `tag_items:<tag_name>` | JSON array | 600s | List of item IDs with a tag |

**Examples:**
```
Key: item_tags:550e8400-e29b-41d4-a716-446655440000
Value: ["vegan","organic","gluten-free"]
TTL: 600

Key: tag_items:vegan
Value: ["550e8400-...","660f9500-...","770a1600-..."]
TTL: 600
```

#### 4.6 Error Definitions

```go
var (
    // ErrTooManyFilters is returned when filter count exceeds MaxFiltersPerRequest
    ErrTooManyFilters = errors.New("too many filters: maximum allowed is 20")

    // ErrEmptyTagName is returned when a filter has an empty tag name
    ErrEmptyTagName = errors.New("filter tag name cannot be empty")

    // ErrTagNameTooLong is returned when a tag name exceeds 100 characters
    ErrTagNameTooLong = errors.New("filter tag name exceeds maximum length of 100 characters")

    // ErrInvalidFilterMode is returned when filter mode is not whitelist or blacklist
    ErrInvalidFilterMode = errors.New("invalid filter mode: must be whitelist or blacklist")
)
```

#### 4.7 Integration with SearchController

```go
// Example usage within SearchController (ARCH-002)
func (sc *SearchController) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
    // 1. Parse and process query (QueryParser)
    // 2. Execute search (text-based or similarity-based)
    searchResults := sc.executeSearch(ctx, req.Query, req.Mode)

    // 3. Apply filters if provided
    if len(req.Filters) > 0 {
        filterReq := FilterRequest{
            ItemIDs: extractIDs(searchResults),
            Filters: req.Filters,
        }
        filterResult, err := sc.filterProcessor.Process(ctx, filterReq)
        if err != nil {
            // Log error but continue with unfiltered results
            log.Warn("filter processing failed", "error", err)
        } else {
            searchResults = filterByIDs(searchResults, filterResult.FilteredIDs)
        }
    }

    // 4. Apply pagination (PaginationHandler)
    // 5. Return response
    return buildResponse(searchResults, req.Page), nil
}
```
