<script lang="ts">
  import ErrorMessage from './ErrorMessage.svelte';
  import type { AppError } from '../api/types';
  import { createErrorBoundaryState, resetBoundary, type ErrorBoundaryState } from '../errors/errorHandling';

  interface Props {
    children?: import('svelte').Snippet;
    fallbackError?: AppError;
    onRetry?: () => void;
    onRecover?: () => void;
  }

  let { children, fallbackError, onRetry, onRecover }: Props = $props();
  let boundary: ErrorBoundaryState = $state(createErrorBoundaryState());

  $effect(() => {
    if (fallbackError) {
      boundary = { ...boundary, hasError: true, error: fallbackError };
    }
  });

  function recover() {
    boundary = resetBoundary(boundary);
    onRecover?.();
  }
</script>

{#if boundary.hasError && boundary.error}
  <main class="min-h-screen bg-background p-4 text-text-primary">
    <ErrorMessage error={boundary.error} onRetry={onRetry} onRecover={recover} />
  </main>
{:else}
  {@render children?.()}
{/if}
