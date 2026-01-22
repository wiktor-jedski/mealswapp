# [ARCH-001] - Web Application Module

**Description:** The responsive single-page application (SPA) built with Svelte that serves as the primary user interface, handling all client-side rendering, state management with Svelte stores + TanStack Query, local caching via Service Worker + localStorage, and offline functionality.

| Attribute | Value |
| :--- | :--- |
| **Type** | Module |
| **Static Aspects** | SearchView, SidebarComponent, ResultsGrid, AutocompleteDropdown, ThemeProvider, OfflineBanner, SettingsPanel, LocalStorageManager, ServiceWorker |
| **Dependencies** | ARCH-010 (API Gateway), ARCH-011 (Caching Layer), TanStack Query |
| **Traceability** | SW-REQ-001, SW-REQ-002, SW-REQ-003, SW-REQ-005, SW-REQ-007, SW-REQ-008, SW-REQ-009, SW-REQ-011, SW-REQ-012, SW-REQ-013, SW-REQ-014, SW-REQ-015, SW-REQ-018, SW-REQ-025, SW-REQ-048, SW-REQ-077, SW-REQ-085, SW-REQ-086, SW-REQ-087, SW-REQ-088, SW-REQ-089 |

**Dynamic Behavior:**

- **Initialization:** On application load, initializes search mode to 'Single Item' and enables all macronutrient toggles. Detects system theme preference and applies user-stored preference override.
- **Search Input:** Debounces user input by 150ms before triggering API calls. Manages focus states for keyboard navigation (Tab/Shift+Tab).
- **Offline Detection:** Monitors browser online/offline events. Switches to cached data display and shows offline indicator when disconnected.
- **Theme Switching:** Real-time CSS variable updates when user toggles light/dark mode. Persists selection to localStorage.

**Interface Definition:**

- `Input`: User interactions (keyboard, mouse, touch), system events (online/offline), API responses (JSON)
- `Output`: HTTP requests to API Gateway, localStorage writes, DOM updates

**Alternative Analysis (BP6):**

- *Chosen Approach:* Single-Page Application with client-side routing and state management
- *Alternative Considered:* Server-Side Rendering (SSR) with hydration
- *Trade-off:* SPA provides better offline capability (SW-REQ-087, SW-REQ-088) and reduces server load. SSR would improve initial load SEO but adds server complexity and breaks offline-first design. Since Mealswapp is an authenticated app (not SEO-critical), SPA is superior.

**Reference Documentation:** 
- 02_APPENDIX_A.md
