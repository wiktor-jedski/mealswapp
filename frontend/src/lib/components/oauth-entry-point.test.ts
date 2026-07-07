import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

import type { OAuthProvider } from "../api/generated";
import { refreshOAuthCallbackSession, startOAuthProvider } from "./oauth-entry-point";

// Implements DESIGN-018 OAuthEntryPoint unit verification for generated routes and callback refresh.

const source = readFileSync(join(import.meta.dir, "OAuthEntryPoint.svelte"), "utf8");

function dependencies(patch: {
	getOAuthStartUrl?: (provider: OAuthProvider, returnTo?: string) => string;
	navigate?: (url: string) => void;
	currentReturnPath?: () => string;
	refreshAuthSessionAfterOAuthReturn?: (returnUrl?: string | URL, signal?: AbortSignal) => Promise<{
		status: "authenticated";
		userId: string;
	}>;
} = {}) {
	return {
		getOAuthStartUrl:
			patch.getOAuthStartUrl ??
			((provider: OAuthProvider, returnTo = "/") =>
				returnTo === "/"
					? `/api/v1/auth/oauth/${provider}/start`
					: `/api/v1/auth/oauth/${provider}/start?return_to=${encodeURIComponent(returnTo)}`),
		navigate: patch.navigate ?? (() => undefined),
		currentReturnPath: patch.currentReturnPath ?? (() => "/"),
		refreshAuthSessionAfterOAuthReturn:
			patch.refreshAuthSessionAfterOAuthReturn ??
			(async () => ({ status: "authenticated" as const, userId: "oauth-user" }))
	};
}

test("Google provider routes through generated backend OAuth start endpoint with current return path", () => {
	const navigations: string[] = [];

	expect(
		startOAuthProvider(
			"google",
			dependencies({
				currentReturnPath: () => "/subscription?plan=annual",
				navigate: (url) => navigations.push(url)
			})
		)
	).toEqual({ ok: true, url: "/api/v1/auth/oauth/google/start?return_to=%2Fsubscription%3Fplan%3Dannual" });
	expect(
		startOAuthProvider(
			"apple",
			dependencies({
				navigate: (url) => navigations.push(url)
			})
		)
	).toEqual({
		ok: false,
		errorMessage: "This sign-in provider is temporarily unavailable. Please try another sign-in method."
	});
	expect(navigations).toEqual(["/api/v1/auth/oauth/google/start?return_to=%2Fsubscription%3Fplan%3Dannual"]);
	expect(navigations.join(" ")).not.toContain("client_secret");
});

test("provider-unavailable and unsafe URLs fail closed with user-safe messaging", () => {
	const navigations: string[] = [];
	const result = startOAuthProvider(
		"google",
		dependencies({
			getOAuthStartUrl: () => "/api/v1/auth/oauth/google/start?client_secret=bad",
			navigate: (url) => navigations.push(url)
		})
	);

	expect(result.ok).toBe(false);
	expect(result.errorMessage).toBe("This sign-in provider is temporarily unavailable. Please try another sign-in method.");
	expect(navigations).toEqual([]);
});

test("unsafe return paths are reduced before navigation", () => {
	const navigations: string[] = [];
	const result = startOAuthProvider(
		"google",
		dependencies({
			currentReturnPath: () => "https://evil.test/steal",
			navigate: (url) => navigations.push(url)
		})
	);

	expect(result).toEqual({ ok: true, url: "/api/v1/auth/oauth/google/start" });
	expect(navigations).toEqual(["/api/v1/auth/oauth/google/start"]);
});

test("callback-return handling refreshes server session without inferring success from URL parameters", async () => {
	const calls: Array<string | URL | undefined> = [];
	const projection = await refreshOAuthCallbackSession(
		"https://app.test/auth/callback?success=true&code=opaque",
		dependencies({
			refreshAuthSessionAfterOAuthReturn: async (returnUrl) => {
				calls.push(returnUrl);
				return { status: "authenticated", userId: "oauth-user" };
			}
		})
	);

	expect(projection).toEqual({ status: "authenticated", userId: "oauth-user" });
	expect(calls).toEqual(["https://app.test/auth/callback?success=true&code=opaque"]);
});

test("component declares Google-only action plus callback refresh feedback", () => {
	expect(source).toContain("<!-- Implements DESIGN-018 OAuthEntryPoint");
	expect(source).toContain('mode?: "login" | "register"');
	expect(source).toContain('mode === "register" ? "Register with a provider" : "Sign in with a provider"');
	expect(source).toContain('data-oauth-provider={provider.id}');
	expect(source).toContain('label: "Google"');
	expect(source).not.toContain("Continue with Apple");
	expect(source).toContain("refreshOAuthCallbackSession(window.location.href");
	expect(source).toContain("We could not finish sign-in. Please try again.");
});

test("Google action follows custom Google button visual requirements", () => {
	expect(source).toContain("border-[#747775]");
	expect(source).toContain("bg-white");
	expect(source).toContain("text-[#1f1f1f]");
	expect(source).toContain("h-10");
	expect(source).toContain("items-center justify-center gap-3");
	expect(source).toContain("whitespace-nowrap");
	expect(source).toContain('viewBox="0 0 18 18"');
	expect(source).toContain('fill="#4285F4"');
	expect(source).toContain('fill="#34A853"');
	expect(source).toContain('fill="#FBBC05"');
	expect(source).toContain('fill="#EA4335"');
});
