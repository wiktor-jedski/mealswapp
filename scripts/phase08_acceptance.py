#!/usr/bin/env python3
"""Build and validate Phase 08 requirement acceptance evidence.

Implements DESIGN-014 MetricsCollector for Task 280 acceptance result collection.
"""

from __future__ import annotations

import argparse
import html
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from collections import defaultdict
from dataclasses import dataclass
from pathlib import Path, PurePosixPath
from typing import Any, Iterable

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_MANIFEST = ROOT / "docs/testing/phase08/acceptance-manifest.json"
DEFAULT_FINDING_HISTORY = ROOT / "docs/testing/phase08/finding-history.json"
DEFAULT_OPEN_DOC = ROOT / "docs/implementation/04_OPEN.md"
DEFAULT_LOG_ROOT = ROOT / "logs/phase08-acceptance"
EVIDENCE_NAMESPACE = "evidence"
REQUIREMENTS = {
    "SW-REQ-019",
    "SW-REQ-032",
    "SW-REQ-033",
    "SW-REQ-043",
    "SW-REQ-054",
    "SW-REQ-055",
    "SW-REQ-056",
    "SW-REQ-057",
    "SW-REQ-072",
    "SW-REQ-073",
    "SW-REQ-084",
    "SW-REQ-090",
}
RESULT_STATUSES = {"PASS", "FAIL", "BLOCKED"}
FINDING_STATUSES = {"OPEN DEFECT", "OPEN BLOCKER", "CLOSED"}
EVIDENCE_TYPES = {"playwright", "backend", "database", "redis", "deployed-logs", "download"}
FINDINGS_START = "<!-- phase08-acceptance-findings:start -->"
FINDINGS_END = "<!-- phase08-acceptance-findings:end -->"
RUN_ID_RE = re.compile(r"\A[a-z0-9][a-z0-9-]{0,63}\Z")
SAFE_ID_RE = re.compile(r"\A[A-Z0-9][A-Z0-9._:-]{0,127}\Z")
REQUEST_ID_RE = re.compile(r"\A[a-zA-Z0-9][a-zA-Z0-9._:-]{0,127}\Z")
BACKEND_SUMMARY_RE = re.compile(
    r"\A[a-z][a-z0-9_]{0,63}(?:=[a-zA-Z0-9][a-zA-Z0-9._:-]{0,127})?\Z"
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
SENSITIVE_KEY_ALIASES = (
    "email",
    "credential",
    "password",
    "cookie",
    "csrf",
    "idempotency",
    "secret",
    "databaseurl",
    "providerpayload",
    "stack",
    "query",
    "itemid",
    "userid",
    "nametext",
)
SENSITIVE_VALUE_PATTERNS = (
    re.compile(r"(?i)\b(?:password|cookie|csrf|secret|authorization|bearer)\s*[:=]"),
    re.compile(r"(?i)\b(?:postgres(?:ql)?|redis)://"),
    re.compile(r"(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b"),
    re.compile(r"(?i)\b(?:traceback|goroutine \d+|panic:|stack trace)\b"),
)


class ValidationError(ValueError):
    """Raised when an acceptance artifact violates its contract."""


@dataclass(frozen=True)
class SourceEntry:
    """One verification step or acceptance criterion from req_tests.md."""

    requirement: str
    kind: str
    line: int
    text: str


def strict_json_loads(text: str, context: str) -> dict[str, Any]:
    """Decode one JSON object while rejecting duplicate keys."""

    def reject_duplicates(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
        result: dict[str, Any] = {}
        for key, value in pairs:
            if key in result:
                raise ValidationError(f"{context} contains a duplicate JSON key")
            result[key] = value
        return result

    try:
        value = json.loads(text, object_pairs_hook=reject_duplicates)
    except json.JSONDecodeError as error:
        raise ValidationError(f"{context} is not valid JSON") from error
    if not isinstance(value, dict):
        raise ValidationError(f"{context} must contain a JSON object")
    return value


def read_text(path: Path, context: str) -> str:
    """Read UTF-8 text without exposing filesystem or byte diagnostics."""

    try:
        return path.read_text(encoding="utf-8")
    except (OSError, UnicodeError) as error:
        raise ValidationError(f"cannot read {context}") from error


def load_json(path: Path) -> dict[str, Any]:
    """Load one JSON object with duplicate-key rejection."""

    return strict_json_loads(read_text(path, "JSON document"), "JSON document")


def req_test_entries(path: Path) -> list[SourceEntry]:
    """Extract the authoritative Phase 08 steps and acceptance criteria."""

    current_requirement = ""
    kind = ""
    entries: list[SourceEntry] = []
    for line_number, raw_line in enumerate(read_text(path, "requirement test source").splitlines(), 1):
        heading = re.match(r"^#{3,4} (SW-REQ-\d{3})\b", raw_line)
        if heading:
            current_requirement = heading.group(1)
            kind = "step"
            continue
        if current_requirement not in REQUIREMENTS:
            continue
        if raw_line in {"Dev check:", "### Dev verification"}:
            kind = "step"
            continue
        if raw_line == "Accept when:":
            kind = "criterion"
            continue
        if raw_line.startswith("- "):
            entries.append(SourceEntry(current_requirement, kind or "step", line_number, raw_line[2:]))
            continue
        if raw_line.startswith("Accept when ") and raw_line.endswith("."):
            entries.append(SourceEntry(current_requirement, "criterion", line_number, raw_line))
    return entries


def validate_manifest(manifest: dict[str, Any], req_tests: Path) -> list[dict[str, Any]]:
    """Validate stable one-to-one source coverage and return flattened criteria."""

    if manifest.get("schema") != "mealswapp.phase08-acceptance-manifest.v1":
        raise ValidationError("unsupported manifest schema")
    scenarios = manifest.get("scenarios")
    if not isinstance(scenarios, list) or not scenarios:
        raise ValidationError("manifest scenarios must be a non-empty array")
    scenario_ids: set[str] = set()
    criterion_ids: set[str] = set()
    mappings: dict[tuple[str, int, str, str], dict[str, Any]] = {}
    flattened: list[dict[str, Any]] = []
    for scenario in scenarios:
        if not isinstance(scenario, dict):
            raise ValidationError("each scenario must be an object")
        scenario_id = scenario.get("id")
        requirement = scenario.get("requirement")
        owner = scenario.get("owner")
        environment = scenario.get("requiredEnvironment")
        evidence_types = scenario.get("evidenceTypes")
        if not isinstance(scenario_id, str) or not SAFE_ID_RE.fullmatch(scenario_id):
            raise ValidationError("scenario has invalid stable ID")
        if scenario_id in scenario_ids:
            raise ValidationError(f"duplicate scenario ID: {scenario_id}")
        scenario_ids.add(scenario_id)
        if requirement not in REQUIREMENTS:
            raise ValidationError(f"{scenario_id}: invalid requirement mapping")
        if not isinstance(owner, str) or not owner.strip():
            raise ValidationError(f"{scenario_id}: missing scenario owner")
        if not isinstance(environment, str) or not environment.strip():
            raise ValidationError(f"{scenario_id}: missing required environment")
        if (
            not isinstance(evidence_types, list)
            or not evidence_types
            or any(item not in EVIDENCE_TYPES for item in evidence_types)
        ):
            raise ValidationError(f"{scenario_id}: invalid evidence types")
        criteria = scenario.get("criteria")
        if not isinstance(criteria, list) or not criteria:
            raise ValidationError(f"{scenario_id}: criteria must be non-empty")
        for criterion in criteria:
            if not isinstance(criterion, dict):
                raise ValidationError(f"{scenario_id}: criterion must be an object")
            criterion_id = criterion.get("id")
            source = criterion.get("source")
            source_kind = criterion.get("kind")
            text = criterion.get("text")
            if not isinstance(criterion_id, str) or not SAFE_ID_RE.fullmatch(criterion_id):
                raise ValidationError(f"{scenario_id}: invalid criterion ID")
            if criterion_id in criterion_ids:
                raise ValidationError(f"duplicate criterion ID: {criterion_id}")
            criterion_ids.add(criterion_id)
            if (
                not isinstance(source, dict)
                or source.get("file") != "req_tests.md"
                or not isinstance(source.get("line"), int)
            ):
                raise ValidationError(f"{criterion_id}: invalid source")
            if source_kind not in {"step", "criterion"} or not isinstance(text, str) or not text:
                raise ValidationError(f"{criterion_id}: invalid kind or text")
            mapping = (requirement, source["line"], source_kind, text)
            if mapping in mappings:
                raise ValidationError(f"duplicate source mapping: req_tests.md:{source['line']}")
            resolved = {
                **criterion,
                "scenarioId": scenario_id,
                "requirement": requirement,
                "owner": owner,
                "requiredEnvironment": environment,
                "evidenceTypes": evidence_types,
            }
            mappings[mapping] = resolved
            flattened.append(resolved)
    actual = req_test_entries(req_tests)
    expected_mappings = {
        (entry.requirement, entry.line, entry.kind, entry.text): entry for entry in actual
    }
    missing = expected_mappings.keys() - mappings.keys()
    orphaned = mappings.keys() - expected_mappings.keys()
    if missing:
        first = expected_mappings[sorted(missing)[0]]
        raise ValidationError(f"unmapped req_tests.md entry at line {first.line}: {first.text}")
    if orphaned:
        requirement, line, _, _ = sorted(orphaned)[0]
        raise ValidationError(f"orphan manifest mapping: {requirement} req_tests.md:{line}")
    if {item["requirement"] for item in flattened} != REQUIREMENTS:
        raise ValidationError("manifest does not cover the exact Task 280 requirement set")
    return sorted(flattened, key=lambda item: (item["requirement"], item["scenarioId"], item["id"]))


def extract_finding_ledger(text: str) -> dict[str, Any]:
    """Extract the machine-readable finding ledger from the Phase 08 document."""

    if text.count(FINDINGS_START) != 1 or text.count(FINDINGS_END) != 1:
        raise ValidationError("Phase 08 finding ledger markers must occur exactly once")
    payload = text.split(FINDINGS_START, 1)[1].split(FINDINGS_END, 1)[0].strip()
    if payload.startswith("```json") and payload.endswith("```"):
        payload = payload[7:-3].strip()
    ledger = strict_json_loads(payload, "Phase 08 finding ledger")
    if set(ledger) != {"schema", "findings"}:
        raise ValidationError("Phase 08 finding ledger contains unsupported fields")
    if ledger.get("schema") != "mealswapp.phase08-findings.v1":
        raise ValidationError("unsupported Phase 08 finding ledger schema")
    return ledger


def validate_finding_history(history: dict[str, Any]) -> set[str]:
    """Validate the mandatory append-only finding ID registry."""

    if set(history) != {"schema", "findingIds"}:
        raise ValidationError("finding history contains unsupported fields")
    if history.get("schema") != "mealswapp.phase08-finding-history.v1":
        raise ValidationError("unsupported finding history schema")
    finding_ids = history.get("findingIds")
    if (
        not isinstance(finding_ids, list)
        or any(not isinstance(item, str) or not SAFE_ID_RE.fullmatch(item) for item in finding_ids)
        or finding_ids != sorted(set(finding_ids))
    ):
        raise ValidationError("finding history IDs must be unique and sorted")
    return set(finding_ids)


def validate_findings(
    ledger: dict[str, Any],
    scenario_ids: set[str],
    historical_ids: set[str],
) -> dict[str, dict[str, Any]]:
    """Validate finding completeness, uniqueness, and retained history."""

    findings = ledger.get("findings")
    if not isinstance(findings, list):
        raise ValidationError("finding ledger findings must be an array")
    by_id: dict[str, dict[str, Any]] = {}
    roots: dict[str, str] = {}
    required_text = ("observed", "expected", "evidence", "owner", "retestCondition")
    for finding in findings:
        if not isinstance(finding, dict):
            raise ValidationError("finding must be an object")
        finding_id = finding.get("id")
        root = finding.get("rootCauseId")
        status = finding.get("status")
        if not isinstance(finding_id, str) or not SAFE_ID_RE.fullmatch(finding_id):
            raise ValidationError("finding has invalid ID")
        if finding_id in by_id:
            raise ValidationError(f"duplicate finding ID: {finding_id}")
        if not isinstance(root, str) or not SAFE_ID_RE.fullmatch(root):
            raise ValidationError(f"{finding_id}: invalid root cause ID")
        if root in roots:
            raise ValidationError(f"duplicate findings for root cause {root}: {roots[root]}, {finding_id}")
        if status not in FINDING_STATUSES:
            raise ValidationError(f"{finding_id}: invalid status")
        allowed_fields = {
            "id",
            "rootCauseId",
            "status",
            "requirements",
            "scenarios",
            "observed",
            "expected",
            "evidence",
            "owner",
            "retestCondition",
        }
        if status == "CLOSED":
            allowed_fields |= {"closedDate", "passingEvidence"}
        if set(finding) != allowed_fields:
            raise ValidationError(f"{finding_id}: finding contains unsupported or missing fields")
        requirements = finding.get("requirements")
        scenarios = finding.get("scenarios")
        if (
            not isinstance(requirements, list)
            or not requirements
            or any(item not in REQUIREMENTS for item in requirements)
        ):
            raise ValidationError(f"{finding_id}: missing or invalid requirements")
        if (
            not isinstance(scenarios, list)
            or not scenarios
            or any(item not in scenario_ids for item in scenarios)
        ):
            raise ValidationError(f"{finding_id}: missing or invalid scenarios")
        for field in required_text:
            if not isinstance(finding.get(field), str) or not finding[field].strip():
                raise ValidationError(f"{finding_id}: missing {field}")
        safe_relative_path(finding["evidence"], f"{finding_id} evidence")
        assert_safe_value(finding, f"finding {finding_id}")
        if status == "CLOSED":
            if not isinstance(finding.get("closedDate"), str) or not re.fullmatch(
                r"\d{4}-\d{2}-\d{2}", finding["closedDate"]
            ):
                raise ValidationError(f"{finding_id}: closed finding needs closedDate")
            if not isinstance(finding.get("passingEvidence"), str) or not finding["passingEvidence"].strip():
                raise ValidationError(f"{finding_id}: closed finding needs passingEvidence")
            safe_relative_path(finding["passingEvidence"], f"{finding_id} passing evidence")
        elif "closedDate" in finding or "passingEvidence" in finding:
            raise ValidationError(f"{finding_id}: unresolved finding cannot have closure fields")
        by_id[finding_id] = finding
        roots[root] = finding_id
    current_ids = set(by_id)
    deleted = historical_ids - current_ids
    if deleted:
        raise ValidationError(f"historical findings were deleted: {', '.join(sorted(deleted))}")
    unregistered = current_ids - historical_ids
    if unregistered:
        raise ValidationError(f"findings missing from history: {', '.join(sorted(unregistered))}")
    return by_id


def safe_relative_path(value: Any, field: str) -> str:
    """Validate a report-local evidence link."""

    if not isinstance(value, str) or not value:
        raise ValidationError(f"{field} must be a non-empty relative path")
    if any(ord(character) < 32 or ord(character) == 127 for character in value):
        raise ValidationError(f"{field} must be a safe relative path")
    if any(character in value for character in ("\\", ":", "%", "?", "#")):
        raise ValidationError(f"{field} must be a safe relative path")
    path = PurePosixPath(value)
    if (
        path.is_absolute()
        or not path.parts
        or any(part in {"", ".", ".."} or part.startswith(".") for part in path.parts)
    ):
        raise ValidationError(f"{field} must be a safe relative path")
    return value


def assert_safe_value(value: Any, context: str = "result", key: str = "") -> None:
    """Reject sensitive fields and recognizable secret/diagnostic values."""

    normalized_key = re.sub(r"[^a-z0-9]", "", key.casefold())
    if normalized_key and any(alias in normalized_key for alias in SENSITIVE_KEY_ALIASES):
        raise ValidationError(f"{context} contains prohibited field {key}")
    if isinstance(value, dict):
        for nested_key, nested_value in value.items():
            assert_safe_value(nested_value, context, str(nested_key))
    elif isinstance(value, list):
        for nested_value in value:
            assert_safe_value(nested_value, context)
    elif isinstance(value, str):
        for pattern in SENSITIVE_VALUE_PATTERNS:
            if pattern.search(value):
                raise ValidationError(f"{context} contains prohibited sensitive value")


def normalize_evidence(items: Any, allowed_types: Iterable[str], criterion_id: str) -> list[dict[str, str]]:
    """Validate and deterministically order evidence references."""

    if not isinstance(items, list):
        raise ValidationError(f"{criterion_id}: evidence must be an array")
    normalized: list[dict[str, str]] = []
    for item in items:
        if not isinstance(item, dict) or set(item) != {"type", "path"}:
            raise ValidationError(f"{criterion_id}: evidence must contain only type and path")
        evidence_type = item["type"]
        if evidence_type not in allowed_types:
            raise ValidationError(f"{criterion_id}: evidence type {evidence_type} is not allowed")
        normalized.append({"type": evidence_type, "path": safe_relative_path(item["path"], "evidence path")})
    return sorted(normalized, key=lambda item: (item["type"], item["path"]))


def normalize_results(
    raw_results: list[dict[str, Any]],
    criteria: list[dict[str, Any]],
) -> list[dict[str, Any]]:
    """Validate criterion results and preserve one deterministic result per manifest entry."""

    expected = {criterion["id"]: criterion for criterion in criteria}
    provided: dict[str, dict[str, Any]] = {}
    allowed_keys = {
        "criterionId",
        "status",
        "rootCauseId",
        "resolvedRootCauseIds",
        "requestIds",
        "evidence",
        "backendEvidence",
    }
    for result in raw_results:
        if not isinstance(result, dict) or set(result) - allowed_keys:
            raise ValidationError("criterion result contains unsupported fields")
        criterion_id = result.get("criterionId")
        if criterion_id not in expected:
            raise ValidationError(f"orphan result: {criterion_id}")
        if criterion_id in provided:
            raise ValidationError(f"duplicate criterion result: {criterion_id}")
        status = result.get("status")
        if status not in RESULT_STATUSES:
            raise ValidationError(f"{criterion_id}: invalid result status")
        root = result.get("rootCauseId")
        if status == "PASS" and root is not None:
            raise ValidationError(f"{criterion_id}: PASS cannot have an unresolved root cause")
        if status != "PASS" and (not isinstance(root, str) or not SAFE_ID_RE.fullmatch(root)):
            raise ValidationError(f"{criterion_id}: non-pass result requires rootCauseId")
        resolved = result.get("resolvedRootCauseIds", [])
        if (
            not isinstance(resolved, list)
            or any(not isinstance(item, str) or not SAFE_ID_RE.fullmatch(item) for item in resolved)
        ):
            raise ValidationError(f"{criterion_id}: invalid resolved root causes")
        request_ids = result.get("requestIds", [])
        if (
            not isinstance(request_ids, list)
            or any(not isinstance(item, str) or not REQUEST_ID_RE.fullmatch(item) for item in request_ids)
        ):
            raise ValidationError(f"{criterion_id}: invalid request IDs")
        backend = result.get("backendEvidence", [])
        if (
            not isinstance(backend, list)
            or any(
                not isinstance(item, str)
                or not BACKEND_SUMMARY_RE.fullmatch(item)
                or item.split("=", 1)[0] not in BACKEND_SUMMARY_KEYS
                for item in backend
            )
        ):
            raise ValidationError(f"{criterion_id}: backendEvidence must contain sanitized summaries")
        assert_safe_value(backend, f"{criterion_id} backend evidence")
        criterion = expected[criterion_id]
        provided[criterion_id] = {
            "criterionId": criterion_id,
            "status": status,
            "rootCauseId": root,
            "resolvedRootCauseIds": sorted(set(resolved)),
            "requestIds": sorted(set(request_ids)),
            "evidence": normalize_evidence(result.get("evidence", []), criterion["evidenceTypes"], criterion_id),
            "backendEvidence": sorted(set(backend)),
        }
    missing = expected.keys() - provided.keys()
    if missing:
        raise ValidationError(f"missing criterion results: {', '.join(sorted(missing))}")
    return [provided[criterion["id"]] for criterion in criteria]


def synchronize_results(
    results: list[dict[str, Any]],
    criteria: list[dict[str, Any]],
    findings: dict[str, dict[str, Any]],
) -> list[dict[str, Any]]:
    """Link every result root cause to a complete finding and enforce closure."""

    by_root = {finding["rootCauseId"]: finding for finding in findings.values()}
    criterion_map = {criterion["id"]: criterion for criterion in criteria}
    linked: list[dict[str, Any]] = []
    for result in results:
        criterion = criterion_map[result["criterionId"]]
        finding_ids: list[str] = []
        root = result["rootCauseId"]
        if root is not None:
            finding = by_root.get(root)
            if finding is None or (
                finding["status"] == "CLOSED"
                and (
                    criterion["requirement"] not in finding["requirements"]
                    or criterion["scenarioId"] not in finding["scenarios"]
                )
            ):
                raise ValidationError(f"{result['criterionId']}: non-pass root cause has no unresolved finding")
            if (
                criterion["requirement"] not in finding["requirements"]
                or criterion["scenarioId"] not in finding["scenarios"]
            ):
                raise ValidationError(f"{result['criterionId']}: finding mapping does not cover result")
            finding_ids.append(finding["id"])
        for resolved_root in result["resolvedRootCauseIds"]:
            finding = by_root.get(resolved_root)
            if finding is None or finding["status"] != "CLOSED":
                raise ValidationError(f"{result['criterionId']}: passing retest did not close finding")
            finding_ids.append(finding["id"])
        linked.append({**criterion, **result, "findingIds": sorted(set(finding_ids))})
    return linked


def overall_status(results: Iterable[dict[str, Any]]) -> tuple[str, int]:
    """Return aggregate status and Task 280 exit code."""

    statuses = {result["status"] for result in results}
    if "FAIL" in statuses:
        return "FAIL", 1
    if "BLOCKED" in statuses:
        return "BLOCKED", 2
    return "PASS", 0


def evidence_paths(linked: Iterable[dict[str, Any]]) -> list[str]:
    """Return each unique report evidence path in deterministic order."""

    return sorted(
        {
            evidence["path"]
            for result in linked
            for evidence in result["evidence"]
        }
    )


def namespace_evidence(linked: Iterable[dict[str, Any]]) -> list[dict[str, Any]]:
    """Project evidence links into the protected report evidence namespace."""

    return [
        {
            **result,
            "evidence": [
                {**evidence, "path": f"{EVIDENCE_NAMESPACE}/{evidence['path']}"}
                for evidence in result["evidence"]
            ],
        }
        for result in linked
    ]


def copy_evidence(paths: Iterable[str], evidence_root: Path, staging_dir: Path) -> None:
    """Copy validated regular evidence files into report staging."""

    if evidence_root.is_symlink() or not evidence_root.is_dir():
        raise ValidationError("evidence root must be an existing regular directory")
    for relative in paths:
        safe_relative_path(relative, "evidence path")
        source = evidence_root / relative
        parents = [source, *source.parents]
        try:
            root_index = parents.index(evidence_root)
        except ValueError as error:
            raise ValidationError("evidence path escaped its root") from error
        if any(path.is_symlink() for path in parents[:root_index]):
            raise ValidationError("evidence path cannot contain symlinks")
        if not source.is_file():
            raise ValidationError("linked evidence file is missing")
        destination = staging_dir / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        try:
            shutil.copyfile(source, destination)
        except OSError as error:
            raise ValidationError("cannot preserve linked evidence") from error


def write_report(
    run_id: str,
    linked: list[dict[str, Any]],
    output_root: Path,
    *,
    evidence_root: Path | None = None,
    producer_failures: Iterable[str] = (),
) -> tuple[Path, int]:
    """Atomically finalize deterministic JSON and HTML reports."""

    if not RUN_ID_RE.fullmatch(run_id):
        raise ValidationError("run ID must be lowercase alphanumeric/hyphen and at most 64 characters")
    failures = sorted(set(producer_failures))
    if any(not SAFE_ID_RE.fullmatch(failure) for failure in failures):
        raise ValidationError("producer failure has an invalid safe code")
    linked_evidence = evidence_paths(linked)
    published_linked = namespace_evidence(linked)
    if linked_evidence and evidence_root is None:
        raise ValidationError("linked evidence requires an evidence root")
    run_dir = output_root / run_id
    staging_dir = output_root / f".{run_id}.tmp"
    try:
        output_root.mkdir(parents=True, exist_ok=True)
        if run_dir.exists() or staging_dir.exists():
            raise ValidationError("run output already exists")
        staging_dir.mkdir()
    except OSError as error:
        raise ValidationError("cannot create report output") from error
    status, exit_code = overall_status(published_linked)
    if failures:
        status, exit_code = "FAIL", 1
    grouped: dict[str, list[str]] = defaultdict(list)
    for result in published_linked:
        if result["rootCauseId"]:
            grouped[result["rootCauseId"]].append(result["criterionId"])
    report = {
        "schema": "mealswapp.phase08-acceptance-report.v1",
        "runId": run_id,
        "status": status,
        "exitCode": exit_code,
        "producerFailures": failures,
        "rootCauses": [
            {"id": root, "criteria": sorted(ids)} for root, ids in sorted(grouped.items())
        ],
        "results": published_linked,
    }
    assert_safe_value(report, "final report")
    json_text = json.dumps(report, indent=2, ensure_ascii=True, sort_keys=True) + "\n"
    rows = []
    for result in published_linked:
        links = " ".join(
            f'<a href="{html.escape(item["path"], quote=True)}">{html.escape(item["type"])}</a>'
            for item in result["evidence"]
        )
        rows.append(
            "<tr>"
            f"<td>{html.escape(result['requirement'])}</td>"
            f"<td>{html.escape(result['scenarioId'])}</td>"
            f"<td>{html.escape(result['criterionId'])}</td>"
            f"<td>{html.escape(result['status'])}</td>"
            f"<td>{html.escape(', '.join(result['requestIds']))}</td>"
            f"<td>{links}</td>"
            f"<td>{html.escape('; '.join(result['backendEvidence']))}</td>"
            f"<td>{html.escape(', '.join(result['findingIds']))}</td>"
            "</tr>"
        )
    html_text = (
        "<!doctype html><meta charset=\"utf-8\"><title>Phase 08 acceptance</title>"
        f"<h1>Phase 08 acceptance: {html.escape(status)}</h1>"
        "<table><thead><tr><th>Requirement</th><th>Scenario</th><th>Criterion</th>"
        "<th>Status</th><th>Request IDs</th><th>Evidence</th><th>Backend evidence</th>"
        f"<th>Findings</th></tr></thead><tbody>{''.join(rows)}</tbody></table>"
    )
    try:
        if evidence_root is not None:
            copy_evidence(
                linked_evidence,
                evidence_root,
                staging_dir / EVIDENCE_NAMESPACE,
            )
        for name, content in (("report.json", json_text), ("report.html", html_text)):
            (staging_dir / name).write_text(content, encoding="utf-8")
        staging_dir.replace(run_dir)
    except ValidationError:
        shutil.rmtree(staging_dir, ignore_errors=True)
        raise
    except (OSError, UnicodeError) as error:
        shutil.rmtree(staging_dir, ignore_errors=True)
        raise ValidationError("cannot finalize report artifacts") from error
    return run_dir, exit_code


def load_result_files(paths: Iterable[Path]) -> list[dict[str, Any]]:
    """Load result arrays while retaining results from every producer."""

    results: list[dict[str, Any]] = []
    for path in paths:
        payload = load_json(path)
        values = payload.get("results")
        if not isinstance(values, list):
            raise ValidationError(f"{path}: results must be an array")
        results.extend(values)
    return results


def execute_producers(plan_path: Path, result_dir: Path) -> tuple[list[Path], list[str]]:
    """Execute every producer even after failures and return published result paths."""

    plan = load_json(plan_path)
    producers = plan.get("producers")
    if not isinstance(producers, list) or not producers:
        raise ValidationError("producer plan must contain a non-empty producers array")
    result_dir.mkdir(parents=True, exist_ok=True)
    result_paths: list[Path] = []
    errors: list[str] = []
    for producer in producers:
        if not isinstance(producer, dict) or set(producer) != {"id", "command", "result"}:
            raise ValidationError("producer entries require only id, command, and result")
        producer_id = producer["id"]
        command = producer["command"]
        result_name = safe_relative_path(producer["result"], "producer result")
        if not isinstance(producer_id, str) or not SAFE_ID_RE.fullmatch(producer_id):
            raise ValidationError("producer has invalid ID")
        if not isinstance(command, list) or not command or any(not isinstance(arg, str) for arg in command):
            raise ValidationError(f"{producer_id}: command must be a non-empty argv array")
        result_path = result_dir / result_name
        try:
            completed = subprocess.run(
                command,
                cwd=ROOT,
                check=False,
                env={**os.environ, "PHASE08_ACCEPTANCE_RESULT_DIR": str(result_dir)},
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
            if completed.returncode:
                errors.append(f"{producer_id}:EXIT-{completed.returncode}")
            if result_path.is_file():
                result_paths.append(result_path)
            else:
                errors.append(f"{producer_id}:MISSING-RESULT")
        except OSError:
            errors.append(f"{producer_id}:SPAWN-FAILED")
    return result_paths, errors


def acceptance_context(args: argparse.Namespace) -> tuple[dict[str, Any], list[dict[str, Any]], dict[str, dict[str, Any]]]:
    """Load and validate the manifest, mandatory history, and current findings."""

    manifest = load_json(args.manifest)
    criteria = validate_manifest(manifest, args.req_tests)
    history_ids = validate_finding_history(load_json(args.finding_history))
    ledger = extract_finding_ledger(read_text(args.open_doc, "Phase 08 findings document"))
    findings = validate_findings(ledger, {item["scenarioId"] for item in criteria}, history_ids)
    return manifest, criteria, findings


def command_validate(args: argparse.Namespace) -> int:
    """Validate manifest and finding ledger without producing a report."""

    manifest, criteria, _ = acceptance_context(args)
    print(f"phase08 acceptance contracts valid: scenarios={len(manifest['scenarios'])} criteria={len(criteria)}")
    return 0


def command_report(args: argparse.Namespace) -> int:
    """Validate results/findings and emit final artifacts."""

    _, criteria, findings = acceptance_context(args)
    if args.requirement:
        criteria = [
            criterion
            for criterion in criteria
            if criterion["requirement"] == args.requirement
        ]
    raw_results = load_result_files(args.result)
    results = normalize_results(raw_results, criteria)
    linked = synchronize_results(results, criteria, findings)
    run_dir, exit_code = write_report(
        args.run_id,
        linked,
        args.output_root,
        evidence_root=args.evidence_root,
    )
    print(f"phase08 acceptance report: {run_dir.relative_to(ROOT) if run_dir.is_relative_to(ROOT) else run_dir}")
    return exit_code


def command_run(args: argparse.Namespace) -> int:
    """Run all producers, then finalize available complete evidence."""

    with tempfile.TemporaryDirectory(prefix="mealswapp-phase08-results-") as directory:
        paths, producer_errors = execute_producers(args.plan, Path(directory))
        if producer_errors:
            print("producer diagnostics: " + ", ".join(producer_errors), file=sys.stderr)
        _, criteria, findings = acceptance_context(args)
        raw_results = load_result_files(paths)
        results = normalize_results(raw_results, criteria)
        linked = synchronize_results(results, criteria, findings)
        run_dir, exit_code = write_report(
            args.run_id,
            linked,
            args.output_root,
            evidence_root=Path(directory),
            producer_failures=producer_errors,
        )
        print(
            f"phase08 acceptance report: "
            f"{run_dir.relative_to(ROOT) if run_dir.is_relative_to(ROOT) else run_dir}"
        )
        return exit_code


def build_parser() -> argparse.ArgumentParser:
    """Build the command-line parser."""

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest", type=Path, default=DEFAULT_MANIFEST)
    parser.add_argument("--req-tests", type=Path, default=ROOT / "req_tests.md")
    parser.add_argument("--open-doc", type=Path, default=DEFAULT_OPEN_DOC)
    parser.add_argument("--finding-history", type=Path, default=DEFAULT_FINDING_HISTORY)
    subparsers = parser.add_subparsers(dest="command", required=True)
    validate = subparsers.add_parser("validate")
    validate.set_defaults(handler=command_validate)
    for name, handler in (("report", command_report), ("run", command_run)):
        subparser = subparsers.add_parser(name)
        subparser.add_argument("--run-id", required=True)
        subparser.add_argument("--output-root", type=Path, default=DEFAULT_LOG_ROOT)
        if name == "report":
            subparser.add_argument("--result", type=Path, action="append", required=True)
            subparser.add_argument("--evidence-root", type=Path)
            subparser.add_argument(
                "--requirement",
                choices=sorted(REQUIREMENTS),
                help="finalize one requirement-specific producer result",
            )
        else:
            subparser.add_argument("--plan", type=Path, required=True)
        subparser.set_defaults(handler=handler)
    return parser


def main(argv: list[str] | None = None) -> int:
    """Run the Task 280 acceptance contract command."""

    args = build_parser().parse_args(argv)
    try:
        return args.handler(args)
    except ValidationError as error:
        print(f"phase08 acceptance validation failed: {error}", file=sys.stderr)
        return 1
    except (OSError, UnicodeError, subprocess.SubprocessError):
        print("phase08 acceptance operation failed safely", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
