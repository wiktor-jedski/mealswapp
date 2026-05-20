<script lang="ts">
  import { createApiClient } from '../api/client';
  import type { DietOptimizationRequest, Entitlement, FoodItemViewModel, MacroValues } from '../api/types';
  import AutocompleteDropdown from './AutocompleteDropdown.svelte';
  import OptimizationPanel from './OptimizationPanel.svelte';
  import ResultsGrid from './ResultsGrid.svelte';
  import SettingsPanel from './SettingsPanel.svelte';
  import SidebarComponent from './SidebarComponent.svelte';
  import { createAutocompleteController, createAutocompleteState, type AutocompleteState } from '../search/autocompleteState';
  import { checkoutReturnShouldRefresh, createEntitlementViewState, defaultFreeEntitlement, type EntitlementUsage } from '../entitlements/entitlementState';
  import { createSearchController, createDefaultSearchState, type SearchState } from '../search/searchState';
  import { createDefaultOptimizationState, createOptimizationController, type OptimizationState } from '../search/optimizationState';
  import { LocalStorageManager } from '../storage/localStorageManager';

  let search: SearchState = $state(createDefaultSearchState());
  let autocomplete: AutocompleteState = $state(createAutocompleteState());
  let optimization: OptimizationState = $state(createDefaultOptimizationState());
  let entitlement: Entitlement = $state(defaultFreeEntitlement);
  let entitlementUsage: EntitlementUsage = $state({ searchesUsed: 0 });
  let settingsOpen = $state(false);
  let entitlementView = $derived(createEntitlementViewState(entitlement, entitlementUsage));
  const api = createApiClient();
  const controller = createSearchController({ api, localStorageManager: new LocalStorageManager() });
  const autocompleteController = createAutocompleteController({
    api,
    onSelect: (option) => {
      controller.setQuery(option.label);
    }
  });
  const optimizationController = createOptimizationController({ api });
  controller.subscribe((next) => {
    search = next;
  });
  autocompleteController.subscribe((next) => {
    autocomplete = next;
  });
  optimizationController.subscribe((next) => {
    optimization = next;
  });
  function submitOptimization() {
    void optimizationController.submit(buildOptimizationRequest(search.response?.items ?? []));
  }
  function buildOptimizationRequest(items: FoodItemViewModel[]): DietOptimizationRequest {
    const meals = items.slice(0, 5).map((item) => {
      const quantity = item.matchingQuantity ?? 100;
      const multiplier = quantity / 100;
      return {
        id: item.id,
        name: item.name,
        quantity,
        macros: scaleMacroValues(item.macros, multiplier),
        calories: Math.round((item.calories ?? 0) * multiplier)
      };
    });
    const targetMacros = meals.reduce<MacroValues>(
      (total, meal) => ({
        protein: total.protein + (meal.macros?.protein ?? 0),
        carbs: total.carbs + (meal.macros?.carbs ?? 0),
        fat: total.fat + (meal.macros?.fat ?? 0)
      }),
      { protein: 0, carbs: 0, fat: 0 }
    );
    return {
      originalMeals: meals,
      targetMacros: {
        protein: Math.max(1, Math.round(targetMacros.protein)),
        carbs: Math.max(1, Math.round(targetMacros.carbs)),
        fat: Math.max(1, Math.round(targetMacros.fat))
      },
      excludedIds: meals.map((meal) => meal.id),
      tolerancePercent: 10
    };
  }
  function scaleMacroValues(macros: MacroValues, multiplier: number): MacroValues {
    return {
      protein: Math.round(macros.protein * multiplier * 10) / 10,
      carbs: Math.round(macros.carbs * multiplier * 10) / 10,
      fat: Math.round(macros.fat * multiplier * 10) / 10
    };
  }
  async function refreshEntitlement() {
    try {
      entitlement = await api.getEntitlement();
    } catch {
      entitlement = defaultFreeEntitlement;
    }
  }
  if (typeof window !== 'undefined' && checkoutReturnShouldRefresh(window.location.href)) {
    void refreshEntitlement();
  } else {
    void refreshEntitlement();
  }
</script>

<main class="min-h-screen bg-background text-text-primary">
  <div class="mx-auto grid max-w-app grid-cols-1 gap-4 px-4 py-6 sm:grid-cols-12">
      <SidebarComponent
      mode={search.mode}
      enabledMacros={search.enabledMacros}
      filters={search.filters}
      modeGates={entitlementView.modeGates}
      usageLabel={entitlementView.usageLabel}
      upgradePrompt={entitlementView.upgradePrompt}
      onModeChange={(mode) => {
        if (!entitlementView.modeGates[mode]?.locked) {
          controller.setMode(mode);
        }
      }}
      onMacroChange={(macro, enabled) => controller.setMacro(macro, enabled)}
      onFiltersChange={(filters) => controller.setFilters(filters)}
      onAction={(action) => {
        if (action === 'settings' || action === 'profile') {
          settingsOpen = true;
        }
      }}
    />

    <section class="sm:col-span-9" aria-labelledby="search-heading">
      <h1 id="search-heading" class="text-2xl font-bold">Mealswapp</h1>
      <div class="mt-4 rounded border border-secondary bg-surface p-4">
        <label for="search" class="font-mono text-sm font-medium text-text-muted">Search food</label>
        <input
          id="search"
          class="mt-2 w-full rounded border border-secondary bg-surface px-3 py-2 text-text-primary outline-none focus:border-primary"
          placeholder="Start with a food item"
          type="search"
          value={search.query}
          aria-autocomplete="list"
          aria-controls="search-autocomplete"
          aria-activedescendant={autocomplete.selectedIndex >= 0 ? `search-autocomplete-option-${autocomplete.selectedIndex}` : undefined}
          oninput={(event) => {
            controller.setQuery(event.currentTarget.value);
            autocompleteController.setQuery(event.currentTarget.value);
          }}
          onkeydown={(event) => {
            if (autocompleteController.handleKey(event.key, event.shiftKey)) {
              event.preventDefault();
            }
          }}
          onblur={() => autocompleteController.blur()}
        />
        <AutocompleteDropdown
          id="search-autocomplete"
          query={autocomplete.query}
          options={autocomplete.options}
          selectedIndex={autocomplete.selectedIndex}
          isOpen={autocomplete.isOpen}
          isLoading={autocomplete.isLoading}
          errorMessage={autocomplete.error?.message}
          onHover={(index) => autocompleteController.hover(index)}
          onSelect={(index) => autocompleteController.select(index)}
        />
      </div>

      <ResultsGrid
        response={search.response}
        status={search.status}
        errorMessage={search.error?.message}
        onPageChange={(page) => controller.setPage(page)}
      />
      {#if search.mode === 'diet'}
        <OptimizationPanel
          state={optimization}
          onSubmit={submitOptimization}
          onCancel={() => optimizationController.cancel()}
        />
      {/if}
      <SettingsPanel
        open={settingsOpen}
        onClose={() => {
          settingsOpen = false;
        }}
        onSavePreferences={() => {
          settingsOpen = false;
        }}
      />
    </section>
  </div>
</main>
