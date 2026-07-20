<script lang="ts">
  import { searchStore, setMode } from "../stores/search";
  import type { SearchMode } from "../api/generated";

  // Implements DESIGN-001 SearchView mode controls (Catalog, Substitution, Daily Diet, Daily Diet Alternative).

  interface Props {
    onModeChange?: (mode: SearchMode) => void;
  }

  let { onModeChange = setMode }: Props = $props();

  /**
   * Mode options rendered above the search bar. Selecting one calls `setMode`, which resets
   * incompatible state and pagination through the search store.
   */
  const modeOptions: { value: SearchMode; id: string; label: string; description: string }[] = [
    {
      value: "catalog",
      id: "search-mode-catalog",
      label: "Catalog",
      description: "Find foods, meals, or ingredients by name."
    },
    {
      value: "substitution",
      id: "search-mode-substitution",
      label: "Substitution",
      description: "Find alternatives for a food using quantity and unit context."
    },
    {
      value: "daily_diet",
      id: "search-mode-daily-diet",
      label: "Daily Diet",
      description: "Search across saved daily diets."
    },
    {
      value: "daily_diet_alternative",
      id: "search-mode-daily-diet-alternative",
      label: "Daily Diet Alternative",
      description: "Search for replacements within a saved daily diet."
    }
  ];

  /** Plain active-mode explanation shown below the centered mode buttons. */
  let activeDescription = $derived(
    modeOptions.find((option) => option.value === $searchStore.mode)?.description ?? ""
  );
</script>

<!-- Implements DESIGN-001 SearchView mode controls positioned above the search bar and macro controls. -->
<nav class="grid justify-items-center gap-3 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-3 text-center" aria-label="Search modes">
  <div class="flex flex-wrap justify-center gap-2">
    {#each modeOptions as option}
      <button
        id={option.id}
        type="button"
        class="rounded border px-3 py-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
        class:border-[var(--color-primary)]={$searchStore.mode === option.value}
        aria-pressed={$searchStore.mode === option.value}
        onclick={() => onModeChange(option.value)}
      >
        {option.label}
      </button>
    {/each}
  </div>

  <p class="max-w-xl text-sm text-[var(--color-muted)]" data-search-mode-description>
    {activeDescription}
  </p>
</nav>
