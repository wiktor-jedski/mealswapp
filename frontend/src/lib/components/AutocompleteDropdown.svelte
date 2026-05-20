<script lang="ts">
  import type { RankedAutocomplete } from '../api/types';
  import { highlightParts } from '../search/autocompleteState';

  interface Props {
    id: string;
    query: string;
    options: RankedAutocomplete[];
    selectedIndex: number;
    isOpen: boolean;
    isLoading?: boolean;
    errorMessage?: string;
    onSelect?: (index: number) => void;
    onHover?: (index: number) => void;
  }

  let { id, query, options, selectedIndex, isOpen, isLoading = false, errorMessage, onSelect, onHover }: Props = $props();
</script>

{#if isOpen}
  <div
    id={id}
    class="mt-2 max-h-72 overflow-auto rounded border border-secondary bg-surface shadow-sm"
    role="listbox"
    aria-busy={isLoading}
  >
    {#if errorMessage}
      <p class="px-3 py-2 text-sm text-error">{errorMessage}</p>
    {:else if isLoading}
      <p class="px-3 py-2 font-mono text-sm text-text-muted">Loading</p>
    {:else if options.length === 0}
      <p class="px-3 py-2 text-sm text-text-muted">No suggestions</p>
    {:else}
      {#each options as option, index}
        <button
          id={`${id}-option-${index}`}
          class="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm"
          class:bg-secondary={selectedIndex === index}
          class:text-primary={selectedIndex === index}
          role="option"
          aria-selected={selectedIndex === index}
          type="button"
          onmouseenter={() => onHover?.(index)}
          onmousedown={(event) => {
            event.preventDefault();
            onSelect?.(index);
          }}
        >
          <span>
            {#each highlightParts(option.label, query) as part}
              {#if part.highlighted}
                <mark class="bg-accent px-0.5 text-text-primary">{part.text}</mark>
              {:else}
                {part.text}
              {/if}
            {/each}
          </span>
          <span class="font-mono text-xs text-text-muted">#{option.rank}</span>
        </button>
      {/each}
    {/if}
  </div>
{/if}
