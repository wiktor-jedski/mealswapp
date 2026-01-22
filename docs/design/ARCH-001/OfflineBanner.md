# Detailed Design: OfflineBanner

**Traceability:** [ARCH-001]

---

## 1. Data Structures & Types

### 1.1 Network Status Types

```typescript
type NetworkStatus = 'online' | 'offline';

interface NetworkState {
  status: NetworkStatus;
  lastOnlineTimestamp: number | null;
  offlineDuration: number;
}
```

### 1.2 Banner Display Configuration

```typescript
type BannerVariant = 'default' | 'stale-data' | 'reconnecting';

interface OfflineBannerConfig {
  showDismissButton: boolean;
  showReconnectingState: boolean;
  staleDurationThreshold: number;
  animationDuration: number;
}

export const DEFAULT_CONFIG: OfflineBannerConfig = {
  showDismissButton: false,
  showReconnectingState: true,
  staleDurationThreshold: 300000,
  animationDuration: 200
};
```

### 1.3 Banner State

```typescript
interface OfflineBannerState {
  isVisible: boolean;
  variant: BannerVariant;
  isAnimating: boolean;
}
```

### 1.4 Storage Keys

```typescript
const LAST_ONLINE_STORAGE_KEY = 'mealswapp_last_online';
```

### 1.5 Svelte Store Types

```typescript
import { type Writable, writable, derived } from 'svelte/store';

interface NetworkStoreValue {
  status: NetworkStatus;
  isOffline: boolean;
  lastOnlineTimestamp: number | null;
  offlineDuration: number;
}

function createNetworkStore() {
  const { subscribe, set, update }: Writable<NetworkStoreValue> = writable({
    status: 'online',
    isOffline: false,
    lastOnlineTimestamp: null,
    offlineDuration: 0
  });

  let offlineDurationInterval: ReturnType<typeof setInterval> | null = null;

  function startOfflineTimer() {
    if (offlineDurationInterval) return;
    offlineDurationInterval = setInterval(() => {
      update(state => {
        if (state.lastOnlineTimestamp) {
          return {
            ...state,
            offlineDuration: Date.now() - state.lastOnlineTimestamp
          };
        }
        return state;
      });
    }, 1000);
  }

  function stopOfflineTimer() {
    if (offlineDurationInterval) {
      clearInterval(offlineDurationInterval);
      offlineDurationInterval = null;
    }
  }

  return {
    subscribe,
    setOnline: () => {
      const timestamp = Date.now();
      try {
        localStorage.setItem(LAST_ONLINE_STORAGE_KEY, String(timestamp));
      } catch {
        console.warn('Could not persist online timestamp');
      }
      stopOfflineTimer();
      set({
        status: 'online',
        isOffline: false,
        lastOnlineTimestamp: timestamp,
        offlineDuration: 0
      });
    },
    setOffline: () => {
      update(state => ({
        ...state,
        status: 'offline',
        isOffline: true
      }));
      startOfflineTimer();
    },
    initialize: (initialOnline: boolean, storedTimestamp: number | null) => {
      const timestamp = initialOnline ? Date.now() : storedTimestamp;
      set({
        status: initialOnline ? 'online' : 'offline',
        isOffline: !initialOnline,
        lastOnlineTimestamp: timestamp,
        offlineDuration: 0
      });
      if (!initialOnline && timestamp) {
        startOfflineTimer();
      }
    }
  };
}

export const networkStore = createNetworkStore();
export const isOffline = derived(networkStore, $n => $n.isOffline);
```

### 1.6 Component Props

```typescript
interface OfflineBannerProps {
  position?: 'top' | 'bottom';
  className?: string;
  zIndex?: number;
}

interface NetworkProviderProps {
  children: snippet;
  onStatusChange?: (status: NetworkStatus) => void;
  pollInterval?: number;
}
```

### 1.7 CSS Custom Properties (consumed from ThemeProvider)

```typescript
const CONSUMED_CSS_VARS = {
  backgroundColor: '--color-warning',
  textColor: '--text-inverse',
  iconColor: '--text-inverse',
  borderColor: '--color-warning',
  shadowColor: '--shadow-color'
};
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Network Status Detection

```
FUNCTION initializeNetworkDetection():
  1. Check initial network status
     1.1. IF navigator.onLine exists:
          initialStatus = navigator.onLine ? 'online' : 'offline'
     1.2. ELSE:
          initialStatus = 'online'
          Log warning: 'navigator.onLine not supported'

  2. Register browser event listeners
     2.1. window.addEventListener('online', handleOnline)
     2.2. window.addEventListener('offline', handleOffline)

  3. Start backup polling (for edge cases where events don't fire)
     3.1. IF pollInterval > 0:
          pollIntervalId = setInterval(checkNetworkStatus, pollInterval)

  4. Load persisted last-online timestamp
     4.1. TRY:
          stored = localStorage.getItem(LAST_ONLINE_STORAGE_KEY)
          IF stored:
            lastOnlineTimestamp = parseInt(stored, 10)
     4.2. CATCH:
          lastOnlineTimestamp = null

  5. IF initialStatus === 'online':
     5.1. Update lastOnlineTimestamp to Date.now()
     5.2. Persist to localStorage

  6. Initialize networkStore
     networkStore.initialize(initialStatus === 'online', lastOnlineTimestamp)

  RETURN pollIntervalId
```

### 2.2 Online Event Handler

```
FUNCTION handleOnline():
  1. Update store state
     networkStore.setOnline()

  2. Notify listeners
     IF previousStatus === 'offline':
       2.1. Trigger banner hide animation
       2.2. IF onStatusChange callback:
            CALL onStatusChange('online')
       2.3. Dispatch custom event:
            window.dispatchEvent(new CustomEvent('mealswapp:networkchange', {
              detail: { status: 'online', previousStatus: 'offline' }
            }))
```

### 2.3 Offline Event Handler

```
FUNCTION handleOffline():
  1. Update store state
     networkStore.setOffline()

  2. Notify listeners
     IF previousStatus === 'online':
       2.1. Trigger banner show animation
       2.2. IF onStatusChange callback:
            CALL onStatusChange('offline')
       2.3. Dispatch custom event:
            window.dispatchEvent(new CustomEvent('mealswapp:networkchange', {
              detail: { status: 'offline', previousStatus: 'online' }
            }))
```

### 2.4 Backup Network Polling

```
FUNCTION checkNetworkStatus():
  1. IF navigator.onLine exists:
     currentOnline = navigator.onLine
  ELSE:
     RETURN

  2. Compare with current state
     currentState = get(networkStore)
     IF currentOnline AND currentState.status === 'offline':
       CALL handleOnline()
     ELSE IF NOT currentOnline AND currentState.status === 'online':
       CALL handleOffline()
```

### 2.5 Banner Visibility Logic

```
FUNCTION determineBannerState(networkState: NetworkStoreValue, config: OfflineBannerConfig): OfflineBannerState
  1. Determine visibility
     isVisible = networkState.status === 'offline'

  2. Determine variant based on offline duration
     IF NOT isVisible:
       variant = 'default'
     ELSE IF networkState.offlineDuration >= config.staleDurationThreshold:
       variant = 'stale-data'
     ELSE:
       variant = 'default'

  3. RETURN { isVisible, variant, isAnimating: false }
```

### 2.6 Banner Animation Flow

```
FUNCTION showBanner():
  1. Set initial state (banner positioned off-screen)
     bannerElement.style.transform = 'translateY(-100%)'

  2. Make visible in DOM
     isVisible = true
     isAnimating = true

  3. Trigger animation frame
     requestAnimationFrame(() => {
       bannerElement.style.transform = 'translateY(0)'
     })

  4. Wait for animation completion
     setTimeout(() => {
       isAnimating = false
     }, config.animationDuration)

FUNCTION hideBanner():
  1. Start exit animation
     isAnimating = true
     bannerElement.style.transform = 'translateY(-100%)'

  2. Wait for animation completion
     setTimeout(() => {
       isVisible = false
       isAnimating = false
     }, config.animationDuration)
```

### 2.7 Staleness Duration Formatting

```
FUNCTION formatOfflineDuration(durationMs: number): string
  1. Convert to appropriate unit
     seconds = Math.floor(durationMs / 1000)
     minutes = Math.floor(seconds / 60)
     hours = Math.floor(minutes / 60)

  2. Return human-readable string
     IF hours > 0:
       RETURN `${hours}h ${minutes % 60}m`
     ELSE IF minutes > 0:
       RETURN `${minutes}m`
     ELSE:
       RETURN `${seconds}s`
```

### 2.8 NetworkProvider Implementation (Svelte)

```svelte
<script lang="typescript">
  import { onMount, onDestroy } from 'svelte';
  import { networkStore } from './networkStore';

  let { children, onStatusChange, pollInterval = 30000 } = $props();

  let pollIntervalId: ReturnType<typeof setInterval> | null = null;
  let previousStatus: NetworkStatus | null = null;

  function handleOnline() {
    const prev = previousStatus;
    previousStatus = 'online';
    networkStore.setOnline();
    if (prev === 'offline' && onStatusChange) {
      onStatusChange('online');
    }
    window.dispatchEvent(new CustomEvent('mealswapp:networkchange', {
      detail: { status: 'online', previousStatus: 'offline' }
    }));
  }

  function handleOffline() {
    const prev = previousStatus;
    previousStatus = 'offline';
    networkStore.setOffline();
    if (prev === 'online' && onStatusChange) {
      onStatusChange('offline');
    }
    window.dispatchEvent(new CustomEvent('mealswapp:networkchange', {
      detail: { status: 'offline', previousStatus: 'online' }
    }));
  }

  function checkNetworkStatus() {
    const current = $networkStore.status;
    const online = navigator.onLine;
    if (online && current === 'offline') {
      handleOnline();
    } else if (!online && current === 'online') {
      handleOffline();
    }
  }

  onMount(() => {
    const stored = localStorage.getItem(LAST_ONLINE_STORAGE_KEY);
    const lastOnline = stored ? parseInt(stored, 10) : null;
    networkStore.initialize(navigator.onLine !== false, lastOnline);
    previousStatus = $networkStore.status;

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    if (pollInterval > 0) {
      pollIntervalId = setInterval(checkNetworkStatus, pollInterval);
    }
  });

  onDestroy(() => {
    window.removeEventListener('online', handleOnline);
    window.removeEventListener('offline', handleOffline);
    if (pollIntervalId) {
      clearInterval(pollIntervalId);
    }
  });
</script>

{@render children()}
```

### 2.9 OfflineBanner Component Implementation (Svelte)

```svelte
<script lang="typescript">
  import { networkStore } from '../stores/networkStore';
  import { DEFAULT_CONFIG } from '../config/offlineBanner';

  let { position = 'top', className = '', zIndex = 1000 }: OfflineBannerProps = $props();

  let isAnimating = $state(false);
  let isRendered = $state(false);

  let network = $derived($networkStore);

  $effect(() => {
    if (network.isOffline && !isRendered) {
      isRendered = true;
      isAnimating = true;
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          isAnimating = false;
        });
      });
    } else if (!network.isOffline && isRendered) {
      isAnimating = true;
      setTimeout(() => {
        isRendered = false;
        isAnimating = false;
      }, DEFAULT_CONFIG.animationDuration);
    }
  });

  let variant = $derived(
    network.offlineDuration >= DEFAULT_CONFIG.staleDurationThreshold
      ? 'stale-data'
      : 'default'
  );

  let translateDirection = $derived(position === 'top' ? '-100%' : '100%');
  let transform = $derived(
    isAnimating && !network.isOffline
      ? `translateY(${translateDirection})`
      : 'translateY(0)'
  );
</script>

{#if isRendered}
  <div
    role="alert"
    aria-live="polite"
    class="offline-banner offline-banner--{position} {className}"
    style:transform={transform}
    style:z-index={zIndex}
    style:transition="transform {DEFAULT_CONFIG.animationDuration}ms ease-out"
  >
    <svg aria-hidden="true" class="offline-banner__icon">
      <circle cx="12" cy="12" r="10" fill="currentColor" />
      <line x1="4.93" y1="4.93" x2="19.07" y2="19.07" stroke="white" stroke-width="2" />
    </svg>
    <span class="offline-banner__text">
      {#if variant === 'stale-data'}
        You're offline. Showing cached data from {formatOfflineDuration(network.offlineDuration)} ago.
      {:else}
        You're offline. Showing cached data.
      {/if}
    </span>
  </div>
{/if}
```

### 2.10 useNetwork Helper (Svelte)

```typescript
import { networkStore } from '../stores/networkStore';

export function useNetwork() {
  return {
    get status() { return $networkStore.status; },
    get isOffline() { return $networkStore.isOffline; },
    get lastOnlineTimestamp() { return $networkStore.lastOnlineTimestamp; },
    get offlineDuration() { return $networkStore.offlineDuration; }
  };
}
```

---

## 3. State Management & Error Handling

### 3.1 State Transition Diagram

```
                     ┌─────────────────────────────────────────┐
                     │              INITIALIZING               │
                     │  (Check navigator.onLine, load stored)  │
                     └──────────────────┬──────────────────────┘
                                        │
                       ┌────────────────┴────────────────┐
                       │                                 │
                       ▼                                 ▼
              ┌────────────────┐                ┌────────────────┐
              │     ONLINE     │                │    OFFLINE     │
              │                │                │                │
              │ Banner: hidden │                │ Banner: visible│
              │ Duration: 0    │                │ Duration: ++   │
              └───────┬────────┘                └───────┬────────┘
                      │                                 │
                      │    'offline' event              │    'online' event
                      │    navigator.onLine=false       │    navigator.onLine=true
                      │                                 │
                      └────────────────┬────────────────┘
                                       │
                                       ▼
                              ┌────────────────┐
                              │ STATE_CHANGE   │
                              │ (Animate,      │
                              │  notify,       │
                              │  persist)      │
                              └────────────────┘
```

### 3.2 Banner Variant Transitions

```
                   ┌─────────────────┐
                   │     HIDDEN      │
                   │ (status=online) │
                   └────────┬────────┘
                            │
                            │ status changes to 'offline'
                            ▼
                   ┌─────────────────┐
                   │    DEFAULT      │
                   │ "You're offline"│
                   │ duration < 5min │
                   └────────┬────────┘
                            │
                            │ offlineDuration >= staleDurationThreshold
                            ▼
                   ┌─────────────────┐
                   │   STALE-DATA    │
                   │ "Showing cached │
                   │  data from Xm"  │
                   └────────┬────────┘
                            │
                            │ status changes to 'online'
                            ▼
                   ┌─────────────────┐
                   │     HIDDEN      │
                   │ (auto-dismiss)  │
                   └─────────────────┘
```

### 3.3 Error States

| Error State | Trigger | User Impact | Recovery Action |
|:------------|:--------|:------------|:----------------|
| **NAVIGATOR_ONLINE_UNAVAILABLE** | Browser doesn't support `navigator.onLine` | Offline detection may not work | Default to 'online', rely on API failure detection |
| **EVENT_LISTENER_FAILED** | Cannot add online/offline listeners | Banner won't auto-show/hide | Fall back to polling only |
| **STORAGE_UNAVAILABLE** | localStorage blocked | Last online timestamp not persisted | Track in memory only |
| **STORE_MISSING** | Component used outside NetworkProvider | Banner doesn't render | Show console warning |

### 3.4 Error Handling Implementation

```typescript
type NetworkErrorType =
  | 'NAVIGATOR_UNAVAILABLE'
  | 'EVENT_LISTENER_FAILED'
  | 'STORAGE_UNAVAILABLE'
  | 'STORE_MISSING';

interface NetworkError {
  type: NetworkErrorType;
  message: string;
  recoverable: boolean;
  fallbackBehavior: string;
}

function handleNetworkDetectionError(error: unknown): void {
  console.warn('[OfflineBanner] Network detection error:', error);
}

function safeAddEventListener(
  event: string,
  handler: EventListener
): boolean {
  try {
    window.addEventListener(event, handler);
    return true;
  } catch (error) {
    console.warn(`Failed to add ${event} listener: ${error}`);
    return false;
  }
}
```

### 3.5 Graceful Degradation

| Scenario | Degraded Behavior | Core Functionality |
|:---------|:------------------|:-------------------|
| **No navigator.onLine** | No automatic detection | Manual detection via API failures |
| **Events don't fire** | Backup polling detects changes | Slight delay in detection |
| **localStorage unavailable** | Offline duration starts from component mount | Banner displays correctly |
| **CSS transitions unsupported** | Banner appears/disappears instantly | Full functionality |

---

## 4. Component Interfaces

### 4.1 NetworkProvider Component (Svelte)

```svelte
<script lang="typescript">
  import type { Snippet } from 'svelte';

  interface NetworkProviderProps {
    children: Snippet;
    onStatusChange?: (status: NetworkStatus) => void;
    pollInterval?: number;
  }

  let { children, onStatusChange, pollInterval = 30000 }: NetworkProviderProps = $props();
</script>

<slot />
```

### 4.2 OfflineBanner Component (Svelte)

```svelte
<script lang="typescript">
  interface OfflineBannerProps {
    position?: 'top' | 'bottom';
    className?: string;
    zIndex?: number;
  }
</script>

<!-- Component implementation -->
```

### 4.3 Network Store (Svelte)

```typescript
import { derived, writable } from 'svelte/store';

export interface NetworkContextValue {
  status: NetworkStatus;
  isOffline: boolean;
  lastOnlineTimestamp: number | null;
  offlineDuration: number;
}

export const networkStore: Writable<NetworkContextValue>;
export const isOffline: Readable<boolean>;
```

### 4.4 Utility Functions (Exported)

```typescript
export function getNetworkStatus(): NetworkStatus;

export function formatOfflineDuration(durationMs: number): string;

export function isDataStale(
  lastOnlineTimestamp: number | null,
  threshold?: number
): boolean;
```

### 4.5 Event Types (for non-framework listeners)

```typescript
interface NetworkChangeEventDetail {
  status: NetworkStatus;
  previousStatus: NetworkStatus;
}

type NetworkChangeEvent = CustomEvent<NetworkChangeEventDetail>;
```

### 4.6 CSS Class Contract

```css
.offline-banner { }
.offline-banner--top { }
.offline-banner--bottom { }

.offline-banner--default { }
.offline-banner--stale-data { }

.offline-banner__icon { }
.offline-banner__text { }
.offline-banner__dismiss { }
```

---

## 5. Integration Requirements

### 5.1 Application Root Setup (Svelte)

```svelte
<script lang="typescript">
  import { NetworkProvider } from './providers/NetworkProvider';
  import { OfflineBanner } from './components/OfflineBanner';
</script>

<NetworkProvider
  onStatusChange={(status) => {
    if (status === 'offline') {
      console.log('user went offline');
    }
  }}
>
  <OfflineBanner position="top" />
  <slot />
</NetworkProvider>
```

### 5.2 CSS Implementation

```css
.offline-banner {
  position: fixed;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px 16px;
  background-color: var(--color-warning);
  color: var(--text-inverse);
  font-size: 14px;
  font-weight: 500;
  box-shadow: 0 2px 8px var(--shadow-color);
  transition: transform 200ms ease-out;
  z-index: 1000;
}

.offline-banner--top {
  top: 0;
}

.offline-banner--bottom {
  bottom: 0;
}

.offline-banner__icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.offline-banner__text {
  text-align: center;
}

body:has(.offline-banner--top[data-visible="true"]) {
  padding-top: 48px;
}

body:has(.offline-banner--bottom[data-visible="true"]) {
  padding-bottom: 48px;
}

@media (prefers-reduced-motion: reduce) {
  .offline-banner {
    transition: none;
  }
}
```

### 5.3 Usage in Other Components (Svelte)

```svelte
<script lang="typescript">
  import { networkStore } from '../stores/networkStore';
</script>

<div>
  <SearchInput />
  {#if $networkStore.isOffline}
    <p class="search-hint">
      Searching cached results only while offline.
    </p>
  {/if}
  <ResultsGrid />
</div>
```

### 5.4 Service Worker Coordination

```typescript
self.addEventListener('fetch', (event: FetchEvent) => {
  event.respondWith(
    caches.match(event.request).then((cachedResponse) => {
      if (cachedResponse && !navigator.onLine) {
        const headers = new Headers(cachedResponse.headers);
        headers.set('X-Served-From-Cache', 'true');
        return new Response(cachedResponse.body, {
          status: cachedResponse.status,
          statusText: cachedResponse.statusText,
          headers
        });
      }
      return fetch(event.request);
    })
  );
});
```

---

## 6. Performance Considerations

| Optimization | Implementation | Impact |
|:-------------|:---------------|:-------|
| **Event-driven detection** | Use native `online`/`offline` events | No polling overhead |
| **Minimal polling** | Backup poll only every 30s | Negligible CPU usage |
| **Conditional rendering** | `{#if}` block when online | No DOM nodes when not needed |
| **CSS transitions** | Hardware-accelerated transform | Smooth 60fps animation |
| **Svelte store derivation** | derived() for computed values | Efficient reactivity |
| **Lazy duration update** | 1s interval only when offline | No timer when online |

---

## 7. Accessibility Considerations

| Requirement | Implementation |
|:------------|:---------------|
| **Screen reader announcement** | `role="alert"` and `aria-live="polite"` |
| **No focus trap** | Banner is non-interactive |
| **Color contrast** | Warning background with inverse text meets WCAG AA |
| **Icon has alt text** | Icon marked `aria-hidden`, text provides context |
| **Reduced motion** | Respects `prefers-reduced-motion` media query |
| **No content obscuring** | Body padding adjusts when banner visible |

### 7.1 ARIA Implementation

```html
<div
  role="alert"
  aria-live="polite"
  aria-atomic="true"
  class="offline-banner"
>
  <svg aria-hidden="true" class="offline-banner__icon">...</svg>
  <span class="offline-banner__text">
    You're offline. Showing cached data.
  </span>
</div>
```

---

## 8. Testing Requirements

### 8.1 Unit Test Cases (Bun + @testing-library/svelte)

| Test Case | Input | Expected Output |
|:----------|:------|:----------------|
| Initial state online | `navigator.onLine = true` | `status='online'`, banner hidden |
| Initial state offline | `navigator.onLine = false` | `status='offline'`, banner visible |
| Online to offline transition | Dispatch 'offline' event | Banner appears with animation |
| Offline to online transition | Dispatch 'online' event | Banner hides with animation |
| Stale data variant | `offlineDuration >= 300000` | Banner shows stale-data message |
| Duration formatting (seconds) | `45000ms` | "45s" |
| Duration formatting (minutes) | `125000ms` | "2m" |
| Duration formatting (hours) | `7500000ms` | "2h 5m" |
| Store updates | Manual status change | Reactive UI update |

### 8.2 Integration Test Cases

| Test Case | Scenario | Expected Behavior |
|:----------|:---------|:------------------|
| Browser offline event | Disable network in DevTools | Banner appears within 100ms |
| Browser online event | Re-enable network | Banner disappears after animation |
| Persistence across refresh | Go offline, refresh while offline | Banner visible on load |
| Multiple rapid toggles | Toggle online/offline quickly | No animation glitches, correct final state |
| With Service Worker | Offline with cached data | Banner + cached content displayed |
| Screen reader | Go offline | "You're offline" announced |

### 8.3 E2E Test Cases (Playwright)

| Test Case | Steps | Expected Result |
|:----------|:------|:----------------|
| Full offline flow | 1. Load app, 2. Go offline, 3. Search, 4. Go online | Banner shows/hides, cached results work |
| Stale data warning | 1. Go offline, 2. Wait 5+ minutes, 3. Check banner | Shows duration in message |

---

## Changelog

### 2026-01-22 (Rev 1.0)

**Added:**
- Initial detailed design document for OfflineBanner
- Network status detection using browser `online`/`offline` events
- Backup polling mechanism for edge cases
- Svelte stores for app-wide network state
- Banner variants: default and stale-data
- Slide animation with reduced-motion support
- CSS implementation using ThemeProvider variables
- Integration with Service Worker caching (ARCH-011)
- Accessibility implementation with ARIA attributes
- Comprehensive test cases

**Updated 2026-01-22:**
- Migrated from React to Svelte per tech stack requirements
- Replaced React hooks with Svelte stores, $state, and $effect
- Updated component syntax from JSX to Svelte template syntax
- Replaced React context with Svelte writable/derived stores
- Updated type definitions for Svelte 5 patterns (snippets, $props, $state, $derived, $effect)

**Design Decisions:**
- Banner auto-hides on reconnect (no manual dismiss required)
- Stale data threshold set to 5 minutes to match typical cache TTL
- Position defaults to 'top' for consistency with common notification patterns
- Warning color used (not error) since offline is a degraded but functional state
