import { afterEach, expect, test } from "bun:test";
import { get } from "svelte/store";
import { writable } from "svelte/store";

import {
	DailyDietClientError,
	type DailyDietApi
} from "../api/daily-diet-client";
import type { DailyDiet, DailyDietReplaceRequest } from "../api/generated";
import {
	createDailyDietController,
	createInitialDailyDietState,
	type DailyDietState
} from "./daily-diet";
import { addSubstitutionInput, resetSearch, searchStore, setMode } from "./search";

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

function testController(api: DailyDietApi): { controller: ReturnType<typeof createDailyDietController>; store: ReturnType<typeof writable<DailyDietState>> } {
	const store = writable(createInitialDailyDietState());
	return { controller: createDailyDietController({ api, store }), store };
}

afterEach(() => {
	resetSearch();
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
	expect(get(store)).toMatchObject({ status: "success", collections: source, selectedId: null });

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

test("replace optimistically updates only a reversible matching edit and rolls back on error", async () => {
	const original = diet();
	let rejectReplace!: (error: unknown) => void;
	const pending = new Promise<DailyDiet>((_resolve, reject) => { rejectReplace = reject; });
	const { controller, store } = testController(fakeApi({
		listDailyDiets: async () => [original],
		replaceDailyDiet: async () => pending
	}));
	await controller.load();

	const replacing = controller.replace(original.id, replacement, { csrfToken: "csrf" });
	expect(get(store).collections[0]?.name).toBe("Updated day");
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

test("successful replacement replaces the optimistic projection with server-derived DTO state", async () => {
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
	const { controller, store } = testController(fakeApi({ listDailyDiets: async () => [selected] }));
	await controller.load();

	setMode("substitution");
	addSubstitutionInput({ foodObjectId: "food-1", quantity: 100, unit: "g" });
	controller.select(selected.id);

	expect(get(store).selectedId).toBe(selected.id);
	const searchState = get(searchStore);
	expect(searchState.mode).toBe("substitution");
	if (searchState.mode === "substitution") {
		expect(searchState.substitutionInputs).toHaveLength(1);
	}

	controller.select(null);
	expect(get(store).selectedId).toBeNull();
});

test("initial Daily Diet state is memory-only and contains no session persistence fields", () => {
	const state = createInitialDailyDietState();
	expect(state).toEqual({ collections: [], selectedId: null, status: "idle", mutation: "idle", loading: false, error: null });
	expect(state).not.toHaveProperty("userId");
	expect(state).not.toHaveProperty("csrfToken");
	expect(state).not.toHaveProperty("accessToken");
});
