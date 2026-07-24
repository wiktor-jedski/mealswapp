#!/usr/bin/env python3

# Implements DESIGN-014 MetricsCollector coverage-exception contract tests.

import unittest
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


if __name__ == "__main__":
	unittest.main()
