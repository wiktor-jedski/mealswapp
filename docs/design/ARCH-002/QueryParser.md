## FILE: QueryParser.md
**Traceability:** ARCH-002

### 1. Data Structures & Types

```go
// SearchMode indicates the type of search to execute
type SearchMode int

const (
    SearchModeText       SearchMode = 1 // Standard text-based search
    SearchModeSimilarity SearchMode = 2 // Cosine similarity-based search
    SearchModeImplicit   SearchMode = 3 // Auto-triggered similarity search (empty query + ingredients)
)

// TagFilterType indicates whether tags should be included or excluded
type TagFilterType int

const (
    TagFilterWhitelist TagFilterType = 1 // Only include items with these tags
    TagFilterBlacklist TagFilterType = 2 // Exclude items with these tags
)

// TagFilter represents a filter to apply based on tags
type TagFilter struct {
    Type TagFilterType // Whitelist or blacklist
    Tags []string      // Tag identifiers to filter
}

// SearchRequest represents the raw input from the client
type SearchRequest struct {
    Query       string      // Raw search query string
    Mode        SearchMode  // Explicit search mode (may be overridden)
    Filters     []TagFilter // Tag-based filters to apply
    Page        int         // Page number for pagination (1-indexed)
    Ingredients []string    // List of ingredient IDs for similarity search
}

// ParsedQuery represents the validated and normalized query ready for execution
type ParsedQuery struct {
    NormalizedQuery   string            // Sanitized, trimmed, lowercased query
    EffectiveMode     SearchMode        // Final search mode after implicit detection
    WhitelistTags     []string          // Consolidated whitelist tags
    BlacklistTags     []string          // Consolidated blacklist tags
    Page              int               // Validated page number (min: 1)
    PageSize          int               // Results per page (fixed: 10)
    Ingredients       []string          // Validated ingredient IDs
    IsImplicitTrigger bool              // True if implicit similarity search was triggered
    SearchStrategy    SearchStrategy    // Strategy to use for search execution
}

// SearchStrategy indicates which search path to take
type SearchStrategy int

const (
    StrategyTextSearch       SearchStrategy = 1 // Use PostgreSQL text search
    StrategySimilaritySearch SearchStrategy = 2 // Use Similarity Engine (ARCH-003)
    StrategyEmptyResult      SearchStrategy = 3 // Return empty results immediately
)

// ParseResult contains the parsed query and any validation warnings
type ParseResult struct {
    Query    ParsedQuery   // The parsed and validated query
    Warnings []ParseWarning // Non-fatal issues encountered during parsing
}

// ParseWarning represents a non-fatal validation issue
type ParseWarning struct {
    Code    WarningCode // Warning identifier
    Message string      // Human-readable warning message
    Field   string      // Field that caused the warning
}

// WarningCode identifies specific warning types
type WarningCode int

const (
    WarningQueryTruncated    WarningCode = 1 // Query exceeded max length
    WarningPageOutOfRange    WarningCode = 2 // Page number was adjusted
    WarningInvalidIngredient WarningCode = 3 // Ingredient ID was invalid/removed
    WarningDuplicateTag      WarningCode = 4 // Duplicate tag was removed
    WarningConflictingFilter WarningCode = 5 // Tag in both whitelist and blacklist
)

// ParserConfig holds configuration for the query parser
type ParserConfig struct {
    MaxQueryLength        int // Maximum allowed query length (default: 200)
    MinQueryLength        int // Minimum query length for text search (default: 1)
    MaxIngredientsCount   int // Maximum ingredients for similarity search (default: 50)
    MinIngredientsForImpl int // Min ingredients to trigger implicit search (default: 2)
    MaxTagsPerFilter      int // Maximum tags per filter type (default: 20)
    PageSize              int // Fixed page size (default: 10)
    MaxPageNumber         int // Maximum allowed page number (default: 100)
}

// ValidationError represents a fatal parsing error
type ValidationError struct {
    Code    ErrorCode // Error identifier
    Message string    // Human-readable error message
    Field   string    // Field that caused the error
}

// ErrorCode identifies specific error types
type ErrorCode int

const (
    ErrorInvalidMode       ErrorCode = 1 // Unknown search mode
    ErrorMalformedFilter   ErrorCode = 2 // Filter structure is invalid
    ErrorNoSearchCriteria  ErrorCode = 3 // Neither query nor ingredients provided
)
```

### 2. Logic & Algorithms (Step-by-Step)

#### 2.1 Main Parsing Flow

```
FUNCTION Parse(request SearchRequest, config ParserConfig) -> (ParseResult, error):
    1. INITIALIZE
       - Create empty ParsedQuery
       - Create empty warnings slice
       - Set PageSize = config.PageSize (10)

    2. VALIDATE SEARCH MODE
       - IF request.Mode < 1 OR request.Mode > 3:
           RETURN error(ErrorInvalidMode, "Invalid search mode")
       - Set ParsedQuery.EffectiveMode = request.Mode

    3. NORMALIZE QUERY
       - result, queryWarnings = NormalizeQuery(request.Query, config)
       - APPEND queryWarnings to warnings
       - Set ParsedQuery.NormalizedQuery = result

    4. VALIDATE AND NORMALIZE PAGE
       - IF request.Page < 1:
           Set ParsedQuery.Page = 1
           ADD warning(WarningPageOutOfRange, "Page adjusted to 1")
       - ELSE IF request.Page > config.MaxPageNumber:
           Set ParsedQuery.Page = config.MaxPageNumber
           ADD warning(WarningPageOutOfRange, "Page adjusted to max")
       - ELSE:
           Set ParsedQuery.Page = request.Page

    5. PROCESS FILTERS
       - filterResult, filterWarnings, filterError = ProcessFilters(request.Filters, config)
       - IF filterError != nil:
           RETURN error(filterError)
       - APPEND filterWarnings to warnings
       - Set ParsedQuery.WhitelistTags = filterResult.WhitelistTags
       - Set ParsedQuery.BlacklistTags = filterResult.BlacklistTags

    6. VALIDATE INGREDIENTS
       - ingredientResult, ingredientWarnings = ValidateIngredients(request.Ingredients, config)
       - APPEND ingredientWarnings to warnings
       - Set ParsedQuery.Ingredients = ingredientResult

    7. DETECT IMPLICIT TRIGGER
       - IF ParsedQuery.NormalizedQuery == "" AND len(ParsedQuery.Ingredients) >= config.MinIngredientsForImpl:
           Set ParsedQuery.IsImplicitTrigger = true
           Set ParsedQuery.EffectiveMode = SearchModeImplicit

    8. DETERMINE SEARCH STRATEGY
       - ParsedQuery.SearchStrategy = DetermineStrategy(ParsedQuery)

    9. FINAL VALIDATION
       - IF ParsedQuery.SearchStrategy == StrategyEmptyResult AND NOT intentional:
           IF ParsedQuery.NormalizedQuery == "" AND len(ParsedQuery.Ingredients) == 0:
               // Valid case: user cleared search
           ELSE IF no search criteria after validation:
               RETURN error(ErrorNoSearchCriteria, "No valid search criteria")

    10. RETURN ParseResult{Query: ParsedQuery, Warnings: warnings}
```

#### 2.2 Query Normalization Algorithm

```
FUNCTION NormalizeQuery(query string, config ParserConfig) -> (string, []ParseWarning):
    warnings = []

    1. HANDLE NIL/EMPTY
       - IF query == nil OR query == "":
           RETURN ("", warnings)

    2. TRIM WHITESPACE
       - normalized = strings.TrimSpace(query)

    3. CHECK LENGTH
       - IF len(normalized) > config.MaxQueryLength:
           normalized = normalized[0:config.MaxQueryLength]
           ADD warning(WarningQueryTruncated, "Query truncated to max length")

    4. SANITIZE
       - Remove control characters (ASCII 0-31, 127)
       - Collapse multiple spaces to single space
       - normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

    5. CONVERT TO LOWERCASE
       - normalized = strings.ToLower(normalized)

    6. ESCAPE SQL WILDCARDS
       - Replace '%' with '\%'
       - Replace '_' with '\_'

    7. RETURN (normalized, warnings)
```

#### 2.3 Filter Processing Algorithm

```
FUNCTION ProcessFilters(filters []TagFilter, config ParserConfig) -> (FilterResult, []ParseWarning, error):
    warnings = []
    whitelistTags = []
    blacklistTags = []
    seenWhitelist = map[string]bool{}
    seenBlacklist = map[string]bool{}

    1. ITERATE THROUGH FILTERS
       - FOR each filter in filters:
           - IF filter.Type NOT IN (TagFilterWhitelist, TagFilterBlacklist):
               RETURN error(ErrorMalformedFilter, "Invalid filter type")

           - FOR each tag in filter.Tags:
               - normalizedTag = strings.TrimSpace(strings.ToLower(tag))
               - IF normalizedTag == "":
                   CONTINUE // Skip empty tags

               - IF filter.Type == TagFilterWhitelist:
                   - IF seenWhitelist[normalizedTag]:
                       ADD warning(WarningDuplicateTag, "Duplicate whitelist tag")
                       CONTINUE
                   - IF seenBlacklist[normalizedTag]:
                       ADD warning(WarningConflictingFilter, "Tag in both whitelist and blacklist, using whitelist")
                       DELETE normalizedTag from blacklistTags
                       DELETE from seenBlacklist
                   - IF len(whitelistTags) < config.MaxTagsPerFilter:
                       APPEND normalizedTag to whitelistTags
                       seenWhitelist[normalizedTag] = true

               - IF filter.Type == TagFilterBlacklist:
                   - IF seenBlacklist[normalizedTag]:
                       ADD warning(WarningDuplicateTag, "Duplicate blacklist tag")
                       CONTINUE
                   - IF seenWhitelist[normalizedTag]:
                       ADD warning(WarningConflictingFilter, "Tag in both whitelist and blacklist, whitelist takes precedence")
                       CONTINUE // Whitelist wins
                   - IF len(blacklistTags) < config.MaxTagsPerFilter:
                       APPEND normalizedTag to blacklistTags
                       seenBlacklist[normalizedTag] = true

    2. RETURN FilterResult{WhitelistTags: whitelistTags, BlacklistTags: blacklistTags}, warnings, nil
```

#### 2.4 Ingredient Validation Algorithm

```
FUNCTION ValidateIngredients(ingredients []string, config ParserConfig) -> ([]string, []ParseWarning):
    warnings = []
    validIngredients = []
    seen = map[string]bool{}

    1. HANDLE NIL/EMPTY
       - IF ingredients == nil OR len(ingredients) == 0:
           RETURN ([], warnings)

    2. VALIDATE EACH INGREDIENT
       - FOR each ingredientID in ingredients:
           - trimmedID = strings.TrimSpace(ingredientID)

           - // Validate UUID format (standard 8-4-4-4-12 format)
           - IF NOT isValidUUID(trimmedID):
               ADD warning(WarningInvalidIngredient, "Invalid ingredient ID format")
               CONTINUE

           - // Check for duplicates
           - IF seen[trimmedID]:
               CONTINUE // Silently skip duplicates

           - // Check max count
           - IF len(validIngredients) >= config.MaxIngredientsCount:
               ADD warning(WarningInvalidIngredient, "Maximum ingredients exceeded, some ignored")
               BREAK

           - APPEND trimmedID to validIngredients
           - seen[trimmedID] = true

    3. RETURN (validIngredients, warnings)
```

#### 2.5 Strategy Determination Algorithm

```
FUNCTION DetermineStrategy(query ParsedQuery) -> SearchStrategy:
    1. CHECK FOR IMPLICIT SIMILARITY SEARCH
       - IF query.IsImplicitTrigger:
           RETURN StrategySimilaritySearch

    2. CHECK EXPLICIT SIMILARITY MODE
       - IF query.EffectiveMode == SearchModeSimilarity:
           - IF len(query.Ingredients) > 0:
               RETURN StrategySimilaritySearch
           - ELSE:
               // Fallback to text search if no ingredients for similarity
               RETURN StrategyTextSearch

    3. CHECK FOR TEXT SEARCH
       - IF query.NormalizedQuery != "" AND len(query.NormalizedQuery) >= MinQueryLength:
           RETURN StrategyTextSearch

    4. CHECK FOR EMPTY VALID STATE
       - IF query.NormalizedQuery == "" AND len(query.Ingredients) == 0:
           // User has no search criteria - return empty
           RETURN StrategyEmptyResult

    5. DEFAULT
       - // Has ingredients but not enough for implicit, no query
       - RETURN StrategyEmptyResult
```

#### 2.6 UUID Validation

```
FUNCTION isValidUUID(s string) -> bool:
    // UUID format: 8-4-4-4-12 hexadecimal characters
    // Example: 550e8400-e29b-41d4-a716-446655440000

    1. CHECK LENGTH
       - IF len(s) != 36:
           RETURN false

    2. CHECK HYPHENS
       - IF s[8] != '-' OR s[13] != '-' OR s[18] != '-' OR s[23] != '-':
           RETURN false

    3. CHECK HEX CHARACTERS
       - positions = [0-7, 9-12, 14-17, 19-22, 24-35]
       - FOR each position in positions:
           - IF NOT isHexDigit(s[position]):
               RETURN false

    4. RETURN true
```

### 3. State Management & Error Handling

#### 3.1 Error States

| Error State | Cause | Detection | Response | HTTP Status |
|:------------|:------|:----------|:---------|:------------|
| Invalid Search Mode | Mode value not in 1-3 range | request.Mode < 1 OR > 3 | Return ValidationError | 400 Bad Request |
| Malformed Filter | Filter type invalid or structure corrupt | Filter validation fails | Return ValidationError | 400 Bad Request |
| No Search Criteria | Empty query AND no ingredients AND mode requires input | After all validation | Return ValidationError | 400 Bad Request |

#### 3.2 Warning States (Non-Fatal)

| Warning State | Cause | Detection | Behavior | User Impact |
|:--------------|:------|:----------|:---------|:------------|
| Query Truncated | Query exceeds 200 characters | len(query) > MaxQueryLength | Truncate and continue | Results based on truncated query |
| Page Out of Range | Page < 1 or > 100 | Page validation | Adjust to valid range | Results from adjusted page |
| Invalid Ingredient | Ingredient ID not valid UUID | UUID validation | Skip ingredient | Fewer items in similarity search |
| Duplicate Tag | Same tag appears multiple times | Set membership check | Remove duplicate | No user impact |
| Conflicting Filter | Tag in both whitelist and blacklist | Cross-check filters | Whitelist takes precedence | Tag included in results |

#### 3.3 State Transitions

```
                    ┌─────────────┐
                    │   RECEIVE   │
                    │   REQUEST   │
                    └──────┬──────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  VALIDATE   │
                    │    MODE     │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │ Invalid    │ Valid      │
              ▼            ▼            │
    ┌─────────────┐ ┌─────────────┐    │
    │ RETURN_ERR  │ │ NORMALIZE   │    │
    │ (400)       │ │   QUERY     │    │
    └─────────────┘ └──────┬──────┘    │
                           │            │
                           ▼            │
                    ┌─────────────┐    │
                    │  VALIDATE   │    │
                    │    PAGE     │    │
                    └──────┬──────┘    │
                           │ (may add warning)
                           ▼            │
                    ┌─────────────┐    │
                    │  PROCESS    │    │
                    │  FILTERS    │    │
                    └──────┬──────┘    │
                           │            │
              ┌────────────┼────────────┤
              │ Error      │ Success    │
              ▼            ▼            │
    ┌─────────────┐ ┌─────────────┐    │
    │ RETURN_ERR  │ │ VALIDATE    │    │
    │ (400)       │ │ INGREDIENTS │    │
    └─────────────┘ └──────┬──────┘    │
                           │            │
                           ▼            │
                    ┌─────────────┐    │
                    │   DETECT    │    │
                    │  IMPLICIT   │    │
                    └──────┬──────┘    │
                           │            │
                           ▼            │
                    ┌─────────────┐    │
                    │ DETERMINE   │    │
                    │  STRATEGY   │    │
                    └──────┬──────┘    │
                           │            │
                           ▼            │
                    ┌─────────────┐    │
                    │   RETURN    │◄───┘
                    │ ParseResult │
                    └─────────────┘
```

#### 3.4 Implicit Trigger Detection Logic

```
┌────────────────────────────────────────────────────────┐
│                    IMPLICIT TRIGGER                    │
├────────────────────────────────────────────────────────┤
│                                                        │
│   Conditions for Implicit Similarity Search:           │
│   ┌────────────────────────────────────────────────┐  │
│   │ 1. NormalizedQuery == ""           (empty)     │  │
│   │ 2. len(Ingredients) >= 2           (2+ items)  │  │
│   │ 3. Mode != SearchModeSimilarity    (not explicit)│ │
│   └────────────────────────────────────────────────┘  │
│                                                        │
│   When triggered:                                      │
│   - EffectiveMode = SearchModeImplicit                │
│   - IsImplicitTrigger = true                          │
│   - SearchStrategy = StrategySimilaritySearch         │
│                                                        │
│   User Intent: User has added ingredients to their    │
│   list and cleared/left empty the search bar,         │
│   indicating they want similar recipe suggestions.    │
│                                                        │
└────────────────────────────────────────────────────────┘
```

#### 3.5 Performance Constraints

| Constraint | Target | Enforcement |
|:-----------|:-------|:------------|
| Parse Time | < 1ms | No I/O operations; pure computation |
| Memory | < 10KB per request | Bounded input sizes via config limits |
| Allocations | Minimal | Reuse slices where possible |

### 4. Component Interfaces

#### 4.1 Public Interface

```go
// QueryParser handles validation and normalization of search requests
type QueryParser interface {
    // Parse validates and normalizes a SearchRequest into a ParsedQuery.
    // Returns ParseResult with the parsed query and any warnings.
    // Returns error only for fatal validation failures (HTTP 400 cases).
    Parse(ctx context.Context, request SearchRequest) (ParseResult, error)
}
```

#### 4.2 Internal Functions

```go
// NewQueryParser creates a new parser instance with the given configuration.
// Parameters:
//   - config: Parser configuration (use DefaultParserConfig if nil)
// Returns:
//   - QueryParser implementation
func NewQueryParser(config ParserConfig) QueryParser

// normalizeQuery sanitizes and normalizes a raw query string.
// Parameters:
//   - query: Raw query string from request
//   - config: Parser configuration
// Returns:
//   - normalized: Sanitized, lowercased query string
//   - warnings: Any non-fatal issues encountered
func normalizeQuery(query string, config ParserConfig) (normalized string, warnings []ParseWarning)

// processFilters validates and consolidates tag filters.
// Parameters:
//   - filters: Slice of TagFilter from request
//   - config: Parser configuration
// Returns:
//   - result: Consolidated whitelist and blacklist tags
//   - warnings: Any non-fatal issues encountered
//   - err: Fatal validation error if filter is malformed
func processFilters(filters []TagFilter, config ParserConfig) (result FilterResult, warnings []ParseWarning, err error)

// validateIngredients validates and deduplicates ingredient IDs.
// Parameters:
//   - ingredients: Slice of ingredient ID strings
//   - config: Parser configuration
// Returns:
//   - valid: Validated ingredient IDs
//   - warnings: Any non-fatal issues encountered
func validateIngredients(ingredients []string, config ParserConfig) (valid []string, warnings []ParseWarning)

// determineStrategy selects the appropriate search strategy based on parsed query.
// Parameters:
//   - query: The parsed query after all validation
// Returns:
//   - strategy: The SearchStrategy to use
func determineStrategy(query ParsedQuery) SearchStrategy

// isValidUUID checks if a string is a valid UUID v4 format.
// Parameters:
//   - s: String to validate
// Returns:
//   - true if valid UUID format, false otherwise
func isValidUUID(s string) bool

// escapeSQLWildcards escapes SQL LIKE wildcards in a string.
// Parameters:
//   - s: String to escape
// Returns:
//   - escaped string with % and _ prefixed with backslash
func escapeSQLWildcards(s string) string
```

#### 4.3 Default Configuration Values

```go
var DefaultParserConfig = ParserConfig{
    MaxQueryLength:        200,
    MinQueryLength:        1,
    MaxIngredientsCount:   50,
    MinIngredientsForImpl: 2,
    MaxTagsPerFilter:      20,
    PageSize:              10,
    MaxPageNumber:         100,
}
```

#### 4.4 Error and Warning Constructors

```go
// NewValidationError creates a new validation error.
// Parameters:
//   - code: ErrorCode identifying the error type
//   - message: Human-readable error message
//   - field: The request field that caused the error
// Returns:
//   - ValidationError
func NewValidationError(code ErrorCode, message string, field string) *ValidationError

// Error implements the error interface for ValidationError.
func (e *ValidationError) Error() string {
    return fmt.Sprintf("[%d] %s (field: %s)", e.Code, e.Message, e.Field)
}

// NewParseWarning creates a new parse warning.
// Parameters:
//   - code: WarningCode identifying the warning type
//   - message: Human-readable warning message
//   - field: The request field that caused the warning
// Returns:
//   - ParseWarning
func NewParseWarning(code WarningCode, message string, field string) ParseWarning
```

#### 4.5 Integration with SearchController

```go
// Example usage in SearchController (ARCH-002)

func (c *SearchController) Search(ctx *fiber.Ctx) error {
    // 1. Bind request
    var request SearchRequest
    if err := ctx.BodyParser(&request); err != nil {
        return ctx.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
    }

    // 2. Parse and validate
    parseResult, err := c.parser.Parse(ctx.Context(), request)
    if err != nil {
        var validationErr *ValidationError
        if errors.As(err, &validationErr) {
            return ctx.Status(400).JSON(fiber.Map{
                "error": validationErr.Message,
                "code":  validationErr.Code,
                "field": validationErr.Field,
            })
        }
        return ctx.Status(500).JSON(fiber.Map{"error": "Internal server error"})
    }

    // 3. Route to appropriate handler based on strategy
    switch parseResult.Query.SearchStrategy {
    case StrategyTextSearch:
        return c.handleTextSearch(ctx, parseResult.Query)
    case StrategySimilaritySearch:
        return c.handleSimilaritySearch(ctx, parseResult.Query)
    case StrategyEmptyResult:
        return c.handleEmptyResult(ctx, parseResult.Query)
    }

    return ctx.Status(500).JSON(fiber.Map{"error": "Unknown search strategy"})
}
```

#### 4.6 JSON Serialization

```go
// ParsedQuery JSON representation for logging/debugging
type ParsedQueryJSON struct {
    NormalizedQuery   string   `json:"normalized_query"`
    EffectiveMode     int      `json:"effective_mode"`
    WhitelistTags     []string `json:"whitelist_tags"`
    BlacklistTags     []string `json:"blacklist_tags"`
    Page              int      `json:"page"`
    PageSize          int      `json:"page_size"`
    IngredientCount   int      `json:"ingredient_count"`
    IsImplicitTrigger bool     `json:"is_implicit_trigger"`
    SearchStrategy    int      `json:"search_strategy"`
}

// ToJSON converts ParsedQuery to its JSON representation.
// Note: Ingredients are not included to avoid logging user data.
func (q ParsedQuery) ToJSON() ParsedQueryJSON {
    return ParsedQueryJSON{
        NormalizedQuery:   q.NormalizedQuery,
        EffectiveMode:     int(q.EffectiveMode),
        WhitelistTags:     q.WhitelistTags,
        BlacklistTags:     q.BlacklistTags,
        Page:              q.Page,
        PageSize:          q.PageSize,
        IngredientCount:   len(q.Ingredients),
        IsImplicitTrigger: q.IsImplicitTrigger,
        SearchStrategy:    int(q.SearchStrategy),
    }
}
```
