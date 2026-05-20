import { describe, expect, it } from 'bun:test';
import {
  checkoutReturnShouldRefresh,
  createEntitlementViewState,
  defaultFreeEntitlement,
  nextAllowedMode
} from './entitlementState';
import type { Entitlement } from '../api/types';

describe('frontend entitlement gating', () => {
  it('locks paid modes for free users and shows remaining usage', () => {
    const view = createEntitlementViewState(defaultFreeEntitlement, { searchesUsed: 1 });

    expect(view.modeGates.single.locked).toBe(false);
    expect(view.modeGates.replacement.locked).toBe(true);
    expect(view.modeGates.replacement.reason).toBe('upgrade');
    expect(view.modeGates.diet.locked).toBe(true);
    expect(view.usageLabel).toBe('1 / 3 searches');
    expect(view.upgradePrompt).toBe('Upgrade to unlock replacement and diet modes.');
  });

  it('locks free single mode after the free search limit is reached', () => {
    const view = createEntitlementViewState(defaultFreeEntitlement, { searchesUsed: 3 });

    expect(view.modeGates.single.locked).toBe(true);
    expect(view.modeGates.single.reason).toBe('limit');
    expect(view.upgradePrompt).toBe('Upgrade for unlimited searches.');
  });

  it('unlocks paid modes for active trial and paid entitlements', () => {
    const trial = entitlement('trial');
    const paid = entitlement('paid');

    expect(createEntitlementViewState(trial, { searchesUsed: 99 }).modeGates.diet.locked).toBe(false);
    expect(createEntitlementViewState(paid, { searchesUsed: 99 }).modeGates.replacement.locked).toBe(false);
    expect(createEntitlementViewState(paid, { searchesUsed: 99 }).usageLabel).toBe('Unlimited searches');
  });

  it('blocks paid modes for billing recovery states', () => {
    const view = createEntitlementViewState({ ...entitlement('paid'), status: 'past_due' }, { searchesUsed: 0 });

    expect(view.modeGates.single.locked).toBe(false);
    expect(view.modeGates.diet.locked).toBe(true);
    expect(view.modeGates.diet.reason).toBe('billing');
    expect(view.upgradePrompt).toBe('Update billing to restore paid modes.');
  });

  it('falls back to the next allowed mode when a locked mode is selected', () => {
    const view = createEntitlementViewState(defaultFreeEntitlement, { searchesUsed: 0 });

    expect(nextAllowedMode(view, 'diet')).toBe('single');
    expect(nextAllowedMode(view, 'single')).toBe('single');
  });

  it('detects checkout return states that should refresh entitlement status', () => {
    expect(checkoutReturnShouldRefresh('/search?checkout=success')).toBe(true);
    expect(checkoutReturnShouldRefresh('/account?subscription=updated')).toBe(true);
    expect(checkoutReturnShouldRefresh('/search?checkout=cancelled')).toBe(false);
  });
});

function entitlement(tier: 'trial' | 'paid'): Entitlement {
  return {
    userId: 'user-1',
    tier,
    status: 'active',
    searchLimitPer24h: -1,
    allowedModes: ['single', 'replacement', 'diet']
  };
}
