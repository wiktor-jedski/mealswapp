import { describe, expect, it } from 'bun:test';
import {
  buildProfilePreferenceUpdate,
  canDeleteAccount,
  createSettingsState,
  updateMacroPreference,
  updateTheme,
  updateUnitSystem
} from './settingsPanel';

describe('SettingsPanel state', () => {
  it('loads profile preferences with safe defaults', () => {
    const state = createSettingsState({ id: 'user-1', metadata: { themePreference: 'dark' } });

    expect(state.themePreference).toBe('dark');
    expect(state.unitSystem).toBe('metric');
    expect(state.enabledMacros).toEqual({ protein: true, carbs: true, fat: true });
    expect(state.disclaimerVisible).toBe(true);
    expect(state.consentVisible).toBe(true);
  });

  it('updates theme, unit, and macro preferences', () => {
    const state = updateMacroPreference(updateUnitSystem(updateTheme(createSettingsState(), 'light'), 'imperial'), 'fat', false);

    expect(state.themePreference).toBe('light');
    expect(state.unitSystem).toBe('imperial');
    expect(state.enabledMacros.fat).toBe(false);
  });

  it('builds a profile preference update payload', () => {
    const state = updateMacroPreference(updateTheme(createSettingsState(), 'dark'), 'carbs', false);

    expect(buildProfilePreferenceUpdate(state)).toEqual({
      unitSystem: 'metric',
      themePreference: 'dark',
      metadata: {
        themePreference: 'dark',
        unitSystem: 'metric',
        enabledMacros: { protein: true, carbs: false, fat: true }
      }
    });
  });

  it('requires explicit delete confirmation', () => {
    expect(canDeleteAccount('delete')).toBe(false);
    expect(canDeleteAccount(' DELETE ')).toBe(true);
  });
});
