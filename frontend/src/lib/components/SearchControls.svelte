<script lang="ts">
  import type { SearchFilterKind, SearchMode, SubstitutionUnit } from "../api/generated";
  import { addSearchFilter, addSubstitutionInput, removeSearchFilter, removeSubstitutionInput, type SearchStateStore } from "../search/search-state";
  import type { RankedAutocomplete } from "../api/generated";
  import AutocompleteDropdown from "./AutocompleteDropdown.svelte";

  // Implements DESIGN-001 SearchView mode and substitution controls.
  export let state: SearchStateStore;
  export let loadAutocomplete: (query: string) => Promise<RankedAutocomplete[]>;
  export let onSearch: () => void;

  const modes: { value: SearchMode; label: string }[] = [
    { value: "catalog", label: "Catalog" },
    { value: "substitution", label: "Substitution" },
    { value: "daily_diet_alternative", label: "Daily Diet Alternative" }
  ];
  let foodObjectId = "";
  let quantity = 100;
  let unit: SubstitutionUnit = "g";
  let filterId = "";
  let filterKind: SearchFilterKind = "food_category";
  let filterInclude = true;

  function addInput() {
    const id = foodObjectId.trim();
    if (!id || quantity <= 0) return;
    state.update((current) => addSubstitutionInput(current, { foodObjectId: id, quantity, unit }));
    foodObjectId = "";
    quantity = 100;
  }

  function selectSuggestion(item: RankedAutocomplete) {
    foodObjectId = item.itemId;
    addInput();
  }

  function addFilter() {
    if (!filterId.trim()) return;
    state.update((current) => addSearchFilter(current, { filterId: filterId.trim(), kind: filterKind, include: filterInclude }));
    filterId = "";
  }
</script>

<!-- Implements DESIGN-001 SearchView documented mode, search, and Substitution Input order. -->
<section class="grid gap-4" aria-label="Search controls">
  <div class="flex flex-wrap gap-2" role="group" aria-label="Search mode">
    {#each modes as mode}
      <button type="button" aria-pressed={$state.mode === mode.value} on:click={() => state.setMode(mode.value)} class="rounded border border-[var(--color-border)] px-3 py-2 focus-visible:outline focus-visible:outline-2 focus-visible:outline-[var(--color-primary)]">{mode.label}</button>
    {/each}
  </div>

  <label class="grid gap-1 font-medium" for="search-query">
    Food search
    <input id="search-query" bind:value={$state.query} on:input={() => state.update((current) => ({ ...current, page: 1 }))} class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2" placeholder="Search foods" />
  </label>
  <form class="flex flex-wrap gap-2" aria-label="Search filters" on:submit|preventDefault={addFilter}>
    <label>Filter ID <input aria-label="Filter ID" bind:value={filterId} class="rounded border border-[var(--color-border)] px-2 py-1" /></label>
    <label>Filter kind <select aria-label="Filter kind" bind:value={filterKind}><option value="food_category">Food Category</option><option value="culinary_role">Culinary Role</option><option value="physical_state">Physical State</option><option value="allergen">Allergen</option><option value="dietary_preset">Dietary Preset</option></select></label>
    <label>Filter mode <select aria-label="Filter mode" bind:value={filterInclude}><option value={true}>Include</option><option value={false}>Exclude</option></select></label>
    <button type="submit">Add filter</button>
    <ul class="w-full">{#each $state.filters as filter}<li>{filter.kind}: {filter.filterId} ({filter.include ? "include" : "exclude"}) <button type="button" aria-label={`Remove filter ${filter.filterId}`} on:click={() => state.update((current) => removeSearchFilter(current, filter))}>Remove</button></li>{/each}</ul>
  </form>

  {#if $state.mode === "substitution"}
    <form class="grid gap-3 rounded border border-[var(--color-border)] p-4" on:submit|preventDefault={addInput}>
      <h3 class="font-semibold">Substitution inputs</h3>
      <AutocompleteDropdown loadSuggestions={loadAutocomplete} onSelect={selectSuggestion} />
      <label>Selected food ID <input aria-label="Selected food ID" bind:value={foodObjectId} class="rounded border border-[var(--color-border)] px-2 py-1" /></label>
      <label>Quantity <input aria-label="Quantity" type="number" min="0.01" step="any" bind:value={quantity} class="rounded border border-[var(--color-border)] px-2 py-1" /></label>
      <label>Unit <select aria-label="Unit" bind:value={unit} class="rounded border border-[var(--color-border)] px-2 py-1"><option value="g">g</option><option value="ml">ml</option><option value="oz">oz</option><option value="fl_oz">fl oz</option></select></label>
      <button type="submit" class="rounded bg-[var(--color-primary)] px-3 py-2 text-[var(--color-on-primary)]">Add substitution input</button>
      <ul aria-label="Selected substitution inputs">
        {#each $state.substitutionInputs as input (input.foodObjectId)}
          <li class="flex items-center justify-between gap-2"><span>{input.foodObjectId}: {input.quantity} {input.unit}</span><button type="button" aria-label={`Remove ${input.foodObjectId}`} on:click={() => state.update((current) => removeSubstitutionInput(current, input.foodObjectId))}>Remove</button></li>
        {/each}
      </ul>
    </form>
  {:else if $state.mode === "daily_diet_alternative"}
    <label class="grid gap-1" for="daily-diet-id">Daily diet ID <input id="daily-diet-id" bind:value={$state.dailyDietId} class="rounded border border-[var(--color-border)] px-3 py-2" /></label>
    <p role="status">Daily Diet Alternative currently returns a structured availability response until Phase 07.</p>
  {/if}
  <button type="button" on:click={onSearch} class="rounded bg-[var(--color-primary)] px-4 py-2 font-semibold text-[var(--color-on-primary)]">Search</button>
</section>
