#!/usr/bin/env python3
"""Run Task 285 deployed centralized-logging acceptance."""

# Implements DESIGN-014 LogAggregator SW-REQ-084 deployed acceptance.
from __future__ import annotations

import argparse
import datetime as dt
import importlib.util
import ipaddress
import json
import os
import secrets
import socket
import subprocess
import sys
from pathlib import Path
from collections.abc import Callable, Sequence
from typing import Any
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parents[1]
CRITERIA = (
    "P08-SWR084-STEP-01",
    "P08-SWR084-STEP-02",
    "P08-SWR084-STEP-03",
    "P08-SWR084-ACCEPT-01",
    "P08-SWR084-ACCEPT-02",
    "P08-SWR084-ACCEPT-03",
    "P08-SWR084-ACCEPT-04",
)
DEPLOYMENT_ROOT = "ROOT-T285-DEPLOYED-ENVIRONMENT"
ACTION_ROOT = "ROOT-T285-ACTION-COVERAGE"
PRODUCT_ROOT = "ROOT-T285-CENTRALIZED-LOGGING"
REPORT_ROOT = ROOT / "logs/phase08-acceptance"
ARTIFACT_ROOT = ROOT / "logs/task285-deployed"


class SafeArgumentParser(argparse.ArgumentParser):
    """Raise input errors so main can publish a structured blocked report."""

    def error(self, message: str) -> None:
        raise argparse.ArgumentError(None, message)


def load_module(name: str, path: Path):
    """Load one repository script without creating a package surface."""
    spec = importlib.util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"{name} could not be loaded")
    module = importlib.util.module_from_spec(spec)
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module


sink = load_module("task285_log_sink", Path(__file__).with_name("task285_log_sink.py"))
REQUIRED_CATEGORIES = frozenset(sink.CANONICAL_EVENT_CONTRACTS)


def resolve_addresses(host: str) -> Sequence[str]:
    """Resolve every address used to reject non-public deployed targets."""
    try:
        return tuple(
            sorted({item[4][0] for item in socket.getaddrinfo(host, 443, type=socket.SOCK_STREAM)})
        )
    except socket.gaierror as error:
        raise sink.SinkUnavailableBlocked("deployed test target ownership cannot be verified") from error


def is_public_address(value: str) -> bool:
    """Return whether an address is globally routable, including mapped IPv6."""
    try:
        address = ipaddress.ip_address(value)
    except ValueError:
        return False
    if isinstance(address, ipaddress.IPv6Address) and address.ipv4_mapped:
        address = address.ipv4_mapped
    return address.is_global


def validate_deployed_url(
    value: str,
    acknowledgement: str,
    approved_host: str,
    *,
    resolver: Callable[[str], Sequence[str]] = resolve_addresses,
) -> str:
    """Require an approved, credential-free, publicly routed HTTPS origin."""
    parsed = urlsplit(value)
    hostname = (parsed.hostname or "").rstrip(".").casefold()
    approved = approved_host.rstrip(".").casefold()
    if (
        acknowledgement != "deployed-test-with-centralized-log-sink"
        or parsed.scheme != "https"
        or not hostname
        or not approved
        or hostname != approved
        or parsed.username
        or parsed.password
        or parsed.path not in {"", "/"}
        or parsed.query
        or parsed.fragment
    ):
        raise sink.SinkUnavailableBlocked("deployed test target is not explicitly acknowledged")
    try:
        literal = ipaddress.ip_address(hostname)
        addresses = (str(literal),)
    except ValueError:
        addresses = tuple(resolver(hostname))
    if not addresses or any(not is_public_address(address) for address in addresses):
        raise sink.SinkUnavailableBlocked("deployed test target is not publicly routed")
    return value.rstrip("/")


def blocked_results(root: str) -> list[dict[str, Any]]:
    """Materialize every SW-REQ-084 row as BLOCKED for one synchronized root."""
    return [
        {
            "criterionId": criterion,
            "status": "BLOCKED",
            "rootCauseId": root,
            "requestIds": [],
            "evidence": [{"type": "deployed-logs", "path": "sink-evidence.json"}],
            "backendEvidence": ["log_sink_state=blocked"],
        }
        for criterion in CRITERIA
    ]


def write_json(path: Path, value: object) -> None:
    """Write one private deterministic acceptance artifact."""
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    path.chmod(0o600)


def write_blocked_evidence(directory: Path, code: str) -> None:
    """Record only a fixed blocker code, never configuration diagnostics."""
    write_json(
        directory / "sink-evidence.json",
        {
            "schema": "mealswapp.task285-sink-evidence.v1",
            "sink": "gcp_cloud_logging",
            "status": "BLOCKED",
            "code": code,
            "observations": [],
        },
    )


def run_playwright(
    directory: Path,
    environment: dict[str, str],
    base_url: str,
    provenance: str,
) -> Path:
    """Drive production UI/API actions and return the safe receipt path."""
    action_file = directory / "actions.json"
    playwright_environment = {
        **environment,
        "MEALSWAPP_TASK285_DEPLOYED_BASE_URL": base_url,
        "MEALSWAPP_TASK285_ACTION_FILE": str(action_file),
        "MEALSWAPP_TASK285_ACTION_PROVENANCE": provenance,
        "MEALSWAPP_TASK285_MARKER": environment["MEALSWAPP_TASK285_MARKER"],
        "PLAYWRIGHT_OUTPUT_DIR": str(directory / "playwright"),
    }
    try:
        completed = subprocess.run(
            [
                "bunx",
                "playwright",
                "test",
                "-c",
                "playwright.task285.config.ts",
                "tests/task285-centralized-logging.spec.ts",
            ],
            cwd=ROOT / "frontend",
            env=playwright_environment,
            check=False,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            timeout=300,
        )
    except (OSError, subprocess.SubprocessError) as error:
        raise sink.SinkUnavailableBlocked("deployed action production did not complete") from error
    if completed.returncode or not action_file.is_file():
        raise sink.SinkUnavailableBlocked("deployed action production did not complete")
    return action_file


def query_window(receipt: dict[str, Any]) -> sink.QueryWindow:
    """Expand the action interval within the fixed 15-minute query bound."""
    started = sink.parse_timestamp(receipt.get("startedAt"), "action start") - dt.timedelta(seconds=30)
    finished = sink.parse_timestamp(receipt.get("finishedAt"), "action finish") + dt.timedelta(minutes=2)
    return sink.QueryWindow(started, min(finished, started + dt.timedelta(seconds=sink.MAX_WINDOW_SECONDS)))


def safe_probes(environment: dict[str, str]) -> list[str]:
    """Return runtime private values used only for in-memory redaction scans."""
    names = (
        "MEALSWAPP_TASK285_ADMIN_EMAIL",
        "MEALSWAPP_TASK285_ADMIN_PASSWORD",
        "MEALSWAPP_TASK285_EXTERNAL_QUERY",
        "MEALSWAPP_TASK285_DEPENDENCY_QUERY",
        "MEALSWAPP_TASK285_USER_LOOKUP",
        "MEALSWAPP_TASK285_MARKER",
    )
    return [environment[name] for name in names if len(environment.get(name, "")) >= 4]


def retention_probe(environment: dict[str, str]) -> tuple[sink.ExpectedEvent, sink.QueryWindow]:
    """Require a safe event that proves the minimum 90-day deployed retention."""
    request_id = environment.get("MEALSWAPP_TASK285_RETENTION_REQUEST_ID", "")
    try:
        occurred = sink.parse_timestamp(
            environment.get("MEALSWAPP_TASK285_RETENTION_OCCURRED_AT", ""),
            "retention occurrence",
        )
    except sink.SinkMalformed as error:
        raise sink.SinkUnavailableBlocked("centralized log retention evidence is incomplete") from error
    if (
        not sink.REQUEST_ID_RE.fullmatch(request_id)
        or occurred > dt.datetime.now(dt.timezone.utc) - dt.timedelta(days=90)
    ):
        raise sink.SinkUnavailableBlocked("centralized log retention evidence is incomplete")
    return (
        sink.ExpectedEvent(
            "retention_probe",
            request_id,
            "logging_retention",
            "centralized_log",
            "retained",
        ),
        sink.QueryWindow(occurred - dt.timedelta(minutes=1), occurred + dt.timedelta(minutes=1)),
    )


def evaluate(
    expected: list[sink.ExpectedEvent],
    observed: list[sink.SafeEvent],
    evidence_path: str,
) -> list[dict[str, Any]]:
    """Map sink and action evidence to every Task 280 criterion truthfully."""
    sink.validate_expected_events(expected)
    categories = {item.category for item in expected}
    request_ids = sorted(item.request_id for item in expected)
    contract_findings = sink.verify_event_contract(expected, observed)
    complete_actions = REQUIRED_CATEGORIES <= categories
    matched = not contract_findings
    statuses = {
        "P08-SWR084-STEP-01": "PASS" if {"manual_success", "manual_failure"} <= categories and matched else "FAIL",
        "P08-SWR084-STEP-02": "PASS" if {"external_search", "external_import"} <= categories and matched else "FAIL",
        "P08-SWR084-STEP-03": "PASS" if observed and matched else "FAIL",
        "P08-SWR084-ACCEPT-01": "PASS" if observed and matched else "FAIL",
        "P08-SWR084-ACCEPT-02": "PASS" if matched else "FAIL",
        "P08-SWR084-ACCEPT-03": "PASS",
        "P08-SWR084-ACCEPT-04": "PASS" if complete_actions and observed and matched else "FAIL",
    }
    results = []
    for criterion in CRITERIA:
        status = statuses[criterion]
        results.append(
            {
                "criterionId": criterion,
                "status": status,
                **({"rootCauseId": ACTION_ROOT if not complete_actions else PRODUCT_ROOT} if status != "PASS" else {}),
                "requestIds": request_ids,
                "evidence": [{"type": "deployed-logs", "path": evidence_path}],
                "backendEvidence": [
                    f"log_sink_state={'centralized' if observed else 'missing'}",
                    f"request_correlation={'complete' if matched else 'incomplete'}",
                ],
            }
        )
    return results


def verify(
    directory: Path,
    environment: dict[str, str],
    *,
    client: sink.CloudLoggingClient | None = None,
    _action_producer: Callable[[Path, dict[str, str], str, str], Path] = run_playwright,
    _resolver: Callable[[str], Sequence[str]] = resolve_addresses,
) -> tuple[list[dict[str, Any]], str]:
    """Produce actions through Playwright, query the sink, and return safe results."""
    base_url = validate_deployed_url(
        environment.get("MEALSWAPP_TASK285_DEPLOYED_BASE_URL", ""),
        environment.get("MEALSWAPP_TASK285_DEPLOYMENT_ACK", ""),
        environment.get("MEALSWAPP_TASK285_APPROVED_DEPLOYED_HOST", ""),
        resolver=_resolver,
    )
    config = sink.SinkConfig.from_environment(environment)
    provenance = secrets.token_hex(24)
    action_path = _action_producer(directory, environment, base_url, provenance)
    try:
        receipt = json.loads(action_path.read_text(encoding="utf-8"))
        expected = sink.load_expected_events(receipt, provenance)
        window = query_window(receipt)
    except (
        OSError,
        UnicodeError,
        json.JSONDecodeError,
        sink.SinkMalformed,
        ValueError,
        RecursionError,
    ) as error:
        raise sink.SinkUnavailableBlocked("deployed action receipt is invalid") from error
    categories = {item.category for item in expected}
    if not REQUIRED_CATEGORIES <= categories:
        raise sink.SinkUnavailableBlocked("deployed action coverage is incomplete")
    cloud = client or sink.CloudLoggingClient(config)
    observed, polls, pages = sink.poll_for_events(
        cloud,
        expected,
        window,
        safe_probes({**environment, "MEALSWAPP_TASK285_MARKER": environment.get("MEALSWAPP_TASK285_MARKER", "")}),
    )
    retained, retention_window = retention_probe(environment)
    retention_observed, retention_polls, retention_pages = sink.poll_for_events(
        cloud,
        [retained],
        retention_window,
        safe_probes(environment),
    )
    if sink.verify_event_contract([retained], retention_observed):
        raise sink.SinkUnavailableBlocked("centralized log retention evidence is incomplete")
    observed.extend(retention_observed)
    polls += retention_polls
    pages += retention_pages
    evidence = {
        "schema": "mealswapp.task285-sink-evidence.v1",
        "sink": "gcp_cloud_logging",
        "status": "QUERIED",
        "windowSeconds": int((window.end - window.start).total_seconds()),
        "polls": polls,
        "pages": pages,
        "observations": [
            {
                "requestId": item.request_id,
                "timestamp": item.timestamp,
                "ingestedAt": item.ingested_at,
                "action": item.action,
                "resource": item.resource,
                "outcome": item.outcome,
            }
            for item in observed
        ],
    }
    write_json(directory / "sink-evidence.json", evidence)
    return evaluate(expected, observed, "sink-evidence.json"), "PASS"


def finalize_report(run_id: str, directory: Path, results: list[dict[str, Any]]) -> int:
    """Synchronize results through Task 280 and return its exact exit semantics."""
    result_path = directory / "results.json"
    write_json(result_path, {"results": results})
    try:
        completed = subprocess.run(
            [
                sys.executable,
                str(ROOT / "scripts/phase08_acceptance.py"),
                "report",
                "--run-id",
                f"task285-{run_id}",
                "--result",
                str(result_path),
                "--evidence-root",
                str(directory),
                "--requirement",
                "SW-REQ-084",
            ],
            cwd=ROOT,
            check=False,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            timeout=60,
        )
    except (OSError, subprocess.SubprocessError) as error:
        raise sink.SinkUnavailableBlocked("acceptance report could not be finalized") from error
    report = REPORT_ROOT / f"task285-{run_id}" / "report.json"
    if not report.is_file():
        raise sink.SinkUnavailableBlocked("acceptance report could not be finalized")
    return completed.returncode


def write_fallback_report(run_id: str, directory: Path, results: list[dict[str, Any]], code: str) -> None:
    """Write a complete safe report when Task 280 report execution is unavailable."""
    report_directory = REPORT_ROOT / f"task285-{run_id}"
    write_json(
        report_directory / "report.json",
        {
            "schema": "mealswapp.task285-blocked-report.v1",
            "runId": f"task285-{run_id}",
            "status": "BLOCKED",
            "exitCode": 2,
            "findingIds": ["P08-FIND-285-001"],
            "code": code,
            "evidenceRoot": str(directory.relative_to(ROOT)),
            "results": results,
        },
    )


def result_exit_code(results: list[dict[str, Any]]) -> int:
    """Return Task 280 PASS/FAIL/BLOCKED exit semantics for deferred runs."""
    statuses = {item.get("status") for item in results}
    if "FAIL" in statuses:
        return 1
    if "BLOCKED" in statuses:
        return 2
    return 0


def main(argv: list[str] | None = None) -> int:
    """Run deployed acceptance or deterministically publish synchronized BLOCKED evidence."""
    parser = SafeArgumentParser(exit_on_error=False)
    parser.add_argument("--defer-report", action="store_true")
    run_id = secrets.token_hex(12)
    directory = ARTIFACT_ROOT / run_id
    directory.mkdir(parents=True, mode=0o700)
    environment = dict(os.environ)
    environment["MEALSWAPP_TASK285_MARKER"] = f"logaccept-{secrets.token_hex(12)}"
    defer_report = False
    try:
        try:
            args = parser.parse_args(argv)
        except (argparse.ArgumentError, SystemExit) as error:
            raise sink.SinkUnavailableBlocked("acceptance runner input is invalid") from error
        defer_report = args.defer_report
        results, _ = verify(directory, environment)
    except sink.RedactionViolation:
        write_blocked_evidence(directory, "FORBIDDEN-CENTRALIZED-DATA")
        results = [
            {
                **item,
                "status": "FAIL",
                "rootCauseId": PRODUCT_ROOT,
                "backendEvidence": ["log_sink_state=unsafe"],
            }
            for item in blocked_results(PRODUCT_ROOT)
        ]
    except sink.SinkMalformed:
        write_blocked_evidence(directory, "MALFORMED-CENTRALIZED-EVIDENCE")
        results = [
            {
                **item,
                "status": "FAIL",
                "rootCauseId": PRODUCT_ROOT,
                "backendEvidence": ["log_sink_state=malformed"],
            }
            for item in blocked_results(PRODUCT_ROOT)
        ]
    except sink.SinkAuthorizationBlocked:
        write_blocked_evidence(directory, "QUERY-AUTHORITY-BLOCKED")
        results = blocked_results(DEPLOYMENT_ROOT)
    except sink.SinkUnavailableBlocked as error:
        code = "ACTION-COVERAGE-BLOCKED" if "action" in str(error) else "DEPLOYMENT-SINK-BLOCKED"
        write_blocked_evidence(directory, code)
        results = blocked_results(ACTION_ROOT if "action" in str(error) else DEPLOYMENT_ROOT)
    except Exception:
        write_blocked_evidence(directory, "RUNNER-VALIDATION-BLOCKED")
        results = blocked_results(DEPLOYMENT_ROOT)
    if defer_report:
        write_json(directory / "results.json", {"results": results})
        return result_exit_code(results)
    try:
        return finalize_report(run_id, directory, results)
    except Exception:
        write_blocked_evidence(directory, "REPORT-FINALIZATION-BLOCKED")
        results = blocked_results(DEPLOYMENT_ROOT)
        write_json(directory / "results.json", {"results": results})
        write_fallback_report(run_id, directory, results, "REPORT-FINALIZATION-BLOCKED")
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
