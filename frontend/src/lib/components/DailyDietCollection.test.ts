import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";

const source = readFileSync(new URL("./DailyDietCollection.svelte", import.meta.url), "utf8");

// Implements DESIGN-001 SearchView Daily Diet default quantity wiring verification.
test("new draft meals use the basis-aware display quantity and unit", () => {
	expect(source).toContain("defaultDisplayQuantity(selection.item.macroBasis, $preferencesStore.unitSystem)");
	expect(source).toContain("displayUnitForBasis(selection.item.macroBasis, $preferencesStore.unitSystem)");
	expect(source).not.toMatch(/quantity:\s*100,\s*\n\s*unit: displayUnitForBasis/);
});

// Implements DESIGN-001 SearchView identity-scoped Daily Diet draft verification.
test("identity changes clear every component-local Daily Diet draft field", () => {
	expect(source).toContain("function resetIdentityOwnedDraft(): void");
	expect(source).toContain('draftName = "My Daily Diet"');
	expect(source).toContain("draftMeals = []");
	expect(source).toContain("consumedSelectionKeys = new Set(selections.map((selection) => selection.key))");
	expect(source).toContain("draftError = null");
	expect(source).toContain("serverAggregate = null");
	expect(source).toContain("savedDietId = null");
	expect(source.match(/resetIdentityOwnedDraft\(\)/g)?.length ?? 0).toBeGreaterThanOrEqual(3);
});

// Implements DESIGN-001 SearchView create-intent edit and pending-click verification.
test("draft edits clear retry ownership and the save button suppresses pending clicks", () => {
	expect(source).toContain("clearDailyDietCreateIntent");
	expect(source).toContain("function updateQuantity");
	expect(source).toContain("function updateUnit");
	expect(source).toContain("function moveMeal");
	expect(source).toContain("function removeMeal");
	expect(source).toContain("function resetDraft");
	expect(source.match(/clearDailyDietCreateIntent\(\)/g)?.length ?? 0).toBeGreaterThanOrEqual(7);
	expect(source).toContain('$dailyDietStore.mutation === "creating"');
});

test("a confirmed collection remains an explicit replace target while edited macros stay non-authoritative", () => {
	expect(source).toContain("let editingDietId = $state<string | null>(null)");
	expect(source).toContain("? await replaceDailyDiet(editingDietId, request)");
	expect(source).toContain("editingDietId = saved.id");
	expect(source).toContain('editingDietId ? "Update Daily Diet" : "Save Daily Diet"');
	expect(source).toContain('disabled={!canEdit || draftMeals.length < 2 || $dailyDietStore.mutation !== "idle"}');
});
