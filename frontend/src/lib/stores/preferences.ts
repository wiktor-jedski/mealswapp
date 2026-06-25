import { writable } from "svelte/store";

// Implements DESIGN-001 SidebarComponent unit-system preference persistence independent of server availability.

/**
 * Supported measurement unit systems for the sidebar unit preference row.
 *
 * @remarks Implements DESIGN-001 SidebarComponent unit preference controls.
 */
export type UnitSystem = "metric" | "imperial";

/**
 * Locally persisted account-display preferences owned by the sidebar unit preference row.
 *
 * @remarks Implements DESIGN-001 LocalStorageManager settings persistence.
 */
export interface SearchPreferences {
  unitSystem: UnitSystem;
}

/**
 * localStorage key under which the {@link SearchPreferences} blob is persisted as JSON.
 *
 * @remarks Implements DESIGN-001 LocalStorageManager client persistence key.
 */
export const PREFERENCES_STORAGE_KEY = "mealswapp.preferences";

/**
 * Returns the default preferences: metric units. Used as the fallback whenever stored
 * preferences are missing, malformed, or fail validation.
 *
 * @remarks Implements DESIGN-001 SidebarComponent default unit preference.
 */
export function createDefaultPreferences(): SearchPreferences {
  return { unitSystem: "metric" };
}

/**
 * Svelte writable store holding the current account-display preferences.
 *
 * @remarks Implements DESIGN-001 SidebarComponent unit preference store initialization.
 */
export const preferencesStore = writable<SearchPreferences>(createDefaultPreferences());

/**
 * Validates that an unknown value is a well-formed {@link SearchPreferences} object.
 *
 * @remarks Implements DESIGN-001 LocalStorageManager schema validation for preferences.
 */
export function isValidPreferences(value: unknown): value is SearchPreferences {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const candidate = value as { unitSystem?: unknown };
  return candidate.unitSystem === "metric" || candidate.unitSystem === "imperial";
}

/**
 * Loads preferences from localStorage and updates the store, falling back to defaults
 * when storage is unavailable, missing, malformed, or holds an invalid unit system.
 * Safe to call during SSR when `window` is undefined.
 *
 * @remarks Implements DESIGN-001 LocalStorageManager loadSettings and storage-unavailable fallback.
 */
export function initPreferences(): void {
  if (typeof window === "undefined") {
    preferencesStore.set(createDefaultPreferences());
    return;
  }

  let raw: string | null;
  try {
    raw = window.localStorage.getItem(PREFERENCES_STORAGE_KEY);
  } catch {
    preferencesStore.set(createDefaultPreferences());
    return;
  }

  if (raw === null) {
    preferencesStore.set(createDefaultPreferences());
    return;
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    preferencesStore.set(createDefaultPreferences());
    return;
  }

  preferencesStore.set(isValidPreferences(parsed) ? parsed : createDefaultPreferences());
}

/**
 * Updates the active unit system in the store and persists it to localStorage. Safe
 * during SSR and when localStorage throws (e.g. quota exceeded or disabled).
 *
 * @remarks Implements DESIGN-001 SidebarComponent unit preference persistence.
 */
export function setUnitSystem(unit: UnitSystem): void {
  preferencesStore.update((prefs) => ({ ...prefs, unitSystem: unit }));

  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(
      PREFERENCES_STORAGE_KEY,
      JSON.stringify({ unitSystem: unit } satisfies SearchPreferences)
    );
  } catch {
    // Storage unavailable or quota exceeded; the in-memory store still serves callers.
  }
}

/**
 * Resets the preferences store to defaults without touching localStorage. Used by tests.
 *
 * @remarks Implements DESIGN-001 SidebarComponent default restoration.
 */
export function resetPreferences(): void {
  preferencesStore.set(createDefaultPreferences());
}
