<script lang="ts">
  import type { FoodObject, SimilarityMetadata } from "../api/generated";
  import ResultCard from "./ResultCard.svelte";

  // Implements DESIGN-001 ResultsGrid container: stable card layout, pagination controls, image fallback, and similarity badges.

  /** Maximum number of result cards rendered per page (deterministic Phase 04 page size of 10). */
  const PAGE_SIZE = 10;

  /** Number of skeleton placeholders shown while the first page loads with no previous results. */
  const SKELETON_COUNT = 3;

  /** Search result items for the current page; the grid never renders more than PAGE_SIZE cards. */
  export let results: FoodObject[] = [];

  /** Similarity metadata rows matched to result items by `itemId`. */
  export let similarityMetadata: SimilarityMetadata[] = [];

  /** Parallel similarity score array used as a fallback when metadata is absent for an item. */
  export let similarityScores: number[] = [];

  /** True while a search request is in flight; previous results stay visible when present. */
  export let loading: boolean = false;

  /** User-facing error message; when non-null the grid renders the error state instead of results. */
  export let error: string | null = null;

  /** Total result count used to derive pagination boundaries. */
  export let totalCount: number = 0;

  /** Current one-based page index. */
  export let page: number = 1;

  /** Called with the next page index when the user clicks Previous or Next. */
  export let onPageChange: (page: number) => void = () => {};

  /** Cards rendered for the current page, capped at PAGE_SIZE so no page renders more than 10 items. */
  $: pagedResults = results.slice(0, PAGE_SIZE);

  /** Total page count derived from totalCount and the fixed page size, with a minimum of one page. */
  $: totalPages = Math.max(1, Math.ceil(totalCount / PAGE_SIZE));

  /** Previous button is disabled on page one. */
  $: hasPrev = page > 1;

  /** Next button is disabled on the last page. */
  $: hasNext = page < totalPages;

  /** Lookup of similarity metadata by item id for per-card matching. */
  $: similarityByItemId = new Map(similarityMetadata.map((meta) => [meta.itemId, meta]));

  /** Returns the similarity metadata row matching `itemId`, or null when no row exists. */
  function findSimilarity(itemId: string): SimilarityMetadata | null {
    return similarityByItemId.get(itemId) ?? null;
  }
</script>

<!-- Implements DESIGN-001 ResultsGrid -->
<section
  class="grid gap-4 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4"
  aria-label="Search results"
  data-results-grid
>
  {#if error}
    <div class="text-sm text-[var(--color-accent)]" role="alert" data-results-error>{error}</div>
  {:else if loading && pagedResults.length === 0}
    <ul class="grid gap-3" data-results-skeletons>
      {#each Array.from({ length: SKELETON_COUNT }) as _, index (index)}
        <li
          class="h-32 animate-pulse rounded border border-[var(--color-border)] bg-[var(--color-muted)]"
          data-result-skeleton
        ></li>
      {/each}
    </ul>
  {:else if pagedResults.length === 0}
    <p class="text-sm text-[var(--color-muted)]" data-results-empty>No results found.</p>
  {:else}
    {#if loading}
      <div class="text-sm text-[var(--color-muted)]" data-results-loading-overlay aria-live="polite">
        Loading…
      </div>
    {/if}

    <ul class="grid gap-3" data-results-list>
      {#each pagedResults as item, index (item.id)}
        <li>
          <ResultCard
            item={item}
            similarity={findSimilarity(item.id)}
            fallbackScore={similarityScores[index] ?? null}
          />
        </li>
      {/each}
    </ul>

    <nav class="flex items-center gap-3" aria-label="Results pagination" data-results-pagination>
      <button
        type="button"
        class="rounded border border-[var(--color-border)] px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
        on:click={() => onPageChange(page - 1)}
        disabled={!hasPrev}
        data-results-prev
      >
        Previous
      </button>
      <span class="font-data text-xs" data-results-page>Page {page} of {totalPages}</span>
      <button
        type="button"
        class="rounded border border-[var(--color-border)] px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
        on:click={() => onPageChange(page + 1)}
        disabled={!hasNext}
        data-results-next
      >
        Next
      </button>
    </nav>
  {/if}
</section>
