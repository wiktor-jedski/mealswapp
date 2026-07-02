#!/usr/bin/env python3

# Implements DESIGN-006 AuthController, DESIGN-008 DataExporter, DESIGN-010 CSRFValidator, and DESIGN-015 DisclaimerRenderer live UAT verification.

import argparse
import http.cookiejar
import importlib.util
import json
import sys
import time
import urllib.error
import urllib.request
from email.message import Message
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
STACK_SCRIPT = ROOT / "scripts" / "verify-local-stack.py"
ACCESS_COOKIE = "mealswapp_access"
REFRESH_COOKIE = "mealswapp_refresh"
PRIVACY_VERSION = "dev-privacy-v1"
TERMS_VERSION = "dev-terms-v1"


def load_stack_module() -> Any:
	spec = importlib.util.spec_from_file_location("verify_local_stack", STACK_SCRIPT)
	if spec is None or spec.loader is None:
		raise RuntimeError(f"cannot load {STACK_SCRIPT}")
	module = importlib.util.module_from_spec(spec)
	spec.loader.exec_module(module)
	return module


class APIClient:
	def __init__(self, base_url: str) -> None:
		self.base_url = base_url
		self.cookies = http.cookiejar.CookieJar()
		self.opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.cookies))

	def request(
		self,
		method: str,
		path: str,
		body: dict[str, Any] | None = None,
		headers: dict[str, str] | None = None,
	) -> tuple[int, Message, bytes, Any]:
		data = None
		request_headers = dict(headers or {})
		if body is not None:
			data = json.dumps(body).encode("utf-8")
			request_headers["Content-Type"] = "application/json"
		request = urllib.request.Request(
			f"{self.base_url}{path}",
			data=data,
			headers=request_headers,
			method=method,
		)
		try:
			with self.opener.open(request, timeout=10) as response:
				raw = response.read()
				return response.status, response.headers, raw, decode_body(raw, response.headers)
		except urllib.error.HTTPError as exc:
			raw = exc.read()
			return exc.code, exc.headers, raw, decode_body(raw, exc.headers)


def decode_body(raw: bytes, headers: Message) -> Any:
	content_type = headers.get("Content-Type", "")
	if raw and "json" in content_type:
		return json.loads(raw.decode("utf-8"))
	return raw.decode("utf-8") if raw else None


def assert_status(status: int, expected: int, label: str, body: Any) -> None:
	if status != expected:
		raise AssertionError(f"{label}: status {status}, want {expected}: {body!r}")


def assert_envelope(payload: Any, label: str) -> dict[str, Any]:
	if not isinstance(payload, dict):
		raise AssertionError(f"{label}: response is not a JSON object: {payload!r}")
	if payload.get("status") != "ok" or not payload.get("requestId"):
		raise AssertionError(f"{label}: invalid envelope: {payload!r}")
	data = payload.get("data")
	if not isinstance(data, dict):
		raise AssertionError(f"{label}: envelope data is not an object: {payload!r}")
	return data


def set_cookie_headers(headers: Message) -> list[str]:
	return headers.get_all("Set-Cookie") or []


def assert_auth_cookies(headers: Message, label: str) -> None:
	cookies = set_cookie_headers(headers)
	for name in (ACCESS_COOKIE, REFRESH_COOKIE):
		matching = [cookie for cookie in cookies if cookie.startswith(f"{name}=")]
		if not matching:
			raise AssertionError(f"{label}: missing {name} Set-Cookie header: {cookies!r}")
		if "httponly" not in matching[0].lower():
			raise AssertionError(f"{label}: {name} cookie is not HttpOnly: {matching[0]!r}")


def assert_cleared_auth_cookies(headers: Message, label: str) -> None:
	cookies = set_cookie_headers(headers)
	for name in (ACCESS_COOKIE, REFRESH_COOKIE):
		matching = [cookie for cookie in cookies if cookie.startswith(f"{name}=")]
		if not matching:
			raise AssertionError(f"{label}: missing cleared {name} cookie: {cookies!r}")
		lowered = matching[0].lower()
		if "max-age=0" not in lowered and "expires=" not in lowered:
			raise AssertionError(f"{label}: {name} cookie was not cleared: {matching[0]!r}")


def assert_no_token_leak(payload: Any, label: str) -> None:
	raw = json.dumps(payload, sort_keys=True)
	for key in ("accessToken", "refreshToken", "passwordResetToken", "resetToken"):
		if key in raw:
			raise AssertionError(f"{label}: leaked token field {key}: {payload!r}")


def fetch_csrf(client: APIClient) -> str:
	status, _, _, payload = client.request("GET", "/api/v1/auth/csrf-token")
	assert_status(status, 200, "fetch CSRF token", payload)
	data = assert_envelope(payload, "fetch CSRF token")
	token = data.get("csrfToken")
	if not isinstance(token, str) or not token:
		raise AssertionError(f"fetch CSRF token: missing csrfToken: {payload!r}")
	return token


def verify_phase03_uat(base_url: str) -> None:
	client = APIClient(base_url)
	email = f"phase03-uat-{int(time.time() * 1000)}@example.test"
	password = "StrongerPassword1!"

	status, headers, _, payload = client.request(
		"POST",
		"/api/v1/auth/register",
		{
			"email": email,
			"password": password,
			"privacyPolicyVersion": PRIVACY_VERSION,
			"termsVersion": TERMS_VERSION,
		},
	)
	assert_status(status, 201, "register", payload)
	register_data = assert_envelope(payload, "register")
	if not register_data.get("userId"):
		raise AssertionError(f"register: missing userId: {payload!r}")
	assert_auth_cookies(headers, "register")
	assert_no_token_leak(payload, "register")

	csrf = fetch_csrf(client)
	status, _, _, payload = client.request(
		"PUT",
		"/api/v1/profile",
		{"displayName": "Phase UAT", "unitSystem": "imperial", "themePreference": "dark"},
		{"X-CSRF-Token": csrf},
	)
	assert_status(status, 200, "update profile", payload)
	profile_data = assert_envelope(payload, "update profile")
	if profile_data.get("displayName") != "Phase UAT" or profile_data.get("unitSystem") != "imperial":
		raise AssertionError(f"update profile: unexpected data: {payload!r}")

	status, _, _, payload = client.request("GET", "/api/v1/profile")
	assert_status(status, 200, "read profile", payload)
	profile_data = assert_envelope(payload, "read profile")
	if profile_data.get("displayName") != "Phase UAT" or profile_data.get("themePreference") != "dark":
		raise AssertionError(f"read profile: unexpected data: {payload!r}")

	status, _, _, payload = client.request("POST", "/api/v1/auth/verify-email", {}, {"X-CSRF-Token": csrf})
	assert_status(status, 200, "verify email", payload)
	verify_data = assert_envelope(payload, "verify email")
	if verify_data.get("hasVerifiedLoginMethod") is not True:
		raise AssertionError(f"verify email: expected verified login method: {payload!r}")

	for reset_email in (email, "missing-phase03-uat@example.test"):
		status, _, _, payload = client.request(
			"POST",
			"/api/v1/auth/password-reset/request",
			{"email": reset_email},
		)
		assert_status(status, 200, f"password reset request {reset_email}", payload)
		reset_data = assert_envelope(payload, f"password reset request {reset_email}")
		if reset_data.get("accepted") is not True:
			raise AssertionError(f"password reset request {reset_email}: not accepted: {payload!r}")
		assert_no_token_leak(payload, f"password reset request {reset_email}")

	status, _, _, payload = client.request("GET", "/api/v1/saved-items?kind=favorite")
	assert_status(status, 200, "list saved items", payload)
	if not isinstance(assert_envelope(payload, "list saved items").get("items"), list):
		raise AssertionError(f"list saved items: items is not a list: {payload!r}")

	status, _, _, payload = client.request("GET", "/api/v1/search-history")
	assert_status(status, 200, "list search history", payload)
	if not isinstance(assert_envelope(payload, "list search history").get("history"), list):
		raise AssertionError(f"list search history: history is not a list: {payload!r}")

	status, headers, _, payload = client.request("GET", "/api/v1/account/export?format=json")
	assert_status(status, 200, "JSON export", payload)
	if "application/json" not in headers.get("Content-Type", ""):
		raise AssertionError(f"JSON export: wrong content type: {headers.get('Content-Type')!r}")
	if payload.get("user", {}).get("email") != email or payload.get("customItems") != []:
		raise AssertionError(f"JSON export: unexpected bundle: {payload!r}")
	if "format" in payload:
		raise AssertionError(f"JSON export: obsolete format field is present: {payload!r}")

	status, headers, raw, payload = client.request("GET", "/api/v1/account/export?format=csv")
	assert_status(status, 200, "CSV export", payload)
	if "text/csv" not in headers.get("Content-Type", ""):
		raise AssertionError(f"CSV export: wrong content type: {headers.get('Content-Type')!r}")
	if b"customItems,count,0" not in raw:
		raise AssertionError(f"CSV export: missing custom item fallback row: {raw.decode('utf-8')!r}")

	for location in ("login", "account"):
		status, _, _, payload = client.request("GET", f"/api/v1/disclaimers?location={location}")
		assert_status(status, 200, f"{location} disclaimer", payload)
		disclaimer = assert_envelope(payload, f"{location} disclaimer")
		if disclaimer.get("location") != location or not disclaimer.get("markdown"):
			raise AssertionError(f"{location} disclaimer: unexpected payload: {payload!r}")

	status, headers, _, payload = client.request("GET", "/api/v1/auth/oauth/google/start")
	if not (500 <= status < 600):
		raise AssertionError(f"OAuth start: expected fail-closed 5xx, got {status}: {payload!r}")
	if headers.get("Location"):
		raise AssertionError(f"OAuth start: unexpected redirect while provider is unavailable: {headers.get('Location')!r}")

	status, headers, _, payload = client.request("POST", "/api/v1/auth/logout", {}, {"X-CSRF-Token": csrf})
	assert_status(status, 204, "logout", payload)
	assert_cleared_auth_cookies(headers, "logout")

	status, _, _, payload = client.request("GET", "/api/v1/profile")
	assert_status(status, 401, "profile after logout", payload)

	status, headers, _, payload = client.request(
		"POST",
		"/api/v1/auth/login",
		{"email": email, "password": password},
	)
	assert_status(status, 200, "login after logout", payload)
	assert_auth_cookies(headers, "login after logout")

	csrf = fetch_csrf(client)
	status, headers, _, payload = client.request("DELETE", "/api/v1/account", {}, {"X-CSRF-Token": csrf})
	assert_status(status, 200, "delete account", payload)
	delete_data = assert_envelope(payload, "delete account")
	if delete_data.get("status") != "pending" or not delete_data.get("requestId"):
		raise AssertionError(f"delete account: unexpected payload: {payload!r}")
	assert_cleared_auth_cookies(headers, "delete account")

	status, _, _, payload = client.request(
		"POST",
		"/api/v1/auth/login",
		{"email": email, "password": password},
	)
	assert_status(status, 401, "login after deletion request", payload)

	for path in ("/api/v1/profile", "/api/v1/account/export"):
		status, _, _, payload = client.request("GET", path)
		assert_status(status, 401, f"{path} after deletion request", payload)


def main() -> int:
	parser = argparse.ArgumentParser(description="Run Phase 03 live API UAT verification.")
	parser.add_argument("--keep-services", action="store_true", help="leave services started by this command running")
	args = parser.parse_args()

	stack = load_stack_module()
	started_services: set[str] = set()
	api_process = None
	try:
		# Implements DESIGN-014 MetricsCollector aggregate gate reuse of local dependencies.
		started_services = stack.ensure_local_dependencies()
		stack.run_migrations()
		port = stack.free_port()
		api_process = stack.start_api(port)
		base_url = f"http://127.0.0.1:{port}"
		stack.wait_for_http(f"{base_url}/health")
		verify_phase03_uat(base_url)

		print("Phase 03 UAT verification passed.")
		return 0
	finally:
		if api_process is not None:
			stack.stop_process(api_process)
		if not args.keep_services:
			for service in sorted(started_services):
				stack.run(["docker", "compose", "stop", service])


if __name__ == "__main__":
	sys.exit(main())
