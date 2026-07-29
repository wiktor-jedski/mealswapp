import importlib.util
import unittest
import re
from pathlib import Path
from types import SimpleNamespace
from unittest import mock

SPEC = importlib.util.spec_from_file_location("task283", Path(__file__).with_name("run-task283-acceptance.py"))
assert SPEC and SPEC.loader
module = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(module)

class Task283SafetyTests(unittest.TestCase):
    def test_spec_names_exactly_the_44_task283_criteria_and_uses_requirement_roots(self):
        source = (Path(__file__).parents[1] / "frontend/tests/task283-manual-catalog.spec.ts").read_text()
        ids = set(re.findall(r"P08-SWR\d{3}-(?:STEP|ACCEPT)-\d{2}", source))
        self.assertEqual(ids, set(module.CRITERIA))
        self.assertNotIn("ROOT-T283-ACCEPTANCE-INFRASTRUCTURE", source)
        for root in module.ROOTS.values():
            self.assertIn(root, source)

    def test_runner_preserves_only_task280_backend_keys_and_adds_proof_as_evidence(self):
        source = Path(module.__file__).read_text()
        self.assertIn("BACKEND_KEYS", source)
        self.assertIn("task283-proof-index.json", source)
        self.assertIn("operation_proof", source)
        self.assertNotIn("globalFoodItemCount", source)

    def test_shared_helper_does_not_break_task281_and_task282_producers(self):
        helper = (Path(__file__).parents[1] / "frontend/tests/task281-acceptance-helpers.ts").read_text()
        self.assertNotIn("unsafe backend evidence summary", helper)
        self.assertIn("recordAcceptance", helper)

    def test_reporter_allowlists_task280_backend_summary_keys(self):
        reporter = (Path(__file__).parents[1] / "frontend/tests/phase08-acceptance-reporter.ts").read_text()
        for key in ("http_status", "rollback_state", "request_correlation"):
            self.assertIn(f'"{key}"', reporter)
        self.assertIn("BACKEND_EVIDENCE_KEYS.has", reporter)
    def test_accepts_only_loopback_run_owned_test_database(self):
        module.parse_run_owned_database("postgres://127.0.0.1:5432/mealswapp_e2e_abc123_test")

    def test_rejects_development_database(self):
        with self.assertRaises(ValueError): module.parse_run_owned_database("postgres://127.0.0.1:5432/mealswapp")

    def test_rejects_remote_database(self):
        with self.assertRaises(ValueError): module.parse_run_owned_database("postgres://example.test:5432/mealswapp_e2e_abc_test")

    def test_rejects_deceptive_userinfo_query_and_path(self):
        for value in (
            "postgres://127.0.0.1@evil.test:5432/mealswapp_e2e_abc_test",
            "postgres://127.0.0.1:5432/mealswapp_e2e_abc_test?sslmode=disable",
            "postgres://127.0.0.1:5432/mealswapp_e2e_abc_test/extra",
        ):
            with self.assertRaises(ValueError): module.parse_run_owned_database(value)

    def test_rejects_mutating_evidence_sql(self):
        with self.assertRaises(ValueError): module.read_only_sql("UPDATE food_items SET name='x'")

    def test_manifest_criteria_are_unique_and_complete(self):
        self.assertEqual(len(module.CRITERIA), 44)
        self.assertEqual(len(set(module.CRITERIA)), 44)
        self.assertEqual(set(module.REQUIREMENT_CRITERIA), module.TASK_REQUIREMENTS)

    def test_harness_owns_real_stack_lifecycle_and_declares_two_api_contract(self):
        self.assertTrue(issubclass(module.Task283Harness, module.real_stack.Harness))
        self.assertFalse(hasattr(module, "task282"))
        self.assertIn("api-2", module.Task283Harness.start_application_stack.__doc__)

    def test_nonpass_criteria_use_requirement_specific_task283_roots(self):
        import tempfile, json
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory); (path / "browser.json").write_text(json.dumps({"results": []}))
            module.Task283Harness.combine_results(path)
            results = json.loads((path / "results.json").read_text())["results"]
            for result in results:
                requirement = next(k for k, ids in module.REQUIREMENT_CRITERIA.items() if result["criterionId"] in ids)
                self.assertEqual(result["rootCauseId"], module.ROOTS["P08-SWR" + requirement.split("-")[-1]])

    def test_finalize_status_precedence(self):
        import tempfile, json
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory); payload = {"results": [{"criterionId": c, "status": "PASS"} for c in module.CRITERIA]}
            payload["results"][0]["status"] = "BLOCKED"; payload["results"][1]["status"] = "FAIL"
            (path / "results.json").write_text(json.dumps(payload))
            self.assertEqual(module.finalize_reports("abc", path, True), 1)

    def test_read_only_proof_rejects_mutation(self):
        with self.assertRaises(ValueError): module.read_only_sql("UPDATE food_items SET name='x'")

    def test_runner_bootstraps_admin_before_browser_and_verifies_target(self):
        source = Path(module.__file__).read_text()
        self.assertIn('"./cmd/admin-bootstrap"', source)
        self.assertIn("administrator bootstrap returned an unexpected target", source)
        self.assertIn('"actor=operator"', source)

    def test_proof_uses_global_private_partition_not_global_owner_column(self):
        source = Path(module.__file__).read_text()
        self.assertIn("custom_food_items", source)
        self.assertNotIn("food_items WHERE owner_id", source)

    def test_redis_proof_reads_generation_key_and_independent_operation_snapshots(self):
        source = Path(module.__file__).read_text()
        self.assertIn("classification:cache-generation:v1", source)
        self.assertIn("redis_generation_actual", source)
        self.assertNotIn("actual[key_name] = expected[key_name]", source)

    def test_redis_proof_rejects_missing_or_altered_independent_snapshot(self):
        import json
        import tempfile
        before_id = "12345678-1234-4234-8234-123456789abc"
        after_id = "abcdefab-1234-4234-8234-123456789abc"
        expected = {"generationBefore": "4", "generationAfter": "4", "generationDelta": 0}
        ids = {"generationBefore": before_id, "generationAfter": after_id}
        with tempfile.TemporaryDirectory() as directory:
            for snapshot_id in (before_id, after_id):
                (Path(directory) / f"{snapshot_id}.json").write_text(json.dumps({
                    "schema": "mealswapp.task283-redis-observation.v1",
                    "id": snapshot_id,
                    "key": "classification:cache-generation:v1",
                    "value": "4",
                }))
            self.assertEqual(module.redis_generation_actual(expected, ids, Path(directory))["generationDelta"], 0)
            path = Path(directory) / f"{after_id}.json"
            path.write_text(path.read_text().replace('"4"', '"5"'))
            actual = module.redis_generation_actual(expected, ids, Path(directory))
            self.assertIn("generationAfter:value", module.compare_expected(expected, actual))
            self.assertIn("generationDelta:value", module.compare_expected(expected, actual))
            path.unlink()
            with self.assertRaisesRegex(ValueError, "unavailable"):
                module.redis_generation_actual(expected, ids, Path(directory))

    def test_redis_proof_rejects_incomplete_snapshot_identities(self):
        import tempfile
        expected = {"generationBefore": "4", "generationAfter": "4", "generationDelta": 0}
        complete = {
            "generationBefore": "12345678-1234-4234-8234-123456789abc",
            "generationAfter": "abcdefab-1234-4234-8234-123456789abc",
        }
        cases = (
            None,
            {},
            {"generationBefore": complete["generationBefore"]},
            {**complete, "unexpected": "aaaaaaaa-1234-4234-8234-123456789abc"},
        )
        with tempfile.TemporaryDirectory() as directory:
            for snapshot_ids in cases:
                with self.subTest(snapshot_ids=snapshot_ids):
                    with self.assertRaisesRegex(ValueError, "snapshots are incomplete"):
                        module.redis_generation_actual(expected, snapshot_ids, Path(directory))

    def test_redis_proof_rejects_duplicate_snapshot_identities(self):
        import tempfile
        snapshot_id = "12345678-1234-4234-8234-123456789abc"
        expected = {"generationBefore": "4", "generationAfter": "4", "generationDelta": 0}
        ids = {"generationBefore": snapshot_id, "generationAfter": snapshot_id}
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaisesRegex(ValueError, "identities are not unique"):
                module.redis_generation_actual(expected, ids, Path(directory))

    def test_redis_proof_rejects_malformed_snapshot_identities_and_observations(self):
        import json
        import tempfile
        before_id = "12345678-1234-4234-8234-123456789abc"
        after_id = "abcdefab-1234-4234-8234-123456789abc"
        expected = {"generationBefore": "4", "generationAfter": "4", "generationDelta": 0}
        valid_ids = {"generationBefore": before_id, "generationAfter": after_id}
        valid_observation = {
            "schema": "mealswapp.task283-redis-observation.v1",
            "id": before_id,
            "key": "classification:cache-generation:v1",
            "value": "4",
        }
        malformed_identities = (None, 123, "", "../snapshot", "not-a-uuid")
        malformed_observations = (
            ("invalid-json", "{", "unavailable"),
            ("not-an-object", json.dumps([]), "invalid"),
            ("schema", json.dumps({**valid_observation, "schema": "wrong"}), "invalid"),
            ("id", json.dumps({**valid_observation, "id": after_id}), "invalid"),
            ("key", json.dumps({**valid_observation, "key": "wrong"}), "invalid"),
            ("value-type", json.dumps({**valid_observation, "value": 4}), "invalid"),
            ("value-format", json.dumps({**valid_observation, "value": "4x"}), "invalid"),
        )
        with tempfile.TemporaryDirectory() as directory:
            observation_directory = Path(directory)
            (observation_directory / f"{after_id}.json").write_text(json.dumps({
                **valid_observation,
                "id": after_id,
            }))
            for identity in malformed_identities:
                ids = {**valid_ids, "generationBefore": identity}
                with self.subTest(identity=identity):
                    with self.assertRaisesRegex(ValueError, "snapshot identity is invalid"):
                        module.redis_generation_actual(expected, ids, observation_directory)
            for name, contents, error in malformed_observations:
                (observation_directory / f"{before_id}.json").write_text(contents)
                with self.subTest(observation=name):
                    with self.assertRaisesRegex(ValueError, f"observation is {error}"):
                        module.redis_generation_actual(expected, valid_ids, observation_directory)
        self.assertEqual(expected, {"generationBefore": "4", "generationAfter": "4", "generationDelta": 0})

    def test_browser_failure_is_diagnosed_and_results_are_still_combined(self):
        source = Path(module.__file__).read_text()
        self.assertIn("browser-diagnostics.txt", source)
        self.assertIn("write_synthetic_browser", source)
        self.assertIn('self.combine_results(evidence, "BLOCKED"', source)

    def test_managed_browser_output_is_mandatory_and_both_projects_are_required(self):
        source = Path(module.__file__).read_text()
        self.assertIn('if not browser_result.is_file()', source)
        self.assertIn('real-stack-mobile-chromium', source)
        self.assertIn('browser_payload.get("projects", [])', source)

    def test_timeout_and_signal_finalizer_emit_all_mapped_rows(self):
        import tempfile, json
        with tempfile.TemporaryDirectory() as directory:
            harness = object.__new__(module.Task283Harness)
            harness.artifacts = Path(directory)
            harness.run_id = "a" * 24
            self.assertEqual(module.finalize_failure_reports(harness, True), 2)
            results = json.loads((Path(directory) / "acceptance/results.json").read_text())["results"]
            self.assertEqual(len(results), 44)
            self.assertTrue(all(item["status"] == "BLOCKED" for item in results))
            self.assertEqual({item["criterionId"] for item in results}, set(module.CRITERIA))
            for requirement, criteria in module.REQUIREMENT_CRITERIA.items():
                shard = json.loads((Path(directory) / f"acceptance/{requirement}.json").read_text())["results"]
                self.assertEqual({item["criterionId"] for item in shard}, set(criteria))

    def test_main_finalizes_timeout_and_report_signal_paths(self):
        for run_error, report_error in ((TimeoutError("setup timeout"), None), (None, InterruptedError("SIGTERM"))):
            with self.subTest(run_error=run_error, report_error=report_error):
                harness = SimpleNamespace(run_id="a" * 24, artifacts=Path("/tmp/task283-finalizer-test"))
                harness.run = mock.Mock(side_effect=run_error)
                with (
                    mock.patch.object(module.real_stack, "install_signal_handlers", return_value={}),
                    mock.patch.object(module.real_stack, "restore_signal_handlers"),
                    mock.patch.object(module.real_stack, "validate_environment"),
                    mock.patch.object(module.real_stack.PostgresTarget, "parse", return_value=object()),
                    mock.patch.object(module, "Task283Harness", return_value=harness),
                    mock.patch.object(module, "finalize_reports", side_effect=report_error),
                    mock.patch.object(module, "finalize_failure_reports", return_value=2) as finalize_failure,
                ):
                    self.assertEqual(module.main(["--defer-report"]), 2)
                finalize_failure.assert_called_once_with(harness, True)

    def test_operation_expectations_detect_stale_write_and_generation_changes(self):
        self.assertEqual(module.compare_expected({"staleWriteStatus": 409}, {"staleWriteStatus": 200}), ["staleWriteStatus:value"])
        self.assertIn("generationDelta:value", module.compare_expected({"generationDelta": 1}, {"generationDelta": 0}))

    def test_playwright_config_registers_task283_reporter_capability_and_mobile(self):
        config = (Path(module.__file__).parents[1] / "frontend/playwright.real-stack.config.ts").read_text()
        self.assertIn("task283", config)
        self.assertIn("validateHarnessCapability(\"283\")", config)
        self.assertIn("task281 || task282 || task283", config)
        self.assertIn("task281 || task283", config)

    def test_parameterized_proof_uses_psql_variables(self):
        source = Path(module.__file__).read_text()
        self.assertIn('variables.extend((f"--set=p{index}=', source)
        self.assertIn('f":\'p{index}\'"', source)
        self.assertNotIn('queriesParameterized": True', source)
