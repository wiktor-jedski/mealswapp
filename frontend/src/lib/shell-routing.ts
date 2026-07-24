import type { SearchMode } from "./api/generated";

// Implements DESIGN-001 SearchView canonical SPA routes and browser-history restoration.
// Implements DESIGN-009 UserAdminPanel canonical administration route.

export type ShellView = "search" | "subscription" | "administration" | "privacy" | "terms";
export type BillingReturnState = "success" | "cancel" | null;

export interface ShellRoute {
	view: ShellView;
	mode: SearchMode;
	billingReturn: BillingReturnState;
}

const searchModes = new Set<SearchMode>([
	"catalog",
	"substitution",
	"daily_diet",
	"daily_diet_alternative"
]);

/** Parses canonical and legacy billing-return URLs into one shell route projection. */
export function parseShellRoute(value: string | URL): ShellRoute {
	const url = value instanceof URL ? value : new URL(value, "http://frontend.local");
	if (url.pathname === "/privacy") return { view: "privacy", mode: "catalog", billingReturn: null };
	if (url.pathname === "/terms") return { view: "terms", mode: "catalog", billingReturn: null };
	if (url.pathname === "/admin") return { view: "administration", mode: "catalog", billingReturn: null };
	if (url.pathname === "/subscription") {
		return {
			view: "subscription",
			mode: "catalog",
			billingReturn: billingReturnState(url.searchParams.get("checkout"))
		};
	}
	if (url.pathname === "/billing/success" || url.pathname === "/billing/cancel") {
		return {
			view: "subscription",
			mode: "catalog",
			billingReturn: url.pathname.endsWith("success") ? "success" : "cancel"
		};
	}
	const requestedMode = url.searchParams.get("mode");
	return {
		view: "search",
		mode: isSearchMode(requestedMode) ? requestedMode : "catalog",
		billingReturn: null
	};
}

/** Returns the canonical URL for one Search mode. Catalog owns the clean root URL. */
export function searchRoute(mode: SearchMode): string {
	return mode === "catalog" ? "/" : `/?mode=${encodeURIComponent(mode)}`;
}

/** Returns the canonical URL for a top-level shell view. */
export function shellViewRoute(view: ShellView, mode: SearchMode = "catalog"): string {
	switch (view) {
		case "search": return searchRoute(mode);
		case "subscription": return "/subscription";
		case "administration": return "/admin";
		case "privacy": return "/privacy";
		case "terms": return "/terms";
	}
}

function billingReturnState(value: string | null): BillingReturnState {
	return value === "success" || value === "cancel" ? value : null;
}

function isSearchMode(value: string | null): value is SearchMode {
	return value !== null && searchModes.has(value as SearchMode);
}
