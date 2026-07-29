#!/usr/bin/env python3
"""Run Task 282 controlled-provider curation acceptance on a disposable real stack."""

# Implements DESIGN-009 DataImporter and DESIGN-012 provider real-stack acceptance.

from __future__ import annotations

import argparse
import importlib.util
import json
import os
import secrets
import subprocess
import sys
from pathlib import Path
from typing import Sequence

ROOT = Path(__file__).resolve().parents[1]
HARNESS_PATH = Path(__file__).with_name("run-real-stack-e2e.py")
SPEC = importlib.util.spec_from_file_location("task282_real_stack", HARNESS_PATH)
if SPEC is None or SPEC.loader is None:
    raise RuntimeError("real-stack harness could not be loaded")
real_stack = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = real_stack
SPEC.loader.exec_module(real_stack)

REQUIREMENT_CRITERIA = {
    "SW-REQ-033": tuple(f"P08-SWR033-STEP-{number:02d}" for number in range(1, 6)),
    "SW-REQ-055": (
        *(f"P08-SWR055-STEP-{number:02d}" for number in range(1, 8)),
        *(f"P08-SWR055-ACCEPT-{number:02d}" for number in range(1, 8)),
    ),
    "SW-REQ-090": (
        *(f"P08-SWR090-STEP-{number:02d}" for number in range(1, 5)),
        "P08-SWR090-ACCEPT-01",
    ),
}
CRITERIA = tuple(
    criterion
    for requirement in ("SW-REQ-033", "SW-REQ-055", "SW-REQ-090")
    for criterion in REQUIREMENT_CRITERIA[requirement]
)
PROJECTS = ("real-stack-desktop-chromium",)
INFRASTRUCTURE_ROOT = "ROOT-T282-ACCEPTANCE-INFRASTRUCTURE"
SYNCHRONIZED_ROOTS = (
    "ROOT-T282-OFF-METADATA",
    "ROOT-T282-USDA-OPTIONAL-PORTION",
    "ROOT-T282-VOCABULARY-DISABLE",
    INFRASTRUCTURE_ROOT,
)
REPORT_ROOT = ROOT / "logs/phase08-acceptance"
OWNERLESS_COUNT_SQL = """WITH catalog_identity AS (
    SELECT name, NULL::uuid AS owner_id FROM food_items
    UNION ALL
    SELECT name, owner_id FROM custom_food_items
)
SELECT count(*) FROM catalog_identity
WHERE name LIKE 'Acceptance curated %' AND owner_id IS NULL"""


def persistence_evidence_is_valid(values: dict[str, int | bool]) -> bool:
    """Require one global curation effect; a same-name private row cannot satisfy it."""
    return (
        values["foodItemCount"] == 1
        and values["curatedImportCount"] == 1
        and values["auditCount"] == 1
        and values["ownerlessCount"] == 1
        and values["editedCount"] == 1
        and values["redisReachable"] is True
    )


class Task282Harness(real_stack.Harness):
    """Owns one provider fixture, API, browser, database, Redis, and report producer."""

    fixture_port: int | None = None

    def application_environment(
        self, database_url: str, redis_url: str, api_port: int, frontend_port: int
    ) -> dict[str, str]:
        environment = super().application_environment(
            database_url, redis_url, api_port, frontend_port
        )
        if self.fixture_port is not None:
            environment.update(
                {
                    "MEALSWAPP_USDA_API_KEY": "task282-controlled-key",
                    "MEALSWAPP_USDA_ENDPOINT": f"http://127.0.0.1:{self.fixture_port}/usda",
                    "MEALSWAPP_OPENFOODFACTS_ENDPOINT": f"http://127.0.0.1:{self.fixture_port}/openfoodfacts",
                    "MEALSWAPP_EXTERNAL_PROVIDER_TIMEOUT": "300ms",
                    "MEALSWAPP_TASK282_DROP_IMPORT_RESPONSE_ONCE": "1",
                }
            )
        return environment

    def execute(self) -> None:
        reservations: list[real_stack.PortReservation] = []
        try:
            real_stack.create_database(self.target, self.database, self.comment)
            self.events.append("database_created")
            redis_port = self.start_redis()
            reservations = real_stack.reserve_ports(3)
            fixture_reservation, api_reservation, frontend_reservation = reservations
            self.fixture_port = fixture_reservation.port
            database_url = self.target.database_url(self.database)
            redis_url = f"redis://127.0.0.1:{redis_port}/0"
            env = self.application_environment(
                database_url, redis_url, api_reservation.port, frontend_reservation.port
            )
            real_stack.run_command(
                ["go", "run", "./cmd/migrate", "up"],
                cwd=ROOT / "backend",
                env=env,
                timeout=self.timeout,
            )
            self.events.append("migrations_applied")
            assert self.raw_dir is not None
            api_binary = self.raw_dir / "mealswapp-api"
            bootstrap_binary = self.raw_dir / "admin-bootstrap"
            real_stack.run_command(
                ["go", "build", "-o", str(api_binary), "./cmd/api"],
                cwd=ROOT / "backend",
                env=env,
                timeout=self.timeout,
            )
            real_stack.run_command(
                ["go", "build", "-o", str(bootstrap_binary), "./cmd/admin-bootstrap"],
                cwd=ROOT / "backend",
                env=env,
                timeout=self.timeout,
            )
            fixture = self.start_process(
                "task282-provider-fixture",
                [sys.executable, str(Path(__file__).with_name("task282_provider_fixture.py")), "--port", str(fixture_reservation.port)],
                ROOT,
                env,
                "provider-fixture.raw.log",
                fixture_reservation,
            )
            real_stack.wait_http(
                f"http://127.0.0.1:{reservations[0].port}/health", fixture, self.timeout
            )
            self.events.append("provider_fixture_ready")
            _, api_port, frontend_port, _api, frontend = self.start_application_stack(
                api_binary, database_url, redis_url, [api_reservation, frontend_reservation]
            )
            self.events.append("application_ready")
            user, request_ids = real_stack.register_fixture(
                f"http://127.0.0.1:{api_port}", self.run_id
            )
            self.request_ids.extend(request_ids)
            bootstrap = real_stack.run_command(
                [str(bootstrap_binary), "--environment", "development"],
                cwd=ROOT / "backend",
                env=env,
                input_text=user["email"] + "\n",
                timeout=self.timeout,
            )
            if f"user_id={user['user_id']}" not in bootstrap.stdout:
                raise RuntimeError("administrator bootstrap returned an unexpected target")
            real_stack.psql(
                self.target,
                "INSERT INTO entitlements "
                "(user_id,tier,status,search_limit_per_24h,allowed_modes,expires_at) "
                f"VALUES ('{user['user_id']}'::uuid,'trial','active',100,"
                "ARRAY['catalog','substitution','daily_diet','daily_diet_alternative'],now()+interval '1 day')",
                database=self.database,
            )
            evidence = self.artifacts / "acceptance"
            evidence.mkdir()
            capability_nonce = secrets.token_hex(24)
            capability_path = self.raw_dir / "task282-harness-capability.json"
            capability_path.write_text(
                json.dumps(
                    {
                        "schema": "mealswapp.task282-harness-capability.v1",
                        "runId": self.run_id,
                        "nonce": capability_nonce,
                        "baseURL": f"http://127.0.0.1:{frontend_port}",
                        "evidenceRoot": str(evidence),
                        "frontendProcess": {
                            "pid": frontend.pid,
                            "startToken": real_stack.validate_process_start_token(
                                real_stack.process_start_token(frontend.pid)
                            ),
                        },
                    },
                    sort_keys=True,
                )
                + "\n",
                encoding="utf-8",
            )
            capability_path.chmod(0o600)
            playwright_env = {
                **env,
                "MEALSWAPP_TASK282_REAL_E2E": "1",
                "MEALSWAPP_REAL_STACK_MANAGED": "1",
                "MEALSWAPP_REAL_STACK_BASE_URL": f"http://127.0.0.1:{frontend_port}",
                "MEALSWAPP_TASK282_CAPABILITY_FILE": str(capability_path),
                "MEALSWAPP_TASK282_CAPABILITY_NONCE": capability_nonce,
                "MEALSWAPP_E2E_EMAIL": user["email"],
                "MEALSWAPP_E2E_PASSWORD": user["password"],
                "PHASE08_ACCEPTANCE_RESULT_DIR": str(evidence),
                "MEALSWAPP_PHASE08_RESULT_FILE": "browser.json",
                "MEALSWAPP_PHASE08_CRITERIA": ",".join(CRITERIA),
                "MEALSWAPP_PHASE08_EXPECTED_PROJECTS": ",".join(PROJECTS),
                "MEALSWAPP_PHASE08_INFRASTRUCTURE_ROOT": INFRASTRUCTURE_ROOT,
                "MEALSWAPP_PHASE08_SYNCHRONIZED_ROOTS": ",".join(SYNCHRONIZED_ROOTS),
                "PLAYWRIGHT_OUTPUT_DIR": str(self.raw_dir / "playwright"),
            }
            try:
                real_stack.run_command(
                    [
                        "bunx", "playwright", "test", "-c",
                        "playwright.real-stack.config.ts",
                        "tests/task282-external-curation.spec.ts",
                    ],
                    cwd=ROOT / "frontend",
                    env=playwright_env,
                    timeout=self.timeout,
                )
            except subprocess.CalledProcessError as error:
                self.events.append("browser_product_nonpass")
                text = "\n".join(
                    value for value in (error.stdout, error.stderr) if isinstance(value, str)
                )
                lines = [
                    line[:400]
                    for line in text.splitlines()
                    if any(marker in line for marker in ("task282-", "Error:", "Expected:", "Received:", "TimeoutError:"))
                ][:80]
                safe = "\n".join(lines)
                for secret in (user["email"], user["password"], self.run_id):
                    safe = safe.replace(secret, "[redacted]")
                (evidence / "browser-diagnostics.txt").write_text(safe + "\n", encoding="utf-8")
            self.write_backend_evidence(evidence)
            self.combine_results(evidence)
            self.events.append("task282_results_finalized")
        finally:
            for reservation in reservations:
                reservation.release()

    def write_backend_evidence(self, evidence: Path) -> None:
        queries = {
            "foodItemCount": "SELECT count(*) FROM food_items WHERE name LIKE 'Acceptance curated %'",
            "curatedImportCount": "SELECT count(*) FROM curated_imports ci JOIN food_items fi ON fi.id=ci.food_item_id WHERE fi.name LIKE 'Acceptance curated %'",
            "auditCount": "SELECT count(*) FROM admin_audit_entries ae JOIN food_items fi ON fi.id=ae.entity_id WHERE fi.name LIKE 'Acceptance curated %' AND ae.action='import_food'",
            "ownerlessCount": OWNERLESS_COUNT_SQL,
            "editedCount": "SELECT count(*) FROM food_items WHERE name LIKE 'Acceptance curated %' AND protein_per_100=12.5 AND carbohydrates_per_100=18.25 AND fat_per_100=3.75",
        }
        values = {
            name: int(real_stack.psql(self.target, sql, database=self.database))
            for name, sql in queries.items()
        }
        redis_state = real_stack.run_command(
            ["docker", "exec", self.container, "redis-cli", "ping"], timeout=10
        ).stdout.strip()
        values["redisReachable"] = redis_state == "PONG"
        values["validated"] = persistence_evidence_is_valid(values)
        backend = evidence / "backend"
        backend.mkdir()
        (backend / "curation.json").write_text(
            json.dumps(values, indent=2, sort_keys=True) + "\n", encoding="utf-8"
        )
        if not values["validated"]:
            raise RuntimeError("Task 282 persistence evidence did not match exact-once invariants")

    @staticmethod
    def combine_results(evidence: Path, failure_status: str | None = None) -> None:
        try:
            values = json.loads((evidence / "browser.json").read_text(encoding="utf-8"))["results"]
        except (OSError, UnicodeError, json.JSONDecodeError, KeyError, TypeError):
            values = []
        by_id: dict[str, dict[str, object]] = {}
        duplicate = False
        if isinstance(values, list):
            for value in values:
                if not isinstance(value, dict) or not isinstance(value.get("criterionId"), str):
                    continue
                criterion = value["criterionId"]
                if criterion in by_id:
                    duplicate = True
                by_id[criterion] = value
        if duplicate:
            by_id = {}
        results: list[dict[str, object]] = []
        for criterion in CRITERIA:
            source = by_id.get(criterion, {})
            status = source.get("status") if isinstance(source, dict) else None
            if status not in {"PASS", "FAIL", "BLOCKED"}:
                status = "BLOCKED"
                source = {}
            if failure_status is not None:
                status = failure_status
            result: dict[str, object] = {
                "criterionId": criterion,
                "status": status,
                "requestIds": source.get("requestIds", []),
                "evidence": source.get("evidence", []),
                "backendEvidence": source.get("backendEvidence", []),
            }
            if (evidence / "backend/curation.json").is_file():
                result["evidence"] = [
                    *result["evidence"],  # type: ignore[misc]
                    {"type": "backend", "path": "backend/curation.json"},
                ]
            if status != "PASS":
                root = source.get("rootCauseId") if isinstance(source, dict) else None
                result["rootCauseId"] = root if root in SYNCHRONIZED_ROOTS else INFRASTRUCTURE_ROOT
            results.append(result)
        (evidence / "results.json").write_text(
            json.dumps({"results": results}, indent=2, sort_keys=True) + "\n",
            encoding="utf-8",
        )
        for requirement, criteria in REQUIREMENT_CRITERIA.items():
            selected = [result for result in results if result["criterionId"] in criteria]
            (evidence / f"{requirement}.json").write_text(
                json.dumps({"results": selected}, indent=2, sort_keys=True) + "\n",
                encoding="utf-8",
            )

    def export_diagnostics(self, result: str) -> None:
        """Keep run-level diagnostics aligned with a caught browser product failure."""
        if result == "passed" and "browser_product_nonpass" in self.events:
            result = "product_nonpass"
        super().export_diagnostics(result)


def finalize_reports(run_id: str, evidence: Path, defer_report: bool) -> int:
    statuses = {
        result["status"]
        for result in json.loads((evidence / "results.json").read_text(encoding="utf-8"))["results"]
    }
    expected = 1 if "FAIL" in statuses else 2 if "BLOCKED" in statuses else 0
    if defer_report:
        print(f"Task 282 result retained: {run_id}")
        return expected
    codes: list[int] = []
    for requirement in REQUIREMENT_CRITERIA:
        command = [
            sys.executable,
            str(Path(__file__).with_name("phase08_acceptance.py")),
            "report",
            "--run-id", f"task282-{run_id}-{requirement.lower()}",
            "--result", str(evidence / f"{requirement}.json"),
            "--evidence-root", str(evidence),
            "--requirement", requirement,
        ]
        result = subprocess.run(command, cwd=ROOT, check=False)
        codes.append(result.returncode)
    return 1 if 1 in codes else 2 if 2 in codes else 0


def parse_args(argv: Sequence[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--timeout-seconds", type=float, default=240)
    parser.add_argument("--defer-report", action="store_true")
    return parser.parse_args(argv)


def main(argv: Sequence[str] | None = None) -> int:
    args = parse_args(sys.argv[1:] if argv is None else argv)
    run_id = secrets.token_hex(12)
    harness: Task282Harness | None = None
    try:
        real_stack.validate_environment(os.environ.get("MEALSWAPP_ENV", "development"))
        target = real_stack.PostgresTarget.parse(
            os.environ.get(
                "MEALSWAPP_E2E_POSTGRES_ADMIN_URL",
                "postgres://mealswapp:mealswapp@127.0.0.1:5432/postgres",
            )
        )
        harness = Task282Harness(target, timeout=args.timeout_seconds)
        run_id = harness.run_id
        harness.run()
        return finalize_reports(run_id, harness.artifacts / "acceptance", args.defer_report)
    except BaseException as error:
        evidence = (
            harness.artifacts / "acceptance"
            if harness is not None
            else real_stack.ARTIFACT_ROOT / run_id / "acceptance"
        )
        evidence.mkdir(parents=True, exist_ok=True)
        Task282Harness.combine_results(evidence, failure_status="BLOCKED")
        print(f"Task 282 acceptance failed safely: {type(error).__name__}", file=sys.stderr)
        return finalize_reports(run_id, evidence, args.defer_report) or 2


if __name__ == "__main__":
    raise SystemExit(main())
