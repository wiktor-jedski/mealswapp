#!/usr/bin/env python3
from __future__ import annotations

import os
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VALID_TASK_STATUSES = {"OPEN", "PREPARED", "REJECTED", "PASSED"}


def main() -> int:
    checks = [
        ("docs", check_docs),
        ("compose", check_compose),
        ("backend", check_backend),
        ("frontend", check_frontend),
        ("migrations", check_migrations),
    ]

    for name, check in checks:
        print(f"==> {name}")
        check()

    print("All checks passed.")
    return 0


def check_docs() -> None:
    task_list = ROOT / "docs/implementation/02_TASK_LIST.md"
    rows = [
        line
        for line in task_list.read_text(encoding="utf-8").splitlines()
        if re.match(r"^\|\s*\d+\s*\|", line)
    ]
    if not rows:
        raise SystemExit("No task rows found in docs/implementation/02_TASK_LIST.md")

    seen_ids: set[int] = set()
    for row in rows:
        cells = [cell.strip() for cell in row.strip("|").split("|")]
        task_id = int(cells[0])
        status = cells[3]
        if task_id in seen_ids:
            raise SystemExit(f"Duplicate task ID: {task_id}")
        if status not in VALID_TASK_STATUSES:
            raise SystemExit(f"Invalid status for task {task_id}: {status}")
        seen_ids.add(task_id)

    arch_files = {path.stem for path in (ROOT / "docs/architecture").glob("ARCH-*.md")}
    for design_file in (ROOT / "docs/design").glob("DESIGN-*.md"):
        text = design_file.read_text(encoding="utf-8")
        match = re.search(r"\*\*Traceability:\*\*\s*(ARCH-\d+)", text)
        if not match:
            raise SystemExit(f"Missing traceability in {design_file}")
        if match.group(1) not in arch_files:
            raise SystemExit(f"{design_file} references missing {match.group(1)}")


def check_compose() -> None:
    if not command_exists("docker"):
        print("docker not installed; skipping compose config validation")
        return

    run(["docker", "compose", "config"], cwd=ROOT)


def check_backend() -> None:
    env = os.environ.copy()
    env["GOCACHE"] = str(ROOT / "backend/.go-cache")
    run(["go", "test", "./..."], cwd=ROOT / "backend", env=env)


def check_frontend() -> None:
    run(["bun", "test"], cwd=ROOT / "frontend")
    run(["bun", "run", "build"], cwd=ROOT / "frontend")


def check_migrations() -> None:
    migrations_dir = ROOT / "db/migrations"
    if not migrations_dir.is_dir():
        raise SystemExit("Missing db/migrations directory")

    migration_files = sorted(migrations_dir.glob("*.sql"))
    if not migration_files:
        print("no SQL migrations yet; placeholder validation passed")
        return

    pairs: dict[str, set[str]] = {}
    for migration in migration_files:
        match = re.match(r"^(\d{4,}_.+)\.(up|down)\.sql$", migration.name)
        if not match:
            raise SystemExit(f"Migration filename must be ordered and end in .up.sql or .down.sql: {migration.name}")
        pairs.setdefault(match.group(1), set()).add(match.group(2))

    for name, directions in pairs.items():
        if directions != {"up", "down"}:
            raise SystemExit(f"Migration {name} must have both up and down files")


def command_exists(command: str) -> bool:
    return any((Path(path) / command).exists() for path in os.environ.get("PATH", "").split(os.pathsep))


def run(command: list[str], cwd: Path, env: dict[str, str] | None = None) -> None:
    subprocess.run(command, cwd=cwd, env=env, check=True)


if __name__ == "__main__":
    raise SystemExit(main())
