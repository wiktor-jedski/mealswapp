import { expect, test, type Page, type Route, type TestInfo } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import type {
	AuthSessionEnvelope,
	AutocompleteEnvelope,
	CSRFTokenEnvelope,
	DailyDiet,
	DailyDietCollectionEnvelope,
	DailyDietCreateRequest,
	DailyDietEnvelope,
	DietOptimizationRequest,
	EntitlementStatusEnvelope,
	FoodObjectEnvelope,
	OptimizationAlternative,
	OptimizationJobAcknowledgementEnvelope,
	OptimizationJobCompleted,
	OptimizationJobData,
	OptimizationJobFailed,
	OptimizationJobStatusEnvelope,
	ProfileEnvelope,
	SearchResponseEnvelope
} from "../src/lib/api/generated";

// Implements DESIGN-001 SearchView Task 207 browser acceptance coverage.
// Implements DESIGN-004 JobStatusTracker API-compatible worker fixtures and terminal-state paths.
// Implements DESIGN-008 SavedDataRepository authenticated selection, aggregation, and save coverage.
// Verifies IT-ARCH-004-001, IT-ARCH-004-005, IT-ARCH-004-006, IT-ARCH-004-008,
// ARCH-004, and SW-REQ-006/SW-REQ-021/SW-REQ-030/SW-REQ-042/SW-REQ-080.

type AuthFixture = "anonymous" | "free" | "trial" | "paid";
type Outcome = "nominal" | "infeasible" | "timeout" | "expired" | "retry-timeout";

interface BrowserFixture {
	auth: AuthFixture;
	outcome?: Outcome;
	initialDiet?: DailyDiet;
}

interface FixtureRuntime {
	dailyDiet: DailyDiet | null;
	submissions: DietOptimizationRequest[];
	idempotencyKeys: string[];
	polls: number;
}

const USER_ID = "task-207-user";
const DIET_ID = "00000000-0000-0000-0000-000000000207";
const JOB_ID = "00000000-0000-0000-0000-000000000207";
const APPLE_ID = "00000000-0000-0000-0000-000000000208";
const OATS_ID = "00000000-0000-0000-0000-000000000209";
const ENTRY_IDS = ["00000000-0000-0000-0000-000000000210", "00000000-0000-0000-0000-000000000211"] as const;

async function fulfillJson(route: Route, status: number, body: unknown): Promise<void> {
	await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

function appError(code: string, message: string, category: "auth" | "validation" | "dependency" | "timeout" = "validation") {
	return { category, code, message, retryable: code !== "solver_infeasible" };
}

function profileEnvelope(): ProfileEnvelope {
	return {
		status: "ok",
		requestId: "task-207-profile",
		data: {
			userId: USER_ID,
			displayName: "Task 207 User",
			unitSystem: "metric",
			themePreference: "light",
			requiresUnitRecalculation: false
		}
	};
}

function sessionEnvelope(): AuthSessionEnvelope {
	return {
		status: "ok",
		requestId: "task-207-session",
		data: {
			userId: USER_ID,
			role: "user",
			hasVerifiedLoginMethod: true,
			accessExpiresAt: "2026-07-11T13:00:00Z",
			refreshExpiresAt: "2026-07-18T13:00:00Z"
		}
	};
}

function entitlementEnvelope(tier: Exclude<AuthFixture, "anonymous">): EntitlementStatusEnvelope {
	const paidModes = ["catalog", "substitution", "daily_diet", "daily_diet_alternative"] as const;
	return {
		status: "ok",
		requestId: `task-207-entitlement-${tier}`,
		data: {
			userId: USER_ID,
			tier,
			status: "active",
			allowedModes: tier === "free" ? ["catalog"] : [...paidModes],
			searchLimitPer24h: tier === "free" ? 10 : 25,
			usageUsed: 0,
			usageRemaining: tier === "free" ? 10 : null,
			usageWindowStartedAt: "2026-07-11T00:00:00Z",
			trialExpiresAt: tier === "trial" ? "2026-07-18T00:00:00Z" : null,
			billingRecoveryState: "none"
		}
	};
}

function meal(id: typeof APPLE_ID | typeof OATS_ID): FoodObjectEnvelope {
	const apple = id === APPLE_ID;
	return {
		status: "ok",
		requestId: `task-207-${id}`,
		data: {
			id,
			objectType: "food_item",
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

function autocompleteEnvelope(query: string): AutocompleteEnvelope {
	const normalized = query.toLowerCase();
	const items = [
		{ itemId: APPLE_ID, objectType: "food_item" as const, label: "Apple", exactMatch: normalized === "apple", levenshteinDistance: 0, length: 5, rank: 1 },
		{ itemId: OATS_ID, objectType: "food_item" as const, label: "Oats", exactMatch: normalized === "oats", levenshteinDistance: 0, length: 4, rank: 2 }
	].filter((item) => item.label.toLowerCase().includes(normalized));
	return { status: "ok", requestId: "task-207-autocomplete", data: { items } };
}

function searchEnvelope(): SearchResponseEnvelope {
	return {
		status: "ok",
		requestId: "task-207-search",
		data: { items: [], totalCount: 0, page: 1, similarityScores: [], similarityMetadata: [], warnings: [] }
	};
}

function savedDiet(name = "Training day"): DailyDiet {
	return {
		id: DIET_ID,
		name,
		entries: [
			{ id: ENTRY_IDS[0], foodObjectId: APPLE_ID, foodObjectType: "food_item", quantity: 150, unit: "g", position: 0 },
			{ id: ENTRY_IDS[1], foodObjectId: OATS_ID, foodObjectType: "food_item", quantity: 100, unit: "g", position: 1 }
		],
		aggregateMacros: { protein: 31, carbohydrates: 82, fat: 7.2, calories: 500 },
		createdAt: "2026-07-11T00:00:00Z",
		updatedAt: "2026-07-11T00:00:00Z"
	};
}

function dietFromRequest(request: DailyDietCreateRequest): DailyDiet {
	return {
		...savedDiet(request.name),
		entries: request.entries.map((entry, index) => ({ id: ENTRY_IDS[index]!, ...entry }))
	};
}

function acknowledgement(): OptimizationJobAcknowledgementEnvelope {
	return {
		status: "accepted",
		requestId: "task-207-accepted",
		data: { jobId: JOB_ID, status: "queued", pollUrl: `/api/v1/optimization/jobs/${JOB_ID}` }
	};
}

function alternative(mealId: string, calories: number, similarityScore: number): OptimizationAlternative {
	return {
		meals: [{ mealId, quantity: 100, unit: "g", position: 0 }],
		macros: { protein: 31, carbohydrates: 82, fat: 7.2, calories },
		similarityScore
	};
}

function jobBase(status: OptimizationJobData["status"]): { jobId: string; dailyDietId: string; status: typeof status; pollUrl: string; createdAt: string } {
	return {
		jobId: JOB_ID,
		dailyDietId: DIET_ID,
		status,
		pollUrl: `/api/v1/optimization/jobs/${JOB_ID}`,
		createdAt: "2026-07-11T00:00:00Z"
	};
}

function completedJob(): OptimizationJobStatusEnvelope {
	const data: OptimizationJobCompleted = {
		...jobBase("completed"),
		startedAt: "2026-07-11T00:00:01Z",
		finishedAt: "2026-07-11T00:00:02Z",
		alternatives: [
			alternative(APPLE_ID, 460, 0.91),
			alternative(OATS_ID, 480, 0.82),
			alternative("00000000-0000-0000-0000-000000000212", 500, 0.73)
		]
	};
	return { status: "ok", requestId: "task-207-completed", data };
}

function failedJob(code: "solver_infeasible" | "solver_timeout"): OptimizationJobStatusEnvelope {
	const data: OptimizationJobFailed = {
		...jobBase("failed"),
		startedAt: "2026-07-11T00:00:01Z",
		finishedAt: "2026-07-11T00:00:02Z",
		alternatives: [],
		failure: { code, message: code === "solver_timeout" ? "Optimization took too long. Please try again." : "No meal combination matches the requested targets." }
	};
	return { status: "ok", requestId: `task-207-${code}`, data };
}

async function installFixture(page: Page, fixture: BrowserFixture): Promise<FixtureRuntime> {
	const runtime: FixtureRuntime = {
		dailyDiet: fixture.initialDiet ?? null,
		submissions: [],
		idempotencyKeys: [],
		polls: 0
	};
	const authenticated = fixture.auth !== "anonymous";

	await page.route(/\/api\/v1\/profile$/, (route) => authenticated
		? fulfillJson(route, 200, profileEnvelope())
		: fulfillJson(route, 401, { status: "error", requestId: "task-207-anonymous-profile", error: appError("anonymous_session", "Please sign in.", "auth") }));
	await page.route(/\/api\/v1\/auth\/refresh$/, (route) => authenticated
		? fulfillJson(route, 200, sessionEnvelope())
		: fulfillJson(route, 401, { status: "error", requestId: "task-207-anonymous-session", error: appError("anonymous_session", "Please sign in.", "auth") }));
	await page.route(/\/api\/v1\/auth\/csrf-token$/, (route) => fulfillJson(route, 200, {
		status: "ok",
		requestId: "task-207-csrf",
		data: { csrfToken: "csrf-task-207" }
	} satisfies CSRFTokenEnvelope));
	await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => authenticated
		? fulfillJson(route, 200, entitlementEnvelope(fixture.auth as Exclude<AuthFixture, "anonymous">))
		: fulfillJson(route, 401, { status: "error", requestId: "task-207-anonymous-entitlement", error: appError("anonymous_session", "Please sign in.", "auth") }));
	await page.route(/\/api\/v1\/search-history$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "task-207-history", data: { history: [] } }));
	await page.route(/\/api\/v1\/saved-items\?kind=favorite$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "task-207-favorites", data: { items: [] } }));
	await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => {
		const query = new URL(route.request().url()).searchParams.get("query") ?? "";
		return fulfillJson(route, 200, autocompleteEnvelope(query));
	});
	await page.route(new RegExp(`/api/v1/food-objects/(${APPLE_ID}|${OATS_ID})(?:\\?.*)?$`), (route) => {
		const id = new URL(route.request().url()).pathname.split("/").pop() as typeof APPLE_ID | typeof OATS_ID;
		return fulfillJson(route, 200, meal(id));
	});
	await page.route(/\/api\/v1\/search$/, (route) => fulfillJson(route, 200, searchEnvelope()));
	await page.route(/\/api\/v1\/daily-diets$/, async (route) => {
		if (route.request().method() === "POST") {
			if (fixture.auth === "free") {
				return fulfillJson(route, 403, { status: "error", requestId: "task-207-free-save", error: appError("entitlement_denied", "Daily Diet is available on trial and paid plans.", "dependency") });
			}
			const request = route.request().postDataJSON() as DailyDietCreateRequest;
			runtime.dailyDiet = dietFromRequest(request);
			return fulfillJson(route, 201, { status: "ok", requestId: "task-207-diet-created", data: runtime.dailyDiet } satisfies DailyDietEnvelope);
		}
		const diets = runtime.dailyDiet ? [runtime.dailyDiet] : [];
		return fulfillJson(route, 200, { status: "ok", requestId: "task-207-diet-list", data: { diets } } satisfies DailyDietCollectionEnvelope);
	});

	await page.route(/\/api\/v1\/optimization\/jobs$/, async (route) => {
		const request = route.request().postDataJSON() as DietOptimizationRequest;
		runtime.submissions.push(request);
		runtime.idempotencyKeys.push(route.request().headers()["idempotency-key"] ?? "");
		return fulfillJson(route, 202, acknowledgement());
	});
	await page.route(/\/api\/v1\/optimization\/jobs\/[0-9a-f-]+$/, async (route) => {
		runtime.polls += 1;
		if (fixture.outcome === "expired") {
			return fulfillJson(route, 410, { status: "error", requestId: "task-207-expired", error: appError("result_expired", "optimization result has expired", "validation") });
		}
		if (fixture.outcome === "infeasible") return fulfillJson(route, 200, failedJob("solver_infeasible"));
		if (fixture.outcome === "timeout" || (fixture.outcome === "retry-timeout" && runtime.submissions.length === 1)) {
			return fulfillJson(route, 200, failedJob("solver_timeout"));
		}
		if (fixture.outcome === "nominal" && runtime.polls === 1) {
			return fulfillJson(route, 200, { status: "ok", requestId: "task-207-queued", data: jobBase("queued") } satisfies OptimizationJobStatusEnvelope);
		}
		if (fixture.outcome === "nominal" && runtime.polls === 2) {
			return fulfillJson(route, 200, { status: "ok", requestId: "task-207-processing", data: { ...jobBase("processing"), startedAt: "2026-07-11T00:00:01Z" } } satisfies OptimizationJobStatusEnvelope);
		}
		return fulfillJson(route, 200, completedJob());
	});

	return runtime;
}

async function selectMeal(page: Page, query: string, label: string): Promise<void> {
	const search = page.getByLabel("Food search");
	await search.focus();
	await page.keyboard.type(query);
	await expect(page.getByRole("listbox", { name: "Autocomplete suggestions" })).toBeVisible();
	await expect(page.getByRole("option", { name: label, exact: true })).toBeVisible();
	await page.keyboard.press("Enter");
	await expect(page.locator("[data-daily-diet-meal]").filter({ hasText: label })).toBeVisible();
}

async function openDailyDiet(page: Page): Promise<void> {
	await page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Daily Diet", exact: true }).focus();
	await page.keyboard.press("Enter");
	await expect(page.locator("[data-daily-diet-collection]")).toBeVisible();
}

async function chooseSavedDiet(page: Page, name = "Training day"): Promise<void> {
	const navigation = page.getByRole("navigation", { name: "Search modes" });
	const dailyDietMode = navigation.getByRole("button", { name: "Daily Diet", exact: true });
	const mode = navigation.getByRole("button", { name: "Daily Diet Alternative", exact: true });
	await dailyDietMode.focus();
	await expect(dailyDietMode).toBeFocused();
	await page.keyboard.press("Tab");
	await expect(mode).toBeFocused();
	await page.keyboard.press("Enter");
	await expect(mode).toHaveAttribute("aria-pressed", "true");
	const choice = page.getByRole("radio", { name: `Use ${name} as Daily Diet Alternative input` });
	await expect(choice).toBeVisible();
	await choice.focus();
	await page.keyboard.press("Enter");
	await expect(page.locator("[data-optimization-workflow]")).toBeVisible();
}

async function saveTwoMealDiet(page: Page): Promise<void> {
	await selectMeal(page, "apple", "Apple");
	await selectMeal(page, "oats", "Oats");
	const quantity = page.getByLabel("Quantity for Apple");
	await quantity.focus();
	await page.keyboard.press("ControlOrMeta+A");
	await page.keyboard.type("150");
	const name = page.getByLabel("Collection name");
	await name.focus();
	await page.keyboard.press("ControlOrMeta+A");
	await page.keyboard.type("Training day");
	const save = page.getByRole("button", { name: "Save", exact: true });
	await save.focus();
	await page.keyboard.press("Enter");
	await expect(page.locator("[data-daily-diet-server-total]")).toHaveText("Totals confirmed by the server.");
}

async function assertNoOverflowOrClippedControls(page: Page): Promise<void> {
	const layout = await page.evaluate(() => {
		const visible = (element: Element): element is HTMLElement => {
			const node = element as HTMLElement;
			const style = getComputedStyle(node);
			return style.display !== "none" && style.visibility !== "hidden" && node.getClientRects().length > 0;
		};
		const controls = [...document.querySelectorAll("button, input, select, textarea")].filter(visible).map((element) => {
			const rect = element.getBoundingClientRect();
			return {
				label: element.getAttribute("aria-label") ?? element.textContent?.trim().slice(0, 40) ?? element.tagName,
				left: rect.left,
				right: rect.right,
				width: rect.width,
				overflows: element.scrollWidth > element.clientWidth + 1
			};
		});
		return {
			documentWidth: Math.max(document.documentElement.scrollWidth, document.body.scrollWidth),
			viewportWidth: window.innerWidth,
			clipped: controls.filter((control) => control.left < -1 || control.right > window.innerWidth + 1 || control.overflows)
		};
	});
	 expect(layout.documentWidth).toBeLessThanOrEqual(layout.viewportWidth + 1);
	 expect(layout.clipped, `controls outside viewport or clipped: ${JSON.stringify(layout.clipped)}`).toEqual([]);
}

async function assertAxe(page: Page): Promise<void> {
	const result = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
	const serious = result.violations.filter((violation) => violation.impact === "serious" || violation.impact === "critical");
	 expect(serious, serious.map((violation) => `${violation.id}: ${violation.help}`).join("\n")).toEqual([]);
}

async function disableMotion(page: Page): Promise<void> {
	await page.addStyleTag({ content: "*, *::before, *::after { animation: none !important; transition: none !important; }" });
}

test("paid fixture completes the keyboard-only meal selection, save, polling, alternatives, axe, responsive, and screenshot path", async ({ page }, testInfo: TestInfo) => {
	const runtime = await installFixture(page, { auth: "paid", outcome: "nominal" });
	await page.goto("/");
	await openDailyDiet(page);
	await expect(page.locator("[data-daily-diet-empty]")).toBeVisible();
	await saveTwoMealDiet(page);
	await expect(page.locator("[data-macro-protein]")).toHaveText("31g");
	await expect(page.locator("[data-macro-carbs]")).toHaveText("82g");
	await disableMotion(page);
	await page.screenshot({ path: testInfo.outputPath("task-207-daily-diet.png"), fullPage: true, animations: "disabled" });

	await chooseSavedDiet(page);
	const submit = page.getByRole("button", { name: "Generate alternatives" });
	await submit.focus();
	await page.keyboard.press("Enter");
	await expect(page.locator("[data-optimization-skeleton]")).toBeVisible();
	await expect(page.locator("[data-optimization-alternative]")).toHaveCount(3);
	await expect(page.locator("[data-optimization-results]")).toContainText("Validated alternatives");
	await expect(page.getByRole("button", { name: "Generate fresh alternatives" })).toBeVisible();

	expect(runtime.submissions).toEqual([{
		dailyDietId: DIET_ID,
		tolerancePercent: 10,
		excludedMealIds: []
	}]);
	expect(runtime.idempotencyKeys).toHaveLength(1);
	expect(runtime.idempotencyKeys[0]?.length).toBeGreaterThanOrEqual(8);
	expect(runtime.polls).toBe(3);
	await assertNoOverflowOrClippedControls(page);
	await assertAxe(page);
});

test("trial fixture can select a saved Daily Diet and complete optimization", async ({ page }) => {
	const runtime = await installFixture(page, { auth: "trial", outcome: "nominal", initialDiet: savedDiet() });
	await page.goto("/");
	await chooseSavedDiet(page);
	await page.getByRole("button", { name: "Generate alternatives" }).click();
	await expect(page.locator("[data-optimization-alternative]")).toHaveCount(3);
	await expect(page.locator("[data-optimization-entitlement]")).toHaveCount(0);
	expect(runtime.submissions).toHaveLength(1);
	await assertNoOverflowOrClippedControls(page);
	await assertAxe(page);
});

test("infeasible fixture presents the safe terminal message without alternatives", async ({ page }) => {
	await installFixture(page, { auth: "paid", outcome: "infeasible", initialDiet: savedDiet() });
	await page.goto("/");
	await chooseSavedDiet(page);
	await page.getByRole("button", { name: "Generate alternatives" }).click();
	await expect(page.locator("[data-optimization-error]")).toContainText("No meal combination matched");
	await expect(page.locator("[data-optimization-alternative]")).toHaveCount(0);
	await assertNoOverflowOrClippedControls(page);
	await assertAxe(page);
});

test("timeout fixture supports a keyboard retry with a new safe submission", async ({ page }) => {
	const runtime = await installFixture(page, { auth: "paid", outcome: "retry-timeout", initialDiet: savedDiet() });
	await page.goto("/");
	await chooseSavedDiet(page);
	await page.getByRole("button", { name: "Generate alternatives" }).click();
	await expect(page.locator("[data-optimization-error]")).toContainText("took too long");
	const retry = page.getByRole("button", { name: "Try again" });
	await retry.focus();
	await expect(retry).toBeFocused();
	await page.keyboard.press("Enter");
	await expect(page.locator("[data-optimization-alternative]")).toHaveCount(3);
	expect(runtime.submissions).toHaveLength(2);
	expect(runtime.idempotencyKeys[0]).not.toBe(runtime.idempotencyKeys[1]);
	await assertNoOverflowOrClippedControls(page);
	await assertAxe(page);
});

// Verifies IT-ARCH-004-008, ARCH-004, DESIGN-001 SearchView,
// DESIGN-004 JobStatusTracker, DESIGN-017 RetryManager, and SW-REQ-006/SW-REQ-043/SW-REQ-080.
test("expired-result fixture presents the retryable expired state and no stale result", async ({ page }) => {
	await installFixture(page, { auth: "paid", outcome: "expired", initialDiet: savedDiet() });
	await page.goto("/");
	await chooseSavedDiet(page);
	await page.getByRole("button", { name: "Generate alternatives" }).click();
	await expect(page.locator("[data-optimization-error]")).toContainText("result has expired");
	await expect(page.locator("[data-optimization-results]")).toHaveCount(0);
	await expect(page.getByRole("button", { name: "Try again" })).toBeVisible();
	await assertNoOverflowOrClippedControls(page);
	await assertAxe(page);
});

test("anonymous fixture gives sign-in guidance without protected Daily Diet requests", async ({ page }) => {
	let protectedDietRequests = 0;
	await installFixture(page, { auth: "anonymous" });
	await page.on("request", (request) => {
		if (request.url().includes("/api/v1/daily-diets")) protectedDietRequests += 1;
	});
	await page.goto("/");
	await openDailyDiet(page);
	await expect(page.locator("[data-daily-diet-auth-guidance]")).toBeVisible();
	await expect(page.getByRole("button", { name: "Sign in to continue" })).toBeVisible();
	expect(protectedDietRequests).toBe(0);
	await assertNoOverflowOrClippedControls(page);
	await assertAxe(page);
});

test("free fixture shows entitlement guidance and disables Daily Diet save and optimization", async ({ page }) => {
	await installFixture(page, { auth: "free" });
	await page.goto("/");
	await openDailyDiet(page);
	await expect(page.locator("[data-daily-diet-entitlement]")).toContainText("not included in your current plan");
	await expect(page.getByRole("button", { name: "Save", exact: true })).toBeDisabled();
	await page.getByRole("navigation", { name: "Search modes" }).getByRole("button", { name: "Daily Diet Alternative", exact: true }).click();
	await expect(page.locator("[data-daily-diet-alternative-entitlement]")).toContainText("not included in your current plan");
	await expect(page.locator("[data-optimization-submit]")).toHaveCount(0);
	await assertNoOverflowOrClippedControls(page);
	await assertAxe(page);
});
