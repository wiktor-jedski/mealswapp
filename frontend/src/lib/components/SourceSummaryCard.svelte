<script lang="ts">
  import type { SourceSummary, SubstitutionUnit } from "../api/generated";
  import { preferencesStore } from "../stores/preferences";
  import { convertQuantity, formatDisplayQuantity, unitLabel } from "../units";

  // Implements DESIGN-001 ResultsGrid substitution source summary card.

  /** Backend-calculated totals for the user's selected substitution input list. */
  export let sourceSummary: SourceSummary;

  /** Display mass total in the active sidebar unit system, keeping mass separate from volume. */
  $: massUnit = ($preferencesStore.unitSystem === "imperial" ? "oz" : "g") satisfies SubstitutionUnit;

  /** Display volume total in the active sidebar unit system, keeping volume separate from mass. */
  $: volumeUnit = ($preferencesStore.unitSystem === "imperial" ? "fl_oz" : "ml") satisfies SubstitutionUnit;

  /** User-facing mass amount. */
  $: displayGrams = convertQuantity(sourceSummary.totalGrams, "g", massUnit);

  /** User-facing volume amount. */
  $: displayMilliliters = convertQuantity(sourceSummary.totalMilliliters, "ml", volumeUnit);

  /** True when the input list contains at least one solid/mass-based amount. */
  $: hasMass = sourceSummary.totalGrams > 0;

  /** True when the input list contains at least one liquid/volume-based amount. */
  $: hasVolume = sourceSummary.totalMilliliters > 0;

  function macroValue(value: number): string {
    return formatDisplayQuantity(value);
  }
</script>

<!-- Implements DESIGN-001 ResultsGrid substitution source summary. -->
<article
  class="grid gap-3 rounded border border-[var(--color-primary)] bg-[var(--color-surface)] p-4"
  aria-label="Your Meal"
  data-source-summary-card
>
  <div>
    <h3 class="text-left text-base font-semibold" data-source-summary-title>Your Meal</h3>
  </div>

  <dl class="grid gap-1 font-data text-xs" data-source-summary-macros>
    <div class="grid grid-cols-[5rem_auto] items-center gap-3">
      <dt class="text-[var(--color-muted)]">Protein</dt>
      <dd>{macroValue(sourceSummary.macros.protein)}g</dd>
    </div>
    <div class="grid grid-cols-[5rem_auto] items-center gap-3">
      <dt class="text-[var(--color-muted)]">Carbs</dt>
      <dd>{macroValue(sourceSummary.macros.carbohydrates)}g</dd>
    </div>
    <div class="grid grid-cols-[5rem_auto] items-center gap-3">
      <dt class="text-[var(--color-muted)]">Fat</dt>
      <dd>{macroValue(sourceSummary.macros.fat)}g</dd>
    </div>
    <div class="grid grid-cols-[5rem_auto] items-center gap-3" data-source-summary-calories>
      <dt class="text-[var(--color-muted)]">Calories</dt>
      <dd>{macroValue(sourceSummary.calories)} kcal</dd>
    </div>
  </dl>

  <p class="text-left font-data text-xs text-[var(--color-muted)]" data-source-summary-amount>
    {#if hasMass}
      {formatDisplayQuantity(displayGrams)} {unitLabel(massUnit)}
    {/if}
    {#if hasMass && hasVolume}
      +
    {/if}
    {#if hasVolume}
      {formatDisplayQuantity(displayMilliliters)} {unitLabel(volumeUnit)}
    {/if}
  </p>
</article>
