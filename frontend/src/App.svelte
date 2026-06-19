<script lang="ts">
  import { QueryClient, QueryClientProvider } from "@tanstack/svelte-query";
  import SearchShell from "./lib/components/SearchShell.svelte";
  import { registerServiceWorker } from "./lib/cache/service-worker";
  import { initTheme, cleanupTheme } from "./lib/stores/theme";
  import { initPreferences } from "./lib/stores/preferences";
  import { initOffline, cleanupOffline } from "./lib/stores/offline";

  // Implements DESIGN-001 SearchView SPA bootstrap: TanStack Query context, theme, preferences, and offline lifecycle.
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        refetchOnWindowFocus: false
      }
    }
  });

  // Implements DESIGN-016 ThemeProvider, DESIGN-001 SettingsPanel persistence, and DESIGN-001 OfflineBanner lifecycle.
  $effect(() => {
    initPreferences();
    initTheme();
    initOffline();
    return () => {
      cleanupTheme();
      cleanupOffline();
    };
  });

  registerServiceWorker({ enabled: import.meta.env.PROD });
</script>

<!-- Implements DESIGN-001 SearchView shell composition with TanStack Query context. -->
<QueryClientProvider client={queryClient}>
  <SearchShell />
</QueryClientProvider>
