import { get, writable } from "svelte/store";
import type { AuthSessionProjection } from "./auth-session";

// Implements DESIGN-018 AuthView and AuthenticatedActionGuard login-only auth surface state.

export type ProtectedActionKind = "checkout" | "entitlement_refresh" | "profile" | "saved_data" | "account";

export interface AuthenticatedActionRequest {
	kind: ProtectedActionKind;
	label: string;
	continueAfterAuth: () => Promise<void>;
}

export interface AuthGuardDecision {
	allowed: boolean;
	reason?: "anonymous" | "expired" | "unverified" | "locked";
	signInAction?: AuthenticatedActionRequest;
}

export interface AuthSurfaceState {
	open: boolean;
	pendingAction: AuthenticatedActionRequest | null;
}

/** Login auth surface state with at most one retained protected action. */
export const authSurfaceStore = writable<AuthSurfaceState>({
	open: false,
	pendingAction: null
});

/** Opens the login surface without changing Catalog Search state. */
export function openLoginSurface(action: AuthenticatedActionRequest | null = null): void {
	authSurfaceStore.set({ open: true, pendingAction: action });
}

/** Closes the login surface and drops any pending protected action. */
export function closeAuthSurface(): void {
	authSurfaceStore.set({ open: false, pendingAction: null });
}

/** Retains one protected action for retry after successful login. */
export function queueProtectedAction(action: AuthenticatedActionRequest): void {
	openLoginSurface(action);
}

/** Builds the protected-action decision from frontend-safe auth state. */
export function buildAuthGuardDecision(
	session: AuthSessionProjection,
	action: AuthenticatedActionRequest
): AuthGuardDecision {
	if (session.status === "authenticated" && session.hasVerifiedLoginMethod === true) {
		return { allowed: true };
	}
	if (session.status === "authenticated") {
		return { allowed: false, reason: "unverified", signInAction: action };
	}
	if (session.status === "expired") {
		return { allowed: false, reason: "expired", signInAction: action };
	}
	if (session.status === "locked") {
		return { allowed: false, reason: "locked", signInAction: action };
	}
	if (session.status === "anonymous") {
		return { allowed: false, reason: "anonymous", signInAction: action };
	}
	return { allowed: false, reason: "anonymous", signInAction: action };
}

/** Queues protected actions that require sign-in and returns whether the caller may continue. */
export function requestProtectedAction(
	session: AuthSessionProjection,
	action: AuthenticatedActionRequest
): AuthGuardDecision {
	const decision = buildAuthGuardDecision(session, action);
	if (!decision.allowed && decision.signInAction) {
		queueProtectedAction(decision.signInAction);
	}
	return decision;
}

/** Runs the retained protected action exactly once after authentication succeeds. */
export async function runQueuedProtectedActionAfterAuth(): Promise<void> {
	const action = get(authSurfaceStore).pendingAction;
	authSurfaceStore.set({ open: false, pendingAction: null });
	if (action) {
		await action.continueAfterAuth();
	}
}

/** Resets login auth surface state between tests. */
export function resetAuthSurface(): void {
	authSurfaceStore.set({ open: false, pendingAction: null });
}
