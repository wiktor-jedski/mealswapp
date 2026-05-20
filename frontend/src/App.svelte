<script lang="ts">
  import AdminView from './lib/components/AdminView.svelte';
  import ErrorBoundary from './lib/components/ErrorBoundary.svelte';
  import SearchView from './lib/components/SearchView.svelte';
  import ThemeProvider from './lib/components/ThemeProvider.svelte';
  const adminRoute = typeof window !== 'undefined' && window.location.pathname.startsWith('/admin');
</script>

<ThemeProvider>
  <ErrorBoundary>
    {#if adminRoute}
      <AdminView />
    {:else}
      <SearchView />
    {/if}
  </ErrorBoundary>
</ThemeProvider>
