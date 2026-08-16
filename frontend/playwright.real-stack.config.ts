import { readFileSync, realpathSync, statSync } from "node:fs";
import { resolve } from "node:path";
import { defineConfig, devices } from "@playwright/test";

const managedByHarness = process.env.MEALSWAPP_REAL_STACK_MANAGED === "1";
const task281 = process.env.MEALSWAPP_TASK281_REAL_E2E === "1";
const task282 = process.env.MEALSWAPP_TASK282_REAL_E2E === "1";
const task283 = process.env.MEALSWAPP_TASK283_REAL_E2E === "1";
const task284 = process.env.MEALSWAPP_TASK284_REAL_E2E === "1";
const managedBaseURL = process.env.MEALSWAPP_REAL_STACK_BASE_URL;
if (task281) validateHarnessCapability("281");
if (task282) validateHarnessCapability("282");
if (task283) validateHarnessCapability("283");
if (task284) validateHarnessCapability("284");
const baseURL = managedBaseURL ?? "http://localhost:5173";

// Implements DESIGN-018 AuthenticatedActionGuard real local-stack UAT harness.
export default defineConfig({
	testDir: "./tests",
	fullyParallel: false,
	forbidOnly: true,
	retries: 0,
	workers: 1,
	reporter: task281 || task282 || task283 || task284 ? [["list"], ["./tests/phase08-acceptance-reporter.ts"]] : [["list"]],
	outputDir: process.env.PLAYWRIGHT_OUTPUT_DIR,
	use: {
		baseURL,
		trace: "retain-on-failure",
		screenshot: "only-on-failure"
	},
	projects: [
		{
			name: "real-stack-desktop-chromium",
			use: { ...devices["Desktop Chrome"] }
		},
		...(task281 || task283
			? [{
				name: "real-stack-mobile-chromium",
				use: { ...devices["Pixel 7"] }
			}]
			: [])
	],
	webServer: managedByHarness ? undefined : {
		command: "bun run dev -- --host localhost --port 5173 --strictPort",
		url: "http://localhost:5173",
		reuseExistingServer: true,
		timeout: 120_000
	}
});

interface HarnessCapability {
	schema: string;
	runId: string;
	nonce: string;
	baseURL: string;
	evidenceRoot: string;
	frontendProcess: { pid: number; startToken: string };
}

/** Validates the private capability manifest and proves the managed task targets its owned listener and evidence root. */
function validateHarnessCapability(task: "281" | "282" | "283" | "284"): void {
	const capabilityPath = process.env[`MEALSWAPP_TASK${task}_CAPABILITY_FILE`];
	const nonce = process.env[`MEALSWAPP_TASK${task}_CAPABILITY_NONCE`];
	const evidenceRoot = process.env.PHASE08_ACCEPTANCE_RESULT_DIR;
	if (!managedByHarness || !managedBaseURL || !capabilityPath || !nonce || !evidenceRoot) {
		throw new Error(`Task ${task} real-stack tests require the managed isolated harness and its owned base URL.`);
	}
	const path = realpathSync(capabilityPath);
	const file = statSync(path);
	const parent = statSync(resolve(path, ".."));
	if (!file.isFile() || file.uid !== process.getuid?.() || (file.mode & 0o077) !== 0 ||
		parent.uid !== process.getuid?.() || (parent.mode & 0o077) !== 0) {
		throw new Error(`Task ${task} harness capability is not private to the current user.`);
	}
	const capability = JSON.parse(readFileSync(path, "utf8")) as Partial<HarnessCapability>;
	if (
		capability.schema !== `mealswapp.task${task}-harness-capability.v1` ||
		!capability.runId?.match(/^[0-9a-f]{24}$/) ||
		!capability.nonce?.match(/^[0-9a-f]{48}$/) ||
		capability.nonce !== nonce
	) {
		throw new Error(`Task ${task} harness capability is invalid.`);
	}
	const expectedEvidenceRoot = resolve(process.cwd(), "../logs/real-stack-e2e", capability.runId, "acceptance");
	if (
		capability.baseURL !== managedBaseURL ||
		capability.evidenceRoot !== evidenceRoot ||
		resolve(evidenceRoot) !== expectedEvidenceRoot
	) {
		throw new Error(`Task ${task} base URL or evidence root does not match the harness capability manifest.`);
	}
	const url = new URL(managedBaseURL);
	if (
		url.protocol !== "http:" ||
		url.hostname !== "127.0.0.1" ||
		!url.port ||
		url.pathname !== "/" ||
		url.username ||
		url.password ||
		url.search ||
		url.hash
	) {
		throw new Error(`Task ${task} harness base URL must be an uncredentialed loopback HTTP origin.`);
	}
	const processIdentity = capability.frontendProcess;
	if (
		!processIdentity ||
		!Number.isSafeInteger(processIdentity.pid) ||
		processIdentity.pid <= 1 ||
		!processIdentity.startToken?.match(/^[1-9][0-9]*$/)
	) {
		throw new Error(`Task ${task} frontend process identity is invalid.`);
	}
	const proc = `/proc/${processIdentity.pid}`;
	const stat = readFileSync(`${proc}/stat`, "utf8");
	const fields = stat.slice(stat.lastIndexOf(")") + 2).trim().split(/\s+/);
	const command = readFileSync(`${proc}/cmdline`, "utf8");
	if (
		fields[19] !== processIdentity.startToken ||
		realpathSync(`${proc}/cwd`) !== realpathSync(process.cwd()) ||
		!command.includes(`--port\0${url.port}\0`) ||
		!command.includes("--strictPort")
	) {
		throw new Error(`Task ${task} base URL is not owned by the live managed frontend process.`);
	}
}
