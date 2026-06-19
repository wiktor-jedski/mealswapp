<script lang="ts">
  import type { SettingsStore, SearchSettings } from "../stores/settings";

  // Implements DESIGN-001 SettingsPanel typed settings binding.
  export let settings: SettingsStore;

  function updateMacro(name: keyof SearchSettings["enabledMacros"], event: Event) {
    settings.setMacro(name, (event.currentTarget as HTMLInputElement).checked);
  }
</script>

<!-- Implements DESIGN-001 SettingsPanel accessible macro and unit controls. -->
<section aria-labelledby="search-settings-title" class="grid gap-4 rounded border border-[var(--color-border)] p-4">
  <h3 id="search-settings-title" class="font-semibold">Search settings</h3>
  <fieldset class="grid gap-2">
    <legend class="text-sm font-medium">Displayed macros</legend>
    <label><input type="checkbox" checked={$settings.enabledMacros.protein} on:change={(event) => updateMacro("protein", event)} class="focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--color-primary)]" /> Protein</label>
    <label><input type="checkbox" checked={$settings.enabledMacros.carbohydrate} on:change={(event) => updateMacro("carbohydrate", event)} class="focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--color-primary)]" /> Carbohydrate</label>
    <label><input type="checkbox" checked={$settings.enabledMacros.fat} on:change={(event) => updateMacro("fat", event)} class="focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--color-primary)]" /> Fat</label>
  </fieldset>
  <label class="grid gap-1 text-sm font-medium" for="unit-system">
    Unit system
    <select id="unit-system" value={$settings.unitSystem} on:change={(event) => settings.setUnitSystem(event.currentTarget.value as SearchSettings["unitSystem"])} class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--color-primary)]">
      <option value="metric">Metric</option>
      <option value="imperial">Imperial</option>
    </select>
  </label>
</section>
