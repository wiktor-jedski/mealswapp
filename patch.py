import os

shell_path = "frontend/src/lib/components/SearchShell.svelte"
with open(shell_path, "r") as f:
    content = f.read()

content = content.replace(
    """  import { fetchFoodObject } from "../api/search-client";
  import { preferencesStore } from "../stores/preferences";
  import { displayUnitForBasis } from "../units";

  // Implements DESIGN-001 SearchView shell composition: sidebar, mode controls, autocomplete search bar, mode-specific controls, results, and offline status.""",
    """  import { fetchFoodObject } from "../api/search-client";
  import { preferencesStore } from "../stores/preferences";
  import { displayUnitForBasis } from "../units";
  import { createQuery } from "@tanstack/svelte-query";
  import { buildEntitlementQueryOptions } from "../api/entitlement-client";

  // Implements DESIGN-001 SearchView shell composition: sidebar, mode controls, autocomplete search bar, mode-specific controls, results, and offline status.
  const entitlementQuery = createQuery(buildEntitlementQueryOptions());
  let entitlement = $derived($entitlementQuery.data);"""
)

content = content.replace(
    """      <SearchModes />

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
        <SubstitutionInputs />
      {:else if activeMode === "daily_diet_alternative"}
        <DailyDietControls {rejection} />
      {/if}""",
    """      <SearchModes {entitlement} />

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
        <SubstitutionInputs {entitlement} />
      {:else if activeMode === "daily_diet_alternative"}
        <DailyDietControls {rejection} {entitlement} />
      {/if}"""
)

with open(shell_path, "w") as f:
    f.write(content)

print("Patched SearchShell.svelte")
