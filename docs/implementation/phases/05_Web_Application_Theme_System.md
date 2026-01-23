## Phase 5: Web Application & Theme System

**Goal:** Build the Svelte frontend with responsive UI

### Components & Static Aspects

#### ARCH-001 - Web Application Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **SearchView** | Main search input with mode toggles | `components/SearchView.svelte` |
| **SidebarComponent** | Collapsible sidebar with history/favorites | `components/SidebarComponent.svelte` |
| **ResultsGrid** | Display search results with images, macros, similarity | `components/ResultsGrid.svelte` |
| **AutocompleteDropdown** | Dropdown list of suggestions with keyboard nav | `components/AutocompleteDropdown.svelte` |
| **OfflineBanner** | Visual indicator when offline | `components/OfflineBanner.svelte` |
| **SettingsPanel** | User preferences (unit, theme) | `components/SettingsPanel.svelte` |
| **LocalStorageManager** | Manage localStorage for queries/history | `api/cache.ts` |

#### ARCH-016 - Theme & Style Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **ThemeProvider** | Svelte context for theme state | `theme/ThemeProvider.svelte` |
| **ColorPalette** | CSS custom property definitions | `theme/colors.ts` |
| **TypographySystem** | Font families, sizes, weights | `theme/typography.ts` |
| **LayoutGrid** | 12-column grid, responsive breakpoints | `theme/layout.ts` |
| **ComponentStyles** | Shared component style utilities | `theme/components.ts` |

#### ARCH-017 - Error Handling Module
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **ErrorBoundary** | Svelte error boundary component | `components/ErrorBoundary.svelte` |
| **RetryManager** | Automatic retry logic on connectivity restore | `api/retry.ts` |
| **ErrorMessageMapper** | Map technical errors to user-friendly messages | `api/error_mapper.ts` |

#### ARCH-011 (partial) - Caching Layer (Client)
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **ServiceWorkerCache** | Service Worker for offline caching | `static/sw.js` |
| **LocalStorageCache** | 20 recent queries, 5 search history | `api/cache.ts` |

### Testing
- [ ] Responsive layout: desktop (12-column), mobile (single column < 640px)
- [ ] Theme switching updates CSS variables
- [ ] System theme preference detection (prefers-color-scheme)
- [ ] Search debounce delays API calls by 150ms
- [ ] Keyboard navigation (Tab/Shift+Tab) through autocomplete
- [ ] Offline banner appears when disconnected
- [ ] Service Worker caches images and API responses
- [ ] localStorage stores 20 recent queries, 5 search history (LRU)
- [ ] WCAG 2.1 AA color contrast compliance (4.5:1 ratio)
- [ ] Error boundary catches component errors without full crash

---

