<script lang="ts">
  import { tick } from "svelte";

  import { AuthClientError } from "../api/auth-client";
  import type { AppError, LoginRequest } from "../api/generated";
  import { loginWithEmail } from "../stores/auth-session";
  import { runQueuedProtectedActionAfterAuth } from "../stores/auth-surface";

  // Implements DESIGN-018 LoginView email/password login, safe feedback, lockout metadata, duplicate-submit prevention, and successful-session handoff.

  const maxLoginRetryAfterSeconds = 60 * 60;
  interface LoginFormState {
    email: string;
    password: string;
    submitting: boolean;
    error?: AppError;
    retryAfterSeconds?: number;
  }

  const genericInvalidCredentialMessage = "Email or password is incorrect.";
  let emailInput: HTMLInputElement;
  let email = $state("");
  let password = $state("");
  let submitting = $state(false);
  let errorMessage = $state("");
  let retryAfterSeconds = $state<number | undefined>(undefined);
  let formErrorId = "login-form-error";

  let canSubmit = $derived(email.trim().length > 0 && password.length > 0 && !submitting);

  $effect(() => {
    void tick().then(() => emailInput?.focus());
  });

  /** Submits a generated LoginRequest and clears raw password text after every attempt. */
  async function submitLogin(): Promise<void> {
    if (!canSubmit) {
      errorMessage = "Enter your email and password.";
      return;
    }

    submitting = true;
    errorMessage = "";
    retryAfterSeconds = undefined;
    const request: LoginRequest = {
      email: email.trim(),
      password
    };

    try {
      await loginWithEmail(request);
      password = "";
      await runQueuedProtectedActionAfterAuth();
    } catch (error) {
      password = "";
      const safeFeedback = mapLoginFeedback(error);
      errorMessage = safeFeedback.message;
      retryAfterSeconds = safeFeedback.retryAfterSeconds;
    } finally {
      request.password = "";
      submitting = false;
    }
  }

  function mapLoginFeedback(error: unknown): { message: string; retryAfterSeconds?: number } {
    if (error instanceof AuthClientError) {
      if (isAccountHold(error)) {
        return { message: "This account is locked by an administrative or compliance hold. Contact support to restore access." };
      }
      if (error.status === 401 || error.appError.code === "invalid_credentials") {
        return { message: genericInvalidCredentialMessage };
      }
      if (error.status === 429 || error.status === 403 || error.appError.code.includes("lock")) {
        return {
          message: error.appError.message,
          retryAfterSeconds: normalizeRetryAfterSeconds(error.retryAfterSeconds)
        };
      }
      return { message: error.appError.message };
    }
    return { message: "Login is temporarily unavailable. Please try again." };
  }

  function isAccountHold(error: AuthClientError): boolean {
    return (
      error.status === 423 ||
      ["account_hold", "admin_hold", "compliance_hold", "account_disabled"].includes(error.appError.code)
    );
  }

  function normalizeRetryAfterSeconds(seconds: number | undefined): number | undefined {
    if (seconds === undefined || !Number.isFinite(seconds) || seconds < 0) {
      return undefined;
    }
    return Math.min(Math.ceil(seconds), maxLoginRetryAfterSeconds);
  }
</script>

<!-- Implements DESIGN-018 LoginView accessible email/password form with focusable controls and safe auth feedback. -->
<form
  class="grid gap-4"
  aria-labelledby="login-view-title"
  aria-describedby={errorMessage ? formErrorId : undefined}
  data-login-view
  onsubmit={(event) => {
    event.preventDefault();
    void submitLogin();
  }}
>
  <div>
    <h2 id="login-view-title" class="text-lg font-bold text-[var(--color-text)]">Sign in</h2>
    <p class="text-sm text-[var(--color-muted)]">Continue with your Mealswapp account.</p>
  </div>

  <label class="grid gap-1 text-sm font-semibold text-[var(--color-text)]" for="login-email">
    Email
    <input
      id="login-email"
      bind:this={emailInput}
      bind:value={email}
      class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-[var(--color-text)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
      autocomplete="email"
      inputmode="email"
      name="email"
      required
      type="email"
    />
  </label>

  <label class="grid gap-1 text-sm font-semibold text-[var(--color-text)]" for="login-password">
    Password
    <input
      id="login-password"
      bind:value={password}
      class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-[var(--color-text)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
      autocomplete="current-password"
      name="password"
      required
      type="password"
    />
  </label>

  {#if errorMessage}
    <div
      id={formErrorId}
      class="rounded border border-[var(--color-error)] px-3 py-2 text-sm text-[var(--color-error)]"
      role="alert"
    >
      <p>{errorMessage}</p>
      {#if retryAfterSeconds !== undefined}
        <p class="mt-1 font-[var(--font-data)]">Try again in {retryAfterSeconds} seconds.</p>
      {/if}
    </div>
  {/if}

  <button
    type="submit"
    class="rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)] transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:cursor-wait disabled:opacity-70"
    disabled={!canSubmit}
  >
    {submitting ? "Signing in..." : "Sign in"}
  </button>
</form>
