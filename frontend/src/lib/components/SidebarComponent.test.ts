import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-001 SidebarComponent static-source verification.
//
// Bun's isolated install-cache layout breaks transitive resolution for
// `svelte/server`/`svelte/compiler`, and no DOM library (jsdom/happy-dom) is
// installed, so the component cannot be rendered in a Bun unit test. Instead
// these tests assert the Svelte source declares the documented sidebar
// behaviors: desktop-left placement, collapse/expand toggle, mobile toggle,
// authenticated history and favorites loaded through generated Phase 03
// contracts, unit preference control, anonymous sign-in guidance, authenticated Search/Subscription
// navigation, history-entry selection restoring search state, and API failures that never block core search. Sidebar search-mode buttons
// are intentionally omitted because mode switching lives in the main view. `vite build`
// compiles the component, validating the Svelte source at build time.

const source = readFileSync(join(import.meta.dir, "SidebarComponent.svelte"), "utf8");
const appCss = readFileSync(join(import.meta.dir, "../../app.css"), "utf8");

function countOccurrences(haystack: string, needle: string): number {
	return haystack.split(needle).length - 1;
}

// Implements DESIGN-001 SidebarComponent desktop-left placement verification.
test("renders a desktop-left aside with sticky left-column placement classes", () => {
	expect(source).toContain("desktop-sidebar-left");
	expect(source).toContain('aria-label="Activity sidebar"');
	expect(source).toContain("sm:sticky");
	expect(source).toContain("sm:top-0");
	expect(source).toContain("sm:h-screen");
	expect(source).toContain("sm:border-r");
	expect(source).toContain('data-sidebar');
});

// Implements DESIGN-001 SidebarComponent collapse/expand behavior verification.
test("declares a desktop collapse toggle that hides content when collapsed", () => {
	expect(source).toContain("sidebar-collapse-toggle");
	expect(source).toContain("data-sidebar-collapse");
	expect(source).toContain("toggleCollapsed");
	expect(source).toContain("$sidebarStore.collapsed");
	expect(source).toContain('aria-expanded={!$sidebarStore.collapsed}');
	expect(source).toContain('$sidebarStore.collapsed ? "»" : "«"');
	// Collapse shrinks the desktop width and hides the inner content block on sm+.
	expect(source).toContain("sm:w-14");
	expect(source).toContain("sm:w-60");
	expect(source).not.toContain("sm:p-2");
	expect(source).toContain("transition-[width,padding-right]");
	expect(source).toContain("motion-reduce:transition-none");
	expect(source).toContain("sidebar-animated-content");
	expect(appCss).toContain("opacity 150ms ease-out 200ms");
	expect(appCss).toContain("visibility 0s linear 200ms");
	expect(appCss).toContain("[data-collapsed=\"true\"] .sidebar-animated-content");
	expect(appCss).toContain("opacity 0s linear");
});

// Implements DESIGN-001 SidebarComponent mobile toggle behavior verification.
test("declares a mobile-only open toggle and a mobile-only close button bound to mobileOpen", () => {
	expect(source).toContain("mobile-sidebar-toggle");
	expect(source).toContain("data-sidebar-mobile-toggle");
	expect(source).toContain("sm:hidden");
	expect(source).toContain("self-center");
	expect(source).toContain('aria-label="Open activity sidebar"');
	expect(source).toContain('<span aria-hidden="true">☰</span>');
	expect(source).not.toContain("☰ Activity");
	expect(source).toContain("toggleMobileOpen");
	expect(source).toContain("sidebar-mobile-close");
	expect(source).toContain("data-sidebar-mobile-close");
	expect(source).toContain("setMobileOpen(false)");
	expect(source).toContain("$sidebarStore.mobileOpen");
	// Mobile open/closed gates content visibility on small screens.
	expect(source).toContain("$sidebarStore.mobileOpen ? 'flex' : 'hidden'");
	expect(source).toContain("flex-1 flex-col");
});

// Implements DESIGN-001 SidebarComponent authenticated contract loading verification.
test("imports generated Phase 03 contract types and fetches history and favorites with credentials after session auth", () => {
	expect(source).toContain('import type {');
	expect(source).toContain("SearchHistoryEnvelope");
	expect(source).toContain("SavedItemsEnvelope");
	expect(source).toContain("SearchHistoryEntry");
	expect(source).toContain("SavedItem");
	expect(source).toContain('from "../api/generated"');
	expect(source).toContain('import { authSessionStore, clearAuthSession } from "../stores/auth-session"');
	expect(source).toContain('import { buildAuthGuardDecision } from "../stores/auth-surface"');
	expect(source).toContain("sidebarProtectedActionsAllowed()");
	expect(source).toContain("buildAuthGuardDecision($authSessionStore");
	expect(source).toContain('kind: "saved_data"');
	expect(source).not.toContain('"/api/v1/profile"');
	expect(source).toContain('"/api/v1/search-history"');
	expect(source).toContain('"/api/v1/saved-items?kind=favorite"');
	expect(source).toContain("credentials: \"include\"");
	// Two credentialed GETs: history and favorites only after the authenticated-action guard allows sidebar data.
	expect(countOccurrences(source, "credentials: \"include\"")).toBe(2);
});

// Implements DESIGN-018 AuthenticatedActionGuard sidebar sign-in entry verification.
test("renders a sidebar sign-in action when the session store reports anonymous", () => {
	expect(source).toContain("data-sidebar-sign-in");
	expect(source).toContain("onSignIn");
	expect(source).toContain("w-full rounded bg-[var(--color-primary)] px-3 py-2 text-sm font-semibold text-[var(--color-on-primary)]");
	expect(source).not.toContain("Sign in to see your history and favorites.");
	// Authenticated state is gated by AuthSessionStore so anonymous users see guidance, not protected calls.
	expect(source).toContain("authenticating");
	expect(source).toContain("authenticated");
	expect(source).toContain('$authSessionStore.status === "unknown"');
	expect(source).toContain("response.status === 401");
	expect(source).toContain('clearAuthSession("expired")');
});

// Implements DESIGN-001 SidebarComponent authenticated Search and Subscription navigation verification.
test("declares authenticated Search and Subscription sidebar links only in the authenticated branch", () => {
	const navPos = source.indexOf("data-sidebar-navigation");
	const authenticatedBranchPos = source.indexOf("{:else}");
		const signOutPos = source.indexOf("data-sidebar-sign-out");
		const historyPos = source.indexOf("data-sidebar-history");
		const unitsPos = source.indexOf("data-sidebar-units");
		expect(signOutPos).toBeGreaterThan(unitsPos);
		expect(navPos).toBeGreaterThan(signOutPos);
		expect(navPos).toBeGreaterThan(authenticatedBranchPos);
		expect(navPos).toBeLessThan(historyPos);
		expect(source).toContain("Implements DESIGN-016 ComponentStyles handheld focus order for account navigation after sign-out");
	expect(source).toContain('aria-label="Account navigation"');
	expect(source).toContain("data-sidebar-nav-search");
	expect(source).toContain("data-sidebar-nav-subscription");
	expect(source).toContain("data-sidebar-sign-out");
	expect(source).toContain("Search");
	expect(source).toContain("Subscription");
	expect(source).toContain("Sign out");
	expect(source).toContain("onNavigateSearch");
	expect(source).toContain("onNavigateSubscription");
	expect(source).toContain("onSignOut");
	expect(source).toContain("activeView === 'search'");
	expect(source).toContain("activeView === 'subscription'");
	expect(source).toContain("border-[var(--color-primary)] text-[var(--color-text)]");
	expect(source).toContain("border-transparent text-[var(--color-muted)]");
	expect(source).not.toContain("activeView === 'subscription' ? 'bg-[var(--color-secondary)]");
	expect(source).toContain('aria-current={activeView === "search" ? "page" : undefined}');
	expect(source).toContain('aria-current={activeView === "subscription" ? "page" : undefined}');
});

// Implements DESIGN-016 ComponentStyles legal sidebar footer navigation verification.
test("declares sidebar footer links for Privacy Policy and Terms of Service", () => {
	expect(source).toContain("data-sidebar-legal");
	expect(source).toContain("data-sidebar-nav-privacy");
	expect(source).toContain("data-sidebar-nav-terms");
	expect(source).toContain("Privacy Policy");
	expect(source).toContain("Terms of Service");
	expect(source).toContain("onNavigatePrivacy");
	expect(source).toContain("onNavigateTerms");
	expect(source).toContain("onLegalNavigationSelect");
	expect(source).toContain('aria-current={activeView === "privacy" ? "page" : undefined}');
	expect(source).toContain('aria-current={activeView === "terms" ? "page" : undefined}');
});

// Implements DESIGN-001 SidebarComponent mobile navigation usability verification.
test("sidebar navigation closes the mobile drawer after account navigation", () => {
	expect(source).toContain("onSidebarNavigationSelect");
	expect(source).toContain("setMobileOpen(false)");
	expect(source).toContain('onclick={() => onSidebarNavigationSelect("search")}');
	expect(source).toContain('onclick={() => onSidebarNavigationSelect("subscription")}');
	expect(source).toContain('onclick={() => onLegalNavigationSelect("privacy")}');
	expect(source).toContain('onclick={() => onLegalNavigationSelect("terms")}');
});

// Implements DESIGN-001 SidebarComponent history entry selection restoring search state verification.
test("selecting a history entry calls setQuery with the query and setMode with the validated mode", () => {
	expect(source).toContain("onHistoryEntrySelect");
	expect(source).toContain("setQuery(entry.query)");
	expect(source).toContain("setMode(entry.mode)");
	expect(source).toContain("isSearchMode(entry.mode)");
	expect(source).toContain("onNavigateSearch()");
	expect(source).toContain("onclick={() => onHistoryEntrySelect(entry)}");
	expect(source).toContain("data-sidebar-history-entry={entry.id}");
	expect(source).toContain("w-full truncate rounded border border-transparent px-3 py-1 text-left text-sm");
});

// Implements DESIGN-001 SidebarComponent API failures never block core search verification.
test("wraps history and favorites fetches in try/catch that sets inline error state instead of throwing", () => {
	// Each protected sidebar loader has a try block and a catch that assigns a local error string.
	expect(countOccurrences(source, "} catch {")).toBeGreaterThanOrEqual(2);
	expect(source).toContain("historyError =");
	expect(source).toContain("favoritesError =");
	expect(source).toContain("data-sidebar-history-error");
	expect(source).toContain("data-sidebar-favorites-error");
	// The component must not rethrow or expose a throw that propagates to the parent.
	expect(source).not.toContain("throw new Error");
	expect(source).not.toContain("throw new SearchClientError");
});

// Implements DESIGN-001 SidebarComponent duplicate search-mode navigation removal verification.
test("does not declare duplicate search-mode buttons in the sidebar", () => {
	expect(source).not.toContain("data-sidebar-modes");
	expect(source).not.toContain('aria-label="Search mode navigation"');
	expect(source).not.toContain('id: "sidebar-mode-catalog"');
	expect(source).not.toContain('id: "sidebar-mode-substitution"');
	expect(source).not.toContain('id: "sidebar-mode-daily-diet"');
	expect(source).not.toContain("onModeSelect");
	expect(source).not.toContain("searchStore");
});

// Implements DESIGN-001 SidebarComponent compact account-section heading verification.
test("history and favorites use the standard compact h3 heading style", () => {
	expect(source).toContain('<h3 class="text-base font-semibold text-[var(--color-text)]">History</h3>');
	expect(source).toContain('<h3 class="text-base font-semibold text-[var(--color-text)]">Favorites</h3>');
	expect(source).not.toContain('class="font-data text-xs uppercase text-[var(--color-muted)]">History');
});

// Implements DESIGN-001 SidebarComponent unit preference row verification.
test("declares a compact account-level unit preference row", () => {
	expect(source).toContain('import { preferencesStore, setUnitSystem } from "../stores/preferences"');
	expect(source).toContain('data-sidebar-units');
	expect(source).toContain('for="sidebar-unit-system"');
	expect(source).toContain("Units:");
	expect(source).toContain("$preferencesStore.unitSystem");
	expect(source).toContain("setUnitSystem");
	expect(source).toContain('value: "metric"');
	expect(source).toContain('value: "imperial"');
	expect(source).not.toContain("<SettingsPanel");
	expect(source).not.toContain("data-sidebar-settings");
});

// Implements DESIGN-016 ThemeProvider binary sidebar switch verification.
test("declares a binary light/dark theme switch directly under the sidebar brand", () => {
	const brandPos = source.indexOf('<h1 class="text-2xl font-semibold">{branding}</h1>');
	const switchPos = source.indexOf("data-sidebar-theme-toggle");
	expect(brandPos).toBeGreaterThan(-1);
	expect(switchPos).toBeGreaterThan(brandPos);
	expect(source).toContain('import { resolvedTheme, setThemePreference } from "../stores/theme"');
	expect(source).toContain("onThemeToggle");
	expect(source).toContain('aria-label="Theme preference"');
	expect(source).toContain('aria-pressed={$resolvedTheme === "dark"}');
	expect(source).toContain('setThemePreference($resolvedTheme === "dark" ? "light" : "dark")');
	expect(source).toContain("Current theme: {$resolvedTheme}");
	expect(source).toContain("<svg");
});

// Implements DESIGN-001 SidebarComponent traceability verification.
test("cites the DESIGN-001 SidebarComponent source", () => {
	expect(source).toContain("<!-- Implements DESIGN-001 SidebarComponent -->");
	expect(source).toContain("Implements DESIGN-001 SidebarComponent");
});
