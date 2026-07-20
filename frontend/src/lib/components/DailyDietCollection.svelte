<script lang="ts">
  import { onDestroy } from "svelte";
  import type {
    AppError,
    CanonicalQuantityUnit,
    ClassificationSummary,
    DailyDiet,
    FoodObject,
    MacroProjection,
    RankedAutocomplete
  } from "../api/generated";
  import { DailyDietClientError } from "../api/daily-diet-client";
  import { fetchFoodObject } from "../api/search-client";
  import {
    convertQuantity,
    defaultDisplayQuantity,
    displayUnitForBasis,
    formatCalories,
    formatDisplayQuantity,
    macroBasisDisplayLabel,
    unitOptionsForBasis
  } from "../units";
  import { preferencesStore } from "../stores/preferences";
  import {
    clearDailyDietCreateIntent,
    clearDailyDietState,
    createDailyDiet,
    dailyDietStore,
    deleteDailyDiet,
    loadDailyDiets,
    replaceDailyDiet
  } from "../stores/daily-diet";
  import type { AuthStatus } from "../stores/auth-session";
  import AutocompleteDropdown from "./AutocompleteDropdown.svelte";

  // Implements DESIGN-001 SearchView authenticated Daily Diet create, lookup, and replacement editor.
  // Implements DESIGN-008 SavedDataRepository user-owned entries, unique names, and server aggregates.

  export interface DailyDietEditSelection {
    key: number;
    diet: DailyDiet;
  }

  interface DraftFoodObject {
    key: number;
    item: FoodObject;
    quantity: number;
    unit: CanonicalQuantityUnit;
  }

  let {
    authStatus = "unknown",
    authenticated = false,
    userId = null,
    executionAllowed = true,
    entitlementFeedback = null,
    selectedDiet = null,
    onEditDiet = () => undefined,
    onSignIn = () => undefined
  }: {
    authStatus?: AuthStatus;
    authenticated?: boolean;
    userId?: string | null;
    executionAllowed?: boolean;
    entitlementFeedback?: string | null;
    selectedDiet?: DailyDietEditSelection | null;
    onEditDiet?: (diet: DailyDiet) => void;
    onSignIn?: () => void;
  } = $props();

  let draftName = $state("My Daily Diet");
  let draftFoodObjects = $state<DraftFoodObject[]>([]);
  let foodSearchQuery = $state("");
  let loadedUserId = $state<string | null>(null);
  let draftError = $state<string | null>(null);
  let serverAggregate = $state<MacroProjection | null>(null);
  let savedDietId = $state<string | null>(null);
  let editingDietId = $state<string | null>(null);
  let editingLoading = $state(false);
  let savedListOpen = $state(true);
  let lastEditSelectionKey = $state<number | null>(null);
  let nextDraftKey = 0;
  let hydrationGeneration = 0;
  const hydrationControllers = new Set<AbortController>();

  let canEdit = $derived(authenticated && executionAllowed);
  let collectionError = $derived<AppError | null>($dailyDietStore.error);
  let savedDiet = $derived(
    savedDietId ? $dailyDietStore.collections.find((diet) => diet.id === savedDietId) ?? null : null
  );
  let aggregate = $derived(serverAggregate ?? calculateAggregate(draftFoodObjects));

  onDestroy(cancelHydration);

  $effect(() => {
    if (authStatus === "authenticated" && authenticated && userId && loadedUserId !== userId) {
      if (loadedUserId !== null) {
        resetIdentityOwnedDraft();
        clearDailyDietState();
      }
      loadedUserId = userId;
      void loadDailyDiets().catch(() => undefined);
      return;
    }
    if (!authenticated && loadedUserId !== null) {
      loadedUserId = null;
      resetIdentityOwnedDraft();
      clearDailyDietState();
    }
  });

  $effect(() => {
    if (selectedDiet && selectedDiet.key !== lastEditSelectionKey && canEdit) {
      lastEditSelectionKey = selectedDiet.key;
      void openForEditing(selectedDiet.diet);
    }
  });

  function calculateAggregate(items: DraftFoodObject[]): MacroProjection {
    return items.reduce(
      (total, draft) => {
        const baseUnit = draft.item.macroBasis === "100ml" ? "ml" : "g";
        const scale = convertQuantity(draft.quantity, draft.unit, baseUnit) / 100;
        return {
          protein: total.protein + draft.item.macros.protein * scale,
          carbohydrates: total.carbohydrates + draft.item.macros.carbohydrates * scale,
          fat: total.fat + draft.item.macros.fat * scale,
          calories: total.calories + draft.item.calories * scale
        };
      },
      { protein: 0, carbohydrates: 0, fat: 0, calories: 0 }
    );
  }

  /** Hydrates a selected autocomplete result into the editable Food Object card list. */
  async function addFoodObject(suggestion: RankedAutocomplete): Promise<void> {
    if (!canEdit || editingLoading) return;
    foodSearchQuery = "";
    const generation = hydrationGeneration;
    const controller = new AbortController();
    hydrationControllers.add(controller);
    draftError = null;
    try {
      const item = await fetchFoodObject(suggestion.itemId, controller.signal, suggestion.objectType);
      if (controller.signal.aborted || generation !== hydrationGeneration) return;
      clearDailyDietCreateIntent();
      draftFoodObjects = [...draftFoodObjects, {
        key: ++nextDraftKey,
        item,
        quantity: defaultDisplayQuantity(item.macroBasis, $preferencesStore.unitSystem),
        unit: displayUnitForBasis(item.macroBasis, $preferencesStore.unitSystem)
      }];
      serverAggregate = null;
      savedDietId = null;
    } catch {
      if (!controller.signal.aborted && generation === hydrationGeneration) {
        draftError = "That food or meal could not be added. Please try again.";
      }
    } finally {
      hydrationControllers.delete(controller);
    }
  }

  /** Loads all persisted Food Objects before exposing an existing Daily Diet as editable state. */
  async function openForEditing(diet: DailyDiet): Promise<void> {
    cancelHydration();
    const generation = hydrationGeneration;
    const controller = new AbortController();
    hydrationControllers.add(controller);
    editingLoading = true;
    editingDietId = diet.id;
    draftName = diet.name;
    draftFoodObjects = [];
    serverAggregate = null;
    savedDietId = null;
    draftError = null;
    foodSearchQuery = "";
    try {
      const entries = [...diet.entries].sort((left, right) => left.position - right.position);
      const items = await Promise.all(entries.map((entry) => fetchFoodObject(entry.foodObjectId, controller.signal, entry.foodObjectType)));
      if (controller.signal.aborted || generation !== hydrationGeneration) return;
      draftFoodObjects = entries.map((entry, index) => ({
        key: ++nextDraftKey,
        item: items[index],
        quantity: entry.quantity,
        unit: entry.unit
      }));
      serverAggregate = diet.aggregateMacros;
      savedDietId = diet.id;
    } catch {
      if (!controller.signal.aborted && generation === hydrationGeneration) {
        draftError = "This Daily Diet could not be opened. Please try again.";
      }
    } finally {
      hydrationControllers.delete(controller);
      if (generation === hydrationGeneration) editingLoading = false;
    }
  }

  function cancelHydration(): void {
    hydrationGeneration += 1;
    for (const controller of hydrationControllers) controller.abort();
    hydrationControllers.clear();
    editingLoading = false;
  }

  function markDraftChanged(): void {
    clearDailyDietCreateIntent();
    serverAggregate = null;
    savedDietId = null;
    draftError = null;
  }

  function updateQuantity(key: number, event: Event): void {
    const quantity = Number((event.currentTarget as HTMLInputElement).value);
    draftFoodObjects = draftFoodObjects.map((draft) => draft.key === key ? { ...draft, quantity } : draft);
    markDraftChanged();
  }

  function updateUnit(key: number, event: Event): void {
    const unit = (event.currentTarget as HTMLSelectElement).value as CanonicalQuantityUnit;
    draftFoodObjects = draftFoodObjects.map((draft) => draft.key === key ? { ...draft, unit } : draft);
    markDraftChanged();
  }

  function moveFoodObject(key: number, direction: -1 | 1): void {
    const index = draftFoodObjects.findIndex((draft) => draft.key === key);
    const target = index + direction;
    if (index < 0 || target < 0 || target >= draftFoodObjects.length) return;
    const next = [...draftFoodObjects];
    [next[index], next[target]] = [next[target], next[index]];
    draftFoodObjects = next;
    markDraftChanged();
  }

  function removeFoodObject(key: number): void {
    draftFoodObjects = draftFoodObjects.filter((draft) => draft.key !== key);
    markDraftChanged();
  }

  async function saveCollection(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    if (!canEdit || editingLoading) return;
    const name = draftName.trim();
    if (name.length === 0) {
      draftError = "Give this one-day collection a name.";
      return;
    }
    const duplicate = $dailyDietStore.collections.find((diet) =>
      diet.id !== editingDietId && diet.name.trim().toLocaleLowerCase() === name.toLocaleLowerCase()
    );
    if (duplicate) {
      draftError = "A Daily Diet with this name already exists.";
      return;
    }
    if (draftFoodObjects.length < 2) {
      draftError = "Add at least two foods or meals before saving your Daily Diet.";
      return;
    }
    if (draftFoodObjects.some((draft) => !Number.isFinite(draft.quantity) || draft.quantity <= 0)) {
      draftError = "Each item needs a quantity greater than zero.";
      return;
    }
    draftError = null;
    try {
      const request = {
        name,
        entries: draftFoodObjects.map((draft, position) => ({
          foodObjectId: draft.item.id,
          foodObjectType: draft.item.objectType,
          quantity: draft.quantity,
          unit: draft.unit,
          position
        }))
      };
      const result = editingDietId
        ? await replaceDailyDiet(editingDietId, request)
        : await createDailyDiet(request);
      draftName = result.name;
      savedDietId = result.id;
      editingDietId = result.id;
      serverAggregate = result.aggregateMacros;
    } catch (error) {
      draftError = error instanceof DailyDietClientError && error.status === 409
        ? "A Daily Diet with this name already exists."
        : "Your Daily Diet could not be saved. Please try again.";
    }
  }

  async function deleteEditingDiet(): Promise<void> {
    if (!editingDietId || !canEdit || $dailyDietStore.mutation !== "idle") return;
    if (!window.confirm(`Delete “${draftName}”? This cannot be undone.`)) return;
    try {
      await deleteDailyDiet(editingDietId);
      resetDraft();
    } catch {
      draftError = "Your Daily Diet could not be deleted. Please try again.";
    }
  }

  function resetDraft(): void {
    cancelHydration();
    clearDailyDietCreateIntent();
    draftName = "My Daily Diet";
    draftFoodObjects = [];
    foodSearchQuery = "";
    serverAggregate = null;
    savedDietId = null;
    editingDietId = null;
    draftError = null;
  }

  /** Clears all editable state owned by the previous authenticated identity. */
  function resetIdentityOwnedDraft(): void {
    resetDraft();
    lastEditSelectionKey = null;
  }

  function foodCategories(item: FoodObject): ClassificationSummary[] {
    return item.classifications.filter((classification) => classification.kind === "food_category");
  }

  function itemInitial(item: FoodObject): string {
    const category = item.primaryFoodCategory ?? foodCategories(item)[0] ?? null;
    return (category?.name ?? item.name).charAt(0).toUpperCase();
  }
</script>

<!-- Implements DESIGN-001 SearchView Daily Diet create and edit surface. -->
<section class="grid gap-4 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4" aria-labelledby="daily-diet-editor-title" data-daily-diet-collection>
  <div class="flex items-start justify-between gap-3">
    <div class="grid gap-1">
      <h2 id="daily-diet-editor-title" class="text-lg font-semibold">{editingDietId ? "Edit your Daily Diet" : "Build your Daily Diet"}</h2>
      <p class="text-sm text-[var(--color-muted)]">Add at least two foods or meals to make a one-day collection.</p>
    </div>
    {#if editingDietId}
      <button type="button" class="rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:cursor-not-allowed" onclick={resetDraft} disabled={$dailyDietStore.mutation !== "idle"} data-daily-diet-new>New</button>
    {/if}
  </div>

  {#if authStatus === "unknown" || authStatus === "authenticating"}
    <p class="rounded border border-[var(--color-border)] px-3 py-3 text-sm text-[var(--color-muted)]" role="status" data-daily-diet-auth-loading>Checking your sign-in status…</p>
  {:else if !authenticated}
    <div class="grid gap-3 rounded border border-[var(--color-border)] px-3 py-3" data-daily-diet-auth-guidance>
      <p class="text-sm">Sign in to build and save a Daily Diet.</p>
      <button type="button" class="w-fit rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" onclick={onSignIn}>Sign in to continue</button>
    </div>
  {:else}
    {#if entitlementFeedback}
      <p class="rounded border border-[var(--color-accent)] px-3 py-2 text-sm" role="alert" data-daily-diet-entitlement>{entitlementFeedback}</p>
    {/if}

    <form class="grid gap-4" onsubmit={saveCollection} aria-label="Daily Diet collection form">
      <label class="grid gap-1 text-sm font-medium" for="daily-diet-name">
        Collection name
        <input id="daily-diet-name" class="h-10 rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" value={draftName} oninput={(event) => { draftName = (event.currentTarget as HTMLInputElement).value; markDraftChanged(); }} disabled={!canEdit || editingLoading} />
      </label>

      <div class="grid gap-1">
        <span class="text-sm font-medium">Add foods or meals</span>
        <AutocompleteDropdown query={foodSearchQuery} placeholder="Search foods or meals to add…" focusKey={editingDietId ?? "new-daily-diet"} focusOnMount={false} selectFirstOnEnter={true} onQueryInput={(value) => (foodSearchQuery = value)} onSelect={(item) => void addFoodObject(item)} />
      </div>

      {#if editingLoading}
        <p class="rounded border border-[var(--color-border)] px-3 py-4 text-sm text-[var(--color-muted)]" role="status">Loading Daily Diet items…</p>
      {:else if draftFoodObjects.length === 0}
        <p class="rounded border border-dashed border-[var(--color-border)] px-3 py-4 text-sm text-[var(--color-muted)]" data-daily-diet-empty>No foods or meals added yet. Search above and choose one to start your day.</p>
      {:else}
        <ol class="grid gap-3" aria-label="Foods and meals in this Daily Diet" data-daily-diet-meals>
          {#each draftFoodObjects as draft, index (draft.key)}
            <li class="relative grid gap-3 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4" data-daily-diet-meal={draft.item.id}>
              <h3 class="pr-40 text-left text-base font-semibold">{index + 1}. {draft.item.name}</h3>

              <div class="grid gap-3 sm:grid-cols-[96px_1fr_auto]">
                <div class="grid h-24 w-24 place-items-center rounded bg-[var(--color-muted)]" data-daily-diet-image-wrapper>
                  {#if draft.item.imageUrl}
                    <img class="h-24 w-24 rounded object-cover" src={draft.item.imageUrl} alt={draft.item.name} loading="lazy" />
                  {:else}
                    <div class="grid place-items-center text-center" role="img" aria-label={draft.item.primaryFoodCategory?.name ?? draft.item.name}>
                      <span class="font-data text-2xl font-semibold text-[var(--color-on-muted)]" aria-hidden="true">{itemInitial(draft.item)}</span>
                      {#if draft.item.primaryFoodCategory}<span class="mt-1 px-1 text-xs text-[var(--color-on-muted)]">{draft.item.primaryFoodCategory.name}</span>{/if}
                    </div>
                  {/if}
                </div>

                <div class="grid h-24 content-between">
                  <dl class="grid gap-1 font-data text-xs" data-daily-diet-item-macros>
                    <div class="grid grid-cols-[5rem_auto] gap-3"><dt class="text-[var(--color-muted)]">Protein</dt><dd>{draft.item.macros.protein}g</dd></div>
                    <div class="grid grid-cols-[5rem_auto] gap-3"><dt class="text-[var(--color-muted)]">Carbs</dt><dd>{draft.item.macros.carbohydrates}g</dd></div>
                    <div class="grid grid-cols-[5rem_auto] gap-3"><dt class="text-[var(--color-muted)]">Fat</dt><dd>{draft.item.macros.fat}g</dd></div>
                    <div class="grid grid-cols-[5rem_auto] gap-3"><dt class="text-[var(--color-muted)]">Calories</dt><dd>{formatCalories(draft.item.calories)} kcal</dd></div>
                  </dl>
                  <p class="font-data text-[0.68rem] leading-none text-[var(--color-muted)]">{macroBasisDisplayLabel(draft.item.macroBasis, $preferencesStore.unitSystem)}</p>
                </div>

                <div class="grid grid-cols-[10.5ch_7ch] content-start gap-2 justify-self-start sm:justify-self-end sm:pt-10">
                  <label class="grid gap-0.5 text-[0.68rem] leading-none text-[var(--color-muted)]" for={`daily-diet-quantity-${draft.key}`}>Quantity
                    <input id={`daily-diet-quantity-${draft.key}`} class="h-8 w-[10.5ch] rounded border border-[var(--color-border)] bg-transparent px-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" type="number" min="0.01" step="0.01" value={draft.quantity} aria-label={`Quantity for ${draft.item.name}`} oninput={(event) => updateQuantity(draft.key, event)} disabled={!canEdit} />
                  </label>
                  <label class="grid gap-0.5 text-[0.68rem] leading-none text-[var(--color-muted)]" for={`daily-diet-unit-${draft.key}`}>Unit
                    <select id={`daily-diet-unit-${draft.key}`} class="h-8 w-[7ch] rounded border border-[var(--color-border)] bg-transparent px-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" value={draft.unit} aria-label={`Unit for ${draft.item.name}`} onchange={(event) => updateUnit(draft.key, event)} disabled={!canEdit}>
                      {#each unitOptionsForBasis(draft.item.macroBasis, $preferencesStore.unitSystem) as option}<option value={option.value}>{option.label}</option>{/each}
                    </select>
                  </label>
                </div>
              </div>

              {#if foodCategories(draft.item).length > 0}
                <div class="flex flex-wrap justify-start gap-1 pr-40">{#each foodCategories(draft.item) as category (category.id)}<span class="rounded bg-[var(--color-muted)] px-2 py-0.5 text-xs text-[var(--color-on-muted)]">{category.name}</span>{/each}</div>
              {/if}

              <div class="absolute right-3 top-3 flex gap-2">
                <button type="button" class="rounded border px-2 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" aria-label={`Move ${draft.item.name} up`} onclick={() => moveFoodObject(draft.key, -1)} disabled={!canEdit || index === 0}>↑</button>
                <button type="button" class="rounded border px-2 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" aria-label={`Move ${draft.item.name} down`} onclick={() => moveFoodObject(draft.key, 1)} disabled={!canEdit || index === draftFoodObjects.length - 1}>↓</button>
                <button type="button" class="rounded border border-[var(--color-accent)] bg-[var(--color-accent)] px-2 py-2 text-sm text-[var(--color-on-accent)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:cursor-not-allowed disabled:opacity-60" aria-label={`Remove ${draft.item.name}`} onclick={() => removeFoodObject(draft.key)} disabled={!canEdit}>Remove</button>
              </div>
            </li>
          {/each}
        </ol>
      {/if}

      <section class="grid gap-2 rounded border border-[var(--color-border)] p-3" aria-labelledby="daily-diet-aggregate-title" data-daily-diet-aggregate>
        <h3 id="daily-diet-aggregate-title" class="font-data text-xs uppercase text-[var(--color-muted)]">One-day aggregate macros</h3>
        <dl class="grid grid-cols-2 gap-2 font-data text-sm sm:grid-cols-4">
          <div><dt class="text-[var(--color-muted)]">Protein</dt><dd data-macro-protein>{formatDisplayQuantity(aggregate.protein)}g</dd></div>
          <div><dt class="text-[var(--color-muted)]">Carbs</dt><dd data-macro-carbs>{formatDisplayQuantity(aggregate.carbohydrates)}g</dd></div>
          <div><dt class="text-[var(--color-muted)]">Fat</dt><dd data-macro-fat>{formatDisplayQuantity(aggregate.fat)}g</dd></div>
          <div><dt class="text-[var(--color-muted)]">Calories</dt><dd data-macro-calories>{formatCalories(aggregate.calories)} kcal</dd></div>
        </dl>
        {#if savedDiet}<p class="text-xs text-[var(--color-muted)]" role="status" data-daily-diet-server-total>Totals confirmed by the server.</p>{/if}
      </section>

      {#if draftError || collectionError}
        <p class="rounded border border-[var(--color-error)] px-3 py-2 text-sm" role="alert" data-daily-diet-save-error>{draftError ?? collectionError?.message}</p>
      {/if}

      <div class="flex flex-wrap gap-2">
        <button type="submit" class="rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:cursor-not-allowed" disabled={!canEdit || editingLoading || draftFoodObjects.length < 2 || $dailyDietStore.mutation !== "idle"} data-daily-diet-save>
          {$dailyDietStore.mutation === "creating" || $dailyDietStore.mutation === "replacing" ? "Saving…" : editingDietId ? "Update" : "Save"}
        </button>
        {#if editingDietId}
          <button type="button" class="ml-auto rounded border border-[var(--color-accent)] bg-[var(--color-accent)] px-3 py-2 text-sm text-[var(--color-on-accent)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:cursor-not-allowed disabled:opacity-60" onclick={() => void deleteEditingDiet()} disabled={!canEdit || $dailyDietStore.mutation !== "idle"}>{$dailyDietStore.mutation === "deleting" ? "Removing…" : "Remove"}</button>
        {:else}
          <button type="button" class="ml-auto rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" onclick={resetDraft} disabled={!canEdit || draftFoodObjects.length === 0}>Clear draft</button>
        {/if}
      </div>
    </form>

    <section class="grid gap-2" aria-labelledby="saved-daily-diets-title" data-saved-daily-diets>
      <h3 id="saved-daily-diets-title" class="text-base font-semibold text-[var(--color-text)]">
        <button type="button" class="flex w-full items-center justify-between rounded py-1 text-left focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" aria-expanded={savedListOpen} onclick={() => (savedListOpen = !savedListOpen)}>
          <span>Saved Daily Diets</span><span class="text-sm font-normal text-[var(--color-muted)]">{savedListOpen ? "Hide" : "Show"}</span>
        </button>
      </h3>
      {#if savedListOpen}
        {#if $dailyDietStore.status === "loading" && $dailyDietStore.collections.length === 0}
          <p class="rounded border border-[var(--color-border)] px-3 py-3 text-sm text-[var(--color-muted)]" role="status" data-saved-daily-diets-loading>Loading saved Daily Diets…</p>
        {:else if $dailyDietStore.status === "error" && $dailyDietStore.collections.length === 0}
          <div class="grid gap-2 rounded border border-[var(--color-error)] px-3 py-3" role="alert" data-saved-daily-diets-error><p>{$dailyDietStore.error?.message ?? "Saved Daily Diets could not be loaded."}</p><button type="button" class="w-fit rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" onclick={() => void loadDailyDiets()}>Try again</button></div>
        {:else if $dailyDietStore.collections.length === 0}
          <p class="rounded border border-dashed border-[var(--color-border)] px-3 py-3 text-sm text-[var(--color-muted)]" data-saved-daily-diets-empty>No saved Daily Diets yet.</p>
        {:else}
          <ul class="grid gap-2">
            {#each $dailyDietStore.collections as diet (diet.id)}
              <li data-saved-daily-diet={diet.id}>
                <button type="button" class="w-full rounded border border-[var(--color-border)] p-3 text-left focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" aria-label={`Edit ${diet.name}`} onclick={() => onEditDiet(diet)}>
                  <span class="flex flex-wrap items-start justify-between gap-2"><span><span class="block font-medium">{diet.name}</span><span class="block text-xs text-[var(--color-muted)]">{diet.entries.length} items · {formatCalories(diet.aggregateMacros.calories)} kcal</span></span><span class="font-data text-xs text-[var(--color-muted)]">{formatDisplayQuantity(diet.aggregateMacros.protein)}g protein</span></span>
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      {/if}
    </section>
  {/if}
</section>
