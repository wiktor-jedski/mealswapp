<script lang="ts">
  import type { MacroToggles, UserProfile } from '../api/types';
  import type { ThemePreference } from '../theme/theme';
  import {
    buildProfilePreferenceUpdate,
    canDeleteAccount,
    createSettingsState,
    updateMacroPreference,
    updateTheme,
    updateUnitSystem
  } from '../search/settingsPanel';

  interface Props {
    open: boolean;
    profile?: UserProfile;
    onClose?: () => void;
    onSavePreferences?: (update: Partial<UserProfile>) => void;
    onExport?: (format: 'json' | 'csv') => void;
    onDeleteAccount?: () => void;
  }

  const themeOptions: Array<[ThemePreference, string]> = [
    ['system', 'System'],
    ['light', 'Light'],
    ['dark', 'Dark']
  ];
  const macroLabels: Array<[keyof MacroToggles, string]> = [
    ['protein', 'Protein'],
    ['carbs', 'Carbs'],
    ['fat', 'Fat']
  ];

  let { open, profile, onClose, onSavePreferences, onExport, onDeleteAccount }: Props = $props();
  let settings = $state(createSettingsState());
  let loadedProfileId = $state<string | undefined>();

  $effect(() => {
    const profileId = profile?.id ?? profile?.userId;
    if (profileId && profileId !== loadedProfileId) {
      settings = createSettingsState(profile);
      loadedProfileId = profileId;
    }
  });
</script>

{#if open}
  <section class="mt-4 rounded border border-secondary bg-surface p-4" aria-labelledby="settings-heading">
    <div class="flex items-start justify-between gap-3">
      <div>
        <h2 id="settings-heading" class="text-lg font-bold">Settings</h2>
        <p class="text-sm text-text-muted">{profile?.displayName ?? profile?.email ?? 'Guest profile'}</p>
      </div>
      <button class="rounded border border-secondary px-3 py-2 text-sm" type="button" onclick={() => onClose?.()}>Close</button>
    </div>

    <div class="mt-4 grid gap-4 sm:grid-cols-2">
      <fieldset class="rounded border border-secondary p-3">
        <legend class="font-mono text-xs uppercase text-text-muted">Theme</legend>
        <div class="mt-3 grid gap-2">
          {#each themeOptions as [value, label]}
            <label class="flex items-center gap-2 text-sm">
              <input
                type="radio"
                name="theme"
                checked={settings.themePreference === value}
                onchange={() => {
                  settings = updateTheme(settings, value);
                }}
              />
              <span>{label}</span>
            </label>
          {/each}
        </div>
      </fieldset>

      <fieldset class="rounded border border-secondary p-3">
        <legend class="font-mono text-xs uppercase text-text-muted">Units</legend>
        <div class="mt-3 grid gap-2">
          <label class="flex items-center gap-2 text-sm">
            <input type="radio" name="unit" checked={settings.unitSystem === 'metric'} onchange={() => (settings = updateUnitSystem(settings, 'metric'))} />
            <span>Metric</span>
          </label>
          <label class="flex items-center gap-2 text-sm">
            <input type="radio" name="unit" checked={settings.unitSystem === 'imperial'} onchange={() => (settings = updateUnitSystem(settings, 'imperial'))} />
            <span>Imperial</span>
          </label>
        </div>
      </fieldset>

      <fieldset class="rounded border border-secondary p-3">
        <legend class="font-mono text-xs uppercase text-text-muted">Macros</legend>
        <div class="mt-3 grid gap-2">
          {#each macroLabels as [key, label]}
            <label class="flex items-center justify-between gap-3 text-sm">
              <span>{label}</span>
              <input
                type="checkbox"
                checked={settings.enabledMacros[key]}
                oninput={(event) => {
                  settings = updateMacroPreference(settings, key, event.currentTarget.checked);
                }}
              />
            </label>
          {/each}
        </div>
      </fieldset>

      <section class="rounded border border-secondary p-3" aria-labelledby="data-heading">
        <h3 id="data-heading" class="font-mono text-xs uppercase text-text-muted">Data</h3>
        <div class="mt-3 flex flex-wrap gap-2">
          <button class="rounded border border-secondary px-3 py-2 text-sm" type="button" onclick={() => onExport?.('json')}>Export JSON</button>
          <button class="rounded border border-secondary px-3 py-2 text-sm" type="button" onclick={() => onExport?.('csv')}>Export CSV</button>
        </div>
        <label class="mt-3 block text-sm">
          Confirm deletion
          <input
            class="mt-1 w-full rounded border border-secondary bg-surface px-3 py-2"
            value={settings.deleteConfirmation}
            oninput={(event) => {
              settings = { ...settings, deleteConfirmation: event.currentTarget.value };
            }}
          />
        </label>
        <button
          class="mt-2 rounded border border-error px-3 py-2 text-sm text-error disabled:opacity-50"
          type="button"
          disabled={!canDeleteAccount(settings.deleteConfirmation)}
          onclick={() => onDeleteAccount?.()}
        >
          Delete account
        </button>
      </section>
    </div>

    <div class="mt-4 rounded border border-secondary bg-background p-3 text-sm text-text-muted">
      <p>Mealswapp does not provide medical advice.</p>
      <p class="mt-2">Privacy policy, terms, and nutrition disclaimer consent are required for account features.</p>
    </div>

    <button class="mt-4 rounded bg-primary px-4 py-2 text-sm font-bold text-white" type="button" onclick={() => onSavePreferences?.(buildProfilePreferenceUpdate(settings))}>
      Save preferences
    </button>
  </section>
{/if}
