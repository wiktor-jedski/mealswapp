import { expect, test } from "bun:test";

import type { AppError } from "../api/generated";
import type { AuthSessionProjection } from "../stores/auth-session";
import {
	DEFAULT_CONSENT_VERSIONS,
	canSubmitRegistration,
	createRegisterFormState,
	submitRegistration,
	validateRegistration,
	type ConsentVersions
} from "./register-controller";

// Implements DESIGN-018 RegisterView and ConsentGate behavior verification.

const consent: ConsentVersions = DEFAULT_CONSENT_VERSIONS;

function validForm() {
	return {
		...createRegisterFormState(consent),
		email: "person@example.com",
		password: "correct-horse-1",
		confirmPassword: "correct-horse-1",
		privacyAccepted: true,
		termsAccepted: true
	};
}

function appError(code: string, message = "Auth request failed."): AppError {
	return { category: "validation", code, message, retryable: false };
}

function authError(error: AppError): Error & { appError: AppError } {
	return Object.assign(new Error(error.message), { appError: error });
}

function session(overrides: Partial<AuthSessionProjection> = {}): AuthSessionProjection {
	return {
		status: "authenticated",
		userId: "user-1",
		role: "user",
		hasVerifiedLoginMethod: true,
		...overrides
	};
}

test("registration cannot submit without current consent versions checked", () => {
	const form = validForm();
	form.termsAccepted = false;
	expect(canSubmitRegistration(form, consent)).toBe(false);
	expect(validateRegistration(form, consent).consent).toBe(
		"Accept the current Privacy Policy and Terms of Service."
	);

	form.termsAccepted = true;
	form.termsVersion = "terms-old-v1";
	expect(canSubmitRegistration(form, consent)).toBe(false);
	expect(validateRegistration(form, consent).consent).toBe(
		"Accept the current Privacy Policy and Terms of Service."
	);
});

test("password mismatch and policy failures are mapped without echoing password values", () => {
	const form = validForm();
	form.password = "short";
	form.confirmPassword = "different";

	const errors = validateRegistration(form, consent);

	expect(errors.password).toBe("Use at least 12 characters.");
	expect(errors.confirmPassword).toBe("Passwords do not match.");
	expect(JSON.stringify(errors)).not.toContain("short");
	expect(JSON.stringify(errors)).not.toContain("different");
});

test("duplicate email returns login-mode outcome and clears raw password fields", async () => {
	const form = validForm();
	let requestPassword = "";

	const result = await submitRegistration(form, consent, {
		registerWithEmail: (request) => {
			requestPassword = request.password;
			throw authError(appError("duplicate_email", "Email already registered."));
		},
		loadConsentVersions: async () => consent
	});

	expect(result.status).toBe("duplicate_email");
	expect(requestPassword).toBe("correct-horse-1");
	expect(form.password).toBe("");
	expect(form.confirmPassword).toBe("");
});

test("stale consent refreshes current versions and requires re-acceptance", async () => {
	const form = validForm();
	const refreshed = {
		privacyPolicyVersion: "privacy-2026-08",
		termsVersion: "terms-2026-08",
		effectiveAt: "2026-08-01T00:00:00Z"
	};

	const result = await submitRegistration(form, consent, {
		registerWithEmail: () => {
			throw authError(appError("consent_version_stale", "Legal terms changed."));
		},
		loadConsentVersions: async () => refreshed
	});

	expect(result.status).toBe("consent_stale");
	expect(result.consentVersions).toEqual(refreshed);
	expect(result.validation.consent).toBe("Legal terms changed. Review and accept the current versions.");
	expect(form.password).toBe("");
	expect(form.confirmPassword).toBe("");
});

test("successful registration returns an authenticated browser-session projection", async () => {
	const form = validForm();
	const result = await submitRegistration(form, consent, {
		registerWithEmail: async (request) => {
			expect(request).toEqual({
				email: "person@example.com",
				password: "correct-horse-1",
				privacyPolicyVersion: "dev-privacy-v1",
				termsVersion: "dev-terms-v1"
			});
			return session();
		},
		loadConsentVersions: async () => consent
	});

	expect(result.status).toBe("registered");
	expect(result.session?.status).toBe("authenticated");
	expect(JSON.stringify(result.session)).not.toMatch(/password|token/i);
	expect(form.password).toBe("");
	expect(form.confirmPassword).toBe("");
});

test("unverified login method restrictions are surfaced from server state", async () => {
	const form = validForm();
	const result = await submitRegistration(form, consent, {
		registerWithEmail: async () => session({ hasVerifiedLoginMethod: false }),
		loadConsentVersions: async () => consent
	});

	expect(result.status).toBe("unverified");
	expect(result.session?.hasVerifiedLoginMethod).toBe(false);
});
