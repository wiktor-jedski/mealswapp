import type { AuthSessionProjection } from "./stores/auth-session";

// Implements DESIGN-009 UserAdminPanel client visibility and route gating from the server-verified session projection.

/** Presentation state for the administration route; backend authorization remains authoritative for every API call. */
export type AdminAccessState = "loading" | "allowed" | "denied" | "error";

/** Resolves fail-closed administration visibility from the current authenticated session projection. */
export function resolveAdminAccess(session: AuthSessionProjection): AdminAccessState {
	if (session.status === "unknown" || session.status === "authenticating") return "loading";
	if (session.status === "error" || (session.status === "authenticated" && session.error)) return "error";
	return session.status === "authenticated" && !!session.userId && session.role === "admin" && session.hasVerifiedLoginMethod === true
		? "allowed"
		: "denied";
}

/** Returns the verified administrator identity used to reset feature-local state on account changes. */
export function verifiedAdminIdentity(session: AuthSessionProjection): string | null {
	return resolveAdminAccess(session) === "allowed" ? session.userId ?? null : null;
}
