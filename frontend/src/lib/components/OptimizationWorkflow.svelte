<script lang="ts">
  import type { DietOptimizationRequest } from "../api/generated";
  import { dailyDietStore } from "../stores/daily-diet";
  import { createOptimizationController, type OptimizationState } from "../stores/optimization";

  // Implements DESIGN-001 SearchView OptimizationWorkflow for the selected saved Daily Diet.
  // Implements DESIGN-004 JobStatusTracker queued/processing/terminal state presentation.

  let {
    selectedDietId = null,
    identityId = null,
    executionAllowed = true
  }: {
    selectedDietId?: string | null;
    identityId?: string | null;
    executionAllowed?: boolean;
  } = $props();

  const controller = createOptimizationController();
  const optimizationStore = controller.store;
  let configuredIdentityId = $state<string | null | undefined>(undefined);
  let configuredDietId = $state<string | null>(null);
  let tolerancePercent = $state(10);
  let formError = $state<string | null>(null);

  let selectedDiet = $derived(
    selectedDietId ? $dailyDietStore.collections.find((diet) => diet.id === selectedDietId) ?? null : null
  );
  let optimizationState = $derived<OptimizationState>($optimizationStore);
  let activeRequest = $derived(selectedDietId ? buildRequest(selectedDietId, tolerancePercent) : null);
  let busy = $derived(optimizationState.phase === "submitting" || optimizationState.phase === "queued" || optimizationState.phase === "processing");
  let canSubmit = $derived(Boolean(selectedDiet && executionAllowed && activeRequest && !busy && $dailyDietStore.mutation === "idle"));

  $effect(() => {
    if (configuredIdentityId === identityId && configuredDietId === selectedDietId) return;
    configuredIdentityId = identityId;
    configuredDietId = selectedDietId;
    controller.setIdentity(identityId);
    controller.setDiet(selectedDietId);
    formError = null;
    void controller.resume();
  });

  $effect(() => () => controller.dispose());

  function buildRequest(
    dailyDietId: string,
    tolerance: number
  ): DietOptimizationRequest {
    return {
      dailyDietId,
      tolerancePercent: tolerance,
      excludedMealIds: []
    };
  }

  function updateTolerance(event: Event): void {
    const value = Number((event.currentTarget as HTMLInputElement).value);
    tolerancePercent = value;
    formError = null;
  }

  async function submitOptimization(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    if (!activeRequest || !canSubmit) return;
    if (!Number.isFinite(activeRequest.tolerancePercent) || activeRequest.tolerancePercent < 0) {
      formError = "Tolerance must be zero or greater.";
      return;
    }
    if (activeRequest.tolerancePercent > 100) {
      formError = "Tolerance must be 100% or less.";
      return;
    }
    formError = null;
    await controller.submit(activeRequest);
  }

  async function retryOptimization(): Promise<void> {
    formError = null;
    await controller.retry(activeRequest ?? undefined);
  }

  function formatNumber(value: number): string {
    return Number.isInteger(value) ? String(value) : value.toFixed(1);
  }
</script>

<!-- Implements DESIGN-001 SearchView selected Daily Diet optimization controls and result view. -->
<section
  class="grid gap-4 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4"
  aria-labelledby="optimization-title"
  data-optimization-workflow
>
  <div class="grid gap-1">
    <h2 id="optimization-title" class="text-lg font-semibold">Optimize this Daily Diet</h2>
    <p class="text-sm text-[var(--color-muted)]">Alternatives match the selected diet’s server-calculated macro targets.</p>
  </div>

  {#if !selectedDiet}
    <p class="rounded border border-dashed border-[var(--color-border)] px-3 py-3 text-sm text-[var(--color-muted)]" role="status" data-optimization-empty>
      Select a saved Daily Diet to generate alternatives.
    </p>
  {:else}
    <form class="grid gap-4" aria-label="Daily Diet optimization form" onsubmit={submitOptimization}>
      <fieldset class="grid gap-3">
        <legend class="font-data text-xs uppercase text-[var(--color-muted)]">Server-derived target macros</legend>
        <dl class="grid grid-cols-2 gap-3 rounded border border-[var(--color-border)] p-3 font-data text-sm sm:grid-cols-3">
          <div><dt class="text-[var(--color-muted)]">Protein</dt><dd data-optimization-target-protein>{formatNumber(selectedDiet.aggregateMacros.protein)}g</dd></div>
          <div><dt class="text-[var(--color-muted)]">Carbohydrates</dt><dd data-optimization-target-carbohydrates>{formatNumber(selectedDiet.aggregateMacros.carbohydrates)}g</dd></div>
          <div><dt class="text-[var(--color-muted)]">Fat</dt><dd data-optimization-target-fat>{formatNumber(selectedDiet.aggregateMacros.fat)}g</dd></div>
        </dl>
        <div class="grid gap-3 sm:max-w-xs">
          <label class="grid gap-1 text-sm" for="optimization-tolerance">
            Tolerance (%)
            <input id="optimization-tolerance" class="rounded border border-[#E0E0E0] bg-white px-3 py-2 font-data text-sm text-[#111827] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" type="number" min="0" max="100" step="0.1" value={tolerancePercent} oninput={updateTolerance} disabled={!executionAllowed || busy} />
          </label>
        </div>
      </fieldset>

      {#if !executionAllowed}
        <p class="rounded border border-[var(--color-accent)] px-3 py-2 text-sm" role="alert" data-optimization-entitlement>
          Optimization is available on active trial and paid plans.
        </p>
      {/if}

      {#if formError}
        <p class="rounded border border-[var(--color-error)] px-3 py-2 text-sm" role="alert" data-optimization-form-error>{formError}</p>
      {/if}

      <div class="flex flex-wrap items-center gap-2">
        <button
          type="submit"
          class="rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)] transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:cursor-not-allowed"
          disabled={!canSubmit}
          data-optimization-submit
        >
          {#if optimizationState.phase === "submitting"}Submitting…{:else if busy}Optimization in progress…{:else}Generate alternatives{/if}
        </button>
        {#if optimizationState.retryMode !== "none" && optimizationState.phase !== "completed"}
          <button type="button" class="rounded border px-3 py-2 text-sm transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" onclick={() => void retryOptimization()} data-optimization-retry>
            Try again
          </button>
        {/if}
      </div>
    </form>

    {#if optimizationState.phase === "submitting" || optimizationState.phase === "queued" || optimizationState.phase === "processing"}
      <div class="grid gap-2 rounded border border-[var(--color-border)] p-3" role="status" aria-live="polite" data-optimization-progress>
        <p class="text-sm font-medium">
          {#if optimizationState.phase === "submitting"}Submitting your saved diet…{:else if optimizationState.phase === "queued"}Queued for optimization…{:else}Building validated alternatives…{/if}
        </p>
        <div class="grid gap-2 sm:grid-cols-3" aria-hidden="true" data-optimization-skeleton>
          {#each [1, 2, 3] as card}
            <div class="grid gap-2 rounded border border-[var(--color-border)] p-3 motion-safe:animate-pulse" data-optimization-skeleton-card={card}>
              <span class="h-4 w-2/3 rounded bg-[var(--color-border)]"></span>
              <span class="h-3 w-full rounded bg-[var(--color-border)]"></span>
              <span class="h-3 w-4/5 rounded bg-[var(--color-border)]"></span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    {#if optimizationState.phase === "failed" || optimizationState.phase === "expired"}
      <div class="grid gap-2 rounded border border-[var(--color-error)] p-3" role="alert" data-optimization-error>
        <p class="font-medium">{optimizationState.failure?.message ?? "Optimization could not be completed."}</p>
        {#if optimizationState.failure?.code === "solver_infeasible"}
          <p class="text-sm text-[var(--color-muted)]">Try increasing the tolerance or editing the saved Daily Diet.</p>
        {/if}
      </div>
    {/if}

    {#if optimizationState.alternatives.length > 0}
      <section class="grid gap-3" aria-labelledby="optimization-results-title" data-optimization-results>
        <div class="grid gap-1">
          <h3 id="optimization-results-title" class="font-data text-xs uppercase text-[var(--color-muted)]">Validated alternatives</h3>
          <p class="text-sm text-[var(--color-muted)]">{optimizationState.alternatives.length} {optimizationState.alternatives.length === 1 ? "alternative" : "alternatives"} found.</p>
        </div>
        <ol class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {#each optimizationState.alternatives as alternative, index}
            <li class="grid gap-3 rounded border border-[var(--color-border)] p-3" data-optimization-alternative={index + 1}>
              <h4 class="font-medium">Alternative {index + 1}</h4>
              <dl class="grid grid-cols-2 gap-2 font-data text-sm">
                <div><dt class="text-[var(--color-muted)]">Protein</dt><dd data-optimization-protein>{formatNumber(alternative.macros.protein)}g</dd></div>
                <div><dt class="text-[var(--color-muted)]">Carbs</dt><dd data-optimization-carbs>{formatNumber(alternative.macros.carbohydrates)}g</dd></div>
                <div><dt class="text-[var(--color-muted)]">Fat</dt><dd data-optimization-fat>{formatNumber(alternative.macros.fat)}g</dd></div>
                <div><dt class="text-[var(--color-muted)]">Calories</dt><dd data-optimization-calories>{formatNumber(alternative.macros.calories)} kcal</dd></div>
              </dl>
              <p class="text-xs text-[var(--color-muted)]">Similarity {Math.round(alternative.similarityScore * 100)}%</p>
              <ul class="grid gap-1 border-t border-[var(--color-border)] pt-2 text-xs text-[var(--color-muted)]" aria-label={`Meals in alternative ${index + 1}`}>
                {#each alternative.meals as meal}
                  <li>{meal.mealId} · {formatNumber(meal.quantity)} {meal.unit}</li>
                {/each}
              </ul>
            </li>
          {/each}
        </ol>
        {#if optimizationState.phase === "completed"}
          <button type="button" class="justify-self-start rounded border px-3 py-2 text-sm transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" onclick={() => activeRequest && void controller.submit(activeRequest)} data-optimization-new>
            Generate fresh alternatives
          </button>
        {/if}
      </section>
    {/if}
  {/if}
</section>
