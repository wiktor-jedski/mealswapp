import { describe, expect, test } from "bun:test";
import { get } from "svelte/store";
import type { KeyValueStorage } from "../cache/search-lru";
import { createSettingsStore, defaultSearchSettings } from "./settings";

class MemoryStorage implements KeyValueStorage {
  values = new Map<string, string>();
  getItem(key: string) { return this.values.get(key) ?? null; }
  setItem(key: string, value: string) { this.values.set(key, value); }
  removeItem(key: string) { this.values.delete(key); }
}

// Implements DESIGN-001 SettingsPanel persistence verification.
describe("settings store", () => {
  test("defaults every macro to enabled and metric units", () => expect(defaultSearchSettings()).toEqual({ unitSystem: "metric", enabledMacros: { protein: true, carbohydrate: true, fat: true } }));

  test("persists macro and unit changes and restores them", () => {
    const storage = new MemoryStorage();
    const settings = createSettingsStore(storage);
    settings.setMacro("fat", false);
    settings.setUnitSystem("imperial");
    expect(get(createSettingsStore(storage))).toEqual({ unitSystem: "imperial", enabledMacros: { protein: true, carbohydrate: true, fat: false } });
  });

  test("invalid and unavailable storage falls back safely", () => {
    const invalid = new MemoryStorage();
    invalid.setItem("mealswapp.search-settings.v1", JSON.stringify({ unitSystem: "stones" }));
    expect(get(createSettingsStore(invalid))).toEqual(defaultSearchSettings());
    const throwing: KeyValueStorage = { getItem: () => { throw new Error("denied"); }, setItem: () => { throw new Error("denied"); }, removeItem: () => {} };
    const settings = createSettingsStore(throwing);
    settings.setMacro("protein", false);
    expect(get(settings).enabledMacros.protein).toBe(false);
  });

  test("default storage lookup is server-safe", () => expect(get(createSettingsStore())).toEqual(defaultSearchSettings()));
});
