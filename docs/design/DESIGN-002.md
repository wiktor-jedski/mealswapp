## FILE: DESIGN-002.md
**Traceability:** ARCH-002

**Static aspects covered:** SearchController, AutocompleteRanker, QueryParser, PaginationHandler, FilterProcessor, FunctionalityTagWeighter.

### 0. Static Aspect Responsibilities
- `SearchController`: owns Fiber endpoint handlers, request validation, service orchestration, and response envelopes.
- `AutocompleteRanker`: owns exact-match, Levenshtein, and string-length ranking.
- `QueryParser`: owns normalization and strategy selection for text, replacement, and implicit similarity searches.
- `PaginationHandler`: owns page bounds, page size of 10, offsets, and response metadata.
- `FilterProcessor`: owns include/exclude tag validation and repository query translation.
- `FunctionalityTagWeighter`: owns replacement-search score boosts for shared functionality tags.

### 1. Data Structures & Types
- `type SearchMode = "single" | "replacement" | "diet"`
- `interface SearchRequest { query: string; mode: SearchMode; filters: TagFilter[]; page: number; ingredients?: string[]; sourceItemId?: string }`
- `interface SearchResponse { items: FoodItem[]; totalCount: number; page: number; similarityScores: number[]; warnings: string[] }`
- `interface ParsedQuery { normalizedText: string; tokens: string[]; strategy: "text" | "similarity" | "implicit_similarity"; limit: number; offset: number }`
- `interface TagFilter { tagId: string; kind: "category" | "functionality"; include: boolean }`
- `interface RankedAutocomplete { itemId: string; label: string; exactMatch: boolean; levenshteinDistance: number; length: number; rank: number }`
- `interface SearchCandidate { item: FoodItem; textScore: number; similarityScore?: number; tagMatchCount: number; finalScore: number }`

### 2. Logic & Algorithms (Step-by-Step)
1. Validate `page >= 1`, `query` length, filter shape, and allowed `mode`.
2. Normalize query text by trimming whitespace, lowercasing, and collapsing internal spaces.
3. If `query` is empty and at least two ingredient IDs are supplied, choose `implicit_similarity`; otherwise choose text or replacement search from mode.
4. Check ARCH-011 Redis cache with a deterministic key made from normalized query, mode, filters, page, and ingredients.
5. For autocomplete, fetch candidate names from ARCH-005 and rank by exact match first, Levenshtein distance second, and shorter string length third.
6. For text search, ask ARCH-005 for filtered food items with a maximum page size of 10.
7. For replacement or implicit similarity, call ARCH-003 with the source macro vector and candidate macro vectors.
8. Apply functionality tag weighting using `finalScore = similarityScore * (1 + 0.2 * tagMatchCount)` when a source item exists.
9. Sort by `finalScore` descending for similarity requests and by text rank for pure text requests.
10. Store successful responses in ARCH-011 with a TTL selected by cache policy and return the response.

### 3. State Management & Error Handling
- `invalid_request`: malformed mode, page, filter, or ingredient input; return 400 with field-level details.
- `cache_hit`: return cached `SearchResponse` and skip repository calls.
- `cache_miss`: continue to repository and similarity processing.
- `empty_results`: return 200 with empty `items`, `totalCount = 0`, and no exception.
- `similarity_unavailable`: degrade to text results with a warning when ARCH-003 times out or fails.
- `repository_unavailable`: return service error because basic search depends on ARCH-005.
- `ranking_timeout`: stop expensive ranking and return best available page if repository data exists.

### 4. Component Interfaces
- `func (c *SearchController) Search(ctx *fiber.Ctx) error`
- `func (c *SearchController) Autocomplete(ctx *fiber.Ctx) error`
- `func ParseSearchRequest(ctx *fiber.Ctx) (SearchRequest, error)`
- `func BuildParsedQuery(req SearchRequest) ParsedQuery`
- `func RankAutocomplete(query string, candidates []FoodItem) []RankedAutocomplete`
- `func ApplyFilters(query ParsedQuery, filters []TagFilter) RepositoryQuery`
- `func Paginate(page int, pageSize int) (limit int, offset int)`
- `func ApplyFunctionalityWeight(candidates []SearchCandidate, sourceTags []string) []SearchCandidate`
- `func BuildSearchCacheKey(req SearchRequest) string`
