<script lang="ts">
  import { createQuery } from "@tanstack/svelte-query";
  import {
    searchStore,
    substitutionState,
    setMode,
    setQuery,
    submitSearch,
    addSubstitutionInput,
    setSubstitutionInputItem,
    updateSubstitutionInput
  } from "../stores/search";
  import { sidebarStore } from "../stores/sidebar";
  import type {
    DailyDiet,
    SearchMode,
    SearchRejection,
    RankedAutocomplete
  } from "../api/generated";
  import SidebarComponent from "./SidebarComponent.svelte";
  import SearchModes from "./SearchModes.svelte";
  import AutocompleteDropdown from "./AutocompleteDropdown.svelte";
  import SavedDailyDietSearch from "./SavedDailyDietSearch.svelte";
  import SubstitutionInputs from "./SubstitutionInputs.svelte";
  import DailyDietCollection, { type DailyDietEditSelection } from "./DailyDietCollection.svelte";
  import DailyDietControls from "./DailyDietControls.svelte";
  import SearchResults from "./SearchResults.svelte";
  import OfflineBanner from "./OfflineBanner.svelte";
  import SubscriptionBilling from "./SubscriptionBilling.svelte";
  import LoginView from "./LoginView.svelte";
  import OAuthEntryPoint from "./OAuthEntryPoint.svelte";
  import RegisterView from "./RegisterView.svelte";
  import AdministrationPanel from "./AdministrationPanel.svelte";
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
  import { clearDailyDietState, dailyDietStore, selectDailyDiet } from "../stores/daily-diet";
  import { parseShellRoute, searchRoute, shellViewRoute, type ShellView } from "../shell-routing";
  import { resolveAdminAccess, verifiedAdminIdentity } from "../admin-access";

  // Implements DESIGN-001 SearchView shell composition: sidebar, mode controls, entitlement gate, autocomplete search bar, mode-specific controls, results, offline status, and DESIGN-018 login auth surface.

  interface Props {
    oauthCallbackReturn?: boolean;
  }

  let { oauthCallbackReturn = false }: Props = $props();

  const startupRoute = parseShellRoute(window.location.href);
  if (startupRoute.view === "search") {
    setMode(startupRoute.mode);
  }

  /** Structured Daily Diet Alternative rejection lifted from the 422 SearchRejection envelope by SearchResults. */
  let rejection = $state<SearchRejection | null>(null);

  /** Saved Daily Diet selection queued for the editor; the key allows selecting the same diet again. */
  let dailyDietEditSelection = $state<DailyDietEditSelection | null>(null);
  let dailyDietEditSelectionKey = 0;
  let dailyDietStateUserId = $state<string | null>(null);

  /** True while an explicit submitted search request is fetching results. */
  let searchInFlight = $state(false);

  /** Mode-specific input guidance for the primary SearchView combobox. */
  const searchPlaceholders: Record<SearchMode, string> = {
    catalog: "Search foods, meals, or ingredients…",
    substitution: "Search a food to add as a substitution target…",
    daily_diet: "Search saved Daily Diets by name…",
    daily_diet_alternative: "Search saved Daily Diets by name…"
  };

  /** Active mode mirrored from the store for shell-level conditional rendering and focus keys. */
  let activeMode = $derived($searchStore.mode);

  $effect.pre(() => {
    const authenticatedUserId = $authSessionStore.status === "authenticated"
      ? $authSessionStore.userId ?? null
      : null;
    if (dailyDietStateUserId !== authenticatedUserId) {
      clearDailyDietState();
      dailyDietEditSelection = null;
      dailyDietStateUserId = authenticatedUserId;
    }
  });

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
    substitutionInputCount: $substitutionState?.substitutionInputs.length ?? 0
  }));

  /** Active email auth mode for the guarded protected-action modal. */
  let authSurfaceMode = $state<"login" | "register">("login");

  /** Active authenticated shell surface; search store state is preserved while the subscription view is open. */
  let activeView = $state<ShellView>(startupRoute.view);

  /** Guards one direct subscription-route attempt after session probing resolves. */
  let guardedInitialSubscriptionRoute = $state(false);

  /** Verified administrator identity owning feature-local administration state. */
  let administrationIdentity = $state<string | null>(null);

  /** Safe feedback after a denied direct route; never exposes protected controls or data. */
  let administrationDenied = $state(false);

  /** Administration presentation state derived only from the current server-refreshed session projection. */
  let administrationAccess = $derived(resolveAdminAccess($authSessionStore));

  /** Guards one OAuth callback modal handoff while keeping the normal application shell mounted. */
  let handledOAuthCallbackReturn = $state(false);

  // Keeps the shared entitlement stores synchronized with TanStack Query state for all SearchView controls.
  $effect(() => {
    if (entitlementQuery.data) {
      setEntitlementStatus(entitlementQuery.data);
    }
  });

  $effect(() => {
    if (activeView !== "administration" || administrationAccess === "loading" || administrationAccess === "error") return;
    const identity = verifiedAdminIdentity($authSessionStore);
    if (identity !== null && (administrationIdentity === null || administrationIdentity === identity)) {
      administrationIdentity = identity;
      return;
    }
    denyAdministrationRoute();
  });

  $effect(() => {
    if (!$authSurfaceStore.open) {
      authSurfaceMode = "login";
    }
  });

  $effect(() => {
    if (oauthCallbackReturn && !handledOAuthCallbackReturn) {
      handledOAuthCallbackReturn = true;
      authSurfaceMode = "login";
      openLoginSurface();
    }
  });

  $effect(() => {
    if (["anonymous", "expired", "locked", "error"].includes($authSessionStore.status) && activeView === "subscription") {
      activeView = "search";
      replaceBrowserRoute(searchRoute($searchStore.mode));
    }
  });

  $effect(() => {
    if (!guardedInitialSubscriptionRoute && startupRoute.view === "subscription" && !authenticating()) {
      guardedInitialSubscriptionRoute = true;
      openSubscriptionView(true);
    }
  });

  $effect(() => {
    const onPopState = () => applyBrowserRoute();
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
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
          foodObjectType: item.objectType,
          quantity: 100,
          unit: $preferencesStore.unitSystem === "imperial" ? "oz" : "g"
        },
        item.label
      );
      void hydrateSubstitutionInput(item.itemId, item.objectType);
      setQuery("");
    } else {
      setQuery(item.label);
      submitSearch(item.label);
    }
  }

  /** Opens a user-owned saved Daily Diet in the editor. */
  function editDailyDiet(diet: DailyDiet): void {
    dailyDietEditSelection = { key: ++dailyDietEditSelectionKey, diet };
  }

  /** Routes a saved-diet autocomplete choice to the active mode's editor or alternative workflow. */
  function chooseSavedDailyDiet(diet: DailyDiet): void {
    if (activeMode === "daily_diet_alternative") {
      selectDailyDiet(diet.id);
      return;
    }
    editDailyDiet(diet);
  }

  /**
   * Hydrates autocomplete-selected Substitution Inputs with rich FoodObject display data.
   * Failures are intentionally silent because the fallback label card remains usable.
   */
  async function hydrateSubstitutionInput(foodObjectId: string, foodObjectType: "food_item" | "meal"): Promise<void> {
    try {
      const item = await fetchFoodObject(foodObjectId, new AbortController().signal, foodObjectType);
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
    if (activeMode !== "substitution" && activeMode !== "daily_diet" && entitlementDecision.canExecute) {
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

  /** True while startup auth probing or an auth mutation is still resolving. */
  function authenticating(): boolean {
    return $authSessionStore.status === "unknown" || $authSessionStore.status === "authenticating";
  }

  /** Returns to the primary SearchView without clearing query, mode, inputs, or results cache state. */
  function openSearchView(): void {
    activeView = "search";
    pushBrowserRoute(searchRoute($searchStore.mode));
  }

  /** Opens the placeholder Privacy Policy view without clearing SearchView state. */
  function openPrivacyView(): void {
    activeView = "privacy";
    pushBrowserRoute(shellViewRoute("privacy"));
  }

  /** Opens the placeholder Terms of Service view without clearing SearchView state. */
  function openTermsView(): void {
    activeView = "terms";
    pushBrowserRoute(shellViewRoute("terms"));
  }

  /** Opens the authenticated Subscription view only after DESIGN-018 guard approval. */
  function openSubscriptionView(preserveCurrentURL = false): void {
    const decision = requestProtectedAction($authSessionStore, {
      kind: "account",
      label: "Open Subscription",
      continueAfterAuth: async () => {
        activeView = "subscription";
        if (!preserveCurrentURL) pushBrowserRoute(shellViewRoute("subscription"));
      }
    });
    if (decision.reason === "expired") {
      clearAuthSession("expired");
    }
    if (decision.allowed) {
      activeView = "subscription";
      if (!preserveCurrentURL) pushBrowserRoute(shellViewRoute("subscription"));
    }
  }

  /** Opens the administration shell only for a verified admin projection; server authorization remains authoritative. */
  function openAdministrationView(preserveCurrentURL = false): void {
    if (administrationAccess !== "allowed") {
      if (administrationAccess !== "loading" && administrationAccess !== "error") denyAdministrationRoute();
      return;
    }
    administrationDenied = false;
    administrationIdentity = verifiedAdminIdentity($authSessionStore);
    activeView = "administration";
    if (!preserveCurrentURL) pushBrowserRoute(shellViewRoute("administration"));
  }

  /** Fails a denied or changed-account administration route closed without clearing Search state. */
  function denyAdministrationRoute(): void {
    administrationIdentity = null;
    administrationDenied = true;
    activeView = "search";
    replaceBrowserRoute(searchRoute($searchStore.mode));
  }

  /** Returns an imported item to ordinary local catalog search without retaining administration draft state. */
  function viewImportedItemInLocalSearch(name: string): void {
    setMode("catalog");
    setQuery(name);
    submitSearch(name);
    activeView = "search";
    pushBrowserRoute(searchRoute("catalog"));
  }

  /** Selects a Search mode and records it for refresh and Back/Forward restoration. */
  function selectSearchMode(mode: SearchMode): void {
    setMode(mode);
    activeView = "search";
    pushBrowserRoute(searchRoute(mode));
  }

  /** Restores the shell surface and Search mode from browser Back/Forward navigation. */
  function applyBrowserRoute(): void {
    const route = parseShellRoute(window.location.href);
    if (route.view === "search") {
      setMode(route.mode);
      activeView = "search";
      return;
    }
    activeView = route.view;
    if (route.view === "subscription" && !authenticating()) {
      openSubscriptionView(true);
    } else if (route.view === "administration" && administrationAccess !== "loading") {
      openAdministrationView(true);
    }
  }

  function pushBrowserRoute(route: string): void {
    if (`${window.location.pathname}${window.location.search}` !== route) {
      window.history.pushState(null, "", route);
    }
  }

  function replaceBrowserRoute(route: string): void {
    if (`${window.location.pathname}${window.location.search}` !== route) {
      window.history.replaceState(null, "", route);
    }
  }

  /** Ends the current cookie-backed session and returns protected UI to anonymous search mode. */
  async function signOut(): Promise<void> {
    try {
      await logoutCurrentSession();
      openSearchView();
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
        onNavigateAdministration={openAdministrationView}
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
      {#if activeView === "administration"}
        <!-- Implements DESIGN-009 UserAdminPanel route-level fail-closed administration boundary. -->
        {#if administrationAccess !== "denied"}
          <AdministrationPanel access={administrationAccess} onViewLocalItem={viewImportedItemInLocalSearch} />
        {/if}
      {:else if activeView === "subscription"}
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
        <!-- Implements DESIGN-016 ComponentStyles legal content view and DESIGN-015 DisclaimerRenderer placement. -->
        <section class="grid max-w-3xl gap-4" aria-labelledby="terms-view-title" data-terms-view>
          <h1 id="terms-view-title" class="text-2xl font-semibold text-[var(--color-text)]">
            Terms of Service
          </h1>
          <p class="text-sm leading-6 text-[var(--color-muted)]">
            Terms of Service placeholder text. Final legal content will be added before production release.
          </p>
          <h2 class="text-lg font-semibold text-[var(--color-text)]">Medical information</h2>
          <p class="text-sm leading-6 text-[var(--color-muted)]" data-medical-disclaimer>
            Mealswapp provides general food and nutrition information. It does not provide medical advice,
            diagnosis, or treatment. Consult a qualified healthcare professional for guidance about your health.
          </p>
        </section>
      {:else}
        {#if administrationDenied}
          <p class="rounded border border-[var(--color-error)] bg-[var(--color-surface)] px-3 py-2 text-sm" role="alert" data-admin-access-denied>
            Administration access is unavailable for this session.
          </p>
        {/if}
        <!-- Visual order: mode controls → autocomplete search bar → mode-specific controls → results → offline status. -->
        <SearchModes onModeChange={selectSearchMode} />

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

        {#if activeMode === "daily_diet" || activeMode === "daily_diet_alternative"}
          <SavedDailyDietSearch
            diets={$dailyDietStore.collections}
            loading={$dailyDietStore.status === "loading"}
            focusKey={activeMode}
            onSelect={chooseSavedDailyDiet}
          />
        {:else}
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
        {/if}

        {#if activeMode === "substitution"}
          <SubstitutionInputs executionAllowed={entitlementDecision.canExecute} entitlementFeedback={entitlementDecision.feedback} />
        {:else if activeMode === "daily_diet"}
          <DailyDietCollection
            authStatus={$authSessionStore.status}
            authenticated={$authSessionStore.status === "authenticated" && $authSessionStore.hasVerifiedLoginMethod === true}
            userId={$authSessionStore.userId ?? null}
            executionAllowed={entitlementDecision.canExecute}
            entitlementFeedback={entitlementDecision.feedback}
            selectedDiet={dailyDietEditSelection}
            onEditDiet={editDailyDiet}
            onSignIn={() => {
              requestProtectedAction($authSessionStore, {
                kind: "saved_data",
                label: "build a Daily Diet",
                continueAfterAuth: async () => undefined
              });
              authSurfaceMode = "login";
            }}
          />
        {:else if activeMode === "daily_diet_alternative"}
          <DailyDietControls
            {rejection}
            authStatus={$authSessionStore.status}
            authenticated={$authSessionStore.status === "authenticated" && $authSessionStore.hasVerifiedLoginMethod === true}
            userId={$authSessionStore.userId ?? null}
            executionAllowed={entitlementDecision.canExecute}
            entitlementFeedback={entitlementDecision.feedback}
            onSignIn={() => {
              requestProtectedAction($authSessionStore, {
                kind: "saved_data",
                label: "use a saved Daily Diet",
                continueAfterAuth: async () => undefined
              });
              authSurfaceMode = "login";
            }}
          />
        {/if}

        {#if activeMode !== "daily_diet"}
          <SearchResults
            searchEnabled={entitlementDecision.canExecute}
            onRejection={(r) => (rejection = r)}
            onSearchInFlightChange={(searching) => (searchInFlight = searching)}
          />
        {/if}

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
        <OAuthEntryPoint mode={authSurfaceMode} callbackReturn={oauthCallbackReturn} />
        {#if authSurfaceMode === "login"}
          <LoginView />
        {:else}
          <RegisterView
            onRegistered={(session) => {
              if (session.hasVerifiedLoginMethod === true) {
                void runQueuedProtectedActionAfterAuth();
              }
            }}
            onSwitchToLogin={() => (authSurfaceMode = "login")}
          />
        {/if}
      </div>
    </div>
  {/if}
</main>
