import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-009 UserAdminPanel task-268 visual and accessibility source verification.

const components = Object.fromEntries(
	["AdministrationPanel", "AdminDataManagement", "AdminPrivateData", "ExternalImportWorkflow"].map((name) => [
		name,
		readFileSync(join(import.meta.dir, `${name}.svelte`), "utf8")
	])
);

function openings(source: string, element: string): string[] {
	return [...source.matchAll(new RegExp(`<(?:${element})\\b[^>]*>`, "gs"))].map(([opening]) => opening);
}

function expectClasses(opening: string, required: string[]): void {
	const classes = opening.match(/\bclass="([^"]*)"/s)?.[1] ?? "";
	for (const requiredClass of required) expect(classes.split(/\s+/)).toContain(requiredClass);
}

function themeToken(css: string, selector: string, token: string): string {
	const block = css.match(new RegExp(`${selector.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\s*\\{([^}]*)\\}`, "s"))?.[1] ?? "";
	return block.match(new RegExp(`${token.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}:\\s*(#[0-9a-f]{6})`, "i"))?.[1] ?? "";
}

function contrastRatio(foreground: string, background: string): number {
	const luminance = (color: string): number => {
		const channels = color.slice(1).match(/../g)?.map((channel) => Number.parseInt(channel, 16) / 255) ?? [];
		const linear = channels.map((channel) => channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4);
		return 0.2126 * linear[0]! + 0.7152 * linear[1]! + 0.0722 * linear[2]!;
	};
	const [lighter, darker] = [luminance(foreground), luminance(background)].sort((a, b) => b - a);
	return (lighter! + 0.05) / (darker! + 0.05);
}

test("every Phase 08 administration button uses the standard reduced-motion transition", () => {
	for (const name of ["AdminDataManagement", "AdminPrivateData", "ExternalImportWorkflow"]) {
		const buttons = openings(components[name]!, "button");
		expect(buttons.length).toBeGreaterThan(0);
		for (const button of buttons) expectClasses(button, ["transition-all", "duration-200", "motion-reduce:transition-none"]);
	}
});

test("every affected administration form control uses Surface, Border, and a two-pixel Primary focus ring", () => {
	for (const name of ["AdminDataManagement", "ExternalImportWorkflow"]) {
		for (const control of openings(components[name]!, "input|select|textarea")) {
			expectClasses(control, ["border", "border-[var(--color-border)]", "bg-[var(--color-surface)]", "focus:ring-2", "focus:ring-[var(--color-primary)]"]);
		}
	}
});

test("administration headings alone use Bold 700 without mechanically changing other emphasized text", () => {
	for (const source of Object.values(components)) {
		for (const heading of openings(source, "h[1-6]")) {
			expectClasses(heading, ["font-bold"]);
			expect(heading).not.toContain("font-semibold");
		}
	}
	expect(components.ExternalImportWorkflow).toContain('<legend class="font-semibold">Food categories</legend>');
	expect(components.ExternalImportWorkflow).toContain('<p class="font-semibold">Partial results</p>');
});

test("filled destructive buttons use the dedicated accessible error foreground", () => {
	for (const name of ["AdminDataManagement", "AdminPrivateData"]) {
		const destructiveButtons = openings(components[name]!, "button").filter((button) => button.includes("bg-[var(--color-error)]"));
		expect(destructiveButtons.length).toBeGreaterThan(0);
		for (const button of destructiveButtons) {
			expectClasses(button, ["text-[var(--color-on-error)]"]);
			expect(button).not.toContain("text-[var(--color-on-muted)]");
		}
	}
});

test("error foreground contrast is WCAG AA in light and dark themes", () => {
	const css = readFileSync(join(import.meta.dir, "../../app.css"), "utf8");
	for (const selector of [":root", ':root[data-theme="dark"]']) {
		const ratio = contrastRatio(themeToken(css, selector, "--color-on-error"), themeToken(css, selector, "--color-error"));
		expect(ratio).toBeGreaterThanOrEqual(4.5);
	}
});
