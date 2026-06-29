import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-001 OfflineBanner accessible status and message verification.
//
// Bun's isolated install-cache layout breaks transitive resolution for
// `svelte/server`/`svelte/compiler`, and no DOM library (jsdom/happy-dom) is
// installed, so the component cannot be rendered in a Bun unit test. Instead
// these tests assert the component declares the documented offline/stale
// messages, an accessible live region, and traceability. `vite build` compiles
// the component, validating the Svelte source.
//
// NOTE: These tests do NOT claim Phase 09 service-worker API/image interception
// coverage, which remains Phase 09 scope per docs/implementation/04_OPEN.md.

const source = readFileSync(
	join(import.meta.dir, "OfflineBanner.svelte"),
	"utf8"
);

// Implements DESIGN-001 OfflineBanner offline cached-result message verification.
test("declares the offline cached-results message", () => {
	expect(source).toContain("You're offline. Showing cached results.");
});

// Implements DESIGN-001 OfflineBanner stale-result message verification.
test("declares the offline stale-results message", () => {
	expect(source).toContain("You're offline. Results may be stale.");
});

// Implements DESIGN-001 OfflineBanner uncached actionable feedback verification.
test("declares the uncached offline actionable feedback message", () => {
	expect(source).toContain("You're offline. Search is unavailable until you reconnect.");
});

// Implements DESIGN-001 OfflineBanner accessible live region verification.
test("declares an accessible status live region", () => {
	expect(source).toContain('role="status"');
	expect(source).toContain('aria-live="polite"');
});

// Implements DESIGN-001 OfflineBanner traceability verification.
test("cites the DESIGN-001 OfflineBanner source", () => {
	expect(source).toContain("<!-- Implements DESIGN-001 OfflineBanner -->");
});

// Implements DESIGN-001 OfflineBanner offline store subscription verification.
test("subscribes to the offlineStatus store", () => {
	expect(source).toContain("$offlineStatus");
});
