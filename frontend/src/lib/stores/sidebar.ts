import { writable } from "svelte/store";

// Implements DESIGN-001 SidebarComponent responsive collapse and mobile-toggle state.

/**
 * Sidebar collapse and mobile-open state consumed by the SidebarComponent.
 *
 * @remarks Implements DESIGN-001 SidebarComponent responsive collapse behavior.
 */
export interface SidebarState {
	collapsed: boolean;
	mobileOpen: boolean;
}

/**
 * Default sidebar state: expanded on desktop and closed on mobile. Used as the
 * startup value and as the reset target for tests.
 *
 * @remarks Implements DESIGN-001 SidebarComponent startup initialization.
 */
export function createInitialSidebarState(): SidebarState {
	return { collapsed: false, mobileOpen: false };
}

/**
 * localStorage key under which the desktop `collapsed` flag is persisted as JSON.
 *
 * @remarks Implements DESIGN-001 LocalStorageManager sidebar persistence key.
 */
export const SIDEBAR_STORAGE_KEY = "mealswapp.sidebar";

/**
 * Svelte writable store holding the current SidebarComponent collapse and mobile toggle state.
 *
 * @remarks Implements DESIGN-001 SidebarComponent Svelte store initialization.
 */
export const sidebarStore = writable<SidebarState>(createInitialSidebarState());

/**
 * Validates that an unknown value is a well-formed persisted sidebar preference.
 *
 * @remarks Implements DESIGN-001 LocalStorageManager schema validation for sidebar state.
 */
export function isValidSidebarState(value: unknown): value is { collapsed: boolean } {
	if (typeof value !== "object" || value === null) {
		return false;
	}
	const candidate = value as { collapsed?: unknown };
	return typeof candidate.collapsed === "boolean";
}

/**
 * Loads the persisted desktop collapse preference from localStorage and seeds the store.
 * `mobileOpen` is never persisted: the sidebar always starts closed on mobile so a
 * stale open flag never traps focus on small screens. Safe during SSR and when
 * localStorage throws. Falls back to defaults when storage is unavailable, missing,
 * malformed, or holds an invalid shape.
 *
 * @remarks Implements DESIGN-001 LocalStorageManager loadSettings and storage-unavailable fallback for the sidebar.
 */
export function initSidebar(): void {
	if (typeof window === "undefined") {
		sidebarStore.set(createInitialSidebarState());
		return;
	}

	let raw: string | null;
	try {
		raw = window.localStorage.getItem(SIDEBAR_STORAGE_KEY);
	} catch {
		// Storage reads are optional; default closed mobile state keeps the page usable.
		sidebarStore.set(createInitialSidebarState());
		return;
	}

	if (raw === null) {
		sidebarStore.set(createInitialSidebarState());
		return;
	}

	let parsed: unknown;
	try {
		parsed = JSON.parse(raw);
	} catch {
		// Malformed persisted UI state is ignored instead of blocking the sidebar.
		sidebarStore.set(createInitialSidebarState());
		return;
	}

	const collapsed = isValidSidebarState(parsed) ? parsed.collapsed : false;
	sidebarStore.set({ collapsed, mobileOpen: false });
}

/**
 * Toggles the desktop collapse flag and persists the new value to localStorage.
 *
 * @remarks Implements DESIGN-001 SidebarComponent desktop collapse persistence.
 */
export function toggleCollapsed(): void {
	sidebarStore.update((state) => {
		const next = { ...state, collapsed: !state.collapsed };
		persistSidebarCollapsed(next.collapsed);
		return next;
	});
}

/**
 * Sets the mobile open flag without touching the desktop collapse preference.
 *
 * @remarks Implements DESIGN-001 SidebarComponent mobile toggle state.
 */
export function setMobileOpen(open: boolean): void {
	sidebarStore.update((state) => ({ ...state, mobileOpen: open }));
}

/**
 * Toggles the mobile open flag.
 *
 * @remarks Implements DESIGN-001 SidebarComponent mobile toggle behavior.
 */
export function toggleMobileOpen(): void {
	sidebarStore.update((state) => ({ ...state, mobileOpen: !state.mobileOpen }));
}

/**
 * Resets the sidebar store to defaults without touching localStorage. Used by tests.
 *
 * @remarks Implements DESIGN-001 SidebarComponent default restoration.
 */
export function resetSidebar(): void {
	sidebarStore.set(createInitialSidebarState());
}

/**
 * Persists the desktop collapse flag to localStorage. Safe during SSR and when
 * localStorage throws (e.g. quota exceeded or disabled).
 *
 * @remarks Implements DESIGN-001 LocalStorageManager sidebar persistence.
 */
function persistSidebarCollapsed(collapsed: boolean): void {
	if (typeof window === "undefined") {
		return;
	}

	try {
		window.localStorage.setItem(
			SIDEBAR_STORAGE_KEY,
			JSON.stringify({ collapsed } satisfies { collapsed: boolean })
		);
	} catch {
		// Storage unavailable or quota exceeded; the in-memory store still serves callers.
		return;
	}
}
