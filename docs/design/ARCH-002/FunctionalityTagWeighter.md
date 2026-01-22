## FILE: FunctionalityTagWeighter.md
**Traceability:** ARCH-002, SW-REQ-031

### 1. Data Structures & Types

```go
// FunctionalityTag represents a tag describing the functional purpose of a food item
// Examples: "protein-source", "carb-base", "sweetener", "binder", "thickener"
type FunctionalityTag struct {
    ID   string // Unique identifier of the tag
    Name string // Human-readable tag name
}

// WeightedItem represents a search result with combined similarity and tag weight scores
type WeightedItem struct {
    ItemID          string            // Unique identifier of the food item
    Name            string            // Display name of the food item
    SimilarityScore float64           // Cosine similarity score from ARCH-003 (0.0 - 1.0)
    TagMatchCount   int               // Number of functionality tags matching the source item
    MatchedTags     []FunctionalityTag // Tags that matched between source and this item
    FinalScore      float64           // Combined score after weighting
}

// WeightingRequest represents the input for functionality tag weighting
type WeightingRequest struct {
    SourceItemID string          // ID of the item being replaced
    Candidates   []CandidateItem // Items to weight (from similarity search)
}

// CandidateItem represents a candidate from the similarity search
type CandidateItem struct {
    ItemID          string  // Unique identifier of the food item
    Name            string  // Display name of the food item
    SimilarityScore float64 // Cosine similarity score from ARCH-003
}

// WeightingResponse contains weighted and re-ranked results
type WeightingResponse struct {
    Items       []WeightedItem // Ordered by FinalScore descending (best first)
    SourceTags  []FunctionalityTag // Functionality tags of the source item
    ProcessTime int64          // Processing time in milliseconds
}

// WeighterConfig holds configuration for the functionality tag weighter
type WeighterConfig struct {
    TagMatchBoostFactor float64 // Boost multiplier per matching tag (default: 0.2)
    MaxBoostCap         float64 // Maximum total boost cap (default: 1.0, i.e., 100% max boost)
    CacheTTLSeconds     int     // TTL for cached tag lookups (default: 600)
}
```

### 2. Logic & Algorithms (Step-by-Step)

#### 2.1 Main Weighting Flow

```
FUNCTION WeightByFunctionalityTags(ctx context.Context, request WeightingRequest) -> WeightingResponse:
    1. INPUT VALIDATION
       - IF sourceItemID is empty:
           RETURN error: "source item ID required"
       - IF candidates is empty:
           RETURN empty WeightingResponse with ProcessTime

    2. FETCH SOURCE ITEM TAGS
       - Generate cache key: "func_tags:" + sourceItemID
       - Check Redis cache (github.com/redis/go-redis/v9)
       - IF cache hit:
           sourceTags = cached tags
       - ELSE:
           Query PostgreSQL for source item functionality tags:
           SELECT ft.id, ft.name
           FROM functionality_tags ft
           INNER JOIN food_item_functionality_tags fift ON ft.id = fift.tag_id
           WHERE fift.food_item_id = $1

           Cache result with TTL = CacheTTLSeconds
           sourceTags = query result

    3. IF sourceTags is empty:
       - No weighting possible; return candidates with FinalScore = SimilarityScore
       - FOR each candidate in candidates:
           weightedItem = WeightedItem{
               ItemID: candidate.ItemID,
               Name: candidate.Name,
               SimilarityScore: candidate.SimilarityScore,
               TagMatchCount: 0,
               MatchedTags: [],
               FinalScore: candidate.SimilarityScore
           }
           ADD to weightedItems
       - RETURN WeightingResponse with weightedItems

    4. BUILD SOURCE TAG SET
       - sourceTagSet = SET of tag IDs from sourceTags
       - sourceTagMap = MAP of tag ID -> FunctionalityTag

    5. BATCH FETCH CANDIDATE TAGS
       - Extract all candidate item IDs
       - Query PostgreSQL for all candidate tags in single query:
         SELECT fift.food_item_id, ft.id, ft.name
         FROM functionality_tags ft
         INNER JOIN food_item_functionality_tags fift ON ft.id = fift.tag_id
         WHERE fift.food_item_id = ANY($1)
       - Build candidateTagsMap: MAP of itemID -> []FunctionalityTag

    6. CALCULATE WEIGHTED SCORES
       - FOR each candidate in candidates:
           candidateTags = candidateTagsMap[candidate.ItemID] OR []
           matchedTags = []
           tagMatchCount = 0

           FOR each tag in candidateTags:
               IF tag.ID IN sourceTagSet:
                   tagMatchCount++
                   ADD sourceTagMap[tag.ID] to matchedTags

           boost = CalculateBoost(tagMatchCount, config)
           finalScore = candidate.SimilarityScore * (1 + boost)

           weightedItem = WeightedItem{
               ItemID: candidate.ItemID,
               Name: candidate.Name,
               SimilarityScore: candidate.SimilarityScore,
               TagMatchCount: tagMatchCount,
               MatchedTags: matchedTags,
               FinalScore: finalScore
           }
           ADD to weightedItems

    7. SORT BY FINAL SCORE
       - Sort weightedItems by FinalScore DESC (higher score = better match)
       - IF FinalScores are equal, sort by SimilarityScore DESC
       - IF SimilarityScores are equal, sort by Name ASC

    8. RETURN WeightingResponse with weightedItems, sourceTags, and processTime
```

#### 2.2 Boost Calculation Algorithm

```
FUNCTION CalculateBoost(tagMatchCount int, config WeighterConfig) -> float64:
    1. CALCULATE RAW BOOST
       rawBoost = tagMatchCount * config.TagMatchBoostFactor

    2. APPLY CAP
       boost = MIN(rawBoost, config.MaxBoostCap)

    3. RETURN boost
```

**Boost Examples (with default config: TagMatchBoostFactor=0.2, MaxBoostCap=1.0):**

| Tag Matches | Raw Boost | Capped Boost | Final Score Multiplier |
|:------------|:----------|:-------------|:-----------------------|
| 0 | 0.0 | 0.0 | 1.0x |
| 1 | 0.2 | 0.2 | 1.2x |
| 2 | 0.4 | 0.4 | 1.4x |
| 3 | 0.6 | 0.6 | 1.6x |
| 4 | 0.8 | 0.8 | 1.8x |
| 5+ | 1.0+ | 1.0 | 2.0x (capped) |

#### 2.3 Final Score Formula

```
finalScore = similarityScore * (1 + boost)

where:
  boost = MIN(tagMatchCount * TagMatchBoostFactor, MaxBoostCap)
```

**Worked Example:**

Source item: "Chicken Breast" with tags ["protein-source", "lean-meat", "grillable"]

| Candidate | Similarity | Matching Tags | Tag Count | Boost | Final Score |
|:----------|:-----------|:--------------|:----------|:------|:------------|
| Turkey Breast | 0.92 | protein-source, lean-meat, grillable | 3 | 0.6 | 0.92 * 1.6 = 1.472 |
| Tofu | 0.85 | protein-source | 1 | 0.2 | 0.85 * 1.2 = 1.020 |
| Salmon | 0.88 | protein-source, grillable | 2 | 0.4 | 0.88 * 1.4 = 1.232 |
| White Rice | 0.45 | (none) | 0 | 0.0 | 0.45 * 1.0 = 0.450 |

**Resulting Rank (by FinalScore DESC):**
1. Turkey Breast (1.472)
2. Salmon (1.232)
3. Tofu (1.020)
4. White Rice (0.450)

### 3. State Management & Error Handling

#### 3.1 Error States

| Error State | Cause | Detection | Response | HTTP Status |
|:------------|:------|:----------|:---------|:------------|
| Empty Source ID | SourceItemID not provided | len(sourceItemID) == 0 | Return error response | 400 Bad Request |
| Source Item Not Found | SourceItemID doesn't exist in database | No rows returned for source item | Return error: "source item not found" | 404 Not Found |
| Empty Candidates | No candidates provided | len(candidates) == 0 | Return empty WeightingResponse | 200 OK |
| Database Timeout | PostgreSQL query exceeds 100ms | Context deadline exceeded | Return candidates with SimilarityScore as FinalScore (no weighting) | 200 OK |
| Database Connection Error | PostgreSQL connection failed | lib/pq or pgx connection error | Log error, return candidates unweighted | 200 OK (degraded) |
| Redis Unavailable | Redis connection failed | github.com/redis/go-redis/v9 error | Proceed without cache (degrade gracefully) | 200 OK |

#### 3.2 State Transitions

```
                    ┌─────────────┐
                    │   IDLE      │
                    └──────┬──────┘
                           │ Receive WeightingRequest
                           ▼
                    ┌─────────────┐
                    │ VALIDATING  │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │ Invalid    │ Valid      │
              ▼            ▼            │
    ┌─────────────┐ ┌─────────────────┐ │
    │RETURN_ERROR │ │FETCH_SOURCE_TAGS│ │
    └─────────────┘ └───────┬─────────┘ │
                            │           │
               ┌────────────┼───────────┤
               │ Cache Hit  │Cache Miss │
               ▼            ▼           │
    ┌─────────────────┐ ┌──────────────┐│
    │USE_CACHED_TAGS  │ │ QUERY_DB     ││
    └────────┬────────┘ └──────┬───────┘│
             │                 │         │
             └────────┬────────┘         │
                      │                  │
                      ▼                  │
            ┌─────────────────┐          │
            │ CHECK_TAGS_EMPTY│          │
            └────────┬────────┘          │
                     │                   │
        ┌────────────┼───────────┐       │
        │ No Tags    │ Has Tags  │       │
        ▼            ▼           │       │
┌───────────────┐ ┌─────────────┐│       │
│RETURN_UNWEIGHT│ │FETCH_CAND_  ││       │
│ED_RESULTS     │ │TAGS_BATCH   ││       │
└───────────────┘ └──────┬──────┘│       │
                         │       │       │
                         ▼       │       │
                  ┌─────────────┐│       │
                  │CALC_WEIGHTS ││       │
                  └──────┬──────┘│       │
                         │       │       │
                         ▼       │       │
                  ┌─────────────┐│       │
                  │   SORTING   │◄───────┘
                  └──────┬──────┘
                         │
                         ▼
                  ┌─────────────┐
                  │ RETURN_OK   │
                  └─────────────┘
```

#### 3.3 Graceful Degradation

When the weighting operation fails or times out, the component degrades gracefully:

```
FUNCTION WeightWithFallback(ctx context.Context, request WeightingRequest) -> WeightingResponse:
    1. TRY full weighting
       result, err = WeightByFunctionalityTags(ctx, request)

    2. IF err != nil OR context timeout:
       - Log warning: "functionality tag weighting degraded"
       - Return fallback response (candidates sorted by SimilarityScore only)

       fallbackItems = []
       FOR each candidate in request.Candidates:
           fallbackItems = append(fallbackItems, WeightedItem{
               ItemID: candidate.ItemID,
               Name: candidate.Name,
               SimilarityScore: candidate.SimilarityScore,
               TagMatchCount: 0,
               MatchedTags: nil,
               FinalScore: candidate.SimilarityScore,
           })

       Sort fallbackItems by FinalScore DESC
       RETURN WeightingResponse{Items: fallbackItems, SourceTags: nil}

    3. RETURN result
```

#### 3.4 Performance Constraints

| Constraint | Target | Enforcement |
|:-----------|:-------|:------------|
| Total Processing Time | < 50ms | Context timeout of 45ms on DB queries |
| Tag Cache Lookup | < 5ms | Redis GET with 5ms timeout |
| Source Tag Query | < 20ms | Query timeout + indexed lookup |
| Batch Candidate Query | < 30ms | Query timeout + ANY() with index |
| Memory | < 500KB per request | Candidate limit inherited from search (max 100) |

### 4. Component Interfaces

#### 4.1 Public Interface

```go
// FunctionalityTagWeighter applies relevance boost based on matching functionality tags
type FunctionalityTagWeighter interface {
    // Weight applies functionality tag weighting to similarity search results.
    // During replacement searches, items sharing functionality tags with the
    // source item receive a score boost, prioritizing contextually appropriate
    // replacements.
    //
    // Parameters:
    //   - ctx: Context with timeout (should have ~50ms budget)
    //   - request: WeightingRequest containing source item and candidates
    //
    // Returns:
    //   - WeightingResponse with re-ranked items
    //   - error if source item not found or validation fails
    //
    // On timeout or database errors, returns candidates with SimilarityScore
    // as FinalScore (graceful degradation).
    Weight(ctx context.Context, request WeightingRequest) (WeightingResponse, error)
}
```

#### 4.2 Internal Functions

```go
// NewFunctionalityTagWeighter creates a new weighter instance with the given dependencies.
//
// Parameters:
//   - repo: Tag repository for fetching functionality tags (ARCH-005)
//   - cache: Redis cache client (ARCH-011)
//   - config: Weighter configuration
//
// Returns:
//   - FunctionalityTagWeighter implementation
func NewFunctionalityTagWeighter(
    repo TagRepository,
    cache *redis.Client,
    config WeighterConfig,
) FunctionalityTagWeighter

// TagRepository defines the interface for fetching functionality tags
type TagRepository interface {
    // GetFunctionalityTagsByItemID returns functionality tags for a food item.
    //
    // Parameters:
    //   - ctx: Context with timeout
    //   - itemID: Food item UUID
    //
    // Returns:
    //   - slice of FunctionalityTag
    //   - error if database operation fails
    GetFunctionalityTagsByItemID(ctx context.Context, itemID string) ([]FunctionalityTag, error)

    // GetFunctionalityTagsByItemIDs returns functionality tags for multiple food items.
    // Used for batch fetching candidate tags.
    //
    // Parameters:
    //   - ctx: Context with timeout
    //   - itemIDs: Slice of food item UUIDs
    //
    // Returns:
    //   - map of itemID -> []FunctionalityTag
    //   - error if database operation fails
    GetFunctionalityTagsByItemIDs(ctx context.Context, itemIDs []string) (map[string][]FunctionalityTag, error)
}

// calculateBoost computes the score boost for a given number of tag matches.
//
// Parameters:
//   - tagMatchCount: Number of functionality tags matching the source item
//   - config: Weighter configuration
//
// Returns:
//   - boost: Multiplier to add to base score (capped by MaxBoostCap)
func calculateBoost(tagMatchCount int, config WeighterConfig) float64

// findMatchingTags identifies which tags from candidateTags exist in sourceTagSet.
//
// Parameters:
//   - candidateTags: Tags belonging to a candidate item
//   - sourceTagSet: Set of tag IDs from the source item
//   - sourceTagMap: Map of tag ID to FunctionalityTag for lookup
//
// Returns:
//   - matchedTags: Slice of tags that matched
//   - matchCount: Number of matches
func findMatchingTags(
    candidateTags []FunctionalityTag,
    sourceTagSet map[string]struct{},
    sourceTagMap map[string]FunctionalityTag,
) (matchedTags []FunctionalityTag, matchCount int)

// generateTagCacheKey creates a deterministic cache key for tag lookup.
//
// Parameters:
//   - itemID: Food item UUID
//
// Returns:
//   - Cache key string in format "func_tags:<itemID>"
func generateTagCacheKey(itemID string) string

// sortWeightedItems sorts items by FinalScore descending, with tiebreakers.
//
// Parameters:
//   - items: Slice of WeightedItem to sort in place
//
// Sort order:
//   1. FinalScore DESC
//   2. SimilarityScore DESC (tiebreaker)
//   3. Name ASC (tiebreaker)
func sortWeightedItems(items []WeightedItem)
```

#### 4.3 Default Configuration Values

```go
var DefaultWeighterConfig = WeighterConfig{
    TagMatchBoostFactor: 0.2,  // 20% boost per matching tag
    MaxBoostCap:         1.0,  // Maximum 100% total boost (2x multiplier)
    CacheTTLSeconds:     600,  // 10 minute cache for tag lookups
}
```

#### 4.4 SQL Queries

```sql
-- GetFunctionalityTagsByItemID: Fetch tags for a single item
SELECT ft.id, ft.name
FROM functionality_tags ft
INNER JOIN food_item_functionality_tags fift ON ft.id = fift.tag_id
WHERE fift.food_item_id = $1;

-- GetFunctionalityTagsByItemIDs: Batch fetch tags for multiple items
SELECT fift.food_item_id, ft.id, ft.name
FROM functionality_tags ft
INNER JOIN food_item_functionality_tags fift ON ft.id = fift.tag_id
WHERE fift.food_item_id = ANY($1);
```

**Required Indexes:**

```sql
-- Index for single item tag lookup
CREATE INDEX idx_food_item_func_tags_item_id
ON food_item_functionality_tags (food_item_id);

-- Index for batch tag lookup
CREATE INDEX idx_food_item_func_tags_item_id_tag_id
ON food_item_functionality_tags (food_item_id, tag_id);

-- Index on functionality_tags for name lookups (if needed)
CREATE INDEX idx_functionality_tags_name
ON functionality_tags (name);
```

#### 4.5 Redis Cache Schema

| Key Pattern | Value Type | TTL | Description |
|:------------|:-----------|:----|:------------|
| `func_tags:<item_id>` | JSON | 600s | Serialized []FunctionalityTag for an item |

**Example:**

```
Key: func_tags:550e8400-e29b-41d4-a716-446655440000
Value: [{"id":"tag-001","name":"protein-source"},{"id":"tag-002","name":"lean-meat"}]
TTL: 600
```

#### 4.6 Integration with Search Flow

The FunctionalityTagWeighter is called by the SearchController during replacement searches:

```
SearchController.Search(request):
    1. Parse query and determine search mode

    2. IF mode == REPLACEMENT_SEARCH:
       a. Get similarity results from ARCH-003 (Similarity Engine)
          similarityResults = SimilarityEngine.FindSimilar(sourceItem, threshold)

       b. Apply functionality tag weighting
          candidates = map similarityResults to CandidateItem
          weightingRequest = WeightingRequest{
              SourceItemID: request.sourceItemID,
              Candidates: candidates
          }
          weightedResults = FunctionalityTagWeighter.Weight(ctx, weightingRequest)

       c. Apply pagination to weighted results
          paginatedResults = Paginate(weightedResults.Items, request.page)

       d. Return SearchResponse with weighted, paginated results

    3. ELSE (text search):
       ... normal search flow ...
```
