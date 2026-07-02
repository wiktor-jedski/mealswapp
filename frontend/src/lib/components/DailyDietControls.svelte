<script lang="ts">
  import { searchStore, setDailyDietId } from "../stores/search";
  import type { SearchRejection } from "../api/generated";

  // Implements DESIGN-001 SearchView Daily Diet Alternative controls and Phase 04 structured rejection display.

  /**
   * Phase 04 structured rejection surface. Task 151 wires the actual `SearchRejection` envelope from the
   * 422 response; until then the component reads `searchStore.error` as the rejection message so the UI
   * shape is in place without creating Phase 07 job or worker behavior.
   */
  let {
    rejection = null,
    executionAllowed = true
  }: {
    rejection?: SearchRejection | null;
    executionAllowed?: boolean;
  } = $props();

  /** UUID-shaped validation pattern for the daily diet id input. */
  const dailyDietIdPattern = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$";

  /** Updates the daily diet id in the store, clearing it when the input is empty. */
  function onDailyDietIdInput(event: Event): void {
    const value = (event.currentTarget as HTMLInputElement).value;
    setDailyDietId(value.length > 0 ? value : undefined);
  }
</script>

<!-- Implements DESIGN-001 SearchView Daily Diet Alternative controls and Phase 04 structured rejection display. -->
<section class="grid gap-3 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4" aria-label="Daily diet alternative controls">
  <label class="text-sm font-medium" for="daily-diet-id">Daily diet id</label>
  <input
    id="daily-diet-id"
    class="rounded border border-[var(--color-border)] bg-transparent px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
    type="text"
    placeholder="00000000-0000-0000-0000-000000000000"
    pattern={dailyDietIdPattern}
    value={$searchStore.dailyDietId ?? ""}
    aria-disabled={!executionAllowed}
    oninput={onDailyDietIdInput}
  />

  {#if rejection || $searchStore.error}
    <div class="rounded border border-[var(--color-border)] p-3" role="alert" aria-label="Search rejection">
      {#if rejection}
        <p class="text-sm font-medium" data-rejection-code={rejection?.code}>{rejection?.code}</p>
        <p class="text-sm" data-rejection-message>{rejection?.message}</p>
        {#if rejection?.field}
          <p class="text-sm text-[var(--color-muted)]" data-rejection-field>{rejection?.field}</p>
        {/if}
      {:else}
        <p class="text-sm" data-rejection-message>{$searchStore.error}</p>
      {/if}
    </div>
  {/if}
</section>
