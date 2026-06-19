<script lang="ts">
  import { onMount } from "svelte";
  import { searchStore, setQuery, setMode } from "../stores/search";
  import {
    sidebarStore,
    toggleCollapsed,
    toggleMobileOpen,
    setMobileOpen,
    initSidebar
  } from "../stores/sidebar";
  import SettingsPanel from "./SettingsPanel.svelte";
  import type {
    ProfileData,
    ProfileEnvelope,
    SavedItem,
    SavedItemsEnvelope,
    SearchHistoryEntry,
    SearchHistoryEnvelope,
    SearchMode
  } from "../api/generated";

  // Implements DESIGN-001 SidebarComponent navigation, history, favorites, settings, and responsive collapse.

  /** Authenticated profile endpoint used to detect signed-in state without exposing tokens. */
  const PROFILE_ENDPOINT = "/api/v1/profile";

  /** Authenticated search-history list endpoint served by ARCH-008 SearchHistoryRepository. */
  const SEARCH_HISTORY_ENDPOINT = "/api/v1/search-history";

  /** Authenticated saved-items list endpoint filtered to favorites, served by ARCH-008 SavedDataRepository. */
  const SAVED_ITEMS_FAVORITES_ENDPOINT = "/api/v1/saved-items?kind=favorite";

  /**
   * Mode options rendered in the sidebar for Catalog, Substitution, and Daily Diet
   * Alternative navigation. Selecting one calls `setMode` via {@link onModeSelect}.
   */
  const modeOptions: { value: SearchMode; id: string; label: string }[] = [
    { value: "catalog", id: "sidebar-mode-catalog", label: "Catalog" },
    { value: "substitution", id: "sidebar-mode-substitution", label: "Substitution" },
    { value: "daily_diet_alternative", id: "sidebar-mode-daily-diet", label: "Daily Diet Alternative" }
  ];

  /** Authenticated profile loaded from `/api/v1/profile`; `null` while loading or anonymous. */
  let profile: ProfileData | null = null;
  /** True while the profile probe is in flight; gates the anonymous guidance block. */
  let authenticating = true;
  /** True when the profile probe returned a valid profile envelope. */
  let authenticated = false;
  /** Inline auth-probe error message; never propagated to the parent so core search stays usable. */
  let authError: string | null = null;

  /** Authenticated search-history entries loaded from `/api/v1/search-history`. */
  let history: SearchHistoryEntry[] = [];
  let historyLoading = false;
  /** Inline history error message; never propagated to the parent so core search stays usable. */
  let historyError: string | null = null;

  /** Authenticated favorite saved items loaded from `/api/v1/saved-items?kind=favorite`. */
  let favorites: SavedItem[] = [];
  let favoritesLoading = false;
  /** Inline favorites error message; never propagated to the parent so core search stays usable. */
  let favoritesError: string | null = null;

  /** Local settings-panel visibility flag, toggled by the Settings entry point. */
  let settingsOpen = false;

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

  /** Switches the active search mode and closes the mobile sidebar so the results area is reachable. */
  function onModeSelect(mode: SearchMode): void {
    setMode(mode);
    setMobileOpen(false);
  }

  /** Toggles the local settings panel visibility without navigating away from search. */
  function onSettingsToggle(): void {
    settingsOpen = !settingsOpen;
  }

  /** Branding shown in the sidebar header; falls back to the product name when the profile has no display name. */
  $: branding = profile?.displayName && profile.displayName.length > 0 ? profile.displayName : "Mealswapp";
</script>

<!-- Implements DESIGN-001 SidebarComponent -->
<aside
  class="desktop-sidebar-left flex flex-col gap-2 border-b border-[var(--color-border)] bg-[var(--color-surface)] p-3 sm:sticky sm:top-0 sm:h-screen sm:border-b-0 sm:border-r sm:pr-5 {$sidebarStore.collapsed ? 'sm:w-14 sm:p-2' : 'sm:w-60'}"
  aria-label="Activity sidebar"
  data-sidebar
  data-collapsed={$sidebarStore.collapsed}
  data-mobile-open={$sidebarStore.mobileOpen}
>
  <!-- Implements DESIGN-001 SidebarComponent mobile toggle: visible only on small screens when the sidebar is closed. -->
  {#if !$sidebarStore.mobileOpen}
    <button
      type="button"
      class="mobile-sidebar-toggle rounded border border-[var(--color-border)] px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] sm:hidden"
      aria-label="Open activity sidebar"
      aria-expanded={false}
      aria-controls="activity-sidebar-content"
      on:click={toggleMobileOpen}
      data-sidebar-mobile-toggle
    >
      ☰ Activity
    </button>
  {/if}

  <!-- Implements DESIGN-001 SidebarComponent desktop collapse/expand toggle: visible only on sm+ screens. -->
  <button
    type="button"
    class="sidebar-collapse-toggle hidden self-end rounded border border-[var(--color-border)] px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] sm:block"
    aria-label={$sidebarStore.collapsed ? "Expand sidebar" : "Collapse sidebar"}
    aria-expanded={!$sidebarStore.collapsed}
    on:click={toggleCollapsed}
    data-sidebar-collapse
  >
    {$sidebarStore.collapsed ? "»" : "«"}
  </button>

  <!-- Implements DESIGN-001 SidebarComponent content: visible on mobile only when mobileOpen, on desktop only when not collapsed. -->
  <div
    id="activity-sidebar-content"
    class="grid gap-4 {$sidebarStore.mobileOpen ? 'block' : 'hidden'} {$sidebarStore.collapsed ? 'sm:hidden' : 'sm:grid'}"
    data-sidebar-content
  >
    <!-- Implements DESIGN-001 SidebarComponent mobile close button: visible only on small screens when the sidebar is open. -->
    <button
      type="button"
      class="sidebar-mobile-close self-end rounded border border-[var(--color-border)] px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] sm:hidden"
      aria-label="Close activity sidebar"
      aria-expanded={true}
      aria-controls="activity-sidebar-content"
      on:click={() => setMobileOpen(false)}
      data-sidebar-mobile-close
    >
      ✕
    </button>

    <h1 class="text-2xl font-semibold">{branding}</h1>

    <!-- Implements DESIGN-001 SidebarComponent search-mode navigation. -->
    <nav class="flex flex-col gap-1" aria-label="Search mode navigation" data-sidebar-modes>
      {#each modeOptions as option}
        <button
          id={option.id}
          type="button"
          class="rounded px-2 py-1 text-left text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
          class:bg-[var(--color-primary)]={$searchStore.mode === option.value}
          class:text-white={$searchStore.mode === option.value}
          aria-pressed={$searchStore.mode === option.value}
          on:click={() => onModeSelect(option.value)}
        >
          {option.label}
        </button>
      {/each}
    </nav>

    {#if authenticating}
      <p class="text-sm text-[var(--color-muted)]" data-sidebar-loading>Loading activity…</p>
    {:else if !authenticated}
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
        {:else if historyLoading}
          <p class="text-sm text-[var(--color-muted)]">Loading…</p>
        {:else if history.length === 0}
          <p class="text-sm text-[var(--color-muted)]">No recent searches.</p>
        {:else}
          <ul class="grid gap-1">
            {#each history as entry}
              <li>
                <button
                  type="button"
                  class="w-full truncate rounded px-2 py-1 text-left text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
                  data-sidebar-history-entry={entry.id}
                  on:click={() => onHistoryEntrySelect(entry)}
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
        {:else if favoritesLoading}
          <p class="text-sm text-[var(--color-muted)]">Loading…</p>
        {:else if favorites.length === 0}
          <p class="text-sm text-[var(--color-muted)]">No favorites yet.</p>
        {:else}
          <ul class="grid gap-1">
            {#each favorites as favorite}
              <li class="truncate px-2 py-1 text-sm" data-sidebar-favorite={favorite.itemId}>
                {favorite.itemId}
              </li>
            {/each}
          </ul>
        {/if}
      </section>

      <!-- Implements DESIGN-001 SidebarComponent settings entry point. -->
      <button
        type="button"
        class="self-start rounded border border-[var(--color-border)] px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
        aria-expanded={settingsOpen}
        on:click={onSettingsToggle}
        data-sidebar-settings
      >
        Settings
      </button>
      {#if settingsOpen}
        <SettingsPanel />
      {/if}

      {#if authError}
        <p class="text-sm text-[var(--color-muted)]" data-sidebar-auth-error>{authError}</p>
      {/if}
    {/if}
  </div>
</aside>
