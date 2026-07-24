import type {
	AdminClassificationCollectionEnvelope,
	AdminClassificationEnvelope,
	AdminDeletionRetryEnvelope,
	AdminItemEnvelope,
	AdminUserPageEnvelope,
	CuratedImportEnvelope,
	CustomItemEnvelope,
	ExternalSearchEnvelope,
	FilterOptionsEnvelope
} from "./generated";

type Assert<T extends true> = T;
type AssertFalse<T extends false> = T;
type IsStrictOk<T> = T extends { status: "ok"; requestId: string; data: unknown } ? true : false;
type IsAssignable<TSource, TTarget> = TSource extends TTarget ? true : false;

// Implements DESIGN-009 AdminController strict generated success-envelope verification.
export type Phase08SuccessEnvelopeTypeChecks = [
	Assert<IsStrictOk<CustomItemEnvelope>>,
	Assert<IsStrictOk<FilterOptionsEnvelope>>,
	Assert<IsStrictOk<ExternalSearchEnvelope>>,
	Assert<IsStrictOk<CuratedImportEnvelope>>,
	Assert<IsStrictOk<AdminItemEnvelope>>,
	Assert<IsStrictOk<AdminClassificationEnvelope>>,
	Assert<IsStrictOk<AdminClassificationCollectionEnvelope>>,
	Assert<IsStrictOk<AdminUserPageEnvelope>>,
	Assert<IsStrictOk<AdminDeletionRetryEnvelope>>,
	AssertFalse<IsAssignable<{ status: "ok"; requestId: string }, AdminClassificationEnvelope>>,
	AssertFalse<
		IsAssignable<
			{
				status: "error";
				requestId: string;
				data: { classification: { id: string; name: string; kind: "food_category" } };
			},
			AdminClassificationEnvelope
		>
	>
];
