"""Task 278 operator tests for safe deterministic global catalog export."""

from __future__ import annotations

import contextlib
import csv
import importlib.util
import io
import json
import sys
import tempfile
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from unittest import mock

SCRIPTS = Path(__file__).parent
sys.path.insert(0, str(SCRIPTS))
SPEC = importlib.util.spec_from_file_location("export_global_catalog", SCRIPTS / "export-global-catalog.py")
EXPORT = importlib.util.module_from_spec(SPEC)
assert SPEC.loader
SPEC.loader.exec_module(EXPORT)
IMPORT_SPEC = importlib.util.spec_from_file_location("roundtrip_import_global_catalog", SCRIPTS / "import-global-catalog.py")
IMPORTER = importlib.util.module_from_spec(IMPORT_SPEC)
assert IMPORT_SPEC.loader
IMPORT_SPEC.loader.exec_module(IMPORTER)

ITEM_ID = "00000000-0000-4000-8000-000000000001"
DOCUMENT = {
    "schema": "mealswapp.global-catalog.v1",
    "items": [{
        "idempotencyKey": f"global-catalog:{ITEM_ID}",
        "item": {
            "id": ITEM_ID, "name": "Żurek secret-name", "physicalState": "liquid",
            "prepTimeMinutes": 5, "averageUnitWeightGrams": None,
            "averageServingVolumeMilliliters": 250, "densityGramsPerMilliliter": 1.03,
            "densitySourceProvider": "usda", "densitySourceFoodId": "density-42", "densitySourceKind": "imported",
            "macrosPer100": {"protein": 1, "carbohydrates": 2.5, "fat": 3},
            "micros": {"Iron": 1.25}, "foodCategoryNames": ["Soup"],
            "culinaryRoleNames": ["Main"], "allergenKeys": ["gluten"],
            "imageUrl": None, "imageAlt": None, "sourceProvider": "usda", "externalId": "42",
            "createdAt": "2026-07-01T00:00:00Z", "updatedAt": "2026-07-01T00:00:00Z",
            "deletedAt": None,
            "classifications": {
                "foodCategories": [{"id": "00000000-0000-4000-8000-000000000071", "name": "Soup"}],
                "culinaryRoles": [{"id": "00000000-0000-4000-8000-000000000072", "name": "Main"}],
            },
            "curatedSources": [{"id": ITEM_ID, "provider": "usda", "externalId": "42", "status": "imported"}],
        },
    }],
}


class OperatorHandler(BaseHTTPRequestHandler):
    export_status = 200
    export_body = json.dumps(DOCUMENT, ensure_ascii=False, separators=(",", ":")).encode() + b"\n"
    partial = False
    paths: list[str] = []

    def do_POST(self):
        type(self).paths.append(self.path)
        body = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        if b"admin@example.test" not in body or b"password-secret" not in body:
            self.send_response(401)
            self.end_headers()
            return
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Set-Cookie", "session=memory-only; HttpOnly")
        self.end_headers()
        self.wfile.write(b'{"status":"ok"}')

    def do_GET(self):
        type(self).paths.append(self.path)
        if self.path == "/api/v1/auth/csrf-token":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(b'{"data":{"csrfToken":"csrf-secret"}}')
            return
        if self.path.startswith("/api/v1/admin/catalog-export"):
            self.send_response(type(self).export_status)
            self.send_header("Content-Type", "application/json")
            self.send_header("X-Request-ID", "00000000-0000-4000-8000-000000000099")
            if type(self).partial:
                self.send_header("Content-Length", str(len(type(self).export_body) + 100))
            self.end_headers()
            body = type(self).export_body
            self.wfile.write(body[: len(body) // 2] if type(self).partial else body)
            return
        self.send_response(404)
        self.end_headers()

    def log_message(self, *_args):
        return


class ExportOperatorTests(unittest.TestCase):
    def setUp(self):
        OperatorHandler.export_status = 200
        OperatorHandler.export_body = json.dumps(DOCUMENT, ensure_ascii=False, separators=(",", ":")).encode() + b"\n"
        OperatorHandler.partial = False
        OperatorHandler.paths = []
        self.server = ThreadingHTTPServer(("127.0.0.1", 0), OperatorHandler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.base_url = f"http://127.0.0.1:{self.server.server_port}"

    def tearDown(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join()

    def run_operator(self, output: Path, *arguments: str):
        stdout, stderr = io.StringIO(), io.StringIO()
        with (
            mock.patch("builtins.input", return_value="admin@example.test"),
            mock.patch("global_catalog_session.getpass.getpass", return_value="password-secret"),
            contextlib.redirect_stdout(stdout),
            contextlib.redirect_stderr(stderr),
        ):
            status = EXPORT.run(["--base-url", self.base_url, "--output", str(output), *arguments])
        return status, stdout.getvalue(), stderr.getvalue()

    def test_json_output_include_deleted_summary_and_safe_streams(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "catalog.json"
            output.write_text("prior destination", encoding="utf-8")
            status, stdout, stderr = self.run_operator(output, "--include-deleted")
            self.assertEqual(status, 0)
            self.assertEqual(json.loads(output.read_text(encoding="utf-8")), DOCUMENT)
            self.assertIn("/api/v1/admin/catalog-export?includeDeleted=true", OperatorHandler.paths)
            self.assertIn("items=1", stdout)
            self.assertEqual(stderr, "")
            for secret in ("admin@example.test", "password-secret", "csrf-secret", "session=memory-only", "Żurek secret-name"):
                self.assertNotIn(secret, stdout + stderr)
            self.assertEqual(list(Path(directory).glob(".*.tmp")), [])

    def test_pretty_json_and_csv_are_parseable_and_deterministic(self):
        with tempfile.TemporaryDirectory() as directory:
            pretty = Path(directory) / "catalog.json"
            csv_path = Path(directory) / "catalog.csv"
            self.assertEqual(self.run_operator(pretty, "--pretty")[0], 0)
            first = pretty.read_bytes()
            self.assertIn(b"\n  \"items\":", first)
            self.assertEqual(self.run_operator(pretty, "--pretty")[0], 0)
            self.assertEqual(pretty.read_bytes(), first)
            self.assertEqual(self.run_operator(csv_path, "--format", "csv")[0], 0)
            with csv_path.open(encoding="utf-8", newline="") as source:
                rows = list(csv.DictReader(source))
            self.assertEqual(rows[0]["id"], ITEM_ID)
            self.assertEqual(tuple(rows[0]), EXPORT.CSV_COLUMNS)
            self.assertEqual(rows[0]["densitySourceKind"], "imported")
            self.assertEqual(json.loads(rows[0]["allergenKeys"]), ["gluten"])
            self.assertEqual(json.loads(rows[0]["micros"]), {"Iron": 1.25})

    def test_partial_and_http_failures_preserve_existing_destination(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "catalog.json"
            output.write_text("preserve-me", encoding="utf-8")
            OperatorHandler.partial = True
            status, stdout, stderr = self.run_operator(output)
            self.assertNotEqual(status, 0)
            self.assertEqual(output.read_text(encoding="utf-8"), "preserve-me")
            self.assertEqual(stdout, "")
            self.assertNotIn("Żurek secret-name", stderr)
            OperatorHandler.partial = False
            OperatorHandler.export_status = 403
            self.assertNotEqual(self.run_operator(output)[0], 0)
            self.assertEqual(output.read_text(encoding="utf-8"), "preserve-me")

    def test_invalid_response_and_output_options_fail_nonzero(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "catalog.json"
            output.write_text("old", encoding="utf-8")
            OperatorHandler.export_body = b'{"schema":"wrong","items":[]}'
            self.assertNotEqual(self.run_operator(output)[0], 0)
            self.assertEqual(output.read_text(encoding="utf-8"), "old")
            with self.assertRaises(SystemExit):
                EXPORT.run(["--base-url", self.base_url, "--output", str(output), "--format", "csv", "--pretty"])
            with self.assertRaises(SystemExit):
                EXPORT.run(["--base-url", self.base_url, "--output", str(output), "--format", "xml"])

    def test_temporary_files_are_same_directory_and_publication_fsyncs(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "catalog.json"
            observed: list[Path] = []
            original = EXPORT.tempfile.NamedTemporaryFile

            def recording_temp(*args, **kwargs):
                observed.append(Path(kwargs["dir"]))
                return original(*args, **kwargs)

            with (
                mock.patch.object(EXPORT.tempfile, "NamedTemporaryFile", side_effect=recording_temp),
                mock.patch.object(EXPORT.os, "fsync", wraps=EXPORT.os.fsync) as fsync,
                mock.patch.object(EXPORT.os, "replace", wraps=EXPORT.os.replace) as replace,
            ):
                self.assertEqual(self.run_operator(output, "--pretty")[0], 0)
            self.assertTrue(observed)
            self.assertEqual(set(observed), {output.parent})
            self.assertGreaterEqual(fsync.call_count, 3)
            replace.assert_called_once()

    def test_directory_fsync_failure_restores_prior_destination_or_removes_new_file(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "catalog.json"
            output.write_bytes(b"prior-destination-bytes")
            with mock.patch.object(EXPORT, "sync_directory", side_effect=OSError("injected directory fsync failure")):
                status, stdout, stderr = self.run_operator(output, "--pretty")
            self.assertNotEqual(status, 0)
            self.assertEqual(output.read_bytes(), b"prior-destination-bytes")
            self.assertEqual(stdout, "")
            self.assertNotIn("injected", stderr)
            self.assertEqual(list(Path(directory).glob(".*.tmp")), [])

            output.unlink()
            with mock.patch.object(EXPORT, "sync_directory", side_effect=OSError("injected directory fsync failure")):
                self.assertNotEqual(self.run_operator(output)[0], 0)
            self.assertFalse(output.exists())
            self.assertEqual(list(Path(directory).glob(".*.tmp")), [])

    def test_exact_numeric_precision_and_adversarial_json_validation(self):
        with tempfile.TemporaryDirectory() as directory:
            raw = json.dumps(DOCUMENT, ensure_ascii=False, separators=(",", ":"))
            exact = "0.123456789012345678901234567890123456789"
            raw = raw.replace('"protein":1,', f'"protein":{exact},')
            source = Path(directory) / "source.json"
            source.write_text(raw, encoding="utf-8")
            document = EXPORT.load_export(source)
            compact = Path(directory) / "compact.json"
            OperatorHandler.export_body = raw.encode() + b"\n"
            self.assertEqual(self.run_operator(compact)[0], 0)
            self.assertIn(exact, compact.read_text(encoding="utf-8"))
            pretty = Path(directory) / "pretty.json"
            csv_path = Path(directory) / "catalog.csv"
            EXPORT.write_json(document, pretty, True)
            EXPORT.write_csv(document, csv_path)
            self.assertIn(exact, pretty.read_text(encoding="utf-8"))
            with csv_path.open(encoding="utf-8", newline="") as csv_source:
                self.assertEqual(next(csv.DictReader(csv_source))["proteinPer100"], exact)

            invalid_documents = [
                raw.replace(exact, "NaN"),
                raw.replace(exact, "Infinity"),
                raw.replace(exact, "-Infinity"),
                raw.replace(f'"global-catalog:{ITEM_ID}"', '"global-catalog:wrong"', 1),
                raw.replace('"prepTimeMinutes":5', '"prepTimeMinutes":true'),
                raw.replace('"name":"Żurek secret-name"', '"name":"Żurek secret-name","unexpected":"field"'),
                raw.replace('"protein":' + exact, '"protein":-1'),
                raw.replace('"protein":' + exact, '"protein":1,"protein":2'),
            ]
            for index, invalid in enumerate(invalid_documents):
                candidate = Path(directory) / f"invalid-{index}.json"
                candidate.write_text(invalid, encoding="utf-8")
                with self.assertRaises(EXPORT.ExportError, msg=f"invalid case {index}"):
                    EXPORT.load_export(candidate)

    def test_temp_creation_and_body_read_failures_close_response_and_cleanup(self):
        class Response:
            status = 200
            headers = {"X-Request-ID": "safe-request"}

            def __init__(self, read_error: Exception | None = None):
                self.closed = False
                self.read_error = read_error

            def read(self, _size):
                if self.read_error is not None:
                    raise self.read_error
                return b""

            def close(self):
                self.closed = True

        class Session:
            def __init__(self, response):
                self.response = response

            def open(self, *_args):
                return self.response

        with tempfile.TemporaryDirectory() as directory:
            destination = Path(directory) / "catalog.json"
            response = Response()
            with (
                mock.patch.object(EXPORT, "temporary_path", side_effect=OSError("temp failed")),
                self.assertRaises(OSError),
            ):
                EXPORT.stream_response(Session(response), False, destination)
            self.assertTrue(response.closed)

            response = Response(OSError("read failed"))
            with self.assertRaises(OSError):
                EXPORT.stream_response(Session(response), False, destination)
            self.assertTrue(response.closed)
            self.assertEqual(list(Path(directory).glob(".*.tmp")), [])

    def test_export_round_trip_preserves_supported_fields_and_strips_curated_metadata(self):
        class PreflightClient:
            def request(self, _method, path):
                if "food_category" in path:
                    body = {"status": "ok", "data": {"classifications": [{
                        "id": "00000000-0000-4000-8000-000000000071",
                        "name": "Soup", "kind": "food_category",
                    }]}}
                elif "culinary_role" in path:
                    body = {"status": "ok", "data": {"classifications": [{
                        "id": "00000000-0000-4000-8000-000000000072",
                        "name": "Main", "kind": "culinary_role",
                    }]}}
                else:
                    body = {"status": "ok", "data": {"options": [{"filterId": "gluten", "kind": "allergen"}]}}
                return IMPORTER.APIResponse(200, body, {})

        entries = IMPORTER.validate_document(DOCUMENT)
        resolved = IMPORTER.preflight(PreflightClient(), entries)
        self.assertEqual(len(resolved), 1)
        key, body = resolved[0]
        self.assertEqual(key, f"global-catalog:{ITEM_ID}")
        self.assertEqual(body["name"], DOCUMENT["items"][0]["item"]["name"])
        self.assertEqual(body["micros"], {"Iron": 1.25})
        self.assertEqual(body["densitySourceKind"], "imported")
        self.assertEqual(body["allergenKeys"], ["gluten"])
        self.assertEqual(body["foodCategoryIds"], ["00000000-0000-4000-8000-000000000071"])
        self.assertEqual(body["culinaryRoleIds"], ["00000000-0000-4000-8000-000000000072"])
        for informational in IMPORTER.INFORMATIONAL_FIELDS:
            self.assertNotIn(informational, body)


if __name__ == "__main__":
    unittest.main()
