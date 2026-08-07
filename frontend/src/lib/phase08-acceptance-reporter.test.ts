// Implements DESIGN-014 MetricsCollector acceptance project-scope verification.

import { expect, test } from "bun:test";

import { resolveRequiredProjects } from "../../tests/phase08-acceptance-reporter";

test("accepts the manifest-authorized mobile scope for Task 303 criteria", () => {
	const expected = new Set(["real-stack-desktop-chromium", "real-stack-mobile-chromium"]);

	expect([...resolveRequiredProjects("P08-SWR056-STEP-01", ["real-stack-mobile-chromium"], expected)]).toEqual([
		"real-stack-mobile-chromium"
	]);
});

test("rejects an unallowlisted criterion's attempt to narrow project coverage", () => {
	const expected = new Set(["real-stack-desktop-chromium", "real-stack-mobile-chromium"]);

	expect([...resolveRequiredProjects("P08-SWR054-STEP-01", ["real-stack-mobile-chromium"], expected)]).toEqual([
		"real-stack-desktop-chromium",
		"real-stack-mobile-chromium"
	]);
});
