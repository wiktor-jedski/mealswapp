import {
	buildCustomItemListRequestInit,
	buildCustomItemUrl,
	CUSTOM_ITEMS_ENDPOINT,
	type ClassificationSummary,
	type CustomItem,
	type FoodObject,
	type FoodObjectType
} from "./generated";

// Implements DESIGN-008 ProfileController owner-scoped custom-item selection.

const MAX_RESPONSE_BYTES = 1024 * 1024;

/** Safe error exposed by private custom-item reads. */
export class CustomItemClientError extends Error {
	constructor() {
		super("Your custom foods could not be loaded. Please try again.");
		this.name = "CustomItemClientError";
	}
}

/** Food Object card data accepted by Daily Diet, including private custom foods. */
export type DailyDietFoodObject = Omit<FoodObject, "objectType"> & { objectType: FoodObjectType };

/** Lists active custom foods owned by the authenticated user. */
export async function listCustomFoodObjects(signal?: AbortSignal): Promise<DailyDietFoodObject[]> {
	const response = await fetch(CUSTOM_ITEMS_ENDPOINT, buildCustomItemListRequestInit({ signal }));
	const value = await responseValue(response);
	if (!record(value) || value.status !== "ok" || typeof value.requestId !== "string" || !record(value.data) || !Array.isArray(value.data.items)) {
		throw new CustomItemClientError();
	}
	return value.data.items.map(decodeCustomItem).map(toFoodObject);
}

/** Loads one active custom food through the authenticated owner-scoped endpoint. */
export async function fetchCustomFoodObject(itemId: string, signal?: AbortSignal): Promise<DailyDietFoodObject> {
	const response = await fetch(buildCustomItemUrl(itemId), buildCustomItemListRequestInit({ signal }));
	const value = await responseValue(response);
	if (!record(value) || value.status !== "ok" || typeof value.requestId !== "string") throw new CustomItemClientError();
	return toFoodObject(decodeCustomItem(value.data));
}

function decodeCustomItem(value: unknown): CustomItem {
	if (!record(value) || !uuid(value.id) || typeof value.name !== "string" || value.name.trim() === "" ||
		"ownerId" in value || "userId" in value || "OwnerID" in value || "UserID" in value ||
		"densitySourceProvider" in value || "densitySourceFoodId" in value ||
		(value.physicalState !== "solid" && value.physicalState !== "liquid") || !record(value.macrosPer100) ||
		!finiteNonnegative(value.macrosPer100.protein) || !finiteNonnegative(value.macrosPer100.carbohydrates) ||
		!finiteNonnegative(value.macrosPer100.fat) || !Array.isArray(value.foodCategories) || !Array.isArray(value.culinaryRoles)) {
		throw new CustomItemClientError();
	}
	const foodCategories = value.foodCategories.map(classification);
	const culinaryRoles = value.culinaryRoles.map(classification);
	return { ...value, foodCategories, culinaryRoles } as unknown as CustomItem;
}

function toFoodObject(item: CustomItem): DailyDietFoodObject {
	const classifications = [...item.foodCategories, ...item.culinaryRoles];
	return {
		id: item.id,
		objectType: "custom_food_item",
		name: item.name,
		physicalState: item.physicalState,
		imageUrl: item.imageUrl ?? null,
		classifications,
		primaryFoodCategory: item.foodCategories[0] ?? null,
		macros: item.macrosPer100,
		macroBasis: item.physicalState === "liquid" ? "100ml" : "100g",
		calories: item.macrosPer100.protein * 4 + item.macrosPer100.carbohydrates * 4 + item.macrosPer100.fat * 9
	};
}

function classification(value: unknown): ClassificationSummary {
	if (!record(value) || !uuid(value.id) || typeof value.name !== "string" ||
		(value.kind !== "food_category" && value.kind !== "culinary_role")) throw new CustomItemClientError();
	return { id: value.id, name: value.name, kind: value.kind };
}

async function responseValue(response: Response): Promise<unknown> {
	if (!response.ok) throw new CustomItemClientError();
	try {
		const declared = response.headers.get("content-length");
		if (declared !== null && /^\d+$/.test(declared) && Number(declared) > MAX_RESPONSE_BYTES) throw new CustomItemClientError();
		const reader = response.body?.getReader();
		if (!reader) throw new CustomItemClientError();
		const chunks: Uint8Array[] = [];
		let size = 0;
		try {
			while (true) {
				const { value, done } = await reader.read();
				if (done) break;
				size += value.byteLength;
				if (size > MAX_RESPONSE_BYTES) {
					await reader.cancel();
					throw new CustomItemClientError();
				}
				chunks.push(value);
			}
		} finally {
			reader.releaseLock();
		}
		const bytes = new Uint8Array(size);
		let offset = 0;
		for (const chunk of chunks) {
			bytes.set(chunk, offset);
			offset += chunk.byteLength;
		}
		return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)) as unknown;
	} catch {
		throw new CustomItemClientError();
	}
}

function record(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null && !Array.isArray(value);
}

function finiteNonnegative(value: unknown): value is number {
	return typeof value === "number" && Number.isFinite(value) && value >= 0;
}

function uuid(value: unknown): value is string {
	return typeof value === "string" && /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(value);
}
