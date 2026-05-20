import type { MacroToggles, UserProfile } from '../api/types';
import type { ThemePreference } from '../theme/theme';

export interface SettingsState {
  themePreference: ThemePreference;
  unitSystem: 'metric' | 'imperial';
  enabledMacros: MacroToggles;
  disclaimerVisible: boolean;
  consentVisible: boolean;
  exportFormat: 'json' | 'csv';
  deleteConfirmation: string;
}

export function createSettingsState(profile?: UserProfile): SettingsState {
  return {
    themePreference: profile?.themePreference ?? metadataTheme(profile) ?? 'system',
    unitSystem: profile?.unitSystem ?? 'metric',
    enabledMacros: { protein: true, carbs: true, fat: true },
    disclaimerVisible: true,
    consentVisible: true,
    exportFormat: 'json',
    deleteConfirmation: ''
  };
}

export function updateTheme(state: SettingsState, themePreference: ThemePreference): SettingsState {
  return { ...state, themePreference };
}

export function updateUnitSystem(state: SettingsState, unitSystem: 'metric' | 'imperial'): SettingsState {
  return { ...state, unitSystem };
}

export function updateMacroPreference(state: SettingsState, macro: keyof MacroToggles, enabled: boolean): SettingsState {
  return { ...state, enabledMacros: { ...state.enabledMacros, [macro]: enabled } };
}

export function canDeleteAccount(confirmation: string): boolean {
  return confirmation.trim() === 'DELETE';
}

export function buildProfilePreferenceUpdate(state: SettingsState): Partial<UserProfile> {
  return {
    unitSystem: state.unitSystem,
    themePreference: state.themePreference,
    metadata: {
      themePreference: state.themePreference,
      unitSystem: state.unitSystem,
      enabledMacros: state.enabledMacros
    }
  };
}

function metadataTheme(profile?: UserProfile): ThemePreference | undefined {
  const value = profile?.metadata?.themePreference;
  return value === 'system' || value === 'light' || value === 'dark' ? value : undefined;
}
