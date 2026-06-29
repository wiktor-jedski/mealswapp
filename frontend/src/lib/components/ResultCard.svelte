<script lang="ts">
  import type { FoodObject, SimilarityMetadata, SimilarityTier } from "../api/generated";
  import { preferencesStore } from "../stores/preferences";
  import {
    convertQuantity,
    displayUnitForBasis,
    formatDisplayQuantity,
    macroBasisDisplayLabel,
    unitLabel
  } from "../units";

  // Implements DESIGN-001 ResultsGrid result card: image fallback, classifications, macros with basis, calories, and similarity badge.

  let {
    item,
    similarity = null,
    fallbackScore = null,
    onAddToSubstitution = null
  }: {
    /** Food object rendered by this card. */
    item: FoodObject;
    /** Similarity metadata matched to this item by `itemId`; null when no similarity row exists (e.g. Catalog mode). */
    similarity?: SimilarityMetadata | null;
    /** Optional fallback similarity score from the parallel `similarityScores` array; used only when metadata is absent. */
    fallbackScore?: number | null;
    /** Optional action that adds this full Catalog item to the Substitution Input list. */
    onAddToSubstitution?: ((item: FoodObject) => void) | null;
  } = $props();

  /** True when the item image failed to load and the category placeholder should replace it. */
  let imageFailed = $state(false);

  /** Food Category classifications (kind === "food_category") rendered as visible chips. */
  let foodCategories = $derived(item.classifications.filter(
    (classification) => classification.kind === "food_category"
  ));

  /** Primary Food Category used for the image placeholder when no image is provided or the image fails to load. */
  let placeholderCategory = $derived(item.primaryFoodCategory ?? foodCategories[0] ?? null);

  /** Initial shown inside the category placeholder tile. */
  let placeholderInitial = $derived(placeholderCategory
    ? placeholderCategory.name.charAt(0).toUpperCase()
    : "?");

  /** Human-readable nutrition basis label, following the sidebar unit preference. */
  let macroBasisLabel = $derived(macroBasisDisplayLabel(item.macroBasis, $preferencesStore.unitSystem));

  /** Metric unit returned by the backend-calculated replacement quantity. */
  let matchingQuantityMetricUnit = $derived.by<"ml" | "g">(() => item.macroBasis === "100ml" ? "ml" : "g");

  /** Unit used to display the backend-calculated replacement quantity. */
  let matchingQuantityDisplayUnit = $derived(displayUnitForBasis(item.macroBasis, $preferencesStore.unitSystem));

  /** User-facing rounded replacement quantity calculated by the backend. */
  let matchingQuantityLabel = $derived(similarity
    ? `${formatDisplayQuantity(convertQuantity(similarity.matchingQuantity, matchingQuantityMetricUnit, matchingQuantityDisplayUnit))} ${unitLabel(matchingQuantityDisplayUnit)}`
    : null);

  /** Scale factor for substitution result macros; Catalog cards remain on the per-100 basis. */
  let macroScale = $derived(similarity ? similarity.matchingQuantity / 100 : 1);

  /** User-facing macro values for the rendered card context. */
  let displayMacros = $derived({
    protein: item.macros.protein * macroScale,
    carbohydrates: item.macros.carbohydrates * macroScale,
    fat: item.macros.fat * macroScale
  });

  /** User-facing calories for the rendered card context. */
  let displayCalories = $derived(item.calories * macroScale);

  /** Context label below the macro table. */
  let macroContextLabel = $derived(matchingQuantityLabel ? `for about ${matchingQuantityLabel}` : macroBasisLabel);

  /** Resolved similarity score: prefer metadata, fall back to the parallel scores array. */
  let similarityScore = $derived(similarity?.score ?? fallbackScore);

  /** Resolved similarity tier; null when no metadata is available. */
  let similarityTier = $derived(similarity?.tier ?? null);

  /** True when an <img> should be rendered: the item provides a URL and it has not failed. */
  let showImage = $derived(Boolean(item.imageUrl) && !imageFailed);

  /** Tailwind classes and short label for each similarity tier badge. */
  const tierStyles: Record<SimilarityTier, { label: string; classes: string }> = {
    excellent: { label: "Excellent", classes: "bg-[var(--color-primary)] text-[var(--color-on-primary)]" },
    good: { label: "Good", classes: "bg-[var(--color-primary)] text-[var(--color-on-primary)]" },
    fair: { label: "Fair", classes: "bg-[var(--color-accent)] text-[var(--color-on-accent)]" },
    poor: { label: "Poor", classes: "bg-[var(--color-muted)] text-[var(--color-on-muted)]" }
  };

  /** Resets the broken-image flag whenever the item's image URL changes so a new image retries. */
  $effect(() => resetBrokenImage(item.imageUrl));

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
  class="relative grid gap-3 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4"
  aria-label={item.name}
  data-result-card
  data-result-id={item.id}
>
  <h3 class="text-left text-base font-semibold" data-result-name>{item.name}</h3>

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
          onerror={onImageError}
          data-result-image
        />
      {:else}
        <div
          class="grid place-items-center text-center"
          role="img"
          aria-label={placeholderCategory ? placeholderCategory.name : item.name}
          data-result-placeholder
        >
          <span class="font-data text-2xl font-semibold text-[var(--color-on-muted)]" aria-hidden="true">{placeholderInitial}</span>
          {#if placeholderCategory}
            <span class="mt-1 px-1 text-xs text-[var(--color-on-muted)]">{placeholderCategory.name}</span>
          {/if}
        </div>
      {/if}
    </div>

    <div class="grid h-24 content-between">
      <dl class="grid gap-1 font-data text-xs" data-result-macros>
        <div class="grid grid-cols-[5rem_auto] items-center gap-3">
          <dt class="text-[var(--color-muted)]">Protein</dt>
          <dd>{formatDisplayQuantity(displayMacros.protein)}g</dd>
        </div>
        <div class="grid grid-cols-[5rem_auto] items-center gap-3">
          <dt class="text-[var(--color-muted)]">Carbs</dt>
          <dd>{formatDisplayQuantity(displayMacros.carbohydrates)}g</dd>
        </div>
        <div class="grid grid-cols-[5rem_auto] items-center gap-3">
          <dt class="text-[var(--color-muted)]">Fat</dt>
          <dd>{formatDisplayQuantity(displayMacros.fat)}g</dd>
        </div>
        <div class="grid grid-cols-[5rem_auto] items-center gap-3" data-result-calories>
          <dt class="text-[var(--color-muted)]">Calories</dt>
          <dd>{formatDisplayQuantity(displayCalories)} kcal</dd>
        </div>
      </dl>
      <p class="font-data text-[0.68rem] leading-none text-[var(--color-muted)]" data-result-macro-basis>{macroContextLabel}</p>
    </div>
  </div>

  {#if foodCategories.length > 0}
    <div class="flex flex-wrap justify-start gap-1 pr-12 text-left" data-result-categories>
      {#each foodCategories as category (category.id)}
        <span class="rounded bg-[var(--color-muted)] px-2 py-0.5 text-xs text-[var(--color-on-muted)]">{category.name}</span>
      {/each}
    </div>
  {/if}

  {#if onAddToSubstitution}
    <button
      type="button"
      class="absolute bottom-4 right-4 flex h-9 w-9 items-center justify-center rounded-full border border-[var(--color-primary)] bg-[var(--color-primary)] text-xl font-semibold leading-none text-[var(--color-on-primary)] shadow-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
      aria-label={`Add ${item.name} to substitutions`}
      onclick={() => onAddToSubstitution?.(item)}
      data-result-add-substitution
    >
      <span class="-translate-y-px leading-none" aria-hidden="true">+</span>
    </button>
  {/if}

  {#if similarityScore !== null}
    <div class="flex flex-wrap items-center gap-2" data-result-similarity>
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
