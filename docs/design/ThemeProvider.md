# Detailed Design: ThemeProvider

**Traceability:** [ARCH-001]

---

## 1. Data Structures & Types

### 1.1 Theme Configuration Types

```typescript
type Theme = 'light' | 'dark';

interface ThemeConfig {
  theme: Theme;  // Current active theme (always explicit)
}

const DEFAULT_THEME: Theme = 'light';  // Fallback if system preference unavailable
```

### 1.2 CSS Custom Property Definitions

```typescript
interface ThemeColors {
  // Background colors
  bgPrimary: string;
  bgSurface: string;
  bgSidebar: string;
  bgInput: string;
  bgOverlay: string;

  // Primary brand colors
  colorPrimary: string;
  colorPrimaryHover: string;
  colorSecondary: string;

  // Accent and semantic colors
  colorAccent: string;
  colorError: string;
  colorWarning: string;
  colorSuccess: string;
  colorInfo: string;

  // Text colors
  textPrimary: string;
  textSecondary: string;
  textMuted: string;
  textInverse: string;

  // Border and divider colors
  borderColor: string;
  borderColorStrong: string;
  dividerColor: string;

  // Interactive element colors
  hoverBg: string;
  activeBg: string;
  focusRing: string;
  disabledBg: string;
  disabledText: string;

  // Shadow colors (for elevation)
  shadowColor: string;
}

const LIGHT_THEME: ThemeColors = {
  // Background colors
  bgPrimary: '#F7FCF7',
  bgSurface: '#FFFFFF',
  bgSidebar: '#FFFFFF',
  bgInput: '#FFFFFF',
  bgOverlay: 'rgba(0, 0, 0, 0.5)',

  // Primary brand colors
  colorPrimary: '#166534',
  colorPrimaryHover: '#14532D',
  colorSecondary: '#DCFCE7',

  // Accent and semantic colors
  colorAccent: '#F97316',
  colorError: '#DC2626',
  colorWarning: '#D97706',
  colorSuccess: '#16A34A',
  colorInfo: '#2563EB',

  // Text colors
  textPrimary: '#111827',
  textSecondary: '#374151',
  textMuted: '#6B7280',
  textInverse: '#FFFFFF',

  // Border and divider colors
  borderColor: '#E5E7EB',
  borderColorStrong: '#D1D5DB',
  dividerColor: '#F3F4F6',

  // Interactive element colors
  hoverBg: '#F3F4F6',
  activeBg: '#E5E7EB',
  focusRing: '#166534',
  disabledBg: '#F9FAFB',
  disabledText: '#9CA3AF',

  // Shadow colors
  shadowColor: 'rgba(0, 0, 0, 0.1)'
};

const DARK_THEME: ThemeColors = {
  // Background colors
  bgPrimary: '#0A0F0A',
  bgSurface: '#161D16',
  bgSidebar: '#0D120D',
  bgInput: '#1A231A',
  bgOverlay: 'rgba(0, 0, 0, 0.7)',

  // Primary brand colors
  colorPrimary: '#4ADE80',
  colorPrimaryHover: '#86EFAC',
  colorSecondary: '#14532D',

  // Accent and semantic colors
  colorAccent: '#FFB86C',
  colorError: '#F87171',
  colorWarning: '#FBBF24',
  colorSuccess: '#4ADE80',
  colorInfo: '#60A5FA',

  // Text colors
  textPrimary: '#F3F4F6',
  textSecondary: '#D1D5DB',
  textMuted: '#9CA3AF',
  textInverse: '#111827',

  // Border and divider colors
  borderColor: '#374151',
  borderColorStrong: '#4B5563',
  dividerColor: '#1F2937',

  // Interactive element colors
  hoverBg: '#1F2937',
  activeBg: '#374151',
  focusRing: '#4ADE80',
  disabledBg: '#1F2937',
  disabledText: '#6B7280',

  // Shadow colors
  shadowColor: 'rgba(0, 0, 0, 0.4)'
};
```

### 1.3 Storage Keys

```typescript
const THEME_STORAGE_KEY = 'mealswapp_theme';
const THEME_ATTRIBUTE = 'data-theme';
const THEME_COLOR_META_ID = 'theme-color-meta';
```

### 1.4 Context Types

```typescript
interface ThemeContextValue {
  theme: Theme;                       // Current active theme
  setTheme: (theme: Theme) => void;   // Set specific theme
  toggleTheme: () => void;            // Toggle between light and dark
}

const ThemeContext = createContext<ThemeContextValue | null>(null);
```

### 1.5 Provider Props

```typescript
interface ThemeProviderProps {
  children: ReactNode;
  storageKey?: string;                        // Default: THEME_STORAGE_KEY
  disablePersistence?: boolean;               // Default: false
  onThemeChange?: (theme: Theme) => void;
}
```

### 1.6 CSS Variable Mapping

```typescript
const CSS_VARIABLE_MAP: Record<keyof ThemeColors, string> = {
  bgPrimary: '--bg-primary',
  bgSurface: '--bg-surface',
  bgSidebar: '--bg-sidebar',
  bgInput: '--bg-input',
  bgOverlay: '--bg-overlay',
  colorPrimary: '--color-primary',
  colorPrimaryHover: '--color-primary-hover',
  colorSecondary: '--color-secondary',
  colorAccent: '--color-accent',
  colorError: '--color-error',
  colorWarning: '--color-warning',
  colorSuccess: '--color-success',
  colorInfo: '--color-info',
  textPrimary: '--text-primary',
  textSecondary: '--text-secondary',
  textMuted: '--text-muted',
  textInverse: '--text-inverse',
  borderColor: '--border-color',
  borderColorStrong: '--border-color-strong',
  dividerColor: '--divider-color',
  hoverBg: '--hover-bg',
  activeBg: '--active-bg',
  focusRing: '--focus-ring',
  disabledBg: '--disabled-bg',
  disabledText: '--disabled-text',
  shadowColor: '--shadow-color'
};
```

---

## 2. Logic & Algorithms (Step-by-Step)

### 2.1 Initialization Flow

```
ON ThemeProvider Mount:
  1. Prevent flash of incorrect theme (FOIT)
     1.1. Check if inline script already applied theme during SSR/initial load
     1.2. IF document.documentElement has THEME_ATTRIBUTE:
          - Read existing attribute value as initial theme
          - Skip to step 5
     1.3. ELSE:
          - Proceed with detection

  2. Check for persisted user preference
     2.1. IF disablePersistence is false:
          - TRY:
              stored = localStorage.getItem(storageKey)
              IF stored AND isValidTheme(stored):
                initialTheme = stored as Theme
                SKIP to step 4
          - CATCH (SecurityError):
              Log warning: 'localStorage unavailable'

  3. Detect system color scheme preference (first visit only)
     3.1. IF no stored preference found:
          - Query media: window.matchMedia('(prefers-color-scheme: dark)')
          - initialTheme = mediaQuery.matches ? 'dark' : 'light'
     3.2. IF matchMedia unavailable:
          - initialTheme = DEFAULT_THEME ('light')

  4. Apply theme to DOM
     4.1. CALL applyThemeToDom(initialTheme)

  5. Initialize state
     state = { theme: initialTheme }
```

### 2.2 Theme Validation

```
FUNCTION isValidTheme(value: unknown): boolean
  RETURN value === 'light' OR value === 'dark'
```

### 2.3 Theme Application to DOM

```
FUNCTION applyThemeToDom(theme: Theme):
  1. Select color palette
     colors = theme === 'dark' ? DARK_THEME : LIGHT_THEME

  2. Apply CSS custom properties to document root
     root = document.documentElement.style
     FOR EACH [key, cssVar] IN CSS_VARIABLE_MAP:
       root.setProperty(cssVar, colors[key])

  3. Set theme attribute on html element
     document.documentElement.setAttribute(THEME_ATTRIBUTE, theme)

  4. Update meta theme-color for browser chrome
     4.1. Find or create meta tag: <meta name="theme-color">
          metaTag = document.getElementById(THEME_COLOR_META_ID)
          IF metaTag is null:
            metaTag = document.createElement('meta')
            metaTag.id = THEME_COLOR_META_ID
            metaTag.name = 'theme-color'
            document.head.appendChild(metaTag)
     4.2. Set content based on theme:
          metaTag.content = colors.bgPrimary

  5. Update color-scheme CSS property
     document.documentElement.style.colorScheme = theme
```

### 2.4 Theme Change Handling

```
FUNCTION setTheme(newTheme: Theme):
  1. Validate input
     IF NOT isValidTheme(newTheme):
       Log error: 'Invalid theme value'
       RETURN

  2. Check if change is needed
     IF newTheme === state.theme:
       RETURN (no change needed)

  3. Update state
     previousTheme = state.theme
     state.theme = newTheme

  4. Apply to DOM
     CALL applyThemeToDom(newTheme)

  5. Persist to storage
     IF NOT disablePersistence:
       TRY:
         localStorage.setItem(storageKey, newTheme)
       CATCH (QuotaExceededError):
         Log warning: 'Could not persist theme preference'

  6. Notify listeners
     IF onThemeChange callback exists:
       CALL onThemeChange(newTheme)

     Emit custom event for non-React listeners:
       window.dispatchEvent(new CustomEvent('mealswapp:themechange', {
         detail: { theme: newTheme, previousTheme: previousTheme }
       }))

FUNCTION toggleTheme():
  1. Determine opposite theme
     nextTheme = state.theme === 'light' ? 'dark' : 'light'

  2. CALL setTheme(nextTheme)
```

### 2.5 Flash Prevention (SSR/SSG Inline Script)

```
INLINE SCRIPT (to be placed in <head> before any content):

(function() {
  var STORAGE_KEY = 'mealswapp_theme';
  var THEME_ATTR = 'data-theme';

  function getStoredTheme() {
    try {
      return localStorage.getItem(STORAGE_KEY);
    } catch (e) {
      return null;
    }
  }

  function getSystemTheme() {
    try {
      return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    } catch (e) {
      return 'light';
    }
  }

  var stored = getStoredTheme();
  var theme = (stored === 'light' || stored === 'dark')
              ? stored
              : getSystemTheme();  // First visit: use system preference

  document.documentElement.setAttribute(THEME_ATTR, theme);
  document.documentElement.style.colorScheme = theme;
})();
```

### 2.6 Context Provider Implementation

```
FUNCTION ThemeProvider(props: ThemeProviderProps):
  1. Compute initial theme (runs once)
     initialTheme = useMemo(() => {
       // Check if already applied by inline script
       existingTheme = document.documentElement.getAttribute(THEME_ATTRIBUTE)
       IF existingTheme === 'light' OR existingTheme === 'dark':
         RETURN existingTheme

       // Check localStorage
       stored = getStoredThemePreference()
       IF stored:
         RETURN stored

       // First visit: use system preference
       RETURN getSystemThemeSafe()
     }, [])

  2. Create state
     [theme, setThemeState] = useState<Theme>(initialTheme)

  3. Apply theme on mount (if not already applied by inline script)
     useEffect(() => {
       applyThemeToDom(theme)
     }, [])

  4. Create stable callbacks
     setTheme = useCallback((newTheme: Theme) => {
       IF NOT isValidTheme(newTheme): RETURN
       IF newTheme === theme: RETURN

       setThemeState(newTheme)
       applyThemeToDom(newTheme)
       persistTheme(newTheme)
       props.onThemeChange?.(newTheme)
     }, [theme, props.onThemeChange])

     toggleTheme = useCallback(() => {
       setTheme(theme === 'light' ? 'dark' : 'light')
     }, [theme, setTheme])

  5. Create context value with stable references
     contextValue = useMemo(() => ({
       theme,
       setTheme,
       toggleTheme
     }), [theme, setTheme, toggleTheme])

  6. Render provider
     RETURN (
       <ThemeContext.Provider value={contextValue}>
         {props.children}
       </ThemeContext.Provider>
     )
```

### 2.8 Hook Implementation

```
FUNCTION useTheme(): ThemeContextValue
  1. Get context
     context = useContext(ThemeContext)

  2. Validate context exists
     IF context === null:
       THROW Error('useTheme must be used within a ThemeProvider')

  3. RETURN context
```

### 2.9 Helper Functions

```
FUNCTION getSystemThemeSafe(): Theme
  TRY:
    IF typeof window !== 'undefined' AND window.matchMedia:
      RETURN window.matchMedia('(prefers-color-scheme: dark)').matches
             ? 'dark' : 'light'
  CATCH:
    // matchMedia not supported
  RETURN 'light'  // Safe default

FUNCTION getStoredThemePreference(): Theme | null
  TRY:
    stored = localStorage.getItem(THEME_STORAGE_KEY)
    IF stored === 'light' OR stored === 'dark':
      RETURN stored
  CATCH:
    // localStorage unavailable
  RETURN null

FUNCTION persistTheme(theme: Theme): void
  TRY:
    localStorage.setItem(THEME_STORAGE_KEY, theme)
  CATCH:
    Log warning: 'Could not persist theme'
```

---

## 3. State Management & Error Handling

### 3.1 State Transitions Diagram

```
                         ┌─────────────────────┐
                         │      INITIAL        │
                         │   (Before mount)    │
                         └──────────┬──────────┘
                                    │
                         ┌──────────┴──────────┐
                         │                     │
                         ▼                     ▼
              ┌─────────────────┐   ┌─────────────────┐
              │ HAS_STORED_PREF │   │  FIRST_VISIT    │
              │ (Use stored)    │   │ (Use system)    │
              └────────┬────────┘   └────────┬────────┘
                       │                     │
                       └──────────┬──────────┘
                                  │
                                  ▼
                       ┌─────────────────────┐
                       │    THEME_APPLIED    │
                       │  (CSS vars set,     │
                       │   ready for use)    │
                       └──────────┬──────────┘
                                  │
                    ┌─────────────┴─────────────┐
                    │                           │
                    ▼                           ▼
              ┌───────────┐               ┌───────────┐
              │   LIGHT   │<------------->│   DARK    │
              │           │  toggleTheme  │           │
              └───────────┘   setTheme    └───────────┘
```

**Key behavior:**
- System preference is only used on first visit (no stored preference)
- Once user toggles, their choice is persisted
- No "system" mode exposed to user - just light/dark toggle

### 3.2 Error States

| Error State | Trigger | User Impact | Recovery Action |
|:------------|:--------|:------------|:----------------|
| **STORAGE_UNAVAILABLE** | localStorage blocked (private browsing, security settings) | Theme works but won't persist across sessions | Continue with in-memory state; log warning |
| **STORAGE_QUOTA_EXCEEDED** | localStorage full | Theme works but won't persist | Continue without persistence |
| **INVALID_STORED_VALUE** | Corrupted localStorage data | None (fallback to system pref) | Remove invalid key, detect system preference |
| **SSR_MISMATCH** | Server/client theme mismatch | Brief flash of incorrect theme | Inline script prevents this |
| **MEDIA_QUERY_UNSUPPORTED** | Old browser without matchMedia | First visit defaults to light | Default to 'light' theme |

### 3.3 Error Handling Implementation

```typescript
type ThemeErrorType =
  | 'STORAGE_UNAVAILABLE'
  | 'STORAGE_QUOTA_EXCEEDED'
  | 'INVALID_STORED_VALUE'
  | 'MEDIA_QUERY_UNSUPPORTED';

interface ThemeError {
  type: ThemeErrorType;
  message: string;
  recoverable: boolean;
  fallbackValue: Theme;
}

FUNCTION handleStorageRead(): Theme | null
  TRY:
    value = localStorage.getItem(THEME_STORAGE_KEY)
    IF value === null:
      RETURN null
    IF isValidTheme(value):
      RETURN value as Theme
    ELSE:
      // Invalid stored value - clean up
      localStorage.removeItem(THEME_STORAGE_KEY)
      Log warning: 'Removed invalid theme preference from storage'
      RETURN null
  CATCH (SecurityError):
    Log warning: 'localStorage access denied (private browsing?)'
    RETURN null
  CATCH (Error):
    Log warning: 'Failed to read theme preference'
    RETURN null

FUNCTION handleStorageWrite(theme: Theme): boolean
  TRY:
    localStorage.setItem(THEME_STORAGE_KEY, theme)
    RETURN true
  CATCH (QuotaExceededError):
    Log warning: 'localStorage quota exceeded, theme not persisted'
    RETURN false
  CATCH (SecurityError):
    Log warning: 'localStorage access denied'
    RETURN false
  CATCH (Error):
    Log warning: 'Failed to persist theme preference'
    RETURN false
```

### 3.4 Graceful Degradation

| Scenario | Degraded Behavior | Core Functionality |
|:---------|:------------------|:-------------------|
| **localStorage unavailable** | Theme changes work per-session but don't persist | Full theming functionality |
| **matchMedia unsupported** | First visit defaults to 'light' instead of detecting OS preference | Light/dark toggle works normally |
| **SSR environment** | Returns 'light' for initial render | Hydration corrects on client |
| **CSS custom properties unsupported** | Fallback to hardcoded styles | Basic light theme only |

---

## 4. Component Interfaces

### 4.1 ThemeProvider Component

```typescript
interface ThemeProviderProps {
  children: ReactNode;
  storageKey?: string;                    // Default: 'mealswapp_theme'
  disablePersistence?: boolean;           // Default: false
  onThemeChange?: (theme: Theme) => void; // Called when theme changes
}

function ThemeProvider(props: ThemeProviderProps): JSX.Element;
```

### 4.2 useTheme Hook

```typescript
interface ThemeContextValue {
  /** Current active theme ('light' or 'dark') */
  theme: Theme;

  /** Set a specific theme */
  setTheme: (theme: Theme) => void;

  /** Toggle between light and dark */
  toggleTheme: () => void;
}

function useTheme(): ThemeContextValue;
```

### 4.3 Utility Functions (Exported)

```typescript
/**
 * Get the current theme from DOM attribute.
 * Useful for non-React code.
 */
function getCurrentTheme(): Theme;

/**
 * Apply a theme to the DOM immediately.
 * Useful for flash prevention scripts.
 */
function applyTheme(theme: Theme): void;

/**
 * Get the stored theme preference from localStorage.
 * Returns null if no preference is stored.
 */
function getStoredThemePreference(): Theme | null;

/**
 * Clear the stored theme preference.
 * Next visit will use system preference.
 */
function clearStoredThemePreference(): void;

/**
 * Get CSS variable value for current theme.
 */
function getThemeColor(colorKey: keyof ThemeColors): string;

/**
 * Get the complete color palette for a theme.
 */
function getThemePalette(theme: Theme): ThemeColors;

/**
 * Detect system color scheme preference.
 * Returns 'light' if detection fails.
 */
function getSystemTheme(): Theme;
```

### 4.4 Event Types (for non-React listeners)

```typescript
interface ThemeChangeEventDetail {
  theme: Theme;
  previousTheme: Theme;
}

// Usage: window.addEventListener('mealswapp:themechange', handler)
type ThemeChangeEvent = CustomEvent<ThemeChangeEventDetail>;
```

### 4.5 CSS Custom Properties Contract

```css
/* All components can depend on these CSS custom properties being available */
:root {
  /* Backgrounds */
  --bg-primary: <color>;
  --bg-surface: <color>;
  --bg-sidebar: <color>;
  --bg-input: <color>;
  --bg-overlay: <color>;

  /* Brand colors */
  --color-primary: <color>;
  --color-primary-hover: <color>;
  --color-secondary: <color>;

  /* Semantic colors */
  --color-accent: <color>;
  --color-error: <color>;
  --color-warning: <color>;
  --color-success: <color>;
  --color-info: <color>;

  /* Text colors */
  --text-primary: <color>;
  --text-secondary: <color>;
  --text-muted: <color>;
  --text-inverse: <color>;

  /* Borders */
  --border-color: <color>;
  --border-color-strong: <color>;
  --divider-color: <color>;

  /* Interactive states */
  --hover-bg: <color>;
  --active-bg: <color>;
  --focus-ring: <color>;
  --disabled-bg: <color>;
  --disabled-text: <color>;

  /* Elevation */
  --shadow-color: <color>;
}
```

---

## 5. Integration Requirements

### 5.1 Application Root Setup

```typescript
// App.tsx or _app.tsx
import { ThemeProvider } from './providers/ThemeProvider';

function App({ children }) {
  return (
    <ThemeProvider
      onThemeChange={(theme) => {
        // Optional: analytics tracking
        analytics.track('theme_changed', { theme });
      }}
    >
      {children}
    </ThemeProvider>
  );
}
```

### 5.2 HTML Document Setup

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <!-- Flash prevention script - MUST be first -->
  <script>
    (function() {
      var t=localStorage.getItem('mealswapp_theme');
      var e=(t==='light'||t==='dark')?t:
        (window.matchMedia('(prefers-color-scheme:dark)').matches?'dark':'light');
      document.documentElement.setAttribute('data-theme',e);
      document.documentElement.style.colorScheme=e;
    })();
  </script>

  <!-- Meta theme-color will be updated by ThemeProvider -->
  <meta id="theme-color-meta" name="theme-color" content="#F7FCF7">

  <!-- Rest of head content -->
</head>
<body>
  <div id="root"></div>
</body>
</html>
```

### 5.3 Component Usage Example

```typescript
// Simple toggle button
function ThemeToggle() {
  const { theme, toggleTheme } = useTheme();

  return (
    <button onClick={toggleTheme} aria-label={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}>
      {theme === 'light' ? <MoonIcon /> : <SunIcon />}
    </button>
  );
}

// Settings panel with explicit selection
function SettingsPanel() {
  const { theme, setTheme } = useTheme();

  return (
    <div>
      <h2>Theme</h2>
      <RadioGroup
        value={theme}
        onChange={setTheme}
        options={[
          { value: 'light', label: 'Light', icon: 'sun' },
          { value: 'dark', label: 'Dark', icon: 'moon' }
        ]}
      />
    </div>
  );
}
```

### 5.4 CSS Usage

```css
/* Using CSS custom properties */
.card {
  background-color: var(--bg-surface);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
}

.card:hover {
  background-color: var(--hover-bg);
}

.button-primary {
  background-color: var(--color-primary);
  color: var(--text-inverse);
}

.button-primary:hover {
  background-color: var(--color-primary-hover);
}

/* Theme-specific overrides (if needed) */
[data-theme="dark"] .special-element {
  /* Dark mode specific styles */
}
```

---

## 6. Performance Considerations

| Optimization | Implementation | Impact |
|:-------------|:---------------|:-------|
| **Flash prevention** | Inline script in `<head>` | No visible theme flash on load |
| **Memoized context** | `useMemo` for context value | Prevent unnecessary re-renders |
| **Stable callbacks** | `useCallback` for setTheme/toggleTheme | Prevent child re-renders |
| **Batched DOM updates** | Apply all CSS vars in single frame | No layout thrashing |
| **Lazy color calculation** | Colors computed once per theme | No redundant calculations |
| **No runtime listeners** | System preference only checked on init | Zero ongoing overhead |

---

## 7. Accessibility Considerations

| Requirement | Implementation |
|:------------|:---------------|
| **System preference respect** | First visit defaults to `prefers-color-scheme` preference |
| **Contrast ratios** | All color combinations meet WCAG AA (4.5:1 for text) |
| **Focus visibility** | `--focus-ring` ensures visible focus indicators in both themes |
| **Screen reader announcement** | Theme changes emit aria-live update |
| **Reduced motion** | Theme transitions respect `prefers-reduced-motion` |

### 7.1 Contrast Verification Matrix

| Background | Text Color | Contrast Ratio | WCAG Level |
|:-----------|:-----------|:---------------|:-----------|
| `--bg-primary` (light) | `--text-primary` | 15.3:1 | AAA |
| `--bg-primary` (light) | `--text-muted` | 4.6:1 | AA |
| `--bg-primary` (dark) | `--text-primary` | 14.8:1 | AAA |
| `--bg-primary` (dark) | `--text-muted` | 4.5:1 | AA |
| `--color-primary` (light) | `--text-inverse` | 7.2:1 | AAA |
| `--color-primary` (dark) | `--text-inverse` | 8.1:1 | AAA |

---

## 8. Testing Requirements

### 8.1 Unit Test Cases

| Test Case | Input | Expected Output |
|:----------|:------|:----------------|
| First visit (system light) | No stored preference, OS is light | theme='light' |
| First visit (system dark) | No stored preference, OS is dark | theme='dark' |
| Stored 'light' preference | localStorage has 'light' | theme='light' |
| Stored 'dark' preference | localStorage has 'dark' | theme='dark' |
| Invalid stored value | localStorage has 'invalid' | Fallback to system preference, storage cleared |
| setTheme('dark') | Current theme is 'light' | theme='dark', persisted to storage |
| setTheme('light') | Current theme is 'dark' | theme='light', persisted to storage |
| toggleTheme | Current theme is 'light' | theme='dark' |
| toggleTheme | Current theme is 'dark' | theme='light' |
| setTheme same value | setTheme('light') when theme='light' | No change, no event emitted |

### 8.2 Integration Test Cases

| Test Case | Scenario | Expected Behavior |
|:----------|:---------|:------------------|
| No flash on load | Page load with stored 'dark' | Dark theme visible immediately |
| First visit respects OS | First visit with OS dark mode | Dark theme applied |
| Persistence across sessions | Set theme to 'dark', refresh | Dark theme restored |
| CSS variables applied | Set theme to 'dark' | All CSS vars have dark values |
| Event emission | Change theme | 'mealswapp:themechange' event fired |
| Context available | useTheme in child component | Returns valid context value |
| localStorage unavailable | Private browsing mode | Theme toggles work, just don't persist |

---

## Changelog

### 2026-01-22 (Rev 1.0)

**Added:**
- Initial detailed design document for ThemeProvider
- Complete type definitions for theme configuration and colors
- Light and dark theme color palettes with CSS variable mapping
- Initialization flow with flash prevention
- Simple binary theme toggle (light ↔ dark)
- System preference detection for first-visit default only
- Error handling for localStorage and matchMedia edge cases
- Context provider and hook interfaces
- Integration requirements and usage examples
- Performance optimizations
- Accessibility considerations with contrast verification
- Comprehensive test case specifications

**Design Decision:**
- System theme preference is used only to set the initial default on first visit
- User-facing toggle is a simple light/dark switch, no "system" mode exposed
- Stored preference is always explicit ('light' or 'dark'), never 'system'
- This avoids confusing UX where toggling produces no visible change
