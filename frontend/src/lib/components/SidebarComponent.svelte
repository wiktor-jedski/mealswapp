<script lang="ts">
  import type { MacroToggles, SearchMode, TagFilter } from '../api/types';
  import { createEntitlementViewState, type ModeGate } from '../entitlements/entitlementState';
  import { defaultDietaryOptions, type SidebarAction } from '../search/sidebarState';

  interface Props {
    mode: SearchMode;
    enabledMacros: MacroToggles;
    filters: TagFilter[];
    modeGates?: Record<SearchMode, ModeGate>;
    usageLabel?: string;
    upgradePrompt?: string;
    authenticated?: boolean;
    onModeChange?: (mode: SearchMode) => void;
    onMacroChange?: (macro: keyof MacroToggles, enabled: boolean) => void;
    onFiltersChange?: (filters: TagFilter[]) => void;
    onAction?: (action: SidebarAction) => void;
  }

  const modes: Array<[SearchMode, string]> = [
    ['single', 'Single item'],
    ['replacement', 'Replacement'],
    ['diet', 'Diet']
  ];
  const macroLabels: Array<[keyof MacroToggles, string]> = [
    ['protein', 'Protein'],
    ['carbs', 'Carbs'],
    ['fat', 'Fat']
  ];

  let {
    mode,
    enabledMacros,
    filters,
    modeGates = createEntitlementViewState().modeGates,
    usageLabel = createEntitlementViewState().usageLabel,
    upgradePrompt,
    authenticated = false,
    onModeChange,
    onMacroChange,
    onFiltersChange,
    onAction
  }: Props = $props();
  let collapsed = $state(false);

  function toggleFilter(id: string) {
    const existing = filters.find((filter) => filter.tagId === id);
    if (existing) {
      onFiltersChange?.(filters.filter((filter) => filter.tagId !== id));
      return;
    }
    const kind = id.startsWith('allergen-') ? 'allergen' : 'diet';
    onFiltersChange?.([...filters, { tagId: id, kind, include: kind === 'diet' }]);
  }

  function isFilterActive(id: string) {
    return filters.some((filter) => filter.tagId === id);
  }
</script>

<aside class="sm:col-span-3">
  <button
    class="mb-3 w-full rounded border border-secondary bg-surface px-3 py-2 text-left text-sm text-text-primary sm:hidden"
    type="button"
    aria-expanded={!collapsed}
    onclick={() => {
      collapsed = !collapsed;
    }}
  >
    Controls
  </button>

  <div class:hidden={collapsed} class="grid gap-3 sm:block">
    <nav aria-label="Search modes" class="rounded border border-secondary bg-surface p-3">
      <p class="font-mono text-xs font-medium uppercase text-text-muted">Mode</p>
      <div class="mt-3 grid gap-2">
        {#each modes as [itemMode, label]}
          <button
            class="rounded border px-3 py-2 text-left text-sm transition-all duration-200"
            class:border-primary={mode === itemMode}
            class:border-secondary={mode !== itemMode}
            class:bg-primary={mode === itemMode}
            class:text-white={mode === itemMode}
            class:text-text-primary={mode !== itemMode}
            class:font-bold={mode === itemMode}
            type="button"
            disabled={modeGates[itemMode]?.locked}
            aria-disabled={modeGates[itemMode]?.locked}
            aria-describedby={modeGates[itemMode]?.locked ? `mode-${itemMode}-locked` : undefined}
            onclick={() => onModeChange?.(itemMode)}
          >
            <span>{label}</span>
            {#if modeGates[itemMode]?.locked}
              <span id={`mode-${itemMode}-locked`} class="block font-mono text-xs text-text-muted">Upgrade</span>
            {/if}
          </button>
        {/each}
      </div>
      <p class="mt-3 font-mono text-xs text-text-muted">{usageLabel}</p>
      {#if upgradePrompt}
        <button class="mt-2 rounded border border-accent px-3 py-2 text-left text-sm text-text-primary" type="button" onclick={() => onAction?.('settings')}>
          {upgradePrompt}
        </button>
      {/if}
    </nav>

    <section aria-labelledby="macro-heading" class="rounded border border-secondary bg-surface p-3">
      <p id="macro-heading" class="font-mono text-xs font-medium uppercase text-text-muted">Macros</p>
      <div class="mt-3 grid gap-2">
        {#each macroLabels as [key, label]}
          <label class="flex items-center justify-between gap-3 rounded border border-secondary px-3 py-2 text-sm">
            <span>{label}</span>
            <input
              type="checkbox"
              checked={enabledMacros[key]}
              oninput={(event) => onMacroChange?.(key, event.currentTarget.checked)}
            />
          </label>
        {/each}
      </div>
    </section>

    <section aria-labelledby="dietary-heading" class="rounded border border-secondary bg-surface p-3">
      <p id="dietary-heading" class="font-mono text-xs font-medium uppercase text-text-muted">Dietary</p>
      <div class="mt-3 flex flex-wrap gap-2">
        {#each defaultDietaryOptions as option}
          <button
            class="rounded border px-3 py-2 text-sm"
            class:border-primary={isFilterActive(option.id)}
            class:bg-secondary={isFilterActive(option.id)}
            class:border-secondary={!isFilterActive(option.id)}
            type="button"
            onclick={() => toggleFilter(option.id)}
          >
            {option.label}
          </button>
        {/each}
      </div>
    </section>

    <section aria-labelledby="saved-heading" class="rounded border border-secondary bg-surface p-3">
      <p id="saved-heading" class="font-mono text-xs font-medium uppercase text-text-muted">Saved</p>
      <div class="mt-3 grid gap-2">
        <button class="rounded border border-secondary px-3 py-2 text-left text-sm" type="button" onclick={() => onAction?.('saved-searches')}>Saved searches</button>
        <button class="rounded border border-secondary px-3 py-2 text-left text-sm" type="button" onclick={() => onAction?.('history')}>History</button>
        <button class="rounded border border-secondary px-3 py-2 text-left text-sm" type="button" onclick={() => onAction?.('favorites')}>Favorites</button>
      </div>
    </section>

    <section aria-labelledby="account-heading" class="rounded border border-secondary bg-surface p-3">
      <p id="account-heading" class="font-mono text-xs font-medium uppercase text-text-muted">Account</p>
      <div class="mt-3 grid gap-2">
        <button class="rounded border border-secondary px-3 py-2 text-left text-sm" type="button" onclick={() => onAction?.('profile')}>
          {authenticated ? 'Profile' : 'Sign in'}
        </button>
        <button class="rounded border border-secondary px-3 py-2 text-left text-sm" type="button" onclick={() => onAction?.('settings')}>Settings</button>
      </div>
    </section>
  </div>
</aside>
