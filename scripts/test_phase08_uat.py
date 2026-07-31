#!/usr/bin/env python3
"""Adversarial tests for the Task 286 acceptance gate."""

from __future__ import annotations

import copy
import contextlib
import io
import json
import shutil
import sys
import tempfile
import unittest
from collections import Counter
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
import phase08_uat as uat


class Phase08UATTests(unittest.TestCase):
    """Prove the committed gate fails closed for every Task 286 integrity rule."""

    @classmethod
    def setUpClass(cls) -> None:
        cls.report = json.loads(uat.REPORT_JSON.read_text(encoding="utf-8"))
        cls.uat_text = uat.UAT.read_text(encoding="utf-8")

    def validate(self, report: dict | None = None, text: str | None = None) -> None:
        uat.validate_final(
            copy.deepcopy(report if report is not None else self.report),
            text if text is not None else self.uat_text,
        )

    @staticmethod
    def recount(report: dict) -> None:
        counts = Counter(item["status"] for item in report["results"])
        report["counts"] = {status: counts[status] for status in ("PASS", "FAIL", "BLOCKED")}
        report["gateStatus"] = "FAIL" if counts["FAIL"] else "BLOCKED" if counts["BLOCKED"] else "PASS"

    def test_committed_report_is_current_and_complete(self) -> None:
        self.validate()
        self.assertEqual(91, len(self.report["results"]))
        self.assertEqual({"PASS": 72, "FAIL": 8, "BLOCKED": 11}, self.report["counts"])
        self.assertEqual(11, len(self.report["openFindings"]))
        self.assertEqual(
            "PASSED",
            next(item["status"] for item in self.report["taskTrace"] if item["taskId"] == 286),
        )

    def test_missing_criterion_is_rejected(self) -> None:
        report = copy.deepcopy(self.report)
        report["results"].pop()
        with self.assertRaisesRegex(uat.UATError, "every manifest criterion"):
            self.validate(report)

    def test_weakened_or_contradictory_status_is_rejected(self) -> None:
        report = copy.deepcopy(self.report)
        result = next(item for item in report["results"] if item["status"] == "FAIL")
        result["status"] = "PASS"
        self.recount(report)
        with self.assertRaisesRegex(uat.UATError, "status was weakened"):
            self.validate(report)

    def test_malformed_nested_source_types_are_rejected(self) -> None:
        mutations = {
            "taskId": "281",
            "requestIds": "request-id",
            "backendEvidence": {"http_status": 200},
        }
        for field, value in mutations.items():
            with self.subTest(field=field):
                report = copy.deepcopy(self.report)
                report["results"][0]["sources"][0][field] = value
                with self.assertRaises(uat.UATError):
                    self.validate(report)

    def test_list_valued_hash_operands_are_bounded_validation_errors(self) -> None:
        mutations = (
            (("results", 0, "criterionId"), []),
            (("results", 0, "sources", 0, "report"), []),
            (("results", 0, "sources", 0, "status"), []),
        )
        for path, value in mutations:
            with self.subTest(path=path):
                report = copy.deepcopy(self.report)
                target = report
                for part in path[:-1]:
                    target = target[part]
                target[path[-1]] = value
                with self.assertRaises(uat.UATError):
                    self.validate(report)

    def test_all_nested_operation_fields_are_type_guarded(self) -> None:
        mutations = (
            (("counts", "PASS"), []),
            (("openFindings", 0), []),
            (("results", 0, "requirement"), []),
            (("results", 0, "scenarioId"), {}),
            (("results", 0, "kind"), []),
            (("results", 0, "text"), {}),
            (("results", 0, "status"), []),
            (("results", 0, "rootCauseIds"), {}),
            (("results", 0, "rootCauseIds"), [[]]),
            (("results", 0, "findingIds"), {}),
            (("results", 0, "findingIds"), [[]]),
            (("results", 0, "sources"), {}),
            (("results", 0, "sources", 0, "taskId"), []),
            (("results", 0, "sources", 0, "rootCauseId"), []),
            (("results", 0, "sources", 0, "requestIds"), {}),
            (("results", 0, "sources", 0, "backendEvidence"), {}),
            (("results", 0, "sources", 0, "evidence"), {}),
            (("results", 0, "sources", 0, "evidence", 0, "type"), []),
            (("results", 0, "sources", 0, "evidence", 0, "path"), []),
            (("taskTrace", 0, "taskId"), []),
            (("taskTrace", 0, "status"), []),
            (("taskTrace", 0, "component"), {}),
            (("taskTrace", 0, "evidence"), {}),
            (("taskTrace", 0, "evidence", 0, "path"), []),
            (("taskTrace", 0, "evidence", 0, "sha256"), []),
            (("controls", 0, "path"), []),
            (("closureEvidence", 0, "sha256"), []),
            (("historicalUat", "path"), []),
            (("historicalUat", "sha256"), {}),
            (("acceptanceDecision", "status"), []),
            (("acceptanceDecision", "owner"), {}),
            (("acceptanceDecision", "date"), []),
            (("acceptanceDecision", "deviations"), {}),
        )
        for path, value in mutations:
            with self.subTest(path=path):
                report = copy.deepcopy(self.report)
                target = report
                for part in path[:-1]:
                    target = target[part]
                target[path[-1]] = value
                with self.assertRaises(uat.UATError):
                    self.validate(report)

        deviation = {
            "criterionId": [],
            "owner": "owner",
            "date": "2026-07-29",
            "reason": "reason",
            "retestCondition": "retest",
            "expiry": "2026-08-29",
        }
        report = copy.deepcopy(self.report)
        report["acceptanceDecision"]["deviations"] = [deviation]
        with self.assertRaises(uat.UATError):
            self.validate(report)

    def test_cli_rejects_malformed_nested_values_without_traceback(self) -> None:
        report = copy.deepcopy(self.report)
        report["results"][0]["criterionId"] = []
        with tempfile.TemporaryDirectory(dir=uat.ROOT / "logs") as temporary:
            report_path = Path(temporary) / "report.json"
            uat_path = Path(temporary) / "uat.md"
            report_path.write_text(json.dumps(report), encoding="utf-8")
            uat_path.write_text(self.uat_text, encoding="utf-8")
            original_report, original_uat = uat.REPORT_JSON, uat.UAT
            uat.REPORT_JSON, uat.UAT = report_path, uat_path
            stderr = io.StringIO()
            try:
                with contextlib.redirect_stderr(stderr):
                    exit_code = uat.main(["validate"])
            finally:
                uat.REPORT_JSON, uat.UAT = original_report, original_uat
        self.assertEqual(1, exit_code)
        self.assertIn("Phase 08.02 UAT invalid:", stderr.getvalue())
        self.assertNotIn("Traceback", stderr.getvalue())

    def test_duplicate_sources_are_rejected(self) -> None:
        report = copy.deepcopy(self.report)
        report["results"][0]["sources"].append(
            copy.deepcopy(report["results"][0]["sources"][0])
        )
        with self.assertRaisesRegex(uat.UATError, "duplicate final source identity"):
            self.validate(report)

    def test_coordinated_fail_source_and_result_weakening_is_rejected(self) -> None:
        report = copy.deepcopy(self.report)
        result = next(item for item in report["results"] if item["status"] == "FAIL")
        for source in result["sources"]:
            source["status"] = "PASS"
            source["rootCauseId"] = None
        result["status"] = "PASS"
        result["rootCauseIds"] = []
        result["findingIds"] = []
        self.recount(report)
        with self.assertRaisesRegex(uat.UATError, "trusted source evidence"):
            self.validate(report)

    def test_coordinated_blocked_source_and_result_weakening_is_rejected(self) -> None:
        report = copy.deepcopy(self.report)
        result = next(item for item in report["results"] if item["status"] == "BLOCKED")
        for source in result["sources"]:
            source["status"] = "PASS"
            source["rootCauseId"] = None
        result["status"] = "PASS"
        result["rootCauseIds"] = []
        result["findingIds"] = []
        self.recount(report)
        with self.assertRaisesRegex(uat.UATError, "trusted source evidence"):
            self.validate(report)

    def test_nonpass_without_synchronized_finding_is_rejected(self) -> None:
        report = copy.deepcopy(self.report)
        result = next(item for item in report["results"] if item["status"] != "PASS")
        result["findingIds"] = []
        with self.assertRaisesRegex(uat.UATError, "finding synchronization"):
            self.validate(report)

    def test_closed_finding_without_exact_retest_evidence_is_rejected(self) -> None:
        report = copy.deepcopy(self.report)
        report["closureEvidence"].pop()
        with self.assertRaisesRegex(uat.UATError, "exact passing retest evidence"):
            self.validate(report)

    def test_orphan_source_evidence_is_rejected(self) -> None:
        original = Path(self.report["results"][0]["sources"][0]["report"]).parent
        with tempfile.TemporaryDirectory(dir=uat.ROOT / "logs") as temporary:
            copied = Path(temporary) / "source"
            shutil.copytree(uat.ROOT / original, copied)
            (copied / "orphan.json").write_text("{}\n", encoding="utf-8")
            criteria, findings = uat.load_controls()
            with self.assertRaisesRegex(uat.UATError, "orphan evidence"):
                uat.validate_source_report(
                    copied / "report.json",
                    {item["id"]: item for item in criteria},
                    findings,
                )

    def test_source_result_hash_operands_are_bounded_validation_errors(self) -> None:
        original = Path(self.report["results"][0]["sources"][0]["report"]).parent
        criteria, findings = uat.load_controls()
        criteria_by_id = {item["id"]: item for item in criteria}
        for field in ("criterionId", "status"):
            with self.subTest(field=field), tempfile.TemporaryDirectory(
                dir=uat.ROOT / "logs"
            ) as temporary:
                copied = Path(temporary) / "source"
                shutil.copytree(uat.ROOT / original, copied)
                report_path = copied / "report.json"
                report = json.loads(report_path.read_text(encoding="utf-8"))
                report["results"][0][field] = []
                report_path.write_text(json.dumps(report), encoding="utf-8")
                with self.assertRaises(uat.UATError):
                    uat.validate_source_report(report_path, criteria_by_id, findings)

    def test_malformed_build_input_is_bounded_before_operations(self) -> None:
        source = json.loads(uat.INPUT.read_text(encoding="utf-8"))
        mutations = (
            (("historicalUat",), []),
            (("reports",), [[]]),
            (("reports", 0, "path"), []),
            (("acceptanceDecision", "status"), []),
            (
                ("acceptanceDecision",),
                {
                    "status": "ACCEPTED_WITH_DEVIATIONS",
                    "owner": "owner",
                    "date": "2026-07-29",
                    "deviations": [
                        {
                            "criterionId": [],
                            "owner": "owner",
                            "date": "2026-07-29",
                            "reason": "reason",
                            "retestCondition": "retest",
                            "expiry": "2026-08-29",
                        }
                    ],
                },
            ),
        )
        for path, value in mutations:
            with self.subTest(path=path):
                malformed = copy.deepcopy(source)
                target = malformed
                for part in path[:-1]:
                    target = target[part]
                target[path[-1]] = value
                with tempfile.TemporaryDirectory(dir=uat.ROOT / "logs") as temporary:
                    input_path = Path(temporary) / "input.json"
                    input_path.write_text(json.dumps(malformed), encoding="utf-8")
                    with self.assertRaises(uat.UATError):
                        uat.build(input_path)

    def test_build_cli_rejects_malformed_input_and_source_without_traceback(self) -> None:
        source = json.loads(uat.INPUT.read_text(encoding="utf-8"))
        cases = (("historicalUat", []), ("acceptanceDecision", {"status": []}))
        for field, value in cases:
            with self.subTest(field=field), tempfile.TemporaryDirectory(
                dir=uat.ROOT / "logs"
            ) as temporary:
                malformed = copy.deepcopy(source)
                malformed[field] = value
                input_path = Path(temporary) / "input.json"
                input_path.write_text(json.dumps(malformed), encoding="utf-8")
                stderr = io.StringIO()
                with contextlib.redirect_stderr(stderr):
                    exit_code = uat.main(["build", "--input", str(input_path)])
                self.assertEqual(1, exit_code)
                self.assertIn("Phase 08.02 UAT invalid:", stderr.getvalue())
                self.assertNotIn("Traceback", stderr.getvalue())

        original = Path(self.report["results"][0]["sources"][0]["report"]).parent
        with tempfile.TemporaryDirectory(dir=uat.ROOT / "logs") as temporary:
            copied = Path(temporary) / "source"
            shutil.copytree(uat.ROOT / original, copied)
            report_path = copied / "report.json"
            report = json.loads(report_path.read_text(encoding="utf-8"))
            report["results"][0]["criterionId"] = []
            report_path.write_text(json.dumps(report), encoding="utf-8")
            malformed = copy.deepcopy(source)
            malformed["reports"] = [
                {
                    "taskId": 281,
                    "path": report_path.relative_to(uat.ROOT).as_posix(),
                }
            ]
            input_path = Path(temporary) / "input.json"
            input_path.write_text(json.dumps(malformed), encoding="utf-8")
            stderr = io.StringIO()
            with contextlib.redirect_stderr(stderr):
                exit_code = uat.main(["build", "--input", str(input_path)])
            self.assertEqual(1, exit_code)
            self.assertIn("Phase 08.02 UAT invalid:", stderr.getvalue())
            self.assertNotIn("Traceback", stderr.getvalue())

    def test_all_open_findings_are_required(self) -> None:
        report = copy.deepcopy(self.report)
        report["openFindings"].pop()
        with self.assertRaisesRegex(uat.UATError, "open findings"):
            self.validate(report)

    def test_recursive_sensitive_evidence_metadata_is_rejected(self) -> None:
        original = Path(self.report["results"][0]["sources"][0]["report"]).parent
        with tempfile.TemporaryDirectory(dir=uat.ROOT / "logs") as temporary:
            copied = Path(temporary) / "source"
            shutil.copytree(uat.ROOT / original, copied)
            report_path = copied / "report.json"
            report = json.loads(report_path.read_text(encoding="utf-8"))
            report["results"][0]["evidence"][0]["metadata"] = {
                "connection": {"databaseUrl": "postgresql://user:secret@host/db"}
            }
            report_path.write_text(json.dumps(report), encoding="utf-8")
            criteria, findings = uat.load_controls()
            with self.assertRaisesRegex(uat.UATError, "prohibited sensitive metadata"):
                uat.validate_source_report(
                    report_path,
                    {item["id"]: item for item in criteria},
                    findings,
                )

    def test_recursive_sensitive_final_report_metadata_is_rejected(self) -> None:
        report = copy.deepcopy(self.report)
        report["results"][0]["sources"][0]["evidence"][0]["metadata"] = {
            "connection": {"databaseUrl": "postgresql://user:secret@host/db"}
        }
        with self.assertRaisesRegex(uat.UATError, "prohibited sensitive metadata"):
            self.validate(report)

    def test_normalized_sensitive_aliases_are_rejected_at_any_final_depth(self) -> None:
        for key in ("databaseUrl", "database_url", "DATABASE-URL", "data.base---url"):
            with self.subTest(key=key):
                report = copy.deepcopy(self.report)
                report["acceptanceDecision"]["nested"] = {
                    "one": [{"two": {"three": {key: "redacted"}}}]
                }
                with self.assertRaisesRegex(uat.UATError, "prohibited sensitive metadata"):
                    self.validate(report)

    def test_invalid_calendar_dates_are_rejected(self) -> None:
        report = copy.deepcopy(self.report)
        report["acceptanceDecision"] = {
            "status": "REJECTED",
            "owner": "project-owner",
            "date": "2026-99-99",
            "deviations": [],
        }
        with self.assertRaisesRegex(uat.UATError, "project owner and date"):
            self.validate(report)

    def test_acceptance_without_owner_and_date_is_rejected(self) -> None:
        report = copy.deepcopy(self.report)
        report["acceptanceDecision"]["status"] = "REJECTED"
        with self.assertRaisesRegex(uat.UATError, "project owner and date"):
            self.validate(report)

    def test_mandatory_nonpass_prevents_plain_acceptance(self) -> None:
        report = copy.deepcopy(self.report)
        report["acceptanceDecision"] = {
            "status": "ACCEPTED",
            "owner": "project-owner",
            "date": "2026-07-29",
            "deviations": [],
        }
        with self.assertRaisesRegex(uat.UATError, "prevent acceptance"):
            self.validate(report)

    def test_incomplete_accepted_deviations_are_rejected(self) -> None:
        report = copy.deepcopy(self.report)
        report["acceptanceDecision"] = {
            "status": "ACCEPTED_WITH_DEVIATIONS",
            "owner": "project-owner",
            "date": "2026-07-29",
            "deviations": [],
        }
        with self.assertRaisesRegex(uat.UATError, "every mandatory non-pass"):
            self.validate(report)

    def test_centralized_logging_cannot_pass_from_blocked_console_evidence(self) -> None:
        report = copy.deepcopy(self.report)
        for result in report["results"]:
            if result["requirement"] == "SW-REQ-084":
                result["status"] = "PASS"
                for source in result["sources"]:
                    source["status"] = "PASS"
                result["rootCauseIds"] = []
                result["findingIds"] = []
        self.recount(report)
        with self.assertRaisesRegex(uat.UATError, "console evidence"):
            self.validate(report)

    def test_erasure_pass_requires_worker_backend_evidence(self) -> None:
        report = copy.deepcopy(self.report)
        for result in report["results"]:
            if result["requirement"] == "SW-REQ-073":
                for source in result["sources"]:
                    source["backendEvidence"] = [
                        value for value in source["backendEvidence"] if value != "worker_state=completed"
                    ]
        with self.assertRaisesRegex(uat.UATError, "worker/backend evidence"):
            self.validate(report)

    def test_uat_requires_unchecked_owner_decision_fields(self) -> None:
        with self.assertRaisesRegex(uat.UATError, "decision fields"):
            self.validate(text=self.uat_text.replace("Project owner:", "Decision owner:"))


if __name__ == "__main__":
    unittest.main()
