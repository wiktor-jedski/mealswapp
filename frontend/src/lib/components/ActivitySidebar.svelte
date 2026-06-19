<script lang="ts">
  import { onMount } from "svelte";
  import type { SearchHistoryEntry } from "../api/generated";
  import type { ActivityData } from "../api/activity-client";
  import type { SearchMode } from "../api/generated";
  import type { SearchStateStore } from "../search/search-state";
  import { setThemePreference, themePreference, type ThemePreference } from "../stores/theme";

  // Implements DESIGN-001 SidebarComponent activity and navigation dependencies.
  export let searchState: SearchStateStore;
  export let loadActivity: () => Promise<ActivityData>;

  let collapsed = false;
  let mobileOpen = false;
  let activity: ActivityData | null = null;
  let failed = false;
  const modes: { value: SearchMode; label: string }[] = [{ value: "catalog", label: "Catalog" }, { value: "substitution", label: "Substitution" }, { value: "daily_diet_alternative", label: "Daily Diet" }];

  onMount(async () => {
    try { activity = await loadActivity(); } catch { failed = true; }
  });

  function restore(entry: SearchHistoryEntry) {
    const mode = modes.some((candidate) => candidate.value === entry.mode) ? entry.mode as SearchMode : "catalog";
    searchState.update((state) => ({ ...state, mode, query: entry.query, page: 1, substitutionInputs: [], dailyDietId: undefined }));
  }
</script>

<!-- Implements DESIGN-001 SidebarComponent desktop-left, collapse, mobile toggle, history, favorites, and settings entry. -->
<div class="min-w-0 sm:col-span-3 sm:min-h-screen">
  <button type="button" class="mb-3 sm:hidden" aria-expanded={mobileOpen} aria-controls="activity-sidebar" on:click={() => mobileOpen = !mobileOpen}>Activity</button>
  <aside id="activity-sidebar" aria-label="Activity sidebar" class:hidden={!mobileOpen} class:collapsed class="w-full border-b border-[var(--color-border)] pb-4 sm:block sm:w-56 sm:border-b-0 sm:border-r sm:pr-4" data-collapsed={collapsed}>
    <div class="flex items-center justify-between gap-2"><h1 class:hidden={collapsed} class="text-2xl font-semibold">Mealswapp</h1><button type="button" aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"} on:click={() => collapsed = !collapsed}>{collapsed ? "»" : "«"}</button></div>
    {#if !collapsed}
      <nav class="mt-5 grid gap-2" aria-label="Sidebar search modes">{#each modes as mode}<button type="button" on:click={() => searchState.setMode(mode.value)}>{mode.label}</button>{/each}</nav>
      <section class="mt-5" aria-labelledby="history-title"><h2 id="history-title" class="font-semibold">History</h2>
        {#if failed}<p>Activity unavailable. Search remains available.</p>
        {:else if activity && !activity.authenticated}<p>Sign in to view history and favorites.</p>
        {:else if activity}<ul>{#each activity.history as entry}<li><button type="button" on:click={() => restore(entry)}>{entry.query}</button></li>{/each}</ul>
        {:else}<p>Loading activity…</p>{/if}
      </section>
      {#if activity?.authenticated}<section class="mt-5" aria-labelledby="favorites-title"><h2 id="favorites-title" class="font-semibold">Favorites</h2><ul>{#each activity.favorites as item}<li class="break-all">{item.itemId}</li>{/each}</ul></section>{/if}
      <a class="mt-5 block" href="#search-settings-title">Settings</a>
      <label class="mt-4 grid gap-1" for="sidebar-theme">Theme preference<select id="sidebar-theme" value={$themePreference} on:change={(event) => setThemePreference(event.currentTarget.value as ThemePreference)}><option value="system">System</option><option value="light">Light</option><option value="dark">Dark</option></select></label>
    {/if}
  </aside>
</div>
