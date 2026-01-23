# ColorPalette Module

**Traceability:** ARCH-016

## 1. Data Structures & Types

```typescript
type ThemeMode = 'light' | 'dark' | 'system';

interface ColorToken {
  token: string;
  lightValue: string;
  darkValue: string;
  usage: string;
  contrastRatio: number;
  wcagCompliant: boolean;
}

interface ColorPalette {
  bgPrimary: ColorToken;
  bgSurface: ColorToken;
  colorPrimary: ColorToken;
  colorSecondary: ColorToken;
  colorAccent: ColorToken;
  colorError: ColorToken;
  textPrimary: ColorToken;
  textMuted: ColorToken;
}

interface ThemeContextValue {
  mode: ThemeMode;
  resolvedTheme: 'light' | 'dark';
  colors: ColorPalette;
  setTheme: (mode: ThemeMode) => void;
  toggleTheme: () => void;
}

interface ColorContrastResult {
  foreground: string;
  background: string;
  ratio: number;
  passesAA: boolean;
  passesAALarge: boolean;
  passesAAA: boolean;
  passesAAALarge: boolean;
}

interface ColorPaletteConfig {
  light: Record<string, string>;
  dark: Record<string, string>;
}
```

## 2. Logic & Algorithms

### 2.1 Theme Initialization Algorithm

```
1. On component mount:
   a. Read 'theme' from localStorage
   b. If localStorage value exists:
      i. Parse stored value as ThemeMode
      ii. Set mode to stored value
   c. Else:
      i. Read navigator.mediaQuery.prefers-color-scheme
      ii. Set mode to 'system'

2. Resolve actual theme:
   a. If mode is 'system':
      i. Check window.matchMedia('(prefers-color-scheme: dark)')
      ii. If matches: resolvedTheme = 'dark'
      iii. Else: resolvedTheme = 'light'
   b. Else: resolvedTheme = mode

3. Apply color tokens:
   a. For each token in ColorPalette:
      i. Get value from appropriate theme object
      ii. Set CSS custom property on document.documentElement
```

### 2.2 Color Contrast Validation Algorithm

```
1. Input: foregroundColor, backgroundColor

2. Calculate relative luminance for each color:
   a. Convert hex to RGB
   b. Apply sRGB transformation formula
   c. Calculate relative luminance L = 0.2126*R + 0.7152*G + 0.0722*B

3. Calculate contrast ratio:
   a. L1 = luminance of lighter color
   b. L2 = luminance of darker color
   c. ratio = (L1 + 0.05) / (L2 + 0.05)

4. Validate against WCAG 2.1 AA requirements:
   a. Normal text (≤18pt): requires ratio ≥ 4.5:1
   b. Large text (>18pt/bold>14pt): requires ratio ≥ 3:1
   c. UI components: requires ratio ≥ 3:1

5. Return validation result with all pass/fail states
```

### 2.3 Theme Switching Algorithm

```
1. User triggers theme change:
   a. Call setTheme(newMode)

2. Update localStorage:
   a. localStorage.setItem('theme', newMode)

3. Resolve new theme:
   a. If newMode is 'system':
      i. Check system preference
      ii. Set resolvedTheme accordingly
   b. Else: resolvedTheme = newMode

4. Update CSS custom properties:
   a. For each token in ColorPalette:
      i. Get value from resolvedTheme object
      ii. Set property on document.documentElement

5. Dispatch custom event 'themechange':
   a. event.detail = { mode, resolvedTheme }
```

### 2.4 Color Token Application Algorithm

```
1. Define token mappings:
   tokenMap = {
     '--bg-primary': { light: '#F7FCF7', dark: '#0A0F0A' },
     '--bg-surface': { light: '#FFFFFF', dark: '#161D16' },
     '--color-primary': { light: '#166534', dark: '#4ADE80' },
     '--color-secondary': { light: '#DCFCE7', dark: '#86EFAC' },
     '--color-accent': { light: '#F97316', dark: '#FFB86C' },
     '--color-error': { light: '#DC2626', dark: '#F87171' },
     '--text-primary': { light: '#111827', dark: '#F3F4F6' },
     '--text-muted': { light: '#6B7280', dark: '#9CA3AF' }
   }

2. On theme resolution complete:
   a. Iterate through tokenMap entries
   b. Get value based on resolvedTheme
   c. document.documentElement.style.setProperty(token, value)

3. Log all tokens applied (development mode only)
```

## 3. State Management & Error Handling

### 3.1 State Machine

```
States:
  - INITIALIZING: Reading stored preference and system preference
  - READY: Theme resolved and CSS variables applied
  - UPDATING: Theme change in progress
  - ERROR: Failed to read/write localStorage

Transitions:
  INITIALIZING → READY (on successful initialization)
  INITIALIZING → ERROR (on localStorage access failure)
  READY → UPDATING (on setTheme/toggleTheme call)
  UPDATING → READY (on CSS variables updated)
  UPDATING → ERROR (on CSS custom property set failure)
  ERROR → READY (on retry after clearing localStorage)
```

### 3.2 Error States

| Error | Cause | Handling |
| :--- | :--- | :--- |
| localStorage Access Denied | Privacy settings or disabled cookies | Fallback to system preference, log warning |
| Invalid Theme Value | Corrupted localStorage data | Clear localStorage, reset to 'system' |
| CSS Variable Set Failed | Stylesheet locked or CSP restriction | Log error, retry with inline styles |
| Contrast Validation Failed | Color combination doesn't meet WCAG | Log warning, suggest alternative colors |
| System Query Not Available | Browser doesn't support matchMedia | Fallback to 'light' theme |

### 3.3 Error Recovery Strategy

```typescript
function handleThemeError(error: ThemeError): void {
  switch (error.type) {
    case 'LOCALSTORAGE_ACCESS_DENIED':
      console.warn('localStorage unavailable, using system preference');
      themeStore.set({ mode: 'system', resolvedTheme: getSystemTheme() });
      break;
    case 'INVALID_STORED_VALUE':
      console.warn('Invalid theme in localStorage, resetting');
      localStorage.removeItem('theme');
      themeStore.set({ mode: 'system', resolvedTheme: getSystemTheme() });
      break;
    case 'CSS_VAR_SET_FAILED':
      console.error('Failed to set CSS custom property:', error.token);
      retryThemeApplication();
      break;
    case 'CONTRAST_VALIDATION_FAILED':
      console.warn('Color contrast issue:', error.details);
      notifyDesignTeam(error.details);
      break;
    case 'MATCHMEDIA_UNAVAILABLE':
      console.warn('matchMedia unavailable, defaulting to light');
      themeStore.set({ mode: 'light', resolvedTheme: 'light' });
      break;
  }
}
```

### 3.4 Accessibility Enforcement

- All color combinations validated at render time
- Failed validations logged with specific color pairs
- Development mode: Overlay showing contrast ratio on hover
- Production mode: Log to error tracking service
- Build-time check: Static analysis of color token combinations

## 4. Component Interfaces

### 4.1 ThemeProvider Component

```typescript
interface ThemeProviderProps {
  defaultMode?: ThemeMode;
  onThemeChange?: (theme: ThemeContextValue) => void;
  children: ComponentChildren;
}

function ThemeProvider(props: ThemeProviderProps): ComponentReturnType;
```

### 4.2 useColorPalette Hook

```typescript
function useColorPalette(): {
  colors: ColorPalette;
  mode: ThemeMode;
  resolvedTheme: 'light' | 'dark';
  setTheme: (mode: ThemeMode) => void;
  toggleTheme: () => void;
  validateContrast: (foreground: string, background: string) => ColorContrastResult;
};
```

### 4.3 Theme Initialization Module

```typescript
function initializeTheme(): Promise<ThemeContextValue>;

function getSystemTheme(): 'light' | 'dark';

function resolveTheme(mode: ThemeMode): 'light' | 'dark';

function applyColorTokens(theme: 'light' | 'dark'): void;
```

### 4.4 Color Token Module

```typescript
function getColorTokens(theme: 'light' | 'dark'): Record<string, string>;

function getTokenValue(token: string, theme: 'light' | 'dark'): string;

function setCssVariable(token: string, value: string): void;

function setAllCssVariables(theme: 'light' | 'dark'): void;
```

### 4.5 Contrast Validation Module

```typescript
function hexToRgb(hex: string): { r: number; g: number; b: number } | null;

function calculateLuminance(r: number, g: number, b: number): number;

function calculateContrastRatio(color1: string, color2: string): number;

function validateContrast(
  foreground: string,
  background: string
): ColorContrastResult;

function validateColorPalette(palette: ColorPalette): {
  valid: boolean;
  failures: Array<{
    token: string;
    foreground: string;
    background: string;
    ratio: number;
    required: number;
  }>;
};
```

### 4.6 Theme Persistence Module

```typescript
function saveThemeToStorage(mode: ThemeMode): void;

function loadThemeFromStorage(): ThemeMode | null;

function clearThemeFromStorage(): void;

function subscribeToSystemThemeChange(callback: (theme: 'light' | 'dark') =>ubscribeFunction;
```

 void): Uns### 4.7 Color Palette Constants

```typescript
const COLOR_TOKENS: ColorPaletteConfig = {
  light: {
    '--bg-primary': '#F7FCF7',
    '--bg-surface': '#FFFFFF',
    '--color-primary': '#166534',
    '--color-secondary': '#DCFCE7',
    '--color-accent': '#F97316',
    '--color-error': '#DC2626',
    '--text-primary': '#111827',
    '--text-muted': '#6B7280'
  },
  dark: {
    '--bg-primary': '#0A0F0A',
    '--bg-surface': '#161D16',
    '--color-primary': '#4ADE80',
    '--color-secondary': '#86EFAC',
    '--color-accent': '#FFB86C',
    '--color-error': '#F87171',
    '--text-primary': '#F3F4F6',
    '--text-muted': '#9CA3AF'
  }
};

const WCAG_REQUIREMENTS = {
  normalText: 4.5,
  largeText: 3.0,
  uiComponents: 3.0
};
```
