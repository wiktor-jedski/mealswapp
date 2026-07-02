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
  import { buildEntitlementQueryOptions, EntitlementClientError } from "../api/entitlement-client";
  import { fetchFoodObject } from "../api/search-client";
  import { entitlementErrorStore, entitlementStatusStore, setEntitlementError, setEntitlementStatus } from "../stores/entitlement";
  import { preferencesStore } from "../stores/preferences";
  import { resolveSearchEntitlement } from "../search-entitlement";
  import { displayUnitForBasis } from "../units";

  // Implements DESIGN-001 SearchView shell composition: sidebar, mode controls, entitlement gate, autocomplete search bar, mode-specific controls, results, and offline status.

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
  const entitlementQuery = createQuery(buildEntitlementQueryOptions);

  /** Entitlement gate decision for visible feedback and request execution. */
  let entitlementDecision = $derived(resolveSearchEntitlement({
    status: $entitlementStatusStore,
    error: $entitlementErrorStore,
    mode: activeMode,
    substitutionInputCount: $searchStore.substitutionInputs.length
  }));

  // Keeps the shared entitlement stores synchronized with TanStack Query state for all SearchView controls.
  $effect(() => {
    if (entitlementQuery.data) {
      setEntitlementStatus(entitlementQuery.data);
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
</script>

<!-- Implements DESIGN-001 SearchView, SidebarComponent, and DESIGN-016 LayoutGrid (viewport-left sidebar, centered content below 1280px). -->
<main class="min-h-screen">
  <!-- Implements DESIGN-016 LayoutGrid: full-width grid above 640px so SidebarComponent sits on the viewport's far-left edge. -->
  <section class="grid min-h-screen content-start gap-6 px-4 py-6 transition-[grid-template-columns] duration-200 ease-out motion-reduce:transition-none sm:px-0 sm:py-0 {$sidebarStore.collapsed ? 'sm:grid-cols-[3.5rem_minmax(0,1fr)]' : 'sm:grid-cols-[15rem_minmax(0,1fr)]'}">
    <!-- Implements DESIGN-001 SidebarComponent placed in the viewport-left grid column. -->
    <aside>
      <SidebarComponent />
    </aside>

    <div class="flex w-full max-w-5xl flex-col gap-5 sm:mx-auto sm:px-6 sm:py-6">
      <!-- Visual order: mode controls → autocomplete search bar → mode-specific controls → results → offline status. -->
      <!-- Implements DESIGN-007 SubscriptionController billing controls stay outside search gating ownership. -->
      <SubscriptionBilling />

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
    </div>
  </section>
</main>
