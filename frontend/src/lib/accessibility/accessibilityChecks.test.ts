import { describe, expect, it } from 'bun:test';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import {
  contrastRatio,
  passesWcagAA,
  searchShellFocusOrder,
  searchShellLayout,
  wrapsWithinLines
} from './accessibilityChecks';
import { palettes } from '../theme/theme';

const sourceRoot = join(import.meta.dir, '..');

function source(path: string): string {
  return readFileSync(join(sourceRoot, path), 'utf8');
}

describe('frontend accessibility and responsive checks', () => {
  it('keeps light and dark theme text contrast at WCAG AA levels', () => {
    for (const palette of Object.values(palettes)) {
      expect(passesWcagAA(palette.textPrimary, palette.bgPrimary)).toBe(true);
      expect(passesWcagAA(palette.textPrimary, palette.bgSurface)).toBe(true);
      expect(passesWcagAA(palette.textMuted, palette.bgSurface)).toBe(true);
      expect(passesWcagAA(palette.colorError, palette.bgSurface)).toBe(true);
      expect(contrastRatio(palette.colorPrimary, palette.bgSurface)).toBeGreaterThanOrEqual(3);
    }
  });

  it('models mobile, tablet, and desktop search shell layouts', () => {
    expect(searchShellLayout(375)).toMatchObject({ columns: 1, sidebarPlacement: 'top', resultColumns: 1 });
    expect(searchShellLayout(768)).toMatchObject({ columns: 12, sidebarPlacement: 'left', resultColumns: 2 });
    expect(searchShellLayout(1280)).toMatchObject({ columns: 12, sidebarPlacement: 'left', resultColumns: 3 });
  });

  it('defines a keyboard-only path from controls through results and settings', () => {
    expect(searchShellFocusOrder).toEqual([
      'sidebar-toggle',
      'mode-single',
      'mode-replacement',
      'mode-diet',
      'macro-protein',
      'macro-carbs',
      'macro-fat',
      'search-input',
      'autocomplete-option',
      'result-card',
      'pagination',
      'settings'
    ]);
  });

  it('keeps long control labels wrap-safe inside compact surfaces', () => {
    expect(wrapsWithinLines('Diet alternative generation', 16, 2)).toBe(true);
    expect(wrapsWithinLines('Supercalifragilisticexpialidocious', 16, 2)).toBe(false);
  });

  it('keeps search input and autocomplete wired with ARIA relationships', () => {
    const searchView = source('components/SearchView.svelte');
    const autocomplete = source('components/AutocompleteDropdown.svelte');

    expect(searchView).toContain('aria-labelledby="search-heading"');
    expect(searchView).toContain('aria-autocomplete="list"');
    expect(searchView).toContain('aria-controls="search-autocomplete"');
    expect(searchView).toContain('aria-activedescendant=');
    expect(autocomplete).toContain('role="listbox"');
    expect(autocomplete).toContain('role="option"');
    expect(autocomplete).toContain('aria-selected=');
    expect(autocomplete).toContain('aria-busy=');
  });

  it('keeps sidebar controls labelled and keyboard-addressable', () => {
    const sidebar = source('components/SidebarComponent.svelte');

    expect(sidebar).toContain('aria-expanded={!collapsed}');
    expect(sidebar).toContain('aria-label="Search modes"');
    expect(sidebar).toContain('aria-labelledby="macro-heading"');
    expect(sidebar).toContain('aria-labelledby="dietary-heading"');
    expect(sidebar).toContain('type="checkbox"');
  });

  it('keeps offline and optimization status regions announced', () => {
    const offline = source('components/OfflineBanner.svelte');
    const optimization = source('components/OptimizationPanel.svelte');

    expect(offline).toContain('aria-live="polite"');
    expect(offline).toContain('aria-label="Connection status"');
    expect(optimization).toContain('aria-live="polite"');
    expect(optimization).toContain('aria-labelledby="optimization-heading"');
  });

  it('keeps settings and account data controls labelled', () => {
    const settings = source('components/SettingsPanel.svelte');

    expect(settings).toContain('aria-labelledby="settings-heading"');
    expect(settings).toContain('<legend class="font-mono text-xs uppercase text-text-muted">Theme</legend>');
    expect(settings).toContain('<legend class="font-mono text-xs uppercase text-text-muted">Units</legend>');
    expect(settings).toContain('<legend class="font-mono text-xs uppercase text-text-muted">Macros</legend>');
    expect(settings).toContain('Confirm deletion');
  });

  it('keeps admin workflows labelled and table headers scoped', () => {
    const admin = source('components/AdminView.svelte');

    expect(admin).toContain('aria-label="Admin sections"');
    expect(admin).toContain('aria-current=');
    expect(admin).toContain('for="admin-external-query"');
    expect(admin).toContain('id="admin-external-query"');
    expect(admin).toContain('for="admin-external-provider"');
    expect(admin).toContain('id="admin-external-provider"');
    expect(admin).toContain('for="admin-tag-kind"');
    expect(admin).toContain('role="alert"');
    expect(admin).toContain('scope="col"');
    expect(admin).toContain('aria-labelledby="import-preview-heading"');
  });

  it('keeps global focus, text fitting, and reduced-motion guards in CSS', () => {
    const css = readFileSync(join(sourceRoot, '..', 'app.css'), 'utf8');

    expect(css).toContain(':focus-visible');
    expect(css).toContain('outline: 2px solid var(--color-primary)');
    expect(css).toContain('overflow-wrap: anywhere');
    expect(css).toContain('prefers-reduced-motion: reduce');
  });
});
