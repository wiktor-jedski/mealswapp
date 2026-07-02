import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-001 SearchView Daily Diet Alternative controls and structured rejection verification.
//
// Static-source assertions verify the daily diet id input is UUID-shaped and bound to
// `setDailyDietId`, the structured `SearchRejection` surface exposes code/message/field
// through a labelled alert region, and no Phase 07 job or worker behavior is introduced.

const source = readFileSync(join(import.meta.dir, "DailyDietControls.svelte"), "utf8");

// Implements DESIGN-001 SearchView generated rejection type reuse verification.
test("imports the generated SearchRejection type without handwritten duplicates", () => {
	expect(source).toContain('import type { SearchRejection } from "../api/generated"');
});

// Implements DESIGN-001 SearchView Daily Diet Alternative id input verification.
test("daily diet id input is UUID-shaped and bound to setDailyDietId", () => {
	expect(source).toContain("setDailyDietId");
	expect(source).toContain("executionAllowed");
	expect(source).toContain('id="daily-diet-id"');
	expect(source).toContain("pattern=");
	expect(source).toContain("[0-9a-fA-F]");
	expect(source).toContain("$searchStore.dailyDietId");
	expect(source).toContain("aria-disabled={!executionAllowed}");
	expect(source).toContain("oninput={onDailyDietIdInput}");
});

// Implements DESIGN-001 SearchView Phase 04 structured rejection display surface verification.
test("structured rejection display surface exposes SearchRejection code, message, and field", () => {
	expect(source).toContain('role="alert"');
	expect(source).toContain('aria-label="Search rejection"');
	expect(source).toContain("rejection?.code");
	expect(source).toContain("rejection?.message");
	expect(source).toContain("rejection?.field");
	expect(source).toContain("data-rejection-message");
	expect(source).toContain("data-rejection-code");
});

// Implements DESIGN-001 SearchView Phase 07 job behavior exclusion verification.
test("does not create Phase 07 job or worker behavior", () => {
	expect(source).not.toContain("createJob");
	expect(source).not.toContain("startWorker");
	expect(source).not.toContain("queueJob");
	expect(source).not.toContain("optimizeDiet");
});

// Implements DESIGN-001 SearchView Daily Diet Alternative landmark and traceability verification.
test("section landmark cites the DESIGN source", () => {
	expect(source).toContain('aria-label="Daily diet alternative controls"');
	expect(source).toContain("<!-- Implements DESIGN-001 SearchView Daily Diet Alternative controls");
});
