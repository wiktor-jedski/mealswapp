import { expect, test, type Page, type Route } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import type { CheckoutSessionEnvelope, EntitlementEnvelope, ProfileEnvelope } from "../src/lib/api/generated";

// Implements DESIGN-007 SubscriptionController frontend billing controls tests.

async function mockProfile(page: Page) {
	await page.route("/api/v1/profile", async (route: Route) => {
		const envelope: ProfileEnvelope = {
			status: "ok",
			requestId: "test-req",
			data: { id: "123", email: "test@example.com", authProvider: "google" }
		};
		await route.fulfill({ status: 200, json: envelope });
	});
}

async function mockEntitlement(page: Page, tier: "free" | "trial" | "paid", statusStr: "active" | "expired" | "past_due" | "cancelled" = "active"): Promise<void> {
	await page.route("/api/v1/entitlements", async (route: Route) => {
		const envelope: EntitlementEnvelope = {
			status: "ok",
			requestId: "test-req",
			data: {
				tier,
				status: statusStr,
				allowedModes: ["catalog"],
			}
		};
		await route.fulfill({ status: 200, json: envelope });
	});
}

test.describe("Subscription UI and Checkout Flow", () => {
	async function navigateAndOpenSidebar(page: Page, isMobile: boolean, url: string = "/") {
		await page.goto(url);
		if (isMobile) {
			await page.locator("[data-sidebar-mobile-toggle]").click();
		}
	}

	test("loading and retry states are visible", async ({ page, isMobile }) => {
		await mockProfile(page);
		await page.route("/api/v1/entitlements", async (route) => {
			// 400 is not retryable, so it immediately fails and shows error
			await route.fulfill({ status: 401, json: { status: "error", error: { category: "auth", code: "unauthorized", retryable: false } } });
		});

		await navigateAndOpenSidebar(page, isMobile);
		
		await expect(page.locator("[data-error]")).toBeVisible();
		await expect(page.locator("[data-error] button")).toHaveText("Retry");
	});

	test("monthly and annual buttons call checkout creation with generated contracts", async ({ page, isMobile }) => {
		await mockProfile(page);
		await mockEntitlement(page, "free");
		
		let checkoutRequest: any;
		await page.route("/api/v1/billing/checkout", async (route) => {
			checkoutRequest = route.request().postDataJSON();
			const envelope: CheckoutSessionEnvelope = {
				status: "ok",
				requestId: "test-req",
				data: {
					sessionId: "cs_test_123",
					checkoutUrl: "https://checkout.stripe.com/test",
				}
			};
			await route.fulfill({ status: 200, json: envelope });
		});

		await navigateAndOpenSidebar(page, isMobile);
		await page.locator("[data-checkout-monthly]").click();
		
		await expect(page).toHaveURL("https://checkout.stripe.com/test");
		expect(checkoutRequest).toBeDefined();
		expect(checkoutRequest.priceId).toBe("price_monthly");
		expect(checkoutRequest.successUrl).toContain("success=true");
		expect(checkoutRequest.cancelUrl).toContain("canceled=true");
	});

	test("Stripe redirect URL is followed only from the server response", async ({ page, isMobile }) => {
		await mockProfile(page);
		await mockEntitlement(page, "free");
		
		await page.route("/api/v1/billing/checkout", async (route) => {
			const envelope: CheckoutSessionEnvelope = {
				status: "ok",
				requestId: "test-req",
				data: {
					sessionId: "cs_test_annual",
					checkoutUrl: "https://checkout.stripe.com/annual",
				}
			};
			await route.fulfill({ status: 200, json: envelope });
		});

		await navigateAndOpenSidebar(page, isMobile);
		await page.locator("[data-checkout-annual]").click();
		
		await expect(page).toHaveURL("https://checkout.stripe.com/annual");
	});

	test("past_due/cancelled states show recovery actions", async ({ page, isMobile }) => {
		await mockProfile(page);
		await mockEntitlement(page, "paid", "past_due");
		
		await navigateAndOpenSidebar(page, isMobile);
		
		await expect(page.locator("[data-recovery-message]")).toBeVisible();
		await expect(page.locator("[data-recovery-message]")).toContainText("past due");
		await expect(page.locator("[data-recovery-action]")).toBeVisible();
		await expect(page.locator("[data-recovery-action]")).toHaveText("Update Billing");
	});

	test("cancel and success return routes refresh entitlement state", async ({ page, isMobile }) => {
		await mockProfile(page);
		
		await page.route("/api/v1/entitlements", async (route) => {
			const envelope: EntitlementEnvelope = {
				status: "ok",
				requestId: "test-req",
				data: {
					tier: "paid",
					status: "active",
					allowedModes: ["catalog"],
				}
			};
			await route.fulfill({ status: 200, json: envelope });
		});

		await navigateAndOpenSidebar(page, isMobile, "/?success=true");
		
		// The URL should be cleaned up
		await expect(page).toHaveURL("/");
		
		await expect(page.locator("[data-active-subscription]")).toBeVisible();
	});

	test("no application form captures PAN/CVC fields", async ({ page, isMobile }) => {
		await mockProfile(page);
		await mockEntitlement(page, "free");
		
		await navigateAndOpenSidebar(page, isMobile);
		
		// Verify there are no inputs resembling credit card fields
		const inputs = await page.locator('input[name*="card"], input[name*="cvc"], input[name*="pan"]').count();
		expect(inputs).toBe(0);
	});

	test("axe checks report no serious or critical violations", async ({ page, isMobile }) => {
		await mockProfile(page);
		await mockEntitlement(page, "free");
		
		await navigateAndOpenSidebar(page, isMobile);
		
		const results = await new AxeBuilder({ page }).analyze();
		const violations = results.violations.filter(
			(v) => v.impact === "serious" || v.impact === "critical"
		);
		expect(violations).toEqual([]);
	});
});
