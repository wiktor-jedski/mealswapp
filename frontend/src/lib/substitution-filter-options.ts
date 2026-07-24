import type { FilterOption, FoodObject, SearchFilter, SearchFilterKind } from "./api/generated";

// Implements DESIGN-001 SearchView dynamic substitution filter projection.

export type SubstitutionFilterOption = SearchFilter & {
	label: string;
	description: string;
	searchText: string;
};

/** Preserves backend ordering and labels, then appends missing selected-item classifications. */
export function substitutionFilterOptions(
	backendOptions: FilterOption[],
	selectedItems: FoodObject[],
	include: boolean
): SubstitutionFilterOption[] {
	const projected = backendOptions
		.filter((option) => include ? option.includeAllowed : option.excludeAllowed)
		.map((option) => projectBackendOption(option, include));
	const seen = new Set(projected.map(optionIdentity));

	for (const item of selectedItems) {
		for (const classification of item.classifications) {
			const option: SubstitutionFilterOption = {
				filterId: classification.id,
				kind: classification.kind,
				include,
				label: classification.name,
				description: kindLabel(classification.kind),
				searchText: `${classification.name} ${kindLabel(classification.kind)}`
			};
			const identity = optionIdentity(option);
			if (!seen.has(identity)) {
				seen.add(identity);
				projected.push(option);
			}
		}
	}
	return projected;
}

function projectBackendOption(option: FilterOption, include: boolean): SubstitutionFilterOption {
	return {
		filterId: option.filterId,
		kind: option.kind,
		include,
		label: option.label,
		description: kindLabel(option.kind),
		searchText: `${option.label} ${kindLabel(option.kind)} ${option.labelKey ?? ""}`.trim()
	};
}

function kindLabel(kind: SearchFilterKind): string {
	return kind.replaceAll("_", " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function optionIdentity(option: Pick<SearchFilter, "filterId" | "kind">): string {
	return `${option.kind}:${option.filterId}`;
}
