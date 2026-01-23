# ComponentStyles

**Traceability:** ARCH-016

## 1. Data Structures & Types

### 1.1 Theme Mode Type
```typescript
type ThemeMode = 'light' | 'dark' | 'system';
```

### 1.2 Theme Context Interface
```typescript
interface ThemeContextValue {
  mode: ThemeMode;
  resolvedTheme: 'light' | 'dark';
  setMode: (mode: ThemeMode) => void;
  toggleTheme: () => void;
  systemPreference: 'light' | 'dark';
}
```

### 1.3 Color Token Map
```typescript
interface ColorTokens {
  bgPrimary: string;
  bgSurface: string;
  colorPrimary: string;
  colorSecondary: string;
  colorAccent: string;
  colorError: string;
  textPrimary: string;
  textMuted: string;
}

interface ThemeColors {
  light: ColorTokens;
  dark: ColorTokens;
}
```

### 1.4 Component Variant Types
```typescript
type ButtonVariant = 'primary' | 'secondary' | 'accent' | 'error' | 'ghost';
type ButtonSize = 'sm' | 'md' | 'lg';
type CardVariant = 'elevated' | 'outlined' | 'filled';
type BadgeVariant = 'success' | 'warning' | 'error' | 'info' | 'neutral';
```

### 1.5 Typography Scale Interface
```typescript
interface TypographyScale {
  fontSize: {
    xs: string;
    sm: string;
    base: string;
    lg: string;
    xl: string;
    '2xl': string;
    '3xl': string;
    '4xl': string;
  };
  fontWeight: {
    normal: number;
    medium: number;
    semibold: number;
    bold: number;
  };
  lineHeight: {
    tight: string;
    normal: string;
    relaxed: string;
  };
}
```

### 1.6 Spacing Scale Interface
```typescript
interface SpacingScale {
  0: string;
  1: string;
  2: string;
  3: string;
  4: string;
  5: string;
  6: string;
  8: string;
  10: string;
  12: string;
  16: string;
}
```

### 1.7 Breakpoint Interface
```typescript
interface Breakpoints {
  sm: string;
  md: string;
  lg: string;
  xl: string;
  '2xl': string;
}
```

### 1.8 Component Style Config Interface
```typescript
interface ButtonStyleConfig {
  variant: ButtonVariant;
  size: ButtonSize;
  fullWidth: boolean;
  disabled: boolean;
  loading: boolean;
}

interface CardStyleConfig {
  variant: CardVariant;
  padding: keyof SpacingScale;
  hoverable: boolean;
  clickable: boolean;
}

interface BadgeStyleConfig {
  variant: BadgeVariant;
  size: ButtonSize;
  rounded: boolean;
}
```

## 2. Logic & Algorithms

### 2.1 Theme Initialization Algorithm

```
1. ON COMPONENT MOUNT:
   a. Read 'theme' from localStorage
   b. IF localStorage value exists:
      - SET mode TO localStorage value
   c. ELSE:
      - SET mode TO 'system'
   d. READ navigator.mediaQuery.prefersColorSchemed
   e. SET systemPreference TO 'dark' OR 'light'
   f. RESOLVE resolvedTheme:
      - IF mode === 'system':
        - resolvedTheme = systemPreference
      - ELSE:
        - resolvedTheme = mode
   g. APPLY resolvedTheme CSS custom properties TO document.documentElement

2. ON THEME CHANGE:
   a. UPDATE localStorage WITH new mode
   b. RESOLVE new resolvedTheme
   c. APPLY CSS custom properties
   d. EMIT themechange event FOR sibling components
```

### 2.2 CSS Custom Property Application Algorithm

```
APPLY THEME COLORS(theme: 'light' | 'dark'):
1. GET colorTokens FROM ThemeColors[theme]
2. FOR EACH key IN colorTokens:
   a. SET CSS custom property --bg-primary, --color-primary, etc.
   b. TO corresponding token value ON document.documentElement.style
3. ATTACH media query listener FOR system preference changes
   a. IF mode === 'system':
      - RE-RESOLVE resolvedTheme
      - RE-APPLY theme colors
```

### 2.3 Button Style Resolution Algorithm

```
GET BUTTON STYLES(config: ButtonStyleConfig):
1. DETERMINE background color:
   a. SWITCH config.variant:
      - 'primary': var(--color-primary)
      - 'secondary': var(--color-secondary)
      - 'accent': var(--color-accent)
      - 'error': var(--color-error)
      - 'ghost': 'transparent'

2. DETERMINE text color:
   a. SWITCH config.variant:
      - 'primary': WHITE FOR dark theme, BLACK FOR light theme
      - 'secondary': var(--text-primary)
      - 'accent': BLACK
      - 'error': WHITE
      - 'ghost': var(--text-primary)

3. DETERMINE padding:
   a. SWITCH config.size:
      - 'sm': '0.5rem 1rem'
      - 'md': '0.75rem 1.5rem'
      - 'lg': '1rem 2rem'

4. DETERMINE font size:
   a. SWITCH config.size:
      - 'sm': var(--font-size-sm)
      - 'md': var(--font-size-base)
      - 'lg': var(--font-size-lg)

5. APPLY fullWidth modifier:
   a. IF config.fullWidth:
      - width: 100%
      - display: flex
      - justify-content: center

6. APPLY disabled state:
   a. IF config.disabled:
      - opacity: 0.5
      - cursor: not-allowed
      - pointer-events: none

7. APPLY loading state:
   a. IF config.loading:
      - ADD spinner icon
      - ADD text-muted overlay
```

### 2.4 Card Style Resolution Algorithm

```
GET CARD STYLES(config: CardStyleConfig):
1. DETERMINE base styles:
   a. background: var(--bg-surface)
   b. border-radius: var(--radius-md, 8px)
   c. padding: var(--spacing-{config.padding})

2. DETERMINE variant styles:
   a. SWITCH config.variant:
      - 'elevated':
        - box-shadow: var(--shadow-lg)
        - border: none
      - 'outlined':
        - border: 1px solid var(--color-border)
        - box-shadow: none
      - 'filled':
        - background: var(--bg-primary)
        - border: none
        - box-shadow: none

3. DETERMINE hover effects:
   a. IF config.hoverable:
      - ON hover:
        - transform: translateY(-2px)
        - box-shadow: var(--shadow-xl)
        - transition: all 0.2s ease

4. DETERMINE clickable indicators:
   a. IF config.clickable:
      - cursor: pointer
      - ADD focus ring FOR accessibility
```

### 2.5 Badge Style Resolution Algorithm

```
GET BADGE STYLES(config: BadgeStyleConfig):
1. DETERMINE background color:
   a. SWITCH config.variant:
      - 'success': var(--color-success)
      - 'warning': var(--color-warning)
      - 'error': var(--color-error)
      - 'info': var(--color-primary)
      - 'neutral': var(--color-secondary)

2. DETERMINE text color:
   a. SWITCH config.variant:
      - 'success', 'error', 'info': WHITE
      - 'warning', 'neutral': var(--text-primary)

3. DETERMINE padding:
   a. SWITCH config.size:
      - 'sm': '0.125rem 0.5rem'
      - 'md': '0.25rem 0.75rem'
      - 'lg': '0.375rem 1rem'

4. DETERMINE border radius:
   a. IF config.rounded:
      - border-radius: 9999px
   b. ELSE:
      - border-radius: var(--radius-sm, 4px)

5. DETERMINE font properties:
   a. font-size: var(--font-size-{config.size})
   b. font-weight: var(--font-weight-medium)
```

### 2.6 Typography Style Application Algorithm

```
APPLY TYPOGRAPHY(styles: TypographyStyleConfig):
1. FOR heading elements (h1-h6):
   a. SET font-family: var(--font-family-sans)
   b. SET font-weight: var(--font-weight-bold)
   c. SET line-height: var(--line-height-tight)
   d. SET letter-spacing: -0.025em FOR h1-h3

2. FOR body text:
   a. SET font-family: var(--font-family-sans)
   b. SET font-weight: var(--font-weight-normal)
   c. SET line-height: var(--line-height-normal)

3. FOR captions and small text:
   a. SET font-size: var(--font-size-sm)
   b. SET color: var(--text-muted)
```

## 3. State Management & Error Handling

### 3.1 Theme Context State Machine

```
STATE: ThemeContext
├── mode: 'light' | 'dark' | 'system'
├── resolvedTheme: 'light' | 'dark'
├── systemPreference: 'light' | 'dark'
├── error: ThemeError | null
└── loading: boolean

STATE TRANSITIONS:
1. INITIALIZING → READY
   - Trigger: Mount complete, preferences read
   - Action: Apply resolved theme colors

2. READY → UPDATING
   - Trigger: User calls setMode()
   - Action: Persist to localStorage, resolve new theme

3. UPDATING → READY
   - Trigger: Theme colors applied
   - Action: Emit themechange event

4. READY → ERROR
   - Trigger: localStorage unavailable, invalid color value
   - Action: Fallback to system preference, log error
```

### 3.2 Error States

| Error Type | Condition | Handling |
| :--- | :--- | :--- |
| ThemePersistenceError | localStorage access fails | Fallback to system preference, log warning |
| InvalidColorToken | CSS variable has invalid value | Use fallback color, log error |
| ContrastViolation | Color combination fails WCAG AA | Log warning, suggest alternative palette |
| SystemPreferenceUnavailable | navigator.mediaQuery unavailable | Default to light theme |
| ThemeTransitionTimeout | CSS transition exceeds 500ms | Force apply colors, clean up listeners |

### 3.3 Accessibility Validation Rules

```
VALIDATE COLOR CONTRAST(fg: string, bg: string, context: string):
1. CALCULATE luminance ratio:
   a. Convert hex to RGB
   b. Apply sRGB luminance formula
   c. Compute ratio: (L1 + 0.05) / (L2 + 0.05)

2. CHECK AGAINST WCAG 2.1 AA:
   a. IF context === 'normal-text':
      - REQUIRE ratio >= 4.5:1
   b. IF context === 'large-text':
      - REQUIRE ratio >= 3:1
   c. IF context === 'ui-components':
      - REQUIRE ratio >= 3:1

3. IF VIOLATION DETECTED:
   a. LOG warning with token names
   b. SUGGEST alternative token from palette
```

### 3.4 Component State Management

```
BUTTON COMPONENT STATE:
├── idle: Default appearance
├── hover: Mouse over, apply hover styles
├── active: Mouse down, apply active styles
├── focus: Keyboard focus, apply focus ring
├── disabled: Disabled prop, apply disabled styles
└── loading: Loading prop, apply loading styles

CARD COMPONENT STATE:
├── default: Base variant styles
├── hover: Hoverable variant, mouse over
├── active: Clickable variant, mouse down
└── focus: Clickable variant, keyboard focus

BADGE COMPONENT STATE:
├── default: Variant styles
└── disabled: Parent disabled, reduced opacity
```

## 4. Component Interfaces

### 4.1 ThemeProvider Interface

```typescript
interface ThemeProviderProps {
  children: React.ReactNode;
  defaultMode?: ThemeMode;
  storageKey?: string;
  onThemeChange?: (mode: ThemeMode, resolvedTheme: 'light' | 'dark') => void;
}

class ThemeProvider {
  props: ThemeProviderProps;

  constructor(props: ThemeProviderProps);

  render(): JSX.Element;

  getThemeContext(): ThemeContextValue;

  private handleSystemPreferenceChange(event: MediaQueryListEvent): void;

  private persistTheme(mode: ThemeMode): void;

  private resolveTheme(mode: ThemeMode): 'light' | 'dark';

  private applyThemeColors(theme: 'light' | 'dark'): void;
}
```

### 4.2 useTheme Hook Interface

```typescript
interface UseThemeReturn {
  mode: ThemeMode;
  resolvedTheme: 'light' | 'dark';
  setMode: (mode: ThemeMode) => void;
  toggleTheme: () => void;
  systemPreference: 'light' | 'dark';
}

function useTheme(): UseThemeReturn;

function useThemeMode(): ThemeMode;

function useResolvedTheme(): 'light' | 'dark';
```

### 4.3 Button Component Interface

```typescript
interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  fullWidth?: boolean;
  loading?: boolean;
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
  children: React.ReactNode;
}

interface ButtonStyleProps {
  variant: ButtonVariant;
  size: ButtonSize;
  fullWidth: boolean;
  disabled: boolean;
  loading: boolean;
}

class Button {
  props: ButtonProps;

  constructor(props: ButtonProps);

  render(): JSX.Element;

  private getVariantStyles(): React.CSSProperties;

  private getSizeStyles(): React.CSSProperties;

  private getLoadingStyles(): React.CSSProperties;

  private handleClick(event: React.MouseEvent<HTMLButtonElement>): void;
}
```

### 4.4 Card Component Interface

```typescript
interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: CardVariant;
  padding?: keyof SpacingScale;
  hoverable?: boolean;
  clickable?: boolean;
  onClick?: () => void;
  children: React.ReactNode;
}

interface CardStyleProps {
  variant: CardVariant;
  padding: keyof SpacingScale;
  hoverable: boolean;
  clickable: boolean;
}

class Card {
  props: CardProps;

  constructor(props: CardProps);

  render(): JSX.Element;

  private getVariantStyles(): React.CSSProperties;

  private getHoverStyles(): React.CSSProperties;

  private handleClick(): void;
}
```

### 4.5 Badge Component Interface

```typescript
interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: BadgeVariant;
  size?: ButtonSize;
  rounded?: boolean;
  children: React.ReactNode;
}

interface BadgeStyleProps {
  variant: BadgeVariant;
  size: ButtonSize;
  rounded: boolean;
}

class Badge {
  props: BadgeProps;

  constructor(props: BadgeProps);

  render(): JSX.Element;

  private getVariantStyles(): React.CSSProperties;

  private getSizeStyles(): React.CSSProperties;
}
```

### 4.6 Style Utilities Interface

```typescript
namespace StyleUtils {
  function getColorToken(token: keyof ColorTokens): string;

  function getSpacing(size: keyof SpacingScale): string;

  function getTypography(scale: keyof TypographyScale): TypographyScale[keyof TypographyScale];

  function getBreakpoint(name: keyof Breakpoints): string;

  function applyTheme(theme: 'light' | 'dark'): void;

  function getCSSVariable(name: string): string;

  function setCSSVariable(name: string, value: string): void;

  function resolveColor(hex: string, theme: 'light' | 'dark'): string;

  function checkContrast(fg: string, bg: string): number;
}
```

### 4.7 CSS Custom Properties Registry

```typescript
interface CSSCustomProperties {
  '--bg-primary': string;
  '--bg-surface': string;
  '--color-primary': string;
  '--color-secondary': string;
  '--color-accent': string;
  '--color-error': string;
  '--text-primary': string;
  '--text-muted': string;
  '--font-family-sans': string;
  '--font-size-xs': string;
  '--font-size-sm': string;
  '--font-size-base': string;
  '--font-size-lg': string;
  '--font-size-xl': string;
  '--font-size-2xl': string;
  '--font-size-3xl': string;
  '--font-size-4xl': string;
  '--font-weight-normal': number;
  '--font-weight-medium': number;
  '--font-weight-semibold': number;
  '--font-weight-bold': number;
  '--line-height-tight': string;
  '--line-height-normal': string;
  '--line-height-relaxed': string;
  '--spacing-0': string;
  '--spacing-1': string;
  '--spacing-2': string;
  '--spacing-3': string;
  '--spacing-4': string;
  '--spacing-5': string;
  '--spacing-6': string;
  '--spacing-8': string;
  '--spacing-10': string;
  '--spacing-12': string;
  '--spacing-16': string;
  '--radius-sm': string;
  '--radius-md': string;
  '--radius-lg': string;
  '--shadow-sm': string;
  '--shadow-md': string;
  '--shadow-lg': string;
  '--shadow-xl': string;
}

declare global {
  interface Document {
    element.style: CSSCustomProperties & HTMLElement['style'];
  }
}
```
