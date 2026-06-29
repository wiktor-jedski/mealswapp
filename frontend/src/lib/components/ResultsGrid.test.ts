import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-001 ResultsGrid container structure verification.
//
// Static-source assertions verify the grid enforces the 10-item page cap,
// renders empty/error states, wires pagination with disabled
// boundaries and an onPageChange callback, retains previous results while
// loading, and conditionally matches SimilarityMetadata by itemId with a similarityScores
// fallback. `vite build` compiles the component once Task 151 wires it into
// SearchShell, validating the Svelte source at build time.

const source = readFileSync(join(import.meta.dir, "ResultsGrid.svelte"), "utf8");

// Implements DESIGN-001 ResultsGrid generated-type imports verification.
test("imports ResultCard and generated types without handwritten duplicates", () => {
	expect(source).toContain("import ResultCard from \"./ResultCard.svelte\"");
	expect(source).toContain("import SourceSummaryCard from \"./SourceSummaryCard.svelte\"");
	expect(source).toContain("FoodObject");
	expect(source).toContain("SimilarityMetadata");
	expect(source).toContain("SourceSummary");
	expect(source).toContain("from \"../api/generated\"");
});

// Implements DESIGN-001 ResultsGrid documented props verification.
test("declares the documented container props", () => {
	expect(source).toContain("} = $props()");
	expect(source).toContain("results?: FoodObject[]");
	expect(source).toContain("similarityMetadata?: SimilarityMetadata[]");
	expect(source).toContain("similarityScores?: number[]");
	expect(source).toContain("showSimilarity?: boolean");
	expect(source).toContain("sourceSummary?: SourceSummary | null");
	expect(source).toContain("onAddToSubstitution?: ((item: FoodObject) => void) | null");
	expect(source).toContain("error?: string | null");
	expect(source).toContain("totalCount?: number");
	expect(source).toContain("page?: number");
	expect(source).toContain("onPageChange?: (page: number) => void");
});

// Implements DESIGN-001 ResultsGrid maximum-10-item page cap verification.
test("enforces the 10-item page cap via a PAGE_SIZE slice", () => {
	expect(source).toContain("PAGE_SIZE = 10");
	expect(source).toContain(".slice(0, PAGE_SIZE)");
	expect(source).toContain("{#each pagedResults as item, index (item.id)}");
});

// Implements DESIGN-001 ResultsGrid similarity matching by itemId verification.
test("matches SimilarityMetadata by itemId and falls back to similarityScores by index", () => {
	expect(source).toContain("similarityByItemId");
	expect(source).toContain("meta.itemId");
	expect(source).toContain("showSimilarity ? similarityScores[index] ?? null : null");
	expect(source).toContain("findSimilarity(item.id)");
});

// Implements DESIGN-001 ResultsGrid mode-specific similarity display verification.
test("can suppress similarity rendering for modes where match percentage is not meaningful", () => {
	expect(source).toContain("showSimilarity ? findSimilarity(item.id) : null");
	expect(source).toContain("showSimilarity ? similarityScores[index] ?? null : null");
});

// Implements DESIGN-001 ResultsGrid substitution source summary verification.
test("renders the source summary card before substitution results only when similarity is shown", () => {
	expect(source).toContain("{#if showSimilarity && sourceSummary}");
	expect(source).toContain("<SourceSummaryCard {sourceSummary} />");
	const summaryPos = source.indexOf("<SourceSummaryCard {sourceSummary} />");
	const resultsPos = source.indexOf("{#each pagedResults as item");
	expect(summaryPos).toBeGreaterThan(-1);
	expect(resultsPos).toBeGreaterThan(summaryPos);
});

// Implements DESIGN-001 SearchView Catalog-to-Substitution action propagation verification.
test("passes the optional add-to-substitutions action to each result card", () => {
	expect(source).toContain("{onAddToSubstitution}");
});

// Implements DESIGN-001 ResultsGrid artifact-free loading verification.
test("does not render skeleton result rows during loading", () => {
	expect(source).not.toContain("data-results-skeletons");
	expect(source).not.toContain("data-result-skeleton");
	expect(source).not.toContain("animate-pulse");
	expect(source).not.toContain("SKELETON_COUNT");
});

// Implements DESIGN-001 ResultsGrid zero-result empty state verification.
test("renders zero-result empty text when not loading and no results", () => {
	expect(source).toContain("No results found.");
	expect(source).toContain("pagedResults.length === 0 && !loading");
	expect(source).toContain("data-results-empty");
});

// Implements DESIGN-001 ResultsGrid error state verification.
test("renders an error state from the error prop with an alert role", () => {
	expect(source).toContain("{#if error}");
	expect(source).toContain("data-results-error");
	expect(source).toContain('role="alert"');
});

// Implements DESIGN-001 ResultsGrid previous-page retention verification.
test("retains previous results while loading without rendering flickering loading text", () => {
	expect(source).not.toContain("data-results-loading-overlay");
	expect(source).not.toContain("Loading…");
	expect(source).toContain("{:else}");
	expect(source).toContain("data-results-list");
});

// Implements DESIGN-001 ResultsGrid pagination page-request wiring verification.
test("Previous and Next buttons call onPageChange with page - 1 and page + 1", () => {
	expect(source).toContain("onclick={() => onPageChange(page - 1)}");
	expect(source).toContain("onclick={() => onPageChange(page + 1)}");
	expect(source).toContain("data-results-prev");
	expect(source).toContain("data-results-next");
	expect(source).toContain("data-results-page");
});

// Implements DESIGN-001 ResultsGrid pagination disabled-boundaries verification.
test("Previous and Next disabled bindings derive from page and totalPages", () => {
	expect(source).toContain("hasPrev = $derived(page > 1)");
	expect(source).toContain("hasNext = $derived(page < totalPages)");
	expect(source).toContain("disabled={!hasPrev}");
	expect(source).toContain("disabled={!hasNext}");
	expect(source).toContain("Math.ceil(totalCount / PAGE_SIZE)");
});

// Implements DESIGN-001 ResultsGrid container traceability verification.
test("cites the DESIGN-001 ResultsGrid source", () => {
	expect(source).toContain("<!-- Implements DESIGN-001 ResultsGrid -->");
});
