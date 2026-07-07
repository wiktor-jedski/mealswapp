import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

// Implements DESIGN-018 LoginView component source verification for login-only auth behavior.

const source = readFileSync(join(import.meta.dir, "LoginView.svelte"), "utf8");

test("renders accessible email and password controls with safe autocomplete hints", () => {
	expect(source).toContain('for="login-email"');
	expect(source).toContain('id="login-email"');
	expect(source).toContain('autocomplete="email"');
	expect(source).toContain('for="login-password"');
	expect(source).toContain('id="login-password"');
	expect(source).toContain('autocomplete="current-password"');
	expect(source).toContain('aria-labelledby="login-view-title"');
});

test("validates completeness and disables duplicate submissions", () => {
	expect(source).toContain("let canSubmit = $derived(email.trim().length > 0 && password.length > 0 && !submitting)");
	expect(source).toContain('disabled={!canSubmit}');
	expect(source).toContain('submitting ? "Signing in..." : "Sign in"');
	expect(source).toContain('"Enter your email and password."');
});

test("uses generated login store handoff and clears password after success or failure", () => {
	expect(source).toContain("const request: LoginRequest");
	expect(source).toContain("await loginWithEmail(request)");
	expect(source).toContain("password = \"\"");
	expect(source.match(/password = ""/g)?.length).toBeGreaterThanOrEqual(2);
	expect(source).toContain("await runQueuedProtectedActionAfterAuth()");
});

test("maps invalid credentials generically and renders safe retry timing for lockouts", () => {
	expect(source).toContain('"Email or password is incorrect."');
	expect(source).not.toContain("No account");
	expect(source).not.toContain("email exists");
	expect(source).toContain("normalizeRetryAfterSeconds(error.retryAfterSeconds)");
	expect(source).toContain("const maxLoginRetryAfterSeconds = 60 * 60");
	expect(source).toContain("Try again in {retryAfterSeconds} seconds.");
	expect(source).toContain('role="alert"');
});

test("cites the design source", () => {
	expect(source).toContain("Implements DESIGN-018 LoginView");
});
