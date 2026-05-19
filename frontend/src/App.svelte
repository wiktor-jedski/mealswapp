<script lang="ts">
  const macroToggles = ['Protein', 'Carbs', 'Fat'];
</script>

<main class="min-h-screen bg-background text-text-primary">
  <div class="mx-auto grid max-w-app grid-cols-1 gap-4 px-4 py-6 sm:grid-cols-12">
    <aside class="sm:col-span-3">
      <nav aria-label="Search modes" class="rounded border border-secondary bg-surface p-3">
        <p class="font-mono text-xs font-medium uppercase text-text-muted">Mode</p>
        <div class="mt-3 grid gap-2">
          <button class="rounded border border-primary bg-primary px-3 py-2 text-left text-sm font-bold text-white transition-all duration-200">
            Single item
          </button>
          <button class="rounded border border-secondary px-3 py-2 text-left text-sm text-text-primary transition-all duration-200">
            Replacement
          </button>
          <button class="rounded border border-secondary px-3 py-2 text-left text-sm text-text-primary transition-all duration-200">
            Diet
          </button>
        </div>
      </nav>
    </aside>

    <section class="sm:col-span-9" aria-labelledby="search-heading">
      <h1 id="search-heading" class="text-2xl font-bold">Mealswapp</h1>
      <div class="mt-4 rounded border border-secondary bg-surface p-4">
        <label for="search" class="font-mono text-sm font-medium text-text-muted">Search food</label>
        <input
          id="search"
          class="mt-2 w-full rounded border border-[#E0E0E0] bg-white px-3 py-2 text-text-primary outline-none focus:border-primary"
          placeholder="Start with a food item"
          type="search"
        />
        <div class="mt-4 flex flex-wrap gap-2" aria-label="Enabled macros">
          {#each macroToggles as macro}
            <span class="rounded bg-secondary px-2 py-1 font-mono text-xs font-medium text-primary">{macro}</span>
          {/each}
        </div>
      </div>
    </section>
  </div>
</main>
