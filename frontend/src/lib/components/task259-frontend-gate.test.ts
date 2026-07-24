import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-009 UserAdminPanel representative component coverage for task 259.

const component = (name: string): string => readFileSync(join(import.meta.dir, `${name}.svelte`), "utf8");

test("the administration gate composes every Phase 08 feature surface", () => {
	const panel = component("AdministrationPanel");
	const dataManagement = component("AdminDataManagement");
	const externalImport = component("ExternalImportWorkflow");

	expect(panel).toContain("<ExternalImportWorkflow");
	expect(panel).toContain("<AdminDataManagement");
	expect(dataManagement).toContain("Manual global items");
	expect(dataManagement).toContain("Food Categories and Culinary Roles");
	expect(dataManagement).toContain("Restricted user lookup");
	expect(externalImport).toContain("External food search");
	expect(externalImport).toContain("Retry import safely");
});

test("representative administration components expose accessible state and destructive-action boundaries", () => {
	const panel = component("AdministrationPanel");
	const dataManagement = component("AdminDataManagement");
	const externalImport = component("ExternalImportWorkflow");

	expect(panel).toContain('role="status"');
	expect(panel).toContain('role="alert"');
	expect(dataManagement).toContain('aria-modal="true"');
	expect(dataManagement).toContain("inert={confirmation ? true : undefined}");
	expect(dataManagement).toContain("node.showModal()");
	expect(dataManagement).toContain("confirmationDialog.close()");
	expect(dataManagement).toContain('event.key !== "Tab"');
	expect(dataManagement).toContain("use:focusOnMount");
	expect(externalImport).toContain('aria-live="polite"');
});
