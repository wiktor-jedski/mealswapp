<script lang="ts">
  import type { OfflineState } from '../offline/offlineState';

  interface Props {
    state: OfflineState;
    onRetry?: () => void;
  }

  let { state, onRetry }: Props = $props();
</script>

{#if state.status !== 'online' || state.message}
  <section
    class="border-b border-secondary bg-secondary px-4 py-2 text-sm text-text-primary"
    aria-live="polite"
    aria-label="Connection status"
  >
    <div class="mx-auto flex max-w-app flex-wrap items-center justify-between gap-2">
      <div>
        <span class="font-mono text-xs uppercase text-primary">{state.status}</span>
        <p class="mt-1">{state.message}</p>
      </div>
      {#if state.queuedRetries > 0 || state.status === 'reconnecting'}
        <button
          class="rounded border border-primary px-3 py-1.5 text-sm font-medium text-primary disabled:opacity-50"
          type="button"
          disabled={!state.isOnline}
          onclick={() => onRetry?.()}
        >
          Retry
        </button>
      {/if}
    </div>
  </section>
{/if}
