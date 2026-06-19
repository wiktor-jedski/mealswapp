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
// contracts, anonymous sign-in guidance, history-entry selection restoring
// search state, and API failures that never block core search. `vite build`
// compiles the component, validating the Svelte source at build time.

const source = readFileSync(join(import.meta.dir, "SidebarComponent.svelte"), "utf8");

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
	expect(source).toContain("$sidebarStore.collapsed ? 'sm:hidden' : 'sm:grid'");
});

// Implements DESIGN-001 SidebarComponent mobile toggle behavior verification.
test("declares a mobile-only open toggle and a mobile-only close button bound to mobileOpen", () => {
	expect(source).toContain("mobile-sidebar-toggle");
	expect(source).toContain("data-sidebar-mobile-toggle");
	expect(source).toContain("sm:hidden");
	expect(source).toContain("toggleMobileOpen");
	expect(source).toContain("sidebar-mobile-close");
	expect(source).toContain("data-sidebar-mobile-close");
	expect(source).toContain("setMobileOpen(false)");
	expect(source).toContain("$sidebarStore.mobileOpen");
	// Mobile open/closed gates content visibility on small screens.
	expect(source).toContain("$sidebarStore.mobileOpen ? 'block' : 'hidden'");
});

// Implements DESIGN-001 SidebarComponent authenticated contract loading verification.
test("imports generated Phase 03 contract types and fetches profile, history, and favorites with credentials", () => {
	expect(source).toContain('import type {');
	expect(source).toContain("ProfileEnvelope");
	expect(source).toContain("SearchHistoryEnvelope");
	expect(source).toContain("SavedItemsEnvelope");
	expect(source).toContain("SearchHistoryEntry");
	expect(source).toContain("SavedItem");
	expect(source).toContain("ProfileData");
	expect(source).toContain('from "../api/generated"');
	expect(source).toContain('"/api/v1/profile"');
	expect(source).toContain('"/api/v1/search-history"');
	expect(source).toContain('"/api/v1/saved-items?kind=favorite"');
	expect(source).toContain("credentials: \"include\"");
	// Three credentialed GETs: profile probe, history, and favorites.
	expect(countOccurrences(source, "credentials: \"include\"")).toBe(3);
});

// Implements DESIGN-001 SidebarComponent anonymous empty/sign-in guidance verification.
test("renders sign-in guidance when the profile probe returns anonymous", () => {
	expect(source).toContain("data-sidebar-anonymous");
	expect(source).toContain("Sign in to see your history and favorites.");
	// Authenticated state is gated by a profile-probe flag so anonymous users see guidance, not errors.
	expect(source).toContain("authenticating");
	expect(source).toContain("authenticated");
	expect(source).toContain("response.status === 401");
});

// Implements DESIGN-001 SidebarComponent history entry selection restoring search state verification.
test("selecting a history entry calls setQuery with the query and setMode with the validated mode", () => {
	expect(source).toContain("onHistoryEntrySelect");
	expect(source).toContain("setQuery(entry.query)");
	expect(source).toContain("setMode(entry.mode)");
	expect(source).toContain("isSearchMode(entry.mode)");
	expect(source).toContain("on:click={() => onHistoryEntrySelect(entry)}");
	expect(source).toContain("data-sidebar-history-entry={entry.id}");
});

// Implements DESIGN-001 SidebarComponent API failures never block core search verification.
test("wraps profile, history, and favorites fetches in try/catch that sets inline error state instead of throwing", () => {
	// Each of the three async loaders has a try block and a catch that assigns a local error string.
	expect(countOccurrences(source, "} catch {")).toBeGreaterThanOrEqual(3);
	expect(source).toContain("authError =");
	expect(source).toContain("historyError =");
	expect(source).toContain("favoritesError =");
	expect(source).toContain("data-sidebar-history-error");
	expect(source).toContain("data-sidebar-favorites-error");
	expect(source).toContain("data-sidebar-auth-error");
	// The component must not rethrow or expose a throw that propagates to the parent.
	expect(source).not.toContain("throw new Error");
	expect(source).not.toContain("throw new SearchClientError");
});

// Implements DESIGN-001 SidebarComponent search-mode navigation verification.
test("declares Catalog, Substitution, and Daily Diet Alternative mode buttons bound to setMode", () => {
	expect(source).toContain('id: "sidebar-mode-catalog"');
	expect(source).toContain('id: "sidebar-mode-substitution"');
	expect(source).toContain('id: "sidebar-mode-daily-diet"');
	expect(source).toContain('value: "catalog"');
	expect(source).toContain('value: "substitution"');
	expect(source).toContain('value: "daily_diet_alternative"');
	expect(source).toContain("setMode(mode)");
	expect(source).toContain("on:click={() => onModeSelect(option.value)}");
	expect(source).toContain("$searchStore.mode === option.value");
	expect(source).toContain("aria-pressed={$searchStore.mode === option.value}");
});

// Implements DESIGN-001 SidebarComponent settings entry point verification.
test("declares a settings entry point that toggles the SettingsPanel inline", () => {
	expect(source).toContain("data-sidebar-settings");
	expect(source).toContain("onSettingsToggle");
	expect(source).toContain("settingsOpen");
	expect(source).toContain("aria-expanded={settingsOpen}");
	expect(source).toContain("<SettingsPanel />");
});

// Implements DESIGN-001 SidebarComponent traceability verification.
test("cites the DESIGN-001 SidebarComponent source", () => {
	expect(source).toContain("<!-- Implements DESIGN-001 SidebarComponent -->");
	expect(source).toContain("Implements DESIGN-001 SidebarComponent");
});
