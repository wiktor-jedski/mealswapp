<script lang="ts">
  import type {
    AppError,
    CanonicalQuantityUnit,
    FoodObject,
    MacroProjection
  } from "../api/generated";
  import { convertQuantity, defaultDisplayQuantity, displayUnitForBasis, formatDisplayQuantity, unitLabel } from "../units";
  import { preferencesStore } from "../stores/preferences";
  import {
    clearDailyDietCreateIntent,
    clearDailyDietState,
    createDailyDiet,
    dailyDietStore,
    loadDailyDiets,
    replaceDailyDiet
  } from "../stores/daily-diet";
  import type { AuthStatus } from "../stores/auth-session";

  // Implements DESIGN-001 SearchView authenticated Daily Diet collection editor.
  // Implements DESIGN-008 SavedDataRepository server-owned meal entries and aggregate macros.

  export interface DailyDietMealSelection {
    key: number;
    item: FoodObject;
  }

  interface DraftMeal {
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
    selections = [],
    selectionError = null,
    onSignIn = () => undefined
  }: {
    authStatus?: AuthStatus;
    authenticated?: boolean;
    userId?: string | null;
    executionAllowed?: boolean;
    entitlementFeedback?: string | null;
    selections?: DailyDietMealSelection[];
    selectionError?: string | null;
    onSignIn?: () => void;
  } = $props();

  let draftName = $state("My Daily Diet");
  let draftMeals = $state<DraftMeal[]>([]);
  let consumedSelectionKeys = $state<Set<number>>(new Set());
  let loadedUserId = $state<string | null>(null);
  let draftError = $state<string | null>(null);
  let serverAggregate = $state<MacroProjection | null>(null);
  let savedDietId = $state<string | null>(null);
  let editingDietId = $state<string | null>(null);

  let canEdit = $derived(authenticated && executionAllowed);
  let collectionError = $derived<AppError | null>($dailyDietStore.error);
  let savedDiet = $derived(
    savedDietId ? $dailyDietStore.collections.find((diet) => diet.id === savedDietId) ?? null : null
  );
  let aggregate = $derived(serverAggregate ?? calculateAggregate(draftMeals));

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
    const nextKeys = new Set(consumedSelectionKeys);
    let added = false;
    for (const selection of selections) {
      if (nextKeys.has(selection.key)) continue;
      if (!canEdit) continue;
      nextKeys.add(selection.key);
      added = true;
	  clearDailyDietCreateIntent();
      draftMeals = [
        ...draftMeals,
        {
          key: selection.key,
          item: selection.item,
          quantity: defaultDisplayQuantity(selection.item.macroBasis, $preferencesStore.unitSystem),
          unit: displayUnitForBasis(selection.item.macroBasis, $preferencesStore.unitSystem)
        }
      ];
      draftError = null;
      serverAggregate = null;
      savedDietId = null;
    }
    if (added) consumedSelectionKeys = nextKeys;
  });

  function calculateAggregate(meals: DraftMeal[]): MacroProjection {
    return meals.reduce(
      (total, meal) => {
        const baseUnit = meal.item.macroBasis === "100ml" ? "ml" : "g";
        const scale = convertQuantity(meal.quantity, meal.unit, baseUnit) / 100;
        return {
          protein: total.protein + meal.item.macros.protein * scale,
          carbohydrates: total.carbohydrates + meal.item.macros.carbohydrates * scale,
          fat: total.fat + meal.item.macros.fat * scale,
          calories: total.calories + meal.item.calories * scale
        };
      },
      { protein: 0, carbohydrates: 0, fat: 0, calories: 0 }
    );
  }

  function updateQuantity(key: number, event: Event): void {
	clearDailyDietCreateIntent();
    const quantity = Number((event.currentTarget as HTMLInputElement).value);
    draftMeals = draftMeals.map((meal) => meal.key === key ? { ...meal, quantity } : meal);
    serverAggregate = null;
    savedDietId = null;
  }

  function updateUnit(key: number, event: Event): void {
	clearDailyDietCreateIntent();
    const unit = (event.currentTarget as HTMLSelectElement).value as CanonicalQuantityUnit;
    draftMeals = draftMeals.map((meal) => meal.key === key ? { ...meal, unit } : meal);
    serverAggregate = null;
    savedDietId = null;
  }

  function moveMeal(key: number, direction: -1 | 1): void {
    const index = draftMeals.findIndex((meal) => meal.key === key);
    const target = index + direction;
    if (index < 0 || target < 0 || target >= draftMeals.length) return;
	clearDailyDietCreateIntent();
    const next = [...draftMeals];
    [next[index], next[target]] = [next[target], next[index]];
    draftMeals = next;
    serverAggregate = null;
    savedDietId = null;
  }

  function removeMeal(key: number): void {
	clearDailyDietCreateIntent();
    draftMeals = draftMeals.filter((meal) => meal.key !== key);
    serverAggregate = null;
    savedDietId = null;
  }

  async function saveCollection(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    if (!canEdit) return;
    const name = draftName.trim();
    if (name.length === 0) {
      draftError = "Give this one-day collection a name.";
      return;
    }
    if (draftMeals.length < 2) {
      draftError = "Add at least two meals before saving your Daily Diet.";
      return;
    }
    if (draftMeals.some((meal) => !Number.isFinite(meal.quantity) || meal.quantity <= 0)) {
      draftError = "Each meal needs a quantity greater than zero.";
      return;
    }
    draftError = null;
    try {
      const request = {
        name,
        entries: draftMeals.map((meal, position) => ({
          mealId: meal.item.id,
          quantity: meal.quantity,
          unit: meal.unit,
          position
        }))
      };
      const saved = editingDietId
        ? await replaceDailyDiet(editingDietId, request)
        : await createDailyDiet(request);
      savedDietId = saved.id;
      editingDietId = saved.id;
      serverAggregate = saved.aggregateMacros;
    } catch {
      draftError = "Your Daily Diet could not be saved. Please try again.";
    }
  }

  function resetDraft(): void {
	clearDailyDietCreateIntent();
    draftMeals = [];
    serverAggregate = null;
    savedDietId = null;
    editingDietId = null;
    draftError = null;
  }

  /** Clears all editable state owned by the previous authenticated identity. */
  function resetIdentityOwnedDraft(): void {
    draftName = "My Daily Diet";
    draftMeals = [];
    consumedSelectionKeys = new Set(selections.map((selection) => selection.key));
    draftError = null;
    serverAggregate = null;
    savedDietId = null;
    editingDietId = null;
  }
</script>

<!-- Implements DESIGN-001 SearchView Daily Diet collection editor and authenticated-action guidance. -->
<section
  class="grid gap-4 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4"
  aria-labelledby="daily-diet-editor-title"
  data-daily-diet-collection
>
  <div class="grid gap-1">
    <h2 id="daily-diet-editor-title" class="text-lg font-semibold">Build your Daily Diet</h2>
    <p class="text-sm text-[var(--color-muted)]">Add at least two meals to make a one-day collection.</p>
  </div>

  {#if authStatus === "unknown" || authStatus === "authenticating"}
    <p class="rounded border border-[var(--color-border)] px-3 py-3 text-sm text-[var(--color-muted)]" role="status" data-daily-diet-auth-loading>
      Checking your sign-in status…
    </p>
  {:else if !authenticated}
    <div class="grid gap-3 rounded border border-[var(--color-border)] px-3 py-3" data-daily-diet-auth-guidance>
      <p class="text-sm">Sign in to build and save a Daily Diet.</p>
      <button
        type="button"
        class="w-fit rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)] transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
        onclick={onSignIn}
      >
        Sign in to continue
      </button>
    </div>
  {:else}
    {#if entitlementFeedback}
      <p class="rounded border border-[var(--color-accent)] px-3 py-2 text-sm" role="alert" data-daily-diet-entitlement>
        {entitlementFeedback}
      </p>
    {/if}

    {#if selectionError}
      <p class="rounded border border-[var(--color-error)] px-3 py-2 text-sm" role="alert" data-daily-diet-selection-error>
        {selectionError}
      </p>
    {/if}

    <form class="grid gap-4" onsubmit={saveCollection} aria-label="Daily Diet collection form">
      <label class="grid gap-1 text-sm font-medium" for="daily-diet-name">
        Collection name
        <input
          id="daily-diet-name"
          class="rounded border border-[#E0E0E0] bg-white px-3 py-2 text-sm text-[#111827] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
          value={draftName}
          oninput={(event) => { clearDailyDietCreateIntent(); draftName = (event.currentTarget as HTMLInputElement).value; }}
          disabled={!canEdit}
        />
      </label>

      {#if draftMeals.length === 0}
        <p class="rounded border border-dashed border-[var(--color-border)] px-3 py-4 text-sm text-[var(--color-muted)]" data-daily-diet-empty>
          No meals added yet. Search above and choose a meal to start your day.
        </p>
      {:else}
        <ol class="grid gap-3" aria-label="Meals in this Daily Diet" data-daily-diet-meals>
          {#each draftMeals as meal, index (meal.key)}
            <li class="grid gap-3 rounded border border-[var(--color-border)] p-3 sm:grid-cols-[minmax(0,1fr)_auto]" data-daily-diet-meal={meal.item.id}>
              <div class="grid min-w-0 gap-1">
                <h3 class="truncate font-medium">{index + 1}. {meal.item.name}</h3>
                <p class="text-xs text-[var(--color-muted)]">{meal.item.macroBasis === "100ml" ? "Values per 100 ml" : "Values per 100 g"}</p>
                <div class="grid gap-2 sm:grid-cols-[minmax(0,9rem)_auto]">
                  <label class="grid gap-1 text-xs text-[var(--color-muted)]" for={`daily-diet-quantity-${meal.key}`}>
                    Quantity
                    <input
                      id={`daily-diet-quantity-${meal.key}`}
                      class="rounded border border-[#E0E0E0] bg-white px-2 py-2 text-sm text-[#111827] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
                      type="number"
                      min="0.01"
                      step="0.01"
                      value={meal.quantity}
                      aria-label={`Quantity for ${meal.item.name}`}
                      oninput={(event) => updateQuantity(meal.key, event)}
                      disabled={!canEdit}
                    />
                  </label>
                  <label class="grid gap-1 text-xs text-[var(--color-muted)]" for={`daily-diet-unit-${meal.key}`}>
                    Unit
                    <select
                      id={`daily-diet-unit-${meal.key}`}
                      class="rounded border border-[#E0E0E0] bg-white px-2 py-2 text-sm text-[#111827] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
                      value={meal.unit}
                      aria-label={`Unit for ${meal.item.name}`}
                      onchange={(event) => updateUnit(meal.key, event)}
                      disabled={!canEdit}
                    >
                      <option value={displayUnitForBasis(meal.item.macroBasis, $preferencesStore.unitSystem)}>
                        {unitLabel(displayUnitForBasis(meal.item.macroBasis, $preferencesStore.unitSystem))}
                      </option>
                    </select>
                  </label>
                </div>
              </div>
              <div class="flex flex-wrap items-start gap-2 sm:justify-end">
                <button type="button" class="rounded border px-2 py-2 text-sm transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" aria-label={`Move ${meal.item.name} up`} onclick={() => moveMeal(meal.key, -1)} disabled={!canEdit || index === 0}>↑</button>
                <button type="button" class="rounded border px-2 py-2 text-sm transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" aria-label={`Move ${meal.item.name} down`} onclick={() => moveMeal(meal.key, 1)} disabled={!canEdit || index === draftMeals.length - 1}>↓</button>
                <button type="button" class="rounded border border-[var(--color-error)] px-2 py-2 text-sm transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" aria-label={`Remove ${meal.item.name}`} onclick={() => removeMeal(meal.key)} disabled={!canEdit}>Remove</button>
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
          <div><dt class="text-[var(--color-muted)]">Calories</dt><dd data-macro-calories>{formatDisplayQuantity(aggregate.calories)} kcal</dd></div>
        </dl>
        {#if savedDiet}
          <p class="text-xs text-[var(--color-muted)]" role="status" data-daily-diet-server-total>Totals confirmed by the server.</p>
        {/if}
      </section>

      {#if draftError || collectionError}
        <p class="rounded border border-[var(--color-error)] px-3 py-2 text-sm" role="alert" data-daily-diet-save-error>
          {draftError ?? collectionError?.message}
        </p>
      {/if}

      <div class="flex flex-wrap gap-2">
        <button
          type="submit"
          class="rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)] transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:cursor-not-allowed"
          disabled={!canEdit || draftMeals.length < 2 || $dailyDietStore.mutation !== "idle"}
          data-daily-diet-save
        >
          {$dailyDietStore.mutation === "creating" || $dailyDietStore.mutation === "replacing" ? "Saving…" : editingDietId ? "Update Daily Diet" : "Save Daily Diet"}
        </button>
        <button type="button" class="rounded border px-3 py-2 text-sm transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" onclick={resetDraft} disabled={!canEdit || draftMeals.length === 0}>
          Clear draft
        </button>
      </div>
    </form>

    {#if $dailyDietStore.status === "loading" && $dailyDietStore.collections.length === 0}
      <p class="rounded border border-[var(--color-border)] px-3 py-3 text-sm text-[var(--color-muted)]" role="status" data-saved-daily-diets-loading>
        Loading saved Daily Diets…
      </p>
    {:else if $dailyDietStore.status === "error" && $dailyDietStore.collections.length === 0}
      <div class="grid gap-2 rounded border border-[var(--color-error)] px-3 py-3" role="alert" data-saved-daily-diets-error>
        <p>{$dailyDietStore.error?.message ?? "Saved Daily Diets could not be loaded."}</p>
        <button type="button" class="w-fit rounded border px-3 py-2 text-sm transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" onclick={() => void loadDailyDiets()}>
          Try again
        </button>
      </div>
    {:else if $dailyDietStore.collections.length === 0}
      <p class="rounded border border-dashed border-[var(--color-border)] px-3 py-3 text-sm text-[var(--color-muted)]" data-saved-daily-diets-empty>
        No saved Daily Diets yet.
      </p>
    {:else}
      <section class="grid gap-2" aria-label="Saved Daily Diets" data-saved-daily-diets>
        <h2 class="font-data text-xs uppercase text-[var(--color-muted)]">Saved Daily Diets</h2>
        <ul class="grid gap-2">
          {#each $dailyDietStore.collections as diet (diet.id)}
            <li class="rounded border border-[var(--color-border)] p-3" data-saved-daily-diet={diet.id}>
              <div class="flex flex-wrap items-start justify-between gap-2">
                <div>
                  <h3 class="font-medium">{diet.name}</h3>
                  <p class="text-xs text-[var(--color-muted)]">{diet.entries.length} meals · {formatDisplayQuantity(diet.aggregateMacros.calories)} kcal</p>
                </div>
                <span class="font-data text-xs text-[var(--color-muted)]">{formatDisplayQuantity(diet.aggregateMacros.protein)}g protein</span>
              </div>
            </li>
          {/each}
        </ul>
      </section>
    {/if}
  {/if}
</section>
