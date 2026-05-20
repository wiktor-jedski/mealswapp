<script lang="ts">
  import ErrorBoundary from './lib/components/ErrorBoundary.svelte';
  import SearchView from './lib/components/SearchView.svelte';
  import ThemeProvider from './lib/components/ThemeProvider.svelte';
</script>

<ThemeProvider>
  <ErrorBoundary>
    <SearchView />
  </ErrorBoundary>
</ThemeProvider>
