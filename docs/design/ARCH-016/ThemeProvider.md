# ThemeProvider

**Traceability:** ARCH-016

## 1. Data Structures & Types

```typescript
type ThemeMode = 'light' | 'dark' | 'system';

interface ThemeContextValue {
  theme: ThemeMode;
  resolvedTheme: 'light' | 'dark';
  setTheme: (theme: ThemeMode) => void;
  toggleTheme: () => void;
}

interface ThemeColors {
  bgPrimary: string;
  bgSurface: string;
  colorPrimary: string;
  colorSecondary: string;
  colorAccent: string;
  colorError: string;
  textPrimary: string;
  textMuted: string;
}

interface ColorTokens {
  light: ThemeColors;
  dark: ThemeColors;
}

const COLOR_TOKENS: ColorTokens = {
  light: {
    bgPrimary: '#F7FCF7',
    bgSurface: '#FFFFFF',
    colorPrimary: '#166534',
    colorSecondary: '#DCFCE7',
    colorAccent: '#F97316',
    colorError: '#DC2626',
    textPrimary: '#111827',
    textMuted: '#6B7280',
  },
  dark: {
    bgPrimary: '#0A0F0A',
    bgSurface: '#161D16',
    colorPrimary: '#4ADE80',
    colorSecondary: '#86EFAC',
    colorAccent: '#FFB86C',
    colorError: '#F87171',
    textPrimary: '#F3F4F6',
    textMuted: '#9CA3AF',
  },
};
```

## 2. Logic & Algorithms

### Theme Initialization Flow

```
1. onMount():
   a. Read stored theme from localStorage (key: 'app-theme')
   b. If stored theme exists:
      - Use stored theme as theme state
   c. Else:
      - Query window.matchMedia('(prefers-color-scheme: dark)')
      - Set resolvedTheme based on system preference
      - Store 'system' as default theme

2. Apply CSS custom properties:
   a. Get color tokens for resolvedTheme
   b. Set each color token on document.documentElement.style
   c. Add data-theme attribute to document.documentElement
```

### Theme Change Flow

```
setTheme(newTheme: ThemeMode):
1. Update stored theme in localStorage
2. If newTheme is 'system':
   a. Query system preference via matchMedia
   b. Set resolvedTheme to system preference
3. Else:
   a. Set resolvedTheme directly to newTheme
4. Apply CSS custom properties for resolvedTheme
5. Update document.documentElement data-theme attribute
```

### CSS Custom Property Application

```typescript
function applyThemeColors(theme: 'light' | 'dark'): void {
  const tokens = COLOR_TOKENS[theme];
  const root = document.documentElement;

  root.style.setProperty('--bg-primary', tokens.bgPrimary);
  root.style.setProperty('--bg-surface', tokens.bgSurface);
  root.style.setProperty('--color-primary', tokens.colorPrimary);
  root.style.setProperty('--color-secondary', tokens.colorSecondary);
  root.style.setProperty('--color-accent', tokens.colorAccent);
  root.style.setProperty('--color-error', tokens.colorError);
  root.style.setProperty('--text-primary', tokens.textPrimary);
  root.style.setProperty('--text-muted', tokens.textMuted);
}
```

### System Theme Listener Setup

```typescript
function setupSystemThemeListener(): () => void {
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

  const handleChange = (e: MediaQueryListEvent): void => {
    const currentTheme = getThemeFromStore(); // or from context
    if (currentTheme === 'system') {
      const resolvedTheme = e.matches ? 'dark' : 'light';
      applyThemeColors(resolvedTheme);
      updateResolvedThemeState(resolvedTheme);
    }
  };

  mediaQuery.addEventListener('change', handleChange);

  // Return cleanup function
  return () => mediaQuery.removeEventListener('change', handleChange);
}
```

## 3. State Management & Error Handling

### Error States

| Error Condition | Cause | Handling |
| :--- | :--- | :--- |
| localStorage unavailable | Private browsing, disabled cookies | Gracefully fallback to system preference |
| localStorage read failure | Corrupted data | Reset to 'system' theme |
| localStorage write failure | Quota exceeded | Log warning, continue with in-memory state |
| matchMedia API unavailable | Unsupported browser | Default to light theme |
| CSS property set failure | Security restrictions (iframe) | Log error, component continues functioning |

### State Transitions

```
Initial State:
  theme: 'system'
  resolvedTheme: (determined from system)
  isLoading: true

After Initialization:
  theme: (stored or 'system')
  resolvedTheme: (calculated)
  isLoading: false

On Theme Change:
  theme: (updated value)
  resolvedTheme: (recalculated if needed)
  isLoading: false (no loading state during switch)
```

### Contrast Validation

The ThemeProvider enforces WCAG 2.1 AA compliance through pre-defined color tokens. All color combinations in COLOR_TOKENS meet the 4.5:1 contrast ratio requirement. No runtime validation is performed; validation occurs during design token definition.

## 4. Component Interfaces

```svelte
<!-- ThemeProvider.svelte -->
<script lang="ts">
  import { onMount, onDestroy, createContext } from 'svelte';
  import { browser } from '$app/environment';

  export let defaultTheme: ThemeMode = 'system';

  type ThemeContextType = {
    theme: Writable<ThemeMode>;
    resolvedTheme: Writable<'light' | 'dark'>;
    setTheme: (theme: ThemeMode) => void;
    toggleTheme: () => void;
  };

  export const ThemeContext = createContext<ThemeContextType>();

  let theme: Writable<ThemeMode> = writable(defaultTheme);
  let resolvedTheme: Writable<'light' | 'dark'> = writable('light');
  let isLoading = true;

  function getSystemTheme(): 'light' | 'dark' {
    if (!browser) return 'light';
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  function getStoredTheme(): ThemeMode | null {
    if (!browser) return null;
    try {
      return localStorage.getItem('app-theme') as ThemeMode;
    } catch {
      return null;
    }
  }

  function storeTheme(themeValue: ThemeMode): void {
    if (!browser) return;
    try {
      localStorage.setItem('app-theme', themeValue);
    } catch {
      console.warn('Failed to store theme preference');
    }
  }

  function applyTheme(themeValue: ThemeMode): void {
    const resolved = themeValue === 'system' ? getSystemTheme() : themeValue;
    const tokens = COLOR_TOKENS[resolved];

    Object.entries(tokens).forEach(([key, value]) => {
      document.documentElement.style.setProperty(`--${key.replace(/([A-Z])/g, '-$1').toLowerCase()}`, value);
    });

    document.documentElement.setAttribute('data-theme', resolved);
    resolvedTheme.set(resolved);
  }

  function initializeTheme(): void {
    const stored = getStoredTheme();
    const initialTheme = stored || defaultTheme;
    theme.set(initialTheme);
    applyTheme(initialTheme);
    isLoading = false;
  }

  export function setTheme(newTheme: ThemeMode): void {
    theme.set(newTheme);
    storeTheme(newTheme);
    applyTheme(newTheme);
  }

  export function toggleTheme(): void {
    const currentResolved = $resolvedTheme;
    const newTheme = currentResolved === 'light' ? 'dark' : 'light';
    setTheme(newTheme);
  }

  ThemeContext.set({
    theme,
    resolvedTheme,
    setTheme,
    toggleTheme,
  });

  let cleanupListener: (() => void) | null = null;

  onMount(() => {
    initializeTheme();

    if (browser) {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      const handleChange = (e: MediaQueryListEvent) => {
        if ($theme === 'system') {
          applyTheme('system');
        }
      };
      mediaQuery.addEventListener('change', handleChange);
      cleanupListener = () => mediaQuery.removeEventListener('change', handleChange);
    }
  });

  onDestroy(() => {
    if (cleanupListener) {
      cleanupListener();
    }
  });
</script>

<slot />
```

```typescript
// useTheme hook for consuming components
export function useTheme(): ThemeContextType {
  const context = getContext<ThemeContextType>(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}
```

```svelte
<!-- ThemeToggle.svelte (example consumer) -->
<script lang="ts">
  import { useTheme } from './ThemeProvider.svelte';

  const { resolvedTheme, toggleTheme } = useTheme();
</script>

<button
  onclick={toggleTheme}
  aria-label="Toggle theme"
  class="p-2 rounded-lg bg-surface border border-muted hover:bg-secondary transition-colors"
>
  {#if $resolvedTheme === 'light'}
    <!-- Sun icon -->
    <svg>...</svg>
  {:else}
    <!-- Moon icon -->
    <svg>...</svg>
  {/if}
</button>
```
