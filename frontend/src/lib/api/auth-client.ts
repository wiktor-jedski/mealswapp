import {
	AUTH_CSRF_TOKEN_ENDPOINT,
	AUTH_LOGIN_ENDPOINT,
	AUTH_LOGOUT_ENDPOINT,
	AUTH_REFRESH_ENDPOINT,
	AUTH_REGISTER_ENDPOINT,
	BILLING_ENTITLEMENT_ENDPOINT,
	PROFILE_ENDPOINT,
	buildCsrfTokenRequestInit,
	buildEntitlementStatusRequestInit,
	buildLoginRequestInit,
	buildLogoutRequestInit,
	buildOAuthStartUrl,
	buildProfileRequestInit,
	buildRefreshSessionRequestInit,
	buildRegisterRequestInit,
	type AppError,
	type AuthSessionData,
	type AuthSessionEnvelope,
	type CSRFTokenEnvelope,
	type EntitlementStatusData,
	type EntitlementStatusEnvelope,
	type Envelope,
	type LoginRequest,
	type OAuthProvider,
	type ProfileData,
	type ProfileEnvelope,
	type RegisterRequest
} from "./generated";

// Implements DESIGN-018 AuthApiClient generated-contract frontend auth wrapper.
// Implements DESIGN-017 ErrorMessageMapper safe auth error mapping.

const maxRetryAfterDisplaySeconds = 60 * 60;

/** CSRF context passed only across one auth mutation request boundary. */
export interface AuthRequestContext {
	csrfToken: string;
	requestId?: string;
}

/** Error thrown by auth API calls after mapping server envelopes to user-safe AppError values. */
export class AuthClientError extends Error {
	readonly appError: AppError;
	readonly status: number;
	readonly retryAfterSeconds?: number;

	constructor(appError: AppError, status: number, retryAfterSeconds?: number) {
		super(appError.message);
		this.name = "AuthClientError";
		this.appError = appError;
		this.status = status;
		this.retryAfterSeconds = retryAfterSeconds;
	}
}

/** Fetches a fresh CSRF token with cookies included before a protected auth mutation. */
export async function fetchCsrfToken(signal?: AbortSignal): Promise<AuthRequestContext> {
	const response = await fetch(AUTH_CSRF_TOKEN_ENDPOINT, buildCsrfTokenRequestInit({ signal }));
	const envelope = await decodeAuthEnvelope<CSRFTokenEnvelope>(
		response,
		"Authentication security check is temporarily unavailable. Please try again."
	);
	return { csrfToken: envelope.data.csrfToken, requestId: envelope.requestId };
}

/** Registers with generated DTOs, clears the caller-owned password field, and returns safe session data. */
export async function registerWithEmail(
	request: RegisterRequest,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): Promise<AuthSessionData> {
	try {
		const response = await fetch(
			AUTH_REGISTER_ENDPOINT,
			buildRegisterRequestInit(request, { csrfToken: options.csrfToken, signal: options.signal })
		);
		return sanitizeSessionData(
			await decodeAuthData<AuthSessionEnvelope, AuthSessionData>(
				response,
				"Registration is temporarily unavailable. Please try again."
			)
		);
	} finally {
		request.password = "";
	}
}

/** Logs in with generated DTOs, clears the caller-owned password field, and returns safe session data. */
export async function loginWithEmail(
	request: LoginRequest,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): Promise<AuthSessionData> {
	try {
		const response = await fetch(
			AUTH_LOGIN_ENDPOINT,
			buildLoginRequestInit(request, { csrfToken: options.csrfToken, signal: options.signal })
		);
		return sanitizeSessionData(
			await decodeAuthData<AuthSessionEnvelope, AuthSessionData>(
				response,
				"Login is temporarily unavailable. Please try again."
			)
		);
	} finally {
		request.password = "";
	}
}

/** Logs out the current HttpOnly-cookie session using a fresh CSRF token. */
export async function logoutCurrentSession(
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): Promise<void> {
	const response = await fetch(
		AUTH_LOGOUT_ENDPOINT,
		buildLogoutRequestInit({ csrfToken: options.csrfToken, signal: options.signal })
	);
	await decodeEmptyAuthResponse(response, "Logout is temporarily unavailable. Please try again.");
}

/** Refreshes the cookie-backed session and returns only frontend-safe session data. */
export async function refreshAuthSession(signal?: AbortSignal): Promise<AuthSessionData> {
	const response = await fetch(AUTH_REFRESH_ENDPOINT, buildRefreshSessionRequestInit({ signal }));
	return sanitizeSessionData(
		await decodeAuthData<AuthSessionEnvelope, AuthSessionData>(
			response,
			"Your session could not be refreshed. Please sign in again."
		)
	);
}

/** Probes the frontend-safe profile endpoint with cookies included. */
export async function probeProfileSession(signal?: AbortSignal): Promise<ProfileData> {
	const response = await fetch(PROFILE_ENDPOINT, buildProfileRequestInit({ signal }));
	return sanitizeProfileData(
		await decodeAuthData<ProfileEnvelope, ProfileData>(
			response,
			"Your profile is temporarily unavailable. Please try again."
		)
	);
}

/** Constructs the generated OAuth provider start URL without embedding provider secrets. */
export function getOAuthStartUrl(provider: OAuthProvider, returnTo = "/"): string {
	return buildOAuthStartUrl(provider, returnTo);
}

/** Refreshes entitlement data after auth changes using the generated billing contract. */
export async function refreshEntitlementAfterAuth(signal?: AbortSignal): Promise<EntitlementStatusData> {
	const response = await fetch(BILLING_ENTITLEMENT_ENDPOINT, buildEntitlementStatusRequestInit({ signal }));
	return decodeAuthData<EntitlementStatusEnvelope, EntitlementStatusData>(
		response,
		"Entitlement status is temporarily unavailable. Please try again."
	);
}

/** Runs the session and entitlement refresh sequence used after successful OAuth callback return. */
export async function refreshAuthStateAfterOAuthReturn(signal?: AbortSignal): Promise<{
	session: AuthSessionData;
	entitlement: EntitlementStatusData;
}> {
	const session = await refreshAuthSession(signal);
	const entitlement = await refreshEntitlementAfterAuth(signal);
	return { session, entitlement };
}

async function decodeEmptyAuthResponse(response: Response, fallbackMessage: string): Promise<void> {
	const envelope = await readJsonEnvelope(response);
	if (!response.ok) {
		throw mapAuthError(envelope, response, fallbackMessage);
	}
}

async function decodeAuthData<TEnvelope extends Envelope<TData>, TData extends object>(
	response: Response,
	fallbackMessage: string
): Promise<TData> {
	const envelope = await decodeAuthEnvelope<TEnvelope>(response, fallbackMessage);
	return envelope.data as TData;
}

async function decodeAuthEnvelope<TEnvelope extends Envelope<unknown>>(
	response: Response,
	fallbackMessage: string
): Promise<TEnvelope & { data: NonNullable<TEnvelope["data"]> }> {
	const envelope = await readJsonEnvelope(response);
	if (!response.ok) {
		throw mapAuthError(envelope, response, fallbackMessage);
	}
	if (!envelope?.data) {
		throw malformedAuthEnvelopeError(response.status, envelope?.requestId);
	}
	return envelope as TEnvelope & { data: NonNullable<TEnvelope["data"]> };
}

async function readJsonEnvelope(response: Response): Promise<Envelope<unknown> | null> {
	try {
		const body = (await response.json()) as unknown;
		if (typeof body !== "object" || body === null) {
			return null;
		}
		return body as Envelope<unknown>;
	} catch {
		return null;
	}
}

function mapAuthError(envelope: Envelope<unknown> | null, response: Response, fallbackMessage: string): AuthClientError {
	const status = response.status;
	const source = envelope?.error ?? null;
	const appError: AppError = {
		category: source?.category ?? categoryForAuthStatus(status),
		code: source?.code ?? codeForAuthStatus(status),
		message: safeMessage(source?.message) ? source.message : fallbackMessage,
		retryable: source?.retryable ?? retryableForAuthStatus(status)
	};
	if (source?.requestId) {
		appError.requestId = source.requestId;
	} else if (envelope?.requestId) {
		appError.requestId = envelope.requestId;
	}
	return new AuthClientError(appError, status, retryAfterSeconds(response));
}

function malformedAuthEnvelopeError(status: number, requestId: string | undefined): AuthClientError {
	const appError: AppError = {
		category: "server",
		code: "malformed_envelope",
		message: "Authentication response is temporarily unavailable. Please try again.",
		retryable: true
	};
	if (requestId) {
		appError.requestId = requestId;
	}
	return new AuthClientError(appError, status);
}

function categoryForAuthStatus(status: number): AppError["category"] {
	switch (status) {
		case 400:
		case 409:
			return "validation";
		case 401:
			return "auth";
		case 403:
			return "security";
		case 429:
			return "timeout";
		case 503:
			return "dependency";
		default:
			return "unknown";
	}
}

function codeForAuthStatus(status: number): string {
	switch (status) {
		case 400:
			return "auth_validation_failed";
		case 401:
			return "invalid_credentials";
		case 403:
			return "csrf_invalid";
		case 409:
			return "duplicate_or_stale_auth_state";
		case 429:
			return "auth_rate_limited";
		case 503:
			return "auth_unavailable";
		default:
			return "auth_request_failed";
	}
}

function retryableForAuthStatus(status: number): boolean {
	return status === 429 || status === 503;
}

function retryAfterSeconds(response: Response): number | undefined {
	const value = response.headers.get("Retry-After");
	if (!value) {
		return undefined;
	}
	const trimmedValue = value.trim();
	if (/^\d+$/.test(trimmedValue)) {
		return boundedRetrySeconds(Number(trimmedValue));
	}

	const retryAt = Date.parse(trimmedValue);
	if (!Number.isFinite(retryAt)) {
		return undefined;
	}
	return boundedRetrySeconds(Math.ceil((retryAt - Date.now()) / 1000));
}

function boundedRetrySeconds(seconds: number): number | undefined {
	if (!Number.isFinite(seconds) || seconds < 0) {
		return undefined;
	}
	return Math.min(seconds, maxRetryAfterDisplaySeconds);
}

function safeMessage(message: string | undefined): message is string {
	return Boolean(message && message.length <= 240 && !message.includes("\n") && !message.includes(" at "));
}

function sanitizeSessionData(data: AuthSessionData): AuthSessionData {
	return {
		userId: data.userId,
		role: data.role,
		hasVerifiedLoginMethod: data.hasVerifiedLoginMethod,
		accessExpiresAt: data.accessExpiresAt,
		refreshExpiresAt: data.refreshExpiresAt
	};
}

function sanitizeProfileData(data: ProfileData): ProfileData {
	return {
		userId: data.userId,
		displayName: data.displayName,
		unitSystem: data.unitSystem,
		themePreference: data.themePreference,
		requiresUnitRecalculation: data.requiresUnitRecalculation
	};
}
