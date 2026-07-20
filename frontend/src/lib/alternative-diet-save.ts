import { DailyDietClientError } from "./api/daily-diet-client";
import type { DailyDiet, DailyDietCreateRequest, OptimizationAlternative } from "./api/generated";

// Implements DESIGN-001 SearchView saving generated Daily Diet alternatives.
// Implements DESIGN-008 SavedDataRepository user-scoped unique Daily Diet names.

const MAX_DAILY_DIET_NAME_LENGTH = 120;

/** Creates one saved Daily Diet through the existing mutation controller. */
export type AlternativeDietCreate = (request: DailyDietCreateRequest) => Promise<DailyDiet>;

/** Builds the numbered saved name while preserving the suffix within the API limit. */
export function alternativeDietName(sourceName: string, number: number): string {
	if (!Number.isSafeInteger(number) || number < 1) throw new TypeError("Alternative number must be a positive safe integer");
	const suffix = ` - Alternative ${number}`;
	const available = MAX_DAILY_DIET_NAME_LENGTH - suffix.length;
	if (available < 1) throw new RangeError("Alternative number is too large to name");
	return `${sourceName.trim().slice(0, available).trimEnd()}${suffix}`;
}

/**
 * Saves a generated alternative, advancing its number across local or
 * authoritative server-side duplicate-name collisions.
 */
export async function saveAlternativeDiet(
	sourceName: string,
	preferredNumber: number,
	alternative: OptimizationAlternative,
	existingDiets: readonly DailyDiet[],
	create: AlternativeDietCreate
): Promise<DailyDiet> {
	const occupiedNames = new Set(existingDiets.map((diet) => canonicalName(diet.name)));
	let number = preferredNumber;
	while (Number.isSafeInteger(number)) {
		const name = alternativeDietName(sourceName, number);
		if (occupiedNames.has(canonicalName(name))) {
			number += 1;
			continue;
		}
		const request: DailyDietCreateRequest = {
			name,
			entries: alternative.meals.map((meal, position) => ({
				foodObjectId: meal.mealId,
				foodObjectType: "meal",
				quantity: meal.quantity,
				unit: meal.unit,
				position
			}))
		};
		try {
			return await create(request);
		} catch (error) {
			if (!isDuplicateName(error)) throw error;
			occupiedNames.add(canonicalName(name));
			number += 1;
		}
	}
	throw new RangeError("No safe alternative number remains");
}

function canonicalName(value: string): string {
	return value.trim().toLocaleLowerCase();
}

function isDuplicateName(error: unknown): boolean {
	return error instanceof DailyDietClientError && error.status === 409 && error.appError.code === "duplicate_daily_diet_name";
}
