#!/usr/bin/env python3
"""Query and sanitize deployed GCP Cloud Logging evidence for Task 285."""

# Implements DESIGN-014 LogAggregator deployed acceptance boundary.
from __future__ import annotations

import dataclasses
import datetime as dt
import hmac
import json
import math
import re
import subprocess
import time
import urllib.error
import urllib.request
from collections.abc import Callable, Mapping, Sequence
from typing import Any

UTC = dt.timezone.utc
REQUEST_ID_RE = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$",
    re.I,
)
PROJECT_RE = re.compile(r"^[a-z][a-z0-9-]{4,28}[a-z0-9]$")
FORBIDDEN_KEY_PARTS = (
    "email",
    "password",
    "credential",
    "authorization",
    "authentication",
    "authheader",
    "cookie",
    "csrf",
    "token",
    "secret",
    "apikey",
    "idempotency",
    "userid",
    "itemid",
    "externalid",
    "query",
    "searchtext",
    "providerpayload",
    "url",
    "uri",
    "diagnostic",
    "stack",
    "traceback",
    "textpayload",
    "httprequest",
    "useragent",
)
FORBIDDEN_KEY_NAMES = frozenset(
    {
        "name",
        "displayname",
        "firstname",
        "lastname",
        "fullname",
        "username",
        "message",
        "logmessage",
        "consolemessage",
        "auth",
        "authentication",
        "bearer",
        "privatekey",
        "publickey",
        "encryptionkey",
        "signingkey",
        "clientkey",
        "accesskey",
        "refreshkey",
    }
)
EMAIL_RE = re.compile(r"(?<![A-Za-z0-9._%+-])[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}")
URL_RE = re.compile(r"\b(?:https?|postgres(?:ql)?|redis)://", re.I)
JWT_RE = re.compile(r"\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b")
UUID_RE = re.compile(r"\b[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\b", re.I)
ALLOWED_EVENT_KEYS = frozenset({"requestId", "action", "resource", "outcome"})
CANONICAL_EVENT_CONTRACTS = {
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
MAX_WINDOW_SECONDS = 900
MAX_PAGES = 20
MAX_ENTRIES = 5000


class SinkError(RuntimeError):
    """Base class for safe centralized-sink failures."""


class SinkAuthorizationBlocked(SinkError):
    """The configured identity cannot query the centralized sink."""


class SinkUnavailableBlocked(SinkError):
    """The centralized sink cannot be queried completely in time."""


class SinkMalformed(SinkError):
    """The centralized sink returned an unsupported or unsafe shape."""


class RedactionViolation(SinkError):
    """A centralized event contains a forbidden field or value."""


@dataclasses.dataclass(frozen=True)
class SinkConfig:
    """Bounded GCP Cloud Logging query configuration."""

    project_id: str
    resource_names: tuple[str, ...]
    poll_seconds: float = 2.0
    timeout_seconds: float = 60.0
    page_size: int = 500
    access_token_command: tuple[str, ...] = (
        "gcloud",
        "auth",
        "application-default",
        "print-access-token",
    )

    @classmethod
    def from_environment(cls, environment: Mapping[str, str]) -> "SinkConfig":
        """Load strict non-secret sink coordinates from the deployed environment."""
        project = environment.get("MEALSWAPP_TASK285_GCP_PROJECT", "")
        raw_resources = environment.get("MEALSWAPP_TASK285_LOG_RESOURCE_NAMES", "")
        resources = tuple(item.strip() for item in raw_resources.split(",") if item.strip())
        if not PROJECT_RE.fullmatch(project):
            raise SinkUnavailableBlocked("centralized log project is not configured")
        if not resources or any(item != f"projects/{project}" for item in resources):
            raise SinkUnavailableBlocked("centralized log resources are not least-scope project resources")
        try:
            poll_seconds = float(environment.get("MEALSWAPP_TASK285_LOG_POLL_SECONDS", "2"))
            timeout_seconds = float(environment.get("MEALSWAPP_TASK285_LOG_TIMEOUT_SECONDS", "60"))
            page_size = int(environment.get("MEALSWAPP_TASK285_LOG_PAGE_SIZE", "500"))
        except ValueError as error:
            raise SinkUnavailableBlocked("centralized log query bounds are invalid") from error
        if (
            not math.isfinite(poll_seconds)
            or not math.isfinite(timeout_seconds)
            or not 0 < poll_seconds <= 10
            or not 0 < timeout_seconds <= 300
            or not 1 <= page_size <= 1000
        ):
            raise SinkUnavailableBlocked("centralized log query bounds are unsafe")
        return cls(project, resources, poll_seconds, timeout_seconds, page_size)


@dataclasses.dataclass(frozen=True)
class QueryWindow:
    """One closed UTC logging query interval."""

    start: dt.datetime
    end: dt.datetime

    def __post_init__(self) -> None:
        if self.start.tzinfo is None or self.end.tzinfo is None:
            raise ValueError("log query timestamps must be timezone-aware")
        duration = (self.end - self.start).total_seconds()
        if duration <= 0 or duration > MAX_WINDOW_SECONDS:
            raise ValueError("log query window must be positive and at most 15 minutes")


@dataclasses.dataclass(frozen=True)
class ExpectedEvent:
    """Safe action-to-centralized-event correlation contract."""

    category: str
    request_id: str
    action: str
    resource: str
    outcome: str


@dataclasses.dataclass(frozen=True)
class SafeEvent:
    """Sanitized centralized event retained as acceptance evidence."""

    request_id: str
    timestamp: str
    ingested_at: str
    action: str
    resource: str
    outcome: str


def format_timestamp(value: dt.datetime) -> str:
    """Format one UTC timestamp for the Cloud Logging filter."""
    return value.astimezone(UTC).isoformat(timespec="microseconds").replace("+00:00", "Z")


def parse_timestamp(value: object, field: str) -> dt.datetime:
    """Parse one RFC 3339 UTC timestamp without accepting naive values."""
    if not isinstance(value, str) or len(value) > 64:
        raise SinkMalformed(f"{field} is missing or malformed")
    try:
        parsed = dt.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError as error:
        raise SinkMalformed(f"{field} is missing or malformed") from error
    if parsed.tzinfo is None:
        raise SinkMalformed(f"{field} is missing or malformed")
    return parsed.astimezone(UTC)


def validate_expected_events(events: Sequence[ExpectedEvent]) -> None:
    """Require the exact deployed action categories and fixed event tuples."""
    actual = {
        item.category: (item.action, item.resource, item.outcome)
        for item in events
    }
    if len(actual) != len(events) or actual != CANONICAL_EVENT_CONTRACTS:
        raise SinkMalformed("action event contract is not canonical")


def load_expected_events(value: object, provenance: str | None = None) -> list[ExpectedEvent]:
    """Validate the Playwright action receipt without retaining fixture data."""
    if not isinstance(value, dict) or set(value) != {
        "schema",
        "provenance",
        "startedAt",
        "finishedAt",
        "events",
    }:
        raise SinkMalformed("action receipt shape is invalid")
    if (
        value["schema"] != "mealswapp.task285-actions.v1"
        or not isinstance(value["provenance"], str)
        or not re.fullmatch(r"[0-9a-f]{48}", value["provenance"])
        or provenance is not None
        and not secrets_compare(value["provenance"], provenance)
        or not isinstance(value["events"], list)
    ):
        raise SinkMalformed("action receipt contract is invalid")
    started = parse_timestamp(value["startedAt"], "action start")
    finished = parse_timestamp(value["finishedAt"], "action finish")
    if finished < started:
        raise SinkMalformed("action receipt timestamps are reversed")
    events: list[ExpectedEvent] = []
    categories: set[str] = set()
    request_ids: set[str] = set()
    for raw in value["events"]:
        if not isinstance(raw, dict) or set(raw) != {"category", "requestId", "action", "resource", "outcome"}:
            raise SinkMalformed("action event shape is invalid")
        if (
            not all(isinstance(raw[key], str) and re.fullmatch(r"[a-z][a-z0-9_]{0,47}", raw[key]) for key in ("category", "action", "resource", "outcome"))
            or not REQUEST_ID_RE.fullmatch(raw["requestId"])
            or raw["category"] in categories
            or raw["requestId"].lower() in request_ids
        ):
            raise SinkMalformed("action event identity is invalid")
        categories.add(raw["category"])
        request_ids.add(raw["requestId"].lower())
        events.append(ExpectedEvent(raw["category"], raw["requestId"], raw["action"], raw["resource"], raw["outcome"]))
    events.sort(key=lambda item: item.category)
    validate_expected_events(events)
    return events


def secrets_compare(left: str, right: str) -> bool:
    """Compare run provenance without timing-dependent early exit."""
    return hmac.compare_digest(left, right)


class CloudLoggingClient:
    """Minimal paginated Cloud Logging entries:list adapter."""

    endpoint = "https://logging.googleapis.com/v2/entries:list"

    def __init__(
        self,
        config: SinkConfig,
        *,
        opener: Callable[[urllib.request.Request, float], bytes] | None = None,
        token_loader: Callable[[float], str] | None = None,
    ) -> None:
        self.config = config
        self._opener = opener or self._open
        self._token_loader = token_loader or self._load_token

    def _load_token(self, timeout: float) -> str:
        """Acquire an application-default access token without exposing it."""
        try:
            completed = subprocess.run(
                self.config.access_token_command,
                check=True,
                capture_output=True,
                text=True,
                timeout=min(timeout, 15),
            )
        except (OSError, subprocess.SubprocessError) as error:
            raise SinkAuthorizationBlocked("centralized log query authority is unavailable") from error
        token = completed.stdout.strip()
        if not token or any(char.isspace() for char in token):
            raise SinkAuthorizationBlocked("centralized log query authority is unavailable")
        return token

    @staticmethod
    def _open(request: urllib.request.Request, timeout: float) -> bytes:
        """Execute one Cloud Logging request and classify only safe status metadata."""
        try:
            with urllib.request.urlopen(request, timeout=timeout) as response:
                return response.read(4 * 1024 * 1024 + 1)
        except urllib.error.HTTPError as error:
            error.close()
            if error.code in {401, 403}:
                raise SinkAuthorizationBlocked("centralized log query was not authorized") from error
            raise SinkUnavailableBlocked(f"centralized log query failed with status {error.code}") from error
        except (urllib.error.URLError, TimeoutError) as error:
            raise SinkUnavailableBlocked("centralized log query was unavailable") from error

    def query(
        self,
        request_ids: Sequence[str],
        window: QueryWindow,
        *,
        deadline: float | None = None,
        monotonic: Callable[[], float] = time.monotonic,
    ) -> tuple[list[dict[str, Any]], int]:
        """Read every bounded result page for the requested correlations."""
        deadline = deadline if deadline is not None else monotonic() + self.config.timeout_seconds
        unique_ids = sorted(set(request_ids))
        if not unique_ids or len(unique_ids) > 64 or any(not REQUEST_ID_RE.fullmatch(item) for item in unique_ids):
            raise ValueError("one to 64 safe request IDs are required")
        request_terms = " OR ".join(
            f'jsonPayload.requestId="{item}"' for item in unique_ids
        )
        query_filter = (
            f'timestamp>="{format_timestamp(window.start)}" '
            f'AND timestamp<="{format_timestamp(window.end)}" AND ({request_terms})'
        )
        remaining = deadline - monotonic()
        if remaining <= 0:
            raise SinkUnavailableBlocked("centralized log query exceeded its deadline")
        token = self._token_loader(remaining)
        entries: list[dict[str, Any]] = []
        next_page = ""
        pages = 0
        while True:
            body: dict[str, Any] = {
                "resourceNames": list(self.config.resource_names),
                "filter": query_filter,
                "orderBy": "timestamp asc",
                "pageSize": self.config.page_size,
            }
            if next_page:
                body["pageToken"] = next_page
            request = urllib.request.Request(
                self.endpoint,
                data=json.dumps(body, separators=(",", ":")).encode(),
                headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
                method="POST",
            )
            try:
                remaining = deadline - monotonic()
                if remaining <= 0:
                    raise SinkUnavailableBlocked("centralized log query exceeded its deadline")
                raw = self._opener(request, min(remaining, 30))
            except urllib.error.HTTPError as error:
                error.close()
                if error.code in {401, 403}:
                    raise SinkAuthorizationBlocked("centralized log query was not authorized") from error
                raise SinkUnavailableBlocked(f"centralized log query failed with status {error.code}") from error
            except (urllib.error.URLError, TimeoutError) as error:
                raise SinkUnavailableBlocked("centralized log query was unavailable") from error
            if len(raw) > 4 * 1024 * 1024:
                raise SinkMalformed("centralized log page exceeds the evidence bound")
            try:
                payload = json.loads(raw)
            except (UnicodeError, json.JSONDecodeError) as error:
                raise SinkMalformed("centralized log response is malformed") from error
            if not isinstance(payload, dict) or set(payload) - {"entries", "nextPageToken"}:
                raise SinkMalformed("centralized log response shape is malformed")
            page_entries = payload.get("entries", [])
            if not isinstance(page_entries, list) or any(not isinstance(item, dict) for item in page_entries):
                raise SinkMalformed("centralized log entries are malformed")
            entries.extend(page_entries)
            pages += 1
            if len(entries) > MAX_ENTRIES or pages > MAX_PAGES:
                raise SinkMalformed("centralized log pagination exceeds the evidence bound")
            next_page = payload.get("nextPageToken", "")
            if not isinstance(next_page, str) or len(next_page) > 2048:
                raise SinkMalformed("centralized log pagination token is malformed")
            if not next_page:
                return entries, pages


def extract_payload(entry: Mapping[str, Any]) -> dict[str, Any]:
    """Extract one structured event without accepting console text as evidence."""
    payload = entry.get("jsonPayload")
    if not isinstance(payload, dict):
        raise SinkMalformed("centralized entry is not a structured JSON payload")
    return payload


def scan_forbidden(
    value: Any,
    request_ids: set[str],
    probes: Sequence[str],
    *,
    key: str = "",
) -> None:
    """Reject forbidden keys, recognizable secrets, and run-owned fixture values."""
    normalized_key = re.sub(r"[^a-z0-9]", "", key.casefold())
    if (
        normalized_key
        and normalized_key != "requestid"
        and (
            normalized_key in FORBIDDEN_KEY_NAMES
            or any(part in normalized_key for part in FORBIDDEN_KEY_PARTS)
        )
    ):
        raise RedactionViolation("centralized event contains a forbidden field")
    if isinstance(value, dict):
        for nested_key, nested_value in value.items():
            if not isinstance(nested_key, str):
                raise RedactionViolation("centralized event contains a non-string field")
            scan_forbidden(nested_value, request_ids, probes, key=nested_key)
    elif isinstance(value, list):
        for nested in value:
            scan_forbidden(nested, request_ids, probes)
    elif isinstance(value, str):
        lowered = value.casefold()
        if EMAIL_RE.search(value) or URL_RE.search(value) or JWT_RE.search(value):
            raise RedactionViolation("centralized event contains a forbidden value")
        if any(probe and probe.casefold() in lowered for probe in probes):
            raise RedactionViolation("centralized event contains a run-owned private value")
        for identifier in UUID_RE.findall(value):
            if identifier.lower() not in request_ids:
                raise RedactionViolation("centralized event contains a non-correlation identifier")


def sanitize_entries(
    entries: Sequence[Mapping[str, Any]],
    expected: Sequence[ExpectedEvent],
    window: QueryWindow,
    probes: Sequence[str] = (),
) -> list[SafeEvent]:
    """Scan raw sink results and retain only the fixed safe event projection."""
    request_ids = {item.request_id.lower() for item in expected}
    safe: list[SafeEvent] = []
    for entry in entries:
        scan_forbidden(entry, request_ids, probes)
        payload = extract_payload(entry)
        request_id = payload.get("requestId")
        if not isinstance(request_id, str) or request_id.lower() not in request_ids:
            continue
        if set(payload) != ALLOWED_EVENT_KEYS:
            raise RedactionViolation("centralized event contains fields outside the fixed acceptance vocabulary")
        if not all(
            isinstance(payload.get(field), str)
            and re.fullmatch(r"[a-z][a-z0-9_]{0,47}", payload[field])
            for field in ("action", "resource", "outcome")
        ):
            raise SinkMalformed("centralized event fields are malformed")
        created = parse_timestamp(entry.get("timestamp"), "event timestamp")
        ingested = parse_timestamp(entry.get("receiveTimestamp"), "ingestion timestamp")
        if created < window.start.astimezone(UTC) or created > window.end.astimezone(UTC) + dt.timedelta(minutes=2):
            raise SinkMalformed("centralized event timestamp is outside the action window")
        if ingested < created or ingested > dt.datetime.now(UTC) + dt.timedelta(minutes=2):
            raise SinkMalformed("centralized ingestion timestamp is inconsistent")
        safe.append(
            SafeEvent(
                request_id.lower(),
                format_timestamp(created),
                format_timestamp(ingested),
                payload["action"],
                payload["resource"],
                payload["outcome"],
            )
        )
    return sorted(safe, key=lambda item: (item.request_id, item.timestamp, item.action))


def poll_for_events(
    client: CloudLoggingClient,
    expected: Sequence[ExpectedEvent],
    window: QueryWindow,
    probes: Sequence[str] = (),
    *,
    monotonic: Callable[[], float] = time.monotonic,
    sleeper: Callable[[float], None] = time.sleep,
) -> tuple[list[SafeEvent], int, int]:
    """Poll delayed ingestion until every request correlation is observed."""
    deadline = monotonic() + client.config.timeout_seconds
    polls = 0
    total_pages = 0
    while True:
        if monotonic() >= deadline:
            raise SinkUnavailableBlocked("centralized log ingestion remained incomplete")
        entries, pages = client.query(
            [item.request_id for item in expected],
            window,
            deadline=deadline,
            monotonic=monotonic,
        )
        polls += 1
        total_pages += pages
        safe = sanitize_entries(entries, expected, window, probes)
        observed = {item.request_id for item in safe}
        if all(item.request_id.lower() in observed for item in expected):
            return safe, polls, total_pages
        if monotonic() >= deadline:
            raise SinkUnavailableBlocked("centralized log ingestion remained incomplete")
        remaining = deadline - monotonic()
        if remaining <= 0:
            raise SinkUnavailableBlocked("centralized log ingestion remained incomplete")
        sleeper(min(client.config.poll_seconds, remaining))


def verify_event_contract(
    expected: Sequence[ExpectedEvent],
    observed: Sequence[SafeEvent],
) -> list[str]:
    """Return fixed finding codes for missing, duplicate, or inconsistent outcomes."""
    findings: list[str] = []
    by_request: dict[str, list[SafeEvent]] = {}
    for event in observed:
        by_request.setdefault(event.request_id, []).append(event)
    for item in expected:
        matches = [
            event
            for event in by_request.get(item.request_id.lower(), [])
            if (event.action, event.resource, event.outcome)
            == (item.action, item.resource, item.outcome)
        ]
        if len(matches) != 1:
            findings.append(f"EVENT-{item.category.upper()}-COUNT")
    admin_categories = {"manual_success", "manual_failure", "audit_failure"}
    if any(item.category in admin_categories for item in expected):
        admin_outcomes = {
            item.outcome
            for item in expected
            if item.category in admin_categories
        }
        if not {"succeeded", "validation_failed", "audit_failed"} <= admin_outcomes:
            findings.append("ADMIN-OUTCOMES-INCOMPLETE")
    return sorted(findings)
