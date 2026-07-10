import { derived, writable } from "svelte/store";

import type { AppError, EntitlementStatusData } from "../api/generated";

// Implements DESIGN-001 SearchView current user entitlement Svelte state.

/**
 * Holds the latest generated entitlement status payload resolved by TanStack Query.
 *
 * @remarks Implements DESIGN-001 SearchView current user entitlement store.
 */
export const entitlementStatusStore = writable<EntitlementStatusData | null>(null);

/**
 * Holds the latest recoverable entitlement or billing error surfaced by the client.
 *
 * @remarks Implements DESIGN-001 SearchView entitlement error state.
 */
export const entitlementErrorStore = writable<AppError | null>(null);

/**
 * Derived list of generated search modes currently allowed by the user's entitlement.
 *
 * @remarks Implements DESIGN-001 SearchView allowed search modes entitlement state.
 */
export const allowedSearchModesStore = derived(entitlementStatusStore, ($status) => $status?.allowedModes ?? []);

/**
 * Derived remaining usage count for metered plans; `null` means the generated contract reports no cap.
 *
 * @remarks Implements DESIGN-001 SearchView usage remaining entitlement state.
 */
export const usageRemainingStore = derived(entitlementStatusStore, ($status) => $status?.usageRemaining ?? null);

/**
 * Stores a newly fetched entitlement payload and clears stale billing errors.
 *
 * @remarks Implements DESIGN-001 SearchView entitlement state update.
 */
export function setEntitlementStatus(status: EntitlementStatusData): void {
	entitlementStatusStore.set(status);
	entitlementErrorStore.set(null);
}

/**
 * Stores the current billing or entitlement error while preserving any previous status for UI fallback.
 *
 * @remarks Implements DESIGN-001 SearchView recoverable entitlement error update.
 */
export function setEntitlementError(error: AppError): void {
	entitlementErrorStore.set(error);
}

/**
 * Resets entitlement state between sessions and tests.
 *
 * @remarks Implements DESIGN-001 SearchView entitlement state reset.
 */
export function resetEntitlementState(): void {
	entitlementStatusStore.set(null);
	entitlementErrorStore.set(null);
}
