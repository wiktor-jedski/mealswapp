"""Tests for the Task 280 Phase 08 acceptance contracts.

Implements DESIGN-014 MetricsCollector acceptance-report regression coverage.
"""

from __future__ import annotations

import copy
import contextlib
import importlib.util
import io
import json
import os
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

MODULE_PATH = Path(__file__).with_name("phase08_acceptance.py")
SPEC = importlib.util.spec_from_file_location("phase08_acceptance", MODULE_PATH)
assert SPEC and SPEC.loader
acceptance = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = acceptance
SPEC.loader.exec_module(acceptance)


class Phase08AcceptanceTests(unittest.TestCase):
    """Verify manifest coverage, reporting, sanitization, and finding history."""

    @classmethod
    def setUpClass(cls) -> None:
        cls.manifest = acceptance.load_json(acceptance.DEFAULT_MANIFEST)
        cls.criteria = acceptance.validate_manifest(
            cls.manifest, acceptance.ROOT / "req_tests.md"
        )
        cls.scenario_ids = {item["scenarioId"] for item in cls.criteria}

    def pass_results(self) -> list[dict[str, object]]:
        """Return one minimal passing result for every manifest criterion."""

        return [
            {
                "criterionId": criterion["id"],
                "status": "PASS",
                "requestIds": [],
                "evidence": [],
                "backendEvidence": [],
            }
            for criterion in self.criteria
        ]

    @staticmethod
    def ledger(*findings: dict[str, object]) -> dict[str, object]:
        """Build a finding ledger fixture."""

        return {"schema": "mealswapp.phase08-findings.v1", "findings": list(findings)}

    @staticmethod
    def finding(
        finding_id: str = "P08-FIND-001",
        root: str = "ROOT-001",
        status: str = "OPEN DEFECT",
        requirement: str = "SW-REQ-054",
        scenario: str = "P08-SWR054-ADMINISTRATIVE-ACCESS",
    ) -> dict[str, object]:
        """Build a complete sanitized finding fixture."""

        value: dict[str, object] = {
            "id": finding_id,
            "rootCauseId": root,
            "status": status,
            "requirements": [requirement],
            "scenarios": [scenario],
            "observed": "Restricted route returned an unexpected response.",
            "expected": "Restricted route enforces the documented role boundary.",
            "evidence": "evidence/admin-boundary.txt",
            "owner": "admin-auth",
            "retestCondition": "Run the isolated administrative access scenario.",
        }
        if status == "CLOSED":
            value["closedDate"] = "2026-07-28"
            value["passingEvidence"] = "evidence/admin-boundary-retest.txt"
        return value

    def validate_ledger(
        self, *findings: dict[str, object]
    ) -> dict[str, dict[str, object]]:
        """Validate a ledger against its complete mandatory history."""

        return acceptance.validate_findings(
            self.ledger(*findings),
            self.scenario_ids,
            {str(finding["id"]) for finding in findings},
        )

    def test_manifest_maps_every_source_entry_exactly_once(self) -> None:
        self.assertEqual(12, len(self.manifest["scenarios"]))
        self.assertEqual(91, len(self.criteria))
        self.assertEqual(
            91, len(acceptance.req_test_entries(acceptance.ROOT / "req_tests.md"))
        )
        self.assertEqual(acceptance.REQUIREMENTS, {item["requirement"] for item in self.criteria})
        self.assertTrue(all(item["owner"] for item in self.criteria))
        self.assertTrue(all(item["requiredEnvironment"] for item in self.criteria))
        self.assertTrue(all(item["evidenceTypes"] for item in self.criteria))

    def test_manifest_rejects_duplicate_and_orphan_mappings(self) -> None:
        duplicate = copy.deepcopy(self.manifest)
        duplicate["scenarios"][0]["criteria"][1]["id"] = duplicate["scenarios"][0]["criteria"][0]["id"]
        with self.assertRaisesRegex(acceptance.ValidationError, "duplicate criterion ID"):
            acceptance.validate_manifest(duplicate, acceptance.ROOT / "req_tests.md")
        orphan = copy.deepcopy(self.manifest)
        orphan["scenarios"][0]["criteria"][0]["source"]["line"] = 999
        with self.assertRaisesRegex(acceptance.ValidationError, "unmapped req_tests.md"):
            acceptance.validate_manifest(orphan, acceptance.ROOT / "req_tests.md")
        kind_tamper = copy.deepcopy(self.manifest)
        kind_tamper["scenarios"][0]["criteria"][0]["kind"] = "criterion"
        with self.assertRaisesRegex(acceptance.ValidationError, "unmapped req_tests.md"):
            acceptance.validate_manifest(kind_tamper, acceptance.ROOT / "req_tests.md")

    def test_manifest_has_required_json_sidecar_traceability(self) -> None:
        sidecar = acceptance.DEFAULT_MANIFEST.with_name(
            acceptance.DEFAULT_MANIFEST.name + "-trace.md"
        ).read_text(encoding="utf-8")
        self.assertIn("DESIGN-014", sidecar)
        self.assertIn("MetricsCollector", sidecar)
        for requirement in sorted(acceptance.REQUIREMENTS):
            self.assertIn(requirement, sidecar)
        history_sidecar = acceptance.DEFAULT_FINDING_HISTORY.with_name(
            acceptance.DEFAULT_FINDING_HISTORY.name + "-trace.md"
        ).read_text(encoding="utf-8")
        self.assertIn("DESIGN-014", history_sidecar)
        self.assertIn("append-only", history_sidecar)

    def test_mixed_report_is_ordered_correlated_grouped_and_exit_one(self) -> None:
        raw = self.pass_results()
        raw[0] = {
            "criterionId": self.criteria[0]["id"],
            "status": "FAIL",
            "rootCauseId": "ROOT-001",
            "requestIds": ["req-b", "req-a"],
            "evidence": [
                {"type": "playwright", "path": "traces/failure.zip"},
                {"type": "playwright", "path": "screenshots/failure.png"},
                {"type": "backend", "path": "backend/request-summary.json"},
            ],
            "backendEvidence": ["audit_count=0", "mutation_count=0"],
        }
        raw[1] = {
            "criterionId": self.criteria[1]["id"],
            "status": "BLOCKED",
            "rootCauseId": "ROOT-001",
            "requestIds": [],
            "evidence": [],
            "backendEvidence": [],
        }
        finding = self.finding(
            requirement=self.criteria[0]["requirement"],
            scenario=self.criteria[0]["scenarioId"],
        )
        finding["requirements"] = sorted(
            {self.criteria[0]["requirement"], self.criteria[1]["requirement"]}
        )
        finding["scenarios"] = sorted(
            {self.criteria[0]["scenarioId"], self.criteria[1]["scenarioId"]}
        )
        findings = self.validate_ledger(finding)
        normalized = acceptance.normalize_results(list(reversed(raw)), self.criteria)
        linked = acceptance.synchronize_results(normalized, self.criteria, findings)
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for relative in (
                "traces/failure.zip",
                "screenshots/failure.png",
                "backend/request-summary.json",
            ):
                path = root / "evidence" / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(b"sanitized")
            run_dir, code = acceptance.write_report(
                "mixed-run", linked, root, evidence_root=root / "evidence"
            )
            report = json.loads((run_dir / "report.json").read_text(encoding="utf-8"))
            rendered = (run_dir / "report.html").read_text(encoding="utf-8")
            self.assertTrue((run_dir / "evidence/screenshots/failure.png").is_file())
        self.assertEqual(1, code)
        self.assertEqual("FAIL", report["status"])
        self.assertEqual(["req-a", "req-b"], report["results"][0]["requestIds"])
        self.assertEqual(
            ["evidence/screenshots/failure.png", "evidence/traces/failure.zip"],
            [
                item["path"]
                for item in report["results"][0]["evidence"]
                if item["type"] == "playwright"
            ],
        )
        self.assertEqual(
            [self.criteria[0]["id"], self.criteria[1]["id"]],
            report["rootCauses"][0]["criteria"],
        )
        self.assertIn('href="evidence/screenshots/failure.png"', rendered)
        self.assertIn("P08-FIND-001", rendered)

    def test_fixture_exit_codes_are_zero_one_two(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            passing = acceptance.normalize_results(self.pass_results(), self.criteria)
            passing = acceptance.synchronize_results(passing, self.criteria, {})
            self.assertEqual(0, acceptance.write_report("pass-run", passing, root)[1])

            finding = self.finding(
                requirement=self.criteria[0]["requirement"],
                scenario=self.criteria[0]["scenarioId"],
            )
            findings = self.validate_ledger(finding)
            for status, expected_code in (("FAIL", 1), ("BLOCKED", 2)):
                raw = self.pass_results()
                raw[0].update({"status": status, "rootCauseId": "ROOT-001"})
                linked = acceptance.synchronize_results(
                    acceptance.normalize_results(raw, self.criteria),
                    self.criteria,
                    findings,
                )
                self.assertEqual(
                    expected_code,
                    acceptance.write_report(f"{status.lower()}-run", linked, root)[1],
                )

    def test_requirement_report_accepts_one_complete_manifest_scenario(self) -> None:
        requirement = "SW-REQ-054"
        criteria = [
            criterion
            for criterion in self.criteria
            if criterion["requirement"] == requirement
        ]
        results = [
            {
                "criterionId": criterion["id"],
                "status": "PASS",
                "requestIds": [],
                "evidence": [],
                "backendEvidence": [],
            }
            for criterion in criteria
        ]
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            result = root / "result.json"
            result.write_text(json.dumps({"results": results}), encoding="utf-8")
            args = acceptance.build_parser().parse_args(
                [
                    "report",
                    "--run-id",
                    "task281-fixture",
                    "--output-root",
                    str(root / "reports"),
                    "--result",
                    str(result),
                    "--requirement",
                    requirement,
                ]
            )
            self.assertEqual(0, acceptance.command_report(args))
            report = json.loads(
                (root / "reports/task281-fixture/report.json").read_text(
                    encoding="utf-8"
                )
            )
        self.assertEqual(
            {criterion["id"] for criterion in criteria},
            {result["criterionId"] for result in report["results"]},
        )

    def test_report_rejects_missing_duplicate_and_orphan_results(self) -> None:
        with self.assertRaisesRegex(acceptance.ValidationError, "missing criterion"):
            acceptance.normalize_results(self.pass_results()[:-1], self.criteria)
        duplicate = self.pass_results() + [self.pass_results()[0]]
        with self.assertRaisesRegex(acceptance.ValidationError, "duplicate criterion"):
            acceptance.normalize_results(duplicate, self.criteria)
        orphan = self.pass_results()
        orphan[0]["criterionId"] = "P08-ORPHAN"
        with self.assertRaisesRegex(acceptance.ValidationError, "orphan result"):
            acceptance.normalize_results(orphan, self.criteria)

    def test_evidence_links_must_be_safe_and_relative(self) -> None:
        for path in (
            "/tmp/evidence.png",
            "../evidence.png",
            "https://host/evidence",
            ".hidden",
            "safe/.hidden",
            "a/%2e%2e/secret",
            "a/%2fsecret",
            "a?query",
            "a#fragment",
            "a\nb",
            "a\x00b",
        ):
            raw = self.pass_results()
            raw[0]["evidence"] = [{"type": "playwright", "path": path}]
            with self.subTest(path=path), self.assertRaises(acceptance.ValidationError):
                acceptance.normalize_results(raw, self.criteria)

    def test_report_rejects_missing_and_symlink_evidence(self) -> None:
        raw = self.pass_results()
        raw[0]["evidence"] = [{"type": "playwright", "path": "screenshots/proof.png"}]
        linked = acceptance.synchronize_results(
            acceptance.normalize_results(raw, self.criteria), self.criteria, {}
        )
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            evidence = root / "evidence"
            evidence.mkdir()
            with self.assertRaisesRegex(acceptance.ValidationError, "missing"):
                acceptance.write_report(
                    "missing-evidence", linked, root, evidence_root=evidence
                )
            target = root / "target.png"
            target.write_bytes(b"proof")
            (evidence / "screenshots").mkdir()
            (evidence / "screenshots/proof.png").symlink_to(target)
            with self.assertRaisesRegex(acceptance.ValidationError, "symlink"):
                acceptance.write_report(
                    "symlink-evidence", linked, root, evidence_root=evidence
                )

    def test_protected_report_names_are_preserved_in_evidence_namespace(self) -> None:
        raw = self.pass_results()
        raw[0]["evidence"] = [
            {"type": "playwright", "path": "report.json"},
            {"type": "playwright", "path": "report.html"},
        ]
        linked = acceptance.synchronize_results(
            acceptance.normalize_results(raw, self.criteria), self.criteria, {}
        )
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "source"
            source.mkdir()
            (source / "report.json").write_bytes(b"producer-json-evidence")
            (source / "report.html").write_bytes(b"producer-html-evidence")
            run_dir, code = acceptance.write_report(
                "protected-names", linked, root / "reports", evidence_root=source
            )
            report = json.loads((run_dir / "report.json").read_text(encoding="utf-8"))
            rendered = (run_dir / "report.html").read_text(encoding="utf-8")
            self.assertEqual(0, code)
            self.assertEqual(
                b"producer-json-evidence",
                (run_dir / "evidence/report.json").read_bytes(),
            )
            self.assertEqual(
                b"producer-html-evidence",
                (run_dir / "evidence/report.html").read_bytes(),
            )
            self.assertEqual(
                ["evidence/report.html", "evidence/report.json"],
                [item["path"] for item in report["results"][0]["evidence"]],
            )
            self.assertIn('href="evidence/report.json"', rendered)

    def test_sensitive_fields_and_values_are_rejected(self) -> None:
        prohibited = [
            {"email": "redacted"},
            {"credentials": "redacted"},
            {"cookies": "redacted"},
            {"csrfToken": "redacted"},
            {"idempotencyKey": "redacted"},
            {"userId": "redacted"},
            {"itemId": "redacted"},
            {"queryText": "redacted"},
            {"nameText": "redacted"},
            {"providerPayload": "redacted"},
            {"databaseUrl": "redacted"},
            {"secret": "redacted"},
            {"stackDiagnostics": "redacted"},
            {"safe": "person@example.test"},
            {"safe": "postgresql://host/db"},
            {"safe": "Traceback (most recent call last)"},
        ]
        for value in prohibited:
            with self.subTest(value=value), self.assertRaises(acceptance.ValidationError):
                acceptance.assert_safe_value(value)
        raw = self.pass_results()
        raw[0]["backendEvidence"] = ["raw_provider_payload=opaque"]
        with self.assertRaisesRegex(acceptance.ValidationError, "sanitized summaries"):
            acceptance.normalize_results(raw, self.criteria)

    def test_finding_requires_complete_unresolved_metadata(self) -> None:
        for field in (
            "requirements",
            "scenarios",
            "observed",
            "expected",
            "evidence",
            "owner",
            "retestCondition",
        ):
            finding = self.finding()
            del finding[field]
            with self.subTest(field=field), self.assertRaises(acceptance.ValidationError):
                acceptance.validate_findings(
                    self.ledger(finding), self.scenario_ids, {str(finding["id"])}
                )

    def test_nonpass_requires_matching_unresolved_finding(self) -> None:
        raw = self.pass_results()
        raw[0].update({"status": "FAIL", "rootCauseId": "ROOT-001"})
        normalized = acceptance.normalize_results(raw, self.criteria)
        with self.assertRaisesRegex(acceptance.ValidationError, "no unresolved finding"):
            acceptance.synchronize_results(normalized, self.criteria, {})
        closed = self.finding(
            status="CLOSED",
            requirement=self.criteria[0]["requirement"],
            scenario=self.criteria[0]["scenarioId"],
        )
        findings = self.validate_ledger(closed)
        with self.assertRaisesRegex(acceptance.ValidationError, "no unresolved finding"):
            acceptance.synchronize_results(normalized, self.criteria, findings)

    def test_duplicate_root_cause_findings_are_rejected(self) -> None:
        first = self.finding()
        second = self.finding(finding_id="P08-FIND-002")
        with self.assertRaisesRegex(acceptance.ValidationError, "duplicate findings"):
            acceptance.validate_findings(
                self.ledger(first, second), self.scenario_ids, {"P08-FIND-001", "P08-FIND-002"}
            )

    def test_embedded_ledger_rejects_duplicate_json_keys(self) -> None:
        document = (
            acceptance.FINDINGS_START
            + '\n```json\n{"schema":"mealswapp.phase08-findings.v1",'
            + '"findings":[],"findings":[]}\n```\n'
            + acceptance.FINDINGS_END
        )
        with self.assertRaisesRegex(acceptance.ValidationError, "duplicate JSON key"):
            acceptance.extract_finding_ledger(document)

    def test_passing_retest_closes_and_retains_historical_finding(self) -> None:
        closed = self.finding(
            status="CLOSED",
            requirement=self.criteria[0]["requirement"],
            scenario=self.criteria[0]["scenarioId"],
        )
        findings = self.validate_ledger(closed)
        raw = self.pass_results()
        raw[0]["resolvedRootCauseIds"] = ["ROOT-001"]
        linked = acceptance.synchronize_results(
            acceptance.normalize_results(raw, self.criteria), self.criteria, findings
        )
        self.assertEqual(["P08-FIND-001"], linked[0]["findingIds"])
        with self.assertRaisesRegex(acceptance.ValidationError, "deleted"):
            acceptance.validate_findings(self.ledger(), self.scenario_ids, {"P08-FIND-001"})

    def test_closed_finding_requires_date_and_passing_evidence(self) -> None:
        for field in ("closedDate", "passingEvidence"):
            closed = self.finding(status="CLOSED")
            del closed[field]
            with self.subTest(field=field), self.assertRaises(acceptance.ValidationError):
                acceptance.validate_findings(
                    self.ledger(closed), self.scenario_ids, {"P08-FIND-001"}
                )

    def test_failure_during_finalization_publishes_no_partial_run(self) -> None:
        results = acceptance.normalize_results(self.pass_results(), self.criteria)
        linked = acceptance.synchronize_results(results, self.criteria, {})
        original = Path.write_text
        calls = 0

        def fail_second(path: Path, content: str, **kwargs: object) -> int:
            nonlocal calls
            calls += 1
            if calls == 2:
                raise OSError("injected finalization failure")
            return original(path, content, **kwargs)

        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with mock.patch.object(Path, "write_text", fail_second):
                with self.assertRaisesRegex(
                    acceptance.ValidationError, "cannot finalize report artifacts"
                ):
                    acceptance.write_report("atomic-run", linked, root)
            self.assertFalse((root / "atomic-run").exists())
            self.assertFalse((root / ".atomic-run.tmp").exists())

    def test_all_producers_run_after_individual_failures(self) -> None:
        plan = {
            "producers": [
                {"id": "PLAYWRIGHT", "command": ["first"], "result": "first.json"},
                {"id": "BACKEND", "command": ["second"], "result": "second.json"},
            ]
        }
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            plan_path = root / "plan.json"
            plan_path.write_text(json.dumps(plan), encoding="utf-8")
            with mock.patch.object(
                acceptance.subprocess,
                "run",
                side_effect=[
                    mock.Mock(returncode=1),
                    mock.Mock(returncode=0),
                ],
            ) as run:
                paths, errors = acceptance.execute_producers(plan_path, root / "results")
        self.assertEqual(2, run.call_count)
        self.assertEqual([], paths)
        self.assertEqual(
            ["PLAYWRIGHT:EXIT-1", "PLAYWRIGHT:MISSING-RESULT", "BACKEND:MISSING-RESULT"],
            errors,
        )

    def test_run_preserves_evidence_and_nonzero_producer_cannot_return_zero(self) -> None:
        results = self.pass_results()
        results[0]["evidence"] = [
            {"type": "playwright", "path": "screenshots/kept.png"}
        ]
        payload = json.dumps({"results": results}, ensure_ascii=True)
        first = (
            "import json,os,pathlib,sys;"
            "root=pathlib.Path(os.environ['PHASE08_ACCEPTANCE_RESULT_DIR']);"
            "(root/'screenshots').mkdir();"
            "(root/'screenshots/kept.png').write_bytes(b'sanitized');"
            f"(root/'first.json').write_text({payload!r},encoding='utf-8');"
            "sys.exit(7)"
        )
        second_payload = json.dumps({"results": []})
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            marker = root / "later-ran"
            second = (
                "import json,os,pathlib;"
                "root=pathlib.Path(os.environ['PHASE08_ACCEPTANCE_RESULT_DIR']);"
                f"pathlib.Path({str(marker)!r}).write_text('yes',encoding='utf-8');"
                f"(root/'second.json').write_text({second_payload!r},encoding='utf-8')"
            )
            plan = {
                "producers": [
                    {
                        "id": "PLAYWRIGHT",
                        "command": [sys.executable, "-c", first],
                        "result": "first.json",
                    },
                    {
                        "id": "BACKEND",
                        "command": [sys.executable, "-c", second],
                        "result": "second.json",
                    },
                ]
            }
            plan_path = root / "plan.json"
            plan_path.write_text(json.dumps(plan), encoding="utf-8")
            code = acceptance.main(
                [
                    "run",
                    "--run-id",
                    "producer-failure",
                    "--output-root",
                    str(root / "reports"),
                    "--plan",
                    str(plan_path),
                ]
            )
            report_path = root / "reports/producer-failure/report.json"
            report = json.loads(report_path.read_text(encoding="utf-8"))
            self.assertEqual(1, code)
            self.assertEqual("FAIL", report["status"])
            self.assertEqual(1, report["exitCode"])
            self.assertEqual(["PLAYWRIGHT:EXIT-7"], report["producerFailures"])
            self.assertTrue(marker.is_file())
            self.assertEqual(
                b"sanitized",
                (
                    root
                    / "reports/producer-failure/evidence/screenshots/kept.png"
                ).read_bytes(),
            )

    def test_producer_stdout_stderr_and_traceback_are_never_emitted_by_cli(self) -> None:
        payload = json.dumps({"results": self.pass_results()}, ensure_ascii=True)
        producer = (
            "import os,pathlib,sys;"
            "print('SUPER_SECRET_STDOUT');"
            "print('Traceback (most recent call last): SUPER_SECRET_STDERR',file=sys.stderr);"
            "root=pathlib.Path(os.environ['PHASE08_ACCEPTANCE_RESULT_DIR']);"
            f"(root/'result.json').write_text({payload!r},encoding='utf-8');"
            "raise RuntimeError('SUPER_SECRET_EXCEPTION')"
        )
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            plan = {
                "producers": [
                    {
                        "id": "BACKEND",
                        "command": [sys.executable, "-c", producer],
                        "result": "result.json",
                    }
                ]
            }
            plan_path = root / "plan.json"
            plan_path.write_text(json.dumps(plan), encoding="utf-8")
            stdout = io.StringIO()
            stderr = io.StringIO()
            with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
                code = acceptance.main(
                    [
                        "run",
                        "--run-id",
                        "sanitized-streams",
                        "--output-root",
                        str(root / "reports"),
                        "--plan",
                        str(plan_path),
                    ]
                )
            emitted = stdout.getvalue() + stderr.getvalue()
            self.assertEqual(1, code)
            self.assertNotIn("SUPER_SECRET", emitted)
            self.assertNotIn("Traceback", emitted)
            self.assertIn("producer diagnostics: BACKEND:EXIT-1", emitted)
            self.assertIn("phase08 acceptance report:", stdout.getvalue())

    def test_missing_and_spawn_failed_producers_never_return_zero(self) -> None:
        payload = json.dumps({"results": self.pass_results()}, ensure_ascii=True)
        passing = (
            "import os,pathlib;"
            "root=pathlib.Path(os.environ['PHASE08_ACCEPTANCE_RESULT_DIR']);"
            f"(root/'pass.json').write_text({payload!r},encoding='utf-8')"
        )
        for producer_id, command, diagnostic in (
            ("MISSING", [sys.executable, "-c", "pass"], "MISSING:MISSING-RESULT"),
            ("SPAWN", ["/definitely/not/a/command"], "SPAWN:SPAWN-FAILED"),
        ):
            with self.subTest(producer_id=producer_id), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                plan = {
                    "producers": [
                        {"id": producer_id, "command": command, "result": "missing.json"},
                        {
                            "id": "PASSING",
                            "command": [sys.executable, "-c", passing],
                            "result": "pass.json",
                        },
                    ]
                }
                plan_path = root / "plan.json"
                plan_path.write_text(json.dumps(plan), encoding="utf-8")
                code = acceptance.main(
                    [
                        "run",
                        "--run-id",
                        f"{producer_id.lower()}-producer",
                        "--output-root",
                        str(root / "reports"),
                        "--plan",
                        str(plan_path),
                    ]
                )
                report = json.loads(
                    (root / f"reports/{producer_id.lower()}-producer/report.json").read_text(
                        encoding="utf-8"
                    )
                )
                self.assertEqual(1, code)
                self.assertEqual("FAIL", report["status"])
                self.assertIn(diagnostic, report["producerFailures"])

    def test_history_registry_is_mandatory_for_normal_cli(self) -> None:
        finding = self.finding()
        document = (
            "Phase 08\n"
            + acceptance.FINDINGS_START
            + "\n```json\n"
            + json.dumps(self.ledger(finding))
            + "\n```\n"
            + acceptance.FINDINGS_END
        )
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            open_doc = root / "open.md"
            open_doc.write_text(document, encoding="utf-8")
            empty_history = root / "empty-history.json"
            empty_history.write_text(
                json.dumps(
                    {
                        "schema": "mealswapp.phase08-finding-history.v1",
                        "findingIds": [],
                    }
                ),
                encoding="utf-8",
            )
            stderr = io.StringIO()
            with contextlib.redirect_stderr(stderr):
                code = acceptance.main(
                    [
                        "--open-doc",
                        str(open_doc),
                        "--finding-history",
                        str(empty_history),
                        "validate",
                    ]
                )
            self.assertEqual(1, code)
            self.assertIn("missing from history", stderr.getvalue())
            history = root / "history.json"
            history.write_text(
                json.dumps(
                    {
                        "schema": "mealswapp.phase08-finding-history.v1",
                        "findingIds": ["P08-FIND-001"],
                    }
                ),
                encoding="utf-8",
            )
            self.assertEqual(
                0,
                acceptance.main(
                    [
                        "--open-doc",
                        str(open_doc),
                        "--finding-history",
                        str(history),
                        "validate",
                    ]
                ),
            )
            with contextlib.redirect_stderr(io.StringIO()):
                self.assertEqual(
                    1,
                    acceptance.main(
                        [
                            "--open-doc",
                            str(open_doc),
                            "--finding-history",
                            str(root / "missing.json"),
                            "validate",
                        ]
                    ),
                )

    def test_cli_external_errors_are_sanitized_without_traceback(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            malformed = root / "manifest.json"
            malformed.write_bytes(b"\xffsecret-path")
            stderr = io.StringIO()
            with contextlib.redirect_stderr(stderr):
                code = acceptance.main(["--manifest", str(malformed), "validate"])
            self.assertEqual(1, code)
            message = stderr.getvalue()
            self.assertNotIn("Traceback", message)
            self.assertNotIn(str(malformed), message)
            self.assertNotIn("secret-path", message)

            result = root / "results.json"
            result.write_text(
                json.dumps({"results": self.pass_results()}), encoding="utf-8"
            )
            stderr = io.StringIO()
            with mock.patch.object(Path, "write_text", side_effect=OSError("private path")):
                with contextlib.redirect_stderr(stderr):
                    code = acceptance.main(
                        [
                            "report",
                            "--run-id",
                            "write-failure",
                            "--output-root",
                            str(root / "reports"),
                            "--result",
                            str(result),
                        ]
                    )
            self.assertEqual(1, code)
            self.assertNotIn("Traceback", stderr.getvalue())
            self.assertNotIn("private path", stderr.getvalue())

    def test_current_open_document_ledger_is_valid(self) -> None:
        ledger = acceptance.extract_finding_ledger(
            acceptance.DEFAULT_OPEN_DOC.read_text(encoding="utf-8")
        )
        history = acceptance.validate_finding_history(
            acceptance.load_json(acceptance.DEFAULT_FINDING_HISTORY)
        )
        findings = acceptance.validate_findings(ledger, self.scenario_ids, history)
        self.assertEqual(history, set(findings))


if __name__ == "__main__":
    unittest.main()
