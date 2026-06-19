<script lang="ts">
  import { onDestroy } from "svelte";
  import type { RankedAutocomplete } from "../api/generated";
  import { AutocompleteController, AUTOCOMPLETE_DEBOUNCE_MS } from "./autocomplete-controller";

  // Implements DESIGN-001 AutocompleteDropdown ranked suggestion display, keyboard focus movement, selection, and dismissal.

  /**
   * Current query text. The parent owns typing and feeds debounced updates through this prop;
   * the dropdown reacts to changes and schedules a 150ms-debounced autocomplete fetch.
   */
  export let query: string;

  /** Called when the user selects a suggestion via Enter or option click. */
  export let onSelect: (item: RankedAutocomplete) => void;

  /**
   * Optional input-event forwarder so a wired parent can capture typing into the search store.
   * Defaults to a no-op so the component stays self-contained before Task 151 wires it.
   */
  export let onQueryInput: (value: string) => void = () => {};

  /** Stable id linking the combobox input to its listbox via `aria-controls`. */
  const listboxId = "autocomplete-listbox";

  let items: RankedAutocomplete[] = [];
  let isOpen = false;
  let activeIndex = -1;
  let listboxEl: HTMLUListElement | undefined;
  let inputEl: HTMLInputElement | undefined;

  const controller = new AutocompleteController({
    delayMs: AUTOCOMPLETE_DEBOUNCE_MS,
    onResults: (next) => {
      items = next;
      isOpen = next.length > 0;
      activeIndex = next.length > 0 ? 0 : -1;
    },
    onError: () => {
      items = [];
      isOpen = false;
      activeIndex = -1;
    }
  });

  onDestroy(() => controller.dispose());

  // Implements DESIGN-001 AutocompleteDropdown 150ms-debounced fetch driven by query prop changes.
  $: if (query !== undefined) controller.input(query);

  /** Forwards typing to the parent so the search store can update the `query` prop. */
  function onInput(event: Event): void {
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
      onSelect(item);
    }
    inputEl?.focus();
  }

  /** Moves the active option by `direction` (1 forward, -1 backward) with wrap-around and focus. */
  function moveActive(direction: 1 | -1): void {
    if (!isOpen || items.length === 0) {
      return;
    }
    activeIndex = (activeIndex + direction + items.length) % items.length;
    const option = listboxEl?.children.item(activeIndex) as HTMLElement | null;
    option?.focus();
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
      case "Enter":
        if (isOpen && activeIndex >= 0) {
          event.preventDefault();
          selectActive();
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

  function onOptionClick(item: RankedAutocomplete, index: number): void {
    activeIndex = index;
    selectActive();
  }
</script>

<!-- Implements DESIGN-001 AutocompleteDropdown -->
<div class="grid gap-1">
  <label class="sr-only" for="autocomplete-input">Food search</label>
  <input
    id="autocomplete-input"
    bind:this={inputEl}
    type="text"
    class="rounded border border-[var(--color-border)] bg-transparent px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
    role="combobox"
    aria-expanded={isOpen}
    aria-controls={listboxId}
    aria-autocomplete="list"
    aria-activedescendant={isOpen && activeIndex >= 0 ? `${listboxId}-option-${activeIndex}` : undefined}
    value={query}
    on:input={onInput}
    on:keydown={onInputKeydown}
  />

  {#if isOpen && items.length > 0}
    <ul
      id={listboxId}
      bind:this={listboxEl}
      class="grid gap-0 m-0 list-none rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-0"
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
          on:click={() => onOptionClick(item, index)}
          on:keydown={(event) => onOptionKeydown(event, index)}
        >
          {item.label}
        </li>
      {/each}
    </ul>
  {/if}
</div>
