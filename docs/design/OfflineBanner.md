# Detailed Design: OfflineBanner

**Traceability:** [ARCH-001]

---

## 1. Data Structures & Types

### 1.1 Network Status Types

```typescript
type NetworkStatus = 'online' | 'offline';

interface NetworkState {
  status: NetworkStatus;
  lastOnlineTimestamp: number | null;  // Unix timestamp of last known online state
  offlineDuration: number;             // Milliseconds since going offline (0 if online)
}
```

### 1.2 Banner Display Configuration

```typescript
type BannerVariant = 'default' | 'stale-data' | 'reconnecting';

interface OfflineBannerConfig {
  showDismissButton: boolean;          // Default: false (banner auto-hides on reconnect)
  showReconnectingState: boolean;      // Default: true
  staleDurationThreshold: number;      // Milliseconds before showing stale warning (default: 300000 = 5min)
  animationDuration: number;           // Banner slide animation (default: 200ms)
}

const DEFAULT_CONFIG: OfflineBannerConfig = {
  showDismissButton: false,
  showReconnectingState: true,
  staleDurationThreshold: 300000,
  animationDuration: 200
};
```

### 1.3 Banner State

```typescript
interface OfflineBannerState {
  isVisible: boolean;                  // Whether banner is rendered and visible
  variant: BannerVariant;              // Current display variant
  isAnimating: boolean;                // Whether entry/exit animation is in progress
}
```

### 1.4 Storage Keys

```typescript
const LAST_ONLINE_STORAGE_KEY = 'mealswapp_last_online';
```

### 1.5 Context Types

```typescript
interface NetworkContextValue {
  status: NetworkStatus;               // Current network status
  isOffline: boolean;                  // Convenience: status === 'offline'
  lastOnlineTimestamp: number | null;  // When connection was last active
  offlineDuration: number;             // How long device has been offline
}

const NetworkContext = createContext<NetworkContextValue | null>(null);
```

### 1.6 Component Props

```typescript
interface OfflineBannerProps {
  position?: 'top' | 'bottom';         // Default: 'top'
  className?: string;                  // Additional CSS classes
  zIndex?: number;                     // Default: 1000
}

interface NetworkProviderProps {
  children: ReactNode;
  onStatusChange?: (status: NetworkStatus) => void;
  pollInterval?: number;               // Backup poll interval in ms (default: 30000)
}
```

### 1.7 CSS Custom Properties (consumed from ThemeProvider)

```typescript
// OfflineBanner uses these CSS variables from ThemeProvider
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
          initialStatus = 'online'  // Assume online if API unavailable
          Log warning: 'navigator.onLine not supported'

  2. Register browser event listeners
     2.1. window.addEventListener('online', handleOnline)
     2.2. window.addEventListener('offline', handleOffline)

  3. Start backup polling (for edge cases where events don't fire)
     3.1. IF pollInterval > 0:
          setInterval(checkNetworkStatus, pollInterval)

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

  RETURN { status: initialStatus, lastOnlineTimestamp }
```

### 2.2 Online Event Handler

```
FUNCTION handleOnline():
  1. Update state
     previousStatus = state.status
     state.status = 'online'
     state.lastOnlineTimestamp = Date.now()
     state.offlineDuration = 0

  2. Persist timestamp
     TRY:
       localStorage.setItem(LAST_ONLINE_STORAGE_KEY, String(Date.now()))
     CATCH:
       Log warning: 'Could not persist online timestamp'

  3. Notify listeners
     IF previousStatus === 'offline':
       3.1. Trigger banner hide animation
       3.2. IF onStatusChange callback:
            CALL onStatusChange('online')
       3.3. Dispatch custom event:
            window.dispatchEvent(new CustomEvent('mealswapp:networkchange', {
              detail: { status: 'online', previousStatus: 'offline' }
            }))
```

### 2.3 Offline Event Handler

```
FUNCTION handleOffline():
  1. Update state
     previousStatus = state.status
     state.status = 'offline'
     // lastOnlineTimestamp remains unchanged (tracks last known online time)

  2. Start offline duration timer
     IF NOT offlineDurationInterval:
       offlineDurationInterval = setInterval(() => {
         IF state.lastOnlineTimestamp:
           state.offlineDuration = Date.now() - state.lastOnlineTimestamp
       }, 1000)

  3. Notify listeners
     IF previousStatus === 'online':
       3.1. Trigger banner show animation
       3.2. IF onStatusChange callback:
            CALL onStatusChange('offline')
       3.3. Dispatch custom event:
            window.dispatchEvent(new CustomEvent('mealswapp:networkchange', {
              detail: { status: 'offline', previousStatus: 'online' }
            }))
```

### 2.4 Backup Network Polling

```
FUNCTION checkNetworkStatus():
  // Backup check in case browser events don't fire (rare edge cases)
  1. IF navigator.onLine exists:
     currentOnline = navigator.onLine
  ELSE:
     RETURN  // Cannot determine status without API

  2. Compare with current state
     IF currentOnline AND state.status === 'offline':
       CALL handleOnline()
     ELSE IF NOT currentOnline AND state.status === 'online':
       CALL handleOffline()
```

### 2.5 Banner Visibility Logic

```
FUNCTION determineBannerState(networkState: NetworkState, config: OfflineBannerConfig): OfflineBannerState
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
     bannerElement.style.transform = 'translateY(-100%)'  // For top position
     // OR 'translateY(100%)' for bottom position

  2. Make visible in DOM
     state.isVisible = true
     state.isAnimating = true

  3. Trigger animation frame
     requestAnimationFrame(() => {
       bannerElement.style.transform = 'translateY(0)'
     })

  4. Wait for animation completion
     setTimeout(() => {
       state.isAnimating = false
     }, config.animationDuration)

FUNCTION hideBanner():
  1. Start exit animation
     state.isAnimating = true
     bannerElement.style.transform = 'translateY(-100%)'  // Slide up for top
     // OR 'translateY(100%)' for bottom

  2. Wait for animation completion
     setTimeout(() => {
       state.isVisible = false
       state.isAnimating = false
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

### 2.8 NetworkProvider Implementation

```
FUNCTION NetworkProvider(props: NetworkProviderProps):
  1. Initialize state
     [networkState, setNetworkState] = useState<NetworkState>(() => {
       return initializeNetworkDetection()
     })

  2. Set up event listeners
     useEffect(() => {
       handleOnline = () => {
         setNetworkState(prev => ({
           status: 'online',
           lastOnlineTimestamp: Date.now(),
           offlineDuration: 0
         }))
         props.onStatusChange?.('online')
       }

       handleOffline = () => {
         setNetworkState(prev => ({
           ...prev,
           status: 'offline'
         }))
         props.onStatusChange?.('offline')
       }

       window.addEventListener('online', handleOnline)
       window.addEventListener('offline', handleOffline)

       // Backup polling
       pollIntervalId = null
       IF props.pollInterval && props.pollInterval > 0:
         pollIntervalId = setInterval(checkNetworkStatus, props.pollInterval)

       // Cleanup
       RETURN () => {
         window.removeEventListener('online', handleOnline)
         window.removeEventListener('offline', handleOffline)
         IF pollIntervalId:
           clearInterval(pollIntervalId)
       }
     }, [props.onStatusChange, props.pollInterval])

  3. Update offline duration
     useEffect(() => {
       IF networkState.status === 'offline' AND networkState.lastOnlineTimestamp:
         intervalId = setInterval(() => {
           setNetworkState(prev => ({
             ...prev,
             offlineDuration: Date.now() - (prev.lastOnlineTimestamp || Date.now())
           }))
         }, 1000)

         RETURN () => clearInterval(intervalId)
     }, [networkState.status, networkState.lastOnlineTimestamp])

  4. Create context value
     contextValue = useMemo(() => ({
       status: networkState.status,
       isOffline: networkState.status === 'offline',
       lastOnlineTimestamp: networkState.lastOnlineTimestamp,
       offlineDuration: networkState.offlineDuration
     }), [networkState])

  5. Render provider
     RETURN (
       <NetworkContext.Provider value={contextValue}>
         {props.children}
       </NetworkContext.Provider>
     )
```

### 2.9 OfflineBanner Component Implementation

```
FUNCTION OfflineBanner(props: OfflineBannerProps):
  1. Get network context
     network = useNetwork()
     IF network === null:
       THROW Error('OfflineBanner must be used within NetworkProvider')

  2. Local state for animation
     [isAnimating, setIsAnimating] = useState(false)
     [isRendered, setIsRendered] = useState(false)

  3. Handle visibility changes
     useEffect(() => {
       IF network.isOffline AND NOT isRendered:
         // Show banner
         setIsRendered(true)
         setIsAnimating(true)
         requestAnimationFrame(() => {
           requestAnimationFrame(() => {
             setIsAnimating(false)
           })
         })
       ELSE IF NOT network.isOffline AND isRendered:
         // Hide banner with animation
         setIsAnimating(true)
         setTimeout(() => {
           setIsRendered(false)
           setIsAnimating(false)
         }, DEFAULT_CONFIG.animationDuration)
     }, [network.isOffline])

  4. Determine variant
     variant = useMemo(() => {
       IF network.offlineDuration >= DEFAULT_CONFIG.staleDurationThreshold:
         RETURN 'stale-data'
       RETURN 'default'
     }, [network.offlineDuration])

  5. Render nothing if not visible
     IF NOT isRendered:
       RETURN null

  6. Compute styles
     position = props.position || 'top'
     translateDirection = position === 'top' ? '-100%' : '100%'
     transform = isAnimating AND NOT network.isOffline
                 ? `translateY(${translateDirection})`
                 : 'translateY(0)'

  7. Render banner
     RETURN (
       <div
         role="alert"
         aria-live="polite"
         className={classNames('offline-banner', `offline-banner--${position}`, props.className)}
         style={{
           transform,
           zIndex: props.zIndex || 1000,
           transition: `transform ${DEFAULT_CONFIG.animationDuration}ms ease-out`
         }}
       >
         <OfflineIcon aria-hidden="true" />
         <span className="offline-banner__text">
           {variant === 'stale-data'
             ? `You're offline. Showing cached data from ${formatOfflineDuration(network.offlineDuration)} ago.`
             : "You're offline. Showing cached data."}
         </span>
       </div>
     )
```

### 2.10 useNetwork Hook

```
FUNCTION useNetwork(): NetworkContextValue
  1. Get context
     context = useContext(NetworkContext)

  2. Validate context exists
     IF context === null:
       THROW Error('useNetwork must be used within a NetworkProvider')

  3. RETURN context
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
| **CONTEXT_MISSING** | Component used outside NetworkProvider | Banner doesn't render | Throw error with clear message |

### 3.4 Error Handling Implementation

```typescript
type NetworkErrorType =
  | 'NAVIGATOR_UNAVAILABLE'
  | 'EVENT_LISTENER_FAILED'
  | 'STORAGE_UNAVAILABLE'
  | 'CONTEXT_MISSING';

interface NetworkError {
  type: NetworkErrorType;
  message: string;
  recoverable: boolean;
  fallbackBehavior: string;
}

FUNCTION handleNetworkDetectionError(error: unknown): NetworkState
  1. Log error for debugging
     console.warn('[OfflineBanner] Network detection error:', error)

  2. Return safe fallback state
     RETURN {
       status: 'online',          // Assume online by default
       lastOnlineTimestamp: null,
       offlineDuration: 0
     }

FUNCTION safeAddEventListener(
  event: string,
  handler: EventListener
): boolean
  TRY:
    window.addEventListener(event, handler)
    RETURN true
  CATCH (error):
    Log warning: `Failed to add ${event} listener: ${error}`
    RETURN false
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

### 4.1 NetworkProvider Component

```typescript
interface NetworkProviderProps {
  children: ReactNode;
  /** Callback fired when network status changes */
  onStatusChange?: (status: NetworkStatus) => void;
  /** Backup polling interval in ms. Set to 0 to disable. Default: 30000 */
  pollInterval?: number;
}

function NetworkProvider(props: NetworkProviderProps): JSX.Element;
```

### 4.2 OfflineBanner Component

```typescript
interface OfflineBannerProps {
  /** Banner position. Default: 'top' */
  position?: 'top' | 'bottom';
  /** Additional CSS classes */
  className?: string;
  /** Z-index for stacking. Default: 1000 */
  zIndex?: number;
}

function OfflineBanner(props: OfflineBannerProps): JSX.Element | null;
```

### 4.3 useNetwork Hook

```typescript
interface NetworkContextValue {
  /** Current network status */
  status: NetworkStatus;
  /** Convenience: true when offline */
  isOffline: boolean;
  /** Unix timestamp of last known online state, null if never online */
  lastOnlineTimestamp: number | null;
  /** Milliseconds since going offline (0 when online) */
  offlineDuration: number;
}

function useNetwork(): NetworkContextValue;
```

### 4.4 Utility Functions (Exported)

```typescript
/**
 * Check current network status synchronously.
 * Falls back to 'online' if navigator.onLine unavailable.
 */
function getNetworkStatus(): NetworkStatus;

/**
 * Format offline duration for display.
 * @example formatOfflineDuration(125000) => "2m"
 */
function formatOfflineDuration(durationMs: number): string;

/**
 * Check if cached data should be considered stale.
 * @param lastOnlineTimestamp - When data was last refreshed
 * @param threshold - Staleness threshold in ms (default: 300000)
 */
function isDataStale(
  lastOnlineTimestamp: number | null,
  threshold?: number
): boolean;
```

### 4.5 Event Types (for non-React listeners)

```typescript
interface NetworkChangeEventDetail {
  status: NetworkStatus;
  previousStatus: NetworkStatus;
}

// Usage: window.addEventListener('mealswapp:networkchange', handler)
type NetworkChangeEvent = CustomEvent<NetworkChangeEventDetail>;
```

### 4.6 CSS Class Contract

```css
/* Base banner classes */
.offline-banner { }
.offline-banner--top { }
.offline-banner--bottom { }

/* Variant modifiers */
.offline-banner--default { }
.offline-banner--stale-data { }

/* Internal elements */
.offline-banner__icon { }
.offline-banner__text { }
.offline-banner__dismiss { }
```

---

## 5. Integration Requirements

### 5.1 Application Root Setup

```typescript
// App.tsx
import { NetworkProvider } from './providers/NetworkProvider';
import { OfflineBanner } from './components/OfflineBanner';

function App({ children }) {
  return (
    <ThemeProvider>
      <NetworkProvider
        onStatusChange={(status) => {
          // Optional: analytics or Service Worker notification
          if (status === 'offline') {
            analytics.track('user_went_offline');
          }
        }}
      >
        <OfflineBanner position="top" />
        {children}
      </NetworkProvider>
    </ThemeProvider>
  );
}
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

/* Ensure banner doesn't interfere with main content */
body:has(.offline-banner--top[data-visible="true"]) {
  padding-top: 48px;
}

body:has(.offline-banner--bottom[data-visible="true"]) {
  padding-bottom: 48px;
}

/* Reduced motion support */
@media (prefers-reduced-motion: reduce) {
  .offline-banner {
    transition: none;
  }
}
```

### 5.3 Usage in Other Components

```typescript
// SearchView.tsx - conditionally show offline hint
function SearchView() {
  const { isOffline } = useNetwork();

  return (
    <div>
      <SearchInput />
      {isOffline && (
        <p className="search-hint">
          Searching cached results only while offline.
        </p>
      )}
      <ResultsGrid />
    </div>
  );
}
```

### 5.4 Service Worker Coordination

```typescript
// In Service Worker (sw.ts)
// Notify when serving cached responses

self.addEventListener('fetch', (event) => {
  event.respondWith(
    caches.match(event.request).then((cachedResponse) => {
      if (cachedResponse && !navigator.onLine) {
        // Add header to indicate cached response
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
| **Conditional rendering** | Return `null` when online | No DOM nodes when not needed |
| **CSS transitions** | Hardware-accelerated transform | Smooth 60fps animation |
| **Memoized context** | `useMemo` for context value | Prevent cascading re-renders |
| **Lazy duration update** | 1s interval only when offline | No timer when online |

---

## 7. Accessibility Considerations

| Requirement | Implementation |
|:------------|:---------------|
| **Screen reader announcement** | `role="alert"` and `aria-live="polite"` |
| **No focus trap** | Banner is non-interactive (no dismiss button by default) |
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

### 8.1 Unit Test Cases

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
| Context missing | useNetwork outside provider | Throws error |

### 8.2 Integration Test Cases

| Test Case | Scenario | Expected Behavior |
|:----------|:---------|:------------------|
| Browser offline event | Disable network in DevTools | Banner appears within 100ms |
| Browser online event | Re-enable network | Banner disappears after animation |
| Persistence across refresh | Go offline, refresh while offline | Banner visible on load |
| Multiple rapid toggles | Toggle online/offline quickly | No animation glitches, correct final state |
| With Service Worker | Offline with cached data | Banner + cached content displayed |
| Screen reader | Go offline | "You're offline" announced |

### 8.3 E2E Test Cases

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
- NetworkProvider context for app-wide network state
- Banner variants: default and stale-data
- Slide animation with reduced-motion support
- CSS implementation using ThemeProvider variables
- useNetwork hook for consuming network state
- Integration with Service Worker caching (ARCH-011)
- Accessibility implementation with ARIA attributes
- Comprehensive test cases

**Design Decisions:**
- Banner auto-hides on reconnect (no manual dismiss required)
- Stale data threshold set to 5 minutes to match typical cache TTL
- Position defaults to 'top' for consistency with common notification patterns
- Warning color used (not error) since offline is a degraded but functional state
