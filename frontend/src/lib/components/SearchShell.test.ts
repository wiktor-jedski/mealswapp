import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-001 SearchView shell composition verification.
//
// Static-source assertions verify the Task 151 composed shell: SidebarComponent, mode controls,
// autocomplete search bar, mode-specific controls, results, and
// offline banner in the documented visual order, plus traceability. `vite build` compiles the
// full shell, validating the composed Svelte source at build time.

const source = readFileSync(join(import.meta.dir, "SearchShell.svelte"), "utf8");

function indexOf(fragment: string): number {
	return source.indexOf(fragment);
}

// Implements DESIGN-001 SearchView composed component presence verification.
test("composes sidebar, mode controls, autocomplete, mode-specific controls, results, and offline banner", () => {
	expect(source).toContain("<SidebarComponent");
	expect(source).toContain("<SearchModes />");
	expect(source).toContain("<AutocompleteDropdown");
	expect(source).toContain("<SubstitutionInputs");
	expect(source).toContain("<DailyDietControls");
	expect(source).not.toContain("<SettingsPanel");
	expect(source).toContain("<SearchResults");
	expect(source).toContain("<OfflineBanner />");
});

// Implements DESIGN-001 SearchView and SidebarComponent subscription view separation verification.
test("renders billing controls only in the authenticated Subscription view branch", () => {
	expect(source).toContain("type ShellView = \"search\" | \"subscription\" | \"privacy\" | \"terms\"");
	expect(source).toContain("data-subscription-view");
	expect(source).toContain("activeView === \"subscription\"");
	expect(source).toContain("<SubscriptionBilling />");
	expect(source).not.toContain("subscription-view-title");
	expect(source).toContain("{:else}");
	expect(source.indexOf("<SubscriptionBilling />")).toBeLessThan(source.indexOf("<SearchModes />"));
	expect(source).toContain("openSubscriptionView");
	expect(source).toContain("requestProtectedAction($authSessionStore");
	expect(source).toContain('kind: "account"');
});

// Implements DESIGN-018 OAuthEntryPoint auth-modal composition verification.
test("renders Google OAuth entry inside the protected-action auth modal", () => {
	expect(source).toContain('import OAuthEntryPoint from "./OAuthEntryPoint.svelte"');
	expect(source).toContain("<OAuthEntryPoint mode={authSurfaceMode} callbackReturn={oauthCallbackReturn} />");
	expect(source.indexOf("<OAuthEntryPoint mode={authSurfaceMode}")).toBeLessThan(source.indexOf("<LoginView />"));
	expect(source.indexOf("<OAuthEntryPoint mode={authSurfaceMode}")).toBeLessThan(source.indexOf("<RegisterView"));
});

// Implements DESIGN-018 AuthView modal-only composition and OAuth callback handoff verification.
test("uses the modal as the sole auth surface and preserves OAuth callback refresh", () => {
	expect(source).toContain("oauthCallbackReturn?: boolean");
	expect(source).toContain("openLoginSurface()");
	expect(source).toContain("callbackReturn={oauthCallbackReturn}");
	expect(source).toContain("session.hasVerifiedLoginMethod === true");
	expect(source).not.toContain("DisclaimerPanel");
});

// Implements DESIGN-001 SearchView state preservation when returning from Subscription view.
test("passes sidebar navigation callbacks that return to search without resetting search state", () => {
	expect(source).toContain("onNavigateSearch={openSearchView}");
	expect(source).toContain("onNavigateSubscription={openSubscriptionView}");
	expect(source).toContain("onNavigatePrivacy={openPrivacyView}");
	expect(source).toContain("onNavigateTerms={openTermsView}");
	expect(source).toContain("onSignIn={() =>");
	expect(source).toContain("onSignOut={() => void signOut()}");
	expect(source).toContain("function openSearchView(): void");
	expect(source).toContain("function openPrivacyView(): void");
	expect(source).toContain("function openTermsView(): void");
	expect(source).toContain('activeView = "search"');
	expect(source).not.toContain("resetSearch");
});

// Implements DESIGN-016 ComponentStyles legal views and DESIGN-015 DisclaimerRenderer placement verification.
test("renders legal views with medical information in Terms of Service", () => {
	expect(source).toContain('path === "/privacy"');
	expect(source).toContain('path === "/terms"');
	expect(source).toContain("data-privacy-view");
	expect(source).toContain("data-terms-view");
	expect(source).toContain("Privacy Policy placeholder text.");
	expect(source).toContain("Terms of Service placeholder text.");
	expect(source).toContain("data-medical-disclaimer");
	expect(source).toContain("It does not provide medical advice");
});

// Implements DESIGN-001 SearchView entitlement query and feedback wiring verification.
test("starts the entitlement query and renders usage plus blocked-mode feedback", () => {
	expect(source).toContain("buildEntitlementQueryOptions");
	expect(source).toContain("setEntitlementStatus");
	expect(source).toContain("setEntitlementError");
	expect(source).toContain("resolveSearchEntitlement");
	expect(source).toContain("data-entitlement-usage");
	expect(source).toContain("data-entitlement-feedback");
});

// Implements DESIGN-001 SearchView documented visual order verification.
test("visual order: modes → autocomplete → mode controls → results → offline banner", () => {
	const modesPos = indexOf("<SearchModes />");
	const searchPos = indexOf("<AutocompleteDropdown");
	const resultsPos = indexOf("<SearchResults");
	const offlinePos = indexOf("<OfflineBanner />");
	expect(modesPos).toBeGreaterThan(-1);
	expect(searchPos).toBeGreaterThan(modesPos);
	expect(resultsPos).toBeGreaterThan(searchPos);
	expect(offlinePos).toBeGreaterThan(resultsPos);
});

// Implements DESIGN-001 SearchView product-facing controls verification.
test("does not expose debug-style filter or search settings sections in the main Catalog surface", () => {
	expect(source).not.toContain("Search modes</h2>");
	expect(source).not.toContain('aria-label="Search filters"');
	expect(source).not.toContain('id="filter-id"');
	expect(source).not.toContain("Add filter");
	expect(source).not.toContain('aria-label="Search settings"');
});

// Implements DESIGN-001 SearchView search bar bound to setQuery via autocomplete verification.
test("autocomplete search bar is bound to setQuery and has no disabled attribute", () => {
	expect(source).toContain("setQuery");
	expect(source).toContain("submitSearch");
	expect(source).toContain("onSubmit={onAutocompleteSubmit}");
	expect(source).toContain('activeMode !== "substitution"');
	expect(source).toContain("entitlementDecision.canExecute");
	expect(source).not.toContain("disabled");
});

// Implements DESIGN-001 SearchView submitted-search spinner wiring verification.
test("passes submitted search loading state into the autocomplete search bar", () => {
	expect(source).toContain("let searchInFlight = $state(false)");
	expect(source).toContain("searching={searchInFlight}");
	expect(source).toContain('selectFirstOnEnter={activeMode === "substitution" || activeMode === "daily_diet"}');
	expect(source).toContain("onSearchInFlightChange");
	expect(source).toContain("searchInFlight = searching");
});

// Implements DESIGN-001 SearchView selected Substitution Input hydration wiring verification.
test("hydrates substitution autocomplete selections with food-object detail data", () => {
	expect(source).toContain("fetchFoodObject");
	expect(source).toContain("hydrateSubstitutionInput(item.itemId)");
	expect(source).toContain("setSubstitutionInputItem(item)");
	expect(source).toContain("displayUnitForBasis(item.macroBasis, $preferencesStore.unitSystem)");
	expect(source).toContain("updateSubstitutionInput");
});

// Implements DESIGN-001 SearchView mode-specific placeholder guidance verification.
test("passes mode-specific placeholder guidance to the search input", () => {
	expect(source).toContain("const searchPlaceholders: Record<SearchMode, string>");
	expect(source).toContain("catalog: \"Search foods, meals, or ingredients…\"");
	expect(source).toContain("substitution: \"Search a food to add as a substitution target…\"");
	expect(source).toContain("daily_diet: \"Search meals to add to your day…\"");
	expect(source).toContain("daily_diet_alternative: \"Search within a saved daily diet or paste its ID…\"");
	expect(source).toContain("placeholder={searchPlaceholders[activeMode]}");
});

// Implements DESIGN-001 SearchView initial and mode-change search focus verification.
test("passes the active mode as the autocomplete focus key", () => {
	expect(source).toContain("let activeMode = $derived($searchStore.mode)");
	expect(source).toContain("focusKey={activeMode}");
});

// Implements DESIGN-016 ThemeProvider top-level dropdown removal verification.
test("does not render the previous system light dark theme dropdown in the main header", () => {
	expect(source).not.toContain("setThemePreference");
	expect(source).not.toContain("<option value=\"system\">System</option>");
	expect(source).not.toContain("<option value=\"light\">Light</option>");
	expect(source).not.toContain("<option value=\"dark\">Dark</option>");
});

// Implements DESIGN-016 LayoutGrid animated sidebar column verification.
test("animates the desktop sidebar grid column between expanded and collapsed widths", () => {
	expect(source).toContain('import { sidebarStore } from "../stores/sidebar"');
	expect(source).toContain("transition-[grid-template-columns]");
	expect(source).toContain("content-start");
	expect(source).toContain("duration-200");
	expect(source).toContain("motion-reduce:transition-none");
	expect(source).toContain("sm:grid-cols-[3.5rem_minmax(0,1fr)]");
	expect(source).toContain("sm:grid-cols-[15rem_minmax(0,1fr)]");
});

// Implements DESIGN-001 SearchView mode-specific controls composition verification.
test("mode-specific controls render conditionally based on searchStore.mode", () => {
	expect(source).toContain('activeMode === "substitution"');
	expect(source).toContain('activeMode === "daily_diet_alternative"');
	expect(source).toContain("executionAllowed={entitlementDecision.canExecute}");
	expect(source).toContain("searchEnabled={entitlementDecision.canExecute}");
});

// Implements DESIGN-001 SearchView Daily Diet rejection wiring verification.
test("DailyDietControls receives the rejection lifted from SearchResults", () => {
	expect(source).toContain("{rejection}");
	expect(source).toContain("onRejection");
});

// Implements DESIGN-001 SearchView shell traceability verification.
test("shell cites the DESIGN source", () => {
	expect(source).toContain("<!-- Implements DESIGN-001 SearchView");
});
