import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

import type { DisclaimerData } from "../api/generated";
import {
	BUNDLED_LOGIN_DISCLAIMER,
	loadLoginDisclaimerViewModel
} from "./disclaimer-panel";

// Implements DESIGN-018 DisclaimerPanel unit verification for generated content and fallback behavior.

const source = readFileSync(join(import.meta.dir, "DisclaimerPanel.svelte"), "utf8");

const generatedDisclaimer: DisclaimerData = {
	location: "login",
	version: "2026-07",
	markdown: "Generated medical disclaimer.",
	effectiveAt: "2026-07-05T00:00:00.000Z"
};

test("loads generated login disclaimer content", async () => {
	const viewModel = await loadLoginDisclaimerViewModel({
		loadDisclaimer: async (location) => {
			expect(location).toBe("login");
			return generatedDisclaimer;
		}
	});

	expect(viewModel).toEqual({
		version: "2026-07",
		bodyMarkdown: "Generated medical disclaimer.",
		effectiveAt: "2026-07-05T00:00:00.000Z",
		unavailable: false
	});
});

test("renders bundled fallback when the disclaimer API is unavailable", async () => {
	const viewModel = await loadLoginDisclaimerViewModel({
		loadDisclaimer: async () => {
			throw new Error("network unavailable");
		}
	});

	expect(viewModel).toEqual(BUNDLED_LOGIN_DISCLAIMER);
	expect(viewModel.bodyMarkdown).toContain("not medical advice");
	expect(viewModel.unavailable).toBe(true);
});

test("component declares mandatory accessible disclaimer content and fallback status", () => {
	expect(source).toContain("<!-- Implements DESIGN-018 DisclaimerPanel");
	expect(source).toContain("data-auth-disclaimer");
	expect(source).toContain("Medical disclaimer");
	expect(source).toContain("data-disclaimer-fallback");
	expect(source).toContain('role="status"');
	expect(source).toContain('aria-live="polite"');
});
