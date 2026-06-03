<script lang="ts">
  import { themePreference, setThemePreference } from "../stores/theme";

  // Implements DESIGN-001 SidebarComponent search-mode shell labels.
  const modes = ["Catalog", "Substitution", "Daily Diet"];
</script>

<!-- Implements DESIGN-001 SearchView, SidebarComponent, SettingsPanel, and DESIGN-016 LayoutGrid. -->
<main class="min-h-screen">
  <section class="mx-auto grid min-h-screen max-w-6xl gap-6 px-4 py-6 sm:grid-cols-[220px_1fr] sm:px-6">
    <aside class="border-b border-[var(--color-border)] pb-4 sm:border-b-0 sm:border-r sm:pb-0 sm:pr-5">
      <h1 class="text-2xl font-semibold">Mealswapp</h1>
      <nav class="mt-6 grid gap-2" aria-label="Search modes">
        {#each modes as mode, index}
          <button
            class="rounded border border-[var(--color-border)] px-3 py-2 text-left text-sm font-medium"
            aria-current={index === 0 ? "page" : undefined}
          >
            {mode}
          </button>
        {/each}
      </nav>
    </aside>

    <div class="flex flex-col gap-5">
      <header class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="font-data text-xs uppercase text-[var(--color-muted)]">Phase 00 Shell</p>
          <h2 class="mt-1 text-xl font-semibold">Search foundation</h2>
        </div>
        <select
          class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm"
          bind:value={$themePreference}
          on:change={(event) => setThemePreference(event.currentTarget.value as "system" | "light" | "dark")}
          aria-label="Theme preference"
        >
          <option value="system">System</option>
          <option value="light">Light</option>
          <option value="dark">Dark</option>
        </select>
      </header>

      <form class="grid gap-3 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
        <label class="text-sm font-medium" for="search">Food search</label>
        <input
          id="search"
          class="rounded border border-[var(--color-border)] bg-transparent px-3 py-2"
          placeholder="Search will be implemented in Phase 05"
          disabled
        />
      </form>
    </div>
  </section>
</main>
