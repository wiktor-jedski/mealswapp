import { afterEach, expect, test } from "bun:test";
import { get } from "svelte/store";
import {
	PREFERENCES_STORAGE_KEY,
	createDefaultPreferences,
	initPreferences,
	isValidPreferences,
	preferencesStore,
	resetPreferences,
	setUnitSystem
} from "./preferences";

// Implements DESIGN-001 SidebarComponent unit preference persistence and restore verification.

const originalWindow = globalThis.window;

afterEach(() => {
	if (originalWindow === undefined) {
		delete (globalThis as { window?: unknown }).window;
	} else {
		Object.defineProperty(globalThis, "window", {
			configurable: true,
			value: originalWindow
		});
	}
	try {
		if (typeof window !== "undefined") {
			window.localStorage.removeItem(PREFERENCES_STORAGE_KEY);
		}
	} catch {
		// ignore
	}
	resetPreferences();
});

// Implements DESIGN-001 SidebarComponent default unit preference verification.
test("createDefaultPreferences defaults to metric units", () => {
	expect(createDefaultPreferences()).toEqual({ unitSystem: "metric" });
});

// Implements DESIGN-001 SidebarComponent initial unit preference store value verification.
test("preferencesStore starts with metric unit system", () => {
	expect(get(preferencesStore).unitSystem).toBe("metric");
});

// Implements DESIGN-001 LocalStorageManager preferences schema validation verification.
test("isValidPreferences accepts metric and imperial and rejects everything else", () => {
	expect(isValidPreferences({ unitSystem: "metric" })).toBe(true);
	expect(isValidPreferences({ unitSystem: "imperial" })).toBe(true);
	expect(isValidPreferences({ unitSystem: "bogus" })).toBe(false);
	expect(isValidPreferences({})).toBe(false);
	expect(isValidPreferences(null)).toBe(false);
	expect(isValidPreferences("metric")).toBe(false);
	expect(isValidPreferences({ unitSystem: "metric", extra: 1 })).toBe(true);
});

// Implements DESIGN-001 LocalStorageManager settings persistence verification.
test("setUnitSystem updates the store and persists to localStorage", () => {
	const storage = createStorage();
	setWindowGlobals(storage);

	setUnitSystem("imperial");

	expect(get(preferencesStore).unitSystem).toBe("imperial");
	expect(storage.getItem(PREFERENCES_STORAGE_KEY)).toBe(JSON.stringify({ unitSystem: "imperial" }));
});

// Implements DESIGN-001 LocalStorageManager settings restore verification.
test("initPreferences restores a valid stored unit system", () => {
	const storage = createStorage({ [PREFERENCES_STORAGE_KEY]: JSON.stringify({ unitSystem: "imperial" }) });
	setWindowGlobals(storage);

	initPreferences();

	expect(get(preferencesStore).unitSystem).toBe("imperial");
});

// Implements DESIGN-001 LocalStorageManager invalid stored settings fallback verification.
test("initPreferences falls back to metric for an invalid stored unit system", () => {
	const storage = createStorage({ [PREFERENCES_STORAGE_KEY]: JSON.stringify({ unitSystem: "bogus" }) });
	setWindowGlobals(storage);

	initPreferences();

	expect(get(preferencesStore).unitSystem).toBe("metric");
});

// Implements DESIGN-001 LocalStorageManager malformed JSON fallback verification.
test("initPreferences falls back to metric for malformed JSON", () => {
	const storage = createStorage({ [PREFERENCES_STORAGE_KEY]: "{not valid json" });
	setWindowGlobals(storage);

	initPreferences();

	expect(get(preferencesStore).unitSystem).toBe("metric");
});

// Implements DESIGN-001 LocalStorageManager missing settings fallback verification.
test("initPreferences falls back to metric when nothing is stored", () => {
	const storage = createStorage();
	setWindowGlobals(storage);

	initPreferences();

	expect(get(preferencesStore).unitSystem).toBe("metric");
});

// Implements DESIGN-001 LocalStorageManager storage-unavailable fallback verification.
test("initPreferences falls back to metric when localStorage getItem throws", () => {
	const throwingStorage = {
		getItem(): string | null {
			throw new Error("denied");
		},
		setItem(): void {
			throw new Error("denied");
		},
		removeItem(): void {
			throw new Error("denied");
		}
	};
	setWindowGlobals(throwingStorage);

	expect(() => initPreferences()).not.toThrow();
	expect(get(preferencesStore).unitSystem).toBe("metric");
});

// Implements DESIGN-001 LocalStorageManager setItem failure tolerance verification.
test("setUnitSystem still updates the store when localStorage setItem throws", () => {
	const throwingStorage = {
		getItem(): string | null {
			return null;
		},
		setItem(): void {
			throw new Error("quota exceeded");
		},
		removeItem(): void {
			throw new Error("denied");
		}
	};
	setWindowGlobals(throwingStorage);

	expect(() => setUnitSystem("imperial")).not.toThrow();
	expect(get(preferencesStore).unitSystem).toBe("imperial");
});

// Implements DESIGN-001 LocalStorageManager SSR initialization fallback verification.
test("initPreferences is safe and defaults to metric when window is undefined", () => {
	delete (globalThis as { window?: unknown }).window;

	expect(() => initPreferences()).not.toThrow();
	expect(get(preferencesStore).unitSystem).toBe("metric");
});

// Implements DESIGN-001 LocalStorageManager SSR persistence tolerance verification.
test("setUnitSystem is safe and updates the store when window is undefined", () => {
	delete (globalThis as { window?: unknown }).window;

	expect(() => setUnitSystem("imperial")).not.toThrow();
	expect(get(preferencesStore).unitSystem).toBe("imperial");
});

// Implements DESIGN-001 SidebarComponent persistence round-trip verification.
test("unit changes persist across initPreferences calls backed by the same storage", () => {
	const storage = createStorage();
	setWindowGlobals(storage);

	setUnitSystem("imperial");
	resetPreferences();
	expect(get(preferencesStore).unitSystem).toBe("metric");

	initPreferences();
	expect(get(preferencesStore).unitSystem).toBe("imperial");
});

function createStorage(initial: Record<string, string> = {}): MapStorage {
	return new MapStorage(initial);
}

class MapStorage {
	private data: Map<string, string>;
	constructor(initial: Record<string, string> = {}) {
		this.data = new Map(Object.entries(initial));
	}
	getItem(key: string): string | null {
		return this.data.has(key) ? (this.data.get(key) as string) : null;
	}
	setItem(key: string, value: string): void {
		this.data.set(key, value);
	}
	removeItem(key: string): void {
		this.data.delete(key);
	}
}

function setWindowGlobals(storage: { getItem(k: string): string | null; setItem(k: string, v: string): void; removeItem(k: string): void }): void {
	Object.defineProperty(globalThis, "window", {
		configurable: true,
		value: { localStorage: storage }
	});
}
