<script lang="ts">
  import { createQuery } from "@tanstack/svelte-query";
  import {
    searchStore,
    setQuery,
    submitSearch,
    addSubstitutionInput,
    setSubstitutionInputItem,
    updateSubstitutionInput
  } from "../stores/search";
  import { sidebarStore } from "../stores/sidebar";
  import type {
    SearchMode,
    SearchRejection,
    RankedAutocomplete
  } from "../api/generated";
  import SidebarComponent from "./SidebarComponent.svelte";
  import SearchModes from "./SearchModes.svelte";
  import AutocompleteDropdown from "./AutocompleteDropdown.svelte";
  import SubstitutionInputs from "./SubstitutionInputs.svelte";
  import DailyDietControls from "./DailyDietControls.svelte";
  import SearchResults from "./SearchResults.svelte";
  import OfflineBanner from "./OfflineBanner.svelte";
  import SubscriptionBilling from "./SubscriptionBilling.svelte";
  import LoginView from "./LoginView.svelte";
  import OAuthEntryPoint from "./OAuthEntryPoint.svelte";
  import RegisterView from "./RegisterView.svelte";
  import {
    authSurfaceStore,
    buildAuthGuardDecision,
    closeAuthSurface,
    openLoginSurface,
    requestProtectedAction,
    runQueuedProtectedActionAfterAuth
  } from "../stores/auth-surface";
  import { authSessionStore, clearAuthSession, logoutCurrentSession } from "../stores/auth-session";
  import { buildEntitlementQueryOptions, EntitlementClientError } from "../api/entitlement-client";
  import { fetchFoodObject } from "../api/search-client";
  import { entitlementErrorStore, entitlementStatusStore, setEntitlementError, setEntitlementStatus } from "../stores/entitlement";
  import { preferencesStore } from "../stores/preferences";
  import { resolveSearchEntitlement } from "../search-entitlement";
  import { displayUnitForBasis } from "../units";

  // Implements DESIGN-001 SearchView shell composition: sidebar, mode controls, entitlement gate, autocomplete search bar, mode-specific controls, results, offline status, and DESIGN-018 login auth surface.

  type ShellView = "search" | "subscription" | "privacy" | "terms";

  /** Structured Daily Diet Alternative rejection lifted from the 422 SearchRejection envelope by SearchResults. */
  let rejection = $state<SearchRejection | null>(null);

  /** True while an explicit submitted search request is fetching results. */
  let searchInFlight = $state(false);

  /** Mode-specific input guidance for the primary SearchView combobox. */
  const searchPlaceholders: Record<SearchMode, string> = {
    catalog: "Search foods, meals, or ingredients…",
    substitution: "Search a food to add as a substitution target…",
    daily_diet: "Search saved daily diets…",
    daily_diet_alternative: "Search within a saved daily diet or paste its ID…"
  };

  /** Active mode mirrored from the store for shell-level conditional rendering and focus keys. */
  let activeMode = $derived($searchStore.mode);

  /** Current-user entitlement query resolved through the generated billing client. */
  const entitlementQuery = createQuery(() => ({
    ...buildEntitlementQueryOptions(),
    enabled: entitlementRefreshAllowed()
  }));

  /** Entitlement gate decision for visible feedback and request execution. */
  let entitlementDecision = $derived(resolveSearchEntitlement({
    status: $entitlementStatusStore,
    error: $entitlementErrorStore,
    mode: activeMode,
    substitutionInputCount: $searchStore.substitutionInputs.length
  }));

  /** Active email auth mode for the guarded protected-action modal. */
  let authSurfaceMode = $state<"login" | "register">("login");

  /** Active authenticated shell surface; search store state is preserved while the subscription view is open. */
  let activeView = $state<ShellView>(initialShellView());

  /** Guards one direct subscription-route attempt after session probing resolves. */
  let guardedInitialSubscriptionRoute = $state(false);

  // Keeps the shared entitlement stores synchronized with TanStack Query state for all SearchView controls.
  $effect(() => {
    if (entitlementQuery.data) {
      setEntitlementStatus(entitlementQuery.data);
    }
  });

  $effect(() => {
    if (!$authSurfaceStore.open) {
      authSurfaceMode = "login";
    }
  });

  $effect(() => {
    if ($authSessionStore.status !== "authenticated" && activeView === "subscription") {
      activeView = "search";
    }
  });

  $effect(() => {
    if (!guardedInitialSubscriptionRoute && initialShellView() === "subscription" && !authenticating()) {
      guardedInitialSubscriptionRoute = true;
      openSubscriptionView();
    }
  });

  // Anonymous entitlement failures remain recoverable so Catalog Search can continue without a session.
  $effect(() => {
    if (entitlementQuery.error instanceof EntitlementClientError) {
      setEntitlementError(entitlementQuery.error.appError);
    }
  });

  /**
   * Handles autocomplete selection: in Substitution mode adds a Substitution Input from the
   * suggestion's food object id; otherwise commits the selected suggestion label as the search.
   */
  function onAutocompleteSelect(item: RankedAutocomplete): void {
    if (activeMode === "substitution") {
      addSubstitutionInput(
        {
          foodObjectId: item.itemId,
          quantity: 100,
          unit: $preferencesStore.unitSystem === "imperial" ? "oz" : "g"
        },
        item.label
      );
      void hydrateSubstitutionInput(item.itemId);
      setQuery("");
    } else {
      setQuery(item.label);
      submitSearch(item.label);
    }
  }

  /**
   * Hydrates autocomplete-selected Substitution Inputs with rich FoodObject display data.
   * Failures are intentionally silent because the fallback label card remains usable.
   */
  async function hydrateSubstitutionInput(foodObjectId: string): Promise<void> {
    try {
      const item = await fetchFoodObject(foodObjectId, new AbortController().signal);
      setSubstitutionInputItem(item);
      updateSubstitutionInput(foodObjectId, {
        unit: displayUnitForBasis(item.macroBasis, $preferencesStore.unitSystem)
      });
    } catch {
      // Implements DESIGN-001 SearchView resilient selected-item hydration fallback.
      return;
    }
  }

  /** Commits typed text only for result-searching modes; Substitution uses autocomplete as an item picker. */
  function onAutocompleteSubmit(query: string): void {
    if (activeMode !== "substitution" && entitlementDecision.canExecute) {
      submitSearch(query);
    }
  }

  /** Checks whether authenticated entitlement refresh may make protected billing calls. */
  function entitlementRefreshAllowed(): boolean {
    return buildAuthGuardDecision($authSessionStore, {
      kind: "entitlement_refresh",
      label: "Refresh billing access",
      continueAfterAuth: async () => undefined
    }).allowed;
  }

  /** Chooses the billing surface for hosted-checkout return routes and explicit subscription links. */
  function initialShellView(): ShellView {
    const path = window.location.pathname;
    if (path === "/privacy") {
      return "privacy";
    }
    if (path === "/terms") {
      return "terms";
    }
    return path === "/subscription" || path === "/billing/success" || path === "/billing/cancel"
      ? "subscription"
      : "search";
  }

  /** True while startup auth probing or an auth mutation is still resolving. */
  function authenticating(): boolean {
    return $authSessionStore.status === "unknown" || $authSessionStore.status === "authenticating";
  }

  /** Returns to the primary SearchView without clearing query, mode, inputs, or results cache state. */
  function openSearchView(): void {
    activeView = "search";
  }

  /** Opens the placeholder Privacy Policy view without clearing SearchView state. */
  function openPrivacyView(): void {
    activeView = "privacy";
  }

  /** Opens the placeholder Terms of Service view without clearing SearchView state. */
  function openTermsView(): void {
    activeView = "terms";
  }

  /** Opens the authenticated Subscription view only after DESIGN-018 guard approval. */
  function openSubscriptionView(): void {
    const decision = requestProtectedAction($authSessionStore, {
      kind: "account",
      label: "Open Subscription",
      continueAfterAuth: async () => {
        activeView = "subscription";
      }
    });
    if (decision.reason === "expired") {
      clearAuthSession("expired");
    }
    if (decision.allowed) {
      activeView = "subscription";
    }
  }

  /** Ends the current cookie-backed session and returns protected UI to anonymous search mode. */
  async function signOut(): Promise<void> {
    try {
      await logoutCurrentSession();
      activeView = "search";
    } catch {
      // Logout failures are already reflected through the DESIGN-018 AuthSessionStore error projection.
    }
  }

</script>

<!-- Implements DESIGN-001 SearchView, SidebarComponent, and DESIGN-016 LayoutGrid (viewport-left sidebar, centered content below 1280px). -->
<main class="min-h-screen">
  <!-- Implements DESIGN-016 LayoutGrid: full-width grid above 640px so SidebarComponent sits on the viewport's far-left edge. -->
  <section class="grid min-h-screen content-start gap-6 px-4 py-6 transition-[grid-template-columns] duration-200 ease-out motion-reduce:transition-none sm:px-0 sm:py-0 {$sidebarStore.collapsed ? 'sm:grid-cols-[3.5rem_minmax(0,1fr)]' : 'sm:grid-cols-[15rem_minmax(0,1fr)]'}">
    <!-- Implements DESIGN-001 SidebarComponent placed in the viewport-left grid column. -->
    <aside>
      <SidebarComponent
        {activeView}
        onNavigateSearch={openSearchView}
        onNavigateSubscription={openSubscriptionView}
        onNavigatePrivacy={openPrivacyView}
        onNavigateTerms={openTermsView}
        onSignIn={() => {
          authSurfaceMode = "login";
          openLoginSurface();
        }}
        onSignOut={() => void signOut()}
      />
    </aside>

    <div class="flex w-full max-w-5xl flex-col gap-5 sm:mx-auto sm:px-6 sm:py-6">
      {#if activeView === "subscription"}
        <!-- Implements DESIGN-001 SidebarComponent authenticated Subscription navigation target, guarded by DESIGN-018. -->
        <section class="grid gap-4" data-subscription-view>
          <SubscriptionBilling />
        </section>
      {:else if activeView === "privacy"}
        <!-- Implements DESIGN-016 ComponentStyles placeholder legal content view. -->
        <section class="grid max-w-3xl gap-4" aria-labelledby="privacy-view-title" data-privacy-view>
          <h1 id="privacy-view-title" class="text-2xl font-semibold text-[var(--color-text)]">
            Privacy Policy
          </h1>
          <p class="text-sm leading-6 text-[var(--color-muted)]">
            Privacy Policy placeholder text. Final legal content will be added before production release.
          </p>
        </section>
      {:else if activeView === "terms"}
        <!-- Implements DESIGN-016 ComponentStyles placeholder legal content view. -->
        <section class="grid max-w-3xl gap-4" aria-labelledby="terms-view-title" data-terms-view>
          <h1 id="terms-view-title" class="text-2xl font-semibold text-[var(--color-text)]">
            Terms of Service
          </h1>
          <p class="text-sm leading-6 text-[var(--color-muted)]">
            Terms of Service placeholder text. Final legal content will be added before production release.
          </p>
        </section>
      {:else}
        <!-- Visual order: mode controls → autocomplete search bar → mode-specific controls → results → offline status. -->
        <SearchModes />

        {#if entitlementDecision.usageText}
          <p class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 font-data text-sm text-[var(--color-muted)]" role="status" data-entitlement-usage>
            {entitlementDecision.usageText}
          </p>
        {/if}

        {#if entitlementDecision.feedback}
          <div class="rounded border border-[var(--color-accent)] bg-[var(--color-surface)] px-3 py-2 text-sm" role="alert" data-entitlement-feedback>
            {entitlementDecision.feedback}
          </div>
        {/if}

        <AutocompleteDropdown
          query={$searchStore.query}
          placeholder={searchPlaceholders[activeMode]}
          focusKey={activeMode}
          searching={searchInFlight}
          selectFirstOnEnter={activeMode === "substitution"}
          onQueryInput={setQuery}
          onSubmit={onAutocompleteSubmit}
          onSelect={onAutocompleteSelect}
        />

        {#if activeMode === "substitution"}
          <SubstitutionInputs executionAllowed={entitlementDecision.canExecute} entitlementFeedback={entitlementDecision.feedback} />
        {:else if activeMode === "daily_diet_alternative"}
          <DailyDietControls {rejection} executionAllowed={entitlementDecision.canExecute} />
        {/if}

        <SearchResults
          searchEnabled={entitlementDecision.canExecute}
          onRejection={(r) => (rejection = r)}
          onSearchInFlightChange={(searching) => (searchInFlight = searching)}
        />

        <OfflineBanner />
      {/if}
    </div>
  </section>

  {#if $authSurfaceStore.open}
    <!-- Implements DESIGN-018 AuthView sign-in/register guidance that preserves SearchView state while open or closed. -->
    <div
      class="fixed inset-0 z-50 grid place-items-center bg-black/45 px-4 py-6"
      role="presentation"
      onclick={closeAuthSurface}
      data-auth-surface
    >
      <div
        class="grid max-h-[calc(100vh-3rem)] w-full max-w-md gap-4 overflow-y-auto rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4 shadow-lg"
        role="dialog"
        aria-modal="true"
        aria-labelledby={authSurfaceMode === "login" ? "login-view-title" : "register-title"}
        tabindex="-1"
        onclick={(event) => event.stopPropagation()}
        onkeydown={(event) => event.stopPropagation()}
      >
        {#if $authSurfaceStore.pendingAction}
          <p class="rounded border border-[var(--color-border)] px-3 py-2 text-sm text-[var(--color-muted)]" role="status" data-auth-guidance>
            Sign in or create an account to {$authSurfaceStore.pendingAction.label.toLowerCase()}.
          </p>
        {/if}
        <div class="grid grid-cols-2 gap-2" role="group" aria-label="Authentication mode">
          <button
            type="button"
            class="rounded border px-3 py-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
            class:border-[var(--color-primary)]={authSurfaceMode === "login"}
            aria-pressed={authSurfaceMode === "login"}
            onclick={() => (authSurfaceMode = "login")}
          >
            Sign in
          </button>
          <button
            type="button"
            class="rounded border px-3 py-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
            class:border-[var(--color-primary)]={authSurfaceMode === "register"}
            aria-pressed={authSurfaceMode === "register"}
            onclick={() => (authSurfaceMode = "register")}
          >
            Create account
          </button>
        </div>
        <OAuthEntryPoint mode={authSurfaceMode} />
        {#if authSurfaceMode === "login"}
          <LoginView />
        {:else}
          <RegisterView
            onRegistered={() => {
              void runQueuedProtectedActionAfterAuth();
            }}
            onSwitchToLogin={() => (authSurfaceMode = "login")}
          />
        {/if}
      </div>
    </div>
  {/if}
</main>
