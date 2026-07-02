import type { AppError, EntitlementStatusData, SearchMode } from "./api/generated";

// Implements DESIGN-001 SearchView entitlement gating decisions over generated billing contracts.

/**
 * Search UI context needed to decide whether the current mode may execute.
 *
 * @remarks Implements DESIGN-001 SearchView entitlement-aware mode execution gate.
 */
export interface SearchEntitlementContext {
	status: EntitlementStatusData | null;
	error: AppError | null;
	mode: SearchMode;
	substitutionInputCount: number;
}

/**
 * UI-facing entitlement decision for SearchView controls and request execution.
 *
 * @remarks Implements DESIGN-001 SearchView entitlement feedback state.
 */
export interface SearchEntitlementDecision {
	canExecute: boolean;
	feedback: string | null;
	usageText: string | null;
}

/**
 * Resolves the current SearchView entitlement decision without duplicating generated billing DTOs.
 *
 * @remarks Implements DESIGN-001 SearchView free/trial/paid search-mode entitlement gating.
 */
export function resolveSearchEntitlement(context: SearchEntitlementContext): SearchEntitlementDecision {
	const usageText = usageRemainingText(context.status);
	const feedback = entitlementFeedback(context);
	return {
		canExecute: feedback === null,
		feedback,
		usageText
	};
}

/**
 * Formats the generated remaining-usage field for a compact SearchView status line.
 *
 * @remarks Implements DESIGN-001 SearchView free-user usage counter display.
 */
export function usageRemainingText(status: EntitlementStatusData | null): string | null {
	if (status === null || status.usageRemaining === null) {
		return null;
	}
	const searchLabel = status.usageRemaining === 1 ? "search" : "searches";
	return `${status.usageRemaining} free ${searchLabel} remaining`;
}

function entitlementFeedback(context: SearchEntitlementContext): string | null {
	const { status, error, mode, substitutionInputCount } = context;
	if (mode === "catalog") {
		return null;
	}

	if (status === null) {
		if (error?.category === "auth" || error?.code === "anonymous_session") {
			return "Sign in to use paid search modes.";
		}
		return null;
	}

	if (!status.allowedModes.includes(mode)) {
		return `${modeLabel(mode)} is not included in your current plan.`;
	}

	if (status.tier === "free") {
		if (isDailyDietMode(mode)) {
			return `${modeLabel(mode)} is available on trial and paid plans.`;
		}
		if (mode === "substitution" && substitutionInputCount > 1) {
			return "Free Substitution searches support one input. Remove extra inputs or start a trial to compare multiple foods.";
		}
		if (mode === "substitution" && status.usageRemaining === 0) {
			return "You have used all free Substitution searches for this usage window.";
		}
	}

	return null;
}

function isDailyDietMode(mode: SearchMode): boolean {
	return mode === "daily_diet" || mode === "daily_diet_alternative";
}

function modeLabel(mode: SearchMode): string {
	switch (mode) {
		case "catalog":
			return "Catalog Search";
		case "substitution":
			return "Substitution";
		case "daily_diet":
			return "Daily Diet";
		case "daily_diet_alternative":
			return "Daily Diet Alternative";
	}
}
