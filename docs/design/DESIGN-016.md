## FILE: DESIGN-016.md
**Traceability:** ARCH-016

**Static aspects covered:** ThemeProvider, ColorPalette, TypographySystem, LayoutGrid, ComponentStyles.

### 0. Static Aspect Responsibilities
- `ThemeProvider`: owns theme preference resolution and CSS variable application.
- `ColorPalette`: owns light and dark mode token values and contrast validation inputs.
- `TypographySystem`: owns Inter UI text, Roboto Mono data labels, and sizing conventions.
- `LayoutGrid`: owns responsive 12-column to single-column layout rules.
- `ComponentStyles`: owns Tailwind utility conventions and reusable component styling tokens.

### 1. Data Structures & Types
- `type ThemePreference = "system" | "light" | "dark"`
- `type ResolvedTheme = "light" | "dark"`
- `interface ThemeState { preference: ThemePreference; resolved: ResolvedTheme; systemTheme: ResolvedTheme }`
- `interface ColorPalette { bgPrimary: string; bgSurface: string; colorPrimary: string; colorSecondary: string; colorAccent: string; colorError: string; textPrimary: string; textMuted: string }`
- `interface TypographySystem { uiFont: "Inter"; dataFont: "Roboto Mono"; baseSizePx: number; labelSizePx: number }`
- `interface LayoutGrid { columns: 12 | 1; gapPx: number; breakpointPx: 640 }`
- `interface ContrastCheck { foreground: string; background: string; ratio: number; passesAA: boolean }`

### 2. Logic & Algorithms (Step-by-Step)
1. On client startup, read theme preference from ARCH-001 settings.
2. Resolve `system` by reading `prefers-color-scheme`; otherwise use the explicit user choice.
3. Apply light or dark palette values to CSS custom properties on the document root.
4. Subscribe to system theme changes and recompute only when preference is `system`.
5. Use Tailwind utility classes backed by CSS variables for component styling.
6. Use 12-column layout above 640px and single-column layout below 640px.
7. Before adding or changing color pairs, run contrast validation and reject combinations below WCAG AA 4.5:1 for normal text.
8. Persist user theme selection through ARCH-001 `LocalStorageManager`.

### 3. State Management & Error Handling
- `system_light`: system preference resolves to light.
- `system_dark`: system preference resolves to dark.
- `user_light`: explicit light mode overrides system.
- `user_dark`: explicit dark mode overrides system.
- `invalid_preference`: fallback to `system`.
- `contrast_failure`: block token update during design/test validation.
- `storage_unavailable`: use in-memory theme state for current session.

### 4. Component Interfaces
- `function resolveTheme(preference: ThemePreference, systemTheme: ResolvedTheme): ResolvedTheme`
- `function applyTheme(theme: ResolvedTheme): void`
- `function setThemePreference(preference: ThemePreference): void`
- `function getPalette(theme: ResolvedTheme): ColorPalette`
- `function getLayoutGrid(widthPx: number): LayoutGrid`
- `function checkContrast(foreground: string, background: string): ContrastCheck`
- `function subscribeToSystemTheme(callback: (theme: ResolvedTheme) => void): () => void`
