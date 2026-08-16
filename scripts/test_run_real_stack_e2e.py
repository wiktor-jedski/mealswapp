"""Safety and boundary tests for the Task 279 real-stack harness."""

# Implements DESIGN-005 RepositoryInterfaces isolated real-stack test persistence.

from __future__ import annotations

import importlib.util
import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

SPEC = importlib.util.spec_from_file_location("run_real_stack_e2e", Path(__file__).with_name("run-real-stack-e2e.py"))
assert SPEC and SPEC.loader
harness = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = harness
SPEC.loader.exec_module(harness)


class SafetyValidationTests(unittest.TestCase):
    def test_stale_cleanup_rejects_production_before_any_boundary(self) -> None:
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "", "postgres")
        with mock.patch.dict(harness.os.environ, {"MEALSWAPP_ENV": "production"}), mock.patch.object(
            harness.Path, "exists"
        ) as filesystem, mock.patch.object(harness, "remove_owned_redis") as redis, mock.patch.object(
            harness, "drop_owned_database"
        ) as postgres:
            with self.assertRaises(harness.SafetyError):
                harness.cleanup_stale(target, 900, now=901)
            filesystem.assert_not_called()
            redis.assert_not_called()
            postgres.assert_not_called()
        with mock.patch.dict(harness.os.environ, {"MEALSWAPP_ENV": "development"}), mock.patch.object(
            harness.Path, "exists"
        ) as filesystem:
            with self.assertRaises(harness.SafetyError):
                harness.cleanup_stale(target, 900, now=901, environment="")
            filesystem.assert_not_called()

    def test_rejects_unsafe_targets_before_process_boundary(self) -> None:
        unsafe_urls = [
            "postgres://user:pass@db.example.test:5432/postgres",
            "postgres://user:pass@localhost:5432/postgres",
            "postgres://user:pass@127.0.0.1:5432/mealswapp",
            "https://user:pass@127.0.0.1:5432/postgres",
            "postgres://127.0.0.1/postgres",
        ]
        with mock.patch.object(harness, "run_command") as run:
            for value in unsafe_urls:
                with self.assertRaises(harness.SafetyError):
                    harness.PostgresTarget.parse(value)
            for value in ["mealswapp", "other_test", "mealswapp_e2e_bad_test", "mealswapp_e2e_" + "a" * 24]:
                with self.assertRaises(harness.SafetyError):
                    harness.validate_database_name(value)
            harness.validate_environment("development")
            with self.assertRaises(harness.SafetyError):
                harness.validate_environment("production")
            run.assert_not_called()

    def test_database_deletion_requires_name_and_matching_comment(self) -> None:
        run_id = "a" * 24
        name = harness.database_name(run_id)
        comment = harness.ownership_comment(run_id, 100)
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "secret", "postgres")
        with mock.patch.object(harness, "database_comment", return_value="wrong"), mock.patch.object(harness, "psql") as psql:
            with self.assertRaises(harness.SafetyError):
                harness.drop_owned_database(target, name, run_id, comment)
            psql.assert_not_called()
        with mock.patch.object(harness, "database_comment", return_value=comment), mock.patch.object(harness, "psql") as psql:
            self.assertTrue(harness.drop_owned_database(target, name, run_id, comment))
            self.assertEqual(psql.call_count, 2)

    def test_database_creation_rejects_mismatched_name_and_comment_ids_before_postgres(self) -> None:
        name = harness.database_name("a" * 24)
        comment = harness.ownership_comment("b" * 24, 100)
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "", "postgres")
        with mock.patch.object(harness, "psql") as psql:
            with self.assertRaises(harness.SafetyError):
                harness.create_database(target, name, comment)
            psql.assert_not_called()

    def test_redis_cleanup_requires_exact_run_label(self) -> None:
        run_id = "b" * 24
        container = f"mealswapp-e2e-{run_id}"
        inspect = {"Config": {"Labels": {harness.REDIS_RUN_LABEL: "c" * 24}}}
        with mock.patch.object(harness, "docker_inspect", return_value=inspect), mock.patch.object(harness, "run_command") as run:
            with self.assertRaises(harness.SafetyError):
                harness.remove_owned_redis(container, run_id)
            run.assert_not_called()

    def test_stale_cleanup_enforces_minimum_age_and_is_idempotent(self) -> None:
        with self.assertRaises(harness.SafetyError):
            harness.validate_stale_age(harness.MIN_STALE_AGE_SECONDS - 1)
        run_id = "d" * 24
        state = {
            "run_id": run_id,
            "created_at": 1,
            "database": harness.database_name(run_id),
            "ownership_comment": harness.ownership_comment(run_id, 1),
            "redis_container": f"mealswapp-e2e-{run_id}",
            "processes": [],
            "raw_directory": None,
            "status": "active",
        }
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "", "postgres")
        with tempfile.TemporaryDirectory() as directory, mock.patch.object(harness, "ARTIFACT_ROOT", Path(directory)):
            path = Path(directory) / run_id
            path.mkdir()
            (path / "state.json").write_text(json.dumps(state), encoding="utf-8")
            with mock.patch.object(harness, "remove_owned_redis", return_value=True), mock.patch.object(
                harness, "drop_owned_database", return_value=True
            ):
                self.assertEqual(harness.cleanup_stale(target, 900, now=901), 1)
                self.assertEqual(harness.cleanup_stale(target, 900, now=902), 0)


class DiagnosticsTests(unittest.TestCase):
    def test_diagnostics_failure_cannot_skip_cleanup_or_replace_original_failure(self) -> None:
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "", "postgres")
        with tempfile.TemporaryDirectory() as directory, mock.patch.object(harness, "ARTIFACT_ROOT", Path(directory)):
            instance = harness.Harness(target)
            original = RuntimeError("original")
            with mock.patch.object(harness, "create_database", side_effect=original), mock.patch.object(
                instance, "export_diagnostics", side_effect=OSError("diagnostics")
            ), mock.patch.object(instance, "cleanup") as cleanup:
                with self.assertRaisesRegex(RuntimeError, "original"):
                    instance.run()
                cleanup.assert_called_once()

    def test_diagnostics_allow_only_run_and_request_ids(self) -> None:
        run_id = "e" * 24
        request_id = "7d444840-9dc0-11d1-b245-5ffdce74fad2"
        payload = harness.safe_diagnostics(
            run_id,
            [request_id, "person@example.test", "cookie=secret", "postgres://secret"],
            ["api_ready", "name Person", '{"body":"secret"}'],
            "failed",
        )
        encoded = json.dumps(payload)
        self.assertIn(run_id, encoded)
        self.assertIn(request_id, encoded)
        for forbidden in ("@", "cookie", "csrf", "postgres:", "Person", "body", "secret"):
            self.assertNotIn(forbidden, encoded)

    def test_process_port_signal_and_filesystem_boundaries_are_injectable(self) -> None:
        with mock.patch.object(harness.socket, "socket") as socket_type:
            socket_type.return_value.getsockname.return_value = ("127.0.0.1", 49152)
            self.assertEqual(harness.reserve_port(), 49152)
        controller = harness.SignalController()
        with mock.patch.object(harness.signal, "signal", side_effect=lambda number, handler: handler) as install:
            previous = harness.install_signal_handlers(controller)
            self.assertEqual(len(previous), 3)
            self.assertEqual(install.call_count, 3)
        completed = subprocess.CompletedProcess(["true"], 0, "ok", "")
        process = mock.Mock(returncode=0)
        process.communicate.return_value = ("ok", "")
        with mock.patch.object(subprocess, "Popen", return_value=process) as popen:
            self.assertEqual(harness.run_command(["true"]).stdout, "ok")
            popen.assert_called_once()

    def test_missing_or_reused_process_token_never_signals_a_process_group(self) -> None:
        with mock.patch.object(harness, "process_start_token", return_value=None), mock.patch.object(
            harness.os, "killpg"
        ) as kill_group, mock.patch.object(harness.os, "kill") as kill:
            with self.assertRaises(harness.SafetyError):
                harness.stop_process(4321, None)
            kill.assert_not_called()
            kill_group.assert_not_called()
        with mock.patch.object(harness, "process_start_token", return_value="new"), mock.patch.object(
            harness.os, "killpg"
        ) as kill_group, mock.patch.object(harness.os, "kill") as kill:
            with self.assertRaises(harness.SafetyError):
                harness.stop_process(4321, "old")
            kill.assert_not_called()
            kill_group.assert_not_called()

    def test_spawn_publication_failure_rolls_back_the_exact_child(self) -> None:
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "", "postgres")
        process = mock.Mock(pid=4321)
        process.wait.return_value = 0
        with tempfile.TemporaryDirectory() as directory, mock.patch.object(harness, "ARTIFACT_ROOT", Path(directory)):
            instance = harness.Harness(target)
            instance.raw_dir = Path(directory)
            with mock.patch.object(harness.subprocess, "Popen", return_value=process), mock.patch.object(
                harness, "process_start_token", return_value="123"
            ), mock.patch.object(instance, "write_state", side_effect=OSError("state")), mock.patch.object(
                harness, "rollback_spawned_process"
            ) as rollback:
                with self.assertRaises(OSError):
                    instance.start_process("api", ["api"], Path(directory), {}, "api.log")
                rollback.assert_called_once_with(process)
                self.assertEqual(instance.processes, [])

    def test_spawn_rollback_failure_keeps_process_tracked_and_reports_both_failures(self) -> None:
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "", "postgres")
        process = mock.Mock(pid=4321)
        statuses: list[str] = []

        def write_state(status: str) -> None:
            statuses.append(status)
            if len(statuses) == 1:
                raise OSError("publication")

        with tempfile.TemporaryDirectory() as directory, mock.patch.object(harness, "ARTIFACT_ROOT", Path(directory)):
            instance = harness.Harness(target)
            instance.raw_dir = Path(directory)
            with mock.patch.object(harness.subprocess, "Popen", return_value=process), mock.patch.object(
                harness, "process_start_token", return_value="123"
            ), mock.patch.object(instance, "write_state", side_effect=write_state), mock.patch.object(
                harness, "rollback_spawned_process", side_effect=RuntimeError("rollback")
            ):
                with self.assertRaises(BaseExceptionGroup) as raised:
                    instance.start_process("api", ["api"], Path(directory), {}, "api.log")
            self.assertEqual([str(error) for error in raised.exception.exceptions], ["publication", "rollback"])
            self.assertEqual([owned.process for owned in instance.processes], [process])
            self.assertEqual(statuses, ["active", "rollback_failed"])

    def test_spawn_gate_prevents_command_execution_before_state_publication(self) -> None:
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "", "postgres")
        with tempfile.TemporaryDirectory() as directory, mock.patch.object(harness, "ARTIFACT_ROOT", Path(directory) / "artifacts"):
            marker = Path(directory) / "orphaned"
            instance = harness.Harness(target)
            instance.raw_dir = Path(directory)
            with mock.patch.object(instance, "write_state", side_effect=OSError("publication")):
                with self.assertRaises(OSError):
                    instance.start_process(
                        "api",
                        [sys.executable, "-c", "from pathlib import Path; import sys; Path(sys.argv[1]).write_text('ran')", str(marker)],
                        Path(directory),
                        os.environ,
                        "spawn.log",
                    )
            self.assertFalse(marker.exists())
            self.assertEqual(instance.processes, [])

    def test_port_reservation_is_held_until_process_ownership_is_published(self) -> None:
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "", "postgres")
        process = mock.Mock(pid=4321)
        order: list[str] = []
        reservation = mock.Mock()
        reservation.release.side_effect = lambda: order.append("port_released")
        with tempfile.TemporaryDirectory() as directory, mock.patch.object(harness, "ARTIFACT_ROOT", Path(directory)):
            instance = harness.Harness(target)
            instance.raw_dir = Path(directory)
            with mock.patch.object(harness.subprocess, "Popen", return_value=process), mock.patch.object(
                harness, "process_start_token", return_value="123"
            ), mock.patch.object(instance, "write_state", side_effect=lambda _status: order.append("state_published")):
                instance.start_process("api", ["api"], Path(directory), {}, "api.log", reservation)
        self.assertEqual(order, ["state_published", "port_released"])

    def test_cleanup_never_kills_group_when_managed_leader_token_is_unavailable(self) -> None:
        process = mock.Mock(pid=4321)
        process.poll.return_value = 0
        owned = harness.OwnedProcess("frontend", process, "123")
        with mock.patch.object(harness, "process_start_token", return_value=None), mock.patch.object(
            harness, "process_group_exists", return_value=True
        ), mock.patch.object(harness.os, "killpg") as kill_group:
            with self.assertRaises(harness.SafetyError):
                harness.stop_owned_process(owned)
        kill_group.assert_not_called()

    def test_teardown_queues_handled_signals_without_interrupting(self) -> None:
        controller = harness.SignalController()
        controller.begin_teardown()
        controller.handle(harness.signal.SIGTERM, None)
        self.assertEqual(controller.received, harness.signal.SIGTERM)

    def test_signal_received_during_final_diagnostics_cannot_return_success(self) -> None:
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "", "postgres")
        controller = harness.SignalController()
        with tempfile.TemporaryDirectory() as directory, mock.patch.object(harness, "ARTIFACT_ROOT", Path(directory)):
            instance = harness.Harness(target, signal_controller=controller)

            def export(result: str) -> None:
                if result == "passed":
                    controller.handle(harness.signal.SIGTERM, None)

            with mock.patch.object(instance, "execute"), mock.patch.object(instance, "cleanup", return_value=[]), mock.patch.object(
                instance, "export_diagnostics", side_effect=export
            ):
                with self.assertRaises(InterruptedError):
                    instance.run()

    def test_run_command_timeout_reaps_descendant_process_group(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            child_pid = Path(directory) / "child.pid"
            script = (
                "import pathlib,subprocess,sys,time;"
                "child=subprocess.Popen(['sleep','60']);"
                "pathlib.Path(sys.argv[1]).write_text(str(child.pid));"
                "time.sleep(60)"
            )
            with self.assertRaises(subprocess.TimeoutExpired):
                harness.run_command([sys.executable, "-c", script, str(child_pid)], timeout=0.2)
            pid = int(child_pid.read_text(encoding="utf-8"))
            for _ in range(50):
                if not Path(f"/proc/{pid}").exists():
                    break
                harness.time.sleep(0.02)
            self.assertFalse(Path(f"/proc/{pid}").exists())

    def test_failure_injection_is_explicit_and_does_not_change_default(self) -> None:
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "", "postgres")
        normal = harness.Harness(target)
        injected = harness.Harness(target, inject_assertion_failure=True, inject_api_timeout=True)
        self.assertFalse(normal.inject_assertion_failure)
        self.assertFalse(normal.inject_api_timeout)
        self.assertTrue(injected.inject_assertion_failure)
        self.assertTrue(injected.inject_api_timeout)

    def test_sanitized_artifacts_survive_owned_cleanup(self) -> None:
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "", "postgres")
        with tempfile.TemporaryDirectory() as directory, mock.patch.object(harness, "ARTIFACT_ROOT", Path(directory)):
            instance = harness.Harness(target)
            instance.artifacts.mkdir(parents=True)
            instance.export_diagnostics("failed")
            with mock.patch.object(harness, "remove_owned_redis", return_value=False), mock.patch.object(
                harness, "drop_owned_database", return_value=False
            ):
                instance.cleanup()
            self.assertTrue((instance.artifacts / "diagnostics.json").is_file())
            self.assertTrue((instance.artifacts / "trace.ndjson").is_file())
            self.assertTrue((instance.artifacts / "screenshots/sanitized.png").is_file())

    def test_raw_artifact_cleanup_requires_exact_run_owned_path(self) -> None:
        run_id = "1" * 24
        with tempfile.TemporaryDirectory() as directory:
            owned = Path(directory) / f"mealswapp-e2e-{run_id}-raw"
            owned.mkdir()
            with mock.patch.object(harness.tempfile, "gettempdir", return_value=directory):
                self.assertTrue(harness.remove_owned_raw_directory(str(owned), run_id))
                with self.assertRaises(harness.SafetyError):
                    harness.remove_owned_raw_directory(str(Path(directory) / "unrelated"), run_id)


class IntegrationContractTests(unittest.TestCase):
    def test_port_reservations_are_distinct_and_close_on_collision(self) -> None:
        first, second = mock.Mock(), mock.Mock()
        first.getsockname.return_value = ("127.0.0.1", 49152)
        second.getsockname.return_value = ("127.0.0.1", 49152)
        with mock.patch.object(harness.socket, "socket", side_effect=[first, second]):
            with self.assertRaises(harness.SafetyError):
                harness.reserve_ports(2)
        first.close.assert_called_once()
        second.close.assert_called_once()

    def test_distinct_port_reservations_remain_bound_until_released(self) -> None:
        reservations = harness.reserve_ports(2)
        try:
            self.assertNotEqual(reservations[0].port, reservations[1].port)
            for reservation in reservations:
                competitor = harness.socket.socket(harness.socket.AF_INET, harness.socket.SOCK_STREAM)
                try:
                    with self.assertRaises(OSError):
                        competitor.bind(("127.0.0.1", reservation.port))
                finally:
                    competitor.close()
        finally:
            for reservation in reservations:
                reservation.release()

    def test_listener_exit_after_reservation_release_retries_whole_stack_with_new_ports(self) -> None:
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "", "postgres")
        instance = harness.Harness(target)
        first = [mock.Mock(port=41001), mock.Mock(port=41002)]
        second = [mock.Mock(port=42001), mock.Mock(port=42002)]
        failed = mock.Mock(pid=41001)
        failed.poll.return_value = 1
        api = mock.Mock(pid=42001)
        frontend = mock.Mock(pid=42002)
        with mock.patch.object(harness, "reserve_ports", side_effect=[second]), mock.patch.object(
            instance, "start_process", side_effect=[failed, api, frontend]
        ) as start, mock.patch.object(
            harness, "wait_http", side_effect=[harness.ListenerExitedError("collision"), None, None]
        ), mock.patch.object(instance, "retire_process") as retire:
            env, api_port, frontend_port, _, _ = instance.start_application_stack(
                Path("/api"), "postgres://redacted", "redis://127.0.0.1:1234/0", first
            )
        retire.assert_called_once_with(failed)
        self.assertEqual(start.call_count, 3)
        self.assertEqual((api_port, frontend_port), (42001, 42002))
        self.assertEqual(env["MEALSWAPP_VITE_API_TARGET"], "http://127.0.0.1:42001")

    def test_competitor_claim_between_release_and_exec_is_detected(self) -> None:
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "", "postgres")
        reservation = harness.reserve_ports(1)[0]
        competitor = harness.socket.socket(harness.socket.AF_INET, harness.socket.SOCK_STREAM)
        release = reservation.release

        def claim_reserved_port() -> None:
            release()
            competitor.bind(("127.0.0.1", reservation.port))
            competitor.listen()

        command = [
            sys.executable,
            "-c",
            (
                "import os,socket,time;"
                "listener=socket.socket();"
                "listener.bind(('127.0.0.1',int(os.environ['MEALSWAPP_HTTP_PORT'])));"
                "listener.listen();time.sleep(5)"
            ),
        ]
        with tempfile.TemporaryDirectory() as directory, mock.patch.object(
            harness, "ARTIFACT_ROOT", Path(directory) / "artifacts"
        ):
            instance = harness.Harness(target)
            instance.raw_dir = Path(directory)
            env = {**os.environ, "MEALSWAPP_HTTP_PORT": str(reservation.port)}
            try:
                with mock.patch.object(reservation, "release", side_effect=claim_reserved_port):
                    process = instance.start_process(
                        "api", command, Path(directory), env, "api.log", reservation
                    )
                with self.assertRaises(harness.ListenerExitedError):
                    harness.wait_http(
                        f"http://127.0.0.1:{reservation.port}/health", process, 2
                    )
            finally:
                competitor.close()
            process.wait(timeout=2)
            instance.retire_process(process)
            self.assertEqual(instance.processes, [])

    def test_task261_has_no_direct_sql_or_child_process_promotion(self) -> None:
        source = (harness.REPO / "frontend/tests/task261-real-admin-flow.spec.ts").read_text(encoding="utf-8")
        self.assertNotIn("node:child_process", source)
        self.assertNotRegex(source, r"\b(UPDATE|INSERT|DELETE|TRUNCATE)\s+(users|food_items)")
        shell = (harness.REPO / "scripts/verify-task-261-ui.sh").read_text(encoding="utf-8")
        self.assertIn("run-real-stack-e2e.py", shell)
        self.assertNotIn("start-services.sh", shell)

    def test_vite_proxy_is_configurable(self) -> None:
        source = (harness.REPO / "frontend/vite.config.ts").read_text(encoding="utf-8")
        self.assertIn("MEALSWAPP_VITE_API_TARGET", source)
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "secret", "postgres")
        instance = harness.Harness(target)
        env = instance.application_environment("postgres://redacted", "redis://127.0.0.1:1234/0", 43210, 43211)
        self.assertEqual(env["MEALSWAPP_VITE_API_TARGET"], "http://127.0.0.1:43210")
        self.assertEqual(instance.state_path.parent.name, instance.run_id)

    def test_redis_readiness_uses_only_run_owned_container(self) -> None:
        run_id = "f" * 24
        target = harness.PostgresTarget("127.0.0.1", 5432, "user", "", "postgres")
        instance = harness.Harness(target)
        instance.run_id = run_id
        instance.container = f"mealswapp-e2e-{run_id}"
        inspect = {
            "Config": {"Labels": {harness.REDIS_RUN_LABEL: run_id}},
            "NetworkSettings": {"Ports": {"6379/tcp": [{"HostIp": "127.0.0.1", "HostPort": "49153"}]}},
        }
        completed = subprocess.CompletedProcess([], 0, "PONG\n", "")
        with mock.patch.object(harness, "docker_inspect", return_value=inspect), mock.patch.object(
            harness, "run_command", return_value=completed
        ) as run:
            self.assertEqual(instance.start_redis(), 49153)
            flattened = [" ".join(call.args[0]) for call in run.call_args_list]
            self.assertTrue(any(f"{harness.REDIS_RUN_LABEL}={run_id}" in command for command in flattened))
            self.assertTrue(any(f"docker exec {instance.container} redis-cli ping" in command for command in flattened))


if __name__ == "__main__":
    unittest.main()
