import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-008 DataExporter/ProfileController Admin Panel component-boundary verification.

const source = readFileSync(join(import.meta.dir, "AdminPrivateData.svelte"), "utf8");

test("uses the generated account-data client for explicit export-backed deletion", () => {
	expect(source).toContain('import { accountDataApi, type AccountDataApi } from "../api/account-data-client"');
	expect(source).toContain("await api.loadExport");
	expect(source).toContain("await api.deleteCustomItem");
	expect(source).toContain("Confirm private item deletion");
	expect(source).toContain("authoritative export refreshed");
	expect(source).not.toContain("fetch(");
});
