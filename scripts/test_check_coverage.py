#!/usr/bin/env python3

# Implements DESIGN-014 MetricsCollector coverage-exception contract tests.

import threading
import unittest
from types import SimpleNamespace
from unittest import mock

import scripts.check as check


BACKEND_PATH = "internal/example/service.go"
FRONTEND_PATH = "src/lib/example.ts"


def reason_catalog(reasons: dict[str, str]) -> str:
	return "\n".join(f"- `{reason_id}` — {reason}" for reason_id, reason in reasons.items())


def document(backend_rows: str = "", frontend_rows: str = "", *, backend_reasons: str | None = None, frontend_reasons: str | None = None) -> str:
	return f"""## Phase 08

### Testing coverage deviations

<!-- phase08-backend-coverage-contract:start -->
Measured Phase 08 scope: `0/1` statements (`0.0%`).
{backend_rows}
{backend_reasons if backend_reasons is not None else reason_catalog(check.BACKEND_EXCEPTION_REASONS)}
<!-- phase08-backend-coverage-contract:end -->

<!-- frontend-coverage-contract:start -->
{frontend_rows}
{frontend_reasons if frontend_reasons is not None else reason_catalog(check.FRONTEND_EXCEPTION_REASONS)}
<!-- frontend-coverage-contract:end -->
"""


def profile(count: int = 0) -> str:
	return f"mode: set\nexample/backend/{BACKEND_PATH}:1.1,2.1 1 {count}\n"


def backend_row() -> str:
	return f"| `{BACKEND_PATH}` | `0/1` | `0.0%` | `1.1-2.1` | `B1` |"


def frontend_output(functions: str = "50.00", lines: str = "75.00") -> str:
	return f"{FRONTEND_PATH} | {functions} | {lines} | 2\n"


def frontend_row(phase: str = "Phase 08", functions: str = "50.00", lines: str = "75.00", reason: str = "F4") -> str:
	return f"| `{FRONTEND_PATH}` | {phase} | {functions}% | {lines}% | `2` | `{reason}` |"


class Phase08BackendCoverageContractTests(unittest.TestCase):
	def validate(self, doc: str, measured_profile: str = profile()) -> None:
		with mock.patch.object(check, "PHASE08_GO_SOURCES", {BACKEND_PATH}):
			check.validate_phase08_go_coverage(measured_profile, doc)

	def test_accepts_exact_measured_exception(self) -> None:
		self.validate(document(backend_rows=backend_row()))

	def test_rejects_missing_exception(self) -> None:
		with self.assertRaisesRegex(SystemExit, "missing=.*service.go"):
			self.validate(document())

	def test_rejects_malformed_exception(self) -> None:
		row = backend_row().replace("`0/1`", "`zero/one`")
		with self.assertRaisesRegex(SystemExit, "Malformed Phase 08 backend"):
			self.validate(document(backend_rows=row))

	def test_rejects_over_broad_exception(self) -> None:
		with self.assertRaisesRegex(SystemExit, "over-broad=.*service.go"):
			self.validate(document(backend_rows=backend_row()).replace("`0/1` statements (`0.0%`)", "`1/1` statements (`100.0%`)"), profile(1))

	def test_rejects_unjustified_exception(self) -> None:
		with self.assertRaisesRegex(SystemExit, "unjustified reason B1"):
			self.validate(document(backend_rows=backend_row(), backend_reasons=""))


class FrontendCoverageContractTests(unittest.TestCase):
	def validate(self, doc: str, measured_output: str = frontend_output()) -> None:
		check.validate_frontend_exception_contract(measured_output, doc)

	def test_accepts_exact_semantic_exception(self) -> None:
		self.validate(document(frontend_rows=frontend_row()))

	def test_rejects_missing_exception(self) -> None:
		with self.assertRaisesRegex(SystemExit, "missing=.*example.ts"):
			self.validate(document())

	def test_rejects_malformed_exception(self) -> None:
		with self.assertRaisesRegex(SystemExit, "Malformed frontend"):
			self.validate(document(frontend_rows=frontend_row(functions="fifty")))

	def test_rejects_over_broad_exception(self) -> None:
		with self.assertRaisesRegex(SystemExit, "over-broad=.*example.ts"):
			self.validate(document(frontend_rows=frontend_row()), frontend_output("100.00", "100.00"))

	def test_rejects_unjustified_exception(self) -> None:
		with self.assertRaisesRegex(SystemExit, "unjustified reason F1"):
			self.validate(document(frontend_rows=frontend_row(), frontend_reasons=""))

	def test_rejects_stale_metrics_and_wrong_phase_owner(self) -> None:
		with self.assertRaisesRegex(SystemExit, "stale"):
			self.validate(document(frontend_rows=frontend_row(lines="74.00")))
		with mock.patch.object(check, "PHASE08_FRONTEND_SOURCES", {FRONTEND_PATH}):
			with self.assertRaisesRegex(SystemExit, "not phase-bound"):
				check.validate_phase08_frontend_coverage(frontend_output(), document(frontend_rows=frontend_row(phase="Phase 07")))


class CheckOrchestrationTests(unittest.TestCase):
	def test_independent_steps_overlap(self) -> None:
		barrier = threading.Barrier(2)

		def rendezvous(value: str) -> str:
			barrier.wait(timeout=1)
			return value

		results = check.execute_steps([
			check.CheckStep("one", lambda: rendezvous("first")),
			check.CheckStep("two", lambda: rendezvous("second")),
		])

		self.assertEqual(results, {"one": "first", "two": "second"})

	def test_quick_backend_packages_map_go_and_embedded_sql(self) -> None:
		self.assertEqual(
			check.quick_backend_packages({
				"backend/internal/auth/password.go",
				"backend/internal/repository/sql/compliance.sql",
				"docs/implementation/04_OPEN.md",
			}),
			["./internal/auth", "./internal/repository"],
		)

	def test_frontend_lane_uses_coverage_as_the_unit_test_pass(self) -> None:
		ready = threading.Event()
		failures: list[BaseException] = []
		with (
			mock.patch.object(check, "run") as run,
			mock.patch.object(check, "validate_frontend_coverage", return_value="coverage") as coverage,
		):
			result = check.run_frontend_lane(ready, failures)

		self.assertEqual(result, "coverage")
		self.assertTrue(ready.is_set())
		self.assertEqual(failures, [])
		self.assertEqual(
			[call.args[0] for call in run.call_args_list],
			[["bun", "run", "typecheck"], ["bun", "run", "build"]],
		)
		coverage.assert_called_once_with()

	def test_browser_lane_runs_only_the_complete_playwright_suite(self) -> None:
		ready = threading.Event()
		ready.set()
		with (
			mock.patch.object(check, "run") as run,
			mock.patch.object(check, "validate_frontend_e2e") as e2e,
		):
			check.run_browser_lane("check", ready, [])

		run.assert_called_once_with([
			"python3", "scripts/verify-frontend.py", "--screenshot-stem", "check",
		])
		e2e.assert_called_once_with(reuse_build=True)

	def test_backend_lane_omits_redundant_plain_full_test_pass(self) -> None:
		with (
			mock.patch.object(check, "validate_stripe_webhook_tests"),
			mock.patch.object(check, "validate_phase0601_backend_auth_billing_smoke_tests"),
			mock.patch.object(check, "running_compose_services", side_effect=[set(), set()]),
			mock.patch.object(check, "run"),
			mock.patch.object(check, "run_env") as run_env,
			mock.patch.object(check, "validate_phase07_backend_workflows"),
			mock.patch.object(check, "validate_go_coverage", return_value="coverage"),
		):
			result = check.run_backend_lane()

		self.assertEqual(result, "coverage")
		go_test_commands = [
			call.args[0]
			for call in run_env.call_args_list
			if call.args and call.args[0][:2] == ["go", "test"]
		]
		self.assertEqual(
			go_test_commands,
			[["go", "test", "-race", "./...", "-p", "1", "-count=1"]],
		)

	def test_phase07_exact_functions_come_from_isolated_package_profiles(self) -> None:
		documented = (
			"`internal/queue/job_queue.go:276 Reserve` | `60.0%`\n"
			"queue 60.0%\n"
		)

		def fake_run_env(command: list[str], *_args: object, **_kwargs: object) -> SimpleNamespace:
			if command[:2] == ["go", "test"]:
				return SimpleNamespace(stdout="coverage: 60.0% of statements\n", stderr="")
			return SimpleNamespace(
				stdout=(
					"example/backend/internal/queue/job_queue.go:276:\tReserve\t60.0%\n"
					"total:\t(statements)\t60.0%\n"
				),
				stderr="",
			)

		with (
			mock.patch.object(check, "PHASE07_GO_PACKAGES", {"example/queue"}),
			mock.patch.object(check, "OPEN_POINTS", mock.Mock(read_text=mock.Mock(return_value=documented))),
			mock.patch.object(check, "run_env", side_effect=fake_run_env),
		):
			check.validate_phase07_go_coverage(
				"example/backend/internal/queue/job_queue.go:276:\tReserve\t68.0%\n",
			)


if __name__ == "__main__":
	unittest.main()
