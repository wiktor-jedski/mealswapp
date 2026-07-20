import { expect, test } from "bun:test";

import { DailyDietClientError } from "./api/daily-diet-client";
import type { DailyDiet, DailyDietCreateRequest, OptimizationAlternative } from "./api/generated";
import { alternativeDietName, saveAlternativeDiet } from "./alternative-diet-save";

// Implements DESIGN-001 SearchView generated-alternative save verification.

const alternative: OptimizationAlternative = {
	meals: [{ mealId: "00000000-0000-0000-0000-000000000001", name: "Banana", quantity: 100, unit: "g", position: 0 }],
	macros: { protein: 1, carbohydrates: 23, fat: 0, calories: 98 },
	similarityScore: 0
};

function diet(name: string): DailyDiet {
	return {
		id: "00000000-0000-0000-0000-000000000010",
		name,
		entries: [{ id: "00000000-0000-0000-0000-000000000011", foodObjectId: alternative.meals[0]!.mealId, foodObjectType: "meal", quantity: 100, unit: "g", position: 0 }],
		aggregateMacros: alternative.macros,
		createdAt: "2026-07-20T00:00:00Z",
		updatedAt: "2026-07-20T00:00:00Z"
	};
}

test("saves with the card number and projects meals into ordered Daily Diet entries", async () => {
	let captured: DailyDietCreateRequest | null = null;
	const saved = await saveAlternativeDiet("Training day", 1, alternative, [], async (request) => {
		captured = request;
		return diet(request.name);
	});
	expect(saved.name).toBe("Training day - Alternative 1");
	expect(captured).toEqual({
		name: "Training day - Alternative 1",
		entries: [{ foodObjectId: alternative.meals[0]!.mealId, foodObjectType: "meal", quantity: 100, unit: "g", position: 0 }]
	});
});

test("skips local names case-insensitively and retries authoritative duplicate conflicts", async () => {
	const attempted: string[] = [];
	const saved = await saveAlternativeDiet("Training day", 1, alternative, [diet("training day - alternative 1")], async (request) => {
		attempted.push(request.name);
		if (request.name.endsWith("2")) {
			throw new DailyDietClientError(
				{ category: "validation", code: "duplicate_daily_diet_name", message: "duplicate", retryable: false },
				409
			);
		}
		return diet(request.name);
	});
	expect(attempted).toEqual(["Training day - Alternative 2", "Training day - Alternative 3"]);
	expect(saved.name).toBe("Training day - Alternative 3");
});

test("does not retry unrelated failures", async () => {
	const failure = new DailyDietClientError(
		{ category: "dependency", code: "daily_diet_unavailable", message: "unavailable", retryable: true },
		503
	);
	await expect(saveAlternativeDiet("Training day", 1, alternative, [], async () => { throw failure; })).rejects.toBe(failure);
});

test("keeps long generated names within the Daily Diet contract limit", () => {
	const name = alternativeDietName("x".repeat(120), 12);
	expect(name.length).toBe(120);
	expect(name.endsWith(" - Alternative 12")).toBe(true);
});
