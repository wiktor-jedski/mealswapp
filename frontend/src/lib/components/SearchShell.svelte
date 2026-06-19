<script lang="ts">
  import { themePreference, setThemePreference } from "../stores/theme";
  import {
    searchStore,
    setQuery,
    addSubstitutionInput,
    addFilter,
    removeFilter
  } from "../stores/search";
  import type {
    SearchFilterKind,
    SearchRejection,
    RankedAutocomplete
  } from "../api/generated";
  import SidebarComponent from "./SidebarComponent.svelte";
  import SearchModes from "./SearchModes.svelte";
  import AutocompleteDropdown from "./AutocompleteDropdown.svelte";
  import SubstitutionInputs from "./SubstitutionInputs.svelte";
  import DailyDietControls from "./DailyDietControls.svelte";
  import SettingsPanel from "./SettingsPanel.svelte";
  import SearchResults from "./SearchResults.svelte";
  import OfflineBanner from "./OfflineBanner.svelte";

  // Implements DESIGN-001 SearchView shell composition: sidebar, mode controls, autocomplete search bar, mode-specific controls, filters, settings, results, and offline status.

  /** Structured Daily Diet Alternative rejection lifted from the 422 SearchRejection envelope by SearchResults. */
  let rejection: SearchRejection | null = null;

  /** Draft state for the minimal filter composer wiring the search store filter capability. */
  let draftFilterId = "";
  let draftFilterKind: SearchFilterKind = "food_category";
  let draftFilterInclude = true;

  /** Filter kind options mirroring the generated `SearchFilterKind` union. */
  const filterKinds: SearchFilterKind[] = [
    "food_category",
    "culinary_role",
    "physical_state",
    "allergen",
    "dietary_preset"
  ];

  /**
   * Handles autocomplete selection: in Substitution mode adds a Substitution Input from the
   * suggestion's food object id; otherwise sets the query to the suggestion label so results update.
   */
  function onAutocompleteSelect(item: RankedAutocomplete): void {
    if ($searchStore.mode === "substitution") {
      addSubstitutionInput({ foodObjectId: item.itemId, quantity: 100, unit: "g" });
    } else {
      setQuery(item.label);
    }
  }

  /** Adds the draft filter to the search store, resetting the id field. Empty ids are ignored. */
  function addFilterInput(): void {
    const trimmed = draftFilterId.trim();
    if (trimmed.length === 0) {
      return;
    }
    addFilter({ filterId: trimmed, kind: draftFilterKind, include: draftFilterInclude });
    draftFilterId = "";
  }
</script>

<!-- Implements DESIGN-001 SearchView, SidebarComponent, SettingsPanel, and DESIGN-016 LayoutGrid (12-column desktop, single-column below 640px). -->
<main class="min-h-screen">
  <!-- Implements DESIGN-016 LayoutGrid: 12-column grid above 640px, single column below 640px, max-width 1280px. -->
  <section class="mx-auto grid min-h-screen max-w-7xl gap-6 px-4 py-6 sm:grid-cols-12 sm:px-6">
    <!-- Implements DESIGN-001 SidebarComponent placed in the left 3-column grid cell; Task 147 renders the full activity sidebar. -->
    <aside class="sm:col-span-3">
      <SidebarComponent />
    </aside>

    <div class="flex flex-col gap-5 sm:col-span-9">
      <header class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="font-data text-xs uppercase text-[var(--color-muted)]">Phase 05 Search</p>
          <h2 class="mt-1 text-xl font-semibold">Search modes</h2>
        </div>
        <select
          class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm"
          value={$themePreference}
          on:change={(event) => setThemePreference(event.currentTarget.value as "system" | "light" | "dark")}
          aria-label="Theme preference"
        >
          <option value="system">System</option>
          <option value="light">Light</option>
          <option value="dark">Dark</option>
        </select>
      </header>

      <!-- Visual order: mode controls → autocomplete search bar → mode-specific controls → filters → macro controls → results → offline status. -->
      <SearchModes />

      <AutocompleteDropdown
        query={$searchStore.query}
        onQueryInput={setQuery}
        onSelect={onAutocompleteSelect}
      />

      {#if $searchStore.mode === "substitution"}
        <SubstitutionInputs />
      {:else if $searchStore.mode === "daily_diet_alternative"}
        <DailyDietControls {rejection} />
      {/if}

      <!-- Implements DESIGN-001 SearchView filter composer wiring the search store filter state. -->
      <section class="grid gap-3 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4" aria-label="Search filters">
        <div class="flex flex-wrap items-center gap-2">
          <label class="sr-only" for="filter-id">Filter id</label>
          <input
            id="filter-id"
            class="rounded border border-[var(--color-border)] bg-transparent px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
            type="text"
            placeholder="Filter id"
            bind:value={draftFilterId}
            on:keydown={(event) => {
              if (event.key === "Enter") {
                event.preventDefault();
                addFilterInput();
              }
            }}
          />
          <label class="sr-only" for="filter-kind">Filter kind</label>
          <select
            id="filter-kind"
            class="rounded border border-[var(--color-border)] bg-transparent px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
            bind:value={draftFilterKind}
          >
            {#each filterKinds as kind (kind)}
              <option value={kind}>{kind}</option>
            {/each}
          </select>
          <label class="flex items-center gap-1 text-sm" for="filter-include">
            <input id="filter-include" type="checkbox" class="h-4 w-4" bind:checked={draftFilterInclude} />
            Include
          </label>
          <button
            type="button"
            class="rounded border border-[var(--color-border)] px-3 py-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
            on:click={addFilterInput}
          >
            Add filter
          </button>
        </div>

        {#if $searchStore.filters.length > 0}
          <ul class="grid gap-1" data-active-filters>
            {#each $searchStore.filters as filter (filter.filterId)}
              <li class="flex items-center justify-between rounded border border-[var(--color-border)] px-2 py-1 text-sm" data-filter-id={filter.filterId}>
                <span>{filter.filterId} ({filter.kind}, {filter.include ? "include" : "exclude"})</span>
                <button
                  type="button"
                  class="rounded border border-[var(--color-border)] px-2 py-0.5 text-xs focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
                  on:click={() => removeFilter(filter.filterId)}
                  data-filter-remove={filter.filterId}
                >
                  Remove
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </section>

      <SettingsPanel />

      <SearchResults onRejection={(r) => (rejection = r)} />

      <OfflineBanner />
    </div>
  </section>
</main>
