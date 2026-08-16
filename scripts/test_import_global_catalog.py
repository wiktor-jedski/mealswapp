"""Focused tests for the Task 277 global-catalog operator."""

from __future__ import annotations

import importlib.util
import io
import json
import sys
import tempfile
import threading
import unittest
from contextlib import redirect_stderr, redirect_stdout
from email.message import Message
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from unittest import mock

MODULE_PATH = Path(__file__).with_name("import-global-catalog.py")
sys.path.insert(0, str(MODULE_PATH.parent))
SPEC = importlib.util.spec_from_file_location("import_global_catalog", MODULE_PATH)
assert SPEC and SPEC.loader
IMPORTER = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = IMPORTER
SPEC.loader.exec_module(IMPORTER)


def valid_entry(key: str = "stable-item-key-0001") -> dict:
    return {
        "idempotencyKey": key,
        "item": {
            "name": "Tofu",
            "physicalState": "solid",
            "prepTimeMinutes": 5,
            "averageUnitWeightGrams": 100,
            "macrosPer100": {"protein": 18, "carbohydrates": 3, "fat": 9},
            "micros": {"Iron": 2},
            "foodCategoryNames": ["Protein"],
            "culinaryRoleNames": ["Main"],
            "allergenKeys": ["peanut"],
        },
    }


class ValidationTests(unittest.TestCase):
    def test_schema_duplicate_keys_key_rules_and_limit(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory, "catalog.json")
            path.write_text('{"schema":"mealswapp.global-catalog.v1","schema":"x","items":[]}')
            with self.assertRaises(IMPORTER.DuplicateJSONKey):
                IMPORTER.load_document(path)
        with self.assertRaises(IMPORTER.CatalogError):
            IMPORTER.validate_document({"items": [valid_entry("short")]})
        with self.assertRaises(IMPORTER.CatalogError):
            IMPORTER.validate_document({"items": [valid_entry(), valid_entry()]})
        with self.assertRaises(IMPORTER.CatalogError):
            IMPORTER.load_document(self.write({"schema": IMPORTER.SCHEMA, "items": [valid_entry(str(i).zfill(8)) for i in range(501)]}))

    def test_metric_micronutrient_allergen_and_classification_rules(self) -> None:
        cases = []
        liquid = valid_entry()
        liquid["item"] |= {"physicalState": "liquid", "averageServingVolumeMilliliters": 250}
        cases.append(liquid)
        solid_density = valid_entry()
        solid_density["item"]["densityGramsPerMilliliter"] = 1
        cases.append(solid_density)
        unknown_micro = valid_entry()
        unknown_micro["item"]["micros"] = {"Unknown": 1}
        cases.append(unknown_micro)
        bad_allergen = valid_entry()
        bad_allergen["item"]["allergenKeys"] = ["Dairy"]
        cases.append(bad_allergen)
        mixed_classification = valid_entry()
        mixed_classification["item"]["foodCategoryIds"] = ["aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"]
        cases.append(mixed_classification)
        for entry in cases:
            with self.subTest(entry=entry):
                with self.assertRaises(IMPORTER.CatalogError):
                    IMPORTER.validate_document({"items": [entry]})

    def test_valid_metric_liquid_and_uuid_alternative(self) -> None:
        entry = valid_entry()
        entry["item"].pop("foodCategoryNames")
        entry["item"]["foodCategoryIds"] = ["aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"]
        entry["item"] |= {
            "physicalState": "liquid",
            "densityGramsPerMilliliter": 1.03,
            "densitySourceKind": "manual",
            "averageServingVolumeMilliliters": 250,
        }
        IMPORTER.validate_document({"items": [entry]})

    def test_malformed_runtime_types_and_nonfinite_numbers_raise_only_catalog_errors(self) -> None:
        malformed = []
        unhashable_state = valid_entry()
        unhashable_state["item"]["physicalState"] = []
        malformed.append(unhashable_state)
        huge_number = valid_entry()
        huge_number["item"]["macrosPer100"]["protein"] = 10**10000
        malformed.append(huge_number)
        unhashable_allergen = valid_entry()
        unhashable_allergen["item"]["allergenKeys"] = [[]]
        malformed.append(unhashable_allergen)
        unhashable_classification = valid_entry()
        unhashable_classification["item"]["foodCategoryNames"] = [[]]
        malformed.append(unhashable_classification)
        liquid_bad_density = valid_entry()
        liquid_bad_density["item"] |= {"physicalState": "liquid", "densityGramsPerMilliliter": []}
        malformed.append(liquid_bad_density)
        liquid_bad_provider = valid_entry()
        liquid_bad_provider["item"] |= {
            "physicalState": "liquid",
            "densityGramsPerMilliliter": 1,
            "densitySourceKind": "imported",
            "densitySourceProvider": [],
            "densitySourceFoodId": {},
        }
        malformed.append(liquid_bad_provider)
        for index, entry in enumerate(malformed):
            with self.subTest(index=index):
                with self.assertRaises(IMPORTER.CatalogError):
                    IMPORTER.validate_document({"items": [entry]})
        for malformed_document in (None, [], {}, {"items": None}):
            with self.subTest(document=malformed_document):
                with self.assertRaises(IMPORTER.CatalogError):
                    IMPORTER.validate_document(malformed_document)
        with tempfile.TemporaryDirectory() as directory:
            for index, constant in enumerate(("NaN", "Infinity", "-Infinity")):
                path = Path(directory, f"catalog-{index}.json")
                path.write_text(f'{{"schema":"mealswapp.global-catalog.v1","items":[{{"idempotencyKey":"stable-key","item":{{"name":"x","physicalState":"solid","macrosPer100":{{"protein":{constant},"carbohydrates":0,"fat":0}},"micros":{{}},"allergenKeys":[]}}}}]}}')
                with self.subTest(constant=constant):
                    with self.assertRaises(IMPORTER.CatalogError):
                        IMPORTER.validate_document(IMPORTER.load_document(path))

    def test_optional_fields_are_typed_and_bounded_before_login_or_mutation(self) -> None:
        malformed = []
        for field, value in (
            ("prepTimeMinutes", []),
            ("prepTimeMinutes", True),
            ("prepTimeMinutes", 100_000_000),
            ("averageUnitWeightGrams", "100"),
            ("averageUnitWeightGrams", 100_000_000),
            ("averageServingVolumeMilliliters", {}),
            ("averageServingVolumeMilliliters", 100_000_000),
            ("densityGramsPerMilliliter", []),
            ("densityGramsPerMilliliter", 100_000_000),
            ("densitySourceProvider", []),
            ("densitySourceProvider", "x" * 201),
            ("densitySourceFoodId", {}),
            ("densitySourceFoodId", "x" * 201),
            ("densitySourceKind", []),
            ("foodCategoryNames", "Protein"),
            ("foodCategoryNames", ["x" * 121]),
            ("culinaryRoleNames", [[]]),
            ("foodCategoryIds", ["aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"] * 101),
            ("culinaryRoleIds", [[]]),
            ("imageUrl", []),
            ("imageUrl", "x" * 2049),
            ("imageUrl", "ftp://example.test/image.png"),
        ):
            entry = valid_entry()
            if field.startswith("density") or field == "averageServingVolumeMilliliters":
                entry["item"] |= {
                    "physicalState": "liquid",
                    "densityGramsPerMilliliter": 1,
                    "densitySourceKind": "manual",
                }
            if field == "foodCategoryIds":
                entry["item"].pop("foodCategoryNames")
            if field == "culinaryRoleIds":
                entry["item"].pop("culinaryRoleNames")
            entry["item"][field] = value
            malformed.append((field, entry))

        for field, entry in malformed:
            with self.subTest(field=field, value=entry["item"][field]):
                catalog = self.write({"schema": IMPORTER.SCHEMA, "items": [entry]})
                stderr = io.StringIO()
                with (
                    mock.patch("builtins.input") as login_prompt,
                    mock.patch.object(IMPORTER.getpass, "getpass") as password_prompt,
                    mock.patch.object(IMPORTER, "APIClient") as client,
                    redirect_stderr(stderr),
                ):
                    code = IMPORTER.run([str(catalog), "--base-url", "https://example.test"])
                self.assertEqual(code, 2)
                self.assertRegex(stderr.getvalue(), r"^item 1: [^\n]+\n$")
                login_prompt.assert_not_called()
                password_prompt.assert_not_called()
                client.assert_not_called()

    def test_json_integer_digit_overflow_is_a_safe_prelogin_catalog_error(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory, "catalog.json")
            path.write_text(
                '{"schema":"mealswapp.global-catalog.v1","items":[{"idempotencyKey":"stable-key",'
                '"item":{"name":"x","physicalState":"solid","prepTimeMinutes":'
                + "9" * 10_000
                + ',"macrosPer100":{"protein":0,"carbohydrates":0,"fat":0},"micros":{},"allergenKeys":[]}}]}'
            )
            with self.assertRaisesRegex(IMPORTER.CatalogError, "unreadable or invalid JSON"):
                IMPORTER.load_document(path)
            stderr = io.StringIO()
            with (
                mock.patch("builtins.input") as login_prompt,
                mock.patch.object(IMPORTER.getpass, "getpass") as password_prompt,
                mock.patch.object(IMPORTER, "APIClient") as client,
                redirect_stderr(stderr),
            ):
                code = IMPORTER.run([str(path), "--base-url", "https://example.test"])
            self.assertEqual(code, 2)
            self.assertEqual(stderr.getvalue(), "catalog document is unreadable or invalid JSON\n")
            login_prompt.assert_not_called()
            password_prompt.assert_not_called()
            client.assert_not_called()

    def test_unexpected_operator_failure_is_sanitized_without_traceback(self) -> None:
        stderr = io.StringIO()
        with mock.patch.object(IMPORTER, "load_document", side_effect=TypeError("private request body")), redirect_stderr(stderr):
            code = IMPORTER.run(["catalog.json", "--base-url", "https://example.test"])
        self.assertEqual(code, 2)
        self.assertEqual(stderr.getvalue(), "operator request or report failed\n")

    def test_fingerprint_is_safe_and_stable(self) -> None:
        key = "private-stable-key"
        fingerprint = IMPORTER.key_fingerprint(key)
        self.assertEqual(fingerprint, IMPORTER.key_fingerprint(key))
        self.assertNotIn(key, fingerprint)
        self.assertRegex(fingerprint, r"^[0-9a-f]{12}$")

    def write(self, value: dict) -> Path:
        directory = tempfile.mkdtemp()
        path = Path(directory, "catalog.json")
        path.write_text(json.dumps(value))
        self.addCleanup(lambda: __import__("shutil").rmtree(directory))
        return path


class StubClient:
    def __init__(self, duplicate_name: bool = False, missing_allergen: bool = False):
        category = [{"id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "name": "Protein", "kind": "food_category"}]
        if duplicate_name:
            category.append({"id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "name": " protein ", "kind": "food_category"})
        self.responses = {
            "/api/v1/admin/classifications?kind=food_category": {"classifications": category},
            "/api/v1/admin/classifications?kind=culinary_role": {
                "classifications": [{"id": "cccccccc-cccc-cccc-cccc-cccccccccccc", "name": "Main", "kind": "culinary_role"}]
            },
            "/api/v1/search/filter-options?mode=substitution": {
                "options": [] if missing_allergen else [{"filterId": "peanut", "kind": "allergen"}]
            },
        }

    def request(self, _method: str, path: str) -> object:
        return IMPORTER.APIResponse(200, {"status": "ok", "data": self.responses[path]}, Message())


class PreflightTests(unittest.TestCase):
    def test_names_and_uuid_resolve_to_canonical_request(self) -> None:
        resolved = IMPORTER.preflight(StubClient(), IMPORTER.validate_document({"items": [valid_entry()]}))
        self.assertEqual(resolved[0][1]["foodCategoryIds"], ["aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"])
        self.assertEqual(resolved[0][1]["allergenKeys"], ["peanut"])
        uuid_entry = valid_entry()
        uuid_entry["item"].pop("foodCategoryNames")
        uuid_entry["item"]["foodCategoryIds"] = ["aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"]
        IMPORTER.preflight(StubClient(), IMPORTER.validate_document({"items": [uuid_entry]}))

    def test_uppercase_classification_uuid_resolves_and_is_canonicalized(self) -> None:
        entry = valid_entry()
        entry["item"].pop("foodCategoryNames")
        entry["item"]["foodCategoryIds"] = ["AAAAAAAA-AAAA-AAAA-AAAA-AAAAAAAAAAAA"]
        resolved = IMPORTER.preflight(StubClient(), IMPORTER.validate_document({"items": [entry]}))
        self.assertEqual(resolved[0][1]["foodCategoryIds"], ["aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"])

    def test_unknown_or_ambiguous_remote_references_fail_preflight(self) -> None:
        with self.assertRaises(IMPORTER.CatalogError):
            IMPORTER.preflight(StubClient(duplicate_name=True), IMPORTER.validate_document({"items": [valid_entry()]}))
        with self.assertRaises(IMPORTER.CatalogError):
            IMPORTER.preflight(StubClient(missing_allergen=True), IMPORTER.validate_document({"items": [valid_entry()]}))


class FakeClock:
    def __init__(self) -> None:
        self.value = 0.0
        self.sleeps: list[float] = []

    def now(self) -> float:
        return self.value

    def sleep(self, seconds: float) -> None:
        self.sleeps.append(seconds)
        self.value += seconds


class SequenceClient(IMPORTER.APIClient):
    def __init__(self, responses: list[object], clock: FakeClock):
        super().__init__("https://example.test", sleep=clock.sleep, now=clock.now)
        self.responses = responses
        self.requests: list[tuple[str, bytes]] = []
        self.csrf = "csrf-secret"

    def request(self, method: str, path: str, body=None, headers=None):
        self.requests.append((headers["Idempotency-Key"], json.dumps(body, sort_keys=True).encode()))
        response = self.responses.pop(0)
        if isinstance(response, Exception):
            raise response
        return response


class RetryTests(unittest.TestCase):
    def response(self, status: int, retry_after: str | None = None):
        headers = Message()
        if retry_after is not None:
            headers["Retry-After"] = retry_after
        return IMPORTER.APIResponse(status, {}, headers)

    def test_5xx_and_retry_after_reuse_identical_key_body_with_pacing(self) -> None:
        clock = FakeClock()
        client = SequenceClient([self.response(500), self.response(429, "3"), self.response(201)], clock)
        self.assertEqual(client.create("stable-key", {"name": "x"}), ("succeeded", 201))
        self.assertEqual(len(set(client.requests)), 1)
        self.assertGreaterEqual(sum(clock.sleeps), 5.1)
        self.assertTrue(all(delay <= IMPORTER.MAX_RETRY_AFTER_SECONDS for delay in clock.sleeps))

    def test_5xx_honors_valid_bounded_retry_after_before_identical_retry(self) -> None:
        clock = FakeClock()
        client = SequenceClient([self.response(503, "4"), self.response(201)], clock)
        self.assertEqual(client.create("stable-key", {"name": "x"}), ("succeeded", 201))
        self.assertEqual(len(set(client.requests)), 1)
        self.assertIn(4.0, clock.sleeps)

    def test_permanent_partial_and_auth_failures(self) -> None:
        clock = FakeClock()
        self.assertEqual(SequenceClient([self.response(409)], clock).create("stable-key", {}), ("failed", 409))
        with self.assertRaises(IMPORTER.AuthorizationError):
            SequenceClient([self.response(403)], clock).create("stable-key", {})
        self.assertIsNone(IMPORTER.retry_after_seconds("31"))
        self.assertEqual(IMPORTER.retry_after_seconds("2"), 2)


class OperatorServer(BaseHTTPRequestHandler):
    creates = 0
    seen_cookie = False
    seen_csrf = False

    def log_message(self, _format, *_args):
        pass

    def reply(self, status: int, data: dict, cookie: str | None = None):
        payload = json.dumps({"status": "ok", "requestId": "safe", "data": data}).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        if cookie:
            self.send_header("Set-Cookie", cookie)
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        body = json.loads(self.rfile.read(length))
        if self.path == "/api/v1/auth/login":
            self.server.password = body["password"]
            self.reply(200, {}, "session=in-memory; HttpOnly")
            return
        type(self).creates += 1
        type(self).seen_csrf = self.headers.get("X-CSRF-Token") == "fresh-token"
        self.reply(201, {"id": "dddddddd-dddd-dddd-dddd-dddddddddddd"})

    def do_GET(self):
        type(self).seen_cookie |= "session=in-memory" in self.headers.get("Cookie", "")
        if self.path == "/api/v1/auth/csrf-token":
            self.reply(200, {"csrfToken": "fresh-token"})
        elif self.path.endswith("food_category"):
            self.reply(200, {"classifications": [{"id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "name": "Protein", "kind": "food_category"}]})
        elif self.path.endswith("culinary_role"):
            self.reply(200, {"classifications": [{"id": "cccccccc-cccc-cccc-cccc-cccccccccccc", "name": "Main", "kind": "culinary_role"}]})
        else:
            self.reply(200, {"mode": "substitution", "options": [{"filterId": "peanut", "kind": "allergen"}]})


class EndToEndTests(unittest.TestCase):
    def setUp(self) -> None:
        OperatorServer.creates = 0
        OperatorServer.seen_cookie = False
        OperatorServer.seen_csrf = False
        self.server = ThreadingHTTPServer(("127.0.0.1", 0), OperatorServer)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()

    def tearDown(self) -> None:
        self.server.shutdown()
        self.server.server_close()

    def test_authenticated_dry_run_is_zero_mutation_and_output_is_pii_safe(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            catalog = Path(directory, "catalog.json")
            report = Path(directory, "report.json")
            catalog.write_text(json.dumps({"schema": IMPORTER.SCHEMA, "items": [valid_entry()]}))
            stdout, stderr = io.StringIO(), io.StringIO()
            with mock.patch("builtins.input", return_value="admin@example.test"), mock.patch.object(IMPORTER.getpass, "getpass", return_value="secret-password"), redirect_stdout(stdout), redirect_stderr(stderr):
                code = IMPORTER.run([
                    str(catalog), "--base-url", f"http://127.0.0.1:{self.server.server_port}",
                    "--dry-run", "--report", str(report),
                ])
            combined = stdout.getvalue() + stderr.getvalue() + report.read_text()
            self.assertEqual(code, 0)
            self.assertEqual(OperatorServer.creates, 0)
            self.assertTrue(OperatorServer.seen_cookie)
            for secret in ("secret-password", "admin@example.test", "stable-item-key-0001", "fresh-token", "session=in-memory", "Tofu"):
                self.assertNotIn(secret, combined)

    def test_create_uses_fresh_csrf_and_in_memory_cookie(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            catalog = Path(directory, "catalog.json")
            catalog.write_text(json.dumps({"schema": IMPORTER.SCHEMA, "items": [valid_entry()]}))
            with mock.patch("builtins.input", return_value="admin@example.test"), mock.patch.object(IMPORTER.getpass, "getpass", return_value="secret-password"), redirect_stdout(io.StringIO()):
                code = IMPORTER.run([
                    str(catalog), "--base-url", f"http://127.0.0.1:{self.server.server_port}",
                ])
            self.assertEqual(code, 0)
            self.assertEqual(OperatorServer.creates, 1)
            self.assertTrue(OperatorServer.seen_cookie and OperatorServer.seen_csrf)
            self.assertEqual(self.server.password, "secret-password")

    def test_partial_failure_is_nonzero_and_report_order_is_stable(self) -> None:
        class PartialClient(StubClient):
            outcomes = iter([("failed", 400), ("succeeded", 201)])

            def __init__(self, _base_url: str):
                super().__init__()

            def login(self, _email: str, _password: str) -> None:
                pass

            def create(self, _key: str, _body: dict):
                return next(self.outcomes)

        with tempfile.TemporaryDirectory() as directory:
            catalog = Path(directory, "catalog.json")
            report = Path(directory, "report.json")
            catalog.write_text(json.dumps({
                "schema": IMPORTER.SCHEMA,
                "items": [valid_entry("stable-item-key-0001"), valid_entry("stable-item-key-0002")],
            }))
            with mock.patch.object(IMPORTER, "APIClient", PartialClient), mock.patch("builtins.input", return_value="admin@example.test"), mock.patch.object(
                IMPORTER.getpass, "getpass", return_value="secret-password"
            ), redirect_stdout(io.StringIO()):
                code = IMPORTER.run([
                    str(catalog), "--base-url", "https://example.test",
                    "--report", str(report),
                ])
            results = json.loads(report.read_text())["results"]
            self.assertEqual(code, 1)
            self.assertEqual([result["index"] for result in results], [1, 2])
            self.assertEqual([result["outcome"] for result in results], ["failed", "succeeded"])


if __name__ == "__main__":
    unittest.main()
