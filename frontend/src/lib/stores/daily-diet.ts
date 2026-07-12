import { get, writable, type Writable } from "svelte/store";

import {
	dailyDietApi,
	DailyDietClientError,
	type DailyDietApi,
	type DailyDietMutationOptions
} from "../api/daily-diet-client";
import type {
	AppError,
	DailyDiet,
	DailyDietCreateRequest,
	DailyDietReplaceRequest
} from "../api/generated";
import { setDailyDietId } from "./search";

// Implements DESIGN-001 SearchView Daily Diet collection store/controller.
// Implements DESIGN-008 SavedDataRepository user-owned collection state without client persistence.

/** Server-state load lifecycle for the in-memory Daily Diet collection store. */
export type DailyDietLoadStatus = "idle" | "loading" | "success" | "empty" | "error";

/** Current Daily Diet mutation, if one is being submitted. */
export type DailyDietMutation = "idle" | "creating" | "replacing" | "deleting";

/** Frontend state for authenticated user-owned collections; no session secrets or user IDs are stored. */
export interface DailyDietState {
	collections: DailyDiet[];
	selectedId: string | null;
	status: DailyDietLoadStatus;
	mutation: DailyDietMutation;
	loading: boolean;
	error: AppError | null;
}

/** Creates a blank memory-only Daily Diet state. */
export function createInitialDailyDietState(): DailyDietState {
	return {
		collections: [],
		selectedId: null,
		status: "idle",
		mutation: "idle",
		loading: false,
		error: null
	};
}

/** Svelte store containing only server-returned Daily Diet data and reversible UI selection. */
export const dailyDietStore = writable<DailyDietState>(createInitialDailyDietState());

/** Controller dependencies can be replaced with fakes without changing the production store. */
export interface DailyDietControllerOptions {
	api?: DailyDietApi;
	store?: Writable<DailyDietState>;
}

/** Controller operations for loading and reconciling user-owned Daily Diet collections. */
export interface DailyDietController {
	load(signal?: AbortSignal): Promise<DailyDiet[]>;
	create(request: DailyDietCreateRequest, options?: DailyDietMutationOptions): Promise<DailyDiet>;
	replace(dietId: string, request: DailyDietReplaceRequest, options?: Pick<DailyDietMutationOptions, "csrfToken" | "signal">): Promise<DailyDiet>;
	select(dietId: string | null): void;
	remove(dietId: string, options?: Pick<DailyDietMutationOptions, "csrfToken" | "signal">): Promise<void>;
	clear(): void;
}

/** Creates a Daily Diet controller bound to a Svelte store and generated-contract API. */
export function createDailyDietController({ api = dailyDietApi, store = dailyDietStore }: DailyDietControllerOptions = {}): DailyDietController {
	let operation = 0;

	async function load(signal?: AbortSignal): Promise<DailyDiet[]> {
		const currentOperation = ++operation;
		update(store, (state) => ({ ...state, status: "loading", loading: true, error: null, mutation: "idle" }));
		try {
			const collections = await api.listDailyDiets(signal);
			if (currentOperation !== operation) return collections;
			update(store, (state) => ({
				...state,
				collections,
				selectedId: retainSelection(state.selectedId, collections),
				status: collections.length === 0 ? "empty" : "success",
				mutation: "idle",
				loading: false,
				error: null
			}));
			return collections;
		} catch (error) {
			if (currentOperation === operation) setFailure(store, error);
			throw error;
		}
	}

	async function create(request: DailyDietCreateRequest, options: DailyDietMutationOptions = {}): Promise<DailyDiet> {
		const currentOperation = ++operation;
		update(store, (state) => ({ ...state, mutation: "creating", loading: true, error: null }));
		try {
			const created = await api.createDailyDiet(request, options);
			if (currentOperation === operation) {
				update(store, (state) => {
					const collections = upsert(state.collections, created);
					return {
						...state,
						collections,
						status: "success",
						mutation: "idle",
						loading: false,
						error: null
					};
				});
			}
			return created;
		} catch (error) {
			if (currentOperation === operation) setFailure(store, error);
			throw error;
		}
	}

	async function replace(
		dietId: string,
		request: DailyDietReplaceRequest,
		options: Pick<DailyDietMutationOptions, "csrfToken" | "signal"> = {}
	): Promise<DailyDiet> {
		const currentOperation = ++operation;
		const previous = get(store);
		const existing = previous.collections.find((diet) => diet.id === dietId);
		update(store, (state) => ({
			...state,
			collections: existing && canOptimisticallyReplace(existing, request)
				? upsert(state.collections, optimisticReplace(existing, request))
				: state.collections,
			mutation: "replacing",
			loading: true,
			error: null
		}));
		try {
			const replaced = await api.replaceDailyDiet(dietId, request, options);
			if (currentOperation === operation) {
				update(store, (state) => ({
					...state,
					collections: upsert(state.collections, replaced),
					selectedId: retainSelection(state.selectedId, upsert(state.collections, replaced)),
					status: "success",
					mutation: "idle",
					loading: false,
					error: null
				}));
			}
			return replaced;
		} catch (error) {
			if (currentOperation === operation) {
				store.set({ ...previous, status: "error", mutation: "idle", loading: false, error: projectError(error) });
			}
			throw error;
		}
	}

	function select(dietId: string | null): void {
		const state = get(store);
		if (dietId !== null && !state.collections.some((diet) => diet.id === dietId)) {
			return;
		}
		store.update((current) => ({ ...current, selectedId: dietId, error: null }));
		// Daily Diet Alternative owns this projection; Catalog and Substitution state remain untouched.
		setDailyDietId(dietId ?? undefined);
	}

	async function remove(
		dietId: string,
		options: Pick<DailyDietMutationOptions, "csrfToken" | "signal"> = {}
	): Promise<void> {
		const currentOperation = ++operation;
		update(store, (state) => ({ ...state, mutation: "deleting", loading: true, error: null }));
		try {
			await api.deleteDailyDiet(dietId, options);
			if (currentOperation === operation) {
				update(store, (state) => {
					const collections = state.collections.filter((diet) => diet.id !== dietId);
					return {
						...state,
						collections,
						selectedId: state.selectedId === dietId ? null : state.selectedId,
						status: collections.length === 0 ? "empty" : "success",
						mutation: "idle",
						loading: false,
						error: null
					};
				});
				if (get(store).selectedId === null) setDailyDietId(undefined);
			}
		} catch (error) {
			if (currentOperation === operation) setFailure(store, error);
			throw error;
		}
	}

	function clear(): void {
		operation += 1;
		store.set(createInitialDailyDietState());
		setDailyDietId(undefined);
	}

	return { load, create, replace, select, remove, clear };
}

/** Default production controller used by Daily Diet UI components. */
export const dailyDietController = createDailyDietController();

/** Loads server state and exposes loading/success/empty/error transitions. */
export const loadDailyDiets = dailyDietController.load;

/** Creates a collection and reconciles the server-returned aggregate DTO into the store. */
export const createDailyDiet = dailyDietController.create;

/** Replaces a collection with rollback for the narrow safe optimistic projection. */
export const replaceDailyDiet = dailyDietController.replace;

/** Selects only a collection already present in server state. */
export const selectDailyDiet = dailyDietController.select;

/** Deletes a collection only after server success; failures retain the prior collection list. */
export const deleteDailyDiet = dailyDietController.remove;

/** Clears in-memory Daily Diet state, for logout/account changes and deterministic teardown. */
export const clearDailyDietState = dailyDietController.clear;

function update(store: Writable<DailyDietState>, updater: (state: DailyDietState) => DailyDietState): void {
	store.update(updater);
}

function setFailure(store: Writable<DailyDietState>, error: unknown): void {
	store.update((state) => ({ ...state, status: "error", mutation: "idle", loading: false, error: projectError(error) }));
}

function projectError(error: unknown): AppError {
	if (error instanceof DailyDietClientError) {
		return error.appError;
	}
	return {
		category: "unknown",
		code: "daily_diet_request_failed",
		message: "Saved daily diets are temporarily unavailable. Please try again.",
		retryable: true
	};
}

function retainSelection(selectedId: string | null, collections: DailyDiet[]): string | null {
	return selectedId !== null && collections.some((diet) => diet.id === selectedId) ? selectedId : null;
}

function upsert(collections: DailyDiet[], diet: DailyDiet): DailyDiet[] {
	const existingIndex = collections.findIndex((item) => item.id === diet.id);
	if (existingIndex < 0) return [...collections, diet];
	return collections.map((item, index) => (index === existingIndex ? diet : item));
}

function canOptimisticallyReplace(existing: DailyDiet, request: DailyDietReplaceRequest): boolean {
	return existing.entries.length === request.entries.length && request.entries.every((entry, index) => {
		const current = existing.entries[index];
		return current?.mealId === entry.mealId && current.position === entry.position;
	});
}

function optimisticReplace(existing: DailyDiet, request: DailyDietReplaceRequest): DailyDiet {
	return {
		...existing,
		name: request.name,
		entries: existing.entries.map((entry, index) => ({ ...entry, ...request.entries[index] }))
	};
}
