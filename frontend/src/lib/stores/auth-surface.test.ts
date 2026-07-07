import { afterEach, expect, test } from "bun:test";
import { get } from "svelte/store";

import {
	authSurfaceStore,
	buildAuthGuardDecision,
	closeAuthSurface,
	openLoginSurface,
	queueProtectedAction,
	requestProtectedAction,
	resetAuthSurface,
	runQueuedProtectedActionAfterAuth
} from "./auth-surface";
import type { AuthSessionProjection } from "./auth-session";

// Implements DESIGN-018 AuthenticatedActionGuard pending protected-action verification.

afterEach(() => {
	resetAuthSurface();
});

test("opens and closes the login surface without a pending action", () => {
	openLoginSurface();
	expect(get(authSurfaceStore)).toEqual({ open: true, pendingAction: null });

	closeAuthSurface();
	expect(get(authSurfaceStore)).toEqual({ open: false, pendingAction: null });
});

test("queues one protected action and runs it exactly once after auth", async () => {
	let runs = 0;
	queueProtectedAction({
		kind: "checkout",
		label: "Continue checkout",
		continueAfterAuth: async () => {
			runs += 1;
		}
	});

	expect(get(authSurfaceStore).open).toBe(true);
	expect(get(authSurfaceStore).pendingAction?.label).toBe("Continue checkout");

	await runQueuedProtectedActionAfterAuth();
	await runQueuedProtectedActionAfterAuth();

	expect(runs).toBe(1);
	expect(get(authSurfaceStore)).toEqual({ open: false, pendingAction: null });
});

test("builds guard decisions for authenticated, anonymous, expired, locked, unverified, and unresolved sessions", () => {
	const action = checkoutAction();

	expect(buildAuthGuardDecision(session("authenticated"), action)).toEqual({ allowed: true });
	expect(buildAuthGuardDecision({ status: "authenticated", userId: "user-1" }, action)).toEqual({
		allowed: false,
		reason: "unverified",
		signInAction: action
	});
	expect(buildAuthGuardDecision({ ...session("authenticated"), hasVerifiedLoginMethod: false }, action)).toEqual({
		allowed: false,
		reason: "unverified",
		signInAction: action
	});
	expect(buildAuthGuardDecision(session("anonymous"), action)).toEqual({
		allowed: false,
		reason: "anonymous",
		signInAction: action
	});
	expect(buildAuthGuardDecision(session("expired"), action)).toEqual({
		allowed: false,
		reason: "expired",
		signInAction: action
	});
	expect(buildAuthGuardDecision(session("locked"), action)).toEqual({
		allowed: false,
		reason: "locked",
		signInAction: action
	});
	for (const status of ["unknown", "authenticating", "error"] as const) {
		expect(buildAuthGuardDecision(session(status), action)).toEqual({
			allowed: false,
			reason: "anonymous",
			signInAction: action
		});
	}
});

test("requestProtectedAction opens guidance and canceling auth clears the queued action", () => {
	const decision = requestProtectedAction(session("anonymous"), checkoutAction());

	expect(decision.allowed).toBe(false);
	expect(get(authSurfaceStore).open).toBe(true);
	expect(get(authSurfaceStore).pendingAction?.kind).toBe("checkout");

	closeAuthSurface();

	expect(get(authSurfaceStore)).toEqual({ open: false, pendingAction: null });
});

function checkoutAction() {
	return {
		kind: "checkout" as const,
		label: "Continue checkout",
		continueAfterAuth: async () => undefined
	};
}

function session(status: AuthSessionProjection["status"]): AuthSessionProjection {
	return {
		status,
		userId: status === "authenticated" ? "user-1" : undefined,
		hasVerifiedLoginMethod: status === "authenticated" ? true : undefined
	};
}
