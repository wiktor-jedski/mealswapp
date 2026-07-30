#!/usr/bin/env python3
"""Exercise Task 290 through one disposable real API and verify audit writes.

Implements DESIGN-009 AdminController micronutrient vocabulary operator acceptance.
"""

from __future__ import annotations

import argparse
import importlib.util
import json
import os
import re
import secrets
import subprocess
import sys
import types
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OPERATOR = ROOT / "scripts/manage-micronutrient-vocabulary.py"
KEY_PATTERN = re.compile(r"^[A-Z][A-Za-z0-9]{2,119}$")
REQUIRED_ACTIONS = {
    "micronutrient.create",
    "micronutrient.display_name.update",
    "micronutrient.unit.update",
    "micronutrient.deactivate",
    "micronutrient.reactivate",
}
COMMAND_NAMES = ["list", "add-dry-run", "add", "add-existing-dry-run", "update-display-name", "update-unit", "deactivate", "reactivate"]
ARTIFACT_SCHEMA = "mealswapp.task290-acceptance.v1"
RUN_ID_PATTERN = re.compile(r"^[0-9a-f]{24}$")


def load_real_stack() -> object:
    """Load the repository-owned disposable-stack lifecycle helpers."""
    spec = importlib.util.spec_from_file_location("task290_real_stack", ROOT / "scripts/run-real-stack-e2e.py")
    if spec is None or spec.loader is None:
        raise AcceptanceError("real-stack harness is unavailable")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class AcceptanceError(RuntimeError):
    """Represent one safe disposable-stack acceptance failure."""


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    """Require the real API origin and explicit environment from the runner."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--environment", choices=("development", "staging", "production"), required=True)
    target = parser.add_mutually_exclusive_group(required=True)
    target.add_argument("--base-url")
    target.add_argument("--start-disposable", action="store_true")
    parser.add_argument("--api-source-root", type=Path, default=ROOT, help="Task 289 source tree used only with --start-disposable")
    parser.add_argument("--database-url", help="optional disposable-stack URL for read-only audit proof")
    parser.add_argument("--timeout-seconds", type=float, default=120)
    return parser.parse_args(argv)


def run_operator(args: argparse.Namespace, command: list[str]) -> tuple[int, str, str]:
    """Run the actual operator with credentials supplied only through process input."""
    email = os.environ.get("MEALSWAPP_TASK290_ADMIN_EMAIL", "")
    password = os.environ.get("MEALSWAPP_TASK290_ADMIN_PASSWORD", "")
    if not email or not password:
        raise AcceptanceError("acceptance credentials are unavailable")
    operator_args = [
        "--environment", args.environment,
        "--base-url", args.base_url,
    ]
    if args.environment == "production":
        operator_args.append("--confirm-production")
    operator_args.extend(command)
    completed = subprocess.run(
        [sys.executable, str(OPERATOR), *operator_args],
        cwd=ROOT,
        input=f"{email}\n{password}\n",
        capture_output=True,
        text=True,
        timeout=30,
        check=False,
        env={**os.environ, "MEALSWAPP_ENV": args.environment},
    )
    stdout = completed.stdout.replace("Administrator email: ", "")
    stderr = "\n".join(
        line for line in completed.stderr.splitlines()
        if not line.startswith("Warning: Password input may be echoed.")
        and "GetPassWarning" not in line
        and line.strip() != "passwd = fallback_getpass(prompt, stream)"
        and line.strip() != "Administrator password:"
    )
    if stderr:
        stderr += "\n"
    for secret in (email, password):
        if secret in stdout or secret in stderr:
            raise AcceptanceError("operator output contains credential material")
    return completed.returncode, stdout, stderr


def list_keys(args: argparse.Namespace) -> set[str]:
    """Read the operator's deterministic list projection and validate its shape."""
    status, stdout, stderr = run_operator(args, ["list"])
    if status != 0:
        raise AcceptanceError(f"real API list failed status={status} error={stderr.strip() or 'none'}")
    if stderr or not stdout.startswith("key\tdisplay_name\tunit\tactive\n"):
        raise AcceptanceError("real API list failed response")
    keys = set()
    for line in stdout.splitlines()[1:]:
        fields = line.split("\t")
        if len(fields) != 4 or not KEY_PATTERN.fullmatch(fields[0]) or fields[3] not in {"true", "false"}:
            raise AcceptanceError("real API list projection is invalid")
        keys.add(fields[0])
    return keys


def list_state(args: argparse.Namespace) -> dict[str, tuple[str, str, bool]]:
    """Return the validated final projection keyed by canonical vocabulary key."""
    status, stdout, stderr = run_operator(args, ["list"])
    if status != 0:
        raise AcceptanceError(f"real API list failed status={status} error={stderr.strip() or 'none'}")
    if stderr or not stdout.startswith("key\tdisplay_name\tunit\tactive\n"):
        raise AcceptanceError("real API list failed response")
    state: dict[str, tuple[str, str, bool]] = {}
    for line in stdout.splitlines()[1:]:
        fields = line.split("\t")
        if len(fields) != 4 or not KEY_PATTERN.fullmatch(fields[0]) or fields[3] not in {"true", "false"}:
            raise AcceptanceError("real API list projection is invalid")
        state[fields[0]] = (fields[1], fields[2], fields[3] == "true")
    return state


def parse_audit_rows(output: str) -> list[dict[str, str]]:
    """Validate the exact five-row action/entity projection without raw database data."""
    rows: list[dict[str, str]] = []
    for line in output.splitlines():
        fields = line.split("\t")
        if len(fields) != 2 or fields[1] != "micronutrient_vocabulary" or fields[0] not in REQUIRED_ACTIONS:
            raise AcceptanceError("disposable audit projection is invalid")
        rows.append({"action": fields[0], "entityType": fields[1]})
    if len(rows) != len(REQUIRED_ACTIONS) or {row["action"] for row in rows} != REQUIRED_ACTIONS:
        raise AcceptanceError("disposable audit projection is incomplete")
    return rows


def read_audit_rows(database_url: str) -> list[dict[str, str]]:
    """Read only bounded action/entity rows from the disposable database."""
    query = "SELECT action || E'\\t' || entity_type FROM admin_audit_entries WHERE entity_type='micronutrient_vocabulary' ORDER BY created_at, id"
    try:
        result = subprocess.run(
            ["psql", "-X", "-Atqc", query, database_url],
            capture_output=True,
            text=True,
            timeout=10,
            check=True,
        )
    except (OSError, subprocess.SubprocessError) as error:
        raise AcceptanceError("disposable audit read failed") from error
    return parse_audit_rows(result.stdout)


def read_audit_actions(database_url: str) -> set[str]:
    """Return action names for focused tests and external acceptance callers."""
    return {row["action"] for row in read_audit_rows(database_url)}


def validate_acceptance_artifact(value: object) -> dict[str, object]:
    """Validate the closed, secret-free committed acceptance report schema."""
    if not isinstance(value, dict) or set(value) != {"schema", "runId", "commands", "auditRows", "finalState", "cleanupVerified"}:
        raise AcceptanceError("acceptance artifact is invalid")
    if value["schema"] != ARTIFACT_SCHEMA or not isinstance(value["runId"], str) or not RUN_ID_PATTERN.fullmatch(value["runId"]):
        raise AcceptanceError("acceptance artifact is invalid")
    if value["commands"] != COMMAND_NAMES or not isinstance(value["cleanupVerified"], bool):
        raise AcceptanceError("acceptance artifact is invalid")
    rows = value["auditRows"]
    if not isinstance(rows, list) or len(rows) != 5:
        raise AcceptanceError("acceptance artifact is invalid")
    for row in rows:
        if not isinstance(row, dict) or set(row) != {"action", "entityType"} or row["action"] not in REQUIRED_ACTIONS or row["entityType"] != "micronutrient_vocabulary":
            raise AcceptanceError("acceptance artifact is invalid")
    if {row["action"] for row in rows} != REQUIRED_ACTIONS:
        raise AcceptanceError("acceptance artifact is invalid")
    final = value["finalState"]
    if not isinstance(final, dict) or set(final) != {"key", "displayName", "unit", "active"} or not isinstance(final["key"], str) or not KEY_PATTERN.fullmatch(final["key"]):
        raise AcceptanceError("acceptance artifact is invalid")
    if not isinstance(final["displayName"], str) or final["displayName"] != "Task 290 acceptance updated" or final["unit"] != "mcg" or final["active"] is not True:
        raise AcceptanceError("acceptance artifact is invalid")
    return value


def execute(args: argparse.Namespace) -> tuple[str, tuple[str, str, bool]]:
    """Perform every command, dry-run branch, and optional read-only audit proof."""
    if os.environ.get("MEALSWAPP_TASK290_DISPOSABLE") != "1":
        raise AcceptanceError("acceptance requires a disposable stack")
    if args.environment != "development" and not args.base_url.startswith("https://"):
        raise AcceptanceError("disposable API target is invalid")
    key = f"Task290{secrets.token_hex(8)}"
    display_name = "Task 290 acceptance"
    if not KEY_PATTERN.fullmatch(key):
        raise AcceptanceError("acceptance key generation failed")
    list_keys(args)
    if run_operator(args, [
        "add", "--key", key, "--display-name", display_name, "--unit", "mg", "--dry-run",
    ])[0] != 0:
        raise AcceptanceError("new-key dry-run unexpectedly failed")
    status, _, stderr = run_operator(args, [
        "add", "--key", key, "--display-name", display_name, "--unit", "mg",
    ])
    if status != 0 or stderr:
        raise AcceptanceError("real API add failed")
    status, _, stderr = run_operator(args, [
        "add", "--key", key, "--display-name", display_name, "--unit", "mg", "--dry-run",
    ])
    if status == 0 or "already exists" not in stderr:
        raise AcceptanceError("existing-key dry-run behavior is incorrect")
    for command in (
        ["update-display-name", "--key", key, "--display-name", "Task 290 acceptance updated"],
        ["update-unit", "--key", key, "--unit", "mcg"],
        ["deactivate", "--key", key],
        ["reactivate", "--key", key],
    ):
        status, _, stderr = run_operator(args, command)
        if status != 0 or stderr:
            raise AcceptanceError("real API mutation failed")
    if key not in list_keys(args):
        raise AcceptanceError("real API final vocabulary state is missing")
    final_state = list_state(args).get(key)
    if final_state != ("Task 290 acceptance updated", "mcg", True):
        raise AcceptanceError("real API final vocabulary state is incorrect")
    if args.database_url:
        if not REQUIRED_ACTIONS.issubset(read_audit_actions(args.database_url)):
            raise AcceptanceError("real API audit entries are incomplete")
    return key, final_state


class Task290Harness:
    """Own one Task 289 API, database, Redis instance, and cleanup lifecycle."""

    def __init__(self, real_stack: object, timeout: float, source_root: Path) -> None:
        self.real_stack = real_stack
        self.timeout = timeout
        self.source_root = source_root.resolve()
        if not (self.source_root / "backend" / "go.mod").is_file():
            raise AcceptanceError("Task 289 API source tree is invalid")
        self.run_id = real_stack.secrets.token_hex(12)
        self.target = real_stack.PostgresTarget.parse(os.environ.get(
            "MEALSWAPP_E2E_POSTGRES_ADMIN_URL",
            "postgres://mealswapp:mealswapp@127.0.0.1:5432/postgres",
        ))

    def execute(self) -> None:
        """Create, migrate, run, inspect, and leave teardown to the parent harness."""
        harness = self.real_stack.Harness(self.target, timeout=self.timeout)
        harness.run_id = self.run_id
        harness.database = self.real_stack.database_name(self.run_id)
        harness.comment = self.real_stack.ownership_comment(self.run_id, harness.created_at)
        harness.artifacts = self.real_stack.ARTIFACT_ROOT / self.run_id
        harness.state_path = harness.artifacts / "state.json"

        def owned_execute(_harness: object) -> None:
            self.real_stack.create_database(harness.target, harness.database, harness.comment)
            redis_port = harness.start_redis()
            api_reservation = self.real_stack.reserve_ports(1)[0]
            frontend_port = self.real_stack.reserve_port()
            database_url = harness.target.database_url(harness.database)
            env = harness.application_environment(database_url, f"redis://127.0.0.1:{redis_port}/0", api_reservation.port, frontend_port)
            go_tmp = harness.raw_dir / "go-tmp" if harness.raw_dir is not None else None
            if go_tmp is None:
                raise AcceptanceError("run-owned build workspace is unavailable")
            go_tmp.mkdir(mode=0o700)
            env["GOTMPDIR"] = str(go_tmp)
            self.real_stack.run_command(["go", "run", "./cmd/migrate", "up"], cwd=self.source_root / "backend", env=env, timeout=self.timeout)
            assert harness.raw_dir is not None
            api_binary = harness.raw_dir / "mealswapp-api"
            bootstrap_binary = harness.raw_dir / "admin-bootstrap"
            for output, command in ((api_binary, "./cmd/api"), (bootstrap_binary, "./cmd/admin-bootstrap")):
                self.real_stack.run_command(["go", "build", "-o", str(output), command], cwd=self.source_root / "backend", env=env, timeout=self.timeout)
            api_port = api_reservation.port
            api = harness.start_process("api", [str(api_binary)], self.source_root / "backend", env, "api.raw.log", api_reservation)
            self.real_stack.wait_http(f"http://127.0.0.1:{api_port}/health", api, self.timeout)
            fixture, request_ids = self.real_stack.register_fixture(f"http://127.0.0.1:{api_port}", self.run_id)
            harness.request_ids.extend(request_ids)
            bootstrap = self.real_stack.run_command(
                [str(bootstrap_binary), "--environment", "development", "--email", fixture["email"]],
                cwd=self.source_root / "backend", env=env, timeout=self.timeout,
            )
            if bootstrap.returncode != 0:
                raise AcceptanceError("administrator bootstrap failed")
            previous = os.environ.copy()
            os.environ.update({
                "MEALSWAPP_TASK290_DISPOSABLE": "1",
                "MEALSWAPP_TASK290_ADMIN_EMAIL": fixture["email"],
                "MEALSWAPP_TASK290_ADMIN_PASSWORD": fixture["password"],
            })
            try:
                try:
                    key, final_state = execute(argparse.Namespace(
                        environment="development",
                        base_url=f"http://127.0.0.1:{api_port}",
                        database_url=database_url,
                        start_disposable=False,
                    ))
                except BaseException as error:
                    harness.events.append(f"task290_command_failure_{type(error).__name__.lower()}")
                    raise
            finally:
                os.environ.clear()
                os.environ.update(previous)
            rows = self.real_stack.psql(
                harness.target,
                "SELECT action || E'\\t' || entity_type FROM admin_audit_entries WHERE entity_type='micronutrient_vocabulary' ORDER BY created_at, id",
                database=harness.database,
            )
            audit_rows = parse_audit_rows(rows)
            artifact = {
                "schema": ARTIFACT_SCHEMA,
                "runId": self.run_id,
                "commands": COMMAND_NAMES,
                "auditRows": audit_rows,
                "finalState": {"key": key, "displayName": final_state[0], "unit": final_state[1], "active": final_state[2]},
                "cleanupVerified": False,
            }
            validate_acceptance_artifact(artifact)
            (harness.artifacts / "task290-acceptance.json").write_text(json.dumps(artifact, indent=2) + "\n", encoding="utf-8")
            harness.events.extend(["task290_api_started", "task290_migrations_applied", "task290_commands_passed", "task290_audit_verified", "task290_final_state_verified"])

        original_execute = harness.execute
        harness.execute = types.MethodType(owned_execute, harness)
        try:
            harness.run()
            state = json.loads(harness.state_path.read_text(encoding="utf-8"))
            if state.get("status") not in {"cleaned", "cleaned_diagnostics_failed"}:
                raise AcceptanceError("disposable cleanup was not recorded")
            report_path = harness.artifacts / "task290-acceptance.json"
            report = json.loads(report_path.read_text(encoding="utf-8"))
            report["cleanupVerified"] = True
            validate_acceptance_artifact(report)
            report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        finally:
            harness.execute = original_execute


def main(argv: list[str] | None = None) -> int:
    """Run the disposable acceptance and emit only fixed safe categories."""
    try:
        args = parse_args(argv)
        if args.start_disposable:
            if args.environment != "development":
                raise AcceptanceError("disposable API requires development environment")
            Task290Harness(load_real_stack(), args.timeout_seconds, args.api_source_root).execute()
        else:
            execute(args)
        print("task290_real_api=passed audit=verified_or_not_requested")
        return 0
    except SystemExit:
        raise
    except AcceptanceError as error:
        print(str(error), file=sys.stderr)
        return 2
    except Exception as error:
        detail = f" detail={str(error)}" if isinstance(error, TypeError) else ""
        print(f"task290_real_api=failed error={type(error).__name__}{detail}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
