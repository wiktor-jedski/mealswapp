<script lang="ts">
  import type { AppError, FoodObject, SearchRejection, SearchResponse, SimilarityMetadata } from "../api/generated";

  // Implements DESIGN-001 ResultsGrid generated result rendering contract.
  export let response: SearchResponse | null = null;
  export let loading = false;
  export let error: AppError | null = null;
  export let rejection: SearchRejection | null = null;
  export let onPage: (page: number) => void;
  export let onRetry: () => void;

  const pageSize = 10;
  $: items = response?.items.slice(0, pageSize) ?? [];
  $: totalPages = Math.max(1, Math.ceil((response?.totalCount ?? 0) / pageSize));

  function similarityFor(item: FoodObject): SimilarityMetadata | undefined {
    return response?.similarityMetadata.find((metadata) => metadata.itemId === item.id);
  }

  function placeholder(item: FoodObject): string {
    const name = item.primaryFoodCategory?.name.toLocaleLowerCase() ?? "";
    if (name.includes("fruit")) return "/assets/placeholders/fruit.svg";
    if (name.includes("vegetable")) return "/assets/placeholders/vegetable.svg";
    return "/assets/placeholders/default.svg";
  }

  function imageFailed(event: Event, item: FoodObject) {
    const image = event.currentTarget as HTMLImageElement;
    image.onerror = null;
    image.src = placeholder(item);
  }
</script>

<!-- Implements DESIGN-001 ResultsGrid stable cards, states, and maximum-10 pagination. -->
<section aria-labelledby="results-title" aria-busy={loading} class="grid gap-4">
  <h3 id="results-title" class="text-lg font-semibold">Search results</h3>
  {#if error}
    <div role="alert" class="rounded border border-[var(--color-error)] p-4">
      {#if rejection}<p>{rejection.message}</p><p class="font-data text-sm">Code: {rejection.code}{#if rejection.field} · Field: {rejection.field}{/if}</p>{:else}{error.message}{/if}
      {#if error.retryable}<button type="button" on:click={onRetry} class="ml-3">Retry</button>{/if}
    </div>
  {:else if loading && !response}
    <div aria-label="Loading search results" class="grid gap-3 sm:grid-cols-2">{#each Array(4) as _}<div class="h-56 animate-pulse rounded bg-[var(--color-border)]"></div>{/each}</div>
  {:else if response && items.length === 0}
    <p>No foods matched your search.</p>
  {:else if response}
    {#if loading}<p role="status">Loading page {response.page}…</p>{/if}
    <div class="grid gap-4 sm:grid-cols-2" data-testid="results-grid">
      {#each items as item (item.id)}
        {@const similarity = similarityFor(item)}
        <article class="grid h-[27rem] grid-rows-[10rem_1fr] overflow-hidden rounded border border-[var(--color-border)] bg-[var(--color-surface)]" data-testid="result-card">
          <img src={item.imageUrl || placeholder(item)} alt="" on:error={(event) => imageFailed(event, item)} class="h-40 w-full object-cover" />
          <div class="grid gap-2 p-4">
            <h4 class="font-semibold">{item.name}</h4>
            <p>{item.classifications.filter((value) => value.kind === "food_category").map((value) => value.name).join(", ") || "Uncategorized"}</p>
            <dl class="grid grid-cols-2 gap-1 font-data text-sm">
              <dt>Protein</dt><dd>{item.macros.protein}g / {item.macros.basis}</dd>
              <dt>Carbohydrate</dt><dd>{item.macros.carbohydrate}g / {item.macros.basis}</dd>
              <dt>Fat</dt><dd>{item.macros.fat}g / {item.macros.basis}</dd>
              <dt>Calories</dt><dd>{item.calories}</dd>
            </dl>
            {#if similarity}<p>Similarity {Math.round(similarity.score * 100)}% · {similarity.tier}</p>{/if}
          </div>
        </article>
      {/each}
    </div>
    <nav aria-label="Results pages" class="flex items-center justify-between">
      <button type="button" disabled={response.page <= 1 || loading} on:click={() => onPage(response!.page - 1)}>Previous</button>
      <span>Page {response.page} of {totalPages}</span>
      <button type="button" disabled={response.page >= totalPages || loading} on:click={() => onPage(response!.page + 1)}>Next</button>
    </nav>
  {/if}
</section>
