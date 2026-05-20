import type { Entitlement, SearchMode } from '../api/types';

export interface EntitlementUsage {
  searchesUsed: number;
  windowResetAt?: string;
}

export interface ModeGate {
  mode: SearchMode;
  locked: boolean;
  reason?: 'upgrade' | 'limit' | 'billing';
}

export interface EntitlementViewState {
  entitlement: Entitlement;
  usage: EntitlementUsage;
  modeGates: Record<SearchMode, ModeGate>;
  upgradePrompt?: string;
  usageLabel: string;
}

export const defaultFreeEntitlement: Entitlement = {
  userId: '',
  tier: 'free',
  status: 'active',
  searchLimitPer24h: 3,
  allowedModes: ['single']
};

export function createEntitlementViewState(
  entitlement: Entitlement = defaultFreeEntitlement,
  usage: EntitlementUsage = { searchesUsed: 0 }
): EntitlementViewState {
  const allowedModes = new Set(entitlement.allowedModes);
  const billingBlocked = entitlement.status === 'past_due' || entitlement.status === 'cancelled' || entitlement.status === 'expired';
  const exhausted = entitlement.searchLimitPer24h >= 0 && usage.searchesUsed >= entitlement.searchLimitPer24h;
  const modes: SearchMode[] = ['single', 'replacement', 'diet'];
  const modeGates = Object.fromEntries(
    modes.map((mode) => {
      const paidModeLocked = !allowedModes.has(mode);
      const singleLimitLocked = mode === 'single' && exhausted;
      const locked = billingBlocked ? mode !== 'single' : paidModeLocked || singleLimitLocked;
      const reason = billingBlocked && mode !== 'single' ? 'billing' : singleLimitLocked ? 'limit' : paidModeLocked ? 'upgrade' : undefined;
      return [mode, { mode, locked, reason }];
    })
  ) as Record<SearchMode, ModeGate>;

  return {
    entitlement,
    usage,
    modeGates,
    upgradePrompt: upgradePrompt(entitlement, exhausted),
    usageLabel: usageLabel(entitlement, usage)
  };
}

export function canUseMode(view: EntitlementViewState, mode: SearchMode): boolean {
  return !view.modeGates[mode]?.locked;
}

export function nextAllowedMode(view: EntitlementViewState, preferred: SearchMode): SearchMode {
  if (canUseMode(view, preferred)) {
    return preferred;
  }
  return canUseMode(view, 'single') ? 'single' : preferred;
}

export function checkoutReturnShouldRefresh(url: string): boolean {
  try {
    const parsed = new URL(url, 'https://mealswapp.local');
    return parsed.searchParams.get('checkout') === 'success' || parsed.searchParams.get('subscription') === 'updated';
  } catch {
    return false;
  }
}

function usageLabel(entitlement: Entitlement, usage: EntitlementUsage): string {
  if (entitlement.searchLimitPer24h < 0) {
    return 'Unlimited searches';
  }
  return `${Math.min(usage.searchesUsed, entitlement.searchLimitPer24h)} / ${entitlement.searchLimitPer24h} searches`;
}

function upgradePrompt(entitlement: Entitlement, exhausted: boolean): string | undefined {
  if (entitlement.status === 'past_due') {
    return 'Update billing to restore paid modes.';
  }
  if (exhausted) {
    return 'Upgrade for unlimited searches.';
  }
  if (entitlement.tier === 'free') {
    return 'Upgrade to unlock replacement and diet modes.';
  }
  return undefined;
}
