<script lang="ts">
	import { tick } from "svelte";
	import { registerWithEmail as registerSessionWithEmail } from "../stores/auth-session";
	import type { AuthSessionProjection } from "../stores/auth-session";
	import {
		DEFAULT_CONSENT_VERSIONS,
		createRegisterFormState,
		loadCurrentConsentVersions,
		submitRegistration,
		type ConsentVersions,
		type RegisterControllerDependencies,
		type RegisterValidationResult
	} from "./register-controller";

	// Implements DESIGN-018 RegisterView email/password account creation and ConsentGate.

	interface Props {
		initialConsentVersions?: ConsentVersions;
		registerWithEmail?: RegisterControllerDependencies["registerWithEmail"];
		loadConsentVersions?: RegisterControllerDependencies["loadConsentVersions"];
		onRegistered?: (session: AuthSessionProjection) => void;
		onSwitchToLogin?: () => void;
	}

	let {
		initialConsentVersions = DEFAULT_CONSENT_VERSIONS,
		registerWithEmail = registerSessionWithEmail,
		loadConsentVersions = loadCurrentConsentVersions,
		onRegistered,
		onSwitchToLogin
	}: Props = $props();

	// svelte-ignore state_referenced_locally
	let consentVersions = $state<ConsentVersions>({ ...initialConsentVersions });
	// svelte-ignore state_referenced_locally
	let form = $state(createRegisterFormState(consentVersions));
	let validation = $state<RegisterValidationResult>({});
	let statusMessage = $state("");
	let alertMessage = $state("");
	let unverifiedLoginMethod = $state(false);
	let emailInput: HTMLInputElement;
	let submitDisabled = $derived(
		form.email.trim().length === 0 ||
			form.password.length === 0 ||
			form.confirmPassword.length === 0 ||
			!form.privacyAccepted ||
			!form.termsAccepted ||
			form.privacyPolicyVersion !== consentVersions.privacyPolicyVersion ||
			form.termsVersion !== consentVersions.termsVersion ||
		form.submitting
	);

	$effect(() => {
		void tick().then(() => emailInput?.focus());
	});

	/** Applies the single UI consent checkbox to both required legal-version acceptances. */
	function setCombinedConsent(accepted: boolean): void {
		form.privacyPolicyVersion = consentVersions.privacyPolicyVersion;
		form.termsVersion = consentVersions.termsVersion;
		form.privacyAccepted = accepted;
		form.termsAccepted = accepted;
	}

	/** Runs local validation, server registration, stale-consent recovery, and successful-session handoff. */
	async function handleSubmit(): Promise<void> {
		form.submitting = true;
		alertMessage = "";
		statusMessage = "";
		unverifiedLoginMethod = false;

		const result = await submitRegistration(
			form,
			consentVersions,
			{ registerWithEmail, loadConsentVersions }
		);

		form.submitting = false;
		validation = result.validation;

		if (result.status === "invalid") {
			alertMessage = "";
			return;
		}
		if (result.status === "duplicate_email") {
			alertMessage = "An account already exists for this email. Switch to login to continue.";
			return;
		}
		if (result.status === "consent_stale" && result.consentVersions) {
			consentVersions = result.consentVersions;
			form.privacyPolicyVersion = consentVersions.privacyPolicyVersion;
			form.termsVersion = consentVersions.termsVersion;
			form.privacyAccepted = false;
			form.termsAccepted = false;
			alertMessage = "Privacy Policy or Terms changed. Review and accept the current versions.";
			return;
		}
		if (result.status === "error") {
			alertMessage = result.error?.message ?? "Registration is temporarily unavailable. Please try again.";
			return;
		}
		if (result.session) {
			unverifiedLoginMethod = result.status === "unverified";
			statusMessage = "Registration complete. Your browser session is authenticated.";
			onRegistered?.(result.session);
		}
	}

</script>

<!-- Implements DESIGN-018 RegisterView registration form, duplicate-email feedback, and successful-session handoff. -->
<section
	class="grid gap-4"
	aria-labelledby="register-title"
	data-register-view
>
	<div>
		<h1 id="register-title" class="text-xl font-bold text-[var(--color-text)]">Create account</h1>
	</div>

	{#if alertMessage}
		<div class="grid gap-2 rounded border border-[var(--color-error)] p-3" role="alert">
			<p class="text-sm text-[var(--color-error)]">{alertMessage}</p>
			{#if alertMessage.includes("Switch to login") && onSwitchToLogin}
				<button
					type="button"
					class="w-fit rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
					onclick={onSwitchToLogin}
				>
					Log in instead
				</button>
			{/if}
		</div>
	{/if}

	{#if statusMessage}
		<p class="rounded border border-[var(--color-primary)] bg-[var(--color-secondary)] px-3 py-2 text-sm text-[var(--color-text)]" role="status">
			{statusMessage}
		</p>
	{/if}

	{#if unverifiedLoginMethod}
		<p class="rounded border border-[var(--color-border)] px-3 py-2 text-sm text-[var(--color-muted)]" role="status">
			Verify your email before using features that require a verified login method.
		</p>
	{/if}

	<form class="grid gap-4" onsubmit={(event) => { event.preventDefault(); void handleSubmit(); }}>
		<label class="grid gap-1 text-sm font-semibold text-[var(--color-text)]">
			Email
			<input
				bind:this={emailInput}
				class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-[var(--color-text)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
				name="email"
				type="email"
				autocomplete="email"
				bind:value={form.email}
				aria-invalid={validation.email ? "true" : "false"}
				aria-describedby={validation.email ? "register-email-error" : undefined}
			/>
		</label>
		{#if validation.email}
			<p id="register-email-error" class="text-sm text-[var(--color-error)]">{validation.email}</p>
		{/if}

		<label class="grid gap-1 text-sm font-semibold text-[var(--color-text)]">
			Password
			<input
				class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-[var(--color-text)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
				name="password"
				type="password"
				autocomplete="new-password"
				bind:value={form.password}
				aria-invalid={validation.password ? "true" : "false"}
				aria-describedby={validation.password ? "register-password-error" : undefined}
			/>
		</label>
		{#if validation.password}
			<p id="register-password-error" class="text-sm text-[var(--color-error)]">{validation.password}</p>
		{/if}

		<label class="grid gap-1 text-sm font-semibold text-[var(--color-text)]">
			Confirm password
			<input
				class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-[var(--color-text)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]"
				name="confirmPassword"
				type="password"
				autocomplete="new-password"
				bind:value={form.confirmPassword}
				aria-invalid={validation.confirmPassword ? "true" : "false"}
				aria-describedby={validation.confirmPassword ? "register-confirm-error" : undefined}
			/>
		</label>
		{#if validation.confirmPassword}
			<p id="register-confirm-error" class="text-sm text-[var(--color-error)]">{validation.confirmPassword}</p>
		{/if}

		<fieldset class="grid gap-3">
			<legend class="pb-1 text-sm font-bold text-[var(--color-text)]">Consent</legend>
			<label class="flex gap-2 text-sm text-[var(--color-text)]">
				<input
					name="legalConsentAccepted"
					type="checkbox"
					checked={form.privacyAccepted && form.termsAccepted}
					aria-describedby={validation.consent ? "register-consent-error" : undefined}
					onchange={(event) => setCombinedConsent((event.currentTarget as HTMLInputElement).checked)}
				/>
				<span>
					I accept the current
					<a class="font-semibold underline underline-offset-2" href="/privacy" target="_blank" rel="noreferrer">Privacy Policy</a>
					and
					<a class="font-semibold underline underline-offset-2" href="/terms" target="_blank" rel="noreferrer">Terms of Service</a>.
				</span>
			</label>
			{#if validation.consent}
				<p id="register-consent-error" class="text-sm text-[var(--color-error)]">{validation.consent}</p>
			{/if}
		</fieldset>

		<button
			type="submit"
			class="rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] disabled:cursor-not-allowed disabled:opacity-60"
			disabled={submitDisabled}
		>
			{form.submitting ? "Creating account..." : "Create account"}
		</button>
	</form>
</section>
