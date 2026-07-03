<script lang="ts">
  import { createMutation, createQuery } from "@tanstack/svelte-query";
  import {
    buildCheckoutMutationOptions,
    buildEntitlementQueryOptions,
    EntitlementClientError,
    type CheckoutMutationVariables,
    type EntitlementQueryKey
  } from "../api/entitlement-client";
  import type {
    CheckoutPlan,
    EntitlementStatusData,
    CheckoutSessionData
  } from "../api/generated";
  import {
    setEntitlementError,
    setEntitlementStatus
  } from "../stores/entitlement";

  // Implements DESIGN-007 SubscriptionController hosted checkout controls and billing recovery UI.

  const priceLabels: Record<CheckoutPlan, { title: string; cadence: string; description: string }> = {
    monthly: {
      title: "Monthly",
      cadence: "$3 / month",
      description: "Flexible access with monthly billing."
    },
    annual: {
      title: "Annual",
      cadence: "$25 / year",
      description: "Twelve months of paid access with annual billing."
    }
  };

  /** Generated checkout plans rendered as distinct hosted-checkout choices. */
  const checkoutPlans: CheckoutPlan[] = ["monthly", "annual"];

  /** Current browser return route used after hosted checkout redirects back into the SPA. */
  let returnState = $state<"success" | "cancel" | null>(null);

  /** Last checkout plan selected by the user, used for loading and retry affordances. */
  let selectedPlan = $state<CheckoutPlan | null>(null);

  /** Whether checkout is being created from a billing recovery CTA instead of the plan cards. */
  let recoveryCheckout = $state(false);

  /** Entitlement status query drives billing state and refreshes return routes. */
  const entitlementQuery = createQuery<
    EntitlementStatusData,
    EntitlementClientError,
    EntitlementStatusData,
    EntitlementQueryKey
  >(() => buildEntitlementQueryOptions());

  /** Hosted-checkout mutation uses generated billing contracts and idempotency behavior. */
  const checkoutMutation = createMutation<CheckoutSessionData, EntitlementClientError, CheckoutMutationVariables>(
    () => ({
      ...buildCheckoutMutationOptions(),
      onSuccess: (checkout) => {
        window.location.assign(checkout.checkoutUrl);
      }
    })
  );

  $effect(() => {
    const path = window.location.pathname;
    if (path === "/billing/success") {
      returnState = "success";
      void entitlementQuery.refetch();
    } else if (path === "/billing/cancel") {
      returnState = "cancel";
      void entitlementQuery.refetch();
    }
  });

  $effect(() => {
    if (entitlementQuery.data) {
      setEntitlementStatus(entitlementQuery.data);
    }
  });

  $effect(() => {
    const error = entitlementQuery.error ?? checkoutMutation.error;
    if (error instanceof EntitlementClientError) {
      setEntitlementError(error.appError);
    }
  });

  let status = $derived(entitlementQuery.data ?? null);
  let loadingPlan = $derived(checkoutMutation.isPending ? selectedPlan : null);
  let checkoutError = $derived(
    checkoutMutation.error instanceof EntitlementClientError ? checkoutMutation.error.appError.message : null
  );
  let entitlementError = $derived(
    entitlementQuery.error instanceof EntitlementClientError ? entitlementQuery.error.appError.message : null
  );
  let recoveryRequired = $derived(
    status?.billingRecoveryState === "action_required" ||
      status?.billingRecoveryState === "cancelled" ||
      status?.status === "past_due" ||
      status?.status === "cancelled"
  );
  let recoveryLabel = $derived(status?.status === "cancelled" ? "Restart billing" : "Update billing");

  /** Starts server-side checkout creation; raw card details are collected only by the hosted provider. */
  function startCheckout(plan: CheckoutPlan, recovery = false): void {
    selectedPlan = plan;
    recoveryCheckout = recovery;
    checkoutMutation.mutate({
      request: {
        plan,
        successUrl: buildReturnUrl("success", plan),
        cancelUrl: buildReturnUrl("cancel", plan)
      }
    });
  }

  /** Builds absolute return URLs accepted by the generated checkout creation request. */
  function buildReturnUrl(state: "success" | "cancel", plan: CheckoutPlan): string {
    const url = new URL(`/billing/${state}`, window.location.origin);
    url.searchParams.set("plan", plan);
    return url.toString();
  }

  /** Refetches entitlement status after a recoverable billing or return-route failure. */
  function retryEntitlement(): void {
    void entitlementQuery.refetch();
  }
</script>

<!-- Implements DESIGN-007 SubscriptionController billing-state, checkout, cancellation return, and success return UI. -->
<section
  class="grid gap-4 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4"
  aria-labelledby="subscription-billing-title"
  data-subscription-billing
>
  <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
    <div>
      <h2 id="subscription-billing-title" class="text-lg font-bold text-[var(--color-text)]">
        Subscription
      </h2>
      <p class="text-sm text-[var(--color-muted)]">
        Manage paid search access through hosted checkout.
      </p>
    </div>

    {#if status}
      <div class="rounded border border-[var(--color-border)] px-3 py-2 text-sm">
        <span class="font-[var(--font-data)] uppercase tracking-normal text-[var(--color-muted)]">Plan</span>
        <span class="ml-2 font-semibold capitalize">{status.tier} · {status.status}</span>
      </div>
    {/if}
  </div>

  {#if returnState === "success"}
    <p class="rounded border border-[var(--color-primary)] bg-[var(--color-secondary)] px-3 py-2 text-sm text-[var(--color-text)]" role="status">
      Checkout completed. Billing access is refreshing.
    </p>
  {:else if returnState === "cancel"}
    <p class="rounded border border-[var(--color-border)] px-3 py-2 text-sm text-[var(--color-muted)]" role="status">
      Checkout was cancelled. Your current entitlement is unchanged.
    </p>
  {/if}

  {#if entitlementQuery.isFetching && !status}
    <div class="grid gap-2" aria-label="Loading billing status">
      <div class="h-5 w-40 rounded bg-[var(--color-secondary)]"></div>
      <div class="h-16 rounded bg-[var(--color-secondary)]"></div>
    </div>
  {:else if entitlementError}
    <div class="grid gap-3 rounded border border-[var(--color-error)] p-3" role="alert">
      <p class="text-sm text-[var(--color-error)]">{entitlementError}</p>
      <button
        type="button"
        class="w-fit rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)] transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
        onclick={retryEntitlement}
      >
        Retry billing status
      </button>
    </div>
  {/if}

  {#if recoveryRequired}
    <div class="grid gap-3 rounded border border-[var(--color-error)] p-3" data-billing-recovery>
      <div>
        <h3 class="text-base font-semibold text-[var(--color-text)]">Billing action needed</h3>
        <p class="text-sm text-[var(--color-muted)]">
          Paid features are paused until billing is recovered through hosted checkout.
        </p>
      </div>
      <button
        type="button"
        class="w-fit rounded bg-[var(--color-accent)] px-3 py-2 text-sm font-semibold text-[var(--color-on-accent)] transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:cursor-wait disabled:opacity-70"
        disabled={checkoutMutation.isPending}
        onclick={() => startCheckout("monthly", true)}
      >
        {checkoutMutation.isPending && recoveryCheckout ? "Opening hosted checkout..." : recoveryLabel}
      </button>
    </div>
  {/if}

  <div class="grid gap-3 sm:grid-cols-2">
    {#each checkoutPlans as plan}
      <article class="grid gap-3 rounded border border-[var(--color-border)] p-3">
        <div>
          <h3 class="text-base font-semibold text-[var(--color-text)]">{priceLabels[plan].title}</h3>
          <p class="font-[var(--font-data)] text-sm text-[var(--color-muted)]">{priceLabels[plan].cadence}</p>
          <p class="mt-1 text-sm text-[var(--color-muted)]">{priceLabels[plan].description}</p>
        </div>
        <button
          type="button"
          class="rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)] transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:cursor-wait disabled:opacity-70"
          disabled={checkoutMutation.isPending}
          aria-describedby={checkoutError && selectedPlan === plan ? `checkout-error-${plan}` : undefined}
          onclick={() => startCheckout(plan)}
        >
          {loadingPlan === plan && !recoveryCheckout ? "Creating checkout..." : `Choose ${priceLabels[plan].title}`}
        </button>
        {#if checkoutError && selectedPlan === plan && !checkoutMutation.isPending}
          <p id={`checkout-error-${plan}`} class="text-sm text-[var(--color-error)]" role="alert">
            {checkoutError}
          </p>
          <button
            type="button"
            class="w-fit rounded border border-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-primary)] transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
            onclick={() => startCheckout(plan, recoveryCheckout)}
          >
            Retry checkout
          </button>
        {/if}
      </article>
    {/each}
  </div>
</section>
