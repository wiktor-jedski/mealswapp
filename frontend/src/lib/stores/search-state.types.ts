import type {
	CatalogSearchState,
	DailyDietAlternativeSearchState,
	DailyDietSearchState,
	SearchState,
	SubstitutionSearchState
} from "./search";

// Implements DESIGN-001 SearchView mode-safe SearchState compile-time verification.

const commonState = {
	query: "apple",
	submittedQuery: "apple",
	searchSubmitted: true,
	filters: [],
	page: 1,
	loading: false,
	error: null
};

// Positive constructions prove every mode has a complete, assignable state shape.
const validCatalogState = {
	...commonState,
	mode: "catalog"
} satisfies CatalogSearchState;

const validSubstitutionState = {
	...commonState,
	mode: "substitution",
	substitutionInputs: [{ foodObjectId: "food-1", quantity: 100, unit: "g" }],
	substitutionInputLabels: { "food-1": "Apple" },
	substitutionInputItems: {}
} satisfies SubstitutionSearchState;

const validDailyDietState = {
	...commonState,
	mode: "daily_diet",
	dailyDietCollections: [{ id: "diet-1", name: "Weekday diet" }]
} satisfies DailyDietSearchState;

const validDailyDietAlternativeState = {
	...commonState,
	mode: "daily_diet_alternative",
	dailyDietId: "diet-1"
} satisfies DailyDietAlternativeSearchState;

// Negative constructions prove Catalog rejects every mode-specific field.
const catalogWithSubstitutionInputs: CatalogSearchState = {
	...validCatalogState,
	// @ts-expect-error Catalog cannot carry Substitution Inputs.
	substitutionInputs: []
};
const catalogWithSubstitutionInputLabels: CatalogSearchState = {
	...validCatalogState,
	// @ts-expect-error Catalog cannot carry Substitution Input labels.
	substitutionInputLabels: {}
};
const catalogWithSubstitutionInputItems: CatalogSearchState = {
	...validCatalogState,
	// @ts-expect-error Catalog cannot carry Substitution Input display items.
	substitutionInputItems: {}
};
const catalogWithDailyDietId: CatalogSearchState = {
	...validCatalogState,
	// @ts-expect-error Catalog cannot carry a Daily Diet id.
	dailyDietId: "diet-1"
};
const catalogWithDailyDietCollections: CatalogSearchState = {
	...validCatalogState,
	// @ts-expect-error Catalog cannot carry Daily Diet collections.
	dailyDietCollections: []
};

// Negative constructions prove Substitution rejects all Daily Diet-owned fields.
const substitutionWithDailyDietId: SubstitutionSearchState = {
	...validSubstitutionState,
	// @ts-expect-error Substitution cannot carry a Daily Diet id.
	dailyDietId: "diet-1"
};
const substitutionWithDailyDietCollections: SubstitutionSearchState = {
	...validSubstitutionState,
	// @ts-expect-error Substitution cannot carry Daily Diet collections.
	dailyDietCollections: []
};

// Negative constructions prove Daily Diet and Daily Diet Alternative reject other mode fields.
const dailyDietWithSubstitutionInputs: DailyDietSearchState = {
	...validDailyDietState,
	// @ts-expect-error Daily Diet cannot carry Substitution Inputs.
	substitutionInputs: []
};
const dailyDietWithSubstitutionInputLabels: DailyDietSearchState = {
	...validDailyDietState,
	// @ts-expect-error Daily Diet cannot carry Substitution Input labels.
	substitutionInputLabels: {}
};
const dailyDietWithSubstitutionInputItems: DailyDietSearchState = {
	...validDailyDietState,
	// @ts-expect-error Daily Diet cannot carry Substitution Input display items.
	substitutionInputItems: {}
};
const dailyDietWithDailyDietId: DailyDietSearchState = {
	...validDailyDietState,
	// @ts-expect-error Daily Diet owns collections, not a single alternative id.
	dailyDietId: "diet-1"
};
const dailyDietAlternativeWithSubstitutionInputs: DailyDietAlternativeSearchState = {
	...validDailyDietAlternativeState,
	// @ts-expect-error Daily Diet Alternative cannot carry Substitution Inputs.
	substitutionInputs: []
};
const dailyDietAlternativeWithSubstitutionInputLabels: DailyDietAlternativeSearchState = {
	...validDailyDietAlternativeState,
	// @ts-expect-error Daily Diet Alternative cannot carry Substitution Input labels.
	substitutionInputLabels: {}
};
const dailyDietAlternativeWithSubstitutionInputItems: DailyDietAlternativeSearchState = {
	...validDailyDietAlternativeState,
	// @ts-expect-error Daily Diet Alternative cannot carry Substitution Input display items.
	substitutionInputItems: {}
};
const dailyDietAlternativeWithDailyDietCollections: DailyDietAlternativeSearchState = {
	...validDailyDietAlternativeState,
	// @ts-expect-error Daily Diet Alternative owns one id, not Daily Diet collections.
	dailyDietCollections: []
};

// Negative constructions prove every required mode-owned field is required.
// @ts-expect-error Substitution requires substitutionInputs.
const substitutionWithoutInputs: SubstitutionSearchState = {
	...commonState,
	mode: "substitution",
	substitutionInputLabels: {},
	substitutionInputItems: {}
};
// @ts-expect-error Substitution requires substitutionInputLabels.
const substitutionWithoutLabels: SubstitutionSearchState = {
	...commonState,
	mode: "substitution",
	substitutionInputs: [],
	substitutionInputItems: {}
};
// @ts-expect-error Substitution requires substitutionInputItems.
const substitutionWithoutItems: SubstitutionSearchState = {
	...commonState,
	mode: "substitution",
	substitutionInputs: [],
	substitutionInputLabels: {}
};
// @ts-expect-error Daily Diet requires dailyDietCollections.
const dailyDietWithoutCollections: DailyDietSearchState = {
	...commonState,
	mode: "daily_diet"
};
// @ts-expect-error Daily Diet Alternative requires dailyDietId, even when empty.
const dailyDietAlternativeWithoutId: DailyDietAlternativeSearchState = {
	...commonState,
	mode: "daily_diet_alternative"
};

type Assert<Condition extends true> = Condition;
type HasKey<Type, Key extends PropertyKey> = Key extends keyof Type ? true : false;

type CatalogCannotCarrySubstitutionInputs = Assert<
	HasKey<CatalogSearchState, "substitutionInputs"> extends false ? true : false
>;
type CatalogCannotCarryDailyDietId = Assert<
	HasKey<CatalogSearchState, "dailyDietId"> extends false ? true : false
>;
type CatalogCannotCarryDailyDietCollections = Assert<
	HasKey<CatalogSearchState, "dailyDietCollections"> extends false ? true : false
>;
type SubstitutionCannotCarryDailyDietId = Assert<
	HasKey<SubstitutionSearchState, "dailyDietId"> extends false ? true : false
>;
type SubstitutionCannotCarryDailyDietCollections = Assert<
	HasKey<SubstitutionSearchState, "dailyDietCollections"> extends false ? true : false
>;
type DailyDietCannotCarrySubstitutionInputs = Assert<
	HasKey<DailyDietSearchState, "substitutionInputs"> extends false ? true : false
>;
type DailyDietCannotCarryDailyDietId = Assert<
	HasKey<DailyDietSearchState, "dailyDietId"> extends false ? true : false
>;
type DailyDietOwnsCollections = Assert<
	HasKey<DailyDietSearchState, "dailyDietCollections">
>;
type DailyDietAlternativeOwnsId = Assert<
	HasKey<DailyDietAlternativeSearchState, "dailyDietId">
>;
type DailyDietAlternativeCannotCarrySubstitutionInputs = Assert<
	HasKey<DailyDietAlternativeSearchState, "substitutionInputs"> extends false ? true : false
>;
type DailyDietAlternativeCannotCarryDailyDietCollections = Assert<
	HasKey<DailyDietAlternativeSearchState, "dailyDietCollections"> extends false ? true : false
>;

// Keep the assertions in the module so TypeScript checks them without runtime casts or values.
export type SearchStateModeShapeChecks =
	| CatalogCannotCarrySubstitutionInputs
	| CatalogCannotCarryDailyDietId
	| CatalogCannotCarryDailyDietCollections
	| SubstitutionCannotCarryDailyDietId
	| SubstitutionCannotCarryDailyDietCollections
	| DailyDietCannotCarrySubstitutionInputs
	| DailyDietCannotCarryDailyDietId
	| DailyDietOwnsCollections
	| DailyDietAlternativeOwnsId
	| DailyDietAlternativeCannotCarrySubstitutionInputs
	| DailyDietAlternativeCannotCarryDailyDietCollections
	| (SearchState extends CatalogSearchState | SubstitutionSearchState | DailyDietSearchState | DailyDietAlternativeSearchState ? true : false);
