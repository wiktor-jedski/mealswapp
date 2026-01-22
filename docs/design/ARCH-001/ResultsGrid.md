# Detailed Design: ResultsGrid

**Traceability:** [ARCH-001]

---

## 1. Data Structures & Types

### 1.1 Food Item Types

```typescript
interface FoodItem {
  id: string;
  name: string;
  imageUrl: string | null;
  physicalState: 'solid' | 'liquid';
  prepTime: number;                    // minutes
  averageUnitWeight: number;           // grams
  macros: MacroNutrients;
  micros?: MicroNutrients;
  categoryTags: Tag[];
  functionalityTags: Tag[];
}

interface MacroNutrients {
  protein: number;                     // per 100g/ml
  carbs: number;
  fat: number;
  calories: number;                    // derived: (P*4 + C*4 + F*9)
}

interface MicroNutrients {
  sodium?: number;
  fiber?: number;
  sugar?: number;
  [key: string]: number | undefined;
}

interface Tag {
  id: string;
  name: string;
  type: 'category' | 'functionality';
}
```

### 1.2 Similarity Result Types

```typescript
type SimilarityTier = 'excellent' | 'good' | 'fair' | 'poor';

interface SimilarityIndicator {
  tier: SimilarityTier;
  score: number;                       // 0.0 - 1.0 (displayed as percentage)
  colorHex: string;
  imageUrl: string;                    // Server-hosted indicator icon
}

interface SimilarityResult {
  item: FoodItem;
  similarity: SimilarityIndicator;
  matchingQuantity: number;            // Quantity to match original macros
  matchType: 'calorie' | 'protein';    // What the quantity matches
}

const SIMILARITY_THRESHOLDS: Record<SimilarityTier, { min: number; max: number; color: string; icon: string }> = {
  excellent: {
    min: 0.85,
    max: 1.0,
    color: '#22C55E',                  // Green
    icon: '/assets/indicators/star.png'
  },
  good: {
    min: 0.70,
    max: 0.84,
    color: '#84CC16',                  // Light Green
    icon: '/assets/indicators/sparkle.png'
  },
  fair: {
    min: 0.55,
    max: 0.69,
    color: '#EAB308',                  // Yellow
    icon: '/assets/indicators/thumbs-up.png'
  },
  poor: {
    min: 0.40,
    max: 0.54,
    color: '#EF4444',                  // Red
    icon: '/assets/indicators/thumbs-down.png'
  }
};

const MIN_SIMILARITY_THRESHOLD = 0.40;
```

### 1.3 Pagination Types

```typescript
interface PaginationState {
  currentPage: number;
  pageSize: number;
  totalCount: number;
  totalPages: number;
  hasNextPage: boolean;
  hasPreviousPage: boolean;
}

const DEFAULT_PAGE_SIZE = 10;
const MAX_PAGE_SIZE = 50;
const PAGE_SIZE_OPTIONS = [10, 25, 50];
```

### 1.4 Sorting Types

```typescript
type SortField = 'similarity' | 'name' | 'calories' | 'protein' | 'carbs' | 'fat';
type SortDirection = 'asc' | 'desc';

interface SortState {
  field: SortField;
  direction: SortDirection;
}

const DEFAULT_SORT: SortState = {
  field: 'similarity',
  direction: 'desc'
};
```

### 1.5 View Mode Types

```typescript
type ViewMode = 'grid' | 'list';

interface ViewConfig {
  mode: ViewMode;
  columns: number;                     // For grid mode (responsive)
  showMacroDetails: boolean;           // Expand macro breakdown
  showMicronutrients: boolean;         // Show additional nutrients
}

const DEFAULT_VIEW_CONFIG: ViewConfig = {
  mode: 'grid',
  columns: 3,
  showMacroDetails: true,
  showMicronutrients: false
};

const RESPONSIVE_COLUMNS = {
  mobile: 1,                           // < 640px
  tablet: 2,                           // 640-1023px
  desktop: 3,                          // 1024-1439px
  wide: 4                              // >= 1440px
};
```

### 1.6 Loading & Empty States

```typescript
type ResultsStatus = 'idle' | 'loading' | 'success' | 'empty' | 'error' | 'offline';

interface LoadingState {
  isLoading: boolean;
  isLoadingMore: boolean;              // For infinite scroll/pagination
  loadingMessage: string;
}

interface EmptyState {
  type: 'no_results' | 'no_matches' | 'no_search';
  title: string;
  message: string;
  suggestions: string[];
  showClearFilters: boolean;
}

const EMPTY_STATES: Record<EmptyState['type'], Omit<EmptyState, 'type' | 'showClearFilters'>> = {
  no_results: {
    title: 'No items found',
    message: 'Try adjusting your search terms or filters.',
    suggestions: ['Use broader search terms', 'Remove some filters', 'Check spelling']
  },
  no_matches: {
    title: 'No similar items found',
    message: 'No items meet the minimum similarity threshold (40%).',
    suggestions: ['Try a different item', 'Adjust macro toggles', 'Search for related foods']
  },
  no_search: {
    title: 'Start searching',
    message: 'Enter a food name or select ingredients to find alternatives.',
    suggestions: []
  }
};
```

### 1.7 Selection Types

```typescript
interface SelectionState {
  selectedItemId: string | null;       // Currently selected for detail view
  comparisonItemIds: string[];         // Items selected for side-by-side comparison
  maxComparisonItems: number;
}

const MAX_COMPARISON_ITEMS = 3;
```

### 1.8 Result Card Actions

```typescript
type CardAction = 'select' | 'save' | 'compare' | 'replace' | 'view_details';

interface CardActionConfig {
  action: CardAction;
  label: string;
  icon: string;
  requiresAuth: boolean;
  requiresPaidTier: boolean;
}

const CARD_ACTIONS: CardActionConfig[] = [
  {
    action: 'select',
    label: 'Select as replacement',
    icon: 'check-circle',
    requiresAuth: false,
    requiresPaidTier: false
  },
  {
    action: 'save',
    label: 'Save item',
    icon: 'bookmark',
    requiresAuth: true,
    requiresPaidTier: false
  },
  {
    action: 'compare',
    label: 'Add to comparison',
    icon: 'columns',
    requiresAuth: false,
    requiresPaidTier: true
  },
  {
    action: 'view_details',
    label: 'View full details',
    icon: 'info',
    requiresAuth: false,
    requiresPaidTier: false
  }
];
```

### 1.9 Offline Cache Types

```typescript
interface CachedResult {
  query: string;
  mode: 'single' | 'recipe' | 'diet';
  results: SimilarityResult[];
  timestamp: number;
  expiresAt: number;
}

const CACHE_TTL_MS = 24 * 60 * 60 * 1000;  // 24 hours
const MAX_CACHED_RESULTS = 20;
const RESULTS_CACHE_KEY = 'mealswapp_cached_results';
```

### 1.10 Complete ResultsGrid State

```typescript
interface ResultsGridState {
  status: ResultsStatus;
  results: SimilarityResult[];
  pagination: PaginationState;
  sort: SortState;
  view: ViewConfig;
  selection: SelectionState;
  loading: LoadingState;
  error: ResultsError | null;
  sourceItem: FoodItem | null;         // The item being replaced
  isOffline: boolean;
  isCachedData: boolean;
  lastUpdated: number | null;
}

const INITIAL_RESULTS_GRID_STATE: ResultsGridState = {
  status: 'idle',
  results: [],
  pagination: {
    currentPage: 1,
    pageSize: DEFAULT_PAGE_SIZE,
    totalCount: 0,
    totalPages: 0,
    hasNextPage: false,
    hasPreviousPage: false
  },
  sort: DEFAULT_SORT,
  view: DEFAULT_VIEW_CONFIG,
  selection: {
    selectedItemId: null,
    comparisonItemIds: [],
    maxComparisonItems: MAX_COMPARISON_ITEMS
  },
  loading: {
    isLoading: false,
    isLoadingMore: false,
    loadingMessage: ''
  },
  error: null,
  sourceItem: null,
  isOffline: false,
  isCachedData: false,
  lastUpdated: null
};
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Initialization Flow

```
ON ResultsGrid Mount:
  1. Register event listeners
     1.1. Subscribe to search events from SearchView: 'search:submit'
     1.2. Add online/offline listeners: 'online', 'offline'
     1.3. Add resize listener for responsive columns

  2. Check for initial search parameters (from URL or props)
     2.1. Parse URL query parameters: ?q=, &mode=, &page=, &sort=
     2.2. IF parameters exist:
          - Set state.sourceItem from URL
          - Trigger initial search via TanStack Query
     2.3. ELSE:
          - Set state.status = 'idle'

  3. Load view preferences from localStorage
     3.1. Read 'mealswapp_results_view' key
     3.2. IF exists AND valid:
          - Set state.view = storedPreferences
     3.3. Apply responsive column count based on viewport

  4. Check online status
     4.1. Set state.isOffline = !navigator.onLine
     4.2. IF offline:
          - Attempt to load cached results from Service Worker cache
```

### 2.2 Receiving Search Results

```
ON Search Results Received (searchResponse: SearchResponse):
  1. Validate response
     IF searchResponse.items is empty OR null:
       1.1. Determine empty state type based on context
       1.2. Set state.status = 'empty'
       1.3. Set state.results = []
       1.4. RETURN

  2. Transform API response to SimilarityResult[]
     FOR each item IN searchResponse.items:
       2.1. Map similarity score to tier:
            tier = getSimilarityTier(item.similarityScore)
       2.2. Get indicator config:
            indicator = SIMILARITY_THRESHOLDS[tier]
       2.3. Create SimilarityResult:
            result = {
              item: item.foodItem,
              similarity: {
                tier: tier,
                score: item.similarityScore,
                colorHex: indicator.color,
                imageUrl: indicator.icon
              },
              matchingQuantity: item.matchingQuantity,
              matchType: searchResponse.matchType
            }
       2.4. Add to results array

  3. Update pagination state
     state.pagination = {
       currentPage: searchResponse.page,
       pageSize: searchResponse.pageSize,
       totalCount: searchResponse.totalCount,
       totalPages: Math.ceil(searchResponse.totalCount / searchResponse.pageSize),
       hasNextPage: searchResponse.page < totalPages,
       hasPreviousPage: searchResponse.page > 1
     }

  4. Apply current sort
     CALL sortResults(state.results, state.sort)

  5. Update state
     state.status = 'success'
     state.results = sortedResults
     state.lastUpdated = Date.now()
     state.isCachedData = false

  6. Cache results for offline use
     CALL cacheResults(searchResponse.query, state.results)

  7. Announce results to screen reader
     announce(`${state.pagination.totalCount} results found`)

FUNCTION getSimilarityTier(score: number): SimilarityTier
  IF score >= SIMILARITY_THRESHOLDS.excellent.min:
    RETURN 'excellent'
  ELSE IF score >= SIMILARITY_THRESHOLDS.good.min:
    RETURN 'good'
  ELSE IF score >= SIMILARITY_THRESHOLDS.fair.min:
    RETURN 'fair'
  ELSE:
    RETURN 'poor'
```

### 2.3 Sorting Results

```
FUNCTION handleSortChange(field: SortField):
  1. Determine new sort direction
     IF state.sort.field === field:
       // Toggle direction if same field
       newDirection = state.sort.direction === 'asc' ? 'desc' : 'asc'
     ELSE:
       // Default direction for new field
       IF field === 'similarity':
         newDirection = 'desc'  // Highest similarity first
       ELSE IF field === 'name':
         newDirection = 'asc'   // Alphabetical A-Z
       ELSE:
         newDirection = 'desc'  // Highest values first for macros

  2. Update sort state
     state.sort = { field: field, direction: newDirection }

  3. Sort results
     CALL sortResults(state.results, state.sort)

  4. Reset to first page
     state.pagination.currentPage = 1

  5. Update URL without navigation
     updateUrlParams({ sort: field, dir: newDirection, page: 1 })

FUNCTION sortResults(results: SimilarityResult[], sort: SortState): void
  results.sort((a, b) => {
    LET valueA, valueB

    SWITCH sort.field:
      CASE 'similarity':
        valueA = a.similarity.score
        valueB = b.similarity.score

      CASE 'name':
        valueA = a.item.name.toLowerCase()
        valueB = b.item.name.toLowerCase()

      CASE 'calories':
        valueA = a.item.macros.calories
        valueB = b.item.macros.calories

      CASE 'protein':
        valueA = a.item.macros.protein
        valueB = b.item.macros.protein

      CASE 'carbs':
        valueA = a.item.macros.carbs
        valueB = b.item.macros.carbs

      CASE 'fat':
        valueA = a.item.macros.fat
        valueB = b.item.macros.fat

    // Handle string comparison for names
    IF typeof valueA === 'string':
      comparison = valueA.localeCompare(valueB)
    ELSE:
      comparison = valueA - valueB

    RETURN sort.direction === 'asc' ? comparison : -comparison
  })
```

### 2.4 Pagination Handling

```
FUNCTION handlePageChange(newPage: number):
  1. Validate page number
     IF newPage < 1 OR newPage > state.pagination.totalPages:
       RETURN

  2. Show loading state for page transition
     state.loading.isLoadingMore = true
     state.loading.loadingMessage = 'Loading more results...'

  3. Request new page from API via TanStack Query
     TRY:
       // Use TanStack Query for caching, retry, and state management
       response = await queryClient.fetchQuery({
         queryKey: ['search/results', { query: currentQuery, page: newPage, pageSize: state.pagination.pageSize }],
         queryFn: () => GET /api/v1/search/results?{
           query: currentQuery,
           page: newPage,
           pageSize: state.pagination.pageSize,
           sort: state.sort.field,
           direction: state.sort.direction
         },
         staleTime: 5 * 60 * 1000 // 5 minutes
       })

        // Update results (append or replace based on mode)
        state.results = response.items.map(transformToSimilarityResult)
        state.pagination.currentPage = newPage
        state.pagination.hasNextPage = newPage < state.pagination.totalPages
        state.pagination.hasPreviousPage = newPage > 1

      CATCH error:
        CALL handleResultsError(error)

      FINALLY:
        state.loading.isLoadingMore = false

   4. Scroll to top of results grid
      document.getElementById('results-grid')?.scrollTo({ top: 0, behavior: 'smooth' })

   5. Update URL
      updateUrlParams({ page: newPage })

   6. Announce page change
      announce(`Page ${newPage} of ${state.pagination.totalPages}`)
```

### 2.5 View Mode Switching

```
FUNCTION handleViewModeChange(newMode: ViewMode):
  1. IF newMode === state.view.mode:
     RETURN

  2. Update view mode
     state.view.mode = newMode

  3. Adjust columns for grid mode
     IF newMode === 'grid':
       state.view.columns = getResponsiveColumns()
     ELSE:
       state.view.columns = 1

  4. Persist preference
     localStorage.setItem('mealswapp_results_view', JSON.stringify(state.view))

  5. Re-render grid with animation
     // CSS transition handles smooth layout change

FUNCTION getResponsiveColumns(): number
  width = window.innerWidth

  IF width >= 1440:
    RETURN RESPONSIVE_COLUMNS.wide
  ELSE IF width >= 1024:
    RETURN RESPONSIVE_COLUMNS.desktop
  ELSE IF width >= 640:
    RETURN RESPONSIVE_COLUMNS.tablet
  ELSE:
    RETURN RESPONSIVE_COLUMNS.mobile

FUNCTION handleResize():
  1. Get new column count
     newColumns = getResponsiveColumns()

  2. IF state.view.mode === 'grid' AND newColumns !== state.view.columns:
     state.view.columns = newColumns
```

### 2.6 Item Selection & Comparison

```
FUNCTION handleItemSelect(itemId: string):
  1. IF state.selection.selectedItemId === itemId:
     // Deselect if already selected
     state.selection.selectedItemId = null
     emit('item:deselected')
     RETURN

  2. Set selected item
     state.selection.selectedItemId = itemId

  3. Find full item data
     selectedItem = state.results.find(r => r.item.id === itemId)

  4. Emit selection event with full data
     emit('item:selected', {
       item: selectedItem.item,
       similarity: selectedItem.similarity,
       matchingQuantity: selectedItem.matchingQuantity
     })

  5. Update URL with selected item (for sharing)
     updateUrlParams({ selected: itemId })

FUNCTION handleAddToComparison(itemId: string):
  1. Check if already in comparison
     IF state.selection.comparisonItemIds.includes(itemId):
       1.1. Remove from comparison
       state.selection.comparisonItemIds =
         state.selection.comparisonItemIds.filter(id => id !== itemId)
       1.2. announce(`Item removed from comparison`)
       RETURN

  2. Check comparison limit
     IF state.selection.comparisonItemIds.length >= MAX_COMPARISON_ITEMS:
       2.1. Show toast: `Maximum ${MAX_COMPARISON_ITEMS} items can be compared`
       2.2. RETURN

  3. Add to comparison
     state.selection.comparisonItemIds.push(itemId)

  4. Announce addition
     announce(`Item added to comparison. ${state.selection.comparisonItemIds.length} of ${MAX_COMPARISON_ITEMS} items selected.`)

FUNCTION handleViewComparison():
  1. Gather comparison items
     comparisonItems = state.selection.comparisonItemIds.map(id =>
       state.results.find(r => r.item.id === id)
     )

  2. Emit comparison view event
     emit('comparison:view', {
       items: comparisonItems,
       sourceItem: state.sourceItem
     })

FUNCTION handleClearComparison():
  state.selection.comparisonItemIds = []
  announce('Comparison cleared')
```

### 2.7 Card Actions Handling

```
FUNCTION handleCardAction(action: CardAction, itemId: string):
  1. Find item
     result = state.results.find(r => r.item.id === itemId)
     IF !result:
       RETURN

  2. Handle action based on type:

     CASE action === 'select':
       CALL handleItemSelect(itemId)

     CASE action === 'save':
       2.1. Check authentication
            IF !isAuthenticated():
              emit('auth:required', { redirectAfter: 'save', itemId })
              RETURN
       2.2. Call save API
            TRY:
              await POST /api/v1/user/saved-items { itemId }
              showToast('Item saved successfully')
              // Update item's saved status locally
              result.isSaved = true
            CATCH error:
              showToast('Failed to save item. Please try again.')

     CASE action === 'compare':
       3.1. Check subscription (if required)
            IF CARD_ACTIONS[action].requiresPaidTier AND !hasPaidSubscription():
              emit('upgrade:required', { feature: 'comparison' })
              RETURN
       3.2. CALL handleAddToComparison(itemId)

     CASE action === 'view_details':
       4.1. Navigate to detail view
            emit('item:view_details', { item: result.item, similarity: result.similarity })
            // Or navigate: router.push(`/items/${itemId}`)

FUNCTION handleCardHover(itemId: string, isHovered: boolean):
  1. IF isHovered:
     // Preload item details for faster access
     prefetchItemDetails(itemId)

  2. Update hover state for visual feedback
     // Handled by CSS :hover, no state update needed
```

### 2.8 Loading States

```
FUNCTION showLoadingState(message?: string):
  state.status = 'loading'
  state.loading.isLoading = true
  state.loading.loadingMessage = message || 'Searching for alternatives...'

FUNCTION showLoadingMoreState():
  state.loading.isLoadingMore = true
  state.loading.loadingMessage = 'Loading more results...'

FUNCTION hideLoadingState():
  state.loading.isLoading = false
  state.loading.isLoadingMore = false
  state.loading.loadingMessage = ''

FUNCTION renderLoadingSkeletons(count: number): void
  // Render skeleton cards matching current view mode
  IF state.view.mode === 'grid':
    RETURN renderGridSkeletons(count, state.view.columns)
  ELSE:
    RETURN renderListSkeletons(count)

FUNCTION getLoadingSkeletonCount(): number
  // Return count based on viewport and view mode
  IF state.view.mode === 'list':
    RETURN 5
  ELSE:
    RETURN state.view.columns * 2  // Two rows of skeletons
```

### 2.9 Error Handling

```
FUNCTION handleResultsError(error: unknown):
  1. Classify error
     errorState = classifyResultsError(error)

  2. Update state
     state.status = 'error'
     state.error = errorState

  3. Log error for monitoring
     logError('results_grid_error', error, { query: currentQuery })

  4. Check for offline fallback
     IF errorState.type === 'NETWORK_ERROR' AND hasCachedResults():
       4.1. Load cached results
       4.2. Set state.isCachedData = true
       4.3. Show offline banner with cache age
       4.4. state.status = 'success'
       RETURN

  5. Announce error to screen reader
     announce(errorState.message)

FUNCTION classifyResultsError(error: unknown): ResultsError
  IF !navigator.onLine OR error instanceof NetworkError:
    RETURN {
      type: 'NETWORK_ERROR',
      message: "You're offline. Please check your connection.",
      retryable: true,
      showCachedData: true
    }

  IF error.name === 'AbortError' OR error.name === 'TimeoutError':
    RETURN {
      type: 'TIMEOUT_ERROR',
      message: 'Search is taking too long. Please try again.',
      retryable: true,
      showCachedData: false
    }

  IF error.status === 429:
    retryAfter = parseInt(error.headers?.get('Retry-After') || '60')
    RETURN {
      type: 'RATE_LIMITED',
      message: 'Too many searches. Please wait a moment.',
      retryable: true,
      retryAfterSeconds: retryAfter,
      showCachedData: false
    }

  IF error.status >= 500:
    RETURN {
      type: 'SERVER_ERROR',
      message: 'Something went wrong on our end. Please try again.',
      retryable: true,
      showCachedData: false
    }

  RETURN {
    type: 'UNKNOWN_ERROR',
    message: 'An unexpected error occurred.',
    retryable: true,
    showCachedData: false
  }

FUNCTION handleRetry():
  1. Clear error state
     state.error = null

  2. Re-trigger last search
     emit('search:retry', { query: lastQuery, page: state.pagination.currentPage })
```

### 2.10 Offline Caching (Service Worker + localStorage)

```
FUNCTION cacheResults(query: string, results: SimilarityResult[]):
  1. Store in Service Worker cache for offline access
     // Service Worker caches API responses
     cacheName = 'search-results-v1'
     cache.put(`/api/v1/search/results?q=${query}`, new Response(JSON.stringify({
       query,
       results,
       timestamp: Date.now()
     })))

  2. Also persist to localStorage for quick access
     TRY:
       existingCache = JSON.parse(localStorage.getItem(RESULTS_CACHE_KEY) || '[]')
     CATCH:
       existingCache = []

  3. Check for existing entry for this query
     existingIndex = existingCache.findIndex(c => c.query === query)
     IF existingIndex !== -1:
       existingCache.splice(existingIndex, 1)

  4. Create new cache entry
     newEntry: CachedResult = {
       query: query,
       mode: currentSearchMode,
       results: results,
       timestamp: Date.now(),
       expiresAt: Date.now() + CACHE_TTL_MS
     }

  5. Add to beginning (most recent first)
     existingCache.unshift(newEntry)

  6. Trim to max size and remove expired
     now = Date.now()
     existingCache = existingCache
       .filter(c => c.expiresAt > now)
       .slice(0, MAX_CACHED_RESULTS)

  7. Persist cache to localStorage
     TRY:
       localStorage.setItem(RESULTS_CACHE_KEY, JSON.stringify(existingCache))
     CATCH quotaError:
       // Storage full - remove oldest entries
       existingCache = existingCache.slice(0, MAX_CACHED_RESULTS / 2)
       localStorage.setItem(RESULTS_CACHE_KEY, JSON.stringify(existingCache))

FUNCTION loadCachedResults(query: string): SimilarityResult[] | null
  1. First try Service Worker cache (preferred for offline)
     TRY:
       cachedResponse = await caches.match(`/api/v1/search/results?q=${query}`)
       IF cachedResponse:
         data = await cachedResponse.json()
         IF data.expiresAt > Date.now():
           RETURN data.results
     CATCH:
       // Fall through to localStorage

  2. Fall back to localStorage
     TRY:
       cache = JSON.parse(localStorage.getItem(RESULTS_CACHE_KEY) || '[]')
     CATCH:
       RETURN null

  3. Find matching entry
     entry = cache.find(c => c.query === query && c.expiresAt > Date.now())

  4. IF entry found:
     RETURN entry.results
  ELSE:
     RETURN null

FUNCTION hasCachedResults(): boolean
  // Check both caches
  cache = JSON.parse(localStorage.getItem(RESULTS_CACHE_KEY) || '[]')
  RETURN cache.length > 0

FUNCTION getCacheAge(query: string): string | null
  cache = JSON.parse(localStorage.getItem(RESULTS_CACHE_KEY) || '[]')
  entry = cache.find(c => c.query === query)
  IF entry:
    ageMs = Date.now() - entry.timestamp
    IF ageMs < 60000:
      RETURN 'less than a minute ago'
    ELSE IF ageMs < 3600000:
      RETURN `${Math.floor(ageMs / 60000)} minutes ago`
    ELSE:
      RETURN `${Math.floor(ageMs / 3600000)} hours ago`
  RETURN null
```

### 2.11 Macro Display Formatting

```
FUNCTION formatMacroValue(value: number, unit: 'g' | 'kcal'): string
  IF unit === 'kcal':
    RETURN Math.round(value).toString()
  ELSE:
    // Display one decimal place for grams
    IF value >= 10:
      RETURN Math.round(value).toString()
    ELSE:
      RETURN value.toFixed(1)

FUNCTION calculateScaledMacros(baseMacros: MacroNutrients, quantity: number): MacroNutrients
  scaleFactor = quantity / 100  // Base macros are per 100g/ml
  RETURN {
    protein: baseMacros.protein * scaleFactor,
    carbs: baseMacros.carbs * scaleFactor,
    fat: baseMacros.fat * scaleFactor,
    calories: baseMacros.calories * scaleFactor
  }

FUNCTION getMacroBarWidths(macros: MacroNutrients): { protein: number; carbs: number; fat: number }
  // Calculate percentage of calories from each macro
  proteinCals = macros.protein * 4
  carbsCals = macros.carbs * 4
  fatCals = macros.fat * 9
  totalCals = proteinCals + carbsCals + fatCals

  IF totalCals === 0:
    RETURN { protein: 0, carbs: 0, fat: 0 }

  RETURN {
    protein: (proteinCals / totalCals) * 100,
    carbs: (carbsCals / totalCals) * 100,
    fat: (fatCals / totalCals) * 100
  }
```

### 2.12 Image Loading & Fallback

```
FUNCTION handleImageLoad(itemId: string, imageUrl: string):
  // Image loaded successfully - no action needed

FUNCTION handleImageError(itemId: string):
  1. Mark image as failed
     failedImages.add(itemId)

  2. Use fallback placeholder
     // Fallback handled by CSS or component state

FUNCTION getImageSrc(item: FoodItem): string
  IF item.imageUrl AND !failedImages.has(item.id):
    RETURN item.imageUrl
  ELSE:
    // Return category-based placeholder
    RETURN getCategoryPlaceholder(item.categoryTags)

FUNCTION getCategoryPlaceholder(tags: Tag[]): string
  // Map category to default placeholder image
  categoryTag = tags.find(t => t.type === 'category')
  IF categoryTag:
    SWITCH categoryTag.name:
      CASE 'Fruits': RETURN '/assets/placeholders/fruit.svg'
      CASE 'Vegetables': RETURN '/assets/placeholders/vegetable.svg'
      CASE 'Proteins': RETURN '/assets/placeholders/protein.svg'
      CASE 'Grains': RETURN '/assets/placeholders/grain.svg'
      CASE 'Dairy': RETURN '/assets/placeholders/dairy.svg'
      DEFAULT: RETURN '/assets/placeholders/food-default.svg'
  RETURN '/assets/placeholders/food-default.svg'
```

---

## 3. State Management & Error Handling

State management uses Svelte stores for local UI state combined with TanStack Query for server state (caching, retry, synchronization).

### 3.1 State Transitions Diagram

```
                                    ┌─────────────┐
                                    │    IDLE     │
                                    │ (No search) │
                                    └──────┬──────┘
                                           │
                                    Search submitted
                                           │
                                           ▼
                                    ┌─────────────┐
                           ┌───────│   LOADING   │───────┐
                           │       │             │       │
                           │       └─────────────┘       │
                           │                             │
                     Success (results)            Error / Empty
                           │                             │
                           ▼                             ▼
                    ┌─────────────┐              ┌─────────────┐
                    │   SUCCESS   │              │    ERROR    │
                    │ (Results    │              │   /EMPTY    │
                    │  displayed) │              │             │
                    └──────┬──────┘              └──────┬──────┘
                           │                           │
              ┌────────────┼────────────┐              │
              │            │            │              │
         Sort/Filter    Paginate    New Search     Retry
              │            │            │              │
              └────────────┴─────┬──────┴──────────────┘
                                 │
                                 ▼
                          ┌─────────────┐
                          │   LOADING   │
                          └─────────────┘
```

### 3.2 Offline State Diagram (Service Worker + localStorage)

```
                     ┌─────────────┐
                     │   ONLINE    │
                     │  (Normal)   │
                     └──────┬──────┘
                            │
                     Network lost
                            │
                            ▼
                     ┌─────────────┐
               ┌─────│   OFFLINE   │─────┐
               │     └─────────────┘     │
               │                         │
         Has Service              No cached
         Worker cache               data
               │                         │
               ▼                         ▼
        ┌─────────────┐          ┌─────────────┐
        │   CACHED    │          │   OFFLINE   │
        │  (Stale     │          │   ERROR     │
        │   banner)   │          │             │
        └─────────────┘          └─────────────┘
               │                         │
               └──────────┬──────────────┘
                          │
                   Network restored
                          │
                          ▼
                   ┌─────────────┐
                   │   ONLINE    │
                   │  (Refresh)  │
                   └─────────────┘
```

### 3.3 Error States

| Error State | Trigger | User Message | Recovery Action |
|:------------|:--------|:-------------|:----------------|
| **NETWORK_ERROR** | Fetch fails, offline | "You're offline. Showing cached results." | Auto-retry on reconnect; Show cached data if available |
| **TIMEOUT_ERROR** | Response > 10s | "Search is taking too long. Please try again." | Manual retry button |
| **NO_RESULTS** | Empty response, valid query | "No items found for '{query}'." | Suggestions: broaden terms, remove filters |
| **NO_MATCHES** | No items above 40% threshold | "No similar items found above 40% match." | Suggestions: try different item, adjust toggles |
| **SERVER_ERROR** | 5xx response | "Something went wrong. Please try again." | Manual retry button |
| **RATE_LIMITED** | 429 response | "Too many searches. Please wait {n} seconds." | Countdown timer; Auto-retry after delay |
| **IMAGE_LOAD_ERROR** | Image 404/timeout | Silent (show placeholder) | Use category placeholder |

### 3.4 Error Types Definition

```typescript
type ResultsErrorType =
  | 'NETWORK_ERROR'
  | 'TIMEOUT_ERROR'
  | 'NO_RESULTS'
  | 'NO_MATCHES'
  | 'SERVER_ERROR'
  | 'RATE_LIMITED'
  | 'UNKNOWN_ERROR';

interface ResultsError {
  type: ResultsErrorType;
  message: string;
  retryable: boolean;
  retryAfterSeconds?: number;
  showCachedData: boolean;
}
```

### 3.5 Graceful Degradation

| Scenario | Degraded Functionality | Core Functionality Preserved |
|:---------|:-----------------------|:-----------------------------|
| **Similarity service slow** | Results without similarity scores/colors (TanStack Query timeout) | Item list, basic data |
| **Image CDN down** | Category placeholder images | All data, interactions |
| **Offline mode** | Service Worker cache serves stale results with banner | View previous results |
| **Comparison feature down** | Comparison disabled | Selection, details, save |
| **Save API down** | Save action shows error (with TanStack Query retry) | Browse, select, compare |

---

## 4. Component Interfaces

### 4.1 ResultsGrid Component Props

```typescript
interface ResultsGridProps {
  sourceItem?: FoodItem;
  initialResults?: SimilarityResult[];
  onItemSelect?: (item: FoodItem, similarity: SimilarityIndicator) => void;
  onComparisonRequest?: (items: SimilarityResult[]) => void;
  onError?: (error: ResultsError) => void;
  className?: string;
}

// Svelte stores for state management
import { writable, derived } from 'svelte/store';

// Server state managed by TanStack Query
import { createQuery, createMutation } from '@tanstack/svelte-query';
```

### 4.2 Internal Component Functions

```typescript
// Results Management
function receiveSearchResults(response: SearchResponse): void;
function transformToSimilarityResult(item: SearchResponseItem): SimilarityResult;
function getSimilarityTier(score: number): SimilarityTier;
function clearResults(): void;

// Sorting
function handleSortChange(field: SortField): void;
function sortResults(results: SimilarityResult[], sort: SortState): void;
function getSortIndicator(field: SortField): 'asc' | 'desc' | null;

// Pagination
function handlePageChange(page: number): void;
function handlePageSizeChange(size: number): void;
function calculatePaginationRange(): { start: number; end: number };

// View Mode
function handleViewModeChange(mode: ViewMode): void;
function getResponsiveColumns(): number;
function handleResize(): void;

// Selection & Comparison
function handleItemSelect(itemId: string): void;
function handleAddToComparison(itemId: string): void;
function handleViewComparison(): void;
function handleClearComparison(): void;
function isItemSelected(itemId: string): boolean;
function isItemInComparison(itemId: string): boolean;

// Card Actions
function handleCardAction(action: CardAction, itemId: string): void;
function handleCardHover(itemId: string, isHovered: boolean): void;
function getAvailableActions(item: FoodItem): CardAction[];

// Loading States
function showLoadingState(message?: string): void;
function showLoadingMoreState(): void;
function hideLoadingState(): void;
function renderLoadingSkeletons(count: number): void;

// Error Handling
function handleResultsError(error: unknown): void;
function classifyResultsError(error: unknown): ResultsError;
function handleRetry(): void;
function dismissError(): void;

// Offline & Caching
function cacheResults(query: string, results: SimilarityResult[]): void;
function loadCachedResults(query: string): SimilarityResult[] | null;
function hasCachedResults(): boolean;
function getCacheAge(query: string): string | null;
function handleOnline(): void;
function handleOffline(): void;

// Display Formatting
function formatMacroValue(value: number, unit: 'g' | 'kcal'): string;
function calculateScaledMacros(macros: MacroNutrients, quantity: number): MacroNutrients;
function getMacroBarWidths(macros: MacroNutrients): MacroBarWidths;

// Image Handling
function handleImageLoad(itemId: string): void;
function handleImageError(itemId: string): void;
function getImageSrc(item: FoodItem): string;
function getCategoryPlaceholder(tags: Tag[]): string;

// URL State
function updateUrlParams(params: Partial<UrlParams>): void;
function parseUrlParams(): UrlParams;

// Accessibility
function announce(message: string): void;
function getFocusableCardElements(): HTMLElement[];
```

### 4.3 Event Emitters

```typescript
// Events emitted by ResultsGrid
interface ResultsGridEvents {
  'item:selected': (data: { item: FoodItem; similarity: SimilarityIndicator; matchingQuantity: number }) => void;
  'item:deselected': () => void;
  'item:view_details': (data: { item: FoodItem; similarity: SimilarityIndicator }) => void;
  'comparison:view': (data: { items: SimilarityResult[]; sourceItem: FoodItem | null }) => void;
  'search:retry': (data: { query: string; page: number }) => void;
  'auth:required': (data: { redirectAfter: string; itemId?: string }) => void;
  'upgrade:required': (data: { feature: string }) => void;
  'results:loaded': (data: { count: number; cached: boolean }) => void;
  'results:error': (error: ResultsError) => void;
}
```

### 4.4 API Interface Contracts

```typescript
// GET /api/v1/search/results
interface SearchResultsRequest {
  query?: string;
  ingredients?: Array<{ id: string; quantity: number; unit: string }>;
  mode: 'single' | 'recipe' | 'diet';
  page?: number;
  pageSize?: number;
  sort?: SortField;
  direction?: SortDirection;
  filters?: TagFilter[];
  macroToggles?: MacroToggleState;
}

interface SearchResultsResponse {
  items: SearchResponseItem[];
  totalCount: number;
  page: number;
  pageSize: number;
  matchType: 'calorie' | 'protein';
  cached: boolean;
}

interface SearchResponseItem {
  foodItem: FoodItem;
  similarityScore: number;
  matchingQuantity: number;
}

// POST /api/v1/user/saved-items
interface SaveItemRequest {
  itemId: string;
}

interface SaveItemResponse {
  success: boolean;
  savedAt: string;
}
```

### 4.5 localStorage Keys

| Key | Type | Description |
|:----|:-----|:------------|
| `mealswapp_cached_results` | `CachedResult[]` | UI preferences and quick-access cache (supplemented by Service Worker) |
| `mealswapp_results_view` | `ViewConfig` | View mode and preferences (Svelte store sync) |
| `mealswapp_results_pagesize` | `number` | Preferred page size |

---

## 5. UI Component Structure

```
ResultsGrid
├── ResultsHeader
│   ├── ResultsCount
│   │   └── "X results for {query}"
│   │
│   ├── SourceItemBadge (when replacing specific item)
│   │   ├── SourceItemImage
│   │   ├── SourceItemName
│   │   └── ClearButton
│   │
│   ├── SortControls
│   │   ├── SortDropdown
│   │   │   ├── SortOption (Similarity)
│   │   │   ├── SortOption (Name)
│   │   │   ├── SortOption (Calories)
│   │   │   ├── SortOption (Protein)
│   │   │   ├── SortOption (Carbs)
│   │   │   └── SortOption (Fat)
│   │   └── SortDirectionToggle
│   │
│   └── ViewControls
│       ├── ViewModeToggle (Grid/List)
│       └── PageSizeSelector
│
├── OfflineBanner (conditional: when showing cached data)
│   ├── OfflineIcon
│   ├── OfflineMessage
│   └── CacheAgeIndicator
│
├── ComparisonBar (conditional: when items in comparison)
│   ├── ComparisonCount ("2 of 3 items")
│   ├── ComparisonItemPreviews[]
│   ├── ViewComparisonButton
│   └── ClearComparisonButton
│
├── ResultsContainer
│   ├── LoadingState (conditional: when loading)
│   │   └── SkeletonCard[] (matches view mode)
│   │
│   ├── ErrorState (conditional: when error)
│   │   ├── ErrorIcon
│   │   ├── ErrorTitle
│   │   ├── ErrorMessage
│   │   ├── RetryButton (if retryable)
│   │   └── SuggestionsList
│   │
│   ├── EmptyState (conditional: when empty)
│   │   ├── EmptyIcon
│   │   ├── EmptyTitle
│   │   ├── EmptyMessage
│   │   ├── SuggestionsList
│   │   └── ClearFiltersButton (conditional)
│   │
│   └── ResultsList (conditional: when success)
│       └── ResultCard[] (Grid or List layout)
│           ├── CardImage
│           │   ├── FoodImage
│           │   ├── SimilarityBadge
│           │   │   ├── SimilarityIcon (server-hosted)
│           │   │   └── SimilarityPercentage
│           │   └── ComparisonCheckbox (overlay)
│           │
│           ├── CardContent
│           │   ├── ItemName
│           │   ├── TagChips
│           │   │   └── TagChip[] (category & functionality)
│           │   │
│           │   ├── MacroSummary
│           │   │   ├── CalorieDisplay
│           │   │   └── MacroBar (P/C/F visual)
│           │   │
│           │   ├── MacroDetails (expandable)
│           │   │   ├── ProteinRow
│           │   │   ├── CarbsRow
│           │   │   └── FatRow
│           │   │
│           │   └── MatchingQuantity
│           │       └── "Use Xg to match calories/protein"
│           │
│           └── CardActions
│               ├── SelectButton (primary)
│               ├── SaveButton
│               ├── CompareButton
│               └── DetailsButton
│
├── Pagination
│   ├── PreviousButton
│   ├── PageNumbers
│   │   └── PageButton[] (with ellipsis for large ranges)
│   ├── NextButton
│   └── PageInfo ("Page X of Y")
│
└── ScreenReaderAnnouncements (aria-live region)
```

---

## 6. Accessibility Requirements

| Element | ARIA Attributes | Keyboard Support |
|:--------|:----------------|:-----------------|
| Results Grid | `role="grid"`, `aria-label="Search results"`, `aria-busy` | Tab to enter, Arrow keys |
| Result Card | `role="gridcell"`, `aria-selected`, `tabindex="0"` | Enter to select, Space for actions |
| Similarity Badge | `aria-label="Similarity: X%"` | - |
| Sort Dropdown | `role="listbox"`, `aria-label="Sort by"` | Arrow keys, Enter to select |
| View Toggle | `role="radiogroup"`, `aria-label="View mode"` | Arrow keys within group |
| Pagination | `role="navigation"`, `aria-label="Pagination"` | Tab between buttons |
| Page Button | `aria-current="page"` (current), `aria-label="Page X"` | Enter/Space |
| Comparison Checkbox | `role="checkbox"`, `aria-checked`, `aria-label="Add {name} to comparison"` | Space to toggle |
| Error State | `role="alert"`, `aria-live="assertive"` | - |
| Loading State | `aria-busy="true"`, `aria-live="polite"` | - |

**Focus Management:**
- On results load: Focus first result card
- On error: Focus retry button
- On page change: Focus first result of new page
- On card select: Maintain focus on card, announce selection
- On comparison add: Announce count, maintain focus
- On empty state: Focus suggestions or search input

**Screen Reader Announcements:**
- Results loaded: "{count} results found, sorted by {field}"
- Sort changed: "Sorted by {field}, {direction}"
- Page changed: "Page {n} of {total}"
- Item selected: "{name} selected as replacement"
- Comparison updated: "{name} added to comparison. {n} of {max} items."
- Error: Announce error message
- Offline: "Showing cached results from {time}"

---

## 7. Visual Design Specifications

### 7.1 Similarity Indicator Styling

| Tier | Score Range | Background | Text Color | Icon |
|:-----|:------------|:-----------|:-----------|:-----|
| Excellent | 85-100% | `#22C55E` (green) | White | `/assets/indicators/star.png` |
| Good | 70-84% | `#84CC16` (lime) | White | `/assets/indicators/sparkle.png` |
| Fair | 55-69% | `#EAB308` (yellow) | Black | `/assets/indicators/thumbs-up.png` |
| Poor | 40-54% | `#EF4444` (red) | White | `/assets/indicators/thumbs-down.png` |

### 7.2 Card Dimensions

| View Mode | Card Width | Card Height | Gap |
|:----------|:-----------|:------------|:----|
| Grid (mobile) | 100% | Auto | 16px |
| Grid (tablet) | calc(50% - 8px) | Auto | 16px |
| Grid (desktop) | calc(33.33% - 11px) | Auto | 16px |
| Grid (wide) | calc(25% - 12px) | Auto | 16px |
| List | 100% | 120px | 12px |

### 7.3 Macro Bar Colors

```css
--macro-protein: #3B82F6;  /* Blue */
--macro-carbs: #F59E0B;    /* Amber */
--macro-fat: #EF4444;      /* Red */
```

---

## 8. Performance Considerations

| Optimization | Implementation | Impact |
|:-------------|:---------------|:-------|
| **TanStack Query caching** | Automatic caching with configurable staleTime | Reduced API calls, instant back-nav |
| Virtual scrolling | Render only visible cards (for large result sets) | Memory: O(visible) vs O(total) |
| Image lazy loading | `loading="lazy"` + IntersectionObserver | Faster initial render |
| Skeleton loading | Placeholder cards matching layout | Perceived performance |
| Debounced resize | 100ms debounce on resize handler | Prevent layout thrashing |
| Memoized sorting | Memo sort results, skip if unchanged | Prevent unnecessary recalcs |
| Result caching | Service Worker + localStorage (LRU, 20 queries max) | Offline support, faster back-nav |
| Preload on hover | Prefetch item details on card hover | Faster detail view |
| CSS containment | `contain: layout style` on cards | Isolate repaints |

---

## Changelog

### 2026-01-22 (Rev 1.0)

**Added:**
- Initial detailed design document for ResultsGrid component
- Complete type definitions for results, similarity, pagination, and view state
- Step-by-step algorithms for result handling, sorting, pagination, and selection
- Offline caching specifications with Service Worker + localStorage (LRU eviction)
- TanStack Query integration for data fetching and caching
- Error handling and graceful degradation specifications
- Similarity indicator visual specifications per ARCH-003 requirements
- Accessibility requirements (WCAG 2.1 AA)
- Component interface contracts and event emitters
- UI component structure hierarchy
- Performance optimization strategies

**Updated for Tech Stack Compliance:**
- Added TanStack Query integration for pagination and API fetching
- Updated caching section to use Service Worker + localStorage dual-layer cache
- Added Svelte stores + TanStack Query to state management section
- Updated graceful degradation to reference TanStack Query timeout/retry behavior
