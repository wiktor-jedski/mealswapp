import { expect, test } from "bun:test";
import { parseShellRoute, searchRoute, shellViewRoute } from "./shell-routing";

// Implements DESIGN-001 SearchView canonical SPA route verification.
test("parses top-level, search-mode, and billing-return routes", () => {
	expect(parseShellRoute("/")).toEqual({ view: "search", mode: "catalog", billingReturn: null });
	expect(parseShellRoute("/?mode=daily_diet")).toEqual({ view: "search", mode: "daily_diet", billingReturn: null });
	expect(parseShellRoute("/?mode=unknown")).toEqual({ view: "search", mode: "catalog", billingReturn: null });
	expect(parseShellRoute("/subscription?checkout=success&plan=monthly")).toEqual({ view: "subscription", mode: "catalog", billingReturn: "success" });
	expect(parseShellRoute("/billing/cancel?plan=annual")).toEqual({ view: "subscription", mode: "catalog", billingReturn: "cancel" });
	expect(parseShellRoute("/privacy").view).toBe("privacy");
	expect(parseShellRoute("/terms").view).toBe("terms");
	expect(parseShellRoute("/admin").view).toBe("administration");
});

// Implements DESIGN-001 SearchView canonical SPA URL generation verification.
test("generates clean view URLs and mode-preserving Search URLs", () => {
	expect(searchRoute("catalog")).toBe("/");
	expect(searchRoute("substitution")).toBe("/?mode=substitution");
	expect(shellViewRoute("search", "daily_diet_alternative")).toBe("/?mode=daily_diet_alternative");
	expect(shellViewRoute("subscription")).toBe("/subscription");
	expect(shellViewRoute("administration")).toBe("/admin");
	expect(shellViewRoute("privacy")).toBe("/privacy");
	expect(shellViewRoute("terms")).toBe("/terms");
});
