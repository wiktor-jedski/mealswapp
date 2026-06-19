<script lang="ts">
  import { searchStore, setMode } from "../stores/search";
  import type { SearchMode } from "../api/generated";

  // Implements DESIGN-001 SearchView mode controls (Catalog, Substitution, Daily Diet Alternative).

  /**
   * Mode options rendered above the search bar and macro controls. Selecting one calls `setMode`,
   * which resets incompatible state and pagination through the search store.
   */
  const modeOptions: { value: SearchMode; id: string; label: string }[] = [
    { value: "catalog", id: "search-mode-catalog", label: "Catalog" },
    { value: "substitution", id: "search-mode-substitution", label: "Substitution" },
    { value: "daily_diet_alternative", id: "search-mode-daily-diet", label: "Daily Diet Alternative" }
  ];
</script>

<!-- Implements DESIGN-001 SearchView mode controls positioned above the search bar and macro controls. -->
<nav class="flex flex-wrap gap-2 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-3" aria-label="Search modes">
  {#each modeOptions as option}
    <button
      id={option.id}
      type="button"
      class="rounded border px-3 py-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
      class:border-[var(--color-primary)]={$searchStore.mode === option.value}
      aria-pressed={$searchStore.mode === option.value}
      on:click={() => setMode(option.value)}
    >
      {option.label}
    </button>
  {/each}
</nav>
