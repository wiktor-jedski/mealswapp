import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-008 DataExporter/ProfileController Admin Panel component-boundary verification.

const source = readFileSync(join(import.meta.dir, "AdminPrivateData.svelte"), "utf8");

test("uses the generated account-data client for explicit export-backed deletion", () => {
	expect(source).toContain('accountDataApi, type AccountDataApi } from "../api/account-data-client"');
	expect(source).toContain("await api.loadExport");
	expect(source).toContain("await api.deleteCustomItem");
	expect(source).toContain("Permanently delete private item");
	expect(source).toContain("permanently and irreversibly removes");
	expect(source).toContain("There is no Trash or Restore workflow");
	expect(source).toContain("authoritative export refreshed");
	expect(source).not.toContain("fetch(");
});

test("fails closed before every authoritative refresh and distinguishes deletion verification failure", () => {
	expect(source).toContain("const current = beginOperation()");
	expect(source).toContain("pendingDelete = undefined");
	expect(source).toContain("items = []");
	expect(source).toContain('message = ""');
	expect(source).toContain("if (!isCurrent(current)) return");
	expect(source).toContain("The private item was deleted, but current account data could not be verified.");
	expect(source.indexOf("const current = beginOperation()")).toBeLessThan(source.indexOf("await api.loadExport(current.controller.signal)"));
	expect(source).toContain("affectedDiets");
	expect(source).toContain("Open Daily Diets, remove this item from each listed diet, save the diet, then retry permanent deletion.");
	expect(source).toContain('aria-label="Saved diets blocking permanent deletion"');
});
