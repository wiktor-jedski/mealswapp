import { expect, test, type Page, type Route } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import type {
	AuthSessionEnvelope,
	DailyDiet,
	DailyDietCollectionEnvelope,
	EntitlementStatusEnvelope,
	ProfileEnvelope
} from "../src/lib/api/generated";

// Implements DESIGN-001 SearchView OptimizationWorkflow browser coverage for task 205.
// Implements DESIGN-004 JobStatusTracker submission, polling, safe retry, and terminal result projection.
// Verifies IT-ARCH-004-006, ARCH-004, and SW-REQ-006/SW-REQ-021/SW-REQ-030.

async function fulfillJson(route: Route, status: number, body: unknown): Promise<void> {
	await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

const profile: ProfileEnvelope = {
	status: "ok",
	requestId: "optimization-profile",
	data: { userId: "optimization-user", displayName: "Optimization User", unitSystem: "metric", themePreference: "system", requiresUnitRecalculation: false }
};

const session: AuthSessionEnvelope = {
	status: "ok",
	requestId: "optimization-session",
	data: { userId: "optimization-user", role: "user", hasVerifiedLoginMethod: true, accessExpiresAt: "2026-07-12T13:00:00Z", refreshExpiresAt: "2026-07-19T13:00:00Z" }
};

const entitlement: EntitlementStatusEnvelope = {
	status: "ok",
	requestId: "optimization-entitlement",
	data: {
		userId: "optimization-user",
		tier: "paid",
		status: "active",
		allowedModes: ["catalog", "substitution", "daily_diet", "daily_diet_alternative"],
		searchLimitPer24h: 25,
		usageUsed: 0,
		usageRemaining: null,
		usageWindowStartedAt: "2026-07-11T00:00:00Z",
		trialExpiresAt: null,
		billingRecoveryState: "none"
	}
};

function savedDiet(id = "00000000-0000-0000-0000-000000000001", name = "Training day"): DailyDiet {
	return {
		id,
		name,
		entries: [{ id: `${id}-entry`, mealId: "00000000-0000-0000-0000-000000000010", quantity: 100, unit: "g", position: 0 }],
		aggregateMacros: { protein: 40, carbohydrates: 80, fat: 20, calories: 640 },
		createdAt: "2026-07-11T00:00:00Z",
		updatedAt: "2026-07-11T00:00:00Z"
	};
}

async function stubAuthenticatedPage(page: Page, diets = [savedDiet()]): Promise<void> {
	await page.route(/\/api\/v1\/profile$/, (route) => fulfillJson(route, 200, profile));
	await page.route(/\/api\/v1\/auth\/refresh$/, (route) => fulfillJson(route, 200, session));
	await page.route(/\/api\/v1\/auth\/csrf-token$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "optimization-csrf", data: { csrfToken: "csrf-optimization" } }));
	await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => fulfillJson(route, 200, entitlement));
	await page.route(/\/api\/v1\/search-history$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "optimization-history", data: { history: [] } }));
	await page.route(/\/api\/v1\/saved-items\?kind=favorite$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "optimization-favorites", data: { items: [] } }));
	await page.route(/\/api\/v1\/daily-diets$/, (route) => fulfillJson(route, 200, { status: "ok", requestId: "optimization-diets", data: { diets } } satisfies DailyDietCollectionEnvelope));
}

async function chooseDiet(page: Page, name = "Training day"): Promise<void> {
	await page.getByRole("button", { name: "Daily Diet Alternative", exact: true }).click();
	await expect(page.getByRole("radio", { name: `Use ${name} as Daily Diet Alternative input` })).toBeVisible();
	await page.getByRole("radio", { name: `Use ${name} as Daily Diet Alternative input` }).click();
	await expect(page.locator("[data-optimization-workflow]")).toBeVisible();
}

function acknowledgement(jobId: string): Record<string, unknown> {
	return { status: "accepted", requestId: "optimization-accepted", data: { jobId, status: "queued", pollUrl: `/api/v1/optimization/jobs/${jobId}` } };
}

function job(jobId: string, status: "queued" | "processing" | "completed"): Record<string, unknown> {
	const base = {
		jobId,
		dailyDietId: "00000000-0000-0000-0000-000000000001",
		status,
		pollUrl: `/api/v1/optimization/jobs/${jobId}`,
		createdAt: "2026-07-11T00:00:00Z"
	};
	if (status === "queued") return { status: "ok", requestId: "optimization-queued", data: base };
	if (status === "processing") return { status: "ok", requestId: "optimization-processing", data: { ...base, startedAt: "2026-07-11T00:00:01Z" } };
	return {
		status: "ok",
		requestId: "optimization-completed",
		data: {
			...base,
			startedAt: "2026-07-11T00:00:01Z",
			finishedAt: "2026-07-11T00:00:02Z",
			alternatives: [
				{ meals: [{ mealId: "meal-a", quantity: 100, unit: "g", position: 0 }], macros: { protein: 40, carbohydrates: 80, fat: 20, calories: 620 }, similarityScore: 0.91 },
				{ meals: [{ mealId: "meal-b", quantity: 120, unit: "g", position: 0 }], macros: { protein: 41, carbohydrates: 79, fat: 20, calories: 630 }, similarityScore: 0.82 },
				{ meals: [{ mealId: "meal-c", quantity: 90, unit: "g", position: 0 }], macros: { protein: 39, carbohydrates: 81, fat: 21, calories: 640 }, similarityScore: 0.73 }
			]
		}
	};
}

test("submits generated optimization request, shows bounded skeleton progress, renders three alternatives, and passes axe", async ({ page }) => {
	await stubAuthenticatedPage(page);
	const jobId = "00000000-0000-0000-0000-000000000002";
	const keys: string[] = [];
	let polls = 0;
	let body: Record<string, unknown> | null = null;
	await page.route(/\/api\/v1\/optimization\/jobs$/, async (route) => {
		body = route.request().postDataJSON() as Record<string, unknown>;
		keys.push(route.request().headers()["idempotency-key"] ?? "");
		await fulfillJson(route, 202, acknowledgement(jobId));
	});
	await page.route(/\/api\/v1\/optimization\/jobs\/[0-9a-f-]+$/, async (route) => {
		polls += 1;
		await fulfillJson(route, 200, job(jobId, polls === 1 ? "queued" : polls === 2 ? "processing" : "completed"));
	});

	await page.goto("/");
	await chooseDiet(page);
	const submit = page.getByRole("button", { name: "Generate alternatives" });
	await submit.focus();
	await submit.press("Enter");
	await expect(page.locator("[data-optimization-skeleton]")).toBeVisible();
	await expect(page.locator("[data-optimization-alternative]")).toHaveCount(3);
	await expect(page.locator("[data-optimization-calories]").first()).toHaveText("620 kcal");
	await expect(page.locator("[data-optimization-results]")).toContainText("Validated alternatives");
	await expect(page.getByRole("button", { name: "Generate fresh alternatives" })).toBeVisible();

	expect(body).toEqual({ dailyDietId: "00000000-0000-0000-0000-000000000001", tolerancePercent: 10, excludedMealIds: [] });
	expect(keys).toHaveLength(1);
	expect(keys[0]?.length).toBeGreaterThanOrEqual(8);
	const axe = await new AxeBuilder({ page }).include("[data-optimization-workflow]").analyze();
	expect(axe.violations.filter((violation) => violation.impact === "serious" || violation.impact === "critical")).toEqual([]);
});

test("retries an ambiguous submission with the same key and does not leak results after selecting another diet", async ({ page }) => {
	const secondDiet = savedDiet("00000000-0000-0000-0000-000000000003", "Rest day");
	await stubAuthenticatedPage(page, [savedDiet(), secondDiet]);
	const jobId = "00000000-0000-0000-0000-000000000004";
	const keys: string[] = [];
	let submissions = 0;
	await page.route(/\/api\/v1\/optimization\/jobs$/, async (route) => {
		submissions += 1;
		keys.push(route.request().headers()["idempotency-key"] ?? "");
		if (submissions === 1) {
			await route.abort();
			return;
		}
		await fulfillJson(route, 202, acknowledgement(jobId));
	});
	await page.route(/\/api\/v1\/optimization\/jobs\/[0-9a-f-]+$/, (route) => fulfillJson(route, 200, job(jobId, "completed")));

	await page.goto("/");
	await chooseDiet(page);
	await page.getByRole("button", { name: "Generate alternatives" }).click();
	const retry = page.getByRole("button", { name: "Try again" });
	await expect(retry).toBeVisible();
	await retry.focus();
	await expect(retry).toBeFocused();
	await retry.press("Enter");
	await expect(page.locator("[data-optimization-results]")).toBeVisible();
	expect(submissions).toBe(2);
	expect(keys[1]).toBe(keys[0]);

	await page.getByRole("radio", { name: "Use Rest day as Daily Diet Alternative input" }).click();
	await expect(page.locator("[data-optimization-empty]")).toHaveCount(0);
	await expect(page.locator("[data-optimization-results]")).toHaveCount(0);
	await expect(page.locator("[data-optimization-submit]")).toBeEnabled();
});
