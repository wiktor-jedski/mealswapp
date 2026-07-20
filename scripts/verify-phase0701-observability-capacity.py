#!/usr/bin/env python3

"""Run the deterministic Phase 07.01 Task 234 regression gate."""

# Implements DESIGN-014 MetricsCollector and LogAggregator Task 234 verification.
# Verifies IT-ARCH-004-007, ARCH-004, and SW-REQ-080/SW-REQ-082.

from __future__ import annotations

import os
import subprocess
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
BACKEND = ROOT / "backend"


def run(command: list[str], cwd: Path, *, reject_skip: bool = False) -> None:
    environment = os.environ.copy()
    environment.setdefault("GOCACHE", str(BACKEND / ".go-cache"))
    environment.setdefault("GOMODCACHE", str(BACKEND / ".go-mod-cache"))
    environment.setdefault("MEALSWAPP_REDIS_URL", "redis://localhost:6379/13")
    print("+", " ".join(command))
    completed = subprocess.run(
        command,
        cwd=cwd,
        env=environment,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        check=False,
    )
    print(completed.stdout, end="")
    if completed.returncode:
        raise SystemExit(completed.returncode)
    if reject_skip and "--- SKIP:" in completed.stdout:
        raise SystemExit("Task 234 requires live Redis/restart fixtures; skipped tests fail the gate")
    if reject_skip and ("[no tests to run]" in completed.stdout or "warning: no tests to run" in completed.stdout):
        raise SystemExit("Task 234 test selection was empty; no-test package results fail the gate")


def main() -> int:
    run(["python3", "-m", "unittest", "scripts/test_verify_optimization_capacity.py"], ROOT)
    focused = "^(TestTask234.*|TestTask225LockCleanupIsBoundedAndObservable|TestTask225LiveManagerRecoversAfterRedisRestart|TestLPSolverWrapperCleanupFailureIsBoundedAndPreservesPrimaryResult|TestLPSolverWrapperTerminatesRealChildAndCleansDeadlineDirectory)$"
    packages = ["./internal/observability", "./internal/httpapi", "./internal/queue", "./internal/worker", "./internal/optimization"]
    run(["go", "test", "-v", *packages, "-run", focused, "-count=1"], BACKEND, reject_skip=True)
    run(["go", "test", "-v", "-race", *packages, "-run", focused, "-count=1"], BACKEND, reject_skip=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
