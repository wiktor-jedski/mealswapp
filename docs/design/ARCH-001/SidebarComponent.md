# Detailed Design: SidebarComponent

**Traceability:** [ARCH-001]

---

## 1. Data Structures & Types

### 1. Navigation Types

```typescript
import type { Writable } from 'svelte/store';

type NavigationRoute = 'search' | 'saved' | 'history' | 'profile' | 'settings';

interface NavigationItem {
  route: NavigationRoute;
  label: string;
  icon: string;
  requiresAuth: boolean;
  requiresPaidTier: boolean;
  badge?: NavigationBadge;
}

interface NavigationBadge {
  count: number;              // Number to display (0 = hidden)
  type: 'info' | 'warning';   // Badge color/style
}

const NAVIGATION_ITEMS: NavigationItem[] = [
  {
    route: 'search',
    label: 'Search',
    icon: 'search',
    requiresAuth: false,
    requiresPaidTier: false
  },
  {
    route: 'saved',
    label: 'Saved Items',
    icon: 'bookmark',
    requiresAuth: true,
    requiresPaidTier: false
  },
  {
    route: 'history',
    label: 'History',
    icon: 'clock',
    requiresAuth: false,
    requiresPaidTier: false
  },
  {
    route: 'profile',
    label: 'Profile',
    icon: 'user',
    requiresAuth: true,
    requiresPaidTier: false
  },
  {
    route: 'settings',
    label: 'Settings',
    icon: 'settings',
    requiresAuth: false,
    requiresPaidTier: false
  }
];
```

### 1.2 User Summary Types

```typescript
import { writable } from 'svelte/store';

type SubscriptionTier = 'free' | 'trial' | 'paid';

interface UserSummary {
  isAuthenticated: boolean;
  userId: string | null;
  displayName: string | null;
  email: string | null;
  avatarUrl: string | null;
  subscriptionTier: SubscriptionTier;
  trialDaysRemaining: number | null;
  searchesRemaining: number | null;
}

export const userStore = writable<UserSummary>({
  isAuthenticated: false,
  userId: null,
  displayName: null,
  email: null,
  avatarUrl: null,
  subscriptionTier: 'free',
  trialDaysRemaining: null,
  searchesRemaining: 3
});
```

### 1.3 Sidebar State Types

```typescript
import { writable } from 'svelte/store';

type SidebarDisplayMode = 'expanded' | 'collapsed' | 'hidden';

interface SidebarState {
  displayMode: SidebarDisplayMode;
  activeRoute: NavigationRoute;
  isMobileMenuOpen: boolean;
  isUserMenuOpen: boolean;
}

export const sidebarStore = writable<SidebarState>({
  displayMode: 'expanded',
  activeRoute: 'search',
  isMobileMenuOpen: false,
  isUserMenuOpen: false
});

export const activeRouteStore = writable<NavigationRoute>('search');
export const themeStore = writable<'light' | 'dark'>('light');
export const isOnlineStore = writable<boolean>(navigator.onLine);
export const pendingSyncCountStore = writable<number>(0);
```

### 1.4 Theme Toggle Types

```typescript
type Theme = 'light' | 'dark';

interface ThemeToggleConfig {
  value: Theme;
  label: string;
  icon: string;
}

const THEME_OPTIONS: ThemeToggleConfig[] = [
  { value: 'light', label: 'Light', icon: 'sun' },
  { value: 'dark', label: 'Dark', icon: 'moon' }
];

// Note: Theme persistence handled by ThemeProvider
// System preference used only for first-visit default
```

### 1.5 Quick Actions Types

```typescript
interface QuickAction {
  id: string;
  type: 'recent_search' | 'saved_item' | 'favorite';
  label: string;
  subtitle?: string;
  timestamp: number;
  data: QuickActionData;
}

type QuickActionData =
  | { type: 'recent_search'; query: string; mode: 'single' | 'recipe' | 'diet' }
  | { type: 'saved_item'; itemId: string; itemName: string }
  | { type: 'favorite'; itemId: string; itemName: string };

const MAX_QUICK_ACTIONS = 5;
const QUICK_ACTIONS_STORAGE_KEY = 'mealswapp_quick_actions';

// Using Svelte writable store for reactive updates
export const quickActionsStore = writable<QuickAction[]>([]);
```

### 1.6 Responsive Breakpoint Types

```typescript
interface ResponsiveConfig {
  breakpoint: number;
  displayMode: SidebarDisplayMode;
}

const RESPONSIVE_BREAKPOINTS: ResponsiveConfig[] = [
  { breakpoint: 1024, displayMode: 'expanded' },   // >= 1024px: full sidebar
  { breakpoint: 768, displayMode: 'collapsed' },   // 768-1023px: icon-only
  { breakpoint: 0, displayMode: 'hidden' }         // < 768px: hamburger menu
];

const SIDEBAR_WIDTH_EXPANDED = 256;  // px
const SIDEBAR_WIDTH_COLLAPSED = 64;  // px
```

### 1.7 User Menu Actions Types

```typescript
type UserMenuAction = 'view_profile' | 'manage_subscription' | 'export_data' | 'logout';

interface UserMenuOption {
  action: UserMenuAction;
  label: string;
  icon: string;
  requiresAuth: boolean;
  isDanger: boolean;       // For destructive actions (red styling)
}

const USER_MENU_OPTIONS: UserMenuOption[] = [
  {
    action: 'view_profile',
    label: 'View Profile',
    icon: 'user',
    requiresAuth: true,
    isDanger: false
  },
  {
    action: 'manage_subscription',
    label: 'Manage Subscription',
    icon: 'credit-card',
    requiresAuth: true,
    isDanger: false
  },
  {
    action: 'export_data',
    label: 'Export My Data',
    icon: 'download',
    requiresAuth: true,
    isDanger: false
  },
  {
    action: 'logout',
    label: 'Log Out',
    icon: 'log-out',
    requiresAuth: true,
    isDanger: true
  }
];
```

### 1.8 Offline Indicator Types

```typescript
interface OfflineIndicatorState {
  isVisible: boolean;
  message: string;
  showSyncStatus: boolean;
  pendingChanges: number;
}

const OFFLINE_MESSAGES = {
  offline: 'You are offline',
  syncing: 'Syncing changes...',
  syncComplete: 'All changes synced',
  syncFailed: 'Sync failed. Will retry.'
};
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Initialization Flow

```
ON SidebarComponent Mount (onMount):
  1. Determine initial display mode based on viewport width
     1.1. Get current window.innerWidth
     1.2. Find matching breakpoint from RESPONSIVE_BREAKPOINTS
     1.3. Update $sidebarStore.displayMode

  2. Subscribe to theme store
     2.1. Subscribe to theme store: $themeStore
     2.2. Update local state: state.theme = $themeStore
     // Theme store handles persistence and DOM updates

  3. Load user session (if authenticated)
     3.1. Check for existing auth token in HttpOnly cookie (via API call)
     3.2. IF authenticated:
          - Call GET /api/v1/user/me using TanStack Query
          - Update $userStore with response data
     3.3. ELSE:
          - Set $userStore = ANONYMOUS_USER

  4. Load quick actions from localStorage
     4.1. Read localStorage key: QUICK_ACTIONS_STORAGE_KEY
     4.2. Parse and validate JSON array
     4.3. Filter out items older than 7 days
     4.4. Keep only MAX_QUICK_ACTIONS items
     4.5. Update $quickActionsStore

  5. Register event listeners
     5.1. Add resize listener: window.addEventListener('resize', handleResize)
     5.2. Add online listener: window.addEventListener('online', handleOnline)
     5.3. Add offline listener: window.addEventListener('offline', handleOffline)
     5.4. Add click-outside listener for dropdown menus (via svelte:window)

  6. Check initial online status
     6.1. Set $isOnlineStore = navigator.onLine
     6.2. IF offline:
          - Load pending sync count from localStorage

  7. Determine active route from current URL
     7.1. Parse window.location.pathname
     7.2. Match against NAVIGATION_ITEMS routes
     7.3. Update $activeRouteStore = matchedRoute OR 'search'

  RETURN cleanup function to remove event listeners on destroy
```

### 2.2 Navigation Handling

```
FUNCTION handleNavigation(route: NavigationRoute):
  1. Check authentication requirement
     IF NAVIGATION_ITEMS[route].requiresAuth AND NOT $userStore.isAuthenticated:
       1.1. Store intended route in sessionStorage: 'mealswapp_redirect_after_login'
       1.2. Dispatch 'authRequired' event
       1.3. RETURN (do not navigate)

  2. Check subscription requirement
     IF NAVIGATION_ITEMS[route].requiresPaidTier:
       IF $userStore.subscriptionTier === 'free':
         2.1. Show upgrade modal
         2.2. RETURN (do not navigate)

  3. Update active route
     $activeRouteStore = route

  4. Close mobile menu if open
     IF $sidebarStore.isMobileMenuOpen:
       $sidebarStore.isMobileMenuOpen = false

  5. Navigate to route
     5.1. Push to browser history: history.pushState(null, '', `/${route}`)
     5.2. Dispatch 'navigationChange' event with route

  6. Update document title
     document.title = `${NAVIGATION_ITEMS[route].label} | Mealswapp`

  7. Scroll to top of main content
     document.getElementById('main-content')?.scrollTo(0, 0)
```

### 2.3 Theme Toggle Handling

```
NOTE: Theme management is handled by theme store (see themeStore.ts).
SidebarComponent subscribes to theme store and provides UI controls.

FUNCTION handleThemeToggle():
  1. Get current theme from store
     currentTheme = $themeStore

  2. Toggle theme
     $themeStore = currentTheme === 'light' ? 'dark' : 'light'
     // Theme store handles:
     // - CSS variable updates
     // - DOM attribute updates
     // - localStorage persistence
     // - Event dispatching

FUNCTION handleThemeSelect(newTheme: Theme):
  1. Set theme store
     $themeStore = newTheme

// Theme state is accessed from store:
// $themeStore reflects current value for UI rendering
```

### 2.4 Responsive Handling

```
FUNCTION handleResize():
  1. Get current viewport width
     width = window.innerWidth

  2. Determine appropriate display mode
     newMode = 'expanded'  // default
     FOR breakpoint IN RESPONSIVE_BREAKPOINTS (sorted descending):
       IF width >= breakpoint.breakpoint:
         newMode = breakpoint.displayMode
         BREAK

  3. IF newMode !== $sidebarStore.displayMode:
     3.1. Update store: $sidebarStore.displayMode = newMode
     3.2. IF newMode === 'hidden':
           - Ensure mobile menu is closed: $sidebarStore.isMobileMenuOpen = false
     3.3. Dispatch 'displayModeChange' event with newMode

  4. Update main content margin (via CSS class)
     - Use Tailwind classes based on displayMode
     - expanded: ml-64 (256px)
     - collapsed: ml-16 (64px)
     - hidden: ml-0
```

### 2.5 Mobile Menu Toggle

```
FUNCTION toggleMobileMenu():
  1. Toggle store state
     $sidebarStore.isMobileMenuOpen = !$sidebarStore.isMobileMenuOpen

  2. Handle body scroll
     IF $sidebarStore.isMobileMenuOpen:
       2.1. Add 'overflow-hidden' class to body
       2.2. Set focus trap within sidebar
     ELSE:
       2.1. Remove 'overflow-hidden' class from body
       2.2. Release focus trap
       2.3. Return focus to hamburger button

  3. Animate menu
     IF $sidebarStore.isMobileMenuOpen:
       - Slide in from left with overlay (Tailwind transitions)
     ELSE:
       - Slide out to left, fade overlay

FUNCTION handleMobileOverlayClick():
  1. Close mobile menu
     $sidebarStore.isMobileMenuOpen = false

  2. Restore body scroll
     Remove 'overflow-hidden' class from body
```

### 2.6 User Menu Handling

```
FUNCTION toggleUserMenu():
  1. Toggle store state
     $sidebarStore.isUserMenuOpen = !$sidebarStore.isUserMenuOpen

  2. IF $sidebarStore.isUserMenuOpen:
     2.1. Position dropdown below user avatar
     2.2. Add click-outside listener
     2.3. Focus first menu item

FUNCTION handleUserMenuAction(action: UserMenuAction):
  1. Close user menu
     $sidebarStore.isUserMenuOpen = false

  2. Handle action:
     CASE action === 'view_profile':
       - Navigate to profile page
       - CALL handleNavigation('profile')

     CASE action === 'manage_subscription':
       - Open subscription management
       - IF $userStore.subscriptionTier === 'free':
         - Redirect to pricing page
       - ELSE:
         - Open Stripe customer portal (via API)

     CASE action === 'export_data':
       - Trigger data export
       - CALL initiateDataExport()

     CASE action === 'logout':
       - Show confirmation dialog: "Are you sure you want to log out?"
       - IF confirmed:
         - CALL handleLogout()

FUNCTION handleLogout():
  1. Call logout API
     POST /api/v1/auth/logout

  2. Clear local state
     $userStore = ANONYMOUS_USER
     $sidebarStore.isUserMenuOpen = false

  3. Clear sensitive localStorage items
     localStorage.removeItem('mealswapp_user_preferences')
     // Note: Keep theme preference and search history

  4. Navigate to search (public route)
     CALL handleNavigation('search')

  5. Dispatch logout event
     dispatch('logout')
```

### 2.7 Display Mode Toggle (Manual)

```
FUNCTION toggleSidebarCollapse():
  1. Only applicable when displayMode is 'expanded' or 'collapsed'
     IF $sidebarStore.displayMode === 'hidden':
       RETURN

  2. Toggle between expanded and collapsed
     IF $sidebarStore.displayMode === 'expanded':
       $sidebarStore.displayMode = 'collapsed'
     ELSE:
       $sidebarStore.displayMode = 'expanded'

  3. Persist preference
     localStorage.setItem('mealswapp_sidebar_collapsed',
                          $sidebarStore.displayMode === 'collapsed')

  4. Update main content margin
     - Handled via reactive Tailwind classes based on $sidebarStore.displayMode

  5. Dispatch toggle event
     dispatch('toggle', $sidebarStore.displayMode)
```

### 2.8 Quick Actions Handling

```
FUNCTION loadQuickActions(): QuickAction[]
  1. Read from localStorage
     stored = localStorage.getItem(QUICK_ACTIONS_STORAGE_KEY)

  2. IF stored is null:
     RETURN []

  3. Parse JSON
     TRY:
       actions = JSON.parse(stored)
     CATCH:
       RETURN []

  4. Validate and filter
     validActions = actions.filter(action =>
       isValidQuickAction(action) AND
       (Date.now() - action.timestamp) < 7 * 24 * 60 * 60 * 1000  // 7 days
     )

  5. Limit to max items
     RETURN validActions.slice(0, MAX_QUICK_ACTIONS)

FUNCTION addQuickAction(action: QuickAction):
  1. Load current actions
     actions = loadQuickActions()

  2. Check for duplicate
     existingIndex = actions.findIndex(a =>
       a.type === action.type AND a.data === action.data
     )
     IF existingIndex !== -1:
       - Remove existing (will be re-added at top)
       actions.splice(existingIndex, 1)

  3. Add new action at beginning
     actions.unshift({
       ...action,
       id: generateUUID(),
       timestamp: Date.now()
     })

  4. Trim to max size
     actions = actions.slice(0, MAX_QUICK_ACTIONS)

  5. Persist and update store
     localStorage.setItem(QUICK_ACTIONS_STORAGE_KEY, JSON.stringify(actions))
     $quickActionsStore = actions

FUNCTION handleQuickActionClick(action: QuickAction):
  1. Based on action type:
     CASE action.type === 'recent_search':
       - Navigate to search
       - Dispatch 'searchRestore' event with action.data

     CASE action.type === 'saved_item':
       - Navigate to item detail view
       - Navigate to: /items/${action.data.itemId}

     CASE action.type === 'favorite':
       - Navigate to item detail view
       - Navigate to: /items/${action.data.itemId}
```

### 2.9 Offline Status Handling

```
FUNCTION handleOffline():
  1. Update store
     $isOnlineStore = false

  2. Load pending sync count
     pendingData = localStorage.getItem('mealswapp_pending_sync')
     IF pendingData:
       $pendingSyncCount = JSON.parse(pendingData).length
     ELSE:
       $pendingSyncCount = 0

  3. Update navigation badges
     // History route shows pending count
     historyNav = NAVIGATION_ITEMS.find(n => n.route === 'history')
     IF $pendingSyncCount > 0:
       historyNav.badge = { count: $pendingSyncCount, type: 'warning' }

FUNCTION handleOnline():
  1. Update store
     $isOnlineStore = true

  2. Trigger background sync
     IF $pendingSyncCount > 0:
       dispatch('syncStart')
       // Sync handler will update $pendingSyncCount on completion

  3. Clear badges after successful sync
     // Handled by sync completion event listener

FUNCTION handleSyncComplete(result: { success: boolean, syncedCount: number }):
  1. IF result.success:
     1.1. $pendingSyncCount = 0
     1.2. Clear navigation badge
     1.3. Show brief success toast: "All changes synced"
  2. ELSE:
     2.1. Show error toast: "Sync failed. Will retry."
```

### 2.10 Subscription Status Display

```
FUNCTION getSubscriptionStatusDisplay(): { label: string, variant: string, showUpgrade: boolean }
  tier = $userStore.subscriptionTier

  CASE tier === 'paid':
    RETURN {
      label: 'Pro',
      variant: 'success',
      showUpgrade: false
    }

  CASE tier === 'trial':
    days = $userStore.trialDaysRemaining
    RETURN {
      label: `Trial: ${days} day${days !== 1 ? 's' : ''} left`,
      variant: days <= 2 ? 'warning' : 'info',
      showUpgrade: true
    }

  CASE tier === 'free':
    searches = $userStore.searchesRemaining
    RETURN {
      label: `Free: ${searches}/3 searches`,
      variant: searches === 0 ? 'error' : 'default',
      showUpgrade: true
    }

FUNCTION handleUpgradeClick():
  1. Track analytics event
     dispatch('analyticsEvent', { type: 'upgrade_click', source: 'sidebar' })

  2. Navigate to pricing/upgrade page
     IF $userStore.isAuthenticated:
       - Navigate to: /upgrade
     ELSE:
       - Store redirect intent: sessionStorage.setItem('mealswapp_redirect_after_login', '/upgrade')
       - Dispatch 'authRequired' event
```

### 2.11 Theme State Synchronization

```
NOTE: System theme detection is handled by theme store on first visit only.
SidebarComponent subscribes to theme store.

ON theme Store Change:
  1. Component re-renders with new $themeStore value
  2. UI updates to reflect current theme (toggle icon, selected option)
  3. No manual DOM manipulation needed - theme store handles it
```

---

## 3. State Management & Error Handling

### 3.1 State Transitions Diagram

```
NOTE: State is managed via Svelte writable stores:
- sidebarStore: { displayMode, isMobileMenuOpen, isUserMenuOpen }
- activeRouteStore: current navigation route
- userStore: user session data
- themeStore: 'light' | 'dark'

                                    ┌─────────────────┐
                                    │     INITIAL     │
                                    │    (Mount)      │
                                    └────────┬────────┘
                                             │
                                             ▼
                           ┌─────────────────────────────────┐
                           │            READY                 │
                           │  (Sidebar visible, functional)   │
                           └───────────────┬─────────────────┘
                                           │
              ┌────────────────────────────┼────────────────────────────┐
              │                            │                            │
              ▼                            ▼                            ▼
     ┌────────────────┐          ┌────────────────┐          ┌────────────────┐
     │   EXPANDED     │<────────>│   COLLAPSED    │          │    HIDDEN      │
     │  (Full width)  │  Toggle  │  (Icons only)  │          │ (Mobile menu)  │
     └────────────────┘          └────────────────┘          └───────┬────────┘
                                                                     │
                                                          ┌──────────┴──────────┐
                                                          │                     │
                                                          ▼                     ▼
                                                 ┌────────────────┐    ┌────────────────┐
                                                 │  MENU_CLOSED   │    │  MENU_OPEN     │
                                                 │                │<-->│  (Overlay)     │
                                                 └────────────────┘    └────────────────┘
```

### 3.2 User Menu State Diagram

```
                    ┌─────────────┐
                    │   CLOSED    │
                    │             │
                    └──────┬──────┘
                           │ Click avatar
                           ▼
                    ┌─────────────┐
                    │    OPEN     │
                    │  (Dropdown) │
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
         ▼                 ▼                 ▼
    Click outside    Select action    Press Escape
         │                 │                 │
         └─────────────────┼─────────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │   CLOSED    │
                    │ (+ action)  │
                    └─────────────┘
```

### 3.3 Error States

| Error State | Trigger | User Message | Recovery Action |
|:------------|:--------|:-------------|:----------------|
| **AUTH_FETCH_FAILED** | GET /user/me fails | Silent (show anonymous) | Retry on next interaction |
| **LOGOUT_FAILED** | POST /logout fails | "Couldn't log out. Please try again." | Show retry button |
| **THEME_PERSIST_FAILED** | localStorage quota exceeded | Silent (theme still applied) | Clear old data |
| **NAVIGATION_BLOCKED** | Protected route without auth | "Please log in to access this feature." | Show login prompt |
| **SUBSCRIPTION_CHECK_FAILED** | API error | Silent (allow access) | Log for monitoring |
| **SYNC_FAILED** | Network error during sync | "Sync failed. Will retry." | Auto-retry when online |

### 3.4 Error Handling Implementation

```typescript
import { toast } from './toastStore';

interface SidebarError {
  type: 'AUTH_FETCH_FAILED' | 'LOGOUT_FAILED' | 'THEME_PERSIST_FAILED' |
        'NAVIGATION_BLOCKED' | 'SUBSCRIPTION_CHECK_FAILED' | 'SYNC_FAILED';
  message: string;
  recoverable: boolean;
  silent: boolean;
}

function handleSidebarError(error: unknown, context: string): SidebarError
  1. IF context === 'auth_fetch':
     RETURN {
       type: 'AUTH_FETCH_FAILED',
       message: 'Failed to load user data',
       recoverable: true,
       silent: true
     }

  2. IF context === 'logout':
     RETURN {
       type: 'LOGOUT_FAILED',
       message: "Couldn't log out. Please try again.",
       recoverable: true,
       silent: false
     }

  3. IF context === 'navigation' AND error.code === 'AUTH_REQUIRED':
     RETURN {
       type: 'NAVIGATION_BLOCKED',
       message: 'Please log in to access this feature.',
       recoverable: true,
       silent: false
     }

  4. IF context === 'theme_persist':
     RETURN {
       type: 'THEME_PERSIST_FAILED',
       message: 'Could not save theme preference',
       recoverable: false,
       silent: true
     }

  5. DEFAULT:
     RETURN {
       type: 'SYNC_FAILED',
       message: 'An error occurred. Please try again.',
       recoverable: true,
       silent: false
     }

function displayError(error: SidebarError): void
  IF error.silent:
    RETURN

  IF error.recoverable:
    toast.warning(error.message)
  ELSE:
    toast.error(error.message)
```

### 3.5 Graceful Degradation

| Scenario | Degraded Functionality | Core Functionality Preserved |
|:---------|:-----------------------|:-----------------------------|
| **Auth API down** | User shown as anonymous | Navigation, theme, search |
| **localStorage full** | Theme/preferences not persisted | All features work in-session |
| **Offline mode** | Sync badge shows pending count | Full navigation, cached data |
| **Subscription API down** | All features unlocked (fail-open) | Full access (log for review) |
| **Avatar image fails** | Show initials placeholder | User menu fully functional |

---

## 4. Component Interfaces

### 4.1 SidebarComponent Props & Events

```svelte
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { NavigationRoute } from './types';
  import { themeStore, userStore, sidebarStore, quickActionsStore } from './stores';

  export let initialRoute: NavigationRoute = 'search';

  const dispatch = createEventDispatcher<{
    navigationChange: NavigationRoute;
    displayModeChange: string;
    authRequired: void;
    logout: void;
    syncStart: void;
    searchRestore: { query: string; mode: string };
    analyticsEvent: { type: string; source: string };
  }>();
</script>
```

### 4.2 Internal Component Functions

```typescript
// Svelte stores used for state management
import { themeStore, userStore, sidebarStore, quickActionsStore, isOnlineStore, pendingSyncCountStore, activeRouteStore } from './stores';

// Props
let initialRoute: NavigationRoute = 'search';

// Initialization
function initializeSidebar(): void;
async function loadUserSession(): Promise<UserSummary>;
function loadThemePreference(): ThemeOption;
function loadQuickActions(): QuickAction[];

// Navigation
function handleNavigation(route: NavigationRoute): void;
function getActiveRoute(): NavigationRoute;
function isRouteAccessible(route: NavigationRoute): boolean;
function updateBrowserUrl(route: NavigationRoute): void;

// Display Mode
function handleResize(): void;
function toggleSidebarCollapse(): void;
function getDisplayMode(): SidebarDisplayMode;

// Mobile Menu
function toggleMobileMenu(): void;
function handleMobileOverlayClick(): void;
function isMobileMenuOpen(): boolean;

// Theme (handled by themeStore)
function handleThemeToggle(): void;
function handleThemeSelect(theme: Theme): void;
// Theme state accessed via $themeStore

// User Menu
function toggleUserMenu(): void;
function handleUserMenuAction(action: UserMenuAction): void;
async function handleLogout(): Promise<void>;
function getUserDisplayInfo(): { name: string; avatar: string | null };

// Subscription
function getSubscriptionStatusDisplay(): { label: string; variant: string; showUpgrade: boolean };
function handleUpgradeClick(): void;

// Quick Actions
function loadQuickActions(): QuickAction[];
function addQuickAction(action: QuickAction): void;
function handleQuickActionClick(action: QuickAction): void;
function removeQuickAction(actionId: string): void;

// Offline Handling
function handleOnline(): void;
function handleOffline(): void;
function handleSyncComplete(result: SyncResult): void;
function getPendingSyncCount(): number;

// Error Handling
function handleSidebarError(error: unknown, context: string): SidebarError;
function displayError(error: SidebarError): void;

// Cleanup
function cleanup(): void;  // Remove event listeners on destroy
```

### 4.3 Event Dispatchers

```typescript
// Svelte createEventDispatcher usage for parent components
const dispatch = createEventDispatcher<{
  'navigationChange': NavigationRoute;
  'themeChange': 'light' | 'dark';
  'displayModeChange': SidebarDisplayMode;
  'authRequired': void;
  'logout': void;
  'syncStart': void;
  'searchRestore': { query: string; mode: string };
  'analyticsEvent': { type: string; source: string };
}>();

// Dispatch examples:
// dispatch('navigationChange', route);
// dispatch('themeChange', $themeStore);
// dispatch('authRequired');
```

### 4.4 API Interface Contracts

```typescript
// GET /api/v1/user/me
interface UserMeResponse {
  id: string;
  email: string;
  displayName: string;
  avatarUrl: string | null;
  subscription: {
    tier: SubscriptionTier;
    trialEndsAt: string | null;      // ISO date
    searchesUsedToday: number;       // For free tier
  };
  preferences: {
    theme: ThemeOption;
    unitSystem: 'metric' | 'imperial';
  };
}

// POST /api/v1/auth/logout
interface LogoutResponse {
  success: boolean;
}

// POST /api/v1/user/export
interface ExportDataResponse {
  jobId: string;
  downloadUrl: string;    // Available when job completes
  expiresAt: string;      // ISO date
}
```

### 4.5 localStorage Keys

| Key | Type | Description |
|:----|:-----|:------------|
| `mealswapp_theme` | `'light' \| 'dark'` | User theme preference (managed by ThemeProvider) |
| `mealswapp_quick_actions` | `QuickAction[]` | Recent quick actions |
| `mealswapp_sidebar_collapsed` | `boolean` | Manual collapse preference |
| `mealswapp_pending_sync` | `PendingSyncItem[]` | Offline changes awaiting sync |

---

## 5. UI Component Structure

```
SidebarComponent (Svelte)
├── SidebarContainer (div class="fixed left-0 top-0 h-full ...")
│   │
│   ├── Logo/Brand
│   │   ├── LogoIcon
│   │   └── LogoText (class="hidden when collapsed")
│   │
│   ├── NavigationList
│   │   └── NavigationItem[] (for each route)
│   │       ├── NavIcon
│   │       ├── NavLabel (class="hidden when collapsed")
│   │       ├── NavBadge (conditional)
│   │       └── ActiveIndicator (conditional)
│   │
│   ├── QuickActionsSection (class="hidden when collapsed")
│   │   ├── SectionHeader ("Recent")
│   │   └── QuickActionItem[]
│   │       ├── ActionIcon
│   │       ├── ActionLabel
│   │       └── ActionTimestamp
│   │
│   ├── Spacer (flex-grow)
│   │
│   ├── OfflineIndicator (conditional: when offline)
│   │   ├── OfflineIcon
│   │   ├── OfflineMessage
│   │   └── PendingSyncBadge (conditional)
│   │
│   ├── ThemeToggle
│   │   ├── ThemeIcon (sun/moon based on $themeStore)
│   │   └── ThemeSelector (class="hidden when collapsed")
│   │       └── ThemeOption[] (light/dark)
│   │
│   ├── SubscriptionStatus (conditional: when authenticated)
│   │   ├── StatusBadge
│   │   └── UpgradeButton (conditional)
│   │
│   ├── UserSection
│   │   ├── IF authenticated:
│   │   │   ├── UserAvatar
│   │   │   ├── UserName (class="hidden when collapsed")
│   │   │   └── UserMenuDropdown (conditional: when open)
│   │   │       └── UserMenuOption[]
│   │   │           ├── OptionIcon
│   │   │           └── OptionLabel
│   │   │
│   │   └── IF not authenticated:
│   │       └── LoginButton
│   │           ├── LoginIcon
│   │           └── LoginLabel (class="hidden when collapsed")
│   │
│   └── CollapseToggle (visible when expanded/collapsed mode)
│       └── ChevronIcon
│
├── MobileOverlay (conditional: when mobile menu open)
│   (div class="fixed inset-0 bg-black/50 z-40")
│
└── MobileMenuButton (conditional: when displayMode === 'hidden')
    └── HamburgerIcon / CloseIcon
```

---

## 6. Accessibility Requirements

| Element | ARIA Attributes | Keyboard Support |
|:--------|:----------------|:-----------------|
| Sidebar | `role="navigation"`, `aria-label="Main navigation"` | - |
| Navigation List | `role="menubar"`, `aria-orientation="vertical"` | Arrow Up/Down to navigate |
| Navigation Item | `role="menuitem"`, `aria-current="page"` (when active) | Enter to select, Tab to move |
| User Menu Button | `aria-haspopup="menu"`, `aria-expanded` | Enter/Space to toggle |
| User Menu | `role="menu"` | Arrow keys, Enter to select, Escape to close |
| User Menu Item | `role="menuitem"` | Enter/Space to activate |
| Theme Toggle | `role="switch"`, `aria-checked`, `aria-label="Dark mode"` | Space/Enter to toggle |
| Theme Selector | `role="radiogroup"`, `aria-label="Theme selection"` | Arrow keys within group |
| Collapse Toggle | `aria-expanded`, `aria-controls="sidebar-content"` | Enter/Space to toggle |
| Mobile Menu Button | `aria-expanded`, `aria-controls="mobile-sidebar"` | Enter/Space to toggle |
| Offline Indicator | `role="status"`, `aria-live="polite"` | - |

**Focus Management:**
- On mount: Do not auto-focus sidebar (let main content receive focus)
- Mobile menu open: Trap focus within sidebar, focus first navigation item
- Mobile menu close: Return focus to hamburger button
- User menu open: Focus first menu item
- User menu close: Return focus to avatar/trigger button
- On navigation: Focus moves to main content area

**Screen Reader Announcements:**
- Route change: Announce new page title
- Theme change: "Theme changed to {light/dark}"
- Offline status: "You are now offline" / "Back online"
- Sync status: "Syncing {n} changes" / "All changes synced"
- Login/Logout: "Logged in as {name}" / "Logged out"

**Reduced Motion:**
```css
@media (prefers-reduced-motion: reduce) {
  .sidebar-transition,
  .mobile-menu-transition,
  .dropdown-transition {
    transition: none;
  }
}
```

**Tailwind Classes Used:**
- Sidebar container: `fixed left-0 top-0 h-full bg-white dark:bg-gray-900 transition-all duration-300`
- Expanded: `w-64` (256px)
- Collapsed: `w-16` (64px)
- Hidden: `w-0 -translate-x-full`
- Navigation items: `flex items-center gap-3 px-4 py-3 hover:bg-gray-100 dark:hover:bg-gray-800`
- Mobile overlay: `fixed inset-0 bg-black/50 z-40`
```

---

## 7. Responsive Behavior

### 7.1 Breakpoint Behavior

| Viewport Width | Display Mode | Sidebar Width | Main Content Offset | Interaction |
|:---------------|:-------------|:--------------|:--------------------|:------------|
| >= 1024px | Expanded | 256px | 256px | Always visible, collapsible |
| 768-1023px | Collapsed | 64px | 64px | Icon-only, expand on hover (optional) |
| < 768px | Hidden | 0 | 0 | Hamburger menu, slide-in overlay |

### 7.2 Mobile Menu Behavior

```
Mobile Menu Animation:
  OPEN:
    1. Show overlay (fade in 200ms)
    2. Slide sidebar from left (transform 300ms ease-out)
    3. Lock body scroll
    4. Focus first nav item

  CLOSE:
    1. Slide sidebar to left (transform 200ms ease-in)
    2. Fade overlay (200ms)
    3. Unlock body scroll
    4. Focus hamburger button
```

### 7.3 Collapsed Mode Interactions

```
ON Mouse Enter (when collapsed):
  1. IF user preference allows expand-on-hover:
     1.1. Show labels with slide animation (200ms)
     1.2. Temporarily expand to full width
     1.3. Do NOT affect main content margin (overlay behavior)

ON Mouse Leave:
  1. IF temporarily expanded:
     1.1. Collapse back to icon-only (200ms)
```

---

## 8. Performance Considerations

| Optimization | Implementation | Impact |
|:-------------|:---------------|:-------|
| Lazy load user data | TanStack Query with staleTime | Faster initial render |
| Debounced resize | 100ms debounce on resize handler | Prevent layout thrashing |
| CSS containment | `contain: layout style` on sidebar | Isolate repaints |
| Will-change | `will-change: transform` on mobile menu | Smoother animations |
| Svelte reactive statements | Use $store syntax | Efficient updates |
| Image lazy loading | `loading="lazy"` on avatar | Faster initial paint |
| Prefers-reduced-motion | Disable animations when preferred | Respect user settings |

---

## Changelog

### 2026-01-22 (Rev 1.2)

**Changed:**
- Migrated from React-style patterns to Svelte conventions
- Replaced `useTheme()` hook with Svelte `themeStore` subscription
- Replaced `emit()` with Svelte `createEventDispatcher`
- Updated state management to use Svelte stores (`sidebarStore`, `userStore`, etc.)
- Replaced React component props with Svelte props and event dispatching
- Updated CSS to use Tailwind classes as per tech stack
- Changed data fetching to use TanStack Query with Svelte Query
- Updated initialization to use `onMount` lifecycle function

### 2026-01-22 (Rev 1.1)

**Changed:**
- Theme toggle simplified to binary light/dark (removed 'system' option)
- Theme management delegated to ThemeProvider component
- Updated theme-related types, functions, and UI structure
- System theme detection only used on first visit for initial default

### 2026-01-22 (Rev 1.0)

**Added:**
- Initial detailed design document for SidebarComponent
- Complete type definitions for navigation, user, and state objects
- Step-by-step algorithms for all interactions
- Responsive behavior specifications for all breakpoints
- Error handling and graceful degradation specifications
- Full accessibility requirements (WCAG 2.1 AA)
- Component interface contracts and event emitters
- UI component structure hierarchy
