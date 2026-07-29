import { afterEach, expect, test } from "bun:test";
import { get } from "svelte/store";
import {
	PREFERENCES_STORAGE_KEY,
	createDefaultPreferences,
	initPreferences,
	isValidPreferences,
	loadAuthenticatedUnitPreference,
	preferencesStore,
	resetPreferences,
	restoreAnonymousUnitPreference,
	retryUnitPreference,
	setPreferenceDependencies,
	setUnitSystem,
	unitPreferenceStatusStore
} from "./preferences";
import type { ProfileData } from "../api/generated";

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
		resetPreferences();
		return;
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

// Implements DESIGN-001 SettingsPanel authenticated profile hydration verification.
test("authenticated profile replaces but never overwrites the anonymous device preference", async () => {
	const storage = createStorage({ [PREFERENCES_STORAGE_KEY]: JSON.stringify({ unitSystem: "imperial" }) });
	setWindowGlobals(storage);
	initPreferences();

	await loadAuthenticatedUnitPreference("account-a", async () => profile("account-a", "metric"));

	expect(get(preferencesStore).unitSystem).toBe("metric");
	expect(storage.getItem(PREFERENCES_STORAGE_KEY)).toBe(JSON.stringify({ unitSystem: "imperial" }));

	restoreAnonymousUnitPreference();
	expect(get(preferencesStore).unitSystem).toBe("imperial");
});

// Implements DESIGN-008 PreferenceManager confirmed-save verification.
test("authenticated changes update the UI only after the server confirms the profile", async () => {
	let confirm!: (value: ProfileData) => void;
	const response = new Promise<ProfileData>((resolve) => {
		confirm = resolve;
	});
	setPreferenceDependencies({
		fetchCsrfToken: async () => ({ csrfToken: "csrf" }),
		updateProfileSession: async () => response
	});
	await loadAuthenticatedUnitPreference("account-a", async () => profile("account-a", "metric"));

	const saving = setUnitSystem("imperial");
	await Promise.resolve();
	expect(get(preferencesStore).unitSystem).toBe("metric");
	expect(get(unitPreferenceStatusStore)).toMatchObject({ state: "saving", pendingUnitSystem: "imperial" });

	confirm(profile("account-a", "imperial"));
	await saving;
	expect(get(preferencesStore).unitSystem).toBe("imperial");
	expect(get(unitPreferenceStatusStore)).toMatchObject({ state: "ready", userId: "account-a" });
});

// Implements DESIGN-001 SettingsPanel account-switch cancellation and isolation verification.
test("switching accounts aborts stale loads and ignores the prior account result", async () => {
	let firstSignal: AbortSignal | undefined;
	let resolveFirst!: (value: ProfileData) => void;
	const first = loadAuthenticatedUnitPreference("account-a", async (signal) => {
		firstSignal = signal;
		return new Promise<ProfileData>((resolve) => {
			resolveFirst = resolve;
		});
	});

	await loadAuthenticatedUnitPreference("account-b", async () => profile("account-b", "imperial"));
	expect(firstSignal?.aborted).toBe(true);
	resolveFirst(profile("account-a", "metric"));
	await first;

	expect(get(preferencesStore).unitSystem).toBe("imperial");
	expect(get(unitPreferenceStatusStore)).toMatchObject({ state: "ready", userId: "account-b" });
});

// Implements DESIGN-001 SettingsPanel visible recoverable failure verification.
test("failed saves preserve the confirmed value and retry the pending unit", async () => {
	let attempts = 0;
	setPreferenceDependencies({
		fetchCsrfToken: async () => ({ csrfToken: "csrf" }),
		updateProfileSession: async (request) => {
			attempts += 1;
			if (attempts === 1) {
				throw new Error("offline");
			}
			return profile("account-a", request.unitSystem);
		}
	});
	await loadAuthenticatedUnitPreference("account-a", async () => profile("account-a", "metric"));

	await setUnitSystem("imperial");
	expect(get(preferencesStore).unitSystem).toBe("metric");
	expect(get(unitPreferenceStatusStore)).toMatchObject({
		state: "error",
		operation: "save",
		pendingUnitSystem: "imperial"
	});

	await retryUnitPreference();
	expect(get(preferencesStore).unitSystem).toBe("imperial");
	expect(attempts).toBe(2);
});

test("failed account loads cannot mutate or leak into the anonymous device preference", async () => {
	const storage = createStorage({ [PREFERENCES_STORAGE_KEY]: JSON.stringify({ unitSystem: "imperial" }) });
	setWindowGlobals(storage);
	initPreferences();

	await loadAuthenticatedUnitPreference("account-a", async () => {
		throw new Error("offline");
	});
	await setUnitSystem("imperial");

	expect(get(preferencesStore).unitSystem).toBe("metric");
	expect(get(unitPreferenceStatusStore)).toMatchObject({ state: "error", operation: "load" });
	expect(storage.getItem(PREFERENCES_STORAGE_KEY)).toBe(JSON.stringify({ unitSystem: "imperial" }));
});

test("mismatched profiles fail closed for loads and saves", async () => {
	await loadAuthenticatedUnitPreference("account-a", async () => profile("account-b", "imperial"));
	expect(get(unitPreferenceStatusStore)).toMatchObject({ state: "error", operation: "load" });

	await loadAuthenticatedUnitPreference("account-a", async () => profile("account-a", "metric"));
	setPreferenceDependencies({
		fetchCsrfToken: async () => ({ csrfToken: "csrf" }),
		updateProfileSession: async () => profile("account-b", "imperial")
	});
	await setUnitSystem("imperial");
	expect(get(preferencesStore).unitSystem).toBe("metric");
	expect(get(unitPreferenceStatusStore)).toMatchObject({ state: "error", operation: "save" });
});

test("aborted and late load results cannot replace the current account", async () => {
	const aborted = loadAuthenticatedUnitPreference("account-a", (signal) =>
		new Promise<ProfileData>((_resolve, reject) => {
			signal?.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")));
		})
	);
	await loadAuthenticatedUnitPreference("account-b", async () => profile("account-b", "imperial"));
	await aborted;
	expect(get(preferencesStore).unitSystem).toBe("imperial");
});

test("aborted and late saves cannot replace a newer account", async () => {
	let resolveLate!: (value: ProfileData) => void;
	setPreferenceDependencies({
		fetchCsrfToken: async () => ({ csrfToken: "csrf" }),
		updateProfileSession: async () =>
			new Promise<ProfileData>((resolve) => {
				resolveLate = resolve;
			})
	});
	await loadAuthenticatedUnitPreference("account-a", async () => profile("account-a", "metric"));
	const saving = setUnitSystem("imperial");
	await Promise.resolve();
	await loadAuthenticatedUnitPreference("account-b", async () => profile("account-b", "metric"));
	resolveLate(profile("account-a", "imperial"));
	await saving;
	expect(get(preferencesStore).unitSystem).toBe("metric");

	setPreferenceDependencies({
		fetchCsrfToken: async () => ({ csrfToken: "csrf" }),
		updateProfileSession: async (_request, { signal }) =>
			new Promise<ProfileData>((_resolve, reject) => {
				signal?.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")));
			})
	});
	const abortedSave = setUnitSystem("imperial");
	await Promise.resolve();
	restoreAnonymousUnitPreference();
	await abortedSave;
	expect(get(unitPreferenceStatusStore)).toEqual({ authority: "anonymous", state: "ready" });
});

test("retry is a no-op when no recoverable failure exists", async () => {
	await expect(retryUnitPreference()).resolves.toBeUndefined();
});

function profile(userId: string, unitSystem: "metric" | "imperial"): ProfileData {
	return {
		userId,
		displayName: userId,
		unitSystem,
		themePreference: "system",
		requiresUnitRecalculation: false
	};
}

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
