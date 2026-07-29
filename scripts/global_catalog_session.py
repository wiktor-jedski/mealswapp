"""Shared interactive authenticated session for global catalog operators.

Implements DESIGN-009 AdminController operator credential boundary.
"""

from __future__ import annotations

import getpass
import http.cookiejar
import json
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from typing import Any


class SessionError(ValueError):
    """Represent a safe authentication or protocol failure."""


class AuthorizationError(SessionError):
    """Represent authentication or authorization failure."""


@dataclass
class APIResponse:
    """Carry one decoded response without persisting session credentials."""

    status: int
    body: Any
    headers: Any


def prompt_credentials() -> tuple[str, str]:
    """Read administrator credentials interactively without argument exposure."""
    email_address = input("Administrator email: ").strip()
    if not email_address:
        raise SessionError("administrator email is required")
    return email_address, getpass.getpass("Administrator password: ")


class AuthenticatedSession:
    """Use a process-memory-only cookie jar for operator requests."""

    def __init__(self, base_url: str):
        parsed = urllib.parse.urlsplit(base_url)
        if parsed.scheme not in {"http", "https"} or not parsed.netloc or parsed.username or parsed.password:
            raise SessionError("base URL is invalid")
        self.base_url = base_url.rstrip("/")
        self.jar = http.cookiejar.CookieJar()
        self.opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.jar))
        self.csrf = ""

    def open(self, method: str, path: str, body: Any = None, headers: dict[str, str] | None = None):
        """Open one request; callers own and close the returned response stream."""
        payload = None if body is None else json.dumps(body, separators=(",", ":"), sort_keys=True).encode()
        request = urllib.request.Request(self.base_url + path, data=payload, method=method)
        request.add_header("Accept", "application/json")
        if payload is not None:
            request.add_header("Content-Type", "application/json")
        for key, value in (headers or {}).items():
            request.add_header(key, value)
        try:
            return self.opener.open(request, timeout=30)
        except urllib.error.HTTPError as error:
            return error

    def request(self, method: str, path: str, body: Any = None, headers: dict[str, str] | None = None) -> APIResponse:
        """Send one request and decode bounded control-plane JSON."""
        response = self.open(method, path, body, headers)
        try:
            raw = response.read()
        finally:
            response.close()
        try:
            decoded = json.loads(raw) if raw else None
        except json.JSONDecodeError:
            decoded = None
        return APIResponse(response.status, decoded, response.headers)

    def login(self, email_address: str, password: str) -> None:
        """Authenticate and acquire fresh in-memory CSRF state."""
        response = self.request("POST", "/api/v1/auth/login", {"email": email_address, "password": password})
        if response.status != 200:
            raise AuthorizationError("authentication failed")
        csrf = self.request("GET", "/api/v1/auth/csrf-token")
        try:
            token = csrf.body["data"]["csrfToken"]
        except (TypeError, KeyError):
            token = ""
        if csrf.status != 200 or not isinstance(token, str) or not token:
            raise AuthorizationError("fresh CSRF acquisition failed")
        self.csrf = token
