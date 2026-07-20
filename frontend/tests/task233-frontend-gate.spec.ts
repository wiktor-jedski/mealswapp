import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page, type Route, type TestInfo } from "@playwright/test";
import type {
	AuthSessionEnvelope,
	DailyDiet,
	DailyDietCollectionEnvelope,
	DailyDietEnvelope,
	DietOptimizationRequest,
	EntitlementStatusEnvelope,
	FoodObjectEnvelope,
	OptimizationJobStatusEnvelope,
	ProfileEnvelope
} from "../src/lib/api/generated";

// Implements DESIGN-001 SearchView Task 233 functional, browser, responsive, theme, and accessibility gate.

const USER_A = "00000000-0000-0000-0000-000000000233";
const USER_B = "00000000-0000-0000-0000-000000000234";
const DIET_A = "00000000-0000-0000-0000-000000000235";
const DIET_B = "00000000-0000-0000-0000-000000000236";
const JOB_ID = "00000000-0000-0000-0000-000000000237";
const APPLE_ID = "00000000-0000-0000-0000-000000000238";
const OATS_ID = "00000000-0000-0000-0000-000000000239";
const ENTRY_A = "00000000-0000-0000-0000-000000000240";
const ENTRY_B = "00000000-0000-0000-0000-000000000241";
const ALT_ID = "00000000-0000-0000-0000-000000000242";

type Outcome = "nominal" | "queue-once" | "infeasible" | "timeout-once" | "malformed-poll" | "processing";

interface GateOptions {
	lostCreateResponse?: boolean;
	malformedListOnce?: boolean;
	outcome?: Outcome;
	delayMealHydration?: boolean;
}

interface GateRuntime {
	createKeys: string[];
	replaceBodies: Record<string, unknown>[];
	submissionKeys: string[];
	submissions: DietOptimizationRequest[];
	polls: number;
	setOutcome(outcome: Outcome): void;
	waitForDelayedHydration(): Promise<void>;
	releaseDelayedHydration(): void;
	delayedHydrationWasAborted(): boolean;
}

function fulfillJson(route: Route, status: number, body: unknown): Promise<void> {
	return route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

function session(userId: string): AuthSessionEnvelope {
	return {
		status: "ok",
		requestId: `task-233-session-${userId}`,
		data: {
			userId,
			role: "user",
			hasVerifiedLoginMethod: true,
			accessExpiresAt: "2026-07-18T13:00:00Z",
			refreshExpiresAt: "2026-07-25T13:00:00Z"
		}
	};
}

function profile(userId: string): ProfileEnvelope {
	return {
		status: "ok",
		requestId: `task-233-profile-${userId}`,
		data: { userId, displayName: userId === USER_A ? "Account A" : "Account B", unitSystem: "metric", themePreference: "light", requiresUnitRecalculation: false }
	};
}

function entitlement(userId: string): EntitlementStatusEnvelope {
	return {
		status: "ok",
		requestId: `task-233-entitlement-${userId}`,
		data: {
			userId,
			tier: "paid",
			status: "active",
			allowedModes: ["catalog", "substitution", "daily_diet", "daily_diet_alternative"],
			searchLimitPer24h: 25,
			usageUsed: 0,
			usageRemaining: null,
			usageWindowStartedAt: "2026-07-18T00:00:00Z",
			trialExpiresAt: null,
			billingRecoveryState: "none"
		}
	};
}

function meal(id: typeof APPLE_ID | typeof OATS_ID): FoodObjectEnvelope {
	const apple = id === APPLE_ID;
	return {
		status: "ok",
		requestId: `task-233-meal-${id}`,
		data: {
			id,
			name: apple ? "Apple" : "Oats",
			physicalState: "solid",
			imageUrl: null,
			classifications: [{ id: "breakfast", name: "Breakfast", kind: "food_category" }],
			primaryFoodCategory: { id: "breakfast", name: "Breakfast", kind: "food_category" },
			macros: apple ? { protein: 1, carbohydrates: 14, fat: 0.2 } : { protein: 13, carbohydrates: 68, fat: 7 },
			macroBasis: "100g",
			calories: apple ? 52 : 389
		}
	};
}

function diet(id: string, name: string, macros = { protein: 31, carbohydrates: 82, fat: 7.2, calories: 500 }): DailyDiet {
	return {
		id,
		name,
		entries: [
			{ id: ENTRY_A, mealId: APPLE_ID, quantity: 150, unit: "g", position: 0 },
			{ id: ENTRY_B, mealId: OATS_ID, quantity: 100, unit: "g", position: 1 }
		],
		aggregateMacros: macros,
		createdAt: "2026-07-18T00:00:00Z",
		updatedAt: "2026-07-18T00:00:00Z"
	};
}

function job(status: "queued" | "processing" | "completed" | "failed", code?: "solver_infeasible" | "solver_timeout"): OptimizationJobStatusEnvelope {
	const base = { jobId: JOB_ID, dailyDietId: DIET_A, status, pollUrl: `/api/v1/optimization/jobs/${JOB_ID}`, createdAt: "2026-07-18T00:00:00Z" };
	if (status === "queued") return { status: "ok", requestId: "task-233-queued", data: base };
	if (status === "processing") return { status: "ok", requestId: "task-233-processing", data: { ...base, startedAt: "2026-07-18T00:00:01Z" } };
	if (status === "failed") {
		return {
			status: "ok",
			requestId: "task-233-failed",
			data: {
				...base,
				startedAt: "2026-07-18T00:00:01Z",
				finishedAt: "2026-07-18T00:00:02Z",
				alternatives: [],
				failure: { code: code!, message: code === "solver_timeout" ? "Optimization took too long. Please try again." : "No meal combination matches the requested targets." }
			}
		};
	}
	return {
		status: "ok",
		requestId: "task-233-completed",
		data: {
			...base,
			startedAt: "2026-07-18T00:00:01Z",
			finishedAt: "2026-07-18T00:00:02Z",
			alternatives: [{ meals: [{ mealId: ALT_ID, quantity: 120, unit: "g", position: 0 }], macros: { protein: 45, carbohydrates: 90, fat: 12, calories: 648 }, similarityScore: 0.91 }]
		}
	};
}

async function installGate(page: Page, options: GateOptions = {}): Promise<GateRuntime> {
	await page.emulateMedia({ reducedMotion: "reduce" });
	let currentUser: string | null = USER_A;
	let outcome: Outcome = options.outcome ?? "nominal";
	let listAttempts = 0;
	let createAttempts = 0;
	let submissionAttempts = 0;
	let outcomeSubmissionStart = 0;
	let delayedHydrationAborted = false;
	let markDelayedHydrationStarted!: () => void;
	let releaseDelayedHydration!: () => void;
	const delayedHydrationStarted = new Promise<void>((resolve) => { markDelayedHydrationStarted = resolve; });
	const delayedHydrationRelease = new Promise<void>((resolve) => { releaseDelayedHydration = resolve; });
	const diets = new Map<string, DailyDiet[]>([[USER_A, []], [USER_B, [diet(DIET_B, "Account B diet", { protein: 22, carbohydrates: 44, fat: 8, calories: 336 })]]]);
	const runtime: GateRuntime = {
		createKeys: [],
		replaceBodies: [],
		submissionKeys: [],
		submissions: [],
		polls: 0,
		setOutcome: (next) => { outcome = next; outcomeSubmissionStart = submissionAttempts; },
		waitForDelayedHydration: () => delayedHydrationStarted,
		releaseDelayedHydration,
		delayedHydrationWasAborted: () => delayedHydrationAborted
	};
	page.on("requestfailed", (request) => {
		if (options.delayMealHydration && request.url().endsWith(`/api/v1/food-objects/${APPLE_ID}`)) delayedHydrationAborted = true;
	});

	await page.route(/\/api\/v1\/profile$/, (route) => currentUser ? fulfillJson(route, 200, profile(currentUser)) : fulfillJson(route, 401, errorEnvelope("anonymous_session", "Please sign in.", false)));
	await page.route(/\/api\/v1\/auth\/refresh$/, (route) => currentUser ? fulfillJson(route, 200, session(currentUser)) : fulfillJson(route, 401, errorEnvelope("anonymous_session", "Please sign in.", false)));
	await page.route(/\/api\/v1\/auth\/csrf-token$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "task-233-csrf", data: { csrfToken: "csrf-task-233" } }));
	await page.route(/\/api\/v1\/auth\/logout$/, (route) => { currentUser = null; return fulfillJson(route, 200, { status: "ok", requestId: "task-233-logout" }); });
	await page.route(/\/api\/v1\/auth\/login$/, (route) => { currentUser = USER_B; return fulfillJson(route, 200, session(USER_B)); });
	await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => currentUser ? fulfillJson(route, 200, entitlement(currentUser)) : fulfillJson(route, 401, errorEnvelope("anonymous_session", "Please sign in.", false)));
	await page.route(/\/api\/v1\/search-history$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "task-233-history", data: { history: [] } }));
	await page.route(/\/api\/v1\/saved-items\?kind=favorite$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "task-233-favorites", data: { items: [] } }));
	await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => {
		const query = new URL(route.request().url()).searchParams.get("query")?.toLowerCase() ?? "";
		const items = [
			{ itemId: APPLE_ID, label: "Apple", exactMatch: query === "apple", levenshteinDistance: 0, length: 5, rank: 1 },
			{ itemId: OATS_ID, label: "Oats", exactMatch: query === "oats", levenshteinDistance: 0, length: 4, rank: 1 }
		].filter((item) => item.label.toLowerCase().includes(query));
		return fulfillJson(route, 200, { status: "ok", requestId: "task-233-autocomplete", data: { items } });
	});
	await page.route(`**/api/v1/food-objects/${APPLE_ID}`, async (route) => {
		if (options.delayMealHydration) {
			markDelayedHydrationStarted();
			await delayedHydrationRelease;
		}
		try {
			await fulfillJson(route, 200, meal(APPLE_ID));
		} catch (error) {
			if (route.request().failure()) {
				delayedHydrationAborted = true;
				return;
			}
			throw error;
		}
	});
	await page.route(`**/api/v1/food-objects/${OATS_ID}`, (route) => fulfillJson(route, 200, meal(OATS_ID)));
	await page.route(/\/api\/v1\/daily-diets$/, async (route) => {
		if (!currentUser) return fulfillJson(route, 401, errorEnvelope("anonymous_session", "Please sign in.", false));
		if (route.request().method() === "POST") {
			createAttempts += 1;
			runtime.createKeys.push(route.request().headers()["idempotency-key"] ?? "");
			const request = route.request().postDataJSON() as { name: string };
			const created = diet(DIET_A, request.name);
			diets.set(currentUser, [created]);
			if (options.lostCreateResponse && createAttempts === 1) return route.abort("failed");
			return fulfillJson(route, 201, { status: "ok", requestId: "task-233-created", data: created } satisfies DailyDietEnvelope);
		}
		listAttempts += 1;
		if (options.malformedListOnce && listAttempts === 1) return fulfillJson(route, 200, { status: "ok", requestId: "task-233-malformed", data: { diets: [{ id: "not-a-uuid", name: "<script>unsafe()</script>" }] } });
		return fulfillJson(route, 200, { status: "ok", requestId: "task-233-list", data: { diets: diets.get(currentUser) ?? [] } } satisfies DailyDietCollectionEnvelope);
	});
	await page.route(/\/api\/v1\/daily-diets\/[0-9a-f-]+$/, async (route) => {
		if (!currentUser) return fulfillJson(route, 401, errorEnvelope("anonymous_session", "Please sign in.", false));
		if (route.request().method() !== "PUT") return fulfillJson(route, 204, "");
		const request = route.request().postDataJSON() as Record<string, unknown>;
		runtime.replaceBodies.push(request);
		const replaced = diet(DIET_A, String(request.name), { protein: 45, carbohydrates: 90, fat: 12, calories: 648 });
		diets.set(currentUser, [replaced]);
		return fulfillJson(route, 200, { status: "ok", requestId: "task-233-replaced", data: replaced } satisfies DailyDietEnvelope);
	});
	await page.route(/\/api\/v1\/optimization\/jobs$/, async (route) => {
		submissionAttempts += 1;
		runtime.submissionKeys.push(route.request().headers()["idempotency-key"] ?? "");
		runtime.submissions.push(route.request().postDataJSON() as DietOptimizationRequest);
		if (outcome === "queue-once" && submissionAttempts === outcomeSubmissionStart + 1) return fulfillJson(route, 503, errorEnvelope("queue_unavailable", "redis://secret.internal failed", true));
		return fulfillJson(route, 202, { status: "accepted", requestId: "task-233-accepted", data: { jobId: JOB_ID, status: "queued", pollUrl: `/api/v1/optimization/jobs/${JOB_ID}` } });
	});
	await page.route(/\/api\/v1\/optimization\/jobs\/[0-9a-f-]+$/, async (route) => {
		runtime.polls += 1;
		if (outcome === "malformed-poll") return fulfillJson(route, 200, { status: "ok", requestId: "task-233-malformed-poll", data: { ...job("completed").data, alternatives: [{ backendDiagnostic: "postgres://secret" }] } });
		if (outcome === "infeasible") return fulfillJson(route, 200, job("failed", "solver_infeasible"));
		if (outcome === "timeout-once" && submissionAttempts === outcomeSubmissionStart + 1) return fulfillJson(route, 200, job("failed", "solver_timeout"));
		if (outcome === "processing") return fulfillJson(route, 200, job("processing"));
		if (runtime.polls === 1) return fulfillJson(route, 200, job("queued"));
		if (runtime.polls === 2) return fulfillJson(route, 200, job("processing"));
		return fulfillJson(route, 200, job("completed"));
	});
	return runtime;
}

function errorEnvelope(code: string, message: string, retryable: boolean): Record<string, unknown> {
	return { status: "error", requestId: `task-233-${code}`, error: { category: code === "anonymous_session" ? "auth" : "dependency", code, message, retryable } };
}

async function openMode(page: Page, name: "Catalog" | "Daily Diet" | "Daily Diet Alternative"): Promise<void> {
	const button = page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name, exact: true });
	await button.focus();
	await button.press("Enter");
}

async function addMeal(page: Page, query: string, label: string): Promise<void> {
	const input = page.getByLabel("Food search");
	await input.fill(query);
	await expect(page.getByRole("option", { name: label, exact: true })).toBeVisible();
	await input.press("Enter");
	await expect(page.locator("[data-daily-diet-meal]").filter({ hasText: label })).toBeVisible();
}

async function chooseDiet(page: Page, name: string): Promise<void> {
	await openMode(page, "Daily Diet Alternative");
	const choice = page.getByRole("radio", { name: `Use ${name} as Daily Diet Alternative input` });
	await choice.focus();
	await choice.press("Enter");
	await expect(page.locator("[data-optimization-workflow]")).toBeVisible();
}

async function setTheme(page: Page, theme: "light" | "dark"): Promise<void> {
	const open = page.getByLabel("Open activity sidebar");
	if (await open.isVisible()) await open.click();
	const toggle = page.getByLabel("Theme preference");
	const dark = await toggle.getAttribute("aria-pressed") === "true";
	if ((theme === "dark") !== dark) await toggle.click();
	await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
	const close = page.getByLabel("Close activity sidebar");
	if (await close.isVisible()) await close.click();
}

async function assertAccessibleAndSafe(page: Page): Promise<void> {
	const axe = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
	const serious = axe.violations.filter((violation) => violation.impact === "serious" || violation.impact === "critical");
	expect(serious, serious.map((violation) => `${violation.id}: ${violation.help}`).join("\n")).toEqual([]);
	await expect(page.locator("body")).not.toContainText(/redis:\/\/|postgres:\/\/|<script>|backendDiagnostic|secret\.internal/i);
	await expect(page.locator("[data-optimization-progress]")).toHaveCount(0);
	const layout = await page.evaluate(() => ({ width: Math.max(document.documentElement.scrollWidth, document.body.scrollWidth), viewport: window.innerWidth }));
	expect(layout.width).toBeLessThanOrEqual(layout.viewport + 1);
}

// Verifies IT-ARCH-004-001 and IT-ARCH-004-006, ARCH-004,
// DESIGN-001 SearchView,
// DESIGN-004 JobStatusTracker, and SW-REQ-006/SW-REQ-021/SW-REQ-030.
test("lost create response replays one write, replace installs authoritative macros, and selected optimization is safe in both themes", async ({ page }, testInfo: TestInfo) => {
	const runtime = await installGate(page, { lostCreateResponse: true });
	await page.goto("/");
	await openMode(page, "Daily Diet");
	await addMeal(page, "apple", "Apple");
	await addMeal(page, "oats", "Oats");
	await page.getByLabel("Collection name").fill("Training day");
	await page.getByRole("button", { name: "Save Daily Diet" }).press("Enter");
	await expect(page.locator("[data-daily-diet-save-error]")).toContainText("could not be saved");
	await page.getByRole("button", { name: "Save Daily Diet" }).press("Enter");
	await expect(page.locator("[data-daily-diet-server-total]")).toBeVisible();
	expect(runtime.createKeys).toHaveLength(2);
	expect(runtime.createKeys[1]).toBe(runtime.createKeys[0]);
	await expect(page.locator("[data-macro-protein]")).toHaveText("31g");

	await page.getByLabel("Quantity for Apple").fill("175");
	await expect(page.locator("[data-daily-diet-server-total]")).toHaveCount(0);
	await page.getByRole("button", { name: "Update Daily Diet" }).press("Enter");
	await expect(page.locator("[data-macro-protein]")).toHaveText("45g");
	expect(runtime.replaceBodies).toHaveLength(1);
	await chooseDiet(page, "Training day");
	await expect(page.locator("[data-optimization-target-protein]")).toHaveText("45g");
	await page.getByRole("button", { name: "Generate alternatives" }).press("Enter");
	await expect(page.locator("[data-optimization-progress]")).toContainText(/Queued|Building/);
	await expect(page.locator("[data-optimization-alternative]")).toHaveCount(1);
	expect(runtime.submissions[0]).toEqual({ dailyDietId: DIET_A, tolerancePercent: 10, excludedMealIds: [] });

	for (const theme of ["light", "dark"] as const) {
		await setTheme(page, theme);
		await assertAccessibleAndSafe(page);
		await page.screenshot({ path: testInfo.outputPath(`task-233-${theme}.png`), fullPage: true, animations: "disabled" });
	}
});

// Verifies IT-ARCH-004-006 and IT-ARCH-004-008, ARCH-004, DESIGN-001 SearchView,
// DESIGN-004 JobStatusTracker, DESIGN-017 ErrorMessageMapper, and
// SW-REQ-006/SW-REQ-021/SW-REQ-080.
test("malformed collection and optimization payloads fail closed and recover without rendering unsafe state", async ({ page }) => {
	const runtime = await installGate(page, { malformedListOnce: true, outcome: "malformed-poll" });
	await page.goto("/");
	await openMode(page, "Daily Diet Alternative");
	await expect(page.locator("[data-daily-diet-alternative-error]")).toBeVisible();
	await expect(page.locator("body")).not.toContainText("unsafe()");
	await page.getByRole("button", { name: "Try again" }).click();
	await expect(page.locator("[data-daily-diet-alternative-empty]")).toBeVisible();

	await openMode(page, "Daily Diet");
	await addMeal(page, "apple", "Apple");
	await addMeal(page, "oats", "Oats");
	await page.getByRole("button", { name: "Save Daily Diet" }).click();
	await chooseDiet(page, "My Daily Diet");
	await page.getByRole("button", { name: "Generate alternatives" }).click();
	await expect(page.locator("[data-optimization-error]")).toContainText("invalid response");
	await expect(page.locator("[data-optimization-results]")).toHaveCount(0);
	expect(runtime.polls).toBe(1);
	await assertAccessibleAndSafe(page);
});

// Verifies IT-ARCH-004-005 and IT-ARCH-004-006, ARCH-004,
// DESIGN-001 SearchView,
// DESIGN-004 JobStatusTracker, DESIGN-017 RetryManager, and
// SW-REQ-021/SW-REQ-080.
test("queue ambiguity reuses its key and terminal timeout rotates the next intentional submission", async ({ page }) => {
	const runtime = await installGate(page, { outcome: "queue-once" });
	await page.goto("/");
	await openMode(page, "Daily Diet");
	await addMeal(page, "apple", "Apple");
	await addMeal(page, "oats", "Oats");
	await page.getByRole("button", { name: "Save Daily Diet" }).click();
	await chooseDiet(page, "My Daily Diet");
	await page.getByRole("button", { name: "Generate alternatives" }).click();
	await expect(page.locator("[data-optimization-error]")).toContainText("queue is temporarily unavailable");
	await page.getByRole("button", { name: "Try again" }).press("Enter");
	await expect(page.locator("[data-optimization-alternative]")).toHaveCount(1);
	expect(runtime.submissionKeys[1]).toBe(runtime.submissionKeys[0]);

	runtime.setOutcome("timeout-once");
	await page.getByRole("button", { name: "Generate fresh alternatives" }).click();
	await expect(page.locator("[data-optimization-error]")).toContainText("took too long");
	await page.getByRole("button", { name: "Try again" }).click();
	await expect(page.locator("[data-optimization-alternative]")).toHaveCount(1);
	expect(new Set(runtime.submissionKeys).size).toBeGreaterThan(1);
	await assertAccessibleAndSafe(page);
});

test("infeasible output stays terminal, actionable, and free of stale alternatives", async ({ page }) => {
	await installGate(page, { outcome: "infeasible" });
	await page.goto("/");
	await openMode(page, "Daily Diet");
	await addMeal(page, "apple", "Apple");
	await addMeal(page, "oats", "Oats");
	await page.getByRole("button", { name: "Save Daily Diet" }).click();
	await chooseDiet(page, "My Daily Diet");
	await page.getByRole("button", { name: "Generate alternatives" }).click();
	await expect(page.locator("[data-optimization-error]")).toContainText("No meal combination matched");
	await expect(page.locator("[data-optimization-error]")).toContainText("increasing the tolerance");
	await expect(page.locator("[data-optimization-results]")).toHaveCount(0);
	await assertAccessibleAndSafe(page);
});

// Verifies IT-ARCH-004-006 and IT-ARCH-004-008, ARCH-004, DESIGN-001 SearchView,
// DESIGN-004 JobStatusTracker, DESIGN-017 RetryManager, and
// SW-REQ-006/SW-REQ-043/SW-REQ-080 identity-safe cancellation and recovery.
test("remount resumes acknowledged polling, then logout and account change clear every prior-user artifact", async ({ page }) => {
	const runtime = await installGate(page, { outcome: "processing" });
	await page.goto("/");
	await openMode(page, "Daily Diet");
	await addMeal(page, "apple", "Apple");
	await addMeal(page, "oats", "Oats");
	await page.getByRole("button", { name: "Save Daily Diet" }).click();
	await chooseDiet(page, "My Daily Diet");
	await page.getByRole("button", { name: "Generate alternatives" }).click();
	await expect(page.locator("[data-optimization-progress]")).toContainText("Building validated alternatives");
	const submissionsBeforeRemount = runtime.submissions.length;
	await openMode(page, "Catalog");
	runtime.setOutcome("nominal");
	await chooseDiet(page, "My Daily Diet");
	await expect(page.locator("[data-optimization-alternative]")).toHaveCount(1);
	expect(runtime.submissions).toHaveLength(submissionsBeforeRemount);

	runtime.setOutcome("processing");
	await page.getByRole("button", { name: "Generate fresh alternatives" }).click();
	await expect(page.locator("[data-optimization-progress]")).toBeVisible();
	const open = page.getByLabel("Open activity sidebar");
	if (await open.isVisible()) await open.click();
	await page.getByRole("button", { name: "Sign out" }).click();
	await expect(page.locator("[data-optimization-results]")).toHaveCount(0);
	await expect(page.locator("[data-daily-diet-alternative-auth-guidance]")).toBeVisible();

	const signIn = page.getByRole("button", { name: "Sign in to continue" });
	await signIn.click();
	await page.getByLabel("Email").fill("account-b@example.com");
	await page.getByLabel("Password").fill("CorrectHorseBatteryStaple1!");
	await page.getByRole("dialog", { name: "Sign in" }).getByRole("form").getByRole("button", { name: "Sign in", exact: true }).click();
	await expect(page.getByRole("radio", { name: "Use Account B diet as Daily Diet Alternative input" })).toBeVisible();
	await page.getByRole("radio", { name: "Use Account B diet as Daily Diet Alternative input" }).click();
	await expect(page.locator("[data-optimization-target-protein]")).toHaveText("22g");
	await expect(page.locator("[data-optimization-results]")).toHaveCount(0);
	await assertAccessibleAndSafe(page);
});

// Verifies IT-ARCH-004-006, ARCH-004, DESIGN-001 SearchView,
// DESIGN-017 RetryManager, and SW-REQ-006/SW-REQ-043 identity cancellation.
test("delayed Daily Diet hydration cannot cross logout and account change", async ({ page }) => {
	const runtime = await installGate(page, { delayMealHydration: true });
	await page.goto("/");
	await openMode(page, "Daily Diet");
	await page.getByLabel("Food search").fill("apple");
	await expect(page.getByRole("option", { name: "Apple", exact: true })).toBeVisible();
	await page.getByLabel("Food search").press("Enter");
	await runtime.waitForDelayedHydration();

	const open = page.getByLabel("Open activity sidebar");
	if (await open.isVisible()) await open.click();
	await page.getByRole("button", { name: "Sign out" }).click();
	await expect(page.locator("[data-daily-diet-auth-guidance]")).toBeVisible();
	await page.getByRole("button", { name: "Sign in to continue" }).click();
	await page.getByLabel("Email").fill("account-b@example.com");
	await page.getByLabel("Password").fill("CorrectHorseBatteryStaple1!");
	await page.getByRole("dialog", { name: "Sign in" }).getByRole("form").getByRole("button", { name: "Sign in", exact: true }).click();
	await expect(page.locator(`[data-saved-daily-diet="${DIET_B}"]`)).toBeVisible();

	runtime.releaseDelayedHydration();
	await expect.poll(runtime.delayedHydrationWasAborted).toBe(true);
	await expect(page.locator("[data-daily-diet-meal]")).toHaveCount(0);
});

test("delayed Daily Diet hydration cannot cross a mode change", async ({ page }) => {
	const runtime = await installGate(page, { delayMealHydration: true });
	await page.goto("/");
	await openMode(page, "Daily Diet");
	await page.getByLabel("Food search").fill("apple");
	await expect(page.getByRole("option", { name: "Apple", exact: true })).toBeVisible();
	await page.getByLabel("Food search").press("Enter");
	await runtime.waitForDelayedHydration();

	await openMode(page, "Catalog");
	runtime.releaseDelayedHydration();
	await expect.poll(runtime.delayedHydrationWasAborted).toBe(true);
	await openMode(page, "Daily Diet");
	await expect(page.locator("[data-daily-diet-meal]")).toHaveCount(0);
});
