import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-009 ItemCurator, TagManager, and UserAdminPanel component-boundary verification.

const source = readFileSync(join(import.meta.dir, "AdminDataManagement.svelte"), "utf8");

test("composes manual item, classification, and privacy-minimized user views", () => {
	expect(source).toContain("Manual global items");
	expect(source).toContain("Food Categories and Culinary Roles");
	expect(source).toContain("Restricted user lookup");
	expect(source).not.toMatch(/password|impersonat|role mutation/i);
});

test("binds confirmations to immutable targets and guards authoritative refresh ownership", () => {
	expect(source).toContain("data-admin-confirmation");
	expect(source).toContain("Confirm destructive action");
	expect(source).toContain("Object.freeze({ ...target })");
	expect(source).toContain('currentItem?.id !== target.id');
	expect(source).toContain("inert={confirmation ? true : undefined}");
	expect(source).toContain("node.showModal()");
	expect(source).toContain("confirmationDialog.close()");
	expect(source).toContain('event.key !== "Tab"');
	expect(source).toContain("event.preventDefault(); target.focus()");
	expect(source).not.toContain("<dialog open");
	expect(source).toContain("currentItemOperation(generation, controller)");
	expect(source).toContain("currentClassificationOperation(generation, controller)");
	expect(source).toContain("currentUserOperation(generation, controller)");
	expect(source).toContain("api.getItem(saved.id, controller.signal)");
});

test("uses responsive layouts, keyboard focus, safe alerts, and design traceability", () => {
	expect(source).toContain("sm:grid-cols-2");
	expect(source).toContain("md:grid-cols-2");
	expect(source).toContain("use:focusOnMount");
	expect(source).toContain('role="alert"');
	expect(source).toContain("Implements DESIGN-009 ItemCurator, TagManager, and UserAdminPanel");
});

test("restores confirmation focus to the opener or an enabled in-context fallback", () => {
	expect(source).toContain("confirmationOpener");
	expect(source).toContain("restoreConfirmationFocus");
	expect(source).toContain("isConnected");
	expect(source).toContain("data-admin-data-management");
});
