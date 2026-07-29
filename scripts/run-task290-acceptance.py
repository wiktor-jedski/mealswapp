#!/usr/bin/env python3
"""Exercise Task 290 through one disposable real API and verify audit writes.

Implements DESIGN-009 AdminController micronutrient vocabulary operator acceptance.
"""

from __future__ import annotations

import argparse
import os
import re
import secrets
import subprocess
import sys
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


class AcceptanceError(RuntimeError):
    """Represent one safe disposable-stack acceptance failure."""


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    """Require the real API origin and explicit environment from the runner."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--environment", choices=("development", "staging", "production"), required=True)
    parser.add_argument("--base-url", required=True)
    parser.add_argument("--database-url", help="optional disposable-stack URL for read-only audit proof")
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
    for secret in (email, password):
        if secret in completed.stdout or secret in completed.stderr:
            raise AcceptanceError("operator output contains credential material")
    return completed.returncode, completed.stdout, completed.stderr


def list_keys(args: argparse.Namespace) -> set[str]:
    """Read the operator's deterministic list projection and validate its shape."""
    status, stdout, stderr = run_operator(args, ["list"])
    if status != 0 or stderr or not stdout.startswith("key\tdisplay_name\tunit\tactive\n"):
        raise AcceptanceError("real API list failed")
    keys = set()
    for line in stdout.splitlines()[1:]:
        fields = line.split("\t")
        if len(fields) != 4 or not KEY_PATTERN.fullmatch(fields[0]) or fields[3] not in {"true", "false"}:
            raise AcceptanceError("real API list projection is invalid")
        keys.add(fields[0])
    return keys


def read_audit_actions(database_url: str) -> set[str]:
    """Read only bounded action names from the disposable database, never from the operator."""
    query = "SELECT DISTINCT action FROM admin_audit_entries WHERE entity_type='micronutrient_vocabulary'"
    try:
        result = subprocess.run(
            ["psql", database_url, "-Atqc", query],
            capture_output=True,
            text=True,
            timeout=10,
            check=True,
        )
    except (OSError, subprocess.SubprocessError) as error:
        raise AcceptanceError("disposable audit read failed") from error
    return {line for line in result.stdout.splitlines() if line}


def execute(args: argparse.Namespace) -> None:
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
    if args.database_url:
        if not REQUIRED_ACTIONS.issubset(read_audit_actions(args.database_url)):
            raise AcceptanceError("real API audit entries are incomplete")


def main(argv: list[str] | None = None) -> int:
    """Run the disposable acceptance and emit only fixed safe categories."""
    try:
        execute(parse_args(argv))
        print("task290_real_api=passed audit=verified_or_not_requested")
        return 0
    except SystemExit:
        raise
    except AcceptanceError as error:
        print(str(error), file=sys.stderr)
        return 2
    except Exception:
        print("task290_real_api=failed", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
