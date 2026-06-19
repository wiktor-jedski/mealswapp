<script lang="ts">
  import { searchStore, addSubstitutionInput, removeSubstitutionInput, updateSubstitutionInput } from "../stores/search";
  import type { SubstitutionUnit } from "../api/generated";

  // Implements DESIGN-001 SearchView Substitution Input composition (quantity-bearing accumulation and removal).

  /**
   * Canonical units accepted by substitution search inputs, mirroring the generated `SubstitutionUnit` union.
   */
  const unitOptions: { value: SubstitutionUnit; label: string }[] = [
    { value: "g", label: "g" },
    { value: "ml", label: "ml" },
    { value: "oz", label: "oz" },
    { value: "fl_oz", label: "fl oz" }
  ];

  /** Draft form state for the next Substitution Input. Autocomplete (Task 145) replaces the raw foodObjectId input. */
  let draftFoodObjectId = "";
  let draftQuantity = 100;
  let draftUnit: SubstitutionUnit = "g";

  /** User-facing message for duplicate or invalid add attempts; deterministic feedback per the verification criteria. */
  let draftMessage = "";

  /**
   * Adds one Substitution Input on Enter or Add-button press. Deterministic duplicate handling:
   * if the foodObjectId already exists, the add is rejected with a message and the store is not touched.
   * Quantities must be positive finite numbers; canonical units reach `SearchRequest.substitutionInputs` via the store.
   */
  function addInput(): void {
    const trimmedId = draftFoodObjectId.trim();
    if (trimmedId.length === 0) {
      draftMessage = "Enter a food object id.";
      return;
    }
    if (draftQuantity <= 0 || !Number.isFinite(draftQuantity)) {
      draftMessage = "Quantity must be a positive number.";
      return;
    }
    if ($searchStore.substitutionInputs.some((existing) => existing.foodObjectId === trimmedId)) {
      draftMessage = `Duplicate food object "${trimmedId}" is already added.`;
      return;
    }
    addSubstitutionInput({ foodObjectId: trimmedId, quantity: draftQuantity, unit: draftUnit });
    draftFoodObjectId = "";
    draftQuantity = 100;
    draftUnit = "g";
    draftMessage = "";
  }

  /** Enter on the foodObjectId input (stand-in for selected autocomplete + Enter) adds one Substitution Input. */
  function onFoodObjectIdKeydown(event: KeyboardEvent): void {
    if (event.key === "Enter") {
      event.preventDefault();
      addInput();
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
</script>

<!-- Implements DESIGN-001 SearchView Substitution Input controls. -->
<section class="grid gap-3 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4" aria-label="Substitution inputs">
  <div class="grid gap-2 sm:grid-cols-[1fr_auto_auto_auto]">
    <label class="sr-only" for="substitution-food-object-id">Food object id</label>
    <input
      id="substitution-food-object-id"
      class="rounded border border-[var(--color-border)] bg-transparent px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
      type="text"
      placeholder="Food object id (autocomplete arrives in Task 145)"
      bind:value={draftFoodObjectId}
      on:keydown={onFoodObjectIdKeydown}
    />
    <label class="sr-only" for="substitution-quantity">Quantity</label>
    <input
      id="substitution-quantity"
      class="rounded border border-[var(--color-border)] bg-transparent px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
      type="number"
      min="0"
      step="1"
      bind:value={draftQuantity}
    />
    <label class="sr-only" for="substitution-unit">Unit</label>
    <select
      id="substitution-unit"
      class="rounded border border-[var(--color-border)] bg-transparent px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
      bind:value={draftUnit}
    >
      {#each unitOptions as option}
        <option value={option.value}>{option.label}</option>
      {/each}
    </select>
    <button
      type="button"
      class="rounded border border-[var(--color-border)] px-3 py-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
      on:click={addInput}
    >
      Add
    </button>
  </div>

  {#if draftMessage}
    <p class="text-sm text-[var(--color-muted)]" role="status">{draftMessage}</p>
  {/if}

  {#if $searchStore.substitutionInputs.length > 0}
    <ul class="grid gap-2">
      {#each $searchStore.substitutionInputs as input (input.foodObjectId)}
        <li class="grid gap-2 sm:grid-cols-[1fr_auto_auto_auto] items-center">
          <span class="text-sm font-medium" data-food-object-id={input.foodObjectId}>{input.foodObjectId}</span>
          <label class="sr-only" for={`qty-${input.foodObjectId}`}>Quantity for {input.foodObjectId}</label>
          <input
            id={`qty-${input.foodObjectId}`}
            class="rounded border border-[var(--color-border)] bg-transparent px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
            type="number"
            min="0"
            step="1"
            value={input.quantity}
            on:input={(event) => onRowQuantityInput(input.foodObjectId, event)}
          />
          <label class="sr-only" for={`unit-${input.foodObjectId}`}>Unit for {input.foodObjectId}</label>
          <select
            id={`unit-${input.foodObjectId}`}
            class="rounded border border-[var(--color-border)] bg-transparent px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
            value={input.unit}
            on:change={(event) => onRowUnitChange(input.foodObjectId, event)}
          >
            {#each unitOptions as option}
              <option value={option.value}>{option.label}</option>
            {/each}
          </select>
          <button
            type="button"
            class="rounded border border-[var(--color-border)] px-3 py-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
            on:click={() => removeSubstitutionInput(input.foodObjectId)}
          >
            Remove
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</section>
