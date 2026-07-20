import { get, writable, type Writable } from "svelte/store";

import {
	dailyDietApi,
	DailyDietClientError,
	generateDailyDietIdempotencyKey,
	type DailyDietApi,
	type DailyDietMutationOptions
} from "../api/daily-diet-client";
import type {
	AppError,
	DailyDiet,
	DailyDietCreateRequest,
	DailyDietReplaceRequest
} from "../api/generated";
import { selectedDailyDietId } from "./selected-daily-diet";

// Implements DESIGN-001 SearchView Daily Diet collection store/controller.
// Implements DESIGN-008 SavedDataRepository user-owned collection state without client persistence.

/** Server-state load lifecycle for the in-memory Daily Diet collection store. */
export type DailyDietLoadStatus = "idle" | "loading" | "success" | "empty" | "error";

/** Current Daily Diet mutation, if one is being submitted. */
export type DailyDietMutation = "idle" | "creating" | "replacing" | "deleting";

/** Frontend state for authenticated user-owned collections; no session secrets or user IDs are stored. */
	export interface DailyDietState {
	collections: DailyDiet[];
	status: DailyDietLoadStatus;
	mutation: DailyDietMutation;
	loading: boolean;
	error: AppError | null;
}

/** Creates a blank memory-only Daily Diet state. */
export function createInitialDailyDietState(): DailyDietState {
	return {
		collections: [],
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
	selectionStore?: Writable<string | null>;
	createIdempotencyKey?: typeof generateDailyDietIdempotencyKey;
}

/** Controller operations for loading and reconciling user-owned Daily Diet collections. */
export interface DailyDietController {
	load(signal?: AbortSignal): Promise<DailyDiet[]>;
	create(request: DailyDietCreateRequest, options?: DailyDietMutationOptions): Promise<DailyDiet>;
	discardCreateIntent(): void;
	replace(dietId: string, request: DailyDietReplaceRequest, options?: Pick<DailyDietMutationOptions, "csrfToken" | "signal">): Promise<DailyDiet>;
	select(dietId: string | null): void;
	remove(dietId: string, options?: Pick<DailyDietMutationOptions, "csrfToken" | "signal">): Promise<void>;
	clear(): void;
}

/** Creates a Daily Diet controller bound to a Svelte store and generated-contract API. */
export function createDailyDietController({
	api = dailyDietApi,
	store = dailyDietStore,
	selectionStore = selectedDailyDietId,
	createIdempotencyKey = generateDailyDietIdempotencyKey
}: DailyDietControllerOptions = {}): DailyDietController {
	type ReadLifecycle = { controller: AbortController; promise: Promise<DailyDiet[]> };
	type MutationLifecycle = { kind: Exclude<DailyDietMutation, "idle">; controller: AbortController; promise: Promise<unknown> };
	let activeRead: ReadLifecycle | null = null;
	let activeMutation: MutationLifecycle | null = null;
	let createIntent: { fingerprint: string; idempotencyKey: string } | null = null;

	function load(signal?: AbortSignal): Promise<DailyDiet[]> {
		if (signal?.aborted) return Promise.reject(signal.reason ?? abortReason("Aborted"));
		activeRead?.controller.abort(abortReason("Superseded Daily Diet read"));
		const controller = new AbortController();
		const chained = chainAbortSignal(controller.signal, signal);
		const lifecycle: ReadLifecycle = { controller, promise: Promise.resolve([]) };
		activeRead = lifecycle;
		if (!activeMutation) update(store, (state) => ({ ...state, status: "loading", loading: true, error: null }));
		const promise = (async () => {
			try {
				while (activeMutation) await waitForSettlement(activeMutation.promise, chained.signal);
				throwIfAborted(chained.signal);
				if (activeRead === lifecycle) update(store, (state) => ({ ...state, status: "loading", loading: true, error: null }));
				const collections = await api.listDailyDiets(chained.signal);
				if (activeRead !== lifecycle || chained.signal.aborted) return collections;
				selectionStore.update((selectedId) => retainSelection(selectedId, collections));
				store.set({
					collections,
					status: collections.length === 0 ? "empty" : "success",
					mutation: "idle",
					loading: false,
					error: null
				});
				return collections;
			} catch (error) {
				if (activeRead === lifecycle && !chained.signal.aborted) setFailure(store, error);
				throw error;
			} finally {
				chained.cancel();
				if (activeRead === lifecycle) activeRead = null;
			}
		})();
		lifecycle.promise = promise;
		return promise;
	}

	function create(request: DailyDietCreateRequest, options: DailyDietMutationOptions = {}): Promise<DailyDiet> {
		if (activeMutation?.kind === "creating") return activeMutation.promise as Promise<DailyDiet>;
		if (activeMutation) return Promise.reject(mutationInProgress());
		const fingerprint = JSON.stringify(request);
		try {
			if (createIntent?.fingerprint !== fingerprint) {
				createIntent = { fingerprint, idempotencyKey: createIdempotencyKey() };
			}
		} catch (error) {
			setFailure(store, error);
			return Promise.reject(error);
		}
		const intent = createIntent;
		if (!intent) throw new Error("Daily Diet create intent was not initialized");
		return runMutation("creating", options.signal, async (signal) => {
			return api.createDailyDiet(request, { ...options, signal, idempotencyKey: intent.idempotencyKey });
		}, (created) => {
			update(store, (state) => ({ ...state, collections: upsert(state.collections, created) }));
			if (createIntent === intent) createIntent = null;
		});
	}

	function discardCreateIntent(): void {
		createIntent = null;
	}

	async function replace(
		dietId: string,
		request: DailyDietReplaceRequest,
		options: Pick<DailyDietMutationOptions, "csrfToken" | "signal"> = {}
	): Promise<DailyDiet> {
		if (activeMutation) return Promise.reject(mutationInProgress());
		return runMutation(
			"replacing",
			options.signal,
			(signal) => api.replaceDailyDiet(dietId, request, { ...options, signal }),
			(replaced) => update(store, (state) => ({ ...state, collections: upsert(state.collections, replaced) }))
		);
	}

	function select(dietId: string | null): void {
		const state = get(store);
		if (dietId !== null && !state.collections.some((diet) => diet.id === dietId)) {
			return;
		}
		selectionStore.set(dietId);
		store.update((current) => ({ ...current, error: null }));
	}

	async function remove(
		dietId: string,
		options: Pick<DailyDietMutationOptions, "csrfToken" | "signal"> = {}
	): Promise<void> {
		if (activeMutation) return Promise.reject(mutationInProgress());
		return runMutation("deleting", options.signal, async (signal) => {
			await api.deleteDailyDiet(dietId, { ...options, signal });
		}, () => {
			update(store, (state) => ({ ...state, collections: state.collections.filter((diet) => diet.id !== dietId) }));
			selectionStore.update((selectedId) => selectedId === dietId ? null : selectedId);
		});
	}

	function clear(): void {
		activeRead?.controller.abort(abortReason("Daily Diet state cleared"));
		activeMutation?.controller.abort(abortReason("Daily Diet state cleared"));
		activeRead = null;
		activeMutation = null;
		createIntent = null;
		store.set(createInitialDailyDietState());
		selectionStore.set(null);
	}

	function runMutation<T>(
		kind: Exclude<DailyDietMutation, "idle">,
		externalSignal: AbortSignal | undefined,
		execute: (signal: AbortSignal) => Promise<T>,
		commit: (result: T) => void
	): Promise<T> {
		if (externalSignal?.aborted) return Promise.reject(externalSignal.reason ?? abortReason("Aborted"));
		activeRead?.controller.abort(abortReason("Daily Diet mutation started"));
		activeRead = null;
		const controller = new AbortController();
		const chained = chainAbortSignal(controller.signal, externalSignal);
		const lifecycle: MutationLifecycle = { kind, controller, promise: Promise.resolve() };
		activeMutation = lifecycle;
		update(store, (state) => ({ ...state, mutation: kind, loading: true, error: null }));
		const promise = (async () => {
			try {
				throwIfAborted(chained.signal);
				const result = await execute(chained.signal);
				if (activeMutation === lifecycle && !chained.signal.aborted) {
					commit(result);
					finishMutation(store);
				}
				return result;
			} catch (error) {
				if (activeMutation === lifecycle) {
					if (chained.signal.aborted) finishCancelledMutation(store);
					else setFailure(store, error);
				}
				throw error;
			} finally {
				chained.cancel();
				if (activeMutation === lifecycle) activeMutation = null;
			}
		})();
		lifecycle.promise = promise;
		return promise;
	}

	return { load, create, discardCreateIntent, replace, select, remove, clear };
}

/** Default production controller used by Daily Diet UI components. */
export const dailyDietController = createDailyDietController();

/** Loads server state and exposes loading/success/empty/error transitions. */
export const loadDailyDiets = dailyDietController.load;

/** Creates a collection and reconciles the server-returned aggregate DTO into the store. */
export const createDailyDiet = dailyDietController.create;

/** Clears the memory-only retry key when the editable create intent changes. */
export const clearDailyDietCreateIntent = dailyDietController.discardCreateIntent;

/** Replaces a collection only with the decoded server-returned aggregate DTO. */
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

function finishMutation(store: Writable<DailyDietState>): void {
	store.update((state) => ({
		...state,
		status: state.collections.length === 0 ? "empty" : "success",
		mutation: "idle",
		loading: false,
		error: null
	}));
}

function finishCancelledMutation(store: Writable<DailyDietState>): void {
	store.update((state) => ({ ...state, mutation: "idle", loading: false }));
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

function mutationInProgress(): DailyDietClientError {
	return new DailyDietClientError({
		category: "validation",
		code: "daily_diet_mutation_in_progress",
		message: "Another Daily Diet change is still in progress.",
		retryable: true
	}, 409);
}

function abortReason(message: string): DOMException {
	return new DOMException(message, "AbortError");
}

function throwIfAborted(signal: AbortSignal): void {
	if (signal.aborted) throw signal.reason ?? abortReason("Aborted");
}

function waitForSettlement(promise: Promise<unknown>, signal: AbortSignal): Promise<void> {
	throwIfAborted(signal);
	return new Promise((resolve, reject) => {
		const abort = () => reject(signal.reason ?? abortReason("Aborted"));
		signal.addEventListener("abort", abort, { once: true });
		promise.then(() => resolve(), () => resolve()).finally(() => signal.removeEventListener("abort", abort));
	});
}

function chainAbortSignal(primary: AbortSignal, secondary?: AbortSignal): { signal: AbortSignal; cancel: () => void } {
	const controller = new AbortController();
	const abort = (source: AbortSignal) => controller.abort(source.reason ?? abortReason("Aborted"));
	const onPrimary = () => abort(primary);
	const onSecondary = () => secondary && abort(secondary);
	if (primary.aborted) abort(primary);
	else primary.addEventListener("abort", onPrimary, { once: true });
	if (secondary?.aborted) abort(secondary);
	else secondary?.addEventListener("abort", onSecondary, { once: true });
	return {
		signal: controller.signal,
		cancel: () => {
			primary.removeEventListener("abort", onPrimary);
			secondary?.removeEventListener("abort", onSecondary);
		}
	};
}
