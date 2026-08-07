import { get, writable } from "svelte/store";

import * as authApi from "../api/auth-client";
import type { ProfileData, ProfileUpdateRequest } from "../api/generated";

// Implements DESIGN-001 SettingsPanel authoritative anonymous/authenticated unit preference flow.
// Implements DESIGN-008 PreferenceManager server-confirmed profile preference persistence.

/** Supported measurement unit systems for frontend display and quantity entry. */
export type UnitSystem = "metric" | "imperial";

/** Anonymous browser-local display preferences. */
export interface SearchPreferences {
	unitSystem: UnitSystem;
}

/** Recoverable lifecycle state for the authoritative unit preference. */
export interface UnitPreferenceStatus {
	authority: "anonymous" | "authenticated";
	state: "ready" | "loading" | "saving" | "error";
	userId?: string;
	operation?: "load" | "save";
	pendingUnitSystem?: UnitSystem;
	message?: string;
}

interface PreferenceDependencies {
	fetchCsrfToken: typeof authApi.fetchCsrfToken;
	probeProfileSession: typeof authApi.probeProfileSession;
	updateProfileSession: typeof authApi.updateProfileSession;
}

/** Browser-local key reserved for the anonymous device preference. */
export const PREFERENCES_STORAGE_KEY = "mealswapp.preferences";

const defaultDependencies: PreferenceDependencies = {
	fetchCsrfToken: authApi.fetchCsrfToken,
	probeProfileSession: authApi.probeProfileSession,
	updateProfileSession: authApi.updateProfileSession
};
let dependencies = defaultDependencies;
let activeProfile: ProfileData | null = null;
let activeController: AbortController | null = null;
let generation = 0;

/** Returns the anonymous first-use default. */
export function createDefaultPreferences(): SearchPreferences {
	return { unitSystem: "metric" };
}

/** Current unit system consumed by all frontend rendering and input conversion. */
export const preferencesStore = writable<SearchPreferences>(createDefaultPreferences());

/** Load/save/error state rendered by the accessible sidebar preference control. */
export const unitPreferenceStatusStore = writable<UnitPreferenceStatus>({
	authority: "anonymous",
	state: "ready"
});

/** Validates the anonymous localStorage payload. */
export function isValidPreferences(value: unknown): value is SearchPreferences {
	if (typeof value !== "object" || value === null) {
		return false;
	}
	const candidate = value as { unitSystem?: unknown };
	return candidate.unitSystem === "metric" || candidate.unitSystem === "imperial";
}

/** Loads the anonymous device preference without consulting account state. */
export function initPreferences(): void {
	restoreAnonymousUnitPreference();
}

/**
 * Loads one account's server profile as the authoritative preference.
 *
 * A later load, account switch, or sign-out aborts this request and invalidates its result.
 */
export async function loadAuthenticatedUnitPreference(
	userId: string,
	loadProfile: (signal?: AbortSignal) => Promise<ProfileData> = dependencies.probeProfileSession
): Promise<ProfileData | null> {
	const operation = startAuthenticatedOperation(userId, "loading");
	try {
		const profile = await loadProfile(operation.controller.signal);
		if (!isCurrent(operation.id, userId)) {
			return null;
		}
		if (profile.userId !== userId) {
			throw new Error("Profile account did not match the authenticated session.");
		}
		activeProfile = profile;
		preferencesStore.set({ unitSystem: profile.unitSystem });
		unitPreferenceStatusStore.set({ authority: "authenticated", state: "ready", userId });
		return profile;
	} catch (error) {
		if (!isCurrent(operation.id, userId) || operation.controller.signal.aborted) {
			return null;
		}
		unitPreferenceStatusStore.set({
			authority: "authenticated",
			state: "error",
			userId,
			operation: "load",
			message: "Couldn't load your unit preference. Try again."
		});
		return null;
	}
}

/**
 * Changes the active unit. Anonymous changes are local and immediate; authenticated
 * changes update the UI only after the profile API confirms the saved value.
 */
export async function setUnitSystem(unit: UnitSystem): Promise<void> {
	const status = get(unitPreferenceStatusStore);
	if (status.authority === "authenticated" && activeProfile === null) {
		return;
	}
	if (activeProfile === null) {
		preferencesStore.set({ unitSystem: unit });
		writeAnonymousPreferences(unit);
		return;
	}

	const profile = activeProfile;
	const operation = startAuthenticatedOperation(profile.userId, "saving", unit);
	const request: ProfileUpdateRequest = {
		displayName: profile.displayName,
		unitSystem: unit,
		themePreference: profile.themePreference
	};
	try {
		const { csrfToken } = await dependencies.fetchCsrfToken(operation.controller.signal);
		const confirmed = await dependencies.updateProfileSession(request, {
			csrfToken,
			signal: operation.controller.signal
		});
		if (!isCurrent(operation.id, profile.userId)) {
			return;
		}
		if (confirmed.userId !== profile.userId) {
			throw new Error("Profile account did not match the authenticated session.");
		}
		activeProfile = confirmed;
		preferencesStore.set({ unitSystem: confirmed.unitSystem });
		unitPreferenceStatusStore.set({
			authority: "authenticated",
			state: "ready",
			userId: profile.userId
		});
	} catch {
		if (!isCurrent(operation.id, profile.userId) || operation.controller.signal.aborted) {
			return;
		}
		unitPreferenceStatusStore.set({
			authority: "authenticated",
			state: "error",
			userId: profile.userId,
			operation: "save",
			pendingUnitSystem: unit,
			message: "Couldn't save your unit preference. Try again."
		});
	}
}

/** Retries the most recent authenticated load or save failure. */
export async function retryUnitPreference(): Promise<void> {
	const status = get(unitPreferenceStatusStore);
	if (status?.state !== "error" || !status.userId) {
		return;
	}
	if (status.operation === "save" && status.pendingUnitSystem) {
		await setUnitSystem(status.pendingUnitSystem);
	} else {
		await loadAuthenticatedUnitPreference(status.userId);
	}
}

/** Cancels account work and restores the separate anonymous device preference. */
export function restoreAnonymousUnitPreference(): void {
	cancelActiveOperation();
	activeProfile = null;
	preferencesStore.set(readAnonymousPreferences());
	unitPreferenceStatusStore.set({ authority: "anonymous", state: "ready" });
}

/** Test-only API dependency replacement. */
export function setPreferenceDependencies(patch: Partial<PreferenceDependencies>): void {
	dependencies = { ...defaultDependencies, ...patch };
}

/** Resets preference state and pending requests between tests. */
export function resetPreferences(): void {
	cancelActiveOperation();
	dependencies = defaultDependencies;
	activeProfile = null;
	preferencesStore.set(createDefaultPreferences());
	unitPreferenceStatusStore.set({ authority: "anonymous", state: "ready" });
}

function startAuthenticatedOperation(
	userId: string,
	state: "loading" | "saving",
	pendingUnitSystem?: UnitSystem
): { id: number; controller: AbortController } {
	cancelActiveOperation();
	const controller = new AbortController();
	activeController = controller;
	generation += 1;
	if (state === "loading") {
		activeProfile = null;
		preferencesStore.set(createDefaultPreferences());
	}
	unitPreferenceStatusStore.set({
		authority: "authenticated",
		state,
		userId,
		...(pendingUnitSystem ? { pendingUnitSystem } : {})
	});
	return { id: generation, controller };
}

function cancelActiveOperation(): void {
	activeController?.abort();
	activeController = null;
	generation += 1;
}

function isCurrent(id: number, userId: string): boolean {
	const current = get(unitPreferenceStatusStore);
	return id === generation && current?.authority === "authenticated" && current.userId === userId;
}

function readAnonymousPreferences(): SearchPreferences {
	if (typeof window === "undefined") {
		return createDefaultPreferences();
	}
	try {
		const raw = window.localStorage.getItem(PREFERENCES_STORAGE_KEY);
		if (raw === null) {
			return createDefaultPreferences();
		}
		const parsed: unknown = JSON.parse(raw);
		return isValidPreferences(parsed) ? parsed : createDefaultPreferences();
	} catch {
		return createDefaultPreferences();
	}
}

function writeAnonymousPreferences(unitSystem: UnitSystem): void {
	if (typeof window === "undefined") {
		return;
	}
	try {
		window.localStorage.setItem(PREFERENCES_STORAGE_KEY, JSON.stringify({ unitSystem }));
	} catch {
		// Anonymous storage is optional; the in-memory preference remains usable.
	}
}
