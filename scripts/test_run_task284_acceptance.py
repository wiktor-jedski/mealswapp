import importlib.util
import io
import json
import re
import tempfile
import unittest
from contextlib import redirect_stdout
from pathlib import Path
from unittest import mock

SPEC = importlib.util.spec_from_file_location("task284", Path(__file__).with_name("run-task284-acceptance.py"))
assert SPEC and SPEC.loader
module = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(module)


class Task284AcceptanceTests(unittest.TestCase):
    def write_results(self, evidence, statuses=None):
        statuses = statuses or {}
        (evidence / "results.json").write_text(json.dumps({
            "results": [
                {
                    "criterionId": criterion,
                    "status": statuses[criterion][0] if isinstance(statuses.get(criterion), tuple) else statuses.get(criterion, "PASS"),
                    **({"rootCauseId": statuses[criterion][1]} if isinstance(statuses.get(criterion), tuple) else {}),
                }
                for criterion in module.CRITERIA
            ]
        }))

    def test_manifest_and_spec_cover_exact_task_criteria(self):
        source = (Path(__file__).parents[1] / "frontend/tests/task284-private-erasure.spec.ts").read_text()
        identifiers = set(re.findall(r"P08-SWR\d{3}-(?:STEP|ACCEPT)-\d{2}", source))
        self.assertEqual(identifiers, set(module.CRITERIA))
        self.assertEqual(len(module.CRITERIA), 17)
        self.assertEqual(set(module.REQUIREMENT_CRITERIA), module.TASK_REQUIREMENTS)

    def test_snapshot_request_rejects_extra_malformed_and_mismatched_targets(self):
        identifier = "12345678-1234-4234-8234-123456789abc"
        user_b = "22345678-1234-4234-8234-123456789abc"
        capability = {"nonce": "run-owned-nonce", "userA": identifier, "userB": user_b}
        value = {
            "schema": "mealswapp.task284-snapshot-request.v2",
            "id": identifier,
            "capabilityNonce": capability["nonce"],
            "phase": "before",
            "userA": identifier,
            "userB": user_b,
            "itemA": identifier,
            "deletedItemA": "32345678-1234-4234-8234-123456789abc",
            "itemB": "42345678-1234-4234-8234-123456789abc",
            "globalItem": "52345678-1234-4234-8234-123456789abc",
            "deletionRequest": None,
        }
        self.assertEqual(module.validate_snapshot_request(value, identifier, capability), value)
        for invalid in (
            {**value, "extra": True},
            {**value, "phase": "during"},
            {**value, "userA": "mealswapp"},
            {**value, "userA": user_b},
            {**value, "capabilityNonce": "arbitrary"},
        ):
            with self.subTest(invalid=invalid):
                with self.assertRaises(ValueError):
                    module.validate_snapshot_request(invalid, identifier, capability)
        with self.assertRaises(ValueError):
            module.validate_snapshot_request(value, "abcdefab-1234-4234-8234-123456789abc", capability)

    def test_snapshot_collection_rejects_arbitrary_database_targets_and_rebinding(self):
        identifier = "12345678-1234-4234-8234-123456789abc"
        request = {
            "phase": "before",
            "userA": identifier,
            "userB": "22345678-1234-4234-8234-123456789abc",
            "itemA": "32345678-1234-4234-8234-123456789abc",
            "deletedItemA": "42345678-1234-4234-8234-123456789abc",
            "itemB": "52345678-1234-4234-8234-123456789abc",
            "globalItem": "62345678-1234-4234-8234-123456789abc",
            "deletionRequest": None,
        }
        harness = object.__new__(module.Task284Harness)
        harness.bound_targets = {}
        harness.bound_deletion_request = None
        harness.proof_query_observations = []
        harness.target = object()
        harness.database = "owned"
        original = module.task283.read_only_psql
        module.task283.read_only_psql = lambda *_args, **_kwargs: "0"
        try:
            with self.assertRaisesRegex(ValueError, "not the exact run-owned fixture"):
                harness.collect_snapshot(request)
        finally:
            module.task283.read_only_psql = original
        harness.bound_targets = {
            key: request[key] for key in ("itemA", "deletedItemA", "itemB", "globalItem")
        }
        with self.assertRaisesRegex(ValueError, "differ from the run-owned fixture binding"):
            harness.collect_snapshot({**request, "itemB": identifier})

    def test_nonpass_results_use_requirement_specific_synchronized_roots(self):
        with tempfile.TemporaryDirectory() as directory:
            evidence = Path(directory)
            (evidence / "browser.json").write_text(json.dumps({"results": []}))
            module.Task284Harness.combine_results(evidence)
            results = json.loads((evidence / "results.json").read_text())["results"]
            self.assertEqual(len(results), len(module.CRITERIA))
            for result in results:
                requirement = next(key for key, criteria in module.REQUIREMENT_CRITERIA.items() if result["criterionId"] in criteria)
                self.assertEqual(result["status"], "BLOCKED")
                self.assertEqual(result["rootCauseId"], module.ROOTS[requirement])

    def test_backend_proof_failure_overrides_browser_pass(self):
        with tempfile.TemporaryDirectory() as directory:
            evidence = Path(directory)
            backend = evidence / "backend"
            backend.mkdir()
            (backend / "task284-proof.json").write_text(json.dumps({"assertionFailures": ["a.userExists:value"]}))
            (evidence / "browser.json").write_text(json.dumps({
                "results": [{"criterionId": criterion, "status": "PASS"} for criterion in module.CRITERIA]
            }))
            module.Task284Harness.combine_results(evidence)
            self.assertTrue(all(
                item["status"] == "FAIL"
                for item in json.loads((evidence / "results.json").read_text())["results"]
            ))

    def test_failure_finalizer_emits_every_mapped_row(self):
        with tempfile.TemporaryDirectory() as directory:
            harness = object.__new__(module.Task284Harness)
            harness.artifacts = Path(directory)
            harness.run_id = "a" * 24
            with redirect_stdout(io.StringIO()):
                self.assertEqual(module.finalize_failure_reports(harness, False), 2)
            results = json.loads((Path(directory) / "acceptance/results.json").read_text())["results"]
            self.assertEqual({item["criterionId"] for item in results}, set(module.CRITERIA))
            self.assertTrue(all(item["status"] == "BLOCKED" for item in results))

    def test_default_finalization_prints_each_requirement_and_skips_report_validation(self):
        with tempfile.TemporaryDirectory() as directory:
            evidence = Path(directory)
            statuses = {module.REQUIREMENT_CRITERIA["SW-REQ-043"][0]: ("FAIL", module.ROOTS["SW-REQ-043"])}
            self.write_results(evidence, statuses)
            output = io.StringIO()
            with redirect_stdout(output), mock.patch.object(module.subprocess, "run") as run:
                code = module.finalize_reports("a" * 24, evidence, False)
        self.assertEqual(code, 1)
        self.assertEqual(output.getvalue().splitlines(), [
            "SW-REQ-043 fail FAIL ROOT-T284-PRIVATE-ISOLATION",
            "SW-REQ-072 ok",
            "SW-REQ-073 ok",
        ])
        run.assert_not_called()

    def test_report_flag_runs_all_report_validations_without_masking_requirement_failure(self):
        with tempfile.TemporaryDirectory() as directory:
            evidence = Path(directory)
            statuses = {module.REQUIREMENT_CRITERIA["SW-REQ-043"][0]: ("FAIL", module.ROOTS["SW-REQ-043"])}
            self.write_results(evidence, statuses)
            with mock.patch.object(module.subprocess, "run", side_effect=[
                mock.Mock(returncode=0), mock.Mock(returncode=0), mock.Mock(returncode=0),
            ]) as run, redirect_stdout(io.StringIO()):
                code = module.finalize_reports("a" * 24, evidence, True)
        self.assertEqual(code, 1)
        self.assertEqual(run.call_count, len(module.REQUIREMENT_CRITERIA))

    def test_main_maps_default_and_report_cli_flags(self):
        for arguments, report in (([], False), (["--report"], True)):
            with self.subTest(arguments=arguments):
                harness = mock.Mock(run_id="a" * 24, artifacts=Path("/tmp/task284-cli-test"))
                with (
                    mock.patch.object(module.real_stack, "install_signal_handlers", return_value={}),
                    mock.patch.object(module.real_stack, "restore_signal_handlers"),
                    mock.patch.object(module.real_stack, "validate_environment"),
                    mock.patch.object(module.real_stack.PostgresTarget, "parse", return_value=object()),
                    mock.patch.object(module, "Task284Harness", return_value=harness),
                    mock.patch.object(module, "finalize_reports", return_value=0) as finalize,
                ):
                    self.assertEqual(module.main(arguments), 0)
                finalize.assert_called_once_with(
                    harness.run_id, harness.artifacts / "acceptance", report
                )

    def test_runner_uses_production_worker_and_parameterized_read_only_proof(self):
        source = Path(module.__file__).read_text()
        self.assertIn('"./cmd/worker"', source)
        self.assertIn('self.start_process("worker"', source)
        self.assertIn("task283.read_only_psql", source)
        self.assertIn("snapshot targets differ from the run-owned fixture binding", source)
        self.assertIn("deletion request is not bound to run-owned user A", source)
        self.assertNotIn("FLUSHALL", source)
        self.assertNotRegex(source, r"run_command\([^)]*\bTRUNCATE\b")

    def test_generated_client_has_private_lifecycle_export_and_deletion_builders(self):
        generated = (Path(__file__).parents[1] / "frontend/src/lib/api/generated.ts").read_text()
        generator = (Path(__file__).parents[1] / "scripts/generate-api-types.py").read_text()
        for symbol in (
            "buildCustomItemMutationRequestInit",
            "buildAccountExportRequestInit",
            "buildAccountDeletionRequestInit",
        ):
            self.assertIn(symbol, generated)
            self.assertIn(symbol, generator)

    def test_playwright_config_requires_private_task284_capability(self):
        source = (Path(__file__).parents[1] / "frontend/playwright.real-stack.config.ts").read_text()
        self.assertIn('validateHarnessCapability("284")', source)
        self.assertIn('"281" | "282" | "283" | "284"', source)


if __name__ == "__main__":
    unittest.main()
