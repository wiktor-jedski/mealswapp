import { afterEach, expect, test } from "bun:test";
import { get } from "svelte/store";
import {
	cleanupTheme,
	initTheme,
	resolvedTheme,
	resolveTheme,
	setThemePreference,
	themePreference
} from "./theme";

// Implements DESIGN-016 ThemeProvider system/light/dark resolution, sidebar selection,
// local persistence, live system-theme subscription, listener cleanup, and
// storage-unavailable fallback verification.

const originalWindow = globalThis.window;
const originalDocument = globalThis.document;

afterEach(() => {
	cleanupTheme();
	Object.defineProperty(globalThis, "window", {
		configurable: true,
		value: originalWindow
	});
	Object.defineProperty(globalThis, "document", {
		configurable: true,
		value: originalDocument
	});
	themePreference.set("system");
	resolvedTheme.set("light");
});

// Implements DESIGN-016 ThemeProvider system preference verification.
test("resolveTheme uses system value for system preference", () => {
	expect(resolveTheme("system", "dark")).toBe("dark");
	expect(resolveTheme("system", "light")).toBe("light");
});

// Implements DESIGN-016 ThemeProvider explicit preference verification.
test("resolveTheme honors explicit preference", () => {
	expect(resolveTheme("light", "dark")).toBe("light");
	expect(resolveTheme("dark", "light")).toBe("dark");
});

// Implements DESIGN-016 ThemeProvider server-side fallback verification.
test("setThemePreference resolves without browser globals", () => {
	Object.defineProperty(globalThis, "window", {
		configurable: true,
		value: undefined
	});

	setThemePreference("system");

	expect(get(themePreference)).toBe("system");
	expect(get(resolvedTheme)).toBe("light");
});

// Implements DESIGN-016 ThemeProvider server-side initialization verification.
test("initTheme does nothing without browser globals", () => {
	themePreference.set("dark");
	Object.defineProperty(globalThis, "window", {
		configurable: true,
		value: undefined
	});

	initTheme();

	expect(get(themePreference)).toBe("dark");
});

// Implements DESIGN-016 ThemeProvider first-load system preference verification.
test("initTheme defaults to system and resolves to the live system theme on first load", () => {
	const localStorage = createLocalStorage("");
	setBrowserGlobals({ darkMode: true, localStorage });

	initTheme();

	expect(get(themePreference)).toBe("system");
	expect(get(resolvedTheme)).toBe("dark");
	expect(globalThis.document.documentElement.dataset.theme).toBe("dark");
});

// Implements DESIGN-016 ThemeProvider invalid stored preference verification.
test("initTheme falls back to system for invalid stored preference", () => {
	const localStorage = createLocalStorage("invalid");
	setBrowserGlobals({ darkMode: true, localStorage });

	initTheme();

	expect(get(themePreference)).toBe("system");
	expect(get(resolvedTheme)).toBe("dark");
	expect(globalThis.document.documentElement.dataset.theme).toBe("dark");
	expect(localStorage.getItem("mealswapp.theme")).toBe("system");
});

// Implements DESIGN-016 ThemeProvider stored explicit preference verification.
test("initTheme applies stored explicit preference", () => {
	const localStorage = createLocalStorage("light");
	setBrowserGlobals({ darkMode: true, localStorage });

	initTheme();

	expect(get(themePreference)).toBe("light");
	expect(get(resolvedTheme)).toBe("light");
	expect(globalThis.document.documentElement.dataset.theme).toBe("light");
});

// Implements DESIGN-016 ThemeProvider explicit override updates and persists verification.
test("setThemePreference updates resolvedTheme and persists the choice", () => {
	const localStorage = createLocalStorage("system");
	setBrowserGlobals({ darkMode: true, localStorage });

	setThemePreference("light");

	expect(get(themePreference)).toBe("light");
	expect(get(resolvedTheme)).toBe("light");
	expect(globalThis.document.documentElement.dataset.theme).toBe("light");
	expect(localStorage.getItem("mealswapp.theme")).toBe("light");

	setThemePreference("dark");

	expect(get(resolvedTheme)).toBe("dark");
	expect(globalThis.document.documentElement.dataset.theme).toBe("dark");
	expect(localStorage.getItem("mealswapp.theme")).toBe("dark");
});

// Implements DESIGN-016 ThemeProvider persistence across reload verification.
test("initTheme restores the persisted preference across a simulated reload", () => {
	const localStorage = createLocalStorage("dark");
	setBrowserGlobals({ darkMode: false, localStorage });

	initTheme();
	expect(get(themePreference)).toBe("dark");
	expect(get(resolvedTheme)).toBe("dark");

	// Simulate a reload: reset stores, re-init from the same storage.
	themePreference.set("system");
	resolvedTheme.set("light");
	cleanupTheme();

	initTheme();

	expect(get(themePreference)).toBe("dark");
	expect(get(resolvedTheme)).toBe("dark");
});

// Implements DESIGN-016 ThemeProvider live system-theme subscription in system mode verification.
test("system preference recomputes resolvedTheme when the system theme changes live", () => {
	const mql = createMediaQueryList(false);
	const localStorage = createLocalStorage("");
	setBrowserGlobals({ darkMode: false, localStorage, mql });

	initTheme();
	expect(get(resolvedTheme)).toBe("light");

	mql.setMatches(true);

	expect(get(resolvedTheme)).toBe("dark");
	expect(globalThis.document.documentElement.dataset.theme).toBe("dark");
});

// Implements DESIGN-016 ThemeProvider explicit override ignores live system changes verification.
test("explicit light/dark overrides ignore live system-theme changes", () => {
	const mql = createMediaQueryList(false);
	const localStorage = createLocalStorage("");
	setBrowserGlobals({ darkMode: false, localStorage, mql });

	initTheme();
	setThemePreference("light");
	expect(get(resolvedTheme)).toBe("light");

	mql.setMatches(true);

	expect(get(resolvedTheme)).toBe("light");
	expect(globalThis.document.documentElement.dataset.theme).toBe("light");
});

// Implements DESIGN-016 ThemeProvider listener cleanup verification.
test("cleanupTheme removes the same change handler that initTheme registered", () => {
	const mql = createMediaQueryList(false);
	const localStorage = createLocalStorage("");
	setBrowserGlobals({ darkMode: false, localStorage, mql });

	initTheme();

	const registered = mql.addEventListenerCalls.find((c) => c.type === "change")?.listener;
	expect(registered).toBeDefined();

	cleanupTheme();

	const removed = mql.removeEventListenerCalls.find((c) => c.type === "change")?.listener;
	expect(removed).toBe(registered);
	expect(mql.removeEventListenerCalls).toHaveLength(1);
});

// Implements DESIGN-016 ThemeProvider listener cleanup stops live updates verification.
test("cleanupTheme stops further system-theme updates from reaching the store", () => {
	const mql = createMediaQueryList(false);
	const localStorage = createLocalStorage("");
	setBrowserGlobals({ darkMode: false, localStorage, mql });

	initTheme();
	cleanupTheme();

	mql.setMatches(true);

	expect(get(resolvedTheme)).toBe("light");
});

// Implements DESIGN-016 ThemeProvider cleanup idempotency verification.
test("cleanupTheme is safe to call before initTheme and multiple times", () => {
	const mql = createMediaQueryList(false);
	const localStorage = createLocalStorage("");
	setBrowserGlobals({ darkMode: false, localStorage, mql });

	expect(() => cleanupTheme()).not.toThrow();

	initTheme();
	cleanupTheme();
	mql.removeEventListenerCalls.length = 0;

	expect(() => cleanupTheme()).not.toThrow();
	expect(mql.removeEventListenerCalls).toHaveLength(0);
});

// Implements DESIGN-016 ThemeProvider storage-unavailable read fallback verification.
test("initTheme falls back to system when localStorage getItem throws", () => {
	const throwingStorage = createThrowingStorage();
	setBrowserGlobals({ darkMode: true, localStorage: throwingStorage });

	expect(() => initTheme()).not.toThrow();
	expect(get(themePreference)).toBe("system");
	expect(get(resolvedTheme)).toBe("dark");
});

// Implements DESIGN-016 ThemeProvider storage-unavailable write fallback verification.
test("setThemePreference still updates the store when localStorage setItem throws", () => {
	const throwingStorage = createThrowingStorage();
	setBrowserGlobals({ darkMode: true, localStorage: throwingStorage });

	expect(() => setThemePreference("light")).not.toThrow();
	expect(get(themePreference)).toBe("light");
	expect(get(resolvedTheme)).toBe("light");
	expect(globalThis.document.documentElement.dataset.theme).toBe("light");
});

// Implements DESIGN-016 ThemeProvider media-query-list without addEventListener safety verification.
test("initTheme is safe when matchMedia returns a legacy media query list without addEventListener", () => {
	const localStorage = createLocalStorage("");
	setBrowserGlobals({ darkMode: true, localStorage, mql: createLegacyMediaQueryList(true) });

	expect(() => initTheme()).not.toThrow();
	expect(get(themePreference)).toBe("system");
	expect(get(resolvedTheme)).toBe("dark");
});

// Implements DESIGN-016 ThemeProvider subscribe-exactly-once verification.
test("initTheme subscribes to system theme exactly once across repeated calls without cleanup", () => {
	const mql = createMediaQueryList(false);
	const localStorage = createLocalStorage("");
	setBrowserGlobals({ darkMode: false, localStorage, mql });

	initTheme();
	const initialListenerCount = mql.addEventListenerCalls.filter((c) => c.type === "change").length;
	expect(initialListenerCount).toBe(1);

	// Calling initTheme again without cleanup must not register a second change listener.
	initTheme();
	const repeatedListenerCount = mql.addEventListenerCalls.filter((c) => c.type === "change").length;
	expect(repeatedListenerCount).toBe(1);
});

// Implements DESIGN-016 ThemeProvider cleanup SSR safety verification.
test("cleanupTheme is safe when window is undefined", () => {
	Object.defineProperty(globalThis, "window", {
		configurable: true,
		value: undefined
	});

	expect(() => cleanupTheme()).not.toThrow();
});

// Implements DESIGN-016 ThemeProvider matchMedia-unavailable read fallback verification.
test("setThemePreference resolves to light when window exists but matchMedia is not a function", () => {
	const localStorage = createLocalStorage("");
	Object.defineProperty(globalThis, "window", {
		configurable: true,
		value: {
			localStorage,
			matchMedia: undefined as unknown as typeof window.matchMedia
		}
	});
	Object.defineProperty(globalThis, "document", {
		configurable: true,
		value: {
			documentElement: {
				dataset: {} as Record<string, string>
			}
		}
	});

	expect(() => setThemePreference("system")).not.toThrow();
	expect(get(themePreference)).toBe("system");
	expect(get(resolvedTheme)).toBe("light");
	expect(globalThis.document.documentElement.dataset.theme).toBe("light");
});

// Implements DESIGN-016 ThemeProvider matchMedia-unavailable subscription safety verification.
test("initTheme skips system subscription and applies stored preference when matchMedia is not a function", () => {
	const localStorage = createLocalStorage("dark");
	Object.defineProperty(globalThis, "window", {
		configurable: true,
		value: {
			localStorage,
			matchMedia: undefined as unknown as typeof window.matchMedia
		}
	});
	Object.defineProperty(globalThis, "document", {
		configurable: true,
		value: {
			documentElement: {
				dataset: {} as Record<string, string>
			}
		}
	});

	expect(() => initTheme()).not.toThrow();
	expect(get(themePreference)).toBe("dark");
	expect(get(resolvedTheme)).toBe("dark");
	expect(globalThis.document.documentElement.dataset.theme).toBe("dark");
});

function createLocalStorage(initialValue: string): {
	getItem: (key: string) => string | null;
	setItem: (key: string, value: string) => void;
} {
	const storage = new Map<string, string>(
		initialValue.length > 0 ? [["mealswapp.theme", initialValue]] : []
	);
	return {
		getItem: (key) => storage.get(key) ?? null,
		setItem: (key, value) => {
			storage.set(key, value);
		}
	};
}

function createThrowingStorage(): {
	getItem: (key: string) => string | null;
	setItem: (key: string, value: string) => void;
} {
	return {
		getItem: () => {
			throw new Error("denied");
		},
		setItem: () => {
			throw new Error("quota exceeded");
		}
	};
}

interface MediaQueryListLike {
	matches: boolean;
	media: string;
	onchange: ((this: MediaQueryList, ev: MediaQueryListEvent) => unknown) | null;
	addEventListener: MediaQueryList["addEventListener"];
	removeEventListener: MediaQueryList["removeEventListener"];
	dispatchEvent: MediaQueryList["dispatchEvent"];
}

function createMediaQueryList(initialMatches: boolean): MediaQueryListLike & {
	addEventListenerCalls: Array<{ type: string; listener: EventListenerOrEventListenerObject }>;
	removeEventListenerCalls: Array<{ type: string; listener: EventListenerOrEventListenerObject }>;
	setMatches: (matches: boolean) => void;
} {
	const calls = {
		addEventListenerCalls: [] as Array<{ type: string; listener: EventListenerOrEventListenerObject }>,
		removeEventListenerCalls: [] as Array<{ type: string; listener: EventListenerOrEventListenerObject }>
	};
	const listeners = new Map<string, Set<EventListenerOrEventListenerObject>>();
	let matches = initialMatches;
	const target: MediaQueryListLike = {
		get matches() {
			return matches;
		},
		media: "(prefers-color-scheme: dark)",
		onchange: null,
		addEventListener(type, listener) {
			calls.addEventListenerCalls.push({ type, listener });
			const set = listeners.get(type) ?? new Set();
			set.add(listener);
			listeners.set(type, set);
		},
		removeEventListener(type, listener) {
			calls.removeEventListenerCalls.push({ type, listener });
			listeners.get(type)?.delete(listener);
		},
		dispatchEvent(event) {
			const set = listeners.get(event.type);
			if (set) {
				for (const listener of set) {
					(listener as EventListener).call(target, event);
				}
			}
			return true;
		}
	};
	return {
		...target,
		...calls,
		setMatches(next: boolean) {
			matches = next;
			const event = new Event("change") as MediaQueryListEvent;
			Object.defineProperty(event, "matches", { configurable: true, value: next });
			target.dispatchEvent(event);
		}
	};
}

function createLegacyMediaQueryList(initialMatches: boolean): MediaQueryListLike {
	return {
		matches: initialMatches,
		media: "(prefers-color-scheme: dark)",
		onchange: null,
		addEventListener: undefined as unknown as MediaQueryList["addEventListener"],
		removeEventListener: undefined as unknown as MediaQueryList["removeEventListener"],
		dispatchEvent: () => true
	};
}

function setBrowserGlobals(options: {
	darkMode: boolean;
	localStorage: ReturnType<typeof createLocalStorage> | ReturnType<typeof createThrowingStorage>;
	mql?: ReturnType<typeof createMediaQueryList> | MediaQueryListLike;
}): void {
	const mql = options.mql ?? createMediaQueryList(options.darkMode);
	Object.defineProperty(globalThis, "window", {
		configurable: true,
		value: {
			localStorage: options.localStorage,
			matchMedia: () => mql
		}
	});
	Object.defineProperty(globalThis, "document", {
		configurable: true,
		value: {
			documentElement: {
				dataset: {} as Record<string, string>
			}
		}
	});
}
