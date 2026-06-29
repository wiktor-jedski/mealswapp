<script lang="ts">
  import type { FoodObject, SimilarityMetadata, SourceSummary } from "../api/generated";
  import ResultCard from "./ResultCard.svelte";
  import SourceSummaryCard from "./SourceSummaryCard.svelte";

  // Implements DESIGN-001 ResultsGrid container: stable card layout, pagination controls, image fallback, and similarity badges.

  /** Maximum number of result cards rendered per page (deterministic Phase 04 page size of 10). */
  const PAGE_SIZE = 10;

  let {
    results = [],
    similarityMetadata = [],
    similarityScores = [],
    showSimilarity = true,
    sourceSummary = null,
    onAddToSubstitution = null,
    error = null,
    loading = false,
    totalCount = 0,
    page = 1,
    onPageChange = () => {}
  }: {
    /** Search result items for the current page; the grid never renders more than PAGE_SIZE cards. */
    results?: FoodObject[];
    /** Similarity metadata rows matched to result items by `itemId`. */
    similarityMetadata?: SimilarityMetadata[];
    /** Parallel similarity score array used as a fallback when metadata is absent for an item. */
    similarityScores?: number[];
    /** Whether similarity match badges should be shown for this search mode. */
    showSimilarity?: boolean;
    /** Optional substitution source totals rendered before substitution results. */
    sourceSummary?: SourceSummary | null;
    /** Optional Catalog action for adding a full result item to the Substitution Input list. */
    onAddToSubstitution?: ((item: FoodObject) => void) | null;
    /** User-facing error message; when non-null the grid renders the error state instead of results. */
    error?: string | null;
    /** True while the explicit submitted search request is in flight. */
    loading?: boolean;
    /** Total result count used to derive pagination boundaries. */
    totalCount?: number;
    /** Current one-based page index. */
    page?: number;
    /** Called with the next page index when the user clicks Previous or Next. */
    onPageChange?: (page: number) => void;
  } = $props();

  /** Cards rendered for the current page, capped at PAGE_SIZE so no page renders more than 10 items. */
  let pagedResults = $derived(results.slice(0, PAGE_SIZE));

  /** Total page count derived from totalCount and the fixed page size, with a minimum of one page. */
  let totalPages = $derived(Math.max(1, Math.ceil(totalCount / PAGE_SIZE)));

  /** Previous button is disabled on page one. */
  let hasPrev = $derived(page > 1);

  /** Next button is disabled on the last page. */
  let hasNext = $derived(page < totalPages);

  /** Lookup of similarity metadata by item id for per-card matching. */
  let similarityByItemId = $derived(new Map(similarityMetadata.map((meta) => [meta.itemId, meta])));

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
  {:else if pagedResults.length === 0 && !loading}
    <p class="text-sm text-[var(--color-muted)]" data-results-empty>No results found.</p>
  {:else}
    <ul class="grid gap-3" data-results-list>
      {#if showSimilarity && sourceSummary}
        <li>
          <SourceSummaryCard {sourceSummary} />
        </li>
      {/if}

      {#each pagedResults as item, index (item.id)}
        <li>
          <ResultCard
            item={item}
            similarity={showSimilarity ? findSimilarity(item.id) : null}
            fallbackScore={showSimilarity ? similarityScores[index] ?? null : null}
            {onAddToSubstitution}
          />
        </li>
      {/each}
    </ul>

    <nav class="flex items-center gap-3" aria-label="Results pagination" data-results-pagination>
      <button
        type="button"
        class="rounded border border-[var(--color-border)] px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
        onclick={() => onPageChange(page - 1)}
        disabled={!hasPrev}
        data-results-prev
      >
        Previous
      </button>
      <span class="font-data text-xs" data-results-page>Page {page} of {totalPages}</span>
      <button
        type="button"
        class="rounded border border-[var(--color-border)] px-3 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
        onclick={() => onPageChange(page + 1)}
        disabled={!hasNext}
        data-results-next
      >
        Next
      </button>
    </nav>
  {/if}
</section>
