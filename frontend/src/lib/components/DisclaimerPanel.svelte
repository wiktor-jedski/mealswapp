<script lang="ts">
  import { onMount } from "svelte";
  import {
    BUNDLED_LOGIN_DISCLAIMER,
    loadLoginDisclaimerViewModel,
    type DisclaimerViewModel
  } from "./disclaimer-panel";

  // Implements DESIGN-018 DisclaimerPanel mandatory login-screen medical disclaimer UI.

  let disclaimer = $state<DisclaimerViewModel>(BUNDLED_LOGIN_DISCLAIMER);
  let loading = $state(true);

  onMount(() => {
    const controller = new AbortController();
    loadLoginDisclaimerViewModel(undefined, controller.signal)
      .then((nextDisclaimer) => {
        disclaimer = nextDisclaimer;
      })
      .finally(() => {
        loading = false;
      });
    return () => controller.abort();
  });
</script>

<!-- Implements DESIGN-018 DisclaimerPanel login fallback and unavailable-state feedback. -->
<section
  class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4 text-sm leading-6"
  aria-labelledby="auth-disclaimer-heading"
  data-auth-disclaimer
  data-unavailable={disclaimer.unavailable ? "true" : "false"}
>
  <div class="flex flex-wrap items-center justify-between gap-2">
    <h2 id="auth-disclaimer-heading" class="text-base font-semibold">Medical disclaimer</h2>
    <span class="font-data text-xs text-[var(--color-muted)]">v{disclaimer.version}</span>
  </div>
  {#if disclaimer.unavailable}
    <p class="mt-2 text-sm text-[var(--color-muted)]" role="status" aria-live="polite" data-disclaimer-fallback>
      Current disclaimer content is temporarily unavailable. Showing the bundled medical disclaimer.
    </p>
  {:else if loading}
    <p class="mt-2 text-sm text-[var(--color-muted)]" role="status" aria-live="polite">Loading disclaimer…</p>
  {/if}
  <p class="mt-3 whitespace-pre-line">{disclaimer.bodyMarkdown}</p>
  <p class="mt-2 font-data text-xs text-[var(--color-muted)]">Effective {disclaimer.effectiveAt}</p>
</section>
