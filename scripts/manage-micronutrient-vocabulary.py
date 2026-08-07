#!/usr/bin/env python3
"""Manage the canonical micronutrient vocabulary through the administration API.

Implements DESIGN-009 AdminController micronutrient vocabulary operator.
"""

from __future__ import annotations

import argparse
import datetime
import email.utils
import json
import re
import ssl
import sys
import time
import unicodedata
import urllib.error
import urllib.parse
from typing import Any, Callable

from global_catalog_session import (
    APIResponse,
    AuthenticatedSession,
    AuthorizationError,
    SessionError,
    prompt_credentials,
)

KEY_PATTERN = re.compile(r"^[A-Z][A-Za-z0-9]{2,119}$")
REQUEST_ID_PATTERN = re.compile(r"^[A-Za-z0-9-]{1,120}$")
UNITS = ("g", "mg", "mcg")
MAX_ATTEMPTS = 3
MAX_RETRY_AFTER_SECONDS = 5.0
MAX_RESPONSE_BYTES = 256 * 1024
ENTRY_FIELDS = {"key", "displayName", "unit", "active"}
TASK289_ENTRY_FIELDS = {"Key", "DisplayName", "Unit", "Active"}


class OperatorError(ValueError):
    """Represent one safe operator-facing validation or protocol failure."""


class ConflictError(OperatorError):
    """Represent an in-use or concurrent vocabulary conflict."""


def validate_key(value: str) -> str:
    """Validate an immutable canonical micronutrient key."""
    if not KEY_PATTERN.fullmatch(value):
        raise argparse.ArgumentTypeError("key must be a canonical micronutrient key")
    return value


def validate_display_name(value: str) -> str:
    """Validate a normalized bounded display name without terminal controls."""
    if (
        not isinstance(value, str)
        or not value
        or value != value.strip()
        or len(value) > 120
        or any(unicodedata.category(character).startswith("C") for character in value)
    ):
        raise argparse.ArgumentTypeError("display name is invalid")
    return value


def validate_target(environment: str, base_url: str, confirm_production: bool) -> str:
    """Require one explicit origin whose transport is appropriate for the environment."""
    if base_url != base_url.strip() or any(ord(character) < 32 for character in base_url):
        raise OperatorError("API target is invalid")
    parsed = urllib.parse.urlsplit(base_url)
    if (
        parsed.scheme not in {"http", "https"}
        or not parsed.netloc
        or parsed.username
        or parsed.password
        or parsed.path not in {"", "/"}
        or parsed.query
        or parsed.fragment
    ):
        raise OperatorError("API target is invalid")
    try:
        host = parsed.hostname
        parsed.port
    except ValueError as error:
        raise OperatorError("API target is invalid") from error
    loopback = host in {"localhost", "127.0.0.1", "::1"}
    if parsed.scheme == "http" and (environment != "development" or not loopback):
        raise OperatorError("API target requires HTTPS")
    if environment == "production" and not confirm_production:
        raise OperatorError("production confirmation is required")
    if environment != "production" and confirm_production:
        raise OperatorError("production confirmation does not match environment")
    return f"{parsed.scheme}://{parsed.netloc}"


def retry_after_seconds(value: str | None, now: datetime.datetime | None = None) -> float | None:
    """Parse a delta or HTTP-date Retry-After within the operator's fixed bound."""
    if not value:
        return None
    try:
        delay = float(value)
    except ValueError:
        try:
            target = email.utils.parsedate_to_datetime(value)
            current = now or datetime.datetime.now(datetime.timezone.utc)
            if target.tzinfo is None:
                target = target.replace(tzinfo=datetime.timezone.utc)
            delay = (target - current).total_seconds()
        except (TypeError, ValueError, OverflowError):
            return None
    return delay if 0 <= delay <= MAX_RETRY_AFTER_SECONDS else None


def validate_entry(value: Any) -> dict[str, Any]:
    """Validate one exact administration vocabulary projection."""
    if not isinstance(value, dict) or (set(value) != ENTRY_FIELDS and set(value) != TASK289_ENTRY_FIELDS):
        raise OperatorError("API response is invalid")
    if set(value) == TASK289_ENTRY_FIELDS:
        value = {
            "key": value["Key"],
            "displayName": value["DisplayName"],
            "unit": value["Unit"],
            "active": value["Active"],
        }
    key = value["key"]
    display_name = value["displayName"]
    try:
        validate_key(key)
        validate_display_name(display_name)
    except (argparse.ArgumentTypeError, TypeError, AttributeError) as error:
        raise OperatorError("API response is invalid") from error
    if value["unit"] not in UNITS or not isinstance(value["active"], bool):
        raise OperatorError("API response is invalid")
    return {"key": key, "displayName": display_name, "unit": value["unit"], "active": value["active"]}


def response_data(response: APIResponse, field: str, expected_status: int) -> Any:
    """Extract one field from an exact success envelope."""
    body = response.body
    if (
        response.status != expected_status
        or not isinstance(body, dict)
        or set(body) != {"status", "requestId", "data"}
        or body["status"] != "ok"
        or not isinstance(body["requestId"], str)
        or not REQUEST_ID_PATTERN.fullmatch(body["requestId"])
        or not isinstance(body["data"], dict)
        or set(body["data"]) != {field}
    ):
        raise OperatorError("API response is invalid")
    return body["data"][field]


def parse_list_response(response: APIResponse) -> list[dict[str, Any]]:
    """Validate deterministic, duplicate-free vocabulary listing."""
    values = response_data(response, "micronutrients", 200)
    if not isinstance(values, list) or len(values) > 1000:
        raise OperatorError("API response is invalid")
    entries = [validate_entry(value) for value in values]
    keys = [entry["key"] for entry in entries]
    if keys != sorted(keys) or len(keys) != len(set(keys)):
        raise OperatorError("API response is invalid")
    return entries


class VocabularyClient(AuthenticatedSession):
    """Call the vocabulary API with process-memory credentials and bounded retries."""

    def __init__(self, base_url: str, sleep: Callable[[float], None] | None = None):
        super().__init__(base_url)
        self.sleep = sleep or time.sleep

    def request(
        self,
        method: str,
        path: str,
        body: Any = None,
        headers: dict[str, str] | None = None,
    ) -> APIResponse:
        """Decode one bounded JSON response while rejecting duplicate object keys."""
        response = self.open(method, path, body, headers)
        try:
            raw = response.read(MAX_RESPONSE_BYTES + 1)
            status = response.status
            response_headers = response.headers
        finally:
            response.close()
        if len(raw) > MAX_RESPONSE_BYTES:
            raise OperatorError("API response is invalid")

        def reject_duplicates(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
            decoded: dict[str, Any] = {}
            for key, value in pairs:
                if key in decoded:
                    raise OperatorError("API response is invalid")
                decoded[key] = value
            return decoded

        try:
            decoded = json.loads(raw, object_pairs_hook=reject_duplicates) if raw else None
        except (json.JSONDecodeError, UnicodeDecodeError) as error:
            raise OperatorError("API response is invalid") from error
        return APIResponse(status, decoded, response_headers)

    def list_entries(self) -> list[dict[str, Any]]:
        """Return the validated authoritative vocabulary."""
        try:
            response = self.request("GET", "/api/v1/admin/micronutrients")
        except (urllib.error.URLError, TimeoutError, ssl.SSLError) as error:
            raise OperatorError("operator request failed") from error
        if response.status in {401, 403}:
            raise AuthorizationError("authorization failed")
        return parse_list_response(response)

    def reconcile(self, desired: dict[str, Any]) -> dict[str, Any] | None:
        """Return authoritative desired state after an ambiguous mutation outcome."""
        try:
            entries = self.list_entries()
        except AuthorizationError:
            raise
        except OperatorError:
            return None
        for entry in entries:
            if all(entry.get(field) == value for field, value in desired.items()):
                return entry
        return None

    def mutate(
        self,
        method: str,
        path: str,
        body: dict[str, str] | None,
        expected_status: int,
        desired: dict[str, Any],
    ) -> dict[str, Any]:
        """Send one identical CSRF-protected mutation across bounded ambiguous retries."""
        headers = {"X-CSRF-Token": self.csrf}
        for attempt in range(MAX_ATTEMPTS):
            try:
                response = self.request(method, path, body, headers)
            except (urllib.error.URLError, TimeoutError, ssl.SSLError):
                reconciled = self.reconcile(desired)
                if reconciled is not None:
                    return reconciled
                if attempt + 1 < MAX_ATTEMPTS:
                    continue
                raise OperatorError("operator request failed")
            if response.status in {401, 403}:
                raise AuthorizationError("authorization failed")
            if response.status == 409:
                raise ConflictError("mutation blocked by an in-use or concurrent conflict")
            if response.status == 429:
                delay = retry_after_seconds(response.headers.get("Retry-After"))
                if delay is not None and attempt + 1 < MAX_ATTEMPTS:
                    self.sleep(delay)
                    continue
                raise OperatorError("mutation failed permanently")
            if 500 <= response.status <= 599:
                reconciled = self.reconcile(desired)
                if reconciled is not None:
                    return reconciled
                if attempt + 1 == MAX_ATTEMPTS:
                    raise OperatorError("mutation failed permanently")
                delay = retry_after_seconds(response.headers.get("Retry-After"))
                if delay is not None:
                    self.sleep(delay)
                continue
            if response.status != expected_status:
                raise OperatorError("mutation failed permanently")
            try:
                return validate_entry(response_data(response, "micronutrient", expected_status))
            except OperatorError:
                reconciled = self.reconcile(desired)
                if reconciled is not None:
                    return reconciled
                raise
        raise OperatorError("operator request failed")


def build_parser() -> argparse.ArgumentParser:
    """Build the closed command-line contract."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--environment", required=True, choices=("development", "staging", "production"))
    parser.add_argument("--base-url", required=True)
    parser.add_argument("--confirm-production", action="store_true")
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("list", help="list active and inactive entries")

    add = commands.add_parser("add", help="add one canonical entry")
    add.add_argument("--key", required=True, type=validate_key)
    add.add_argument("--display-name", required=True, type=validate_display_name)
    add.add_argument("--unit", required=True, choices=UNITS)

    display = commands.add_parser("update-display-name", help="update one display name")
    display.add_argument("--key", required=True, type=validate_key)
    display.add_argument("--display-name", required=True, type=validate_display_name)

    unit = commands.add_parser("update-unit", help="safely update one unused entry's unit")
    unit.add_argument("--key", required=True, type=validate_key)
    unit.add_argument("--unit", required=True, choices=UNITS)

    for name in ("deactivate", "reactivate"):
        lifecycle = commands.add_parser(name, help=f"{name} one canonical entry")
        lifecycle.add_argument("--key", required=True, type=validate_key)
    for command in (add, display, unit, *[commands.choices[name] for name in ("deactivate", "reactivate")]):
        command.add_argument("--dry-run", action="store_true")
    return parser


def mutation_request(args: argparse.Namespace) -> tuple[str, str, dict[str, str] | None, int]:
    """Map one validated mutation command to the administration route."""
    escaped_key = urllib.parse.quote(args.key, safe="")
    if args.command == "add":
        return "POST", "/api/v1/admin/micronutrients", {
            "key": args.key, "displayName": args.display_name, "unit": args.unit,
        }, 201
    if args.command == "update-display-name":
        return "PUT", f"/api/v1/admin/micronutrients/{escaped_key}/display-name", {
            "displayName": args.display_name,
        }, 200
    if args.command == "update-unit":
        return "PUT", f"/api/v1/admin/micronutrients/{escaped_key}/unit", {"unit": args.unit}, 200
    return "POST", f"/api/v1/admin/micronutrients/{escaped_key}/{args.command}", None, 200


def desired_state(args: argparse.Namespace) -> dict[str, Any]:
    """Describe the observable authoritative state required after one command."""
    desired: dict[str, Any] = {"key": args.key}
    if args.command == "add":
        desired.update({"displayName": args.display_name, "unit": args.unit, "active": True})
    elif args.command == "update-display-name":
        desired["displayName"] = args.display_name
    elif args.command == "update-unit":
        desired["unit"] = args.unit
    elif args.command == "deactivate":
        desired["active"] = False
    else:
        desired["active"] = True
    return desired


def print_entries(entries: list[dict[str, Any]]) -> None:
    """Print stable, terminal-safe tab-separated vocabulary state."""
    print("key\tdisplay_name\tunit\tactive")
    for entry in entries:
        print(f"{entry['key']}\t{entry['displayName']}\t{entry['unit']}\t{str(entry['active']).lower()}")


def run(argv: list[str] | None = None) -> int:
    """Run one operator command and return its process exit status."""
    args = build_parser().parse_args(argv)
    try:
        base_url = validate_target(args.environment, args.base_url, args.confirm_production)
        email_address, password = prompt_credentials()
        client = VocabularyClient(base_url)
        client.login(email_address, password)
        if args.command == "list":
            print_entries(client.list_entries())
            return 0
        if args.dry_run:
            entries = client.list_entries()
            existing_keys = {entry["key"] for entry in entries}
            if args.command == "add" and args.key in existing_keys:
                raise ConflictError("micronutrient key already exists")
            if args.command != "add" and args.key not in existing_keys:
                raise OperatorError("micronutrient key was not found")
            print(f"validated command={args.command} key={args.key} dry_run=true")
            return 0
        method, path, body, expected_status = mutation_request(args)
        desired = desired_state(args)
        entry = client.mutate(method, path, body, expected_status, desired)
        if any(entry.get(field) != value for field, value in desired.items()):
            raise OperatorError("API response is invalid")
        print(
            f"succeeded command={args.command} key={entry['key']} "
            f"unit={entry['unit']} active={str(entry['active']).lower()}"
        )
        return 0
    except (OperatorError, SessionError) as error:
        print(str(error), file=sys.stderr)
        return 2
    except Exception:
        print("operator request failed", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(run())
