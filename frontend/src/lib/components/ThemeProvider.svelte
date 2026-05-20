<script lang="ts">
  import { onMount } from 'svelte';
  import {
    createThemeController,
    type MediaQueryListLike,
    type ThemePreference,
    type ThemeState
  } from '../theme/theme';

  interface Props {
    initialPreference?: ThemePreference;
    syncPreference?: (preference: ThemePreference) => Promise<void> | void;
    children?: import('svelte').Snippet;
  }

  let { initialPreference, syncPreference, children }: Props = $props();
  let state: ThemeState = $state({ preference: 'system', resolved: 'light', systemTheme: 'light' });
  let setThemePreference: (preference: ThemePreference) => Promise<ThemeState> = async () => state;

  onMount(() => {
    const query = window.matchMedia('(prefers-color-scheme: dark)');
    const controller = createThemeController({ syncPreference });
    state = controller.getState();
    if (initialPreference) {
      void controller.setThemePreference(initialPreference).then((next) => {
        state = next;
      });
    }
    setThemePreference = async (preference: ThemePreference) => {
      state = await controller.setThemePreference(preference);
      return state;
    };

    const onChange = (event: { matches: boolean }) => {
      state = controller.handleSystemTheme(event.matches ? 'dark' : 'light');
    };
    addSystemThemeListener(query, onChange);
    return () => removeSystemThemeListener(query, onChange);
  });

  function addSystemThemeListener(query: MediaQueryListLike, callback: (event: { matches: boolean }) => void) {
    query.addEventListener?.('change', callback);
  }

  function removeSystemThemeListener(query: MediaQueryListLike, callback: (event: { matches: boolean }) => void) {
    query.removeEventListener?.('change', callback);
  }
</script>

<svelte:head>
  <meta name="color-scheme" content={state.resolved} />
</svelte:head>

<div data-theme-preference={state.preference} data-theme-resolved={state.resolved}>
  {@render children?.()}
</div>
