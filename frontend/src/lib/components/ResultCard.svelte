<script lang="ts">
  import type { FoodObject, SimilarityMetadata, SimilarityTier } from "../api/generated";

  // Implements DESIGN-001 ResultsGrid result card: image fallback, classifications, macros with basis, calories, and similarity badge.

  /** Food object rendered by this card. */
  export let item: FoodObject;

  /** Similarity metadata matched to this item by `itemId`; null when no similarity row exists (e.g. Catalog mode). */
  export let similarity: SimilarityMetadata | null = null;

  /** Optional fallback similarity score from the parallel `similarityScores` array; used only when metadata is absent. */
  export let fallbackScore: number | null = null;

  /** True when the item image failed to load and the category placeholder should replace it. */
  let imageFailed = false;

  /** Food Category classifications (kind === "food_category") rendered as visible chips. */
  $: foodCategories = item.classifications.filter(
    (classification) => classification.kind === "food_category"
  );

  /** Primary Food Category used for the image placeholder when no image is provided or the image fails to load. */
  $: placeholderCategory = item.primaryFoodCategory ?? foodCategories[0] ?? null;

  /** Initial shown inside the category placeholder tile. */
  $: placeholderInitial = placeholderCategory
    ? placeholderCategory.name.charAt(0).toUpperCase()
    : "?";

  /** Human-readable macro basis label, e.g. "per 100g" or "per 100ml". */
  $: macroBasisLabel = item.macroBasis === "100ml" ? "per 100ml" : "per 100g";

  /** Resolved similarity score: prefer metadata, fall back to the parallel scores array. */
  $: similarityScore = similarity?.score ?? fallbackScore;

  /** Resolved similarity tier; null when no metadata is available. */
  $: similarityTier = similarity?.tier ?? null;

  /** True when an <img> should be rendered: the item provides a URL and it has not failed. */
  $: showImage = Boolean(item.imageUrl) && !imageFailed;

  /** Tailwind classes and short label for each similarity tier badge. */
  const tierStyles: Record<SimilarityTier, { label: string; classes: string }> = {
    excellent: { label: "Excellent", classes: "bg-[var(--color-primary)] text-white" },
    good: { label: "Good", classes: "bg-[var(--color-primary)] text-white" },
    fair: { label: "Fair", classes: "bg-[var(--color-accent)] text-white" },
    poor: { label: "Poor", classes: "bg-[var(--color-muted)] text-white" }
  };

  /** Resets the broken-image flag whenever the item's image URL changes so a new image retries. */
  $: resetBrokenImage(item.imageUrl);

  function resetBrokenImage(_imageUrl: string | null | undefined): void {
    imageFailed = false;
  }

  /** Marks the image as failed so the category placeholder replaces it. */
  function onImageError(): void {
    imageFailed = true;
  }
</script>

<!-- Implements DESIGN-001 ResultsGrid -->
<article
  class="grid gap-3 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4"
  aria-label={item.name}
  data-result-card
  data-result-id={item.id}
>
  <div class="grid gap-3 sm:grid-cols-[96px_1fr]">
    <div
      class="grid h-24 w-24 place-items-center rounded bg-[var(--color-muted)]"
      data-result-image-wrapper
    >
      {#if showImage}
        <img
          class="h-24 w-24 rounded object-cover"
          src={item.imageUrl ?? undefined}
          alt={item.name}
          loading="lazy"
          on:error={onImageError}
          data-result-image
        />
      {:else}
        <div
          class="grid place-items-center text-center"
          role="img"
          aria-label={placeholderCategory ? placeholderCategory.name : item.name}
          data-result-placeholder
        >
          <span class="font-data text-2xl font-semibold text-white" aria-hidden="true">{placeholderInitial}</span>
          {#if placeholderCategory}
            <span class="mt-1 px-1 text-xs text-white">{placeholderCategory.name}</span>
          {/if}
        </div>
      {/if}
    </div>

    <div class="grid gap-2">
      <h3 class="text-base font-semibold" data-result-name>{item.name}</h3>

      {#if foodCategories.length > 0}
        <ul class="flex flex-wrap gap-1" data-result-categories>
          {#each foodCategories as category (category.id)}
            <li class="rounded bg-[var(--color-muted)] px-2 py-0.5 text-xs text-white">{category.name}</li>
          {/each}
        </ul>
      {/if}

      <dl class="grid gap-1 font-data text-xs" data-result-macros>
        <div class="flex gap-2">
          <dt class="text-[var(--color-muted)]">Protein</dt>
          <dd>{item.macros.protein}g</dd>
        </div>
        <div class="flex gap-2">
          <dt class="text-[var(--color-muted)]">Carbs</dt>
          <dd>{item.macros.carbohydrates}g</dd>
        </div>
        <div class="flex gap-2">
          <dt class="text-[var(--color-muted)]">Fat</dt>
          <dd>{item.macros.fat}g</dd>
        </div>
      </dl>
      <p class="font-data text-xs text-[var(--color-muted)]" data-result-macro-basis>{macroBasisLabel}</p>
      <p class="font-data text-xs" data-result-calories>{item.calories} kcal {macroBasisLabel}</p>
    </div>
  </div>

  {#if similarityScore !== null}
    <div class="flex items-center gap-2" data-result-similarity>
      <span class="font-data text-sm" data-result-similarity-score
        >{Math.round(similarityScore * 100)}% match</span
      >
      {#if similarityTier}
        <span
          class={`rounded px-2 py-0.5 text-xs font-medium ${tierStyles[similarityTier].classes}`}
          data-result-similarity-tier
          >{tierStyles[similarityTier].label}</span
        >
      {/if}
    </div>
  {/if}
</article>
