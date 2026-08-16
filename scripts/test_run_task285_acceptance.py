import datetime as dt
import importlib.util
import json
import subprocess
import tempfile
import unittest
import urllib.error
from pathlib import Path
from unittest import mock
from urllib.parse import urlsplit

SPEC = importlib.util.spec_from_file_location(
    "task285", Path(__file__).with_name("run-task285-acceptance.py")
)
assert SPEC and SPEC.loader
task285 = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(task285)
sink = task285.sink
UTC = dt.timezone.utc
PROVENANCE = "a" * 48


def timestamp(value: dt.datetime) -> str:
    return value.isoformat().replace("+00:00", "Z")


def action_receipt(now: dt.datetime) -> dict:
    events = []
    contracts = {
        "auth_success": ("auth_login", "session", "succeeded"),
        "auth_failure": ("auth_login", "session", "failed"),
        "external_search": ("external_search", "external_catalog", "succeeded"),
        "external_import": ("curated_import", "global_item", "succeeded"),
        "manual_success": ("manual_create", "global_item", "succeeded"),
        "manual_failure": ("manual_create", "global_item", "validation_failed"),
        "classification": ("classification_create", "classification", "succeeded"),
        "user_administration": ("user_lookup", "user_administration", "succeeded"),
        "validation": ("manual_create", "global_item", "validation_failed"),
        "dependency": ("external_search", "external_catalog", "dependency_failed"),
        "audit_failure": ("manual_create", "global_item", "audit_failed"),
    }
    for index, (category, contract) in enumerate(contracts.items(), 1):
        request_id = f"00000000-0000-4000-8000-{index:012d}"
        events.append(
            {
                "category": category,
                "requestId": request_id,
                "action": contract[0],
                "resource": contract[1],
                "outcome": contract[2],
            }
        )
    return {
        "schema": "mealswapp.task285-actions.v1",
        "provenance": PROVENANCE,
        "startedAt": timestamp(now),
        "finishedAt": timestamp(now + dt.timedelta(seconds=10)),
        "events": events,
    }


def cloud_entry(event: sink.ExpectedEvent, created: dt.datetime) -> dict:
    return {
        "timestamp": timestamp(created),
        "receiveTimestamp": timestamp(created + dt.timedelta(seconds=2)),
        "jsonPayload": {
            "requestId": event.request_id,
            "action": event.action,
            "resource": event.resource,
            "outcome": event.outcome,
        },
    }


class FakeClient:
    def __init__(self, responses, *, timeout=1, poll=0.01):
        self.responses = list(responses)
        self.config = sink.SinkConfig(
            "mealswapp-test",
            ("projects/mealswapp-test",),
            poll_seconds=poll,
            timeout_seconds=timeout,
        )
        self.calls = 0

    def query(self, _request_ids, _window, **_kwargs):
        response = self.responses[min(self.calls, len(self.responses) - 1)]
        self.calls += 1
        if isinstance(response, BaseException):
            raise response
        return response, 1


class Task285SinkTests(unittest.TestCase):
    def test_configuration_is_least_scope_and_bounded(self):
        config = sink.SinkConfig.from_environment(
            {
                "MEALSWAPP_TASK285_GCP_PROJECT": "mealswapp-test",
                "MEALSWAPP_TASK285_LOG_RESOURCE_NAMES": "projects/mealswapp-test",
                "MEALSWAPP_TASK285_LOG_PAGE_SIZE": "1000",
            }
        )
        self.assertEqual(config.page_size, 1000)
        for invalid in (
            {},
            {
                "MEALSWAPP_TASK285_GCP_PROJECT": "mealswapp-test",
                "MEALSWAPP_TASK285_LOG_RESOURCE_NAMES": "organizations/123",
            },
            {
                "MEALSWAPP_TASK285_GCP_PROJECT": "mealswapp-test",
                "MEALSWAPP_TASK285_LOG_RESOURCE_NAMES": "projects/mealswapp-test",
                "MEALSWAPP_TASK285_LOG_PAGE_SIZE": "1001",
            },
        ):
            with self.assertRaises(sink.SinkUnavailableBlocked):
                sink.SinkConfig.from_environment(invalid)

    def test_deployed_target_rejects_local_credentials_and_missing_acknowledgement(self):
        self.assertEqual(
            task285.validate_deployed_url(
                "https://acceptance.example.test",
                "deployed-test-with-centralized-log-sink",
                "acceptance.example.test",
                resolver=lambda _host: ["8.8.8.8"],
            ),
            "https://acceptance.example.test",
        )
        for url, ack in (
            ("http://acceptance.example.test", "deployed-test-with-centralized-log-sink"),
            ("https://localhost", "deployed-test-with-centralized-log-sink"),
            ("https://user:pass@example.test", "deployed-test-with-centralized-log-sink"),
            ("https://acceptance.example.test/path", "deployed-test-with-centralized-log-sink"),
            ("https://acceptance.example.test", ""),
        ):
            with self.assertRaises(sink.SinkUnavailableBlocked):
                task285.validate_deployed_url(
                    url,
                    ack,
                    "acceptance.example.test",
                    resolver=lambda _host: ["8.8.8.8"],
                )
        for url in (
            "https://127.0.0.2",
            "https://127.1.2.3",
            "https://10.0.0.2",
            "https://169.254.2.3",
            "https://192.0.2.1",
            "https://[::1]",
            "https://[::ffff:127.0.0.1]",
        ):
            with self.assertRaises(sink.SinkUnavailableBlocked):
                task285.validate_deployed_url(
                    url,
                    "deployed-test-with-centralized-log-sink",
                    urlsplit(url).hostname or "",
                )
        with self.assertRaises(sink.SinkUnavailableBlocked):
            task285.validate_deployed_url(
                "https://acceptance.example.test",
                "deployed-test-with-centralized-log-sink",
                "other.example.test",
                resolver=lambda _host: ["8.8.8.8"],
            )
        with self.assertRaises(sink.SinkUnavailableBlocked):
            task285.validate_deployed_url(
                "https://acceptance.example.test",
                "deployed-test-with-centralized-log-sink",
                "acceptance.example.test",
                resolver=lambda _host: ["10.0.0.2"],
            )

    def test_query_window_is_closed_and_bounded(self):
        now = dt.datetime.now(UTC)
        sink.QueryWindow(now, now + dt.timedelta(minutes=15))
        for end in (now, now + dt.timedelta(seconds=901)):
            with self.assertRaises(ValueError):
                sink.QueryWindow(now, end)

    def test_cloud_adapter_paginates_and_never_places_token_in_body(self):
        bodies = []
        responses = [
            json.dumps({"entries": [], "nextPageToken": "next"}).encode(),
            json.dumps({"entries": []}).encode(),
        ]

        def opener(request, _timeout):
            bodies.append(json.loads(request.data))
            self.assertTrue(request.headers["Authorization"].startswith("Bearer "))
            return responses[len(bodies) - 1]

        client = sink.CloudLoggingClient(
            sink.SinkConfig("mealswapp-test", ("projects/mealswapp-test",)),
            opener=opener,
            token_loader=lambda _timeout: "private-access-token",
        )
        now = dt.datetime.now(UTC)
        entries, pages = client.query(
            ["00000000-0000-4000-8000-000000000001"],
            sink.QueryWindow(now, now + dt.timedelta(minutes=1)),
        )
        self.assertEqual((entries, pages), ([], 2))
        self.assertNotIn("private-access-token", json.dumps(bodies))
        self.assertEqual(bodies[1]["pageToken"], "next")

    def test_cloud_adapter_classifies_authorization_and_malformed_results(self):
        def denied(_request, _timeout):
            raise urllib.error.HTTPError("safe", 403, "denied", {}, None)

        client = sink.CloudLoggingClient(
            sink.SinkConfig("mealswapp-test", ("projects/mealswapp-test",)),
            opener=denied,
            token_loader=lambda _timeout: "token",
        )
        now = dt.datetime.now(UTC)
        with self.assertRaises(sink.SinkAuthorizationBlocked):
            client.query(
                ["00000000-0000-4000-8000-000000000001"],
                sink.QueryWindow(now, now + dt.timedelta(minutes=1)),
            )
        malformed = sink.CloudLoggingClient(
            client.config,
            opener=lambda _request, _timeout: b'{"entries":"not-an-array"}',
            token_loader=lambda _timeout: "token",
        )
        with self.assertRaises(sink.SinkMalformed):
            malformed.query(
                ["00000000-0000-4000-8000-000000000001"],
                sink.QueryWindow(now, now + dt.timedelta(minutes=1)),
            )

    def test_cloud_adapter_uses_one_deadline_across_pages(self):
        elapsed = [0.0]
        timeouts = []

        def monotonic():
            return elapsed[0]

        def opener(_request, timeout):
            timeouts.append(timeout)
            elapsed[0] += 0.6
            return json.dumps({"entries": [], "nextPageToken": "next"}).encode()

        client = sink.CloudLoggingClient(
            sink.SinkConfig(
                "mealswapp-test",
                ("projects/mealswapp-test",),
                timeout_seconds=1,
            ),
            opener=opener,
            token_loader=lambda _timeout: "token",
        )
        now = dt.datetime.now(UTC)
        with self.assertRaises(sink.SinkUnavailableBlocked):
            client.query(
                ["00000000-0000-4000-8000-000000000001"],
                sink.QueryWindow(now, now + dt.timedelta(minutes=1)),
                deadline=1.0,
                monotonic=monotonic,
            )
        self.assertEqual(len(timeouts), 2)
        self.assertAlmostEqual(timeouts[1], 0.4)

    def test_delayed_ingestion_polls_then_correlates(self):
        now = dt.datetime.now(UTC)
        expected = sink.load_expected_events(action_receipt(now))[:1]
        client = FakeClient([[], [cloud_entry(expected[0], now)]])
        clock = iter([0.0, 0.0, 0.1, 0.1, 0.1])
        observed, polls, pages = sink.poll_for_events(
            client,
            expected,
            sink.QueryWindow(now - dt.timedelta(seconds=1), now + dt.timedelta(minutes=1)),
            monotonic=lambda: next(clock),
            sleeper=lambda _seconds: None,
        )
        self.assertEqual((len(observed), polls, pages), (1, 2, 2))

    def test_incomplete_ingestion_times_out_without_false_pass(self):
        now = dt.datetime.now(UTC)
        expected = sink.load_expected_events(action_receipt(now))[:1]
        client = FakeClient([[]], timeout=0.01)
        clock = iter([0.0, 0.02])
        with self.assertRaises(sink.SinkUnavailableBlocked):
            sink.poll_for_events(
                client,
                expected,
                sink.QueryWindow(now - dt.timedelta(seconds=1), now + dt.timedelta(minutes=1)),
                monotonic=lambda: next(clock),
                sleeper=lambda _seconds: None,
            )

    def test_redaction_rejects_forbidden_keys_values_and_non_request_ids(self):
        request_id = "00000000-0000-4000-8000-000000000001"
        for value in (
            {"email": "redacted"},
            {"message": "person@example.test"},
            {"message": "https://internal.example.test"},
            {"message": "00000000-0000-4000-8000-000000000002"},
            {"message": "private-search-marker"},
        ):
            with self.assertRaises(sink.RedactionViolation):
                sink.scan_forbidden(value, {request_id}, ["private-search-marker"])
        sink.scan_forbidden({"requestId": request_id, "outcome": "succeeded"}, {request_id}, [])

    def test_sanitizer_rejects_console_text_extra_fields_and_bad_timestamps(self):
        now = dt.datetime.now(UTC)
        expected = sink.load_expected_events(action_receipt(now))[:1]
        window = sink.QueryWindow(now - dt.timedelta(seconds=1), now + dt.timedelta(minutes=1))
        with self.assertRaises(sink.RedactionViolation):
            sink.sanitize_entries([{"textPayload": "{}"}], expected, window)
        entry = cloud_entry(expected[0], now)
        entry["jsonPayload"]["userId"] = "redacted"
        with self.assertRaises(sink.RedactionViolation):
            sink.sanitize_entries([entry], expected, window)
        entry = cloud_entry(expected[0], now)
        entry["receiveTimestamp"] = timestamp(now - dt.timedelta(seconds=1))
        with self.assertRaises(sink.SinkMalformed):
            sink.sanitize_entries([entry], expected, window)

    def test_sanitizer_scans_text_http_and_nested_entry_metadata(self):
        now = dt.datetime.now(UTC)
        expected = sink.load_expected_events(action_receipt(now))[:1]
        window = sink.QueryWindow(now - dt.timedelta(seconds=1), now + dt.timedelta(minutes=1))
        for metadata in (
            {"textPayload": "private-admin@example.test"},
            {"httpRequest": {"requestUrl": "https://private.example.test/admin"}},
            {"labels": {"nested": [{"csrfToken": "private"}]}},
            {"labels": {"displayName": "Alice"}},
            {"textPayload": "INFO user Alice"},
            {"httpRequest": {"userAgent": "private-client/1.0"}},
            {"labels": {"authorization": "Basic private"}},
            {"metadata": [{"display_name": "Alice"}]},
            {"metadata": [{"authorizationHeader": "Bearer private"}]},
            {"http_request": {"user_agent": "private-client/1.0"}},
        ):
            with self.assertRaises(sink.RedactionViolation):
                sink.sanitize_entries(
                    [{**cloud_entry(expected[0], now), **metadata}],
                    expected,
                    window,
                )

    def test_sanitizer_rejects_deeply_nested_key_material_aliases(self):
        now = dt.datetime.now(UTC)
        expected = sink.load_expected_events(action_receipt(now))[:1]
        window = sink.QueryWindow(now - dt.timedelta(seconds=1), now + dt.timedelta(minutes=1))
        aliases = (
            "privateKey",
            "publicKey",
            "encryptionKey",
            "signingKey",
            "clientKey",
            "accessKey",
            "refreshKey",
            "private_key",
            "PUBLIC-KEY",
            "Encryption Key",
            "SIGNING.KEY",
            "client/key",
            "ACCESS_KEY",
            "refresh:key",
        )
        for alias in aliases:
            with self.subTest(alias=alias), self.assertRaises(sink.RedactionViolation):
                sink.sanitize_entries(
                    [
                        {
                            **cloud_entry(expected[0], now),
                            "metadata": {"level1": [{"level2": {alias: "redacted"}}]},
                        }
                    ],
                    expected,
                    window,
                )

    def test_action_receipt_requires_canonical_tuple_provenance_and_ordered_timestamps(self):
        now = dt.datetime.now(UTC)
        receipt = action_receipt(now)
        receipt["events"][2]["action"] = "arbitrary"
        with self.assertRaises(sink.SinkMalformed):
            sink.load_expected_events(receipt, PROVENANCE)
        receipt = action_receipt(now)
        with self.assertRaises(sink.SinkMalformed):
            sink.load_expected_events(receipt, "b" * 48)
        receipt = action_receipt(now)
        receipt["events"][0]["requestId"] = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
        receipt["events"][1]["requestId"] = "AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA"
        with self.assertRaises(sink.SinkMalformed):
            sink.load_expected_events(receipt, PROVENANCE)
        receipt = action_receipt(now)
        receipt["finishedAt"] = timestamp(now - dt.timedelta(seconds=1))
        with self.assertRaises(sink.SinkMalformed):
            sink.load_expected_events(receipt, PROVENANCE)
        expected = sink.load_expected_events(action_receipt(now))
        expected[0] = sink.ExpectedEvent(
            expected[0].category,
            expected[0].request_id,
            "arbitrary",
            expected[0].resource,
            expected[0].outcome,
        )
        with self.assertRaises(sink.SinkMalformed):
            task285.evaluate(expected, [], "sink-evidence.json")

    def test_verify_classifies_reversed_playwright_receipt_as_action_blocked(self):
        now = dt.datetime.now(UTC)
        environment = {
            "MEALSWAPP_TASK285_DEPLOYED_BASE_URL": "https://acceptance.example.test",
            "MEALSWAPP_TASK285_DEPLOYMENT_ACK": "deployed-test-with-centralized-log-sink",
            "MEALSWAPP_TASK285_APPROVED_DEPLOYED_HOST": "acceptance.example.test",
            "MEALSWAPP_TASK285_GCP_PROJECT": "mealswapp-test",
            "MEALSWAPP_TASK285_LOG_RESOURCE_NAMES": "projects/mealswapp-test",
        }

        def reversed_receipt(
            destination: Path,
            _environment: dict[str, str],
            _base_url: str,
            provenance: str,
        ) -> Path:
            receipt = action_receipt(now)
            receipt.update(
                provenance=provenance,
                finishedAt=timestamp(now - dt.timedelta(seconds=1)),
            )
            path = destination / "actions.json"
            path.write_text(json.dumps(receipt))
            return path

        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaisesRegex(
                sink.SinkUnavailableBlocked,
                "action receipt is invalid",
            ):
                task285.verify(
                    Path(directory),
                    environment,
                    _action_producer=reversed_receipt,
                    _resolver=lambda _host: ["8.8.8.8"],
                )

    def test_event_contract_requires_one_exact_event_and_all_admin_outcomes(self):
        now = dt.datetime.now(UTC)
        expected = sink.load_expected_events(action_receipt(now))
        observed = [
            sink.SafeEvent(
                item.request_id,
                timestamp(now),
                timestamp(now + dt.timedelta(seconds=1)),
                item.action,
                item.resource,
                item.outcome,
            )
            for item in expected
        ]
        self.assertEqual(sink.verify_event_contract(expected, observed), [])
        self.assertTrue(sink.verify_event_contract(expected, observed[:-1]))

    def test_retention_probe_requires_ninety_day_safe_correlation(self):
        old = dt.datetime.now(UTC) - dt.timedelta(days=91)
        event, window = task285.retention_probe(
            {
                "MEALSWAPP_TASK285_RETENTION_REQUEST_ID": "00000000-0000-4000-8000-000000000099",
                "MEALSWAPP_TASK285_RETENTION_OCCURRED_AT": timestamp(old),
            }
        )
        self.assertEqual(event.outcome, "retained")
        self.assertEqual((window.end - window.start).total_seconds(), 120)
        with self.assertRaises(sink.SinkUnavailableBlocked):
            task285.retention_probe({})

    def test_missing_configuration_emits_every_blocked_result_without_credentials(self):
        with tempfile.TemporaryDirectory() as directory:
            destination = Path(directory)
            task285.write_blocked_evidence(destination, "DEPLOYMENT-SINK-BLOCKED")
            results = task285.blocked_results(task285.DEPLOYMENT_ROOT)
            self.assertEqual({item["criterionId"] for item in results}, set(task285.CRITERIA))
            self.assertTrue(all(item["status"] == "BLOCKED" for item in results))
            self.assertNotIn("credential", (destination / "sink-evidence.json").read_text().lower())

    def test_exit_semantics_distinguish_pass_fail_and_blocked(self):
        self.assertEqual(task285.result_exit_code([{"status": "PASS"}]), 0)
        self.assertEqual(task285.result_exit_code([{"status": "PASS"}, {"status": "FAIL"}]), 1)
        self.assertEqual(task285.result_exit_code([{"status": "PASS"}, {"status": "BLOCKED"}]), 2)
        self.assertEqual(task285.result_exit_code([{"status": "FAIL"}, {"status": "BLOCKED"}]), 1)

    def test_complete_action_and_sink_contract_maps_every_criterion_to_pass(self):
        now = dt.datetime.now(UTC)
        expected = sink.load_expected_events(action_receipt(now))
        observed = [
            sink.SafeEvent(
                item.request_id,
                timestamp(now),
                timestamp(now + dt.timedelta(seconds=1)),
                item.action,
                item.resource,
                item.outcome,
            )
            for item in expected
        ]
        results = task285.evaluate(expected, observed, "sink-evidence.json")
        self.assertEqual(len(results), len(task285.CRITERIA))
        self.assertTrue(all(item["status"] == "PASS" for item in results))

    def test_complete_verifier_writes_only_sanitized_sink_projection(self):
        now = dt.datetime.now(UTC)
        receipt = action_receipt(now)
        expected = sink.load_expected_events(receipt)
        retained_at = now - dt.timedelta(days=91)
        retained = sink.ExpectedEvent(
            "retention_probe",
            "00000000-0000-4000-8000-000000000099",
            "logging_retention",
            "centralized_log",
            "retained",
        )
        client = FakeClient(
            [
                [cloud_entry(item, now) for item in expected],
                [cloud_entry(retained, retained_at)],
            ]
        )
        environment = {
            "MEALSWAPP_TASK285_DEPLOYED_BASE_URL": "https://acceptance.example.test",
            "MEALSWAPP_TASK285_DEPLOYMENT_ACK": "deployed-test-with-centralized-log-sink",
            "MEALSWAPP_TASK285_APPROVED_DEPLOYED_HOST": "acceptance.example.test",
            "MEALSWAPP_TASK285_GCP_PROJECT": "mealswapp-test",
            "MEALSWAPP_TASK285_LOG_RESOURCE_NAMES": "projects/mealswapp-test",
            "MEALSWAPP_TASK285_RETENTION_REQUEST_ID": retained.request_id,
            "MEALSWAPP_TASK285_RETENTION_OCCURRED_AT": timestamp(retained_at),
            "MEALSWAPP_TASK285_ADMIN_EMAIL": "private-admin@example.test",
            "MEALSWAPP_TASK285_ADMIN_PASSWORD": "private-password",
            "MEALSWAPP_TASK285_EXTERNAL_QUERY": "private-query",
            "MEALSWAPP_TASK285_DEPENDENCY_QUERY": "private-dependency",
            "MEALSWAPP_TASK285_USER_LOOKUP": "private-user@example.test",
            "MEALSWAPP_TASK285_MARKER": "private-marker",
        }
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            def produce_actions(
                destination: Path,
                _environment: dict[str, str],
                _base_url: str,
                provenance: str,
            ) -> Path:
                actions = destination / "actions.json"
                actions.write_text(json.dumps({**receipt, "provenance": provenance}))
                return actions

            results, status = task285.verify(
                root,
                environment,
                client=client,
                _action_producer=produce_actions,
                _resolver=lambda _host: ["8.8.8.8"],
            )
            artifact = (root / "sink-evidence.json").read_text()
        self.assertEqual(status, "PASS")
        self.assertTrue(all(item["status"] == "PASS" for item in results))
        for private in (
            "private-admin",
            "private-password",
            "private-query",
            "private-dependency",
            "private-user",
            "private-marker",
            "mealswapp-test",
        ):
            self.assertNotIn(private, artifact)

    def test_playwright_process_failures_are_safe_blockers(self):
        environment = {"MEALSWAPP_TASK285_MARKER": "safe-marker"}
        with tempfile.TemporaryDirectory() as directory:
            for failure in (
                subprocess.TimeoutExpired(["bunx"], 1),
                OSError("unsafe local diagnostic"),
            ):
                with mock.patch.object(task285.subprocess, "run", side_effect=failure):
                    with self.assertRaisesRegex(
                        sink.SinkUnavailableBlocked,
                        "deployed action production did not complete",
                    ):
                        task285.run_playwright(
                            Path(directory),
                            environment,
                            "https://acceptance.example.test",
                            PROVENANCE,
                        )

    def test_main_turns_input_validation_and_report_failures_into_complete_blocked_reports(self):
        for argv, verification_failure, report_failure in (
            (["--actions-file", "forbidden.json"], None, OSError("spawn")),
            ([], ValueError("reversed input"), OSError("spawn")),
            ([], RecursionError("malformed nesting"), subprocess.TimeoutExpired(["report"], 1)),
        ):
            with self.subTest(argv=argv, failure=type(verification_failure).__name__):
                with tempfile.TemporaryDirectory() as directory:
                    root = Path(directory)
                    with (
                        mock.patch.object(task285, "ROOT", root),
                        mock.patch.object(task285, "ARTIFACT_ROOT", root / "artifacts"),
                        mock.patch.object(task285, "REPORT_ROOT", root / "reports"),
                        mock.patch.object(task285, "finalize_report", side_effect=report_failure),
                        mock.patch.object(task285, "verify", side_effect=verification_failure)
                        if verification_failure
                        else mock.patch.object(task285, "verify"),
                    ):
                        self.assertEqual(task285.main(argv), 2)
                    reports = list((root / "reports").glob("task285-*/report.json"))
                    self.assertEqual(len(reports), 1)
                    report = json.loads(reports[0].read_text())
                    self.assertEqual(report["status"], "BLOCKED")
                    self.assertEqual(report["findingIds"], ["P08-FIND-285-001"])
                    self.assertEqual(len(report["results"]), 7)
                    self.assertTrue(all(item["status"] == "BLOCKED" for item in report["results"]))


if __name__ == "__main__":
    unittest.main()
