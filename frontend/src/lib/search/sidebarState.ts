import type { Entitlement, MacroToggles, SearchMode, TagFilter, UUID } from '../api/types';
import { createEntitlementViewState, type EntitlementUsage, type ModeGate } from '../entitlements/entitlementState';

export type SidebarAction = 'saved-searches' | 'history' | 'favorites' | 'profile' | 'settings';

export interface DietaryOption {
  id: UUID;
  label: string;
}

export interface SidebarState {
  isCollapsed: boolean;
  activeMode: SearchMode;
  enabledMacros: MacroToggles;
  dietaryFilters: TagFilter[];
  modeGates: Record<SearchMode, ModeGate>;
  usageLabel: string;
  upgradePrompt?: string;
}

export const defaultDietaryOptions: DietaryOption[] = [
  { id: 'diet-vegan', label: 'Vegan' },
  { id: 'diet-vegetarian', label: 'Vegetarian' },
  { id: 'allergen-dairy', label: 'Dairy-free' },
  { id: 'allergen-gluten', label: 'Gluten-free' }
];

const unlockedModeGates: Record<SearchMode, ModeGate> = {
  single: { mode: 'single', locked: false },
  replacement: { mode: 'replacement', locked: false },
  diet: { mode: 'diet', locked: false }
};

export function createSidebarState(activeMode: SearchMode = 'single'): SidebarState {
  return {
    isCollapsed: false,
    activeMode,
    enabledMacros: { protein: true, carbs: true, fat: true },
    dietaryFilters: [],
    modeGates: unlockedModeGates,
    usageLabel: 'Unlimited searches'
  };
}

export function toggleSidebar(state: SidebarState): SidebarState {
  return { ...state, isCollapsed: !state.isCollapsed };
}

export function setSidebarMode(state: SidebarState, mode: SearchMode): SidebarState {
  if (state.modeGates[mode]?.locked) {
    return state;
  }
  return { ...state, activeMode: mode };
}

export function applySidebarEntitlement(state: SidebarState, entitlement: Entitlement, usage?: EntitlementUsage): SidebarState {
  const view = createEntitlementViewState(entitlement, usage);
  return {
    ...state,
    activeMode: view.modeGates[state.activeMode]?.locked ? 'single' : state.activeMode,
    modeGates: view.modeGates,
    usageLabel: view.usageLabel,
    upgradePrompt: view.upgradePrompt
  };
}

export function setSidebarMacro(state: SidebarState, macro: keyof MacroToggles, enabled: boolean): SidebarState {
  return { ...state, enabledMacros: { ...state.enabledMacros, [macro]: enabled } };
}

export function toggleDietaryFilter(state: SidebarState, option: DietaryOption): SidebarState {
  const existing = state.dietaryFilters.find((filter) => filter.tagId === option.id);
  if (existing) {
    return { ...state, dietaryFilters: state.dietaryFilters.filter((filter) => filter.tagId !== option.id) };
  }
  const kind = option.id.startsWith('allergen-') ? 'allergen' : 'diet';
  const include = kind === 'diet';
  return {
    ...state,
    dietaryFilters: [...state.dietaryFilters, { tagId: option.id, kind, include }]
  };
}

export function sidebarLayout(widthPx: number): 'mobile' | 'desktop' {
  return widthPx < 640 ? 'mobile' : 'desktop';
}
