<script lang="ts">
  import { onMount } from "svelte";
  import { searchStore, substitutionState, removeSubstitutionInput, requestSubstitutionSearch, setFilters, updateSubstitutionInput } from "../stores/search";
  import type { ClassificationSummary, FilterOption, FoodObject, SearchFilter, SearchFilterKind, SubstitutionInput, SubstitutionUnit } from "../api/generated";
  import { fetchSubstitutionFilterOptions } from "../api/filter-options-client";
  import { substitutionFilterOptions, type SubstitutionFilterOption } from "../substitution-filter-options";
  import { preferencesStore } from "../stores/preferences";
  import type { UnitSystem } from "../stores/preferences";
  import {
    convertQuantity,
    displayUnitForBasis,
    macroBasisDisplayLabel,
    normalizeDisplayQuantity,
    unitOptionsForBasis
  } from "../units";

  // Implements DESIGN-001 SearchView two-step Substitution Input composition (selected items, quantities, units, and explicit search).

  /**
   * Entitlement-controlled execution state. Composition remains editable even when the final
   * Substitution request is blocked.
   *
   * @remarks Implements DESIGN-001 SearchView Substitution entitlement execution gate.
   */
  let {
    executionAllowed = true,
    entitlementFeedback = null
  }: {
    executionAllowed?: boolean;
    entitlementFeedback?: string | null;
  } = $props();

  let includeFilterQuery = $state("");
  let excludeFilterQuery = $state("");
  let includeFilterOpen = $state(false);
  let excludeFilterOpen = $state(false);

  let backendFilterOptions = $state<FilterOption[]>([]);
  let filterOptionsStatus = $state<"loading" | "ready" | "empty" | "error">("loading");
  let filterOptionsRequest = 0;
  let filterOptionsAbort: AbortController | null = null;

  let selectedItems = $derived(Object.values($substitutionState?.substitutionInputItems ?? {}));
  let includeFilterOptions = $derived(substitutionFilterOptions(backendFilterOptions, selectedItems, true));
  let excludeFilterOptions = $derived(substitutionFilterOptions(backendFilterOptions, selectedItems, false));
  let visibleIncludeOptions = $derived(visibleFilterOptions(includeFilterOptions, includeFilterQuery));
  let visibleExcludeOptions = $derived(visibleFilterOptions(excludeFilterOptions, excludeFilterQuery));
  let activeIncludeFilters = $derived($searchStore.filters.filter((filter) => filter.include));
  let activeExcludeFilters = $derived($searchStore.filters.filter((filter) => !filter.include));
  $effect(() => {
    synchronizeInputUnits(
      $preferencesStore.unitSystem,
      $substitutionState?.substitutionInputs ?? [],
      $substitutionState?.substitutionInputItems ?? {}
    );
  });

  onMount(() => {
    void loadFilterOptions();
    const refresh = () => void loadFilterOptions(true);
    window.addEventListener("focus", refresh);
    return () => {
      window.removeEventListener("focus", refresh);
      filterOptionsAbort?.abort();
    };
  });

  /** Refreshes backend policy; sequence + abort guards prevent stale responses winning. */
  async function loadFilterOptions(retainCurrent = false): Promise<void> {
    const request = ++filterOptionsRequest;
    filterOptionsAbort?.abort();
    const controller = new AbortController();
    filterOptionsAbort = controller;
    if (!retainCurrent || backendFilterOptions.length === 0) filterOptionsStatus = "loading";
    try {
      const options = await fetchSubstitutionFilterOptions(controller.signal);
      if (request !== filterOptionsRequest) return;
      backendFilterOptions = options;
      filterOptionsStatus = options.length === 0 ? "empty" : "ready";
    } catch {
      if (request !== filterOptionsRequest || controller.signal.aborted) return;
      filterOptionsStatus = "error";
    }
  }

  /** Guards NaN/empty quantity edits so only finite values reach the store. */
  function onRowQuantityInput(foodObjectId: string, event: Event): void {
    const next = Number((event.currentTarget as HTMLInputElement).value);
    if (Number.isFinite(next)) {
      updateSubstitutionInput(foodObjectId, { quantity: next });
    }
  }

  function onRowUnitChange(foodObjectId: string, event: Event): void {
    updateSubstitutionInput(foodObjectId, {
      unit: (event.currentTarget as HTMLSelectElement).value as SubstitutionUnit
    });
  }

  function onFilterOptionMouseDown(option: SubstitutionFilterOption, event: MouseEvent): void {
    event.preventDefault();
    addSubstitutionFilter(option);
  }

  function onFilterOptionKeydown(option: SubstitutionFilterOption, event: KeyboardEvent): void {
    if (event.key !== "Enter" && event.key !== " ") return;
    event.preventDefault();
    addSubstitutionFilter(option);
  }

  /** Human-facing label for a selected substitution input; falls back to id only for legacy stored rows. */
  function inputLabel(foodObjectId: string): string {
    return $substitutionState?.substitutionInputLabels[foodObjectId] ?? foodObjectId;
  }

  /** Placeholder initial for selected-item cards; autocomplete currently supplies label/id only. */
  function inputInitial(foodObjectId: string): string {
    return inputLabel(foodObjectId).charAt(0).toUpperCase() || "?";
  }

  function inputItem(foodObjectId: string): FoodObject | undefined {
    return $substitutionState?.substitutionInputItems[foodObjectId];
  }

  function foodCategories(item: FoodObject): ClassificationSummary[] {
    return item.classifications.filter((classification) => classification.kind === "food_category");
  }

  function macroBasisLabel(item: FoodObject): string {
    return macroBasisDisplayLabel(item.macroBasis, $preferencesStore.unitSystem);
  }

  function rowUnitOptions(item: FoodObject | undefined, unitSystem: UnitSystem): { value: SubstitutionUnit; label: string }[] {
    return unitOptionsForBasis(item?.macroBasis ?? "100g", unitSystem);
  }

  function synchronizeInputUnits(
    unitSystem: UnitSystem,
    inputs: SubstitutionInput[],
    items: Record<string, FoodObject>
  ): void {
    for (const input of inputs) {
      const item = items[input.foodObjectId];
      if (!item) {
        continue;
      }
      const targetUnit = displayUnitForBasis(item.macroBasis, unitSystem);
      if (input.unit !== targetUnit) {
        updateSubstitutionInput(input.foodObjectId, {
          quantity: normalizeDisplayQuantity(convertQuantity(input.quantity, input.unit, targetUnit)),
          unit: targetUnit
        });
      }
    }
  }

  function itemInitial(item: FoodObject): string {
    const category = item.primaryFoodCategory ?? foodCategories(item)[0] ?? null;
    return category ? category.name.charAt(0).toUpperCase() : item.name.charAt(0).toUpperCase();
  }

  function visibleFilterOptions(options: SubstitutionFilterOption[], query: string): SubstitutionFilterOption[] {
    const normalizedQuery = query.trim().toLowerCase();
    const activeKeys = new Set($searchStore.filters.map(filterKey));
    return options
      .filter((option) => !activeKeys.has(filterKey(option)))
      .filter((option) => normalizedQuery === "" || option.searchText.toLowerCase().includes(normalizedQuery) || option.label.toLowerCase().includes(normalizedQuery))
      .slice(0, 6);
  }

  function addSubstitutionFilter(option: SubstitutionFilterOption): void {
    setFilters([...$searchStore.filters.filter((filter) => !sameFilter(filter, option)), {
      filterId: option.filterId,
      kind: option.kind,
      include: option.include
    }]);
    includeFilterQuery = "";
    excludeFilterQuery = "";
    includeFilterOpen = false;
    excludeFilterOpen = false;
  }

  function removeSubstitutionFilter(filter: SearchFilter): void {
    setFilters($searchStore.filters.filter((existing) => !sameFilter(existing, filter)));
  }

  function filterLabel(filter: SearchFilter): string {
    return [...includeFilterOptions, ...excludeFilterOptions].find((option) => sameFilter(option, filter))?.label ?? filter.filterId;
  }

  function filterDescription(filter: SearchFilter): string {
    return [...includeFilterOptions, ...excludeFilterOptions].find((option) => sameFilter(option, filter))?.description ?? filterKindLabel(filter.kind);
  }

  function filterKindLabel(kind: SearchFilterKind): string {
    return kind.replace("_", " ");
  }

  function sameFilter(a: Pick<SearchFilter, "filterId" | "kind" | "include">, b: Pick<SearchFilter, "filterId" | "kind" | "include">): boolean {
    return a.filterId === b.filterId && a.kind === b.kind && a.include === b.include;
  }

  function filterKey(filter: Pick<SearchFilter, "filterId" | "kind" | "include">): string {
    return `${filter.include ? "include" : "exclude"}:${filter.kind}:${filter.filterId}`;
  }

  function onFilterInputKeydown(options: SubstitutionFilterOption[], event: KeyboardEvent): void {
    if (event.key !== "Enter" || options.length === 0) {
      return;
    }
    event.preventDefault();
    addSubstitutionFilter(options[0]);
  }

  /** Sends a Substitution request only when the current entitlement allows execution. */
  function onSubstitutionSearch(): void {
    if (executionAllowed) {
      requestSubstitutionSearch();
    }
  }
</script>

<!-- Implements DESIGN-001 SearchView Substitution Input controls. -->
<section class="grid gap-3 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4" aria-label="Substitution inputs">
  {#if ($substitutionState?.substitutionInputs.length ?? 0) > 0}
    <ul class="grid gap-3">
      {#each $substitutionState?.substitutionInputs ?? [] as input (input.foodObjectId)}
        {@const selectedItem = inputItem(input.foodObjectId)}
        <li class="relative grid gap-3 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4" data-substitution-card>
          <h3 class="text-left text-base font-semibold" data-food-object-id={input.foodObjectId}>{inputLabel(input.foodObjectId)}</h3>

          <div class="grid gap-3 sm:grid-cols-[96px_1fr_auto]">
            {#if selectedItem}
              <div
                class="grid h-24 w-24 place-items-center rounded bg-[var(--color-muted)]"
                data-substitution-image-wrapper
              >
                {#if selectedItem.imageUrl}
                  <img
                    class="h-24 w-24 rounded object-cover"
                    src={selectedItem.imageUrl}
                    alt={selectedItem.name}
                    loading="lazy"
                    data-substitution-image
                  />
                {:else}
                  <div
                    class="grid place-items-center text-center"
                    role="img"
                    aria-label={selectedItem.primaryFoodCategory ? selectedItem.primaryFoodCategory.name : selectedItem.name}
                    data-substitution-placeholder
                  >
                    <span class="font-data text-2xl font-semibold text-[var(--color-on-muted)]" aria-hidden="true">{itemInitial(selectedItem)}</span>
                    {#if selectedItem.primaryFoodCategory}
                      <span class="mt-1 px-1 text-xs text-[var(--color-on-muted)]">{selectedItem.primaryFoodCategory.name}</span>
                    {/if}
                  </div>
                {/if}
              </div>

              <div class="grid h-24 content-between">
                <dl class="grid gap-1 font-data text-xs" data-substitution-macros>
                  <div class="grid grid-cols-[5rem_auto] items-center gap-3">
                    <dt class="text-[var(--color-muted)]">Protein</dt>
                    <dd>{selectedItem.macros.protein}g</dd>
                  </div>
                  <div class="grid grid-cols-[5rem_auto] items-center gap-3">
                    <dt class="text-[var(--color-muted)]">Carbs</dt>
                    <dd>{selectedItem.macros.carbohydrates}g</dd>
                  </div>
                  <div class="grid grid-cols-[5rem_auto] items-center gap-3">
                    <dt class="text-[var(--color-muted)]">Fat</dt>
                    <dd>{selectedItem.macros.fat}g</dd>
                  </div>
                  <div class="grid grid-cols-[5rem_auto] items-center gap-3" data-substitution-calories>
                    <dt class="text-[var(--color-muted)]">Calories</dt>
                    <dd>{selectedItem.calories} kcal</dd>
                  </div>
                </dl>
                <p class="font-data text-[0.68rem] leading-none text-[var(--color-muted)]" data-substitution-macro-basis>{macroBasisLabel(selectedItem)}</p>
              </div>
            {:else}
              <div
                class="grid h-24 w-24 place-items-center rounded bg-[var(--color-muted)]"
                role="img"
                aria-label={inputLabel(input.foodObjectId)}
                data-substitution-placeholder
              >
                <span class="font-data text-2xl font-semibold text-[var(--color-on-muted)]" aria-hidden="true">{inputInitial(input.foodObjectId)}</span>
              </div>

              <div class="hidden sm:block" aria-hidden="true"></div>
            {/if}

            <div class="grid grid-cols-[10.5ch_7ch] content-start gap-2 justify-self-start sm:justify-self-end" data-substitution-controls>
              <div class="grid gap-0.5">
                <label class="text-[0.68rem] leading-none text-[var(--color-muted)]" for={`qty-${input.foodObjectId}`}>Quantity</label>
                <input
                  id={`qty-${input.foodObjectId}`}
                  class="h-8 w-[10.5ch] rounded border border-[var(--color-border)] bg-transparent px-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
                  type="number"
                  min="0"
                  step="0.1"
                  value={input.quantity}
                  aria-label={`Quantity for ${inputLabel(input.foodObjectId)}`}
                  oninput={(event) => onRowQuantityInput(input.foodObjectId, event)}
                />
              </div>

              <div class="grid gap-0.5">
                <label class="text-[0.68rem] leading-none text-[var(--color-muted)]" for={`unit-${input.foodObjectId}`}>Unit</label>
                <select
                  id={`unit-${input.foodObjectId}`}
                  class="h-8 w-[7ch] rounded border border-[var(--color-border)] bg-transparent px-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
                  value={input.unit}
                  aria-label={`Unit for ${inputLabel(input.foodObjectId)}`}
                  onchange={(event) => onRowUnitChange(input.foodObjectId, event)}
                >
                  {#each rowUnitOptions(selectedItem, $preferencesStore.unitSystem) as option}
                    <option value={option.value}>{option.label}</option>
                  {/each}
                </select>
              </div>
            </div>
          </div>

          {#if selectedItem}
            <div class="flex flex-wrap justify-start gap-1 pr-12 text-left" data-substitution-categories>
              {#if foodCategories(selectedItem).length > 0}
                {#each foodCategories(selectedItem) as category (category.id)}
                  <span class="rounded bg-[var(--color-muted)] px-2 py-0.5 text-xs text-[var(--color-on-muted)]">{category.name}</span>
                {/each}
              {/if}
            </div>
          {/if}

          <button
            type="button"
            class="absolute bottom-4 right-4 flex h-9 w-9 items-center justify-center rounded-full border border-[var(--color-accent)] bg-[var(--color-accent)] text-xl font-semibold leading-none text-[var(--color-on-accent)] shadow-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
            aria-label={`Remove ${inputLabel(input.foodObjectId)} from substitutions`}
            onclick={() => removeSubstitutionInput(input.foodObjectId)}
          >
            <span class="-translate-y-px leading-none" aria-hidden="true">−</span>
          </button>
        </li>
      {/each}
    </ul>
  {:else}
    <p class="text-sm text-[var(--color-muted)]" data-substitution-empty>
      Search above and select foods or meals to build your substitution input list.
    </p>
  {/if}

  {#if ($substitutionState?.substitutionInputs.length ?? 0) > 0}
  <div class="grid gap-3 rounded border border-[var(--color-border)] bg-[var(--color-background)] p-3" aria-label="Substitution filters" data-substitution-filters>
    {#if filterOptionsStatus === "loading"}
      <p class="text-sm text-[var(--color-muted)]" role="status" data-filter-options-loading>Loading filter options…</p>
    {:else if filterOptionsStatus === "empty"}
      <p class="text-sm text-[var(--color-muted)]" role="status" data-filter-options-empty>No additional filter options are available. Selected-item classifications remain available.</p>
    {:else if filterOptionsStatus === "error"}
      <div class="flex flex-wrap items-center gap-2" role="alert" data-filter-options-error>
        <p class="text-sm text-[var(--color-muted)]">Filter options are temporarily unavailable. Selected-item classifications remain available.</p>
        <button type="button" class="rounded border border-[var(--color-border)] px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" onclick={() => loadFilterOptions()}>Retry filter options</button>
      </div>
    {/if}
    <div class="grid gap-3 md:grid-cols-2">
      <div class="relative grid gap-1">
        <label class="text-xs font-medium text-[var(--color-muted)]" for="substitution-include-filter">Must include</label>
        <input
          id="substitution-include-filter"
          class="h-10 rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
          type="text"
          role="combobox"
          aria-expanded={includeFilterOpen}
          aria-controls="substitution-include-filter-options"
          placeholder="Search categories or roles…"
          bind:value={includeFilterQuery}
          onfocus={() => (includeFilterOpen = true)}
          oninput={() => (includeFilterOpen = true)}
          onkeydown={(event) => onFilterInputKeydown(visibleIncludeOptions, event)}
          onblur={() => setTimeout(() => (includeFilterOpen = false), 100)}
          data-substitution-include-filter
        />
        {#if includeFilterOpen}
          <div
            id="substitution-include-filter-options"
            class="absolute left-0 right-0 top-full z-20 mt-1 max-h-56 overflow-auto rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-1 shadow-lg"
            role="listbox"
            data-substitution-include-options
          >
            {#if visibleIncludeOptions.length > 0}
              {#each visibleIncludeOptions as option (filterKey(option))}
                <button
                  type="button"
                  class="grid w-full gap-0.5 rounded px-2 py-1.5 text-left text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
                  role="option"
                  aria-selected="false"
                  onmousedown={(event) => onFilterOptionMouseDown(option, event)}
                  onkeydown={(event) => onFilterOptionKeydown(option, event)}
                >
                  <span>{option.label}</span>
                  <span class="text-xs text-[var(--color-muted)]">{option.description}</span>
                </button>
              {/each}
            {:else}
              <p class="px-2 py-1.5 text-sm text-[var(--color-muted)]">No include filters found.</p>
            {/if}
          </div>
        {/if}
      </div>

      <div class="relative grid gap-1">
        <label class="text-xs font-medium text-[var(--color-muted)]" for="substitution-exclude-filter">Must exclude</label>
        <input
          id="substitution-exclude-filter"
          class="h-10 rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
          type="text"
          role="combobox"
          aria-expanded={excludeFilterOpen}
          aria-controls="substitution-exclude-filter-options"
          placeholder="Search allergies or diets…"
          bind:value={excludeFilterQuery}
          onfocus={() => (excludeFilterOpen = true)}
          oninput={() => (excludeFilterOpen = true)}
          onkeydown={(event) => onFilterInputKeydown(visibleExcludeOptions, event)}
          onblur={() => setTimeout(() => (excludeFilterOpen = false), 100)}
          data-substitution-exclude-filter
        />
        {#if excludeFilterOpen}
          <div
            id="substitution-exclude-filter-options"
            class="absolute left-0 right-0 top-full z-20 mt-1 max-h-56 overflow-auto rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-1 shadow-lg"
            role="listbox"
            data-substitution-exclude-options
          >
            {#if visibleExcludeOptions.length > 0}
              {#each visibleExcludeOptions as option (filterKey(option))}
                <button
                  type="button"
                  class="grid w-full gap-0.5 rounded px-2 py-1.5 text-left text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
                  role="option"
                  aria-selected="false"
                  onmousedown={(event) => onFilterOptionMouseDown(option, event)}
                  onkeydown={(event) => onFilterOptionKeydown(option, event)}
                >
                  <span>{option.label}</span>
                  <span class="text-xs text-[var(--color-muted)]">{option.description}</span>
                </button>
              {/each}
            {:else}
              <p class="px-2 py-1.5 text-sm text-[var(--color-muted)]">No exclude filters found.</p>
            {/if}
          </div>
        {/if}
      </div>
    </div>

    {#if activeIncludeFilters.length > 0}
      <div class="grid gap-1 text-left" data-substitution-include-chips>
        <p class="text-xs text-[var(--color-muted)]">Must include</p>
        <div class="flex flex-wrap gap-1">
          {#each activeIncludeFilters as filter (filterKey(filter))}
            <button
              type="button"
              class="rounded-full border border-[var(--color-primary)] bg-[var(--color-primary)] px-2 py-0.5 text-xs text-[var(--color-on-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
              title={filterDescription(filter)}
              aria-label={`Remove include filter ${filterLabel(filter)}`}
              onclick={() => removeSubstitutionFilter(filter)}
            >
              {filterLabel(filter)} ×
            </button>
          {/each}
        </div>
      </div>
    {/if}

    {#if activeExcludeFilters.length > 0}
      <div class="grid gap-1 text-left" data-substitution-exclude-chips>
        <p class="text-xs text-[var(--color-muted)]">Must exclude</p>
        <div class="flex flex-wrap gap-1">
          {#each activeExcludeFilters as filter (filterKey(filter))}
            <button
              type="button"
              class="rounded-full border border-[var(--color-accent)] bg-[var(--color-accent)] px-2 py-0.5 text-xs text-[var(--color-on-accent)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
              title={filterDescription(filter)}
              aria-label={`Remove exclude filter ${filterLabel(filter)}`}
              onclick={() => removeSubstitutionFilter(filter)}
            >
              {filterLabel(filter)} ×
            </button>
          {/each}
        </div>
      </div>
    {/if}
  </div>
  {/if}

  <button
    type="button"
    class="w-full rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)] transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:cursor-not-allowed disabled:opacity-60"
    disabled={($substitutionState?.substitutionInputs.length ?? 0) === 0 || !executionAllowed}
    aria-describedby={entitlementFeedback ? "substitution-entitlement-feedback" : undefined}
    onclick={onSubstitutionSearch}
    data-substitution-search
  >
    Find substitutions
  </button>

  {#if entitlementFeedback}
    <p id="substitution-entitlement-feedback" class="text-sm text-[var(--color-muted)]" data-substitution-entitlement-feedback>
      {entitlementFeedback}
    </p>
  {/if}
</section>
