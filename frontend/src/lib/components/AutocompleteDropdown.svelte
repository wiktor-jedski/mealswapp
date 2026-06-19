<script lang="ts">
  import { onDestroy } from "svelte";
  import type { RankedAutocomplete } from "../api/generated";
  import { createDebouncer } from "../search/debounce";

  // Implements DESIGN-001 AutocompleteDropdown server-ranked interaction contract.
  export let loadSuggestions: (query: string) => Promise<RankedAutocomplete[]>;
  export let onSelect: (item: RankedAutocomplete) => void;
  export let label = "Find substitution food";

  let query = "";
  let suggestions: RankedAutocomplete[] = [];
  let open = false;
  let activeIndex = -1;
  const listboxId = `autocomplete-${Math.random().toString(36).slice(2)}`;
  const debouncer = createDebouncer(150, load);

  function load(value: string) {
    const requested = value.trim();
    if (!requested) { suggestions = []; open = false; return; }
    loadSuggestions(requested).then((items) => {
      if (query.trim() !== requested) return;
      suggestions = items;
      activeIndex = items.length > 0 ? 0 : -1;
      open = items.length > 0;
    }).catch(() => { suggestions = []; open = false; });
  }

  function inputChanged() {
    activeIndex = -1;
    debouncer.schedule(query);
  }

  function select(item: RankedAutocomplete) {
    query = item.label;
    open = false;
    suggestions = [];
    activeIndex = -1;
    onSelect(item);
  }

  function keydown(event: KeyboardEvent) {
    if (event.key === "ArrowDown" && suggestions.length > 0) {
      event.preventDefault();
      activeIndex = (activeIndex + 1) % suggestions.length;
    } else if (event.key === "ArrowUp" && suggestions.length > 0) {
      event.preventDefault();
      activeIndex = (activeIndex - 1 + suggestions.length) % suggestions.length;
    } else if (event.key === "Enter" && open && activeIndex >= 0) {
      event.preventDefault();
      select(suggestions[activeIndex]);
    } else if (event.key === "Escape") {
      open = false;
      activeIndex = -1;
    }
  }

  onDestroy(() => debouncer.cancel());
</script>

<!-- Implements DESIGN-001 AutocompleteDropdown expanding-in-flow accessible combobox. -->
<div class="grid gap-1">
  <label for={`${listboxId}-input`}>{label}</label>
  <input id={`${listboxId}-input`} role="combobox" aria-autocomplete="list" aria-expanded={open} aria-controls={listboxId} aria-activedescendant={activeIndex >= 0 ? `${listboxId}-${activeIndex}` : undefined} bind:value={query} on:input={inputChanged} on:keydown={keydown} class="rounded border border-[var(--color-border)] px-3 py-2" />
  {#if open}
    <div id={listboxId} role="listbox" aria-label="Autocomplete suggestions" class="grid rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-1">
      {#each suggestions as suggestion, index (suggestion.itemId)}
        <button id={`${listboxId}-${index}`} type="button" role="option" aria-selected={index === activeIndex} on:mouseenter={() => activeIndex = index} on:click={() => select(suggestion)} class="rounded px-3 py-2 text-left focus-visible:outline focus-visible:outline-2 focus-visible:outline-[var(--color-primary)]">{suggestion.label}</button>
      {/each}
    </div>
  {/if}
</div>
