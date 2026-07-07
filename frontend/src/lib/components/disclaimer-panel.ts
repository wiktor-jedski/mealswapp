import { loadDisclaimer } from "../api/auth-client";
import type { DisclaimerData } from "../api/generated";

// Implements DESIGN-018 DisclaimerPanel login-screen disclaimer loading and fallback view model.

/** Login-screen disclaimer shape consumed by the auth UI. */
export interface DisclaimerViewModel {
	version: string;
	bodyMarkdown: string;
	effectiveAt: string;
	unavailable: boolean;
}

/** Side-effect boundary used to load disclaimer content. */
export interface DisclaimerPanelDependencies {
	loadDisclaimer: typeof loadDisclaimer;
}

/** Bundled medical disclaimer shown when the generated disclaimer endpoint is unavailable. */
export const BUNDLED_LOGIN_DISCLAIMER: DisclaimerViewModel = {
	version: "bundled-2026-07",
	effectiveAt: "2026-07-05T00:00:00.000Z",
	unavailable: true,
	bodyMarkdown:
		"Mealswapp provides nutrition search and meal-planning support only. It is not medical advice, diagnosis, or treatment. Consult a qualified clinician before changing a diet for allergies, medical conditions, medication interactions, pregnancy, or pediatric care."
};

const defaultDependencies: DisclaimerPanelDependencies = { loadDisclaimer };

/** Loads generated login disclaimer content, falling back to bundled medical disclaimer copy. */
export async function loadLoginDisclaimerViewModel(
	dependencies: DisclaimerPanelDependencies = defaultDependencies,
	signal?: AbortSignal
): Promise<DisclaimerViewModel> {
	try {
		const data = await dependencies.loadDisclaimer("login", signal);
		return disclaimerDataToViewModel(data, false);
	} catch {
		return { ...BUNDLED_LOGIN_DISCLAIMER };
	}
}

function disclaimerDataToViewModel(data: DisclaimerData, unavailable: boolean): DisclaimerViewModel {
	return {
		version: data.version,
		bodyMarkdown: data.markdown,
		effectiveAt: data.effectiveAt,
		unavailable
	};
}
