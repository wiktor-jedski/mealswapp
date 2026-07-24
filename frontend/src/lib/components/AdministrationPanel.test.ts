import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-009 UserAdminPanel component-boundary and responsive-shell verification.

const source = readFileSync(join(import.meta.dir, "AdministrationPanel.svelte"), "utf8");

test("declares feature-local loading and error boundaries", () => {
	expect(source).toContain('access === "loading"');
	expect(source).toContain('role="status"');
	expect(source).toContain("data-admin-loading");
	expect(source).toContain('access === "error"');
	expect(source).toContain('role="alert"');
	expect(source).toContain("data-admin-error");
});

test("renders only shell-level responsive administration regions", () => {
	expect(source).toContain("Administration Panel");
	expect(source).toContain("data-admin-responsive-grid");
	expect(source).toContain("sm:grid-cols-2");
	expect(source).toContain("lg:grid-cols-3");
	expect(source).not.toContain("fetch(");
	expect(source).not.toContain("/api/v1/admin/");
});

test("states that client visibility is not backend authorization", () => {
	expect(source).toContain("data-admin-server-auth-notice");
	expect(source).toContain("The server authorizes every administration request.");
	expect(source).toContain("Implements DESIGN-009 UserAdminPanel");
});

// Implements DESIGN-009 ExternalSearchProxy task 255 composition verification.
test("composes the external import workflow only inside the allowed administration branch", () => {
	expect(source).toContain('import ExternalImportWorkflow from "./ExternalImportWorkflow.svelte"');
	expect(source).toContain("<ExternalImportWorkflow {onViewLocalItem} />");
	expect(source.indexOf("<ExternalImportWorkflow")).toBeGreaterThan(source.indexOf("{:else}"));
});

test("composes generated-client private-data controls inside the allowed administration branch", () => {
	expect(source).toContain('import AdminPrivateData from "./AdminPrivateData.svelte"');
	expect(source).toContain("<AdminPrivateData />");
	expect(source.indexOf("<AdminPrivateData")).toBeGreaterThan(source.indexOf("{:else}"));
});
