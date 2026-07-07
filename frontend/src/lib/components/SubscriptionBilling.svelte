<script lang="ts">
  import { createMutation } from "@tanstack/svelte-query";
  import {
    buildCheckoutMutationOptions,
    createBillingPortalSession,
    fetchEntitlementStatus,
    EntitlementClientError,
    type CheckoutMutationVariables
  } from "../api/entitlement-client";
  import type {
    CheckoutPlan,
    CheckoutSessionData,
    BillingPortalSessionData,
    EntitlementStatusData
  } from "../api/generated";
  import {
    entitlementErrorStore,
    entitlementStatusStore,
    setEntitlementError,
    setEntitlementStatus
  } from "../stores/entitlement";
  import { authSessionStore, clearAuthSession } from "../stores/auth-session";
  import { buildAuthGuardDecision, requestProtectedAction } from "../stores/auth-surface";

  // Implements DESIGN-007 SubscriptionController hosted checkout controls and billing recovery UI.
  // Implements DESIGN-018 AuthenticatedActionGuard checkout handoff to login.

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
  const checkoutConfirmationPollIntervalMs = 2_000;
  const checkoutConfirmationPollAttempts = 15;

  /** Current browser return route used after hosted checkout redirects back into the SPA. */
  let returnState = $state<"success" | "cancel" | null>(null);

  /** Last checkout plan selected by the user, used for loading and retry affordances. */
  let selectedPlan = $state<CheckoutPlan | null>(null);

  /** Whether checkout is being created from a billing recovery CTA instead of the plan cards. */
  let recoveryCheckout = $state(false);

  /** Guards hosted-checkout return routes so they refresh entitlement once after mount. */
  let handledReturnRefresh = $state(false);

  /** Manual entitlement refresh state for billing-return and retry controls. */
  let refreshingEntitlement = $state(false);

  /** True while the success return route is waiting for Stripe webhook-confirmed entitlement. */
  let confirmingCheckout = $state(false);

  /** True after the success return route could not observe paid entitlement within the bounded wait. */
  let checkoutConfirmationPending = $state(false);

  /** Hosted-checkout mutation uses generated billing contracts and idempotency behavior. */
  const checkoutMutation = createMutation<CheckoutSessionData, EntitlementClientError, CheckoutMutationVariables>(
    () => ({
      ...buildCheckoutMutationOptions(),
      onSuccess: (checkout) => {
        window.location.assign(checkout.checkoutUrl);
      }
    })
  );

  /** Hosted billing portal mutation sends users to Stripe for cancellation and payment management. */
  const portalMutation = createMutation<BillingPortalSessionData, EntitlementClientError, void>(
    () => ({
      mutationFn: () => createBillingPortalSession({ returnUrl: buildSubscriptionReturnUrl() }),
      onSuccess: (portal) => {
        window.location.assign(portal.portalUrl);
      }
    })
  );

  $effect(() => {
    const path = window.location.pathname;
    if (path === "/billing/success") {
      returnState = "success";
    } else if (path === "/billing/cancel") {
      returnState = "cancel";
    }
    if (returnState === "success" && !handledReturnRefresh) {
      handledReturnRefresh = true;
      void confirmCheckoutEntitlement();
    } else if (returnState === "cancel" && !handledReturnRefresh) {
      handledReturnRefresh = true;
      void refreshEntitlementWhenAllowed();
    }
  });

  let status = $derived($entitlementStatusStore);
  let loadingPlan = $derived(checkoutMutation.isPending ? selectedPlan : null);
  let checkoutError = $derived(
    checkoutMutation.error instanceof EntitlementClientError ? checkoutMutation.error.appError.message : null
  );
  let portalError = $derived(
    portalMutation.error instanceof EntitlementClientError ? portalMutation.error.appError.message : null
  );
  let checkoutErrorId = "checkout-error";
  let portalErrorId = "billing-portal-error";
  let entitlementError = $derived($entitlementErrorStore?.message ?? null);
  let paidActive = $derived(status?.tier === "paid" && status.status === "active");
  let recoveryRequired = $derived(
    status?.billingRecoveryState === "action_required" ||
      status?.billingRecoveryState === "cancelled" ||
      status?.status === "past_due" ||
      status?.status === "cancelled"
  );
  let recoveryLabel = $derived(status?.status === "cancelled" ? "Restart billing" : "Update billing");

  /** Starts server-side checkout creation; raw card details are collected only by the hosted provider. */
  function startCheckout(plan: CheckoutPlan, recovery = false): void {
    const decision = requestProtectedAction($authSessionStore, {
      kind: "checkout",
      label: `Continue ${priceLabels[plan].title} checkout`,
      continueAfterAuth: async () => startCheckout(plan, recovery)
    });
    if (!decision.allowed) {
      if (decision.reason === "expired") {
        clearAuthSession("expired");
      }
      return;
    }

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

  /** Builds the hosted billing portal return URL for this SPA view. */
  function buildSubscriptionReturnUrl(): string {
    return new URL("/subscription", window.location.origin).toString();
  }

  /** Opens Stripe-hosted subscription management for cancellation and payment changes. */
  function openBillingPortal(): void {
    const decision = requestProtectedAction($authSessionStore, {
      kind: "checkout",
      label: "Manage subscription",
      continueAfterAuth: async () => openBillingPortal()
    });
    if (!decision.allowed) {
      if (decision.reason === "expired") {
        clearAuthSession("expired");
      }
      return;
    }
    portalMutation.mutate();
  }

  /** Refetches entitlement status after a recoverable billing or return-route failure. */
  function retryEntitlement(): void {
    const decision = requestProtectedAction($authSessionStore, {
      kind: "entitlement_refresh",
      label: "Refresh billing status",
      continueAfterAuth: async () => {
        await refreshEntitlement();
      }
    });
    if (decision.reason === "expired") {
      clearAuthSession("expired");
    }
    if (decision.allowed) {
      void refreshEntitlement();
    }
  }

  /** Checks whether automatic billing entitlement refresh may call protected APIs. */
  function entitlementRefreshAllowed(): boolean {
    return buildAuthGuardDecision($authSessionStore, {
      kind: "entitlement_refresh",
      label: "Refresh billing status",
      continueAfterAuth: async () => undefined
    }).allowed;
  }

  /** Runs one entitlement refresh for explicit billing recovery paths without creating another live query. */
  async function refreshEntitlementWhenAllowed(): Promise<void> {
    if (!entitlementRefreshAllowed()) {
      return;
    }
    await refreshEntitlement();
  }

  /** Refreshes shared entitlement state through the generated billing client. */
  async function refreshEntitlement(): Promise<EntitlementStatusData | null> {
    refreshingEntitlement = true;
    try {
      const status = await fetchEntitlementStatus();
      setEntitlementStatus(status);
      return status;
    } catch (error) {
      if (error instanceof EntitlementClientError) {
        setEntitlementError(error.appError);
      }
      return null;
    } finally {
      refreshingEntitlement = false;
    }
  }

  /** Polls briefly after hosted Checkout returns because Stripe webhook delivery is asynchronous. */
  async function confirmCheckoutEntitlement(): Promise<void> {
    if (!entitlementRefreshAllowed()) {
      return;
    }
    confirmingCheckout = true;
    checkoutConfirmationPending = false;
    for (let attempt = 0; attempt < checkoutConfirmationPollAttempts; attempt += 1) {
      const status = await refreshEntitlement();
      if (status?.tier === "paid" && status.status === "active") {
        confirmingCheckout = false;
        return;
      }
      if (attempt < checkoutConfirmationPollAttempts - 1) {
        await delay(checkoutConfirmationPollIntervalMs);
      }
    }
    confirmingCheckout = false;
    checkoutConfirmationPending = true;
  }

  function delay(ms: number): Promise<void> {
    return new Promise((resolve) => window.setTimeout(resolve, ms));
  }

</script>

<!-- Implements DESIGN-007 SubscriptionController billing-state, checkout, cancellation return, and success return UI. -->
<section
  class="grid gap-4 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4"
  aria-label="Subscription billing"
  data-subscription-billing
>
  {#if status}
    <div class="w-fit rounded border border-[var(--color-border)] px-3 py-2 text-sm">
      <span class="font-[var(--font-data)] uppercase tracking-normal text-[var(--color-muted)]">Plan</span>
      <span class="ml-2 font-semibold capitalize">{status.tier} · {status.status}</span>
    </div>
  {/if}

  {#if returnState === "success"}
    <p class="rounded border border-[var(--color-primary)] bg-[var(--color-secondary)] px-3 py-2 text-sm text-[var(--color-text)]" role="status">
      {#if status?.tier === "paid" && status.status === "active"}
        Checkout completed. Billing access is active.
      {:else if checkoutConfirmationPending}
        Checkout completed. Waiting for Stripe confirmation.
      {:else if confirmingCheckout || refreshingEntitlement}
        Checkout completed. Confirming billing access.
      {:else}
        Checkout completed. Billing access will update after Stripe confirmation.
      {/if}
    </p>
  {:else if returnState === "cancel"}
    <p class="rounded border border-[var(--color-border)] px-3 py-2 text-sm text-[var(--color-muted)]" role="status">
      Checkout was cancelled. Your current entitlement is unchanged.
    </p>
  {/if}

  {#if refreshingEntitlement && !status}
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

  {#if paidActive}
    <div class="grid gap-3 rounded border border-[var(--color-border)] p-3">
      <div>
        <h3 class="text-base font-semibold text-[var(--color-text)]">Active paid subscription</h3>
        <p class="text-sm text-[var(--color-muted)]">
          Billing interval details are managed by Stripe.
        </p>
      </div>
      <button
        type="button"
        class="w-fit rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)] transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:cursor-wait disabled:opacity-70"
        disabled={portalMutation.isPending}
        aria-describedby={portalError ? portalErrorId : undefined}
        onclick={openBillingPortal}
      >
        {portalMutation.isPending ? "Opening billing portal..." : "Manage or cancel subscription"}
      </button>
    </div>
  {:else}
    <div class="grid gap-3 sm:grid-cols-2">
      {#each checkoutPlans as plan}
        <article class="flex h-full flex-col gap-3 rounded border border-[var(--color-border)] p-3">
          <div>
            <h3 class="text-base font-semibold text-[var(--color-text)]">{priceLabels[plan].title}</h3>
            <p class="font-[var(--font-data)] text-sm text-[var(--color-muted)]">{priceLabels[plan].cadence}</p>
            <p class="mt-1 text-sm text-[var(--color-muted)]">{priceLabels[plan].description}</p>
          </div>
          <button
            type="button"
            class="mt-auto rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)] transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:cursor-wait disabled:opacity-70"
            disabled={checkoutMutation.isPending}
            aria-describedby={checkoutError ? checkoutErrorId : undefined}
            onclick={() => startCheckout(plan)}
          >
            {loadingPlan === plan && !recoveryCheckout ? "Creating checkout..." : `Choose ${priceLabels[plan].title}`}
          </button>
        </article>
      {/each}
    </div>
  {/if}
  {#if portalError && !portalMutation.isPending}
    <p id={portalErrorId} class="text-sm text-[var(--color-error)]" role="alert">
      {portalError}
    </p>
  {/if}
  {#if checkoutError && !checkoutMutation.isPending}
    <p id={checkoutErrorId} class="text-sm text-[var(--color-error)]" role="alert">
      {checkoutError}
    </p>
  {/if}
</section>
