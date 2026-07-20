#!/usr/bin/env python3

# Implements DESIGN-011 RedisCache and DESIGN-005 RepositoryInterfaces local stack verification.

import contextlib
import argparse
import json
import os
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
BACKEND = ROOT / "backend"
# Destructive migration verification must never target the development database.
# Implements DESIGN-005 RepositoryInterfaces isolated integration-test persistence.
TEST_DATABASE_NAME = "mealswapp_test"
DATABASE_URL = f"postgres://mealswapp:mealswapp@localhost:5432/{TEST_DATABASE_NAME}?sslmode=disable"
REDIS_URL = "redis://localhost:6379/0"
COMPOSE_SERVICES = ("postgres", "redis")
HEALTH_ENDPOINTS = ("/health", "/ready", "/api/v1/health", "/api/v1/ready")


def run(command: list[str], cwd: Path = ROOT, env: dict[str, str] | None = None, capture: bool = False) -> subprocess.CompletedProcess[str]:
	print(f"+ {' '.join(command)}")
	return subprocess.run(
		command,
		cwd=cwd,
		check=True,
		text=True,
		env={**os.environ, **(env or {})},
		capture_output=capture,
	)


def can_use_docker_compose() -> bool:
	try:
		subprocess.run(["docker", "compose", "version"], cwd=ROOT, check=True, capture_output=True, text=True)
	except (FileNotFoundError, subprocess.CalledProcessError):
		return False
	return True


def port_accepts_connection(host: str, port: int) -> bool:
	try:
		with socket.create_connection((host, port), timeout=1):
			return True
	except OSError:
		return False


def local_dependency_ports_accept_connections() -> bool:
	return port_accepts_connection("127.0.0.1", 5432) and port_accepts_connection("127.0.0.1", 6379)


def running_compose_services() -> set[str]:
	result = run(["docker", "compose", "ps", "--status", "running", "--services"], capture=True)
	return {line.strip() for line in result.stdout.splitlines() if line.strip()}


def service_health(service: str) -> str:
	result = run(["docker", "compose", "ps", "--format", "json", service], capture=True)
	for line in result.stdout.splitlines():
		if not line.strip():
			continue
		payload = json.loads(line)
		return str(payload.get("Health") or payload.get("State") or "").lower()
	return ""


def wait_for_compose_health(service: str, timeout: float = 60.0) -> None:
	deadline = time.monotonic() + timeout
	while time.monotonic() < deadline:
		health = service_health(service)
		if health in {"healthy", "running"}:
			return
		time.sleep(1)
	raise TimeoutError(f"{service} did not become healthy within {timeout:.0f}s")


def free_port() -> int:
	with contextlib.closing(socket.socket(socket.AF_INET, socket.SOCK_STREAM)) as sock:
		sock.bind(("127.0.0.1", 0))
		return int(sock.getsockname()[1])


def backend_env(port: int | None = None) -> dict[str, str]:
	env = {
		"GOCACHE": str(BACKEND / ".go-cache"),
		"GOMODCACHE": str(BACKEND / ".go-mod-cache"),
		"MEALSWAPP_DATABASE_URL": DATABASE_URL,
		"MEALSWAPP_REDIS_URL": REDIS_URL,
		"MEALSWAPP_ENV": "development",
	}
	if port is not None:
		env["MEALSWAPP_HTTP_PORT"] = str(port)
	return env


def run_migrations() -> None:
	run(["go", "run", "./cmd/migrate", "up"], BACKEND, backend_env())
	run(["go", "run", "./cmd/migrate", "down"], BACKEND, backend_env())
	run(["go", "run", "./cmd/migrate", "up"], BACKEND, backend_env())


def ensure_test_database() -> None:
	# Implements DESIGN-005 RepositoryInterfaces fresh-stack test database bootstrap.
	query = f"SELECT 1 FROM pg_database WHERE datname = '{TEST_DATABASE_NAME}'"
	result = run([
		"docker", "compose", "exec", "-T", "postgres",
		"psql", "-U", "mealswapp", "-d", "postgres", "-tAc", query,
	], capture=True)
	if result.stdout.strip() == "1":
		return
	run([
		"docker", "compose", "exec", "-T", "postgres",
		"createdb", "-U", "mealswapp", TEST_DATABASE_NAME,
	])


def ensure_local_dependencies() -> set[str]:
	# Implements DESIGN-014 MetricsCollector aggregate gate reuse of local dependencies.
	if local_dependency_ports_accept_connections():
		print("Using existing local PostgreSQL and Redis services on ports 5432 and 6379.")
		return set()
	if not can_use_docker_compose():
		raise SystemExit("docker compose or existing local PostgreSQL/Redis services are required for local stack verification")

	initially_running = running_compose_services()
	started_services = set(COMPOSE_SERVICES) - initially_running
	run(["docker", "compose", "up", "-d", *COMPOSE_SERVICES])
	for service in COMPOSE_SERVICES:
		wait_for_compose_health(service)
	return started_services


def start_api(port: int) -> subprocess.Popen[str]:
	print(f"+ go run ./cmd/api  # MEALSWAPP_HTTP_PORT={port}")
	return subprocess.Popen(
		["go", "run", "./cmd/api"],
		cwd=BACKEND,
		text=True,
		env={**os.environ, **backend_env(port)},
		stdout=subprocess.PIPE,
		stderr=subprocess.STDOUT,
	)


def start_worker() -> subprocess.Popen[str]:
	# Implements DESIGN-014 MetricsCollector worker heartbeat readiness gate.
	print("+ go run ./cmd/worker")
	return subprocess.Popen(
		["go", "run", "./cmd/worker"],
		cwd=BACKEND,
		text=True,
		env={**os.environ, **backend_env()},
		stdout=subprocess.PIPE,
		stderr=subprocess.STDOUT,
	)


def stop_process(process: subprocess.Popen[str]) -> None:
	if process.poll() is not None:
		return
	process.terminate()
	try:
		process.wait(timeout=10)
	except subprocess.TimeoutExpired:
		process.kill()
		process.wait(timeout=5)


def wait_for_http(url: str, timeout: float = 30.0) -> None:
	deadline = time.monotonic() + timeout
	last_error: Exception | None = None
	while time.monotonic() < deadline:
		try:
			with urllib.request.urlopen(url, timeout=2) as response:
				if 200 <= response.status < 300:
					return
		except urllib.error.HTTPError as exc:
			last_error = exc
			if 200 <= exc.code < 500:
				return
		except OSError as exc:
			last_error = exc
		time.sleep(0.5)
	raise TimeoutError(f"{url} did not respond within {timeout:.0f}s: {last_error}")


def assert_endpoint(base_url: str, path: str) -> None:
	url = f"{base_url}{path}"
	print(f"+ GET {url}")
	with urllib.request.urlopen(url, timeout=5) as response:
		body = response.read().decode("utf-8")
		if response.status != 200:
			raise RuntimeError(f"{url} returned {response.status}: {body}")
		payload = json.loads(body)
		if not payload.get("requestId"):
			raise RuntimeError(f"{url} response is missing requestId: {body}")
		status = payload.get("status")
		if path.endswith("/ready") and status != "ready":
			raise RuntimeError(f"{url} status = {status!r}, want 'ready': {body}")
		if path.endswith("/health") and status != "ok":
			raise RuntimeError(f"{url} status = {status!r}, want 'ok': {body}")


def main() -> int:
	parser = argparse.ArgumentParser()
	parser.add_argument("--keep-services", action="store_true", help="leave services started by this command running")
	args = parser.parse_args()
	if not can_use_docker_compose():
		raise SystemExit("docker compose is required for local stack verification")

	started_services: set[str] = set()
	api_process: subprocess.Popen[str] | None = None
	worker_process: subprocess.Popen[str] | None = None
	try:
		started_services = ensure_local_dependencies()
		ensure_test_database()
		run_migrations()

		port = free_port()
		api_process = start_api(port)
		worker_process = start_worker()
		base_url = f"http://127.0.0.1:{port}"
		wait_for_http(f"{base_url}/health")
		wait_for_http(f"{base_url}/ready")
		for endpoint in HEALTH_ENDPOINTS:
			assert_endpoint(base_url, endpoint)

		print("Local stack verification passed.")
		return 0
	finally:
		if worker_process is not None:
			stop_process(worker_process)
		if api_process is not None:
			stop_process(api_process)
		if not args.keep_services:
			for service in sorted(started_services):
				run(["docker", "compose", "stop", service])


if __name__ == "__main__":
	sys.exit(main())
