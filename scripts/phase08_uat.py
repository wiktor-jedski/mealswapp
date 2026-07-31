#!/usr/bin/env python3
"""Build and validate the Phase 08.02 cross-cutting acceptance gate.

Implements DESIGN-014 MetricsCollector for Task 286 acceptance aggregation.
"""

from __future__ import annotations

import argparse
import hashlib
import html
import json
import re
import shutil
import sys
import tempfile
from collections import Counter, defaultdict
from datetime import date
from pathlib import Path
from pathlib import PurePosixPath
from typing import Any

import phase08_acceptance as acceptance

ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ROOT / "docs/testing/phase08/acceptance-manifest.json"
OPEN_DOC = ROOT / "docs/implementation/04_OPEN.md"
HISTORY = ROOT / "docs/testing/phase08/finding-history.json"
TASK_LIST = ROOT / "docs/implementation/02_TASK_LIST.md"
INPUT = ROOT / "docs/testing/phase08/uat-input.json"
REPORT_JSON = ROOT / "docs/implementation/implemented/08.02_PHASE_REPORT.json"
REPORT_HTML = ROOT / "docs/implementation/implemented/08.02_PHASE_REPORT.html"
UAT = ROOT / "docs/implementation/implemented/08.02_PHASE_UAT.md"
EVIDENCE_ROOT = ROOT / "docs/implementation/implemented/08.02_PHASE_EVIDENCE"
SCHEMA = "mealswapp.phase08.02-uat-report.v1"
INPUT_SCHEMA = "mealswapp.phase08.02-uat-input.v1"
STATUS_RANK = {"PASS": 0, "BLOCKED": 1, "FAIL": 2}
TASK_ROW_RE = re.compile(r"^\|\s*(\d+)\s*\|")
SOURCE_REPORT_KEYS = {
    "schema",
    "runId",
    "status",
    "exitCode",
    "producerFailures",
    "rootCauses",
    "results",
}
SOURCE_RESULT_KEYS = {
    "backendEvidence",
    "criterionId",
    "evidence",
    "evidenceTypes",
    "findingIds",
    "id",
    "kind",
    "owner",
    "requestIds",
    "requiredEnvironment",
    "requirement",
    "resolvedRootCauseIds",
    "rootCauseId",
    "scenarioId",
    "source",
    "status",
    "text",
}
FINAL_REPORT_KEYS = {
    "schema",
    "generatedAtUtc",
    "gateStatus",
    "counts",
    "acceptanceDecision",
    "historicalUat",
    "openFindings",
    "controls",
    "closureEvidence",
    "runContextEvidence",
    "sourceEvidence",
    "taskTrace",
    "results",
    "renderedArtifacts",
}
FINAL_RESULT_KEYS = {
    "criterionId",
    "requirement",
    "scenarioId",
    "kind",
    "text",
    "status",
    "rootCauseIds",
    "findingIds",
    "sources",
}
FINAL_SOURCE_KEYS = {
    "taskId",
    "report",
    "status",
    "rootCauseId",
    "requestIds",
    "evidence",
    "backendEvidence",
}
HASH_ROW_KEYS = {"path", "sha256"}
TASK_TRACE_KEYS = {
    "taskId",
    "component",
    "staticAspect",
    "status",
    "description",
    "dependsOn",
    "verificationCriteria",
    "evidence",
}
DECISION_KEYS = {"status", "owner", "date", "deviations"}
DEVIATION_KEYS = {"criterionId", "owner", "date", "reason", "retestCondition", "expiry"}
INPUT_KEYS = {
    "schema",
    "generatedAtUtc",
    "historicalUat",
    "reports",
    "supportingArtifacts",
    "taskEvidence",
    "acceptanceDecision",
}


class UATError(ValueError):
    """Raised when Phase 08.02 acceptance evidence is incomplete or contradictory."""


def sha256(path: Path) -> str:
    """Return the SHA-256 digest of one regular file."""

    try:
        relative_path = path.relative_to(ROOT)
        path.resolve(strict=True).relative_to(ROOT.resolve())
    except (OSError, ValueError) as error:
        raise UATError("evidence is missing or leaves the repository") from error
    current = ROOT
    if any((current := current / part).is_symlink() for part in relative_path.parts):
        raise UATError(f"evidence has a symlink component: {relative_path.as_posix()}")
    if not path.is_file():
        raise UATError(f"evidence is missing or unsafe: {relative_path.as_posix()}")
    return hashlib.sha256(path.read_bytes()).hexdigest()


def relative(path: Path) -> str:
    """Return a repository-relative POSIX path."""

    try:
        return path.resolve().relative_to(ROOT.resolve()).as_posix()
    except ValueError as error:
        raise UATError("evidence path leaves the repository") from error


def repo_path(value: Any) -> Path:
    """Resolve one strict repository-relative path without traversal."""

    if not isinstance(value, str):
        raise UATError("evidence path must be a string")
    path = PurePosixPath(value)
    if path.is_absolute() or not path.parts or any(part in {"", ".", ".."} for part in path.parts):
        raise UATError("evidence path must be repository-relative")
    return ROOT.joinpath(*path.parts)


def load_json(path: Path) -> dict[str, Any]:
    """Load a duplicate-key-free JSON object."""

    try:
        return acceptance.strict_json_loads(path.read_text(encoding="utf-8"), relative(path))
    except (OSError, UnicodeError, acceptance.ValidationError) as error:
        raise UATError(f"cannot load JSON evidence: {relative(path)}") from error


def load_controls() -> tuple[list[dict[str, Any]], dict[str, dict[str, Any]]]:
    """Load and validate the Task 280 manifest and synchronized finding ledger."""

    try:
        criteria = acceptance.validate_manifest(load_json(MANIFEST), ROOT / "req_tests.md")
        history = acceptance.validate_finding_history(load_json(HISTORY))
        ledger = acceptance.extract_finding_ledger(OPEN_DOC.read_text(encoding="utf-8"))
        findings = acceptance.validate_findings(
            ledger, {item["scenarioId"] for item in criteria}, history
        )
    except (OSError, UnicodeError, acceptance.ValidationError) as error:
        raise UATError("Task 280 controls are invalid") from error
    return criteria, findings


def task_rows() -> dict[int, dict[str, str]]:
    """Parse Tasks 276-286 without changing the task list."""

    rows: dict[int, dict[str, str]] = {}
    for line in TASK_LIST.read_text(encoding="utf-8").splitlines():
        match = TASK_ROW_RE.match(line)
        if not match or not 276 <= int(match.group(1)) <= 286:
            continue
        columns = [part.strip() for part in line.strip().strip("|").split("|")]
        if len(columns) != 9:
            raise UATError(f"Task {match.group(1)} row is malformed")
        task_id = int(columns[0])
        rows[task_id] = {
            "taskId": task_id,
            "component": columns[1],
            "staticAspect": columns[2],
            "status": columns[3],
            "description": columns[5],
            "dependsOn": columns[6],
            "verificationCriteria": columns[8],
        }
    if set(rows) != set(range(276, 287)):
        raise UATError("Task trace must cover Tasks 276-286 exactly")
    return rows


def expected_report_status(results: list[dict[str, Any]]) -> tuple[str, int]:
    """Compute the truthful Task 280 status and exit code."""

    statuses = {item["status"] for item in results}
    if "FAIL" in statuses:
        return "FAIL", 1
    if "BLOCKED" in statuses:
        return "BLOCKED", 2
    return "PASS", 0


def validate_string_list(
    value: Any,
    label: str,
    pattern: re.Pattern[str] | None = None,
) -> list[str]:
    """Return one sorted unique string list with an optional item contract."""

    if (
        not isinstance(value, list)
        or any(not isinstance(item, str) or (pattern is not None and not pattern.fullmatch(item)) for item in value)
        or value != sorted(set(value))
    ):
        raise UATError(f"{label} must be a sorted unique string array")
    return value


def validate_backend_evidence(value: Any, criterion_id: str) -> list[str]:
    """Validate sanitized backend summaries using the Task 280 contract."""

    items = validate_string_list(
        value,
        f"{criterion_id}: backendEvidence",
        acceptance.BACKEND_SUMMARY_RE,
    )
    if any(item.split("=", 1)[0] not in acceptance.BACKEND_SUMMARY_KEYS for item in items):
        raise UATError(f"{criterion_id}: backendEvidence contains an unsupported summary")
    return items


def validate_source_report(
    path: Path,
    criteria_by_id: dict[str, dict[str, Any]],
    findings: dict[str, dict[str, Any]],
) -> dict[str, Any]:
    """Validate one producer report and every evidence link it claims."""

    report = load_json(path)
    try:
        acceptance.assert_safe_value(report, f"source report {relative(path)}")
    except acceptance.ValidationError as error:
        raise UATError(f"source report contains prohibited sensitive metadata: {relative(path)}") from error
    if set(report) != SOURCE_REPORT_KEYS:
        raise UATError(f"source report schema is malformed: {relative(path)}")
    if report["schema"] != "mealswapp.phase08-acceptance-report.v1":
        raise UATError(f"unsupported source report: {relative(path)}")
    if (
        not isinstance(report["runId"], str)
        or not acceptance.RUN_ID_RE.fullmatch(report["runId"])
        or not isinstance(report["exitCode"], int)
        or isinstance(report["exitCode"], bool)
    ):
        raise UATError(f"source report identity is malformed: {relative(path)}")
    validate_string_list(report["producerFailures"], "producerFailures")
    results = report["results"]
    if not isinstance(results, list) or not results:
        raise UATError(f"source report has no results: {relative(path)}")
    sha256(path.parent / "report.html")
    seen: set[str] = set()
    referenced = {"report.json", "report.html"}
    for result in results:
        if not isinstance(result, dict) or set(result) != SOURCE_RESULT_KEYS:
            raise UATError(f"source result schema is malformed: {relative(path)}")
        criterion_id = result["criterionId"]
        if not isinstance(criterion_id, str):
            raise UATError(f"source result criterionId is malformed: {relative(path)}")
        if criterion_id not in criteria_by_id or criterion_id in seen:
            raise UATError(f"orphan or duplicate criterion in {relative(path)}")
        seen.add(criterion_id)
        criterion = criteria_by_id[criterion_id]
        for key in (
            "evidenceTypes",
            "id",
            "kind",
            "owner",
            "requiredEnvironment",
            "requirement",
            "scenarioId",
            "source",
            "text",
        ):
            expected_value = criterion["id"] if key == "id" else criterion[key]
            if result[key] != expected_value:
                raise UATError(f"{criterion_id}: contradictory source metadata")
        validate_string_list(
            result["requestIds"],
            f"{criterion_id}: requestIds",
            acceptance.REQUEST_ID_RE,
        )
        validate_backend_evidence(result["backendEvidence"], criterion_id)
        validate_string_list(
            result["resolvedRootCauseIds"],
            f"{criterion_id}: resolvedRootCauseIds",
            acceptance.SAFE_ID_RE,
        )
        validate_string_list(
            result["findingIds"],
            f"{criterion_id}: findingIds",
            acceptance.SAFE_ID_RE,
        )
        status = result["status"]
        root = result["rootCauseId"]
        if not isinstance(status, str) or status not in STATUS_RANK:
            raise UATError(f"{criterion_id}: invalid source status")
        if status == "PASS" and root is not None:
            raise UATError(f"{criterion_id}: PASS has an unresolved root")
        if status != "PASS" and (
            not isinstance(root, str) or not acceptance.SAFE_ID_RE.fullmatch(root)
        ):
            raise UATError(f"{criterion_id}: non-pass requires a valid root cause")
        if status != "PASS":
            finding = next(
                (
                    item
                    for item in findings.values()
                    if item["rootCauseId"] == root and item["status"] != "CLOSED"
                ),
                None,
            )
            if not any(
                item["rootCauseId"] == root
                and criterion["requirement"] in item["requirements"]
                and criterion["scenarioId"] in item["scenarios"]
                for item in findings.values()
            ):
                raise UATError(f"{criterion_id}: non-pass lacks an open synchronized finding")
        resolved_roots = set(result["resolvedRootCauseIds"])
        if any(
            finding["rootCauseId"] in resolved_roots and finding["status"] != "CLOSED"
            for finding in findings.values()
        ):
            raise UATError(f"{criterion_id}: resolved root cause is not closed")
        claimed_roots = resolved_roots | ({root} if root is not None else set())
        expected_finding_ids = sorted(
            finding["id"]
            for finding in findings.values()
            if finding["rootCauseId"] in claimed_roots
        )
        if result["findingIds"] != expected_finding_ids:
            raise UATError(f"{criterion_id}: source finding linkage is contradictory")
        evidence_items = result["evidence"]
        if not isinstance(evidence_items, list):
            raise UATError(f"{criterion_id}: evidence must be an array")
        for evidence in evidence_items:
            if not isinstance(evidence, dict) or set(evidence) != {"type", "path"}:
                raise UATError(f"{criterion_id}: evidence must contain only type and path")
            if evidence["type"] not in criterion["evidenceTypes"]:
                raise UATError(f"{criterion_id}: evidence type is not allowed")
            evidence_path = acceptance.safe_relative_path(evidence.get("path"), "evidence path")
            candidate = path.parent / evidence_path
            try:
                candidate.resolve().relative_to(path.parent.resolve())
            except ValueError as error:
                raise UATError(f"{criterion_id}: evidence escapes its report") from error
            sha256(candidate)
            referenced.add(evidence_path)
    root_causes = report["rootCauses"]
    if not isinstance(root_causes, list):
        raise UATError("source rootCauses must be an array")
    root_ids: set[str] = set()
    for root_cause in root_causes:
        if not isinstance(root_cause, dict) or set(root_cause) != {"id", "criteria"}:
            raise UATError("source root cause schema is malformed")
        root_id = root_cause["id"]
        if (
            not isinstance(root_id, str)
            or not acceptance.SAFE_ID_RE.fullmatch(root_id)
            or root_id in root_ids
        ):
            raise UATError("source root cause identity is malformed")
        root_ids.add(root_id)
        validate_string_list(root_cause["criteria"], f"{root_id}: criteria", acceptance.SAFE_ID_RE)
    expected_roots = {
        result["rootCauseId"]: sorted(
            item["criterionId"] for item in results if item["rootCauseId"] == result["rootCauseId"]
        )
        for result in results
        if result["rootCauseId"] is not None
    }
    if {item["id"]: item["criteria"] for item in root_causes} != expected_roots:
        raise UATError("source root cause linkage is contradictory")
    status, exit_code = expected_report_status(results)
    if report["status"] != status or report["exitCode"] != exit_code:
        raise UATError(f"source report has contradictory aggregate status: {relative(path)}")
    actual_files = {
        item.relative_to(path.parent).as_posix()
        for item in path.parent.rglob("*")
        if item.is_file()
    }
    if actual_files - referenced:
        raise UATError(
            f"source report contains orphan evidence: {sorted(actual_files - referenced)[0]}"
        )
    return report


def canonical_source_reports(
    criteria_by_id: dict[str, dict[str, Any]],
    findings: dict[str, dict[str, Any]],
) -> list[tuple[int, Path, dict[str, Any]]]:
    """Load the exact selected copied reports with task identities from trusted controls."""

    source = load_json(INPUT)
    reports = source.get("reports")
    if not isinstance(reports, list):
        raise UATError("UAT input reports must be an array")
    selected: list[tuple[int, Path, dict[str, Any]]] = []
    identities: set[tuple[int, Path]] = set()
    destinations: set[Path] = set()
    for item in reports:
        if not isinstance(item, dict) or set(item) != {"taskId", "path"}:
            raise UATError("UAT input report selection is malformed")
        task_id = item["taskId"]
        original = repo_path(item["path"])
        if not isinstance(task_id, int) or isinstance(task_id, bool) or task_id not in range(281, 286):
            raise UATError("UAT input source task identity is malformed")
        copied = EVIDENCE_ROOT / original.parent.name / "report.json"
        identity = (task_id, copied)
        if identity in identities or copied in destinations:
            raise UATError("UAT input source identities must be unique")
        identities.add(identity)
        destinations.add(copied)
        selected.append((task_id, copied, validate_source_report(copied, criteria_by_id, findings)))
    if {task_id for task_id, _, _ in selected} != set(range(281, 286)):
        raise UATError("source reports must cover Tasks 281-285")
    return selected


def validate_decision(
    decision: Any,
    results: Any,
) -> None:
    """Reject acceptance without accountable ownership or complete deviations."""

    allowed = {"PENDING", "ACCEPTED", "REJECTED", "ACCEPTED_WITH_DEVIATIONS"}
    if not isinstance(decision, dict) or set(decision) != DECISION_KEYS:
        raise UATError("acceptance decision schema is malformed")
    status = decision["status"]
    owner = decision["owner"]
    decision_date = decision["date"]
    deviations = decision["deviations"]
    if not isinstance(status, str) or status not in allowed:
        raise UATError("acceptance decision is invalid")
    if not isinstance(owner, str) or not isinstance(decision_date, str):
        raise UATError("acceptance decision owner and date must be strings")
    if not isinstance(deviations, list):
        raise UATError("acceptance deviations must be an array")
    if not isinstance(results, list) or any(
        not isinstance(item, dict)
        or not isinstance(item.get("criterionId"), str)
        or not isinstance(item.get("status"), str)
        or item["status"] not in STATUS_RANK
        for item in results
    ):
        raise UATError("acceptance decision results are malformed")
    nonpass = {item["criterionId"] for item in results if item["status"] != "PASS"}
    if status == "PENDING":
        if owner or decision_date or deviations:
            raise UATError("pending acceptance cannot claim owner, date, or accepted deviations")
        return
    if not owner.strip() or not valid_date(decision_date):
        raise UATError("acceptance requires project owner and date")
    if status == "ACCEPTED" and nonpass:
        raise UATError("mandatory non-pass criteria prevent acceptance")
    if status == "ACCEPTED_WITH_DEVIATIONS":
        covered: set[str] = set()
        required = {"criterionId", "owner", "date", "reason", "retestCondition", "expiry"}
        for deviation in deviations:
            if not isinstance(deviation, dict) or set(deviation) != required:
                raise UATError("accepted deviation is incomplete")
            if any(not isinstance(deviation[key], str) or not deviation[key].strip() for key in required):
                raise UATError("accepted deviation fields must be non-empty")
            if deviation["criterionId"] not in nonpass:
                raise UATError("accepted deviation does not map a mandatory non-pass criterion")
            if not valid_date(deviation["date"]) or not valid_date(deviation["expiry"]):
                raise UATError("accepted deviation dates are invalid")
            covered.add(deviation["criterionId"])
        if covered != nonpass:
            raise UATError("every mandatory non-pass criterion needs an accepted deviation")
    elif deviations:
        raise UATError("deviations are allowed only for accepted-with-deviations")


def valid_date(value: Any) -> bool:
    """Return whether a value is a real ISO calendar date."""

    try:
        return isinstance(value, str) and date.fromisoformat(value).isoformat() == value
    except ValueError:
        return False


def validate_build_input(source: Any) -> dict[str, Any]:
    """Type-check build input before dictionary, status, set, or path operations."""

    if not isinstance(source, dict) or set(source) != INPUT_KEYS:
        raise UATError("UAT input schema is malformed")
    if source["schema"] != INPUT_SCHEMA:
        raise UATError("unsupported UAT input schema")
    if not isinstance(source["generatedAtUtc"], str):
        raise UATError("UAT input generation time is malformed")
    historical = source["historicalUat"]
    if (
        not isinstance(historical, dict)
        or set(historical) != HASH_ROW_KEYS
        or not isinstance(historical["path"], str)
        or not isinstance(historical["sha256"], str)
        or not re.fullmatch(r"[0-9a-f]{64}", historical["sha256"])
    ):
        raise UATError("historical UAT input is malformed")
    repo_path(historical["path"])
    for key, valid_tasks in (
        ("reports", range(281, 286)),
        ("supportingArtifacts", range(281, 286)),
        ("taskEvidence", range(276, 286)),
    ):
        items = source[key]
        if not isinstance(items, list):
            raise UATError(f"UAT input {key} must be an array")
        for item in items:
            if not isinstance(item, dict) or set(item) != {"taskId", "path"}:
                raise UATError(f"UAT input {key} entry is malformed")
            task_id = item["taskId"]
            if (
                not isinstance(task_id, int)
                or isinstance(task_id, bool)
                or task_id not in valid_tasks
            ):
                raise UATError(f"UAT input {key} task identity is malformed")
            repo_path(item["path"])
    decision = source["acceptanceDecision"]
    if not isinstance(decision, dict) or set(decision) != DECISION_KEYS:
        raise UATError("acceptance decision schema is malformed")
    if any(not isinstance(decision[key], str) for key in ("status", "owner", "date")):
        raise UATError("acceptance decision scalar fields are malformed")
    deviations = decision["deviations"]
    if not isinstance(deviations, list):
        raise UATError("acceptance deviations must be an array")
    for deviation in deviations:
        if (
            not isinstance(deviation, dict)
            or set(deviation) != DEVIATION_KEYS
            or any(not isinstance(deviation[key], str) for key in DEVIATION_KEYS)
        ):
            raise UATError("accepted deviation is malformed")
    return source


def aggregate(
    source_reports: list[tuple[int, Path, dict[str, Any]]],
    criteria: list[dict[str, Any]],
    findings: dict[str, dict[str, Any]],
) -> list[dict[str, Any]]:
    """Aggregate duplicate cross-feature observations without weakening failures."""

    observations: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for task_id, source_path, report in source_reports:
        for result in report["results"]:
            root = result.get("rootCauseId")
            resolved = (
                result["status"] != "PASS"
                and root is not None
                and any(
                    finding["rootCauseId"] == root and finding["status"] == "CLOSED"
                    for finding in findings.values()
                )
                and not any(
                    finding["rootCauseId"] == root and finding["status"] != "CLOSED"
                    for finding in findings.values()
                )
            )
            observations[result["criterionId"]].append(
                {
                    "taskId": task_id,
                    "report": relative(source_path),
                    "status": "PASS" if resolved else result["status"],
                    "rootCauseId": None if resolved else root,
                    "requestIds": result.get("requestIds", []),
                    "evidence": result.get("evidence", []),
                    "backendEvidence": result.get("backendEvidence", []),
                }
            )
    missing = [item["id"] for item in criteria if item["id"] not in observations]
    if missing:
        raise UATError(f"missing mandatory criteria: {', '.join(missing)}")
    results: list[dict[str, Any]] = []
    for criterion in criteria:
        sources = observations[criterion["id"]]
        status = max((item["status"] for item in sources), key=STATUS_RANK.__getitem__)
        roots = sorted(
            {item["rootCauseId"] for item in sources if item["status"] != "PASS"}
        )
        finding_ids = sorted(
            finding["id"]
            for finding in findings.values()
            if finding["rootCauseId"] in roots and finding["status"] != "CLOSED"
        )
        results.append(
            {
                "criterionId": criterion["id"],
                "requirement": criterion["requirement"],
                "scenarioId": criterion["scenarioId"],
                "kind": criterion["kind"],
                "text": criterion["text"],
                "status": status,
                "rootCauseIds": roots,
                "findingIds": finding_ids,
                "sources": sources,
            }
        )
    return results


def verify_special_evidence(results: list[dict[str, Any]]) -> None:
    """Enforce centralized-logging and erasure evidence boundaries."""

    for result in results:
        if result["requirement"] == "SW-REQ-084" and result["status"] == "PASS":
            if not any(
                evidence.get("type") == "deployed-logs"
                and "log_sink_state=centralized" in source["backendEvidence"]
                for source in result["sources"]
                for evidence in source["evidence"]
            ):
                raise UATError("centralized logging cannot pass from console evidence")
        if result["requirement"] == "SW-REQ-073" and result["status"] == "PASS":
            if not any(
                "worker_state=completed" in source["backendEvidence"]
                and any(evidence.get("type") == "backend" for evidence in source["evidence"])
                for source in result["sources"]
            ):
                raise UATError("erasure cannot pass without worker/backend evidence")


def copy_report_tree(source: Path, destination: Path) -> None:
    """Copy one sanitized report tree through an atomic staging directory."""

    destination.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix=".phase08-uat-", dir=destination.parent) as staging:
        staged = Path(staging) / destination.name
        shutil.copytree(source.parent, staged)
        if destination.exists():
            shutil.rmtree(destination)
        staged.replace(destination)


def evidence_hashes(paths: list[Path]) -> list[dict[str, str]]:
    """Hash a deterministic set of evidence files."""

    return [
        {"path": relative(path), "sha256": sha256(path)}
        for path in sorted(set(paths), key=lambda item: relative(item))
    ]


def render_html(report: dict[str, Any]) -> str:
    """Render a self-contained human-readable acceptance report."""

    rows = "".join(
        "<tr>"
        f"<td>{html.escape(item['requirement'])}</td>"
        f"<td>{html.escape(item['criterionId'])}</td>"
        f"<td>{html.escape(item['status'])}</td>"
        f"<td>{html.escape(', '.join(item['findingIds']))}</td>"
        f"<td>{html.escape(', '.join(str(source['taskId']) for source in item['sources']))}</td>"
        "</tr>"
        for item in report["results"]
    )
    return (
        "<!doctype html><meta charset=\"utf-8\"><title>Phase 08.02 acceptance</title>"
        f"<h1>Phase 08.02 acceptance: {report['gateStatus']}</h1>"
        f"<p>Decision: {report['acceptanceDecision']['status']}. "
        f"PASS {report['counts']['PASS']}; FAIL {report['counts']['FAIL']}; "
        f"BLOCKED {report['counts']['BLOCKED']}.</p>"
        f"<p>Open findings: {html.escape(', '.join(report['openFindings']))}.</p>"
        "<table><thead><tr><th>Requirement</th><th>Criterion</th><th>Status</th>"
        "<th>Findings</th><th>Source tasks</th></tr></thead><tbody>"
        f"{rows}</tbody></table>"
    )


def render_uat(report: dict[str, Any]) -> str:
    """Render the separate Phase 08.02 UAT and pending owner decision."""

    requirements = defaultdict(Counter)
    for result in report["results"]:
        requirements[result["requirement"]][result["status"]] += 1
    summary = "\n".join(
        f"| {requirement} | {counts['PASS']} | {counts['FAIL']} | {counts['BLOCKED']} |"
        for requirement, counts in sorted(requirements.items())
    )
    open_findings = report["openFindings"]
    tasks = "\n".join(
        f"| {item['taskId']} | {item['status']} | {item['component']} |"
        for item in report["taskTrace"]
    )
    return f"""# Phase 08.02 UAT — Real-Stack Requirement Acceptance

This document is separate from and does not rewrite the historical
`08_PHASE_UAT.md`. Task delivery status is not requirement acceptance.

## Scope and decision

The Task 280 manifest contains 12 scenarios and 91 mandatory criteria. Fresh
Task 281-285 execution produced **{report['counts']['PASS']} PASS**,
**{report['counts']['FAIL']} FAIL**, and **{report['counts']['BLOCKED']} BLOCKED**.
The Phase 08.02 gate is therefore **{report['gateStatus']}** and project-owner
acceptance remains unchecked. Supporting mocked/component checks are quality
evidence only; the aggregate below uses the mandatory real-stack/deployed
reports and preserves the strongest observed status (`FAIL > BLOCKED > PASS`).

Decision: ☐ Accepted  ☐ Rejected  ☐ Accepted with recorded deviations

Project owner: ____________________  Date: ____________________

## Requirement results

| Requirement | PASS | FAIL | BLOCKED |
|---|---:|---:|---:|
{summary}

Detailed criterion/source/evidence mappings and SHA-256 fingerprints are in
`08.02_PHASE_REPORT.json`; the readable table is in
`08.02_PHASE_REPORT.html`. Sanitized committed source reports, screenshots,
backend proof, traces, and log-sink evidence are under
`08.02_PHASE_EVIDENCE/`.

## Task trace

| Task | Delivery status | Scope |
|---:|---|---|
{tasks}

## Synchronized findings and deviations

Unresolved findings: {", ".join(open_findings) if open_findings else "none"}.

No deviation is accepted. Each unresolved record remains synchronized with
`docs/implementation/04_OPEN.md`, names an owner and retest condition, and is
retained in `docs/testing/phase08/finding-history.json`. The gate must not be
accepted while any mandatory criterion is non-pass unless the project owner
records owner, date, reason, retest condition, and expiry for every deviation.

## Automated verification

- Run each Task 281-284 isolated real-stack producer and Task 285 deployed-log
  verifier. Expected outcome is truthful per-criterion PASS/FAIL/BLOCKED, not
  an artificially green process.
- `python3 scripts/phase08_uat.py validate`
- `python3 scripts/phase08_acceptance.py validate`
- `python3 scripts/validate-task-list.py`
- `python3 scripts/validate-traceability.py`
- `python3 scripts/check.py --quick`
- `python3 scripts/check.py --output docs/implementation/implemented/08.02_PHASE_CHECK.html`
- `git diff --check`

## Project-owner acceptance checks

1. Review every non-pass criterion and its synchronized finding against the
   committed source report and evidence hash.
2. Rerun SW-REQ-084 against an approved deployed origin and independently
   queried centralized sink; console/browser output is insufficient.
3. Confirm SW-REQ-073 only from production-worker and backend erasure proof.
4. Exercise desktop/mobile, keyboard-only, light/dark, focus, heading,
   transition, reduced-motion, responsive, authorization, privacy, audit,
   idempotency, cache, provider, catalog, and regression paths represented by
   the 91-criterion report.
5. Accept only when all criteria pass, or record a dated accountable deviation
   with reason and retest/expiry for every remaining non-pass criterion.
"""


def build(input_path: Path = INPUT) -> dict[str, Any]:
    """Build committed evidence, JSON/HTML reports, and UAT from fresh runs."""

    source = validate_build_input(load_json(input_path))
    criteria, findings = load_controls()
    criteria_by_id = {item["id"]: item for item in criteria}
    historical = source["historicalUat"]
    historical_path = repo_path(historical["path"])
    if sha256(historical_path) != historical["sha256"]:
        raise UATError("historical Phase 08 UAT changed")
    reports: list[tuple[int, Path, dict[str, Any]]] = []
    copied_paths: list[Path] = []
    destination_names: set[str] = set()
    for item in source["reports"]:
        task_id = item["taskId"]
        original = repo_path(item["path"])
        validate_source_report(original, criteria_by_id, findings)
        destination = EVIDENCE_ROOT / original.parent.name
        if destination.name in destination_names:
            raise UATError("source report destinations must be unique")
        destination_names.add(destination.name)
        copy_report_tree(original, destination)
        copied_report = destination / "report.json"
        copied = validate_source_report(copied_report, criteria_by_id, findings)
        reports.append((task_id, copied_report, copied))
        copied_paths.extend(path for path in destination.rglob("*") if path.is_file())
    context_paths: list[Path] = []
    for item in source["supportingArtifacts"]:
        task_id = item["taskId"]
        original = repo_path(item["path"])
        sha256(original)
        parts = original.relative_to(ROOT).parts
        if "real-stack-e2e" not in parts or parts.index("real-stack-e2e") + 1 >= len(parts):
            raise UATError("supporting artifact path is malformed")
        run_id = parts[parts.index("real-stack-e2e") + 1]
        destination = EVIDENCE_ROOT / "run-context" / f"task-{task_id}" / run_id / original.name
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(original, destination)
        context_paths.append(destination)
    if {task_id for task_id, _, _ in reports} != set(range(281, 286)):
        raise UATError("source reports must cover Tasks 281-285")
    results = aggregate(reports, criteria, findings)
    verify_special_evidence(results)
    decision = source["acceptanceDecision"]
    validate_decision(decision, results)
    rows = task_rows()
    task_evidence_by_id: dict[int, list[Path]] = defaultdict(list)
    for item in source["taskEvidence"]:
        task_id = item["taskId"]
        path = repo_path(item["path"])
        sha256(path)
        task_evidence_by_id[task_id].append(path)
    if set(task_evidence_by_id) != set(range(276, 286)):
        raise UATError("supporting evidence must cover Tasks 276-285")
    task_trace = []
    for task_id, row in rows.items():
        evidence = evidence_hashes(task_evidence_by_id.get(task_id, []))
        task_trace.append({**row, "evidence": evidence})
    counts = Counter(item["status"] for item in results)
    gate_status = "FAIL" if counts["FAIL"] else "BLOCKED" if counts["BLOCKED"] else "PASS"
    closure_paths = []
    for finding in findings.values():
        if finding["status"] == "CLOSED":
            path = ROOT / finding["passingEvidence"]
            sha256(path)
            closure_paths.append(path)
    report = {
        "schema": SCHEMA,
        "generatedAtUtc": source["generatedAtUtc"],
        "gateStatus": gate_status,
        "counts": {status: counts[status] for status in ("PASS", "FAIL", "BLOCKED")},
        "acceptanceDecision": decision,
        "historicalUat": {"path": relative(historical_path), "sha256": sha256(historical_path)},
        "openFindings": sorted(
            finding["id"] for finding in findings.values() if finding["status"] != "CLOSED"
        ),
        "controls": evidence_hashes([MANIFEST, HISTORY, OPEN_DOC, TASK_LIST]),
        "closureEvidence": evidence_hashes(closure_paths),
        "runContextEvidence": evidence_hashes(context_paths),
        "sourceEvidence": evidence_hashes(copied_paths),
        "taskTrace": task_trace,
        "results": results,
    }
    html_text = render_html(report)
    uat_text = render_uat(report)
    report["renderedArtifacts"] = [
        {"path": relative(REPORT_HTML), "sha256": hashlib.sha256(html_text.encode()).hexdigest()},
        {"path": relative(UAT), "sha256": hashlib.sha256(uat_text.encode()).hexdigest()},
    ]
    REPORT_JSON.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    REPORT_HTML.write_text(html_text, encoding="utf-8")
    UAT.write_text(uat_text, encoding="utf-8")
    validate_final(report, uat_text)
    return report


def validate_hash_rows(rows: Any) -> None:
    """Validate every recorded evidence fingerprint against current content."""

    if not isinstance(rows, list):
        raise UATError("evidence hash manifest must be an array")
    for row in rows:
        if (
            not isinstance(row, dict)
            or set(row) != HASH_ROW_KEYS
            or not isinstance(row["path"], str)
            or not isinstance(row["sha256"], str)
        ):
            raise UATError("evidence hash row is malformed")
        path = repo_path(row["path"])
        if sha256(path) != row["sha256"]:
            raise UATError(f"stale evidence hash: {row['path']}")


def validate_final_schema(report: Any, uat_text: Any) -> dict[str, Any]:
    """Type-check the complete final-report tree before keyed or hashed access."""

    if not isinstance(report, dict) or set(report) != FINAL_REPORT_KEYS:
        raise UATError("final report schema is malformed")
    if not isinstance(uat_text, str):
        raise UATError("UAT content must be text")
    for key in ("schema", "generatedAtUtc", "gateStatus"):
        if not isinstance(report[key], str):
            raise UATError(f"final report {key} is malformed")
    if report["gateStatus"] not in STATUS_RANK:
        raise UATError("final gate status is invalid")
    counts = report["counts"]
    if (
        not isinstance(counts, dict)
        or set(counts) != set(STATUS_RANK)
        or any(not isinstance(value, int) or isinstance(value, bool) or value < 0 for value in counts.values())
    ):
        raise UATError("final report counts are malformed")
    validate_string_list(report["openFindings"], "final openFindings", acceptance.SAFE_ID_RE)
    results = report["results"]
    if not isinstance(results, list):
        raise UATError("final report results must be an array")
    for item in results:
        if not isinstance(item, dict) or set(item) != FINAL_RESULT_KEYS:
            raise UATError("final result schema is malformed")
        for key in ("criterionId", "requirement", "scenarioId", "kind", "text", "status"):
            if not isinstance(item[key], str):
                raise UATError(f"final result {key} is malformed")
        if item["status"] not in STATUS_RANK:
            raise UATError(f"{item['criterionId']}: final status is invalid")
        validate_string_list(
            item["rootCauseIds"],
            f"{item['criterionId']}: final rootCauseIds",
            acceptance.SAFE_ID_RE,
        )
        validate_string_list(
            item["findingIds"],
            f"{item['criterionId']}: final findingIds",
            acceptance.SAFE_ID_RE,
        )
        sources = item["sources"]
        if not isinstance(sources, list) or not sources:
            raise UATError(f"{item['criterionId']}: missing source evidence")
        for source in sources:
            if not isinstance(source, dict) or set(source) != FINAL_SOURCE_KEYS:
                raise UATError(f"{item['criterionId']}: final source evidence is malformed")
            if (
                not isinstance(source["taskId"], int)
                or isinstance(source["taskId"], bool)
                or source["taskId"] not in range(281, 286)
            ):
                raise UATError(f"{item['criterionId']}: final source taskId is malformed")
            for key in ("report", "status"):
                if not isinstance(source[key], str):
                    raise UATError(f"{item['criterionId']}: final source {key} is malformed")
            if source["status"] not in STATUS_RANK:
                raise UATError(f"{item['criterionId']}: source status is invalid")
            root = source["rootCauseId"]
            if root is not None and (
                not isinstance(root, str) or not acceptance.SAFE_ID_RE.fullmatch(root)
            ):
                raise UATError(f"{item['criterionId']}: final source rootCauseId is malformed")
            validate_string_list(
                source["requestIds"],
                f"{item['criterionId']}: final requestIds",
                acceptance.REQUEST_ID_RE,
            )
            validate_backend_evidence(source["backendEvidence"], item["criterionId"])
            evidence_items = source["evidence"]
            if not isinstance(evidence_items, list):
                raise UATError(f"{item['criterionId']}: final evidence must be an array")
            for evidence in evidence_items:
                if (
                    not isinstance(evidence, dict)
                    or set(evidence) != {"type", "path"}
                    or not isinstance(evidence["type"], str)
                    or not isinstance(evidence["path"], str)
                ):
                    raise UATError(
                        f"{item['criterionId']}: final evidence must contain string type and path"
                    )
    trace = report["taskTrace"]
    if not isinstance(trace, list):
        raise UATError("final task trace must be an array")
    for item in trace:
        if not isinstance(item, dict) or set(item) != TASK_TRACE_KEYS:
            raise UATError("final task trace row is malformed")
        if (
            not isinstance(item["taskId"], int)
            or isinstance(item["taskId"], bool)
            or item["taskId"] not in range(276, 287)
        ):
            raise UATError("final task trace identity is malformed")
        if any(
            not isinstance(item[key], str)
            for key in TASK_TRACE_KEYS - {"taskId", "evidence"}
        ):
            raise UATError(f"Task {item['taskId']} trace metadata is malformed")
        validate_hash_rows(item["evidence"])
    for section in (
        "controls",
        "closureEvidence",
        "renderedArtifacts",
        "runContextEvidence",
        "sourceEvidence",
    ):
        validate_hash_rows(report[section])
    historical = report["historicalUat"]
    if (
        not isinstance(historical, dict)
        or set(historical) != HASH_ROW_KEYS
        or not isinstance(historical["path"], str)
        or not isinstance(historical["sha256"], str)
    ):
        raise UATError("historical Phase 08 UAT reference is malformed")
    decision = report["acceptanceDecision"]
    if not isinstance(decision, dict) or set(decision) != DECISION_KEYS:
        raise UATError("acceptance decision schema is malformed")
    for key in ("status", "owner", "date"):
        if not isinstance(decision[key], str):
            raise UATError(f"acceptance decision {key} is malformed")
    deviations = decision["deviations"]
    if not isinstance(deviations, list):
        raise UATError("acceptance deviations must be an array")
    for deviation in deviations:
        if (
            not isinstance(deviation, dict)
            or set(deviation) != DEVIATION_KEYS
            or any(not isinstance(deviation[key], str) for key in DEVIATION_KEYS)
        ):
            raise UATError("accepted deviation is malformed")
    return report


def validate_final(report: dict[str, Any], uat_text: str) -> None:
    """Validate the committed final report, hashes, traceability, and UAT."""

    try:
        acceptance.assert_safe_value(report, "final report")
    except acceptance.ValidationError as error:
        raise UATError("final report contains prohibited sensitive metadata") from error
    report = validate_final_schema(report, uat_text)
    if report["schema"] != SCHEMA:
        raise UATError("unsupported final report schema")
    criteria, findings = load_controls()
    expected_open_findings = sorted(
        finding["id"] for finding in findings.values() if finding["status"] != "CLOSED"
    )
    if report.get("openFindings") != expected_open_findings:
        raise UATError("final report open findings are not synchronized with 04_OPEN.md")
    expected_finding_text = f"Unresolved findings: {', '.join(expected_open_findings)}."
    if expected_finding_text not in uat_text:
        raise UATError("UAT open findings are not synchronized with 04_OPEN.md")
    expected = {item["id"]: item for item in criteria}
    results = report["results"]
    if {item["criterionId"] for item in results} != set(expected):
        raise UATError("final report must cover every manifest criterion exactly once")
    if len(results) != len(expected):
        raise UATError("final report contains duplicate criteria")
    source_reports: set[Path] = set()
    for item in results:
        criterion = expected[item["criterionId"]]
        for key in ("requirement", "scenarioId", "kind", "text"):
            if item.get(key) != criterion.get(key):
                raise UATError(f"{item['criterionId']}: final report contradicts manifest")
        sources = item["sources"]
        identities = [(source["taskId"], source["report"]) for source in sources]
        if len(identities) != len(set(identities)):
            raise UATError(f"{item['criterionId']}: duplicate final source identity")
        source_statuses = [source.get("status") for source in sources]
        if any(status not in STATUS_RANK for status in source_statuses):
            raise UATError(f"{item['criterionId']}: source status is invalid")
        strongest = max(source_statuses, key=STATUS_RANK.__getitem__)
        if item.get("status") != strongest:
            raise UATError(f"{item['criterionId']}: aggregate status was weakened")
        for source in sources:
            source_report = repo_path(source["report"])
            if not source_report.is_file():
                raise UATError(f"{item['criterionId']}: source report is missing")
            source_reports.add(source_report)
            evidence_items = source["evidence"]
            for evidence in evidence_items:
                if not isinstance(evidence, dict) or set(evidence) != {"type", "path"}:
                    raise UATError(
                        f"{item['criterionId']}: final evidence must contain only type and path"
                    )
                if evidence["type"] not in criterion["evidenceTypes"]:
                    raise UATError(f"{item['criterionId']}: final evidence type is not allowed")
                evidence_path = acceptance.safe_relative_path(evidence["path"], "evidence path")
                sha256(source_report.parent / evidence_path)
        if item["status"] != "PASS":
            roots = set(item.get("rootCauseIds", []))
            open_findings = {
                finding["id"]
                for finding in findings.values()
                if finding["rootCauseId"] in roots and finding["status"] != "CLOSED"
            }
            if set(item.get("findingIds", [])) != open_findings or not open_findings:
                raise UATError(f"{item['criterionId']}: non-pass finding synchronization is stale")
    verify_special_evidence(results)
    criteria_by_id = {item["id"]: item for item in criteria}
    trusted_reports = canonical_source_reports(criteria_by_id, findings)
    trusted_results = aggregate(trusted_reports, criteria, findings)
    if results != trusted_results:
        raise UATError("final results or source linkage contradict trusted source evidence")
    trusted_paths = {path for _, path, _ in trusted_reports}
    if source_reports != trusted_paths:
        raise UATError("final source report set contradicts trusted source selection")
    counts = Counter(item["status"] for item in trusted_results)
    expected_counts = {status: counts[status] for status in ("PASS", "FAIL", "BLOCKED")}
    if report.get("counts") != expected_counts:
        raise UATError("final report counts contradict criterion results")
    expected_gate = "FAIL" if counts["FAIL"] else "BLOCKED" if counts["BLOCKED"] else "PASS"
    if report.get("gateStatus") != expected_gate:
        raise UATError("final gate status contradicts criterion results")
    validate_decision(report.get("acceptanceDecision", {}), trusted_results)
    verify_special_evidence(trusted_results)
    rows = task_rows()
    trace = report.get("taskTrace")
    if not isinstance(trace, list) or {item.get("taskId") for item in trace} != set(rows):
        raise UATError("final report must trace Tasks 276-286")
    for item in trace:
        if item["status"] != rows[item["taskId"]]["status"]:
            raise UATError(f"Task {item['taskId']} status is stale")
        validate_hash_rows(item.get("evidence"))
    for section in (
        "controls",
        "closureEvidence",
        "renderedArtifacts",
        "runContextEvidence",
        "sourceEvidence",
    ):
        validate_hash_rows(report.get(section))
    expected_closures = {
        finding["passingEvidence"]
        for finding in findings.values()
        if finding["status"] == "CLOSED"
    }
    actual_closures = {item["path"] for item in report["closureEvidence"]}
    if actual_closures != expected_closures:
        raise UATError("closed findings lack exact passing retest evidence")
    historical = report.get("historicalUat", {})
    if sha256(repo_path(historical.get("path"))) != historical.get("sha256"):
        raise UATError("historical Phase 08 UAT was rewritten")
    if (
        "Decision: ☐ Accepted  ☐ Rejected  ☐ Accepted with recorded deviations" not in uat_text
        or "Project owner: ____________________  Date: ____________________" not in uat_text
        or "Task delivery status is not requirement acceptance" not in uat_text
    ):
        raise UATError("UAT decision fields or delivery/acceptance distinction are missing")


def command_validate() -> int:
    """Validate the committed Task 286 report and UAT."""

    validate_final(load_json(REPORT_JSON), UAT.read_text(encoding="utf-8"))
    print("Phase 08.02 UAT valid: 91 criteria, Tasks 276-286, evidence hashes current")
    return 0


def main(argv: list[str] | None = None) -> int:
    """Run the Task 286 builder or validator with bounded diagnostics."""

    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("build", "validate"))
    parser.add_argument("--input", type=Path, default=INPUT)
    args = parser.parse_args(argv)
    try:
        if args.command == "build":
            report = build(args.input)
            print(
                "Phase 08.02 UAT built: "
                + ", ".join(f"{key}={value}" for key, value in report["counts"].items())
            )
            return 0
        return command_validate()
    except (OSError, UnicodeError, UATError, acceptance.ValidationError) as error:
        print(f"Phase 08.02 UAT invalid: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
