import { afterEach, expect, test } from "bun:test";
import { fetchCustomFoodObject, listCustomFoodObjects } from "./custom-item-client";

// Implements DESIGN-008 SavedDataRepository custom Food Object client verification.

const originalFetch = globalThis.fetch;
const itemId = "11111111-1111-4111-8111-111111111111";

afterEach(() => {
	globalThis.fetch = originalFetch;
});

test("lists owner-scoped custom items as Daily Diet Food Objects", async () => {
	let requested = "";
	globalThis.fetch = (async (input, init) => {
		requested = String(input);
		expect(init?.credentials).toBe("include");
		return Response.json(envelope([customItem()]));
	}) as typeof fetch;

	const items = await listCustomFoodObjects();

	expect(requested).toBe("/api/v1/custom-items");
	expect(items).toEqual([{
		id: itemId,
		objectType: "custom_food_item",
		name: "Home shake",
		physicalState: "liquid",
		imageUrl: null,
		classifications: [],
		primaryFoodCategory: null,
		macros: { protein: 2, carbohydrates: 3, fat: 4 },
		macroBasis: "100ml",
		calories: 56
	}]);
});

test("loads one custom item for saved-diet edit hydration", async () => {
	globalThis.fetch = (async (input) => {
		expect(String(input)).toBe(`/api/v1/custom-items/${itemId}`);
		return Response.json(envelope(customItem()));
	}) as typeof fetch;

	expect((await fetchCustomFoodObject(itemId)).objectType).toBe("custom_food_item");
});

test("fails closed on malformed or unsuccessful custom-item responses", async () => {
	globalThis.fetch = (async () => Response.json({ data: { items: [{ ownerId: itemId }] } })) as typeof fetch;
	await expect(listCustomFoodObjects()).rejects.toThrow("custom foods could not be loaded");
	globalThis.fetch = (async () => new Response("", { status: 404 })) as typeof fetch;
	await expect(fetchCustomFoodObject(itemId)).rejects.toThrow("custom foods could not be loaded");
	globalThis.fetch = (async () => Response.json(envelope([{ ...customItem(), ownerId: itemId }]))) as typeof fetch;
	await expect(listCustomFoodObjects()).rejects.toThrow("custom foods could not be loaded");
	globalThis.fetch = (async () => Response.json(envelope([{ ...customItem(), densitySourceProvider: "usda", densitySourceFoodId: "171265" }]))) as typeof fetch;
	await expect(listCustomFoodObjects()).rejects.toThrow("custom foods could not be loaded");
});

function envelope(data: unknown): Record<string, unknown> {
	return { status: "ok", requestId: "task-297", data: Array.isArray(data) ? { items: data } : data };
}

function customItem(): Record<string, unknown> {
	return {
		id: itemId,
		name: "Home shake",
		physicalState: "liquid",
		prepTimeMinutes: 0,
		densityGramsPerMilliliter: 1,
		densitySourceKind: "manual",
		macrosPer100: { protein: 2, carbohydrates: 3, fat: 4 },
		micros: {},
		foodCategories: [],
		culinaryRoles: []
	};
}
