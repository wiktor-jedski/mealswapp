// Generated from api/openapi.yaml by scripts/generate-api-types.py.
// Implements DESIGN-017 ErrorMessageMapper shared frontend contracts.

export type ErrorCategory =
	| "validation"
	| "auth"
	| "entitlement"
	| "network"
	| "timeout"
	| "server"
	| "dependency"
	| "unknown";

// Implements DESIGN-017 ErrorMessageMapper AppError contract.
/** User-safe classified server error returned by the API gateway. */
export interface AppError {
	category: ErrorCategory;
	code: string;
	message: string;
	retryable: boolean;
	requestId?: string;
}

// Implements DESIGN-017 GlobalExceptionHandler response envelope.
/** Shared API response wrapper with request correlation metadata. */
export interface Envelope<TData extends Record<string, unknown> = Record<string, unknown>> {
	status: string;
	requestId: string;
	data?: TData;
	error?: AppError | null;
}

// Implements DESIGN-014 UptimeMonitor liveness contract.
/** Process liveness payload. */
export interface HealthData extends Record<string, unknown> {
	service: string;
}

// Implements DESIGN-014 UptimeMonitor readiness contract.
/** Dependency-readiness payload. */
export interface ReadinessData extends Record<string, unknown> {
	checks: Record<string, string>;
}

// Implements DESIGN-006 AuthController CSRF token-delivery contract.
/** Session-bound synchronizer token delivered to SPA clients. */
export interface CSRFTokenData extends Record<string, unknown> {
	csrfToken: string;
}

// Implements DESIGN-006 AuthController CSRF token-delivery contract.
/** CSRF token response envelope. */
export type CSRFTokenEnvelope = Envelope<CSRFTokenData>;

// Implements DESIGN-006 AuthController frontend auth contract.
/** Authenticated session metadata; token values are carried only by HttpOnly cookies. */
export interface AuthSessionData extends Record<string, unknown> {
	userId: string;
	role: "user" | "admin";
	hasVerifiedLoginMethod: boolean;
	accessExpiresAt: string;
	refreshExpiresAt: string;
}

// Implements DESIGN-006 AuthController frontend auth contract.
/** Authenticated session response envelope. */
export type AuthSessionEnvelope = Envelope<AuthSessionData>;

// Implements DESIGN-006 AuthController frontend registration contract.
/** Registration request accepted by the account API. */
export interface RegisterRequest {
	email: string;
	password: string;
	privacyPolicyVersion: string;
	termsVersion: string;
}

// Implements DESIGN-006 AuthController frontend login contract.
/** Email/password login request. */
export interface LoginRequest {
	email: string;
	password: string;
}

// Implements DESIGN-006 AuthController frontend verification contract.
/** Verification completion payload. */
export interface VerifyEmailData extends Record<string, unknown> {
	hasVerifiedLoginMethod: true;
}

// Implements DESIGN-006 AuthController frontend verification contract.
/** Verification completion response envelope. */
export type VerifyEmailEnvelope = Envelope<VerifyEmailData>;

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password reset request that never reveals account existence. */
export interface PasswordResetRequest {
	email: string;
}

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Single-use password reset token consumption request. */
export interface PasswordResetConsumeRequest {
	token: string;
	newPassword: string;
}

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password-reset request acceptance payload. */
export interface PasswordResetAcceptedData extends Record<string, unknown> {
	accepted: true;
}

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password-reset request acceptance envelope. */
export type PasswordResetRequestEnvelope = Envelope<PasswordResetAcceptedData>;

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password-reset completion payload. */
export interface PasswordResetConsumeData extends Record<string, unknown> {
	reset: true;
}

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password-reset completion envelope. */
export type PasswordResetConsumeEnvelope = Envelope<PasswordResetConsumeData>;

// Implements DESIGN-006 OAuthHandler frontend provider contract.
/** Supported OAuth identity providers. */
export type OAuthProvider = "google" | "apple";

// Implements DESIGN-008 PreferenceManager frontend profile contract.
/** User profile and preference response data. */
export interface ProfileData extends Record<string, unknown> {
	userId: string;
	displayName: string;
	unitSystem: "metric" | "imperial";
	themePreference: "system" | "light" | "dark";
	requiresUnitRecalculation: boolean;
}

// Implements DESIGN-008 PreferenceManager frontend profile contract.
/** User profile response envelope. */
export type ProfileEnvelope = Envelope<ProfileData>;

// Implements DESIGN-008 PreferenceManager frontend profile contract.
/** Mutable profile preference request. */
export interface ProfileUpdateRequest {
	displayName?: string;
	unitSystem: "metric" | "imperial";
	themePreference: "system" | "light" | "dark";
}

// Implements DESIGN-008 SavedDataRepository frontend saved-data contract.
/** One saved favorite, meal, or reserved diet reference. */
export interface SavedItem {
	id: string;
	itemId: string;
	kind: "favorite" | "saved_meal" | "saved_diet";
}

// Implements DESIGN-008 SavedDataRepository frontend saved-data contract.
/** Saved item query filter. */
export type SavedItemKind = SavedItem["kind"];

// Implements DESIGN-008 SavedDataRepository frontend saved-data contract.
/** Saved item collection payload. */
export interface SavedItemsData extends Record<string, unknown> {
	items: SavedItem[];
}

// Implements DESIGN-008 SavedDataRepository frontend saved-data contract.
/** Saved item collection response envelope. */
export type SavedItemsEnvelope = Envelope<SavedItemsData>;

// Implements DESIGN-008 SearchHistoryRepository frontend history contract.
/** One decrypted search-history entry at the API boundary. */
export interface SearchHistoryEntry {
	id: string;
	query: string;
	mode: string;
	filtersHash: string;
}

// Implements DESIGN-008 SearchHistoryRepository frontend history contract.
/** Search-history collection payload. */
export interface SearchHistoryData extends Record<string, unknown> {
	history: SearchHistoryEntry[];
}

// Implements DESIGN-008 SearchHistoryRepository frontend history contract.
/** Search-history collection response envelope. */
export type SearchHistoryEnvelope = Envelope<SearchHistoryData>;

// Implements DESIGN-008 DataExporter frontend export contract.
/** JSON account export bundle. */
export interface ExportBundle {
	user: Record<string, unknown>;
	consent: Array<Record<string, unknown>>;
	savedItems: SavedItem[];
	history: SearchHistoryEntry[];
	customItems: Array<Record<string, unknown>>;
}

// Implements DESIGN-008 DataExporter frontend export contract.
/** Supported account export formats. */
export type ExportFormat = "json" | "csv";

// Implements DESIGN-008 AccountDeleter frontend deletion contract.
/** Account deletion request response data. */
export interface DeletionRequestData extends Record<string, unknown> {
	requestId: string;
	status: "pending" | "processing" | "completed" | "failed";
}

// Implements DESIGN-008 AccountDeleter frontend deletion contract.
/** Account deletion response envelope. */
export type DeletionRequestEnvelope = Envelope<DeletionRequestData>;

// Implements DESIGN-015 DisclaimerRenderer frontend disclaimer contract.
/** Stable Markdown disclaimer content for login and account surfaces. */
export interface DisclaimerData extends Record<string, unknown> {
	location: "login" | "account";
	version: string;
	markdown: string;
	fallback: boolean;
	alert?: string;
}

// Implements DESIGN-015 DisclaimerRenderer frontend disclaimer contract.
/** Disclaimer response envelope. */
export type DisclaimerEnvelope = Envelope<DisclaimerData>;
