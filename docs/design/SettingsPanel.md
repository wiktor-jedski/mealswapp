# Detailed Design: SettingsPanel

**Traceability:** [ARCH-001]

**Tech Stack Compliance:**
- Frontend Framework: Svelte 5 (using runes: `$state`, `$effect`, `$derived`, `$props`)
- Build Tool: Bun
- State Management: Svelte stores + TanStack Query
- CSS: Tailwind
- Testing: Bun test runner + @testing-library/svelte + Playwright

---

## 1. Data Structures & Types

### 1.1 Panel Visibility State

```typescript
type SettingsPanelMode = 'closed' | 'open';

interface SettingsPanelState {
  mode: SettingsPanelMode;
  activeSection: SettingsSectionId | null;  // For accordion/scroll-to behavior
  isAnimating: boolean;
}

type SettingsSectionId =
  | 'appearance'
  | 'search-defaults'
  | 'storage'
  | 'about';
```

### 1.2 Settings Data Types (from LocalStorageManager)

```typescript
// Re-exported from LocalStorageManager for convenience
interface UserPreferences {
  version: number;
  searchMode: 'single' | 'multi';
  defaultMacroToggles: MacroToggleState;
  defaultSort: SortOption;
  resultsPerPage: number;
}

interface MacroToggleState {
  calories: boolean;
  protein: boolean;
  carbs: boolean;
  fat: boolean;
}

type SortOption = 'relevance' | 'name' | 'calories_asc' | 'calories_desc';

// Theme types (from ThemeProvider)
type Theme = 'light' | 'dark';
```

### 1.3 Storage Usage Types (from LocalStorageManager and ServiceWorkerCache)

```typescript
// LocalStorage usage (from LocalStorageManager)
interface LocalStorageUsage {
  totalUsed: number;          // Bytes used by Mealswapp in localStorage
  quota: number;              // Available quota (estimated 5MB)
  percentUsed: number;        // 0-100
  breakdown: {
    queryCache: number;
    searchHistory: number;
    preferences: number;
    metadata: number;
  };
}

// Service Worker Cache API usage (for images per ARCH-011)
interface ServiceWorkerCacheUsage {
  totalUsed: number;          // Bytes used in Cache API
  imageCount: number;         // Number of cached images
  apiResponseCount: number;   // Number of cached API responses
  isAvailable: boolean;       // Whether SW cache is accessible
}

// Combined storage usage for display
interface StorageUsage {
  localStorage: LocalStorageUsage;
  serviceWorkerCache: ServiceWorkerCacheUsage | null;  // null if SW not available
  combinedTotal: number;      // Sum of both caches
}
```

### 1.4 Component Props

```typescript
interface SettingsPanelProps {
  isOpen: boolean;
  onClose: () => void;
  initialSection?: SettingsSectionId;  // Scroll to section on open
}

interface SettingsSectionProps {
  id: SettingsSectionId;
  title: string;
  children: Snippet;  // Svelte uses Snippet for slot content
}
```

### 1.5 Confirmation Dialog Types

```typescript
type ConfirmationAction =
  | 'clear-query-cache'     // localStorage query cache
  | 'clear-image-cache'     // Service Worker image cache (ARCH-011)
  | 'clear-history'
  | 'clear-all-data'
  | 'reset-preferences';

interface ConfirmationDialogState {
  isOpen: boolean;
  action: ConfirmationAction | null;
  title: string;
  message: string;
  confirmLabel: string;
  isDestructive: boolean;
}

const CONFIRMATION_CONFIGS: Record<ConfirmationAction, Omit<ConfirmationDialogState, 'isOpen' | 'action'>> = {
  'clear-query-cache': {
    title: 'Clear Search Cache',
    message: 'This will remove all cached search results from localStorage. Your next searches will fetch fresh data from the server.',
    confirmLabel: 'Clear Cache',
    isDestructive: false
  },
  'clear-image-cache': {
    title: 'Clear Cached Images',
    message: 'This will remove all cached food images. Images will be re-downloaded as needed. This may increase data usage on your next searches.',
    confirmLabel: 'Clear Images',
    isDestructive: false
  },
  'clear-history': {
    title: 'Clear Search History',
    message: 'This will remove all your recent search history. This action cannot be undone.',
    confirmLabel: 'Clear History',
    isDestructive: false
  },
  'clear-all-data': {
    title: 'Clear All Data',
    message: 'This will clear all cached data (search results and images), search history, and reset your preferences to defaults. The page will reload after clearing.',
    confirmLabel: 'Clear All Data',
    isDestructive: true
  },
  'reset-preferences': {
    title: 'Reset Preferences',
    message: 'This will reset all your preferences (search defaults, display settings) to their original values.',
    confirmLabel: 'Reset Preferences',
    isDestructive: false
  }
};
```

### 1.6 Results Per Page Options

```typescript
const RESULTS_PER_PAGE_OPTIONS = [5, 10, 15, 20] as const;
type ResultsPerPage = typeof RESULTS_PER_PAGE_OPTIONS[number];
```

### 1.7 Sort Option Labels

```typescript
const SORT_OPTION_LABELS: Record<SortOption, string> = {
  'relevance': 'Relevance',
  'name': 'Name (A-Z)',
  'calories_asc': 'Calories (Low to High)',
  'calories_desc': 'Calories (High to Low)'
};
```

### 1.8 Context Types

```typescript
interface SettingsPanelContextValue {
  isOpen: boolean;
  open: (section?: SettingsSectionId) => void;
  close: () => void;
  toggle: () => void;
}

const SETTINGS_PANEL_CONTEXT_KEY = 'settings-panel-context';
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Panel Open/Close Flow

```
FUNCTION openSettingsPanel(initialSection?: SettingsSectionId):
  1. Update state
     panelState.mode = 'open'
     panelState.activeSection = initialSection || null
     panelState.isAnimating = true

  2. Prevent body scroll
     document.body.style.overflow = 'hidden'

  3. Set focus trap
     3.1. Store currently focused element for restoration
          previouslyFocused = document.activeElement
     3.2. Move focus to panel close button (or first focusable element)

  4. IF initialSection provided:
     4.1. After animation completes, scroll section into view
          setTimeout(() => {
            scrollToSection(initialSection)
          }, ANIMATION_DURATION)

  5. Announce to screen readers
     // aria-live region will announce "Settings opened"

FUNCTION closeSettingsPanel():
  1. Start exit animation
     panelState.isAnimating = true

  2. Restore body scroll
     document.body.style.overflow = ''

  3. Restore focus
     IF previouslyFocused AND previouslyFocused.isConnected:
       previouslyFocused.focus()

  4. After animation completes
     setTimeout(() => {
       panelState.mode = 'closed'
       panelState.activeSection = null
       panelState.isAnimating = false
     }, ANIMATION_DURATION)
```

### 2.2 Keyboard Navigation

```
FUNCTION handleKeyDown(event: KeyboardEvent):
  1. Handle Escape key
     IF event.key === 'Escape':
       event.preventDefault()
       CALL closeSettingsPanel()

  2. Handle Tab key (focus trap)
     IF event.key === 'Tab':
       focusableElements = getFocusableElements(panelRef)
       firstElement = focusableElements[0]
       lastElement = focusableElements[focusableElements.length - 1]

       IF event.shiftKey:
         // Shift+Tab: if on first element, wrap to last
         IF document.activeElement === firstElement:
           event.preventDefault()
           lastElement.focus()
       ELSE:
         // Tab: if on last element, wrap to first
         IF document.activeElement === lastElement:
           event.preventDefault()
           firstElement.focus()

FUNCTION getFocusableElements(container: HTMLElement): HTMLElement[]
  selector = 'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
  RETURN Array.from(container.querySelectorAll(selector))
         .filter(el => !el.disabled && el.offsetParent !== null)
```

### 2.3 Theme Selection

```
FUNCTION handleThemeChange(newTheme: Theme):
  1. Get theme context
     { setTheme } = useTheme()

  2. Apply new theme
     setTheme(newTheme)
     // ThemeProvider handles persistence and DOM updates
```

### 2.4 Search Mode Change

```
FUNCTION handleSearchModeChange(mode: 'single' | 'multi'):
  1. Get storage manager
     storage = LocalStorageManager.getInstance()

  2. Update preferences
     result = storage.updateUserPreferences({ searchMode: mode })

  3. Handle result
     IF result.success:
       // Update local state to reflect change
       setPreferences(prev => ({ ...prev, searchMode: mode }))
     ELSE:
       showToast('Could not save preference', 'error')
```

### 2.5 Macro Toggle Defaults Change

```
FUNCTION handleMacroToggleChange(macro: keyof MacroToggleState, enabled: boolean):
  1. Get current preferences
     storage = LocalStorageManager.getInstance()
     currentResult = storage.getUserPreferences()
     current = currentResult.success ? currentResult.data : DEFAULT_USER_PREFERENCES

  2. Create updated toggles
     updatedToggles = {
       ...current.defaultMacroToggles,
       [macro]: enabled
     }

  3. Save updated preferences
     result = storage.updateUserPreferences({
       defaultMacroToggles: updatedToggles
     })

  4. Handle result
     IF result.success:
       setPreferences(prev => ({
         ...prev,
         defaultMacroToggles: updatedToggles
       }))
     ELSE:
       showToast('Could not save preference', 'error')
```

### 2.6 Sort Default Change

```
FUNCTION handleSortChange(sort: SortOption):
  1. Get storage manager
     storage = LocalStorageManager.getInstance()

  2. Update preferences
     result = storage.updateUserPreferences({ defaultSort: sort })

  3. Handle result
     IF result.success:
       setPreferences(prev => ({ ...prev, defaultSort: sort }))
```

### 2.7 Results Per Page Change

```
FUNCTION handleResultsPerPageChange(count: ResultsPerPage):
  1. Validate input
     IF NOT RESULTS_PER_PAGE_OPTIONS.includes(count):
       Log error: 'Invalid results per page value'
       RETURN

  2. Get storage manager
     storage = LocalStorageManager.getInstance()

  3. Update preferences
     result = storage.updateUserPreferences({ resultsPerPage: count })

  4. Handle result
     IF result.success:
       setPreferences(prev => ({ ...prev, resultsPerPage: count }))
```

### 2.8 Storage Usage Refresh

```
FUNCTION refreshStorageUsage():
  1. Set loading state
     setStorageLoading(true)

  2. Fetch localStorage usage
     storage = LocalStorageManager.getInstance()
     localStorageResult = storage.getStorageUsage()

  3. Fetch Service Worker cache usage (ARCH-011)
     swCacheResult = await getServiceWorkerCacheUsage()

  4. Combine usage data
     combinedUsage: StorageUsage = {
       localStorage: localStorageResult.success ? localStorageResult.data : null,
       serviceWorkerCache: swCacheResult,
       combinedTotal: (localStorageResult.data?.totalUsed || 0) +
                      (swCacheResult?.totalUsed || 0)
     }

  5. Update state
     setStorageUsage(combinedUsage)

  6. Clear loading state
     setStorageLoading(false)
```

### 2.8.1 Service Worker Cache Usage Query (ARCH-011)

```
FUNCTION getServiceWorkerCacheUsage(): Promise<ServiceWorkerCacheUsage | null>
  1. Check Service Worker availability
     IF NOT ('serviceWorker' IN navigator):
       RETURN null

  2. Check if Service Worker is registered
     registration = await navigator.serviceWorker.ready
     IF NOT registration:
       RETURN null

  3. Send message to Service Worker to calculate cache size
     3.1. Create message channel for response
          channel = new MessageChannel()

     3.2. Send request to SW
          registration.active.postMessage(
            { type: 'GET_CACHE_USAGE' },
            [channel.port2]
          )

     3.3. Wait for response (with timeout)
          response = await Promise.race([
            new Promise(resolve => {
              channel.port1.onmessage = (event) => resolve(event.data)
            }),
            new Promise(resolve => setTimeout(() => resolve(null), 5000))
          ])

  4. Parse response
     IF response AND response.success:
       RETURN {
         totalUsed: response.totalBytes,
         imageCount: response.imageCount,
         apiResponseCount: response.apiResponseCount,
         isAvailable: true
       }
     ELSE:
       RETURN { totalUsed: 0, imageCount: 0, apiResponseCount: 0, isAvailable: false }
```

### 2.9 Clear Query Cache Flow (localStorage)

```
FUNCTION handleClearQueryCache():
  1. Show confirmation dialog
     setConfirmation({
       isOpen: true,
       action: 'clear-query-cache',
       ...CONFIRMATION_CONFIGS['clear-query-cache']
     })

FUNCTION confirmClearQueryCache():
  1. Get storage manager
     storage = LocalStorageManager.getInstance()

  2. Clear localStorage query cache
     result = storage.invalidateQueryCache()

  3. Handle result
     IF result.success:
       showToast('Search cache cleared')
       CALL refreshStorageUsage()
     ELSE:
       showToast('Failed to clear cache', 'error')

  4. Close confirmation dialog
     setConfirmation({ isOpen: false, action: null })
```

### 2.9.1 Clear Image Cache Flow (Service Worker - ARCH-011)

```
FUNCTION handleClearImageCache():
  1. Check Service Worker availability
     IF NOT ('serviceWorker' IN navigator):
       showToast('Image cache not available', 'warning')
       RETURN

  2. Show confirmation dialog
     setConfirmation({
       isOpen: true,
       action: 'clear-image-cache',
       ...CONFIRMATION_CONFIGS['clear-image-cache']
     })

FUNCTION confirmClearImageCache():
  1. Set loading state
     setClearing(true)

  2. Send message to Service Worker to clear image cache
     registration = await navigator.serviceWorker.ready

     IF NOT registration.active:
       showToast('Service Worker not active', 'error')
       RETURN

     channel = new MessageChannel()
     registration.active.postMessage(
       { type: 'CLEAR_IMAGE_CACHE' },
       [channel.port2]
     )

  3. Wait for confirmation from Service Worker
     response = await new Promise(resolve => {
       channel.port1.onmessage = (event) => resolve(event.data)
       setTimeout(() => resolve({ success: false }), 10000)  // 10s timeout
     })

  4. Handle result
     IF response.success:
       showToast(`Cleared ${response.itemsDeleted} cached images`)
       CALL refreshStorageUsage()
     ELSE:
       showToast('Failed to clear image cache', 'error')

  5. Clear loading state
     setClearing(false)

  6. Close confirmation dialog
     setConfirmation({ isOpen: false, action: null })
```

### 2.10 Clear History Flow

```
FUNCTION handleClearHistory():
  1. Show confirmation dialog
     setConfirmation({
       isOpen: true,
       action: 'clear-history',
       ...CONFIRMATION_CONFIGS['clear-history']
     })

FUNCTION confirmClearHistory():
  1. Get storage manager
     storage = LocalStorageManager.getInstance()

  2. Clear history
     result = storage.clearSearchHistory()

  3. Handle result
     IF result.success:
       showToast('Search history cleared')
       CALL refreshStorageUsage()
     ELSE:
       showToast('Failed to clear history', 'error')

  4. Close confirmation dialog
     setConfirmation({ isOpen: false, action: null })
```

### 2.11 Clear All Data Flow (localStorage + Service Worker per ARCH-011)

```
FUNCTION handleClearAllData():
  1. Show confirmation dialog
     setConfirmation({
       isOpen: true,
       action: 'clear-all-data',
       ...CONFIRMATION_CONFIGS['clear-all-data']
     })

FUNCTION confirmClearAllData():
  1. Set clearing state
     setClearing(true)

  2. Clear localStorage data
     storage = LocalStorageManager.getInstance()
     localStorageResult = storage.clearAllData()

  3. Clear Service Worker cache (ARCH-011)
     swCleared = false
     IF 'serviceWorker' IN navigator:
       TRY:
         registration = await navigator.serviceWorker.ready
         IF registration.active:
           channel = new MessageChannel()
           registration.active.postMessage(
             { type: 'CLEAR_ALL_CACHES' },
             [channel.port2]
           )

           response = await new Promise(resolve => {
             channel.port1.onmessage = (event) => resolve(event.data)
             setTimeout(() => resolve({ success: false }), 10000)
           })

           swCleared = response.success
       CATCH:
         Log warning: 'Could not clear Service Worker cache'

  4. Handle result
     IF localStorageResult.success:
       message = swCleared
         ? 'All data cleared. Reloading...'
         : 'Local data cleared (image cache may remain). Reloading...'
       showToast(message)

       // Dispatch event for other components (per LocalStorageManager spec)
       window.dispatchEvent(new CustomEvent('mealswapp:storagecleared', {
         detail: { reason: 'user_initiated', includesServiceWorker: swCleared }
       }))

       // Reload after brief delay for toast visibility
       setTimeout(() => {
         window.location.reload()
       }, 1500)
     ELSE:
       showToast('Failed to clear data', 'error')
       setClearing(false)

  5. Close confirmation dialog (if not reloading)
     setConfirmation({ isOpen: false, action: null })
```

### 2.12 Reset Preferences Flow

```
FUNCTION handleResetPreferences():
  1. Show confirmation dialog
     setConfirmation({
       isOpen: true,
       action: 'reset-preferences',
       ...CONFIRMATION_CONFIGS['reset-preferences']
     })

FUNCTION confirmResetPreferences():
  1. Get storage manager
     storage = LocalStorageManager.getInstance()

  2. Reset preferences
     result = storage.resetUserPreferences()

  3. Handle result
     IF result.success:
       showToast('Preferences reset to defaults')
       // Reload preferences state
       CALL loadPreferences()
     ELSE:
       showToast('Failed to reset preferences', 'error')

  4. Close confirmation dialog
     setConfirmation({ isOpen: false, action: null })
```

### 2.13 Load Preferences on Mount

```
FUNCTION loadPreferences():
  1. Get storage manager
     storage = LocalStorageManager.getInstance()

  2. Load preferences
     result = storage.getUserPreferences()

  3. Update state
     IF result.success:
       setPreferences(result.data)
     ELSE:
       setPreferences(DEFAULT_USER_PREFERENCES)
       Log warning: 'Using default preferences'
```

### 2.14 Storage Size Formatting

```
FUNCTION formatStorageSize(bytes: number): string
  1. Define units
     units = ['B', 'KB', 'MB', 'GB']

  2. Find appropriate unit
     unitIndex = 0
     size = bytes

     WHILE size >= 1024 AND unitIndex < units.length - 1:
       size = size / 1024
       unitIndex += 1

  3. Format with appropriate precision
     IF unitIndex === 0:
       RETURN `${size} ${units[unitIndex]}`
     ELSE:
       RETURN `${size.toFixed(1)} ${units[unitIndex]}`
```

### 2.15 SettingsPanel Component Implementation

```
<script lang="ts">
  import { setContext, onMount, onDestroy } from 'svelte';
  import { writable } from 'svelte/store';

  interface SettingsPanelProps {
    isOpen: boolean;
    onClose: () => void;
    initialSection?: SettingsSectionId;
  }

  let { isOpen, onClose, initialSection }: SettingsPanelProps = $props();

  let preferences = $state<UserPreferences>(DEFAULT_USER_PREFERENCES);
  let storageUsage = $state<StorageUsage | null>(null);
  let storageLoading = $state(false);
  let isClearing = $state(false);
  let confirmation = $state<ConfirmationDialogState>({
    isOpen: false,
    action: null
  });

  let theme = $state<Theme>('light');
  let panelElement = $state<HTMLElement | null>(null);
  let previouslyFocused = $state<HTMLElement | null>(null);
  let closeButton = $state<HTMLButtonElement | null>(null);

  const panelState = $state({
    mode: 'closed' as 'closed' | 'open',
    activeSection: null as SettingsSectionId | null,
    isAnimating: false
  });

  $effect(() => {
    if (isOpen) {
      openSettingsPanel();
    } else {
      closeSettingsPanel();
    }
  });

  $effect(() => {
    if (!isOpen) return;

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault();
        onClose();
      }
    }

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  });

  function openSettingsPanel() {
    panelState.mode = 'open';
    panelState.activeSection = initialSection || null;
    panelState.isAnimating = true;
    document.body.style.overflow = 'hidden';
    previouslyFocused = document.activeElement as HTMLElement;

    loadPreferences();
    refreshStorageUsage();

    setTimeout(() => {
      closeButton?.focus();
      panelState.isAnimating = false;

      if (initialSection) {
        scrollToSection(initialSection);
      }
    }, ANIMATION_DURATION);
  }

  function closeSettingsPanel() {
    panelState.isAnimating = true;

    setTimeout(() => {
      document.body.style.overflow = '';
      if (previouslyFocused?.isConnected) {
        previouslyFocused.focus();
      }
      panelState.mode = 'closed';
      panelState.activeSection = null;
      panelState.isAnimating = false;
    }, ANIMATION_DURATION);
  }

  function scrollToSection(sectionId: SettingsSectionId) {
    const element = document.getElementById(`settings-section-${sectionId}`);
    element?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }

  function loadPreferences() {
    const storage = LocalStorageManager.getInstance();
    const result = storage.getUserPreferences();

    if (result.success) {
      preferences = result.data;
    } else {
      preferences = DEFAULT_USER_PREFERENCES;
    }
  }

  function refreshStorageUsage() {
    storageLoading = true;
    const storage = LocalStorageManager.getInstance();

    Promise.all([
      storage.getStorageUsage(),
      getServiceWorkerCacheUsage()
    ]).then(([localStorageResult, swCacheResult]) => {
      storageUsage = {
        localStorage: localStorageResult.success ? localStorageResult.data! : null,
        serviceWorkerCache: swCacheResult,
        combinedTotal: (localStorageResult.data?.totalUsed || 0) +
                       (swCacheResult?.totalUsed || 0)
      };
      storageLoading = false;
    });
  }

  async function getServiceWorkerCacheUsage(): Promise<ServiceWorkerCacheUsage | null> {
    if (!('serviceWorker' in navigator)) return null;

    try {
      const registration = await navigator.serviceWorker.ready;
      if (!registration?.active) return null;

      const channel = new MessageChannel();
      registration.active.postMessage(
        { type: 'GET_CACHE_USAGE' },
        [channel.port2]
      );

      const response = await Promise.race([
        new Promise<CacheUsageResponse | null>(resolve => {
          channel.port1.onmessage = (event) => resolve(event.data);
        }),
        new Promise<null>(resolve => setTimeout(() => resolve(null), 5000))
      ]);

      if (response) {
        return {
          totalUsed: response.totalBytes,
          imageCount: response.imageCount,
          apiResponseCount: response.apiResponseCount,
          isAvailable: true
        };
      }
    } catch {
      return null;
    }

    return { totalUsed: 0, imageCount: 0, apiResponseCount: 0, isAvailable: false };
  }

  function handleConfirmAction() {
    switch (confirmation.action) {
      case 'clear-query-cache':
        confirmClearQueryCache();
        break;
      case 'clear-image-cache':
        confirmClearImageCache();
        break;
      case 'clear-history':
        confirmClearHistory();
        break;
      case 'clear-all-data':
        confirmClearAllData();
        break;
      case 'reset-preferences':
        confirmResetPreferences();
        break;
    }
  }

  // ... other handler functions (handleThemeChange, handleSearchModeChange, etc.)
</script>

{#if panelState.mode === 'open'}
  <div
    class="settings-panel__backdrop"
    onclick={onClose}
    aria-hidden="true"
  ></div>

  <aside
    bind:this={panelElement}
    role="dialog"
    aria-modal="true"
    aria-labelledby="settings-panel-title"
    class="settings-panel"
  >
    <header class="settings-panel__header">
      <h2 id="settings-panel-title" class="settings-panel__title">
        Settings
      </h2>
      <button
        bind:this={closeButton}
        onclick={onClose}
        class="settings-panel__close"
        aria-label="Close settings"
      >
        <CloseIcon aria-hidden="true" />
      </button>
    </header>

    <div class="settings-panel__content">
      <SettingsSection id="appearance" title="Appearance">
        <ThemeSelector theme={theme} onChange={setTheme} />
      </SettingsSection>

      <SettingsSection id="search-defaults" title="Search Defaults">
        <SearchModeSelector
          value={preferences.searchMode}
          onChange={handleSearchModeChange}
        />
        <MacroTogglesSelector
          value={preferences.defaultMacroToggles}
          onChange={handleMacroToggleChange}
        />
        <SortSelector
          value={preferences.defaultSort}
          onChange={handleSortChange}
        />
        <ResultsPerPageSelector
          value={preferences.resultsPerPage}
          onChange={handleResultsPerPageChange}
        />
      </SettingsSection>

      <SettingsSection id="storage" title="Storage">
        <StorageUsageDisplay
          usage={storageUsage}
          loading={storageLoading}
        />
        <StorageActions
          onClearQueryCache={handleClearQueryCache}
          onClearImageCache={handleClearImageCache}
          onClearHistory={handleClearHistory}
          onClearAllData={handleClearAllData}
          isServiceWorkerAvailable={storageUsage?.serviceWorkerCache?.isAvailable ?? false}
          isClearing={isClearing}
        />
      </SettingsSection>

      <SettingsSection id="about" title="About">
        <AboutInfo />
      </SettingsSection>

      <button
        onclick={handleResetPreferences}
        class="settings-panel__reset-button"
      >
        Reset All Preferences
      </button>
    </div>
  </aside>

  <ConfirmationDialog
    isOpen={confirmation.isOpen}
    title={confirmation.title}
    message={confirmation.message}
    confirmLabel={confirmation.confirmLabel}
    isDestructive={confirmation.isDestructive}
    onConfirm={handleConfirmAction}
    onCancel={() => confirmation = { ...confirmation, isOpen: false, action: null }}
  />
{/if}
```

---

## 3. State Management & Error Handling

### 3.1 State Diagram

```
                          ┌─────────────────────────────────────┐
                          │              CLOSED                 │
                          │  (Panel not rendered, no scroll     │
                          │   lock, no focus trap)              │
                          └──────────────────┬──────────────────┘
                                             │
                                             │ open() called
                                             ▼
                          ┌─────────────────────────────────────┐
                          │            ANIMATING_IN             │
                          │  (Slide animation, body scroll      │
                          │   locked, loading data)             │
                          └──────────────────┬──────────────────┘
                                             │
                                             │ animation complete
                                             ▼
                          ┌─────────────────────────────────────┐
                          │              OPEN                   │
                          │  (Interactive, focus trapped,       │
                          │   preferences loaded)               │
                          └──────────────────┬──────────────────┘
                                             │
                        ┌────────────────────┴────────────────────┐
                        │                                         │
                        ▼                                         ▼
             ┌─────────────────────┐                   ┌─────────────────────┐
             │  CONFIRMATION_OPEN  │                   │    ANIMATING_OUT    │
             │  (Dialog shown,     │                   │  (Close triggered,  │
             │   panel still open) │                   │   restoring state)  │
             └──────────┬──────────┘                   └──────────┬──────────┘
                        │                                         │
                        │ confirm/cancel                          │ animation complete
                        ▼                                         ▼
             ┌─────────────────────┐                   ┌─────────────────────┐
             │       OPEN          │                   │       CLOSED        │
             │ (Action executed)   │                   │ (Focus restored)    │
             └─────────────────────┘                   └─────────────────────┘
```

### 3.2 Settings Modification Flow

```
                    ┌─────────────────────────────┐
                    │       USER_INTERACTION      │
                    │  (Change toggle, select     │
                    │   dropdown option, etc.)    │
                    └──────────────┬──────────────┘
                                   │
                                   ▼
                    ┌─────────────────────────────┐
                    │       VALIDATE_INPUT        │
                    │  (Check value is valid)     │
                    └──────────────┬──────────────┘
                                   │
                      ┌────────────┴────────────┐
                      │ Valid                   │ Invalid
                      ▼                         ▼
           ┌─────────────────────┐   ┌─────────────────────┐
           │   PERSIST_CHANGE    │   │   SHOW_ERROR        │
           │ (LocalStorageManager│   │ (Toast notification)│
           │  updatePreferences) │   └─────────────────────┘
           └──────────┬──────────┘
                      │
           ┌──────────┴──────────┐
           │ Success             │ Failure
           ▼                     ▼
┌─────────────────────┐   ┌─────────────────────┐
│   UPDATE_UI_STATE   │   │    REVERT_UI        │
│ (Optimistic update  │   │ (Show previous      │
│  confirmed)         │   │  value, show error) │
└─────────────────────┘   └─────────────────────┘
```

### 3.3 Error States

| Error State | Trigger | User Impact | Recovery Action |
|:------------|:--------|:------------|:----------------|
| **STORAGE_UNAVAILABLE** | LocalStorage blocked | Cannot persist changes | Show warning, changes work per-session only |
| **STORAGE_QUOTA_EXCEEDED** | Storage full when saving | Preference not saved | Clear cache first, retry save |
| **PREFERENCES_CORRUPTED** | Invalid data in storage | Defaults loaded instead | Reset preferences, clear corrupted data |
| **CLEAR_OPERATION_FAILED** | Error during cache clear | Data not cleared | Show error toast, suggest retry |
| **CONTEXT_MISSING** | Panel used outside providers | Panel won't render | Throw error with clear message |

### 3.4 Error Handling Implementation

```typescript
type SettingsErrorType =
  | 'STORAGE_UNAVAILABLE'
  | 'STORAGE_QUOTA_EXCEEDED'
  | 'PREFERENCES_CORRUPTED'
  | 'CLEAR_OPERATION_FAILED'
  | 'CONTEXT_MISSING';

interface SettingsError {
  type: SettingsErrorType;
  message: string;
  recoverable: boolean;
  userMessage: string;  // Message to show in toast
}

const ERROR_MESSAGES: Record<SettingsErrorType, SettingsError> = {
  'STORAGE_UNAVAILABLE': {
    type: 'STORAGE_UNAVAILABLE',
    message: 'localStorage is not available',
    recoverable: false,
    userMessage: 'Settings cannot be saved in private browsing mode'
  },
  'STORAGE_QUOTA_EXCEEDED': {
    type: 'STORAGE_QUOTA_EXCEEDED',
    message: 'localStorage quota exceeded',
    recoverable: true,
    userMessage: 'Storage full. Try clearing cache first.'
  },
  'PREFERENCES_CORRUPTED': {
    type: 'PREFERENCES_CORRUPTED',
    message: 'Preferences data is corrupted',
    recoverable: true,
    userMessage: 'Preferences reset to defaults due to data error'
  },
  'CLEAR_OPERATION_FAILED': {
    type: 'CLEAR_OPERATION_FAILED',
    message: 'Failed to clear storage',
    recoverable: true,
    userMessage: 'Could not clear data. Please try again.'
  },
  'CONTEXT_MISSING': {
    type: 'CONTEXT_MISSING',
    message: 'SettingsPanel must be used within required providers',
    recoverable: false,
    userMessage: 'Application error. Please refresh the page.'
  }
};

FUNCTION handleSettingsError(error: StorageError | Error): void
  1. Determine error type
     errorType = mapToSettingsErrorType(error)
     config = ERROR_MESSAGES[errorType]

  2. Log for debugging
     console.error(`[SettingsPanel] ${config.message}`, error)

  3. Show user feedback
     showToast(config.userMessage, config.recoverable ? 'warning' : 'error')

  4. IF errorType === 'PREFERENCES_CORRUPTED':
     // Attempt automatic recovery
     storage = LocalStorageManager.getInstance()
     storage.resetUserPreferences()
     loadPreferences()  // Reload with defaults
```

### 3.5 Graceful Degradation

| Scenario | Degraded Behavior | Core Functionality |
|:---------|:------------------|:-------------------|
| **localStorage unavailable** | Changes work per-session but don't persist | All settings interactive |
| **Storage quota exceeded** | Cannot save new preferences until cache cleared | Existing preferences work |
| **Theme context missing** | Theme section hidden, warning logged | Other sections work |
| **Network offline** | All local settings work normally | Full functionality |

---

## 4. Component Interfaces

### 4.1 SettingsPanel Component

```typescript
interface SettingsPanelProps {
  /** Whether the panel is currently open */
  isOpen: boolean;
  /** Callback fired when panel should close */
  onClose: () => void;
  /** Optional section to scroll to on open */
  initialSection?: SettingsSectionId;
}

function SettingsPanel(props: SettingsPanelProps): void;
```

### 4.2 SettingsPanelProvider Component

```typescript
interface SettingsPanelProviderProps {
  children: Snippet;
}

function SettingsPanelProvider(props: SettingsPanelProviderProps): void;
```

### 4.3 useSettingsPanel Hook

```typescript
interface SettingsPanelContextValue {
  /** Whether the panel is currently open */
  isOpen: boolean;
  /** Open the settings panel, optionally scrolling to a section */
  open: (section?: SettingsSectionId) => void;
  /** Close the settings panel */
  close: () => void;
  /** Toggle the settings panel */
  toggle: () => void;
}

function useSettingsPanel(): SettingsPanelContextValue;
```

In Svelte, this is implemented using `setContext` and `getContext`:

```typescript
const SETTINGS_PANEL_CONTEXT_KEY = 'settings-panel-context';

function createSettingsPanelStore() {
  const { subscribe, update } = writable({
    isOpen: false,
    activeSection: null as SettingsSectionId | null
  });

  return {
    subscribe,
    open: (section?: SettingsSectionId) => update(s => ({ ...s, isOpen: true, activeSection: section || null })),
    close: () => update(s => ({ ...s, isOpen: false, activeSection: null })),
    toggle: () => update(s => ({ ...s, isOpen: !s.isOpen, activeSection: null }))
  };
}

const settingsPanelStore = createSettingsPanelStore();
setContext(SETTINGS_PANEL_CONTEXT_KEY, settingsPanelStore);
```

### 4.4 SettingsSection Component

```typescript
interface SettingsSectionProps {
  /** Unique identifier for the section */
  id: SettingsSectionId;
  /** Section header title */
  title: string;
  /** Section content */
  children: Snippet;
}

function SettingsSection(props: SettingsSectionProps): void;
```

**Svelte implementation:**

```svelte
<script lang="ts">
  let { id, title, children }: SettingsSectionProps = $props();
</script>

<section id="settings-section-{id}" class="settings-section">
  <header class="settings-section__header">
    <h3 class="settings-section__title">{title}</h3>
  </header>
  <div class="settings-section__content">
    {@render children()}
  </div>
</section>
```

### 4.5 ThemeSelector Component

```typescript
interface ThemeSelectorProps {
  /** Current theme value */
  value: Theme;
  /** Callback fired when theme changes */
  onChange: (theme: Theme) => void;
}

function ThemeSelector(props: ThemeSelectorProps): void;
```

### 4.6 SearchModeSelector Component

```typescript
interface SearchModeSelectorProps {
  /** Current search mode value */
  value: 'single' | 'multi';
  /** Callback fired when mode changes */
  onChange: (mode: 'single' | 'multi') => void;
}

function SearchModeSelector(props: SearchModeSelectorProps): void;
```

### 4.7 MacroTogglesSelector Component

```typescript
interface MacroTogglesSelectorProps {
  /** Current macro toggle states */
  value: MacroToggleState;
  /** Callback fired when a toggle changes */
  onChange: (macro: keyof MacroToggleState, enabled: boolean) => void;
}

function MacroTogglesSelector(props: MacroTogglesSelectorProps): void;
```

### 4.8 SortSelector Component

```typescript
interface SortSelectorProps {
  /** Current sort option */
  value: SortOption;
  /** Callback fired when sort changes */
  onChange: (sort: SortOption) => void;
}

function SortSelector(props: SortSelectorProps): void;
```

### 4.9 ResultsPerPageSelector Component

```typescript
interface ResultsPerPageSelectorProps {
  /** Current results per page value */
  value: number;
  /** Callback fired when value changes */
  onChange: (count: ResultsPerPage) => void;
}

function ResultsPerPageSelector(props: ResultsPerPageSelectorProps): void;
```

### 4.10 StorageUsageDisplay Component

```typescript
interface StorageUsageDisplayProps {
  /** Storage usage data, null if unavailable */
  usage: StorageUsage | null;
  /** Whether usage is currently loading */
  loading: boolean;
}

function StorageUsageDisplay(props: StorageUsageDisplayProps): void;
```

### 4.6 SearchModeSelector Component

```typescript
interface SearchModeSelectorProps {
  /** Current search mode value */
  value: 'single' | 'multi';
  /** Callback fired when mode changes */
  onChange: (mode: 'single' | 'multi') => void;
}

function SearchModeSelector(props: SearchModeSelectorProps): JSX.Element;
```

### 4.7 MacroTogglesSelector Component

```typescript
interface MacroTogglesSelectorProps {
  /** Current macro toggle states */
  value: MacroToggleState;
  /** Callback fired when a toggle changes */
  onChange: (macro: keyof MacroToggleState, enabled: boolean) => void;
}

function MacroTogglesSelector(props: MacroTogglesSelectorProps): JSX.Element;
```

### 4.8 SortSelector Component

```typescript
interface SortSelectorProps {
  /** Current sort option */
  value: SortOption;
  /** Callback fired when sort changes */
  onChange: (sort: SortOption) => void;
}

function SortSelector(props: SortSelectorProps): JSX.Element;
```

### 4.9 ResultsPerPageSelector Component

```typescript
interface ResultsPerPageSelectorProps {
  /** Current results per page value */
  value: number;
  /** Callback fired when value changes */
  onChange: (count: ResultsPerPage) => void;
}

function ResultsPerPageSelector(props: ResultsPerPageSelectorProps): JSX.Element;
```

### 4.10 StorageUsageDisplay Component

```typescript
interface StorageUsageDisplayProps {
  /** Storage usage data, null if unavailable */
  usage: StorageUsage | null;
  /** Whether usage is currently loading */
  loading: boolean;
}

function StorageUsageDisplay(props: StorageUsageDisplayProps): JSX.Element;

/**
 * Displays:
 * - Combined total usage with visual progress bar
 * - localStorage breakdown (query cache, history, preferences)
 * - Service Worker cache breakdown (images, API responses) if available
 * - Visual progress bars for each tier
 *
 * Render structure:
 * ┌─────────────────────────────────────────────┐
 * │ Total Storage Used                          │
 * │ ████████████░░░░░░░░ 2.3 MB / 10 MB        │
 * │                                             │
 * │ Local Data (localStorage)                   │
 * │   Search cache      450 KB                  │
 * │   Search history    12 KB                   │
 * │   Preferences       2 KB                    │
 * │                                             │
 * │ Cached Images (Service Worker)              │
 * │   Food images       1.8 MB (127 images)    │
 * │   API responses     38 KB                   │
 * │   -- or --                                  │
 * │   Image cache not available                 │
 * └─────────────────────────────────────────────┘
 */
```

### 4.11 StorageActions Component

```typescript
interface StorageActionsProps {
  /** Callback for clear query cache action (localStorage) */
  onClearQueryCache: () => void;
  /** Callback for clear image cache action (Service Worker - ARCH-011) */
  onClearImageCache: () => void;
  /** Callback for clear history action */
  onClearHistory: () => void;
  /** Callback for clear all data action */
  onClearAllData: () => void;
  /** Whether Service Worker cache is available */
  isServiceWorkerAvailable: boolean;
  /** Whether a clear operation is in progress */
  isClearing: boolean;
}

function StorageActions(props: StorageActionsProps): void;
```

### 4.12 ConfirmationDialog Component

```typescript
interface ConfirmationDialogProps {
  /** Whether the dialog is open */
  isOpen: boolean;
  /** Dialog title */
  title: string;
  /** Dialog message/description */
  message: string;
  /** Label for confirm button */
  confirmLabel: string;
  /** Whether action is destructive (affects button styling) */
  isDestructive: boolean;
  /** Callback fired on confirm */
  onConfirm: () => void;
  /** Callback fired on cancel */
  onCancel: () => void;
}

function ConfirmationDialog(props: ConfirmationDialogProps): void | null;
```

### 4.13 AboutInfo Component

```typescript
// No props - displays static app information
function AboutInfo(): void;
```

### 4.12 ConfirmationDialog Component

```typescript
interface ConfirmationDialogProps {
  /** Whether the dialog is open */
  isOpen: boolean;
  /** Dialog title */
  title: string;
  /** Dialog message/description */
  message: string;
  /** Label for confirm button */
  confirmLabel: string;
  /** Whether action is destructive (affects button styling) */
  isDestructive: boolean;
  /** Callback fired on confirm */
  onConfirm: () => void;
  /** Callback fired on cancel */
  onCancel: () => void;
}

function ConfirmationDialog(props: ConfirmationDialogProps): JSX.Element | null;
```

### 4.13 AboutInfo Component

```typescript
// No props - displays static app information
function AboutInfo(): JSX.Element;
```

### 4.14 CSS Class Contract

```css
/* Panel structure */
.settings-panel { }
.settings-panel__backdrop { }
.settings-panel__header { }
.settings-panel__title { }
.settings-panel__close { }
.settings-panel__content { }
.settings-panel__reset-button { }

/* Section structure */
.settings-section { }
.settings-section__header { }
.settings-section__title { }
.settings-section__content { }

/* Theme selector */
.theme-selector { }
.theme-selector__option { }
.theme-selector__option--active { }
.theme-selector__icon { }
.theme-selector__label { }

/* Toggle/switch */
.settings-toggle { }
.settings-toggle__track { }
.settings-toggle__thumb { }
.settings-toggle--checked { }

/* Select/dropdown */
.settings-select { }
.settings-select__trigger { }
.settings-select__options { }
.settings-select__option { }
.settings-select__option--selected { }

/* Storage display */
.storage-usage { }
.storage-usage__bar { }
.storage-usage__bar-fill { }
.storage-usage__text { }
.storage-usage__breakdown { }
.storage-usage__breakdown-item { }

/* Storage actions */
.storage-actions { }
.storage-actions__button { }
.storage-actions__button--destructive { }

/* Confirmation dialog */
.confirmation-dialog { }
.confirmation-dialog__backdrop { }
.confirmation-dialog__content { }
.confirmation-dialog__title { }
.confirmation-dialog__message { }
.confirmation-dialog__actions { }
.confirmation-dialog__button { }
.confirmation-dialog__button--confirm { }
.confirmation-dialog__button--cancel { }
.confirmation-dialog__button--destructive { }
```

### 4.15 Service Worker Message Types (ARCH-011 Coordination)

```typescript
// Messages sent from SettingsPanel to Service Worker
type ServiceWorkerCommand =
  | { type: 'GET_CACHE_USAGE' }
  | { type: 'CLEAR_IMAGE_CACHE' }
  | { type: 'CLEAR_ALL_CACHES' };

// Responses from Service Worker to SettingsPanel
interface CacheUsageResponse {
  success: true;
  totalBytes: number;
  imageCount: number;
  apiResponseCount: number;
}

interface ClearCacheResponse {
  success: boolean;
  itemsDeleted?: number;
  bytesFreed?: number;
  error?: string;
}

// Service Worker must implement these message handlers:
// - GET_CACHE_USAGE: Calculate and return Cache API storage usage
// - CLEAR_IMAGE_CACHE: Delete all entries from image cache
// - CLEAR_ALL_CACHES: Delete all Mealswapp caches (images + API responses)
```

---

## 5. Integration Requirements

### 5.1 Application Root Setup

```svelte
<!-- App.svelte -->
<script lang="ts">
  import { SettingsPanelProvider } from './providers/SettingsPanelProvider.svelte';
  import { SettingsPanelContainer } from './components/SettingsPanelContainer.svelte';
</script>

<ThemeProvider>
  <NetworkProvider>
    <SettingsPanelProvider>
      {@render children()}
      <SettingsPanelContainer />
    </SettingsPanelProvider>
  </NetworkProvider>
</ThemeProvider>

<!-- SettingsPanelContainer.svelte -->
<script lang="ts">
  import { useSettingsPanel } from './hooks/useSettingsPanel.svelte';
  import { SettingsPanel } from './components/SettingsPanel.svelte';

  const { isOpen, close } = useSettingsPanel();
</script>

<SettingsPanel isOpen={$isOpen} onClose={close} />
```

### 5.2 Opening Settings from Other Components

```svelte
<!-- Sidebar.svelte -->
<script lang="ts">
  import { useSettingsPanel } from '../hooks/useSettingsPanel.svelte';

  const { open } = useSettingsPanel();
</script>

<nav class="sidebar">
  <!-- ... other nav items ... -->
  <button
    onclick={() => open()}
    class="sidebar__settings-button"
    aria-label="Open settings"
  >
    <SettingsIcon aria-hidden="true" />
    Settings
  </button>
</nav>

<!-- Or use keyboard shortcut in a component -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { useSettingsPanel } from '../hooks/useSettingsPanel.svelte';

  const { toggle } = useSettingsPanel();

  onMount(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === ',') {
        e.preventDefault();
        toggle();
      }
    }

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  });
</script>
```

### 5.3 CSS Implementation (Tailwind)

```svelte
<!-- Panel backdrop -->
<div
  class="fixed inset-0 bg-black/50 z-[999] animate-fade-in"
  onclick={onClose}
  aria-hidden="true"
/>

<!-- Panel container -->
<aside
  class="fixed top-0 right-0 bottom-0 w-full max-w-md bg-surface shadow-xl z-[1000] flex flex-col animate-slide-in-right"
>
  <!-- Header -->
  <header class="flex items-center justify-between px-5 py-4 border-b border-border">
    <h2 id="settings-panel-title" class="text-lg font-semibold text-primary m-0">
      Settings
    </h2>
    <button
      class="flex items-center justify-center w-8 h-8 border-none bg-transparent rounded-md text-secondary cursor-pointer hover:bg-hover transition-colors"
      onclick={onClose}
      aria-label="Close settings"
    >
      <CloseIcon class="w-5 h-5" aria-hidden="true" />
    </button>
  </header>

  <!-- Content -->
  <div class="flex-1 overflow-y-auto px-5 py-5">
    <!-- Sections with Tailwind -->
    <section class="mb-6">
      <h3 class="text-sm font-semibold text-secondary uppercase tracking-wide m-0 mb-3">
        Appearance
      </h3>
      <div class="flex flex-col gap-4">
        <!-- Theme selector using Tailwind -->
        <div class="flex gap-3">
          <button
            class="flex-1 flex flex-col items-center gap-2 p-4 border-2 border-border rounded-lg bg-transparent cursor-pointer hover:bg-hover transition-all"
            class:border-primary={theme === 'light'}
            class:bg-secondary={theme === 'light'}
          >
            <SunIcon class="w-6 h-6 text-primary" />
            <span class="text-sm font-medium text-primary">Light</span>
          </button>
          <button
            class="flex-1 flex flex-col items-center gap-2 p-4 border-2 border-border rounded-lg bg-transparent cursor-pointer hover:bg-hover transition-all"
            class:border-primary={theme === 'dark'}
            class:bg-secondary={theme === 'dark'}
          >
            <MoonIcon class="w-6 h-6 text-primary" />
            <span class="text-sm font-medium text-primary">Dark</span>
          </button>
        </div>
      </div>
    </section>
  </div>
</aside>

<style>
  @keyframes fade-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  @keyframes slide-in-right {
    from { transform: translateX(100%); }
    to { transform: translateX(0); }
  }

  .animate-fade-in {
    animation: fade-in 200ms ease-out;
  }

  .animate-slide-in-right {
    animation: slide-in-right 200ms ease-out;
  }
</style>
```
  text-decoration: underline;
}

.settings-panel__reset-button:hover {
  color: var(--text-secondary);
}

/* Confirmation dialog */
.confirmation-dialog {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1001;
}

.confirmation-dialog__backdrop {
  position: absolute;
  inset: 0;
  background-color: var(--bg-overlay);
}

.confirmation-dialog__content {
  position: relative;
  width: 90%;
  max-width: 400px;
  padding: 24px;
  background-color: var(--bg-surface);
  border-radius: 12px;
  box-shadow: 0 8px 32px var(--shadow-color);
}

.confirmation-dialog__title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 12px 0;
}

.confirmation-dialog__message {
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.5;
  margin: 0 0 24px 0;
}

.confirmation-dialog__actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.confirmation-dialog__button {
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 150ms ease;
}

.confirmation-dialog__button--cancel {
  border: 1px solid var(--border-color);
  background: transparent;
  color: var(--text-primary);
}

.confirmation-dialog__button--cancel:hover {
  background-color: var(--hover-bg);
}

.confirmation-dialog__button--confirm {
  border: none;
  background-color: var(--color-primary);
  color: var(--text-inverse);
}

.confirmation-dialog__button--confirm:hover {
  background-color: var(--color-primary-hover);
}

.confirmation-dialog__button--destructive {
  background-color: var(--color-error);
}

.confirmation-dialog__button--destructive:hover {
  background-color: #b91c1c;  /* Darker red */
}

/* Reduced motion */
@media (prefers-reduced-motion: reduce) {
  .settings-panel,
  .settings-panel__backdrop,
  .confirmation-dialog__content {
    animation: none;
  }

  .settings-toggle__thumb,
  .storage-usage__bar-fill {
    transition: none;
  }
}

/* Mobile responsiveness */
@media (max-width: 480px) {
  .settings-panel {
    max-width: 100%;
  }

  .theme-selector {
    flex-direction: column;
  }

  .confirmation-dialog__actions {
    flex-direction: column;
  }

  .confirmation-dialog__button {
    width: 100%;
  }
}
```

### 5.4 Integration with LocalStorageManager

```typescript
// SettingsPanel uses LocalStorageManager directly for storage operations
import { LocalStorageManager } from '../services/LocalStorageManager';

// In SettingsPanel component:
let { isOpen }: SettingsPanelProps = $props();
const storage = LocalStorageManager.getInstance();

$effect(() => {
  if (isOpen) {
    const result = storage.getUserPreferences();
    if (result.success) {
      preferences = result.data;
    }
  }
});

function handleSearchModeChange(mode: 'single' | 'multi') {
  const result = storage.updateUserPreferences({ searchMode: mode });
  if (result.success) {
    preferences = result.data;
  } else {
    showToast('Could not save preference', 'error');
  }
}

function refreshStorageUsage() {
  const result = storage.getStorageUsage();
  if (result.success) {
    storageUsage = result.data;
  }
}

async function handleClearCache() {
  const result = await storage.invalidateQueryCache();
  if (result.success) {
    showToast('Cache cleared');
    refreshStorageUsage();
  }
}
```

### 5.5 Integration with Service Worker (ARCH-011)

```typescript
// Service Worker must handle messages from SettingsPanel for cache management

// sw.ts - Message handler implementation
self.addEventListener('message', (event: ExtendableMessageEvent) => {
  const { type } = event.data;
  const port = event.ports[0];

  switch (type) {
    case 'GET_CACHE_USAGE':
      handleGetCacheUsage(port);
      break;
    case 'CLEAR_IMAGE_CACHE':
      handleClearImageCache(port);
      break;
    case 'CLEAR_ALL_CACHES':
      handleClearAllCaches(port);
      break;
  }
});

async function handleGetCacheUsage(port: MessagePort) {
  try {
    const imageCache = await caches.open('mealswapp-images');
    const apiCache = await caches.open('mealswapp-api');

    const imageKeys = await imageCache.keys();
    const apiKeys = await apiCache.keys();

    // Estimate size by fetching each response
    let totalBytes = 0;
    for (const request of [...imageKeys, ...apiKeys]) {
      const cache = request.url.includes('/api/') ? apiCache : imageCache;
      const response = await cache.match(request);
      if (response) {
        const blob = await response.clone().blob();
        totalBytes += blob.size;
      }
    }

    port.postMessage({
      success: true,
      totalBytes,
      imageCount: imageKeys.length,
      apiResponseCount: apiKeys.length
    });
  } catch (error) {
    port.postMessage({ success: false, error: error.message });
  }
}

async function handleClearImageCache(port: MessagePort) {
  try {
    const deleted = await caches.delete('mealswapp-images');
    // Recreate empty cache
    await caches.open('mealswapp-images');

    port.postMessage({
      success: true,
      itemsDeleted: deleted ? 'all' : 0
    });
  } catch (error) {
    port.postMessage({ success: false, error: error.message });
  }
}

async function handleClearAllCaches(port: MessagePort) {
  try {
    const cacheNames = await caches.keys();
    const mealswappCaches = cacheNames.filter(name =>
      name.startsWith('mealswapp-')
    );

    await Promise.all(mealswappCaches.map(name => caches.delete(name)));

    port.postMessage({
      success: true,
      itemsDeleted: mealswappCaches.length
    });
  } catch (error) {
    port.postMessage({ success: false, error: error.message });
  }
}
```

### 5.6 Integration with ThemeProvider

```svelte
<!-- ThemeSelector.svelte -->
<script lang="ts">
  import { theme, setTheme } from '../stores/theme.svelte';

  interface ThemeSelectorProps {
    value: Theme;
    onChange: (theme: Theme) => void;
  }

  let { value, onChange }: ThemeSelectorProps = $props();

  function handleSelect(newTheme: Theme) {
    setTheme(newTheme);
    onChange(newTheme);
  }
</script>

<div class="flex gap-3" role="radiogroup" aria-label="Theme">
  <button
    role="radio"
    aria-checked={value === 'light'}
    class="flex-1 flex flex-col items-center gap-2 p-4 border-2 border-border rounded-lg cursor-pointer transition-all"
    class:border-primary={value === 'light'}
    class:bg-secondary={value === 'light'}
    onclick={() => handleSelect('light')}
  >
    <SunIcon class="w-6 h-6 text-primary" aria-hidden="true" />
    <span class="text-sm font-medium text-primary">Light</span>
  </button>
  <button
    role="radio"
    aria-checked={value === 'dark'}
    class="flex-1 flex flex-col items-center gap-2 p-4 border-2 border-border rounded-lg cursor-pointer transition-all"
    class:border-primary={value === 'dark'}
    class:bg-secondary={value === 'dark'}
    onclick={() => handleSelect('dark')}
  >
    <MoonIcon class="w-6 h-6 text-primary" aria-hidden="true" />
    <span class="text-sm font-medium text-primary">Dark</span>
  </button>
</div>
```

---

## 6. Performance Considerations

| Optimization | Implementation | Impact |
|:-------------|:---------------|:-------|
| **Lazy rendering** | Panel not rendered when closed | Zero DOM cost when hidden |
| **Data loading on open** | Preferences/usage loaded when panel opens | No startup overhead |
| **Stable handlers** | Regular functions for all handlers | Reactive dependencies managed by Svelte |
| **Debounced saves** | No debounce needed (discrete selections) | Immediate feedback |
| **CSS animations** | Hardware-accelerated transforms | Smooth 60fps open/close |
| **Confirmation dialogs** | Rendered only when needed | Minimal DOM presence |
| **Svelte runes** | Fine-grained reactivity | Only changed parts re-render |

### 6.1 Derived Computations

```svelte
<script lang="ts">
  let storageUsage = $state<StorageUsage | null>(null);

  // Derived value - automatically updates when storageUsage changes
  let formattedUsage = $derived.by(() => {
    if (!storageUsage) return null;
    return {
      used: formatStorageSize(storageUsage.totalUsed),
      quota: formatStorageSize(storageUsage.quota),
      percent: Math.round(storageUsage.combinedTotal / storageUsage.localStorage?.quota * 100)
    };
  });
</script>
```

---

## 7. Accessibility Considerations

| Requirement | Implementation |
|:------------|:---------------|
| **Focus management** | Focus trapped within panel when open; restored on close |
| **Keyboard navigation** | Escape closes panel; Tab cycles through controls |
| **Screen reader** | `role="dialog"`, `aria-modal="true"`, `aria-labelledby` |
| **Toggle controls** | Proper `role="switch"`, `aria-checked` states |
| **Radio groups** | Theme selector uses `role="radiogroup"`, `role="radio"` |
| **Destructive actions** | Confirmation dialogs prevent accidental data loss |
| **Color contrast** | All text/interactive elements meet WCAG AA |
| **Reduced motion** | Respects `prefers-reduced-motion` for animations |

### 7.1 ARIA Implementation

```html
<!-- Panel structure -->
<aside
  role="dialog"
  aria-modal="true"
  aria-labelledby="settings-panel-title"
  class="settings-panel"
>
  <header class="settings-panel__header">
    <h2 id="settings-panel-title">Settings</h2>
    <button aria-label="Close settings">
      <svg aria-hidden="true">...</svg>
    </button>
  </header>

  <!-- Theme selector -->
  <div role="radiogroup" aria-label="Theme">
    <button role="radio" aria-checked="true">Light</button>
    <button role="radio" aria-checked="false">Dark</button>
  </div>

  <!-- Toggle switch -->
  <label id="calories-label">Show Calories</label>
  <button
    role="switch"
    aria-checked="true"
    aria-labelledby="calories-label"
  >
    <span class="settings-toggle__track">
      <span class="settings-toggle__thumb"></span>
    </span>
  </button>

  <!-- Destructive action -->
  <button aria-describedby="clear-all-warning">
    Clear All Data
  </button>
  <span id="clear-all-warning" class="visually-hidden">
    Warning: This action cannot be undone
  </span>
</aside>
```

### 7.2 Focus Management Implementation

```svelte
<script lang="ts">
  interface SettingsPanelProps {
    isOpen: boolean;
    onClose: () => void;
  }

  let { isOpen, onClose }: SettingsPanelProps = $props();
  let panelElement = $state<HTMLElement | null>(null);
  let closeButton = $state<HTMLButtonElement | null>(null);
  let previouslyFocused = $state<HTMLElement | null>(null);

  $effect(() => {
    if (isOpen) {
      previouslyFocused = document.activeElement as HTMLElement;

      const timer = setTimeout(() => {
        closeButton?.focus();
      }, 200);

      return () => clearTimeout(timer);
    } else {
      if (previouslyFocused?.isConnected) {
        previouslyFocused.focus();
      }
    }
  });

  $effect(() => {
    if (!isOpen || !panelElement) return;

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Tab') {
        const focusable = getFocusableElements(panelElement);
        const first = focusable[0];
        const last = focusable[focusable.length - 1];

        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    }

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  });

  function getFocusableElements(container: HTMLElement): HTMLElement[] {
    const selector = 'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])';
    return Array.from(container.querySelectorAll(selector))
      .filter(el => !el.hasAttribute('disabled') && el.offsetParent !== null) as HTMLElement[];
  }
</script>
```

---

## 8. Testing Requirements

### 8.1 Unit Test Cases

| Test Case | Input | Expected Output |
|:----------|:------|:----------------|
| Panel opens | `isOpen={true}` | Panel rendered, focus on close button |
| Panel closes on Escape | Press Escape key | `onClose` called |
| Panel closes on backdrop click | Click backdrop | `onClose` called |
| Theme change | Select "Dark" option | `setTheme('dark')` called |
| Search mode change | Select "Multi" | Preference updated, UI reflects change |
| Macro toggle change | Toggle "Calories" off | Preference updated, toggle unchecked |
| Sort change | Select "Name (A-Z)" | Preference updated, select shows new value |
| Results per page change | Select "20" | Preference updated |
| Clear query cache | Click "Clear Search Cache" | Confirmation shown |
| Confirm clear query cache | Click "Clear Cache" in dialog | localStorage cache cleared, toast shown |
| Clear image cache | Click "Clear Cached Images" | Confirmation shown (if SW available) |
| Confirm clear image cache | Click "Clear Images" in dialog | SW cache cleared, toast shows count |
| Image cache unavailable | No Service Worker | Button disabled or hidden |
| Cancel clear cache | Click "Cancel" in dialog | Dialog closed, no action |
| Clear all data | Confirm in dialog | Data cleared, page reloads |
| Reset preferences | Confirm in dialog | Defaults restored |
| Storage usage display | Usage data loaded | Progress bar shows correct percentage |
| Focus trap | Tab from last element | Focus moves to first element |
| Focus restoration | Close panel | Focus returns to trigger element |

### 8.2 Integration Test Cases

| Test Case | Scenario | Expected Behavior |
|:----------|:---------|:------------------|
| Preferences persist | Change theme, refresh page | Theme restored |
| Storage usage accurate | Cache some queries | Usage reflects localStorage + SW cache |
| Clear query cache reduces usage | Clear query cache | localStorage usage decreases |
| Clear image cache reduces usage | Clear image cache | SW cache usage decreases |
| Clear all data clears both | Clear all data | Both localStorage and SW cache cleared |
| Theme integration | Change theme in panel | App theme changes immediately |
| SW unavailable fallback | No Service Worker registered | Image cache section hidden, no errors |
| Keyboard shortcut | Press Cmd+, | Panel opens |
| Open from sidebar | Click settings button | Panel opens with animation |
| Scroll to section | `open('storage')` | Panel opens, scrolls to storage section |

### 8.3 Accessibility Test Cases

| Test Case | Scenario | Expected Behavior |
|:----------|:---------|:------------------|
| Screen reader announcement | Open panel | "Settings dialog" announced |
| Toggle state | Toggle macro | "Checked"/"Unchecked" announced |
| Theme selection | Select theme | "Light selected" announced |
| Confirmation dialog | Open dialog | Dialog content announced |
| Reduced motion | User prefers reduced motion | No animations |
| Keyboard only | Navigate panel with keyboard | All controls accessible |

### 8.4 Edge Case Test Cases

| Test Case | Scenario | Expected Behavior |
|:----------|:---------|:------------------|
| Storage unavailable | Private browsing mode | Warning shown, changes per-session only |
| Storage quota exceeded | Try to save when full | Error toast, suggest clearing cache |
| Corrupted preferences | Invalid data in storage | Defaults loaded, data cleaned |
| Rapid toggle | Toggle same switch quickly | Final state correct, no race conditions |
| Multiple clear actions | Click clear during pending clear | Action ignored until first completes |
| Theme context missing | Panel without ThemeProvider | Theme section hidden gracefully |
| SW message timeout | Service Worker doesn't respond | Timeout after 10s, graceful error |
| SW not registered | First visit, SW not yet active | Image cache shows "not available" |
| SW cache calculation error | Cache API throws | Shows 0 bytes, logs warning |
| Clear all with SW failure | SW clear fails, localStorage succeeds | Partial success message, page reloads |

---

## Changelog

### 2026-01-22 (Rev 1.1)

**Updated:**
- Migrated from React to Svelte as per tech stack specification
- Replaced React hooks (`useState`, `useEffect`, `useCallback`, `useRef`) with Svelte runes (`$state`, `$effect`, `$derived`)
- Replaced `ReactNode` children with Svelte `Snippet` type for slot content
- Replaced `JSX.Element` return types with `void` (Svelte components don't return JSX)
- Updated component props interface to use `$props()` syntax
- Replaced `createContext` with Svelte's `setContext`/`getContext` using Svelte stores
- Updated `SettingsPanelProvider` to use Svelte context API
- Updated all component interfaces to use Svelte patterns
- Replaced `useTheme()` hook integration with Svelte store subscriptions
- Updated CSS implementation to use Tailwind classes per tech stack
- Updated integration examples to use Svelte `.svelte` file syntax

### 2026-01-22 (Rev 1.0)

**Added:**
- Initial detailed design document for SettingsPanel
- Slide-in panel with backdrop and focus management
- Appearance section with theme selector (light/dark)
- Search defaults section (mode, macro toggles, sort, results per page)
- Storage section with usage display and clear actions
- About section for app information
- Confirmation dialogs for destructive actions
- SettingsPanelProvider for global panel state management
- useSettingsPanel hook for opening/closing panel from anywhere
- Integration with LocalStorageManager for preferences persistence
- Integration with ThemeProvider for theme management
- Comprehensive CSS implementation with CSS variables
- Full accessibility implementation (ARIA, focus management, keyboard nav)
- Error handling with graceful degradation
- Performance optimizations
- Complete test case specifications

**Design Decisions:**
- Slide-in panel from right (common mobile/desktop pattern)
- Confirmation required for all clear/reset operations (prevent data loss)
- Storage usage always visible (transparency for user)
- Immediate feedback for all settings changes (no "Save" button)
- Reset preferences as separate action from clear all data
- Sections are scrollable, not accordion (all settings visible)
- Theme selector uses visual buttons, not dropdown (preview appearance)
- Keyboard shortcut Cmd/Ctrl+, for quick access (standard pattern)
- Separate clear actions for query cache vs image cache (user control)
- Service Worker cache operations use MessageChannel for reliable responses

**Alignment with LocalStorageManager (ARCH-001):**
- Uses `LocalStorageManager.getInstance()` singleton
- Consumes `UserPreferences` type for search defaults
- Uses `LocalStorageUsage` type for localStorage usage display
- Calls `invalidateQueryCache()` for query cache clear
- Calls `clearSearchHistory()` for history clear
- Calls `clearAllData()` for full localStorage reset
- Calls `resetUserPreferences()` for preferences reset
- Dispatches `mealswapp:storagecleared` event on clear all

**Alignment with Service Worker Cache (ARCH-011):**
- Queries Service Worker for Cache API usage via message passing
- Displays combined storage usage (localStorage + Cache API)
- Separate "Clear Cached Images" action for image cache only
- Clear All Data clears both localStorage and Service Worker caches
- Graceful degradation when Service Worker unavailable
- Service Worker message types defined: GET_CACHE_USAGE, CLEAR_IMAGE_CACHE, CLEAR_ALL_CACHES
- Timeout handling for unresponsive Service Worker (10s)

**Alignment with API Gateway (ARCH-010):**
- No direct integration required - SettingsPanel operates on client-side only
- Cleared cache will cause fresh requests through API Gateway on next search
