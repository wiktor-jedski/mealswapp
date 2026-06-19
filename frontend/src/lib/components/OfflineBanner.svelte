<script lang="ts">
  import { offlineStatus } from "../stores/offline";

  // Implements DESIGN-001 OfflineBanner online/offline and stale-data indicators.

  $: online = $offlineStatus.online;
  $: showingCached = $offlineStatus.showingCached;
  $: showingStale = $offlineStatus.showingStale;
  $: message = resolveOfflineBannerMessage(online, showingCached, showingStale);

  function resolveOfflineBannerMessage(
    online: boolean,
    showingCached: boolean,
    showingStale: boolean
  ): string {
    if (online) {
      return "Online";
    }
    if (showingCached) {
      return "You're offline. Showing cached results.";
    }
    if (showingStale) {
      return "You're offline. Results may be stale.";
    }
    return "You're offline. Search is unavailable until you reconnect.";
  }
</script>

<!-- Implements DESIGN-001 OfflineBanner -->
{#if !online}
  <div
    class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-2 text-sm font-medium"
    role="status"
    aria-live="polite"
    data-offline-banner
    data-online="false"
    data-showing-cached={showingCached ? "true" : "false"}
    data-showing-stale={showingStale ? "true" : "false"}
  >
    {message}
  </div>
{/if}
