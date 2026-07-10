## FILE: DESIGN-002.md
**Traceability:** ARCH-002

**Static aspects covered:** SearchController, AutocompleteRanker, QueryParser, PaginationHandler, FilterProcessor, CulinaryRoleWeighter.

### 0. Static Aspect Responsibilities
- `SearchController`: owns Fiber endpoint handlers, request validation, service orchestration, selected-item FoodObject hydration, and response envelopes.
- `AutocompleteRanker`: owns exact-match, Levenshtein, and string-length ranking.
- `QueryParser`: owns normalization and strategy selection for Catalog Search, Substitution Search, and Daily Diet Alternative Search.
- `PaginationHandler`: owns page bounds, page size of 10, offsets, and response metadata.
- `FilterProcessor`: owns include/exclude Search filter validation, Exclusion Rule conflict detection, and repository query translation.
- `CulinaryRoleWeighter`: owns single-input Substitution Search score boosts for shared Culinary Roles.

### 1. Data Structures & Types
- `type SearchMode = "catalog" | "substitution" | "daily_diet" | "daily_diet_alternative"`
- `interface SearchRequest { query: string; mode: SearchMode; filters: SearchFilter[]; page: number; substitutionInputs?: SubstitutionInput[]; dailyDietId?: string }`
- `interface SubstitutionInput { foodObjectId: UUID; quantity: decimal; unit: string }`
- `interface SearchResponse { items: FoodObject[]; totalCount: number; page: number; similarityScores: number[]; warnings: string[]; rejection?: SearchRejection }`
- `interface SearchRejection { code: string; message: string; field?: string }`
- `interface ParsedQuery { normalizedText: string; tokens: string[]; strategy: "catalog" | "substitution" | "daily_diet" | "daily_diet_alternative"; limit: number; offset: number }`
- `interface SearchFilter { filterId: string; kind: "food_category" | "culinary_role" | "food_object_type" | "allergen" | "dietary_preset"; include: boolean }`
- `interface RankedAutocomplete { itemId: string; label: string; exactMatch: boolean; levenshteinDistance: number; length: number; rank: number }`
- `interface SearchCandidate { item: FoodObject; textScore: number; similarityScore?: number; culinaryRoleMatchCount: number; finalScore: number }`
- `interface FoodObjectEnvelope { status: "ok"; requestId: string; data: FoodObject }`

### 2. Logic & Algorithms (Step-by-Step)
1. Validate `page >= 1`, `query` length, filter shape, Substitution Input quantities, and allowed `mode`.
2. Normalize query text by trimming whitespace, lowercasing, and collapsing internal spaces.
3. If Substitution Inputs are present, choose Substitution Search regardless of input count; adding inputs refines one search operation rather than switching modes.
4. Reject contradictory filters and Exclusion Rule conflicts with user-facing feedback before repository or similarity work.
5. Check ARCH-011 Redis cache with a deterministic key made from normalized query, mode, filters, page, and Substitution Inputs.
6. For autocomplete, fetch candidate names from ARCH-005 and rank by exact match first, Levenshtein distance second, and shorter string length third.
7. For Catalog Search, ask ARCH-005 for filtered Food Objects with a maximum page size of 10.
8. For Substitution Search, combine input Food Quantities into one Macro Profile and call ARCH-003 with that source profile and candidate Macro Profiles.
9. Apply Culinary Role weighting using `finalScore = similarityScore * (1 + 0.2 * tagMatchCount)` only when exactly one Substitution Input exists.
10. Sort by `finalScore` descending for Substitution Search and by text rank for Catalog Search.
11. Store successful responses in ARCH-011 with a TTL selected by cache policy and return the response.

### 3. State Management & Error Handling
- `invalid_request`: malformed mode, page, filter, or Substitution Input; return 400 with field-level details.
- `rejected_search`: validly shaped but contradictory Search constraints; return 422 with a user-facing `SearchRejection`.
- `cache_hit`: return cached `SearchResponse` and skip repository calls.
- `cache_miss`: continue to repository and similarity processing.
- `empty_results`: return 200 with empty `items`, `totalCount = 0`, and no exception.
- `similarity_unavailable`: return service error for Substitution Search because substitutes depend on ARCH-003.
- `repository_unavailable`: return service error because basic search depends on ARCH-005.
- `ranking_timeout`: stop expensive ranking and return best available page if repository data exists.

### 4. Component Interfaces
- `func (c *SearchController) Search(ctx *fiber.Ctx) error`
- `func (c *SearchController) Autocomplete(ctx *fiber.Ctx) error`
- `func (c *SearchController) GetFoodObject(ctx *fiber.Ctx) error`
- `func ParseSearchRequest(ctx *fiber.Ctx) (SearchRequest, error)`
- `func BuildParsedQuery(req SearchRequest) ParsedQuery`
- `func RankAutocomplete(query string, candidates []FoodItem) []RankedAutocomplete`
- `func ApplyFilters(query ParsedQuery, filters []SearchFilter) RepositoryQuery`
- `func Paginate(page int, pageSize int) (limit int, offset int)`
- `func ApplyCulinaryRoleWeight(candidates []SearchCandidate, sourceCulinaryRoles []string) []SearchCandidate`
- `func BuildSearchCacheKey(req SearchRequest) string`
