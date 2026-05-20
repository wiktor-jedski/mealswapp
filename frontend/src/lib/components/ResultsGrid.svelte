<script lang="ts">
  import type { SearchResponse } from '../api/types';
  import { imageForItem, paginationState, quantityLabel, scaleMacros, similarityPercent } from '../search/resultsGrid';

  interface Props {
    response?: SearchResponse;
    status: 'idle' | 'debouncing' | 'loading' | 'success' | 'empty' | 'error';
    errorMessage?: string;
    onPageChange?: (page: number) => void;
  }

  let { response, status, errorMessage, onPageChange }: Props = $props();
  let page = $derived(response ? paginationState(response) : undefined);
</script>

<section class="mt-4 min-h-32 rounded border border-secondary bg-surface p-4" aria-live="polite">
  {#if status === 'loading'}
    <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {#each Array(3) as _}
        <div class="h-40 animate-pulse rounded border border-secondary bg-background"></div>
      {/each}
    </div>
  {:else if status === 'empty'}
    <p class="text-sm text-text-muted">No results found.</p>
  {:else if status === 'error'}
    <p class="text-sm text-error">{errorMessage ?? 'Search failed.'}</p>
  {:else if status === 'success' && response}
    <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {#each response.items as item}
        {@const displayedMacros = scaleMacros(item.macros, item.matchingQuantity ?? 100)}
        <article class="grid min-h-72 grid-rows-[auto_1fr_auto] rounded border border-secondary bg-surface p-3">
          <img
            class="aspect-[4/3] w-full rounded border border-secondary object-cover"
            src={imageForItem(item)}
            alt=""
            loading="lazy"
          />
          <div class="mt-3">
            <div class="flex items-start justify-between gap-3">
              <h2 class="text-sm font-bold">{item.name}</h2>
              {#if item.similarity}
                <span class="flex items-center gap-1 font-mono text-xs text-text-muted">
                  <img class="h-5 w-5" src={item.similarity.imageUrl} alt="" />
                  {similarityPercent(item)}
                </span>
              {/if}
            </div>
            <div class="mt-2 flex flex-wrap gap-1">
              {#each item.tags as tag}
                <span class="rounded bg-secondary px-2 py-1 font-mono text-[11px] text-primary">{tag}</span>
              {/each}
            </div>
          </div>
          <dl class="mt-3 grid grid-cols-2 gap-2 font-mono text-xs text-text-muted">
            <div>
              <dt>Protein</dt>
              <dd class="text-text-primary">{displayedMacros.protein} g</dd>
            </div>
            <div>
              <dt>Carbs</dt>
              <dd class="text-text-primary">{displayedMacros.carbs} g</dd>
            </div>
            <div>
              <dt>Fat</dt>
              <dd class="text-text-primary">{displayedMacros.fat} g</dd>
            </div>
            <div>
              <dt>Calories</dt>
              <dd class="text-text-primary">{item.calories ?? 0}</dd>
            </div>
            <div class="col-span-2">
              <dt>Quantity</dt>
              <dd class="text-text-primary">{quantityLabel(item)}</dd>
            </div>
          </dl>
        </article>
      {/each}
    </div>

    {#if page}
      <nav class="mt-4 flex items-center justify-between gap-3" aria-label="Results pages">
        <button
          class="rounded border border-secondary px-3 py-2 text-sm disabled:opacity-50"
          type="button"
          disabled={!page.canPrevious}
          onclick={() => onPageChange?.(page.page - 1)}
        >
          Previous
        </button>
        <span class="font-mono text-sm text-text-muted">Page {page.page} of {page.totalPages}</span>
        <button
          class="rounded border border-secondary px-3 py-2 text-sm disabled:opacity-50"
          type="button"
          disabled={!page.canNext}
          onclick={() => onPageChange?.(page.page + 1)}
        >
          Next
        </button>
      </nav>
    {/if}
  {:else}
    <p class="text-sm text-text-muted">Start with a food item.</p>
  {/if}
</section>
