<script lang="ts">
  import { searchStore, toggleMacro } from "../stores/search";
  import { preferencesStore, setUnitSystem } from "../stores/preferences";
  import type { MacroToggleKey } from "../stores/search";
  import type { UnitSystem } from "../stores/preferences";

  // Implements DESIGN-001 SettingsPanel macro toggle and unit preference controls.

  const macros: { key: MacroToggleKey; id: string; label: string }[] = [
    { key: "protein", id: "macro-protein", label: "Protein" },
    { key: "carbohydrates", id: "macro-carbohydrates", label: "Carbohydrates" },
    { key: "fat", id: "macro-fat", label: "Fat" }
  ];

  const unitSystems: { value: UnitSystem; id: string; label: string }[] = [
    { value: "metric", id: "unit-metric", label: "Metric" },
    { value: "imperial", id: "unit-imperial", label: "Imperial" }
  ];
</script>

<!-- Implements DESIGN-001 SettingsPanel -->
<section
  class="grid gap-5 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4"
  aria-label="Search settings"
>
  <fieldset class="grid gap-3">
    <legend class="font-data text-xs uppercase text-[var(--color-muted)]">Macro display</legend>
    {#each macros as macro}
      <div class="flex items-center gap-2">
        <input
          id={macro.id}
          type="checkbox"
          class="h-4 w-4 rounded border-[var(--color-border)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
          checked={$searchStore.enabledMacros[macro.key]}
          on:change={() => toggleMacro(macro.key)}
        />
        <label class="text-sm font-medium" for={macro.id}>{macro.label}</label>
      </div>
    {/each}
  </fieldset>

  <fieldset class="grid gap-3">
    <legend class="font-data text-xs uppercase text-[var(--color-muted)]">Unit system</legend>
    {#each unitSystems as unit}
      <div class="flex items-center gap-2">
        <input
          id={unit.id}
          type="radio"
          name="unit-system"
          value={unit.value}
          class="h-4 w-4 border-[var(--color-border)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
          checked={$preferencesStore.unitSystem === unit.value}
          on:change={() => setUnitSystem(unit.value)}
        />
        <label class="text-sm font-medium" for={unit.id}>{unit.label}</label>
      </div>
    {/each}
  </fieldset>
</section>
