# [ARCH-016] - Theme & Style Module

**Description:** Client-side theming system implementing the Style Guide specifications for consistent visual presentation across light and dark modes.

| Attribute | Value |
| :--- | :--- |
| **Type** | Module |
| **Static Aspects** | ThemeProvider, ColorPalette, TypographySystem, LayoutGrid, ComponentStyles |
| **Dependencies** | ARCH-001 (Web Application) |
| **Traceability** | SW-REQ-014, SW-REQ-015, SW-REQ-085, SW-REQ-089 |

**Dynamic Behavior:**

- **Theme Detection:** Reads system prefers-color-scheme on load. User preference overrides system setting.
- **Variable Switching:** Updates CSS custom properties for all color tokens when theme changes.
- **Responsive Layout:** 12-column grid collapses to single column below 640px breakpoint.
- **Accessibility Enforcement:** Validates all color combinations meet WCAG 2.1 AA 4.5:1 contrast ratio.

**Interface Definition:**

- `Input`: Theme preference (system or user), viewport dimensions
- `Output`: CSS custom property values, responsive layout classes

**Color Tokens (Light Mode):**

| Token | Value | Usage |
| :--- | :--- | :--- |
| --bg-primary | #F7FCF7 | Main background |
| --bg-surface | #FFFFFF | Cards, containers |
| --color-primary | #166534 | Buttons, headers |
| --color-secondary | #DCFCE7 | Badges, highlights |
| --color-accent | #F97316 | Special offers |
| --color-error | #DC2626 | Validation errors |
| --text-primary | #111827 | Body text |
| --text-muted | #6B7280 | Secondary labels |

**Color Tokens (Dark Mode):**

| Token | Value | Usage |
| :--- | :--- | :--- |
| --bg-primary | #0A0F0A | Main background |
| --bg-surface | #161D16 | Cards, containers |
| --color-primary | #4ADE80 | Buttons, active states |
| --color-secondary | #86EFAC | Secondary actions |
| --color-accent | #FFB86C | Best match badges |
| --color-error | #F87171 | Alerts |
| --text-primary | #F3F4F6 | Body text |
| --text-muted | #9CA3AF | Descriptions |

**Alternative Analysis (BP6):**

- *Chosen Approach:* CSS Custom Properties with theme provider context
- *Alternative Considered:* CSS-in-JS with runtime theme switching (Styled Components, Emotion)
- *Trade-off:* CSS Custom Properties provide zero-runtime theme switching with native browser support. CSS-in-JS adds JavaScript bundle size and runtime overhead. For the defined color palette (SW-REQ-089), native CSS variables are simpler and more performant.
