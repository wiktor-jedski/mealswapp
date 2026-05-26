import { afterEach, expect, test } from "bun:test";
import { get } from "svelte/store";
import { initTheme, resolvedTheme, resolveTheme, setThemePreference, themePreference } from "./theme";

const originalWindow = globalThis.window;
const originalDocument = globalThis.document;

afterEach(() => {
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

function createLocalStorage(initialValue: string) {
  const storage = new Map<string, string>([["mealswapp.theme", initialValue]]);
  return {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => storage.set(key, value)
  };
}

function setBrowserGlobals(options: { darkMode: boolean; localStorage: ReturnType<typeof createLocalStorage> }) {
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: {
      localStorage: options.localStorage,
      matchMedia: () => ({ matches: options.darkMode })
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
