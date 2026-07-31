import { expect, test } from "bun:test";
import type { AdminClassification } from "./api/generated";
import { classificationHierarchy } from "./classification-hierarchy";

// Implements DESIGN-009 TagManager deterministic hierarchy verification.

const category = (id: string, name: string, parentId?: string): AdminClassification => ({ id, name, kind: "food_category", ...(parentId ? { parentId } : {}) });

test("renders parents before deterministically ordered descendants", () => {
	const values = [
		category("00000000-0000-4000-8000-000000000004", "Berry", "00000000-0000-4000-8000-000000000002"),
		category("00000000-0000-4000-8000-000000000003", "vegetable", "00000000-0000-4000-8000-000000000001"),
		category("00000000-0000-4000-8000-000000000002", "Fruit", "00000000-0000-4000-8000-000000000001"),
		category("00000000-0000-4000-8000-000000000001", "Food")
	];

	expect(classificationHierarchy(values).map(({ value, depth, parentName }) => [value.name, depth, parentName])).toEqual([
		["Food", 0, undefined],
		["Fruit", 1, "Food"],
		["Berry", 2, "Fruit"],
		["vegetable", 1, "Food"]
	]);
	expect(classificationHierarchy(values)).toEqual(classificationHierarchy([...values].reverse()));
});

test("keeps malformed orphan and cycle input finite and stable", () => {
	const values = [
		category("00000000-0000-4000-8000-000000000001", "A", "00000000-0000-4000-8000-000000000002"),
		category("00000000-0000-4000-8000-000000000002", "B", "00000000-0000-4000-8000-000000000001"),
		category("00000000-0000-4000-8000-000000000003", "Orphan", "00000000-0000-4000-8000-000000000099")
	];

	expect(classificationHierarchy(values).map(({ value }) => value.name)).toEqual(["Orphan", "A", "B"]);
});

test("breaks normalized-name ties by exact name and stable ID", () => {
	const first = category("00000000-0000-4000-8000-000000000001", "Fruit");
	const values = [
		category("00000000-0000-4000-8000-000000000003", "fruit"),
		category("00000000-0000-4000-8000-000000000002", "Fruit"),
		first,
		first
	];

	expect(classificationHierarchy(values).map(({ value }) => value.id)).toEqual([
		"00000000-0000-4000-8000-000000000001",
		"00000000-0000-4000-8000-000000000002",
		"00000000-0000-4000-8000-000000000003"
	]);
});
