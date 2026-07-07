<script lang="ts">
  import { onMount } from "svelte";
  import type { OAuthProvider } from "../api/generated";
  import { refreshOAuthCallbackSession, startOAuthProvider } from "./oauth-entry-point";

  // Implements DESIGN-018 OAuthEntryPoint Google provider entry and callback-return refresh UI.

  interface Props {
    callbackReturn?: boolean;
    mode?: "login" | "register";
  }

  let { callbackReturn = false, mode = "login" }: Props = $props();
  let heading = $derived(mode === "register" ? "Register with a provider" : "Sign in with a provider");
  let busyProvider = $state<OAuthProvider | null>(null);
  let refreshing = $state(false);
  let message = $state<string | null>(null);

  const providers: Array<{ id: OAuthProvider; label: string }> = [
    { id: "google", label: "Google" }
  ];

  onMount(() => {
    if (!callbackReturn) {
      return;
    }
    refreshing = true;
    const controller = new AbortController();
    refreshOAuthCallbackSession(window.location.href, undefined, controller.signal)
      .then(() => {
        message = "Sign-in session refreshed.";
      })
      .catch(() => {
        message = "We could not finish sign-in. Please try again.";
      })
      .finally(() => {
        refreshing = false;
      });
    return () => controller.abort();
  });

  function onProviderClick(provider: OAuthProvider): void {
    busyProvider = provider;
    message = null;
    const result = startOAuthProvider(provider);
    if (!result.ok) {
      message = result.errorMessage ?? "This sign-in provider is temporarily unavailable. Please try another sign-in method.";
      busyProvider = null;
    }
  }
</script>

<!-- Implements DESIGN-018 OAuthEntryPoint backend provider start actions and user-safe unavailable messaging. -->
<section class="space-y-3" aria-labelledby="oauth-entry-heading" data-oauth-entry>
  <h2 id="oauth-entry-heading" class="text-base font-semibold">{heading}</h2>
  <div class="grid gap-2 sm:grid-cols-2">
    {#each providers as provider}
      <button
        type="button"
        class="flex h-10 w-full items-center justify-center gap-3 rounded border border-[#747775] bg-white px-3 text-sm font-medium leading-5 text-[#1f1f1f] transition-colors hover:bg-[#f8fafd] focus:outline focus:outline-2 focus:outline-offset-2 focus:outline-[var(--color-primary)] active:bg-[#f1f3f4] disabled:cursor-not-allowed disabled:opacity-60"
        disabled={refreshing || busyProvider !== null}
        aria-busy={busyProvider === provider.id ? "true" : "false"}
        data-oauth-provider={provider.id}
        onclick={() => onProviderClick(provider.id)}
      >
        <svg aria-hidden="true" class="h-[18px] w-[18px] shrink-0" viewBox="0 0 18 18">
          <path
            fill="#4285F4"
            d="M17.64 9.20455c0-.63818-.05727-1.25273-.16364-1.84364H9v3.48182h4.84364c-.20864 1.125-.84273 2.07818-1.79591 2.71636v2.25818h2.90864C16.65818 14.25273 17.64 11.94545 17.64 9.20455z"
          />
          <path
            fill="#34A853"
            d="M9 18c2.43 0 4.46727-.80636 5.95636-2.18273l-2.90864-2.25818c-.80636.54-1.83727.85909-3.04773.85909-2.34409 0-4.32818-1.58318-5.03682-3.71045H.95727v2.33182C2.43818 15.98182 5.48182 18 9 18z"
          />
          <path
            fill="#FBBC05"
            d="M3.96318 10.70773c-.18-.54-.28227-1.11727-.28227-1.70773s.10227-1.16773.28227-1.70773V4.96045H.95727C.34773 6.17591 0 7.55227 0 9s.34773 2.82409.95727 4.03955l3.00591-2.33182z"
          />
          <path
            fill="#EA4335"
            d="M9 3.58182c1.32136 0 2.50773.45409 3.44045 1.34591l2.58136-2.58136C13.46318.89182 11.42591 0 9 0 5.48182 0 2.43818 2.01818.95727 4.96045l3.00591 2.33182C4.67182 5.165 6.65591 3.58182 9 3.58182z"
          />
        </svg>
        <span class="whitespace-nowrap">{provider.label}</span>
      </button>
    {/each}
  </div>
  {#if refreshing}
    <p role="status" aria-live="polite" class="text-sm text-[var(--color-muted)]" data-oauth-refreshing>
      Refreshing sign-in session…
    </p>
  {/if}
  {#if message}
    <p role="status" aria-live="polite" class="text-sm text-[var(--color-muted)]" data-oauth-message>{message}</p>
  {/if}
</section>
