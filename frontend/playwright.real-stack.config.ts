import { defineConfig, devices } from "@playwright/test";

// Implements DESIGN-018 AuthenticatedActionGuard real local-stack UAT harness.
export default defineConfig({
	testDir: "./tests",
	fullyParallel: false,
	forbidOnly: true,
	retries: 0,
	workers: 1,
	reporter: [["list"]],
	use: {
		baseURL: "http://localhost:5173",
		trace: "retain-on-failure",
		screenshot: "only-on-failure"
	},
	projects: [
		{
			name: "real-stack-desktop-chromium",
			use: { ...devices["Desktop Chrome"] }
		}
	],
	webServer: {
		command: "bun run dev -- --host localhost --port 5173 --strictPort",
		url: "http://localhost:5173",
		reuseExistingServer: true,
		timeout: 120_000
	}
});
