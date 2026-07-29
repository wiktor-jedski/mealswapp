"""Task 290 tests for the authenticated micronutrient vocabulary operator."""

# Implements DESIGN-009 AdminController micronutrient vocabulary operator verification.

from __future__ import annotations

import contextlib
import importlib.util
import io
import json
import sys
import threading
import unittest
import urllib.error
from email.message import Message
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from unittest import mock

SCRIPTS = Path(__file__).parent
sys.path.insert(0, str(SCRIPTS))
SPEC = importlib.util.spec_from_file_location(
    "manage_micronutrient_vocabulary",
    SCRIPTS / "manage-micronutrient-vocabulary.py",
)
OPERATOR = importlib.util.module_from_spec(SPEC)
assert SPEC.loader
SPEC.loader.exec_module(OPERATOR)

REQUEST_ID = "00000000-0000-4000-8000-000000000099"
ENTRIES = [
    {"key": "Iron", "displayName": "Iron", "unit": "mg", "active": True},
    {"key": "VitaminK", "displayName": "Vitamin K", "unit": "mcg", "active": False},
]


def envelope(field: str, value: object) -> bytes:
    """Build one contract-conforming gateway envelope."""
    return json.dumps(
        {"status": "ok", "requestId": REQUEST_ID, "data": {field: value}},
        separators=(",", ":"),
    ).encode()


class VocabularyHandler(BaseHTTPRequestHandler):
    """Provide a controllable fake authenticated administration boundary."""

    requests: list[dict[str, object]] = []
    list_status = 200
    list_body = envelope("micronutrients", ENTRIES)
    mutation_statuses: list[int] = []
    mutation_body: bytes | None = None
    auth_status = 200

    def record(self, body: bytes) -> None:
        type(self).requests.append({
            "method": self.command,
            "path": self.path,
            "body": body,
            "csrf": self.headers.get("X-CSRF-Token"),
            "cookie": self.headers.get("Cookie"),
        })

    def send(self, status: int, body: bytes = b"", headers: dict[str, str] | None = None) -> None:
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        for key, value in (headers or {}).items():
            self.send_header(key, value)
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self) -> None:
        self.record(b"")
        if self.path == "/api/v1/auth/csrf-token":
            self.send(200, b'{"data":{"csrfToken":"csrf-secret"}}')
            return
        if self.path == "/api/v1/admin/micronutrients":
            self.send(type(self).list_status, type(self).list_body)
            return
        self.send(404)

    def do_POST(self) -> None:
        self.handle_write()

    def do_PUT(self) -> None:
        self.handle_write()

    def handle_write(self) -> None:
        body = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        self.record(body)
        if self.path == "/api/v1/auth/login":
            if b"admin@example.test" not in body or b"password-secret" not in body:
                self.send(401)
                return
            self.send(
                type(self).auth_status,
                b'{"status":"ok"}',
                {"Set-Cookie": "session=memory-only; Path=/; HttpOnly; SameSite=Strict"},
            )
            return
        statuses = type(self).mutation_statuses
        status = statuses.pop(0) if statuses else 201 if self.path == "/api/v1/admin/micronutrients" else 200
        if status == 429:
            self.send(status, b'{"raw":"database://secret"}', {"Retry-After": "0"})
            return
        entry = dict(ENTRIES[1])
        if self.path == "/api/v1/admin/micronutrients":
            entry = {"key": "VitaminK", "displayName": "Vitamin K", "unit": "mcg", "active": True}
        elif self.path.endswith("/display-name"):
            entry["displayName"] = "Vitamin K1"
        elif self.path.endswith("/unit"):
            entry["unit"] = "mg"
        elif self.path.endswith("/deactivate"):
            entry["active"] = False
        elif self.path.endswith("/reactivate"):
            entry["active"] = True
        response_body = type(self).mutation_body or envelope("micronutrient", entry)
        self.send(status, response_body)

    def log_message(self, *_args: object) -> None:
        return


class VocabularyOperatorTests(unittest.TestCase):
    """Exercise the complete CLI contract against the fake HTTP boundary."""

    def setUp(self) -> None:
        VocabularyHandler.requests = []
        VocabularyHandler.list_status = 200
        VocabularyHandler.list_body = envelope("micronutrients", ENTRIES)
        VocabularyHandler.mutation_statuses = []
        VocabularyHandler.mutation_body = None
        VocabularyHandler.auth_status = 200
        self.server = ThreadingHTTPServer(("127.0.0.1", 0), VocabularyHandler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.base_url = f"http://127.0.0.1:{self.server.server_port}"

    def tearDown(self) -> None:
        self.server.shutdown()
        self.server.server_close()
        self.thread.join()

    def run_operator(self, command: list[str]) -> tuple[int, str, str]:
        """Run with interactive-only credentials and captured safe streams."""
        stdout, stderr = io.StringIO(), io.StringIO()
        with (
            mock.patch("builtins.input", return_value="admin@example.test") as email_prompt,
            mock.patch("global_catalog_session.getpass.getpass", return_value="password-secret") as password_prompt,
            contextlib.redirect_stdout(stdout),
            contextlib.redirect_stderr(stderr),
        ):
            status = OPERATOR.run([
                "--environment", "development",
                "--base-url", self.base_url,
                *command,
            ])
        email_prompt.assert_called_once()
        password_prompt.assert_called_once()
        return status, stdout.getvalue(), stderr.getvalue()

    def mutation_requests(self) -> list[dict[str, object]]:
        """Return requests after login and CSRF acquisition."""
        return [
            request for request in VocabularyHandler.requests
            if request["path"] not in {"/api/v1/auth/login", "/api/v1/auth/csrf-token"}
        ]

    def test_list_is_complete_deterministic_and_safe(self) -> None:
        status, stdout, stderr = self.run_operator(["list"])
        self.assertEqual(status, 0)
        self.assertEqual(
            stdout,
            "key\tdisplay_name\tunit\tactive\n"
            "Iron\tIron\tmg\ttrue\n"
            "VitaminK\tVitamin K\tmcg\tfalse\n",
        )
        self.assertEqual(stderr, "")
        self.assertEqual(self.mutation_requests()[0]["method"], "GET")
        for secret in (
            "admin@example.test", "password-secret", "csrf-secret",
            "session=memory-only", "database://secret",
        ):
            self.assertNotIn(secret, stdout + stderr)

    def test_add_uses_cookie_csrf_and_exact_closed_body(self) -> None:
        status, stdout, stderr = self.run_operator([
            "add", "--key", "VitaminK", "--display-name", "Vitamin K", "--unit", "mcg",
        ])
        self.assertEqual(status, 0)
        self.assertIn("succeeded command=add key=VitaminK unit=mcg active=true", stdout)
        self.assertEqual(stderr, "")
        mutation = self.mutation_requests()[0]
        self.assertEqual(mutation["method"], "POST")
        self.assertEqual(mutation["path"], "/api/v1/admin/micronutrients")
        self.assertEqual(
            json.loads(mutation["body"]),
            {"key": "VitaminK", "displayName": "Vitamin K", "unit": "mcg"},
        )
        self.assertEqual(mutation["csrf"], "csrf-secret")
        self.assertEqual(mutation["cookie"], "session=memory-only")

    def test_all_update_and_lifecycle_routes_have_exact_requests(self) -> None:
        cases = [
            (
                ["update-display-name", "--key", "VitaminK", "--display-name", "Vitamin K1"],
                "PUT", "/api/v1/admin/micronutrients/VitaminK/display-name",
                {"displayName": "Vitamin K1"},
            ),
            (
                ["update-unit", "--key", "VitaminK", "--unit", "mg"],
                "PUT", "/api/v1/admin/micronutrients/VitaminK/unit", {"unit": "mg"},
            ),
            (
                ["deactivate", "--key", "VitaminK"],
                "POST", "/api/v1/admin/micronutrients/VitaminK/deactivate", None,
            ),
            (
                ["reactivate", "--key", "VitaminK"],
                "POST", "/api/v1/admin/micronutrients/VitaminK/reactivate", None,
            ),
        ]
        for arguments, method, path, body in cases:
            with self.subTest(arguments=arguments):
                VocabularyHandler.requests = []
                status, stdout, stderr = self.run_operator(arguments)
                self.assertEqual((status, stderr), (0, ""))
                self.assertIn("succeeded", stdout)
                mutation = self.mutation_requests()[0]
                self.assertEqual((mutation["method"], mutation["path"]), (method, path))
                self.assertEqual(json.loads(mutation["body"]) if mutation["body"] else None, body)
                self.assertEqual(mutation["csrf"], "csrf-secret")

    def test_dry_run_authenticates_and_never_mutates(self) -> None:
        for arguments in (
            ["add", "--key", "VitaminE", "--display-name", "Vitamin E", "--unit", "mg", "--dry-run"],
            ["update-display-name", "--key", "VitaminK", "--display-name", "Vitamin K1", "--dry-run"],
            ["update-unit", "--key", "VitaminK", "--unit", "mg", "--dry-run"],
            ["deactivate", "--key", "VitaminK", "--dry-run"],
            ["reactivate", "--key", "VitaminK", "--dry-run"],
        ):
            with self.subTest(arguments=arguments):
                VocabularyHandler.requests = []
                status, stdout, stderr = self.run_operator(arguments)
                self.assertEqual((status, stderr), (0, ""))
                self.assertIn("dry_run=true", stdout)
                requests = self.mutation_requests()
                self.assertEqual([(request["method"], request["path"]) for request in requests], [
                    ("GET", "/api/v1/admin/micronutrients"),
                ])

        status, stdout, stderr = self.run_operator([
            "add", "--key", "Iron", "--display-name", "Iron", "--unit", "mg", "--dry-run",
        ])
        self.assertEqual(status, 2)
        self.assertEqual(stdout, "")
        self.assertEqual(stderr, "micronutrient key already exists\n")

    def test_dry_run_missing_key_and_permanent_failures_exit_nonzero(self) -> None:
        status, stdout, stderr = self.run_operator([
            "deactivate", "--key", "VitaminE", "--dry-run",
        ])
        self.assertEqual(status, 2)
        self.assertEqual(stdout, "")
        self.assertEqual(stderr, "micronutrient key was not found\n")

        VocabularyHandler.mutation_statuses = [400]
        status, stdout, stderr = self.run_operator([
            "update-unit", "--key", "VitaminK", "--unit", "mg",
        ])
        self.assertEqual(status, 2)
        self.assertEqual(stdout, "")
        self.assertEqual(stderr, "mutation failed permanently\n")

    def test_authentication_and_authorization_abort_without_retry(self) -> None:
        VocabularyHandler.auth_status = 401
        status, stdout, stderr = self.run_operator(["list"])
        self.assertEqual(status, 2)
        self.assertEqual(stdout, "")
        self.assertEqual(stderr, "authentication failed\n")
        self.assertEqual(len(VocabularyHandler.requests), 1)

        VocabularyHandler.auth_status = 200
        VocabularyHandler.list_status = 403
        VocabularyHandler.requests = []
        status, stdout, stderr = self.run_operator(["list"])
        self.assertEqual(status, 2)
        self.assertEqual(stdout, "")
        self.assertEqual(stderr, "authorization failed\n")
        self.assertEqual(len(self.mutation_requests()), 1)

    def test_in_use_conflict_is_safe_nonzero_and_not_retried(self) -> None:
        VocabularyHandler.mutation_statuses = [409]
        VocabularyHandler.mutation_body = (
            b'{"email":"admin@example.test","body":"database://secret","csrf":"csrf-secret"}'
        )
        status, stdout, stderr = self.run_operator([
            "deactivate", "--key", "VitaminK",
        ])
        self.assertEqual(status, 2)
        self.assertEqual(stdout, "")
        self.assertEqual(stderr, "mutation blocked by an in-use or concurrent conflict\n")
        self.assertEqual(len(self.mutation_requests()), 1)
        for secret in ("admin@example.test", "database://secret", "csrf-secret"):
            self.assertNotIn(secret, stdout + stderr)

    def test_5xx_and_retry_after_reuse_identical_request(self) -> None:
        VocabularyHandler.mutation_statuses = [500, 429, 200]
        with mock.patch.object(OPERATOR.time, "sleep") as sleep:
            status, stdout, stderr = self.run_operator([
                "update-unit", "--key", "VitaminK", "--unit", "mg",
            ])
        self.assertEqual((status, stderr), (0, ""))
        self.assertIn("succeeded", stdout)
        requests = [request for request in self.mutation_requests() if request["method"] != "GET"]
        self.assertEqual(len(requests), 3)
        self.assertEqual(
            {(request["method"], request["path"], request["body"], request["csrf"]) for request in requests},
            {("PUT", "/api/v1/admin/micronutrients/VitaminK/unit", b'{"unit":"mg"}', "csrf-secret")},
        )
        sleep.assert_called_once_with(0.0)

    def test_transport_ambiguity_is_bounded_and_identical(self) -> None:
        client = OPERATOR.VocabularyClient("http://127.0.0.1:1", sleep=lambda _delay: None)
        client.csrf = "csrf-secret"
        with mock.patch.object(
            client,
            "request",
            side_effect=urllib.error.URLError("database://secret"),
        ) as request, mock.patch.object(client, "list_entries", return_value=[ENTRIES[1]]) as listing:
            entry = client.mutate(
                "POST", "/api/v1/admin/micronutrients/VitaminK/reactivate", None, 200,
                {"key": "VitaminK", "active": False},
            )
        self.assertEqual(entry, ENTRIES[1])
        self.assertEqual(request.call_count, 1)
        listing.assert_called_once()

        with mock.patch.object(
            client, "request", side_effect=urllib.error.URLError("database://secret"),
        ) as request, mock.patch.object(client, "list_entries", return_value=[]) as listing:
            with self.assertRaisesRegex(OPERATOR.OperatorError, "^operator request failed$"):
                client.mutate(
                    "POST", "/api/v1/admin/micronutrients/VitaminK/reactivate", None, 200,
                    {"key": "VitaminK", "active": True},
                )
        self.assertEqual(request.call_count, OPERATOR.MAX_ATTEMPTS)
        self.assertEqual(request.call_args_list, [request.call_args_list[0]] * OPERATOR.MAX_ATTEMPTS)
        self.assertEqual(listing.call_count, OPERATOR.MAX_ATTEMPTS)

    def test_invalid_responses_and_ordering_never_print_raw_body(self) -> None:
        invalid = [
            {"status": "ok", "requestId": REQUEST_ID, "data": {"micronutrients": list(reversed(ENTRIES))}},
            {"status": "ok", "requestId": REQUEST_ID, "data": {"micronutrients": [ENTRIES[0], ENTRIES[0]]}},
            {"status": "ok", "requestId": REQUEST_ID, "data": {"micronutrients": [
                {**ENTRIES[0], "displayName": "secret\nstack"},
            ]}},
            {"status": "ok", "requestId": REQUEST_ID, "data": {"micronutrients": [
                {**ENTRIES[0], "databaseUrl": "database://secret"},
            ]}},
            {"status": "ok", "requestId": "", "data": {"micronutrients": ENTRIES}},
        ]
        for body in invalid:
            with self.subTest(body=body):
                VocabularyHandler.list_body = json.dumps(body).encode()
                status, stdout, stderr = self.run_operator(["list"])
                self.assertEqual(status, 2)
                self.assertEqual(stdout, "")
                self.assertEqual(stderr, "API response is invalid\n")
                self.assertNotIn("secret", stdout + stderr)
                self.assertNotIn("database://", stdout + stderr)

        VocabularyHandler.list_body = (
            b'{"status":"ok","status":"ok","requestId":"' + REQUEST_ID.encode()
            + b'","data":{"micronutrients":[]}}'
        )
        self.assertEqual(self.run_operator(["list"])[0], 2)
        VocabularyHandler.list_body = b" " * (OPERATOR.MAX_RESPONSE_BYTES + 1)
        self.assertEqual(self.run_operator(["list"])[0], 2)

    def test_mutation_response_must_match_requested_authoritative_state(self) -> None:
        VocabularyHandler.mutation_body = envelope("micronutrient", ENTRIES[0])
        status, stdout, stderr = self.run_operator([
            "update-unit", "--key", "VitaminK", "--unit", "mg",
        ])
        self.assertEqual(status, 2)
        self.assertEqual(stdout, "")
        self.assertEqual(stderr, "API response is invalid\n")

        VocabularyHandler.mutation_body = b'{"status":"ok","requestId":"bad"}'
        VocabularyHandler.list_body = envelope("micronutrients", [
            {"key": "VitaminK", "displayName": "Vitamin K", "unit": "mg", "active": False},
        ])
        status, stdout, stderr = self.run_operator([
            "update-unit", "--key", "VitaminK", "--unit", "mg",
        ])
        self.assertEqual(status, 0)
        self.assertIn("succeeded command=update-unit key=VitaminK unit=mg active=false", stdout)
        self.assertEqual(stderr, "")

    def test_explicit_environment_target_and_argument_validation_precede_credentials(self) -> None:
        invalid_targets = [
            ("development", "http://example.test", False),
            ("staging", "http://127.0.0.1:8080", False),
            ("production", "https://api.example.test", False),
            ("staging", "https://api.example.test", True),
            ("development", "http://user:secret@127.0.0.1:8080", False),
            ("development", "http://127.0.0.1:8080/api?raw=1", False),
            ("development", "http://127.0.0.1:8080/health", False),
            ("development", " http://127.0.0.1:8080", False),
        ]
        for environment, target, confirmation in invalid_targets:
            with self.subTest(environment=environment, target=target):
                arguments = ["--environment", environment, "--base-url", target]
                if confirmation:
                    arguments.append("--confirm-production")
                arguments.append("list")
                with mock.patch("builtins.input") as prompt:
                    stdout, stderr = io.StringIO(), io.StringIO()
                    with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
                        status = OPERATOR.run(arguments)
                self.assertEqual(status, 2)
                prompt.assert_not_called()
                self.assertEqual(stdout.getvalue(), "")
                self.assertNotIn("secret", stderr.getvalue())

        self.assertEqual(
            OPERATOR.validate_target("production", "https://api.example.test/", True),
            "https://api.example.test",
        )
        for arguments in (
            ["--environment", "development", "--base-url", self.base_url],
            ["--environment", "development", "--base-url", self.base_url, "add", "--key", "Na", "--display-name", "Sodium", "--unit", "mg"],
            ["--environment", "development", "--base-url", self.base_url, "add", "--key", "VitaminE", "--display-name", " bad", "--unit", "mg"],
            ["--environment", "development", "--base-url", self.base_url, "update-unit", "--key", "VitaminE", "--unit", "oz"],
        ):
            with self.subTest(arguments=arguments), self.assertRaises(SystemExit):
                OPERATOR.run(arguments)

    def test_retry_after_and_request_timeout_are_bounded(self) -> None:
        self.assertEqual(OPERATOR.retry_after_seconds("5"), 5)
        self.assertIsNone(OPERATOR.retry_after_seconds("5.1"))
        self.assertIsNone(OPERATOR.retry_after_seconds("-1"))
        self.assertIsNone(OPERATOR.retry_after_seconds("not-a-date"))

        class Response:
            def __init__(self, body: bytes):
                self.status = 200
                self.headers = Message()
                self.body = body
                self.read_limits: list[int] = []

            def read(self, _limit: int = -1) -> bytes:
                self.read_limits.append(_limit)
                return self.body

            def close(self) -> None:
                return None

        session = OPERATOR.VocabularyClient("http://127.0.0.1:8080")
        with mock.patch.object(
            session.opener, "open", return_value=Response(envelope("micronutrients", [])),
        ) as opened:
            response = session.request("GET", "/api/v1/admin/micronutrients")
        self.assertEqual(response.status, 200)
        self.assertEqual(opened.call_args.kwargs["timeout"], 30)

        login_response = Response(b'{"status":"ok"}')
        csrf_response = Response(b'{"data":{"csrfToken":"csrf-secret"}}')
        with mock.patch.object(
            session.opener, "open", side_effect=[login_response, csrf_response],
        ) as opened:
            session.login("admin@example.test", "password-secret")
        self.assertEqual(opened.call_count, 2)
        self.assertEqual(login_response.read_limits, [OPERATOR.MAX_RESPONSE_BYTES + 1])
        self.assertEqual(csrf_response.read_limits, [OPERATOR.MAX_RESPONSE_BYTES + 1])

    def test_operator_has_no_database_or_process_execution_boundary(self) -> None:
        source = (SCRIPTS / "manage-micronutrient-vocabulary.py").read_text(encoding="utf-8")
        for forbidden in (
            "psycopg", "psycopg2", "asyncpg", "sqlalchemy", "sqlite3",
            "DATABASE_URL", "postgresql://", "subprocess", "os.system",
        ):
            self.assertNotIn(forbidden, source)
        self.assertIn('"/api/v1/admin/micronutrients"', source)


if __name__ == "__main__":
    unittest.main()
