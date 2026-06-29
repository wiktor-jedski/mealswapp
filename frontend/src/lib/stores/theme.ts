import { writable } from "svelte/store";
import { get } from "svelte/store";

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
 * Internal handle to the `prefers-color-scheme` media query list so the change
 * listener registered by {@link initTheme} can be removed by {@link cleanupTheme}.
 *
 * @remarks Implements DESIGN-016 ThemeProvider system-theme subscription lifecycle.
 */
let mediaQueryList: MediaQueryList | null = null;

/**
 * Internal handle to the change listener registered on {@link mediaQueryList}.
 *
 * @remarks Implements DESIGN-016 ThemeProvider system-theme subscription lifecycle.
 */
let systemThemeHandler: ((event: MediaQueryListEvent) => void) | null = null;

/**
 * Resolves a user preference into the concrete theme that should be applied.
 *
 * @remarks Implements DESIGN-016 ThemeProvider resolveTheme.
 */
export function resolveTheme(preference: ThemePreference, systemTheme: ResolvedTheme): ResolvedTheme {
	return preference === "system" ? systemTheme : preference;
}

/**
 * Reads the live system theme from `prefers-color-scheme: dark`, defaulting to
 * `light` whenever the browser global is unavailable (SSR) or the media query is
 * unsupported. Never throws.
 *
 * @remarks Implements DESIGN-016 ThemeProvider system-theme probe.
 */
function readSystemTheme(): ResolvedTheme {
	if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
		return "light";
	}
	return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

/**
 * Applies the resolved theme to the {@link resolvedTheme} store and to the document
 * root `data-theme` token so CSS custom properties switch palettes. Safe during SSR
 * where `document` is undefined.
 *
 * @remarks Implements DESIGN-016 ThemeProvider token application.
 */
function applyResolvedTheme(resolved: ResolvedTheme): void {
	resolvedTheme.set(resolved);
	if (typeof document !== "undefined") {
		document.documentElement.dataset.theme = resolved;
	}
}

/**
 * Subscribes to live `prefers-color-scheme` changes exactly once. The handler only
 * recomputes the resolved theme while the active preference is `system`; explicit
 * `light`/`dark` overrides ignore system changes. No-op on SSR and on browsers
 * without `MediaQueryList.addEventListener`.
 *
 * @remarks Implements DESIGN-016 ThemeProvider subscribeToSystemTheme.
 */
function ensureSystemThemeSubscription(): void {
	if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
		return;
	}
	if (systemThemeHandler !== null) {
		return;
	}
	mediaQueryList = window.matchMedia("(prefers-color-scheme: dark)");
	systemThemeHandler = (event: MediaQueryListEvent): void => {
		if (get(themePreference) !== "system") {
			return;
		}
		applyResolvedTheme(event.matches ? "dark" : "light");
	};
	if (typeof mediaQueryList.addEventListener === "function") {
		mediaQueryList.addEventListener("change", systemThemeHandler);
	}
}

/**
 * Initializes theme state from browser storage when browser globals are available.
 * Reads the persisted preference, falls back to `system` for missing, malformed, or
 * invalid values, subscribes to live system-theme changes, and applies the resolved
 * theme. Storage reads that throw (private mode, disabled localStorage) leave the
 * in-memory store on the `system` default without propagating the error. No-op on
 * SSR so server renders keep whatever store value the caller set.
 *
 * @remarks Implements DESIGN-016 ThemeProvider startup initialization and storage-unavailable fallback.
 */
export function initTheme(): void {
	if (typeof window === "undefined") {
		return;
	}

	let stored: string | null = null;
	try {
		stored = window.localStorage.getItem(storageKey);
	} catch {
		// Storage unavailable (private mode, disabled localStorage): keep in-memory default.
		stored = null;
	}
	const preference: ThemePreference =
		stored === "light" || stored === "dark" || stored === "system" ? stored : "system";

	ensureSystemThemeSubscription();
	setThemePreference(preference);
}

/**
 * Persists the user preference and applies the resolved theme to the document.
 * Storage writes that throw (quota exceeded, disabled localStorage) leave the
 * in-memory store updated so the current session keeps the chosen theme without
 * propagating the error. On SSR, resolves against a `light` system default and
 * skips persistence.
 *
 * @remarks Implements DESIGN-016 ThemeProvider preference persistence, token application, and storage-unavailable fallback.
 */
export function setThemePreference(preference: ThemePreference): void {
	themePreference.set(preference);

	if (typeof window === "undefined") {
		applyResolvedTheme(resolveTheme(preference, "light"));
		return;
	}

	const resolved = resolveTheme(preference, readSystemTheme());
	applyResolvedTheme(resolved);

	try {
		window.localStorage.setItem(storageKey, preference);
	} catch {
		// Storage unavailable or quota exceeded; the in-memory store still serves callers.
		return;
	}
}

/**
 * Removes the `prefers-color-scheme` change listener registered by {@link initTheme}.
 * Safe on SSR, safe to call before {@link initTheme}, and safe to call multiple times.
 *
 * @remarks Implements DESIGN-016 ThemeProvider system-theme subscription teardown.
 */
export function cleanupTheme(): void {
	if (systemThemeHandler === null || mediaQueryList === null) {
		return;
	}
	if (typeof mediaQueryList.removeEventListener === "function") {
		mediaQueryList.removeEventListener("change", systemThemeHandler);
	}
	systemThemeHandler = null;
	mediaQueryList = null;
}
