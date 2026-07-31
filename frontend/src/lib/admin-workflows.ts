import type { AdminDeletionSummary, AdminItem, AdminItemRequest } from "./api/generated";

// Implements DESIGN-009 ItemCurator and UserAdminPanel client validation without replacing server authority.

/** Text-backed administration item form state before server-contract parsing. */
export interface AdminItemForm {
	name: string;
	physicalState: "solid" | "liquid";
	prepTimeMinutes: string;
	averageUnitWeightGrams: string;
	averageServingVolumeMilliliters: string;
	protein: string;
	carbohydrates: string;
	fat: string;
	density: string;
	densitySourceProvider: string;
	densitySourceFoodId: string;
	densitySourceKind: "" | "imported" | "manual" | "estimated";
	micros: string;
	foodCategoryIds: string[];
	culinaryRoleIds: string[];
	allergenKeys: string[];
	imageUrl: string;
}

/** Converts form text to the generated item request or one actionable validation error. */
export function parseAdminItemForm(form: AdminItemForm): { request?: AdminItemRequest; error?: string } {
	const name = form.name.trim().replace(/\s+/gu, " ");
	if (!name || name.length > 200) return { error: "Enter an item name of at most 200 characters." };
	const protein = number(form.protein);
	const carbohydrates = number(form.carbohydrates);
	const fat = number(form.fat);
	if ([protein, carbohydrates, fat].some((value) => value === undefined || value < 0 || value > 99_999_999.9999)) return { error: "Macros must be bounded non-negative numbers." };
	if (form.physicalState === "solid" && protein! + carbohydrates! + fat! > 100) return { error: "Solid macros cannot total more than 100 per 100 g." };
	let micros: Record<string, number>;
	try {
		const parsed = JSON.parse(form.micros || "{}") as unknown;
		if (!record(parsed) || Object.keys(parsed).length > 200 || Object.keys(parsed).some((key) => !key || key.length > 120 || key.includes("\0") || typeof parsed[key] !== "number" || !Number.isFinite(parsed[key]) || parsed[key] < 0 || parsed[key] > 99_999_999.9999)) throw new Error();
		micros = parsed as Record<string, number>;
	} catch { return { error: "Micronutrients must be a JSON object with non-negative numeric values." }; }
	const prepTimeMinutes = optionalNumber(form.prepTimeMinutes, true);
	const averageUnitWeightGrams = optionalNumber(form.averageUnitWeightGrams);
	const averageServingVolumeMilliliters = optionalNumber(form.averageServingVolumeMilliliters);
	if (prepTimeMinutes === null || averageUnitWeightGrams === null || averageServingVolumeMilliliters === null) return { error: "Preparation and measure values must be bounded positive numbers (preparation may be zero)." };
	if (form.foodCategoryIds.length > 100 || form.culinaryRoleIds.length > 100 || !uniqueUuids(form.foodCategoryIds) || !uniqueUuids(form.culinaryRoleIds)) return { error: "Select at most 100 unique valid classifications of each kind." };
	if (form.allergenKeys.length > 100 || new Set(form.allergenKeys).size !== form.allergenKeys.length || form.allergenKeys.some((key) => !/^[a-z][a-z0-9_]{0,119}$/.test(key))) return { error: "Select at most 100 unique canonical allergens." };
	const imageUrl = form.imageUrl.trim();
	if (imageUrl && (!safeUriReference(imageUrl) || imageUrl.length > 2048 || imageUrl.includes("\0"))) return { error: "Enter a valid HTTP(S) or relative image URL of at most 2048 characters." };
	const request: AdminItemRequest = {
		name,
		physicalState: form.physicalState,
		...(prepTimeMinutes === undefined ? {} : { prepTimeMinutes }),
		...(averageUnitWeightGrams === undefined ? {} : { averageUnitWeightGrams }),
		macrosPer100: { protein: protein!, carbohydrates: carbohydrates!, fat: fat! },
		micros,
		foodCategoryIds: form.foodCategoryIds,
		culinaryRoleIds: form.culinaryRoleIds,
		allergenKeys: form.allergenKeys,
		...(imageUrl ? { imageUrl } : {})
	};
	if (form.physicalState === "liquid") {
		const density = number(form.density);
		if (density === undefined || density <= 0 || density > 99_999_999.9999) return { error: "Liquid items require a positive density." };
		const densitySourceProvider = form.densitySourceProvider.trim();
		const densitySourceFoodId = form.densitySourceFoodId.trim();
		const densitySourceKind = form.densitySourceKind || "manual";
		if (densitySourceProvider.length > 200 || densitySourceFoodId.length > 200 || densitySourceProvider.includes("\0") || densitySourceFoodId.includes("\0")) return { error: "Density provenance fields must be at most 200 characters." };
		if (densitySourceKind === "imported" && (!densitySourceFoodId || !["usda", "openfoodfacts"].includes(densitySourceProvider))) return { error: "Imported density requires a trusted provider and source food ID." };
		request.densityGramsPerMilliliter = density;
		request.densitySourceKind = densitySourceKind;
		if (densitySourceProvider) request.densitySourceProvider = densitySourceProvider;
		if (densitySourceFoodId) request.densitySourceFoodId = densitySourceFoodId;
		if (averageServingVolumeMilliliters !== undefined) request.averageServingVolumeMilliliters = averageServingVolumeMilliliters;
	}
	return { request };
}

/** Compares an authoritative global item with one normalized mutation snapshot. */
export function adminItemMatchesRequest(item: AdminItem, request: AdminItemRequest): boolean {
	const itemCategoryIds = item.foodCategoryIds ?? item.foodCategories.map(({ id }) => id);
	const itemRoleIds = item.culinaryRoleIds ?? item.culinaryRoles.map(({ id }) => id);
	return item.name === request.name
		&& item.physicalState === request.physicalState
		&& item.prepTimeMinutes === (request.prepTimeMinutes ?? 0)
		&& item.averageUnitWeightGrams === request.averageUnitWeightGrams
		&& item.averageServingVolumeMilliliters === request.averageServingVolumeMilliliters
		&& item.densityGramsPerMilliliter === request.densityGramsPerMilliliter
		&& item.densitySourceProvider === request.densitySourceProvider
		&& item.densitySourceFoodId === request.densitySourceFoodId
		&& item.densitySourceKind === request.densitySourceKind
		&& item.imageUrl === request.imageUrl
		&& item.macrosPer100.protein === request.macrosPer100.protein
		&& item.macrosPer100.carbohydrates === request.macrosPer100.carbohydrates
		&& item.macrosPer100.fat === request.macrosPer100.fat
		&& equalRecord(item.micros, request.micros)
		&& equalSet(itemCategoryIds, request.foodCategoryIds ?? [])
		&& equalSet(itemRoleIds, request.culinaryRoleIds ?? [])
		&& equalSet(item.allergenKeys, request.allergenKeys);
}

/** Mirrors the documented deletion retry eligibility rule for control visibility. */
export function deletionRetryEligible(deletion: AdminDeletionSummary | undefined): boolean {
	return deletion?.status === "failed" && (deletion.failureCategory === "permanent" || deletion.failureCategory === "unknown" || (deletion.failureCategory === "transient" && deletion.retryCount >= 3));
}

/** Creates a memory-only key for one deliberate manual-item create intent. */
export function newAdminItemKey(): string {
	return `admin-item-${crypto.randomUUID()}`;
}

function number(value: string): number | undefined {
	if (!value.trim()) return undefined;
	const parsed = Number(value);
	return Number.isFinite(parsed) ? parsed : undefined;
}

function optionalNumber(value: string, integer = false): number | null | undefined {
	if (!value.trim()) return undefined;
	const parsed = number(value);
	return parsed === undefined || parsed < 0 || parsed > 99_999_999.9999 || (!integer && parsed === 0) || (integer && !Number.isInteger(parsed)) ? null : parsed;
}

function uniqueUuids(values: string[]): boolean {
	return new Set(values).size === values.length && values.every((value) => /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(value));
}

function safeUriReference(value: string): boolean {
	try {
		const parsed = new URL(value, "https://mealswapp.invalid");
		return !parsed.protocol || parsed.protocol === "http:" || parsed.protocol === "https:";
	} catch { return false; }
}

function equalRecord(left: Record<string, number>, right: Record<string, number>): boolean {
	const keys = Object.keys(left);
	return keys.length === Object.keys(right).length && keys.every((key) => left[key] === right[key]);
}

function equalSet(left: string[], right: string[]): boolean {
	return JSON.stringify([...left].sort()) === JSON.stringify([...right].sort());
}

function record(value: unknown): value is Record<string, unknown> { return typeof value === "object" && value !== null && !Array.isArray(value); }
