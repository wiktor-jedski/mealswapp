<script lang="ts">
  import { onDestroy, tick } from "svelte";
  import type { RankedAutocomplete } from "../api/generated";
  import { AutocompleteController, AUTOCOMPLETE_DEBOUNCE_MS } from "./autocomplete-controller";

  // Implements DESIGN-001 AutocompleteDropdown ranked suggestion display, keyboard focus movement, selection, and dismissal.

  let {
    query,
    onSelect,
    onSubmit = () => {},
    onQueryInput = () => {},
    placeholder = "Search foods, meals, or ingredients…",
    focusKey = 0,
    searching = false
  }: {
    /**
     * Current query text. The parent owns typing and feeds debounced updates through this prop;
     * the dropdown reacts to changes and schedules a 150ms-debounced autocomplete fetch.
     */
    query: string;
    /** Called when the user selects a suggestion via Enter or option click. */
    onSelect: (item: RankedAutocomplete) => void;
    /** Called when the user commits the typed query without selecting a suggestion. */
    onSubmit?: (query: string) => void;
    /**
     * Optional input-event forwarder so a wired parent can capture typing into the search store.
     * Defaults to a no-op so the component stays self-contained before Task 151 wires it.
     */
    onQueryInput?: (value: string) => void;
    /** Mode-specific guidance shown before the user enters a query. */
    placeholder?: string;
    /** Changes when the parent wants the combobox to receive focus, e.g. initial load or mode switch. */
    focusKey?: string | number;
    /** True while an explicit submitted search request is fetching results. */
    searching?: boolean;
  } = $props();

  /** Stable id linking the combobox input to its listbox via `aria-controls`. */
  const listboxId = "autocomplete-listbox";

  let items = $state<RankedAutocomplete[]>([]);
  let isOpen = $state(false);
  let activeIndex = $state(-1);
  let listboxEl = $state<HTMLUListElement | undefined>(undefined);
  let inputEl = $state<HTMLInputElement | undefined>(undefined);
  let lastFocusedKey: string | number | null = null;
  let suppressedSelectedQuery: string | null = null;

  const controller = new AutocompleteController({
    delayMs: AUTOCOMPLETE_DEBOUNCE_MS,
    onResults: (next) => {
      items = next;
      isOpen = next.length > 0;
      activeIndex = -1;
    },
    onError: () => {
      items = [];
      isOpen = false;
      activeIndex = -1;
    }
  });

  onDestroy(() => controller.dispose());

  // Implements DESIGN-001 AutocompleteDropdown 150ms-debounced fetch driven by user-typed query prop changes.
  $effect(() => {
    if (suppressedSelectedQuery === query) {
      suppressedSelectedQuery = null;
      controller.cancel();
    } else {
      controller.input(query);
    }
  });

  // Implements DESIGN-001 SearchView search-bar focus on initial load and mode changes.
  $effect(() => {
    void focusSearchInput(focusKey, inputEl);
  });

  async function focusSearchInput(nextFocusKey: string | number, element: HTMLInputElement | undefined): Promise<void> {
    if (!element || nextFocusKey === lastFocusedKey) {
      return;
    }
    lastFocusedKey = nextFocusKey;
    await tick();
    element.focus();
  }

  /** Forwards typing to the parent so the search store can update the `query` prop. */
  function onInput(event: Event): void {
    suppressedSelectedQuery = null;
    onQueryInput((event.currentTarget as HTMLInputElement).value);
  }

  /** Hides the listbox and resets the active option without affecting the query. */
  function dismiss(): void {
    isOpen = false;
    activeIndex = -1;
  }

  /** Selects the active option, notifies the parent, and returns focus to the combobox input. */
  function selectActive(): void {
    if (!isOpen) {
      return;
    }
    const item = items[activeIndex];
    dismiss();
    if (item) {
      suppressedSelectedQuery = item.label;
      controller.cancel();
      onSelect(item);
    }
    inputEl?.focus();
  }

  /** Moves the active option by `direction` (1 forward, -1 backward) with wrap-around. */
  function moveActive(direction: 1 | -1, focusOption = true): void {
    if (!isOpen || items.length === 0) {
      return;
    }
    activeIndex = activeIndex < 0
      ? direction === 1 ? 0 : items.length - 1
      : (activeIndex + direction + items.length) % items.length;
    const option = listboxEl?.children.item(activeIndex) as HTMLElement | null;
    if (focusOption) {
      option?.focus();
    }
  }

  function onInputKeydown(event: KeyboardEvent): void {
    switch (event.key) {
      case "Tab":
        if (!isOpen || items.length === 0) {
          return;
        }
        event.preventDefault();
        moveActive(event.shiftKey ? -1 : 1);
        break;
      case "ArrowDown":
        if (isOpen && items.length > 0) {
          event.preventDefault();
          moveActive(1, false);
        }
        break;
      case "ArrowUp":
        if (isOpen && items.length > 0) {
          event.preventDefault();
          moveActive(-1, false);
        }
        break;
      case "Enter":
        if (isOpen && activeIndex >= 0) {
          event.preventDefault();
          selectActive();
        } else if (query.trim().length > 0) {
          event.preventDefault();
          dismiss();
          suppressedSelectedQuery = query;
          controller.cancel();
          onSubmit(query);
        }
        break;
      case "Escape":
        if (isOpen) {
          event.preventDefault();
          dismiss();
          inputEl?.focus();
        }
        break;
      default:
        break;
    }
  }

  function onOptionKeydown(event: KeyboardEvent, index: number): void {
    if (event.key === "Tab") {
      event.preventDefault();
      activeIndex = index;
      moveActive(event.shiftKey ? -1 : 1);
    } else if (event.key === "ArrowDown") {
      event.preventDefault();
      activeIndex = index;
      moveActive(1);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      activeIndex = index;
      moveActive(-1);
    } else if (event.key === "Enter") {
      event.preventDefault();
      activeIndex = index;
      selectActive();
    } else if (event.key === "Escape") {
      event.preventDefault();
      dismiss();
      inputEl?.focus();
    }
  }

  function onOptionClick(index: number): void {
    activeIndex = index;
    selectActive();
  }
</script>

<!-- Implements DESIGN-001 AutocompleteDropdown -->
<div class="relative grid gap-1">
  <label class="sr-only" for="autocomplete-input">Food search</label>
  <input
    id="autocomplete-input"
    bind:this={inputEl}
    type="text"
    class="truncate rounded border border-[var(--color-border)] bg-transparent px-3 py-2 pr-10 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
    role="combobox"
    aria-expanded={isOpen}
    aria-controls={listboxId}
    aria-autocomplete="list"
    aria-activedescendant={isOpen && activeIndex >= 0 ? `${listboxId}-option-${activeIndex}` : undefined}
    {placeholder}
    value={query}
    oninput={onInput}
    onkeydown={onInputKeydown}
  />

  {#if searching}
    <div
      class="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 rounded-full border-2 border-[var(--color-border)] border-t-[var(--color-primary)] motion-safe:animate-spin"
      role="status"
      aria-label="Searching"
      data-search-spinner
    >
      <span class="sr-only">Searching</span>
    </div>
  {/if}

  {#if isOpen && items.length > 0}
    <ul
      id={listboxId}
      bind:this={listboxEl}
      class="absolute left-0 top-full z-20 mt-1 grid w-full gap-0 m-0 list-none rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-0 shadow-lg"
      role="listbox"
      aria-label="Autocomplete suggestions"
    >
      {#each items as item, index (item.itemId)}
        <li
          id={`${listboxId}-option-${index}`}
          class="cursor-pointer px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
          role="option"
          aria-selected={index === activeIndex}
          tabindex={index === activeIndex ? 0 : -1}
          class:bg-[var(--color-muted)]={index === activeIndex}
          onclick={() => onOptionClick(index)}
          onkeydown={(event) => onOptionKeydown(event, index)}
        >
          {item.label}
        </li>
      {/each}
    </ul>
  {/if}
</div>
