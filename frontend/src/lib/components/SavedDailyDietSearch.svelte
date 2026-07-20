<script lang="ts">
  import type { DailyDiet } from "../api/generated";

  // Implements DESIGN-001 AutocompleteDropdown keyboard behavior for user-owned Daily Diet lookup.

  let {
    diets,
    loading = false,
    focusKey = 0,
    onSelect
  }: {
    diets: DailyDiet[];
    loading?: boolean;
    focusKey?: string | number;
    onSelect: (diet: DailyDiet) => void;
  } = $props();

  const listboxId = "saved-daily-diet-search-listbox";
  let query = $state("");
  let open = $state(false);
  let activeIndex = $state(0);
  let inputEl = $state<HTMLInputElement | undefined>(undefined);
  let lastFocusKey: string | number | null = null;
  let matches = $derived(rankDailyDiets(diets, query));

  $effect(() => {
    if (focusKey !== lastFocusKey && inputEl) {
      lastFocusKey = focusKey;
      inputEl.focus();
    }
  });

  function rankDailyDiets(collections: DailyDiet[], value: string): DailyDiet[] {
    const needle = value.trim().toLocaleLowerCase();
    if (needle === "") return [];
    return [...collections]
      .filter((diet) => diet.name.toLocaleLowerCase().includes(needle))
      .sort((left, right) => {
        const leftName = left.name.toLocaleLowerCase();
        const rightName = right.name.toLocaleLowerCase();
        const leftRank = leftName === needle ? 0 : leftName.startsWith(needle) ? 1 : 2;
        const rightRank = rightName === needle ? 0 : rightName.startsWith(needle) ? 1 : 2;
        return leftRank - rightRank || leftName.localeCompare(rightName);
      });
  }

  function onInput(event: Event): void {
    query = (event.currentTarget as HTMLInputElement).value;
    activeIndex = 0;
    open = query.trim().length > 0;
  }

  function selectDiet(diet: DailyDiet): void {
    query = diet.name;
    open = false;
    activeIndex = 0;
    onSelect(diet);
    inputEl?.focus();
  }

  function onKeydown(event: KeyboardEvent): void {
    if (event.key === "Escape") {
      open = false;
      return;
    }
    if (!open || matches.length === 0) return;
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      const direction = event.key === "ArrowDown" ? 1 : -1;
      activeIndex = (activeIndex + direction + matches.length) % matches.length;
      return;
    }
    if (event.key === "Enter") {
      event.preventDefault();
      selectDiet(matches[activeIndex] ?? matches[0]);
    }
  }
</script>

<!-- Implements DESIGN-001 AutocompleteDropdown saved Daily Diet lookup. -->
<div class="relative grid gap-1" data-saved-daily-diet-search>
  <label class="sr-only" for="saved-daily-diet-search-input">Search saved Daily Diets</label>
  <input
    id="saved-daily-diet-search-input"
    bind:this={inputEl}
    type="text"
    class="truncate rounded border border-[var(--color-border)] bg-transparent px-3 py-2 pr-10 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
    role="combobox"
    aria-expanded={open && matches.length > 0}
    aria-controls={listboxId}
    aria-autocomplete="list"
    aria-activedescendant={open && matches.length > 0 ? `${listboxId}-option-${activeIndex}` : undefined}
    placeholder="Search saved Daily Diets by name…"
    value={query}
    oninput={onInput}
    onfocus={() => (open = query.trim().length > 0)}
    onkeydown={onKeydown}
  />

  {#if loading}
    <div class="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 rounded-full border-2 border-[var(--color-border)] border-t-[var(--color-primary)] motion-safe:animate-spin" role="status" aria-label="Loading saved Daily Diets">
      <span class="sr-only">Loading saved Daily Diets</span>
    </div>
  {/if}

  {#if open && matches.length > 0}
    <ul id={listboxId} class="absolute left-0 top-full z-20 mt-1 grid w-full list-none rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-0 shadow-lg" role="listbox" aria-label="Saved Daily Diet suggestions">
      {#each matches as diet, index (diet.id)}
        <li
          id={`${listboxId}-option-${index}`}
          class="cursor-pointer px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
          class:bg-[var(--color-muted)]={index === activeIndex}
          role="option"
          aria-selected={index === activeIndex}
          tabindex="-1"
          onmousedown={(event) => event.preventDefault()}
          onclick={() => selectDiet(diet)}
          onkeydown={(event) => { if (event.key === "Enter") selectDiet(diet); }}
        >
          {diet.name}
        </li>
      {/each}
    </ul>
  {/if}
</div>
