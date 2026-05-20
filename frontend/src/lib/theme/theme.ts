export type ThemePreference = 'system' | 'light' | 'dark';
export type ResolvedTheme = 'light' | 'dark';

export interface ThemeState {
  preference: ThemePreference;
  resolved: ResolvedTheme;
  systemTheme: ResolvedTheme;
}

export interface ColorPalette {
  bgPrimary: string;
  bgSurface: string;
  colorPrimary: string;
  colorSecondary: string;
  colorAccent: string;
  colorError: string;
  textPrimary: string;
  textMuted: string;
}

export interface ThemeStorage {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
}

export interface ThemeRoot {
  dataset: Record<string, string | undefined>;
  classList: {
    toggle(name: string, force?: boolean): boolean;
  };
  style: {
    setProperty(name: string, value: string): void;
  };
}

export interface ThemeControllerOptions {
  storage?: ThemeStorage;
  root?: ThemeRoot;
  matchMedia?: (query: string) => MediaQueryListLike;
  syncPreference?: (preference: ThemePreference) => Promise<void> | void;
}

export interface MediaQueryListLike {
  matches: boolean;
  addEventListener?: (event: 'change', callback: (event: { matches: boolean }) => void) => void;
  removeEventListener?: (event: 'change', callback: (event: { matches: boolean }) => void) => void;
}

export const themeStorageKey = 'mealswapp.themePreference';

export const palettes: Record<ResolvedTheme, ColorPalette> = {
  light: {
    bgPrimary: '#F7FCF7',
    bgSurface: '#FFFFFF',
    colorPrimary: '#166534',
    colorSecondary: '#DCFCE7',
    colorAccent: '#F97316',
    colorError: '#DC2626',
    textPrimary: '#111827',
    textMuted: '#4B5563'
  },
  dark: {
    bgPrimary: '#0A0F0A',
    bgSurface: '#161D16',
    colorPrimary: '#4ADE80',
    colorSecondary: '#14532D',
    colorAccent: '#FFB86C',
    colorError: '#F87171',
    textPrimary: '#F3F4F6',
    textMuted: '#D1D5DB'
  }
};

export function resolveTheme(preference: ThemePreference, systemTheme: ResolvedTheme): ResolvedTheme {
  return preference === 'system' ? systemTheme : preference;
}

export function getPalette(theme: ResolvedTheme): ColorPalette {
  return palettes[theme];
}

export function normalizeThemePreference(value: string | null | undefined): ThemePreference {
  if (value === 'light' || value === 'dark' || value === 'system') {
    return value;
  }
  return 'system';
}

export function getSystemTheme(matchMediaFn = defaultMatchMedia): ResolvedTheme {
  return matchMediaFn('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export function loadThemePreference(storage = defaultStorage()): ThemePreference {
  if (!storage) {
    return 'system';
  }
  try {
    return normalizeThemePreference(storage.getItem(themeStorageKey));
  } catch {
    return 'system';
  }
}

export function saveThemePreference(preference: ThemePreference, storage = defaultStorage()): boolean {
  if (!storage) {
    return false;
  }
  try {
    storage.setItem(themeStorageKey, preference);
    return true;
  } catch {
    return false;
  }
}

export function applyTheme(theme: ResolvedTheme, root = defaultRoot()): void {
  if (!root) {
    return;
  }
  const palette = getPalette(theme);
  root.dataset.theme = theme;
  root.classList.toggle('dark', theme === 'dark');
  root.style.setProperty('--color-bg-primary', palette.bgPrimary);
  root.style.setProperty('--color-bg-surface', palette.bgSurface);
  root.style.setProperty('--color-primary', palette.colorPrimary);
  root.style.setProperty('--color-secondary', palette.colorSecondary);
  root.style.setProperty('--color-accent', palette.colorAccent);
  root.style.setProperty('--color-error', palette.colorError);
  root.style.setProperty('--color-text-primary', palette.textPrimary);
  root.style.setProperty('--color-text-muted', palette.textMuted);
}

export function createThemeController(options: ThemeControllerOptions = {}) {
  const storage = options.storage ?? defaultStorage();
  const root = options.root ?? defaultRoot();
  const matchMediaFn = options.matchMedia ?? defaultMatchMedia;
  let state: ThemeState = {
    preference: loadThemePreference(storage),
    systemTheme: getSystemTheme(matchMediaFn),
    resolved: 'light'
  };
  state = { ...state, resolved: resolveTheme(state.preference, state.systemTheme) };
  applyTheme(state.resolved, root);

  async function setThemePreference(preference: ThemePreference): Promise<ThemeState> {
    state = {
      ...state,
      preference: normalizeThemePreference(preference),
      resolved: resolveTheme(normalizeThemePreference(preference), state.systemTheme)
    };
    saveThemePreference(state.preference, storage);
    applyTheme(state.resolved, root);
    await options.syncPreference?.(state.preference);
    return state;
  }

  function handleSystemTheme(systemTheme: ResolvedTheme): ThemeState {
    state = {
      ...state,
      systemTheme,
      resolved: resolveTheme(state.preference, systemTheme)
    };
    applyTheme(state.resolved, root);
    return state;
  }

  function getState(): ThemeState {
    return state;
  }

  return { getState, setThemePreference, handleSystemTheme };
}

function defaultRoot(): ThemeRoot | undefined {
  return typeof document === 'undefined' ? undefined : document.documentElement;
}

function defaultStorage(): ThemeStorage | undefined {
  return typeof localStorage === 'undefined' ? undefined : localStorage;
}

function defaultMatchMedia(query: string): MediaQueryListLike {
  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    return window.matchMedia(query);
  }
  return { matches: false };
}
