# FILE: AutocompleteDropdown.md
**Traceability:** ARCH-001

---

## 1. Data Structures & Types

### 1.1 Component Props Interface

```typescript
interface AutocompleteDropdownProps {
  /** Placeholder text for the input field */
  placeholder?: string;
  /** Maximum number of suggestions to display */
  maxSuggestions?: number;  // Default: 10
  /** Debounce delay in milliseconds */
  debounceMs?: number;  // Default: 150
  /** Callback when user selects an item */
  onSelect: (item: FoodItem) => void;
  /** Callback when input value changes (after debounce) */
  onSearch?: (query: string) => void;
  /** Whether the component is disabled */
  disabled?: boolean;
  /** Minimum characters required before searching */
  minChars?: number;  // Default: 2
  /** Additional CSS classes */
  class?: string;
}
```

### 1.2 Internal State Interface

```typescript
interface AutocompleteState {
  /** Current input value (raw, not debounced) */
  inputValue: string;
  /** List of suggestions from API/cache */
  suggestions: FoodItem[];
  /** Whether dropdown is visible */
  isOpen: boolean;
  /** Index of currently highlighted suggestion (-1 = none) */
  highlightedIndex: number;
  /** Loading state for API requests */
  isLoading: boolean;
  /** Error state */
  error: AutocompleteError | null;
  /** Whether currently offline */
  isOffline: boolean;
  /** Whether results are from cache */
  isFromCache: boolean;
}
```

### 1.3 FoodItem Interface

```typescript
interface FoodItem {
  id: string;
  name: string;
  category: string;
  imageUrl: string | null;
  macros: {
    calories: number;
    protein: number;
    carbohydrates: number;
    fat: number;
  };
  servingSize: string;
}
```

### 1.4 Error Types

```typescript
type AutocompleteError =
  | { type: 'NETWORK_ERROR'; message: string }
  | { type: 'TIMEOUT_ERROR'; message: string }
  | { type: 'API_ERROR'; statusCode: number; message: string }
  | { type: 'NO_RESULTS'; message: string };
```

### 1.5 Cache Entry Structure

```typescript
interface CacheEntry {
  query: string;
  results: FoodItem[];
  timestamp: number;  // Unix timestamp ms
}

interface SearchHistoryEntry {
  query: string;
  timestamp: number;
}
```

### 1.6 Keyboard Navigation Constants

```typescript
const KEYBOARD_KEYS = {
  ARROW_DOWN: 'ArrowDown',
  ARROW_UP: 'ArrowUp',
  ENTER: 'Enter',
  ESCAPE: 'Escape',
  TAB: 'Tab',
} as const;
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Component Initialization

```
1. Initialize Svelte store with default values:
   - inputValue = ''
   - suggestions = []
   - isOpen = false
   - highlightedIndex = -1
   - isLoading = false
   - error = null
   - isOffline = !navigator.onLine
   - isFromCache = false

2. Set up event listeners using $effect:
   - Add 'online' event listener → set isOffline = false
   - Add 'offline' event listener → set isOffline = true
   - Add document click listener for outside click detection

3. Create debounced search function with debounceMs delay (default 150ms)

4. Load recent search history from localStorage for initial suggestions
```

### 2.2 Input Change Handler Algorithm

```
ON inputChange(newValue):
  1. Update inputValue immediately (for responsive UI)

  2. IF newValue.length < minChars:
     a. Clear suggestions
     b. Set isOpen = false
     c. Cancel any pending debounced search
     d. RETURN

  3. Set isLoading = true

  4. Call debouncedSearch(newValue)
```

### 2.3 Debounced Search Algorithm

```
debouncedSearch(query): [debounced by 150ms]
  1. Normalize query:
     - Trim whitespace
     - Convert to lowercase for cache lookup

  2. Check localStorage cache:
     a. Get cached results for normalized query
     b. IF cache hit AND cache age < 5 minutes:
        - Set suggestions = cached results
        - Set isFromCache = true
        - Set isLoading = false
        - Set isOpen = true (if suggestions.length > 0)
        - Continue to step 3 for background refresh (stale-while-revalidate)

  3. IF isOffline:
     a. IF no cache hit:
        - Set error = { type: 'NETWORK_ERROR', message: 'You are offline. Showing cached results only.' }
        - Load any partial matches from recent queries cache
     b. Set isLoading = false
     c. RETURN

  4. Make API request:
     a. Call searchAPI(query) with 10s timeout (per ARCH-010)
     b. ON success:
        - Set suggestions = response.items
        - Set isFromCache = false
        - Set isLoading = false
        - Set error = null
        - Set isOpen = true (if suggestions.length > 0)
        - Update localStorage cache with new results
        - Add query to search history (max 5 recent)
     c. ON timeout:
        - Set error = { type: 'TIMEOUT_ERROR', message: 'Search is taking longer than expected. Please try again.' }
        - Keep any cached results visible
        - Set isLoading = false
     d. ON network error:
        - Set error = { type: 'NETWORK_ERROR', message: 'Unable to reach server.' }
        - Fall back to cached results if available
        - Set isLoading = false
     e. ON API error (4xx/5xx):
        - Set error = { type: 'API_ERROR', statusCode, message }
        - Set isLoading = false
```

### 2.4 Keyboard Navigation Algorithm

```
ON keyDown(event):
  1. IF dropdown is not open:
     a. IF key is ARROW_DOWN and inputValue.length >= minChars:
        - Open dropdown with current suggestions (or history)
        - event.preventDefault()
     b. RETURN

  2. SWITCH event.key:

     CASE ARROW_DOWN:
       a. event.preventDefault()
       b. IF highlightedIndex < suggestions.length - 1:
          - highlightedIndex++
       c. ELSE:
          - highlightedIndex = 0  // Wrap to top
       d. Scroll highlighted item into view

     CASE ARROW_UP:
       a. event.preventDefault()
       b. IF highlightedIndex > 0:
          - highlightedIndex--
       c. ELSE:
          - highlightedIndex = suggestions.length - 1  // Wrap to bottom
       d. Scroll highlighted item into view

     CASE ENTER:
       a. event.preventDefault()
       b. IF highlightedIndex >= 0:
          - selectItem(suggestions[highlightedIndex])
       c. ELSE IF suggestions.length === 1:
          - selectItem(suggestions[0])  // Auto-select single result

     CASE ESCAPE:
       a. event.preventDefault()
       b. closeDropdown()
       c. Clear highlightedIndex

     CASE TAB:
       a. IF highlightedIndex >= 0:
          - selectItem(suggestions[highlightedIndex])
       b. closeDropdown()
       // Allow default tab behavior to proceed
```

### 2.5 Item Selection Algorithm

```
selectItem(item):
  1. Set inputValue = item.name
  2. Call onSelect(item) callback
  3. closeDropdown()
  4. Clear highlightedIndex
  5. Add to recent selections in localStorage
```

### 2.6 Cache Management Algorithm

```
CACHE CONSTANTS:
  MAX_CACHED_QUERIES = 20
  MAX_HISTORY_ENTRIES = 5
  CACHE_TTL_MS = 5 * 60 * 1000  // 5 minutes
  CACHE_KEY = 'mealswapp_search_cache'
  HISTORY_KEY = 'mealswapp_search_history'

saveToCache(query, results):
  1. Get existing cache from localStorage
  2. Add new entry with current timestamp
  3. IF cache.length > MAX_CACHED_QUERIES:
     - Remove oldest entry (LRU eviction)
  4. Save updated cache to localStorage

getFromCache(query):
  1. Get cache from localStorage
  2. Find entry where entry.query === normalizedQuery
  3. IF found AND (now - entry.timestamp) < CACHE_TTL_MS:
     - RETURN entry.results
  4. RETURN null

addToHistory(query):
  1. Get history from localStorage
  2. Remove duplicate if exists
  3. Add new entry at front
  4. IF history.length > MAX_HISTORY_ENTRIES:
     - Remove oldest entry
  5. Save updated history to localStorage

getRecentHistory():
  1. Get history from localStorage
  2. RETURN history entries (max 5)
```

### 2.7 Focus Management Algorithm

```
ON inputFocus:
  1. IF inputValue.length >= minChars AND suggestions.length > 0:
     - Set isOpen = true
  2. ELSE IF inputValue.length === 0:
     - Show recent search history as suggestions
     - Set isOpen = true (if history exists)

ON inputBlur:
  1. Delay close by 150ms to allow click on suggestion
  2. IF not clicking on dropdown:
     - closeDropdown()

ON outsideClick(event):
  1. IF click target is not within component:
     - closeDropdown()
```

---

## 3. State Management & Error Handling

### 3.1 State Transitions

```
                    ┌─────────────────────────────────────────────────────────────┐
                    │                                                             │
                    ▼                                                             │
              ┌──────────┐                                                        │
              │   IDLE   │◄────────────────────────────────────────┐              │
              │          │                                         │              │
              └────┬─────┘                                         │              │
                   │                                               │              │
                   │ input change (length >= minChars)             │              │
                   ▼                                               │              │
              ┌──────────┐                                         │              │
              │ DEBOUNCE │──────150ms────┐                         │              │
              │ PENDING  │               │                         │              │
              └────┬─────┘               │                         │              │
                   │                     │                         │              │
                   │ new input           │ debounce complete       │              │
                   │ (reset timer)       ▼                         │              │
                   │              ┌─────────────┐                  │              │
                   └─────────────►│   LOADING   │                  │              │
                                  │             │                  │              │
                                  └──────┬──────┘                  │              │
                                         │                         │              │
                   ┌─────────────────────┼─────────────────────┐   │              │
                   │                     │                     │   │              │
                   ▼                     ▼                     ▼   │              │
            ┌───────────┐         ┌───────────┐         ┌───────────┐             │
            │  SUCCESS  │         │   ERROR   │         │  TIMEOUT  │             │
            │           │         │           │         │           │             │
            └─────┬─────┘         └─────┬─────┘         └─────┬─────┘             │
                  │                     │                     │                   │
                  │ show results        │ show error +        │ show timeout +    │
                  │                     │ cached results      │ cached results    │
                  ▼                     ▼                     ▼                   │
            ┌─────────────────────────────────────────────────────────┐           │
            │                    DROPDOWN OPEN                        │           │
            │                                                         │           │
            │  - User navigates with arrows                           │           │
            │  - User clicks suggestion → SELECT → IDLE               │           │
            │  - User presses Enter → SELECT → IDLE                   │───────────┘
            │  - User presses Escape → CLOSE → IDLE                   │
            │  - User clicks outside → CLOSE → IDLE                   │
            │  - Input cleared → CLOSE → IDLE                         │
            └─────────────────────────────────────────────────────────┘
```

### 3.2 Error States

| Error Type | Trigger | User Message | Recovery Action |
|:-----------|:--------|:-------------|:----------------|
| `NETWORK_ERROR` | No network connectivity | "Unable to connect. Showing cached results." | Show cached results; auto-retry when back online |
| `TIMEOUT_ERROR` | API request exceeds 10s | "Search is taking longer than expected." | Show cached results; allow manual retry |
| `API_ERROR` (4xx) | Invalid request | "Invalid search query." | Clear input; show suggestions |
| `API_ERROR` (5xx) | Server error | "Something went wrong. Please try again." | Show retry button; use cached results |
| `NO_RESULTS` | Empty API response | "No results found for '{query}'" | Show search tips; suggest similar queries |

### 3.3 Offline Handling

```
IF navigator.onLine === false:
  1. Show offline indicator icon in input
  2. Set isOffline = true in state
  3. On search attempt:
     a. Check localStorage cache for query
     b. IF cache hit:
        - Display cached results
        - Show "Offline - showing cached results" banner
     c. ELSE:
        - Show "No cached results available offline"
        - Display recent search history instead

ON 'online' event:
  1. Set isOffline = false
  2. IF pending search exists:
     - Trigger API search
  3. Remove offline indicator
  4. Optional: Background refresh of displayed cached results
```

### 3.4 Loading State Visual Behavior

```
DURING isLoading === true:
  1. Show spinner icon in input field (replaces search icon)
  2. IF cached results exist:
     - Continue showing cached results with subtle opacity reduction
     - Show "Updating..." indicator
  3. ELSE:
     - Show skeleton loader placeholders in dropdown
     - Display 3-5 skeleton items
  4. Disable keyboard selection (prevent selecting incomplete data)
```

---

## 4. Component Interfaces

### 4.1 Public Component API

```typescript
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { createAutocompleteStore } from './stores/autocomplete';
  import { createQuery } from '@tanstack/svelte-query';

  type Props = {
    placeholder?: string;
    maxSuggestions?: number;
    debounceMs?: number;
    onSelect: (item: FoodItem) => void;
    onSearch?: (query: string) => void;
    disabled?: boolean;
    minChars?: number;
    class?: string;
  };

  let {
    placeholder = 'Search for food...',
    maxSuggestions = 10,
    debounceMs = 150,
    onSelect,
    onSearch,
    disabled = false,
    minChars = 2,
    class: className = ''
  }: Props = $props();

  const dispatch = createEventDispatcher();
  const autocompleteStore = createAutocompleteStore({ debounceMs, minChars, maxSuggestions });
</script>
```

### 4.2 Internal Functions

```typescript
/**
 * Handles input field value changes
 * Updates store and triggers debounced search
 *
 * @param event - Input event
 */
function handleInputChange(event: Event): void;

/**
 * Debounced search function - waits 150ms before executing
 * Checks cache first, then makes API request if needed
 *
 * @param query - Search query string
 */
function debouncedSearch(query: string): void;

/**
 * Handles all keyboard navigation within the component
 * Manages arrow keys, Enter, Escape, and Tab
 *
 * @param event - Keyboard event
 */
function handleKeyDown(event: KeyboardEvent): void;

/**
 * Handles selection of a suggestion item
 * Updates input, calls onSelect callback, closes dropdown
 *
 * @param item - Selected food item
 */
function selectItem(item: FoodItem): void;

/**
 * Closes the dropdown and resets highlight index
 */
function closeDropdown(): void;

/**
 * Opens the dropdown if there are suggestions to show
 */
function openDropdown(): void;

/**
 * Handles click on a suggestion item
 *
 * @param item - Clicked food item
 * @param event - Mouse event
 */
function handleSuggestionClick(item: FoodItem, event: MouseEvent): void;

/**
 * Handles mouse entering a suggestion item
 * Updates highlighted index for hover state
 *
 * @param index - Index of hovered item
 */
function handleSuggestionHover(index: number): void;

/**
 * Scrolls the highlighted suggestion into view within dropdown
 *
 * @param index - Index of item to scroll to
 */
function scrollToHighlighted(index: number): void;
```

### 4.3 Cache Functions

```typescript
/**
 * Saves search results to localStorage cache
 * Implements LRU eviction when cache exceeds 20 entries
 *
 * @param query - Normalized search query
 * @param results - Array of food items
 */
function saveToCache(query: string, results: FoodItem[]): void;

/**
 * Retrieves cached results for a query
 * Returns null if not found or expired (>5 min)
 *
 * @param query - Normalized search query
 * @returns Cached results or null
 */
function getFromCache(query: string): FoodItem[] | null;

/**
 * Adds a search query to recent history
 * Maintains max 5 entries with deduplication
 *
 * @param query - Search query to add
 */
function addToHistory(query: string): void;

/**
 * Retrieves recent search history
 *
 * @returns Array of recent search queries (max 5)
 */
function getRecentHistory(): SearchHistoryEntry[];

/**
 * Clears all cached data (queries and history)
 * Called on user logout or manual cache clear
 */
function clearCache(): void;
```

### 4.4 API Integration

```typescript
/**
 * Makes search API request to backend
 * Includes 10s timeout per ARCH-010
 *
 * @param query - Search query
 * @param signal - AbortController signal for cancellation
 * @returns Promise resolving to food items array
 * @throws NetworkError, TimeoutError, ApiError
 */
async function searchAPI(
  query: string,
  signal?: AbortSignal
): Promise<FoodItem[]>;

// API Endpoint (per ARCH-010):
// GET /api/v1/search?q={query}&limit={maxSuggestions}
//
// Headers:
//   Content-Type: application/json
//   Authorization: Bearer {token}
//
// Response:
//   { items: FoodItem[], total: number }
```

### 4.5 Svelte Stores & TanStack Query

```typescript
// Store for autocomplete state management
function createAutocompleteStore(options: {
  debounceMs: number;
  minChars: number;
  maxSuggestions: number;
}) {
  const { debounceMs, minChars, maxSuggestions } = options;

  const inputValue = $state('');
  const suggestions = $state<FoodItem[]>([]);
  const isOpen = $state(false);
  const highlightedIndex = $state(-1);
  const isLoading = $state(false);
  const error = $state<AutocompleteError | null>(null);
  const isOffline = $state(!navigator.onLine);
  const isFromCache = $state(false);

  return {
    get inputValue() { return inputValue; },
    setInputValue(value: string) { inputValue = value; },
    get suggestions() { return suggestions; },
    setSuggestions(value: FoodItem[]) { suggestions = value; },
    get isOpen() { return isOpen; },
    setIsOpen(value: boolean) { isOpen = value; },
    get highlightedIndex() { return highlightedIndex; },
    setHighlightedIndex(value: number) { highlightedIndex = value; },
    get isLoading() { return isLoading; },
    setIsLoading(value: boolean) { isLoading = value; },
    get error() { return error; },
    setError(value: AutocompleteError | null) { error = value; },
    get isOffline() { return isOffline; },
    setIsOffline(value: boolean) { isOffline = value; },
    get isFromCache() { return isFromCache; },
    setIsFromCache(value: boolean) { isFromCache = value; },
    reset() {
      inputValue = '';
      suggestions = [];
      isOpen = false;
      highlightedIndex = -1;
      isLoading = false;
      error = null;
      isFromCache = false;
    },
  };
}

// TanStack Query for API requests with caching
function createSearchQuery(query: () => string, options: { maxSuggestions: number }) {
  return createQuery({
    queryKey: ['search', query],
    queryFn: async ({ signal }) => {
      return searchAPI(query(), signal);
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 30 * 60 * 1000, // 30 minutes
  });
}

/**
 * Hook for managing debounced value using Svelte's $derived
 *
 * @param value - Value to debounce
 * @param delay - Debounce delay in ms
 * @returns Debounced value
 */
function useDebounce<T>(value: () => T, delay: number): () => T {
  let timeout: ReturnType<typeof setTimeout>;
  const debouncedValue = $state(value());

  $effect(() => {
    clearTimeout(timeout);
    timeout = setTimeout(() => {
      debouncedValue = value();
    }, delay);
    return () => clearTimeout(timeout);
  });

  return () => debouncedValue;
}

/**
 * Hook for managing online/offline status
 *
 * @returns Current online status
 */
function useOnlineStatus(): () => boolean {
  const online = $state(navigator.onLine);

  $effect(() => {
    const handleOnline = () => online = true;
    const handleOffline = () => online = false;

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  });

  return () => online;
}
```

### 4.6 Accessibility (ARIA) Attributes

```svelte
<!-- Input element attributes: -->
<input
  role="combobox"
  aria-expanded={isOpen}
  aria-controls="autocomplete-listbox"
  aria-activedescendant={highlightedIndex >= 0 ? `suggestion-${highlightedIndex}` : undefined}
  aria-autocomplete="list"
  aria-haspopup="listbox"
/>

<!-- Dropdown list attributes: -->
<ul
  id="autocomplete-listbox"
  role="listbox"
  aria-label="Search suggestions"
>
  {#each suggestions as item, index (item.id)}
    <li
      id="suggestion-{index}"
      role="option"
      aria-selected={index === highlightedIndex}
      on:click={() => selectItem(item)}
      on:mouseenter={() => handleSuggestionHover(index)}
    >
      {item.name}
    </li>
  {/each}
</ul>
```
