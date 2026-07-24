import { expect, test } from "bun:test";
import { deletionRetryEligible, newAdminItemKey, parseAdminItemForm, type AdminItemForm } from "./admin-workflows";

// Implements DESIGN-009 ItemCurator and UserAdminPanel validation and legal-retry verification.

const valid = (overrides: Partial<AdminItemForm> = {}): AdminItemForm => ({
	name: "Broth", physicalState: "solid", prepTimeMinutes: "", averageUnitWeightGrams: "", averageServingVolumeMilliliters: "", protein: "10", carbohydrates: "20", fat: "5", density: "", densitySourceProvider: "", densitySourceFoodId: "", densitySourceKind: "", micros: "{\"sodium\":1}", foodCategoryIds: [], culinaryRoleIds: [], imageUrl: "", ...overrides
});

test("builds generated-contract solid and liquid item requests", () => {
	expect(parseAdminItemForm(valid()).request).toEqual({ name: "Broth", physicalState: "solid", macrosPer100: { protein: 10, carbohydrates: 20, fat: 5 }, micros: { sodium: 1 }, foodCategoryIds: [], culinaryRoleIds: [] });
	expect(parseAdminItemForm(valid({ physicalState: "liquid", density: "1.02" })).request).toMatchObject({ physicalState: "liquid", densityGramsPerMilliliter: 1.02, densitySourceKind: "manual" });
});

test("round-trips image, preparation, measures, and imported density provenance", () => {
	expect(parseAdminItemForm(valid({
		physicalState: "liquid", prepTimeMinutes: "12", averageUnitWeightGrams: "250", averageServingVolumeMilliliters: "240", density: "1.03",
		densitySourceProvider: "usda", densitySourceFoodId: "171265", densitySourceKind: "imported", imageUrl: "https://images.example.test/milk.png"
	})).request).toMatchObject({
		prepTimeMinutes: 12, averageUnitWeightGrams: 250, averageServingVolumeMilliliters: 240, densityGramsPerMilliliter: 1.03,
		densitySourceProvider: "usda", densitySourceFoodId: "171265", densitySourceKind: "imported", imageUrl: "https://images.example.test/milk.png"
	});
});

test("rejects invalid macros, micronutrients, names, and missing liquid density", () => {
	for (const form of [valid({ name: " " }), valid({ protein: "-1" }), valid({ fat: "80", carbohydrates: "30" }), valid({ micros: "[]" }), valid({ micros: "{\"iron\":-1}" }), valid({ physicalState: "liquid" })]) {
		expect(parseAdminItemForm(form).request).toBeUndefined();
	}
});

test("accepts dense-liquid macro values on a 100 ml basis", () => {
	expect(parseAdminItemForm(valid({ physicalState: "liquid", density: "1.2", carbohydrates: "110" })).request).toMatchObject({
		physicalState: "liquid",
		densityGramsPerMilliliter: 1.2,
		macrosPer100: { protein: 10, carbohydrates: 110, fat: 5 }
	});
});

test("permits only permanent, unknown, or exhausted transient failed deletion retries", () => {
	const base = { requestId: "00000000-0000-4000-8000-000000000001", requestedAt: "2026-07-21T00:00:00Z" } as const;
	expect(deletionRetryEligible({ ...base, status: "failed", failureCategory: "permanent", retryCount: 0 })).toBeTrue();
	expect(deletionRetryEligible({ ...base, status: "failed", failureCategory: "unknown", retryCount: 0 })).toBeTrue();
	expect(deletionRetryEligible({ ...base, status: "failed", failureCategory: "transient", retryCount: 3 })).toBeTrue();
	expect(deletionRetryEligible({ ...base, status: "failed", failureCategory: "transient", retryCount: 2 })).toBeFalse();
	expect(deletionRetryEligible({ ...base, status: "processing", retryCount: 3 })).toBeFalse();
});

test("creates a memory-only namespaced item idempotency key", () => {
	expect(newAdminItemKey()).toMatch(/^admin-item-[0-9a-f-]{36}$/);
});
