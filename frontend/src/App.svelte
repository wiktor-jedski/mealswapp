<script lang="ts">
  import SearchShell from "./lib/components/SearchShell.svelte";
  import { registerServiceWorker } from "./lib/cache/service-worker";
  import { initTheme } from "./lib/stores/theme";

  // Implements DESIGN-016 ThemeProvider and DESIGN-011 ServiceWorkerCache bootstrap.
  initTheme();
  registerServiceWorker({ enabled: import.meta.env.PROD });
</script>

<!-- Implements DESIGN-001 SearchView shell composition. -->
<SearchShell />
