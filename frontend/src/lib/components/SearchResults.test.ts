import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-001 SearchView committed search request source verification.
//
// SearchResults compiles under `vite build`; these static assertions lock the request
// orchestration to SW-REQ-002 so typing updates autocomplete but not server-side results.

const source = readFileSync(join(import.meta.dir, "SearchResults.svelte"), "utf8");

// Implements DESIGN-001 SearchView committed server-side query execution verification.
test("uses the committed query for server-side search execution", () => {
	expect(source).toContain("const committedSearchStore = derived(searchStore");
	expect(source).toContain("query: storeState.submittedQuery");
	expect(source).toContain("createSearchQueryOptions(committedSearchStore, localCache)");
	expect(source).not.toContain("setTimeout(() =>");
});

// Implements DESIGN-001 SearchView committed offline cache-key verification.
test("uses the committed query for offline cache indicator lookup", () => {
	expect(source).toContain("searchRequestKey({ ...state, query: state.submittedQuery })");
	expect(source).toContain("setShowingCached");
	expect(source).toContain("setShowingStale");
	expect(source).toContain("const isStaleCachedResult = localCache.isStale(key, LOCAL_CACHE_STALE_MS)");
});

// Implements DESIGN-001 SearchView immediate non-query state update verification.
test("keeps immediate search state available for visible pagination state", () => {
	expect(source).toContain("let state = $derived($searchStore)");
	expect(source).not.toContain("searchStore.subscribe");
});

// Implements DESIGN-001 SearchView initial empty-results suppression verification.
test("hides the ResultsGrid until the user enters a non-empty query", () => {
	expect(source).toContain("hasStartedSearching");
	expect(source).toContain("currentOptions.enabled === true");
	expect(source).toContain("{#if hasStartedSearching}");
	expect(source).toContain("<ResultsGrid");
});

// Implements DESIGN-001 SearchView submitted-search loading signal verification.
test("lifts submitted search loading state and suppresses empty result flicker while fetching", () => {
	expect(source).toContain("onSearchInFlightChange");
	expect(source).toContain("hasStartedSearching && query.isFetching === true");
	expect(source).toContain("onSearchInFlightChange(searchInFlight)");
	expect(source).toContain("loading={searchInFlight}");
});

// Implements DESIGN-001 SearchView mode-specific similarity display verification.
test("suppresses similarity display for Catalog results", () => {
	expect(source).toContain('showSimilarity={state.mode !== "catalog"}');
});

// Implements DESIGN-001 SearchView Catalog-to-Substitution action verification.
test("adds full Catalog result items to the Substitution Input list", () => {
	expect(source).toContain("addCatalogResultToSubstitutions");
	expect(source).toContain("addSubstitutionInput");
	expect(source).toContain("displayUnitForBasis(item.macroBasis, $preferencesStore.unitSystem)");
	expect(source).toContain('onAddToSubstitution={state.mode === "catalog" ? addCatalogResultToSubstitutions : null}');
});

// Implements DESIGN-001 ResultsGrid substitution source summary propagation verification.
test("passes backend sourceSummary into the results grid", () => {
	expect(source).toContain("sourceSummary={query.data?.sourceSummary ?? null}");
});
