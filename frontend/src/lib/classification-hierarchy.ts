import type { AdminClassification } from "./api/generated";

// Implements DESIGN-009 TagManager deterministic classification hierarchy presentation.

/** One classification with its stable position in an accessible hierarchy. */
export interface ClassificationHierarchyRow {
	value: AdminClassification;
	depth: number;
	parentName?: string;
}

/** Orders a classification hierarchy by normalized sibling name and stable ID. */
export function classificationHierarchy(values: AdminClassification[]): ClassificationHierarchyRow[] {
	const byId = new Map(values.map((value) => [value.id, value]));
	const children = new Map<string | undefined, AdminClassification[]>();
	for (const value of values) {
		const parentId = value.parentId && byId.has(value.parentId) ? value.parentId : undefined;
		children.set(parentId, [...(children.get(parentId) ?? []), value]);
	}
	for (const siblings of children.values()) siblings.sort(compareClassifications);

	const rows: ClassificationHierarchyRow[] = [];
	const visited = new Set<string>();
	const append = (value: AdminClassification, depth: number): void => {
		if (visited.has(value.id)) return;
		visited.add(value.id);
		rows.push({ value, depth, ...(value.parentId && byId.has(value.parentId) ? { parentName: byId.get(value.parentId)!.name } : {}) });
		for (const child of children.get(value.id) ?? []) append(child, depth + 1);
	};
	for (const root of children.get(undefined) ?? []) append(root, 0);
	for (const value of [...values].sort(compareClassifications)) append(value, 0);
	return rows;
}

function compareClassifications(left: AdminClassification, right: AdminClassification): number {
	const leftName = left.name.normalize("NFKC").toLowerCase();
	const rightName = right.name.normalize("NFKC").toLowerCase();
	if (leftName !== rightName) return leftName < rightName ? -1 : 1;
	if (left.name !== right.name) return left.name < right.name ? -1 : 1;
	return left.id < right.id ? -1 : left.id === right.id ? 0 : 1;
}
