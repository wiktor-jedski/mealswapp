<script lang="ts">
  import { QueryClient, QueryClientProvider } from "@tanstack/svelte-query";
  import AuthSurface from "./lib/components/AuthSurface.svelte";
  import SearchShell from "./lib/components/SearchShell.svelte";
  import { registerServiceWorker } from "./lib/cache/service-worker";
  import { initTheme, cleanupTheme } from "./lib/stores/theme";
  import { initPreferences } from "./lib/stores/preferences";
  import { initOffline, cleanupOffline } from "./lib/stores/offline";
  import { initAuthSessionStore, probeAuthSession } from "./lib/stores/auth-session";

  // Implements DESIGN-001 SearchView SPA bootstrap: TanStack Query context, theme, preferences, offline lifecycle, and auth session probe.
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        refetchOnWindowFocus: false
      }
    }
  });

  // Implements DESIGN-016 ThemeProvider, DESIGN-001 SidebarComponent unit preference persistence, DESIGN-001 OfflineBanner lifecycle, and DESIGN-018 AuthSessionStore startup probe.
  $effect(() => {
    initPreferences();
    initTheme();
    initOffline();
    initAuthSessionStore();
    void probeAuthSession();
    return () => {
      cleanupTheme();
      cleanupOffline();
    };
  });

  registerServiceWorker({ enabled: import.meta.env.PROD });

  // Implements DESIGN-018 AuthView route selection for OAuth callback return handling and registration entry.
  let isAuthRoute = $state(isCurrentAuthRoute());
  let isOAuthCallbackRoute = $state(isCurrentOAuthCallbackRoute());
  let isRegisterRoute = $state(isCurrentRegisterRoute());

  $effect(() => {
    isAuthRoute = isCurrentAuthRoute();
    isOAuthCallbackRoute = isCurrentOAuthCallbackRoute();
    isRegisterRoute = isCurrentRegisterRoute();
  });

  function isCurrentAuthRoute(): boolean {
    const path = window.location.pathname;
    return path === "/auth" || path === "/auth/callback" || path === "/auth/register";
  }

  function isCurrentOAuthCallbackRoute(): boolean {
    return window.location.pathname === "/auth/callback";
  }

  function isCurrentRegisterRoute(): boolean {
    return window.location.pathname === "/auth/register";
  }
</script>

<!-- Implements DESIGN-001 SearchView shell composition with TanStack Query context. -->
<QueryClientProvider client={queryClient}>
  {#if isAuthRoute}
    <AuthSurface callbackReturn={isOAuthCallbackRoute} registerMode={isRegisterRoute} />
  {:else}
    <SearchShell />
  {/if}
</QueryClientProvider>
