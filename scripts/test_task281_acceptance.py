"""Contract tests for the Task 281 isolated SW-REQ-054 acceptance runner.

Implements DESIGN-009 AdminController acceptance-runner verification.
"""

from __future__ import annotations

import importlib.util
import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

MODULE_PATH = Path(__file__).with_name("run-task281-acceptance.py")
SPEC = importlib.util.spec_from_file_location("run_task281_acceptance", MODULE_PATH)
assert SPEC and SPEC.loader
task281 = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = task281
SPEC.loader.exec_module(task281)


class Task281AcceptanceTests(unittest.TestCase):
    """Verify complete criterion output and the safe read-only evidence boundary."""

    def test_combiner_emits_each_sw_req_054_criterion_once(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "backend").mkdir()
            (root / "backend/admin-authorization.json").write_text(
                "{}\n", encoding="utf-8"
            )
            for phase, criteria in (
                ("prebootstrap", task281.PRE_CRITERIA),
                ("postbootstrap", task281.POST_CRITERIA),
            ):
                (root / f"{phase}.json").write_text(
                    json.dumps(
                        {
                            "results": [
                                {
                                    "criterionId": criterion,
                                    "status": "PASS",
                                    "requestIds": [],
                                    "evidence": [],
                                    "backendEvidence": [],
                                }
                                for criterion in criteria
                            ]
                        }
                    ),
                    encoding="utf-8",
                )
            task281.Task281Harness.combine_results(root)
            results = json.loads(
                (root / "results.json").read_text(encoding="utf-8")
            )["results"]
        expected = set(task281.PRE_CRITERIA + task281.POST_CRITERIA)
        self.assertEqual(9, len(results))
        self.assertEqual(expected, {result["criterionId"] for result in results})
        self.assertTrue(
            all(
                result["evidence"]
                == [
                    {
                        "type": "backend",
                        "path": "backend/admin-authorization.json",
                    }
                ]
                for result in results
            )
        )
        resolved = next(
            result
            for result in results
            if result["criterionId"] == "P08-SWR054-STEP-02"
        )
        self.assertEqual(
            [task281.RESOLVED_MOBILE_NAVIGATION_ROOT],
            resolved["resolvedRootCauseIds"],
        )

    def test_missing_phase_is_blocked_never_skipped_or_passed(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            task281.Task281Harness.combine_results(root)
            results = json.loads(
                (root / "results.json").read_text(encoding="utf-8")
            )["results"]
        self.assertEqual(9, len(results))
        self.assertEqual({"BLOCKED"}, {result["status"] for result in results})
        self.assertEqual(
            {task281.INFRASTRUCTURE_ROOT},
            {result["rootCauseId"] for result in results},
        )

    def test_task_flag_without_managed_harness_fails_closed(self) -> None:
        environment = {
            **os.environ,
            "MEALSWAPP_TASK281_REAL_E2E": "1",
        }
        environment.pop("MEALSWAPP_REAL_STACK_MANAGED", None)
        environment.pop("MEALSWAPP_REAL_STACK_BASE_URL", None)
        result = subprocess.run(
            [
                "bunx",
                "playwright",
                "test",
                "-c",
                "playwright.real-stack.config.ts",
                "--list",
                "tests/task281-prebootstrap.spec.ts",
            ],
            cwd=MODULE_PATH.parents[1] / "frontend",
            env=environment,
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertNotEqual(0, result.returncode)
        self.assertIn("managed isolated harness", result.stderr + result.stdout)

    def test_foreign_base_url_is_rejected_by_harness_capability(self) -> None:
        run_id = "1" * 24
        nonce = "2" * 48
        with tempfile.TemporaryDirectory() as directory:
            capability_path = Path(directory) / "capability.json"
            capability_path.write_text(
                json.dumps(
                    {
                        "schema": "mealswapp.task281-harness-capability.v1",
                        "runId": run_id,
                        "nonce": nonce,
                        "baseURL": "http://127.0.0.1:41000",
                        "evidenceRoot": str(
                            MODULE_PATH.parents[1]
                            / "logs/real-stack-e2e"
                            / run_id
                            / "acceptance"
                        ),
                        "frontendProcess": {"pid": os.getpid(), "startToken": "1"},
                    }
                ),
                encoding="utf-8",
            )
            capability_path.chmod(0o600)
            environment = {
                **os.environ,
                "MEALSWAPP_TASK281_REAL_E2E": "1",
                "MEALSWAPP_REAL_STACK_MANAGED": "1",
                "MEALSWAPP_REAL_STACK_BASE_URL": "http://127.0.0.1:65530",
                "MEALSWAPP_TASK281_CAPABILITY_FILE": str(capability_path),
                "MEALSWAPP_TASK281_CAPABILITY_NONCE": nonce,
                "PHASE08_ACCEPTANCE_RESULT_DIR": str(
                    MODULE_PATH.parents[1]
                    / "logs/real-stack-e2e"
                    / run_id
                    / "acceptance"
                ),
            }
            result = subprocess.run(
                [
                    "bunx",
                    "playwright",
                    "test",
                    "-c",
                    "playwright.real-stack.config.ts",
                    "--list",
                    "tests/task281-prebootstrap.spec.ts",
                ],
                cwd=MODULE_PATH.parents[1] / "frontend",
                env=environment,
                check=False,
                capture_output=True,
                text=True,
            )
        self.assertNotEqual(0, result.returncode)
        self.assertIn("does not match the harness capability", result.stderr + result.stdout)

    def test_failure_before_acceptance_attachment_publishes_synchronized_report(self) -> None:
        criterion = task281.PRE_CRITERIA[0]
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            evidence = root / "evidence"
            evidence.mkdir()
            script = (
                'import Reporter from "./tests/phase08-acceptance-reporter.ts";'
                "const reporter=new Reporter();"
                "reporter.onBegin({},{});"
                f'reporter.onTestEnd({{title:"unexpected [{criterion}]",parent:{{project:()=>({{name:"real-stack-desktop-chromium"}})}}}},'
                '{status:"failed",attachments:[]});'
                "reporter.onEnd({});"
            )
            environment = {
                **os.environ,
                "PHASE08_ACCEPTANCE_RESULT_DIR": str(evidence),
                "MEALSWAPP_PHASE08_RESULT_FILE": "prebootstrap.json",
                "MEALSWAPP_PHASE08_CRITERIA": criterion,
                "MEALSWAPP_PHASE08_EXPECTED_PROJECTS": ",".join(task281.PROJECTS),
            }
            subprocess.run(
                ["bun", "-e", script],
                cwd=MODULE_PATH.parents[1] / "frontend",
                env=environment,
                check=True,
                capture_output=True,
                text=True,
            )
            task281.Task281Harness.combine_results(evidence)
            output = root / "reports"
            code = task281.finalize_report(
                "producer-failure",
                evidence,
                defer_report=False,
                output_root=output,
            )
            report = json.loads(
                (output / "task281-producer-failure/report.json").read_text()
            )
        self.assertEqual(1, code)
        self.assertEqual(9, len(report["results"]))
        self.assertEqual("FAIL", report["status"])
        self.assertEqual(
            [task281.INFRASTRUCTURE_ROOT],
            [root["id"] for root in report["rootCauses"]],
        )

    def test_malformed_attachment_publishes_synchronized_nonzero_report(self) -> None:
        criterion = task281.PRE_CRITERIA[0]
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            evidence = root / "evidence"
            evidence.mkdir()
            attachment = json.dumps(
                {
                    "criterionIds": [criterion],
                    "requestIds": [],
                    "evidence": [{"type": [], "path": "x"}],
                    "backendEvidence": [],
                }
            )
            script = (
                'import Reporter from "./tests/phase08-acceptance-reporter.ts";'
                "const reporter=new Reporter();"
                "reporter.onBegin({},{});"
                f'reporter.onTestEnd({{title:"malformed [{criterion}]",parent:{{project:()=>'
                '({name:"real-stack-desktop-chromium"})}},'
                f'{{status:"failed",attachments:[{{name:"phase08-acceptance",body:Buffer.from('
                f"{json.dumps(attachment)})}}]}});"
                "reporter.onEnd({});"
            )
            environment = {
                **os.environ,
                "PHASE08_ACCEPTANCE_RESULT_DIR": str(evidence),
                "MEALSWAPP_PHASE08_RESULT_FILE": "prebootstrap.json",
                "MEALSWAPP_PHASE08_CRITERIA": criterion,
                "MEALSWAPP_PHASE08_EXPECTED_PROJECTS": ",".join(task281.PROJECTS),
            }
            subprocess.run(
                ["bun", "-e", script],
                cwd=MODULE_PATH.parents[1] / "frontend",
                env=environment,
                check=True,
                capture_output=True,
                text=True,
            )
            phase_path = evidence / "prebootstrap.json"
            producer = json.loads(phase_path.read_text())
            self.assertEqual([], producer["results"][0]["evidence"])
            self.assertEqual(
                task281.INFRASTRUCTURE_ROOT,
                producer["results"][0]["rootCauseId"],
            )
            producer["results"][0]["evidence"] = [{"type": [], "path": "x"}]
            phase_path.write_text(json.dumps(producer), encoding="utf-8")
            task281.Task281Harness.combine_results(evidence)
            output = root / "reports"
            code = task281.finalize_report(
                "malformed-attachment",
                evidence,
                defer_report=False,
                output_root=output,
            )
            report = json.loads(
                (output / "task281-malformed-attachment/report.json").read_text()
            )
        self.assertNotEqual(0, code)
        self.assertEqual(9, len(report["results"]))
        self.assertEqual("BLOCKED", report["status"])
        self.assertEqual(
            [task281.INFRASTRUCTURE_ROOT],
            [root["id"] for root in report["rootCauses"]],
        )

    def test_backend_evidence_asserts_counts_actor_and_redis(self) -> None:
        fixture_id = "11111111-1111-4111-8111-111111111111"
        cases = (
            ("mutation", ("1", "2", "2"), "PONG"),
            ("audit", ("2", "1", "2"), "PONG"),
            ("actor", ("2", "2", "1"), "PONG"),
            ("redis", ("2", "2", "2"), "NO"),
        )
        for label, counts, redis_state in cases:
            with self.subTest(label=label), tempfile.TemporaryDirectory() as directory:
                harness = object.__new__(task281.Task281Harness)
                harness.target = object()
                harness.database = "mealswapp_e2e_test"
                harness.container = "owned-redis"
                with (
                    mock.patch.object(task281.real_stack, "psql", side_effect=counts),
                    mock.patch.object(
                        task281.real_stack,
                        "run_command",
                        return_value=mock.Mock(stdout=redis_state),
                    ),
                ):
                    with self.assertRaises(task281.EvidenceAssertionError):
                        harness.write_backend_evidence(Path(directory), fixture_id)
                payload = json.loads(
                    (Path(directory) / "backend/admin-authorization.json").read_text()
                )
                self.assertFalse(payload["validated"])

    def test_backend_evidence_accepts_exact_owned_results(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            harness = object.__new__(task281.Task281Harness)
            harness.target = object()
            harness.database = "mealswapp_e2e_test"
            harness.container = "owned-redis"
            with (
                mock.patch.object(task281.real_stack, "psql", side_effect=("2", "2", "2")) as psql,
                mock.patch.object(
                    task281.real_stack,
                    "run_command",
                    return_value=mock.Mock(stdout="PONG"),
                ),
            ):
                harness.write_backend_evidence(
                    Path(directory), "11111111-1111-4111-8111-111111111111"
                )
            self.assertIn("admin_user_id = '11111111-1111-4111-8111-111111111111'::uuid", psql.call_args_list[2].args[1])
            payload = json.loads(
                (Path(directory) / "backend/admin-authorization.json").read_text()
            )
            self.assertTrue(payload["validated"])

    def test_lifecycle_failures_emit_complete_results_and_nonzero_report(self) -> None:
        failures = (
            subprocess.TimeoutExpired(["playwright"], 1),
            RuntimeError("bootstrap failed"),
            task281.EvidenceAssertionError("evidence failed"),
        )
        for error in failures:
            with self.subTest(error=type(error).__name__), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                evidence = root / "evidence"
                output = root / "reports"
                code = task281.finalize_failure(
                    "unit-run",
                    evidence,
                    error,
                    defer_report=False,
                    output_root=output,
                )
                results = json.loads((evidence / "results.json").read_text())["results"]
                report = json.loads(
                    (output / "task281-unit-run/report.json").read_text()
                )
                expected_status = (
                    "FAIL"
                    if isinstance(error, task281.EvidenceAssertionError)
                    else "BLOCKED"
                )
                self.assertEqual(1 if expected_status == "FAIL" else 2, code)
                self.assertEqual(9, len(results))
                self.assertEqual({expected_status}, {result["status"] for result in results})
                self.assertEqual(
                    {task281.INFRASTRUCTURE_ROOT},
                    {result["rootCauseId"] for result in results},
                )
                self.assertEqual(expected_status, report["status"])
                self.assertNotEqual(0, report["exitCode"])
                self.assertEqual(
                    [task281.INFRASTRUCTURE_ROOT],
                    [root["id"] for root in report["rootCauses"]],
                )

    def test_runner_uses_bootstrap_and_read_only_database_evidence(self) -> None:
        source = MODULE_PATH.read_text(encoding="utf-8")
        pre = source.index('"tests/task281-prebootstrap.spec.ts"')
        bootstrap = source.index('"administrator_bootstrapped"')
        post = source.index('"tests/task281-postbootstrap.spec.ts"')
        self.assertLess(pre, bootstrap)
        self.assertLess(bootstrap, post)
        self.assertIn("SELECT count(*) FROM classifications", source)
        self.assertIn("SELECT count(*) FROM admin_audit_entries", source)
        self.assertNotIn("UPDATE users", source)
        self.assertNotIn("DELETE FROM", source)
        self.assertNotIn("TRUNCATE", source)
        post = (MODULE_PATH.parents[1] / "frontend/tests/task281-postbootstrap.spec.ts").read_text()
        self.assertNotIn('"mutation_count=1"', post)
        self.assertNotIn('"audit_count=1"', post)


if __name__ == "__main__":
    unittest.main()
