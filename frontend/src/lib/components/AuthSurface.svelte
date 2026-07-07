<script lang="ts">
  import DisclaimerPanel from "./DisclaimerPanel.svelte";
  import OAuthEntryPoint from "./OAuthEntryPoint.svelte";
  import RegisterView from "./RegisterView.svelte";

  // Implements DESIGN-018 AuthView shell for disclaimer and OAuth entry surfaces.

  interface Props {
    callbackReturn?: boolean;
    registerMode?: boolean;
  }

  let { callbackReturn = false, registerMode = false }: Props = $props();
</script>

<!-- Implements DESIGN-018 AuthView scoped auth surface with registration entry and existing disclaimer/OAuth surfaces. -->
<main class="mx-auto flex min-h-screen w-full max-w-3xl flex-col gap-6 px-4 py-8 sm:px-6" data-auth-surface>
  {#if registerMode}
    <RegisterView onSwitchToLogin={() => { window.location.assign("/auth"); }} />
  {:else}
    <header class="space-y-2">
      <h1 class="text-2xl font-semibold">Sign in</h1>
      <p class="text-sm leading-6 text-[var(--color-muted)]">
        Use a secure provider session to continue.
      </p>
    </header>
    <DisclaimerPanel />
    <OAuthEntryPoint {callbackReturn} />
  {/if}
</main>
