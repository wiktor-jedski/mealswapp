import { defineConfig, devices } from "@playwright/test";
import { validateTask285Target } from "./task285-target";

// Implements DESIGN-014 LogAggregator deployed acceptance isolation.
const baseURL = await validateTask285Target(
	process.env.MEALSWAPP_TASK285_DEPLOYED_BASE_URL,
	process.env.MEALSWAPP_TASK285_DEPLOYMENT_ACK,
	process.env.MEALSWAPP_TASK285_APPROVED_DEPLOYED_HOST
);

export default defineConfig({
	testDir: "./tests",
	fullyParallel: false,
	forbidOnly: true,
	retries: 0,
	workers: 1,
	reporter: [["list"]],
	outputDir: process.env.PLAYWRIGHT_OUTPUT_DIR,
	use: {
		baseURL,
		trace: "off",
		screenshot: "off",
		video: "off"
	},
	projects: [{ name: "deployed-centralized-logging", use: { ...devices["Desktop Chrome"] } }]
});
