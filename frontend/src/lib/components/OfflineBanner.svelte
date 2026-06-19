<script lang="ts">
  // Implements DESIGN-001 OfflineBanner cached/stale/online-only status inputs.
  export let online: boolean;
  export let cached = false;
  export let stale = false;
  export let cacheMiss = false;
</script>

<!-- Implements DESIGN-001 OfflineBanner connectivity feedback without Phase 09 interception claims. -->
{#if !online || cached}
  <div role="status" class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-3">
    {#if cached && stale}Cached stale results are shown. Reconnect to refresh.
    {:else if cached}Cached results are shown while offline.
    {:else if cacheMiss}You are offline and this search is not cached. Reconnect and try again.
    {:else}You are offline. Cached searches remain available.{/if}
  </div>
{/if}
