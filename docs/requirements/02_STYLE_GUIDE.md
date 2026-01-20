#### 1. Visual Identity (Light Mode)

| Role | Hex Code | Usage |
| :--- | :--- | :--- |
| **Background** | `#F7FCF7` | **Mint White** - Very subtle green-tinted white for a "fresh" feel. |
| **Surface** | `#FFFFFF` | **Pure White** - For cards, containers, and elevated sections. |
| **Primary** | `#166534` | **Forest Green** - Strong, trustworthy green for buttons and headers. |
| **Secondary** | `#DCFCE7` | **Soft Mint** - For pill badges, secondary buttons, and row highlights. |
| **Accent** | `#F97316` | **Fresh Orange** - High-contrast color for "Special Offers" or "Macro Match." |
| **Error** | `#DC2626` | **Alert Red** - For validation errors and Stripe payment failures. |
| **Text-Primary** | `#111827` | **Deep Slate** - High-readability black for all body and header text. |
| **Text-Muted** | `#6B7280` | **Steel Gray** - For secondary labels, timestamps, and placeholder text. |

#### 2. Visual Identity (Dark Mode)

| Role | Hex Code | Usage |
| :--- | :--- | :--- |
| **Background** | `#0A0F0A` | **Deep Obsidian Green** - The primary app background. |
| **Surface** | `#161D16` | **Elevated Green-Gray** - For cards, search bars, and sections. |
| **Primary** | `#4ADE80` | **Vibrant Mint** - High-visibility green for buttons and active states. |
| **Secondary** | `#86EFAC` | **Soft Sage** - Secondary actions and borders. |
| **Accent** | `#FFB86C` | **Soft Amber** - For Diet Search "Best Match" badges and high-energy CTAs. |
| **Error** | `#F87171` | **Muted Red** - For alerts and Stripe payment issues. |
| **Text-Primary** | `#F3F4F6` | **Off-White** - Primary headers and body text (90% opacity). |
| **Text-Muted** | `#9CA3AF` | **Cool Gray** - For descriptions, labels, and metadata. |

#### 3. Typography
| Element | Font Family | Size / Weight |
| :--- | :--- | :--- |
| **Headings** | Inter / Sans-Serif | Bold (700) |
| **Body** | Inter / Sans-Serif | Regular (400) |
| **Data/Labels** | Roboto Mono | Medium (500) |

#### 4. Global Layout Patterns
*This defines the "Skeleton" for the Architect (SWE.2).*

- **Grid System:** 12-column CSS Grid with a max-width of `1280px`.
- **Breakpoints:**
    - Mobile: `< 640px` (Single column, sidebar as a toggle).
    - Desktop: `> 1024px` (Sidebar as a toggle, visible by default, main content area).
- **Navigation:** search in the middle, left-sidebar for additional info as per requirements.

#### 5. Component Standards
- **Buttons:** 4px border-radius, `transition: all 0.2s ease`.
- **Inputs:** White background with 1px border (`#E0E0E0`). Focus state must use Primary color.
- **Loading States:** Use "Skeleton Screens" rather than spinning icons for Diet Search results.

#### 6. Compliance & Accessibility
- **Contrast:** All Text/Background combinations must pass **WCAG 2.1 AA** (minimum 4.5:1 ratio).
- **Responsive:** No horizontal scrolling allowed at any viewport size above 320px.

