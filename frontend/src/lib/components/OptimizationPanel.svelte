<script lang="ts">
  import type { OptimizationState } from '../search/optimizationState';

  interface Props {
    state: OptimizationState;
    onSubmit?: () => void;
    onCancel?: () => void;
  }

  let { state, onSubmit, onCancel }: Props = $props();
  let busy = $derived(state.status === 'submitting' || state.status === 'queued' || state.status === 'processing');
  let alternatives = $derived(state.status === 'failed' ? state.partialAlternatives : state.alternatives);
</script>

<section class="mt-4 rounded border border-secondary bg-surface p-4" aria-live="polite" aria-labelledby="optimization-heading">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h2 id="optimization-heading" class="text-base font-bold">Diet optimization</h2>
      <p class="mt-1 text-sm text-text-muted">Submit current diet-mode results for macro-balanced alternatives.</p>
    </div>
    <div class="flex gap-2">
      {#if busy}
        <button class="rounded border border-secondary px-3 py-2 text-sm" type="button" onclick={() => onCancel?.()}>
          Cancel
        </button>
      {/if}
      <button
        class="rounded bg-primary px-3 py-2 text-sm font-medium text-white disabled:opacity-50"
        type="button"
        disabled={busy}
        onclick={() => onSubmit?.()}
      >
        {busy ? 'Optimizing' : 'Optimize diet'}
      </button>
    </div>
  </div>

  {#if state.status !== 'idle'}
    <div class="mt-4">
      <div class="flex items-center justify-between gap-3 font-mono text-xs text-text-muted">
        <span>{state.status}</span>
        <span>{state.progress}%</span>
      </div>
      <div class="mt-2 h-2 rounded bg-background">
        <div class="h-2 rounded bg-primary" style={`width: ${Math.min(Math.max(state.progress, 0), 100)}%`}></div>
      </div>
    </div>
  {/if}

  {#if state.message}
    <p class="mt-3 text-sm" class:text-error={state.status === 'failed'}>{state.message}</p>
  {/if}

  {#if alternatives.length > 0}
    <div class="mt-4 grid gap-3">
      {#each alternatives as alternative, index}
        <article class="rounded border border-secondary p-3">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h3 class="text-sm font-bold">Alternative {index + 1}</h3>
            <span class="font-mono text-xs text-text-muted">{Math.round(alternative.calories)} kcal</span>
          </div>
          <dl class="mt-2 grid grid-cols-3 gap-2 font-mono text-xs text-text-muted">
            <div>
              <dt>Protein</dt>
              <dd class="text-text-primary">{Math.round(alternative.macros.protein)} g</dd>
            </div>
            <div>
              <dt>Carbs</dt>
              <dd class="text-text-primary">{Math.round(alternative.macros.carbs)} g</dd>
            </div>
            <div>
              <dt>Fat</dt>
              <dd class="text-text-primary">{Math.round(alternative.macros.fat)} g</dd>
            </div>
          </dl>
          <ul class="mt-2 grid gap-1 text-sm">
            {#each alternative.meals as meal}
              <li class="flex justify-between gap-3">
                <span>{meal.itemId}</span>
                <span class="font-mono text-xs text-text-muted">{meal.quantity} g</span>
              </li>
            {/each}
          </ul>
        </article>
      {/each}
    </div>
  {/if}
</section>
