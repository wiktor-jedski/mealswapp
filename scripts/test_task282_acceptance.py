"""Contract tests for the Task 282 controlled-provider acceptance harness."""

# Implements DESIGN-009 DataImporter acceptance lifecycle verification.

from __future__ import annotations

import importlib.util
import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]


def load(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"cannot load {path}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


task282 = load("task282_acceptance", ROOT / "scripts/run-task282-acceptance.py")
fixture = load("task282_provider_fixture", ROOT / "scripts/task282_provider_fixture.py")


class Task282AcceptanceTests(unittest.TestCase):
    """Verify complete fail-closed result production without external services."""

    def test_manifest_surface_is_complete_and_unique(self) -> None:
        self.assertEqual(len(task282.CRITERIA), 24)
        self.assertEqual(len(set(task282.CRITERIA)), 24)
        self.assertEqual(
            set(task282.REQUIREMENT_CRITERIA),
            {"SW-REQ-033", "SW-REQ-055", "SW-REQ-090"},
        )

    def test_missing_browser_output_blocks_every_criterion_and_splits_requirements(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            evidence = Path(directory)
            task282.Task282Harness.combine_results(evidence)
            results = json.loads((evidence / "results.json").read_text())["results"]
            self.assertEqual(len(results), 24)
            self.assertEqual({item["status"] for item in results}, {"BLOCKED"})
            self.assertEqual(
                {item["rootCauseId"] for item in results},
                {task282.INFRASTRUCTURE_ROOT},
            )
            for requirement, criteria in task282.REQUIREMENT_CRITERIA.items():
                selected = json.loads((evidence / f"{requirement}.json").read_text())["results"]
                self.assertEqual([item["criterionId"] for item in selected], list(criteria))

    def test_known_product_root_is_preserved_but_unknown_root_fails_closed(self) -> None:
        values = [
            {
                "criterionId": criterion,
                "status": "FAIL",
                "rootCauseId": (
                    "ROOT-T282-OFF-METADATA"
                    if criterion == "P08-SWR055-STEP-02"
                    else "unsafe-unregistered-root"
                ),
                "requestIds": [],
                "evidence": [],
                "backendEvidence": [],
            }
            for criterion in task282.CRITERIA
        ]
        with tempfile.TemporaryDirectory() as directory:
            evidence = Path(directory)
            (evidence / "browser.json").write_text(json.dumps({"results": values}))
            task282.Task282Harness.combine_results(evidence)
            results = {
                item["criterionId"]: item
                for item in json.loads((evidence / "results.json").read_text())["results"]
            }
            self.assertEqual(
                results["P08-SWR055-STEP-02"]["rootCauseId"],
                "ROOT-T282-OFF-METADATA",
            )
            self.assertEqual(
                results["P08-SWR055-STEP-01"]["rootCauseId"],
                task282.INFRASTRUCTURE_ROOT,
            )

    def test_duplicate_browser_criteria_fail_closed_instead_of_overwriting(self) -> None:
        for earlier, later in (("FAIL", "PASS"), ("PASS", "FAIL")):
            with self.subTest(earlier=earlier, later=later), tempfile.TemporaryDirectory() as directory:
                evidence = Path(directory)
                duplicated = [
                    {
                        "criterionId": task282.CRITERIA[0],
                        "status": status,
                        "rootCauseId": "ROOT-T282-OFF-METADATA",
                    }
                    for status in (earlier, later)
                ]
                (evidence / "browser.json").write_text(json.dumps({"results": duplicated}))
                task282.Task282Harness.combine_results(evidence)
                results = json.loads((evidence / "results.json").read_text())["results"]
                self.assertEqual({item["status"] for item in results}, {"BLOCKED"})
                self.assertEqual(
                    {item["rootCauseId"] for item in results},
                    {task282.INFRASTRUCTURE_ROOT},
                )

    def test_ownerless_projection_and_validation_reject_private_identity(self) -> None:
        self.assertIn("SELECT name, owner_id FROM custom_food_items", task282.OWNERLESS_COUNT_SQL)
        self.assertIn("owner_id IS NULL", task282.OWNERLESS_COUNT_SQL)
        base = {
            "foodItemCount": 1,
            "curatedImportCount": 1,
            "auditCount": 1,
            "ownerlessCount": 1,
            "editedCount": 1,
            "redisReachable": True,
        }
        self.assertTrue(task282.persistence_evidence_is_valid(base))
        for field, value in (("ownerlessCount", 0), ("foodItemCount", 0)):
            with self.subTest(field=field):
                mismatch = {**base, field: value}
                self.assertFalse(task282.persistence_evidence_is_valid(mismatch))

    def test_task282_flag_without_owned_capability_fails_closed(self) -> None:
        environment = {
            **os.environ,
            "MEALSWAPP_TASK282_REAL_E2E": "1",
            "MEALSWAPP_REAL_STACK_MANAGED": "1",
            "MEALSWAPP_REAL_STACK_BASE_URL": "http://127.0.0.1:65534",
            "PHASE08_ACCEPTANCE_RESULT_DIR": "/tmp/mealswapp-task282-missing-capability",
        }
        environment.pop("MEALSWAPP_TASK282_CAPABILITY_FILE", None)
        environment.pop("MEALSWAPP_TASK282_CAPABILITY_NONCE", None)
        result = subprocess.run(
            [
                "bunx", "playwright", "test", "-c",
                "playwright.real-stack.config.ts", "--list",
                "tests/task282-external-curation.spec.ts",
            ],
            cwd=ROOT / "frontend",
            env=environment,
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("managed isolated harness", result.stderr + result.stdout)

    def test_task282_capability_rejects_foreign_base_url(self) -> None:
        run_id, nonce = "1" * 24, "2" * 48
        with tempfile.TemporaryDirectory() as directory:
            capability = Path(directory) / "capability.json"
            evidence = ROOT / "logs/real-stack-e2e" / run_id / "acceptance"
            capability.write_text(json.dumps({
                "schema": "mealswapp.task282-harness-capability.v1",
                "runId": run_id,
                "nonce": nonce,
                "baseURL": "http://127.0.0.1:41000",
                "evidenceRoot": str(evidence),
                "frontendProcess": {"pid": os.getpid(), "startToken": "1"},
            }))
            capability.chmod(0o600)
            environment = {
                **os.environ,
                "MEALSWAPP_TASK282_REAL_E2E": "1",
                "MEALSWAPP_REAL_STACK_MANAGED": "1",
                "MEALSWAPP_REAL_STACK_BASE_URL": "http://127.0.0.1:65534",
                "MEALSWAPP_TASK282_CAPABILITY_FILE": str(capability),
                "MEALSWAPP_TASK282_CAPABILITY_NONCE": nonce,
                "PHASE08_ACCEPTANCE_RESULT_DIR": str(evidence),
            }
            result = subprocess.run(
                [
                    "bunx", "playwright", "test", "-c",
                    "playwright.real-stack.config.ts", "--list",
                    "tests/task282-external-curation.spec.ts",
                ],
                cwd=ROOT / "frontend",
                env=environment,
                check=False,
                capture_output=True,
                text=True,
            )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("does not match the harness capability", result.stderr + result.stdout)

    def test_browser_product_nonpass_is_truthful_in_diagnostics(self) -> None:
        harness = object.__new__(task282.Task282Harness)
        harness.events = ["browser_product_nonpass"]
        with mock.patch.object(task282.real_stack.Harness, "export_diagnostics") as export:
            harness.export_diagnostics("passed")
        export.assert_called_once_with("product_nonpass")

    def test_fixture_shapes_include_supported_nutrients_and_known_metadata(self) -> None:
        usda = fixture.usda_food()
        off = fixture.off_product(metadata=True)
        malformed = fixture.off_product(malformed=True)
        self.assertEqual(usda["fdcId"], 282001)
        self.assertEqual(off["nutriments"]["energy_modifier"], "~")
        self.assertEqual(malformed["nutriments"]["proteins_100g"], "bad")

    def test_optional_usda_fixture_mixes_usable_and_unusable_portion_evidence(self) -> None:
        food = fixture.usda_food()
        food["foodMeasures"] = [
            {"gramWeight": 224, "disseminationText": "1 cup", "amount": None, "measureUnit": None},
            {"gramWeight": 12, "disseminationText": "variable portion", "amount": None, "measureUnit": None},
        ]
        self.assertEqual(len(food["foodMeasures"]), 2)
        rejected = fixture.usda_food()
        rejected["fdcId"] = 0
        self.assertEqual(rejected["fdcId"], 0)


if __name__ == "__main__":
    unittest.main()
