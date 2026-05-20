<script lang="ts">
  import type { AppError } from '../api/types';
  import { mapErrorMessage } from '../errors/errorHandling';

  interface Props {
    error: AppError;
    onRetry?: () => void;
    onRecover?: () => void;
  }

  let { error, onRetry, onRecover }: Props = $props();
</script>

<section class="rounded border border-error bg-surface p-4" role="alert">
  <p class="text-sm font-bold text-error">{mapErrorMessage(error)}</p>
  {#if error.requestId}
    <p class="mt-2 font-mono text-xs text-text-muted">Request {error.requestId}</p>
  {/if}
  <div class="mt-3 flex flex-wrap gap-2">
    {#if error.retryable}
      <button class="rounded border border-secondary px-3 py-2 text-sm" type="button" onclick={() => onRetry?.()}>Retry</button>
    {/if}
    <button class="rounded border border-secondary px-3 py-2 text-sm" type="button" onclick={() => onRecover?.()}>Dismiss</button>
  </div>
</section>
