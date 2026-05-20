import { describe, expect, it } from 'bun:test';
import {
  applyTheme,
  createThemeController,
  loadThemePreference,
  normalizeThemePreference,
  resolveTheme,
  saveThemePreference,
  themeStorageKey,
  type ResolvedTheme,
  type ThemePreference,
  type ThemeRoot,
  type ThemeStorage
} from './theme';

describe('ThemeProvider state', () => {
  it('defaults invalid or missing preferences to system', () => {
    expect(normalizeThemePreference(null)).toBe('system');
    expect(normalizeThemePreference('neon')).toBe('system');
    expect(normalizeThemePreference('dark')).toBe('dark');
  });

  it('resolves system and explicit theme preferences', () => {
    expect(resolveTheme('system', 'dark')).toBe('dark');
    expect(resolveTheme('system', 'light')).toBe('light');
    expect(resolveTheme('light', 'dark')).toBe('light');
    expect(resolveTheme('dark', 'light')).toBe('dark');
  });

  it('loads and persists theme preference', () => {
    const storage = new MemoryStorage();
    expect(loadThemePreference(storage)).toBe('system');

    expect(saveThemePreference('dark', storage)).toBe(true);
    expect(storage.values[themeStorageKey]).toBe('dark');
    expect(loadThemePreference(storage)).toBe('dark');
  });

  it('applies light and dark CSS variables to the root', () => {
    const root = new MemoryRoot();
    applyTheme('dark', root);

    expect(root.dataset.theme).toBe('dark');
    expect(root.classes.has('dark')).toBe(true);
    expect(root.properties['--color-bg-primary']).toBe('#0A0F0A');
    expect(root.properties['--color-text-primary']).toBe('#F3F4F6');

    applyTheme('light', root);
    expect(root.dataset.theme).toBe('light');
    expect(root.classes.has('dark')).toBe(false);
    expect(root.properties['--color-bg-primary']).toBe('#F7FCF7');
  });

  it('initializes from system preference when no override exists', () => {
    const root = new MemoryRoot();
    const controller = createThemeController({
      storage: new MemoryStorage(),
      root,
      matchMedia: matchMediaFor('dark')
    });

    expect(controller.getState()).toEqual({ preference: 'system', resolved: 'dark', systemTheme: 'dark' });
    expect(root.dataset.theme).toBe('dark');
  });

  it('persists explicit user override and syncs it through the API hook', async () => {
    const storage = new MemoryStorage();
    const root = new MemoryRoot();
    const synced: ThemePreference[] = [];
    const controller = createThemeController({
      storage,
      root,
      matchMedia: matchMediaFor('light'),
      syncPreference: (preference) => {
        synced.push(preference);
      }
    });

    const state = await controller.setThemePreference('dark');

    expect(state.preference).toBe('dark');
    expect(state.resolved).toBe('dark');
    expect(storage.values[themeStorageKey]).toBe('dark');
    expect(root.dataset.theme).toBe('dark');
    expect(synced).toEqual(['dark']);
  });

  it('updates resolved theme on system changes only when preference is system', async () => {
    const controller = createThemeController({
      storage: new MemoryStorage(),
      root: new MemoryRoot(),
      matchMedia: matchMediaFor('light')
    });

    expect(controller.handleSystemTheme('dark').resolved).toBe('dark');
    await controller.setThemePreference('light');
    expect(controller.handleSystemTheme('dark').resolved).toBe('light');
  });
});

class MemoryStorage implements ThemeStorage {
  values: Record<string, string> = {};

  getItem(key: string): string | null {
    return this.values[key] ?? null;
  }

  setItem(key: string, value: string): void {
    this.values[key] = value;
  }
}

class MemoryRoot implements ThemeRoot {
  dataset: Record<string, string> = {};
  classes = new Set<string>();
  properties: Record<string, string> = {};
  classList = {
    toggle: (name: string, force?: boolean) => {
      const shouldHave = force ?? !this.classes.has(name);
      if (shouldHave) {
        this.classes.add(name);
      } else {
        this.classes.delete(name);
      }
      return shouldHave;
    }
  };
  style = {
    setProperty: (name: string, value: string) => {
      this.properties[name] = value;
    }
  };
}

function matchMediaFor(theme: ResolvedTheme) {
  return () => ({ matches: theme === 'dark' });
}
