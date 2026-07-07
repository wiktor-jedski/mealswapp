import { getOAuthStartUrl } from "../api/auth-client";
import type { OAuthProvider } from "../api/generated";
import {
	refreshAuthSessionAfterOAuthReturn,
	type AuthSessionProjection
} from "../stores/auth-session";

// Implements DESIGN-018 OAuthEntryPoint generated-provider routing and callback refresh orchestration.

export interface OAuthEntryDependencies {
	getOAuthStartUrl: (provider: OAuthProvider, returnTo?: string) => string;
	navigate: (url: string) => void;
	refreshAuthSessionAfterOAuthReturn: typeof refreshAuthSessionAfterOAuthReturn;
	currentReturnPath: () => string;
}

export interface OAuthEntryResult {
	ok: boolean;
	url?: string;
	errorMessage?: string;
}

const defaultDependencies: OAuthEntryDependencies = {
	getOAuthStartUrl,
	navigate: (url) => {
		window.location.assign(url);
	},
	refreshAuthSessionAfterOAuthReturn,
	currentReturnPath: () => `${window.location.pathname}${window.location.search}`
};

const safeUnavailableMessage = "This sign-in provider is temporarily unavailable. Please try another sign-in method.";

/** Starts provider login only through generated backend routes and never accepts client-side provider secrets. */
export function startOAuthProvider(
	provider: OAuthProvider,
	dependencies: OAuthEntryDependencies = defaultDependencies
): OAuthEntryResult {
	try {
		const returnTo = safeClientReturnPath(dependencies.currentReturnPath());
		const url = dependencies.getOAuthStartUrl(provider, returnTo);
		if (!isSafeProviderStartUrl(provider, url)) {
			return { ok: false, errorMessage: safeUnavailableMessage };
		}
		dependencies.navigate(url);
		return { ok: true, url };
	} catch {
		return { ok: false, errorMessage: safeUnavailableMessage };
	}
}

/** Refreshes cookie-backed session state after an OAuth callback without trusting URL parameters. */
export async function refreshOAuthCallbackSession(
	returnUrl: string | URL,
	dependencies: OAuthEntryDependencies = defaultDependencies,
	signal?: AbortSignal
): Promise<AuthSessionProjection> {
	return dependencies.refreshAuthSessionAfterOAuthReturn(returnUrl, signal);
}

function isSafeProviderStartUrl(provider: OAuthProvider, url: string): boolean {
	if (provider !== "google") {
		return false;
	}
	if (!url.startsWith(`/api/v1/auth/oauth/${provider}/start`)) {
		return false;
	}
	const lower = url.toLowerCase();
	if (["client_secret", "access_token", "refresh_token", "id_token", "code="].some((secretName) =>
		lower.includes(secretName)
	)) {
		return false;
	}
	try {
		const parsed = new URL(url, "http://frontend.local");
		if (parsed.origin !== "http://frontend.local" || parsed.pathname !== `/api/v1/auth/oauth/${provider}/start`) {
			return false;
		}
		const returnTo = parsed.searchParams.get("return_to");
		return returnTo === null || safeClientReturnPath(returnTo) === returnTo;
	} catch {
		return false;
	}
}

function safeClientReturnPath(value: string): string {
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
