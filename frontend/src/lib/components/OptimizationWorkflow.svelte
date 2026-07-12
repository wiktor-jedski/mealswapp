<script lang="ts">
  import type { DietOptimizationRequest } from "../api/generated";
  import { dailyDietStore } from "../stores/daily-diet";
  import { createOptimizationController, type OptimizationState } from "../stores/optimization";

  // Implements DESIGN-001 SearchView OptimizationWorkflow for the selected saved Daily Diet.
  // Implements DESIGN-004 JobStatusTracker queued/processing/terminal state presentation.

  let {
    selectedDietId = null,
    executionAllowed = true
  }: {
    selectedDietId?: string | null;
    executionAllowed?: boolean;
  } = $props();

  const controller = createOptimizationController();
  const optimizationStore = controller.store;
  let configuredDietId = $state<string | null>(null);
  let targetProtein = $state(0);
  let targetCarbohydrates = $state(0);
  let targetFat = $state(0);
  let tolerancePercent = $state(10);
  let formError = $state<string | null>(null);

  let selectedDiet = $derived(
    selectedDietId ? $dailyDietStore.collections.find((diet) => diet.id === selectedDietId) ?? null : null
  );
  let optimizationState = $derived<OptimizationState>($optimizationStore);
  let activeRequest = $derived(selectedDietId ? buildRequest(selectedDietId, targetProtein, targetCarbohydrates, targetFat, tolerancePercent) : null);
  let busy = $derived(optimizationState.phase === "submitting" || optimizationState.phase === "queued" || optimizationState.phase === "processing");
  let canSubmit = $derived(Boolean(selectedDiet && executionAllowed && activeRequest && !busy));

  $effect(() => {
    if (configuredDietId === selectedDietId) return;
    configuredDietId = selectedDietId;
    controller.setDiet(selectedDietId);
    formError = null;
    if (selectedDiet) {
      targetProtein = selectedDiet.aggregateMacros.protein;
      targetCarbohydrates = selectedDiet.aggregateMacros.carbohydrates;
      targetFat = selectedDiet.aggregateMacros.fat;
    } else {
      targetProtein = 0;
      targetCarbohydrates = 0;
      targetFat = 0;
    }
  });

  $effect(() => () => controller.dispose());

  function buildRequest(
    dailyDietId: string,
    protein: number,
    carbohydrates: number,
    fat: number,
    tolerance: number
  ): DietOptimizationRequest {
    return {
      dailyDietId,
      targetMacros: { protein, carbohydrates, fat },
      tolerancePercent: tolerance,
      excludedMealIds: []
    };
  }

  function updateNumber(field: "protein" | "carbohydrates" | "fat" | "tolerance", event: Event): void {
    const value = Number((event.currentTarget as HTMLInputElement).value);
    if (field === "protein") targetProtein = value;
    if (field === "carbohydrates") targetCarbohydrates = value;
    if (field === "fat") targetFat = value;
    if (field === "tolerance") tolerancePercent = value;
    formError = null;
  }

  async function submitOptimization(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    if (!activeRequest || !canSubmit) return;
    if ([...Object.values(activeRequest.targetMacros), activeRequest.tolerancePercent].some((value) => !Number.isFinite(value) || value < 0)) {
      formError = "Targets and tolerance must be zero or greater.";
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
    await controller.retry();
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
    <p class="text-sm text-[var(--color-muted)]">Keep the selected diet’s macro targets or adjust them before generating alternatives.</p>
  </div>

  {#if !selectedDiet}
    <p class="rounded border border-dashed border-[var(--color-border)] px-3 py-3 text-sm text-[var(--color-muted)]" role="status" data-optimization-empty>
      Select a saved Daily Diet to generate alternatives.
    </p>
  {:else}
    <form class="grid gap-4" aria-label="Daily Diet optimization form" onsubmit={submitOptimization}>
      <fieldset class="grid gap-3">
        <legend class="font-data text-xs uppercase text-[var(--color-muted)]">Target macros</legend>
        <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <label class="grid gap-1 text-sm" for="optimization-protein">
            Protein (g)
            <input id="optimization-protein" class="rounded border border-[var(--color-border)] bg-transparent px-3 py-2 font-data text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" type="number" min="0" step="0.1" value={targetProtein} oninput={(event) => updateNumber("protein", event)} disabled={!executionAllowed || busy} />
          </label>
          <label class="grid gap-1 text-sm" for="optimization-carbohydrates">
            Carbohydrates (g)
            <input id="optimization-carbohydrates" class="rounded border border-[var(--color-border)] bg-transparent px-3 py-2 font-data text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" type="number" min="0" step="0.1" value={targetCarbohydrates} oninput={(event) => updateNumber("carbohydrates", event)} disabled={!executionAllowed || busy} />
          </label>
          <label class="grid gap-1 text-sm" for="optimization-fat">
            Fat (g)
            <input id="optimization-fat" class="rounded border border-[var(--color-border)] bg-transparent px-3 py-2 font-data text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" type="number" min="0" step="0.1" value={targetFat} oninput={(event) => updateNumber("fat", event)} disabled={!executionAllowed || busy} />
          </label>
          <label class="grid gap-1 text-sm" for="optimization-tolerance">
            Tolerance (%)
            <input id="optimization-tolerance" class="rounded border border-[var(--color-border)] bg-transparent px-3 py-2 font-data text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" type="number" min="0" max="100" step="0.1" value={tolerancePercent} oninput={(event) => updateNumber("tolerance", event)} disabled={!executionAllowed || busy} />
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
          class="rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:cursor-not-allowed disabled:opacity-60"
          disabled={!canSubmit}
          data-optimization-submit
        >
          {#if optimizationState.phase === "submitting"}Submitting…{:else if busy}Optimization in progress…{:else}Generate alternatives{/if}
        </button>
        {#if optimizationState.retryMode !== "none" && optimizationState.phase !== "completed"}
          <button type="button" class="rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" onclick={() => void retryOptimization()} data-optimization-retry>
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
          <p class="text-sm text-[var(--color-muted)]">Try increasing the tolerance or changing the macro targets.</p>
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
          <button type="button" class="justify-self-start rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" onclick={() => activeRequest && void controller.submit(activeRequest)} data-optimization-new>
            Generate fresh alternatives
          </button>
        {/if}
      </section>
    {/if}
  {/if}
</section>
