import { writable, type Writable } from "svelte/store";
import type { KeyValueStorage } from "../cache/search-lru";

const SETTINGS_KEY = "mealswapp.search-settings.v1";

// Implements DESIGN-001 SettingsPanel typed local preferences.
export interface SearchSettings {
  unitSystem: "metric" | "imperial";
  enabledMacros: { protein: boolean; carbohydrate: boolean; fat: boolean };
}

export interface SettingsStore extends Writable<SearchSettings> {
  setUnitSystem(value: SearchSettings["unitSystem"]): void;
  setMacro(name: keyof SearchSettings["enabledMacros"], enabled: boolean): void;
}

export function defaultSearchSettings(): SearchSettings {
  return { unitSystem: "metric", enabledMacros: { protein: true, carbohydrate: true, fat: true } };
}

// Implements DESIGN-001 SettingsPanel local persistence with storage fallback.
export function createSettingsStore(storage: KeyValueStorage | null = safeLocalStorage()): SettingsStore {
  const store = writable(loadSettings(storage)) as SettingsStore;
  const persist = (settings: SearchSettings) => {
    try { storage?.setItem(SETTINGS_KEY, JSON.stringify(settings)); } catch { /* In-memory settings remain active. */ }
  };
  store.setUnitSystem = (unitSystem) => store.update((settings) => {
    const next = { ...settings, unitSystem };
    persist(next);
    return next;
  });
  store.setMacro = (name, enabled) => store.update((settings) => {
    const next = { ...settings, enabledMacros: { ...settings.enabledMacros, [name]: enabled } };
    persist(next);
    return next;
  });
  return store;
}

function loadSettings(storage: KeyValueStorage | null): SearchSettings {
  try {
    const raw = storage?.getItem(SETTINGS_KEY);
    if (!raw) return defaultSearchSettings();
    const parsed: unknown = JSON.parse(raw);
    return isSearchSettings(parsed) ? parsed : defaultSearchSettings();
  } catch {
    return defaultSearchSettings();
  }
}

function isSearchSettings(value: unknown): value is SearchSettings {
  if (typeof value !== "object" || value === null) return false;
  const candidate = value as Record<string, unknown>;
  const macros = candidate.enabledMacros;
  return (candidate.unitSystem === "metric" || candidate.unitSystem === "imperial") && typeof macros === "object" && macros !== null &&
    typeof (macros as Record<string, unknown>).protein === "boolean" &&
    typeof (macros as Record<string, unknown>).carbohydrate === "boolean" &&
    typeof (macros as Record<string, unknown>).fat === "boolean";
}

function safeLocalStorage(): KeyValueStorage | null {
  try { return typeof window === "undefined" ? null : window.localStorage; } catch { return null; }
}
