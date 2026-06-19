<script lang="ts">
  import { onDestroy } from "svelte";
  import SearchShell from "./lib/components/SearchShell.svelte";
  import { registerServiceWorker } from "./lib/cache/service-worker";
  import { initTheme } from "./lib/stores/theme";
  import { QueryClient, QueryClientProvider } from "@tanstack/svelte-query";

  // Implements DESIGN-016 ThemeProvider and DESIGN-011 ServiceWorkerCache bootstrap.
  const destroyTheme = initTheme();
  const queryClient = new QueryClient();
  onDestroy(destroyTheme);
  registerServiceWorker({ enabled: import.meta.env.PROD });
</script>

<!-- Implements DESIGN-001 SearchView shell composition. -->
<QueryClientProvider client={queryClient}>
  <SearchShell />
</QueryClientProvider>
