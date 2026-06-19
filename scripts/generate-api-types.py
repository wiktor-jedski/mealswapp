#!/usr/bin/env python3

# Implements DESIGN-017 ErrorMessageMapper frontend contract generation.

import argparse
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
OPENAPI = ROOT / "api" / "openapi.yaml"
OUTPUT = ROOT / "frontend" / "src" / "lib" / "api" / "generated.ts"
REQUIRED_MARKERS = (
	"AppError:",
	"Envelope:",
	"CSRFTokenEnvelope:",
	"AuthSessionEnvelope:",
	"VerifyEmailEnvelope:",
	"PasswordResetConsumeEnvelope:",
	"PasswordResetRequestEnvelope:",
	"ProfileEnvelope:",
	"SavedItemsEnvelope:",
	"SearchHistoryEnvelope:",
	"ExportBundle:",
	"DeletionRequestEnvelope:",
	"DisclaimerEnvelope:",
	"SearchMode:",
	"SearchFilterKind:",
	"SearchRequest:",
	"CacheMetadata:",
	"SearchResponse:",
	"SearchResponseEnvelope:",
	"SearchRejectionEnvelope:",
	"AutocompleteResponse:",
	"AutocompleteEnvelope:",
	"/api/v1/search:",
	"/api/v1/search/autocomplete:",
	"/api/v1/auth/register:",
	"/api/v1/auth/login:",
	"/api/v1/auth/logout:",
	"/api/v1/auth/refresh:",
	"/api/v1/auth/verify-email:",
	"/api/v1/auth/password-reset/request:",
	"/api/v1/auth/password-reset/consume:",
	"/api/v1/auth/oauth/{provider}/start:",
	"/api/v1/auth/oauth/{provider}/callback:",
	"/api/v1/profile:",
	"/api/v1/saved-items:",
	"/api/v1/search-history:",
	"/api/v1/account/export:",
	"/api/v1/account:",
	"/api/v1/disclaimers:",
)
GENERATED = """// Generated from api/openapi.yaml by scripts/generate-api-types.py.
// Implements DESIGN-017 ErrorMessageMapper shared frontend contracts.

export type ErrorCategory =
\t| "validation"
\t| "auth"
\t| "entitlement"
\t| "network"
\t| "timeout"
\t| "server"
\t| "dependency"
\t| "unknown";

// Implements DESIGN-017 ErrorMessageMapper AppError contract.
/** User-safe classified server error returned by the API gateway. */
export interface AppError {
\tcategory: ErrorCategory;
\tcode: string;
\tmessage: string;
\tretryable: boolean;
\trequestId?: string;
}

// Implements DESIGN-017 GlobalExceptionHandler response envelope.
/** Shared API response wrapper with request correlation metadata. */
export interface Envelope<TData extends Record<string, unknown> = Record<string, unknown>> {
\tstatus: string;
\trequestId: string;
\tdata?: TData;
\terror?: AppError | null;
}

// Implements DESIGN-014 UptimeMonitor liveness contract.
/** Process liveness payload. */
export interface HealthData extends Record<string, unknown> {
\tservice: string;
}

// Implements DESIGN-014 UptimeMonitor readiness contract.
/** Dependency-readiness payload. */
export interface ReadinessData extends Record<string, unknown> {
\tchecks: Record<string, string>;
}

// Implements DESIGN-006 AuthController CSRF token-delivery contract.
/** Session-bound synchronizer token delivered to SPA clients. */
export interface CSRFTokenData extends Record<string, unknown> {
\tcsrfToken: string;
}

// Implements DESIGN-006 AuthController CSRF token-delivery contract.
/** CSRF token response envelope. */
export type CSRFTokenEnvelope = Envelope<CSRFTokenData>;

// Implements DESIGN-006 AuthController frontend auth contract.
/** Authenticated session metadata; token values are carried only by HttpOnly cookies. */
export interface AuthSessionData extends Record<string, unknown> {
\tuserId: string;
\trole: "user" | "admin";
\thasVerifiedLoginMethod: boolean;
\taccessExpiresAt: string;
\trefreshExpiresAt: string;
}

// Implements DESIGN-006 AuthController frontend auth contract.
/** Authenticated session response envelope. */
export type AuthSessionEnvelope = Envelope<AuthSessionData>;

// Implements DESIGN-006 AuthController frontend registration contract.
/** Registration request accepted by the account API. */
export interface RegisterRequest {
\temail: string;
\tpassword: string;
\tprivacyPolicyVersion: string;
\ttermsVersion: string;
}

// Implements DESIGN-006 AuthController frontend login contract.
/** Email/password login request. */
export interface LoginRequest {
\temail: string;
\tpassword: string;
}

// Implements DESIGN-006 AuthController frontend verification contract.
/** Verification completion payload. */
export interface VerifyEmailData extends Record<string, unknown> {
\thasVerifiedLoginMethod: true;
}

// Implements DESIGN-006 AuthController frontend verification contract.
/** Verification completion response envelope. */
export type VerifyEmailEnvelope = Envelope<VerifyEmailData>;

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password reset request that never reveals account existence. */
export interface PasswordResetRequest {
\temail: string;
}

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Single-use password reset token consumption request. */
export interface PasswordResetConsumeRequest {
\ttoken: string;
\tnewPassword: string;
}

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password-reset request acceptance payload. */
export interface PasswordResetAcceptedData extends Record<string, unknown> {
\taccepted: true;
}

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password-reset request acceptance envelope. */
export type PasswordResetRequestEnvelope = Envelope<PasswordResetAcceptedData>;

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password-reset completion payload. */
export interface PasswordResetConsumeData extends Record<string, unknown> {
\treset: true;
}

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password-reset completion envelope. */
export type PasswordResetConsumeEnvelope = Envelope<PasswordResetConsumeData>;

// Implements DESIGN-006 OAuthHandler frontend provider contract.
/** Supported OAuth identity providers. */
export type OAuthProvider = "google" | "apple";

// Implements DESIGN-008 PreferenceManager frontend profile contract.
/** User profile and preference response data. */
export interface ProfileData extends Record<string, unknown> {
\tuserId: string;
\tdisplayName: string;
\tunitSystem: "metric" | "imperial";
\tthemePreference: "system" | "light" | "dark";
\trequiresUnitRecalculation: boolean;
}

// Implements DESIGN-008 PreferenceManager frontend profile contract.
/** User profile response envelope. */
export type ProfileEnvelope = Envelope<ProfileData>;

// Implements DESIGN-008 PreferenceManager frontend profile contract.
/** Mutable profile preference request. */
export interface ProfileUpdateRequest {
\tdisplayName?: string;
\tunitSystem: "metric" | "imperial";
\tthemePreference: "system" | "light" | "dark";
}

// Implements DESIGN-008 SavedDataRepository frontend saved-data contract.
/** One saved favorite, meal, or reserved diet reference. */
export interface SavedItem {
\tid: string;
\titemId: string;
\tkind: "favorite" | "saved_meal" | "saved_diet";
}

// Implements DESIGN-008 SavedDataRepository frontend saved-data contract.
/** Saved item query filter. */
export type SavedItemKind = SavedItem["kind"];

// Implements DESIGN-008 SavedDataRepository frontend saved-data contract.
/** Saved item collection payload. */
export interface SavedItemsData extends Record<string, unknown> {
\titems: SavedItem[];
}

// Implements DESIGN-008 SavedDataRepository frontend saved-data contract.
/** Saved item collection response envelope. */
export type SavedItemsEnvelope = Envelope<SavedItemsData>;

// Implements DESIGN-008 SearchHistoryRepository frontend history contract.
/** One decrypted search-history entry at the API boundary. */
export interface SearchHistoryEntry {
\tid: string;
\tquery: string;
\tmode: string;
\tfiltersHash: string;
}

// Implements DESIGN-008 SearchHistoryRepository frontend history contract.
/** Search-history collection payload. */
export interface SearchHistoryData extends Record<string, unknown> {
\thistory: SearchHistoryEntry[];
}

// Implements DESIGN-008 SearchHistoryRepository frontend history contract.
/** Search-history collection response envelope. */
export type SearchHistoryEnvelope = Envelope<SearchHistoryData>;

// Implements DESIGN-008 DataExporter frontend export contract.
/** JSON account export bundle. */
export interface ExportBundle {
\tuser: Record<string, unknown>;
\tconsent: Array<Record<string, unknown>>;
\tsavedItems: SavedItem[];
\thistory: SearchHistoryEntry[];
\tcustomItems: Array<Record<string, unknown>>;
}

// Implements DESIGN-008 DataExporter frontend export contract.
/** Supported account export formats. */
export type ExportFormat = "json" | "csv";

// Implements DESIGN-008 AccountDeleter frontend deletion contract.
/** Account deletion request response data. */
export interface DeletionRequestData extends Record<string, unknown> {
\trequestId: string;
\tstatus: "pending" | "processing" | "completed" | "failed";
}

// Implements DESIGN-008 AccountDeleter frontend deletion contract.
/** Account deletion response envelope. */
export type DeletionRequestEnvelope = Envelope<DeletionRequestData>;

// Implements DESIGN-015 DisclaimerRenderer frontend disclaimer contract.
/** Stable Markdown disclaimer content for login and account surfaces. */
export interface DisclaimerData extends Record<string, unknown> {
\tlocation: "login" | "account";
\tversion: string;
\tmarkdown: string;
\tfallback: boolean;
\talert?: string;
}

// Implements DESIGN-015 DisclaimerRenderer frontend disclaimer contract.
/** Disclaimer response envelope. */
export type DisclaimerEnvelope = Envelope<DisclaimerData>;

// Implements DESIGN-002 SearchController frontend search-mode contract.
/** Supported search workflows exposed by the search API. */
export type SearchMode = "catalog" | "substitution" | "daily_diet_alternative";

// Implements DESIGN-002 SearchController frontend search-filter contract.
/** Supported filter classes accepted by the search API. */
export type SearchFilterKind =
\t| "food_category"
\t| "culinary_role"
\t| "physical_state"
\t| "allergen"
\t| "dietary_preset";

// Implements DESIGN-002 SearchController frontend search-filter contract.
/** One include or exclude filter applied to a search request. */
export interface SearchFilter {
\tfilterId: string;
\tkind: SearchFilterKind;
\tinclude: boolean;
}

// Implements DESIGN-002 SearchController frontend substitution contract.
/** Canonical units accepted by substitution search inputs. */
export type SubstitutionUnit = "g" | "ml" | "oz" | "fl_oz";

// Implements DESIGN-002 SearchController frontend substitution contract.
/** Quantity-bearing food input for substitution searches. */
export interface SubstitutionInput {
\tfoodObjectId: string;
\tquantity: number;
\tunit: SubstitutionUnit;
}

// Implements DESIGN-002 SearchController frontend search request contract.
/** Request payload for catalog, substitution, and daily-diet alternative search. */
export interface SearchRequest {
\tquery: string;
\tmode: SearchMode;
\tfilters?: SearchFilter[];
\tpage: number;
\tsubstitutionInputs?: SubstitutionInput[];
\tdailyDietId?: string;
}

// Implements DESIGN-002 SearchController frontend classification result contract.
/** Classification identity returned with each search result. */
export interface ClassificationSummary {
\tid: string;
\tname: string;
\tkind: "food_category" | "culinary_role";
}

// Implements DESIGN-001 MacroSummary frontend result contract.
/** Normalized macronutrients and their physical-state display basis. */
export interface MacroSummary {
\tprotein: number;
\tcarbohydrate: number;
\tfat: number;
\tbasis: "100g" | "100ml";
}

// Implements DESIGN-002 SearchController frontend food-object result contract.
/** Food object returned by search and autocomplete-related result flows. */
export interface FoodObject {
\tid: string;
\tname: string;
\tphysicalState: "solid" | "liquid";
\timageUrl?: string | null;
\tclassifications: ClassificationSummary[];
\tprimaryFoodCategory: ClassificationSummary | null;
\tmacros: MacroSummary;
\tcalories: number;
}

// Implements DESIGN-002 SearchController frontend similarity metadata contract.
/** User-facing nutritional similarity tier. */
export type SimilarityTier = "excellent" | "good" | "fair" | "poor";

// Implements DESIGN-002 SearchController frontend similarity metadata contract.
/** Similarity display metadata for a ranked search result. */
export interface SimilarityMetadata {
\titemId: string;
\tscore: number;
\ttier: SimilarityTier;
\timageUrl: string;
\tmatchingQuantity: number;
}

// Implements DESIGN-011 SearchCache frontend cache metadata contract.
/** Cache status metadata returned with search-domain responses. */
export interface CacheMetadata {
\tstatus: "hit" | "miss";
\tnamespace: string;
\tschemaVersion: string;
\tttlSeconds: number;
}

// Implements DESIGN-002 SearchController frontend search rejection contract.
/** Structured, user-facing search rejection detail. */
export interface SearchRejection {
\tcode: string;
\tmessage: string;
\tfield?: string;
}

// Implements DESIGN-002 SearchController frontend search response contract.
/** Search result payload with ranking, warnings, and optional cache metadata. */
export interface SearchResponse extends Record<string, unknown> {
\titems: FoodObject[];
\ttotalCount: number;
\tpage: number;
\tsimilarityScores: number[];
\tsimilarityMetadata: SimilarityMetadata[];
\twarnings: string[];
\tcache?: CacheMetadata;
}

// Implements DESIGN-002 SearchController frontend search response contract.
/** Successful search response envelope. */
export type SearchResponseEnvelope = Envelope<SearchResponse>;

// Implements DESIGN-017 ErrorMessageMapper frontend search error contract.
/** Search rejection response envelope with safe error text. */
export interface SearchRejectionEnvelope extends Envelope<{ rejection: SearchRejection }> {
\tstatus: "error";
\tdata: { rejection: SearchRejection };
\terror: AppError;
}

// Implements DESIGN-002 SearchController frontend autocomplete contract.
/** Ranked autocomplete suggestion. */
export interface RankedAutocomplete {
\titemId: string;
\tlabel: string;
\texactMatch: boolean;
\tlevenshteinDistance: number;
\tlength: number;
\trank: number;
}

// Implements DESIGN-002 SearchController frontend autocomplete contract.
/** Autocomplete payload with ranked suggestions and optional cache metadata. */
export interface AutocompleteResponse extends Record<string, unknown> {
\titems: RankedAutocomplete[];
\tcache?: CacheMetadata;
}

// Implements DESIGN-002 SearchController frontend autocomplete contract.
/** Successful autocomplete response envelope. */
export type AutocompleteEnvelope = Envelope<AutocompleteResponse>;
"""


def main() -> int:
	parser = argparse.ArgumentParser(description="Generate shared frontend API types from the OpenAPI contract.")
	parser.add_argument("--check", action="store_true", help="Fail if generated frontend types have drifted.")
	args = parser.parse_args()
	source = OPENAPI.read_text(encoding="utf-8")
	missing = [marker for marker in REQUIRED_MARKERS if marker not in source]
	if missing:
		print(f"OpenAPI contract missing required markers: {missing}")
		return 1
	if args.check:
		if not OUTPUT.exists() or OUTPUT.read_text(encoding="utf-8") != GENERATED:
			print(f"Generated API types are stale: run `python3 {Path(__file__).name}`")
			return 1
		print("Generated API types are current.")
		return 0
	OUTPUT.parent.mkdir(parents=True, exist_ok=True)
	OUTPUT.write_text(GENERATED, encoding="utf-8")
	print(f"Generated {OUTPUT.relative_to(ROOT)}")
	return 0


if __name__ == "__main__":
	sys.exit(main())
