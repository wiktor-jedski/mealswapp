#!/usr/bin/env python3

# Implements DESIGN-010 RouteHandler, CSRFValidator, SecurityHeaderMiddleware, and DESIGN-013 TLSEnforcer live UAT verification.

import argparse
import importlib.util
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request
from email.message import Message
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[1]
BACKEND = ROOT / "backend"
STACK_SCRIPT = ROOT / "scripts" / "verify-local-stack.py"
SECURITY_HEADERS = {
	"Content-Security-Policy": "default-src 'self'",
	"X-Frame-Options": "DENY",
	"X-Content-Type-Options": "nosniff",
	"Referrer-Policy": "strict-origin-when-cross-origin",
	"Permissions-Policy": "camera=(), microphone=(), geolocation=()",
}
PROBES = ("/health", "/ready", "/api/v1/health", "/api/v1/ready")


class NoRedirect(urllib.request.HTTPRedirectHandler):
	def redirect_request(self, req: urllib.request.Request, fp: Any, code: int, msg: str, headers: Message, newurl: str) -> None:
		return None


def load_stack_module() -> Any:
	spec = importlib.util.spec_from_file_location("verify_local_stack", STACK_SCRIPT)
	if spec is None or spec.loader is None:
		raise RuntimeError(f"cannot load {STACK_SCRIPT}")
	module = importlib.util.module_from_spec(spec)
	spec.loader.exec_module(module)
	return module


def decode_body(raw: bytes, headers: Message) -> Any:
	content_type = headers.get("Content-Type", "")
	if raw and "json" in content_type:
		return json.loads(raw.decode("utf-8"))
	return raw.decode("utf-8") if raw else None


def request(url: str, headers: dict[str, str] | None = None, follow_redirects: bool = True) -> tuple[int, Message, bytes, Any]:
	opener = urllib.request.build_opener()
	if not follow_redirects:
		opener = urllib.request.build_opener(NoRedirect)
	req = urllib.request.Request(url, headers=headers or {})
	try:
		with opener.open(req, timeout=10) as response:
			raw = response.read()
			return response.status, response.headers, raw, decode_body(raw, response.headers)
	except urllib.error.HTTPError as exc:
		raw = exc.read()
		return exc.code, exc.headers, raw, decode_body(raw, exc.headers)


def assert_status(status: int, expected: int, label: str, body: Any) -> None:
	if status != expected:
		raise AssertionError(f"{label}: status {status}, want {expected}: {body!r}")


def assert_security_headers(headers: Message, label: str, require_hsts: bool = False) -> None:
	for name, expected in SECURITY_HEADERS.items():
		actual = headers.get(name)
		if actual != expected:
			raise AssertionError(f"{label}: {name} = {actual!r}, want {expected!r}")
	if require_hsts and not headers.get("Strict-Transport-Security", "").startswith("max-age="):
		raise AssertionError(f"{label}: missing Strict-Transport-Security header")


def assert_probe(base_url: str, path: str) -> None:
	status, headers, _, payload = request(f"{base_url}{path}")
	assert_status(status, 200, path, payload)
	assert_security_headers(headers, path)
	if not isinstance(payload, dict) or payload.get("status") not in {"ok", "ready"} or not payload.get("requestId"):
		raise AssertionError(f"{path}: invalid envelope: {payload!r}")
	if path.endswith("/ready"):
		data = payload.get("data")
		checks = data.get("checks") if isinstance(data, dict) else None
		if not isinstance(checks, dict) or checks.get("postgres") != "ok" or checks.get("redis") != "ok":
			raise AssertionError(f"{path}: readiness dependencies are not ok: {payload!r}")


def set_cookie_headers(headers: Message) -> list[str]:
	return headers.get_all("Set-Cookie") or []


def assert_csrf_token(base_url: str) -> None:
	status, headers, _, payload = request(f"{base_url}/api/v1/auth/csrf-token")
	assert_status(status, 200, "CSRF token", payload)
	assert_security_headers(headers, "CSRF token")
	if not isinstance(payload, dict) or payload.get("status") != "ok" or not payload.get("requestId"):
		raise AssertionError(f"CSRF token: invalid envelope: {payload!r}")
	data = payload.get("data")
	if not isinstance(data, dict) or not isinstance(data.get("csrfToken"), str) or not data["csrfToken"]:
		raise AssertionError(f"CSRF token: missing csrfToken: {payload!r}")
	cookies = set_cookie_headers(headers)
	for name in ("mealswapp_csrf", "mealswapp_session"):
		matching = [cookie for cookie in cookies if cookie.startswith(f"{name}=")]
		if not matching:
			raise AssertionError(f"CSRF token: missing {name} cookie: {cookies!r}")
		lowered = matching[0].lower()
		if "httponly" not in lowered or "samesite=strict" not in lowered:
			raise AssertionError(f"CSRF token: insecure {name} cookie attributes: {matching[0]!r}")


def run_httpapi_gateway_tests() -> None:
	env = {
		**os.environ,
		"GOCACHE": str(BACKEND / ".go-cache"),
		"GOMODCACHE": str(BACKEND / ".go-mod-cache"),
	}
	command = ["go", "test", "./internal/httpapi/...", "-count=1"]
	print(f"+ {' '.join(command)}")
	subprocess.run(command, cwd=BACKEND, env=env, text=True, check=True)


def start_tls_api(stack: Any, port: int) -> subprocess.Popen[str]:
	env = {
		**os.environ,
		**stack.backend_env(port),
		"MEALSWAPP_ENFORCE_TLS": "true",
	}
	print(f"+ go run ./cmd/api  # MEALSWAPP_HTTP_PORT={port} MEALSWAPP_ENFORCE_TLS=true")
	return subprocess.Popen(
		["go", "run", "./cmd/api"],
		cwd=BACKEND,
		text=True,
		env=env,
		stdout=subprocess.PIPE,
		stderr=subprocess.STDOUT,
	)


def assert_tls_redirect_spoof_resistant(stack: Any) -> None:
	port = stack.free_port()
	process = start_tls_api(stack, port)
	try:
		base_url = f"http://127.0.0.1:{port}"
		wait_for_redirect(f"{base_url}/health")
		status, headers, _, payload = request(
			f"{base_url}/health",
			headers={"X-Forwarded-Proto": "https"},
			follow_redirects=False,
		)
		if status not in {301, 308}:
			raise AssertionError(f"TLS redirect: status {status}, want redirect despite spoofed header: {payload!r}")
		location = headers.get("Location", "")
		if not location.startswith("https://"):
			raise AssertionError(f"TLS redirect: Location = {location!r}, want https URL")
	finally:
		stack.stop_process(process)


def wait_for_redirect(url: str, timeout: float = 30.0) -> None:
	deadline = time.monotonic() + timeout
	last_error: Exception | None = None
	while time.monotonic() < deadline:
		try:
			status, _, _, _ = request(url, follow_redirects=False)
			if status in {301, 308}:
				return
		except OSError as exc:
			last_error = exc
		time.sleep(0.5)
	raise TimeoutError(f"{url} did not return a redirect within {timeout:.0f}s: {last_error}")


def verify_phase02_uat(base_url: str, stack: Any) -> None:
	for path in PROBES:
		assert_probe(base_url, path)
	assert_csrf_token(base_url)
	run_httpapi_gateway_tests()
	assert_tls_redirect_spoof_resistant(stack)


def main() -> int:
	parser = argparse.ArgumentParser(description="Run Phase 02 live API UAT verification.")
	parser.add_argument("--keep-services", action="store_true", help="leave services started by this command running")
	args = parser.parse_args()

	stack = load_stack_module()
	if not stack.can_use_docker_compose():
		raise SystemExit("docker compose is required for Phase 02 UAT verification")

	started_services: set[str] = set()
	api_process = None
	try:
		initially_running = stack.running_compose_services()
		started_services = set(stack.COMPOSE_SERVICES) - initially_running
		stack.run(["docker", "compose", "up", "-d", *stack.COMPOSE_SERVICES])
		for service in stack.COMPOSE_SERVICES:
			stack.wait_for_compose_health(service)

		stack.run_migrations()
		port = stack.free_port()
		api_process = stack.start_api(port)
		base_url = f"http://127.0.0.1:{port}"
		stack.wait_for_http(f"{base_url}/health")
		verify_phase02_uat(base_url, stack)

		print("Phase 02 UAT verification passed.")
		return 0
	finally:
		if api_process is not None:
			stack.stop_process(api_process)
		if not args.keep_services:
			for service in sorted(started_services):
				stack.run(["docker", "compose", "stop", service])


if __name__ == "__main__":
	sys.exit(main())
