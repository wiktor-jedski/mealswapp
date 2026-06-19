import { get, writable } from "svelte/store";

/**
 * User-selectable theme preference.
 *
 * @remarks Implements DESIGN-016 ThemeProvider theme state contracts.
 */
export type ThemePreference = "system" | "light" | "dark";

/**
 * Concrete theme applied to the document after resolving system preference.
 *
 * @remarks Implements DESIGN-016 ThemeProvider theme state contracts.
 */
export type ResolvedTheme = "light" | "dark";

/**
 * Stores the current user theme preference.
 *
 * @remarks Implements DESIGN-016 ThemeProvider theme state contracts.
 */
export const themePreference = writable<ThemePreference>("system");

/**
 * Stores the resolved light or dark theme.
 *
 * @remarks Implements DESIGN-016 ThemeProvider theme state contracts.
 */
export const resolvedTheme = writable<ResolvedTheme>("light");

const storageKey = "mealswapp.theme";

/**
 * Resolves a user preference into the concrete theme that should be applied.
 *
 * @remarks Implements DESIGN-016 ThemeProvider resolveTheme.
 */
export function resolveTheme(preference: ThemePreference, systemTheme: ResolvedTheme): ResolvedTheme {
	return preference === "system" ? systemTheme : preference;
}

/**
 * Initializes theme state from browser storage when browser globals are available.
 *
 * @remarks Implements DESIGN-016 ThemeProvider startup initialization.
 */
export function initTheme(): () => void {
	if (typeof window === "undefined") {
		return () => {};
	}

	let stored: ThemePreference | null = null;
	try { stored = window.localStorage.getItem(storageKey) as ThemePreference | null; } catch { /* Use system preference in memory. */ }
	const preference = stored === "light" || stored === "dark" || stored === "system" ? stored : "system";
	setThemePreference(preference);
	const media = window.matchMedia("(prefers-color-scheme: dark)");
	const changed = (event: MediaQueryListEvent) => {
		if (get(themePreference) !== "system") return;
		const resolved = event.matches ? "dark" : "light";
		resolvedTheme.set(resolved);
		document.documentElement.dataset.theme = resolved;
	};
	media.addEventListener?.("change", changed);
	return () => media.removeEventListener?.("change", changed);
}

/**
 * Persists the user preference and applies the resolved theme to the document.
 *
 * @remarks Implements DESIGN-016 ThemeProvider preference persistence and token application.
 */
export function setThemePreference(preference: ThemePreference): void {
	themePreference.set(preference);

	if (typeof window === "undefined") {
		resolvedTheme.set(resolveTheme(preference, "light"));
		return;
	}

	const systemTheme = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
	const resolved = resolveTheme(preference, systemTheme);
	resolvedTheme.set(resolved);
	document.documentElement.dataset.theme = resolved;
	try { window.localStorage.setItem(storageKey, preference); } catch { /* Keep the selected theme in memory. */ }
}
