import { expect, test } from "bun:test";

import { resolveAdminAccess, verifiedAdminIdentity } from "./admin-access";
import type { AuthSessionProjection } from "./stores/auth-session";

// Implements DESIGN-009 UserAdminPanel fail-closed access-state verification.

test("allows only a verified admin session", () => {
	const admin = session({ status: "authenticated", userId: "admin-1", role: "admin", hasVerifiedLoginMethod: true });
	expect(resolveAdminAccess(admin)).toBe("allowed");
	expect(verifiedAdminIdentity(admin)).toBe("admin-1");
});

test("keeps startup and mutation states local to the loading boundary", () => {
	expect(resolveAdminAccess(session({ status: "unknown" }))).toBe("loading");
	expect(resolveAdminAccess(session({ status: "authenticating", role: "admin" }))).toBe("loading");
});

test("fails closed for anonymous, standard, unverified, and malformed admin sessions", () => {
	const denied: AuthSessionProjection[] = [
		session({ status: "anonymous" }),
		session({ status: "authenticated", userId: "user-1", role: "user", hasVerifiedLoginMethod: true }),
		session({ status: "authenticated", userId: "admin-1", role: "admin", hasVerifiedLoginMethod: false }),
		session({ status: "authenticated", role: "admin", hasVerifiedLoginMethod: true })
	];
	for (const candidate of denied) {
		expect(resolveAdminAccess(candidate)).toBe("denied");
		expect(verifiedAdminIdentity(candidate)).toBeNull();
	}
});

test("isolates session probe failures in the administration error boundary", () => {
	expect(resolveAdminAccess(session({ status: "error" }))).toBe("error");
	expect(resolveAdminAccess(session({
		status: "authenticated",
		userId: "admin-1",
		role: "admin",
		hasVerifiedLoginMethod: true,
		error: { category: "dependency", code: "probe_failed", message: "Safe failure", retryable: true }
	}))).toBe("error");
});

function session(patch: Partial<AuthSessionProjection>): AuthSessionProjection {
	return { status: "anonymous", ...patch };
}
