// Generated from api/openapi.yaml by scripts/generate-api-types.py.
// Implements DESIGN-017 ErrorMessageMapper shared frontend contracts.

export type ErrorCategory =
	| "validation"
	| "auth"
	| "entitlement"
	| "security"
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
export interface Envelope<TData = unknown> {
	status: string;
	requestId: string;
	data?: TData;
	error?: AppError | null;
}

// Implements DESIGN-014 UptimeMonitor liveness contract.
/** Process liveness payload. */
export interface HealthData {
	service: string;
}

// Implements DESIGN-014 UptimeMonitor readiness contract.
/** Dependency-readiness payload. */
export interface ReadinessData {
	checks: Record<string, string>;
}

// Implements DESIGN-006 AuthController CSRF token-delivery contract.
/** Session-bound synchronizer token delivered to SPA clients. */
export interface CSRFTokenData {
	csrfToken: string;
}

// Implements DESIGN-006 AuthController CSRF token-delivery contract.
/** CSRF token response envelope. */
export type CSRFTokenEnvelope = Envelope<CSRFTokenData>;

// Implements DESIGN-006 AuthController frontend auth contract.
/** Authenticated session metadata; token values are carried only by HttpOnly cookies. */
export interface AuthSessionData {
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
export interface VerifyEmailData {
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
export interface PasswordResetAcceptedData {
	accepted: true;
}

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password-reset request acceptance envelope. */
export type PasswordResetRequestEnvelope = Envelope<PasswordResetAcceptedData>;

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password-reset completion payload. */
export interface PasswordResetConsumeData {
	reset: true;
}

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password-reset completion envelope. */
export type PasswordResetConsumeEnvelope = Envelope<PasswordResetConsumeData>;

// Implements DESIGN-006 OAuthHandler frontend provider contract.
/** Supported OAuth identity providers. */
export type OAuthProvider = "google" | "apple";

// Implements DESIGN-018 AuthApiClient generated endpoint contract.
/** CSRF retrieval endpoint used before protected auth mutations. */
export const AUTH_CSRF_TOKEN_ENDPOINT = "/api/v1/auth/csrf-token" as const;

// Implements DESIGN-018 AuthApiClient generated endpoint contract.
/** Email/password registration endpoint. */
export const AUTH_REGISTER_ENDPOINT = "/api/v1/auth/register" as const;

// Implements DESIGN-018 AuthApiClient generated endpoint contract.
/** Email/password login endpoint. */
export const AUTH_LOGIN_ENDPOINT = "/api/v1/auth/login" as const;

// Implements DESIGN-018 AuthApiClient generated endpoint contract.
/** Current-session logout endpoint. */
export const AUTH_LOGOUT_ENDPOINT = "/api/v1/auth/logout" as const;

// Implements DESIGN-018 AuthApiClient generated endpoint contract.
/** Refresh-cookie session recovery endpoint. */
export const AUTH_REFRESH_ENDPOINT = "/api/v1/auth/refresh" as const;

// Implements DESIGN-018 AuthApiClient generated OAuth contract.
/** Builds the provider-specific OAuth start endpoint with an optional relative return path. */
export function buildOAuthStartUrl(provider: OAuthProvider, returnTo = "/"): string {
	const base = `/api/v1/auth/oauth/${provider}/start`;
	const safeReturnTo = safeOAuthReturnPath(returnTo);
	return safeReturnTo === "/" ? base : `${base}?return_to=${encodeURIComponent(safeReturnTo)}`;
}

function safeOAuthReturnPath(value: string): string {
	try {
		if (!value || !value.startsWith("/") || value.startsWith("//") || value.includes("\\")) {
			return "/";
		}
		const parsed = new URL(value, "http://frontend.local");
		if (parsed.origin !== "http://frontend.local" || !parsed.pathname.startsWith("/")) {
			return "/";
		}
		return `${parsed.pathname}${parsed.search}`;
	} catch {
		return "/";
	}
}

// Implements DESIGN-018 AuthApiClient generated request contract.
/** Shared JSON headers used by generated auth mutation helpers. */
export type AuthJsonMutationHeaders = Record<string, string> & {
	Accept: "application/json";
	"Content-Type": "application/json";
};

// Implements DESIGN-018 AuthApiClient generated request contract.
/** Shared JSON mutation request init for auth payload submissions. */
export interface AuthJsonMutationRequestInit extends Omit<RequestInit, "body" | "credentials" | "headers" | "method"> {
	method: "POST";
	credentials: "include";
	headers: AuthJsonMutationHeaders;
	body: string;
}

// Implements DESIGN-018 AuthApiClient generated request contract.
/** Credentialed GET request init for auth/session reads. */
export interface AuthGetRequestInit extends Omit<RequestInit, "credentials" | "headers" | "method"> {
	method: "GET";
	credentials: "include";
	headers: {
		Accept: "application/json";
	};
}

// Implements DESIGN-018 AuthApiClient generated request contract.
/** Credentialed POST request init for body-less auth mutations such as logout and refresh. */
export interface AuthPostRequestInit extends Omit<RequestInit, "credentials" | "headers" | "method"> {
	method: "POST";
	credentials: "include";
	headers: {
		Accept: "application/json";
		"X-CSRF-Token"?: string;
	};
}

// Implements DESIGN-018 AuthApiClient generated CSRF retrieval contract.
/** Builds the generated CSRF token retrieval request. */
export function buildCsrfTokenRequestInit(options: { signal?: AbortSignal } = {}): AuthGetRequestInit {
	return buildCredentialedGetRequestInit(options);
}

// Implements DESIGN-018 AuthApiClient generated registration contract.
/** Builds the generated email/password registration request. */
export function buildRegisterRequestInit(
	request: RegisterRequest,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): AuthJsonMutationRequestInit {
	return buildAuthJsonMutationRequestInit(request, options);
}

// Implements DESIGN-018 AuthApiClient generated login contract.
/** Builds the generated email/password login request. */
export function buildLoginRequestInit(
	request: LoginRequest,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): AuthJsonMutationRequestInit {
	return buildAuthJsonMutationRequestInit(request, options);
}

// Implements DESIGN-018 AuthApiClient generated logout contract.
/** Builds the generated current-session logout request. */
export function buildLogoutRequestInit(options: { csrfToken?: string; signal?: AbortSignal } = {}): AuthPostRequestInit {
	return buildAuthPostRequestInit(options);
}

// Implements DESIGN-018 AuthApiClient generated session recovery contract.
/** Builds the generated refresh-cookie session recovery request. */
export function buildRefreshSessionRequestInit(options: { signal?: AbortSignal } = {}): AuthPostRequestInit {
	return buildAuthPostRequestInit(options);
}

function buildAuthJsonMutationRequestInit<TRequest extends object>(
	request: TRequest,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): AuthJsonMutationRequestInit {
	const headers: AuthJsonMutationHeaders = {
		Accept: "application/json",
		"Content-Type": "application/json"
	};
	if (options.csrfToken) {
		headers["X-CSRF-Token"] = options.csrfToken;
	}
	return {
		method: "POST",
		credentials: "include",
		headers,
		body: JSON.stringify(request),
		signal: options.signal
	};
}

function buildAuthPostRequestInit(options: { csrfToken?: string; signal?: AbortSignal } = {}): AuthPostRequestInit {
	const headers: AuthPostRequestInit["headers"] = {
		Accept: "application/json"
	};
	if (options.csrfToken) {
		headers["X-CSRF-Token"] = options.csrfToken;
	}
	return {
		method: "POST",
		credentials: "include",
		headers,
		signal: options.signal
	};
}

function buildCredentialedGetRequestInit(options: { signal?: AbortSignal } = {}): AuthGetRequestInit {
	return {
		method: "GET",
		credentials: "include",
		headers: {
			Accept: "application/json"
		},
		signal: options.signal
	};
}

// Implements DESIGN-008 PreferenceManager frontend profile contract.
/** User profile and preference response data. */
export interface ProfileData {
	userId: string;
	displayName: string;
	unitSystem: "metric" | "imperial";
	themePreference: "system" | "light" | "dark";
	requiresUnitRecalculation: boolean;
}

// Implements DESIGN-008 PreferenceManager frontend profile contract.
/** User profile response envelope. */
export type ProfileEnvelope = Envelope<ProfileData>;

// Implements DESIGN-018 AuthApiClient generated session probe contract.
/** Authenticated profile endpoint used as the frontend-safe session probe. */
export const PROFILE_ENDPOINT = "/api/v1/profile" as const;

// Implements DESIGN-018 AuthApiClient generated session probe contract.
/** Builds the generated profile/session probe request. */
export function buildProfileRequestInit(options: { signal?: AbortSignal } = {}): AuthGetRequestInit {
	return buildCredentialedGetRequestInit(options);
}

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
export interface SavedItemsData {
	items: SavedItem[];
}

// Implements DESIGN-008 SavedDataRepository frontend saved-data contract.
/** Saved item collection response envelope. */
export type SavedItemsEnvelope = Envelope<SavedItemsData>;

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** Canonical quantity units accepted by saved daily-diet entries. */
export type CanonicalQuantityUnit = "g" | "ml" | "oz" | "fl_oz";

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** One ordered meal quantity supplied to or returned from a saved diet. */
export interface MealQuantity {
	mealId: string;
	quantity: number;
	unit: CanonicalQuantityUnit;
	position: number;
}

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** One persisted saved-diet meal entry. */
export interface DailyDietMealEntry extends MealQuantity {
	id: string;
}

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** Server-derived aggregate macros and calories for one saved diet. */
export interface MacroProjection {
	protein: number;
	carbohydrates: number;
	fat: number;
	calories: number;
}

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** User-owned saved daily-diet collection; ownership is never client-supplied. */
export interface DailyDiet {
	id: string;
	name: string;
	entries: DailyDietMealEntry[];
	aggregateMacros: MacroProjection;
	createdAt: string;
	updatedAt: string;
}

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** Client-editable saved-diet fields with no authoritative aggregate totals. */
export interface DailyDietCreateRequest {
	name: string;
	entries: MealQuantity[];
}

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** Client-editable replacement fields with server-recalculated aggregates. */
export interface DailyDietReplaceRequest extends DailyDietCreateRequest {}

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** One saved-diet response envelope. */
export type DailyDietEnvelope = Envelope<DailyDiet>;

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** Saved-diet collection response payload. */
export interface DailyDietCollectionData {
	diets: DailyDiet[];
}

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** Saved-diet collection response envelope. */
export type DailyDietCollectionEnvelope = Envelope<DailyDietCollectionData>;

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet endpoint contract.
export const DAILY_DIETS_ENDPOINT = "/api/v1/daily-diets" as const;

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet endpoint contract.
/** Builds one user-scoped saved-diet URL. */
export function buildDailyDietUrl(dietId: string): string {
	return `${DAILY_DIETS_ENDPOINT}/${encodeURIComponent(dietId)}`;
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
export type DailyDietCreateHeaders = Record<string, string> & {
	Accept: "application/json";
	"Content-Type": "application/json";
	"Idempotency-Key": IdempotencyKey;
	"X-CSRF-Token"?: string;
};

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
export interface DailyDietCreateRequestInit extends Omit<RequestInit, "body" | "credentials" | "headers" | "method"> {
	method: "POST";
	credentials: "include";
	headers: DailyDietCreateHeaders;
	body: string;
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
/** Builds a CSRF- and idempotency-aware saved-diet creation request. */
export function buildDailyDietCreateRequestInit(
	request: DailyDietCreateRequest,
	idempotencyKey: IdempotencyKey,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): DailyDietCreateRequestInit {
	const headers: DailyDietCreateHeaders = {
		Accept: "application/json",
		"Content-Type": "application/json",
		"Idempotency-Key": idempotencyKey
	};
	if (options.csrfToken) headers["X-CSRF-Token"] = options.csrfToken;
	return { method: "POST", credentials: "include", headers, body: JSON.stringify(request), signal: options.signal };
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
/** Credentialed read request for a saved-diet collection or item. */
export interface DailyDietGetRequestInit extends Omit<RequestInit, "credentials" | "headers" | "method"> {
	method: "GET";
	credentials: "include";
	headers: { Accept: "application/json" };
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
/** Builds the generated saved-diet collection read request. */
export function buildDailyDietListRequestInit(options: { signal?: AbortSignal } = {}): DailyDietGetRequestInit {
	return { method: "GET", credentials: "include", headers: { Accept: "application/json" }, signal: options.signal };
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
/** Builds the generated saved-diet item read request. */
export function buildDailyDietGetRequestInit(options: { signal?: AbortSignal } = {}): DailyDietGetRequestInit {
	return buildDailyDietListRequestInit(options);
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
export type DailyDietMutationHeaders = Record<string, string> & {
	Accept: "application/json";
	"Content-Type": "application/json";
	"X-CSRF-Token"?: string;
};

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
export interface DailyDietReplaceRequestInit extends Omit<RequestInit, "body" | "credentials" | "headers" | "method"> {
	method: "PUT";
	credentials: "include";
	headers: DailyDietMutationHeaders;
	body: string;
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
/** Builds the generated CSRF-protected saved-diet replacement request. */
export function buildDailyDietReplaceRequestInit(
	request: DailyDietReplaceRequest,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): DailyDietReplaceRequestInit {
	const headers: DailyDietMutationHeaders = {
		Accept: "application/json",
		"Content-Type": "application/json"
	};
	if (options.csrfToken) headers["X-CSRF-Token"] = options.csrfToken;
	return { method: "PUT", credentials: "include", headers, body: JSON.stringify(request), signal: options.signal };
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
export interface DailyDietDeleteRequestInit extends Omit<RequestInit, "credentials" | "headers" | "method"> {
	method: "DELETE";
	credentials: "include";
	headers: { Accept: "application/json"; "X-CSRF-Token"?: string };
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
/** Builds the generated CSRF-protected saved-diet deletion request. */
export function buildDailyDietDeleteRequestInit(options: { csrfToken?: string; signal?: AbortSignal } = {}): DailyDietDeleteRequestInit {
	const headers: DailyDietDeleteRequestInit["headers"] = { Accept: "application/json" };
	if (options.csrfToken) headers["X-CSRF-Token"] = options.csrfToken;
	return { method: "DELETE", credentials: "include", headers, signal: options.signal };
}

// Implements DESIGN-004 JobStatusTracker frontend optimization contract.
/** Asynchronous optimization submission for one server-owned saved diet. */
export interface DietOptimizationRequest {
	dailyDietId: string;
	tolerancePercent: number;
	excludedMealIds: string[];
}

// Implements DESIGN-004 JobStatusTracker frontend optimization contract.
export type OptimizationStatus = "queued" | "processing" | "completed" | "failed" | "cancelled";

// Implements DESIGN-004 JobStatusTracker frontend safe failure contract.
export type OptimizationFailureCode =
	| "failed_validation"
	| "solver_timeout"
	| "solver_infeasible"
	| "queue_unavailable"
	| "worker_crash"
	| "result_expired";

// Implements DESIGN-004 JobStatusTracker frontend completed-alternative contract.
export interface OptimizationAlternative {
	meals: MealQuantity[];
	macros: MacroProjection;
	similarityScore: number;
}

// Implements DESIGN-004 JobStatusTracker frontend completed-alternative contract.
/** Type-level mirror of the OpenAPI maximum-three-alternatives constraint. */
export type OptimizationAlternativeList =
	| []
	| [OptimizationAlternative]
	| [OptimizationAlternative, OptimizationAlternative]
	| [OptimizationAlternative, OptimizationAlternative, OptimizationAlternative];

// Implements DESIGN-004 JobStatusTracker frontend completed-alternative contract.
/** A completed job must contain at least one and at most three alternatives. */
export type CompletedOptimizationAlternativeList =
	| [OptimizationAlternative]
	| [OptimizationAlternative, OptimizationAlternative]
	| [OptimizationAlternative, OptimizationAlternative, OptimizationAlternative];

// Implements DESIGN-004 JobStatusTracker frontend safe failure contract.
export interface OptimizationFailure {
	code: OptimizationFailureCode;
	message: string;
}

// Implements DESIGN-004 JobStatusTracker frontend 202 acknowledgement contract.
export interface OptimizationJobAcknowledgementData {
	jobId: string;
	status: "queued";
	pollUrl: string;
}

// Implements DESIGN-004 JobStatusTracker frontend 202 acknowledgement contract.
export type OptimizationJobAcknowledgementEnvelope = Envelope<OptimizationJobAcknowledgementData> & {
	status: "accepted";
};

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
/** Fields shared by every user-scoped optimization polling state. */
export interface OptimizationJobCommon {
	jobId: string;
	dailyDietId: string;
	pollUrl: string;
	createdAt: string;
}

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
/** Queued jobs contain acknowledgement metadata only. */
export interface OptimizationJobQueued extends OptimizationJobCommon {
	status: "queued";
	startedAt?: never;
	finishedAt?: never;
	alternatives?: never;
	failure?: never;
}

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
/** Processing jobs expose start metadata but no result or failure payload. */
export interface OptimizationJobProcessing extends OptimizationJobCommon {
	status: "processing";
	startedAt: string;
	finishedAt?: never;
	alternatives?: never;
	failure?: never;
}

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
/** Completed jobs require one to three alternatives and cannot carry a failure. */
export interface OptimizationJobCompleted extends OptimizationJobCommon {
	status: "completed";
	startedAt: string;
	finishedAt: string;
	alternatives: CompletedOptimizationAlternativeList;
	failure?: never;
}

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
/** Failed jobs require a safe failure and may retain validated partial alternatives. */
export interface OptimizationJobFailed extends OptimizationJobCommon {
	status: "failed";
	startedAt?: string | null;
	finishedAt?: string | null;
	alternatives?: OptimizationAlternativeList;
	failure: OptimizationFailure;
}

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
/** Cancelled jobs are terminal without alternatives or failure details. */
export interface OptimizationJobCancelled extends OptimizationJobCommon {
	status: "cancelled";
	finishedAt: string;
	startedAt?: never;
	alternatives?: never;
	failure?: never;
}

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
/** Discriminated polling union; status selects the only valid payload shape. */
export type OptimizationJobData =
	| OptimizationJobQueued
	| OptimizationJobProcessing
	| OptimizationJobCompleted
	| OptimizationJobFailed
	| OptimizationJobCancelled;

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
export type OptimizationJobStatusEnvelope = Envelope<OptimizationJobData>;

// Implements DESIGN-004 JobStatusTracker frontend endpoint contract.
export const OPTIMIZATION_JOBS_ENDPOINT = "/api/v1/optimization/jobs" as const;

// Implements DESIGN-004 JobStatusTracker frontend endpoint contract.
/** Builds one user-scoped optimization polling URL. */
export function buildOptimizationJobUrl(jobId: string): string {
	return `${OPTIMIZATION_JOBS_ENDPOINT}/${encodeURIComponent(jobId)}`;
}

// Implements DESIGN-004 JobStatusTracker frontend submission request contract.
export type OptimizationSubmissionHeaders = Record<string, string> & {
	Accept: "application/json";
	"Content-Type": "application/json";
	"Idempotency-Key": IdempotencyKey;
	"X-CSRF-Token"?: string;
};

// Implements DESIGN-004 JobStatusTracker frontend submission request contract.
export interface OptimizationSubmissionRequestInit extends Omit<RequestInit, "body" | "credentials" | "headers" | "method"> {
	method: "POST";
	credentials: "include";
	headers: OptimizationSubmissionHeaders;
	body: string;
}

// Implements DESIGN-004 JobStatusTracker frontend submission request contract.
/** Builds a CSRF- and idempotency-aware asynchronous optimization request. */
export function buildOptimizationSubmissionRequestInit(
	request: DietOptimizationRequest,
	idempotencyKey: IdempotencyKey,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): OptimizationSubmissionRequestInit {
	const headers: OptimizationSubmissionHeaders = {
		Accept: "application/json",
		"Content-Type": "application/json",
		"Idempotency-Key": idempotencyKey
	};
	if (options.csrfToken) headers["X-CSRF-Token"] = options.csrfToken;
	return { method: "POST", credentials: "include", headers, body: JSON.stringify(request), signal: options.signal };
}

// Implements DESIGN-004 JobStatusTracker frontend polling request contract.
export interface OptimizationJobRequestInit extends Omit<RequestInit, "credentials" | "headers" | "method"> {
	method: "GET";
	credentials: "include";
	headers: { Accept: "application/json" };
}

// Implements DESIGN-004 JobStatusTracker frontend polling request contract.
/** Builds a credentialed optimization job polling request. */
export function buildOptimizationJobRequestInit(options: { signal?: AbortSignal } = {}): OptimizationJobRequestInit {
	return { method: "GET", credentials: "include", headers: { Accept: "application/json" }, signal: options.signal };
}

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
export interface SearchHistoryData {
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
export interface DeletionRequestData {
	requestId: string;
	status: "pending" | "processing" | "completed" | "failed";
}

// Implements DESIGN-008 AccountDeleter frontend deletion contract.
/** Account deletion response envelope. */
export type DeletionRequestEnvelope = Envelope<DeletionRequestData>;

// Implements DESIGN-015 DisclaimerRenderer frontend disclaimer contract.
/** Stable Markdown disclaimer content for login and account surfaces. */
export interface DisclaimerData {
	location: "login" | "account";
	version: string;
	markdown: string;
	fallback: boolean;
	alert?: string;
}

// Implements DESIGN-015 DisclaimerRenderer frontend disclaimer contract.
/** Disclaimer response envelope. */
export type DisclaimerEnvelope = Envelope<DisclaimerData>;

// Implements DESIGN-018 AuthApiClient generated disclaimer contract.
/** Disclaimer endpoint used by login and account auth surfaces. */
export const DISCLAIMER_ENDPOINT = "/api/v1/disclaimers" as const;

// Implements DESIGN-018 AuthApiClient generated disclaimer contract.
/** Supported disclaimer locations documented by the OpenAPI contract. */
export type DisclaimerLocation = DisclaimerData["location"];

// Implements DESIGN-018 AuthApiClient generated disclaimer contract.
/** Builds the generated disclaimer URL with an explicit location query. */
export function buildDisclaimerUrl(location: DisclaimerLocation = "login"): `/api/v1/disclaimers?location=${DisclaimerLocation}` {
	return `${DISCLAIMER_ENDPOINT}?location=${location}`;
}

// Implements DESIGN-018 AuthApiClient generated disclaimer contract.
/** Builds the generated disclaimer retrieval request. */
export function buildDisclaimerRequestInit(options: { signal?: AbortSignal } = {}): AuthGetRequestInit {
	return buildCredentialedGetRequestInit(options);
}

// Implements DESIGN-007 SubscriptionController frontend billing endpoint contract.
/** Entitlement status endpoint path exported for generated-type-only frontend gating. */
export const BILLING_ENTITLEMENT_ENDPOINT = "/api/v1/billing/entitlement" as const;

// Implements DESIGN-007 SubscriptionController frontend billing endpoint contract.
/** Checkout creation endpoint path exported with its generated request helpers. */
export const BILLING_CHECKOUT_ENDPOINT = "/api/v1/billing/checkout" as const;

// Implements DESIGN-007 SubscriptionController frontend billing endpoint contract.
/** Billing portal creation endpoint path exported with its generated request helpers. */
export const BILLING_PORTAL_ENDPOINT = "/api/v1/billing/portal" as const;

// Implements DESIGN-007 SubscriptionController frontend checkout idempotency contract.
/** Stable client-generated idempotency key sent with checkout creation retries. */
export type IdempotencyKey = string;

// Implements DESIGN-017 ErrorMessageMapper frontend billing error contract.
/** Billing statuses documented by the OpenAPI billing and entitlement contract. */
export type BillingErrorStatus = 400 | 401 | 402 | 409 | 422 | 503;

// Implements DESIGN-017 ErrorMessageMapper frontend billing error contract.
/** User-safe billing error codes consumed by frontend billing and entitlement gates. */
export type BillingErrorCode =
	| "billing_payment_required"
	| "billing_recovery_required"
	| "billing_portal_unavailable"
	| "checkout_idempotency_conflict"
	| "checkout_invalid_request"
	| "checkout_validation_failed"
	| "stripe_unavailable"
	| "entitlement_unavailable";

// Implements DESIGN-017 ErrorMessageMapper frontend billing error contract.
/** Classified billing error envelope returned by checkout and entitlement endpoints. */
export interface BillingErrorEnvelope extends Envelope {
	status: "error";
	error: AppError & {
		category: "auth" | "entitlement" | "validation" | "dependency";
		code: BillingErrorCode | (string & {});
	};
}

// Implements DESIGN-007 SubscriptionController frontend checkout contract.
/** Public checkout billing period accepted by hosted checkout creation. */
export type CheckoutPlan = "monthly" | "annual";

// Implements DESIGN-007 SubscriptionController frontend checkout contract.
/** Hosted checkout creation request. Raw payment-card data is not accepted. */
export interface CheckoutCreateRequest {
	plan: CheckoutPlan;
	successUrl: string;
	cancelUrl: string;
}

// Implements DESIGN-007 SubscriptionController frontend checkout idempotency contract.
/** Headers required to create or replay checkout sessions without duplicate side effects. */
export type CheckoutCreateHeaders = Record<string, string> & {
	Accept: "application/json";
	"Content-Type": "application/json";
	"Idempotency-Key": IdempotencyKey;
};

// Implements DESIGN-007 SubscriptionController frontend checkout idempotency contract.
/** RequestInit shape for the generated checkout creation helper. */
export interface CheckoutCreateRequestInit extends Omit<RequestInit, "body" | "credentials" | "headers" | "method"> {
	method: "POST";
	credentials: "include";
	headers: CheckoutCreateHeaders;
	body: string;
}

// Implements DESIGN-007 SubscriptionController frontend checkout idempotency contract.
/** Builds a checkout creation request with the required idempotency header. */
export function buildCheckoutCreateRequestInit(
	request: CheckoutCreateRequest,
	idempotencyKey: IdempotencyKey,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): CheckoutCreateRequestInit {
	const headers: CheckoutCreateHeaders = {
		Accept: "application/json",
		"Content-Type": "application/json",
		"Idempotency-Key": idempotencyKey
	};
	if (options.csrfToken) {
		headers["X-CSRF-Token"] = options.csrfToken;
	}
	return {
		method: "POST",
		credentials: "include",
		headers,
		body: JSON.stringify(request),
		signal: options.signal
	};
}

// Implements DESIGN-007 SubscriptionController frontend checkout contract.
/** Sanitized hosted checkout session response. */
export interface CheckoutSessionData {
	checkoutSessionId: string;
	checkoutUrl: string;
	plan: CheckoutPlan;
	priceId: string;
	amountCents: number;
}

// Implements DESIGN-007 SubscriptionController frontend checkout contract.
/** Hosted checkout session response envelope. */
export type CheckoutSessionEnvelope = Envelope<CheckoutSessionData>;

// Implements DESIGN-007 SubscriptionController frontend billing portal contract.
/** Hosted billing portal creation request. */
export interface BillingPortalCreateRequest {
	returnUrl: string;
}

// Implements DESIGN-007 SubscriptionController frontend billing portal contract.
/** Hosted billing portal creation request headers. */
export type BillingPortalCreateHeaders = Record<string, string> & {
	Accept: "application/json";
	"Content-Type": "application/json";
};

// Implements DESIGN-007 SubscriptionController frontend billing portal contract.
/** RequestInit shape for the generated billing portal creation helper. */
export interface BillingPortalCreateRequestInit extends Omit<RequestInit, "body" | "credentials" | "headers" | "method"> {
	method: "POST";
	credentials: "include";
	headers: BillingPortalCreateHeaders;
	body: string;
}

// Implements DESIGN-007 SubscriptionController frontend billing portal contract.
/** Builds a billing portal creation request. */
export function buildBillingPortalCreateRequestInit(
	request: BillingPortalCreateRequest,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): BillingPortalCreateRequestInit {
	const headers: BillingPortalCreateHeaders = {
		Accept: "application/json",
		"Content-Type": "application/json"
	};
	if (options.csrfToken) {
		headers["X-CSRF-Token"] = options.csrfToken;
	}
	return {
		method: "POST",
		credentials: "include",
		headers,
		body: JSON.stringify(request),
		signal: options.signal
	};
}

// Implements DESIGN-007 SubscriptionController frontend billing portal contract.
/** Sanitized hosted billing portal session response. */
export interface BillingPortalSessionData {
	portalUrl: string;
}

// Implements DESIGN-007 SubscriptionController frontend billing portal contract.
/** Hosted billing portal session response envelope. */
export type BillingPortalSessionEnvelope = Envelope<BillingPortalSessionData>;

// Implements DESIGN-007 StripeWebhookHandler frontend-visible webhook contract.
/** Verified Stripe webhook processing result. */
export interface StripeWebhookData {
	eventId: string;
	eventType: string;
	duplicate: boolean;
}

// Implements DESIGN-007 StripeWebhookHandler frontend-visible webhook contract.
/** Stripe webhook processing response envelope. */
export type StripeWebhookEnvelope = Envelope<StripeWebhookData>;

// Implements DESIGN-007 SubscriptionController frontend entitlement contract.
/** Subscription tier exposed by entitlement status reads. */
export type SubscriptionTier = "free" | "trial" | "paid";

// Implements DESIGN-007 SubscriptionController frontend entitlement contract.
/** Persisted entitlement state exposed without provider identifiers. */
export type EntitlementState = "active" | "expired" | "past_due" | "cancelled";

// Implements DESIGN-007 SubscriptionController frontend billing-state contract.
/** Frontend-safe billing recovery state derived from provider status. */
export type BillingRecoveryState = "none" | "action_required" | "cancelled" | "expired";

// Implements DESIGN-007 SubscriptionController frontend entitlement contract.
/** Sanitized entitlement and billing status payload for the current user. */
export interface EntitlementStatusData {
	userId: string;
	tier: SubscriptionTier;
	status: EntitlementState;
	allowedModes: SearchMode[];
	searchLimitPer24h: number;
	usageUsed: number;
	usageRemaining: number | null;
	usageWindowStartedAt: string | null;
	trialExpiresAt: string | null;
	billingRecoveryState: BillingRecoveryState;
}

// Implements DESIGN-007 SubscriptionController frontend entitlement contract.
/** Entitlement status response envelope. */
export type EntitlementStatusEnvelope = Envelope<EntitlementStatusData>;

// Implements DESIGN-007 SubscriptionController frontend entitlement contract.
/** RequestInit shape for generated entitlement status reads. */
export interface EntitlementStatusRequestInit extends Omit<RequestInit, "credentials" | "headers" | "method"> {
	method: "GET";
	credentials: "include";
	headers: {
		Accept: "application/json";
	};
}

// Implements DESIGN-007 SubscriptionController frontend entitlement contract.
/** Builds an entitlement status request that consumes only generated entitlement types. */
export function buildEntitlementStatusRequestInit(
	options: { signal?: AbortSignal } = {}
): EntitlementStatusRequestInit {
	return {
		method: "GET",
		credentials: "include",
		headers: {
			Accept: "application/json"
		},
		signal: options.signal
	};
}

// Implements DESIGN-002 SearchController frontend search-mode contract.
/** Supported search workflows exposed by the search API. */
export type SearchMode = "catalog" | "substitution" | "daily_diet" | "daily_diet_alternative";

// Implements DESIGN-002 SearchController frontend search-filter contract.
/** Supported filter classes accepted by the search API. */
export type SearchFilterKind =
	| "food_category"
	| "culinary_role"
	| "physical_state"
	| "allergen"
	| "dietary_preset";

// Implements DESIGN-002 SearchController frontend search-filter contract.
/** One include or exclude filter applied to a search request. */
export interface SearchFilter {
	filterId: string;
	kind: SearchFilterKind;
	include: boolean;
}

// Implements DESIGN-002 SearchController frontend substitution contract.
/** Canonical units accepted by substitution search inputs. */
export type SubstitutionUnit = "g" | "ml" | "oz" | "fl_oz";

// Implements DESIGN-002 SearchController frontend substitution contract.
/** Quantity-bearing food input for substitution searches. */
export interface SubstitutionInput {
	foodObjectId: string;
	quantity: number;
	unit: SubstitutionUnit;
}

// Implements DESIGN-002 SearchController frontend search request contract.
/** Request payload for catalog, substitution, and daily-diet alternative search. */
export interface SearchRequest {
	query: string;
	mode: SearchMode;
	filters?: SearchFilter[];
	page: number;
	substitutionInputs?: SubstitutionInput[];
	dailyDietId?: string;
}

// Implements DESIGN-002 SearchController frontend classification summary contract.
/** Public classification identity summary for a search result item. */
export interface ClassificationSummary {
	id: string;
	name: string;
	kind: "food_category" | "culinary_role";
}

// Implements DESIGN-002 SearchController frontend macro profile contract.
/** Protein, carbohydrate, and fat macro values on a 100g or 100ml basis. */
export interface MacroProfile {
	protein: number;
	carbohydrates: number;
	fat: number;
}

// Implements DESIGN-002 SearchController frontend substitution source summary contract.
/** Macro and amount totals for the user's selected substitution input list. */
export interface SourceSummary {
	macros: MacroProfile;
	calories: number;
	totalGrams: number;
	totalMilliliters: number;
}

// Implements DESIGN-002 SearchController frontend food-object result contract.
/** Food object returned by search and autocomplete-related result flows. */
export interface FoodObject {
	id: string;
	name: string;
	physicalState: "solid" | "liquid";
	imageUrl?: string | null;
	classifications: ClassificationSummary[];
	primaryFoodCategory: ClassificationSummary | null;
	macros: MacroProfile;
	macroBasis: "100g" | "100ml";
	calories: number;
}

// Implements DESIGN-002 SearchController frontend food-object detail contract.
/** Successful food-object detail response envelope. */
export type FoodObjectEnvelope = Envelope<FoodObject>;

// Implements DESIGN-002 SearchController frontend similarity metadata contract.
/** User-facing nutritional similarity tier. */
export type SimilarityTier = "excellent" | "good" | "fair" | "poor";

// Implements DESIGN-002 SearchController frontend similarity metadata contract.
/** Similarity display metadata for a ranked search result. */
export interface SimilarityMetadata {
	itemId: string;
	score: number;
	tier: SimilarityTier;
	imageUrl: string;
	matchingQuantity: number;
}

// Implements DESIGN-011 SearchCache frontend cache metadata contract.
/** Cache status metadata returned with search-domain responses. */
export interface CacheMetadata {
	status: "hit" | "miss";
	namespace: string;
	schemaVersion: string;
	ttlSeconds: number;
}

// Implements DESIGN-002 SearchController frontend search rejection contract.
/** Structured, user-facing search rejection detail. */
export interface SearchRejection {
	code: string;
	message: string;
	field?: string;
}

// Implements DESIGN-002 SearchController frontend search response contract.
/** Search result payload with ranking, warnings, and optional cache metadata. */
export interface SearchResponse {
	items: FoodObject[];
	totalCount: number;
	page: number;
	similarityScores: number[];
	similarityMetadata: SimilarityMetadata[];
	sourceSummary?: SourceSummary;
	warnings: string[];
	cache?: CacheMetadata;
}

// Implements DESIGN-002 SearchController frontend search response contract.
/** Successful search response envelope. */
export type SearchResponseEnvelope = Envelope<SearchResponse>;

// Implements DESIGN-017 ErrorMessageMapper frontend search error contract.
/** Search rejection response envelope with safe error text. */
export interface SearchRejectionEnvelope extends Envelope<{ rejection: SearchRejection }> {
	status: "error";
	data: { rejection: SearchRejection };
	error: AppError;
}

// Implements DESIGN-002 SearchController frontend autocomplete contract.
/** Ranked autocomplete suggestion. */
export interface RankedAutocomplete {
	itemId: string;
	label: string;
	exactMatch: boolean;
	levenshteinDistance: number;
	length: number;
	rank: number;
}

// Implements DESIGN-002 SearchController frontend autocomplete contract.
/** Autocomplete payload with ranked suggestions and optional cache metadata. */
export interface AutocompleteResponse {
	items: RankedAutocomplete[];
	cache?: CacheMetadata;
}

// Implements DESIGN-002 SearchController frontend autocomplete contract.
/** Successful autocomplete response envelope. */
export type AutocompleteEnvelope = Envelope<AutocompleteResponse>;
