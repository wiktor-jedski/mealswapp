#!/usr/bin/env python3

# Implements DESIGN-001 SearchView and DESIGN-004 JobStatusTracker response-contract drift verification.

import importlib.util
import re
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
GENERATOR_PATH = ROOT / "scripts" / "generate-api-types.py"
SPEC = importlib.util.spec_from_file_location("generate_api_types", GENERATOR_PATH)
if SPEC is None or SPEC.loader is None:
	raise RuntimeError("could not load API type generator")
GENERATOR = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(GENERATOR)


class OperationResponseDriftTest(unittest.TestCase):
	def test_phase08_routes_security_statuses_and_safe_dtos_match_generated_types(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		self.assertEqual(GENERATOR.phase08_contract_mismatches(source), [])
		generated = GENERATOR.generated_contract(source)
		for symbol in (
			"ErrorEnvelope",
			*GENERATOR.PHASE08_SUCCESS_ENVELOPES,
			"CuratedImportRequest",
		):
			self.assertRegex(generated, rf"export (?:interface|type) {symbol}\b")
		for forbidden in ("rawPayload", "auditSnapshot", "ownerId", "passwordHash", "accessToken"):
			self.assertNotIn(forbidden, generated)

	def test_phase08_security_or_warning_drift_is_rejected(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		custom_post = GENERATOR.operation_block(source, "/api/v1/custom-items", "post") or ""
		without_csrf = source.replace(custom_post, custom_post.replace("          csrfHeader: []\n", "", 1), 1)
		self.assertTrue(any("csrfHeader" in item for item in GENERATOR.phase08_contract_mismatches(without_csrf)))
		unbounded_warning = source.replace("              maxItems: 4\n", "", 1)
		self.assertTrue(any("warning" in item.lower() for item in GENERATOR.phase08_contract_mismatches(unbounded_warning)))

	def test_phase08_classification_names_match_runtime_normalization(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		for schema in ("AdminClassificationRequest", "AdminClassification"):
			block = GENERATOR.schema_block(source, schema) or ""
			self.assertIn(GENERATOR.ADMIN_CLASSIFICATION_NAME_RULE, block)

	def test_phase08_classification_name_drift_is_rejected(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		for schema in ("AdminClassificationRequest", "AdminClassification"):
			block = GENERATOR.schema_block(source, schema) or ""
			for mutation in (
				GENERATOR.ADMIN_CLASSIFICATION_NAME_RULE.replace("maxLength: 120", "maxLength: 121"),
				GENERATOR.ADMIN_CLASSIFICATION_NAME_RULE.replace("          description:", "          x-description:"),
			):
				with self.subTest(schema=schema, mutation=mutation):
					mutated = source.replace(block, block.replace(GENERATOR.ADMIN_CLASSIFICATION_NAME_RULE, mutation, 1), 1)
					mismatches = GENERATOR.phase08_contract_mismatches(mutated)
					self.assertTrue(any(schema in mismatch for mismatch in mismatches), mismatches)

	def test_phase08_generated_success_envelopes_are_strict(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		generated = GENERATOR.generated_contract(source)
		self.assertIn('export interface OkEnvelope<TData> {\n\tstatus: "ok";\n\trequestId: string;\n\tdata: TData;\n}', generated)
		for alias in GENERATOR.PHASE08_SUCCESS_ENVELOPES:
			self.assertRegex(generated, rf"export type {alias} = OkEnvelope<")
			self.assertNotRegex(generated, rf"export type {alias} = Envelope<")

	def test_phase08_source_success_envelopes_cannot_be_weakened(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		for schema in GENERATOR.PHASE08_SUCCESS_ENVELOPES:
			block = GENERATOR.schema_block(source, schema)
			self.assertIsNotNone(block)
			for rule, mutation in (
				("object type", block.replace("      type: object\n", "      type: string\n", 1)),
				("closed object", block.replace("      additionalProperties: false\n", "      additionalProperties: true\n", 1)),
				("required data", block.replace("      required: [status, requestId, data]\n", "      required: [status, requestId]\n", 1)),
				("status const ok", block.replace("          const: ok\n", "          enum: [ok]\n", 1)),
				("data property", block.replace("        data:\n", "        payload:\n", 1)),
			):
				with self.subTest(schema=schema, rule=rule):
					self.assertNotEqual(mutation, block, f"{schema} lacks the expected {rule} source rule")
					mutated = source.replace(f"    {schema}:\n{block}", f"    {schema}:\n{mutation}", 1)
					mismatches = GENERATOR.phase08_contract_mismatches(mutated)
					self.assertTrue(any(schema in mismatch for mismatch in mismatches), mismatches)

			without_schema = source.replace(f"    {schema}:\n{block}", "", 1)
			with self.subTest(schema=schema, rule="schema exists"):
				mismatches = GENERATOR.phase08_contract_mismatches(without_schema)
				self.assertTrue(any(schema in mismatch for mismatch in mismatches), mismatches)

	def test_custom_item_name_and_classification_contracts_match_generated_types(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		self.assertEqual(GENERATOR.custom_item_contract_mismatches(source), [])
		generated = GENERATOR.generated_contract(source)
		self.assertIn("export interface CustomItem extends CustomItemRequest", generated)
		classification = generated[generated.index("export interface ClassificationSummary"):generated.index("export interface CustomItemRequest")]
		self.assertNotIn("parentId", classification)
		fields = GENERATOR.schema_block(source, "CustomItemFields") or ""
		name = fields[fields.index("        name:"):fields.index("        physicalState:")]
		pattern = re.search(r"pattern: '([^']+)'", name)
		self.assertIsNotNone(pattern)
		self.assertIsNotNone(re.fullmatch(pattern.group(1), " Tofu "))
		self.assertIsNone(re.fullmatch(pattern.group(1), "   "))
		self.assertIsNone(re.fullmatch(pattern.group(1), "bad\x00name"))

	def test_custom_item_name_or_parent_projection_drift_is_rejected(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		name_pattern = GENERATOR.CUSTOM_ITEM_NAME_RULE.splitlines(keepends=True)[-1]
		without_name_pattern = source.replace(name_pattern, "", 1)
		self.assertTrue(any("name" in item for item in GENERATOR.custom_item_contract_mismatches(without_name_pattern)))
		kind = "        kind:\n          type: string\n          enum: [food_category, culinary_role]\n"
		prefix, suffix = source.rsplit(kind, 1)
		mutated = prefix + kind + "        parentId:\n          type: string\n" + suffix
		self.assertTrue(any("properties" in item for item in GENERATOR.custom_item_contract_mismatches(mutated)))

	def test_runtime_error_contract_matches_generated_type_policy(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		self.assertEqual(GENERATOR.app_error_contract_mismatches(source), [])

	def test_malformed_retryability_and_category_contracts_are_rejected(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		for old, new, expected in (
			("          type: boolean\n        requestId:", "          type: string\n        requestId:", "retryable"),
			("dependency, rate_limit, unknown", "dependency, unknown", "category"),
		):
			with self.subTest(field=expected):
				mismatches = GENERATOR.app_error_contract_mismatches(source.replace(old, new, 1))
				self.assertTrue(any(expected in mismatch for mismatch in mismatches), mismatches)

	def test_optimization_terminal_vocabulary_and_similarity_projection_match_generated_contract(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		self.assertEqual(GENERATOR.optimization_contract_mismatches(source), [])
		generated = GENERATOR.generated_contract(source)
		for code in ("failed_validation", "solver_timeout", "solver_infeasible", "worker_crash"):
			self.assertIn(f'| "{code}"', generated)
		for non_terminal in ("queue_unavailable", "result_expired"):
			failure_block = generated[generated.index("export type OptimizationFailureCode ="):generated.index("export interface OptimizationAlternative")]
			self.assertNotIn(f'| "{non_terminal}"', failure_block)
		self.assertIn("enum: [failed_validation, solver_timeout, solver_infeasible, worker_crash]", source)
		self.assertIn("multipleOf: 0.0001", source)
		self.assertIn("Quantity-weighted Jaccard similarity", source)

	def test_deliberate_optimization_decoder_contract_drift_is_rejected(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")

		def mutate_schema(name: str, old: str, new: str) -> str:
			marker = f"    {name}:\n"
			block = GENERATOR.schema_block(source, name)
			self.assertIsNotNone(block)
			original = marker + block
			mutated = original.replace(old, new, 1)
			self.assertNotEqual(mutated, original)
			start = source.rindex(original)
			return source[:start] + mutated + source[start + len(original):]

		for schema, old, new in (
			("MealQuantity", "          multipleOf: 0.001\n", "          multipleOf: 0.01\n"),
			("MacroProjection", "          maximum: 1000000000\n", "          maximum: 999999999\n"),
			("OptimizationFailureCode", "worker_crash]", "queue_unavailable]"),
			("OptimizationAlternative", "          maxItems: 100\n", "          maxItems: 101\n"),
			("OptimizationAlternative", "          multipleOf: 0.0001\n", "          multipleOf: 0.001\n"),
			("OptimizationFailure", "          maxLength: 240\n", "          maxLength: 241\n"),
			("OptimizationJobAcknowledgementData", "          format: uuid\n", "          type: integer\n"),
			("OptimizationJobAcknowledgementEnvelope", "          enum: [accepted]\n", "          enum: [ok]\n"),
			("OptimizationJobData", "propertyName: status", "propertyName: state"),
			("OptimizationJobQueued", "      additionalProperties: false\n", "      additionalProperties: true\n"),
			("OptimizationJobProcessing", "            - const: processing\n", "            - const: queued\n"),
			("OptimizationJobCompleted", "          minItems: 1\n", "          minItems: 0\n"),
			("OptimizationJobFailed", "            - \"null\"\n", "            - integer\n"),
			("OptimizationJobCancelled", "createdAt, finishedAt]", "createdAt]"),
			("OptimizationJobStatusEnvelope", "          enum: [ok]\n", "          enum: [accepted]\n"),
		):
			with self.subTest(schema=schema, mutation=old):
				mismatches = GENERATOR.optimization_contract_mismatches(mutate_schema(schema, old, new))
				self.assertTrue(any(schema in mismatch for mismatch in mismatches), mismatches)

	def test_optimization_submission_must_retain_caller_key_parameter(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		start = source.index("  /api/v1/optimization/jobs:\n")
		end = source.index("  /api/v1/optimization/jobs/{jobId}:\n", start)
		mutated = source[:start] + source[start:end].replace(
			'        - $ref: "#/components/parameters/IdempotencyKey"\n',
			'        - $ref: "#/components/parameters/OAuthProvider"\n',
			1,
		) + source[end:]

		mismatches = GENERATOR.optimization_contract_mismatches(mutated)

		self.assertTrue(any("IdempotencyKey" in mismatch for mismatch in mismatches), mismatches)

	def test_optimization_request_id_bounds_and_safe_characters_cannot_drift(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		request_id_rule = (
			"        requestId:\n"
			"          type: string\n"
			"          minLength: 1\n"
			"          maxLength: 120\n"
			"          pattern: '^[A-Za-z0-9._:-]+$'\n"
		)
		for schema in ("OptimizationJobAcknowledgementEnvelope", "OptimizationJobStatusEnvelope"):
			block = GENERATOR.schema_block(source, schema)
			self.assertIsNotNone(block)
			self.assertIn(request_id_rule, block)
			for expected, unsafe_rule in (
				("too-short", request_id_rule.replace("minLength: 1", "minLength: 0")),
				("too-long", request_id_rule.replace("maxLength: 120", "maxLength: 121")),
				("reviewer-stricter-maximum", request_id_rule.replace("maxLength: 120", "maxLength: 10")),
				("unsafe", request_id_rule.replace("'^[A-Za-z0-9._:-]+$'", "'^.*$'")),
			):
				with self.subTest(schema=schema, mutation=expected):
					mutated = source.replace(f"    {schema}:\n{block}", f"    {schema}:\n{block.replace(request_id_rule, unsafe_rule, 1)}", 1)
					mismatches = GENERATOR.optimization_contract_mismatches(mutated)
					self.assertTrue(any(schema in mismatch for mismatch in mismatches), mismatches)

	def test_generated_output_drift_is_detected(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		generated = GENERATOR.generated_contract(source)
		checked_in = (ROOT / "frontend" / "src" / "lib" / "api" / "generated.ts").read_text(encoding="utf-8")
		self.assertEqual(generated, checked_in)
		self.assertNotEqual(generated, checked_in.replace('status: "queued";', 'status: "waiting";', 1))

	def test_current_contract_matches_audited_response_matrix(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		self.assertEqual(GENERATOR.operation_response_mismatches(source), [])

	def test_rate_limit_category_matches_generated_contract(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		generated = GENERATOR.generated_contract(source)
		self.assertIn("dependency, rate_limit, unknown", source)
		self.assertIn('| "rate_limit"', generated)

	def test_collection_list_404_is_audited(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		operation = ("/api/v1/daily-diets", "get")
		self.assertIn("404", GENERATOR.REQUIRED_OPERATION_RESPONSES[operation])
		self.assertIn("404", GENERATOR.operation_response_statuses(source, *operation))

	def test_daily_diet_success_policy_and_generated_decoder_bounds_cannot_drift_silently(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		generated = GENERATOR.generated_contract(source)
		self.assertEqual(GENERATOR.daily_diet_contract_mismatches(source), [])
		self.assertEqual(GENERATOR.operation_response_statuses(source, "/api/v1/daily-diets", "get") & {"200", "201", "204"}, {"200"})
		self.assertEqual(GENERATOR.operation_response_statuses(source, "/api/v1/daily-diets", "post") & {"200", "201", "204"}, {"201"})
		self.assertEqual(GENERATOR.operation_response_statuses(source, "/api/v1/daily-diets/{dietId}", "get") & {"200", "201", "204"}, {"200"})
		self.assertEqual(GENERATOR.operation_response_statuses(source, "/api/v1/daily-diets/{dietId}", "put") & {"200", "201", "204"}, {"200"})
		self.assertEqual(GENERATOR.operation_response_statuses(source, "/api/v1/daily-diets/{dietId}", "delete") & {"200", "201", "204"}, {"204"})
		for contract in (
			"enum: [g, ml, oz, fl_oz]",
			"maximum: 1000000",
			"multipleOf: 0.001",
			"maximum: 99",
			"maxItems: 100",
			"maximum: 1000000000",
		):
			self.assertIn(contract, source)
		self.assertIn('export type CanonicalQuantityUnit = "g" | "ml" | "oz" | "fl_oz";', generated)
		self.assertIn("buildDailyDietListRequestInit", generated)
		self.assertIn("buildDailyDietCreateRequestInit", generated)
		self.assertIn("buildDailyDietDeleteRequestInit", generated)

	def test_deliberate_daily_diet_decoder_contract_drift_is_rejected(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")

		def mutate_schema(name: str, old: str, new: str) -> str:
			marker = f"    {name}:\n"
			block = GENERATOR.schema_block(source, name)
			self.assertIsNotNone(block)
			original = marker + block
			mutated = original.replace(old, new, 1)
			self.assertNotEqual(mutated, original)
			start = source.rindex(original)
			return source[:start] + mutated + source[start + len(original):]

		for old, new, schema in (
			("        maxLength: 255\n", "        maxLength: 256\n", "IdempotencyKey"),
			("        type: string\n        minLength: 8\n", "        type: integer\n        minLength: 8\n", "IdempotencyKey"),
			("      enum: [g, ml, oz, fl_oz]\n", "      enum: [g, ml]\n", "CanonicalQuantityUnit"),
			("      type: string\n      enum: [g, ml, oz, fl_oz]\n", "      type: integer\n      enum: [g, ml, oz, fl_oz]\n", "CanonicalQuantityUnit"),
			("        id:\n          type: string\n", "        id:\n          type: integer\n", "DailyDietFoodObjectEntry"),
			("        foodObjectId:\n          type: string\n", "        foodObjectId:\n          type: integer\n", "DailyDietFoodObjectEntry"),
			("        foodObjectType:\n          $ref: \"#/components/schemas/FoodObjectType\"\n", "        foodObjectType:\n          type: string\n", "DailyDietFoodObjectEntry"),
			("        quantity:\n          type: number\n", "        quantity:\n          type: string\n", "DailyDietFoodObjectEntry"),
			("          multipleOf: 0.001\n", "          multipleOf: 0.01\n", "DailyDietFoodObjectEntry"),
			("        unit:\n          $ref: \"#/components/schemas/CanonicalQuantityUnit\"\n", "        unit:\n          type: string\n", "DailyDietFoodObjectEntry"),
			("        position:\n          type: integer\n", "        position:\n          type: number\n", "DailyDietFoodObjectEntry"),
			("        protein:\n          type: number\n", "        protein:\n          type: string\n", "MacroProjection"),
			("          maximum: 1000000000\n", "          maximum: 999999999\n", "MacroProjection"),
			("        id:\n          type: string\n", "        id:\n          type: integer\n", "DailyDiet"),
			("        name:\n          type: string\n", "        name:\n          type: integer\n", "DailyDiet"),
			("        entries:\n          type: array\n", "        entries:\n          type: object\n", "DailyDiet"),
			("          maxItems: 100\n", "          maxItems: 101\n", "DailyDiet"),
			("            $ref: \"#/components/schemas/DailyDietFoodObjectEntry\"\n", "            $ref: \"#/components/schemas/MealQuantity\"\n", "DailyDiet"),
			("        aggregateMacros:\n          $ref: \"#/components/schemas/MacroProjection\"\n", "        aggregateMacros:\n          type: object\n", "DailyDiet"),
			("        createdAt:\n          type: string\n", "        createdAt:\n          type: integer\n", "DailyDiet"),
			("        status:\n          type: string\n", "        status:\n          type: integer\n", "DailyDietEnvelope"),
			("          enum: [ok]\n", "          enum: [accepted]\n", "DailyDietEnvelope"),
			("        requestId:\n          type: string\n", "        requestId:\n          type: integer\n", "DailyDietEnvelope"),
			("        data:\n          $ref: \"#/components/schemas/DailyDiet\"\n", "        data:\n          type: object\n", "DailyDietEnvelope"),
			("        requestId:\n          type: string\n", "        requestId:\n          type: integer\n", "DailyDietCollectionEnvelope"),
			("        data:\n          type: object\n", "        data:\n          type: array\n", "DailyDietCollectionEnvelope"),
			("            diets:\n              type: array\n", "            diets:\n              type: object\n", "DailyDietCollectionEnvelope"),
			("                $ref: \"#/components/schemas/DailyDiet\"\n", "                $ref: \"#/components/schemas/MealQuantity\"\n", "DailyDietCollectionEnvelope"),
			("        updatedAt:\n          type: string\n          format: date-time\n", "        updatedAt:\n          type: string\n          format: date-time\n        debug:\n          type: string\n", "DailyDiet"),
		):
			with self.subTest(schema=schema):
				mutated = mutate_schema(schema, old, new)
				mismatches = GENERATOR.daily_diet_contract_mismatches(mutated)
				self.assertTrue(any(schema in mismatch for mismatch in mismatches), mismatches)

	def test_daily_diet_create_must_retain_the_idempotency_parameter_reference(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		start = source.index("  /api/v1/daily-diets:\n")
		end = source.index("  /api/v1/daily-diets/{dietId}:\n", start)
		mutated_operation = source[start:end].replace(
			'        - $ref: "#/components/parameters/IdempotencyKey"\n',
			'        - $ref: "#/components/parameters/OAuthProvider"\n',
			1,
		)
		self.assertNotEqual(mutated_operation, source[start:end])
		mutated = source[:start] + mutated_operation + source[end:]

		mismatches = GENERATOR.daily_diet_contract_mismatches(mutated)

		self.assertTrue(any("IdempotencyKey" in mismatch for mismatch in mismatches), mismatches)

	def test_deliberate_response_mismatch_is_rejected(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		start = source.index("  /api/v1/optimization/jobs:\n")
		end = source.index("  /api/v1/optimization/jobs/{jobId}:\n", start)
		mutated = source[:start] + source[start:end].replace('        "429":\n          $ref: "#/components/responses/TooManyRequests"\n', "", 1) + source[end:]

		mismatches = GENERATOR.operation_response_mismatches(mutated)

		self.assertEqual(len(mismatches), 1)
		self.assertIn("POST /api/v1/optimization/jobs", mismatches[0])
		self.assertIn("'429'", mismatches[0])

	def test_deliberate_extra_audited_operation_is_rejected(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		marker = "    post:\n      operationId: postApiV1DailyDiet\n"
		extra = "    patch:\n      operationId: patchApiV1DailyDiets\n      responses:\n        \"418\":\n          $ref: \"#/components/responses/Error\"\n"
		mutated = source.replace(marker, extra + marker, 1)

		mismatches = GENERATOR.operation_response_mismatches(mutated)

		self.assertEqual(mismatches, ["unexpected audited operation: PATCH /api/v1/daily-diets"])

	def test_wildcard_response_keys_are_rejected_by_exact_policy(self) -> None:
		source = (ROOT / "api" / "openapi.yaml").read_text(encoding="utf-8")
		marker = '        "202":\n          $ref: "#/components/responses/OptimizationJobAcknowledgement"\n'
		for wildcard in ("1XX", "2XX", "3XX", "4XX", "5XX"):
			with self.subTest(wildcard=wildcard):
				mutated = source.replace(marker, marker + f'        "{wildcard}":\n          $ref: "#/components/responses/Error"\n', 1)

				mismatches = GENERATOR.operation_response_mismatches(mutated)

				self.assertEqual(len(mismatches), 1)
				self.assertIn("POST /api/v1/optimization/jobs", mismatches[0])
				self.assertIn(repr(wildcard), mismatches[0])


if __name__ == "__main__":
	unittest.main()
