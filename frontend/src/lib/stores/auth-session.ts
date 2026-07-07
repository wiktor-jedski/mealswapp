import { get, writable } from "svelte/store";

import * as authApi from "../api/auth-client";
import type {
	AppError,
	AuthSessionData,
	EntitlementStatusData,
	LoginRequest,
	RegisterRequest
} from "../api/generated";
import { setEntitlementError, setEntitlementStatus } from "./entitlement";

// Implements DESIGN-018 AuthSessionStore frontend-safe session projection and lifecycle orchestration.

/** Frontend-visible auth state derived from server responses and safe errors. */
export type AuthStatus =
	| "unknown"
	| "anonymous"
	| "authenticating"
	| "authenticated"
	| "expired"
	| "locked"
	| "error";

/** Cookie-backed session projection; never includes bearer tokens, raw CSRF secrets, or passwords. */
export interface AuthSessionProjection {
	status: AuthStatus;
	userId?: string;
	email?: string;
	displayName?: string;
	hasVerifiedLoginMethod?: boolean;
	role?: "user" | "admin";
	lastCheckedAt?: string;
	error?: AppError;
}

interface AuthSessionDependencies {
	fetchCsrfToken: typeof authApi.fetchCsrfToken;
	registerWithEmail: typeof authApi.registerWithEmail;
	loginWithEmail: typeof authApi.loginWithEmail;
	logoutCurrentSession: typeof authApi.logoutCurrentSession;
	refreshAuthSession: typeof authApi.refreshAuthSession;
	probeProfileSession: typeof authApi.probeProfileSession;
	refreshEntitlementAfterAuth: typeof authApi.refreshEntitlementAfterAuth;
	setEntitlementStatus: (status: EntitlementStatusData) => void;
	setEntitlementError: (error: AppError) => void;
	now: () => string;
}

const AUTH_SESSION_STORAGE_KEY = "mealswapp.auth-session";
const defaultDependencies: AuthSessionDependencies = {
	fetchCsrfToken: authApi.fetchCsrfToken,
	registerWithEmail: authApi.registerWithEmail,
	loginWithEmail: authApi.loginWithEmail,
	logoutCurrentSession: authApi.logoutCurrentSession,
	refreshAuthSession: authApi.refreshAuthSession,
	probeProfileSession: authApi.probeProfileSession,
	refreshEntitlementAfterAuth: authApi.refreshEntitlementAfterAuth,
	setEntitlementStatus,
	setEntitlementError,
	now: () => new Date().toISOString()
};
let dependencies = defaultDependencies;

/** Creates the default startup session state before the server probe completes. */
export function createInitialAuthSession(): AuthSessionProjection {
	return { status: "unknown" };
}

/** Svelte store holding only frontend-safe session projection fields. */
export const authSessionStore = writable<AuthSessionProjection>(createInitialAuthSession());

/**
 * Starts unknown so protected actions wait for a current server probe.
 *
 * @remarks Implements DESIGN-018 AuthSessionStore startup initialization before server verification.
 */
export function initAuthSessionStore(): void {
	authSessionStore.set(createInitialAuthSession());
}

/**
 * Probes the cookie-backed profile endpoint and stores anonymous, expired, locked, error, or authenticated state.
 *
 * @remarks Implements DESIGN-018 AuthSessionStore startup session probing.
 */
export async function probeAuthSession(signal?: AbortSignal): Promise<AuthSessionProjection> {
	authSessionStore.set({ status: "unknown", lastCheckedAt: dependencies.now() });
	try {
		const profile = await dependencies.probeProfileSession(signal);
		const session = await dependencies.refreshAuthSession(signal);
		return setAuthSession(sessionToProjection(session, profile.displayName));
	} catch (error) {
		return setAuthSession(errorToProjection(error));
	}
}

/**
 * Persists and publishes a sanitized session projection.
 *
 * @remarks Implements DESIGN-018 AuthSessionStore frontend-safe projection storage.
 */
export function setAuthSession(session: AuthSessionProjection): AuthSessionProjection {
	const projection = sanitizeProjection(session);
	authSessionStore.set(projection);
	writeStoredProjection(projection);
	return projection;
}

/**
 * Clears authenticated user fields while preserving unrelated anonymous Catalog Search state.
 *
 * @remarks Implements DESIGN-018 AuthSessionStore logout, expired, and anonymous clearing.
 */
export function clearAuthSession(reason: "logout" | "expired" | "anonymous"): AuthSessionProjection {
	const status = reason === "expired" ? "expired" : "anonymous";
	const projection = sanitizeProjection({ status, lastCheckedAt: dependencies.now() });
	authSessionStore.set(projection);
	writeStoredProjection(projection);
	return projection;
}

/**
 * Registers with CSRF protection, refreshes entitlement state, and stores the safe authenticated projection.
 *
 * @remarks Implements DESIGN-018 AuthSessionStore registration session refresh and entitlement coordination.
 */
export async function registerWithEmail(
	request: RegisterRequest,
	signal?: AbortSignal
): Promise<AuthSessionProjection> {
	return authenticateWithMutation((csrfToken) =>
		dependencies.registerWithEmail(request, { csrfToken, signal })
	);
}

/**
 * Logs in with CSRF protection, refreshes entitlement state, and stores the safe authenticated projection.
 *
 * @remarks Implements DESIGN-018 AuthSessionStore login session refresh and entitlement coordination.
 */
export async function loginWithEmail(request: LoginRequest, signal?: AbortSignal): Promise<AuthSessionProjection> {
	return authenticateWithMutation((csrfToken) => dependencies.loginWithEmail(request, { csrfToken, signal }));
}

/**
 * Logs out via the API, then clears only auth projection state.
 *
 * @remarks Implements DESIGN-018 AuthSessionStore logout clearing with Catalog Search preservation.
 */
export async function logoutCurrentSession(signal?: AbortSignal): Promise<void> {
	authSessionStore.set({ ...get(authSessionStore), status: "authenticating" });
	try {
		const { csrfToken } = await dependencies.fetchCsrfToken(signal);
		await dependencies.logoutCurrentSession({ csrfToken, signal });
		clearAuthSession("logout");
	} catch (error) {
		setAuthSession(errorToProjection(error));
		throw error;
	}
}

/**
 * Refreshes server session and entitlement after an OAuth return; URL parameters are ignored.
 *
 * @remarks Implements DESIGN-018 AuthSessionStore OAuth-return refresh without URL-param success inference.
 */
export async function refreshAuthSessionAfterOAuthReturn(
	_returnUrl?: string | URL,
	signal?: AbortSignal
): Promise<AuthSessionProjection> {
	authSessionStore.set({ ...get(authSessionStore), status: "authenticating" });
	try {
		const session = await dependencies.refreshAuthSession(signal);
		const projection = setAuthSession(sessionToProjection(session));
		await refreshEntitlement();
		return projection;
	} catch (error) {
		const projection = setAuthSession(errorToProjection(error));
		throw Object.assign(error instanceof Error ? error : new Error("OAuth session refresh failed"), { projection });
	}
}

/**
 * Resets auth store state between tests without clearing browser storage.
 *
 * @remarks Implements DESIGN-018 AuthSessionStore deterministic state reset.
 */
export function resetAuthSessionStore(): void {
	dependencies = defaultDependencies;
	authSessionStore.set(createInitialAuthSession());
}

/** Test-only dependency replacement for API and clock boundaries. */
export function setAuthSessionDependencies(patch: Partial<AuthSessionDependencies>): void {
	dependencies = { ...defaultDependencies, ...patch };
}

async function authenticateWithMutation(
	mutate: (csrfToken: string) => Promise<AuthSessionData>
): Promise<AuthSessionProjection> {
	authSessionStore.set({ ...get(authSessionStore), status: "authenticating" });
	try {
		const { csrfToken } = await dependencies.fetchCsrfToken();
		const session = await mutate(csrfToken);
		const projection = setAuthSession(sessionToProjection(session));
		await refreshEntitlement();
		return projection;
	} catch (error) {
		const projection = setAuthSession(errorToProjection(error));
		throw Object.assign(error instanceof Error ? error : new Error("Authentication failed"), { projection });
	}
}

async function refreshEntitlement(): Promise<void> {
	try {
		dependencies.setEntitlementStatus(await dependencies.refreshEntitlementAfterAuth());
	} catch (error) {
		const appError = extractAppError(error);
		if (appError) {
			dependencies.setEntitlementError(appError);
		}
	}
}

function sessionToProjection(session: AuthSessionData, displayName?: string): AuthSessionProjection {
	return sanitizeProjection({
		status: "authenticated",
		userId: session.userId,
		displayName,
		role: session.role,
		hasVerifiedLoginMethod: session.hasVerifiedLoginMethod,
		lastCheckedAt: dependencies.now()
	});
}

function errorToProjection(error: unknown): AuthSessionProjection {
	const appError = extractAppError(error);
	const status = mapAuthErrorToStatus(error, appError);
	return sanitizeProjection({ status, error: appError, lastCheckedAt: dependencies.now() });
}

function mapAuthErrorToStatus(error: unknown, appError: AppError | undefined): AuthStatus {
	const statusCode = error instanceof authApi.AuthClientError ? error.status : undefined;
	if (statusCode === 401) {
		return appError?.code === "session_expired" || appError?.code === "session_invalid" ? "expired" : "anonymous";
	}
	if (statusCode === 403 && (appError?.code.includes("locked") || appError?.code.includes("lockout"))) {
		return "locked";
	}
	if (statusCode === 429) {
		return "locked";
	}
	return "error";
}

function extractAppError(error: unknown): AppError | undefined {
	return error instanceof authApi.AuthClientError ? error.appError : undefined;
}

function sanitizeProjection(session: AuthSessionProjection): AuthSessionProjection {
	const projection: AuthSessionProjection = { status: session.status };
	if (session.userId) {
		projection.userId = session.userId;
	}
	if (session.email) {
		projection.email = session.email;
	}
	if (session.displayName) {
		projection.displayName = session.displayName;
	}
	if (session.hasVerifiedLoginMethod !== undefined) {
		projection.hasVerifiedLoginMethod = session.hasVerifiedLoginMethod;
	}
	if (session.role) {
		projection.role = session.role;
	}
	if (session.lastCheckedAt) {
		projection.lastCheckedAt = session.lastCheckedAt;
	}
	if (session.error) {
		projection.error = session.error;
	}
	return projection;
}

function readStoredProjection(): AuthSessionProjection | null {
	if (typeof window === "undefined") {
		return null;
	}
	try {
		const raw = window.sessionStorage.getItem(AUTH_SESSION_STORAGE_KEY);
		if (raw === null) {
			return null;
		}
		const parsed = JSON.parse(raw) as unknown;
		return isStoredProjection(parsed) ? parsed : null;
	} catch {
		return null;
	}
}

function writeStoredProjection(session: AuthSessionProjection): void {
	if (typeof window === "undefined") {
		return;
	}
	try {
		if (session.status === "authenticated") {
			window.sessionStorage.setItem(AUTH_SESSION_STORAGE_KEY, JSON.stringify(sanitizeProjection(session)));
		} else {
			window.sessionStorage.removeItem(AUTH_SESSION_STORAGE_KEY);
		}
	} catch {
		// Storage is optional; HttpOnly-cookie auth remains usable with the in-memory store.
		return;
	}
}

function isStoredProjection(value: unknown): value is AuthSessionProjection {
	if (typeof value !== "object" || value === null) {
		return false;
	}
	const candidate = value as AuthSessionProjection;
	return (
		candidate.status === "authenticated" &&
		typeof candidate.userId === "string" &&
		(candidate.displayName === undefined || typeof candidate.displayName === "string") &&
		(candidate.hasVerifiedLoginMethod === undefined || typeof candidate.hasVerifiedLoginMethod === "boolean") &&
		(candidate.role === undefined || candidate.role === "user" || candidate.role === "admin")
	);
}
