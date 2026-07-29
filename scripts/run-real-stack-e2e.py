#!/usr/bin/env python3
"""Run Task 261 against a disposable PostgreSQL, Redis, API, and frontend stack."""

# Implements DESIGN-005 RepositoryInterfaces isolated real-stack test persistence.

from __future__ import annotations

import argparse
import contextlib
import dataclasses
import http.cookiejar
import json
import os
import re
import secrets
import shutil
import signal
import socket
import struct
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request
import zlib
from pathlib import Path
from typing import Iterable, Mapping, Sequence

REPO = Path(__file__).resolve().parent.parent
ARTIFACT_ROOT = REPO / "logs" / "real-stack-e2e"
RUN_ID_RE = re.compile(r"^[0-9a-f]{24}$")
DATABASE_RE = re.compile(r"^mealswapp_e2e_([0-9a-f]{24})_test$")
CONTAINER_RE = re.compile(r"^mealswapp-e2e-([0-9a-f]{24})$")
REQUEST_ID_RE = re.compile(r"\b[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\b", re.I)
OWNER_PREFIX = "mealswapp-e2e-owner"
REDIS_RUN_LABEL = "mealswapp.e2e.run"
REDIS_CREATED_LABEL = "mealswapp.e2e.created"
MIN_STALE_AGE_SECONDS = 900
DEFAULT_STALE_AGE_SECONDS = 3600


class SafetyError(RuntimeError):
    """Raised before an unsafe external target can be contacted."""


class ListenerExitedError(RuntimeError):
    """Raised when a reserved-port child exits before listener readiness."""


@dataclasses.dataclass(frozen=True)
class PostgresTarget:
    """Validated loopback PostgreSQL maintenance connection."""

    host: str
    port: int
    user: str
    password: str
    database: str

    @classmethod
    def parse(cls, value: str) -> "PostgresTarget":
        parsed = urllib.parse.urlsplit(value)
        if parsed.scheme not in {"postgres", "postgresql"}:
            raise SafetyError("PostgreSQL administrator URL must use postgres")
        if parsed.hostname not in {"127.0.0.1", "::1"}:
            raise SafetyError("PostgreSQL administrator host must be loopback")
        if parsed.path != "/postgres":
            raise SafetyError("PostgreSQL administrator database must be postgres")
        if not parsed.username or parsed.port is None:
            raise SafetyError("PostgreSQL administrator URL is incomplete")
        return cls(
            host=parsed.hostname,
            port=parsed.port,
            user=urllib.parse.unquote(parsed.username),
            password=urllib.parse.unquote(parsed.password or ""),
            database="postgres",
        )

    def command(self, database: str | None = None) -> list[str]:
        return ["psql", "-X", "-v", "ON_ERROR_STOP=1", "-h", self.host, "-p", str(self.port), "-U", self.user, "-d", database or self.database]

    def environment(self) -> dict[str, str]:
        return {**os.environ, "PGPASSWORD": self.password}

    def database_url(self, database: str) -> str:
        validate_database_name(database)
        authority = f"{urllib.parse.quote(self.user, safe='')}:{urllib.parse.quote(self.password, safe='')}@{self.host}:{self.port}"
        return f"postgres://{authority}/{database}?sslmode=disable"


@dataclasses.dataclass
class OwnedProcess:
    """One child process with Linux PID-reuse protection metadata."""

    name: str
    process: subprocess.Popen[bytes]
    start_token: str


@dataclasses.dataclass
class PortReservation:
    """A loopback port kept bound until its managed child is ready to exec."""

    listener: socket.socket
    port: int

    def release(self) -> None:
        self.listener.close()


@dataclasses.dataclass
class SignalController:
    """Queue handled signals once teardown starts instead of interrupting cleanup."""

    received: int | None = None
    tearing_down: bool = False

    def handle(self, signum: int, _frame: object) -> None:
        self.received = signum
        if not self.tearing_down:
            raise InterruptedError(f"received signal {signal.Signals(signum).name}")

    def begin_teardown(self) -> None:
        self.tearing_down = True

    def end_teardown(self) -> None:
        self.tearing_down = False


def validate_run_id(run_id: str) -> str:
    if not RUN_ID_RE.fullmatch(run_id):
        raise SafetyError("run identifier is unsafe")
    return run_id


def database_name(run_id: str) -> str:
    return f"mealswapp_e2e_{validate_run_id(run_id)}_test"


def ownership_comment(run_id: str, created_at: int) -> str:
    return f"{OWNER_PREFIX} run={validate_run_id(run_id)} created={created_at}"


def validate_database_name(name: str, expected_run_id: str | None = None) -> str:
    match = DATABASE_RE.fullmatch(name)
    if not match or name == "mealswapp":
        raise SafetyError("database name is not an owned E2E test database")
    if expected_run_id is not None and match.group(1) != validate_run_id(expected_run_id):
        raise SafetyError("database name does not match the run")
    return name


def validate_environment(environment: str) -> None:
    if environment != "development":
        raise SafetyError("real-stack E2E environment must be development")


def validate_stale_age(age_seconds: int) -> int:
    if age_seconds < MIN_STALE_AGE_SECONDS:
        raise SafetyError(f"stale cleanup age must be at least {MIN_STALE_AGE_SECONDS} seconds")
    return age_seconds


def run_command(
    args: Sequence[str],
    *,
    cwd: Path = REPO,
    env: Mapping[str, str] | None = None,
    input_text: str | None = None,
    timeout: float = 180,
) -> subprocess.CompletedProcess[str]:
    process = subprocess.Popen(
        list(args),
        cwd=cwd,
        env=dict(env) if env is not None else None,
        text=True,
        stdin=subprocess.PIPE if input_text is not None else None,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        start_new_session=True,
    )
    try:
        stdout, stderr = process.communicate(input=input_text, timeout=timeout)
    except BaseException:
        rollback_spawned_process(process)
        with contextlib.suppress(subprocess.TimeoutExpired):
            process.communicate(timeout=1)
        raise
    if process.returncode:
        raise subprocess.CalledProcessError(process.returncode, list(args), output=stdout, stderr=stderr)
    return subprocess.CompletedProcess(list(args), process.returncode, stdout, stderr)


def process_group_exists(process_group_id: int) -> bool:
    try:
        os.killpg(process_group_id, 0)
    except ProcessLookupError:
        return False
    for stat_path in Path("/proc").glob("[0-9]*/stat"):
        try:
            stat = stat_path.read_text(encoding="utf-8")
            fields = stat[stat.rfind(")") + 2 :].split()
            if int(fields[2]) == process_group_id and fields[0] != "Z":
                return True
        except (OSError, ValueError, IndexError):
            continue
    return False


def rollback_spawned_process(process: subprocess.Popen[object]) -> None:
    """Terminate the exact process group created by this Popen boundary."""

    process_group_id = process.pid
    with contextlib.suppress(ProcessLookupError):
        os.killpg(process_group_id, signal.SIGTERM)
    try:
        process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        pass
    if process_group_exists(process_group_id):
        with contextlib.suppress(ProcessLookupError):
            os.killpg(process_group_id, signal.SIGKILL)
    with contextlib.suppress(subprocess.TimeoutExpired):
        process.wait(timeout=5)
    if process_group_exists(process_group_id):
        raise RuntimeError("managed process group did not terminate")


def psql(target: PostgresTarget, sql: str, *, database: str | None = None) -> str:
    result = run_command(
        [*target.command(database), "-Atqc", sql],
        env=target.environment(),
    )
    return result.stdout.strip()


def create_database(target: PostgresTarget, name: str, comment: str) -> None:
    run_id = DATABASE_RE.fullmatch(validate_database_name(name)).group(1)  # type: ignore[union-attr]
    parsed_run_id, _ = parse_ownership_comment(comment)
    if parsed_run_id != run_id:
        raise SafetyError("database name and ownership comment identify different runs")
    if psql(target, f"SELECT 1 FROM pg_database WHERE datname = '{name}'") == "1":
        raise SafetyError("owned database name already exists")
    psql(target, f'CREATE DATABASE "{name}"')
    try:
        psql(target, f"""COMMENT ON DATABASE "{name}" IS '{comment}'""")
    except Exception:
        with contextlib.suppress(Exception):
            psql(target, f'DROP DATABASE "{name}"')
        raise


def database_comment(target: PostgresTarget, name: str) -> str:
    validate_database_name(name)
    return psql(target, f"SELECT COALESCE(shobj_description(oid, 'pg_database'), '<unmarked>') FROM pg_database WHERE datname = '{name}'")


def drop_owned_database(target: PostgresTarget, name: str, run_id: str, expected_comment: str) -> bool:
    validate_database_name(name, run_id)
    if ownership_comment_run(expected_comment) != run_id:
        raise SafetyError("expected database ownership comment does not match the run")
    actual = database_comment(target, name)
    if not actual:
        return False
    if actual != expected_comment:
        raise SafetyError("database ownership comment does not match")
    psql(target, f"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '{name}' AND pid <> pg_backend_pid()")
    psql(target, f'DROP DATABASE "{name}"')
    return True


def ownership_comment_run(comment: str) -> str:
    return parse_ownership_comment(comment)[0]


def parse_ownership_comment(comment: str) -> tuple[str, int]:
    match = re.fullmatch(rf"{re.escape(OWNER_PREFIX)} run=([0-9a-f]{{24}}) created=([0-9]+)", comment)
    if not match:
        raise SafetyError("database ownership comment is invalid")
    return match.group(1), int(match.group(2))


def validate_ownership_comment(comment: str, run_id: str, created_at: int) -> str:
    expected = ownership_comment(run_id, created_at)
    if comment != expected:
        raise SafetyError("database ownership comment metadata is inconsistent")
    return comment


def docker_inspect(container: str) -> dict[str, object] | None:
    if not CONTAINER_RE.fullmatch(container):
        raise SafetyError("Redis container name is unsafe")
    try:
        output = run_command(["docker", "inspect", container], timeout=20).stdout
    except subprocess.CalledProcessError:
        return None
    values = json.loads(output)
    return values[0] if values else None


def inspect_labels(inspect: Mapping[str, object]) -> Mapping[str, str]:
    config = inspect.get("Config")
    if not isinstance(config, Mapping):
        return {}
    labels = config.get("Labels")
    return labels if isinstance(labels, Mapping) else {}


def remove_owned_redis(container: str, run_id: str) -> bool:
    validate_run_id(run_id)
    inspect = docker_inspect(container)
    if inspect is None:
        return False
    if inspect_labels(inspect).get(REDIS_RUN_LABEL) != run_id:
        raise SafetyError("Redis container ownership label does not match")
    run_command(["docker", "rm", "-f", container], timeout=30)
    return True


def reserve_ports(count: int) -> list[PortReservation]:
    if count < 1:
        raise ValueError("at least one port must be reserved")
    reservations: list[PortReservation] = []
    try:
        for _ in range(count):
            listener = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 0)
            reservations.append(PortReservation(listener, 0))
            listener.bind(("127.0.0.1", 0))
            reservations[-1].port = int(listener.getsockname()[1])
        ports = [reservation.port for reservation in reservations]
        if len(set(ports)) != len(ports):
            raise SafetyError("port reservations are not distinct")
        return reservations
    except BaseException:
        for reservation in reservations:
            reservation.release()
        raise


def reserve_port() -> int:
    reservation = reserve_ports(1)[0]
    try:
        return reservation.port
    finally:
        reservation.release()


def process_start_token(pid: int) -> str | None:
    try:
        return Path(f"/proc/{pid}/stat").read_text(encoding="utf-8").split()[21]
    except (OSError, IndexError):
        return None


def validate_process_start_token(start_token: str | None) -> str:
    if start_token is None or not re.fullmatch(r"[1-9][0-9]*", start_token):
        raise SafetyError("process ownership token is unavailable")
    return start_token


def stop_process(pid: int, start_token: str | None, *, group: bool = True) -> bool:
    expected = validate_process_start_token(start_token)
    if pid <= 1:
        raise SafetyError("process identifier is unsafe")
    current = process_start_token(pid)
    if current is None and not Path(f"/proc/{pid}").exists():
        return False
    if current is None:
        raise SafetyError("process ownership cannot be established")
    if current != expected:
        raise SafetyError("process ownership token does not match")
    if group and os.getpgid(pid) != pid:
        raise SafetyError("managed process is not its process-group leader")
    process_group_id = pid
    with contextlib.suppress(ProcessLookupError):
        (os.killpg(process_group_id, signal.SIGTERM) if group else os.kill(pid, signal.SIGTERM))
    deadline = time.monotonic() + 5
    while (process_group_exists(process_group_id) if group else process_start_token(pid) == expected) and time.monotonic() < deadline:
        time.sleep(0.05)
    if process_group_exists(process_group_id) if group else process_start_token(pid) == expected:
        with contextlib.suppress(ProcessLookupError):
            (os.killpg(process_group_id, signal.SIGKILL) if group else os.kill(pid, signal.SIGKILL))
    deadline = time.monotonic() + 5
    while (process_group_exists(process_group_id) if group else process_start_token(pid) == expected) and time.monotonic() < deadline:
        time.sleep(0.05)
    if process_group_exists(process_group_id) if group else process_start_token(pid) == expected:
        raise RuntimeError("managed process did not terminate")
    return True


def stop_owned_process(owned: OwnedProcess) -> bool:
    """Stop and reap a directly owned child plus every descendant in its group."""

    pid = owned.process.pid
    expected = validate_process_start_token(owned.start_token)
    current = process_start_token(pid)
    if current is None:
        if process_group_exists(pid):
            raise SafetyError("direct child ownership cannot be established")
        return False
    elif current != expected or os.getpgid(pid) != pid:
        raise SafetyError("direct child ownership does not match")
    with contextlib.suppress(ProcessLookupError):
        os.killpg(pid, signal.SIGTERM)
    try:
        owned.process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        pass
    if process_group_exists(pid):
        with contextlib.suppress(ProcessLookupError):
            os.killpg(pid, signal.SIGKILL)
    with contextlib.suppress(subprocess.TimeoutExpired):
        owned.process.wait(timeout=5)
    deadline = time.monotonic() + 5
    while process_group_exists(pid) and time.monotonic() < deadline:
        time.sleep(0.05)
    if process_group_exists(pid):
        raise RuntimeError("direct child process group did not terminate")
    return True


def wait_http(url: str, process: subprocess.Popen[bytes], timeout: float) -> None:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise ListenerExitedError("managed process exited before readiness")
        try:
            with urllib.request.urlopen(url, timeout=0.5) as response:
                if response.status < 500:
                    return
        except (urllib.error.URLError, TimeoutError):
            time.sleep(0.1)
    raise TimeoutError("managed process readiness timed out")


def request_json(
    opener: urllib.request.OpenerDirector,
    url: str,
    method: str,
    body: Mapping[str, object] | None = None,
    headers: Mapping[str, str] | None = None,
) -> tuple[int, dict[str, object]]:
    encoded = None if body is None else json.dumps(body).encode()
    request_headers = {"Accept": "application/json", **(headers or {})}
    if encoded is not None:
        request_headers["Content-Type"] = "application/json"
    request = urllib.request.Request(url, data=encoded, method=method, headers=request_headers)
    with opener.open(request, timeout=10) as response:
        parsed = json.loads(response.read())
        return response.status, parsed


def register_fixture(api_origin: str, run_id: str) -> tuple[dict[str, str], list[str]]:
    cookie_jar = http.cookiejar.CookieJar()
    opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cookie_jar))
    email = f"e2e-{run_id}@example.test"
    password = f"E2e-{run_id}-Password!"
    request_ids: list[str] = []
    status, registered = request_json(
        opener,
        f"{api_origin}/api/v1/auth/register",
        "POST",
        {
            "email": email,
            "password": password,
            "privacyPolicyVersion": "dev-privacy-v1",
            "termsVersion": "dev-terms-v1",
        },
    )
    user_id = str(data_field(registered, "userId"))
    request_ids.append(str(registered.get("requestId", "")))
    if status != 201 or not re.fullmatch(r"[0-9a-f-]{36}", user_id, re.I):
        raise RuntimeError("fixture registration failed")
    status, csrf_body = request_json(opener, f"{api_origin}/api/v1/auth/csrf-token", "GET")
    csrf = str(data_field(csrf_body, "csrfToken"))
    request_ids.append(str(csrf_body.get("requestId", "")))
    if status != 200 or not csrf:
        raise RuntimeError("fixture CSRF acquisition failed")
    status, verified = request_json(
        opener,
        f"{api_origin}/api/v1/auth/verify-email",
        "POST",
        headers={"X-CSRF-Token": csrf},
    )
    request_ids.append(str(verified.get("requestId", "")))
    if status != 200:
        raise RuntimeError("fixture verification failed")
    return {
        "email": email,
        "password": password,
        "user_id": user_id,
        "classification": f"E2E classification {run_id}",
        "private_item": f"E2E private {run_id}",
        "global_item": f"E2E global {run_id}",
    }, [value for value in request_ids if REQUEST_ID_RE.fullmatch(value)]


def data_field(body: Mapping[str, object], key: str) -> object:
    data = body.get("data")
    return data.get(key, "") if isinstance(data, Mapping) else ""


def safe_diagnostics(run_id: str, request_ids: Iterable[str], events: Iterable[str], result: str) -> dict[str, object]:
    validate_run_id(run_id)
    safe_requests = sorted({value for value in request_ids if REQUEST_ID_RE.fullmatch(value)})
    safe_events = [value for value in events if re.fullmatch(r"[a-z0-9_-]{1,64}", value)]
    return {"run_id": run_id, "request_ids": safe_requests, "events": safe_events, "result": result}


class Harness:
    """Own and tear down one complete E2E execution."""

    def __init__(
        self,
        target: PostgresTarget,
        *,
        timeout: float = 120,
        inject_assertion_failure: bool = False,
        inject_api_timeout: bool = False,
        signal_controller: SignalController | None = None,
    ) -> None:
        validate_environment(os.environ.get("MEALSWAPP_ENV", "development"))
        self.target = target
        self.timeout = timeout
        self.inject_assertion_failure = inject_assertion_failure
        self.inject_api_timeout = inject_api_timeout
        self.signal_controller = signal_controller
        self.run_id = secrets.token_hex(12)
        self.created_at = int(time.time())
        self.database = database_name(self.run_id)
        self.comment = ownership_comment(self.run_id, self.created_at)
        self.container = f"mealswapp-e2e-{self.run_id}"
        self.artifacts = ARTIFACT_ROOT / self.run_id
        self.processes: list[OwnedProcess] = []
        self.request_ids: list[str] = []
        self.events: list[str] = []
        self.raw_dir: Path | None = None
        self.state_path = self.artifacts / "state.json"

    def write_state(self, status: str) -> None:
        self.artifacts.mkdir(parents=True, exist_ok=True)
        payload = {
            "run_id": self.run_id,
            "created_at": self.created_at,
            "database": self.database,
            "ownership_comment": self.comment,
            "redis_container": self.container,
            "processes": [
                {"name": owned.name, "pid": owned.process.pid, "start_token": owned.start_token}
                for owned in self.processes
            ],
            "raw_directory": str(self.raw_dir) if self.raw_dir is not None else None,
            "status": status,
        }
        temporary_path: Path | None = None
        try:
            with tempfile.NamedTemporaryFile("w", dir=self.artifacts, prefix=".state-", encoding="utf-8", delete=False) as temporary:
                temporary_path = Path(temporary.name)
                temporary.write(json.dumps(payload, indent=2) + "\n")
                temporary.flush()
                os.fsync(temporary.fileno())
            os.replace(temporary_path, self.state_path)
        finally:
            if temporary_path is not None and temporary_path.exists():
                temporary_path.unlink()

    def start_process(
        self,
        name: str,
        args: Sequence[str],
        cwd: Path,
        env: Mapping[str, str],
        log_name: str,
        reservation: PortReservation | None = None,
    ) -> subprocess.Popen[bytes]:
        assert self.raw_dir is not None
        log = (self.raw_dir / log_name).open("wb")
        read_fd, release_fd = os.pipe()
        gate = (
            "import os,sys;"
            "fd=int(sys.argv[1]);"
            "ready=os.read(fd,1);os.close(fd);"
            "sys.exit(125) if ready!=b'1' else os.execvpe(sys.argv[2],sys.argv[2:],os.environ)"
        )
        process: subprocess.Popen[bytes] | None = None
        published = False
        try:
            process = subprocess.Popen(
                [sys.executable, "-c", gate, str(read_fd), *args],
                cwd=cwd,
                env=dict(env),
                stdout=log,
                stderr=subprocess.STDOUT,
                start_new_session=True,
                pass_fds=(read_fd,),
            )
            token = validate_process_start_token(process_start_token(process.pid))
            self.processes.append(OwnedProcess(name, process, token))
            published = True
            self.write_state("active")
            if reservation is not None:
                reservation.release()
            os.write(release_fd, b"1")
            return process
        except BaseException as start_error:
            if process is not None:
                try:
                    rollback_spawned_process(process)
                except BaseException as rollback_error:
                    state_error: BaseException | None = None
                    if published:
                        try:
                            self.write_state("rollback_failed")
                        except BaseException as error:
                            state_error = error
                    errors = [start_error, rollback_error]
                    if state_error is not None:
                        errors.append(state_error)
                    raise BaseExceptionGroup("managed process start and rollback failed", errors)
            if published:
                self.processes = [owned for owned in self.processes if owned.process is not process]
                with contextlib.suppress(Exception):
                    self.write_state("active")
            raise
        finally:
            log.close()
            os.close(read_fd)
            os.close(release_fd)

    def run(self) -> int:
        self.artifacts.mkdir(parents=True, exist_ok=False)
        with tempfile.TemporaryDirectory(prefix=f"mealswapp-e2e-{self.run_id}-") as raw:
            self.raw_dir = Path(raw)
            self.write_state("starting")
            failure: tuple[type[BaseException], BaseException, object] | None = None
            test_passed = False
            try:
                try:
                    self.execute()
                    test_passed = True
                except BaseException:
                    failure = sys.exc_info()  # type: ignore[assignment]
            finally:
                if self.signal_controller is not None:
                    self.signal_controller.begin_teardown()
            diagnostics_failed = False
            try:
                self.export_diagnostics("tearing_down")
            except Exception:
                diagnostics_failed = True
            cleanup_failures = self.cleanup()
            signal_received = self.signal_controller.received if self.signal_controller is not None else None
            result = "passed" if test_passed and failure is None and not cleanup_failures and signal_received is None else "failed"
            try:
                self.export_diagnostics(result)
            except Exception:
                diagnostics_failed = True
            late_signal = self.signal_controller.received if self.signal_controller is not None else None
            if late_signal is not None and signal_received is None:
                signal_received = late_signal
                result = "failed"
                try:
                    self.export_diagnostics(result)
                except Exception:
                    diagnostics_failed = True
            final_status = "cleaned"
            if cleanup_failures:
                final_status = "cleanup_failed"
            elif diagnostics_failed:
                final_status = "cleaned_diagnostics_failed"
            with contextlib.suppress(Exception):
                self.write_state(final_status)
            if self.signal_controller is not None:
                self.signal_controller.end_teardown()
                signal_received = self.signal_controller.received
            if failure is not None:
                raise failure[1].with_traceback(failure[2])  # type: ignore[arg-type]
            if signal_received is not None:
                raise InterruptedError(f"received signal {signal.Signals(signal_received).name}")
            if cleanup_failures:
                raise RuntimeError("owned environment cleanup failed")
            if diagnostics_failed:
                raise RuntimeError("diagnostics export failed")
            return 0

    def retire_process(self, process: subprocess.Popen[bytes]) -> None:
        owned = next((candidate for candidate in self.processes if candidate.process is process), None)
        if owned is None:
            raise SafetyError("managed process is not tracked")
        stop_owned_process(owned)
        if process_group_exists(process.pid):
            raise RuntimeError("managed process group remains after cleanup")
        self.processes.remove(owned)
        self.write_state("active")

    def start_application_stack(
        self,
        api_binary: Path,
        database_url: str,
        redis_url: str,
        reservations: list[PortReservation],
    ) -> tuple[dict[str, str], int, int, subprocess.Popen[bytes], subprocess.Popen[bytes]]:
        """Start both listeners, retrying the complete coupled stack after a bind race."""

        for attempt in range(3):
            attempt_reservations = reservations
            api_reservation, frontend_reservation = attempt_reservations
            api_port, frontend_port = api_reservation.port, frontend_reservation.port
            env = self.application_environment(database_url, redis_url, api_port, frontend_port)
            started: list[subprocess.Popen[bytes]] = []
            retry = False
            try:
                api = self.start_process("api", [str(api_binary)], REPO / "backend", env, "api.raw.log", api_reservation)
                started.append(api)
                readiness_port = reserve_port() if self.inject_api_timeout else api_port
                wait_http(f"http://127.0.0.1:{readiness_port}/health", api, min(self.timeout, 3) if self.inject_api_timeout else self.timeout)
                frontend = self.start_process(
                    "frontend",
                    ["bun", "run", "dev", "--", "--host", "127.0.0.1", "--port", str(frontend_port), "--strictPort"],
                    REPO / "frontend",
                    env,
                    "frontend.raw.log",
                    frontend_reservation,
                )
                started.append(frontend)
                wait_http(f"http://127.0.0.1:{frontend_port}", frontend, self.timeout)
                return env, api_port, frontend_port, api, frontend
            except ListenerExitedError:
                for process in reversed(started):
                    self.retire_process(process)
                self.events.append("port_bind_retry")
                if attempt == 2:
                    raise
                retry = True
            finally:
                for reservation in attempt_reservations:
                    reservation.release()
            if retry:
                reservations = reserve_ports(2)
        raise RuntimeError("application listener retry exhausted")

    def execute(self) -> None:
        reservations: list[PortReservation] = []
        try:
            create_database(self.target, self.database, self.comment)
            self.events.append("database_created")
            redis_port = self.start_redis()
            reservations = reserve_ports(2)
            api_reservation, frontend_reservation = reservations
            api_port, frontend_port = api_reservation.port, frontend_reservation.port
            database_url = self.target.database_url(self.database)
            redis_url = f"redis://127.0.0.1:{redis_port}/0"
            env = self.application_environment(database_url, redis_url, api_port, frontend_port)
            run_command(["go", "run", "./cmd/migrate", "up"], cwd=REPO / "backend", env=env, timeout=self.timeout)
            self.events.append("migrations_applied")
            assert self.raw_dir is not None
            api_binary = self.raw_dir / "mealswapp-api"
            bootstrap_binary = self.raw_dir / "admin-bootstrap"
            run_command(["go", "build", "-o", str(api_binary), "./cmd/api"], cwd=REPO / "backend", env=env, timeout=self.timeout)
            run_command(["go", "build", "-o", str(bootstrap_binary), "./cmd/admin-bootstrap"], cwd=REPO / "backend", env=env, timeout=self.timeout)
            env, api_port, frontend_port, api, frontend = self.start_application_stack(
                api_binary, database_url, redis_url, reservations
            )
            self.events.append("api_ready")
            fixture, request_ids = register_fixture(f"http://127.0.0.1:{api_port}", self.run_id)
            self.request_ids.extend(request_ids)
            bootstrap = run_command(
                [str(bootstrap_binary), "--environment", "development"],
                cwd=REPO / "backend",
                env=env,
                input_text=fixture["email"] + "\n",
                timeout=self.timeout,
            )
            if f"user_id={fixture['user_id']}" not in bootstrap.stdout:
                raise RuntimeError("administrator bootstrap returned an unexpected target")
            self.events.append("administrator_bootstrapped")
            self.events.append("frontend_ready")
            playwright_env = {
                **env,
                "MEALSWAPP_TASK261_REAL_E2E": "1",
                "MEALSWAPP_REAL_STACK_MANAGED": "1",
                "MEALSWAPP_REAL_STACK_BASE_URL": f"http://127.0.0.1:{frontend_port}",
                "MEALSWAPP_E2E_USER_ID": fixture["user_id"],
                "MEALSWAPP_E2E_EMAIL": fixture["email"],
                "MEALSWAPP_E2E_PASSWORD": fixture["password"],
                "MEALSWAPP_E2E_CLASSIFICATION": fixture["classification"],
                "MEALSWAPP_E2E_PRIVATE_ITEM": fixture["private_item"],
                "MEALSWAPP_E2E_GLOBAL_ITEM": fixture["global_item"],
                "PLAYWRIGHT_OUTPUT_DIR": str(self.raw_dir / "playwright"),
            }
            if self.inject_assertion_failure:
                playwright_env["MEALSWAPP_E2E_INJECT_ASSERTION_FAILURE"] = "1"
            run_command(
                ["bunx", "playwright", "test", "-c", "playwright.real-stack.config.ts", "tests/task261-real-admin-flow.spec.ts"],
                cwd=REPO / "frontend",
                env=playwright_env,
                timeout=self.timeout,
            )
            self.events.append("playwright_passed")
        finally:
            for reservation in reservations:
                reservation.release()

    def application_environment(self, database_url: str, redis_url: str, api_port: int, frontend_port: int) -> dict[str, str]:
        return {
            **os.environ,
            "MEALSWAPP_ENV": "development",
            "MEALSWAPP_DATABASE_URL": database_url,
            "MEALSWAPP_REDIS_URL": redis_url,
            "MEALSWAPP_HTTP_PORT": str(api_port),
            "MEALSWAPP_FRONTEND_ORIGIN": f"http://127.0.0.1:{frontend_port}",
            "MEALSWAPP_ALLOWED_ORIGINS": f"http://127.0.0.1:{frontend_port}",
            "MEALSWAPP_VITE_API_TARGET": f"http://127.0.0.1:{api_port}",
            "BUN_TMPDIR": str(REPO / "frontend" / ".bun-tmp"),
            "BUN_INSTALL": str(REPO / "frontend" / ".bun-install"),
            "GOCACHE": str(REPO / "backend" / ".go-cache"),
            "GOMODCACHE": str(REPO / "backend" / ".go-mod-cache"),
        }

    def start_redis(self) -> int:
        run_command(
            [
                "docker", "run", "-d", "--rm", "--name", self.container,
                "--label", f"{REDIS_RUN_LABEL}={self.run_id}",
                "--label", f"{REDIS_CREATED_LABEL}={self.created_at}",
                "-p", "127.0.0.1::6379", "redis:7-alpine",
                "redis-server", "--save", "", "--appendonly", "no",
            ],
            timeout=60,
        )
        inspect = docker_inspect(self.container)
        if inspect is None or inspect_labels(inspect).get(REDIS_RUN_LABEL) != self.run_id:
            raise SafetyError("Redis ownership could not be verified")
        ports = inspect.get("NetworkSettings", {})
        ports = ports.get("Ports", {}) if isinstance(ports, Mapping) else {}
        binding = ports.get("6379/tcp", []) if isinstance(ports, Mapping) else []
        if not isinstance(binding, list) or not binding or binding[0].get("HostIp") != "127.0.0.1":
            raise SafetyError("Redis did not bind to loopback")
        port = int(binding[0]["HostPort"])
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            try:
                if run_command(["docker", "exec", self.container, "redis-cli", "ping"], timeout=5).stdout.strip() == "PONG":
                    self.events.append("redis_ready")
                    return port
            except subprocess.CalledProcessError:
                time.sleep(0.1)
        raise TimeoutError("Redis readiness timed out")

    def export_diagnostics(self, result: str) -> None:
        diagnostics = safe_diagnostics(self.run_id, self.request_ids, self.events, result)
        (self.artifacts / "diagnostics.json").write_text(json.dumps(diagnostics, indent=2) + "\n", encoding="utf-8")
        (self.artifacts / "trace.ndjson").write_text(
            "".join(json.dumps({"run_id": self.run_id, "event": event}) + "\n" for event in diagnostics["events"]),
            encoding="utf-8",
        )
        screenshots = self.artifacts / "screenshots"
        screenshots.mkdir(exist_ok=True)
        write_sanitized_png(screenshots / "sanitized.png")
        (screenshots / "sanitized.txt").write_text(f"run_id={self.run_id}\nall source pixels and private fields removed\n", encoding="utf-8")

    def cleanup(self) -> list[str]:
        failures: list[str] = []
        for owned in reversed(self.processes):
            try:
                stop_owned_process(owned)
            except Exception:
                failures.append(f"{owned.name}_cleanup_failed")
        try:
            remove_owned_redis(self.container, self.run_id)
        except Exception:
            failures.append("redis_cleanup_failed")
        try:
            drop_owned_database(self.target, self.database, self.run_id, self.comment)
        except Exception:
            failures.append("database_cleanup_failed")
        self.events.extend(failures)
        try:
            self.write_state("cleanup_failed" if failures else "cleaned")
        except Exception:
            failures.append("state_cleanup_failed")
        return failures


def cleanup_stale(
    target: PostgresTarget,
    age_seconds: int,
    now: int | None = None,
    *,
    environment: str | None = None,
) -> int:
    validate_environment(environment if environment is not None else os.environ.get("MEALSWAPP_ENV", "development"))
    age_seconds = validate_stale_age(age_seconds)
    now = int(time.time()) if now is None else now
    cleaned = 0
    if not ARTIFACT_ROOT.exists():
        return cleaned
    for state_path in sorted(ARTIFACT_ROOT.glob("*/state.json")):
        try:
            state = json.loads(state_path.read_text(encoding="utf-8"))
            run_id = validate_run_id(str(state["run_id"]))
            created_at = int(state["created_at"])
            if now - created_at < age_seconds:
                continue
            raw_directory = state.get("raw_directory")
            if state.get("status") == "cleaned":
                if isinstance(raw_directory, str) and remove_owned_raw_directory(raw_directory, run_id):
                    cleaned += 1
                continue
            name = validate_database_name(str(state["database"]), run_id)
            comment = validate_ownership_comment(str(state["ownership_comment"]), run_id, created_at)
            container = str(state["redis_container"])
            if container != f"mealswapp-e2e-{run_id}":
                raise SafetyError("stale state Redis marker is invalid")
            for process in state.get("processes", []):
                if isinstance(process, Mapping):
                    stop_process(int(process["pid"]), process.get("start_token") if isinstance(process.get("start_token"), str) else None)
            remove_owned_redis(container, run_id)
            drop_owned_database(target, name, run_id, comment)
            if isinstance(raw_directory, str):
                remove_owned_raw_directory(raw_directory, run_id)
            state["status"] = "cleaned"
            state_path.write_text(json.dumps(state, indent=2) + "\n", encoding="utf-8")
            cleaned += 1
        except (KeyError, TypeError, ValueError, json.JSONDecodeError):
            continue
    return cleaned


def remove_owned_raw_directory(value: str, run_id: str) -> bool:
    """Remove only the exact temporary directory pattern belonging to one run."""

    validate_run_id(run_id)
    path = Path(value)
    expected_parent = Path(tempfile.gettempdir()).resolve()
    if path.parent.resolve() != expected_parent or path.is_symlink():
        raise SafetyError("raw artifact directory parent is unsafe")
    if not re.fullmatch(rf"mealswapp-e2e-{re.escape(run_id)}-[a-z0-9_]+", path.name):
        raise SafetyError("raw artifact directory name is unsafe")
    if not path.exists():
        return False
    shutil.rmtree(path)
    return True


def write_sanitized_png(path: Path) -> None:
    """Write a content-free one-pixel PNG without retaining source screenshot pixels."""

    def chunk(kind: bytes, data: bytes) -> bytes:
        return struct.pack(">I", len(data)) + kind + data + struct.pack(">I", zlib.crc32(kind + data) & 0xFFFFFFFF)

    path.write_bytes(
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", struct.pack(">IIBBBBB", 1, 1, 8, 2, 0, 0, 0))
        + chunk(b"IDAT", zlib.compress(b"\x00\x00\x00\x00"))
        + chunk(b"IEND", b"")
    )


def install_signal_handlers(controller: SignalController | None = None) -> dict[int, signal.Handlers]:
    controller = controller or SignalController()
    previous: dict[int, signal.Handlers] = {}

    for signum in (signal.SIGINT, signal.SIGTERM, signal.SIGHUP):
        previous[signum] = signal.signal(signum, controller.handle)
    return previous


def restore_signal_handlers(previous: Mapping[int, signal.Handlers]) -> None:
    for signum, handler in previous.items():
        signal.signal(signum, handler)


def parse_args(argv: Sequence[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--cleanup-stale", action="store_true", help="remove marked runs older than the age guard")
    parser.add_argument("--stale-age-seconds", type=int, default=DEFAULT_STALE_AGE_SECONDS)
    parser.add_argument("--timeout-seconds", type=float, default=120)
    parser.add_argument("--inject-assertion-failure", action="store_true", help=argparse.SUPPRESS)
    parser.add_argument("--inject-api-timeout", action="store_true", help=argparse.SUPPRESS)
    return parser.parse_args(argv)


def main(argv: Sequence[str] | None = None) -> int:
    args = parse_args(sys.argv[1:] if argv is None else argv)
    try:
        environment = os.environ.get("MEALSWAPP_ENV", "development")
        validate_environment(environment)
        target = PostgresTarget.parse(
            os.environ.get(
                "MEALSWAPP_E2E_POSTGRES_ADMIN_URL",
                "postgres://mealswapp:mealswapp@127.0.0.1:5432/postgres",
            )
        )
        controller = SignalController()
        previous = install_signal_handlers(controller)
        try:
            if args.cleanup_stale:
                controller.begin_teardown()
                count = cleanup_stale(target, args.stale_age_seconds, environment=environment)
                if controller.received is not None:
                    raise InterruptedError(f"received signal {signal.Signals(controller.received).name}")
                print(f"stale_cleanup_complete resources={count}")
                return 0
            if args.stale_age_seconds != DEFAULT_STALE_AGE_SECONDS:
                raise SafetyError("--stale-age-seconds is valid only with --cleanup-stale")
            return Harness(
                target,
                timeout=args.timeout_seconds,
                inject_assertion_failure=args.inject_assertion_failure,
                inject_api_timeout=args.inject_api_timeout,
                signal_controller=controller,
            ).run()
        finally:
            restore_signal_handlers(previous)
    except (
        SafetyError,
        RuntimeError,
        subprocess.SubprocessError,
        TimeoutError,
        InterruptedError,
        urllib.error.URLError,
        BaseExceptionGroup,
    ) as error:
        print(f"real-stack E2E failed: {type(error).__name__}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
