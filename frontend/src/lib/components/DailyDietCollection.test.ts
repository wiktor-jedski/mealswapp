import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";

const source = readFileSync(new URL("./DailyDietCollection.svelte", import.meta.url), "utf8");

// Implements DESIGN-001 SearchView theme-consistent Daily Diet input verification.
test("collection name and item controls use the shared dark-mode treatment", () => {
	expect(source).toContain('id="daily-diet-name"');
	expect(source).toContain("border-[var(--color-border)] bg-[var(--color-surface)]");
	expect(source).toContain("bg-transparent px-2 text-sm");
	expect(source).not.toContain("bg-white");
});

// Implements DESIGN-001 AutocompleteDropdown Daily Diet item composition verification.
test("the editor owns a first-highlighted food and meal autocomplete", () => {
	expect(source).toContain("<AutocompleteDropdown");
	expect(source).toContain('placeholder="Search foods or meals to add…"');
	expect(source).toContain("selectFirstOnEnter={true}");
	expect(source).toContain("focusOnMount={false}");
	expect(source).toContain("addFoodObject(item)");
	expect(source).toContain("fetchFoodObject(suggestion.itemId, controller.signal, suggestion.objectType)");
	expect(source).toContain("defaultDisplayQuantity(item.macroBasis, $preferencesStore.unitSystem)");
});

// Implements DESIGN-001 SearchView saved Daily Diet edit hydration verification.
test("a selected saved diet hydrates ordered entries and preserves quantities for replacement", () => {
	expect(source).toContain("selectedDiet?: DailyDietEditSelection | null");
	expect(source).toContain("openForEditing(selectedDiet.diet)");
	expect(source).toContain("sort((left, right) => left.position - right.position)");
	expect(source).toContain("fetchFoodObject(entry.foodObjectId, controller.signal, entry.foodObjectType)");
	expect(source).toContain("quantity: entry.quantity");
	expect(source).toContain("unit: entry.unit");
	expect(source).toContain("? await replaceDailyDiet(editingDietId, request)");
});

// Implements DESIGN-008 SavedDataRepository unique-name and deletion workflow verification.
test("create and edit enforce unique names and edit mode exposes guarded removal", () => {
	expect(source).toContain("diet.name.trim().toLocaleLowerCase() === name.toLocaleLowerCase()");
	expect(source).toContain("A Daily Diet with this name already exists.");
	expect(source).toContain("deleteDailyDiet(editingDietId)");
	expect(source).toContain("window.confirm");
	expect(source).toContain('"Removing…" : "Remove"');
	expect(source).toContain("Clear draft");
	expect(source).toContain("ml-auto");
});

// Implements DESIGN-001 SearchView explicit transition from editing to a new Daily Diet draft.
test("edit mode exposes a right-aligned New action that resets the draft", () => {
	expect(source).toContain('class="flex items-start justify-between gap-3"');
	expect(source).toContain("data-daily-diet-new");
	expect(source).toContain("onclick={resetDraft}");
	expect(source).toContain(">New</button>");
	expect(source).toContain('editingDietId ? "Update" : "Save"');
});

// Implements DESIGN-001 SearchView saved-list disclosure and heading verification.
test("saved Daily Diets use a compact heading and a show-hide disclosure", () => {
	expect(source).toContain('class="text-base font-semibold text-[var(--color-text)]"');
	expect(source).toContain("aria-expanded={savedListOpen}");
	expect(source).toContain('savedListOpen ? "Hide" : "Show"');
	expect(source).toContain("onEditDiet(diet)");
});

// Implements DESIGN-001 SearchView Substitution-style Food Object card verification.
test("item cards show image, nutrition basis, per-basis macros, and retained ordering controls", () => {
	expect(source).toContain("data-daily-diet-image-wrapper");
	expect(source).toContain("data-daily-diet-item-macros");
	expect(source).toContain("macroBasisDisplayLabel(draft.item.macroBasis, $preferencesStore.unitSystem)");
	expect(source).toContain("draft.item.macros.protein");
	expect(source).toContain("formatCalories(draft.item.calories)");
	expect(source).toContain("moveFoodObject(draft.key, -1)");
	expect(source).toContain("moveFoodObject(draft.key, 1)");
	expect(source).toContain("removeFoodObject(draft.key)");
});

// Implements DESIGN-001 SearchView whole-kilocalorie display verification.
test("rounds item, aggregate, and saved-diet calories through one display formatter", () => {
	expect(source).toContain("formatCalories(draft.item.calories)");
	expect(source).toContain("formatCalories(aggregate.calories)");
	expect(source).toContain("formatCalories(diet.aggregateMacros.calories)");
});

// Implements DESIGN-001 SearchView shared destructive-action color verification.
test("remove and delete actions use the Substitution remove-button accent colors", () => {
	expect(source.match(/border-\[var\(--color-accent\)\] bg-\[var\(--color-accent\)\]/g)).toHaveLength(2);
	expect(source.match(/bg-\[var\(--color-accent\)\]/g)).toHaveLength(2);
	expect(source.match(/text-\[var\(--color-on-accent\)\]/g)).toHaveLength(2);
	expect(source).not.toContain('class="rounded border border-[var(--color-error)] px-2 py-2');
	expect(source).not.toContain('class="ml-auto rounded border border-[var(--color-error)] px-3 py-2');
});

// Implements DESIGN-001 SearchView identity and stale hydration isolation verification.
test("identity changes and teardown cancel outstanding editor hydration", () => {
	expect(source).toContain("function resetIdentityOwnedDraft(): void");
	expect(source).toContain("cancelHydration()");
	expect(source).toContain("controller.abort()");
	expect(source).toContain("generation !== hydrationGeneration");
	expect(source).toContain("onDestroy(cancelHydration)");
});
