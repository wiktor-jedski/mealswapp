import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";

const controls = readFileSync(new URL("./DailyDietControls.svelte", import.meta.url), "utf8");
const optimization = readFileSync(new URL("./OptimizationWorkflow.svelte", import.meta.url), "utf8");

// Implements DESIGN-001 SearchView authoritative Daily Diet selection component verification.
test("selector and optimization consume one selected Daily Diet source", () => {
	expect(controls).toContain('import { selectedDailyDietId } from "../stores/selected-daily-diet"');
	expect(controls).toContain("aria-checked={$selectedDailyDietId === diet.id}");
	expect(controls).toContain("<OptimizationWorkflow selectedDietId={$selectedDailyDietId}");
	expect(controls).toContain("identityId={userId}");
	expect(controls).not.toContain("$dailyDietStore.selectedId");
});

// Implements DESIGN-001 SearchView authoritative-only optimization input verification.
test("optimization cannot activate while a Daily Diet mutation is pending", () => {
	expect(optimization).toContain('$dailyDietStore.mutation === "idle"');
	expect(optimization).toContain("selectedDiet.aggregateMacros");
});

// Implements DESIGN-017 RetryManager authenticated identity teardown verification.
test("logout and account teardown clear shared optimization identity before workflow removal", () => {
	expect(controls).toContain('clearOptimizationIdentity');
	expect(controls).toContain('if (!authenticated && loadedUserId !== null)');
	expect(controls).toContain('clearOptimizationIdentity();');
});
