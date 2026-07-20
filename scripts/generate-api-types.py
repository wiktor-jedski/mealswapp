#!/usr/bin/env python3

# Implements DESIGN-017 ErrorMessageMapper frontend contract generation.

import argparse
import json
import re
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
	"DailyDiet:",
	"DailyDietCreateRequest:",
	"DailyDietCollectionEnvelope:",
	"MacroProjection:",
	"DietOptimizationRequest:",
	"OptimizationJobAcknowledgementEnvelope:",
	"OptimizationJobStatusEnvelope:",
	"OptimizationJobQueued:",
	"OptimizationJobProcessing:",
	"OptimizationJobCompleted:",
	"OptimizationJobFailed:",
	"OptimizationJobCancelled:",
	"OptimizationFailureCode:",
	"/api/v1/daily-diets:",
	"/api/v1/daily-diets/{dietId}:",
	"/api/v1/optimization/jobs:",
	"/api/v1/optimization/jobs/{jobId}:",
	"name: Idempotency-Key",
	"SearchHistoryEnvelope:",
	"ExportBundle:",
	"DeletionRequestEnvelope:",
	"DisclaimerEnvelope:",
	"CheckoutCreateRequest:",
	"CheckoutSessionEnvelope:",
	"StripeWebhookEnvelope:",
	"EntitlementStatusEnvelope:",
	"IdempotencyKey:",
	"name: Idempotency-Key",
	"billingRecoveryState:",
	"SearchMode:",
	"SearchFilterKind:",
	"SearchRequest:",
	"SourceSummary:",
	"CacheMetadata:",
	"SearchResponse:",
	"FoodObjectEnvelope:",
	"SearchResponseEnvelope:",
	"SearchRejectionEnvelope:",
	"AutocompleteResponse:",
	"AutocompleteEnvelope:",
	"/api/v1/search:",
	"/api/v1/search/autocomplete:",
	"/api/v1/food-objects/{id}:",
	"/api/v1/auth/csrf-token:",
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
	"/api/v1/billing/entitlement:",
	"/api/v1/billing/checkout:",
	"/api/v1/billing/stripe/webhook:",
	"/api/v1/disclaimers:",
)

# Implements DESIGN-004 JobStatusTracker and DESIGN-008 SavedDataRepository
# operation-response drift enforcement.
REQUIRED_OPERATION_RESPONSES = {
	("/api/v1/daily-diets", "get"): {"200", "401", "403", "404", "500", "503", "504"},
	("/api/v1/daily-diets", "post"): {"201", "400", "401", "403", "404", "409", "500", "503", "504"},
	("/api/v1/daily-diets/{dietId}", "get"): {"200", "400", "401", "403", "404", "500", "503", "504"},
	("/api/v1/daily-diets/{dietId}", "put"): {"200", "400", "401", "403", "404", "409", "500", "503", "504"},
	("/api/v1/daily-diets/{dietId}", "delete"): {"204", "400", "401", "403", "404", "500", "503", "504"},
	("/api/v1/optimization/jobs", "post"): {"202", "400", "401", "403", "404", "409", "429", "500", "503", "504"},
	("/api/v1/optimization/jobs/{jobId}", "get"): {"200", "400", "401", "403", "404", "410", "500", "503", "504"},
}

AUDITED_OPERATION_PREFIXES = ("/api/v1/daily-diets", "/api/v1/optimization/jobs")
HTTP_METHODS = {"get", "put", "post", "delete", "options", "head", "patch", "trace"}
APP_ERROR_CATEGORIES = (
	"validation", "auth", "entitlement", "security", "network", "timeout",
	"server", "dependency", "rate_limit", "unknown",
)

DAILY_DIET_SCHEMA_RULES = {
	"IdempotencyKey": (
		"      name: Idempotency-Key\n",
		"      in: header\n",
		"      required: true\n",
		"      schema:\n        type: string\n        minLength: 8\n        maxLength: 255\n",
	),
	"CanonicalQuantityUnit": ("      type: string\n      enum: [g, ml, oz, fl_oz]\n",),
	"FoodObjectType": ("      type: string\n      enum: [food_item, meal]\n",),
	"FoodObjectQuantity": (
		"      type: object\n",
		"      additionalProperties: false\n",
		"      required: [foodObjectId, foodObjectType, quantity, unit, position]\n",
		"        foodObjectId:\n          type: string\n          format: uuid\n",
		"        foodObjectType:\n          $ref: \"#/components/schemas/FoodObjectType\"\n",
	),
	"DailyDietFoodObjectEntry": (
		"      type: object\n",
		"      additionalProperties: false\n",
		"      required: [id, foodObjectId, foodObjectType, quantity, unit, position]\n",
		"        id:\n          type: string\n          format: uuid\n",
		"        foodObjectId:\n          type: string\n          format: uuid\n",
		"        foodObjectType:\n          $ref: \"#/components/schemas/FoodObjectType\"\n",
		"        quantity:\n          type: number\n          exclusiveMinimum: 0\n          maximum: 1000000\n          multipleOf: 0.001\n",
		"        unit:\n          $ref: \"#/components/schemas/CanonicalQuantityUnit\"\n",
		"        position:\n          type: integer\n          minimum: 0\n          maximum: 99\n",
	),
	"MacroProjection": (
		"      type: object\n",
		"      additionalProperties: false\n",
		"      required: [protein, carbohydrates, fat, calories]\n",
		"        protein:\n          type: number\n          minimum: 0\n          maximum: 1000000000\n",
		"        carbohydrates:\n          type: number\n          minimum: 0\n          maximum: 1000000000\n",
		"        fat:\n          type: number\n          minimum: 0\n          maximum: 1000000000\n",
		"        calories:\n          type: number\n          minimum: 0\n          maximum: 1000000000\n",
	),
	"DailyDiet": (
		"      type: object\n",
		"      additionalProperties: false\n",
		"      required: [id, name, entries, aggregateMacros, createdAt, updatedAt]\n",
		"        id:\n          type: string\n          format: uuid\n",
		"        name:\n          type: string\n          minLength: 1\n          maxLength: 120\n",
		"        entries:\n          type: array\n          minItems: 1\n          maxItems: 100\n          items:\n            $ref: \"#/components/schemas/DailyDietFoodObjectEntry\"\n",
		"        aggregateMacros:\n          $ref: \"#/components/schemas/MacroProjection\"\n",
		"        createdAt:\n          type: string\n          format: date-time\n",
		"        updatedAt:\n          type: string\n          format: date-time\n",
	),
	"DailyDietEnvelope": (
		"      type: object\n",
		"      required: [status, requestId, data]\n",
		"        status:\n          type: string\n          enum: [ok]\n",
		"        requestId:\n          type: string\n",
		"        data:\n          $ref: \"#/components/schemas/DailyDiet\"\n",
	),
	"DailyDietCollectionEnvelope": (
		"      type: object\n",
		"      required: [status, requestId, data]\n",
		"        status:\n          type: string\n          enum: [ok]\n",
		"        requestId:\n          type: string\n",
		"        data:\n          type: object\n          additionalProperties: false\n          required: [diets]\n",
		"            diets:\n              type: array\n              items:\n                $ref: \"#/components/schemas/DailyDiet\"\n",
	),
}

DAILY_DIET_SCHEMA_RULE_COUNTS = {
	"DailyDietFoodObjectEntry": {
		"          format: uuid\n": 2,
	},
	"MacroProjection": {
		"          minimum: 0\n": 4,
		"          maximum: 1000000000\n": 4,
	},
	"DailyDiet": {
		"          format: uuid\n": 1,
		"          format: date-time\n": 2,
	},
}

DAILY_DIET_PROPERTY_NAMES = {
	"FoodObjectQuantity": {"foodObjectId", "foodObjectType", "quantity", "unit", "position"},
	"DailyDietFoodObjectEntry": {"id", "foodObjectId", "foodObjectType", "quantity", "unit", "position"},
	"MacroProjection": {"protein", "carbohydrates", "fat", "calories"},
	"DailyDiet": {"id", "name", "entries", "aggregateMacros", "createdAt", "updatedAt"},
	"DailyDietEnvelope": {"status", "requestId", "data"},
	"DailyDietCollectionEnvelope": {"status", "requestId", "data"},
}

# Implements DESIGN-001 SearchView strict optimization response decoding.
OPTIMIZATION_REQUEST_ID_RULE = (
	"          type: string\n"
	"          minLength: 1\n"
	"          maxLength: 120\n"
	"          pattern: '^[A-Za-z0-9._:-]+$'\n"
)

OPTIMIZATION_SCHEMA_RULES = {
	"MealQuantity": (
		"      type: object\n",
		"      additionalProperties: false\n",
		"      required: [mealId, name, quantity, unit, position]\n",
		"        mealId:\n          type: string\n          format: uuid\n",
		"        name:\n          type: string\n          minLength: 1\n          maxLength: 200\n",
		"        quantity:\n          type: number\n          exclusiveMinimum: 0\n          maximum: 1000000\n          multipleOf: 0.001\n",
		"        unit:\n          $ref: \"#/components/schemas/CanonicalQuantityUnit\"\n",
		"        position:\n          type: integer\n          minimum: 0\n          maximum: 99\n",
	),
	"MacroProjection": DAILY_DIET_SCHEMA_RULES["MacroProjection"],
	"OptimizationStatus": ("      type: string\n      enum: [queued, processing, completed, failed, cancelled]\n",),
	"OptimizationFailureCode": ("      type: string\n      enum: [failed_validation, solver_timeout, solver_infeasible, worker_crash]\n",),
	"OptimizationAlternative": (
		"      type: object\n",
		"      additionalProperties: false\n",
		"      required: [meals, macros, similarityScore]\n",
		"        meals:\n          type: array\n          minItems: 1\n          maxItems: 100\n          items:\n            $ref: \"#/components/schemas/MealQuantity\"\n",
		"        macros:\n          $ref: \"#/components/schemas/MacroProjection\"\n",
		"        similarityScore:\n          type: number\n          minimum: 0\n          maximum: 1\n          multipleOf: 0.0001\n",
	),
	"OptimizationFailure": (
		"      type: object\n",
		"      additionalProperties: false\n",
		"      required: [code, message]\n",
		"        code:\n          $ref: \"#/components/schemas/OptimizationFailureCode\"\n",
		"        message:\n          type: string\n          minLength: 1\n          maxLength: 240\n",
	),
	"OptimizationJobAcknowledgementData": (
		"      type: object\n",
		"      additionalProperties: false\n",
		"      required: [jobId, status, pollUrl]\n",
		"        jobId:\n          type: string\n          format: uuid\n",
		"        status:\n          type: string\n          enum: [queued]\n",
		"        pollUrl:\n          type: string\n          format: uri-reference\n          minLength: 1\n",
	),
	"OptimizationJobAcknowledgementEnvelope": (
		"      type: object\n",
		"      required: [status, requestId, data]\n",
		"        status:\n          type: string\n          enum: [accepted]\n",
		"        data:\n          $ref: \"#/components/schemas/OptimizationJobAcknowledgementData\"\n",
	),
	"OptimizationJobData": (
		"      oneOf:\n        - $ref: \"#/components/schemas/OptimizationJobQueued\"\n        - $ref: \"#/components/schemas/OptimizationJobProcessing\"\n        - $ref: \"#/components/schemas/OptimizationJobCompleted\"\n        - $ref: \"#/components/schemas/OptimizationJobFailed\"\n        - $ref: \"#/components/schemas/OptimizationJobCancelled\"\n",
		"      discriminator:\n        propertyName: status\n",
	),
	"OptimizationJobQueued": (
		"      type: object\n",
		"      additionalProperties: false\n",
		"      required: [jobId, dailyDietId, status, pollUrl, createdAt]\n",
		"        status:\n          allOf:\n            - $ref: \"#/components/schemas/OptimizationStatus\"\n            - const: queued\n",
	),
	"OptimizationJobProcessing": (
		"      type: object\n",
		"      additionalProperties: false\n",
		"      required: [jobId, dailyDietId, status, pollUrl, createdAt, startedAt]\n",
		"        status:\n          allOf:\n            - $ref: \"#/components/schemas/OptimizationStatus\"\n            - const: processing\n",
		"        startedAt:\n          type: string\n          format: date-time\n",
	),
	"OptimizationJobCompleted": (
		"      type: object\n",
		"      additionalProperties: false\n",
		"      required: [jobId, dailyDietId, status, pollUrl, createdAt, startedAt, finishedAt, alternatives]\n",
		"        status:\n          allOf:\n            - $ref: \"#/components/schemas/OptimizationStatus\"\n            - const: completed\n",
		"        startedAt:\n          type: string\n          format: date-time\n",
		"        finishedAt:\n          type: string\n          format: date-time\n",
		"        alternatives:\n          type: array\n          minItems: 1\n          maxItems: 3\n          items:\n            $ref: \"#/components/schemas/OptimizationAlternative\"\n",
	),
	"OptimizationJobFailed": (
		"      type: object\n",
		"      additionalProperties: false\n",
		"      required: [jobId, dailyDietId, status, pollUrl, createdAt, failure]\n",
		"        status:\n          allOf:\n            - $ref: \"#/components/schemas/OptimizationStatus\"\n            - const: failed\n",
		"        startedAt:\n          type:\n            - string\n            - \"null\"\n          format: date-time\n",
		"        finishedAt:\n          type:\n            - string\n            - \"null\"\n          format: date-time\n",
		"        alternatives:\n          type: array\n          maxItems: 3\n          items:\n            $ref: \"#/components/schemas/OptimizationAlternative\"\n",
		"        failure:\n          $ref: \"#/components/schemas/OptimizationFailure\"\n",
	),
	"OptimizationJobCancelled": (
		"      type: object\n",
		"      additionalProperties: false\n",
		"      required: [jobId, dailyDietId, status, pollUrl, createdAt, finishedAt]\n",
		"        status:\n          allOf:\n            - $ref: \"#/components/schemas/OptimizationStatus\"\n            - const: cancelled\n",
		"        finishedAt:\n          type: string\n          format: date-time\n",
	),
	"OptimizationJobStatusEnvelope": (
		"      type: object\n",
		"      required: [status, requestId, data]\n",
		"        status:\n          type: string\n          enum: [ok]\n",
		"        data:\n          $ref: \"#/components/schemas/OptimizationJobData\"\n",
	),
}

OPTIMIZATION_PROPERTY_NAMES = {
	"MealQuantity": {"mealId", "name", "quantity", "unit", "position"},
	"MacroProjection": {"protein", "carbohydrates", "fat", "calories"},
	"OptimizationAlternative": {"meals", "macros", "similarityScore"},
	"OptimizationFailure": {"code", "message"},
	"OptimizationJobAcknowledgementData": {"jobId", "status", "pollUrl"},
	"OptimizationJobAcknowledgementEnvelope": {"status", "requestId", "data"},
	"OptimizationJobQueued": {"jobId", "dailyDietId", "status", "pollUrl", "createdAt"},
	"OptimizationJobProcessing": {"jobId", "dailyDietId", "status", "pollUrl", "createdAt", "startedAt"},
	"OptimizationJobCompleted": {"jobId", "dailyDietId", "status", "pollUrl", "createdAt", "startedAt", "finishedAt", "alternatives"},
	"OptimizationJobFailed": {"jobId", "dailyDietId", "status", "pollUrl", "createdAt", "startedAt", "finishedAt", "alternatives", "failure"},
	"OptimizationJobCancelled": {"jobId", "dailyDietId", "status", "pollUrl", "createdAt", "finishedAt"},
	"OptimizationJobStatusEnvelope": {"status", "requestId", "data"},
}


def schema_block(source: str, name: str) -> str | None:
	"""Return one top-level component parameter or schema block."""
	matches = list(re.finditer(rf"(?ms)^    {re.escape(name)}:\n(.*?)(?=^    [A-Za-z][A-Za-z0-9]*:\n|\Z)", source))
	return matches[-1].group(1) if matches else None


def daily_diet_contract_mismatches(source: str) -> list[str]:
	"""Describe OpenAPI drift from assumptions enforced by the strict Daily Diet decoder."""
	mismatches = []
	for name, rules in DAILY_DIET_SCHEMA_RULES.items():
		block = schema_block(source, name)
		if block is None:
			mismatches.append(f"{name} is missing")
			continue
		for rule in rules:
			if rule not in block:
				mismatches.append(f"{name} drifted: missing {rule.strip()}")
		for rule, expected_count in DAILY_DIET_SCHEMA_RULE_COUNTS.get(name, {}).items():
			actual_count = block.count(rule)
			if actual_count != expected_count:
				mismatches.append(
					f"{name} drifted: expected {expected_count} occurrences of {rule.strip()}, found {actual_count}"
				)
		if expected_properties := DAILY_DIET_PROPERTY_NAMES.get(name):
			actual_properties = set(re.findall(r"(?m)^        ([A-Za-z][A-Za-z0-9]*):$", block))
			if actual_properties != expected_properties:
				mismatches.append(
					f"{name} properties drifted: expected {sorted(expected_properties)}, found {sorted(actual_properties)}"
				)
	create_operation = operation_block(source, "/api/v1/daily-diets", "post")
	parameter_ref = '        - $ref: "#/components/parameters/IdempotencyKey"\n'
	if create_operation is None or create_operation.count(parameter_ref) != 1:
		mismatches.append("Daily Diet create IdempotencyKey parameter reference drifted")
	return mismatches


def optimization_contract_mismatches(source: str) -> list[str]:
	"""Describe OpenAPI drift from assumptions enforced by the strict optimization decoder."""
	mismatches = []
	for name, rules in OPTIMIZATION_SCHEMA_RULES.items():
		block = schema_block(source, name)
		if block is None:
			mismatches.append(f"{name} is missing")
			continue
		for rule in rules:
			if rule not in block:
				mismatches.append(f"{name} drifted: missing {rule.strip()}")
		if expected_properties := OPTIMIZATION_PROPERTY_NAMES.get(name):
			actual_properties = set(re.findall(r"(?m)^        ([A-Za-z][A-Za-z0-9]*):$", block))
			if actual_properties != expected_properties:
				mismatches.append(
					f"{name} properties drifted: expected {sorted(expected_properties)}, found {sorted(actual_properties)}"
				)
	for name in ("OptimizationJobQueued", "OptimizationJobProcessing", "OptimizationJobCompleted", "OptimizationJobFailed", "OptimizationJobCancelled"):
		block = schema_block(source, name) or ""
		for field, fmt in (("jobId", "uuid"), ("dailyDietId", "uuid"), ("pollUrl", "uri-reference"), ("createdAt", "date-time")):
			rule = f"        {field}:\n          type: string\n          format: {fmt}\n"
			if rule not in block:
				mismatches.append(f"{name} drifted: missing {field} {fmt}")
	for name in ("OptimizationJobAcknowledgementEnvelope", "OptimizationJobStatusEnvelope"):
		block = schema_block(source, name) or ""
		match = re.search(r"(?m)^        requestId:\n((?:          [^\n]*\n)+)", block)
		if match is None or match.group(1) != OPTIMIZATION_REQUEST_ID_RULE:
			mismatches.append(f"{name} requestId bounds or safe-character policy drifted")
	post = operation_block(source, "/api/v1/optimization/jobs", "post") or ""
	if post.count('        - $ref: "#/components/parameters/IdempotencyKey"\n') != 1:
		mismatches.append("optimization submission IdempotencyKey parameter reference drifted")
	return mismatches


def operation_block(source: str, path: str, method: str) -> str | None:
	"""Return one OpenAPI operation block with original line endings."""
	lines = source.splitlines(keepends=True)
	try:
		path_start = lines.index(f"  {path}:\n")
	except ValueError:
		return None
	path_end = next((index for index in range(path_start + 1, len(lines)) if re.match(r"^  \S", lines[index])), len(lines))
	try:
		method_start = lines.index(f"    {method}:\n", path_start + 1, path_end)
	except ValueError:
		return None
	method_end = next((index for index in range(method_start + 1, path_end) if re.match(r"^    \S", lines[index])), path_end)
	return "".join(lines[method_start:method_end])


def app_error_contract_mismatches(source: str) -> list[str]:
	"""Describe OpenAPI AppError drift that would weaken runtime-safe mapping."""
	match = re.search(r"(?ms)^    AppError:\n(.*?)(?=^    Envelope:)", source)
	if match is None:
		return ["AppError schema is missing"]
	block = match.group(1)
	mismatches = []
	if "      required: [category, code, message, retryable]\n" not in block:
		mismatches.append("AppError required fields drifted")
	category_match = re.search(r"(?m)^          enum: \[([^]]+)]$", block)
	categories = tuple(value.strip() for value in category_match.group(1).split(",")) if category_match else ()
	if categories != APP_ERROR_CATEGORIES:
		mismatches.append(f"AppError category enum drifted: {categories}")
	if not re.search(r"(?m)^        retryable:\n          type: boolean$", block):
		mismatches.append("AppError retryable must remain boolean")
	if not re.search(r"(?m)^        requestId:\n          type: string$", block):
		mismatches.append("AppError requestId must remain string")
	return mismatches


def audited_operation_keys(source: str) -> set[tuple[str, str]]:
	"""Return every Daily Diet and optimization operation declared by OpenAPI."""
	operations = set()
	path = ""
	for line in source.splitlines():
		if match := re.match(r"^  (/[^:]+):$", line):
			path = match.group(1)
			continue
		if path and any(path == prefix or path.startswith(prefix + "/") for prefix in AUDITED_OPERATION_PREFIXES):
			if match := re.match(r"^    ([a-z]+):$", line):
				method = match.group(1)
				if method in HTTP_METHODS:
					operations.add((path, method))
	return operations


def operation_response_statuses(source: str, path: str, method: str) -> set[str]:
	"""Return explicitly declared response statuses for one OpenAPI operation."""
	lines = source.splitlines()
	path_line = f"  {path}:"
	method_line = f"    {method}:"
	try:
		path_start = lines.index(path_line)
	except ValueError:
		return set()
	path_end = next((index for index in range(path_start + 1, len(lines)) if re.match(r"^  \S", lines[index])), len(lines))
	try:
		method_start = lines.index(method_line, path_start + 1, path_end)
	except ValueError:
		return set()
	method_end = next((index for index in range(method_start + 1, path_end) if re.match(r"^    \S", lines[index])), path_end)
	try:
		responses_start = lines.index("      responses:", method_start + 1, method_end)
	except ValueError:
		return set()
	responses_end = next((index for index in range(responses_start + 1, method_end) if re.match(r"^      \S", lines[index])), method_end)
	return {
		match.group(1)
		for line in lines[responses_start + 1:responses_end]
		if (match := re.match(r'^        ["\']?([1-5](?:[0-9]{2}|XX)|default)["\']?:', line))
	}


def operation_response_mismatches(source: str) -> list[str]:
	"""Describe audited operation responses that differ from the route policy."""
	mismatches = []
	expected_operations = set(REQUIRED_OPERATION_RESPONSES)
	actual_operations = audited_operation_keys(source)
	for path, method in sorted(expected_operations - actual_operations):
		mismatches.append(f"missing audited operation: {method.upper()} {path}")
	for path, method in sorted(actual_operations - expected_operations):
		mismatches.append(f"unexpected audited operation: {method.upper()} {path}")
	for path, method in sorted(expected_operations & actual_operations):
		expected = REQUIRED_OPERATION_RESPONSES[(path, method)]
		actual = operation_response_statuses(source, path, method)
		if actual != expected:
			mismatches.append(f"{method.upper()} {path}: expected {sorted(expected)}, found {sorted(actual)}")
	return mismatches

GENERATED = """// Generated from api/openapi.yaml by scripts/generate-api-types.py.
// Implements DESIGN-017 ErrorMessageMapper shared frontend contracts.

export type ErrorCategory =
\t| "validation"
\t| "auth"
\t| "entitlement"
\t| "security"
\t| "network"
\t| "timeout"
\t| "server"
\t| "dependency"
\t| "rate_limit"
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
export interface Envelope<TData = unknown> {
\tstatus: string;
\trequestId: string;
\tdata?: TData;
\terror?: AppError | null;
}

// Implements DESIGN-014 UptimeMonitor liveness contract.
/** Process liveness payload. */
export interface HealthData {
\tservice: string;
}

// Implements DESIGN-014 UptimeMonitor readiness contract.
/** Dependency-readiness payload. */
export interface ReadinessData {
\tchecks: Record<string, string>;
}

// Implements DESIGN-006 AuthController CSRF token-delivery contract.
/** Session-bound synchronizer token delivered to SPA clients. */
export interface CSRFTokenData {
\tcsrfToken: string;
}

// Implements DESIGN-006 AuthController CSRF token-delivery contract.
/** CSRF token response envelope. */
export type CSRFTokenEnvelope = Envelope<CSRFTokenData>;

// Implements DESIGN-006 AuthController frontend auth contract.
/** Authenticated session metadata; token values are carried only by HttpOnly cookies. */
export interface AuthSessionData {
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
export interface VerifyEmailData {
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
export interface PasswordResetAcceptedData {
\taccepted: true;
}

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password-reset request acceptance envelope. */
export type PasswordResetRequestEnvelope = Envelope<PasswordResetAcceptedData>;

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password-reset completion payload. */
export interface PasswordResetConsumeData {
\treset: true;
}

// Implements DESIGN-006 AuthController frontend password-reset contract.
/** Password-reset completion envelope. */
export type PasswordResetConsumeEnvelope = Envelope<PasswordResetConsumeData>;

// Implements DESIGN-006 OAuthHandler frontend provider contract.
/** Supported OAuth identity providers. */
export type OAuthProvider = "google" | "apple";

// Implements DESIGN-018 AuthApiClient generated endpoint contract.
/** CSRF retrieval endpoint used before protected auth mutations. */
export const AUTH_CSRF_TOKEN_ENDPOINT = "/api/v1/auth/csrf-token" as const;

// Implements DESIGN-018 AuthApiClient generated endpoint contract.
/** Email/password registration endpoint. */
export const AUTH_REGISTER_ENDPOINT = "/api/v1/auth/register" as const;

// Implements DESIGN-018 AuthApiClient generated endpoint contract.
/** Email/password login endpoint. */
export const AUTH_LOGIN_ENDPOINT = "/api/v1/auth/login" as const;

// Implements DESIGN-018 AuthApiClient generated endpoint contract.
/** Current-session logout endpoint. */
export const AUTH_LOGOUT_ENDPOINT = "/api/v1/auth/logout" as const;

// Implements DESIGN-018 AuthApiClient generated endpoint contract.
/** Refresh-cookie session recovery endpoint. */
export const AUTH_REFRESH_ENDPOINT = "/api/v1/auth/refresh" as const;

// Implements DESIGN-018 AuthApiClient generated OAuth contract.
/** Builds the provider-specific OAuth start endpoint with an optional relative return path. */
export function buildOAuthStartUrl(provider: OAuthProvider, returnTo = "/"): string {
	const base = `/api/v1/auth/oauth/${provider}/start`;
	const safeReturnTo = safeOAuthReturnPath(returnTo);
	return safeReturnTo === "/" ? base : `${base}?return_to=${encodeURIComponent(safeReturnTo)}`;
}

function safeOAuthReturnPath(value: string): string {
	try {
		if (!value || !value.startsWith("/") || value.startsWith("//") || value.includes("\\\\")) {
			return "/";
		}
		const parsed = new URL(value, "http://frontend.local");
		if (parsed.origin !== "http://frontend.local" || !parsed.pathname.startsWith("/")) {
			return "/";
		}
		return `${parsed.pathname}${parsed.search}`;
	} catch {
		return "/";
	}
}

// Implements DESIGN-018 AuthApiClient generated request contract.
/** Shared JSON headers used by generated auth mutation helpers. */
export type AuthJsonMutationHeaders = Record<string, string> & {
	Accept: "application/json";
	"Content-Type": "application/json";
};

// Implements DESIGN-018 AuthApiClient generated request contract.
/** Shared JSON mutation request init for auth payload submissions. */
export interface AuthJsonMutationRequestInit extends Omit<RequestInit, "body" | "credentials" | "headers" | "method"> {
	method: "POST";
	credentials: "include";
	headers: AuthJsonMutationHeaders;
	body: string;
}

// Implements DESIGN-018 AuthApiClient generated request contract.
/** Credentialed GET request init for auth/session reads. */
export interface AuthGetRequestInit extends Omit<RequestInit, "credentials" | "headers" | "method"> {
	method: "GET";
	credentials: "include";
	headers: {
		Accept: "application/json";
	};
}

// Implements DESIGN-018 AuthApiClient generated request contract.
/** Credentialed POST request init for body-less auth mutations such as logout and refresh. */
export interface AuthPostRequestInit extends Omit<RequestInit, "credentials" | "headers" | "method"> {
	method: "POST";
	credentials: "include";
	headers: {
		Accept: "application/json";
		"X-CSRF-Token"?: string;
	};
}

// Implements DESIGN-018 AuthApiClient generated CSRF retrieval contract.
/** Builds the generated CSRF token retrieval request. */
export function buildCsrfTokenRequestInit(options: { signal?: AbortSignal } = {}): AuthGetRequestInit {
	return buildCredentialedGetRequestInit(options);
}

// Implements DESIGN-018 AuthApiClient generated registration contract.
/** Builds the generated email/password registration request. */
export function buildRegisterRequestInit(
	request: RegisterRequest,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): AuthJsonMutationRequestInit {
	return buildAuthJsonMutationRequestInit(request, options);
}

// Implements DESIGN-018 AuthApiClient generated login contract.
/** Builds the generated email/password login request. */
export function buildLoginRequestInit(
	request: LoginRequest,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): AuthJsonMutationRequestInit {
	return buildAuthJsonMutationRequestInit(request, options);
}

// Implements DESIGN-018 AuthApiClient generated logout contract.
/** Builds the generated current-session logout request. */
export function buildLogoutRequestInit(options: { csrfToken?: string; signal?: AbortSignal } = {}): AuthPostRequestInit {
	return buildAuthPostRequestInit(options);
}

// Implements DESIGN-018 AuthApiClient generated session recovery contract.
/** Builds the generated refresh-cookie session recovery request. */
export function buildRefreshSessionRequestInit(options: { signal?: AbortSignal } = {}): AuthPostRequestInit {
	return buildAuthPostRequestInit(options);
}

function buildAuthJsonMutationRequestInit<TRequest extends object>(
	request: TRequest,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): AuthJsonMutationRequestInit {
	const headers: AuthJsonMutationHeaders = {
		Accept: "application/json",
		"Content-Type": "application/json"
	};
	if (options.csrfToken) {
		headers["X-CSRF-Token"] = options.csrfToken;
	}
	return {
		method: "POST",
		credentials: "include",
		headers,
		body: JSON.stringify(request),
		signal: options.signal
	};
}

function buildAuthPostRequestInit(options: { csrfToken?: string; signal?: AbortSignal } = {}): AuthPostRequestInit {
	const headers: AuthPostRequestInit["headers"] = {
		Accept: "application/json"
	};
	if (options.csrfToken) {
		headers["X-CSRF-Token"] = options.csrfToken;
	}
	return {
		method: "POST",
		credentials: "include",
		headers,
		signal: options.signal
	};
}

function buildCredentialedGetRequestInit(options: { signal?: AbortSignal } = {}): AuthGetRequestInit {
	return {
		method: "GET",
		credentials: "include",
		headers: {
			Accept: "application/json"
		},
		signal: options.signal
	};
}

// Implements DESIGN-008 PreferenceManager frontend profile contract.
/** User profile and preference response data. */
export interface ProfileData {
\tuserId: string;
\tdisplayName: string;
\tunitSystem: "metric" | "imperial";
\tthemePreference: "system" | "light" | "dark";
\trequiresUnitRecalculation: boolean;
}

// Implements DESIGN-008 PreferenceManager frontend profile contract.
/** User profile response envelope. */
export type ProfileEnvelope = Envelope<ProfileData>;

// Implements DESIGN-018 AuthApiClient generated session probe contract.
/** Authenticated profile endpoint used as the frontend-safe session probe. */
export const PROFILE_ENDPOINT = "/api/v1/profile" as const;

// Implements DESIGN-018 AuthApiClient generated session probe contract.
/** Builds the generated profile/session probe request. */
export function buildProfileRequestInit(options: { signal?: AbortSignal } = {}): AuthGetRequestInit {
	return buildCredentialedGetRequestInit(options);
}

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
export interface SavedItemsData {
\titems: SavedItem[];
}

// Implements DESIGN-008 SavedDataRepository frontend saved-data contract.
/** Saved item collection response envelope. */
export type SavedItemsEnvelope = Envelope<SavedItemsData>;

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** Canonical quantity units accepted by saved daily-diet entries. */
export type CanonicalQuantityUnit = "g" | "ml" | "oz" | "fl_oz";

/** Distinguishes Food Items from Meals in Daily Diet entries. */
export type FoodObjectType = "food_item" | "meal";

/** One ordered Food Object quantity supplied to a saved Daily Diet. */
export interface FoodObjectQuantity {
	foodObjectId: string;
	foodObjectType: FoodObjectType;
	quantity: number;
	unit: CanonicalQuantityUnit;
	position: number;
}

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** One ordered meal quantity supplied to or returned from a saved diet. */
export interface MealQuantity {
	mealId: string;
	name: string;
	quantity: number;
	unit: CanonicalQuantityUnit;
	position: number;
}

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** One persisted saved-diet meal entry. */
export interface DailyDietFoodObjectEntry extends FoodObjectQuantity {
	id: string;
}

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** Server-derived aggregate macros and calories for one saved diet. */
export interface MacroProjection {
	protein: number;
	carbohydrates: number;
	fat: number;
	calories: number;
}

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** User-owned saved daily-diet collection; ownership is never client-supplied. */
export interface DailyDiet {
	id: string;
	name: string;
	entries: DailyDietFoodObjectEntry[];
	aggregateMacros: MacroProjection;
	createdAt: string;
	updatedAt: string;
}

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** Client-editable saved-diet fields with no authoritative aggregate totals. */
export interface DailyDietCreateRequest {
	name: string;
	entries: FoodObjectQuantity[];
}

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** Client-editable replacement fields with server-recalculated aggregates. */
export interface DailyDietReplaceRequest extends DailyDietCreateRequest {}

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** One saved-diet response envelope. */
export type DailyDietEnvelope = Envelope<DailyDiet>;

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** Saved-diet collection response payload. */
export interface DailyDietCollectionData {
	diets: DailyDiet[];
}

// Implements DESIGN-008 SavedDataRepository frontend daily-diet contract.
/** Saved-diet collection response envelope. */
export type DailyDietCollectionEnvelope = Envelope<DailyDietCollectionData>;

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet endpoint contract.
export const DAILY_DIETS_ENDPOINT = "/api/v1/daily-diets" as const;

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet endpoint contract.
/** Builds one user-scoped saved-diet URL. */
export function buildDailyDietUrl(dietId: string): string {
	return `${DAILY_DIETS_ENDPOINT}/${encodeURIComponent(dietId)}`;
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
export type DailyDietCreateHeaders = Record<string, string> & {
	Accept: "application/json";
	"Content-Type": "application/json";
	"Idempotency-Key": IdempotencyKey;
	"X-CSRF-Token"?: string;
};

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
export interface DailyDietCreateRequestInit extends Omit<RequestInit, "body" | "credentials" | "headers" | "method"> {
	method: "POST";
	credentials: "include";
	headers: DailyDietCreateHeaders;
	body: string;
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
/** Builds a CSRF- and idempotency-aware saved-diet creation request. */
export function buildDailyDietCreateRequestInit(
	request: DailyDietCreateRequest,
	idempotencyKey: IdempotencyKey,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): DailyDietCreateRequestInit {
	const headers: DailyDietCreateHeaders = {
		Accept: "application/json",
		"Content-Type": "application/json",
		"Idempotency-Key": idempotencyKey
	};
	if (options.csrfToken) headers["X-CSRF-Token"] = options.csrfToken;
	return { method: "POST", credentials: "include", headers, body: JSON.stringify(request), signal: options.signal };
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
/** Credentialed read request for a saved-diet collection or item. */
export interface DailyDietGetRequestInit extends Omit<RequestInit, "credentials" | "headers" | "method"> {
	method: "GET";
	credentials: "include";
	headers: { Accept: "application/json" };
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
/** Builds the generated saved-diet collection read request. */
export function buildDailyDietListRequestInit(options: { signal?: AbortSignal } = {}): DailyDietGetRequestInit {
	return { method: "GET", credentials: "include", headers: { Accept: "application/json" }, signal: options.signal };
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
/** Builds the generated saved-diet item read request. */
export function buildDailyDietGetRequestInit(options: { signal?: AbortSignal } = {}): DailyDietGetRequestInit {
	return buildDailyDietListRequestInit(options);
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
export type DailyDietMutationHeaders = Record<string, string> & {
	Accept: "application/json";
	"Content-Type": "application/json";
	"X-CSRF-Token"?: string;
};

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
export interface DailyDietReplaceRequestInit extends Omit<RequestInit, "body" | "credentials" | "headers" | "method"> {
	method: "PUT";
	credentials: "include";
	headers: DailyDietMutationHeaders;
	body: string;
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
/** Builds the generated CSRF-protected saved-diet replacement request. */
export function buildDailyDietReplaceRequestInit(
	request: DailyDietReplaceRequest,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): DailyDietReplaceRequestInit {
	const headers: DailyDietMutationHeaders = {
		Accept: "application/json",
		"Content-Type": "application/json"
	};
	if (options.csrfToken) headers["X-CSRF-Token"] = options.csrfToken;
	return { method: "PUT", credentials: "include", headers, body: JSON.stringify(request), signal: options.signal };
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
export interface DailyDietDeleteRequestInit extends Omit<RequestInit, "credentials" | "headers" | "method"> {
	method: "DELETE";
	credentials: "include";
	headers: { Accept: "application/json"; "X-CSRF-Token"?: string };
}

// Implements DESIGN-008 SavedDataRepository frontend authenticated daily-diet request contract.
/** Builds the generated CSRF-protected saved-diet deletion request. */
export function buildDailyDietDeleteRequestInit(options: { csrfToken?: string; signal?: AbortSignal } = {}): DailyDietDeleteRequestInit {
	const headers: DailyDietDeleteRequestInit["headers"] = { Accept: "application/json" };
	if (options.csrfToken) headers["X-CSRF-Token"] = options.csrfToken;
	return { method: "DELETE", credentials: "include", headers, signal: options.signal };
}

// Implements DESIGN-004 JobStatusTracker frontend optimization contract.
/** Asynchronous optimization submission for one server-owned saved diet. */
export interface DietOptimizationRequest {
	dailyDietId: string;
	tolerancePercent: number;
	excludedMealIds: string[];
}

// Implements DESIGN-004 JobStatusTracker frontend optimization contract.
export type OptimizationStatus = "queued" | "processing" | "completed" | "failed" | "cancelled";

// Implements DESIGN-004 JobStatusTracker frontend safe failure contract.
export type OptimizationFailureCode =
	| "failed_validation"
	| "solver_timeout"
	| "solver_infeasible"
	| "worker_crash";

// Implements DESIGN-004 JobStatusTracker frontend completed-alternative contract.
export interface OptimizationAlternative {
	meals: MealQuantity[];
	macros: MacroProjection;
	similarityScore: number;
}

// Implements DESIGN-004 JobStatusTracker frontend completed-alternative contract.
/** Type-level mirror of the OpenAPI maximum-three-alternatives constraint. */
export type OptimizationAlternativeList =
	| []
	| [OptimizationAlternative]
	| [OptimizationAlternative, OptimizationAlternative]
	| [OptimizationAlternative, OptimizationAlternative, OptimizationAlternative];

// Implements DESIGN-004 JobStatusTracker frontend completed-alternative contract.
/** A completed job must contain at least one and at most three alternatives. */
export type CompletedOptimizationAlternativeList =
	| [OptimizationAlternative]
	| [OptimizationAlternative, OptimizationAlternative]
	| [OptimizationAlternative, OptimizationAlternative, OptimizationAlternative];

// Implements DESIGN-004 JobStatusTracker frontend safe failure contract.
export interface OptimizationFailure {
	code: OptimizationFailureCode;
	message: string;
}

// Implements DESIGN-004 JobStatusTracker frontend 202 acknowledgement contract.
export interface OptimizationJobAcknowledgementData {
	jobId: string;
	status: "queued";
	pollUrl: string;
}

// Implements DESIGN-004 JobStatusTracker frontend 202 acknowledgement contract.
export type OptimizationJobAcknowledgementEnvelope = Envelope<OptimizationJobAcknowledgementData> & {
	status: "accepted";
};

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
/** Fields shared by every user-scoped optimization polling state. */
export interface OptimizationJobCommon {
	jobId: string;
	dailyDietId: string;
	pollUrl: string;
	createdAt: string;
}

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
/** Queued jobs contain acknowledgement metadata only. */
export interface OptimizationJobQueued extends OptimizationJobCommon {
	status: "queued";
	startedAt?: never;
	finishedAt?: never;
	alternatives?: never;
	failure?: never;
}

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
/** Processing jobs expose start metadata but no result or failure payload. */
export interface OptimizationJobProcessing extends OptimizationJobCommon {
	status: "processing";
	startedAt: string;
	finishedAt?: never;
	alternatives?: never;
	failure?: never;
}

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
/** Completed jobs require one to three alternatives and cannot carry a failure. */
export interface OptimizationJobCompleted extends OptimizationJobCommon {
	status: "completed";
	startedAt: string;
	finishedAt: string;
	alternatives: CompletedOptimizationAlternativeList;
	failure?: never;
}

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
/** Failed jobs require a safe failure and may retain validated partial alternatives. */
export interface OptimizationJobFailed extends OptimizationJobCommon {
	status: "failed";
	startedAt?: string | null;
	finishedAt?: string | null;
	alternatives?: OptimizationAlternativeList;
	failure: OptimizationFailure;
}

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
/** Cancelled jobs are terminal without alternatives or failure details. */
export interface OptimizationJobCancelled extends OptimizationJobCommon {
	status: "cancelled";
	finishedAt: string;
	startedAt?: never;
	alternatives?: never;
	failure?: never;
}

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
/** Discriminated polling union; status selects the only valid payload shape. */
export type OptimizationJobData =
	| OptimizationJobQueued
	| OptimizationJobProcessing
	| OptimizationJobCompleted
	| OptimizationJobFailed
	| OptimizationJobCancelled;

// Implements DESIGN-004 JobStatusTracker frontend polling contract.
export type OptimizationJobStatusEnvelope = Envelope<OptimizationJobData>;

// Implements DESIGN-004 JobStatusTracker frontend endpoint contract.
export const OPTIMIZATION_JOBS_ENDPOINT = "/api/v1/optimization/jobs" as const;

// Implements DESIGN-004 JobStatusTracker frontend endpoint contract.
/** Builds one user-scoped optimization polling URL. */
export function buildOptimizationJobUrl(jobId: string): string {
	return `${OPTIMIZATION_JOBS_ENDPOINT}/${encodeURIComponent(jobId)}`;
}

// Implements DESIGN-004 JobStatusTracker frontend submission request contract.
export type OptimizationSubmissionHeaders = Record<string, string> & {
	Accept: "application/json";
	"Content-Type": "application/json";
	"Idempotency-Key": IdempotencyKey;
	"X-CSRF-Token"?: string;
};

// Implements DESIGN-004 JobStatusTracker frontend submission request contract.
export interface OptimizationSubmissionRequestInit extends Omit<RequestInit, "body" | "credentials" | "headers" | "method"> {
	method: "POST";
	credentials: "include";
	headers: OptimizationSubmissionHeaders;
	body: string;
}

// Implements DESIGN-004 JobStatusTracker frontend submission request contract.
/** Builds a CSRF- and idempotency-aware asynchronous optimization request. */
export function buildOptimizationSubmissionRequestInit(
	request: DietOptimizationRequest,
	idempotencyKey: IdempotencyKey,
	options: { csrfToken?: string; signal?: AbortSignal } = {}
): OptimizationSubmissionRequestInit {
	const headers: OptimizationSubmissionHeaders = {
		Accept: "application/json",
		"Content-Type": "application/json",
		"Idempotency-Key": idempotencyKey
	};
	if (options.csrfToken) headers["X-CSRF-Token"] = options.csrfToken;
	return { method: "POST", credentials: "include", headers, body: JSON.stringify(request), signal: options.signal };
}

// Implements DESIGN-004 JobStatusTracker frontend polling request contract.
export interface OptimizationJobRequestInit extends Omit<RequestInit, "credentials" | "headers" | "method"> {
	method: "GET";
	credentials: "include";
	headers: { Accept: "application/json" };
}

// Implements DESIGN-004 JobStatusTracker frontend polling request contract.
/** Builds a credentialed optimization job polling request. */
export function buildOptimizationJobRequestInit(options: { signal?: AbortSignal } = {}): OptimizationJobRequestInit {
	return { method: "GET", credentials: "include", headers: { Accept: "application/json" }, signal: options.signal };
}

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
export interface SearchHistoryData {
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
export interface DeletionRequestData {
\trequestId: string;
\tstatus: "pending" | "processing" | "completed" | "failed";
}

// Implements DESIGN-008 AccountDeleter frontend deletion contract.
/** Account deletion response envelope. */
export type DeletionRequestEnvelope = Envelope<DeletionRequestData>;

// Implements DESIGN-015 DisclaimerRenderer frontend disclaimer contract.
/** Stable Markdown disclaimer content for login and account surfaces. */
export interface DisclaimerData {
\tlocation: "login" | "account";
\tversion: string;
\tmarkdown: string;
\tfallback: boolean;
\talert?: string;
}

// Implements DESIGN-015 DisclaimerRenderer frontend disclaimer contract.
/** Disclaimer response envelope. */
export type DisclaimerEnvelope = Envelope<DisclaimerData>;

// Implements DESIGN-018 AuthApiClient generated disclaimer contract.
/** Disclaimer endpoint used by login and account auth surfaces. */
export const DISCLAIMER_ENDPOINT = "/api/v1/disclaimers" as const;

// Implements DESIGN-018 AuthApiClient generated disclaimer contract.
/** Supported disclaimer locations documented by the OpenAPI contract. */
export type DisclaimerLocation = DisclaimerData["location"];

// Implements DESIGN-018 AuthApiClient generated disclaimer contract.
/** Builds the generated disclaimer URL with an explicit location query. */
export function buildDisclaimerUrl(location: DisclaimerLocation = "login"): `/api/v1/disclaimers?location=${DisclaimerLocation}` {
	return `${DISCLAIMER_ENDPOINT}?location=${location}`;
}

// Implements DESIGN-018 AuthApiClient generated disclaimer contract.
/** Builds the generated disclaimer retrieval request. */
export function buildDisclaimerRequestInit(options: { signal?: AbortSignal } = {}): AuthGetRequestInit {
	return buildCredentialedGetRequestInit(options);
}

// Implements DESIGN-007 SubscriptionController frontend billing endpoint contract.
/** Entitlement status endpoint path exported for generated-type-only frontend gating. */
export const BILLING_ENTITLEMENT_ENDPOINT = "/api/v1/billing/entitlement" as const;

// Implements DESIGN-007 SubscriptionController frontend billing endpoint contract.
/** Checkout creation endpoint path exported with its generated request helpers. */
export const BILLING_CHECKOUT_ENDPOINT = "/api/v1/billing/checkout" as const;

// Implements DESIGN-007 SubscriptionController frontend billing endpoint contract.
/** Billing portal creation endpoint path exported with its generated request helpers. */
export const BILLING_PORTAL_ENDPOINT = "/api/v1/billing/portal" as const;

// Implements DESIGN-007 SubscriptionController frontend checkout idempotency contract.
/** Stable client-generated idempotency key sent with checkout creation retries. */
export type IdempotencyKey = string;

// Implements DESIGN-017 ErrorMessageMapper frontend billing error contract.
/** Billing statuses documented by the OpenAPI billing and entitlement contract. */
export type BillingErrorStatus = 400 | 401 | 402 | 409 | 422 | 503;

// Implements DESIGN-017 ErrorMessageMapper frontend billing error contract.
/** User-safe billing error codes consumed by frontend billing and entitlement gates. */
export type BillingErrorCode =
\t| "billing_payment_required"
\t| "billing_recovery_required"
\t| "billing_portal_unavailable"
\t| "checkout_idempotency_conflict"
\t| "checkout_invalid_request"
\t| "checkout_validation_failed"
\t| "stripe_unavailable"
\t| "entitlement_unavailable";

// Implements DESIGN-017 ErrorMessageMapper frontend billing error contract.
/** Classified billing error envelope returned by checkout and entitlement endpoints. */
export interface BillingErrorEnvelope extends Envelope {
\tstatus: "error";
\terror: AppError & {
\t\tcategory: "auth" | "entitlement" | "validation" | "dependency";
\t\tcode: BillingErrorCode | (string & {});
\t};
}

// Implements DESIGN-007 SubscriptionController frontend checkout contract.
/** Public checkout billing period accepted by hosted checkout creation. */
export type CheckoutPlan = "monthly" | "annual";

// Implements DESIGN-007 SubscriptionController frontend checkout contract.
/** Hosted checkout creation request. Raw payment-card data is not accepted. */
export interface CheckoutCreateRequest {
\tplan: CheckoutPlan;
\tsuccessUrl: string;
\tcancelUrl: string;
}

// Implements DESIGN-007 SubscriptionController frontend checkout idempotency contract.
/** Headers required to create or replay checkout sessions without duplicate side effects. */
export type CheckoutCreateHeaders = Record<string, string> & {
\tAccept: "application/json";
\t"Content-Type": "application/json";
\t"Idempotency-Key": IdempotencyKey;
};

// Implements DESIGN-007 SubscriptionController frontend checkout idempotency contract.
/** RequestInit shape for the generated checkout creation helper. */
export interface CheckoutCreateRequestInit extends Omit<RequestInit, "body" | "credentials" | "headers" | "method"> {
\tmethod: "POST";
\tcredentials: "include";
\theaders: CheckoutCreateHeaders;
\tbody: string;
}

// Implements DESIGN-007 SubscriptionController frontend checkout idempotency contract.
/** Builds a checkout creation request with the required idempotency header. */
export function buildCheckoutCreateRequestInit(
\trequest: CheckoutCreateRequest,
\tidempotencyKey: IdempotencyKey,
\toptions: { csrfToken?: string; signal?: AbortSignal } = {}
): CheckoutCreateRequestInit {
\tconst headers: CheckoutCreateHeaders = {
\t\tAccept: "application/json",
\t\t"Content-Type": "application/json",
\t\t"Idempotency-Key": idempotencyKey
\t};
\tif (options.csrfToken) {
\t\theaders["X-CSRF-Token"] = options.csrfToken;
\t}
\treturn {
\t\tmethod: "POST",
\t\tcredentials: "include",
\t\theaders,
\t\tbody: JSON.stringify(request),
\t\tsignal: options.signal
\t};
}

// Implements DESIGN-007 SubscriptionController frontend checkout contract.
/** Sanitized hosted checkout session response. */
export interface CheckoutSessionData {
\tcheckoutSessionId: string;
\tcheckoutUrl: string;
\tplan: CheckoutPlan;
\tpriceId: string;
\tamountCents: number;
}

// Implements DESIGN-007 SubscriptionController frontend checkout contract.
/** Hosted checkout session response envelope. */
export type CheckoutSessionEnvelope = Envelope<CheckoutSessionData>;

// Implements DESIGN-007 SubscriptionController frontend billing portal contract.
/** Hosted billing portal creation request. */
export interface BillingPortalCreateRequest {
\treturnUrl: string;
}

// Implements DESIGN-007 SubscriptionController frontend billing portal contract.
/** Hosted billing portal creation request headers. */
export type BillingPortalCreateHeaders = Record<string, string> & {
\tAccept: "application/json";
\t"Content-Type": "application/json";
};

// Implements DESIGN-007 SubscriptionController frontend billing portal contract.
/** RequestInit shape for the generated billing portal creation helper. */
export interface BillingPortalCreateRequestInit extends Omit<RequestInit, "body" | "credentials" | "headers" | "method"> {
\tmethod: "POST";
\tcredentials: "include";
\theaders: BillingPortalCreateHeaders;
\tbody: string;
}

// Implements DESIGN-007 SubscriptionController frontend billing portal contract.
/** Builds a billing portal creation request. */
export function buildBillingPortalCreateRequestInit(
\trequest: BillingPortalCreateRequest,
\toptions: { csrfToken?: string; signal?: AbortSignal } = {}
): BillingPortalCreateRequestInit {
\tconst headers: BillingPortalCreateHeaders = {
\t\tAccept: "application/json",
\t\t"Content-Type": "application/json"
\t};
\tif (options.csrfToken) {
\t\theaders["X-CSRF-Token"] = options.csrfToken;
\t}
\treturn {
\t\tmethod: "POST",
\t\tcredentials: "include",
\t\theaders,
\t\tbody: JSON.stringify(request),
\t\tsignal: options.signal
\t};
}

// Implements DESIGN-007 SubscriptionController frontend billing portal contract.
/** Sanitized hosted billing portal session response. */
export interface BillingPortalSessionData {
\tportalUrl: string;
}

// Implements DESIGN-007 SubscriptionController frontend billing portal contract.
/** Hosted billing portal session response envelope. */
export type BillingPortalSessionEnvelope = Envelope<BillingPortalSessionData>;

// Implements DESIGN-007 StripeWebhookHandler frontend-visible webhook contract.
/** Verified Stripe webhook processing result. */
export interface StripeWebhookData {
\teventId: string;
\teventType: string;
\tduplicate: boolean;
}

// Implements DESIGN-007 StripeWebhookHandler frontend-visible webhook contract.
/** Stripe webhook processing response envelope. */
export type StripeWebhookEnvelope = Envelope<StripeWebhookData>;

// Implements DESIGN-007 SubscriptionController frontend entitlement contract.
/** Subscription tier exposed by entitlement status reads. */
export type SubscriptionTier = "free" | "trial" | "paid";

// Implements DESIGN-007 SubscriptionController frontend entitlement contract.
/** Persisted entitlement state exposed without provider identifiers. */
export type EntitlementState = "active" | "expired" | "past_due" | "cancelled";

// Implements DESIGN-007 SubscriptionController frontend billing-state contract.
/** Frontend-safe billing recovery state derived from provider status. */
export type BillingRecoveryState = "none" | "action_required" | "cancelled" | "expired";

// Implements DESIGN-007 SubscriptionController frontend entitlement contract.
/** Sanitized entitlement and billing status payload for the current user. */
export interface EntitlementStatusData {
\tuserId: string;
\ttier: SubscriptionTier;
\tstatus: EntitlementState;
\tallowedModes: SearchMode[];
\tsearchLimitPer24h: number;
\tusageUsed: number;
\tusageRemaining: number | null;
\tusageWindowStartedAt: string | null;
\ttrialExpiresAt: string | null;
\tbillingRecoveryState: BillingRecoveryState;
}

// Implements DESIGN-007 SubscriptionController frontend entitlement contract.
/** Entitlement status response envelope. */
export type EntitlementStatusEnvelope = Envelope<EntitlementStatusData>;

// Implements DESIGN-007 SubscriptionController frontend entitlement contract.
/** RequestInit shape for generated entitlement status reads. */
export interface EntitlementStatusRequestInit extends Omit<RequestInit, "credentials" | "headers" | "method"> {
\tmethod: "GET";
\tcredentials: "include";
\theaders: {
\t\tAccept: "application/json";
\t};
}

// Implements DESIGN-007 SubscriptionController frontend entitlement contract.
/** Builds an entitlement status request that consumes only generated entitlement types. */
export function buildEntitlementStatusRequestInit(
\toptions: { signal?: AbortSignal } = {}
): EntitlementStatusRequestInit {
\treturn {
\t\tmethod: "GET",
\t\tcredentials: "include",
\t\theaders: {
\t\t\tAccept: "application/json"
\t\t},
\t\tsignal: options.signal
\t};
}

// Implements DESIGN-002 SearchController frontend search-mode contract.
/** Supported search workflows exposed by the search API. */
export type SearchMode = "catalog" | "substitution" | "daily_diet" | "daily_diet_alternative";

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
export type SubstitutionUnit = CanonicalQuantityUnit;

// Implements DESIGN-002 SearchController frontend substitution contract.
/** Quantity-bearing food input for substitution searches. */
export interface SubstitutionInput {
\tfoodObjectId: string;
\tfoodObjectType?: FoodObjectType;
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

// Implements DESIGN-002 SearchController frontend classification summary contract.
/** Public classification identity summary for a search result item. */
export interface ClassificationSummary {
	id: string;
	name: string;
	kind: "food_category" | "culinary_role";
}

// Implements DESIGN-002 SearchController frontend macro profile contract.
/** Protein, carbohydrate, and fat macro values on a 100g or 100ml basis. */
export interface MacroProfile {
	protein: number;
	carbohydrates: number;
	fat: number;
}

// Implements DESIGN-002 SearchController frontend substitution source summary contract.
/** Macro and amount totals for the user's selected substitution input list. */
export interface SourceSummary {
\tmacros: MacroProfile;
\tcalories: number;
\ttotalGrams: number;
\ttotalMilliliters: number;
}

// Implements DESIGN-002 SearchController frontend food-object result contract.
/** Food object returned by search and autocomplete-related result flows. */
export interface FoodObject {
	id: string;
	objectType: FoodObjectType;
	name: string;
	physicalState: "solid" | "liquid";
	imageUrl?: string | null;
	classifications: ClassificationSummary[];
	primaryFoodCategory: ClassificationSummary | null;
	macros: MacroProfile;
	macroBasis: "100g" | "100ml";
	calories: number;
}

// Implements DESIGN-002 SearchController frontend food-object detail contract.
/** Successful food-object detail response envelope. */
export type FoodObjectEnvelope = Envelope<FoodObject>;

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
export interface SearchResponse {
\titems: FoodObject[];
\ttotalCount: number;
\tpage: number;
\tsimilarityScores: number[];
\tsimilarityMetadata: SimilarityMetadata[];
\tsourceSummary?: SourceSummary;
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
\tobjectType: FoodObjectType;
\tlabel: string;
\texactMatch: boolean;
\tlevenshteinDistance: number;
\tlength: number;
\trank: number;
}

// Implements DESIGN-002 SearchController frontend autocomplete contract.
/** Autocomplete payload with ranked suggestions and optional cache metadata. */
export interface AutocompleteResponse {
\titems: RankedAutocomplete[];
\tcache?: CacheMetadata;
}

// Implements DESIGN-002 SearchController frontend autocomplete contract.
/** Successful autocomplete response envelope. */
export type AutocompleteEnvelope = Envelope<AutocompleteResponse>;
"""


def generated_contract(source: str) -> str:
	"""Render shared quantity enums from the OpenAPI source of truth."""
	if source.count('$ref: "#/components/schemas/CanonicalQuantityUnit"') != 4:
		raise ValueError("all saved-diet and substitution units must reference CanonicalQuantityUnit")
	match = re.search(r"(?m)^    CanonicalQuantityUnit:\n(?:      .*\n)*?      enum: \[([^]]+)]$", source)
	if match is None:
		raise ValueError("OpenAPI CanonicalQuantityUnit enum is missing")
	units = [unit.strip() for unit in match.group(1).split(",")]
	if units != ["g", "ml", "oz", "fl_oz"]:
		raise ValueError(f"unexpected canonical quantity units: {units}")
	quantity_type = " | ".join(json.dumps(unit) for unit in units)
	return GENERATED.replace('export type CanonicalQuantityUnit = "g" | "ml" | "oz" | "fl_oz";', f"export type CanonicalQuantityUnit = {quantity_type};")


def main() -> int:
	parser = argparse.ArgumentParser(description="Generate shared frontend API types from the OpenAPI contract.")
	parser.add_argument("--check", action="store_true", help="Fail if generated frontend types have drifted.")
	args = parser.parse_args()
	source = OPENAPI.read_text(encoding="utf-8")
	missing = [marker for marker in REQUIRED_MARKERS if marker not in source]
	if missing:
		print(f"OpenAPI contract missing required markers: {missing}")
		return 1
	response_mismatches = operation_response_mismatches(source)
	if response_mismatches:
		print("OpenAPI audited operation response drift:\n" + "\n".join(response_mismatches))
		return 1
	error_contract_mismatches = app_error_contract_mismatches(source)
	if error_contract_mismatches:
		print("OpenAPI AppError contract drift:\n" + "\n".join(error_contract_mismatches))
		return 1
	daily_diet_mismatches = daily_diet_contract_mismatches(source)
	if daily_diet_mismatches:
		print("OpenAPI Daily Diet decoder contract drift:\n" + "\n".join(daily_diet_mismatches))
		return 1
	optimization_mismatches = optimization_contract_mismatches(source)
	if optimization_mismatches:
		print("OpenAPI optimization decoder contract drift:\n" + "\n".join(optimization_mismatches))
		return 1
	try:
		generated = generated_contract(source)
	except ValueError as error:
		print(error)
		return 1
	if args.check:
		if not OUTPUT.exists() or OUTPUT.read_text(encoding="utf-8") != generated:
			print(f"Generated API types are stale: run `python3 {Path(__file__).name}`")
			return 1
		print("Generated API types are current.")
		return 0
	OUTPUT.parent.mkdir(parents=True, exist_ok=True)
	OUTPUT.write_text(generated, encoding="utf-8")
	print(f"Generated {OUTPUT.relative_to(ROOT)}")
	return 0


if __name__ == "__main__":
	sys.exit(main())
