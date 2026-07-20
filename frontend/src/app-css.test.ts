import { expect, test } from "bun:test";

// Implements DESIGN-016 ComponentStyles reduced-motion theme transition verification.
test("reduced motion disables global color transitions during theme changes", async () => {
	const css = await Bun.file(new URL("./app.css", import.meta.url)).text();

	expect(css).toContain("@media (prefers-reduced-motion: reduce)");
	expect(css).toContain("*,\n  *::before,\n  *::after");
	expect(css).toContain("transition: none !important");
	expect(css).toContain("animation: none !important");
});
