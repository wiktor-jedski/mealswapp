import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-009 ExternalSearchProxy, ItemCurator, and DataImporter component-boundary verification.

const source = readFileSync(join(import.meta.dir, "ExternalImportWorkflow.svelte"), "utf8");

test("covers provider selection, pagination, and all safe external states", () => {
	expect(source).toContain('value="usda"');
	expect(source).toContain('value="openfoodfacts"');
	expect(source).toContain('value="all"');
	expect(source).toContain("requestSearch(page - 1)");
	expect(source).toContain("requestSearch(page + 1)");
	for (const state of ["loading", "empty", "error"]) expect(source).toContain(`searchState === "${state}"`);
	expect(source).toContain('"empty" : "results"');
	expect(source).toContain("providerWarningLabels");
	expect(source).not.toContain("warning.message");
	expect(source).toContain("searchController?.abort()");
	expect(source).toContain("sequence !== searchSequence");
	expect(source).toContain("controller.signal");
});

test("provides editable drafts, normalization warnings, density, and classification selection", () => {
	expect(source).toContain("bind:value={draft.name}");
	expect(source).toContain("bind:value={draft.macrosPer100.protein}");
	expect(source).toContain("missing_liquid_density");
	expect(source).toContain("uncertain_unit_conversion");
	expect(source).toContain("suspicious_liquid_macros");
	expect(source).toContain("toggleClassification");
	expect(source).toContain("foodCategoryIds");
	expect(source).toContain("culinaryRoleIds");
	expect(source).toContain("densitySourceKind");
	expect(source).toContain("updateDensity");
	expect(source).toContain("updatePhysicalState");
	expect(source).toContain("Density provenance");
});

test("keeps one idempotency key through conflict and ambiguous retry paths", () => {
	expect(source.match(/createImportIdempotencyKey\(\)/g)?.length).toBe(2);
	expect(source).toContain("importCuratedItem(draftSnapshot, keySnapshot)");
	expect(source).toContain("const draftSnapshot = snapshotDraft(draft, confirmNameConflict)");
	expect(source).toContain("const keySnapshot = importKey");
	expect(source).toContain('error.appError.code === "name_conflict_confirmation_required"');
	expect(source).toContain('importState = "blockedConflict"');
	expect(source).toContain('importState = "ambiguous"');
	expect(source).toContain("Confirm merge");
	expect(source).toContain("Start a fresh import attempt");
	expect(source).toContain("Retry import safely");
	expect(source).toContain("submitImport(importConfirmNameConflict)");
});

test("owns deferred imports and explicit draft-to-search transitions", () => {
	expect(source).toContain("const ownershipToken = ++importOwnershipToken");
	expect(source.match(/ownershipToken !== importOwnershipToken/g)?.length).toBe(2);
	expect(source).toContain("function invalidateImportOwnership()");
	expect(source).toContain("invalidateImportOwnership();");
	expect(source).toContain("resetCompletedDraft();");
	expect(source).toContain("resetCuration();");
	expect(source).toContain("pendingSearch = request");
	expect(source).toContain("Keep editing");
	expect(source).toContain("Discard draft and search");
	expect(source).toContain('role="alertdialog"');
	expect(source).toContain('aria-describedby="draft-search-description"');
	expect(source).toContain("handleDraftBoundaryKeydown");
	expect(source).toContain("use:openDraftBoundary");
	expect(source).toContain("node.showModal()");
	expect(source).toContain("document.addEventListener(\"focusin\", recoverFocus)");
	expect(source).toContain("inert={pendingSearch ? true : undefined}");
	expect(source).toContain('disabled={importState === "importing"}');
	expect(source).toContain('disabled={searchState === "loading" || importState === "importing"}');
	expect(source).toContain('<fieldset class="grid gap-4" disabled={importState === "importing"}>');
});

test("offers a keyboard-native local-search handoff after import", () => {
	expect(source).toContain("onViewLocalItem(importResult!.name)");
	expect(source).toContain("View in local search");
	expect(source).toContain('type="submit"');
	expect(source).toContain("focus:ring-2");
});
