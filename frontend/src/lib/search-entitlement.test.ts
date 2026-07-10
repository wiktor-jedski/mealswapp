import { expect, test } from "bun:test";

import type { AppError, EntitlementStatusData, SearchMode } from "./api/generated";
import { resolveSearchEntitlement, usageRemainingText } from "./search-entitlement";

// Implements DESIGN-001 SearchView entitlement gating unit verification.

function entitlement(overrides: Partial<EntitlementStatusData> = {}): EntitlementStatusData {
	return {
		userId: "user-entitlement",
		tier: "free",
		status: "active",
		allowedModes: ["catalog", "substitution"],
		searchLimitPer24h: 25,
		usageUsed: 5,
		usageRemaining: 20,
		usageWindowStartedAt: "2026-07-02T00:00:00Z",
		trialExpiresAt: null,
		billingRecoveryState: "none",
		...overrides
	};
}

function decision(status: EntitlementStatusData | null, mode: SearchMode, substitutionInputCount = 0, error: AppError | null = null) {
	return resolveSearchEntitlement({ status, error, mode, substitutionInputCount });
}

// Implements DESIGN-001 SearchView free-user usage counter verification.
test("formats generated free usage remaining for the SearchView status line", () => {
	expect(usageRemainingText(entitlement({ usageRemaining: 1 }))).toBe("1 free search remaining");
	expect(usageRemainingText(entitlement({ usageRemaining: 20 }))).toBe("20 free searches remaining");
	expect(usageRemainingText(entitlement({ usageRemaining: null }))).toBeNull();
});

// Implements DESIGN-001 SearchView free single-input Substitution gate verification.
test("allows free single-input Substitution until the usage limit", () => {
	expect(decision(entitlement(), "substitution", 1)).toEqual({
		canExecute: true,
		feedback: null,
		usageText: "20 free searches remaining"
	});
});

// Implements DESIGN-001 SearchView free multi-input Substitution gate verification.
test("blocks free multi-input Substitution before request execution", () => {
	const result = decision(entitlement(), "substitution", 2);

	expect(result.canExecute).toBe(false);
	expect(result.feedback).toContain("support one input");
});

// Implements DESIGN-001 SearchView free usage-limit gate verification.
test("blocks free Substitution when no usage remains", () => {
	const result = decision(entitlement({ usageRemaining: 0 }), "substitution", 1);

	expect(result.canExecute).toBe(false);
	expect(result.feedback).toContain("used all free Substitution searches");
});

// Implements DESIGN-001 SearchView paid-mode entitlement verification.
test("blocks Daily Diet modes for free users and allows them for trial and paid users", () => {
	expect(decision(entitlement({ allowedModes: ["catalog", "substitution", "daily_diet_alternative"] }), "daily_diet_alternative").canExecute).toBe(false);
	expect(decision(entitlement({ tier: "trial", allowedModes: ["catalog", "substitution", "daily_diet", "daily_diet_alternative"], usageRemaining: null }), "daily_diet").canExecute).toBe(true);
	expect(decision(entitlement({ tier: "paid", allowedModes: ["catalog", "substitution", "daily_diet_alternative"], usageRemaining: null }), "daily_diet_alternative").canExecute).toBe(true);
});

// Implements DESIGN-001 SearchView anonymous Catalog Search verification.
test("keeps anonymous Catalog Search usable while blocking paid modes after an auth entitlement error", () => {
	const authError: AppError = {
		category: "auth",
		code: "anonymous_session",
		message: "Sign in to view your billing status.",
		retryable: false
	};

	expect(decision(null, "catalog", 0, authError).canExecute).toBe(true);
	expect(decision(null, "daily_diet_alternative", 0, authError).canExecute).toBe(false);
});
