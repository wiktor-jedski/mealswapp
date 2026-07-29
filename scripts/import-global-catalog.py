#!/usr/bin/env python3
"""Import a moderate global catalog through the authenticated admin API.

Implements DESIGN-009 ItemCurator JSON global catalog import operator.
"""

from __future__ import annotations

import argparse
import datetime
import email.utils
import hashlib
import json
import math
import re
import ssl
import sys
import time
import urllib.error
import urllib.parse
import uuid
from pathlib import Path
from typing import Any, Callable

from global_catalog_session import (
    APIResponse,
    AuthenticatedSession,
    AuthorizationError,
    SessionError,
    getpass,
    prompt_credentials,
)

SCHEMA = "mealswapp.global-catalog.v1"
MAX_ITEMS = 500
PACE_SECONDS = 2.1
MAX_RETRY_AFTER_SECONDS = 30.0
MAX_ATTEMPTS = 3
MAX_TEXT_LENGTH = 200
MAX_CLASSIFICATION_NAME_LENGTH = 120
MAX_IMAGE_URL_LENGTH = 2048
MICRONUTRIENT_KEYS = frozenset(
    {"Sodium", "Potassium", "Calcium", "Iron", "VitaminC", "VitaminD", "Fiber", "Sugar"}
)
KEY_PATTERN = re.compile(r"^[^\x00-\x1f\x7f]{8,255}$")
ALLERGEN_PATTERN = re.compile(r"^[a-z][a-z0-9_]{0,119}$")
MAX_NUTRITION_VALUE = 99_999_999.9999
INFORMATIONAL_FIELDS = {
    "id", "imageAlt", "sourceProvider", "externalId", "createdAt", "updatedAt",
    "deletedAt", "classifications", "curatedSources",
}


class CatalogError(ValueError):
    """Represent a safe operator-facing validation or protocol failure."""


class DuplicateJSONKey(CatalogError):
    """Represent duplicate object keys rejected before interpretation."""


def reject_duplicate_keys(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    """Build an object while rejecting duplicate JSON keys at every depth."""
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            raise DuplicateJSONKey("catalog contains duplicate JSON keys")
        result[key] = value
    return result


def reject_nonfinite_number(_value: str) -> None:
    """Reject JSON's non-standard NaN and infinity constants."""
    raise CatalogError("catalog contains a non-finite JSON number")


def load_document(path: Path) -> dict[str, Any]:
    """Load one bounded, duplicate-key-safe catalog document."""
    try:
        document = json.loads(
            path.read_text(encoding="utf-8"),
            object_pairs_hook=reject_duplicate_keys,
            parse_constant=reject_nonfinite_number,
        )
    except CatalogError:
        raise
    except (OSError, UnicodeError, ValueError, OverflowError) as error:
        raise CatalogError("catalog document is unreadable or invalid JSON") from error
    if not isinstance(document, dict) or set(document) != {"schema", "items"} or document["schema"] != SCHEMA:
        raise CatalogError(f"catalog must be a closed {SCHEMA} document")
    items = document["items"]
    if not isinstance(items, list) or not items or len(items) > MAX_ITEMS:
        raise CatalogError(f"catalog must contain between 1 and {MAX_ITEMS} items")
    return document


def canonical_name(value: str) -> str:
    """Normalize a portable classification name for unambiguous matching."""
    return " ".join(value.split()).casefold()


def finite_nonnegative(value: Any) -> bool:
    """Return whether a JSON number is finite, bounded, non-boolean, and nonnegative."""
    if not isinstance(value, (int, float)) or isinstance(value, bool):
        return False
    try:
        return math.isfinite(value) and 0 <= value <= MAX_NUTRITION_VALUE
    except OverflowError:
        return False


def bounded_string(value: Any, maximum: int) -> bool:
    """Return whether a JSON value is a bounded PostgreSQL-safe string."""
    return isinstance(value, str) and len(value) <= maximum and "\x00" not in value


def valid_image_url(value: Any) -> bool:
    """Validate the API's bounded HTTP(S) or relative URI-reference contract."""
    if not bounded_string(value, MAX_IMAGE_URL_LENGTH) or any(ord(character) <= 0x20 or ord(character) == 0x7F for character in value):
        return False
    try:
        parsed = urllib.parse.urlsplit(value)
    except ValueError:
        return False
    return parsed.scheme.lower() in {"", "http", "https"} and (not parsed.scheme or bool(parsed.netloc))


def validate_item(index: int, entry: Any) -> dict[str, Any]:
    """Validate one versioned catalog entry without using remote state."""
    if not isinstance(entry, dict) or set(entry) != {"idempotencyKey", "item"}:
        raise CatalogError(f"item {index}: entry fields are invalid")
    key = entry["idempotencyKey"]
    if not isinstance(key, str) or not KEY_PATTERN.fullmatch(key) or key != key.strip():
        raise CatalogError(f"item {index}: idempotency key is invalid")
    item = entry["item"]
    if not isinstance(item, dict):
        raise CatalogError(f"item {index}: item must be an object")
    allowed = {
        "name", "physicalState", "prepTimeMinutes", "averageUnitWeightGrams",
        "averageServingVolumeMilliliters", "densityGramsPerMilliliter", "densitySourceProvider",
        "densitySourceFoodId", "densitySourceKind", "macrosPer100", "micros",
        "foodCategoryNames", "culinaryRoleNames", "foodCategoryIds", "culinaryRoleIds",
        "allergenKeys", "imageUrl", *INFORMATIONAL_FIELDS,
    }
    required = {"name", "physicalState", "macrosPer100", "micros", "allergenKeys"}
    if not required <= set(item) or not set(item) <= allowed or any(item[field] is None for field in required):
        raise CatalogError(f"item {index}: item fields are invalid")
    if not bounded_string(item["name"], MAX_TEXT_LENGTH) or not item["name"].strip():
        raise CatalogError(f"item {index}: name is invalid")
    state = item["physicalState"]
    if not isinstance(state, str) or state not in {"solid", "liquid"}:
        raise CatalogError(f"item {index}: physicalState is invalid")
    prep = item.get("prepTimeMinutes", 0)
    if not isinstance(prep, int) or isinstance(prep, bool) or not 0 <= prep <= MAX_NUTRITION_VALUE:
        raise CatalogError(f"item {index}: prepTimeMinutes is invalid")
    macros = item["macrosPer100"]
    if not isinstance(macros, dict) or set(macros) != {"protein", "carbohydrates", "fat"} or not all(finite_nonnegative(v) for v in macros.values()):
        raise CatalogError(f"item {index}: macrosPer100 is invalid")
    if state == "solid" and sum(macros.values()) > 100:
        raise CatalogError(f"item {index}: solid macros exceed 100 grams")
    for field in ("averageUnitWeightGrams", "averageServingVolumeMilliliters", "densityGramsPerMilliliter"):
        if item.get(field) is not None and (not finite_nonnegative(item[field]) or item[field] <= 0):
            raise CatalogError(f"item {index}: metric measure is invalid")
    for field in ("densitySourceProvider", "densitySourceFoodId"):
        if item.get(field) is not None and not bounded_string(item[field], MAX_TEXT_LENGTH):
            raise CatalogError(f"item {index}: density source is invalid")
    if item.get("densitySourceKind") is not None and (
        not isinstance(item["densitySourceKind"], str)
        or item["densitySourceKind"] not in {"manual", "estimated", "imported"}
    ):
        raise CatalogError(f"item {index}: density source is invalid")
    if item.get("imageUrl") is not None and not valid_image_url(item["imageUrl"]):
        raise CatalogError(f"item {index}: imageUrl is invalid")
    liquid_fields = {"averageServingVolumeMilliliters", "densityGramsPerMilliliter", "densitySourceProvider", "densitySourceFoodId", "densitySourceKind"}
    if state == "solid" and any(item.get(field) is not None for field in liquid_fields):
        raise CatalogError(f"item {index}: solid item contains liquid fields")
    if state == "liquid":
        density = item.get("densityGramsPerMilliliter")
        source_kind = item.get("densitySourceKind")
        if not finite_nonnegative(density) or density <= 0 or not isinstance(source_kind, str) or source_kind not in {"manual", "estimated", "imported"}:
            raise CatalogError(f"item {index}: liquid density is invalid")
        if source_kind == "imported" and (
            not isinstance(item.get("densitySourceProvider"), str)
            or item.get("densitySourceProvider") not in {"usda", "openfoodfacts"}
            or not isinstance(item.get("densitySourceFoodId"), str)
            or not item["densitySourceFoodId"].strip()
        ):
            raise CatalogError(f"item {index}: imported density evidence is invalid")
    micros = item["micros"]
    if not isinstance(micros, dict) or any(key not in MICRONUTRIENT_KEYS or not finite_nonnegative(value) for key, value in micros.items()):
        raise CatalogError(f"item {index}: micronutrients are invalid")
    allergens = item["allergenKeys"]
    if (
        not isinstance(allergens, list)
        or len(allergens) > 100
        or any(not isinstance(key, str) or not ALLERGEN_PATTERN.fullmatch(key) for key in allergens)
        or len(set(allergens)) != len(allergens)
    ):
        raise CatalogError(f"item {index}: allergenKeys are invalid")
    normalized_item = dict(item)
    for names_field, ids_field in (("foodCategoryNames", "foodCategoryIds"), ("culinaryRoleNames", "culinaryRoleIds")):
        if item.get(names_field) is not None and item.get(ids_field) is not None:
            raise CatalogError(f"item {index}: classification names and UUIDs are alternatives")
        values = item.get(names_field)
        if values is None:
            values = item.get(ids_field) or []
        if (
            not isinstance(values, list)
            or len(values) > 100
            or any(
                not bounded_string(value, MAX_CLASSIFICATION_NAME_LENGTH)
                or not value.strip()
                for value in values
            )
        ):
            raise CatalogError(f"item {index}: classifications are invalid")
        if item.get(ids_field) is not None:
            try:
                canonical_ids = [str(uuid.UUID(value)) for value in values]
                if any(uuid.UUID(value).int == 0 for value in canonical_ids):
                    raise ValueError
            except (ValueError, AttributeError) as error:
                raise CatalogError(f"item {index}: classification UUID is invalid") from error
            if len(set(canonical_ids)) != len(canonical_ids):
                raise CatalogError(f"item {index}: classifications are invalid")
            normalized_item[ids_field] = canonical_ids
        elif len(set(values)) != len(values):
            raise CatalogError(f"item {index}: classifications are invalid")
    validate_informational_metadata(index, item)
    return {"idempotencyKey": key, "item": normalized_item}


def validate_informational_metadata(index: int, item: dict[str, Any]) -> None:
    """Validate export-only metadata that manual-item import intentionally ignores."""
    if item.get("id") is not None:
        try:
            if uuid.UUID(item["id"]).int == 0:
                raise ValueError
        except (ValueError, TypeError, AttributeError) as error:
            raise CatalogError(f"item {index}: informational id is invalid") from error
    for field in ("imageAlt", "sourceProvider", "externalId"):
        if item.get(field) is not None and not bounded_string(item[field], MAX_TEXT_LENGTH):
            raise CatalogError(f"item {index}: informational metadata is invalid")
    for field in ("createdAt", "updatedAt"):
        if field in item and (not isinstance(item[field], str) or not item[field]):
            raise CatalogError(f"item {index}: informational timestamp is invalid")
    if item.get("deletedAt") is not None and not isinstance(item["deletedAt"], str):
        raise CatalogError(f"item {index}: informational timestamp is invalid")
    classifications = item.get("classifications")
    if classifications is not None and (
        not isinstance(classifications, dict)
        or set(classifications) != {"foodCategories", "culinaryRoles"}
        or any(not isinstance(classifications[field], list) for field in classifications)
    ):
        raise CatalogError(f"item {index}: classification metadata is invalid")
    curated = item.get("curatedSources")
    if curated is not None and not isinstance(curated, list):
        raise CatalogError(f"item {index}: curated-source metadata is invalid")


def validate_document(document: Any) -> list[dict[str, Any]]:
    """Validate every item and stable key before any authentication or mutation."""
    if not isinstance(document, dict) or not isinstance(document.get("items"), list):
        raise CatalogError("catalog items are invalid")
    entries = [validate_item(index, entry) for index, entry in enumerate(document["items"], 1)]
    keys = [entry["idempotencyKey"] for entry in entries]
    if len(set(keys)) != len(keys):
        raise CatalogError("catalog contains duplicate idempotency keys")
    return entries


def key_fingerprint(key: str) -> str:
    """Return a safe, irreversible report identity for one operator key."""
    return hashlib.sha256(key.encode("utf-8")).hexdigest()[:12]


class APIClient(AuthenticatedSession):
    """Use an in-memory cookie jar for authenticated operator requests."""

    def __init__(self, base_url: str, sleep: Callable[[float], None] = time.sleep, now: Callable[[], float] = time.monotonic):
        try:
            super().__init__(base_url)
        except SessionError as error:
            raise CatalogError(str(error)) from error
        self.sleep = sleep
        self.now = now
        self.last_mutation: float | None = None

    def pace(self) -> None:
        """Keep every create attempt at least 2.1 seconds apart."""
        if self.last_mutation is not None:
            self.sleep(max(0.0, PACE_SECONDS - (self.now() - self.last_mutation)))
        self.last_mutation = self.now()

    def create(self, key: str, body: dict[str, Any]) -> tuple[str, int]:
        """Create or replay one item with bounded same-key ambiguity retries."""
        for attempt in range(MAX_ATTEMPTS):
            self.pace()
            try:
                response = self.request(
                    "POST", "/api/v1/admin/items", body,
                    {"Idempotency-Key": key, "X-CSRF-Token": self.csrf},
                )
            except (urllib.error.URLError, TimeoutError, ssl.SSLError):
                if attempt + 1 == MAX_ATTEMPTS:
                    return "failed", 0
                continue
            if response.status == 201:
                return "succeeded", response.status
            if response.status in {401, 403}:
                raise AuthorizationError("authorization failed during import")
            if response.status in {400, 409}:
                return "failed", response.status
            if response.status == 429:
                delay = retry_after_seconds(response.headers.get("Retry-After"))
                if delay is None or attempt + 1 == MAX_ATTEMPTS:
                    return "failed", response.status
                self.sleep(delay)
                continue
            if 500 <= response.status <= 599 and attempt + 1 < MAX_ATTEMPTS:
                delay = retry_after_seconds(response.headers.get("Retry-After"))
                if delay is not None:
                    self.sleep(delay)
                continue
            return "failed", response.status
        return "failed", 0


def retry_after_seconds(value: str | None, now: datetime.datetime | None = None) -> float | None:
    """Parse a bounded Retry-After delta or HTTP date."""
    if not value:
        return None
    try:
        delay = float(value)
    except ValueError:
        try:
            target = email.utils.parsedate_to_datetime(value)
            current = now or datetime.datetime.now(datetime.timezone.utc)
            delay = (target - current).total_seconds()
        except (TypeError, ValueError, OverflowError):
            return None
    if not 0 <= delay <= MAX_RETRY_AFTER_SECONDS:
        return None
    return delay


def response_data(response: APIResponse, key: str) -> Any:
    """Read a success envelope or raise a safe protocol error."""
    try:
        if response.status != 200 or response.body["status"] != "ok":
            raise KeyError
        return response.body["data"][key]
    except (TypeError, KeyError) as error:
        raise CatalogError("preflight response is invalid") from error


def preflight(client: APIClient, entries: list[dict[str, Any]]) -> list[tuple[str, dict[str, Any]]]:
    """Resolve every active classification and allergen before the first mutation."""
    catalogs: dict[str, list[dict[str, Any]]] = {}
    for kind in ("food_category", "culinary_role"):
        response = client.request("GET", f"/api/v1/admin/classifications?kind={kind}")
        classifications = response_data(response, "classifications")
        if not isinstance(classifications, list) or any(
            not isinstance(item, dict) or item.get("kind") != kind or not isinstance(item.get("name"), str)
            for item in classifications
        ):
            raise CatalogError("classification preflight response is invalid")
        catalogs[kind] = classifications
    options = response_data(client.request("GET", "/api/v1/search/filter-options?mode=substitution"), "options")
    if not isinstance(options, list) or any(not isinstance(option, dict) for option in options):
        raise CatalogError("allergen preflight response is invalid")
    allergens = {option.get("filterId") for option in options if option.get("kind") == "allergen"}
    resolved: list[tuple[str, dict[str, Any]]] = []
    for index, entry in enumerate(entries, 1):
        source = entry["item"]
        body = {key: value for key, value in source.items() if key not in {
            "foodCategoryNames", "culinaryRoleNames", "foodCategoryIds", "culinaryRoleIds",
            *INFORMATIONAL_FIELDS,
        } and value is not None}
        for kind, names_field, ids_field, output_field in (
            ("food_category", "foodCategoryNames", "foodCategoryIds", "foodCategoryIds"),
            ("culinary_role", "culinaryRoleNames", "culinaryRoleIds", "culinaryRoleIds"),
        ):
            try:
                by_id = {str(uuid.UUID(str(item["id"]))): item for item in catalogs[kind]}
            except (KeyError, TypeError, ValueError) as error:
                raise CatalogError("classification preflight response contains an invalid UUID") from error
            by_name: dict[str, list[str]] = {}
            for item in catalogs[kind]:
                by_name.setdefault(canonical_name(item["name"]), []).append(str(uuid.UUID(str(item["id"]))))
            if ids_field in source:
                values = source[ids_field]
                if any(value not in by_id for value in values):
                    raise CatalogError(f"item {index}: classification UUID is unknown")
                body[output_field] = sorted(values)
            else:
                result = []
                for name in source.get(names_field, []):
                    matches = by_name.get(canonical_name(name), [])
                    if len(matches) != 1:
                        raise CatalogError(f"item {index}: classification name is unknown or ambiguous")
                    result.append(matches[0])
                body[output_field] = sorted(result)
        if any(key not in allergens for key in source["allergenKeys"]):
            raise CatalogError(f"item {index}: allergen key is unknown or inactive")
        body["allergenKeys"] = sorted(source["allergenKeys"])
        resolved.append((entry["idempotencyKey"], body))
    return resolved


def write_report(path: Path | None, records: list[dict[str, Any]]) -> None:
    """Write stable, credential- and PII-free per-item results."""
    if path is not None:
        path.write_text(json.dumps({"schema": SCHEMA, "results": records}, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def run(argv: list[str] | None = None) -> int:
    """Run the authenticated import and return a process exit status."""
    parser = argparse.ArgumentParser(description="Import at most 500 global catalog items")
    parser.add_argument("catalog", type=Path)
    parser.add_argument("--base-url", required=True)
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--report", type=Path)
    args = parser.parse_args(argv)
    try:
        entries = validate_document(load_document(args.catalog))
        email_address, password = prompt_credentials()
        client = APIClient(args.base_url)
        client.login(email_address, password)
        resolved = preflight(client, entries)
        if args.dry_run:
            records = [{"index": index, "keyFingerprint": key_fingerprint(key), "outcome": "validated"} for index, (key, _) in enumerate(resolved, 1)]
            write_report(args.report, records)
            print(f"validated={len(records)} failed=0 dry_run=true")
            return 0
        records = []
        for index, (key, body) in enumerate(resolved, 1):
            outcome, status = client.create(key, body)
            records.append({"index": index, "keyFingerprint": key_fingerprint(key), "outcome": outcome, "httpStatus": status})
        write_report(args.report, records)
        failed = sum(record["outcome"] != "succeeded" for record in records)
        print(f"succeeded={len(records) - failed} failed={failed}")
        return 1 if failed else 0
    except Exception as error:
        if not isinstance(error, (CatalogError, SessionError)):
            error = CatalogError("operator request or report failed")
        print(str(error), file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(run())
