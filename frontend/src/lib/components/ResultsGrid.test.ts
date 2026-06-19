import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-001 ResultsGrid container structure verification.
//
// Static-source assertions verify the grid enforces the 10-item page cap,
// renders skeleton/empty/error/loading states, wires pagination with disabled
// boundaries and an onPageChange callback, retains previous results while
// loading, and matches SimilarityMetadata by itemId with a similarityScores
// fallback. `vite build` compiles the component once Task 151 wires it into
// SearchShell, validating the Svelte source at build time.

const source = readFileSync(join(import.meta.dir, "ResultsGrid.svelte"), "utf8");

// Implements DESIGN-001 ResultsGrid generated-type imports verification.
test("imports ResultCard and generated types without handwritten duplicates", () => {
	expect(source).toContain("import ResultCard from \"./ResultCard.svelte\"");
	expect(source).toContain("FoodObject");
	expect(source).toContain("SimilarityMetadata");
	expect(source).toContain("from \"../api/generated\"");
});

// Implements DESIGN-001 ResultsGrid documented props verification.
test("declares the documented container props", () => {
	expect(source).toContain("export let results");
	expect(source).toContain("export let similarityMetadata");
	expect(source).toContain("export let similarityScores");
	expect(source).toContain("export let loading");
	expect(source).toContain("export let error");
	expect(source).toContain("export let totalCount");
	expect(source).toContain("export let page");
	expect(source).toContain("export let onPageChange");
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
	expect(source).toContain("similarityScores[index] ?? null");
	expect(source).toContain("findSimilarity(item.id)");
});

// Implements DESIGN-001 ResultsGrid loading skeleton state verification.
test("renders loading skeletons when loading with no previous results", () => {
	expect(source).toContain("loading && pagedResults.length === 0");
	expect(source).toContain("data-results-skeletons");
	expect(source).toContain("animate-pulse");
});

// Implements DESIGN-001 ResultsGrid zero-result empty state verification.
test("renders zero-result empty text when not loading and no results", () => {
	expect(source).toContain("No results found.");
	expect(source).toContain("pagedResults.length === 0");
	expect(source).toContain("data-results-empty");
});

// Implements DESIGN-001 ResultsGrid error state verification.
test("renders an error state from the error prop with an alert role", () => {
	expect(source).toContain("{#if error}");
	expect(source).toContain("data-results-error");
	expect(source).toContain('role="alert"');
});

// Implements DESIGN-001 ResultsGrid previous-page retention verification.
test("retains previous results while loading via a polite loading overlay", () => {
	expect(source).toContain("data-results-loading-overlay");
	expect(source).toContain('aria-live="polite"');
});

// Implements DESIGN-001 ResultsGrid pagination page-request wiring verification.
test("Previous and Next buttons call onPageChange with page - 1 and page + 1", () => {
	expect(source).toContain("on:click={() => onPageChange(page - 1)}");
	expect(source).toContain("on:click={() => onPageChange(page + 1)}");
	expect(source).toContain("data-results-prev");
	expect(source).toContain("data-results-next");
	expect(source).toContain("data-results-page");
});

// Implements DESIGN-001 ResultsGrid pagination disabled-boundaries verification.
test("Previous and Next disabled bindings derive from page and totalPages", () => {
	expect(source).toContain("hasPrev = page > 1");
	expect(source).toContain("hasNext = page < totalPages");
	expect(source).toContain("disabled={!hasPrev}");
	expect(source).toContain("disabled={!hasNext}");
	expect(source).toContain("Math.ceil(totalCount / PAGE_SIZE)");
});

// Implements DESIGN-001 ResultsGrid container traceability verification.
test("cites the DESIGN-001 ResultsGrid source", () => {
	expect(source).toContain("<!-- Implements DESIGN-001 ResultsGrid -->");
});
