## FILE: AutocompleteRanker.md
**Traceability:** ARCH-002

### 1. Data Structures & Types

```go
// AutocompleteRequest represents the input for autocomplete ranking
type AutocompleteRequest struct {
    Query     string   // User-entered search prefix (trimmed, lowercased)
    MaxResults int     // Maximum number of suggestions to return (default: 10)
}

// AutocompleteSuggestion represents a single ranked suggestion
type AutocompleteSuggestion struct {
    ItemID          string  // Unique identifier of the food item
    Name            string  // Display name of the food item
    MatchType       MatchType // Type of match that ranked this item
    Score           float64 // Composite ranking score (lower is better)
    LevenshteinDist int     // Edit distance from query (0 for exact match)
}

// MatchType indicates how the suggestion matched the query
type MatchType int

const (
    MatchTypeExact      MatchType = 1 // Exact prefix match
    MatchTypeFuzzy      MatchType = 2 // Levenshtein-based match
    MatchTypeLengthOnly MatchType = 3 // Fallback to length sorting
)

// AutocompleteResponse contains ranked suggestions
type AutocompleteResponse struct {
    Suggestions []AutocompleteSuggestion // Ordered by rank (best first)
    QueryTime   int64                     // Processing time in milliseconds
}

// CandidateItem represents a food item from the data layer
type CandidateItem struct {
    ID   string
    Name string
}

// RankerConfig holds configuration for the autocomplete ranker
type RankerConfig struct {
    MaxLevenshteinDistance int     // Maximum edit distance to consider (default: 3)
    ExactMatchBoost        float64 // Score multiplier for exact matches (default: 0.0)
    FuzzyPenaltyFactor     float64 // Penalty factor per edit distance (default: 10.0)
    LengthPenaltyFactor    float64 // Penalty factor per character length (default: 0.1)
    CacheTTLSeconds        int     // TTL for cached results (default: 300)
}
```

### 2. Logic & Algorithms (Step-by-Step)

#### 2.1 Main Ranking Flow

```
FUNCTION RankAutocomplete(request AutocompleteRequest) -> AutocompleteResponse:
    1. INPUT VALIDATION
       - IF query is empty OR length < 1:
           RETURN empty AutocompleteResponse
       - Normalize query: trim whitespace, convert to lowercase

    2. CACHE LOOKUP
       - Generate cache key: "autocomplete:" + MD5(normalized_query)
       - Check Redis cache (github.com/redis/go-redis/v9)
       - IF cache hit:
           RETURN cached AutocompleteResponse

    3. FETCH CANDIDATES
       - Query PostgreSQL for candidate items matching prefix:
         SELECT id, name FROM food_items
         WHERE LOWER(name) LIKE $1 || '%'
         ORDER BY LENGTH(name) ASC
         LIMIT 100
       - IF no exact prefix matches found:
           Fetch top 100 items by name length for fuzzy matching:
           SELECT id, name FROM food_items
           ORDER BY LENGTH(name) ASC
           LIMIT 100

    4. SCORE EACH CANDIDATE
       - FOR each candidate in candidates:
           score = CalculateScore(normalized_query, candidate)
           ADD to scored_candidates list

    5. SORT BY SCORE
       - Sort scored_candidates by Score ASC (lower score = better match)
       - IF scores are equal, sort by Name length ASC
       - IF lengths are equal, sort alphabetically

    6. TRUNCATE RESULTS
       - Take first min(request.MaxResults, len(scored_candidates)) items
       - Convert to AutocompleteSuggestion array

    7. CACHE RESULTS
       - Store in Redis with TTL = CacheTTLSeconds
       - Key: "autocomplete:" + MD5(normalized_query)

    8. RETURN AutocompleteResponse with suggestions and query time
```

#### 2.2 Score Calculation Algorithm

```
FUNCTION CalculateScore(query string, candidate CandidateItem) -> (score float64, matchType MatchType, distance int):
    normalized_name = LOWERCASE(candidate.Name)

    1. CHECK EXACT MATCH
       - IF normalized_name STARTS_WITH query:
           - IF normalized_name == query:
               RETURN (ExactMatchBoost, MatchTypeExact, 0)
           - ELSE (prefix match):
               length_penalty = (len(normalized_name) - len(query)) * LengthPenaltyFactor
               RETURN (ExactMatchBoost + length_penalty, MatchTypeExact, 0)

    2. CALCULATE LEVENSHTEIN DISTANCE
       - distance = LevenshteinDistance(query, normalized_name[0:min(len(query)+3, len(normalized_name))])
       - IF distance <= MaxLevenshteinDistance:
           fuzzy_score = distance * FuzzyPenaltyFactor
           length_penalty = len(normalized_name) * LengthPenaltyFactor
           RETURN (fuzzy_score + length_penalty, MatchTypeFuzzy, distance)

    3. FALLBACK TO LENGTH SORTING
       - length_score = 100 + len(normalized_name) * LengthPenaltyFactor
       - RETURN (length_score, MatchTypeLengthOnly, MaxLevenshteinDistance + 1)
```

#### 2.3 Levenshtein Distance Algorithm

```
FUNCTION LevenshteinDistance(s1 string, s2 string) -> int:
    // Use Wagner-Fischer algorithm with O(min(m,n)) space optimization

    1. ENSURE s1 is shorter string for space efficiency
       - IF len(s1) > len(s2):
           SWAP s1, s2

    2. INITIALIZE previous row
       - prev = array of size len(s1) + 1
       - FOR i = 0 TO len(s1):
           prev[i] = i

    3. ITERATE through s2
       - FOR j = 1 TO len(s2):
           - current = array of size len(s1) + 1
           - current[0] = j
           - FOR i = 1 TO len(s1):
               - IF s1[i-1] == s2[j-1]:
                   cost = 0
               - ELSE:
                   cost = 1
               - current[i] = MIN(
                   prev[i] + 1,        // deletion
                   current[i-1] + 1,   // insertion
                   prev[i-1] + cost    // substitution
               )
           - prev = current

    4. RETURN prev[len(s1)]
```

#### 2.4 Score Ranking Priority

The three-tier priority is implemented through the scoring system:

| Priority | Match Type | Score Range | Criteria |
|:---------|:-----------|:------------|:---------|
| 1 (highest) | Exact Match | 0.0 - 9.9 | Query is prefix of item name |
| 2 | Fuzzy Match | 10.0 - 39.9 | Levenshtein distance 1-3 |
| 3 (lowest) | Length Only | 100.0+ | No prefix/fuzzy match |

Within each tier, items are further sorted by string length (shorter = better).

### 3. State Management & Error Handling

#### 3.1 Error States

| Error State | Cause | Detection | Response | HTTP Status |
|:------------|:------|:----------|:---------|:------------|
| Empty Query | Query string is empty or whitespace only | len(strings.TrimSpace(query)) == 0 | Return empty suggestions array | 200 OK |
| Query Too Long | Query exceeds 100 characters | len(query) > 100 | Truncate to 100 chars, proceed | 200 OK |
| Database Timeout | PostgreSQL query exceeds 80ms | Context deadline exceeded | Return cached results if available, else empty array | 200 OK |
| Database Connection Error | PostgreSQL connection failed | lib/pq or pgx connection error | Log error, return cached results if available, else 503 | 503 Service Unavailable |
| Redis Unavailable | Redis connection failed | github.com/redis/go-redis/v9 error | Proceed without cache (degrade gracefully) | 200 OK |
| No Candidates Found | No items match query | len(candidates) == 0 | Return empty suggestions array | 200 OK |

#### 3.2 State Transitions

```
                    ┌─────────────┐
                    │   IDLE      │
                    └──────┬──────┘
                           │ Receive AutocompleteRequest
                           ▼
                    ┌─────────────┐
                    │ VALIDATING  │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │ Invalid    │ Valid      │
              ▼            ▼            │
    ┌─────────────┐ ┌─────────────┐    │
    │ RETURN_EMPTY│ │CACHE_LOOKUP │    │
    └─────────────┘ └──────┬──────┘    │
                           │            │
              ┌────────────┼────────────┤
              │ Cache Hit  │ Cache Miss │
              ▼            ▼            │
    ┌─────────────┐ ┌─────────────┐    │
    │RETURN_CACHED│ │ FETCHING    │    │
    └─────────────┘ └──────┬──────┘    │
                           │            │
              ┌────────────┼────────────┤
              │ DB Error   │ Success    │
              ▼            ▼            │
    ┌─────────────┐ ┌─────────────┐    │
    │RETURN_FALLBK│ │  SCORING    │    │
    └─────────────┘ └──────┬──────┘    │
                           │            │
                           ▼            │
                    ┌─────────────┐    │
                    │  SORTING    │    │
                    └──────┬──────┘    │
                           │            │
                           ▼            │
                    ┌─────────────┐    │
                    │  CACHING    │◄───┘
                    └──────┬──────┘
                           │
                           ▼
                    ┌─────────────┐
                    │ RETURN_OK   │
                    └─────────────┘
```

#### 3.3 Performance Constraints

| Constraint | Target | Enforcement |
|:-----------|:-------|:------------|
| Total Response Time | < 100ms | Context timeout of 90ms on DB query |
| Cache Lookup | < 5ms | Redis GET with 5ms timeout |
| DB Query | < 80ms | Query timeout + LIMIT 100 |
| Scoring | < 10ms | Early termination if MaxResults reached |
| Memory | < 1MB per request | Candidate limit of 100 items |

#### 3.4 Timeout Handling

```
FUNCTION ExecuteWithTimeout(ctx context.Context, timeout time.Duration, fn func() error) error:
    1. Create child context with timeout
       childCtx, cancel := context.WithTimeout(ctx, timeout)
       defer cancel()

    2. Execute function in goroutine
       done := make(chan error, 1)
       go func() {
           done <- fn()
       }()

    3. Wait for completion or timeout
       select {
           case err := <-done:
               return err
           case <-childCtx.Done():
               return context.DeadlineExceeded
       }
```

### 4. Component Interfaces

#### 4.1 Public Interface

```go
// AutocompleteRanker handles autocomplete suggestion ranking
type AutocompleteRanker interface {
    // Rank returns ranked autocomplete suggestions for the given query.
    // Returns empty slice if query is invalid or no matches found.
    // Respects the 100ms total response time constraint.
    Rank(ctx context.Context, request AutocompleteRequest) (AutocompleteResponse, error)
}
```

#### 4.2 Internal Functions

```go
// NewAutocompleteRanker creates a new ranker instance with the given dependencies.
// Parameters:
//   - repo: Data repository for fetching food items (ARCH-005)
//   - cache: Redis cache client (ARCH-011)
//   - config: Ranker configuration
// Returns:
//   - AutocompleteRanker implementation
func NewAutocompleteRanker(
    repo FoodItemRepository,
    cache *redis.Client,
    config RankerConfig,
) AutocompleteRanker

// FoodItemRepository defines the interface for fetching food item candidates
type FoodItemRepository interface {
    // FindByNamePrefix returns items whose names start with the given prefix.
    // Results are ordered by name length ascending.
    // Parameters:
    //   - ctx: Context with timeout
    //   - prefix: Lowercased search prefix
    //   - limit: Maximum number of results
    // Returns:
    //   - slice of CandidateItem
    //   - error if database operation fails
    FindByNamePrefix(ctx context.Context, prefix string, limit int) ([]CandidateItem, error)

    // FindTopByLength returns items ordered by name length.
    // Used as fallback when no prefix matches exist.
    // Parameters:
    //   - ctx: Context with timeout
    //   - limit: Maximum number of results
    // Returns:
    //   - slice of CandidateItem
    //   - error if database operation fails
    FindTopByLength(ctx context.Context, limit int) ([]CandidateItem, error)
}

// calculateScore computes the ranking score for a candidate item.
// Parameters:
//   - query: Normalized (lowercase, trimmed) search query
//   - candidate: Food item candidate
//   - config: Ranker configuration
// Returns:
//   - score: Composite ranking score (lower is better)
//   - matchType: Type of match that produced this score
//   - distance: Levenshtein distance (0 for exact matches)
func calculateScore(
    query string,
    candidate CandidateItem,
    config RankerConfig,
) (score float64, matchType MatchType, distance int)

// levenshteinDistance computes the edit distance between two strings.
// Uses space-optimized Wagner-Fischer algorithm.
// Parameters:
//   - s1: First string (will be normalized internally)
//   - s2: Second string (will be normalized internally)
// Returns:
//   - Edit distance (number of insertions, deletions, or substitutions)
func levenshteinDistance(s1, s2 string) int

// generateCacheKey creates a deterministic cache key for a query.
// Parameters:
//   - query: Normalized search query
// Returns:
//   - Cache key string in format "autocomplete:<md5_hash>"
func generateCacheKey(query string) string

// normalizeQuery prepares a query string for processing.
// Trims whitespace, converts to lowercase, truncates to 100 chars.
// Parameters:
//   - query: Raw query string
// Returns:
//   - Normalized query string
func normalizeQuery(query string) string
```

#### 4.3 Default Configuration Values

```go
var DefaultRankerConfig = RankerConfig{
    MaxLevenshteinDistance: 3,
    ExactMatchBoost:        0.0,
    FuzzyPenaltyFactor:     10.0,
    LengthPenaltyFactor:    0.1,
    CacheTTLSeconds:        300,
}
```

#### 4.4 SQL Queries

```sql
-- FindByNamePrefix: Fetch candidates matching prefix
SELECT id, name
FROM food_items
WHERE LOWER(name) LIKE $1 || '%'
ORDER BY LENGTH(name) ASC
LIMIT $2;

-- FindTopByLength: Fallback fetch by length
SELECT id, name
FROM food_items
ORDER BY LENGTH(name) ASC
LIMIT $1;
```

**Required Index:**
```sql
CREATE INDEX idx_food_items_name_lower ON food_items (LOWER(name));
CREATE INDEX idx_food_items_name_length ON food_items (LENGTH(name));
```

#### 4.5 Redis Cache Schema

| Key Pattern | Value Type | TTL | Description |
|:------------|:-----------|:----|:------------|
| `autocomplete:<md5_hash>` | JSON | 300s | Serialized AutocompleteResponse |

**Example:**
```
Key: autocomplete:a1b2c3d4e5f6...
Value: {"suggestions":[{"itemId":"uuid","name":"Apple","matchType":1,"score":0.5,"levenshteinDist":0}],"queryTime":45}
TTL: 300
```
