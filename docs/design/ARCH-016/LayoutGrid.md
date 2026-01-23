# LayoutGrid

**Traceability:** ARCH-016

## 1. Data Structures & Types

```typescript
interface GridBreakpoints {
  sm: number;
  md: number;
  lg: number;
  xl: number;
  '2xl': number;
}

interface GridConfig {
  columns: number;
  gutter: number;
  margin: number;
  maxWidth: number;
}

interface GridBreakpointConfig {
  columns: number;
  gutter: number;
  margin: number;
}

type GridSpan = number | 'full' | 'auto';

interface GridItemProps {
  span?: GridSpan;
  start?: number;
  end?: number;
  hideBelow?: keyof GridBreakpoints;
  showBelow?: keyof GridBreakpoints;
}

const DEFAULT_BREAKPOINTS: GridBreakpoints = {
  sm: 640,
  md: 768,
  lg: 1024,
  xl: 1280,
  '2xl': 1536,
};

const DEFAULT_CONFIG: GridConfig = {
  columns: 12,
  gutter: 16,
  margin: 16,
  maxWidth: 1280,
};

const BREAKPOINT_CONFIGS: Record<keyof GridBreakpoints, GridBreakpointConfig> = {
  sm: { columns: 4, gutter: 12, margin: 12 },
  md: { columns: 8, gutter: 16, margin: 20 },
  lg: { columns: 12, gutter: 20, margin: 24 },
  xl: { columns: 12, gutter: 24, margin: 32 },
  '2xl': { columns: 12, gutter: 24, margin: 32 },
};

interface ThemeContext {
  theme: 'light' | 'dark' | 'system';
  resolvedTheme: 'light' | 'dark';
  breakpoints: GridBreakpoints;
}
```

## 2. Logic & Algorithms

### 2.1 Grid Container Initialization

```
1. Import BREAKPOINT_CONFIGS from theme constants
2. Read viewport width from window.innerWidth
3. Determine active breakpoint by comparing viewport against BREAKPOINTS thresholds
4. Apply corresponding grid CSS custom properties:
   - --grid-columns: columns from active breakpoint config
   - --grid-gutter: gutter from active breakpoint config
   - --grid-margin: margin from active breakpoint config
5. Set container max-width and padding based on config
6. Create media query listeners for each breakpoint
7. On breakpoint change, update CSS custom properties
```

### 2.2 Grid Item Span Calculation

```
FOR each grid item:
1. Parse span value:
   IF span is 'full':
      span = --grid-columns
   ELSE IF span is 'auto':
      Calculate based on remaining space
      Default to 1 column if no explicit span
   ELSE:
      Validate span is between 1 and --grid-columns
      Clamp value if outside valid range

2. Calculate column position:
   IF start is specified:
      column-start = start
      column-end = start + span
   ELSE IF end is specified:
      column-end = end
      column-start = end - span
   ELSE:
      column-start = auto-placement (CSS grid algorithm)
      column-end = auto

3. Apply responsive hiding:
   FOR each hideBelow breakpoint:
      Add display: none for viewports below threshold
   FOR each showBelow breakpoint:
      Add display: none for viewports at or above threshold
```

### 2.3 Breakpoint Detection Algorithm

```
FUNCTION detectActiveBreakpoint(viewportWidth: number): keyof GridBreakpoints
   FOR breakpoint IN ['2xl', 'xl', 'lg', 'md', 'sm'] IN REVERSE ORDER:
      IF viewportWidth >= BREAKPOINTS[breakpoint]:
         RETURN breakpoint
   RETURN 'sm'
END FUNCTION

FUNCTION setupBreakpointListeners():
   breakpoints = ['sm', 'md', 'lg', 'xl', '2xl']
   FOR EACH breakpoint IN breakpoints:
      query = `(min-width: ${BREAKPOINTS[breakpoint]}px)`
      mediaQueryList = window.matchMedia(query)
      
      handler = (event: MediaQueryListEvent) => {
         IF event.matches:
            activeBreakpoint = detectActiveBreakpoint(window.innerWidth)
            updateGridProperties(activeBreakpoint)
      }
      
      mediaQueryList.addEventListener('change', handler)
   END FOR
END FUNCTION
```

### 2.4 Responsive Grid CSS Generation

```css
.grid-container {
  display: grid;
  grid-template-columns: repeat(var(--grid-columns), 1fr);
  gap: var(--grid-gutter);
  padding-inline: var(--grid-margin);
  max-width: var(--grid-max-width, 1280px);
  margin-inline: auto;
}

@media (max-width: 640px) {
  .grid-container {
    grid-template-columns: 1fr;
    gap: 12px;
    padding-inline: 12px;
  }
}

@media (min-width: 641px) and (max-width: 768px) {
  .grid-container {
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
    padding-inline: 20px;
  }
}

@media (min-width: 769px) and (max-width: 1024px) {
  .grid-container {
    grid-template-columns: repeat(8, 1fr);
    gap: 20px;
    padding-inline: 24px;
  }
}

@media (min-width: 1025px) and (max-width: 1280px) {
  .grid-container {
    grid-template-columns: repeat(12, 1fr);
    gap: 20px;
    padding-inline: 24px;
  }
}

@media (min-width: 1281px) {
  .grid-container {
    grid-template-columns: repeat(12, 1fr);
    gap: 24px;
    padding-inline: 32px;
  }
}
```

### 2.5 Theme-Aware Grid Styling

```
FUNCTION applyThemeAwareStyles(theme: 'light' | 'dark'):
   IF theme === 'light':
      document.documentElement.style.setProperty('--grid-bg-primary', '#F7FCF7');
      document.documentElement.style.setProperty('--grid-bg-surface', '#FFFFFF');
      document.documentElement.style.setProperty('--grid-border-color', '#E5E7EB');
   ELSE IF theme === 'dark':
      document.documentElement.style.setProperty('--grid-bg-primary', '#0A0F0A');
      document.documentElement.style.setProperty('--grid-bg-surface', '#161D16');
      document.documentElement.style.setProperty('--grid-border-color', '#2D332D');
   END IF
END FUNCTION
```

## 3. State Management & Error Handling

### 3.1 Possible Error States

| Error State | Cause | Recovery Strategy |
|-------------|-------|-------------------|
| Invalid span value | span > columns or span < 1 | Clamp to valid range, log warning |
| Viewport undefined | SSR or missing window object | Use default breakpoint, defer to client |
| Breakpoint mismatch | Window resize during animation | Debounce resize handler, re-evaluate |
| Theme transition flicker | CSS custom property race condition | Use CSS `transition` on color tokens only |
| Grid overflow | Fixed content > available columns | Allow horizontal scroll, warn in console |
| Missing breakpoint config | Undefined breakpoint key | Fallback to sm config, log error |

### 3.2 State Transitions

```
INITIAL STATE:
   - breakpoint: 'sm' (detected from window width)
   - theme: 'system' (from localStorage or default)
   - config: BREAKPOINT_CONFIGS['sm']

STATE: Breakpoint Change
   TRIGGER: viewport crosses breakpoint threshold
   ACTION:
      1. Debounce handler fires (100ms delay)
      2. Detect new breakpoint
      3. Update CSS custom properties
      4. Emit 'breakpointChange' event
   VALIDATION: Ensure properties exist before setting

STATE: Theme Change
   TRIGGER: User toggles theme or system preference changes
   ACTION:
      1. Read new theme preference
      2. Resolve to light/dark
      3. Apply theme-aware grid styles
      4. Emit 'themeChange' event
   VALIDATION: Validate CSS property names exist

STATE: Grid Item Mount
   TRIGGER: Grid item component initializes
   ACTION:
      1. Read props (span, start, end, hideBelow, showBelow)
      2. Calculate grid-column values
      3. Apply inline styles for positioning
      4. Apply responsive visibility classes
   VALIDATION: Check span does not exceed max columns
```

### 3.3 Error Boundary Implementation

```typescript
class GridErrorBoundary extends SvelteComponent {
  fallback: Component;

  onError(error: Error, errorInfo: ErrorInfo): void {
    console.error('[LayoutGrid] Error:', error.message);
    console.error('[LayoutGrid] Component:', errorInfo.componentStack);
    return this.fallback;
  }
}
```

## 4. Component Interfaces

### 4.1 GridContainer Component

```typescript
import type { SvelteComponent } from 'svelte';

interface GridContainerProps {
  as?: string;
  theme?: 'light' | 'dark' | 'system';
  maxWidth?: number | string;
  gap?: number | string;
  columns?: number;
  class?: string;
  children: Snippet;
}

class GridContainer extends SvelteComponent<GridContainerProps> {}

export function createGridContext(): {
  parent: Writable<GridContext>;
  children: Writable<GridContext>;
} {
  const gridContext = writable<GridContext>({
    columns: 12,
    theme: 'system',
    breakpoint: 'lg',
  });

  return {
    parent: gridContext,
    children: gridContext,
  };
}
```

### 4.2 GridItem Component

```typescript
import type { SvelteComponent } from 'svelte';
import type { GridSpan } from './types';

interface GridItemProps {
  span?: GridSpan;
  start?: number;
  end?: number;
  hideBelow?: keyof GridBreakpoints;
  showBelow?: keyof GridBreakpoints;
  align?: 'start' | 'center' | 'end' | 'stretch';
  class?: string;
  children: Snippet;
}

class GridItem extends SvelteComponent<GridItemProps> {}

export function calculateGridSpan(
  span: GridSpan,
  maxColumns: number
): { columnStart: number; columnEnd: number } {
  if (span === 'full') {
    return { columnStart: 1, columnEnd: maxColumns + 1 };
  }
  if (span === 'auto') {
    return { columnStart: 'auto', columnEnd: 'auto' };
  }
  const clampedSpan = Math.max(1, Math.min(span, maxColumns));
  return { columnStart: 'auto', columnEnd: `span ${clampedSpan}` };
}
```

### 4.3 GridStore (Svelte Store)

```typescript
import { writable, derived, type Writable, type Readable } from 'svelte/store';

interface GridState {
  breakpoint: keyof GridBreakpoints;
  theme: 'light' | 'dark' | 'system';
  resolvedTheme: 'light' | 'dark';
  viewportWidth: number;
}

function createGridStore(): {
  subscribe: Readable<GridState>;
  setBreakpoint: (breakpoint: keyof GridBreakpoints) => void;
  setTheme: (theme: 'light' | 'dark' | 'system') => void;
  initialize: () => void;
  destroy: () => void;
} {
  const initialState: GridState = {
    breakpoint: 'sm',
    theme: 'system',
    resolvedTheme: 'light',
    viewportWidth: 0,
  };

  const { subscribe, update } = writable<GridState>(initialState);

  let resizeObserver: ResizeObserver;
  let mediaQueryListeners: Map<string, MediaQueryList> = new Map();

  function setBreakpoint(breakpoint: keyof GridBreakpoints): void {
    update((state) => ({ ...state, breakpoint }));
  }

  function setTheme(theme: 'light' | 'dark' | 'system'): void {
    update((state) => ({ ...state, theme }));
  }

  function resolveTheme(theme: 'light' | 'dark' | 'system'): 'light' | 'dark' {
    if (theme !== 'system') return theme;
    if (typeof window === 'undefined') return 'light';
    return window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'light';
  }

  function initialize(): void {
    if (typeof window === 'undefined') return;

    update((state) => ({
      ...state,
      viewportWidth: window.innerWidth,
      breakpoint: detectActiveBreakpoint(window.innerWidth),
      resolvedTheme: resolveTheme(state.theme),
    }));

    setupMediaQueryListeners();
    setupResizeObserver();
  }

  function destroy(): void {
    mediaQueryListeners.forEach((listener) => {
      listener.removeEventListener?.('change', () => {});
    });
    resizeObserver?.disconnect();
  }

  return {
    subscribe,
    setBreakpoint,
    setTheme,
    initialize,
    destroy,
  };
}

export const gridStore = createGridStore();
```

### 4.4 Grid Utility Functions

```typescript
export function getBreakpointConfig(
  breakpoint: keyof GridBreakpoints
): GridBreakpointConfig {
  return BREAKPOINT_CONFIGS[breakpoint] ?? BREAKPOINT_CONFIGS['sm'];
}

export function isBreakpointActive(
  breakpoint: keyof GridBreakpoints,
  viewportWidth: number
): boolean {
  const threshold = BREAKPOINTS[breakpoint];
  return viewportWidth >= threshold;
}

export function generateGridAreaStyle(
  span: GridSpan,
  start?: number,
  end?: number,
  maxColumns: number = 12
): string {
  if (span === 'full') {
    return `grid-column: 1 / -1;`;
  }
  if (span === 'auto') {
    return `grid-column: auto;`;
  }
  if (start !== undefined) {
    return `grid-column: ${start} / span ${span};`;
  }
  if (end !== undefined) {
    return `grid-column: ${end - span} / ${end};`;
  }
  const clampedSpan = Math.max(1, Math.min(span, maxColumns));
  return `grid-column: span ${clampedSpan};`;
}

export function generateResponsiveHideStyles(
  hideBelow?: keyof GridBreakpoints,
  showBelow?: keyof GridBreakpoints
): string {
  let styles = '';

  if (hideBelow) {
    const threshold = BREAKPOINTS[hideBelow];
    styles += `@media (max-width: ${threshold - 1}px) { display: none; } `;
  }

  if (showBelow) {
    const threshold = BREAKPOINTS[showBelow];
    styles += `@media (min-width: ${threshold}px) { display: none; } `;
  }

  return styles;
}
```

### 4.5 Tailwind CSS Grid Plugin

```typescript
// tailwind.config.js plugin
function gridPlugin({ addUtilities, theme }) {
  const gridStyles = {
    '.grid-12': {
      display: 'grid',
      gridTemplateColumns: 'repeat(12, minmax(0, 1fr))',
      gap: theme('spacing.4'),
    },
    '.grid-8': {
      display: 'grid',
      gridTemplateColumns: 'repeat(8, minmax(0, 1fr))',
      gap: theme('spacing.4'),
    },
    '.grid-4': {
      display: 'grid',
      gridTemplateColumns: 'repeat(4, minmax(0, 1fr))',
      gap: theme('spacing.3'),
    },
    '.grid-auto': {
      display: 'grid',
      gridTemplateColumns: 'repeat(auto-fit, minmax(min(100%, 300px), 1fr))',
      gap: theme('spacing.4'),
    },
  };

  addUtilities(gridStyles, ['responsive', 'hover']);
}
```
