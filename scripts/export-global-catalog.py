#!/usr/bin/env python3
"""Export the complete ownerless global catalog through the authenticated admin API.

Implements DESIGN-009 AdminController global catalog export operator.
"""

from __future__ import annotations

import argparse
import csv
import datetime
import io
import json
import os
import re
import shutil
import sys
import tempfile
import uuid
from decimal import Decimal
from pathlib import Path
from typing import Any

from global_catalog_session import AuthenticatedSession, SessionError, prompt_credentials

SCHEMA = "mealswapp.global-catalog.v1"
CHUNK_SIZE = 64 * 1024
REQUEST_ID_PATTERN = re.compile(r"^[A-Za-z0-9-]{1,120}$")
ALLERGEN_PATTERN = re.compile(r"^[a-z][a-z0-9_]{0,119}$")
MAX_NUMBER = Decimal("99999999.999999")
MICRONUTRIENT_KEYS = frozenset(
    {"Sodium", "Potassium", "Calcium", "Iron", "VitaminC", "VitaminD", "Fiber", "Sugar"}
)
CSV_COLUMNS = (
    "id", "idempotencyKey", "name", "physicalState", "prepTimeMinutes",
    "averageUnitWeightGrams", "averageServingVolumeMilliliters", "densityGramsPerMilliliter",
    "densitySourceProvider", "densitySourceFoodId", "densitySourceKind", "proteinPer100",
    "carbohydratesPer100", "fatPer100", "micros", "foodCategoryNames", "culinaryRoleNames",
    "allergenKeys", "imageUrl", "imageAlt", "sourceProvider", "externalId", "createdAt",
    "updatedAt", "deletedAt", "classifications", "curatedSources",
)
ITEM_FIELDS = {
    "id", "name", "physicalState", "prepTimeMinutes", "averageUnitWeightGrams",
    "averageServingVolumeMilliliters", "densityGramsPerMilliliter", "densitySourceProvider",
    "densitySourceFoodId", "densitySourceKind", "macrosPer100", "micros",
    "foodCategoryNames", "culinaryRoleNames", "allergenKeys", "imageUrl", "imageAlt",
    "sourceProvider", "externalId", "createdAt", "updatedAt", "deletedAt",
    "classifications", "curatedSources",
}


class ExportError(ValueError):
    """Represent a safe operator-facing export failure."""


def reject_duplicate_keys(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    """Reject ambiguous duplicate JSON object keys."""
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            raise ExportError("export response is invalid")
        result[key] = value
    return result


def reject_nonfinite_number(_value: str) -> None:
    """Reject JSON's non-standard NaN and infinity constants."""
    raise ExportError("export response is invalid")


def load_export(path: Path) -> dict[str, Any]:
    """Parse exact JSON numbers and verify the complete export representation."""
    try:
        with path.open(encoding="utf-8") as source:
            document = json.load(
                source,
                object_pairs_hook=reject_duplicate_keys,
                parse_float=Decimal,
                parse_constant=reject_nonfinite_number,
            )
    except ExportError:
        raise
    except (OSError, UnicodeError, ValueError, OverflowError) as error:
        raise ExportError("export response is invalid") from error
    if (
        not isinstance(document, dict)
        or set(document) != {"schema", "items"}
        or document["schema"] != SCHEMA
        or not isinstance(document["items"], list)
        or "generatedAt" in document
    ):
        raise ExportError("export response is invalid")
    previous_id = ""
    for entry in document["items"]:
        item_id = validate_entry(entry)
        if item_id <= previous_id:
            raise ExportError("export response is invalid")
        previous_id = item_id
    return document


def validate_entry(entry: Any) -> str:
    """Validate one closed catalog entry and return its canonical UUID."""
    if not isinstance(entry, dict) or set(entry) != {"idempotencyKey", "item"}:
        raise ExportError("export response is invalid")
    item = entry["item"]
    if not isinstance(item, dict) or set(item) != ITEM_FIELDS:
        raise ExportError("export response is invalid")
    try:
        parsed_item_id = uuid.UUID(item["id"])
        item_id = str(parsed_item_id)
    except (ValueError, TypeError, AttributeError) as error:
        raise ExportError("export response is invalid") from error
    if parsed_item_id.int == 0 or item_id != item["id"] or entry["idempotencyKey"] != "global-catalog:" + item_id:
        raise ExportError("export response is invalid")
    if not valid_text(item["name"], maximum=200) or item["physicalState"] not in {"solid", "liquid"}:
        raise ExportError("export response is invalid")
    if not isinstance(item["prepTimeMinutes"], int) or isinstance(item["prepTimeMinutes"], bool) or item["prepTimeMinutes"] < 0:
        raise ExportError("export response is invalid")
    for field in ("averageUnitWeightGrams", "averageServingVolumeMilliliters", "densityGramsPerMilliliter"):
        if item[field] is not None and (not finite_number(item[field]) or item[field] <= 0):
            raise ExportError("export response is invalid")
    for field in ("densitySourceProvider", "densitySourceFoodId", "imageAlt", "sourceProvider", "externalId"):
        if item[field] is not None and not valid_text(item[field], maximum=200):
            raise ExportError("export response is invalid")
    if item["imageUrl"] is not None and not valid_text(item["imageUrl"], maximum=2048):
        raise ExportError("export response is invalid")
    if item["densitySourceKind"] not in {None, "manual", "estimated", "imported"}:
        raise ExportError("export response is invalid")
    if item["physicalState"] == "solid" and any(item[field] is not None for field in (
        "averageServingVolumeMilliliters", "densityGramsPerMilliliter", "densitySourceProvider",
        "densitySourceFoodId", "densitySourceKind",
    )):
        raise ExportError("export response is invalid")
    if item["physicalState"] == "liquid" and (
        item["densityGramsPerMilliliter"] is None or item["densitySourceKind"] is None
    ):
        raise ExportError("export response is invalid")
    validate_number_map(item["macrosPer100"], {"protein", "carbohydrates", "fat"})
    validate_number_map(item["micros"])
    if any(key not in MICRONUTRIENT_KEYS for key in item["micros"]):
        raise ExportError("export response is invalid")
    validate_sorted_strings(item["foodCategoryNames"], 120)
    validate_sorted_strings(item["culinaryRoleNames"], 120)
    validate_sorted_strings(item["allergenKeys"], 120, ALLERGEN_PATTERN)
    classifications = item["classifications"]
    if not isinstance(classifications, dict) or set(classifications) != {"foodCategories", "culinaryRoles"}:
        raise ExportError("export response is invalid")
    validate_classifications(classifications["foodCategories"], item["foodCategoryNames"])
    validate_classifications(classifications["culinaryRoles"], item["culinaryRoleNames"])
    validate_curated_sources(item["curatedSources"])
    validate_timestamp(item["createdAt"])
    validate_timestamp(item["updatedAt"])
    if item["deletedAt"] is not None:
        validate_timestamp(item["deletedAt"])
    return item_id


def valid_text(value: Any, maximum: int = 2048) -> bool:
    """Return whether a value is bounded safe text."""
    return isinstance(value, str) and 0 < len(value) <= maximum and "\x00" not in value


def finite_number(value: Any) -> bool:
    """Return whether a value is a finite exact JSON number."""
    if isinstance(value, bool) or not isinstance(value, (int, Decimal)):
        return False
    return (not isinstance(value, Decimal) or value.is_finite()) and abs(value) <= MAX_NUMBER


def validate_number_map(value: Any, exact_keys: set[str] | None = None) -> None:
    """Validate a JSON number map without converting exact decimals to floats."""
    if not isinstance(value, dict) or (exact_keys is not None and set(value) != exact_keys):
        raise ExportError("export response is invalid")
    if any(not valid_text(key) or not finite_number(number) or number < 0 for key, number in value.items()):
        raise ExportError("export response is invalid")


def validate_sorted_strings(value: Any, maximum: int, pattern: re.Pattern[str] | None = None) -> None:
    """Validate a duplicate-free deterministically sorted string array."""
    if (
        not isinstance(value, list)
        or any(not valid_text(item, maximum) or pattern is not None and not pattern.fullmatch(item) for item in value)
        or value != sorted(set(value))
    ):
        raise ExportError("export response is invalid")


def validate_classifications(value: Any, expected_names: list[str]) -> None:
    """Validate sorted UUID/name relationship metadata."""
    if not isinstance(value, list):
        raise ExportError("export response is invalid")
    identities: list[tuple[str, str]] = []
    for classification in value:
        if not isinstance(classification, dict) or set(classification) != {"id", "name"} or not valid_text(classification["name"], 120):
            raise ExportError("export response is invalid")
        try:
            parsed_identifier = uuid.UUID(classification["id"])
            identifier = str(parsed_identifier)
        except (ValueError, TypeError, AttributeError) as error:
            raise ExportError("export response is invalid") from error
        if parsed_identifier.int == 0 or identifier != classification["id"]:
            raise ExportError("export response is invalid")
        identities.append((classification["name"], identifier))
    if identities != sorted(set(identities)) or [name for name, _ in identities] != expected_names:
        raise ExportError("export response is invalid")


def validate_curated_sources(value: Any) -> None:
    """Validate sorted informational curated-source identities."""
    if not isinstance(value, list):
        raise ExportError("export response is invalid")
    identities: list[tuple[str, str, str]] = []
    for source in value:
        if (
            not isinstance(source, dict)
            or set(source) != {"id", "provider", "externalId", "status"}
            or not valid_text(source["provider"], 200)
            or not valid_text(source["externalId"], 200)
            or source["status"] not in {"draft", "imported", "conflict", "rejected"}
        ):
            raise ExportError("export response is invalid")
        try:
            parsed_identifier = uuid.UUID(source["id"])
            identifier = str(parsed_identifier)
        except (ValueError, TypeError, AttributeError) as error:
            raise ExportError("export response is invalid") from error
        if parsed_identifier.int == 0 or identifier != source["id"]:
            raise ExportError("export response is invalid")
        identities.append((source["provider"], source["externalId"], identifier))
    if identities != sorted(set(identities)):
        raise ExportError("export response is invalid")


def validate_timestamp(value: Any) -> None:
    """Validate one timezone-aware RFC3339 timestamp."""
    if not isinstance(value, str) or len(value) > 64 or not value.endswith("Z"):
        raise ExportError("export response is invalid")
    try:
        parsed = datetime.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError as error:
        raise ExportError("export response is invalid") from error
    if parsed.tzinfo is None or parsed.utcoffset() != datetime.timedelta(0):
        raise ExportError("export response is invalid")


def temporary_path(destination: Path) -> Path:
    """Create one private same-directory temporary path."""
    handle = tempfile.NamedTemporaryFile(
        mode="xb", prefix=f".{destination.name}.", suffix=".tmp",
        dir=destination.parent, delete=False,
    )
    path = Path(handle.name)
    handle.close()
    return path


def flush_and_sync(handle) -> None:
    """Flush Python and kernel file buffers before publication."""
    handle.flush()
    os.fsync(handle.fileno())


def stream_response(session: AuthenticatedSession, include_deleted: bool, destination: Path) -> tuple[Path, str]:
    """Download the API response to a same-directory durable temporary file."""
    query = "?includeDeleted=true" if include_deleted else ""
    response = session.open("GET", "/api/v1/admin/catalog-export" + query)
    path: Path | None = None
    try:
        request_id = response.headers.get("X-Request-ID", "")
        path = temporary_path(destination)
        if response.status in {401, 403}:
            raise ExportError("catalog export authorization failed")
        if response.status != 200:
            raise ExportError("catalog export request failed")
        with path.open("wb") as target:
            while chunk := response.read(CHUNK_SIZE):
                target.write(chunk)
            flush_and_sync(target)
        return path, request_id if REQUEST_ID_PATTERN.fullmatch(request_id) else "unavailable"
    except Exception:
        if path is not None:
            path.unlink(missing_ok=True)
        raise
    finally:
        response.close()


def write_json(document: dict[str, Any], path: Path, pretty: bool) -> None:
    """Write deterministic UTF-8 JSON without losing Decimal precision."""
    with path.open("w", encoding="utf-8", newline="\n") as target:
        write_json_value(target, document, pretty)
        target.write("\n")
        flush_and_sync(target)


def write_json_value(target, value: Any, pretty: bool, depth: int = 0) -> None:
    """Write one validated JSON value while preserving exact numeric tokens."""
    if value is None:
        target.write("null")
    elif value is True:
        target.write("true")
    elif value is False:
        target.write("false")
    elif isinstance(value, (int, Decimal)):
        target.write(str(value))
    elif isinstance(value, str):
        target.write(json.dumps(value, ensure_ascii=False))
    elif isinstance(value, list):
        write_json_collection(target, value, pretty, depth, "[", "]")
    elif isinstance(value, dict):
        items = sorted(value.items()) if pretty else value.items()
        write_json_collection(target, list(items), pretty, depth, "{", "}", object_items=True)
    else:
        raise ExportError("export response is invalid")


def write_json_collection(target, values: list[Any], pretty: bool, depth: int, opening: str, closing: str, object_items: bool = False) -> None:
    """Write one JSON array or object with deterministic separators."""
    target.write(opening)
    for index, value in enumerate(values):
        if index:
            target.write(",")
        if pretty:
            target.write("\n" + "  " * (depth + 1))
        if object_items:
            key, nested = value
            target.write(json.dumps(key, ensure_ascii=False) + (": " if pretty else ":"))
            write_json_value(target, nested, pretty, depth + 1)
        else:
            write_json_value(target, value, pretty, depth + 1)
    if pretty and values:
        target.write("\n" + "  " * depth)
    target.write(closing)


def json_cell(value: Any) -> str:
    """Encode nested CSV values for deterministic human inspection."""
    output = io.StringIO()
    write_json_value(output, value, False)
    return output.getvalue()


def write_csv(document: dict[str, Any], path: Path) -> None:
    """Write a stable inspection CSV without changing the import representation."""
    with path.open("w", encoding="utf-8", newline="") as target:
        writer = csv.DictWriter(target, fieldnames=CSV_COLUMNS, lineterminator="\n")
        writer.writeheader()
        for entry in document["items"]:
            item = entry["item"]
            macros = item.get("macrosPer100") or {}
            writer.writerow({
                "id": item.get("id"), "idempotencyKey": entry["idempotencyKey"], "name": item.get("name"),
                "physicalState": item.get("physicalState"), "prepTimeMinutes": item.get("prepTimeMinutes"),
                "averageUnitWeightGrams": item.get("averageUnitWeightGrams"),
                "averageServingVolumeMilliliters": item.get("averageServingVolumeMilliliters"),
                "densityGramsPerMilliliter": item.get("densityGramsPerMilliliter"),
                "densitySourceProvider": item.get("densitySourceProvider"), "densitySourceFoodId": item.get("densitySourceFoodId"),
                "densitySourceKind": item.get("densitySourceKind"),
                "proteinPer100": macros.get("protein"), "carbohydratesPer100": macros.get("carbohydrates"),
                "fatPer100": macros.get("fat"), "micros": json_cell(item.get("micros") or {}),
                "foodCategoryNames": json_cell(item.get("foodCategoryNames") or []),
                "culinaryRoleNames": json_cell(item.get("culinaryRoleNames") or []),
                "allergenKeys": json_cell(item.get("allergenKeys") or []),
                "imageUrl": item.get("imageUrl"), "imageAlt": item.get("imageAlt"),
                "sourceProvider": item.get("sourceProvider"), "externalId": item.get("externalId"),
                "createdAt": item.get("createdAt"), "updatedAt": item.get("updatedAt"),
                "deletedAt": item.get("deletedAt"), "classifications": json_cell(item.get("classifications") or {}),
                "curatedSources": json_cell(item.get("curatedSources") or []),
            })
        flush_and_sync(target)


def publish(source: Path, destination: Path) -> None:
    """Publish durably, restoring prior destination bytes on every failure."""
    backup: Path | None = None
    try:
        if os.path.lexists(destination):
            backup = temporary_path(destination)
            with destination.open("rb") as previous, backup.open("wb") as saved:
                shutil.copyfileobj(previous, saved, CHUNK_SIZE)
                flush_and_sync(saved)
        os.replace(source, destination)
        try:
            sync_directory(destination.parent)
            if backup is not None:
                backup.unlink()
        except Exception:
            restore_destination(backup, destination)
            raise
    except Exception:
        if backup is not None:
            backup.unlink(missing_ok=True)
        raise


def restore_destination(backup: Path | None, destination: Path) -> None:
    """Restore a prior destination, or remove a newly created one, after failure."""
    if backup is None:
        destination.unlink(missing_ok=True)
    else:
        os.replace(backup, destination)
    try:
        sync_directory(destination.parent)
    except OSError:
        pass


def sync_directory(path: Path) -> None:
    """Fsync one directory entry and always close its descriptor."""
    directory = os.open(path, os.O_RDONLY)
    try:
        os.fsync(directory)
    finally:
        os.close(directory)


def run(argv: list[str] | None = None) -> int:
    """Run one authenticated export and return a process exit status."""
    parser = argparse.ArgumentParser(description="Export the ownerless global catalog")
    parser.add_argument("--base-url", required=True)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--format", choices=("json", "csv"), default="json")
    parser.add_argument("--pretty", action="store_true")
    parser.add_argument("--include-deleted", action="store_true")
    args = parser.parse_args(argv)
    if args.pretty and args.format != "json":
        parser.error("--pretty is supported only with --format json")
    downloaded: Path | None = None
    converted: Path | None = None
    try:
        if not args.output.name or not args.output.parent.is_dir():
            raise ExportError("output directory is unavailable")
        email_address, password = prompt_credentials()
        session = AuthenticatedSession(args.base_url)
        session.login(email_address, password)
        downloaded, request_id = stream_response(session, args.include_deleted, args.output)
        document = load_export(downloaded)
        if args.format == "json" and not args.pretty:
            converted = downloaded
            downloaded = None
        else:
            converted = temporary_path(args.output)
            if args.format == "json":
                write_json(document, converted, True)
            else:
                write_csv(document, converted)
        publish(converted, args.output)
        converted = None
        print(f"output={args.output} items={len(document['items'])} request_id={request_id}")
        return 0
    except Exception as error:
        if not isinstance(error, (ExportError, SessionError)):
            error = ExportError("catalog export failed")
        print(str(error), file=sys.stderr)
        return 1
    finally:
        if downloaded is not None:
            downloaded.unlink(missing_ok=True)
        if converted is not None:
            converted.unlink(missing_ok=True)


if __name__ == "__main__":
    raise SystemExit(run())
