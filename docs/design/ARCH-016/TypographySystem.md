# TypographySystem

**Traceability:** ARCH-016

## 1. Data Structures & Types

```typescript
interface TypographyScale {
  fontFamily: {
    sans: string;
    mono: string;
  };
  fontSize: {
    xs: string;
    sm: string;
    base: string;
    lg: string;
    xl: string;
    '2xl': string;
    '3xl': string;
    '4xl': string;
    '5xl': string;
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
  letterSpacing: {
    tight: string;
    normal: string;
    wide: string;
  };
}

interface TypographyTokens {
  --font-sans: string;
  --font-mono: string;
  --text-xs: string;
  --text-sm: string;
  --text-base: string;
  --text-lg: string;
  --text-xl: string;
  --text-2xl: string;
  --text-3xl: string;
  --text-4xl: string;
  --text-5xl: string;
  --font-normal: number;
  --font-medium: number;
  --font-semibold: number;
  --font-bold: number;
  --leading-tight: string;
  --leading-normal: string;
  --leading-relaxed: string;
  --tracking-tight: string;
  --tracking-normal: string;
  --tracking-wide: string;
}

interface TypographyConfig {
  scale: 'minor-third' | 'major-third' | 'perfect-fourth' | 'golden-ratio';
  baseSize: number;
  fontFamilies: {
    primary: string;
    secondary: string;
    mono: string;
  };
  customSizes?: Record<string, string>;
}

type TypographyVariant =
  | 'h1'
  | 'h2'
  | 'h3'
  | 'h4'
  | 'h5'
  | 'h6'
  | 'body'
  | 'body-sm'
  | 'body-lg'
  | 'caption'
  | 'overline';

interface TypographyStyle {
  fontSize: string;
  fontWeight: number;
  lineHeight: string;
  letterSpacing: string;
  textTransform?: 'uppercase' | 'lowercase' | 'capitalize';
  fontFamily: string;
}

interface TypographyVariantMap {
  [key in TypographyVariant]: TypographyStyle;
}
```

## 2. Logic & Algorithms

### 2.1 Typography Scale Generation Algorithm

```
FUNCTION generateTypographyScale(baseSize: number, scaleRatio: number): TypographyScale

  sizes := [
    'xs',
    'sm',
    'base',
    'lg',
    'xl',
    '2xl',
    '3xl',
    '4xl',
    '5xl'
  ]

  baseIndex := sizes.indexOf('base')
  sizeMap := {}

  FOR i FROM 0 TO sizes.length - 1 DO
    exponent := i - baseIndex
    IF exponent < 0 THEN
      sizeMap[sizes[i]] := ROUND(baseSize / POW(scaleRatio, ABS(exponent))) + 'px'
    ELSE IF exponent = 0 THEN
      sizeMap[sizes[i]] := baseSize + 'px'
    ELSE
      sizeMap[sizes[i]] := ROUND(baseSize * POW(scaleRatio, exponent)) + 'px'
    END IF
  END FOR

  RETURN {
    fontFamily: {
      sans: 'Inter, system-ui, -apple-system, sans-serif',
      mono: 'JetBrains Mono, Menlo, monospace'
    },
    fontSize: sizeMap,
    fontWeight: {
      normal: 400,
      medium: 500,
      semibold: 600,
      bold: 700
    },
    lineHeight: {
      tight: '1.25',
      normal: '1.5',
      relaxed: '1.75'
    },
    letterSpacing: {
      tight: '-0.025em',
      normal: '0',
      wide: '0.025em'
    }
  }
END FUNCTION
```

### 2.2 Typography Tokens CSS Generation Algorithm

```
FUNCTION generateCSSTokens(scale: TypographyScale): TypographyTokens

  tokens := {} AS TypographyTokens

  tokens['--font-sans'] := scale.fontFamily.sans
  tokens['--font-mono'] := scale.fontFamily.mono

  FOR EACH size IN Object.keys(scale.fontSize) DO
    tokens['--text-' + size] := scale.fontSize[size]
  END FOR

  FOR EACH weight IN Object.keys(scale.fontWeight) DO
    tokens['--font-' + weight] := scale.fontWeight[weight]
  END FOR

  tokens['--leading-tight'] := scale.lineHeight.tight
  tokens['--leading-normal'] := scale.lineHeight.normal
  tokens['--leading-relaxed'] := scale.lineHeight.relaxed

  tokens['--tracking-tight'] := scale.letterSpacing.tight
  tokens['--tracking-normal'] := scale.letterSpacing.normal
  tokens['--tracking-wide'] := scale.letterSpacing.wide

  RETURN tokens
END FUNCTION
```

### 2.3 Variant Style Resolution Algorithm

```
FUNCTION resolveTypographyVariant(variant: TypographyVariant, variantMap: TypographyVariantMap): TypographyStyle

  IF variantMap[variant] DOES NOT EXIST THEN
    RETURN variantMap['body']
  END IF

  RETURN variantMap[variant]
END FUNCTION
```

### 2.4 Responsive Typography Adjustment Algorithm

```
FUNCTION applyResponsiveTypography(
  baseStyle: TypographyStyle,
  viewport: { width: number; height: number },
  breakpoint: 'sm' | 'md' | 'lg' | 'xl' | '2xl'
): TypographyStyle

  scaleFactors := {
    sm: 0.875,
    md: 0.9375,
    lg: 1,
    xl: 1,
    '2xl': 1
  }

  factor := scaleFactors[breakpoint]

  baseFontSize := PARSE_FLOAT(REPLACE(baseStyle.fontSize, 'px', ''))
  adjustedSize := ROUND(baseFontSize * factor)

  adjustedStyle := {...baseStyle}
  adjustedStyle.fontSize := adjustedSize + 'px'

  RETURN adjustedStyle
END FUNCTION
```

## 3. State Management & Error Handling

### 3.1 Possible Error States

| Error State | Condition | Severity |
| :--- | :--- | :--- |
| `INVALID_BASE_SIZE` | baseSize <= 0 or baseSize > 100 | fatal |
| `INVALID_SCALE_RATIO` | scaleRatio <= 1 or scaleRatio >= 2 | fatal |
| `MISSING_FONT_FAMILY` | any font family is empty string | warning |
| `INVALID_FONT_SIZE` | computed font size is NaN or Infinity | warning |
| `CONTRAST_VIOLATION` | text color on background fails WCAG AA | error |
| `INVALID_BREAKPOINT` | unknown breakpoint name provided | warning |
| `VARIANT_NOT_FOUND` | requested typography variant undefined | warning |

### 3.2 State Transitions

```
STATE MACHINE TypographySystemState

  STATES: uninitialized, initializing, ready, error

  TRANSITIONS:

    uninitialized --(initialize)--> initializing

    initializing --(scaleGenerated)--> ready
    initializing --(initializationError)--> error

    ready --(updateTheme)--> initializing
    ready --(updateViewport)--> ready
    ready --(invalidConfig)--> error

    error --(retryInitialization)--> initializing
END STATE MACHINE
```

### 3.3 Error Handling Strategy

```typescript
class TypographySystemError extends Error {
  code: TypographyErrorCode;
  context: Record<string, unknown>;

  constructor(
    code: TypographyErrorCode,
    message: string,
    context: Record<string, unknown> = {}
  ) {
    super(message);
    this.code = code;
    this.context = context;
    this.name = 'TypographySystemError';
  }
}

FUNCTION validateTypographyConfig(config: TypographyConfig): ValidationResult

  errors := []

  IF config.baseSize <= 0 OR config.baseSize > 100 THEN
    errors.push({
      code: 'INVALID_BASE_SIZE',
      message: 'Base size must be between 1 and 100 pixels'
    })
  END IF

  validRatios := [1.067, 1.125, 1.2, 1.25, 1.333, 1.414, 1.5, 1.618]
  IF NOT validRatios.includes(config.scaleRatio) AND config.scaleRatio !== 'custom' THEN
    errors.push({
      code: 'INVALID_SCALE_RATIO',
      message: 'Scale ratio must be a valid musical interval ratio'
    })
  END IF

  FOR EACH font IN Object.values(config.fontFamilies) DO
    IF font === '' OR font === undefined THEN
      errors.push({
        code: 'MISSING_FONT_FAMILY',
        message: 'All font families must be defined'
      })
    END IF
  END FOR

  RETURN {
    valid: errors.length === 0,
    errors
  }
END FUNCTION
```

## 4. Component Interfaces

### 4.1 TypographyProvider (Svelte Store)

```typescript
import { writable, derived, type Writable, type Readable } from 'svelte/store';

type ThemeMode = 'light' | 'dark';

interface TypographyContextValue {
  scale: Readable<TypographyScale>;
  tokens: Readable<TypographyTokens>;
  variants: Readable<TypographyVariantMap>;
  currentTheme: Writable<ThemeMode>;
  updateViewport: (width: number, height: number) => void;
  getVariantStyle: (variant: TypographyVariant) => TypographyStyle;
}

function createTypographyProvider(
  initialConfig: TypographyConfig,
  themeMode: ThemeMode
): TypographyContextValue
```

### 4.2 Typography Utilities Module

```typescript
function getFontSizeToken(size: keyof TypographyScale['fontSize']): string

function getFontWeightToken(weight: keyof TypographyScale['fontWeight']): number

function getLineHeightToken(height: keyof TypographyScale['lineHeight']): string

function getLetterSpacingToken(spacing: keyof TypographyScale['letterSpacing']): string

function applyTypographyStyles(
  element: HTMLElement,
  variant: TypographyVariant
): void

function calculateRelativeFontSize(
  baseSize: number,
  targetSizeInPixels: number
): string

function generateFluidTypography(
  minViewport: number,
  maxViewport: number,
  minSize: number,
  maxSize: number
): string

function validateTextContrast(
  textColor: string,
  backgroundColor: string,
  level: 'AA' | 'AAA' = 'AA'
): boolean
```

### 4.3 Tailwind Typography Plugin Configuration

```typescript
import type { TypographyConfig } from './types';

function generateTailwindTypographyPlugin(config: TypographyConfig): object

function extendTailwindConfig(
  userConfig: object,
  typographyConfig: TypographyConfig
): object
```

### 4.4 Typography Component (Svelte)

```svelte
<script lang="ts">
  import type { TypographyVariant } from './types';

  export let variant: TypographyVariant = 'body';
  export let as: string = 'p';
  export let theme: 'light' | 'dark' | 'system' = 'system';
  export let className: string = '';

  import { typographyContext } from './context';

  $: style = $typographyContext.getVariantStyle(variant);
  $: resolvedClasses = `${className} typography-${variant}`;
</script>

<svelte:element
  this={as}
  class={resolvedClasses}
  style:font-size={style.fontSize}
  style:font-weight={style.fontWeight}
  style:line-height={style.lineHeight}
  style:letter-spacing={style.letterSpacing}
  style:font-family={style.fontFamily}
>
  <slot />
</svelte:element>
```

### 4.5 Typography Hook (for non-component contexts)

```typescript
import { getContext } from 'svelte';
import type { TypographyContextValue } from './TypographyProvider';

function useTypography(): TypographyContextValue {
  const context = getContext<TypographyContextValue>('typography');
  return context;
}
```
