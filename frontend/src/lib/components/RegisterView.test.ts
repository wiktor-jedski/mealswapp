import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-018 RegisterView static component verification.

const source = readFileSync(join(import.meta.dir, "RegisterView.svelte"), "utf8");

test("renders email, password confirmation, and current consent checkboxes", () => {
	expect(source).toContain('name="email"');
	expect(source).toContain("bind:this={emailInput}");
	expect(source).toContain("emailInput?.focus()");
	expect(source).toContain('name="password"');
	expect(source).toContain('name="confirmPassword"');
	expect(source).toContain('name="legalConsentAccepted"');
	expect(source).toContain("setCombinedConsent");
	expect(source).not.toContain("Accept all");
	expect(source).toContain('href="/privacy"');
	expect(source).toContain('href="/terms"');
	expect(source).toContain('target="_blank"');
	expect(source).toContain('rel="noreferrer"');
	expect(source).not.toContain("Current versions:");
	expect(source).not.toContain("Refresh legal versions");
	expect(source).not.toContain("Use email and password");
});

test("registration uses the auth session store and clears duplicate-email users toward login", () => {
	expect(source).toContain("registerSessionWithEmail");
	expect(source).toContain("onSwitchToLogin");
	expect(source).toContain("Log in instead");
	expect(source).toContain("An account already exists for this email.");
});

test("stale consent and unverified login method feedback are present", () => {
	expect(source).toContain("Privacy Policy or Terms changed.");
	expect(source).toContain("Verify your email before using features that require a verified login method.");
	expect(source).toContain("setCombinedConsent");
});

test("component cites the DESIGN source", () => {
	expect(source).toContain("Implements DESIGN-018 RegisterView");
	expect(source).toContain("Implements DESIGN-018 RegisterView email/password account creation and ConsentGate");
});
