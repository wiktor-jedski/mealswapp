# Detailed Design: AutocompleteDropdown

**Traceability:** [ARCH-001], [ARCH-002]

---

## 1. Data Structures & Types

### 1.1 Autocomplete Item Types

```typescript
type MatchType = 'exact' | 'fuzzy' | 'partial';

interface AutocompleteItem {
  id: string;
  name: string;
  imageUrl: string | null;
  categoryTags: string[];
  functionalityTags: string[];
  macros: MacroData;
  matchType: MatchType;
  matchScore: number;                    // 0-1 relevance score from backend ranking
  highlightRanges: HighlightRange[];     // Character ranges to highlight in name
}

interface MacroData {
  protein: number;    // per 100g/ml
  carbs: number;
  fat: number;
  calories: number;   // Derived: (protein * 4) + (carbs * 4) + (fat * 9)
}

interface HighlightRange {
  start: number;      // Start index (inclusive)
  end: number;        // End index (exclusive)
}
```

### 1.2 Search History Item Types

```typescript
interface SearchHistoryItem {
  id: string;
  query: string;
  mode: 'single' | 'recipe' | 'diet';
  timestamp: number;
  resultCount: number;
}

const MAX_HISTORY_DISPLAY = 5;
const HISTORY_STORAGE_KEY = 'mealswapp_search_history';
```

### 1.3 Dropdown State Types

```typescript
type DropdownSection = 'results' | 'history' | 'empty' | 'loading' | 'error';

interface AutocompleteDropdownState {
  isOpen: boolean;
  activeSection: DropdownSection;
  items: AutocompleteItem[];
  historyItems: SearchHistoryItem[];
  highlightedIndex: number;             // -1 = none, 0+ = item index
  highlightedSection: 'results' | 'history' | null;
  totalCount: number;                    // Total matches from backend (may exceed displayed)
  isLoading: boolean;
  error: DropdownError | null;
  query: string;                         // Current search query (for display)
  hasMoreResults: boolean;               // true if totalCount > items.length
}

const INITIAL_DROPDOWN_STATE: AutocompleteDropdownState = {
  isOpen: false,
  activeSection: 'empty',
  items: [],
  historyItems: [],
  highlightedIndex: -1,
  highlightedSection: null,
  totalCount: 0,
  isLoading: false,
  error: null,
  query: '',
  hasMoreResults: false
};

const MAX_VISIBLE_ITEMS = 8;
const ITEM_HEIGHT_PX = 72;              // Height of each autocomplete item
const MAX_DROPDOWN_HEIGHT_PX = 480;     // Maximum dropdown height (6.5 items visible)
```

### 1.4 Error Types

```typescript
type DropdownErrorType =
  | 'NETWORK_ERROR'
  | 'TIMEOUT'
  | 'NO_RESULTS'
  | 'RATE_LIMITED'
  | 'SERVER_ERROR';

interface DropdownError {
  type: DropdownErrorType;
  message: string;
  retryable: boolean;
  retryAfterMs?: number;
}

const ERROR_MESSAGES: Record<DropdownErrorType, string> = {
  NETWORK_ERROR: "You're offline. Check your connection.",
  TIMEOUT: "Search timed out. Please try again.",
  NO_RESULTS: "No items found for \"{query}\".",
  RATE_LIMITED: "Too many searches. Please wait.",
  SERVER_ERROR: "Something went wrong. Please try again."
};
```

### 1.5 Positioning Types

```typescript
type DropdownPosition = 'below' | 'above';

interface DropdownPositionConfig {
  position: DropdownPosition;
  top: number;
  left: number;
  width: number;
  maxHeight: number;
}

interface ViewportBounds {
  viewportHeight: number;
  inputRect: DOMRect;
  spaceBelow: number;
  spaceAbove: number;
}
```

### 1.6 Keyboard Navigation Types

```typescript
type NavigationDirection = 'up' | 'down' | 'first' | 'last';

interface KeyboardAction {
  key: string;
  action: 'navigate' | 'select' | 'close' | 'clear' | 'none';
  direction?: NavigationDirection;
}

const KEYBOARD_MAPPINGS: Record<string, KeyboardAction> = {
  'ArrowDown': { key: 'ArrowDown', action: 'navigate', direction: 'down' },
  'ArrowUp': { key: 'ArrowUp', action: 'navigate', direction: 'up' },
  'Enter': { key: 'Enter', action: 'select' },
  'Tab': { key: 'Tab', action: 'select' },
  'Escape': { key: 'Escape', action: 'close' },
  'Home': { key: 'Home', action: 'navigate', direction: 'first' },
  'End': { key: 'End', action: 'navigate', direction: 'last' }
};
```

### 1.7 Callback Types

```typescript
interface AutocompleteDropdownCallbacks {
  onSelect: (item: AutocompleteItem) => void;
  onSelectHistory: (historyItem: SearchHistoryItem) => void;
  onClose: () => void;
  onClearHistory: () => void;
  onRemoveHistoryItem: (itemId: string) => void;
  onViewAllResults: () => void;
  onRetry: () => void;
}
```

### 1.8 Animation Types

```typescript
interface AnimationConfig {
  duration: number;
  easing: string;
}

const DROPDOWN_ANIMATIONS: Record<string, AnimationConfig> = {
  open: { duration: 150, easing: 'ease-out' },
  close: { duration: 100, easing: 'ease-in' },
  highlightChange: { duration: 50, easing: 'ease-out' }
};
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Dropdown Open/Close Logic

```
FUNCTION openDropdown(query: string, items: AutocompleteItem[], historyItems: SearchHistoryItem[]):
  1. Determine active section based on content:
     IF isLoading:
       state.activeSection = 'loading'
     ELSE IF error !== null:
       state.activeSection = 'error'
     ELSE IF query.length === 0 AND historyItems.length > 0:
       state.activeSection = 'history'
       state.items = []
       state.historyItems = historyItems.slice(0, MAX_HISTORY_DISPLAY)
     ELSE IF query.length > 0 AND items.length > 0:
       state.activeSection = 'results'
       state.items = items.slice(0, MAX_VISIBLE_ITEMS)
       state.hasMoreResults = totalCount > MAX_VISIBLE_ITEMS
     ELSE IF query.length > 0 AND items.length === 0:
       state.activeSection = 'empty'
     ELSE:
       state.activeSection = 'empty'
       state.historyItems = []

  2. Calculate dropdown position:
     positionConfig = calculateDropdownPosition()
     applyPositionStyles(positionConfig)

  3. Reset highlight state:
     state.highlightedIndex = -1
     state.highlightedSection = null

  4. Open dropdown:
     state.isOpen = true
     state.query = query

  5. Animate open:
     applyOpenAnimation()

  6. Update ARIA attributes:
     updateAriaExpanded(true)
     announceToScreenReader(getAnnouncementText())

FUNCTION closeDropdown():
  1. IF state.isOpen === false:
     RETURN (already closed)

  2. Animate close:
     applyCloseAnimation()

  3. After animation completes:
     state.isOpen = false
     state.highlightedIndex = -1
     state.highlightedSection = null

  4. Update ARIA attributes:
     updateAriaExpanded(false)

  5. Notify parent:
     callbacks.onClose()
```

### 2.2 Dropdown Position Calculation

```
FUNCTION calculateDropdownPosition(): DropdownPositionConfig
  1. Get viewport and input element bounds:
     viewportHeight = window.innerHeight
     inputRect = inputElement.getBoundingClientRect()

  2. Calculate available space:
     spaceBelow = viewportHeight - inputRect.bottom - 16  // 16px margin from viewport edge
     spaceAbove = inputRect.top - 16

  3. Calculate required height:
     itemCount = Math.max(state.items.length, state.historyItems.length)
     IF state.activeSection === 'loading':
       requiredHeight = 120  // Loading spinner height
     ELSE IF state.activeSection === 'error':
       requiredHeight = 160  // Error message height
     ELSE IF state.activeSection === 'empty':
       requiredHeight = 120  // Empty state message height
     ELSE:
       contentHeight = itemCount * ITEM_HEIGHT_PX
       headerHeight = state.activeSection === 'history' ? 40 : 0  // "Recent Searches" header
       footerHeight = state.hasMoreResults ? 48 : 0              // "View all results" footer
       requiredHeight = Math.min(contentHeight + headerHeight + footerHeight, MAX_DROPDOWN_HEIGHT_PX)

  4. Determine position (prefer below, flip if insufficient space):
     IF spaceBelow >= requiredHeight:
       position = 'below'
       top = inputRect.bottom + 4  // 4px gap
       maxHeight = Math.min(requiredHeight, spaceBelow)
     ELSE IF spaceAbove >= requiredHeight:
       position = 'above'
       top = inputRect.top - requiredHeight - 4
       maxHeight = Math.min(requiredHeight, spaceAbove)
     ELSE:
       // Not enough space either direction, use larger space
       IF spaceBelow >= spaceAbove:
         position = 'below'
         top = inputRect.bottom + 4
         maxHeight = spaceBelow
       ELSE:
         position = 'above'
         maxHeight = spaceAbove
         top = inputRect.top - maxHeight - 4

  5. Calculate horizontal positioning:
     left = inputRect.left
     width = inputRect.width

  6. RETURN { position, top, left, width, maxHeight }

FUNCTION applyPositionStyles(config: DropdownPositionConfig):
  dropdownElement.style.top = `${config.top}px`
  dropdownElement.style.left = `${config.left}px`
  dropdownElement.style.width = `${config.width}px`
  dropdownElement.style.maxHeight = `${config.maxHeight}px`
  dropdownElement.setAttribute('data-position', config.position)
```

### 2.3 Keyboard Navigation

```
FUNCTION handleKeyDown(event: KeyboardEvent):
  1. Get keyboard action mapping:
     action = KEYBOARD_MAPPINGS[event.key]
     IF action is undefined:
       RETURN  // Not a handled key

  2. Handle based on action type:
     CASE action.action === 'navigate':
       event.preventDefault()
       handleNavigate(action.direction)

     CASE action.action === 'select':
       IF state.highlightedIndex >= 0:
         event.preventDefault()
         handleSelect()
       ELSE IF event.key === 'Tab':
         // Allow default tab behavior if nothing highlighted
         closeDropdown()

     CASE action.action === 'close':
       event.preventDefault()
       closeDropdown()

FUNCTION handleNavigate(direction: NavigationDirection):
  1. Determine navigable items:
     IF state.activeSection === 'results':
       itemCount = state.items.length
       section = 'results'
     ELSE IF state.activeSection === 'history':
       itemCount = state.historyItems.length
       section = 'history'
     ELSE:
       RETURN  // No items to navigate

  2. IF itemCount === 0:
     RETURN

  3. Calculate new index based on direction:
     currentIndex = state.highlightedIndex

     CASE direction === 'down':
       IF currentIndex === -1:
         newIndex = 0
       ELSE IF currentIndex >= itemCount - 1:
         newIndex = 0  // Wrap to beginning
       ELSE:
         newIndex = currentIndex + 1

     CASE direction === 'up':
       IF currentIndex === -1:
         newIndex = itemCount - 1
       ELSE IF currentIndex === 0:
         newIndex = itemCount - 1  // Wrap to end
       ELSE:
         newIndex = currentIndex - 1

     CASE direction === 'first':
       newIndex = 0

     CASE direction === 'last':
       newIndex = itemCount - 1

  4. Update state:
     state.highlightedIndex = newIndex
     state.highlightedSection = section

  5. Scroll highlighted item into view:
     scrollItemIntoView(newIndex)

  6. Update ARIA active descendant:
     updateAriaActiveDescendant(newIndex)

FUNCTION scrollItemIntoView(index: number):
  1. Get item element:
     itemElement = dropdownElement.querySelector(`[data-index="${index}"]`)
     IF itemElement is null:
       RETURN

  2. Get scroll container:
     scrollContainer = dropdownElement.querySelector('.dropdown-scroll-container')

  3. Calculate visibility:
     containerRect = scrollContainer.getBoundingClientRect()
     itemRect = itemElement.getBoundingClientRect()

  4. Scroll if needed:
     IF itemRect.bottom > containerRect.bottom:
       // Item is below visible area
       scrollContainer.scrollTop += itemRect.bottom - containerRect.bottom + 8
     ELSE IF itemRect.top < containerRect.top:
       // Item is above visible area
       scrollContainer.scrollTop -= containerRect.top - itemRect.top + 8
```

### 2.4 Item Selection

```
FUNCTION handleSelect():
  1. Validate selection state:
     IF state.highlightedIndex < 0:
       RETURN
     IF state.highlightedSection === null:
       RETURN

  2. Get selected item:
     IF state.highlightedSection === 'results':
       selectedItem = state.items[state.highlightedIndex]
       IF selectedItem is undefined:
         RETURN
       callbacks.onSelect(selectedItem)

     ELSE IF state.highlightedSection === 'history':
       selectedHistoryItem = state.historyItems[state.highlightedIndex]
       IF selectedHistoryItem is undefined:
         RETURN
       callbacks.onSelectHistory(selectedHistoryItem)

  3. Close dropdown:
     closeDropdown()

FUNCTION handleItemClick(index: number, section: 'results' | 'history'):
  1. Update highlight state (for visual feedback):
     state.highlightedIndex = index
     state.highlightedSection = section

  2. Call appropriate selection handler:
     IF section === 'results':
       callbacks.onSelect(state.items[index])
     ELSE:
       callbacks.onSelectHistory(state.historyItems[index])

  3. Close dropdown:
     closeDropdown()
```

### 2.5 Mouse Interaction

```
FUNCTION handleItemMouseEnter(index: number, section: 'results' | 'history'):
  1. Update highlight state:
     state.highlightedIndex = index
     state.highlightedSection = section

  2. Update ARIA active descendant:
     updateAriaActiveDescendant(index)

FUNCTION handleItemMouseLeave():
  1. Clear highlight if mouse leaves item area:
     // Note: Only clear if mouse leaves dropdown entirely
     // This prevents flicker when moving between items

FUNCTION handleDropdownMouseLeave():
  1. Clear highlight state:
     state.highlightedIndex = -1
     state.highlightedSection = null

  2. Clear ARIA active descendant:
     clearAriaActiveDescendant()

FUNCTION handleClickOutside(event: MouseEvent):
  1. Check if click is outside dropdown and input:
     dropdownContains = dropdownElement.contains(event.target)
     inputContains = inputElement.contains(event.target)

  2. IF NOT dropdownContains AND NOT inputContains:
     closeDropdown()
```

### 2.6 Text Highlighting in Results

```
FUNCTION renderHighlightedName(name: string, highlightRanges: HighlightRange[]): HTMLElement[]
  1. IF highlightRanges is empty or null:
     RETURN [createTextNode(name)]

  2. Sort ranges by start index:
     sortedRanges = highlightRanges.sort((a, b) => a.start - b.start)

  3. Build segments:
     segments = []
     currentIndex = 0

     FOR each range IN sortedRanges:
       // Add non-highlighted text before this range
       IF range.start > currentIndex:
         segments.push({
           text: name.substring(currentIndex, range.start),
           highlighted: false
         })

       // Add highlighted text
       segments.push({
         text: name.substring(range.start, range.end),
         highlighted: true
       })

       currentIndex = range.end

     // Add remaining non-highlighted text
     IF currentIndex < name.length:
       segments.push({
         text: name.substring(currentIndex),
         highlighted: false
       })

  4. Convert segments to elements:
     elements = []
     FOR each segment IN segments:
       IF segment.highlighted:
         element = createHighlightSpan(segment.text)
       ELSE:
         element = createTextNode(segment.text)
       elements.push(element)

     RETURN elements

FUNCTION createHighlightSpan(text: string): HTMLElement
  span = document.createElement('span')
  span.className = 'autocomplete-highlight'
  span.textContent = text
  RETURN span
```

### 2.7 History Section Handling

```
FUNCTION handleClearHistory():
  1. Show confirmation (inline, not modal):
     // Display "Clear all?" with Yes/Cancel options

  2. IF user confirms:
     2.1. Clear history items:
          state.historyItems = []
     2.2. Update localStorage:
          localStorage.removeItem(HISTORY_STORAGE_KEY)
     2.3. Update active section:
          state.activeSection = 'empty'
     2.4. Notify parent:
          callbacks.onClearHistory()
     2.5. Announce to screen reader:
          announceToScreenReader("Search history cleared")

FUNCTION handleRemoveHistoryItem(itemId: string):
  1. Find and remove item:
     index = state.historyItems.findIndex(item => item.id === itemId)
     IF index === -1:
       RETURN

  2. Remove from state:
     state.historyItems.splice(index, 1)

  3. Update localStorage:
     storedHistory = JSON.parse(localStorage.getItem(HISTORY_STORAGE_KEY) || '[]')
     updatedHistory = storedHistory.filter(item => item.id !== itemId)
     localStorage.setItem(HISTORY_STORAGE_KEY, JSON.stringify(updatedHistory))

  4. Notify parent:
     callbacks.onRemoveHistoryItem(itemId)

  5. Update active section if empty:
     IF state.historyItems.length === 0:
       state.activeSection = 'empty'

  6. Adjust highlight if needed:
     IF state.highlightedIndex >= state.historyItems.length:
       state.highlightedIndex = Math.max(0, state.historyItems.length - 1)
       IF state.historyItems.length === 0:
         state.highlightedIndex = -1

  7. Announce removal:
     announceToScreenReader("Search removed from history")
```

### 2.8 View All Results Handling

```
FUNCTION handleViewAllResults():
  1. Close dropdown:
     closeDropdown()

  2. Trigger full search:
     callbacks.onViewAllResults()

  3. Announce to screen reader:
     announceToScreenReader(`Viewing all ${state.totalCount} results`)
```

### 2.9 Error State Handling

```
FUNCTION displayError(errorType: DropdownErrorType, query: string):
  1. Create error object:
     errorMessage = ERROR_MESSAGES[errorType]
     IF errorType === 'NO_RESULTS':
       errorMessage = errorMessage.replace('{query}', query)

     state.error = {
       type: errorType,
       message: errorMessage,
       retryable: errorType !== 'NO_RESULTS',
       retryAfterMs: errorType === 'RATE_LIMITED' ? 60000 : undefined
     }

  2. Update section:
     state.activeSection = 'error'

  3. Open dropdown to show error:
     state.isOpen = true

  4. Announce error:
     announceToScreenReader(errorMessage)

FUNCTION handleRetry():
  1. Clear error state:
     state.error = null

  2. Show loading state:
     state.activeSection = 'loading'
     state.isLoading = true

  3. Trigger retry:
     callbacks.onRetry()
```

### 2.10 Loading State Handling

```
FUNCTION showLoading():
  1. Update state:
     state.isLoading = true
     state.activeSection = 'loading'
     state.error = null

  2. Ensure dropdown is open:
     IF NOT state.isOpen:
       state.isOpen = true
       applyOpenAnimation()

FUNCTION hideLoading():
  1. Update state:
     state.isLoading = false

  2. Section will be updated when results arrive
```

---

## 3. State Management & Error Handling

### 3.1 State Transitions Diagram

```
                                    ┌─────────────────┐
                                    │     CLOSED      │
                                    │                 │
                                    └────────┬────────┘
                                             │
                                             │ User focuses input OR
                                             │ Types query
                                             ▼
                           ┌─────────────────────────────────┐
                           │              OPEN               │
                           │                                 │
                           └───────────────┬─────────────────┘
                                           │
              ┌────────────────────────────┼────────────────────────────┐
              │                            │                            │
              ▼                            ▼                            ▼
     ┌────────────────┐          ┌────────────────┐          ┌────────────────┐
     │    LOADING     │          │    RESULTS     │          │    HISTORY     │
     │  (Spinner)     │          │  (Item List)   │          │  (When empty   │
     └───────┬────────┘          └───────┬────────┘          │    query)      │
             │                           │                    └───────┬────────┘
             │                           │                            │
             ▼                           │                            │
     ┌────────────────┐                  │                            │
     │    ERROR       │<─────────────────┘                            │
     │  (Message +    │                                               │
     │   Retry)       │                                               │
     └───────┬────────┘                                               │
             │                                                        │
             │                           ┌────────────────┐           │
             └──────────────────────────>│     EMPTY      │<──────────┘
                                         │  (No results)  │
                                         └───────┬────────┘
                                                 │
                                                 │ Escape / Click outside /
                                                 │ Select item
                                                 ▼
                                         ┌────────────────┐
                                         │     CLOSED     │
                                         │                │
                                         └────────────────┘
```

### 3.2 Highlight State Transitions

```
                    ┌─────────────────┐
                    │  NO HIGHLIGHT   │
                    │  (index: -1)    │
                    └────────┬────────┘
                             │
           Arrow Down        │         Mouse Enter
           Arrow Up          │
                             ▼
                    ┌─────────────────┐
                    │   HIGHLIGHTED   │
           ┌───────>│  (index: N)     │<───────┐
           │        └────────┬────────┘        │
           │                 │                 │
   Arrow   │    Enter/Tab    │      Arrow      │  Mouse
   Keys    │    Click        │      Keys       │  Enter
           │                 ▼                 │
           │        ┌─────────────────┐        │
           │        │    SELECTED     │        │
           │        │  (Callback +    │        │
           │        │   Close)        │        │
           │        └─────────────────┘        │
           │                                   │
           └───────────────────────────────────┘
                    Navigate between items
```

### 3.3 Error States

| Error State | Trigger | User Message | Visual | Recovery Action |
|:------------|:--------|:-------------|:-------|:----------------|
| **NETWORK_ERROR** | Fetch fails, navigator.onLine = false | "You're offline. Check your connection." | Offline icon, muted text | Auto-retry when online; show cached history if available |
| **TIMEOUT** | API response > 10s | "Search timed out. Please try again." | Clock icon, retry button | Manual retry button |
| **NO_RESULTS** | API returns empty array | "No items found for "{query}"." | Search icon with X, muted text | Suggest clearing filters, trying different terms |
| **RATE_LIMITED** | API returns 429 | "Too many searches. Please wait." | Timer icon, countdown | Auto-retry after delay; show countdown |
| **SERVER_ERROR** | API returns 5xx | "Something went wrong. Please try again." | Warning icon, retry button | Manual retry button |

### 3.4 Error Handling Implementation

```typescript
FUNCTION handleDropdownError(error: unknown, query: string): DropdownError
  1. IF !navigator.onLine OR error instanceof TypeError (network):
     RETURN {
       type: 'NETWORK_ERROR',
       message: "You're offline. Check your connection.",
       retryable: true
     }

  2. IF error.name === 'AbortError' OR error.name === 'TimeoutError':
     RETURN {
       type: 'TIMEOUT',
       message: "Search timed out. Please try again.",
       retryable: true
     }

  3. IF error.status === 429:
     retryAfter = parseInt(error.headers?.get('Retry-After') || '60') * 1000
     RETURN {
       type: 'RATE_LIMITED',
       message: "Too many searches. Please wait.",
       retryable: true,
       retryAfterMs: retryAfter
     }

  4. IF error.status >= 500:
     RETURN {
       type: 'SERVER_ERROR',
       message: "Something went wrong. Please try again.",
       retryable: true
     }

  5. IF error.status === 200 AND response.items.length === 0:
     RETURN {
       type: 'NO_RESULTS',
       message: `No items found for "${query}".`,
       retryable: false
     }

  6. DEFAULT:
     RETURN {
       type: 'SERVER_ERROR',
       message: "An unexpected error occurred.",
       retryable: true
     }
```

### 3.5 Graceful Degradation

| Scenario | Degraded Functionality | Core Functionality Preserved |
|:---------|:-----------------------|:-----------------------------|
| **Offline mode** | Cannot fetch new results | Show search history, cached queries |
| **Image CDN down** | Show placeholder images | All item data, selection, navigation |
| **Slow connection** | Extended loading state | Results eventually display; history available |
| **localStorage full** | History not persisted | Current session history works |
| **CSS animation fails** | Instant open/close | Full keyboard/mouse functionality |

---

## 4. Component Interfaces

### 4.1 AutocompleteDropdown Props

```typescript
interface AutocompleteDropdownProps {
  // Required props
  inputElement: HTMLInputElement;        // Reference to search input for positioning
  isOpen: boolean;                       // Controlled open state
  query: string;                         // Current search query
  items: AutocompleteItem[];             // Search results
  isLoading: boolean;                    // Loading state

  // Optional props
  historyItems?: SearchHistoryItem[];    // Search history (shown when query empty)
  error?: DropdownError | null;          // Error state
  totalCount?: number;                   // Total results (for "View all" footer)
  highlightedIndex?: number;             // Controlled highlight (for keyboard nav)
  maxItems?: number;                     // Override MAX_VISIBLE_ITEMS

  // Callbacks
  onSelect: (item: AutocompleteItem) => void;
  onSelectHistory: (item: SearchHistoryItem) => void;
  onClose: () => void;
  onHighlightChange?: (index: number) => void;
  onClearHistory?: () => void;
  onRemoveHistoryItem?: (itemId: string) => void;
  onViewAllResults?: () => void;
  onRetry?: () => void;

  // Accessibility
  inputId: string;                       // ID of input for ARIA relationships
  listboxId?: string;                    // Custom ID for listbox

  // Styling
  className?: string;
  position?: 'auto' | 'below' | 'above'; // Position override
}
```

### 4.2 Internal Component Functions

```typescript
// Lifecycle
function mountDropdown(): void;
function unmountDropdown(): void;
function updateDropdown(props: AutocompleteDropdownProps): void;

// Position Management
function calculateDropdownPosition(): DropdownPositionConfig;
function applyPositionStyles(config: DropdownPositionConfig): void;
function handleWindowResize(): void;
function handleWindowScroll(): void;

// Open/Close
function openDropdown(query: string, items: AutocompleteItem[], historyItems: SearchHistoryItem[]): void;
function closeDropdown(): void;
function applyOpenAnimation(): void;
function applyCloseAnimation(): void;

// Keyboard Navigation
function handleKeyDown(event: KeyboardEvent): void;
function handleNavigate(direction: NavigationDirection): void;
function handleSelect(): void;
function scrollItemIntoView(index: number): void;

// Mouse Interaction
function handleItemClick(index: number, section: 'results' | 'history'): void;
function handleItemMouseEnter(index: number, section: 'results' | 'history'): void;
function handleItemMouseLeave(): void;
function handleDropdownMouseLeave(): void;
function handleClickOutside(event: MouseEvent): void;

// Content Rendering
function renderResultsSection(): HTMLElement;
function renderHistorySection(): HTMLElement;
function renderLoadingSection(): HTMLElement;
function renderErrorSection(): HTMLElement;
function renderEmptySection(): HTMLElement;
function renderHighlightedName(name: string, ranges: HighlightRange[]): HTMLElement[];
function renderMacroSummary(macros: MacroData): HTMLElement;
function renderCategoryTags(tags: string[], maxVisible: number): HTMLElement;

// History Management
function handleClearHistory(): void;
function handleRemoveHistoryItem(itemId: string): void;

// Error Handling
function displayError(errorType: DropdownErrorType, query: string): void;
function handleRetry(): void;

// Loading State
function showLoading(): void;
function hideLoading(): void;

// Accessibility
function updateAriaExpanded(expanded: boolean): void;
function updateAriaActiveDescendant(index: number): void;
function clearAriaActiveDescendant(): void;
function announceToScreenReader(message: string): void;
function getAnnouncementText(): string;

// Cleanup
function removeEventListeners(): void;
function clearTimers(): void;
```

### 4.3 Event Handling Interface

```typescript
// Events the dropdown listens to
interface DropdownEventListeners {
  'keydown': (event: KeyboardEvent) => void;      // On input element
  'click': (event: MouseEvent) => void;           // On document (click outside)
  'resize': () => void;                           // On window
  'scroll': () => void;                           // On window
  'online': () => void;                           // On window
  'offline': () => void;                          // On window
}

// Internal events
interface DropdownInternalEvents {
  'item:highlight': (index: number, section: string) => void;
  'item:select': (item: AutocompleteItem | SearchHistoryItem) => void;
  'dropdown:open': () => void;
  'dropdown:close': () => void;
  'error:display': (error: DropdownError) => void;
  'error:retry': () => void;
}
```

### 4.4 ARIA Attributes Specification

```typescript
// Input element attributes (set by parent)
interface InputAriaAttributes {
  'role': 'combobox';
  'aria-expanded': 'true' | 'false';
  'aria-haspopup': 'listbox';
  'aria-controls': string;                    // listbox ID
  'aria-activedescendant': string | undefined; // highlighted item ID
  'aria-autocomplete': 'list';
}

// Dropdown container attributes
interface DropdownAriaAttributes {
  'role': 'listbox';
  'id': string;
  'aria-label': 'Search suggestions';
  'aria-busy': 'true' | 'false';             // During loading
}

// Item attributes
interface ItemAriaAttributes {
  'role': 'option';
  'id': string;                               // For aria-activedescendant
  'aria-selected': 'true' | 'false';
  'aria-posinset': number;                    // Position in set
  'aria-setsize': number;                     // Total items
}

// Group attributes (for history section)
interface GroupAriaAttributes {
  'role': 'group';
  'aria-labelledby': string;                  // Header ID
}
```

---

## 5. UI Component Structure

```
AutocompleteDropdown
├── DropdownContainer
│   │   [role="listbox", aria-label="Search suggestions"]
│   │   [data-position="below" | "above"]
│   │
│   ├── ScrollContainer
│   │   │   [class="dropdown-scroll-container"]
│   │   │
│   │   ├── ResultsSection (when activeSection === 'results')
│   │   │   └── ResultItem[] (for each item)
│   │   │       ├── ItemImage
│   │   │       │   ├── Image (if imageUrl)
│   │   │       │   └── Placeholder (if no imageUrl)
│   │   │       ├── ItemContent
│   │   │       │   ├── ItemName (with highlight spans)
│   │   │       │   ├── CategoryTags
│   │   │       │   │   └── TagChip[] (max 3)
│   │   │       │   └── MacroSummary
│   │   │       │       ├── ProteinValue
│   │   │       │       ├── CarbsValue
│   │   │       │       └── FatValue
│   │   │       └── MatchIndicator (exact/fuzzy/partial)
│   │   │
│   │   ├── HistorySection (when activeSection === 'history')
│   │   │   ├── SectionHeader
│   │   │   │   ├── HeaderText ("Recent Searches")
│   │   │   │   └── ClearButton
│   │   │   └── HistoryItem[] (for each history item)
│   │   │       ├── HistoryIcon
│   │   │       ├── HistoryContent
│   │   │       │   ├── QueryText
│   │   │       │   └── ModeLabel + Timestamp
│   │   │       └── RemoveButton
│   │   │
│   │   ├── LoadingSection (when activeSection === 'loading')
│   │   │   ├── LoadingSpinner
│   │   │   └── LoadingText ("Searching...")
│   │   │
│   │   ├── ErrorSection (when activeSection === 'error')
│   │   │   ├── ErrorIcon
│   │   │   ├── ErrorMessage
│   │   │   └── RetryButton (if retryable)
│   │   │
│   │   └── EmptySection (when activeSection === 'empty')
│   │       ├── EmptyIcon
│   │       └── EmptyMessage
│   │
│   └── Footer (when hasMoreResults)
│       └── ViewAllButton
│           ├── ButtonText ("View all {totalCount} results")
│           └── ArrowIcon
│
└── LiveRegion (for screen reader announcements)
    [role="status", aria-live="polite", aria-atomic="true"]
```

---

## 6. Accessibility Requirements

### 6.1 ARIA Implementation

| Element | ARIA Attributes | Purpose |
|:--------|:----------------|:--------|
| Search Input | `role="combobox"`, `aria-expanded`, `aria-controls`, `aria-activedescendant`, `aria-autocomplete="list"` | Indicates expandable search with suggestions |
| Dropdown | `role="listbox"`, `aria-label="Search suggestions"`, `aria-busy` | Container for selectable options |
| Result Item | `role="option"`, `aria-selected`, `aria-posinset`, `aria-setsize` | Selectable suggestion option |
| History Section | `role="group"`, `aria-labelledby` | Groups related history items |
| History Header | `id` (for group labelledby) | Labels the history group |
| Clear History Button | `aria-label="Clear search history"` | Describes action |
| Remove History Item | `aria-label="Remove {query} from history"` | Describes action for specific item |
| Loading State | `aria-busy="true"` on listbox | Indicates content loading |
| Error Message | `role="alert"` | Announces errors immediately |
| Live Region | `role="status"`, `aria-live="polite"` | Non-intrusive announcements |

### 6.2 Keyboard Support

| Key | Dropdown Closed | Dropdown Open |
|:----|:----------------|:--------------|
| **Arrow Down** | Open dropdown, highlight first item | Move highlight down (wrap to first) |
| **Arrow Up** | Open dropdown, highlight last item | Move highlight up (wrap to last) |
| **Enter** | N/A | Select highlighted item; or submit search if no highlight |
| **Tab** | N/A | Select highlighted item and move focus; or close and move focus |
| **Escape** | N/A | Close dropdown, return focus to input |
| **Home** | N/A | Highlight first item |
| **End** | N/A | Highlight last item |
| **Any printable** | Type in input | Type in input (dropdown updates) |

### 6.3 Focus Management

```
Focus Flow:
  1. Focus stays on input element at all times while dropdown is open
  2. Keyboard navigation updates aria-activedescendant (not actual focus)
  3. On item selection: focus remains on input (cleared for next search)
  4. On Escape: focus remains on input
  5. On Tab with selection: focus moves to next focusable element
  6. Clear/Remove buttons in history are mouse-only (not in tab order)
     - These actions accessible via keyboard shortcuts if needed
```

### 6.4 Screen Reader Announcements

| Event | Announcement |
|:------|:-------------|
| Dropdown opens with results | "{N} suggestions available" |
| Dropdown opens with history | "{N} recent searches" |
| Dropdown opens with no results | "No results found for {query}" |
| Highlight changes | Item name is read (via activedescendant) |
| Item selected | "{Item name} selected" |
| Error displayed | Error message |
| History cleared | "Search history cleared" |
| History item removed | "Search removed from history" |
| Loading starts | "Searching..." |
| Rate limited | "Too many searches. Please wait {N} seconds." |

### 6.5 Color Contrast

| Element | Foreground | Background | Contrast Ratio | WCAG Level |
|:--------|:-----------|:-----------|:---------------|:-----------|
| Item name | `--text-primary` | `--bg-surface` | 12.63:1 (light), 15.04:1 (dark) | AAA |
| Highlight text | `--color-primary` | `--color-secondary` | 4.91:1 (light), 5.23:1 (dark) | AA |
| Muted text (macros) | `--text-muted` | `--bg-surface` | 4.69:1 (light), 4.52:1 (dark) | AA |
| Error text | `--color-error` | `--bg-surface` | 5.12:1 (light), 4.89:1 (dark) | AA |
| Highlighted item bg | `--text-primary` | `--hover-bg` | 11.24:1 (light), 12.31:1 (dark) | AAA |

---

## 7. Performance Considerations

### 7.1 Optimizations

| Optimization | Implementation | Impact |
|:-------------|:---------------|:-------|
| **Limited item rendering** | Max 8 items rendered, virtual scroll not needed | Consistent <16ms render |
| **Debounced position updates** | 100ms debounce on resize/scroll handlers | Prevent layout thrashing |
| **CSS containment** | `contain: layout style paint` on dropdown | Isolate repaints |
| **Will-change hint** | `will-change: transform, opacity` on dropdown | Smoother open/close |
| **Image lazy loading** | `loading="lazy"` on item images | Faster initial render |
| **Memoized highlight ranges** | Cache highlight calculation per query | Avoid re-computation |
| **Event delegation** | Single click handler on container | Fewer event listeners |
| **RAF for scroll** | requestAnimationFrame for scrollIntoView | Smooth scrolling |

### 7.2 Memory Management

```typescript
// Cleanup on unmount
function cleanup():
  1. Remove all event listeners:
     document.removeEventListener('click', handleClickOutside)
     window.removeEventListener('resize', handleWindowResize)
     window.removeEventListener('scroll', handleWindowScroll)
     window.removeEventListener('online', handleOnline)
     window.removeEventListener('offline', handleOffline)
     inputElement.removeEventListener('keydown', handleKeyDown)

  2. Clear any pending timers:
     clearTimeout(debounceTimerId)
     clearTimeout(retryTimerId)
     clearTimeout(animationTimerId)

  3. Clear references:
     dropdownElement = null
     inputElement = null
     scrollContainer = null

  4. Reset state:
     state = INITIAL_DROPDOWN_STATE
```

### 7.3 Animation Performance

```css
/* GPU-accelerated animations only */
.autocomplete-dropdown {
  transform: translateY(0);
  opacity: 1;
  transition: transform 150ms ease-out, opacity 150ms ease-out;
}

.autocomplete-dropdown[data-state="closed"] {
  transform: translateY(-8px);
  opacity: 0;
  pointer-events: none;
}

/* Reduce motion preference */
@media (prefers-reduced-motion: reduce) {
  .autocomplete-dropdown {
    transition: none;
  }
}
```

---

## 8. Integration with SearchView

### 8.1 Parent-Child Communication

```typescript
// SearchView manages dropdown state
interface SearchViewDropdownIntegration {
  // SearchView provides to AutocompleteDropdown:
  inputElement: HTMLInputElement;
  isOpen: boolean;
  query: string;
  items: AutocompleteItem[];
  historyItems: SearchHistoryItem[];
  isLoading: boolean;
  error: DropdownError | null;
  totalCount: number;

  // AutocompleteDropdown calls back to SearchView:
  onSelect: (item: AutocompleteItem) => void;
  onSelectHistory: (item: SearchHistoryItem) => void;
  onClose: () => void;
  onViewAllResults: () => void;
  onRetry: () => void;
}
```

### 8.2 Event Flow

```
User types in SearchView input
         │
         ▼
SearchView debounces (150ms)
         │
         ▼
SearchView calls API
         │
         ▼
SearchView receives results
         │
         ▼
SearchView passes props to AutocompleteDropdown
         │
         ▼
AutocompleteDropdown renders items
         │
         ▼
User navigates/selects item
         │
         ▼
AutocompleteDropdown calls onSelect callback
         │
         ▼
SearchView handles selection (add to ingredients, navigate, etc.)
         │
         ▼
SearchView sets isOpen=false
         │
         ▼
AutocompleteDropdown closes
```

---

## Changelog

### 2026-01-22 (Rev 1.0)

**Added:**
- Initial detailed design document for AutocompleteDropdown component
- Complete type definitions for items, state, errors, and positioning
- Step-by-step algorithms for all user interactions
- Keyboard navigation with full ARIA support
- Mouse interaction handling
- Text highlighting for search matches
- History section with clear/remove functionality
- Error state handling with retry logic
- Position calculation with viewport awareness
- State transition diagrams
- Full accessibility specification (WCAG 2.1 AA)
- Performance optimization strategies
- Integration specification with SearchView parent component
