#!/usr/bin/env python3
"""Run isolated real-stack SW-REQ-054 administrative authorization acceptance."""

# Implements DESIGN-009 AdminController real-stack administrative authorization acceptance.

from __future__ import annotations

import argparse
import importlib.util
import json
import os
import re
import secrets
import subprocess
import sys
import uuid
from pathlib import Path
from typing import Sequence

ROOT = Path(__file__).resolve().parents[1]
HARNESS_PATH = Path(__file__).with_name("run-real-stack-e2e.py")
SPEC = importlib.util.spec_from_file_location("real_stack_e2e", HARNESS_PATH)
if SPEC is None or SPEC.loader is None:
    raise RuntimeError("real-stack harness could not be loaded")
real_stack = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = real_stack
SPEC.loader.exec_module(real_stack)

PRE_CRITERIA = (
    "P08-SWR054-STEP-01",
    "P08-SWR054-STEP-02",
    "P08-SWR054-ACCEPT-01",
    "P08-SWR054-ACCEPT-02",
)
POST_CRITERIA = (
    "P08-SWR054-STEP-03",
    "P08-SWR054-STEP-04",
    "P08-SWR054-ACCEPT-03",
    "P08-SWR054-ACCEPT-04",
    "P08-SWR054-ACCEPT-05",
)
PROJECTS = ("real-stack-desktop-chromium", "real-stack-mobile-chromium")
RESOLVED_MOBILE_NAVIGATION_ROOT = "ROOT-T281-MOBILE-SIDEBAR"
INFRASTRUCTURE_ROOT = "ROOT-T281-ACCEPTANCE-INFRASTRUCTURE"
SYNCHRONIZED_ROOTS = {INFRASTRUCTURE_ROOT, "ROOT-T281-STALE-REFRESH"}
PHASE08_REPORT_ROOT = ROOT / "logs/phase08-acceptance"
REQUEST_ID_PATTERN = re.compile(r"^[a-zA-Z0-9][a-zA-Z0-9._:-]{0,127}$")
BACKEND_SUMMARY_PATTERN = re.compile(
    r"^[a-z][a-z0-9_]{0,63}(?:=[a-zA-Z0-9][a-zA-Z0-9._:-]{0,127})?$"
)
BACKEND_SUMMARY_KEYS = {
    "http_status",
    "mutation_count",
    "audit_count",
    "row_count",
    "owner_state",
    "cache_generation",
    "worker_state",
    "provider_state",
    "log_sink_state",
    "metric_basis",
    "export_record_count",
    "request_correlation",
    "rollback_state",
}


class EvidenceAssertionError(RuntimeError):
    """Raised when isolated backend evidence contradicts browser acceptance."""


class ReportFinalizationError(RuntimeError):
    """Raised when Task 280 does not publish the requested synchronized report."""


class Task281Harness(real_stack.Harness):
    """Runs browser/API evidence on both sides of administrator bootstrap."""

    def execute(self) -> None:
        reservations: list[real_stack.PortReservation] = []
        try:
            real_stack.create_database(self.target, self.database, self.comment)
            self.events.append("database_created")
            redis_port = self.start_redis()
            reservations = real_stack.reserve_ports(2)
            api_port, frontend_port = reservations[0].port, reservations[1].port
            database_url = self.target.database_url(self.database)
            redis_url = f"redis://127.0.0.1:{redis_port}/0"
            env = self.application_environment(database_url, redis_url, api_port, frontend_port)
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
            env, api_port, frontend_port, _api, frontend = self.start_application_stack(
                api_binary, database_url, redis_url, reservations
            )
            self.events.append("api_ready")
            fixture, request_ids = real_stack.register_fixture(
                f"http://127.0.0.1:{api_port}", self.run_id
            )
            self.request_ids.extend(request_ids)
            evidence = self.artifacts / "acceptance"
            evidence.mkdir()
            stale_state = self.raw_dir / "stale-state"
            stale_state.mkdir()
            capability_nonce = secrets.token_hex(24)
            capability_path = self.raw_dir / "task281-harness-capability.json"
            capability_path.write_text(
                json.dumps(
                    {
                        "schema": "mealswapp.task281-harness-capability.v1",
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
                "MEALSWAPP_TASK281_REAL_E2E": "1",
                "MEALSWAPP_REAL_STACK_MANAGED": "1",
                "MEALSWAPP_REAL_STACK_BASE_URL": f"http://127.0.0.1:{frontend_port}",
                "MEALSWAPP_TASK281_CAPABILITY_FILE": str(capability_path),
                "MEALSWAPP_TASK281_CAPABILITY_NONCE": capability_nonce,
                "MEALSWAPP_E2E_EMAIL": fixture["email"],
                "MEALSWAPP_E2E_PASSWORD": fixture["password"],
                "MEALSWAPP_E2E_STALE_STATE_DIR": str(stale_state),
                "PHASE08_ACCEPTANCE_RESULT_DIR": str(evidence),
                "MEALSWAPP_PHASE08_EXPECTED_PROJECTS": ",".join(PROJECTS),
                "PLAYWRIGHT_OUTPUT_DIR": str(self.raw_dir / "playwright"),
            }
            self.run_playwright_phase(
                "prebootstrap",
                "tests/task281-prebootstrap.spec.ts",
                PRE_CRITERIA,
                playwright_env,
            )
            bootstrap = real_stack.run_command(
                [str(bootstrap_binary), "--environment", "development"],
                cwd=ROOT / "backend",
                env=env,
                input_text=fixture["email"] + "\n",
                timeout=self.timeout,
            )
            if f"user_id={fixture['user_id']}" not in bootstrap.stdout:
                raise RuntimeError("administrator bootstrap returned an unexpected target")
            self.events.append("administrator_bootstrapped")
            self.run_playwright_phase(
                "postbootstrap",
                "tests/task281-postbootstrap.spec.ts",
                POST_CRITERIA,
                playwright_env,
            )
            self.write_backend_evidence(evidence, fixture["user_id"])
            self.combine_results(evidence)
            self.events.append("task281_results_finalized")
        finally:
            for reservation in reservations:
                reservation.release()

    def run_playwright_phase(
        self,
        phase: str,
        test_path: str,
        criteria: Sequence[str],
        environment: dict[str, str],
    ) -> None:
        phase_environment = {
            **environment,
            "MEALSWAPP_PHASE08_RESULT_FILE": f"{phase}.json",
            "MEALSWAPP_PHASE08_CRITERIA": ",".join(criteria),
        }
        try:
            real_stack.run_command(
                [
                    "bunx",
                    "playwright",
                    "test",
                    "-c",
                    "playwright.real-stack.config.ts",
                    test_path,
                ],
                cwd=ROOT / "frontend",
                env=phase_environment,
                timeout=self.timeout,
            )
            self.events.append(f"{phase}_passed")
        except subprocess.CalledProcessError:
            self.events.append(f"{phase}_failed")
            print(safe_playwright_diagnostics(phase, self.run_id, environment, sys.exc_info()[1]), file=sys.stderr)

    def write_backend_evidence(self, evidence: Path, fixture_user_id: str) -> None:
        expected_count = len(PROJECTS)
        actor_id = str(uuid.UUID(fixture_user_id))
        mutation_count = int(
            real_stack.psql(
                self.target,
                "SELECT count(*) FROM classifications WHERE name LIKE 'Acceptance %'",
                database=self.database,
            )
        )
        audit_count = int(
            real_stack.psql(
                self.target,
                "SELECT count(*) FROM admin_audit_entries WHERE action = 'classification.create'",
                database=self.database,
            )
        )
        actor_count = int(
            real_stack.psql(
                self.target,
                "SELECT count(*) FROM admin_audit_entries "
                "WHERE action = 'classification.create' AND actor_kind = 'administrator' "
                f"AND admin_user_id = '{actor_id}'::uuid",
                database=self.database,
            )
        )
        redis_state = real_stack.run_command(
            [
                "docker",
                "exec",
                self.container,
                "redis-cli",
                "-p",
                "6379",
                "ping",
            ],
            timeout=10,
        ).stdout.strip()
        backend = evidence / "backend"
        backend.mkdir()
        payload = {
            "classificationMutationCount": mutation_count,
            "classificationAuditCount": audit_count,
            "administratorActorCount": actor_count,
            "expectedMutationCount": expected_count,
            "redisState": "reachable" if redis_state == "PONG" else "unavailable",
            "fixtureScope": "owned-isolated-test-database",
            "validated": (
                mutation_count == expected_count
                and audit_count == expected_count
                and actor_count == expected_count
                and redis_state == "PONG"
            ),
        }
        (backend / "admin-authorization.json").write_text(
            json.dumps(payload, indent=2, sort_keys=True) + "\n",
            encoding="utf-8",
        )
        if not payload["validated"]:
            raise EvidenceAssertionError(
                "backend acceptance evidence did not match expected isolated mutations"
            )

    @staticmethod
    def combine_results(
        evidence: Path,
        *,
        failure_status: str | None = None,
    ) -> None:
        results: list[dict[str, object]] = []
        for phase, criteria in (("prebootstrap", PRE_CRITERIA), ("postbootstrap", POST_CRITERIA)):
            path = evidence / f"{phase}.json"
            try:
                payload = json.loads(path.read_text(encoding="utf-8"))
                values = payload["results"]
            except (OSError, UnicodeError, json.JSONDecodeError, KeyError, TypeError):
                values = []
            by_criterion = {
                criterion: [
                    item
                    for item in values
                    if isinstance(item, dict) and item.get("criterionId") == criterion
                ]
                for criterion in criteria
            } if isinstance(values, list) else {}
            for criterion in criteria:
                matches = by_criterion.get(criterion, [])
                result = matches[0] if len(matches) == 1 else {}
                status = result.get("status")
                request_ids = result.get("requestIds", [])
                linked_evidence = result.get("evidence", [])
                backend_evidence = result.get("backendEvidence", [])
                if (
                    not isinstance(status, str)
                    or status not in {"PASS", "FAIL", "BLOCKED"}
                    or not _safe_request_ids(request_ids)
                    or not _safe_evidence(linked_evidence)
                    or not _safe_backend_evidence(backend_evidence)
                ):
                    result = {}
                    status = "BLOCKED"
                normalized = {
                    "criterionId": criterion,
                    "status": status,
                    "requestIds": result.get("requestIds", []),
                    "evidence": result.get("evidence", []),
                    "backendEvidence": result.get("backendEvidence", []),
                }
                if status != "PASS":
                    root = result.get("rootCauseId")
                    normalized["rootCauseId"] = (
                        root
                        if isinstance(root, str) and root in SYNCHRONIZED_ROOTS
                        else INFRASTRUCTURE_ROOT
                    )
                results.append(normalized)
        backend_path = evidence / "backend/admin-authorization.json"
        for result in results:
            result.setdefault("evidence", [])
            if backend_path.is_file():
                result["evidence"].append(
                    {"type": "backend", "path": "backend/admin-authorization.json"}
                )
            if failure_status is not None:
                result["status"] = failure_status
                result["rootCauseId"] = INFRASTRUCTURE_ROOT
                result["resolvedRootCauseIds"] = []
            if (
                result["criterionId"] == "P08-SWR054-STEP-02"
                and result["status"] == "PASS"
            ):
                result["resolvedRootCauseIds"] = [RESOLVED_MOBILE_NAVIGATION_ROOT]
        (evidence / "results.json").write_text(
            json.dumps({"results": results}, indent=2, sort_keys=True) + "\n",
            encoding="utf-8",
        )


def _safe_request_ids(values: object) -> bool:
    return isinstance(values, list) and all(
        isinstance(value, str) and REQUEST_ID_PATTERN.fullmatch(value)
        for value in values
    )


def _safe_evidence(values: object) -> bool:
    return isinstance(values, list) and all(
        isinstance(value, dict)
        and set(value) == {"type", "path"}
        and isinstance(value["type"], str)
        and value["type"] in {"playwright", "backend"}
        and isinstance(value["path"], str)
        and value["path"]
        and not Path(value["path"]).is_absolute()
        and ".." not in Path(value["path"]).parts
        for value in values
    )


def _safe_backend_evidence(values: object) -> bool:
    return isinstance(values, list) and all(
        isinstance(value, str)
        and BACKEND_SUMMARY_PATTERN.fullmatch(value)
        and value.split("=", 1)[0] in BACKEND_SUMMARY_KEYS
        for value in values
    )


def parse_args(argv: Sequence[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--timeout-seconds", type=float, default=180)
    parser.add_argument(
        "--defer-report",
        action="store_true",
        help="retain the isolated result for finding synchronization before reporting",
    )
    return parser.parse_args(argv)


def safe_playwright_diagnostics(
    phase: str,
    run_id: str,
    environment: dict[str, str],
    error: BaseException | None,
) -> str:
    """Return only redacted test titles and assertion summaries from a failed phase."""

    if not isinstance(error, subprocess.CalledProcessError):
        return f"{phase}_failed"
    text = "\n".join(part for part in (error.stdout, error.stderr) if isinstance(part, str))
    sensitive = (
        environment.get("MEALSWAPP_E2E_EMAIL", ""),
        environment.get("MEALSWAPP_E2E_PASSWORD", ""),
        run_id,
    )
    lines: list[str] = []
    for line in text.splitlines():
        if not any(
            marker in line
            for marker in (
                "task281-",
                "Error:",
                "TimeoutError:",
                "Expected:",
                "Received:",
                "Call log:",
            )
        ):
            continue
        for value in sensitive:
            if value:
                line = line.replace(value, "[redacted]")
        line = re.sub(
            r"[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}",
            "[uuid]",
            line,
            flags=re.IGNORECASE,
        )
        lines.append(line[:300])
        if len(lines) == 40:
            break
    return "\n".join(lines) if lines else f"{phase}_failed"


def finalize_report(
    run_id: str,
    evidence: Path,
    *,
    defer_report: bool,
    output_root: Path | None = None,
) -> int:
    """Finalize one structured Task 281 result and return its truthful status."""

    results = json.loads((evidence / "results.json").read_text(encoding="utf-8"))[
        "results"
    ]
    statuses = {result["status"] for result in results}
    product_code = 1 if "FAIL" in statuses else 2 if "BLOCKED" in statuses else 0
    if defer_report:
        print(f"Task 281 result retained: {run_id}")
        return product_code
    command = [
        sys.executable,
        str(Path(__file__).with_name("phase08_acceptance.py")),
        "report",
        "--run-id",
        f"task281-{run_id}",
        "--result",
        str(evidence / "results.json"),
        "--evidence-root",
        str(evidence),
        "--requirement",
        "SW-REQ-054",
    ]
    if output_root is not None:
        command.extend(("--output-root", str(output_root)))
    report = subprocess.run(
        command,
        cwd=ROOT,
        check=False,
    )
    report_path = (output_root or PHASE08_REPORT_ROOT) / f"task281-{run_id}" / "report.json"
    if report.returncode not in {0, 1, 2} or not report_path.is_file():
        raise ReportFinalizationError("Task 280 did not publish the synchronized report")
    published = json.loads(report_path.read_text(encoding="utf-8"))
    if published.get("runId") != f"task281-{run_id}" or published.get("exitCode") != report.returncode:
        raise ReportFinalizationError("Task 280 published an inconsistent report")
    return report.returncode


def finalize_failure(
    run_id: str,
    evidence: Path,
    error: BaseException,
    *,
    defer_report: bool,
    output_root: Path | None = None,
) -> int:
    """Convert lifecycle failures into a complete fail-closed acceptance report."""

    evidence.mkdir(parents=True, exist_ok=True)
    failure_status = "FAIL" if isinstance(error, EvidenceAssertionError) else "BLOCKED"
    Task281Harness.combine_results(evidence, failure_status=failure_status)
    print(f"Task 281 acceptance failed safely: {type(error).__name__}", file=sys.stderr)
    code = finalize_report(
        run_id,
        evidence,
        defer_report=defer_report,
        output_root=output_root,
    )
    return code if code else 1


def main(argv: Sequence[str] | None = None) -> int:
    args = parse_args(sys.argv[1:] if argv is None else argv)
    harness: Task281Harness | None = None
    fallback_run_id = secrets.token_hex(12)
    try:
        environment = os.environ.get("MEALSWAPP_ENV", "development")
        real_stack.validate_environment(environment)
        target = real_stack.PostgresTarget.parse(
            os.environ.get(
                "MEALSWAPP_E2E_POSTGRES_ADMIN_URL",
                "postgres://mealswapp:mealswapp@127.0.0.1:5432/postgres",
            )
        )
        controller = real_stack.SignalController()
        previous = real_stack.install_signal_handlers(controller)
        harness = Task281Harness(
            target,
            timeout=args.timeout_seconds,
            signal_controller=controller,
        )
        try:
            harness.run()
        finally:
            real_stack.restore_signal_handlers(previous)
        return finalize_report(
            harness.run_id,
            harness.artifacts / "acceptance",
            defer_report=args.defer_report,
        )
    except (Exception, BaseExceptionGroup) as error:
        run_id = harness.run_id if harness is not None else fallback_run_id
        evidence = (
            harness.artifacts / "acceptance"
            if harness is not None
            else real_stack.ARTIFACT_ROOT / run_id / "acceptance"
        )
        try:
            return finalize_failure(
                run_id,
                evidence,
                error,
                defer_report=args.defer_report,
            )
        except Exception as report_error:
            print(
                f"Task 281 report finalization failed safely: {type(report_error).__name__}",
                file=sys.stderr,
            )
            return 1


if __name__ == "__main__":
    raise SystemExit(main())
