<script lang="ts">
  import { onMount } from "svelte";
  import { setQuery, setMode } from "../stores/search";
  import {
    sidebarStore,
    toggleCollapsed,
    toggleMobileOpen,
    setMobileOpen,
    initSidebar
  } from "../stores/sidebar";
  import { resolvedTheme, setThemePreference } from "../stores/theme";
  import { preferencesStore, setUnitSystem } from "../stores/preferences";
  import type {
    ProfileData,
    ProfileEnvelope,
    SavedItem,
    SavedItemsEnvelope,
    SearchHistoryEntry,
    SearchHistoryEnvelope,
    SearchMode
  } from "../api/generated";
  import type { UnitSystem } from "../stores/preferences";

  // Implements DESIGN-001 SidebarComponent navigation, history, favorites, units, and responsive collapse.

  /** Authenticated profile endpoint used to detect signed-in state without exposing tokens. */
  const PROFILE_ENDPOINT = "/api/v1/profile";

  /** Authenticated search-history list endpoint served by ARCH-008 SearchHistoryRepository. */
  const SEARCH_HISTORY_ENDPOINT = "/api/v1/search-history";

  /** Authenticated saved-items list endpoint filtered to favorites, served by ARCH-008 SavedDataRepository. */
  const SAVED_ITEMS_FAVORITES_ENDPOINT = "/api/v1/saved-items?kind=favorite";

  /** Account-level unit options rendered as a compact sidebar preference row. */
  const unitSystems: { value: UnitSystem; label: string }[] = [
    { value: "metric", label: "Metric" },
    { value: "imperial", label: "Imperial" }
  ];

  /** Authenticated profile loaded from `/api/v1/profile`; `null` while loading or anonymous. */
  let profile = $state<ProfileData | null>(null);
  /** True while the profile probe is in flight; gates the anonymous guidance block. */
  let authenticating = $state(true);
  /** True when the profile probe returned a valid profile envelope. */
  let authenticated = $state(false);
  /** Inline auth-probe error message; never propagated to the parent so core search stays usable. */
  let authError = $state<string | null>(null);

  /** Authenticated search-history entries loaded from `/api/v1/search-history`. */
  let history = $state<SearchHistoryEntry[]>([]);
  let historyLoading = $state(false);
  /** Inline history error message; never propagated to the parent so core search stays usable. */
  let historyError = $state<string | null>(null);

  /** Authenticated favorite saved items loaded from `/api/v1/saved-items?kind=favorite`. */
  let favorites = $state<SavedItem[]>([]);
  let favoritesLoading = $state(false);
  /** Inline favorites error message; never propagated to the parent so core search stays usable. */
  let favoritesError = $state<string | null>(null);

  onMount(() => {
    initSidebar();
    void loadSidebar();
  });

  /** Type guard ensuring a stored history `mode` is one of the supported SearchMode values before calling setMode. */
  function isSearchMode(value: string): value is SearchMode {
    return value === "catalog" || value === "substitution" || value === "daily_diet_alternative";
  }

  /**
   * Probes `/api/v1/profile` to detect the signed-in state. A 401 means anonymous (no error);
   * any other failure sets {@link authError} and treats the user as anonymous so public
   * Catalog Search stays usable. When authenticated, kicks off history and favorites loads.
   */
  async function loadSidebar(): Promise<void> {
    authenticating = true;
    authError = null;
    try {
      const response = await fetch(PROFILE_ENDPOINT, {
        method: "GET",
        credentials: "include",
        headers: { Accept: "application/json" }
      });
      if (response.status === 401) {
        authenticated = false;
        profile = null;
        authenticating = false;
        return;
      }
      if (!response.ok) {
        authenticated = false;
        profile = null;
        authError = "Couldn't load your activity right now.";
        authenticating = false;
        return;
      }
      const envelope = (await response.json()) as ProfileEnvelope;
      profile = envelope.data ?? null;
      authenticated = profile !== null;
      authenticating = false;
    } catch {
      authenticated = false;
      profile = null;
      authError = "Couldn't load your activity right now.";
      authenticating = false;
      return;
    }
    if (authenticated) {
      void loadHistory();
      void loadFavorites();
    }
  }

  /**
   * Loads authenticated search history from `/api/v1/search-history`. Non-2xx and network
   * failures set {@link historyError} inline; the error never propagates to the parent so
   * core search stays usable.
   */
  async function loadHistory(): Promise<void> {
    historyLoading = true;
    historyError = null;
    try {
      const response = await fetch(SEARCH_HISTORY_ENDPOINT, {
        method: "GET",
        credentials: "include",
        headers: { Accept: "application/json" }
      });
      if (!response.ok) {
        historyError = "Couldn't load history.";
        historyLoading = false;
        return;
      }
      const envelope = (await response.json()) as SearchHistoryEnvelope;
      history = envelope.data?.history ?? [];
    } catch {
      historyError = "Couldn't load history.";
    }
    historyLoading = false;
  }

  /**
   * Loads authenticated favorites from `/api/v1/saved-items?kind=favorite`. Non-2xx and
   * network failures set {@link favoritesError} inline; the error never propagates to the
   * parent so core search stays usable.
   */
  async function loadFavorites(): Promise<void> {
    favoritesLoading = true;
    favoritesError = null;
    try {
      const response = await fetch(SAVED_ITEMS_FAVORITES_ENDPOINT, {
        method: "GET",
        credentials: "include",
        headers: { Accept: "application/json" }
      });
      if (!response.ok) {
        favoritesError = "Couldn't load favorites.";
        favoritesLoading = false;
        return;
      }
      const envelope = (await response.json()) as SavedItemsEnvelope;
      favorites = envelope.data?.items ?? [];
    } catch {
      favoritesError = "Couldn't load favorites.";
    }
    favoritesLoading = false;
  }

  /**
   * Restores search state from a selected history entry by calling `setQuery` with the
   * entry's query and `setMode` with the entry's mode when the stored mode is one of the
   * supported SearchMode values. Closes the mobile sidebar so focus returns to the results.
   *
   * @remarks Implements DESIGN-001 SidebarComponent selecting a history entry restores search state.
   */
  function onHistoryEntrySelect(entry: SearchHistoryEntry): void {
    setQuery(entry.query);
    if (isSearchMode(entry.mode)) {
      setMode(entry.mode);
    }
    setMobileOpen(false);
  }

  /**
   * Converts the current resolved theme into an explicit binary light/dark preference.
   * The default stored `system` preference keeps following OS changes until this button is used.
   *
   * @remarks Implements DESIGN-016 ThemeProvider binary sidebar theme switch.
   */
  function onThemeToggle(): void {
    setThemePreference($resolvedTheme === "dark" ? "light" : "dark");
  }

  /** Branding shown in the sidebar header; falls back to the product name when the profile has no display name. */
  let branding = $derived(profile?.displayName && profile.displayName.length > 0 ? profile.displayName : "Mealswapp");
</script>

<!-- Implements DESIGN-001 SidebarComponent -->
<aside
  class="desktop-sidebar-left flex flex-col gap-2 overflow-hidden border-b border-[var(--color-border)] bg-[var(--color-surface)] p-3 transition-[width,padding-right] duration-200 ease-out motion-reduce:transition-none sm:sticky sm:top-0 sm:h-screen sm:border-b-0 sm:border-r {$sidebarStore.collapsed ? 'sm:w-14' : 'sm:w-60 sm:pr-5'}"
  aria-label="Activity sidebar"
  data-sidebar
  data-collapsed={$sidebarStore.collapsed}
  data-mobile-open={$sidebarStore.mobileOpen}
>
  <!-- Implements DESIGN-001 SidebarComponent mobile toggle: visible only on small screens when the sidebar is closed. -->
  {#if !$sidebarStore.mobileOpen}
    <button
      type="button"
      class="mobile-sidebar-toggle flex h-10 w-10 self-center items-center justify-center rounded border border-[var(--color-border)] text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] sm:hidden"
      aria-label="Open activity sidebar"
      aria-expanded={false}
      aria-controls="activity-sidebar-content"
      onclick={toggleMobileOpen}
      data-sidebar-mobile-toggle
    >
      <span aria-hidden="true">☰</span>
    </button>
  {/if}

  <!-- Implements DESIGN-001 SidebarComponent desktop collapse/expand toggle: visible only on sm+ screens. -->
  <button
    type="button"
    class="sidebar-collapse-toggle hidden self-end rounded border border-[var(--color-border)] px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] sm:block"
    aria-label={$sidebarStore.collapsed ? "Expand sidebar" : "Collapse sidebar"}
    aria-expanded={!$sidebarStore.collapsed}
    onclick={toggleCollapsed}
    data-sidebar-collapse
  >
    {$sidebarStore.collapsed ? "»" : "«"}
  </button>

  <!-- Implements DESIGN-001 SidebarComponent content: visible on mobile only when mobileOpen, on desktop only when not collapsed. -->
  <div
    id="activity-sidebar-content"
    class="sidebar-animated-content grid gap-4 {$sidebarStore.mobileOpen ? 'block' : 'hidden'} sm:grid"
    data-sidebar-content
  >
    <!-- Implements DESIGN-001 SidebarComponent mobile close button: visible only on small screens when the sidebar is open. -->
    <button
      type="button"
      class="sidebar-mobile-close self-end rounded border border-[var(--color-border)] px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] sm:hidden"
      aria-label="Close activity sidebar"
      aria-expanded={true}
      aria-controls="activity-sidebar-content"
      onclick={() => setMobileOpen(false)}
      data-sidebar-mobile-close
    >
      ✕
    </button>

    <h1 class="text-2xl font-semibold">{branding}</h1>

    <!-- Implements DESIGN-016 ThemeProvider binary light/dark switch placed under the sidebar brand. -->
    <div class="flex items-center justify-between gap-3 rounded-full border border-[var(--color-border)] bg-[var(--color-bg)] p-1.5 shadow-sm" data-sidebar-theme-toggle>
      <span class="sr-only">Current theme: {$resolvedTheme}</span>
      <span class="flex h-8 w-8 items-center justify-center rounded-full text-[var(--color-accent)]" aria-hidden="true">
        <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none">
          <circle cx="12" cy="12" r="4" stroke="currentColor" stroke-width="2" />
          <path d="M12 2v3M12 19v3M4.93 4.93l2.12 2.12M16.95 16.95l2.12 2.12M2 12h3M19 12h3M4.93 19.07l2.12-2.12M16.95 7.05l2.12-2.12" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
        </svg>
      </span>
      <button
        type="button"
        class="relative h-8 w-14 rounded-full border border-[var(--color-border)] bg-[var(--color-surface)] transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
        aria-label="Theme preference"
        aria-pressed={$resolvedTheme === "dark"}
        onclick={onThemeToggle}
      >
        <span
          class="absolute top-1/2 h-6 w-6 -translate-y-1/2 rounded-full bg-[var(--color-primary)] shadow transition-[left]"
          style:left={$resolvedTheme === "dark" ? "calc(100% - 1.75rem)" : "0.25rem"}
          aria-hidden="true"
        ></span>
      </button>
      <span class="flex h-8 w-8 items-center justify-center rounded-full text-[var(--color-primary)]" aria-hidden="true">
        <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none">
          <path d="M20.5 15.5A8.5 8.5 0 0 1 8.5 3.5 7 7 0 1 0 20.5 15.5Z" stroke="currentColor" stroke-width="2" stroke-linejoin="round" />
        </svg>
      </span>
    </div>

    <!-- Implements DESIGN-001 SidebarComponent account-level unit preference control. -->
    <div class="flex items-center gap-2" data-sidebar-units>
      <label class="text-sm text-[var(--color-muted)]" for="sidebar-unit-system">Units:</label>
      <select
        id="sidebar-unit-system"
        class="min-w-0 flex-1 rounded border border-[var(--color-border)] bg-transparent px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
        value={$preferencesStore.unitSystem}
        onchange={(event) => setUnitSystem((event.currentTarget as HTMLSelectElement).value as UnitSystem)}
      >
        {#each unitSystems as unit (unit.value)}
          <option value={unit.value}>{unit.label}</option>
        {/each}
      </select>
    </div>

    {#if !authenticating}
      {#if !authenticated}
        <!-- Implements DESIGN-001 SidebarComponent anonymous empty/sign-in guidance. -->
        <p class="text-sm" data-sidebar-anonymous>
          Sign in to see your history and favorites.
        </p>
      {:else}
        <!-- Implements DESIGN-001 SidebarComponent authenticated search history list loaded from generated Phase 03 contracts. -->
        <section class="grid gap-2" aria-label="Search history" data-sidebar-history>
          <h2 class="font-data text-xs uppercase text-[var(--color-muted)]">History</h2>
          {#if historyError}
            <p class="text-sm text-[var(--color-muted)]" data-sidebar-history-error>{historyError}</p>
          {:else if !historyLoading && history.length === 0}
            <p class="text-sm text-[var(--color-muted)]">No recent searches.</p>
          {:else if !historyLoading}
            <ul class="grid gap-1">
              {#each history as entry}
                <li>
                  <button
                    type="button"
                    class="w-full truncate rounded px-2 py-1 text-left text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
                    data-sidebar-history-entry={entry.id}
                    onclick={() => onHistoryEntrySelect(entry)}
                  >
                    {entry.query}
                  </button>
                </li>
              {/each}
            </ul>
          {/if}
        </section>

        <!-- Implements DESIGN-001 SidebarComponent authenticated favorites list loaded from generated Phase 03 contracts. -->
        <section class="grid gap-2" aria-label="Favorites" data-sidebar-favorites>
          <h2 class="font-data text-xs uppercase text-[var(--color-muted)]">Favorites</h2>
          {#if favoritesError}
            <p class="text-sm text-[var(--color-muted)]" data-sidebar-favorites-error>{favoritesError}</p>
          {:else if !favoritesLoading && favorites.length === 0}
            <p class="text-sm text-[var(--color-muted)]">No favorites yet.</p>
          {:else if !favoritesLoading}
            <ul class="grid gap-1">
              {#each favorites as favorite}
                <li class="truncate px-2 py-1 text-sm" data-sidebar-favorite={favorite.itemId}>
                  {favorite.itemId}
                </li>
              {/each}
            </ul>
          {/if}
        </section>

        {#if authError}
          <p class="text-sm text-[var(--color-muted)]" data-sidebar-auth-error>{authError}</p>
        {/if}
      {/if}
    {/if}
  </div>
</aside>
