"""Task 290 disposable-stack acceptance harness contract tests."""

# Implements DESIGN-009 AdminController real API operator acceptance verification.

from __future__ import annotations

import importlib.util
import json
import os
import sys
import unittest
from argparse import Namespace
from pathlib import Path
from unittest import mock

SCRIPTS = Path(__file__).parent
SPEC = importlib.util.spec_from_file_location("run_task290_acceptance", SCRIPTS / "run-task290-acceptance.py")
HARNESS = importlib.util.module_from_spec(SPEC)
assert SPEC.loader
SPEC.loader.exec_module(HARNESS)


class Task290AcceptanceTests(unittest.TestCase):
    """Keep the real-stack runner bounded, disposable, and credential-safe."""

    def args(self) -> Namespace:
        return Namespace(environment="development", base_url="http://127.0.0.1:8080", database_url=None)

    def test_requires_explicit_disposable_stack_marker(self) -> None:
        with mock.patch.dict(os.environ, {}, clear=True):
            with self.assertRaisesRegex(HARNESS.AcceptanceError, "disposable stack"):
                HARNESS.execute(self.args())

    def test_operator_password_is_process_input_not_command_arguments(self) -> None:
        with mock.patch.dict(
            os.environ,
            {
                "MEALSWAPP_TASK290_DISPOSABLE": "1",
                "MEALSWAPP_TASK290_ADMIN_EMAIL": "admin@example.test",
                "MEALSWAPP_TASK290_ADMIN_PASSWORD": "password-secret",
            },
            clear=True,
        ), mock.patch.object(HARNESS.subprocess, "run") as run:
            run.return_value = Namespace(returncode=0, stdout="key\tdisplay_name\tunit\tactive\n", stderr="")
            HARNESS.run_operator(self.args(), ["list"])
            command = run.call_args.args[0]
            self.assertNotIn("admin@example.test", command)
            self.assertNotIn("password-secret", command)
            self.assertEqual(run.call_args.kwargs["input"], "admin@example.test\npassword-secret\n")

    def test_real_acceptance_runs_all_mutations_and_requires_audit_actions_when_requested(self) -> None:
        commands: list[list[str]] = []
        key = "Task290abcdef12345678"
        dry_runs = 0

        def fake_run(_args: Namespace, command: list[str]) -> tuple[int, str, str]:
            nonlocal dry_runs
            commands.append(command)
            if command == ["list"]:
                keys = [previous[index + 1] for previous in commands for index, value in enumerate(previous[:-1]) if value == "--key"]
                updated = any(previous and previous[0] == "update-unit" for previous in commands)
                row = f"{keys[-1]}\tTask 290 acceptance updated\t{'mcg' if updated else 'mg'}\ttrue\n" if keys else ""
                return 0, f"key\tdisplay_name\tunit\tactive\n{row}", ""
            if command[-1:] == ["--dry-run"] and command[0] == "add":
                dry_runs += 1
                if dry_runs == 2:
                    return 2, "", "micronutrient key already exists\n"
                return 0, "validated command=add key=Task290abcdef12345678 dry_run=true\n", ""
            return 0, "succeeded\n", ""

        with mock.patch.dict(
            os.environ,
            {"MEALSWAPP_TASK290_DISPOSABLE": "1"},
            clear=True,
        ), mock.patch.object(HARNESS, "run_operator", side_effect=fake_run), mock.patch.object(
            HARNESS, "read_audit_actions", return_value=HARNESS.REQUIRED_ACTIONS,
        ):
            args = Namespace(environment="development", base_url="http://127.0.0.1:8080", database_url="redacted")
            HARNESS.execute(args)
        self.assertEqual([command[0] for command in commands], [
            "list", "add", "add", "add", "update-display-name", "update-unit", "deactivate", "reactivate", "list", "list",
        ])

    def test_audit_psql_uses_options_before_database_target(self) -> None:
        completed = Namespace(
            stdout="\n".join(f"{action}\tmicronutrient_vocabulary" for action in HARNESS.REQUIRED_ACTIONS) + "\n",
            stderr="", returncode=0,
        )
        with mock.patch.object(HARNESS.subprocess, "run", return_value=completed) as run:
            self.assertEqual(HARNESS.read_audit_actions("postgres://redacted"), HARNESS.REQUIRED_ACTIONS)
        command = run.call_args.args[0]
        self.assertEqual(command[:3], ["psql", "-X", "-Atqc"])
        self.assertEqual(command[-1], "postgres://redacted")
        self.assertIn("cwd=self.source_root / \"backend\"", (Path(HARNESS.__file__).read_text() if HARNESS.__file__ else ""))

    def artifact(self) -> dict[str, object]:
        return {
            "schema": HARNESS.ARTIFACT_SCHEMA,
            "runId": "0123456789abcdef01234567",
            "commands": HARNESS.COMMAND_NAMES,
            "auditRows": [
                {"action": action, "entityType": "micronutrient_vocabulary"}
                for action in sorted(HARNESS.REQUIRED_ACTIONS)
            ],
            "finalState": {
                "key": "Task290abcdef12345678",
                "displayName": "Task 290 acceptance updated",
                "unit": "mcg",
                "active": True,
            },
            "cleanupVerified": True,
        }

    def test_acceptance_artifact_requires_exact_audit_rows_and_final_tuple(self) -> None:
        self.assertEqual(HARNESS.validate_acceptance_artifact(self.artifact()), self.artifact())
        for invalid in (
            {**self.artifact(), "auditRows": self.artifact()["auditRows"][:-1]},
            {**self.artifact(), "auditRows": [{"action": "micronutrient.create", "entityType": "users"}] * 5},
            {**self.artifact(), "finalState": {**self.artifact()["finalState"], "unit": "mg"}},
            {**self.artifact(), "finalState": {**self.artifact()["finalState"], "active": False}},
            {**self.artifact(), "secret": "password-secret"},
        ):
            with self.subTest(invalid=invalid):
                with self.assertRaises(HARNESS.AcceptanceError):
                    HARNESS.validate_acceptance_artifact(invalid)

    def test_audit_projection_rejects_raw_or_incomplete_rows(self) -> None:
        valid = "\n".join(f"{action}\tmicronutrient_vocabulary" for action in HARNESS.REQUIRED_ACTIONS)
        self.assertEqual(len(HARNESS.parse_audit_rows(valid)), 5)
        for invalid in (valid.replace("micronutrient.create", "password-secret"), valid.split("\n", 1)[0]):
            with self.assertRaises(HARNESS.AcceptanceError):
                HARNESS.parse_audit_rows(invalid)

    def test_committed_passing_artifact_validates_and_contains_no_secret_fields(self) -> None:
        artifact_path = SCRIPTS.parent / "docs/implementation/evidence/task-290-acceptance.json"
        artifact = json.loads(artifact_path.read_text(encoding="utf-8"))
        self.assertEqual(HARNESS.validate_acceptance_artifact(artifact), artifact)
        serialized = json.dumps(artifact)
        for forbidden in ("password", "cookie", "csrf", "databaseurl", "postgres://", "@example.test"):
            self.assertNotIn(forbidden, serialized.lower())


if __name__ == "__main__":
    unittest.main()
