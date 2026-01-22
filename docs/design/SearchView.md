# Detailed Design: SearchView

**Traceability:** [ARCH-001]

---

## 1. Data Structures & Types

### 1.1 Search Mode Types

```typescript
type SearchMode = 'single' | 'recipe' | 'diet';

interface SearchModeConfig {
  mode: SearchMode;
  label: string;
  description: string;
  maxIngredients: number;      // 1 for single, unlimited for recipe/diet
  showIngredientList: boolean; // false for single, true for recipe/diet
}

const SEARCH_MODES: Record<SearchMode, SearchModeConfig> = {
  single: {
    mode: 'single',
    label: 'Single Item',
    description: 'Find alternatives for one food item',
    maxIngredients: 1,
    showIngredientList: false
  },
  recipe: {
    mode: 'recipe',
    label: 'Recipe',
    description: 'Build a recipe and find alternatives',
    maxIngredients: Infinity,
    showIngredientList: true
  },
  diet: {
    mode: 'diet',
    label: 'Full Diet',
    description: 'Optimize your entire diet',
    maxIngredients: Infinity,
    showIngredientList: true
  }
};
```

### 1.2 Macronutrient Toggle Types

```typescript
type MacroType = 'protein' | 'carbs' | 'fat';

interface MacroToggle {
  type: MacroType;
  label: string;
  enabled: boolean;
  color: string;  // CSS variable reference
}

interface MacroToggleState {
  protein: boolean;
  carbs: boolean;
  fat: boolean;
}

const DEFAULT_MACRO_TOGGLES: MacroToggleState = {
  protein: true,
  carbs: true,
  fat: true
};
```

### 1.3 Search Input State

```typescript
interface SearchInputState {
  query: string;
  debouncedQuery: string;       // Query after 150ms debounce
  isFocused: boolean;
  isLoading: boolean;
  hasError: boolean;
  errorMessage: string | null;
}

const INITIAL_SEARCH_INPUT_STATE: SearchInputState = {
  query: '',
  debouncedQuery: '',
  isFocused: false,
  isLoading: false,
  hasError: false,
  errorMessage: null
};
```

### 1.4 Autocomplete Types

```typescript
interface AutocompleteItem {
  id: string;
  name: string;
  imageUrl: string | null;
  categoryTags: string[];
  macros: {
    protein: number;
    carbs: number;
    fat: number;
  };
  matchType: 'exact' | 'fuzzy' | 'partial';  // For ranking display
  matchScore: number;                         // 0-1 relevance score
}

interface AutocompleteState {
  items: AutocompleteItem[];
  isOpen: boolean;
  highlightedIndex: number;    // -1 = none highlighted
  totalCount: number;          // Total matches (may exceed displayed)
}

const INITIAL_AUTOCOMPLETE_STATE: AutocompleteState = {
  items: [],
  isOpen: false,
  highlightedIndex: -1,
  totalCount: 0
};

const MAX_AUTOCOMPLETE_ITEMS = 8;
```

### 1.5 Selected Ingredients List

```typescript
interface SelectedIngredient {
  id: string;
  name: string;
  imageUrl: string | null;
  quantity: number;
  unit: 'g' | 'ml' | 'oz' | 'fl_oz' | 'unit';
  macros: {
    protein: number;
    carbs: number;
    fat: number;
  };
  scaledMacros: {              // Macros adjusted for quantity
    protein: number;
    carbs: number;
    fat: number;
  };
}

interface IngredientListState {
  ingredients: SelectedIngredient[];
  totalMacros: {
    protein: number;
    carbs: number;
    fat: number;
    calories: number;
  };
}
```

### 1.6 Tag Filter Types

```typescript
type TagFilterMode = 'include' | 'exclude';

interface TagFilter {
  tagId: string;
  tagName: string;
  tagType: 'category' | 'functionality';
  mode: TagFilterMode;
}

interface TagFilterState {
  activeFilters: TagFilter[];
  availableCategoryTags: Tag[];
  availableFunctionalityTags: Tag[];
  isFilterPanelOpen: boolean;
}

interface Tag {
  id: string;
  name: string;
  type: 'category' | 'functionality';
  itemCount: number;           // Number of items with this tag
}
```

### 1.7 Search History Types

```typescript
interface SearchHistoryItem {
  id: string;
  query: string;
  mode: SearchMode;
  timestamp: number;           // Unix timestamp
  resultCount: number;
}

const MAX_SEARCH_HISTORY_ITEMS = 5;
const SEARCH_HISTORY_STORAGE_KEY = 'mealswapp_search_history';
```

### 1.8 Offline State Types

```typescript
interface OfflineState {
  isOnline: boolean;
  lastOnlineTimestamp: number | null;
  cachedQueriesCount: number;
  showOfflineBanner: boolean;
}
```

### 1.9 Complete SearchView State

```typescript
interface SearchViewState {
  mode: SearchMode;
  macroToggles: MacroToggleState;
  searchInput: SearchInputState;
  autocomplete: AutocompleteState;
  ingredients: IngredientListState;
  tagFilters: TagFilterState;
  searchHistory: SearchHistoryItem[];
  offline: OfflineState;
  theme: 'light' | 'dark' | 'system';
}

const INITIAL_SEARCH_VIEW_STATE: SearchViewState = {
  mode: 'single',
  macroToggles: DEFAULT_MACRO_TOGGLES,
  searchInput: INITIAL_SEARCH_INPUT_STATE,
  autocomplete: INITIAL_AUTOCOMPLETE_STATE,
  ingredients: { ingredients: [], totalMacros: { protein: 0, carbs: 0, fat: 0, calories: 0 } },
  tagFilters: { activeFilters: [], availableCategoryTags: [], availableFunctionalityTags: [], isFilterPanelOpen: false },
  searchHistory: [],
  offline: { isOnline: true, lastOnlineTimestamp: null, cachedQueriesCount: 0, showOfflineBanner: false },
  theme: 'system'
};
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Initialization Flow

```
ON SearchView Mount:
  1. Load search history from localStorage (key: SEARCH_HISTORY_STORAGE_KEY)
     1.1. Parse JSON array
     1.2. Validate each item has required fields (id, query, mode, timestamp)
     1.3. Filter out items older than 30 days
     1.4. Keep only MAX_SEARCH_HISTORY_ITEMS most recent items
     1.5. Store in state.searchHistory

  2. Load theme preference
     2.1. Check localStorage for 'mealswapp_theme' key
     2.2. If exists, set state.theme to stored value
     2.3. If not exists, set state.theme to 'system'
     2.4. Apply theme by calling applyTheme(state.theme)

  3. Initialize search mode to 'single'
     3.1. Set state.mode = 'single'

  4. Enable all macro toggles
     4.1. Set state.macroToggles = { protein: true, carbs: true, fat: true }

  5. Register offline event listeners
     5.1. Add event listener for 'online' event -> handleOnline()
     5.2. Add event listener for 'offline' event -> handleOffline()
     5.3. Set state.offline.isOnline = navigator.onLine

  6. Load available tags from API (or cache if offline)
     6.1. Call GET /api/v1/tags
     6.2. Populate state.tagFilters.availableCategoryTags
     6.3. Populate state.tagFilters.availableFunctionalityTags

  7. Focus search input if no other element is focused
```

### 2.2 Search Input Handling with Debounce

```
ON Search Input Change (newValue: string):
  1. Immediately update state.searchInput.query = newValue

  2. Clear any existing debounce timer

  3. If newValue is empty:
     3.1. Set state.searchInput.debouncedQuery = ''
     3.2. Close autocomplete: state.autocomplete.isOpen = false
     3.3. Clear autocomplete items: state.autocomplete.items = []
     3.4. RETURN early (no API call)

  4. Start new debounce timer (150ms):
     AFTER 150ms:
       4.1. Set state.searchInput.debouncedQuery = newValue
       4.2. Set state.searchInput.isLoading = true
       4.3. Call fetchAutocomplete(newValue)

FUNCTION fetchAutocomplete(query: string):
  1. Build request parameters:
     - query: string
     - mode: state.mode
     - filters: state.tagFilters.activeFilters
     - limit: MAX_AUTOCOMPLETE_ITEMS

  2. Check offline status:
     IF state.offline.isOnline === false:
       2.1. Retrieve cached results from localStorage
       2.2. Filter cached items matching query (case-insensitive contains)
       2.3. Set state.autocomplete.items = filteredCachedItems
       2.4. Set state.autocomplete.isOpen = true
       2.5. Set state.searchInput.isLoading = false
       2.6. RETURN

  3. Try API call:
     TRY:
       3.1. response = await GET /api/v1/search/autocomplete?{params}
       3.2. Set state.autocomplete.items = response.items
       3.3. Set state.autocomplete.totalCount = response.totalCount
       3.4. Set state.autocomplete.isOpen = true
       3.5. Set state.autocomplete.highlightedIndex = -1
       3.6. Cache results in localStorage for offline use
     CATCH error:
       3.7. IF error is NetworkError:
            - handleOffline()
            - Retry with cached data (step 2)
       3.8. ELSE:
            - Set state.searchInput.hasError = true
            - Set state.searchInput.errorMessage = mapErrorToUserMessage(error)
     FINALLY:
       3.9. Set state.searchInput.isLoading = false
```

### 2.3 Autocomplete Keyboard Navigation

```
ON Keydown in Search Input (event: KeyboardEvent):

  CASE event.key === 'ArrowDown':
    1. Prevent default scroll behavior
    2. IF autocomplete is closed AND query is not empty:
       - Open autocomplete
       - RETURN
    3. Calculate new index:
       newIndex = (state.autocomplete.highlightedIndex + 1) % state.autocomplete.items.length
    4. Set state.autocomplete.highlightedIndex = newIndex
    5. Scroll highlighted item into view

  CASE event.key === 'ArrowUp':
    1. Prevent default behavior
    2. IF highlightedIndex === -1:
       - Set highlightedIndex = items.length - 1
    3. ELSE:
       - newIndex = (highlightedIndex - 1 + items.length) % items.length
       - Set state.autocomplete.highlightedIndex = newIndex
    4. Scroll highlighted item into view

  CASE event.key === 'Enter':
    1. Prevent form submission
    2. IF highlightedIndex >= 0:
       - Select item at highlightedIndex
       - Call handleSelectAutocompleteItem(items[highlightedIndex])
    3. ELSE IF query is not empty:
       - Trigger full search with current query
       - Call handleSubmitSearch()

  CASE event.key === 'Escape':
    1. Close autocomplete dropdown
    2. Set state.autocomplete.isOpen = false
    3. Set state.autocomplete.highlightedIndex = -1

  CASE event.key === 'Tab':
    1. IF autocomplete is open AND highlightedIndex >= 0:
       - Select highlighted item
       - Prevent default tab behavior
    2. ELSE:
       - Allow default tab behavior (move to next focusable element)
```

### 2.4 Selecting an Autocomplete Item

```
FUNCTION handleSelectAutocompleteItem(item: AutocompleteItem):
  1. Close autocomplete dropdown
     1.1. Set state.autocomplete.isOpen = false
     1.2. Set state.autocomplete.highlightedIndex = -1

  2. Clear search input
     2.1. Set state.searchInput.query = ''
     2.2. Set state.searchInput.debouncedQuery = ''

  3. Based on current search mode:

     IF state.mode === 'single':
       3.1. Clear any existing ingredients
       3.2. Add item as the single selected ingredient
       3.3. Navigate to results view with this item
       3.4. Call navigateToResults(item.id)

     IF state.mode === 'recipe' OR state.mode === 'diet':
       3.1. Create new SelectedIngredient:
            newIngredient = {
              id: item.id,
              name: item.name,
              imageUrl: item.imageUrl,
              quantity: 100,          // Default to 100g/ml
              unit: 'g',              // Default unit
              macros: item.macros,
              scaledMacros: item.macros  // Initially same as base
            }
       3.2. Append to ingredients list:
            state.ingredients.ingredients.push(newIngredient)
       3.3. Recalculate total macros:
            Call recalculateTotalMacros()
       3.4. Return focus to search input for next ingredient

  4. Add to search history
     4.1. Call addToSearchHistory(item.name, state.mode)
```

### 2.5 Search Mode Switching

```
FUNCTION handleSearchModeChange(newMode: SearchMode):
  1. IF newMode === state.mode:
     - RETURN (no change needed)

  2. Store previous mode for transition logic
     previousMode = state.mode

  3. Update mode
     state.mode = newMode

  4. Handle ingredient list transitions:

     IF previousMode === 'single' AND (newMode === 'recipe' OR newMode === 'diet'):
       4.1. Keep any selected single item in the list
       4.2. Show ingredient list panel

     IF (previousMode === 'recipe' OR previousMode === 'diet') AND newMode === 'single':
       4.1. IF ingredients.length > 1:
            - Show confirmation dialog: "Switching to Single Item mode will clear your ingredient list. Continue?"
            - IF user confirms: Clear all ingredients
            - IF user cancels: Revert state.mode = previousMode, RETURN
       4.2. IF ingredients.length === 1:
            - Keep the single ingredient
       4.3. Hide ingredient list panel

  5. Clear current search input and autocomplete
     5.1. state.searchInput.query = ''
     5.2. state.autocomplete.isOpen = false
     5.3. state.autocomplete.items = []

  6. Update UI to reflect new mode
     6.1. Update mode selector visual state
     6.2. Show/hide ingredient list based on SEARCH_MODES[newMode].showIngredientList
```

### 2.6 Macro Toggle Handling

```
FUNCTION handleMacroToggle(macroType: MacroType):
  1. Get current state of this toggle
     currentValue = state.macroToggles[macroType]

  2. Count currently enabled toggles
     enabledCount = Object.values(state.macroToggles).filter(v => v).length

  3. Validation: At least one macro must remain enabled
     IF currentValue === true AND enabledCount === 1:
       3.1. Show toast notification: "At least one macronutrient must be selected"
       3.2. RETURN (don't allow disabling)

  4. Toggle the value
     state.macroToggles[macroType] = !currentValue

  5. IF search results are currently displayed:
     5.1. Re-fetch results with updated macro filters
     5.2. Similarity calculations will use only enabled macros

  6. Update visual state of toggle button
```

### 2.7 Ingredient Quantity Adjustment

```
FUNCTION handleQuantityChange(ingredientId: string, newQuantity: number):
  1. Validate quantity
     IF newQuantity <= 0:
       1.1. Show validation error
       1.2. RETURN
     IF newQuantity > 10000:  // Reasonable max
       1.1. Cap at 10000
       1.2. Show notification: "Maximum quantity is 10kg/10L"

  2. Find ingredient in list
     ingredientIndex = state.ingredients.ingredients.findIndex(i => i.id === ingredientId)
     IF ingredientIndex === -1:
       RETURN (ingredient not found)

  3. Update quantity
     state.ingredients.ingredients[ingredientIndex].quantity = newQuantity

  4. Recalculate scaled macros for this ingredient
     baseMacros = state.ingredients.ingredients[ingredientIndex].macros
     scaleFactor = newQuantity / 100  // Base macros are per 100g/ml
     state.ingredients.ingredients[ingredientIndex].scaledMacros = {
       protein: baseMacros.protein * scaleFactor,
       carbs: baseMacros.carbs * scaleFactor,
       fat: baseMacros.fat * scaleFactor
     }

  5. Recalculate total macros
     Call recalculateTotalMacros()

FUNCTION recalculateTotalMacros():
  1. Initialize totals
     totals = { protein: 0, carbs: 0, fat: 0, calories: 0 }

  2. Sum all scaled macros
     FOR each ingredient IN state.ingredients.ingredients:
       totals.protein += ingredient.scaledMacros.protein
       totals.carbs += ingredient.scaledMacros.carbs
       totals.fat += ingredient.scaledMacros.fat

  3. Calculate calories (standard formula)
     totals.calories = (totals.protein * 4) + (totals.carbs * 4) + (totals.fat * 9)

  4. Round to 1 decimal place
     totals.protein = Math.round(totals.protein * 10) / 10
     totals.carbs = Math.round(totals.carbs * 10) / 10
     totals.fat = Math.round(totals.fat * 10) / 10
     totals.calories = Math.round(totals.calories)

  5. Update state
     state.ingredients.totalMacros = totals
```

### 2.8 Removing an Ingredient

```
FUNCTION handleRemoveIngredient(ingredientId: string):
  1. Find ingredient index
     index = state.ingredients.ingredients.findIndex(i => i.id === ingredientId)
     IF index === -1:
       RETURN

  2. Remove from array
     state.ingredients.ingredients.splice(index, 1)

  3. Recalculate totals
     Call recalculateTotalMacros()

  4. IF list is now empty AND mode is 'single':
     - Focus search input
```

### 2.9 Tag Filtering

```
FUNCTION handleAddTagFilter(tag: Tag, mode: TagFilterMode):
  1. Check if tag is already in active filters
     existingIndex = state.tagFilters.activeFilters.findIndex(f => f.tagId === tag.id)

     IF existingIndex !== -1:
       1.1. IF existing filter has same mode:
            - RETURN (already applied)
       1.2. IF existing filter has different mode:
            - Update mode: state.tagFilters.activeFilters[existingIndex].mode = mode
            - Re-fetch search results
            - RETURN

  2. Create new filter
     newFilter: TagFilter = {
       tagId: tag.id,
       tagName: tag.name,
       tagType: tag.type,
       mode: mode
     }

  3. Add to active filters
     state.tagFilters.activeFilters.push(newFilter)

  4. IF search results are displayed:
     - Re-fetch with updated filters

  5. IF autocomplete is open:
     - Re-fetch autocomplete with updated filters

FUNCTION handleRemoveTagFilter(tagId: string):
  1. Find and remove filter
     state.tagFilters.activeFilters = state.tagFilters.activeFilters.filter(f => f.tagId !== tagId)

  2. Re-fetch results if applicable
```

### 2.10 Search History Management

```
FUNCTION addToSearchHistory(query: string, mode: SearchMode):
  1. Check for duplicate query
     existingIndex = state.searchHistory.findIndex(h => h.query.toLowerCase() === query.toLowerCase())
     IF existingIndex !== -1:
       1.1. Remove existing entry (will be re-added at top)
       state.searchHistory.splice(existingIndex, 1)

  2. Create new history item
     newItem: SearchHistoryItem = {
       id: generateUUID(),
       query: query,
       mode: mode,
       timestamp: Date.now(),
       resultCount: 0  // Updated after results load
     }

  3. Add to beginning of array
     state.searchHistory.unshift(newItem)

  4. Trim to max size
     IF state.searchHistory.length > MAX_SEARCH_HISTORY_ITEMS:
       state.searchHistory = state.searchHistory.slice(0, MAX_SEARCH_HISTORY_ITEMS)

  5. Persist to localStorage
     localStorage.setItem(SEARCH_HISTORY_STORAGE_KEY, JSON.stringify(state.searchHistory))

FUNCTION handleSelectHistoryItem(historyItem: SearchHistoryItem):
  1. Set search mode to history item's mode
     state.mode = historyItem.mode

  2. Set search query
     state.searchInput.query = historyItem.query

  3. Trigger search
     Call fetchAutocomplete(historyItem.query)

FUNCTION handleClearSearchHistory():
  1. Show confirmation dialog: "Clear all search history?"
  2. IF confirmed:
     2.1. state.searchHistory = []
     2.2. localStorage.removeItem(SEARCH_HISTORY_STORAGE_KEY)
```

### 2.11 Implicit Similarity Search Trigger

```
ON Search Input Blur OR Submit with Empty Query:
  1. Check conditions for implicit trigger:
     - state.searchInput.query is empty or whitespace
     - state.ingredients.ingredients.length >= 2
     - state.mode is 'recipe' or 'diet'

  2. IF all conditions met:
     2.1. Show loading state
     2.2. Build similarity search request:
          request = {
            ingredients: state.ingredients.ingredients.map(i => ({
              id: i.id,
              quantity: i.quantity,
              unit: i.unit
            })),
            macroToggles: state.macroToggles,
            filters: state.tagFilters.activeFilters
          }
     2.3. Call POST /api/v1/search/similar
     2.4. Navigate to results view with similarity results
```

### 2.12 Offline Mode Handling

```
FUNCTION handleOffline():
  1. Update state
     state.offline.isOnline = false
     state.offline.showOfflineBanner = true
     state.offline.lastOnlineTimestamp = Date.now()

  2. Load cached query count
     cachedQueries = localStorage.getItem('mealswapp_cached_queries')
     state.offline.cachedQueriesCount = cachedQueries ? JSON.parse(cachedQueries).length : 0

  3. Show offline banner with message:
     "You're offline. Showing cached results."

FUNCTION handleOnline():
  1. Update state
     state.offline.isOnline = true
     state.offline.showOfflineBanner = false

  2. IF there was a pending search:
     - Retry the search with fresh data

  3. Sync any locally modified data
     - Background sync of search history
```

### 2.13 Theme Application

```
FUNCTION applyTheme(theme: 'light' | 'dark' | 'system'):
  1. Determine effective theme
     IF theme === 'system':
       effectiveTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
     ELSE:
       effectiveTheme = theme

  2. Apply CSS variables based on effective theme
     IF effectiveTheme === 'light':
       document.documentElement.style.setProperty('--bg-primary', '#F7FCF7')
       document.documentElement.style.setProperty('--bg-surface', '#FFFFFF')
       document.documentElement.style.setProperty('--color-primary', '#166534')
       document.documentElement.style.setProperty('--color-secondary', '#DCFCE7')
       document.documentElement.style.setProperty('--color-accent', '#F97316')
       document.documentElement.style.setProperty('--color-error', '#DC2626')
       document.documentElement.style.setProperty('--text-primary', '#111827')
       document.documentElement.style.setProperty('--text-muted', '#6B7280')

     IF effectiveTheme === 'dark':
       document.documentElement.style.setProperty('--bg-primary', '#0A0F0A')
       document.documentElement.style.setProperty('--bg-surface', '#161D16')
       document.documentElement.style.setProperty('--color-primary', '#4ADE80')
       document.documentElement.style.setProperty('--color-secondary', '#86EFAC')
       document.documentElement.style.setProperty('--color-accent', '#FFB86C')
       document.documentElement.style.setProperty('--color-error', '#F87171')
       document.documentElement.style.setProperty('--text-primary', '#F3F4F6')
       document.documentElement.style.setProperty('--text-muted', '#9CA3AF')

  3. Set data attribute for component styling
     document.documentElement.setAttribute('data-theme', effectiveTheme)

  4. Persist preference
     localStorage.setItem('mealswapp_theme', theme)

  5. Listen for system theme changes (if theme === 'system')
     IF theme === 'system':
       window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
         applyTheme('system')
       })
```

---

## 3. State Management & Error Handling

### 3.1 State Transitions Diagram

```
                                    ┌─────────────┐
                                    │   INITIAL   │
                                    │   (Mount)   │
                                    └──────┬──────┘
                                           │
                                           ▼
                                    ┌─────────────┐
                                    │    IDLE     │
                      ┌────────────>│  (Ready)    │<────────────┐
                      │             └──────┬──────┘             │
                      │                    │                    │
                      │         User types │                    │
                      │                    ▼                    │
                      │             ┌─────────────┐             │
                      │             │  DEBOUNCING │             │
                      │             │  (150ms)    │             │
                      │             └──────┬──────┘             │
                      │                    │                    │
                      │         Timer fires│                    │
                      │                    ▼                    │
                      │             ┌─────────────┐             │
                      │             │   LOADING   │─────────────┤
                      │             │ (API call)  │   Error     │
                      │             └──────┬──────┘             │
                      │                    │                    │
                      │            Success │                    │
                      │                    ▼                    │
                      │             ┌─────────────┐             │
                      │             │ AUTOCOMPLETE│             │
                      │             │   OPEN      │─────────────┘
                      │             └──────┬──────┘   Escape/
                      │                    │         Click outside
                      │       Select item  │
                      │                    ▼
                      │             ┌─────────────┐
                      └─────────────│  SELECTED   │
                                    │ (Item added)│
                                    └─────────────┘
```

### 3.2 Error States

| Error State | Trigger | User Message | Recovery Action |
|:------------|:--------|:-------------|:----------------|
| **NETWORK_ERROR** | Fetch fails, offline | "You're offline. Showing cached results." | Auto-retry on reconnect; Show cached data |
| **API_TIMEOUT** | Response > 10s | "Search is taking longer than expected. Please try again." | Manual retry button; Cancel option |
| **NO_RESULTS** | Empty response array | "No items found for '{query}'. Try different keywords." | Show search suggestions; Clear filters option |
| **INVALID_INPUT** | Validation failure | "Please enter at least 2 characters to search." | Focus input; Show character count |
| **SERVER_ERROR** | 5xx response | "Something went wrong. Please try again." | Manual retry button; Log error for monitoring |
| **RATE_LIMITED** | 429 response | "Too many searches. Please wait a moment." | Show countdown timer; Auto-retry after delay |
| **CACHE_ERROR** | localStorage quota | Silent (log only) | Evict oldest cached queries |

### 3.3 Error Handling Implementation

```typescript
interface ErrorState {
  type: 'NETWORK_ERROR' | 'API_TIMEOUT' | 'NO_RESULTS' | 'INVALID_INPUT' | 'SERVER_ERROR' | 'RATE_LIMITED' | 'CACHE_ERROR';
  message: string;
  retryable: boolean;
  retryAfterMs?: number;
}

FUNCTION handleSearchError(error: unknown): ErrorState {
  1. IF error is NetworkError OR !navigator.onLine:
     RETURN {
       type: 'NETWORK_ERROR',
       message: "You're offline. Showing cached results.",
       retryable: true
     }

  2. IF error.name === 'AbortError' OR error.name === 'TimeoutError':
     RETURN {
       type: 'API_TIMEOUT',
       message: "Search is taking longer than expected. Please try again.",
       retryable: true
     }

  3. IF error.status === 429:
     retryAfter = parseInt(error.headers.get('Retry-After') || '60') * 1000
     RETURN {
       type: 'RATE_LIMITED',
       message: "Too many searches. Please wait a moment.",
       retryable: true,
       retryAfterMs: retryAfter
     }

  4. IF error.status >= 500:
     RETURN {
       type: 'SERVER_ERROR',
       message: "Something went wrong. Please try again.",
       retryable: true
     }

  5. DEFAULT:
     RETURN {
       type: 'SERVER_ERROR',
       message: "An unexpected error occurred.",
       retryable: false
     }
```

### 3.4 Graceful Degradation Behavior

| Scenario | Degraded Functionality | Core Functionality Preserved |
|:---------|:-----------------------|:-----------------------------|
| **Redis cache down** | Slower responses (~500ms vs ~10ms) | Full search functionality |
| **Similarity service slow** | Results without similarity indicators | Text search, basic results |
| **Offline mode** | Limited to cached queries only | View cached results, history |
| **Image CDN down** | Placeholder images shown | All data and interactions |
| **Theme storage fails** | Uses system theme | All other functionality |

---

## 4. Component Interfaces

### 4.1 SearchView Component

```typescript
interface SearchViewProps {
  initialMode?: SearchMode;
  onNavigateToResults: (searchParams: SearchParams) => void;
  onError?: (error: ErrorState) => void;
}

interface SearchParams {
  query?: string;
  mode: SearchMode;
  ingredients: SelectedIngredient[];
  filters: TagFilter[];
  macroToggles: MacroToggleState;
}
```

### 4.2 Internal Component Functions

```typescript
// Search Input Management
function handleSearchInputChange(value: string): void;
function handleSearchInputFocus(): void;
function handleSearchInputBlur(): void;
function handleSearchInputKeyDown(event: KeyboardEvent): void;
function clearSearchInput(): void;

// Autocomplete Management
function fetchAutocomplete(query: string): Promise<void>;
function handleSelectAutocompleteItem(item: AutocompleteItem): void;
function handleAutocompleteMouseEnter(index: number): void;
function closeAutocomplete(): void;

// Mode Management
function handleSearchModeChange(mode: SearchMode): void;
function getSearchModeConfig(mode: SearchMode): SearchModeConfig;

// Macro Toggle Management
function handleMacroToggle(macroType: MacroType): void;
function getMacroToggleState(): MacroToggleState;

// Ingredient List Management
function addIngredient(item: AutocompleteItem): void;
function removeIngredient(ingredientId: string): void;
function updateIngredientQuantity(ingredientId: string, quantity: number): void;
function updateIngredientUnit(ingredientId: string, unit: string): void;
function clearIngredients(): void;
function recalculateTotalMacros(): void;

// Tag Filter Management
function addTagFilter(tag: Tag, mode: TagFilterMode): void;
function removeTagFilter(tagId: string): void;
function clearAllFilters(): void;
function toggleFilterPanel(): void;

// Search History Management
function addToSearchHistory(query: string, mode: SearchMode): void;
function selectHistoryItem(item: SearchHistoryItem): void;
function removeHistoryItem(itemId: string): void;
function clearSearchHistory(): void;
function loadSearchHistory(): SearchHistoryItem[];
function persistSearchHistory(): void;

// Offline Management
function handleOnline(): void;
function handleOffline(): void;
function getCachedResults(query: string): AutocompleteItem[] | null;
function cacheResults(query: string, results: AutocompleteItem[]): void;

// Theme Management
function applyTheme(theme: 'light' | 'dark' | 'system'): void;
function getEffectiveTheme(): 'light' | 'dark';
function handleSystemThemeChange(): void;

// Error Handling
function handleSearchError(error: unknown): ErrorState;
function displayError(errorState: ErrorState): void;
function retryLastSearch(): void;
function dismissError(): void;

// Utility Functions
function debounce<T extends (...args: any[]) => void>(fn: T, delay: number): T;
function generateUUID(): string;
function mapErrorToUserMessage(error: unknown): string;
```

### 4.3 API Interface Contracts

```typescript
// GET /api/v1/search/autocomplete
interface AutocompleteRequest {
  query: string;
  mode: SearchMode;
  filters?: TagFilter[];
  limit?: number;  // Default: 8
}

interface AutocompleteResponse {
  items: AutocompleteItem[];
  totalCount: number;
  cached: boolean;
}

// GET /api/v1/tags
interface TagsResponse {
  categoryTags: Tag[];
  functionalityTags: Tag[];
}

// POST /api/v1/search/similar
interface SimilaritySearchRequest {
  ingredients: Array<{
    id: string;
    quantity: number;
    unit: string;
  }>;
  macroToggles: MacroToggleState;
  filters: TagFilter[];
  page?: number;
  limit?: number;
}

interface SimilaritySearchResponse {
  results: SimilarityResult[];
  totalCount: number;
  page: number;
  pageSize: number;
}

interface SimilarityResult {
  item: FoodItem;
  similarityScore: number;
  similarityTier: 'excellent' | 'good' | 'fair' | 'poor';
  indicatorColor: string;
  indicatorImageUrl: string;
  matchingQuantity: number;
}
```

### 4.4 Event Emitters

```typescript
// Events emitted by SearchView for parent components
interface SearchViewEvents {
  'search:submit': (params: SearchParams) => void;
  'search:error': (error: ErrorState) => void;
  'mode:change': (mode: SearchMode) => void;
  'offline:change': (isOnline: boolean) => void;
  'theme:change': (theme: 'light' | 'dark') => void;
}
```

### 4.5 localStorage Keys

| Key | Type | Description |
|:----|:-----|:------------|
| `mealswapp_search_history` | `SearchHistoryItem[]` | Recent search queries |
| `mealswapp_theme` | `'light' \| 'dark' \| 'system'` | User theme preference |
| `mealswapp_cached_queries` | `Record<string, AutocompleteItem[]>` | Offline cache (LRU, max 20) |
| `mealswapp_cached_tags` | `TagsResponse` | Cached available tags |

---

## 5. UI Component Structure

```
SearchView
├── SearchModeSelector
│   ├── ModeButton (Single Item)
│   ├── ModeButton (Recipe)
│   └── ModeButton (Full Diet)
│
├── SearchInputContainer
│   ├── SearchIcon
│   ├── SearchInput
│   ├── ClearButton (visible when query exists)
│   └── LoadingSpinner (visible when loading)
│
├── AutocompleteDropdown (conditional: when open)
│   ├── AutocompleteItem[]
│   │   ├── ItemImage
│   │   ├── ItemName
│   │   ├── ItemTags
│   │   └── ItemMacroSummary
│   └── SearchHistorySection (when query is empty)
│       └── HistoryItem[]
│
├── MacroToggleBar
│   ├── MacroToggle (Protein)
│   ├── MacroToggle (Carbs)
│   └── MacroToggle (Fat)
│
├── TagFilterBar
│   ├── ActiveFilterChip[]
│   ├── AddFilterButton
│   └── FilterPanel (conditional: when open)
│       ├── CategoryTagList
│       └── FunctionalityTagList
│
├── IngredientList (conditional: recipe/diet mode)
│   ├── IngredientRow[]
│   │   ├── IngredientImage
│   │   ├── IngredientName
│   │   ├── QuantityInput
│   │   ├── UnitSelector
│   │   ├── MacroDisplay
│   │   └── RemoveButton
│   └── TotalMacrosSummary
│
├── OfflineBanner (conditional: when offline)
│
└── ErrorDisplay (conditional: when error)
    ├── ErrorMessage
    └── RetryButton
```

---

## 6. Accessibility Requirements

| Element | ARIA Attribute | Keyboard Support |
|:--------|:---------------|:-----------------|
| Search Input | `role="combobox"`, `aria-expanded`, `aria-controls`, `aria-activedescendant` | Standard text input |
| Autocomplete | `role="listbox"`, `aria-label="Search suggestions"` | Arrow keys, Enter, Escape |
| Autocomplete Item | `role="option"`, `aria-selected` | Focusable via arrow keys |
| Mode Buttons | `role="radiogroup"` | Arrow keys within group |
| Macro Toggles | `role="checkbox"`, `aria-checked` | Space to toggle |
| Filter Chips | `role="button"`, `aria-label="Remove filter"` | Enter/Space to remove |
| Ingredient Remove | `aria-label="Remove {ingredient name}"` | Enter/Space |

**Focus Management:**
- On mount: Focus search input
- On autocomplete open: First item receives `aria-activedescendant` (not focus)
- On autocomplete close: Return focus to search input
- On ingredient add: Return focus to search input
- On error: Focus retry button

**Screen Reader Announcements:**
- Autocomplete results count: "N suggestions available"
- Loading state: "Searching..."
- Error state: Announce error message
- Offline state: "You are offline. Showing cached results."

---

## 7. Performance Considerations

| Optimization | Implementation | Impact |
|:-------------|:---------------|:-------|
| Input debouncing | 150ms delay before API call | ~70% reduction in API calls |
| Autocomplete limit | Max 8 items displayed | Consistent render performance |
| Virtual scrolling | Not needed (small list) | N/A |
| Image lazy loading | `loading="lazy"` on images | Faster initial render |
| Memoized calculations | Memo total macros, filtered tags | Prevent unnecessary recalcs |
| localStorage caching | LRU cache for offline, max 20 queries | Offline functionality |

---

## Changelog

### 2026-01-22 (Rev 1.0)

**Added:**
- Initial detailed design document for SearchView component
- Complete type definitions for all state objects
- Step-by-step algorithms for all user interactions
- Error handling specifications
- Component interface contracts
- Accessibility requirements
