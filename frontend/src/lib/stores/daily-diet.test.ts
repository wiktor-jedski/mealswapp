import { afterEach, expect, test } from "bun:test";
import { get } from "svelte/store";
import { writable } from "svelte/store";

import {
	DailyDietClientError,
	type DailyDietApi
} from "../api/daily-diet-client";
import type { DailyDiet, DailyDietCreateRequest, DailyDietReplaceRequest } from "../api/generated";
import {
	createDailyDietController,
	createInitialDailyDietState,
	type DailyDietState
} from "./daily-diet";
import { addSubstitutionInput, buildSearchRequest, resetSearch, searchStore, setMode } from "./search";
import { selectedDailyDietId } from "./selected-daily-diet";

// Implements DESIGN-001 SearchView Daily Diet collection store/controller verification.
// Implements DESIGN-008 SavedDataRepository server-state reconciliation and user ownership projection.

function diet(id = "diet-1", name = "Training day"): DailyDiet {
	return {
		id,
		name,
		entries: [{ id: `${id}-entry`, mealId: "meal-1", quantity: 100, unit: "g", position: 0 }],
		aggregateMacros: { protein: 20, carbohydrates: 30, fat: 10, calories: 290 },
		createdAt: "2026-07-11T00:00:00Z",
		updatedAt: "2026-07-11T00:00:00Z"
	};
}

const replacement: DailyDietReplaceRequest = {
	name: "Updated day",
	entries: [{ mealId: "meal-1", quantity: 150, unit: "g", position: 0 }]
};

function fakeApi(overrides: Partial<DailyDietApi> = {}): DailyDietApi {
	return {
		listDailyDiets: async () => [],
		createDailyDiet: async () => diet("diet-created", "Created day"),
		replaceDailyDiet: async () => diet("diet-1", "Updated day"),
		deleteDailyDiet: async () => undefined,
		...overrides
	};
}

function testController(api: DailyDietApi): {
	controller: ReturnType<typeof createDailyDietController>;
	store: ReturnType<typeof writable<DailyDietState>>;
	selection: ReturnType<typeof writable<string | null>>;
} {
	const store = writable(createInitialDailyDietState());
	const selection = writable<string | null>(null);
	return { controller: createDailyDietController({ api, store, selectionStore: selection }), store, selection };
}

interface Deferred<T> {
	promise: Promise<T>;
	resolve(value: T): void;
	reject(error: unknown): void;
}

function deferred<T>(): Deferred<T> {
	let resolve!: (value: T) => void;
	let reject!: (error: unknown) => void;
	const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej; });
	return { promise, resolve, reject };
}

function abortable<T>(pending: Deferred<T>, signal?: AbortSignal): Promise<T> {
	if (signal?.aborted) return Promise.reject(signal.reason);
	return new Promise<T>((resolve, reject) => {
		const abort = () => reject(signal?.reason ?? new DOMException("Aborted", "AbortError"));
		signal?.addEventListener("abort", abort, { once: true });
		pending.promise.then(resolve, reject).finally(() => signal?.removeEventListener("abort", abort));
	});
}

const createRequest: DailyDietCreateRequest = {
	name: "Created day",
	entries: [{ mealId: "meal-1", quantity: 100, unit: "g", position: 0 }]
};

afterEach(() => {
	resetSearch();
	selectedDailyDietId.set(null);
});

test("load exposes loading, success, and empty server states", async () => {
	const source = [diet()];
	const { controller, store } = testController(fakeApi({ listDailyDiets: async () => source }));

	const pending = new Promise<DailyDiet[]>((resolve) => setTimeout(() => resolve(source), 0));
	const pendingController = testController(fakeApi({ listDailyDiets: async () => pending }));
	const loadingPromise = pendingController.controller.load();
	expect(get(pendingController.store)).toMatchObject({ status: "loading", loading: true, error: null });
	await loadingPromise;
	expect(get(pendingController.store)).toMatchObject({ status: "success", loading: false, collections: source });

	await controller.load();
	expect(get(store)).toMatchObject({ status: "success", collections: source });

	const empty = testController(fakeApi({ listDailyDiets: async () => [] }));
	await empty.controller.load();
	expect(get(empty.store)).toMatchObject({ status: "empty", loading: false, collections: [] });
});

test("load and mutations project only safe AppError data on failure", async () => {
	const failure = new DailyDietClientError(
		{ category: "security", code: "daily_diet_unavailable", message: "Saved daily diet is unavailable.", retryable: false },
		403
	);
	const { controller, store } = testController(fakeApi({ listDailyDiets: async () => { throw failure; } }));

	await expect(controller.load()).rejects.toBe(failure);
	const state = get(store);
	expect(state.status).toBe("error");
	expect(state.error).toEqual(failure.appError);
	expect(state.error).not.toHaveProperty("raw");
});

test("create and delete reconcile server state without optimistic unsafe writes", async () => {
	const created = diet("diet-created", "Created day");
	let deleted = false;
	const { controller, store } = testController(fakeApi({
		createDailyDiet: async () => created,
		deleteDailyDiet: async () => { deleted = true; }
	}));

	await controller.load();
	await controller.create({ name: created.name, entries: [{ mealId: "meal-1", quantity: 100, unit: "g", position: 0 }] });
	expect(get(store).collections).toEqual([created]);
	expect(get(store).collections[0]?.aggregateMacros).toEqual(created.aggregateMacros);

	const deleting = controller.remove(created.id);
	expect(get(store)).toMatchObject({ mutation: "deleting", loading: true, collections: [created] });
	await deleting;
	expect(deleted).toBe(true);
	expect(get(store)).toMatchObject({ status: "empty", mutation: "idle", loading: false, collections: [] });
});

test("replacement keeps the last authoritative DTO and macros until server success or failure", async () => {
	const original = diet();
	let rejectReplace!: (error: unknown) => void;
	const pending = new Promise<DailyDiet>((_resolve, reject) => { rejectReplace = reject; });
	const { controller, store } = testController(fakeApi({
		listDailyDiets: async () => [original],
		replaceDailyDiet: async () => pending
	}));
	await controller.load();

	const replacing = controller.replace(original.id, replacement, { csrfToken: "csrf" });
	expect(get(store).collections[0]?.name).toBe("Training day");
	expect(get(store).collections[0]?.aggregateMacros).toEqual(original.aggregateMacros);
	expect(get(store)).toMatchObject({ mutation: "replacing", loading: true });

	rejectReplace(new DailyDietClientError(
		{ category: "validation", code: "daily_diet_invalid_request", message: "Could not update diet.", retryable: false },
		400
	));
	await expect(replacing).rejects.toBeInstanceOf(DailyDietClientError);
	expect(get(store).collections).toEqual([original]);
	expect(get(store)).toMatchObject({ status: "error", mutation: "idle", loading: false });
});

test("successful replacement installs only the decoded server-derived DTO state", async () => {
	const original = diet();
	const serverDiet = {
		...diet("diet-1", "Updated day"),
		entries: [{ id: "server-entry", mealId: "meal-1", quantity: 150, unit: "g" as const, position: 0 }],
		aggregateMacros: { protein: 30, carbohydrates: 45, fat: 15, calories: 435 },
		updatedAt: "2026-07-11T00:01:00Z"
	};
	const { controller, store } = testController(fakeApi({
		listDailyDiets: async () => [original],
		replaceDailyDiet: async () => serverDiet
	}));
	await controller.load();
	await controller.replace(original.id, replacement);

	expect(get(store).collections).toEqual([serverDiet]);
	expect(get(store).collections[0]?.entries[0]?.id).toBe("server-entry");
	expect(get(store).collections[0]?.aggregateMacros.calories).toBe(435);
});

test("selection is reversible and does not overwrite Catalog or Substitution state", async () => {
	const selected = diet();
	const { controller, selection } = testController(fakeApi({ listDailyDiets: async () => [selected] }));
	await controller.load();

	setMode("substitution");
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" });
	controller.select(selected.id);

	expect(get(selection)).toBe(selected.id);
	const searchState = get(searchStore);
	expect(searchState.mode).toBe("substitution");
	if (searchState.mode === "substitution") {
		expect(searchState.substitutionInputs).toHaveLength(1);
	}

	controller.select(null);
	expect(get(selection)).toBeNull();
});

test("initial Daily Diet state is memory-only and contains no session persistence fields", () => {
	const state = createInitialDailyDietState();
	expect(state).toEqual({ collections: [], status: "idle", mutation: "idle", loading: false, error: null });
	expect(state).not.toHaveProperty("selectedId");
	expect(state).not.toHaveProperty("userId");
	expect(state).not.toHaveProperty("csrfToken");
	expect(state).not.toHaveProperty("accessToken");
});

test("lost create response reuses one caller-owned key and reconciles one server diet", async () => {
	const keys: string[] = [];
	let attempts = 0;
	const created = diet("diet-created", "Created day");
	const store = writable(createInitialDailyDietState());
	const controller = createDailyDietController({
		store,
		createIdempotencyKey: () => "daily-diet-operation-1",
		api: fakeApi({
			createDailyDiet: async (_request, options) => {
				keys.push(options.idempotencyKey);
				attempts += 1;
				if (attempts === 1) throw new DailyDietClientError(
					{ category: "network", code: "daily_diet_network_error", message: "lost response", retryable: true },
					0
				);
				return created;
			}
		})
	});

	await expect(controller.create(createRequest)).rejects.toBeInstanceOf(DailyDietClientError);
	await expect(controller.create(createRequest)).resolves.toEqual(created);
	expect(keys).toEqual(["daily-diet-operation-1", "daily-diet-operation-1"]);
	expect(get(store).collections).toEqual([created]);
});

test("pending create clicks share one request and cannot fork the active intent", async () => {
	let resolveCreate!: (value: DailyDiet) => void;
	const pending = new Promise<DailyDiet>((resolve) => { resolveCreate = resolve; });
	let calls = 0;
	const controller = createDailyDietController({
		store: writable(createInitialDailyDietState()),
		createIdempotencyKey: () => "daily-diet-operation-1",
		api: fakeApi({ createDailyDiet: async () => { calls += 1; return pending; } })
	});

	const first = controller.create(createRequest);
	const second = controller.create({ ...createRequest, name: "A changed pending click" });
	expect(second).toBe(first);
	expect(calls).toBe(1);
	resolveCreate(diet("diet-created", "Created day"));
	await expect(first).resolves.toBeDefined();
});

test("draft edit, success, and clear rotate or clear retained create ownership", async () => {
	const keys = ["daily-diet-operation-1", "daily-diet-operation-2", "daily-diet-operation-3", "daily-diet-operation-4"];
	const used: string[] = [];
	let fail = true;
	const controller = createDailyDietController({
		store: writable(createInitialDailyDietState()),
		createIdempotencyKey: () => keys.shift()!,
		api: fakeApi({
			createDailyDiet: async (_request, options) => {
				used.push(options.idempotencyKey);
				if (fail) throw new DailyDietClientError(
					{ category: "network", code: "daily_diet_network_error", message: "lost response", retryable: true },
					0
				);
				return diet("diet-created", "Created day");
			}
		})
	});

	await expect(controller.create(createRequest)).rejects.toBeDefined();
	controller.discardCreateIntent();
	fail = false;
	await controller.create(createRequest);
	await controller.create(createRequest);
	controller.clear();
	await controller.create(createRequest);
	expect(used).toEqual([
		"daily-diet-operation-1",
		"daily-diet-operation-2",
		"daily-diet-operation-3",
		"daily-diet-operation-4"
	]);
});

test("secure-random failure fails safely before API I/O and keeps retry data memory-only", async () => {
	let calls = 0;
	const store = writable(createInitialDailyDietState());
	const controller = createDailyDietController({
		store,
		createIdempotencyKey: () => { throw new DailyDietClientError(
			{ category: "security", code: "secure_random_unavailable", message: "A secure Daily Diet request could not be created. Please try again.", retryable: true },
			0
		); },
		api: fakeApi({ createDailyDiet: async () => { calls += 1; return diet(); } })
	});

	await expect(controller.create(createRequest)).rejects.toMatchObject({ appError: { code: "secure_random_unavailable" } });
	expect(calls).toBe(0);
	expect(get(store)).toMatchObject({ status: "error", mutation: "idle", loading: false, error: { code: "secure_random_unavailable" } });
	expect(JSON.stringify(get(store))).not.toContain("daily-diet-operation");
});

test("create keys never touch browser storage and clear across identity lifecycle", async () => {
	let storageAccesses = 0;
	const originalWindow = globalThis.window;
	Object.defineProperty(globalThis, "window", {
		configurable: true,
		value: {
			get localStorage() { storageAccesses += 1; throw new Error("must not access localStorage"); },
			get sessionStorage() { storageAccesses += 1; throw new Error("must not access sessionStorage"); }
		}
	});
	const keys = ["daily-diet-account-a", "daily-diet-account-b"];
	const seen: string[] = [];
	const controller = createDailyDietController({
		store: writable(createInitialDailyDietState()),
		createIdempotencyKey: () => keys.shift()!,
		api: fakeApi({
			createDailyDiet: async (_request, options) => {
				seen.push(options.idempotencyKey);
				throw new DailyDietClientError(
					{ category: "network", code: "daily_diet_network_error", message: "lost response", retryable: true },
					0
				);
			}
		})
	});
	try {
		await expect(controller.create(createRequest)).rejects.toBeDefined();
		controller.clear();
		await expect(controller.create(createRequest)).rejects.toBeDefined();
		expect(seen).toEqual(["daily-diet-account-a", "daily-diet-account-b"]);
		expect(storageAccesses).toBe(0);
	} finally {
		Object.defineProperty(globalThis, "window", { configurable: true, value: originalWindow });
	}
});

test("load/load aborts the older read and only the newer server snapshot is installed", async () => {
	const first = deferred<DailyDiet[]>();
	const second = deferred<DailyDiet[]>();
	const signals: AbortSignal[] = [];
	let calls = 0;
	const { controller, store } = testController(fakeApi({
		listDailyDiets: async (signal) => {
			signals.push(signal!);
			return abortable(calls++ === 0 ? first : second, signal);
		}
	}));

	const older = controller.load();
	const olderOutcome = older.then(() => null, (error: unknown) => error);
	const newer = controller.load();
	expect(signals[0]?.aborted).toBe(true);
	second.resolve([diet("diet-new")]);
	await expect(newer).resolves.toEqual([diet("diet-new")]);
	expect(await olderOutcome).toMatchObject({ name: "AbortError" });
	first.resolve([diet("diet-old")]);
	expect(get(store).collections).toEqual([diet("diet-new")]);
});

test("load/create cancels the stale read and preserves the confirmed create", async () => {
	const read = deferred<DailyDiet[]>();
	const created = diet("diet-created", "Created day");
	let readSignal: AbortSignal | undefined;
	const { controller, store } = testController(fakeApi({
		listDailyDiets: async (signal) => { readSignal = signal; return abortable(read, signal); },
		createDailyDiet: async () => created
	}));

	const loading = controller.load();
	const loadingOutcome = loading.then(() => null, (error: unknown) => error);
	await expect(controller.create(createRequest)).resolves.toEqual(created);
	expect(readSignal?.aborted).toBe(true);
	expect(await loadingOutcome).toMatchObject({ name: "AbortError" });
	read.resolve([]);
	expect(get(store).collections).toEqual([created]);
});

test("create/load queues the read behind mutation confirmation without losing the write", async () => {
	const creation = deferred<DailyDiet>();
	const listing = deferred<DailyDiet[]>();
	const created = diet("diet-created", "Created day");
	let listCalls = 0;
	const { controller, store } = testController(fakeApi({
		createDailyDiet: async () => creation.promise,
		listDailyDiets: async () => { listCalls += 1; return listing.promise; }
	}));

	const creating = controller.create(createRequest);
	const loading = controller.load();
	expect(listCalls).toBe(0);
	creation.resolve(created);
	await creating;
	await Promise.resolve();
	expect(listCalls).toBe(1);
	listing.resolve([created]);
	await loading;
	expect(get(store).collections).toEqual([created]);
});

test("replace/select/failure keeps newer selection and the last authoritative diet", async () => {
	const failure = new DailyDietClientError(
		{ category: "validation", code: "daily_diet_invalid_request", message: "Could not update diet.", retryable: false },
		400
	);
	const replacing = deferred<DailyDiet>();
	const original = diet("diet-1");
	const other = diet("diet-2");
	const { controller, store, selection } = testController(fakeApi({
		listDailyDiets: async () => [original, other],
		replaceDailyDiet: async () => replacing.promise
	}));
	await controller.load();
	controller.select(original.id);
	const pending = controller.replace(original.id, replacement);
	controller.select(other.id);
	replacing.reject(failure);
	await expect(pending).rejects.toBe(failure);
	expect(get(selection)).toBe(other.id);
	expect(get(store).collections).toEqual([original, other]);
});

test("replace/load serializes the read after authoritative replacement", async () => {
	const replacing = deferred<DailyDiet>();
	const serverDiet = { ...diet(), name: "Server update", aggregateMacros: { protein: 40, carbohydrates: 50, fat: 20, calories: 540 } };
	let listCalls = 0;
	const { controller, store } = testController(fakeApi({
		listDailyDiets: async () => { listCalls += 1; return listCalls === 1 ? [diet()] : [serverDiet]; },
		replaceDailyDiet: async () => replacing.promise
	}));
	await controller.load();
	const pendingReplace = controller.replace("diet-1", replacement);
	const pendingLoad = controller.load();
	expect(listCalls).toBe(1);
	replacing.resolve(serverDiet);
	await pendingReplace;
	await pendingLoad;
	expect(get(store).collections).toEqual([serverDiet]);
});

// Verifies IT-ARCH-004-006, ARCH-004, DESIGN-001 SearchView, and
// SW-REQ-006/SW-REQ-043 authoritative deletion/read collaboration.
test("delete/load serializes the read and cannot resurrect a confirmed deletion", async () => {
	const deletion = deferred<void>();
	const removed = diet("diet-1");
	const retained = diet("diet-2");
	let listCalls = 0;
	const { controller, store } = testController(fakeApi({
		listDailyDiets: async () => { listCalls += 1; return listCalls === 1 ? [removed, retained] : [retained]; },
		deleteDailyDiet: async () => deletion.promise
	}));
	await controller.load();
	const pendingDelete = controller.remove(removed.id);
	const pendingLoad = controller.load();
	deletion.resolve();
	await pendingDelete;
	await pendingLoad;
	expect(get(store).collections).toEqual([retained]);
});

test("clear/logout aborts read and mutation lifecycles and ignores late completions", async () => {
	const read = deferred<DailyDiet[]>();
	let readSignal: AbortSignal | undefined;
	const reading = testController(fakeApi({ listDailyDiets: async (signal) => { readSignal = signal; return read.promise; } }));
	const pendingRead = reading.controller.load();
	reading.controller.clear();
	expect(readSignal?.aborted).toBe(true);
	read.resolve([diet("late-read")]);
	await pendingRead;
	expect(get(reading.store)).toEqual(createInitialDailyDietState());

	const mutation = deferred<DailyDiet>();
	let mutationSignal: AbortSignal | undefined;
	const mutating = testController(fakeApi({ createDailyDiet: async (_request, options) => { mutationSignal = options.signal; return mutation.promise; } }));
	const pendingMutation = mutating.controller.create(createRequest);
	mutating.controller.clear();
	expect(mutationSignal?.aborted).toBe(true);
	mutation.resolve(diet("late-create"));
	await pendingMutation;
	expect(get(mutating.store)).toEqual(createInitialDailyDietState());
});

test("duplicate activation shares create and rejects overlapping distinct mutations", async () => {
	const creation = deferred<DailyDiet>();
	let createCalls = 0;
	let replaceCalls = 0;
	const { controller } = testController(fakeApi({
		createDailyDiet: async () => { createCalls += 1; return creation.promise; },
		replaceDailyDiet: async () => { replaceCalls += 1; return diet(); }
	}));
	const first = controller.create(createRequest);
	expect(controller.create(createRequest)).toBe(first);
	await expect(controller.replace("diet-1", replacement)).rejects.toMatchObject({ appError: { code: "daily_diet_mutation_in_progress" } });
	expect(createCalls).toBe(1);
	expect(replaceCalls).toBe(0);
	creation.resolve(diet("diet-created"));
	await first;
});

test("caller abort propagates to reads and mutations and leaves authoritative state unchanged", async () => {
	const original = diet();
	const read = deferred<DailyDiet[]>();
	const replacementResult = deferred<DailyDiet>();
	const readAbort = new AbortController();
	const mutationAbort = new AbortController();
	let receivedReadSignal: AbortSignal | undefined;
	let receivedMutationSignal: AbortSignal | undefined;
	let listCalls = 0;
	const { controller, store } = testController(fakeApi({
		listDailyDiets: async (signal) => {
			listCalls += 1;
			if (listCalls === 1) return [original];
			receivedReadSignal = signal;
			return abortable(read, signal);
		},
		replaceDailyDiet: async (_id, _request, options) => {
			receivedMutationSignal = options.signal;
			return abortable(replacementResult, options.signal);
		}
	}));
	await controller.load();
	const pendingRead = controller.load(readAbort.signal);
	readAbort.abort(new DOMException("caller read abort", "AbortError"));
	await expect(pendingRead).rejects.toMatchObject({ name: "AbortError" });
	expect(receivedReadSignal?.aborted).toBe(true);

	const pendingMutation = controller.replace(original.id, replacement, { signal: mutationAbort.signal });
	mutationAbort.abort(new DOMException("caller mutation abort", "AbortError"));
	await expect(pendingMutation).rejects.toMatchObject({ name: "AbortError" });
	expect(receivedMutationSignal?.aborted).toBe(true);
	expect(get(store).collections).toEqual([original]);
});

test("caller abort promptly cancels a read queued behind a mutation", async () => {
	const creation = deferred<DailyDiet>();
	const abortController = new AbortController();
	let listCalls = 0;
	const { controller } = testController(fakeApi({
		createDailyDiet: async () => creation.promise,
		listDailyDiets: async () => { listCalls += 1; return []; }
	}));
	const creating = controller.create(createRequest);
	const loading = controller.load(abortController.signal);
	abortController.abort(new DOMException("caller queued read abort", "AbortError"));
	await expect(loading).rejects.toMatchObject({ name: "AbortError" });
	expect(listCalls).toBe(0);
	creation.resolve(diet("diet-created"));
	await creating;
});

test("pre-aborted read settles without API I/O and a later read remains usable", async () => {
	const aborted = new AbortController();
	aborted.abort(new DOMException("caller read abort", "AbortError"));
	let listCalls = 0;
	const expected = [diet()];
	const { controller, store } = testController(fakeApi({
		listDailyDiets: async () => { listCalls += 1; return expected; }
	}));

	await expect(controller.load(aborted.signal)).rejects.toMatchObject({ name: "AbortError" });
	expect(listCalls).toBe(0);
	expect(get(store)).toEqual(createInitialDailyDietState());

	await expect(controller.load()).resolves.toEqual(expected);
	expect(listCalls).toBe(1);
	expect(get(store)).toMatchObject({ status: "success", loading: false, collections: expected });
});

test("pre-aborted mutations settle without API I/O and later mutations remain usable", async () => {
	const aborted = new AbortController();
	aborted.abort(new DOMException("caller mutation abort", "AbortError"));
	let createCalls = 0;
	let replaceCalls = 0;
	let deleteCalls = 0;
	const created = diet("diet-created", "Created day");
	const replaced = diet("diet-created", "Updated day");
	const { controller, store } = testController(fakeApi({
		createDailyDiet: async () => { createCalls += 1; return created; },
		replaceDailyDiet: async () => { replaceCalls += 1; return replaced; },
		deleteDailyDiet: async () => { deleteCalls += 1; }
	}));

	await expect(controller.create(createRequest, { signal: aborted.signal })).rejects.toMatchObject({ name: "AbortError" });
	expect(createCalls).toBe(0);
	expect(get(store)).toEqual(createInitialDailyDietState());
	await expect(controller.create(createRequest)).resolves.toEqual(created);

	await expect(controller.replace(created.id, replacement, { signal: aborted.signal })).rejects.toMatchObject({ name: "AbortError" });
	expect(replaceCalls).toBe(0);
	expect(get(store)).toMatchObject({ mutation: "idle", loading: false, collections: [created] });
	await expect(controller.replace(created.id, replacement)).resolves.toEqual(replaced);

	await expect(controller.remove(replaced.id, { signal: aborted.signal })).rejects.toMatchObject({ name: "AbortError" });
	expect(deleteCalls).toBe(0);
	expect(get(store)).toMatchObject({ mutation: "idle", loading: false, collections: [replaced] });
	await expect(controller.remove(replaced.id)).resolves.toBeUndefined();
	expect(get(store)).toMatchObject({ status: "empty", mutation: "idle", loading: false, collections: [] });
});

test("synchronous mutation execution failure settles ownership and permits retry", async () => {
	let attempts = 0;
	const recovered = diet("diet-1", "Recovered day");
	const { controller, store } = testController(fakeApi({
		replaceDailyDiet: () => {
			attempts += 1;
			if (attempts === 1) throw new Error("synchronous failure");
			return Promise.resolve(recovered);
		}
	}));

	await expect(controller.replace("diet-1", replacement)).rejects.toThrow("synchronous failure");
	expect(get(store)).toMatchObject({ status: "error", mutation: "idle", loading: false });
	await expect(controller.replace("diet-1", replacement)).resolves.toEqual(recovered);
	expect(attempts).toBe(2);
	expect(get(store)).toMatchObject({ status: "success", mutation: "idle", loading: false, collections: [recovered] });
});

test("authoritative selection survives mode round trips and drives one emitted diet id", async () => {
	const first = diet("diet-1");
	const second = diet("diet-2");
	const store = writable(createInitialDailyDietState());
	const controller = createDailyDietController({
		store,
		selectionStore: selectedDailyDietId,
		api: fakeApi({ listDailyDiets: async () => [first, second] })
	});
	await controller.load();
	controller.select(first.id);
	expect(get(selectedDailyDietId)).toBe(first.id);
	setMode("daily_diet_alternative");
	expect(buildSearchRequest(get(searchStore)).dailyDietId).toBe(first.id);
	setMode("catalog");
	setMode("substitution");
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" });
	expect(get(selectedDailyDietId)).toBe(first.id);
	expect(buildSearchRequest(get(searchStore))).not.toHaveProperty("dailyDietId");
	setMode("daily_diet_alternative");
	expect(buildSearchRequest(get(searchStore)).dailyDietId).toBe(first.id);
	controller.select(second.id);
	expect(buildSearchRequest(get(searchStore)).dailyDietId).toBe(second.id);
});

// Verifies IT-ARCH-004-006, ARCH-004, DESIGN-001 SearchView, and
// SW-REQ-006/SW-REQ-043 selected-diet identity reconciliation.
test("reload, deletion, empty state, and identity clear reconcile authoritative selection", async () => {
	const first = diet("diet-1");
	const second = diet("diet-2");
	let server = [first, second];
	const store = writable(createInitialDailyDietState());
	const controller = createDailyDietController({
		store,
		selectionStore: selectedDailyDietId,
		api: fakeApi({
			listDailyDiets: async () => server,
			deleteDailyDiet: async (id) => { server = server.filter((item) => item.id !== id); }
		})
	});
	await controller.load();
	controller.select(first.id);
	await controller.remove(second.id);
	expect(get(selectedDailyDietId)).toBe(first.id);
	server = [first, second];
	await controller.load();
	controller.select(second.id);
	await controller.remove(second.id);
	expect(get(selectedDailyDietId)).toBeNull();
	controller.select(first.id);
	server = [];
	await controller.load();
	expect(get(selectedDailyDietId)).toBeNull();
	server = [first];
	await controller.load();
	controller.select(first.id);
	server = [second];
	await controller.load();
	expect(get(selectedDailyDietId)).toBeNull();
	controller.select(second.id);
	controller.clear();
	expect(get(selectedDailyDietId)).toBeNull();
	expect(get(store)).toEqual(createInitialDailyDietState());
});
