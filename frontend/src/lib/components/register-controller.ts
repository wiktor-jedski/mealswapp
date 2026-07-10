import type { AppError, RegisterRequest } from "../api/generated";
import type { AuthSessionProjection } from "../stores/auth-session";

// Implements DESIGN-018 RegisterView and ConsentGate registration validation and submission orchestration.

/** Current legal versions required before email/password registration can be submitted. */
export interface ConsentVersions {
	privacyPolicyVersion: string;
	termsVersion: string;
	effectiveAt: string;
}

/** Form state owned by the registration surface. */
export interface RegisterFormState {
	email: string;
	password: string;
	confirmPassword: string;
	privacyPolicyVersion?: string;
	termsVersion?: string;
	privacyAccepted: boolean;
	termsAccepted: boolean;
	submitting: boolean;
	error?: AppError;
}

/** User-safe registration field errors rendered by RegisterView. */
export interface RegisterValidationResult {
	email?: string;
	password?: string;
	confirmPassword?: string;
	consent?: string;
}

/** Side-effect boundaries used by RegisterView and tests. */
export interface RegisterControllerDependencies {
	registerWithEmail: (request: RegisterRequest, signal?: AbortSignal) => Promise<AuthSessionProjection>;
	loadConsentVersions: (signal?: AbortSignal) => Promise<ConsentVersions>;
}

/** Result of one registration submission attempt after client and server validation. */
export interface RegisterSubmitResult {
	status: "registered" | "invalid" | "duplicate_email" | "consent_stale" | "unverified" | "locked" | "error";
	session?: AuthSessionProjection;
	error?: AppError;
	consentVersions?: ConsentVersions;
	validation: RegisterValidationResult;
}

/** Bundled consent versions used until the backend exposes a dedicated current-consent endpoint. */
export const DEFAULT_CONSENT_VERSIONS: ConsentVersions = {
	privacyPolicyVersion: "dev-privacy-v1",
	termsVersion: "dev-terms-v1",
	effectiveAt: "2026-07-05T00:00:00Z"
};

/** Returns a fresh copy of the bundled current consent versions. */
export async function loadCurrentConsentVersions(_signal?: AbortSignal): Promise<ConsentVersions> {
	return { ...DEFAULT_CONSENT_VERSIONS };
}

/** Builds the initial registration form state from the current consent versions. */
export function createRegisterFormState(consent: ConsentVersions): RegisterFormState {
	return {
		email: "",
		password: "",
		confirmPassword: "",
		privacyPolicyVersion: consent.privacyPolicyVersion,
		termsVersion: consent.termsVersion,
		privacyAccepted: false,
		termsAccepted: false,
		submitting: false
	};
}

/** Determines whether the current form has all client-side requirements needed to submit. */
export function canSubmitRegistration(state: RegisterFormState, consent: ConsentVersions): boolean {
	return Object.keys(validateRegistration(state, consent)).length === 0 && !state.submitting;
}

/** Validates registration without echoing raw password content. */
export function validateRegistration(
	state: RegisterFormState,
	consent: ConsentVersions
): RegisterValidationResult {
	const errors: RegisterValidationResult = {};
	const email = state.email.trim();
	if (!email || !email.includes("@")) {
		errors.email = "Enter a valid email address.";
	}
	if (state.password.length < 12) {
		errors.password = "Use at least 12 characters.";
	}
	if (state.password !== state.confirmPassword) {
		errors.confirmPassword = "Passwords do not match.";
	}
	if (
		!state.privacyAccepted ||
		!state.termsAccepted ||
		state.privacyPolicyVersion !== consent.privacyPolicyVersion ||
		state.termsVersion !== consent.termsVersion
	) {
		errors.consent = "Accept the current Privacy Policy and Terms of Service.";
	}
	return errors;
}

/** Submits registration, clears raw password fields, and maps expected auth outcomes for the UI. */
export async function submitRegistration(
	state: RegisterFormState,
	consent: ConsentVersions,
	dependencies: RegisterControllerDependencies,
	signal?: AbortSignal
): Promise<RegisterSubmitResult> {
	const validation = validateRegistration(state, consent);
	if (Object.keys(validation).length > 0) {
		return { status: "invalid", validation };
	}

	const request: RegisterRequest = {
		email: state.email.trim(),
		password: state.password,
		privacyPolicyVersion: consent.privacyPolicyVersion,
		termsVersion: consent.termsVersion
	};

	try {
		const session = await dependencies.registerWithEmail(request, signal);
		return {
			status: session.hasVerifiedLoginMethod === false ? "unverified" : "registered",
			session,
			validation: {}
		};
	} catch (error) {
		const appError = extractAppError(error);
		if (isDuplicateEmail(appError)) {
			return { status: "duplicate_email", error: appError, validation: {} };
		}
		if (isStaleConsent(appError)) {
			const consentVersions = await dependencies.loadConsentVersions(signal);
			return {
				status: "consent_stale",
				error: appError,
				consentVersions,
				validation: { consent: "Legal terms changed. Review and accept the current versions." }
			};
		}
		if (isUnverifiedLoginMethod(appError)) {
			return { status: "unverified", error: appError, validation: {} };
		}
		if (isAccountHold(appError)) {
			return { status: "locked", error: appError, validation: {} };
		}
		return { status: "error", error: appError ?? fallbackError(), validation: {} };
	} finally {
		state.password = "";
		state.confirmPassword = "";
		request.password = "";
	}
}

function extractAppError(error: unknown): AppError | undefined {
	if (typeof error !== "object" || error === null || !("appError" in error)) {
		return undefined;
	}
	const appError = (error as { appError?: unknown }).appError;
	if (typeof appError !== "object" || appError === null) {
		return undefined;
	}
	return appError as AppError;
}

function isDuplicateEmail(error: AppError | undefined): boolean {
	return Boolean(error?.code === "duplicate_email" || error?.code === "email_already_registered");
}

function isStaleConsent(error: AppError | undefined): boolean {
	return Boolean(
		error?.code === "consent_stale" ||
			error?.code === "consent_version_stale" ||
			error?.code === "stale_consent_versions"
	);
}

function isUnverifiedLoginMethod(error: AppError | undefined): boolean {
	return Boolean(
		error?.code === "unverified_login_method" ||
			error?.code === "login_method_unverified" ||
			error?.code === "email_unverified"
	);
}

function isAccountHold(error: AppError | undefined): boolean {
	return Boolean(
		error?.code === "account_locked" ||
			error?.code === "account_hold" ||
			error?.code === "admin_hold" ||
			error?.code === "compliance_hold" ||
			error?.code === "account_disabled"
	);
}

function fallbackError(): AppError {
	return {
		category: "unknown",
		code: "registration_failed",
		message: "Registration is temporarily unavailable. Please try again.",
		retryable: true
	};
}
