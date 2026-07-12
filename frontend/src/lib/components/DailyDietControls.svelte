<script lang="ts">
  import type { SearchRejection } from "../api/generated";
  import { clearDailyDietState, dailyDietStore, loadDailyDiets, selectDailyDiet } from "../stores/daily-diet";
  import type { AuthStatus } from "../stores/auth-session";
  import { formatDisplayQuantity } from "../units";
  import OptimizationWorkflow from "./OptimizationWorkflow.svelte";

  // Implements DESIGN-001 SearchView Daily Diet Alternative saved-collection selector.
  // Implements DESIGN-008 SavedDataRepository server-derived collection and macro projection.

  let {
    rejection = null,
    authStatus = "unknown",
    authenticated = false,
    userId = null,
    executionAllowed = true,
    entitlementFeedback = null,
    onSignIn = () => undefined
  }: {
    rejection?: SearchRejection | null;
    authStatus?: AuthStatus;
    authenticated?: boolean;
    userId?: string | null;
    executionAllowed?: boolean;
    entitlementFeedback?: string | null;
    onSignIn?: () => void;
  } = $props();

  let loadedUserId = $state<string | null>(null);

  $effect(() => {
    if (authStatus === "authenticated" && authenticated && userId && loadedUserId !== userId) {
      if (loadedUserId !== null) clearDailyDietState();
      loadedUserId = userId;
      void loadDailyDiets().catch(() => undefined);
      return;
    }
    if (!authenticated && loadedUserId !== null) {
      loadedUserId = null;
      clearDailyDietState();
    }
  });
</script>

<!-- Implements DESIGN-001 SearchView Daily Diet Alternative controls and structured rejection display. -->
<section class="grid gap-3 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4" aria-labelledby="daily-diet-alternative-title" data-daily-diet-alternative-controls>
  <div class="grid gap-1">
    <h2 id="daily-diet-alternative-title" class="text-lg font-semibold">Choose a saved Daily Diet</h2>
    <p class="text-sm text-[var(--color-muted)]">Select the one-day collection to use as the alternative-search input.</p>
  </div>

  {#if authStatus === "unknown" || authStatus === "authenticating"}
    <p class="rounded border border-[var(--color-border)] px-3 py-3 text-sm text-[var(--color-muted)]" role="status" data-daily-diet-alternative-auth-loading>
      Checking your sign-in status…
    </p>
  {:else if !authenticated}
    <div class="grid gap-3 rounded border border-[var(--color-border)] px-3 py-3" data-daily-diet-alternative-auth-guidance>
      <p class="text-sm">Sign in to use a saved Daily Diet for alternatives.</p>
      <button type="button" class="w-fit rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" onclick={onSignIn}>
        Sign in to continue
      </button>
    </div>
  {:else}
    {#if entitlementFeedback}
      <p class="rounded border border-[var(--color-accent)] px-3 py-2 text-sm" role="alert" data-daily-diet-alternative-entitlement>
        {entitlementFeedback}
      </p>
    {/if}

    {#if rejection}
      <div class="rounded border border-[var(--color-border)] p-3" role="alert" aria-label="Search rejection" data-daily-diet-alternative-rejection>
        <p class="text-sm font-medium" data-rejection-code={rejection.code}>{rejection.code}</p>
        <p class="text-sm" data-rejection-message>{rejection.message}</p>
        {#if rejection.field}<p class="text-sm text-[var(--color-muted)]" data-rejection-field>{rejection.field}</p>{/if}
      </div>
    {/if}

    {#if $dailyDietStore.loading && $dailyDietStore.collections.length === 0}
      <p class="rounded border border-[var(--color-border)] px-3 py-3 text-sm text-[var(--color-muted)]" role="status" data-daily-diet-alternative-loading>
        Loading saved Daily Diets…
      </p>
    {:else if $dailyDietStore.status === "error" && $dailyDietStore.collections.length === 0}
      <div class="grid gap-2 rounded border border-[var(--color-error)] px-3 py-3" role="alert" data-daily-diet-alternative-error>
        <p>{$dailyDietStore.error?.message ?? "Saved Daily Diets could not be loaded."}</p>
        <button type="button" class="w-fit rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" onclick={() => void loadDailyDiets()}>
          Try again
        </button>
      </div>
    {:else if $dailyDietStore.collections.length === 0}
      <p class="rounded border border-dashed border-[var(--color-border)] px-3 py-3 text-sm text-[var(--color-muted)]" data-daily-diet-alternative-empty>
        No saved Daily Diets yet. Build one in Daily Diet first.
      </p>
    {:else}
      <div class="grid gap-2" role="radiogroup" aria-label="Saved Daily Diet choices" data-daily-diet-choices>
        {#each $dailyDietStore.collections as diet (diet.id)}
          <button
            type="button"
            class="grid gap-1 rounded border p-3 text-left focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] {($dailyDietStore.selectedId === diet.id ? 'border-[var(--color-primary)]' : 'border-[var(--color-border)]')}"
            role="radio"
            aria-checked={$dailyDietStore.selectedId === diet.id}
            aria-label={`Use ${diet.name} as Daily Diet Alternative input`}
            onclick={() => selectDailyDiet(diet.id)}
            disabled={!executionAllowed}
            data-daily-diet-choice={diet.id}
          >
            <span class="font-medium">{diet.name}</span>
            <span class="font-data text-xs text-[var(--color-muted)]">{diet.entries.length} meals · {formatDisplayQuantity(diet.aggregateMacros.protein)}g protein · {formatDisplayQuantity(diet.aggregateMacros.calories)} kcal</span>
          </button>
        {/each}
      </div>
      {#if $dailyDietStore.selectedId}
        <p class="text-sm text-[var(--color-muted)]" role="status" data-daily-diet-alternative-selected>
          Selected collection is ready for Daily Diet Alternative search.
        </p>
        <OptimizationWorkflow selectedDietId={$dailyDietStore.selectedId} executionAllowed={executionAllowed} />
      {/if}
    {/if}
  {/if}

  {#if rejection === null && $dailyDietStore.error && $dailyDietStore.collections.length > 0}
    <p class="rounded border border-[var(--color-error)] px-3 py-2 text-sm" role="alert" data-daily-diet-alternative-error-message>{$dailyDietStore.error.message}</p>
  {/if}
</section>
