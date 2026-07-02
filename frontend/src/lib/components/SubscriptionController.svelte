<script lang="ts">
  import { createQuery, useQueryClient } from "@tanstack/svelte-query";
  import { buildEntitlementQueryOptions, createCheckoutSession } from "../api/entitlement-client";
  import { onMount } from "svelte";

  // Implements DESIGN-007 SubscriptionController frontend billing controls.

  const queryClient = useQueryClient();
  const entitlementQuery = createQuery(() => buildEntitlementQueryOptions());
  
  let processing = $state(false);
  let error = $state<string | null>(null);

  let entitlement = $derived(entitlementQuery.data);
  let tier = $derived(entitlement?.tier);
  let status = $derived(entitlement?.status);

  onMount(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.has("success") || params.has("canceled")) {
      // Clean up URL
      const newUrl = window.location.pathname;
      window.history.replaceState({}, document.title, newUrl);
      
      // Refresh entitlement state immediately upon returning from Stripe
      void queryClient.invalidateQueries({ queryKey: ["entitlement"] });
    }
  });

  async function handleCheckout(priceId: string) {
    processing = true;
    error = null;
    try {
      const response = await createCheckoutSession({
        priceId,
        successUrl: window.location.origin + window.location.pathname + "?success=true",
        cancelUrl: window.location.origin + window.location.pathname + "?canceled=true"
      });
      // Follow Stripe redirect URL from server; no PAN/CVC is collected locally.
      window.location.href = response.checkoutUrl;
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to start checkout";
      processing = false;
    }
  }
</script>

<section class="grid gap-2 border-t border-[var(--color-border)] pt-4 mt-4" aria-label="Subscription" data-subscription-controller>
  <h2 class="font-data text-xs uppercase text-[var(--color-muted)]">Subscription</h2>
  
  {#if entitlementQuery.isLoading}
    <p class="text-sm text-[var(--color-muted)]" data-loading>Loading...</p>
  {:else if entitlementQuery.isError}
    <div class="grid gap-2" data-error>
      <p class="text-sm text-red-500">Could not load billing state.</p>
      <button 
        type="button"
        class="w-full rounded bg-[var(--color-primary)] px-2 py-1 text-sm text-[var(--color-bg)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
        onclick={() => entitlementQuery.refetch()}
      >
        Retry
      </button>
    </div>
  {:else if entitlement}
    {#if status === "past_due" || status === "cancelled"}
      <div class="rounded border border-[var(--color-primary)] bg-[var(--color-surface)] p-2 text-sm text-[var(--color-primary)]" data-recovery-message>
        <p>Your subscription is {status === "past_due" ? "past due" : "cancelled"}. Update your payment method to restore access.</p>
        <button 
          type="button"
          class="mt-2 w-full rounded bg-[var(--color-primary)] px-2 py-1 text-[var(--color-bg)] hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:opacity-50"
          onclick={() => handleCheckout("price_monthly")}
          disabled={processing}
          data-recovery-action
        >
          {processing ? "Processing..." : "Update Billing"}
        </button>
      </div>
    {:else if tier === "free" || tier === "trial"}
      <div class="grid gap-2">
        <p class="text-sm text-[var(--color-muted)]">Upgrade for unlimited searches and features.</p>
        {#if error}
          <p class="text-sm text-red-500" data-checkout-error>{error}</p>
        {/if}
        <button
          type="button"
          class="w-full rounded bg-[var(--color-primary)] px-2 py-1 text-sm text-[var(--color-bg)] focus:outline-none focus:ring-2 focus:ring-offset-1 focus:ring-[var(--color-primary)] disabled:opacity-50"
          onclick={() => handleCheckout("price_monthly")}
          disabled={processing}
          data-checkout-monthly
        >
          {processing ? "Processing..." : "Upgrade Monthly ($5/mo)"}
        </button>
        <button
          type="button"
          class="w-full rounded border border-[var(--color-primary)] bg-transparent px-2 py-1 text-sm text-[var(--color-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:opacity-50"
          onclick={() => handleCheckout("price_annual")}
          disabled={processing}
          data-checkout-annual
        >
          {processing ? "Processing..." : "Upgrade Annual ($50/yr)"}
        </button>
      </div>
    {:else if tier === "paid"}
      <p class="text-sm text-[var(--color-accent)]" data-active-subscription>Active paid subscription.</p>
    {/if}
  {/if}
</section>
