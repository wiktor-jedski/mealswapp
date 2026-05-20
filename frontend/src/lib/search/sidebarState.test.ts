import { describe, expect, it } from 'bun:test';
import {
  createSidebarState,
  setSidebarMacro,
  setSidebarMode,
  sidebarLayout,
  toggleDietaryFilter,
  toggleSidebar
} from './sidebarState';

describe('SidebarComponent state', () => {
  it('starts expanded with single mode and all macros enabled', () => {
    const state = createSidebarState();

    expect(state.isCollapsed).toBe(false);
    expect(state.activeMode).toBe('single');
    expect(state.enabledMacros).toEqual({ protein: true, carbs: true, fat: true });
  });

  it('toggles responsive collapsed state', () => {
    const collapsed = toggleSidebar(createSidebarState());
    expect(collapsed.isCollapsed).toBe(true);
    expect(toggleSidebar(collapsed).isCollapsed).toBe(false);
  });

  it('emits mode and macro state changes', () => {
    const modeState = setSidebarMode(createSidebarState(), 'replacement');
    const macroState = setSidebarMacro(modeState, 'fat', false);

    expect(modeState.activeMode).toBe('replacement');
    expect(macroState.enabledMacros).toEqual({ protein: true, carbs: true, fat: false });
  });

  it('emits dietary include and allergen exclude filters', () => {
    const withVegan = toggleDietaryFilter(createSidebarState(), { id: 'diet-vegan', label: 'Vegan' });
    const withDairy = toggleDietaryFilter(withVegan, { id: 'allergen-dairy', label: 'Dairy-free' });

    expect(withDairy.dietaryFilters).toEqual([
      { tagId: 'diet-vegan', kind: 'diet', include: true },
      { tagId: 'allergen-dairy', kind: 'allergen', include: false }
    ]);
    expect(toggleDietaryFilter(withDairy, { id: 'diet-vegan', label: 'Vegan' }).dietaryFilters).toEqual([
      { tagId: 'allergen-dairy', kind: 'allergen', include: false }
    ]);
  });

  it('uses the 640px responsive breakpoint', () => {
    expect(sidebarLayout(320)).toBe('mobile');
    expect(sidebarLayout(640)).toBe('desktop');
  });
});
